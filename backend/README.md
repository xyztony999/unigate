# UniGate 后端

Java 21、Spring Boot 3.5.16、Maven、PostgreSQL；Flyway 11.20.3 执行 25 表迁移。前端通过同源 `/api/v1` 访问。合同直接随构建装入原 `openapi-unigate.json`，85 个操作具有输入及事务内成功响应校验，业务服务负责权限、状态和持久化。

首次启动必须配置 `UNIGATE_DB_URL`、`UNIGATE_DB_USER`、`UNIGATE_DB_PASSWORD`、`UNIGATE_MASTER_KEY`（Base64 编码的 32 字节随机秘密）、`UNIGATE_BOOTSTRAP_USERNAME`、`UNIGATE_BOOTSTRAP_PASSWORD`（至少 12 位）。首次账号创建后必须改密再重新登录；已存在账号不会被启动环境覆盖。

开发 HTTP 显式设 `UNIGATE_COOKIE_SECURE=false`，生产默认开启 Secure Cookie，并通过 HTTPS 与同源代理访问；`UNIGATE_ALLOWED_ORIGIN` 设置浏览器实际 Origin。工作人员 Cookie 为 SameSite=Lax 以允许受 state 保护的 OAuth 顶层回调，其余能力 Cookie 为 Strict；所有业务 POST 仍要求 Origin、CSRF，匿名写另校验原 `X-Client-Context-ID`。

会话 CSRF 由部署秘密和会话 ID 派生，重复或并行读取 bootstrap / SessionMe 不会令另一标签页表单失效；升级前仍有效会话的随机 CSRF 摘要继续兼容至会话结束。密码使用带域分离的 SHA-256 预摘要及 bcrypt（成本 12），支持合同允许的长 Unicode 密码，并兼容旧 bcrypt 账号；修改密码立即撤销全部工作人员会话。

```powershell
mvn -f backend/pom.xml verify
java -jar backend/target/unigate-0.1.0.jar
```

真实 PostgreSQL 测试需额外设 `UNIGATE_TEST_DB_URL`、`UNIGATE_TEST_DB_USER`、`UNIGATE_TEST_DB_PASSWORD`；使用专用测试数据库，测试自行创建隔离 schema。未设置测试数据库时集成测试明确跳过，不能将此视为集成验收通过。

`UNIGATE_KEY_DIRECTORY` 指向独立保护的活动数据密钥目录，`UNIGATE_STORAGE_DIRECTORY` 为受管加密文件目录。活动密钥为每活动独立随机 DEK，主密钥只作封装；数据库不保存 DEK。清理写入外置 `.revoked` 墓碑再删除密钥，所有活动访问和解密均检查墓碑。**密钥目录和删除墓碑不能随旧数据库备份回滚**；恢复须保留最新删除依据，并在隔离环境验证后再开放服务。程序不会把尚未验证的备份覆盖标为完成，清理报告的 BACKUPS 范围持续显示失败及原因。

通知和平台均为显式部署配置；默认不实际发送邮件或短信。飞书/腾讯只读取官方允许域名及部署白名单资源，仍需真实个人/组织账号权限验证。配置、实际支持载体及验证边界见 [INTEGRATIONS.md](INTEGRATIONS.md)。

版本取自官方 [Spring Boot Maven Central](https://repo.maven.apache.org/maven2/org/springframework/boot/spring-boot-starter-parent/3.5.16/) 及 [Flyway Maven Central](https://repo.maven.apache.org/maven2/org/flywaydb/flyway-core/11.20.3/)。Flyway PostgreSQL 模块说明见[官方文档](https://documentation.red-gate.com/fd/postgresql-database-277579325.html)。这属于已确认 Java/Spring Boot 栈的实现版本锁定，取代未验证的 4.1.1 候选。
