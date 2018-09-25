DOCKER_TAG ?= latest
.PHONY: up run stop build specs

ALL: vendorSync

attach/%:
	docker-compose run $*

clear: ## clears generated artifacts
	docker-compose down -v --remove-orphans --rmi local
	rm -fr vendor/*/
	rm -f contract/iryo.abi contract/iryo.wasm contract/iryo.wast
	rm -fr .data/

init: up/nodeos run/cleos up/deploy ## sets the nodeos up - creates master, iryo, iryo.token accounts and publishes contracts on them

up: up/nodeos up/api up/patient1 up/patient2 up/doctor1 up/doctor2 ## start nodeos, api and clients

up/%: stop/% ## start a service in background
	docker-compose up -d $*

run/%: stop/% ## run a service in foreground
	docker-compose up $*

stop: ## stop all services in docker-compose
	docker-compose stop

stop/%: ## stop a service in docker-compose
	docker-compose stop $*

logs: ## shows docker compose logs
	docker-compose logs -f --tail=0 $*

specs: ## builds the protobuf spec
	$(MAKE) -C ./specs

test: test/unit ## run all tests

test/unit: ## run all unit tests
	go test ./...

test/unit/%: ## run unit tests for a specific project
	go test ./$*

vendorSync: go.mod ## syncs the vendor folder to match vendor.json
	go mod vendor

vendorUpdate: ## updates the vendor folder
	go mod tidy

help: ## displays this message
	@grep -E '^[a-zA-Z_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

watch/%: ## helper for running tasks on file change (requires watchdog)
	watchmedo shell-command -i "./.git/*;./.data/*;./.bin/*" --recursive --ignore-directories --wait --command "$(MAKE) $*"

package: package/api package/client

package/%:
	docker build --build-arg BIN=$* --file=Dockerfile.build --tag=iryo/poc-$*:$(DOCKER_TAG) .

publish: publish/api publish/client

publish/%: package/%
	docker push iryo/poc-$*:$(DOCKER_TAG)
