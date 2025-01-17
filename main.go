//go:generate go run ./build/download-manifests
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog"

	"github.com/projectsyn/steward/pkg/agent"
	"github.com/projectsyn/steward/pkg/images"

	"github.com/alecthomas/kingpin/v2"
)

// Version is the steward version (set during build)
var Version = "unreleased"

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Set("v", "3")
	klog.Info("Starting SYN cluster agent 🕵️")
	klog.Infof("Version %s", Version)
	app := kingpin.New("steward", "Steward makes your Kubernetes cluster SYN managed. 🎉")
	app.DefaultEnvars()
	app.Version(Version)
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM)
	go receiveSignal(signalCh, cancel)

	agent := agent.Agent{}
	app.Action(func(*kingpin.ParseContext) error {
		return agent.Run(ctx)
	})

	app.Flag("api", "API URL to connect to").Required().URLVar(&agent.APIURL)
	app.Flag("token", "Token to authenticate to the API").Required().StringVar(&agent.Token)
	app.Flag("cluster-id", "ID of own cluster").Required().StringVar(&agent.ClusterID)
	app.Flag("cloud", "Cloud type this cluster is running on").StringVar(&agent.CloudType)
	app.Flag("region", "Cloud region this cluster is running in").StringVar(&agent.CloudRegion)
	app.Flag("distribution", "Kubernetes distribution this cluster is running").StringVar(&agent.Distribution)
	app.Flag("namespace", "Namespace in which steward is running").Default("syn").StringVar(&agent.Namespace)
	app.Flag("operator-namespace", "Namespace in which the ArgoCD operator will be running").Default("syn-argocd-operator").StringVar(&agent.OperatorNamespace)
	app.Flag("argo-image", "Image to be used for the Argo CD deployments").Default(images.DefaultArgoCDImage).StringVar(&agent.ArgoCDImage)
	app.Flag("redis-image", "Image to be used for the Argo CD Redis deployment").Default(images.DefaultRedisImage).StringVar(&agent.RedisImage)
	app.
		Flag(
			"additional-facts-config-map",
			"Additional facts added to the dynamic facts in the cluster object. Keys in the configmap's data field can't override existing keys.").
		Default("additional-facts").
		StringVar(&agent.AdditionalFactsConfigMap)
	app.
		Flag(
			"additional-root-apps-config-map",
			"Config map holding metadata for additional ArgoCD root apps and app projects.").
		Default("additional-root-apps").
		StringVar(&agent.AdditionalRootAppsConfigMap)
	app.
		Flag(
			"ocp-oauth-route-namespace",
			"Namespace for the OpenShift OAuth route").
		Default("openshift-authentication").
		StringVar(&agent.OCPOAuthRouteNamespace)
	app.
		Flag(
			"ocp-oauth-route-name",
			"Name of the OpenShift OAuth route").
		Default("oauth-openshift").
		StringVar(&agent.OCPOAuthRouteName)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func receiveSignal(signalCh chan os.Signal, cancel context.CancelFunc) {
	for {
		select {
		case sig := <-signalCh:
			klog.V(3).Infof("Received signal '%v'", sig)
			cancel()
		}
	}
}
