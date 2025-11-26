# --- Stage 1: Build Environment ---
FROM golang:1.20 AS builder

WORKDIR /app
# 复制 go.mod 和 go.sum，先下载依赖以加速后续构建
COPY go.mod go.sum ./

# 复制所有源码
COPY pkg cmd ./

# 编译命令：生成 Linux 兼容的静态链接可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o custom-scheduler ./cmd/scheduler/gpuSelect/main.go

# --- Stage 2: Final Minimal Image ---
# 使用 scratch 或 distroless 基础镜像以获得最高的安全性
FROM gcr.io/distroless/static-debian12

# 复制编译好的可执行文件
COPY --from=builder /app/custom-scheduler /usr/local/bin/custom-scheduler

# 容器启动命令
ENTRYPOINT ["/usr/local/bin/custom-scheduler"]