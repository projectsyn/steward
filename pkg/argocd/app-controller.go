package argocd

import (
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createApplicationControllerDeployment(clientset *kubernetes.Clientset, namespace, argoImage string) error {
	name := "argocd-application-controller"
	labels := map[string]string{
		"app.kubernetes.io/component": "application-controller",
		"app.kubernetes.io/name":      name,
	}
	for k, v := range argoLabels {
		labels[k] = v
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
							Name:  name,
							Image: argoImage,
							Command: []string{
								"argocd-application-controller",
								"--status-processors",
								"20",
								"--operation-processors",
								"10",
								"--app-resync",
								"10",
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 8082,
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											IntVal: 8082,
										},
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											IntVal: 8082,
										},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
		},
	}

	_, err := clientset.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Warning("Argo CD application-controller already exists")
		} else {
			return err
		}
	} else {
		klog.Info("Created Argo CD application-controller deployment")
	}
	return nil
}
