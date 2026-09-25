# syntax=docker/dockerfile:1.7
#
# review-service Dockerfile
#
# 多阶段构建：
#   1. builder  : golang:1.24-alpine 编译静态二进制
#   2. runtime  : gcr.io/distroless/static-debian12:nonroot
#                  - 无 shell / 无包管理器 / < 30MB
#                  - 内置 /etc/passwd 含 nonroot (UID 65532)
#
# 构建镜像名默认 doctors/<svc>:latest；docker-compose.deploy.yml 引用此 tag。
# 通过 --build-arg SVC=<name> 选择要编译的 cmd 路径。
#
# 注：distroless 镜像无 curl / wget / shell，无法在容器内对 /healthz 做 HTTP 探针。
#     改用 `-healthz` flag：服务进程以独立模式启动 :9090 HTTP server（持续 200 OK），
#     Docker HEALTHCHECK 用 `["/app/server", "-healthz"]` 即可触发。

ARG SVC=review

# ---------- Builder ----------
FROM golang:1.24-alpine AS builder
ARG SVC
WORKDIR /src

# 先 COPY go.mod / go.sum，最大化构建缓存命中
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源码（services / shared；monorepo 单模块）
COPY services/ ./services/
COPY shared/ ./shared/

# 静态二进制（CGO=0）；-trimpath 去绝对路径；-ldflags 去符号表与调试信息
# 使用单行 RUN + && 链，避免 shell 多行续行符差异
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/app ./services/${SVC}/cmd

# ---------- Runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
COPY --from=builder /out/app /app/server

# 默认 HTTP 端口；运行期可通过 DOCTORS_<SVC>_HTTP_ADDR 覆盖
EXPOSE 8080

# distroless:nonroot 已内置 nonroot 用户（UID 65532）
USER nonroot:nonroot

# distroless 无 shell / 无 curl / 无 wget；容器内 HEALTHCHECK 由服务二进制 -healthz 模式承担
# （:9090 独立 HTTP server 持续 200 OK）；fail=3 / start-period=10s 容忍冷启动
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/server", "-healthz"]

ENTRYPOINT ["/app/server"]