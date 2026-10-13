# Specmetry

Specmetry 是一个 local-first 的 AI Coding Agent 项目进度与交付监督平台。它以经人确认的项目基准（目标 → 需求 → 验收标准）为尺子，区分 Agent 自述（Claim）、执行证据（Check Run）和人工验收（Decision），并针对确定的代码快照推导交付状态和差距。

> 状态：MVP 开发中。M0（工程骨架与存储基础）已完成，尚无面向用户的功能。

- 产品与设计：[docs/vision.md](docs/vision.md)、[docs/prd.md](docs/prd.md)、[docs/mvp.md](docs/mvp.md)、[docs/architecture.md](docs/architecture.md)
- 里程碑：[docs/implementation-plan.md](docs/implementation-plan.md)
- 开发指南：[docs/development.md](docs/development.md)

```bash
go build -tags nowebui -o dist/ ./cmd/specmetry
./dist/specmetry version
./dist/specmetry doctor
```
