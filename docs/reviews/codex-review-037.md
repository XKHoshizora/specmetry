# Codex Review 037：PR #17 项目名称代码围栏修复的独立审查

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #17](https://github.com/XKHoshizora/specmetry/pull/17)
- 基准提交：`01bd50ff411986863722359de8aa5dacf509b8cb`
- 目标提交：`a106f9e6780e297c3f0f74c6d24cc42da69da894`
- CI：[GitHub Actions 39015547400](https://github.com/XKHoshizora/specmetry/actions/runs/39015547400)
- CI 实际检出的 PR 合并快照：`947e1d760823c53bbc9c3fb21f542d61778f0c56`
- 范围：项目名称代码围栏修复、两项新增 CLI 测试、相关 Schema、直接相关文档，以及证明拒绝顺序、事务和兼容行为所必需的调用链。不重新全面审查已经获得 GO 的九个 M1 工作包，不重新展开 Review 036 已关闭的六项证据缺口。
- 方法：固定提交差异静态审查、PR HEAD 与 CI 身份核对、隔离副本中的修复前后对照、现有定向测试，以及独立的完整业务事实与文件比对。

**最终结论：GO。PR #17 可以合并，项目名称修复满足 M1 的该项前置条件。**

审查前后核查的 PR HEAD 均与指定目标一致，没有提交漂移。没有把旧提交的测试结果用于未经审查的新提交。

## BLOCKER

无。未发现新的可复现缺陷或影响结论的关键证据缺口。

### 新名称的保护正确

- [validateName](../../internal/project/init.go#L209) 使用 `strings.Contains` 拒绝任意位置的三个连续反引号。
- 显式 `--name` 和默认主 Worktree 目录名都在数据库打开前校验；默认名称非法时提示使用 `--name`。
- 独立验证了开头、中间、结尾及四个连续反引号；空值、无效 UTF-8、换行、控制字符、201 字符仍被拒绝。
- 单个、两个连续反引号，以及中文、日文、Emoji 和 200 个 Unicode 码点仍可使用。
- 非法输入返回 exit 2，不回显名称或 canary；JSON 符合 `error.v1`。合法输出、数据库名称和 Marker 原值一致。

显式名称在 [Prepare](../../internal/project/init.go#L103) 中校验；默认名称在 [主 Worktree 目录名处理](../../internal/project/init.go#L152) 中使用同一校验器。CLI 的 `runInit` 只有在 `Prepare` 成功之后才打开数据库，因此非法新名称不会先创建数据目录、登记项目或写 Marker。

新增 CLI 测试覆盖 Git 和非 Git 项目、显式名称与默认名称、首次初始化和数据目录已存在的情形。默认目录名含围栏时，提供合法 `--name` 可以初始化；并没有错误地将路径本身当作名称而继续拒绝。

### 旧项目的拒绝顺序正确

- [update](../../internal/project/init.go#L474) 首先检查已登记名称，位于改名拒绝、阶段变更、settings 更新和 Worktree 更新之前。
- 校验失败使写事务回滚；Marker 和 exclude 的写入在事务成功后才可能执行。
- 不是提交业务写入后，再因输出 Schema 失败而报错。

[Plan.Apply](../../internal/project/init.go#L298) 在 `DB.WriteTx` 内查找已有项目并调用 `update`。名称检查失败后，事务返回错误，后续输出构造、Marker 和 exclude 写入不会执行。

这也避免了旧的“不改名”错误先引用非法登记名称：新增检查发生在该错误之前，拒绝消息只包含项目 ID 和固定的问题类别，不引用旧名称。

独立补充检查没有只依赖现有测试的部分计数摘要：

- 对所有应用表逐表、逐字段比较，覆盖项目名称、阶段、完整 settings、Worktree 明细及 `last_seen_at`、seq 和其他业务事实。
- 覆盖重复 Init、合法的其他 `--name`、原非法 `--name`、`--stage`、`--skip-sources`、`--skip-inventory`，以及新的关联 Worktree。
- 文本和 JSON 模式均返回 exit 2；JSON 的错误分类和 Pointer 正确，不包含旧名称或运行时生成的 canary。
- Marker、exclude、主仓库及关联 Worktree 的文件和目录快照保持不变。
- 在隔离环境中删除 Marker/exclude 后重复上述拒绝，二者都没有被补写。

上述业务事实比对全部通过，没有静默改名、阶段或 settings 变更，没有登记新 Worktree，也没有额外消耗 seq。

### 辅助文件与业务事实分别判断

本机独立实验只观察到 `specmetry.db-shm` 修改时间变化，内容哈希未变；没有将这一结果表述为数据目录逐字节或所有文件元数据绝对零变化。

旧项目名称检查需要在普通数据库 Open 后读取登记事实。这不等同于数据库版本不兼容拒绝或 doctor 的只读诊断路径。按既有存储行为区分普通连接的辅助文件活动与业务写入，没有据此新增存储契约豁免，也没有放宽版本保护或 doctor 的约束。

### 输出契约正确

- `init.v1/project.name` 同样拒绝三个连续反引号。
- SCH-150 实际参与测试；测试先验证基础示例合法，再应用名称变异并要求失败。
- 合法成功输出仍通过 `init.v1` 校验。
- 没有替换名称为占位符，也没有使输出与数据库或 Marker 事实不一致。
- 没有新增数据库 CHECK 或修改迁移，这一事实已如实披露。

数据库旧行可能仍含围栏名称；当前处理是 Init 有说明地拒绝，而不是篡改数据或输出一个不符合 Schema 的文档。

## NON-BLOCKER

### N1：产品决定记录需要收尾更新

位置：[review-resolution §47](../review-resolution.md#47-项目名称禁止代码围栏的独立安全修复)，特别是其“旧项目的兼容”和状态段落；PR #17 描述中也仍保留“待负责人确认”。

本次用户指令已正式选择方案 1，可据此更新为已批准，无须再次决策。这属于决定状态的文档收尾，不是实现缺少授权或需要重新讨论产品方案。

### N2：旧项目测试的无写入摘要可以加强

位置：[projectname_test.go:198](../../internal/cli/projectname_test.go#L198)。

现有摘要主要比较名称、阶段、Worktree 数量、seq 和 Marker，未直接覆盖 settings、Worktree 全字段及 exclude。生产检查顺序正确，且本轮独立补证已经验证这些边界，因此不阻塞。

建议后续将完整 settings、Worktree 明细和文件不变性断言纳入仓库测试，以避免未来回归只能依靠当前的部分摘要被检测。

### “其他命令不受影响”的核查范围

实际动态运行了 `source list` 和 `target show`，均成功并符合各自 Schema。静态检查了当前其他业务命令使用的 `project.Lookup` 路径，没有发现它们因本补丁进入 Init 的新增拒绝。

没有把两个成功对照扩大为所有命令的完整 E2E 兼容保证。文档关于其他命令的表述应按当前已实现命令和实际核查范围理解；M2 `status.v1` 的名称兼容仍单独处理。

### 既有独立事项不关闭

以下事项继续保留，不因本次 CI 通过而关闭，也不归类为已接受风险：

- Windows 首次创建数据库期间偶发 `Access is denied`。
- doctor 的 GIT_TRACE 环境隔离问题。
- 本地默认测试超时维护事项。

本轮没有修改或修复上述实现。

## PRODUCT DECISION

**方案 1 已由产品负责人在本次审查请求中正式选择，并已正确实现。** 本次审查明确记录其含义：

1. 新登记的项目名称禁止包含三个连续反引号。
2. 已登记名称若违反此规则，Init 返回有说明的 `invalid_input / exit 2`。
3. 不静默改名，不改写旧名称、Marker 或其他业务事实。
4. 接受这类旧项目暂时不能通过 Init 修改阶段、补写 Marker 或登记关联 Worktree 的限制。
5. 本决定仅针对这类名称，不代表所有开发版本数据均不需要兼容。
6. 本轮不增加改名命令，不修改数据库结构或迁移。
7. 不据此关闭其他未解决风险。

没有新的问题需要负责人裁决。固定目标文档中的“待确认”应按 NON-BLOCKER N1 收尾更新，而不是重新要求负责人决定。

M2 `status.v1` 的名称兼容仅作为后续设计约束记录，本轮不实现，也没有推定所有读取接口都必须拒绝旧项目。

## ACCEPTED RISK

仅记录本次明确接受的局部限制：已有代码围栏名称的项目暂时不能通过 Init 修改阶段、补写 Marker 或登记关联 Worktree。

没有将这一取舍扩大为普遍不兼容旧开发数据，也没有将其他安全、可靠性或维护问题一并接受。

## 实际测试与证据边界

### 本机 Windows 隔离运行

目标提交的两项新增 CLI 测试通过：

- `TestInitRefusesACodeFenceInTheProjectName`。
- `TestInitWithARegisteredNameThatHoldsACodeFence`。

`internal/project`、`internal/contract` 直接相关测试通过；Schema 测试检查了 38 个合法示例和 150 个非法反例，包含 SCH-150。

独立补充测试通过：

- `TestReview037LegacyAllFactsAndFiles`：完整应用表数据比对，Marker/exclude 存在与缺失两种状态，主仓库和关联 Worktree 文件不变性，以及拒绝信息不回显动态 canary。
- `TestReview037NewNameBoundaries`：围栏位置、既有名称边界、无状态创建，以及 200 个混合 Unicode 码点的原值保存。

这些补充测试只存在于隔离副本，没有写入原仓库，也没有描述为 PR 已经收录的测试。

本机两个既有符号链接测试因权限不足跳过：

- `TestListFilesSkipsSymlinks`。
- `TestMarkerRefusesSymlinkedDirectory`。

Ubuntu、macOS CI 中对应测试实际通过，没有将本机跳过描述为通过。这些测试不涉及本补丁改变的名称校验逻辑。

### 修复前后对照

在固定基准的隔离副本中，复制并运行相同两项新增 CLI 测试，二者均失败，实际复现了：

- 围栏名称被成功登记，并写入数据库和 Marker。
- 旧登记名称进入输出或“不改名”的错误消息。
- 旧项目阶段和 Worktree 发生改变，seq 增加。

目标提交上的对应测试通过。这证明测试能够检测原缺陷，不只是当前实现上的正常路径冒烟。

没有将基准测试中因首个失败而未继续执行的分支冒称为逐一完成的动态复验；其余拒绝位置和边界由目标测试及静态控制流共同核查。

### CI 身份与执行记录

已核查 [CI 39015547400](https://github.com/XKHoshizora/specmetry/actions/runs/39015547400)：

- 首轮 attempt 1，8/8 success。
- CI 元数据的目标为 `a106f9e6780e297c3f0f74c6d24cc42da69da894`。
- 实际检出 `refs/remotes/pull/17/merge`，快照为 `947e1d760823c53bbc9c3fb21f542d61778f0c56`。
- 合并快照的父提交分别为指定基准 `01bd50ff411986863722359de8aa5dacf509b8cb` 和指定目标 `a106f9e6780e297c3f0f74c6d24cc42da69da894`。
- 合并快照与目标提交的树均为 `13ee7361136ee77c02b2ac0e997ddc1a653781f4`。
- 三平台均实际运行两项新增 CLI 测试及 Schema 反例测试并通过，没有跳过这些本轮关键测试。
- Ubuntu 启用 Race Detector。

跨平台结论来自 CI 日志核查；本机独立运行环境为 Windows，两类证据未混同。没有借本次 CI 结果重新关闭旧版 Git、跨用户权限或其他范围外验证事项。

## 范围控制与最终回答

差异核查确认没有修改 Git ref、Target 名称或 Source、Proposal、Review、Question、Scope 的其他行为，没有修改 SQLite 迁移、Windows 首次建库竞争或 doctor GIT_TRACE 的实现。

| 问题 | 结论 |
|---|---|
| 是否有 BLOCKER？ | 无。新旧名称拒绝及无业务写入边界有源码和独立复验证据。 |
| 产品负责人方案 1 是否正确实现？ | 是。拒绝旧围栏名称，不静默改名，不提交业务变更，不新增迁移。 |
| 是否有直接引入的安全、隐私或一致性回归？ | 未发现。 |
| PR #17 是否可以合并？ | **可以，GO。** |
| 项目名称修复是否满足 M1 的该项前置条件？ | 是。 |
| 是否仍有需要负责人决定的问题？ | 没有。仅需将现有“待确认”记录更新为本次已批准的方案 1。 |

**最终结论：GO。该结论不等于 M1 整体验收 GO。**

审查结束时原仓库干净，未修改、提交、推送或合并，未启动 M2。本文仅在随后收到用户明确保存要求后新增；未修改其他文件，未创建提交，未推送或合并。
