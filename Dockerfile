FROM golang:1.22.0-alpine3.19 AS build
RUN apk add -U --no-cache make git
COPY / /src/gptscript
WORKDIR /src/gptscript
RUN make build

FROM alpine AS release
COPY --from=build /src/gptscript/bin /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/gptscript"]
