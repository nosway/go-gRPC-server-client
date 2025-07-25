package main

import (
	"flag"
	"fmt"
	"log"

	"go-grpc-server-client/internal/client"
)

func main() {
	serverAddr := flag.String("server", "localhost:50051", "The server address in the format of host:port")
	flag.Parse()

	// 클라이언트 생성
	userClient, err := client.NewUserClient(*serverAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer userClient.Close()

	fmt.Println("=== gRPC User Service Client ===")

	// 사용자 생성
	fmt.Println("\n1. Creating users...")
	user1, err := userClient.CreateUser("John Doe", "john@example.com", 30)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	}

	user2, err := userClient.CreateUser("Jane Smith", "jane@example.com", 25)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	}

	// 사용자 목록 조회
	fmt.Println("\n2. Listing all users...")
	users, err := userClient.ListUsers()
	if err != nil {
		log.Printf("Failed to list users: %v", err)
	} else {
		for _, user := range users {
			fmt.Printf("  - ID: %d, Name: %s, Email: %s, Age: %d\n",
				user.Id, user.Name, user.Email, user.Age)
		}
	}

	// 특정 사용자 조회
	if user1 != nil {
		fmt.Printf("\n3. Getting user with ID %d...\n", user1.Id)
		retrievedUser, err := userClient.GetUser(user1.Id)
		if err != nil {
			log.Printf("Failed to get user: %v", err)
		} else {
			fmt.Printf("  - ID: %d, Name: %s, Email: %s, Age: %d\n",
				retrievedUser.Id, retrievedUser.Name, retrievedUser.Email, retrievedUser.Age)
		}

		// 사용자 정보 업데이트
		fmt.Printf("\n4. Updating user with ID %d...\n", user1.Id)
		updatedUser, err := userClient.UpdateUser(user1.Id, "John Updated", "john.updated@example.com", 31)
		if err != nil {
			log.Printf("Failed to update user: %v", err)
		} else {
			fmt.Printf("  - Updated: ID: %d, Name: %s, Email: %s, Age: %d\n",
				updatedUser.Id, updatedUser.Name, updatedUser.Email, updatedUser.Age)
		}
	}

	// 사용자 삭제
	if user2 != nil {
		fmt.Printf("\n5. Deleting user with ID %d...\n", user2.Id)
		err := userClient.DeleteUser(user2.Id)
		if err != nil {
			log.Printf("Failed to delete user: %v", err)
		} else {
			fmt.Println("  - User deleted successfully")
		}
	}

	// 최종 사용자 목록 조회
	fmt.Println("\n6. Final user list...")
	finalUsers, err := userClient.ListUsers()
	if err != nil {
		log.Printf("Failed to list users: %v", err)
	} else {
		for _, user := range finalUsers {
			fmt.Printf("  - ID: %d, Name: %s, Email: %s, Age: %d\n",
				user.Id, user.Name, user.Email, user.Age)
		}
	}

	fmt.Println("\n=== Client demo completed ===")
}
