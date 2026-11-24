#!/usr/bin/env bash
# UniGate 工作人员密码一键重置(主要用于管理员忘记自己的密码):
#   bash /opt/unigate/reset-admin.sh <用户名>
# 生成随机临时密码并打印到当前终端; 该账号全部会话立即失效, 登录后强制改密。
# 读取 /opt/unigate/unigate.env 连接数据库, 全程无 SQL; 以 root 或可读该文件的部署用户运行。
set -euo pipefail

BASE_DIR="${BASE_DIR:-/opt/unigate}"
ENV_FILE="$BASE_DIR/unigate.env"
USERNAME="${1:?用法: reset-admin.sh <用户名>}"

case "$USERNAME" in '' | *[!A-Za-z0-9._@-]*) echo "FATAL: 用户名含非法字符" >&2; exit 1 ;; esac
[ -f "$ENV_FILE" ] || { echo "FATAL: 缺少 $ENV_FILE" >&2; exit 1; }
[ -f "$BASE_DIR/current/unigate" ] || { echo "FATAL: 缺少后端二进制, 请先完成一次部署" >&2; exit 1; }

set -a
. "$ENV_FILE"
set +a
export UNIGATE_RESET_PASSWORD_FOR="$USERNAME"

# 非 web 模式: 重置执行完进程即退出, 不启 HTTP, 也不会与运行中的服务争抢端口
exec "$BASE_DIR/current/unigate" --reset-password-for="$USERNAME" --no-web
