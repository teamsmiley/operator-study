# 03. SimpleApp Operator 구현

## 목표

SimpleApp CR을 만들면 Deployment가 자동으로 생성되는 Operator.

```
사용자가 이걸 apply하면:          Controller가 이걸 만들어줌:

kind: SimpleApp                  kind: Deployment
spec:                            spec:
  image: nginx:1.25      --->      replicas: 2
  replicas: 2                      template:
                                     containers:
                                       - image: nginx:1.25
```

## 1단계: CRD 필드 정의 (api/v1/simpleapp_types.go)

기존 예제 필드 `Foo`를 제거하고, 실제 필요한 필드를 추가했다.

```go
type SimpleAppSpec struct {
    // 컨테이너 이미지 (예: nginx:1.25)
    // +kubebuilder:validation:Required
    Image string `json:"image"`

    // 실행할 Pod 수 (기본값 1, 최대 10)
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=10
    // +kubebuilder:default=1
    // +optional
    Replicas *int32 `json:"replicas,omitempty"`
}
```

### 왜 Replicas는 `*int32` (포인터)인가?

- `int32`로 하면 값을 안 넣었을 때 0이 되어 "0개의 Pod"를 원한다는 뜻이 됨
- `*int32`로 하면 값을 안 넣았을 때 nil이 되어 "지정 안 함 -> 기본값 사용"이 됨
- `+kubebuilder:default=1`과 조합하면, 사용자가 replicas를 생략하면 1이 기본값으로 들어감

## 2단계: Controller 구현 (internal/controller/simpleapp_controller.go)

### Reconcile 함수의 흐름

```
Reconcile 호출
    |
    v
1. SimpleApp CR 가져오기 (r.Get)
    |-- 삭제됨(NotFound) -> 종료 (OwnerReference가 Deployment 정리)
    |-- 에러 -> 에러 반환 (자동 재시도)
    v
2. Deployment 가져오기 (r.Get)
    |-- 없음(NotFound) -> 3a로
    |-- 있음 -> 3b로
    v
3a. Deployment 생성
    - buildDeployment()로 Deployment 오브젝트 구성
    - OwnerReference 설정
    - r.Create()로 생성
    v
3b. Deployment 업데이트 (변경 있을 때만)
    - image 비교
    - replicas 비교
    - 변경 있으면 r.Update()
```

### 핵심 코드 설명

#### CR 조회 (Desired State 확인)

```go
var app myappsv1.SimpleApp
if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
    if errors.IsNotFound(err) {
        return ctrl.Result{}, nil  // 삭제된 경우, 무시
    }
    return ctrl.Result{}, err      // 다른 에러, 재시도
}
```

#### OwnerReference 설정

```go
ctrl.SetControllerReference(&app, deploy, r.Scheme)
```

이 한 줄이 하는 일:
- Deployment의 metadata에 SimpleApp을 "소유자"로 등록
- SimpleApp이 삭제되면 Kubernetes가 Deployment를 자동으로 삭제 (Garbage Collection)
- 직접 Deployment를 삭제하는 코드를 쓸 필요가 없음

#### Owns로 Deployment 변경 감지

```go
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&myappsv1.SimpleApp{}).      // SimpleApp 변경 시 Reconcile
        Owns(&appsv1.Deployment{}).      // 소유한 Deployment 변경 시에도 Reconcile
        Named("simpleapp").
        Complete(r)
}
```

`Owns`가 없으면: 누군가 Deployment를 직접 수정해도 Controller가 모름
`Owns`가 있으면: Deployment가 바뀌면 Controller가 감지하고 원래 상태로 되돌림 (자가 치유)

#### buildDeployment -- Deployment 오브젝트 생성

SimpleApp의 spec을 바탕으로 Deployment 오브젝트를 조립하는 함수.

```go
func (r *SimpleAppReconciler) buildDeployment(app *myappsv1.SimpleApp) *appsv1.Deployment {
    replicas := int32(1)
    if app.Spec.Replicas != nil {
        replicas = *app.Spec.Replicas
    }

    return &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      app.Name,       // SimpleApp과 같은 이름
            Namespace: app.Namespace,   // SimpleApp과 같은 namespace
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": app.Name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": app.Name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "app",
                            Image: app.Spec.Image,
                        },
                    },
                },
            },
        },
    }
}
```

#### Deployment 업데이트 로직

Deployment가 이미 존재하면, image와 replicas가 변경되었는지 비교하고 필요할 때만 업데이트한다.

```go
replicas := int32(1)
if app.Spec.Replicas != nil {
    replicas = *app.Spec.Replicas
}

needsUpdate := false
if *deploy.Spec.Replicas != replicas {
    deploy.Spec.Replicas = &replicas
    needsUpdate = true
}
if deploy.Spec.Template.Spec.Containers[0].Image != app.Spec.Image {
    deploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
    needsUpdate = true
}

if needsUpdate {
    r.Update(ctx, &deploy)
}
```

변경이 없으면 `r.Update()`를 호출하지 않는다. 이것이 멱등성(Idempotency)이다.
notes/01에서 배운 "같은 작업을 여러 번 수행해도 결과가 동일한 성질"이 여기서 구현된다.

#### RBAC marker

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps,...
// +kubebuilder:rbac:groups=apps,resources=deployments,...
```

Controller가 Deployment를 만들려면 Deployment에 대한 권한이 필요하다.
두 번째 줄은 `make manifests` 시 RBAC YAML에 Deployment 접근 권한을 추가한다.

## 3단계: 빌드 및 배포

### make install과 make run 이해하기

#### make install -- "CRD를 클러스터에 등록"

내부적으로 하는 일:

```text
make install
    |
    v
controller-gen으로 CRD YAML 생성 (config/crd/bases/에 저장)
    |
    v
kubectl apply -f config/crd/bases/
```

Kubernetes API Server에 "SimpleApp이라는 리소스가 존재한다"를 등록하는 것이다.
비유: 도서관에 새로운 장르를 등록하는 것. 아직 책(CR)은 없고, 장르(CRD)만 등록한 상태.

이걸 안 하면 `kubectl apply -f simpleapp.yaml` 할 때 `no matches for kind "SimpleApp"` 에러가 난다.

#### make run -- "Controller를 로컬 PC에서 실행"

내부적으로 하는 일:

```text
make run
    |
    v
go run ./cmd/main.go
    |
    v
Manager 시작
    |
    v
API Server에 Watch 연결 (SimpleApp과 Deployment 변경을 감시)
    |
    v
이벤트 대기중... (아무것도 안 함)
    |
[누군가 SimpleApp CR을 생성/수정/삭제]
    |
    v
Reconcile 함수 호출!
```

비유: 도서관에 장르를 등록한 다음, 사서(Controller)가 출근하는 것.
사서가 없으면 새 책(CR)이 들어와도 아무도 분류하지 않는다.

#### 전체 흐름 (순서가 중요하다)

```text
1. make install     --> CRD 등록 (장르 등록)
2. make run         --> Controller 실행 (사서 출근)
3. kubectl apply CR --> CR 생성 (새 책 입고)
4. Controller       --> Reconcile 실행 (사서가 책을 분류 -> Deployment 생성)
```

순서가 바뀌면 안 되는 이유:
- `make install` 없이 CR 생성 -> "SimpleApp이 뭔지 모름" 에러
- `make run` 없이 CR 생성 -> CR은 etcd에 저장되지만 아무 일도 안 일어남 (Deployment 안 만들어짐)

### 개발 모드 vs 운영 배포

| 항목 | make run (개발용) | make deploy (운영용) |
|------|-------------------|---------------------|
| 실행 위치 | 로컬 PC | 클러스터 안 (Pod) |
| 권한 | kubeconfig 사용 | RBAC (ServiceAccount) |
| 재시작 | 수동 (Ctrl+C, 다시 실행) | 자동 (Pod crash -> 재시작) |
| 용도 | 개발/디버깅 | 실제 운영 |

#### 운영 배포 순서

```bash
# 1. Docker 이미지 빌드
make docker-build IMG=ghcr.io/teamsmiley/myoperator:v0.1.0

# 2. 이미지를 레지스트리에 push
make docker-push IMG=ghcr.io/teamsmiley/myoperator:v0.1.0

# 3. 클러스터에 배포 (CRD + Controller Deployment + RBAC + ServiceAccount)
make deploy IMG=ghcr.io/teamsmiley/myoperator:v0.1.0
```

#### 운영 배포 흐름

```text
개발자 PC                           클러스터
---------                           --------
코드 수정
    |
make docker-build  --> 이미지 생성
    |
make docker-push   --> 레지스트리에 업로드 --> 레지스트리 (ghcr.io 등)
    |                                            |
make deploy        --> kubectl apply ----------> CRD 등록
                                                 Controller Pod 생성
                                                 RBAC 설정
```

배포되면 Controller Pod가 클러스터 안에서 `make run`과 동일한 일을 한다.
로컬 PC에서 터미널을 열어둘 필요가 없다.

#### k3d에서 로컬 이미지로 테스트 (레지스트리 없이)

```bash
make docker-build IMG=myoperator:v0.1.0
k3d image import myoperator:v0.1.0 -c operator-lab
make deploy IMG=myoperator:v0.1.0
```

### 빌드

```bash
cd ~/Desktop/Operator/myoperator

# types.go 변경 후 코드 생성 (DeepCopy 등)
make generate

# CRD YAML, RBAC YAML 생성
make manifests

# 빌드 확인
make build
```

### 배포 (순서가 중요하다)

```bash
# 1. kubectl이 k3d 클러스터를 가리키고 있는지 확인
kubectx k3d-operator-lab

# 2. CRD를 클러스터에 설치 (반드시 make run 전에)
make install

# 3. CRD 설치 확인
kubectl get crd simpleapps.apps.example.com

# 4. Controller를 로컬에서 실행 (터미널을 점유한다)
make run
```

### 주의사항

- `make install`을 하지 않으면 CR 생성 시 `no matches for kind "SimpleApp"` 에러 발생
- `make run`은 현재 셸의 KUBECONFIG를 사용한다. k3d kubeconfig가 KUBECONFIG에 포함되어 있어야 한다
- `make install` -> `make run` 순서를 지켜야 한다. Controller가 뜰 때 CRD가 있어야 Watch 등록이 된다

## 4단계: 테스트

`make run`은 터미널을 점유하므로, **다른 터미널을 열어서** 테스트한다.

### 샘플 파일 위치

kubebuilder가 자동 생성한 샘플 파일: `config/samples/apps_v1_simpleapp.yaml`

```yaml
apiVersion: apps.example.com/v1
kind: SimpleApp
metadata:
  name: my-app
  namespace: default
spec:
  image: nginx:1.25
  replicas: 2
```

### CR 생성 및 확인

```bash
# SimpleApp CR 생성
kubectl apply -f config/samples/apps_v1_simpleapp.yaml

# CR 확인
kubectl get simpleapp

# Controller가 만든 Deployment 확인
kubectl get deploy

# Pod 확인 (replicas: 2이므로 2개 떠야 한다)
kubectl get pods
```

### 실행 결과 예시

```
# kubectl get simpleapp
NAME     AGE
my-app   0s

# kubectl get deploy
NAME     READY   UP-TO-DATE   AVAILABLE   AGE
my-app   2/2     2            2           20s

# kubectl get pods
NAME                      READY   STATUS    RESTARTS   AGE
my-app-6b444657f4-x59kb   1/1     Running   0          21s
my-app-6b444657f4-x9fcq   1/1     Running   0          21s
```

`make run` 터미널에는 다음 로그가 출력된다:
```
INFO    Deployment 생성    {"controller": "simpleapp", ..., "name": "my-app"}
```

### 추가 테스트 (아직 미실행)

```bash
# replicas 변경 테스트 -- Controller가 Deployment를 업데이트하는지
kubectl patch simpleapp my-app --type merge -p '{"spec":{"replicas":3}}'
kubectl get pods

# image 변경 테스트
kubectl patch simpleapp my-app --type merge -p '{"spec":{"image":"nginx:1.26"}}'
kubectl get deploy -o wide

# 자가 치유 테스트 -- Deployment를 직접 수정하면 Controller가 되돌리는지
kubectl scale deploy my-app --replicas=5
kubectl get pods -w

# 삭제 테스트 -- SimpleApp 삭제 시 Deployment도 같이 삭제되는지 (OwnerReference)
kubectl delete simpleapp my-app
kubectl get deploy
```
