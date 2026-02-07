FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /openclaw-creator .

FROM alpine:3.21
RUN apk add --no-cache ansible openssh-client python3 py3-pexpect
COPY --from=build /openclaw-creator /usr/local/bin/openclaw-creator
COPY ansible/ /app/ansible/
WORKDIR /app
ENV ANSIBLE_DIR=/app/ansible
EXPOSE 8080
ENTRYPOINT ["openclaw-creator"]
