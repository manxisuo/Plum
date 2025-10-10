package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/manxisuo/plum/controller/internal/grpc"
	"github.com/manxisuo/plum/controller/internal/notify"
	"github.com/manxisuo/plum/controller/internal/store"
)

func intervalSeconds() int {
	if v := os.Getenv("TASK_SCHED_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
}

// Start a minimal scheduler: Pending -> Running -> Succeeded (builtin only)
func Start() {
	go func() {
		iv := time.Duration(intervalSeconds()) * time.Second
		for {
			time.Sleep(iv)
			tick()
		}
	}()
}

func tick() {
	tasks, err := store.Current.ListTasks()
	if err != nil {
		return
	}
	now := time.Now().Unix()
	// reflect task state to workflow stepRuns
	for _, t := range tasks {
		if t.Labels != nil {
			runID := t.Labels["runId"]
			stepID := t.Labels["stepId"]
			if runID != "" && stepID != "" {
				switch t.State {
				case "Running":
					_ = store.Current.UpdateStepRunTask(runID, stepID, t.TaskID, "Running", t.StartedAt)
				case "Succeeded", "Failed", "Timeout", "Canceled":
					_ = store.Current.UpdateStepRunFinished(runID, stepID, t.State, t.FinishedAt)
				}
			}
		}
	}
	// workflow sequential progression
	if runs, err := store.Current.ListWorkflowRuns(); err == nil {
		for _, r := range runs {
			if r.State != "Running" {
				continue
			}
			steps, _ := store.Current.ListWorkflowSteps(r.WorkflowID)
			srs, _ := store.Current.ListStepRuns(r.RunID)
			nextOrd := -1
			if len(srs) == 0 {
				nextOrd = 0
			} else {
				last := srs[len(srs)-1]
				if last.State == "Succeeded" {
					nextOrd = last.Ord + 1
				} else if last.State == "Failed" {
					_ = store.Current.UpdateWorkflowRunState(r.RunID, "Failed", now)
					continue
				}
			}
			if nextOrd >= 0 && nextOrd < len(steps) {
				st := steps[nextOrd]
				name := st.Name
				executor := st.Executor
				targetKind := st.TargetKind
				targetRef := st.TargetRef
				labels := map[string]string{"workflowId": r.WorkflowID, "runId": r.RunID, "stepId": st.StepID}
				// Merge step labels first (workflow step labels take precedence)
				if st.Labels != nil {
					for k, v := range st.Labels {
						labels[k] = v
					}
				}
				payloadJSON := "{}"
				if st.DefinitionID != "" {
					if td, ok, _ := store.Current.GetTaskDef(st.DefinitionID); ok {
						if td.Name != "" {
							name = td.Name
						}
						if td.Executor != "" {
							executor = td.Executor
						}
						if td.TargetKind != "" {
							targetKind = td.TargetKind
						}
						if td.TargetRef != "" {
							targetRef = td.TargetRef
						}
						// Use TaskDefinition's default payload if available
						if td.DefaultPayloadJSON != "" {
							payloadJSON = td.DefaultPayloadJSON
						}
						// TaskDefinition labels only override if not already set by step
						for k, v := range td.Labels {
							if _, exists := labels[k]; !exists {
								labels[k] = v
							}
						}
						labels["defId"] = td.DefID
					}
				}
				newID, _ := store.Current.CreateTask(store.Task{Name: name, Executor: executor, TargetKind: targetKind, TargetRef: targetRef, State: "Pending", PayloadJSON: payloadJSON, TimeoutSec: st.TimeoutSec, MaxRetries: st.MaxRetries, CreatedAt: now, Labels: labels})
				_ = store.Current.InsertStepRun(store.StepRun{RunID: r.RunID, StepID: st.StepID, TaskID: newID, State: "Pending", Ord: st.Ord})
				notify.PublishTasks()
			} else if nextOrd >= len(steps) {
				_ = store.Current.UpdateWorkflowRunState(r.RunID, "Succeeded", now)
			}
		}
	}
	for _, t := range tasks {
		if t.State == "Pending" {
			// minimal: mark Running
			_ = store.Current.UpdateTaskRunning(t.TaskID, now, "controller", t.Attempt+1)
			// builtin executors: Name prefix "builtin." executes locally
			if len(t.Name) >= 8 && t.Name[:8] == "builtin." {
				runBuiltin(t)
			} else if t.Executor == "embedded" {
				runEmbedded(t)
			} else if t.Executor == "service" {
				runService(t)
			} else if t.Executor == "os_process" {
				runOSProcess(t)
			}
			notify.PublishTasks()
		}
	}
	// watchdog: mark long-running running tasks as Failed
	for _, t := range tasks {
		if t.State == "Running" && t.StartedAt > 0 {
			if now-t.StartedAt >= int64(embeddedTimeoutMs()/1000+5) {
				_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "controller watchdog timeout", now, t.Attempt)
			}
		}
	}
}

func runBuiltin(t store.Task) {
	// simulate simple builtins: builtin.echo, builtin.sleep, builtin.delay, builtin.fail
	var payload map[string]any
	_ = json.Unmarshal([]byte(t.PayloadJSON), &payload)
	switch t.Name {
	case "builtin.echo":
		res := map[string]any{"echo": payload}
		b, _ := json.Marshal(res)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", string(b), "", time.Now().Unix(), t.Attempt)
	case "builtin.sleep":
		d := 1.0
		if v, ok := payload["seconds"]; ok {
			switch vv := v.(type) {
			case float64:
				d = vv
			}
		}
		time.Sleep(time.Duration(d*1000) * time.Millisecond)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", "{}", "", time.Now().Unix(), t.Attempt)
	case "builtin.delay":
		// builtin.delay: 默认延迟3秒，可通过payload指定秒数
		d := 3.0
		if v, ok := payload["seconds"]; ok {
			switch vv := v.(type) {
			case float64:
				d = vv
			case int:
				d = float64(vv)
			}
		}
		time.Sleep(time.Duration(d*1000) * time.Millisecond)
		res := map[string]any{"message": fmt.Sprintf("Delayed for %.1f seconds", d), "seconds": d}
		b, _ := json.Marshal(res)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", string(b), "", time.Now().Unix(), t.Attempt)
	case "builtin.fail":
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "builtin fail", time.Now().Unix(), t.Attempt)
	default:
		log.Printf("tasks: unknown builtin %s", t.Name)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "unknown builtin", time.Now().Unix(), t.Attempt)
	}
	notify.PublishTasks()
}

func embeddedTimeoutMs() int {
	if v := os.Getenv("TASK_EMBEDDED_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 5000
}

func runEmbedded(t store.Task) {
	// First try new gRPC-based embedded workers
	embeddedWorkers, err := store.Current.ListEmbeddedWorkers()
	if err == nil && len(embeddedWorkers) > 0 {
		if runEmbeddedGRPC(t, embeddedWorkers) {
			return
		}
	}

	// Fallback to legacy HTTP-based workers
	workers, err := store.Current.ListWorkers()
	if err != nil || len(workers) == 0 {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "no workers available", time.Now().Unix(), t.Attempt)
		return
	}

	if runEmbeddedHTTP(t, workers) {
		return
	}

	// If both fail, mark task as failed
	_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "no suitable worker found", time.Now().Unix(), t.Attempt)
}

func runEmbeddedGRPC(t store.Task, embeddedWorkers []store.EmbeddedWorker) bool {
	var candidates []*store.EmbeddedWorker
	// First, find all workers that support this task name
	for i := range embeddedWorkers {
		w := &embeddedWorkers[i]
		for _, name := range w.Tasks {
			if name == t.Name {
				candidates = append(candidates, w)
				break
			}
		}
	}

	if len(candidates) == 0 {
		return false
	}

	// Filter by target type if specified
	var candidate *store.EmbeddedWorker
	if t.TargetKind == "node" && t.TargetRef != "" {
		// Find worker on specific node
		for _, w := range candidates {
			if w.NodeID == t.TargetRef {
				candidate = w
				break
			}
		}
	} else if t.TargetKind == "app" && t.TargetRef != "" {
		// For app, find workers that match the app name
		for _, w := range candidates {
			if w.AppName == t.TargetRef {
				candidate = w
				break
			}
		}
	} else {
		// No specific target, pick first available
		candidate = candidates[0]
	}

	if candidate == nil {
		return false
	}

	// Create gRPC client and execute task
	client, err := grpc.NewTaskClient(candidate.GRPCAddress)
	if err != nil {
		log.Printf("Failed to create gRPC client for worker %s: %v", candidate.WorkerID, err)
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(embeddedTimeoutMs())*time.Millisecond)
	defer cancel()

	log.Printf("tasks: dispatch %s to gRPC worker %s address=%s", t.Name, candidate.WorkerID, candidate.GRPCAddress)
	result, err := client.ExecuteTask(ctx, t.TaskID, t.Name, t.PayloadJSON)
	if err != nil {
		log.Printf("tasks: gRPC call error: %v", err)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", err.Error(), time.Now().Unix(), t.Attempt)
		return true
	}

	_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", result, "", time.Now().Unix(), t.Attempt)
	return true
}

func runEmbeddedHTTP(t store.Task, workers []store.Worker) bool {
	var candidates []*store.Worker
	// First, find all workers that support this task name
	for i := range workers {
		w := &workers[i]
		for _, name := range w.Tasks {
			if name == t.Name {
				candidates = append(candidates, w)
				break
			}
		}
	}

	if len(candidates) == 0 {
		return false
	}

	// Filter by target type if specified
	var candidate *store.Worker
	if t.TargetKind == "node" && t.TargetRef != "" {
		// Find worker on specific node
		for _, w := range candidates {
			if w.NodeID == t.TargetRef {
				candidate = w
				break
			}
		}
	} else if t.TargetKind == "deployment" && t.TargetRef != "" {
		// For deployment, we need to find workers that are part of this deployment
		for _, w := range candidates {
			if w.Labels != nil && w.Labels["deploymentId"] == t.TargetRef {
				candidate = w
				break
			}
		}
	} else if t.TargetKind == "app" && t.TargetRef != "" {
		// For app, find workers that are part of this application/service group
		// Support both old "serviceName" and new "appName" labels for backward compatibility
		for _, w := range candidates {
			if w.Labels != nil && (w.Labels["appName"] == t.TargetRef || w.Labels["serviceName"] == t.TargetRef) {
				candidate = w
				break
			}
		}
	} else {
		// No target specified or unsupported target type, use first available worker
		candidate = candidates[0]
	}

	if candidate == nil || candidate.URL == "" {
		return false
	}

	log.Printf("tasks: dispatch %s to HTTP worker %s url=%s", t.Name, candidate.WorkerID, candidate.URL)
	// synchronous POST to worker URL with task payload
	m := map[string]any{"taskId": t.TaskID, "name": t.Name}
	var payload any
	_ = json.Unmarshal([]byte(t.PayloadJSON), &payload)
	m["payload"] = payload
	bs, _ := json.Marshal(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(embeddedTimeoutMs())*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, candidate.URL, bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("tasks: call worker error: %v", err)
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", err.Error(), time.Now().Unix(), t.Attempt)
		return true
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", string(rb), "", time.Now().Unix(), t.Attempt)
	} else {
		log.Printf("tasks: worker responded %d body=%s", resp.StatusCode, string(rb))
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", string(rb), resp.Status, time.Now().Unix(), t.Attempt)
	}
	return true
}

// runService dispatches task to a healthy service endpoint discovered from registry.
// Expectation: t.TargetKind == "service" and t.TargetRef == serviceName
func runService(t store.Task) {
	serviceName := t.TargetRef
	if serviceName == "" {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "missing TargetRef(serviceName)", time.Now().Unix(), t.Attempt)
		return
	}
	// Optional overrides via labels
	desiredVersion := ""
	desiredProtocol := ""
	desiredPort := 0
	callPath := "/task"
	if t.Labels != nil {
		if v := t.Labels["serviceVersion"]; v != "" {
			desiredVersion = v
		}
		if v := t.Labels["serviceProtocol"]; v != "" {
			desiredProtocol = v
		}
		if v := t.Labels["servicePort"]; v != "" {
			if p, err := strconv.Atoi(v); err == nil && p > 0 {
				desiredPort = p
			}
		}
		if v := t.Labels["servicePath"]; v != "" {
			if v[0] != '/' {
				v = "/" + v
			}
			callPath = v
		}
	}
	// Discover healthy endpoints
	eps, err := store.Current.ListEndpointsByService(serviceName, desiredVersion, desiredProtocol)
	if err != nil || len(eps) == 0 {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "no healthy endpoints", time.Now().Unix(), t.Attempt)
		return
	}
	// pick first endpoint (MVP; future: random/rr/hash)
	ep := eps[0]
	// Build URL. For MVP we call fixed path "/task" on instance
	scheme := "http"
	if desiredProtocol != "" {
		scheme = desiredProtocol
	} else if ep.Protocol != "" {
		scheme = ep.Protocol
	}
	port := ep.Port
	if desiredPort > 0 {
		port = desiredPort
	}
	url := scheme + "://" + ep.IP + ":" + strconv.Itoa(port) + callPath

	var payload any
	_ = json.Unmarshal([]byte(t.PayloadJSON), &payload)
	body := map[string]any{"taskId": t.TaskID, "name": t.Name, "payload": payload}
	bs, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(embeddedTimeoutMs())*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", err.Error(), time.Now().Unix(), t.Attempt)
		return
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", string(rb), "", time.Now().Unix(), t.Attempt)
	} else {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", string(rb), resp.Status, time.Now().Unix(), t.Attempt)
	}
}

// runOSProcess executes a task by launching an external OS process
func runOSProcess(t store.Task) {
	var payload map[string]any
	_ = json.Unmarshal([]byte(t.PayloadJSON), &payload)

	// Parse command and arguments from payload
	// Expected payload format: {"command": "ls", "args": ["-la", "/tmp"], "workingDir": "/tmp"}
	command, ok := payload["command"].(string)
	if !ok || command == "" {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", "{}", "command is required in payload", time.Now().Unix(), t.Attempt)
		return
	}

	var args []string
	if argsInterface, exists := payload["args"]; exists {
		if argsList, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	// Get working directory
	workingDir := ""
	if wd, exists := payload["workingDir"]; exists {
		if wdStr, ok := wd.(string); ok {
			workingDir = wdStr
		}
	}

	// Create context with timeout
	timeoutSec := t.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 300 // default 5 minutes
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up input/output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Add input from payload if provided
	if input, exists := payload["input"]; exists {
		if inputStr, ok := input.(string); ok {
			cmd.Stdin = strings.NewReader(inputStr)
		}
	}

	// Set environment variables if provided
	if env, exists := payload["env"]; exists {
		if envMap, ok := env.(map[string]interface{}); ok {
			envVars := os.Environ() // inherit current environment
			for key, value := range envMap {
				if valueStr, ok := value.(string); ok {
					envVars = append(envVars, fmt.Sprintf("%s=%s", key, valueStr))
				}
			}
			cmd.Env = envVars
		}
	}

	// Execute the command
	startTime := time.Now()
	err := cmd.Run()
	finishTime := time.Now()

	// Prepare result
	result := map[string]interface{}{
		"command":    command,
		"args":       args,
		"workingDir": workingDir,
		"exitCode":   cmd.ProcessState.ExitCode(),
		"durationMs": finishTime.Sub(startTime).Milliseconds(),
		"stdout":     stdout.String(),
		"stderr":     stderr.String(),
	}

	resultJSON, _ := json.Marshal(result)

	// Determine task result based on exit code and context
	if ctx.Err() == context.DeadlineExceeded {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Timeout", string(resultJSON), "process timeout", finishTime.Unix(), t.Attempt)
	} else if err != nil {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", string(resultJSON), err.Error(), finishTime.Unix(), t.Attempt)
	} else if cmd.ProcessState.ExitCode() == 0 {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Succeeded", string(resultJSON), "", finishTime.Unix(), t.Attempt)
	} else {
		_ = store.Current.UpdateTaskFinished(t.TaskID, "Failed", string(resultJSON), fmt.Sprintf("process exited with code %d", cmd.ProcessState.ExitCode()), finishTime.Unix(), t.Attempt)
	}
}
