package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"go-grpc-server-client/internal/client"
	"go-grpc-server-client/internal/server"
	pb "go-grpc-server-client/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TestEnvironment struct {
	MySQLContainer testcontainers.Container
	RedisContainer testcontainers.Container
	MySQLDSN       string
	RedisAddr      string
	GRPCServer     *grpc.Server
	GRPCClient     pb.UserServiceClient
	Client         *client.UserClient
	ServerPort     int
}

func setupTestEnvironment(t testing.TB) *TestEnvironment {
	ctx := context.Background()

	// Start MySQL container
	mysqlContainer, err := mysql.RunContainer(ctx,
		testcontainers.WithImage("mysql:8.0"),
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("MySQL init process done. Ready for start up."),
		),
	)
	require.NoError(t, err)

	// Start Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections"),
		),
	)
	require.NoError(t, err)

	// Get container endpoints
	mysqlHost, err := mysqlContainer.Host(ctx)
	require.NoError(t, err)
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306")
	require.NoError(t, err)

	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	mysqlDSN := fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb", mysqlHost, mysqlPort.Port())
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// Wait for MySQL to be ready and test connection
	require.Eventually(t, func() bool {
		db, err := sql.Open("mysql", mysqlDSN)
		if err != nil {
			return false
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			return false
		}
		return true
	}, 30*time.Second, 1*time.Second, "MySQL should be ready")

	// Set environment variables
	os.Setenv("MYSQL_DSN", mysqlDSN)
	os.Setenv("LOCK_TYPE", "redis")
	os.Setenv("REDIS_ADDR", redisAddr)

	// Start gRPC server
	serverPort := 50051
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	userServer := server.NewUserServer(mysqlDSN, "redis", redisAddr, "")
	pb.RegisterUserServiceServer(grpcServer, userServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Create gRPC client
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	grpcClient := pb.NewUserServiceClient(conn)
	userClient, err := client.NewUserClient(fmt.Sprintf("localhost:%d", serverPort))
	require.NoError(t, err)

	return &TestEnvironment{
		MySQLContainer: mysqlContainer,
		RedisContainer: redisContainer,
		MySQLDSN:       mysqlDSN,
		RedisAddr:      redisAddr,
		GRPCServer:     grpcServer,
		GRPCClient:     grpcClient,
		Client:         userClient,
		ServerPort:     serverPort,
	}
}

func teardownTestEnvironment(t testing.TB, env *TestEnvironment) {
	if env.Client != nil {
		env.Client.Close()
	}
	if env.GRPCServer != nil {
		env.GRPCServer.Stop()
	}
	if env.MySQLContainer != nil {
		env.MySQLContainer.Terminate(context.Background())
	}
	if env.RedisContainer != nil {
		env.RedisContainer.Terminate(context.Background())
	}
}

func TestIntegration_CreateAndGetUser(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Test CreateUser
	createReq := &pb.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(t, err)
	assert.True(t, createResp.Success)
	assert.NotNil(t, createResp.User)
	assert.Equal(t, "John Doe", createResp.User.Name)
	assert.Equal(t, "john@example.com", createResp.User.Email)
	assert.Equal(t, int32(30), createResp.User.Age)

	userID := createResp.User.Id

	// Test GetUser
	getReq := &pb.GetUserRequest{Id: userID}
	getResp, err := env.GRPCClient.GetUser(ctx, getReq)
	require.NoError(t, err)
	assert.True(t, getResp.Success)
	assert.NotNil(t, getResp.User)
	assert.Equal(t, userID, getResp.User.Id)
	assert.Equal(t, "John Doe", getResp.User.Name)
}

func TestIntegration_UpdateUser(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Jane Smith",
		Email: "jane@example.com",
		Age:   25,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(t, err)
	assert.True(t, createResp.Success)

	userID := createResp.User.Id

	// Test UpdateUser
	updateReq := &pb.UpdateUserRequest{
		Id:    userID,
		Name:  "Jane Updated",
		Email: "jane.updated@example.com",
		Age:   26,
	}

	updateResp, err := env.GRPCClient.UpdateUser(ctx, updateReq)
	require.NoError(t, err)
	assert.True(t, updateResp.Success)
	assert.NotNil(t, updateResp.User)
	assert.Equal(t, userID, updateResp.User.Id)
	assert.Equal(t, "Jane Updated", updateResp.User.Name)
	assert.Equal(t, "jane.updated@example.com", updateResp.User.Email)
	assert.Equal(t, int32(26), updateResp.User.Age)
}

func TestIntegration_DeleteUser(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Bob Johnson",
		Email: "bob@example.com",
		Age:   35,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(t, err)
	assert.True(t, createResp.Success)

	userID := createResp.User.Id

	// Test DeleteUser
	deleteReq := &pb.DeleteUserRequest{Id: userID}
	deleteResp, err := env.GRPCClient.DeleteUser(ctx, deleteReq)
	require.NoError(t, err)
	assert.True(t, deleteResp.Success)

	// Verify user is deleted
	getReq := &pb.GetUserRequest{Id: userID}
	getResp, err := env.GRPCClient.GetUser(ctx, getReq)
	require.NoError(t, err)
	assert.False(t, getResp.Success)
	assert.Equal(t, "User not found", getResp.Message)
}

func TestIntegration_ListUsers(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Create multiple users
	users := []*pb.CreateUserRequest{
		{Name: "Alice", Email: "alice@example.com", Age: 28},
		{Name: "Bob", Email: "bob@example.com", Age: 32},
		{Name: "Charlie", Email: "charlie@example.com", Age: 29},
	}

	for _, userReq := range users {
		resp, err := env.GRPCClient.CreateUser(ctx, userReq)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	}

	// Test ListUsers
	listReq := &pb.ListUsersRequest{Page: 1, Limit: 100}
	listResp, err := env.GRPCClient.ListUsers(ctx, listReq)
	require.NoError(t, err)
	assert.True(t, listResp.Success)
	assert.Len(t, listResp.Users, 3)
	assert.Equal(t, int32(3), listResp.Total)

	// Verify all users are present
	userNames := make(map[string]bool)
	for _, user := range listResp.Users {
		userNames[user.Name] = true
	}

	assert.True(t, userNames["Alice"])
	assert.True(t, userNames["Bob"])
	assert.True(t, userNames["Charlie"])
}

func TestIntegration_ConcurrentUserOperations(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Concurrent User",
		Email: "concurrent@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(t, err)
	assert.True(t, createResp.Success)

	userID := createResp.User.Id

	// Test concurrent updates
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			updateReq := &pb.UpdateUserRequest{
				Id:    userID,
				Name:  fmt.Sprintf("User %d", id),
				Email: fmt.Sprintf("user%d@example.com", id),
				Age:   int32(30 + id),
			}

			_, err := env.GRPCClient.UpdateUser(ctx, updateReq)
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify the final state
	getReq := &pb.GetUserRequest{Id: userID}
	getResp, err := env.GRPCClient.GetUser(ctx, getReq)
	require.NoError(t, err)
	assert.True(t, getResp.Success)
	assert.Equal(t, userID, getResp.User.Id)
}

func TestIntegration_ClientWrapper(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	// Test using the client wrapper
	user, err := env.Client.CreateUser("Test User", "test@example.com", 25)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, int32(25), user.Age)

	// Test GetUser
	retrievedUser, err := env.Client.GetUser(user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, retrievedUser.Id)
	assert.Equal(t, user.Name, retrievedUser.Name)

	// Test UpdateUser
	updatedUser, err := env.Client.UpdateUser(user.Id, "Updated User", "updated@example.com", 26)
	require.NoError(t, err)
	assert.Equal(t, "Updated User", updatedUser.Name)
	assert.Equal(t, "updated@example.com", updatedUser.Email)
	assert.Equal(t, int32(26), updatedUser.Age)

	// Test DeleteUser
	err = env.Client.DeleteUser(user.Id)
	require.NoError(t, err)

	// Verify deletion
	_, err = env.Client.GetUser(user.Id)
	assert.Error(t, err)
}
