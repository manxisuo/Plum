package failover

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"plum/controller/internal/notify"
	"plum/controller/internal/store"
)

type NodeHealth string

const (
	Healthy   NodeHealth = "Healthy"
	Unhealthy NodeHealth = "Unhealthy"
	Unknown   NodeHealth = "Unknown"
)

func ttlSeconds() int64 {
	if v := os.Getenv("HEARTBEAT_TTL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return int64(n)
		}
	}
	return 15
}

func intervalSeconds() int {
	if v := os.Getenv("FAILOVER_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 5
}

func enabled() bool {
	v := os.Getenv("FAILOVER_ENABLED")
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes"
}

// Start launches a background loop that performs failover for assignments on Unhealthy nodes.
func Start() {
	if !enabled() {
		log.Printf("failover: disabled by env")
		return
	}
	rand.Seed(time.Now().UnixNano())
	go func() {
		iv := time.Duration(intervalSeconds()) * time.Second
		ttl := ttlSeconds()
		for {
			time.Sleep(iv)
			doOneLoop(ttl)
		}
	}()
}

func doOneLoop(ttlSec int64) {
	nodes, err := store.Current.ListNodes()
	if err != nil {
		log.Printf("failover: list nodes error: %v", err)
		return
	}
	now := time.Now()
	healthySet := make(map[string]bool)
	unhealthy := make([]string, 0)
	for _, n := range nodes {
		delta := now.Unix() - n.LastSeen.Unix()
		if delta <= ttlSec {
			healthySet[n.NodeID] = true
		} else {
			unhealthy = append(unhealthy, n.NodeID)
		}
	}
	if len(healthySet) == 0 {
		// nothing to migrate to
		return
	}
	for _, bad := range unhealthy {
		migrateNode(bad, healthySet)
	}
}

func migrateNode(badNode string, healthySet map[string]bool) {
	assigns, err := store.Current.ListAssignmentsForNode(badNode)
	if err != nil {
		log.Printf("failover: list assignments for %s error: %v", badNode, err)
		return
	}
	healthyNodes := make([]string, 0, len(healthySet))
	for id := range healthySet {
		healthyNodes = append(healthyNodes, id)
	}
	// random helper
	pick := func() (string, bool) {
		if len(healthyNodes) == 0 {
			return "", false
		}
		return healthyNodes[rand.Intn(len(healthyNodes))], true
	}
	for _, a := range assigns {
		// Only migrate Desired=Running
		if a.Desired != store.DesiredRunning {
			continue
		}
		// Stop old assignment first (idempotent)
		_ = store.Current.UpdateAssignmentDesired(a.InstanceID, store.DesiredStopped)
		target, ok := pick()
		if !ok || target == badNode {
			continue
		}
		// Create new assignment on target
		newIID := store.Current.NewInstanceID(a.TaskID)
		err := store.Current.AddAssignment(target, store.Assignment{
			InstanceID:  newIID,
			TaskID:      a.TaskID,
			NodeID:      target,
			Desired:     store.DesiredRunning,
			ArtifactURL: a.ArtifactURL,
			StartCmd:    a.StartCmd,
		})
		if err != nil {
			log.Printf("failover: add assignment %s->%s error: %v", a.InstanceID, target, err)
		} else {
			log.Printf("failover: migrated instance %s (task %s) from %s to %s as %s", a.InstanceID, a.TaskID, badNode, target, newIID)
			notify.Publish(target)
		}
		// small jitter to avoid burst
		time.Sleep(time.Duration(50+rand.Intn(200)) * time.Millisecond)
	}
}

// ComputeHealth returns nodeId -> health for current nodes.
func ComputeHealth() map[string]NodeHealth {
	ttl := ttlSeconds()
	now := time.Now()
	out := make(map[string]NodeHealth)
	nodes, err := store.Current.ListNodes()
	if err != nil {
		return out
	}
	for _, n := range nodes {
		delta := now.Unix() - n.LastSeen.Unix()
		if delta <= ttl {
			out[n.NodeID] = Healthy
		} else {
			out[n.NodeID] = Unhealthy
		}
	}
	return out
}
