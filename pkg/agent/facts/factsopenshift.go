package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
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

type OpenshiftRoute struct {
	Spec struct {
		Host string
	}
}

func (col FactCollector) fetchOpenshiftVersion(ctx context.Context) (*SemanticVersion, error) {
	body, err := col.Client.RESTClient().Get().AbsPath("/apis/config.openshift.io/v1/clusterversions/version").Do(ctx).Raw()
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

func (col FactCollector) fetchOpenshiftOAuthRoute(ctx context.Context) (string, error) {
	body, err := col.Client.RESTClient().Get().
		AbsPath(
			path.Join("/apis/route.openshift.io/v1/namespaces", col.OAuthRouteNamespace, "routes", col.OAuthRouteName),
		).Do(ctx).Raw()
	if err != nil {
		if errors.IsNotFound(err) {
			// API server doesn't know `routes` or there is no resource, so we are not running on openshift.
			return "", nil
		}
		return "", fmt.Errorf("unable to fetch the openshift route: %w", err)
	}
	var route OpenshiftRoute
	err = json.Unmarshal(body, &route)
	if err != nil {
		return "", fmt.Errorf("unable to parse the openshift route: %w", err)
	}

	return route.Spec.Host, nil
}

type SemanticVersion struct {
	Major string
	Minor string
	Patch string
}

func processOpenshiftVersion(v OpenshiftVersion) (*SemanticVersion, error) {
	currentVersion := ""
	lastedUpdate := time.Time{}
	for _, h := range v.Status.History {
		if h.State == "Completed" && h.Verified == true && h.CompletionTime.After(lastedUpdate) {
			currentVersion = h.Version
			lastedUpdate = h.CompletionTime
		}
	}
	versionFact, err := parseSematicVersion(currentVersion)
	if err != nil {
		klog.Warningf("unable to parse version %v : %s\nFalling back to desired version", versionFact, err)
		versionFact, err = parseSematicVersion(v.Status.Desired.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to parse desiredVersion: %w", err)
		}
	}
	return versionFact, err
}

func parseSematicVersion(s string) (*SemanticVersion, error) {
	vs := strings.Split(s, ".")
	if len(vs) != 3 {
		return &SemanticVersion{
			Major: "",
			Minor: "",
			Patch: "",
		}, fmt.Errorf("unknown version %q", s)

	}
	v := &SemanticVersion{}
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
