FROM docker.io/golang:1.27 AS build

ARG VERSION

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make test
RUN make build

FROM gcr.io/distroless/static:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

COPY --from=build /app/steward /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/steward" ]
