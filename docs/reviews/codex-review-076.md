# Codex Review 076：PR #33 有效性门禁定向复审

## 1．结论

**技术 NO-GO；暂不建议合并 PR #33。Review 075 B1 尚不能整体关闭。**

原来的两个明确反例已得到修复：畸形的 `limit not-a-pid` 记录和没有建立后代的 Windows fanout，现在都使真实内层场景返回 `invalid`、exit 1。现有门禁回归确实重新启动真实场景，并同时核对场景 verdict 和子进程退出码，没有直接伪造最终 verdict。

仍有两项本轮范围内的阻塞：

1. **B1-R1：一次进程状态观察失败，可以被后来的成功观察覆盖，最终仍 confirmed、exit 0。** 已在 Windows 隔离副本中独立复现；外层 `TestGate` 同样通过。
2. **B1-R2：Linux 不完整进程链的提前拒绝，没有收尾已经启动、且脱离原组的部分进程链。** 这是具体调用链的源码级反例；本轮没有在 Linux 动态复现，不能将其写成已观察到的 CI 遗留进程。

Review 075 N1～N3 的定向处理成立，另有一个不产生绿色假通过的分类问题，列为本轮 N4。未发现生产代码或冻结契约的变更。

这次审查不批准产品选择、不冻结规范、不授权生产实现，也不宣布 WP2.5 或 M2 验收。M-14、M-15、M-16 保持未解决。

## 2．固定身份、工作区与范围

审查日期：2026-10-23（Asia/Tokyo）。仓库：[XKHoshizora/specmetry](https://github.com/XKHoshizora/specmetry)，[PR #33](https://github.com/XKHoshizora/specmetry/pull/33)。

| 对象 | 完整身份 |
|---|---|
| base | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2` |
| 本轮 HEAD | `3ddd14fb1d478e333ad922310f9bcfc2b70033d9` |
| 修复后的原型代码 | `a793cedfcef5596bad05612df7f4165490319f28` |
| 上轮对象 | `932b4a80852210a9fcc04cb0baf79aa4de04a7cf` |
| 回归先行 | `5140588e4cbb54eb224dc8131b113856ed43c781` |

开始与结束的 GitHub API 核验均为上述 HEAD/base，PR 为 open、未合并，`auto_merge=null`；未发生对象漂移。本地 HEAD 同为固定目标；开始时工作区干净，报告路径不存在。测试和临时变异在仓库外的独立目录进行，没有切换、清理或改写原工作区。

完整读取了适用 AGENTS.md、[Review 075](codex-review-075.md)、本轮差异、[原型报告](../wp2.5-prototype.md)与 [review-resolution](../review-resolution.md) §90，追踪了共享判定、报告、负载、观察、归属和收尾调用链。

实际增量为 **19 个文件，+1692/−441，其中 13 个原型文件**：

- 文档：AGENTS.md、implementation-plan、review-resolution、新增 Review 075、准备草案头部、原型报告。
- 原型：emain_unix、exclusive_linux、gate、helper、helper_unix、job_windows、main、pipe、report、tree_darwin、tree_linux、tree_unix、tree_windows 的测试文件。

提交顺序为：

1. `1ee51059fb0f5b7d6f1e0e711af39fa9fec5f1e2`：归档 Review 075。
2. `5140588e4cbb54eb224dc8131b113856ed43c781`：回归先行。
3. `a793cedfcef5596bad05612df7f4165490319f28`：门禁修复。
4. `fe553d3d0ff7e321360353d7580102273d927647`：报告收口。
5. `3ddd14fb1d478e333ad922310f9bcfc2b70033d9`：§90 与状态同步。

已独立确认 `a793ced → HEAD` 只有五份文档变化；生产代码、Schema、迁移、根模块依赖、`ci.yml`、原型 workflow、architecture、data-model 和 acceptance-criteria 在本轮均未变。`git diff --check` 通过。

review-resolution 的旧文件完整保留为新文件的前缀，§84～§89 未改写。归档 Review 075 的 SHA-256 为：

`92a7b398352a41b01b277c83aad32eca7c703cd878e863297cf076f86ece584a`

与 §90 的登记一致；审查身份仍指向 `932b4a8`，原 NO-GO 没有被替换为实施方的关闭结论。

## 3．BLOCKER

### B1-R1：一次观察错误没有进入不可撤销的 invalid 状态

**位置与规则依据**

- [Windows pidProc.ended](../../prototype/wp25/tree_windows_test.go#L122)：等待错误只写入 `p.err`，返回 false。
- [Unix pidProc.ended](../../prototype/wp25/tree_unix_test.go#L145)：unknown 同样只写入 `p.err`；后来的 absent、身份变化或僵尸回收可以返回 true，回收分支还会覆盖 `p.err`。
- [scen.cleanup](../../prototype/wp25/main_test.go#L288)：继续轮询，只要最后所有 `ended()` 为 true 就返回 `verified`；没有把此前的观察错误加入 `s.invalid`。
- [scen.done](../../prototype/wp25/main_test.go#L237)：最终可以落入 confirmed。

原型报告 §2.2、§2.4 及 §90 声明观察读不到应为 invalid；本轮要求也明确要求核查错误是否会被后来的成功覆盖。这不是要求对观察失败永久拒绝清理，而是要求**清理最终成功与本次实验有效性分别保存**。

**独立反例**

在目标代码的仓库外副本中，只改变 Windows `pidProc.ended` 的一次观察结果：第一次调用仍执行真实 `WaitForSingleObject`，随后把这一次结果替换为观察失败；以后调用恢复原逻辑。没有修改场景预测、负载、最终 verdict 或外层判定：

```go
ev, err := windows.WaitForSingleObject(p.h, 0)
if p.err == nil { // 本轮独立的一次性故障注入
    p.err = errors.New("review076 one-shot observation failure")
    return false
}
// 以下保留原错误分支与 WAIT_OBJECT_0 判定
```

运行真实 `TestPipeSilentHolderReal`，三个父 helper、三个后代均建立并完成归属核验。结果：

```text
WP25RESULT id=T1.pipe.silent-holder verdict=confirmed launches=3 work=3
cleanup="verified: 6 processes gone (... pid ... (review076 one-shot observation failure), ...)"
WP25SUMMARY confirmed=1 invalid=0
exit 0
```

再经真实外层 `TestGate/control.silent-holder` 启动同一内层：

```text
got=confirmed inner_exit=0 pass=true
```

这不是假造最终结果，也不是使辅助进程提前失败的无关拒绝。错误沿原有 `p.err → cleanup → done` 路径发生，清理后的对象确实已经结束，但本次观察错误没有使实验失效。清理字符串中保留错误文字不能代替 verdict 的拒绝；外层回归仍报告通过。

Microsoft 的 [WaitForSingleObject 文档](https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-waitforsingleobject)区分等待失败与对象已发信号；本次注入验证错误传播，不声称实际发生了 Windows 内核故障。Unix 的一次 unknown 后读到 absent 或成功回收，也有相同的源码路径；未在本机动态运行 Unix 变体。

**影响**

现有 `cleanup.state-unknown` 使用持续失败注入，只能证明持续失败到期限后会 invalid，发现不了“一次失败、下一次成功”。因此“三态贯穿且任何观察失败都拒绝”的证据尚未建立；它仍是 Review 075 B1 的残余。

**最小修正**

使观察错误进入场景级、不可被后续成功清除的 invalid 状态，或由观察接口独立返回错误并由清理调用方累积。可以继续只针对已核验对象做安全回收，但不得因后来回收成功而恢复本次实验有效性。增加一次失败后成功的真实场景回归，并核对外层确实收到 invalid、非零退出。

### B1-R2：短链提前拒绝绕过了脱离原组的已启动后代清理

**位置**

- [runCheck](../../prototype/wp25/exclusive_linux_test.go#L260)：等待全部 depth 条记录成功后，才逐条 `ownReported`；清扫位于其后。
- [chain-main / chain](../../prototype/wp25/helper_unix_test.go#L52)：链在新 session 中，`chain-short` 把四层变成三层；链节点保持运行。
- [exclusive 的退出](../../prototype/wp25/exclusive_linux_test.go#L69)：离开排他区间就清除 subreaper。
- [startTree 提前收尾](../../prototype/wp25/tree_unix_test.go#L201)：只终止原组。
- [awaitReport](../../prototype/wp25/report_test.go#L173) 和 [cleanup](../../prototype/wp25/main_test.go#L288)：前者错误时直接终止测试 goroutine，后者只检查 `s.procs` 中已经登记的对象。

**具体路径与证据强度**

现有门禁 `load.chain-short → TestAdoptionChain → runCheck(depth=4)` 就触发该路径，不需要另造不合法历史或扩大信号范围：

1. `chain-main` 在原检查进程组中；它启动一个 `Setsid=true` 的三层链。
2. 三个链节点写出合法 PID 记录，然后保持运行。它们不在原进程组。
3. 父端等四条记录，五秒后 `awaitReport` 调用 `invalidf/FailNow`；`ownReported` 的循环尚未执行，三条已存在的节点没有进入 `s.procs`，后面的 `sweep` 也没有执行。
4. 栈展开时，`exclusive` 先清除 subreaper。`scen.done` 的提前组清理只能终止原来的 `chain-main`；不能触及新 session 的链。
5. 通用清理只核验已登记的主进程，随后内层返回 invalid，外层把该拒绝记为 PASS。三条链没有父端收尾，只剩各自开始运行后 60 秒的 helper 定时器。

已核对修复后 Ubuntu 日志，门禁原因恰为：

```text
only 3 of 4 pid records within 5s
got=invalid inner_exit=1 pass=true
```

以上残留结论是**源码级证明**，不是本轮在 Linux 观察到的存活 PID 或 CI 进程清单。本机 WSL 入口返回 `Wsl/E_ACCESSDENIED`，没有启动 Linux 夹具；没有绕过环境拒绝。Linux 官方 [subreaper 说明](https://man7.org/linux/man-pages/man2/PR_SET_CHILD_SUBREAPER.2const.html)支持这里的收养边界：解除该属性不是终止或回收后代。

**影响**

拒绝不完整负载本身是正确的，但不能把这个 PASS 同时用作“拒绝路径已经安全回收本轮对象”的证据。当前代码会在仍有自己的逃逸链时结束内层运行，依赖 helper 内部定时器收尾，违背本原型“判定前处理自己的后台对象”、父端收尾不依赖内部定时器的声明。主进程行数或已登记对象数不等于全部已启动对象。

**最小修正**

在等待完整负载之前就建立失败路径的所有权及清理责任；针对已核验的部分链，在排他区间和所需身份前提仍成立时完成回收，然后释放 subreaper/租约。不得对未核验的报告 PID 直接发信号，也不得靠扩大组或进程扫描范围补救。增加真实短链拒绝的外层检查：不仅断言 invalid/非零，还须独立确认已启动链全部结束并按适用关系回收。相同的报告错误、父 helper 等待失败路径应复用该收尾保证。

## 4．已修复部分、直接回归与 N1～N3

| 项目 | 本轮裁定与证据 |
|---|---|
| 原 `limit not-a-pid` | 已修复。旧门禁内层 confirmed/0；新门禁因 malformed PID invalid/1，外层正确 PASS。 |
| 原 fanout 零后代 | 已修复。旧门禁五轮 work=5 即 confirmed/0；新门禁等不到 30 个 PID，invalid/1。 |
| fanout 部分负载 | 新版 10/30 不能通过；正常对照五轮建立 30 个不同后代、31 个 Job 活动进程，work=155。每个报告 PID 另经 Job 归属核验。 |
| 缺失、删除、畸形、未知、重复、截断、超长 | 严格解析和终判实际执行；本机相应外层测试核对了内层 invalid 和 exit 1。 |
| 规格外的合法记录 | `report.unexpected-kind` 到终判才拒绝，不是靠解析失败或前置无关错误。 |
| 轮询未完成行 vs 终判截断 | 实现分开处理；本轮额外直接调用原解析、等待、终判函数验证。等待到期不返回部分 PID 作为成功结果。 |
| 父 helper 非预期退出 | `parent-fail` 真实先启动后代再 exit 5，新版由负载门禁 invalid；Windows 拒绝路径通过原 Job 收尾。不能由此推论 Linux 的逃逸短链也已收尾。 |
| 归属拒绝 | `ownership.foreign-pid` 使用本次场景自己启动、但不在检查树中的兄弟进程。Windows 独立运行得到 not-in-Job 的 invalid/1，原型清理自己的两个对象；未向未知进程发信号。 |
| 三态观察 | Linux 枚举错误不再变成空集合，Unix unknown 不再在该次调用中直接等同 absent。持续错误回归有效；仍受 B1-R1 限制。 |
| 实际进程链与 runCheck | 正常 CI 有完整链、阶段及回收断言；短链的负载拒绝有效，拒绝后的资源保证受 B1-R2 限制。 |
| 健康、合法反例、故障注入 | 外层正常 silent-holder、normal-drain、capacity-stop 反例、Windows fail-assign 均仍 confirmed/0，说明没有修成一律 invalid。 |

**Review 075 N1：收口成立。** C-4 已明确归入 D20 的待决范围；当前批次停止与同一宿主的新调用准入分开，单次 CLI 与长驻宿主可能持续拒绝的代价明确。没有用原型观察批准此选择。

**Review 075 N2：本轮修正成立。** Windows 已核验后代的观察句柄有显式 release；失败链用关闭调用的返回值检查 Runner 的进程/Job 句柄，不再向已关闭句柄调用验证 API。本机 fail-create、fail-assign、fail-confirm、fail-resume、query-fails 均正常完成；没有重现 Review 075 的 `0xc0000008`。这证明这些测试路径，不等于所有 Windows 资源分支均已穷尽；观察失败的 verdict 问题另见 B1-R1。

**Review 075 N3：文档收口成立。** helper 内部定时器只在执行到 Go 代码后生效，不能约束尚未恢复的 CREATE_SUSPENDED；父端等待与待冻结的独立期限要求已分开。Ubuntu 内核按不同运行分别记录，没有继续把旧运行的 7.0 与本轮的 6.17 混用。B1-R2 是另一条实际拒绝收尾缺口，不能由这项措辞修正豁免。

## 5．NON-BLOCKER

### N4：epipe 报告写入失败被归为 refuted，而不是 invalid

[holder helper](../../prototype/wp25/helper_test.go#L74) 仍忽略 `report("epipe")` 的错误并 exit 3。[F1 场景](../../prototype/wp25/pipe_test.go#L143) 允许 `epipe` 计数为零的最终报告，把没有事件作为预测不符。

本轮在另一份隔离副本中，仅使 `report("epipe")` 返回注入的写入错误；真实 holder、父 helper、管道和清理继续运行。结果为：

```text
T1.pipe.F1-skip-drain verdict=refuted
reason="differs: holder's late writes fail"
0 epipe reports; cleanup verified: 2 processes gone
exit 1
```

工具无法报告观察结果被误记为机制被推翻，不符合“辅助报告失败为 invalid”的分类承诺。不过它没有形成 confirmed/绿色假通过，故单列 NON-BLOCKER，不将其夸大为机制已经被反证。建议传递独立的报告失败状态，区分“没有观察到预测事件”和“观察工具失败”，增加对应分类回归。

## 6．本轮独立运行与边界

环境：Windows 11、build 26300、amd64；已有 Go 1.27.1。使用固定对象导出的仓库外副本，禁止工具链自动下载；没有安装或升级工具。原型代码和外层测试均构建为真实测试二进制。

| 独立验证 | 对象与结果 |
|---|---|
| 原两个反例的旧版重放 | `5140588`：外层 `report.malformed`、`load.fanout-zero` 均 FAIL，因为内层 confirmed/0；整体 exit 1。正常排空对照 confirmed。 |
| 新版对应正反例 | 固定目标：上述两项、fanout-partial、foreign-pid、正常 fanout 共五项外层 PASS；无效内层均 invalid/1，健康 fanout confirmed/0。 |
| 其余现有 Windows 门禁 | 另一组 13 项全部 PASS；与前组构成当前 Windows 全部 18 项，不重复计数。包括缺失、未知、重复、合法但多余的 kind、截断、超长、删除、parent-fail、持续 cleanup-unconfirmed 及正对照。 |
| 直接机制回归 | fail-create、assign、confirm、resume、query-fails、normal-drain、F1-skip-drain 七项根场景均 confirmed/0；不把外层重跑的相同场景再加为新机制数量。 |
| 报告协议额外小测试 | 直接调用真实函数：未完成行轮询可继续、终判拒绝；到期不返回部分集合；畸形/未知/重复/非规范 PID、65 字节行、超大文件、规格外 kind、删除和目录型读取均拒绝；健康报告通过。 |
| 单次观察错误 | 一次性故障注入；真实 silent-holder 及真实外层仍绿色，见 B1-R1。不是内核故障实测。 |
| helper 写报告错误 | 仅注入 epipe 写错误；实际为 refuted/1，见 N4。 |
| Linux 短链 | 已做源码与正式日志核验，未在本机运行，见 B1-R2。 |

门禁回归 PASS 与机制 confirmed 分开统计。上述额外解析测试不是新的机制场景，也没有加入仓库或 CI。

准备阶段的失败如实保留：首次直接调用测试二进制时 PowerShell 参数传递导致 `flag provided but not defined: -test`，没有执行场景，改为显式参数数组后运行；一次变异目录尚未建立导致启动目录无效，以及一次使用 shell 不支持的花括号文件列表导致解析失败，都不计为有效实验。WSL 入口为 `Wsl/E_ACCESSDENIED`，没有 Linux 进程开始；没有安装工具、提权绕过或据此判断产品失败。

本轮没有重跑 macOS、Linux 的机制实验，没有再次运行 T5 解释器全集，没有构造不可中断睡眠、强制 PID 复用或对未知进程发信号。macOS 采用完整日志和平台源码核验；不能记为本机动态通过。作者的 WSL/Windows 实验、M1～M7 变异及其第一次 M7 编译失败仍是作者证据，不冒称本轮复现。

所有本轮实际启动的 Windows 夹具按其持有的进程/Job 句柄收尾；结束时隔离目录下的测试程序无存活进程。没有因发现 Linux 源码反例而主动运行会留下逃逸链的实验。

## 7．CI 的独立核验

分别读取了五个 run 的正式元数据、每个 Job 的结果及实际 checkout 日志；没有触发或重跑 workflow。

| run | 事件、对象、attempt | 本轮核验的正式结果 | 适用范围 |
|---|---|---|---|
| [39830995366](https://github.com/XKHoshizora/specmetry/actions/runs/39830995366) | push，`5140588`，1 | completed/failure，三 Job 均失败 | 回归先行；原型旧门禁的预期失败。 |
| [39831007212](https://github.com/XKHoshizora/specmetry/actions/runs/39831007212) | pull_request，`5140588`，1 | completed/success，8/8 | 同提交常规 CI，不运行原型。 |
| [39835302375](https://github.com/XKHoshizora/specmetry/actions/runs/39835302375) | push，`a793ced`，1 | completed/success，三 Job 均成功 | 修复后原型，直接检出该代码 SHA。 |
| [39835309004](https://github.com/XKHoshizora/specmetry/actions/runs/39835309004) | pull_request，`a793ced`，1 | **本轮已为 completed/success，8/8** | 追加后来取得的状态；不改写作者交付时汇总不一致、未计成功的历史。 |
| [39839185387](https://github.com/XKHoshizora/specmetry/actions/runs/39839185387) | pull_request，`3ddd14f`，1 | completed/success，8/8 | 最终文档 HEAD 的常规 CI。 |

三组常规 CI 的八 Job 分别为 Web、Go lint、三平台 Go tests、三平台 Build with dashboard。每组八份 checkout 日志与 Git commit API 交叉核对如下：

| PR CI | 实际合并快照 | 父提交（顺序为 base、HEAD） | tree，与对应 HEAD 相同 |
|---|---|---|---|
| 39831007212 | `457c24e2a0515d7429ce51ad611f8dc978849270` | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`、`5140588e4cbb54eb224dc8131b113856ed43c781` | `e4d932a03254791f6704ac60bf05e09701f8dec4` |
| 39835309004 | `7a979082c7a4ba596623e6b53fd3f9fe6b7dbc97` | 同一 base、`a793cedfcef5596bad05612df7f4165490319f28` | `d8f1505c2b043564621ba0ddd7396cc9e95426bd` |
| 39839185387 | `1d12c4a8b750272283f938a3581aa2f940871dca` | 同一 base、`3ddd14fb1d478e333ad922310f9bcfc2b70033d9` | `98755800858ce8be5105411879b3135bfdbf859c` |

最终快照与用户提供的预期相符。诊断 workflow 属 push 事件，六份诊断 Job 日志分别直接检出 `5140588` 或 `a793ced`，没有把 PR 合并快照的证据混到仓库外实验。

### 原型计数与跳过

| 平台 | 回归先行：场景 / 门禁失败 | 修复后：场景 / 门禁通过 |
|---|---|---|
| Ubuntu | 38 confirmed；10/17 外层失败 | 38 confirmed；19/19 外层通过 |
| macOS | 27 confirmed；9/13 外层失败 | 27 confirmed；15/15 外层通过 |
| Windows | 29 confirmed；10/16 外层失败 | 29 confirmed；18/18 外层通过 |

回归先行失败项的实际内层均为 confirmed/0；不能把旧的普通场景绿色统计当成门禁可信性证明。旧 `chain-short` 已经 invalid，但原因是主进程等待超时；新版原因才是五秒后缺少一层负载。外层 PASS 不证明其清理完整，见 B1-R2。

修复后三个场景 ID 集合分别为 38、27、29，并集 **53**。workflow 的 Results 步骤再次打印 `WP25RESULT`，日志分别出现 76、54、58 行；按场景 ID 和执行阶段核对，不能重复计数。`WP25GATE` 单独统计，内层执行不加入外层机制汇总。

唯一顶层专用测试跳过是 `TestInnerForeignPID`：它只通过 `ownership.foreign-pid` 外层启动，在内层实际运行并得到预期 invalid/1。平台专属场景未编入其他平台，不算该平台通过；作者 WSL 缺少 pwsh 的 unsupported 也不转记为通过。

原型 Ubuntu 的 Prototype 步骤根据 `RUNNER_OS=Linux` 带 `-race` 执行；macOS/Windows 未启用原型 race。常规 Ubuntu CI 的普通 Go 测试同样带 `-race`；其八项 root 跨账户用例属于随后单独的 `sudo store.test -test.run CrossUser` 步骤，确实通过，但不能称为 root race 步骤。普通步骤中的跨账户跳过、真实旧 Git/unsafe Git 等设计跳过仍是跳过；没有据此重新验收旧模块。

常规 CI 不执行独立原型模块。两种 CI 成功均不能证明本报告 B1-R1 的一次性故障已被检出，也不能证明原型外的生产接线、事务或三平台 Runner 已完成。

## 8．证据支持范围、产品决定与剩余门槛

| 门槛 | 当前可以保留的局部支持 | 仍不能宣称成立的部分 |
|---|---|---|
| T1 / T6 | 正常排空、停止读取、D/J 阶段、封闭入口、别名、容量、最后文档；原始两个门禁反例及本轮对应负载的拒绝。 | 一切观察错误必失效、全部提前拒绝均安全回收；O3 永久阻塞任务自动释放；生产 Execute/输出接线。 |
| T2 | Linux 正常排他区间、完整链、pidfd 清扫及已有反例的 CI 观察。 | 部分链失败的收尾；生产全局闸门拓扑；全部启动/回收/信号路径的 C-9 约束。 |
| T3 | 所测 macOS arm64 上的退出观察、未回收组长与组身份；严格报告后的对应 CI。 | 其他 ABI/版本、枚举失败与重试预算的完整生产规格、脱离 session 的通用发现。 |
| T4 | 所测挂起启动失败链、Job 归属、30 后代负载、显式观察句柄关闭。 | 一次观察失败的 invalid 传播；全部原生失败点；真实异步终止窗口；生产期限及所有权移交。 |
| T5 | 本轮 CI 中所测适配器的历史/当前观察可继续按其字符串往返范围使用。 | 全版本、全脚本文本、实际解释器发现与 PATH/worktree 排除保证。 |

报告已正确把旧诊断 confirmed 汇总降为历史观测，没有证据说明旧正常记录一定实际损坏；失效的是门禁无假通过的总括。当前健康运行恢复了不少具体观察，但尚未恢复 B1 的完整拒绝保证，不能一概抹去健康观察，也不能把新绿色汇总作为穷尽证明。

**PRODUCT DECISION：没有新增已经批准或由本审查代选的项目。** 原有 14 项 D1～D9、D11、D13、D18、D19、D20 仍待负责人决定。C-4 对同一宿主后续调用的拒绝及其代价须在 D20 中明确；不是新增第 15 项，也不因门禁通过自动获批。

**ACCEPTED RISK：本轮没有新增。** 既有方案 B 只按已批准的局部边界适用，不豁免 Log 完整性、正常合作并发、误杀无关进程、其他账户或隐私保证；本报告的观察失效和拒绝收尾缺口不归入已接受风险。

冻结/实现授权前仍需：

1. 修复并独立核验 B1-R1、B1-R2；N4 的工具故障分类单独登记处理，不把它误写为机制反证。
2. 定清 C-9 的能力约束设计、Linux 闸门全局所有者及上下文 Git/排他区间先后、取消与 F1 宿主状态；macOS ABI、支持范围、枚举错误预算；Windows 父端期限、未覆盖失败点及资源移交；T5 解释器发现和 PATH 规格。
3. 负责人选择上述产品项，并同步草案规定的规范、验收和契约；本轮不替代该决定。

授权以后、生产验收以前再完成生产 Runner/CLI、真实取消和继承句柄、配置固定、空批次、首次 worktree 与 seq、迁移/Schema、事务与 §8.6 的组合失败测试、闸门及 C-9 接线和三平台生产回归。本原型没有覆盖这些能力，**不把这些后续工作自动列为本 PR 当前的 BLOCKER**。

## 9．最终回答与接手范围

**A．Review 075 B1：不能整体关闭；PR #33 技术 NO-GO，暂不建议合并。** 两个原始明确反例已拒绝，但仍有一次观察失败后的假通过，以及 Linux 短链拒绝清理缺口。

**B．Review 075 N1～N3：本轮处理成立。** N1 的 C-4/D20 待决关系、N2 的句柄关闭与不再访问关闭句柄、N3 的父端期限/内部定时器及版本记录，均按适用范围收口；不由此推论所有资源与失败分支已验证。本轮新增 N4 为工具故障的错误分类。

**C．修复后证据：支持表中所列的局部机制和已运行的负载/协议拒绝，不能宣布 T1～T6 全部满足或 C-1～C-18 已冻结。** 14 项产品选择仍未批准，生产 WP2.5 未授权。

**D．后续最小接手：实施方仅修原型的错误累积和不完整链收尾，补真实内层/外层拒绝与资源核验，并据实际结果调整证据说明；再由独立审查核对。** 不需要借此重写生产 Runner、重开既有已验收模块或降低保证。

结束核验：远程身份、PR open/未合并状态与开始一致；仅新增本未提交报告，未修改其他原仓库文件，未提交、推送、合并、改基或触发 workflow。审查到此停止，等待负责人决定。
