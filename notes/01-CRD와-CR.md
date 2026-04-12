# 01. CRD와 CR -- kubebuilder 없이 직접 해보기

## 목표

kubebuilder 없이, YAML 두 개만으로 CRD와 CR의 관계를 이해한다.

## CRD란?

Kubernetes에는 원래 정해진 리소스들이 있다:

- Pod, Deployment, Service, ConfigMap 등

CRD(Custom Resource Definition)는 **"새로운 종류의 리소스를 추가로 등록"**하는 것이다.

비유: 학교에 원래 "교실", "급식실", "도서관"이 있다.
CRD는 **"간식 신청"이라는 새로운 종류를 학교 시스템에 등록**하는 것이다.

## CR이란?

CRD로 종류를 등록한 후, 그 종류의 **실제 인스턴스**를 만드는 것이 CR(Custom Resource)이다.

비유: "간식 신청"이라는 종류를 등록한 후, **"초코파이 30개 신청서"를 실제로 제출**하는 것이다.

```text
CRD = "간식 신청이라는 종류가 있다" (종류 등록)
CR  = "초코파이 30개를 신청합니다" (실제 신청서)
```

## 실습 파일

`examples/` 폴더에 2개 파일이 있다. kubebuilder 없이 직접 작성한 것이다.

### crd.yaml -- 종류 등록 (설계도)

```yaml
# "Snack이라는 리소스 종류가 있다"를 Kubernetes에 등록
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: snacks.school.example.com
spec:
  group: school.example.com # API 그룹
  names:
    kind: Snack # YAML에서 쓰는 이름
    plural: snacks # kubectl get snacks
    singular: snack
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                menu: # 간식 메뉴
                  type: string
                quantity: # 수량
                  type: integer
```

### cr.yaml -- 실제 신청서

```yaml
# "초코파이 30개를 신청합니다"
apiVersion: school.example.com/v1
kind: Snack
metadata:
  name: afternoon-snack
  namespace: default
spec:
  menu: chocopie
  quantity: 30
```

## 실습 순서

### 1단계: CRD 없이 CR 먼저 apply (실패)

```bash
kubectl apply -f examples/cr.yaml
```

결과:

```
error: resource mapping not found for name: "afternoon-snack"
no matches for kind "Snack" in version "school.example.com/v1"
```

Kubernetes가 "Snack? 그런 리소스는 모른다"고 거부한다.
종류 등록(CRD)을 안 했으니 당연하다.

### 2단계: CRD 등록

```bash
kubectl apply -f examples/crd.yaml
```

결과:

```
customresourcedefinition.apiextensions.k8s.io/snacks.school.example.com created
```

이제 Kubernetes가 "Snack이라는 종류가 있다"를 알게 되었다.

### 3단계: CR 다시 apply (성공)

```bash
kubectl apply -f examples/cr.yaml
```

결과:

```
snack.school.example.com/afternoon-snack created
```

### 4단계: 확인

```bash
# CR 목록 확인
kubectl get snack

# CR 상세 확인
kubectl get snack afternoon-snack -o yaml
```

CR은 만들어졌지만, **아무 일도 일어나지 않는다.**
간식 신청서가 접수는 됐는데, 담당자(Controller)가 없으니 간식이 안 나온다.

## 정리

```text
1. CRD 없이 CR 생성 --> 실패 ("Snack이 뭔지 모름")
2. CRD 등록          --> "Snack이라는 종류가 있다"를 등록
3. CR 생성           --> "초코파이 30개 신청" 접수됨
4. 하지만             --> 아무 일도 안 일어남 (Controller가 없으니까)
```

| 구분       | 파일     | 비유                   | 하는 일                          |
| ---------- | -------- | ---------------------- | -------------------------------- |
| CRD        | crd.yaml | "간식 신청" 종류 등록  | Kubernetes가 Snack을 인식하게 함 |
| CR         | cr.yaml  | "초코파이 30개" 신청서 | etcd에 데이터로 저장됨           |
| Controller | (없음)   | 담당자 없음            | 아무도 신청서를 처리 안 함       |

## 청소

```bash
kubectl delete snack afternoon-snack
kubectl delete crd snacks.school.example.com
```

## 다음 단계

Controller를 추가하면 뭐가 달라지는지 확인한다.
