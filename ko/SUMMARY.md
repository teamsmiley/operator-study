# Table of contents

* [K8S Operator](Readme.md)

## Phase 1: 환경 구성 + 기초 개념

* [환경 설정 (Go, Docker, k3d, kubebuilder)](notes/00-environment-setup.md)
* [CRD와 CR](notes/01-CRD와-CR.md)
* [CRD 파일 상세 설명](notes/01.1-CRD-파일-설명.md)
* [Controller](notes/02-Controller.md)

## Phase 2: 첫 번째 Operator 만들기

* [CRD + CR + Controller 합치기](notes/03-CRD-CR-Controller-합치기.md)
* [빌드와 배포](notes/04-빌드와-배포.md)

## Phase 3: 심화

* [RBAC 개념](notes/05-RBAC.md)
* [RBAC 적용](notes/06-RBAC-적용.md)
* [Status Subresource](notes/07-Status-Subresource.md)
* [Owner Reference와 Garbage Collection](notes/08-Owner-Reference와-Garbage-Collection.md)
* [Finalizer](notes/09-Finalizer.md)
* [Webhook 개념](notes/10-Webhook.md)
* [Mutating Webhook 개념](notes/10.1-Mutating-Webhook.md)
* [Mutating Webhook 구현](notes/10.2-Mutating-Webhook-구현.md)
* [Validating Webhook 개념](notes/10.3-Validating-Webhook.md)
* [Validating Webhook 구현](notes/10.4-Validating-Webhook-구현.md)
* [에러 핸들링과 Retry 전략](notes/11-에러-핸들링과-Retry-전략.md)

## Phase 4: 운영

* [모니터링과 로깅](notes/12-모니터링과-로깅.md)
* [테스트 전략](notes/13-테스트-전략.md)
