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

type Deployment struct {
	DeploymentID string
	Name         string
	Labels       map[string]string
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

// Embedded Worker capability (executor=embedded)
type Worker struct {
	WorkerID string
	NodeID   string
	URL      string   // callback endpoint (optional for MVP)
	Tasks    []string // supported task names
	Labels   map[string]string
	Capacity int
	LastSeen int64
}

// Workflow (sequential MVP)
type WorkflowStep struct {
	StepID     string
	Name       string // taskName
	Executor   string // service|embedded|os_process
	TimeoutSec int
	MaxRetries int
	Ord        int // sequence order
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
	DeleteArtifact(id string) error

	ListDeployments() ([]Deployment, error)
	GetDeployment(id string) (Deployment, bool, error)
	DeleteDeployment(id string) error
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

	// Workflows (sequential MVP)
	CreateWorkflow(wf Workflow) (string, error)
	ListWorkflows() ([]Workflow, error)
	GetWorkflow(id string) (Workflow, bool, error)
	CreateWorkflowRun(workflowID string) (string, error)
	GetWorkflowRun(runID string) (WorkflowRun, bool, error)
	ListWorkflowRuns() ([]WorkflowRun, error)
	ListWorkflowSteps(id string) ([]WorkflowStep, error)
	ListStepRuns(runID string) ([]StepRun, error)
	InsertStepRun(sr StepRun) error
	UpdateStepRunTask(runID string, stepID string, taskID string, state string, startedAt int64) error
	UpdateStepRunFinished(runID string, stepID string, state string, finishedAt int64) error
	UpdateWorkflowRunState(runID string, state string, ts int64) error
}

var Current Store

func SetCurrent(s Store) { Current = s }
