package argocd

import (
	"fmt"
	"net/url"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	repoString = `
- type: git
  url: %s
  sshPrivateKeySecret:
    name: %s
    key: %s
`
	pluginString = `
- name: kapitan
  generate:
    command: [kapitan, refs, --reveal, --refs-path, ../../refs/, --file, ./]
`
)

func createArgoCDConfigMaps(cluster *api.Cluster, clientset *kubernetes.Clientset, namespace string) error {
	if cluster.GitRepo == nil || cluster.GitRepo.Url == nil {
		return fmt.Errorf("No git repo information received from API for cluster '%s'", cluster.Id)
	}
	gitURL, err := url.Parse(*cluster.GitRepo.Url)
	if err != nil {
		return err
	}
	cmLabel := map[string]string{
		"app.kubernetes.io/part-of": "argocd",
	}
	if cluster.GitRepo.HostKeys != nil {

		knownHostsConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:   argoSSHConfigMapName,
				Labels: cmLabel,
			},
			Data: map[string]string{
				"ssh_known_hosts": *cluster.GitRepo.HostKeys,
			},
		}
		if err := createOrUpdateConfigMap(clientset, namespace, knownHostsConfigMap); err != nil {
			return nil
		}
	}
	tlsConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   argoTLSConfigMapName,
			Labels: cmLabel,
		},
	}
	rbacConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   argoRbacConfigMapName,
			Labels: cmLabel,
		},
	}
	argoConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   argoConfigMapName,
			Labels: cmLabel,
		},
		Data: map[string]string{
			"repositories":                 fmt.Sprintf(repoString, gitURL, argoSSHSecretName, argoSSHPrivateKey),
			"configManagementPlugins":      pluginString,
			"application.instanceLabelKey": "argocd.argoproj.io/instance",
		},
	}

	if err := createOrUpdateConfigMap(clientset, namespace, tlsConfigMap); err != nil {
		return nil
	}
	if err := createOrUpdateConfigMap(clientset, namespace, rbacConfigMap); err != nil {
		return nil
	}
	if err := createOrUpdateConfigMap(clientset, namespace, argoConfigMap); err != nil {
		return nil
	}
	return nil
}

func createOrUpdateConfigMap(clientset *kubernetes.Clientset, namespace string, configMap *corev1.ConfigMap) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(configMap)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Infof("Update existing ConfigMap %s", configMap.Name)
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(configMap)
		}
		return err
	}
	klog.Infof("Created new ConfigMap")
	return nil
}
