package flux

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"git.vshn.net/syn/steward/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	synNamespace         = "syn"
	fluxImage            = "docker.io/fluxcd/flux:1.15.0"
	fluxLabels           = map[string]string{"app": "flux", "app.kubernetes.io/managed-by": "syn-agent"}
	fluxSSHSecretName    = "flux-ssh-key"
	fluxSSHPublicKey     = "public_key"
	fluxSSHConfigMapName = "flux-ssh-config"
)

// ApplyFlux reconciles the flux deployment
func ApplyFlux(ctx context.Context, clientset *kubernetes.Clientset, apiClient *api.Client, gitInfo *api.GitInfo) error {
	pods, err := clientset.CoreV1().Pods(synNamespace).List(metav1.ListOptions{
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
	return bootstrapFlux(ctx, clientset, apiClient, gitInfo)
}

func bootstrapFlux(ctx context.Context, clientset *kubernetes.Clientset, apiClient *api.Client, gitInfo *api.GitInfo) error {
	err := createKnownHostsConfigMap(gitInfo, clientset)
	if err != nil {
		return err
	}
	err = createRBAC(clientset)
	if err != nil {
		return err
	}
	return createFluxDeployment(gitInfo, clientset)
}
