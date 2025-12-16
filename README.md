# Webhook Processor

A robust, scalable webhook processing system built with Go, implementing Clean Architecture principles. The system handles internal events (+credit/postback, -debit/chargeback) and transforms them into external webhook calls with comprehensive retry mechanisms and distributed worker processing.

## Features

### Core Functionality

- **Event Processing**: Handles +credit (postback) and -debit (chargeback) events
- **Distributed Workers**: 10 configurable workers with proper locking mechanism
- **Retry Logic**: 6 retries with exponential backoff and random jitter
- **URL Construction**: Dynamic GET request URL building with parameters
- **Database Tracking**: Comprehensive retry attempt logging in PostgreSQL

### Architecture

- **Clean Architecture**: Proper separation of concerns with domain, application, and infrastructure layers
- **GORM ORM**: Type-safe database operations with PostgreSQL
- **Distributed Processing**: Multiple pod instances with worker coordination
- **Graceful Shutdown**: Proper cleanup and resource management

### Monitoring & Observability

- **Structured Logging**: JSON/logfmt logging with configurable levels
- **Statistics API**: Real-time processing statistics
- **Health Checks**: Application health monitoring
- **Performance Metrics**: Retry attempt tracking and duration logging

## Quick Start

### Using Docker Compose (Recommended)

1. **Clone and setup**:

   ```bash
   git clone <repository-url>
   cd webhook-processor
   cp .env.example .env
   ```

2. **Start the system**:

   ```bash
   make dev-setup
   ```

3. **Create sample webhooks**:

   ```bash
   make create-sample-webhooks
   ```

4. **Check statistics**:
   ```bash
   make get-stats
   ```

### Manual Setup

1. **Install dependencies**:

   ```bash
   make deps
   ```

2. **Setup PostgreSQL** and run migrations:

   ```bash
   psql -U postgres -d webhook_processor -f scripts/init.sql
   ```

3. **Build and run**:
   ```bash
   make build
   ./bin/webhook-processor  # In one terminal
   ./bin/webhook-api       # In another terminal
   ```

## Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

### Key Configuration Options

| Variable               | Default | Description                              |
| ---------------------- | ------- | ---------------------------------------- |
| `WORKER_COUNT`         | 10      | Number of concurrent workers             |
| `WORKER_BATCH_SIZE`    | 50      | Webhooks processed per batch             |
| `WORKER_POLL_INTERVAL` | 5s      | How often workers check for new webhooks |
| `WORKER_LOCK_DURATION` | 5m      | How long a worker holds a lock           |
| `HTTP_TIMEOUT`         | 30s     | Webhook request timeout                  |
| `LOG_LEVEL`            | info    | Logging level (debug, info, warn, error) |

## API Usage

### Create Webhook Entry

```bash
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "+credit",
    "event_id": "tx_123",
    "config_id": 1,
    "url_params": {
      "value": "100",
      "user_id": "user_123",
      "token": "abc123",
      "signature": "xyz789"
    }
  }'
```

### Get Statistics

```bash
curl -X GET http://localhost:8080/webhooks/stats
```

### Health Check

```bash
curl -X GET http://localhost:8080/health
```

## Database Schema

### Webhook Queue Table

The system uses a comprehensive table structure to track all retry attempts:

```sql
CREATE TABLE webhook_queue (
    id BIGSERIAL PRIMARY KEY,
    queue_id UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,

    -- Event information
    event_type VARCHAR(50) NOT NULL, -- '+credit' or '-debit'
    event_id VARCHAR(255),

    -- Webhook details
    config_id BIGINT NOT NULL REFERENCES webhook_configs(id),
    webhook_url TEXT NOT NULL,
    url_params JSONB NOT NULL DEFAULT '{}',

    -- Processing status
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'

    -- Retry tracking
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 6,
    next_retry_at TIMESTAMP DEFAULT NOW(),

    -- Individual retry attempt tracking (retry_0 through retry_6)
    retry_0_started_at TIMESTAMP,
    retry_0_completed_at TIMESTAMP,
    retry_0_duration_ms BIGINT,
    retry_0_http_status INTEGER,
    retry_0_response_body TEXT,
    retry_0_error TEXT,
    -- ... (similar for retry_1 through retry_6)

    -- Worker coordination
    worker_id VARCHAR(100),
    locked_at TIMESTAMP,
    lock_expires_at TIMESTAMP,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

## Retry Mechanism

The system implements a sophisticated retry mechanism:

1. **Exponential Backoff**: Base delay of 1 second, doubling with each retry
2. **Jitter**: ±25% random variation to prevent thundering herd
3. **Maximum Delay**: Capped at 5 minutes
4. **Comprehensive Tracking**: Each attempt is logged with timing, status, and response data

### Retry Schedule Example

| Attempt     | Base Delay | With Jitter Range | Max Delay |
| ----------- | ---------- | ----------------- | --------- |
| 0 (initial) | Immediate  | -                 | -         |
| 1           | 1s         | 0.75s - 1.25s     | 1.25s     |
| 2           | 2s         | 1.5s - 2.5s       | 2.5s      |
| 3           | 4s         | 3s - 5s           | 5s        |
| 4           | 8s         | 6s - 10s          | 10s       |
| 5           | 16s        | 12s - 20s         | 20s       |
| 6           | 32s        | 24s - 40s         | 40s       |

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Webhook API   │    │ Worker Pool     │    │   PostgreSQL    │
│   (HTTP Server) │    │ (10 Workers)    │    │   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ Creates webhook       │ Processes webhooks    │ Stores state
         │ entries              │ with locking          │ and attempts
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ External APIs   │
                    │ (HTTP GET)      │
                    └─────────────────┘
```

### Clean Architecture Layers

1. **Domain Layer** (`internal/domain/`):

   - Entities: Core business objects
   - Repositories: Data access interfaces
   - Services: Business logic interfaces

2. **Application Layer** (`internal/application/`):

   - Use Cases: Business logic implementation
   - Workers: Processing coordination

3. **Infrastructure Layer** (`internal/infrastructure/`):
   - Database: GORM models and repositories
   - HTTP: External service communication
   - Configuration: Environment management

## Development

### Available Make Targets

```bash
make help                    # Show all available targets
make build                   # Build binaries
make test                    # Run tests
make test-coverage          # Run tests with coverage
make docker-up              # Start with Docker Compose
make docker-down            # Stop Docker Compose
make create-sample-webhooks # Create test webhooks
make get-stats              # Get processing statistics
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# View coverage report
open coverage.html
```

### Monitoring

The system provides comprehensive monitoring through:

1. **Structured Logs**: All operations are logged with context
2. **Statistics API**: Real-time processing metrics
3. **Health Checks**: Application and dependency status
4. **Database Metrics**: Retry attempts, success rates, processing times

## Deployment

### Docker Deployment

```bash
# Build and deploy
make docker-build
make docker-up

# Scale workers
docker-compose up --scale webhook-processor=3
```

### Kubernetes Deployment

The system is designed for Kubernetes deployment with:

- Multiple pod instances
- Distributed worker coordination
- Database connection pooling
- Graceful shutdown handling

## Performance Considerations

1. **Database Indexes**: Optimized for webhook status and retry time queries
2. **Connection Pooling**: Configurable database connection limits
3. **Worker Coordination**: Efficient locking mechanism prevents duplicate processing
4. **Batch Processing**: Workers process multiple webhooks per cycle
5. **Timeout Management**: Configurable timeouts prevent hanging requests

## Security

1. **Non-root Container**: Docker containers run as non-root user
2. **Environment Variables**: Sensitive configuration via environment
3. **Database Security**: SSL support and connection limits
4. **Input Validation**: Request validation and sanitization

## Troubleshooting

### Common Issues

1. **Workers not processing**: Check database connectivity and worker configuration
2. **High retry rates**: Verify external webhook endpoints are accessible
3. **Database locks**: Monitor lock expiration and cleanup intervals
4. **Memory usage**: Adjust batch sizes and worker counts based on load

### Debugging

```bash
# View logs
make docker-logs

# Check worker statistics
make get-stats

# Monitor database
docker-compose exec postgres psql -U postgres -d webhook_processor -c "SELECT status, COUNT(*) FROM webhook_queue GROUP BY status;"
```

## Contributing

1. Follow Clean Architecture principles
2. Add tests for new functionality
3. Update documentation
4. Use conventional commit messages

## License

[Add your license information here]
