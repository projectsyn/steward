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

func createRepoServerDeployment(clientset *kubernetes.Clientset, namespace, argoImage string) error {
	name := "argocd-repo-server"
	labels := map[string]string{
		"app.kubernetes.io/component": "server",
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
				Name: "server",
				Port: 8081,
				TargetPort: intstr.IntOrString{
					IntVal: 8081,
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
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: "ssh-known-hosts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: argoSSHConfigMapName,
									},
								},
							},
						},
						corev1.Volume{
							Name: "tls-certs",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: argoTLSConfigMapName,
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "argocd-repo-server",
							Image: argoImage,
							Command: []string{
								"uid_entrypoint.sh",
								"argocd-repo-server",
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 8081,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name:      "ssh-known-hosts",
									MountPath: "/app/config/ssh",
								},
								corev1.VolumeMount{
									Name:      "tls-certs",
									MountPath: "/app/config/tls",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: 8081,
										},
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: 8081,
										},
									},
								},
								InitialDelaySeconds: 1,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}
	if _, err := clientset.CoreV1().Services(namespace).Create(service); err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Info("Argo CD repo-server service already exists")
		} else {
			return err
		}
	} else {
		klog.Info("Created Argo CD repo-server service")
	}
	if _, err := clientset.AppsV1().Deployments(namespace).Create(deployment); err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Info("Argo CD repo-server deployment already exists")
		} else {
			return err
		}
	} else {
		klog.Info("Created Argo CD repo-server deployment")
	}
	return nil
}
