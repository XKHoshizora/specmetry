# Review 151：PR #57 的 N-150-1 定向复审

## 1. 裁定

**固定对象 `db220ce3ad7becc2e50c749a6681ab9db17eee9c`：技术 NO-GO，暂不建议合并。**

本轮发现一项测试收尾阻塞 **B-151-1**：新增 `endTasks` 以 `Closeout.Started` 判断是否可以补跑 `Finish`，不能区分“从未发起”与“已经发起，但内部 Started 尚未发布”。既有 `finish` 等待超时后丢失该次调用的完成通道，新 helper 可以再次调用 `Finish`。Started 已发布也不能证明原调用已经返回。本轮要求覆盖的失败出口因而没有完整收口。

这不是工具终判的新缺陷：`Job.Finish` 的 `finalized` 防重仍在，不能据此声称会有两个内层 `chain.Closeout` 同时执行，也没有本轮发现的生产假通过、真实句柄或子进程泄漏。问题是测试生命周期的单次调用、完成消费及失败收尾承诺没有兑现。不能因为原 N-150-1 是非阻塞项，就把这次新 helper 的具体错误当作已经完成的修复。

| 本轮问题 | 裁定 |
|---|---|
| N-150-1 的历史覆盖描述 | 主要更正成立；§11.1 有一处非阻塞计数错误 N-151-1，27 个应为 26 个实例 |
| 正常走完的 37 项测试、49 个实例 | 新增最终断言及写者／诊断终态核对有源码与既有日志支持，按下文限定采证 |
| 失败出口生命周期 | 尚有 B-151-1，不能声称所有受审出口都能不重复地收尾 |
| N-150-1 整项 | **在本次限定范围内尚不能关闭** |
| Review 150 已关闭的实现阻塞 | 保持原限定关闭；本轮未发现工具实现的直接回退 |
| PR #57 合并建议 | 先在测试范围修正 B-151-1，再定向复审；不要求修改正式工具协议 |

依据为 [Review 150](codex-review-150.md)、[处理记录 §196](../review-resolution.md)、[实现记录 §9.1、§10.2、§10.6、§11](../wp2.5-c9-evidence-tool.md)、实际测试及其消费链。第二阶段说明仍是待决材料，本报告不确认 E-10(ii)，不授权 LE-5①／②、真实调用、平台控制协议、LC-9、正式取证或第三批生产实施。

## 2. 固定身份、范围与登记

### 2.1 开始核验

本地工作区开始时干净，报告目标不存在。核验得到：

| 项目 | 实际值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry`；origin 为 `https://github.com/XKHoshizora/specmetry.git` |
| 分支 | `test/wp2.5-c9-evidence-n150-1` |
| HEAD | `db220ce3ad7becc2e50c749a6681ab9db17eee9c` |
| HEAD tree | `db114b629057f04da5e68ee22b040205a990130e` |
| `prototype/c9evidence` tree | `d311efa4423ce294753eb7848799cf2ebd3f242d` |
| PR #57 head／base | 上述 HEAD／`cd44f674f220116a19f82b2f8034a451d2ac642a` |
| 远程工作分支／master | 分别为上述 HEAD／base；PR open、未合并 |

本地 Git 对象用于核对本地身份与差异；GitHub 只读接口用于核对远程 refs、PR 和运行记录。后续 shell 的 `git ls-remote` 遇到 DNS 解析失败，合并快照也不在本地对象库；没有 fetch、切换或补齐对象，改由可用的 GitHub 接口重新核对。不能把这个失败说成已经成功执行了远程 Git 命令，也没有据此推测身份漂移。

### 2.2 改动与授权顺序

base → HEAD 共 13 个文件，328 行新增、16 行删除。工具模块内只有以下五份测试文件变化：

- `internal/job/regress148_test.go`
- `internal/job/regress148b_test.go`
- `internal/job/regress149_test.go`
- `internal/job/regress149b_test.go`
- 新增 `internal/job/tasks_test.go`

其余八份为 AGENTS、实施计划、处理记录、工具记录及相关文档的状态同步。没有工具非测试实现、正式入口、依赖、workflow、预算或终判协议的改动。

登记提交 `758da73149cd7eb44194b52f293ee0d6a9d3405a` 先于测试提交 `3a3622274abc513d128a52e90d5a95c9925eb18c`；最终 `db220ce` 只补实现记录。`3a36222` 与最终 HEAD 的工具子树均为 `d311efa4423ce294753eb7848799cf2ebd3f242d`。§196 的要求包括“不重复调用已经执行过的 Finish”和失败出口也有界收尾，正是 B-151-1 的依据。

`review-resolution.md` 的 base 原始内容（1,386,275 字节）是当前文件的逐字节前缀；新增 6,388 字节为 §196，历史未回改。登记提交之后该文件没有再改。当前差异的 `git diff --check` 无输出。

Review 150 在 base 与 HEAD 间未改变：24,892 字节，SHA-256 `2b0ad33a6679e26f5bcd55034088218772e689a713e76beffbabab1eed8501ce`。

### 2.3 PR #56 收尾

GitHub 元数据确认 PR #56 已合并：最终 head `2b3e5b51b6c91dc27289828405e54bf125dea992`，merge `cd44f674f220116a19f82b2f8034a451d2ac642a`。合并父提交依次为 `9137279554deed5a542dbd61b871d1cfa79bc323`、上述最终 head，tree `35a89f8f6bcc3d3cf299a73fa6ca250ccb848072` 与最终 head 相同；工具子树保持 `95d92292b7807d211eab6fdbe6f7cdd6c5ab32f5`。

本轮还读取了最终 PR CI `43589182874` 与 master push CI `43590287451` 的正式元数据：attempt 1、success，分别关联最终 head 与上述 merge。没有重新全量读取这两次历史的全部日志；更细的历史执行结论仍按已有审查与收尾登记的范围引用，不冒称本轮重验。

## 3. 覆盖描述与计数

### 3.1 37 项与 49 个实例

独立检查四个回归文件、循环及前后探针原始日志：Review 148 的作业层为 26 项（15＋11），Review 149 为 11 项（7＋4），合计 **37 项**。

其中 `TestB148RbTimelyWorkParentLate` 有 10 个 Job，`TestB148RbWorkDoneAtOrAfterW` 有 2 个，`TestB149RealFailureKeptApartFromRefusal` 有 3 个；其余每项 1 个。所以实例总数为 `37 + 9 + 1 + 2 = 49`，不能将循环中的 Job 当作额外顶层测试。

旧版本的实际分组如下。下表的“调用 Finish”以源码及该次探针的 Started 状态共同确认，只用于这些已完成观察的实例，不把这个状态泛化为任意调用的返回证明。

| 旧状态 | 顶层测试数 | Job 实例数 | 写者／诊断状态 |
|---|---:|---:|---|
| 已 Finish、已封口 | 16 | **26** | success/timely；complete-with-terminator/timely |
| 已 Finish、封口前到期，之后记录已唤醒写者 | 3 | 3 | expired-incomplete/late；无诊断登记 |
| 已 Finish、封口前到期，之后没有记录变化 | 2 | 2 | 写者仍等待；无诊断登记 |
| 未 Finish、已到 D，写者被后续记录唤醒 | 2 | 2 | 写者已到 expired-incomplete/late；无诊断登记 |
| 未 Finish、未封口、未锁存 | 14 | 16 | 写者仍等待；无诊断登记 |
| 合计 | 37 | 49 | 旧审计中等待写者的为 16 项、18 个实例 |

因此，没有 Finish 的是 16 项、18 个实例，其中两项的写者已结束；已 Finish 的实例中也有两个写者仍等待。两种状态都不能仅凭是否调用 Finish 推导泄漏或完成。这里涉及内存文件系统和进程内任务，没有据此认定真实文件描述符、子进程泄漏。

§9.1 对原 Review 148 的 7 项、§10.2 对原 Review 149 的 9 项未调用 Finish 的更正成立。§10.6 保留旧时点的“待处理”，并交叉指向 §196／§11，没有把历史源码改写为当时已包含新增断言。Review 147 的作业回归与 `job_test.go` 不在本次生命周期审计内；本轮没有把它们纳入“均已收尾”或“均未收尾”的结论。

### 3.2 N-151-1：旧封口组实例数及概括措辞

位置：固定 HEAD 的 `docs/wp2.5-c9-evidence-tool.md:667`，§11.1 第一行写“16 项（27 个实例）”。应为 **26**：这 16 项里只有 10 次循环与 2 次循环增加实例，`16+9+1=26`。原始 `audit-probe.log` 也为 26；用 27 会使全表合计成为 50，与 49 冲突。这是可直接文字更正的非阻塞项，不推翻实际清单和日志。

另建议把 `tasks_test.go:14–15` 的概括限定为“未封口且仍等待记录变化的写者”；已有源码与记录说明，未 Finish 或封口前到期并不必然意味着写者还在等待。§11.2“所有实例写者与诊断都有终态”也应按紧接着的细分理解为“写者有终态，已登记的诊断有终态；另外七例没有诊断登记”，不应读成 49 例都创建了诊断任务。

## 4. Helper 与实际接线

源码位置均针对固定 HEAD；代码链接指向仓库相对路径。

### 4.1 已成立部分

[tasks_test.go](../../prototype/c9evidence/internal/job/tasks_test.go) 第 30–46 行确实区分：没有诊断登记为 `none`；登记过但没有终态为空串。`tasksEnded` 不会把后者当成完成。写者缺失或没有终态同样不能通过。

已知本次 Finish 已返回且日志已封口时，`endTasks` 不追加记录，只通过 `Outer.Watch` 等待已有写者与已登记诊断的终态，有 10 秒测试等待上界；超时记测试错误，不静默忽略。`wantTasks` 再比较由测试预先指定的状态和资格。

已知 Finish 已经从封口前到期出口返回时，helper 要求 latch 已置，才在内存日志追加明确标为测试用途的唤醒项。`Writer.admit`（`internal/chain/writer.go:284–300`）在下个决策点消费原 D／latch，发布 expired-incomplete/late 后拒绝外部写入。因而该项不会作为 artifact 内容写入，也不启动新的文件操作；不重取 W／D，不改善已返回的 Final，不清除原始错误。正式入口并没有因此获得自动唤醒写者的能力。

`settle`（`regress148_test.go:277–295`）的顺序是：运行原有释放函数 → 有界等待 WorkDone → 调用 endTasks。释放仍在原回归已经检查“父端先返回”之后，或在该测试的失败 Cleanup 内发生；没有把回归改为先释放再等待父端。WorkDone 超时会明确失败。新增正文断言位于原有的调用次数、真实错误、来源、拒绝、持有断言之后，没有删除或放宽它们。

终态证据的准确含义是**最后状态已发布**。检查写者与诊断终态发布后的源码，相关终态路径只剩解锁和返回，没有再安排外部 I/O 或新的等待；因此足以支持本轮检查这些任务不再进行实质工作的限定结论。但没有额外的 goroutine join，不能写成已经观测到运行时完成全部退出调度。这个边界本身不新增阻塞。

### 4.2 B-151-1：Started 不能替代 Finish 的发起与返回状态

**级别：BLOCKER（P2，测试生命周期范围）。**

直接修改位置：`internal/job/tasks_test.go:63–72`，特别是第 65–66 行；接点为 `regress148_test.go:283–295` 的 Cleanup。依赖源码：`regress147b_test.go:44–54` 的既有 `finish`，`job.go:771–779`，`internal/chain/closeout.go:220–235`。文档对应 §11.2 第 686–691 行以及 §196 的单次 Finish／失败收尾要求。

`endTasks` 的判断为：

```go
if !j.outer.Snapshot().Closeout.Started {
    finish(t, j)
}
```

但是 `Started` 在 `chain.Closeout` 内、参数校验和进入其步骤 1 后才发布。`Job.Finish` 先在另一把锁下置 `j.finalized=true`，再调用 `chain.Closeout`。而测试 `finish` 先启动 goroutine，等待至 `bound` 后只返回 `(zero,false)`，没有向新 helper 交付仍在执行的调用、完成通道或最终结果。

**可达的事件顺序：**

1. 某个新增 `finishWant` 经既有 `finish` 发起第一次 Finish。该调用已经执行 `job.go:777`，随后在 `chain.Closeout` 发布 Started 之前暂停。
2. 测试的 10 秒等待结束，`finish` 记错误并返回 false；`finishWant` 返回。这个超时不是取消第一调用，第一调用仍可能继续。
3. 测试正文的 endTasks，或稍后的 settle/Cleanup，读取到 Started=false，于是再次调用 Finish。
4. 第二次调用被 `j.finalized` 拒绝，返回 Invalid 和 `Finish called twice`；endTasks 没有消费这个返回值。第一调用此时仍可以没有封口、没有置 latch。
5. helper 在第 69–72 行把这个暂态判成“收尾留下未封口且无 latch”，记错误后返回；第一调用既没有被本 helper 等待完成，也没有保留在单次测试收尾管理中，其迟到 Final 无处消费。它之后仍可恢复执行。

另一种顺序是第一 goroutine 尚未进入 Finish，第二次先执行；无论谁先赢得 finalized，都已经违背“不重复发起／调用 Finish”的保证。这里不要求系统调用可中断或调度器按时唤醒；只要求在已有等待超时之后，不把未返回操作误认为从未发起。

**Started=true 也不是解决办法。** 它只说明进入过内层步骤 1。原调用仍可能尚未封口或返回。此时 endTasks 可能把过程中的“无封口、无 latch”当成返回后的异常；或者在写者有终态、暂时没有诊断登记时结束等待，而第一次 Finish 仍可能随后登记诊断。两项手写的异步 Finish（`regress148_test.go:350–361、391–402`）虽有第二次有界等待，其失败之后同样没有把“原调用仍未返回”的状态交给新 Cleanup。应统一管理这些调用，而非仅修一个 helper 的判断表达式。

**影响与范围：** 测试已经发生错误，不是证明一个健康场景会假通过；但本 PR 声称处理的失败收尾仍然可能重复调用、误判未完成状态，并在第一调用返回前结束本侧生命周期管理。真实工具的 finalized 防重仍有效，所以不能说会并发执行两个完整 Closeout。也不能从这条静态序列推断已经发生某次原生资源泄漏。

现有 failure-exit-probe 在第一次 Finish 之前主动 Fatal，正好验证的是“确实从未发起”的分支；它没有覆盖上述“已发起、等待超时、尚未返回”分支。20 次重复和正常审计日志不能补成这项覆盖。

**最小修正范围：仅测试辅助与相应定向回归。**

- 在启动 Finish goroutine **之前**登记本测试的单次收尾请求；持有其完成通道与原结果，明确区分未发起、已发起未返回、已返回。
- 所有受审调用点，包括两处手写异步调用，共用这份测试状态。Cleanup 只有在从未发起时才补跑；已有请求只等待同一请求，不按 Closeout.Started 重启。
- 原 Finish 已返回后，才应用“封口等待”或“封口前到期唤醒”的分类。等待失败要记失败并保留未完成责任，不把暂时无诊断等同于以后不会登记。
- 保持原父端返回顺序、W／D、断言和正式工具接口；用有进入握手、独立释放及有界等待的测试覆盖未发布 Started／已 Started 未返回的失败出口。不能靠延长 bound 或保证快速调度消除窗口。

不要求重做 Review 147 或 job_test.go 的全部生命周期审计；只是共享 helper 的变更及本次实际调用点必须兑现单次收尾。没有要求本轮现在执行或修复该问题。

## 5. 新增最终断言与正常收尾证据

原来没有 Finish 的 16 项均已新增最终判断，并保留原来的场景断言。固定预期按规则可独立推出，不是从实际 Final 反算：

| 场景（省略 TestB148／TestB149 前缀） | 新增预期及依据 |
|---|---|
| PublishedVerificationCoversEveryRecord、HealthyHoldsReleased、StepOpsHealthy、LC2ChangeSteps | 结论 4、退出 0、consumed；所需工作与相应健康条件成立 |
| LateSuccessIsNotTimely | 结论 2、非零、consumed；核验晚到不补回及时资格 |
| FailureIsNotARefusal | 结论 2、非零、consumed；根目录清理真实失败属于外层有效性问题 |
| BrokenRecordFoundByTheWork | 结论 1、非零、consumed；来源核验缺陷已被保存并消费 |
| MkdirAllCrossesW、WriteCrossesW、RemoveCrossesW、StepAtWIsRefusedCleanupAdmitted、RealFailureKeptApartFromRefusal（3 实例）、RefusedStepIsNamed、RemoveThenMkdirSteps | 结论 1、非零、consumed；准备不完整使必要变体未命中，不能仅以已有结果及时当成完成 |
| RcRefusedPreparationIsRecordedApart、RcRefusedCleanupKeepsTheHold | 结论 1、非零、expired-at-m.Run；原场景在 D 进入收尾，不推定成功 |

`finishWant` 使用传入的预期结论及固定 ExitCode 映射比较，不用被测实际结果生成预期。此 helper 本身的超时问题已单列 B-151-1，不能因此抹掉正常执行时已取得的判断证据。

全部 37 项有明确的写者／诊断状态比较。修改后审计的 49 个实例中，42 个封口，写者 success/timely、诊断 complete-with-terminator/timely；七个封口前到期，写者 expired-incomplete/late、没有诊断登记。

这些状态是记录链能否完成的证据，不是作业有效的替代条件。例如来源坏记录、准备拒绝或外层真实失败仍可以有成功关闭的记录链；上述最终结论仍是 1／2。没有把记录完整、终结行存在或 helper 主动唤醒写成原场景有效、正式工具自然结束或正式取证成立。

## 6. 现有本机证据的独立读取

### 6.1 取得范围及哈希

读取用户指定的 `s196/logs`，八份日志均可取得；独立核对字节数、SHA-256、命令记录、结果与身份。以下路径均相对于该日志目录：

| 文件 | 字节 | SHA-256 |
|---|---:|---|
| audit-probe.log | 12665 | `b049862583281c01520472c8c499baf1f1ca488a5f7d675eb86563674188d06c` |
| audit-probe-after.log | 13001 | `83d2dee04dcee9863e32b3fbf86a17bbaefdc5b798991e1d02d68bed4e817f32` |
| failure-exit-probe.log | 779 | `5be5d55ace031831a9db140f61fc3d10d0b8d1d75b39a9db3205cc0c7e256456` |
| targeted-test.log | 4702 | `ddfa323b27cd3d09d432d218057df78635175df9e664f18191990f01530a1e07` |
| affected-package.log | 6163 | `5b1be3b14f3fa64b567d73ac9138c4f78f52dc9c295b95bb945dd2c371520188` |
| full-test.log | 46995 | `91eb6f884f411c61a84db24afe03bcac3f2fedcb3399ea6e529c2c87bf70b614` |
| regress147-149-x20.log | 122284 | `9103210885c3739337a42546802ef47ebb7f9a9bbfc33fe4112fc31788f66cc8` |
| static.log | 2699 | `25a46df11c700c8733241c545f73fe5bee6f4a9ff39b59656a51de9c1fe4d248` |

日志支持所记录的本机 Windows/amd64、Go1.27.3、代码及工作区关联。正式五份测试／静态日志记载提交 `3a36222`，测试前后干净，其工具子树已与最终 HEAD 独立比对一致。日志所记环境隔离、工具哈希与命令可供追溯；这不是对当时执行二进制、实际读取字节或记录不可篡改性的认证，也不是审查者独立复现。

### 6.2 临时探针的源码和时机

可取得 `s196/audit-export`、`s196/audit-export-after` 中的临时源码，不只是日志：

- 修改前副本基于 `758da73`，工具子树仍为旧 `95d92292…`。在 `newJob148` 增加的探针读取外层状态、封口状态及工作状态，没有修改工具判定。被探针改写的 `regress148_test.go` SHA-256 为 `8a60b63a773ea77d731ac3643bc76114122bd9a8b34da48d255ded6a21b530a2`，与该次记录相符。
- 修改后审计是提交前工作区的测试改动加临时探针，不应写成固定提交 3a36222 的原封不动测试。临时 `tasks_test.go` 的哈希记录为 `3264aa9defdad1e7935e74cc4f56bac27b1fe83815e2e479938b2515f9a1ffe4`。
- 探针 Cleanup 在 `newJob148` 中登记；各测试随后才登记 settle，因此 settle 先执行、探针后观察；更早登记的临时目录清理在探针之后。这一顺序已结合固定 Go 源码核对：`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`，`src/testing/testing.go` 的 Cleanup 与 runCleanup（1438 行附近、1785 行附近）采用后进先出。[固定官方源码](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/testing/testing.go)
- 探针所谓 `finish_called` 实际取的是 `Closeout.Started`，不是每次 Finish 调用的计数，也不是 Finish-return 通知。正常审计中的记录与源码相符，但这个观测维度不能证伪 B-151-1 的超时窗口。

`zz_failexit_probe_test.go` 也可取得，SHA-256 `dfc222fc2c3279c45261009cfb3843b81323ab3c3fa73ecaee18098fbc0cc676`。它先进入受控核验阻塞、等 RunDomains 在 W 返回，再在任何 Finish 之前主动 `t.Fatal`。Cleanup 释放、等 WorkDone、补跑收尾，最后探针观察已发布终态。退出 1 是预期失败，不是正式全量测试失败；它证明这个单一失败出口，不证明所有失败、超时或已有 Finish 未返回的路径。

### 6.3 正式测试与静态记录

| 既有记录 | 独立读到的结果及限度 |
|---|---|
| 定向 `B148\|B149` 作业测试 | 37 个顶层 PASS，0 FAIL |
| 作业包全量 | 55 个顶层 PASS，包 ok |
| 工具模块全量 | 11 个有测试的包 ok；198 个顶层、271 个子测试 PASS；1 个 SKIP；0 FAIL |
| 重复 20 次 | 61 个不同顶层测试，各 20 次，合计 1220 个 PASS；仅所列四个包，不是全部模块重复 20 次 |
| 静态记录 | 格式、vet、带标签 vet、六配置交叉 vet、正式入口只编译、模块校验、导入闭包记录均成功；第三方导入仍只有 x/mod/sumdb/dirhash |

全量 SKIP 为 `TestTakeNegatives/symbolic_link`，本机符号链接准备权限不足，不能当成该子测试运行通过。类型检查故障夹具的 panic 诊断不等于整个测试失败，按固定预期与最终 PASS 区分。

读取了记录脚本：日志中的 `go $*` 是参数展示，执行使用保留参数边界的 `"$G3" "$@"`；不能把展示中的正则 `|` 误读为实际 shell 管道。交叉 vet／只编译不等于相应平台运行，20 次重复不是 -race 或全调度覆盖。旧 s192～s194 失败及后续成功按原审查范围保留，没有重新全量核验或改写历史。

## 7. 当前常规 CI

独立读取 [run 43596222426](https://github.com/XKHoshizora/specmetry/actions/runs/43596222426) 正式元数据和八份实际 job 日志：pull_request，attempt 1，completed/success，8/8。

每份日志都检出 `abe3a126ca9f063268797850778f07c58d2d6757`，不是只看 run.head_sha。GitHub commit 对象确认其父提交依次为固定 base、固定 HEAD，tree 为 `db114b629057f04da5e68ee22b040205a990130e`，与 HEAD 相同。

| Job ID | 作业 | 实际工具链／主要回执 |
|---|---|---|
| 132451176257 | Go lint | go1.27.4 linux/amd64；0 issues |
| 132451176427 | Go tests macOS | go1.27.4 darwin/arm64；27 个 ok 包；7 条 SKIP |
| 132451176438 | Go tests Ubuntu | go1.27.4 linux/amd64；27 个 ok 包；普通测试步骤 15 条 SKIP；独立 root CrossUser 八项 PASS |
| 132451176456 | Go tests Windows | go1.27.4 windows/amd64；27 个 ok 包；10 条 SKIP |
| 132451176504 | Web | success；不安装 Go，没有给此作业补写 Go 版本 |
| 132451479172 | Build with dashboard macOS | go1.27.4；构建及 dashboard smoke 成功 |
| 132451479187 | Build with dashboard Windows | go1.27.4；构建及 dashboard smoke 成功 |
| 132451479191 | Build with dashboard Ubuntu | go1.27.4；构建及 dashboard smoke 成功 |

三平台的 27 个 ok 是根模块有测试包的统计，不能加上独立 root 测试就写成 28 个不同包。Ubuntu 普通测试使用 -race；root CrossUser 是另行编译并以 sudo 运行的步骤，不把这八项写成普通权限测试通过或独立 root race 证据。

macOS 的七个 SKIP 是五个额外真实 Git／旧 Git 环境测试和两个 M-14 探针。Ubuntu 普通步骤另跳过八个要求 root 的 CrossUser，之后独立步骤确实执行并通过。Windows 为上述七项加三个不可读存储／源文件的 POSIX 权限场景。八份日志未发现测试 FAIL 或作业 error；跳过与通过分开记录。

CLI 包耗时分别为 macOS 219.564s、Ubuntu 463.304s、Windows 359.263s，只是该次观测，不是上界、根因证据或维护事项关闭依据。

**常规 CI 没有运行 `prototype/c9evidence` 独立模块及本次修改的测试。** 七个显示版本的作业使用 Go1.27.4；本机日志是 Go1.27.3。根模块 CI 的绿色、Ubuntu race 都不能替代本工具模块的运行、竞态检查或正式取证，更不能否定 B-151-1 的静态事件序列。

## 8. 下一步及保留边界

可以保留：本次历史覆盖更正的主要部分、正常路径的新增 Final 与终态断言、已有本机日志及其准确证据范围。应最小更正 N-151-1，并对 B-151-1 增加测试侧单次 Finish 请求／返回管理和有界失败收尾；不需要修改工具实现、原预算、正式终判、平台控制方案或扩大正式 LC 清单。

本报告没有重开 Review 150 的 B-147-1、B-147-4、B-147-2、B-147-3、N-147-1～3 或 TestWriterExpiresMidway 裁定。它们按各自已审范围保持；当前 NO-GO 针对 PR #57 新增测试收尾的责任窗口。

没有执行 Go、测试、构建、采集器、实验或 CI 操作；没有修改受审源码、测试、文档、依赖、workflow 或 PR 元数据。只读了现有源码、Git 对象、固定官方源码、日志与哈希，新增本报告。正式入口、真实调用者、E-10(ii)、LE-5①／②、N2、LC-9、正式取证和第三批生产实施均不因本报告获得授权，WP2.5／M2 不作验收。

## 9. 结束身份

2026-11-14 22:17（Asia/Tokyo），在报告已写入后，再次核对本地 Git、远程两个分支及 PR #57：分支仍为 `test/wp2.5-c9-evidence-n150-1`；HEAD、tree、工具子树分别仍为 `db220ce3ad7becc2e50c749a6681ab9db17eee9c`、`db114b629057f04da5e68ee22b040205a990130e`、`d311efa4423ce294753eb7848799cf2ebd3f242d`。远程 master／PR base 仍为 `cd44f674f220116a19f82b2f8034a451d2ac642a`，远程工作分支／PR head 仍为固定 HEAD，PR open、未合并。

受审跟踪文件无未提交差异；工作区唯一新增为 `docs/reviews/codex-review-151.md`，保持未提交。报告内仓库相对链接均指向现存文件。未发现身份漂移或其他人的修改。完成审查后暂停，等待负责人决定。
