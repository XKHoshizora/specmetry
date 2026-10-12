# Specmetry 产品需求文档（PRD）

> 状态：Draft v0.1 · 2026-10-13
> 相关文档：[vision.md](vision.md) · [mvp.md](mvp.md) · [architecture.md](architecture.md) · [data-model.md](data-model.md) · [acceptance-criteria.md](acceptance-criteria.md)

功能需求的优先级定义如下：

- **P0**：MVP 必须完成
- **P1**：MVP 应当完成，时间不足时可推迟
- **P2**：MVP 之后再做

## 1. 用户与场景

### 1.1 主要用户

**独立开发者 / 外包开发者**：用 Claude Code 等 AI coding agent 交付客户项目。手上有客户需求文档（可能不完整），代码主要由 Agent 编写。需要随时知道项目还差什么，并能向客户说明进度。

### 1.2 关键场景

| # | 场景 | 期望结果 |
|---|---|---|
| S1 | **只有文档的新项目**：客户发来需求文档，还没写代码 | 得到经确认的基准，所有需求为"未开始"，同时列出文档中的歧义问题 |
| S2 | **中途接入开发中的项目**：已有文档和部分代码 | 得到基准，以及每条验收标准当前的自述和证据状态，并给出差距清单 |
| S3 | **文档不完整 / 几乎只有代码** | 从代码反推候选需求，并明确标识为"反推"；生成大量待澄清问题；只有经人确认后才进入基准 |
| S4 | **日常推进**：Agent 完成一段工作 | 运行 `sync` 后，自述、证据和状态变化被记录；Dashboard 自动刷新 |
| S5 | **需求变更**：客户改需求 | 新文档生成变更提案，确认后基准升版；受影响的旧验收被标记为"需复核" |
| S6 | **交付前检查** | 得到"是否可交付"的判定，并列出每一条不满足的原因 |

## 2. 术语表

| 术语 | 英文 | 定义 |
|---|---|---|
| 项目 | Project | 一个被 Specmetry 跟踪的代码仓库。MVP 中一个项目对应一个仓库根目录。 |
| 来源文档 | Source | 被登记为需求依据的本地文件，如客户需求、PRD、设计文档 |
| 项目基准 | Project Baseline | 经人确认的目标、需求、验收标准的集合，带版本号 |
| 目标 | Goal | 项目要达成的业务或交付目标，用来组织需求 |
| 需求 | Requirement | 一项要交付的能力或约束 |
| 验收标准 | Acceptance Criterion（AC） | 属于某个需求、可以判定真假的条件。**验收的最小单位。** |
| 检查 | Check | 绑定在 AC 上、经人确认的命令，由 Specmetry 执行以产生证据 |
| 提案 | Proposal | 对基准的一组变更建议（新增、修改、退役），经人工审核后才生效 |
| 待澄清问题 | Question | 提取或盘点时发现的歧义或缺失信息 |
| Agent 自述 | Claim | Agent 对某个 AC 实现情况的结构化陈述 |
| 执行证据 | Evidence | Specmetry 执行或观察到的客观记录，如检查结果、提交 |
| 正式验收 | Acceptance | 人对某个 AC 作出的接受、拒收或豁免决定 |
| 活动记录 | Event | Hook 采集的 Agent 会话元数据，如会话起止、命令、被修改的文件路径 |
| 差距 | Gap | 由状态推导出的、阻碍交付的具体问题 |
| 交付就绪 | Delivery Readiness | 所有必需需求已验收或已豁免，并且没有阻塞问题 |
| 通道 | Channel | 操作的发起方式：`agent_cli`（非交互 CLI）、`human_cli`（交互式终端）、`dashboard`、`hook`、`system` |

## 3. 核心概念

### 3.1 三层事实

- **Claim（Agent 自述）**：只能让 AC 到达"自述完成 / 进行中 / 受阻"，**永远不能**让 AC 变为"已验证"或"已验收"。
- **Evidence（执行证据）**：只有 Specmetry 自己执行的、经人确认的检查命令，才能产生 `verifies`（验证性）证据。提交记录只是 `relates`（关联性）证据，可以表示"进行中"，但不能证明正确。Hook 采集的活动记录只用于时间线和会话视图。
- **Acceptance（正式验收）**：只能由人通过交互式终端或 Dashboard 产生。验收与 AC 的版本（revision）绑定。

### 3.2 状态

AC 状态与需求状态由事实**推导**，推导规则见 [data-model.md §6](data-model.md#6-状态推导)。产品界面把细分状态归入 6 个展示分组：

| 展示分组 | 包含的 AC 状态 |
|---|---|
| 未开始 | `not_started` |
| 进行中 | `in_progress` |
| 自述完成 | `claimed` |
| 已验证 | `verified` |
| 已验收 | `accepted`、`waived` |
| 需关注 | `failing`、`regressed`、`rejected`、`stale`、`blocked` |

### 3.3 权限矩阵

| 操作 | Agent（非交互 CLI） | 人（交互式 CLI） | 人（Dashboard） | Specmetry 自动 |
|---|---|---|---|---|
| 初始化项目、登记来源、确认来源变化已处理（`ack`） | ✓ | ✓ | — | — |
| 安装 / 卸载 Skill 与 Hooks（修改 Agent 配置） | ✗ | ✓ | — | — |
| 打开 Dashboard（从不在输出中打印令牌） | ✓ | ✓ | — | — |
| 提交提案（待审核） | ✓ | ✓ | — | — |
| 人工编辑基准（提案并立即应用，`propose --apply`） | ✗ | ✓ | ✓ | — |
| 审核提案（生成基准版本） | ✗ | ✓ | ✓ | — |
| 回答 / 驳回问题 | ✗ | ✓ | ✓ | — |
| 提交 Claim | ✓ | ✓（`manual` 类） | — | — |
| 触发检查 | ✓ | ✓ | ✗（MVP） | — |
| 产生 Evidence | — | — | — | ✓ |
| 验收决定（接受 / 拒收 / 豁免） | ✗ | ✓ | ✓ | — |
| 查看状态与差距 | ✓ | ✓ | ✓ | — |
| 删除项目数据 | ✗ | ✓ | ✗ | — |
| 直接设置状态 | ✗ | ✗ | ✗ | 只能推导 |

"✗"由通道识别与 Hook 拦截共同保证，威胁模型是"防误操作，不防恶意"。详见 [architecture.md §5](architecture.md#5-信任边界与操作通道)。

## 4. 功能需求

### F1 项目初始化

| ID | 需求 | 优先级 |
|---|---|---|
| FR-INIT-01 | `specmetry init` 在仓库根目录创建项目标识 `.specmetry/project.json`，并在本地数据库注册项目。重复执行是幂等的。 | P0 |
| FR-INIT-02 | 检测项目的起始阶段（`docs_only` / `in_development` / `incomplete_docs`），给出建议，并允许用 `--stage` 覆盖 | P0 |
| FR-INIT-03 | 发现候选来源文档，列出供确认，不自动登记 | P0 |
| FR-INIT-04 | `specmetry setup` 安装或检测 Agent Skill 与 Hooks。修改任何 Agent 配置文件前，必须展示变更内容并得到显式确认。 | P0 |
| FR-INIT-05 | 初始化是可恢复的分步流程，进度由数据推导（见 §4.1.1）。再次运行 `/specmetry init` 时，从下一步继续。 | P0 |
| FR-INIT-06 | 默认把 `.specmetry/` 写入 `.git/info/exclude`（只影响本地、不进入版本控制），并在输出中告知；`--no-exclude` 可以关闭；不修改 `.gitignore`（见 PD-07） | P1 |
| FR-INIT-07 | 全流程不需要账号，也不需要网络 | P0 |
| FR-INIT-08 | 检测常见测试命令（如 `go test`、`npm test`、`pytest`），作为提示返回给 Agent | P1 |
| FR-INIT-09 | 在非 git 目录中也能初始化。此时给出警告，并禁用基于 git 的功能。 | P0 |

#### 4.1.1 初始化流程

不同起始阶段的差异：

| 起始阶段 | 判定提示（启发式，可覆盖） | 基准来源 | 现状盘点 |
|---|---|---|---|
| `docs_only` | 有需求或设计文档；源代码文件少于阈值 | 从文档提取 | 不需要对照代码，所有 AC 为 `not_started`；可以预先定义检查 |
| `in_development` | 文档和代码都存在 | 从文档提取；代码中存在但文档未提及的功能，作为 `inferred_from_code` 提案 | Agent 对照代码盘点，运行检查，扫描 git 历史 |
| `incomplete_docs` | 以代码为主，文档缺失或很少 | 文档提取，加上从代码反推（`inferred_from_code`，必须逐条确认），并生成较多 Question | 同 `in_development` |

起始阶段只影响初始化流程的引导，不影响数据模型和状态推导。

初始化进度（`init_progress`）由数据推导，不单独存储。各步骤的条件是累积的：只有前一步满足，后一步才可能成立。

| 步骤 | 完成条件 | 下一步动作 |
|---|---|---|
| 1. `initialized` | 项目已注册 | 确认并登记来源文档 |
| 2. `sources_registered` | 已登记至少 1 个来源，或显式跳过（`init --skip-sources`，适用于仅有代码的项目） | Agent 提取基准，提交提案 |
| 3. `baseline_proposed` | 至少存在 1 个提案 | 人审核提案 |
| 4. `baseline_confirmed` | 至少存在 1 个基准版本 | Agent 盘点现状并运行检查 |
| 5. `inventoried` | 至少完成 1 次 `audit` 自述批次，或显式跳过（`init --skip-inventory`）；`docs_only` 项目在基准确认后自动视为完成 | 进入日常循环 |

完整流程：

1. **一次性准备（每台机器）**：安装 `specmetry` 二进制 → 在终端执行 `specmetry setup` 安装 Skill 与 Hooks（展示改动，确认后写入）。
2. **项目初始化**：在 Agent 中执行 `/specmetry init`。Skill 调用 `specmetry init --json` 获取起始阶段、候选文档和测试命令提示。
3. **登记来源**：Agent 在对话中与用户确认来源列表，然后调用 `specmetry source add`。来源可以位于仓库之外，比如下载目录里的客户文档。
4. **提取基准**：Agent 阅读来源（`in_development` / `incomplete_docs` 还要阅读代码），按 Skill 中的粒度指南生成提案 JSON，然后调用 `specmetry propose`。Specmetry 做 schema 校验和来源引用校验，并立即创建其中的 Question。
5. **人工审核**：Agent 执行 `specmetry dashboard --page review` 为用户打开审核队列，然后停止。用户在审核队列（或终端中的 `specmetry review`）中逐条接受、编辑后接受或拒绝，并回答问题。提交审核即生成基准 v1。只有已确认的条目参与进度计算。
6. **现状盘点**：用户再次执行 `/specmetry init`（或 `/specmetry audit`）。Agent 对照代码逐条提交 `audit` 类 Claim，并附文件和测试引用。随后 `specmetry check run --all` 产生初始证据，Specmetry 汇总 git 历史的基础事实。
7. **差距报告**：Agent 调用 `specmetry gaps --json` 汇总结果，并执行 `specmetry dashboard --page gaps` 为用户打开差距页。

### F2 来源文档

| ID | 需求 | 优先级 |
|---|---|---|
| FR-SRC-01 | 登记本地文件为来源（仓库内或仓库外均可），记录路径、类型、sha256 和大小 | P0 |
| FR-SRC-02 | 检测来源变化（哈希变化、文件缺失），提示"基准可能需要更新"（产生 `SOURCE_CHANGED` 差距） | P0 |
| FR-SRC-03 | 支持把粘贴的文本保存为本地快照文件后登记，例如客户的聊天消息 | P1 |
| FR-SRC-04 | Specmetry 不解析文档语义，由 Agent 阅读文档。Specmetry 只读取文本类文件，用于校验引用。 | P0 |

### F3 基准提取与确认

| ID | 需求 | 优先级 |
|---|---|---|
| FR-BL-01 | 接收结构化提案：对 goal / requirement / criterion 的 `add` / `update` / `retire`，外加 question。做严格的 schema 校验和引用校验，校验不通过则整体拒绝。 | P0 |
| FR-BL-02 | 来源引用校验：引用片段（excerpt）在规范化空白后必须在来源文件中逐字出现，否则标记为 `not_found`；非文本来源标记为 `not_checkable` | P0 |
| FR-BL-03 | 每个条目带 `origin`（`extracted` / `inferred_from_code` / `manual`）和 `confidence`。`inferred_from_code` 条目在所有界面中都有醒目标识。 | P0 |
| FR-BL-04 | 人工审核：逐条接受、编辑后接受或拒绝，支持按目标批量操作。每次提交审核生成一个新的基准版本。 | P0 |
| FR-BL-05 | 只有已确认的条目参与进度、差距和就绪计算 | P0 |
| FR-BL-06 | 基准历史可查询：谁、何时、经哪个通道、改了什么（修改前后对比）、来自哪个提案条目 | P0 |
| FR-BL-07 | 修改已确认的 AC 时，审核者选择 `material` 或 `editorial`。`material` 变更使旧证据和旧验收失效（变为 `stale`）。修改检查命令一律视为 `material`。 | P0 |
| FR-BL-08 | 人可以直接手工新增或编辑条目。手工编辑也走提案路径并立即应用，同样产生版本。 | P0 |
| FR-BL-09 | 提案基于旧版本，并且目标条目在此期间已被修改时，该条目标记为 `conflict`，需要对照当前值重新审核 | P0 |
| FR-BL-10 | 待澄清问题：创建、回答、驳回，可标记为 `blocking`，可关联到需求或 AC | P0 |
| FR-BL-11 | 基准导出为 Markdown 或 JSON | P1 |

### F4 现状盘点

| ID | 需求 | 优先级 |
|---|---|---|
| FR-INV-01 | 基准确认后，`/specmetry audit` 引导 Agent 逐条对照代码，提交 `audit` 类 Claim，并附文件、测试和提交引用 | P0 |
| FR-INV-02 | Claim 中的文件引用由 Specmetry 校验是否存在，提交引用由 git 校验；无效的引用会被标记，但不拒绝 Claim | P0 |
| FR-INV-03 | 盘点时汇总 git 历史的基础事实（提交数、最近提交时间、近期活跃度），供 Agent 盘点参考；此后持续扫描新提交，把引用了基准编号的提交关联为 `relates` 证据。由于编号在基准确认时才分配，关联通常只发生在基准确认之后。 | P0 |
| FR-INV-04 | `specmetry check run --all` 运行所有已确认的检查命令，产生初始证据 | P0 |
| FR-INV-05 | Audit 可以为缺少检查命令的自动化 AC 提议检查命令（以提案形式，需人工确认） | P1 |
| FR-INV-06 | 记录"未映射工作"：Agent 做了、但不对应任何基准条目的工作 | P1 |

### F5 Agent 自述

| ID | 需求 | 优先级 |
|---|---|---|
| FR-CLM-01 | Agent 通过 CLI 批量提交 Claim，内容包括目标 AC、断言（`done` / `partial` / `not_done` / `blocked`）、摘要和引用 | P0 |
| FR-CLM-02 | Claim 只能使 AC 到达 `claimed` / `in_progress` / `blocked` | P0 |
| FR-CLM-03 | 摘要不超过 500 字符；所有文本在持久化前经过密钥脱敏 | P0 |
| FR-CLM-04 | Claim 自动记录 AC 当时的 revision、git HEAD，以及所属 Agent 会话（如果能获得） | P0 |
| FR-CLM-05 | Claim 只能指向当前已确认、未退役的 AC | P0 |

### F6 执行证据

| ID | 需求 | 优先级 |
|---|---|---|
| FR-EV-01 | AC 可以绑定一个经人确认的检查命令。`specmetry check run` 执行它，并记录结果（`pass` / `fail` / `error`）、退出码、时长、git HEAD、工作区是否有未提交改动、脱敏后的输出尾部和完整输出的哈希。 | P0 |
| FR-EV-02 | 检查有超时限制，超时时终止整个进程树并记为 `error` | P0 |
| FR-EV-03 | `check run` 只能执行基准中已确认的命令，不接受临时传入的命令 | P0 |
| FR-EV-04 | 提交中出现基准编号（trailer 或正文）时，建立 `relates` 关联 | P0 |
| FR-EV-05 | 证据不可修改、不可删除（`purge` 除外） | P0 |
| FR-EV-06 | 展示证据产生时的 HEAD 落后当前 HEAD 多少个提交 | P1 |
| FR-EV-07 | 导入 CI 结果作为证据 | P2 |

### F7 正式验收

| ID | 需求 | 优先级 |
|---|---|---|
| FR-ACC-01 | 人在 Dashboard 或交互式 CLI 中对 AC 作出 `accepted` / `rejected` / `waived` 决定，并附说明。系统记录当时的证据快照、git HEAD 和通道。 | P0 |
| FR-ACC-02 | Agent 通道不能产生验收：非交互 CLI 返回退出码 3；PreToolUse Hook 拦截 Agent 调用人类专属命令 | P0 |
| FR-ACC-03 | 验收与 AC 的 revision 绑定；`material` 修订后，旧验收不再生效 | P0 |
| FR-ACC-04 | 已验收的 AC 之后出现失败证据时，状态变为 `regressed` | P0 |
| FR-ACC-05 | `waived` 与 `rejected` 必须填写说明 | P0 |
| FR-ACC-06 | 新决定取代旧决定，历史完整保留 | P0 |

### F8 状态与差距

| ID | 需求 | 优先级 |
|---|---|---|
| FR-ST-01 | 按 [data-model.md §6](data-model.md#6-状态推导) 的决策表推导 AC 状态和需求状态，不提供任何直接设置状态的接口 | P0 |
| FR-ST-02 | 差距分析：输出差距类型、严重度、关联条目和建议的下一步 | P0 |
| FR-ST-03 | 交付就绪判定；未就绪时逐条列出原因 | P0 |
| FR-ST-04 | `specmetry status`（人类可读输出，或 `--json`）与 `specmetry gaps` | P0 |
| FR-ST-05 | 冲突标记：自述与证据矛盾时标记 | P0 |
| FR-ST-06 | 进度快照与趋势 | P1 |
| FR-ST-07 | 交付报告导出（Markdown）：需求状态、验收记录、未决问题（见 PD-08） | P1 |

### F9 持续更新与采集

| ID | 需求 | 优先级 |
|---|---|---|
| FR-COL-01 | 适配 Claude Code Hooks：SessionStart、SessionEnd、PostToolUse（命令执行、文件编辑）、Stop，以及用于拦截的 PreToolUse | P0 |
| FR-COL-02 | Hook 处理永远不阻塞 Agent，也不让 Agent 出错：无论发生什么都以 0 退出（PreToolUse 拦截人类专属命令是唯一例外） | P0 |
| FR-COL-03 | 事件至少投递一次，入库按事件 ID 去重（幂等） | P0 |
| FR-COL-04 | 采集按字段白名单进行，落盘前完成脱敏。不读取 transcript，不保存文件内容、提示词和工具输出正文。 | P0 |
| FR-COL-05 | `/specmetry sync` 在一段工作结束时提交 Claim、关联提交、运行相关检查，并输出状态变化 | P0 |
| FR-COL-06 | Stop Hook 可以选择提醒 Agent 执行 sync，默认关闭（见 PD-06） | P1 |
| FR-COL-07 | 在没有 Hook 的 Agent 环境中，CLI、检查和 git 扫描仍能工作（降级模式） | P0 |
| FR-COL-08 | 支持 git worktree：在同一仓库的任一 worktree 中工作，都能归属到同一项目 | P0 |

### F10 Web Dashboard

| ID | 需求 | 优先级 |
|---|---|---|
| FR-UI-01 | `specmetry dashboard` 启动本地服务并打开浏览器，只监听回环地址 | P0 |
| FR-UI-02 | **项目概览**：交付就绪判定、必需需求验收率、状态分组漏斗、"自述未证实"数、主要差距、最近活动 | P0 |
| FR-UI-03 | **基准**：目标 → 需求 → AC 的树形视图，显示状态、优先级和来源引用（可展开原文片段），支持按状态和优先级筛选 | P0 |
| FR-UI-04 | **审核队列**：提案条目逐条展示（修改类显示前后对比），支持接受、编辑后接受、拒绝，以及修改类的 `material` / `editorial` 选择；集中展示待澄清问题 | P0 |
| FR-UI-05 | **需求详情**：各 AC 的时间线（自述、证据、验收），以及验收操作 | P0 |
| FR-UI-06 | **差距**：按类型和严重度分组，附下一步建议 | P0 |
| FR-UI-07 | **活动**：Agent 会话、提交、事件元数据 | P1 |
| FR-UI-08 | **趋势图** | P1 |
| FR-UI-09 | **设置与健康**：Skill / Hook 安装状态、spool 积压、数据位置、保留期 | P0 |
| FR-UI-10 | 数据变化后，已打开的页面在 5 秒内反映出来 | P0 |
| FR-UI-11 | `inferred_from_code` 条目、未通过校验的引用、在有未提交改动的工作区上产生的证据，都有可见标识 | P0 |
| FR-UI-12 | 多项目列表 | P0 |

### F11 隐私与数据管理

| ID | 需求 | 优先级 |
|---|---|---|
| FR-PRV-01 | 默认不发出任何出站网络请求，没有遥测 | P0 |
| FR-PRV-02 | 所有要持久化的自由文本字段（命令、摘要、输出尾部、提交标题等）都经过密钥模式脱敏 | P0 |
| FR-PRV-03 | `specmetry purge` 删除一个项目的全部数据，需要交互确认 | P0 |
| FR-PRV-04 | 可配置原始事件和检查日志的保留期（默认分别为 90 天、30 天） | P1 |
| FR-PRV-05 | Dashboard 不加载任何外部资源（通过 CSP 强制） | P0 |

### F12 Agent Skill

| ID | 需求 | 优先级 |
|---|---|---|
| FR-SK-01 | Skill 包内嵌在二进制中发布，由 `specmetry setup` 安装；版本与二进制一致，不一致时由 `doctor` 报告 | P0 |
| FR-SK-02 | 提供 `init`、`status`、`sync`、`goals`、`audit` 五个命令，行为见 §5 | P0 |
| FR-SK-03 | Skill 只通过 CLI 的 JSON 接口读写数据，不直接访问数据库或数据目录 | P0 |
| FR-SK-04 | Skill 指令明确禁止：执行人类专属命令；在 Claim 中粘贴代码正文或密钥；为了让检查通过而削弱测试或检查命令 | P0 |
| FR-SK-05 | Skill 包含需求粒度指南：一条需求约等于一项用户可感知的能力或一条约束，AC 必须可判定 | P0 |

## 5. Agent Skill 命令规格

调用语法取决于具体 Agent。在 Claude Code 中设计为 `/specmetry <子命令>`，以实现时核实为准。所有子命令都只通过 `specmetry ... --json` 与 Specmetry 交互。

### `/specmetry init`

- **用途**：初始化项目，并推进到下一步未完成的初始化步骤（见 §4.1.1）。
- **步骤**：
  1. 调用 `specmetry init --json`，读取 `init_progress`、`stage`、候选文档和测试提示。
  2. 根据进度执行下一步：确认并登记来源 → 提取并提交提案 → 提示用户审核并**停止** → （基准确认后）执行 audit 流程并运行检查。
- **产出**：进度说明，以及下一步需要人做什么。需要人工审核时，执行 `specmetry dashboard --page review` 为用户打开审核页。
- **禁止**：确认提案、回答问题、写入验收、修改项目代码。

### `/specmetry status`

- **用途**：只读的进度摘要。
- **步骤**：调用 `specmetry status --json` 和 `specmetry gaps --json`。
- **产出**：交付就绪判定、必需需求验收率、各分组计数、最重要的 5 个差距，以及按差距推导出的下一步建议。
- **禁止**：任何写操作。

### `/specmetry sync`

- **用途**：一段工作结束后，把进展记录为事实。
- **步骤**：
  1. 获取当前基准和本会话的起始 HEAD（`specmetry status --json --session current`）。
  2. 根据本会话的改动和提交，判断哪些 AC 受到影响。
  3. 对受影响的 AC 提交 `sync` 类 Claim，并记录未映射工作。
  4. 对受影响、且有检查命令的 AC 执行 `specmetry check run <AC...>`。
  5. 如果发现对话中出现了基准之外的新需求，提交提案（不会自动应用）。
  6. 输出状态变化：哪些 AC 状态变了，新增了哪些差距。
- **禁止**：在 Claim 中写入"已验收"之类的措辞；修改基准；验收。

### `/specmetry goals`

- **用途**：查看基准，或基于新输入（新文档、用户口述的变更）生成基准变更提案。
- **步骤**：
  1. 用 `specmetry baseline show --json` 获取当前基准。
  2. 如果有新输入，先登记来源（如适用），再生成相对当前版本的 `add` / `update` / `retire` 提案并提交。
  3. 提示用户审核。
- **禁止**：确认提案。

### `/specmetry audit [范围]`

- **用途**：对照代码深度盘点（全部或指定的需求或 AC）。
- **步骤**：
  1. 获取基准和当前状态。
  2. 在范围内逐条检查代码和测试，提交 `audit` 类 Claim，并附引用。
  3. 指出与已有 Claim 或证据矛盾的地方。
  4. 为缺少检查命令的自动化 AC 提议检查命令（P1）。
  5. 运行范围内的检查。
- **建议**：Skill 建议在一个新会话中执行 audit，以减少"自己审自己"的偏差。
- **禁止**：修改代码、确认提案、验收。

## 6. CLI 命令概览

完整约定（输出格式、退出码、通道识别）见 [architecture.md §9](architecture.md#9-接口约定)。

| 命令 | 用途 | 允许的通道 |
|---|---|---|
| `specmetry version` | 版本信息 | 任意 |
| `specmetry setup [--scope user\|project] [--uninstall]` | 一次性安装、更新或卸载 Skill 与 Hooks | 人 |
| `specmetry init [--name] [--stage] [--no-exclude] [--skip-sources] [--skip-inventory]` | 初始化或推进当前项目 | 任意 |
| `specmetry doctor` | 健康检查（版本、Hook、Skill、spool、数据库） | 任意 |
| `specmetry source add\|list\|remove\|ack` | 管理来源文档；`ack` 确认来源变化已处理 | 任意（`ack` 记录 actor 与通道） |
| `specmetry propose --file <path\|->` | 提交提案（JSON），进入审核队列 | 任意 |
| `specmetry propose --apply` | 人工编辑：提交后立即应用 | 人 |
| `specmetry proposal list\|show` | 查看提案及条目（含冲突条目的三方对比） | 任意 |
| `specmetry review` | 交互式审核提案 | 人 |
| `specmetry baseline show\|history` / `export --format md\|json` | 查看或导出基准（导出为 P1） | 任意 |
| `specmetry question list` / `answer` / `dismiss` | 待澄清问题 | `list`：任意；其余：人 |
| `specmetry claim --file <path\|->` | 提交 Claim 批次 | 任意（`kind = manual` 仅限人） |
| `specmetry check run [AC...] [--all]` | 执行已确认的检查 | 任意 |
| `specmetry accept\|reject\|waive <AC> [-m 说明]` | 验收决定 | 人 |
| `specmetry status [--session current] [--diff-since <时间>]` / `gaps` | 状态与差距；可附带本会话信息或某时刻以来的状态变化 | 任意 |
| `specmetry report` | 交付报告（P1） | 任意 |
| `specmetry dashboard [--page <页面>]` | 启动本地 Dashboard（已在运行时复用），并在浏览器中打开指定页面。令牌只交给浏览器，不打印到输出。 | 任意 |
| `specmetry hook <agent> <event>` | Hook 入口（内部使用） | Hook |
| `specmetry purge` | 删除项目数据 | 人 |

## 7. 非功能需求

| ID | 需求 | 指标 |
|---|---|---|
| NFR-01 | 无账号、无网络依赖 | 断网环境下完成全部 MVP 流程 |
| NFR-02 | Hook 开销 | 在参考机器上，单次 Hook 处理墙钟时间 p95 ≤ 100 ms（含进程启动）；非 Specmetry 项目内 p95 ≤ 30 ms |
| NFR-03 | 采集可靠性 | Hook 在任何错误条件下都以 0 退出；除磁盘写满外，已写入 spool 的事件不会丢失 |
| NFR-04 | 规模 | 500 条需求 / 2000 个 AC / 10 万条事件时，`status --json` < 1 s，Dashboard 首屏 < 2 s |
| NFR-05 | 平台 | Windows 10/11、macOS 13+、Linux（x86_64、arm64）均为一级支持 |
| NFR-06 | 安装 | 单一二进制；运行时只依赖系统中已有的 `git` |
| NFR-07 | 数据可移植 | 数据是单个 SQLite 文件，schema 有文档（基准导出见 FR-BL-11，P1） |
| NFR-08 | 可升级 | schema 只做前向迁移，迁移前自动备份数据库 |
| NFR-09 | 可诊断 | 本地日志文件，以及 `specmetry doctor` |
| NFR-10 | 路径健壮性 | 路径中含空格或非 ASCII（如中文）字符、文件使用 CRLF 换行时，功能正常 |

## 8. 不在范围内

见 [mvp.md §3](mvp.md#3-范围外明确不做)。

## 9. 产品待决策事项

以下事项尚未确定。"建议默认"是本轮设计采用的假设，**需要产品负责人确认**。

| ID | 问题 | 选项 | 建议默认 | 最晚决定于 |
|---|---|---|---|---|
| PD-01 | 交付就绪的口径 | 仅 `must` / `must` + `should` / 可配置 | 仅 `must`，项目设置中预留开关 | M2 |
| PD-02 | 主进度指标的口径 | 需求级验收率 / AC 级验收率 / 加权 | 必需需求的验收率作为主指标，另附 AC 漏斗 | M2 |
| PD-03 | 验收粒度 | 仅 AC 级 / 允许需求级直接验收 | 仅 AC 级；没有 AC 的需求显示 `needs_criteria` | M1 |
| PD-04 | AC 修订是否使验收失效 | 总是失效 / 审核者选择 `material`、`editorial` / 永不失效 | 审核者选择；检查命令变更一律视为 `material` | M1 |
| PD-05 | MVP 官方支持哪些 Agent | 仅 Claude Code / 加上 Codex 等 | Claude Code 完整支持（Skill + Hooks）；其他 Agent 通过 AGENTS.md 指令和 CLI 降级使用 | M3 |
| PD-06 | Stop Hook 提醒 sync 是否算"干预 Agent" | 不提供 / 默认关闭 / 默认开启 | 提供，默认关闭 | M3 |
| PD-07 | 在仓库中留下的痕迹 | marker 写入 `.git/info/exclude` / 建议提交 / 不用 marker，按路径识别 | 默认 exclude，可选择提交（为未来的团队共享预留） | M1 |
| PD-08 | 交付报告是否进入 MVP | P0 / P1 / 延后 | P1 | M5 |
| PD-09 | 界面和文档语言 | 中文 / 英文 / 双语 | 设计文档用中文；代码、CLI 输出、提交信息用英文；Dashboard 文本外置，首发语言待定 | M5 |
| PD-10 | 开源许可证 | Apache-2.0 / MIT / AGPL-3.0 / 双许可 | **未给出倾向**：需要结合 Hosted 商业模式决定。Apache-2.0 利于采用并带专利授权；AGPL 能限制第三方托管竞争，但会抑制企业采用。 | 接受外部贡献之前（M0） |
| PD-11 | Local 与 Hosted 的功能边界 | 本地单人能力完整，Hosted 只增加身份、同步和协作 / 部分高级功能仅在 Hosted 提供 | 前者 | 首次公开发布之前 |
| PD-12 | 优先级方案 | MoSCoW（`must` / `should` / `could` / `wont`）/ P0–P3 | MoSCoW；`wont` 表示本次不交付，不参与指标 | M1 |
| PD-13 | 是否支持多个交付阶段（里程碑） | MVP 不支持 / 支持 | 不支持；MVP 只有单一交付范围，靠优先级表达 | M1 |
| PD-14 | Agent 能否代为记录用户在对话中给出的问题答复 | 能 / 不能 | 不能：问题由人回答；Agent 可以根据答复提交提案 | M1 |
| PD-15 | Skill 质量评测阈值 | — | 初始阈值：基准提取召回率 ≥ 80%；提交的提案中，未通过校验的引用占比 ≤ 10% | M4 |
