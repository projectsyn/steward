package argocd

import (
	"context"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	pluginString = `
- name: kapitan
  generate:
    command: [kapitan, refs, --reveal, --refs-path, ../../refs/, --file, ./]
`
)

func createArgoCDConfigMaps(ctx context.Context, cluster *api.Cluster, clientset *kubernetes.Clientset, namespace string) error {
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
		if err := createOrUpdateConfigMap(ctx, clientset, namespace, knownHostsConfigMap); err != nil {
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
			"configManagementPlugins":            pluginString,
			"application.instanceLabelKey":       "argocd.argoproj.io/instance",
			"application.resourceTrackingMethod": "label",
		},
	}

	if err := createOrUpdateConfigMap(ctx, clientset, namespace, tlsConfigMap); err != nil {
		return nil
	}
	if err := createOrUpdateConfigMap(ctx, clientset, namespace, rbacConfigMap); err != nil {
		return nil
	}
	if err := createOrUpdateConfigMap(ctx, clientset, namespace, argoConfigMap); err != nil {
		return nil
	}
	return nil
}

func createOrUpdateConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace string, configMap *corev1.ConfigMap) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, createOpts)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Infof("Update existing ConfigMap %s", configMap.Name)
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, updateOpts)
		}
		return err
	}
	klog.Infof("Created new ConfigMap")
	return nil
}
