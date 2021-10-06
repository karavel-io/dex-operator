/*
Copyright 2021 © MIKAMAI s.r.l

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

package v1alpha1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dexlog = logf.Log.WithName("dex-resource")

const (
	DexDefaultImage = "ghcr.io/dexidp/dex:latest"
)

func (in *Dex) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-dex-karavel-io-v1alpha1-dex,mutating=true,failurePolicy=fail,groups=dex.karavel.io,resources=dexes,verbs=create;update,versions=v1alpha1,name=mdex.kb.io

var _ webhook.Defaulter = &Dex{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Dex) Default() {
	dexlog.Info("default", "name", in.Name)

	if in.Spec.Image == "" {
		in.Spec.Image = DexDefaultImage
	}

	if in.Spec.ServiceAccountName == "" {
		in.Spec.ServiceAccountName = in.Name
	}
}

// Change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-dex-karavel-io-v1alpha1-dex,mutating=false,failurePolicy=fail,groups=dex.karavel.io,resources=dexes,versions=v1alpha1,name=vdex.kb.io

var _ webhook.Validator = &Dex{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Dex) ValidateCreate() error {
	dexlog.Info("validate create", "name", in.Name)
	gk := in.GroupVersionKind().GroupKind()
	errs := make([]*field.Error, 0)

	if err := in.validatePublicURL(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(gk, in.Name, errs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Dex) ValidateUpdate(old runtime.Object) error {
	dexlog.Info("validate create", "name", in.Name)
	gk := in.GroupVersionKind().GroupKind()
	errs := make([]*field.Error, 0)

	if err := in.validatePublicURL(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(gk, in.Name, errs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Dex) ValidateDelete() error {
	dexlog.Info("validate delete", "name", in.Name)
	return nil
}

func (in *Dex) validatePublicURL() *field.Error {
	u, err := url.Parse(in.Spec.PublicURL)
	if err != nil {
		return &field.Error{
			Type:     field.ErrorTypeInvalid,
			Field:    "publicURL",
			BadValue: in.Spec.PublicURL,
			Detail:   err.Error(),
		}
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return &field.Error{
			Type:     field.ErrorTypeNotSupported,
			Field:    "publicURL",
			BadValue: u.Scheme,
			Detail:   "must be one of http, https",
		}
	}

	return nil
}
