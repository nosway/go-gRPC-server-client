.PHONY: proto build run-server run-client clean install-protoc test test-unit test-integration test-performance benchmark coverage docker-build docker-run docker-stop docker-logs docker-clean docker-test docker-benchmark docker-monitoring

# Protocol Buffers 컴파일
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/service.proto

# 빌드
build: proto
	mkdir -p bin
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go

# 서버 실행
run-server: build
	./bin/server

# 클라이언트 실행
run-client: build
	./bin/client

# 정리
clean:
	rm -rf bin/
	rm -f proto/*.pb.go

# Protocol Buffers 설치
install-protoc:
	# macOS
	brew install protobuf
	# Go 플러그인 설치
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 모든 테스트 실행
test: test-unit test-integration test-performance

# 단위 테스트 실행
test-unit:
	go test -v ./internal/...

# 통합 테스트 실행 (Docker 필요)
test-integration:
	go test -v ./tests/... -run "TestIntegration"

# 성능 테스트 실행 (Docker 필요)
test-performance:
	go test -v ./tests/... -run "TestPerformance"

# 벤치마크 실행 (Docker 필요)
benchmark:
	go test -bench=. -benchmem ./tests/...

# 테스트 커버리지
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 테스트 커버리지 (상세)
coverage-verbose:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 테스트 실행 (짧은 모드)
test-short:
	go test -v -short ./...

# 테스트 실행 (병렬)
test-parallel:
	go test -v -parallel 4 ./...

# 테스트 실행 (타임아웃 설정)
test-timeout:
	go test -v -timeout 5m ./tests/...

# 테스트 실행 (메모리 프로파일링)
test-memprofile:
	go test -v -memprofile=mem.prof ./tests/...
	go tool pprof -top mem.prof

# 테스트 실행 (CPU 프로파일링)
test-cpuprofile:
	go test -v -cpuprofile=cpu.prof ./tests/...
	go tool pprof -top cpu.prof

# Docker 관련 명령어들

# Docker 이미지 빌드
docker-build:
	docker build -t grpc-server-client .

# Docker Compose로 전체 환경 실행
docker-run:
	docker-compose up -d

# Docker Compose로 환경 중지
docker-stop:
	docker-compose down

# Docker 로그 확인
docker-logs:
	docker-compose logs -f

# Docker 환경 정리 (볼륨 포함)
docker-clean:
	docker-compose down -v
	docker system prune -f

# Docker 환경에서 통합 테스트 실행
docker-test:
	docker-compose up -d mysql redis etcd
	sleep 10
	go test -v ./tests/... -run "TestIntegration"
	docker-compose down

# Docker 환경에서 성능 테스트 실행
docker-benchmark:
	docker-compose up -d mysql redis etcd
	sleep 10
	go test -bench=. -benchmem ./tests/...
	docker-compose down

# 모니터링 환경만 실행 (Prometheus + Grafana)
docker-monitoring:
	docker-compose up -d prometheus grafana
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000 (admin/admin)"

# 로컬 서버 실행 (Docker 환경과 연결)
run-server-local:
	MYSQL_DSN="testuser:testpass@tcp(localhost:3306)/testdb" \
	LOCK_TYPE=redis \
	REDIS_ADDR=localhost:6379 \
	ETCD_ENDPOINTS=localhost:2379 \
	LOG_LEVEL=info \
	HEALTHCHECK_EXTERNAL=on \
	./bin/server

# 로컬 클라이언트 실행
run-client-local:
	./bin/client

# 환경 상태 확인
docker-status:
	docker-compose ps

# 헬스체크
docker-health:
	@echo "Checking MySQL health..."
	@curl -s http://localhost:3306 || echo "MySQL not accessible"
	@echo "Checking Redis health..."
	@curl -s http://localhost:6379 || echo "Redis not accessible"
	@echo "Checking gRPC server health..."
	@curl -s http://localhost:2112/healthz || echo "gRPC server not accessible"
	@echo "Checking Prometheus health..."
	@curl -s http://localhost:9090/-/healthy || echo "Prometheus not accessible"
	@echo "Checking Grafana health..."
	@curl -s http://localhost:3000/api/health || echo "Grafana not accessible"

# 전체 환경 시작 (모니터링 포함)
start-full:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	sleep 15
	@echo "Environment is ready!"
	@echo "gRPC Server: localhost:50051"
	@echo "Health Check: http://localhost:2112/healthz"
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo ""
	@echo "To run the server locally: make run-server-local"
	@echo "To run the client: make run-client-local" 