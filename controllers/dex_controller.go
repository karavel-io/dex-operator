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
	"github.com/mikamai/dex-operator/dex"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
)

var (
	requeueAfterError = 30 * time.Second
)

// DexReconciler reconciles a Dex object
type DexReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dex.karavel.io,resources=dexes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events;configmaps;serviceaccounts;services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=dex.coreos.com,resources=*,verbs=*
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create

func (r *DexReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dex", req.NamespacedName)

	log.Info("Reconciling Dex resource")
	var d dexv1alpha1.Dex
	if err := r.Get(ctx, req.NamespacedName, &d); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	first := d.Status.Phase == dexv1alpha1.NoPhase
	if first {
		d.Status.Phase = dexv1alpha1.PhaseInitialising
		d.Status.Ready = false
		if err := r.Client.Status().Update(ctx, &d); err != nil {
			return r.ManageError(ctx, &d, err)
		}
	}

	start := metav1.Now()
	cm, err := dex.ConfigMap(&d)
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfterError}, err
	}
	cmo := new(v1.ConfigMap)
	cmo.Name = cm.Name
	cmo.Namespace = cm.Namespace
	log.Info("Reconciling ConfigMap", "name", cmo.Name, "namespace", cmo.Namespace)
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, cmo, func() error {
		cmo.Labels = cm.Labels
		cmo.Annotations = cm.Annotations
		cmo.Data = cm.Data
		return controllerutil.SetControllerReference(&d, cmo, r.Scheme)
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile ConfigMap"))
	}

	sa := dex.ServiceAccount(&d)
	sao := new(v1.ServiceAccount)
	sao.Name = sa.Name
	sao.Namespace = sa.Namespace
	log.Info("Reconciling ServiceAccount", "name", sao.Name, "namespace", sao.Namespace)
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, sao, func() error {
		sao.Annotations = sa.Annotations
		sao.Labels = sa.Labels
		return controllerutil.SetControllerReference(&d, sao, r.Scheme)
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile ServiceAccount"))
	}

	cr := dex.ClusterRole()
	cro := new(rbacv1.ClusterRole)
	cro.Name = cr.Name
	log.Info("Reconciling ClusterRole", "name", cro.Name)
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, cro, func() error {
		cro.Annotations = cr.Annotations
		cro.Labels = cr.Labels
		cro.Rules = cr.Rules
		return nil
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile ClusterRole"))
	}

	crb := dex.ClusterRoleBinding(&d, &sa, &cr)
	crbo := new(rbacv1.ClusterRoleBinding)
	crbo.Name = crb.Name
	log.Info("Reconciling ClusterRoleBinding", "name", crbo.Name)
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, crbo, func() error {
		crbo.Annotations = crb.Annotations
		crbo.Labels = crb.Labels
		crbo.Subjects = crb.Subjects
		crbo.RoleRef = crb.RoleRef
		return nil
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile ClusterRoleBinding"))
	}

	dep := dex.Deployment(&d, &cm, &sa)
	depo := new(appsv1.Deployment)
	depo.Name = dep.Name
	depo.Namespace = dep.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, depo, func() error {
		log.Info("Reconciling Deployment", "name", depo.Name, "namespace", depo.Namespace, "version", depo.ResourceVersion)
		if depo.CreationTimestamp.IsZero() {
			depo.Labels = dep.Labels
			depo.Spec.Selector = dep.Spec.Selector
			depo.Spec.Template.Labels = dep.Spec.Template.Labels
		}
		depo.Spec.Replicas = dep.Spec.Replicas
		depo.Spec.Template = dep.Spec.Template
		return controllerutil.SetControllerReference(&d, depo, r.Scheme)
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile Deployment"))
	}
	sel, err := metav1.LabelSelectorAsSelector(depo.Spec.Selector)
	if err != nil {
		return r.ManageError(ctx, &d, err)
	}

	d.Status.Selector = sel.String()
	d.Status.Replicas = *depo.Spec.Replicas

	svc := dex.Service(&d)
	svco := new(v1.Service)
	svco.Name = svc.Name
	svco.Namespace = svc.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, svco, func() error {
		log.Info("Reconciling Service", "name", svco.Name, "namespace", svco.Namespace, "version", svco.ResourceVersion)
		svco.Labels = svc.Labels
		if svco.CreationTimestamp.IsZero() {
			svco.Spec.Selector = svc.Spec.Selector
		}
		svco.Spec.Type = svc.Spec.Type
		svco.Spec.Ports = svc.Spec.Ports
		return controllerutil.SetControllerReference(&d, svco, r.Scheme)
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile Service"))
	}

	msvc := dex.MetricsService(&d)
	msvco := new(v1.Service)
	msvco.Name = msvc.Name
	msvco.Namespace = msvc.Namespace
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, msvco, func() error {
		log.Info("Reconciling Metrics Service", "name", msvco.Name, "namespace", msvco.Namespace, "version", msvco.ResourceVersion)
		msvco.Labels = msvc.Labels
		if msvco.CreationTimestamp.IsZero() {
			msvco.Spec.Selector = msvc.Spec.Selector
		}
		msvco.Spec.Type = msvc.Spec.Type
		msvco.Spec.Ports = msvc.Spec.Ports
		return controllerutil.SetControllerReference(&d, msvco, r.Scheme)
	})
	if err != nil {
		return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile metrics Service"))
	}

	ing, err := dex.Ingress(&d)
	if err != nil {
		return r.ManageError(ctx, &d, err)
	}
	ingo := new(networkingv1beta1.Ingress)
	ingo.Name = ing.Name
	ingo.Namespace = ing.Namespace
	if *d.Spec.Ingress.Enabled {
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, ingo, func() error {
			log.Info("Reconciling Ingress", "name", ingo.Name, "namespace", ingo.Namespace, "version", ingo.ResourceVersion)
			ingo.Labels = ing.Labels
			ingo.Annotations = ing.Annotations
			ingo.Spec = ing.Spec
			return controllerutil.SetControllerReference(&d, ingo, r.Scheme)
		})
		if err != nil {
			return r.ManageError(ctx, &d, errors.Wrap(err, "failed to reconcile Ingress"))
		}
	} else {
		log.Info("Removing Ingress", "name", ingo.Name, "namespace", ingo.Namespace, "version", ingo.ResourceVersion)
		if err := r.Client.Delete(ctx, ingo); err != nil && !kuberrors.IsNotFound(err) {
			return r.ManageError(ctx, &d, err)
		}
	}

	if first {
		r.Recorder.PastEventf(&d, start, v1.EventTypeNormal, "Creating", "Creating resources")
		r.Recorder.Event(&d, v1.EventTypeNormal, "Created", "Creating resources")
	}
	log.Info("Finished reconciling Dex resource")
	return r.ManageSuccess(ctx, &d)
}

func (r *DexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dexv1alpha1.Dex{}).
		Owns(&v1.ConfigMap{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&appsv1.Deployment{}).
		Owns(&v1.Service{}).
		Owns(&networkingv1beta1.Ingress{}).
		Complete(r)
}

func (r *DexReconciler) ManageSuccess(ctx context.Context, dex *dexv1alpha1.Dex) (ctrl.Result, error) {
	dex.Status.Message = "active"
	dex.Status.Ready = true
	dex.Status.Phase = dexv1alpha1.PhaseActive

	if err := r.Client.Status().Update(ctx, dex); err != nil {
		return ctrl.Result{
			RequeueAfter: requeueAfterError,
		}, err
	}
	return ctrl.Result{}, nil
}

func (r *DexReconciler) ManageError(ctx context.Context, dex *dexv1alpha1.Dex, issue error) (ctrl.Result, error) {
	dex.Status.Message = issue.Error()
	dex.Status.Ready = false
	dex.Status.Phase = dexv1alpha1.PhaseFailing

	r.Recorder.Event(dex, v1.EventTypeWarning, "Error", issue.Error())

	return ctrl.Result{
		RequeueAfter: requeueAfterError,
		Requeue:      true,
	}, r.Client.Status().Update(ctx, dex)
}
