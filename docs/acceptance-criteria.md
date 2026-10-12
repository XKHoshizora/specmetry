# Specmetry MVP 验收标准

> 状态：Draft v0.1 · 2026-10-13
> 本文列出 MVP 中可测试的验收项。每一项都对应 [prd.md](prd.md) 中的功能需求（FR）或非功能需求（NFR）。所有 P0 验收项通过，是 MVP 完成的必要条件（见 [mvp.md §7](mvp.md#7-mvp-完成定义)）。
> 从 M4 起，本文将作为 Specmetry 自身的基准导入（dogfooding）。

## 1. 约定

### 1.1 编号与格式

- 编号格式为 `MVP-<领域>-<序号>`，与产品内的 AC 编号（`AC-001-1`）区分开。
- 每一项按"前置 → 操作 → 期望"书写。
- 标注 `[P1]` 的项对应 P1 需求，时间不足时可以推迟。

### 1.2 验证方式

| 标记 | 含义 |
|---|---|
| 单元 | Go 单元测试，不涉及 I/O |
| 集成 | 在临时的 `SPECMETRY_HOME` 和临时仓库中调用真实的二进制 |
| E2E | 完整场景回放（§4） |
| UI | 对内嵌服务和种子数据库运行 Playwright |
| 基准 | 性能基准测试。参考机器为 4 核、16 GB 内存、SSD；CI 上使用阈值的 2 倍作为回归门槛。 |
| 评测 | 在真实 Agent 上运行，人工或半自动评分；不作为 CI 硬性门槛，但属于 MVP 发布门槛 |
| 审查 | 代码或规格审查清单 |
| 人工 | 人工检查 |

除非另有说明，所有"集成"和"E2E"项都必须在 Windows、macOS 和 Linux 的 CI 上通过。

### 1.3 测试夹具（`testdata/fixtures/`）

| 夹具 | 内容 |
|---|---|
| `fx-docs-only` | `docs/client-brief.md`（8 条需求，其中 1 条故意没有可判定的验收描述，另有 2 处歧义），没有源代码 |
| `fx-in-dev` | 同一份 client brief，加上一个部分实现的小应用：5 条需求已实现，其中 1 个测试失败。附带若干历史提交（不含基准编号）。测试脚本会在基准确认后追加带 `Specmetry-Refs:` trailer 的提交。 |
| `fx-incomplete` | 只有两段话的 README，加上实现了 6 个功能的代码，没有需求文档 |
| `fx-scale` | 生成器产生 500 条需求、2000 个 AC、10 万条事件 |
| `fx-hook-payloads` | Claude Code 各类 Hook 输入的样本，在不应保存的字段中埋入 canary 字符串 |
| 录制的 Agent 输出 | 每个项目夹具都附带 `proposal.json`、`claims-audit.json`、`claims-sync.json`，用于确定性回放 |
| 期望结果（oracle） | 每个项目夹具都附带 `expected-status.json`、`expected-gaps.json`、`expected-requirements.json`（基准提取评测的标准答案） |

## 2. 验收项

### 2.1 项目初始化（INIT）

| ID | 验收条件（前置 → 操作 → 期望） | 验证 | 需求 |
|---|---|---|---|
| MVP-INIT-01 | 未初始化的 git 仓库 → `specmetry init --name demo --json` → 退出码 0；生成 `.specmetry/project.json`，其中 `project_id` 是合法的 ULID；`status --json` 返回该项目，且 `init_progress = initialized` | 集成 | FR-INIT-01 |
| MVP-INIT-02 | 已初始化的项目 → 再次执行 `init` → 退出码 0；`project_id` 不变；数据库中仍然只有 1 个项目；输出当前的 `init_progress` | 集成 | FR-INIT-01 |
| MVP-INIT-03 | 三个项目夹具 → `init --json` → 检测到的 `stage` 依次为 `docs_only`、`in_development`、`incomplete_docs`；加上 `--stage` 时以用户指定的值为准 | 集成 | FR-INIT-02 |
| MVP-INIT-04 | `fx-docs-only` → `init --json` → `candidate_sources` 中包含 `docs/client-brief.md`；`source list` 为空（不会自动登记） | 集成 | FR-INIT-03 |
| MVP-INIT-05 | 伪终端中执行 `specmetry setup` → 先展示待写入的 Hook 与 Skill 变更：拒绝时，相关文件逐字节不变；接受时，文件被写入；再次执行不会产生重复的 Hook 条目；执行 `setup --uninstall` 后，配置文件在语义上恢复原样 | 集成 | FR-INIT-04 |
| MVP-INIT-06 | 非 TTY 环境中执行 `specmetry setup` → 退出码 3，没有修改任何文件 | 集成 | FR-INIT-04 |
| MVP-INIT-07 | 按初始化流程逐步推进 → 每一步之后 `init_progress` 都符合 [data-model.md §6.9](data-model.md#69-初始化进度)；`init --skip-sources` 之后可以跳过来源这一步；在没有基准版本时执行 `init --skip-inventory`，`init_progress` 不会越过 `baseline_proposed` 之前的步骤 | 集成 | FR-INIT-05 |
| MVP-INIT-08 | `[P1]` 在 git 仓库中执行 `init` → `.git/info/exclude` 中包含 `.specmetry/`；`.gitignore` 不变；`git status --porcelain` 中不出现 `.specmetry` | 集成 | FR-INIT-06 |
| MVP-INIT-09 | 非 git 目录 → `init` → 成功并给出警告；`doctor` 报告基于 git 的功能已禁用；之后的检查证据中 `git_head` 为 null | 集成 | FR-INIT-09 |
| MVP-INIT-10 | `[P1]` 包含 `go.mod`、`package.json`（带 test 脚本）、`pytest.ini` 的夹具 → `init --json` → `test_hints` 中给出对应的测试命令 | 集成 | FR-INIT-08 |

### 2.2 来源文档（SRC）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-SRC-01 | 登记一个仓库内的文件，以及一个仓库外、路径中含空格和中文的文件 → 两者都被记录，包括 `kind`、`location`、`sha256`、`size_bytes` | 集成 | FR-SRC-01、NFR-10 |
| MVP-SRC-02 | 修改已登记的来源 → `source list` 显示 `changed`，并出现 `SOURCE_CHANGED` 差距 → 执行 `source ack` → 差距消失；删除文件 → 显示 `missing` | 集成 | FR-SRC-02 |
| MVP-SRC-03 | `source add` 一个不存在的路径 → 退出码 2，没有写入任何数据 | 集成 | FR-SRC-01 |
| MVP-SRC-04 | `[P1]` 从 stdin 传入文本作为快照来源 → 生成快照文件，`location = snapshot` | 集成 | FR-SRC-03 |

### 2.3 基准提取与确认（BL）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-BL-01 | 提交 `fx-docs-only/proposal.json` → 退出码 0；条目数与文件一致，全部为 `pending`；其中的 Question 立即为 `open`；指标中的需求数仍为 0 | 集成 | FR-BL-01、FR-BL-05 |
| MVP-BL-02 | 为每条"拒绝"类校验规则（V-P01 ~ V-P06、V-P08、V-P10）准备一个反例 → 逐一提交 → 均返回退出码 2；错误 JSON 中带有指向出错字段的 JSON Pointer；所有表的行数不变 | 集成 | FR-BL-01 |
| MVP-BL-03 | 引用片段校验：只有空白或换行（含 CRLF）不同 → `verified`；改动一个字 → `not_found`；来源是 PDF → `not_checkable`；来源文件已删除 → `source_missing` | 单元 + 集成 | FR-BL-02、FR-SRC-04、NFR-10 |
| MVP-BL-04 | 提交包含 `inferred_from_code` 条目的提案 → `origin` 被保存；`baseline show` 的输出和 Dashboard 中都有"反推"标识 | 集成 + UI | FR-BL-03、FR-UI-11 |
| MVP-BL-05 | 5 个新增条目 → 接受 3 个、拒绝 2 个，然后提交 → 生成基准版本 1，其中恰好包含 3 个新增条目；编号按顺序分配；被拒绝的条目不在基准中 | 集成 | FR-BL-04 |
| MVP-BL-06 | 拒绝提案内的某个需求条目，同时接受指向它的 AC 条目 → 这次审核提交整体失败，不产生版本 | 集成 | FR-BL-04 |
| MVP-BL-07 | 编辑后接受 → `payload_json` 保留原值，`final_payload_json` 保存最终值；基准使用最终值 | 集成 | FR-BL-04 |
| MVP-BL-08 | 在非 TTY 环境中执行 `review`，或用会接受条目的参数调用 → 退出码 3，不产生版本 | 集成 | FR-BL-04、FR-ACC-02 |
| MVP-BL-09 | 已有 v1 → 再提交一个只包含新增需求和 AC（不含问题）的提案，但不审核 → 除了 `pending_review_items` 指标和 `PENDING_REVIEW` 差距，所有指标、差距和状态都不变 | 集成 | FR-BL-05 |
| MVP-BL-10 | 经过 3 个版本之后 → `baseline history --json` 列出每个版本的每一条变更，包括修改前后的值、actor、channel 和来源提案条目 | 集成 | FR-BL-06 |
| MVP-BL-11 | 修改 AC 的 `statement`：选 `material` 时 revision 加 1，选 `editorial` 时不变；修改 `check_command` 时，即使选择了 `editorial`，revision 也会加 1 | 集成 | FR-BL-07 |
| MVP-BL-12 | 通过 Dashboard API 提交一次人工编辑 → 生成一个 `origin = human_edit` 的提案，并在同一个事务中应用为新版本 | 集成 | FR-BL-08 |
| MVP-BL-13 | 提案 A 以 v1 为基础修改 REQ-001；之后提案 B 被接受，生成 v2，也修改了 REQ-001 → A 中的该条目变为 `conflict`；`proposal show --json`（以及 M5 中的 Dashboard API）返回提案值、提案时的值、当前值三方 | 集成 | FR-BL-09 |
| MVP-BL-14 | 退役 REQ-003 之后再新增一个需求 → 新编号不会复用 REQ-003 | 集成 | TD-14 |
| MVP-BL-15 | 退役一个有 2 个 AC 的需求 → 在同一版本中，2 个 AC 都变为 `retired`，并有对应的变更记录 | 集成 | FR-BL-04 |
| MVP-BL-16 | 在交互式终端或 Dashboard 中回答问题，会被记录；在非 TTY 中执行 `question answer` → 退出码 3；存在 `open` 且 `blocking` 的问题时，就绪判定为 `not_ready`，原因中包含 `BLOCKING_QUESTION` | 集成 | FR-BL-10、PD-14 |
| MVP-BL-17 | `[P1]` `baseline export --format md` 与 `--format json` → 输出包含全部 `active` 条目和来源引用；JSON 输出能通过 schema 校验 | 集成 | FR-BL-11 |

### 2.4 现状盘点（INV）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-INV-01 | 提交 audit 批次，其中有引用不存在文件的条目，也有引用未知提交的条目 → Claim 照常写入；对应的 `refs_check_json` 分别为 `missing` 和 `unknown_commit`；响应中带警告 | 集成 | FR-INV-01、FR-INV-02 |
| MVP-INV-02 | 基准确认后，git 扫描新增的提交：带 trailer `Specmetry-Refs: REQ-001` 的提交 → 生成提交类证据和 `relates` 链接（`link_source = commit_trailer`）；正文中提到 `AC-002-1` 的提交 → 生成链接（`commit_message`）；提到不存在编号的提交 → 忽略；不引用任何编号的提交 → 不入库；重复扫描 → 没有重复数据 | 集成 | FR-INV-03、FR-EV-04 |
| MVP-INV-03 | `fx-in-dev` 中有 4 个带检查的 AC（3 个通过、1 个失败）→ `check run --all` → 生成恰好 4 条证据，结果与预期一致；退出码 10 | 集成 | FR-INV-04 |
| MVP-INV-04 | 提交第一个 audit 批次后 → `init_progress = inventoried` | 集成 | FR-INIT-05 |

### 2.5 Agent 自述（CLM）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-CLM-01 | 一个没有证据的 AC → 提交 `assertion = done` 的 Claim → AC 状态为 `claimed`（不是 `verified`），`claim_unverified` 加 1 | 集成 | FR-CLM-01、FR-CLM-02 |
| MVP-CLM-02 | Claim 的目标分别是：只存在于待审提案中的条目、已退役的 AC、不存在的编号 → 均返回退出码 2，不写入任何数据 | 集成 | FR-CLM-05 |
| MVP-CLM-03 | `summary` 超过 500 字符 → 退出码 2；`summary` 中包含一个模拟密钥 → 保存的是 `[REDACTED:…]`，响应中带脱敏警告 | 集成 | FR-CLM-03、FR-PRV-02 |
| MVP-CLM-04 | 提交 Claim → 记录 `criterion_revision` 和 `git_head`；当项目中恰好有一个活跃的 Agent 会话时，`session_id` 关联到这个会话 | 集成 | FR-CLM-04 |
| MVP-CLM-05 | 非 TTY 环境中提交 `kind = manual` 的批次 → 退出码 3，不写入任何数据 | 集成 | V-C06 |
| MVP-CLM-06 | `[P1]` 提交包含 `unmapped_work` 的批次 → 保存为 `target_type = none` 的 Claim；在 Dashboard 活动页中可以看到 | 集成 + UI | FR-INV-06 |

### 2.6 执行证据（EV）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-EV-01 | 一个会通过的检查 → `check run AC-x` → 证据中 `result = pass`、`exit_code = 0`，并有 `duration_ms`、`git_head` 和 `git_dirty = false`；生成带当前 revision 的 `verifies` 链接；退出码 0 | 集成 | FR-EV-01 |
| MVP-EV-02 | 一个会失败的检查 → 证据中 `result = fail`，并记录实际的退出码；CLI 退出码 10 | 集成 | FR-EV-01 |
| MVP-EV-03 | 检查会启动一个长时间睡眠的子进程，超时设为 2 秒 → 结果为 `error`（超时）；之后整个进程树都不存在 | 集成 | FR-EV-02 |
| MVP-EV-04 | 检查命令不存在 → 结果为 `error`，CLI 不会崩溃 | 集成 | FR-EV-02 |
| MVP-EV-05 | 检查输出超过 4 KB，且包含一个模拟密钥 → `output_tail` 不超过 4096 字节，且已脱敏；完整日志文件同样已脱敏；`output_sha256` 已记录 | 集成 | FR-EV-01、FR-PRV-02 |
| MVP-EV-06 | `check run` 没有接受临时命令的参数；对一个没有检查的 AC 执行 → 退出码 2 | 集成 | FR-EV-03 |
| MVP-EV-07 | 直接对数据库中的 `evidence` 或 `acceptances` 执行 UPDATE 或 DELETE → 被触发器中止 | 集成 | FR-EV-05 |
| MVP-EV-08 | 在有未提交改动的工作区运行检查 → `git_dirty = true`；该 AC 带 `dirty_evidence` 标记 | 集成 | FR-EV-01 |
| MVP-EV-09 | `[P1]` 产生证据之后再新增 3 个提交 → 状态输出显示该证据落后 3 个提交 | 集成 | FR-EV-06 |

### 2.7 正式验收（ACC）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-ACC-01 | 一个 `verified` 的 AC → 通过 Dashboard API（带有效令牌）接受 → 写入一条 Acceptance，其中 `channel = dashboard`，`evidence_ids_json` 包含最新证据，并记录了 `git_head`；AC 变为 `accepted` | 集成 | FR-ACC-01 |
| MVP-ACC-02 | 非 TTY 环境中执行 `specmetry accept AC-x` → 退出码 3，不写入数据；在伪终端中执行并完成确认 → 写入，`channel = human_cli` | 集成 | FR-ACC-01、FR-ACC-02 |
| MVP-ACC-03 | 在伪终端中执行，但环境中存在已知的 Agent 环境变量 → 退出码 3 | 集成 | FR-ACC-02 |
| MVP-ACC-04 | 把以下 PreToolUse payload 喂给 `specmetry hook claude-code pre-tool-use`：`specmetry accept AC-001-1`、`cd app && specmetry waive AC-002-1 -m x`、`specmetry review`、`specmetry purge` → 输出拒绝决定，并记录 `guard_denied` 事件；`specmetry status --json` 则被放行 | 单元 + 集成 | FR-ACC-02 |
| MVP-ACC-05 | 已验收的 AC 发生 `material` 修订 → 状态为 `stale` → 运行检查并通过 → `verified` → 再次接受 → `accepted` | 集成 | FR-ACC-03 |
| MVP-ACC-06 | 已验收的 AC 之后检查失败 → `regressed` → 再次运行检查并通过 → `accepted` | 集成 | FR-ACC-04 |
| MVP-ACC-07 | `reject` 或 `waive` 时说明为空 → CLI 返回退出码 2，API 返回 400 | 集成 | FR-ACC-05 |
| MVP-ACC-08 | 对同一 AC 先后作出接受、拒收、接受三次决定 → 三条记录都保留，`supersedes_id` 形成链，当前决定是最后一条 | 集成 | FR-ACC-06 |
| MVP-ACC-09 | 拒收之前已有通过的证据 → 拒收后，如果没有新的事实，状态为 `rejected`；拒收后 Agent 提交 `done` → `claimed`；拒收后再运行检查并通过 → `verified` | 单元 | FR-ST-01（决策表第 4 条） |

### 2.8 状态与差距（ST）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-ST-01 | AC 决策表的 11 条规则，每条至少有一个命中用例；每对相邻规则至少有一个"优先级冲突"用例 | 单元 | FR-ST-01、FR-CLM-02 |
| MVP-ST-02 | 需求聚合表的 12 条规则，每条至少有一个用例 | 单元 | FR-ST-01 |
| MVP-ST-03 | 性质测试：把同一组事实随机打乱插入顺序 1000 次 → 每次的 AC 状态、需求状态、差距和指标都相同 | 单元 | FR-ST-01 |
| MVP-ST-04 | 审查 OpenAPI 规格和 CLI 命令树 → 不存在设置状态、在 Dashboard 中执行检查、删除证据或验收的操作 | 审查 | FR-ST-01、FR-EV-05 |
| MVP-ST-05 | 对每个项目夹具执行 `gaps --json` → 输出与 `expected-gaps.json` 完全一致（类型、严重度、排序） | 集成 | FR-ST-02 |
| MVP-ST-06 | 依次构造以下情形：没有基准版本；有 `must` 需求未验收；有阻塞问题；全部满足 → 前三种为 `not_ready`，原因代码分别为 `NO_BASELINE`、`REQUIREMENT_INCOMPLETE`（附需求编号和相关差距）、`BLOCKING_QUESTION`；最后一种为 `ready` | 单元 + 集成 | FR-ST-03 |
| MVP-ST-07 | `C = done` 且 `E = fail`，以及 `C = not_done` 且 `E = pass` → 两种情况都带 `conflict` 标记，并产生 `CONFLICT` 差距 | 单元 | FR-ST-05 |
| MVP-ST-08 | 在 `fx-scale` 上执行 `status --json` → 在参考机器上耗时少于 1 秒 | 基准 | NFR-04 |
| MVP-ST-09 | `status` 的人类可读输出包含：就绪判定、必需需求验收率、各展示分组的计数、前 5 个差距；与 golden 文件一致 | 集成 | FR-ST-04 |
| MVP-ST-10 | `[P1]` 连续发生变更 → 快照之间的间隔不小于 60 秒；验收和基准变更不受间隔限制，立即生成快照 | 集成 | FR-ST-06 |
| MVP-ST-11 | `[P1]` `specmetry report` → 生成的 Markdown 中包含就绪判定、各需求的状态、验收记录和未决问题 | 集成 | FR-ST-07 |

### 2.9 采集（COL）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-COL-01 | 连续调用 200 次 Hook（PostToolUse payload）：在 Specmetry 项目内，p95 ≤ 100 ms；在非 Specmetry 目录中，p95 ≤ 30 ms。三个平台分别测量。 | 基准 | NFR-02 |
| MVP-COL-02 | 以下情况中 Hook 都以 0 退出，且 stdout 中没有输出：stdin 格式错误、stdin 为空、未知的事件类型、spool 目录不可写、`SPECMETRY_HOME` 不存在 | 集成 | FR-COL-02 |
| MVP-COL-03 | 在非 Specmetry 项目目录中触发 Hook → 不会生成 spool 文件 | 集成 | FR-COL-02 |
| MVP-COL-04 | 把同一批 spool 文件入库两次；模拟"事务已提交、文件还没删除"时崩溃，然后重新入库 → `events` 中没有重复行 | 集成 | FR-COL-03 |
| MVP-COL-05 | **Canary 测试**：`fx-hook-payloads` 在以下字段中埋入了唯一的 canary 字符串——Write 的文件内容、Edit 的 old/new 字符串、tool_response 的输出、提示词字段、transcript_path、命令中的模拟密钥。重放并入库后，对 `$SPECMETRY_HOME` 下的所有文件（数据库、WAL、spool、日志）做字节搜索 → 找不到任何 canary | 集成 | FR-COL-04、FR-PRV-02 |
| MVP-COL-06 | 把 `transcript_path` 设为一个 FIFO（Linux 和 macOS）→ Hook 在时间预算内正常退出，证明它没有打开这个路径 | 集成 | FR-COL-04 |
| MVP-COL-07 | 重放 SessionStart 和 SessionEnd → 生成一条 `agent_sessions` 记录，`head_start`、`head_end` 和 `end_reason` 都正确 | 集成 | FR-COL-01 |
| MVP-COL-08 | Dashboard 运行时触发 Hook → 5 秒内这个事件出现在 `/api/v1/projects/{id}/overview` 的"最近活动"中 | 集成 | FR-UI-10 |
| MVP-COL-09 | 在项目的一个 git worktree 中触发 Hook，并执行 `claim` 和 `check run` → 都归属到同一个项目 | 集成 | FR-COL-08 |
| MVP-COL-10 | 回放 `claims-sync.json`，并运行受影响的检查 → `status --json --diff-since` 列出了状态发生变化的 AC，以及新增的差距 | 集成 | FR-COL-05 |
| MVP-COL-11 | 没有安装任何 Hook → 用 CLI 完成 E2E-2 的流程 → 结果与 oracle 一致；健康页显示"没有 Hook 数据" | E2E | FR-COL-07 |
| MVP-COL-12 | `[P1]` 默认配置下，Stop Hook 不输出任何内容；开启提醒后，输出 sync 提醒 | 集成 | FR-COL-06 |
| MVP-COL-13 | spool 中有一个损坏的文件 → 它被移到 `spool/dead/`，其余文件正常入库 | 集成 | FR-COL-02 |

### 2.10 Agent Skill（SK）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-SK-01 | 执行 `setup` → Skill 被安装到目标目录，版本号与二进制一致；手动改动 Skill 文件后 → `doctor` 报告版本不一致 | 集成 | FR-SK-01 |
| MVP-SK-02 | 静态检查：Skill 文件中出现的每一个 `specmetry …` 调用，都能被 CLI 命令树解析，用到的参数也都存在 | 单元 | FR-SK-03 |
| MVP-SK-03 | 静态检查：Skill 的"禁止事项"部分覆盖了 PreToolUse 拦截清单中的全部人类专属命令 | 单元 | FR-SK-04 |
| MVP-SK-04 | Skill 包中的 JSON Schema 与二进制内嵌的 schema 逐字节一致 | 单元 | FR-SK-03 |
| MVP-SK-05 | 在 Claude Code 中对三个项目夹具执行 `/specmetry init`，然后按 PD-15 评分：提交的提案在 2 次尝试内通过校验；与 `expected-requirements.json` 相比，召回率 ≥ 80%；`not_found` 的引用 ≤ 10%；提交提案后会停下来等待审核；没有任何人类专属命令真正被执行 | 评测 | FR-SK-02、FR-SK-05 |
| MVP-SK-06 | 在 `fx-in-dev` 上确认基准之后执行 `/specmetry audit` → 至少 90% 的已确认 AC 收到了 Claim；没有任何 Claim 指向未确认的编号；检查被执行 | 评测 | FR-SK-02 |
| MVP-SK-07 | `[P1]` audit 为没有检查命令的自动化 AC 提议检查命令，并且是以提案的形式提交 | 评测 | FR-INV-05 |

### 2.11 Web Dashboard（UI）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-UI-01 | 启动 Dashboard → 只监听 `127.0.0.1`；通过本机的非回环地址连接会被拒绝 | 集成 | FR-UI-01 |
| MVP-UI-02 | 没有令牌或令牌错误 → 401；Host 为 `evil.example` → 403；写请求的 Origin 来自其他源 → 403；响应头中包含 `Content-Security-Policy: default-src 'self'` | 集成 | FR-UI-01、FR-PRV-05 |
| MVP-UI-03 | 依次打开所有页面，记录全部网络请求 → 所有请求都是同源的 | UI | FR-PRV-05 |
| MVP-UI-04 | 在 `fx-scale` 上打开概览页 → 在参考机器上首屏少于 2 秒 | 基准 | NFR-04 |
| MVP-UI-05 | 对每个项目夹具 → 概览页显示的就绪判定、验收率、各分组计数、"自述未证实"数，都与 `status --json` 一致 | UI | FR-UI-02 |
| MVP-UI-06 | 基准页显示目标 → 需求 → AC 的树；按状态和优先级筛选后，结果与 CLI 一致；可以展开来源引用的原文片段 | UI | FR-UI-03 |
| MVP-UI-07 | 在审核队列中，接受、编辑后接受、拒绝条目，并对修改类条目选择 `material` 或 `editorial`，然后提交 → 生成新版本；冲突条目会显示三方的值 | UI | FR-UI-04 |
| MVP-UI-08 | 需求详情页中，各 AC 的时间线（自述、证据、验收）按时间排序；可以接受、拒收、豁免；豁免时说明为空，界面不允许提交 | UI | FR-UI-05 |
| MVP-UI-09 | 差距页的分组和排序与 `gaps --json` 一致 | UI | FR-UI-06 |
| MVP-UI-10 | 健康页显示：Hook 与 Skill 的安装状态、spool 积压数量、数据目录路径、保留期 | UI | FR-UI-09 |
| MVP-UI-11 | 页面打开时，在 CLI 中提交一个 Claim → 页面在 5 秒内更新 | UI | FR-UI-10 |
| MVP-UI-12 | "反推"条目、`not_found` 的引用、`dirty_evidence`，都有可见的标识 | UI | FR-UI-11 |
| MVP-UI-13 | 有两个项目时 → 项目列表中显示两个项目，并且可以切换 | UI | FR-UI-12 |
| MVP-UI-14 | `[P1]` 活动页显示会话、提交和事件元数据，不显示任何 C0 级数据 | UI | FR-UI-07 |
| MVP-UI-15 | `[P1]` 趋势图使用状态快照数据，与快照记录一致 | UI | FR-UI-08 |
| MVP-UI-16 | 在当前版本的 Chrome、Edge、Firefox、Safari 中走一遍主流程 | 人工 | mvp.md §4 支持矩阵 |

### 2.12 隐私（PRV）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-PRV-01 | 在禁止出站网络的环境中（如 Linux network namespace），E2E-1 ~ E2E-7 全部通过；代码审查确认，除了监听回环地址，没有任何网络调用 | E2E + 审查 | FR-PRV-01、FR-INIT-07、NFR-01 |
| MVP-PRV-02 | 有两个项目 A 和 B → 在伪终端中对 A 执行 `purge` 并输入项目名确认 → 所有表中 A 的行数为 0，A 的检查日志和 spool 文件被删除，B 的数据不变；在非 TTY 中执行 → 退出码 3 | 集成 | FR-PRV-03 |
| MVP-PRV-03 | 每条内置脱敏规则至少有 2 个应命中的样本和 2 个不应命中的样本 | 单元 | FR-PRV-02 |
| MVP-PRV-04 | `[P1]` 设置事件保留期为 1 天，并伪造时间 → 入库时会清除过期事件；证据和验收不受影响 | 集成 | FR-PRV-04 |
| MVP-PRV-05 | 在 Unix 上，数据目录的权限为 0700 | 集成 | architecture.md §10.5 |

### 2.13 平台与工程（PLT）

| ID | 验收条件 | 验证 | 需求 |
|---|---|---|---|
| MVP-PLT-01 | 三个平台的 CI 上全部测试通过 | 集成 | NFR-05 |
| MVP-PLT-02 | 项目根目录、来源路径、`SPECMETRY_HOME` 中都包含空格和中文 → 初始化、登记来源、检查、Hook 全部正常 | 集成 | NFR-10 |
| MVP-PLT-03 | 用一个合成的 v1 → v2 迁移测试：迁移前生成备份文件；迁移中途失败会整体回滚 | 集成 | NFR-08 |
| MVP-PLT-04 | 在只安装了 git 的干净虚拟机中，发布的二进制能完成 E2E-1 | 人工 | NFR-06 |
| MVP-PLT-05 | `doctor --json` 报告：版本、数据库路径、schema 版本、Hook 安装状态、Skill 版本、spool 积压 | 集成 | NFR-09 |
| MVP-PLT-06 | `docs/data-model.md` 中的表定义与迁移脚本产生的实际 schema 一致（表、字段、唯一约束） | 审查 | NFR-07 |

## 3. 关键不变量（贯穿所有测试）

以下性质任何时候都必须成立。E2E 结束时会自动断言：

1. Agent 通道写入的数据中，没有任何 `acceptances` 记录，也没有任何 `baseline_versions` 记录。
2. 每一条 `verifies` 链接都来自 `check_run` 证据，而每一条 `check_run` 证据执行的命令都与当时基准中的检查命令一致（通过 `command_sha256` 比对）。
3. 不存在 `claimed` 以上的状态是仅由 Claim 推出来的。
4. `$SPECMETRY_HOME` 中找不到任何 canary 字符串。
5. 项目仓库中，除了 `.specmetry/` 和 `.git/info/exclude`（以及选择项目级安装时、经确认写入的 `.claude/settings.local.json`），Specmetry 自身没有修改任何文件。检查命令的副作用不在此列。与运行前的快照比对。

## 4. 端到端场景

所有场景都使用录制的 Agent 输出回放，人工操作通过 API 或伪终端脚本完成，最终结果与 oracle 比对。

| ID | 场景 | 步骤概要 | 期望 |
|---|---|---|---|
| E2E-1 | 只有文档 | `fx-docs-only`：init → 登记来源 → 提交提案 → 审核（接受全部，拒绝 1 个低置信度条目）→ status / gaps | 所有 `must` AC 为 `not_started`；存在 `NEEDS_CRITERIA`（夹具中故意缺少 AC 的那条需求）和 `BLOCKING_QUESTION`；`init_progress = inventoried`；就绪判定为 `not_ready`，并列出原因 |
| E2E-2 | 开发中 | `fx-in-dev`：在 E2E-1 的步骤之后，回放 audit Claim → `check run --all` → 脚本追加一个带 trailer 的提交 → git 扫描 | AC 状态与 `expected-status.json` 一致；"自述完成但检查失败"的 AC 是 `failing` 且带 `conflict`；新提交被正确关联，被引用需求下原本为 `not_started` 的 AC 变为 `in_progress` |
| E2E-3 | 文档不完整 | `fx-incomplete`：init（`incomplete_docs`）→ 提交包含 `inferred_from_code` 条目和问题的提案 → 审核时拒绝 1 个反推条目 → 回答阻塞问题 | 反推标识在 CLI 和 API 中都存在；回答阻塞问题之前，就绪原因中包含 `BLOCKING_QUESTION`，回答之后该原因消失 |
| E2E-4 | 日常循环 | 在 E2E-2 的基础上：回放 Hook 事件（会话开始、编辑、命令、结束）→ 提交 sync Claim → 新增一个带 trailer 的提交 → 运行检查 | 生成会话记录；状态变化输出正确；Dashboard 在 5 秒内反映变化 |
| E2E-5 | 需求变更 | 修改客户文档 → 出现 `SOURCE_CHANGED` → 提交 `update` 提案（对一个已验收的 AC 做 `material` 修订）→ 该 AC 变为 `stale` → 检查通过 → `verified` → 接受 → `accepted`；另走一遍 `editorial` 路径 | `editorial` 修订之后，AC 保持 `accepted` |
| E2E-6 | 交付 | 在 E2E-2 的基础上：回答全部阻塞问题；通过提案为缺少 AC 的 `must` 需求补充 AC → 让所有 `must` AC 的检查通过并逐一接受，其中 1 个附说明豁免；`should` 需求保持未完成 → 就绪判定为 `ready`，并带警告 → 修改代码，使某个已验收 AC 的检查失败，然后运行检查 | 先是 `ready` 加警告；检查失败后 AC 为 `regressed`，就绪判定变为 `not_ready` |
| E2E-7 | Agent 自我验收被阻止 | 回放调用 `specmetry accept` 的 PreToolUse payload；在非 TTY 中直接执行 `accept`、`review`、`question answer` | 拦截的命令被拒绝，非 TTY 调用返回退出码 3；数据库中没有新增验收和版本；存在 `guard_denied` 事件 |

## 5. 需求追溯

| 需求 | 验收项 |
|---|---|
| FR-INIT-01 ~ 09 | INIT-01 ~ 10、INV-04、PRV-01 |
| FR-SRC-01 ~ 04 | SRC-01 ~ 04、BL-03 |
| FR-BL-01 ~ 11 | BL-01 ~ 17 |
| FR-INV-01 ~ 06 | INV-01 ~ 03、SK-07、CLM-06 |
| FR-CLM-01 ~ 05 | CLM-01 ~ 05、ST-01 |
| FR-EV-01 ~ 06 | EV-01 ~ 09、INV-02 |
| FR-ACC-01 ~ 06 | ACC-01 ~ 09、E2E-7 |
| FR-ST-01 ~ 07 | ST-01 ~ 11 |
| FR-COL-01 ~ 08 | COL-01 ~ 13、E2E-4 |
| FR-UI-01 ~ 12 | UI-01 ~ 16、COL-08、BL-04 |
| FR-PRV-01 ~ 05 | PRV-01 ~ 05、COL-05、CLM-03、EV-05、UI-02、UI-03 |
| FR-SK-01 ~ 05 | SK-01 ~ 07 |
| NFR-01 ~ 10 | PRV-01、COL-01、COL-02 ~ 04、ST-08、UI-04、PLT-01 ~ 06、SRC-01、BL-03 |

FR-EV-07（CI 证据，P2）不在 MVP 中，所以没有对应的验收项。

## 6. 里程碑分配

每个里程碑的退出条件是：分配给它的验收项全部通过，并且之前里程碑的验收项仍然通过（见 [mvp.md §6](mvp.md#6-里程碑)）。有些验收项横跨多层，就按"CLI 部分"和"UI / API 部分"拆开分配。

| 里程碑 | 验收项 |
|---|---|
| M0 工程骨架 | PLT-01、PLT-03、PRV-05 |
| M1 基准 | INIT-01 ~ 04、INIT-08、INIT-10；SRC-01、SRC-03、SRC-04；BL-01 ~ 03、BL-04（CLI 部分）、BL-05 ~ 08、BL-10、BL-11、BL-13 ~ 15、BL-17 |
| M2 事实与状态 | INIT-07、INIT-09；SRC-02；BL-09、BL-16；INV-01 ~ 04；CLM-01 ~ 03、CLM-05；EV-01 ~ 09；ACC-02、ACC-03、ACC-05、ACC-06、ACC-07（CLI 部分）、ACC-08、ACC-09；ST-01 ~ 03、ST-05 ~ 11；PRV-02、PRV-03；PLT-06 |
| M3 采集 | INIT-05、INIT-06；CLM-04；ACC-04；COL-01 ~ 07、COL-09、COL-10、COL-12、COL-13；PRV-04；PLT-05 |
| M4 Skill | SK-01 ~ 07 |
| M5 Dashboard | BL-04（UI 部分）、BL-12、BL-13（API 部分）；CLM-06；ACC-01、ACC-07（API 部分）；ST-04；COL-08、COL-11；UI-01 ~ 16 |
| M6 加固与发布 | PRV-01；PLT-02、PLT-04；E2E-1 ~ 7；§3 的全部不变量 |
