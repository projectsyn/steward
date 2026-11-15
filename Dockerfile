FROM docker.io/golang:1.27 AS build

ARG VERSION

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make test
RUN make build

FROM gcr.io/distroless/static:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931

COPY --from=build /app/steward /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/steward" ]
