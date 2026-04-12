/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	myappsv1 "github.com/teamsmiley/myoperator/api/v1"
)

// Finalizer 이름 -- 도메인/용도 형식이 관례
const simpleAppFinalizer = "apps.example.com/finalizer"

// SimpleAppReconciler reconciles a SimpleApp object
type SimpleAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=simpleapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile -- SimpleApp CR이 변경될 때마다 호출된다.
// 현재 상태를 원하는 상태로 맞추는 것이 이 함수의 역할이다.
func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. SimpleApp CR을 가져온다 (Desired State 확인)
	var app myappsv1.SimpleApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		if errors.IsNotFound(err) {
			log.Info("SimpleApp 리소스가 삭제됨, 무시")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Finalizer 처리 -- 삭제 요청이 들어왔는지 확인한다
	if !app.DeletionTimestamp.IsZero() {
		// 삭제가 요청됨 (deletionTimestamp가 찍힘)
		if controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
			// 정리 작업 실행 (실제 Operator에서는 여기서 외부 리소스를 정리한다)
			// 예: AWS S3 버킷 삭제, DNS 레코드 정리, 외부 API 호출 등
			log.Info("Finalizer 정리 작업 실행", "name", app.Name)

			// Finalizer 제거 → Kubernetes가 진짜 삭제를 진행한다
			controllerutil.RemoveFinalizer(&app, simpleAppFinalizer)
			if err := r.Update(ctx, &app); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Finalizer가 아직 없으면 추가한다 (최초 생성 시)
	if !controllerutil.ContainsFinalizer(&app, simpleAppFinalizer) {
		controllerutil.AddFinalizer(&app, simpleAppFinalizer)
		if err := r.Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. 이 SimpleApp에 대응하는 Deployment가 이미 있는지 확인한다
	var deploy appsv1.Deployment
	err := r.Get(ctx, req.NamespacedName, &deploy)

	if errors.IsNotFound(err) {
		// 3a. Deployment가 없으면 새로 만든다
		log.Info("Deployment 생성", "name", app.Name)
		deploy := r.buildDeployment(&app)

		// OwnerReference 설정 -- SimpleApp이 삭제되면 Deployment도 같이 삭제된다
		if err := ctrl.SetControllerReference(&app, deploy, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, deploy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// 3b. Deployment가 이미 있으면, spec이 변경되었는지 확인하고 업데이트한다
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
		log.Info("Deployment 업데이트", "name", app.Name, "image", app.Spec.Image, "replicas", replicas)
		if err := r.Update(ctx, &deploy); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. Status 업데이트 -- Deployment의 실제 상태를 SimpleApp CR에 기록한다
	if deploy.Status.AvailableReplicas == replicas {
		// 원하는 수만큼 Pod가 준비됨 → Available
		meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReady",
			Message:            "모든 Pod가 정상 동작 중",
			ObservedGeneration: app.Generation,
		})
		meta.RemoveStatusCondition(&app.Status.Conditions, "Progressing")
	} else {
		// 아직 Pod가 준비되지 않음 → Progressing
		meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
			Type:               "Progressing",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentUpdating",
			Message:            "Pod 배포 진행 중",
			ObservedGeneration: app.Generation,
		})
		meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             metav1.ConditionFalse,
			Reason:             "DeploymentUpdating",
			Message:            "Pod 배포 진행 중",
			ObservedGeneration: app.Generation,
		})
	}

	// Status subresource를 통해 업데이트 (.../simpleapps/my-app/status 경로)
	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildDeployment -- SimpleApp spec을 바탕으로 Deployment 오브젝트를 생성한다
func (r *SimpleAppReconciler) buildDeployment(app *myappsv1.SimpleApp) *appsv1.Deployment {
	replicas := int32(1)
	if app.Spec.Replicas != nil {
		replicas = *app.Spec.Replicas
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": app.Name,
					},
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

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappsv1.SimpleApp{}).
		Owns(&appsv1.Deployment{}). // Deployment 변경도 감지한다
		Named("simpleapp").
		Complete(r)
}
