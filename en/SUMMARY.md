# Table of contents

* [K8S Operator](Readme.md)

## Phase 1: Environment Setup + Basics

* [Environment Setup (Go, Docker, k3d, kubebuilder)](notes/00-environment-setup.md)
* [CRD and CR](notes/01-crd-and-cr.md)
* [CRD File Explained](notes/01.1-crd-file-explained.md)
* [Controller](notes/02-controller.md)

## Phase 2: Building Your First Operator

* [Combining CRD + CR + Controller](notes/03-combining-crd-cr-controller.md)
* [Build and Deploy](notes/04-build-and-deploy.md)

## Phase 3: Advanced Topics

* [RBAC Concepts](notes/05-rbac.md)
* [Applying RBAC](notes/06-applying-rbac.md)
* [Status Subresource](notes/07-status-subresource.md)
* [Owner Reference and Garbage Collection](notes/08-owner-reference-and-garbage-collection.md)
* [Finalizer](notes/09-finalizer.md)
* [Webhook Concepts](notes/10-webhook.md)
* [Mutating Webhook Concepts](notes/10.1-mutating-webhook.md)
* [Mutating Webhook Implementation](notes/10.2-mutating-webhook-implementation.md)
* [Validating Webhook Concepts](notes/10.3-validating-webhook.md)
* [Validating Webhook Implementation](notes/10.4-validating-webhook-implementation.md)
* [Error Handling and Retry Strategy](notes/11-error-handling-and-retry-strategy.md)

## Phase 4: Operations

* [Monitoring and Logging](notes/12-monitoring-and-logging.md)
* [Testing Strategy](notes/13-testing-strategy.md)
