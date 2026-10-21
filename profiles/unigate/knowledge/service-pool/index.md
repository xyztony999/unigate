<!-- kb-generated: 2026-10-20; sources: docs/00-intake.md; assumptions: 0 -->
# 选择服务池索引

ADR-042采用合同，ADR-043采用项目检查；D3按项目依据完成。条目为设计契约，不是已上线服务，不能冒称原版全条目通过。

外部名单已确认支持 Excel/CSV 导入和平台 API 接入（ADR-012），首批为飞书问卷和腾讯文档（ADR-013）；第三方连接器接口不等于内部选择器服务。官方接口与授权的初步核实见 workspace/05-reports/connector-feasibility.md，用户账号权限与文档类型仍待确认。

首轮均用个人账号测试，保留组织账号接入（ADR-014）；所需权限是否可用仍须实际验证，不以账号类型选择替代能力核验。

包括EventOptions、MyGrantList（仅本人）、GrantEventOptions（管理员授权目标薄元数据）、ResourceList/ResourceFields、AccountList（仅管理员）、SourceList、PartyList（负责人核对），合同位于workspace/02-detailed/api/。条目唯一维护common-spec/05-select-service-pool.md。D3的37项机械及16项人工核对通过，D4确认版调用已有选择器，无新增名单访问权；ADR-044已确认七项，Gate1独立检查PASS。条目仍为设计契约，无运行服务。
