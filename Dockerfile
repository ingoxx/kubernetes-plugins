# 使用一个最小化的基础镜像，如 scratch 或 alpine
FROM gotec007/go:v1.22 AS builder
ARG TARGETOS
ARG TARGETARCH

# 设置工作目录
WORKDIR /app

# 复制您编译好的调度器可执行文件
COPY scheduler-plugin-custom .

# (可选) 复制您的调度器配置，如果需要自定义配置的话
# COPY scheduler-config.yaml /etc/kubernetes/scheduler-config.yaml

# 定义容器启动时运行的命令
ENTRYPOINT ["/app/scheduler-plugin-custom"]