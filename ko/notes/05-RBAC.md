# 05. RBAC -- 누가 뭘 할 수 있는지 정하는 규칙

## RBAC란?

**Role-Based Access Control**의 약자.
"누가(Subject), 무엇을(Resource), 어떻게(Verb) 할 수 있는지"를 정의하는 Kubernetes의 권한 시스템이다.

## 비유: 회사 사무실 출입 시스템

```text
사원증 (ServiceAccount)      → "나는 김개발이다"
출입 권한표 (Role)            → "3층 서버실: 출입/조회 가능, 수정 불가"
출입 권한 부여 (RoleBinding)  → "김개발에게 이 권한표를 준다"
```

사원증만 있으면 아무것도 못 한다.
권한표만 있으면 아무에게도 적용이 안 된다.
**둘을 연결(Binding)해야** 비로소 권한이 생긴다.

## 핵심 리소스 4개

### API Group이란?

Kubernetes 리소스는 **API Group**으로 분류된다.
rules에서 `apiGroups`를 지정할 때 이 그룹명을 쓴다.

| API Group | 리소스 예시 | apiGroups 값 |
| --------- | ----------- | ------------ |
| core (핵심) | Pod, Service, ConfigMap, Secret, Node | `""` (빈 문자열) |
| apps | Deployment, StatefulSet, DaemonSet | `"apps"` |
| batch | Job, CronJob | `"batch"` |
| rbac.authorization.k8s.io | Role, ClusterRole | `"rbac.authorization.k8s.io"` |
| 커스텀 (우리 Operator) | SimpleApp | `"apps.example.com"` |

핵심: **core API group은 이름이 없어서 빈 문자열 `""`로 표현한다.**
Pod, Service, ConfigMap 같은 가장 기본적인 리소스가 여기에 속한다.

```yaml
# core 리소스 (Pod, ConfigMap 등)에 접근할 때
apiGroups: [""]           # ← 빈 문자열 = core API group

# apps 그룹 (Deployment 등)에 접근할 때
apiGroups: ["apps"]

# 커스텀 리소스에 접근할 때
apiGroups: ["apps.example.com"]
```

### 1. Role (네임스페이스 범위)

특정 네임스페이스 안에서만 유효한 권한 묶음.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default # 이 네임스페이스에서만 유효
  name: pod-reader
rules:
  - apiGroups: [""] # core API group (Pod, Service 등)
    resources: ["pods"] # 대상 리소스
    verbs: ["get", "list", "watch"] # 허용 동작
```

### 2. ClusterRole (클러스터 전체 범위)

모든 네임스페이스에서 유효하거나, 네임스페이스가 없는 리소스(Node 등)에 사용.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: deployment-manager # namespace 필드가 없다!
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
```

### 3. RoleBinding (Role을 Subject에 연결)

"이 사람에게 이 권한을 준다"는 연결 고리.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: default
subjects: # 누구에게?
  - kind: ServiceAccount
    name: my-operator
    namespace: default
roleRef: # 어떤 권한을?
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

### 4. ClusterRoleBinding (ClusterRole을 Subject에 연결)

ClusterRole을 클러스터 전체에 걸쳐 부여.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: deployment-manager-binding
subjects:
  - kind: ServiceAccount
    name: my-operator
    namespace: myoperator-system
roleRef:
  kind: ClusterRole
  name: deployment-manager
  apiGroup: rbac.authorization.k8s.io
```

## 짝 관계: 권한 정의와 권한 부여는 항상 쌍으로 움직인다

| 권한 정의 (무엇을 허용?) | 권한 부여 (누구에게 연결?) | 적용 범위                         |
| ------------------------ | -------------------------- | --------------------------------- |
| Role                     | RoleBinding                | 특정 네임스페이스 1개 안에서만    |
| ClusterRole              | ClusterRoleBinding         | 클러스터 전체 (모든 네임스페이스) |

정리하면:

```text
Role  ──────────►  RoleBinding           (네임스페이스 범위)
  "무엇을 허용"       "누구에게 연결"

ClusterRole  ───►  ClusterRoleBinding    (클러스터 전체)
  "무엇을 허용"       "누구에게 연결"
```

주의: ClusterRole을 RoleBinding으로 연결할 수도 있다.
이 경우 클러스터 범위의 권한을 특정 네임스페이스에 한정해서 부여하는 효과가 생긴다.

| 조합                             | 의미                                                       |
| -------------------------------- | ---------------------------------------------------------- |
| Role + RoleBinding               | 가장 기본. 한 네임스페이스에서만 유효                      |
| ClusterRole + ClusterRoleBinding | 클러스터 전체에서 유효                                     |
| ClusterRole + RoleBinding        | ClusterRole의 권한을 특정 네임스페이스에만 한정 부여       |
| Role + ClusterRoleBinding        | 불가능 (Role은 네임스페이스 리소스라 클러스터 바인딩 불가) |

## Subject: 권한을 받는 주체 3종류

| Subject 종류   | 설명                                      | 예시                                      |
| -------------- | ----------------------------------------- | ----------------------------------------- |
| ServiceAccount | Pod에 부여되는 계정. Operator는 이걸 쓴다 | system:serviceaccount:default:my-operator |
| User           | 사람 사용자 (kubeconfig 인증서)           | kubectl 사용하는 개발자                   |
| Group          | 사용자 그룹                               | system:masters (관리자 그룹)              |

### ServiceAccount란?

사람이 kubectl로 접근할 때는 kubeconfig의 인증서(User)를 쓴다.
하지만 **Pod 안에서 실행되는 프로그램**(Operator 포함)은 사람이 아니다.
이런 프로그램에게 신분증을 주는 것이 ServiceAccount다.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-operator
  namespace: myoperator-system
```

이것만으로는 아무 권한이 없다. 빈 사원증일 뿐이다.
RoleBinding이나 ClusterRoleBinding으로 권한을 연결해야 의미가 생긴다.

Pod에서 ServiceAccount를 사용하려면 spec에 지정한다:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-operator-pod
spec:
  serviceAccountName: my-operator # 이 Pod는 my-operator 계정으로 동작
  containers:
    - name: operator
      image: my-operator:v0.1.0
```

전체 흐름을 정리하면:

```text
ServiceAccount 생성       "빈 사원증 발급"
       │
ClusterRole 생성          "권한표 작성"
       │
ClusterRoleBinding 생성   "사원증에 권한표 연결"
       │
Pod에 serviceAccountName  "사원증을 들고 출근"
       │
Pod 안의 프로그램이 API 호출 → kubelet이 ServiceAccount 토큰을 자동 주입
                               → API Server가 토큰 확인 → 권한 검사 → 허용/거부
```

## 자주 쓰는 Verbs

| Verb   | 의미      | HTTP 대응            |
| ------ | --------- | -------------------- |
| get    | 단일 조회 | GET /pods/nginx      |
| list   | 목록 조회 | GET /pods            |
| watch  | 변경 감시 | GET /pods?watch=true |
| create | 생성      | POST /pods           |
| update | 전체 수정 | PUT /pods/nginx      |
| patch  | 부분 수정 | PATCH /pods/nginx    |
| delete | 삭제      | DELETE /pods/nginx   |

## Operator와 RBAC의 관계

핵심: `make run` (로컬 실행)과 `make deploy` (클러스터 배포)에서 RBAC가 다르게 동작한다.

```text
로컬 실행 (make run)
  → kubeconfig의 admin 권한 사용
  → 뭐든 다 됨
  → RBAC 문제를 모른 채 넘어갈 수 있다

클러스터 배포 (make deploy)
  → ServiceAccount 권한만 사용
  → Role/ClusterRole에 정의된 것만 가능
  → 권한이 빠지면 "forbidden" 에러 발생!
```

그래서 Operator를 실제로 배포할 때 RBAC를 정확히 설정해야 한다.

## 다음 단계

우리 SimpleApp Operator에 RBAC가 어떻게 적용되어 있는지 실제 코드에서 확인한다.
(Go 코드의 marker 주석 → make manifests → YAML 자동 생성 흐름)
