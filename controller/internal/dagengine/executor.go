package dagengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

// Loop状态跟踪
type LoopState struct {
	CurrentIteration int                    // 当前迭代次数
	MaxIterations    int                    // 最大迭代次数（-1表示无限制）
	ConditionMet     bool                   // 循环条件是否满足
	LoopVarValue     map[string]interface{} // 循环变量值
}

// DAG执行器
type DAGExecutor struct {
	runID      string
	dag        store.WorkflowDAG
	nodeStates map[string]NodeState      // 节点执行状态
	taskIDs    map[string]string         // nodeID -> taskID
	results    map[string]map[string]any // taskID -> result JSON
	loopStates map[string]*LoopState     // nodeID -> loopState（Loop节点状态）
	mu         sync.RWMutex              // 保护并发访问

	// 外部控制系统集成（可选）
	taskID            string
	initialPayload    map[string]any
	stageControlBase  string
	httpClient        *http.Client
	stagePayloadCache map[string]map[string]any
	nodeOutputs       map[string]map[string]any
	nodeErrors        map[string]string
	nodeStage         map[string]string
	stageBeginSent    map[string]bool
	stageResultSent   map[string]bool
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

func NewDAGExecutor(runID string, dag store.WorkflowDAG, payload map[string]any) *DAGExecutor {
	states := make(map[string]NodeState)
	for nodeID := range dag.Nodes {
		states[nodeID] = NodePending
	}

	exec := &DAGExecutor{
		runID:             runID,
		dag:               dag,
		nodeStates:        states,
		taskIDs:           make(map[string]string),
		results:           make(map[string]map[string]any),
		loopStates:        make(map[string]*LoopState),
		initialPayload:    payload,
		stagePayloadCache: make(map[string]map[string]any),
		nodeOutputs:       make(map[string]map[string]any),
		nodeErrors:        make(map[string]string),
		nodeStage:         make(map[string]string),
		stageBeginSent:    make(map[string]bool),
		stageResultSent:   make(map[string]bool),
	}

	// 从环境变量或 payload 中获取外部控制系统的基础 URL（可选）
	base := os.Getenv("STAGE_CONTROL_BASE")
	if payload != nil {
		if v, ok := payload["stageControlBase"].(string); ok && strings.TrimSpace(v) != "" {
			base = v
		}
		// 兼容旧的 payload 字段名
		if base == "" {
			if v, ok := payload["mainControlBase"].(string); ok && strings.TrimSpace(v) != "" {
				base = v
			}
		}
		if v, ok := payload["taskId"].(string); ok {
			exec.taskID = v
		}
	}

	exec.stageControlBase = strings.TrimRight(base, "/")
	exec.httpClient = &http.Client{Timeout: 5 * time.Second}

	return exec
}

// Tick - 调度一轮
func (e *DAGExecutor) Tick(storeInst store.Store) error {
	e.mu.Lock()
	defer e.mu.Unlock()

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
			e.nodeErrors[nodeID] = ""
			if task.ResultJSON != "" {
				var result map[string]any
				if err := json.Unmarshal([]byte(task.ResultJSON), &result); err == nil {
					if result != nil {
						if stdoutVal, ok := result["stdout"]; ok {
							if stdoutStr, ok := stdoutVal.(string); ok && stdoutStr != "" {
								var stdoutJSON map[string]any
								if err := json.Unmarshal([]byte(stdoutStr), &stdoutJSON); err == nil {
									for k, v := range stdoutJSON {
										result[k] = v
									}
								}
							}
						}
					}
					e.results[taskID] = result
					e.nodeOutputs[nodeID] = result
				}
			}
		case "Failed", "Timeout":
			e.nodeStates[nodeID] = NodeFailed
			e.nodeErrors[nodeID] = task.Error
		}

		state := e.nodeStates[nodeID]
		if (state == NodeSucceeded || state == NodeFailed) && !e.stageResultSent[nodeID] {
			stage := e.nodeStage[nodeID]
			if stage != "" {
				var result map[string]any
				errMsg := ""
				if state == NodeSucceeded {
					result = e.nodeOutputs[nodeID]
				} else {
					errMsg = e.nodeErrors[nodeID]
					if errMsg == "" {
						errMsg = "stage execution failed"
					}
				}
				e.notifyStageResult(nodeID, stage, result, errMsg)
			}
			e.stageResultSent[nodeID] = true
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
	case store.TriggerAllFailed:
		return e.allFailed(predecessors)
	case store.TriggerOneFailed:
		return e.oneFailed(predecessors)
	case store.TriggerAllDone:
		return e.allDone(predecessors)
	case store.TriggerNoneFailed:
		return e.noneFailed(predecessors)
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

// 所有前驱都失败
func (e *DAGExecutor) allFailed(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		if e.nodeStates[id] != NodeFailed {
			return false
		}
	}
	return len(nodeIDs) > 0
}

// 至少一个前驱失败
func (e *DAGExecutor) oneFailed(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		if e.nodeStates[id] == NodeFailed {
			return true
		}
	}
	return false
}

// 所有前驱都完成（成功或失败）
func (e *DAGExecutor) allDone(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		state := e.nodeStates[id]
		if state != NodeSucceeded && state != NodeFailed && state != NodeSkipped {
			return false
		}
	}
	return len(nodeIDs) > 0
}

// 没有前驱失败（所有前驱成功或跳过）
func (e *DAGExecutor) noneFailed(nodeIDs []string) bool {
	for _, id := range nodeIDs {
		state := e.nodeStates[id]
		if state == NodeFailed {
			return false
		}
		// 确保至少有成功的节点，不能全部都是跳过
		if len(nodeIDs) == 1 && state == NodeSkipped {
			return false
		}
	}
	return len(nodeIDs) > 0
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
	case store.NodeTypeLoop:
		return e.scheduleLoopNode(nodeID, node, storeInst)
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

	payloadJSON := node.PayloadJSON
	if payloadJSON == "" {
		payloadJSON = taskDef.DefaultPayloadJSON
	}

	stageKey := e.getStageKey(node, taskDef)
	if stageKey != "" {
		e.nodeStage[nodeID] = stageKey
		// 只有在配置了外部控制系统时才尝试获取阶段 payload
		if e.stageControlBase != "" {
			prepared, skip, prepErr := e.prepareStagePayload(stageKey)
			if prepErr != nil {
				e.nodeErrors[nodeID] = prepErr.Error()
				e.nodeStates[nodeID] = NodeFailed
				log.Printf("[DAGExecutor] Failed to prepare payload for node %s (stage=%s): %v", nodeID, stageKey, prepErr)
				if stageKey != "" && !e.stageResultSent[nodeID] {
					e.notifyStageResult(nodeID, stageKey, nil, prepErr.Error())
					e.stageResultSent[nodeID] = true
				}
				return prepErr
			}
			if skip {
				e.nodeStates[nodeID] = NodeSkipped
				e.stageResultSent[nodeID] = true
				log.Printf("[DAGExecutor] Stage %s skipped for node %s", stageKey, nodeID)
				return nil
			}
			if prepared != "" {
				payloadJSON = prepared
			}
		}
		// 如果没有配置外部控制系统，使用节点本身的 payload（node.PayloadJSON 或 taskDef.DefaultPayloadJSON）
	}

	task := store.Task{
		Name:        taskDef.Name,
		Executor:    taskDef.Executor,
		TargetKind:  taskDef.TargetKind,
		TargetRef:   taskDef.TargetRef,
		State:       "Pending",
		PayloadJSON: payloadJSON,
		TimeoutSec:  node.TimeoutSec,
		MaxRetries:  node.MaxRetries,
		CreatedAt:   time.Now().Unix(),
		Labels: map[string]string{
			"dagRunId":    e.runID,
			"dagNodeId":   nodeID,
			"dagNodeName": node.Name,
		},
	}

	taskID, err := storeInst.CreateTask(task)
	if err != nil {
		e.nodeStates[nodeID] = NodeFailed
		return err
	}

	e.taskIDs[nodeID] = taskID
	e.nodeStates[nodeID] = NodeRunning
	log.Printf("[DAGExecutor] Scheduled task node %s -> task %s", nodeID, taskID)

	if stageKey != "" && !e.stageBeginSent[nodeID] {
		e.notifyStageBegin(nodeID, stageKey)
		e.stageBeginSent[nodeID] = true
	}

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

// 调度Loop节点
func (e *DAGExecutor) scheduleLoopNode(nodeID string, node store.WorkflowNode, storeInst store.Store) error {
	if node.LoopCondition == nil {
		e.nodeStates[nodeID] = NodeFailed
		return fmt.Errorf("loop node missing condition")
	}

	// 初始化Loop状态
	loopState, exists := e.loopStates[nodeID]
	if !exists {
		loopState = &LoopState{
			CurrentIteration: 0,
			MaxIterations:    -1,
			ConditionMet:     true,
			LoopVarValue:     make(map[string]interface{}),
		}
		e.loopStates[nodeID] = loopState
	}

	// 检查循环条件
	shouldContinue, err := e.evaluateLoopCondition(nodeID, node.LoopCondition, loopState)
	if err != nil {
		e.nodeStates[nodeID] = NodeFailed
		return err
	}

	if !shouldContinue {
		// 循环结束
		e.nodeStates[nodeID] = NodeSucceeded
		log.Printf("[DAGExecutor] Loop node %s completed after %d iterations", nodeID, loopState.CurrentIteration)
		return nil
	}

	// 更新循环状态
	loopState.CurrentIteration++
	if loopState.CurrentIteration >= 1 {
		// 设置循环变量值
		if node.LoopCondition.LoopVarName != "" {
			loopState.LoopVarValue[node.LoopCondition.LoopVarName] = loopState.CurrentIteration - 1
		}
	}

	// 重置循环体内节点的状态，让它们重新执行
	successors := e.getSuccessors(nodeID)
	for _, succID := range successors {
		if succNode, ok := e.dag.Nodes[succID]; ok {
			// 只重置循环体内的Task节点
			if succNode.Type == store.NodeTypeTask {
				e.nodeStates[succID] = NodePending
				// 清除相关的Task ID，让它们重新创建
				delete(e.taskIDs, succID)
			}
		}
	}

	log.Printf("[DAGExecutor] Loop node %s iteration %d started", nodeID, loopState.CurrentIteration)
	return nil
}

// 评估Loop条件
func (e *DAGExecutor) evaluateLoopCondition(nodeID string, condition *store.LoopCondition, loopState *LoopState) (bool, error) {
	switch condition.Type {
	case "count":
		// 基于计数的循环
		if condition.Count <= 0 {
			return false, fmt.Errorf("invalid loop count: %d", condition.Count)
		}
		return loopState.CurrentIteration < condition.Count, nil

	case "condition":
		// 基于条件的循环
		if condition.SourceTask == "" {
			return false, fmt.Errorf("loop condition requires sourceTask")
		}

		// 获取源任务的结果
		sourceTaskID := e.taskIDs[condition.SourceTask]
		if sourceTaskID == "" {
			return false, fmt.Errorf("source task not found: %s", condition.SourceTask)
		}

		result, ok := e.results[sourceTaskID]
		if !ok {
			return false, fmt.Errorf("source task result not available")
		}

		// 获取字段值
		fieldValue, ok := result[condition.Field]
		if !ok {
			return false, fmt.Errorf("field not found: %s", condition.Field)
		}

		// 转换并比较
		leftStr := fmt.Sprintf("%v", fieldValue)
		rightStr := condition.Value

		leftNum, leftIsNum := toFloat(leftStr)
		rightNum, rightIsNum := toFloat(rightStr)

		switch condition.Operator {
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
			return false, fmt.Errorf("unknown operator: %s", condition.Operator)
		}

	default:
		return false, fmt.Errorf("unknown loop condition type: %s", condition.Type)
	}
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

func toStringMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if m, ok := value.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

func marshalJSON(data map[string]any) (string, error) {
	if data == nil {
		return "", nil
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(buf), nil
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
	e.mu.RLock()
	defer e.mu.RUnlock()

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
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]string)
	for nodeID, state := range e.nodeStates {
		result[nodeID] = string(state)
	}
	return result
}

func (e *DAGExecutor) getStageKey(node store.WorkflowNode, def store.TaskDefinition) string {
	// 只使用显式的 labels 来识别阶段，不进行业务相关的猜测
	if def.Labels != nil {
		if stage := strings.TrimSpace(def.Labels["workflow.stage"]); stage != "" {
			return strings.ToLower(stage)
		}
		if stage := strings.TrimSpace(def.Labels["stage"]); stage != "" {
			return strings.ToLower(stage)
		}
	}
	// 如果 labels 不存在，返回空字符串，不尝试从名称中猜测
	return ""
}

func (e *DAGExecutor) prepareStagePayload(stage string) (string, bool, error) {
	if stage == "" {
		return "", false, nil
	}

	if cached, ok := e.stagePayloadCache[stage]; ok {
		txt, err := marshalJSON(cached)
		return txt, false, err
	}

	payload, skip, err := e.fetchStagePayload(stage)
	if err != nil || skip {
		return "", skip, err
	}

	e.stagePayloadCache[stage] = payload
	txt, err := marshalJSON(payload)
	return txt, false, err
}

func (e *DAGExecutor) fetchStagePayload(stage string) (map[string]any, bool, error) {
	if e.taskID == "" {
		return nil, false, fmt.Errorf("task payload missing taskId for stage %s", stage)
	}
	if e.stageControlBase == "" {
		return nil, false, fmt.Errorf("stage control base URL not configured")
	}

	url := fmt.Sprintf("%s/api/task/%s/stage/%s/input", e.stageControlBase, e.taskID, stage)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[DAGExecutor] Stage %s not applicable: %s", stage, strings.TrimSpace(string(body)))
		return nil, true, nil
	}

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("stage input request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false, err
	}
	return data, false, nil
}

func (e *DAGExecutor) notifyStageBegin(nodeID, stage string) {
	if e.taskID == "" || e.stageControlBase == "" {
		return
	}
	url := fmt.Sprintf("%s/api/task/%s/stage/%s/begin", e.stageControlBase, e.taskID, stage)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString("{}"))
	if err != nil {
		log.Printf("[DAGExecutor] Failed to build stage begin request (stage=%s): %v", stage, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		log.Printf("[DAGExecutor] Stage begin request failed (stage=%s): %v", stage, err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[DAGExecutor] Stage begin request failed (stage=%s) status=%d", stage, resp.StatusCode)
	} else {
		log.Printf("[DAGExecutor] Stage begin recorded (stage=%s)", stage)
	}
}

func (e *DAGExecutor) notifyStageResult(nodeID, stage string, result map[string]any, errMsg string) {
	if e.taskID == "" || e.stageControlBase == "" {
		return
	}

	payload := make(map[string]any, len(result)+1)
	if errMsg != "" {
		payload["status"] = "error"
		payload["message"] = errMsg
	} else if result != nil {
		for k, v := range result {
			payload[k] = v
		}
		if _, ok := payload["status"]; !ok {
			payload["status"] = "success"
		}
	} else {
		payload["status"] = "success"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DAGExecutor] Failed to marshal stage result payload (stage=%s): %v", stage, err)
		return
	}

	url := fmt.Sprintf("%s/api/task/%s/stage/%s/result", e.stageControlBase, e.taskID, stage)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[DAGExecutor] Failed to build stage result request (stage=%s): %v", stage, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		log.Printf("[DAGExecutor] Stage result request failed (stage=%s): %v", stage, err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[DAGExecutor] Stage result request failed (stage=%s) status=%d", stage, resp.StatusCode)
	} else {
		log.Printf("[DAGExecutor] Stage result recorded (stage=%s)", stage)
	}
}
