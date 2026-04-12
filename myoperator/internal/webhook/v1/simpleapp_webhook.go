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
	"context"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/teamsmiley/myoperator/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var simpleapplog = logf.Log.WithName("simpleapp-resource")

// SetupSimpleAppWebhookWithManager registers the webhook for SimpleApp in the manager.
func SetupSimpleAppWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &appsv1.SimpleApp{}).
		WithDefaulter(&SimpleAppCustomDefaulter{}).
		WithValidator(&SimpleAppCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-apps-example-com-v1-simpleapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=simpleapps,verbs=create;update,versions=v1,name=msimpleapp-v1.kb.io,admissionReviewVersions=v1

// SimpleAppCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind SimpleApp when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type SimpleAppCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default -- Mutating Webhook. CR이 저장되기 전에 기본값을 자동으로 채운다.
func (d *SimpleAppCustomDefaulter) Default(_ context.Context, obj *appsv1.SimpleApp) error {
	simpleapplog.Info("Defaulting for SimpleApp", "name", obj.GetName())

	// image에 태그가 없으면 :latest 추가
	if !strings.Contains(obj.Spec.Image, ":") {
		obj.Spec.Image = obj.Spec.Image + ":latest"
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-apps-example-com-v1-simpleapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=simpleapps,verbs=create;update,versions=v1,name=vsimpleapp-v1.kb.io,admissionReviewVersions=v1

// SimpleAppCustomValidator struct is responsible for validating the SimpleApp resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SimpleAppCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type SimpleApp.
func (v *SimpleAppCustomValidator) ValidateCreate(_ context.Context, obj *appsv1.SimpleApp) (admission.Warnings, error) {
	simpleapplog.Info("Validation for SimpleApp upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type SimpleApp.
func (v *SimpleAppCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *appsv1.SimpleApp) (admission.Warnings, error) {
	simpleapplog.Info("Validation for SimpleApp upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type SimpleApp.
func (v *SimpleAppCustomValidator) ValidateDelete(_ context.Context, obj *appsv1.SimpleApp) (admission.Warnings, error) {
	simpleapplog.Info("Validation for SimpleApp upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
