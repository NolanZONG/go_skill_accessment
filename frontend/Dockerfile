# syntax=docker/dockerfile:1.6

# ---- build ----
FROM node:20-bookworm-slim AS build
WORKDIR /app
ARG VITE_API_URL=http://localhost:5007
ENV VITE_API_URL=$VITE_API_URL
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# ---- runtime ----
FROM nginx:1.27-alpine AS runtime
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
