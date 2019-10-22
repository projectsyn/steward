package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"git.vshn.net/syn/steward/pkg/agent"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Version is the steward version (set during build)
var Version = "unreleased"

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting SYN cluster agent 🕵️")
	logrus.Infof("Version %s", Version)
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
	app.Flag("cloud", "Cloud type this cluster is running on").StringVar(&agent.CloudType)
	app.Flag("region", "Cloud region this cluster is running in").StringVar(&agent.CloudRegion)
	app.Flag("distribution", "Kubernetes distribution this cluster is running").StringVar(&agent.Distribution)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func receiveSignal(signalCh chan os.Signal, cancel context.CancelFunc) {
	for {
		select {
		case sig := <-signalCh:
			logrus.Debugf("Received signal '%v'", sig)
			cancel()
		}
	}
}
