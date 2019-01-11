#! /bin/bash

# e2e tests basic for operator

# Debug infos
echo "KUBECONFIG variable point to :"
echo "$KUBECONFIG"
echo 
# check if the cluster is up-and-running
echo " -- cluster info --"
kubectl cluster-info -v5 --kubeconfig="$KUBECONFIG"
echo

#   make local-run VERBOSE_LEVEL=5 of the k8s operator (outside k8s cluster)
make local-run KUBECONFIG=$KUBECONFIG
