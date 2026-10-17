# Codex Review 035：WP1.8 最终定向独立复审

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #14](https://github.com/XKHoshizora/specmetry/pull/14)
- 审查目标：`7f654088b0df8c5cca1e2d81bca295e8143b8c89`
- 修复提交：`f0dab526b9b927ad6d1476012eb391e0c6abf8eb`
- 修复前基准：`90beb48d3c5cc2d8b7a7cbbc045b0925ab793ead`
- CI：[GitHub Actions 38957016420](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420)
- 定位：仅复核 Review 034 的 B1、B2 及直接回归，不重新展开 WP1.8 全量审查。
- 方法：实际差异及必要调用链静态审查、隔离副本中的修复前后对照、独立 CLI 输入边界测试、既有定向回归，以及三平台 CI 元数据和日志核查。

**最终结论：GO。两项 BLOCKER 均可关闭，未发现新的实质性阻塞问题。**

## BLOCKER

无。

### B1：交互式 Scope 备注——已关闭

- [target.go:368](../../internal/cli/target.go#L368) 区分省略备注与显式空字符串，并在人工确认前将交互式输入构造成完整 `scope.v1`，使用与文件输入相同的 Validator。
- [confirm.go:32](../../internal/scope/confirm.go#L32) 使用可选备注；[confirm.go:179](../../internal/scope/confirm.go#L179) 正确保留 SQL NULL。

独立验证结果：

| 输入 | `--file` 与 `-m` 的结果 |
|---|---|
| 多行代码围栏、行内代码围栏 | exit 2，包含 `/note` 错误定位 |
| 1001 Unicode 字符、显式空备注 | exit 2，在人工确认前拒绝 |
| 恰好 1000 Unicode 字符，包括 Emoji | 成功，按原值保存 |
| 普通中文、日文、Emoji | 成功，按原值保存 |
| 完全省略备注 | 成功，保存 SQL NULL |

非法输入没有显示确认提示、没有读取确认输入，没有改变业务事实或 seq；文本错误和完整 `error.v1` 均未回显合成原文。密钥输入仍被拒绝，数据目录检查未发现拒绝内容。

审核附带 Scope 继续使用 `review.v1/confirmationInReview`，不要求独立确认中的 `baseline_version`；合法备注保存、缺省备注及 `/scope/note` 拒绝路径没有发现回归。重复 Requirement 提前由 Schema 拒绝，仅改变诊断指针，不改变拒绝语义。

具体而言：

- `scope.Input.Document()` 补齐独立确认需要的 `schema`、Target、Baseline Version、Source Set Hash、Requirement 清单、attestations、豁免数组和前次确认 ID。
- 未提供 `-m` 时省略 `note`；显式 `-m ""` 保留空值并由 Schema 拒绝，不再被当成未提供备注。
- 审核中的 Scope 没有调用独立文档校验来错误要求 `baseline_version`；仍由审核文档的条件约束校验，提交时绑定审核完成后的版本。
- `Note` 从 `string` 调整为 `*string` 后，写入处区分原值与 SQL NULL，读取模型原有的可选备注行为保持正确。
- 两种输入模式的原始业务内容仍在持久化前接受密钥扫描。文件输入先进行的 Schema 校验沿用脱敏错误处理，不回显尚未通过扫描的文本。
- 重复清单仍返回 exit 2；错误指针从业务校验的 `/requirements/1` 变为 Schema 的 `/requirements`，属于诊断位置调整。

### B2：Target 名称——已关闭

三处约束一致地禁止三个连续反引号：

- [target.go:69](../../internal/scope/target.go#L69)：`CheckName`。
- [target.go:93](../../internal/scope/target.go#L93)：`SetTarget` 在分配 seq、归档旧目标之前再次校验。
- [target-scope.v1.schema.json:34](../../schemas/target-scope.v1.schema.json#L34)：输出 Schema。

显式名称、同 ref 改名、ref 自动生成名称均受到保护。非法名称返回 exit 2、指向 `--name`，没有提示或写入；旧 Target、Scope 和 seq 保持不变。自动名称不合法时，提供合法 `--name` 可以设置该 ref。

中文、日文、Emoji、单个及两个连续反引号仍可正常使用；既有长度、UTF-8 和控制字符限制保持有效。

`SetTarget` 的再次校验发生在查询及更新活动目标、调用 `NextSeq` 之前，因此不是只依靠 CLI 的校验。输出 Schema 的新增代码围栏约束与写入校验在本次修复的规则上保持一致。

## NON-BLOCKER

### N1：保留 Review 034 的跨项目 Requirement 外键测试覆盖不足

位置：[scope_test.go:87](../../internal/store/sqlite/scope_test.go#L87)。

测试仍可能由另一条外键先拒绝。此前实际约束已独立验证正确，本轮不重新升级。

建议按既有维护记录，构造属于项目 B 的真实 Requirement，并由项目 A 的合法 confirmation/project 引用它，以单独验证 Requirement 的复合外键。此次补丁没有修改迁移或外键实现。

### N2：Windows 数据库创建并发可靠性问题需独立跟踪

位置：[openrace_test.go:155](../../internal/store/sqlite/openrace_test.go#L155)。

已核查首次 CI 日志：`TestCheckWhileTheDatabaseIsCreated` 在 round 17、check 11 返回 `Access is denied`。本次增量未修改存储层、数据目录检查、迁移或该测试，未发现与 B1/B2 的直接关联。

重跑通过不能证明根因消失，也不能据此一般性忽略权限错误；应作为独立的 Windows 数据库创建并发可靠性问题另行定位。本轮不修复，不作为 WP1.8 的 BLOCKER，也不将它归类为已接受风险。

## ACCEPTED RISK

未新增接受风险。

真实豁免生命周期及完整交付状态推导仍按已批准边界安排在 M2。此前已经独立审查通过的 V-T01、V-T02、Review + Scope 原子性、Target 并发、迁移和安全确认没有因本轮修复被重新展开为全量审查。

## 测试证据与验证边界

### 修复前后对照

在隔离的修复前副本中，运行目标提交新增的相同 CLI 回归测试，复现了原始缺陷：

- 交互式代码围栏备注返回 exit 0 并写入。
- 1001 字符备注进入人工确认后才返回 exit 1。
- 显式空备注被交互式入口接受。
- 围栏 Target 名称创建或归档目标。

目标提交中的 `TestScopeNoteIsCheckedLikeAFile` 和 `TestTargetNameHasNoCodeFence` 均通过。对照证明新增测试能够发现修复前的问题，而不是仅证明正常命令可以运行。

### 本机隔离验证

本机环境为 Windows。所有测试及独立补充断言均在隔离副本和临时项目、数据目录中运行。

通过的定向 CLI 测试包括：

- `TestScopeNoteIsCheckedLikeAFile`。
- `TestTargetNameHasNoCodeFence`。
- `TestScopeInReview`。
- `TestTargetShowAndSet`。
- `TestScopeConfirm`。
- `TestScopeConfirmedMeanwhile`。
- `TestTargetAndScopeNeedAPerson`。
- `TestTargetAndScopeConfirmationNotShownWhole`。
- `TestConfirmationWithJSON`。
- `TestHumanOnlyCommandsConfirmThroughConfirm`。

独立补充检查进一步覆盖：

- 非法备注经文件、`-m`、文本及 JSON 错误模式提交时，均在提示前拒绝，确认输入读取次数为 0。
- 对整个 JSON stdout 解码并检查 EOF，验证 `error.v1`、`invalid_input`、`/note` 和空 stderr。
- 拒绝前后比较 Target、Scope、Baseline、Proposal、Source、Question 等相关业务状态及 seq。
- 检查原始合成 canary 不进入输出或数据目录。
- 使用包含非 BMP Emoji 的 1000 个 Unicode 码点备注，确认 SQL 原值与长度。
- 直接检查 `note IS NULL`，确认省略备注没有被保存为空字符串。
- 审核附带合法备注时，确认原值及审核后 Baseline Version 正确保存。
- 文本模式下非法 Target 名称在提示前拒绝，输入读取次数为 0，错误不回显名称。

`internal/scope`、`internal/contract`、`internal/termtext`、`internal/channel` 的测试通过，包括 V-T01 合成非空集合、V-T02 独立连接竞争、写入层名称校验，以及 Writer/Guard 的短写入与错误传播。

Schema 检查通过 38 个合法示例和 149 个非法反例，覆盖 SCH-140～149；非法反例测试先验证未修改的基础示例有效，再应用变异并要求拒绝。

本轮没有重新执行全套 WP1.8 首次审查，也没有将独立补充测试描述为仓库已经收录的测试。

### GitHub CI

已核查 [CI attempt 2](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420/attempts/2) 的元数据和相关日志：

- 八项任务全部对应目标提交 `7f654088b0df8c5cca1e2d81bca295e8143b8c89`，结论均为 success。
- Windows、Ubuntu、macOS 均实际运行新增的两项 CLI 回归测试并通过，没有跳过。
- 三平台相关 Scope、Target、审核、确认和 Schema 测试均有实际通过记录。
- Ubuntu 测试命令使用 `-race`。
- attempt 1 的 Windows 创建竞争失败按上文独立保留，没有因最终绿色而抹去。

跨平台结论来自 CI 日志核查；本机独立运行环境为 Windows，没有将两者混同。

## 最终判断

| 问题 | 结论 |
|---|---|
| Review 034 B1 能否关闭？ | 可以关闭。交互式备注与文件输入均执行 `scope.v1` 校验。 |
| Review 034 B2 能否关闭？ | 可以关闭。CLI、写入层及输出 Schema 一致禁止代码围栏名称。 |
| 两种 Scope 输入模式是否符合相同隐私与 Schema 规则？ | 是。非法备注确认前拒绝，合法原值保存，省略备注为 SQL NULL，密钥扫描保持有效。 |
| Target 名称写入校验和输出 Schema 是否一致？ | 本次修复的代码围栏限制一致；既有名称限制保持有效。 |
| 是否存在本次修改直接引入的回归？ | 未发现实质性安全、数据一致性或确认流程回归。 |
| WP1.8 是否可以正式验收？ | 可以通过本次 WP1.8 代码验收。 |
| PR #14 是否可以合并？ | 可以合并。 |

**最终结论：GO。**

这不代表 M1 整体验收完成；M1 的最终退出条件仍由产品负责人另行确认。

审查前后原仓库均干净，审查过程未修改文件、提交、推送或合并。本文仅在随后收到用户明确保存要求后新增，未修改其他文件，未创建提交，未推送或合并，也未启动 M2。
