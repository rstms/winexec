# go makefile

program != basename $$(pwd)

latest_release != gh release list --json tagName --jq '.[0].tagName' | tr -d v
version != cat VERSION

gitclean := $(if $(shell git status --porcelain),$(error git status is dirty),)

status:
	@echo latest_release: $(latest_release)

build: fmt
	fix go build

fmt: go.sum
	fix go fmt . ./...

go.mod:
	go mod init

go.sum: go.mod
	go mod tidy

install: build
	go install

test:
	fix -- go test -failfast -v .
	fix -- go test -failfast -v ./...

release: build
	$(gitclean)
	gh release create v$(version) --notes "v$(version)"

testclean:
	rm -f testdata/*.out
	rm -f testdata/*.err

clean: testclean
	rm -f $(program)
	go clean

sterile: clean
	go clean -r
	go clean -cache
	go clean -modcache
	rm -f go.mod go.sum
