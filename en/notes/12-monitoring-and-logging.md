# Monitoring and Logging

## One-Line Summary

controller-runtime has logging (zap), metrics (Prometheus), and health checks built in,
so developers can use them right away without any additional setup.

---

## 1. Logging

### Logging Library: zap

controller-runtime uses Go's `logr` interface,
with the **zap** library as the default implementation.

The configuration in `cmd/main.go`:

```go
opts := zap.Options{
    Development: true,   // Development mode: human-readable format
}
ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
```

| Mode                 | Output Format                  | Use Case                                |
| -------------------- | ------------------------------ | --------------------------------------- |
| `Development: true`  | Human-readable text            | Local development                       |
| `Development: false` | JSON (structured logs)         | Production (suitable for ELK, Loki, etc.) |

### Using the Logger

How to get the logger in a Controller:

```go
log := logf.FromContext(ctx)   // Get logger from context
```

When retrieved via `FromContext`, **CR information (name, namespace) is automatically included**.
You can tell which resource a log entry relates to without adding it manually.

### Log Levels

```go
// Info -- Record normal operations
log.Info("Creating Deployment", "name", app.Name)

// Error -- Record errors (used directly for permanent errors)
log.Error(err, "External API call failed", "url", apiURL)

// Debug -- Set to V(1) or higher (not printed by default)
log.V(1).Info("Detailed debug info", "key", value)
```

| Level | Function          | When to Use                                   |
| ----- | ----------------- | --------------------------------------------- |
| Info  | `log.Info()`      | Key operations (create, update, delete)        |
| Error | `log.Error()`     | Permanent errors (when retry is pointless)     |
| Debug | `log.V(1).Info()` | Detailed debugging (not shown in production)   |

### Log Usage in Our Code

```go
log.Info("SimpleApp resource deleted, ignoring")                             // line 59
log.Info("Finalizer cleanup executing", "name", app.Name)                    // line 71
log.Info("Creating Deployment", "name", app.Name)                            // line 96
log.Info("Updating Deployment", "name", app.Name, "image", ..., "replicas", ...) // line 130
```

All use `log.Info()` only.
When an error is returned via `return ctrl.Result{}, err`, controller-runtime automatically logs it:

```
ERROR  Reconciler error  {"controller": "simpleapp", "name": "myapp", "error": "..."}
```

### Structured Logging

controller-runtime logs are written as **key-value pairs**:

```go
// Good example -- key-value pairs enable searchable fields
log.Info("Creating Deployment", "name", app.Name, "replicas", replicas)
// Output: INFO Creating Deployment {"name": "myapp", "replicas": 3}

// Bad example -- concatenating with fmt.Sprintf makes searching difficult
log.Info(fmt.Sprintf("Creating Deployment %s (replicas: %d)", app.Name, replicas))
// Output: INFO Creating Deployment myapp (replicas: 3)
```

When written as key-value pairs, log aggregation systems (ELK, Loki, etc.) can search by individual fields.

---

## 2. Metrics (Prometheus)

### Built-in Metrics

controller-runtime automatically provides metrics in Prometheus format.
These metrics are collected by default without writing any code:

| Metric                                      | Meaning                                      |
| ------------------------------------------- | -------------------------------------------- |
| `controller_runtime_reconcile_total`        | Reconcile call count (by success/error)       |
| `controller_runtime_reconcile_errors_total` | Reconcile error count                         |
| `controller_runtime_reconcile_time_seconds` | Reconcile execution time                      |
| `workqueue_depth`                           | Number of items queued in the Work Queue       |
| `workqueue_adds_total`                      | Number of items added to the Work Queue        |
| `workqueue_retries_total`                   | Number of retries                              |

### Metrics Server Configuration

The metrics server is configured in `cmd/main.go`:

```go
// Metrics bind address (default: "0" = disabled)
flag.StringVar(&metricsAddr, "metrics-bind-address", "0", ...)

// Pass metrics options when creating the Manager
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Metrics: metricsServerOptions,  // Metrics server configuration
    ...
})
```

To enable metrics, specify the address at runtime:

```bash
# Expose metrics over HTTP
--metrics-bind-address=:8080

# Expose metrics over HTTPS (recommended for production)
--metrics-bind-address=:8443 --metrics-secure=true
```

### ServiceMonitor (Configuring Prometheus to Scrape)

A ServiceMonitor is defined in `config/prometheus/monitor.yaml`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: controller-manager-metrics-monitor
spec:
  endpoints:
    - path: /metrics
      port: https
  selector:
    matchLabels:
      control-plane: controller-manager
```

In a cluster with Prometheus Operator installed, applying this ServiceMonitor
causes Prometheus to automatically scrape the metrics.

### Adding Custom Metrics

Beyond the built-in metrics, you can add business metrics:

```go
import "sigs.k8s.io/controller-runtime/pkg/metrics"
import "github.com/prometheus/client_golang/prometheus"

var (
    simpleAppCount = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "simpleapp_active_count",
        Help: "Number of currently active SimpleApp resources",
    })
)

func init() {
    metrics.Registry.MustRegister(simpleAppCount)
}

// Update inside Reconcile
simpleAppCount.Set(float64(activeCount))
```

---

## 3. Health Checks

### Liveness and Readiness

Health check endpoints are registered in `cmd/main.go`:

```go
// Liveness -- "Is the process alive?"
mgr.AddHealthzCheck("healthz", healthz.Ping)

// Readiness -- "Is it ready to accept requests?"
mgr.AddReadyzCheck("readyz", healthz.Ping)
```

The default port is `:8081`:

```bash
curl http://localhost:8081/healthz   # Liveness
curl http://localhost:8081/readyz    # Readiness
```

### Usage in Kubernetes

Configure probes in the Deployment so Kubernetes checks automatically:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8081
  initialDelaySeconds: 15
  periodSeconds: 20
readinessProbe:
  httpGet:
    path: /readyz
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

| Probe     | Behavior on Failure                                |
| --------- | -------------------------------------------------- |
| Liveness  | Pod is restarted                                   |
| Readiness | Pod is removed from Service (no traffic is sent)   |

---

## 4. Leader Election

When running multiple Operator replicas (for high availability), you need to decide which one actually executes Reconcile.

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    LeaderElection:   enableLeaderElection,
    LeaderElectionID: "1ad1a747.example.com",
})
```

```bash
# Enable leader election
--leader-elect=true
```

| State      | Behavior                                                    |
| ---------- | ----------------------------------------------------------- |
| Leader     | Executes Reconcile, collects metrics, handles Webhooks      |
| Non-Leader | Standby (automatically promoted if the Leader dies)         |

---

## Summary

```
Operator Running
+-- Logging (zap)
|   +-- log.Info()       -- Record key operations
|   +-- log.Error()      -- Record permanent errors
|   +-- Auto error logs  -- controller-runtime logs when Reconcile returns err
+-- Metrics (:8080 or :8443)
|   +-- Built-in metrics -- reconcile count, duration, queue depth, etc.
|   +-- Custom metrics   -- Add business-specific indicators
+-- Health Checks (:8081)
|   +-- /healthz         -- Liveness (is it dead?)
|   +-- /readyz          -- Readiness (is it ready?)
+-- Leader Election
    +-- Only one instance among many executes Reconcile
```
