package flux

import (
	"fmt"
	"strings"

	"git.vshn.net/syn/steward/pkg/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createKnownHostsConfigMap(gitInfo *api.GitInfo, clientset *kubernetes.Clientset, namespace string) error {
	var knownHosts strings.Builder
	for _, key := range gitInfo.HostKeys {
		for k, v := range key {
			if _, err := fmt.Fprintf(&knownHosts, "%v %v %v\n", gitInfo.HostName, k, v); err != nil {
				return err
			}
		}
	}
	fluxConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fluxSSHConfigMapName,
			Labels: fluxLabels,
		},
		Data: map[string]string{
			"known_hosts": knownHosts.String(),
		},
	}
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(fluxConfigMap)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Info("Update existing KnownHosts ConfigMap")
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(fluxConfigMap)
		}
		return err
	}

	klog.Infof("Created new KnownHosts ConfigMap")

	return nil
}
