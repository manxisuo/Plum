package dagengine

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

// DAG编排器 - 管理所有DAG运行
type DAGOrchestrator struct {
	store     store.Store
	executors map[string]*DAGExecutor // runID -> executor
	mu        sync.RWMutex
	stopCh    chan struct{}
}

func NewDAGOrchestrator(s store.Store) *DAGOrchestrator {
	return &DAGOrchestrator{
		store:     s,
		executors: make(map[string]*DAGExecutor),
		stopCh:    make(chan struct{}),
	}
}

// Start - 启动编排器
func (o *DAGOrchestrator) Start() {
	go o.loop()
	log.Println("[DAGOrchestrator] Started")
}

// Stop - 停止编排器
func (o *DAGOrchestrator) Stop() {
	close(o.stopCh)
	log.Println("[DAGOrchestrator] Stopped")
}

// 主循环
func (o *DAGOrchestrator) loop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.tick()
		}
	}
}

func (o *DAGOrchestrator) tick() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 处理每个活跃的executor
	for runID, executor := range o.executors {
		if err := executor.Tick(o.store); err != nil {
			log.Printf("[DAGOrchestrator] Executor %s tick error: %v", runID, err)
		}

		// 检查是否完成
		if finished, finalState := executor.IsFinished(); finished {
			log.Printf("[DAGOrchestrator] Run %s finished with state: %s", runID, finalState)

			// 更新WorkflowRun状态
			_ = o.store.UpdateWorkflowRunState(runID, finalState, time.Now().Unix())

			// 移除executor
			delete(o.executors, runID)
		}
	}
}

// StartDAGRun - 启动一个DAG运行
func (o *DAGOrchestrator) StartDAGRun(workflowID string) (string, error) {
	// 获取DAG定义
	dag, ok, err := o.store.GetWorkflowDAG(workflowID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("workflow not found: %s", workflowID)
	}

	// 创建WorkflowRun记录
	run := store.WorkflowRun{
		WorkflowID: workflowID,
		State:      "Running",
		CreatedAt:  time.Now().Unix(),
		StartedAt:  time.Now().Unix(),
	}

	runID := newRunID()
	run.RunID = runID

	// 存储到数据库（复用现有表）
	if err := o.store.CreateWorkflowRunWithID(run); err != nil {
		return "", err
	}

	// 创建executor
	executor := NewDAGExecutor(runID, dag)

	o.mu.Lock()
	o.executors[runID] = executor
	o.mu.Unlock()

	log.Printf("[DAGOrchestrator] Started DAG run %s for workflow %s", runID, workflowID)
	return runID, nil
}

// GetRunStatus - 获取运行状态（包含节点状态）
func (o *DAGOrchestrator) GetRunStatus(runID string) map[string]string {
	o.mu.RLock()
	executor, ok := o.executors[runID]
	o.mu.RUnlock()

	if ok {
		// 运行中：从executor获取实时状态
		return executor.GetNodeStates()
	}

	// 已完成：从Task记录重建节点状态
	tasks, err := o.store.ListTasks()
	if err != nil {
		return nil
	}

	nodeStates := make(map[string]string)

	// 先收集所有Task节点的状态
	for _, task := range tasks {
		if task.Labels != nil && task.Labels["dagRunId"] == runID {
			nodeID := task.Labels["dagNodeId"]
			if nodeID != "" {
				// 映射Task状态到Node状态
				switch task.State {
				case "Succeeded":
					nodeStates[nodeID] = "Succeeded"
				case "Failed":
					nodeStates[nodeID] = "Failed"
				case "Running":
					nodeStates[nodeID] = "Running"
				default:
					nodeStates[nodeID] = "Pending"
				}
			}
		}
	}

	// 获取DAG定义来计算Parallel节点状态
	run, ok, err := o.store.GetWorkflowRun(runID)
	if err != nil || !ok {
		return nodeStates
	}

	dag, ok, err := o.store.GetWorkflowDAG(run.WorkflowID)
	if err != nil || !ok {
		return nodeStates
	}

	// 计算Parallel节点状态
	for nodeID, node := range dag.Nodes {
		if node.Type == store.NodeTypeParallel {
			// 获取Parallel节点的所有子节点
			children := o.getChildren(nodeID, dag)
			if len(children) > 0 {
				// 计算子节点状态来决定Parallel节点状态
				parallelState := o.calculateParallelState(children, nodeStates)
				nodeStates[nodeID] = parallelState
			}
		}
	}

	return nodeStates
}

// 获取节点的直接子节点
func (o *DAGOrchestrator) getChildren(nodeID string, dag store.WorkflowDAG) []string {
	var children []string
	for _, edge := range dag.Edges {
		if edge.From == nodeID {
			children = append(children, edge.To)
		}
	}
	return children
}

// 根据子节点状态计算Parallel节点状态
func (o *DAGOrchestrator) calculateParallelState(children []string, nodeStates map[string]string) string {
	if len(children) == 0 {
		return "Succeeded"
	}

	hasFailed := false
	hasRunning := false
	hasPending := false

	for _, childID := range children {
		state, exists := nodeStates[childID]
		if !exists {
			state = "Pending"
		}

		switch state {
		case "Failed":
			hasFailed = true
		case "Running":
			hasRunning = true
		case "Pending":
			hasPending = true
		}
	}

	// 优先级：Failed > Running > Pending > Succeeded
	if hasFailed {
		return "Failed"
	}
	if hasRunning || hasPending {
		return "Running"
	}

	// 所有子节点都成功，Parallel节点也成功
	return "Succeeded"
}

func newRunID() string {
	return fmt.Sprintf("dagrun-%s-%03d", time.Now().Format("20060102-150405"), time.Now().Nanosecond()/1000000)
}
