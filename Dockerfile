FROM golang:1.15.7 as builder
WORKDIR /workspace

# Build binaries
COPY . .
RUN CGO_ENABLED=0 GO111MODULE=on go build -a -o cnsenter cmd/cnsenter/main.go
RUN CGO_ENABLED=0 GO111MODULE=on go build -a -o kcchecker cmd/kcchecker/main.go

# Build image
FROM alpine:3.13.1
RUN apk add nmap-ncat
COPY --from=builder /workspace/cnsenter /usr/bin/cnsenter
COPY --from=builder /workspace/kcchecker /usr/bin/kcchecker

CMD ["kcchecker"]
