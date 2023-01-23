DOCKER := $(shell which docker)
TEST_CONTAINERS=$(shell docker ps --filter "label=ibc-test" -a -q)

cleanup-ibc-test-containers:
	for id in $(TEST_CONTAINERS) ; do \
		$(DOCKER) stop $$id ; \
		$(DOCKER) rm $$id ; \
	done

e2e-test: cleanup-ibc-test-containers
	./scripts/run-e2e.sh $(entrypoint) $(test)

compatibility-tests:
	./scripts/run-compatibility-tests.sh $(release_branch)

.PHONY: cleanup-ibc-test-containers e2e-test compatibility-tests
