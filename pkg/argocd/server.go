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

func createServerDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, argoImage string) error {
	name := "argocd-server"
	labels := map[string]string{
		"app.kubernetes.io/component": "server",
		"app.kubernetes.io/name":      name,
	}
	for k, v := range argoLabels {
		labels[k] = v
	}
	annotations := argoAnnotations
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
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
							Name: "static-files",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
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
					ServiceAccountName: "steward",
					Containers: []corev1.Container{
						corev1.Container{
							Name:  name,
							Image: argoImage,
							Command: []string{
								"argocd-server",
								"--staticassets",
								"/shared/app",
								"--insecure",
							},
							Ports: []corev1.ContainerPort{
								corev1.ContainerPort{
									ContainerPort: 8080,
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
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
								InitialDelaySeconds: 3,
								PeriodSeconds:       30,
							},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, createOpts)
	if err != nil {
		if k8serr.IsAlreadyExists(err) {
			klog.Warning("Argo CD server already exists")
			return nil
		}
		return err
	}
	klog.Info("Created Argo CD server deployment")
	return nil
}
