# 使用官方的 Go 镜像作为构建环境
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 将 Go 模块文件复制到工作目录
COPY go.mod .
COPY go.sum .

# 下载依赖
RUN go mod download

# 将项目代码复制到工作目录
COPY . .

# 构建 Go 应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -o todo main.go

# 使用轻量级的 Alpine 镜像作为运行时环境
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/todo .
RUN chmod +x /app/todo
COPY ./etc /app/etc

# 暴露应用程序运行的端口
EXPOSE 8080

# 运行应用程序
CMD ["./todo","-c","etc/todo.yaml"]