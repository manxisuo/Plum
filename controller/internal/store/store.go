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
	InstanceID  string
	TaskID      string
	NodeID      string
	Desired     DesiredState
	ArtifactURL string
	StartCmd    string
}

type InstanceStatus struct {
	InstanceID string
	Phase      string
	ExitCode   int
	Healthy    bool
	TsUnix     int64
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

type Task struct {
	TaskID string
	Name   string
	Labels map[string]string
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
	DeleteAssignmentsForTask(taskID string) error
	UpdateAssignmentDesired(instanceID string, desired DesiredState) error
	AppendStatus(instanceID string, st InstanceStatus) error
	LatestStatus(instanceID string) (InstanceStatus, bool, error)

	CountAssignmentsByArtifactPath(path string) (int, error)
	CountAssignmentsForNode(nodeID string) (int, error)
	CreateTask(name string, labels map[string]string) (string, []string, error)
	NewInstanceID(taskID string) string

	SaveArtifact(a Artifact) (string, error)
	ListArtifacts() ([]Artifact, error)
	GetArtifact(id string) (Artifact, bool, error)
	DeleteArtifact(id string) error

	ListTasks() ([]Task, error)
	GetTask(id string) (Task, bool, error)
	DeleteTask(id string) error
	ListAssignmentsForTask(taskID string) ([]Assignment, error)

	// Services / discovery
	ReplaceEndpointsForInstance(nodeID string, instanceID string, eps []Endpoint) error
	UpdateEndpointHealthForInstance(instanceID string, eps []Endpoint) error
	DeleteEndpointsForInstance(instanceID string) error
	ListEndpointsByService(serviceName string, version string, protocol string) ([]Endpoint, error)
	ListServices() ([]string, error)
}

var Current Store

func SetCurrent(s Store) { Current = s }
