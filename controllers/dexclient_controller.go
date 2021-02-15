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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
	"github.com/mikamai/dex-operator/dex"
	"github.com/mikamai/dex-operator/utils"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// DexClientReconciler reconciles a DexClient object
type DexClientReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;events,verbs=get;list;watch;create;update;patch

func (r *DexClientReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dexclient", req.NamespacedName)

	log.Info("reconciling DexClient resource")
	var dc dexv1alpha1.DexClient
	if err := r.Get(ctx, req.NamespacedName, &dc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if dc.CreationTimestamp.IsZero() && dc.Status.Phase != dexv1alpha1.PhaseInitialising {
		dc.Status.Phase = dexv1alpha1.PhaseInitialising
		dc.Status.Ready = false
		if err := r.Client.Status().Update(ctx, &dc); err != nil {
			return r.ManageError(ctx, &dc, err)
		}
	}

	sel := dc.Spec.InstanceSelector.DeepCopy()
	if sel.MatchLabels == nil {
		sel.MatchLabels = make(map[string]string)
	}

	sel.MatchLabels[dex.ServiceMarkerLabel] = dex.ServiceMarkerApi
	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return r.ManageError(ctx, &dc, err)
	}

	var list v1.ServiceList
	err = r.Client.List(ctx, &list, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return r.ManageError(ctx, &dc, errors.Wrap(err, "failed to fetch Dex services list"))
	}

	finalizer := "clients.finalizers.dex.karavel.io"
	if dc.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(dc.ObjectMeta.Finalizers, finalizer) {
			log.Info("adding finalizer")
			controllerutil.AddFinalizer(&dc, finalizer)
			if err := r.Update(ctx, &dc); err != nil {
				return r.ManageError(ctx, &dc, err)
			}
		}
	} else {
		if utils.ContainsString(dc.ObjectMeta.Finalizers, finalizer) {
			for _, svc := range list.Items {

				// our finalizer is present, so lets handle any external dependency
				op, err := dex.DeleteDexClient(ctx, log, &svc, &dc)
				if err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return r.ManageError(ctx, &dc, err)
				}
				if op == dex.OpDeleted {
					r.Recorder.Eventf(&dc, "Normal", "Deleted", "deleted client from Dex instance")
				}
			}

			// remove our finalizer from the list and update it.
			log.Info("removing finalizer")
			controllerutil.RemoveFinalizer(&dc, finalizer)
			if err := r.Update(context.Background(), &dc); err != nil {
				return r.ManageError(ctx, &dc, err)
			}
		}

		// Stop reconciliation as the item is being deleted
		return r.ManageSuccess(ctx, &dc)
	}

	if len(list.Items) == 0 {
		return r.ManageError(ctx, &dc, errors.New("no matching Dex instances found"))
	}

	var secret string
	if !dc.Spec.Public {
		sec, s, err := dex.Secret(&dc)
		if err != nil {
			return r.ManageError(ctx, &dc, err)
		}
		log.Info("reconciling Secret")
		seco := new(v1.Secret)
		seco.Name = sec.Name
		seco.Namespace = sec.Namespace
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, seco, func() error {
			seco.Data = sec.Data
			return controllerutil.SetControllerReference(&dc, seco, r.Scheme)
		})
		if err != nil {
			return r.ManageError(ctx, &dc, errors.Wrap(err, "failed to reconcile Secret"))
		}
		secret = s
	}

	for _, svc := range list.Items {
		start := metav1.Now()
		instance := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		l := log.WithValues("dex", instance)

		op, err := dex.AssertDexClient(ctx, l, &svc, &dc, secret)
		if err != nil {
			return r.ManageError(ctx, &dc, err)
		}
		if op == dex.OpCreated {
			r.Recorder.PastEventf(&dc, start, "Normal", "Creating", "Creating on Dex instance %s", instance)
			r.Recorder.Eventf(&dc, "Normal", "Created", "Created on Dex instance %s", instance)
		}
	}

	return r.ManageSuccess(ctx, &dc)
}

func (r *DexClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dexv1alpha1.DexClient{}).
		Owns(&v1.Secret{}).
		Complete(r)
}

func (r *DexClientReconciler) ManageSuccess(ctx context.Context, client *dexv1alpha1.DexClient) (ctrl.Result, error) {
	if client.Status.Phase != dexv1alpha1.PhaseActive {
		client.Status.Message = "active"
		client.Status.Ready = true
		client.Status.Phase = dexv1alpha1.PhaseActive
		client.Status.ClientID = client.ClientID()

		if err := r.Client.Status().Update(ctx, client); err != nil {
			return ctrl.Result{
				RequeueAfter: requeueAfterError,
			}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *DexClientReconciler) ManageError(ctx context.Context, client *dexv1alpha1.DexClient, issue error) (ctrl.Result, error) {
	client.Status.Message = issue.Error()
	client.Status.Ready = false
	client.Status.Phase = dexv1alpha1.PhaseFailing
	client.Status.ClientID = ""
	r.Recorder.Event(client, "Warning", "Error", issue.Error())

	return ctrl.Result{
		RequeueAfter: requeueAfterError,
	}, r.Client.Status().Update(ctx, client)
}
