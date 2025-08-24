# go makefile

program != basename $$(pwd)
go_version = go1.24.5
version != cat VERSION
latest_release != gh release list --json tagName --jq '.[0].tagName' | tr -d v
gitclean = $(if $(shell git status --porcelain),$(error git status is dirty),$(info git status is clean))
rstms_modules = $(shell awk <go.mod '/^module/{next} /rstms/{print $$1}')
$(program): build

build: fmt
	fix go build

fmt: go.sum
	fix go fmt . ./...

go.mod:
	$(go_version) mod init

go.sum: go.mod
	go mod tidy

install: build
	go install

test: fmt
	go test -v -failfast . ./...

debug: fmt
	go test -v -failfast -count=1 -run $(test) . ./...

release:
	$(gitclean)
	@$(if $(update),gh release delete -y v$(version),)
	gh release create v$(version) --notes "v$(version)"

latest_module_release = $(shell gh --repo $(1) release list --json tagName --jq '.[0].tagName')

update:
	@echo checking dependencies for updated versions 
	@$(foreach module,$(rstms_modules),go get $(module)@$(call latest_module_release,$(module));)

logclean: 
	echo >/var/log/boxen

clean: logclean
	rm -f $(program) *.core 
	rm -f server/certs/*.pem
	go clean

sterile: clean
	which $(program) && go clean -i || true
	go clean
	go clean -cache
	go clean -modcache
	rm -f go.mod go.sum

certs:
	$(if $(SYSTEMROOT),scripts/generate_certs.cmd,scripts/generate_certs)
