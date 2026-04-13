# 09. Webhook -- Validation and Defaults on CR Create/Update

## Why Is This Needed?

CRD validation markers can handle basic validation:

```go
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=10
Replicas *int32 `json:"replicas,omitempty"`
```

However, the following types of validation are impossible with markers alone:

- The image name must include a tag (`nginx:1.25` OK, `nginx` NG)
- Only specific registries are allowed (must start with `docker.io/`)
- When scaling down replicas, you cannot reduce by more than half at once
- Validate against other resources (e.g., ConfigMaps)

Webhook handles this kind of business logic validation.

## CRD Validation vs Webhook

|                          | CRD validation marker                     | Validating Webhook                                  |
| ------------------------ | ----------------------------------------- | --------------------------------------------------- |
| Execution location       | Inside API Server (OpenAPI schema)        | External HTTP server (Controller Pod)               |
| Validation level         | Simple: type, min/max, required, enum     | Complex: cross-field comparison, external lookups, business logic |
| Code required?           | No (marker annotations only)              | Yes (write Go functions)                            |
| Cross-resource reference | Not possible                              | Possible                                            |

Principle: Use CRD validation when it suffices. Add a Webhook only when CRD validation is not enough.
Webhooks require a separate server and certificate management, which increases complexity.

## Two Types

| Type       | When it runs                                   | What it does            | Analogy                                                          |
| ---------- | ---------------------------------------------- | ----------------------- | ---------------------------------------------------------------- |
| Mutating   | On CR create/update request (runs first)       | Auto-modifies values    | Reception desk: "You don't have a name tag. Let me add one for you." |
| Validating | On CR create/update request (runs after Mutating) | Validates and allows/denies | Security guard: "No badge, no entry."                            |

Execution order:

```text
User submits CR create/update request
  -> Mutating Webhook: auto-fills missing values (sets defaults, etc.)
  -> Validating Webhook: validates against rules
  -> If passed, stored in etcd
  -> If rejected, error returned
```

## Implementation

(Covered in the next steps)
