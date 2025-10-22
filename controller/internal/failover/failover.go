package failover

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/manxisuo/plum/controller/internal/notify"
	"github.com/manxisuo/plum/controller/internal/store"
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
	// 优化：降低心跳TTL从15秒到3秒，实现快速故障检测
	return 3
}

func intervalSeconds() int {
	if v := os.Getenv("FAILOVER_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	// 优化：降低故障转移间隔从5秒到1秒，实现快速迁移
	return 1
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
	// 优化：并行迁移多个应用，减少迁移延迟
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数为5，避免过载

	for _, a := range assigns {
		// Only migrate Desired=Running
		if a.Desired != store.DesiredRunning {
			continue
		}

		wg.Add(1)
		go func(assignment store.Assignment) {
			defer wg.Done()

			// 性能监控：记录迁移开始时间
			migrationStart := time.Now()

			// 获取信号量，限制并发
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Stop old assignment first (idempotent)
			_ = store.Current.UpdateAssignmentDesired(assignment.InstanceID, store.DesiredStopped)
			target, ok := pick()
			if !ok || target == badNode {
				return
			}
			// Create new assignment on target
			newIID := store.Current.NewInstanceID(assignment.DeploymentID)
			err := store.Current.AddAssignment(target, store.Assignment{
				InstanceID:   newIID,
				DeploymentID: assignment.DeploymentID,
				NodeID:       target,
				Desired:      store.DesiredRunning,
				ArtifactURL:  assignment.ArtifactURL,
				StartCmd:     assignment.StartCmd,
				AppName:      assignment.AppName,    // 复制应用名称
				AppVersion:   assignment.AppVersion, // 复制应用版本
			})
			if err != nil {
				log.Printf("failover: add assignment %s->%s error: %v", assignment.InstanceID, target, err)
			} else {
				// 性能监控：记录迁移完成时间
				migrationDuration := time.Since(migrationStart)
				log.Printf("性能监控: 实例 %s 迁移耗时 %v", assignment.InstanceID, migrationDuration)

				// 检查是否超过2秒阈值
				if migrationDuration > 2*time.Second {
					log.Printf("⚠️  性能警告: 实例 %s 迁移时间 %v 超过2秒阈值", assignment.InstanceID, migrationDuration)
				}

				log.Printf("failover: migrated instance %s (deployment %s) from %s to %s as %s", assignment.InstanceID, assignment.DeploymentID, badNode, target, newIID)
				notify.Publish(target)
			}
		}(a)
	}

	// 等待所有迁移完成
	wg.Wait()
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
