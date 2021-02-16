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
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dexlog = logf.Log.WithName("dex-resource")

const (
	DexDefaultImage = "quay.io/dexidp/dex:latest"
)

func (in *Dex) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-dex-karavel-io-v1alpha1-dex,mutating=true,failurePolicy=fail,groups=dex.karavel.io,resources=dices,verbs=create;update,versions=v1alpha1,name=mdex.kb.io

var _ webhook.Defaulter = &Dex{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Dex) Default() {
	dexlog.Info("default", "name", in.Name)

	if in.Spec.Image == "" {
		in.Spec.Image = DexDefaultImage
	}
}
