# syntax=docker/dockerfile:1.7
#
# patient-miniapp Dockerfile（uni-app + Vue 3 + uView Plus）
#
# 多阶段：
#   1. builder : node:20-alpine 装依赖并执行 `npm run build:h5`
#                uni-app v3 默认 H5 输出到 dist/build/h5/
#   2. runtime : nginx:1.27-alpine 提供静态托管（listen :80）
#
# 镜像名 doctors/patient-miniapp:latest；docker-compose.deploy.yml 引用此 tag。
#
# 注：uni-app H5 是 SPA（hash 或 history 路由），nginx 配置开启 try_files 回退。

# ---------- Builder ----------
FROM node:20-alpine AS builder

WORKDIR /src

# 最大化依赖缓存命中：先拷贝 lock 文件
COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund

# 再拷贝源码
COPY . .

# 构建 H5 bundle；产物在 dist/build/h5/
RUN npm run build:h5

# ---------- Runtime ----------
FROM nginx:1.27-alpine AS runtime

# nginx 默认配置（包含 mime.types / conf.d）
# 移除 default.conf，避免与我们的配置冲突
RUN rm -f /etc/nginx/conf.d/default.conf

COPY --from=builder /src/dist/build/h5/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# nginx:1.27-alpine 自带 wget；HEALTHCHECK 用 wget 探 /index.html
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -q --spider http://127.0.0.1/index.html || exit 1

# nginx -g "daemon off;" 是官方镜像默认 CMD
CMD ["nginx", "-g", "daemon off;"]