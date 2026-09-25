# syntax=docker/dockerfile:1.7
#
# admin-web Dockerfile（Vite + React 18 + TypeScript → nginx）
#
# 多阶段：
#   1. builder : node:20-alpine 装依赖并执行 `npm run build`
#                Vite 默认输出到 dist/
#   2. runtime : nginx:1.27-alpine 提供静态托管（listen :80；compose 映射 :8092）
#
# 镜像名 doctors/admin-web:latest；docker-compose.deploy.yml 引用此 tag。

# ---------- Builder ----------
FROM node:20-alpine AS builder

WORKDIR /src

# 先拷贝 lock 文件
COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund

# 拷贝源码
COPY . .

# 构建：tsc -b && vite build；产物在 dist/
RUN npm run build

# ---------- Runtime ----------
FROM nginx:1.27-alpine AS runtime

RUN rm -f /etc/nginx/conf.d/default.conf

COPY --from=builder /src/dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# nginx:1.27-alpine 自带 wget
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -q --spider http://127.0.0.1/index.html || exit 1

CMD ["nginx", "-g", "daemon off;"]