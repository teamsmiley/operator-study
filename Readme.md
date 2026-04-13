# Kubernetes Operator & Custom Controller

Kubernetes Operator 패턴과 Custom Controller를 체계적으로 학습하기 위한 프로젝트.
개념 설명 -> 실습 -> 리뷰 순서로 진행하며, 모든 학습 노트는 한국어/영어 두 가지 언어로 제공된다.

## 학습 로드맵

| Phase | 주제 | 노트 |
|-------|------|------|
| **1. 환경 구성 + 기초** | 도구 설치, k3d 클러스터, CRD/CR, Controller | 00 ~ 02 |
| **2. 첫 Operator** | kubebuilder 프로젝트 생성, 빌드 및 배포 | 03 ~ 04 |
| **3. 심화** | RBAC, Status, Owner Reference, Finalizer, Webhook, 에러 핸들링 | 05 ~ 11 |
| **4. 운영** | 모니터링/로깅, 테스트 전략 | 12 ~ 13 |

## 디렉토리 구조

```
Operator/
  ko/                   # 한국어 학습 노트 (GitBook)
    SUMMARY.md           # 목차
    notes/               # 00 ~ 13 학습 노트
  en/                   # English learning notes (GitBook)
    SUMMARY.md           # Table of contents
    notes/               # 00 ~ 13 translated notes
  examples/             # kubebuilder 없이 CRD/CR 직접 실습
  myoperator/           # kubebuilder 기반 Operator 프로젝트
    api/v1/              # CRD 정의 (SimpleApp)
    internal/controller/ # Reconcile 로직
    internal/webhook/v1/ # Validating/Mutating Webhook
```

## 기술 스택

- **언어**: Go
- **프레임워크**: controller-runtime (kubebuilder)
- **로컬 클러스터**: k3d (k3s 기반)

## 학습 노트

한국어와 영어 두 가지 버전으로 제공된다.

- **한국어**: [ko/notes/](ko/notes/)
- **English**: [en/notes/](en/notes/)

### 목차

1. [환경 설정](ko/notes/00-environment-setup.md) -- Go, Docker, k3d, kubebuilder 설치
2. [CRD와 CR](ko/notes/01-CRD와-CR.md) -- Custom Resource Definition 개념과 실습
3. [CRD 파일 상세 설명](ko/notes/01.1-CRD-파일-설명.md) -- crd.yaml 한 줄씩 해부
4. [Controller](ko/notes/02-Controller.md) -- Reconciliation Loop, Watch 메커니즘
5. [CRD + CR + Controller 합치기](ko/notes/03-CRD-CR-Controller-합치기.md) -- kubebuilder로 통합
6. [빌드와 배포](ko/notes/04-빌드와-배포.md) -- make install/run/deploy
7. [RBAC 개념](ko/notes/05-RBAC.md) -- Role, ClusterRole, Binding
8. [RBAC 적용](ko/notes/06-RBAC-적용.md) -- kubebuilder RBAC 마커 실습
9. [Status Subresource](ko/notes/07-Status-Subresource.md) -- 상태 보고 패턴
10. [Owner Reference와 Garbage Collection](ko/notes/08-Owner-Reference와-Garbage-Collection.md)
11. [Finalizer](ko/notes/09-Finalizer.md) -- 삭제 전 정리 로직
12. [Webhook 개념](ko/notes/10-Webhook.md) -- Admission Controller 이해
13. [Mutating Webhook 개념](ko/notes/10.1-Mutating-Webhook.md)
14. [Mutating Webhook 구현](ko/notes/10.2-Mutating-Webhook-구현.md)
15. [Validating Webhook 개념](ko/notes/10.3-Validating-Webhook.md)
16. [Validating Webhook 구현](ko/notes/10.4-Validating-Webhook-구현.md)
17. [에러 핸들링과 Retry 전략](ko/notes/11-에러-핸들링과-Retry-전략.md) -- Backoff, Rate Limiting
18. [모니터링과 로깅](ko/notes/12-모니터링과-로깅.md) -- zap, Prometheus, 헬스체크
19. [테스트 전략](ko/notes/13-테스트-전략.md) -- envtest, Webhook 테스트, E2E

## 참고 자료

- [kubebuilder book](https://book.kubebuilder.io)
- [Kubernetes 공식 문서 - Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Kubernetes 공식 문서 - Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [controller-runtime GoDoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- Programming Kubernetes (O'Reilly)
