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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "github.com/teamsmiley/myoperator/api/v1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("SimpleApp Webhook", func() {
	var (
		obj       *appsv1.SimpleApp
		oldObj    *appsv1.SimpleApp
		defaulter SimpleAppCustomDefaulter
		validator SimpleAppCustomValidator
	)

	BeforeEach(func() {
		obj = &appsv1.SimpleApp{}
		oldObj = &appsv1.SimpleApp{}
		defaulter = SimpleAppCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		validator = SimpleAppCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating SimpleApp under Defaulting Webhook", func() {
		It("태그 없는 image에 :latest를 추가해야 한다", func() {
			obj.Spec.Image = "nginx"
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Image).To(Equal("nginx:latest"))
		})

		It("태그가 있는 image는 그대로 유지해야 한다", func() {
			obj.Spec.Image = "nginx:1.25"
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.Spec.Image).To(Equal("nginx:1.25"))
		})
	})

	Context("When creating SimpleApp under Validating Webhook", func() {
		It("태그가 있는 image는 생성을 허용해야 한다", func() {
			obj.Spec.Image = "nginx:1.25"
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("태그가 없는 image는 생성을 거부해야 한다", func() {
			obj.Spec.Image = "nginx"
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When updating SimpleApp under Validating Webhook", func() {
		It("태그가 있는 image로 수정은 허용해야 한다", func() {
			oldObj.Spec.Image = "nginx:1.24"
			obj.Spec.Image = "nginx:1.25"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("태그가 없는 image로 수정은 거부해야 한다", func() {
			oldObj.Spec.Image = "nginx:1.24"
			obj.Spec.Image = "nginx"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})
	})

})
