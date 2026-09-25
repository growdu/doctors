# syntax=docker/dockerfile:1.7
#
# escort-app Dockerfile（Flutter 3.24 → web → nginx）
#
# 多阶段：
#   1. builder : flutter:3.24  拉 deps + flutter build web --release
#                Flutter web 输出到 build/web/
#   2. runtime : nginx:1.27-alpine 提供静态托管（listen :80；compose 映射 :8080）
#
# 镜像名 doctors/escort-app:latest；docker-compose.deploy.yml 引用此 tag。
#
# 注：Flutter 镜像体积大（~2GB），CI 缓存命中率优化点是 pub cache。

# ---------- Builder ----------
FROM ghcr.io/cirruslabs/flutter:3.24.5 AS builder

WORKDIR /src

# 先拷贝 pubspec，命中 pub-get 缓存
COPY pubspec.yaml pubspec.lock* ./
RUN flutter pub get --offline || flutter pub get

# 再拷贝源码
COPY . .

# 再次 pub get 确保新依赖（lib/ 里加的包会被识别）
RUN flutter pub get

# 构建 web release；产物在 build/web/
RUN flutter build web --release

# ---------- Runtime ----------
FROM nginx:1.27-alpine AS runtime

RUN rm -f /etc/nginx/conf.d/default.conf

COPY --from=builder /src/build/web/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# nginx:1.27-alpine 自带 wget
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -q --spider http://127.0.0.1/index.html || exit 1

CMD ["nginx", "-g", "daemon off;"]