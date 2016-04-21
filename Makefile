.PHONY: build bench profile deploy clean deepclean

CMD=bamstats

build: cli/$(CMD) cli/$(CMD)-linux

cli/$(CMD): cli/bamstats.go *.go GoDeps/GoDeps.json
	@cd cli && go build -o $(CMD)

cli/$(CMD)-linux: cli/bamstats.go *.go GoDeps/GoDeps.json
	@cd cli && GOOS=linux go build -o $(CMD)-linux

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof

profile: cpu.prof
	@go tool pprof bamstats.test cpu.prof

deploy: build
	@scp cli/$(CMD)-linux ant:~/bin/$(CMD)

clean:
	@rm cli/$(CMD) cli/$(CMD)-linux

deepclean: clean
	@rm bamstats.test cpu.prof
