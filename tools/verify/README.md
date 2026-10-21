# 实现验证

`check_implementation.py`：只读检查正式设计 SHA256、85 操作和 33 页面在前后端的逐项投影、DTO/枚举、打包合同、迁移表数与已执行测试证据。带 `--write-report` 写入 `Gate2-cross-stack-check.json`。它不启动服务，不自动判定 Gate 2，也不将静态操作名存在性当成该分支已执行。

`browser-smoke.mjs`：真实 Vue / Spring Boot / PostgreSQL / Edge headless 链路，验证登录、创建活动导航、手机报名、3 人组二维码、横屏 kiosk 激活与 2+1 分批核销；故意丢弃已提交响应，验证原 ID 恢复、非新放行提示及人数不重复累计，再逐页读取 18 个管理页面。页面截图和带时间结果在忽略的 `.local/`。首轮运行还验证首次改密注销及重新登录，后续轮次不会反复改密。

浏览器脚本需要已运行的本地前后端和 PostgreSQL、已安装 Edge，以及根 README 中说明的 Playwright 依赖。每轮新建测试活动；不清空已有开发数据，不连接真实平台账号或对外发送通知。

`python tools/dev/local.py verify` 执行真实 PostgreSQL 的 Maven 集成验证、前端类型检查/构建及 Vitest。覆盖率分别见 JaCoCo 和 `npm --prefix frontend run test:coverage`。缺少专用测试数据库时不能用测试跳过代替通过。

判定依据为活动栈 SW-01—07、引擎 GR-01—06 和确认的业务合同，人工复核及实际边界记录于 `profiles/unigate/workspace/05-reports/gate2-unigate-2026-10-21.md`。设备/真实外部权限/性能/部署备份证据均单独验收，不由本工具出具认证。
