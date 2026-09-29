#!/usr/bin/env bash
# scripts/gen-api.sh
#
# 用 OpenAPI Generator 从 ../admin-web/openapi/contracts.yaml 生成 dart-dio client 到
# lib/api/generated/。生成的代码可直接在 lib/api/generated/api.dart 引用。
#
# 前置条件（需本地手动安装，无需 CI / build 跑）：
#   dart pub global activate openapi_generator_cli
#   export PATH="$PATH:$HOME/.pub-cache/bin"
#
# 用法（escort-app 根目录）：
#   bash scripts/gen-api.sh
#
# 对应 spec：2026-09-24-order-matching-redesign.md §4.1
#           2026-09-24-escort-app-setup.md §A1-A7

set -euo pipefail

# 切到脚本所在目录（即 escort-app 根）
cd "$(dirname "$0")/.."

INPUT_SPEC="../admin-web/openapi/contracts.yaml"
OUTPUT_DIR="lib/api/generated"

if [[ ! -f "$INPUT_SPEC" ]]; then
  echo "[gen-api] ERROR: input spec not found at $INPUT_SPEC" >&2
  echo "[gen-api] 请确认 admin-web/openapi/contracts.yaml 存在（commit 4caefbb 之后应为 269 行）。" >&2
  exit 1
fi

if ! command -v openapi-generator-cli >/dev/null 2>&1; then
  echo "[gen-api] ERROR: openapi-generator-cli not found in PATH" >&2
  echo "[gen-api] 安装：dart pub global activate openapi_generator_cli" >&2
  echo "[gen-api] 然后 export PATH=\"\$PATH:\$HOME/.pub-cache/bin\"" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

echo "[gen-api] input  = $INPUT_SPEC"
echo "[gen-api] output = $OUTPUT_DIR"
echo "[gen-api] lang   = dart-dio"

openapi-generator-cli generate \
  --input-spec "$INPUT_SPEC" \
  --generator-name dart-dio \
  --output "$OUTPUT_DIR" \
  --additional-properties=generateAliasClasses=false,useEnumExtension=true,generateExamples=false

echo "[gen-api] DONE. Generated client under $OUTPUT_DIR"
echo "[gen-api] 提示：生成的代码与 escort-app/lib/models/order.dart 的手写模型字段应一致"
echo "[gen-api]   （selectedEscortId / escortPendingExpireAt / escortRejectReason 等）。"