# 03. CRD + CR + Controller 합치기 = Operator

## 지금까지 배운 것

```text
01. CRD  = 새로운 리소스 종류를 등록 (간식 신청 양식 등록)
02. CR   = 그 종류의 실제 인스턴스 (초코파이 30개 신청서)
03. Controller = 감시하고 맞춰주는 프로그램 (담당 선생님)
```

이 세 가지를 합치면 **Operator**이다.

## 목표

SimpleApp이라는 CR을 만들면, Controller가 자동으로 Deployment를 만들어주는 Operator.

```text
사용자가 하는 일:             Controller가 하는 일:

"nginx를 2개 실행해줘"        "알겠습니다" -> Deployment 생성
                                           -> Pod 2개 실행됨
kind: SimpleApp
spec:                   --->  kind: Deployment
  image: nginx:1.25           spec:
  replicas: 2                   replicas: 2
                                containers:
                                  - image: nginx:1.25
```

## kubebuilder란?

CRD, Controller, 빌드 설정 등을 **자동으로 생성해주는 도구**이다.

01장에서 CRD를 직접 YAML로 작성했는데, kubebuilder는 **Go 코드로 CRD를 정의하면 YAML을 자동 생성**해준다.

```text
직접 하는 방법:
  crd.yaml 직접 작성 + controller.go 직접 작성 + 빌드 설정 직접 작성

kubebuilder 사용:
  kubebuilder가 뼈대 코드를 자동 생성 -> 우리는 핵심 로직만 작성
```

비유: 집을 지을 때

- 직접: 기초 공사, 배관, 전기, 벽, 지붕 전부 직접
- kubebuilder: 기본 골조가 완성된 상태에서 인테리어만 하면 됨

## 프로젝트 생성 과정

### 1단계: 프로젝트 초기화

```bash
cd ~/Desktop/Operator/myoperator
kubebuilder init --domain example.com --repo github.com/teamsmiley/myoperator
```

| 옵션     | 값                               | 의미                                 |
| -------- | -------------------------------- | ------------------------------------ |
| --domain | example.com                      | CRD API Group의 도메인 부분          |
| --repo   | github.com/teamsmiley/myoperator | Go module 이름 (실제 repo 없어도 됨) |

이 명령은 빌드 설정(Makefile, Dockerfile, go.mod 등)만 생성한다. 아직 CRD와 Controller는 없다.

### 2단계: CRD + Controller 생성

```bash
kubebuilder create api --group apps --version v1 --kind SimpleApp --resource --controller
```

| 옵션         | 값        | 의미                                                |
| ------------ | --------- | --------------------------------------------------- |
| --group      | apps      | API Group 이름 (domain과 합쳐져서 apps.example.com) |
| --version    | v1        | 리소스 버전                                         |
| --kind       | SimpleApp | 리소스 이름 (YAML에서 kind: SimpleApp)              |
| --resource   |           | CRD 타입 파일 생성                                  |
| --controller |           | Controller 파일 생성                                |

이 명령으로 **두 파일**이 핵심적으로 생성된다.

## 핵심 파일 2개

kubebuilder가 많은 파일을 생성하지만, 우리가 수정하는 파일은 **2개뿐**이다.

### 파일 1: api/v1/simpleapp_types.go -- CRD 정의

01장에서 `crd.yaml`을 직접 작성했던 것을 **Go 코드로 작성**하는 것이다.

```text
01장 (직접 YAML):                    kubebuilder (Go 코드):

schema:                              type SimpleAppSpec struct {
  properties:                            Image    string
    menu:                                Replicas *int32
      type: string           <-->    }
    quantity:
      type: integer
```

kubebuilder가 Go 코드를 읽어서 CRD YAML을 자동 생성해준다 (`make manifests`).

실제 코드:

```go
// SimpleAppSpec -- 사용자가 "원하는 상태"를 선언하는 부분
type SimpleAppSpec struct {
    // 컨테이너 이미지 (예: nginx:1.25)
    Image string `json:"image"`

    // 실행할 Pod 수 (기본값 1, 최대 10)
    Replicas *int32 `json:"replicas,omitempty"`
}
```

01장의 Snack CRD에서 `menu`, `quantity` 필드를 정의한 것처럼,
여기서는 `Image`, `Replicas` 필드를 정의한 것이다.

**참고: CRD 필드 이름은 자유이다.** `pizza`, `count`로 해도 동작한다.
하지만 Controller 안에서 이 값을 k8s Deployment에 넣어주기 때문에,
k8s 기본 리소스와 **같은 이름**을 쓰는 것이 관례이다.

```text
CRD 필드 (자유)         Controller가 변환         k8s Deployment (정해져 있음)
-----------------       ----------------         --------------------------
spec.image       -->    Controller가 읽어서 -->   containers[0].image
spec.replicas    -->    Controller가 읽어서 -->   spec.replicas
```

CRD 필드 이름 = 자유 (관례상 같게)
k8s Deployment 필드 = 정해져 있음 (바꿀 수 없음)

### 파일 2: internal/controller/simpleapp_controller.go -- Controller

02장에서 배운 Controller의 Reconcile 함수가 여기에 있다.

kubebuilder가 생성한 초기 상태 (before):

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    _ = logf.FromContext(ctx)

    // TODO(user): your logic here

    return ctrl.Result{}, nil
}
```

우리가 변경한 상태 (after):

```go
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := logf.FromContext(ctx)

    // 1. SimpleApp CR을 가져온다 (원하는 상태 확인)
    var app myappsv1.SimpleApp
    if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil  // 삭제된 경우, 무시
        }
        return ctrl.Result{}, err
    }

    // 2. Deployment가 있는지 확인한다
    var deploy appsv1.Deployment
    err := r.Get(ctx, req.NamespacedName, &deploy)

    if errors.IsNotFound(err) {
        // 2a. 없으면 -> 만든다
        log.Info("Deployment 생성", "name", app.Name)
        deploy := r.buildDeployment(&app)
        ctrl.SetControllerReference(&app, deploy, r.Scheme)
        return ctrl.Result{}, r.Create(ctx, deploy)
    }

    // 2b. 있으면 -> 비교하고, 바뀌었으면 업데이트
    needsUpdate := false
    if *deploy.Spec.Replicas != *app.Spec.Replicas {
        deploy.Spec.Replicas = app.Spec.Replicas
        needsUpdate = true
    }
    if deploy.Spec.Template.Spec.Containers[0].Image != app.Spec.Image {
        deploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
        needsUpdate = true
    }
    if needsUpdate {
        return ctrl.Result{}, r.Update(ctx, &deploy)
    }
    return ctrl.Result{}, nil
}
```

before vs after 비교:

|                | before (kubebuilder 초기) | after (우리가 수정)                 |
| -------------- | ------------------------- | ----------------------------------- |
| Reconcile 내용 | 빈 칸 (TODO)              | CR 조회 -> Deployment 생성/업데이트 |
| 코드 줄 수     | 2줄                       | 약 30줄                             |
| 동작           | 아무것도 안 함            | SimpleApp CR에 맞춰 Deployment 관리 |

## 전체 흐름 한눈에 보기

```text
1. kubebuilder create api     --> 뼈대 코드 생성 (types.go + controller.go)
2. types.go 수정              --> CRD 필드 정의 (Image, Replicas)
3. controller.go 수정         --> Reconcile 로직 작성
4. make install               --> CRD를 클러스터에 등록 (01장의 crd.yaml apply와 같음)
5. make run                   --> Controller 시작 (Watch 대기)
6. kubectl apply CR           --> CR 생성 (01장의 cr.yaml apply와 같음)
7. Controller가 감지          --> Reconcile 실행 -> Deployment 생성
8. Deployment Controller      --> Pod 생성 (이건 k8s 내장 Controller가 처리)
```

## 정리

| 01장에서 직접 한 것    | kubebuilder가 대신 해주는 것                   |
| ---------------------- | ---------------------------------------------- |
| crd.yaml 직접 작성     | types.go에서 자동 생성 (`make manifests`)      |
| kubectl apply crd.yaml | `make install`                                 |
| Controller 없었음      | controller.go 뼈대 자동 생성, Reconcile만 구현 |

**우리가 실제로 작성한 코드: types.go의 필드 정의 + controller.go의 Reconcile 로직. 이 두 가지가 전부이다.**

다음 문서: [04-빌드와-배포.md](04-빌드와-배포.md) -- make 명령어, 배포 방법
