package grpc

import (
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/manxisuo/plum/controller/internal/store"
	"github.com/manxisuo/plum/controller/proto"
)

// WorkerConnection 表示一个 worker 连接
type WorkerConnection struct {
	WorkerID   string
	NodeID     string
	InstanceID string
	AppName    string
	AppVersion string
	Tasks      []string
	Labels     map[string]string
	Stream     proto.TaskService_TaskStreamServer
	TaskChan   chan *proto.TaskRequest
	LastSeen   time.Time
	taskMap    map[string]*proto.TaskRequest // task_id -> TaskRequest 映射
	mu         sync.RWMutex
}

// TaskStreamServer gRPC 服务端实现
type TaskStreamServer struct {
	proto.UnimplementedTaskServiceServer
	workers   map[string]*WorkerConnection
	workersMu sync.RWMutex
	store     store.Store
}

// NewTaskStreamServer 创建新的 gRPC 服务端
func NewTaskStreamServer(s store.Store) *TaskStreamServer {
	return &TaskStreamServer{
		workers: make(map[string]*WorkerConnection),
		store:   s,
	}
}

// TaskStream 实现双向流 RPC
func (s *TaskStreamServer) TaskStream(stream proto.TaskService_TaskStreamServer) error {
	ctx := stream.Context()
	var workerConn *WorkerConnection
	var registered bool

	for {
		select {
		case <-ctx.Done():
			if workerConn != nil {
				s.removeWorker(workerConn.WorkerID)
			}
			return ctx.Err()
		default:
			// 接收来自 worker 的消息
			ack, err := stream.Recv()
			if err != nil {
				if workerConn != nil {
					log.Printf("[gRPC] Worker %s disconnected: %v", workerConn.WorkerID, err)
					s.removeWorker(workerConn.WorkerID)
				}
				return err
			}

			// 处理不同类型的消息
			switch msg := ack.Message.(type) {
			case *proto.TaskAck_Register:
				// Worker 注册
				if !registered {
					reg := msg.Register
					workerConn = &WorkerConnection{
						WorkerID:   reg.WorkerId,
						NodeID:     reg.NodeId,
						InstanceID: reg.InstanceId,
						AppName:    reg.AppName,
						AppVersion: reg.AppVersion,
						Tasks:      reg.Tasks,
						Labels:     reg.Labels,
						Stream:     stream,
						TaskChan:   make(chan *proto.TaskRequest, 10),
						LastSeen:   time.Now(),
						taskMap:    make(map[string]*proto.TaskRequest),
					}
					s.addWorker(workerConn)
					registered = true
					log.Printf("[gRPC] Worker registered: %s (node=%s, app=%s, tasks=%v)",
						workerConn.WorkerID, workerConn.NodeID, workerConn.AppName, workerConn.Tasks)

					// 保存到数据库（用于 UI 显示）
					worker := store.EmbeddedWorker{
						WorkerID:    reg.WorkerId,
						NodeID:      reg.NodeId,
						InstanceID:  reg.InstanceId,
						AppName:     reg.AppName,
						AppVersion:  reg.AppVersion,
						GRPCAddress: "", // 流式模式下不需要 GRPCAddress
						Tasks:       reg.Tasks,
						Labels:      reg.Labels,
						LastSeen:    time.Now().Unix(),
					}
					if err := s.store.RegisterEmbeddedWorker(worker); err != nil {
						log.Printf("[gRPC] Failed to save worker to database: %v", err)
					} else {
						log.Printf("[gRPC] Worker saved to database: %s", reg.WorkerId)
					}

					// 在后台处理任务推送
					go s.handleTaskPush(workerConn)
				}

			case *proto.TaskAck_Result:
				// 任务执行结果
				if workerConn != nil {
					result := msg.Result
					taskID := result.TaskId
					if taskID == "" {
						log.Printf("[gRPC] Task result missing task_id from worker %s", workerConn.WorkerID)
						continue
					}

					log.Printf("[gRPC] Task result from worker %s: task_id=%s, error=%v",
						workerConn.WorkerID, taskID, result.Error != "")

					// 更新任务状态
					if result.Error != "" {
						s.store.UpdateTaskFinished(taskID, "Failed", "{}", result.Error, time.Now().Unix(), 0)
					} else {
						s.store.UpdateTaskFinished(taskID, "Succeeded", result.Result, "", time.Now().Unix(), 0)
					}

					// 清理任务映射
					workerConn.mu.Lock()
					delete(workerConn.taskMap, taskID)
					workerConn.mu.Unlock()
				}

			case *proto.TaskAck_Heartbeat:
				// 心跳
				if workerConn != nil {
					workerConn.mu.Lock()
					workerConn.LastSeen = time.Now()
					workerConn.mu.Unlock()

					// 更新数据库中的心跳时间
					if err := s.store.HeartbeatEmbeddedWorker(workerConn.WorkerID, time.Now().Unix()); err != nil {
						log.Printf("[gRPC] Failed to update heartbeat in database: %v", err)
					}
				}
			}
		}
	}
}

// handleTaskPush 处理任务推送
func (s *TaskStreamServer) handleTaskPush(worker *WorkerConnection) {
	for {
		select {
		case task := <-worker.TaskChan:
			if err := worker.Stream.Send(task); err != nil {
				log.Printf("[gRPC] Failed to send task to worker %s: %v", worker.WorkerID, err)
				s.removeWorker(worker.WorkerID)
				return
			}
			// 保存任务映射
			worker.mu.Lock()
			worker.taskMap[task.TaskId] = task
			worker.mu.Unlock()
			log.Printf("[gRPC] Task dispatched to worker %s: task_id=%s, name=%s",
				worker.WorkerID, task.TaskId, task.Name)
		case <-worker.Stream.Context().Done():
			return
		}
	}
}

// PushTask 推送任务到 worker
func (s *TaskStreamServer) PushTask(workerID string, task *proto.TaskRequest) bool {
	s.workersMu.RLock()
	worker, exists := s.workers[workerID]
	s.workersMu.RUnlock()

	if !exists {
		return false
	}

	select {
	case worker.TaskChan <- task:
		return true
	default:
		log.Printf("[gRPC] Worker %s task channel full, task dropped", workerID)
		return false
	}
}

// FindWorker 查找可用的 worker
func (s *TaskStreamServer) FindWorker(taskName string, targetKind, targetRef string) *WorkerConnection {
	s.workersMu.RLock()
	defer s.workersMu.RUnlock()

	var candidates []*WorkerConnection

	// 查找支持该任务的 worker
	for _, worker := range s.workers {
		for _, t := range worker.Tasks {
			if t == taskName {
				candidates = append(candidates, worker)
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 根据 target 过滤
	if targetKind == "node" && targetRef != "" {
		for _, w := range candidates {
			if w.NodeID == targetRef {
				return w
			}
		}
	} else if targetKind == "app" && targetRef != "" {
		for _, w := range candidates {
			if w.AppName == targetRef {
				return w
			}
		}
	}

	// 返回第一个可用的
	if len(candidates) > 0 {
		return candidates[0]
	}

	return nil
}

// addWorker 添加 worker
func (s *TaskStreamServer) addWorker(worker *WorkerConnection) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()
	s.workers[worker.WorkerID] = worker
}

// removeWorker 移除 worker
func (s *TaskStreamServer) removeWorker(workerID string) {
	s.workersMu.Lock()
	defer s.workersMu.Unlock()
	if worker, exists := s.workers[workerID]; exists {
		close(worker.TaskChan)
		delete(s.workers, workerID)
		log.Printf("[gRPC] Worker removed: %s", workerID)
	}
}

var globalServer *TaskStreamServer

// GetServer 获取全局 server 实例（供 scheduler 使用）
func GetServer() *TaskStreamServer {
	return globalServer
}

// StartServer 启动 gRPC 服务器
func StartServer(addr string, s store.Store) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    10 * time.Second,
			Timeout: 3 * time.Second,
		}),
	)

	taskServer := NewTaskStreamServer(s)
	globalServer = taskServer
	proto.RegisterTaskServiceServer(grpcServer, taskServer)

	go func() {
		log.Printf("[gRPC] Server listening on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[gRPC] Server failed: %v", err)
		}
	}()

	return grpcServer, nil
}
