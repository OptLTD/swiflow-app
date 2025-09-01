# 第一阶段：构建 Go 应用
FROM golang:1.24-alpine AS builder

# 设置工作目录和环境变量
WORKDIR /data/src
ENV GOPROXY=https://goproxy.cn

# 安装必要的工具
# RUN apk add --no-cache git make gcc musl-dev

# 复制源代码
COPY ./src-core .

# 下载依赖并构建
RUN go mod download
RUN go build -a -o /data/swiflow-app ./main.go

# 清理不需要的文件
RUN rm -rf /data/src && \
    rm -rf /var/cache/apk/*
    # apk del git make gcc musl-dev && \

# 第二阶段：Python(alpine) + Node + uv
FROM python:3.12-alpine

# 安装 nodejs 和 npm
RUN apk add --no-cache nodejs npm && \
    pip install --upgrade pip && \
    pip install uv && \
    rm -rf /root/.cache

COPY --from=builder /data/swiflow-app /data/swiflow-app

# 设置工作目录和权限
WORKDIR /data
RUN chmod +x /data/swiflow-app

# 设置环境变量
ENV IN_CONTAINER=yes
ENV SWIFLOW_HOME=/home

# 暴露端口和入口点
EXPOSE 11235

RUN node -v && npm -v && python3 --version && pip --version && uv --version

ENTRYPOINT ["/data/swiflow-app", "-m", "serve", "-d", "server mode run"]
