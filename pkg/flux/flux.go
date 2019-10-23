package flux

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"os"

	"git.vshn.net/syn/steward/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	synNamespace = "syn"
	fluxImage    = "docker.io/fluxcd/flux:1.15.0"
	fluxLabels   = map[string]string{"app": "flux", "app.kubernetes.io/managed-by": "syn-agent"}
)

// ApplyFlux reconciles the flux deployment
func ApplyFlux(ctx context.Context, gitInfo *api.GitInfo) error {
	kubecfg := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
	if kubecfg == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubecfg)
	}
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

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
	return bootstrapFlux(ctx, clientset, gitInfo)
}

func bootstrapFlux(ctx context.Context, clientset *kubernetes.Clientset, gitInfo *api.GitInfo) error {
	_, err := clientset.CoreV1().Secrets(synNamespace).Get("flux-git-deploy", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info("No SSH secret found, generate new key")
			err = createSSHSecret(clientset)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	err = createKnownHostsConfigMap(gitInfo, clientset)
	if err != nil {
		return err
	}

	return createFluxDeployment(gitInfo, clientset)
}
