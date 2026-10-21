---
name: web-stack-specs
description: Java Spring Boot 与 Vue Web 的 API、页面契约及验证规约。
---

# Web 栈规约入口

阶段前读业务 Profile 和状态索引；业务事实从激活知识库读取，路径按 workspace 槽位解析。

- API 与数据：api-conventions.md。
- Vue PageSpec 与客户端交互：page-specs.md。
- 构建和验证：verification.md。
- 本层约束：../../rules/web-stack-rules.md。

当前业务命令若为 design，仅生成设计文档、Profile 与规范，不生成业务代码。D0 的未确认项不得擅自改写为事实；D1 可提出明确标注的技术方案，按引擎规定确认。
