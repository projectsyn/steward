package flux

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"k8s.io/apimachinery/pkg/api/errors"

	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSSHSecret creates a new SSH key if it doesn't exist already and returns the public key
func CreateSSHSecret(clientset *kubernetes.Clientset) (string, error) {
	secret, err := clientset.CoreV1().Secrets(synNamespace).Get(fluxSSHSecretName, metav1.GetOptions{})
	if err == nil {
		publicKey := secret.Data[fluxSSHPublicKey]
		return string(publicKey), nil
	} else if !errors.IsNotFound(err) {
		return "", err
	}

	klog.Info("No SSH secret found, generate new key")

	publicKey, privateKey, err := generateSSHKey()
	if err != nil {
		return "", err
	}
	klog.Infof("Public key: %v", publicKey)
	fluxSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fluxSSHSecretName,
			Labels: fluxLabels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"identity":       []byte(privateKey),
			fluxSSHPublicKey: []byte(publicKey),
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

	publicKey, err := extractPublicKey(privateKey)
	if err != nil {
		return "", "", err
	}

	return publicKey, string(pem.EncodeToMemory(pemPriv)), nil
}

func extractPublicKey(privateKey *rsa.PrivateKey) (string, error) {
	publicRsaKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return string(pubKeyBytes), nil
}
