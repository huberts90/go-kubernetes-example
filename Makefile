DOCKER_IMAGE:=investigate-go-configmap
VERSION:=latest


.PHONY: docker-build
docker-build: ## build deployable image, and run tests, using official toolchain
	DOCKER_BUILDKIT=1 docker build \
		-f Dockerfile \
		-t $(DOCKER_IMAGE):$(VERSION) \
		.

.PHONY: run-k8s
run-k8s:
	DOCKER_TAG=$(DOCKER_IMAGE):$(VERSION) ./scripts/setup.sh