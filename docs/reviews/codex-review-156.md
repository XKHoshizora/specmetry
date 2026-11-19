# Review 156：PR #58 三个残余的定向复审

## 1. 裁定

**本轮修复技术 GO；第二阶段在已授权、已审查的实现范围内技术 GO。当前不建议合并 PR #58。** 两个判断分别成立：三个实现残余可以关闭，但最终 HEAD 的常规 CI run **43908188853、attempt 1 为 failure（7/8）**，Windows 的 SQLite 测试失败尚未处理。本报告没有重跑或修复它，也没有把专用测试的成功折算成常规 CI 成功。

| 项目 | 裁定 |
|---|---|
| B-154-2/3-R-H | **关闭**。包级根保存实际登记表、LC-9 实例和 H；原对象与错误不随局部函数返回而失去持有者。 |
| B-154-2、B-154-3 | **在受审第二阶段范围内整体关闭**。沿用 Review 155 已认可的对象登记、异常责任、H 任务与判定接线，补齐进程寿命的根持有关系。 |
| B-154-4-R-S | **关闭**。退出种类、退出码和信号身份进入实际比较，SIGTERM 不再冒充 SIGPIPE／SIGKILL，普通退出 141 不冒充信号退出。 |
| B-154-4-R-C | **按本报告第 5 节的受控前提关闭**。不再从暂停父端 Wait 推导子进程存活；请求、退出、回收、错误和到期分别消费。五分支不构成一般性的因果识别器或所有环境的穷尽证明。 |
| B-154-4 | **在受审范围内整体关闭**。保留 Review 155 对来源、站点、次数及完整错误集合的限定，再加本次退出身份与 RC-3 修复。 |
| B-154-1、N-154-1／2 | 保持关闭／完成。没有发现本轮的直接反证。 |
| 新增 BLOCKER | **未发现有具体可达反例支持的新增实现阻塞。** N-156-1 是分支名称与证据解释的非阻塞精化；常规 CI failure 单独阻止当前合并建议。 |

这里的 GO 不认证正式 R、不说明正式 120 秒预算已经验证，不使任何 G 项或精确例外生效；也不授权正式运行、第三批生产实施或 WP2.5／M2 验收。

## 2. 固定身份、范围与登记

审查日期：2026-11-19。开始与结束分别读取本地 Git、工作区和 GitHub 当前对象，核验如下。

| 对象 | 核验值 |
|---|---|
| PR | XKHoshizora/specmetry #58，OPEN |
| 本地／远程分支 | `test/wp2.5-c9-evidence-phase2` |
| HEAD／PR head | `a37ef0a5765ac0f9ce52551f60f1e515f9a658c9` |
| HEAD tree | `edac63f9f5919c35430a1a288b0c98d2c6ec25c8` |
| base／PR base／远程 master | `09e81ee3a35a1fdcfd604f509851f1769e43fd3e` |
| 上轮受审 HEAD | `764da7e37d12476c28f0092daea4315f6b0cce9c` |
| 专用代码点／远程 `c9test/phase2` | `5e6c3a4b93b5189c807bee4d179494e837a02b30` |
| 专用代码点 tree | `b6ac8721dedcd6d152aaab4facf74cb7f616e2b6` |
| 两代码点工具模块子树 | `755ffc0c2465959fa5528d07b5c079154e8e4feb` |
| 两代码点测试 workflow blob | `4387acdc683a2014c235f3991bf854a30d6c7c2c`，`.github/workflows/c9-tool-test.yml` |

已读取当前 AGENTS、[Review 155](codex-review-155.md)、[处理记录](../review-resolution.md) §204／§205、[第二阶段记录](../wp2.5-c9-evidence-phase2.md) §11，以及[实施计划](../wp2.5-implementation-plan.md) §12.14 的持续持有、原始结果、到期和终判规则。沿实际增量追踪 keep、toolctl、LC-9/H、Judge、realcall 和 Job 消费者；没有重新展开未改的第一阶段机制。

提交顺序为 `e8d0908`（归档）→ `54f60f8`（§204 登记）→ `991e329`（旧逻辑上的回归与伪工具输入）→ `5e6c3a4`（修复）→ `a37ef0a`（证据及状态记录）。归档提交只增加 Review 155；当前归档为 **27834 字节**，SHA-256 **`7b81d1e273281e648ba24367ff9803c585a22455bdbbed3d63c43b5fb48b5280`**，归档与最终 HEAD 的 blob 均为 `dbf81742b61a2ee079e3572b1947d0db1eaabeb5`。这核实了仓库中的归档身份；没有另取一份仓库外原始副本作比较。

按原始字节比较，上轮 `review-resolution.md` 的 **1465918 字节是当前 1494634 字节的完整相同前缀**。§204 在代码之前登记，§205 追加结果，没有回改旧处理记录。状态仍为“已修复，待独立复审”，没有预先宣布本轮关闭。

独立比较 `5e6c3a4 → a37ef0a`：只有文档、归档证据和状态说明，工具模块与两个 workflow 均无变化；不是仅采用作者的“只改 docs”说明。上轮至本轮的技术增量限于独立工具模块，未修改冻结正文、验收、根模块生产代码、依赖或预算。实施计划等大段状态行追加本轮进度，其直接协议正文未改变。

`git diff --check` 的提示位于三份新归档作业日志：macOS 11 处、Ubuntu 12 处尾随空白，Windows 1722 行的 CRLF 被该检查报出。没有以消除提示为由改写原始日志；本轮源码与其他修订未出现对应提示。

## 3. R-H：实际持有关系及其边界

以下源码位置均指固定 HEAD `a37ef0a`；工具源码同时等同于专用代码点 `5e6c3a4`。

### 3.1 根引用与实际接线

- [keep.go](../../prototype/c9evidence/internal/keep/keep.go) 第 27～59 行：包级 `table.objs []any` 保存对象本身，`ents` 只用于登记。Hold、Get、Entries 共用互斥锁；没有删除、释放、清空或仅保留弱引用的路径。锁内不做 Close、Wait 或其他外部 I/O。
- [toolctl.go](../../prototype/c9evidence/internal/toolctl/toolctl.go) 第 386～410 行：每个 Registry 在 New 返回前交给 keep，之后 Registry 保留 Record，Record 保留实际 Cmd、Process、四端、pidfd 状态、原错误及任务状态。
- [lc9.go](../../prototype/c9evidence/internal/controls/lc9.go) 第 311～322、402～426 行：实例在任何准备前交给 keep；实例继续持有环境、夹具、管道端、Registry 和 H。Outcome 的 owner 序号只是可追溯索引，实际根对象不是这些整数。
- [lc9holder.go](../../prototype/c9evidence/internal/controls/lc9holder.go) 第 105～111 行：H 在工作开始前单独登记；第 59～96 行保留每端对象、原关闭错误、释放知识，以及 Cmd、Wait 原错误和回收知识。

因此，正常返回、准入拒绝、半构造、启动任务 panic、全部任务已经发布终态之后，都不会因局部栈退出而失去已经登记的对象。没有把“持有者仍有一个序号”替代成“实际资源仍可达”。包级表的锁没有引入与 Registry／H 的反向锁依赖；Get 返回对象后释放表锁，后续读取仍须遵守对象自己的锁。

### 3.2 释放、回收与失败消费

H 的 closeEnd（第 190～222 行）先登记 closing／pending，再在锁外调用既有 cleanupStep；拒绝保留 held，真实错误保存原值且置 unknown，nil 才置 confirmed。已开始或已关闭的端不再次 Close。waitTask（第 370～389 行）保存真实 werr，只有取得 ProcessState 才记 reap confirmed；无状态不冒称已回收。

H 的错误仍经 faultSet → Instance.Faults → 规范行 → Judge；H 的必要终态仍参与 Outcome.Terminal（lc9.go 第 419～426 行）。Judge 第 219～272 行没有因为 owner 存在而豁免未进入、未终结、完整预期不符或外层清理错误。原来“先保存被测拒绝，再释放 H”的第 396～402 行顺序保持。

R2-f 仍不补 Wait：recoverStart 与 H2 中保留 C／panic-before-wait、Wait 未发起、Exit 缺失、回收未知；Linux pidfd 仍打开且未假记关闭。keep 的根使这些实际责任保留到测试进程结束，**不是**证明子进程已经回收。测试进程结束后的系统处置，也不能补记成本进程唯一 Wait 的证据。正常已关闭对象同样保留，不因此重新取得动作权限。

### 3.3 回归的有效范围

H1／H3 使用独立 noinline 函数返回弱引用；退出局部函数、观察任务终态后执行 GC，再检查引用。旧实现上的实际回收可以证明那次运行确实失去根；新实现中 GC 后对象仍可得只证明这些观测。**到进程结束一直可达的依据是包级强引用及无删除路径的静态接线**，不是三次 GC 或测试名。

H2／H4 进一步从 keep.Get 取回同一个 Registry／Record／H／Cmd／文件，检查原错误和释放／回收状态；H5 检查顺序与对象身份。H3／H4 的 Close 错误是在真实 Close 返回 nil 后由接缝代替的错误，不是自然发生的句柄关闭故障，不证明实际泄漏。它验证了工具面对“报告关闭失败”的责任知识；真实操作若已失败，接缝不会抹除该错误。

任务计数归零或 Terminal 只代表相应最后状态已发布；报告不把它扩张成观察到了 goroutine 的最后一次退出调度。keep 也不证明正式资源预算成立。

## 4. R-S：退出身份确实进入比较与终判

实际链路为 ProcessState → exitOf → Exit／Fault.Exit → toolctl.Lines 及 controls.faultKind → Instance.Faults／完整 Lines → Judge。相关位置：toolctl.go 第 198～234、446～458、789～798、1232～1259 行；[exit_unix.go](../../prototype/c9evidence/internal/toolctl/exit_unix.go) 第 16～24 行；[exit_other.go](../../prototype/c9evidence/internal/toolctl/exit_other.go) 第 9～11 行；[lines.go](../../prototype/c9evidence/internal/toolctl/lines.go) 的 Wait 行；lc9.go 第 73～105、413～426、625～641、729～745、809～819、1030～1037 行。

Unix 同时保存 Signaled 与 Signal，未知信号名回退为 `signal-<数值>`，不会合并成一个泛化的“signal”。Windows 保留完整正常退出码。正常 `exited 141` 与 `signaled SIGPIPE` 是不同字符串；SIGTERM、SIGKILL、SIGPIPE 也分别参与 Wait 行和错误行比较。固定 Go 的 [exec_posix.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec_posix.go) 支持“信号退出的 ExitCode 都是 -1，因此仅比较 ExitCode 不足”的依据；本次没有继续依赖 -1 区分信号。

独立预期按伪工具行为写定：overflow 在 Unix 为 SIGPIPE，Windows 为 exited 0；expiry／pidfd-late-publish 在 Linux 为 SIGKILL、Windows 为 exited 1，darwin 无请求、释放 stdin 后 exited 0。它们不是从被测 ProcessState 反算预期。

S2 以一次可完成的写入后 SIGTERM 替代原 Flood，保留 overflow 前提；S5 在 EOF 与 pidfd 打开成立后推进 W，先前 SIGTERM 结束不因后来请求 SIGKILL 而被改写。S3 的正常 141、S4 的额外关闭错误、S6 的三种规范行及 S7（R1-e）的整体结果继续检查直接消费者。计数和重复错误未被去重；原错误值留在结构化对象中，来源／站点仍由完整行比较消费。

`successLocked` 新增 `exited 0` 条件后，四端关闭、两个 EOF、pidfd、请求状态、Expired、Unknown、Failures、Faults、Injected、Panic、Skipped、全部准入及晚到检查仍在。`problemsLocked` 把退出身份送入调用结果；Job 第 757～771 行继续消费登记表问题和 pending。因而零退出码不能遮盖其他必要失败，也没有放宽到期调用。

RC-2（realcall_test.go 第 280～308 行）改用同一结构化故障身份，但预期独立写定：Unix 允许精确 SIGPIPE＋唯一对应 Wait 错误，或确实自然 exited 0 且无故障；Windows 为后一类。另一个信号、普通非零退出或额外错误不能仅凭 overflow 命中。健康控制条件另行检查，没有被该分支替代。

## 5. R-C：五个分支、平台依据与因果限制

### 5.1 判定的实际范围

[rc3_test.go](../../prototype/c9evidence/internal/realcall/rc3_test.go) 第 102～161 行要求父端确实 Expired、CallResult 与 item11 都不是成功、有晚到发布、唯一 Wait 已调用且返回、B 类且有退出状态、两个输出完整无错、没有额外拒绝／注入／panic／skipped。随后对下表作精确比较。真实调用入口还调用 healthyControl，并先等待登记表 Terminal，核对四次准入、实际对象、四端释放与 Linux pidfd 的唯一关闭；纯函数本身不是整个控制协议的替代验证器。

| 平台分支 | 请求与结束组合 | 允许的故障集合／未知 |
|---|---|---|
| Linux 1 | ReqDone、Sends=1、请求 nil；SIGKILL | 仅 `wait operation exit-status signaled SIGKILL`，无清理未知 |
| Linux 2 | ReqDone、Sends=1、请求 nil；exited 0 | 无故障、无清理未知 |
| Windows 1 | ReqDone、Sends=1、请求 nil；exited 1 | 仅 `wait operation exit-status exited 1`，无清理未知 |
| Windows 2 | ReqDone、Sends=1、请求失败；exited 0 | 仅 `request operation TerminateProcess access-denied`，无清理未知 |
| darwin | ReqNone、Sends=0；exited 0 | 无故障，保留既定的一条“不发终止请求”的清理未知 |

请求没有准入、未返回或只登记未发起不能进入前四分支；Wait 未返回也不能进入。多一个关闭错误、同站点错误来源改变、同样失败但错误种类不同、故障重复，均不能匹配精确故障列表。rc3Facts 分开记录请求、子进程结束、Wait、父端决定、晚到事件和 unknown，没有把请求成功当作回收。

Windows 的 permission 分类使用 `os.SyscallError.Syscall == "TerminateProcess"` 和错误身份，不能把 DuplicateHandle 的权限错误当作请求已命中；固定 Go 的 [syscall_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/syscall/syscall_windows.go) 第 189～194 行说明它也接受 Go 的 EACCES／EPERM 抽象值。因此这是当前真实返回路径及构造测试使用的错误类别，不宜在将来扩展成任意错误包装的精确 Win32 码认证。本轮真实调用没有这些替身返回路径，未据此发现实际额外错误被放行。

### 5.2 独立平台核对

**Linux。** 核对固定 Linux v6.18 的 [signal.c](https://github.com/torvalds/linux/blob/v6.18/kernel/signal.c) 中 pidfd_send_signal → do_pidfd_send_signal → kill_pid_info_type／group_send_sig_info，以及 prepare_signal 第 877～883 行：已进入非 core 的组退出时，信号可被忽略而返回成功；不能把 nil 推成“杀死了仍活着的进程”。[exit.c](https://github.com/torvalds/linux/blob/v6.18/kernel/exit.c) 的组退出状态与 do_exit 中 exit_files 的顺序，支持 C2 对专用 silent 伪工具的前提推演。

该推演依赖实际子进程身份、尚未唯一 Wait、SIGCHLD 未被忽略／无其他回收者、pidfd 仍有效，以及受控工具没有改变这些条件。C2 的 silent 输入自身不提前关闭 stdout/stderr；因此它先观察两个 EOF，再发请求，是可审查的“已经进入退出而尚未回收”对照。此结论不能推广成所有 pidfd、所有僵尸状态、所有信号／内核都必然 nil。

本机 WSL 日志记录内核 `6.18.33.2-microsoft-standard-WSL2`；Ubuntu runner 也实际运行 C2 并得到 nil／exited 0，但归档未记录其内核版本。准确口径是“两处 Linux 环境有相同观测，其中一处内核版本已知，加上固定主线源码依据”，不是已认证两个确切内核构建或所有 Linux 版本。

**Windows。** 固定 Go [exec_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec_windows.go) 第 55～74 行显示：取得句柄使用责任后，Kill 复制终止句柄并调用 TerminateProcess(handle, 1)；本回归把唯一 Wait 阻塞在调用前，所以不应先由该 Wait 把 os.Process 标记为完成。微软 [TerminateProcess 文档](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-terminateprocess) 说明跨进程调用是异步的，完成终止仍须等待；进程已终止后调用会以 ERROR_ACCESS_DENIED 失败。这分别给出“请求 nil、后来 Wait 取得 1”和“先自行正常结束、请求 access denied、后来 Wait 取得 0”的依据。

这里没有从请求返回时刻推导回收时刻。相反顺序、另一退出码或额外真实错误继续使受控测试失败。本轮没有取得可支持新增合法组合的动态材料，也未找到在当前实际接线和受控输入前提下应放行却被新断言拒绝的具体源码反例；这不是声明五个分支穷尽了任意 Windows 程序、外部干扰或所有平台实现。

**darwin。** 既定协议不发送终止请求。后续自然 exited 0 和 Wait 返回不追认此前到期为成功，也不清掉到期时已经登记的清理未知。新的判定保持该边界。

### 5.3 N-156-1：分支标签应改为观测事实，非阻塞

位置：rc3_test.go 第 29～42、53～54 行；第二阶段记录第 487、507 行；专用运行 README 的 RC-3 行。

`ended by the request` 比观测强，尤其 Windows 上 **请求 nil＋exited 1 不能单独区别 Kill 的 1 与程序自主 exit(1)**。§11.8 已披露这个限制，故不能把该分支或 artifact 中同名字段再当成因果证明。Linux `exited before the request` 也不宜被读成请求开始前所有退出步骤已经完成：能够支持的是组退出已经确定或相容的退出状态，以及随后唯一 Wait 的结果，不能从这一组合恢复完整时间线。

建议在获授权的文字收尾中，将两个标签改为事实组合，例如“request-nil / signaled SIGKILL”“request-nil / exited 1”“request-nil / exited 0”“TerminateProcess-access-denied / exited 0”，或明确标注它们只是模式名；同步注释及证据解读，不改写历史原始日志。该项不要求新增实验、不扩大允许错误，也不改变本轮判定：真实事实已分字段保存，调用一直被拒绝，现有文档明确保留 Windows 的身份歧义，未据名称给予额外执行或成功权限。因此列非阻塞精化，不把残余改名重复计数。

## 6. 原始证据与回归质量

### 6.1 本机 local-5／local-6

实际读取并核对 [local-5](../evidence/wp2.5-c9-phase2/local-5/README.md)、[local-6](../evidence/wp2.5-c9-phase2/local-6/README.md) 的原始日志、命令头、退出结果、代码点、树／子树、工作区计数和 SHA256SUMS。local-5 的 2 项、local-6 的 3 项哈希全部一致；没有运行这些命令。

| 记录 | 实际支持的结论 |
|---|---|
| `991e329`、local-5 Windows | H1／H3 失败；S3／S4 已通过；S2／S5／C2 平台不适用。退出 1 是预期的旧实现回归结果。 |
| `991e329`、local-5 WSL | H1／H3、S2／S5 失败；S3／S4、C2 已通过。S2／S5 原判定把不同信号当作 hit；不能把三个旧通过项目算成旧缺陷复现。 |
| `5e6c3a4`、local-6 Windows | 538 条 PASS＝237 顶层＋301 子测试，0 FAIL、6 SKIP。其中五项为平台限制，另一个是符号链接权限。 |
| `5e6c3a4`、local-6 WSL | 14 包退出 0；544 PASS＝242 顶层＋302 子测试，0 FAIL、0 SKIP。是 Windows 工具链交叉编译后在 WSL 运行的二进制。 |
| local-6 static | 九组交叉 vet、格式检查、mod verify、正式入口只编译均记录退出 0；不等于三个平台运行，也不是正式入口执行。 |

local-5 的实际 Git 增量只加入可在旧逻辑上运行的回归与伪工具模式，尚未改变被测控制／LC-9 判定。H2／H4／H5、S1／S6、C1 以及需新字段的断言只作为修复后证据，不写成旧实现上的动态失败。S7 是既有 R1-e 的身份断言增强，不另虚增一个新顶层测试。

日志记录 Windows Go1.27.3 路径及 SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`，`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`；local-5 前后各一个变更路径，local-6 前后各两个证据目录路径。记录提供代码关联，不是不可变二进制或历史执行环境认证，也不是审查者复现。

### 6.2 专用运行 43906849891

已独立取得[正式运行元数据与作业日志](https://github.com/XKHoshizora/specmetry/actions/runs/43906849891)，不是只读作者回执。push、attempt 1、三平台 success。三个实际 checkout 均为 `5e6c3a4`；身份步骤打印相同 tree、工具子树、workflow blob、干净工作区及对应 GITHUB_SHA。三作业均为 **Go1.27.3**；Ubuntu amd64、macOS arm64、Windows amd64。目标配置包含另一架构，不把它描述成宿主也运行了另一架构的进程。

| 作业 | 步骤 A | Linux race | 步骤 B |
|---|---|---|---|
| Ubuntu `133437885177` | 14 包，544 PASS（242＋302），0 SKIP | 14 包、544 PASS，无 DATA RACE 告警 | 31 PASS（6 顶层＋25 子测试），0 FAIL／SKIP |
| macOS `133437883940` | 14 包，541 PASS（239＋302），3 SKIP | 未运行 | 同上 |
| Windows `133437884693` | 14 包，539 PASS（237＋302），5 SKIP | 未运行 | 同上 |

macOS 跳过 `TestR154RequestOtherKind`、`TestR155ExpiryOtherSignal`、`TestR155ExitedBeforeRequest`；Windows 再跳过 `TestR155OverflowOtherSignal`、`TestR155OverflowSIGPIPEHit`。专用 Windows 的符号链接用例实际通过，不能把本机的权限 SKIP 套过来。

步骤 B 的 31 条包含 **C1 的 1 顶层＋23 构造子测试**以及 RC-0～4 的 5 顶层＋RC-1 两个配置；不能把全部 31 条都称为真实 Go 调用。日志和作业步骤顺序均为先 A（Ubuntu 包括 race），再 B；不是由最终 success 倒推该顺序。

LC-9 的 Linux 20 个命中，其他两平台 11 个命中、9 个不适用；不适用不是 SKIP 文本或错误豁免。RC-2 原始记录为 Unix SIGPIPE、Windows exited 0。RC-3 原始 JSON 的事实为：Ubuntu 请求一次 nil／SIGKILL；Windows 一次 nil／exited 1；darwin 无请求／exited 0 且保留平台 unknown；三者父端 Expired、调用失败、有晚到发布、唯一 Wait 已返回。Windows access-denied 分支没有动态命中；C1 只验证构造消费。Linux C2 在本机及 runner 动态命中，但不是实际 go 的 RC-3 分支。

RC-1 两配置三调用、RC-4 六条调用与作业健康收尾继续通过；RC-4 的结论 4／退出 0只针对临时夹具 `example.com/c9fx`。T2 参数／目标接线证据按前轮限定保留，不升级成正式 R 或全部输入兼容性证明。

### 6.3 artifact 与哈希层次

实际逐成员读取三个归档，核对 **13＋14＋13＝40 个文件**与 ARTIFACTS.sha256 全部相符；对目录 SHA256SUMS 的 **10 项**也全部相符。重新打包文件的 SHA-256 分别为：

| 文件 | SHA-256 |
|---|---|
| macOS tar.gz | `a71c01e062b8cec8d3da46300331e0f6b03bff30e6322165e01efc9d1963de59` |
| Ubuntu tar.gz | `aa55c1ca7ecdd639301e9e6a198bb6960a5e6928e16ecbaef49ae28475b90d04` |
| Windows tar.gz | `48402e7f255bd7cc1d29224a3ecfe51d72eb01da7b7a4ee5de5435e922d5e9f9` |

线上 artifact 元数据为 macOS `14245042864`、Ubuntu `14245592701`、Windows `14245962553`，均关联该 run／代码点且未过期；对应 ZIP digest 与在线上传回执相符。**本轮没有另下载原始 ZIP 对每成员重新认证**：本轮完成的是仓库重打包成员的独立哈希核对、原始作业日志与元数据关联，不声称 tar.gz 等于 ZIP，也不以这些哈希认证 runner 环境。

前两次专用失败、后续历史成功仍按 Review 154／155 的既有范围保留，本轮没有重新全量验证历史。没有把“未观测到残留进程”当作回收证据；读取、到期、请求任务 panic，SIGCHLD 忽略拒绝及 ProcessState 缺失等未动态构造路径仍按前轮的静态限定，不借三平台绿色宣布全覆盖。

## 7. 最终常规 CI：必须保留的 failure

已独立读取[run 43908188853](https://github.com/XKHoshizora/specmetry/actions/runs/43908188853) 的八份实际日志和元数据。pull_request、attempt 1、正式 **failure**。全部检出完整合并快照：

`0ff7c18fb11a1c1b6b766cc0560983760f0aed21`

GitHub Git commit 对象的两个父提交依次为固定 base、固定 HEAD，tree 为 `edac63f9f5919c35430a1a288b0c98d2c6ec25c8`，等于受审 HEAD tree。本结论没有仅使用 run.head_sha。

| 作业 ID | 作业 | 结论及实际证据 |
|---|---|---|
| `133442615197` | Go lint | success；Go1.27.4；lint `0 issues` |
| `133442615826` | Windows Go tests | **failure**；Go1.27.4；26 个包 ok，`internal/store/sqlite` 包 FAIL；10 SKIP |
| `133442615860` | Ubuntu Go tests | success；Go1.27.4；27 个包 ok；主测试使用 race，15 SKIP；之后独立 root CrossUser 8 项实际通过 |
| `133442616286` | macOS Go tests | success；Go1.27.4；27 个包 ok；7 SKIP |
| `133442615840` | Web | success；不安装 Go，不补写 Go 版本 |
| `133443367530` | Windows dashboard build | success；Go1.27.4 |
| `133443367663` | Ubuntu dashboard build | success；Go1.27.4 |
| `133443367716` | macOS dashboard build | success；Go1.27.4 |

Windows 的原始失败为：`TestSeqIsUniqueAcrossConnections`，`internal/store/sqlite/store_test.go:546`，`database is locked (5) (SQLITE_BUSY)`，该测试报告 9.72 秒；包级结果 65.973 秒。不是 CLI 超时，也没有证据可判为历史失败的同一根因。Ubuntu 与 macOS 同名测试在各自环境通过，不抵消 Windows 失败。

SKIP 的口径：三平台共有五项真实 fetch／旧 Git 条件测试和两个 M14 条件测试；Windows 另有三个不可读存储／来源测试，合计 10。Ubuntu 主步骤另有八项 CrossUser，合计 15；这八项随后在独立 root 步骤实际运行并通过，不能仍称全部未运行，也不能把主步骤直接记成通过。macOS 合计 7。Windows、macOS 没有 Linux 的 root CrossUser 步骤。

八个作业中七个显示 Go1.27.4，Web 无 Go。根模块 CI 没有运行独立 `prototype/c9evidence` 模块；其中的 Ubuntu race 不等于专用工具模块 race。当前失败尚无负责人处置结果，本报告不授权重跑、不认定偶发或某项维护事项已解决。**技术修复可采，当前合并仍应停下。**

## 8. 直接回退检查及后续条件

本轮没有更改准入余量、W／D、pidfd 预留／关闭状态机、唯一 Wait、真实错误与注入分层、第一阶段来源核验或 Finish 协议。新增根持有不会自行关闭／回收；更严格的退出身份也没有撤掉其余成功条件。Judge 的完整集合、额外错误和外层清理拒绝仍实际接线。未发现本轮修复造成的直接回退。

后续仍分开处理：

1. **当前 PR 合并前**：负责人处置最终常规 CI 的 Windows SQLite failure；保留首轮失败、固定代码点及实际处理范围。若决定修复或重跑，须按该决定另取得证据，本报告不代行。N-156-1 可在获授权的文档收尾中精化，不改历史证据。
2. **正式取证准备**：N2、正式 R、正式运行 C／T、运行场所与分支、正式 E-10 适用前提及清理未知代价仍须单独决定；本阶段 E-10(ii) 的接受仅限授权测试。F-1 的正式 workflow PATH／安装布局处置尚未实施，须按实际环境核验并审查；F-2 的不存在路径／扩展名分类措辞整理仍留在对应阶段。
3. **正式结果**：本阶段临时夹具、伪工具、构造快照、Linux race、GC 观测与真实三种调用各有范围；未运行正式入口、未采集正式 R、未测正式作业预算。不能据它们宣称 G 项满足、精确例外获批或依赖整体运行信任已被接受。
4. **生产及验收**：第三批生产实施、WP2.5／M2 验收仍未授权；第一阶段已关闭事项与旧维护项按原限定保持。

## 9. 审查执行与结束状态

本轮只做源码／文档静态阅读、Git 对象比较、只读哈希与归档读取、线上已有 CI／官方源码查询。没有运行 Go、测试、伪工具、真实调用、扫描器或实验；没有修改受审文件、workflow、PR、CI 或分支。

读取 Git 远端时，命令行 `git ls-remote` 遇到 DNS 解析失败；远程身份改由 GitHub connector 的 PR 与三个 ref API 独立读取确认，没有 fetch、重置或切换。正式 workflow 的历史运行列表没有取得新的全量核验，本报告只确认本轮两份 workflow 未改、取得的专用日志只运行授权夹具，及正式运行权限没有变化，不用缺失的查询补写全历史事实。

开始工作区干净；结束时本地 HEAD／tree／工具子树、远端两分支及 master、PR head/base 与上表一致，无身份漂移。唯一新增文件为本报告 `docs/reviews/codex-review-156.md`，保持未提交。审查结束后暂停，等待负责人决定。
