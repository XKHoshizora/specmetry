# Codex Review 079：PR #33 的组合终判与保险清理定向复审

## 1．结论分层

**A．门禁实现及直接回归：技术 NO-GO。**

Review 078 的 **B1-R2-W-R 原组合判定缺陷可以关闭**：预期拒绝不再豁免工具错误、未知结束状态或补救失败；旧版六个错误 PASS 在新版本均被拒绝。已经关闭的 B1-R1-S 保持关闭，原始 B1 的已知假通过路径得到本轮修复证据支持。

但是，本轮增加的 Unix 保险清理引入了一个独立的 **BLOCKER B2**：调用方把仅由见证记录提供、未被本轮普查确认归属的身份加入 `known`，保险清理随后仅凭 PID 与启动时间匹配发 SIGKILL。见证即使已经被判无效，也没有阻止这一动作。身份准确不等于属于本轮检查。下文给出 Windows 上安全验证的共用数据路径，以及 Unix 动作分支的严格源码证明。

因此，可以关闭 B1 的原判定问题，**不能宣布整套原型门禁与清理已整体通过，也不建议目前合并 PR #33**。新 B2 是直接回归，不是要求本原型提前实现生产 Runner。

**B．当前合并条件：未满足。** 最终常规 CI `39968755313` 为 attempt 1、completed/failure、7/8；Windows 的 seq 跨进程测试报告 SQLITE_BUSY。原型诊断 success 和旧 CI 均不能替代这一失败。即使没有 B2，本轮也不能给出完整合并 GO。

**C．规范和授权：没有改变。** 14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收。M-14、M-15、M-16 保持未解决；没有新增接受风险。

## 2．固定身份与审查范围

| 对象 | 身份 |
|---|---|
| PR | [XKHoshizora/specmetry#33](https://github.com/XKHoshizora/specmetry/pull/33) |
| base | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2` |
| Review 078 对象 | `f82dfb8fad7f6078af663d6cb68a8073b9f97169` |
| 回归先行 | `4efc9fcfb3cda5e10a765c7c35c864c7f8d6673e` |
| 修复代码 | `31c0b290aed741f2506503aa0bb73790779a1a16` |
| 本轮目标 HEAD | `8910825c8145f7eea0cec6451ff8a84b885e48ab` |

开始核验为 **2026-10-23 14:42:31 UTC**；完成技术核查时 **14:56:55 UTC** 再次核验，报告落盘后于 **15:03:25 UTC** 最终复核。远程 HEAD/base 始终与上表相同，PR 为 open、未合并、`auto_merge=null`；最终常规 CI 仍为 attempt 1、completed/failure。没有混用其他提交。

开始时本地 HEAD 为目标，工作区干净，目标报告不存在。所有编译、运行和临时验证均在仓库外 `review079` 隔离目录；原工作区没有切换或清理。原仓库仅新增本报告。

已完整阅读适用 AGENTS、[Review 078](codex-review-078.md)、本轮 12 文件差异、[原型文档](../wp2.5-prototype.md) §2.10 与相关证据/限制，以及 [review-resolution](../review-resolution.md) §93；只追踪判断修复与直接回归所需的共用调用方。

独立核对本轮 **12 文件，+1081/−99**：

- 六个文档：AGENTS、implementation-plan、review-resolution、新增 Review 078、check-runner 草案头部、prototype 结果文档。
- 六个原型文件：新增 `gate_combo_test.go`，以及 `gate_test.go`、`main_test.go`、`witness_check_test.go`、`witness_unix_test.go`、`witness_windows_test.go`。
- `31c0b29 → 8910825` 只有五个文档变化；原型树完全相同。
- 相对上轮，生产代码、Schema、迁移、根模块依赖、`ci.yml`、原型 workflow 未变，正式冻结契约未变。

提交顺序为：`bebc1a3` 归档、`4efc9fc` 回归先行、`31c0b29` 修复、`07064d6` 结果文档、`8910825` 状态同步。Review 078 文件 SHA-256 为 `d2195de1d72e5a98479eb2a3449181aff9759393b703169da3ca7bc7afcab4ed`，与 §93 登记一致；§93 为追加，未改写历史审查或 §84～§92 的结论。

## 3．BLOCKER：B2，Unix 保险清理把未确认归属的见证身份升级为信号授权

### 3.1 位置与规则

均以本轮固定 HEAD 为准：

- [gate_test.go:489](../../prototype/wp25/gate_test.go#L489)：对 `created` 记录执行 `testedAlive`，这里只确认该 PID/启动身份存在。
- [gate_test.go:529](../../prototype/wp25/gate_test.go#L529)：`known` 先放入普查结果，又在 530～533 行追加 **存活且不在本轮普查中** 的见证身份。
- [gate_test.go:615](../../prototype/wp25/gate_test.go#L615)：即使前面的判定已经拒绝，仍无条件把混合来源的 `known` 交给保险清理。
- [witness_unix_test.go:155](../../prototype/wp25/witness_unix_test.go#L155)：157～160 行仅凭身份将 `known` 加入动作集合；181～182 行再确认启动时间后直接 `syscall.Kill(..., SIGKILL)`，没有核对标记或此前的独立归属证明。

依据是本次指令第五节“不能把无效见证 PID 转为操作授权”，以及原型文档 §2.10、652 行“保险清理只操作本轮的对象”“不使用无效见证中的 PID”。这也是 Review 078 已保留的归属与身份分离要求。

**问题不在于免除第二次环境读取本身。** 对此前已经由本轮独立普查确认归属的对象，保存其身份并在动作前再核对启动时间，可以保留 macOS EINVAL 修正方向。问题是当前 `known` 同时混入了仅来自见证、恰恰未被普查确认的对象。

### 3.2 可到达的源码反例

设 `P` 是同用户、仍存活、启动身份可读，但不属于该内层检查的进程；最简单的合成反例就是外层测试进程自身。

1. 真实内层正常运行，留下正常 `root`、`created`、`launched` 和封条。
2. 仅将其中一条 `created` 的 PID/启动身份改成 `P` 的真实值，保持五字段格式、创建者和记录数量。它通过语法解析；原 `launched` 与新 `created` 不再匹配，后续见证校验会明确拒绝。
3. 本轮普查没有 `P`，但 `recordAlive` 因身份准确返回存活。代码一方面记录 `census:` 问题，另一方面把 `P` 放进 `known`。
4. 原 Unix 被测 `remediate` 还要求 `tagged(P, p.tag)`，因此不会向没有本轮标记的 `P` 发信号；它会报告仍存活/失败。
5. 新保险清理不沿用这一归属前提。即使已有 `census`、`witness-receipt` 等拒绝理由，`insure(known)` 仍按准确的启动身份发送 SIGKILL。

如果 `P` 是外层自身，Unix 的进程枚举原本就排除当前外层 PID，以上“不在普查中”是确定的；启动身份的再次核对只会进一步确认它确实是外层，不能证明它属于内层。判定为 FAIL 之后再误操作，也仍违反动作边界。

这不是依赖 PID 复用、身份读取失败、任意恶意新代码或全新生产功能的假设；是当前清理函数接受格式可读、归属无效见证时的一条直接分支。也没有声称健康 CI 实际出现过这种记录。

### 3.3 本轮独立验证与严格边界

本机只有 Windows，因此**没有在 Unix 上实际执行该 SIGKILL**，没有终止任何未知进程。

在目标的仓库外副本中，运行真实 normal-drain 内层；在外层读取见证前，仅把一条现存 `created` 记录的 PID/启动身份替换成**本轮外层自身**。未修改 `known` 构造、判定或最终结果。只在 Windows 保险清理入口打印它收到的清单，并用真实 API 检查当前进程是否属于内层 Job。

实际观测：

```text
REVIEW079_KNOWN_OUTSIDE
pid=59592 start=134372409242521580
in_inner_job=false err=<nil>
```

同一真实 `WP25GATE` 又明确记录：

```text
census: process 59592@... alive by its record, not found by the census
witness-receipt: ... ran without its creation record
pass=false
```

也就是说，**已经被拒绝的见证确实把不属于内层的身份传到了保险清理**。这部分由真实进程路径独立复现；Unix 无标记检查的发信号分支由上述源码直接证明。Windows 的保险清理以专属 Job 限定对象，本次没有向外层发信号，测试按预期 exit 1。没有把这一 Windows 观察写成 Unix 上已动态误杀的证据。

### 3.4 影响及最小修正

影响是原型测试工具在见证无效时，可能对本轮检查之外的同用户进程发 SIGKILL，包括外层本身；这既破坏实验的可靠收尾，也超出明确授权的对象范围。不是已经发现生产 Runner 的同类缺陷。

最小修正：

1. 区分“记录声称存在的身份”和“已经独立证明属于本轮的身份”。供 Unix 免重读环境的保险清理使用的清单，只能来自之前成功的本轮普查，或同等独立的直接创建/归属证明；不能仅由见证记录和 `recordAlive` 生成。
2. 保留对可信旧普查身份的动作前复核，继续支持 macOS 首次 EINVAL 的修正；**不要求重新读取每个已可信对象的环境**。
3. 新普查失败或无法证明归属时，保守报告并仅处置已经有归属证明的对象；不要把“已判失败”当成扩大动作范围的许可，也不要简单跳过所有失败路径的保险清理。
4. 补一个“格式合法但归属无效的 created 指向外层/独立自有对照进程”的负例，验证没有向该对象发送信号；另保留此前可信普查、后来环境读取失败的正例及两项故意遗留对照。测试只操作明确自建对象，或以信号调用接缝证明拒绝。

这属于保险清理的必要技术修正，不需要新增产品风险接受，不要求重写所有 T1～T6。

## 4．原 B1-R2-W-R 的修复证据

**该原判定缺陷可以关闭。** 不能因为新 B2 而抹掉已经成立的精确修复。

[gate_test.go:374](../../prototype/wp25/gate_test.go#L374) 的 `judge` 把以下维度分开：

| 维度 | 本轮核查 |
|---|---|
| 内层结论与退出状态 | `runGate` 对用例要求的 verdict/exit 计算 `verdictOK`；见证故障用例原本允许只记录内层，不强求其 invalid，属于明确用例语义 |
| 预期拒绝 | 按冒号前完整类别匹配，不再用宽前缀命中；没找到不能通过 |
| 其他证据问题 | 未在本用例允许列表里的类别进入 Unexpected，默认拒绝 |
| 外层工具失败 | `inner/census/state-unknown/hit/still-alive/insurance` 先于允许连带类别处理，不能被豁免 |
| 补救 | 仅 not-needed/succeeded 可通过；failed/not-run 不通过 |
| 最终状态 | 仅 ended 可通过；alive 优先于 unknown，unknown 不转成 ended |
| 无回执正例 | 要求实际看到创建后未回执、已经结束的对象 |
| 保险矛盾 | 被测路径称 ended，保险却发现对象或不能确认时追加不可豁免问题，再次判定，只能降低结果 |

允许的连带类别仅是三个明确用例：未回执遗留的 `unaccounted`、无记录遗留的 `unrecorded/witness-count`、根写入失败的 `witness-raised/witness-count`。其他用例没有默认通配。未知的新类别也进入拒绝路径。

### 4.1 真实组合的修复前后复验

在 `4efc9fc` 与固定目标两个隔离副本中，均独立运行真实 `TestGateCombos`，并带正常 normal-drain 以满足诊断的非零场景门禁。十项的要求写死在测试表中，不由 `judge` 生成期望。

| 组合 | 新版被测自检所见 | 旧版 self PASS | 新版 self PASS | 新版组合 |
|---|---|---|---|---|
| reject-remedy-ok | 预期命中，补救成功，ended | true | true | PASS |
| reject-remedy-error | 预期命中，补救失败，alive | **true，错误** | false | PASS |
| reject-unrecorded-remedy-error | 同上，对象无记录 | **true，错误** | false | PASS |
| reject-end-unconfirmed | 补救已执行但确认失败，unknown | **true，错误** | false | PASS |
| reject-census-error | 普查工具错误，unknown | **true，错误** | false | PASS |
| reject-state-unknown | 记录状态读取错误，unknown | **true，错误** | false | PASS |
| reject-with-unexpected | 存在未允许连带问题 | **true，错误** | false | PASS |
| expected-missing | 没有预期拒绝 | false | false | PASS |
| healthy | 不需补救，ended | true | true | PASS |
| healthy-state-unknown | 健康内层，但外层读数 unknown | false | false | PASS |

旧版本机结果：10 个组合中 6 个失败，整个进程 **exit 1**；失败明确是被测自检仍给错误 PASS。新版：10/10 组合通过，原两项精确补救反例均为 self FAIL，整个定向运行 exit 0。

原精确反例在新版本机分别看到自有 helper `66268`、`68528`：内层 invalid/1，补救前存活，故障补救返回同一对象和错误，Final=alive、self.Pass=false；保险随后 `ended 1`，没有把原 self FAIL 恢复。旧版对应 helper `22220`、`14400` 在相同路径仍是错误 self PASS。PID 仅用于绑定本轮记录，不是跨运行身份比较。

注入位置是真实路径：remedy-error 在非空自有负载上不执行补救；另三种先调用实际普查、状态读取或补救，再替换该步骤回答。它们不直接改 pass、不改内层 verdict、不改见证。已逐项读取实际内层 verdict/exit：例如 census-error 可保持内层 confirmed/0而仅外层无效，不能强行要求所有组合的内层都 invalid。

### 4.2 判定表与独立边界

`TestGateJudge` 13 行固定期望实际通过，包括未执行补救、未知 Final、未命中注入、保险矛盾和额外类别。两类证据互补：真实组合验证接线，表用例验证真实路径不易触发的状态；不是两套实现相互生成期望。

另在仓库外增加七个固定期望的独立行，验证：工具错误即使写入允许列表仍拒绝；`leftover` 不冒充 `left`；未声明类别拒绝；Final 已结束也不能豁免未执行补救；内层不匹配拒绝；缺少要求的无回执观察拒绝；合法连带现象仍通过。全部符合预期。

## 5．保险清理的判定与 macOS 边界

不涉及 B2 归属问题的终判接线成立：

- 被测路径的 Remedy/Final/Problems 先保存、先判定，之后才保险清理，最后才打印和累计最终结果。
- 保险不能把失败恢复成通过。若 Final=ended 却保险发现残留或失败，会追加 `insurance:` 再判；若 Final 原本不是 ended，self 已拒绝，组合还单独检查保险是否成功。
- 后一种情形不一定再次把失败写入 ToolErrors，但 Insurance 原始结果被保留，上层组合失败；没有把“没新增某个字段”误认为整体成功。
- `checkGate` 最终 self.Pass=false 会 `t.Errorf`，`checkCombo` 不符合要求会 `t.Errorf`，TestMain 返回 Go 测试实际失败码。汇总与实际返回值一致。

本轮还做了一个独立保险确认故障实验：Windows 包装器先完整执行真实保险清理，随后仅把其成功回答替换成 `FAILED`，确保没有为注入遗留对象。结果：

- 健康 normal-drain 的被测路径原本 ended，追加 insurance 工具错误，门禁 FAIL。
- reject-remedy-error 的 self 原本已经 FAIL，保险失败使更上层组合也 FAIL。
- 汇总 gate_failed=1、combo_failed=1，整个进程 exit 1。不是只检验辅助函数返回错误。

### macOS 首次 EINVAL

独立核验旧诊断 `39964760402` 的 macOS 完整日志：无记录遗留组合先发现进程 `1549`，随后保险重新读取其环境返回 EINVAL，Insurance=FAILED；组合同时报告错误 self PASS 与保险未能确认结束。

不能把这一次整体失败全称为预期 B1 回归失败，也不能把 helper 设置了 8 秒兜底当成其最终结束或被清理的独立证据。该 runner 上后续状态没有核验；“处于 exec/退出过渡”只是作者猜测，原因未证实。

新诊断同一 macOS 组合先按此前普查保存的身份处理进程 `2141`，Insurance=`ended 1`；被测路径仍为 failed/alive、self FAIL。新路径确实运行成功，但不证明旧 EINVAL 根因消失，也不把环境读取失败一般化成可靠不存在。

**应保留的正确方向**是复用此前可信普查的归属和身份；**B2 的错误**是把未获同等归属证明的见证身份一起放入该清单。修正这一来源边界即可，不必撤销整个 EINVAL 处置方向。

保险清理与被测路径使用相同的身份读取/普查方法，只是绕开被测故障接缝；不是第二套完全独立的归属机制，也不能弥补普查方法本身的缺陷。

## 6．本轮本机验证与限制

环境：Windows 11 `10.0.26300`、amd64，现有 Go 1.27.1。固定对象通过 Git archive 导出到仓库外 `review079`；没有安装升级工具、下载新依赖或运行用户项目检查。

| 独立验证 | 结果 |
|---|---|
| 目标：TestPipeNormalDrain、全部 TestGate/TestGateCombos、TestGateJudge、TestWitnessJudge、TestRecordAlive、TestStartPoints | 首次实际运行 exit 0；1 个正常机制、32 项门禁、10 个组合；13 行 GateJudge 和 24 行 WitnessJudge 实际通过 |
| 回归先行：normal-drain + 全部十个组合 | exit 1；六项精确错误 PASS 被检出，健康和其他对照保持预期 |
| 七个独立固定判定边界 | exit 0；正常场景对照同时通过 |
| 真实保险清理后注入确认失败 | exit 1；门禁与上层组合均失败，资源先实际清理 |
| 归属无效的 created 指向本轮外层自身 | exit 1；证实它流入 known/保险入口，但未被本轮 Job 接纳；未增加任何对该对象的信号调用 |

本轮没有编译失败后把它算作检出，也没有为绿色重跑同一个失败场景。旧版本 exit 1、定向故障 exit 1、归属反例 exit 1 均保留，分别代表预期回归检出、失败传播以及 B2 数据路径的拒绝/传递观察。

原型机制 confirmed、门禁 PASS、被测 self verdict、上层 combo PASS、直接单元测试分别统计。组合中正确的 self FAIL 不写入 gateResults，也不新增机制 confirmed。Results 步骤的重印不重复计数。

各本机测试有明确 30/40/60/180 秒测试期限，正常完成；实验仅用本轮创建或本轮外层自身的身份。Windows 专属 Job 和实际保险清理负责本轮对象收尾。结束时未发现隔离目录下运行进程或本轮已知 helper 残留；没有声称系统其他任务均无进程。

证据文件位于仓库外 `review079` 的 `fixed-run.log`、`old-run.log`、`dimensions-run.log`、`insurance-mutation.log`、`ownership-run.log`。不作为原仓库额外新增内容。

Linux/macOS 没有本轮本机动态复验。相应结论来自源码和正式三平台完整日志；B2 的 Unix 动作后果是源码级证明，不能冒称已在 Unix 误杀复现。作者 M25～M30 仅作为辅助证据；其中首次 M25 因未使用变量编译失败属于无效实验，不计检出。本轮没有重做所有变异或全部 T1～T6。

## 7．CI：正式状态、检出身份与适用性

以下通过正式 run/job API 和实际日志核对，均 attempt 1，没有重跑或触发 workflow。

| Run | 事件 / HEAD | 正式状态 | 证据范围 |
|---|---|---|---|
| [39964760402](https://github.com/XKHoshizora/specmetry/actions/runs/39964760402) | push / `4efc9fc` | completed/failure，三作业 failure | 回归先行诊断；直接检出该代码 |
| [39966556540](https://github.com/XKHoshizora/specmetry/actions/runs/39966556540) | push / `31c0b29` | completed/success，三作业 success | 修复后诊断；直接检出该代码 |
| [39964764139](https://github.com/XKHoshizora/specmetry/actions/runs/39964764139) | pull_request / `4efc9fc` | completed/cancelled | 7 个作业 success，Windows Go tests cancelled |
| [39966562140](https://github.com/XKHoshizora/specmetry/actions/runs/39966562140) | pull_request / `31c0b29` | completed/cancelled | 7 个作业 success，Windows Go tests cancelled |
| [39968755313](https://github.com/XKHoshizora/specmetry/actions/runs/39968755313) | pull_request / `8910825` | **completed/failure，7/8** | 最终常规 CI，不运行原型 |

两次 cancelled 的 Windows 日志实际记录 `The operation was canceled`；原工作流的 cancel-in-progress 设置未变，其时间与后续推送相符。没有把取消当作成功、超时或产品错误，也没有把已完成七项补成 8/8。

三个常规 CI 的合并快照均为 base 与各自 HEAD 两个父提交，树与各自 HEAD 一致：

```text
4efc9fc -> f373e34bc0ac393eda5d7533b58cbe7614bde500
           tree a57cc23f3a6b480bb5cffd80885f3b5d1b51dca8
31c0b29 -> f956e15c61db8c344502ace0aff22be1201ef6ee
           tree 6bbb1fc574d58f0655244431a05931a6ed0f756f
8910825 -> f12acdae8483dc3fc32b94b2bc9ba646615c5ee3
           tree c53968919433981e23fd204c59217b2d81780b1a
```

最终八个作业的实际 checkout 都是 `f12acdae...`：

| Job | 名称 | 结果 |
|---|---|---|
| 120466046331 | Web | success |
| 120466046772 | Go tests (macos-latest) | success |
| 120466046775 | Go tests (ubuntu-latest) | success |
| **120466046999** | **Go tests (windows-latest)** | **failure** |
| 120466047241 | Go lint | success |
| 120466252293 | Build with dashboard (macos-latest) | success |
| 120466252332 | Build with dashboard (ubuntu-latest) | success |
| 120466252440 | Build with dashboard (windows-latest) | success |

### 7.1 原型的实际场景与组合

| 诊断/平台 | 唯一机制场景 | 门禁 | 组合 |
|---|---|---|---|
| 旧版 Ubuntu | 38 confirmed | 36/36 PASS | 4/10 PASS，六项失败 |
| 旧版 macOS | 27 confirmed | 27/27 PASS | 4/10 PASS；其中还含保险 EINVAL |
| 旧版 Windows | 28 confirmed、1 refuted | 32/32 PASS | 4/10 PASS，六项失败 |
| 新版 Ubuntu | 38 confirmed | 36/36 PASS | 10/10 PASS |
| 新版 macOS | 27 confirmed | 27/27 PASS | 10/10 PASS |
| 新版 Windows | 29 confirmed | 32/32 PASS | 10/10 PASS |

新版三平台为 Go 1.27.3；Ubuntu 内核 6.17.0-1022-azure、macOS 26.6.2 / Darwin 25.6.0 arm64、Windows 10.0.26100 amd64。与本机版本差异保留。

Ubuntu 原型实际条件设置 `race=-race` 后执行 go test；macOS/Windows 未带 -race。四个直接测试在三平台均有 PASS，包括新 GateJudge 的 13 行。顶层仅两个 inner-only 测试 SKIP；它们在 Gate/Combo 的真实子进程中实际执行，其内层 verdict/exit、负载和盘点结果可核对。SKIP 不算机制通过。

按场景 ID 去重后三平台并集仍是 **53**。不把平台次数相加当作独立场景数，不把 Results 重印或组合中的 self 结果重复累计。

旧 Windows 的独立 refuted 是 N6：阶段 D **999.9851 ms**。不能把三平台旧诊断的全部失败都归因于预期 B1 组合失败；macOS 的保险失败也单独保留。

### 7.2 最终 Windows 常规 CI 失败

完整日志的关键记录为：

```text
TestSeqIsUniqueAcrossProcesses
store_test.go:571:
exit status 1: database is locked (5) (SQLITE_BUSY)
--- FAIL: TestSeqIsUniqueAcrossProcesses (5.26s)
FAIL github.com/XKHoshizora/specmetry/internal/store/sqlite 47.086s
```

Windows 非 CLI 组失败后，CLI 组仍继续，实际为 `internal/cli 404.478s` 成功，最终作业 exit 1。没有吞掉第一组退出码，也没有发生本轮新的 CLI 超时。

本轮没有修改或运行原型于这条常规测试路径，但这**不证明失败根因已经确定或与改动绝对无关**。571 行是父测试转发辅助进程输出的位置；辅助进程既调用 Open，也执行 writeFact，仅凭这段输出不能排他定位发生在哪一内部阶段。

错误特征与 M-15 已登记的 SQLITE_BUSY 观察相似，须保留该调查边界。M-14 的定义是版本探测路径的 SQLITE_READONLY_RECOVERY(264)；不能因为同名测试失败就将当前错误并入 M-14，也不能只凭无版本包装就断言完整因果链。没有批准重试、映射变更或风险豁免。

最终 Ubuntu 常规 Go 测试实际带 -race；独立 root CrossUser 步骤实际运行八项，普通步骤对应跳过不等于通过，root 步骤也不是原型的 -race 证据。其他平台专属/权限/旧 Git 跳过保持既有边界，不据本轮常规 CI 重新验收旧模块。

## 8．非阻塞意见、文档与待决范围

- **N7 可关闭**：§3.4 改成“当时证据/当前证据”的明确追溯，§4～§8 指向当前 `39966556540/31c0b29` 并更新数值；Review 078 原文没有被改写。
- **N4 保持未修复、非阻塞**：epipe 报告失败的 refuted/invalid 分类问题仍登记，没有本轮新绿色放行路径的证据，不要求为本轮顺带改生产。
- **N6 保持未修复、非阻塞**：计时口径差异和严格比较的随机拒绝仍存在，旧 run 的 refuted 如实保留；没有因新 run 成功关闭，也没有用任意容差取得绿色。
- C-4/D20、C-19/C-20 的前提和待决状态保持。源码扫描仍只是当前原型启动入口检查，不等于生产 C-9。
- §93 忠实登记 Review 078 的身份、B1-R1-S 关闭、B1-R2-W-R 待复审和原 NO-GO；没有自行宣布关闭或批准产品选择。AGENTS 和实施计划只同步状态。

**PRODUCT DECISION：没有新增可代替 B2 修复的产品选择。** 原有 D1～D9、D11、D13、D18、D19、D20 共 14 项继续待批准。

**ACCEPTED RISK：无新增。** 既有方案 B 不豁免对非本轮对象发信号，不放宽 Log、数据库、来源权限或交付证据保证。

## 9．最小后续任务与最终答复

| 问题 | 本轮裁定 |
|---|---|
| B1-R1-S | 保持 Review 078 的关闭结论。 |
| B1-R2-W-R | **可以关闭**；精确反例及六项旧错误 PASS 均被新判定拒绝。 |
| B1 原始及已知判定缺陷 | 可按既有原型门禁范围收口；**不等于本轮直接回归 B2 也已通过**。 |
| 门禁实现和直接回归的技术结论 | **NO-GO：新增 B2，Unix 保险清理的归属来源不可信。** |
| 当前合并条件 | **未满足**，且最终常规 CI 正式失败，不能用旧绿色替代。 |
| 准备/冻结门槛是否全部完成 | 没有。T1～T6 仍是局部支持，14 项产品选择及既有支持范围/失败协议规格仍待处理。 |

本 PR 的必要最小修正是 B2 的清单来源隔离与自有/非本轮对象正反例；不需要重做全部机制，也不要求提前完成生产 Runner。最终常规 CI 的失败须由负责人另行安排处理并核验对应可合并对象；本轮不重跑、不用风险接受绕过它，也不要求原型 PR 顺带修复整个存储层。

冻结前继续保留全局闸门拓扑、C-9 规则设计、macOS ABI/支持版本/枚举预算、Windows 父端期限与原生失败边界、T5 来源/PATH/worktree 规则及产品选择。生产接入、迁移 0009、正式契约、事务/取消/配置/空批次、CLI/E2E 与生产三平台测试属于后续获准实现和验收阶段，没有在本轮被宣布完成。

归档说明：这是本轮新写的完整报告，仓库引用使用相对路径；未改写历史报告。除本报告外，原仓库没有新增或修改文件；没有提交、推送、改基、合并、重跑 CI 或授权生产实施。
