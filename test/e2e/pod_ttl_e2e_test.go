package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Pod TTL controller", func() {
	const namespace = "default"

	Describe("when a pod has a ttl annotation", func() {
		It("should delete the pod after the ttl expires", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-ttl-expired",
					Namespace: namespace,
					Annotations: map[string]string{
						"ttl": "3s",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: "registry.k8s.io/pause:3.9",
						},
					},
				},
			}

			By("creating a pod with ttl=3s")
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("waiting for the controller to delete the pod")
			Eventually(func(g Gomega) {
				fetched := &corev1.Pod{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pod), fetched)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}, 30*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("when a pod has no ttl annotation", func() {
		It("should not delete the pod", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-no-ttl",
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pause",
							Image: "registry.k8s.io/pause:3.9",
						},
					},
				},
			}

			By("creating a pod without a ttl annotation")
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("waiting briefly and confirming the pod still exists")
			Consistently(func(g Gomega) {
				fetched := &corev1.Pod{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pod), fetched)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Name).To(Equal("e2e-no-ttl"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())

			// cleanup so later test runs don't collide on name
			Expect(k8sClient.Delete(ctx, pod)).To(Succeed())
		})
	})
})
