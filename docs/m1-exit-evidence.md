# M1 退出条件证据核对

- 核对日期：2026-10-17（第 2 版；codex-review-038 之后做了文档收尾，见修订记录）
- 核对对象：master 上的合并提交 `5aad555`（PR #17 合并后），树 `13ee736`。证据来自三批提交，每批的适用提交如下：

  | 证据 | 来源 | 适用提交 | 审查 | master CI |
  |---|---|---|---|---|
  | 九个工作包的原有证据 | WP1.0 ~ WP1.8 的合并（第 2 节） | `19600ec`（PR #14 合并） | 各工作包的最终 GO | [run 39000934831](https://github.com/XKHoshizora/specmetry/actions/runs/39000934831)，8/8 |
  | 6 项缺口的补充证据 | PR #16（测试、测试辅助程序、acceptance-criteria §1 与 SRC-07 标注） | `01bd50f`（审查目标 `e1d2ea7`） | codex-review-036 GO | [run 39012262429](https://github.com/XKHoshizora/specmetry/actions/runs/39012262429)，8/8 |
  | 项目名称的安全修复（M1 最终验收前的独立修复） | PR #17 | `5aad555`（审查目标 `a106f9e`） | codex-review-037 GO | [run 39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)，8/8 |

  后两批没有改动前一批的业务实现：PR #16 只有测试与文档；PR #17 只改了项目名称的校验，以及对应的 schema 与测试。第 4 节"CI"一列取最新的 master CI（run 39019706495，`5aad555`）。
- 依据：
  - [implementation-plan.md §5 M1](implementation-plan.md#m1-基准来源交付目标与范围) 的退出条件；
  - [acceptance-criteria.md](acceptance-criteria.md) 中标注 M1 的验收项与反例；
  - [review-resolution.md](review-resolution.md) 的处理记录；
  - `docs/reviews/` 中各工作包的 Codex 审查原文；
  - master 的三平台 CI 日志，以及本地 Windows 与 WSL Linux 的运行。
- 方法：
  - 对每个验收项，逐条对照验收文字与测试中的断言，而不只是看测试是否运行；
  - 读取 CI 日志中每个测试的 `PASS` / `SKIP` / `FAIL`；
  - 需要真实旧版 git 的检查，在本地 WSL 中用与 master 相同的代码重跑。
- 结论在第 10 节。本文整理 M1 验收所依据的证据，本身不宣布验收。**M1 已由产品负责人于 2026-10-17 正式验收**（验收对象 `7e77fd6`；review-resolution §50）。
- 修订记录：
  - 第 1 版（提交 `cf72f20`）：59 个 P0 项中 53 项 PASS，6 项 INCOMPLETE（CH-01、CH-02、SRC-07、BL-21、BL-27、PRV-07），建议补齐证据后再验收。
  - 第 2 版（提交 `9b8ea5d`，codex-review-038 的审查目标）：6 项缺口由 PR #16 补齐，审查 036 GO（SRC-07 由审查方独立审查通过）；项目名称修复由 PR #17 完成，审查 037 GO。每项缺口如何关闭，见第 4 节与第 10 节。
  - 第 2 版的文档收尾（审查 038 之后，不改变结论）：
    - 按审查 038 的建议，改写 PASS 的定义；
    - CH-01、CH-02 的依据改写为证据链。第 2 版曾以"这两格不是第 1 版的缺口"和"按'或'可以接受"作为说明，这两句话本身不能证明验收成立，现已替换为审查 038 的裁定；
    - 第 10 节的结论更新为审查 038 的 GO。

## 1. 退出条件

> **退出条件（P0）**：acceptance-criteria 中标注 M1 的全部 P0 项（包括 CH-01、CH-02，以及 SRC-08、TGT-10、WT-01 的 M1 部分），以及 CX-14、CX-17（CLI 部分）。标注为"M1 / M2"的项，在 M1 中只验证写入侧（退出码、写入的数据、`init --json` 的输出），推导侧（差距、交付状态）在 M2 中验证。
> **可选（P1）**：MVP-INIT-08、INIT-10、INV-05。

按 acceptance-criteria 中"里程碑"一列统计，含 M1 的共 62 项：

- P0 59 项：57 个常规验收项，以及 CX-14、CX-17；
- P1 3 项：INIT-08、INIT-10、INV-05，与 implementation-plan 列出的 P1 一致。

## 2. 工作包的验收与合并

| 工作包 | 内容 | 最终审查 | PR | 合并提交 | 处理记录 |
|---|---|---|---|---|---|
| WP1.0 | 测试夹具 | codex-review-013 GO | #4 | `84e6e70` | review-resolution §17 ~ §18 |
| WP1.1 | 项目与 worktree 登记、`init` | codex-review-017 GO（015、016 NO-GO） | #6 | `568432d` | §20 ~ §22 |
| WP1.2 | 通道识别与确认 | codex-review-014 GO | #5 | `4c5a038` | §19 |
| WP1.3 | 密钥识别 | codex-review-011 GO（010 的处理） | #3 | `cee87b4` | §15 ~ §16 |
| WP1.4 | 来源 | codex-review-019 GO（018 NO-GO） | #7 | `35d060f` | §23 ~ §24 |
| WP1.5 | 提案、引用校验、PD-26、TD-26 | codex-review-024 GO（020 ~ 023 NO-GO） | #8 | `429e94e` | §25 ~ §30 |
| WP1.6 | 审核、基准、`propose --apply` | codex-review-026 GO（025 NO-GO） | #9 | `c9c9cf8` | §31 ~ §32 |
| WP1.7 | 问题 | codex-review-028 GO（027 NO-GO） | #10 | `867bd60` | §33 ~ §34 |
| WP1.8 | 交付目标与范围 | codex-review-035 GO（034 NO-GO） | #14 | `19600ec` | §43 ~ §45 |

另有以下修复与证据补充，同样已合并：

| 修复 | 审查 | PR | 合并提交 | 处理记录 |
|---|---|---|---|---|
| CLI 错误不回显不可信参数 | codex-review-030 GO | #11 | `a91016e` | §35 ~ §37 |
| 终端输出转义控制字符 | codex-review-032 GO | #12 | `26c787b` | §38 ~ §40 |
| 人工确认与 JSON 输出分流 | codex-review-033 GO | #13 | `fbd6dff` | §41 ~ §42 |
| M1 退出证据的补充（6 项缺口） | codex-review-036 GO | #16 | `01bd50f` | §46 |
| 项目名称禁止代码围栏（M1 最终验收前的独立安全修复） | codex-review-037 GO | #17 | `5aad555` | §47 ~ §48 |

以上合并提交都在 master 历史中（已用 `git merge-base --is-ancestor` 逐一确认）。

## 3. 三平台 CI 与运行环境

**最新的 master CI**：[run 39019706495](https://github.com/XKHoshizora/specmetry/actions/runs/39019706495)，提交 `5aad555`，一次通过，8/8 成功。

| 平台 | git | Go 测试（顶层与子测试） |
|---|---|---|
| ubuntu-latest（带 `-race`） | 2.55.0 | 587 PASS，6 SKIP，0 FAIL |
| windows-latest | 2.56.0 | 583 PASS，3 SKIP，0 FAIL |
| macos-latest | 2.56.0 | 586 PASS，2 SKIP，0 FAIL |

此前各批的 master CI 也都是 8/8：`19600ec`（run 39000934831）、`01bd50f`（run 39012262429）。

跳过的测试，以及它们与 M1 的关系：

- `TestInitRefusesRealOldGit`、`TestARealGitThatFetches`：三个平台都跳过。它们需要环境中提供真实的旧版 git，CI 中按设计不提供（review-resolution §29）。已在本地用与 `5aad555` 相同的代码重跑，见第 7 节。
- ubuntu 的四个跨用户测试（`TestCrossUser*`）：属于 M0 的存储验收，在同一 job 中由单独的 root 步骤运行。
- windows 的 `TestSourceListUnreadableSource`：需要当前用户无法读取的文件，在 Windows 上跳过；它不是任何 M1 验收项的唯一证据。

除这些之外，本文列出的每个测试（共 100 个测试名，包括 PR #16、PR #17 的新测试）在三个平台上都是 PASS。

**本地运行**：在 PR #17 的分支上进行，代码与 master `5aad555` 的树 `13ee736` 相同。

- Windows（git 2.56.0）：
  - `scripts/check.sh` **没有完整通过**：`internal/cli` 触发 Go 默认的 10 分钟超时，此前没有测试失败；其余各项检查和其他包都通过。
  - `internal/cli` 单独以 `-timeout 30m` 运行，通过（1048 秒）。
  - web 检查与带 Dashboard 的构建通过。
- WSL Linux（git 2.43.0，`SPECMETRY_TEST_GIT_OBJECTS=safe`）：19 个测试二进制全部通过，其中 `cli` 147 项。
  - `TestBuiltBinary` 与 `TestShellCompletion*` 需要 go 工具链构建二进制，WSL 中没有，所以跳过；它们在 CI 的三个平台上运行。

**PR #14 的 CI**：[run 38957016420](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420)。

- attempt 1：Windows 的 `TestCheckWhileTheDatabaseIsCreated` 偶发失败（`Access is denied`），已登记为维护事项 M-1（第 9 节）；
- attempt 2：只重跑失败的 job，8/8 通过。

## 4. P0 验收项逐项结果

状态的含义：

- **PASS**：M1 各项要求均有适用的测试、独立审查或规定的其他验证证据；自动化测试注明对应提交和 CI 结果。例如 SRC-07 的"不解析文档语义"来自 codex-review-036 的独立调用链审查，不是 CI 中的测试；
- **INCOMPLETE**：有一条缺少验证证据（第 2 版中已没有这样的项；第 1 版中的 6 项如何关闭，在对应行中注明）；
- **DEFERRED**：只用于"M1 / M2"等项中属于后续里程碑的部分，并注明依据。

表中测试名省略了包路径，除特别注明外，都在 `internal/cli`。"CI"一列是最新 master CI（`5aad555`）上 ubuntu / windows / macos 的结果。三类集成证据（进程内、进程级、正式二进制）的定义与证明边界，见 acceptance-criteria §1（产品负责人已批准）。

### 4.1 项目初始化与 worktree（WP1.1、WP1.5）

| ID | 工作包 | 实现 | 测试 | M1 验证结果 | CI | 审查 | 状态 |
|---|---|---|---|---|---|---|---|
| MVP-INIT-01 | WP1.1 | `cli/init.go`；`project/init.go`；`project/files.go` `WriteMarker` | `TestInitRegistersProject` | 退出码 0；marker 中的 `project_id` 与输出一致，并且是合法的 ULID；`init_progress = initialized`；输出通过 `init.v1` 校验 | PPP | 017 GO | PASS |
| MVP-INIT-02 | WP1.1 | `project/init.go` `find`、`update` | `TestInitIsIdempotent` | `project.id` 不变，`created = false`，只有 1 个项目；`seq` 不变，marker 逐字节不变 | PPP | 017 | PASS |
| MVP-INIT-03 | WP1.1 | `project/rules.go` `DetectStage` | `TestInitDetectsStage`、`TestInitStageOverride` | 三个夹具的 `stage_detected` 分别正确；`--stage` 改变 `stage`，`stage_detected` 不变 | PPP | 017 | PASS |
| MVP-INIT-04 | WP1.1、WP1.4 | `project/rules.go` `Candidates` | `TestInitCandidateSources`、`TestInitRegistersNoSource` | 候选来源恰为 `[docs/client-brief.md, README.md]`，排除项不在其中；`source list` 为空，并通过 `source-list.v1` 校验 | PPP | 017、019 | PASS |
| MVP-INIT-11 | WP1.1（无法读取的提交一项随 WP1.5 加入） | `project/init.go` `Prepare`、`snapshotOf` | `TestInitDefaultTarget`、`TestInitRefusesACommitItCannotRead` | `ref = refs/heads/main`；分离 HEAD 时 `target = null`，并且只有 `detached_head_no_target` 一个警告；在关联 worktree 中首次执行，目标仍是主分支；没有提交时 `snapshot = null`；提交无法读取时退出码 6 | PPP | 017、024 | PASS（M1）；`NO_TARGET` DEFERRED（M2，见第 5 节） |
| MVP-INIT-12 | WP1.5（PD-26） | `gitx/objects.go` 能力探测与拒绝；`project/init.go` 在列出文件前检查 | 替身：`TestInitRefusesAGitThatFetches`、`TestInitWithAGitThatNeverFetches`、`TestInitRefusesACommitItCannotRead`；`gitx` 的 `TestAGitThatFetchesReadsNoObject`、`TestProbe*`、`TestListFiles*`；真实 git：`TestARealGitThatFetches` | 替身测试断言：退出码 6；数据目录和仓库（包括修改时间）都不变；不运行读取对象的命令；临时目录被删除。真实 git 的结果见第 7 节：2.32.0 判为会抓取，公开命令行为正确；2.43、2.56 判为不会抓取，全部测试通过 | PPP（替身）；真实 git 在本地 | 020 ~ 024 | PASS |
| MVP-INIT-13 | WP1.5（TD-26） | `gitx` `ObjectFormat`；`project/init.go` `Prepare` | `TestInitRefusesSHA256Repositories`；`gitx` `TestObjectFormat` | 3 个真实 SHA-256 仓库都是退出码 6：不创建数据目录；已有 3 个项目时，数据目录、`seq` 与项目数都不变；仓库逐文件不变；SHA-1 对照正常 | PPP | 023、024 | PASS |
| MVP-WT-01 | WP1.1 | `project/init.go` `find`（按 git 公共目录）、`ensureWorktree` | `TestInitRegistersLinkedWorktree` | 用 `git worktree add` 新建 worktree 并在其中执行：`created = false`，`project.id` 相同；数据库中有 2 个不同 ID 的 worktree，关联的那个 `is_main = 0`，`git_dir` 不同；再次执行不再新增 | PPP | 017 | PASS（M1）；`status` 的 `workspace.worktree_id` DEFERRED（M2） |

### 4.2 来源（WP1.4）

| ID | 工作包 | 实现 | 测试 | M1 验证结果 | CI | 审查 | 状态 |
|---|---|---|---|---|---|---|---|
| MVP-SRC-01 | WP1.4 | `cli/source.go`；`source/source.go` `Resolve`、`HashFile`；`source/registry.go` `Register` | `TestSourceAddRegisters` | 仓库内的文件，以及一个仓库外、路径含空格和中文的文件，都登记了路径、类型、位置、`acked_sha256`（由测试独立计算）和大小 | PPP | 019 | PASS |
| MVP-SRC-02 | WP1.4 | `registry.go` `Observe` | `TestSourceListObservesTheDisk` | 修改后为 `changed`，`current_sha256` 是新哈希；删除后为 `missing`，`current_sha256 = null`；通过 `source-list.v1` 校验 | PPP | 019 | PASS |
| MVP-SRC-03 | WP1.4 | `source ack`、`ackPrecondition`；`registry.go` `Acknowledge` | `TestSourceAckAndRemoveRefuseTheAgent`、`TestSourceAckInTerminal`、`TestSourceAckStaleObservedOnUnchangedSource` | 非 TTY 退出码 3；TTY 中 `observed_sha256` 过期（包括确认期间文件变化）退出码 5；一致时写入 `source_acks`（含理由与 `human_cli`），`acked_sha256` 更新，来源集合哈希改变 | PPP | 019 | PASS |
| MVP-SRC-04 | WP1.4、WP1.5 | `source remove`；`registry.go` `Remove`、`Refs` | `TestSourceAckAndRemoveRefuseTheAgent`、`TestSourceHumanOnlyCommandsRefuseRedirection`、`TestSourceRemoveInTerminal`、`TestSourceListRefs` | 非 TTY 退出码 3；TTY 中写入 `removed_seq`，来源集合哈希改变；引用保留，`source list --include-removed --refs` 中列在已移除的来源下 | PPP | 019、024 | PASS（M1）；`status` 中的 `source_removed` DEFERRED（M2） |
| MVP-SRC-05 | WP1.4 | `source.Resolve` | `TestSourceAddRefuses`、`source` `TestResolveRefuses` | 退出码 2，不写入，`seq` 不变 | PPP | 019 | PASS |
| MVP-SRC-06 | WP1.4（PD-25） | `source.SetHash`、`registry.go` `ProjectSetHash` | `source` `TestSetHash*`；CLI 测试中 `sourceList` 独立重算哈希 | 顺序无关、移除后重新登记会改变哈希：单元测试；确认改变哈希、未确认的文件变化不改变哈希：单元与集成测试 | PPP | 019 | PASS |
| MVP-SRC-07（M1 / M2） | WP1.4 | 文档类型白名单（V-S01，只看扩展名）；再次登记已登记的来源时原样返回 | `TestSourceAddRefuses`、`TestSourceAddAnyTypeInTerminal`、`TestSourceAddAgainChangesNothing`、`TestSourceAddAgainOfApprovedNonDocument`；隐私：`TestSourceContentIsNeverStored`、`TestProposeNeverStoresExcerpts`、`TestReviewNeverStoresExcerpts` | 非文档类型退出码 2；TTY 中加 `--allow-any-type` 成功；Agent 再次登记时 `created = false`，`acked_sha256` 不变，状态仍为 `changed`，`seq` 不变。"只为引用校验读取文本、不解析文档语义"：codex-review-036 按实际的读取、校验、持久化与输出调用链独立审查，结论 **PASS**（`source add`、`HashFile`、`Observe`、`CheckExcerpts`、`ListFiles`/`DetectStage`、`hash_only` 约束）。第 1 版为 INCOMPLETE（缺审查记录），由审查 036 关闭 | PPP | 019、036 | PASS（M1）；`SOURCE_CHANGED` 差距 DEFERRED（M2，见第 5 节） |
| MVP-SRC-08 | WP1.4、WP1.5 | `source remove --erase-excerpts`；`registry.go` `Refs` | `TestSourceListRefs`、`TestSourceRemoveInTerminal`、`TestSourceAckAndRemoveRefuseTheAgent` | `hash_only` 项目在 TTY 中执行：来源被移除；每条引用的 `locator` 与 `excerpt_sha256` 保留；`excerpt` 在 JSON 和数据库中都为 NULL；`excerpt_state = not_stored`。非 TTY 时退出码 3，`seq` 不变 | PPP | 019、024 | PASS（M1）；`full` 模式与擦除后的整理 DEFERRED（WP2.11，M2） |

### 4.3 提案、审核与问题（WP1.3、WP1.5、WP1.6、WP1.7）

| ID | 工作包 | 测试 | M1 验证结果 | CI | 审查 | 状态 |
|---|---|---|---|---|---|---|
| MVP-BL-01 | WP1.5 | `TestProposeRecordedProposal` | 录制的提案：18 个条目都为 `pending`，4 个问题都为 `open`，一个 `seq`，没有实体或版本 | PPP | 024 | PASS（M1）；差距 DEFERRED（M2） |
| MVP-BL-02 | WP1.5 | `TestProposeRefuses` | V-P01、V-P03 ~ V-P06、V-P08 都有反例：退出码 2，带 JSON Pointer，行数与 `seq` 不变 | PPP | 024 | PASS |
| MVP-BL-03 | WP1.5 | `TestProposeStale` | `base_version` 过期：退出码 5，`current.base_version`；`target_revision` 过期：退出码 5，带当前的 revision；都不写入 | PPP | 024 | PASS |
| MVP-BL-04 | WP1.5 | `proposal` `TestCheckExcerpts`；`TestProposeChecksExcerpts`、`TestProposeRefuses` | `verified`、`not_found`、`not_checkable`、`source_missing`，以及超过 500 字符退出码 2：单元与 CLI 测试都有断言 | PPP | 024 | PASS |
| MVP-BL-05 | WP1.5 | `TestProposeRefuses`、`TestProposeChecksCodeRefs` | 反推条目没有引用：退出码 2；有引用时保存为 `inferred_from_code` | PPP | 024 | PASS（M1）；`status` 标记 DEFERRED（M2），UI DEFERRED（M5） |
| MVP-BL-06 | WP1.6 | `TestReviewAcceptsSomeItems` | 接受 3 个、拒绝 2 个：生成版本 1，恰好包含这 3 个，按顺序编号 | PPP | 026 | PASS |
| MVP-BL-07 | WP1.6 | `TestReviewNeedsTheParentAccepted` | 退出码 2，指向 `/items/1`，不写入 | PPP | 026 | PASS |
| MVP-BL-08 | WP1.6 | `TestReviewAcceptsWithEdits` | 同时保存建议值与最终值，基准使用最终值 | PPP | 026 | PASS |
| MVP-BL-09 | WP1.6 | `TestReviewInTerminal`（真实 PTY） | 非 TTY、stdout 重定向、设置 Agent 变量：退出码 3，不写入 | PPP | 026 | PASS |
| MVP-BL-10 | WP1.6 | `TestReviewRefusesAStaleVersion`、`TestReviewIsStaleBeforeTheConflict`、`TestReviewIsStaleAfterTheConfirmation` | 退出码 5，指向 `/seen_base_version`，不生成版本 | PPP | 025、026 | PASS |
| MVP-BL-11 | WP1.6 | `TestReviewRetiresCriteriaWithTheirRequirement` | 3 个版本，通过 `baseline-history.v1` 校验；逐条列出变更，以及前后状态、actor 和通道 | PPP | 026 | PASS |
| MVP-BL-12 | WP1.6 | `TestReviewChangeClasses` | `statement` 的 `material` 加 1，`editorial` 不变；检查的命令、工作目录、shell、`inputs` 单独修改时都强制按 `material` 记录 | PPP | 026 | PASS |
| MVP-BL-13 | WP1.5、WP1.6 | `TestProposeApply`、`TestProposeHumanEdit` | 人工编辑在同一事务中应用（提案与版本的 `seq` 相同）；Agent 提交 `human_edit` 退出码 3，不写入 | PPP | 024、026 | PASS |
| MVP-BL-14 | WP1.6 | `TestReviewConflicts`；`contract` SCH-82 ~ 84、126 ~ 129 | 缺少 `acknowledged_conflict`：退出码 2；三方对比（包括 revision 相同的 `editorial` 冲突）；按类型的 schema 条件约束与反例 | PPP | 026 | PASS |
| MVP-BL-15 | WP1.6 | `TestReviewNeverReusesKeys` | 退役的 REQ-003 不再复用；AC 按需求编号 | PPP | 026 | PASS |
| MVP-BL-16 | WP1.6 | `TestReviewRetiresCriteriaWithTheirRequirement` | 退役需求时，在同一版本中级联退役 2 个 AC，并各有变更记录 | PPP | 026 | PASS |
| MVP-BL-17 | WP1.7 | `TestQuestionNeedsAPerson`、`TestQuestionInTerminal`、`TestQuestionAnswerAndDismiss`；SCH-131 ~ 139 | Agent 退出码 3，问题仍为 `open`；TTY 中回答与驳回被记录；`question list --all --json` 显示回答者和通道，并与数据库一致；schema 条件约束与反例 | PPP | 028 | PASS |
| MVP-BL-19 | WP1.6 | `TestReviewApplicationOrder` | `reaffirm`、`retire`、`defer` 在一次审核中生效；`defer` 的 AC 保持不一致 | PPP | 026 | PASS |
| MVP-BL-20 | WP1.6 | `TestReviewApplicationOrder` | 接受 Agent 的 `reaffirm`：`requirement_revision` 更新为父需求的 revision，AC 自身的 revision 不变 | PPP | 026 | PASS |
| MVP-BL-21 | WP1.5 | `TestProposeRefuses`、`TestProposeAcceptsAChecked`；`TestProposeSecretInACheckNamesAnEnvironmentVariable`（PR #16） | 没有 `rationale`、`inputs` 或 `workdir` 越出仓库：退出码 2；使用环境变量的命令被接受；命令中有密钥：退出码 2，指向 `/items/2/data/check/command` 与 `secret_detected: github_token`，JSON 与文本中都提示"reference an environment variable by name instead"，error.v1 合法，不泄露密钥，不写入。第 1 版为 INCOMPLETE（缺提示的断言），由 PR #16 关闭 | PPP | 024、036 | PASS |
| MVP-BL-22 | WP1.6 | `TestReviewCarriesReferences` | `material` 更新不提供新引用时，沿用上一修订的引用，重新校验后写成新行，旧行保留 | PPP | 026 | PASS |
| MVP-BL-23 | WP1.5、WP1.6 | `TestProposeApply`、`TestProposeRefuses` | `update` 条目缺少 `change_class`：退出码 2，不写入；代码围栏、多行标题：退出码 2 | PPP | 024、026 | PASS |
| MVP-BL-24 | WP1.6 | `TestReviewApplicationOrder` | 同一提案中 `reaffirm` 的、新增的和修改的 AC，都取父需求的新 revision | PPP | 026 | PASS |
| MVP-BL-25 | WP1.6、WP1.7 | `TestReviewRelatesQuestions`、`TestQuestionAnsweredBeforeItsItem` | 条目被接受后，问题关联到新编号；被拒绝的条目，问题仍指向原引用 | PPP | 026、028 | PASS（M1）；"不再计为阻塞" DEFERRED（M2） |
| MVP-BL-27 | WP1.6 | `TestReviewChangeClasses`；`TestReviewConflictItemNeedsAChangeClass`（PR #16） | `accepted` 与 `accepted_with_edits` 缺少 `change_class`：退出码 2，指向 `/items/<i>/change_class`，不写入；不采用提案建议的类别；强制字段按 `material` 记录。冲突条目：先断言条目确为 `conflict`、基准为 v2，带 `acknowledged_conflict` 而缺少 `change_class` 时两种接受方式都是退出码 2（`/items/0/change_class`），不分配 `seq`；带 `change_class` 的对照应用为 v3。第 1 版为 INCOMPLETE（缺冲突条目的测试），由 PR #16 关闭 | PPP | 026、036 | PASS |
| MVP-BL-28 | WP1.5 | `TestProposeRecordedProposal`、`TestProposeChecksCodeRefs`、`TestProposeNeverStoresExcerpts`、`TestProposeApply`；SCH-79、113 ~ 125 | 输出通过 `proposal-receipt.v1` 校验；每个条目有 `item_id` 与片段校验结果，代码引用有 `exists`；输出、错误和数据目录中都找不到片段原文 canary；条件约束与反例；回执 ID 与数据库一致 | PPP | 024 | PASS |
| MVP-BL-29 | WP1.5（PD-26） | `TestProposeAndSourcesWithAGitThatFetches`、`TestProposeRefusesPartialCloneByAnyMeans`（替身）；`TestARealGitThatFetches`（真实 git 2.32.0） | 带代码引用的 `propose`：退出码 6，不写入，不分配 `seq`，`.git` 不变；不带代码引用的 `propose` 和 `source` 命令照常工作。真实 git 2.32.0 的检查已在本次重跑通过（第 7 节） | PPP（替身）；真实 git 在本地 | 022 ~ 024 | PASS |
| MVP-PRV-07 | WP1.3，以及各写入工作包 | `secrets` `TestBuiltinRuleSamples`、`TestRejectionNeverEchoesTheMatchedText` 等；`config` `TestLoadRejectsInvalidExtraRuleWithoutEchoingIt`；各命令的 CLI 测试 | 已验证：每条内置规则至少 2 个命中、2 个不命中样本，运行时拼接；环境变量引用不命中；只给出 Pointer 与规则编号；边界反例 ①②③；无效的追加规则退出码 6，不回显。CLI 拒绝写入并且不写入：`init`、`propose`、`source`、`question`、`target set`、`scope confirm` 都有测试；`review`：`TestReviewRefusesASecretAnywhereInTheDocument`（PR #16），密钥位于 `/summary`、`/items/0/note`、嵌套的 `/items/1/final_data/description` 与 `/scope/note`，都在确认之前以退出码 2 拒绝，指向确切字段与规则，不回显，不写入，数据目录中没有。第 1 版为 INCOMPLETE（缺 `review` 的测试），由 PR #16 关闭 | PPP | 011、024、026、028、035、036 | PASS |
| MVP-PRV-08 | WP1.5、WP1.6 | `TestProposeNeverStoresExcerpts`、`TestReviewNeverStoresExcerpts` 等 | 新项目默认 `hash_only`；`excerpt` 为 NULL；`excerpt_sha256` 与校验结果照常写入；数据目录的字节中找不到片段原文 | PPP | 024、026 | PASS |
| CX-14 | WP1.5 | `TestProposeStale` | Agent 以 `base_version = 3` 提交，而当前为 v4：退出码 5，`current.base_version = 4`，不写入。注：v1 ~ v4 由 SQL 种子构造，没有通过真实的 `review` 生成；被测的比较逻辑与版本如何产生无关 | PPP | 024 | PASS |
| CX-17 | WP1.4 | `TestSourceAckAndRemoveRefuseTheAgent`、`TestSourceHumanOnlyCommandsRefuseAgentInTerminal` | Agent 通道执行 `source ack`：退出码 3；`seq` 不变，来源仍为 `changed`，`acked_sha256` 不变；伪终端中设置 Agent 变量时同样拒绝 | PPP | 019 | PASS（CLI 部分）；`SOURCE_CHANGED` 差距与"不是 ready" DEFERRED（M2） |

### 4.4 交付目标与范围、通道识别（WP1.2、WP1.8）

| ID | 工作包 | 实现 | 测试 | M1 验证结果 | CI | 审查 | 状态 |
|---|---|---|---|---|---|---|---|
| MVP-TGT-01 | WP1.8 | `cli/target.go` `newTargetSetCmd`；`scope/target.go` `SetTarget` | `TestTargetShowAndSet`、`TestTargetAndScopeNeedAPerson` | 非 TTY（管道、stdout 重定向）退出码 3，不写入；TTY 中旧目标被归档、新目标被创建，两者使用同一个 `seq`，操作者为人，通道为 `human_cli` | PPP | 034、035 | PASS（M1）；`NO_SCOPE` DEFERRED（M2） |
| MVP-TGT-02 | WP1.8 | `scope.RefKind`；`gitx.ValidRefName` | `TestTargetShowAndSet` | `refs/remotes/origin/main` 退出码 2，指向 `--ref`，没有提示，`seq` 不变；短名称与不合法的 ref 同样拒绝 | PPP | 034 | PASS |
| MVP-TGT-03 | WP1.8 | `scope confirm` 首先调用 `RequireHuman` | `TestScopeConfirm`、`TestTargetAndScopeNeedAPerson` | 非 TTY（真实子进程的管道；伪终端中 stdout 重定向；带与不带 `--json`）退出码 3，没有提示，不写入 | PPP | 034 | PASS |
| MVP-TGT-04 | WP1.8 | `scopeInput`；`scope.v1` 的 `minItems`；`scope.Check` | `TestScopeConfirm`、`TestTargetWithoutOne` | `--requirements ","`、需求为空的 `--file`：退出码 2，指向 `/requirements`，不写入 | PPP | 034 | PASS |
| MVP-TGT-05 | WP1.8 | `scope.Check`；写事务中重新检查 | `TestScopeConfirm`、`TestScopeInReview` | 版本或来源集合哈希过期：退出码 5，`error.current` 给出当前值，不写入，`seq` 不变 | PPP | 034 | PASS |
| MVP-TGT-06 | WP1.8 | 待审数 = `reviewed_seq IS NULL` 的条目数（包括 `conflict`） | `TestScopeConfirm`、`TestScopeInReview` | 不一致：退出码 5；附带在审核中时以审核之后的数量为准；`incomplete_docs` 缺少声明或声明为 false：退出码 2；带声明时写入并保存。注：测试项目中排除的 2 个条目有一个处于冲突，但测试没有单独断言它的 `conflict` 状态 | PPP | 034 | PASS |
| MVP-TGT-07 | WP1.8 | `cli/review.go`；`review/scope.go` `ConfirmScope` | `TestScopeInReview` | 审核接受 3 个新增需求，范围用它们的 `item_id` 表示：解析为新编号，绑定版本 1，与版本使用同一个 `seq`。范围引用被拒绝、未处理或不是新增需求的条目，或者范围的任一前置条件不满足：整次审核失败，版本、实体、编号、范围与 `seq` 都不变 | PPP | 034（另行用触发器验证了晚期失败的全部回滚）、035 | PASS |
| MVP-TGT-08 | WP1.8 | `scopeInput`；`State.MustRequirements` | `TestScopeConfirm` | 默认只选中 `must` 需求（REQ-001、REQ-003，不含 should 的 REQ-002）；`--requirements` 改为指定的需求 | PPP | 034 | PASS |
| MVP-TGT-10 | WP1.8 | `scope.State.WaiversUnder`；`scope.Check` 的集合比较 | 真实 CLI：`TestTargetShowAndSet`、`TestScopeConfirm`、`TestScopeInReview`；单元：`scope` `TestCheckWaivers` | 真实 CLI：`target-scope.v1` 的 `effective_waivers` 为 `[]`；多列一项：退出码 5，`error.current.effective_waivers = []`，不写入；一致时写入，`acknowledged_waivers_json = "[]"`。单元测试用合成的非空集合验证少一项、多一项、已失效和顺序无关（集合为空时无法在 CLI 中构造"少一项"）。注：审核路径没有断言 `error.current` | PPP | 034（V-T01 通过）、035 | PASS（M1）；真实豁免 DEFERRED（M2） |
| MVP-TGT-11 | WP1.8 | 写事务以 `BEGIN IMMEDIATE` 开始，先取得写锁再重新读取和检查 | `TestScopeConfirmedMeanwhile`、`TestScopeInReview`；`scope` `TestConfirmRace`（两个独立连接） | 会话甲以 C1 确认、会话乙已写入 C2：退出码 5，不写入，`error.current` 给出 C2；首次确认为 null 而已有确认：退出码 5；重新读取后以 C2 确认：成功；同进程另一连接、伪终端中另一进程在确认期间写入：退出码 5。审核中：已有确认时为 null 返回退出码 5（current 为 C1），重新读取后成功。注：审核路径没有字面的"C1 对 C2"反例；确认期间过期的两个用例没有断言 `seq` | PPP | 034（另行用两个同时等待的 CLI 会话验证）、035 | PASS |
| MVP-CH-01 | WP1.2、1.4、1.6、1.7、1.8；PR #16 | `channel.Detect`（`executeRoot` 中每次执行识别一次）；每个人类专属命令的执行体首先调用 `RequireHuman`；`confirm` 中的 `channel.Guard` | 见下文矩阵；PR #16：`TestAgentVariablesRefuseTheRemainingHumanOnlyCommands`（进程级）、`TestBuiltBinary`（正式二进制） | 全部 9 个人类专属命令：伪终端中设置 Agent 变量时退出码 3，没有提示，不写入，canary 不在输出和数据目录中；不设变量时显示确认，拒绝后不写入。`propose --apply`、`question dismiss`：两个变量、带与不带 `--json`，进程内另验证读取次数为 0；正式二进制对这两个命令验证了同样的边界。`question answer` 的仓库测试只设置了 `CLAUDECODE`。两个变量经同一个识别循环产生同一个 Detection，随后执行相同的防护；codex-review-038 另用正式二进制独立补验了 `CLAUDE_CODE_ENTRYPOINT`（拒绝，canary 不泄露；无变量、正确确认的对照成功）。补验在 Windows 上，没有进入仓库，也没有在三平台 CI 中运行。第 1 版为 INCOMPLETE（缺这两个命令的伪终端测试），由 PR #16 关闭 | PPP | 014、019、026、028、035、036、038 | PASS |
| MVP-CH-02 | 同上 | 同上 | 见下文矩阵；PR #16：`TestRedirectionRefusesTheRemainingHumanOnlyCommands`（进程级）、`TestBuiltBinary`（正式二进制） | 全部 9 个人类专属命令：stdin 为终端、stdout 重定向时退出码 3，没有提示。`source ack`、`source add --allow-any-type`、`propose --apply`、`question dismiss`：stdout 写入文件并读回，带 `--json` 时文件恰好是一份 error.v1，不带时原因在终端、文件为空；stdin 不是终端（真实子进程）同样退出码 3；不写入。正式二进制对 `question dismiss` 验证了重定向与管道。`review`、`question answer` 的"stdin 不是终端"：按审查 038 的裁定，由证据链成立，见第 8 节第 3 条。第 1 版为 INCOMPLETE（缺这 4 个命令的重定向测试），由 PR #16 关闭 | PPP | 同上 | PASS |

**CH-01 / CH-02 的命令 × 情形矩阵**（伪终端测试在三个平台都运行，没有跳过条件。"PR #16"表示由 PR #16 补上的格子）：

| 命令 | 伪终端 + Agent 变量 | 伪终端中拒绝确认 | stdout 重定向 | stdin 不是终端 | 带 `--json` |
|---|---|---|---|---|---|
| `review` | 两个变量 | 有 | 有（带 `--json`） | 进程内 | 有 |
| `propose --apply` | 两个变量（PR #16；正式二进制） | 有（PR #16） | 有，带与不带 `--json`（PR #16） | 真实子进程（PR #16） | 有 |
| `source ack` | 两个变量 | 有 | 有，带与不带 `--json`（PR #16） | 真实子进程（PR #16） | 有 |
| `source remove` | 两个变量 | 只断言退出码 3 | 有 | 真实子进程 | 有 |
| `source add --allow-any-type` | 两个变量 | 有 | 有，带与不带 `--json`（PR #16） | 真实子进程（PR #16） | 有 |
| `question answer` | `CLAUDECODE` | 有 | 有 | 进程内 | 有 |
| `question dismiss` | 两个变量（PR #16；正式二进制） | 有 | 有，带与不带 `--json`（PR #16；正式二进制） | 真实子进程（PR #16；正式二进制） | 有 |
| `target set` | 两个变量 | 有 | 有 | 真实子进程 | 有 |
| `scope confirm` | 两个变量 | 有 | 有 | 真实子进程 | 有 |

- 共同机制的证据：
  - WP1.2 的替身命令在真实伪终端中验证了两个变量、管道（带与不带 `--json`）和 stdout 重定向（`TestHumanOnlyCommand*`）；
  - `TestConfirmationWithJSON` 对全部 9 个确认，在进程内验证 Agent 通道退出码 3、没有提示、不写入；
  - `TestHumanOnlyCommandsConfirmThroughConfirm` 约束所有命令都通过同一个确认入口。
- 9 个人类专属入口都在执行体开头调用 `RequireHuman`。
- PR #16 的新测试能发现回归：在临时副本中让通道识别忽略 Agent 变量或 stdout 重定向，新测试和 `TestBuiltBinary` 都会失败，进程停在确认提示上等待输入（作者提供的辅助证据，见 PR #16）。
- 只读命令"在任何通道中都可以执行"：`version`、`doctor` 在 `human_cli` 中验证；各读取命令在 `agent_cli` 中都有测试。

## 5. M1 与 M2 的边界（DEFERRED 项及依据）

下列内容不是 M1 的退出条件，没有判为失败。依据是 implementation-plan §5 M1 的退出条件："标注为'M1 / M2'的项，在 M1 中只验证写入侧……推导侧（差距、交付状态）在 M2 中验证"。M2 的退出条件（§5 M2）明确包括"M1 / M2 项的推导侧"。

| 推迟的内容 | 涉及的验收项 | 依据 |
|---|---|---|
| 差距与交付状态的推导：`NO_TARGET`、`NO_SCOPE`、`NO_BASELINE`、`PENDING_REVIEW`、`SCOPE_STALE`、`SOURCE_CHANGED`、`source_removed`，以及"交付状态不是 ready" | INIT-11、TGT-01、BL-01、SRC-04、CX-17；TGT-09 整项为 M2 | 各行的"（M2）"标注；M1 退出条件；WP2.1 |
| `status --json` 中的标记与工作区 | BL-05 的 `inferred` 标记、WT-01 的 `workspace.worktree_id` | 各行的"（M2）"标注 |
| 问题被拒绝后不再计为阻塞 | BL-25 | 该行的"（M2）"标注 |
| 真实验收决定的豁免生命周期 | TGT-10 的 M2 部分 | 该行："M2 用真实的豁免验证……" |
| `full` 片段模式与擦除后的整理 | SRC-08 的 M2 部分 | 该行的 **M2** 部分；implementation-plan WP1.4、WP2.11 |
| UI 标识 | BL-05 的 M5 部分 | 该行的"（M5）"；M5 退出条件中的"BL-05 的 UI 部分" |
| 依赖事实推导的 `init_progress` | INIT-07（M2） | 该行的里程碑为 M2 |
| Check Run、Claim、证据等级、验收决定的完整推导 | CLM、EV、ACC、ST 各项 | 都标注为 M2 或之后 |

| `SOURCE_CHANGED` 阻塞差距的推导及其对交付状态的影响 | SRC-07 的 M2 部分 | 该行现标为"M1 / M2"（PR #16，产品负责人批准），与 WP1.4 验收时的记录一致（review-resolution §24，审查 019）；不是新的推迟 |

第 1 版指出的 SRC-07 标注不一致已经解决：该行的 M1 部分为非文档来源的拒绝与人工例外、重复登记不改变哈希且状态仍为 `changed`，以及"不解析文档语义"的独立审查；M2 部分为 `SOURCE_CHANGED` 差距的推导及其对交付状态的影响。

M1 应完成的写入校验、数据模型和 JSON 读取契约，都没有被推迟到 M2：以上推迟的都是推导、状态或界面。

## 6. 横向核查

**Schema 一致性**

- 生产代码：M1 的 11 个输出，`init.v1`、`source-add.v1`、`source-list.v1`、`proposal-receipt.v1`、`proposal-list.v1`、`proposal-show.v1`、`baseline-show.v1`、`baseline-history.v1`、`question-list.v1`、`target-scope.v1`、`error.v1`，都经过 `writeJSON` 先按 schema 校验再输出（`internal/cli/output.go`）。
- 输入的校验位置：
  - `proposal.v1`：`propose.go`；
  - `review.v1`：`review.go`，`final_data` 再按 `proposal.v1` 的条目定义校验；
  - `scope.v1`：`--file` 与交互式都校验（审查 034 B1 修复后）；
  - `source-ack.v1`：参数与 `--file` 都校验；
  - `question-answer.v1`：`question.go`。
- 测试：每个成功输出都有按具体 schema 的校验（`specmetry` 助手、`proposeOK`、`showProposal`、`baselineShow`、`questionList`、`targetScope` 等）。
- 例外是 codex-review-029 N1，仍未关闭：`cmdline_test.go` 中把 `--json` 写在命令之前的几种写法，只检查以 `{` 开头（维护事项 M-7）。
- 反例共 150 个（SCH-1 ~ 150）。实现输出接口时要补齐的条件约束（implementation-plan"输出契约的语义约束"）：
  - `proposal-receipt.v1`：SCH-113 ~ 125（BL-28）；
  - `proposal-show.v1`：SCH-126 ~ 129（BL-14）；
  - `question-list.v1`：SCH-131 ~ 136（BL-17）；
  - `target-scope.v1`：SCH-140 ~ 147。

  这三处已知缺口都已补齐；implementation-plan 中的"已知缺口"一句已在本版同步更新（维护事项 M-11）。
- PR #17 新增 SCH-150（`init.v1` 的 `project.name` 禁止代码围栏），反例共 150 个。
- `error.v1` 没有反例，`current` 也没有与 `stale_precondition` 关联的条件约束。M1 没有要求这一项，记录在此。
- 输出一致性测试，验证 schema 表达不了的关系：
  - `target-scope.v1`：两个命令的输出相同，ID 互相一致；
  - `source-list.v1`：来源集合哈希独立重算，状态与哈希一致；
  - `question-list.v1`：`proposal_id` 与 `created_seq` 与回执一致；
  - `proposal-show.v1`：条目状态与数据库一致；
  - `proposal-receipt.v1`：来源 ID、条目 ID 与数据库计数一致。

**密钥扫描**（architecture §10.8）

| 写命令 | 扫描的内容 | CLI 测试 |
|---|---|---|
| `init` | 项目名、要保存的路径、默认目标的 ref（项目名另禁止代码围栏，PR #17） | 有 |
| `source add` | 路径（解析前后） | 有 |
| `source ack` | 理由与来源参数 | 有（理由） |
| `source remove` | 来源参数 | 无（参数含密钥的情形） |
| `propose`、`propose --apply` | 整份文档与 worktree 路径 | 有（`--apply` 与 `propose` 共用同一段代码） |
| `review` | 整份文档（`review.go:71`） | 有（`TestReviewRefusesASecretAnywhereInTheDocument`，PR #16；含嵌套字段） |
| `question answer`、`dismiss` | 编号、回答、理由 | 有 |
| `target set` | ref 与名称，在任何回显之前 | 有 |
| `scope confirm` | `--file` 的全部字符串，或 `--requirements` 与 `-m` | 有（`-m`、`--file`） |
| 所有人类专属写入的操作者 | `user.display_name` | 无 |

`--file` 的路径本身不扫描，但它既不回显，也不保存（`readInput` 的错误不含路径，有测试）。

**`seq` 的原子性**

- 人类专属写入在一个写事务中只取一个 `seq`：`target set`、`scope confirm`、`review`（与它附带的范围共用）、`propose --apply`（提案与版本共用）、`question`、`source`。都有测试断言。
- `init` 为每个 worktree 行和默认目标各取一个 `seq`；从新的 worktree 提交提案时，登记 worktree 和保存提案各取一个。data-model 没有要求一个事务只取一个 `seq`，所以不是缺陷。
- 拒绝时不分配 `seq`，由以下比较 `seq` 计数器的助手断言：`humanRefused`/`scopeState`、`reviewRefused`/`state`、`proposeRefused`、`closeState`、`confirmState`、`seq()`；以及存储层的 `TestSeqIsUniqueAcrossConnections`、`TestSeqIsUniqueAcrossProcesses`。例外：确认期间过期的两个 `scope confirm` 用例没有断言 `seq`。

**适用于 M1 的不变量**（acceptance-criteria §3。自动断言在 E2E 中，属于 M2 之后）

| 不变量 | M1 已有的证据 | 留给 E2E 断言的部分 |
|---|---|---|
| 1. 没有 Agent 通道写入的范围确认、来源确认、基准版本或问题回答 | 数据库 CHECK：`scope_confirmations` 与 `baseline_versions` 只接受人的操作者和通道，`source_acks` 有对应约束，都有测试；每个命令有 CLI 拒绝测试。`questions.answered_channel` 有 CHECK，但没有数据库层的测试 | 每个 E2E 结束时的全量检查。注：`delivery_targets.channel` 在数据库层允许 `agent_cli`，`target set` 的人类专属只由 CLI 保证；不变量 1 没有列出交付目标 |
| 4. 数据目录中没有 C0 canary | 通道、密钥、来源内容、备注与名称等测试中的 `FindInTree` 检查 | PRV-02 的综合 canary 测试（M6） |
| 5. 只按 architecture §10.6 写入仓库 | `init` 的测试；`propose` 与 `source` 在 `.git` 不变的测试 | `review`、`question`、`target`、`scope` 的同类断言 |
| 6. 输入输出通过 schema 校验 | 见上文 | 每个 E2E 的断言 |
| 7. `seq` 严格递增 | STO-04 的测试；`seq` 列的唯一约束 | 打乱事实顺序后推导不变（M2） |
| 11. 范围确认的 `seen_scope_confirmation_id` 与豁免集合 | 首次为 NULL、第二次为 C1、5 次确认中 4 次带前一条、豁免集合为 `[]`：都有断言；写入使用写锁下读到的当前确认 | 对全部范围确认的检查；外键没有限定 `seen` 属于同一个目标 |

## 7. 真实 git 兼容性的重跑（PD-26、codex-review-016 B1）

INIT-12 要求"在真实 git 2.32.0 / 2.43 / 2.56 上逐项检查"，BL-29 要求"在真实 git 2.32.0 上逐项检查"。

- CI 中只运行替身测试，真实旧版 git 的测试按设计跳过。
- review-resolution 中记录的真实 git 结果，对应 WP1.5 中间一轮的提交（§27 中的表），最终一轮的结果在 PR #8 的描述中。
- 此后 WP1.6 ~ WP1.8 与 PR #17 改动过 `init`、`source` 与 `propose` 的周边代码。

因此在 WSL 中，用与 master 相同的代码（交叉编译的测试二进制）重跑了两次：第 1 版时用 `19600ec` 的代码，第 2 版时用 `5aad555` 的树 `13ee736` 的代码，结果相同：

| git | 方式 | 结果 |
|---|---|---|
| 2.32.0（官方源码构建，在 PATH 最前） | `SPECMETRY_TEST_GIT_OBJECTS=unsafe` | `TestProbeAgreesWithTheGitOnPath`：判定"会抓取"，与独立判据一致。`TestARealGitThatFetches`（公开命令）：git 项目的 `init` 退出码 6，不创建数据目录，仓库不变；非 git 项目的 `init`、不含代码引用的 `propose`，以及 `source add`、`list`、`ack`、`remove` 都照常工作。都通过 |
| 2.31.1（官方源码构建） | `SPECMETRY_TEST_GIT_2_31` | `TestInitRefusesRealOldGit`：版本检查即拒绝，不写出含 canary 的跟踪文件。通过 |
| 2.43.0（WSL） | `SPECMETRY_TEST_GIT_OBJECTS=safe`，全部测试二进制 | 全部通过；探测判定"不会抓取" |
| 2.56.0（Windows 本地；CI 的 windows、macOS） | 全部测试 | 全部通过 |

另外，在 2.32.0 下 `gitx` 包中有 3 个测试失败：`TestLocateIgnoresRepositoryVariables`、`TestHeadBranchAndCommit`、`TestTreeFilesAndLines`。

- 失败的原因都是 PD-26 设计好的拒绝："this git does not honour GIT_NO_LAZY_FETCH … so Specmetry reads no git objects with it"。
- 这 3 个测试是需要安全 git 的正向测试，不是产品缺陷。其他读取对象的测试在 `SPECMETRY_TEST_GIT_OBJECTS=unsafe` 时会跳过，这 3 个没有这个保护。登记为测试脚手架的小问题（第 9 节 M-9）。
- WP1.5 当时对 2.32.0 的验证范围同样是"探测与替身测试，以及公开命令"（§27）。

WP1.8 新增的命令也会解析 ref（`target show`、`scope show`、`target set`、`scope confirm`）。它们用的 `gitx.CommitOf` 走同一个对象读取保护（`runObjectsCode` → `objectGit`）；`gitx.ValidRefName` 不读取对象。这些命令没有专门的真实 git 2.32.0 测试，它们不在 INIT-12、BL-29 列出的命令之内。

## 8. 证据边界与尚缺少的实际验证

1. **第 1 版的 6 项缺口已全部关闭**：CH-01、CH-02、BL-21、BL-27、PRV-07 由 PR #16 的测试补齐，SRC-07 由审查 036 的独立审查确认（第 4 节）。没有修改业务实现。
2. **三类集成证据**：acceptance-criteria §1 现在区分进程内、进程级与正式二进制三类证据，产品负责人已正式批准（审查 036 记录了这次确认）。
   - 进程内：调用与 `cli.Execute` 相同的生产命令树，配合真实的临时 SQLite 与 git 仓库。证明业务行为，不证明终端检测、重定向、进程退出码与 `main` 的接线。
   - 进程级：测试二进制作为子进程。证明通道识别与进程边界，但不等同于正式发布的二进制。
   - 正式二进制：`go build -tags nowebui ./cmd/specmetry`（`TestBuiltBinary`、`TestShellCompletionIsRefused`）。证明 `main` 的接线、退出码、基本 JSON 读取与 CH-01/02 的关键边界，不证明发布包本身（内嵌 Dashboard 的构建由 CI 的"Build with dashboard"冒烟，打包在 M6）。
   - 规则：涉及 TTY、Agent 环境变量、重定向或进程退出码的验收，必须有进程级或正式二进制的证据。
3. **证据边界与 CH-01、CH-02 的证据链**（codex-review-038 的裁定，不改变状态）：
   - **CH-02：`review`、`question answer` 的"stdin 不是终端"，PASS**。证据链如下：
     1. `executeRoot`（`internal/cli/root.go`）在命令执行之前，按进程的 `os.Stdin`、`os.Stdout` 统一识别通道。
     2. `channel.Decide` 对不是终端的 stdin 一律给出 `agent_cli`。这一操作系统边界已有真实子进程的证据：WP1.2 的 `TestHumanOnlyCommandRefusesWithoutTerminal`，以及 PR #16 对其他命令的管道测试。
     3. `review`（`internal/cli/review.go`）与 `question answer`（`internal/cli/question.go`）都在读取输入、打开数据库和人工确认之前调用同一个 `RequireHuman`，没有命令专属的旁路。
     4. 这两个真实命令的 stdout 重定向有进程级证据。它们的进程内非终端测试调用生产的 `Execute`，没有注入通道，证明命令的接线与拒绝后的业务状态；进程内测试本身不代替真实进程的证据。
     5. `TestBuiltBinary` 验证了正式的 `main` 入口、真实的退出码和关键的通道边界。

     已批准的三类证据定义，不要求每个命令的每种组合都用正式二进制。
   - **CH-01：`question answer` 的两个 Agent 变量**。仓库测试只设置了 `CLAUDECODE`；两个变量经同一个识别循环产生同一个 Detection，随后执行相同的防护。依据是这一共同机制与审查方的补验，而不只是验收文字中的"或"。
   - **证据的来源要分开看**：
     - 仓库中的测试，在三平台 CI 中运行（第 3 节、第 4 节的"CI"一列）；
     - 作者在本地的运行，包括 Windows、WSL 与真实旧版 git 的重跑（第 3 节、第 7 节）。审查 038 没有独立重跑真实旧版 git，这些仍是作者提供的本地证据；
     - 审查方的独立验证：
       - 审查 036 的 SRC-07 调用链审查；
       - 审查 037 的逐表比较；
       - 审查 038 在 Windows 上用正式二进制直接补验 `review`、`question answer` 的真实 stdin 拒绝（stdin 接到保持写端打开、没有数据的管道，stdout 为 ConPTY；文本与 JSON 两种模式都是退出码 3，没有提示，业务状态不变，没有阻塞），以及 `question answer` 的 `CLAUDE_CODE_ENTRYPOINT`。

       这些独立验证没有进入仓库，也没有在三平台 CI 中运行。
   - `TestBuiltBinary` 用 `nowebui` 构建，验证生产入口，不验证发布包。
   - PR #16 的变异检验是作者提供的辅助证据，审查 036 没有独立重跑。
4. **测试强度不足的地方**（不影响状态，供后续改进）：
   - BL-12 只在新增检查时一起修改 `verification`，没有单独修改；
   - BL-05 没有断言被接受的实体保留 `origin = inferred_from_code`；
   - BL-11 对 `proposal_item_id` 只检查是否为空；
   - BL-13 成功的 `propose --apply` 只在进程内（注入 `human_cli`）运行，伪终端中只测试了取消；
   - CX-14 的版本由 SQL 种子构造；
   - SRC-06 中"移除后重新登记会改变哈希"只有单元测试；
   - INIT-12 中"无法判定"的情形只在 `gitx` 层验证；
   - TGT-06 没有单独断言被排除的条目中有 `conflict`；TGT-10 的审核路径没有断言 `error.current`；TGT-11 的审核路径没有字面的"C1 对 C2"反例；
   - `TestCommandLineErrorsQuoteNothing` 没有加入 WP1.8 的命令组（机制对所有命令统一生效）；
   - `TestInvalidCasesFail` 只检查反例被拒绝，不检查被拒绝的位置；
   - 项目名称修复中旧项目测试的无写入摘要，没有直接覆盖 settings、worktree 全字段与 exclude（审查 037 N2，维护事项 M-13）。
5. **自动化不变量**：acceptance-criteria §3 的不变量，由每个 E2E 结束时自动断言。E2E 和这套断言属于 M2 之后（WP2.9 的夹具与 oracle，WP6.1），M1 没有。M1 中各不变量的现有证据见第 6 节。

## 9. 独立维护事项

除注明已关闭的之外都未关闭，本次不修复。优先级：高 = 下一个里程碑开工前处理；中 = 在注明的时间点之前处理；低 = 方便时处理。都不是已接受的风险；唯一明确接受的局部限制，见 M-12。

| 编号 | 事项 | 来源 | 性质 | 建议优先级 |
|---|---|---|---|---|
| M-1 | Windows 首次创建数据库期间，`datadir.Check` 偶发 `Access is denied`（`TestCheckWhileTheDatabaseIsCreated`） | codex-review-035 N2；PR #14 CI attempt 1 | 可靠性；失败方向安全（退出码 6，不写入），但会错误地报告环境问题。此后的 CI 中没有复现，这不代表已经解决 | 高：M3 之前。Hook 与采集会让并发进程成为常态 |
| M-2 | `doctor` 的 git 版本查询继承完整环境（`GIT_TRACE` 等会写出跟踪文件） | codex-review-016 N1 | 安全：违反"非破坏性诊断" | 中：v0.1.0 发布之前（M6 的"必须关闭"清单） |
| ~~M-3~~ | ~~第 1 版中 6 项 INCOMPLETE 的测试与审查证据~~ **已关闭**：PR #16（合并提交 `01bd50f`）补齐测试，SRC-07 由审查 036 独立审查通过；codex-review-036 GO | 第 1 版核对 | 验收证据 | — |
| M-4 | 跨项目需求外键的测试可能由另一条外键先拒绝 | codex-review-034 N1 / 035 N1 | 测试覆盖的准确性；实际约束正确 | 低 |
| M-5 | 自定义 Writer 与进程 stderr 的终端可见性可能不一致 | codex-review-033 N1 | 使用约束；正式二进制不受影响 | 低：将来支持嵌入或包装输出时 |
| M-6 | 首次登记特殊路径的非文档来源时，确认可能无法输入 | codex-review-031 N1 | 可用性；没有错误放行 | 低 |
| ~~M-7~~ | ~~部分成功 JSON 输出的测试只检查以 `{` 开头~~ **已关闭**（2026-10-20）：PR #28（合并提交 `7a36166`，master CI 正式成功）；codex-review-064 GO（review-resolution §76） | codex-review-029 N1 | 测试强度 | 中：M2 新增输出接口之前统一改为按 schema 校验。修复（审查对象 `a482dac`）：`TestCommandLineStillWorks` 的五种 `--json` 写法按 `source-list.v1`、`proposal-list.v1`、`question-list.v1` 校验完整文档，`--json` 在前时与在后的输出相同；`TestValidateJSONTakesOneWholeDocument` 证明校验拒绝文档前后的非空内容。审查 064 的 N1：该夹具不能区分默认列表与 `--all`，过滤语义由既有的 `TestQuestionAnswerAndDismiss` 覆盖 |
| M-8 | WP1.6 N1：确认预览显示审核者选择的 `editorial`，而强制字段会把它升级为 `material`；N2：`baseline history` 没有输出后续处理的理由 | codex-review-025 | 显示与读取模型；持久化正确 | 中：M5 Dashboard 的审核界面之前 |
| M-9 | 在 `SPECMETRY_TEST_GIT_OBJECTS=unsafe` 下，`gitx` 的 3 个正向测试不跳过（第 7 节） | 本次核对 | 测试脚手架 | 低 |
| M-10 | 本地 Windows 上 `internal/cli` 的测试超过 Go 默认的 10 分钟超时，所以本地 `scripts/check.sh` 不能完整通过 | 历次本地运行 | 开发体验；CI 不受影响 | 低 |
| ~~M-11~~ | ~~文档同步~~ **已关闭**，逐项核对：① SRC-07 的里程碑标注已改为"M1 / M2"（PR #16）；② implementation-plan"输出契约的语义约束"中的已知缺口已在本版标为已补齐；③ acceptance-criteria 的 CH-02 一行原称 `source ack` 已在 WP1.4 验证 stdout 重定向，已在本版改为如实的来源（该项证据来自 PR #16）。CH-01 一行同样补注了 PR #16 | 第 1 版核对 | 文档 | — |
| ~~M-12~~ | ~~项目名称允许代码围栏~~ **已修复**：PR #17（合并提交 `5aad555`），codex-review-037 GO。新登记的名称禁止代码围栏，`init.v1` 同步。**已批准的兼容性限制**（方案 1）见第 11 节 | 第 1 版核对 | 隐私规则 | — |
| M-13 | 旧项目测试（`TestInitWithARegisteredNameThatHoldsACodeFence`）的无写入摘要，只比较名称、阶段、worktree 数量、`seq` 与 marker，没有直接覆盖完整的 settings、worktree 明细与 exclude。生产检查顺序正确，审查方已另行独立验证 | codex-review-037 N2 | 测试强度（NON-BLOCKER） | 低：本轮不改测试 |
| M-14 | macOS 上多个进程打开同一个数据库时，读取数据库版本偶发 `SQLITE_READONLY_RECOVERY`（264）。`internal/store/sqlite` 的 `TestSeqIsUniqueAcrossProcesses` 中，一个辅助进程报告 `cannot read the database version: cannot read the database: attempt to write a readonly database (264)`。**根因待调查**："另一个进程正在做 WAL 恢复，造成瞬时竞争"只是假设，没有证实。与 M-1 分开登记，没有证据表明二者同根因。相同的代码此前在 macOS 上通过（run 39108257137、39109376799、39109898115），这次重跑也通过；这不能说明它只是 CI 环境的问题或已经无害，也不关闭本事项 | PR #23 [run 39110669282](https://github.com/XKHoshizora/specmetry/actions/runs/39110669282)：<br>• attempt 1：只有 Go tests (macos-latest) 失败，其余 7 个作业成功；<br>• attempt 2 由"重跑失败的作业"触发：API 列出 8 个作业，作业 ID 都是新的，但实际重新执行的只有 Go tests (macos-latest)（08:27:53 ~ 08:29:37，该测试通过）；其余 7 个作业的日志与起止时间都与 attempt 1 相同，沿用的是 attempt 1 的结果 | 可靠性：跨进程打开数据库；未解决，不是接受的风险 | 高（建议）：M3 之前，与 M-1 相同。Hook 与采集会让并发进程成为常态 |
| M-15 | 真实 `Open` 场景中，写入阶段偶发 `SQLITE_BUSY`：Windows CI 上 200 轮中有 2 轮、共 3 次 `database is locked (5) (SQLITE_BUSY)`，分类为 internal（退出码 1）。**这是待独立审查的诊断观测**（作者报告），根因没有调查。它不属于 M-14 的定义（M-14 是打开时的版本探测），与 M-14 分开登记，没有证据表明二者同根因。本事项本身不批准任何错误映射或重试策略。**新观测（2026-10-20）**：PR #27 合并后的 master push CI [run 39287153098](https://github.com/XKHoshizora/specmetry/actions/runs/39287153098) 的 attempt 1 中，Go tests (windows-latest)（作业 118177199541，检出 `5985b31`）的 `TestSeqIsUniqueAcrossProcesses` 失败：一个 `seqwriter` 辅助进程 `exit status 1: database is locked (5) (SQLITE_BUSY)`；按负责人的一次授权重跑失败作业（attempt 2，作业 118203422167），该测试通过。失败发生在 `Open` 还是写入阶段、与以往观测是否同一根因，都没有确认；根因未验证，未解决，不是已接受的风险；重跑成功不关闭本事项（review-resolution §75）。**再次观测（2026-10-20）**：PR #29（`91e7e5f`）的 [run 39307739451](https://github.com/XKHoshizora/specmetry/actions/runs/39307739451) attempt 1 中，Go tests (windows-latest)（作业 118249164263，检出 `60f8820`）的 `TestSeqIsUniqueAcrossConnections`（`store_test.go:546: database is locked (5) (SQLITE_BUSY)`；同一进程中的多个连接，**是本事项此前没有记录的测试场景**）与 `TestSeqIsUniqueAcrossProcesses`（`store_test.go:571: exit status 1: database is locked (5) (SQLITE_BUSY)`）都失败；同一作业中各包都明显变慢（见 M-16）。错误阶段、与以往观测是否同根因，都没有确认；没有重跑（review-resolution §77） | PR #25（M-14 的诊断工具，未合并）的被测提交 `1d8b73c9e637e339f2e1ebac4c81b5bd029480de`，[run 39118832445](https://github.com/XKHoshizora/specmetry/actions/runs/39118832445) 的 Windows 作业；记录见该 PR 的 `docs/m14-diagnostics.md` §5。登记见 review-resolution §63 | 可靠性：并发写入；未解决，不是接受的风险 | 高（建议）：M3 之前，与 M-1、M-14 相同；WP2.3 的扫描会在调用它的命令中增加写入，实现开工前建议先评估 |
| M-16 | Windows CI 上 `internal/cli` 包的测试超过 `go test` 默认的每包 600 秒超时，耗时余量不足：[run 39307739451](https://github.com/XKHoshizora/specmetry/actions/runs/39307739451) attempt 1 的 Go tests (windows-latest)（作业 118249164263）中，`internal/cli` 在 600.048 s 时 `panic: test timed out after 10m0s`（当时运行的是 `TestSourceHumanOnlyCommandsRefuseAgentInTerminal`，已运行 1 s）。此前 Windows CI 上该包为 282 ~ 454 s（30 分钟前 master `7a36166` 的 run 39303615136 为 454 s）。**观测**：同一作业中其他包也明显变慢（`gitx` 576 s、`gitscan` 153 s、`store/sqlite` 142 s、`project` 56 s，对照 master 的 340 s、59 s、43 s、13 s）。**待验证的解释**：运行器的资源竞争；没有证实。与 M-10（本地 Windows 超过默认超时）分开登记：M-10 是本地环境，本项是 CI。**再次观测（2026-10-21）**：PR #30（WP2.4 实现，未合并）的 [run 39526415079](https://github.com/XKHoshizora/specmetry/actions/runs/39526415079)（HEAD `e70c09b`，检出 `b0ffc03`）中，Go tests (windows-latest)（作业 118981996163）的 `internal/cli` 为 555.293 s，在 600 s 之内，余量约 45 s；同一作业的 `gitx` 为 369.145 s，而前一次 run 39525308601 中二者为 388.787 s、243.428 s。同一作业中其他包变慢只是观测，原因未验证 | run 39307739451（PR #29，`91e7e5f`，检出 `60f8820`）；对照 run 39303615136（master `7a36166`）；review-resolution §77；run 39526415079 与 39525308601（PR #30） | 可靠性：CI 的耗时余量；未解决，不是接受的风险 | **2026-10-21 负责人批准的风险缓解**（超时预算的调整，不是根因修复）：Windows CI 的 Go tests 作业中，`internal/cli` 单独执行一次，显式 `-timeout 20m`；其他包仍按 `go test` 默认的每包 10 分钟执行。包集合取自 `go list -tags nowebui ./...`，每个包只在两次执行之一中出现，任一失败都使作业失败。Ubuntu、macOS 的执行方式、Ubuntu 的 Race Detector 与 root 跨用户步骤不变（CI 调整 PR #31，分支 `ci/m16-windows-cli-timeout`）。**代价**：Windows 上 `internal/cli` 的挂起最晚要到 20 分钟才会被发现。根因未验证；本项保持未解决，不因此关闭。**独立审查（2026-10-22）**：[Review 069](reviews/codex-review-069.md)（审查对象 PR #31 HEAD `0de32c5`，run 39600735762 检出 `3c901fb`）为 GO，没有 BLOCKER，也没有本 PR 新增的 NON-BLOCKER；负责人 2026-10-22 条件授权合并 PR #31。审查同样认定：本次是风险缓解，一次成功的 CI 不证明 20 分钟始终足够，也不证明 CI 变慢的原因 |

## 10. 结论

**9 个工作包都已通过代码验收并合并。59 个 P0 项全部 PASS，没有 INCOMPLETE。最新 master（`5aad555`）的三平台 CI 8/8 通过。**

第 1 版中 6 项 INCOMPLETE 的关闭情况：

| ID | 第 1 版缺少的证据 | 关闭方式 | 合并提交 | 审查 |
|---|---|---|---|---|
| MVP-CH-01 | `propose --apply`、`question dismiss` 在伪终端中设置 Agent 变量的测试 | `TestAgentVariablesRefuseTheRemainingHumanOnlyCommands`；`TestBuiltBinary` | `01bd50f`（PR #16） | 036 GO |
| MVP-CH-02 | `source ack`、`source add --allow-any-type`、`propose --apply`、`question dismiss` 在 stdout 重定向下的真实子进程测试 | `TestRedirectionRefusesTheRemainingHumanOnlyCommands`；`TestBuiltBinary` | `01bd50f`（PR #16） | 036 GO |
| MVP-SRC-07 | "只为引用校验读取文本，不解析文档语义"的审查记录 | 审查方按实际调用链独立审查，PASS；该行改标 M1 / M2 | `01bd50f`（PR #16 的标注） | 036 GO |
| MVP-BL-21 | "提示改用环境变量"的断言 | `TestProposeSecretInACheckNamesAnEnvironmentVariable` | `01bd50f`（PR #16） | 036 GO |
| MVP-BL-27 | 冲突条目缺少 `change_class` 的测试 | `TestReviewConflictItemNeedsAChangeClass` | `01bd50f`（PR #16） | 036 GO |
| MVP-PRV-07 | `review` 拒绝密钥的 CLI 测试 | `TestReviewRefusesASecretAnywhereInTheDocument` | `01bd50f`（PR #16） | 036 GO |

其他结论：

- 没有发现行为缺陷，也没有发现应在 M1 完成、却被推迟到 M2 的写入校验、数据模型或 JSON 读取契约。本版没有新增推迟；DEFERRED 的部分都属于推导、状态或界面，依据见第 5 节。
- M1 最终验收前的独立安全修复（项目名称禁止代码围栏）已完成：PR #17，合并提交 `5aad555`，codex-review-037 GO。它是 M1 验收的前置条件，**不计入 59 个 P0 项**。
- P1：INIT-08、INIT-10 通过；INV-05（`init --json` 输出提交数与最近提交时间）没有实现。它不阻塞 M1。
- 维护事项：M-3、M-11 已关闭，M-12 已修复；其余见第 9 节，都没有作为已接受的风险。

**结论：按退出条件，建议 M1 验收。** codex-review-038 的结论：PR #15 GO；固定代码快照 `5aad555` 的 M1 退出条件 GO，59 PASS、0 INCOMPLETE。证据边界见第 8 节第 3 条。

**验收**：产品负责人于 2026-10-17（Asia/Tokyo）正式验收 M1。

- 验收对象：`7e77fd6`，即 PR #15 的合并提交。它的父提交是 `5aad555` 与 `98a2c34`，树与 `98a2c34` 相同。
- 依据：
  - 审查 038：M1 退出条件 GO，59 PASS、0 INCOMPLETE；
  - 审查 039：最终的文档增量与合并结果 GO；
  - 合并后的 master CI：[run 39025525139](https://github.com/XKHoshizora/specmetry/actions/runs/39025525139)，首轮 8/8 成功。

本节以上的历史结论、审查目标与提交身份都保持原样。第 9 节中未解决的维护事项都保留；接受的风险仍只有方案 1 的局部兼容性限制（第 11 节）。这次验收不授权开始 M2。

## 11. 待决问题与已批准的决定

M1 范围内没有待决问题。第 1 版提出的问题与已确认的决定：

- **项目名称禁止代码围栏**：产品负责人确定统一规则，由 PR #17 实现（审查 037 GO）。对已登记的旧名称，产品负责人正式批准**方案 1**，含义如下：
  1. 新登记的项目名称禁止三个连续反引号（`--name` 与默认的目录名）。
  2. 已登记的名称违反这一规则时，`init` 返回有说明的 `invalid_input`（退出码 2）。
  3. 不静默改名，不改写旧名称、marker 或其他业务事实。旧项目保证的是业务事实（名称、阶段、settings、worktree、`seq`）、marker、exclude 与仓库文件不变。这一检查需要打开数据库读取登记的事实，普通连接可能更新 SQLite 的辅助文件（例如 `-shm` 的修改时间），所以不能说成"数据库未打开"或"数据目录绝对零变化"。
  4. 接受的局部限制：这类旧项目暂时不能通过 `init` 修改阶段、补写 marker 或登记关联 worktree。
  5. 这一决定只针对这类名称，不代表所有开发版本数据都不需要兼容。
  6. 不增加改名命令，不修改数据库结构或迁移。
  7. 不据此关闭其他未解决的风险。
- **三类集成证据的定义**：产品负责人已正式批准（acceptance-criteria §1；审查 036 记录）。
- **SRC-07 的 M1 / M2 划分**：已确认，与审查 019 和 review-resolution §24 一致（第 5 节）。
- **留给后续阶段的设计约束**（本轮不决定）：M2 的 `status.v1` 也有 `project.name`，旧项目的名称在其中如何处理，在 M2 设计时确定；不推定所有读取接口都必须拒绝旧项目。
