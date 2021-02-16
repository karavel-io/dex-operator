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
	"github.com/go-logr/logr"
	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
	"github.com/mikamai/dex-operator/dex"
	"github.com/mikamai/dex-operator/utils"
	v1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
// +kubebuilder:rbac:groups="",resources=secrets;events,verbs=get;list;watch;create;update;patch;delete

func (r *DexClientReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dexclient", req.NamespacedName)

	log.Info("Reconciling DexClient resource")
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

	var d dexv1alpha1.Dex
	k := types.NamespacedName{
		Name:      dc.Spec.InstanceRef.Name,
		Namespace: dc.Spec.InstanceRef.Namespace,
	}
	if k.Namespace == "" {
		k.Namespace = dc.Namespace
	}
	if err := r.Client.Get(ctx, k, &d); err != nil && dc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.ManageError(ctx, &dc, err)
	}

	var svc v1.Service
	svk := types.NamespacedName{
		Name:      d.ServiceName(),
		Namespace: d.Namespace,
	}
	if err := r.Client.Get(ctx, svk, &svc); err != nil && dc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.ManageError(ctx, &dc, err)
	}

	log = log.WithValues("dex", k)

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
			if !svc.CreationTimestamp.IsZero() {
				// our finalizer is present, so lets handle any external dependency
				op, err := dex.DeleteDexClient(ctx, log, &svc, &dc)
				if err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return r.ManageError(ctx, &dc, err)
				}
				if op == dex.OpDeleted {
					r.Recorder.Eventf(&dc, v1.EventTypeNormal, "Deleted", "deleted client from Dex instance")
				}
			}

			// remove our finalizer from the list and update it.
			log.Info("Removing finalizer")
			controllerutil.RemoveFinalizer(&dc, finalizer)
			if err := r.Update(context.Background(), &dc); err != nil {
				return r.ManageError(ctx, &dc, err)
			}
		}

		// Stop reconciliation as the item is being deleted
		return r.ManageSuccess(ctx, &dc)
	}

	sec, secret, err := dex.Secret(&dc)
	if err != nil {
		return r.ManageError(ctx, &dc, err)
	}

	log.Info("Reconciling Secret")
	seco := new(v1.Secret)
	seco.Name = sec.Name
	seco.Namespace = sec.Namespace
	sk := types.NamespacedName{
		Name:      sec.Name,
		Namespace: sec.Namespace,
	}
	err = r.Client.Get(ctx, sk, seco)
	if err != nil && !kuberrors.IsNotFound(err) {
		return r.ManageError(ctx, &dc, err)
	}

	recreate := false
	if kuberrors.IsNotFound(err) {
		log.Info("Secret is missing, creating", "secret", seco.Name)
		seco.StringData = sec.StringData
		if err := controllerutil.SetControllerReference(&dc, seco, r.Scheme); err != nil {
			return r.ManageError(ctx, &dc, err)
		}

		if err := r.Client.Create(ctx, seco); err != nil {
			return r.ManageError(ctx, &dc, err)
		}

		recreate = true
	}

	start := metav1.Now()

	op, err := dex.AssertDexClient(ctx, log, &svc, &dc, secret, recreate)
	if err != nil {
		return r.ManageError(ctx, &dc, err)
	}
	if op == dex.OpCreated {
		r.Recorder.PastEventf(&dc, start, v1.EventTypeNormal, "Creating", "Creating on Dex instance %s", k)
		r.Recorder.Eventf(&dc, v1.EventTypeNormal, "Created", "Created on Dex instance %s", k)
	} else if op == dex.OpUpdated {
		r.Recorder.PastEventf(&dc, start, v1.EventTypeNormal, "Updating", "Updating on Dex instance %s", k)
		r.Recorder.Eventf(&dc, v1.EventTypeNormal, "Updated", "Updated on Dex instance %s", k)
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
	r.Recorder.Event(client, v1.EventTypeWarning, "Error", issue.Error())

	return ctrl.Result{
		RequeueAfter: requeueAfterError,
	}, r.Client.Status().Update(ctx, client)
}
