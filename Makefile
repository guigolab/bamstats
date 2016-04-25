.PHONY: build release releaseFlag bench profile deploy clean deepclean

CMD=bamstats
LDFLAGS=

build: cli/$(CMD) cli/$(CMD)-linux

release: prepareRelease cli/$(CMD) cli/$(CMD)-linux

prepareRelease:
	$(eval LDFLAGS=-ldflags "-X github.com/bamstats.PreVersionString=")

cli/$(CMD): cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && go build $(LDFLAGS) -o $(CMD)

cli/$(CMD)-linux: cli/*.go *.go GoDeps/GoDeps.json
	@cd cli && GOOS=linux go build $(LDFLAGS) -o $(CMD)-linux

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof

profile: cpu.prof
	@go tool pprof bamstats.test cpu.prof

deploy: build
	@scp cli/$(CMD)-linux ant:~/bin/$(CMD)

clean:
	@rm -f cli/$(CMD) cli/$(CMD)-linux

deepclean: clean
	@rm bamstats.test cpu.prof
