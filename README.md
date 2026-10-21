# UniGate

面向大学新生与家长见面会的报名、同行组检票和统计系统。Java 21 / Spring Boot 3.5.16 / PostgreSQL 18.6，Vue 3 / TypeScript。首版在线运行，支持内置 H5 报名及独立名单检票；手机检票台与横屏平板 kiosk 共用服务端核销。

当前本地代码交付 Gate 2 已通过：后端 47 项、前端 42 项测试，16 项跨栈检查和 8 组真实浏览器检查通过；包括 18 管理页面、CSV 导入及丢失响应后的原操作恢复。真实平台账号、安卓硬件和公网部署按下文边界另验。

## 本地启动

需要 Java 21、Maven、Node.js 22.12+、Python 3.12+。下面命令在项目根执行。本项目已在 Windows、Java 21.0.6、Maven 3.9.16、Node 22.23.3 上验证。

```powershell
npm --prefix frontend ci
python tools/dev/local.py db-start
```

Windows 开发助手会下载并校验隔离的 PostgreSQL 二进制，创建开发库与专用测试库；配置与随机秘密只写入被忽略的 `.local/runtime.json`。数据库仅监听本机，不改系统 PostgreSQL 安装。也可自行配置 PostgreSQL，参考 `.env.example`；`compose.yaml` 提供数据库容器，需 Docker daemon 可用，尚未在本机运行验证。

分别打开两个终端：

```powershell
python tools/dev/local.py backend
```

```powershell
python tools/dev/local.py frontend
```

打开 [UniGate](http://localhost:5173)。首次管理员账号来自 `.local/runtime.json` 的 `bootstrapUsername` / `bootstrapPassword`，首次登录强制改密，随后重新登录。已存在账号不会在重启时重置。本工作区浏览器验收已完成改密，**当前本地验收账号密码在同一文件的 `e2eAdminPassword`**；首次部署的新环境没有该字段。不要分享该配置文件。

停止应用可在各终端按 Ctrl+C；数据库可执行 `python tools/dev/local.py db-stop`。当前数据保存在 `.local/pgdata`，重启后继续使用。

## 已实现范围

- 活动、报名策略、五角色与活动授权；报名、人工审核、随行人数变更、入场凭证与私密查询入口。
- CSV/XLSX 预览、映射、冲突处理和选中行整批提交；飞书多维表格问卷结果及腾讯文档结果表适配，保留个人/组织账号接入配置。
- 终端配对、手机/平板扫码；kiosk 横屏左右布局、默认确认 1 人、分批入场、负责人现场补录/增员与纠错。
- 服务端容量、版本与幂等控制；请求结果未知时查询原操作，不重复计数或重复显示新放行。
- 累计入场统计、受控导出、交付记录、通知任务、审计、到期清理及密钥撤销。

第三方权限、OAuth、邮箱和短信需要部署配置，默认关闭通知，不会把未配置服务显示为投递成功。具体载体、配置和限制见 [后端集成说明](backend/INTEGRATIONS.md)。腾讯文档接入需要已获授权的结果表及稳定响应 ID 列；飞书从已授权的多维表格读取，不能承诺任意问卷链接直接抓取。OAuth 过期后需重新授权，目前没有自动刷新。

## 验证

```powershell
python tools/dev/local.py verify
python -X utf8 tools/verify/check_implementation.py
```

开发助手给 Maven 显式传入专用测试数据库配置，运行真实 PostgreSQL 集成测试，再执行前端类型检查、构建与组件测试。直接运行 Maven 而未提供 `UNIGATE_TEST_DB_*` 会跳过数据库测试，不能算完整验收。测试使用独立 schema，不清空开发库。

真实浏览器脚本 `tools/verify/browser-smoke.mjs` 使用 Edge、运行中的前后端与开发库，创建隔离测试活动并验证报名、组票、横屏 kiosk、分批入场和提交后丢失响应的恢复。它会在首次改密成功后保存本地验收密码，不发送外部通知。运行前安装独立测试依赖：

```powershell
npm --prefix .local/browser install --save-exact --no-audit --no-fund @playwright/test@1.58.2
node tools/verify/browser-smoke.mjs
```

验证结果见 [Gate 2 报告](profiles/unigate/workspace/05-reports/gate2-unigate-2026-10-21.md)。后端覆盖率在 `backend/target/site/jacoco/`；本地浏览器结果和截图在 `.local/`，不含共享交付材料。

## 部署与现场使用

生产前端由 `npm --prefix frontend run build` 输出到 `frontend/dist`；后端由 Maven 输出可执行 jar。使用同源 HTTPS，静态服务器设置 SPA fallback，并将 `/api` 代理到后端；`UNIGATE_ALLOWED_ORIGIN` 与实际地址一致，生产保留默认 Secure Cookie。本地助手关闭 Secure 仅用于本机 HTTP。

保持主密钥稳定，独立保护活动密钥目录及删除墓碑，**恢复旧数据库时不能回滚该目录**。受管文件存于非公开目录。备份覆盖尚需部署方提供证据，清理报告会明确保留未验证状态，不将数据库删除冒充所有副本删除。配置细节见 [后端说明](backend/README.md) 和 [前端说明](frontend/README.md)。

安卓相机需要 HTTPS 安全上下文；设备方向锁定和系统级 kiosk 由设备设置负责。本轮浏览器测试不能替代真实手机/平板、相机和扫码枪验收。首版公网不可用时停止核销，没有 Wi-Fi/蓝牙离线协同；500 人、2–5 台设备仍需现场容量验证。

设计和决策保存在 `profiles/unigate/workspace` 与 `profiles/unigate/knowledge`；当前设计门控 Gate 1 已通过，代码交付门控单独记录。
