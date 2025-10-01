package state

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

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

type deployment struct {
	DeploymentID string
	Name         string
	Labels       map[string]string
}

type inMemoryStore struct {
	mu          sync.RWMutex
	nodes       map[string]Node             // nodeId -> Node
	assignments map[string][]Assignment     // nodeId -> assignments
	statuses    map[string][]InstanceStatus // instanceId -> history
	deployments map[string]deployment       // deploymentId -> deployment
}

var Store = newStore()

func newStore() *inMemoryStore {
	return &inMemoryStore{
		nodes:       make(map[string]Node),
		assignments: make(map[string][]Assignment),
		statuses:    make(map[string][]InstanceStatus),
		deployments: make(map[string]deployment),
	}
}

func (s *inMemoryStore) UpsertNode(id string, n Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[id] = n
}

func (s *inMemoryStore) ListAssignmentsForNode(nodeID string) []Assignment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.assignments[nodeID]
	out := make([]Assignment, len(list))
	copy(out, list)
	return out
}

func (s *inMemoryStore) AddAssignment(nodeID string, a Assignment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assignments[nodeID] = append(s.assignments[nodeID], a)
}

func (s *inMemoryStore) AppendStatus(instanceID string, st InstanceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[instanceID] = append(s.statuses[instanceID], st)
}

func (s *inMemoryStore) CreateDeployment(name string, labels map[string]string) (string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	s.deployments[id] = deployment{DeploymentID: id, Name: name, Labels: labels}
	return id, []string{}
}

func (s *inMemoryStore) NewInstanceID(deploymentID string) string {
	return deploymentID + "-" + newID()[:8]
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
