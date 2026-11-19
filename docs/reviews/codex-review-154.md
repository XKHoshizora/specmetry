# Review 154：PR #58 第二阶段实现独立审查

## 1. 裁定

**第二阶段实现技术 NO-GO；当前固定对象不建议合并。** 发现四项范围内的实现阻塞，分别涉及新动作准入、实际资源持有、辅助进程 H 的生命周期，以及负向对照对真实错误的消费。它们不是由“尚未正式取证”“没有覆盖所有调度”或“预算未测”推导出来的。

| 编号 | 裁定 |
|---|---|
| B-154-1 | 启动任务没有在新的启动／准备动作前复核截止；已到 W/D 后仍可开始新的 `Start` |
| B-154-2 | 描述符记录没有持续持有实际文件对象；启动失败／panic 分支也没有完整保留 Cmd 与资源 |
| B-154-3 | H 的半构造、迟到启动、ready 读取及关闭任务未形成完整的持有与终判集合 |
| B-154-4 | LC-9 的规范行把不同来源的错误压成同一个 `error`，允许非预期真实错误冒充预期命中 |

可以保留三平台正常调用、已执行的伪工具／接缝和 pidfd 交错的局部证据；不能据此认定整个控制协议、所有失败出口和第二阶段已技术完成。正式取证仍未授权，第三批生产实施与 WP2.5／M2 验收边界不变。

## 2. 固定身份、实际范围与授权

本报告行号均指 HEAD `8e228f847706fc5fa9657c99078e8c985ccc2eb1`，除非另标固定 Go 提交。仓库链接从本报告目录出发。

| 对象 | 独立核验结果 |
|---|---|
| 本地／远程受审分支 | `test/wp2.5-c9-evidence-phase2` |
| HEAD | `8e228f847706fc5fa9657c99078e8c985ccc2eb1` |
| HEAD tree | `9209b8b924aa9c78b97152eb6bf2f7d768cd19bb` |
| PR #58 | OPEN；head 为上述 HEAD，base/master 为 `09e81ee3a35a1fdcfd604f509851f1769e43fd3e` |
| 专用测试代码点 | `ed84d091f3d352d46cdf525880998ba6ef569b91` |
| 两个代码点的工具模块子树 | 均为 `c943f922827282e2868ef4da84fe8e370f96f126` |
| 两个代码点的测试 workflow blob | 均为 `4387acdc683a2014c235f3991bf854a30d6c7c2c` |
| 专用测试分支 | `c9test/phase2` 仍在 `ed84d09…` |
| 开始工作区 | 干净；本报告不存在 |

读取了 [第二阶段记录](../wp2.5-c9-evidence-phase2.md)、[处理记录](../review-resolution.md) §200／201、[实施计划](../wp2.5-implementation-plan.md) §12.6／12.12／12.14、[限定取证计划](../wp2.5-c9-limited-evidence-plan.md) 的直接规则，以及 Review 143～146 中有关 R/F/O、真实错误、持续持有、准入和最终消费的裁定。检查 base→HEAD 的实际增量，追踪 `toolctl`、`pseudo`、LC-9、`realcall`，以及 collect／job／verdict／formal／run 和 chain 测试的接点。

授权登记 `8a21e4685e9c9885794040a74ebf6b032b1376d3` 先于 `28969bc…` 等代码提交。处理记录旧内容确为新文件的逐字节前缀，§200／201 追加，未回改旧决定。根模块、常规 CI、正式取证 workflow、冻结正文及验收项没有在本 PR 中修改；工具模块按授权引入 x/sys v0.49.0，go.sum 对应根模块既有的 x/mod、x/sys 四行。

`ed84d09→HEAD` 确实没有工具模块或测试 workflow 改动，但不能简写为“只改 docs/”：还包括 AGENTS.md 的状态同步。第二次运行的 README 补上了 workflow blob，并相应更新其 SHA256SUMS；这不是原始日志、tar 或失败结果被替换。应把该事实精确登记，而不是称归档目录所有字节从未改变。

负责人对 E-10(ii) 的接受仅适用于本阶段限定测试，并接受如实失败／未知时的环境销毁代价；这不豁免已规定的资源持有、错误消费和截止准入，也不证明三种调用绝不产生后代。LE-5①／②已获本阶段授权，不应再写作本阶段未授权；正式 R、N2、正式运行 C/T、精确例外及第三批仍未授权。

## 3. BLOCKER

### B-154-1：既有调用准入没有覆盖后台启动任务的新动作

**位置：**[toolctl.go](../../prototype/c9evidence/internal/toolctl/toolctl.go) 313～365、409～483；[engine.go](../../prototype/c9evidence/internal/collect/engine.go) 299～308。依据为实施计划 §12.14 第 1603～1604 行、第 8 条（1663 行）：W 后不开始新的工作／调用，每个任务在新动作前重新读时钟。

`collect` 在调用前的 `AdmitCall` 检查成立，但 `Registry.register` 只处理 SIGCHLD 的全局拒绝，没有工作截止判断；登记后即启动 `startTask`。该任务从创建 stdout 管道、创建 stderr 管道、启动读取任务到 `cmd.Start()` 都没有新的时间复核。请求任务有窗口守卫，不能替代启动端的守卫。

允许的事件顺序：

1. collect 在 W−5s 之前通过调用准入，登记启动任务。
2. 第一个 Pipe、准备钩子或启动前的其他步骤停留；父端在 W 返回失败，终止观察任务随后到 D 结束。
3. 原步骤在 D 后返回；启动任务继续创建下一条管道／移交读取任务，最终新调用 `cmd.Start()`。
4. 这个进程是在工作和终止窗口均已结束之后才启动，先前登记或父端已经非零不能使该动作合法；也不再有窗口内的终止机会。

同一问题不需要系统调用挂起：启动 goroutine 延迟获得调度即可暴露“登记时通过不等于新动作时通过”的缺口。已准入且正在执行的 Start 迟到返回是另一回事，本意见不要求中断它，也不要求时钟检查与下一条指令硬实时原子。

**影响：**正式接线和伪工具接线共用此任务，不能保证截止后只做既定收尾；当前的失败快照并未阻止新进程能力被启动。

**最小修复：**在真实入口和启动任务必要的新动作决策点落实既定 W 准入，明确拒绝／未执行记录及半构造责任；不要用一次 collect 检查覆盖之后所有操作。已有关闭、唯一回收和合法迟到结果仍按其专用协议处理，不把这项修复扩大为截止后禁止一切既有任务收尾。回归需覆盖启动前跨 W/D、未发起 Start、已取得资源和父端出口，保留健康分支。

### B-154-2：记录持有描述文字，却未完整持有实际资源

**位置：**[toolctl.go](../../prototype/c9evidence/internal/toolctl/toolctl.go) `Record`／`Desc`、410～418、426～459、473～509、552～574、628～632。依据为实施计划 §12.14 第 3／4／10 条及第二阶段记录 §4.1、§4.6。

`Desc` 保存状态、Holder 字符串和错误，没有保存 `*os.File`。文件对象只在启动／读取任务局部变量中；`rec.cmd` 又只在 Start 成功的 495 行赋值。任务返回后，记录仍在并不意味着实际文件或 Cmd 仍被该记录持有。

具体路径：

- stdout 管道创建成功，stderr Pipe 失败；启动任务分别尝试关闭第一条管道。其中一次真实 Close 出错，发布 `DescClosed + CloseErr` 后任务返回。该端没有移交给读取任务，没有 Cmd 保存它，登记表只剩“start task”的文字；真实释放仍未知，却已没有协议中的持续对象持有者。
- 正常读取停止后，读端 Close 出错，读取任务减 Tasks 后返回；Cmd 只引用写端，登记表也没有该读端对象。错误被保存这一点成立，但资源持有结论不成立。
- Start 的 C 类失败没有保存本次 Cmd；启动任务恢复 panic 时只记失败并减 Tasks，没有按已构造阶段移交写端／Cmd。若发生在读端已移交之后，读任务仍可能等写端，而启动者已退出。

这里不把每个 Close 错误等同于一定泄漏，也不推断固定 os.File 的具体失败必然遗留 OS 句柄；问题是协议把它定义为“释放未知”，却没有实现承诺的实际持有关系。垃圾回收／终结器或测试进程销毁不能替代记录中的确认与责任。未动态制造标准库 panic 本身不是阻塞；上述恢复路径的责任空洞才是。

**最小修复：**获得资源后、移交或外部动作前，让持有结构保存实际对象和唯一关闭责任；构造 Cmd 后、调用 Start 前保留它。将 Close 已开始／返回的真实结果／释放知识分开，失败不盲重试，未知对象继续可达。panic 与半构造出口按阶段处理；不要求对 C 类未知内部进程猜 PID 或新增回收动作。

### B-154-3：辅助进程 H 的任务集合和清理交接不完整

**位置：**[lc9.go](../../prototype/c9evidence/internal/controls/lc9.go) 67～75、300～334、346～388、395～489。违反实施计划 §12.12 的“独立释放、唯一 Wait、失败收尾与持续持有”，以及限定计划对实例真正终态和额外 O 错误的要求。

有三个同源接点，不能仅靠正常 `holder … waited yes exit 0` 一行关闭：

1. **半构造丢失。**417 行一次外层 step 内连续执行两个 `os.Pipe`。第一个成功、第二个失败后直接返回；第一个管道既没有关闭，也没有登记实际端点。此外第一条 Pipe 跨 W 时第二条仍会开始，沿用了本应逐步骤准入的旧问题。
2. **迟到启动没有交接。**404～409 只检查一次 `h.closed`。`cmd.Start` 已准入后尚未返回时，外层 cleanup 可以把 `closed=true`，看到 `relW=nil/started=false` 后结束这部分清理。之后 Start 成功，432～435 又写入 relW/started，未重新处理已经封闭的清理状态。H 继续等 stdin EOF，唯一 Wait 虽在后台，却没有保证再次执行独立释放的路径。cleanup 在开始 Close 前就把 h.relW 清为 nil；Close 未准入或失败时也不能用这一赋值表示已释放。
3. **ready 任务不是终态来源。**465～475 的任务先发送 ready，再调用 `cleanupStep(...readyR.Close)`；接收 ready 的任务可继续运行。cleanup 等待 H 的 Wait 和工具登记表，但不等待 ready 读取／关闭任务。`Outcome.Terminal` 只取工具 `sn.Terminal`，`holder_errors` 只进入 `Instance.Extra`，不参与 `Judge` 的完整预期。

第 3 点的可达序列是：ready 已发送，ready 任务尚未登记 Close 时暂停；工具退出，先保存拒绝，再释放 H，H 的 Wait 完成，工具读端 EOF 和 Close 完成；LC-9 返回命中，必要工作、来源核验和外层快照均可完成；ready 任务此后才开始其关闭或发布错误。旧的健康快照没有包含这个必要任务，不能作为“全部必要真实结果已终态”的证明。即使某些具体关闭错误会被已经登记的 O 操作拦住，也不能覆盖**尚未登记**的这个窗口。

**影响：**既有父端可以按期限返回，但 H／其管道的责任不闭合；另有把尚可产生真实错误的实例提前算成命中的路径。单纯在最终输出保存一份 h.errs 不解决消费和终态资格。

**最小修复：**H 的启动意图、各端、ready 任务、唯一 Wait、释放请求均先登记；启动发布与外层停止之间有明确交接。各 Pipe 分别准入，半构造保留并收尾。清理不得在发布真实结果之前丢掉对象。实例的完成条件包括 H 的全部必要任务及真实关闭结果，H 的错误进入实际判定，不只进入 Extra。仍先保存被测拒绝再释放 H，不扩大为真实孤儿领养／后代清理协议。

### B-154-4：LC-9 的完整比较没有保留错误来源，可能假命中

**位置：**[lc9.go](../../prototype/c9evidence/internal/controls/lc9.go) 697～718、300～334；[lines.go](../../prototype/c9evidence/internal/toolctl/lines.go) 34～49、79～97；[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 200～248；[job.go](../../prototype/c9evidence/internal/job/job.go) 492～503。

`toolctl` 本身能把真实 Close 错误保存在 `Failures`，但 LC-9 用 `Lines` 判命中：错误文本及来源被压成 `closed error`、`request … error` 等布尔信息；`Judge` 比较这些行，不检查 `Instance.Extra["call"]` 中的完整结构。保留原始结构供展示不等于判定实际消费它。

确定的反例在 `close-real-error`：

1. 计划情况是目标 f 的真实 `f.Close()` 返回 nil，接缝返回 `errSeamClose`，之后再追加 `errInjected`。
2. 若真实 `f.Close()` 返回另一个非预期错误 Eactual，701 行不返回 errSeamClose，而返回 Eactual；706～710 的 InjectClose 仍追加 errInjected。
3. 两种情况均输出 `stdout-r closed error` 和一次 `stdout-r injected`；其余状态可完全相同。完整行比较相等，`Judge` 可以仍为 Hit。
4. LC-9 使用自己的登记表，不是 `j.cfg.Tools` 的真实采集登记表；额外 Eactual 不会因真实采集登记表的 Problems 自动补救。序列化到 controls.jsonl 的 Extra 也没有改变已经得到的 Hit。

这违反既定 R/F/O 边界和完整预期：预期负向结果只能接受指定来源的计划故障，不能把真实前提不成立换成同一类别的成功命中。它也直接触及 Review 144／145 已明确的“真实 Close 出错时不借安全注入满足原预期”。不是要求字符串完全相等，而是要求语义来源、站点、类型及次数足够区分。

**最小修复：**让 LC-9 判定消费完整结构化故障集合，区分真实结果、计划接缝和追加注入；额外真实故障令对照无效。规范展示行可继续存在，但不能代替这些维度。至少固定上述同站点、不同来源的反例及另一必要操作额外失败的拒绝预期，不以扩大允许错误集合解决。

同类消费检查应覆盖其他允许 `error` 的变体。RC-2 目前重点断言 stdout 超量／其读端关闭及 B 类回收，RC-3 允许按计划记录请求失败；二者的通过不能泛化为“已严格排除所有额外工具错误”。本次实际 artifact 可分别检查，不因这项缺陷凭空断言已有运行发生了 Eactual。

## 4. 可保留的实现结论与直接回退检查

### 4.1 平台控制主体

- Linux 的 `reserveLocked`（764～807）在锁下占有 fd 使用责任、登记任务后启动；`requestTask`（845～878）在执行前重新核对 `[W,D)`。`Wait` 发布与 `PidfdClosing` 在同一锁下；`closePidfd`（674～693）在 Reserved／Running 时锁外等待，归还后唯一关闭。没有发现恢复“旧快照使用已关闭 fd”或 PID fallback 的直接回退。Start 后、唯一 Wait 前取得 pidfd 的顺序成立，仍依赖无自动回收／无其他回收者的前提。
- `signal.Ignored(SIGCHLD)` 为真时记录拒绝、没有启动。它不是 SA_NOCLDWAIT 或所有信号处置的证明；这一未动态覆盖路径单凭缺少测试不新增阻塞。
- Windows 发布 `*os.Process` 后请求 Kill；固定标准库的 Wait／Signal 对进程句柄使用 transient 引用，Wait 的释放不把请求变成对复用 PID 的动作。并发可能产生错误，错误须保存，不能由 Kill 返回 nil 证明已回收。
- darwin 不发终止请求，W 到期失败／清理未知；后续实际 Wait 的晚到成功不恢复成功。该代价已按 §200 仅为本阶段接受，不能推广为正式运行已获接受。
- Wait 返回但 ProcessState=nil 时记 C/wait 和失败，不伪造回收；Cmd 在成功启动路径保存，pidfd 按封闭后的占用状态处理。该分支未动态构造，现有静态路径未发现独立于 B-154-2 的新反例。
- 已执行的外部 Start／Wait／Read／Close／pidfd 调用不持有登记表状态锁。请求或关闭挂起时父端的查询出口不依赖取得它们持有的外部锁。短状态复制成本不是硬实时保证，本次不据此另列阻塞。

### 4.2 第一阶段接线

`FormalConfig` 将同一实际调用登记表交给 Invoker 和 Tools；`Job.Finish` 增加 Tools.Terminal，`copyState` 将 Problems 交给 verdict 的第二级，`CallResult.Control` 保存调用状态。正常 R 调用的错误、未完成及未知不会仅因 collect 的旧快照看似成功而变有效。

第一阶段的工作任务完成资格、来源核验工作化、writer／diagnostic 收尾代码及单次 Finish 测试管理没有被重写；旧 W/D、120 秒与主方法规则未变。发现的回退位于本轮新接入的启动及 LC-9 外层任务，不能用第一阶段既有 GO 覆盖；也不据此重开全部第一阶段实现。

### 4.3 F-3 的最小测试修正成立

[walkthrough_test.go](../../prototype/c9evidence/internal/chain/walkthrough_test.go) 95～103 新增等待诊断三行快照写入结果。结合 [diag.go](../../prototype/c9evidence/internal/chain/diag.go) 的同锁发布与等待写者顺序，manifest Close 仍被闸门持住时，看到三条已发布写入意味着诊断已完成这部分必要工作，下一步只能等待写者，不能还有旧快照行待到 D 后写。

这没有提前释放 manifest Close，没有改结论／出口断言，也没有改 chain 正式实现。父端先返回／晚到关闭与诊断终态的原覆盖保留。首次相关 race 运行通过、第二次失败、第三次通过的历史均保留；第三次的单次 race 成功不是全调度证明。

## 5. LC-9、RC 与证据所支持的范围

20 个 LC-9 变体在源码中可逐项对应：normal-eof、half-construction、start-failure、start-failure-not-executable、read-error、overflow、close-injected、close-real-error、expiry、blocked-close、output-not-closed，以及九个 Linux pidfd 变体。九项分别为 open-failed、blocked-send、sealed-before-reserve、reserved-then-sealed-in-window／out-of-window、wait-first、late-publish、abandoned-before-wait、closing-old-snapshot。

新增“存在但不是可执行文件”没有改变分类规则，是在本轮控制协议验证范围内增加的 C 类输入；不是新增产品／风险决定。平台不适用与未命中分开，Linux 20 命中，macOS／Windows 各 11 命中、9 不适用的实际记录存在。但命中判定的充分性受 B-154-3／4 限制。

| 检查 | 本轮可保留的证据 | 不能推出 |
|---|---|---|
| RC-0 | LookPath、绝对路径、逐分量与 EvalSymlinks 诊断；没有工具进程调用 | 不单独证明守卫成立 |
| RC-1 | 每平台两个配置各三次真实调用；28 项检查成立、夹具闭包及 x/mod 哈希关联，K 一致、发现为零 | 未分析实际仓库 R；不证明所有依赖、所有对象或生产扫描已完成 |
| T2 | 记录中单参数 688 字节、SHA-256 `5dfab1fd5637f6990adee91420b93e6dc23122951f55305eb70cc76597bb2ad7`；真实输出通过 Context 解码；另有伪工具参数回显 | 输出合法不能认证真实 cmd/go 收到的一定是原模板；伪工具收到参数不证明真实 go 的全部行为 |
| RC-2 | 三平台真实第 3 种调用的超量、读端关闭与退出结果 | 不推广为全部错误组合都被精确拒绝 |
| RC-3 | Wait 前接缝、假时钟推进，实际 Linux pidfd／Windows Kill／darwin 不终止；终态及请求结果留档 | 不是真实 110 秒挂起压力试验，不证明后代清理；darwin 后续自然结束不是通用清理保证 |
| RC-4 | 真实夹具调用、LC 对照、Job→Finish 的健康演练；六条真实调用记录成功，最终 4／0 | 不覆盖真实调用下所有 D 到期分支，不证明 R 的 120 秒预算，且不能排除本报告反例 |

RC-4 记录的 controls.jsonl 含表头及 118 个对照结果；Linux 为 118 命中，另外两平台为 109 命中＋9 不适用。正常 record／manifest 关联可以作为已运行健康路径的证据，不能作为异常资源责任与完整终判的替代证明。

## 6. F-1、F-2 与非阻塞精化

**F-1 的实际失败成立，测试 PATH 调整保持守卫。**前两次 Windows 日志显示经过 C:→D: junction 的候选在 EvalSymlinks 被拒绝；第三次只在测试步骤 B 前将实际目标 bin 放到 PATH 前，候选解析到 D:，Go 可执行文件哈希与身份步骤相同。没有删掉守卫、放宽版本或改三种参数／工具环境。

本轮从固定 Go 提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff` 读取 [types_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/types_windows.go)、[symlink.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/path/filepath/symlink.go)、[lp_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec/lp_windows.go)、[exec_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec_windows.go)，与本地 go1.27.3 安装中的对应源码逐文件 SHA-256 相同；另读取固定 `os/exec/exec.go` 核对 Start／Wait 和文件输入路径。没有运行 Go。

**N-154-1（措辞）：**`types_windows.go` 164～173 明确有 `winsymlink=0` 的兼容分支。因此 §9.5／§201 的结论应限定为本次安装和实际默认行为；不能推广为所有 Windows 环境都必然失败。正式 workflow 若保持同样安装／PATH／运行条件，仍需先处理这个已知守卫失败；未来 runner 配置也须重新核验，不能由本次观测预言永久布局。该处是正式运行前事项，不是要求本轮越权修改正式 workflow。

**F-2 属于分类前提的精化／测试输入选择。**固定 `lp_windows.go` 的 `lookExtensions` 在已知扩展名分支可直接接受候选；当前 `.exe` 配合实际 PATHEXT 的输入使 Cmd.Err=nil，进入 Start 后失败为 C。没有已知扩展名而预查找先失败则属于 A②。实现没有为通过测试改变 A/C 优先级；实施计划闸门表应在后续获授权整理时补明测试输入前提，不把表格简写解释为推翻 §12.6。

**N-154-2（证据口径）：**§9.1 的“之后只改 docs/”改为“文档与 AGENTS 状态同步”；说明第二次证据目录 README 的 blob 补记与摘要更新，原始日志／tar 未改。§9.4 对 RC-4 的“D 出口接得上”限于实际健康收尾接线；到期、挂起及异常责任另按接缝／静态证据说明。

每路 64 MiB 是本轮明列的有限输出上限，实现没有悄悄扩大时间预算。本报告不据读取量证明内存或运行时间上界；正式资源与预算仍须负责人在后续阶段处理。

## 7. 既有证据的独立核验

### 7.1 三次专用运行

读取 GitHub 正式元数据、全部九份作业日志，与仓库保存的日志、元数据及 tar 内容交叉核对。三次都是 push、attempt 1、不同代码点的新运行，没有把它们写成重跑。

| run／实际 checkout | 作业 ID（Ubuntu／macOS／Windows） | 正式结果与保留事实 |
|---|---|---|
| [43680105408](https://github.com/XKHoshizora/specmetry/actions/runs/43680105408)／`267012b71ea149e036038e3bc7d3eafa4e08b1c7` | 132708668167／132708668081／132708668297 | Ubuntu、macOS success，Windows failure；Windows B 守卫失败，RC-3 在后台 t.Fatalf 的缺陷导致等待失败，不能说首次全部通过 |
| [43680692609](https://github.com/XKHoshizora/specmetry/actions/runs/43680692609)／`f1b405848127fd7e4cdd2985a5c3a5664dac5f8e` | 132710439823／132710439886／132710439672 | Ubuntu race 中 chain TestCase2a 空指针失败，B 未执行；macOS success；Windows 仍因 junction 守卫 failure |
| [43681144176](https://github.com/XKHoshizora/specmetry/actions/runs/43681144176)／`ed84d091f3d352d46cdf525880998ba6ef569b91` | 132711825257／132711825153／132711825263 | 三平台 success；实际运行 Go1.27.3；A 步完成后才执行 B |

最终专用运行的三个平台各 12 个单元测试包、487 个 `--- PASS`、0 FAIL、0 SKIP；487 是包括子测试的日志 PASS 数，不冒称 487 个顶层测试。Ubuntu 另一次 race 记录同为 487 PASS，没有 DATA RACE 告警。真实调用测试 5 个顶层（RC-0～4），含 RC-1 两个子测试，共 7 条 PASS。其他平台没有 race 证据。

最终 Linux／macOS／Windows 真实候选分别是 hostedtoolcache 的 go1.27.3 x64／arm64／x64 安装，`VERSION`、`GOVERSION` 和候选根的关联成立；不是把三个 runner 写成同一二进制。RC-4 耗时约 0.58／0.80／0.78 秒只作观测。

### 7.2 哈希、重打包与本机记录

只读遍历 tar 内容、重算哈希，没有解包执行或运行工具测试：

- 三个 CI 目录各 10 项 SHA256SUMS 全部匹配当前保存文件。
- 三次 tar 的内部原始文件集合分别为 35、26、40 项，逐项对应 ARTIFACTS.sha256，内容哈希全部匹配，没有用压缩包哈希代替内部文件哈希。
- local-1、local-2 各三份日志的 SHA-256 匹配。读取了命令、退出码、代码身份头和实际 PASS／FAIL／SKIP。Windows 原生为 486 PASS、0 FAIL、1 SKIP（本机符号链接权限）；WSL 的交叉编译 Linux 二进制为 12 包、487 PASS、0 SKIP。交叉 vet、编译与运行分别记录，本机没有 RC 真实调用和 race 证据。
- local-2 文件大小为 windows-unit 49824、windows-static 1246、wsl-linux 49467 字节；local-1 对应 49823、1246、49467 字节。它们分别关联 `ed84d09`、`267012b` 的记录，不是本审查重新执行。
- 在线最终 run 的三份 artifact 元数据分别关联 ed84d09，ID 为 Windows 14148626031、Ubuntu 14148507281、macOS 14147834152。仓库保存的是重新打包的 tar.gz，不应与 GitHub ZIP 的 digest 作字面相等比较。

本轮没有另行下载原始 GitHub ZIP，因此独立确认的是“仓库重打包内容⇄原始文件哈希清单”和“在线运行／作业日志／artifact 元数据⇄代码点”的关联；不声称完成原 ZIP 的逐字节认证，更不把当前哈希扩张为历史执行环境的不可变认证。

证据目录的空白行／行尾空格由原始日志产生；排除证据目录后的本次 diff --check 无问题。本轮没有为了消除原始日志提示而改写证据。

### 7.3 最终 HEAD 常规 CI

[run 43681811213](https://github.com/XKHoshizora/specmetry/actions/runs/43681811213)：pull_request、attempt 1、正式 success、8/8。独立读取八份实际日志：

| 作业 | ID | 核验结果 |
|---|---|---|
| Go lint | 132713838362 | Go1.27.4，0 issues |
| Ubuntu Go tests | 132713838520 | 27 个 ok 包；race；15 条 SKIP；独立 root CrossUser 8 项通过 |
| Web | 132713838587 | 成功；没有安装 Go，不补写 Go 版本 |
| Windows Go tests | 132713838684 | Go1.27.4；27 个 ok 包；10 条 SKIP |
| macOS Go tests | 132713838693 | Go1.27.4；27 个 ok 包；7 条 SKIP |
| Ubuntu dashboard build | 132714097731 | Go1.27.4，embedded |
| macOS dashboard build | 132714097736 | Go1.27.4，embedded |
| Windows dashboard build | 132714097786 | Go1.27.4，embedded |

八份日志实际 checkout 均为 `3c7386d247c17beeeaaba413b0c72aba3488563a`。GitHub Git commit 对象的父提交依次为固定 base、固定 HEAD，tree 为 `9209b8b…`，与 HEAD 同树。本地没有该合并快照对象，所以父提交／树的核验来源是 GitHub 对象，不能写成本地 git show 已成功。

普通测试中的非 root CrossUser SKIP 与后续 root 实际执行分开；平台权限／Git 条件跳过仍保留。没有 FAIL，Ubuntu CLI 556.067 秒只作耗时观测，不推断根因或关闭维护事项。七个显示版本的作业为 Go1.27.4；常规 CI 未运行独立工具模块，不能用它支持或反驳本报告的工具缺陷。

## 8. 正式运行边界和下一步

测试 workflow 只由 `c9test/phase2` 的 push 触发，没有 PR、手动或定时触发；步骤 B 依赖前面成功，实际日志与此顺序一致。正式 workflow 仍只匹配 `evidence/c9-run-1`，未改变。检查远程 evidence/ 前缀无分支；枚举现存 380 条 Actions 运行记录，没有 `c9-evidence.yml` 运行。结合专用日志、临时夹具路径、正式入口只编译的命令，未发现正式入口运行或 R 被采集的证据；这不是对已删除历史或一切外部行为的全局证明。

当前应先由负责人决定是否授权本报告四项定向修复、相应回归及新的固定代码点验证；本审查不自行授权或执行。最小范围是启动准入、实际对象持有、H 的完整任务／交接、LC-9 完整错误消费，不需要重做此前已关闭的全部方法或 pidfd 设计。

在技术修复及独立复审之后，正式取证前仍至少需要：

1. 负责人明确正式运行的 N2、C/T、workflow／分支和资源安排；本阶段 E-10(ii) 的接受不能自动扩展到正式运行。
2. 决定并审查正式 Windows PATH／安装处理，保留原守卫和身份核验；更新正式 workflow 的第一阶段陈旧注释时亦不得假定已获运行授权。
3. 对实际 R 的输入、依赖准备、完整记录、对照和 120 秒预算取得另获授权的证据；当前 x/mod 夹具结果不是 G 项整体成立。
4. 保留精确例外、h_list、T-2／L-b1b、原始路径与发送信号风险、第三批生产实施及验收各自的决定门槛。没有任何一项因本报告或绿色 CI 自动满足。

## 9. 操作记录与结束身份

本轮仅静态读取代码／文档、Git 对象、固定官方源码、已有日志和 artifact，进行只读哈希及文本核对。没有运行 Go、测试、伪工具、真实调用、构建、模板、扫描器或实验；没有安装工具、操作 CI、修改 PR、提交、推送或合并。网络直取固定源码曾因 DNS 不可用失败，后改用 GitHub 连接器读取；没有把失败的下载当作源码核验成功。

结束阶段（2026-11-16 01:07 JST）复核本地 HEAD／tree／工具子树／workflow blob、远程受审分支与 master、PR head/base 均仍与 §2 相同，PR 仍 OPEN，专用测试分支仍为 ed84d09。报告写入前工作区干净；写入后工作区只有 `?? docs/reviews/codex-review-154.md`，受审文件与暂存区均无变化。审查结束暂停，等待负责人决定。
