package controller

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodTTLReconciler struct {
	Client client.Client
}

func (r *PodTTLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var pod *corev1.Pod
	if err := r.Client.Get(ctx, req.NamespacedName, pod); err != nil {

		if errors.Is(err, client.ErrNotFound) {
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

	duration, err := time.ParseDuration(ttlValue)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to parse ttl duration")
		return ctrl.Result{}, nil // why do we return nil and not err here?
	}

	currentTimestamp := time.Now()
	creationTime := pod.GetCreationTimestamp()

	if creationTime.Add(duration).Before(currentTimestamp) {
		// delete the pod
	} else {
		// requeue for later
	}

	return ctrl.Result{}, nil
}

func (r *PodTTLReconciler) SetupWithManager(mgr ctrl.Manager) error {

	///add the optimization for not adding pods without ttl annotation

	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Pod{}).Complete(r)
}
