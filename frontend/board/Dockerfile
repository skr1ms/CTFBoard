FROM oven/bun:1 AS builder

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .

ARG VITE_API_BASE_URL
ARG VITE_WS_URL
ARG VITE_SSE_URL
ARG VITE_APP_NAME
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_WS_URL=$VITE_WS_URL \
    VITE_SSE_URL=$VITE_SSE_URL \
    VITE_APP_NAME=$VITE_APP_NAME

RUN bun run build

FROM nginxinc/nginx-unprivileged:alpine AS runner

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8000

CMD ["nginx", "-g", "daemon off;"]
