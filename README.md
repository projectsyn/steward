# Project Syn: Steward

The cluster agent - working together with the [lieutenant-api](https://github.com/projectsyn/lieutenant-api).

**Please note that this project is in it's early stages and under active development**.

## Getting started

1. Make sure [Golang](https://golang.org/doc/install) is installed

1. Run the agent against a locally deployed SYNventory

   ```console
   KUBECONFIG=/path/to/config go run main.go --api http://localhost:5000 --token someToken
   ```

1. Start hacking on the agent

## Argo CD Bootstrapping

Argo CD is bootstrapped (repo-server, app-controller, server) when no existing deployments are found. The required CRDs (Application, AppProject) are compiled into the binary using [statik](https://github.com/rakyll/statik).
To update them, download the [latest manifests](https://github.com/argoproj/argo-cd/tree/master/manifests/crds) and put them in `./manifests/`. Upon running `make generate`, the manifests will be embedded into the resulting binary.

## Release

Create a git tag:

```console
git tag v0.0.2
```

Build the Docker image:

```console
make docker
```
