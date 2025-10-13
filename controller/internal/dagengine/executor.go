package dagengine

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

// DAG执行器
type DAGExecutor struct {
	runID      string
	dag        store.WorkflowDAG
	nodeStates map[string]NodeState      // 节点执行状态
	taskIDs    map[string]string         // nodeID -> taskID
	results    map[string]map[string]any // taskID -> result JSON
}

type NodeState string

const (
	NodePending   NodeState = "Pending"
	NodeReady     NodeState = "Ready"
	NodeRunning   NodeState = "Running"
	NodeSucceeded NodeState = "Succeeded"
	NodeFailed    NodeState = "Failed"
	NodeSkipped   NodeState = "Skipped"
)

func NewDAGExecutor(runID string, dag store.WorkflowDAG) *DAGExecutor {
	states := make(map[string]NodeState)
	for nodeID := range dag.Nodes {
		states[nodeID] = NodePending
	}

	return &DAGExecutor{
		runID:      runID,
		dag:        dag,
		nodeStates: states,
		taskIDs:    make(map[string]string),
		results:    make(map[string]map[string]any),
	}
}

// Tick - 调度一轮
func (e *DAGExecutor) Tick(storeInst store.Store) error {
	// 1. 更新节点状态（从Task状态同步）
	e.syncNodeStates(storeInst)

	// 2. 检查可调度的节点
	for nodeID, node := range e.dag.Nodes {
		if e.nodeStates[nodeID] == NodePending && e.isReady(nodeID, node) {
			if err := e.scheduleNode(nodeID, node, storeInst); err != nil {
				log.Printf("[DAGExecutor] Failed to schedule node %s: %v", nodeID, err)
			}
		}
	}

	return nil
}

// 同步节点状态（从Task状态）
func (e *DAGExecutor) syncNodeStates(storeInst store.Store) {
	for nodeID, taskID := range e.taskIDs {
		if taskID == "" {
			continue
		}

		task, ok, err := storeInst.GetTask(taskID)
		if err != nil || !ok {
			continue
		}

		// 映射Task状态到Node状态
		switch task.State {
		case "Running":
			e.nodeStates[nodeID] = NodeRunning
		case "Succeeded":
			e.nodeStates[nodeID] = NodeSucceeded
			// 缓存结果
			if task.ResultJSON != "" {
				var result map[string]any
				if err := json.Unmarshal([]byte(task.ResultJSON), &result); err == nil {
					e.results[taskID] = result
				}
			}
		case "Failed", "Timeout":
			e.nodeStates[nodeID] = NodeFailed
		}
	}
}

// 检查节点是否就绪
func (e *DAGExecutor) isReady(nodeID string, node store.WorkflowNode) bool {
	// 获取前驱节点
	predecessors := e.getPredecessors(nodeID)

	// 没有前驱，可以执行（起始节点）
	if len(predecessors) == 0 {
		return true
	}

	// 根据TriggerRule判断
	switch node.TriggerRule {
	case store.TriggerOneSuccess:
		return e.oneSucceeded(predecessors)
	default: // all_success
		return e.allSucceeded(predecessors)
	}
}

// 获取前驱节点
func (e *DAGExecutor) getPredecessors(nodeID string) []string {
	var preds []string
	for _, edge := range e.dag.Edges {
		if edge.To == nodeID {
			preds = append(preds, edge.From)
		}
	}
	return preds
}

// 获取后继节点
func (e *DAGExecutor) getSuccessors(nodeID string) []string {
	var succs []string
	for _, edge := range e.dag.Edges {
		if edge.From == nodeID {
			succs = append(succs, edge.To)
		}
	}
	return succs
}

// 所有前驱都成功
func (e *DAGExecutor) allSucceeded(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		if e.nodeStates[id] != NodeSucceeded {
			return false
		}
	}
	return len(nodeIDs) > 0
}

// 至少一个前驱成功
func (e *DAGExecutor) oneSucceeded(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		if e.nodeStates[id] == NodeSucceeded {
			return true
		}
	}
	return false
}

// 调度节点
func (e *DAGExecutor) scheduleNode(nodeID string, node store.WorkflowNode, storeInst store.Store) error {
	switch node.Type {
	case store.NodeTypeTask:
		return e.scheduleTaskNode(nodeID, node, storeInst)
	case store.NodeTypeBranch:
		return e.scheduleBranchNode(nodeID, node, storeInst)
	case store.NodeTypeParallel:
		return e.scheduleParallelNode(nodeID, node, storeInst)
	default:
		return fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// 调度Task节点
func (e *DAGExecutor) scheduleTaskNode(nodeID string, node store.WorkflowNode, storeInst store.Store) error {
	// 获取TaskDefinition
	taskDef, ok, err := storeInst.GetTaskDef(node.TaskDefID)
	if err != nil || !ok {
		e.nodeStates[nodeID] = NodeFailed
		return fmt.Errorf("task definition not found: %s", node.TaskDefID)
	}

	// 创建Task
	task := store.Task{
		Name:        node.Name,
		Executor:    taskDef.Executor,
		TargetKind:  taskDef.TargetKind,
		TargetRef:   taskDef.TargetRef,
		State:       "Pending",
		PayloadJSON: node.PayloadJSON,
		TimeoutSec:  node.TimeoutSec,
		MaxRetries:  node.MaxRetries,
		CreatedAt:   time.Now().Unix(),
		Labels:      make(map[string]string),
	}

	// 优先使用node的payload，否则使用taskDef的默认payload
	if task.PayloadJSON == "" {
		task.PayloadJSON = taskDef.DefaultPayloadJSON
	}

	taskID, err := storeInst.CreateTask(task)
	if err != nil {
		e.nodeStates[nodeID] = NodeFailed
		return err
	}

	e.taskIDs[nodeID] = taskID
	e.nodeStates[nodeID] = NodeRunning
	log.Printf("[DAGExecutor] Scheduled task node %s -> task %s", nodeID, taskID)
	return nil
}

// 调度Branch节点
func (e *DAGExecutor) scheduleBranchNode(nodeID string, node store.WorkflowNode, storeInst store.Store) error {
	if node.Condition == nil {
		e.nodeStates[nodeID] = NodeFailed
		return fmt.Errorf("branch node missing condition")
	}

	// 获取source task的结果
	sourceTaskID := e.taskIDs[node.Condition.SourceTask]
	if sourceTaskID == "" {
		e.nodeStates[nodeID] = NodeFailed
		return fmt.Errorf("source task not found: %s", node.Condition.SourceTask)
	}

	result, ok := e.results[sourceTaskID]
	if !ok {
		e.nodeStates[nodeID] = NodeFailed
		return fmt.Errorf("source task result not available")
	}

	// 求值条件
	conditionMet, err := e.evaluateCondition(node.Condition, result)
	if err != nil {
		e.nodeStates[nodeID] = NodeFailed
		return err
	}

	// 标记分支节点为成功
	e.nodeStates[nodeID] = NodeSucceeded

	// 根据条件结果，跳过不选中的分支
	successors := e.getSuccessors(nodeID)
	for _, succID := range successors {
		edge := e.findEdge(nodeID, succID)
		if edge == nil {
			continue
		}

		// 根据edge类型决定是否跳过
		if edge.EdgeType == "true" && !conditionMet {
			e.nodeStates[succID] = NodeSkipped
			e.skipDownstream(succID)
		} else if edge.EdgeType == "false" && conditionMet {
			e.nodeStates[succID] = NodeSkipped
			e.skipDownstream(succID)
		}
	}

	log.Printf("[DAGExecutor] Branch node %s evaluated: %v", nodeID, conditionMet)
	return nil
}

// 调度Parallel节点
func (e *DAGExecutor) scheduleParallelNode(nodeID string, node store.WorkflowNode, storeInst store.Store) error {
	// Parallel节点本身不执行任务，只是一个控制节点
	// 它的所有后继节点会并行触发（因为前驱都满足了）
	e.nodeStates[nodeID] = NodeSucceeded
	log.Printf("[DAGExecutor] Parallel node %s activated", nodeID)
	return nil
}

// 条件求值（简化版：仅支持基本比较）
func (e *DAGExecutor) evaluateCondition(cond *store.BranchCondition, result map[string]any) (bool, error) {
	// 获取字段值
	fieldValue, ok := result[cond.Field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", cond.Field)
	}

	// 转换为字符串进行比较
	leftStr := fmt.Sprintf("%v", fieldValue)
	rightStr := cond.Value

	// 尝试数字比较
	leftNum, leftIsNum := toFloat(leftStr)
	rightNum, rightIsNum := toFloat(rightStr)

	switch cond.Operator {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	case ">":
		if leftIsNum && rightIsNum {
			return leftNum > rightNum, nil
		}
		return false, fmt.Errorf("operator > requires numbers")
	case ">=":
		if leftIsNum && rightIsNum {
			return leftNum >= rightNum, nil
		}
		return false, fmt.Errorf("operator >= requires numbers")
	case "<":
		if leftIsNum && rightIsNum {
			return leftNum < rightNum, nil
		}
		return false, fmt.Errorf("operator < requires numbers")
	case "<=":
		if leftIsNum && rightIsNum {
			return leftNum <= rightNum, nil
		}
		return false, fmt.Errorf("operator <= requires numbers")
	default:
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}
}

func toFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// 查找边
func (e *DAGExecutor) findEdge(from, to string) *store.WorkflowEdge {
	for i := range e.dag.Edges {
		if e.dag.Edges[i].From == from && e.dag.Edges[i].To == to {
			return &e.dag.Edges[i]
		}
	}
	return nil
}

// 跳过下游所有节点
func (e *DAGExecutor) skipDownstream(nodeID string) {
	e.nodeStates[nodeID] = NodeSkipped
	for _, succ := range e.getSuccessors(nodeID) {
		if e.nodeStates[succ] == NodePending {
			e.skipDownstream(succ)
		}
	}
}

// 检查DAG是否完成
func (e *DAGExecutor) IsFinished() (bool, string) {
	allDone := true
	hasFailure := false

	for _, state := range e.nodeStates {
		if state == NodePending || state == NodeReady || state == NodeRunning {
			allDone = false
			break
		}
		if state == NodeFailed {
			hasFailure = true
		}
	}

	if !allDone {
		return false, "Running"
	}

	if hasFailure {
		return true, "Failed"
	}
	return true, "Succeeded"
}

// 获取节点状态（用于前端展示）
func (e *DAGExecutor) GetNodeStates() map[string]string {
	result := make(map[string]string)
	for nodeID, state := range e.nodeStates {
		result[nodeID] = string(state)
	}
	return result
}
