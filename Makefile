.PHONY: build release releaseFlag bench profile deploy clean deepclean

CMD=bamstats
LDFLAGS=
OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

build: bin/linux/386/$(CMD) \
			 bin/linux/amd64/$(CMD) \
			 bin/darwin/386/$(CMD) \
			 bin/darwin/amd64/$(CMD) \
			 bin/$(CMD)

release: prepareRelease build

prepareRelease:
	$(eval LDFLAGS=-ldflags "-X github.com/bamstats.PreVersionString=")

bin/$(CMD): bin/$(OS)/$(ARCH)/$(CMD)
	@ln -s $$PWD/bin/$(OS)/$(ARCH)/$(CMD) bin/$(CMD)

bin/darwin/386/$(CMD): cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && GOARCH=386 GOOS=darwin go build $(LDFLAGS) -o ../"$@"

bin/linux/386/$(CMD): cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && GOARCH=386 GOOS=linux go build $(LDFLAGS) -o ../"$@"

bin/darwin/amd64/$(CMD): cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && GOARCH=amd64 GOOS=darwin go build $(LDFLAGS) -o ../"$@"

bin/linux/amd64/$(CMD): cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && GOARCH=amd64 GOOS=linux go build $(LDFLAGS) -o ../"$@"

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof

profile: cpu.prof
	@go tool pprof bamstats.test cpu.prof

deploy: build
	@scp bin/linux/amd64/$(CMD) ant:~/bin/$(CMD)

clean:
	@rm -rf bin/*

deepclean: clean
	@rm bamstats.test cpu.prof
