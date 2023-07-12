package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"go.uber.org/multierr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	argoClusterSecretName = "syn-argocd-cluster"
	argoRbacName          = "argocd-application-controller"
	argoRootAppName       = "root"
	argoProjectName       = "syn"
	argoAppsPath          = "manifests/apps/"
)

// Apply reconciles the Argo CD deployments
func Apply(ctx context.Context, config *rest.Config, namespace, operatorNamespace, argoImage, redisArgoImage string, apiClient *api.Client, cluster *api.Cluster) error {
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
		// An ArgoCD custom resource exists in our namespace
		err = fixArgoOperatorDeadlock(ctx, clientset, config, namespace, operatorNamespace)
		return fmt.Errorf("could not fix argocd operator deadlock: %w", err)
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})
	if err != nil {
		return fmt.Errorf("Could not list ArgoCD deployments: %w", err)
	}
	expectedDeploymentCount := 3
	foundDeploymentCount := len(deployments.Items)

	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})
	if err != nil {
		return fmt.Errorf("Could not list ArgoCD statefulsets: %w", err)
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

func fixArgoOperatorDeadlock(ctx context.Context, clientset *kubernetes.Clientset, config *rest.Config, namespace, operatorNamespace string) error {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=argocd",
	})

	if err != nil {
		return fmt.Errorf("Could not list ArgoCD config maps: %w", err)
	}

	if len(configmaps.Items) > 2 {
		// no restart required
		return nil
	}

	pods, err := clientset.CoreV1().Pods(operatorNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Could not list ArgoCD operator pods: %w", err)
	}
	
	for _, pod := range(pods.Items) {
		if pod.CreationTimestamp.Time.After(time.Now().Add(-10 * time.Minute)) {
			klog.Info("ArgoCD Operator pod was recently created, waiting to reboot...")
			return nil
		}
	}

	// if there still exists an argocd-secret not managed by the operator, clean it up:
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoSecretName, metav1.GetOptions{})
	if err != nil  && !errors.IsNotFound(err) {
		return fmt.Errorf("Could not get ArgoCD secret: %w", err)
	}

	if err == nil {
		if len(secret.ObjectMeta.OwnerReferences) == 0 {
			klog.Info("Deleting steward-managed ArgoCD secret")
			err := clientset.CoreV1().Secrets(namespace).Delete(ctx, argoSecretName, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("Could not delete steward-managed ArgoCD secret: %w", err)
			}
		}
	}

	klog.Info("Rebooting ArgoCD operator to resolve deadlock...")
	errors := []error{}
	for _, pod := range(pods.Items) {
		klog.Infof("Removing pod %s", pod.Name)
		err := clientset.CoreV1().Pods(operatorNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		errors = append(errors, err)
	}

	return multierr.Combine(errors ...)
}