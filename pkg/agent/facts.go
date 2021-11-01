package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

type factCollector struct {
	client *kubernetes.Clientset
}

func (col factCollector) fetchDynamicFacts(ctx context.Context) (*api.DynamicClusterFacts, error) {
	facts := api.DynamicClusterFacts{}
	kubeVersion, err := col.fetchKubernetesVersion(ctx)
	if err != nil {
		return nil, err
	}
	facts["kubernetesVersion"] = kubeVersion

	ocpVersion, err := col.fetchOpenshiftVersion(ctx)
	if err != nil {
		return nil, err
	}
	if ocpVersion != nil {
		facts["openshiftVersion"] = ocpVersion
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
		return nil, fmt.Errorf("unable to parse the kubernetes version: %w", err)
	}
	info, err = processKubernetesVersion(info)
	if err != nil {
		return nil, fmt.Errorf("unexpected kubernetes version: %w", err)
	}
	return &info, nil
}

func processKubernetesVersion(v version.Info) (version.Info, error) {
	major := trimVersion(v.Major)
	if major == "" {
		return v, fmt.Errorf("unknown major version %q", v.Major)
	}
	v.Major = major

	minor := trimVersion(v.Minor)
	if minor == "" {
		return v, fmt.Errorf("unknown minor version %q", v.Minor)
	}
	v.Minor = minor
	return v, nil
}

func trimVersion(v string) string {
	res := []rune{}
	for _, r := range v {
		if !unicode.IsDigit(r) {
			break
		}
		res = append(res, r)
	}
	return string(res)
}
