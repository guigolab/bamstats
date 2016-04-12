.PHONY: build bench profile deploy clean deepclean

build: main/bamstats main/bamstats-linux

main/bamstats: main/main.go
	@cd main && go build -o bamstats

main/bamstats-linux: main/main.go
	@cd main && GOOS=linux go build -o bamstats-linux

bench:
	@go test -cpu=1,2,4 -bench . -run NOTHING -benchtime 4s -cpuprofile cpu.prof

profile: cpu.prof
	@go tool pprof bamstats.test cpu.prof

deploy: build
	@scp main/bamstats-linux ant:~/bin

clean:
	@rm main/bamstats*

deepclean: clean
	@rm bamstats.test cpu.prof
