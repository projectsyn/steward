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

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/projectsyn/steward/pkg/agent/facts"
	"github.com/projectsyn/steward/pkg/argocd"
)

// Agent configures the cluster agent
type Agent struct {
	APIURL            *url.URL
	Token             string
	ClusterID         string
	CloudType         string
	CloudRegion       string
	Distribution      string
	Namespace         string
	OperatorNamespace string
	ArgoCDImage       string
	RedisImage        string
	// The configmap containing additional facts to be added to the dynamic facts
	AdditionalFactsConfigMap string

	// The configmap containing metadata for additional root apps to deploy
	AdditionalRootAppsConfigMap string

	// Reference to the OpenShift OAuth route to be added to the dynamic facts
	OCPOAuthRouteNamespace string
	OCPOAuthRouteName      string

	facts facts.FactCollector
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
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	a.facts = facts.FactCollector{
		Client: client,

		OAuthRouteNamespace: a.OCPOAuthRouteNamespace,
		OAuthRouteName:      a.OCPOAuthRouteName,

		AdditionalFactsConfigMapNamespace: a.Namespace,
		AdditionalFactsConfigMapName:      a.AdditionalFactsConfigMap,
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
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	publicKey, err := argocd.CreateSSHSecret(ctx, config, a.Namespace)
	if err != nil {
		klog.Errorf("Error creating SSH secret: %v", err)
		return
	}
	if err := argocd.CreateArgoSecret(ctx, config, a.Namespace, a.Token); err != nil {
		klog.Errorf("Error creating Argo CD secret: %v", err)
		return
	}
	patchCluster := api.ClusterProperties{
		GitRepo: &api.GitRepo{
			DeployKey: &publicKey,
		},
	}
	patchCluster.DynamicFacts, err = a.facts.FetchDynamicFacts(ctx)
	if err != nil {
		klog.Errorf("Error fetching dynamic facts: %v", err)
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
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
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

	if err := argocd.Apply(ctx, config, a.Namespace, a.OperatorNamespace, a.ArgoCDImage, a.RedisImage, a.AdditionalRootAppsConfigMap, cluster); err != nil {
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
