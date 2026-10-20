# Specmetry

Specmetry 是一个 local-first 的 AI Coding Agent 项目进度与交付监督平台。它以经人确认的项目基准（目标 → 需求 → 验收标准）为尺子，区分 Agent 自述（Claim）、执行证据（Check Run）和人工验收（Decision），并针对确定的代码快照推导交付状态和差距。

> 状态：MVP 开发中，尚未发布。M0（工程骨架与存储基础）与 M1（基准、来源、交付目标与范围）已验收；M2（事实、检查执行与交付推导）的首批（WP2.1 推导引擎、WP2.2 独立的 `ready` 复核器）的纯函数层已验收；WP2.3（Git Reader）的库层已验收（CLI 接入由后续工作包完成）；WP2.4（Claim）的规范已冻结，实现尚未完成；其余 M2 工作包尚未开始。

目前的 CLI 可以：初始化项目（`init`）；登记、查看、确认与移除来源（`source`）；提交提案并人工审核、生成基准版本（`propose`、`proposal`、`review`、`baseline`）；回答或驳回待澄清的问题（`question`）；设置交付目标并确认交付范围（`target`、`scope`）；以及 `version`、`doctor`。

尚未实现：`status`、`gaps` 命令（交付状态与差距的推导引擎已作为纯函数实现，尚未接入 CLI）、检查执行、Agent 自述、人工验收、Hook 与 Skill、Dashboard。

- 产品与设计：[docs/vision.md](docs/vision.md)、[docs/prd.md](docs/prd.md)、[docs/mvp.md](docs/mvp.md)、[docs/architecture.md](docs/architecture.md)
- 里程碑：[docs/implementation-plan.md](docs/implementation-plan.md)；M2 开工准备：[docs/m2-readiness.md](docs/m2-readiness.md)
- 开发指南：[docs/development.md](docs/development.md)

```bash
go build -tags nowebui -o dist/ ./cmd/specmetry
./dist/specmetry version
./dist/specmetry doctor
```
