# WP2.5 第三批：C-9 限定取证工具第二阶段的实施与测试记录

> 状态：**第二阶段的实施登记（写代码之前写定；2026-11-15，review-resolution §200）**。本文的 §1 ~ §8 在第一行代码之前提交；实现、测试、失败与证据在之后追加的各节中记录，不改写本登记。独立 PR，不合并；Auto-fix、Auto-merge 保持关闭。之后：实施完成，固定代码点 `ed84d09` 的三平台测试 run 43681144176 success，结果、偏离与发现见 §9（review-resolution §201；待独立审查）。Review 154 为技术 NO-GO（B-154-1 ~ 4），修复在 PR #58 内进行，登记见 §10 与 review-resolution §202；修复已实施，固定代码点 `a0ed8dc` 的三平台测试 run 43737429057 success，见 §10.1 ~ §10.9（review-resolution §203；已修复，待独立复审）。Review 155 为技术 NO-GO（B-154-1 关闭；三个残余 B-154-2/3-R-H、B-154-4-R-S、B-154-4-R-C），修复在 PR #58 内进行，登记见 §11 与 review-resolution §204；修复已实施，固定代码点 `5e6c3a4` 的三平台测试 run 43906849891 success，见 §11.1 ~ §11.9（review-resolution §205；已修复，待独立复审）。
>
> - 依据：负责人 2026-11-15 的决定（§200）；[限定取证计划](wp2.5-c9-limited-evidence-plan.md) §2.3 的 E-10 (ii)、§2.4 的 LE-5、§5.2 的 LC-9；[实施计划](wp2.5-implementation-plan.md) §12.6、§12.12（“工具的控制协议”与“输出不关闭的拒绝路径”）、§12.14 第 1 ~ 13 条；[第一阶段的实现记录](wp2.5-c9-evidence-tool.md)。
> - 本轮**不**授权：对真实仓库 R 的正式采集；正式运行分支（`evidence/` 之下的任何分支）、N2 的决定与正式运行的 C / T；宣布任何 G 项满足、批准精确例外；第三批的生产实施；WP2.5 / M2 的验收。
> - E-10 (ii) 的接受只是负责人为本阶段测试接受的**限定前提**，不是源码证明或动态证明；本文的任何结果都不把它写成已证明。

## 1. 负责人的决定（§200）

| # | 决定 | 本文的读法 |
|---|---|---|
| 1 | 接受 E-10 (ii)，只用于本阶段测试：在固定 Go 1.27.3、已审的三种调用、参数与环境的约束下，接受“不产生后代”的限定前提 | 只适用于三种调用的固定参数与七个工具设定的环境变量（`spec.Args`、`spec.ToolEnv`）；夹具测试只改变工作目录（夹具模块）、测试自己的截止与输出上限。前提不被写成源码证明或动态证明 |
| 2 | 接受本阶段测试中可能出现“清理未知”与残留进程的代价 | 相关调用按既有规则判为失败或未完成，全部结果如实保留。测试环境最终销毁，**不**作为控制协议已确认清理的证据。不扩大为其他运行时信任、风险或例外的接受 |
| 3 | 授权 LE-5 ①：实现并运行已审计划规定的伪工具、辅助进程与 LC-9 | 伪工具是测试二进制的再执行（实施计划 §12.12）；辅助进程包括“输出不关闭”路径的受控输出持有者 H |
| 4 | 授权 LE-5 ②：在控制协议与伪工具的必要检查通过之后，对临时夹具模块执行已审的三种真实 Go 调用，只在专用、可销毁的测试环境中运行 | 只在 GitHub 托管的 runner 上，由本轮新增的测试 workflow 运行；不在本机（日常开发机，包括其中的 WSL）或任何自托管 runner 上执行。同一作业中，真实调用的步骤只在控制协议与伪工具的步骤成功之后运行 |

## 2. 固定的起点

| 项 | 值 |
|---|---|
| 远程 master | `09e81ee3a35a1fdcfd604f509851f1769e43fd3e`（PR #57 的合并提交；第一父 `cd44f674f220116a19f82b2f8034a451d2ac642a`，第二父 `98c035f96a4aa82bc22df6735a9edb09a28e42d6`），树 `b6fd2b8ae46e1e87e5200e61d177243aac61a7f4`，工具模块子树 `2866d4609f26e49095d98ab098877d55cdf59a6e`。本轮开始时 `git fetch` 之后核对，与负责人给出的身份相同，没有漂移 |
| 分支 | `test/wp2.5-c9-evidence-phase2`，从远程 master 新建（已取消对 master 的上游跟踪）。旧分支 `test/wp2.5-c9-evidence-n150-1` 保留在 `98c035f`；落后的本地 `master`（`56d41b9`）不重置，也不作为远程事实 |
| 本机工具链 | `C:\Users\hoshi\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.3.windows-amd64\bin\go.exe`（第一阶段 §190 的工具链；`go1.27.3 windows/amd64`），一律以 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` 直接执行 |
| `golang.org/x/sys` v0.49.0 | 根模块 `go.sum` 第 46、47 行：`golang.org/x/sys v0.49.0 h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=` 与 `golang.org/x/sys v0.49.0/go.mod h1:hNLxWAXmnKAxqDtdwIYC4bM9oQPEecfsnNMuSxOs3og=`。本机模块缓存中已有这一版本（`cache/download/golang.org/x/sys/@v/v0.49.0.zip`），添加依赖不需要下载 |

### 2.1 PR #57 的合并与 master push CI（登记）

- **合并**：`gh pr merge 57 --merge --match-head-commit 98c035f96a4aa82bc22df6735a9edb09a28e42d6`，2026-11-14T18:49:47Z，合并提交 `09e81ee`。没有 squash、rebase、admin 或 auto-merge；分支保留在 `98c035f`。
  - 父提交与树如上表；合并树等于最终 HEAD 的树；工具模块子树不变；Review 153 的归档 blob `d51c56d`（23537 字节，SHA-256 `9c2cbb9e…14de`）不变。
- **合并之前的常规 CI**：run 43627291492（`pull_request`，attempt 1，8/8 success）。
- **master push CI**：run 43628764649（`push`，attempt 1，8/8 success，2026-11-14T18:49:50Z ~ 19:02:30Z），没有重跑。
  - 每个作业的日志都是 `checkout --progress --force -B master refs/remotes/origin/master`，`git log -1 --format=%H` 打印 `09e81ee3a35a1fdcfd604f509851f1769e43fd3e`。
  - Go lint（132550463639）0 issues。Go tests 每个平台 27 个 `ok` 包，没有 FAIL 行与 `panic:`；包集合与 SKIP 集合按名字等于上一次 master push CI run 43590287451。SKIP 数：windows-latest（132550463881）10，ubuntu-latest（132550463908）15，macos-latest（132550463810）7。`internal/cli` 的耗时为 Windows 411.107 s、Ubuntu 483.572 s、macOS 142.696 s（观测，不是上界）。Ubuntu 的 8 项 `TestCrossUser` 在普通用户下跳过，在单独的 root 步骤中通过。
  - Web（132550463847）1 个测试通过。带面板的构建：Ubuntu（132550778221）、Windows（132550778236）、macOS（132550778276）都是 `./web/` 为 `ok`、冒烟测试打印 `dashboard:       embedded`。
  - 七个显示了 Go 版本的作业都是 go1.27.4；Web 作业不安装 Go。日志中没有 `prototype/c9evidence` 之下的路径：这次 CI 没有测试工具模块。

## 3. 范围

### 3.1 实现

| 部分 | 预定的位置 | 依据 |
|---|---|---|
| §12.14 第 3 ~ 13 条的工具控制协议：登记先于运行、描述符的责任、启动与唯一的回收者、父端只查询、完成之前不读、到期与终止、pidfd 的状态机、持续的持有者与晚到的结果、成功的判据、真实与注入分开、包的终判与 `D` 的出口 | `internal/toolctl` | 实施计划 §12.14 |
| 真实调用者：登记表实现 `collect.Invoker`，是工具中唯一启动进程的边界；第一阶段的 `Stopped` 与 `Fake` 保留给既有测试 | `internal/toolctl` | 同上；计划 §4.2 |
| 伪工具与辅助进程：测试二进制以环境变量再执行；各测试二进制的 `TestMain` 在取得任何截止之前先识别它 | `internal/pseudo`，以及使用它的各包的 `TestMain` | 实施计划 §12.12 |
| LC-9 的变体（替换第一阶段“不命中”的占位） | `internal/controls/lc9.go` | 计划 §5.2 的 LC-9；实施计划 §12.12 |
| 包的终判接入（§12.14 第 13 条）：作业把工具登记表作为收尾的一个终态来源，它的未决与失败进入结论 | `internal/job` | 实施计划 §12.14 第 13 条；计划 §7.1 |
| 正式入口的接线：`job.FormalConfig` 改用真实调用者 | `internal/job/formal.go` | 本轮授权：只实现与编译核对，**不运行** |
| 依赖 `golang.org/x/sys` v0.49.0 | 工具模块的 `go.mod`、`go.sum` | LE-2：`go.sum` 只能新增根模块 `go.sum` 第 46、47 行；出现其他新的模块或行即停下报告 |
| 夹具上的真实调用测试 | `internal/realcall`（构建标签 `c9evidence_realcall`） | LE-5 ② |
| 独立的三平台测试 workflow：固定 Go 1.27.3，只由专用测试分支 `c9test/phase2` 的 push 触发 | `.github/workflows/c9-tool-test.yml` | 本轮授权 |

### 3.2 不实现、不运行、不改变

- 不对 R（本仓库）运行采集；不建立或推送 `evidence/` 之下的任何分支；`c9-evidence.yml` 与 `ci.yml` 一个字节都不改（`c9-evidence.yml` 的注释仍写着第一阶段“真实调用者停止”的状态，那是它的历史说明，本轮不改它；正式入口改用真实调用者之后的状态以本文为准）。
- 不增加三种调用之外的进程启动（LE-7 不变）：真实调用测试只使用三种调用的固定参数与工具设定的七个环境变量；伪工具只用于 LC-9 与控制协议的测试。
- 第一阶段已审的方法、T2、规范表示、阶段与分类、记录链与收尾的规则都不改；LE-9 的范围不变（不实现 B′、规范表示、批准登记与 C 的匹配）。
- 冻结正文、验收项、根模块的代码、依赖、`go.mod`、`go.sum`、预算、超时与并行度都不改。

## 4. 实现要点（写代码之前写定）

实现中如须偏离下面任何一项，在之后的实现节中登记偏离与理由，不改写本节。

1. **登记表**（第 3 条）：每次调用在登记表的锁 `regMu` 之下先建立记录，然后才启动任何任务。记录保存：调用的种类、参数、工作目录与工具设定的环境；状态（已登记、启动中、已启动、启动失败、已回收）与按 §12.6 的类（A：能证明没有创建子进程；B：本次 `Wait` 确认回收；C：其余，含阶段 `start`、`wait`、`std-panic`）；`*exec.Cmd` 与 `*os.Process`；启动错误；`Wait` 的错误与 `ProcessState` 的摘要；四个描述符的持有者与关闭状态；两个读取的字节数、EOF、读取错误与超量；pidfd 与终止请求的状态（Linux）或终止请求的状态（Windows）；清理未知的原因；注入的结果（与真实结果分开）；按发布顺序的事件与发布时刻（`W` 之后的为晚到）。登记表持有全部记录直到进程结束，失败只追加。
2. **描述符**（第 4 条）：启动任务先建立标准输出的管道，再建立标准错误的管道；两个都建立之后才把读端移交给读取任务并启动它们。第二个管道失败时，第一个管道的两端由启动任务各关闭一次，不启动读取任务，不 `Start`，记 A 类的启动失败。`Start` 返回之后（成功或失败），启动任务关闭两个写端各一次。读取任务在停止读取（EOF、读取错误或超量）之后关闭自己的读端一次。父端从不关闭描述符。真实 `Close` 的结果为 nil 才确认释放，出错即失败、不重试。
3. **启动与回收**（第 5 条）：以 `exec.Command` 构造（不用 `CommandContext`、`Cancel`、`WaitDelay`）；`Start` 成功之后，Linux 上在锁外 `unix.PidfdOpen(pid, 0)` 并发布；然后调用唯一的一次 `cmd.Wait()`，在同一次锁下发布结果、`ProcessState` 的复制并封闭 pidfd。`Start` 失败不调用 `Wait`；`cmd.Err` 非 nil 或 `Path` 为空时为 A 类（§12.6 的 A ②），其余启动失败为 C 类（阶段 `start`）。启动任务中标准库的 panic 由启动任务恢复，记 C 类（阶段 `std-panic`）。
4. **父端**（第 6、7 条）：`Invoke` 在 `regMu` 之下只读记录，按记录的 `changed` 通道（每次发布时在锁下关闭并换新）与 `W` 的计时器 `select`；记录进入终态即返回已发布状态的副本；到 `W` 时返回失败的副本（“没有在 `W` 之前完成”），任务继续运行。父端不调用 `Wait`、`Close` 或任何可能阻塞的外部调用；输出只在读取任务发布 EOF 之后读取。
5. **到期与终止**（第 8、9 条）：到 `W` 时回收尚未发布，按平台：
   - Linux：在锁下按第 9 条的规则 1 预留一次（窗口 `[W, D)`、pidfd 为 `打开`、请求为 `无`），预留之后在锁外启动请求任务，它先在锁下重新读时钟，在窗口之内才以预留时记下的 fd 调用一次 `unix.PidfdSendSignal(fd, SIGKILL, nil, 0)`，否则为“放弃”。pidfd 尚未发布时，由登记表持有的到期任务在窗口之内、每次记录变化时再尝试，直到预留成功、pidfd 已封闭或窗口结束；`PidfdOpen` 失败即不预留、不发请求，记清理未知，不回退到按 PID 发信号；不使用 `Process.Kill`。启动任务是 pidfd 唯一的关闭者：请求为 `已预留` 或 `运行中` 时它在锁外等待记录变化，之后才 `Close` 一次。
   - Windows：同样的预留与请求任务，以已发布的 `*os.Process` 调用一次 `Kill()`（进程句柄）。
   - darwin：不发终止请求，记清理未知（`os/wait_unimp.go`）。
   - 只终止直接的子进程；每次调用至多一个请求。Linux 上另记录 `signal.Ignored(SIGCHLD)`：为真时 pidfd 的锚定前提不成立，登记表拒绝运行任何调用（失败）。
6. **持续的持有者**（第 10 条）：`W` 或 `D` 之后，任务继续运行到外部调用返回，结果作为晚到的结果写入记录；未完成的关闭、读取、请求、pidfd 与未知的进程都由登记表持有。
7. **成功的判据**（第 11 条）：同一次调用在 `W` 之前全部满足：B 类回收、退出码 0；四个真实 `Close` 都已发布且为 nil；Linux 上 pidfd `已关闭` 且真实结果为 nil；两个读取都发布 EOF，没有读取错误或超量；没有注入；没有终止请求、清理未知或到期。否则失败。
8. **注入**（第 12 条）：接缝只能在真实结果发布之后追加注入；记录分开保存二者，任一为错即失败，注入不覆盖真实结果。
9. **终判**（第 13 条）：作业把登记表作为收尾的一个终态来源；快照点上有记录没有进入终态，或有任何失败、C 类、关闭错误、请求错误、终止未准入或放弃、清理未知、晚到的错误，都使结论不能是 3 或 4（计划 §7.1 的第 2 级：R 中的清理未知与不满足第 11 条的调用）。收尾与 `D` 的唯一出口沿用第一阶段已审的记录链，不另造。
10. **输出的上限**：§12.14 第 4 条没有写定数值。实现取每一路 64 MiB（标准输出与标准错误各自计算），超过即超量。这是提请审查的实现选择；测试用例与夹具测试可以在自己的登记表中设较小的上限。
11. **接缝**：只在测试中设置，生产（正式入口）为空：管道的建立、读取、文件关闭、关闭之后的注入、`Start` 之前对 `Cmd` 的补充（伪工具的标准输入与模式变量）、`PidfdOpen`、`PidfdSendSignal`、pidfd 的关闭、`Kill`，以及固定交错顺序的命名钩子。
12. **伪工具的模式**（环境变量选择；在 `m.Run` 之前识别并以 `os.Exit` 结束，不解析测试的标志）：回显参数（逐个参数的字节数与 SHA-256）、静默退出、写出大量输出、读标准输入到 EOF 后退出，以及持有者 H 的“持有”模式（标准输出为目标写端、从不写入；标准错误只写一行 `ready`；读标准输入到 EOF 后退出）。

## 5. 执行顺序

1. 本登记（纯文档）先提交；之后才写代码。
2. `internal/toolctl`、`internal/pseudo` 与它们的单元测试（实施计划 §12.12 的控制协议用例与 §12.14 第 9 条表中的交错）。本机验证：Windows 原生的 go1.27.3；Linux 以交叉编译的测试二进制在 WSL 中运行，**只运行伪工具与接缝的用例**（真实调用的包没有它的构建标签就不编译，测试二进制中也不含它）。
3. LC-9 的变体、作业的终判接入与正式入口的接线（正式入口只编译与 vet）。
4. 夹具上的真实调用测试与测试 workflow。
5. 把固定的代码点推送到 `c9test/phase2`（只有这个分支触发测试 workflow）：
   - 步骤 A：身份核对、工具模块依赖的准备、vet（含两个构建标签）、正式入口只编译、全部单元测试（含 LC-9 与伪工具；Ubuntu 另以 `-race` 运行一次）；
   - 步骤 B：只在同一作业的步骤 A 全部成功之后，运行夹具上的真实调用测试；
   - 之后打印残留的 `go` 进程（观测，不作为清理的证据），上传日志。
6. 失败保留并分析；修复在新的代码点上验证，推送新的代码点；不以重跑同一代码点代替修复，不删除旧的失败记录。
7. 证据归档（§7）。
8. 开 PR（not for merge），取得最终 HEAD 的常规 CI；失败保留并报告，不自行重跑。

## 6. 验证安排与固定的预期

### 6.1 证据的类别

| 类别 | 含义 |
|---|---|
| 接缝 / 模拟 | 假时钟、接缝固定的顺序、注入的结果、接缝制造的错误；只说明协议在这些构造下的行为 |
| 实际执行 | 伪工具与持有者 H 是真实的子进程；夹具上的 `go env`、`go list` 是真实的工具调用；都在实际的平台控制路径上运行 |
| 静态依据 | 固定 Go 1.27.3 源码的只读阅读（例如 Unix 上参数数组的传递路径） |
| 未验证 | 写明，不由其他类别推出 |

### 6.2 LC-9 的变体（每个用例用自己的登记表与假时钟；外层只在实例进入终态之后判定）

| 变体 | 平台 | 构造 | 预期 |
|---|---|---|---|
| normal-eof | 全部 | 伪工具回显参数，参数为第 2 种调用的参数（T2 是其中一个元素），退出 0 | 成功：B 类、退出 0；四个关闭为 nil；两路 EOF，标准输出的字节等于回显的预期（5 个参数，第 4 个是 T2：688 字节、SHA-256 `5dfab1fd…2ad7`）；Linux 上 pidfd 封闭并关闭一次、没有请求 |
| half-construction | 全部 | 接缝使第二个 `os.Pipe` 失败 | 第一个管道的两端各关闭一次且为 nil；没有读取任务、没有 `Start`；A 类的启动失败；失败 |
| start-failure | 全部 | 以夹具目录中不存在的绝对路径启动 | `Start` 失败，C 类（阶段 `start`），不调用 `Wait`；两个写端关闭；两个读取 EOF（0 字节）并关闭读端；失败 |
| read-error | 全部 | 伪工具静默退出 0；接缝使标准输出的第一次读取返回 E | 标准输出：读取错误 E，读端关闭为 nil；标准错误 EOF；B 类、退出 0；失败 |
| overflow | 全部 | 登记表的上限为 4 KiB；伪工具写出 1 MiB | 标准输出超量，读端关闭为 nil；本次 `Wait` 回收（退出状态如实记录，不比较）；失败 |
| close-injected | 全部 | 伪工具静默退出 0；标准输出读端的真实 `Close` 为 nil 之后注入 E | 真实结果仍为 nil，注入的 E 分开保存；失败 |
| close-real-error | 全部 | 接缝使标准输出读端的关闭返回真实的错误 E1，之后再注入 E2 | E1 与 E2 都保留，注入不使 E1 消失；失败 |
| expiry | 全部（按平台） | 伪工具读标准输入到 EOF；`Start` 之后（Linux 上 pidfd 打开之后）把假时钟推进到 `W` | `Invoke` 在 `W` 返回失败，读取仍阻塞在 `Read` 中。Linux：预留并经 pidfd 请求恰好一次、结果为 nil，进程被 SIGKILL 结束，回收之后 pidfd 关闭一次。Windows：恰好一次 `Kill`、结果为 nil，退出码 1。darwin：不发请求，记清理未知；外层关闭标准输入之后伪工具退出 0，晚到的回收。都判失败 |
| blocked-close | 全部 | 伪工具静默退出 0；接缝使标准输出读端的关闭阻塞，直到外层释放 | 到 `W` 时 `Invoke` 照样返回失败（关闭未完成）；释放之后关闭为 nil，作为晚到的结果；失败 |
| output-not-closed | 全部 | 持有者 H 在伪工具启动之前继承标准输出的写端；伪工具静默退出 0 | 到 `W` 时判定“输出没有关闭”，先保存判定；外层释放 H，H 被它自己的 `Wait` 回收；之后读到的 EOF 是晚到的结果，不改变判定；不对 H 做任何 PID 动作 |
| pidfd-open-failed | Linux | 接缝使 `PidfdOpen` 失败；伪工具读标准输入到 EOF；推进到 `W` | 不能预留，没有请求，记清理未知；外层释放之后回收（晚到）；失败 |
| blocked-send | Linux | 接缝使 `PidfdSendSignal` 阻塞，直到外层释放；推进到 `W` | `Invoke` 照样在 `W` 返回；请求停在运行中时 pidfd 不关闭；释放之后请求完成（晚到），回收之后关闭一次；失败 |
| sealed-before-reserve | Linux | 父端作出到期判断之后、预留之前（钩子），伪工具结束并被回收、pidfd 被封闭 | 预留在锁下被拒绝；没有请求；pidfd 由启动任务关闭一次；记“终止未准入”；失败 |
| reserved-then-sealed-in-window | Linux | 请求任务开始运行之前（钩子），伪工具结束并被回收、pidfd 被封闭；在窗口之内释放请求任务 | 关闭者等待；请求以仍然打开的 fd 发送一次（进程已回收，结果如实记录）；之后关闭一次；失败 |
| reserved-then-sealed-out-of-window | Linux | 同上，但释放之前把假时钟推进到 `D` | 请求为“放弃”，没有发送；之后关闭一次；失败 |
| wait-first | Linux | 伪工具静默退出 0，在 `W` 之前完成；之后把时钟推进到 `W` 再尝试预留 | 调用成功；封闭之后关闭一次；之后的预留被拒绝 |
| pidfd-late-publish | Linux | 接缝使 `PidfdOpen` 阻塞；推进到 `W` 之后才释放 | 到 `W` 时不能预留；pidfd 在窗口之内、封闭之前发布之后，到期任务预留一次并发送一次；进程结束、回收之后关闭一次；失败 |
| abandoned-before-wait | Linux | 请求任务开始运行之前（钩子）把假时钟推进到 `D`，再释放；伪工具仍在运行 | 请求为“放弃”，没有发送，记清理未知；`Wait` 没有返回时放弃不触发关闭；外层释放、回收与封闭之后关闭一次，不再预留第二个请求；失败 |
| closing-old-snapshot | Linux | 封闭之后、关闭之前（钩子）在 `W` 尝试预留 | 预留在锁下被拒绝（`关闭中`），旧的 fd 不被使用；之后关闭一次 |

- 适用表：只在 Linux 的变体在 darwin、Windows 上写作“不适用”（理由：pidfd 只在 Linux，§12.14 第 8 条）；其余变体三个平台都适用。准备失败、实例在外层的期限内没有进入终态，或外层的释放与等待失败，分别按第一阶段的规则判为不命中或未决（§5.5），不改写为不适用。
- 完整预期写在 `lc9.go` 中，由被测记录的规范行比较，与被测代码分开；接缝固定的顺序与注入是接缝证据，伪工具的启动、读取、回收与终止是实际执行。

### 6.3 夹具上的真实调用（只在测试 workflow 的步骤 B）

夹具模块：临时目录中的 `example.com/c9fx`，`go 1.27`，只 `require golang.org/x/mod v0.42.0`，`go.sum` 只有根模块 `go.sum` 第 42、43 行；一个包导入 `golang.org/x/mod/semver`。模块由步骤 A 之前为构建工具模块而做的 `go mod download` 放入模块缓存；三种调用以 `GOPROXY=off` 运行，不下载。

| 测试 | 构造 | 预期 |
|---|---|---|
| RC-1 采集 | 本作业 GOOS 的两个配置，以真实调用者、真实的守卫与真实的类型检查对夹具运行一次 `collect.Collect`（主模块 `example.com/c9fx`，例外、候选与观察清单都为空） | A1-a 成立（setup-go 的安装满足 §12.14 第 1 条的布局）；集合 E 的全部检查成立；发现为零，K 一致；六次调用都满足第 11 条；第 2 种调用的参数数组中 T2 为一个元素、688 字节、SHA-256 `5dfab1fd…2ad7`，输出按上下文文本 v1 解码，B2 全部成立；第 3 种调用的闭包含夹具包、`golang.org/x/mod` v0.42.0（`Sum`、`GoModSum` 与 go.sum 相同）与标准库包；x/mod 的 I-4 重算的 h1 等于 go.sum |
| RC-2 超量 | 测试登记表的上限为 1 KiB；本机配置的第 3 种调用 | 标准输出超量，读端关闭为 nil；本次 `Wait` 回收（退出状态如实记录，不比较）；失败；记录进入终态 |
| RC-3 到期 | 测试登记表用假时钟；本机配置的第 3 种调用；`Start` 之后、`Wait` 之前（钩子）推进到 `W`，待到期的处理发布之后才放行 `Wait` | Linux：经 pidfd 请求恰好一次，回收之后 pidfd 关闭一次；Windows：恰好一次 `Kill`；darwin：不发请求，记清理未知，`go` 自己结束之后晚到的回收。都判失败；记录在真实时间的期限内进入终态。请求或 `Kill` 的错误（例如进程恰好已经结束）如实记录，不改写为成功 |
| RC-4 作业的演练 | 以 `job.Begin` / `RunDomains` / `Finish` 对夹具 R 运行完整的作业：真实调用者、LC-2 ~ LC-9 的全部变体、记录目录在临时目录中 | 最终结论 4、退出状态 0；记录文件与 manifest 一致；工具登记表的六条记录都满足第 11 条且进入终态。这是对夹具的演练，不是正式取证，不使用 R |

- **T2 的参数传递**（每个平台的实际证据）：RC-1 中第 2 种调用在两个配置上的输出合乎上下文文本 v1、B2 成立；LC-9 normal-eof 中 Go 子进程实际收到的第 4 个参数的字节数与 SHA-256 等于 T2。这些说明本平台的 `os/exec` 把 T2 作为一个参数原样传给了 Go 程序；输出的校验仍不认证 cmd/go 收到的就是原模板（方法记录 §3.3.1）。Unix 上参数数组的固定源码路径在实现节中补核（静态依据）。
- RC-1 的守卫或任何检查不成立时，按实际的失败记录、分析与修复；不放宽守卫或检查使之通过。

### 6.4 本机的运行

- Windows 原生：全部单元测试（含伪工具与 LC-9），不含真实调用的包。
- WSL（Linux）：交叉编译的测试二进制，只运行伪工具与接缝的用例，用于 pidfd 交错的迭代；它们不是证据的替代，三平台的证据以测试 workflow 为准。
- 本机不运行 RC-1 ~ RC-4，也不构建带 `c9evidence_realcall` 的测试二进制去运行它们。

### 6.5 预先列出的未覆盖

- `go` 在其他参数、环境或根包下的行为；E-10 (ii) 的完整可达性（它是接受的前提）。
- 真实的孤儿领养或后代清理（H 是测试自己的直接子进程，不是工具的后代）。
- darwin 上到期之后工具进程的去向（只记清理未知）。
- 交叉配置上的原生执行；正式运行的 120 秒预算（RC-4 只是夹具上的观测）。

## 7. 证据的保存

- 测试 workflow 的每个作业把各步的完整输出写入日志文件并上传为 artifact（保存 90 天）；作业日志另由 `gh` 下载。
- 长期保存的位置是本仓库：`docs/evidence/wp2.5-c9-phase2/` 之下，每次运行一个目录（`ci-<run id>/`），本机运行一个目录（`local-<序号>/`）。每个目录有 `README.md`（代码身份：提交、树、工具模块子树、workflow 文件的 blob；工具链；命令；每一步的退出结果）与 `SHA256SUMS`。归档只追加，不删除或改写旧的失败记录。
- 证据目录的提交只改 `docs/`，不改工具模块的子树与 workflow；报告写明被测代码点与最终 HEAD 的子树关系。

## 8. 停止与报告的条件

- 限定前提被打破（例如观测到三种调用产生了后代）、规格冲突、出现须接受的新风险，或必须扩大实施范围时：报告并等待决定。
- `go.sum` 中出现根模块 `go.sum` 第 46、47 行之外的新行：停下报告。
- 最终常规 CI 失败：保留结果并报告，不自行重跑。

## 9. 实施与测试的结果（2026-11-15；固定代码点 `ed84d09`，待独立审查）

本节在实施之后追加；§1 ~ §8 的登记没有改写。登记中写错或在实施中偏离的地方，在 §9.2 逐项列出。

### 9.1 提交与固定代码点

| 提交 | 内容 |
|---|---|
| `8a21e46` | 本登记（§1 ~ §8）与 review-resolution §200；写代码之前提交 |
| `28969bc` | `internal/toolctl`（控制协议与真实调用者）、`internal/pseudo`（伪工具）、`CallResult.Control`；工具模块依赖 `golang.org/x/sys` v0.49.0 |
| `2676c2c` | LC-9 的变体（`internal/controls/lc9.go`）；作业把登记表接入收尾；正式入口的接线；`TestJobEveryControl` 代替第一阶段的 `TestJobPhase1IsInvalid` |
| `267012b` | 夹具上的真实调用测试（`internal/realcall`）与测试 workflow（`.github/workflows/c9-tool-test.yml`）；第 1 次测试运行的代码点 |
| `3fc444c` | 第 1 次运行与本机验证 1 的证据（只改 `docs/`） |
| `f1b4058` | 守卫记录 `LookPath` 的原始结果；RC-0 诊断；RC-3 的测试缺陷修正；Windows 的路径观测步骤；第 2 次运行的代码点 |
| `8f6010c` | 第 2 次运行的证据（只改 `docs/`） |
| `ed84d09` | 两个第一阶段 chain 测试的前提；Windows 步骤 B 之前的 PATH 安排；**第 3 次运行的固定代码点** |
| `9453eb2` | 第 3 次运行与本机验证 2 的证据（只改 `docs/`） |

- **固定代码点** `ed84d091f3d352d46cdf525880998ba6ef569b91`：树 `65c53940869b9ef28b525966d85eaecaa97a0c62`，工具模块子树 `c943f922827282e2868ef4da84fe8e370f96f126`，测试 workflow 的 blob `4387acdc683a2014c235f3991bf854a30d6c7c2c`。之后的提交只改 `docs/`：最终 HEAD 的工具模块子树与测试 workflow 的 blob 都与固定代码点相同。
- 专用测试分支 `c9test/phase2` 停在 `ed84d09`。`ci.yml`、`c9-evidence.yml`、根模块的 `go.mod` 与 `go.sum` 都没有改动；没有建立 `evidence/` 之下的任何分支。
- 工具模块的 `go.mod` 只 `require` `golang.org/x/mod` v0.42.0 与 `golang.org/x/sys` v0.49.0；`go.sum` 恰好是根模块 `go.sum` 的第 42、43、46、47 行，没有其他新行（以固定工具链的 `go mod tidy`、`GOPROXY=off` 生成，只用本机缓存）。

### 9.2 实现与 §4 的对照；偏离与更正

| §4 | 实现（`internal/toolctl/toolctl.go`） |
|---|---|
| 1 登记表 | `register` 在 `regMu` 之下建立记录并计入启动任务，之后才启动它；记录的字段见 `Record`；全部记录持有到进程结束，`Failures` 只追加 |
| 2 描述符 | `startTask` 先建两个管道，再在同一次锁下移交读端并计入两个读取任务；第二个管道失败时由 `closeDesc` 关闭第一个管道的两端，A 类（`pipe`）；`Start` 之后关闭两个写端；`readTask` 停止读取之后关闭自己的读端一次 |
| 3 启动与回收 | `exec.Command`；`cmd.Err` 非 nil 或 `Path` 为空时 A 类（`precondition`），其余启动失败 C 类（`start`）、不调用 `Wait`；Linux 上 `Start` 之后、`Wait` 之前 `PidfdOpen`；唯一的一次 `Wait` 之后在同一次锁下发布结果、复制 `ProcessState`（B 类；为 nil 时 C 类 `wait`）并封闭 pidfd；启动任务中的 panic 由它恢复，C 类 `std-panic` |
| 4 父端 | `Invoke` 只在锁下读记录，按记录的 `changed` 与 `W` 的计时器等待；进入终态即返回副本，到 `W` 即 `expire` 后返回失败的副本 |
| 5 到期与终止 | `expire` → `reserveLocked`（规则 1）→ `requestTask`（规则 5）；pidfd 或进程尚未发布时由登记表持有的 `expiryTask` 在窗口之内重试；`closePidfd` 在请求为已预留或运行中时等待（规则 3、4、8）；Windows 以已发布的 `*os.Process` `Kill` 一次；darwin 记清理未知；Linux 上 `signal.Ignored(SIGCHLD)` 为真时登记表拒绝运行 |
| 6 ~ 9 | 晚到的结果作为事件追加（`Event.Late`）；`successLocked` 是第 11 条；`InjectClose` 只在真实结果发布之后追加；作业以 `Terminal` 与 `Problems` 接入收尾，问题进入结论的第 2 级（`verdict.Inputs.ToolProblems`） |
| 10 输出的上限 | `DefaultLimit` = 64 MiB，每一路分别计算（提请审查的选择，未改） |
| 11、12 | 接缝与伪工具的模式如登记；伪工具只在 `m.Run` 之前由 `pseudo.MaybeRun` 识别 |

偏离、补充与更正（都在实施中发现，逐项保留）：

1. **LC-9 的变体数**：§6.2 列了 19 个，实现了 20 个。新增 `start-failure-not-executable`：以夹具中一个存在、但不是可执行文件的文件启动，三个平台都是 C 类（`start`）的启动失败（Unix 上 fork 之后 exec 失败，Windows 上 `CreateProcess` 失败）。§6.2 的 19 个都按原预期实现。
2. **`start-failure` 在 Windows 上的类**：§6.2 写作 C 类，结果也是 C 类，但依据须补充：可执行文件名带 `.exe`。固定 Go 1.27.3 的 `exec.Command` 对 Windows 上的绝对路径调用 `lookExtensions`（`os/exec/exec.go:466` ~ `480`），后者对已知的扩展名直接接受、不看文件是否存在（`os/exec/lp_windows.go:85` ~ `91`），所以 `cmd.Err` 为 nil，`CreateProcess` 失败，C 类。没有已知的扩展名时，查找先失败，`cmd.Err` 非 nil，按 §12.6 应为 A ②。实现中第一次写的“Windows 为 A”预期是错的，本机运行即不命中，已在写入代码点之前改正。
3. **登记的笔误**：§6.2 normal-eof 一行最初写作“4 个参数”，在写代码之前改正为 5 个参数（第 4 个是 T2），随登记提交。
4. **规范行**：晚到只记为 `late yes / no`，不逐项列出（逐项会随调度变化）；overflow 变体不比较退出码（`wait exit *`），因为它之后的写入得到 SIGPIPE 还是写入错误，取决于平台与时机。
5. **RC-0**：新增的诊断测试（§6.3 没有列出），不启动进程、不判定，只记录 `exec.LookPath("go")`、它的绝对形式、`EvalSymlinks` 的结果，以及每个路径分量的 `Lstat`、`Readlink` 与 `EvalSymlinks`。
6. **守卫的记录**（第一阶段的文件 `internal/collect/guard.go`）：`GuardValues` 增加 `Found`（`LookPath` 的原始结果，为空时省略，所以构造的值与它们的记录字节不变），解析失败的错误写出这一路径。守卫的规则不变。
7. **测试 workflow 在 Windows 步骤 B 之前的 PATH 安排**：见 §9.5 的 F-1。只在测试 workflow 中，只影响步骤 B。
8. **第一阶段的 chain 测试**：`manifestCloseLate`（`TestCase2a…`、`TestCase2b…` 共用）在调用者把时钟推过 `D` 之前，等待诊断任务三行快照行的写入结果。见 §9.5 的 F-3。只改测试，chain 的代码不变。
9. **作业测试**：第一阶段的 `TestJobPhase1IsInvalid`（LC-9 不命中，所以结论 1）在 LC-9 实现之后不再成立，改为 `TestJobEveryControl`（全部对照与假调用者，结论 4、退出 0）；另增 `TestJobToolProblems`（以替身检验登记表的问题进入第 2 级）。
10. **RC-1 ~ RC-3 的截止**：每个配置或用例用自己的登记表与截止，不是正式运行共享的 120 秒；RC-4 用作业自己的 `D`。

### 9.3 三次测试运行（`c9-tool-test.yml`，每次都是 attempt 1，没有重跑）

| 运行 | 代码点 | Ubuntu | macOS | Windows | 证据 |
|---|---|---|---|---|---|
| 43680105408 | `267012b` | success | success | **failure**：步骤 B 中 A1-a 守卫拒绝候选，RC-1 ~ RC-4 没有真实调用；RC-3 另有测试缺陷（在 goroutine 中 `t.Fatalf`） | [ci-43680105408](evidence/wp2.5-c9-phase2/ci-43680105408/README.md) |
| 43680692609 | `f1b4058` | **failure**：`-race` 一步中第一阶段的 chain 测试空指针 panic，步骤 B 被跳过 | success | **failure**：同上的守卫失败；RC-0 取得逐分量的事实 | [ci-43680692609](evidence/wp2.5-c9-phase2/ci-43680692609/README.md) |
| 43681144176 | `ed84d09` | **success** | **success** | **success** | [ci-43681144176](evidence/wp2.5-c9-phase2/ci-43681144176/README.md) |

- 每次失败之后都先保留证据、分析原因，再在新的代码点上修复与验证；没有以重跑同一代码点代替修复，也没有删除旧的失败记录。
- 本机验证：[local-1](evidence/wp2.5-c9-phase2/local-1/README.md)（`267012b`）与 [local-2](evidence/wp2.5-c9-phase2/local-2/README.md)（`ed84d09`）。Windows 原生 486 个 PASS、0 FAIL，1 个已知 SKIP（本机不能创建符号链接）；WSL 中交叉编译的 Linux 二进制 487 个 PASS、0 FAIL；只有伪工具与接缝，没有真实调用。

### 9.4 固定代码点的结果（run 43681144176）

| 项 | Ubuntu | macOS | Windows |
|---|---|---|---|
| 步骤 A：vet（三种标签集合）、格式、正式入口只编译 | 成立 | 成立 | 成立 |
| 步骤 A：单元测试 | 12 个包，487 PASS / 0 FAIL / 0 SKIP | 同左 | 同左 |
| 步骤 A：`-race` | 487 PASS / 0 FAIL，没有数据竞争告警 | — | — |
| LC-9 | 20 个命中 | 11 个命中，9 个不适用 | 11 个命中，9 个不适用 |
| RC-1 | 两个配置的 28 项检查全部成立，K 一致，发现为零；六次调用都满足第 11 条 | 同左 | 同左 |
| RC-2 | 超量；读端关闭为 nil；`go` 因 SIGPIPE 结束 | 同左 | 超量；读端关闭为 nil；`go` 退出 0 |
| RC-3 | 经 pidfd 请求一次（nil），SIGKILL；pidfd 关闭一次 | 不发请求，清理未知；`go` 自己完成，晚到的回收 | `Kill` 一次（nil），退出码 1 |
| RC-4 | 结论 4、退出 0（0.58 s） | 结论 4、退出 0（0.80 s） | 结论 4、退出 0（0.78 s） |
| 残留进程的观测 | 没有 `go` 或测试进程 | 同左 | 同左 |

- 守卫的候选：Ubuntu `/opt/hostedtoolcache/go/1.27.3/x64/bin/go`，macOS `/Users/runner/hostedtoolcache/go/1.27.3/arm64/bin/go`，Windows `D:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe`（F-1 的 PATH 安排之后）；`VERSION` 与 `GOVERSION` 都是 go1.27.3，`GOROOT` 与候选的根相同；可执行文件的 SHA-256 与身份步骤打印的相同。
- RC-4 是夹具上的演练：它说明登记表、收尾与 `D` 的出口在真实调用下接得上，不是正式取证，也不说明 R 上的 120 秒预算够用。

### 9.5 发现

- **F-1：Windows runner 上守卫的候选经过 junction**（实际执行 + 静态依据）。setup-go 在 windows-latest 上的 GOROOT `C:\hostedtoolcache\windows\go\1.27.3\x64` 是指向 `D:\hostedtoolcache\windows\go\1.27.3\x64` 的 junction。固定 Go 1.27.3 中，junction 的 `Lstat` 模式为 `ModeIrregular`、没有 `ModeDir`（`os/types_windows.go` 的 `mode`，Go 1.23 起的 `winsymlink`）；`walkSymlinks` 遇到不是符号链接、也不是目录的中间分量即返回 `syscall.ENOTDIR`（`path/filepath/symlink.go:90` ~ `93`），Windows 上它就是 `ERROR_PATH_NOT_FOUND`（`syscall/zerrors_windows.go:9`）。所以 §12.14 第 1 条的守卫对这一安装的 `go.exe` 必然失败（J-1）。
  - 守卫的规则没有改。测试 workflow 在 Windows 的步骤 B 之前，把 junction 的目标 `bin` 放到 PATH 最前（§12.14 第 1 条对本地运行的要求同样是让真实 GOROOT 的 `bin` 先被找到），守卫照常核对同一份 `go.exe`（SHA-256 相同）。
  - **对正式运行的影响**（未处理，待负责人决定）：`c9-evidence.yml` 没有改，它在 windows-latest 上会得到同样的 A1-a 失败，Windows 作业的结论只能是 2。可选的处置：在正式 workflow 中做同样的 PATH 安排，或修订守卫的解析方法；都须另经审查与决定，可以与 N2 一起处理。本轮不处理。
- **F-2：实施计划 §12.12 闸门表中“以绝对路径启动不存在的文件（三个平台）”的 C 类**（静态依据，§9.2 第 2 项）：只在 Windows 上的文件名带已知扩展名时成立；不带时 `cmd.Err` 先被设置，按 §12.6 应为 A ②。这影响第三批 `procgate` 测试的写法，本轮不处理，只登记。
- **F-3：第一阶段 chain 测试的前提**（实际执行）：`-race` 下 2a 在第 121 行读取写者尚未发布的关闭结果。原因是测试没有保证诊断任务在 `D` 之前写完快照行；诊断任务在 `D` 之后仍有待写的行时走到期出口，这符合它的规则。修正只在测试中加入这一前提（§9.2 第 8 项）。第 1 次运行的 `-race` 中同一测试通过，所以这是取决于调度的测试缺陷；修正之后的一次 `-race` 运行通过，不等于全部调度都已覆盖。
- **F-4：RC-3 在 darwin 上**：到期时不发请求、只记清理未知，`go list` 之后自己完成（标准输出 27998 字节）并被晚到地回收。这次没有残留，但这只是观测；darwin 上工具进程可能在测试进程结束之后继续运行的限制不变。

### 9.6 T2 的参数传递

| 平台 | 实际执行 | 静态依据 |
|---|---|---|
| Linux | RC-1：两个配置的第 2 种调用，参数数组 5 个元素，第 4 个为 T2（688 字节，SHA-256 `5dfab1fd…2ad7`），输出合上下文文本 v1，B2 成立；LC-9 normal-eof：Go 子进程收到的第 4 个参数的字节数与 SHA-256 等于 T2 | `os/exec/exec.go:744` 以 `c.argv()`（`:535` ~ `540`，即 `Cmd.Args`）调用 `os.StartProcess`；`os/exec_posix.go:55` 调用 `syscall.StartProcess`；`syscall/exec_unix.go:162` 以 `SlicePtrFromStrings` 把每个参数复制为一个以 NUL 结尾的字节串（`:84` ~ `98`，参数中有 NUL 即 `EINVAL`），`:208` 交给子进程；Linux 子进程以 `SYS_EXECVE` 传入这个数组（`syscall/exec_linux.go:673`）。没有 shell，也没有拆分 |
| macOS | 同上（两个配置） | 同一路径，子进程经 libc 的 `execve`（`syscall/exec_libc2.go:284`） |
| Windows | 同上（两个配置；PATH 安排之后） | 第一阶段已核对的命令行拼接与拆分（限定取证计划 §1.2 第 2 项） |

输出的校验仍不认证 cmd/go 收到的就是原模板（方法记录 §3.3.1）。

### 9.7 证据的类别

| 类别 | 本轮的内容 |
|---|---|
| 接缝 / 模拟 | LC-9 的假时钟与闸门（固定交错的顺序）；注入的关闭错误；接缝制造的管道、读取、关闭与 `PidfdOpen` 的错误；阻塞的关闭与发送；RC-3 中 `Wait` 之前的钩子；`TestJobToolProblems` 的登记表替身 |
| 实际执行 | 伪工具与持有者 H 是真实的子进程（三个 runner 与本机）；夹具上的 `go env`、`go list` 是真实的工具调用（三个 runner，RC-1 ~ RC-4）；Linux 上经 pidfd 的 SIGKILL（LC-9 与 RC-3）、Windows 上的 `Kill`、darwin 上不发请求，都是实际的平台路径；Ubuntu 上 `-race` 的运行 |
| 静态依据 | §9.6 的参数传递；§9.2 第 2 项与 F-2 的 `lookExtensions`；F-1 的 junction 与 `EvalSymlinks` |
| 未验证 | §9.8 |

### 9.8 没有覆盖、没有运行或仍保留的

- 登记表在 `SIGCHLD` 被忽略时的拒绝、启动任务中标准库的 panic（C 类 `std-panic`）、`Wait` 返回而 `ProcessState` 为 nil（C 类 `wait`）：有代码，没有构造的测试。
- 正式入口只编译与 vet，没有运行；`c9-evidence.yml` 没有改，它的注释仍写着第一阶段的状态；R 没有被分析；正式运行的 120 秒预算没有测量。
- E-10 (ii) 仍是负责人为本阶段测试接受的限定前提，不是证明；“没有观测到残留进程”不是清理的证据。
- `-race` 只在 Linux 上运行；本机没有 `-race`。各用例的交错是接缝固定的顺序，不是全部调度的覆盖。
- 夹具只依赖 `golang.org/x/mod`；第三方的扫描与类型检查只覆盖它与它的标准库闭包。
- 真实的孤儿领养与后代清理没有覆盖：H 是测试自己的直接子进程。
- F-1 的 PATH 安排只在测试 workflow 中。

### 9.9 交接

- 固定代码点 `ed84d09` 与本节的证据交独立审查。
- 之后须负责人决定的：F-1 在正式 workflow 中的处置（可以与 N2 一起）；正式运行的授权（N2、C / T）。本轮不建立正式运行分支，不运行正式取证。

## 10. Review 154 的修复（2026-11-16；登记见 review-resolution §202，写代码之前提交）

Review 154 对固定代码点 `ed84d09` 为技术 NO-GO（B-154-1：启动的准入；B-154-2：实际对象的持有；B-154-3：H 的完整生命周期；B-154-4：LC-9 的完整错误消费）。负责人授权在 PR #58 内修复与回归（review-resolution §202）。修复的方案、共享的接点、回归的预期与验证安排写在 §202 的“本轮的实施登记”中，本节不重复；实现、回归在旧实现上的结果、新的固定代码点与证据，以及 N-154-1、N-154-2 对 §9 的更正，在之后追加的小节中记录。§1 ~ §9 不改写。

### 10.1 提交与固定代码点

| 提交 | 内容 |
|---|---|
| `7c97623` | Review 154 的原样归档 |
| `e3959ff` | review-resolution §202 的实施登记与本节的引言；写代码之前提交 |
| `2cdbc1d` | 能在旧实现上运行的回归（R1-a ~ R1-c、R2-a ~ R2-c、R4-a ~ R4-c），以及 `close-real-error`、`blocked-send` 的接缝中一个只供回归替换真实操作的入口；实现没有其他改动（回归先行的代码点） |
| `8b1f825` | `internal/toolctl`：B-154-1、B-154-2 与结构化的故障；R1、R2 的回归补上新状态的断言，新增 R1-d、R1-e、R2-d ~ R2-f |
| `a0ed8dc` | `internal/controls`：H 的完整生命周期（B-154-3）与 LC-9 消费故障集合（B-154-4）；`internal/realcall`：RC-1 ~ RC-3 消费故障集合；R3-a ~ R3-e；**本轮的固定代码点** |
| 之后 | 证据与记录：只改 `docs/` 与 AGENTS.md 的状态行 |

- **固定代码点** `a0ed8dc0800bc64b8f1c88e715f00fa600d053fd`：树 `9d120b675a08a18ede3fdd2b647c7839c971840e`，工具模块子树 `76a5c49c037b279ba783913cc95a56dd6e93c9eb`，测试 workflow 的 blob `4387acdc683a2014c235f3991bf854a30d6c7c2c`（本轮没有改 workflow）。
- 专用测试分支 `c9test/phase2` 以快进推送到 `a0ed8dc`。`ci.yml`、`c9-evidence.yml`、根模块的 `go.mod` 与 `go.sum`、工具模块的 `go.mod` 与 `go.sum` 都没有改；没有建立 `evidence/` 之下的分支；没有运行正式入口，没有采集 R。

### 10.2 回归先行：旧实现上的结果（本机验证 3）

在回归先行的代码点 `2cdbc1d` 上，Windows 原生与 WSL 中的 Linux 二进制只运行 `TestR154`（[local-3](evidence/wp2.5-c9-phase2/local-3/README.md)）：

| 回归 | 旧实现上的结果 |
|---|---|
| R1-a 入口的准入 | FAIL：`W − 4 s` 时仍启动伪工具 |
| R1-b 准备跨 `W` / `D` | FAIL：第一条管道在 `D` 之后返回，第二条管道与 `Start` 照样进行 |
| R1-c `Start` 尚未发起 | FAIL：过了 `W` 照样 `Start` |
| R2-a `Start` 失败保留 `Cmd` | FAIL：记录没有持有 `Cmd` |
| R2-b 移交之后的 panic | FAIL：写端没有关闭，读取任务一直等待，`Invoke` 在守卫内没有返回 |
| R2-c 两条管道之间的 panic | FAIL：第一条管道的两端保持打开 |
| R4-a 同站点、不同来源 | FAIL：旧的判定命中 |
| R4-b 另一必要操作的额外真实错误 | **PASS**：旧的展示行在预期为 ok 的站点显示 `error`，旧的判定已经不命中。§202 的登记预测旧实现命中，实际不是；它作为固定的拒绝预期保留 |
| R4-c 请求的错误换成另一类（Linux，登记之外新增） | FAIL：旧的判定命中（Windows 上 SKIP） |

依赖新增钩子、接缝或状态的回归（R1-d、R1-e、R2-d ~ R2-f、R3-a ~ R3-e）不能在旧实现上建立前提，没有在旧实现上运行，不计作“已在旧实现上动态复现”。

### 10.3 实现与 §202 登记的对照；偏离与补充

| 登记 | 实现 |
|---|---|
| B-154-1 启动的准入 | `toolctl.go`：`register` 在登记之后以 `admitLocked` 裁定入口（`entry`）；`startTask` 在两条管道与 `Start` 之前各裁定一次（`stdout pipe`、`stderr pipe`、`start`），余量为 `collect.CallMargin`（5 秒）。拒绝由 `refuseLocked` 记为 `not-admitted`、A 类（阶段 `not-admitted`），被拒绝的动作与之后的动作列为“未执行”；已取得的端由启动任务照常关闭。已准入的 `Start` 之后的协议（关闭写端、`PidfdOpen`、唯一的 `Wait`、pidfd 的封闭与关闭）不受截止约束。每次裁定都记入 `Admissions` 并发布 |
| B-154-2 实际对象的持有 | `holdLocked` 在 `Pipe` 返回的同一次锁下把 `*os.File` 记入记录；`Cmd` 在构造之后、`Prepare` 之前记入；`Process` 在 `Start` 成功之后记入；都不清除。`Desc` 增加 `Held` 与 `Release`（`none` / `held` / `pending` / `confirmed` / `unknown`）。`closeDesc` 用记录持有的对象，只关闭打开着的端一次。读取任务从记录取得读端。`recoverStart` 按阶段处理启动任务的 panic（§202 第 2 项的五种情形） |
| B-154-3 H 的生命周期 | 新文件 `lc9holder.go`：H 的持有结构、两次单独准入的管道、各端的登记与关闭状态、`Cmd`、就绪任务与唯一的 `Wait` 的计数、在锁下裁定的唯一释放请求（收尾或启动的一方）、终态的条件与规范行。LC-9 的收尾先停止 H 并在守卫之内等 H 的终态，再等工具记录的终态；实例的 `Terminal` = 工具记录 ∧ H |
| B-154-4 完整的错误消费 | `toolctl.Fault`（站点、来源、错误值与文字、是否晚到）与 `FaultLines`；`controls.Instance` 增加 `Faults`；`lc9.go` 的 `faultKind` 按错误的身份判定类型，接缝替代真实结果时记下被替代的真实结果，H 的故障一并进入；每个变体的完整预期写出全部故障行。RC-1 ~ RC-3 消费完整的故障集合，RC-2、RC-3 按平台写明结果 |

偏离、补充与更正（逐项保留）：

1. **`start-failure-not-executable` 在 Windows 上的真实错误**：预期第一次写作 `ERROR_BAD_EXE_FORMAT`（193），本机运行得到 `ERROR_EXE_MACHINE_TYPE_MISMATCH`（216），在写入代码点之前改正为 216；专用运行的 windows-latest 同为 216（命中）。这是观测到的平台事实，不是源码依据。
2. **R4-b** 在旧实现上已经不命中（见 §10.2），与登记的预测不同。
3. **R4-c**（Linux）是登记之外新增的回归：同一站点上请求的错误换成另一类（EBADF 代替 ESRCH）。
4. **R3-e** 是 §202 中 R3-d “另一用例返回错误”的实现：以整个 `output-not-closed` 变体运行，H 的就绪读端的真实关闭返回错误，判定不命中且理由中有该故障行。
5. **`HookBeforePidfdClose` 的位置**：由 `closePidfd` 内部移到启动任务调用它之前，使 panic 的出口关闭 pidfd 时不再调用测试钩子；既有交错的顺序不变（`sealed-before-reserve`、`closing-old-snapshot` 照样命中）。
6. **展示行的变化**：增加 `admission …`、`skipped …`、`cmd held …`、每个描述符的 `release … object …`、`panic no|yes`；`start` 一行区分 `not-initiated` / `not-returned` / `error` / `ok`（管道失败从 `start error` 改为 `start not-initiated`）；`WaitCalled` 改为“已发起”，另加 `WaitReturned`，`wait not-returned` 为新的一行。
7. **`overflow` 的回收结果**：§9.2 第 4 项的“不比较退出码”被取代，按平台写明（Unix 上 SIGPIPE 结束、退出码 −1、`Wait` 返回 `*exec.ExitError`；Windows 上退出 0）。
8. 测试接缝新增 `Start` 与两个钩子（`start task`、`before start`）；生产（正式入口）都为空。

### 10.4 专用测试运行（`c9-tool-test.yml`，attempt 1，没有重跑）

| 运行 | 代码点 | Ubuntu | macOS | Windows | 证据 |
|---|---|---|---|---|---|
| 43737429057 | `a0ed8dc` | **success** | **success** | **success** | [ci-43737429057](evidence/wp2.5-c9-phase2/ci-43737429057/README.md) |

- 本轮在专用环境中只有这一次运行，没有失败；第 1 ~ 3 次运行（§9.3）的记录保持原样。
- 本机验证：[local-3](evidence/wp2.5-c9-phase2/local-3/README.md)（回归先行，见 §10.2）与 [local-4](evidence/wp2.5-c9-phase2/local-4/README.md)（`a0ed8dc`：Windows 原生 506 条 PASS、2 条 SKIP；WSL 中的 Linux 二进制 508 条 PASS；vet 九种组合、格式、`go mod verify`、正式入口只编译都成立）。

### 10.5 固定代码点的结果（run 43737429057）

| 项 | Ubuntu | macOS | Windows |
|---|---|---|---|
| 步骤 A：单元测试 | 508 条 PASS（顶层 229，子测试 279），0 FAIL，0 SKIP | 507 条 PASS，1 SKIP（R4-c 只在 Linux） | 同 macOS |
| 步骤 A：`-race` | 508 条 PASS，没有数据竞争告警 | — | — |
| LC-9（完整预期含故障集合） | 20 个命中 | 11 个命中，9 个不适用 | 11 个命中，9 个不适用 |
| Review 154 的回归 | 全部通过 | 全部通过（R4-c SKIP） | 同 macOS |
| RC-1 | 两个配置的 28 项检查全部成立；六次调用都满足第 11 条，四个准入、对象的持有与释放都确认，故障为零 | 同左 | 同左 |
| RC-2 | 超量；SIGPIPE 结束；故障集合恰为 `wait operation exit-status` | 同左 | 超量；退出 0；没有故障 |
| RC-3 | pidfd 请求一次（nil）；SIGKILL；故障集合恰为 `wait operation exit-status` | 不发请求，清理未知；退出 0；没有故障 | `Kill` 一次（nil）；退出码 1；故障集合恰为 `wait operation exit-status` |
| RC-4 | 结论 4、退出 0；登记表 6 条全部满足第 11 条；118 个对照命中 | 结论 4、退出 0；109 个命中、9 个不适用 | 同 macOS |
| 残留进程的观测 | 没有 | 没有 | 没有 |

- 三个作业的身份步骤都核对 HEAD 等于 `GITHUB_SHA`、工作区干净，打印上面的树、子树与 workflow 的 blob；三个 runner 的 go1.27.3 的路径与 SHA-256 与第 3 次运行相同；Windows 的步骤 B 照旧使用 junction 的目标（F-1 的测试 workflow 安排，没有改）。
- RC-4 只说明健康路径上收尾的实际接线（见 §10.6 第 3 项）。

### 10.6 对 §9 的更正与限定（N-154-1、N-154-2、测试数量）

§9 的文字保留；以下逐项更正或限定：

1. **“之后的提交只改 `docs/`”**（§9.1）不准确：`ed84d09` 之后的 `9453eb2`、`8e228f8` 除 `docs/` 之外还改了 AGENTS.md（状态行的同步）。工具模块的子树与测试 workflow 的 blob 不变的结论成立。本轮固定代码点 `a0ed8dc` 之后的提交同样只改 `docs/` 与 AGENTS.md 的状态行。
2. **证据目录的更新**：`9453eb2` 把 `ci-43680692609/README.md` 中 workflow 一行由“见 `job-*.log` 的身份步骤”改为写明 blob `6b38ea8526bf12b69a0b611d1da878b5250f3c8d`，并相应更新同一目录 `SHA256SUMS` 中 `README.md` 一项（`aa2c0dd0…` → `37fb1962…`）。该目录的原始日志、`tar.gz`、`ARTIFACTS.sha256`、`run.json`、`jobs.json` 与 `SHA256SUMS` 的其余各项都没有改；其他证据目录在归档之后没有改动。
3. **RC-4 的结论**（§9.4 “登记表、收尾与 `D` 的出口在真实调用下接得上”）限定为：实际运行的是健康路径上的收尾接线（登记表作为终态来源、它的问题进入结论、记录与 manifest 的关联）；到期、挂起与异常时的责任由接缝的单元测试与静态依据说明，RC-4 不覆盖，也不说明真实调用下全部 `D` 到期分支或 R 上的 120 秒预算。
4. **测试数量**：§9.3、§9.4 与本机验证 1、2 的“487 PASS”是日志中包括子测试的 `--- PASS` 条数，即顶层 210 个、子测试 277 个；本机 Windows 的 486 条 PASS 另有 1 条子测试 SKIP（符号链接）。真实调用的测试是 5 个顶层（RC-0 ~ RC-4）与 RC-1 的 2 个子测试，共 7 条。本轮为顶层 229 个、子测试 279 个（新增的 19 个顶层与 2 个子测试都是 Review 154 的回归）。
5. **N-154-1（F-1 的适用边界）**：固定 Go 1.27.3 的 `os/types_windows.go` 中有 `winsymlink=0` 的兼容分支（`GODEBUG`，旧行为把 junction 视为符号链接），第 2 次运行的 RC-0 记录 `GODEBUG` 为空，工具模块的 `go` 指令为 1.27，所以本次是**实际的默认行为**。F-1 的结论只适用于这次 windows-latest 上 setup-go 的安装布局（`C:` 到 `D:` 的 junction）与默认的 `winsymlink`，不推广为所有 Windows 环境都必然失败。正式 workflow 若在同样的安装、PATH 与运行条件下运行，仍须先处理这个已知的守卫失败；未来 runner 的布局须重新核验，不能由本次观测预言。正式 workflow 的处置留待正式运行的准备，本轮不改 `c9-evidence.yml`。
6. **F-2**：只登记分类的前提（Windows 上以已知扩展名的输入使 `Cmd.Err` 为 nil，于是 `Start` 之后失败为 C；没有已知扩展名而预查找先失败时为 A ②），不改分类的优先级；实施计划的闸门表在之后获授权的整理中补明输入前提。

### 10.7 证据的类别（本轮）

| 类别 | 本轮的内容 |
|---|---|
| 接缝 / 模拟 | 回归中的假时钟、闸门与钩子（延迟的调度、跨 `W` / `D` 的管道、停住的 `Prepare` 与 `Start`）；接缝制造或替代的错误与 panic；H 的管道、关闭与启动之后的接缝；R4 中替换的“真实”结果 |
| 实际执行 | 伪工具与 H 是真实的子进程（三个 runner 与本机）；R1-e 中晚返回的真实 `Start` 与之后的真实终止；R2-f 中真实启动而没有 `Wait` 的进程；三个 runner 上 RC-1 ~ RC-4 的真实调用与它们的故障集合；Ubuntu 上 `-race` 的运行 |
| 静态依据 | `winsymlink` 的兼容分支（N-154-1）；Unix 上写入破裂管道的 fd 1 使 Go 程序以 SIGPIPE 结束（`os/signal` 的文档，`overflow` 的预期） |
| 观测 | Windows 上非 PE 文件的启动错误为 216（本机与 windows-latest） |

### 10.8 没有覆盖、没有运行或仍保留的

- panic 的出口只以测试接缝制造（移交之后、两条管道之间、`Start` 中、`Wait` 之前）；标准库的 panic、`Wait` 之中的 panic、关闭之中的 panic 没有构造的测试（有代码，静态路径）。读取任务、到期任务与请求任务不恢复 panic（与原实现相同），它们的 panic 使测试进程失败。
- 登记表在 `SIGCHLD` 被忽略时的拒绝、`Wait` 返回而 `ProcessState` 为 nil：仍没有构造的测试。
- R2-f 按设计不补做回收：Unix 上该伪工具在测试进程结束之前是未回收的子进程，Linux 上它的 pidfd 保持打开，由记录持有。
- 交错由接缝固定，`-race` 只在 Linux 上；不等于全部调度都已覆盖。
- H 是测试自己的直接子进程，不是真实的孤儿领养或后代清理的证据。
- 正式入口只编译与 vet；R 没有被分析；正式运行的 120 秒预算没有测量；F-1 在正式 workflow 中的处置待决。
- 三个 runner 上都没有观测到残留进程，这不是清理的证据；E-10 (ii) 仍只是为本阶段测试接受的限定前提。

### 10.9 交接

- 固定代码点 `a0ed8dc` 与本节的证据交独立复审（Review 155，由负责人安排）；B-154-1 ~ 4 记为“已修复，待独立复审”，本节不宣布关闭。
- PR #58 不合并；Auto-fix、Auto-merge 保持关闭。正式取证、正式运行分支、N2、正式运行的 C / T、G 项、精确例外、第三批的生产实施与 WP2.5 / M2 的验收都未授权。

## 11. Review 155 的修复（2026-11-17；登记见 review-resolution §204，写代码之前提交）

Review 155 对固定代码点 `a0ed8dc` 为技术 NO-GO：B-154-1 关闭；N-154-1／2 完成；B-154-2、B-154-3 共享一个残余 B-154-2/3-R-H（局部登记表、LC-9 的 H 与异常实例没有直到测试进程结束的根持有关系）；B-154-4 有两个残余：B-154-4-R-S（LC-9 的比较链丢失信号退出的区别）与 B-154-4-R-C（RC-3 把“父端尚未 `Wait`”当成“`go` 仍在运行”）。负责人授权在 PR #58 内修复与回归（review-resolution §204）。方案、共享的接点、回归的预期与验证安排写在 §204 的“本轮的实施登记”中，本节不重复；实现、回归在旧实现上的结果、新的固定代码点与证据在之后追加的小节中记录。§1 ~ §10 不改写；§10.8 中“由记录持有”的表述，其持有期限的限定见之后的小节。

### 11.1 提交与固定代码点

| 提交 | 内容 |
|---|---|
| `e8d0908` | Review 155 的原样归档 |
| `54f60f8` | review-resolution §204 的实施登记、本节的引言与状态行；写代码之前提交 |
| `991e329` | 能在旧实现上运行的回归（H1、H3、S2 ~ S5、C2）与它们用到的伪工具的三个新模式（`flood-term`、`flood-exit`、`term`，测试设施）；控制协议与 LC-9 没有改（回归先行的代码点） |
| `5e6c3a4` | 三个残余的修复与只在新实现上运行的回归（H2、H4、H5、S1、S6、S7、C1）；**本轮的固定代码点** |
| 之后 | 证据与记录：只改 `docs/` 与 AGENTS.md 的状态行 |

- **固定代码点** `5e6c3a4b93b5189c807bee4d179494e837a02b30`：树 `b6ac8721dedcd6d152aaab4facf74cb7f616e2b6`，工具模块子树 `755ffc0c2465959fa5528d07b5c079154e8e4feb`，测试 workflow 的 blob `4387acdc683a2014c235f3991bf854a30d6c7c2c`（本轮没有改 workflow）。
- 专用测试分支 `c9test/phase2` 以快进推送到 `5e6c3a4`。`ci.yml`、`c9-evidence.yml`、根模块的代码与 `go.mod`、`go.sum`、工具模块的 `go.mod` 与 `go.sum` 都没有改；没有建立 `evidence/` 之下的分支；没有运行正式入口，没有采集 R。

### 11.2 回归先行：旧实现上的结果（本机验证 5）

在回归先行的代码点 `991e329` 上，Windows 原生与 WSL 中的 Linux 二进制只运行 `TestR155`（[local-5](evidence/wp2.5-c9-phase2/local-5/README.md)）。结果与 §204 登记的预测逐项相同：

| 回归 | Windows | Linux（WSL） | 旧实现的实际表现 |
|---|---|---|---|
| H1 局部返回之后的登记表（R2-f） | FAIL | FAIL | 完整的垃圾回收之后，登记表、记录、`Cmd`、`Process` 与四个文件都被回收 |
| H3 LC-9 返回之后的 H 与登记表 | FAIL | FAIL | 这次运行、登记表、H、H 的 `Cmd` 与就绪读端都被回收 |
| S2 `overflow` 换成 SIGTERM | SKIP | FAIL | 前提成立，旧的判定为命中 |
| S3 普通的退出 141 | PASS | PASS | 旧的判定已不命中（141 与 -1 不同） |
| S4 额外的真实错误 | PASS | PASS | 已被拒绝（Review 154 的修复） |
| S5 `expiry` 换成 SIGTERM（Linux） | SKIP | FAIL | 前提成立，旧的判定为命中 |
| C2 请求之前已经结束（Linux） | SKIP | PASS | 请求 nil、`exit 0`、没有故障：协议本来就有这一合法分支，旧 RC-3 的断言会拒绝它 |

- 弱指针在对象不可达之后变空是运行时的行为，语言不保证；两个平台上都变空，作为旧实现没有根持有的动态观测，不是语言层面的证明。
- 依赖新的持有者、新的规范行或新的判定的回归（H2、H4、H5、S1、S6、S7、C1）没有在旧实现上运行，不计作“已在旧实现上动态复现”。

### 11.3 实现与 §204 登记的对照；偏离与补充

| 登记 | 实现 |
|---|---|
| R-H 进程寿命的持有者 | 新包 `internal/keep`：包级的表按交付的顺序保存对象本身与登记（序号、种类、名称），没有移除或释放的接口。`toolctl.New` 在返回之前交付登记表（种类 `toolctl registry`，名称取自新的 `Options.Name`）；LC-9 的 `run` 在构造之后、任何准备之前交付这次运行（`lc9 instance`），登记表的名称为 `LC-9 <变体>`；`newHolder` 交付 H（`lc9 holder`）。`Outcome.Instance.Extra["owner"]` 记下三者的序号。H 的每个端另记释放的知识（`held` / `pending` / `confirmed` / `unknown`；关闭被拒绝时仍为 `held`）；H 的唯一 `Wait` 另存原错误、回收的知识与退出身份；`controls.Fault` 另存原错误值（不序列化）。没有补做 `Wait`、重试 `Close` 或补写回收 |
| R-S 退出身份进入判定 | `toolctl.Exit` 增加 `Signaled` 与 `Signal`（Unix 上由 `unix.SignalName` 取名，见 `exit_unix.go`；其余平台见 `exit_other.go`）；`ExitKind` / `ExitKindOf`；`Lines` 的回收行为 `wait <退出身份>`；`Fault.Exit` 保存 `*exec.ExitError` 的退出身份；第 11 条要求 `exited 0`。LC-9 的 `faultKind` 为 `exit-status <退出身份>`；`overflow`、`expiry`、`pidfd-late-publish` 的完整预期按平台写明退出身份；H 的 `Wait` 行写出退出身份。RC-2 按结构化的身份判定 |
| R-C RC-3 的合法分支 | 新的无构建标签文件 `internal/realcall/rc3_test.go`：`judgeRC3`（纯函数，§204 的五个合法分支）、`rc3Facts`（分开记录的事实）、共享的 `faultKind` / `faultSet`（RC-1 ~ RC-3 共用）与 C1。RC-3 只调用它、把事实与分支写入证据 JSON；`healthyControl` 照旧 |

偏离、补充与更正（逐项保留）：

1. H1、H2 的弱指针另含四个文件（比 §204 的列举多）。
2. `exitOf` 对 `unix.SignalName` 不认识的信号写作 `signal-<编号>`（本轮没有出现）。
3. 第 11 条的判定改为要求 `exited 0`（原为退出码 0）；问题清单中的“exit code N”改为“the tool's end: <退出身份>”。只多写出身份，不改规则。
4. RC-3 的公共条件（到期、晚到、唯一的 `Wait` 返回、两个输出读到 EOF、没有 `not-admitted`、注入、panic 或未执行的动作）移入 `judgeRC3`，测试体中原有的同类检查删去；RC-1 的故障分类改用共享的 `faultKind`，期望仍为零个故障。
5. `realcall/doc.go` 更新：没有构建标签时该包另编译 `rc3_test.go`，所以步骤 A 也运行 C1（构造的快照，不启动进程）。
6. C1 有 23 个构造用例：5 个合法分支（Linux 两个、Windows 两个、darwin 一个）与 18 个被拒绝的组合。
7. S6 在 Unix 上另包含 `overflow` 自己的伪工具（`flood`，SIGPIPE），与 S2、S3 的结束两两比较。
8. 本轮新增的顶层测试 13 个（toolctl 3、controls 8、keep 1、realcall 1），子测试 23 个（C1）。

### 11.4 专用测试运行（`c9-tool-test.yml`，attempt 1，没有重跑）

| 运行 | 代码点 | Ubuntu | macOS | Windows | 证据 |
|---|---|---|---|---|---|
| 43906849891 | `5e6c3a4` | **success** | **success** | **success** | [ci-43906849891](evidence/wp2.5-c9-phase2/ci-43906849891/README.md) |

- 本轮在专用环境中只有这一次运行，没有失败；第 1 ~ 4 次运行（§9.3、§10.4）的记录保持原样。
- 本机验证：[local-5](evidence/wp2.5-c9-phase2/local-5/README.md)（回归先行，见 §11.2）与 [local-6](evidence/wp2.5-c9-phase2/local-6/README.md)（`5e6c3a4`：Windows 原生 538 条 PASS（顶层 237、子测试 301）、6 条 SKIP；WSL 中的 Linux 二进制 544 条 PASS（顶层 242、子测试 302）；vet 九种组合、格式、`go mod verify`、正式入口只编译都成立）。

### 11.5 固定代码点的结果（run 43906849891）

| 项 | Ubuntu | macOS | Windows |
|---|---|---|---|
| 步骤 A：单元测试 | 544 条 PASS（顶层 242，子测试 302），0 FAIL，0 SKIP | 541 条 PASS（顶层 239），3 SKIP（只在 Linux） | 539 条 PASS（顶层 237），5 SKIP（只在 Linux 或 Unix） |
| 步骤 A：`-race` | 544 条 PASS，没有数据竞争告警 | — | — |
| LC-9（完整预期含退出身份与故障集合） | 20 个命中 | 11 个命中，9 个不适用 | 11 个命中，9 个不适用 |
| Review 155 的回归 | 全部通过 | 适用的全部通过 | 适用的全部通过 |
| RC-1 | 两个配置的 28 项检查全部成立；六次调用都满足第 11 条，故障为零 | 同左 | 同左 |
| RC-2 | 超量；`signaled SIGPIPE`；故障集合恰为 `wait operation exit-status signaled SIGPIPE` | 同左 | 超量；`exited 0`；没有故障 |
| RC-3 | 分支“由请求结束”：请求经 pidfd 1 次、nil；`signaled SIGKILL`；故障恰为 `wait operation exit-status signaled SIGKILL` | 分支“没有请求（darwin）”：清理未知恰为一条；`exited 0`；没有故障 | 分支“由请求结束”：`Kill` 1 次、nil；`exited 1`；故障恰为 `wait operation exit-status exited 1` |
| RC-3 的共同事实 | 父端在 `W` 到期，调用判定失败，第 11 条不成立，有晚到的结果；唯一的 `Wait` 返回、B 类 | 同左 | 同左 |
| RC-4 | 结论 4、退出 0；登记表 6 条全部满足第 11 条；118 个对照命中 | 结论 4、退出 0；109 个命中、9 个不适用 | 同 macOS |
| 残留进程的观测 | 没有 | 没有 | 没有 |

- 三个作业的身份步骤都核对 HEAD 等于 `GITHUB_SHA`、工作区干净，打印上面的树、子树与 workflow 的 blob；三个 runner 的 go1.27.3 的路径与 SHA-256 与第 3、4 次运行相同；Windows 的步骤 B 照旧使用 junction 的目标（F-1 的测试 workflow 安排，没有改）。runner 镜像：ubuntu-26.04（20261111.82.6）、macos-26-arm64（20261102.0473.1）；内核版本没有记录。
- RC-3 本次发生的是“由请求结束”（Linux、Windows）与“没有请求”（darwin）；“请求之前已经结束”的分支本次没有发生，只由 C1 的构造用例与 C2（Linux 上的伪工具：本机 WSL 内核 6.18.33.2 与 ubuntu-latest 都取得请求 nil、`exited 0`）覆盖。Windows 上的这一分支没有确定性的前提可以动态构造。

### 11.6 对 §10 的限定与撤回

1. **§10.8 “由记录持有”的持有期限**：在 `a0ed8dc` 上，记录持有实际对象，但登记表本身只由创建者的局部变量与它的任务引用；测试函数返回、任务全部结束之后没有根（本机验证 5 的 H1、H3 在两个平台上观测到被回收）。所以 §10.8 中“R2-f 的进程、`Cmd` 与 pidfd 由记录持有”只在登记表可达的期间成立，不能读作“直到测试进程结束”。自 `5e6c3a4` 起，登记表、LC-9 的运行实例与 H 都由 `keep` 持有到测试进程结束。
2. **§10.5 与 §202 第 4 项中 RC-3 的预期**：“请求在 `go` 仍运行时发出，所以 Linux 必为 SIGKILL、Windows 必为退出码 1”的推论撤回：暂停的是父端，不是子进程。§10.5 的 RC-3 结果保留为那次运行的观测；RC-3 的判定改为 §204 的合法分支（`judgeRC3`）。
3. **退出码的展示**：§9、§10 中 `wait exit -1` 一类的展示行与预期是当时的格式；自 `5e6c3a4` 起为退出身份（`signaled <信号名>` / `exited <退出码>`）。旧的证据不改写。

### 11.7 证据的类别（本轮）

| 类别 | 本轮的内容 |
|---|---|
| 接缝 / 构造 | C1 的构造快照；S4 中替代 `stderr-r` 关闭结果的错误；H3、H4 中替代 H 就绪读端关闭结果的错误；S5、C2 中停在 `Wait` 之前的钩子 |
| 实际执行 | 伪工具真实的结束：SIGPIPE（S1、S6 与 `overflow`）、SIGTERM（S2、S5、S6）、退出 141（S3、S6）；C2 中对已结束而未回收的伪工具的真实请求（本机 WSL 与 ubuntu-latest）；H1 ~ H4 中真实对象的回收与持有；三个 runner 上 RC-1 ~ RC-4 的真实调用与 RC-3 实际发生的分支；Ubuntu 上 `-race` 的运行 |
| 静态依据 | Go 1.27.3 `os/exec_posix.go`（信号结束的退出码为 -1，`String` 区分信号）与 `os/exec_windows.go`（`Kill` 为 `TerminateProcess(h, 1)`）；微软关于进程结束之后 `TerminateProcess` 以 ERROR_ACCESS_DENIED 失败的文档；主线内核 `kill_pid_info` 与 `prepare_signal` 的推理（对已组退出而未回收的进程发信号返回 0 且被忽略）；`unix.SignalName` |
| 观测 | 旧实现上对象被回收（运行时的行为）；RC-3 本次发生的分支；C2 在两个 Linux 内核上的结果 |

### 11.8 没有覆盖、没有运行或仍保留的

- `keep` 只给责任一个所有者，不是清理的证据：R2-f 仍不补做 `Wait`，它的伪工具在测试进程结束之前是未回收的子进程；进程结束之后由系统处理，不补记为测试进程内的回收。
- Windows 上“请求之前已经结束”的分支只由构造的结果与文档覆盖；Windows 上 `Kill` 的退出码 1 与工具自己的 `exit(1)` 在退出状态上无法区分（`expiry` 靠伪工具的存活前提与请求的结果区分）。
- 读取、到期与请求任务不恢复 panic；标准库的 panic、`Wait` 或关闭之中的 panic、`SIGCHLD` 被忽略时的拒绝、`ProcessState` 为 nil：仍没有构造的测试（§10.8）。
- 交错由接缝固定，`-race` 只在 Linux 上；不等于全部调度都已覆盖。H 是测试自己的直接子进程，不是真实的孤儿领养或后代清理的证据。
- 正式入口只编译与 vet；R 没有被分析；正式运行的 120 秒预算没有测量；F-1 在正式 workflow 中的处置与 F-2 的文字同步待决。
- 三个 runner 上都没有观测到残留进程，这不是清理的证据；E-10 (ii) 仍只是为本阶段测试接受的限定前提。

### 11.9 交接

- 固定代码点 `5e6c3a4` 与本节的证据交独立复审（Review 156，由负责人安排）；B-154-2/3-R-H、B-154-4-R-S、B-154-4-R-C 记为“已修复，待独立复审”，本节不宣布关闭。B-154-1 保持关闭，N-154-1／2 保持完成。
- PR #58 不合并；Auto-fix、Auto-merge 保持关闭。正式取证、正式运行分支、N2、正式运行的 C / T、G 项、精确例外、第三批的生产实施与 WP2.5 / M2 的验收都未授权。
