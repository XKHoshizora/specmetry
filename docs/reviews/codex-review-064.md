# Codex Review 064：PR #28 成功 JSON 输出完整 Schema 校验

## 结论

**技术结论：GO。BLOCKER：0。M-7 的本轮修复要求已满足。**

五种成功 JSON 调用均实际进入对应 Schema 的完整文档校验，两处参数位置对照比较的是等价调用。隔离副本中的错误 Schema 和反转比较变异均被检出。未发现本次三文件修改引入的直接回归。

**合并前审查及 CI 条件已满足，PR #28 可以由负责人合并。** 审查结束时重新读取平台结果，指定 CI `39299026564` 为 attempt 1，正式 `completed / success`，8/8 Job 成功；三平台均实际通过两项本轮关键测试。M-7 仍不能在合并前登记关闭；实际合并结果核验通过后，可以按本次审查结论关闭。

本结论不构成 WP2.3 或 M2 验收，不授权后续工作包。

## 一、身份、范围与只读边界

- 审查日期：2026-10-20（Asia/Tokyo）。
- PR：[XKHoshizora/specmetry#28](https://github.com/XKHoshizora/specmetry/pull/28)。
- 固定 base：`5985b319ac4cdb23a8f9cbc9b40653c154257fec`。
- 固定 HEAD：`a482dacb1bd61cd3b583cffcc519b1d205e18dc1`。
- HEAD 的 tree：`66c43cae1833cad4f8a4d26d3e9af661f0da6351`。
- 实际单一增量提交：`a482dac`，`test: check successful JSON outputs against their schemas (M-7)`；父提交即固定 base。
- 审查开始及结束的远程核查均为上述 HEAD/base；PR 为 Open、未合并、`auto_merge = null`。结束核查时间为 2026-10-20 03:12 JST（2026-10-19 18:12 UTC）。结论不延伸到其他提交。
- CI 合并快照：`d454091c3d464f701a9ffe6f264c2714d90d8bc7`。GitHub Git commit 元数据确认第一父为固定 base、第二父为固定 HEAD，tree 与 HEAD 完全相同。

实际差异只有以下三个文件，共 50 行新增、6 行删除：

| 文件 | 本轮变化 |
|---|---|
| [internal/cli/cmdline_test.go](../../internal/cli/cmdline_test.go) | 五种成功调用传入具体 Schema；增加两处等价调用的输出比较 |
| [internal/contract/contract_test.go](../../internal/contract/contract_test.go) | 增加 `TestValidateJSONTakesOneWholeDocument` |
| [docs/m1-exit-evidence.md](../m1-exit-evidence.md) | 仅修改 §9 的 M-7 行，登记修复、待独立审查及合并前不关闭 |

没有生产代码、Schema、迁移、冻结规则或工作包范围变化。已读取适用的根 [AGENTS.md](../../AGENTS.md)；相关子目录没有更细的 AGENTS.md。

原始事项来自 [Review 029 N1](codex-review-029.md#non-blocker) 和 [review-resolution §36](../review-resolution.md)：成功输出只以 `{` 开头不足以证明符合业务 Schema。此前没有据此认定生产输出错误，本轮也没有新增或批准 Claim 输出契约。M-7 的当前登记正确保留“待独立审查”“合并前不关闭”。M1 证据报告 §6 的旧描述属于其明确标注的历史核对快照，不能用来否定当前 §9 的修复进度，也不应为本轮修复改写历史审查原文。

测试使用固定 HEAD 的 `git archive` 导出到仓库外隔离目录；变异只发生在另外三份导出副本。原工作区起始干净，未切换分支、修改实现或操作 Git 引用；本轮只新增本报告，不提交、推送、合并或改基。

## 二、实际校验链和五种调用

### 1. Schema 参数确实生效

已追踪完整链路：

`TestCommandLineStillWorks` → `specmetry` → `execute` → 公开 `Execute` → 收集完整 stdout → `schemas()` → `contract.ValidateJSON` → JSON 解码及指定 Schema 验证。

- [cmdline_test.go:148](../../internal/cli/cmdline_test.go#L148) 的五行参数均提供非空 Schema 名。
- [source_test.go:76](../../internal/cli/source_test.go#L76) 的 `specmetry` 在退出码为 0、Schema 非空时无条件校验完整 `stdout`。成功路径不依赖对参数中 `--json` 的字符串识别。
- Validator 创建失败或校验失败均调用 `t.Fatal` / `t.Fatalf`，确实使测试失败。若命令异常退出，调用处仍有 `ExitOK` 断言，不会以合法错误信封冒充成功。
- [cli_test.go:19](../../internal/cli/cli_test.go#L19) 调用的是公开 `Execute`，不是直接调用业务读取函数；但这是进程内测试，不能据此声称重新证明了操作系统 TTY 或真实二进制退出行为。
- [contract.go:85](../../internal/contract/contract.go#L85) 对完整字节序列调用 `jsonschema.UnmarshalJSON`，随后按名称取已编译 Schema 并调用 `Validate`。Schema 参数没有被忽略。

### 2. 映射及等价对照

| 成功调用 | 实际 Schema | 参数位置对照 |
|---|---|---|
| `source list --json` | `source-list.v1` | 无 |
| `proposal list --json --pending` | `proposal-list.v1` | 无 |
| `question list --json --all` | `question-list.v1` | 无 |
| `--json question list` | `question-list.v1` | `question list --json` |
| `question --json list --all` | `question-list.v1` | `question list --json --all` |

这三个 Schema 分别要求 `specmetry.source-list/v1`、`specmetry.proposal-list/v1`、`specmetry.question-list/v1` 的判别字段及各自业务结构。两处对照均保持相同的命令、过滤参数和 JSON 模式，只移动 JSON flag 的位置。两边都独立经过 Schema 校验、要求成功退出，并比较整个 stdout；不存在仅校验一边或错把不同过滤条件当作等价调用的问题。

原有帮助、裸命令组、版本输出、可信拼写建议和退出码断言均保留，没有为了通过 Schema 校验删去原行为检查。原先没有要求成功 stderr 为空，本轮也没有把新增 Schema 检查扩大表述为 stderr 的穷尽验证。

## 三、完整文档校验的证明范围

[TestValidateJSONTakesOneWholeDocument](../../internal/contract/contract_test.go#L192) 使用仓库内合法的 `question-list.json`，先验证原文通过，再测试派生输入，前置条件有效。

| 输入 | 预期及独立运行结果 | 实际证明 |
|---|---|---|
| 合法原文 | 通过 | 示例及指定 Schema 可用，负例不是建立在无效基础上 |
| 合法原文后加换行、空格、CRLF、TAB | 通过 | 允许 JSON 文档尾随空白 |
| 尾随 `x`、逗号，或前置 `x` | `invalid_input / exit 2` | 非法 JSON 前后垃圾不能被忽略 |
| 尾随 `{}`、第二份原文、`null` | `invalid_input / exit 2` | 不能只解码第一份文档后接受剩余内容 |
| 用 `[`、`]` 包住原文 | `invalid_input / exit 2` | 这是合法 JSON 数组，由要求根对象的 Schema 拒绝；不能把此例称为 JSON 语法失败 |

已核对本机使用的固定依赖 `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` 的 `loader.go`：先 `Decoder.Decode`，之后调用 `Decoder.Token`，只有返回 `io.EOF` 才成功。因此完整文档要求来自实际解码链，并非依靠首字符、`json.Valid` 或仅取第一段 JSON。Schema 验证在解码成功后进行，二者承担不同约束。

负例检查明确要求 `ExitInvalidInput`，不是只要求任意错误；正例、尾随空白例也逐一要求无错误。没有发现新增测试只运行而不检查结果的情况。本轮证明的是这些完整文档与业务结构约束，不扩张为所有 JSON 解析边界、所有错误输出或尚未批准契约的覆盖。

## 四、M-7 搜索及未检出变异

独立搜索了测试中的 `HasPrefix`、`json.Valid`、首字节检查及显式空 Schema 的 JSON 调用，没有发现本事项范围内仍以 `{` 前缀接受成功 JSON 的遗漏。

作者列出的空 Schema 调用分类核对如下：

| 位置 | 实际用途 | 本轮裁定 |
|---|---|---|
| [m1exit_test.go:263](../../internal/cli/m1exit_test.go#L263) | 密钥输入拒绝，另有 `wholeJSON` / `errorV1`、退出码及无泄露断言 | 错误路径 |
| [propose_test.go:879](../../internal/cli/propose_test.go#L879) | 拒绝的提案，检查 exit 2 及片段不泄露 | 错误路径 |
| [propose_test.go:1055](../../internal/cli/propose_test.go#L1055) | 人工触发事务失败，检查失败与回滚 | 错误路径 |
| [review_terminal_test.go:34](../../internal/cli/review_terminal_test.go#L34) | 非终端拒绝，检查 human_required | 错误路径 |
| [confirm_test.go:209](../../internal/cli/confirm_test.go#L209) | 对原本仅返回文本的成功命令，反向拒绝意外 JSON 或空文本 | 不是接受成功 JSON 的弱断言 |

这些路径不构成本事项遗漏；本轮没有要求统一重做所有错误测试。已有读取助手及 Init 的成功输出也有具体 Schema 校验，不能因另一测试使用文本前缀就认定成功 JSON 仍未校验。

### NON-BLOCKER N1：本夹具不能区分默认列表与 `--all`

[questionsProject](../../internal/cli/question_test.go#L47) 创建四个 open 问题，即使关联提案条目被拒绝，问题仍保持 open。将第一处对照的 `question list --json` 改为 `question list --json --all` 后，两份结果确实相同，隔离变异没有失败。

这限制的是该夹具对过滤语义及错误配对的检出力，不影响两份完整输出都符合 `question-list.v1` 的 M-7 保证。该 Schema 本来允许 open、answered、dismissed 三种状态，不能负责表达调用时是否要求 `--all`。最终代码的两处实际配对也确实等价。

已有 [TestQuestionAnswerAndDismiss](../../internal/cli/question_test.go#L120) 使用关闭后的问题，明确检查默认只剩两个 open、`--all` 包含 answered/dismissed；本轮只静态核对该既有断言，没有把它声称为本次新增覆盖或本机重新执行结果。若以后强化参数过滤语义，可为本测试加入已关闭问题对照；这不是 M-7 合并阻塞，也不应把未检出变异记成“已验证过滤语义”。

## 五、本轮独立运行

实际平台：Windows/amd64，Go `go1.27.1`，Git `2.56.0.windows.2`。设置 `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`，使用已有依赖缓存。CLI 夹具以正常用户权限运行，以适配其私有数据目录 ACL；测试和变异均在仓库外副本完成。

```text
go test -tags nowebui -count=1 -timeout 5m -v -run '^TestCommandLineStillWorks$' ./internal/cli
go test -tags nowebui -count=1 -timeout 3m -v ./internal/contract
```

| 独立验证 | 结果 |
|---|---|
| 未修改目标的 `TestCommandLineStillWorks` | PASS；测试 5.32 秒，包 6.161 秒；五种调用和两处对照均完成 |
| 未修改目标的整个 `internal/contract` 包 | PASS；包 7.310 秒；40 个合法示例、165 个非法反例及新增完整文档测试均通过 |
| 只将根级 JSON Question 行的 Schema 改为 `source-list.v1` | 预期 FAIL，退出 1；明确由助手报告 `stdout does not conform to source-list.v1`，列出缺少字段及判别字段不匹配，证明参数被使用 |
| 只把比较条件 `stdout != want` 反转为 `stdout == want` | 预期 FAIL，退出 1；两处参数位置比较分支分别报错，证明比较断言实际执行 |
| 只给第一处对照调用添加 `--all` | PASS；复现已披露的夹具限制，未将其记为被检出的变异 |

所有三份变异每份只修改一个测试点，没有修改测试助手、生产输出或 Schema。预期变异失败是检出证据，不是目标提交的测试失败。本轮未出现定向测试超时，未为本地测试扩大全局 Timeout，也没有重新跑整个 CLI 包或整仓检查。

作者报告的 Windows 全 CLI 1178 秒、额外本地变异和检查记录属于作者证据；本报告仅将上表列出的操作称为独立复验。不据作者耗时推断根因，不关闭本地超时维护项。

## 六、CI 身份、运行及边界

指定 [run 39299026564](https://github.com/XKHoshizora/specmetry/actions/runs/39299026564) 的 `head_sha` 为固定 HEAD，事件 `pull_request`，`run_attempt = 1`。实际日志中成功取得的各 Job 均 checkout `d454091c3d464f701a9ffe6f264c2714d90d8bc7`；其父提交和 tree 已独立核对，不能把 run 的 HEAD 元数据与实际检出对象混为一谈。

| Job | Job ID | 核查状态 |
|---|---|---|
| Go lint | 118219002734 | completed / success |
| Go tests (windows-latest) | 118219003065 | completed / success |
| Web | 118219003095 | completed / success |
| Go tests (ubuntu-latest) | 118219003276 | completed / success |
| Go tests (macos-latest) | 118219003581 | completed / success |
| Build with dashboard (ubuntu-latest) | 118219202664 | completed / success |
| Build with dashboard (windows-latest) | 118219202690 | completed / success |
| Build with dashboard (macos-latest) | 118219202892 | completed / success |

- Windows、Ubuntu 和 macOS 日志均明确出现 `TestCommandLineStillWorks`、`TestValidateJSONTakesOneWholeDocument` 的 RUN 和 PASS，没有跳过这两项。Windows 的测试耗时分别为 4.30 秒、0.03 秒；其实际 checkout 也已从最终日志核对为上述合并快照。
- Ubuntu 普通测试依据实际 workflow 执行 `go test ... -race ./...`；Linux 跨账户测试在普通非 root 步骤按设计跳过，随后单独编译测试二进制并由 `sudo` 运行，八项实际 PASS。后者没有 `-race`，不能描述为 root 竞态检测。
- 日志还存在真实旧 Git/unsafe Git 需额外环境的测试、M-14 显式诊断测试，以及 Windows 的不可读文件/对象权限测试等设计跳过；Linux root 步骤在 Windows/macOS 按平台条件跳过。它们不是本轮新增测试，没有被重新验收，也不能说整仓所有测试均实际运行。
- 保留核查过程：开始时平台汇总为 `in_progress / null`，Windows/Ubuntu 尚在运行，归档日志接口返回 BlobNotFound；随后 Ubuntu 完成，Windows 仍在运行。结束时 API 正式报告 `completed / success`，Windows 最终日志已可读取。上述接口暂不可用没有被归类为测试失败，也没有用 dashboard build 代替 Go tests 证据。
- run 最终 `updated_at` 为 `2026-10-19T18:10:45Z`；本报告依据结束时读取的正式结果确认 8/8，而非从部分日志推定成功。没有重跑、取消、刷新提交或追加 attempt；未发现本次 attempt 的失败，不据此关闭 M-15 或其他维护事项。

## 七、问题分类与交付判断

**BLOCKER：无。** 当前代码与独立定向验证足以支持 M-7 的技术修复结论，指定 CI 也已正式成功。

**NON-BLOCKER：N1**，即本夹具的默认列表/`--all` 过滤检出边界，详见第四节；不要求为本次 Schema 修复扩展业务测试范围。

没有新的产品待决事项，也没有新增风险接受。M-15 及其他维护事项保持原状态。没有重新审查 WP2.3/B1、Claim 交换契约或历史安全模块。

逐项回答：

1. **M-7 是否满足修复要求：是。** 五种原弱断言均改为正确 Schema 的完整 stdout 校验；未发现范围内遗漏。
2. **PR 技术 GO/NO-GO：GO。** 没有发现实质性 BLOCKER 或直接回归。
3. **最终 CI 与合并条件：本轮合并前检查已满足。** 指定 attempt 1 正式 8/8 成功，固定身份未变；PR #28 可以由负责人合并。本轮未实际合并，不能声称合并后核验已完成。
4. **合并核验后是否可关闭 M-7：可以。** 实际合并内容应保持本次已审差异及适用性，再由负责人登记关闭；本轮不修改维护登记。
5. **验证边界：** 本机只运行 Windows 隔离副本的上述定向测试和三项变异；三平台整仓结果来自本轮核对的 CI，不冒称本机 Linux/macOS 复验；未在本机重跑全 CLI、TTY/E2E 或未变模块。

下一步由负责人决定合并并核对实际合并结果，无须为同一固定提交重复展开一次 M-7 全面审查。若 HEAD/base 或合并树变化，应先确定新增差异的适用性。本审查到此停止，仅新增未提交报告，不自行关闭事项、宣布验收或启动后续工作。
