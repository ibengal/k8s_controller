package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodTTLReconciler struct {
	Client client.Client
}

func (r *PodTTLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return nil
}
