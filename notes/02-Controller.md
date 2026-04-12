# 02. Controller -- 감시하고 맞춰주는 프로그램

## Controller란?

**"원하는 상태"와 "현재 상태"를 비교해서, 다르면 맞춰주는 프로그램**이다.

CR과는 상관없다. Kubernetes에 원래 내장된 Controller들이 많다.

## 비유: 교실 온도 조절기

```text
설정 온도: 24도 (원하는 상태)
현재 온도: 28도 (현재 상태)

온도 조절기가 하는 일:
1. 현재 온도를 확인한다 (28도)
2. 설정 온도와 비교한다 (24도 vs 28도 -- 다르다!)
3. 에어컨을 켠다
4. 다시 1번으로 돌아간다

24도가 되면?
1. 현재 온도를 확인한다 (24도)
2. 설정 온도와 비교한다 (24도 vs 24도 -- 같다!)
3. 아무것도 안 한다
4. 다시 1번으로 돌아간다
```

이것이 **Reconciliation Loop (조정 루프)**이다.

## Kubernetes에 이미 있는 Controller들

Kubernetes를 설치하면 Controller가 이미 여러 개 돌고 있다.

### 예시: Deployment Controller

```bash
# "nginx Pod를 3개 유지해라"라고 선언
kubectl apply -f deployment.yaml   # replicas: 3
```

```text
원하는 상태: Pod 3개
현재 상태:   Pod 0개

Deployment Controller가 하는 일:
1. 현재 Pod 수를 확인 (0개)
2. 원하는 수와 비교 (3개 vs 0개)
3. Pod 3개를 만든다
4. 끝? 아니다. 계속 감시한다

만약 누가 Pod 1개를 죽이면:
1. 현재 Pod 수를 확인 (2개)
2. 원하는 수와 비교 (3개 vs 2개)
3. Pod 1개를 더 만든다
```

이것이 **자가 치유(Self-healing)**이다. 사람이 개입하지 않아도 스스로 복구한다.

### 예시: 다른 내장 Controller들

| Controller            | 감시 대상                    | 하는 일                  |
| --------------------- | ---------------------------- | ------------------------ |
| Deployment Controller | Deployment                   | ReplicaSet 생성/관리     |
| ReplicaSet Controller | ReplicaSet                   | Pod 수를 맞춤            |
| Service Controller    | Service (type: LoadBalancer) | 클라우드 LB 생성         |
| Job Controller        | Job                          | 작업 Pod 생성, 완료 추적 |

모두 같은 패턴이다: **감시 -> 비교 -> 조정 -> 반복**

## Reconcile 함수

Controller의 핵심은 **Reconcile 함수** 하나이다.

```text
무언가 변경됨 (리소스 생성/수정/삭제)
    |
    v
Reconcile 함수 호출
    |
    v
현재 상태 확인 --> 원하는 상태와 비교 --> 필요하면 조정
    |
    v
끝 (다음 변경까지 대기)
```

Reconcile 함수는 "어떤 리소스가 변경되었다"는 알림을 받으면 호출된다.
함수 안에서 현재 상태를 확인하고, 원하는 상태와 다르면 맞춘다.

## Watch: 변경을 어떻게 감지하는가

Controller는 매 초마다 "변경 있나?" 확인하는 것(Polling)이 **아니다**.
API Server에 **Watch 연결**을 걸어두고, 변경이 생기면 **즉시 알림**을 받는다.

```text
API Server                    Controller
    |                              |
    | <-- "Deployment 변경 알려줘" --|  (Watch 등록)
    |                              |
    |  (아무 일 없으면 조용...)       |  (대기중... CPU 안 씀)
    |                              |
  [누군가 Deployment 변경]          |
    |                              |
    |-- "Deployment 바뀌었다!" ---->|  Reconcile 호출!
    |                              |
```

### Watch의 기술적 구현: HTTP Long-Polling

Watch는 내부적으로 **HTTP long-polling (chunked response)**을 사용한다.

일반 HTTP 요청:

```text
Controller --> GET /deployments --> API Server
Controller <-- 응답 [{...}, {...}] -- API Server
(연결 종료)
```

Watch HTTP 요청:

```text
Controller --> GET /deployments?watch=true --> API Server
Controller <-- "연결 유지, 변경 있으면 이 연결로 보내줄게"

[10초 후 -- 변경 발생]
Controller <-- {"type":"MODIFIED","object":{...}}    (같은 연결로 전송)

[30초 후 -- 삭제 발생]
Controller <-- {"type":"DELETED","object":{...}}     (같은 연결로 전송)

... (연결이 끊어질 때까지 계속)
```

핵심: 응답을 한번 보내고 끝이 아니라, **연결을 열어둔 채로 이벤트가 생길 때마다 JSON을 계속 보내준다.**

### Polling vs Watch 비교

| 방식                   | 동작                                  | 비용              | 반응 속도       |
| ---------------------- | ------------------------------------- | ----------------- | --------------- |
| Polling                | 매 N초마다 "변경 있나?" 요청          | CPU/네트워크 낭비 | 느림 (N초 지연) |
| Watch (HTTP long-poll) | 연결 하나 열어두고, 변경 시 즉시 알림 | 거의 없음         | 즉시 (밀리초)   |

### 직접 확인해보기

```bash
# 터미널 1: Watch 스트림 직접 보기
kubectl get snack --watch

# 터미널 2: snack을 만들거나 삭제하면 터미널 1에 실시간으로 출력됨
kubectl apply -f examples/cr.yaml
kubectl delete snack afternoon-snack
```

`kubectl get --watch`가 바로 이 HTTP long-polling을 사용하는 것이다.
Controller도 같은 원리로 동작한다.

### 이벤트 중복 제거

Watch로 이벤트를 받으면 바로 Reconcile하는 것이 아니라, **Work Queue**에 넣는다.
같은 리소스에 대한 이벤트가 여러 개 들어오면 **하나로 합쳐진다 (중복 제거)**.

```text
1초 사이에 Deployment가 3번 변경됨:
  이벤트1: MODIFIED --> Queue에 "my-app" 추가
  이벤트2: MODIFIED --> Queue에 이미 "my-app" 있음, 무시
  이벤트3: MODIFIED --> Queue에 이미 "my-app" 있음, 무시

결과: Reconcile 1번만 실행. 최신 상태만 확인.
```

Controller는 "무슨 이벤트가 왔는지"가 아니라 **"지금 상태가 맞는지"**에 집중한다.

### Watch 타임아웃과 자동 재연결

Watch 연결은 **약 5~10분 후 타임아웃**으로 끊어진다. 영원히 열려있지 않는다.

```text
1. Watch 연결을 연다
2. 5~10분 동안 이벤트를 받는다
3. 타임아웃으로 연결이 끊어진다
4. 자동으로 Watch를 다시 연결한다
5. 2번으로 돌아감 (무한 반복)
```

이 재연결은 **controller-runtime 라이브러리가 자동으로 처리**한다. 우리가 코드를 쓸 필요 없음.

재연결 사이에 이벤트를 놓칠 수도 있는데, 이를 대비한 안전망:

| 메커니즘             | 역할                                                       |
| -------------------- | ---------------------------------------------------------- |
| resourceVersion      | 재연결 시 "마지막으로 본 버전 이후 이벤트만 달라"고 요청   |
| Resync (기본 10시간) | 혹시 놓친 이벤트 대비, 전체 리소스에 대해 Reconcile 재실행 |

## controller-runtime이란?

우리가 Controller를 만들 때 Watch, 이벤트 수신, 재연결, Work Queue, 중복 제거 등을 **직접 구현하지 않는다.**
이 모든 것을 대신 해주는 **Go 라이브러리**가 controller-runtime이다.

```text
우리가 쓰는 것:           controller-runtime이 해주는 것:
-----------------         ----------------------------
Reconcile 함수 구현       Watch 연결/재연결
CRD 필드 정의             이벤트 수신
                          Work Queue 관리
                          중복 제거
                          에러 시 재시도
                          리더 선출 (HA)
```

우리는 **Reconcile 함수 하나만 구현**하면 된다. 나머지는 controller-runtime이 처리한다.

kubebuilder는 이 controller-runtime을 사용하는 **프로젝트 뼈대를 자동 생성해주는 도구**이다.

```text
kubebuilder          = 프로젝트 생성기 (코드 생성)
controller-runtime   = 라이브러리 (실제 동작하는 엔진)
우리의 Reconcile 함수 = 비즈니스 로직 (우리가 작성)
```

## Controller vs Operator

|      | Controller       | Operator                     |
| ---- | ---------------- | ---------------------------- |
| 뭔가 | 프로그램 (코드)  | 패턴 (묶음)                  |
| 구성 | Reconcile 함수   | CRD + Controller + 배포 설정 |
| 비유 | 간식 담당 선생님 | 간식 운영 시스템 전체        |

Operator = CRD + Controller. Controller는 Operator의 일부이다.

## 정리

- Controller = 감시하고 맞춰주는 프로그램
- 핵심 동작 = Reconciliation Loop (감시 -> 비교 -> 조정 -> 반복)
- 변경 감지 = Watch (이벤트 기반, Polling 아님)
- Kubernetes에 이미 여러 내장 Controller가 있다
- 우리가 만들 것 = CR을 Watch하는 **커스텀 Controller**

## 다음 단계

CRD + CR + Controller를 합쳐서 전체 Operator로 만들어본다.
