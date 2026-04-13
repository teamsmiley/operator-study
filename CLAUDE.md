# Kubernetes Operator & Custom Controller 학습 프로젝트

## 프로젝트 목적

Kubernetes Operator 패턴과 Custom Controller를 체계적으로 학습하기 위한 프로젝트.
Claude가 선생님 역할을 하며, 개념 설명 -> 실습 -> 리뷰 순서로 진행한다.

## 교육 방식

- 개념을 설명할 때는 비유와 실제 사례를 함께 제공
- 코드 예제는 단계적으로 복잡도를 높여가며 작성
- 질문이 오면 바로 답을 주지 말고, 힌트를 먼저 주어 스스로 생각하게 유도 (단, 사용자가 직접 답을 요청하면 바로 제공)
- 한 번에 너무 많은 내용을 다루지 않고, 소화 가능한 단위로 나눠서 진행

## 학습 로드맵 (실습 중심)

### Phase 1: 환경 구성 + 기초 개념

- [x] 도구 설치 (Go, Docker, kubectl, k3d, kubebuilder) -- notes/00-environment-setup
- [x] k3d로 로컬 클러스터 생성 및 kubectl 연결 확인 -- notes/00-environment-setup
- [x] CRD와 CR 이해 (kubebuilder 없이 직접 실습) -- notes/01, 01.1
- [x] Controller 패턴 이해 (Reconciliation Loop, Watch) -- notes/02

### Phase 2: 첫 번째 Operator 만들기 (만들면서 배우기)

- [x] kubebuilder로 프로젝트 생성, 핵심 파일 2개 수정 -- notes/03
- [x] 빌드 및 배포 (make install/run/deploy) -- notes/04

### Phase 3: 심화 (동작하는 Operator 위에 기능 추가)

- [x] RBAC 설정 (클러스터 내 배포 시 필요) -- notes/05, 06
- [x] Status subresource 활용 -- notes/07
- [x] Owner Reference와 Garbage Collection -- notes/08
- [x] Finalizer 패턴 -- notes/09
- [x] Webhook (Validating / Mutating) -- notes/10, 10.1, 10.2, 10.3, 10.4
- [x] 에러 핸들링 및 Retry 전략 -- notes/11

### Phase 4: 운영

- [x] 모니터링 및 로깅 -- notes/12
- [x] 테스트 전략 (unit, integration, e2e) -- notes/13

## 기술 스택

- 언어: Go
- 프레임워크: controller-runtime (kubebuilder 기반)
- 로컬 클러스터: k3d (k3s 기반)
- Kubernetes 버전: 최신 stable

## 코드 컨벤션

- Go 표준 프로젝트 레이아웃 준수
- controller-runtime의 관례를 따름
- 주석은 한국어로 작성 (학습 목적)
- 커밋 메시지는 한국어로 작성

## 디렉토리 구조

```text
Operator/
  CLAUDE.md                        # 이 파일
  notes/                           # 학습 노트 (6학년도 이해 가능한 수준)
    00-environment-setup.md        # 도구 설치, k3d 클러스터, kubeconfig
    01-CRD와-CR.md                  # CRD/CR 개념, Snack 예제 실습
    01.1-CRD-파일-설명.md             # crd.yaml 한 줄씩 상세 설명
    02-Controller.md               # Controller, Watch, HTTP long-polling
    03-CRD-CR-Controller-합치기.md   # kubebuilder, 핵심 파일, 코드 비교
    04-빌드와-배포.md                 # make 명령어, 개발/운영 배포
    05-RBAC.md                     # RBAC 개념, Role/ClusterRole
    06-RBAC-적용.md                 # kubebuilder RBAC 마커, 실습
    07-Status-Subresource.md       # Status subresource 활용
    08-Owner-Reference와-Garbage-Collection.md  # Owner Reference, Cascading Deletion
    09-Finalizer.md                # Finalizer 패턴, 정리 로직
    10-Webhook.md                  # Webhook 개념 (Admission Controller)
    10.1-Mutating-Webhook.md       # Mutating Webhook 개념
    10.2-Mutating-Webhook-구현.md   # Mutating Webhook 구현 실습
    10.3-Validating-Webhook.md     # Validating Webhook 개념
    10.4-Validating-Webhook-구현.md # Validating Webhook 구현 실습
    11-에러-핸들링과-Retry-전략.md    # Reconcile 반환값, Backoff, Rate Limiting
    12-모니터링과-로깅.md             # zap 로깅, Prometheus 메트릭, 헬스체크
    13-테스트-전략.md                 # envtest, Webhook 테스트, E2E
  examples/                        # kubebuilder 없이 CRD/CR 직접 실습
  myoperator/                      # 실제 Operator 프로젝트 (kubebuilder)
    api/v1/simpleapp_types.go      # CRD 정의 (image, replicas)
    internal/controller/           # Controller (Reconcile 로직)
    internal/webhook/v1/           # Webhook 핸들러 (Validating/Mutating)
    config/samples/                # 테스트용 CR YAML
    config/webhook/                # Webhook 서비스 설정
```

## 참고 자료

- Kubernetes 공식 문서: Custom Resources, Operator pattern
- kubebuilder book: https://book.kubebuilder.io
- controller-runtime godoc
- Programming Kubernetes (O'Reilly)
