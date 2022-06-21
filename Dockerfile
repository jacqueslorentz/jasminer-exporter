FROM golang:1.18 AS builder
RUN mkdir /build
ADD go.* /build/
ADD *.go /build/
WORKDIR /build

RUN go get github.com/prometheus/client_golang/prometheus
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=0.1.0 -X 'main.build=$(date)'" -o jasminer-exporter . 


FROM scratch
COPY --from=builder /build/jasminer-exporter .
ENTRYPOINT ["./jasminer-exporter"]