package argocd

import (
	"context"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createRedisDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, argoImage, redisImage string) error {
	name := "argocd-redis"
	labels := map[string]string{
		"app.kubernetes.io/component": "redis",
		"app.kubernetes.io/name":      name,
	}
	for k, v := range argoLabels {
		labels[k] = v
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": name,
			},
			Ports: []corev1.ServicePort{{
				Name: "redis",
				Port: 6379,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 6379,
				}},
			},
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "steward",
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "redis",
							Image: redisImage,
							Env: []corev1.EnvVar{
								{Name: "ALLOW_EMPTY_PASSWORD", Value: "yes"},
								{Name: "REDIS_AOF_ENABLED", Value: "no"},
								{Name: "REDIS_EXTRA_FLAGS", Value: "--save ''"},
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 6379,
								},
							},
						},
					},
				},
			},
		},
	}
	if _, err := clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Warning("Argo CD redis service already exists")
		} else {
			return err
		}
	} else {
		klog.Info("Created Argo CD redis service")
	}
	if _, err := clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Warning("Argo CD redis deployment already exists")
		} else {
			return err
		}
	} else {
		klog.Info("Created Argo CD redis deployment")
	}
	return nil
}
