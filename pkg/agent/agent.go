package agent

import (
	"context"
	"net/url"
	"time"

	"git.vshn.net/syn/steward/pkg/api"
	log "github.com/sirupsen/logrus"
)

// Agent configures the cluster agent
type Agent struct {
	APIURL       *url.URL
	Token        string
	CloudType    string
	CloudRegion  string
	Distribution string
}

// Run starts the cluster agent
func (a *Agent) Run(ctx context.Context) error {
	apiClient := api.NewClient(nil)
	apiClient.BaseURL = a.APIURL
	apiClient.Token = a.Token
	ticker := time.NewTicker(1 * time.Minute)

	doUpdate := func() error {
		git, err := apiClient.RegisterCluster(ctx, a.CloudType, a.CloudRegion, a.Distribution)
		if err != nil {
			return err
		}
		log.Debugf("%+v", git)
		return nil
	}

	if err := doUpdate(); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			if err := doUpdate(); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
