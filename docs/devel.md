# Development environment for `dex-operator`

## Project structure

This project follows the conventions presented in the [standard Golang
project](https://github.com/golang-standards/project-layout).

## Dependencies

* `dep` (will be installed automatically if not detected)
* `go >= 1.10`

### Bumping the Kubernetes version used by `dex-operator`

Update the constraints in [`Gopkg.toml`](../Gopkg.toml).

## Building

A simple `make` should be enough. This should compile [the main
function](../cmd/dex-operator/main.go) and generate a `dex-operator` binary as
well as a _Docker_ image.

## Running `dex-operator` in your Development Environment

There are multiple ways you can run the `dex-operator` for bootstrapping
and managinig your Kubernetes cluster:

### ... in your local machine

You can run the `dex-operator` container locally with a
`make local-run`. This will:

  * build the `dex-operator` image
  * run it locally
    * using the `kubeconfig` in `/etc/kubernetes/admin.conf`
    * using the config files in [`../config`](`../config`)
