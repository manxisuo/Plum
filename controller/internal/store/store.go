package store

import "time"

type Node struct {
	NodeID   string
	IP       string
	Labels   map[string]string
	LastSeen time.Time
}

type DesiredState string

const (
	DesiredDeployed DesiredState = "Deployed"
	DesiredRunning  DesiredState = "Running"
	DesiredStopped  DesiredState = "Stopped"
)

type Assignment struct {
	InstanceID   string
	DeploymentID string
	NodeID       string
	Desired      DesiredState
	ArtifactURL  string
	StartCmd     string
	AppName      string // 应用名称
	AppVersion   string // 应用版本
}

type InstanceStatus struct {
	InstanceID string
	Phase      string
	ExitCode   int
	Healthy    bool
	TsUnix     int64
}

// Task (short job) minimal model for Phase A
type Task struct {
	TaskID       string
	Name         string
	Executor     string // service | embedded | os_process
	TargetKind   string // service | deployment | node (depending on executor)
	TargetRef    string // e.g. serviceName/version/protocol or deploymentId/nodeId
	State        string // Pending | Scheduled | Running | Succeeded | Failed | Timeout | Canceled
	PayloadJSON  string // raw JSON input
	ResultJSON   string // raw JSON output
	Error        string
	TimeoutSec   int
	MaxRetries   int
	Attempt      int
	ScheduledOn  string // nodeId or endpoint info (optional)
	CreatedAt    int64
	StartedAt    int64
	FinishedAt   int64
	Labels       map[string]string
	OriginTaskID string // for grouping reruns; empty means original
}

type Artifact struct {
	ArtifactID     string
	AppName        string
	Version        string
	Path           string // served URL path, e.g. /artifacts/xxx.zip (for zip) or empty (for image)
	SHA256         string
	SizeBytes      int64
	CreatedAt      int64
	Type           string // "zip" or "image"
	ImageRepository string // Docker image repository, e.g. "openeuler/openeuler"
	ImageTag       string // Docker image tag, e.g. "22.03"
	PortMappings   string // JSON string for port mappings, e.g. [{"host":8080,"container":80}]
}

type DeploymentStatus string

const (
	DeploymentStopped DeploymentStatus = "Stopped"
	DeploymentRunning DeploymentStatus = "Running"
)

type Deployment struct {
	DeploymentID string
	Name         string
	Labels       map[string]string
	Status       DeploymentStatus // Stopped | Running
}

// Service endpoint exposed by an instance
type Endpoint struct {
	ServiceName string
	InstanceID  string
	NodeID      string
	IP          string
	Port        int
	Protocol    string
	Version     string
	Labels      map[string]string
	Healthy     bool
	LastSeen    int64
}

// Embedded Worker capability (executor=embedded) - Legacy HTTP-based
type Worker struct {
	WorkerID string
	NodeID   string
	URL      string   // callback endpoint (optional for MVP)
	Tasks    []string // supported task names
	Labels   map[string]string
	Capacity int
	LastSeen int64
}

// Embedded Worker capability (executor=embedded) - New gRPC-based
type EmbeddedWorker struct {
	WorkerID    string
	NodeID      string
	InstanceID  string   // app instance ID
	AppName     string   // app name from environment
	AppVersion  string   // app version from environment
	GRPCAddress string   // gRPC server address (host:port)
	Tasks       []string // supported task names
	Labels      map[string]string
	LastSeen    int64
}

// ========== DAG Workflow (新架构) ==========

type NodeType string

const (
	NodeTypeTask     NodeType = "task"     // 任务节点
	NodeTypeParallel NodeType = "parallel" // 并行节点
	NodeTypeBranch   NodeType = "branch"   // 分支节点
	NodeTypeLoop     NodeType = "loop"     // 循环节点
)

type TriggerRule string

const (
	TriggerAllSuccess TriggerRule = "all_success" // 所有前驱成功（默认）
	TriggerOneSuccess TriggerRule = "one_success" // 任一前驱成功
	TriggerAllFailed  TriggerRule = "all_failed"  // 所有前驱失败
	TriggerOneFailed  TriggerRule = "one_failed"  // 任一前驱失败
	TriggerAllDone    TriggerRule = "all_done"    // 所有前驱完成（成功或失败）
	TriggerNoneFailed TriggerRule = "none_failed" // 没有前驱失败（所有前驱成功或跳过）
)

// 分支条件
type BranchCondition struct {
	SourceTask string `json:"sourceTask"` // 依赖的任务节点ID
	Field      string `json:"field"`      // 结果字段路径，如 "code"
	Operator   string `json:"operator"`   // ==, !=, >, <, >=, <=
	Value      string `json:"value"`      // 比较值
}

// 循环条件
type LoopCondition struct {
	Type        string `json:"type"`        // count | condition
	Count       int    `json:"count"`       // 循环次数（type=count时使用）
	SourceTask  string `json:"sourceTask"`  // 依赖的任务节点ID（type=condition时使用）
	Field       string `json:"field"`       // 结果字段路径，如 "items.length"
	Operator    string `json:"operator"`    // ==, !=, >, <, >=, <=
	Value       string `json:"value"`       // 比较值
	LoopVarName string `json:"loopVarName"` // 循环变量名，如 "i" 或 "item"
}

// DAG节点
type WorkflowNode struct {
	NodeID      string
	Type        NodeType
	Name        string
	TriggerRule TriggerRule

	// Task节点配置
	TaskDefID   string
	PayloadJSON string
	TimeoutSec  int
	MaxRetries  int

	// Branch节点配置
	Condition *BranchCondition

	// Parallel节点配置
	WaitPolicy string // all | one

	// Loop节点配置
	LoopCondition *LoopCondition

	// UI位置
	PosX int
	PosY int
}

// DAG边
type WorkflowEdge struct {
	From     string
	To       string
	EdgeType string // normal | true | false (for branch)
}

// DAG工作流
type WorkflowDAG struct {
	WorkflowID string
	Name       string
	Version    int // DAG版本（v2）
	Nodes      map[string]WorkflowNode
	Edges      []WorkflowEdge
	StartNodes []string
	CreatedAt  int64
}

// ========== Legacy Sequential Workflow (向后兼容) ==========

type WorkflowStep struct {
	StepID       string
	Name         string            // taskName
	Executor     string            // service|embedded|os_process
	TargetKind   string            // service|deployment|node (for service executor)
	TargetRef    string            // serviceName for service executor
	Labels       map[string]string // service executor labels (servicePath, servicePort, etc.)
	PayloadJSON  string            // optional: override default payload for this step
	TimeoutSec   int
	MaxRetries   int
	Ord          int    // sequence order
	DefinitionID string // optional: reference TaskDefinition
}

type Workflow struct {
	WorkflowID string
	Name       string
	Labels     map[string]string
	Steps      []WorkflowStep
}

type WorkflowRun struct {
	RunID      string
	WorkflowID string
	State      string // Pending|Running|Succeeded|Failed|Canceled
	CreatedAt  int64
	StartedAt  int64
	FinishedAt int64
}

type StepRun struct {
	RunID      string
	StepID     string
	TaskID     string
	State      string // Pending|Running|Succeeded|Failed
	StartedAt  int64
	FinishedAt int64
	Ord        int
}

// Resource management data models
type ResourceStateDesc struct {
	Type  string // INT|DOUBLE|BOOL|ENUM|STRING
	Name  string
	Value string
	Unit  string
}

type ResourceOpDesc struct {
	Type  string // INT|DOUBLE|BOOL|ENUM|STRING
	Name  string
	Value string
	Unit  string
	Min   string
	Max   string
}

type Resource struct {
	ResourceID string
	NodeID     string
	Type       string // Radar/Sonar/XXGun等
	URL        string // 操作回调URL
	StateDesc  []ResourceStateDesc
	OpDesc     []ResourceOpDesc
	LastSeen   int64
	CreatedAt  int64
}

type ResourceState struct {
	ResourceID string
	Timestamp  int64
	States     map[string]string // name -> value
}

type ResourceOp struct {
	ResourceID string
	Operations []ResourceOperation
	Timestamp  int64
}

type ResourceOperation struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Store interface {
	UpsertNode(id string, n Node) error
	GetNode(id string) (Node, bool, error)
	ListNodes() ([]Node, error)
	DeleteNode(id string) error
	ListAssignmentsForNode(nodeID string) ([]Assignment, error)
	GetAssignment(instanceID string) (Assignment, bool, error)
	AddAssignment(nodeID string, a Assignment) error
	DeleteAssignment(instanceID string) error
	DeleteStatusesForInstance(instanceID string) error
	DeleteAssignmentsForDeployment(deploymentID string) error
	UpdateAssignmentDesired(instanceID string, desired DesiredState) error
	AppendStatus(instanceID string, st InstanceStatus) error
	LatestStatus(instanceID string) (InstanceStatus, bool, error)

	CountAssignmentsByArtifactPath(path string) (int, error)
	CountAssignmentsForNode(nodeID string) (int, error)
	CreateDeployment(name string, labels map[string]string) (string, []string, error)
	NewInstanceID(deploymentID string) string

	SaveArtifact(a Artifact) (string, error)
	ListArtifacts() ([]Artifact, error)
	GetArtifact(id string) (Artifact, bool, error)
	GetArtifactByPath(path string) (Artifact, bool, error)
	DeleteArtifact(id string) error

	ListDeployments() ([]Deployment, error)
	GetDeployment(id string) (Deployment, bool, error)
	DeleteDeployment(id string) error
	UpdateDeploymentStatus(id string, status DeploymentStatus) error
	ListAssignmentsForDeployment(deploymentID string) ([]Assignment, error)

	// Services / discovery
	ReplaceEndpointsForInstance(nodeID string, instanceID string, eps []Endpoint) error
	UpdateEndpointHealthForInstance(instanceID string, eps []Endpoint) error
	TouchEndpointsForInstance(instanceID string, ts int64) error
	DeleteEndpointsForInstance(instanceID string) error
	// 替换指定服务的端点（删除该实例下指定服务的所有端点，然后插入新端点）
	ReplaceEndpointsForInstanceAndService(nodeID string, instanceID string, serviceName string, eps []Endpoint) error
	// 添加单个端点（如果已存在则更新，不删除其他端点）
	AddEndpoint(ep Endpoint) error
	// 删除单个端点（通过主键）
	DeleteEndpoint(serviceName string, instanceID string, ip string, port int, protocol string) error
	// 更新单个端点信息
	UpdateEndpoint(serviceName string, instanceID string, oldIP string, oldPort int, oldProtocol string, ep Endpoint) error
	ListEndpointsByService(serviceName string, version string, protocol string) ([]Endpoint, error)
	// 列出服务的所有端点（包括不健康的，用于管理界面）
	ListAllEndpointsByService(serviceName string) ([]Endpoint, error)
	ListServices() ([]string, error)

	// Tasks (Phase A minimal)
	CreateTask(t Task) (string, error)
	GetTask(id string) (Task, bool, error)
	ListTasks() ([]Task, error)
	DeleteTask(id string) error
	UpdateTaskState(id string, state string) error
	UpdateTaskRunning(id string, startedAt int64, scheduledOn string, attempt int) error
	UpdateTaskFinished(id string, state string, resultJSON string, errMsg string, finishedAt int64, attempt int) error

	// Workers (embedded)
	RegisterWorker(w Worker) error
	HeartbeatWorker(workerID string, capacity int, lastSeen int64) error
	ListWorkers() ([]Worker, error)
	GetWorker(workerID string) (Worker, bool, error)
	DeleteWorker(workerID string) error

	// Embedded Workers (new gRPC-based)
	RegisterEmbeddedWorker(w EmbeddedWorker) error
	HeartbeatEmbeddedWorker(workerID string, lastSeen int64) error
	ListEmbeddedWorkers() ([]EmbeddedWorker, error)
	GetEmbeddedWorker(workerID string) (EmbeddedWorker, bool, error)
	DeleteEmbeddedWorker(workerID string) error

	// Resources
	RegisterResource(r Resource) error
	HeartbeatResource(resourceID string, lastSeen int64) error
	ListResources() ([]Resource, error)
	GetResource(id string) (Resource, bool, error)
	DeleteResource(id string) error
	SubmitResourceState(rs ResourceState) error
	ListResourceStates(resourceID string, limit int) ([]ResourceState, error)

	// Workflows (sequential MVP - legacy)
	CreateWorkflow(wf Workflow) (string, error)
	ListWorkflows() ([]Workflow, error)
	GetWorkflow(id string) (Workflow, bool, error)
	DeleteWorkflow(id string) error
	CreateWorkflowRun(workflowID string) (string, error)
	CreateWorkflowRunWithID(run WorkflowRun) error // 用于DAG运行
	GetWorkflowRun(runID string) (WorkflowRun, bool, error)
	ListWorkflowRuns() ([]WorkflowRun, error)
	ListWorkflowRunsByWorkflow(workflowID string) ([]WorkflowRun, error)
	ListWorkflowSteps(id string) ([]WorkflowStep, error)
	ListStepRuns(runID string) ([]StepRun, error)
	InsertStepRun(sr StepRun) error
	UpdateStepRunTask(runID string, stepID string, taskID string, state string, startedAt int64) error
	UpdateStepRunFinished(runID string, stepID string, state string, finishedAt int64) error
	UpdateWorkflowRunState(runID string, state string, ts int64) error
	DeleteWorkflowRun(runID string) error

	// DAG Workflows (v2)
	CreateWorkflowDAG(dag WorkflowDAG) (string, error)
	GetWorkflowDAG(id string) (WorkflowDAG, bool, error)
	ListWorkflowDAGs() ([]WorkflowDAG, error)
	DeleteWorkflowDAG(id string) error

	// TaskDefinition (for reusable task templates)
	CreateTaskDef(td TaskDefinition) (string, error)
	GetTaskDef(id string) (TaskDefinition, bool, error)
	GetTaskDefByName(name string) (TaskDefinition, bool, error)
	ListTaskDefs() ([]TaskDefinition, error)
	DeleteTaskDef(id string) error

	// References
	CountTasksByOrigin(defID string) (int, error)

	// DistributedKV (namespace-based key-value store)
	PutKV(namespace, key, value, valueType string) error
	GetKV(namespace, key string) (DistributedKV, bool, error)
	DeleteKV(namespace, key string) error
	ListKVByNamespace(namespace string) ([]DistributedKV, error)
	ListKVByPrefix(namespace, prefix string) ([]DistributedKV, error)
	PutKVBatch(namespace string, kvs []DistributedKV) error
	DeleteNamespace(namespace string) error
	ListAllNamespaces() ([]string, error)
	ListKeysByNamespace(namespace string) ([]string, error)
}

// TaskDefinition stores a reusable task template
type TaskDefinition struct {
	DefID      string
	Name       string
	Executor   string
	TargetKind string
	TargetRef  string
	Labels     map[string]string
	// DefaultPayloadJSON stores the default input for runs created from this definition
	DefaultPayloadJSON string
	CreatedAt          int64
}

// DistributedKV stores key-value pairs with namespace isolation
type DistributedKV struct {
	Namespace string
	Key       string
	Value     string
	Type      string // string|int|double|bool
	UpdatedAt int64
}

var Current Store

func SetCurrent(s Store) { Current = s }
