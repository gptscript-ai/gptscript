FROM node:18-alpine as build-ui
RUN apk add -U --no-cache make git
COPY ui /src
WORKDIR /src
RUN make

FROM golang:1.22.0-alpine3.19 AS build-go
RUN apk add -U --no-cache make git
COPY . /src/gptscript
COPY --from=build-ui /src/.output/public /src/gptscript/static/ui
WORKDIR /src/gptscript
RUN make build

FROM alpine AS release
WORKDIR /src
COPY --from=build-go /src/gptscript/bin /usr/local/bin/
COPY --from=build-go /src/gptscript/examples /src/examples
ENTRYPOINT ["/usr/local/bin/gptscript"]
