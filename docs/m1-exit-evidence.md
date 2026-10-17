# M1 退出条件证据核对

- 核对日期：2026-10-17
- 核对对象：master 上的合并提交 `19600ec`（PR #14 合并后）。它的树与审查提交 `7f65408` 完全相同。
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
- 结论在第 10 节。本文只整理证据，**不宣布 M1 完成**；M1 是否验收，由产品负责人决定。

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

另有三项独立安全与维护修复，同样已合并：

| 修复 | 审查 | PR | 合并提交 | 处理记录 |
|---|---|---|---|---|
| CLI 错误不回显不可信参数 | codex-review-030 GO | #11 | `a91016e` | §35 ~ §37 |
| 终端输出转义控制字符 | codex-review-032 GO | #12 | `26c787b` | §38 ~ §40 |
| 人工确认与 JSON 输出分流 | codex-review-033 GO | #13 | `fbd6dff` | §41 ~ §42 |

以上合并提交都在 master 历史中（已用 `git merge-base --is-ancestor` 逐一确认）。

## 3. 三平台 CI 与运行环境

**master 的 CI**：[run 39000934831](https://github.com/XKHoshizora/specmetry/actions/runs/39000934831)，提交 `19600ec`，一次通过，8/8 成功。

| 平台 | git | Go 测试（顶层与子测试） |
|---|---|---|
| ubuntu-latest（带 `-race`） | 2.55.0 | 579 PASS，6 SKIP，0 FAIL |
| windows-latest | 2.56.0 | 575 PASS，3 SKIP，0 FAIL |
| macos-latest | 2.56.0 | 578 PASS，2 SKIP，0 FAIL |

跳过的测试，以及它们与 M1 的关系：

- `TestInitRefusesRealOldGit`、`TestARealGitThatFetches`：三个平台都跳过。它们需要环境中提供真实的旧版 git，CI 中按设计不提供（review-resolution §29）。本次已在本地重跑，见第 7 节。
- ubuntu 的四个跨用户测试（`TestCrossUser*`）：属于 M0 的存储验收，在同一 job 中由单独的 root 步骤运行。
- windows 的 `TestSourceListUnreadableSource`：需要当前用户无法读取的文件，在 Windows 上跳过；它不是任何 M1 验收项的唯一证据。

除这些之外，第 4 节列出的每个测试在三个平台上都是 PASS。

**本地运行**（与 master 相同的代码）：

- Windows（git 2.56.0）：`scripts/check.sh` 中 `internal/cli` 以外的包都通过。`internal/cli` 触发 Go 默认的 10 分钟超时，此前没有测试失败；单独以 `-timeout 30m` 运行，通过。
- WSL Linux（git 2.43.0，`SPECMETRY_TEST_GIT_OBJECTS=safe`）：19 个测试二进制全部通过。

**PR #14 的 CI**：[run 38957016420](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420)。

- attempt 1：Windows 的 `TestCheckWhileTheDatabaseIsCreated` 偶发失败（`Access is denied`），已登记为维护事项 M-1（第 9 节）；
- attempt 2：只重跑失败的 job，8/8 通过。

## 4. P0 验收项逐项结果

状态的含义：

- **PASS**：M1 要求的每一条都有断言，并在 CI 中运行通过；
- **INCOMPLETE**：有一条缺少验证证据（下表注明是否已经实现）；
- **DEFERRED**：只用于"M1 / M2"等项中属于后续里程碑的部分，并注明依据。

表中测试名省略了包路径，除特别注明外，都在 `internal/cli`。"CI"一列是 master CI 上 ubuntu / windows / macos 的结果。

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
| MVP-SRC-07 | WP1.4 | 文档类型白名单（V-S01）；再次登记已登记的来源时原样返回 | `TestSourceAddRefuses`、`TestSourceAddAnyTypeInTerminal`、`TestSourceAddAgainChangesNothing`、`TestSourceAddAgainOfApprovedNonDocument` | 已验证：非文档类型退出码 2；TTY 中加 `--allow-any-type` 成功；Agent 再次登记时 `created = false`，`acked_sha256` 不变，状态仍为 `changed`，`seq` 不变。**缺少**：本行要求的"审查确认 Specmetry 只为引用校验读取文本，不解析文档语义"，在处理记录和审查原文中都没有明确记录 | PPP | 019 | **INCOMPLETE**（缺审查证据）；`SOURCE_CHANGED` 差距 DEFERRED（M2） |
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
| MVP-BL-21 | WP1.5 | `TestProposeRefuses`、`TestProposeAcceptsAChecked` | 已验证：没有 `rationale`、`inputs` 或 `workdir` 越出仓库，退出码 2；命令中有密钥，退出码 2；使用环境变量的命令被接受。**缺少**：本行要求的"提示改用环境变量"没有断言。提示已经实现（`secrets.Reject` 的消息），但没有测试检查它 | PPP | 024 | **INCOMPLETE**（缺测试证据；功能已实现） |
| MVP-BL-22 | WP1.6 | `TestReviewCarriesReferences` | `material` 更新不提供新引用时，沿用上一修订的引用，重新校验后写成新行，旧行保留 | PPP | 026 | PASS |
| MVP-BL-23 | WP1.5、WP1.6 | `TestProposeApply`、`TestProposeRefuses` | `update` 条目缺少 `change_class`：退出码 2，不写入；代码围栏、多行标题：退出码 2 | PPP | 024、026 | PASS |
| MVP-BL-24 | WP1.6 | `TestReviewApplicationOrder` | 同一提案中 `reaffirm` 的、新增的和修改的 AC，都取父需求的新 revision | PPP | 026 | PASS |
| MVP-BL-25 | WP1.6、WP1.7 | `TestReviewRelatesQuestions`、`TestQuestionAnsweredBeforeItsItem` | 条目被接受后，问题关联到新编号；被拒绝的条目，问题仍指向原引用 | PPP | 026、028 | PASS（M1）；"不再计为阻塞" DEFERRED（M2） |
| MVP-BL-27 | WP1.6 | `TestReviewChangeClasses` | 已验证：`accepted` 与 `accepted_with_edits` 缺少 `change_class`，退出码 2，指向 `/items/<i>/change_class`，不写入；不采用提案建议的类别；强制字段按 `material` 记录。**缺少**：本行明确列出的"冲突条目"（带 `acknowledged_conflict`、缺少 `change_class`）没有测试。代码路径相同（`review.Check` 对所有被接受的 `update` 条目统一检查） | PPP | 026 | **INCOMPLETE**（缺测试证据；功能已实现） |
| MVP-BL-28 | WP1.5 | `TestProposeRecordedProposal`、`TestProposeChecksCodeRefs`、`TestProposeNeverStoresExcerpts`、`TestProposeApply`；SCH-79、113 ~ 125 | 输出通过 `proposal-receipt.v1` 校验；每个条目有 `item_id` 与片段校验结果，代码引用有 `exists`；输出、错误和数据目录中都找不到片段原文 canary；条件约束与反例；回执 ID 与数据库一致 | PPP | 024 | PASS |
| MVP-BL-29 | WP1.5（PD-26） | `TestProposeAndSourcesWithAGitThatFetches`、`TestProposeRefusesPartialCloneByAnyMeans`（替身）；`TestARealGitThatFetches`（真实 git 2.32.0） | 带代码引用的 `propose`：退出码 6，不写入，不分配 `seq`，`.git` 不变；不带代码引用的 `propose` 和 `source` 命令照常工作。真实 git 2.32.0 的检查已在本次重跑通过（第 7 节） | PPP（替身）；真实 git 在本地 | 022 ~ 024 | PASS |
| MVP-PRV-07 | WP1.3，以及各写入工作包 | `secrets` `TestBuiltinRuleSamples`、`TestRejectionNeverEchoesTheMatchedText` 等；`config` `TestLoadRejectsInvalidExtraRuleWithoutEchoingIt`；各命令的 CLI 测试 | 已验证：每条内置规则至少 2 个命中、2 个不命中样本，运行时拼接；环境变量引用不命中；只给出 Pointer 与规则编号；边界反例 ①②③；无效的追加规则退出码 6，不回显。CLI 拒绝写入并且不写入：`init`、`propose`、`source`、`question`、`target set`、`scope confirm` 都有测试。**缺少**：`review` 会扫描整份文档（`internal/cli/review.go:71`），但没有 CLI 测试验证它拒绝密钥 | PPP | 011、024、026、028、035 | **INCOMPLETE**（缺 `review` 的测试证据；功能已实现） |
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
| MVP-CH-01 | WP1.2、1.4、1.6、1.7、1.8 | `channel.Detect`（`executeRoot` 中每次执行识别一次）；每个人类专属命令的执行体首先调用 `RequireHuman`；`confirm` 中的 `channel.Guard` | 见下文矩阵 | 本行列出的 `review`、`source ack`、`scope confirm`，以及 WP1.4 的 `source remove`、`source add --allow-any-type` 和 `target set`：两个变量分别在伪终端中验证，canary 不在输出和数据目录中。**缺少**：`propose --apply`、`question dismiss` 没有伪终端 + Agent 变量的测试（`question answer` 只测试了 `CLAUDECODE`，本行写的是"或"，可以接受） | PPP | 014、019、026、028、035 | **INCOMPLETE**（按"全部 M1 人类专属真实命令"；缺测试证据，没有发现行为缺陷） |
| MVP-CH-02 | 同上 | 同上 | 见下文矩阵 | stdout 重定向（伪终端中的真实子进程）：`review`（带 `--json`）、`source remove`、`question answer`、`target set`、`scope confirm` 已验证。**缺少**：`source ack`（本行写明"已在 WP1.4 验证（stdout 重定向……）"，但测试只用了 `source remove`）、`source add --allow-any-type`、`propose --apply`、`question dismiss`。stdin 不是终端：真实子进程只有 `source remove`、`target set`、`scope confirm`，其余命令依靠测试进程本身不是终端（进程内） | PPP | 同上 | **INCOMPLETE**（缺测试证据，没有发现行为缺陷） |

**CH-01 / CH-02 的命令 × 情形矩阵**（伪终端测试在三个平台都运行，没有跳过条件）：

| 命令 | 伪终端 + Agent 变量 | 伪终端中拒绝确认 | stdout 重定向 | stdin 不是终端 | 带 `--json` |
|---|---|---|---|---|---|
| `review` | 两个变量 | 有 | 有（带 `--json`） | 进程内 | 有 |
| `propose --apply` | **无** | 只断言退出码 3 | **无** | 进程内 | 有 |
| `source ack` | 两个变量 | 有 | **无** | 进程内 | 有 |
| `source remove` | 两个变量 | 只断言退出码 3 | 有 | 真实子进程 | 有 |
| `source add --allow-any-type` | 两个变量 | 有 | **无** | 进程内 | 有 |
| `question answer` | `CLAUDECODE` | 有 | 有 | 进程内 | 有 |
| `question dismiss` | **无** | 有 | **无** | 进程内 | 有 |
| `target set` | 两个变量 | 有 | 有 | 真实子进程 | 有 |
| `scope confirm` | 两个变量 | 有 | 有 | 真实子进程 | 有 |

- 共同机制的证据：
  - WP1.2 的替身命令在真实伪终端中验证了两个变量、管道（带与不带 `--json`）和 stdout 重定向（`TestHumanOnlyCommand*`）；
  - `TestConfirmationWithJSON` 对全部 9 个确认，在进程内验证 Agent 通道退出码 3、没有提示、不写入；
  - `TestHumanOnlyCommandsConfirmThroughConfirm` 约束所有命令都通过同一个确认入口。
- 9 个人类专属入口都在执行体开头调用 `RequireHuman`。因此缺口是按命令逐一的真实终端证据不全，不是行为缺陷。
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

**一处标注不一致**：SRC-07 中 `SOURCE_CHANGED` 作为差距的验证，只在 review-resolution §24 中记录为"M2"（WP1.4 验收时，审查 019 GO）。但 acceptance-criteria 中这一行的里程碑是"M1"，而不是"M1 / M2"。按 M1 退出条件的字面，只有"M1 / M2"项的推导侧可以推迟。建议产品负责人确认，并把这一行改标为"M1 / M2"，或在行内注明"（M2）"。本文暂按 §24 的记录处理。

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
- 反例共 149 个（SCH-1 ~ 149）。实现输出接口时要补齐的条件约束（implementation-plan"输出契约的语义约束"）：
  - `proposal-receipt.v1`：SCH-113 ~ 125（BL-28）；
  - `proposal-show.v1`：SCH-126 ~ 129（BL-14）；
  - `question-list.v1`：SCH-131 ~ 136（BL-17）；
  - `target-scope.v1`：SCH-140 ~ 147。

  这三处已知缺口实际上都已补齐，但 implementation-plan 中的"已知缺口"一句没有同步更新（维护事项 M-11）。
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
| `init` | 项目名、要保存的路径、默认目标的 ref | 有 |
| `source add` | 路径（解析前后） | 有 |
| `source ack` | 理由与来源参数 | 有（理由） |
| `source remove` | 来源参数 | 无（参数含密钥的情形） |
| `propose`、`propose --apply` | 整份文档与 worktree 路径 | 有（`--apply` 与 `propose` 共用同一段代码） |
| `review` | 整份文档（`review.go:71`） | **无**（PRV-07 的缺口） |
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
- 此后 WP1.6 ~ WP1.8 改动过 `init`、`source` 与 `propose` 的周边代码。

因此本次在 WSL 中，用与 master 相同的代码（交叉编译的测试二进制）重跑：

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

1. **INCOMPLETE 的 6 项**（第 4 节、第 10 节）：
   - CH-01、CH-02：部分人类专属命令缺少逐一的真实终端测试；
   - BL-21：缺少对"提示改用环境变量"的断言；
   - BL-27：缺少冲突条目缺少 `change_class` 的测试；
   - PRV-07：缺少 `review` 拒绝密钥的 CLI 测试；
   - SRC-07：缺少"只为引用校验读取文本、不解析文档语义"的审查记录。

   前五项的功能都已实现，缺的是测试；SRC-07 缺的是审查结论。它们都不需要修改业务实现。按本次任务的要求，本文不补开发，也不补测试。
2. **"集成"的含义**：acceptance-criteria §1 把"集成"定义为"运行真实的二进制"。实际的测试方式有三种：
   - 非 TTY 的情形，在测试进程内调用 `cli.Execute`（与 `cmd/specmetry/main.go` 调用的是同一个入口）；
   - TTY 的情形，在伪终端中重新运行测试二进制（helper `cli`，与正式二进制相比只多一个测试专用命令）；
   - 只有补全拒绝的测试（`completion_test.go`）构建了真正的 `cmd/specmetry`。

   历次审查把这种方式当作"公开 CLI"接受，但没有文档明确认定它等同于"运行真实的二进制"。建议产品负责人确认。
3. **测试强度不足的地方**（不影响状态，供后续改进）：
   - BL-12 只在新增检查时一起修改 `verification`，没有单独修改；
   - BL-05 没有断言被接受的实体保留 `origin = inferred_from_code`；
   - BL-11 对 `proposal_item_id` 只检查是否为空；
   - BL-13 成功的 `propose --apply` 只在进程内（注入 `human_cli`）运行，伪终端中只测试了取消；
   - CX-14 的版本由 SQL 种子构造；
   - SRC-06 中"移除后重新登记会改变哈希"只有单元测试；
   - INIT-12 中"无法判定"的情形只在 `gitx` 层验证；
   - TGT-06 没有单独断言被排除的条目中有 `conflict`；TGT-10 的审核路径没有断言 `error.current`；TGT-11 的审核路径没有字面的"C1 对 C2"反例；
   - `TestCommandLineErrorsQuoteNothing` 没有加入 WP1.8 的命令组（机制对所有命令统一生效）；
   - `TestInvalidCasesFail` 只检查反例被拒绝，不检查被拒绝的位置。
4. **自动化不变量**：acceptance-criteria §3 的不变量，由每个 E2E 结束时自动断言。E2E 和这套断言属于 M2 之后（WP2.9 的夹具与 oracle，WP6.1），M1 没有。M1 中各不变量的现有证据见第 6 节。

## 9. 独立维护事项

都未关闭，本次不修复。优先级：高 = 下一个里程碑开工前处理；中 = 在注明的时间点之前处理；低 = 方便时处理。

| 编号 | 事项 | 来源 | 性质 | 建议优先级 |
|---|---|---|---|---|
| M-1 | Windows 首次创建数据库期间，`datadir.Check` 偶发 `Access is denied`（`TestCheckWhileTheDatabaseIsCreated`） | codex-review-035 N2；PR #14 CI attempt 1 | 可靠性；失败方向安全（退出码 6，不写入），但会错误地报告环境问题 | 高：M3 之前。Hook 与采集会让并发进程成为常态 |
| M-2 | `doctor` 的 git 版本查询继承完整环境（`GIT_TRACE` 等会写出跟踪文件） | codex-review-016 N1 | 安全：违反"非破坏性诊断" | 中：v0.1.0 发布之前（M6 的"必须关闭"清单） |
| M-3 | 第 10 节中 6 项 INCOMPLETE 的测试与审查证据 | 本次核对 | 验收证据 | 高：M1 验收之前 |
| M-4 | 跨项目需求外键的测试可能由另一条外键先拒绝 | codex-review-034 N1 / 035 N1 | 测试覆盖的准确性；实际约束正确 | 低 |
| M-5 | 自定义 Writer 与进程 stderr 的终端可见性可能不一致 | codex-review-033 N1 | 使用约束；正式二进制不受影响 | 低：将来支持嵌入或包装输出时 |
| M-6 | 首次登记特殊路径的非文档来源时，确认可能无法输入 | codex-review-031 N1 | 可用性；没有错误放行 | 低 |
| M-7 | 部分成功 JSON 输出的测试只检查以 `{` 开头 | codex-review-029 N1 | 测试强度 | 中：M2 新增输出接口之前统一改为按 schema 校验 |
| M-8 | WP1.6 N1：确认预览显示审核者选择的 `editorial`，而强制字段会把它升级为 `material`；N2：`baseline history` 没有输出后续处理的理由 | codex-review-025 | 显示与读取模型；持久化正确 | 中：M5 Dashboard 的审核界面之前 |
| M-9 | 在 `SPECMETRY_TEST_GIT_OBJECTS=unsafe` 下，`gitx` 的 3 个正向测试不跳过（第 7 节） | 本次核对 | 测试脚手架 | 低 |
| M-10 | 本地 Windows 上 `internal/cli` 的测试超过 Go 默认的 10 分钟超时 | 历次本地运行 | 开发体验；CI 不受影响 | 低 |
| M-11 | 文档同步：SRC-07 的里程碑标注与 §24 的推迟记录不一致（第 5 节）；implementation-plan"输出契约的语义约束"中的已知缺口实际已补齐；CH-02 一行称 `source ack` 已验证 stdout 重定向，与测试不符 | 本次核对 | 文档 | 低：随 M-3 一并处理 |
| M-12 | 项目名称允许代码围栏（第 11 节） | 本次核对 | 隐私规则的范围，待产品负责人决定 | 中：v0.1.0 之前确定 |

## 10. 结论

**9 个工作包都已通过代码验收并合并。master 的三平台 CI 8/8 通过。59 个 P0 项中，53 项 PASS，6 项 INCOMPLETE。**

INCOMPLETE 的 6 项：

| ID | 缺少的证据 | 功能 |
|---|---|---|
| MVP-CH-01 | `propose --apply`、`question dismiss` 在伪终端中设置 Agent 变量的测试 | 已实现（`RequireHuman` 位于执行体开头，通道在每次执行时识别） |
| MVP-CH-02 | `source ack`、`source add --allow-any-type`、`propose --apply`、`question dismiss` 在 stdout 重定向下的真实子进程测试 | 同上 |
| MVP-SRC-07 | "只为引用校验读取文本，不解析文档语义"的审查记录 | 设计已规定（architecture、FR-SRC-06），没有审查结论 |
| MVP-BL-21 | 对"提示改用环境变量"的断言 | 已实现（`secrets.Reject` 的消息） |
| MVP-BL-27 | 冲突条目缺少 `change_class` 的测试 | 已实现（代码路径相同） |
| MVP-PRV-07 | `review` 拒绝密钥的 CLI 测试 | 已实现（`review` 扫描整份文档） |

- 这 6 项都是**验证证据不足**，核对中没有发现行为缺陷，也没有发现应在 M1 完成、却被推迟到 M2 的写入校验、数据模型或 JSON 读取契约。
- DEFERRED 的部分都属于推导、状态或界面，依据见第 5 节。
- P1：INIT-08、INIT-10 通过；INV-05（`init --json` 输出提交数与最近提交时间）没有实现。它不阻塞 M1。

**结论：按退出条件"全部 P0 项"，M1 当前不满足退出条件，不建议现在验收 M1。**

建议：

1. 安排一个只补证据的小任务。补齐 CH-01、CH-02、BL-21、BL-27、PRV-07 的测试；请 Codex 对 SRC-07 的"不解析文档语义"给出审查结论。这些都不需要修改业务实现。
2. 同时由产品负责人确认三点：
   - SRC-07 中 `SOURCE_CHANGED` 的里程碑标注（第 5 节）；
   - "集成"测试方式是否满足"运行真实的二进制"（第 8 节第 2 条）；
   - 项目名称是否也应禁止代码围栏（第 11 节）。
3. 完成以上之后，再交给 Codex 做 M1 最终退出条件核查。

## 11. 核对中发现的待决问题

- **项目名称允许代码围栏**：`init --name`（以及默认使用的目录名）由 `validateName` 校验非空、单行、无控制字符、不超过 200 个字符，但不禁止代码围栏；`init.v1` 的 `project.name` 也没有这一限制。
  - architecture §10.2 第 4 条的自由文本清单没有列出项目名称，所以这不是对现有规则的违反。
  - 但它与审查 034 B2 修复后的交付目标名称规则不一致。
  - 是否需要统一，由产品负责人决定。按 AGENTS.md，本文不自行拍板，只在此提出。
