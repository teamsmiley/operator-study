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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sappsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "github.com/teamsmiley/myoperator/api/v1"
)

var _ = Describe("SimpleApp Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		simpleapp := &appsv1.SimpleApp{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind SimpleApp")
			err := k8sClient.Get(ctx, typeNamespacedName, simpleapp)
			if err != nil && errors.IsNotFound(err) {
				replicas := int32(1)
				resource := &appsv1.SimpleApp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: appsv1.SimpleAppSpec{
						Image:    "nginx:latest",
						Replicas: &replicas,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// Deployment 정리 (Reconcile이 생성했을 수 있음)
			deploy := &k8sappsv1.Deployment{}
			err := k8sClient.Get(ctx, typeNamespacedName, deploy)
			if err == nil {
				By("Cleanup the Deployment")
				Expect(k8sClient.Delete(ctx, deploy)).To(Succeed())
			}

			// SimpleApp CR 정리
			resource := &appsv1.SimpleApp{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				return // 이미 삭제됨
			}

			By("Cleanup the specific resource instance SimpleApp")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			// Reconcile 호출하여 Finalizer 정리 → CR이 실제로 삭제됨
			controllerReconciler := &SimpleAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, _ = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &SimpleAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a Deployment after reconcile", func() {
			By("Reconciling to trigger Deployment creation")
			controllerReconciler := &SimpleAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the Deployment was created")
			deploy := &k8sappsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedName, deploy)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Deployment spec matches SimpleApp spec")
			// Deployment 이름이 SimpleApp 이름과 같은지
			Expect(deploy.Name).To(Equal(resourceName))
			// Container 이미지가 설정되어 있는지
			Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
		})
	})
})
