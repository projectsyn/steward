package argocd

import (
	"context"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	argoLabels = map[string]string{
		"app.kubernetes.io/part-of":   "argocd",
		"argocd.argoproj.io/instance": "argocd",
		"steward.syn.tools/bootstrap": "true",
	}
	argoAnnotations = map[string]string{
		"argocd.argoproj.io/sync-options": "Prune=false",
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
)

// Apply reconciles the Argo CD deployments
func Apply(ctx context.Context, config *rest.Config, namespace, argoImage, redisArgoImage string, apiClient *api.Client, cluster *api.Cluster) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group: "argoproj.io",
		Version: "v1alpha1",
		Resource: "argocds",
	}

	argos, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if err == nil && len(argos.Items) > 0 {
		return nil
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})
	if err != nil {
		return err
	}
	expectedDeploymentCount := 3
	foundDeploymentCount := len(deployments.Items)

	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})
	if err != nil {
		return err
	}
	expectedStatefulSetCount := 1
	foundStatefulSetCount := len(statefulsets.Items)

	if foundDeploymentCount == expectedDeploymentCount && foundStatefulSetCount == expectedStatefulSetCount {
		// Found expected deployments, found expected statefulsets, skip
		return nil
	}

	klog.Infof("Found %d of expected %d deployments, found %d of expected %d statefulsets, bootstrapping now", foundDeploymentCount, expectedDeploymentCount, foundStatefulSetCount, expectedStatefulSetCount)
	return bootstrapArgo(ctx, clientset, config, namespace, argoImage, redisArgoImage, apiClient, cluster)
}

func bootstrapArgo(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace, argoImage, redisArgoImage string, apiClient *api.Client, cluster *api.Cluster) error {
	if err := createArgoCDConfigMaps(ctx, cluster, clientset, namespace); err != nil {
		return err
	}

	if err := createArgoCRDs(ctx, config); err != nil {
		return err
	}

	if err := createRedisDeployment(ctx, clientset, namespace, argoImage, redisArgoImage); err != nil {
		return err
	}

	if err := createRepoServerDeployment(ctx, clientset, namespace, argoImage); err != nil {
		return err
	}

	if err := createServerDeployment(ctx, clientset, namespace, argoImage); err != nil {
		return err
	}

	if err := createArgoProject(ctx, cluster, config, namespace); err != nil {
		return err
	}

	if err := createArgoApp(ctx, cluster, config, namespace); err != nil {
		return err
	}

	if err := createApplicationControllerStatefulSet(ctx, clientset, namespace, argoImage); err != nil {
		return err
	}

	return nil
}
