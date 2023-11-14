package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/projectsyn/steward/pkg/agent/facts"
)

func main() {
	ns := flag.String("namespace", "syn", "namespace in which steward is running")
	additionalFactsConfigMap := flag.String("additional-facts-config-map", "additional-facts", "configmap containing additional facts to be added to the dynamic facts")
	ocpOAuthRouteNamespace := flag.String("ocp-oauth-route-namespace", "openshift-authentication", "Namespace for the OpenShift OAuth route")
	ocpOAuthRouteName := flag.String("ocp-oauth-route-name", "oauth-openshift", "Name of the OpenShift OAuth route")

	flag.Parse()

	cfg := config.GetConfigOrDie()

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	c := facts.FactCollector{
		Client: client,

		OAuthRouteNamespace: *ocpOAuthRouteNamespace,
		OAuthRouteName:      *ocpOAuthRouteName,

		AdditionalFactsConfigMapNamespace: *ns,
		AdditionalFactsConfigMapName:      *additionalFactsConfigMap,
	}

	fcts, err := c.FetchDynamicFacts(context.Background())
	if err != nil {
		panic(err)
	}
	out, err := json.MarshalIndent(fcts, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
