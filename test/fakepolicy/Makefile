ROOTDIR ?= $(realpath $(CURDIR)/../..)

include ../../Makefile

.PHONY: manifests
manifests: $(CONTROLLER_GEN) $(KUSTOMIZE) ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build ./config/default > config/deploy.yaml
	cp ./config/samples/* ./test/utils/testdata/
