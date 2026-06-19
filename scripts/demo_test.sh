#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_DIR="${ROOT_DIR}/.demo/test"
COVER_PROFILE="${REPORT_DIR}/coverage.out"

mkdir -p "${REPORT_DIR}"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/demo_test.sh run
  bash scripts/demo_test.sh unit
  bash scripts/demo_test.sh cover
  bash scripts/demo_test.sh report

Description:
  run    : 运行单元测试并生成覆盖率报告（演示推荐）
  unit   : 仅运行 internal/test 单元测试
  cover  : 运行覆盖率统计（当前项目口径）
  report : 输出 coverage.out 的函数级覆盖率结果
EOF
}

ensure_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "[ERROR] 未找到命令: ${cmd}" >&2
    exit 1
  fi
}

cmd_unit() {
  echo "[STEP] 运行 internal/test 单元测试..."
  go test -v ./internal/test
}

cmd_cover() {
  echo "[STEP] 运行覆盖率统计并生成 ${COVER_PROFILE}..."
  go test -v -coverprofile="${COVER_PROFILE}" ./...
}

cmd_report() {
  if [[ ! -f "${COVER_PROFILE}" ]]; then
    echo "[ERROR] 未找到覆盖率文件: ${COVER_PROFILE}" >&2
    echo "[TIP] 先执行: bash scripts/demo_test.sh cover" >&2
    exit 1
  fi
  echo "[STEP] 输出函数级覆盖率报告..."
  go tool cover -func="${COVER_PROFILE}"
}

cmd_run() {
  cmd_unit
  cmd_cover
  cmd_report
  echo "[OK] 单元测试与覆盖率演示完成"
}

main() {
  cd "${ROOT_DIR}"
  ensure_cmd go

  local cmd="${1:-}"
  case "${cmd}" in
    run)
      cmd_run
      ;;
    unit)
      cmd_unit
      ;;
    cover)
      cmd_cover
      ;;
    report)
      cmd_report
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
