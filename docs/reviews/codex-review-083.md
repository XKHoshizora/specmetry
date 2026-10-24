# Codex Review 083：PR #34 停止错误链、N4-R 与 macOS 诊断失败的定向独立复审

## 1．结论

**本轮修复技术 GO：Review 082 的 B1-Ra、B1-Rb 可以关闭，未发现新的 BLOCKER。N4-R 尚不能整体关闭，保留 NON-BLOCKER。**

**当前不建议合并 PR #34：最终代码点的原型诊断为 completed/failure，macOS 的保险清理未能确认完整结果；最终常规 CI 8/8 不能替代它。** 技术修复成立与当前合并条件满足是两项判断。

| 问题 | 裁定 |
|---|---|
| B1-Ra：停止晚到错误及移交 | 关闭。移交持续保存结果的任务，挂起时已有错误和完成后的错误均保留；原反例旧树绿色、新树 invalid |
| B1-Rb：Windows API 失败 | 关闭。ERROR_NOT_FOUND 与实际错误分开；失败持续保存，关闭副本失败进入结果；不把请求取消当作完成 |
| N4-R | 旧组合已修，仍有 **ok 后发生真实工具失败** 的窗口，错误 refuted，见 §4；继续非阻塞，没有绿色假通过 |
| N6 | 未发现直接回退，保留 Review 081 的原时钟口径关闭范围 |
| macOS 最终诊断失败 | 真正的外层工具错误；不是预期截断拒绝本身失败。相关 PID 归属与完整清理结果仍未知 |
| 新 BLOCKER | 无。没有将非阻塞分类残余或未知的 macOS 原因自动升级为生产缺陷 |
| 当前合并条件 | **未满足**。不以中间 SHA 绿色、常规 CI 绿色或“源码未改”豁免最终诊断 |

没有批准任何新的产品选择、保证调整或接受风险。14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14、M-15、M-16 保持未解决。

## 2．固定身份、范围与记录

| 对象 | 身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| Review 082 对象 | `0f1f20b8227bcec097a34c42cca393aaf22254cb` |
| 回归先行 | `e4eadfa9ae3e6e4d21a7532bdd59ee33494f87cd` |
| B1 修复 | `920220c2b3c7fee89b9a30371ac62734d0307c34` |
| N4-R 修复 | `63b74324ee1e61efdcba9d98f3cd7ed810b291b6` |
| 两部分独立门禁补充 | `ba32dbe2c378cad96a18acaadfd6f7095f9726a0` |
| 最终代码 | `2413634ba21838f7275ee4ddd523779f1409c9b6` |
| HEAD | `b51cc5e571dc7f1cad00f49055d862e08a5833f0` |

开始核验为 **2026-10-24 09:27:56 UTC**；技术核查结束时于 **09:41:33 UTC** 再次核对。远程 HEAD/base 均符合固定对象，PR open、未合并、auto_merge=null。没有自行改换对象。开始与技术核查结束时本地 HEAD 均为固定目标、工作区干净，报告不存在。落盘后另作最终核验。

已读取适用 AGENTS、完整 [Review 082](codex-review-082.md)、本轮完整增量、[原型文档](../wp2.5-prototype.md) §2.14 与错误传播表、[处理记录](../review-resolution.md) §98，以及停止、复制、移交、终判、登记、helper/报告、普查和保险清理调用链。

相对上轮 **16 文件，+1130/−76**：11 个原型文件（dest、destscen、gate、helper、main、pipe、report、stop_api_windows、stop_check、stop_unix、stop_windows），5 个文档文件。提交顺序为 986ae62 归档、e4eadfa 回归、920220c 停止结果、63b7432 N4-R、ba32dbe 两部分门禁、2413634 确定性无请求/迟到记录、6742a3f 文档、b51cc5e 状态。

**2413634 → HEAD 两个提交只有四份文档变化**；prototype/.github 无差异，原型子树均为 `95316ac04a1c0ae7b6fce0e79e5fba74d8cc8fac`。生产代码、根依赖、Schema、迁移、ci.yml、原型 workflow、check-runner 草案及冻结契约未变。

Review 082 归档 SHA-256 为 `fe853b8906cf1458d2838da4431996672d9ac5dea75022a6a907f40fee99ab2e`，与 §98 相符。review-resolution 只追加 63 行，§84～§97 和历史报告未改。§98 保留原对象及 NO-GO，没有把实施方“已修复”当作审查关闭。

## 3．B1-Ra、B1-Rb：可以关闭

### 3.1 完整错误链与终态读取

| 环节 | 核查结果 |
|---|---|
| API | File.Close、实际 CancelIoEx 失败、关闭自有副本失败均有传播路径 |
| 停止任务 | [dest_test.go:305](../../prototype/wp25/dest_test.go#L305) 用 mutex 保护错误列表；后来的 nil 或成功不清除旧错误 |
| 阶段 J | 仍先建立唯一 deadline，再异步启动各流停止，没有恢复 J 前同步阻塞 |
| 已完成的停止 | runS6On 将终态错误放入 stopErr，adopt 消费 |
| 挂起的停止 | [473 行](../../prototype/wp25/dest_test.go#L473) 移交绑定任务的 until 方法，后续读取持续结果，不是移交时的错误快照 |
| 接手者 | [main_test.go:298](../../prototype/wp25/main_test.go#L298) 在共同 5 s 截止下逐项读取 ended/error；未结束和错误分别记录 |
| 终判 | invalid 优先；复制结束、进程清理成功、预测成立都不能清除它 |
| 登记 | [destscen_test.go:43](../../prototype/wp25/destscen_test.go#L43) 保留带错完成项，failures 也读取挂起项的已有错误 |

**没有重现最终读取与完成之间的绿色窗口。** 工作函数先保存返回错误，再关闭 done；until 收到 done 后读取加锁结果，此后该任务不会继续追加错误。若截止时只读到 pending，无论错误此时是否已经到达，场景都会因未结束而 invalid。边界上保守读到 pending 不是绿色假通过，接手者仍可读取后来的终态。

多流各自移交并逐项消费；stop.error-one-of-two 实跑证明正常流不会掩盖错误流。阶段 D 内完成的流由 [dest_test.go:459](../../prototype/wp25/dest_test.go#L459) 保存关闭错误，不再忽略。

refuse 候选在挂起或带错完成时拒绝；allow 候选可以放行，但失败记录仍在。没有代批 C-4/D20。场景与登记读取同一任务结果，不会各自重复 Close。

### 3.2 原反例、阶段 D 与正常对照

本轮复用 Review 082 的仓库外独立 Closer 用例，**不修改原判定**：Closer 等待 release、释放自己的 gateReader，再返回同一 sentinel error。分别放到旧机制 e4eadfa 和目标树，经过真实 S6、adopt/done、内层进程和外层门禁。

| 独立用例 | 旧机制 | 目标树 |
|---|---|---|
| 错误在 S6 返回前到达 | invalid / 1 | invalid / 1 |
| 同一错误在 S6 返回后到达 | **confirmed / 0，外层 FAIL** | **invalid / 1，外层 PASS** |
| 正常真实管道 | confirmed / 0 | confirmed / 0 |

目标树晚到错误 reason 为 `observation failed: the stop of out: review082: real stop function returned an error`。不是仅验证辅助函数返回错误。

另外独立补充：

- 健康流在 D 内 EOF，随后 Closer 返回错误：仍为 O1、没有请求停止，但场景 invalid，外层 PASS。
- stopTask 仍 pending 时保存错误：读到 `(false,error)`；完成后错误仍在；登记 refuse 拒绝、allow 放行且保留失败。此项是受控任务状态测试，不冒称原生内核故障。

已提交的 stop.* 七项、TestLingeringFailedStop、TestStopRead 三行在本机通过。旧机制的运行后、复制后、两路之一三条错误路径确实 confirmed，登记错误放行；回归失败原因与 Review 082 相符。

### 3.3 Windows API 契约

[stop_windows_test.go:87](../../prototype/wp25/stop_windows_test.go#L87) 将 ERROR_NOT_FOUND 单独视为没有可取消请求的竞态；第一次其他失败立即回调保存，后续失败计数，完成时保留次数摘要。后来成功不清除第一次失败。

取消找不到请求不表示取消已完成；成功也仅表示请求取消。这符合 [CancelIoEx 官方契约](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelioex)。当前仍等 File.Close 返回，再检查 [82 行](../../prototype/wp25/stop_windows_test.go#L82) 的自有句柄释放结果。DuplicateHandle 失败也在任务中返回错误，没有把同步等待移回 J 前。

身份与释放顺序保持：SyscallConn.Control 引用下取得不可继承副本；取消只用该副本；原 File.Close 返回后关闭副本并退出，ticker 随之停止。没有在释放副本后继续取消，也不操作可能复用的原句柄数值。

**释放失败不等于已释放。** 当前会 invalid、释放未经确认，不再盲目关闭第二次。资源可能保留到进程结束；本轮不把“不重试”称为“已释放”，也没有证明原生失败时无资源保留。故障夹具自己捕获并真实关闭了留开的副本。

本机实跑 TestStopAPIErrors：

1. **nothing-to-cancel 的证据确定**：[31 行](../../prototype/wp25/stop_api_windows_test.go#L31) 包装真实 CancelIoEx 计数；RawConn 闸门保持关闭，只有确认至少一次 ERROR_NOT_FOUND 后才在 54 行开闸。不是随机调度碰巧命中。最终无错误、O2。
2. cancel-fails：接缝返回 ERROR_ACCESS_DENIED；自己的真实写端释放读取后，错误仍在结果中。
3. own-handle-close-fails：接缝不关副本并返回错误；结果有错误，夹具随后真实关闭自己的副本。

后两项是 API 故障注入，不是原生失败。旧机制两项均错误返回 nil，新机制均正确保留错误。

**B1-Ra、B1-Rb 分别可以关闭，Review 081 B1 在当前原型范围内可整体关闭。** 不推广到任意驱动、任意句柄或调度停顿。挂起资源上界、后续宿主策略仍是冻结前门槛。

## 4．N4-R：精确旧组合已修，回答之后仍有残余

### 4.1 当前有效证据

每场景独立报告文件及一次 query，唯一 holder 看到 query 后写自身 PID 的 ok；只在此前写入未失败的循环入口回答，随后继续写。关联依赖受信任 helper 和独立报告文件；调用者没有另外认证每个 ok 的身份，也没有通用多轮询问序号协议，不能扩大为身份认证。

报告仍严格解析，重复与场景清单最终检查继续生效；运行中未完成尾行可以等待，写者结束后的截断必须 invalid。

[pipe_test.go:217](../../prototype/wp25/pipe_test.go#L217) 发问，等待 ok 后在 [225 行](../../prototype/wp25/pipe_test.go#L225) 重读 epipe；再要求每一路都有保留读端和相对停止计数的增量。没有再用求和或单路字节代替双路；计数错误会 invalid，停止/观察/清理错误仍由 done 优先判定。

本机实跑：

| 用例 | 内层结果 |
|---|---|
| control.f1-skip-drain | confirmed / 0 |
| control.f1-read-kept | refuted / 1 |
| report.epipe-unwritable、-alive | invalid / 1 |
| Review 082 单路字节、另一流及报告失败组合 | invalid / 1；旧机制错误 refuted |
| report.ok-unwritable | invalid / 1 |
| report.one-stream-seen | invalid / 1 |
| report.epipe-late | confirmed / 0；等待回答期间到达的真实失败记录参与判断 |

分别在三份隔离副本撤回一条门禁，均编译成功：

| 独立撤回 | 检出 |
|---|---|
| 去掉缺 ok 的拒绝 | report.ok-unwritable 错误 refuted，外层 FAIL |
| 去掉逐路字节拒绝 | report.one-stream-seen 错误 refuted，外层 FAIL |
| 去掉回答等待后的 epipe 重读 | report.epipe-late 错误 invalid，外层 FAIL |

每份副本的真实正常管道仍 confirmed；没有伪造 verdict，没有把编译失败算检出。它们证明三条检查分别有效，不证明全部组合时序完整。

### 4.2 NON-BLOCKER N4-R-S：ok 不是之后观察区间的完成证明

位置：

- [helper_test.go:106](../../prototype/wp25/helper_test.go#L106)：发现 query 后只回答一次。
- [108 行](../../prototype/wp25/helper_test.go#L108)：写 ok 后继续当前及后续写入；ok 的返回错误本身未传播。
- [pipe_test.go:220](../../prototype/wp25/pipe_test.go#L220)：取得 ok。
- [225 行](../../prototype/wp25/pipe_test.go#L225)：重读当时 epipe。
- [233 行](../../prototype/wp25/pipe_test.go#L233)：据累计双路字节与旧 ok 认定反证材料完整。

真实有效的 ok 仅证明回答以前的写入；之后仍可失败。旧双路字节和旧 ok 不会消失，而终态清单允许 epipe 为零，发现不了“后来的报告写不出”。

**仓库外独立组合保持真实写入/报告：**

1. 两路读端都保留，原 helper、归属、父子进程和报告机制保持。
2. holder 看到本轮 query，先正常写出真实 ok；此前全部写入均成功，没有虚假声明。
3. 随后关闭 helper 自己的 stderr 文件，下一次真实 Fprintf 返回错误；epipe 文件按既有接缝以只读打开，真实 WriteString 失败。
4. helper 仅在真实报告失败分支写测试闸门标记并继续存活；父场景取得 ok 后限时等到标记，再按原代码重读及计数。此处只控制调度，没有改判定。
5. 两路目标时段字节增量为 **[1800,1787]**，ok 已到、epipe 为零，仍 **refuted / 1**；独立外层要求 invalid 而 FAIL。两自有进程最终结束。

~~~text
load: the actual report failure occurred after ok
no epipe record; the holder confirmed its writes went on,
and late bytes reached every stream's kept read end after the stop: [1800 1787]
holder's late writes fail: 0 epipe reports
verdict=refuted; inner_exit=1; cleanup=verified: 2 processes gone
~~~

这不是要求证明无限未来：故障已经在本次最终判定之前发生。也不突破“helper 不虚假声明”的前提，它正确回答过去状态，后来才失败。

**N4-R 不能整体关闭，沿用 NON-BLOCKER。** 仍为工具故障误分 refuted/非零，没有绿色假通过，不升级为新 BLOCKER。

最小方向：把正面回答变成一个**有界且完整结束的观察段**，例如回答后不再继续产生待证明的写入并进入受控释放；或通过可关联的完成/失败状态结束这次观察。回答失败、回答后已发生的工具/报告失败须进入 invalid。补真实 ok 后故障回归并保留合法 refuted，不必无限增加证明层或提前完成 Runner。

在修正前，§2.14 的“工具/报告失败都 invalid”须限定实际覆盖区间；不能只用“helper 声明可信”排除时间窗口。

## 5．macOS 失败：事实、推断与未知

### 5.1 最终诊断的实际失败

[run 40074633075 的 macOS Job 120815263065](https://github.com/XKHoshizora/specmetry/actions/runs/40074633075/job/120815263065) 完整日志：

- checkout 最终代码 2413634。
- report.truncated 内层 invalid、exit 1；reason 为报告尾行 `pid 1969` 截断、5 s 未得完整 pid 记录、预期注册缺额一项。
- 被测外层盘点当时 remedy=not-needed、final=ended。
- 随后 insurance=`FAILED: census: process 1989: the environment cannot be read: invalid argument`。
- problems/tool_errors 加入 insurance，pass=false；门禁、Job、run 均失败。

**原预期截断拒绝成立。** expected_found=false 是没有指定显式 reject 类别时的字段，不表示内层没有拒绝。真正额外失败在外层清理验证。

### 5.2 读取与阶段

调用链：

1. [gate_test.go:711](../../prototype/wp25/gate_test.go#L711) 在被测路径判定后调用 insure。
2. [witness_unix_test.go:179](../../prototype/wp25/witness_unix_test.go#L179) 先处理可信旧盘点，再作新 censusTag。
3. [census_darwin_test.go:33](../../prototype/wp25/census_darwin_test.go#L33) 读取 `sysctl kern.procargs2`，即参数/环境数据。
4. 失败后复核 procInfo；只有可靠不存在、僵尸或启动身份变化才跳过，否则返回 census 错误。
5. 该日志符合 insure 中“未有已确认自有存活对象、新普查失败”的出口，不是终止动作失败。
6. [gate_test.go:712](../../prototype/wp25/gate_test.go#L712) 加入不可豁免工具错误，重判失败，没有把 EINVAL 当空集合。

这是有效的保守拒绝，没有据此向 PID 1989 发信号的证据。

### 5.3 证据能说明什么

| 事项 | 裁定 |
|---|---|
| 同 UID、启动时间范围 | 是代码筛选条件，不是归属证明 |
| PID 1989 的本轮标记 | 读取失败，没有获得这次 tag 正证据 |
| 创建记录 | 日志仅打印 created=2、unowned 中 1969:ended，未打印完整原始创建记录；不能单凭这些汇总独立重建作者“1989 不在记录中”的全部核查 |
| 即使接受“未登记” | 也不能排除未登记后代，不能推出无关 |
| 是否本轮对象 | **未知**；同用户/时间重叠不证明属于本轮，缺记录不证明不属于本轮 |
| EINVAL 内部原因 | 未知；失败时完整状态没有记录，不能排他归为正在退出或特定系统机制 |
| final=ended | 是保险清理之前的被测路径结果，不能升级为保险清理之后的全局确认 |
| 清理结果 | 已有记录相关对象有 ended 观测；新普查覆盖范围的完整清理证明未完成，不能声称全部无残留 |
| 新误杀/漏拒绝缺陷 | 没有证据；本次向失败方向传播，没有绿色假通过 |
| 与本轮修复绝对无关 | 未证明；源码未改是事实，不构成因果排除 |

**NON-BLOCKER N-MAC：归属/因果措辞过强。** 原型 §2.14、处理记录 §98 的“其他进程”“与本轮改动无关”等应收敛为“既有未改调用链出现读取失败、归属未确认”。没有独立正证据，不能说它必然无关；也不能说它一定是本轮遗留。需要分别写明阶段、未知项和证据缺口，不改历史审查原文。

### 5.4 回归先行的计划外失败是另一阶段

独立读取 [40072151282 的 macOS Job 120807717558](https://github.com/XKHoshizora/specmetry/actions/runs/40072151282/job/120807717558)：

- report.epipe-unwritable-alive 内层正确 invalid。
- **被测普查** 读取 PID 2088 环境返回 EINVAL，problems 为 census，final=unknown、pass=false。
- 后来 insurance=nothing left。

它与最终代码点“保险清理的新普查失败、此前 final=ended”不同阶段。表面同为 EINVAL 不证明同根因；第一次保险清理后来成功也不能恢复被测观察有效性。

### 5.5 分项裁定

A. 没有证据证明新增误归属、误杀或假通过代码缺陷；确实暴露**最终诊断和完整清理证明的缺口**，作者归属/因果表述不足。不是可忽略噪声。

B. 未变普查路径的失败不撤销 B1-Ra/Rb 的闭环源码和针对性证据；N4-R 不关闭另有 §4 独立反例。不用 CI 失败省略技术复审。

C. **最终原型诊断条件未满足，当前不建议合并。** 本轮未忽略 EINVAL、批准重试、重跑追绿或接受不明残留。

最小补证：先纠正文档，再由负责人决定是否另行授权插桩/有界验证，记录候选启动身份、状态读取结果、可得归属证据、前后盘点边界及保险清理结果。不对未知 PID 动作；过去 PID 的当时状态不能靠现在重查恢复。若拟把无法读取改为可忽略，是另一个保证取舍，须独立论证批准，本轮不代批。

## 6．独立验证、直接回归与边界

实际平台：**Windows 11 10.0.26300/amd64、Go 1.27.1**。使用已有工具，GOTOOLCHAIN=local、GOPROXY=off；所有编译、用例和变异均在仓库外 review083 副本，没有安装升级。

| 运行 | 实际结果 |
|---|---|
| 原样目标：Gate、Combos、StopRead、StopAPIErrors、LingeringFailedStop、PhaseClock、Candidates、GateJudge、三个真实管道场景 | PASS；3 顶层机制场景 confirmed、48 门禁 PASS、13 组合 PASS，不是本机重跑全部 29 个场景 |
| 回归先行 e4eadfa | 三项晚到停止错误错误 confirmed；登记放行；API 故障为 nil；旧混合报告故障 refuted，均被回归检出 |
| Review 082 原独立 Closer | 旧晚到 confirmed/0，新晚到 invalid/1；早到及健康对照保持 |
| D 后 Close 错误、pending/完成后登记保留 | 独立补充通过，见 §3 |
| 有效 ok 后真实工具/报告失败 | 错误 refuted/1，外层 FAIL，见 §4 |
| 三份定向撤回 | 各命中对应回归 FAIL；正常管道仍通过 |

测试进程都有 18～240 s 硬期限，内层沿用原有父端期限及归属/Job 保护。只操作自建且身份确认的进程、管道、句柄；最后没有 review083 测试进程残留，未作未知进程清理。

首次实际结果保留：旧机制、定向故障和变异按上述方式失败；原样新树首次通过，没有重跑抹去失败。源码定位时一次字面通配路径和不存在的平台文件名读取失败，后按真实路径读取；报告首次落盘脚本也因字符串界定错误在执行前失败，未写任何文件。均为审查方准备错误，不计产品缺陷或有效检出。

直接回归：

- N6 的同一 phaseClock、截止、提前唤醒重查、stopwatch 仅观察均未回退。本机三行边界和真实 escaped-holder 通过；预算不变，无容差。
- forge-owner 夹具被拒绝动作；混合场景自有遗留仍实际清理。工具错误、补救失败、状态未知及保险矛盾仍使被测自检失败，最外层组合只因正确检出而 PASS。
- 本机 Windows 的动作仍局限本轮 Job；Unix 零信号依据源码及本轮诊断日志，不冒称本机 Unix 实跑。
- 未在本机运行 Linux/macOS，未构造原生取消/句柄关闭失败、不可中断睡眠或强制 PID 复用。API 接缝与受控任务状态测试不冒充这些原生故障。
- 作者 M45～M53、WSL/Windows 全量结果属于作者证据。本轮只独立做上述三个撤回，不冒称全部复做。
- 保留作者最初 M50 仅偶然检出、M53 未检出，2413634 才补确定性检查的历史；未改写成自始完整。

主要仓库外日志 SHA-256：

| 日志 | SHA-256 |
|---|---|
| fixed-run.log | `e28b0bc42bbfe6cfa5909a43458ac49a4dff78f4af08c43ac3a113d783115fb4` |
| old-run.log | `d18c5865cde9d1f9b04da9493b09f7fb38fb7f5e5ffb2940f8e357f86140bfa0` |
| fixed-exact.log | `91e087dfb1f704f48abc11a245770f0d1c4296f0e7c6ab464f63b70e8924c6c7` |
| old-exact.log | `24d5710e4d80ed75ca614ebc8b280a68fbb2c35a1336652e53058a6ef876869f` |
| postok-run.log | `fef7389048d6d5069810243c008d36b4267cddb542e34cd1d1d00b511ca4ebf4` |
| noack-run.log | `29e6a71f8c3cedb9cabb8c25887797935a5586d8d50514a07d96985cb81aa9ae` |
| nobytes-run.log | `25c4fbe09ae466781b945c83a4bdcf2d94f94381a68c5daf86c1c2a0d45509a3` |
| noread-run.log | `6858237ad6a6ae52df9341c14324f2994d04b4a04830d41a4ebb8f0494cf492b` |

用例、变异和日志未移入原仓库，不冒称已提交回归。

## 7．CI：正式状态与提交适用性

### 7.1 最终原型诊断

[40074633075](https://github.com/XKHoshizora/specmetry/actions/runs/40074633075)：**push、attempt 1、completed/failure**。三个完整日志实际 checkout 均为 2413634 本身。

| 平台 | Job | 机制场景 | 门禁 | 组合 | 正式结果 |
|---|---:|---:|---:|---:|---|
| Ubuntu | 120815263233 | 38 confirmed | 52/52 | 13/13 | success |
| Windows | 120815263197 | 29 confirmed | 48/48 | 13/13 | success |
| macOS | 120815263065 | 27 confirmed | **42/43** | 13/13 | **failure** |

跨平台唯一场景并集 **53**；不是把平台数相加。Results 重印去重；机制、门禁、内层 verdict 和组合分别计数。

Ubuntu 原型步骤设置 race=-race 并执行 go test -count=1 -v $race -timeout 15m ./...，没有竞态报告。其他平台不据此声称 race 通过。

StopRead、LingeringFailedStop、PhaseClock 和其他直接检查实跑；Windows StopAPIErrors 三行实跑。三个内层专用入口（InnerForeignPID、InnerLeak、InnerStopError）顶层按设计 SKIP，门禁中的子进程实际执行另核对，不把顶层跳过算通过。

新回归在 macOS 通过是各自的局部证据，不能把整次诊断改写为三平台成功。

### 7.2 最终常规 CI

[40076445381](https://github.com/XKHoshizora/specmetry/actions/runs/40076445381)：**pull_request、attempt 1、completed/success，8/8**。

| Job | ID | 结果 |
|---|---:|---|
| Web | 120820687270 | success |
| Go tests (ubuntu-latest) | 120820687412 | success |
| Go lint | 120820687419 | success |
| Go tests (macos-latest) | 120820687486 | success |
| Go tests (windows-latest) | 120820687540 | success |
| Build with dashboard (windows-latest) | 120820759422 | success |
| Build with dashboard (ubuntu-latest) | 120820759431 | success |
| Build with dashboard (macos-latest) | 120820759446 | success |

八个 Job 的实际 checkout 均为完整合并快照：

`122d3117eb2b82d65fd1894dfa28e4dfec30a1c8`

父提交分别为固定 base、固定 HEAD；合并树与目标树均为：

`5c547f4b842e301439ff81d8f1ae6cc87f49c675`

Ubuntu 普通 Test 的 -race 与单独 root 跨账户步骤分开：后者 sudo 实跑八个 CrossUser 用例；普通非 root 步骤相应跳过是按设计。真实旧 Git、特定 unsafe-Git/权限用例、默认关闭的 M14 诊断等跳过不计通过。

常规 CI **不运行独立原型模块**，8/8 不抵消原型 failure，不关闭维护事项。

### 7.3 历史分开保留

| 运行 | 事件/attempt | 对象 | 正式结果 |
|---|---|---|---|
| [40072151282](https://github.com/XKHoshizora/specmetry/actions/runs/40072151282) | push / 1 | e4eadfa | failure；设计内回归之外，macOS 另有普查 EINVAL，不全归为预期失败 |
| [40072621588](https://github.com/XKHoshizora/specmetry/actions/runs/40072621588) | push / 1 | 63b7432 | success，中间代码历史观测 |
| [40072845216](https://github.com/XKHoshizora/specmetry/actions/runs/40072845216) | push / 1 | ba32dbe | success，中间代码，尚无最后两项确定性检查 |
| [40072154190](https://github.com/XKHoshizora/specmetry/actions/runs/40072154190) | pull_request / 1 | e4eadfa | cancelled |
| [40072625624](https://github.com/XKHoshizora/specmetry/actions/runs/40072625624) | pull_request / 1 | 63b7432 | cancelled |
| [40072848680](https://github.com/XKHoshizora/specmetry/actions/runs/40072848680) | pull_request / 1 | ba32dbe | success，常规 CI |
| [40074634885](https://github.com/XKHoshizora/specmetry/actions/runs/40074634885) | pull_request / 1 | 2413634 | success，常规 CI，不是原型诊断成功 |

未触发/重跑 workflow，未用中间绿色替代最终失败。更早失败、取消、旧时钟 refuted 与作者首次实验记录继续保留。

## 8．最小后续与最终作答

| 类别 | 最小下一步 |
|---|---|
| 修代码，非阻塞闭环 | 若要关闭 N4-R，完成正面回答后的有界观察/失败协议，补真实 ok 后故障；保留合法反证 |
| 修文档 | 对 1989/2088 使用归属/原因未确认，区分被测普查与保险清理，不声称完整清理完成 |
| 补证据 | 明确最终 macOS 失败缺失的信息，不能用源码未改或中间绿色排除关联，不能以本轮技术 GO 替代 |
| 需负责人授权 | 是否另行插桩、有界验证或工程修复；本轮不批准重跑、重试或放宽 EINVAL 拒绝 |
| 产品选择 | 既有 14 项仍未批准。补工具证据/传播错误是工程工作；把未知归属/不可观察改为可忽略才涉及新的保证取舍，不能默认获批 |

不要求原型提前完成生产 Runner。冻结前仍需定稿 T1～T6 支持范围、C-4/D20 的资源及后续宿主策略、其他未完成必要原型和产品决定；获准实施后才做生产启动/收尾接线、配置/取消、事实/事务写入、输出及 E2E 验收。

最终分别回答：

1. **B1-Ra 可关闭**：原绿色晚到错误已变为 invalid，结果随任务移交、终态读取闭合。
2. **B1-Rb 可关闭**：确定性原生 ERROR_NOT_FOUND 与故障接缝均有效；释放失败未被写成释放成功。
3. **N4-R 不整体关闭**：旧组合及迟到记录已修；ok 后真实工具/报告失败仍可能 refuted，沿用非阻塞。
4. **无新 BLOCKER**：macOS 没有揭示新的绿色假通过证据，但最终诊断与清理证明缺口是真实的，不能豁免。
5. **修复技术 GO；当前不建议合并，合并条件未满足。** 常规 CI 成功、最终原型诊断失败均保留。
6. 本审查不宣布 T1～T6 全部通过、规范冻结、产品批准、生产授权或工作包/里程碑验收。

原仓库仅新增本未提交报告，不改被审内容，不提交、推送、合并、改基、安装升级、重跑 CI 或执行用户配置。维护事项保持原状。

**落盘后最终核验（2026-10-24 09:50:31 UTC）：** 远程仍为固定 HEAD/base，PR open、未合并、auto_merge=null；最终原型诊断仍为 attempt 1、failure，最终常规 CI 仍为 attempt 1、success。本地 HEAD 未变，只有本报告一个未跟踪文件；仓库相对链接目标均存在，未发现本轮 review083 测试进程残留。

