package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

type factCollector struct {
	client *kubernetes.Clientset
}

func (col factCollector) fetchDynamicFacts(ctx context.Context) (*api.DynamicClusterFacts, error) {
	kubeVersion, err := col.client.ServerVersion()
	if err != nil {
		return nil, err
	}
	facts := api.DynamicClusterFacts{
		"kubernetesVersion": kubeVersion,
	}

	return &facts, nil
}

func (col factCollector) fetchKubernetesVersion(ctx context.Context) (*version.Info, error) {
	// We are not using `col.client.ServerVersion()` to get context support
	body, err := col.client.RESTClient().Get().AbsPath("/version").Do(ctx).Raw()
	if err != nil {
		return nil, err
	}
	var info version.Info
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the server version: %v", err)
	}
	return &info, nil
}
