# Codex 审查 020：WP1.5 需求提案与引用校验首次独立代码审查

- 审查日期：2026-10-15
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #8](https://github.com/XKHoshizora/specmetry/pull/8)
- 审查目标：`2abca0014a43b5abf5e67d9b587077b0caeeb480`
- 差异基准：`35d060f5518d7c3abcd4791377cf7b9fde719e3e`
- CI：[Actions run 38607934336](https://github.com/XKHoshizora/specmetry/actions/runs/38607934336)
- 范围：WP1.5 的 propose、Proposal 校验、Source Ref、Code Ref、持久化、JSON Schema、Git 和 Worktree 变更及必要回归。
- 边界：不重新审查 M0 或此前已验收的工作包，不要求提前实现 WP1.6、M2 或 M3。
- 操作记录：审查和反例均在只读原仓库及隔离临时环境中完成；本报告随后按用户要求单独保存，未修改其他仓库文件，未提交、推送或合并。

**结论：NO-GO。** 在目标提交 `2abca001` 中确认了 5 项 BLOCKER，修复前不建议验收 WP1.5 或合并 PR #8。

已核实[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38607934336) 8/8 成功，测试的合并提交 `c5a94b120ad5a60ab7cee7a695bc2d06f8625ca0` 对应目标提交及基准 `35d060f`。以下问题来自实际实现和隔离环境中的独立反例。

## BLOCKER

### B1：Code Ref 检查可能触发隐式 Fetch、执行外部程序

位置：[gitx.go:251](../../internal/gitx/gitx.go#L251)、[gitx.go:324](../../internal/gitx/gitx.go#L324)。

新增的 `cat-file blob` 调用没有禁止 promisor 对象的按需抓取。现有环境隔离、关闭 FSMonitor 和 optional locks，不能阻止这种行为。Git 官方文档也明确说明缺失对象会通过 Fetch 补取。[Git Partial Clone 文档](https://git-scm.com/docs/partial-clone)

**独立反例：**在临时仓库删除一个已提交 blob，配置 promisor remote，并将 upload-pack 指向只写测试标记的本地程序。通过公开 `propose` 引用该文件：

- upload-pack 程序实际执行；
- 项目之外出现测试标记文件；
- 随后 `propose` 才返回退出码 6。

测试没有连接真实网络。但相同的隐式抓取机制在网络远端下可能产生出站请求，违反核心无出站请求和 Git 只读承诺。

**最小修复：**对所有相关 Git 对象读取可靠地禁止 lazy fetch；无法保证时保守拒绝。须验证 Git 2.32 最低版本边界，不能假设新控制选项在旧版本中有效。补充缺失 blob/tree、外部 helper 不执行、仓库文件不变的回归测试。

### B2：hash_only 数据库防线没有保护 `final_payload_json`

位置：[0004_proposals.sql:186](../../internal/store/sqlite/migrations/0004_proposals.sql#L186)、[0004_proposals.sql:221](../../internal/store/sqlite/migrations/0004_proposals.sql#L221)。

现有隐私触发器只检查 INSERT 时的 `sources_json`，没有检查 `final_payload_json`。这与 data-model §4.5 明确要求该字段中的引用也不得保存片段原文不一致。

**独立反例：**

1. 登记真实来源，通过公开 `propose` 验证并提交合成片段。
2. 确认此时数据目录没有片段原文。
3. 使用普通 `WriteTx` 更新待审条目的 `final_payload_json`，放入该来源的 `sources[].excerpt`。
4. 事务成功，数据目录文件中可以找到 canary。

**当前 propose CLI 的原文剥离是有效的。** 此问题是已声明的数据库第二道防线不完整，并非证明 Agent 可以直接通过当前 CLI 写入该字段。

**最小修复：**补齐 INSERT、UPDATE 对 `final_payload_json` 引用原文的 hash_only 约束，并增加数据库及 WAL canary 测试。不需要提前实现 WP1.6 审核功能。

### B3：合法的 `goal: null` 被错误拒绝

位置：[proposal.go:84](../../internal/proposal/proposal.go#L84)、[store.go:278](../../internal/proposal/store.go#L278)。

`stringData` 把 JSON `null` 解码到普通字符串后，返回 `("", true)`。后续将空字符串当作正式 Goal 编号查询。

**独立反例：**提交符合 `proposal.v1` 的新增 Requirement，包含正常的标题、类型、优先级，以及 `"goal": null`。Windows、Linux 均返回退出码 2，错误指向 `/items/0/data/goal`，称不存在对应 Goal。

这使 Schema 明确允许的“未关联 Goal”输入无法提交，也影响更新时清空 Goal 关联。

**最小修复：**区分字符串与 JSON null；null 保留在提案 payload 中，但不执行正式编号查找。补充新增和更新两种公开 CLI 测试。

### B4：行号整数溢出会被当成“没有行号限制”

位置：[proposal.go:260](../../internal/proposal/proposal.go#L260)、[coderef.go:66](../../internal/proposal/coderef.go#L66)。

`lineRange` 在 `strconv.Atoi` 溢出时返回 `ok=false`；调用方将其解释成未提供行号，直接令已存在文件的 `exists=true`。

**独立反例：**对一个只有一行的已提交文件，分别引用：

- 30 个 `9`；
- 30 个 `9` 后接 `-1`；
- `1-` 后接 30 个 `9`。

这些输入通过 Schema，并在 Windows、Linux 的公开 `propose` 输出中被标记为 `exists=true`，其中倒置范围也未被拒绝。

这违反 V-P08 的位置存在性标记，不涉及将其升级为 AC 验收证据。

**最小修复：**明确区分“未提供行号”和“解析失败”；溢出必须安全拒绝或判定不存在，绝不能转成无限制检查。补充整数边界和范围端点溢出测试。

### B5：非普通来源文件可能令提案永久阻塞，取消无效

位置：[excerpt.go:69](../../internal/proposal/excerpt.go#L69)。

代码先 `os.Open`，随后才检查文件是否普通文件。Unix FIFO 在没有写端时会阻塞于 Open，尚未执行到类型检查或后续 Context 检查。

**独立反例：**

- 将已登记的普通来源替换成 FIFO，再通过公开 `propose` 引用，命令阻塞。
- 对 `CheckExcerpts` 设置 25ms Context 超时，超过 300ms 仍无返回；必须额外打开 FIFO 写端才能解除阻塞。

这不符合非普通来源应保守处理及读取可取消的要求。

**最小修复：**在可能阻塞的打开操作之前排除特殊文件，并采用不会因 FIFO 等类型阻塞的打开策略。增加来源类型替换及取消期间无部分结果的测试。

## NON-BLOCKER

### N1：跨分块与读取中取消测试覆盖不足

[excerpt_test.go:133](../../internal/proposal/excerpt_test.go#L133) 最多生成 300,000 个、每个最多三字节的字符，文件不会达到 1 MiB；因此没有覆盖注释声称的跨分块场景。现有取消测试也只在调用前取消。

本轮补充的真正跨 1 MiB Unicode、匹配后出现 NUL/截断 UTF-8、普通文件读取中取消测试均通过。建议将这些有效前置条件纳入仓库测试。

### N2：Git 符号链接测试存在失效前置条件

[gitx_test.go:294](../../internal/gitx/gitx_test.go#L294) 创建了仅存在于 Index 的链接条目，但随后 `repo.Commit` 执行 `git add -A`，会移除工作目录中不存在的该条目。

建议在检查前断言提交树确实包含 `120000`、`160000` 模式。本轮使用真实保留这两种树条目的提交独立验证，类型筛选正确。

## ACCEPTED RISK

- 引用结果绑定读取时的 `source_sha256` 和 `checked_head`，不保证文件系统直到数据库提交都保持不变。本轮改变文件与 HEAD 后再存储，确认旧结果没有被错绑到新对象；来源在读取后被移除时，存储会拒绝且不追加事实或 seq。
- 正式审核、Goal/Requirement/AC 编号分配仍属于 WP1.6；M2 状态推导、M3 Agent 集成不作为本轮阻塞项。
- verified 仅表示片段匹配，Code Ref exists 仅表示位置存在；当前提案没有自动成为正式基准或验收结果。

## 最终回答

| 审查问题 | 结论 |
|---|---|
| V-P01～V-P08 | 主要校验已实现，但 B3、B4、B5 影响当前正确性 |
| hash_only 隐私 | 公开 propose 的原文剥离及输出脱敏通过；数据库保证受 B2 阻塞 |
| Code Ref 与 Worktree 绑定 | 正常 HEAD、分支隔离、普通文件、链接、子模块、空文件及无提交场景通过；B1、B4 必须修复 |
| 事务、并发与版本前置条件 | 已验证过期拒绝、首次 Worktree 登记回滚；六个真实进程的 seq、Q 编号及关联正确，未发现正式 Baseline 被修改 |
| proposal-receipt.v1 | 13 个新增反例及正常 pending 输出通过；B4 仍会产生 Schema 合法但语义错误的标记 |
| 六项实现细化 | 已落地；恰好 8 MiB 接受，超出一字节拒绝且 seq 不变。可取消性和只读 Git 保证仍需修复 |
| 验收并合并 PR #8 | **暂不可以，须先关闭上述 BLOCKER** |

Windows 独立测试中，`proposal`、`baseline`、`contract`、`gitx`、`sqlite` 包通过；CLI 包运行达到默认十分钟总时限，**不能报告整包通过**。WP1.5 相关测试在此前已通过；被中断的既有测试单独重跑通过。macOS 与 Ubuntu Race Detector 的结果依据实际 CI 日志，未冒充本地复验。

**最终代码审查：NO-GO。** 审查期间原仓库保持未修改，未提交、推送或合并。本文件为审查完成后按用户要求保存的完整报告。
