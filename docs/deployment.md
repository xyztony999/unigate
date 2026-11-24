# UniGate 生产部署指南(阿里云 ECS · 宝塔面板 · GitHub Actions)

本文覆盖从一台全新阿里云 ECS 到推送 `main` 分支即自动上线的完整流程。日常使用只需要第 4、6 节;第 2、3、5 节是一次性初始化。

## 1. 架构与信任边界

```
浏览器 ──HTTPS 443──▶ Nginx(宝塔)
                      ├── /        → /www/wwwroot/unigate(Vue 静态产物 + SPA fallback)
                      └── /api/    → http://127.0.0.1:8090(原样转发, 不重写路径)
                                        systemd: unigate.service(Go 静态单二进制, 无 JVM)
                                        └── PostgreSQL(宝塔面板安装管理, 仅监听 127.0.0.1:5432)
```

- **GitHub 只保存 SSH 凭据**(`SSH_HOST/PORT/USER/PRIVATE_KEY`);数据库密码、`UNIGATE_MASTER_KEY`、bootstrap 账号等应用密钥只存在服务器的 `/opt/unigate/unigate.env`(权限 600),不经过 GitHub,开源后仓库不含任何生产秘密。
- 后端是 Go 静态单二进制:OpenAPI 合同与 SQL 迁移都以 `go:embed` 内嵌,流水线在 GitHub runners 上交叉编译(Linux amd64),服务器不需要 Java/Maven/Node 等任何运行时,只接收一个可执行文件。
- **端口约定**:程序默认 `PORT=8080`;本部署服务器 8080 已被其他服务占用,统一用 **8090**——`unigate.env` 的 `PORT`、nginx `proxy_pass`、GitHub variable `APP_PORT` 三处必须一致。
- 两条流水线:`ci.yml`(PR 与 main 的验证)、`deploy.yml`(main 推送或手动触发 → 构建 → 部署)。部署任务引用 `production` 环境,可在仓库设置里加人工审批。

## 2. 前提条件

| 项目 | 要求 |
|---|---|
| 阿里云 ECS | 2核2G 即可(Go 单二进制常驻内存约 50MB,无 JVM;本部署即 2核2G/3Mbps),Alibaba Cloud Linux 3 或 Ubuntu 22.04+;内地地域需域名已完成 ICP 备案 |
| 域名 | A 记录指向 ECS 公网 IP(相机扫码与 Secure Cookie 都要求受信任 HTTPS,裸 IP 不可用) |
| 安全组 | 放行 22(SSH)、80、443;5432 与 8090 不对公网放行(仅本机访问) |
| 宝塔面板 | 已安装 Nginx;PostgreSQL 同样经面板软件商店安装(见 3.3) |

## 3. 服务器一次性初始化

以 root(或具备 sudo 权限的用户)SSH 登录执行。数据库采用宝塔面板管理的 PostgreSQL(见 3.3);若改用 Docker 数据库,把仓库 `deploy/docker-compose.yml` 拷到 `/opt/unigate/` 并在同目录写 `.env` 提供 `UNIGATE_DB_PASSWORD`——部署脚本检测到该文件存在才接管 Docker 分支,不存在则自动跳过。

### 3.1 运行用户与目录

```bash
useradd -r -s /usr/sbin/nologin unigate
mkdir -p /opt/unigate/{releases,current,keys,artifacts,backups}
chown unigate:unigate /opt/unigate/keys /opt/unigate/artifacts
chmod 700 /opt/unigate/keys
```

`keys/`(活动数据密钥)与 `artifacts/`(受管加密文件)由后端进程写入;其余目录由部署 SSH 用户写入,`unigate` 只读二进制。

### 3.2 基础工具(无需 Java 运行时)

后端是 Go 静态单二进制,**不需要安装 JRE/JDK,也没有 JVM 调优**:常驻内存约 50MB,2核2G 机器无需为后端加 swap 安全垫(数据库与面板自身的内存开销另计)。只需 curl 供部署健康检查使用:

```bash
# Alibaba Cloud Linux 3 / CentOS 系
yum install -y curl
# Ubuntu / Debian
# apt update && apt install -y curl
```

### 3.3 数据库(宝塔面板安装 PostgreSQL)

1. 宝塔 → 软件商店 → 搜索 PostgreSQL → 安装(版本 ≥13;项目使用 uuid/timestamptz/jsonb 特性,装面板提供的最新版)。
2. 面板 → 数据库 → PostgreSQL → 添加数据库:库名 `unigate`,设置强密码(下一步填入 `unigate.env`)。数据库保持仅本机监听,确认安全组未放行 5432。
3. 连接信息:主机 `127.0.0.1`、端口 `5432`(面板默认),连同库名/用户/密码填入 3.4 的 `unigate.env`。

> 版本提示:本地开发与 CI 固定 postgres:18.6,面板版本随宝塔浮动——不低于 13 即可运行,排障时先报版本差异。仓库 `deploy/docker-compose.yml` 保留 Docker 方案(与开发/CI 完全同版本)作为备选;面板方案下**不要**把该文件拷到 `/opt/unigate/`。

### 3.4 后端环境文件

```bash
cp deploy/unigate.env.example /opt/unigate/unigate.env   # 同样可先 scp 上来
vi /opt/unigate/unigate.env
chmod 600 /opt/unigate/unigate.env
```

必改项:

- `PORT` 模板已设 `8090`(本服务器 8080 被占);若你的服务器 8080 空闲可改回,但须同步 nginx 与 `APP_PORT`;
- `UNIGATE_DB_PASSWORD` 与面板创建 `unigate` 数据库时设置的密码一致;
- `UNIGATE_MASTER_KEY` 用 `openssl rand -base64 32` 生成,**离线抄录一份,一旦投入生产永不可换**(它封装每个活动的数据密钥,丢失即历史数据不可解密);
- `UNIGATE_BOOTSTRAP_PASSWORD` 至少 12 位(首次登录强制改密,重启不覆盖已有账号);
- `UNIGATE_ALLOWED_ORIGIN=https://<你的域名>`。

该文件必须在使用 systemd 单元(`deploy/unigate.service`)启动前存在,否则服务拒绝启动。

### 3.5 宝塔站点与 HTTPS

1. 宝塔 → 网站 → 添加站点:域名填 `<你的域名>`,纯静态,根目录 `/www/wwwroot/unigate`(目录不存在可让面板创建)。
2. 站点 → SSL → Let's Encrypt 申请证书(面板自动续期);确认 DNS 已生效。
3. 站点 → 配置文件:用仓库 `deploy/nginx-unigate.conf` 的 server 块替换域名与路径后整段粘贴保存。注意 `client_max_body_size 12m`(名单导入上传)、`/api/` 原样转发、`proxy_pass` 端口与 `unigate.env` 的 `PORT` 一致(模板为 8090)是必需项;面板后续若重新生成配置,需检查这几段仍在。
4. `nginx -t && systemctl reload nginx`。

> 在面板 UI 中改动"反向代理"等功能会重写配置文件,与本模板冲突——本方案统一用配置文件管理,不要再用面板的反代功能。

## 4. GitHub 仓库配置

Settings → Secrets and variables → Actions:

**Secrets(部署凭据)**

| 名称 | 值 |
|---|---|
| `SSH_HOST` | ECS 公网 IP |
| `SSH_PORT` | SSH 端口(宝塔改过就用改后的,默认 22) |
| `SSH_USER` | `root`,或具备 sudo 权限的部署用户 |
| `SSH_PRIVATE_KEY` | 专用部署私钥全文(见下) |
| `SSH_KNOWN_HOSTS`(可选) | `ssh-keyscan -p <端口> <IP>` 的输出,固定主机指纹(推荐) |

**Variables(非敏感)**

| 名称 | 默认值 | 说明 |
|---|---|---|
| `WEB_ROOT` | `/www/wwwroot/unigate` | 宝塔站点根目录 |
| `APP_PORT` | `8090` | 后端健康检查端口;本服务器 8080 被占故默认 8090,改动时同步 `unigate.env` 的 `PORT` 与 nginx `proxy_pass` |
| `DEPLOY_MODE` | `systemd` | 部署脚本选路:`systemd`=通用 `deploy.sh`(裸机/斗拱面板等,systemd 管进程);`bt`=宝塔 `deploy-bt.sh`(宝塔「Go项目」管进程,面板可见可重启)。本服务器设 `bt` |

**两种部署模式**:CI 把 `deploy.sh` 与 `deploy-bt.sh` 都上传,SSH 远端按 `DEPLOY_MODE` 选跑哪个,代码零改动即可切换面板/裸机。
- `systemd`(默认):`deploy.sh` 装 systemd 单元 + `systemctl restart`,适合非宝塔环境。
- `bt`(宝塔):`deploy-bt.sh` 经面板 Python API 幂等创建/重启「Go项目」(进程以 `unigate` 用户跑在 `APP_PORT`),不碰 systemd。配套一次性初始化:面板 PostgreSQL 建 `unigate` 库、宝塔「**HTML项目**」站点(纯静态前端 + `/api` 反代到 `APP_PORT`,自定义 nginx 放 `vhost/nginx/extension/<域名>/*.conf`,目录需 755)+ 证书。**`unigate.env` 首行须 `set -a`、末行 `set +a`**——宝塔启动脚本以 `source` 加载 env,不加 `set -a` 则变量不 export,Go 进程读不到 `UNIGATE_MASTER_KEY` 等会启动失败。

部署密钥生成与安装:

```bash
ssh-keygen -t ed25519 -f unigate-deploy -N ""          # 本机生成
cat unigate-deploy.pub >> ~/.ssh/authorized_keys        # 追加到服务器部署用户
# 私钥全文粘贴为 secret SSH_PRIVATE_KEY,私钥文件随即删除
```

**环境保护(推荐)**:Settings → Environments → `production` → 可勾选 Required reviewers(部署前人工确认)并限制只有 `main` 分支可部署。

## 5. 部署流水线行为

`deploy.yml` 在 push 到 `main` 或手动触发时:

1. `backend` 作业:起 PostgreSQL 18.6 服务容器,跑 `go build ./... && go vet ./... && go test ./...`(单元 + 集成测试),再 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o unigate ./cmd/unigate` 交叉编译,产物为 Linux 单二进制 `unigate`;
2. `frontend` 作业:`npm ci` → Vitest → 生产构建,打包 `dist`;
3. `deploy` 作业(引用 `production` 环境):打包 payload(二进制 `unigate`、`dist`、`deploy.sh`、`docker-compose.yml`、`unigate.service`、`reset-admin.sh`)经 `scp/ssh` 上传到 `/opt/unigate/releases/<sha>/` 并执行 `deploy.sh`。

`deploy.sh` 在服务器上:按需更新 systemd 单元(compose 文件仅在启用 Docker 数据库时更新),并把 `reset-admin.sh` 固化到 `/opt/unigate/`(不随 release 清理)→ 原子切换前端目录(保留 `.old`)→ 替换后端二进制(保留 `.prev`,并补可执行位)→ `systemctl restart` → 以 `http://127.0.0.1:8090/api/v1/`(端口取 `APP_PORT`)返回任意 HTTP 状态(404/401 即存活)判定健康,90 秒超时(含首次内嵌 SQL 迁移)→ **失败自动回滚二进制与站点目录并退出非零令流水线变红** → 成功则清理旧 release(默认保留 5 份)。部署窗口为重启的数秒级停机(Go 进程启动为亚秒级,无 JVM 预热)。

首次部署完成前先自查:第 3 节各文件就位、面板中 PostgreSQL 正在运行、安全组 80/443 已放行。

## 6. 验收与日常运维

首次上线验收:

```bash
curl -I https://<域名>/                       # 200, index.html no-cache
curl -o /dev/null -w '%{http_code}\n' https://<域名>/api/v1/   # 404/401 即后端连通
```

浏览器登录 `UNIGATE_BOOTSTRAP_USERNAME`,按提示改密;再用手机实测报名与扫码(真机相机行为是浏览器 E2E 覆盖不了的验收项)。

常用命令:

```bash
systemctl status unigate                       # 服务状态
journalctl -u unigate -f                       # 后端日志(Go slog JSON 走 stderr, journal 统一收集)
ls /opt/unigate/releases                       # 已部署版本(对应 git SHA)
bash /opt/unigate/releases/<旧sha>/deploy.sh   # 手动回滚/重放某个版本
bash /opt/unigate/reset-admin.sh <用户名>       # 忘记密码:重置为临时密码(强制改密,该账号会话全失效)
```

数据库每日备份(自行把产物转存异地)。先找到面板安装的 pg_dump 路径:

```bash
command -v pg_dump || ls -d /www/server/pgsql*/bin /www/server/postgresql*/bin 2>/dev/null
# 记下实际路径, 替换下面 cron 里的 /www/server/pgsql/bin
```

```bash
cat > /etc/cron.d/unigate-backup <<'EOF'
0 3 * * * root . /opt/unigate/unigate.env && PGPASSWORD="$UNIGATE_DB_PASSWORD" /www/server/pgsql/bin/pg_dump -h 127.0.0.1 -U "$UNIGATE_DB_USER" "${UNIGATE_DB_URL##*/}" | gzip > /opt/unigate/backups/db-$(date +\%F).sql.gz && find /opt/unigate/backups -name 'db-*.sql.gz' -mtime +14 -delete
EOF
```

**密钥目录警告**:`/opt/unigate/keys` 与其 `.revoked` 删除墓碑必须与数据库**分开**备份;恢复旧数据库时绝不能连同旧密钥目录一起回滚,否则已删除活动会复活。`UNIGATE_MASTER_KEY` 丢失等价于全部历史活动数据丢失,离线备份是唯一兜底。

## 7. 开源前检查清单

- [ ] 仓库无 `.env`、`.local/`、`unigate.env`(已被 `.gitignore` 覆盖,推送前 `git status` 复核);
- [ ] 所有生产秘密只在 GitHub Secrets 与服务器文件中;
- [ ] `production` 环境已配审批/分支限制,`main` 分支保护已启用;
- [ ] `SSH_PRIVATE_KEY` 为部署专用密钥,泄露后只需换密钥对并清理 `authorized_keys`;
- [ ] README 徽章(可选):`![CI](https://github.com/<owner>/unigate/actions/workflows/ci.yml/badge.svg)`。

## 8. 零 SQL 运维手册

数据库选型已定 PostgreSQL(对比过 MySQL/MongoDB:后端 39 处 `FOR UPDATE`、3 处 `SKIP LOCKED`、87 处 `uuid`、51 处 `timestamptz`、25 处 `::jsonb` 均为 PostgreSQL 支柱特性,迁移成本数天级且无对应收益)。本节的立场是:**日常运维没有任何一条需要手写 SQL 的命令**,需要敲命令的场景全部列在下面,可整段复制。

### 8.1 日常(什么都不用做)

| 事项 | 谁在做 |
|---|---|
| 建表/改表 | 二进制内嵌 SQL 迁移,进程启动时自动执行(`ug_schema_migration` 记录版本,等价原 Flyway 行为) |
| 发版 | GitHub Actions(push 到 main) |
| 备份 | `/etc/cron.d/unigate-backup` 每日 03:00 自动 |
| 应用日志 | `journalctl -u unigate -f` |
| 数据库状态 | 宝塔面板 → 数据库 → PostgreSQL 页面;或 `systemctl status pgsql`(服务名以面板注册为准) |

### 8.2 每月一次的健康检查(2 分钟)

```bash
ls -lh /opt/unigate/backups/ | tail -3       # 备份文件存在、今天有新文件、体积非零
systemctl status pgsql --no-pager            # active(服务名以面板注册为准)
df -h /                                       # 磁盘余量(备份自动保留 14 天)
systemctl is-active unigate                   # active
```

### 8.3 灾难恢复(整段复制,替换日期与 psql 路径)

```bash
# 停应用,避免恢复期间写入
systemctl stop unigate

# 在宝塔面板删除数据库 unigate, 再重新创建同名空库(面板操作, 无 SQL)

# 导入备份
. /opt/unigate/unigate.env
gunzip -c /opt/unigate/backups/db-2026-10-22.sql.gz \
  | PGPASSWORD="$UNIGATE_DB_PASSWORD" /www/server/pgsql/bin/psql -h 127.0.0.1 -U "$UNIGATE_DB_USER" "${UNIGATE_DB_URL##*/}"

systemctl start unigate
journalctl -u unigate -n 30 --no-pager        # 无报错即恢复完成
```

**恢复的铁律**(与第 6 节一致):只恢复数据库,**绝不**把 `/opt/unigate/keys` 回滚成旧版本,否则已删除的活动会复活;`UNIGATE_MASTER_KEY` 保持现状不动。

### 8.4 常见问题速查

| 症状 | 处理 |
|---|---|
| 部署后 502/503 | `journalctl -u unigate -n 50`:多为 `unigate.env` 缺失或数据库密码与面板设置不一致 |
| 数据库连不上 | 面板 → 数据库 → PostgreSQL 查看/重启;或 `systemctl restart pgsql`(服务名以面板注册为准),等 1 分钟 |
| 磁盘告警 | `du -sh /opt/unigate/* /www/server/*`;备份已自动清理,勿动面板 PostgreSQL 数据目录 |
| 忘记管理员密码 | `bash /opt/unigate/reset-admin.sh <用户名>`:打印一次性临时密码,该账号全部会话立即失效,登录后强制改密 |
| 想彻底重来 | 在面板删除并重建 `unigate` 数据库会**清空全部数据**,仅限确认放弃所有活动数据时使用 |
