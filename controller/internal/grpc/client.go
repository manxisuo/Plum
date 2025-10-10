package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/manxisuo/plum/controller/proto"
)

type TaskClient struct {
	conn   *grpc.ClientConn
	client proto.TaskServiceClient
}

func NewTaskClient(grpcAddress string) (*TaskClient, error) {
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := proto.NewTaskServiceClient(conn)
	return &TaskClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *TaskClient) ExecuteTask(ctx context.Context, taskID, name, payload string) (string, error) {
	req := &proto.TaskRequest{
		TaskId:  taskID,
		Name:    name,
		Payload: payload,
	}

	// Set timeout for task execution
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.ExecuteTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gRPC call failed: %v", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("task execution error: %s", resp.Error)
	}

	return resp.Result, nil
}

func (c *TaskClient) HealthCheck(ctx context.Context, workerID string) error {
	req := &proto.HealthRequest{
		WorkerId: workerID,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}

	if !resp.Healthy {
		return fmt.Errorf("worker unhealthy: %s", resp.Message)
	}

	return nil
}

func (c *TaskClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
