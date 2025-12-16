# Level 0 Workers Implementation

## Overview

This system implements **3 dedicated workers** for processing level 0 (immediate) webhooks with atomic locking to prevent race conditions.

## Architecture

### 🔄 **Worker Configuration**

- **3 Level 0 Workers**: All polling for `retry_count = 0` webhooks
- **Poll Interval**: 2 seconds (aggressive polling for immediate processing)
- **Atomic Locking**: PostgreSQL `SELECT FOR UPDATE SKIP LOCKED`

### 🛡️ **Race Condition Prevention**

The system prevents multiple workers from picking the same webhook using:

```sql
SELECT * FROM webhook_queue
WHERE status = 'PENDING'
  AND retry_count = 0
  AND next_retry_at <= NOW()
FOR UPDATE SKIP LOCKED
ORDER BY next_retry_at ASC
LIMIT 1;
```

**Key Features:**

- `FOR UPDATE`: Locks the selected row
- `SKIP LOCKED`: If a row is already locked, skip it and try the next one
- **Atomic**: The entire select-and-lock operation is atomic

### 📊 **Worker Pool Structure**

| Worker            | Retry Level | Poll Interval | Description                 |
| ----------------- | ----------- | ------------- | --------------------------- |
| Level 0 Worker #1 | 0           | 2s            | Immediate webhook attempts  |
| Level 0 Worker #2 | 0           | 2s            | Immediate webhook attempts  |
| Level 0 Worker #3 | 0           | 2s            | Immediate webhook attempts  |
| Level 1 Worker    | 1           | 30s           | First retry (1 min delay)   |
| Level 2 Worker    | 2           | 2m            | Second retry (5 min delay)  |
| Level 3 Worker    | 3           | 5m            | Third retry (10 min delay)  |
| Level 4 Worker    | 4           | 15m           | Fourth retry (30 min delay) |
| Level 5 Worker    | 5           | 30m           | Fifth retry (1 hour delay)  |
| Level 6 Worker    | 6           | 60m           | Final retry (2 hour delay)  |

## 🚀 **How It Works**

### 1. **Webhook Creation**

```go
webhook := &entities.WebhookQueue{
    RetryCount: 0,           // Starts at level 0
    Status: "PENDING",       // Ready for processing
    NextRetryAt: time.Now(), // Process immediately
}
```

### 2. **Worker Competition**

All 3 level-0 workers simultaneously poll every 2 seconds:

```go
// Worker 1, 2, and 3 all execute this concurrently
webhook, err := processor.GetNextWebhookForProcessing(ctx, workerID, 0)
```

### 3. **Atomic Selection**

Only **ONE** worker gets the webhook:

- Worker 1: Gets the webhook ✅
- Worker 2: Gets `nil` (no work available) ❌
- Worker 3: Gets `nil` (no work available) ❌

### 4. **Processing Flow**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Worker #1     │    │   Worker #2     │    │   Worker #3     │
│   (Level 0)     │    │   (Level 0)     │    │   (Level 0)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
    Poll for Level 0        Poll for Level 0        Poll for Level 0
    webhooks every 2s       webhooks every 2s       webhooks every 2s
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│              PostgreSQL Database                                │
│   SELECT ... FOR UPDATE SKIP LOCKED (Atomic Operation)         │
└─────────────────────────────────────────────────────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
    Gets webhook            Gets null               Gets null
    (LOCKED)               (already locked)        (already locked)
         │
         ▼
    Process webhook
    (Success/Retry/Fail)
```

## 🎯 **Benefits**

### **High Throughput**

- 3 workers = 3x processing capacity for immediate webhooks
- 2-second polling = near real-time processing

### **No Race Conditions**

- PostgreSQL's atomic locking prevents duplicate processing
- `SKIP LOCKED` ensures workers don't wait for each other

### **Efficient Resource Usage**

- Workers only process webhooks at their designated retry level
- No wasted cycles on inappropriate retry levels

### **Scalable Design**

- Easy to add more level-0 workers by updating configuration
- Each retry level has optimized polling intervals

## 🔧 **Configuration**

To modify the number of level-0 workers, update `internal/config/config.go`:

```go
func GetDefaultWorkerPoolConfig() WorkerPoolConfig {
    return WorkerPoolConfig{
        Workers: []WorkerConfig{
            // Add more level-0 workers here
            {
                RetryLevel:   0,
                PollInterval: 2 * time.Second,
                Description:  "Level 0 Worker #4 - Immediate webhook attempts",
            },
            // ... existing workers
        },
    }
}
```

## 📈 **Monitoring**

Each worker logs its activity:

```
level=info msg="worker started" retry_level=0 poll_interval=2s description="Level 0 Worker #1"
level=debug msg="processing webhook" worker_id=retry-0-abc123 retry_level=0 queue_id=uuid
level=info msg="webhook completed successfully" worker_id=retry-0-abc123 queue_id=uuid
```

## 🚨 **Error Handling**

If a worker fails to process a webhook:

1. Webhook is reset to `PENDING` status
2. Retry count is incremented
3. Next retry time is calculated based on new retry level
4. Higher-level workers will pick it up later

This ensures **no webhook is lost** even if processing fails.
