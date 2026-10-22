FROM docker.io/golang:1.27 AS build

ARG VERSION

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make test
RUN make build

FROM gcr.io/distroless/static:nonroot@sha256:d62067879de890a1def9da0e7766b7d20810a5e5555cef8506b752437d7a3e88

COPY --from=build /app/steward /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/steward" ]
