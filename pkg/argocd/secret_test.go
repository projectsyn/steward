package argocd

import (
	"context"
	"net/url"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectsyn/lieutenant-api/pkg/api"
)

func TestCreateArgoSecretCreate(t *testing.T) {
	fakeClient := fake.NewClientset()

	ctx := t.Context()

	err := CreateArgoSecret(ctx, fakeClient, "syn", "foo")
	require.NoError(t, err)

	validateSecret(t, ctx, fakeClient, argoSecretName)
}

func TestCreateArgoSecretUpdate(t *testing.T) {
	pwHashBytes, err := bcrypt.GenerateFromPassword([]byte("foo"), bcrypt.DefaultCost)
	require.NoError(t, err)

	cases := map[string]struct {
		secret  *corev1.Secret
		changed bool
	}{
		"empty": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{},
			},
			changed: true,
		},
		"empty password": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{
					"admin.password": []byte(""),
				},
			},
			changed: true,
		},
		"same password": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{
					"admin.password":      pwHashBytes,
					"admin.passwordMtime": []byte(time.Now().Format(time.RFC3339)),
				},
			},
			changed: false,
		},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			fakeClient := fake.NewClientset(tc.secret)

			ctx := t.Context()

			err := CreateArgoSecret(ctx, fakeClient, "syn", "foo")
			require.NoError(t, err)

			argoSecret, err := fakeClient.CoreV1().Secrets("syn").
				Get(ctx, argoSecretName, metav1.GetOptions{})
			require.NoError(t, err)

			found := false
			for _, mfs := range argoSecret.GetManagedFields() {
				if mfs.Manager == fieldManager {
					found = true
					break
				}
			}
			assert.Equalf(t, tc.changed, found, "Looking for field manager %q, should find: %v, found: %v", fieldManager, tc.changed, found)
			validateSecret(t, ctx, fakeClient, tc.secret.GetName())
		})
	}
}

func TestCreateArgoClusterSecretUpdate(t *testing.T) {
	cases := map[string]struct {
		secret  *corev1.Secret
		changed bool
	}{
		"empty": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoClusterSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{},
			},
			changed: true,
		},
		"empty password": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoClusterSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{
					"admin.password": []byte(""),
				},
			},
			changed: true,
		},
		"same password": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      argoClusterSecretName,
					Namespace: "syn",
				},
				Data: map[string][]byte{
					"admin.password":      []byte("foo"),
					"admin.passwordMtime": []byte(time.Now().Format(time.RFC3339)),
				},
			},
			changed: false,
		},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			fakeClient := fake.NewClientset(tc.secret)

			ctx := t.Context()

			err := CreateArgoSecret(ctx, fakeClient, "syn", "foo")
			require.NoError(t, err)

			argoSecret, err := fakeClient.CoreV1().Secrets("syn").
				Get(ctx, argoClusterSecretName, metav1.GetOptions{})
			require.NoError(t, err)

			found := false
			for _, mfs := range argoSecret.GetManagedFields() {
				if mfs.Manager == fieldManager {
					found = true
					break
				}
			}

			assert.Equal(t, tc.changed, found)
			assert.Equal(t, "foo", string(argoSecret.Data["admin.password"]))

			validateMtime(t, string(argoSecret.Data["admin.passwordMtime"]))
		})
	}
}

func makeCluster(t *testing.T, cid, repoUrl string) *api.Cluster {

	apiId := api.Id(cid)
	clusterId := api.ClusterId{Id: &apiId}
	props := api.ClusterProperties{
		GitRepo: &api.GitRepo{
			Url: &repoUrl,
		},
	}
	return &api.Cluster{
		ClusterId:         clusterId,
		ClusterProperties: props,
	}
}

func TestCreateRepoSecret(t *testing.T) {
	sshSecret := makeSSHSecret("thepubkey")
	fakeClient := fake.NewClientset(sshSecret)
	ctx := t.Context()

	cluster := makeCluster(t, "c-test-1234", "https://git.syn.tools/cluster-catalog.git")
	repoUrl, err := url.Parse(*cluster.GitRepo.Url)
	require.NoError(t, err)

	err = createRepoSecret(ctx, cluster, fakeClient, "syn")
	require.NoError(t, err)

	repoSecret, err := fakeClient.CoreV1().Secrets("syn").Get(ctx, argoRepoSecretName, metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(
		t,
		map[string]string{
			"argocd.argoproj.io/secret-type": "repository",
		},
		repoSecret.ObjectMeta.Labels,
	)

	assert.Equal(t, "git", string(repoSecret.Data["type"]))
	assert.Equal(t, repoUrl.String(), string(repoSecret.Data["url"]))
}

func TestCreateSSHSecret(t *testing.T) {
	fakeClient := fake.NewClientset()

	ctx := t.Context()

	pubkey, err := CreateSSHSecret(ctx, fakeClient, "syn")
	require.NoError(t, err)

	sshSecret := validateSSHSecret(t, ctx, fakeClient, pubkey)
	assert.NotEmpty(t, sshSecret.Data[argoSSHPrivateKey])
}

func TestCreateSSHSecretNoUpdate(t *testing.T) {
	sshSecret := makeSSHSecret("thepubkey")
	fakeClient := fake.NewClientset(sshSecret)

	ctx := t.Context()

	pubkey, err := CreateSSHSecret(ctx, fakeClient, "syn")
	require.NoError(t, err)
	assert.Equal(t, "thepubkey", pubkey)

	_ = validateSSHSecret(t, ctx, fakeClient, "thepubkey")
}

func validateSecret(t *testing.T, ctx context.Context, fakeClient *fake.Clientset, name string) {

	argoSecret, err := fakeClient.CoreV1().Secrets("syn").Get(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword(argoSecret.Data["admin.password"], []byte("foo")))
	validateMtime(t, string(argoSecret.Data["admin.passwordMtime"]))
}

func validateMtime(t *testing.T, secretMtime string) {
	parsed, err := time.Parse(time.RFC3339, string(secretMtime))
	assert.NoError(t, err)
	assert.True(t, time.Now().Sub(parsed) < 5*time.Second)
}

func makeSSHSecret(pubkey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      argoSSHSecretName,
			Namespace: "syn",
		},
		Data: map[string][]byte{
			argoSSHPublicKey: []byte(pubkey),
		},
	}
}

func validateSSHSecret(t *testing.T, ctx context.Context, fakeClient *fake.Clientset, pubkey string) *corev1.Secret {
	secret, err := fakeClient.CoreV1().Secrets("syn").Get(ctx, argoSSHSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, pubkey, string(secret.Data[argoSSHPublicKey]))
	return secret
}
