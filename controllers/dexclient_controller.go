/*


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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/mikamai/dex-operator/dex"
	"github.com/mikamai/dex-operator/utils"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
)

// DexClientReconciler reconciles a DexClient object
type DexClientReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

func (r *DexClientReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dexclient", req.NamespacedName)

	log.Info("reconciling DexClient resource")
	var dc dexv1alpha1.DexClient
	if err := r.Get(ctx, req.NamespacedName, &dc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sel := dc.Spec.InstanceSelector
	if sel.MatchLabels == nil {
		sel.MatchLabels = make(map[string]string)
	}

	sel.MatchLabels[dex.ServiceMarkerLabel] = dex.ServiceMarkerApi
	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return ctrl.Result{}, err
	}

	var list v1.ServiceList
	err = r.Client.List(ctx, &list, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfter}, errors.Wrap(err, "failed to fetch Dex services list")
	}

	if len(list.Items) == 0 {
		return ctrl.Result{}, errors.New("no Dex services found")
	}

	finalizer := "clients.finalizers.dex.karavel.io"
	if dc.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(dc.ObjectMeta.Finalizers, finalizer) {
			dc.ObjectMeta.Finalizers = append(dc.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(ctx, &dc); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if utils.ContainsString(dc.ObjectMeta.Finalizers, finalizer) {
			for _, svc := range list.Items {

				// our finalizer is present, so lets handle any external dependency
				if err := dex.DeleteDexClient(ctx, log, &svc, &dc); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return ctrl.Result{RequeueAfter: requeueAfter}, err
				}
			}

			// remove our finalizer from the list and update it.
			dc.ObjectMeta.Finalizers = utils.PopString(dc.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), &dc); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	for _, svc := range list.Items {
		l := log.WithValues("dex", fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
		var secret string
		if !dc.Spec.Public {
			sec, s, err := dex.Secret(&dc)
			if err != nil {
				return ctrl.Result{}, err
			}
			seco := new(v1.Secret)
			seco.Name = sec.Name
			seco.Namespace = sec.Namespace
			_, err = ctrl.CreateOrUpdate(ctx, r.Client, seco, func() error {
				seco.Data = sec.Data
				seco.StringData = sec.StringData
				return controllerutil.SetControllerReference(&dc, seco, r.Scheme)
			})
			if err != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, errors.Wrap(err, "failed to reconcile ConfigMap")
			}
			secret = s
		}
		if err := dex.AssertDexClient(ctx, l, &svc, &dc, secret); err != nil {
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *DexClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dexv1alpha1.DexClient{}).
		Complete(r)
}
