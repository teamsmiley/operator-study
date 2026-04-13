# 02. Controller -- A Program That Watches and Reconciles

## What is a Controller?

**A program that compares "desired state" with "current state" and adjusts them if they differ.**

It's not specific to CRs. Kubernetes already has many built-in Controllers.

## Analogy: Classroom Thermostat

```text
Set temperature: 24C (desired state)
Current temperature: 28C (current state)

What the thermostat does:
1. Check current temperature (28C)
2. Compare with set temperature (24C vs 28C -- different!)
3. Turn on air conditioning
4. Go back to step 1

When it reaches 24C:
1. Check current temperature (24C)
2. Compare with set temperature (24C vs 24C -- same!)
3. Do nothing
4. Go back to step 1
```

This is the **Reconciliation Loop**.

## Controllers Already in Kubernetes

When you install Kubernetes, several Controllers are already running.

### Example: Deployment Controller

```bash
# Declare "maintain 3 nginx Pods"
kubectl apply -f deployment.yaml   # replicas: 3
```

```text
Desired state: 3 Pods
Current state: 0 Pods

What the Deployment Controller does:
1. Check current Pod count (0)
2. Compare with desired count (3 vs 0)
3. Create 3 Pods
4. Done? No. Keep watching.

If someone kills 1 Pod:
1. Check current Pod count (2)
2. Compare with desired count (3 vs 2)
3. Create 1 more Pod
```

This is **self-healing**. The system recovers automatically without human intervention.

### Example: Other Built-in Controllers

| Controller            | Watches                      | Does                           |
| --------------------- | ---------------------------- | ------------------------------ |
| Deployment Controller | Deployment                   | Creates/manages ReplicaSets    |
| ReplicaSet Controller | ReplicaSet                   | Maintains Pod count            |
| Service Controller    | Service (type: LoadBalancer) | Creates cloud load balancers   |
| Job Controller        | Job                          | Creates task Pods, tracks done |

They all follow the same pattern: **Watch -> Compare -> Adjust -> Repeat**

## The Reconcile Function

The core of a Controller is a single **Reconcile function**.

```text
Something changed (resource created/modified/deleted)
    |
    v
Reconcile function is called
    |
    v
Check current state --> Compare with desired state --> Adjust if needed
    |
    v
Done (wait for next change)
```

The Reconcile function is called when it receives a notification that "some resource has changed."
Inside the function, it checks the current state and adjusts it if it differs from the desired state.

## Watch: How Are Changes Detected?

Controllers do NOT check "any changes?" every second (Polling).
They open a **Watch connection** to the API Server and receive **instant notifications** when changes occur.

```text
API Server                    Controller
    |                              |
    | <-- "Notify me of Deployment |  (Watch registration)
    |      changes"                |
    |                              |
    |  (quiet if nothing happens)  |  (waiting... no CPU usage)
    |                              |
  [Someone modifies a Deployment]  |
    |                              |
    |-- "Deployment changed!" ---->|  Reconcile called!
    |                              |
```

### Technical Implementation of Watch: HTTP Long-Polling

Watch internally uses **HTTP long-polling (chunked response)**.

Normal HTTP request:

```text
Controller --> GET /deployments --> API Server
Controller <-- Response [{...}, {...}] -- API Server
(connection closed)
```

Watch HTTP request:

```text
Controller --> GET /deployments?watch=true --> API Server
Controller <-- "Connection stays open, I'll send changes through this connection"

[10 seconds later -- change occurs]
Controller <-- {"type":"MODIFIED","object":{...}}    (sent through same connection)

[30 seconds later -- deletion occurs]
Controller <-- {"type":"DELETED","object":{...}}     (sent through same connection)

... (continues until connection drops)
```

Key point: Instead of sending one response and ending, **the connection stays open and JSON is continuously sent whenever events occur.**

### Polling vs Watch Comparison

| Method                 | Behavior                                        | Cost                | Response Time     |
| ---------------------- | ----------------------------------------------- | ------------------- | ----------------- |
| Polling                | Ask "any changes?" every N seconds              | CPU/network waste   | Slow (N sec delay)|
| Watch (HTTP long-poll) | Keep one connection open, instant notification  | Almost none         | Instant (ms)      |

### Try It Yourself

```bash
# Terminal 1: View the Watch stream directly
kubectl get snack --watch

# Terminal 2: Create or delete a snack -- Terminal 1 shows real-time output
kubectl apply -f examples/cr.yaml
kubectl delete snack afternoon-snack
```

`kubectl get --watch` uses exactly this HTTP long-polling.
Controllers work on the same principle.

### Event Deduplication

When events are received via Watch, they don't trigger Reconcile directly -- they go into a **Work Queue**.
If multiple events come in for the same resource, **they are merged into one (deduplication)**.

```text
3 Deployment changes within 1 second:
  Event 1: MODIFIED --> Add "my-app" to Queue
  Event 2: MODIFIED --> "my-app" already in Queue, skip
  Event 3: MODIFIED --> "my-app" already in Queue, skip

Result: Reconcile runs only once. Only checks the latest state.
```

Controllers focus on **"is the current state correct?"** not "what events came in?"

### Watch Timeout and Auto-Reconnection

Watch connections **time out after ~5-10 minutes**. They don't stay open forever.

```text
1. Open a Watch connection
2. Receive events for 5-10 minutes
3. Connection drops due to timeout
4. Automatically reconnect Watch
5. Go back to step 2 (infinite loop)
```

This reconnection is **handled automatically by the controller-runtime library**. We don't need to write any code.

Events might be missed between reconnections, so there are safety nets:

| Mechanism            | Role                                                             |
| -------------------- | ---------------------------------------------------------------- |
| resourceVersion      | On reconnect, request "only events after the last version I saw" |
| Resync (default 10h) | As a safety net, re-run Reconcile for all resources              |

## What is controller-runtime?

When building a Controller, we **don't implement** Watch, event reception, reconnection, Work Queue, deduplication, etc. ourselves.
**controller-runtime** is the Go library that handles all of this for us.

```text
What we write:                What controller-runtime handles:
-----------------             ----------------------------
Reconcile function            Watch connection/reconnection
CRD field definitions         Event reception
                              Work Queue management
                              Deduplication
                              Retry on errors
                              Leader election (HA)
```

We only need to **implement one Reconcile function**. controller-runtime handles the rest.

kubebuilder is a **project scaffolding tool** that auto-generates code using controller-runtime.

```text
kubebuilder          = project generator (code generation)
controller-runtime   = library (the actual runtime engine)
Our Reconcile func   = business logic (what we write)
```

## Controller vs Operator

|      | Controller        | Operator                        |
| ---- | ----------------- | ------------------------------- |
| What | A program (code)  | A pattern (bundle)              |
| Made of | Reconcile func | CRD + Controller + deploy config |
| Analogy | Snack handler  | Entire snack management system  |

Operator = CRD + Controller. A Controller is part of an Operator.

## Summary

- Controller = A program that watches and reconciles
- Core behavior = Reconciliation Loop (watch -> compare -> adjust -> repeat)
- Change detection = Watch (event-driven, not Polling)
- Kubernetes already has many built-in Controllers
- What we'll build = A **custom Controller** that Watches CRs

## Next Step

Combine CRD + CR + Controller into a complete Operator.
