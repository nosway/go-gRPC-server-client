# Go gRPC Server-Client 프로젝트

이 프로젝트는 Go 언어를 사용하여 구현된 gRPC 서버와 클라이언트입니다. 사용자 정보를 관리하는 CRUD API를 제공하며, MySQL 데이터베이스와 Redis/etcd를 사용한 분산 락을 지원합니다.

## 🚀 주요 기능

- **사용자 관리 API**: 생성, 조회, 목록, 수정, 삭제 기능
- **MySQL 데이터베이스**: 영구 저장소
- **분산 락**: Redis(Redsync) 또는 etcd 선택적 사용
- **동시성 제어**: User ID별 분산 락으로 멀티 인스턴스 환경에서도 안전한 동시성 보장
- **구조화된 로깅**: JSON 형식의 상세한 로깅 시스템 (logrus)
- **포괄적인 테스트**: 단위 테스트, 통합 테스트, 성능 테스트 포함
- **모니터링**: Prometheus 메트릭 수집 및 Grafana 대시보드
- **헬스체크**: HTTP 엔드포인트를 통한 상태 확인
- **Docker 지원**: 완전한 컨테이너화된 개발 환경

## 📁 프로젝트 구조

```
go-grpc-server-client/
├── proto/                    # Protocol Buffers 정의
│   ├── service.proto        # gRPC 서비스 정의
│   ├── service.pb.go        # 생성된 Go 코드
│   └── service_grpc.pb.go   # 생성된 gRPC Go 코드
├── internal/                # 내부 패키지
│   ├── server/             # gRPC 서버 구현
│   │   ├── server.go       # MySQL + Redis/etcd 분산 락
│   │   └── server_test.go  # 서버 단위 테스트
│   └── client/             # gRPC 클라이언트 구현
│       ├── client.go
│       └── client_test.go  # 클라이언트 단위 테스트
├── cmd/                    # 실행 파일
│   ├── server/            # 서버 메인
│   │   └── main.go
│   └── client/            # 클라이언트 메인
│       └── main.go
├── tests/                 # 테스트 파일
│   ├── integration_test.go # 통합 테스트
│   └── performance_test.go # 성능 테스트
├── grafana/               # Grafana 설정
│   ├── dashboards/        # 대시보드 설정
│   └── datasources/       # 데이터소스 설정
├── bin/                   # 빌드된 실행 파일 (자동 생성)
├── go.mod                 # Go 모듈 정의
├── Makefile              # 빌드 및 실행 스크립트
├── docker-compose.yml    # Docker 환경 설정
├── Dockerfile            # 애플리케이션 컨테이너화
├── prometheus.yml        # Prometheus 설정
├── README.md             # 프로젝트 문서
├── .gitignore            # Git 제외 파일
└── cursor.md             # 개발 히스토리
```

## 🛠️ 설치 및 설정

### 1. 필수 요구사항

- Go 1.21+
- Docker Desktop (권장)
- Protocol Buffers 컴파일러 (protoc)

### 2. 프로젝트 설정

```bash
# 저장소 클론
git clone <repository-url>
cd go-grpc-server-client

# Protocol Buffers 컴파일러 설치
make install-protoc

# 의존성 설치
go mod tidy
```

## 🐳 Docker 환경에서 실행 (권장)

### 1. 전체 환경 시작

```bash
# 모든 서비스 시작 (MySQL, Redis, etcd, Prometheus, Grafana)
make start-full
```

이 명령어는 다음 서비스들을 시작합니다:
- **MySQL**: 데이터베이스 (포트 3306)
- **Redis**: 분산 락 (포트 6379)
- **etcd**: 분산 락 대안 (포트 2379)
- **Prometheus**: 메트릭 수집 (포트 9090)
- **Grafana**: 모니터링 대시보드 (포트 3000)

### 2. 서버 실행

```bash
# 로컬에서 서버 실행 (Docker 환경과 연결)
make run-server-local
```

### 3. 클라이언트 실행

```bash
# 클라이언트 실행
make run-client-local
```

### 4. 모니터링 확인

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **gRPC 서버 헬스체크**: http://localhost:2112/healthz

### 5. 환경 관리 명령어

```bash
# 환경 상태 확인
make docker-status

# 서비스 로그 확인
make docker-logs

# 헬스체크
make docker-health

# 환경 중지
make docker-stop

# 환경 정리 (볼륨 포함)
make docker-clean
```

## 🔧 로컬 환경에서 실행

### 1. 환경 변수 설정

```bash
# MySQL 연결 정보
export MYSQL_DSN="user:password@tcp(localhost:3306)/dbname"

# 분산 락 타입 선택 (redis 또는 etcd)
export LOCK_TYPE=redis

# Redis 설정 (LOCK_TYPE=redis인 경우)
export REDIS_ADDR=localhost:6379

# etcd 설정 (LOCK_TYPE=etcd인 경우)
export ETCD_ENDPOINTS=localhost:2379

# 로깅 레벨 설정 (선택사항)
export LOG_LEVEL=info  # debug, info, warn, error, fatal, panic

# 외부 리소스 헬스체크 (선택사항)
export HEALTHCHECK_EXTERNAL=on  # off (기본값)
```

### 2. 서버 실행

```bash
# 서버 빌드 및 실행
make run-server
```

### 3. 클라이언트 실행

```bash
# 새 터미널에서 클라이언트 실행
make run-client
```

## 🧪 테스트

### Docker 환경에서 테스트

```bash
# 통합 테스트 실행
make docker-test

# 성능 테스트 실행
make docker-benchmark

# 모든 테스트 실행
make test
```

### 로컬 환경에서 테스트

```bash
# 단위 테스트만 실행
make test-unit

# 테스트 커버리지 확인
make coverage
```

### 테스트 결과 예시

```bash
# 단위 테스트 실행
$ make test-unit
=== RUN   TestUserServer_CreateUser
--- PASS: TestUserServer_CreateUser (0.00s)
=== RUN   TestUserServer_GetUser
--- PASS: TestUserServer_GetUser (0.00s)
...
PASS
ok      go-grpc-server-client/internal/server   0.123s

# 테스트 커버리지 확인
$ make coverage
ok      go-grpc-server-client/internal/client   0.435s  coverage: 74.4% of statements
ok      go-grpc-server-client/internal/server   0.658s  coverage: 30.6% of statements
total:                                                  (statements)            43.2%
```

## 📊 모니터링 및 메트릭

### Prometheus 메트릭

서버는 다음 메트릭을 자동으로 수집합니다:

- **gRPC 요청 카운터**: `grpc_server_handled_total`
- **gRPC 처리 시간**: `grpc_server_handling_seconds`
- **gRPC 에러 카운터**: `grpc_server_handled_total{grpc_code!="OK"}`
- **Go 런타임 메트릭**: 메모리, CPU, 고루틴 등

### Grafana 대시보드

프로젝트에 포함된 대시보드:
- **gRPC 요청 총량**: 각 API 메서드별 요청 수
- **gRPC 요청 지속시간**: API 응답 시간
- **gRPC 에러**: 에러 발생 현황
- **Go 런타임 메트릭**: 메모리 사용량 등

### 메트릭 확인

```bash
# Prometheus 메트릭 직접 확인
curl http://localhost:2112/metrics

# 헬스체크
curl http://localhost:2112/healthz
```

## 🩺 헬스체크

서버는 `/healthz` 엔드포인트에서 헬스체크를 제공합니다:

```bash
# 기본 헬스체크 (DB만 확인)
curl http://localhost:2112/healthz

# 외부 리소스까지 확인
env HEALTHCHECK_EXTERNAL=on curl http://localhost:2112/healthz
```

응답 예시:
- **정상**: `200 OK` + "ok"
- **DB 오류**: `500 Internal Server Error` + "db error: ..."
- **외부 리소스 오류**: `500 Internal Server Error` + "external error: ..."

## 🔧 추가 테스트 도구

### gRPCurl을 사용한 테스트

```bash
# gRPCurl 설치
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# 서버가 실행 중일 때
# 사용자 목록 조회
grpcurl -plaintext localhost:50051 service.UserService/ListUsers

# 사용자 생성
grpcurl -plaintext -d '{"name": "Test User", "email": "test@example.com", "age": 25}' localhost:50051 service.UserService/CreateUser

# 특정 사용자 조회
grpcurl -plaintext -d '{"id": 1}' localhost:50051 service.UserService/GetUser
```

## 📊 성능 지표

테스트 환경에서의 예상 성능:

- **단일 사용자 생성**: ~1ms
- **단일 사용자 조회**: ~0.5ms
- **사용자 목록 조회 (100명)**: ~5ms
- **동시 요청 처리**: 1000+ req/s
- **분산 락 응답 시간**: ~2ms

## 🚀 빠른 시작 가이드

### 1. Docker 환경에서 전체 테스트

```bash
# 1. 전체 환경 시작
make start-full

# 2. 서버 실행 (새 터미널)
make run-server-local

# 3. 클라이언트 실행 (새 터미널)
make run-client-local

# 4. 모니터링 확인
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)

# 5. 통합 테스트 실행
make docker-test

# 6. 성능 테스트 실행
make docker-benchmark
```

### 2. 로컬 환경에서 빠른 테스트

```bash
# 1. 단위 테스트 실행
make test-unit

# 2. 서버 실행
make run-server

# 3. 클라이언트 실행 (새 터미널)
make run-client
```

## 🐛 문제 해결

### 일반적인 문제들

1. **Docker 서비스 시작 실패**
   ```bash
   # Docker Desktop 상태 확인
   docker --version
   docker-compose --version
   
   # 서비스 재시작
   make docker-stop
   make docker-clean
   make start-full
   ```

2. **포트 충돌**
   ```bash
   # 사용 중인 포트 확인
   lsof -i :50051
   lsof -i :3306
   lsof -i :6379
   
   # 충돌하는 서비스 중지
   brew services stop mysql
   brew services stop redis
   ```

3. **메트릭 접근 불가**
   ```bash
   # 서버 헬스체크
   curl http://localhost:2112/healthz
   
   # Prometheus 메트릭
   curl http://localhost:2112/metrics
   ```

4. **테스트 실패**
   ```bash
   # Docker 환경 상태 확인
   make docker-status
   
   # 서비스 재시작
   make docker-stop
   make start-full
   sleep 15
   make docker-test
   ```

## 🤝 기여하기

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다.

## 📞 문의

프로젝트에 대한 문의사항이 있으시면 이슈를 생성해 주세요. 

## 🖥️ 서버 환경 실행/중단 및 테스트 안내

### 서버/테스트 환경 실행

- **전체 환경 실행 (모든 서비스, 모니터링 포함)**
  ```sh
  make start-full
  # 또는
  docker-compose up -d
  ```

- **서버만 로컬에서 실행 (컨테이너 DB/Redis와 연동)**
  ```sh
  make run-server-local
  ```

- **클라이언트 실행**
  ```sh
  make run-client-local
  ```

### 서버/테스트 환경 중단 및 정리

- **컨테이너만 중단 (데이터는 유지)**
  ```sh
  make docker-stop
  # 또는
  docker-compose down
  ```

- **컨테이너 + 네트워크 + 볼륨(데이터)까지 완전 정리**
  ```sh
  make docker-clean
  # 또는
  docker-compose down -v
  docker system prune -f
  ```

### 테스트 및 벤치마크

- **통합 테스트 (컨테이너 환경에서)**
  ```sh
  make docker-test
  ```

- **성능 벤치마크 (컨테이너 환경에서)**
  ```sh
  make docker-benchmark
  ```

- **실제 gRPC 연산 벤치마크 (서버가 미리 실행 중이어야 함)**
  ```sh
  make bench-client
  # 또는
  go test -bench=. ./bench
  ```

- **단위 테스트 (로컬)**
  ```sh
  make test-unit
  ```

- **테스트 커버리지 확인**
  ```sh
  make coverage
  ```

---

> 컨테이너 환경을 완전히 정리하고 싶을 때는 `make docker-clean`을 사용하세요. 데이터까지 모두 삭제됩니다.
> 서버/DB/Redis가 미리 실행된 상태에서만 bench-client 벤치마크가 의미 있습니다. 