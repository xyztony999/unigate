# Java / Spring Boot 与 Vue 技术绑定

日期：2026-10-20；业务决策 ADR-028；当前仅设计，未生成业务源码。

## 适配判断

读取引擎随发 java-microservice/profile.yaml 及其 API 规约、vue3-engine/profile.yaml 及前端规约入口后，确认前者包含固定微服务中间件/消息规则，后者包含专用页面引擎强制约定。这些并非用户本次选定的框架本身所必需。

因此在项目 profiles/springboot-vue-web 中建立独立技术 Profile，涵盖 Java / Spring Boot 与 Vue Web 的 API、页面及验证基础规约。未修改插件 payload，也未把单体、数据库、JDK 或精确框架版本伪装为用户已确认的选择。

## 检查及激活结果

所有 CLI 使用插件 payload/engine/cli/bin/engine-cli.js，并显式传 --root D:\Project\unigate。

- check unigate：PASS，11 pass、0 warn、0 fail，退出码 0。
- check springboot-vue-web：PASS，11 pass、0 warn、0 fail，退出码 0；其中部分业务专用检查按技术栈类型跳过，不能理解为额外业务验证。
- use unigate：activated，链为 springboot-vue-web → unigate，4 个 copy 挂载，无 warns，退出码 0。
- 重读根 CLAUDE.md、settings.json、挂载事实、两份 manifest 与注入文件，以及挂载后的规则、技能专题和角色文件；当前会话已显式加载。CLI 的“重启会话后生效”提示针对自动注入，其他依赖自动注入的客户端仍需重启。

## 限制

Profile 检查表示结构和引用有效，不代表 D0-D4 完成或 Gate 1 通过。框架精确版本及数据库尚未锁定，构建命令是后续代码工作区的规范约定，当前没有编译/测试通过的声明。Web 栈不覆盖 Android 原生设备的实际运行能力，移动端须独立设计和实测。

技术参考已查询官方 Spring Boot 系统要求与测试说明、Vue 官方指南；架构知识库记录来源。当前未进入任何代码生成角色，也未启动新的代理或 chat。

## 更新记录

- 2026-10-20：按用户框架选择创建项目内设计栈，完成检查、激活及显式加载。
