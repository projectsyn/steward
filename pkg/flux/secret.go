package flux

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createSSHSecret(clientset *kubernetes.Clientset) (string, error) {
	publicKey, privateKey, err := generateSSHKey()
	if err != nil {
		return publicKey, err
	}
	klog.Infof("Public key: %v", publicKey)
	fluxSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "flux-git-deploy",
			Labels: fluxLabels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"identity": privateKey,
		},
	}
	_, err = clientset.CoreV1().Secrets(synNamespace).Create(fluxSecret)
	if err != nil {
		return publicKey, err
	}
	return publicKey, nil
}

func generateSSHKey() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}

	priv := x509.MarshalPKCS1PrivateKey(privateKey)

	pemPriv := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: priv,
	}

	publicRsaKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return string(pubKeyBytes), string(pem.EncodeToMemory(pemPriv)), nil
}
