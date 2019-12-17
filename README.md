# Project Syn: Steward

The cluster agent - working together with the [lieutenant-api](https://github.com/projectsyn/lieutenant-api).

**Please note that this project is in it's early stages and under active development**.

## Getting started

1. Make sure [Golang](https://golang.org/doc/install) is installed

1. Run the agent against a locally deployed SYNventory

   ```console
   go run main.go --api http://localhost:5000 --token someToken
   ```

1. Start hacking on the service

## Build Docker Image

To deploy the agent on a cluster, build the Docker image:

```console
docker build . -t image-name:v0.0.1
```
