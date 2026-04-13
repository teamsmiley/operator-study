# 06. RBAC 적용 -- Go marker에서 YAML 자동 생성까지

## 핵심 흐름

```text
Go 코드의 marker 주석       make manifests       config/rbac/ YAML 파일들
─────────────────────   ──────────────►   ──────────────────────────
+kubebuilder:rbac:...                     role.yaml (자동 생성)
```

kubebuilder는 Go 코드의 `+kubebuilder:rbac` 주석을 읽어서 RBAC YAML을 자동 생성한다.
직접 YAML을 편집할 필요가 없다.

## 우리 Operator의 marker 주석

`internal/controller/simpleapp_controller.go` 40~43번 줄:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

## marker 한 줄 = role.yaml의 rule 한 블록

| Go marker                                                                    | 의미                |
| ---------------------------------------------------------------------------- | ------------------- |
| `groups=apps.example.com,resources=simpleapps,verbs=get;list;...`            | SimpleApp CR을 CRUD |
| `groups=apps.example.com,resources=simpleapps/status,verbs=get;update;patch` | Status 읽기/수정    |
| `groups=apps.example.com,resources=simpleapps/finalizers,verbs=update`       | Finalizer 수정      |
| `groups=apps,resources=deployments,verbs=get;list;...`                       | Deployment를 CRUD   |

## 자동 생성되는 파일 3개와 연결 관계

| 단계 | 파일                               | 종류               | 역할                             |
| ---- | ---------------------------------- | ------------------ | -------------------------------- |
| 1    | `config/rbac/service_account.yaml` | ServiceAccount     | `controller-manager` 사원증 발급 |
| 2    | `config/rbac/role.yaml`            | ClusterRole        | marker에서 읽은 권한표           |
| 3    | `config/rbac/role_binding.yaml`    | ClusterRoleBinding | 1번과 2번을 연결                 |

연결 구조:

```text
service_account.yaml              role.yaml
  name: controller-manager          name: manager-role
  namespace: system                 rules: [marker에서 자동 생성]
        │                                  │
        └────── role_binding.yaml ─────────┘
                  subjects:
                    name: controller-manager   ← ServiceAccount
                  roleRef:
                    name: manager-role         ← ClusterRole
```

05번 노트에서 배운 RBAC 개념이 그대로 적용된 것이다:

- 빈 사원증 = `service_account.yaml`
- 권한표 = `role.yaml`
- 사원증에 권한표 연결 = `role_binding.yaml`

## 실제 파일 내용

### service_account.yaml

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: system
```

### role.yaml (make manifests로 자동 생성)

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
  - apiGroups:
      - apps
    resources:
      - deployments
    verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
  - apiGroups:
      - apps.example.com
    resources:
      - simpleapps
    verbs: ["create", "delete", "get", "list", "patch", "update", "watch"]
  - apiGroups:
      - apps.example.com
    resources:
      - simpleapps/finalizers
    verbs: ["update"]
  - apiGroups:
      - apps.example.com
    resources:
      - simpleapps/status
    verbs: ["get", "patch", "update"]
```

### role_binding.yaml

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: manager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: manager-role # ← role.yaml의 name
subjects:
  - kind: ServiceAccount
    name: controller-manager # ← service_account.yaml의 name
    namespace: system
```

## 그 외 자동 생성 RBAC 파일들

`config/rbac/kustomization.yaml`에서 관리:

| 파일                                | 용도                                           |
| ----------------------------------- | ---------------------------------------------- |
| `leader_election_role.yaml`         | 리더 선출용 권한 (여러 replica 중 하나만 동작) |
| `leader_election_role_binding.yaml` | 리더 선출 권한 연결                            |
| `metrics_auth_role.yaml`            | 메트릭 엔드포인트 인증                         |
| `metrics_auth_role_binding.yaml`    | 메트릭 인증 권한 연결                          |
| `metrics_reader_role.yaml`          | 메트릭 읽기 권한                               |
| `simpleapp_admin_role.yaml`         | 클러스터 관리자용 SimpleApp 전체 권한          |
| `simpleapp_editor_role.yaml`        | 편집자용 SimpleApp 읽기/쓰기 권한              |
| `simpleapp_viewer_role.yaml`        | 조회자용 SimpleApp 읽기 전용 권한              |

admin/editor/viewer 3개는 Operator 자체가 쓰는 것이 아니라,
kubectl 사용자에게 부여하기 위한 편의용 Role이다.

## 정리: 새 권한이 필요할 때의 작업 순서

```text
1. Go 코드에 marker 주석 추가
   // +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list

2. make manifests 실행
   → role.yaml에 새 rule이 자동 추가됨

3. make deploy (또는 make install && make run)
   → 새 권한이 클러스터에 적용됨
```

YAML을 직접 편집하지 않는다. 항상 Go marker가 진실의 원천(source of truth)이다.

## role.yaml의 각 rule이 의미하는 것

각 rule은 Reconcile 함수가 하는 일과 1:1로 대응된다:

| rule (apiGroups / resources) | 의미 | Reconcile에서 쓰는 곳 |
| ---------------------------- | ---- | --------------------- |
| `apps` / `deployments` / 전체 verb | Deployment 조회, 생성, 수정, 삭제 | `r.Get()`, `r.Create()`, `r.Update()` |
| `apps.example.com` / `simpleapps` / 전체 verb | SimpleApp CR 읽기/쓰기 | `r.Get()` (CR 조회) |
| `apps.example.com` / `simpleapps/finalizers` / update | Finalizer 필드 수정 | 아직 안 쓰지만 kubebuilder가 미리 생성 |
| `apps.example.com` / `simpleapps/status` / get,patch,update | Status 서브리소스 수정 | 아직 안 쓰지만 kubebuilder가 미리 생성 |

## 항상 이렇게 생겨야 하나?

아니다. Operator가 하는 일에 따라 달라져야 한다.

예시:

| Operator가 하는 일 | 필요한 추가 권한 |
| ------------------ | --------------- |
| ConfigMap을 읽어서 설정 적용 | `""` / `configmaps` / get,list,watch |
| Service를 생성 | `""` / `services` / get,create,update,delete |
| Pod를 직접 삭제 | `""` / `pods` / get,delete |
| Node 정보 조회 | `""` / `nodes` / get,list |

(`""`는 core API group이다. 자세한 설명은 05-RBAC.md 참고.)

불필요한 권한은 빼야 한다. 최소 권한 원칙(Principle of Least Privilege).

## marker 변경으로 권한 추가/수정하기

### marker 문법

```text
// +kubebuilder:rbac:groups=<API그룹>,resources=<리소스>,verbs=<동작들>
```

| 필드 | 값 | 예시 |
| ---- | -- | ---- |
| groups | API 그룹. core는 `""` | `""`, `apps`, `apps.example.com` |
| resources | 리소스 이름 (복수형) | `pods`, `deployments`, `configmaps` |
| verbs | 세미콜론(`;`)으로 구분 | `get;list;watch` |
| namespace | (선택) 붙이면 Role, 안 붙이면 ClusterRole | `namespace=default` |

### 예시: ConfigMap 읽기 권한 추가

Go 코드에 marker 한 줄 추가:

```go
// 기존 marker들...
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//                         ↑ core API group은 빈 문자열("")
```

`make manifests` 실행하면 `role.yaml`에 자동 추가:

```yaml
rules:
  # ...기존 rules...
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs: ["get", "list", "watch"]
```

### 권한 변경 작업 흐름

```text
1. Go marker 추가/수정/삭제
2. make manifests → role.yaml 자동 재생성
3. make deploy → 클러스터에 적용
```

## 파일별 생성 시점과 재생성 여부

| 파일                                | 생성 시점                | `make manifests`로 재생성?  |
| ----------------------------------- | ------------------------ | --------------------------- |
| `service_account.yaml`              | `kubebuilder init`       | 안 됨. 수동 편집 가능       |
| `role_binding.yaml`                 | `kubebuilder init`       | 안 됨. 수동 편집 가능       |
| `leader_election_role.yaml`         | `kubebuilder init`       | 안 됨                       |
| `leader_election_role_binding.yaml` | `kubebuilder init`       | 안 됨                       |
| `role.yaml`                         | `make manifests`         | 매번 Go marker에서 덮어쓰기 |
| `simpleapp_admin_role.yaml`         | `kubebuilder create api` | 안 됨                       |
| `simpleapp_editor_role.yaml`        | `kubebuilder create api` | 안 됨                       |
| `simpleapp_viewer_role.yaml`        | `kubebuilder create api` | 안 됨                       |

`role.yaml`만 `make manifests` 때마다 재생성된다.
나머지는 scaffold로 한 번 생성된 뒤, 필요하면 직접 수정하는 파일이다.

## controller-manager 이름을 바꾸려면?

`controller-manager`라는 이름은 여러 파일에 걸쳐 있다:

| 파일                                            | 바꿔야 하는 부분                            |
| ----------------------------------------------- | ------------------------------------------- |
| `config/rbac/service_account.yaml`              | `name: controller-manager`                  |
| `config/rbac/role_binding.yaml`                 | `subjects.name: controller-manager`         |
| `config/rbac/leader_election_role_binding.yaml` | `subjects.name: controller-manager`         |
| `config/rbac/metrics_auth_role_binding.yaml`    | `subjects.name: controller-manager`         |
| `config/manager/manager.yaml`                   | `serviceAccountName: controller-manager`    |
| `config/manager/manager.yaml`                   | `control-plane: controller-manager` (label) |

라벨 `control-plane: controller-manager`를 참조하는 파일도 함께 바꿔야 한다:

| 파일                                               | 이유                         |
| -------------------------------------------------- | ---------------------------- |
| `config/prometheus/monitor.yaml`                   | 메트릭 수집 대상 라벨 셀렉터 |
| `config/network-policy/allow-metrics-traffic.yaml` | 네트워크 정책 라벨 셀렉터    |
| `config/default/metrics_service.yaml`              | 서비스 라벨 셀렉터           |

한 곳이라도 빠지면 권한이 끊기거나 메트릭 수집이 안 된다.
특별한 이유가 없으면 kubebuilder가 생성한 이름을 그대로 쓰는 편이 안전하다.

## Operator 여러 개일 때 이름 충돌 방지

같은 클러스터에 Operator 여러 개를 배포하면 `controller-manager`가 충돌할 것 같지만,
kubebuilder는 kustomize로 이 문제를 해결한다.

`config/default/kustomization.yaml`의 핵심 두 줄:

```yaml
namespace: myoperator-system # 모든 리소스를 이 네임스페이스에 격리
namePrefix: myoperator- # 모든 리소스 이름 앞에 접두사 추가
```

`kustomize build`를 거치면 이름이 자동으로 변환된다:

| 원래 이름 (YAML)   | myoperator 배포 시               | youroperator 배포 시               |
| ------------------ | -------------------------------- | ---------------------------------- |
| namespace          | `myoperator-system`              | `youroperator-system`              |
| ServiceAccount     | `myoperator-controller-manager`  | `youroperator-controller-manager`  |
| ClusterRole        | `myoperator-manager-role`        | `youroperator-manager-role`        |
| ClusterRoleBinding | `myoperator-manager-rolebinding` | `youroperator-manager-rolebinding` |

YAML 파일에는 짧은 이름(`controller-manager`)으로 써놓지만,
실제 클러스터에 배포될 때는 접두사+네임스페이스로 격리되어 충돌하지 않는다.

이것이 kubebuilder가 kustomize를 쓰는 이유 중 하나다.

### 같은 네임스페이스에 여러 Operator를 배포하면?

namePrefix가 다르면 같은 네임스페이스 안에서도 충돌하지 않는다.

```text
myoperator-system 네임스페이스 안에:
  myoperator-controller-manager      (ServiceAccount)
  youroperator-controller-manager    (ServiceAccount)
  → 이름이 다르니까 문제없음
```

ClusterRole/ClusterRoleBinding은 네임스페이스 리소스가 아니라 클러스터 전체에 하나만 존재한다.
하지만 이것도 namePrefix 덕분에 이름이 다르고,
RBAC는 **합산(additive) 방식**이라 각 ClusterRole은 자기 ServiceAccount에만 바인딩되어 서로 간섭하지 않는다.
(권한을 "빼앗는" rule은 없고, 항상 "추가"만 한다.)

| 상황                               | 충돌? | 이유                                |
| ---------------------------------- | ----- | ----------------------------------- |
| 같은 네임스페이스, 다른 namePrefix | 없음  | 이름이 다름                         |
| ClusterRole 권한 범위 겹침         | 없음  | 각자 자기 ServiceAccount에만 바인딩 |
| namePrefix가 같은 두 Operator      | 충돌  | 이름이 완전히 같아짐                |

마지막 케이스만 주의하면 된다.
실무에서는 프로젝트 이름이 다르면 namePrefix도 다르니까 거의 일어나지 않는다.

## 다음 단계

Status subresource를 활용하여 CR의 현재 상태를 보여주는 패턴을 배운다.
