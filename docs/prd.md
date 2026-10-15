# Specmetry 产品需求文档（PRD）

> 状态：Draft v0.3 · 2026-10-13（依据 [codex-review-001](reviews/codex-review-001.md) 与 [codex-review-002](reviews/codex-review-002.md) 修订）
> 相关文档：[vision.md](vision.md) · [mvp.md](mvp.md) · [architecture.md](architecture.md) · [data-model.md](data-model.md) · [acceptance-criteria.md](acceptance-criteria.md) · [review-resolution.md](review-resolution.md) · [implementation-plan.md](implementation-plan.md)

优先级的含义：

- **P0**：MVP 必须完成，是里程碑的退出条件。
- **P1**：MVP 中可选，不阻塞里程碑。
- **P2**：MVP 之后再做。

## 1. 用户与场景

**主要用户**：用 Claude Code 等 AI coding agent 交付客户项目的独立开发者或外包开发者。

| # | 场景 | 期望结果 |
|---|---|---|
| S1 | 只有文档的新项目 | 得到经确认的基准和交付范围；交付状态为 `in_progress`；列出文档中的歧义 |
| S2 | 中途接入开发中的项目 | 得到基准、范围和逐条盘点结果；没有盘点到的条目显示"未知"；交付判定只基于在交付快照上执行的检查 |
| S3 | 文档不完整或几乎只有代码 | 从代码反推的条目有明确标识；确认范围时，人必须确认已经补充了代码中看不到的交付目标 |
| S4 | 日常推进 | `sync` 记录自述并在当前 worktree 中运行检查；只有交付快照上的结果才影响交付判定 |
| S5 | 需求变更 | 需求实质修订后，相关 AC 需要逐条确认是否仍然适用；范围需要重新确认 |
| S6 | 交付判定 | 只有在"干净、确定的代码快照、已确认的完整范围、范围内全部验收或豁免"的前提下，才给出 `ready` |
| S7 | 多个 worktree 并行开发 | feature worktree 中的检查结果不会影响交付分支的判定，反之亦然 |

## 2. 术语表

| 术语 | 英文 | 定义 |
|---|---|---|
| 项目 | Project | 被跟踪的代码库，由 git 公共目录确定身份；可以有多个 worktree |
| 工作树 | Worktree | 项目的一个检出目录，有自己独立的 HEAD 和 index |
| 代码上下文 | Code Context | worktree、HEAD 提交、工作区状态（`clean` / `dirty` / `unknown`）三者的组合 |
| 交付目标 | Delivery Target | 人确认的、要交付的 ref（分支或标签）。MVP 中每个项目只有 1 个。 |
| 快照 | Snapshot | 交付目标的 ref 当前指向的提交，是交付判定所针对的代码对象 |
| 交付范围 | Delivery Scope | 人确认的、本次要交付的需求清单，绑定交付目标、基准版本和来源集合 |
| 来源集合 | Source Set | 全部已登记、未移除的来源及其已确认的哈希 |
| 项目基准 | Project Baseline | 经人确认的目标、需求、AC，带版本 |
| 验收标准 | Acceptance Criterion（AC） | 属于某个需求、可以判定的条件，是**唯一的**验收单位（PD-03）：需求的状态只能由它的 AC 汇总得出，不能直接验收需求 |
| 检查 | Check | AC 上经人确认的命令，包含说明"为什么能验证该 AC"的理由 |
| 合格执行 | Qualifying Run | 开始和结束时 HEAD 都等于记录的提交、工作区都干净的检查执行（隔离级还要求通过隔离验证） |
| 证据等级 | Evidence Grade | `observed`（观测级）：只保证开始和结束两个时刻与快照一致，不保证执行期间未改变。`snapshot_isolated`（隔离级）：在私有快照目录中、由外部强制只读的环境中执行，并在执行前后验证。MVP 只提供观测级。 |
| 提案 | Proposal | 对基准的变更建议，经人审核后生效 |
| 重新确认 | Reaffirm | 父需求实质修订后，人确认某个 AC 仍然适用 |
| Agent 自述 | Claim | Agent 对 AC 的结构化陈述，只影响"进度" |
| 进度 | Progress | 由自述和提交推导出的工作情况，可信度低。Claim 不能单独产生 `accepted` 或 `ready`（手动 AC 的例外见 §3.1） |
| 交付状态 | Delivery Status | 相对于快照推导出的状态，见 §3.2 |
| 验收决定 | Decision | 人作出的接受（`normal` / `manual` / `exception`）、拒收或豁免 |
| 沿用 | Carry-forward | 自动化 AC 的 `normal` 验收，在新快照上以同一检查配置、相同的 `inputs` 对象 ID 得到**隔离级**合格通过且结果一致时，可以继续适用（PD-16）。MVP 只有观测级证据，因此不会发生沿用；手动验收和例外验收永不沿用 |
| 差距 | Gap | 由推导得出的待处理项；阻塞差距就是交付判定的原因 |
| 序号 | seq | 数据库分配的单调递增序号，是判断事实先后的唯一依据 |
| 通道 | Channel | `agent_cli` / `human_cli` / `dashboard` / `hook` / `system` |

## 3. 核心概念

### 3.1 三层事实与两种视图

- **Claim**：由 Agent 产生，主要影响进度。它不能单独产生 `accepted` 或 `ready`，也不能让自动化 AC 越过 `in_progress`；对手动 AC，`done` 自述只会把它标为 `ready_for_review`，含义是"等待人工查看"，这不是验证。
- **执行证据**：由 Specmetry 在某个代码上下文中执行检查而产生。只有在**快照上的合格执行**，才会参与交付判定。
- **验收决定**：由人产生，绑定查看时的快照、修订和证据。它是否适用于当前快照，由推导决定。

### 3.2 统一的交付状态

**AC 层面**（相对快照 `S`）：

| 状态 | 含义 | 人需要做什么 |
|---|---|---|
| `not_assessed` | `S` 无法解析 | 设置或修复交付目标 |
| `in_progress` | 在 `S` 上还没有任何验证 | 实现并验证 |
| `failing` | `S` 上的合格检查失败 | 修复 |
| `rejected` | 人拒收。只有代码变化后出现新的合格通过，或者人改判，才会解除；在同一提交上重新运行检查不能解除。 | 返工 |
| `needs_reverification` | 曾经验证或验收过，但对 `S` 不再适用：代码变了、AC 被修订、父需求被修订 | 重新运行检查，或人工复核 |
| `ready_for_review` | 自动化 AC 在 `S` 上合格通过但还没有适用的验收；或者手动 AC 已由 Agent 自述完成 | 人工验收 |
| `accepted` | 有适用于 `S` 的验收 | — |
| `waived` | 已豁免 | — |

**目标层面**：`not_assessed` → `in_progress` → `needs_reverification` → `ready_for_review` → `ready`。完整规则见 [data-model.md §6.8](data-model.md#68-交付状态目标层面)。

交付判定同时给出**证据等级**（`delivery.evidence_grade`）：MVP 中为 `observed` 或 `none`，界面和 CLI 不得把 `observed` 说成"快照已验证"。

`ready` 必须同时满足以下条件：

- 快照可以解析；
- 范围已确认、没有过期、不为空，也不是全部豁免；
- 范围内所有 AC 都有适用于 `S` 的验收或豁免；
- 没有阻塞问题；
- 来源变化都已经由人确认。

**无法判断时，一律给出更保守的状态。**

### 3.3 权限矩阵

| 操作 | Agent（非交互 CLI） | 人（交互式 CLI） | 人（Dashboard） | 自动 |
|---|---|---|---|---|
| 初始化项目、登记来源 | ✓ | ✓ | — | `init` 创建默认交付目标 |
| 确认来源变化（`source ack`） | ✗ | ✓ | ✓ | — |
| 移除来源 | ✗ | ✓ | — | — |
| 修改片段保存模式（`project set excerpt-storage`，PD-24） | ✗ | ✓ | — | — |
| 安装 / 卸载 Skill 与 Hooks | ✗ | ✓ | — | — |
| 设置交付目标 | ✗ | ✓ | — | — |
| 打开 Dashboard（不输出令牌） | ✓ | ✓ | — | — |
| 提交提案（待审核） | ✓ | ✓ | — | — |
| 人工编辑基准（立即应用） | ✗ | ✓ | ✓ | — |
| 审核提案（生成基准版本） | ✗ | ✓ | ✓ | — |
| 确认交付范围 | ✗ | ✓ | ✓ | — |
| 回答 / 驳回问题 | ✗ | ✓ | ✓ | — |
| 提交 Claim（`sync` / `audit`） | ✓ | ✓ | — | — |
| 提交 Claim（`manual`） | ✗ | ✓ | — | — |
| 触发检查 | ✓ | ✓ | ✗ | — |
| 产生检查执行记录、发现提交 | — | — | — | ✓ |
| 验收决定 | ✗ | ✓ | ✓ | — |
| 查看 | ✓ | ✓ | ✓ | — |
| `purge` / 恢复备份 | ✗ | ✓ | ✗ | — |
| 恢复数据库（`db recover`，PD-23，待排期） | ✗ | ✓ | ✗ | — |
| 直接设置状态 | ✗ | ✗ | ✗ | 只能推导 |

"✗"依靠启发式的通道识别和 Hook 拦截来保证，目的是**防止 Agent 误执行，不能证明操作者身份**（见 [architecture.md §5](architecture.md#5-信任边界与操作通道)）。

## 4. 功能需求

### F1 项目初始化

| ID | 需求 | 优先级 |
|---|---|---|
| FR-INIT-01 | `init` 创建 marker 并注册项目，可重复执行 | P0 |
| FR-INIT-02 | 检测起始阶段，可以用 `--stage` 覆盖 | P0 |
| FR-INIT-03 | 列出候选来源，不自动登记 | P0 |
| FR-INIT-04 | `setup` 安装或卸载 Skill 与 Hooks：先展示改动，人确认后才写入 | P0 |
| FR-INIT-05 | 初始化进度可以恢复，由数据推导，条件累积（§4.1.1） | P0 |
| FR-INIT-06 | 默认写入 `.git/info/exclude`；不可写时只给出警告 | P1 |
| FR-INIT-07 | 不需要账号；核心不需要网络 | P0 |
| FR-INIT-08 | 提示常见的测试命令 | P1 |
| FR-INIT-09 | 非 git 目录可以降级使用，但永远不会给出 `ready` | P0 |
| FR-INIT-10 | 以主 worktree 当前所在的分支创建默认交付目标；分离 HEAD 或非 git 时不创建 | P0 |

#### 4.1.1 初始化流程

| 起始阶段 | 基准来源 | 现状盘点 |
|---|---|---|
| `docs_only` | 从文档提取 | 所有 AC 的进度为 `not_started` |
| `in_development` | 从文档提取；代码中存在但文档未提及的功能，作为 `inferred_from_code` 条目 | `audit`（`audit_scope = all`），然后在当前 worktree 运行检查 |
| `incomplete_docs` | 文档提取 + 从代码反推 + 待澄清问题；确认范围时，人必须声明已经补充了代码中看不到的交付目标 | 同上 |

初始化进度（条件累积）：`initialized` → `sources_registered` → `baseline_proposed` → `baseline_confirmed` → `scope_confirmed` → `inventoried`。见 [data-model.md §6.12](data-model.md#612-初始化进度)。

流程：

1. `specmetry setup`（人，每台机器一次）。
2. Agent 中执行 `/specmetry init`：取得阶段、候选文档和默认交付目标。
3. 人在对话中确认来源，Agent 执行 `source add`。
4. Agent 读取 `status --json`，得到 `base_version`；然后提取基准，提交提案。
5. Agent 执行 `dashboard --page review` 打开审核页，然后停止。人审核条目、回答问题，并**在同一次提交中确认交付范围**。
6. 人再次执行 `/specmetry init`：Agent 进行全量 audit，然后在当前 worktree 中执行 `check run --all`。
7. Agent 汇报交付状态、盘点覆盖率和阻塞差距。

### F2 来源

| ID | 需求 | 优先级 |
|---|---|---|
| FR-SRC-01 | 登记本地文件为来源，记录路径、类型、哈希和大小。只接受文档类文件（人可以用 `--allow-any-type` 放开）。重复登记同一路径时什么都不做，也不会更新已确认的哈希。 | P0 |
| FR-SRC-02 | 检测来源变化，产生阻塞差距 `SOURCE_CHANGED` | P0 |
| FR-SRC-03 | `source ack` 只能由人执行，需要填写理由，并绑定确认时磁盘上的实际哈希 | P0 |
| FR-SRC-04 | 移除来源只能由人执行；移除后引用保留，并显示为 `source_removed` | P0 |
| FR-SRC-05 | 计算来源集合哈希，供范围确认绑定 | P0 |
| FR-SRC-06 | Specmetry 不解析文档语义，只为校验引用而读取文本 | P0 |
| FR-SRC-07 | 移除来源时，人可以选择擦除该来源所有引用的片段原文（`source remove --erase-excerpts`），只保留定位、片段哈希和校验结果 | P0 |

### F3 基准

| ID | 需求 | 优先级 |
|---|---|---|
| FR-BL-01 | 提案严格符合 schema，携带 `base_version` 和 `target_revision`；不一致时返回退出码 5 | P0 |
| FR-BL-02 | 引用片段校验（≤ 500 字符）；界面如实说明校验证明了什么 | P0 |
| FR-BL-03 | `origin` 和 `confidence`；反推条目必须有代码引用，并有醒目标识 | P0 |
| FR-BL-04 | 审核：逐条或批量处理；携带 `seen_base_version`；每次提交生成一个版本 | P0 |
| FR-BL-05 | 只有已确认的条目参与推导 | P0 |
| FR-BL-06 | 版本历史可查询 | P0 |
| FR-BL-07 | `material` / `editorial`；部分字段强制为 `material`；审核时接受 `update` 条目，必须由审核者明确选择，未选择即拒绝（PD-04，V-R01） | P0 |
| FR-BL-08 | 人工编辑走同一条路径；因为会立即应用，每个更新都必须声明 `material` 或 `editorial` | P0 |
| FR-BL-09 | 冲突条目展示三方对比，必须明确确认后才能处理 | P0 |
| FR-BL-10 | 待澄清问题：Agent 可以提出，只能由人回答 | P0 |
| FR-BL-11 | **需求实质修订后，其 AC 变为 `parent_revised`**；审核时可以对每个 AC 选择 `reaffirm` / 修订 / 退役 / 暂缓；提案支持 `reaffirm` 操作 | P0 |
| FR-BL-12 | 检查配置必须有理由；不能包含密钥字面量；可以声明 `inputs` | P0 |
| FR-BL-13 | 来源引用和代码引用绑定条目的修订，只追加 | P0 |

### F4 交付目标与范围

| ID | 需求 | 优先级 |
|---|---|---|
| FR-TGT-01 | 每个项目有 1 个 `active` 的交付目标，跟踪一个分支或标签 | P0 |
| FR-TGT-02 | 修改目标只能由人操作（归档旧目标并新建目标），之后需要重新确认范围 | P0 |
| FR-TGT-03 | 每次推导时解析快照；无法解析时为 `not_assessed` | P0 |
| FR-TGT-04 | 范围确认只能由人操作：明确的需求清单（不能为空），绑定目标、基准版本和来源集合哈希；附带声明（范围完整、排除的待审条目数、`incomplete_docs` 项目已补充交付目标）；范围内的全部有效豁免逐条展示，由人确认继续适用；携带读取时的范围确认，防止旧页面覆盖（PD-21，V-T01、V-T02） | P0 |
| FR-TGT-05 | 检测范围是否过期 | P0 |
| FR-TGT-06 | 审核提交可以在同一个事务中附带范围确认 | P0 |
| FR-TGT-07 | 确认范围时，默认选中 `must` 需求（只是界面上的默认值） | P0 |

### F5 Worktree

| ID | 需求 | 优先级 |
|---|---|---|
| FR-WT-01 | 通过 git 公共目录共享项目身份；worktree 第一次出现时自动登记 | P0 |
| FR-WT-02 | Claim 批次、检查执行、会话和事件都记录 worktree 与代码上下文 | P0 |
| FR-WT-03 | 检查在调用者所在的 worktree 中执行 | P0 |
| FR-WT-04 | git 扫描游标按 ref 维护 | P0 |
| FR-WT-05 | `status` 同时给出交付视图和当前工作区视图；Dashboard 明确显示正在观察的目标和快照 | P0 |

### F6 现状盘点

| ID | 需求 | 优先级 |
|---|---|---|
| FR-INV-01 | `audit` 批次必须声明范围（全部或部分需求）；逐条给出断言（可以是 `unknown`）和引用 | P0 |
| FR-INV-02 | 引用存在性校验（只标记，不拒绝） | P0 |
| FR-INV-03 | `init --json` 返回 git 基础事实（提交数、最近提交时间） | P1 |
| FR-INV-04 | `check run --all` | P0 |
| FR-INV-05 | audit 以提案形式为 AC 提议检查命令 | P1 |
| FR-INV-06 | 记录未映射工作 | P1 |
| FR-INV-07 | 计算盘点覆盖率；没有盘点到的条目，进度显示为 `unknown`；局部 audit 不算完成盘点 | P0 |

### F7 Agent 自述

| ID | 需求 | 优先级 |
|---|---|---|
| FR-CLM-01 | 通过 CLI 提交 Claim 批次 | P0 |
| FR-CLM-02 | Claim 不能单独产生 `accepted` 或 `ready`；自动化 AC 不会因为 Claim 越过 `in_progress`；手动 AC 的 `done` 自述只会让它进入 `ready_for_review` | P0 |
| FR-CLM-03 | 摘要不超过 500 字符，只能有一段，不能包含代码块；命中密钥规则即拒绝 | P0 |
| FR-CLM-04 | 携带 `base_version` 和 `target_revision`（不一致时返回退出码 5）；CLI 记录代码上下文；如果同一 worktree 中恰好有一个活跃会话，就关联它 | P0 |
| FR-CLM-05 | 目标必须是 `active` 的 AC；`unknown` 断言只能用于 `audit`；`manual` 批次只能由人提交 | P0 |

### F8 执行证据

| ID | 需求 | 优先级 |
|---|---|---|
| FR-EV-01 | 检查执行绑定开始时的修订、配置哈希、输入对象 ID 和代码上下文；输出只实时转发给调用者，不保存；带 `--json` 时，stdout 只有一份 `check-run.v1` JSON，检查输出转发到 stderr | P0 |
| FR-EV-02 | 超时后终止整个进程树 | P0 |
| FR-EV-03 | 只能执行基准中已确认的命令 | P0 |
| FR-EV-04 | 判定合格执行；开始与结束时的上下文不一致时，记为 `context_changed` | P0 |
| FR-EV-05 | 执行记录只追加 | P0 |
| FR-EV-06 | 提交关联只保存 SHA、时间和关联关系 | P0 |
| FR-EV-07 | 导入 CI 结果 | P2 |
| FR-EV-08 | 每条执行都标注证据等级。MVP 只产生 `observed`；`--isolation required` 在无法建立并验证隔离时不执行、不写记录，返回退出码 6；绝不把观测级记为隔离级 | P0 |
| FR-EV-09 | 隔离执行器（`snapshot_isolated`）：私有快照目录、外部强制只读、执行前后验证；`--isolation auto`（MVP 之后，M7） | P2 |

### F9 验收

| ID | 需求 | 优先级 |
|---|---|---|
| FR-ACC-01 | 只能由人作出决定，并携带目标、快照、AC 修订、需求修订、证据 ID 和 **`seen_decision_id`**（看到的当前决定，首次为 null）；在写事务中原子比较，不一致时返回 409 或退出码 5；客户端不得自动重试，重新读取后可以明确改判 | P0 |
| FR-ACC-02 | 阻止 Agent 通道作出决定（启发式识别加 Hook 拦截），并如实声明能力边界 | P0 |
| FR-ACC-03 | `normal` 验收要求 AC 在 `S` 上的最新合格执行为 `pass`（人可以在自己拒收后改判）；必须确认服务端推导出的风险（观测级证据、结果不一致），`acknowledged_risks` 与之不符时拒绝 | P0 |
| FR-ACC-04 | `manual` 验收只适用于 `S` | P0 |
| FR-ACC-05 | `exception` 验收：只能在没有合格执行时作出；必须填写理由；只适用于 `S`；带警告标记。状态输出和界面必须同时显示它是例外验收、理由、适用的快照和证据等级 `none`；它不删除、也不隐藏已有的实际测试证据（PD-17） | P0 |
| FR-ACC-06 | 在 `S` 上检查失败时，不能接受 | P0 |
| FR-ACC-07 | 拒收和豁免必须填写理由；豁免与快照无关 | P0 |
| FR-ACC-08 | 历史完整保留；当前决定按 `seq` 确定 | P0 |
| FR-ACC-09 | 沿用规则：只适用于自动化 AC 的 `normal` 验收；要求配置和输入相同，在 `S` 上有**隔离级**合格通过，并且结果一致。MVP 没有隔离级证据，因此不会沿用。 | P0 |

### F10 状态与交付判定

| ID | 需求 | 优先级 |
|---|---|---|
| FR-ST-01 | 状态只能推导；不提供任何设置状态的接口 | P0 |
| FR-ST-02 | 按决策表推导 AC 交付状态 | P0 |
| FR-ST-03 | 进度单独推导；Claim 对交付状态的影响仅限于 §3.1 所述 | P0 |
| FR-ST-04 | 推导目标层面的交付状态，只在满足 §3.2 的条件时给出 `ready` | P0 |
| FR-ST-05 | 差距与交付原因统一 | P0 |
| FR-ST-06 | `status` / `gaps` 命令，包括 `--session current` 和 `--diff-since <seq>` | P0 |
| FR-ST-07 | `claimed_without_evidence` 和 `conflict` 的标记与指标 | P0 |
| FR-ST-08 | 按 `seq` 确定性排序；输出附带 `as_of_seq` | P0 |
| FR-ST-09 | 保守规则（[data-model.md §6.9](data-model.md#69-保守规则无法判断时)） | P0 |
| FR-ST-10 | 趋势 | P2 |
| FR-ST-11 | 交付报告导出 | P2 |

### F11 采集

| ID | 需求 | 优先级 |
|---|---|---|
| FR-COL-01 | Claude Code Hooks：SessionStart、SessionEnd、PostToolUse、PostToolUseFailure、Stop；PreToolUse 拦截覆盖 Bash 和 PowerShell | P0 |
| FR-COL-02 | 延迟预算、输入上限和时间上限；丢弃时计数；始终以 0 退出 | P0 |
| FR-COL-03 | 至少投递一次，入库去重 | P0 |
| FR-COL-04 | 只写入 `spool-event.v1` 白名单中的字段 | P0 |
| FR-COL-05 | `sync` 明确比较起点，变化路径包括未跟踪的文件 | P0 |
| FR-COL-06 | Stop 时提醒执行 sync | P2 |
| FR-COL-07 | 没有 Hook 时可以降级使用 | P0 |
| FR-COL-08 | 会话起点 HEAD 通过读文件获取，失败时记为 `unknown` | P0 |
| FR-COL-09 | 固定经过测试的 Claude Code 版本范围，由 `doctor` 报告 | P0 |

### F12 Dashboard

| ID | 需求 | 优先级 |
|---|---|---|
| FR-UI-01 | 只监听回环地址；令牌不打印到输出 | P0 |
| FR-UI-02 | 概览：目标、快照、范围、交付状态、阻塞差距、指标、最近的会话 | P0 |
| FR-UI-03 | 基准树：进度和交付状态；引用以及它实际证明了什么 | P0 |
| FR-UI-04 | 审核队列：AC 的后续处理，以及在同一次提交中确认范围。确认范围时列出：仍待审的提案条目（要求明确排除）、各来源的状态、交付目标和快照。 | P0 |
| FR-UI-05 | 需求详情：快照上的执行（标注证据等级），以及其他上下文中的执行（标注为"不适用于交付"）；Claim、当前决定、允许的决定类型；展示必须确认的风险说明；提交决定时携带前置条件和风险确认 | P0 |
| FR-UI-06 | 差距页 | P0 |
| FR-UI-07 | 完整的活动页 | P2 |
| FR-UI-08 | 趋势图 | P2 |
| FR-UI-09 | 健康页：Hook 与 Skill、spool 积压、drop 计数、worktree、数据目录及其权限 | P0 |
| FR-UI-10 | 数据变化后 5 秒内刷新 | P0 |
| FR-UI-11 | 标识：反推、`ref_not_found`、例外验收、结果不一致、自述无证据 | P0 |
| FR-UI-12 | 多项目列表 | P0 |
| FR-UI-13 | 页面过期时（包括其他页面刚作出决定）：收到 409 后展示最新状态并重新加载，不会产生部分写入，不会自动重试 | P0 |
| FR-UI-14 | 批量验收：对多个 `ready_for_review` 的 AC 逐条提交决定，每条都带各自的前置条件和风险确认 | P1 |

### F13 隐私

规范来源：[architecture.md §10](architecture.md#10-隐私与数据保留契约)。

| ID | 需求 | 优先级 |
|---|---|---|
| FR-PRV-01 | 核心不发出任何出站请求，没有遥测 | P0 |
| FR-PRV-02 | C0 数据永不持久化（检查输出、原始命令、环境变量值、对话等） | P0 |
| FR-PRV-03 | `purge`：只能在交互式终端执行；写墓碑；`VACUUM`；按用户的选择处理备份 | P0 |
| FR-PRV-04 | 保留期：事件 30 天，死信 7 天 | P0 |
| FR-PRV-05 | CSP；不加载外部资源 | P0 |
| FR-PRV-06 | Unix `0700`，Windows 受保护的 ACL；`doctor` 检查权限 | P0 |
| FR-PRV-07 | 检查命令和自由文本命中密钥规则即拒绝写入，不回显命中的内容。规则集只能降低泄露风险，不能保证检测出所有秘密（architecture §10.8） | P0 |
| FR-PRV-08 | 项目设置 `excerpt_storage`：`hash_only`（默认，片段只在内存中校验，只保存定位、哈希和校验结果）或 `full`（保存片段原文，只能由人开启，M2 起可用）（PD-24） | P0 |
| FR-PRV-09 | 交付状态与证据等级的所有人类可读输出（CLI、Dashboard、报告）中，`observed` 证据都不得出现 snapshot-verified 或"快照已验证"之类的措辞；`none` 不得出现"已通过自动化验证"之类的措辞，必须说明没有自动化执行证据（依据手动、例外或豁免） | P0 |

### F14 存储可靠性

| ID | 需求 | 优先级 |
|---|---|---|
| FR-STO-01 | 迁移：加锁、`VACUUM INTO` 备份、在事务中执行 | P0 |
| FR-STO-02 | 版本保护：旧程序拒绝打开更新版本的 schema | P0 |
| FR-STO-03 | 打开数据库时重新执行墓碑删除 | P0 |
| FR-STO-04 | `backup create`、`backup list`、`backup restore` | P1 |
| FR-STO-05 | `seq` 计数器；只追加触发器 | P0 |

### F15 Agent Skill

| ID | 需求 | 优先级 |
|---|---|---|
| FR-SK-01 | Skill 包内嵌在二进制中，由 `setup` 安装，由 `doctor` 校验版本 | P0 |
| FR-SK-02 | 五个命令（§5） | P0 |
| FR-SK-03 | 只通过 CLI 的 JSON 接口读写数据；写入时携带读取版本；收到退出码 5 后重新读取，不盲目重试 | P0 |
| FR-SK-04 | 禁止：通过 CLI、HTTP、浏览器自动化或写文件代做人工操作；在 Claim 中粘贴代码或密钥；为了通过检查而削弱测试或检查 | P0 |
| FR-SK-05 | 需求粒度指南 | P0 |
| FR-SK-06 | 建议在新会话中执行 audit；局部 audit 必须声明范围 | P0 |

## 5. Agent Skill 命令规格

在 Claude Code 中调用方式为 `/specmetry <子命令>`，以实现时核实为准。所有子命令都只通过 `specmetry … --json` 交互，写入时都携带读取到的版本。

| 命令 | 用途与步骤 | 禁止 |
|---|---|---|
| `/specmetry init` | 读取 `init --json`，按初始化进度推进：确认来源 → 提交提案 → 打开审核页并停止 → 范围确认后进行全量 audit（`audit_scope = all`）并运行检查 → 汇报 | 确认提案或范围、回答问题、验收、修改代码 |
| `/specmetry status` | 读取 `status --json` 和 `gaps --json`；汇报交付目标、快照、交付状态和阻塞差距；当前 worktree 不是快照时，单独说明工作区视图 | 任何写操作 |
| `/specmetry sync` | 用 `status --json --session current` 取得比较起点和变化路径；判断受影响的 AC；提交 `sync` 批次；在当前 worktree 中运行相关检查；用 `--diff-since <seq>` 汇报变化；比较起点为 `none` 时，只对实际核实过的 AC 作出断言 | 修改基准、验收、把工作区结果说成交付结论 |
| `/specmetry goals` | 读取基准；针对新输入，生成携带版本的 `add` / `update` / `retire` / `reaffirm` 提案；需求实质修订时，同时提议受影响 AC 的处理方式 | 确认提案 |
| `/specmetry audit [范围]` | 声明 `audit_scope`；逐条给出断言（可以是 `unknown`）和引用；指出矛盾；提议检查命令（P1）；运行检查；建议在新会话中执行 | 修改代码、确认、验收 |

## 6. CLI 命令概览

退出码与约定见 [architecture.md §9](architecture.md#9-接口约定)。面向 Agent 的 `--json` 输出 schema 是：`init.v1`、`status.v1`、`gaps.v1`、`check-run.v1`、`source-add.v1`、`proposal-receipt.v1`；读取模型是：`source-list.v1`、`proposal-list.v1`、`proposal-show.v1`、`baseline-show.v1`、`baseline-history.v1`、`question-list.v1`、`target-scope.v1`；错误输出是 `error.v1`。人类专属的写命令在 MVP 中只输出人类可读文本（出错时带 `--json` 仍输出 `error.v1`）。

| 命令 | 用途 | 通道 |
|---|---|---|
| `version` / `doctor` | 版本；健康与权限检查 | 任意 |
| `setup [--scope user\|project] [--uninstall]` | 安装或卸载 Skill 与 Hooks | 人 |
| `init [--name] [--stage] [--no-exclude] [--skip-sources] [--skip-inventory]` | 初始化或推进 | 任意 |
| `source add\|list [--include-removed] [--refs]` | 登记、查看来源（`add --allow-any-type` 仅限人；`--refs` 列出全部引用及保存了哪些片段原文） | 任意 |
| `source ack\|remove` | 确认来源变化、移除来源 | 人 |
| `project set excerpt-storage <hash_only\|full>` | 修改片段保存模式（PD-24；M2 起可用） | 人 |
| `target show` / `target set --ref <ref>` | 查看、设置交付目标 | `show`：任意；`set`：人 |
| `scope show` / `scope confirm [--file]` | 查看、确认范围 | `show`：任意；`confirm`：人 |
| `proposal list\|show` | 查看提案（含三方对比） | 任意 |
| `propose --file <path\|->` | 提交提案 | 任意 |
| `propose --apply --file` | 人工编辑，立即应用 | 人 |
| `review` | 交互式审核（可以附带范围确认） | 人 |
| `baseline show\|history` | 查看基准 | 任意 |
| `question list` / `answer` / `dismiss` | 待澄清问题 | `list`：任意；其余：人 |
| `claim --file <path\|->` | 提交 Claim 批次 | 任意（`manual` 仅限人） |
| `check run [AC...] [--all] [--isolation off\|required] [--json]` | 在当前 worktree 执行检查（观测级）；`--isolation required` 在 MVP 中返回退出码 6 | 任意 |
| `accept\|reject\|waive <AC>` | 验收决定（交互式确认，显示前置条件；`accept --exception -m` 表示例外验收） | 人 |
| `status [--session current] [--diff-since <seq>]` / `gaps` | 状态与差距 | 任意 |
| `dashboard [--page <页面>]` | 启动或复用 Dashboard | 任意 |
| `hook <agent> <event>` | Hook 入口 | Hook |
| `purge` | 删除项目数据 | 人 |
| `backup create\|list\|restore` | 备份（P1） | `create`、`list`：任意；`restore`：人 |

## 7. 非功能需求

| ID | 需求 | 指标 |
|---|---|---|
| NFR-01 | 网络边界 | 核心不发出出站请求；用人工或录制的输入可以离线完成全部推导。Agent 和检查命令的网络需求不在 Specmetry 的保证范围内（[architecture.md §10.5](architecture.md#105-网络边界)）。 |
| NFR-02 | Hook 开销 | p95 ≤ 100 ms；非 Specmetry 项目内 p95 ≤ 30 ms；200 ms 内部预算；5 s 外部超时 |
| NFR-03 | 采集可靠性 | 始终以 0 退出；已写入 spool 的事件不丢失；丢弃的事件可以计数、可以看到 |
| NFR-04 | 规模 | 2000 个 AC、10 万条事件时，`status --json` < 1 s，Dashboard 首屏 < 2 s |
| NFR-05 | 平台 | Windows 10/11、macOS 13+、Linux（x86_64、arm64） |
| NFR-06 | 安装 | 单一二进制；运行时只依赖 `git` |
| NFR-07 | 数据可移植 | 单个 SQLite 文件；schema 有文档 |
| NFR-08 | 可升级 | 只前向迁移；有一致性备份；旧程序拒绝打开新 schema |
| NFR-09 | 可诊断 | 日志（不含 C0 / C1 内容）、`doctor` |
| NFR-10 | 路径健壮性 | 路径含空格或中文、文件使用 CRLF 时，功能正常 |
| NFR-11 | 确定性 | 同样的事实（`as_of_seq`）、快照和来源观测哈希，推导结果完全相同；这三项都写在输出中 |

## 8. 不在范围内

见 [mvp.md §3](mvp.md#3-范围外明确不做)。

## 9. 产品待决策事项

"建议默认"是本轮设计的工作假设，需要产品负责人确认。

| ID | 问题 | 建议默认 | 状态 / 最晚决定于 |
|---|---|---|---|
| ~~PD-01~~ | ~~交付就绪的口径~~ | 已由人工确认的交付范围取代（默认选中 `must`） | **已关闭** |
| PD-02 | 主进度指标 | 范围内需求的验收率 + AC 状态计数 | M2 |
| PD-03 | 验收粒度 | **已确认（2026-10-14，产品负责人）**：只在 AC 级验收 | 已确认 |
| PD-04 | AC 修订是否使验收失效 | **已确认（2026-10-14，产品负责人）**：由审核者选择 `material` / `editorial`；部分字段强制 `material`。父需求实质修订后，对 AC 执行 `reaffirm` 会保留它已有的执行与验收。**方案 A**：审核时接受 `update` 条目（包括冲突条目和编辑后接受），审核者必须明确选择 `material` 或 `editorial`，未选择即拒绝（V-R01）；提案中的类别只是建议，不能代替审核者的选择；强制 `material` 的字段继续优先。 | 已确认 |
| PD-05 | MVP 正式支持哪些 Agent | Claude Code；其他 Agent 尽力而为 | M3 |
| ~~PD-06~~ | ~~Stop Hook 提醒~~ | 移出 MVP（P2） | **已关闭** |
| PD-07 | 在仓库中留下的痕迹 | **已确认（2026-10-14，产品负责人）**：把 marker 路径写入 `.git/info/exclude`：默认开启，`--no-exclude` 关闭，文件不可写时只给出警告（architecture §10.6）；该功能本身是 P1（FR-INIT-06） | 已确认 |
| ~~PD-08~~ | ~~交付报告~~ | 移出 MVP（P2） | **已关闭**（需要确认） |
| PD-09 | 界面与文档语言 | 文档用中文；代码与 CLI 用英文；Dashboard 文本外置 | M5 |
| PD-10 | 开源许可证 | 产品负责人**倾向 Apache-2.0**（2026-10-14），**尚未确认**。需要确认的事项和第三方声明方案见 [licensing.md](licensing.md) | 接受外部贡献之前 |
| PD-11 | Local 与 Hosted 的边界 | 本地单人能力完整；Hosted 只增加身份、同步和协作 | 首次公开发布之前 |
| PD-12 | 优先级方案 | **已确认（2026-10-14，产品负责人）**：MoSCoW | 已确认 |
| PD-13 | 多交付阶段 | **已确认（2026-10-14，产品负责人）**：MVP 不支持（另见 PD-19）；分阶段交付通过归档旧目标、新建目标完成（FR-TGT-02） | 已确认 |
| PD-14 | Agent 能否代为记录问题的答复 | **已确认（2026-10-14，产品负责人）**：不能。回答和驳回问题都是人类专属操作 | 已确认 |
| PD-15 | Skill 评测阈值 | 提取召回率 ≥ 80%；`not_found` 引用 ≤ 10%；全量 audit 覆盖率 100%（允许 `unknown` 断言） | M4 |
| **PD-16** | **自动化 AC 的验收沿用**：在新快照上，同一检查配置、同一输入对象 ID 合格通过后，验收自动适用 | **已确认（2026-10-14，产品负责人）**：采纳，但**只接受隔离级证据**（codex-review-002 B1）。MVP 没有隔离执行器，所以 MVP 中目标前进后，自动化 AC 也需要人重新验收（可以用批量验收，FR-UI-14）。手动 AC 和例外验收永不沿用。 | 已确认 |
| **PD-17** | 例外验收（自动化 AC 没有合格执行时，由人作出例外判断）是否可以计入 `ready` | **已确认（2026-10-14，产品负责人）**：可以计入，但只适用于该快照，并带警告。要求：① 状态输出和界面必须显示例外、理由、适用的快照和证据等级（`status.v1` 中每个 AC 的 `decision`）；② `evidence_grade = none` 不得表述为自动化验证通过；③ 例外验收不删除、也不隐藏已有的实际测试证据，之后在快照上记录的执行照常显示；④ 全部豁免的范围不得为 `ready`（PD-18）。 | 已确认 |
| **PD-18** | 范围全部豁免时的判定 | **已确认（2026-10-14，产品负责人）**：`not_assessed`（`ALL_WAIVED`），不给出 `ready` | 已确认 |
| **PD-19** | 交付目标的形态 | **已确认（2026-10-14，产品负责人）**：每个项目只有 1 个 `active` 目标，跟踪分支或标签；只有干净的提交快照才能得到 `ready`；非 git 项目永远不会是 `ready` | 已确认 |
| **PD-20** | 隐私契约中允许本地保存的 C1 内容 | **已确认（2026-10-13，产品负责人）**：采纳建议的 C1 内容（来源片段 ≤ 500 字符、各类路径、检查配置、Claim 摘要、人填写的理由）；Hook 不采集命令文本和文件路径。补充约束：允许保存片段**不等于**允许自动持久化客户文档原文——采集策略、保护措施和用户控制见 [architecture.md §10.7](architecture.md#107-来源片段的采集策略保护与用户控制)；C1 默认只存在本地，未来同步必须经过独立授权（architecture §10.1）。 | 已确认 |
| **PD-21** | 范围的粒度 | **已确认（2026-10-14，产品负责人）**：以需求为单位（需求下的全部 AC 都在范围内；不交付的 AC 用豁免）。要求：① 豁免绑定 AC 的 revision，AC 实质修订后失效，需要重新决定；② 确认范围时，`acknowledged_waivers` 必须覆盖范围内**全部**有效豁免，已失效的历史豁免不得列入（V-T01）；③ 写入时校验用户读取的基准版本、当时的范围确认（`seen_scope_confirmation_id`，V-T02）和豁免所对应的 AC revision，任何一项变化都拒绝过期的确认。 | 已确认 |
| **PD-22** | **MVP 只提供观测级证据**（隔离执行器在 MVP 之后） | **已确认（2026-10-13，产品负责人）**：MVP 的 `ready` 表示人在明确确认观测级风险后完成验收；`observed` 证据在任何文档、界面和输出中都不得被描述为 snapshot-verified 或"快照已验证"。**补充说明（2026-10-14，依据 PD-17 的条件）**：范围内只有手动、例外或豁免的 AC 时，`ready` 的证据等级为 `none`，表示没有任何自动化执行证据，此时没有观测级风险需要确认；`none` 同样不得被描述为已通过自动化验证。 | 已确认 |
| **PD-23** | 无法只读确认版本的数据库（WAL 文件被其他程序删除、只剩其一、存在回滚日志）如何恢复 | **已确认（2026-10-13，产品负责人）**：提供独立的显式恢复命令 `specmetry db recover`，不放进 `doctor`（`doctor` 是非破坏性诊断，不执行恢复）。普通打开和 `doctor` 保持严格：遇到这些状态继续拒绝（退出码 6）或报告版本未知。恢复必须由用户明确授权（仅限人），并实际检查 schema 版本、`min_binary_version` 和数据库一致性，不能仅凭用户声明正在使用最新版本；恢复前必须已有一致性备份，不得直接删除 `-wal` / `-shm`，不得绕过版本兼容限制。依赖、排期评估及与已批准内容的冲突见 [implementation-plan.md §8](implementation-plan.md#8-已决策待排期的功能) | 已确认；**排期 M2**，必须在 M4 自测（dogfooding）开始前完成（2026-10-13） |
| **PD-25** | MVP-SRC-06 中"同一组来源"的含义 | **已确认（2026-10-14，产品负责人）**：指 `source_id` 与 `acked_sha256` 都相同的一组来源记录，而不是路径相同的文件。输入顺序不同，集合哈希相同；`source_id` 改变（例如移除后重新登记），集合哈希改变；人工 `ack` 更新 `acked_sha256`，集合哈希改变；文件发生了变化、但尚未经人工确认，集合哈希不变。哈希算法（data-model §4.3）不变 | 已确认 |
| **PD-24** | 来源片段的默认保存模式（`excerpt_storage`） | **已确认（2026-10-14，产品负责人）**：默认 `hash_only`：片段只在内存中校验，数据库中只保存定位、片段哈希和校验结果。M1 不提供原文持久化能力；`full`（保存原文）及其人工授权和删除保证在 M2 实现（WP2.11）。 | 已确认 |
| **PD-26** | Git 对象读取与按需抓取（Lazy Fetch）的兼容策略（codex-review-022） | **已确认（2026-10-15，产品负责人）**：采用**方案 A**，即运行时探测安全能力，不具备时拒绝读取对象。① 全局最低版本保持 Git 2.32，不提高到 2.44 或其他版本。② 只有经可靠验证、实际能阻止 Lazy Fetch 的 Git，才执行可能读取 Git 对象的操作。能力按实际行为探测：在 Specmetry 自己的临时目录中进行，不使用用户仓库，不联网；不依据版本号，也不依据是否认识 `--no-lazy-fetch` 推断。不认识或行为不确定的 Git 一律按不具备处理（fail-closed）。③ 不具备该能力时，涉及对象读取的操作返回退出码 6。④ Git 项目的 `init` 不能安全完成文件发现与快照读取时，在任何业务写入之前返回退出码 6。不修改 `init.v1` 的结构，读取失败不得转换为 `snapshot = null`。⑤ 不开发私有 Git 元数据视图（方案 B），确有实际需求时再立项。⑥ 不允许任何隐式网络请求、外部传输 Helper 的执行，或未获授权的原仓库写入。实现与实测结果（git 2.32.0 不具备；Ubuntu 的 2.43 与 2.56 具备）见 architecture §6.1.2，验收见 MVP-INIT-12、MVP-BL-29。**已批准的实施细化（2026-10-15，产品负责人）**：`ls-files` 不认识 `--sparse` 的 Git，列出文件时返回退出码 6；即使普通索引也许可以读取，也不在用户仓库中试探。候选来源发现只面向 worktree 中实际可访问的普通文件，不要求列举稀疏检出之外的全部文件。**M2 原则（2026-10-15，产品负责人）**：正确的进度推导依赖 Git 对象、而 Git 无法安全读取时，返回 environment（退出码 6），不把环境失败转换为正常的 `not_assessed`。这一原则记入 implementation-plan 的 M2，在 M2 实现时补齐测试。SHA-256 仓库见 TD-26 | 已确认 |
