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

### Phase 1: 환경 구성 + 기초 실습

- [x] Kubernetes Controller 패턴 이해 (Reconciliation Loop) -- notes/01 참고
- [x] kubectl + k3d + kubebuilder 설치 -- notes/00 참고
- [ ] k3d로 로컬 클러스터 생성 및 kubectl 연결 확인
- [ ] kubebuilder로 프로젝트 스캐폴딩
- [ ] 생성된 코드 구조 훑어보기 (CRD, Controller Runtime 개념을 코드로 이해)

### Phase 2: 첫 번째 Operator 만들기 (만들면서 배우기)

- [ ] 간단한 CRD 정의 (예: SimpleApp) -- CRD 개념은 여기서 실습으로 이해
- [ ] Controller 로직 구현 (Reconcile 함수) -- Reconcile Loop를 직접 코드로 체험
- [ ] k3d 클러스터에 배포하고 동작 확인
- [ ] RBAC 설정
- [ ] Status subresource 활용

### Phase 3: 심화 (동작하는 Operator 위에 기능 추가)

- [ ] Finalizer 패턴
- [ ] Owner Reference와 Garbage Collection
- [ ] Webhook (Validating / Mutating)
- [ ] 에러 핸들링 및 Retry 전략

### Phase 4: 운영

- [ ] Operator 배포 (OLM 또는 Helm)
- [ ] 모니터링 및 로깅
- [ ] 테스트 전략 (unit, integration, e2e)

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

## 디렉토리 구조 (예정)

```
Operator/
  CLAUDE.md          # 이 파일
  Readme.md          # 프로젝트 소개
  notes/             # 학습 노트
  examples/          # 단계별 예제 코드
  myoperator/        # 실제 Operator 프로젝트 (kubebuilder로 생성)
```

## 참고 자료

- Kubernetes 공식 문서: Custom Resources, Operator pattern
- kubebuilder book: https://book.kubebuilder.io
- controller-runtime godoc
- Programming Kubernetes (O'Reilly)
