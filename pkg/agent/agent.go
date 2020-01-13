package agent

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/projectsyn/steward/pkg/api"
	"github.com/projectsyn/steward/pkg/argocd"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

// Agent configures the cluster agent
type Agent struct {
	APIURL       *url.URL
	Token        string
	CloudType    string
	CloudRegion  string
	Distribution string
	Namespace    string
	ArgoCDImage  string
}

// Run starts the cluster agent
func (a *Agent) Run(ctx context.Context) error {
	apiClient := api.NewClient(nil)
	apiClient.BaseURL = a.APIURL
	apiClient.Token = a.Token

	kubecfg := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
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
	if git, err := apiClient.RegisterCluster(ctx, a.CloudType, a.CloudRegion, a.Distribution, publicKey); err != nil {
		klog.Error(err)
	} else {
		if err := argocd.Apply(ctx, config, a.Namespace, a.ArgoCDImage, apiClient, git); err != nil {
			klog.Error(err)
		}
	}
}
