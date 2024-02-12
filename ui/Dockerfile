FROM node:18-alpine as builder
ENV NODE_OPTIONS --max_old_space_size=8192
WORKDIR /src
COPY package.json .
COPY yarn.lock .
RUN yarn --pure-lockfile install
COPY . .
ENV NUXT_PUBLIC_APP_VERSION=${NUXT_PUBLIC_APP_VERSION:-dev}
RUN yarn build

FROM node:18-alpine
ENV HOST 0.0.0.0
ENV PORT 80
EXPOSE 80
WORKDIR /src
COPY package.json .
COPY --from=builder /src/.output /src/.output
ENV NUXT_PUBLIC_APP_VERSION=${NUXT_PUBLIC_APP_VERSION:-dev}
CMD ["yarn","start"]
