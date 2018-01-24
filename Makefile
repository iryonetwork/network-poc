.PHONY: up run stop build specs

ALL: vendorSync

clear: ## clears generated artifacts
	docker-compose down
	rm -fr vendor/*/

up: ## start all basic services
	@echo TBD

up/%: stop/% ## start a service in background
	docker-compose up -d $*

run/%: stop/% ## run a service in foreground
	docker-compose up $*

stop: ## stop all services in docker-compose
	docker-compose stop

stop/%: ## stop a service in docker-compose
	docker-compose stop $*

# build/%: ## builds a specific project
# 	@mkdir -p .bin
# 	@if [ -a cmd/$*/main.go ]; then \
# 		GOOS=linux GOARCH=amd64 go build -o ./.bin/$* ./cmd/$* ; \
# 	else \
# 		echo "No sources found in cmd/$*/main.go"; \
# 	fi

buildContract: contract/contract.go ## rebuilds the smart contract

contract/contract.go: contract/contract.sol
	abigen --sol=contract/contract.sol --pkg=contract --out=contract/contract.go

logs: ## shows docker compose logs
	docker-compose logs -f --tail=0 $*

specs: ## builds the protobuf spec
	$(MAKE) -C ./specs

test: test/unit ## run all tests

test/unit: ## run all unit tests
	go test ./...

test/unit/%: ## run unit tests for a specific project
	go test ./$*

vendorSync: vendor/vendor.json ## syncs the vendor folder to match vendor.json
	govendor sync

vendorUpdate: ## updates the vendor folder
	govendor fetch +missing +external
	govendor remove +unused

help: ## displays this message
	@grep -E '^[a-zA-Z_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

watch/%: ## helper for running tasks on file change (requires watchdog)
	watchmedo shell-command -i "./.git/*;./.data/*;./.bin/*" --recursive --ignore-directories --wait --command "$(MAKE) $*"
