# 构建阶段
FROM golang:1.21 AS builder

# 设置工作目录
WORKDIR /app

# 复制当前目录下的所有文件到 /app
COPY . .

# 编译 Go 程序，静态编译为二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 运行阶段，使用最小的 scratch 镜像
FROM scratch

# 复制构建好的二进制文件到 scratch 镜像
COPY --from=builder /app/main /main

# 声明容器使用端口
EXPOSE 8080

# 设置容器启动时执行的程序
ENTRYPOINT ["/main"]