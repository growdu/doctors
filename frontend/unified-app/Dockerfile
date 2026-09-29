# syntax=docker/dockerfile:1.7
#
# unified-app Dockerfile（uni-app + Vue 3 + Vite → nginx）
#
# 多阶段：
#   1. builder : node:20-alpine 装依赖并执行 `pnpm build:h5`
#                uni-app 默认输出到 dist/build/h5
#   2. runtime : nginx:1.27-alpine 提供静态托管（listen :80；compose 映射 :80）
#
# 镜像名 doctors/unified-app:latest；docker-compose.deploy.yml 引用此 tag。
# 对应：plan 2026-09-28-unified-app-v2.md Phase 4

# ---------- Builder ----------
FROM node:20-alpine AS builder

WORKDIR /src

# 先拷贝 lock 文件（pnpm 模式）
COPY package.json pnpm-lock.yaml* .npmrc* ./
RUN corepack enable && corepack prepare pnpm@9 --activate \
    && pnpm install --frozen-lockfile --ignore-scripts || pnpm install --ignore-scripts

# 拷贝源码
COPY . .

# 构建：uni-app 编译 h5 端；产物在 dist/build/h5
RUN pnpm build:h5

# ---------- Runtime ----------
FROM nginx:1.27-alpine AS runtime

RUN rm -f /etc/nginx/conf.d/default.conf

# uni-app h5 产物在 dist/build/h5；nginx root 指向它
COPY --from=builder /src/dist/build/h5/ /usr/share/nginx/html/

# SPA history mode fallback：所有路径 fallback 到 index.html
COPY <<EOF /etc/nginx/conf.d/unified-app.conf
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/javascript;

    # SPA fallback
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # 静态资源长缓存
    location ~* \.(?:js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 30d;
        access_log off;
    }
}
EOF

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1

CMD ["nginx", "-g", "daemon off;"]