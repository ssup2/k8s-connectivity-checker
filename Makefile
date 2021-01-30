.PHONY: test build image
all: test build image

build:
	CGO_ENABLED=0 GO111MODULE=on go build -a -o cnsenter ./cmd/cnsenter/main.go
	CGO_ENABLED=0 GO111MODULE=on go build -a -o kcchecker ./cmd/kcchecker/main.go

image:
	docker build -f Dockerfile -t ssup2/kcchecker:latest .

clean:
	rm -f cnsenter kcchecker

test:
	go test -v ./...
