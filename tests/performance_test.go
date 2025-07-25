package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pb "go-grpc-server-client/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkCreateUser(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createReq := &pb.CreateUserRequest{
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   int32(20 + (i % 50)),
		}

		resp, err := env.GRPCClient.CreateUser(ctx, createReq)
		require.NoError(b, err)
		assert.True(b, resp.Success)
	}
}

func BenchmarkGetUser(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Benchmark User",
		Email: "benchmark@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(b, err)
	userID := createResp.User.Id

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getReq := &pb.GetUserRequest{Id: userID}
		resp, err := env.GRPCClient.GetUser(ctx, getReq)
		require.NoError(b, err)
		assert.True(b, resp.Success)
	}
}

func BenchmarkListUsers(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	// Create some users first
	for i := 0; i < 100; i++ {
		createReq := &pb.CreateUserRequest{
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   int32(20 + (i % 50)),
		}
		resp, err := env.GRPCClient.CreateUser(ctx, createReq)
		require.NoError(b, err)
		assert.True(b, resp.Success)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		listReq := &pb.ListUsersRequest{Page: 1, Limit: 100}
		resp, err := env.GRPCClient.ListUsers(ctx, listReq)
		require.NoError(b, err)
		assert.True(b, resp.Success)
	}
}

func BenchmarkUpdateUser(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Update User",
		Email: "update@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(b, err)
	userID := createResp.User.Id

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updateReq := &pb.UpdateUserRequest{
			Id:    userID,
			Name:  fmt.Sprintf("Updated User %d", i),
			Email: fmt.Sprintf("updated%d@example.com", i),
			Age:   int32(30 + (i % 20)),
		}
		resp, err := env.GRPCClient.UpdateUser(ctx, updateReq)
		require.NoError(b, err)
		assert.True(b, resp.Success)
	}
}

func BenchmarkConcurrentCreateUsers(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pbTest *testing.PB) {
		i := 0
		for pbTest.Next() {
			createReq := &pb.CreateUserRequest{
				Name:  fmt.Sprintf("Concurrent User %d", i),
				Email: fmt.Sprintf("concurrent%d@example.com", i),
				Age:   int32(20 + (i % 50)),
			}

			resp, err := env.GRPCClient.CreateUser(ctx, createReq)
			require.NoError(b, err)
			assert.True(b, resp.Success)
			i++
		}
	})
}

func BenchmarkConcurrentUpdateUser(b *testing.B) {
	env := setupTestEnvironment(b)
	defer teardownTestEnvironment(b, env)

	ctx := context.Background()

	// Create a user first
	createReq := &pb.CreateUserRequest{
		Name:  "Concurrent Update User",
		Email: "concurrent-update@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(b, err)
	userID := createResp.User.Id

	b.ResetTimer()
	b.RunParallel(func(pbTest *testing.PB) {
		i := 0
		for pbTest.Next() {
			updateReq := &pb.UpdateUserRequest{
				Id:    userID,
				Name:  fmt.Sprintf("Concurrent Updated %d", i),
				Email: fmt.Sprintf("concurrent-updated%d@example.com", i),
				Age:   int32(30 + (i % 20)),
			}
			resp, err := env.GRPCClient.UpdateUser(ctx, updateReq)
			require.NoError(b, err)
			assert.True(b, resp.Success)
			i++
		}
	})
}

func TestPerformance_ConcurrentUserOperations(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Test concurrent operations on different users
	const numUsers = 10
	const operationsPerUser = 100

	// Create users
	userIDs := make([]int32, numUsers)
	for i := 0; i < numUsers; i++ {
		createReq := &pb.CreateUserRequest{
			Name:  fmt.Sprintf("Performance User %d", i),
			Email: fmt.Sprintf("perf%d@example.com", i),
			Age:   int32(20 + i),
		}
		resp, err := env.GRPCClient.CreateUser(ctx, createReq)
		require.NoError(t, err)
		userIDs[i] = resp.User.Id
	}

	// Measure concurrent operations
	start := time.Now()
	var wg sync.WaitGroup
	errors := make(chan error, numUsers*operationsPerUser)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()
			userID := userIDs[userIndex]

			for j := 0; j < operationsPerUser; j++ {
				// Update user
				updateReq := &pb.UpdateUserRequest{
					Id:    userID,
					Name:  fmt.Sprintf("User %d Update %d", userIndex, j),
					Email: fmt.Sprintf("user%d-update%d@example.com", userIndex, j),
					Age:   int32(20 + userIndex + j),
				}
				_, err := env.GRPCClient.UpdateUser(ctx, updateReq)
				if err != nil {
					errors <- err
					return
				}

				// Get user
				getReq := &pb.GetUserRequest{Id: userID}
				_, err = env.GRPCClient.GetUser(ctx, getReq)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalOperations := numUsers * operationsPerUser * 2 // update + get per operation

	// Check for errors
	for err := range errors {
		t.Errorf("Operation failed: %v", err)
	}

	t.Logf("Performance Test Results:")
	t.Logf("Total operations: %d", totalOperations)
	t.Logf("Duration: %v", duration)
	t.Logf("Operations per second: %.2f", float64(totalOperations)/duration.Seconds())
	t.Logf("Average operation time: %v", duration/time.Duration(totalOperations))
}

func TestPerformance_LoadTest(t *testing.T) {
	env := setupTestEnvironment(t)
	defer teardownTestEnvironment(t, env)

	ctx := context.Background()

	// Load test parameters
	const numClients = 50
	const requestsPerClient = 20
	const testDuration = 30 * time.Second

	// Create a user for the load test
	createReq := &pb.CreateUserRequest{
		Name:  "Load Test User",
		Email: "loadtest@example.com",
		Age:   30,
	}

	createResp, err := env.GRPCClient.CreateUser(ctx, createReq)
	require.NoError(t, err)
	userID := createResp.User.Id

	// Start load test
	start := time.Now()
	var wg sync.WaitGroup
	results := make(chan time.Duration, numClients*requestsPerClient)
	errors := make(chan error, numClients*requestsPerClient)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			for j := 0; j < requestsPerClient; j++ {
				// Simulate different types of operations
				opStart := time.Now()
				var err error

				switch j % 4 {
				case 0: // Get user
					getReq := &pb.GetUserRequest{Id: userID}
					_, err = env.GRPCClient.GetUser(ctx, getReq)
				case 1: // Update user
					updateReq := &pb.UpdateUserRequest{
						Id:    userID,
						Name:  fmt.Sprintf("Load Test User %d", clientID),
						Email: fmt.Sprintf("loadtest%d@example.com", clientID),
						Age:   int32(30 + clientID),
					}
					_, err = env.GRPCClient.UpdateUser(ctx, updateReq)
				case 2: // List users
					listReq := &pb.ListUsersRequest{Page: 1, Limit: 10}
					_, err = env.GRPCClient.ListUsers(ctx, listReq)
				case 3: // Create new user
					createReq := &pb.CreateUserRequest{
						Name:  fmt.Sprintf("Load User %d-%d", clientID, j),
						Email: fmt.Sprintf("load%d-%d@example.com", clientID, j),
						Age:   int32(20 + clientID + j),
					}
					_, err = env.GRPCClient.CreateUser(ctx, createReq)
				}

				opDuration := time.Since(opStart)
				results <- opDuration

				if err != nil {
					errors <- err
				}

				// Small delay to prevent overwhelming the server
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(start)
	totalRequests := numClients * requestsPerClient

	// Collect results
	var durations []time.Duration
	for d := range results {
		durations = append(durations, d)
	}

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Request error: %v", err)
	}

	// Calculate statistics
	if len(durations) > 0 {
		var total time.Duration
		var min, max time.Duration = durations[0], durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := total / time.Duration(len(durations))
		successRate := float64(len(durations)) / float64(totalRequests) * 100

		t.Logf("Load Test Results:")
		t.Logf("Total requests: %d", totalRequests)
		t.Logf("Successful requests: %d", len(durations))
		t.Logf("Failed requests: %d", errorCount)
		t.Logf("Success rate: %.2f%%", successRate)
		t.Logf("Test duration: %v", duration)
		t.Logf("Average response time: %v", avg)
		t.Logf("Min response time: %v", min)
		t.Logf("Max response time: %v", max)
		t.Logf("Requests per second: %.2f", float64(len(durations))/duration.Seconds())

		// Assertions
		assert.Greater(t, successRate, 95.0, "Success rate should be above 95%")
		assert.Less(t, avg, 100*time.Millisecond, "Average response time should be under 100ms")
	}
}
