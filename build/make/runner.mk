#############################################################
# Some simple run targets
# (for testing things locally)
#############################################################

KUBECTL    = kubectl --kubeconfig=$(KUBECONFIG)

# assuming the k8s cluster is accessed with $(KUBECONFIG),
# deploy the dex-operator manifest file in this cluster.
local-deploy: $(DEX_DEPLOY) docker-image-local
	@echo ">>> (Re)deploying..."
	@[ -r $(KUBECONFIG) ] || $(SUDO_E) chmod 644 $(KUBECONFIG)
	@echo ">>> Deleting any previous resources..."
	-@kubectl get ldapconnectors -o jsonpath="{..metadata.name}" | \
	        xargs -r kubectl delete --all=true ldapconnector 2>/dev/null
	-@kubectl get dexconfigurations -o jsonpath="{..metadata.name}" | \
	        xargs -r kubectl delete --all=true dexconfiguration 2>/dev/null
	@sleep 30
	-@kubectl delete --all=true --cascade=true -f $(DEX_DEPLOY) 2>/dev/null
	@echo ">>> Regenerating manifests..."
	@make manifests
	@echo ">>> Loading manifests..."
	kubectl apply --kubeconfig $(KUBECONFIG) -f $(DEX_DEPLOY)

clean-local-deploy:
	@make manifests
	@echo ">>> Uninstalling manifests..."
	kubectl delete --kubeconfig $(KUBECONFIG) -f $(DEX_DEPLOY)

# Usage:
# - Run it locally:
#   make local-run VERBOSE_LEVEL=5
# - Start a Deployment with the manager:
#   make local-run EXTRA_ARGS="--"
#

local-run: $(DEX_OPER_EXE) $(KUBECONFIG)
	@echo ">>> Loading k8s CRD with kubectl apply"
	@for f in config/sas/*.yaml config/crds/*.yaml ; do $(KUBECTL) apply -f $$f ; done
	@sleep 5
	@echo ">>> Running $(DEX_OPER_EXE)"
	$(DEX_OPER_EXE) manager \
		-v $(VERBOSE_LEVEL) \
		--kubeconfig $(KUBECONFIG) \
		$(EXTRA_ARGS) &

docker-run: $(IMAGE_TAR_GZ)
	@echo ">>> Running $(IMAGE_NAME):latest in the local Docker"
	docker run -it --rm \
               --privileged=true \
               --net=host \
               --security-opt seccomp:unconfined \
               --cap-add=SYS_ADMIN \
               --name=$(IMAGE_BASENAME) \
               $(CONTAINER_VOLUMES) \
               $(IMAGE_NAME):latest $(EXTRA_ARGS)

docker-image-local: local-$(IMAGE_TAR_GZ)

docker-image: $(IMAGE_TAR_GZ)
docker-image-clean:
	-[ -f $(IMAGE_NAME) ] && docker rmi $(IMAGE_NAME)
	rm -f $(IMAGE_TAR_GZ)
