FROM golang:1.26.4 AS build

RUN mkdir /app

WORKDIR /app

ADD ./cmd/ ./cmd
ADD ./internal ./internal
ADD ./pkg ./pkg
ADD ./go.mod .
ADD ./go.sum .

ARG CMD="api"

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/server /app/cmd/${CMD}/main.go

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/server /server

ADD ./keys ./keys
ADD ./docs ./docs

ENTRYPOINT [ "/server" ]

