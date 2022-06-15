module github.com/projectsyn/steward

go 1.16

require (
	github.com/deepmap/oapi-codegen v1.11.0
	github.com/projectsyn/lieutenant-api v0.7.0
	github.com/stretchr/testify v1.7.1
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
)

// Pinned v0.21.2
replace k8s.io/client-go => k8s.io/client-go v0.21.2
