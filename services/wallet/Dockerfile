# syntax=docker/dockerfile:1.7
#
# wallet-service Dockerfile
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
#     healthcheck 实际由 docker-compose.deploy.yml 编排；服务进程依赖 :8080 端口
#     就绪 + 进程存活来判定。当后续为各服务添加 `-healthz` flag 后，可改为：
#         HEALTHCHECK CMD ["/app/server", "-healthz"]

ARG SVC=wallet

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

# distroless 无 shell / 无 curl / 无 wget；容器内 HTTP 探针需服务二进制支持 -healthz
# 这里显式置为 NONE，由 docker-compose.deploy.yml 编排健康检查策略
HEALTHCHECK NONE

ENTRYPOINT ["/app/server"]