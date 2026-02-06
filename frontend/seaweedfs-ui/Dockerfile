FROM node:25-alpine AS build

WORKDIR /app

COPY package.json package-lock.json .npmrc .nvmrc ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY . ./

ARG VITE_API_URL
ARG VITE_HOST
ARG VITE_FILER_PORT
ARG VITE_MASTER_PORT
ARG VITE_MASTER_PROXY_PATH
ENV VITE_API_URL=$VITE_API_URL \
    VITE_HOST=$VITE_HOST \
    VITE_FILER_PORT=$VITE_FILER_PORT \
    VITE_MASTER_PORT=$VITE_MASTER_PORT \
    VITE_MASTER_PROXY_PATH=$VITE_MASTER_PROXY_PATH
RUN npm run build

FROM nginx:alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 5000

CMD ["nginx", "-g", "daemon off;"]
