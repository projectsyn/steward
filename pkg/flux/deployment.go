package flux

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"git.vshn.net/syn/steward/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createFluxDeployment(gitInfo *api.GitInfo, clientset *kubernetes.Clientset, namespace, fluxImage string) error {
	mode := int32(0400)
	fluxDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "flux",
			Labels: fluxLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: fluxLabels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: fluxLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "flux",
					Volumes: []corev1.Volume{{
						Name: "ssh-key",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  fluxSSHSecretName,
								DefaultMode: &mode,
							},
						},
					}, {
						Name: "ssh-config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: fluxSSHConfigMapName,
								},
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "flux",
						Image: fluxImage,
						Args: []string{
							"--git-url", gitInfo.URL,
							"--git-readonly",
							"--git-poll-interval=1m",
							"--git-path=manifests/flux/",
							"--sync-interval=1m",
							"--sync-state=secret",
							"--sync-garbage-collection",
							"--memcached-service=",
							"--registry-exclude-image=*",
							"--k8s-secret-name", fluxSSHSecretName,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: 3030,
						}},
						Env: []corev1.EnvVar{{
							Name:  "KUBECONFIG",
							Value: "/root/.kubectl/config",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "ssh-key",
							ReadOnly:  true,
							MountPath: "/etc/fluxd/ssh",
						}, {
							Name:      "ssh-config",
							ReadOnly:  true,
							MountPath: "/root/.ssh",
						}},
						ImagePullPolicy: corev1.PullAlways,
					}},
				},
			},
		},
	}

	_, err := clientset.AppsV1().Deployments("syn").Create(fluxDeployment)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Info("Update existing flux Deployment")
			_, err = clientset.AppsV1().Deployments(namespace).Update(fluxDeployment)
		}
	} else {
		klog.Info("Created new flux Deployment")
	}
	return err
}
