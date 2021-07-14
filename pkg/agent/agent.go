package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/projectsyn/lieutenant-api/pkg/api"
	"github.com/projectsyn/steward/pkg/argocd"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

// Agent configures the cluster agent
type Agent struct {
	APIURL       *url.URL
	Token        string
	ClusterID    string
	CloudType    string
	CloudRegion  string
	Distribution string
	Namespace    string
	ArgoCDImage  string
	RedisImage   string
}

// Run starts the cluster agent
func (a *Agent) Run(ctx context.Context) error {
	bearerToken, _ := securityprovider.NewSecurityProviderBearerToken(a.Token)
	apiClient, err := api.NewClient(a.APIURL.String(), api.WithRequestEditorFn(bearerToken.Intercept))
	if err != nil {
		return err
	}

	kubecfg := os.Getenv("KUBECONFIG")
	var config *rest.Config
	if kubecfg == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubecfg)
	}
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Minute)

	a.registerCluster(ctx, config, apiClient)

	for {
		select {
		case <-ticker.C:
			a.registerCluster(ctx, config, apiClient)
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Agent) registerCluster(ctx context.Context, config *rest.Config, apiClient *api.Client) {
	publicKey, err := argocd.CreateSSHSecret(config, a.Namespace)
	if err != nil {
		klog.Errorf("Error creating SSH secret: %v", err)
		return
	}
	if err := argocd.CreateArgoSecret(config, a.Namespace, a.Token); err != nil {
		klog.Errorf("Error creating Argo CD secret: %v", err)
		return
	}
	patchCluster := api.ClusterProperties{
		GitRepo: &api.GitRepo{
			DeployKey: &publicKey,
		},
	}

	setFact("cloud", a.CloudType, &patchCluster)
	setFact("region", a.CloudRegion, &patchCluster)
	setFact("distribution", a.Distribution, &patchCluster)

	var buf io.ReadWriter
	buf = new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(patchCluster); err != nil {
		klog.Error(err)
		return
	}
	resp, err := apiClient.UpdateClusterWithBody(ctx, api.ClusterIdParameter(a.ClusterID), api.ContentJSONPatch, buf)
	if err != nil {
		klog.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		reason := &api.Reason{}
		if err := json.NewDecoder(resp.Body).Decode(reason); err != nil {
			klog.Error(err)
			return
		}
		klog.Error(reason.Reason)
		return
	}
	cluster := &api.Cluster{}
	if err := json.NewDecoder(resp.Body).Decode(cluster); err != nil {
		klog.Error(err)
		return
	}

	if err := argocd.Apply(ctx, config, a.Namespace, a.ArgoCDImage, a.RedisImage, apiClient, cluster); err != nil {
		klog.Error(err)
	}
}

func setFact(fact, value string, cluster *api.ClusterProperties) {
	if len(value) == 0 {
		return
	}
	if cluster.Facts == nil {
		cluster.Facts = &api.ClusterFacts{}
	}
	(*cluster.Facts)[fact] = value
}
