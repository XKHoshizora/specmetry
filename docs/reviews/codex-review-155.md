# Review 155：PR #58 第二阶段定向复审

## 1. 裁定

**第二阶段实现技术 NO-GO；当前固定对象不建议合并。** 截止准入的修复成立，实际资源登记、H 的任务集合及错误来源的修复也有可保留的进展，但还有以下三个具体接点。这里没有把真实 R 未采集、正式运行未授权或未覆盖全部调度当作阻塞。

| 项目 | 本轮裁定 |
|---|---|
| B-154-1 | **关闭**。入口、两次 Pipe 和 Start 分别复核原 W 的余量；拒绝与已准入的晚返回分开。 |
| B-154-2 | 主体修复成立，**尚不能整体关闭**：实际对象已放入 Record，但局部登记表及 H 未形成直到测试进程结束的根持有关系。见 B-154-2/3-R-H。 |
| B-154-3 | 半构造、迟到启动交接、ready 后续工作与错误消费的主体修复成立；**尚不能整体关闭**，与上一行共享同一持续持有缺陷，不重复计数。 |
| B-154-4 | 站点、来源、接缝真实结果及错误次数的修复成立，**尚不能整体关闭**：LC-9 仍丢失信号退出的区别；RC-3 新增的精确结果预期又依赖未建立的子进程存活前提。见 R-S、R-C。 |
| N-154-1／2 | **完成**，限于 §10.6 的追加更正和证据口径。F-1 的正式 workflow 处置、F-2 后续文字整理仍是后续事项。 |

本轮没有重新否定 Review 154 已认可的平台控制主体、第一阶段的来源核验／截止接线或 F-3 的既定修正。三平台成功、Linux race 成功及本机记录均保留为对应路径的证据，不能排除下面的静态反例。

## 2. 固定对象、范围与登记

开始及结束核对的固定身份如下：

| 对象 | 核验值 |
|---|---|
| PR | XKHoshizora/specmetry #58，OPEN |
| 本地及远程分支 | `test/wp2.5-c9-evidence-phase2` |
| 最终 HEAD／PR head | `764da7e37d12476c28f0092daea4315f6b0cce9c` |
| HEAD tree | `36f3ce134f485f9475ee8b8ca50711e3728c0c3e` |
| base／PR base／远程 master | `09e81ee3a35a1fdcfd604f509851f1769e43fd3e` |
| 上轮 HEAD | `8e228f847706fc5fa9657c99078e8c985ccc2eb1` |
| 专用代码点／远程 `c9test/phase2` | `a0ed8dc0800bc64b8f1c88e715f00fa600d053fd` |
| 专用代码点整树 | `9d120b675a08a18ede3fdd2b647c7839c971840e` |
| 两代码点工具模块子树 | `76a5c49c037b279ba783913cc95a56dd6e93c9eb` |
| 两代码点测试 workflow blob | `4387acdc683a2014c235f3991bf854a30d6c7c2c`（`.github/workflows/c9-tool-test.yml`） |

已读取当前 AGENTS、[Review 154](codex-review-154.md)、[处理记录](../review-resolution.md) §202／§203、[第二阶段记录](../wp2.5-c9-evidence-phase2.md) §10、[实施计划](../wp2.5-implementation-plan.md) §12.12／§12.14 的直接规则，以及限定取证计划的对照、真实错误和持有边界。审查沿修改后的 toolctl、LC-9/H、controls.Judge、realcall 回归及 Job／正式入口的直接消费者展开。

实际提交顺序为：`7c97623` 单独归档 → `e3959ff` 修复登记 → `2cdbc1d` 回归先行 → `8b1f825` 控制协议修复 → `a0ed8dc` H 与完整错误比较 → `764da7e` 记录与证据。§202 在修复代码之前；§203 追加结果。旧处理记录字节串是新文件的完整前缀，没有回改历史。

归档的 Review 154 为 29,792 字节，SHA-256 为 `2a8d34bde3452298a582ee2d0cb695056ada5f924af4f42ab86e2e1ed3630c88`；工作树文件与独立归档提交的 blob 一致，和登记的摘要相符。本轮未取得另一个仓库外原始副本，不额外声称做过该副本的比对。

从上一轮到本轮的实际差异包含 40 个文件。实现变动集中在 toolctl、controls/LC-9/H、对应回归和 realcall 测试；没有修改第一阶段 chain／job 的协议实现，也没有修改依赖、正式 workflow 或预算。`a0ed8dc` 到最终 HEAD 的实际差异是文档、证据归档和 AGENTS 状态同步，不是只凭作者说明认定“代码未变”。两点的模块及测试 workflow 对象已经分别核对。

## 3. B-154-1：截止准入关闭

固定 HEAD，`prototype/c9evidence/internal/toolctl/toolctl.go` 402～435、启动路径 570～750：

- 入口登记与后台启动任务的 `stdout pipe`、`stderr pipe`、`start` 都重新读取同一个时钟，使用 `W-now >= collect.CallMargin`（5 秒）。前一次准入不授权后一次动作。
- 四次检查使用同一个 W，没有滚动预算。§202 已明确把三个准备／启动动作作为同一次调用的启动余量检查；它比只在最后检查 `now<W` 更保守，没有增加运行时间或改变内部结果的及时判据。
- `refuseLocked` 只在尚未调用 Start 时写 A/not-admitted。已准入的 Start 未返回、迟到成功或 panic 不经过这个拒绝分类，仍按实际阶段保留 C／未知或后续结果。
- 第二次 Pipe 或 Start 被拒绝时，已取得端点有唯一关闭责任。这里完成原调用的关闭、已成功启动后的唯一 Wait 及既有 pidfd 协议，不是再次准入一项新的分析工作；§202 对这个接点已有明确登记。
- Start 之前被延迟调度、准备调用跨 W／D、Start 已准入后晚返回分别有回归和状态记录。不能从“W 后仍有 Wait／关闭”反推新增了一个未准入的工具调用。

本裁定不要求系统调用可中断，也不要求读时钟与下一条机器指令硬实时原子。R1-a～e 的证据种类见 §7：其中依赖新钩子的用例不是旧实现上的动态红绿对照。

## 4. 尚未关闭的具体接点

### B-154-2/3-R-H：记录内有实际对象，但局部记录的持有期限没有闭合

**位置：**固定 HEAD：

- `prototype/c9evidence/internal/toolctl/toolctl.go` 226～261、330～355、1207～1228；
- `prototype/c9evidence/internal/toolctl/regress154b_test.go` 229～256（R2-f）；
- `prototype/c9evidence/internal/controls/lc9.go` 295～306、381～416；
- `prototype/c9evidence/internal/controls/lc9holder.go` 58～79、371～380；
- `prototype/c9evidence/internal/toolctl/main_test.go` 的 TestMain。

**已经修好：**Record 确实保存四个 `*os.File`，Cmd 在 Prepare／Start 之前保存，Process 在成功启动后保存；描述符区分关闭发起、真实返回和释放知识。H 也有实际端点和 Cmd，而不再只有描述文字。问题在这些记录本身的根持有者，而不是再次要求给 Record 增加同样字段。

**具体可达顺序一（受审 R2-f 本身）：**

1. 测试在局部变量中建立 Registry，真实启动静默退出的伪工具。
2. `HookBeforeWait` panic；恢复路径保存 C/panic-before-wait，保留 Cmd、Process 和 Linux pidfd，不补做 Wait，符合该用例选定的异常策略。
3. 读任务到 EOF 并完成关闭，其他任务也返回；`WaitTerminal` 成立，测试读取 Snapshot，确认 `ProcessHeld/CmdHeld` 和 pidfd 状态后返回。
4. Registry 没有移交到包级／进程级的持有表；TestMain 也没有保存它。Snapshot 只有状态副本，不持有这些实际对象。此时不再有存活任务替它提供根引用。

因此，“在测试函数内能看见 `Held=true`”不能兑现“直到测试进程结束仍由记录持有”。局部对象之间的引用环也不是 GC 根。Linux 的原始 pidfd 可能仍是打开的 OS 资源，但这不等于还有可达的程序对象承担责任；os.Process 的内部句柄释放也不是 Wait，更不能证明子进程已回收。本轮没有运行 GC 或观测泄漏，裁定依据是可达对象关系与协议声明的直接不符。

**具体可达顺序二（同一缺陷在 H 的消费者中）：**LC-9 建立局部 `lc9Run`／holder，端点真实关闭返回错误后任务结束。H 的“任务完成”可成立，但释放知识仍未知。`run` 返回的 Outcome 保存工具 Snapshot、故障和 H 的展示行，没有保存实际 holder／Registry。局部调用结束后，已结束任务不能继续替这些未知对象提供根引用。H 已把错误加入 Judge，能拒绝本次对照；拒绝本身仍不完成资源责任。

这不是说所有 Registry 都没有持有者：`prototype/c9evidence/run/run_test.go` 的包级 `current *job.Job`，配合 `internal/job/formal.go` 中同一个 Tools／Invoker，确实保留正式 R 的登记表。缺口是局部夹具／F 域的登记表和 H，以及 R2-f 这种任务结束但回收未知的实例，不应把正式 R 的根引用替它们作证明。

**违反规则：**实施计划 §12.14 第 3、10 条要求登记表持有全部记录及未确认责任直到测试进程结束；§202 要求真实对象、原始错误和异常责任持续保留。第二阶段记录 §10.8 对 R2-f“由记录持有”的表述超出了目前的持有期限。进程结束后的系统收养／资源释放不补成测试进程内的责任记录或可靠回收证据。

**最小修正范围：**在局部调用者退出前，把仍需保留的实际 Registry／holder 交给有明确进程或作业寿命的持有结构；保存原对象、错误及知识状态。可以复用合适的实际所有者，不要求某一种全局变量设计。不得通过盲目重试 Close／Wait、补写 Reaped 或仅追加 `Held` 字段消除问题。R2-f 可以保持“不补做 Wait”的受审异常策略，但必须有真正的后续持有者。针对局部返回后的对象可达性给出检查，不能只再断言一次局部 Snapshot。

**计数：一个共享残余，同时阻止 B-154-2、B-154-3 整体关闭。**

### B-154-4-R-S：LC-9 仍把不同 Unix 信号退出当成同一个预期错误

**位置：**固定 HEAD：

- `prototype/c9evidence/internal/toolctl/toolctl.go` 730～738；
- `prototype/c9evidence/internal/toolctl/lines.go` 79～88；
- `prototype/c9evidence/internal/controls/lc9.go` 56～95、385～400、695～713；
- `prototype/c9evidence/internal/controls/controls.go` 217～265。

toolctl 已保存 `Exit.Code`、`Exited`、`String` 和真实 `*exec.ExitError`。但 LC-9 的比较链只留下 `wait exit -1` 与 `wait operation exit-status`：`faultKind` 把所有 `*exec.ExitError` 归成同一类，`Lines` 不保留具体信号，Judge 比较规范行而不消费 `Extra["call"]` 中的退出身份。

**可达反例：**Unix overflow 用例已经取得 4096 个预期字节并判断超量，之后伪工具在预期 SIGPIPE 发生前因另一个信号（例如 SIGTERM）结束。其余端点关闭／回收成功，无额外请求或工具错误。Wait 同样是 `*exec.ExitError`，ExitCode 同样为 -1，输出 hash、EOF／overflow、任务状态和规范故障集合都可能与现有预期相同，Judge 因而可以给 Hit。真实的非预期终止被冒充为该夹具预期的 broken-pipe 终止。

这不是以信号名称假想新机制：固定 Go 1.27.3 提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff` 的 `src/os/exec_posix.go` 125～162 中，ProcessState.String 区分信号，而 ExitCode 对信号退出给 -1。受审代码已经取得较完整状态，却在最终消费时舍弃了这个区别。本轮通过固定官方源码读取核对，没有运行 Go。

**影响与规则：**§202 对完整错误的要求不仅是“同站点、同来源、同 Go 类型”，还要求非预期真实结果不能匹配计划故障。本次 SIGPIPE 的成功观测可保留，但不能授权所有信号退出。将整份 Snapshot 放入 artifact 而不进入判定也不能修复假命中。

**最小修正范围：**把相关变体实际需要的退出种类／信号身份接入规范结果和独立预期，保持错误来源、次数与其他真实错误；不必比较任意错误全文，也不要求通用错误认证。一起检查其他允许 `wait exit -1` 的 LC-9 分支。RC-2 已检查 `signal: broken pipe`，这项做法的作用不应仅留在 RC 测试而绕过 LC-9 的消费者。

### B-154-4-R-C：RC-3 的新结果预期把“尚未 Wait”误当成“仍在运行”

**位置：**固定 HEAD，`prototype/c9evidence/internal/realcall/realcall_test.go` 353～389、410～425。

RC-3 在父端 `HookBeforeWait` 上暂停，然后推进假时钟到 W，等待终止请求完成，再释放唯一 Wait。新增注释据此断言 go 仍运行，Linux 必须被 SIGKILL 结束、Windows 必须退出 1；这不由握手推出。钩子暂停的是父端，不是子进程。真实 `go list` 对有限夹具可以自行完成。

**允许的事件顺序：**Start 成功 → 父端到达 Wait 前钩子并发出 entered → 测试父端暂未获调度 → go list 正常完成并退出 0，尚未被 Wait → 父端恢复后推进 W 并请求终止 → 放开 Wait。之后 Wait 仍可取得子进程原本的退出码 0；对已经退出的子进程发出终止不会改写其历史退出原因。请求自身的返回也必须按实际值保存，不能靠一项固定断言替代。

由此，一个满足截止拒绝、唯一 Wait、实际回收和完整错误保存的执行也可能因新断言失败。三平台本次恰好观测到 Linux killed／Windows 1 不排除此调度，等待前握手或重复通过均不构成存活证明。

**违反规则与影响：**对照的独立预期必须由实际构造前提推出，不能把等待／回收事件当成退出事件。它是本次修复直接引入的错误拒绝条件，而不是“没有动态覆盖所有调度”的一般建议。

**最小修正范围：**把 RC-3 的父端到期事实、请求结果和实际退出／回收事实分开，写定能够由该真实夹具支持的精确分支；继续拒绝额外真实错误，不得笼统允许任意退出或请求错误。若需要保证活着再终止的证据，应使用有真实存活前提的已授权夹具／另审方案，不可从暂停父端 Wait 推出。修正不应增加真实命令种类或自动扩大实验授权。

## 5. 其余修复的可保留结论

### 5.1 实际资源、异常阶段及平台控制

`holdLocked` 在移交前保存 Pipe 返回的真实对象；Cmd 在 Prepare 接缝前登记。启动任务按实际阶段恢复 panic，已发起 Start、Wait 或 Close 不重试；未返回关闭保持释放未知。Pipe／准备失败、Start 未返回与 Start 成功后未 Wait 已有不同状态和错误记录。父端的 Snapshot 不读未发布的 ProcessState，也不执行外部 I/O。

Linux 的 Wait 发布／封闭准入、请求预留与 pidfd 关闭继续受既有锁和使用责任约束；已预留的请求挡住关闭，无 PID fallback。钩子与实际关闭的边界可避免恢复时重做外部关闭。Windows 继续使用实际 Process 的终止及唯一 Wait；darwin 到期不发终止，保留清理未知。本轮没有发现与 §4 所列残余无关的新平台动作反例。

SIGCHLD 被忽略、标准库内部 panic、Wait 返回但 ProcessState 为空仍只有对应静态分支或限制声明，不能写成已原生覆盖。既有保守 C／未知分类可保留；其中的实际资源保留仍受 R-H 约束。

### 5.2 H 的任务集合和迟到 Start

`lc9holder.go` 已把启动意图、取得的端点、ready 任务、唯一 Wait、释放请求和关闭任务纳入同一个持有对象。stop 与启动发布在锁下确定唯一释放者；Stop 先到而 Start 迟到成功时，由启动路径交接一次释放，不把“尚未返回”当成“未启动”。

ready 成功发布后仍需关闭与任务结束，Terminal 不会仅因 ready 已到达就忽略这些工作。关闭拒绝、真实失败和实际错误有单独记录；H 故障加入 Instance.Faults／比较行，已不只是 Extra。保存被测拒绝后再释放 H 的顺序仍在。任务完成和释放成功依然分开，这是正确的；剩下的是任务完成后的根持有问题。

### 5.3 错误来源、回归及第一阶段消费者

真实 Close 结果先消费；仅真实成功才追加注入。实际失败、替代接缝的返回、真实操作未调用和追加注入有不同来源，额外站点与重复次数保留。R4-a 修复了原先同站点不同来源的假命中；R4-b 原本已不命中，不能计作旧缺陷复现；Linux R4-c 检查另一请求错误没有被当成预期 ESRCH。

整体 `late`／任务未决仍进入比较与 Job 终判，没有只因日志有完整行就认可成功。LC-9 的 Instance.Terminal 同时消费工具与 H；工具登记表的 Problems、第一阶段工作完成资格和最终记录链仍接入既有终判。此处可保留的接线不掩盖 §4 的对象寿命和退出身份缺口。

Windows 的“存在但不是可执行文件”用例在本次工具链／runner 上观测到 216；它是这个具体输入的受控预期，不是所有 Windows 安装的普遍错误码保证。以后不同合法结果不得自动扩充允许错误集合。Unix overflow 的实际 SIGPIPE 同样是具体结果，不能简化成所有 -1 都是该结果。

## 6. N-154-1／2、F-1／F-2 与授权

[第二阶段记录](../wp2.5-c9-evidence-phase2.md) §10.6 已追加更正：

1. **N-154-1 完成。**junction 的失败限定到当时 C→D 的安装、默认 winsymlink 行为和实际 PATH，承认固定源码的 `winsymlink=0` 兼容分支，不再推广成所有 Windows 环境必然失败。当前专用测试的 PATH 调整与其固定环境一起采证；正式 workflow 的处置仍需另行决定和核验。
2. **N-154-2 完成。**固定测试点之后是“文档与 AGENTS 状态同步”；第二次证据 README 的 blob 补记及摘要更新与原日志／tar 的身份分开；RC-4 的接线结论限定为实际健康收尾，而不是动态证明 D 到期和挂起分支。
3. 顶层测试、子测试、PASS 行数和 SKIP 已分开。Review 154 中 487 的历史口径补充为 210 顶层＋277 子测试；本轮新增后的日志统计见下节。
4. F-2 的 A／C 依赖实际 Cmd 前置状态：不存在的可执行路径／扩展名处理不能一概写成 A。补充分类前提是对受控输入的解释，没有改变启动分类优先级；后续文档同步留待相应授权。

E-10(ii) 的本阶段限定测试接受不构成无后代证明。正式 R、N2、正式运行 C/T、G 项、精确例外、第三批实施及 WP2.5/M2 验收没有因此获批。当前修复登记与状态行使用“已修复，待独立复审”，没有把作者结果代替本轮关闭裁定。

## 7. 既有证据的独立核验

本节的“核验”指本轮读取已有原始日志、元数据、Git 对象和归档内容；**没有独立重跑测试**。本机代码／环境关联不扩张为不可变执行产物认证。

### 7.1 回归先行和本机记录

[local-3](../evidence/wp2.5-c9-phase2/local-3/README.md) 两份日志及 SHA256SUMS 已核对：Windows 日志 6,276 字节，WSL 日志 6,771 字节，均与清单摘要相符。记录关联 `2cdbc1d`，模块子树为 `396064f678d5be5b81bedb027208f1771a163d2c`。

- Windows 有 7 个顶层失败：R1-a/b/c、R2-a/b/c、R4-a；R4-b 通过，R4-c 平台 SKIP。
- WSL Linux 在上述 7 项之外还有 R4-c 失败，共 8 项；R4-b 仍通过。退出 1 是这次回归先行结果，保留不抵消。
- 它是旧逻辑配合回归／必要接缝的代码点，不是宣称完全未改的上轮测试集。R1-d/e、R2-d/e/f、R3-a～e 依赖新增阶段、钩子或状态，没有在旧实现上建立前提，不算动态复现。

[local-4](../evidence/wp2.5-c9-phase2/local-4/README.md) 三份文件全部通过大小与摘要核对：

| 文件 | 字节数 | SHA-256 |
|---|---:|---|
| windows-unit.log | 55307 | `01b377b0bb275e38651ee886daacad7d93d6e0f5d7af422bed47922d20d9cb56` |
| windows-static.log | 1246 | `aa35e63044a4c7752095a86eb661071fe87ac7ae3104f752e24ef1f895e0616e` |
| wsl-linux.log | 56806 | `16ab283d0bd56eb0bba996938a160740d744cf203a8c9677a0641171512551a1` |

日志关联 `a0ed8dc` 和本轮模块子树；记录 Go 1.27.3、隔离的工具环境及测试前后身份。Windows 506 条 PASS＝228 顶层＋278 子测试，另两项 SKIP（Linux 专用请求错误回归、符号链接准备权限）；WSL 的 Linux 二进制 508 条 PASS＝229＋279，无 FAIL／SKIP。静态记录包含九种 GOOS／标签组合的 vet、格式检查、模块校验和正式入口只编译，退出均为 0。交叉 vet 不等于跨平台运行；这些本机记录没有执行 RC 真实调用或 Linux race。

### 7.2 专用运行 43737429057

独立读取正式 run、jobs、artifacts 元数据与三份在线完整作业日志：push，`c9test/phase2`，attempt 1，success，3/3，没有重跑。三个作业实际 checkout 都是 `a0ed8dc0800bc64b8f1c88e715f00fa600d053fd`，不是最终文档提交。其模块／workflow 和最终 HEAD 的关系见 §2。

| 作业 | job ID | 工具链 | 单元 PASS（顶层＋子测试） | FAIL／SKIP |
|---|---:|---|---|---|
| Ubuntu | 132883013142 | go1.27.3 linux/amd64 | 508＝229＋279，12 包 | 0／0 |
| macOS | 132883013235 | go1.27.3 darwin/arm64 | 507＝228＋279，12 包 | 0／1（Linux 专用 R4-c） |
| Windows | 132883013375 | go1.27.3 windows/amd64 | 507＝228＋279，12 包 | 0／1（同上） |

Linux 的工具模块 `-race` 另有 508 条 PASS，没有 DATA RACE／FAIL。它不是常规根模块 race，也不证明全部调度和持有期限。

实际步骤顺序满足先 A 后 B：Ubuntu A 单元后执行 race，16:48:32 完成后才进入 B；macOS A 单元结束后进入 B；Windows A 结束、调整专用测试 PATH 后进入 B。这里的时间来自各自日志，只用于确认顺序。

每个平台 RC 为 5 个顶层＋RC-1 两个子测试，共 7 条 PASS，0 FAIL／SKIP。LC-9 为 Linux 20 Hit，其他平台各 11 Hit＋9 不适用。命中记录可保留，但其判定充分性受 R-H／R-S 限定。

RC-1 两个配置各为 28 个检查、3 次真实调用，合计 6 次，分析的是授权夹具，不是 R。T2 688 字节参数记录、实际输出与 B2 消费证明对应接线，不能认证所有未来命令传递／所有消费者或无后代。RC-2 的 Unix broken-pipe／Windows 退出 0、RC-3 的 Linux killed／Windows 1／darwin 自行结束，均是本次观测；RC-3 不因此获得 §4 所缺的前提。

RC-4 本次是结论 4、退出 0 的健康收尾接线，六次调用；耗时约 Linux 456.088349 ms、macOS 646.425959 ms、Windows 1.0693905 s。仅作本次观测，不是 120 秒上界证明，也不证明异常或 D 到期出口全部动态覆盖。

### 7.3 artifact 与记录身份

[ci-43737429057](../evidence/wp2.5-c9-phase2/ci-43737429057/README.md) 的 SHA256SUMS 十项全部匹配。三份重新打包 tar.gz 的成员逐一与 ARTIFACTS.sha256 比较：macOS 13、Ubuntu 14、Windows 13，共 40 项全部匹配；实际读取了其中单元／race／真实调用日志和 RC JSON。

线上 artifact 元数据对应本次 run：Windows `14171091579`、Ubuntu `14170881316`、macOS `14170044774`。这些核对把仓库重打包内容、成员摘要和对应运行记录关联起来；没有把 tar.gz 的摘要写成 GitHub 原始 ZIP 的摘要，也不将其作为原始运行环境或执行字节不可变的认证。

### 7.4 最终 HEAD 常规 CI 43738098787

独立读取正式元数据和八份在线完整作业日志：pull_request，attempt 1，正式 success，8/8。

八份日志实际 checkout 都为 `eadc3902169a93ec1913a13af13a83149cb42ef5`。GitHub Git 对象的父提交依次是固定 base `09e81ee3a35a1fdcfd604f509851f1769e43fd3e`、固定 HEAD `764da7e37d12476c28f0092daea4315f6b0cce9c`；tree 为 `36f3ce134f485f9475ee8b8ca50711e3728c0c3e`，与 HEAD 相同。没有仅使用作业 head_sha 代替实际检出。

| 作业 | job ID | 核验结果 |
|---|---:|---|
| Go lint | 132885059949 | Go 1.27.4，0 issues |
| Web | 132885060201 | success；不安装 Go，不补写版本 |
| Go tests Windows | 132885060230 | Go 1.27.4，27 个 ok 包，10 个 SKIP，无 FAIL |
| Go tests Ubuntu | 132885060257 | Go 1.27.4，27 个 ok 包，普通步骤 15 个 SKIP，无 FAIL；root CrossUser 单独 8 项通过 |
| Go tests macOS | 132885060271 | Go 1.27.4，27 个 ok 包，7 个 SKIP，无 FAIL |
| Dashboard Ubuntu | 132885266729 | Go 1.27.4，embedded 包及 smoke 成功 |
| Dashboard macOS | 132885266749 | 同上 |
| Dashboard Windows | 132885266908 | 同上 |

三个 Go tests 作业共同的 7 个 SKIP 是需要额外环境的真实 Git／只读路径检查：TestClaimWithARealGitThatFetches、TestInitRefusesRealOldGit、TestARealGitThatFetches、TestResolveCommitsWithAGitThatFetches、TestWP23ReadsWithAGitThatFetches、TestM14ReadOnlyProbe、TestM14OpenPath。Windows 另有 3 个不可读存储／来源场景；Ubuntu 普通步骤另有 8 个需要 root 的 CrossUser，随后独立 root 步骤实际执行并通过。不能把这些 SKIP 算成相应场景已在普通步骤运行。

Ubuntu 根模块 race 与上述独立工具模块 race 分开。常规 CI 没有进入独立 `prototype/c9evidence` 模块，不能验证本轮修复。七个显示 Go 版本的作业是 1.27.4；专用工具证据是 1.27.3。

CLI 耗时 Windows 360.534 s、Ubuntu 437.605 s、macOS 179.032 s，仅为观测，不推断根因、资源争用、上界或 M-16 已解决。

### 7.5 历史与未运行范围

Review 154 已核验的 `43680105408`、`43680692609` 两次失败和 `43681144176` 成功按原范围保留。本轮没有重读全部历史作业来声称重新验证；新成功没有抵消旧失败。

当前远程没有 `evidence/` 运行分支；正式 workflow 没有改动。已查询的近期运行清单中，本轮新增的是上述专用和常规运行，没有新增正式取证运行。专用步骤及 artifact 使用夹具，正式入口是只编译而未执行，未见本轮采集 R 的记录。这个核验范围不扩张成对任意已删除历史运行的证明。

## 8. 最小下一步与正式取证前条件

当前先处理三个具体接点：建立局部未知资源的实际长期持有关系；让 LC-9 的终止身份进入完整判定；消除 RC-3 对“Wait 被挡住即仍存活”的错误前提。修复后的验证范围与新代码点由负责人决定，本审查不触发运行，也不自动添加实验。

之后即使第二阶段技术 GO，正式取证前仍须分别完成：

- 正式 workflow 的 F-1 PATH／身份守卫处置及相应审查，不能复制专用环境观测为所有环境的保证；F-2 的分类文字按实际前置条件同步。
- N2、正式运行的 C/T、执行环境及首次运行授权；推送会触发的分支必须落在明确授权之后。
- 固定方法要求的实际输入、工具身份、运行记录、预算和全部对照结果；不得由当前夹具测试代替 R 的证据。
- 依赖策略的生效条件、精确清单审查和批准、G 项逐项决定及冻结／验收同步各自处理；本次没有批准例外、新 L、运行风险或生产实施。

## 9. 操作记录及结束状态

本轮只进行了文本／源码读取、Git 与线上只读查询、哈希核对、已有归档成员和日志读取。没有运行 Go、测试、构建、伪工具、真实工具调用、采集器或实验；没有修改受审文件、workflow、PR、CI，没有提交、推送或合并。

报告生成前的结束复核（本机时间 2026-11-17，UTC+09:00）仍为 §2 全部固定身份，工作区干净，目标报告不存在；生成后再次复核，工作区仅新增本报告。结束时本地 `git ls-remote` 因 DNS 解析失败没有取得结果，已通过 GitHub 只读 API 逐项重核 PR、远程工作分支、master 和专用测试分支，均无漂移；不把失败的本地查询计为成功证据。最终只交付未提交的 `docs/reviews/codex-review-155.md`，随后暂停。

**本报告的 NO-GO 针对受审第二阶段的上述残余；不改写先前限定关闭项，不批准正式 R、N2、正式运行、任何 G 项、精确例外、第三批生产实施或 WP2.5/M2 验收。**
