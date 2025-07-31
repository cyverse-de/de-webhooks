FROM golang:1.21

ENV CGO_ENABLED=0

ARG git_commit=unknown
ARG version="2.9.0"
ARG descriptive_version=unknown

LABEL org.cyverse.git-ref="$git_commit"
LABEL org.cyverse.version="$version"
LABEL org.cyverse.descriptive-version="$descriptive_version"

LABEL org.label-schema.vcs-ref="$git_commit"
LABEL org.label-schema.vcs-url="https://github.com/cyverse-de/de-webhooks"
LABEL org.label-schema.version="$descriptive_version"

WORKDIR /src/de-webhooks

COPY . .
RUN go test ./... && \
    go build .

FROM debian:stable-slim

WORKDIR /app

COPY --from=0 /src/de-webhooks/de-webhooks /bin/de-webhooks

ENTRYPOINT ["de-webhooks"]
