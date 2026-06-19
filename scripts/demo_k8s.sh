#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_DIR="${ROOT_DIR}/.demo/k8s/images"
RUN_DIR="${ROOT_DIR}/.demo/k8s"
PID_DIR="${RUN_DIR}/pids"
mkdir -p "${IMAGE_DIR}"
mkdir -p "${PID_DIR}"

GRPC_IMAGE="voting-system/grpcserver:latest"
HTTP_IMAGE="voting-system/httpserver:latest"
GRPC_TAR="${IMAGE_DIR}/grpcserver.tar"
HTTP_TAR="${IMAGE_DIR}/httpserver.tar"
PF_LOCAL_PORT="${PF_LOCAL_PORT:-18080}"
PF_PID_FILE="${PID_DIR}/http-port-forward.pid"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/demo_k8s.sh build
  bash scripts/demo_k8s.sh load
  bash scripts/demo_k8s.sh deploy
  bash scripts/demo_k8s.sh up
  bash scripts/demo_k8s.sh up-nobuild
  bash scripts/demo_k8s.sh forward
  bash scripts/demo_k8s.sh stop-forward
  bash scripts/demo_k8s.sh verify
  bash scripts/demo_k8s.sh url
  bash scripts/demo_k8s.sh logs [http|grpc|redis]
  bash scripts/demo_k8s.sh status
  bash scripts/demo_k8s.sh reset-votes
  bash scripts/demo_k8s.sh down

Description:
  build       构建 grpc/http Docker 镜像
  load        导出并加载镜像到 minikube
  deploy      apply K8s YAML 并等待就绪
  up          执行 build + load + deploy
  up-nobuild  跳过构建，直接 load + deploy
  forward     启动本地端口转发（127.0.0.1:${PF_LOCAL_PORT} -> svc/httpserver-nodeport:8080）
  stop-forward 停止本地端口转发
  verify      接口功能验证（GET -> POST -> GET）
  url         输出 Web 访问地址
  logs        查看日志（默认展示用法；指定服务则 tail -f）
  status      查看 deploy/pod/svc 状态
  reset-votes 清空 Redis 投票数据
  down        删除 K8s 资源
EOF
}

ensure_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "[ERROR] 未找到命令: ${cmd}" >&2
    exit 1
  fi
}

ensure_base_tools() {
  ensure_cmd kubectl
  ensure_cmd minikube
  ensure_cmd docker
}

ensure_cluster_ready() {
  if ! minikube status --format='{{.Host}}' >/dev/null 2>&1; then
    echo "[INFO] 未检测到运行中的 minikube，尝试启动..."
    minikube start
  fi
}

read_pid() {
  local pid_file="$1"
  if [[ -f "${pid_file}" ]]; then
    tr -d '[:space:]' < "${pid_file}"
  fi
}

is_pid_running() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 1
  fi
  kill -0 "${pid}" >/dev/null 2>&1
}

start_port_forward() {
  local pid
  pid="$(read_pid "${PF_PID_FILE}")"
  if is_pid_running "${pid}"; then
    echo "[INFO] 端口转发已运行，PID=${pid}"
    echo "[INFO] Local URL: http://127.0.0.1:${PF_LOCAL_PORT}/"
    return 0
  fi

  echo "[STEP] 启动端口转发..."
  nohup kubectl port-forward svc/httpserver-nodeport "${PF_LOCAL_PORT}:8080" >/dev/null 2>&1 &
  local pf_pid=$!
  echo "${pf_pid}" > "${PF_PID_FILE}"

  sleep 1
  if ! is_pid_running "${pf_pid}"; then
    echo "[ERROR] 端口转发启动失败，请检查 service 是否存在。" >&2
    exit 1
  fi
  echo "[OK] 端口转发已启动，PID=${pf_pid}"
  echo "[INFO] Local URL: http://127.0.0.1:${PF_LOCAL_PORT}/"
}

stop_port_forward() {
  local pid
  pid="$(read_pid "${PF_PID_FILE}")"
  if ! is_pid_running "${pid}"; then
    rm -f "${PF_PID_FILE}"
    echo "[INFO] 未检测到运行中的端口转发"
    return 0
  fi

  kill "${pid}" >/dev/null 2>&1 || true
  sleep 1
  if is_pid_running "${pid}"; then
    kill -9 "${pid}" >/dev/null 2>&1 || true
  fi
  rm -f "${PF_PID_FILE}"
  echo "[OK] 已停止端口转发"
}

build_images() {
  echo "[STEP] 构建 Docker 镜像..."
  docker build -f "${ROOT_DIR}/deployments/docker/Dockerfile.grpc" -t "${GRPC_IMAGE}" "${ROOT_DIR}"
  docker build -f "${ROOT_DIR}/deployments/docker/Dockerfile.http" -t "${HTTP_IMAGE}" "${ROOT_DIR}"
  echo "[OK] 镜像构建完成"
}

assert_local_images_exist() {
  if ! docker image inspect "${GRPC_IMAGE}" >/dev/null 2>&1; then
    echo "[ERROR] 本地不存在镜像: ${GRPC_IMAGE}" >&2
    echo "[TIP] 先执行: bash scripts/demo_k8s.sh build" >&2
    exit 1
  fi
  if ! docker image inspect "${HTTP_IMAGE}" >/dev/null 2>&1; then
    echo "[ERROR] 本地不存在镜像: ${HTTP_IMAGE}" >&2
    echo "[TIP] 先执行: bash scripts/demo_k8s.sh build" >&2
    exit 1
  fi
}

load_images_to_minikube() {
  echo "[STEP] 导出镜像并加载到 minikube..."
  docker save -o "${GRPC_TAR}" "${GRPC_IMAGE}"
  docker save -o "${HTTP_TAR}" "${HTTP_IMAGE}"
  minikube image load "${GRPC_TAR}"
  minikube image load "${HTTP_TAR}"
  echo "[OK] 镜像已加载到 minikube"
}

deploy_resources() {
  echo "[STEP] 部署 K8s 资源..."
  kubectl apply -f "${ROOT_DIR}/deployments/k8s/redis.yaml"
  kubectl apply -f "${ROOT_DIR}/deployments/k8s/grpcserver.yaml"
  kubectl apply -f "${ROOT_DIR}/deployments/k8s/httpserver.yaml"

  echo "[STEP] 等待服务就绪..."
  kubectl rollout status deploy/redis
  kubectl rollout status deploy/grpcserver
  kubectl rollout status deploy/httpserver
  echo "[OK] 所有 Deployment 就绪"
}

get_web_url() {
  minikube service httpserver-nodeport --url
}

cmd_up() {
  ensure_base_tools
  ensure_cluster_ready
  build_images
  load_images_to_minikube
  deploy_resources
  local url
  url="$(get_web_url)"
  echo "[OK] K8s 演示环境已就绪"
  echo "[INFO] Cluster URL: ${url}"
  start_port_forward
}

cmd_up_nobuild() {
  ensure_base_tools
  ensure_cluster_ready
  assert_local_images_exist
  load_images_to_minikube
  deploy_resources
  local url
  url="$(get_web_url)"
  echo "[OK] K8s 演示环境已就绪（跳过构建）"
  echo "[INFO] Cluster URL: ${url}"
  start_port_forward
}

cmd_verify() {
  ensure_base_tools
  local url="http://127.0.0.1:${PF_LOCAL_PORT}"
  start_port_forward

  echo "[VERIFY] before:"
  curl -s "${url}/api/results" && echo

  echo "[VERIFY] vote Kubernetes:"
  curl -s -X POST "${url}/api/vote" \
    -H "Content-Type: application/json" \
    -d '{"topic_name":"Kubernetes"}' && echo

  echo "[VERIFY] after:"
  curl -s "${url}/api/results" && echo

  echo "[OK] K8s 功能验证完成，可打开浏览器访问: ${url}/"
}

cmd_logs() {
  local target="${1:-}"
  case "${target}" in
    "")
      echo "[INFO] 用法："
      echo "  bash scripts/demo_k8s.sh logs http"
      echo "  bash scripts/demo_k8s.sh logs grpc"
      echo "  bash scripts/demo_k8s.sh logs redis"
      ;;
    http)
      kubectl logs -f deploy/httpserver --tail=200
      ;;
    grpc)
      kubectl logs -f deploy/grpcserver --tail=200
      ;;
    redis)
      kubectl logs -f deploy/redis --tail=200
      ;;
    *)
      echo "[ERROR] 未知日志目标: ${target}" >&2
      usage
      exit 1
      ;;
  esac
}

cmd_status() {
  ensure_base_tools
  kubectl get deploy
  kubectl get pods -o wide
  kubectl get svc
}

cmd_reset_votes() {
  ensure_base_tools
  kubectl exec -it deploy/redis -- redis-cli DEL voting:topics
  echo "[OK] 已清空 voting:topics"
}

cmd_down() {
  ensure_base_tools
  stop_port_forward
  kubectl delete -f "${ROOT_DIR}/deployments/k8s/httpserver.yaml" --ignore-not-found
  kubectl delete -f "${ROOT_DIR}/deployments/k8s/grpcserver.yaml" --ignore-not-found
  kubectl delete -f "${ROOT_DIR}/deployments/k8s/redis.yaml" --ignore-not-found
  echo "[OK] K8s 资源已删除"
}

main() {
  cd "${ROOT_DIR}"
  local cmd="${1:-}"
  case "${cmd}" in
    build)
      ensure_base_tools
      build_images
      ;;
    load)
      ensure_base_tools
      ensure_cluster_ready
      assert_local_images_exist
      load_images_to_minikube
      ;;
    deploy)
      ensure_base_tools
      ensure_cluster_ready
      deploy_resources
      ;;
    up)
      cmd_up
      ;;
    up-nobuild)
      cmd_up_nobuild
      ;;
    forward)
      ensure_base_tools
      start_port_forward
      ;;
    stop-forward)
      stop_port_forward
      ;;
    verify)
      cmd_verify
      ;;
    url)
      ensure_base_tools
      get_web_url
      ;;
    logs)
      ensure_base_tools
      cmd_logs "${2:-}"
      ;;
    status)
      cmd_status
      ;;
    reset-votes)
      cmd_reset_votes
      ;;
    down)
      cmd_down
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
