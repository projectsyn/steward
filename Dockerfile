FROM docker.io/golang:1.13 as build

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG version=v0.0.1

RUN go build -v \
      -ldflags "-X main.Version=${version}" \
      -o steward .

FROM gcr.io/distroless/static:nonroot

COPY --from=build /app/steward /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/steward" ]
