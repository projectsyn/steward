package flux

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	"git.vshn.net/syn/steward/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplyFlux reconciles the flux deployment
func ApplyFlux(ctx context.Context, gitInfo *api.GitInfo) error {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	pods, err := clientset.CoreV1().Pods("syn").List(metav1.ListOptions{
		LabelSelector: "app=flux",
	})
	if err != nil {
		return err
	}
	if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				log.Infof("Found running flux pod: %v/%v", pod.Namespace, pod.Name)
				return nil
			}
			log.Warnf("Found non running flux pod: %v/%v (%v)", pod.Namespace, pod.Name, pod.Status.Phase)
		}
	}
	log.Info("No running flux pod found, bootstrapping now")
	return bootstrapFlux(ctx, clientset)
}

func bootstrapFlux(ctx context.Context, clientset *kubernetes.Clientset) error {
	return nil
}
