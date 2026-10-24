# Codex Review 081：PR #34 原型 N4/N6 修复及直接影响的独立审查

## 1．结论

**PR #34：技术 NO-GO，当前不建议合并。正式 CI 已成功，技术修正条件尚未满足。**

- **N6 可以在原时钟口径问题的范围内关闭。** D/J 的截止、剩余预算及阶段时长已统一到 `phaseClock`；提前唤醒会重新核对截止，stopwatch 仅作观察。确定性边界测试及真实管道正对照均有有效证据。
- **N4 不能整体关闭。** 已解决“报告写不出且持有者已经退出”的精确旧路径，但“没有记录、持有者仍活着，所以写入没有失败”不成立。本轮在真实报告写失败后，仅延后退出，就再次得到 refuted。这个残余仍是工具故障的错误分类，没有变成 confirmed/绿色假通过，沿用原 N4 的非阻塞严重度。
- **新增 BLOCKER B1：当前 Windows S6 驱动可能在 J 阶段之前同步阻塞于 `Close`，无法兑现有界进入封闭/O3 的候选保证。** 换用 `io.Pipe` 正确隔离了时钟测试，却没有修复真实 OS 管道路径。文档仍将其主要归为“检查本身”的缺陷，并以此前的等待推定读取已经发出；现有调用链没有建立这一前提。下文给出有界独立实验、实际驱动栈及对应源码依据。

这不是要求本 PR 提前实现生产 Runner，也没有推翻已关闭的原型 B1/B2。阻塞点是**当前原型及其候选声明的真实缺口**，不能只以“冻结前再明确”保留与之冲突的无条件结论。最小修正可以是验证有效的停止协议，或明确撤回现机制尚不能支持的保证、纳入反例并收口现有候选/结果文档；不是必须先完成生产实现。

14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收。没有新增接受风险；M-14、M-15、M-16 保持未解决。

## 2．固定身份、范围与基线

| 对象 | 身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| 回归先行 | `91229b5beb7b6b06fe3a2d97b3e99f25d1a9e882` |
| 第一版 N6 修复 | `5054e54a14d3e9084aa675880175232e61361c17` |
| 最终修复代码 | `ac3b3bed4489404124d51181c1e8ed902d55c82e` |
| 最终 HEAD | `4e009a6b7fa7c81ca614b6d20789735175a8caaf` |

开始核验为 **2026-10-24 04:21:47 UTC**，完成技术核查时于 **04:34:06 UTC** 再次核对，报告落盘后于 **04:39:13 UTC** 最终复核。远程 HEAD/base 均与上表一致，PR open、未合并、`auto_merge=null`，最终常规 CI 仍为 attempt 1、completed/success。本地 HEAD 为固定目标，开始和技术核查结束时工作区均干净，报告不存在；落盘后仅新增本未提交报告。没有切换、清理或覆盖其他任务内容。所有编译与实验位于仓库外 `review081` 隔离目录。

已读取适用 AGENTS、[Review 076 的 N4](codex-review-076.md)、[Review 078](codex-review-078.md)～[Review 080](codex-review-080.md) 的相关裁定与边界、本轮全部差异、[原型文档](../wp2.5-prototype.md) §2.12、[处理记录](../review-resolution.md) §96 和必要的 helper、报告、清理、S6 调用链。

独立确认本轮 **12 文件，+630/−58**：

- 原型七文件：新增 `clock_check_test.go`、`keep_unix_test.go`、`keep_windows_test.go`；修改 `dest_test.go`、`gate_test.go`、`helper_test.go`、`pipe_test.go`。
- 四份文档：AGENTS、implementation-plan、review-resolution、wp2.5-prototype。
- 诊断 workflow 一份。

提交顺序为：`e7dbcdc` workflow、`91229b5` 回归先行、`3fe5701` N4 修复、`5054e54` N6 修复、`ac3b3be` 时钟测试改用内存管道、`6dc931d` 结果文档、`4e009a6` 合并核验及状态同步。

`ac3b3be → 最终 HEAD` 只有四份文档变化，原型子树同为 `5168b611d03c7a2ec2cfc8b1551597fac46732ef`。生产代码、根模块依赖、Schema、迁移、`ci.yml`、check-runner 草案及冻结契约均未变。workflow 仅在原分支列表中增加精确的 `test/wp2.5-prototype-n4-n6` 及注释；路径过滤、contents:read、并发不取消、三平台、Linux -race、15 分钟测试/20 分钟 Job 期限、PIPESTATUS 失败传播均保持。

### 2.1 PR #33 基线核验

固定 base 的父提交准确为 `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`、`38d15ef950fc1b7ba79ea08dfb47ab042a14cafa`；base 与该收尾 HEAD 的树同为 `daa0e2b02b1345d092766d04b4721953814b1e70`。这里的“树与最终 HEAD 相同”指 PR #33 的收尾 HEAD，不是本轮 PR #34 的目标。

[master CI 39988732786](https://github.com/XKHoshizora/specmetry/actions/runs/39988732786) 为 push、attempt 1、completed/success、8/8；独立读取八个 Job 的 checkout，均为固定 base 本身。Review 080 归档 SHA-256 仍为 `8fff4fb5cf31f1f986c5a93152ec57e8ba31e8c351506d04dc00363aef86ded8`。§96 仅追加 84 行，没有改写 §84～§95 或历史审查。没有重新验收 PR #33 的全部机制。

## 3．BLOCKER B1：Windows 的同步 Close 没有被收尾预算覆盖

### 3.1 固定对象的位置与实际影响

- [dest_test.go:284](../../prototype/wp25/dest_test.go#L284)：真实 `runS6` 委托的 `runS6On`。
- [309 行](../../prototype/wp25/dest_test.go#L309)：在调用者 goroutine 中逐个同步执行 `s.r.Close()`。
- [311 行](../../prototype/wp25/dest_test.go#L311)：**全部 Close 返回以后**才取得 J 阶段起点并设置 `T_join`。
- [319 行](../../prototype/wp25/dest_test.go#L319)：J 等待之后才封闭复制入口。
- [pipe_test.go:177](../../prototype/wp25/pipe_test.go#L177)：真实 F1 场景同样经过这条驱动，不是只供假时钟测试使用。
- [clock_check_test.go:4](../../prototype/wp25/clock_check_test.go#L4)、[原型文档 §2.12](../wp2.5-prototype.md#212-review-076-n4-与-review-078-n6pr-33-合并之后的独立小-pr)：本轮发现这一行为后，改用内存管道隔离直接测试，但仍将其主要定位为检查自身缺陷/生产冻结前前提。
- 原型文档 **C-2、C-3**（1186～1187 行）及 D20 的证据概括仍没有闭合这个停止阶段：无法从“已调用 Close”推出驱动已能进入 J、封闭并报告 O3。

如果该 Close 等待读取返回，则 T_join 尚未开始，复制入口也未封闭。F1 的残留写端可能保持静默，或读取处于不利调度窗口；驱动只能依赖外部写入/关闭来继续。总测试超时会让整个进程失败，但不是 S6 已完成有界收尾、错误输出或资源登记。**这不是操作系统未提供硬实时调度上界的问题，而是协议里存在一个没有自身结束出口的 I/O 等待。**

### 3.2 必须区分的三个时序

1. **Read 根本尚未进入、没有持有 FD 引用时**，先 Close，之后的新 Read 可以直接得到已关闭错误。不能笼统写成“读取未发出时 Close 都会等待”。
2. **Read 已持有 FD 读引用，但尚未发出 ReadFile 时**，Close 的取消可能找不到待取消请求；稍后才发出的同步读取仍能等待数据，而 Close 等待该引用释放。
3. **系统读取已经发出时**，取消针对现有请求；仍须等待实际完成，取消请求本身不是完成证明。

本机 Go 1.27.1 的 `FD.Read` 先取得读锁；`FD.Close` 请求 `CancelIoEx` 后等待引用归零。CI 所用 Go 1.27.3 的对应结构也核对过：[FD.Read / FD.Close 官方源码](https://raw.githubusercontent.com/golang/go/go1.27.3/src/internal/poll/fd_windows.go)、[os.Pipe 官方源码](https://raw.githubusercontent.com/golang/go/go1.27.3/src/os/file_windows.go)。微软说明取消针对 outstanding I/O，并不等待其全部完成，也不会改变句柄使未来操作自动被取消。[CancelIoEx 文档](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelioex)

正常场景中的 5 ms、100 ms 或完整 D 阶段，不是“此刻读取已经到达系统”的同步证明；每次复制循环都可能重新进入相同区间。文档已经诚实披露发现，却仍以“足够等待”推断现路径受限成立，依据不足。

### 3.3 本轮有界独立实验

在 Windows 仓库外副本中调用**未修改的 `runS6`/`runS6On`**，使用自己创建的 `os.Pipe`、真实 `os.File.Close` 和真实 `ReadFile`。通过 `SyscallConn.Read` 在取得相同 FD 读锁后、发出系统读取前设置可释放闸门，用于精确安排该窗口；没有修改 Go 工具链或以自造的“永不返回 Close”替代真实 Close。

顺序为：

```text
自有管道的 FD 读锁已持有，尚未 ReadFile
→ 真实 runS6(skipDrain=true) 调用真实 Close
→ 栈证实已在 FD.Close 的 semaphore wait，取消步骤已越过
→ 放行真实 ReadFile，写端仍由本测试持有
→ 超过 J 的预算，S6 仍未返回且尚未封闭
→ 测试向自己的写端写入 1 字节
→ 读取、Close 和 S6 全部返回并释放
```

首次用 50 ms 的探针预算观察 200 ms；随后保留原型的**默认 T_join=500 ms**，观察 2 s，结论相同。两次均为有意检查有界收尾要求的失败，保留原始结果，没有反复运行直到绿色。

默认预算实验的关键原始输出：

```text
internal/poll.runtime_Semacquire
internal/poll.(*FD).Close                     fd_windows.go:556
os.(*File).Close
...wp25.runS6On                             dest_test.go:309
...wp25.runS6
real runS6 still blocked in Close after 4*T_join (2s), before starting J
REVIEW081_RELEASE_BY_OWN_WRITER err=<nil>
REVIEW081_S6_RETURNED join=0s
```

测试 exit 1，正常排空对照 confirmed；关闭/复制相关任务最终全部返回。未启动不可控持有者，也未等待辅助程序定时到期。探针在释放后用人工 EOF 结束自建读取器，仅用于安全清理，不用其最终 O1/O2 计为生产或真实场景结论。

**证据边界：**这是对真实 OS Close 和实际 S6 驱动的受控 FD 调度复验，读取入口使用 RawConn 以固定临界窗口；不是未修改 `os.File.Read` 在原始场景中的随机复现。原始 `File.Read` 是否具有相同窗口由 Go 的对应读锁—系统调用—释放源码支持。作者 5054e54 的原始 12/180 次超时，本轮没有冒称逐次独立重现。

### 3.4 最小修正

当前不能继续把这个缺口仅列为未来生产问题。最小处理应至少：

1. 给停止读取/Close 自身明确的完成、预算和失败出口，使无法及时停止时仍有可信的封闭及任务/句柄归属协议；不能只把阻塞调用搬到 goroutine 后丢弃。
2. 在原型层加入可安全释放的读引用—取消—提交调度反例及正常对照，区分“请求停止”“Close 返回”“复制已结束”“入口已封闭”。无需生产 Runner 或迁移/CLI。
3. 若本 PR 暂不实现该机制，可明确把现路径降为**受限正例及已确认反例**，撤回其支持完整有界 S6/O3 的主张，同步 §2.12、C-2/C-3、D20 的证据概括与冻结前门槛。不能仅追加“待验证”标签，却继续声称三平台输出收尾整体成立。

单纯延长等待、使用 `io.Pipe` 取代时钟单测、或依赖全局测试超时，都不是实际路径的修复。是否最终改变产品支持范围仍由负责人决定；本轮没有接受无界等待的风险，也没有预先要求某个生产方案。

## 4．N4：旧路径改善，但 alive 不足以证明工具正常

### 4.1 成立的修复部分

[helper_test.go:87](../../prototype/wp25/helper_test.go#L87) 的 holder 现在检查 `report("epipe")` 错误并返回 `exitReport=91`。[report:190](../../prototype/wp25/helper_test.go#L190) 的注入以只读方式打开真实文件，让实际 `WriteString` 失败，没有直接伪造 verdict。

[pipe_test.go:189](../../prototype/wp25/pipe_test.go#L189) 在确认持有者已经结束后再读取报告，能保留退出前刚写出的完整记录；已结束且没有记录时判 invalid。状态未知仍通过既有 `observeErr` 和最终清理进入 invalid，不会因以后读取成功而抹去。

三平台日志与本机正常运行均确认：

| 情况 | 内层结论 / 退出 | 外层门禁 |
|---|---|---|
| 健康 F1，epipe 观察成功 | confirmed / 0 | PASS |
| read-end-kept，工具正常但预测不成立 | refuted / 1 | PASS |
| epipe 写报告失败，helper 已经退出 | invalid / 1 | PASS |

这三种结果不能混为三个机制 confirmed；第三项只证明外层正确拒绝失效的内层运行。

### 4.2 NON-BLOCKER N4-R：失败后、退出前仍可能被误记为反证

位置为 [pipe_test.go:183](../../prototype/wp25/pipe_test.go#L183)～[197 行](../../prototype/wp25/pipe_test.go#L197)，以及 [原型文档 §2.12](../wp2.5-prototype.md#212-review-076-n4-与-review-078-n6pr-33-合并之后的独立小-pr) 的“没有记录/仍在运行/写入没有失败”一行。

**反例：**holder 已发生输出写失败，真实 epipe 报告写入也已失败，但在调用退出前被调度暂停。它仍活着；这个状态不能证明写入没有失败，更不能证明观察工具正常。

在隔离副本中，本轮保留实际 F1 场景、文件写入故障、判定和清理，只在真实失败分支返回 `exitReport` 前插入有界的 5 s 延迟，并增加独立命中标记：

```go
if report("epipe") != nil { // 原来的真实写入已经失败
    if injected("epipe-report-fail") {
        injectHit("Review081.after-real-epipe-report-failure", os.Getpid())
        time.Sleep(5 * time.Second)
    }
    return exitReport
}
```

没有修改内层 verdict 或外层期望。父端照常等报告 3 s；当它检查状态时，helper 还未退出。实际 `report.epipe-unwritable` 给出：

```text
want=invalid  got=refuted  inner_exit=1  inject_hits=2
reason="differs: holder's late writes fail"
cleanup/insurance verified，最终没有残留
外层该项 PASS=false；整个定向运行 exit 1
```

其余健康/保留读端对照仍通过。失败发生后、退出前的调度暂停足以产生结果，不依赖新的生产行为。最终清理成功没有恢复观察；清理还会终止尚未退出的 helper，之后的“已结束”也不能追溯证明它先前没有工具错误。

**严重度与关闭：**N4 仍不能整体关闭；这仍是 refuted/invalid 分类错误，内层和门禁均非零，没有新的 confirmed/绿色放行证据。保留原 N4 的非阻塞严重度，不追溯改变 PR #33 的旧裁定。本轮 NO-GO 另由第三节 B1 支撑。

最小修正是提供与父端同步的明确完成/失败或健康观察依据，或在没有此类依据时保守判观察无效；不能仅凭一瞬间 alive 判反证。Unix 不能获取这类已脱离父进程的后代退出码，因此只更改退出码不足以构成跨平台协议。read-end-kept 要保留 refuted，应另外证明观察工具在目标区间正常、预测确实未发生，而不是把“未取得记录”直接当证明。若要随本 PR 宣布 N4 完成，这个残余须处理。

### 4.3 read-end-kept 的直接检查

[keep_unix_test.go:13](../../prototype/wp25/keep_unix_test.go#L13) 使用原子 `F_DUPFD_CLOEXEC`；[keep_windows_test.go:13](../../prototype/wp25/keep_windows_test.go#L13) 在持有文件控制引用时调用 `DuplicateHandle`，明确禁止继承。重复的是实际读端，而非又开一个无关文件；相关 helper 此时已经启动并被核对，额外句柄不会意外传给它。

两个额外读端均登记到 `s.defer_`，在场景最终清理中关闭；helper/Job 随后按既有身份和归属规则结束，结果日志核验释放。这个反证夹具成立，并不解决一般的“活着且没记录”歧义。

## 5．N6：时钟边界与 I/O 机制分别裁定

### 5.1 时钟修复成立

- [phaseClock:195](../../prototype/wp25/dest_test.go#L195) 与 [waitOn:221](../../prototype/wp25/dest_test.go#L221) 统一 now/wake。
- 剩余时间由同一时钟的 `deadline.Sub(now)` 计算；收到唤醒后继续循环，不直接当成截止。
- 每阶段起点、截止及结果时长都使用该时钟；所有流共享阶段截止，没有逐流重新分配整段预算。
- 正常复制完成仍可提前返回，EOF/O1 没被改成强制用满预算。stopwatch 仅另列 `drainFine/joinFine`，不再作该阶段是否用满的断言。
- `T_kill=5 s`、`T_drain=1 s`、`T_join=500 ms` 及测试/Job 时限均未改变，无新增容差，不承诺物理时间或调度的硬实时上界。

[TestPhaseClock:156](../../prototype/wp25/clock_check_test.go#L156) 的预期来自固定要求：恰好到截止为 1 s；差 1 ns 不结束；早醒 450 µs 后再等剩余 450 µs。测试显式推进时间，不以被测函数计算出的时长反算期望；两条/一条复制器都必须被停止并返回 O2。

本轮在 `91229b5` 独立复验三行失败：旧记录约为 21.7 µs、201.3477 ms；early-wake 错误进入 J。最终固定代码三行全部通过。真实 `T1.pipe.escaped-holder` 仍运行，本机记录 D=**1.0005938 s**（同阶段 stopwatch=**1.000611 s**）、O2/O1、最后文档完整，confirmed/0。

**裁定：旧 N6 的截止/测量口径缺陷可关闭。** 不能由此把 Close 的生存期问题也算作解决，也没有重写旧 refuted 或证明一切竞争已经消失。

### 5.2 第一版超时及换用 io.Pipe

5054e54 中直接测试在假时钟上不经历真实 D 等待，却使用 OS 管道。作者后续 60 次/180 子检查的 12 次失败，报告位置是 `s6Done` 的 5 s 返回期限；源码允许实际驱动正在前述同步 Close 中等待。不是“阶段时间算错”的断言失败，也不是 Go test 的总期限变化。

改用 `io.Pipe` 合理地隔离了时钟单元测试：关闭它的读端能解除当前/后续读取，不依赖 Windows 的取消窗口。本轮对第一版只运行一次边界测试及正常对照，均通过，**没有独立复现作者的 12/180 频率，也没有据这次通过否定其失败**。作者 200 次/600 子检查零失败仍是作者证据，未改称本轮压力验证。

分别裁定：

| 层面 | 结论 |
|---|---|
| A．时钟边界测试 | 已正确隔离并修好，支持关闭 N6。 |
| B．当前真实管道 | 已有正常成功样本；Windows 停止窗口仍使 Close 可能阻塞，J 和封闭尚未开始。见 B1，不能宣称完整机制已成立。 |
| C．冻结前规格/必要原型 | 须明确停止请求、Close、E-copy-end、预算、O3 和资源归属的关系，并以安全调度反例验证；无需提前生产接入。 |

## 6．直接回归及文档状态

保留既有 B1/B2 的关闭范围。固定目标的本机定向运行中：35 项 Windows 门禁回归、13 项组合、直接判定/见证/启动入口测试全部通过；normal-drain、F1-skip-drain、escaped-holder 三个机制场景 confirmed。

修复后诊断三平台也保留：

- forge-owner/stale 的非内层夹具未进入动作入口；Unix 信号接缝计数为 0；夹具核验时仍存活，随后由组合自身结束。
- forge-owner-mixed 中真实自有遗留仍实际被补救结束，非内层夹具被拒绝，最终 unknown 没被伪装成 ended。
- 原十项组合仍区分预期拒绝、工具错误、补救失败/未知状态与保险清理；没有因 N4 新增合法 refuted 对照而放松这些门禁。
- Linux 的短链/损坏链、观察错误和见证拒绝在本轮正式诊断实际运行；本机 Windows 没有运行 Linux 专属机制。

**非阻塞状态收尾：**check-runner 草案头部仍说 PR #33 不合并，旧 PR 标题仍有 not for merging。实际合并已由固定对象和 master CI 证明；这是待维护的事实措辞，不据此否定合并，也不把准备文档合并当作规范冻结。本轮不修改它们。

§96 忠实保留 PR #33 的合并身份和授权边界、旧 N6 读数、第一版 CI 绿色之后的本机失败；但 N4 的 alive 推论及 Windows 停止阶段的现保证须按第三、四节收口。未发现对其他已验收工作包、方案 B 或维护项状态的变更。

## 7．CI：正式结果、checkout 与计数

独立读取 run/job API、各相关完整日志和提交对象；未重跑或触发 workflow。下表均为 attempt 1。

| Run | 事件 / SHA | 正式结果 | 说明 |
|---|---|---|---|
| [39991866363](https://github.com/XKHoshizora/specmetry/actions/runs/39991866363) | push / 91229b5 | completed/failure，三平台均 failure | epipe 门禁得到 refuted 而非 invalid；三个时钟子检查均为断言失败。 |
| [39992975299](https://github.com/XKHoshizora/specmetry/actions/runs/39992975299) | push / 5054e54 | completed/success，3/3 | 保留为真实绿色历史，不代替其后发现的本机直接测试超时。 |
| [39994591073](https://github.com/XKHoshizora/specmetry/actions/runs/39994591073) | push / ac3b3be | completed/success，3/3 | 最终修复代码诊断；各 Job 实际检出该 SHA。 |
| [39996294870](https://github.com/XKHoshizora/specmetry/actions/runs/39996294870) | pull_request / 4e009a6 | completed/success，8/8 | 最终常规 CI，不运行独立原型模块。 |

最终 CI 八个 Job 的实际 checkout 均为 **`a00358c161daa33ffce2e09ae43c002c30d7deb1`**。父提交准确为固定 base 与最终 HEAD；合并树及 HEAD 树均为 **`efea6aae9d1e8e9fe142edfc7635271c22dba14e`**。没有把 PR 元数据 HEAD 当成实际 checkout。

八个 Job：Go lint `120560567131`；Ubuntu Go `120560567422`；Web `120560567457`；Windows Go `120560567482`；macOS Go `120560567575`；macOS/Ubuntu/Windows dashboard build 分别为 `120561900030`、`120561900090`、`120561900147`，均正式 success。

### 7.1 原型诊断的实际计数

| 对象 / 平台 | 机制场景 confirmed | 门禁 PASS | 组合 PASS | 时钟直接检查 |
|---|---:|---:|---:|---|
| 91229b5 Ubuntu | 38/38 | 38/39 | 13/13 | 三行失败 |
| 91229b5 macOS | 27/27 | 29/30 | 13/13 | 三行失败 |
| 91229b5 Windows | 29/29 | 34/35 | 13/13 | 三行失败 |
| 5054e54 Ubuntu/macOS/Windows | 38/27/29，各全 confirmed | 39/30/35，各全 PASS | 各 13/13 | 当次均通过 |
| ac3b3be Ubuntu | 38/38 | 39/39 | 13/13 | 三行通过 |
| ac3b3be macOS | 27/27 | 30/30 | 13/13 | 三行通过 |
| ac3b3be Windows | 29/29 | 35/35 | 13/13 | 三行通过 |

修复后 Job：Ubuntu `120554702004`、macOS `120554701850`、Windows `120554701405`。各直接检查 `TestPhaseClock`、TestCandidates、TestGateJudge、TestWitnessJudge、TestRecordAlive、TestStartPoints 实际 PASS。Results 重印去重后，机制 ID 并集仍为 **53**；不累加三平台数，也不将门禁、组合、直接测试和内层结果重复计为机制 confirmed。

各次诊断顶层设计 SKIP 为两个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak`；相应门禁/组合日志确认它们另启进程实际执行。SKIP 不记通过。新 F1 三项在最终诊断中分别为 confirmed/0、refuted/1、invalid/1，外层均正确按现用例要求 PASS；这些绿色没有覆盖本轮新增的延迟退出调度。

Ubuntu 原型通过 Linux 条件设置 `race=-race` 后实际运行 `go test`；macOS/Windows 原型未开启竞态检测。CI Go 1.27.3；Ubuntu kernel 6.17.0-1022-azure/amd64，macOS 26.6.2（Darwin 25.6.0）/arm64，Windows build 26100/amd64。不能扩张为其他版本或未提交实验的验证。

### 7.2 常规 CI 与维护事项

最终常规 Ubuntu 普通 Go 步骤保留 -race；另外编译存储测试二进制，以 root 运行八项 CrossUser，八项实际 PASS。root 步骤不等于普通 -race 步骤；Windows/macOS 按平台跳过该步骤。Windows 保留动态包分组及 CLI 20 分钟预算。

真实旧 Git、真实会取对象的 Git 夹具、平台权限不适用项、opt-in M-14 诊断等仍有设计跳过；没有将它们计为已验证。常规 CI 没有原型模块，不能替代上述诊断，更不能否定本轮独立反例。

旧 run `39968755313` 的 SQLITE_BUSY、此前取消、N6 各次 refuted 和旧读数均保留，未重跑；本次存储测试通过不关闭 M-15，也不证明旧失败根因。M-14/M-15/M-16 未解决。

## 8．本轮独立运行与限制

平台为 Windows 11 build 26300/amd64，现有 Go 1.27.1；使用 `GOTOOLCHAIN=local`、`GOPROXY=off`，未安装升级工具，未执行用户项目配置。全部编译在仓库外成功。

| 实验 | 硬时限 | 实际结果 |
|---|---|---|
| 回归先行：三个 F1 门禁、三个时钟边界、正常排空 | 60 s | exit 1；N4 门禁和时钟三行按预期检出旧机制；两项 F1 对照通过。 |
| 最终固定代码：TestPhaseClock、TestGate、TestGateCombos、三个真实管道场景及直接检查 | 240 s | exit 0；35 门禁、13 组合、直接测试通过；三个场景 confirmed。Gate 102.19 s、组合 11.88 s。 |
| 第一版 5054e54：一次 TestPhaseClock 与正常排空 | 20 s | exit 0；未复现作者的概率超时。没有反复施压直到失败。 |
| 实际报告失败后延迟退出：三个真实 F1 门禁及正常排空 | 30 s | exit 1；epipe 报告故障命中、仍被误记 refuted；健康/读端保留对照通过。 |
| 实际 OS Close/实际 S6 驱动，受控 FD 调度 | 各 15 s | 首次 50 ms 探针及默认 500 ms J 预算均 exit 1，分别观测 200 ms/2 s 的预算外阻塞；自有写端释放后所有任务返回。 |

首次失败均保留，没有将准备/编译失败算作机制检出，也未为绿色重跑。原始场景运行没有出现本轮 Windows Close 失败；独立调度实验与对应源码证明提供的是另一条窗口证据。作者本机 12/180 次失败、200/600 次无失败及 M35～M38 仍是作者证据，没有冒称本轮全部重现。

本轮未在 Linux/macOS 本机动态运行；通过完整 CI 日志和源码限定对应结论。没有故意遗留进程、强制 PID 复用、不可中断睡眠或信号未知对象；探针只用自有管道和有界可释放任务。结束检查 `review081*` 进程为 0。

仓库外关键日志 SHA-256：

| 日志 | SHA-256 |
|---|---|
| old-run.log | `c90678486c2d745241adc55b1582f424b0c7b3d15ddaa1639d4e66df687b2d46` |
| fixed-run.log | `43f257d868b2ce1ed0a1ac05e985740ced2153142837c8d1a5b2a01fc1853ad1` |
| first-run.log | `8eb3eda72afc3927fc883be1e9fc8cb44e10cbcb236a06c28d1fa298d3680abd` |
| n4-run.log | `a43ff85affea52c1bd3e92389e1d8e18af0b1c8d871f93fb236c82ffc8bd06ca` |
| close-default-run.log | `b54938cb16cbe6e1b588192350519a43999bc9d2d0be48f74066e217d162e806` |

## 9．剩余工作与最终答复

**本 PR 的必要最小修正：**

1. 处理 B1 的当前有界停止/封闭缺口，或撤回相应未成立的机制保证、登记受控反例并同步现候选/证据结论；不能仅说留待生产。
2. 若本 PR 仍以同时关闭 N4 为交付目标，修复第四节的退出前残余，保留健康 confirmed、可靠反证 refuted、观察故障 invalid 三类证据。没有必要把所有工具错误细分为相同原因，但不能以 alive 代替正常观察证明。

**冻结/授权前仍需：**14 项负责人选择（D1～D9、D11、D13、D18～D20），T1/T6 的停止与输出生命周期，Linux 闸门生产拓扑与 C-9 规则设计，macOS ABI/支持范围与枚举失败预算，Windows 父端期限/未覆盖失败点及资源移交，T5 来源/PATH/worktree 排除等具体规格及必要隔离原型。本轮不批准新的风险或保证变化。

**授权后、生产验收前再完成：**Runner/CLI、全局闸门/静态规则接入、真实终端/取消/继承与 JSON、迁移 0009 与正式 check-run 契约、配置固定/空批次/worktree/seq/事务、§8.6 持久化组合、生产隐私和三平台/E2E 测试。它们没有自动成为本 PR 的阻塞。

| 要求 | 裁定 |
|---|---|
| N4 是否关闭？ | **不能整体关闭。** 旧“已结束而无记录”路径已修，退出前残余仍会错误 refuted；非绿色假通过，保持非阻塞严重度。 |
| N6 是否关闭？ | **可按原时钟口径问题关闭。** 不包含 Windows Close 生存期及真实 I/O 有界保证。 |
| PR #34 技术/合并建议？ | **NO-GO，当前不建议合并。** B1 是当前原型驱动及机制声明的缺口。 |
| 正式 CI 是否满足？ | **已满足：最终 HEAD attempt 1，8/8 success；诊断代码身份一致。** 绿色不能替代未覆盖的精确反例。 |
| 既有 B1/B2 是否回退？ | 未发现本轮直接回退，保留 Review 080 的关闭范围和受控归属前提。 |
| 新产品决定/接受风险？ | 没有本轮可自行批准的新增事项；14 项继续未决，没有新增接受风险。 |
| 冻结、生产授权和验收？ | 均未发生；WP2.5/M2 未验收，维护事项未关闭。 |

建议由实施方仅收口上述停止机制/当前声明与 N4 退出前分类，再作定向独立复审；不需要重做已验收模块或整个 T1～T6。报告仅新增本地未提交文件，不修改其他原仓库文件，不提交、推送、改基、合并或重跑 CI。
