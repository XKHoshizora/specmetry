# AGENTS.md

本文件为在本仓库工作的 AI coding agent（Codex、Claude Code 等）提供指引。

## 项目

**Specmetry**：一个 local-first 的 AI Coding Agent 项目进度与交付监督平台。它用经人确认的项目基准（目标 → 需求 → 验收标准）作为尺子，区分三层事实——Agent 自述（Claim）、执行证据（Check Run）、人工验收（Decision）——并针对一个确定的代码快照推导交付状态和差距。

## 当前阶段

**MVP 开发阶段。设计已通过 Codex 审查 003（GO）；M0（工程骨架与存储基础）已于 2026-10-14 验收。M1 尚未开工。**

除非用户明确要求开始某个里程碑，否则不要开始后续里程碑。开工条件见 `docs/implementation-plan.md §2`；开发环境与工程约定见 `docs/development.md`。

## 文档地图

| 文档 | 内容 |
|---|---|
| `docs/vision.md` | 定位、问题、原则 |
| `docs/prd.md` | 术语、权限矩阵、功能需求、Skill 与 CLI、非功能需求、**产品待决策事项（PD-xx）** |
| `docs/mvp.md` | **范围的唯一依据** |
| `docs/architecture.md` | 组件、信任边界、流程、接口、**隐私与数据保留契约（§10，唯一规范来源）**、数据生命周期、Hook 与兼容性、**技术待决策事项（TD-xx）** |
| `docs/data-model.md` | 实体、生命周期、**推导决策表（§6，规范来源）**、版本前置条件 |
| `schemas/` | **交换契约的规范来源**（JSON Schema 2020-12） |
| `docs/acceptance-criteria.md` | 验收项（`MVP-*`）、反例（`CX-*`）、E2E、不变量；每项都标注了里程碑 |
| `docs/implementation-plan.md` | 开工门槛、设计细化截止点、里程碑与依赖 |
| `docs/review-resolution.md` | 审查意见的处理记录 |
| `docs/reviews/` | 外部审查原文（只读） |
| `docs/development.md` | 开发环境、目录结构、常用命令、工程约定 |
| `docs/licensing.md` | 许可证与第三方声明方案（PD-10，待确认） |

## 不可违背的设计约束

1. **不接管 Agent。** 不启动、不调度、不控制任何 Agent。
2. **核心不调用 LLM，不发出出站网络请求，没有遥测。**
3. **三层事实严格区分。**
   - Agent 通道写入的事实只有 Proposal（待审核）和 Claim；此外，它只能初始化项目、登记来源、触发已确认的检查。
   - 确认基准、确认范围、确认或移除来源、设置交付目标、回答或驳回问题、修改片段保存模式、验收，都是**人类专属操作**。这由启发式的通道识别和拦截来保护，目的是防止误执行，**不能证明操作者身份**。
4. **交付判定只针对交付目标的快照提交，证据分级。**
   - 观测级（`observed`）只保证开始和结束时与快照一致，不能被说成"快照已验证"；隔离级（`snapshot_isolated`）需要私有快照、外部强制只读，以及执行前后的验证，MVP 不提供。
   - 沿用只接受隔离级证据；无法判断时给出保守结论。
   - Claim 不能单独产生 `accepted` 或 `ready`；手动 AC 的 `done` 自述只会把它标为"等待人工查看"。
5. **状态只能推导**，不提供任何设置状态的接口。
6. **先后只由 `seq` 决定**，不用时间戳。关键写入携带读取时的版本（验收决定还要携带 `seen_decision_id`），不一致时拒绝（退出码 5 / HTTP 409），客户端不自动重试。
7. **隐私契约**（architecture §10）：
   - 永不持久化：源码与文件内容、diff、任意原始命令文本、工具输出与检查输出、环境变量的值、提示词、完整对话、transcript、密钥。
   - Hook 只写入 `spool-event.v1` 白名单中的字段。
   - 自由文本命中密钥规则时，拒绝写入，不做替换。
8. **项目身份共享，执行上下文按 worktree 隔离。**

## 工作规则

- **范围纪律**：以 `docs/mvp.md` 为准。不加入云端、计费、团队权限、Agent 调度、复杂代码影响分析、完整终端日志、自动合并或部署、内置 LLM。
- **未决问题**：查 PD-xx 和 TD-xx。不自行拍板；发现新的未决问题，就新增条目，或询问用户。
- **一致性**：修改设计时，同步更新文档、`schemas/` 和验收项；保持术语与 `docs/prd.md §2` 一致；保持需求与验收项的追溯关系。
- **语言**：设计文档用中文；代码、标识符、CLI 输出、提交信息用英文（待 PD-09 确定）。

## 技术栈（建议，见 architecture §14）

Go（单一二进制）、SQLite（WAL，候选驱动 `modernc.org/sqlite`）、React + TypeScript + Vite（内嵌到二进制中）。

## 编码约定（开始写代码后适用）

- `internal/domain/derive` 是纯函数包，测试先行；每条决策规则都要有表驱动测试和对应的 CX 反例。
- 测试代码中维护一个独立的 `ready` 复核器，不复用推导引擎。
- 事实类表只追加。
- 所有写入类输入，以及面向 Agent 的输出（`init`、`status`、`gaps`、`check-run`、`error`），必须通过 `schemas/` 校验；面向人的读取模型的 schema，在对应模块开工前补齐。修改格式时，先改 schema，并同步更新 `schemas/examples/`。

## 常用命令

详见 `docs/development.md`。

```bash
sh scripts/check.sh                 # 与 CI 相同的全部检查
go test -tags nowebui ./...         # Go 测试（不含前端）
go vet -tags nowebui ./...
golangci-lint run ./...             # v2.14.0，配置见 .golangci.yml
golangci-lint fmt ./...
cd web && npm ci && npm run check   # 前端：格式、lint、类型检查、测试、构建
```

没有构建 `web/dist` 时，Go 命令需要 `-tags nowebui`。
