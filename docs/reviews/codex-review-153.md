# Review 153：PR #57 的写者空闲握手定向复审

## 1. 裁定

**B-152-1 关闭；N-152-1 完成。固定对象在本轮限定范围内技术 GO，建议合并 PR #57。没有发现新增 BLOCKER 或直接回退。**

本结论来自对握手、实际 Writer／Log／Closeout 及测试调用者的静态推演，并辅以已有原始记录；不是由重复测试或常规 CI 绿色倒推握手正确。

| 项目 | 裁定与范围 |
|---|---|
| B-152-1 | 关闭：到 D 前已建立全部现有记录写完、没有可继续准入的文件工作、直到测试唤醒前无新记录的前提 |
| N-152-1 | 完成：瞬时未取得返回、调用前钩子、精确窗口、请求 done 与后台终态的边界均已澄清 |
| B-151-1、N-151-1、N-150-1 | 保持 Review 152 的限定关闭；N-150-1 仍限原 37 项／49 个实例，不扩大为全部旧测试生命周期审计 |
| 第一阶段工具实现 | Review 150 已关闭问题保持原限定；本轮没有修改或重新验收工具实现 |
| PR #57 | 技术 GO，建议合并这个固定对象；合并由负责人决定，本审查没有执行或授权合并 |
| 后续执行 | 第二阶段、E-10(ii)、LE-5①／②、N2 等仍待决；不产生真实调用、伪工具、平台控制、LC-9、正式取证或第三批生产授权 |

依据：[Review 152 全文](codex-review-152.md)、[处理记录 §198](../review-resolution.md)、[实现记录 §12.2／§13](../wp2.5-c9-evidence-tool.md)，以及本轮全部增量和直接依赖。没有运行 Go、测试、构建、实验或采集器。

## 2. 身份、归档与实际改动

开始核验时工作区干净，目标报告不存在。本地、GitHub 远程分支和 PR #57 head/base 相符，PR open、未合并。

| 对象 | 核验值 |
|---|---|
| 分支 | `test/wp2.5-c9-evidence-n150-1` |
| HEAD | `f9ad771cc9cc76de4361ed94027870db06895670` |
| tree | `73332d1b5d21ab4b33481ab35a2ad84fd5ec4f94` |
| 工具模块 tree | `2866d4609f26e49095d98ab098877d55cdf59a6e` |
| base／远程 master | `cd44f674f220116a19f82b2f8034a451d2ac642a` |
| 上轮对象 | `5d5fd53c274f7412e9774d7f33d3a3bc2d721b02` |

提交顺序成立：`0331e08b07c3c0fce3d09b1bf1ece1867b7587ef` 单独归档 Review 152；`ec173bb6ecac5ef01fff975148d9eb345458951f` 于 2026-11-14 23:32 JST 登记 §198；`4eae0e7a73d6ef8ad9d43cf2e9d3de9af527f3ae` 于 23:37 修改测试；最终提交于 23:44 补记录及状态。授权登记先于代码。

Review 152 为 **28,157 字节**，SHA-256 `3ce2b135165bfcf2910a418728dfa7f39185e94a822c29192762429c4df61759`，与上一轮交付一致。归档提交只新增这份报告。以 Git blob 原始字节核对：旧处理记录 1,403,691 字节是新文件 1,416,484 字节的完整前缀；新增 12,793 字节为 §198，§197 及此前未回改。

上轮对象至 HEAD 共 14 个文件。工具模块只改五份 `_test.go`：新增 `regress152_test.go`；修改 `regress151_test.go` 的到期子测试及注释、`regress148_test.go` 的构造辅助、`tasks_test.go` 的常量提取、`finish_test.go` 的说明。其余是报告归档、§198、实现记录及状态同步。工具非测试代码、正式入口、单次请求算法、bound、W/D、依赖、workflow、冻结正文和终判规则未改；`git diff --check` 无输出。状态仍为待 Review 153，没有提前宣布本轮阻塞关闭。

固定测试提交 `4eae0e7…` 的 tree 为 `2b5732461c759c083663407d80d7b82d2c1da1aa`，工具模块子树与最终 HEAD 完全相同。最终文档提交没有再改测试。

## 3. B-152-1：握手充分性的独立推演

以下行号均指固定 HEAD `f9ad771…`。主要位置：[regress152_test.go](../../prototype/c9evidence/internal/job/regress152_test.go) 131–169、208–222；直接依据：[writer.go](../../prototype/c9evidence/internal/chain/writer.go) 182–236、335–470，[log.go](../../prototype/c9evidence/internal/chain/log.go) 的 Append、take、sealWith，以及 [job.go](../../prototype/c9evidence/internal/job/job.go) 291–344、771 起。

### 3.1 初始化、成功写入与发布状态

`RegisterWriter` 依据不可变的 Log.Files 一次建立全部 FileSnapshot，随后 Start 发布 started，再启动唯一写者。实际顺序是：目录创建并发布 → createAll 顺序创建所有记录文件并发布 → writeUntilSeal 写表头和条目。

握手要求 started、Mkdir 成功、无 Terminal／Errors；每个预先登记的文件均 Create 成功、HeaderWritten、WrittenThrough 等于该文件当前最后序号，且未开始 Close；日志未封口，日志文件名都能在写者文件集中找到。因真实注册代码预先建立非空且完整的文件集合，这里不会通过“空 Files”省掉初始化，也不会把尚未返回的 Create 当成功。

`writeFile` 在真实 Write 返回、normWrite 检查、更新运行哈希之后，才在 outer 锁下发布 Writes／HeaderWritten／WrittenThrough；错误先进入 Errors，失败写不前进 WrittenThrough。单个写者按文件串行处理，不存在另一个写入任务在相同已完成位置后台继续写的路径。

因此，在本场景中所有 HeaderWritten 与各文件末序号都匹配，确能排除尚未开始／未返回的表头或旧条目写入，不能仅凭一次“无终态”就通过。握手没有把测试自设标记当成写完证明。

### 3.2 分开取快照不会拼出假的完成状态

waitWriterIdle 先在 outer.Watch 的锁下取得发布副本及其通知，再读取 Log.Items 和 Sealed。日志条目只追加，序号全局递增；写者只能写入此前 take 实际取得的条目。

如果某条记录在外层副本之后才追加，后读的 Items 会包含它，它所属文件的最后序号必大于旧副本的 WrittenThrough，握手为假。反之，若全部相等，读取 Items 时的全部现有条目必已在该副本所见的写入结果中完成。不会把“较新的日志尾”与“较旧的写入尾”拼成已追上的假状态。Sealed 在这里也只会由 false 变 true，true 被拒绝。

这不是通用的跨锁原子快照证明。它依赖当前实际路径的单写者、追加式日志、固定文件集合，以及接下来没有其他生产者的事实；这些前提在受审用例中成立。

### 3.3 工作完成之后没有剩余生产者

本场景使用固定 FakeClock，RunDomains 期间没有推进到 W，也没有受控工作挂起；RunDomains 返回的是 work.done。runWork 同步完成 F、R、来源核验后，才在 Job 锁下发布 WorkDone 并关闭 done；这些路径的记录追加先于 done，写者自身只发布 outer 状态，不向 Log 无限追加自身记录。

落后对照还显式检查 WorkDone。原到期子测试虽未另写同一断言，但其时钟及调用者接线已经排除 W 返回路径；不能把这一论证扩展到所有 RunDomains 返回情形。

推进到 D 后，Job.Finish 先置 finalized，再调用 Closeout；Closeout 第一步按原 D 走 expired-at-m.Run。这个出口不调用产生封口记录的 Provisional、不 seal、不登记诊断，也不向 Log 追加。Level1 只消费已有状态。因此从握手到 endTasks 唤醒之间，没有第二个记录生产者。

### 3.4 发布之后的局部步骤与旧通知不构成遗漏

握手**不证明 CPU 已经执行到 `<-changed`**。例如最后一次 writeFile 已发布成功，写者可以仍暂停在清空 `pending/header/first/last` 之前。随后只会清空刚完成的缓冲、检查剩余空缓冲、处理未封口分支并等待；不会对刚写过的同一批数据再调用 admit。

若考虑循环曾持有的已关闭通知，关键仍是实际 take 的位置和末序号：旧通知若对应尚未取到的新记录，末序号不相等，握手不能成立；若只是允许继续执行一个空批次，空 pending 在 writer.go:416–417 直接跳过，根本不进入 419 行的 admit。重新 take 在同一 Log 临界区取条目及当前通知，不能丢掉随后追加。没有封口，就不会离开循环去 closeAll 或创建 manifest。

所以准确结论是“当前没有可继续推进的文件工作，直到新记录／封口之前不会因旧工作触发到期终态”。文档 §13.1 中“等待”“不会自行推进”按这一业务含义成立，不应读成已观测到 goroutine 阻塞、没有任何局部指令继续执行。§13.6 已明示没有观测阻塞点；这个限定不改变握手规则，也不产生新的阻塞。以后整理说明时可直接使用“无待处理文件工作”，避免把业务空闲与调度状态混写。

### 3.5 Watch 与有界等待

Watch 的副本和 changed 通道在同一 outer 临界区取得。写者若在副本之后发布，则旧通道关闭；握手暂时为假时可立即重读，不会因为发布落在读取与等待之间而永久漏醒。在本用例 WorkDone 后，能使条件由假变真的只剩写者发布；日志追加只会增加待写尾，不会单独使假变真。

waitWriterIdle 使用一次 `time.After(d)`，不滚动取得新期限；失败由调用者 Fatal，再进入已登记的 settle。这里仍承认调度与普通短临界区的边界，不把有界等待写成硬实时返回保证。没有更改原 W/D 或 bound。

**B-152-1 所需的三项前提在实际受审调用链闭合，可以关闭。**

## 4. 事件顺序、计数、两类对照与清理

### 4.1 原到期对照没有削弱

[regress151_test.go](../../prototype/c9evidence/internal/job/regress151_test.go):270–290 与 regress152_test.go:230–277 的顺序为：

RunDomains → 有界握手 → 保存条目数、各文件发布状态及调用计数 → 推进到 D → finishWant 原结论 2／非零／expired-at-m.Run → 核对原条目数、未封口、latch、无写者终态、无诊断登记、发布状态和文件调用计数不变 → endTasks 追加唯一测试唤醒项 → 写者 expired-incomplete/late。

随后检查恰好多一条 FileCloseout／wakeLine、发布写入不变、文件操作计数不变、所有文件内容没有 wakeLine、记录文件未开始 Close、manifest 未创建、Finish 实际调用恰好一次。没有提前释放阻塞来换取“父端先返回”，也没有用接受任意终态代替主动唤醒证据。正常子测试的原判定未改。

末条测试唤醒先进入 Log，写者下一次有字节要写时经原 admit 拒绝；计数在调用开始时发生，因此“不增加调用计数”确实能拒绝已经启动但尚未返回的文件调用，不能只记录返回次数来藏住进行中操作。

### 4.2 countFS 的接线与锁

regress152_test.go:34–118 的包装器在 Mkdir、Create、Write、Close 进入时先锁下加计数，释放计数锁后才调用原 memFS／memFile；仅可选第一次 Write 在计数之后握手并阻塞。它保留内存文件系统原来的返回和错误，没有改写 Write／Close 结果。

containing 持有 memFS 锁后逐个取 memFile 锁；原 Create 只取 memFS 锁，Write／Close 只取 memFile 锁。计数锁不会跨内存文件调用或 gate 等待，没有反向嵌套的死锁链。调用快照和发布快照不是通用原子组合，但握手成立后的无新文件工作前提使这里的比较稳定；有意挂起的第一次 Write 也不会被计数当成完成。

### 4.3 新增两个顶层用例分别成立

| 用例 | 已建立的前提、断言及限制 |
|---|---|
| HandshakeWaitsForALaggingWriter，285–315 行 | 第一 Write 已进入，gate 确实持有；工作完成后直接判定及 200ms 握手都不能通过，调用数为 1；释放后才可通过，再完整执行主动唤醒对照。短等待失败由持有保证，不靠调度猜测 |
| WriterExpiredByItselfNeedsNoWake，322–363 行 | 第一 Write 持有到原 Finish 到期返回之后；先确认持有中无终态，再释放并有界取得实际终态；endTasks 后条目数及操作计数不变，不追加唤醒，调用仍为 1。它只证明无需唤醒的健康收尾，未替代主动唤醒用例 |

两个用例都在等待 entered 之前登记 settle，内含 `fs.release` 与 `g.freeAll`。释放由 sync.Once 保证可重复调用但只关闭 gate 一次。失败时先释放，再有界等 WorkDone，最后等同一 Finish 请求并收尾任务；目录 Cleanup 的登记更早，按后进先出在这些步骤之后运行。未加入无界等待或重取 W/D。失败出口探针只动态覆盖其中一个具体失败点，见 §6。

### 4.4 共享辅助与直接回退

`newJob148` 现在委托 `newJob148FS(..., newMemFS())`，旧调用者仍使用同一内存文件系统实现、配置和 seams，只有新对照传入计数包装。`wakeLine` 是原文本逐字提取，endTasks 的条件和追加位置不变。finish_test.go 只加说明，单次请求表、done 和原 Final 的保留机制未变。

检查实际 diff 和受影响调用者，没有发现 B-151-1、N-151-1、原限定 N-150-1、第一阶段实现或原父端先返回断言的直接回退。不把未纳入本轮的 Review 147／job_test.go 扩成已完成生命周期整改。

## 5. N-152-1 精化

代码与实现记录 §12.2 后的更正、§13.5 一致：

- notReturned 仅说明该次检查没有取到返回值，不能证明 endTasks 已经开始等待。
- entered 在调用前钩子关闭，只证明钩子到达及锁持有前提，不能证明 CPU 已进入 Finish 第一条语句。
- finalized 已置位而 Started 未发布的精确窗口仍没有动态覆盖；单次请求机制不再依赖这些内部状态，原静态关闭结论保留。
- 请求 done 表示该次 Finish 已返回且 Final 发布；写者／诊断终态表示各自最后状态发布；都不泛化为观测到 goroutine 已完成退出调度。

§12.2 原表作为历史描述保留，紧随其后的明确更正和 §13.5 限定当前读法，未把旧日志改写为新增握手证据。**N-152-1 完成。**

## 6. 本机原始证据：取得范围及独立核对

读取用户指定 `s198` 中日志、补丁、脚本及 demo／audit／failprobe 导出副本的相关源码，没有执行它们。以下大小和 SHA-256 与现有材料、实现记录相符。

| 材料 | 字节 | SHA-256 |
|---|---:|---|
| fix.diff | 20515 | `52deb1f64b6e21ed28f9547d9a5c85efc74b8844cb126c8fea4ae82a0cb7bc26` |
| demo_patch.py | 992 | `fa46cc8f1ff2b09cd2c00722b7c0c50ff979d7107b12934aab7999b8ec5327dc` |
| demo-seam.diff | 202 | `846b9a6009827fc4003e1808043fcf9be2f33d9f80cad1dbfb43153e43df7b2a` |
| probe152_test.go | 3027 | `bd69e103ce45ab43855e387c7f43c567bb5ac6b1f1b1150fe891ee58f776bd12` |
| probe152fail_test.go | 1477 | `56cc6adff68699d8b1397543720e21a72cf98e3a878bda238c5c08c4f580ba08` |
| audit-probe.diff | 749 | `b27c8da8306f4c1e24bf3d6d4b8c9ab85c882d0a6b3f2226fc019f88cf4c8faa` |
| pre-change-demonstration.log | 1961 | `ea361f53ae668f09edb16273a60178388c990038e99345d18df517e0d7508fc3` |
| post-fix-targeted.log | 5834 | `5d9f5c107b680b8d9999727941097a3b96b8fafe0d3b7e9f02efe08e4435db80` |
| post-fix-job.log | 7285 | `7ec7ebe83bc54f65069e6713e715a47e55989b9798994469c96eba9618165230` |
| post-fix-module.log | 48120 | `14a4edfe8645a95d29bec29f0928058638a509650f7c70189404aa061b637a98` |
| post-fix-x20.log | 144736 | `134fe270dee20c732502c568ff7ec0ff63875f39a793baed800306924c1792bb` |
| post-fix-static.log | 2699 | `39edb79fb3e5a500cf3b53b0a26a290b07b5e39c982e47bb740eb60292206985` |
| failure-exit-probe.log | 1074 | `05765fca251f89379eb9209a15b0e0a2b83c5f875851faf1f2a4d2549b197bf8` |
| audit-probe-after-fix.log | 17154 | `3de10f77d942e3e6140dfd62bc62232ee630b95d5982faf046973ac3aa6809e4` |

### 6.1 旧测试展示：实际失败于原断言

demo 从 ec173bb 的旧工具子树导出，加 newJob148 的测试接缝和独立 probe。导出副本的旧 regress151_test.go 哈希仍为 `a28e76ba8853aed8abdacb73123b977e812226f0736cf441f36a180bc0abab0d`，没有改旧断言。

probe 只持有第一次 Write、在调用 Finish 前的钩子释放并等已有发布；未人为添加错误、改期限、改原 Final 或制造第二次 Finish。日志显示正常子测试通过，到期子测试先记录写者已为 expired-incomplete/late，然后在旧文件第 268 行的“writer 尚无终态”断言失败，退出 1。没有出现先前工具错误或探针超时来替代这个命中。

这动态展示的是“Finish 前写者已经自行到期”。“Finish 后、旧断言前恢复”仍只保留 Review 152 的静态可达反例，没有冒称本次也已动态构造。两者命中同一个不成立的原假设。

### 6.2 修复后固定提交的记录

正式日志关联 `4eae0e7…` 及其 tree／工具子树，前后工作区均记录干净；命令使用固定绝对路径 Go1.27.3 windows/amd64，环境 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`。记录中的 go.exe SHA-256 为 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`。本轮没有运行该程序，也没有把日志中的哈希当成历史执行的不可变认证。

独立从原始输出统计：

| 范围／命令 | 结果 |
|---|---|
| job 定向 `-run 'B152|B151|B148|B149' -count=1 -v` | 45 顶层、2 子测试 PASS，退出 0 |
| job 包全量 | 63 顶层、2 子测试 PASS，退出 0 |
| 工具模块 `-count=1 -v ./...` | 11 个有测试包 ok；206 顶层、273 子测试 PASS，0 FAIL，1 SKIP |
| 四包相关回归 `-count=20` | 69 个顶层名称各 20 次，1380 顶层与 40 子测试 PASS，0 FAIL |
| 已有静态检查日志 | gofmt 无输出；vet、带标签 vet、六配置交叉 vet、正式入口只编译、mod verify 均退出 0；第三方导入仍只有 dirhash |

唯一 SKIP 是 `TestTakeNegatives/symbolic_link` 的 Windows 权限不足；另外 controls 的 LC-2 符号链接准备失败也如实输出为本机 miss，不能用顶层测试 PASS 改写为该正式变体命中。命令记录中的正则竖线来自参数展示；读取 runtests.sh 可知实际调用使用 `"$G3" "$@"`，不是把它当 shell 管道执行。

这些是可读取、可关联的实施者既有证据，非审查者独立复现；本机运行不是三平台运行，交叉 vet／编译不是运行，20 次不是 -race 或全调度覆盖。缺少其他调度动态构造本身不阻止本次静态闭合结论。

### 6.3 失败出口与收尾审计探针

失败探针源码及 failprobe 导出副本一致。它在 newJob148FS 返回后、settle 注册前先登记观察 Cleanup；后登记的 settle 先运行，释放第一次 Write 和其他 gate，等待 WorkDone，再 endTasks。探针随后只读 gate、请求 done／Final、调用数、封口及后台终态。它不替 settle 释放或补发。

该次在持有中、短握手失败后主动 Fatal，退出 1 是预期。观察到 hold_released=true、请求 returned／consumed、calls=1、sealed=true、writer success/timely、diag complete-with-terminator/timely。它支持这个具体失败出口的收尾，不能计作正式全量失败，也不支持“全部失败路径均动态覆盖”。

收尾审计补丁实际位于 newJob148FS 返回前，早于测试正文的 settle／释放 Cleanup，晚于临时目录 Cleanup；按 LIFO，正文收尾先于探针观察、目录移除在后。读取了导出副本中的实际位置，未仅靠日志推断。探针在 done 关闭后才读原 Final，不调用 Finish，不追加唤醒。

独立统计 audit 日志得到 **58 个实例**：原 56 加本次两个顶层 Job；所有 request=returned、calls=1、work=done。其中 **48** 个 consumed／sealed=true，writer success/timely、diag complete-with-terminator/timely；**10** 个 expired-at-m.Run／未封口，writer expired-incomplete/late、diag=none。没有把无诊断登记算成一个已返回的诊断任务，也没有把终态算成观测到 goroutine 退出。

本轮所需材料均取得。历史 s192～s197 的结果及限制按 Review 152 原范围沿用，没有重新全量核验，也未用本次成功抵消旧失败。

## 7. 常规 CI 独立核验

读取 [run 43606425360](https://github.com/XKHoshizora/specmetry/actions/runs/43606425360) 正式元数据和八份实际作业日志：pull_request、attempt 1、success、8/8。

八份日志均实际检出 **`4aee4c6c705e233ed09e76c4c91028933a27dc41`**。GitHub Git commit 对象核对父提交依次为 `cd44f674f220116a19f82b2f8034a451d2ac642a`、`f9ad771cc9cc76de4361ed94027870db06895670`；tree 为 `73332d1b5d21ab4b33481ab35a2ad84fd5ec4f94`，与 HEAD 相同。没有只采用 run.head_sha。

| Job ID | 作业 | 回执 |
|---|---|---|
| 132481939611 | Go lint | Go1.27.4 linux/amd64；0 issues |
| 132481939693 | macOS Go tests | Go1.27.4 darwin/arm64；27 个 ok 包，7 条 SKIP；CLI 142.563s |
| 132481939718 | Windows Go tests | Go1.27.4 windows/amd64；27 个 ok 包，10 条 SKIP；CLI 375.527s |
| 132481939762 | Ubuntu Go tests | Go1.27.4 linux/amd64；27 个 ok 包，普通步骤 15 条 SKIP；CLI 558.913s；独立 root CrossUser 8 项 PASS |
| 132481939767 | Web | success；不安装 Go，不补写 Go 版本 |
| 132482180566 | Windows dashboard build | Go1.27.4；成功 |
| 132482180585 | macOS dashboard build | Go1.27.4；成功 |
| 132482180617 | Ubuntu dashboard build | Go1.27.4；成功 |

三平台日志未见测试 FAIL／panic 或作业 error。共同七项 SKIP 为五项受环境控制的真实／旧 Git 测试及两个 M-14 探针；Windows 另三个权限场景；Ubuntu 另八个普通用户 CrossUser SKIP，之后独立 root 步骤八项实际通过。Ubuntu 常规 -race 与独立 root 测试分开计，root 不增加普通包数。

常规 CI 的根模块不运行独立 `prototype/c9evidence` 模块，八份日志均没有该工具测试路径。七个 Go 作业的 Go1.27.4 不能替代本机固定 Go1.27.3 的工具证据。耗时只是观测，不是预算上界、根因证明或 M-16 关闭依据；历史失败和重跑事实不变。

## 8. 下一步、交接与结束边界

建议负责人采纳本轮限定技术裁定后，单独决定 PR #57 固定对象的合并及收尾。没有需要先修的 BLOCKER；关于“空闲不等于观测到阻塞”的解释性限定见 §3.4，不要求修改正式工具或增加实验。

交接提示词：

> Review 153 对 PR #57 的固定 HEAD f9ad771cc9cc76de4361ed94027870db06895670 为限定技术 GO：B-152-1 关闭、N-152-1 完成；B-151-1、N-151-1 和原限定 N-150-1 保持关闭。请先由负责人决定是否接受该裁定及另行授权合并／归档收尾；未经决定，不修改受审对象、不合并、不运行新的验证。后续材料继续区分静态握手论证、本机 Go1.27.3 记录、导出探针与根模块 Go1.27.4 CI。第二阶段、E-10(ii)、LE-5①／②、N2 仍待决，不能由本轮 GO 推导真实调用或正式取证权限。

本轮只读文档、源码、Git、哈希与已有日志，只新增本报告；没有运行 Go、测试、构建、实验或采集器，没有触发／重跑 CI，没有修改 PR、提交、推送或合并。真实调用、伪工具、平台控制协议、LC-9、正式取证、第三批生产实施及 WP2.5／M2 验收均未因此获授权。

2026-11-15 03:23（Asia/Tokyo）结束身份复核：本地与远程工作分支仍为上述固定 HEAD，PR head/base 与远程 master 一致，PR open、未合并；内容树和工具子树未变。报告写入后核实工作区仅新增未跟踪的 `docs/reviews/codex-review-153.md`，受审文件无修改；报告内相对链接检查无缺失。保持未提交，完成后暂停。
