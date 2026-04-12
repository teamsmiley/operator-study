# 09-1. Mutating Webhook -- 기본값 자동 설정

## Mutating Webhook이란?

CR이 생성/수정될 때 **API Server가 etcd에 저장하기 전에** 호출되는 HTTP 엔드포인트다.
요청을 가로채서 값을 자동으로 수정(mutate)한 뒤 돌려보낸다.

비유: 서류를 제출하면 접수 담당자가 빠진 항목을 채워주는 것.

```text
사용자가 CR 생성 요청
  -> API Server가 Mutating Webhook 호출
  -> Webhook: "replicas가 비어있네? 3으로 채워줄게"
  -> 수정된 CR이 다음 단계로 진행 (Validating Webhook -> etcd 저장)
```

## 언제 쓰나?

| 용도           | 예시                                                     |
| -------------- | -------------------------------------------------------- |
| 기본값 설정    | replicas 미지정 시 3으로 설정                            |
| 라벨 자동 추가 | 모든 CR에 `managed-by: myoperator` 라벨 추가             |
| 사이드카 주입  | Pod에 로깅/모니터링 컨테이너 자동 추가 (Istio가 이 방식) |
| 값 정규화      | image 이름에 태그가 없으면 `:latest` 자동 추가           |

## kubebuilder:default marker와의 차이

`+kubebuilder:default=1` 같은 marker도 기본값을 설정한다.
Mutating Webhook과 뭐가 다른가?

|                | kubebuilder:default marker                | Mutating Webhook                                |
| -------------- | ----------------------------------------- | ----------------------------------------------- |
| 실행 위치      | CRD 스키마 (API Server 내장)              | 외부 HTTP 서버                                  |
| 설정 가능한 값 | 고정값만 (`default=1`, `default="nginx"`) | 동적 계산 가능 (현재 시간, 다른 리소스 참조 등) |
| 조건부 설정    | 불가                                      | 가능 ("production NS면 replicas=3, dev면 1")    |

고정된 기본값은 marker로 충분하다. 동적이거나 조건부 기본값이 필요할 때 Mutating Webhook을 쓴다.

## Webhook은 별도 프로젝트인가?

아니다. 같은 Operator 프로젝트 안에 들어간다.

```text
myoperator/
  api/v1/
    simpleapp_types.go          <- CRD 정의 (이미 있음)
    simpleapp_webhook.go        <- Webhook 로직 (kubebuilder가 여기에 생성)
  internal/controller/
    simpleapp_controller.go     <- Reconcile 로직 (이미 있음)
```

같은 바이너리에서 Controller와 Webhook이 동시에 실행된다:

```text
Controller Pod (하나의 프로세스)
  |- Controller Manager: Reconcile 루프 실행
  |- Webhook Server: HTTPS 엔드포인트 리스닝 (포트 9443)
```

## 구현 방법

kubebuilder로 webhook scaffold를 생성한다:

```bash
kubebuilder create webhook --group apps.example.com --version v1 --kind SimpleApp --defaulting
```

`--defaulting` 플래그가 Mutating Webhook을 만든다.

생성되는 파일:

| 파일                          | 역할                                   |
| ----------------------------- | -------------------------------------- |
| `api/v1/simpleapp_webhook.go` | Webhook 로직 (Default 함수)            |
| `config/webhook/`             | Webhook 서버 설정 YAML                 |
| `config/certmanager/`         | TLS 인증서 설정 (Webhook은 HTTPS 필수) |

### 핵심 함수: Default()

```go
// Default는 SimpleApp CR이 생성/수정될 때 호출된다
func (r *SimpleApp) Default() {
    // image에 태그가 없으면 :latest 추가
    if !strings.Contains(r.Spec.Image, ":") {
        r.Spec.Image = r.Spec.Image + ":latest"
    }

    // replicas가 nil이면 기본값 3 설정
    if r.Spec.Replicas == nil {
        replicas := int32(3)
        r.Spec.Replicas = &replicas
    }
}
```

이 함수는 types 파일(SimpleApp 구조체)의 메서드로 정의된다.
Controller가 아니라 **API 타입에 직접 붙는다**는 점이 특징이다.

## Webhook이 HTTPS를 필요로 하는 이유

API Server가 Webhook을 호출할 때 HTTPS로 통신한다.
그래서 인증서가 필요하고, 보통 cert-manager를 사용한다.

```text
API Server --HTTPS--> Webhook 서버 (Controller Pod 안에서 실행)
                      인증서 필요!
```

로컬 개발 시에는 `make run`이 자체 서명 인증서를 자동으로 만들어준다.

## 다음 단계

Validating Webhook을 구현한다.
