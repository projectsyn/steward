package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
)

type OpenshiftVersionDesired struct {
	Version string
}

type OpenshiftVersionHistory struct {
	State          string
	Verified       bool
	Version        string
	CompletionTime time.Time
}

type OpenshiftVersionStatus struct {
	Desired OpenshiftVersionDesired
	History []OpenshiftVersionHistory
}
type OpenshiftVersion struct {
	Status OpenshiftVersionStatus
}

func (col factCollector) fetchOpenshiftVersion(ctx context.Context) (*OpenshiftVersionFact, error) {
	body, err := col.client.RESTClient().Get().AbsPath("/apis/config.openshift.io/v1/clusterversions/version").Do(ctx).Raw()
	if err != nil {
		if errors.IsNotFound(err) {
			// API server doesn't know `clusterversions` or there is no resource, so we are not running on openshift.
			return nil, nil
		}
		return nil, err
	}
	var version OpenshiftVersion
	err = json.Unmarshal(body, &version)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the openshift version: %w", err)
	}

	return processOpenshiftVersion(version)
}

type SemanticVersion struct {
	Major string
	Minor string
	Patch string
}

type OpenshiftVersionFact struct {
	Version        SemanticVersion
	DesiredVersion SemanticVersion
}

func processOpenshiftVersion(v OpenshiftVersion) (*OpenshiftVersionFact, error) {
	currentVersion := ""
	lastedUpdate := time.Time{}
	for _, h := range v.Status.History {
		if h.State == "Completed" && h.Verified == true && h.CompletionTime.After(lastedUpdate) {
			currentVersion = h.Version
			lastedUpdate = h.CompletionTime
		}
	}
	var err error
	versionFact := &OpenshiftVersionFact{}
	versionFact.DesiredVersion, err = parseSematicVersion(v.Status.Desired.Version)
	if err != nil {
		return versionFact, fmt.Errorf("unable to parse desiredVersion: %w", err)
	}
	versionFact.Version, err = parseSematicVersion(currentVersion)
	if err != nil {
		return versionFact, fmt.Errorf("unable to parse version: %w", err)
	}
	return versionFact, nil
}

func parseSematicVersion(s string) (SemanticVersion, error) {
	vs := strings.Split(s, ".")
	if len(vs) != 3 {
		return SemanticVersion{
			Major: "",
			Minor: "",
			Patch: "",
		}, fmt.Errorf("unknown version %q", s)

	}
	v := SemanticVersion{}
	major := trimVersion(vs[0])
	if major == "" {
		return v, fmt.Errorf("unknown major version %q", vs[0])
	}
	v.Major = major
	minor := trimVersion(vs[1])
	if major == "" {
		return v, fmt.Errorf("unknown minor version %q", vs[1])
	}
	v.Minor = minor
	patch := trimVersion(vs[2])
	if major == "" {
		return v, fmt.Errorf("unknown patch version %q", vs[2])
	}
	v.Patch = patch
	return v, nil
}
