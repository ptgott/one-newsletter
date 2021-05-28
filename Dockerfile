FROM golang:1.15
COPY . /home/
WORKDIR /home/
# Alpine doesn't ship with the libraries cgo would be dynamically linked to, so
# we disable cgo to create a static binary.  
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a main.go

FROM alpine:3.13
# Alpine container don't include root certs by default. Add these so we can,
# for example, use TLS when sending email
RUN apk add ca-certificates
WORKDIR /root/
COPY --from=0 /home/main .
ENTRYPOINT ["./main"]
