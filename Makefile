.PHONY: build compress prepareRelase release bench profile deploy clean deepclean
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

build: $(BINARIES) bin/$(CMD)

compress: build $(COMPRESSED_BINARIES)

$(ENVS):
	@$(MAKE) bin/"$@"/$(CMD)

prepareBuild: 
	$(eval COMMIT := $(shell git rev-parse --short HEAD))
	$(eval LDFLAGS := -ldflags "-X github.com/guigolab/bamstats.GitCommit=$(COMMIT)")

$(BINARIES): prepareBuild $(CMD_DIR)/*.go */*.go *.go GoDeps/GoDeps.json
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
	$(eval VER := $(shell bin/bamstats --version | cut -d' ' -f3 | sed 's/^/v/'))
	@[[ $(VER) == $(TAG) ]] && git push && git push --tags || echo "Wrong release version"
	@[[ $(VER) == $(TAG) ]] && (github-release release -t $(TAG) $(PRE) -d "$(DESC)" || true) || true
	@[[ $(VER) == $(TAG) ]]	&& $(MAKE) $(COMPRESSED_BINARIES:%=upload-%) || true

bin/$(CMD): bin/$(OS)/$(ARCH)/$(CMD)
	@ln -fs $$PWD/bin/$(OS)/$(ARCH)/$(CMD) bin/$(CMD)

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof

profile: cpu.prof
	@go tool pprof bamstats.test $?

install: prepareBuild $(CMD_DIR)/*.go */*.go *.go GoDeps/GoDeps.json
	@go install $(LDFLAGS) ./$(CMD_DIR)

deploy: bin/linux/amd64/$(CMD)
	@scp $? ant:~/bin/$(CMD)

clean: 
	@rm -rf bin/*

deepclean: clean
	@rm bamstats.test cpu.prof
