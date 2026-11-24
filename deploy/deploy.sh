#!/usr/bin/env bash
# UniGate 生产部署脚本: 由 GitHub Actions 上传到服务器执行, 也可手动重放:
#   bash /opt/unigate/releases/<sha>/deploy.sh [web-root]
#
# 服务器布局(均位于脚本所在 release 目录的上两级, 默认 /opt/unigate):
#   unigate.env          应用密钥(仅服务器, 不入库)
#   .env                 compose 用的数据库密码
#   docker-compose.yml   PostgreSQL(存在才启用, 见 docs/deployment.md)
#   current/unigate      当前后端(Go 静态单二进制, 无需 Java 运行时)
#   releases/<sha>/      历史版本, 默认保留 5 份
#   keys/ artifacts/     活动密钥与受管文件(严禁随数据库备份回滚)
set -euo pipefail

RELEASE_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_DIR="$(cd "$RELEASE_DIR/../.." && pwd)"
RELEASE_ID="$(basename "$RELEASE_DIR")"
WEB_ROOT="${1:-${WEB_ROOT:-/www/wwwroot/unigate}}"
# 生产服务器 8080 已被占用, 默认 8090; 须与 unigate.env 的 PORT、nginx proxy_pass 一致
APP_PORT="${APP_PORT:-8090}"
SERVICE_NAME="${SERVICE_NAME:-unigate}"
KEEP_RELEASES="${KEEP_RELEASES:-5}"
HEALTH_PATH="${HEALTH_PATH:-/api/v1/}"

log() { printf '[deploy %s] %s\n' "$RELEASE_ID" "$*"; }
run_root() { if [ "$(id -u)" -eq 0 ]; then "$@"; else sudo "$@"; fi; }

# 入参与环境覆盖均做格式校验, 拒绝路径穿越与命令元字符后再参与任何命令
case "$RELEASE_ID" in '' | *[!A-Za-z0-9._-]*) log "FATAL: 非法 release id"; exit 1 ;; esac
case "$SERVICE_NAME" in '' | *[!A-Za-z0-9._-]*) log "FATAL: 非法服务名"; exit 1 ;; esac
case "$APP_PORT" in *[!0-9]* | '') log "FATAL: APP_PORT 必须是数字"; exit 1 ;; esac
case "$WEB_ROOT" in
  /*) ;;
  *) log "FATAL: WEB_ROOT 必须是绝对路径"; exit 1 ;;
esac
case "$WEB_ROOT" in
  *..* | *'$'* | *'`'* | *';'* | *'|'* | *' '*) log "FATAL: WEB_ROOT 含非法字符"; exit 1 ;;
esac
case "$HEALTH_PATH" in
  /*) ;;
  *) log "FATAL: HEALTH_PATH 必须以 / 开头"; exit 1 ;;
esac

[ -f "$RELEASE_DIR/unigate" ] || { log "FATAL: $RELEASE_DIR/unigate 缺失"; exit 1; }
[ -f "$RELEASE_DIR/frontend-dist.tar.gz" ] || { log "FATAL: frontend-dist.tar.gz 缺失"; exit 1; }

# 0. systemd 单元: 内容变化时更新并重载(首次部署自动安装)
if [ "$(id -u)" -eq 0 ] || command -v sudo >/dev/null 2>&1; then
  if ! run_root cmp -s "$RELEASE_DIR/unigate.service" "/etc/systemd/system/$SERVICE_NAME.service" 2>/dev/null; then
    log "安装/更新 systemd 单元 $SERVICE_NAME"
    run_root cp "$RELEASE_DIR/unigate.service" "/etc/systemd/system/$SERVICE_NAME.service"
    run_root systemctl daemon-reload
  fi
fi

# 0.5 运维脚本常驻 BASE_DIR(不随 release 清理)
if [ -f "$RELEASE_DIR/reset-admin.sh" ]; then
  cp -f "$RELEASE_DIR/reset-admin.sh" "$BASE_DIR/reset-admin.sh"
  chmod 750 "$BASE_DIR/reset-admin.sh"
fi

# 1. 数据库: /opt/unigate/docker-compose.yml 存在才接管(选择面板装 PostgreSQL 的部署可删除该文件)
if [ -f "$BASE_DIR/docker-compose.yml" ] && command -v docker >/dev/null 2>&1; then
  log "更新 docker-compose.yml 并确保 PostgreSQL 运行"
  cp -f "$RELEASE_DIR/docker-compose.yml" "$BASE_DIR/docker-compose.yml"
  (cd "$BASE_DIR" && run_root docker compose up -d)
fi

# 2. 前端: 解压并原子切换站点目录, 保留 .old 供回滚(压缩包内容是 dist 的文件, 解到 dist/)
log "发布前端到 $WEB_ROOT"
mkdir -p "$RELEASE_DIR/dist"
tar -xzf "$RELEASE_DIR/frontend-dist.tar.gz" -C "$RELEASE_DIR/dist"
rm -rf "$WEB_ROOT.new"
cp -r "$RELEASE_DIR/dist" "$WEB_ROOT.new"
# 宝塔可能给 .user.ini 加 immutable 属性, 先解锁再删, 否则 rm 失败
run_root chattr -R -i "$WEB_ROOT.old" 2>/dev/null || true
rm -rf "$WEB_ROOT.old"
if [ -d "$WEB_ROOT" ]; then mv "$WEB_ROOT" "$WEB_ROOT.old"; fi
mv "$WEB_ROOT.new" "$WEB_ROOT"
if id www >/dev/null 2>&1; then run_root chown -R www:www "$WEB_ROOT"; fi

# 3. 后端: 暂存新二进制, 保留 .prev 供回滚
log "安装后端二进制"
mkdir -p "$BASE_DIR/current"
rm -f "$BASE_DIR/current/unigate.new"
cp "$RELEASE_DIR/unigate" "$BASE_DIR/current/unigate.new"
# artifact 上传/下载不保留可执行位, 这里显式补上
chmod 755 "$BASE_DIR/current/unigate.new"
if [ -f "$BASE_DIR/current/unigate" ]; then
  cp "$BASE_DIR/current/unigate" "$BASE_DIR/current/unigate.prev"
fi
mv "$BASE_DIR/current/unigate.new" "$BASE_DIR/current/unigate"

# 4. 重启并等待后端应答(404/401 均证明已启动; 启动时的内嵌 SQL 迁移计入等待窗口)
log "重启 $SERVICE_NAME"
run_root systemctl restart "$SERVICE_NAME"

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
  run_root journalctl -u "$SERVICE_NAME" -n 50 --no-pager || true
  rm -rf "$WEB_ROOT.failed"
  if [ -d "$WEB_ROOT.old" ]; then
    mv "$WEB_ROOT" "$WEB_ROOT.failed"
    mv "$WEB_ROOT.old" "$WEB_ROOT"
  fi
  if [ -f "$BASE_DIR/current/unigate.prev" ]; then
    mv "$BASE_DIR/current/unigate.prev" "$BASE_DIR/current/unigate"
    run_root systemctl restart "$SERVICE_NAME"
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

log "部署完成: 前端=$WEB_ROOT 后端=$SERVICE_NAME@$APP_PORT"
