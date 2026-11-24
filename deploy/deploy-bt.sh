#!/usr/bin/env bash
# UniGate 宝塔面板部署脚本: 由 GitHub Actions 上传到服务器执行, 也可手动重放:
#   bash /opt/unigate/releases/<sha>/deploy-bt.sh [web-root]
#
# 与 deploy.sh(systemd 通用版) 的区别: 后端进程交给宝塔「Go项目」管理(面板可见/可重启),
# 前端站点应为宝塔「HTML项目」(纯静态 + /api 反代到本服务端口)。不用 systemd。
# CI 通过仓库变量 DEPLOY_MODE=bt 选择本脚本(默认 systemd 走 deploy.sh)。
#
# 一次性初始化(见 docs/deployment.md): 面板 PostgreSQL 建 unigate 库、/opt/unigate/unigate.env
# (含 set -a 以便宝塔 source 时 export)、宝塔 HTML 站点 + 证书 + /api 反代。Go项目由本脚本幂等创建。
set -euo pipefail

RELEASE_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_DIR="$(cd "$RELEASE_DIR/../.." && pwd)"
RELEASE_ID="$(basename "$RELEASE_DIR")"
WEB_ROOT="${1:-${WEB_ROOT:-/www/wwwroot/unigate}}"
APP_PORT="${APP_PORT:-8090}"
SERVICE_NAME="${SERVICE_NAME:-unigate}"
KEEP_RELEASES="${KEEP_RELEASES:-5}"
HEALTH_PATH="${HEALTH_PATH:-/api/v1/}"
BT_PYTHON="${BT_PYTHON:-/www/server/panel/pyenv/bin/python}"
PANEL_DIR="${PANEL_DIR:-/www/server/panel}"
RUN_USER="${RUN_USER:-unigate}"

log() { printf '[deploy-bt %s] %s\n' "$RELEASE_ID" "$*"; }
run_root() { if [ "$(id -u)" -eq 0 ]; then "$@"; else sudo "$@"; fi; }

case "$RELEASE_ID" in '' | *[!A-Za-z0-9._-]*) log "FATAL: 非法 release id"; exit 1 ;; esac
case "$SERVICE_NAME" in '' | *[!A-Za-z0-9._-]*) log "FATAL: 非法服务名"; exit 1 ;; esac
case "$APP_PORT" in *[!0-9]* | '') log "FATAL: APP_PORT 必须是数字"; exit 1 ;; esac
case "$WEB_ROOT" in /*) ;; *) log "FATAL: WEB_ROOT 必须是绝对路径"; exit 1 ;; esac
case "$WEB_ROOT" in *..* | *'$'* | *'`'* | *';'* | *'|'* | *' '*) log "FATAL: WEB_ROOT 含非法字符"; exit 1 ;; esac
case "$HEALTH_PATH" in /*) ;; *) log "FATAL: HEALTH_PATH 必须以 / 开头"; exit 1 ;; esac

[ -f "$RELEASE_DIR/unigate" ] || { log "FATAL: $RELEASE_DIR/unigate 缺失"; exit 1; }
[ -f "$RELEASE_DIR/frontend-dist.tar.gz" ] || { log "FATAL: frontend-dist.tar.gz 缺失"; exit 1; }
[ -x "$BT_PYTHON" ] || { log "FATAL: 未找到宝塔面板 Python($BT_PYTHON), 本机非宝塔环境请改用 deploy.sh"; exit 1; }

# 宝塔 Go项目管理助手(ensure=不存在则创建, restart=重启)。经面板 Python API, 与面板 UI 一致。
bt_go() {
  local action="$1" helper="$RELEASE_DIR/.bt-go-helper.py"
  cat > "$helper" <<'PY'
import sys, os
action, name, exe, port, env_file, run_user = sys.argv[1:7]
for p in ("/www/server/panel", "/www/server/panel/class"):
    sys.path.insert(0, p)
os.chdir("/www/server/panel")
import public
from projectModel import goModel
m = goModel.main()
find = m.get_project_find(name)
if action == "ensure":
    if find:
        print("exists"); sys.exit(0)
    get = public.dict_obj()
    get.project_name = name; get.project_exe = exe; get.project_ps = "UniGate Go backend"
    get.bind_extranet = 0; get.domains = []; get.is_power_on = 1
    get.run_user = run_user; get.project_cmd = exe; get.port = int(port); get.env_file = env_file
    r = m.create_project(get); print(r); sys.exit(0 if r.get("status") else 1)
elif action == "restart":
    if not find:
        print("PROJECT_MISSING"); sys.exit(1)
    get = public.dict_obj(); get.project_name = name
    r = m.restart_project(get); print(r); sys.exit(0 if r.get("status") else 1)
PY
  run_root "$BT_PYTHON" "$helper" "$action" "$SERVICE_NAME" "$BASE_DIR/current/unigate" "$APP_PORT" "$BASE_DIR/unigate.env" "$RUN_USER"
  local rc=$?
  rm -f "$helper"
  return $rc
}

# 0. 运维脚本常驻 BASE_DIR
if [ -f "$RELEASE_DIR/reset-admin.sh" ]; then
  cp -f "$RELEASE_DIR/reset-admin.sh" "$BASE_DIR/reset-admin.sh"
  chmod 750 "$BASE_DIR/reset-admin.sh"
fi

# 1. 前端: 解压并原子切换站点目录, 保留 .old 供回滚
log "发布前端到 $WEB_ROOT"
mkdir -p "$RELEASE_DIR/dist"
tar -xzf "$RELEASE_DIR/frontend-dist.tar.gz" -C "$RELEASE_DIR/dist"
rm -rf "$WEB_ROOT.new"
cp -r "$RELEASE_DIR/dist" "$WEB_ROOT.new"
rm -rf "$WEB_ROOT.old"
if [ -d "$WEB_ROOT" ]; then mv "$WEB_ROOT" "$WEB_ROOT.old"; fi
mv "$WEB_ROOT.new" "$WEB_ROOT"
if id www >/dev/null 2>&1; then run_root chown -R www:www "$WEB_ROOT"; fi

# 2. 后端: 暂存新二进制, 保留 .prev 供回滚
log "安装后端二进制"
mkdir -p "$BASE_DIR/current"
rm -f "$BASE_DIR/current/unigate.new"
cp "$RELEASE_DIR/unigate" "$BASE_DIR/current/unigate.new"
chmod 755 "$BASE_DIR/current/unigate.new"
if [ -f "$BASE_DIR/current/unigate" ]; then
  cp "$BASE_DIR/current/unigate" "$BASE_DIR/current/unigate.prev"
fi
mv "$BASE_DIR/current/unigate.new" "$BASE_DIR/current/unigate"

# 3. 确保宝塔 Go项目存在(幂等), 再重启加载新二进制
log "确保宝塔 Go项目 $SERVICE_NAME 存在"
bt_go ensure || { log "FATAL: 创建宝塔 Go项目失败"; exit 1; }
log "重启宝塔 Go项目 $SERVICE_NAME"
bt_go restart || { log "FATAL: 重启宝塔 Go项目失败"; exit 1; }

# 4. 等待后端应答(404/401 均证明已启动; 内嵌 SQL 迁移计入等待窗口)
log "等待 http://127.0.0.1:$APP_PORT$HEALTH_PATH"
healthy=""
code=""
for _ in $(seq 1 45); do
  code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APP_PORT$HEALTH_PATH" || true)"
  case "$code" in
    '' | 000 | 502 | 503) sleep 2 ;;
    *) healthy=1; break ;;
  esac
done

if [ -z "$healthy" ]; then
  log "FATAL: 后端未在时限内就绪, 回滚(最近日志如下)"
  run_root tail -n 50 "/www/wwwlogs/go/$SERVICE_NAME.log" 2>/dev/null || true
  rm -rf "$WEB_ROOT.failed"
  if [ -d "$WEB_ROOT.old" ]; then
    mv "$WEB_ROOT" "$WEB_ROOT.failed"
    mv "$WEB_ROOT.old" "$WEB_ROOT"
  fi
  if [ -f "$BASE_DIR/current/unigate.prev" ]; then
    mv "$BASE_DIR/current/unigate.prev" "$BASE_DIR/current/unigate"
    bt_go restart || true
  fi
  exit 1
fi
rm -f "$BASE_DIR/current/unigate.prev"
log "后端就绪(HTTP $code)"

# 5. 只保留最新 N 个 release
log "清理旧 release(保留 $KEEP_RELEASES 份)"
ls -1t "$BASE_DIR/releases" | tail -n +$((KEEP_RELEASES + 1)) | while read -r rel; do
  rm -rf "$BASE_DIR/releases/$rel"
done

log "部署完成: 前端=$WEB_ROOT 后端=宝塔Go项目 $SERVICE_NAME@$APP_PORT"