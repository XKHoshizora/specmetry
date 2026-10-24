# Codex Review 087：PR #34 统一工具状态与完成裁定的定向复审

## 1．结论

**技术 GO。Review 086 的 B-N4-R/T、B-N4-R/I 可以分别关闭；N4 系列在当前受控 helper 协议范围内收口。** 没有发现新的确定 BLOCKER。N-RENAME 的现行补充说明符合已验证的临时目录布局，措辞事项可以关闭。

技术结论与最终常规 CI 分开：run 40116851180 **attempt 1 正式 failure，7/8**，Windows 的 `TestSeqIsUniqueAcrossConnections` 报 SQLITE_BUSY。审查期间由实施方执行获准的 `--failed` 重跑；最后核验为 **attempt 2 正式 completed/success、8/8**。只有 Windows 测试实际重跑，七项成功结果沿用。**当前 CI 合并条件已满足，可以建议合并固定对象的原型 PR #34**；不能把它改写为首轮成功或存储维护问题已解决。

这只是原型修复的技术裁定，不是 T1～T6 全部成立、规范冻结或生产验收。十四项产品选择未批准，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14/M-15/M-16 保持未解决。本轮没有批准保证调整或新增接受风险。

## 2．固定身份、工作区与范围

| 项目 | 实际审查对象 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| 固定 base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| 固定 HEAD | `245b51bea8bd57e28c2716a86c67ca18a91bafaf` |
| Review 086 对象 | `bc5ad7841e31eefe08194019eed3282b72c2b7c0` |
| 最终代码 | `f728e11498a8b4bc5b71ec3cbff1618c795f6969` |
| HEAD 树 | `1089bf4df49e72b6258ab7aee122f25229e0de48` |
| 代码与 HEAD 的原型子树 | `0803171a13905e21d53a6c649b9ca3cd87ca223e` |

开始核验为 **2026-10-24 18:04:47 UTC / 2026-10-25 03:04:47 JST**：远程 HEAD/base 与上述对象一致；PR open、未合并、auto_merge=null。本地 HEAD 同一对象，工作区干净，报告 087 尚不存在。结论仅适用于固定对象。

已完整阅读适用 [AGENTS.md](../../AGENTS.md)、[Review 086](codex-review-086.md)、[原型文档 §2.18](../wp2.5-prototype.md)、[处理记录 §102](../review-resolution.md) 及完整提交增量。此处以章节编号定位；不依赖自动生成的标题锚点。源代码位置均指固定 HEAD。

相对上轮共 **10 文件，+932/−21**：五份文档及五份原型文件。提交依次为：

| 提交 | 范围 |
|---|---|
| `1a1caf279e3f8eb878354d42ab44bac7d94dfbbb` | 原样归档 Review 086 |
| `750e1390d6cc8cc211c144469d582cad239aca77` | 真实 helper 回归先行 |
| `9e3b7ca953357273ef4638d7200b5b4df2f4b88d` | 统一 toolState 与直接检查 |
| `f728e11498a8b4bc5b71ec3cbff1618c795f6969` | limit-after-done 场景等待实际裁定 |
| `188be6a86deb03276d32c752692c7fa18722d00e` | 完成协议、证据与边界文档 |
| `245b51bea8bd57e28c2716a86c67ca18a91bafaf` | §102 与活跃状态同步 |

五个原型文件为 `gate_test.go`、`helper_test.go`、`pipe_test.go`、新增的 `tool_test.go`、`tool_check_test.go`。代码点至 HEAD **只有 AGENTS、implementation-plan、review-resolution、wp2.5-prototype 四份文档变化**；prototype/.github 差异为空。internal/ 与 base 完全相同；生产代码、根依赖、Schema、迁移、ci.yml、原型 workflow 和既有冻结契约未变。

Review 086 的 SHA-256 为 `26c7c1c0fd585627c961ba57e6f27945343f02628f9d1f0bd612b96d128bedd2`，与归档登记一致；原审查身份、NO-GO 和两个残余未被后来状态替换。resolution 仅追加 §102；§84～§101 未改写。§102 准确保持“已修复、待独立复审”，没有代替本轮宣布关闭。

## 3．统一工具状态：按真实拓扑裁定

### 3.1 状态、报告与完成

[tool_test.go](../../prototype/wp25/tool_test.go) :47–113 保存首个错误、期限裁定、完成尝试和发布结果：

- `failed` 不清除先前错误；`report` 即使被调用方忽略返回值，也保存操作错误。
- `complete` 首次设置 final，并在同一把锁下检查已有错误与 limit；整个发布函数执行期间持锁。发布失败保留错误，不能通过第二次 complete 覆盖。
- `published` 成立后，report 不运行操作函数；期限裁定不再报告 limit。两个保护分别存在。
- 输出关闭的真实错误经 `tool.failed` 保存；完成文件的写入、Close、Rename 错误由发布函数返回并保存。

[helper_test.go](../../prototype/wp25/helper_test.go) :41–48、149–158、292–340、390–405、436–516，连接了真实期限回调、模拟期限、输出关闭、报告和完成。`reportOnce` 仅由统一 report 包装调用；`publishDone` 仅由统一 complete 包装调用。没有发现当前正常 holder 路径绕过状态直接发布业务报告或完成。

报告的状态检查与错误保存各自持锁，实际文件操作在锁外；**不能仅凭类型有 mutex 就声称所有操作串行**。这里安全的依据是实际调用拓扑：

1. 主 goroutine 同步完成报告操作之后才尝试完成，没有第二条主路径同时发布。
2. 唯一并行的真实期限回调，必须先在锁下赢得 limit 裁定，才会报告；此时完成已被禁止。
3. 若完成已发布，回调输掉裁定，不开始报告；若回调在发布期间到达，则等待发布终态。

所以在当前拓扑中，不会由 report 的锁外区间漏入一个成功发布。没有把任意新增的并行 report/complete 调用者假定为现有事实；以后改变拓扑时必须重新核查该前提。

故障注入的调度标记、命中日志、见证文件与 Windows marker 仍有其他文件写入；它们不是完成声明本身，各有负载/见证门禁。切断报告的故障分支仍使用延迟和部分写入来构造无效观察，不能把这些注入操作概括为任意文件错误已获穷尽证明。当前真实正反例没有由这些辅助文件形成绿色假通过。

### 3.2 “回调进入”的三个事件

必须分开：**回调函数开始执行 → 取得锁并作 limit 裁定 → 实际开始报告**。

本次有效性边界由第二个事件线性化：期限先赢得裁定，工具已经失败，主路径不能发布 done；完成先取得发布区间，回调可以在物理上开始，但要等待终态，发布成功就不报告，发布失败则仍可取得失败裁定。不能把 goroutine 已被创建或函数已开始，写成 limit 已经设置。

真实 `limit-race-*` 闸门在取得裁定之后才留下标记并等待，完成等待该标记，确实复现了原先“期限已裁定、报告仍在进行”的反例。`limit-after-done` 的特殊前置闸门则故意构造相反顺序；不是正常 helper 增加的任意等待。

锁内文件发布不等待父端、另一入口或回调，因此未发现相互等待的环；父端完成等待、故障闸门及场景收尾都有既定期限。但 **文件系统调用及调度本身没有由 mutex 获得硬时间上界**，见 §6 的非阻塞措辞建议。

## 4．原反例：独立旧版/新版复验

在仓库外由固定 Git 对象导出两个副本：回归先行 `750e139` 与固定目标 `245b51b`。实际运行编译后的真实原型 helper、场景、外层门禁，读取内层 verdict、退出状态、命中和最终清理，不是只接受 TestGate 的 PASS。

| 真实门禁 | 旧机制：内层 / 退出 | 固定目标：内层 / 退出 | 新版命中 |
|---|---|---|---:|
| report.limit-race-epipe | confirmed / 0，外层 FAIL | invalid / 1，外层 PASS | 1 |
| report.limit-race-ok | refuted / 1，外层 FAIL | invalid / 1，外层 PASS | 2 |
| report.ok-fails-epipe-reported | confirmed / 0，外层 FAIL | invalid / 1，外层 PASS | 3 |
| report.epipe-fails-then-ok | refuted / 1，外层 FAIL | invalid / 1，外层 PASS | 3 |

两个期限反例使用提前到 100 ms 的真实 AfterFunc 与可释放闸门。后两项真实以只读文件句柄执行报告写入而失败；另一记录随后确实写入成功。没有直接伪造 verdict，也没有用启动前失败替代目标路径。

旧版四项全失败、进程整体非零；正常排空及健康 F1 仍 confirmed/0。新版四项全拒绝，同时：健康 F1 confirmed/0，保留读端的合法反证 refuted/1，epipe-late confirmed/0，Rename 失败 invalid/1。不是“一律 invalid”。

因此 **B-N4-R/T、B-N4-R/I 分别关闭**。后续报告成功、复制结束和清理成功均未使本次已保存的错误恢复完成资格。N4 系列在受信任 helper、固定入口与受控文件布局下收口；完成标记不是不可信写者认证、断电持久性或任意宿主协议证明。

## 5．完成后回调、发布期间裁定与回归有效性

### 5.1 f728e11 的真实裁定等待

[pipe_test.go](../../prototype/wp25/pipe_test.go) :318–325 在完成读取之后、结束 holder 之前，限时等待 `.limit-decided`，未命中由 needLoad 使场景 invalid。helper 在**真实执行裁定后**写标记并记录第二次命中。

本机固定目标及三平台最终诊断均为 **confirmed/0、两次命中**。已核对中间 `9e3b7ca` 完整日志：Ubuntu/Windows 只有一次命中，macOS 两次；前两者没有证明所需顺序。最终修复没有把旧空通过抹去。

独立最小撤回：只抑制 `.limit-decided` 的写入，保留实际回调、裁定和命中。新场景仍记录两次命中，但因缺少裁定标记为 **invalid/1**，外层 FAIL，整次非零；正常排空仍通过。说明本次不再仅凭一个成功结果或一次早期命中放行。

### 5.2 发布期间的独立进程内对照

在外部观察副本给 `limitEntered` 的函数入口增加仅用于观察的闸门：先确认发布函数已进入且仍持锁，再确认回调已实际进入 limitEntered，然后释放发布。没有改变锁、状态或发布结果。

两项固定预期均通过：发布成功→回调输、后续报告拒绝；发布返回错误→错误保存、回调赢、第二次完成拒绝。没有只以启动 goroutine 后睡眠，推断它已经执行。这是**进程内调度证据**，不是一个真实文件发布卡顿的端到端 helper 场景。

作者 M66～M72 为作者 Windows/WSL 辅助证据；本轮没有把它们改称独立全量撤回。M70 目前只有进程内检查，不自动构成缺陷；M68/M71 单独撤回由另一层保护，因此不要求每撤一道冗余保护都使真实链路失败。M72 的双撤回及其原始记录保留。

## 6．BLOCKER、NON-BLOCKER 与保证范围

**BLOCKER：没有新增的确定技术缺陷。** 最终常规 CI 的正式状态是独立合并检查，见 §9；不因存储维护测试失败自动认定本次原型逻辑错误，也不自行豁免该检查。

**NON-BLOCKER N-087-1：直接检查没有确定证明回调已进入。**

位置：[tool_check_test.go](../../prototype/wp25/tool_check_test.go) :72–110，尤其 87–94。发布入口有显式通知；回调只有 goroutine 启动和 100 ms 等待，没有回调入口通知。独立对照把该 goroutine 明确改成等发布 release 后才调用 limitEntered，此测试仍 PASS，证明它单独不足以支持“发布期间已进入”的精确措辞。

这没有产生当前机制绿色反例：互斥的源码关系、两条真实先赢期限回归、完成后真实回调及本轮显式入口对照相互支持。建议后续用入口握手收紧这项测试，或把证据说明限定为所实际证明的范围；不把它列为本 PR 新的合并阻塞。

**NON-BLOCKER N-087-2：不要把“没有协议循环”写成无条件“没有无界等待”。**

位置：[原型 §2.18](../wp2.5-prototype.md) :1686、[resolution §102](../review-resolution.md) :6052。complete 持锁执行文件 Open/Write/Close/Rename，源码没有为这些系统调用提供硬截止。当前父端、闸门和正常收尾有期限，且锁内没有反向调用入口，足以支持本轮受控实验；不能由此推出所有文件系统延迟都有上界。

最小建议是写成“未引入协议级循环等待；父端期限不变；文件 I/O 与系统调度不提供硬实时保证”。未构造当前受控布局中的不可释放 I/O，也没有据一般性未知要求引入新的生产机制或风险批准。

**N-RENAME：可以关闭措辞事项。** §2.18/§102 明确纠正旧“全有或全无”表述，限定自己的同目录临时文件与已验证三平台布局，并严格读取完整内容。旧段落作为历史保留并有后文更正；没有声称 os.Rename 自动提供所有 Windows 文件系统的原子性。当前本机为 NTFS，正常发布与失败对照成立。

**PRODUCT DECISION：没有本轮新增的负责人待决事项。** 十四项 D1～D9、D11、D13、D18、D19、D20 仍待负责人选择；本审查不批准其中任何一项。

**ACCEPTED RISK：本轮不新增。** 方案 B 既有局部范围不变，不能由本原型 GO 扩大到 Log 完整性、误杀无关进程或其他保证。旧 macOS PID 2088/1989 的归属、EINVAL 根因、当时完整清理继续未知；新绿色不回填，也不把不可追溯本身当永久合并门槛。

## 7．本轮实际运行与直接回归

平台：**Windows 11 / 10.0.26300 / amd64、Go 1.27.1、C: NTFS**。使用已有工具，GOTOOLCHAIN=local、GOPROXY=off。副本、编译产物与日志只写在仓库外本轮 review087 目录；未安装升级工具、运行用户项目配置或改变原仓库实现。

| 独立运行 | 期限与结果 |
|---|---|
| 旧机制四项真实门禁 + 两项健康场景 | 60 s；四项预期外层 FAIL、整体 1；健康保持 |
| 固定目标定向回归 | 420 s；PASS/0，**78 门禁、13 组合、3 个顶层机制 confirmed**；TestGate 303.45 s |
| 发布期间实际入口对照 | 20 s；成功/失败两组、原五组 ToolState 与正常排空，PASS/0 |
| 缺少回调裁定标记 | 20 s；目标内层 invalid、外层 FAIL，整体 1；健康排空通过 |
| 延后进入的测试证据对照 | 20 s；原 TestToolState 仍 PASS，支持 §6 的证据限制；不是生产算法反例 |

固定目标的选择为：

`^(TestGate|TestGateCombos|TestToolState|TestF1Classify|TestStopRead|TestStopAPIErrors|TestLingeringFailedStop|TestPhaseClock|TestCandidates|TestGateJudge|TestPipeNormalDrain|TestPipeEscapedHolder|TestPipeF1SkipDrain)$`

五组 ToolState、23 行 F1Classify、停止任务/API 错误、晚到错误与资源移交、同一时钟/固定预算、组合判定和归属保护均通过。forge-owner 仍拒绝非自有对象、零动作/零信号；混合场景的自有遗留实际清理。补救失败、工具错误或最终状态未知不能使被测自检 PASS。未重做全部 T1～T6、生产 Runner 或已验收模块。

**首次失败和无效准备保留：**

1. 第一次导出在外部目录误用未指定仓库的 git archive，返回 not a git repository；尚未运行实验。改为显式只读指定原仓库后导出成功。
2. 第一次直接启动测试的 PowerShell 参数未整体引用，测试二进制报未知 `-test` 标志并退出 2，没有运行场景；修正为参数数组后运行。
3. 第一条旧版筛选没有匹配门禁子测试，只运行两个健康场景；其 PASS 不计为旧反例证据。修正子测试筛选后实际取得四项误分类。
4. 结束核查的 CIM 接口不可用，不计产品故障，也没有安装或绕过该接口；随后用可用的进程路径枚举与各场景清理账目核查本轮对象，未发现 review087 二进制进程残留。

原生测试运行经工具权限审查，只针对仓库外自建、有测试硬期限的进程与临时文件；本轮没有重新把上一轮受限环境拒绝 Rename 的结果算作产品失败。本轮编译均成功；没有用编译失败充当变异检出。

本机没有动态运行 Linux/macOS；相关结论由源码和相应完整 CI 日志支持。注入的是受控错误及调度，不是原生磁盘故障、断电、所有文件系统或任意不可信程序证明。

## 8．原型诊断与历史运行

[最终原型 run 40110917799](https://github.com/XKHoshizora/specmetry/actions/runs/40110917799)：**push，attempt 1，completed/success**。已取得三项完整作业日志，实际 checkout 均为 `f728e11498a8b4bc5b71ec3cbff1618c795f6969`。

| 平台 / Job ID | 机制 confirmed | 门禁 PASS | 组合 PASS |
|---|---:|---:|---:|
| Ubuntu / 120922265081 | 38 | 82 | 13 |
| Windows / 120922264993 | 29 | 78 | 13 |
| macOS / 120922264918 | 27 | 73 | 13 |

机制场景 ID 去重并集 **53**；Results 步骤重复打印不重复计数。内层 invalid 被正确拒绝是门禁 PASS，不是机制 confirmed；组合也另计。

Ubuntu 实际 Linux 分支设置 -race 并执行原型 go test，未报告竞态。三平台的 ToolState 五组、F1Classify 23 行、PhaseClock、StopRead、LingeringFailedStop、GateJudge、Candidates、StartPoints 实际运行；Windows StopAPIErrors、macOS CensusUnreadable 各在适用平台通过。三项顶层设计 SKIP 为 TestInnerForeignPID、TestInnerLeak、TestInnerStopError；它们由相应门禁内层执行，SKIP 不计通过。

已核对四项新拒绝和健康/合法反证，三平台结果与 §4 表一致。最终 limit-after-done 各两次命中。代码点到 HEAD 没有原型差异，因此这套诊断对 HEAD 内容适用；不是文档 HEAD 又运行了一次原型。

历史分开记录：

- [40109659805](https://github.com/XKHoshizora/specmetry/actions/runs/40109659805)，750e139，push/attempt 1/failure：三项完整日志均只有四项新门禁失败，Ubuntu 77/81、macOS 68/72、Windows 73/77；机制 38/27/29 confirmed、组合各 13。最初作者缺少保留读端、零命中的无效夹具保留，不算有效检出。
- [40110180232](https://github.com/XKHoshizora/specmetry/actions/runs/40110180232)，9e3b7ca，push/attempt 1/success：其完整日志证明 limit-after-done 在 Ubuntu/Windows 只有一次命中，macOS 两次。前两项空通过不能支持该顺序；f728e11 才补齐。
- 同期常规 [40109661506](https://github.com/XKHoshizora/specmetry/actions/runs/40109661506)、[40110181716](https://github.com/XKHoshizora/specmetry/actions/runs/40110181716) 正式 cancelled；[代码点常规 40110920228](https://github.com/XKHoshizora/specmetry/actions/runs/40110920228) 正式 success。没有把这些旧绿色/取消改写成最终 HEAD 首轮成功。
- M71 第一次编译失败、作者 WSL 的 StartPoints 缺源码跳过及无 pwsh 的 unsupported 均保留为作者证据边界，未计为本轮独立运行或三平台全通过。

## 9．最终常规 CI、重跑与合并条件

[run 40116851180](https://github.com/XKHoshizora/specmetry/actions/runs/40116851180) 为 pull_request，HEAD 是固定目标。八项首轮日志实际检出均为：

`c0ec2e49ed6472ac6ac2ffe4c2cbbd1c55acc2db`

GitHub commit API 核对父提交依次为固定 base 与固定 HEAD；树为 `1089bf4df49e72b6258ab7aee122f25229e0de48`，与 HEAD 相同。

| Job | attempt 1 ID / 结果 | attempt 2 ID / 来源 |
|---|---|---|
| Go tests Windows | 120939788019 / **failure** | 120946480814 / 实际重跑、success |
| Go tests macOS | 120939788110 / success | 120946481556 / 沿用 |
| Go tests Ubuntu | 120939788118 / success | 120946505783 / 沿用 |
| Go lint | 120939787867 / success | 120946518491 / 沿用 |
| Web | 120939788121 / success | 120946482382 / 沿用 |
| Build Windows | 120939862188 / success | 120946482338 / 沿用 |
| Build macOS | 120939862209 / success | 120946507581 / 沿用 |
| Build Ubuntu | 120939862292 / success | 120946481904 / 沿用 |

首轮 Windows 的精确失败是 `store_test.go:546: database is locked (5) (SQLITE_BUSY)`，`TestSeqIsUniqueAcrossConnections` 11.97 s FAIL；同作业的 AcrossProcesses 通过。其他七项首轮均正式 success，首轮总体 failure/7-of-8，不得删去。

重跑 Windows 始于 **2026-10-24 18:05:16 UTC**。其余七项 attempt 2 虽有新 Job ID，但 started_at/completed_at 与首轮相同，是沿用成功结果，不是八项全部重跑。没有由本审查触发任何运行。

Ubuntu 普通常规测试的 Linux 分支启用 -race；随后独立 root 跨账户步骤运行并通过八项跨账户路径测试。普通非 root SKIP 与 root 实跑分开；root 步骤不是普通 Race Detector 的替代。macOS/Windows 的 root 步骤按平台跳过。真实旧 Git、默认关闭诊断和平台权限等设计跳过不计通过。

常规 CI **不运行独立原型模块**。本轮未改存储层；失败特征与已有维护观察相似，但不能据路径未改、代码点绿色或重跑成功推定根因已确定、与本次改动绝对无关，或关闭 M-14/M-15/M-16。

最后查询：**attempt 2 completed/success，八项均正式 success**。Windows 重跑于 18:18:19 UTC 结束；AcrossConnections 此次 3.29 s PASS，CLI 381.635 s 通过。八项 attempt 2 的完整日志已再次取得，实际 checkout 全部仍为同一 `c0ec2e49...`；沿用日志的时间戳与原日志相同。Windows 运行记录为动态 33 包、32+1 分组，CLI 保持 20 分钟预算，其他包默认预算；该数量仅是当前快照观察。

因此，**本轮技术 GO 与当前 CI 条件均成立，建议可合并固定对象的原型 PR**。不是用旧绿色替代首轮失败，也没有自行触发第二次重跑或批准任何失败豁免。

## 10．接手边界

当前没有必须补做的原型代码修复。§6 两项是局部测试/措辞改进建议，不新增产品审批；如作后续修改，只需相称的定向核验。固定 HEAD 的最终正式 CI 已核实成功；后续若对象变化，需重新判断新增差异及证据适用性，不能由本审查豁免失败或代为重跑。

冻结前仍缺既有十四项产品选择，以及 C-4/D20 的 O3 资源及同宿主后续调用、Linux 生产级闸门/C-9 全启动回收拓扑、macOS ABI/支持前提、T5 入口/PATH 等规格和必要原型。当前完成状态修复不能代替这些门槛。

获授权后的生产阶段才涉及 Runner 接线、记录/事务、取消、固定配置、空批次、§8.6 持久化组合和 CLI/E2E。没有把未授权的生产工作列成本 PR 阻塞；没有宣布 WP2.5/M2 验收。

本轮只新增本未提交报告，历史原文、被审查代码及其他任务内容保持不变；未提交、推送、合并、改基、安装升级、触发/重跑 CI，未批准新产品选择、保证调整或接受风险。

**结束核验（2026-10-24 18:20 UTC / 2026-10-25 03:20 JST）**：远程 HEAD/base 仍为固定对象，PR open、未合并、auto_merge=null；原型 run 40110917799 仍为 attempt 1 success；常规 run 40116851180 为 attempt 2 completed/success，首轮失败记录保留。本地仍在固定 HEAD，仅新增未跟踪报告 087，跟踪及暂存内容无差异；Review 086 哈希未变，相对链接目标均存在，未发现本轮 review087 二进制进程残留。没有把其他提交或实施方重跑操作混入审查授权。
