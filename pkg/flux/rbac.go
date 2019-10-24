package flux

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createRBAC(clientset *kubernetes.Clientset) error {
	err := createServiceAccount(clientset)
	if err != nil {
		return err
	}

	err = createClusterRoleBinding(clientset)
	if err != nil {
		return err
	}

	return nil
}

func createClusterRoleBinding(clientset *kubernetes.Clientset) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "flux",
			Labels: fluxLabels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "syn-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "flux",
			Namespace: synNamespace,
		}},
	}
	_, err := clientset.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Info("Update existing ClusterRoleBinding")
			_, err = clientset.RbacV1().ClusterRoleBindings().Update(clusterRoleBinding)
		}
		return err
	}

	klog.Infof("Created new ClusterRoleBinding")

	return nil
}

func createServiceAccount(clientset *kubernetes.Clientset) error {
	fluxServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flux",
			Namespace: synNamespace,
			Labels:    fluxLabels,
		},
	}
	_, err := clientset.CoreV1().ServiceAccounts(synNamespace).Create(fluxServiceAccount)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Info("Update existing ServiceAccount")
			_, err = clientset.CoreV1().ServiceAccounts(synNamespace).Update(fluxServiceAccount)
		}
		return err
	}

	klog.Infof("Created new ServiceAccount")

	return nil
}
