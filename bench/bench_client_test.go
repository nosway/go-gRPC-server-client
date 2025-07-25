package bench

import (
	"context"
	"fmt"
	"testing"

	pb "go-grpc-server-client/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcAddr = "localhost:50051"
)

func newGRPCClient(tb testing.TB) pb.UserServiceClient {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		tb.Fatalf("failed to connect to gRPC server: %v", err)
	}
	return pb.NewUserServiceClient(conn)
}

func BenchmarkCreateUser(b *testing.B) {
	client := newGRPCClient(b)
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, err := client.CreateUser(ctx, &pb.CreateUserRequest{
			Name:  fmt.Sprintf("Bench User %d", i),
			Email: fmt.Sprintf("benchuser%d@example.com", i),
			Age:   int32(20 + i%50),
		})
		if err != nil {
			b.Fatalf("CreateUser failed: %v", err)
		}
	}
}

func BenchmarkGetUser(b *testing.B) {
	client := newGRPCClient(b)
	ctx := context.Background()

	// 미리 사용자 생성
	resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Name:  "BenchGetUser",
		Email: "benchgetuser@example.com",
		Age:   30,
	})
	if err != nil {
		b.Fatalf("CreateUser failed: %v", err)
	}
	userID := resp.User.Id

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetUser(ctx, &pb.GetUserRequest{Id: userID})
		if err != nil {
			b.Fatalf("GetUser failed: %v", err)
		}
	}
}

func BenchmarkUpdateUser(b *testing.B) {
	client := newGRPCClient(b)
	ctx := context.Background()

	// 미리 사용자 생성
	resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Name:  "BenchUpdateUser",
		Email: "benchupdateuser@example.com",
		Age:   30,
	})
	if err != nil {
		b.Fatalf("CreateUser failed: %v", err)
	}
	userID := resp.User.Id

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.UpdateUser(ctx, &pb.UpdateUserRequest{
			Id:    userID,
			Name:  fmt.Sprintf("Updated User %d", i),
			Email: fmt.Sprintf("updated%d@example.com", i),
			Age:   int32(30 + i%50),
		})
		if err != nil {
			b.Fatalf("UpdateUser failed: %v", err)
		}
	}
}

func BenchmarkListUsers(b *testing.B) {
	client := newGRPCClient(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.ListUsers(ctx, &pb.ListUsersRequest{Page: 1, Limit: 100})
		if err != nil {
			b.Fatalf("ListUsers failed: %v", err)
		}
	}
}
