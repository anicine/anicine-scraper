.PHONY: run
run: main.go
	@rm -rf *.json *.gz &> /dev/null
	@go run .

.PHONY: build
build: main.go
	./build.sh -c -a both