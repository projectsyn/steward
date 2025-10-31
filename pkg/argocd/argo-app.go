package argocd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/errors"
	k8err "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var (
	argoGroupVersion = schema.GroupVersion{
		Group:   "argoproj.io",
		Version: "v1alpha1",
	}

	argoAppGVR     = argoGroupVersion.WithResource("applications")
	argoProjectGVR = argoGroupVersion.WithResource("appprojects")

	localKubernetesAPI = "https://kubernetes.default.svc"

	additionalRootAppsConfigKey = "teams"
)

func readAdditionalRootAppsConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace, additionalRootAppsConfigMapName string) ([]string, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, additionalRootAppsConfigMapName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info("Additional root apps config map not present")
			return []string{}, nil
		} else {
			return nil, fmt.Errorf("unable to fetch the additional root apps config map: %w", err)
		}
	}
	teamsJson, ok := cm.Data[additionalRootAppsConfigKey]
	if !ok {
		return nil, fmt.Errorf("additional root apps ConfigMap doesn't have key %s", additionalRootAppsConfigKey)
	}
	var teams []string
	if err := json.Unmarshal([]byte(teamsJson), &teams); err != nil {
		return nil, fmt.Errorf("unmarshalling additional root apps ConfigMap contents: %v", err)
	}
	return teams, nil
}

func createArgoProject(ctx context.Context, cluster *api.Cluster, config *rest.Config, namespace, name string) error {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	argoProjectClient := dynamicClient.Resource(argoProjectGVR)

	if _, err = argoProjectClient.Namespace(namespace).Get(ctx, name, v1.GetOptions{}); err == nil {
		return nil
	}

	project := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": argoProjectGVR.Group + "/" + argoProjectGVR.Version,
			"kind":       "AppProject",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"clusterResourceWhitelist": []map[string]interface{}{{
					"group": "*",
					"kind":  "*",
				}},
				"destinations": []map[string]interface{}{{
					"namespace": "*",
					"server":    localKubernetesAPI,
				}},
				"sourceRepos": []string{
					*cluster.GitRepo.Url,
				},
			},
		},
	}

	if _, err = argoProjectClient.Namespace(namespace).Create(ctx, project, createOpts); err != nil {
		if k8err.IsAlreadyExists(err) {
			klog.Warning("Argo Project already exists, skipping... app=", name)
		} else {
			return err
		}
	} else {
		klog.Info("Argo Project created: ", name)
	}
	return nil
}

func createArgoApp(ctx context.Context, cluster *api.Cluster, config *rest.Config, namespace, projectName, name, appsPath string) error {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	argoAppClient := dynamicClient.Resource(argoAppGVR)

	if _, err = argoAppClient.Namespace(namespace).Get(ctx, name, v1.GetOptions{}); err == nil {
		return nil
	}

	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": argoAppGVR.Group + "/" + argoAppGVR.Version,
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"project": projectName,
				"source": map[string]interface{}{
					"repoURL":        *cluster.GitRepo.Url,
					"path":           appsPath + "/",
					"targetRevision": "HEAD",
				},
				"syncPolicy": map[string]interface{}{
					"automated": map[string]interface{}{
						"prune":    false,
						"selfHeal": true,
					},
				},
				"destination": map[string]interface{}{
					"namespace": namespace,
					"server":    localKubernetesAPI,
				},
			},
		},
	}

	if _, err = argoAppClient.Namespace(namespace).Create(ctx, app, createOpts); err != nil {
		if k8err.IsAlreadyExists(err) {
			klog.Warning("Argo App already exists, skipping... app=", name)
		} else {
			return err
		}
	} else {
		klog.Info("Argo App created: ", name)
	}
	return nil
}
