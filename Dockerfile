FROM docker.io/golang:1.13 as build

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG version=v0.0.1

RUN go build -v \
      -ldflags "-X main.Version=v0.0.1" \
      -o steward .

RUN update-ca-certificates

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /app/steward /usr/local/bin/

USER 1001

ENTRYPOINT [ "/usr/local/bin/steward" ]
