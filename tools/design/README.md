# 设计文档检查工具

此目录只维护契约文档校验，不是UniGate后端、前端或业务实现，不启动应用或调用第三方服务。

在项目根运行 `python -X utf8 tools/design/check_contract.py --write-report` 或 `powershell -File tools/design/check.ps1 --write-report`。不带参数只读，带参数仅更新05-reports下D3-project-check.json与D3-check-source.json；不修改阶段/Gate状态，不读取任何账号秘密。

标准库断言核对JSON引用、角色/路由/字段、FL追溯、枚举、重试与身份连续性，并保留内嵌验证技能条目及源文件摘要。它不是完整OpenAPI/JSON Schema验证器，也不是缺失的插件原版check.sh；原版资产和无定义的CH6/GL条目继续列为缺项，不能用工具exit0代替完整D3或Gate1通过。

ADR-043已授权以项目机械＋人工检查作为本项目D3/D4依据，原版缺失记录不再阻塞，但不伪报原版全通过。D3机械结果与人工核对、用户决定由D3-completion.md共同归档，工具单独退出0不能代替人工/用户确认。

页面文档运行`python -X utf8 tools/design/check_pages.py --write-report`，只核对page-contracts.json/角色文档与当前OpenAPI字段、动作、角色、字典和状态，输出D4-project-check.json；不编译Vue或运行相机/设备。D4七项已由ADR-044确认；工具读取确认来源但不自行批准Gate1。

跨阶段运行`python -X utf8 tools/design/check_gate1.py --write-report`（先更新D3/D4检查报告），输出Gate1-cross-stage-check.json，核对19项跨阶段断言并对19份设计产物取SHA256；不自动修改索引或判定Gate1。最终门控还必须人工核对引擎、设计及栈规则，结果见profiles/unigate/workspace/05-reports/gate1-unigate-2026-10-21.md。

ADR-046启动代码阶段后，检查器允许已获授权的backend/frontend和后续状态；不再要求项目永久没有代码。历史Gate1报告保持原快照，续跑默认不带`--write-report`，代码验证另见`tools/verify/`及Gate2报告。
