# Codex Review 102：PR #41 定向独立复审

## 1．结论与适用范围

**Review 101 的 B1、B2、B3 在本次受审的原型调用链范围内可以分别关闭；修复本身技术 GO。当前固定 PR 暂不建议合并。**

两个判断依据不同：三项修复有可追踪的错误保存、所有权及期限实现，且新增九项门禁和相关直接检查在固定代码点的两种 macOS 架构上均通过；但是固定代码点的原型诊断正式为 **failure**，arm64 的 `p3.k-nostatus-once` 没有执行目标观察、没有命中接缝。同一内层运行的 `P3.layout.1` 为 invalid，其具体错误已在外层输出中丢失，现有公开日志与 artifact 无法补回。常规 CI 的 8/8 不能替代该项证据。

| 问题 | 本轮裁定 |
|---|---|
| B1：计划故障覆盖真实枚举错误 | **关闭**：真实错误先进入场景失效状态，再构造候选输入；恢复与补救不清除它 |
| B2：关闭错误及所有权 | **关闭**：受影响的文件、观察器由一次性关闭器持有，错误进入最终结果；不把进程结束当作句柄释放 |
| B3：内部重试与期限 | **关闭**：枚举不再依赖 x/sys 内部无界循环；有限次数、绝对截止及 EINTR 剩余预算进入实际调用链。不是单次系统调用的硬实时保证 |
| 新的机制 BLOCKER | **未发现有具体反例支持的新 BLOCKER** |
| N-102-1：多场景失败理由遗漏 | **NON-BLOCKER 的诊断可追溯性缺陷**，本次确实丢失布局失效详情；不产生已发现的绿色假通过，但妨碍解释当前失败及补证 |
| 当前合并条件 | **未满足**。原型诊断失败保留；先补最小观测或找回现有原始证据，再取得与后续固定对象相符的有效验证。不是要求永久追溯历史未知，也不是要求反复运行直到偶发错误消失 |
| G5 | 获得同一代码点的两架构布局、数值及错误门禁的局部支持；**不认定整体满足** |

本报告不选择生产入口 E1/E2，不调整支持承诺，不接受未知风险，不冻结规范、不授权生产实施、不宣布 WP2.5/M2 验收。Q-3(a) 与其他已批准决定不重开。

## 2．固定身份、开始核验与范围

开始核验：**2026-10-29 16:11:37 JST（07:11:37 UTC）**。

| 对象 | 核验值 |
|---|---|
| 仓库 / PR | `XKHoshizora/specmetry` / [#41](https://github.com/XKHoshizora/specmetry/pull/41) |
| HEAD | `fbcef6fd230cedd60bc4943a6192e3eb1a875698` |
| base / master | `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b` |
| HEAD 树 | `651e8e5a70ca66008f607f37acbb90cd51cccace` |
| 修复代码点 | `eb2317fa0be728202ceb705a5bf5aa72eb6ab012` |
| 回归先行 | `7f9fda0871511f0586d6377b91f70006129f740c` |
| 上轮对象 | `c776f6a5f0e9a1fd09d4a1959e18122fe99b8307` |

GitHub PR、master 引用与本地对象相符；PR open、未合并、`auto_merge=null`。本地分支为 `test/wp2.5-prototype-p3`，开始时工作区、跟踪文件及暂存区均无改动。读取了适用的根 `AGENTS.md`、完整 Review 101、全部增量、原型文档 §15.11、处理记录 §125，以及受影响的共享调用链。没有另取更新对象。

相对上轮共 **20 文件**：六份记录/文档（含新增 Review 101）与十四份原型文件。按提交核验：

| 提交 | 实际内容与次序 |
|---|---|
| `6ae979778775e3a64e19166967a814b458d1789c` | 只归档 Review 101 |
| `7f9fda0871511f0586d6377b91f70006129f740c` | 回归、接缝及必要的可观测接口先行；旧错误处理、旧枚举循环与旧 EINTR 行为仍在，没有先修正被测判定 |
| `eb2317fa0be728202ceb705a5bf5aa72eb6ab012` | 三项修复；改变七份原型文件 |
| `fbcef6fd230cedd60bc4943a6192e3eb1a875698` | 五份文档收尾：AGENTS、implementation-plan、review-resolution、wp2.5-check-runner、wp2.5-prototype |

代码点到 HEAD 的 `prototype/`、`.github/` 差异为空。整轮没有修改生产 `internal/`、Schema、迁移、依赖、根 `ci.yml` 或诊断 workflow。§125 仅追加在原处理记录末尾；旧内容未改写。Review 101 文件为 **26,685 字节**，SHA-256 为 `1c27916733ee2f4a5f8726e8e689b0989327855bdc37f5c7e7b6750d71c40cac`，与归档登记一致。

## 3．B1：真实观察错误先于候选故障保存

主要位置：`prototype/wp25/p3_darwin_test.go:945` 的 `confirmEnd`，同文件的 `enumFault`、`runP3`；`gate_test.go:409`～`:417`。

修复后的次序是：

1. 取得 `groupLive` 的真实返回 `m, err`；清除本次枚举接缝。
2. 构造 R；真实错误立即 `s.flag("observation failed: ...", err)`，保存错误文本，并将该次 R 的列表置空、错误置为真实错误。
3. 单独使用 `in := err` 作为候选输入。计划故障只改 `in`，另记 F 与命中，不改 `err`，不撤销已保存的工具错误。
4. 候选据 `in` 给出“未知，不确认结束”；场景可以继续清理、再成功观察。最终 `s.done()` 仍先按累计 invalid 判定。

因此“真实观察失败 + 计划拒绝 + 后续清理成功”的合法序列不再得到 confirmed。R 保存首读、错误读及末读，真实错误理由不只存在于调试 F 中。非计划的观察错误也走相同保存入口。W/K 改写在原调用成功后才执行；原调用错误先返回，不用计划改写覆盖真实 API 失败。

固定 CI 中的旧/新对照：

| 用例 | `7f9fda0` | `eb2317f`，两个 macOS 作业 |
|---|---|---|
| `p3.enum-error-once`：真实读取层的一次错误与计划故障重叠 | 内层 confirmed / 0，外层 FAIL；命中 1 | invalid / 1，真实读取错误及注入文本均保留，外层 PASS |
| `p3.enum-error-once-healthy`：同一次读取错误，无计划故障 | 已 invalid / 1 | 仍 invalid / 1；不因修复只照顾重叠分支而回退 |
| `control.p3-enum-unknown`：真实读取健康，只有计划故障 | confirmed / 0 | confirmed / 0；确认的是正确拒绝候选输入，不是确认读取失败已经发生 |
| `p3.enum-error-and-close-once` | confirmed / 0 | invalid / 1；枚举和关闭两条理由同时存在，命中 2 |

注入返回一次错误后，后续原生读取及清理恢复；这种恢复没有清除 invalid。这里证明的是**真实场景中的读取返回值接缝**及错误消费链，不是原生 sysctl 本身发生 EIO 的动态证据。原始 Review 101 的旧实现反例、当前源码与这组原生场景中的接缝结果一致。本轮没有重新构造新的 macOS 原生故障。

**B1 关闭范围**：受审 P-3 `confirmEnd` 及上述同源读取/候选输入分离；不外推为所有旧平台接缝都已保存全部并发错误。

## 4．B2：所有权、一次关闭与最终结果

主要位置：`main_test.go:238`、`:317`；`p3_darwin_test.go:154`、`:611`、`:1205`、`:1270`；`tree_darwin_test.go:143`；`tree_unix_test.go:236`；`p1_unix_test.go:46`。

### 4.1 共享关闭器与受影响调用者

`scen.closer` 在句柄成功取得后登记收尾，返回的函数以 `sync.Once` 执行唯一一次关闭并保存结果。即使调用者忽略返回值，关闭错误也通过 `s.flag` 与 `closeFail` 保留。重复调用读到同一结果，不对可能已复用的描述符再次关闭。

| 取得对象 / 退出路径 | 当前责任 |
|---|---|
| 工具 stdout 文件成功，stderr 创建失败 | stdout 已登记；失败分支立即关闭，登记的收尾不重复关闭；成功关闭有 `created before the failure, closed` 观察 |
| 两个文件成功，Start 失败 | 两个关闭器均已建立，启动返回后关闭父端副本；Start 错误仍使场景无效 |
| 正常工具启动与结束 | 文件所有权不依赖后来等待成功；子进程由 child/Cmd 登记，文件由关闭器登记 |
| 工具输出回读 | `readBack` 显式 Open、立即登记、ReadAll、关闭；Read 错误返回，Close 错误即使返回值被忽略也进入场景 |
| P-3 与通用 Unix 场景的观察器 | 取得后立即登记关闭器；显式关闭和最终收尾只能实际关闭一次 |
| P-1 回调提前结束/观察器接缝拒绝 | 已创建的观察器先登记；提前退出仍执行场景收尾，不再依赖回调走到末尾 |
| Kqueue 创建成功但注册失败 | 构造函数尚持所有权，尝试关闭；注册错误与关闭错误以 `errors.Join` 一起返回并使调用者 invalid，不把构造失败说成没有资源责任 |

`exitObserver.close` 现在返回真实 Close 错误；Linux 同一共享接口的调用者也已调整。关闭失败后不会再次盲目关闭，剩余状态明确未知；本轮不声称该句柄已释放。构造失败时的 joined error 也是失效证据，不能由场景的“无进程”清理字样证明文件或 kqueue 已释放。

### 4.2 最终结果与门禁

`s.done()` 先执行登记的收尾，再作进程清理、任务结果消费、报告终检，最后复制累计 invalid 并生成 **WP25RESULT**。`closeFail` 非空时 Cleanup 加 `NOT verified: closing failed (...)`。`runP3` 将接缝重置移到 `s.done()` 之后，因此真正发生在收尾中的观察器关闭也能命中。提前输出的 **WP25P.T** 仍只是此前快照，不能代替最终 WP25RESULT；当前 gate 使用后者，没有用提前的空 T 恢复成功。

新门禁的 `cleanup`、`reasons`、`hits`、`sees` 是用例自己的声明：正常仍默认清理 `verified`；两项关闭失败用例才允许 `NOT verified: closing failed`；部分创建失败才允许 `no processes`。同时要求指定理由、精确命中数及声明的错误数量。双错用例必须同时含枚举错误与关闭错误，不能以另一条错误凑数通过。外层工具错误、归属、补救、Final 与保险清理仍由原 judge 独立检查，没有因允许目标的关闭失败而豁免外层失败。

两架构新门禁均证实：输出文件关闭失败、收尾关闭 kqueue 失败得到 invalid / 1；双错都保留；部分创建失败在不启动工具的情况下关闭已取得文件。旧版两项关闭失败为 confirmed / 0；部分创建用例旧版已 invalid，但缺少已创建文件关闭的观察，**不能把后者写成旧版 confirmed 假通过**。

### 4.3 本轮独立检查

在原样 Windows 副本中直接调用提交中的 `closer` 和 `s.done`，使用本轮自有文件：实际关闭成功后以接缝返回关闭错误，不伪造 verdict。显式路径还由 16 个调用共同争用关闭器，断言底层关闭仅一次。

| 对照 | 实际最终结果 |
|---|---|
| 正常关闭、后续健康操作 | confirmed，包退出 0，关闭次数 1 |
| 显式关闭报错、忽略其返回、后续健康操作 | invalid，Cleanup 为 NOT verified，包退出 1，关闭次数 1 |
| 仅由场景收尾关闭时报错 | 同上；证明晚到收尾错误进入最终结果 |
| 已有观察错误，再叠加关闭错误 | 两条错误均在 Invalid/Reason，invalid / 1；后续健康预测与进程清理不覆盖它们 |

这些是共享 Go 代码的独立动态证据，**不是 macOS Close 原生失败或资源真的泄漏的实验**。本轮四个对照与固定代码的两架构门禁共同支持 B2 在受审链内关闭。

## 5．B3：实际重试、剩余预算及边界

主要位置：`tree_darwin_test.go:71`～`:124`、`:367`；`enum_check_darwin_test.go:29`、`:98`、`:168`、`:262`；`p3_darwin_test.go:1205`；`census_darwin_test.go:100`；`witness_unix_test.go:175`。

### 5.1 消除库内部无界循环

核对固定依赖 **x/sys v0.49.0** 本地源码：`unix/syscall_darwin.go:520` 的 `SysctlKinfoProcSlice` 确有内部 `for`，ENOMEM 后继续；`unix/syscall_bsd.go:471` 的 `SysctlRaw` 是一次取长度、一次取数据，没有同类内部重试。新实现改用后者，再由 `kinfoProcs` 控制：

- 首次读取 + 最多三次重试，即最多四次调用；只有 ENOMEM 可重试。
- 每次决定重试前，检查同一个调用者绝对截止；错误、次数用尽或到期返回错误，不返回伪空列表。
- `kinfoOf` 只接受完整条目长度，复制到正确对齐的结构数组；合法空返回仍可为空集合。
- group、`procList`、census 的实际调用链都进入此函数。新的截止参数已传到 P-1、组确认、emain、排他区间收尾及外层普查；Linux 接口适配没有为它引入新的无限循环。

**首次调用已到截止的行为**不是隐含的超时重试：源码与 §15.11 明确选择仍进行第一次读取，保留成功回答；第一次 ENOMEM 若已到截止则不重试。受审 §19.5 约束的是重试的次数与剩余时间，不承诺撤回一个已取得的可靠回答。一次调用完成较晚与再次获得完整预算不同。没有仅凭这一策略发现新的产品保证变更；不能将它改述成“绝不在截止后调用任何系统 API”。当前直接检查覆盖首次调用在截止后返回，不专门覆盖调用前已经过期；后者本轮按源码裁定，不冒称已动态覆盖。

### 5.2 EINTR、工具步骤及共享调用者

`kqWaitRaw` 在入口固定截止，每轮给 kevent 剩余时间；EINTR 到/过截止返回错误，之前才重试。n=0 是可靠的无事件回答，其他错误不是无事件。初始剩余为零时可以作零超时首次观察；不因一次 EINTR 重新给完整 d。

`toolRaw` 在创建文件、Start 之前固定整步截止，并与观察段截止取最小值；等待只获得 `time.Until(end)`，不再在 Start 返回后重新分配整步预算。超期工具仍由已登记 child 持有，停止、census 与最终收尾继续消费错误。外层保险清理也在第一次普查之前固定截止，之后普查共用它。

这些变化真实约束**用户态重试与传给等待的时间**；没有抢占同步文件操作、Start、单次 sysctl/Close 等系统调用的能力。观察段和清理段事后超限检查只是失效判据，不能冒充对这些单次调用的硬中断或操作系统硬实时保证。B3 关闭不扩大为全套件所有等待均有绝对上界。

### 5.3 回归是否检出原问题

`TestEnumBudget` 的 8 项、`TestEnumChain` 的 3 项与 `TestKqWaitDeadline` 的 4 项采用固定要求和可释放 fake。`bounded` 在调用前设置独立期限，到期先记失败，再让 fake 给终止回答，等待被测 goroutine 返回。旧实现不是被套件 15 分钟超时强杀才算检出。

回归先行两架构均有八个直接子项失败：枚举次数/截止/边界三项、三条枚举调用链、EINTR 不断/边界两项。到期后的 fake 释放使调用返回；没有“停止 fake 后仍不返回”的日志。修复后这些项均 PASS，并保留成功、空列表、畸形条目、其他错误、EINTR 后超时及事件对照。`control.p3-enum-enomem-retried` 的三次 ENOMEM 后真实第四次调用仍 confirmed，四次 ENOMEM 的场景则 invalid，未修成全部拒绝。

作者曾有“重新给全部等待时间”的变异未检出；最终直接检查增加了每次 timeout 到达点 `reach` 的约束，10ms 延迟对照与 5ms 测量余量能检出重新给足预算，剩余预算没有因此改变。作者临时副本变异与 Windows 抽取检查仅为作者证据；本轮没有独立原生重跑 Darwin 的这些 fake 测试，也没有将它们称为原生 ENOMEM/EINTR。

**B3 关闭范围**：本次枚举、kevent EINTR、工具整步预算及其新签名调用者。没有认可不可中断单次调用的最坏耗时，也没有把既有链外等待一并验收。

## 6．arm64 失败、N-102-1 与最小下一步

### 6.1 已知事实与未知原因

[修复诊断 40817305419](https://github.com/XKHoshizora/specmetry/actions/runs/40817305419) 的 arm64 作业 **123261288396** 中，唯一门禁失败发生于 04:26:28～04:26:29 UTC：

| 字段 | 日志实际值 |
|---|---|
| 用例 | `p3.k-nostatus-once` |
| 内层选择 | `TestP3$/^(layout.1|k.exit7.1)` |
| 目标结果 | invalid / 内层退出 1 |
| 目标理由 | `not executed: the layout premise does not hold: the probes of this process gave [invalid]` |
| 目标 launches / work / hits | **0 / 0 / 0** |
| 外层 witness | created=5、no_receipt=5、unowned/shortfall/left 均无缺额或存活项 |
| 外层工具错误 | `hit: the fault never reached its site` |
| 外层结果 | **pass=false**，Final=`ended`，remedy=`not-needed`，insurance=`nothing left` |

`layoutPremise`（`p3_darwin_test.go:306`）读取的是同一内层进程中布局最终 verdict。目标因该前提失败提前结束，尚未到 K1 接缝。五个创建记录是工具启动记录，**不是“目标已经启动”的证据**。`inner_cleanup=no processes` 是目标场景的清理，不是布局场景的文件/观察器清理详情。

现有完整作业日志没有该次内层 `P3.layout.1` 的具体 Reason/Invalid；run 的 artifacts API 返回 **0 个 artifact**。无法从现有证据确定是哪个工具、解析、观察或收尾步骤失效。镜像版本确有变化，但这是元数据，不是原因证明；本报告不推定由镜像、环境或本轮修复造成，也不排除其中任何一种可能。主运行、Intel 或其他内层的布局成功不能补写这一次。

外层普查/身份与保险结果支持其已知进程均结束；不能据此声称那次内层的每个文件或观察器均释放。没有证据支持新增“实际遗留了进程/句柄”的断言。

### 6.2 拒绝正确，追溯不足

**门禁正确失败关闭。** 目标理由不符、命中为零、最终 go test 非零均保留；未把“也是 invalid”当作预期 K1 故障已检出。因而没有本次失败被算作 confirmed/绿色的证据。

**N-102-1 位置**：`gate_test.go:530` 的 `runInner` 与 `:726` 的 `runGate`，特别是 `:766` 的 exact 分支。

`runInner` 捕获内层输出、解析全部 WP25RESULT，但仅返回结果 map，不输出失败内层的完整诊断。`runGate` 主要复制目标的 Reason；because 不匹配时已经令 `verdictOK=false`，随后 exact 的“其他场景不合格”分支不运行。即使运行，该分支也只列 ID/verdict，不保留具体原因。Results 步骤重印的是根层结果，无法补回被捕获的内层详情。

这是**具体的诊断信息损失**，降低错误追溯与有效性审查能力。当前路径仍强制 FAIL，故不把它机械升级为新的绿色假通过 BLOCKER；但当前诊断尚未完成要求的 arm64 K1 证据，合并条件仍不满足。B1～B3 的各目标回归均独立命中并通过，不因该缺失自动撤回三项关闭；也不以关闭三项推定剩余诊断成功。

### 6.3 最小观测与验证范围

优先只读查找当次已保存的内层原始结果；当前仓库日志/artifact 中没有。不要求恢复不存在的文件，或永远证明历史失败的根因。

若另行授权补观测，最小范围是：

1. 无论目标 because/命中是否满足，均保留本次选择的每个非 confirmed 内层结果的 **ID、verdict、Reason、Invalid、Cleanup**；同时保留目标及前提的 launches/work、owned/shortfall 与命中。使用最终 WP25RESULT，不用 WP25P 的提前 T。无需无差别回显原始工具 stdout/stderr。
2. 两个有界回归：前提 layout invalid → 目标未执行/零命中，外层仍 FAIL 且原因可追溯；layout 只在最终关闭时失效 → 保留最终关闭错误，不被提前快照或目标理由替代。健康多场景、双错及额外外层错误对照继续不变。它们验证观测，不放宽 judge。
3. 后续按负责人另行授权，在新的固定对象取得布局及目标均有效、接缝确实命中的 arm64 证据，并保持受影响调用链的验证。若出现真实工具错误仍必须失败；不是追加忽略、重试或“跑到绿”。是否需要扩大调查由具体新增错误决定。

上述观测改动和远程运行**本轮均未实施或授权**。不需要为此重做全部 T1～T6、生产 Runner 或全部无关模块。

## 7．CI 身份、失败历史与逐项计数

### 7.1 回归先行

[40816447439](https://github.com/XKHoshizora/specmetry/actions/runs/40816447439)：push、attempt 1、正式 completed/failure，四作业实际 checkout 均为 `7f9fda0871511f0586d6377b91f70006129f740c`。

| 平台 / job ID | 根层结果（去重） | 门禁 / 组合 | 结果 |
|---|---|---|---|
| arm64 /123258520722 | 96 confirmed = P-3 63 + 既有33 | 93/99；13/13 | failure |
| Intel /123258520785 | 同上 | 93/99；13/13 | failure |
| Ubuntu /123258520783 | 53 confirmed | 111/111；13/13 | success，带 -race |
| Windows /123258520481 | 53 confirmed | 92/92；13/13 | success |

两个 macOS 的六项门禁失败恰为：`p3.enum-error-once`、`p3.enum-enomem-exhausted`、`p3.close-out-once`、`p3.close-kq-once`、`p3.create-err-once`、`p3.enum-error-and-close-once`。不是九项全部失败；三个保护性对照原先已通过。直接检查的八个失败子项另计，不算八个机制场景。父端 fake 截止及后续释放可在日志中核对，没有等到套件总超时。

同提交的常规 [40816451875](https://github.com/XKHoshizora/specmetry/actions/runs/40816451875) 正式 cancelled：Windows tests cancelled，其余七作业 success。根 CI 已有并发取消规则与推送时间相符；不将被取消作业算通过，也不将它当修复证据。本轮没有手动取消或重跑。

### 7.2 修复代码的原型诊断

[40817305419](https://github.com/XKHoshizora/specmetry/actions/runs/40817305419)：push、attempt 1、正式 completed/**failure**；四份完整日志和 WP25META 均为 `eb2317fa0be728202ceb705a5bf5aa72eb6ab012`。

| 平台 / job ID | 根层结果（去重） | 门禁 | 组合 | 正式 job 结果 |
|---|---|---|---|---|
| arm64 /123261288396 | 96/96 confirmed | **98/99** | 13/13 | **failure** |
| Intel /123261288366 | 96/96 confirmed | 99/99 | 13/13 | success |
| Ubuntu /123261288385 | 53/53 confirmed | 111/111 | 13/13 | success |
| Windows /123261288091 | 53/53 confirmed | 92/92 | 13/13 | success |

两种 macOS 的九项新增门禁全 PASS；`TestP3Decode`、`TestP3Parse`、`TestEnumBudget`、`TestEnumChain`、`TestKqWaitDeadline` 五个直接检查均实际 PASS。arm64 失败仅为 §6 的既有 K1 门禁，不能将全 run 写成成功。

每个 macOS 的 P-3 为 **21 个场景族各三次，共63个运行 ID**；不是63种独立机制，更不能与另一个架构相加。既有机制场景的平台并集仍按53个 ID，门禁、组合、直接检查单独统计。日志 Results 步骤重复输出 WP25RESULT/WP25SUMMARY，已按 ID 去重；正常原生运行与计划机制故障分开：P-3 的63次中60次 normal、3次 fault-injection。

两架构根层 P-3 的每次结果均 confirmed、最终清理 verified；这些读数可采纳为本次运行的局部证据。它们不替代失败门禁的内层布局实例。arm64 为 Go1.27.3、macOS26.6.2/25G83、`xnu-12377.161.14~5`；Intel 为 Go1.27.3、macOS26.6.1/25G76、`xnu-12377.161.13~4`。SDK/实际内核与规范固定源码的证据层次不变。

Ubuntu 的 Prototype 步骤实际设置 `race=-race` 后执行 go test，不仅是没有竞态告警。三个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError` 在主运行设计 SKIP，相应内层由门禁选择，不把这些 SKIP 算通过。Darwin 专属直接测试不在 Linux/Windows 编译，不能计为这些平台的动态验证。

### 7.3 最终常规 CI

[40818809502](https://github.com/XKHoshizora/specmetry/actions/runs/40818809502)：pull_request、attempt 1、正式 completed/success，**8/8**。逐份完整日志核对实际 checkout 均为：

`46b67c027cc601095a8f7bf659685c1dd8888a49`。

Git commit API 核对父提交依次为固定 base、HEAD，树为 `651e8e5a70ca66008f607f37acbb90cd51cccace`，与 HEAD 相同。

| 作业 | ID |
|---|---|
| Go tests Windows | 123266156877 |
| Web | 123266157029 |
| Go tests macOS | 123266157055 |
| Go tests Ubuntu | 123266157058 |
| Go lint | 123266157083 |
| Build macOS | 123266258137 |
| Build Windows | 123266258142 |
| Build Ubuntu | 123266258173 |

Ubuntu 普通测试带 -race；root 跨账户步骤另行编译并运行，八项 CrossUser 实际 PASS，**不是 root race**。普通步骤中需要 root 的用例跳过，以及真实旧 Git、平台权限、默认关闭诊断等既有跳过按原适用范围保留。常规 CI 不进入原型独立模块，不能抵消原型 failure，不能证明新原型门禁或生产机制已成立。

## 8．本轮独立执行与证据限制

本机 Windows/amd64，已安装 Go1.27.1；`GOTOOLCHAIN=local`、`GOPROXY=off`。固定 HEAD 用 git archive 导出至仓库外：

`C:\Users\hoshi\.codex\visualizations\2026\10\12\01a13137-991b-7b42-8ea2-3c24008343a0\review-102`。

| 执行 | 预设范围与实际结果 |
|---|---|
| 原样副本定向测试 | 一次，go test 90s；`TestPipeNormalDrain`、`TestGateJudge`、`TestWitnessJudge`、`TestStartPoints`、`TestRecordAlive` 均 PASS，包6.005s，退出0；包含根层真实场景，未因零场景失效 |
| 共享 closer 正常/显式错误/收尾错误/双错 | 四组各一次、各30s测试期限；预设总准入180s。健康包5.964s/退出0；三项故障包分别0.089s、0.087s、0.090s/退出1，均为预期拒绝；16路显式关闭均只实际关闭一次 |
| 过程及清理 | 每组都同时执行真实 normal-drain，结果均 confirmed、已知子进程清理 verified；测试父进程已返回。结束前只读检查五个本轮已知 helper PID（46408、46212、33292、18404、18460）均已不存在；没有向其他 PID 发信号 |

日志为 `windows-direct.log`、`closer-healthy.log`、`closer-explicit.log`、`closer-deferred.log`、`closer-overlap.log`。补充 test 只存在隔离副本，未改被审仓库。关闭故障是实际关闭之后的返回值接缝，不是故意泄漏资源。三个非零是预先要求的故障检出，不能写作原样生产测试失败或忽略它们只报绿色。

一次读取命令加载 PowerShell 个人配置时出现 mise 入口找不到的环境告警；读取仍完成，后续禁用个人配置并直接调用已安装的 Go，没有安装或升级。该告警不是产品失败。本轮未遇到新的无效实验夹具、编译失败或补跑；没有重跑作者的全部变异。作者 Windows/WSL、抽取循环、临时完整包关闭变异及 `confirmEnd` 控制流模型仍按作者证据记录；曾未检出的期限变异、加强后的保护也保留，未称为本轮独立检出。

本机不能原生执行 Darwin。B1 原生场景中的接缝、B3 的 Darwin fake 直接检查与双架构运行是**独立核验的 CI 证据**，不是本机原生复现；没有原生 Close 错误、ENOMEM/EINTR、core 或不可终止故障的本轮动态证据。没有安装工具、运行布局探针或扩展 P-3，没有运行用户项目配置。

## 9．直接回归、遗留项与 G5 分层

| 项目 | 本轮判断 |
|---|---|
| R / N / M、Q-3(a)、W/K/C | 解码与独立规范化的核心判据没有被这次关闭/枚举修复改写；根层两架构数值及布局局部证据可采纳，不改变信号 NULL、读取失败优先等既定语义 |
| exact / Owned / Shortfall | 新增目标声明未改变外层归属及组合 judge。非目标仍须 confirmed；shortfall 仍使场景 invalid，不能由另一个场景的 Owned 抵销；补救不能恢复原结论 |
| 归属与清理 | 纯见证 PID 仍不自动取得动作资格；Job / 标记+身份核验前提保持。外层工具错误、归属未知、补救失败、Final unknown 或保险矛盾仍不能 PASS |
| Review 101 N1 | 显式预期集合及重复结果拒绝仍未实现；当前 P-3 的布局前提与实际工具登记有额外保护，未发现本轮引入新的缺失结果绿色反例。继续非阻塞增强，不将本次“理由丢失”混写为“未执行已算通过” |
| Review 101 N2 | 改写钩子后才构造 wRaw/kRaw，R_before 命名/完整保存问题未修。单字段 F/site 可辅助重建；原生无改写数值不因此失效。继续原非阻塞裁定 |
| Review 101 N3 | §15.11 已把“无需补救”限定为 P-3；元数据同源与已验证组合未擅自更新，相关措辞已收口；生产支持差额仍在 |
| 已登记链外缺口 | Linux 旧 pidfd EINTR、T5/S6 旧夹具关闭消费及部分旧接缝丢失并发理由，不是本轮新修复路径。此次没有据此重开所有 T1～T6，也没有因为它们已登记就给予新增安全豁免 |

旧 `4e690d8` 的原生 R/N/M 读数、无改写布局观测可以作为历史局部观察；其“没有工具失败”“清理全核实”“收尾有界”不能作为完整保证。§15.11 正确作了这种区分，没有将旧运行正式 success 改写为 failure，也没有据 success 否认缺陷。新的固定代码点同样不能把整体诊断 failure 隐去。

| 层次 | 剩余最小工作 |
|---|---|
| 当前 PR 的证据/合并条件 | §6 的布局失败原因丢失与目标未命中；先最小观测/已有证据检查，再取得后续固定对象的有效验证。三项原缺陷已关闭，不需要借此要求生产 Runner |
| 冻结前必需 | 采纳可适用的原型证据并解释剩余诊断缺额；明确生产 E1/E2、ABI/兼容性、承诺支持与已测组合的差额；完成其余 F 类门槛与技术候选的冻结审查 |
| 证据限制 | 只测所列 macOS26 两架构、SDK26.5与实际 xnu-12377.161.x；不是规范固定源码正在这些 runner 上执行的证明。只测指定信号/次数，没有原生 core/ENOMEM/EINTR；不自动删除未测15.x Intel支持 |
| 授权后的生产工作 | Runner/CLI、迁移0009、事务与取消、固定配置/空批次、输出和持久化叠加、C-9 接线及真实平台测试；不提前要求本原型 PR 实现 |
| 非阻塞增强 | N1 显式集合/重复拒绝、N2 R_before，以及首调用已经过截止的独立直接对照。N-102-1 是当前应优先补的追溯信息，但不把历史根因未知设置成永久门槛 |

没有发现需重新批准 Q-3(a) 或其他现有产品语义的变化；也没有批准新的产品取舍或接受风险。维护事项 M-1、M-14～M-17、N-087-1 保持原范围。

## 10．结束核验与操作记录

结束远程/本地核验：**2026-10-29 16:29:13～16:29:46 JST（07:29:13～07:29:46 UTC）**。PR HEAD/base、master、本地 HEAD/树仍为本报告固定身份；PR 仍 open、未合并、auto_merge=null。修复诊断仍 attempt1 failure，最终常规 CI 仍 attempt1 success，没有新 attempt 混入。报告落盘后核对原仓库仅新增本报告，跟踪文件与暂存区差异为空。

本轮未修改其他原仓库文件、夹具、原型或 workflow，未提交、推送、合并、改基、触发/重跑/取消任何远程作业。没有申请新资源或扩展实验类型。**修复技术 GO 与当前暂不建议合并是分层结论，不构成规范冻结、生产入口选择、生产授权或工作包验收。**
