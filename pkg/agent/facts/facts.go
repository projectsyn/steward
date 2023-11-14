package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode"

	"github.com/projectsyn/lieutenant-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type FactCollector struct {
	Client *kubernetes.Clientset

	AdditionalFactsConfigMapNamespace string
	AdditionalFactsConfigMapName      string
}

func (col FactCollector) FetchDynamicFacts(ctx context.Context) (*api.DynamicClusterFacts, error) {
	facts := api.DynamicClusterFacts{}
	kubeVersion, err := col.fetchKubernetesVersion(ctx)
	if err != nil {
		klog.Errorf("Error fetching kubernetes version: %v", err)
	}
	if kubeVersion != nil {
		facts["kubernetesVersion"] = kubeVersion
	}

	ocpVersion, err := col.fetchOpenshiftVersion(ctx)
	if err != nil {
		klog.Errorf("Error fetching openshift version: %v", err)
	}
	if ocpVersion != nil {
		facts["openshiftVersion"] = ocpVersion
	}

	ocpOAuthRoute, err := col.fetchOpenshiftOAuthRoute(ctx)
	if err != nil {
		klog.Errorf("Error fetching openshift oauth route: %v", err)
	}
	if ocpOAuthRoute != "" {
		facts["openshiftOAuthRoute"] = ocpOAuthRoute
	}

	additionalFacts, err := col.fetchAdditionalFacts(ctx)
	if err != nil {
		klog.Errorf("Error fetching additional facts: %v", err)
	}
	for k, v := range additionalFacts {
		facts[k] = v
	}

	return &facts, nil
}

func (col FactCollector) fetchKubernetesVersion(ctx context.Context) (*version.Info, error) {
	// We are not using `col.client.ServerVersion()` to get context support
	body, err := col.Client.RESTClient().Get().AbsPath("/version").Do(ctx).Raw()
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

func (col FactCollector) fetchAdditionalFacts(ctx context.Context) (map[string]string, error) {
	if col.AdditionalFactsConfigMapName == "" {
		return nil, nil
	}
	cm, err := col.Client.CoreV1().ConfigMaps(col.AdditionalFactsConfigMapNamespace).Get(ctx, col.AdditionalFactsConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to fetch the additional facts config map: %w", err)
	}
	return cm.Data, nil
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
