package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/projectsyn/steward/pkg/images"
)

var crds = []string{
	"https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/crds/application-crd.yaml",
	"https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/crds/appproject-crd.yaml",
}

func main() {
	path, err := os.Getwd()
	if err != nil {
		abort(fmt.Errorf("failed to get current directory: %w", err))
	}

	version, err := version(images.DefaultArgoCDImage)
	if err != nil {
		abort(fmt.Errorf("failed to get version: %w", err))
	}

	for _, urlTmpl := range crds {
		err := download(fmt.Sprintf(urlTmpl, version), path+"/manifests/")
		if err != nil {
			abort(fmt.Errorf("failed to download CRD: %w", err))
		}
	}
}

func version(img string) (string, error) {
	_, version, found := strings.Cut(img, ":")

	if !found {
		return "", fmt.Errorf("invalid format, expected to find ':' image: %s", images.DefaultArgoCDImage)
	}

	return version, nil
}

func download(url string, dir string) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	f, err := os.Create(dir + filepath.Base(url))
	if err != nil {
		return err
	}
	defer f.Close()

	h := &bytes.Buffer{}
	h.WriteString("# This file is overridden with `go generate`. DO NOT EDIT.\n")
	h.WriteString("# Download using go generate ./...\n")
	h.WriteString("# url: " + url + "\n")

	_, err = io.Copy(f, io.MultiReader(h, r.Body))
	return err
}

func abort(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
