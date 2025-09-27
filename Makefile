# go makefile

include make/osvars.make

# common config
program != basename $$(pwd)
version != cat VERSION
latest_release != gh release list --json tagName --jq '.[0].tagName' | tr -d v
rstms_modules != awk <go.mod '/^module/{next} /rstms/{print $$1}'
gitclean = $(if $(shell git status --porcelain),$(error git status is dirty),$(info git status is clean))
bin_extension = $(if $(windows),.exe,,)
release_binary = $(program)-v$(version)-$(os)-$(os_version)-$(arch)$(bin_extension)


# common targets

$(program): build

build: fmt 
	fix go build . ./...
	go build

fmt: go.sum
	fix go fmt . ./...

go.mod:
	go mod init

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

release-upload:
	cp $(program)$(bin_extension) $(release_binary) && gh release upload v$(latest_release) $(release_binary) --clobber && rm $(release_binary)

latest_module_release = $(shell gh --repo $(1) release list --json tagName --jq '.[0].tagName')

update:
	@echo checking dependencies for updated versions 
	@$(foreach module,$(rstms_modules),go get $(module)@$(call latest_module_release,$(module));)
	curl -L -o cmd/common.go https://raw.githubusercontent.com/rstms/go-common/master/proxy_common_go
	sed <cmd/common.go >server/common.go 's/^package cmd/package server/'

clean:
	rm -f $(program) *.core 
	go clean

sterile: clean
	which $(program) && go clean -i || true
	go clean
	go clean -cache
	go clean -modcache
	rm -f go.mod go.sum
	rm -rf ~/.cache/netboot
	rm -rf cmd/certs/*
	touch cmd/certs/.placeholder
