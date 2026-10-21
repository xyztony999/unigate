# 构建及验证规约

## 当前阶段

设计阶段使用 Profile 契约检查、路径/字段/枚举一致性和状态场景检查，不生成业务代码或宣称源码已编译。Gate 1 使用引擎 validation 技能，报告按激活 workspace.reports 落盘。

## 代码阶段约定

计划后端目录为项目根 backend，Maven 构建由 `mvn -f backend/pom.xml verify` 执行；实际版本及 wrapper 在工程初始化时锁定。Vue 目录为 frontend，计划 scripts 提供 `typecheck`、`build`、`test`，通过锁定包管理器运行；未生成工程前不假设命令已可运行。

Java / Spring Boot 使用平台测试支持完成领域规则与必要集成验证；Vue 做类型检查、构建及关键交互验证。新增或修改接口依引擎要求补有意义的 E2E，不用单纯复述实现的测试替代合同验证。

必须覆盖：权限越界、幂等重试、并发约束、状态流转、条件必填、导入及来源冲突、通知失败、断联与接管不确定操作。测试样例来自业务规格，避免在通用栈中硬编码某业务数量。

真实短信、邮件和第三方连接器使用模拟或获授权的测试环境，报告区分模拟通过与真实服务验证。Android 热点、持续运行、相机、主备切换属于独立设备验证，不用 Web 单元测试代替。

参考：[Spring Boot 测试](https://docs.spring.io/spring-boot/reference/testing/index.html)、[Vue 官方快速开始](https://vuejs.org/guide/quick-start.html)。
