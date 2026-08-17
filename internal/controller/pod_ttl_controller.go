// Package controller implements the Pod TTL reconciler.
//
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;delete
package controller

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodTTLReconciler struct {
	Client                  client.Client
	MaxConcurrentReconciles int
}

func (r *PodTTLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var pod corev1.Pod
	if err := r.Client.Get(ctx, req.NamespacedName, &pod); err != nil { // why client and not ctrl isnt it in the informer already?

		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.FromContext(ctx).Error(err, "failed to get pod")
		return ctrl.Result{}, err
	}

	annotations := pod.GetAnnotations()

	ttlValue, exists := annotations["ttl"]

	if !exists {
		return ctrl.Result{}, nil
	}

	ttlValueParsed, err := time.ParseDuration(ttlValue)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to parse ttl duration")
		return ctrl.Result{}, nil // why do we return nil and not err here? 
	}

	currentTimestamp := time.Now()
	creationTime := pod.GetCreationTimestamp()

	expiry := creationTime.Add(ttlValueParsed)
	difference := expiry.Sub(currentTimestamp)

	if difference <= 0 {
		// delete the pod

		if err := r.Client.Delete(ctx, &pod); err != nil {
			log.FromContext(ctx).Error(err, "failed to delete pod")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// requeue for later
	return ctrl.Result{RequeueAfter: difference}, nil
}

func (r *PodTTLReconciler) SetupWithManager(mgr ctrl.Manager) error {

	///add the optimization for not adding pods without ttl annotation

	if r.MaxConcurrentReconciles <= 0 {
		r.MaxConcurrentReconciles = 1
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.MaxConcurrentReconciles,
		}).
		Complete(r)
}
