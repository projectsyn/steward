package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/projectsyn/steward/pkg/images"
)

var crds = []string{
	"https://raw.githubusercontent.com/argoproj/argo-cd/{{VERSION}}/manifests/crds/application-crd.yaml",
	"https://raw.githubusercontent.com/argoproj/argo-cd/{{VERSION}}/manifests/crds/appproject-crd.yaml",
}

func main() {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	version, err := version(images.DefaultArgoCDImage)
	if err != nil {
		panic(err)
	}

	for _, urlTmpl := range crds {
		err := download(strings.Replace(urlTmpl, "{{VERSION}}", version, -1), path+"/manifests/")
		if err != nil {
			panic(err)
		}
	}
}

func version(img string) (string, error) {
	_, version, found := strings.Cut(img, ":")

	if !found {
		return "", fmt.Errorf("Invalid format, expected to find ':' image: %s", images.DefaultArgoCDImage)
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
	fmt.Fprintln(f, "# This file is overridden with `go generate`. DO NOT EDIT.")
	fmt.Fprintln(f, "# Download using go generate ./...")
	fmt.Fprintf(f, "# url: %s\n", url)
	io.Copy(f, r.Body)

	return nil
}
