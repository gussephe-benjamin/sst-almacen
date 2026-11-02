# Imagen única: la API en Go sirve también el frontend compilado.

# 1) Frontend
FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2) Backend
FROM golang:1.27-alpine AS api
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# 3) Ejecución
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=America/Lima \
    APP_ENV=production \
    PORT=8080 \
    STATIC_DIR=/app/frontend/dist \
    UPLOAD_DIR=/data/uploads
WORKDIR /app/backend
COPY --from=api /out/api /app/bin/api
COPY --from=web /src/frontend/dist /app/frontend/dist
EXPOSE 8080
CMD ["/app/bin/api"]
