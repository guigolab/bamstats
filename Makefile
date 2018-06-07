.PHONY: build compress prepareDev dev prepareRelase release test bench profile deploy clean deepclean
SHELL := /bin/bash

CMD_DIR=cmd/bamstats

CMD:= bamstats
LDFLAGS :=
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

ENVS := \
	linux/386 \
	linux/amd64 \
	darwin/386 \
	darwin/amd64 \

BINARIES := $(ENVS:%=bin/%/$(CMD))
COMPRESSED_BINARIES := $(BINARIES:=.tar.bz2)

build: $(BINARIES)

compress: build $(COMPRESSED_BINARIES)

prepareDev: 
	$(eval COMMIT := $(shell git rev-parse --short HEAD))
	$(eval LDFLAGS := -ldflags "-X github.com/guigolab/bamstats.GitCommit=$(COMMIT)")

dev: prepareDev build

$(ENVS):
	@$(MAKE) bin/"$@"/$(CMD)

$(BINARIES): $(CMD_DIR)/*.go */*.go *.go
	$(eval TERMS := $(subst /, ,"$@"))
	$(eval GOOS := $(word 2, $(TERMS)))
	$(eval GOARCH := $(word 3, $(TERMS)))
	@echo -n Building $(GOOS)-$(GOARCH)...
	@GOARCH=$(GOARCH) GOOS=$(GOOS) go build $(LDFLAGS) -o "$@" ./$(CMD_DIR)
	@echo DONE

$(COMPRESSED_BINARIES): $(BINARIES)
	$(eval BINARY := $(subst .tar.bz2,,"$@"))
	@echo -n Compressing $(BINARY)...
	@tar -jcf "$@" $(BINARY)
	@echo DONE

$(COMPRESSED_BINARIES:%=upload-%): upload-%: prepareRelease
	$(eval FILE := $(subst upload-,,"$@"))
	$(eval INFO := $(subst /, ,"$(subst /$(CMD),,$(FILE))"))
	@github-release upload -t $(TAG) -n $(CMD)-$(TAG)-$(word 2, $(INFO))-$(word 3, $(INFO)) -f $(FILE) || true

prepareRelease: 
	$(eval TAG := $(shell git describe --abbrev=0 --tags))
	$(eval DESC := $(shell git cat-file -p  $(shell git rev-parse $(TAG)) | tail -n+6))
	$(eval LDFLAGS := -ldflags "-X github.com/guigolab/bamstats.PreVersionString=")
	$(eval PRE := -p)

release: prepareRelease compress

pushRelease: release
	$(eval VER := $(shell bin/bamstats --version | cut -d' ' -f3 | sed 's/^/v/'))
	@[[ $(VER) == $(TAG) ]] && git push && git push --tags || echo "Wrong release version"
	@[[ $(VER) == $(TAG) ]] && (github-release release -t $(TAG) $(PRE) -d "$(DESC)" || true) || true
	@[[ $(VER) == $(TAG) ]]	&& $(MAKE) $(COMPRESSED_BINARIES:%=upload-%) || true

test:
	@go test -cpu=1,2 ./annotation
	@go test -cpu=1,2 ./config
	@go test -cpu=1,2 ./sam
	@go test -cpu=1,2 ./stats
	@go test -cpu=1,2 ./utils
	@go test -cpu=1,2 .

race:
	@go test -cpu=1,2 -race ./annotation
	@go test -cpu=1,2 -race ./config
	@go test -cpu=1,2 -race ./sam
	@go test -cpu=1,2 -race ./stats
	@go test -cpu=1,2 -race ./utils
	@go test -cpu=1,2 -race .

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof -memprofile prof.mem

profile: cpu.prof
	@go tool pprof bamstats.test $?

install: prepareDev $(CMD_DIR)/*.go */*.go *.go
	@go install $(LDFLAGS) ./$(CMD_DIR)

ant: prepareDev bin/linux/amd64/$(CMD)
	$(eval BIN := $(word 2, $?))
	@scp $(BIN) ant:~/bin/$(CMD)

clean: 
	@rm -rf bin/*

deepclean: clean
	@rm bamstats.test cpu.prof
