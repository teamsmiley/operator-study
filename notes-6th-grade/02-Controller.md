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
