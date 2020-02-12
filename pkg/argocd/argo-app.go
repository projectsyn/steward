package argocd

import (
	"github.com/projectsyn/lieutenant-api/pkg/api"
	k8err "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	// Import embedded manifests
	_ "github.com/projectsyn/steward/pkg/manifests"
)

var (
	argoGroupVersion = schema.GroupVersion{
		Group:   "argoproj.io",
		Version: "v1alpha1",
	}

	argoAppGVR     = argoGroupVersion.WithResource("applications")
	argoProjectGVR = argoGroupVersion.WithResource("appprojects")

	localKubernetesAPI = "https://kubernetes.default.svc"
)

func createArgoProject(cluster *api.Cluster, config *rest.Config, namespace string) error {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	argoProjectClient := dynamicClient.Resource(argoProjectGVR)
	project := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": argoProjectGVR.Group + "/" + argoProjectGVR.Version,
			"kind":       "AppProject",
			"metadata": map[string]interface{}{
				"name": argoProjectName,
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

	if _, err = argoProjectClient.Namespace(namespace).Create(project, v1.CreateOptions{}); err != nil {
		if k8err.IsAlreadyExists(err) {
			klog.Info("Argo Project already exists, skip")
		} else {
			return err
		}
	} else {
		klog.Info("Argo Project created")
	}
	return nil
}

func createArgoApp(cluster *api.Cluster, config *rest.Config, namespace string) error {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	argoAppClient := dynamicClient.Resource(argoAppGVR)
	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": argoAppGVR.Group + "/" + argoAppGVR.Version,
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name": argoRootAppName,
			},
			"spec": map[string]interface{}{
				"project": argoProjectName,
				"source": map[string]interface{}{
					"repoURL":        *cluster.GitRepo.Url,
					"path":           argoAppsPath,
					"targetRevision": "HEAD",
				},
				"syncPolicy": map[string]interface{}{
					"automated": map[string]interface{}{
						"prune":    true,
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

	if _, err = argoAppClient.Namespace(namespace).Create(app, v1.CreateOptions{}); err != nil {
		if k8err.IsAlreadyExists(err) {
			klog.Info("Argo App already exists, skip")
		} else {
			return err
		}
	} else {
		klog.Info("Argo App created")
	}
	return nil
}
