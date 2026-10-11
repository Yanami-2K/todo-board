FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/todo-board ./cmd/todo-board

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/todo-board /usr/local/bin/todo-board
ENV ADDR=:8080 \
    DATA_DIR=/data \
    TZ=Asia/Shanghai
WORKDIR /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["todo-board"]
