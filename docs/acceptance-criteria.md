# Specmetry MVP 验收标准

> 状态：Draft v0.3 · 2026-10-14（依据 [codex-review-001](reviews/codex-review-001.md) 与 [codex-review-002](reviews/codex-review-002.md) 修订；M1 开工准备的修订见 [review-resolution.md §13](review-resolution.md#13-m1-开工准备产品决策与设计收尾2026-10-14)）
> 每一项都对应 [prd.md](prd.md) 中的需求，并标注了所属里程碑。里程碑的退出条件是：该里程碑的全部 P0 项和 CX 项通过。§4 的反例项专门验证"设计本身不会给出错误的交付结论"。从 M4 起，本文作为 Specmetry 自身的基准导入。

## 1. 约定

- 编号：`MVP-<领域>-<序号>`（常规项）、`CX-<序号>`（反例项）、`E2E-<序号>`。
- 每一项按"前置 → 操作 → 期望"书写。标注 `[P1]` 的项不阻塞里程碑。
- "**不写入**"指所有表的行数不变，`seq` 计数器也不变。

| 验证方式 | 含义 |
|---|---|
| 单元 | Go 单元测试，不涉及 I/O |
| 集成 | 在临时的 `SPECMETRY_HOME` 和临时仓库中，经由 CLI 的生产入口执行命令；Windows、macOS、Linux 三个平台。按能证明的内容分为三类，见下文"集成测试的三类证据" |
| E2E | 完整场景回放（§5） |
| UI | Playwright 运行内嵌服务和种子数据库 |
| 基准 | 性能测试。参考机器为 4 核、16 GB、SSD；CI 使用 2 倍阈值 |
| 评测 | 真实 Agent 运行；不是 CI 门槛，但是发布门槛 |
| 审查 | 规格或代码审查清单 |
| 人工 | 人工检查 |

**集成测试的三类证据**（产品负责人 2026-10-17 确认）：

| 类别 | 做法 | 能证明 | 不能证明 |
|---|---|---|---|
| 进程内 | 在测试进程中调用 CLI 的生产入口，即 `cli.Execute` 使用的同一个命令树（`newRoot` 与 `executeRoot`、`executeAs`）。使用真实的临时 SQLite 数据库与 git 仓库。通道可以由测试注入（例如 `human_cli`） | 命令的业务行为：参数与输入的校验、前置条件、写入的数据、退出码的取值、`--json` 输出与 schema、事务与回滚 | 进程的操作系统语义：终端检测、真实进程的环境变量、stdout 和 stderr 的重定向、进程退出码。也不能证明 `cmd/specmetry` 入口的接线。注入通道的测试，不证明通道识别本身 |
| 进程级 | 把测试二进制作为子进程重新运行（helper `cli` 等），调用与正式二进制相同的生产入口。它在伪终端中运行（Windows 上为 ConPTY），或者重定向它的 stdout、stdin | 通道识别、Agent 环境变量、终端与重定向、人工确认的显示与读取、进程的退出码 | 不能等同于正式发布的二进制：helper 是测试二进制，可能额外注册测试专用的命令，也不经过 `cmd/specmetry/main.go` |
| 正式二进制 | 用 `go build -tags nowebui ./cmd/specmetry` 编译的二进制，作为子进程运行（`TestBuiltBinary`、`TestShellCompletionIsRefused` 等）。需要终端时，由测试 helper `exec` 作为启动器，二进制自身的 stdin 与 stderr 是伪终端 | `main` 入口的接线、进程退出码、基本的 JSON 读取及其 schema、MVP-CH-01 与 CH-02 的关键进程边界 | 不能证明发布包本身：内嵌 Dashboard 的默认构建由 CI 的"Build with dashboard"另行冒烟，打包与分发在 M6（TD-15）。它是最小的冒烟集合，不代替逐项的业务验收 |

- 涉及 TTY、Agent 环境变量、stdout 或 stderr 重定向、进程退出码的验收（例如 MVP-CH-01、CH-02），必须有进程级或正式二进制的证据。
- 进程内测试可以作为业务行为的集成验证，但不能不加说明地称为进程级的集成测试。
- 三类证据都在三个平台的 CI 中运行。

### 1.1 测试夹具（`testdata/fixtures/`）

| 夹具 | 内容 |
|---|---|
| `fx-docs-only` | `docs/client-brief.md`：8 条需求，其中 1 条没有可以判定的验收描述，另有 2 处歧义；没有代码 |
| `fx-in-dev` | 同一份 brief，加上一个部分实现的应用：5 条需求已实现，其中 1 个测试失败。还有 `feature/x` 分支，比 `main` 多一个功能。测试脚本可以创建第二个 worktree，并追加带 trailer 的提交。 |
| `fx-incomplete` | 只有两段话的 README，加上实现了 6 个功能的代码 |
| `fx-scale` | 生成器产生 500 条需求、2000 个 AC、10 万条事件 |
| `fx-hook-payloads` | Claude Code 各类事件的样本，包括 PostToolUseFailure、PowerShell、超过 1 MiB 的输入。在非白名单字段中埋入 canary 字符串。 |
| `fx-git-heads` | 各种 HEAD 形态：普通分支、`packed-refs`、分离 HEAD、worktree 的 `.git` 文件、格式异常的 HEAD；在 git 支持时，还包括 reftable |
| 录制的 Agent 输出 | 每个项目夹具都有 `proposal.json`、`claims-audit.json`、`claims-sync.json`。夹具的事实序列是确定的，所以其中的版本号和修订号也是固定的。 |
| 期望结果（oracle） | `expected-status.json`（必须符合 `status.v1`）、`expected-gaps.json`、`expected-requirements.json` |

夹具是普通文件加构造脚本：测试在临时目录中用夹具仓库构造器（WP0.5）建立 git 仓库、分支和 worktree，仓库中不提交嵌套的 `.git`。M1 需要 `fx-docs-only`（含录制的 `proposal.json`；另含 `README.md`、`CHANGELOG.md`、`node_modules/x/README.md` 和空的 `docs/empty.md`，用于候选来源的排除规则）、`fx-in-dev` 的文件与分支脚本、`fx-incomplete`（implementation-plan WP1.0）。`fx-in-dev` 的失败测试和录制的 Claim、其余夹具和 oracle，按各自的里程碑补齐（WP2.9 等）。构造器是 `testutil.BuildFixture`，目录约定见 [testdata/fixtures/README.md](../testdata/fixtures/README.md)：每个夹具的 `fixture.json` 列出提交和分支，提交时间与身份固定、git 配置与本机隔离，所以三个平台构造出的提交 ID 相同，并固定在 `expected_refs` 中。

## 2. 常规验收项

### 2.1 项目初始化（INIT）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-INIT-01 | 未初始化的 git 仓库 → `init --name demo --json` → 退出码 0；生成 marker，`project_id` 是合法的 ULID；`init_progress = initialized` | 集成 | FR-INIT-01 | M1 |
| MVP-INIT-02 | 已初始化 → 再次执行 `init` → `project_id` 不变，仍然只有 1 个项目 | 集成 | FR-INIT-01 | M1 |
| MVP-INIT-03 | 三个项目夹具 → `init --json` → `stage_detected` 依次为 `docs_only`、`in_development`、`incomplete_docs`（规则见 [architecture.md §6.1.1](architecture.md#611-阶段识别与候选来源发现m1-实现)）；`--stage` 覆盖后 `stage` 改变，`stage_detected` 保持识别结果；识别过程不读取文件内容（在文件中埋入 canary，数据目录中找不到） | 集成 | FR-INIT-02 | M1 |
| MVP-INIT-04 | `fx-docs-only` → `init --json` → 候选来源中有 `docs/client-brief.md`，并排在 `README.md` 之前；`CHANGELOG.md`、`node_modules/x/README.md` 和空文件 `docs/empty.md` 不在其中；`source list --json`（`source-list.v1`）的 `sources` 为空。WP1.1 验证候选来源的内容与顺序，以及 `init` 不登记来源（`init_progress` 仍为 `initialized`）；`source list` 一项已在 WP1.4 验证 | 集成 | FR-INIT-03 | M1 |
| MVP-INIT-05 | 在伪终端中执行 `setup` → 先展示改动：拒绝时，文件逐字节不变；接受时写入；再次执行不产生重复条目；`setup --uninstall` 后语义上恢复原样 | 集成 | FR-INIT-04 | M3 |
| MVP-INIT-06 | 非 TTY 环境执行 `setup` → 退出码 3，不修改任何文件 | 集成 | FR-INIT-04 | M3 |
| MVP-INIT-07 | 按流程推进 → 每一步之后 `init_progress` 都符合 [data-model.md §6.12](data-model.md#612-初始化进度)；没有基准版本时执行 `init --skip-inventory`，进度不会越过 `baseline_proposed` 之前的步骤 | 集成 | FR-INIT-05 | M2 |
| MVP-INIT-08 | `[P1]` 写入 `.git/info/exclude`；把该文件设为只读后再执行 `init` → 退出码 0，并给出警告 | 集成 | FR-INIT-06 | M1 |
| MVP-INIT-09 | 非 git 目录 → `init` 成功，`target = null`；交付状态为 `not_assessed`（`NO_TARGET`）；检查可以运行，但 `qualifying = false`。`init` 一侧（成功、`target = null`、`not_a_git_repository` 警告、从子目录再次执行仍是同一项目）已在 WP1.1 验证 | 集成 | FR-INIT-09、FR-TGT-03 | M2 |
| MVP-INIT-10 | `[P1]` 有 `go.mod`、`package.json`、`pytest.ini` 的夹具 → 给出测试命令提示 | 集成 | FR-INIT-08 | M1 |
| MVP-INIT-11 | 主 worktree 在 `main` 分支 → `init --json` 中默认交付目标的 `ref = refs/heads/main`；主 worktree 处于分离 HEAD → `target = null`，并给出 `detached_head_no_target` 警告（M1）；差距中有 `NO_TARGET`（M2）。M1 还验证：在关联 worktree 中首次执行时，目标仍是主 worktree 的分支；分支还没有提交时，目标的 `snapshot = null`；分支已有提交、但提交对象无法读取时，不输出 `snapshot = null`，而是退出码 6（MVP-INIT-12） | 集成 | FR-INIT-10 | M1 / M2 |
| MVP-INIT-12 | **git 对象读取的兼容性前提**（PD-26，产品确认，2026-10-15）：① git 不具备安全读取对象的能力（设置 `GIT_NO_LAZY_FETCH` 后读取缺失对象仍会抓取，例如 git 2.32.0；或者无法判定）→ 在 git 项目中执行 `init`（新项目，或已登记的项目）→ 退出码 6；数据目录（不存在时不创建）、项目登记、marker、exclude、`seq` 和 `.git` 都不变；没有运行任何可能读取用户仓库对象的 git 命令；非 git 项目的 `init` 不受影响。② git 具备该能力 → `init` 照常：候选来源和快照正确；主 worktree 分支的提交对象无法读取（例如部分克隆缺少它）→ 退出码 6，不写入，promisor remote 不运行；已登记项目的目标 ref 的提交无法读取 → 退出码 6，写事务回滚，`seq` 不变。③ 稀疏索引不展开：列出文件后 `.git` 逐文件不变（包括修改时间），稀疏检出之外的树对象缺失时不抓取；`ls-files` 不认识 `--sparse` 的 git → 列出文件返回退出码 6，没有在用户仓库中运行 `ls-files`（PD-26 已批准的实施细化）。④ 能力按行为探测：真实 git 2.32.0 判定为不具备，Ubuntu 的 git 2.43.0（`1:2.43.0-1ubuntu7.3`）和 git 2.56 判定为具备；探测的临时目录在成功、失败、超时和取消后都被删除，无法判定的结论不复用 | 集成（三个平台用替身；真实 git 2.32.0 / 2.43 / 2.56 兼容性检查） | FR-INIT-01、FR-INIT-10 | M1 |
| MVP-INIT-13 | **SHA-256 仓库不受支持**（TD-26，2026-10-15）：真实的 SHA-256 仓库（分支有提交、没有提交、分离 HEAD 三种）→ `init` → 退出码 6。数据目录不存在时不创建；已有其他项目时，数据目录逐文件不变，`seq` 不变，没有登记项目；仓库（包括 marker、exclude 和 `.git`）逐文件不变。SHA-1 仓库照常初始化：有提交时快照正确，分支还没有提交时 `snapshot = null` | 集成 | FR-INIT-01、FR-INIT-10 | M1 |

### 2.2 来源（SRC）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-SRC-01 | 登记一个仓库内的文件，和一个仓库外、路径含空格和中文的文件 → 都记录了路径、类型、`acked_sha256` 和大小 | 集成 | FR-SRC-01 | M1 |
| MVP-SRC-02 | 修改已登记的来源 → `source list --json`（`source-list.v1`）显示 `changed`，`current_sha256` 为新哈希；删除文件 → 显示 `missing`，`current_sha256 = null` | 集成 | FR-SRC-02 | M1 |
| MVP-SRC-03 | 非 TTY 执行 `source ack` → 退出码 3；在 TTY 中执行，但 `observed_sha256` 与磁盘不一致 → 退出码 5；一致时 → 写入 `source_acks`（包括理由），`acked_sha256` 更新，来源集合哈希改变 | 集成 | FR-SRC-03 | M1 |
| MVP-SRC-04 | 非 TTY 执行 `source remove` → 退出码 3；在 TTY 中执行 → 写入 `removed_seq`，已有引用显示为 `source_removed`，来源集合哈希改变。WP1.4 验证其余部分；WP1.5 验证引用在移除后保留，并在 `source list --include-removed --refs` 中列在已移除的来源下；引用以 `source_removed` 的形式出现在状态与差距中，随 M2 验证（M2-Q8，2026-10-18 批准）：引用了已移除来源的需求与 AC 在 `status --json` 中带 `source_removed` 标记，目标、需求与 AC 按范围筛选产生不阻塞的 `SOURCE_REMOVED` 警告；移除来源之后先出现阻塞的 `SCOPE_STALE`，重新确认范围之后只剩警告，警告不替代、消除或推迟 `SOURCE_CHANGED` 与 `SCOPE_STALE`（data-model §6.6、§6.10）。里程碑标签由 M1 改为 M1 / M2，只同步已有的 M2 义务，不重新打开 M1 | 集成 | FR-SRC-04 | M1 / M2 |
| MVP-SRC-05 | `source add` 一个不存在的路径 → 退出码 2，不写入 | 集成 | FR-SRC-01 | M1 |
| MVP-SRC-06 | 同一组来源记录（`source_id` 与 `acked_sha256` 均相同）以不同顺序输入 → 来源集合哈希相同；`source_id` 改变（移除后重新登记），或人工 `ack` 更新了 `acked_sha256` → 哈希改变；文件发生变化、但尚未经人工确认 → 哈希不变（PD-25） | 单元 + 集成 | FR-SRC-05 | M1 |
| MVP-SRC-07 | `source add internal/auth/login.go`（非文档类型）→ 退出码 2；在 TTY 中加上 `--allow-any-type` → 成功；修改已登记的来源后，Agent 再次 `source add` 同一路径 → 什么都不做，`acked_sha256` 不变，状态仍为 `changed`（M1）；`SOURCE_CHANGED` 阻塞差距仍然存在，交付状态受其影响（M2）。审查确认 Specmetry 只为引用校验读取文本，不解析文档语义（M1，独立审查）。这一划分与 WP1.4 验收时的记录一致（review-resolution §24），不是新的推迟：M1 的来源登记与确认行为都在 M1 验证 | 集成 + 审查 | FR-SRC-01、FR-SRC-06 | M1 / M2 |
| MVP-SRC-08 | **M1**（`hash_only` 项目）：在 TTY 中执行 `source remove --erase-excerpts` → 来源被移除，引用的 `locator` 和 `excerpt_sha256` 保留，`excerpt` 仍为 NULL，`source list --include-removed --refs --json` 中为 `excerpt_state = not_stored`；非 TTY 执行 → 退出码 3，不写入。**M2**（`full` 模式与 N1 修复之后）：该来源所有已保存的片段变为 NULL，`excerpt_erased_seq` 已写入，`excerpt_state = erased`；整理完成后，对数据库文件和 WAL 做字节搜索，找不到原片段；整理被其他连接的读事务阻塞时，保留 `.compaction-pending` 并报告"整理未完成"，之后的打开会完成整理。WP1.4 验证 M1 中移除、非 TTY 拒绝，以及 `hash_only` 下不擦除任何内容；引用的 `locator`、`excerpt_sha256` 与 `excerpt_state = not_stored` 需要 WP1.5 写入的引用，随 WP1.5 验证 | 集成 | FR-SRC-07 | M1 / M2 |

### 2.3 基准（BL）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-BL-01 | 提交 `fx-docs-only/proposal.json` → 条目为 `pending`，其中的 Question 为 `open`（M1）；推导结果中没有任何需求级或 AC 级的条目，差距只有项目级的（`NO_BASELINE`、`NO_SCOPE` 等）和 `PENDING_REVIEW`（M2） | 集成 | FR-BL-01、05 | M1 / M2 |
| MVP-BL-02 | 每条"拒绝"规则（V-P01、V-P03 ~ V-P06、V-P08）各一个反例 → 都返回退出码 2，错误中带 JSON Pointer，不写入 | 集成 | FR-BL-01 | M1 |
| MVP-BL-03 | `base_version` 过期，或任一 `target_revision` 过期 → 退出码 5，`error.current` 中给出当前值，不写入 | 集成 | FR-BL-01 | M1 |
| MVP-BL-04 | 片段校验：只有空白或换行（含 CRLF）不同 → `verified`；改动一个字 → `not_found`；PDF → `not_checkable`；文件已删除 → `source_missing`；片段超过 500 字符 → 退出码 2 | 单元 + 集成 | FR-BL-02 | M1 |
| MVP-BL-05 | 反推条目没有 `code_refs` → 退出码 2；有引用时 → 保存为 `inferred`（M1）；`status --json` 中带 `inferred` 标记，反推的需求标在需求的 `flags`、反推的 AC 标在 AC 的 `flags`（M2；M2-Q8）；UI 中有标识（M5） | 集成 | FR-BL-03 | M1 / M2 / M5 |
| MVP-BL-06 | 5 个新增条目，接受 3 个、拒绝 2 个 → 生成版本 1，恰好包含这 3 个；编号按顺序分配 | 集成 | FR-BL-04 | M1 |
| MVP-BL-07 | 拒绝提案内的某个需求，同时接受指向它的 AC → 整次审核提交失败，不写入 | 集成 | FR-BL-04 | M1 |
| MVP-BL-08 | 编辑后接受 → 同时保存提议的值和最终值；基准使用最终值 | 集成 | FR-BL-04 | M1 |
| MVP-BL-09 | 非 TTY 执行 `review` → 退出码 3 | 集成 | FR-BL-04 | M1 |
| MVP-BL-10 | 审核提交的 `seen_base_version` 已过期 → 退出码 5，不产生版本 | 集成 | FR-BL-04 | M1 |
| MVP-BL-11 | 经过 3 个版本后 → `baseline history --json`（`baseline-history.v1`）列出每一条变更（包括修改前后的值、actor、通道、来源条目） | 集成 | FR-BL-06 | M1 |
| MVP-BL-12 | `statement` 修改选 `material` 时 revision 加 1，选 `editorial` 时不变；修改 `verification`、检查命令、工作目录、shell 或 `inputs` 时，即使选择 `editorial`，revision 也一律加 1，并按 `material` 记录 | 集成 | FR-BL-07 | M1 |
| MVP-BL-13 | 在 TTY 中执行 `propose --apply` → 生成 `origin = human_edit` 的提案并在同一事务中应用；Agent 通道提交 `origin = human_edit` → 退出码 3 | 集成 | FR-BL-08 | M1 |
| MVP-BL-14 | 产生冲突的条目，审核时没有 `acknowledged_conflict` → 退出码 2；`proposal show --json`（`proposal-show.v1`）给出 `base`（提案读取的版本中的值）、`current`、`proposed` 三方的值；revision 不变的 `editorial` 修改造成的冲突同样给出；实现该输出时补充 schema 条件约束与反例：`type` 与 `proposed`、`base`、`current` 的数据形态一致 | 集成 | FR-BL-09 | M1 |
| MVP-BL-15 | 退役 REQ-003 后再新增 → 不复用 REQ-003；同一需求下的 AC 按 `AC-<n>-1`、`AC-<n>-2` 递增 | 集成 | TD-14 | M1 |
| MVP-BL-16 | 退役有 2 个 AC 的需求 → 在同一版本中级联退役这 2 个 AC，并逐条记录 | 集成 | FR-BL-04 | M1 |
| MVP-BL-17 | Agent 通道执行 `question answer` 或 `question dismiss` → 退出码 3，问题仍为 `open`；在 TTY 中回答或驳回 → 被记录，`question list --all --json`（`question-list.v1`）中显示回答者和通道；实现该输出时补充 schema 条件约束与反例：`answered` 的回答动作只能是 `answer`，`dismissed` 只能是 `dismiss` | 集成 | FR-BL-10 | M1 |
| MVP-BL-18 | 需求以 `material` 方式更新，审核时不附带 AC 的后续处理 → 它的 AC 在 `status --json` 中的 `requirement_revision` 小于父需求的 revision，交付状态为 `needs_reverification`（`parent_revised`） | 集成 | FR-BL-11 | M2 |
| MVP-BL-19 | 审核时附带 `criterion_followups`（`reaffirm` / `retire` / `defer` 各一个）→ 在同一版本中生效；`defer` 的 AC 保持不一致 | 集成 | FR-BL-11 | M1 |
| MVP-BL-20 | Agent 提交 `reaffirm` 提案，人接受 → AC 的 `requirement_revision` 等于父需求的 revision，AC 自身的 revision 不变 | 集成 | FR-BL-11 | M1 |
| MVP-BL-21 | 检查没有 `rationale` → 退出码 2；命令中有模拟密钥 → 退出码 2，提示改用环境变量；`inputs` 或 `workdir` 越出仓库根 → 退出码 2 | 集成 | FR-BL-12、FR-PRV-07 | M1 |
| MVP-BL-22 | 以 `material` 方式更新条目，但不提供新的引用 → 沿用上一修订的引用，重新校验后写成新行（`entity_revision` 为新值），旧行保留 | 集成 | FR-BL-13 | M1 |
| MVP-BL-23 | `propose --apply` 中的 `update` 条目没有 `change_class` → 退出码 2；描述中含有代码围栏，或标题有多行 → 退出码 2 | 集成 | FR-BL-08、FR-PRV-02 | M1 |
| MVP-BL-24 | **应用顺序**：同一提案中，以 `material` 方式修订 REQ-002，并 `reaffirm` AC-002-1，同时新增 AC-002-3 → 接受后，AC-002-1 和 AC-002-3 的 `requirement_revision` 都等于 REQ-002 的新 revision；同一次审核中被接受的 AC 更新，也会刷新 `requirement_revision`；全部条目都被拒绝时，不生成版本 | 集成 | FR-BL-04、FR-BL-11 | M1 |
| MVP-BL-25 | 问题的 `about` 指向提案内的临时引用 → 该条目被接受后，问题关联到新编号；该条目被拒绝后，问题不再计为阻塞（M2） | 集成 | FR-BL-10 | M1 / M2 |
| MVP-BL-26 | **前置**：项目的 `excerpt_storage = full`（WP2.11）。对同一来源反复提交片段，已保存原文的总量超过上限（文本长度的 50% 或 20000 字符）→ 退出码 2。M1 只有 `hash_only`，不计算累计总量；M1 验证的是单个片段超过 500 字符时退出码 2（BL-04），以及原文不落盘（PRV-08） | 集成 | FR-BL-02、FR-PRV-02 | M2 |
| MVP-BL-27 | **审核必须选择修订类别**（PD-04 方案 A，V-R01）：接受 Agent 提案中的 `update` 条目（包括 `accepted_with_edits` 和冲突条目），但没有 `change_class` → 退出码 2，错误指向 `/items/<i>/change_class`，不写入，提案中建议的类别不会被采用；选择 `editorial` 但改动了强制字段 → 按 `material` 记录 | 集成 | FR-BL-07 | M1 |
| MVP-BL-28 | `propose --json` 的输出符合 `proposal-receipt.v1`：每个条目有 `item_id` 和片段校验结果，代码引用有 `exists`；输出、错误信息和日志中都找不到片段原文（canary）；实现该输出时补充 schema 条件约束与反例：Agent 提案（`applied_version = null`）的条目只能是 `pending` 且 `assigned_key = null`，人工编辑的条目为 `accepted` | 集成 | FR-BL-01、FR-BL-02 | M1 |
| MVP-BL-29 | git 不具备安全读取对象的能力（PD-26，见 MVP-INIT-12）→ 含代码引用的 `propose` → 退出码 6，不写入、不分配 `seq`，`.git` 不变，没有运行任何可能读取对象的 git 命令；不含代码引用的 `propose`，以及 `source add`、`list`、`ack`、`remove` 不受影响 | 集成（替身；真实 git 2.32.0 兼容性检查） | FR-BL-01 | M1 |

### 2.4 交付目标与范围（TGT）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-TGT-01 | 非 TTY 执行 `target set` → 退出码 3；在 TTY 中执行 → 旧目标被归档，新目标被创建（M1）；推导结果中出现 `NO_SCOPE`（M2） | 集成 | FR-TGT-01、02 | M1 / M2 |
| MVP-TGT-02 | `target set --ref refs/remotes/origin/main` → 退出码 2（只接受分支和标签） | 集成 | FR-TGT-01 | M1 |
| MVP-TGT-03 | 非 TTY 执行 `scope confirm` → 退出码 3 | 集成 | FR-TGT-04 | M1 |
| MVP-TGT-04 | 需求清单为空 → 退出码 2 | 集成 | FR-TGT-04 | M1 |
| MVP-TGT-05 | `baseline_version` 过期，或 `source_set_hash` 过期 → 退出码 5，不写入 | 集成 | FR-TGT-04 | M1 |
| MVP-TGT-06 | `pending_items_excluded` 与服务端的计数（包括 `conflict` 条目；在审核中附带时，以审核完成后的数量为准）不一致 → 退出码 5；`incomplete_docs` 项目缺少 `missing_goals_reviewed = true` → 退出码 2 | 集成 | FR-TGT-04 | M1 |
| MVP-TGT-07 | **首次审核时一次性确认范围**：审核中接受 3 个新增需求，附带的范围用这 3 个条目的 `item_id` 表示 → 新版本与范围确认在同一事务中写入，`item_id` 被解析为新编号，范围绑定新版本号；范围引用了一个被拒绝的条目，或者范围无效 → 整次审核提交失败 | 集成 | FR-TGT-06 | M1 |
| MVP-TGT-08 | 交互式 `scope confirm` 默认选中全部 `must` 需求，可以修改 | 集成 | FR-TGT-07 | M1 |
| MVP-TGT-09 | 范围确认之后，审核生成了新版本但没有附带范围 → `status --json` 中 `scope.stale = true`，出现阻塞差距 `SCOPE_STALE`；人确认一个新来源后，同样出现 `SCOPE_STALE` | 集成 | FR-TGT-05 | M2 |
| MVP-TGT-10 | **范围确认时确认已有豁免**（PD-21，V-T01）：`scope show --json`（`target-scope.v1`）列出范围内每条有效豁免及其 AC、需求和理由，已失效的历史豁免不出现；确认（独立提交，或附带在审核中，以审核完成后的状态为准）时 `acknowledged_waivers` 少一项、多一项，或列入了已失效的豁免 → 退出码 5，`error.current` 中给出当前的有效豁免，不写入；一致时写入，并在范围确认中保存。M1 验证没有豁免时的写入侧与输出；M2 用真实的豁免验证，并验证 AC 实质修订、父需求修订或新决定之后，该豁免不再出现在有效集合中 | 集成 | FR-TGT-04 | M1 / M2 |
| MVP-TGT-11 | **防止旧页面覆盖范围确认**（PD-21，V-T02）：两个会话都读到同一个范围确认 C1 → 会话乙确认范围（写入 C2）→ 会话甲用 `seen_scope_confirmation_id = C1` 确认 → 退出码 5，不写入，`error.current` 中给出 C2；首次确认时 `seen_scope_confirmation_id = null`，此时已经存在确认 → 退出码 5；会话甲重新读取后用 C2 确认 → 成功。附带在审核中的确认同样适用 | 集成 | FR-TGT-04 | M1 |

### 2.5 Worktree（WT）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-WT-01 | `git worktree add` 新建 worktree，在其中执行 `init --json`（M1）→ `created = false`，`project.id` 与主 worktree 相同，数据库中新登记了这个 worktree（`worktree_id` 不同）；在其中执行 `status --json`（M2）→ `workspace.worktree_id` 与主 worktree 不同，项目相同 | 集成 | FR-WT-01 | M1 / M2 |
| MVP-WT-02 | AC 的检查为 `test -f only-on-feature.txt`，该文件只存在于 `feature/x` 分支。在 feature worktree 中执行 `check run` → `pass`，`worktree_id` 为 feature worktree；在主 worktree 中执行 → `fail` | 集成 | FR-WT-03 | M2 |
| MVP-WT-03 | 每条 Claim 批次、检查执行、会话和事件都有 `worktree_id` 和完整的代码上下文 | 集成 | FR-WT-02 | M2 / M3 |
| MVP-WT-04 | 在 `feature/x` 上提交带 trailer 的提交 → 被发现，且只发现一次；在 `feature/x` 上执行 `reset` 并提交新提交 → 该 ref 被重新扫描，不产生重复 | 集成 | FR-WT-04 | M2 |
| MVP-WT-05 | 在 feature worktree 中执行 `status --json` → `workspace.is_target_snapshot = false`；`delivery` 部分与在主 worktree 中执行时完全相同 | 集成 | FR-WT-05 | M2 |
| MVP-WT-06 | **扫描的范围**（WP2.3 规范 §5.1、§5.2；D2、D7 为产品负责人 2026-10-18 的决定；D4、D5 为审查认可的工程细化；规范已冻结，2026-10-19）：分离 HEAD 上的提交提到已有编号 → 入库，包括未登记的 worktree；分离 HEAD 的游标以规范化的 worktree 路径为身份：移动后的路径是新游标、按新 ref 读取，在原路径上新建的 worktree 沿用原游标，仍按旧 tip 的存在性与祖先关系决定增量还是重扫；从已扫描的 `main` 新建分支并提交一个新提交 → 只读取新提交；分支名或分离 HEAD 的路径命中密钥规则 → 拒绝整次扫描：退出码 2（`invalid_input`），不写提交、关联、`seq` 或游标（所有表的行数与 `seq` 计数器不变），不回显，不只跳过该 ref；git 不支持 `worktree list -z` → 退出码 6，不按行解析；项目还没有需求或 AC → 不扫描、不建立游标；**浅克隆**（产品负责人 2026-10-19 决定 P3）→ 退出码 6，不写提交、关联、`seq` 或游标，不显示为成功的空扫描，不自动拉取或修改浅边界；`shallow` 项存在但不可读或是目录 → 无法确认，同样退出码 6；浅克隆期间被拒绝、之后恢复为完整仓库 → 没有游标时为第一次扫描，已有游标的按增量或重扫，没有遗漏 | 集成 | FR-WT-04 | M2 |

### 2.6 现状盘点（INV）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-INV-01 | `audit` 批次没有 `audit_scope` → 退出码 2；`sync` 批次中使用 `unknown` 断言 → 退出码 2。（WP2.4 规范 D2、D3，2026-10-20 负责人决定，规范已冻结：`audit_scope` 中有不存在或已退役的需求 → 退出码 2；`audit` Claim 的目标 AC 不属于声明的需求 → 退出码 2，在写锁下判断；只覆盖范围内部分 AC 或空 `claims` 仍合法；这些退出码 2 只在 `base_version` 一致时适用，读取之后的合法退役会推进基准版本，先返回退出码 5，见审查 065 B1） | 集成 | FR-INV-01 | M2 |
| MVP-INV-02 | 引用了不存在的文件或未知的提交 → 照常写入，并标记为 `missing` 或 `unknown_commit`。（WP2.4 规范，已冻结：回执中为 `exists = false` / `resolved = false`，与 `refs_check_json` 一致；E4：真实歧义的前缀同样为 `resolved = false`，对象读取失败为退出码 6） | 集成 | FR-INV-02 | M2 |
| MVP-INV-03 | 4 个带检查的 AC → `check run --all` → 恰好 4 条执行记录，都属于当前 worktree；有任何一条不是 `pass` 时，退出码为 10 | 集成 | FR-INV-04 | M2 |
| MVP-INV-04 | 全量 audit → `inventory_coverage = 1`，`init_progress = inventoried` | 集成 | FR-INV-07 | M2 |
| MVP-INV-05 | `[P1]` `init --json` 返回提交数和最近提交时间 | 集成 | FR-INV-03 | M1 |
| MVP-INV-06 | `[P1]` 提交的 `unmapped_work` 被保存为 `target_type = none` 的 Claim | 集成 | FR-INV-06 | M2 |

### 2.7 Agent 自述（CLM）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-CLM-01 | 提交 `done` 的 Claim → 进度为 `claimed`；交付状态仍然是 `in_progress`（`claimed_without_evidence`）；`metrics.claimed_without_evidence` 加 1 | 集成 | FR-CLM-01、02、FR-ST-07 | M2 |
| MVP-CLM-02 | 目标只存在于待审提案中、已退役或不存在 → 退出码 2。（WP2.4 规范，审查 065 B1 后的优先级，已冻结：输入携带当前的 `base_version` 时如此；读取之后另一个审核退役了目标时，基准版本已推进，返回退出码 5） | 集成 | FR-CLM-05 | M2 |
| MVP-CLM-03 | 摘要超过 500 字符、包含代码围栏、包含空行、包含模拟密钥 → 都返回退出码 2 | 集成 | FR-CLM-03 | M2 |
| MVP-CLM-04 | 写入的批次记录了 `worktree_id`、`head`、`tree_state` 和 `base_version`；同一 worktree 中恰好有一个活跃会话时，关联该会话。（WP2.4 规范，已冻结：成功输出按 `claim-receipt.v1` 校验，其 `context` 与写入的批次一致；D1：工作区状态读取的环境错误 → 退出码 6，不写入、不分配 `seq`） | 集成 | FR-CLM-04 | M2 / M3 |
| MVP-CLM-05 | 非 TTY 提交 `kind = manual` → 退出码 3 | 集成 | FR-CLM-05 | M2 |

### 2.8 执行证据（EV）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-EV-01 | 在干净的 worktree 中执行会通过的检查 → 执行记录包含 [data-model.md §4.11](data-model.md#411-check_runs) 列出的全部字段，`qualifying = true`；数据库中没有命令文本和输出 | 集成 | FR-EV-01 | M2 |
| MVP-EV-02 | 会失败的检查 → `fail`、`nonzero_exit` 和实际退出码；CLI 退出码为 10 | 集成 | FR-EV-01 | M2 |
| MVP-EV-03 | 检查启动一个长时间睡眠的子进程（含后台、双重 fork、嵌套 `sh`；Windows 上含 Git for Windows `sh` 的后代），超时设为 2 s → `error` / `timeout`；之后本次检查的进程组（Unix）或 Job（Windows）中没有存活的进程，Linux 上排他区间内被收养的后代也已结束；归属未确认的进程从未收到信号；结束上下文在确认之后才读取。Linux 的启动前提（排他区间或所需能力）不成立 → 不启动，退出码 6，没有新行。**保证调整**（2026-10-25 产品决定 D7，原文为“之后整个进程树都不存在”）：保证范围之外的进程不在这一断言之内，文本输出不称“所有检查进程都已结束”（WP2.5 规范 §8.7） | 集成 | FR-EV-02 | M2 |
| MVP-EV-04 | **按解释器的最终状态分类**（2026-10-25 产品决定 D5，原文为“检查命令不存在 → `error` / `not_found`”；WP2.5 规范 §6、§14）：命令串最终以非零状态结束（例如单独一条不存在的命令：`sh`、`bash` 为 127，`cmd`、`pwsh`、`powershell` 为 1）→ `fail` / `nonzero_exit`，`exit_code` 为解释器的最终退出码，`failure_kind` 不是 `not_found`；主动以同一数值退出的命令（`exit 127`、`exit /b 1`、`exit 1`）得到相同的记录；命令串中有不存在或失败的命令、但最终以 0 结束（`missing; :`、`missing || true`、`false | true`、`cmd` 的 `missing & exit /b 0`、`powershell` 的 `missing; exit 0`）→ 上下文一致时为 `pass`，这是已披露的能力边界；上下文变化、输入缺失、进程无法创建、超时与取消先于最终状态；工作目录不存在 → `error` / `spawn_error`，`exit_code` 为 NULL；新写入的行不含 `not_found`，读取侧（`check-run.v1`、`status.v1`、推导）照常接受这个值 | 集成 | FR-EV-01 | M2 |
| MVP-EV-05 | 检查输出中有 canary 字符串 → 不带 `--json` 时，调用者的 stdout 中能看到它；数据库、WAL、日志和备份中都找不到 | 集成 | FR-EV-01、FR-PRV-02 | M2 |
| MVP-EV-06 | `check run` 没有接受临时命令的参数；对没有检查的 AC 执行 → 退出码 2 | 集成 | FR-EV-03 | M2 |
| MVP-EV-07 | 直接对任一只追加表执行 UPDATE、DELETE、REPLACE（主键冲突与 UNIQUE 冲突）或 UPSERT → 被触发器中止，原记录不变 | 集成 | FR-EV-05、FR-STO-05 | M2 |
| MVP-EV-08 | 工作区有未跟踪的文件时执行检查 → `tree_state_start = dirty`，`qualifying = false` | 集成 | FR-EV-04 | M2 |
| MVP-EV-09 | 检查在执行过程中写入一个未被忽略的文件 → `error` / `context_changed`；检查在执行过程中执行 `git commit` → `context_changed` | 集成 | FR-EV-04 | M2 |
| MVP-EV-10 | 声明了 `inputs` → 按 data-model §4.11.2 记录每个路径的保存原值、mode、类型与对象 ID（M2-Q2，2026-10-18 批准）；`./a`、`a/`、`a\b` 形式的已保存输入按 V-P04 的方式查找，保存的值与 `check_config_hash` 不变；某个输入路径在 HEAD 上不存在（含大小写不同、子模块内部的路径）→ `error` / `input_missing`，不执行命令；条目存在而叶对象缺失、子模块的 gitlink → 照常记录元数据；读取路径所需的提交或树对象失败、git 不具备安全读取能力（PD-26）→ 退出码 6，不执行检查，`check_runs` 不增加任何行 | 集成 | FR-EV-01 | M2 |
| MVP-EV-11 | 提交的标题中有 canary 字符串 → 提交关联正常，数据库中只有 SHA 和时间 | 集成 | FR-EV-06 | M2 |
| MVP-EV-12 | **`--json` 的输出流**：检查同时向 stdout 和 stderr 打印 canary → `check run --json` 的 stdout 恰好是一份能通过 `check-run.v1` 校验的 JSON，其中没有 canary；两个 canary 都出现在 stderr；数据目录中找不到 canary | 集成 | FR-EV-01 | M2 |
| MVP-EV-13 | **证据等级**：MVP 中每条执行的 `evidence_grade` 都是 `observed`；`check run --isolation required` → 不执行检查，`check_runs` 不增加任何行，退出码 6，`error.details` 中有 `isolation_unavailable` | 集成 | FR-EV-08 | M2 |
| MVP-EV-14 | **提交关联的匹配**（TD-17，data-model §4.12；WP2.3 规范 §5.3）：`Specmetry-Ref` trailer 的键不区分大小写，哪些行是 trailer 由 git 判定；编号按 TD-14 的格式整词匹配，小写、位数不足、`GOAL-n` 都不关联；本项目没有的编号忽略，已退役的照常关联；同一提交对同一实体只有一条关联，在 trailer 中出现时为 `trailer`，否则为 `message`；没有任何关联的提交不入库（WP2.3 规范 §9 的 M1 ~ M11） | 单元、集成 | FR-EV-06 | M2 |
| MVP-EV-15 | **关联的生命周期**（审查 044 N1，data-model §4.12）：前提是项目已有其他需求或 AC（否则按 WP2.3 规范的 D5 不扫描）；扫描时编号还不存在的提交不入库，之后的常规增量扫描不再读取它；已入库提交的关联不重算，后来新增的编号不补写；历史改写后的重扫中首次满足条件的提交，以这次扫描的 `seq` 入库 | 集成 | FR-EV-06、FR-WT-04 | M2 |
| MVP-EV-16 | **扫描的写入**（data-model §4.12；WP2.3 规范 §5.2、§5.5、§5.6）：有新入库的提交时，这些提交共用一个 `seq`，关联与游标在同一事务中写入；没有时不分配 `seq`（`seq` 计数器不变），游标照常更新，`scanned_seq` 不变；写入时游标已被另一个进程改变 → 什么都不写；扫描中任何 git 命令失败 → 退出码 6，什么都不写；**严格读取**（WP2.3 规范 §5.2，审查 050 B1、B2）：正向 tip 不可读（不存在或损坏）→ 退出码 6，游标不前进，下一次扫描仍读取它（断言游标，不只断言退出码）；游标的旧 tip 可靠地不存在 → 按改写重扫该 ref，已入库的去重；旧 tip 存在但不可读 → 退出码 6，不写入；新 ref 的排除项中可靠地不存在的旧 tip 不作排除；排除项在分类之后消失 → 退出码 6；`committed_at` 为 UTC Unix 毫秒；**对象读取不被绕过**（审查 052 B1-R）：已写出 commit-graph、排除项在形成计划之后被删除 → 退出码 6，不写入；`info/grafts` 不能隐藏缺失的父提交；读取中途失败 → 已读到的部分结果丢弃，所有提交、关联、`seq` 与游标都不变；**可靠缺失的判据**（审查 052 B2-R）：旧 tip 所在包的 `.idx` 被截断或静默改坏、`.pack` 缺失或不可读、对象目录或 alternates 不可读 → 无法确认，退出码 6，不当作改写重扫；真正被回收的旧 tip → 按改写重扫；**包与索引双向配对**（审查 055 B2-R2）：旧 tip 所在的 `.pack` 失去了 `.idx`（包括在备用对象目录中），或 `objects/pack` 中有不可识别的文件 → 无法确认，退出码 6，不发布扫描结果、不推进游标；存储检查不写入任何文件；带伴随文件（`.bitmap`、`.rev`、`.keep`）的正常回收 → 按改写重扫<br>实现细化（审查 058 之后，待复审；WP2.3 规范 §14）：扫描的 `log` 带 `--no-show-signature`，仓库配置的签名程序不运行；路径中间的提交缺失时 `is-ancestor` 可能返回 1，按改写读取整段历史后严格读取失败，整次不写入；扫描采用经实际 Git 身份行为探测的显式空 grafts / shallow 覆盖，不使用防护路径（§71 获批） | 集成 | FR-WT-04、FR-EV-06 | M2 |
| MVP-EV-17 | **输入路径的读取机制**（TD-23，WP2.3 规范 §4，已冻结）：在夹具仓库中逐项复现规范中 T1 ~ T23 的结果：普通文件、可执行文件、目录、仓库根、符号链接、gitlink、空格与特殊字符、通配字符的字面匹配、大小写、未规范化的路径；路径不存在 → 没有记录；读取路径所需的树对象缺失 → 退出码 6；叶对象缺失 → 照常记录 | 集成 | FR-EV-01 | M2 |
| MVP-EV-18 | **工作区状态的读取**（data-model §1，产品负责人 2026-10-18 决定 D1；WP2.3 规范 §4.3，已冻结）：超级项目与已初始化的子模块都为 clean / process 过滤器配置了写标记的夹具程序，只改修改时间 → 读取之后没有标记（对照组直接运行 `git status` 时有标记，证明夹具会触发）；子模块检出的提交与 gitlink 不同、或子模块中有修改 → `dirty`，仓库配置 `diff.ignoreSubmodules=all` 时也一样；**不报告错误的 `clean`**（审查 052 B3-R1）：提交之后才配置本地过滤器驱动（定义下为 `dirty`）与先配置驱动（定义下为 `clean`）两个方向，以及 racy 的受过滤条目 → 都不报告 `clean`，结果为 `unknown`（产品负责人 2026-10-19 决定 P1），不是 `clean`；受过滤文件的 stat 与索引一致、不 racy → 与定义一致；普通文件只改修改时间 → `clean`；未跟踪、已暂存、删除的文件 → `dirty`；被忽略的文件与 `.specmetry/` 不计入；**原仓库不变**（审查 052 B3-R2）：读取前后原 `.git` 下所有文件的内容与修改时间都不变，不只比较索引与配置；**稀疏门禁前置**（审查 055 B3-R2；P2）：带稀疏目录条目的索引 → `unknown`，在任何可能展开索引的命令之前判定，门禁之后不运行任何 git 命令，超级项目与每个子模块都如此，原 `.git` 下所有文件的内容与修改时间都不变（不只断言最终值）；门禁不能解析索引 → 退出码 6，同样不运行后续命令；超级项目的 `diff-files` 不进入子模块；没有稀疏目录条目的稀疏检出照常读取；gitfile 子模块按解析出的实际 git 目录读取，没有提交的仓库照常读取；任何一步失败 → 不报告 `clean`<br>实现细化（审查 058 之后，待复审；WP2.3 规范 §14）：被定义的 pathspec 排除的 `.specmetry/` 不经任何读取（索引列举、过滤器与 racy 判据、gitlink）影响结果，名称相近的 `.specmetry-not/` 照常计入；已暂存、从索引移除、目录被删除或换成文件的 gitlink 都为 `dirty`；截断的 `link` 位图是环境错误，不 panic；私有目录在受保护的数据目录中，被替换时读取失败且不删除替代物；（审查 059 之后，待复审）删除的重试之间私有目录被替换、使用中 `tmp/` 被重建并放入同名的所有者文件与锁、`tmp/` 或数据目录被换成链接（包括指向原处）→ 读取失败，替代物与见证文件不变，以后的遗留清理也不删除它们；同一目录的临时删除失败照常重试成功 | 集成 | FR-EV-04 | M2 |
| MVP-EV-19 | **解释器与命令传递**（2026-10-25 产品决定 D1、D2；WP2.5 规范 §5）：每个平台上每种可用的解释器，用 WP2.5 规范 §5.5 的字符串集回显，结果与输入逐字节一致；省略 `shell` 时所有平台都用 `sh`，Windows 上从不回退到 `cmd`；Windows 的 `sh` / `bash` 只取 `gitx` 所用 Git 对应的 Git for Windows 的 `usr\bin` 入口，PATH 中只有 WSL 的 `bash.exe` 或其他 `sh` 时不使用它们；Windows `sh` / `bash` 检查看到的 PATH 以 `<Git>\usr\bin` 开头，Specmetry 自己的进程、Git 读取以及 `cmd`、`pwsh` 检查的环境不变；位于 worktree 之内的解释器（经 PATH 的绝对路径、相对路径项、符号链接或联接到达的）不被执行 | 集成 | FR-EV-01、FR-EV-03 | M2 |
| MVP-EV-20 | **解释器不可用**（D1）：所选 AC 所需的任一解释器不可用（Windows 上没有 Git for Windows 的 `sh`；Unix 上的 `cmd`、`powershell`）→ 整批不执行，没有新行，没有启动任何检查进程，退出码 6，`details` 给出 `shell_unavailable` 与受影响的 AC 编号，不回显路径 | 集成 | FR-EV-01 | M2 |
| MVP-EV-21 | **命令中的 CR / LF**（D3、D4）：`shell = cmd` 的命令含 CR 或 LF、`sh` / `bash` / 省略 `shell` 的命令含 CR（包括引号之内）→ `propose`、`review`（含 `accepted_with_edits`）与人工编辑都拒绝（退出码 2，不写入）；`sh` 命令中正常的 LF 多行被接受并执行；不含 `check` 的 `update` 不重新校验已保存的配置；数据库中已有的不符合的配置：`baseline show` 等照常显示，命令、哈希与修订不变，`check run` 在执行前拒绝（退出码 2，整批不执行，没有新行），经人工 `material` 修订之后可以执行 | 集成 | FR-BL-12、FR-EV-03 | M2 |
| MVP-EV-22 | **结束上下文读取失败**（D6）：进程树已确认结束、注入结束上下文的读取失败 → 尝试记录 `error` / `context_changed`，已读到的值保留，读不到的 `head_end` 为 NULL、`tree_state_end` 为 `unknown`，`qualifying = false`；之后的 AC 不执行；退出码 6；输出说明结束上下文读取失败，不说成“已证明发生变化” | 集成 | FR-EV-04 | M2 |
| MVP-EV-23 | **取消**（D8）：向 CLI 进程发送 SIGINT / SIGTERM 或控制台事件 → 进程组 / Job 按收尾协议终止并确认；写事务开始之前取消 → 当前项没有新行，之后的 AC 不执行，之前已提交的执行保留，退出码 6，`message` 写明已取消；写事务开始之后取消 → 当前项按实际的提交结果；取消与进程树没有确认结束同时发生 → 按 MVP-EV-28；在申请排他租约的排队中取消（进程内另有共享持有者没有释放）→ 立即退出排队，不等待那个子进程，当前项不记录，退出码 6，其他等待者不被这次已取消的请求阻塞（Review 091 B1）；输出不说明“检查没有运行”或“可以安全重跑” | 集成（进程级） | FR-EV-01、FR-EV-02 | M2 |
| MVP-EV-24 | **批次汇总**（D9）：每条执行的 `head_start` 都等于 `S` → `is_target_snapshot = true`；两次执行之间切换 HEAD（混合 `head_start`）→ false，但位于 `S`、合格、修订与配置为当前值的那一条照常参与该 AC 的 `V`（推导到 `ready_for_review`）；空批次比较命令开始时的 HEAD；没有 `active` 目标 → false；推导不读取这个字段 | 集成 | FR-EV-01、FR-EV-04 | M2 |
| MVP-EV-25 | **主进程退出之后的后代**（D11）：检查留下一个后台进程后以 0 退出 → 后台进程被终止并确认结束，之后才读取结束上下文；后台进程在被终止之前写入 worktree 的改动反映在结束上下文中（`context_changed`）；主进程的退出码照常记录 | 集成 | FR-EV-02、FR-EV-04 | M2 |
| MVP-EV-26 | **配置在命令开始时固定**（D13）：`check run --all` 的第一项执行期间，人以 `material` 方式修订后面的一个 AC → 后面那一项仍按命令开始时的配置执行，记录旧的 `criterion_revision` 与 `check_config_hash`；它对新修订为 `V = none`；推导规则不变 | 集成 | FR-EV-01 | M2 |
| MVP-EV-27 | **空批次**（D18）：`--all` 选不到可执行的 AC → 不执行任何检查，退出码 0，`runs` 为空；`W` 已登记 → 没有写入，输出已有的 `worktree_id`；`W` 未登记 → 只写一条 worktree 的登记并分配它的 `seq`，输出新的 `worktree_id`；注入登记失败 → 退出码 6，没有写入；注入提交结果不确定 → 退出码 6，说明登记可能已经发生 | 集成 | FR-EV-01、FR-WT-02 | M2 |
| MVP-EV-28 | **进程树没有在 `T_kill` 内确认结束**（D19；测试接缝模拟）→ 不读取结束上下文；尝试记录 `error` / `context_changed`，`head_end` 为 NULL，`tree_state_end = unknown`，`qualifying = false`；`finished_at` 为尝试记录的时刻；分别构造三种情形：主进程的退出已被可靠观察到（`duration_ms` 为从协议启动到可靠观察到退出的运行时间，`exit_code` 为可靠读到的值，Unix 信号为 NULL；在进入终止阶段之后才取得的同样保留，Review 093 B2-R2）、主进程仍在运行、主进程早已退出但退出的观察失败——后两种 `duration_ms` 都是从协议启动到进入终止阶段（决定终止）的定义区间，`exit_code` 为 NULL，输出不称它为运行时间或它的界限（Review 089 B3）；Windows 恢复异常（测试接缝）而终止未确认 → 同样尝试记录，`started_at` 与 `duration_ms` 按恢复异常的定义区间（从观察到异常到进入 S3），可靠取得了主进程的退出状态时保留它的退出码（不论何时取得、退出的因果是否已知），没有取得时 `exit_code` 为 NULL，输出不称已经执行，也不从退出码推断退出的原因（Review 092 B2-R1、Review 093 B2-R2）；之后的 AC 不执行；退出码 6；输出说明进程树没有确认结束、结束上下文没有读取，不报告已结束；注入写入失败 → 这一次没有记录、仍报告可能残留；注入提交结果不确定 → 报告记录不确定 | 集成 | FR-EV-02、FR-EV-04 | M2 |
| MVP-EV-29 | **输出的支持契约**（D20、C-4）：进程内以不受支持的 Writer 调用 → 启动任何检查之前拒绝，退出码 1，没有新行；受控的内存目的端上，注入复制错误（O2）或不结束的复制任务（O3）→ 这一次照常分类并记录（可以合格），之后的 AC 不执行，退出码 6，说明转发不完整；封闭之后没有迟到的检查输出进入目的端，最后的文档完整、在最后；O3 之后同一进程中新的 `check run` 调用 → 不打开数据库、不运行 git、不启动进程，退出码 6；遗留的复制任务与停止任务之后全部结束、资源清单清空，新的调用**仍然**被拒绝；新的进程照常执行（Review 089 B1）；停止操作永不返回（测试接缝）时，J 阶段仍按截止到期、入口封闭，停止报告为挂起并移交给资源清单（Review 089 B2）；**停止任务在 J 结束时没有确认结束**（D21，测试接缝）而复制为 O1、O2 或 O3 → 当前项按其他状态记录（O1 时仍为完整，可以合格），之后的 AC 不执行，退出码 6，同一进程中新的调用被拒绝，任务之后结束仍被拒绝；停止已完成而关闭出错 → 报告“释放未确认”，关闭出错本身不禁用宿主；同时为 O3 时仍按 C-4 禁用（Review 091 N1） | 集成（进程内） | FR-EV-01 | M2 |
| MVP-EV-30 | **组合失败与退出码**（D8；WP2.5 规范 §8.6）：A、B、C、D、E 表的每一行都有用例；持久化的结果叠加在之前的状态之上、不覆盖它（`input_missing` 之后的写入失败报告检查没有运行；进程树没有确认结束之后的写入失败仍报告可能残留）；提交之前的失败与提交阶段的错误分别注入，后者报告记录不确定；退出码按 1 > 6 > 10 > 0，执行项校验失败的 1 不被 6 覆盖；之前已有 `fail` 而随后停止时退出码为 6，`details` 列出已记录的执行（至多 200 项）；文本与 JSON 两种模式都覆盖 | 集成 | FR-EV-01、FR-EV-05 | M2 |

### 2.9 验收（ACC）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-ACC-01 | AC 在 `S` 上为 `ready_for_review`（观测级证据），还没有任何决定 → 在 TTY 中执行 `accept`（`normal`，`seen_decision_id = null`，确认 `observed_evidence`）→ 写入的决定包含 `snapshot`、`evidence_id`、`evidence_grade = observed`、`check_config_hash`、`inputs_json` 和风险确认；AC 变为 `accepted`（原因码 `direct`，带 `observed_evidence` 标记）。在 M5 中通过 API 再验证一次。 | 集成 | FR-ACC-01、03 | M2 / M5 |
| MVP-ACC-02 | 非 TTY 执行 `accept` → 退出码 3；在 TTY 中执行，但存在已知的 Agent 环境变量 → 退出码 3 | 集成 | FR-ACC-02 | M2 |
| MVP-ACC-03 | **必须拒绝**（Bash 和 PowerShell 两种 payload）：`specmetry accept AC-001-1`、`cd app && specmetry waive AC-002-1 -m x`、`specmetry scope confirm`、`specmetry source ack docs/a.md`、`specmetry propose --apply --file x.json`、`specmetry question answer Q-001`、`specmetry question dismiss Q-002`、`specmetry project set excerpt-storage full`、`bash -c "specmetry review"` → 输出拒绝决定，并记录 `guard_denied`。**必须放行**：`specmetry status --json`、`specmetry dashboard --page review`、`echo "specmetry accept"`、正文中含有 accept / review / setup 字样的 `specmetry propose --file - <<EOF … EOF`、`specmetry claim --file -` | 单元 + 集成 | FR-ACC-02 | M3 |
| MVP-ACC-04 | 在 `V = none` 时执行 `normal` 验收 → 退出码 2（提示使用例外）；在 `V = fail` 时作出任何 `accepted` → 退出码 2 | 集成 | FR-ACC-03、06 | M2 |
| MVP-ACC-05 | 例外验收没有理由 → 退出码 2；有理由时 → AC 为 `accepted`（`exception`），并带 `EXCEPTION_ACCEPTANCE` 警告；`status --json` 中该 AC 的 `decision` 显示 `mode = exception`、理由、快照和 `evidence_grade = none`，人类可读输出同样显示这四项。之后在 `S` 上记录合格的 `pass` → `snapshot_run` 显示这条执行，AC 仍为 `accepted`（`exception`），交付判定的证据等级不因此改变；记录 `fail` → `failing`（`regressed`）（PD-17） | 集成 | FR-ACC-05 | M2 |
| MVP-ACC-06 | 手动 AC 在 `S` 上被验收 → `accepted`（`manual`） | 集成 | FR-ACC-04 | M2 |
| MVP-ACC-07 | 拒收或豁免没有理由 → CLI 退出码 2（M2），API 返回 400（M5）；实现 `status.v1` 时补充 schema 条件约束与反例：拒收决定的 `snapshot` 不能为 null（只有豁免可以没有快照） | 集成 | FR-ACC-07 | M2 / M5 |
| MVP-ACC-08 | 自动化 AC 在 `S` 上 `V = pass` → 人依次在同一快照上作出接受、拒收、接受，每次都携带上一条决定的 ID 作为 `seen_decision_id`（第二次接受是人在重新读取后明确改判）→ 三条都保留，当前决定是 `seq` 最大的那一条，AC 为 `accepted` | 集成 | FR-ACC-08、FR-ACC-03 | M2 |
| MVP-ACC-09 | 沿用：在 A 上 `normal` 验收 → 目标前进到 B。**单元**：在 B 上有一条合成的隔离级合格通过 → `accepted`（`carried_forward`），没有声明 `inputs` 时同样适用；如果是观测级合格通过 → `ready_for_review`（`observed_evidence`）。**集成（MVP）**：在 B 上执行 `check run` → `ready_for_review`（`observed_evidence`），没有人参与时**绝不**变为 `accepted`。隔离级的集成验证在 M7 进行。 | 单元 + 集成 | FR-ACC-09 | M2 |
| MVP-ACC-10 | **风险确认**：观测级证据上的 `normal` 验收缺少 `observed_evidence` → 退出码 2（API 为 400）；多确认一项实际不存在的风险 → 退出码 2；`runs(S)` 结果不一致却没有确认 `inconsistent_results` → 退出码 2；例外验收没有确认 `no_automated_evidence` → 退出码 2。以上情况都不写入。 | 集成 | FR-ACC-03、FR-ACC-05 | M2 / M5 |
| MVP-ACC-11 | **并发决定**（用两个独立的数据库连接或进程，写事务使用 `BEGIN IMMEDIATE`）：两个会话都读到 `current_decision_id = D1` → 会话乙拒收（写入 D2）→ 会话甲用 `seen_decision_id = D1` 提交接受 → 退出码 5（API 为 409），不写入，`error.current` 中给出 D2；CLI 在伪终端中收到退出码 5 后不再发出任何请求；会话甲重新读取后，用 `seen_decision_id = D2` 明确接受 → 成功 | 集成 | FR-ACC-01 | M2 / M5 |

### 2.10 推导与交付判定（ST）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-ST-01 | AC 决策表的 11 条规则，每条至少命中一次；每对**可能同时成立**的相邻规则，至少有一个优先级冲突用例；第 7、10、11 条的原因码优先级各有用例 | 单元 | FR-ST-02 | M2 |
| MVP-ST-02 | 需求聚合表的 9 条规则，每条都有用例 | 单元 | FR-ST-02 | M2 |
| MVP-ST-03 | 目标交付表的 6 条规则，每条都有用例；每个状态都同时输出全部阻塞差距 | 单元 | FR-ST-04、05 | M2 |
| MVP-ST-04 | 进度表的 8 种情况，每种都有用例 | 单元 | FR-ST-03 | M2 |
| MVP-ST-05 | 性质测试：生成随机的合法事实序列（data-model §6.14），把行的物理顺序打乱 1000 次（包括共享同一个 `seq` 的记录之间的顺序）→ 推导结果完全相同 | 单元 | FR-ST-08、NFR-11 | M2 |
| MVP-ST-06 | OpenAPI 规格和 CLI 命令树中，没有设置状态、执行检查（Dashboard）、删除事实的操作 | 审查 | FR-ST-01 | M5 |
| MVP-ST-07 | 对每个项目夹具执行 `gaps --json` → 与 `expected-gaps.json` 一致 | 集成 | FR-ST-05 | M2 |
| MVP-ST-08 | `fx-scale` 上执行 `status --json` → 少于 1 s | 基准 | NFR-04 | M2 |
| MVP-ST-09 | `status` 的人类可读输出与 golden 一致：包括目标、快照、交付状态、阻塞差距、工作区视图提示 | 集成 | FR-ST-06 | M2 |
| MVP-ST-10 | 所有夹具上，`init --json`、`status --json`、`gaps --json`、`check run --json` 的输出分别通过 `init.v1`、`status.v1`、`gaps.v1`、`check-run.v1` 的校验；所有错误输出都通过 `error.v1` 的校验 | 集成 | FR-ST-06 | M2 |
| MVP-ST-11 | `status --diff-since <seq>` → 两次推导共用本次命令一次取得的当前值输入（`S`、观测哈希、项目设置），只改变事实的截止序号；只列出 `seq` 之后新增的事实带来的状态变化；输出带 `diff_since_seq`，`project.*` 是两次推导共用的设置。在两次推导之间注入对设置的修改 → 结果与输出都不受影响。只修改设置、没有新事实（例如 `init --skip-sources`、`init --stage docs_only`）→ `as_of_seq` 不变，`status --diff-since <as_of_seq>` 的 `changes_since` 为空：不虚构事实序号或历史事实的变化（data-model §6.13.1） | 集成 | FR-ST-06 | M2 |
| MVP-ST-12 | [data-model.md §6.9](data-model.md#69-保守规则无法判断时) 中的每一行保守规则都有用例 | 单元 + 集成 | FR-ST-09 | M2 |
| MVP-ST-13 | **差距与交付状态的一致性**（性质测试）：随机生成事实序列，每次都满足"交付状态为 `ready` ⇔ 没有阻塞差距"；每个 AC 至多有 1 个状态差距；有范围时，范围外的条目没有差距；`waived` 和 `accepted` 的 AC 没有状态差距；`SOURCE_REMOVED` 等标记类警告从不阻塞 | 单元 | FR-ST-05 | M2 |
| MVP-ST-14 | **推导纯度与重放**（data-model §6.13.1）：相同的完整输入（截止 `as_of_seq` 的事实、`S`、观测哈希、项目设置与参数）→ 推导结果相同；修改磁盘文件但不重新观测 → 推导结果不变。记录一次输出之后修改当前设置（例如在只完成初始化的项目上执行 `init --skip-sources`，即 `sources_skipped` 由 false 变为 true：`init_progress` 由 `initialized` 变为 `sources_registered`，`as_of_seq` 不变）→ 用原输出记录的设置与其他当前值输入，以及同一数据库中截止原 `as_of_seq` 的事实重放 → 仍得到原结果。缺少必要的重放输入（输出中没有项目设置；来源集合中的某个来源没有观测哈希；数据库中缺少截止 `as_of_seq` 的事实）→ 拒绝重放并报告缺少的项，不用当前值补齐；没有观测与观测到文件缺失（NULL）区分开。只凭 `as_of_seq`、`snapshot` 与 `baseline_version` 不构成完整输入；输出不是事实的备份 | 单元 + 集成 | FR-ST-08、NFR-11 | M2 |
| MVP-ST-15 | **设置上下文**：`status --json`、`gaps --json` 中的 `project.stage`、`project.sources_skipped`、`project.inventory_skipped`，等于这次推导实际使用的值；设置与事实在同一个读事务中读取，在读取之后、输出之前注入对设置的修改 → 输出与推导结果都仍按读到的值。修改设置之前与之后各执行一次（其间没有其他写入）→ 两次输出的 `project.*` 各自反映当时的值，`as_of_seq` 相同。只输出这三项设置，`settings_json` 的其他键（例如 `excerpt_storage`）不出现在输出中（data-model §6.13、§6.13.1） | 集成 | FR-ST-08、NFR-11 | M2 |

### 2.11 采集（COL）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-COL-01 | 连续调用 200 次：在项目内 p95 ≤ 100 ms，在非项目目录 p95 ≤ 30 ms；三平台分别测量 | 基准 | NFR-02 | M3 |
| MVP-COL-02 | stdin 格式错误或为空、事件未知、spool 不可写、`SPECMETRY_HOME` 不存在 → 都以 0 退出，stdout 为空 | 集成 | FR-COL-02 | M3 |
| MVP-COL-03 | 在非项目目录中触发 → 不生成 spool 文件 | 集成 | FR-COL-02 | M3 |
| MVP-COL-04 | 同一批 spool 入库两次；在"已提交、未删除文件"时崩溃后重新入库 → 没有重复 | 集成 | FR-COL-03 | M3 |
| MVP-COL-05 | 重放 `fx-hook-payloads` → 每个 spool 文件都能通过 `spool-event.v1` 校验；入库后，数据目录中找不到任何 canary | 集成 | FR-COL-04 | M3 |
| MVP-COL-06 | 把 `transcript_path` 设为一个 FIFO（Unix）→ Hook 在预算内正常退出 | 集成 | FR-COL-04 | M3 |
| MVP-COL-07 | 用 `fx-git-heads` 中的各种 HEAD 形态触发 SessionStart → 能解析的给出正确的 SHA（`hook_file_read`），其余为 `unknown`；入库时的 HEAD 永远不会被写入 `head_start` | 集成 | FR-COL-08 | M3 |
| MVP-COL-08 | PostToolUseFailure → `tool_use`，`outcome = failure` | 集成 | FR-COL-01 | M3 |
| MVP-COL-09 | 输入超过 1 MiB → 计一次 `oversize`；人为制造写入延迟超过 200 ms → 计一次 `timeout`；`hook_drops` 中能看到这些计数 | 集成 | FR-COL-02 | M3 |
| MVP-COL-10 | 在 feature worktree 中触发 Hook → 事件和会话归属于该 worktree | 集成 | FR-WT-02 | M3 |
| MVP-COL-11 | `status --session current` 的比较起点：有上一批次时为 `previous_batch`；只有会话起点时为 `session_start`；两者都没有时为 `none`；`changed_paths` 包含未跟踪的文件 | 集成 | FR-COL-05 | M3 |
| MVP-COL-12 | 不安装任何 Hook，只用 CLI 完成 E2E-2 → 结果与 oracle 一致 | E2E | FR-COL-07 | M2 |
| MVP-COL-13 | spool 中有损坏的文件 → 移入 `dead/`，其余正常入库 | 集成 | FR-COL-02 | M3 |
| MVP-COL-14 | Dashboard 运行时触发 Hook → 5 秒内出现在概览页的"最近会话"中 | 集成 | FR-UI-10 | M5 |
| MVP-COL-15 | `doctor --json` 报告 Claude Code 版本，以及是否在经过测试的范围内 | 集成 | FR-COL-09 | M3 |

### 2.12 Agent Skill（SK）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-SK-01 | `setup` 安装的 Skill 版本与二进制一致；手动改动 Skill 后，`doctor` 报告不一致 | 集成 | FR-SK-01 | M4 |
| MVP-SK-02 | 静态检查：Skill 中的每个 `specmetry …` 调用都能被 CLI 命令树解析 | 单元 | FR-SK-03 | M4 |
| MVP-SK-03 | 静态检查：禁止事项覆盖了拦截清单中的全部子命令，并且明确禁止通过 HTTP、浏览器自动化、直接写文件代做人工操作 | 单元 | FR-SK-04 | M4 |
| MVP-SK-04 | Skill 包中的 schema 与 `schemas/` 逐字节一致 | 单元 | FR-SK-03 | M4 |
| MVP-SK-05 | 在三个项目夹具上执行 `/specmetry init`，按 PD-15 评分：提案在 2 次尝试内通过校验；召回率 ≥ 80%；`not_found` ≤ 10%；提交后停下来等待审核；没有人类专属命令被真正执行 | 评测 | FR-SK-02、05 | M4 |
| MVP-SK-06 | 在 `fx-in-dev` 上确认范围后执行 `/specmetry audit` → `audit_scope = all`，覆盖率 100%（允许 `unknown`）；检查在当前 worktree 中执行 | 评测 | FR-SK-02、06 | M4 |
| MVP-SK-07 | 评测中人为让版本过期 → Agent 收到退出码 5 后重新读取、重新判断，然后再提交，而不是原样重试 | 评测 | FR-SK-03 | M4 |
| MVP-SK-08 | `[P1]` audit 为没有检查的自动化 AC 提议检查命令（提案形式，带理由） | 评测 | FR-INV-05 | M4 |

### 2.13 Dashboard（UI）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-UI-01 | 只监听 `127.0.0.1`；通过本机的非回环地址连接 → 被拒绝 | 集成 | FR-UI-01 | M5 |
| MVP-UI-02 | 没有令牌或令牌错误 → 401；Host 错误 → 403；写请求的 Origin 来自其他源 → 403；响应带 CSP | 集成 | FR-UI-01、FR-PRV-05 | M5 |
| MVP-UI-03 | 遍历所有页面 → 所有请求都是同源的 | UI | FR-PRV-05 | M5 |
| MVP-UI-04 | `fx-scale` 的概览首屏 < 2 s | 基准 | NFR-04 | M5 |
| MVP-UI-05 | 对每个夹具 → 概览页显示的目标、快照、交付状态、阻塞差距和指标，与 `status --json` 一致 | UI | FR-UI-02 | M5 |
| MVP-UI-06 | 基准树显示进度和交付状态；展开引用时，显示"校验只证明片段存在"的说明 | UI | FR-UI-03 | M5 |
| MVP-UI-07 | 审核队列：对 `material` 的需求更新，要求为每个 AC 选择处理方式；可以附带范围确认，一次提交 | UI | FR-UI-04 | M5 |
| MVP-UI-08 | 需求详情：快照上的执行（标注证据等级），以及其他上下文中的执行（标注"不适用于交付"）；显示当前决定；只显示当前允许的决定类型，以及必须确认的风险说明（文字与 data-model §7.3 一致），未逐项确认时不能提交；例外、拒收和豁免必须填写理由；在页面打开期间让 AC 被修订，或者在另一个页面中拒收，然后提交决定 → 收到 409，页面展示最新状态并重新加载，不写入，不自动重试 | UI | FR-UI-05、FR-UI-13 | M5 |
| MVP-UI-09 | 差距页的分组和排序与 `gaps --json` 一致 | UI | FR-UI-06 | M5 |
| MVP-UI-10 | 健康页显示 Hook 与 Skill 的安装状态、spool 积压、drop 计数、worktree 列表、数据目录权限检查结果 | UI | FR-UI-09 | M5 |
| MVP-UI-11 | 页面打开时在 CLI 中写入事实 → 页面 5 秒内更新 | UI | FR-UI-10 | M5 |
| MVP-UI-12 | 显示反推、`ref_not_found`、例外验收、结果不一致、自述无证据等标识 | UI | FR-UI-11 | M5 |
| MVP-UI-13 | 两个项目都显示在列表中，可以切换 | UI | FR-UI-12 | M5 |
| MVP-UI-14 | Agent 通道执行 `dashboard` → stdout 和 stderr 中都没有令牌；`run/dashboard.json` 的权限符合 §11.1 | 集成 | FR-UI-01 | M5 |
| MVP-UI-15 | 在四种当前版本的浏览器中走一遍主流程 | 人工 | mvp.md §4 | M6 |
| MVP-UI-16 | `[P1]` 批量验收 5 个 `ready_for_review` 的 AC，其中一个在提交前被另一个页面拒收 → 其余 4 个成功，那 1 个返回 409，并且不会自动重试 | UI | FR-UI-14 | M5 |

### 2.14 隐私与存储（PRV / STO）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-PRV-01 | 在禁止出站网络的环境中，E2E-1 ~ E2E-8 全部通过；代码审查确认，除了监听回环地址，没有网络调用 | E2E + 审查 | FR-PRV-01、FR-INIT-07 | M6 |
| MVP-PRV-02 | **C0 canary 综合测试**：在检查输出、Hook 的命令文本和工具输出、环境变量的值、文件内容、提示词和 transcript、提交标题中埋入 canary，跑完全部 E2E → 数据库、WAL、spool、日志、备份中都找不到 | E2E | FR-PRV-02 | M6 |
| MVP-PRV-03 | 有项目 A 和 B → 在 TTY 中对 A 执行 `purge` → A 的所有行被删除；数据库文件和 WAL 中找不到 A 的 canary；B 不受影响；默认删除旧备份；`tombstones.json` 中有 A；非 TTY 执行 → 退出码 3 | 集成 | FR-PRV-03、FR-STO-03 | M2 |
| MVP-PRV-04 | 伪造时间，使事件超过 30 天 → 入库时被清除；其他事实不受影响 | 集成 | FR-PRV-04 | M3 |
| MVP-PRV-05 | Unix 上，数据目录为 `0700`，文件为 `0600`；已有的数据目录或其中任何文件（包括数据库、WAL、SHM）权限过宽、属主不是当前用户、是符号链接，或（macOS）带有允许访问的 ACL 时，在写入任何数据之前拒绝（退出码 6），目录内容不变；从根目录到数据目录的任何一级目录属主不是当前用户或 root，或能被其他用户写入（不带 sticky 位，或带 sticky 位但下一级条目不属于当前用户或 root），在创建或写入任何内容之前拒绝（退出码 6）；用真实的跨 UID 替换攻击验证（Linux，测试需要 root 以切换用户） | 集成 | FR-PRV-06 | M0 |
| MVP-PRV-06 | Windows 上，数据目录的 DACL 是受保护的，只授予 TD-22 规定的对象；已有目录或文件不满足（额外的授权对象、DACL 不受保护或不能被新文件继承、属主不符、联接或符号链接）时，在写入任何数据之前拒绝（退出码 6），目录内容不变；路径上的目录是联接或符号链接、属主不是当前用户 / SYSTEM / Administrators / TrustedInstaller，或允许其他用户 DELETE、FILE_DELETE_CHILD、WRITE_DAC、WRITE_OWNER 时，在创建或写入任何内容之前拒绝（退出码 6）；`doctor` 能发现被放宽的权限 | 集成 | FR-PRV-06 | M0 |
| MVP-PRV-07 | [architecture.md §10.8](architecture.md#108-密钥识别规则集td-21) 的每条内置规则至少 2 个命中和 2 个不命中样本（运行时拼接，仓库中不保存形似密钥的字面量）；每个不命中样本不命中任何规则；环境变量引用不命中；命中时退出码 2，`error.details` 只给出 JSON Pointer 和 `secret_detected: <规则编号>`，输出和日志中找不到命中的文本；`config.toml` 追加的规则生效，无效的追加规则使配置加载失败（退出码 6）。边界反例（codex-review-009）：① 同一字段既有环境变量引用、又有一段可识别的密钥 → 拒绝；② 同一字段有多个候选匹配，前一个可以豁免、后一个不能 → 拒绝；③ 追加规则的 pattern 含有 canary 且编译失败 → 退出码 6，错误输出中找不到 canary。WP1.3 验证扫描模块与配置加载；CLI 写入被拒绝且不写入任何数据，在 WP1.5 起随各写入命令验证 | 单元 + 集成 | FR-PRV-07 | M1 |
| MVP-PRV-08 | 新项目的 `excerpt_storage` 默认为 `hash_only`（PD-24）→ 提交含片段的提案后，`source_refs.excerpt` 为 NULL，`excerpt_sha256` 和校验结果照常写入，`proposal_items` 中也只有片段哈希；对数据目录做字节搜索，找不到片段原文 | 集成 | FR-PRV-08 | M1 |
| MVP-PRV-09 | 对所有人类可读输出的 golden 文件做静态检查：`evidence_grade = observed` 时，不出现 snapshot-verified / snapshot verified / "快照已验证"等措辞；`evidence_grade = none` 时，不出现"已通过自动化验证" / verified by automated checks 等措辞，并说明没有自动化执行证据（CLI 在 M2 检查，Dashboard 在 M5 检查） | 单元 + UI | FR-PRV-09 | M2 / M5 |
| MVP-PRV-10 | `project set excerpt-storage full`：非 TTY 或存在已知的 Agent 环境变量 → 退出码 3；在 TTY 中确认后生效，之后写入的引用保存原文，之前写入的不变；`source list --refs --json` 中分别为 `stored` 和 `not_stored` | 集成 | FR-PRV-08 | M2 |
| MVP-STO-01 | 两个进程同时触发迁移 → 一个执行，另一个等待或返回退出码 6；迁移前的备份可以打开，内容一致；锁由操作系统持有，持有锁的进程崩溃后不会阻塞其他进程 | 集成 | FR-STO-01 | M0 |
| MVP-STO-02 | 旧程序打开更新版本的 schema（包括恢复的 `VACUUM INTO` 备份、崩溃遗留的 WAL），或不满足 `min_binary_version`（包括无法解析的值）→ 退出码 6。Specmetry 不修改 schema、业务数据、事实、`seq` 或任何已提交数据，不改写或删除已有的数据库、WAL、SHM 文件，不执行 checkpoint，不绕过版本检查；无法确定版本时一律拒绝，不判为兼容。Specmetry 自身**只允许两种文件副作用**（产品负责人批准，2026-10-13）：① 数据目录中还没有协调文件时，创建空的、权限符合 architecture §11.1、拒绝后不被持有的 `.migrate.lock`；② 仅当另一个 SQLite 进程在版本检查期间并发关闭数据库、删除了 `-wal` / `-shm` 时，原生只读连接新建的空 `-wal` / `-shm`，权限符合同一要求。外部进程自己造成的文件变化不算 Specmetry 的副作用；这两项例外不扩大为一般性的文件修改许可，也不放宽版本保护或数据完整性约束。没有外部竞争操作、且协调文件已存在时，拒绝不改变任何文件。已打开的连接在其他进程升级 schema 或提高 `min_binary_version` 之后写入 → 退出码 6，写入回调不执行，事实和 `seq` 不变；打开过程与另一进程的升级并发时，升级等待打开完成，或旧程序在上述限度内拒绝，不允许在恢复或合并新数据库之后才拒绝；旧进程此后关闭最后一个连接时，升级后的数据库保持完整 | 集成 | FR-STO-02 | M0 |
| MVP-STO-03 | 合成一个会在中途失败的迁移 → 整体回滚，数据库不变 | 集成 | FR-STO-01 | M0 |
| MVP-STO-04 | 多个进程并发写入 → `seq` 严格递增、没有重复 | 集成 | FR-STO-05 | M0 |
| MVP-STO-05 | `[P1]` `backup create` 和 `backup restore`；恢复后墓碑生效 | 集成 | FR-STO-04 | M2 |
| MVP-STO-06 | 对带只追加触发器的表执行 UPDATE、DELETE、`INSERT OR REPLACE` / `REPLACE INTO`（主键冲突、UNIQUE 冲突）、建表时声明的 `ON CONFLICT REPLACE`、UPSERT → 全部被中止，原记录完整保留；连接池新建的每个连接都开启 `recursive_triggers`，缺失时写事务拒绝执行；生产代码和迁移中不出现 REPLACE 冲突处理（静态检查） | 集成 | FR-STO-05 | M0 |
| MVP-STO-07 | `doctor` 是非破坏性诊断：不执行 SQL 写入、迁移、恢复或 checkpoint；用 SQLite 只读连接读取，只在实测确认不修改文件的状态下进行（`-wal` 与 `-shm` 都存在；没有辅助文件的回滚日志模式数据库）；无法保证一致读取时（缺少 `-wal` 与 `-shm`、只有其一、存在回滚日志、另一程序持有写锁、读取期间文件变化）报告版本未知，不报告版本或正常；进行中的事务（WAL 模式或回滚日志模式）未提交的数据永不被报告；没有外部竞争操作时，检查前后数据库、WAL、SHM 的内容和目录项完全一致。**受限的已知行为**（与 MVP-STO-02 的例外 ② 相同，产品负责人批准）：另一个 SQLite 进程在检查期间并发关闭数据库并删除 `-wal`、`-shm` 时，原生只读连接可能新建空的 `-wal` / `-shm`（权限符合安全契约）；此时必须报告版本未知，主数据库、业务数据和已有文件内容不变。命令帮助和文档说明 `doctor` 不提供取证级的绝对只读保证。打开数据库前的版本检查使用同一读取方式，同样不报告未提交数据，它的文件副作用见 MVP-STO-02 | 集成 | FR-STO-02、NFR-09 | M0 |

### 2.15 平台（PLT）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-PLT-01 | 三平台 CI 全部通过 | 集成 | NFR-05 | M0 |
| MVP-PLT-02 | 项目根目录、来源路径、`SPECMETRY_HOME` 都含空格和中文 → 全部功能正常 | 集成 | NFR-10 | M6 |
| MVP-PLT-03 | 在只安装了 git 的干净虚拟机中，发布的二进制能完成 E2E-1 | 人工 | NFR-06 | M6 |
| MVP-PLT-04 | `doctor --json` 报告：版本、数据库路径、schema 版本、权限、Hook 与 Skill、spool 积压 | 集成 | NFR-09 | M3 |
| MVP-PLT-05 | `data-model.md` 与迁移脚本产生的实际 schema 一致 | 审查 | NFR-07 | M2 |

### 2.16 通道识别（CH）

| ID | 前置 → 操作 → 期望 | 验证 | 需求 | 里程碑 |
|---|---|---|---|---|
| MVP-CH-01 | 在伪终端中执行人类专属命令（`review`、`source ack`、`scope confirm`），同时设置 `CLAUDECODE` 或 `CLAUDE_CODE_ENTRYPOINT`（值为 canary）→ 退出码 3，不写入；不设置时进入交互式确认，拒绝确认则不写入；数据目录、输出和日志中都找不到这个 canary。WP1.2 用只在测试中存在的替身命令验证通道识别、拦截、确认、取消和 canary（真实终端，三个平台）；`source ack`、`source remove`（以及 `source add --allow-any-type`）已在 WP1.4 用真实命令验证；`review`、`scope confirm` 随 WP1.6、WP1.8 完成；`propose --apply`、`question dismiss` 在 M1 退出证据的补充中验证（PR #16） | 集成 | FR-BL-04、FR-SRC-03、FR-TGT-04 | M1 |
| MVP-CH-02 | stdin 是终端但 stdout 被重定向，或者 stdin 不是终端 → `agent_cli`，人类专属命令返回退出码 3；加上 `--json` 不改变识别结果；只读命令在任何通道中都可以执行。WP1.2 已用替身命令和只读命令 `version`、`doctor` 验证；`source remove` 已在 WP1.4 验证（stdout 重定向、stdin 非终端、带 `--json`）；`source ack`、`source add --allow-any-type`、`propose --apply`、`question dismiss` 的 stdout 重定向与 stdin 非终端，在 M1 退出证据的补充中用真实子进程验证（PR #16）；其余人类专属命令随其工作包验证 | 集成 | FR-BL-04、FR-SRC-03 | M1 |

## 3. 关键不变量

每个 E2E 结束时都会自动断言以下性质：

1. 不存在任何由 Agent 通道写入的 `decisions`、`scope_confirmations`、`source_acks` 或 `baseline_versions`；不存在由 Agent 通道回答或驳回的问题，也不存在由 Agent 通道修改的片段保存模式。
2. 对每个 `ready` 的判定，由一个**独立实现的检查器**（测试代码，不复用推导引擎；规格见 §3.1）重新核对：
   - `S` 不为 NULL；
   - 范围没有过期、不为空，也不是全部豁免；
   - 每个范围内的 AC 都有适用于 `S` 的决定；
   - 自动化 AC 的 `normal` 决定，都在 `S` 上有对应的合格 `pass`；沿用只依据隔离级执行；
   - 独立重算的 `evidence_grade` 与输出一致；
   - 没有阻塞问题，也没有未确认的来源变化。
3. 不存在仅由 Claim 推出的 `ready_for_review`、`accepted` 或 `ready`（手动 AC 的 `manual_claimed` 规则除外，它只会推出 `ready_for_review`）。
4. 数据目录中找不到任何 C0 canary。
5. 对仓库的写入，只有 [architecture.md §10.6](architecture.md#106-对仓库的写入) 允许的那些。
6. 所有在 `schemas/` 中有对应 schema 的输入和输出都能通过校验（面向人的读取模型的 schema 在对应模块开工前补齐，见 data-model §8）。
7. `seq` 严格递增；把事实打乱顺序后，推导结果不变。
8. 交付状态为 `ready`，当且仅当不存在阻塞差距。
9. MVP 构建中，不存在 `evidence_grade = snapshot_isolated` 的执行；`delivery.evidence_grade` 只可能是 `observed` 或 `none`；没有任何 AC 在没有人参与的情况下，依据观测级证据变为 `accepted`。
10. 每条决定的 `seen_decision_id`，都等于写入时该 AC 当前修订下的前一条最新决定（第一条为 NULL）。
11. 每条范围确认的 `seen_scope_confirmation_id`，都等于写入时该目标的前一条范围确认（第一条为 NULL）；它的 `acknowledged_waivers` 等于写入时的有效豁免集合。

### 3.1 独立 `ready` 复核器的规格（F5）

> 状态：**已冻结**（2026-10-18，产品负责人；审查 045 GO，review-resolution §57），是 WP2.2 的实现依据。复核器实现不变量 #2，只在测试代码中使用（WP2.2）。输入事实的规格见 [data-model.md §6.13](data-model.md#613-推导的输入f4)，当前值输入与重放见 [§6.13.1](data-model.md#6131-当前值的取得重放与---diff-since)，合法事实域见 [§6.14](data-model.md#614-合法事实域与畸形输入)。当前值输入的要求由审查 044 的 B1 补齐。

**独立性**：

1. 复核器放在单独的测试专用包中，不导入推导引擎（`internal/domain/derive`），也不导入 `scope`、`baseline` 等生产包的计算函数。核对逻辑只用标准库；E2E 中读取数据库的前端只额外使用 SQLite 驱动。由测试检查导入关系，两个方向都检查。
2. 谓词按 prd §3.2 的 `ready` 定义、本节的不变量 #2 与 data-model §6.8 的文字写出（下表），反例从 data-model §6.3 的规则推出，而不是从推导引擎的代码推出。只检查导入图不能排除共享的错误假设，所以谓词与反例在实现之前接受独立审查。
3. 不通过放宽 `ready` 的条件来消除两个实现之间的差异。

**谓词**（全部成立才是 `ready`）：

| # | 条件 |
|---|---|
| R1 | 存在基准版本 |
| R2 | 存在活动的交付目标，并且 `S` 不为 NULL |
| R3 | 该目标存在范围确认（`seq` 最大的一条），并且没有过期：它的基准版本等于当前版本，它的来源集合哈希等于由未移除的来源独立重算的哈希 |
| R4 | `RS`（范围中仍为 `active` 的需求）不为空 |
| R5 | `RS` 中每个需求至少有一个 `active` AC |
| R6 | 对 `AS` 中的每个 AC：(a) `requirement_revision` 等于父需求的修订；(b) 当前决定 `D` 存在；(c) `D` 是豁免，或者是适用于 `S` 的接受：手动 AC 的 `manual` 接受且 `D.snapshot = S`；自动化 AC 的 `exception` 接受且 `D.snapshot = S`、`V` 不是失败；自动化 AC 的 `normal` 接受且 `V = pass`，并且 `D.snapshot = S`（直接）或满足沿用条件（只在通用合法事实域中出现） |
| R7 | `AS` 不是全部豁免 |
| R8 | 没有阻塞问题：关联到 `RS` 中需求、`AS` 中 AC 的，或没有关联任何条目的 `open` 且 `blocking` 的问题都不存在；关联条目已被拒绝的问题不计入 |
| R9 | 没有未确认的来源变化：每个未移除的来源，观测哈希都等于已确认的哈希（文件缺失也是变化） |
| R10 | 独立重算的 `evidence_grade` 与输出一致：取 `AS` 中 `normal` 接受所依据的执行的最弱等级（直接：`D.evidence_id` 对应的执行；沿用：`S` 上的隔离级执行）；没有 `normal` 接受时为 `none`。这是输出一致性断言 |

**R6 与 data-model §6.3 第 6 条**：在合法事实历史中，直接适用的 `normal` 接受写入时，`S` 上已有当前修订与配置下的合格通过；检查配置的变化是强制的 `material` 修订，修订变化之后这条决定就不再是当前决定。所以 `V` 只能是 `pass` 或失败，失败由第 4 条先处理。"第 4 条加第 6 条"与 R6 在合法事实域上等价，不是规格冲突；差别只可能出现在畸形输入中。

**输入域**：

- 双向对照只在合法事实域上进行：MVP 合法事实域，以及（需要 data-model §4.11.2 的输入比较）通用合法事实域。
- 生成器必须产生合法但不能 `ready` 的历史，以及共享 `seq` 的合法写入（至少包括"审核接受多个条目并附带范围"与 `propose --apply`），不能为了双向一致而预先排除。
- 生成器同时变化当前值输入（data-model §6.14）：项目设置的各种取值、`S` 为 NULL、来源文件缺失。
- 畸形输入另行测试：两个实现都不崩溃、结果确定；复核器按它自己的谓词给出结论；推导引擎不作业务结果的保证（data-model §6.14，选项 (a)）。

**两套转换器**：各自独立实现，只做原始事实的映射。

- 保留每条事实的 `seq`，包括共享同一个 `seq` 的全部关联事实；不按 `seq` 去重、合并或丢弃记录；记录按各表既有的主键、复合键或约定标识保留，不要求新增 `id` 列。
- `as_of_seq` 分别交给两个实现，由它们各自过滤：保留 `seq ≤ as_of_seq` 的全部事实，同一个 `seq` 的关联事实一起保留或一起排除。转换器不预先过滤。
- 同一个 `seq` 之内的顺序只用 `change_index`、`item_index` 与 data-model §5.5 的应用顺序。
- 保留项目与实体的归属、修订与配置哈希、证据引用与时序（决定的 `evidence_id`、`snapshot`、`mode`、`seen_decision_id` 与 `seq`；执行的上下文、结果、等级与 `seq`），以及全部决定与执行，包括失败的、被取代的记录。
- 不能提前选出当前决定、计算适用性或合格性，也不能合并同一 AC 的多条记录。
- 当前值输入（`S`、来源观测、项目设置）由测试前端取得一次，作为同一份记录交给两个实现；两个实现都不自行读取数据库中的设置、解析 ref 或观测磁盘（data-model §6.13.1）。
- R1 ~ R10 不读取项目设置：设置不改变交付状态，也不改变阻塞差距的存在性、类型与是否阻塞（data-model §6.13），所以只有设置不同的两组输入，`ready` 的结论必须相同。不断言差距对象逐字段相同：诊断原因码可以不同。样本中要有这样的输入对。

**必须有的反例**：在合法事实历史中构造。每条以一个条件为目标，复核器必须在结果中指出该条件；条件之间有结构依赖时（例如没有基准版本就不可能有有效的范围确认），反例只需违反目标条件及其必然的连带条件。

- 当前决定是拒收（即使更早有过接受）；
- 在同一个 `S` 上接受之后，又有一次合格的失败；
- 父需求已被实质修订，而 AC 没有后续处理；
- 范围按版本过期；按来源集合过期；范围中的需求都已退役；需求没有 AC；`AS` 全部豁免；
- 三类阻塞问题各一个，以及关联条目已拒绝、因而不阻塞的对照；
- 来源内容变化；来源文件缺失；
- `S` 为 NULL；决定针对另一个快照（手动、例外、没有沿用的 `normal` 各一个）；只有旧修订的决定。
- 输出一致性：篡改输出的证据等级，R10 失败（不是事实的反例）。

**必须有的正例**：

- 依据观测级执行的 `normal` 接受 → `ready`，证据等级 `observed`；
- 手动接受、例外接受与豁免的组合 → `ready`，证据等级 `none`；不能要求所有 AC 都有自动化的通过；
- 共享 `seq` 的审核历史（一个版本、多条变更、引用与附带的范围确认共用一个 `seq`），`as_of_seq` 恰好等于这个 `seq`；
- （通用合法事实域）依据合成的隔离级执行沿用 → `ready`，证据等级 `snapshot_isolated`。

**双向对照**：推导为 `ready` ⇒ 复核器通过；复核器认为全部条件成立 ⇒ 推导为 `ready`。样本中要有足够数量的 `ready` 结果与合法的不能 `ready` 的历史，以免对照空转。不一致都要查明是哪一方或哪条规则的问题，不能为了一致而改动任意一方；真正的规格歧义报告并等待决定。

## 4. 反例与异常场景（CX）

以下每一项都必须在 M2 结束时通过（标注了其他里程碑的除外）。"**绝不**"表示断言在任何中间步骤都不成立。

| ID | 场景 | 步骤 | 期望 | 里程碑 |
|---|---|---|---|---|
| **CX-01** | **代码改变，但没有重新运行检查**（审查反例 1） | AC 在提交 A 上合格通过，并以 `normal` 验收，交付状态为 `ready` → 在 `main` 上提交 B，删除该功能，**不运行检查** | AC 为 `needs_reverification`（`code_changed`）；交付状态为 `needs_reverification`；**绝不**为 `ready`。之后在 B 上运行检查并失败 → `failing`，交付状态为 `in_progress`。 | M2 |
| **CX-02** | 同一提交，但工作区有未提交的修改 | 目标在 A 上为 `ready` → 在主 worktree 中修改代码（不提交），运行检查并失败 | 这次执行不合格；交付判定仍然针对提交 A，保持 `ready`；用这次执行的 `evidence_id` 作出决定 → 被拒绝（它不是 `S` 上最新的合格执行）。工作区视图显示"未提交的修改导致检查失败"。 | M2 |
| **CX-03** | **两个 worktree 的结果相反**（审查反例 2） | `main`（快照 M）上 AC 的检查失败；feature worktree（提交 F）上通过 | AC 为 `failing`；feature 上的通过只出现在"其他上下文"中，**不影响**交付判定。之后 `main` 快进合并到 F → 快照变为 F，feature worktree 上已有的合格 `pass` 生效，AC 变为 `ready_for_review`。 | M2 |
| **CX-04** | 从 feature worktree 发起检查 | 在 feature worktree 的 cwd 中执行 `check run` | 执行在 feature worktree 中进行（见 MVP-WT-02），不在主 worktree 中 | M2 |
| **CX-05** | **只确认了部分基准**（审查反例 3） | 提取 8 条需求，只确认 1 条并验收，没有范围确认 | 交付状态为 `not_assessed`（`NO_SCOPE`）；**绝不**为 `ready`。确认范围时 `pending_items_excluded` 与实际不符 → 退出码 5；范围为空 → 退出码 2；明确确认"范围只包含这 1 条，排除 7 条待审条目"之后，才可能得到 `ready`。 | M2 |
| **CX-06** | 范围全部豁免 | 范围内的唯一一个 AC 被豁免 | `not_assessed`（`ALL_WAIVED`） | M2 |
| **CX-07** | **父需求实质变更**（审查反例 4） | 需求的全部 AC 已验收，交付状态为 `ready` → 需求以 `material` 方式修订，审核时选择 `defer` | 所有 AC 为 `needs_reverification`（`parent_revised`），差距中有 `PARENT_REVISED`，交付状态不是 `ready`。之后 `reaffirm` 其中一个 → 该 AC 恢复为 `accepted`（快照没变）；`retire` 另一个 → 移出；新增的 AC → `in_progress`。 | M2 |
| **CX-08** | **在过期的页面上提交验收**（codex-001 反例 5、codex-002 B2） | 页面甲读取 AC 详情（rev 1，证据 E1，快照 S1，`current_decision_id = null`）之后，分别发生：(a) AC 被修订为 rev 2；(b) 在 S1 上又执行了一次，得到 E2；(c) 目标前进到 S2；(d) 父需求被修订；**(e) 页面乙对同一快照、同一 AC 作出拒收（D1）** | 五种情况下，页面甲用旧的值（(e) 中为 `seen_decision_id = null`）提交接受 → CLI 返回退出码 5，API 返回 409，**不写入**；(e) 中拒收保持有效，`error.current` 给出 D1。**(f)** 客户端不会自动重试；页面甲重新读取后，携带 `seen_decision_id = D1` 明确改判为接受 → 成功，写入 D2。从没有决定开始，最终**恰好有两条决定**：D1（拒收）和 D2（接受，`seen_decision_id = D1`）。那次被拒绝的过期接受不留下任何决定记录。UI 部分在 M5 验证。 | M2 / M5 |
| **CX-09** | **同一毫秒的事实**（审查反例 6） | 把时钟固定，让两次执行（先 `fail` 后 `pass`）的时间戳相同；再把时钟回拨，让后发生的拒收时间戳更早 | 先后顺序只由 `seq` 决定：`V` 取 `seq` 更大的那次执行；拒收是否生效，同样按 `seq` 判断。结果与时间戳无关。 | M2 |
| **CX-10** | **局部 audit**（审查反例 7） | 只对 REQ-001 执行 `audit`（`audit_scope = requirements`） | `inventory_coverage < 1`；`init_progress` 不是 `inventoried`；其他 AC 的进度为 `unknown`；出现 `INVENTORY_INCOMPLETE` 警告；交付判定不受盘点影响（仍然需要快照证据） | M2 |
| **CX-11** | **`purge` 之后恢复备份**（审查反例 8） | 有项目 A 和 B；迁移时生成备份 X → `purge` A，并选择保留备份 → 手动把 X 复制回数据库位置 → 执行任意命令 | 打开数据库时，墓碑生效，A 被自动删除；B 保持不变；日志中有记录。如果 X 的 schema 版本较旧，打开时会触发迁移：墓碑在迁移**之前**执行，因此新生成的迁移前备份中也找不到 A 的数据（用 canary 断言）。如果 `purge` 时选择默认处理方式，X 会被删除。 | M2 |
| **CX-12** | **Agent 自述完成，但没有有效证据** | 分别构造：(a) Claim `done`，没有任何执行；(b) Claim `done`，只在有未提交修改的 worktree 中执行通过；(c) Claim `done`，只在其他提交（feature）上执行通过；(d) Claim `done`，在 `S` 上执行失败；(e) 手动 AC 的 Claim `done` | (a)(b)：`in_progress`（`claimed_without_evidence`），有 `CLAIMED_WITHOUT_EVIDENCE` 差距；(c)：其他提交上的合格 `pass` 构成历史，所以为 `needs_reverification`（`code_changed`），**不会**是 `ready_for_review`；(d)：`failing`，并带 `conflict`；(e)：`ready_for_review`，交付状态最多也只是 `ready_for_review`。五种情况都**绝不**为 `ready`。 | M2 |
| **CX-13** | Claim 基于旧修订 | Agent 读取 rev 1 → 人把 AC 修订为 rev 2 → Agent 提交 `target_revision = 1` | 退出码 5，不写入 | M2 |
| **CX-14** | 提案基于旧版本 | Agent 读取 v3 → 人审核生成 v4 → Agent 提交 `base_version = 3` | 退出码 5，不写入 | M1 |
| **CX-15** | 检查执行期间 AC 被修订 | 启动一个长时间运行的检查 → 同时由人以 `material` 方式修订该 AC → 检查结束 | 执行记录中的 `criterion_revision` 是旧值；对新修订来说 `V = none`。同一批次中，修订发生在前一项执行期间时，后面的项同样按命令开始时固定的配置执行并记录旧修订（2026-10-25 产品决定 D13，MVP-EV-26） | M2 |
| **CX-16** | 检查结束时 HEAD 已变化，或工作区已被弄脏 | 见 MVP-EV-09 | `context_changed`，不合格 | M2 |
| **CX-17** | Agent 自行确认来源变化 | 来源发生变化 → Agent 通道执行 `source ack` | 退出码 3；`SOURCE_CHANGED` 仍然存在；交付状态不是 `ready` | M1 / M2 |
| **CX-18** | 范围过期 | 范围在 v3 上确认 → 审核生成 v4（新增一条 `must` 需求），但没有重新确认范围；或者 Agent 新登记了一个来源 | `SCOPE_STALE`，交付状态为 `needs_reverification`，**绝不**为 `ready` | M2 |
| **CX-19** | 测试被削弱 | AC 声明了 `inputs`，在 A 上已验收 → 在 B 上修改测试文件（弱化断言），检查通过 | 不沿用验收：`ready_for_review`（`inputs_changed`），交付状态不是 `ready`。只改变 mode、对象 ID 不变（例如可执行位，或把文件换成链接文本相同的符号链接）同样是 `inputs_changed`（data-model §4.11.2） | M2 |
| **CX-20** | 例外验收不沿用 | 在 A 上作出例外验收 → 目标前进到 B | `needs_reverification`（`code_changed`） | M2 |
| **CX-21** | 同一快照上结果不一致 | 在 A 上先 `pass` 后 `fail`，再 `pass`（不稳定的检查），之前在其他快照上有过验收 | 不沿用：`ready_for_review`（`inconsistent_results`），带 `INCONSISTENT_RESULTS` 警告 | M2 |
| **CX-22** | 目标无法解析，或者不是 git 项目 | 删除目标分支；或者在非 git 目录中执行 | `not_assessed`，**绝不**为 `ready` | M2 |
| **CX-24** | marker 让工作区变脏 | 不写 `.git/info/exclude`（`--no-exclude`，或者 exclude 文件只读）→ 在主 worktree 中运行检查；另外，在 `.claude/settings.local.json` 没有被 git 忽略的仓库中执行项目级 `setup` | 未跟踪的 `.specmetry/project.json` 不影响 `tree_state`，执行仍然合格；项目级 `setup` 被拒绝，并建议改用用户级安装 | M2 / M3 |
| **CX-25** | Agent 试图不经返工解除拒收 | (a) 人在 `S1` 上拒收 AC → Agent 在同一提交上再次执行 `check run` 并通过；(b) 在拒收**之前**，提交 `S2` 上已经有一条合格通过，之后目标前进到 `S2`，但没有重新执行检查 | (a)(b) 都仍然是 `rejected`，交付状态不是 `ready`：(b) 中那条通过的 `seq` 小于拒收的 `seq`。之后在 `S2` 上重新执行检查并通过 → `ready_for_review`（`rework_submitted`）。人重新读取后，在 `S1` 上明确改判为接受 → `accepted`。 | M2 |
| **CX-26** | **检查期间修改源码，然后恢复**（codex-002 B1） | **夹具**：范围内只有一条自动化 AC；它在较早的快照 P 上已经以 `normal` 验收；交付目标随后前进到提交 A（A ≠ P），A 上还没有任何执行。**操作**：在 A 上执行检查，检查脚本先改写一个受跟踪的文件，让测试读到改写后的内容并通过，然后用 `git checkout -- <file>` 恢复；开始和结束时工作区都干净，HEAD 都是 A | 这次执行记为 `observed`，合格（观测级无法发现中途变化，这是已声明的边界）。系统**绝不**：把它记为 `snapshot_isolated`；据此沿用 P 上的验收；在没有确认 `observed_evidence` 的情况下接受验收；在输出中称其为"快照已验证"。**人工验收之前**：AC 为 `ready_for_review`（原因码 `observed_evidence`），交付状态为 `ready_for_review`，`evidence_grade` 为 **`none`**（A 上还没有被验收的自动化 AC）；不带 `observed_evidence` 的验收 → 退出码 2。**人确认 `observed_evidence` 并在 A 上 `normal` 验收之后**：AC 为 `accepted`（`direct`，带 `observed_evidence` 标记），交付状态为 `ready`，`evidence_grade` 为 **`observed`**。M7 追加：以 `--isolation required` 执行时，改写会因为只读而失败，或者在执行后的树比对中被发现（`context_changed`）；无论哪种情况，都不会产生隔离级的通过。 | M2（M7 追加） |
| **CX-27** | **检查期间 HEAD 从 A 切到 B 再切回 A**（codex-002 B1） | **夹具**：与 CX-26 相同（P 上已有验收，目标前进到 A）。**操作**：检查脚本执行 `git checkout B`，运行测试，再 `git checkout A` | 断言与 CX-26 相同：只是观测级，不沿用；人工验收之前为 `ready_for_review`，`evidence_grade = none`；人确认风险并验收之后为 `ready`，`evidence_grade = observed`；不称为"快照已验证"。HEAD 在结束时没有回到 A 的情况，仍然是 `context_changed`（MVP-EV-09）。M7 追加：隔离级执行在私有快照目录中进行，用户 worktree 的 HEAD 变化不会影响它。 | M2（M7 追加） |
| **CX-28** | **执行环境无法保证源码只读**（codex-002 B1） | (a) MVP 中执行 `check run --isolation required`；(b) M7 中依次模拟：隔离后端不可用；后端只能修改文件权限位（同一用户可以恢复写权限）；执行前的写入探测成功；物化内容与 `S` 的树不一致 | (a)(b) 中的 `required`：不执行，不写入任何 `check_runs`，退出码 6（`isolation_unavailable`，并给出具体原因）。(b) 中的 `auto`：降级为观测级执行，`evidence_grade = observed`，并记录 `isolation_fallback_reason`。任何情况下，都不存在未经验证的 `snapshot_isolated` 记录。 | M2（a）；M7（b） |
| **CX-23** | Agent 试图通过 Dashboard 代做验收 | Agent 通道执行 `dashboard` | 输出中没有令牌（MVP-UI-14）；Skill 静态检查确认禁止这种做法（MVP-SK-03）。**已知边界**：如果 Agent 读取了仅当前用户可读的运行时文件，它仍然可以调用 API。这一点已在 [architecture.md §5](architecture.md#5-信任边界与操作通道) 中如实声明。 | M5 |
| **CX-29** | **`cmd` 的多行命令只执行第一行而“通过”**（2026-10-25 产品决定 D3） | 一个 `shell = cmd` 的检查，命令为两行：第一行成功，第二行是会失败的测试；分别经 `propose`、`review` 的 `accepted_with_edits` 与人工编辑提交，以及作为数据库中已保存的旧配置执行 | 三条写入路径都拒绝（退出码 2，不写入）；已保存的旧配置照常显示，`check run` 在执行前拒绝（退出码 2，没有新行）。**绝不**出现这一检查的 `pass` 记录 | M2 |
| **CX-30** | **检查留下保证范围之外的进程**（2026-10-25 产品决定 D7） | macOS 上检查以 `setsid` 启动一个后代后退出；Linux 上同样的后代；Windows 上检查经计划任务或服务启动一个不在 Job 中的进程（能构造的平台） | Linux：后代在排他区间内被收养并经 pidfd 终止、回收。macOS 与 Windows 的范围之外的进程：执行按两个时刻照常记录；Specmetry **绝不**向它发信号或回收它，文本与 JSON 输出**绝不**称“所有检查进程都已结束”；它之后对 worktree 的改动在下一次执行的开始上下文中可见（不再 `clean` → 不合格）；`observed_evidence` 的风险确认不改变这些结论 | M2 |

## 5. 端到端场景

全部使用录制的 Agent 输出回放；人工操作通过 API 或伪终端脚本完成；最终结果与 oracle 比对，并断言 §3 的不变量。

| ID | 场景 | 步骤概要 | 期望 |
|---|---|---|---|
| E2E-1 | 只有文档 | `fx-docs-only`：init → 来源 → 提案 → 审核，并用 `item_id` 附带范围确认 → status | 范围内的 AC 全部为 `in_progress`，缺少 AC 的需求为 `needs_criteria`；有 `NEEDS_CRITERIA` 和 `BLOCKING_QUESTION`；交付状态为 `in_progress`；`init_progress = inventoried`（`docs_only` 不需要 audit），没有 `INVENTORY_INCOMPLETE` |
| E2E-2 | 开发中 | `fx-in-dev`：按 E2E-1 的步骤完成初始化和范围确认 → 全量 audit → 在主 worktree 中执行 `check run --all` → 追加带 trailer 的提交 | AC 状态与 oracle 一致；"自述完成但失败"的 AC 带 `conflict`；新提交被正确关联 |
| E2E-3 | 文档不完整 | init（`incomplete_docs`）→ 提案中有反推条目 → 范围确认缺少 `missing_goals_reviewed` 时失败 → 补充之后成功 | 反推标识存在；范围确认的约束生效 |
| E2E-4 | 日常循环与 worktree | 重放 Hook → 在 feature worktree 中 sync 并执行检查 → `main` 快进合并 → 在主 worktree 中执行检查 | 合并之前，feature 上的结果不影响交付；合并后，同一 SHA 上的合格执行生效 |
| E2E-5 | 需求变更 | 来源变化（`SOURCE_CHANGED`）→ 人确认来源（来源集合改变，出现 `SCOPE_STALE`）→ Agent 提交需求的 `material` 修订 → 审核时对一个 AC 选择 `defer`，并重新确认范围（出现 `PARENT_REVISED`，`SCOPE_STALE` 消除）→ 人工编辑修订该 AC（`PARENT_REVISED` 消除） | 各差距按上述顺序出现和消除 |
| E2E-6 | 交付 | 解决全部差距 → 逐条验收（确认观测级风险）→ `ready`（`evidence_grade = observed`）→ 目标前进到新提交 → `needs_reverification` → 执行 `check run --all` → 自动化 AC 为 `ready_for_review`（`observed_evidence`，MVP 中不沿用），手动 AC 为 `needs_reverification` → 人重新验收 → `ready` | 每一步的状态都与 oracle 一致；`ready` 的判定可以用输出记录的完整输入重放（同样的 `as_of_seq`、`S`、观测哈希与项目设置，加上同一数据库中的事实，data-model §6.13.1）；任何输出都没有"快照已验证"的说法 |
| E2E-7 | 人类专属操作被阻止 | 重放拦截 payload；在非 TTY 中执行 `accept`、`review`、`scope confirm`、`source ack`、`target set`、`question answer`、`question dismiss` | 都被拒绝；没有新增决定、版本、范围确认或来源确认；有 `guard_denied` 事件 |
| E2E-8 | 删除与恢复 | 见 CX-11，并加上 C0 canary | 不复活；没有残留 |

## 6. 需求追溯

| 需求组 | 验收项 |
|---|---|
| FR-INIT | INIT-01 ~ 11、PRV-01（FR-INIT-07） |
| FR-SRC | SRC-01 ~ 08、CX-17 |
| FR-BL | BL-01 ~ 28、CX-07、CX-14 |
| FR-TGT | TGT-01 ~ 11、CX-05、CX-06、CX-18（FR-TGT-05）、CX-22（FR-TGT-03）、INIT-09（FR-TGT-03） |
| FR-WT | WT-01 ~ 06、EV-15、EV-16、COL-10、CX-03、CX-04 |
| FR-INV | INV-01 ~ 06、CX-10 |
| FR-CLM | CLM-01 ~ 05、CX-12、CX-13 |
| FR-EV | EV-01 ~ 30、CX-02、CX-15、CX-16、CX-26 ~ 30 |
| FR-ACC | ACC-01 ~ 11、CX-01、CX-08、CX-19 ~ 21、CX-25 ~ 27 |
| FR-ST | ST-01 ~ 15、CLM-01（FR-ST-07）、CX-09、CX-12（FR-ST-07）、CX-22 |
| FR-COL | COL-01 ~ 15 |
| FR-UI | UI-01 ~ 16、CX-08 的 UI 部分（FR-UI-13）、CX-23 |
| FR-PRV、FR-STO | PRV-01 ~ 10、STO-01 ~ 07、EV-05、EV-11、BL-23、BL-26、CX-11（FR-STO-03）、CX-24 |
| FR-SK | SK-01 ~ 08 |
| 通道识别（architecture §5.1） | CH-01 ~ 02、ACC-02、ACC-03、E2E-7 |
| NFR | ST-08、UI-04、COL-01、PLT-01 ~ 05、PRV-01、STO-07 |

P2 项（FR-EV-07、FR-EV-09、FR-ST-10、FR-ST-11、FR-COL-06、FR-UI-07、FR-UI-08）不在 MVP 中，没有对应的验收项；FR-EV-09（隔离执行器）的验收在 M7 中追加到 CX-26 ~ CX-28 和 MVP-ACC-09。

### WP2.3 B1 方案 B 的直接回归（负责人批准，review-resolution §71）

扫描两空值必须真实传入，按 Git 文件身份作行为正反对照；不支持 exit 6 无回退。覆盖两提交、合并／多父、增量排除、缺失／损坏及往返调度，禁止成功漏读。探测自身也使用 v3 合作协议。Windows 原子排他创建冲突、创建身份及同句柄删除；Unix 明确保留获批的 mkdir/open 与最终 unlink/rmdir 窗口，不以域外干预测试证明代码已阻止它。

Review 061 两项直接回归（决定 §72）：空清单和非空清单在 owner 预检前缺失或记录无效时，读取／清理／遗留认领必须环境错误且全部现存内容不变；公开 Clean 覆盖读取前、读取后及清理预检。owner 正常存在可清理，本次可信移除 owner 后目录重试正常，额外／缺失／替代条目原回归继续有效。真实 Git 用原生停止确认和退出通知同步：调用方退出 99 后 Git 仍暂停，下一 Scratch 不得删任何内容；恢复 Git 并确认实际退出后，未被原调用方记录的完成仍保留报错。正常 Wait、取消 Wait 和启动失败已记录完成时允许清理／回收。两类消费者（Clean、空覆盖探测）均覆盖；v1／v2 子树不接管。超时只算测试失败，不作结束证明；平台阻止操作与实际调度通过分开记录。

Review 062 B1-R2-S 直接回归：合法完整清单预检通过、真实 Git 已启动且 Wait 确认正常退出、完成记录完整写入后，仅完成 Sync 注入一次 I/O 失败。直接读取、公开 Clean 与公开 Log 触发的空覆盖探测均 exit 6，本次目录、owner 和全部现存内容快照不变，释放自身句柄／租约；不能被同次 begin／finish 或 cleanup 覆盖失败。启动前 Sync 失败须零启动，完成写入失败也保留；正常完成同步、启动失败、取消等待、可信回收及 ownerGone 重试仍成立。另用独立进程验证后续 Scratch：完整规范 false 且身份／清单一致可回收，true／不完整记录保留报错；不得将后续可读 false 误称前次 Sync 成功或将本次保留宣称为跨进程永久屏障。详见规范 §14.3。

验证正常多进程、锁竞争、取消、崩溃、可信遗留、未知／额外条目保留与清理失败；失败检查既有字段、全部关联、游标及 seq 不变。按创建清单清理，无前缀／年龄／PID 接管。macOS 实际运行与跳过项单列。B2、B3、N1 直接回归，不重新验收旧模块，不改变正式验收编号及推导规则。
