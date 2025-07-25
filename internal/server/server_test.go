package server

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	pb "go-grpc-server-client/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDistributedLocker is a mock implementation of DistributedLocker
type MockDistributedLocker struct {
	mock.Mock
}

func (m *MockDistributedLocker) LockUser(ctx context.Context, userID int32) (UnlockFunc, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Convert the returned function to UnlockFunc
	unlockFunc := args.Get(0).(func())
	return UnlockFunc(unlockFunc), args.Error(1)
}

// Satisfy DistributedLocker interface
func (m *MockDistributedLocker) HealthCheck(ctx context.Context) error {
	return nil
}

// MockDB is a mock implementation of database operations
type MockDB struct {
	mock.Mock
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mockArgs := m.Called(ctx, query, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	mockArgs := m.Called(ctx, query, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	mockArgs := m.Called(ctx, query, args)
	if mockArgs.Get(0) == nil {
		return nil
	}
	return mockArgs.Get(0).(*sql.Row)
}

// MockResult is a mock implementation of sql.Result
type MockResult struct {
	mock.Mock
}

func (m *MockResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func TestUserServer_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.CreateUserRequest
		setup   func(*MockDistributedLocker, *MockDB)
		want    *pb.CreateUserResponse
		wantErr bool
	}{
		{
			name: "successful user creation",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB) {
				// CreateUser doesn't use locks, so we don't need to mock LockUser
				// Mock database insert
				result := &MockResult{}
				result.On("LastInsertId").Return(int64(1), nil)
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want: &pb.CreateUserResponse{
				Success: true,
				Message: "User created successfully",
			},
			wantErr: false,
		},
		{
			name: "database error during insert",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB) {
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database connection failed"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "failed to get last insert ID",
			req: &pb.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB) {
				result := &MockResult{}
				result.On("LastInsertId").Return(int64(0), fmt.Errorf("failed to get last insert ID"))
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locker := &MockDistributedLocker{}
			db := &MockDB{}

			if tt.setup != nil {
				tt.setup(locker, db)
			}

			server := NewUserServerWithDB(db, locker)

			ctx := context.Background()
			got, err := server.CreateUser(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.True(t, got.Success)
			assert.Equal(t, tt.want.Message, got.Message)
			assert.NotNil(t, got.User)
			assert.Equal(t, tt.req.Name, got.User.Name)
			assert.Equal(t, tt.req.Email, got.User.Email)
			assert.Equal(t, tt.req.Age, got.User.Age)

			locker.AssertExpectations(t)
			db.AssertExpectations(t)
		})
	}
}

func TestUserServer_DeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.DeleteUserRequest
		setup   func(*MockDistributedLocker, *MockDB, *MockResult)
		want    *pb.DeleteUserResponse
		wantErr bool
	}{
		{
			name: "user deleted successfully",
			req:  &pb.DeleteUserRequest{Id: 1},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(1)).Return(func() {}, nil)
				result.On("RowsAffected").Return(int64(1), nil)
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want: &pb.DeleteUserResponse{
				Success: true,
				Message: "User deleted successfully",
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req:  &pb.DeleteUserRequest{Id: 999},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(999)).Return(func() {}, nil)
				result.On("RowsAffected").Return(int64(0), nil)
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want: &pb.DeleteUserResponse{
				Success: false,
				Message: "User not found",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locker := &MockDistributedLocker{}
			db := &MockDB{}
			result := &MockResult{}

			if tt.setup != nil {
				tt.setup(locker, db, result)
			}

			server := NewUserServerWithDB(db, locker)

			ctx := context.Background()
			got, err := server.DeleteUser(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Success, got.Success)
			assert.Equal(t, tt.want.Message, got.Message)

			locker.AssertExpectations(t)
			db.AssertExpectations(t)
			result.AssertExpectations(t)
		})
	}
}

func TestRedsyncLocker_LockUser(t *testing.T) {
	// This test requires a real Redis instance
	// In a real scenario, you'd use testcontainers or a mock
	t.Skip("Requires Redis instance")
}

func TestEtcdLocker_LockUser(t *testing.T) {
	// This test requires a real etcd instance
	// In a real scenario, you'd use testcontainers or a mock
	t.Skip("Requires etcd instance")
}

func TestUserServer_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.GetUserRequest
		setup   func(*MockDistributedLocker, *MockDB)
		want    *pb.GetUserResponse
		wantErr bool
	}{
		{
			name: "lock acquisition failed",
			req:  &pb.GetUserRequest{Id: 1},
			setup: func(locker *MockDistributedLocker, db *MockDB) {
				locker.On("LockUser", mock.Anything, int32(1)).Return(nil, fmt.Errorf("lock acquisition failed"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locker := &MockDistributedLocker{}
			db := &MockDB{}

			if tt.setup != nil {
				tt.setup(locker, db)
			}

			server := NewUserServerWithDB(db, locker)

			ctx := context.Background()
			got, err := server.GetUser(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Success, got.Success)
			assert.Equal(t, tt.want.Message, got.Message)

			locker.AssertExpectations(t)
			db.AssertExpectations(t)
		})
	}
}

func TestUserServer_ListUsers(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.ListUsersRequest
		setup   func(*MockDistributedLocker, *MockDB)
		want    *pb.ListUsersResponse
		wantErr bool
	}{
		{
			name: "database error during query",
			req:  &pb.ListUsersRequest{Page: 1, Limit: 10},
			setup: func(locker *MockDistributedLocker, db *MockDB) {
				db.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database connection failed"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locker := &MockDistributedLocker{}
			db := &MockDB{}

			if tt.setup != nil {
				tt.setup(locker, db)
			}

			server := NewUserServerWithDB(db, locker)

			ctx := context.Background()
			got, err := server.ListUsers(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Success, got.Success)
			assert.Equal(t, tt.want.Message, got.Message)
			assert.Equal(t, tt.want.Total, got.Total)

			locker.AssertExpectations(t)
			db.AssertExpectations(t)
		})
	}
}

func TestUserServer_UpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.UpdateUserRequest
		setup   func(*MockDistributedLocker, *MockDB, *MockResult)
		want    *pb.UpdateUserResponse
		wantErr bool
	}{
		{
			name: "lock acquisition failed",
			req: &pb.UpdateUserRequest{
				Id:    1,
				Name:  "John Updated",
				Email: "john.updated@example.com",
				Age:   31,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(1)).Return(nil, fmt.Errorf("lock acquisition failed"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "database error during update",
			req: &pb.UpdateUserRequest{
				Id:    1,
				Name:  "John Updated",
				Email: "john.updated@example.com",
				Age:   31,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(1)).Return(func() {}, nil)
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "user not found for update",
			req: &pb.UpdateUserRequest{
				Id:    999,
				Name:  "John Updated",
				Email: "john.updated@example.com",
				Age:   31,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(999)).Return(func() {}, nil)
				result.On("RowsAffected").Return(int64(0), nil)
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want: &pb.UpdateUserResponse{
				Success: false,
				Message: "User not found",
			},
			wantErr: false,
		},
		{
			name: "failed to get rows affected",
			req: &pb.UpdateUserRequest{
				Id:    1,
				Name:  "John Updated",
				Email: "john.updated@example.com",
				Age:   31,
			},
			setup: func(locker *MockDistributedLocker, db *MockDB, result *MockResult) {
				locker.On("LockUser", mock.Anything, int32(1)).Return(func() {}, nil)
				result.On("RowsAffected").Return(int64(0), fmt.Errorf("failed to get rows affected"))
				db.On("ExecContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(result, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locker := &MockDistributedLocker{}
			db := &MockDB{}
			result := &MockResult{}

			if tt.setup != nil {
				tt.setup(locker, db, result)
			}

			server := NewUserServerWithDB(db, locker)

			ctx := context.Background()
			got, err := server.UpdateUser(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want.Success, got.Success)
			assert.Equal(t, tt.want.Message, got.Message)

			locker.AssertExpectations(t)
			db.AssertExpectations(t)
			result.AssertExpectations(t)
		})
	}
}
