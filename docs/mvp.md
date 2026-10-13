# Specmetry MVP 范围（第一阶段：Local MVP）

> 状态：Draft v0.3 · 2026-10-13（依据 [codex-review-001](reviews/codex-review-001.md) 与 [codex-review-002](reviews/codex-review-002.md) 修订）
> 本文是范围的唯一依据：**不在"范围内"清单中的功能，一律不做。** 需求见 [prd.md](prd.md)，验收项见 [acceptance-criteria.md](acceptance-criteria.md)，实施顺序见 [implementation-plan.md](implementation-plan.md)。

## 1. MVP 目标

一位独立开发者在不注册账号、Specmetry 不联网的前提下，能对任意阶段的项目完成一个**可信的交付闭环**：

> 初始化 → 提取基准并人工确认 → 人工确认交付范围 → 现状盘点 → 在交付快照上验证 → 人工验收 → 得到可复现的交付判定

在任何时刻都能区分三件事：

- **"Agent 说做完了"**：进度；
- **"在交付快照上检查通过了"**：`ready_for_review`；
- **"在这个快照上，范围内全部已验收"**：`ready`。

并且系统**不会**因为旧代码上的验收、其他 worktree 的结果、不完整的范围或过期的读取，给出错误的 `ready`。

## 2. 范围内

| 能力 | 内容 | 需求 |
|---|---|---|
| 项目初始化 | `setup`、`init`、阶段检测、候选来源、可恢复的进度、默认交付目标 | FR-INIT-* |
| 来源 | 登记、变化检测、人工确认、人工移除、来源集合哈希 | FR-SRC-* |
| 基准 | 带前置条件的提案、引用校验、审核、版本、冲突、问题、父需求与 AC 的依赖、检查配置的约束、绑定修订的引用 | FR-BL-* |
| 交付目标与范围 | 单一目标、快照解析、人工确认范围、过期检测 | FR-TGT-* |
| Worktree | 共享身份、隔离上下文、在调用者所在的 worktree 中执行、按 ref 扫描 | FR-WT-* |
| 现状盘点 | 带范围的 audit、覆盖率、"未知"进度 | FR-INV-01、02、04、07 |
| 三层事实 | 带前置条件的 Claim、绑定上下文的**观测级**检查执行（标注证据等级）、带前置条件（含 `seen_decision_id`）和风险确认的人工决定（`normal` / `manual` / `exception`）、沿用规则（MVP 中因为只有观测级证据，实际不会沿用） | FR-CLM-*、FR-EV-01 ~ 06、FR-EV-08、FR-ACC-* |
| 推导与交付判定 | AC 交付状态、进度、需求状态、目标交付状态、差距、保守规则、`seq` 排序 | FR-ST-01 ~ 09 |
| 持续更新 | Claude Code Hooks（含失败事件和 PowerShell）、延迟预算、`sync` 的比较起点、降级模式 | FR-COL-01 ~ 05、07 ~ 09 |
| Agent Skill | `init`、`status`、`sync`、`goals`、`audit` | FR-SK-* |
| Dashboard | 概览、基准、审核与范围、需求详情与决定、差距、健康、多项目、页面过期处理 | FR-UI-01 ~ 06、09 ~ 13 |
| 隐私与存储 | 统一契约、`purge` 与墓碑、权限（含 Windows）、迁移协调、版本保护 | FR-PRV-*、FR-STO-01 ~ 03、05 |
| 交换契约 | `schemas/` 中的全部 JSON Schema | [data-model.md §8](data-model.md#8-交换格式) |

**P1（可选，不阻塞里程碑）**：FR-INIT-06、FR-INIT-08、FR-INV-03、FR-INV-05、FR-INV-06、FR-STO-04、FR-UI-14。

## 3. 范围外（明确不做）

| 不做 | 说明 |
|---|---|
| 云端托管与计费 | 账号、身份认证、远程同步、收费 |
| 团队权限管理 | 多用户、角色、多人验收、客户视图 |
| Agent 任务调度 | 启动、停止、分派 Agent；从 Dashboard 触发 Agent |
| 复杂代码影响分析 | 不做按文件或调用关系判断证据是否失效。MVP 用"快照必须完全一致"这一保守规则代替，只对 `check.inputs` 做对象 ID 比较。 |
| 完整终端日志采集 | 不保存检查输出、工具输出和命令文本 |
| 自动合并或部署 | Specmetry 从不写 git，也不部署 |
| 内置 LLM | 核心不调用任何模型 |
| 工作区（未提交改动）的正式交付判定 | 不做工作区指纹；有未提交改动的执行不合格 |
| 隔离执行器（`snapshot_isolated` 证据） | MVP 之后实现（FR-EV-09、TD-24）。MVP 只产生观测级证据，`--isolation required` 返回退出码 6；普通项目不依赖 Docker 或其他隔离后端 |
| 多个交付目标 / 多交付阶段 | 每个项目只有 1 个 `active` 目标（PD-19） |
| 从 Dashboard 执行检查或修改交付目标 | 只能在 CLI 中执行 |
| P2 项 | 趋势图、完整活动页、交付报告与基准导出、Stop 时提醒 sync、粘贴文本来源、导入 CI 证据 |
| 多仓库项目、非 git 版本控制 | 非 git 目录只能降级使用，永远不会得到 `ready` |
| IDE 插件、桌面应用、TUI | — |

## 4. 支持矩阵

| 维度 | MVP |
|---|---|
| Agent | **Claude Code**：正式支持（Skill + Hooks + 拦截）。Codex 等：尽力而为（AGENTS.md + CLI），沙箱限制和配置指引见 [architecture.md §12.4](architecture.md#124-兼容性矩阵)。 |
| 操作系统 | Windows 10/11、macOS 13+、Linux x86_64 / arm64 |
| 版本控制 | git |
| 浏览器 | 当前版本的 Chrome、Edge、Firefox、Safari |
| 网络 | 核心离线；Agent 和检查命令的网络需求由用户自行负责（[architecture.md §10.5](architecture.md#105-网络边界)） |

## 5. 关键用户流程

1. **首次使用**：`setup` → `/specmetry init` → 审核基准并确认范围 → `/specmetry init`（全量 audit 与检查）→ 查看交付状态。
2. **日常循环**：Agent 在任意 worktree 中工作 → `/specmetry sync`（工作区视图）→ 代码合入交付分支后，在交付分支所在的干净 worktree 中运行检查 → `ready_for_review` → 人工验收。
3. **需求变更**：处理来源变化 → `/specmetry goals` → 审核（包括 AC 后续处理和范围重新确认）。
4. **交付**：交付状态为 `ready` 时，判定对象是"快照 `S` + 基准版本 + 范围确认 + `as_of_seq`"，可以复现，并且附带证据等级（MVP 中为观测级）。

## 6. 里程碑

详见 [implementation-plan.md](implementation-plan.md)。概要：

| 里程碑 | 主题 | 依赖 |
|---|---|---|
| M0 | 工程骨架与存储基础（`seq`、迁移、权限、schema 校验） | — |
| M1 | 基准、来源、交付目标与范围 | M0 |
| M2 | 事实、检查执行与交付推导，以及 `purge` | M1 |
| M3 | Hook 采集与 `setup` | M0（实现）、M2（`sync` 联调） |
| M4 | Agent Skill | M2、M3 |
| M5 | Dashboard | M2（API 稳定后，可以与 M3、M4 并行） |
| M6 | 加固与发布 | 全部 |

每个里程碑的退出条件，是 [acceptance-criteria.md](acceptance-criteria.md) 中标注为该里程碑的 **P0** 验收项和反例项全部通过。P1 项不阻塞。

## 7. MVP 完成定义

- [ ] 全部 P0 验收项、全部 CX 反例项、E2E-1 ~ E2E-8 在三个平台上通过
- [ ] 不变量检查（[acceptance-criteria.md §3](acceptance-criteria.md#3-关键不变量)）在所有 E2E 中自动断言通过
- [ ] 隐私：无出站网络（MVP-PRV-01）、C0 canary（MVP-PRV-02）
- [ ] 性能：MVP-ST-08、MVP-COL-01、MVP-UI-04
- [ ] dogfooding 至少 2 周（从 M4 开始）
- [ ] 至少 3 个真实项目（分别处于三种起始阶段）走完首次使用流程
- [ ] README：安装、`setup`、快速上手、数据位置、隐私契约摘要、能力边界
- [ ] PD-10、PD-11 已有结论

## 8. 风险与缓解

| # | 风险 | 缓解 |
|---|---|---|
| R1 | 提取质量随模型波动 | 人工审核；人工确认范围；用录制的输出做确定性测试 |
| R2 | 自审偏差 | Claim 只影响进度；交付必须有快照证据和人工决定 |
| R3 | 测试被削弱 | 检查需要人确认理由；可以声明 `inputs`，有变化时不沿用验收；最终以人工验收为准 |
| R4 | 每次快照前进都要重新验证，成本较高；MVP 没有隔离级证据，自动化 AC 也需要人重新验收 | `check run --all` 只需一条命令；批量验收（FR-UI-14，P1）；隔离执行器上线后，自动化 AC 可以沿用验收（PD-16）；手动 AC 需要重新确认，这是有意为之 |
| R11 | 观测级证据可能对应到执行期间被临时修改过的代码（并行编辑后恢复、HEAD 切换后恢复） | 不沿用；人验收时必须确认 `observed_evidence` 风险；交付判定标为观测级；隔离执行器在 MVP 之后提供 |
| R5 | 检查会弄脏工作区，导致执行不合格 | 文档要求把构建产物加入忽略；`doctor` 提示；执行结果中给出明确的失败类型 |
| R6 | Claude Code Hook 接口变化 | 固定兼容版本；适配器隔离；丢弃计数可见 |
| R7 | 启发式通道识别被绕过 | 如实声明能力边界；拦截；审计记录；Skill 中的禁止事项 |
| R8 | Windows 差异（ACL、shell、进程树） | 三平台 CI；TD-06、TD-22 |
| R9 | 状态词汇增多，造成理解负担 | 目标层面只突出 5 个状态；每个状态都附带"下一步做什么" |
| R10 | 本项目自身范围膨胀 | 以本文为准；新增范围必须先修改本文 |
