# Codex Review 082：PR #34 停止任务与 N4-R 的定向独立复审

## 1．结论

**PR #34：技术 NO-GO，当前不建议合并。最终常规 CI 已正式成功，技术修正条件仍未满足。**

| 问题 | 本轮裁定 |
|---|---|
| Review 081 B1：同步 Close 挡住阶段 J | 原始调度反例已修复；停止任务在 J 开始后异步启动，所有流共用一个截止 |
| B1 的完整停止、接手与错误契约 | **不能关闭**。挂起停止任务的晚到错误不进入终判；Windows 新增取消及复制句柄释放的错误被忽略，见 §3 |
| N4-R | 精确的“报告失败后延迟退出”已修复；**尚不能整体关闭**。迟到字节与同次报告失败可以共存，仍会误分为 refuted，见 §4。沿用 NON-BLOCKER |
| N6 | 保留 Review 081 在原时钟口径内的关闭结论；未发现直接回退 |
| Windows 重复取消 | 三种受控读取调度获得本机及三平台 CI 的相应支持；不是任意取消失败、任意驱动、任意调度的完成保证 |
| PR 合并条件 | CI 条件满足；技术条件未满足，暂不建议合并 |

本轮没有发现新的生产 Runner 缺陷，也不要求原型提前完成生产接入。阻塞点是**当前原型自己的停止错误与资源移交契约仍能产生绿色假通过**。这与已登记、尚待决定的 O3 无界资源及后续调用策略不同。

没有批准任何产品选择、保证调整或新的接受风险。14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14、M-15、M-16 保持未解决。

## 2．固定身份、范围与核验

| 对象 | 固定身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| 上轮对象 | `4e009a6b7fa7c81ca614b6d20789735175a8caaf` |
| 回归先行 | `bde00fc30cae12c0827a6d3125d04273d10af164` |
| 异步停止修复 | `1a6fec5a988f29332c871ef04ff7ac92d5aedc17` |
| N4-R 修复 | `079f9c72d57a0b182d970b0f8455a80d580e04c1` |
| 最终代码（补充遗留回归后） | `125c4f70114c312828f7f8fc74734761e14c343a` |
| 本轮 HEAD | `0f1f20b8227bcec097a34c42cca393aaf22254cb` |

开始于 **2026-10-24 06:18:05 UTC** 核对远程，技术核查结束于 **06:32:34 UTC** 再次核对：HEAD/base 均与固定对象一致，PR open、未合并、auto_merge=null。没有将新对象混入结论。开始及技术核查结束时本地 HEAD 均为本轮目标、工作区干净，报告此前不存在。落盘后的最终检查另记于文末。

已读取适用 AGENTS、完整 [Review 081](codex-review-081.md)、本轮完整增量、[原型文档](../wp2.5-prototype.md) §2.13、C-2/C-3/D20 的修订、[处理记录](../review-resolution.md) §97，以及停止、复制、场景接手、遗留登记、helper、报告和清理的必要调用链。没有重做已关闭的 PR #33 整包审查。

相对上轮为 **19 文件，+1217/−84**：

- 原型 14 文件：blocking_unix、dest、destscen、gate、helper、keep_darwin、keep_linux、keep_unix、keep_windows、main、pipe、stop_check、stop_unix、stop_windows。
- 文档 5 文件：AGENTS、implementation-plan、review-resolution、wp2.5-prototype、新增 Review 081。

提交顺序：9c612d6 原样归档、bde00fc 回归先行、1a6fec5 异步停止、079f9c7 N4-R、125c4f7 补充遗留门禁、6016bd4 结果文档、0f1f20b 状态同步。

**125c4f7 → HEAD 只有四份文档。** prototype 子树均为 `2bcde12efc6946a920cb6fa8b0f08bb60351cd9f`，prototype/.github 差异为空。生产代码、Schema、迁移、根模块依赖、ci.yml、诊断 workflow、check-runner 草案和既有冻结契约均未变。

Review 081 归档 SHA-256 为 `d1c1227c2b16ccd097db2c29cd128f40ad6811c3079cad1c25643866f0ca9280`，与 §97 登记相符。review-resolution 仅末尾追加 64 行；§84～§96 没有改写。§97 如实保留原受审身份、NO-GO、N6 的局部关闭和“修复待复审”，没有自行宣布 B1 关闭。

## 3．BLOCKER：B1 的错误与资源接手仍不完整

### 3.1 已经成立的修复

[runS6On](../../prototype/wp25/dest_test.go#L375) 现在先在 400～401 行建立 J 起点与唯一截止，再为各流启动停止任务。停止任务的句柄准备、关闭和取消都位于工作 goroutine 内，没有再在调用者中先同步 Close。

复制等待与停止等待均传同一个 deadline，不会逐流重新获得完整 T_join。J 等待结束后封闭复制入口；尚未返回的复制任务和停止任务分别进入 left。这里建立的是有限阶段等待及封闭协议，不是操作系统硬实时调度上界。

本机在真实 os.Pipe、RawConn 读引用和真实系统读取上重跑：

| 调度 | 回归先行树 bde00fc | 本轮目标 |
|---|---|---|
| 已进入 ReadFile | 通过 | O2、stop=done、left=0，通过 |
| 持有读引用，在 J 内才进入系统读取 | 5 s 内 J 未开始，失败 | J 正常开始；开闸后无须写入，O2、done，通过 |
| 持有读引用到 J 截止后 | 5 s 内 J 未开始，失败 | 在阶段时钟截止返回 O3、stop=pending、left=2；封闭有效；开闸后两任务结束，通过 |

最后一行验证了停止与复制分别登记，两种后续调用候选都看到挂起项；没有把返回 O3 当作两任务已经结束。TestStopRead 的默认预算和时钟推进没有增加容差。

**原始“同步 Close 在 J 前阻塞”的反例可以认为已修复；但下面两个直接关联缺口使 B1 的完整契约仍不能关闭。**

### 3.2 B1-Ra：停止晚到错误被遗留接口丢弃，能够 confirmed/0

**准确位置（均为本轮固定 HEAD）：**

- [dest_test.go:294](../../prototype/wp25/dest_test.go#L294)：停止工作函数将 closeRead 的错误保存到 stopTask.err。
- [304 行](../../prototype/wp25/dest_test.go#L304)：stopTask.until 只返回是否结束，不返回 err。
- [432 行](../../prototype/wp25/dest_test.go#L432)：只有运行返回前已结束的停止，才将 err 放入 stopErr。
- [439 行](../../prototype/wp25/dest_test.go#L439)：挂起项交成只有 what/until 的 leftover，失去错误读取接口。
- [main_test.go:213](../../prototype/wp25/main_test.go#L213)：adopt 仅检查当时的 stopErr。
- [293 行](../../prototype/wp25/main_test.go#L293)：场景结束时仅检查 leftover.until 是否为 true。
- [destscen_test.go:39](../../prototype/wp25/destscen_test.go#L39)：lingering.pending 同样会直接丢掉已结束项，不检查其错误。

**预期与实际：** 文档 §2.13 明确“停止出错是观察失败”；adopt 的注释也作同样约定。但停止若在 runS6 返回以后才失败，场景只知道它已结束，最终可以 confirmed。登记持有任务并不等于登记其最终结果。

**独立精确验证：** 在仓库外目标副本新增一个受控 io.Closer；它等待本轮 release，释放自己的 gateReader，然后真实返回固定 sentinel error。没有改 judge、verdict、adopt、done 或 runS6 的判定。相同场景仅改变 release 时刻：

1. 提前 release：错误在 runS6 返回前可见，内层 invalid、exit 1；外层要求 invalid 的检查 PASS。
2. 在场景 defer 回收阶段才 release：J 已返回，停止及复制已移交；随后二者都结束，Closer 返回**同一错误**。内层却 **confirmed、exit 0**；外层要求 invalid 的检查 FAIL。
3. 同一二进制的正常真实管道对照 confirmed，排除全局门禁失效或一律拒绝。

关键结果：

~~~text
review082-early: want invalid, got invalid,   inner_exit=1, pass=true
review082-late:  want invalid, got confirmed, inner_exit=0, pass=false
reason(early): observation failed: stopping the read of out: ...error
~~~

这项是受控停止错误注入，不冒称真实硬件关闭失败；它直接穿过当前真实 S6、接手、清理和外层子进程终判。源码同时证明这个结果不依赖 Windows：错误信息在共享 leftover 类型中已经丢失。

**实际影响：** 工具已经观察到停止失败，却能输出绿色机制证据；lingering 也可能将“带错误的结束”当作无事发生。它违反的是当前原型的有效性与移交契约，不是要求提前实现长驻生产宿主。

**最小修正：** 保留遗留任务的终态和错误，例如使等待返回 ended/error，或在任务对象上提供同步后的结果；adopt/done 与 lingering 都必须消费并保留晚到错误，不能因结束而清除。增加“同一停止错误在 J 前、J 后出现”的真实终判对照。O3 以后允许还是拒绝后续调用仍属 D20，不由这个工程修正决定。

### 3.3 B1-Rb：Windows 取消和复制句柄释放错误被静默丢弃

**位置：** [stop_windows_test.go:62](../../prototype/wp25/stop_windows_test.go#L62)、[65 行](../../prototype/wp25/stop_windows_test.go#L65)。

当前代码：

~~~go
case err := <-closed:
    _ = windows.CloseHandle(own)
    return err
case <-tick.C:
    _ = windows.CancelIoEx(own, nil)
~~~

DuplicateHandle 的失败有返回路径，但重复取消的所有错误、释放自有复制句柄的所有错误均被忽略。即使后者没有释放句柄，closeRead 仍可能返回 nil，stop 被记为 done；当前状态无法区分“原 File.Close 已返回”与“新增的取消资源已经正确释放”。

取消找不到当前请求的 ERROR_NOT_FOUND 可以是正常竞争；不能把其他失败一概按这个正常情形忽略。[CancelIoEx 官方说明](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelioex) 明确区分这些返回情况，成功也只表示请求取消，不表示操作已经结束。[CloseHandle](https://learn.microsoft.com/en-us/windows/win32/api/handleapi/nf-handleapi-closehandle) 同样有失败返回。这里不要求把每次没有请求可取消判为工具故障。

**独立有界注入：** 在另一份目标副本，只给这两个原始调用加可替换函数接缝，保持 closeRead 控制流不变；测试用自己的真实 os.Pipe/RawConn 构造 Close 等读引用：

- CancelIoEx 接缝实际命中后返回 ERROR_ACCESS_DENIED；随后测试仅向自己的管道写入以释放读取。closeRead 返回 **nil**。
- CloseHandle 接缝实际命中，返回 ERROR_ACCESS_DENIED 且不关闭其参数；closeRead 仍返回 **nil**。测试随后用捕获到的自有句柄调用真实 CloseHandle 收尾，复制任务也结束。
- 两条“错误不得丢失”的独立断言均失败；正常真实管道对照通过。

这些是 API 故障注入，不是本机原生取消/句柄关闭故障的动态复现。它们与精确源码分支一起证明错误目前没有传播。没有对其他进程、句柄或未知对象动作。

**最小修正：** 对正常无请求竞争与实际 API 故障分类；实际失败持续影响停止结果，且在保守保留/释放自己资源后进入不可豁免的 invalid。原 File.Close 返回后仍须检查自有句柄释放结果；挂起期间遇到错误，不可只等以后某次成功来清除。修正须与 §3.2 的晚到结果接口一起验证，避免“底层开始返回错误，但接手者仍丢弃”。

这不是否定重复取消方向本身；是当前实现的失败传播和资源完成证据不足。§2.13 的“停止出错是观察失败”、C-2/C-3 的资源说明须忠实覆盖这些失败结果。

## 4．NON-BLOCKER：N4-R 的字节正证据仍不足以排除工具失败

### 4.1 已修复的精确路径与健康对照

本机及本轮诊断均确认：

| 场景 | 本轮结果 |
|---|---|
| control.f1-skip-drain | confirmed，内层 0 |
| 两路读端保留、工具正常，可靠看到迟到字节 | refuted，内层非零 |
| epipe 报告写失败后立即退出 | invalid |
| epipe 报告写失败后再活 5 s | invalid |

旧 bde00fc 的最后一项实际为 refuted；本轮已不再用 alive 直接证明观察正常。等待结束后的管道计数与停止时计数作差，也避免仅拿旧缓冲中的任意字节当正证据。副本中的两个管道分别来自本轮流；主 helper 已退出，保持者身份经原协议确认。

### 4.2 新的组合反例：stdout 字节成功与 stderr/报告失败同时发生

**位置：** [pipe_test.go:181](../../prototype/wp25/pipe_test.go#L181)、[195 行](../../prototype/wp25/pipe_test.go#L195)、[214 行](../../prototype/wp25/pipe_test.go#L214)，[helper_test.go:97](../../prototype/wp25/helper_test.go#L97) 的顺序双流写入及 [报告函数](../../prototype/wp25/helper_test.go#L197)。

当前 no-record 分支以“所有保留读端的总字节增量 > 0”作为工具正常、预测不成立的依据。然而某次 stdout 写入成功，不排除紧随其后的 stderr 写入失败，也不排除报告失败。

**仓库外独立实验保持真实写入及报告路径：**

1. 原 F1 helper、父子建立/归属/退出、S6 与终判都保持。
2. 只保留 stdout 的额外读端；stderr 按正常 S6 停止。
3. holder 在第六轮写入前可控等待 1 s，使该轮真正晚于停止计数；原本的“先 stdout、再 stderr”顺序不变。
4. stdout 实际写入成功；stderr 实际产生管道写错误。epipe 报告按既有注入方式以只读文件打开，WriteString 真实失败；helper 的 after-report-failure 注入点命中，并在失败后保持存活。
5. 原场景取得 **11 个停止后的新字节**、0 份 epipe 报告，仍给出 **refuted、exit 1**。外层要求工具故障 invalid 的检查 FAIL；两个自有进程最终被正常清理。

追加的观察日志保留了原内层完整输出，并未改变其判定：

~~~text
no epipe record; 11 late bytes reached the kept read ends after the stop
holder's late writes fail: 0 epipe reports
verdict=refuted; inner_exit=1; inject_hits=4; final=ended; pass=false
cleanup: verified: 2 processes gone
~~~

因此 §2.13 表中“报告失败 → invalid”和“工具失败不会降为 refuted”的无条件陈述仍超出实现。源码中对计数 API 错误会 invalid，这是正确的；问题是**成功的局部字节观测不能证明整个报告链没有失败**。

**严重度：继续 NON-BLOCKER。** 该路径仍为 refuted/非零，没有形成 confirmed 或绿色假通过；不因本轮未全部关闭就单独升级。N4 不能整体关闭，但本轮 NO-GO 的阻塞依据是 §3。

**最小建议：** 把持有者报告失败与“预测的写入错误”分开传达，或补足能证明这次工具链健康的完成/确认协议。保留合法 refuted；不能用所有缺记录一律 invalid 来取代分类。回归至少覆盖“两路均保留且无故障”“单路正字节 + 另一流失败 + 报告失败”。文档在修正之前应限定现有正证据范围，不能仅在边界段说只测一种构造，同时保留表中的无条件工具健康结论。

## 5．停止、封闭、资源与 Windows 候选的可信范围

| 事件/状态 | 当前支持与限制 |
|---|---|
| J 开始 | 先取起点/截止，再发起异步停止；句柄准备也在停止任务内 |
| 停止已请求 | startStop 已启动；不等于 Close 已返回、取消成功或资源已释放 |
| 停止完成 | 当前 done 表示 closeRead 返回；其错误处理尚有 §3 缺口 |
| 复制完成 | 与停止完成分开；O2 与 O3 没有合并 |
| 入口封闭 | 受控目的端封闭后旧入口不能再写出；TestStopRead 三行均实际核查 |
| left 移交 | 能移交复制器和停止任务；健康挂起对照两项都在，晚到错误尚未保留 |
| 场景接手 | 先释放场景自己的回调/收尾自有进程，再限时等 left；超期会 invalid |
| 后续宿主调用 | lingering 只展示两种候选；C-4/D20 仍未批准，不是生产策略 |
| 最终释放 | 健康释放对照通过；不能由“J 已结束/O3 已返回”推导；API/晚到错误需修正 |

Windows 自有取消句柄的正面依据成立：SyscallConn.Control 持有文件引用时 DuplicateHandle，bInheritHandle=false、同进程及相同访问权；原 File.Close 开始之前取得副本。取消循环只操作该副本，不再对已关闭、可能复用的原始数值句柄动作。[DuplicateHandle 官方说明](https://learn.microsoft.com/en-us/windows/win32/api/handleapi/nf-handleapi-duplicatehandle) 支持副本引用同一对象及不可继承参数的含义。

本机窗口对照在写端不写入的情况下开闸，读取和停止自行结束，支持“经副本取消原句柄对应同步管道读取”的这项候选；三平台 TestStopRead 的 Windows 行亦实跑。作者仓库外探针仍单独属于作者证据。

循环在 File.Close 返回时停止并释放 ticker/复制句柄；若读取始终持有引用而不进入系统调用，循环和相关资源会继续存在。J 可以返回并封闭，但没有据此证明循环已经结束，更没有最坏完成时间保证。每毫秒重试不是硬实时保证；底层支持、错误处理和实际读取调度都是前提。[微软 I/O 取消说明](https://learn.microsoft.com/en-us/windows/win32/fileio/canceling-pending-i-o-operations) 也保留底层驱动支持及取消竞态的限制。

C-2/C-3 及 §97 已正确撤回旧“停止之前等过所以读取必然已发出”的推断，并区分挂起停止与复制资源。其“任何读取状态”“一进入系统调用即完成”等概括仍应限定于正常 API 成功及支持布局，不得掩盖 §3 的失败分支。

## 6．独立验证、直接回归及边界

实际平台：**Windows 11 10.0.26300、windows/amd64、Go 1.27.1**；Git 2.56.0.windows.3。固定对象通过 git archive 解到仓库外独立副本。使用已有 Go、GOTOOLCHAIN=local、GOPROXY=off，没有安装或升级工具。编译为独立测试二进制；每组测试进程有 15～240 s 的硬期限，内层沿用原型的子进程协议。

| 独立运行 | 结果与范围 |
|---|---|
| bde00fc：TestStopRead 三行、报告失败后延迟退出、正常管道 | 对照通过；两个窗口各 5 s 后失败；报告场景错误 refuted，整体非零 |
| HEAD：TestStopRead、TestPhaseClock、三个真实管道场景、全部 Windows TestGate/Combos 及直接表检查 | PASS；3 顶层机制场景 confirmed，37 项门禁 PASS，13 项组合 PASS；不是本机重跑全部 29 场景 |
| 停止错误早到/晚到 | 早到 invalid；晚到 confirmed/0，独立外层检查 FAIL |
| Windows API 故障接缝 | 两个 API 错误都未传播，两个断言 FAIL；真实管道健康对照通过 |
| N4 混合真实写入/报告失败 | refuted/1，外层要求 invalid 的检查 FAIL；健康、合法 refuted、精确旧反例三个对照通过 |
| N4 补充输出观察 | 同一失败保留；11 个迟到字节、报告失败注入命中、内层 refuted、最终清理 ended |

首次并行启动测试二进制时，PowerShell 对未完整引号保护的测试参数进行了拆分，报 flag provided but not defined，四组均 exit 2，**没有执行测试**；同时存在 shell profile 的工具定位告警。这些是审查方启动问题。之后关闭 profile 并逐个完整引用参数，才进行上表实际运行。无效启动日志与后续真实失败均保留；没有把准备失败计为产品检出，也没有用重跑绿色覆盖失败。

### 6.1 遗留门禁的独立检查

本机实际执行 cleanup.leftover-running：内层的闸门不放开，复制任务一直存在到内层进程退出；场景等待 5 s 后为 **invalid**，reason 明确为 cleanup not verified: still running: the copier of out，内层非零，外层 PASS。它不是仅因某条预测不成立而 refuted，也不是仅看作者变异结论。该用例是在修复后补入；作者将它回放到 bde00fc 的结果属于作者证据，本轮没有独立重做那次后补移植。

正常 window-held-past-J 在测试自己释放闸门后，登记中的停止与复制任务都结束；本机没有留下本轮 review082 测试进程。API 故障实验中未关闭的自有副本由夹具真实关闭，复制任务均等到结束。没有向未知 PID 或未知句柄动作。故意不释放的 gate-only 内层是独立测试进程，其结束回收该进程资源，不能称为该内层 goroutine 正常完成。

### 6.2 既有保护与 N6

- forge-owner/forge-owner-mixed/stale 组合实跑。非内层夹具被列入 refused，未进入 acted；归属不明维持不可豁免错误。混合组合中真实自有遗留实际进入 acted 并结束，不能靠关掉全部补救通过。
- 本机是 Windows，未独立执行 Unix 发信号路径；Unix 零信号依据本轮诊断日志及未变归属代码，不能冒称本机 Unix 动态复现。
- 补救失败、观察失败、结束未知及保险清理矛盾的既有组合仍使被测自检失败；最外层组合因正确检出而 PASS。这 13 个组合没有计成 13 个机制 confirmed。
- N6 的 phaseClock、截止、剩余预算和三项确定性边界判定未被本轮改回 stopwatch；新增 awaitOn 会在提前唤醒后重新查同一时钟。TestPhaseClock 三行及真实 escaped-holder 对照通过。预算仍为 T_drain=1 s、T_join=500 ms，没有容差。未发现原时钟问题直接回退。
- 未在本机运行 Linux/macOS，也没有将 CI 或作者 WSL 结果写成本轮本机复现。本机没有运行原型 -race；原型 Linux -race 的证据来自下节真实诊断。
- 没有构造原生句柄复制失败、内核取消失败、CloseHandle 失败、不可中断睡眠或强制 PID 复用；相关结论分别标为故障注入或源码证据。

隔离日志保留在本轮仓库外 review082 目录。便于追溯的 SHA-256：

| 日志 | SHA-256 |
|---|---|
| fixed-run2.log | `e6fa7e5006853ab303d4f96fbc66ca4af6bafb96681fafd357d072393ff74249` |
| old-run2.log | `f94019915fd237511069b401834e13be5095cee623dec42ea0811b2bbff2484d2` |
| late-run2.log | `7364dcaaea58bdf19d5be3242f065cd9be8fb94d31ac38d424099381ef8f473f` |
| api-run.log | `f857d755f467312919122296ad98636e9d1af9a16f008acf37d1886e3ae63f2e` |
| bytes-run2.log | `84e40d63134b9c0e99b660e1e5a0bfa7902aa16f2462dcfa7efb98585f3ad001` |
| bytes-details.log | `5e2bed1739a97f7555af8f70ff41af8ebe464c4b506e2cc0e5c64dc0bd0f45a5` |

这些日志未移入原仓库，也不把独立补充用例冒充为已提交回归。

## 7．CI、提交适用性及历史

### 7.1 最终原型诊断

[run 40066910980](https://github.com/XKHoshizora/specmetry/actions/runs/40066910980)：**push、attempt 1、completed/success**。逐个读取三个 Job 完整日志，实际 checkout 均为代码提交 125c4f7 本身。

| 平台 | Job | 唯一机制场景 | 门禁回归 | 组合 |
|---|---:|---:|---:|---:|
| Ubuntu | 120791990130 | 38 confirmed | 41 PASS | 13 PASS |
| macOS | 120791990342 | 27 confirmed | 32 PASS | 13 PASS |
| Windows | 120791990339 | 29 confirmed | 37 PASS | 13 PASS |

按场景 ID 去重后的跨平台并集为 **53**；38+27+29 是平台观察次数，不能相加成场景总数。Results 步骤重印的行已去重。门禁 PASS、内层 invalid/refuted、最外层组合 PASS 分开统计。

三个 Job 中 TestStopRead 三行、TestPhaseClock 三行、TestCandidates、TestGateJudge、TestWitnessJudge、TestRecordAlive、TestStartPoints 均实际通过。cleanup.leftover-running 的内层 invalid、F1 四种分类也有实际结果行。

Ubuntu 的 Prototype 步骤按 RUNNER_OS=Linux 展开 race=-race，执行 go test -count=1 -v $race -timeout 15m ./...；没有竞态报告。macOS/Windows 未使用该 race 参数。顶层 SKIP 只有 TestInnerForeignPID、TestInnerLeak 两个专供内层使用的入口；对应门禁/组合的子进程实际执行不能计作跳过，也不将顶层跳过计为通过。

本次诊断真实成功，但**未包含 §3/§4 的独立新增反例**，不能反过来证明这些失败路径已覆盖。

### 7.2 最终常规 CI

[run 40067821634](https://github.com/XKHoshizora/specmetry/actions/runs/40067821634)：**pull_request、attempt 1、completed/success，8/8**。

| Job | ID | 最终结果 |
|---|---:|---|
| Go lint | 120794878732 | success |
| Go tests (macos-latest) | 120794878743 | success |
| Web | 120794878800 | success |
| Go tests (ubuntu-latest) | 120794878888 | success |
| Go tests (windows-latest) | 120794878945 | success |
| Build with dashboard (ubuntu-latest) | 120794950218 | success |
| Build with dashboard (macos-latest) | 120794950226 | success |
| Build with dashboard (windows-latest) | 120794950228 | success |

八个 Job 的真实 checkout 均为 `ec1ede45a9f8e7128d3fc3f051e3342b14d73687`。独立从 Git 对象核对：

- 第一父提交：固定 base。
- 第二父提交：固定 HEAD。
- 合并树及目标树：均为 `5b385f6d1543faf5a9aec77b973532259004e668`。

不是仅凭作者回执认定适用。常规 CI 不进入独立 prototype/wp25 模块；不能替代原型诊断或本轮反例。

Ubuntu 普通 Test 步骤使用 -race；另一个 root 跨账户步骤单独编译并 sudo 执行八个 CrossUser 用例，均 PASS。后者不是普通 race 步骤：普通步骤因非 root 跳过它们是按设计行为。macOS/Windows 的 root 步骤按平台跳过。

常规测试还存在有意跳过：真实旧 Git、受特定环境变量控制的 unsafe-Git/自动 fetch 用例、默认关闭的 M14 诊断；Windows 另跳过不可读权限语义的少数用例。未把它们计为通过，也不因整仓 CI 成功重新验收未变模块。

### 7.3 首次失败、中间成功与取消

| 运行 | 身份、事件、attempt | 本轮核验 |
|---|---|---|
| [40066182430](https://github.com/XKHoshizora/specmetry/actions/runs/40066182430) | bde00fc，push，1 | completed/failure；三个 Job 日志均有两个窗口用例失败、epipe-unwritable-alive 错误 refuted；对照通过 |
| [40066611501](https://github.com/XKHoshizora/specmetry/actions/runs/40066611501) | 079f9c7，push，1 | completed/success；早于补充遗留回归，不能混成最终代码统计 |
| [40066184822](https://github.com/XKHoshizora/specmetry/actions/runs/40066184822) | bde00fc，pull_request，1 | completed/cancelled |
| [40066614489](https://github.com/XKHoshizora/specmetry/actions/runs/40066614489) | 079f9c7，pull_request，1 | completed/cancelled |
| [40066914330](https://github.com/XKHoshizora/specmetry/actions/runs/40066914330) | 125c4f7，pull_request，1 | completed/cancelled |

保留作者的第一版对照错误：曾要求“J 请求唤醒”，但复制器已先返回时根本不需唤醒；提交前改为阶段时钟不推进即返回。它属于无效早期夹具，不计为检出生产问题。

cleanup.leftover-running 的回归是修复后补的，不改写成全部测试先行。作者 M39～M44 的定向撤回、Windows/WSL 全量运行属于作者证据；本轮只独立执行本报告列出的反例和正对照，未冒称复做全部变异或 WSL 实验。

没有触发或重跑 workflow。既有更早失败、取消及本机 N6 旧 refuted 继续保留；当前绿色不能用来关闭 M-14/M-15/M-16 或证明其根因消失。

## 8．文档、产品待决与最小后续

**记录同步：** §97 的审查身份和“待复审”准确；原报告未改写。C-2/C-3/D20 已加入挂起停止、J 共享预算和资源无上界的限定，方向正确。但 §2.13 的“停止出错是观察失败”“工具失败不会降为 refuted”尚未被完整实现，需按 §3/§4 修正。

check-runner 草案头部仍有 PR #33 未合并的过时状态、旧标题仍有 not for merging；沿用 Review 081 的非阻塞事实收尾意见。这些不推翻已核验的合并历史，也不把准备文档合并等同规范冻结。

**本 PR 当前必要技术修正：**

1. B1-Ra：使挂起停止的最终错误随任务一同移交，并持续影响场景和登记终态，补早到/晚到错误对照。
2. B1-Rb：区分正常 CancelIoEx 无请求竞争与实际错误；检查自有复制句柄释放，确保错误不丢失、不假报完整结束。补真实管道上的 API 故障接缝及正常对照。
3. 同步现有证据概括，保留已成功的窗口正对照及本轮残余，不能只改成未来生产问题。
4. N4-R 可继续独立按非阻塞跟踪；若要宣布整体关闭，必须补足工具健康证据及本轮混合故障回归。

**冻结前仍有的门槛：** T1/T6 的受控目的端支持、取消/停止失败协议、挂起停止及复制任务资源边界、C-4/D20 对当前批次与同宿主后续调用的策略；其他 T1～T6 的已登记支持范围和必要原型继续保持。14 项产品选择（D1～D9、D11、D13、D18、D19、D20）仍待负责人决定，本轮没有代批、缩小或增加豁免。

**获准实施后、生产验收前：** 生产启动/收尾接线、§8.6 持久化组合、取消、配置固定、空批次、CLI/Schema/事务及真实进程 E2E 等仍归后续生产工作。当前缺陷修正不要求提前完成这些模块。

## 9．最终回答

1. **B1 不能关闭。** 原 J 前同步阻塞已修；停止晚到错误和 Windows API 失败传播仍不符合当前原型契约。
2. **N4-R 不能整体关闭，保持非阻塞。** 精确旧反例已修；新的真实双流/报告组合仍错误 refuted。
3. **N6 没有发现直接回退，维持原口径的关闭。**
4. **Windows 重复取消仅获得受控同步管道布局、成功 API 与可释放调度窗口的支持。** 不证明异常取消一定成功，也不证明资源已结束或任意调度有完成上界。
5. **PR #34 技术 NO-GO，暂不建议合并；最终 CI 条件已满足。**
6. 当前阻塞属于原型自己的有效性与资源结果契约；无界挂起、后续宿主策略及其他未决产品项仍另列冻结前门槛，不混为已接受风险。

本次仅新增本未提交报告。不宣布规范冻结、产品批准、生产授权、WP2.5/M2 验收；M-14/M-15/M-16 保持未解决。

**落盘后最终核验（2026-10-24 06:37:11 UTC）：** 远程仍为固定 HEAD/base，PR open、未合并、auto_merge=null；本地 HEAD 未变，仅有本报告一个未跟踪文件，其他文件没有变更。报告的仓库相对链接目标均存在，未发现本轮 review082 测试进程残留。没有提交、推送、合并、改基或重跑 CI。

