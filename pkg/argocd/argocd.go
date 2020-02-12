package argocd

import (
	"context"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	argoLabels = map[string]string{
		"app.kubernetes.io/part-of":  "argocd",
		"app.kubernetes.io/instance": "argocd",
	}
	argoSSHSecretName     = "argo-ssh-key"
	argoSSHPublicKey      = "sshPublicKey"
	argoSSHPrivateKey     = "sshPrivateKey"
	argoSSHConfigMapName  = "argocd-ssh-known-hosts-cm"
	argoTLSConfigMapName  = "argocd-tls-certs-cm"
	argoRbacConfigMapName = "argocd-rbac-cm"
	argoConfigMapName     = "argocd-cm"
	argoSecretName        = "argocd-secret"
	argoRbacName          = "argocd-application-controller"
	argoRootAppName       = "root"
	argoProjectName       = "syn"
	argoAppsPath          = "manifests/apps/"
	argoRedisImage        = "docker.io/redis:5.0.3"
)

// Apply reconciles the Argo CD deployments
func Apply(ctx context.Context, config *rest.Config, namespace, argoImage string, apiClient *api.Client, cluster *api.Cluster) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	deployments, err := clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})
	if err != nil {
		return err
	}
	expectedCount := 4
	foundCount := len(deployments.Items)
	if foundCount == expectedCount {
		klog.Infof("Found %d of %d deployments", foundCount, expectedCount)
		return nil
	}
	klog.Infof("Found %d of expected %d deployments, bootstrapping now", foundCount, expectedCount)
	return bootstrapArgo(ctx, clientset, config, namespace, argoImage, apiClient, cluster)
}

func bootstrapArgo(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace, argoImage string, apiClient *api.Client, cluster *api.Cluster) error {
	if err := createArgoCDConfigMaps(cluster, clientset, namespace); err != nil {
		return err
	}

	if err := createArgoCRDs(config); err != nil {
		return err
	}

	if err := createRedisDeployment(clientset, namespace, argoImage); err != nil {
		return err
	}

	if err := createRepoServerDeployment(clientset, namespace, argoImage); err != nil {
		return err
	}

	if err := createServerDeployment(clientset, namespace, argoImage); err != nil {
		return err
	}

	if err := createArgoProject(cluster, config, namespace); err != nil {
		return err
	}

	if err := createArgoApp(cluster, config, namespace); err != nil {
		return err
	}

	if err := createApplicationControllerDeployment(clientset, namespace, argoImage); err != nil {
		return err
	}

	return nil
}
