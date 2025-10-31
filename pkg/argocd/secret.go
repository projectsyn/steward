package argocd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CreateArgoSecret creates a new secret for Argo CD
func CreateArgoSecret(ctx context.Context, clientset kubernetes.Interface, namespace, password string) error {
	// bcrypt supports a maximum of 72 bytes for the password
	// https://cs.opensource.google/go/x/crypto/+/bc7d1d1eb54b3530da4f5ec31625c95d7df40231
	if len(password) > 72 {
		password = password[:72]
	}
	pwHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	mtime := time.Now().Format(time.RFC3339)
	if err != nil {
		return err
	}
	clusterSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoClusterSecretName, metav1.GetOptions{})
	if err == nil {
		// If the operator-managed cluster secret exists, the password is updated there instead
		currentPw := clusterSecret.Data["admin.password"]
		if bytes.Compare(currentPw, []byte(password)) == 0 {
			return nil
		}
		clusterSecretApply, err := corev1.ExtractSecret(clusterSecret, fieldManager)
		if err != nil {
			return err
		}
		clusterSecretApply.WithData(
			map[string][]byte{
				"admin.password":      []byte(password),
				"admin.passwordMtime": []byte(mtime),
			},
		)
		_, err = clientset.CoreV1().Secrets(namespace).Apply(ctx, clusterSecretApply, metav1.ApplyOptions{FieldManager: fieldManager, Force: true})
		if err != nil {
			return err
		}
		klog.Info("Argo CD Cluster secret updated with new password")
		return nil
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, argoSecretName, metav1.GetOptions{})
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	argoSecret := corev1.Secret(argoSecretName, namespace)
	infoMsg := "Created new Argo CD secret"
	secretApplyOpts := applyOpts
	if err == nil {
		currentPwHash := secret.Data["admin.password"]
		err = bcrypt.CompareHashAndPassword(currentPwHash, []byte(password))
		if err == nil {
			return nil
		}
		argoSecret, err = corev1.ExtractSecret(secret, fieldManager)
		if err != nil {
			return err
		}
		infoMsg = "Argo CD secret updated with new password"
		secretApplyOpts = metav1.ApplyOptions{
			FieldManager: fieldManager,
			// We need to force the update to ensure the password
			// gets updated in all cases with server-side apply
			Force: true,
		}
	}

	argoSecret.WithData(
		map[string][]byte{
			"admin.password":      pwHashBytes,
			"admin.passwordMtime": []byte(mtime),
		},
	)
	_, err = clientset.CoreV1().Secrets(namespace).Apply(ctx, argoSecret, secretApplyOpts)
	if err != nil {
		return err
	}
	klog.Info(infoMsg)
	return nil
}

// CreateSSHSecret creates a new SSH key if it doesn't exist already and returns the public key
func CreateSSHSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) (string, error) {
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
	sshSecret := corev1.Secret(argoSSHSecretName, namespace)
	sshSecret.WithData(
		map[string][]byte{
			argoSSHPrivateKey: []byte(privateKey),
			argoSSHPublicKey:  []byte(publicKey),
		},
	)

	_, err = clientset.CoreV1().Secrets(namespace).Apply(ctx, sshSecret, applyOpts)
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
