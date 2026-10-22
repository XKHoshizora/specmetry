# Codex Review 075：PR #33，WP2.5 冻结前隔离原型独立审查

## 一、结论

**原型 PR 技术结论：NO-GO，当前不建议合并。**

发现一项实质 BLOCKER：**B1，实验有效性门禁未兑现其声明的拒绝规则**。畸形或丢失的辅助报告可以被当作正常报告；要求 30 个后代的 Windows 场景，在实际没有创建任何后代时仍可得到 `confirmed`。这些问题已在仓库外用真实测试二进制和小规模定向变异复现，不是仅凭忽略错误的写法推断。

这不等于否定所有原型机制。正常管道收尾、受控入口封闭、Linux 排他候选、macOS 退出观察，以及 Windows Job 的部分失败处理，已有各自适用范围内的证据。但 **CI 正式成功、日志中写着 confirmed，与确认门禁足以拒绝无效实验，是两件事**。应先修复门禁，再恢复受影响的证据。

| 交付层次 | 本轮裁定 |
|---|---|
| A．PR #33 技术审查及合并建议 | **NO-GO**；先完成 B1 的最小修复和定向验证 |
| B．冻结前技术门槛 | T1～T6 得到程度不同的局部支持，**尚不能整体宣布完成**；见第五、六、十节 |
| C．产品决定 | D1～D9、D11、D13、D18、D19、D20 这 14 项仍未批准；C-4 与 D20 的关系还应明确 |
| 生产实现、规范冻结、工作包验收 | 本报告均不授权，也不宣布 |
| 新增接受风险 | **无**；M-14、M-15、M-16 保持未解决 |

不重开 Review 074 已关闭的准备文档问题，也不把尚未开发的 Runner、数据库写入或 CLI 接线当作本 PR 应已完成的实现。

## 二、固定身份、范围和工作区

- 审查日期：2026-10-23，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#33](https://github.com/XKHoshizora/specmetry/pull/33)。
- 固定 base：`ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`。
- 固定 HEAD：`932b4a80852210a9fcc04cb0baf79aa4de04a7cf`。
- 最终原型代码及 workflow：`475414228b995c8dbabeb35d78ce6c1dd9bd9a95`。
- 开始、结束核对远程 HEAD/base 均与上述身份一致；PR Open、未合并、`auto_merge=null`。没有换用其他提交。
- 本地 HEAD 与固定 HEAD 一致。开始时工作区干净，Review 075 文件不存在；本轮原仓库只新增这份未提交报告，没有切换、清理或覆盖其他任务内容。

阅读适用 [AGENTS.md](../../AGENTS.md)、[准备稿第 4 版](../wp2.5-check-runner.md)、[Review 071](codex-review-071.md)、[072](codex-review-072.md)、[073](codex-review-073.md)、[074](codex-review-074.md)、处理记录和本轮全部文件。位置均绑定固定 HEAD。

实际 base→HEAD 为 **28 文件，+4588/−4**，包括 22 个原型模块文件、一个专用 workflow、一个原型报告和四个状态/准备文档。提交身份：

| 提交 | 实际内容 |
|---|---|
| `50d69c5bd86b15d7fd165f2996e555ac0c575b3a` | 原型独立模块 |
| `bcf080f4caebdc3784fc5976cbd188a64f6a3895` | 原型诊断 workflow |
| `475414228b995c8dbabeb35d78ce6c1dd9bd9a95` | other-reaper、真实 Git 共享租约、句柄检查和两处标签修正 |
| `dda22bfabf798a1412a992bbf473e868f4ed5e91` | 原型证据文档 |
| `932b4a80852210a9fcc04cb0baf79aa4de04a7cf` | §89、AGENTS、实施计划、准备稿头部 |

**4754142→HEAD 确实只有五个文档文件变化**，没有原型代码或 workflow 变化。根模块的 `internal/`、`go.mod`、`go.sum`、Schema、迁移、`ci.yml` 和既有审查原文均未变；没有生产 Runner、Claim/推导规则变更、迁移 0009 或 SQLite 重试。差异空白检查通过。

原型只有 `_test.go` 和自己的 go.mod/go.sum，依赖为 x/sys v0.49.0；根模块 `./...` 不跨入该模块。常规 CI 日志也没有原型包。这个隔离成立，但不使原型自身的错误变得无关紧要。

## 三、BLOCKER B1：报告与实际工作量的无效门禁存在假通过

### 1．准确位置和规则依据

主要位置：

- [main_test.go 第 349～368 行](../../prototype/wp25/main_test.go#L349)：`readReport` 打开失败返回 nil；字段不足、PID 解析失败直接跳过；未检查 `Scanner.Err()`；未知事件不拒绝。
- [main_test.go 第 229～258 行](../../prototype/wp25/main_test.go#L229)：`done` 只检查成功解析出来的 `limit`，并可在其余预测成立时报告 confirmed。
- [main_test.go 第 372～384 行](../../prototype/wp25/main_test.go#L372)：`waitReport` 到期返回已有集合，未表达“报告错误/未满足数量”这一独立结果。
- [helper_test.go 第 85～95 行](../../prototype/wp25/helper_test.go#L85)：fanout 等待 PID 的结果没有检查，随后仍报告 `fanout`。
- [job_windows_test.go 第 230～255 行](../../prototype/wp25/job_windows_test.go#L230)：只要求一条 fanout 标记；不要求 30 个不同后代，不核对 `ActiveProcesses=31`，查询错误被丢弃；最后只断言五个 Job 到达零。
- [pipe_test.go 第 350～365 行](../../prototype/wp25/pipe_test.go#L350)：silent-holder 不检查 PID 数量、主进程实际退出状态和最终树确认结果是否满足本场景的有效性要求。

依据是原型自己的 [§2.3](../wp2.5-prototype.md#L60)：报告缺失/损坏、所需工作没有发生、期限或清理不能确认，应当 invalid，并使测试失败。用户本轮也明确要求核查这些门禁，不能只依靠正常运行的计数。

### 2．独立反例 A：真实辅助进程给出畸形报告，场景仍通过

在隔离副本中，仅把 helper 的一次报告写入改为：

```go
_, _ = f.WriteString(line + "\nlimit not-a-pid\n")
```

保留原来的合法 PID 行、真实进程、正常管道和终止逻辑。运行原有 `TestPipeSilentHolderReal`，没有用自己编写的替身场景代替它。

实际结果：**exit 0、confirmed=1、invalid=0**；三个真实主进程、三个后代都运行，已知六个进程最终结束。新增的畸形行被 PID 解析分支静默跳过，最终门禁没有拒绝损坏报告。

这证明的是“报告损坏未使实验失效”，不是声称此次真的发生了 60 秒超时；不能把注入的畸形 limit 行当作实际超时证据。

### 3．独立反例 B：要求 30 个后代，实际零后代，仍 confirmed

另一个隔离副本只把 fanout 的 `n := arg(0)` 改为 `n := 0`，其余测试不变。仍运行原有 `TestJobTerminateNotEnded`：五轮真实 Job，每轮只有主进程，**没有请求的 30 个后代**。

实际结果：**exit 0、confirmed=1、work=5、cleanup=5 processes gone**。正常 CI 的同一场景报告 cleanup 为 155 个进程；本反例只有五个，却通过同一门禁。

原因不是“所有进程都没启动”。五个主进程提供了非零 work，从而绕过通用的 `work==0` 检查；场景专属的 31 进程负载要求没有成为判定条件。它可以支持“五个单进程 Job 结束”的观察，不能继续证明原先声明的负载实验。

### 4．独立门禁校准和静态补充

另加仓库外的最小校准测试，调用原来的 `begin/newReport/done/summarize` 链，而不改这些函数：

| 校准输入 | 实际结果 | 应有解释 |
|---|---|---|
| `limit not-a-pid` | confirmed、exit 0 | 畸形记录应 invalid |
| 超过 Scanner 默认长度的行，后接一条完整 limit | confirmed、exit 0 | Scanner 错误被丢弃，后续事件未读到 |
| 创建后删除报告文件 | confirmed、exit 0 | 最终读取失败没有传播 |
| 未定义的事件名 | confirmed、exit 0 | 协议白名单未验证 |
| 声明必须有 work，但为零 | invalid、exit 1 | 这一通用门禁有效 |
| 明确预测不匹配 | refuted、exit 1 | 不匹配的传播有效 |
| 未运行任何场景 | 汇总拒绝、exit 1 | TestMain 的零场景门禁有效 |

校准测试中的合成事件不代表真实进程；它们没有按报告中的数字操作任何 PID。完整真实链证据由前两个反例提供。

观察/清理错误也有同类静态缺口：[Linux allPIDs](../../prototype/wp25/tree_linux_test.go#L45) 丢弃 ReadDir 错误，`groupLive` 可把读取失败变为空集合；[Unix pidProc.ended](../../prototype/wp25/tree_unix_test.go#L105) 把 `procInfo` 的所有失败当作进程已结束。后者在 Linux/Darwin 都没有区分可靠不存在和读取/权限失败。由此不能无条件宣称“未知清理状态一定 invalid”。本轮没有在 Linux/macOS 动态注入这些系统错误，也没有因此声称已观察到误杀。

### 5．实际影响和最小修复

**影响**：用于冻结技术规格的实验，可以在关键见证损坏或目标负载缺失时被计为 confirmed；还可能把“没有获取到进程信息”计为清理已核验。该 PR 的主要交付就是可信实验，因此这是原型层面的阻塞，不是要求它提前实现生产 Runner。

最小修复建议：

1. 报告解析返回明确错误，严格验证事件、PID、字段、完整行及 Scanner 错误；最终清理后的读取必须完整。轮询中的“尚未到齐”可与损坏分开，但不能在终判时静默丢弃。
2. helper 的写报告失败、等待所需数量失败和异常退出应传播到父进程；不要用一个标记代替全部工作已发生。对需要的报告/进程数量建立场景专属门禁。
3. Windows fanout 每轮检查不同后代的数量、身份获取结果以及成功的 Job 活跃计数；本场景要求 31 个进程时，应明确验证。不得只凭累计 work 非零放行。
4. 进程观测和清理至少区分“可靠已结束/可靠仍存在/未知”。未知保留失败状态，不能对归属不明对象补发信号。
5. 增加以上精确反例及健康正对照；修复后恢复受影响场景的诊断证据。无需扩展为生产 CLI/数据库测试或大规模压力实验。

没有证据说明既有两轮 CI 中所有正常记录都实际损坏。其原始观察仍可逐项引用；失效的是“当前门禁已经证明无效实验绝不通过”的总括保证，不应把所有已有机制证据一概抹掉。

## 四、其余有效性、期限与清理核查

判定结构总体清楚：四种 verdict 与四种 kind 分开；`unsupported` 不记作 confirmed；`refuted/invalid` 使测试失败；已失败的测试在 done 中不被包装成正常预测通过。workflow 的 go test 退出状态由 `PIPESTATUS[0]` 传播，Results 步骤的 grep 不是绿色结论的来源。

“反例 confirmed”只表示朴素方法的反例复现。例如任意阻塞 Writer、停止复制导致写满、旁路子进程、其他回收者，并不支持这些被反例否定的机制。

期限方面须准确保留以下区别：

- helper 的 60 秒 AfterFunc 只有执行到 Go 代码后才生效；**CREATE_SUSPENDED 状态下不能依赖它**。Windows 受测管理失败分支实际由父进程的终止/句柄等待预算处理。
- 15 分钟 go test 和 20 分钟 job 期限使诊断作业有总上限，但测试进程被超时终止不等于它已经运行 defer、回收所有子进程。
- 当前 Windows 正常失败分支使用有时限 Wait，并登记直接子进程供清理；这支持这些已执行分支，不支持“每个尚未恢复的 helper 自身也有有效定时器”的说法。冻结 T4 时应明确独立于用户代码的父端期限、提前异常及所有权移交规则。
- Linux 闸门的 Cond 等待、版本查询的 `output()` 和若干原生 API 本身没有场景级可取消包装；不能把正常样本中的耗时说成无条件最坏时限。生产准入/取消规格仍须定清，不要求本 PR 先做生产接线。

本轮没有故意杀死持有未登记挂起子进程的父进程，没有为了探测边界留下不可控进程。无法安全构造的失败不冒充已经动态检验。

## 五、T1～T6：逐项技术证据与边界

### T1/T6：输出与收尾

[dest.Write/seal](../../prototype/wp25/dest_test.go#L49) 把 closed 检查、追加和封闭放在同一个锁下；复制入口和所有者入口分开，没有锁外检查再追加的旧窗口。[runS6](../../prototype/wp25/dest_test.go#L245) 先正常 D 阶段，必要时 Close 读端，再用独立 J 阶段等待，最后无条件封闭本次入口。F1 跳过 D。

O1 只由完整 EOF 路径产生，O2 是返回但不完整，O3 是没有确认返回。容量错误会保留为 write 结果；启用 discard 后继续读取并丢弃，不伪装 O1。owner 的短写单独返回错误，测试没有追加第二份文档。

本轮在 Windows 独立运行 14 个原有场景，涵盖正常双流、F1、D/J、容量正反例、别名、并发、O3 登记和封闭后迟到写入，全部 confirmed。这里只证明合成目的端/管道机制；没有生产 JSON Schema 或 CLI 接线。别名流可以先有检查内容，单次文档整体写入也不使整个流变成一份 JSON。

`os.Pipe` 关闭读端在所测环境能结束阻塞 Read，不能推广成任意 Reader。C-2 的“非阻塞”应理解为 Go 支持关闭/取消读取的具体管道机制；Windows 不能简单套用 Unix O_NONBLOCK 描述。样本中 0.4 ms 以内的结果是观察最大值，不是调度上界或硬实时保证。

O3 登记仍持有 copier、读端和已封闭入口；在 gate 释放后才结束的正对照成立。封闭不是任务回收。当前原型选择性释放了所有人为闸门；没有模拟真实永不返回的 Read 并声称资源已释放。

### T2：Linux 排他归属与身份

[procGate](../../prototype/wp25/exclusive_linux_test.go#L35) 的共享/排他计数、等待排他者优先、共享租约持有至 Cmd.Run 回收，以及并发检查串行化，在源码和 Linux 诊断日志中相符。测试确实包括真实 `git --version`；它证明使用**同一个 gate 实例**的启动不与检查重叠，不证明生产全部调用点已经接入。

空基线包括僵尸；先证明区间内没有其他子进程来源，才把后来收养者归入检查。pidfd 只绑定身份，不决定用途。清扫逐轮收养、核对父进程和 start time、通过 pidfd 发信号，再 poll/waitid 回收；主进程排除在清扫之外，最后才由 Go Wait 回收。这与 [Linux subreaper 的最近祖先收养机制](https://man7.org/linux/man-pages/man2/PR_SET_CHILD_SUBREAPER.2const.html)及 [pidfd 的进程身份接口](https://man7.org/linux/man-pages/man2/pidfd_open.2.html)相容；这些接口本身不识别业务归属。

旁路启动和其他 `wait(-1)` 的反例正确揭示了运行时检查不充分：不把“发现了这次违规”写成所有违规都可检测。源代码的读取错误缺口见 B1。C-9 仍需要完整的启动/回收/信号前提清单，不能只靠导入 os/exec 的名单。

本轮没有本机 Linux 动态重跑；Linux 38 个场景是独立读取的 CI 日志和静态核查证据，不冒充本机复现。没有强制 PID 复用或向未知组发信号。

### T3：macOS

[Darwin 实现](../../prototype/wp25/tree_darwin_test.go#L84) 用 kqueue 注册 NOTE_EXIT/NOTE_EXITSTATUS，等待使用 timespec；退出观察不回收。注册 ESRCH 时须另核对自己未回收的僵尸。waitid 原型采用原始系统调用、固定 siginfo 偏移；它的有效证据绑定 **Darwin 25.6.0、macOS 26.6.2、arm64**，不能自动覆盖其他 ABI/版本。静态依据另核对了 [Apple 的 kevent 手册](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/kevent.2.html)及 [XNU xnu-12377.1.9 的 siginfo 定义](https://raw.githubusercontent.com/apple-oss-distributions/xnu/xnu-12377.1.9/bsd/sys/signal.h)；历史手册和该源码版本不是本轮实际运行系统身份的替代。

成员枚举的 x/sys v0.49.0 包装会检查结构长度，遇 ENOMEM 重取；`groupLive` 将其错误向上传播，确认器不把这个错误直接当作空组。与此同时，通用 pidProc 清理的未知状态折叠仍属 B1；生产版本还须给枚举重试和错误分支保守出口，不能用没有错误的正常枚举样本证明任意权限状态。

27 个场景在两轮真实 macOS 诊断中执行，无 SKIP。组长未回收、组信号、Wait 的顺序有证据；只剩僵尸组长时的 EPERM 是所测系统观察，不能解释为组已消失，也不能推广成所有平台的普通 EPERM 都无害。`setsid` 后代仍存活的边界已如实保留，D7 没有因此获批。

本轮没有 macOS 本机环境；只做源码、依赖包装、官方接口依据和完整日志核查。原型的 arm64 raw ABI 正对照是 CI 证据，不是本轮另一次动态复现。

### T4：Windows

[startJob](../../prototype/wp25/tree_windows_test.go#L148) 实际顺序为建 Job、配置 KILL_ON_JOB_CLOSE、挂起 Start、打开自己的进程句柄、Assign、IsProcessInJob、Resume。失败注入不是先运行用户代码再假装失败。标记文件有正常恢复运行的正对照。[Microsoft 的 CREATE_SUSPENDED 定义](https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags)也说明主线程在恢复前不运行；所以 helper 内部定时器不能为这个阶段提供期限。

Assign/Confirm 失败由 TerminateProcess 加句柄等待处理；已在 Job 的 Resume 前失败由 TerminateJobObject 加 ActiveProcesses 查询处理。终止请求和确认分开；query/terminate 注入失败不会被协议当作已结束，场景另以自身句柄观察之后结束，这个区分正确。

本轮独立通过正常恢复、终止请求失败、查询失败三项原有场景。另一次包含启动失败链的运行，在第一个 FailAssign 的**事后句柄检查**处异常退出，详见第九节；不能把这一轮说成本机失败链全部通过。原 CI 的这三项成功仍按其环境引用。

原型没有观察到成功请求后的真实 ActiveProcesses>0 窗口；不能用注入代替该观察。31 进程负载的正常 CI 日志列出 155 个已结束对象，但其门禁抗缺失能力未通过本轮反例，见 B1。

此外，CreateJob/SetInformation/OpenProcess、线程枚举、真实恢复过程中的部分失败、终止失败的挂起状态等，不都等于已有四个 pre-resume 接缝。C-16 应限制为已测试分支，并在 T4 规格中完整列出剩余责任；不宜概括成所有原生失败点已得到验证。

### T5：解释器

[适配器测试](../../prototype/wp25/adapters_test.go#L57) 先执行 plain 健全性检查，再对 23 个 payload 比较实际解释器输出。不是仅用一个 argv 打印器推断 shell；MSYS/MSVCRT 负对照也确实执行 shell。

这些 payload 被放进 quoted here-document/here-string，支持的是这些脚本文本的往返。它们不穷尽所有 shell 命令、分隔符冲突、启动文件、参数、不同 Windows 安装路径或 PATH 策略。cmd 的独立反例只证明所测 LF/CRLF 多行样本与最终退出状态，不证明缺失内层命令不会被脚本覆盖。

本轮 cmd 原有场景通过。Git for Windows sh 的两个本机健全性调用退出 3221225794，均被正确标成 invalid；没有取得本轮的 23 字符串通过结果。没有修改环境拒绝、安装工具或为此反复尝试。三平台诊断中的适配器成功按各自已记录版本引用。

## 六、C-1～C-18 裁定

“支持”只针对相应机制及实际证据前提，不表示本审查把候选并入冻结规范。B1 修复后，还需恢复相应门禁的可信性。

| 候选 | 证据与技术裁定 | 冻结前还需明确/限制 |
|---|---|---|
| C-1 | poll 毫秒向上取整、kevent timespec 的方向成立；有早返缺陷历史和 timeout 正对照 | 超时等待不提前与事件先到可提前返回分开；不是硬实时保证 |
| C-2 | 支持 Go 自建管道 Close 中断 Read 的所测机制；阻塞模式反例有效 | 按平台描述取消能力，不把所有管道都叫 Unix 非阻塞；保留 O3 兜底 |
| C-3 | O3 任务保留读端/已封闭入口，登记直到结束，机制成立 | Read 返回不一定要等最后一个写端结束：一次数据也可使后续写入被封闭入口拒绝；不得把登记说成释放 |
| C-4 | 两种 admission 分支都被运行，但只是候选策略，不是各自可用性代价已获批准 | 与 D20(a)/(b)、当前批次和同进程新调用分开，见 N1 |
| C-5 | 容量耗尽后继续读并丢弃的正反对照充分支持这个局部机制 | 不保存丢弃内容，保留 O2；与普通真实文件输出路径分开 |
| C-6 | closed/append/seal 同锁、独立 owner 入口成立；本轮独立正对照支持 | “不阻塞”限无外部 I/O/回调的有限追加，不承诺获取锁的实时上界；owner 错误仍可能部分写出 |
| C-7 | 任意阻塞 Writer 的反例成立 | 拒绝这类 Writer 属 D20(i) 的待批准支持契约；原型未实现生产预检 |
| C-8 | 单一 gate 实例的共享租约、排他和串行化得到 Linux 日志支持 | 生产全局唯一实例、取消/等待、所有启动路径、F1 后共享租约策略须定清；不能把 stand-in 当作 gitx 已接入 |
| C-9 | 原型正确证明仅靠运行时不充分；静态规则尚未实现 | 补齐所有启动与回收符号/平台路径及依赖约束，见第十节；不是只列 os/exec 导入即可 |
| C-10 | Go/SIGCHLD 的所测启动行为与原型正反例相符 | 运行中不能由别处改变 disposition/自动回收前提；signal.Ignored 不是任意宿主全部信号配置的证明 |
| C-11 | 发组信号前检查未回收组长是必要保护，other-reaper 反例有效 | 它不是原子 check-and-signal；仍依赖无其他回收者，不宣传已强制 PID 复用验证 |
| C-12 | 信号返回值不等于结束；平台成员查询方向正确 | 结束确认还依赖 E-main、保留组身份、Linux 清扫等；读取错误不能为空，当前缺口见 B1 |
| C-13 | macOS kqueue/WNOWAIT 的 arm64 机制得到真实 CI 支持 | raw syscall ABI、注册/查询失败、枚举重试预算和支持版本应冻结；未检验版本不自动通过 |
| C-14 | pidfd 前后身份核对、有时限等待、无 PID 降级方向成立 | 归属前提独立；所有系统调用错误须保守处理；不支持 pidfd 的真实旧内核仍未测 |
| C-15 | 注入清扫放弃、保留组长和被收养者、下一次基线拒绝的局部证据成立 | 不是任意 F1 或任意嵌入宿主均安全；原型 F1 的组长实际已退出，不能概括仍存活主进程；不等于自动回收 |
| C-16 | 指定 pre-resume 失败分支和成功恢复链有证据 | 本机句柄验证异常、遗漏原生失败点、父端独立期限须分别处理；不能从无 marker 单独证明所有恢复失败都没运行 |
| C-17 | request/confirmation 分离成立，本轮独立注入对照通过 | 无真实异步窗口观察；不据此要求无目标重负载压力或批准边界 |
| C-18 | 所测版本的脚本文本往返和 MSVCRT→MSYS 反例提供有限支持 | 不是任意 shell 文本全称证明；查找/PATH/worktree 排除、安装布局仍需规格；D1～D3 仍待决定 |

## 七、NON-BLOCKER 与直接文档收口

### N1：C-4 不应脱离 D20 无条件称为“不是产品选择”

位置：[原型报告第 296、331 行](../wp2.5-prototype.md#L296)。O3 后的登记数据结构是工程机制；**拒绝同进程后续检查并返回 6** 则还要看获批的调用方支持契约和 D20。

选择 D20(a) 时当前批次本已停止，但它不自动决定宿主的新调用如何处置；D20(b) 若允许继续，C-4 的拒绝策略又必须明确优先关系。建议把 C-4 附入 D20 的支持范围说明，区分本次停止与后续调用、CLI 单次进程与长驻进程内调用。可推荐在登记非空时拒绝，但代价可能是该宿主持续不可执行检查，不宜由少量样本“未出现 O3”推为几乎没有代价。

不要求机械新增第 15 项产品编号。由于所有选项明确未批准，本项不单独阻塞原型合并；在冻结/授权前必须收口。

### N2：Windows 测试基座的句柄和失败证据边界

位置：[tree_windows_test.go 第 94～110 行](../../prototype/wp25/tree_windows_test.go#L94)、[job_windows_test.go 第 69～92 行](../../prototype/wp25/job_windows_test.go#L69)。ownPID 打开的后代观察句柄没有显式 Close 路径；进程结束的检查不等于观察句柄已释放。短命测试进程退出会回收资源，但不能把 cleanup 的 “processes gone” 解释为所有句柄均已主动关闭。

本轮未经修改的 FailAssign 在 `handleValid` 检查已关闭句柄处发生 `0xc0000008`，exit 2，无结果汇总。该观察应保留并调查验证方法的环境适用性；当前没有证据把它归因于某个 Go/Windows 根因或生产 Job 机制。可调整为能够安全核验释放结果的测试方式，并明确句柄所有者；本报告不修改实现，也不以这个本机异常推翻原 CI 的实际成功。

### N3：限定“硬时限”和支持版本的概括

[报告第 40 行](../wp2.5-prototype.md#L40) 的 helper 内部 60 秒时限不能覆盖未恢复执行区间；应与父端的有界终止/确认分开说明，见第四节。原生故障未确认时保留 F1，不能按辅助程序必然自行结束兜底。

同样，[报告第 350 行](../wp2.5-prototype.md#L350) 仍只列 Ubuntu 内核 6.17，而第二轮是 7.0；应与 §3.2 一致。两项都是准确表述和冻结规格收口，不要求为了原型提前完成生产 Runner。

§89 对本地草稿失败和成功计数总体忠实，既有审查和 §84～§88 没有改写。对 T2/T4/全部 confirmed 的总括，应在修复 B1 后按证据强度更新，不能把本报告写成已同意全部机制。

## 八、CI 独立核验

### 1．原型诊断，分 SHA 记录

两组正式 API 均为 **push、attempt 1、completed/success**，无重跑；每个平台实际 checkout 各自固定 SHA。

| Run/对象 | Ubuntu Job | macOS Job | Windows Job | 原始结果 |
|---|---|---|---|---|
| [39815658463](https://github.com/XKHoshizora/specmetry/actions/runs/39815658463)，`bcf080f4caebdc3784fc5976cbd188a64f6a3895` | 119953456628 | 119953457044 | 119953457084 | 37/27/29 confirmed；0 invalid/refuted/unsupported；无 SKIP |
| [39816938304](https://github.com/XKHoshizora/specmetry/actions/runs/39816938304)，`475414228b995c8dbabeb35d78ce6c1dd9bd9a95` | 119957760153 | 119957760105 | 119957760056 | 38/27/29 confirmed；0 invalid/refuted/unsupported；无 SKIP |

六份完整 Job 日志均独立读取。第二轮各平台场景 ID 集合与报告 §3.3 对齐，**并集为 53**，没有漏登或额外场景。38+27+29 是 94 次平台观察，不能称为 94 个独立场景。Results 步骤把结果行再打印一次，也不能重复计数。第一轮尚无 other-reaper，不能混用最终 38 的口径。

Ubuntu workflow 确实按 Linux 分支加入 `-race`；另两平台未开原型 race。Go 版本均为 1.27.3。最终 Ubuntu 记录为 kernel 7.0.0-1017-azure，第一轮为 6.17.0-1022-azure；macOS 26.6.2/arm64，Windows 10.0.26100。它们不是本机 Windows 运行。

两个诊断成功的事实保留，但 B1 的独立反例说明成功不能替代门禁有效性审查。作者 WSL 的 37 confirmed/1 unsupported 和本地 29 confirmed 属作者证据，本轮未重跑其完整集合。

### 2．最终常规 PR CI

[run 39818439605](https://github.com/XKHoshizora/specmetry/actions/runs/39818439605)：**pull_request，HEAD 932b4a8，attempt 1，completed/success，8/8**，没有跳过 Job。

八份实际日志均 checkout `c291ded4c36ac513fcc1666bd0a860908ce26cc9`。GitHub Git commit 对象与本地只读对象核对一致：

- 父提交：`ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`、`932b4a80852210a9fcc04cb0baf79aa4de04a7cf`；
- 树：`45c466d192042fb174e8e8ddab00e53d12a66bac`，与 HEAD 完全相同。

| Job | ID | 正式结果 |
|---|---|---|
| Go tests Ubuntu | 119963183793 | success |
| Go tests macOS | 119963184089 | success |
| Go tests Windows | 119963184591 | success |
| Go lint | 119963184296 | success |
| Web | 119963184423 | success |
| Build with dashboard Ubuntu | 119963366623 | success |
| Build with dashboard Windows | 119963366656 | success |
| Build with dashboard macOS | 119963366664 | success |

普通 Ubuntu 主测试走 `-race`；另行编译并 sudo 执行八项 CrossUser 是独立 root 步骤，不是同一 race 运行。Windows 动态分组仍为当前 33 包中的 32+1，CLI 使用 20 分钟，其余默认；没有改变预算。CLI：Ubuntu 511.606 s、Windows 487.400 s、macOS 180.953 s。

按日志 PASS/SKIP 行计，包含子测试：Ubuntu 3022/15（包括 root 补跑），Windows 2999/10，macOS 3017/7，无 FAIL。普通非 root 步骤的跨账户跳过与 root 实跑分开；真实旧/unsafe Git、平台权限、默认禁用诊断等条件跳过不计为通过。

**常规 CI 没有执行原型模块，不能用它给 T1～T6 增加通过证据。**

较早 [常规 run 39818092337](https://github.com/XKHoshizora/specmetry/actions/runs/39818092337) 对 4754142 的正式结果是 cancelled：三个 Go tests Job cancelled，lint/Web/三个 build 已 success。不能写成该轮全部通过或全部失败；与 workflow 的同分支并发取消规则及作者说明区分记录，没有重跑。

作者本地草稿中提前 poll、僵尸清理假通过、未先确认 Read 阻塞、Windows 计时粒度、场景并发锁等五项失败历史按原报告保留；本轮没有取得其原始草稿逐项独立复现，不将它们计为新通过。

### 3．基线核验

PR #32 实际合并为固定 base `ecb83ba`，父提交为 `bcbb21c169020e4140aa12fc8adf53477fd6245a` 和 `f54850ed0c53af4ab103dcb9c5d3d5bfeb4cf707`；GitHub PR 状态为 merged。master [run 39811328918](https://github.com/XKHoshizora/specmetry/actions/runs/39811328918) 为该 SHA 的 push、attempt 1、completed/success、8/8。本轮仅作基线核验，没有重新验收 PR #32。

## 九、本轮实际验证、失败和未验证边界

隔离副本由固定 HEAD 的 git archive 建立，全部在仓库外 review075 目录。使用已安装的 **Go 1.27.1、Windows/amd64、Windows 10.0.26300**，直接调用现有二进制，`GOTOOLCHAIN=local`、`GOPROXY=off`，没有安装或升级工具。

| 独立运行 | 结果及边界 |
|---|---|
| 原型模块编译 | 成功；未改原仓库 |
| T1/T6 定向原有测试 | 14 confirmed，exit 0；含 O3、双流、封闭、容量、别名、owner 短写及任意 Writer 反例 |
| 真实 helper 报告损坏变异 | 原有 silent-holder 仍 confirmed、exit 0；B1 的假通过 |
| Windows 零后代变异 | 原有五轮 Job 场景仍 confirmed、exit 0，work=5；B1 的假通过 |
| 六个报告/预测/work 校准 | 畸形 PID、Scanner、缺失文件、未知事件四项错误放行；零 work 和预测不匹配正确失败 |
| 零场景校准 | 正确 exit 1；不是“测试框架 PASS 就整次通过” |
| Windows 正常/terminate-fails/query-fails | 3 confirmed，exit 0；均有实际自有进程和清理记录 |
| Windows 启动失败链定向组 | 首项 FailAssign 的事后 handleValid 异常，exit 2；未执行完成其余项，没有 summary，**不计通过** |
| 本机 sh/MSVCRT 适配器 | 两项 plain 健全性调用退出 3221225794，被标为 invalid；未取得本机 23/23 结论，不推定根因 |
| cmd 适配器 | 1 confirmed；同一适配器组因上述两项 invalid 整体 exit 1 |

隔离目录第一次准备命令误指定了尚未创建的 workdir，进程未启动，返回目录无效；创建目录后才编译。这个工具准备失败不属于产品检出或原型通过。

定向测试使用测试二进制自己的 5/20/30/90 秒期限和外部额外等待上限；外部 watchdog 均未触发。进程操作只针对本轮启动的测试映像及其报告/句柄确认的自有对象。结束检查没有仍运行的 review075 测试映像/已知实验进程；没有终止未知进程。辅助报告注入没有使用任何外部真实 PID。

仓库外额外门禁校准文件 SHA-256：`53da5d43c7d48183a9aab774430760fa507a562ee80677e7ad6829ca1bb251c0`。变异和日志只作为本轮独立证据，没有移入原仓库，也没有被描述为作者 CI 已执行的测试。

**未验证**：本机 Linux/macOS 动态运行、真实不可中断睡眠、强制 PID 复用、真实控制台取消、宿主被强杀、服务/计划任务/WMI/COM 代启动、其他 Git for Windows 布局、完整生产启动/回收路径、真实事务和记录。没有为这些限制安装工具、绕过环境拒绝或反复施压。没有运行用户项目的检查配置，也没有重跑/触发任何 workflow。

## 十、冻结前与生产验收前的工作分界

### 1．冻结及实现授权之前

1. **先修复 B1**：严格报告/错误传播、精确负载见证、未知清理状态保守失败；用本报告的定向反例确认修复，并恢复适用诊断证据。无需重做所有已验收模块。
2. 收口 C-1～C-18 的前提与措辞，特别是 O3 的后续调用范围、父端期限、Windows 未测失败点、macOS ABI/枚举错误与预算、Linux F1 宿主状态。
3. C-9 的规则设计应覆盖实际启动和回收能力，而不只列 os/exec 导入：还包括平台 syscall/x/sys 包装、直接 os.Process.Wait、waitid/Wait4 的其他形式、会改自动回收前提的信号设置，以及内部/依赖调用。可以用受控入口白名单和构建平台扫描设计；完整生产 lint 接线在授权后兑现，不能先声称它已存在。
4. 确定全局 gate 的所有者、上下文采集/Git 与排他区间的先后、取消等待与 F1 后资源策略，避免未来同一 gate 内再申请共享租约形成自锁。原型只证明已写出的调度，不证明完整生产拓扑。
5. 完成 T5 冻结前的解释器来源/实际路径/相对 PATH/worktree 排除策略；字符串集成功不能替代这些规格。
6. 负责人明确选择以下 14 项，再按准备稿 §18 同步相关规范、验收和契约；产品决定不能代替修正技术错误。

| 产品项 | 仍需选择的内容及主要代价 |
|---|---|
| D1/D2 | 默认 shell、不回退/不可用时整批拒绝；Windows 入口、Git for Windows 来源及 PATH 改写，可能改变工具解析顺序 |
| D3/D4 | cmd 多行、sh/bash CR 的拒绝与旧配置人工 material 修订；不能静默改保存值和 hash |
| D5 | EV-04 按解释器最终状态或其他方案；命令串/管道掩盖的失败不会额外识别 |
| D6 | 树已结束但结束上下文读取失败时是否尝试保存不合格事实 |
| D7 | 真正管理范围、平台支持及排他条件不成立时行为；macOS setsid 边界不能由本审查接受 |
| D8 | 取消的记录与竞争截止、组合退出码；不能从无记录推断未执行 |
| D9 | 批次汇总规则，仍不改变逐条 runs(S)/V |
| D11 | 主进程正常退出后是否终止后台后代 |
| D13 | 整批固定还是逐项开始固定配置，对后续修订/退役的可见影响 |
| D18 | 空批次首次 worktree 登记及独立 seq 的副作用 |
| D19 | 树结束未确认时是否尝试不合格记录、时间字段语义；不是“已证明上下文变化” |
| D20 | 输出支持范围、转发失败后的记录/停止/退出；同时明确 C-4 的宿主可用性边界 |

以上最晚在生产实现开工前完成。不同选择应有相应规范同步，不能把候选当作已经满足冻结条件。D10、D12、D14～D17、E1～E9 仍按既有契约/工程细化处理，不机械增加产品审批。

### 2．授权实施后、工作包验收前

生产 gate/lint 接入全部启动与回收路径；Runner/CLI、管道/继承目的端、PTY/重定向和真实取消；启动/收尾/短写/提交不确定的逐分支测试；配置固定、空批次、首次 worktree、seq、迁移 0009、Schema/编码、隐私、dbdump 和 §8.6 A/B/C 组合；三平台生产 CI 与工作包规定的集成责任。

本原型没有覆盖记录、事务、取消、配置固定、空批次及持久化组合，不能把“原型三平台成功”扩大为这些内容已验证。也不要求在授权实施前先完成这些生产代码。

### 3．已保留而未接受的新边界

macOS/其他平台无法发现的脱组后代、F1 残留与宿主生命周期、O3 资源及后续调用策略，需要在规格和有关产品选择中准确处理。真实不可中断睡眠、强制 PID 复用和 Windows 异步窗口未动态构造，并不自动要求无限补实验；应以受限机制、官方依据、有效故障注入及保守失败规则界定能够证明的保证。

**ACCEPTED RISK：没有新增。** 方案 B 仅按原 §71/WP2.3 §14.3 的 Git 私有临时区局部边界适用，不豁免 Log 完整性、其他账户保护、合作并发、数据库/来源/隐私或误杀无关进程。M-14、M-15、M-16 不是本轮接受或关闭的风险。

## 十一、最终回答与下一步

- **A．PR #33：技术 NO-GO，暂不建议合并。** 最小阻塞修复限于实验有效性门禁及对应证据；不要求提前生产实现。
- **B．冻结前技术门槛：部分获得有价值支持，尚未整体完成。** 正常观察、预期反例和注入分别计证据；先解决 B1，再按第六、十节收口必要规格与缺项。
- **C．负责人选择：14 项仍未批准；C-4 应在 D20 的支持契约下明确。** 本报告不批准这些选择或接受新风险。

建议由实施方按 B1 做最小修复和定向补证，再交独立审查；负责人随后决定产品取舍、规范冻结和后续授权。本轮没有提交、推送、合并、改基、重跑 CI、关闭维护项或宣布 WP2.5/M2 验收，完成后暂停。
