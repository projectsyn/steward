package argocd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateArgoSecret creates a new secret for Argo CD
func CreateArgoSecret(ctx context.Context, config *rest.Config, namespace, password string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	pwHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	mtime := time.Now().Format(time.RFC3339)
	if err != nil {
		return err
	}
	pwHash := string(pwHashBytes)
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoSecretName, metav1.GetOptions{})
	if err == nil {
		clusterSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoClusterSecretName, metav1.GetOptions{})
		if err == nil {
			// If the operator-managed cluster secret exists, the password is updated there instead
			secret = clusterSecret
		}
		currentPwHash := secret.Data["admin.password"]
		err = bcrypt.CompareHashAndPassword(currentPwHash, []byte(password))
		if err == nil {
			return nil
		}
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData["admin.password"] = pwHash
		secret.StringData["admin.passwordMtime"] = mtime
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		klog.Info("Argo CD secret updated with new password")
		return nil

	} else if !k8serr.IsNotFound(err) {
		return err
	}
	argoSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: argoSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"admin.password":      pwHash,
			"admin.passwordMtime": mtime,
		},
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, argoSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	klog.Infof("Created new Argo CD secret")
	return err
}

// CreateSSHSecret creates a new SSH key if it doesn't exist already and returns the public key
func CreateSSHSecret(ctx context.Context, config *rest.Config, namespace string) (string, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoSSHSecretName, metav1.GetOptions{})
	if err == nil {
		publicKey := secret.Data[argoSSHPublicKey]
		return string(publicKey), nil
	} else if !k8serr.IsNotFound(err) {
		return "", err
	}

	klog.Info("No SSH secret found, generate new key")

	publicKey, privateKey, err := generateSSHKey()
	if err != nil {
		return "", err
	}
	klog.Infof("Public key: %v", publicKey)
	sshSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: argoSSHSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			argoSSHPrivateKey: []byte(privateKey),
			argoSSHPublicKey:  []byte(publicKey),
		},
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, sshSecret, metav1.CreateOptions{})
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
