FROM alpine:latest

COPY car-notifier.gpt ./car-notifier.gpt

RUN apk update && apk add curl postgresql16-client --no-cache && curl https://get.gptscript.ai/install.sh | sh && mkdir /.cache && chmod 777 /.cache

CMD ["gptscript", "--cache=false", "car-notifier.gpt"]
