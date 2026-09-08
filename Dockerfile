FROM golang:1.24-alpine AS backend-build

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM alpine:3.22 AS backend

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=backend-build /out/api /app/api

USER app
EXPOSE 8080

CMD ["/app/api"]

FROM node:22-alpine AS frontend

WORKDIR /app

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./

ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

EXPOSE 5173

CMD ["npm", "run", "dev"]
