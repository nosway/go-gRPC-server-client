package client

import (
	"context"
	"fmt"
	"testing"

	pb "go-grpc-server-client/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockUserServiceClient is a mock implementation of pb.UserServiceClient
type MockUserServiceClient struct {
	mock.Mock
}

func (m *MockUserServiceClient) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetUserResponse), args.Error(1)
}

func (m *MockUserServiceClient) ListUsers(ctx context.Context, in *pb.ListUsersRequest, opts ...grpc.CallOption) (*pb.ListUsersResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ListUsersResponse), args.Error(1)
}

func (m *MockUserServiceClient) CreateUser(ctx context.Context, in *pb.CreateUserRequest, opts ...grpc.CallOption) (*pb.CreateUserResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.CreateUserResponse), args.Error(1)
}

func (m *MockUserServiceClient) UpdateUser(ctx context.Context, in *pb.UpdateUserRequest, opts ...grpc.CallOption) (*pb.UpdateUserResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.UpdateUserResponse), args.Error(1)
}

func (m *MockUserServiceClient) DeleteUser(ctx context.Context, in *pb.DeleteUserRequest, opts ...grpc.CallOption) (*pb.DeleteUserResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.DeleteUserResponse), args.Error(1)
}

func TestUserClient_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.CreateUserRequest
		setup   func(*MockUserServiceClient)
		want    *pb.User
		wantErr bool
	}{
		{
			name: "successful user creation",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(mockClient *MockUserServiceClient) {
				expectedUser := &pb.User{
					Id:        1,
					Name:      "John Doe",
					Email:     "john@example.com",
					Age:       30,
					CreatedAt: "2023-01-01T00:00:00Z",
					UpdatedAt: "2023-01-01T00:00:00Z",
				}
				response := &pb.CreateUserResponse{
					User:    expectedUser,
					Success: true,
					Message: "User created successfully",
				}
				mockClient.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want: &pb.User{
				Id:        1,
				Name:      "John Doe",
				Email:     "john@example.com",
				Age:       30,
				CreatedAt: "2023-01-01T00:00:00Z",
				UpdatedAt: "2023-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "server error",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.CreateUserResponse{
					Success: false,
					Message: "Database error",
				}
				mockClient.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			got, err := client.CreateUser(tt.req.Name, tt.req.Email, tt.req.Age)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Id, got.Id)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Age, got.Age)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_CreateUser_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.CreateUserRequest
		setup   func(*MockUserServiceClient)
		wantErr bool
	}{
		{
			name: "server returns error",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(mockClient *MockUserServiceClient) {
				mockClient.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("server error"))
			},
			wantErr: true,
		},
		{
			name: "server returns failure response",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.CreateUserResponse{
					Success: false,
					Message: "Email already exists",
				}
				mockClient.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			wantErr: true,
		},
		{
			name: "server returns nil user",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.CreateUserResponse{
					Success: true,
					Message: "User created successfully",
					User:    nil,
				}
				mockClient.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			_, err := client.CreateUser(tt.req.Name, tt.req.Email, tt.req.Age)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		userID  int32
		setup   func(*MockUserServiceClient)
		want    *pb.User
		wantErr bool
	}{
		{
			name:   "user found",
			userID: 1,
			setup: func(mockClient *MockUserServiceClient) {
				expectedUser := &pb.User{
					Id:        1,
					Name:      "John Doe",
					Email:     "john@example.com",
					Age:       30,
					CreatedAt: "2023-01-01T00:00:00Z",
					UpdatedAt: "2023-01-01T00:00:00Z",
				}
				response := &pb.GetUserResponse{
					User:    expectedUser,
					Success: true,
					Message: "User found successfully",
				}
				mockClient.On("GetUser", mock.Anything, &pb.GetUserRequest{Id: 1}, mock.Anything).Return(response, nil)
			},
			want: &pb.User{
				Id:        1,
				Name:      "John Doe",
				Email:     "john@example.com",
				Age:       30,
				CreatedAt: "2023-01-01T00:00:00Z",
				UpdatedAt: "2023-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: 999,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.GetUserResponse{
					Success: false,
					Message: "User not found",
				}
				mockClient.On("GetUser", mock.Anything, &pb.GetUserRequest{Id: 999}, mock.Anything).Return(response, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			got, err := client.GetUser(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Id, got.Id)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Age, got.Age)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_GetUser_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		userID  int32
		setup   func(*MockUserServiceClient)
		wantErr bool
	}{
		{
			name:   "server returns error",
			userID: 1,
			setup: func(mockClient *MockUserServiceClient) {
				mockClient.On("GetUser", mock.Anything, &pb.GetUserRequest{Id: 1}, mock.Anything).Return(nil, fmt.Errorf("server error"))
			},
			wantErr: true,
		},
		{
			name:   "server returns failure response",
			userID: 999,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.GetUserResponse{
					Success: false,
					Message: "User not found",
				}
				mockClient.On("GetUser", mock.Anything, &pb.GetUserRequest{Id: 999}, mock.Anything).Return(response, nil)
			},
			wantErr: true,
		},
		{
			name:   "server returns nil user",
			userID: 1,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.GetUserResponse{
					Success: true,
					Message: "User found successfully",
					User:    nil,
				}
				mockClient.On("GetUser", mock.Anything, &pb.GetUserRequest{Id: 1}, mock.Anything).Return(response, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			_, err := client.GetUser(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_ListUsers(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockUserServiceClient)
		want    []*pb.User
		wantErr bool
	}{
		{
			name: "list users successfully",
			setup: func(mockClient *MockUserServiceClient) {
				users := []*pb.User{
					{Id: 1, Name: "John Doe", Email: "john@example.com", Age: 30},
					{Id: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 25},
				}
				response := &pb.ListUsersResponse{
					Users:   users,
					Total:   2,
					Success: true,
					Message: "Users retrieved successfully",
				}
				mockClient.On("ListUsers", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want: []*pb.User{
				{Id: 1, Name: "John Doe", Email: "john@example.com", Age: 30},
				{Id: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 25},
			},
			wantErr: false,
		},
		{
			name: "server error",
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.ListUsersResponse{
					Success: false,
					Message: "Database error",
				}
				mockClient.On("ListUsers", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			got, err := client.ListUsers()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Len(t, got, len(tt.want))
			for i, user := range got {
				assert.Equal(t, tt.want[i].Id, user.Id)
				assert.Equal(t, tt.want[i].Name, user.Name)
				assert.Equal(t, tt.want[i].Email, user.Email)
				assert.Equal(t, tt.want[i].Age, user.Age)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_UpdateUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   int32
		userName string
		email    string
		age      int32
		setup    func(*MockUserServiceClient)
		want     *pb.User
		wantErr  bool
	}{
		{
			name:     "user updated successfully",
			userID:   1,
			userName: "John Updated",
			email:    "john.updated@example.com",
			age:      31,
			setup: func(mockClient *MockUserServiceClient) {
				expectedUser := &pb.User{
					Id:        1,
					Name:      "John Updated",
					Email:     "john.updated@example.com",
					Age:       31,
					CreatedAt: "2023-01-01T00:00:00Z",
					UpdatedAt: "2023-01-01T00:00:00Z",
				}
				response := &pb.UpdateUserResponse{
					User:    expectedUser,
					Success: true,
					Message: "User updated successfully",
				}
				mockClient.On("UpdateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want: &pb.User{
				Id:        1,
				Name:      "John Updated",
				Email:     "john.updated@example.com",
				Age:       31,
				CreatedAt: "2023-01-01T00:00:00Z",
				UpdatedAt: "2023-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name:     "user not found",
			userID:   999,
			userName: "John Updated",
			email:    "john.updated@example.com",
			age:      31,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.UpdateUserResponse{
					Success: false,
					Message: "User not found",
				}
				mockClient.On("UpdateUser", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			got, err := client.UpdateUser(tt.userID, tt.userName, tt.email, tt.age)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Id, got.Id)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Age, got.Age)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_DeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		userID  int32
		setup   func(*MockUserServiceClient)
		wantErr bool
	}{
		{
			name:   "user deleted successfully",
			userID: 1,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.DeleteUserResponse{
					Success: true,
					Message: "User deleted successfully",
				}
				mockClient.On("DeleteUser", mock.Anything, &pb.DeleteUserRequest{Id: 1}, mock.Anything).Return(response, nil)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: 999,
			setup: func(mockClient *MockUserServiceClient) {
				response := &pb.DeleteUserResponse{
					Success: false,
					Message: "User not found",
				}
				mockClient.On("DeleteUser", mock.Anything, &pb.DeleteUserRequest{Id: 999}, mock.Anything).Return(response, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockUserServiceClient{}
			if tt.setup != nil {
				tt.setup(mockClient)
			}

			client := &UserClient{
				client: mockClient,
			}

			err := client.DeleteUser(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUserClient_Close(t *testing.T) {
	// Create a client with a mock connection
	mockClient := &MockUserServiceClient{}
	client := &UserClient{
		client: mockClient,
		conn:   nil, // Set conn to nil to avoid panic
	}

	// This should not panic even with nil conn
	err := client.Close()
	assert.NoError(t, err)
}
