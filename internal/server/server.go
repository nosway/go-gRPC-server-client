package server

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	pb "go-grpc-server-client/proto"

	redis "github.com/go-redis/redis/v8"
	redsync "github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis/goredis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	clientv3 "go.etcd.io/etcd/client/v3"
	concurrency "go.etcd.io/etcd/client/v3/concurrency"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	logger              = logrus.New()
	mainDB              *sql.DB           // for health check
	checkExternalHealth bool              // for health check option
	globalLocker        DistributedLocker // for health check
)

func init() {
	// Configure logrus
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

	// Health check external option
	if v := os.Getenv("HEALTHCHECK_EXTERNAL"); strings.ToLower(v) == "on" {
		checkExternalHealth = true
	}
}

// maskDSN masks sensitive information in DSN string for logging
func maskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}

	// Simple masking: replace password with asterisks
	// Format: user:password@tcp(host:port)/dbname
	if strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		if len(parts) == 2 {
			userPass := parts[0]
			if strings.Contains(userPass, ":") {
				userParts := strings.Split(userPass, ":")
				if len(userParts) >= 2 {
					return userParts[0] + ":****@" + parts[1]
				}
			}
		}
	}

	return dsn
}

// DBInterface defines the interface for database operations
type DBInterface interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// DistributedLocker interface
// HealthCheck returns error if the external lock system is unhealthy
// (optional: not all implementations must support)
type DistributedLocker interface {
	LockUser(ctx context.Context, userID int32) (UnlockFunc, error)
	HealthCheck(ctx context.Context) error
}

// Exported for testing
// UnlockFunc is a function type for releasing locks
type UnlockFunc func()

// Redis(Redsync) 구현체
type RedsyncLocker struct {
	rsync *redsync.Redsync
	rdb   *redis.Client // for health check
}

func NewRedsyncLocker(redisAddr string) *RedsyncLocker {
	logger.WithField("redis_addr", redisAddr).Info("Initializing Redis locker")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	pool := redsyncredis.NewPool(rdb)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.WithError(err).WithField("redis_addr", redisAddr).Fatal("Failed to connect to Redis")
	}

	logger.WithField("redis_addr", redisAddr).Info("Redis locker initialized successfully")
	return &RedsyncLocker{rsync: redsync.New(pool), rdb: rdb}
}

func (l *RedsyncLocker) LockUser(ctx context.Context, userID int32) (UnlockFunc, error) {
	lockKey := fmt.Sprintf("user-lock-%d", userID)
	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"lock_key": lockKey,
	}).Debug("Attempting to acquire Redis lock")

	mutex := l.rsync.NewMutex(lockKey)
	if err := mutex.LockContext(ctx); err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"lock_key": lockKey,
		}).Error("Failed to acquire Redis lock")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"lock_key": lockKey,
	}).Debug("Redis lock acquired successfully")

	return func() {
		mutex.Unlock()
		logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"lock_key": lockKey,
		}).Debug("Redis lock released")
	}, nil
}

// RedsyncLocker implements HealthCheck
func (l *RedsyncLocker) HealthCheck(ctx context.Context) error {
	if l == nil || l.rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return l.rdb.Ping(ctx).Err()
}

// etcd 구현체
type EtcdLocker struct {
	client *clientv3.Client
}

func NewEtcdLocker(endpoints []string) *EtcdLocker {
	logger.WithField("etcd_endpoints", endpoints).Info("Initializing etcd locker")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.WithError(err).WithField("etcd_endpoints", endpoints).Fatal("Failed to connect to etcd")
	}

	logger.WithField("etcd_endpoints", endpoints).Info("etcd locker initialized successfully")
	return &EtcdLocker{client: cli}
}

func (l *EtcdLocker) LockUser(ctx context.Context, userID int32) (UnlockFunc, error) {
	lockKey := fmt.Sprintf("/user-lock-%d", userID)
	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"lock_key": lockKey,
	}).Debug("Attempting to acquire etcd lock")

	sess, err := concurrency.NewSession(l.client, concurrency.WithContext(ctx))
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"lock_key": lockKey,
		}).Error("Failed to create etcd session")
		return nil, err
	}

	mutex := concurrency.NewMutex(sess, lockKey)
	if err := mutex.Lock(ctx); err != nil {
		sess.Close()
		logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"lock_key": lockKey,
		}).Error("Failed to acquire etcd lock")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"lock_key": lockKey,
	}).Debug("etcd lock acquired successfully")

	return func() {
		mutex.Unlock(ctx)
		sess.Close()
		logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"lock_key": lockKey,
		}).Debug("etcd lock released")
	}, nil
}

// EtcdLocker implements HealthCheck
func (l *EtcdLocker) HealthCheck(ctx context.Context) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("etcd client not initialized")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := l.client.Get(ctx, "healthcheck-key")
	return err
}

type UserServer struct {
	pb.UnimplementedUserServiceServer
	db     DBInterface
	locker DistributedLocker
}

func NewUserServer(mysqlDSN, lockType, redisAddr, etcdEndpoints string) *UserServer {
	logger.WithField("lock_type", lockType).Info("Initializing UserServer")

	// MySQL 연결
	logger.WithField("mysql_dsn", maskDSN(mysqlDSN)).Info("Connecting to MySQL database")
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		logger.WithError(err).WithField("mysql_dsn", maskDSN(mysqlDSN)).Fatal("Failed to open MySQL connection")
	}

	mainDB = db // for health check

	if err := db.Ping(); err != nil {
		logger.WithError(err).WithField("mysql_dsn", maskDSN(mysqlDSN)).Fatal("Failed to ping MySQL database")
	}

	logger.Info("MySQL connection established successfully")

	if err := initDB(db); err != nil {
		logger.WithError(err).Fatal("Failed to initialize database schema")
	}

	logger.Info("Database schema initialized successfully")

	// 분산 락 구현체 선택
	var locker DistributedLocker
	switch strings.ToLower(lockType) {
	case "etcd":
		if etcdEndpoints == "" {
			logger.Fatal("ETCD_ENDPOINTS must be set for etcd lock type")
		}
		endpoints := strings.Split(etcdEndpoints, ",")
		locker = NewEtcdLocker(endpoints)
	case "redis":
		if redisAddr == "" {
			logger.Fatal("REDIS_ADDR must be set for redis lock type")
		}
		locker = NewRedsyncLocker(redisAddr)
	default:
		logger.WithField("lock_type", lockType).Fatal("Unknown LOCK_TYPE (must be 'redis' or 'etcd')")
	}

	globalLocker = locker // for health check

	logger.Info("UserServer initialized successfully")
	return &UserServer{
		db:     db,
		locker: locker,
	}
}

// Exported for testing
// UnlockFunc is a function type for releasing locks
// NewUserServerWithDB is a test constructor
func NewUserServerWithDB(db DBInterface, locker DistributedLocker) *UserServer {
	return &UserServer{
		db:     db,
		locker: locker,
	}
}

func initDB(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		age INT NOT NULL,
		created_at VARCHAR(64) NOT NULL,
		updated_at VARCHAR(64) NOT NULL
	);`
	_, err := db.Exec(query)
	return err
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	logger.WithField("user_id", req.Id).Info("GetUser request received")

	unlock, err := s.locker.LockUser(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to acquire lock for GetUser")
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, age, created_at, updated_at FROM users WHERE id = ?`, req.Id)
	var user pb.User
	err = row.Scan(&user.Id, &user.Name, &user.Email, &user.Age, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		logger.WithField("user_id", req.Id).Warn("User not found")
		return &pb.GetUserResponse{Success: false, Message: "User not found"}, nil
	} else if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Database error in GetUser")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"user_id":    req.Id,
		"user_name":  user.Name,
		"user_email": user.Email,
	}).Info("User retrieved successfully")

	return &pb.GetUserResponse{User: &user, Success: true, Message: "User found successfully"}, nil
}

func (s *UserServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	logger.WithFields(logrus.Fields{
		"page":  req.Page,
		"limit": req.Limit,
	}).Info("ListUsers request received")

	rows, err := s.db.QueryContext(ctx, `SELECT id, name, email, age, created_at, updated_at FROM users`)
	if err != nil {
		logger.WithError(err).Error("Database error in ListUsers")
		return nil, err
	}
	defer rows.Close()

	var users []*pb.User
	for rows.Next() {
		var user pb.User
		err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Age, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			logger.WithError(err).Error("Error scanning user row in ListUsers")
			return nil, err
		}
		users = append(users, &user)
	}

	logger.WithField("total_users", len(users)).Info("Users listed successfully")

	return &pb.ListUsersResponse{
		Users:   users,
		Total:   int32(len(users)),
		Success: true,
		Message: "Users retrieved successfully",
	}, nil
}

func (s *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	logger.WithFields(logrus.Fields{
		"user_name":  req.Name,
		"user_email": req.Email,
		"user_age":   req.Age,
	}).Info("CreateUser request received")

	now := time.Now().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (name, email, age, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, req.Name, req.Email, req.Age, now, now)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"user_name":  req.Name,
			"user_email": req.Email,
		}).Error("Database error in CreateUser")
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		logger.WithError(err).Error("Failed to get last insert ID in CreateUser")
		return nil, err
	}

	user := &pb.User{
		Id:        int32(id),
		Name:      req.Name,
		Email:     req.Email,
		Age:       req.Age,
		CreatedAt: now,
		UpdatedAt: now,
	}

	logger.WithFields(logrus.Fields{
		"user_id":    user.Id,
		"user_name":  user.Name,
		"user_email": user.Email,
	}).Info("User created successfully")

	return &pb.CreateUserResponse{User: user, Success: true, Message: "User created successfully"}, nil
}

func (s *UserServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	logger.WithFields(logrus.Fields{
		"user_id":    req.Id,
		"user_name":  req.Name,
		"user_email": req.Email,
		"user_age":   req.Age,
	}).Info("UpdateUser request received")

	unlock, err := s.locker.LockUser(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to acquire lock for UpdateUser")
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	now := time.Now().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `UPDATE users SET name=?, email=?, age=?, updated_at=? WHERE id=?`, req.Name, req.Email, req.Age, now, req.Id)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Database error in UpdateUser")
		return nil, err
	}

	num, err := res.RowsAffected()
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to get rows affected in UpdateUser")
		return nil, err
	}

	if num == 0 {
		logger.WithField("user_id", req.Id).Warn("User not found for update")
		return &pb.UpdateUserResponse{Success: false, Message: "User not found"}, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, age, created_at, updated_at FROM users WHERE id = ?`, req.Id)
	var user pb.User
	err = row.Scan(&user.Id, &user.Name, &user.Email, &user.Age, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to retrieve updated user")
		return nil, err
	}

	logger.WithFields(logrus.Fields{
		"user_id":    user.Id,
		"user_name":  user.Name,
		"user_email": user.Email,
	}).Info("User updated successfully")

	return &pb.UpdateUserResponse{User: &user, Success: true, Message: "User updated successfully"}, nil
}

func (s *UserServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	logger.WithField("user_id", req.Id).Info("DeleteUser request received")

	unlock, err := s.locker.LockUser(ctx, req.Id)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to acquire lock for DeleteUser")
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, req.Id)
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Database error in DeleteUser")
		return nil, err
	}

	num, err := res.RowsAffected()
	if err != nil {
		logger.WithError(err).WithField("user_id", req.Id).Error("Failed to get rows affected in DeleteUser")
		return nil, err
	}

	if num == 0 {
		logger.WithField("user_id", req.Id).Warn("User not found for deletion")
		return &pb.DeleteUserResponse{Success: false, Message: "User not found"}, nil
	}

	logger.WithField("user_id", req.Id).Info("User deleted successfully")
	return &pb.DeleteUserResponse{Success: true, Message: "User deleted successfully"}, nil
}

func RunServer(port int) error {
	logger.WithField("port", port).Info("Starting gRPC server")

	mysqlDSN := os.Getenv("MYSQL_DSN") // 예: "user:password@tcp(localhost:3306)/dbname"
	lockType := os.Getenv("LOCK_TYPE") // "redis" or "etcd"
	redisAddr := os.Getenv("REDIS_ADDR")
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS") // comma-separated

	logger.WithFields(logrus.Fields{
		"mysql_dsn":      maskDSN(mysqlDSN),
		"lock_type":      lockType,
		"redis_addr":     redisAddr,
		"etcd_endpoints": etcdEndpoints,
	}).Info("Server configuration loaded")

	if mysqlDSN == "" || lockType == "" {
		logger.Fatal("MYSQL_DSN and LOCK_TYPE environment variables must be set")
	}

	// Prometheus metrics & healthz HTTP endpoint
	go func() {
		logger.WithField("metrics_port", 2112).Info("Starting Prometheus metrics endpoint at /metrics and health check at /healthz")
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if mainDB != nil {
				if err := mainDB.Ping(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("db error: " + err.Error()))
					return
				}
			}
			if checkExternalHealth {
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				defer cancel()
				// Assuming globalLocker is defined elsewhere or needs to be passed
				// For now, we'll check if the locker is initialized and healthy
				if globalLocker != nil { // Assuming globalLocker is the DistributedLocker
					if err := globalLocker.HealthCheck(ctx); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte("external error: " + err.Error()))
						return
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		http.ListenAndServe(":2112", nil)
	}()

	// gRPC Prometheus interceptors
	grpcMetrics := grpc_prometheus.NewServerMetrics()
	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
	grpcMetrics.InitializeMetrics(s)

	pb.RegisterUserServiceServer(s, NewUserServer(mysqlDSN, lockType, redisAddr, etcdEndpoints))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.WithError(err).WithField("port", port).Error("Failed to listen on port")
		return fmt.Errorf("failed to listen: %v", err)
	}

	logger.WithField("port", port).Info("gRPC server listening")
	return s.Serve(lis)
}
