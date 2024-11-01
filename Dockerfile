FROM golang:1.23 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/mosquitto_exporter

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/mosquitto_exporter /
ENTRYPOINT ["/mosquitto_exporter"]