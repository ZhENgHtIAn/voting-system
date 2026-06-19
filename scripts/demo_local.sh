#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="${ROOT_DIR}/.demo/local"
LOG_DIR="${RUN_DIR}/logs"
PID_DIR="${RUN_DIR}/pids"

GRPC_PID_FILE="${PID_DIR}/grpcserver.pid"
HTTP_PID_FILE="${PID_DIR}/httpserver.pid"
REDIS_PID_FILE="${PID_DIR}/redis.pid"
REDIS_MANAGED_FILE="${PID_DIR}/redis.managed"

GRPC_LOG_FILE="${LOG_DIR}/grpcserver.log"
HTTP_LOG_FILE="${LOG_DIR}/httpserver.log"
REDIS_LOG_FILE="${LOG_DIR}/redis.log"

mkdir -p "${LOG_DIR}" "${PID_DIR}"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/demo_local.sh up
  bash scripts/demo_local.sh down
  bash scripts/demo_local.sh status
  bash scripts/demo_local.sh verify
  bash scripts/demo_local.sh logs [grpc|http|redis]

Description:
  up     : 本地启动 Redis + grpcserver + httpserver（非 K8s）
  down   : 停止由本脚本启动的本地服务
  status : 查看本地服务进程和 API 状态
  verify : 执行一次功能验证（GET -> POST -> GET）
  logs   : 查看日志，默认输出三者日志路径；可 tail 指定服务日志
EOF
}

is_pid_running() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 1
  fi
  kill -0 "${pid}" >/dev/null 2>&1
}

read_pid() {
  local pid_file="$1"
  if [[ -f "${pid_file}" ]]; then
    tr -d '[:space:]' < "${pid_file}"
  fi
}

wait_for_port() {
  local host="$1"
  local port="$2"
  local timeout_secs="$3"
  local i=0
  while (( i < timeout_secs )); do
    if (echo >/dev/tcp/"${host}"/"${port}") >/dev/null 2>&1; then
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  return 1
}

ensure_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "[ERROR] 未找到命令: ${cmd}" >&2
    exit 1
  fi
}

start_redis_if_needed() {
  ensure_cmd redis-cli
  if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
    echo "[INFO] 检测到已有 Redis(127.0.0.1:6379)，复用现有实例。"
    return 0
  fi

  ensure_cmd redis-server
  echo "[INFO] 启动本地 Redis..."
  redis-server --port 6379 --save "" --appendonly no > "${REDIS_LOG_FILE}" 2>&1 &
  local redis_pid=$!
  echo "${redis_pid}" > "${REDIS_PID_FILE}"
  echo "true" > "${REDIS_MANAGED_FILE}"

  if ! wait_for_port "127.0.0.1" "6379" 15; then
    echo "[ERROR] Redis 启动超时，请查看日志: ${REDIS_LOG_FILE}" >&2
    exit 1
  fi
  echo "[INFO] Redis 启动成功，PID=${redis_pid}"
}

start_grpcserver() {
  local existing_pid
  existing_pid="$(read_pid "${GRPC_PID_FILE}")"
  if is_pid_running "${existing_pid}"; then
    echo "[INFO] grpcserver 已运行，PID=${existing_pid}"
    return 0
  fi

  echo "[INFO] 编译 grpcserver..."
  go build -o "${RUN_DIR}/grpcserver" ./cmd/grpcserver

  echo "[INFO] 启动 grpcserver..."
  "${RUN_DIR}/grpcserver" -config "${ROOT_DIR}/configs/grpcserver.yaml" > "${GRPC_LOG_FILE}" 2>&1 &
  local grpc_pid=$!
  echo "${grpc_pid}" > "${GRPC_PID_FILE}"

  if ! wait_for_port "127.0.0.1" "50051" 20; then
    echo "[ERROR] grpcserver 启动超时，请查看日志: ${GRPC_LOG_FILE}" >&2
    exit 1
  fi
  echo "[INFO] grpcserver 启动成功，PID=${grpc_pid}"
}

start_httpserver() {
  local existing_pid
  existing_pid="$(read_pid "${HTTP_PID_FILE}")"
  if is_pid_running "${existing_pid}"; then
    echo "[INFO] httpserver 已运行，PID=${existing_pid}"
    return 0
  fi

  echo "[INFO] 编译 httpserver..."
  go build -o "${RUN_DIR}/httpserver" ./cmd/httpserver

  echo "[INFO] 启动 httpserver..."
  "${RUN_DIR}/httpserver" -config "${ROOT_DIR}/configs/httpserver.yaml" > "${HTTP_LOG_FILE}" 2>&1 &
  local http_pid=$!
  echo "${http_pid}" > "${HTTP_PID_FILE}"

  if ! wait_for_port "127.0.0.1" "8080" 20; then
    echo "[ERROR] httpserver 启动超时，请查看日志: ${HTTP_LOG_FILE}" >&2
    exit 1
  fi
  echo "[INFO] httpserver 启动成功，PID=${http_pid}"
}

cmd_up() {
  ensure_cmd go
  ensure_cmd curl
  start_redis_if_needed
  start_grpcserver
  start_httpserver
  echo "[OK] 本地服务已就绪: http://127.0.0.1:8080/"
  echo "[TIP] 执行验证: bash scripts/demo_local.sh verify"
}

stop_by_pid_file() {
  local name="$1"
  local pid_file="$2"
  local pid
  pid="$(read_pid "${pid_file}")"
  if ! is_pid_running "${pid}"; then
    rm -f "${pid_file}"
    return 0
  fi
  kill "${pid}" >/dev/null 2>&1 || true
  sleep 1
  if is_pid_running "${pid}"; then
    kill -9 "${pid}" >/dev/null 2>&1 || true
  fi
  rm -f "${pid_file}"
  echo "[INFO] 已停止 ${name}"
}

cmd_down() {
  stop_by_pid_file "httpserver" "${HTTP_PID_FILE}"
  stop_by_pid_file "grpcserver" "${GRPC_PID_FILE}"

  if [[ -f "${REDIS_MANAGED_FILE}" ]]; then
    stop_by_pid_file "redis" "${REDIS_PID_FILE}"
    rm -f "${REDIS_MANAGED_FILE}"
  else
    echo "[INFO] Redis 非本脚本启动，未执行停止。"
  fi
}

cmd_status() {
  local grpc_pid http_pid redis_pid
  grpc_pid="$(read_pid "${GRPC_PID_FILE}")"
  http_pid="$(read_pid "${HTTP_PID_FILE}")"
  redis_pid="$(read_pid "${REDIS_PID_FILE}")"

  echo "[STATUS] grpcserver: $([[ -n "${grpc_pid}" ]] && echo "pid=${grpc_pid}" || echo "pid=unknown")"
  echo "[STATUS] httpserver: $([[ -n "${http_pid}" ]] && echo "pid=${http_pid}" || echo "pid=unknown")"
  echo "[STATUS] redis:      $([[ -n "${redis_pid}" ]] && echo "pid=${redis_pid}" || echo "pid=external/unknown")"

  if curl -s "http://127.0.0.1:8080/api/results" >/dev/null 2>&1; then
    echo "[STATUS] http api: healthy"
    curl -s "http://127.0.0.1:8080/api/results" && echo
  else
    echo "[STATUS] http api: unavailable"
  fi
}

cmd_verify() {
  echo "[VERIFY] before:"
  curl -s "http://127.0.0.1:8080/api/results" && echo

  echo "[VERIFY] vote Golang:"
  curl -s -X POST "http://127.0.0.1:8080/api/vote" \
    -H "Content-Type: application/json" \
    -d '{"topic_name":"Golang"}' && echo

  echo "[VERIFY] after:"
  curl -s "http://127.0.0.1:8080/api/results" && echo

  echo "[OK] 本地功能验证完成。可直接打开: http://127.0.0.1:8080/"
}

cmd_logs() {
  local target="${1:-}"
  case "${target}" in
    "")
      echo "[LOG] grpcserver: ${GRPC_LOG_FILE}"
      echo "[LOG] httpserver: ${HTTP_LOG_FILE}"
      echo "[LOG] redis:      ${REDIS_LOG_FILE}"
      ;;
    grpc)
      tail -n 100 -f "${GRPC_LOG_FILE}"
      ;;
    http)
      tail -n 100 -f "${HTTP_LOG_FILE}"
      ;;
    redis)
      tail -n 100 -f "${REDIS_LOG_FILE}"
      ;;
    *)
      echo "[ERROR] 未知日志目标: ${target}" >&2
      usage
      exit 1
      ;;
  esac
}

main() {
  cd "${ROOT_DIR}"
  local cmd="${1:-}"
  case "${cmd}" in
    up)
      cmd_up
      ;;
    down)
      cmd_down
      ;;
    status)
      cmd_status
      ;;
    verify)
      cmd_verify
      ;;
    logs)
      cmd_logs "${2:-}"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
