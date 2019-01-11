#############################################################
# Kind targets  https://github.com/kubernetes-sigs/kind
#############################################################

KIND_VERSION = "0.0.1"

kind-install:
	curl -Lo kind https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-linux-amd64 && chmod +x kind && sudo mv kind /usr/local/bin/

# this step might change later when we have multi-nodes cluster
kind-create-cluster:
	kind create cluster

kind-e2e-tests:
	make e2e-tests KUBECONFIG=$(shell kind get kubeconfig-path)
