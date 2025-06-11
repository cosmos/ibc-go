DOCKER := $(shell which docker)
TEST_CONTAINERS=$(shell docker ps --filter "label=ibc-test" -a -q)

cleanup-ibc-test-containers:
	for id in $(TEST_CONTAINERS) ; do \
		$(DOCKER) stop $$id ; \
		$(DOCKER) rm $$id ; \
	done

init:
	./scripts/init.sh

e2e-test: init cleanup-ibc-test-containers
	./scripts/run-e2e.sh $(test) $(entrypoint)

e2e-suite: init cleanup-ibc-test-containers
	RUN_SUITE="true" ./scripts/run-e2e.sh "" $(entrypoint)

.PHONY: cleanup-ibc-test-containers e2e-test init
