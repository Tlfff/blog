# ----------------------- 编译 -----------------------
# 继承的基础镜像,alpine 版本的 golang 镜像
FROM golang:1.25-alpine AS builder 

# 开启代理，加快拉取依赖速度
ENV GOPROXY=https://goproxy.cn,direct

# 设置工作目录
WORKDIR /build

# 先只拷贝依赖清单，利用 Docker 层缓存：依赖没变就不重新下载
COPY go.mod go.sum ./
# 下载依赖
RUN go mod download

# 拷贝全部源码并编译
COPY . .
# 编译可执行文件
# CGO_ENABLED=0，禁用 CGO，不链接任何 C 动态库
# GOOS=linux，指定目标操作系统为 Linux
# -ldflags="-s -w"，移除调试信息和符号表，减小可执行文件大小
# -o blog，输出可执行文件 blog
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o blog

# ----------------------- 运行 -----------------------
# 拉取 alpine 镜像，运行 blog 可执行文件
FROM alpine:3.20

# 安装必要的软件包
# ca-certificates，安装 CA 证书，用于验证 HTTPS 连接
# tzdata，安装时区数据，用于设置时区
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从 builder 镜像复制 blog 可执行文件到 /app/blog 目录
COPY --from=builder /build/blog ./blog
# 从 builder 镜像复制 config.docker.yaml 到 /app/config/ 目录
COPY --from=builder /build/config/config.docker.yaml ./config/config.yaml
# 从 builder 镜像复制 pkg/resource 目录到 /app/pkg/resource/ 目录
COPY --from=builder /build/pkg/resource ./pkg/resource/

# 设置入口点为 blog 可执行文件
ENTRYPOINT [ "./blog" ]
# 设置默认命令为 blog server
CMD [ "server" ]









