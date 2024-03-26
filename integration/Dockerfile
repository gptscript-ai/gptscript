FROM golang:1.22

WORKDIR /gptscript
COPY . .

RUN make build

CMD ["./bin/gptscript", "--cache=false", "integration/integration.gpt"]
