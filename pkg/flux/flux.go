package flux

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectsyn/steward/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	fluxLabels = map[string]string{
		"app":     "flux",
		"release": "flux",
	}
	fluxSSHSecretName    = "flux-ssh-key"
	fluxSSHPublicKey     = "public_key"
	fluxSSHConfigMapName = "flux-ssh-config"
)

// ApplyFlux reconciles the flux deployment
func ApplyFlux(ctx context.Context, clientset *kubernetes.Clientset, namespace, fluxImage string, apiClient *api.Client, gitInfo *api.GitInfo) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "app=flux",
	})
	if err != nil {
		return err
	}
	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				klog.Infof("Found running flux pod: %v/%v", pod.Namespace, pod.Name)
				return nil
			}
			klog.Warningf("Found non running flux pod: %v/%v (%v)", pod.Namespace, pod.Name, pod.Status.Phase)
		}
	}
	klog.Info("No running flux pod found, bootstrapping now")
	return bootstrapFlux(ctx, clientset, namespace, fluxImage, apiClient, gitInfo)
}

func bootstrapFlux(ctx context.Context, clientset *kubernetes.Clientset, namespace, fluxImage string, apiClient *api.Client, gitInfo *api.GitInfo) error {
	err := createKnownHostsConfigMap(gitInfo, clientset, namespace)
	if err != nil {
		return err
	}
	err = createRBAC(clientset, namespace)
	if err != nil {
		return err
	}
	return createFluxDeployment(gitInfo, clientset, namespace, fluxImage)
}
