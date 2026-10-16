# Codex Review 026：WP1.6 最终定向独立复审

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #9](https://github.com/XKHoshizora/specmetry/pull/9)
- 目标提交：`6f12c124290f1849a145498212cae2c48cab8397`
- 增量基准：`fa58b4420e743521b027f395b921bc178a4d26fb`
- 上轮报告：[Codex Review 025](codex-review-025.md)
- CI：[GitHub Actions 38780643544](https://github.com/XKHoshizora/specmetry/actions/runs/38780643544)
- 审查范围：仅复核 Review 025 的 B1、B2 及直接回归，不重新进行 WP1.6 全量审查。
- 验证方式：增量及必要调用链静态审查、既有三平台 CI 日志核查，以及仓库外隔离副本中的旧版与新版反例对照、定向回归测试。未使用真实项目数据。

**Codex Review 026：GO。Review 025 的 B1、B2 均可关闭。** 未发现本轮修改引入的实质性 BLOCKER，WP1.6 可以通过代码验收，PR #9 可以合并。

审查目标为 `6f12c124290f1849a145498212cae2c48cab8397`，增量基准为 `fa58b4420e743521b027f395b921bc178a4d26fb`。本轮仅检查两项修复及直接回归。

## BLOCKER

无。

### B1：敏感参数回显——关闭

修复在 CLI 和内部接口两层生效：

- `proposal show` 在查询前校验 ULID，格式错误使用固定消息。
- [ShowDocument](../../internal/proposal/read.go#L355) 的 NotFound 消息不再包含传入参数，直接调用也无法绕过。
- `review.Check` 的 Proposal NotFound 同样使用固定消息。
- [baseline history](../../internal/cli/baseline.go#L105) 自行解析 `--since`，不再输出包含原值的整数 flag 解析错误。
- `ulid.Valid` 与 `common.v1` 的现有 ULID 格式约束一致；独立测试比较了长度、逐位置字节和 Unicode 等输入。

独立验证确认：

- 合成 GitHub Token、合法但不存在的 ULID，在文本和 JSON 错误中均不回显。
- 直接调用 `ShowDocument`、`review.Check` 不泄露传入值。
- 敏感、负数及溢出的 `--since` 参数返回退出码 2；JSON 模式正确输出 `invalid_input`。
- 未发现业务表变化，数据目录字节扫描未发现合成 Token。
- 正常 Proposal 和历史查询的回归测试通过。

### B2：过期审核错误分类——关闭

[review.Check](../../internal/review/review.go#L118) 现在首先调用 `CheckVersion`。首次检查、必定回滚的预演、确认后的正式提交，均通过这一入口执行事务内版本检查；`baseline.Apply` 保留第二道检查。

Schema 和密钥扫描仍按既定流程前置执行；进入审核事务后，版本检查优先于冲突、条目状态、编辑内容及 Followup 等业务验证。

独立验证结果：

- **审核前已过期：** 无论 `acknowledged_conflict` 是否为 true，都返回退出码 5、`stale_precondition`、当前版本及 `/seen_base_version`，不出现确认提示。
- **确认期间版本前进：** 另一独立连接先提交后，旧审核确认时返回退出码 5，没有覆盖或额外写入。
- **版本当前但冲突未确认：** 仍正确返回退出码 2。
- **两个审核同时竞争提交：** 结果为 `[0, 5]`，只生成一个版本、一个正式编号和一个新增 seq。

针对确认期间的反例，补充测试使用了非空 Source Ref、Code Ref，并比较相关业务表的**完整内容**；旧审核结束后，与第一份审核提交后的状态完全一致。

## 测试证据与有效性

将目标提交的三个新增回归测试移入基准提交的隔离副本，确认它们均能触发原缺陷：

| 验证项 | 基准提交 | 目标提交 |
|---|---|---|
| 参数回显与错误输出 | 泄露参数，测试失败 | 通过 |
| 审核前已过期 | 未确认冲突时错误返回 2 | 通过 |
| 确认期间版本前进 | 确认后错误返回 2 | 通过 |
| 上一轮独立反例 | 复现原问题 | 全部通过 |
| 独立双连接并发审核 | 通过 | 通过 |

新增测试经过 CLI 命令入口、参数解析和错误输出流程。确认期间测试通过确认输入读取边界，确定性地插入另一连接的提交，前置条件有效。它使用的是**同进程中的独立数据库连接**，不能称为多进程测试，但足以验证本次事务间窗口。

已独立核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38780643544)：8/8 成功，使用合并提交 `2fccbe6`，对应本轮目标与基准。三个新增测试在 Windows、Ubuntu、macOS 均实际运行通过，Ubuntu 启用了 Race Detector：

- `TestProposalShowDoesNotEchoItsArgument`
- `TestReviewIsStaleBeforeTheConflict`
- `TestReviewIsStaleAfterTheConfirmation`

本地隔离验证也通过了定向 CLI 回归、ULID 一致性和共享 Schema 测试。未发现代码死锁或无限等待证据，不要求调整全局测试超时。

## NON-BLOCKER

继续保留 Review 025 的两项记录，本轮不升级：

- **N1：** 确认预览可能显示 `editorial`，实际因强制字段升级为 `material`。
- **N2：** Baseline History 尚未展示已保存的 Followup rationale。

未发现新的直接回归。

## ACCEPTED RISK

沿用既有批准边界，包括通道识别仅防误操作、M1 的 `hash_only` 策略及既定后续工作包范围。未接受新的安全风险，也不需要额外产品决策。

## 最终结论

| 最终问题 | 结论 |
|---|---|
| B1 是否关闭？ | **关闭。** |
| B2 是否关闭？ | **关闭。** |
| 旧版失败、新版通过的反例是否有效？ | **有效，已独立对照验证。** |
| 是否发现直接回归？ | **未发现实质性问题。** |
| WP1.6 是否可以正式验收？ | **可以，代码审查 GO。** |
| PR #9 是否可以合并？ | **可以合并。** |

该结论不代表整个 M1 或后续里程碑已经验收。原仓库在复审期间保持干净，未修改、提交、推送或合并，未开始 WP1.7。本文件为复审完成后按用户要求保存的完整复审意见。
