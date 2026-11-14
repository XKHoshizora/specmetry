# Review 152：PR #57 的单次 Finish 请求修复定向复审

## 1. 裁定

**B-151-1 关闭；N-151-1 完成。N-150-1 原来限定的 37 项作业层回归／49 个实例，其覆盖描述及测试生命周期修复可以在该限定范围内关闭。**

**当前 PR #57 固定对象仍为技术 NO-GO，暂不建议合并：新增到期对照存在一项独立的测试阻塞 B-152-1。** 它没有先确定写者已处理完原记录并进入等待，就在推进到 D 后要求写者“尚无终态”。合法执行中，尚有待处理记录的写者可以先发布到期终态，该测试会误失败。不能用本机重复通过排除此事件顺序。

这两个裁定不矛盾：单次请求机制已修复 Review 151 的重复 Finish／丢失原结果问题；本轮新增回归的前提构造另有问题。没有因此重开 Review 150 已关闭的工具实现问题，也没有发现正式工具判定的直接回退。

| 审查项 | 裁定 |
|---|---|
| B-151-1 | 关闭：登记、单次发起、超时保留、重复读取、返回后消费已接入全部受审调用者 |
| N-151-1 | 完成：26 个实例、Started 观测边界、终态与无诊断的区分已精化 |
| N-150-1 | 原 37 项／49 实例的限定处理内容可关闭；不扩大为所有旧测试或本轮新增测试均无缺陷 |
| 新增 BLOCKER | B-152-1：新增到期对照依赖写者已经等待的未建立前提 |
| 非阻塞精化 | N-152-1：进入握手及“释放前尚未返回”的动态证据措辞应更准确 |
| PR #57 | 当前 NO-GO，先最小修正新增回归；不要求改工具实现、预算或终判协议 |

本轮依据为 [Review 151 全文](codex-review-151.md)、[Review 150 的限定裁定](codex-review-150.md)、[处理记录 §197](../review-resolution.md)、[实现记录 §11／§12](../wp2.5-c9-evidence-tool.md)，以及实际源码、提交差异、既有补丁和日志。未运行 Go、测试、构建、采集器或任何实验。

## 2. 固定对象、归档与授权范围

### 2.1 开始身份

2026-11-14 23:15（Asia/Tokyo）开始核对，本地工作区干净，目标报告不存在。GitHub 只读查询的 PR head/base、远程工作分支及 master 与本地固定对象一致。

| 项目 | 核验值 |
|---|---|
| 仓库／PR | `XKHoshizora/specmetry`，[PR #57](https://github.com/XKHoshizora/specmetry/pull/57)，open、未合并 |
| 分支 | `test/wp2.5-c9-evidence-n150-1` |
| HEAD | `5d5fd53c274f7412e9774d7f33d3a3bc2d721b02` |
| HEAD tree | `000ec5344e180e6e2da4bdba6244459e044a1af8` |
| 工具模块 tree | `3985b9929a75d8fa055ce524ed834fad62c63dc4` |
| base／远程 master | `cd44f674f220116a19f82b2f8034a451d2ac642a` |
| 上轮对象 | `db220ce3ad7becc2e50c749a6681ab9db17eee9c` |

### 2.2 归档、登记与差异

提交顺序为：

1. `62af4bef87984cc430869aed24929ec3d8d86508`：单独归档 Review 151。
2. `8e52b8f6adc72ed3574a9f88d3a30e2cf26a7d1f`：纯文档登记 §197。
3. `ac0e2beb4300f6b4f9ab312542f39dd85e72a0c0`：测试修复与新回归。
4. `5d5fd53…`：实现记录及状态同步。

Review 151 文件为 **25,842 字节**，SHA-256 `640eb3d0d2c4bb14992f811e8cdbe9f8260fdc04e19f202aeb28bab957ace5b5`，与上一轮交付一致，归档提交只新增该报告。§197 在修复提交之前登记；旧处理记录 1,392,663 字节仍是当前 1,403,691 字节文件的逐字节前缀，新增 11,028 字节，没有改写 §196 或此前历史。

上轮 HEAD → 当前 HEAD 共 14 个文件。工具模块的实际变化仅为五份测试文件：新增 `finish_test.go`、`regress151_test.go`，修改 `regress147b_test.go`、`regress148_test.go`、`tasks_test.go`；其余为归档、登记、记录及状态同步。`ac0e2be` 和最终 HEAD 的工具子树完全相同。

工具非测试实现、正式入口、根模块、依赖、workflow、bound、W/D、120 秒预算和正式终判规则没有变化。冻结正文与验收要求也没有变化。当前受审差异的 `git diff --check` 无输出。状态仍写待 Review 152，没有提前宣布关闭或新增执行授权。

## 3. B-151-1：单次请求机制与完整接线

以下源码位置均指固定 HEAD `5d5fd53…`；仓库链接使用相对路径。

### 3.1 登记与调用唯一性成立

[finish_test.go](../../prototype/c9evidence/internal/job/finish_test.go) 第 23–33、66–90 行：请求表以 `*Job` 为键，在同一互斥锁下查找并创建请求，然后解锁、启动 goroutine。登记发生在 goroutine 发起之前。两个调用者并发取得同一 Job 时，只有第一个能建立请求，后者取得原指针；不能各启动一个 Finish。`endTasks` 中先 `requestOf` 再 `requestFinish` 的间隔也不会重复创建，因为后者再次在锁下查表。

实际顺序为：登记请求 → 启动 goroutine → 锁外执行可选测试钩子 → 计数 → 调用 `j.Finish(0)` → 取得原始 Final → 写入请求的 final → 关闭 done。全局请求表不删除 Job 键，因而没有旧地址被复用为新 Job 身份的问题；它也确实持续持有请求和结果直至测试进程结束。这里是有意的测试内保留，不宣称这些对象已被回收。不同 Job 的表项不互相消费或覆盖。

`finishReq.wait`（第 94–107 行）在收到已关闭的 done 后读取 final；通道关闭的同步关系保证此前的写入可见。没有在 done 关闭前读取未发布 Final 的正式 helper 分支。通知采用 close 而非发送一个可被抢走的 Final，多个等待者可以读到同一结果。原 Final 内的原因切片在本次所有消费者中只读，没有新增并发修改路径。

超时只返回等待失败，不删除请求、不启动替代调用、不清空 final；之后仍可等待同一 done 并取得原始结果。若计时器与 done 同时就绪，某次等待可能选择超时，但只会明确记等待失败并保留请求，不能产生第二次 Finish 或恢复成伪成功；这不是正式 W/D 资格的判定点。

### 3.2 所有受审消费者已接入

| 消费者 | 实际路径与裁定 |
|---|---|
| `finish`（regress147b_test.go:44–51） | 改为 tryFinish；失败保留原来的测试错误，不再丢弃后台请求 |
| `finishWant` | 仍消费 finish 返回的原 Final，与独立指定的结论、退出码、出口比较 |
| 两处原手写异步调用（regress148_test.go:350–363、392–404） | 第一次等待失败后记错误、释放原核验阻塞，再等同一请求；第二次超时继续记失败；没有重新调用或 toStruct 中转 goroutine |
| `endTasks` | 取得同一请求；确实从未发起才补发；先等待原调用返回，再检查原 Final.Path |
| `settle`／Cleanup | 保持释放原阻塞 → 有界等待 WorkDone → endTasks 的顺序；不会依据 Started 重启收尾 |

按实际引用检索，没有受 endTasks／settle 管理的 Job 继续绕过请求表直接调用 `j.Finish`。仍直接调用的 `regress147_test.go` 一处、`regress147b_test.go` 四处、`job_test.go` 的 run，都不对同一 Job 使用本轮 endTasks／settle；不能把这些旧测试写成已经纳入本次生命周期整改。

共享 finish helper 也服务于部分 Review 147 测试。这些调用者每个 Job 原本只调用一次，参数仍为 0，原成功／失败判定未变；现在等待失败后有持续持有，不再丢失请求。检查了这一直接影响，没有发现语义回退，但不据此宣布全部旧测试的任务生命周期已审完。

### 3.3 原调用返回前后已正确分开

[tasks_test.go](../../prototype/c9evidence/internal/job/tasks_test.go) 第 73–111 行先等待原请求。等待失败时记错误并直接返回快照，没有用暂态决定封口、latch 或是否唤醒，也不宣称任务已结束。请求仍在表中。原调用后来返回，保存的仍是该次 Final，不是二次调用被拒绝的替代结果。

返回后按原 Path 分类：

- consumed／expired-final：必须已经封口，只等待已有写者与诊断终态。
- expired-at-m.Run／expired-at-snapshot：必须未封口且 latch 已置；写者还没有终态才追加测试唤醒项。
- 其他出口：明确失败，不把 none／配置缺陷当成正常收尾。

测试唤醒发生在原 Final 返回之后。既有写者下一次准入消费原 D／latch，拒绝外部写入；该项仅在内存，不进入 artifact，不改善原结论或清除真实错误。已经有到期终态的写者不需要再次唤醒。没有诊断登记仍为 none；登记后无终态为空，后者不能被视为结束。

这修复了 Review 151 所列的精确窗口：即使 finalized 已置位而 Started 未发布，测试请求仍已登记、done 未关闭。所有消费者只等待这个请求；没有任何动作再依赖这两个内部字段来授权第二次调用。该判断来自源码及所有调用点的接线，不要求动态命中两个锁之间的那一瞬间。

**B-151-1 在受审测试范围内关闭。** 正式工具的防重、W/D 与终判规则没有因本次修复改变。

## 4. 新回归的成立部分、限制及新增问题

### 4.1 六项测试、七个 Job 的证据

[regress151_test.go](../../prototype/c9evidence/internal/job/regress151_test.go) 新增六个顶层测试，最后一个含两个子测试，共七个 Job。

| 场景 | 可采结论与限制 |
|---|---|
| RegisteredNotYetCalled | 钩子阻塞首次调用，进入通知可观察；第一次 200ms 等待不能取得结果，调用数在持有时为 0。释放后实际调用 1 次、原 Final 及终态正确 |
| CallEnteredBeforeStarted | 测试持有 j.mu，且钩子到达；释放前 Finish 不能越过该锁。**不是** finalized 已置位／Started 未发布的动态覆盖；钩子本身在实际函数调用之前，不能单凭 entered 精确证明 CPU 已进入函数体，见 N-152-1 |
| StartedButNotReturned | verifier 的进入握手、Started 的有界观察和未封口断言建立了这个场景；释放后同一请求取得原 Final |
| LateReturnKeepsTheOriginalFinal | 第一次等待失败，释放后第二次等待读取原 Final；没有新增调用。这里“迟到”相对测试等待，不代表正式 W/D 资格晚到 |
| NeverIssuedCleanupIssuesOnce | 没有先发请求，endTasks 补发；再次 endTasks 不重复，之后仍能读取原结果。是直接构造失败清理状态，不是本轮又执行了一次 t.Fatal 探针 |
| normal／expiry before the seal | 正常对照成立；到期对照最终状态的既有日志可采，但前置“写者仍等待”的构造存在 B-152-1 |

前三项的短等待由钩子、持有的 Job 锁或受控 verifier 保证原调用不能完成，不是要求它必须恰好运行 200ms。它们均有释放函数，失败 Cleanup 也能释放；锁释放使用 sync.Once，且晚登记的 unlock Cleanup 在 settle 之前执行。其他等待有界。原“父端先返回”的 Review 148 断言仍在释放之前，没有因请求管理改为先释放再验证。

两处原异步调用的实际超时分支、endTasks 自身等待超过 bound 的分支，本轮既有日志没有动态执行。静态路径已完整：前者保留请求、释放、重复等待；后者错误返回且不再解释封口状态。LateReturn 的同类序列提供部分动态支持，剩余是静态接线证据。没有发现这些分支中尚未支持的动作或结果遗失，**不因这项诚实的覆盖限制另设阻塞**。

### 4.2 B-152-1：到期对照把合法的提前到期终态判成失败

**BLOCKER，P2，仅新增测试。**

位置：固定 HEAD 的 `prototype/c9evidence/internal/job/regress151_test.go:261–270`，尤其第 264–268 行；相关说明为实现记录 §12.2 的到期对照行。

测试执行：

```go
j.RunDomains()
clk.Set(j.D)
finishWant(t, j, chain.Incomplete, chain.PathExpiredAtMRun)
if w, d := taskStates(j.outer.Snapshot()); w != "" || d != diagNone {
    t.Fatalf(...)
}
```

它要求推进到 D 并完成 Finish 后，写者仍然没有终态。然而 `RunDomains`（`job.go:291–324`）只等工作任务的 done 或 W，不等待记录写者消费完日志。工作任务在 `job.go:333–344` 完成核验后即可发布 done；记录仍可能仅在内存中，或者写者正在处理它们。`Begin` 在 `job.go:250–257` 独立启动写者，二者没有“RunDomains 返回意味着写者已空闲”的同步关系。

**合法事件顺序：**

1. 工作任务按时完成，RunDomains 返回；写者尚有一批待写记录，暂停在下一次 admit 之前。更简单的情形是写者 goroutine 尚未开始运行。
2. 测试把假时钟设为 D，单次 Finish 正确返回 Incomplete／expired-at-m.Run，并置到期锁存。
3. 写者恢复，在 `internal/chain/writer.go:288–300` 的 admit 读到 D／latch，正确发布 `expired-incomplete/late`，不再启动文件调用。
4. 测试第 267 行取得这个合法终态，因为 w 非空而 Fatal。随后 Cleanup 可以正常结束，但整个测试被错误判失败。

也可能在 Finish 之前，写者已因 D 发布同一终态；同样合法且触发误失败。所需只是两条普通 goroutine 的允许交错，不要求无限停顿、系统调用不可返回或硬实时边界反例。

**违反的要求：** §197 要求由受控接点建立回归前提、不依赖快速调度；本轮要求检查短等待及握手是否会误判。同时 N-151-1 刚澄清“封口前到期不意味着写者必然还在等待”。这个新测试恰好在没有建立等待前提的情况下将该状态写成必须条件。

**影响：** 正确的工具与正确的 helper 可以被新增测试随机拒绝；当前 20 次通过不能证明不存在这个序列。这不是工具终判或实际文件释放错误，也不是要求扩大正式机制实验。它直接影响本 PR 新回归的可靠性，因此阻止当前固定对象合并。

**最小处理范围：** 仅修该到期对照及必要测试辅助。若要证明“helper 唤醒仍在等待的写者”，在推进到 D 之前，必须有有界、可核对的握手／记录消费完成条件，证明写者已处理完固定的全部现有记录，并且直到 helper 唤醒之前没有后续外部写入准入或新的记录可使其自行推进。可以利用现有测试设施与公开的写入发布状态确定此条件，不要求改正式工具或延长 bound。

不能简单删除第 267 行、增加 sleep，或接受“写者也许早已结束”后仍声称这个用例证明了主动唤醒。如果另把“已经到期结束”列为健康分支，应分别说明它只证明无需唤醒的收尾，并保留一个确实建立了等待前提的唤醒对照。不放宽原 W/D，也不改变工具应当拒绝到期新操作的规则。

### 4.3 N-152-1：动态命中范围的非阻塞精化

`goEndTasks` 启动 goroutine 后，`notReturned`（第 47–55 行）立即作一次带 default 的通道检查，再释放原阻塞。它能证明**那个检查时刻没有取得 endTasks 返回值**，但没有 endTasks 已进入等待的握手；该 goroutine 可能到释放之后才运行。因此这不是确定命中了“endTasks 已在未返回请求上等待”的动态证明。建议将说明收窄，或在测试侧补必要握手；不据此声称当前正确 helper 有假通过缺陷。

同样，CallEnteredBeforeStarted 的 entered 在调用前钩子中关闭，早于计数与 `j.Finish` 调用。它证明钩子已到达、Job 锁被持有，不能严格把它写成已观察到函数第一条语句正在等锁。实际机制不依赖这个精确位置，源码已经保证同一未完成请求不会被重复发起；这项措辞／覆盖精化不阻止 B-151-1 关闭。精确的 finalized/Started 窗口未动态覆盖，文档的这一限制保留正确。

终态仍只表示最后状态已发布。done 则证明该次 Finish 已返回、原 Final 已发布；不能把这两种不同通知合并为所有 goroutine 都已完成退出调度。

## 5. N-151-1、历史计数与新探针

§11.1 已把旧封口组改为 **16 项、26 个实例**；全组仍为 37 项、49 个实例，分组没有静默删除。旧 16 项未调用 Finish 中，两项写者已经结束；旧调用过 Finish 的实例中也有两个仍等待写者。不能由其中一个布尔事实推导全部任务是否结束。

旧探针的补注现在明确读的是 Started，不能当成实际发起次数或返回证明；正常记录的历史结论同时依赖当时测试源码及无等待失败的执行结果。修改后 49 例准确拆为：49 个写者有终态、42 个已登记诊断有终态、七个没有诊断登记。测试注释也明确终态不是 goroutine join。**N-151-1 完成。**

本轮新的 56 例来自旧 49 例加六个新顶层测试所创建的七个 Job。独立重新统计 `audit-probe-after-fix.log`，确为：

- 56 条 Job 记录均 request=returned、calls=1。
- 48 条 consumed／sealed=true，writer success/timely、diag complete-with-terminator/timely。
- 八条 expired-at-m.Run／sealed=false，writer expired-incomplete/late、diag=none。

新增七例为六个封口、一个封口前到期，所以从原 42／7 得到 48／8。它们是该次运行的观察，不保证新到期对照在所有合法交错中都能正确构造，不能抵消 B-152-1。

N-150-1 的原处理内容已经有覆盖更正、原 16 项最终判定和原 37 项生命周期消费，B-151-1 的失败出口管理也已修复，因此可在原限定范围关闭。新 B-152-1 单独登记并阻止 PR 合并，不把所有历史已关闭项重新设为未完成。

## 6. 原始材料、验证记录与边界

### 6.1 取得与代码关联

读取用户指定 `s197` 下现有日志、`pre-fix-seams.diff`、前置 helper／回归副本、`fix.diff`、`probe_patch.py`、`probe.diff`、导出副本中的探针源码及日志脚本。没有执行这些脚本或任何被审程序。

独立核对以下材料大小与 SHA-256，均与实现记录一致：

| 材料 | 字节 | SHA-256 |
|---|---:|---|
| pre-fix-seams.diff | 1606 | `668568a64a6021077d5d76470cf2fff8d2be56b883b3367d68b8dfb4ded4c623` |
| pre-fix-finish_test.go | 1434 | `5376d93ac89c2ff38eee007c6f3190d7db6d05d7246fd4cdec38b90b3d626223` |
| pre-fix-regress151_test.go | 9111 | `a28e76ba8853aed8abdacb73123b977e812226f0736cf441f36a180bc0abab0d` |
| fix.diff | 21582 | `767e3536ffb1c1f91d7a70b8cc8f4326cbb996afa2df27efd26d59fbcdf5652d` |
| probe_patch.py | 1780 | `63a17b0dfb624c0ec2560fd246aba4c6b1dc67e0be2ef92412a7f91948b03a7f` |
| probe.diff | 749 | `d1e74e005f64de06a898c92219cf19a57830d7f0bb60f2a9396a9c5d25193ce3` |

修复前是 `8e52b8f` 的旧工具子树，加接缝及先写定回归，**不是无修改的旧提交**。补丁把原调用路由至可计数、可挂钩的调用点；tryFinish 仍每次创建新 goroutine／结果通道，等待超时仍丢失返回通道。无钩子时保留旧语义，没有提前实现请求登记或结果缓存；有钩子的回归有意改变调度前提。因此适合作为旧逻辑的带接缝对照，不能说它与原来执行逐时刻相同。

修复后正式日志指向 `ac0e2beb4300f6b4f9ab312542f39dd85e72a0c0`，tree `f0218b3d6eff6ecb3a140895f7b1ebac5bd73722`，测试前后工作区干净；其工具子树与最终 HEAD 同为 `3985b9929a75d8fa055ce524ed834fad62c63dc4`。

### 6.2 原始日志核对

| 日志 | 字节 | SHA-256 |
|---|---:|---|
| pre-fix-regressions.log | 4730 | `a897a9dc31d95a37bf48075389e7445f6abfd86da218ee1c862dc3cb99703f80` |
| post-fix-targeted.log | 5611 | `2cabae2d41ecd4b0f0899ecfb0deaa388320f2d33a0f964f621b3292a3954d3d` |
| post-fix-job.log | 7067 | `15bf61501342993617d7f49cc8a11d7b73df7309812c2d7d96aa7c85ae6e2fd2` |
| post-fix-module.log | 47902 | `e4e9a6239d0a2cbfe8202e5b664197bab8fcf381b7e8404b78db9a13d39ace51` |
| post-fix-x20.log | 140365 | `431defa0880bea926cee4ce375776f21827214b6a45c4b637b8be1aa41a4f897` |
| post-fix-static.log | 2699 | `736ac6179d1e659d4481b98c3b40cc28a2f9fc998d92a83ea2b1e689e53586c3` |
| audit-probe-after-fix.log | 16672 | `9454c0535be5a3575f2d0ccaf55bfd124b981c678dbf7bf0bb3cf6ba01da5e12` |
| toolchain-redownload.log | 1027 | `3682414754b7da9b9b93c2799bec65d91ddc2e9b11b8f40d31a40309fa0647b3` |
| xmod-redownload.log | 1747 | `f9142361c6faca3c0230e08270f240c85e9547fe0fb3ab494d3efa4a105dfc75` |

修复前 `go test -count=1 -v -run B151 ./internal/job/` 退出 1，五项失败，一项顶层及其两个子测试通过。须准确解释失败：

- 未返回状态被旧 endTasks 当成返回后的“未封口且无锁存”，是原管理缺陷的实际表现。
- 旧 helper 第二次等结果会重新调用，是本次要修的问题；但回归 `originalFinal` 本身也会调用旧 tryFinish。因此最后的 `Finish called twice` 和计数不能全部归给 Cleanup。
- `NeverIssuedCleanupIssuesOnce` 的旧清理没有重复调用；它失败在回归为取原结果再次调用旧 helper。证明的是结果不保留，不是“从未发起的 Cleanup 也会多发一次”。
- 窗口的精确先后仍以实际可观察状态和源码为限，不由五项失败推导成五种不同生产缺陷。N-152-1 所述握手限制也适用于这些旧逻辑对照。

修复后日志的命令、退出状态及独立统计为：

| 范围 | 结果 |
|---|---|
| B151／B148／B149 定向作业测试 | 43 个顶层、两个子测试 PASS，0 FAIL，退出 0 |
| job 包 | 61 个顶层、两个子测试 PASS，0 FAIL，退出 0 |
| 工具模块全量 | 11 个有测试包 ok；204 个顶层、273 个子测试 PASS，0 FAIL，1 SKIP |
| 四包相关回归重复 20 次 | 67 个顶层名称，每项 20 次，共 1340 个顶层 PASS；另 40 个子测试 PASS，0 FAIL |
| 静态记录 | 格式、vet、带标签 vet、六配置交叉 vet、正式入口只编译、mod verify、导入闭包均记录成功 |

SKIP 仍为 `TestTakeNegatives/symbolic_link` 的本机权限前提不足，不能写成该变体运行通过。第三方导入闭包仍只有 `golang.org/x/mod/sumdb/dirhash`。日志脚本执行参数使用 `"$G3" "$@"`，展示的正则竖线不是运行时 shell 管道。

这是本机 Windows/amd64、Go1.27.3 的既有测试证据，不是三平台运行；交叉 vet／只编译不等于运行；20 次不是 -race 或全调度覆盖。关联到代码、脚本和哈希不等于不可变产物认证，也不是审查者重新运行。

### 6.3 导出探针与 Cleanup 顺序

实际导出副本的探针位于 `newJob148` 返回前，与 `probe.diff`、补丁脚本一致。它早于测试正文登记 settle／释放 Cleanup，晚于临时目录的 Cleanup；Go Cleanup 后进先出的既定语义使释放与 settle 先执行、探针后观察、目录清理最后执行。本轮检查了具体登记位置，未仅凭日志声称顺序正确。

探针只读请求表：无请求、done 未关闭、done 已关闭分别记录 none／PENDING／returned；只有 done 关闭分支才读取 `r.final.Path`。此外读取调用计数、封口、latch、工作状态、写者和诊断终态。它不调用 Finish、不释放阻塞、不制造唤醒。比旧 Started 探针更直接，但所记录的仍是各状态的顺序读取，不是不可变的全进程原子快照。该次 56 个请求都返回、调用数都为 1，记录支持上述限定结论。

### 6.4 工具链和依赖恢复

下载日志记载：先在空目录由 shim 对应的 Go1.27.1 发起固定 Go1.27.3 的工具链恢复，随后使用恢复后的绝对路径；x/mod v0.42.0 的下载 h1 与工具模块 go.sum 一致，后续 GOPROXY=off 的 mod verify 成功。go.mod／go.sum 未改。

本轮只读核对当前 go.exe 的 SHA-256 为 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`，与下载及测试日志一致；没有运行该程序。下载记录列出的工具链 zip 为 76,890,515 字节，ziphash `h1:Ke8xvH3G94wJ4SfCeyCpVAjhpTpJv+iU1Nz6+W97TD4=`；x/mod h1 为 `h1:ICeL3t2L30ljSPHHxDnR4fqVcBt/zR2zJAb9juavmQg=`。

测试日志统一记录后续隔离环境 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`。可确认现有记录与选定工具链相符，不能把当前哈希扩张成历史执行认证。缓存被哪个进程删除、会话批准下载的完整原始对话没有从这些文件独立证明，仍按作者的环境说明保留；不因恢复下载自动推断工具链改变或越权行为。

## 7. 常规 CI 的独立核验

读取 [run 43602103840](https://github.com/XKHoshizora/specmetry/actions/runs/43602103840) 元数据和八份实际 job 日志：pull_request、attempt 1、正式 success、8/8。

八份日志实际 checkout 都为 **`5aad6de31d1e50ecc288f9e47898cd0adaff9b76`**。GitHub commit 对象的父提交依次为固定 base `cd44f674…` 和固定 HEAD `5d5fd53…`，tree `000ec5344e180e6e2da4bdba6244459e044a1af8` 与 HEAD 相同，不仅采用 run.head_sha。

| Job ID | 作业 | 实际回执 |
|---|---|---|
| 132468866065 | Web | success；未安装 Go，不补写 Go 版本 |
| 132468866170 | Ubuntu Go tests | go1.27.4 linux/amd64；27 个 ok 包，普通步骤 15 条 SKIP；独立 root CrossUser 八项 PASS |
| 132468866223 | Go lint | go1.27.4 linux/amd64；0 issues |
| 132468866253 | macOS Go tests | go1.27.4 darwin/arm64；27 个 ok 包，7 条 SKIP |
| 132468866255 | Windows Go tests | go1.27.4 windows/amd64；27 个 ok 包，10 条 SKIP |
| 132469096500 | Windows dashboard build | go1.27.4；构建及 embedded 回执成功 |
| 132469096521 | Ubuntu dashboard build | go1.27.4；构建及 embedded 回执成功 |
| 132469096592 | macOS dashboard build | go1.27.4；构建及 embedded 回执成功 |

未见测试 FAIL 或作业 error。七个共同 SKIP 是五项受环境控制的真实 Git／旧 Git 测试与两个 M-14 探针；Ubuntu 另有普通权限下的八个 CrossUser SKIP，随后 root 步骤确实执行；Windows 另有三个不可读文件／存储的权限场景。Ubuntu 普通测试的 -race 与独立 root 步骤分开，不能把 root 八项写成另一个包或三平台证据。

CLI 耗时为 Ubuntu 552.238s、macOS 196.169s、Windows 372.746s，只是观测，不推断上界、根因或维护事项已解决。

常规 CI 运行根模块，不运行本次修改的独立工具模块；七个 Go 作业的 Go1.27.4 不替代本机 Go1.27.3 的证据。根模块成功与 race 不能证明本次测试 helper 正确，也不能反驳 B-152-1。历史 CI 与失败回执按已有核验范围保留，没有把当前运行说成旧运行重跑，也未重新全量核验历史日志。

## 8. 最小下一步与授权边界

建议保留 B-151-1、N-151-1 及原限定 N-150-1 的关闭裁定，仅定向处理 B-152-1 的到期回归构造，并精化 N-152-1 的动态命中表述。无需更改单次请求机制的核心、正式工具、原期限或重新审查平台协议。

当前不建议合并 PR #57。修正后的技术复审、合并决定与任何后续授权继续分开；本报告没有执行修复或建议自动运行任何新实验。

Review 150 已关闭的第一阶段工具实现问题保持原限定结论。E-10(ii)、LE-5①／②、N2 与第二阶段实施仍待决；真实调用、伪工具、平台控制协议、LC-9、正式取证、第三批生产实施及 WP2.5／M2 验收都未因本轮获得授权。没有采纳提案、接受风险、批准精确条目或宣布 G 项满足。

## 9. 实际操作与结束核验

本轮只读取代码、文档、Git 对象、固定记录、哈希和已有 CI；没有运行 Go、测试、构建、实验或采集器，没有操作 CI、PR、提交、推送或合并。仅新增本报告，保持未提交。

2026-11-14 23:25（Asia/Tokyo）结束复核：本地分支、HEAD、内容树及工具模块子树仍为 §2.1 的固定值；远程工作分支与 PR head 均为 `5d5fd53c274f7412e9774d7f33d3a3bc2d721b02`，远程 master 与 PR base 均为 `cd44f674f220116a19f82b2f8034a451d2ac642a`，PR 仍 open、未合并。工作区仅有未跟踪的 `docs/reviews/codex-review-152.md`，受审文件没有修改，无身份漂移。完成后暂停，等待负责人决定。
