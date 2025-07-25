package client

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "go-grpc-server-client/proto"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var logger = logrus.New()

func init() {
	// Configure logrus for client
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		if level, err := logrus.ParseLevel(logLevel); err == nil {
			logger.SetLevel(level)
		}
	}

	// Default to info level if not set
	if logger.GetLevel() == logrus.PanicLevel {
		logger.SetLevel(logrus.InfoLevel)
	}
}

type UserClient struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserClient(serverAddr string) (*UserClient, error) {
	logger.WithField("server_addr", serverAddr).Info("Connecting to gRPC server")

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.WithError(err).WithField("server_addr", serverAddr).Error("Failed to connect to gRPC server")
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	client := pb.NewUserServiceClient(conn)
	logger.WithField("server_addr", serverAddr).Info("gRPC client connected successfully")

	return &UserClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *UserClient) Close() error {
	if c.conn != nil {
		logger.Info("Closing gRPC client connection")
		err := c.conn.Close()
		if err != nil {
			logger.WithError(err).Error("Error closing gRPC client connection")
		} else {
			logger.Info("gRPC client connection closed successfully")
		}
		return err
	}
	logger.Debug("gRPC client connection was already nil")
	return nil
}

func (c *UserClient) CreateUser(name, email string, age int32) (*pb.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.CreateUserRequest{
		Name:  name,
		Email: email,
		Age:   age,
	}

	resp, err := c.client.CreateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to create user: %s", resp.Message)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("server returned nil user despite success")
	}

	logger.WithFields(logrus.Fields{
		"id":    resp.User.Id,
		"name":  resp.User.Name,
		"email": resp.User.Email,
	}).Info("User created")
	return resp.User, nil
}

func (c *UserClient) GetUser(id int32) (*pb.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.GetUserRequest{Id: id}

	resp, err := c.client.GetUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to get user: %s", resp.Message)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("server returned nil user despite success")
	}

	logger.WithFields(logrus.Fields{
		"id":    resp.User.Id,
		"name":  resp.User.Name,
		"email": resp.User.Email,
	}).Info("User retrieved")
	return resp.User, nil
}

func (c *UserClient) ListUsers() ([]*pb.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.ListUsersRequest{
		Page:  1,
		Limit: 100,
	}

	resp, err := c.client.ListUsers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %v", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to list users: %s", resp.Message)
	}

	logger.WithField("total", resp.Total).Info("Users listed")
	return resp.Users, nil
}

func (c *UserClient) UpdateUser(id int32, name, email string, age int32) (*pb.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.UpdateUserRequest{
		Id:    id,
		Name:  name,
		Email: email,
		Age:   age,
	}

	resp, err := c.client.UpdateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to update user: %s", resp.Message)
	}

	logger.WithFields(logrus.Fields{
		"id":    resp.User.Id,
		"name":  resp.User.Name,
		"email": resp.User.Email,
	}).Info("User updated")
	return resp.User, nil
}

func (c *UserClient) DeleteUser(id int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.DeleteUserRequest{Id: id}

	resp, err := c.client.DeleteUser(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to delete user: %s", resp.Message)
	}

	logger.WithField("id", id).Info("User deleted")
	return nil
}
