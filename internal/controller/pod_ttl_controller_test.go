package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPodTTLController(t *testing.T) {

	//setup

	tests := []struct {
		name        string
		pod         *corev1.Pod
		wantRequeue bool
		wantDeleted bool
	}{
		{
			name: "pod without ttl annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-no-ttl",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
				},
			},
			wantRequeue: false,
			wantDeleted: false,
		},
		{
			name: "pod with ttl not yet expired",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ttl-active",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-5 * time.Minute)),
					Annotations: map[string]string{
						"ttl": "1h",
					},
				},
			},
			wantRequeue: true,
			wantDeleted: false,
		},
		{
			name: "pod with ttl expired",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ttl-expired",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
					Annotations: map[string]string{
						"ttl": "30m",
					},
				},
			},
			wantRequeue: false,
			wantDeleted: true,
		},
		{
			name: "pod with invalid ttl annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ttl-invalid",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
					Annotations: map[string]string{
						"ttl": "not-a-duration",
					},
				},
			},
			wantRequeue: false,
			wantDeleted: false,
		},
		{
			name: "pod with zero ttl already expired",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ttl-zero",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					Annotations: map[string]string{
						"ttl": "0s",
					},
				},
			},
			wantRequeue: false,
			wantDeleted: true,
		},
	}

	//run

	scheme := runtime.NewScheme()
	//_ = corev1.AddToScheme(scheme)
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.pod).Build()
			reconciler := &PodTTLReconciler{
				Client: client,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.pod.Name,
					Namespace: tt.pod.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			if err != nil {
				t.Errorf("test %s failed: %v", tt.name, err)
				return
			}

			//assert

			if tt.wantRequeue && result.RequeueAfter <= 0 {
				t.Errorf("test %s failed: want requeue, got no requeue", tt.name)
			}

			if !tt.wantRequeue && result.RequeueAfter > 0 {
				t.Errorf("test %s failed: want no requeue, got requeue", tt.name)
			}

			err = client.Get(ctx, req.NamespacedName, &corev1.Pod{})

			if tt.wantDeleted && !apierrors.IsNotFound(err) {
				t.Errorf("test %s failed: want deleted, got found", tt.name)
			}

			if !tt.wantDeleted && err != nil {
				t.Errorf("test %s failed: want not deleted, got error", tt.name)
			}
		})
	}
}

func TestPodTTLReconciler_Reconcile_PodNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Empty store: no pods — Get will return NotFound.
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &PodTTLReconciler{Client: client}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "ghost",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected nil error when pod is not found, got %v", err)
	}

	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got RequeueAfter=%v", result.RequeueAfter)
	}

	if result.Requeue {
		t.Errorf("expected Requeue=false, got Requeue=true")
	}
}
