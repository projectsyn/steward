package images

// WARNING: Renovate updates the images in this file. If adding changes double check the
// renovate.json file and it's regexManagers.

const (
	// DefaultArgoCDImage is the default image to use for the ArgoCD deployment.
	// You should also update the CRDs in the manifests/ directory to match this version.
	DefaultArgoCDImage = "quay.io/argoproj/argocd:v2.9.2"
	DefaultRedisImage  = "docker.io/redis:7.4.1"
)
