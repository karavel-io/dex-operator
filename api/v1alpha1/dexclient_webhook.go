/*
Copyright 2021 © MIKAMAI s.in.l

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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dexclientlog = logf.Log.WithName("dexclient-resource")

func (in *DexClient) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// Change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-dex-karavel-io-v1alpha1-dexclient,mutating=false,failurePolicy=fail,groups=dex.karavel.io,resources=dexclients,versions=v1alpha1,name=vdexclient.kb.io

var _ webhook.Validator = &DexClient{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *DexClient) ValidateCreate() error {
	dexclientlog.Info("validate create", "name", in.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *DexClient) ValidateUpdate(old runtime.Object) error {
	dexclientlog.Info("validate update", "name", in.Name)
	gr := schema.GroupResource{
		Group:    "dex.karavel.io",
		Resource: "dexclients",
	}

	dco := old.(*DexClient)
	diff := in.Spec.InstanceRef.Name != dco.Spec.InstanceRef.Name
	diff = diff || in.Spec.InstanceRef.Namespace != dco.Spec.InstanceRef.Namespace
	if diff {
		return apierrors.NewConflict(gr, in.Name, errors.New("field spec.instanceSelector is immutable"))
	}

	if in.Spec.Public != dco.Spec.Public {
		return apierrors.NewConflict(gr, in.Name, errors.New("field spec.public is immutable"))
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *DexClient) ValidateDelete() error {
	dexclientlog.Info("validate delete", "name", in.Name)
	return nil
}
