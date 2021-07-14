package manifests

import "embed"

//go:embed *.yaml
var Manifests embed.FS
