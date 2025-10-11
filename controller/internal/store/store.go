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
	ArtifactID string
	AppName    string
	Version    string
	Path       string // served URL path, e.g. /artifacts/xxx.zip
	SHA256     string
	SizeBytes  int64
	CreatedAt  int64
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

// Workflow (sequential MVP)
type WorkflowStep struct {
	StepID       string
	Name         string            // taskName
	Executor     string            // service|embedded|os_process
	TargetKind   string            // service|deployment|node (for service executor)
	TargetRef    string            // serviceName for service executor
	Labels       map[string]string // service executor labels (servicePath, servicePort, etc.)
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
	DeleteEndpointsForInstance(instanceID string) error
	ListEndpointsByService(serviceName string, version string, protocol string) ([]Endpoint, error)
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

	// Workflows (sequential MVP)
	CreateWorkflow(wf Workflow) (string, error)
	ListWorkflows() ([]Workflow, error)
	GetWorkflow(id string) (Workflow, bool, error)
	DeleteWorkflow(id string) error
	CreateWorkflowRun(workflowID string) (string, error)
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

	// TaskDefinition (for reusable task templates)
	CreateTaskDef(td TaskDefinition) (string, error)
	GetTaskDef(id string) (TaskDefinition, bool, error)
	GetTaskDefByName(name string) (TaskDefinition, bool, error)
	ListTaskDefs() ([]TaskDefinition, error)
	DeleteTaskDef(id string) error

	// References
	CountTasksByOrigin(defID string) (int, error)
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

var Current Store

func SetCurrent(s Store) { Current = s }
