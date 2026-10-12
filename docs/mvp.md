# Specmetry MVP 范围（第一阶段：Local MVP）

> 状态：Draft v0.1 · 2026-10-13
> 本文是范围的唯一依据：**不在"范围内"清单中的功能，一律视为不做。** 需求编号见 [prd.md](prd.md)，验收项见 [acceptance-criteria.md](acceptance-criteria.md)。

## 1. MVP 目标

一位独立开发者在**不注册账号、不联网**的前提下，能对任意阶段的项目完成以下闭环：

> 初始化 → 提取目标基准并人工确认 → 现状盘点 → 需求差距分析 → 随 Agent 工作持续更新进度 → 在本地 Dashboard 中查看与验收

并且在任何时刻，都能区分"Agent 说做完了"、"有证据表明通过了"和"我已验收"这三件事。

## 2. 范围内

| 能力 | 包含内容 | 对应需求 |
|---|---|---|
| 项目初始化 | `setup`、`init`、起始阶段检测、候选文档发现、可恢复的初始化进度、worktree 支持 | FR-INIT-*、FR-COL-08 |
| 来源文档 | 登记、哈希、变化检测 | FR-SRC-01/02/04 |
| 目标基准提取与人工确认 | 提案提交与校验、引用校验、`origin` 标识、逐条审核、版本与历史、`material` / `editorial`、冲突检测、待澄清问题 | FR-BL-01 ~ 10 |
| 现状盘点 | `audit` 自述、引用校验、git 提交关联、运行全部检查 | FR-INV-01 ~ 04 |
| 三层事实 | Claim、检查执行（Evidence）、人工验收（Acceptance）、人类专属操作防护 | FR-CLM-*、FR-EV-01 ~ 05、FR-ACC-* |
| 需求差距分析 | 状态推导、差距、冲突标记、交付就绪判定 | FR-ST-01 ~ 05 |
| 持续进度更新 | Claude Code Hooks、spool + ingest、`sync`、无 Hook 降级 | FR-COL-01 ~ 05、07、08 |
| Agent Skill | `init`、`status`、`sync`、`goals`、`audit` | FR-SK-* |
| Web Dashboard | 项目列表、概览、基准、审核队列、需求详情（含验收操作）、差距、设置与健康 | FR-UI-01 ~ 06、09 ~ 12 |
| 隐私 | 无出站网络、脱敏、`purge`、CSP | FR-PRV-01 ~ 03、05 |

**P1 项**（时间不足时可推迟到 MVP 之后）：FR-INIT-06/08、FR-SRC-03、FR-BL-11、FR-INV-05/06、FR-EV-06、FR-ST-06/07、FR-COL-06、FR-UI-07/08、FR-PRV-04。

## 3. 范围外（明确不做）

| 不做 | 说明 |
|---|---|
| 云端功能 | 账号、身份认证、远程同步、托管服务、收费与订阅 |
| 团队协作 | 多用户、角色权限、多人验收、评论、通知、面向客户的只读分享 |
| Agent 调度 | 启动、停止、排队、分派 Agent；自动修复；从 Dashboard 触发 Agent |
| 内置 LLM | Specmetry 本体不调用任何模型 API，也不集成本地模型（见 TD 与 AD-01） |
| Specmetry 解析文档 | 不解析 PDF、DOCX 等文件，由 Agent 阅读 |
| 从 Dashboard 执行命令 | MVP 中 Dashboard 不能触发检查执行 |
| 任务管理 | 排期、工时、看板式任务分派、燃尽图 |
| CI 集成 | 导入 CI 结果（FR-EV-07，P2） |
| 多仓库项目 | 一个项目对应多个仓库；也不支持多个项目共用一个仓库 |
| 多交付阶段 | 里程碑或分期交付（PD-13） |
| 外部来源 | Notion、Google Docs、Jira 等远程来源（可以先粘贴文本作为快照，P1） |
| 非 git 版本控制 | 非 git 目录只降级支持（FR-INIT-09） |
| 基于文件范围的证据失效 | 代码变化不会自动使证据失效（TD-11） |
| IDE 插件、桌面应用、TUI | — |

## 4. 支持矩阵

| 维度 | MVP 支持 |
|---|---|
| Agent | **Claude Code**：Skill + Hooks 完整支持。其他能执行 shell 的 Agent：通过 AGENTS.md 指令 + CLI 降级使用，没有 Hook 采集（PD-05）。 |
| 操作系统 | Windows 10/11、macOS 13+、Linux x86_64 / arm64 |
| 版本控制 | git（系统已安装） |
| 浏览器 | 当前版本的 Chrome、Edge、Firefox、Safari |

## 5. 关键用户流程

初始化流程详见 [prd.md §4.1.1](prd.md#411-初始化流程)，技术时序见 [architecture.md §6](architecture.md#6-关键流程)。

1. **首次使用**：安装二进制 → `specmetry setup` → 在 Claude Code 中执行 `/specmetry init` → 在 Dashboard 审核基准 → 再次执行 `/specmetry init` 完成盘点 → 查看差距。
2. **日常循环**：Agent 工作（Hook 自动记录活动）→ `/specmetry sync` → Dashboard 自动刷新 → 人对"已验证"的 AC 做验收。
3. **需求变更**：登记新来源 → `/specmetry goals` → 审核提案 → 基准升版 → 受影响的 AC 变为 `stale` → 重新运行检查并验收。
4. **交付前**：`/specmetry status` 或打开 Dashboard 概览 → 查看就绪判定和原因清单 → 逐项处理。

## 6. 里程碑

只排顺序，不估工期。每个里程碑的退出条件是：[acceptance-criteria.md §6](acceptance-criteria.md#6-里程碑分配) 中分配给它的验收项全部通过，并且之前里程碑的验收项仍然通过。

| 里程碑 | 内容 | 需提前确定的决策 |
|---|---|---|
| **M0 工程骨架** | Go module、CLI 框架、数据目录、SQLite 与迁移框架、日志、`version` 与 `doctor`、三平台 CI、前端工程骨架与嵌入 | TD-02、TD-04、TD-10、TD-16、PD-10 |
| **M1 基准** | `init`（marker、阶段检测、候选文档）、来源、提案提交与校验、引用校验、交互式审核、版本与历史、Question、编号 | TD-01、TD-14、TD-18、PD-03、PD-04、PD-07、PD-12、PD-13、PD-14 |
| **M2 事实与状态** | Claim、检查执行器、git 扫描、验收（TTY）、状态推导引擎、差距、就绪、`status` / `gaps`、`purge` | TD-05、TD-06、TD-11、TD-13、TD-17、PD-01、PD-02 |
| **M3 采集** | `hook` 入口、spool、ingest、Claude Code 适配器、PreToolUse 拦截、`setup` 安装与卸载、worktree 解析 | TD-03、TD-12、PD-05、PD-06 |
| **M4 Skill** | Skill 包（5 个命令）、交换格式 JSON Schema、Skill 静态检查、在 fixture 上评测；**开始用 Specmetry 跟踪 Specmetry 自身的开发** | TD-20、PD-15 |
| **M5 Dashboard** | HTTP API、各页面、刷新机制、安全头 | TD-07、TD-08、TD-09、TD-19、PD-08、PD-09 |
| **M6 加固与发布** | 端到端场景、性能基准、无出站网络验证、打包分发、README 与快速上手；发布 v0.1.0 | TD-15、PD-11 |

M5 可以在 M2 的 API 稳定后与 M3、M4 并行。

## 7. MVP 完成定义

- [ ] 所有 P0 功能需求在 [acceptance-criteria.md](acceptance-criteria.md) 中对应的验收项全部通过
- [ ] E2E-1 ~ E2E-7 场景在三个平台的 CI 上通过
- [ ] 隐私测试通过：无出站网络（MVP-PRV-01）、canary 测试（MVP-COL-05）
- [ ] 性能指标达标（MVP-ST-08、MVP-COL-01、MVP-UI-04）
- [ ] Specmetry 自身的开发已用 Specmetry 跟踪至少 2 周（dogfooding），并记录发现的问题
- [ ] 至少用 3 个真实项目（分别处于三种起始阶段）走完首次使用流程，并记录反馈
- [ ] README 包含安装、`setup`、`/specmetry init` 快速上手和数据位置说明
- [ ] [prd.md §9](prd.md#9-产品待决策事项) 与 [architecture.md §16](architecture.md#16-技术待决策事项) 中所有待决策事项都已有结论（特别是 PD-10 许可证、PD-11 Local 与 Hosted 的边界）

## 8. 风险与缓解

| # | 风险 | 缓解 |
|---|---|---|
| R1 | 基准提取质量随 Agent 和模型波动 | 人工审核是强制的；来源引用逐字校验；用录制的 Agent 输出做确定性测试；用 PD-15 的阈值做评测 |
| R2 | Agent 自己审自己，盘点结果偏乐观 | 盘点结果只是 Claim（低可信）；建议在新会话中执行 audit；必须有证据和验收才算完成 |
| R3 | 测试被改弱以"通过检查" | 检查命令变更必须经人确认；证据记录命令哈希；最终以人工验收为准；Skill 明确禁止（FR-SK-04）。后续可考虑记录测试文件哈希。 |
| R4 | Claude Code 的 Hook 接口变化 | 适配器隔离；录制真实 payload 作为 fixture；未知字段直接忽略；解析失败时以 0 退出并记录日志 |
| R5 | 用户不愿意定义检查命令 | `UNVERIFIABLE` 差距持续可见；audit 提议检查命令（P1）；手动类 AC 仍可直接验收 |
| R6 | 提取条目过多，导致审核疲劳 | Skill 中的粒度指南；按目标分组批量接受；按置信度排序；先审 `must` |
| R7 | 跨平台差异（shell、路径、进程树终止） | 三平台 CI；TD-06；Windows 上用 Job Object 终止进程树 |
| R8 | 状态词汇过多，用户难以理解 | 界面只突出 6 个展示分组，细分状态放在悬停提示和详情页 |
| R9 | 在客户仓库中留下痕迹 | 用户级安装；marker 写入 `.git/info/exclude`（PD-07） |
| R10 | 本项目自身范围膨胀 | 以本文为准；新增范围必须先修改本文并获得确认 |
