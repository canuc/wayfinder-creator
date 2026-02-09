FROM node:22-alpine AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY --from=frontend /static/ ./static/
RUN CGO_ENABLED=0 go build -o /openclaw-creator .

FROM golang:1.25-alpine AS build-nodeapi
WORKDIR /src
COPY nodeapi/go.mod ./
COPY nodeapi/*.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /openclaw-node-api-amd64 . && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /openclaw-node-api-arm64 .

FROM alpine:3.21
RUN apk add --no-cache ansible openssh-client python3
COPY --from=build /openclaw-creator /usr/local/bin/openclaw-creator
COPY ansible/ /app/ansible/
COPY --from=build-nodeapi /openclaw-node-api-amd64 /app/ansible/roles/clawdbot/files/openclaw-node-api-amd64
COPY --from=build-nodeapi /openclaw-node-api-arm64 /app/ansible/roles/clawdbot/files/openclaw-node-api-arm64
WORKDIR /app
ENV ANSIBLE_DIR=/app/ansible
EXPOSE 8080
ENTRYPOINT ["openclaw-creator"]
