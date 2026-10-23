# WP2.5 冻结之前的机制原型（test-only）

> 状态：**原型，PR #33，不合并**（[review-resolution.md §89 ~ §91](review-resolution.md)）。依据 [§88](review-resolution.md) 的授权：PR #32 合并核验与 master push CI 全部成功之后，从已核验的 master 另开 PR，把 [WP2.5 规范草案](wp2.5-check-runner.md) §17.2 中 T1 ~ T6 冻结之前的关键可行性问题转成可审查的证据。
>
> - [Review 075](reviews/codex-review-075.md) 对 `932b4a8` 为**技术 NO-GO**：B1，实验的有效性门禁没有兑现它声明的拒绝规则（畸形或缺失的报告、缺失的目标负载仍可得到 confirmed）。第一轮修复见 §2.7。
> - [Review 076](reviews/codex-review-076.md) 对 `3ddd14f` 仍为**技术 NO-GO**：原来的两个反例已修复，但 B1 有两项残余——B1-R1，一次观察失败被之后的成功覆盖，仍得 confirmed；B1-R2，Linux 短链的提前拒绝没有收尾已经启动、脱离原组的链节点。本版已按它修复（§2.8），**待独立复审**；不宣布 B1 关闭或 GO。Review 075 N1 ~ N3 保持审查的结论；Review 076 N4 登记在 §13，本轮没有修复。
> - 这**不是**规范冻结：草案仍是第 4 版，14 项产品选择（草案 §17.3）都没有批准。T1 ~ T6 按 Review 075 的裁定只得到**局部支持**（§11）。
> - 原型成功**不是**生产验收：生产 Check Runner、迁移 0009、CLI 接线都没有授权，WP2.5 没有实现或验收，M2 没有整体验收。
> - 本文中的“候选规格”都待审查；其中涉及可见行为的（例如 C-4、C-7、C-10）依赖对应的产品选择（§9、§10）。本文不新增接受风险，也不批准任何保证调整。

## 0. 摘要

- **Review 075 的 B1**（第一轮，§2.7）：旧的测试基座静默跳过畸形或读不到的报告记录，场景专属的负载不是判定条件，读不到的进程状态被当作“已结束”。第一轮修复（`a793ced`）之后，Review 075 的两个反例都被拒绝。
- **Review 076 的两项残余**（第二轮，§2.8）：
  - **B1-R1**：进程结束的观察失败一次、之后成功时，失败被覆盖，场景仍 confirmed、exit 0；
  - **B1-R2**：Linux 上进程链在排他区间内因负载不完整被拒绝时，已经启动、脱离原组的链节点既没有登记也没有被清扫，只剩辅助进程自己的 60 s 定时器。
- **回归先行**（`1df33df`）：外层门禁检查增加了独立于内层的核验——见证文件记下每个已启动的辅助进程与它自己的启动时间，内层结束之后外层核对它们都已结束、并且都已登记或被申报为未入账；新增 `observe.fail-once` 与 `load.chain-corrupt`。在旧门禁上，三平台诊断 run 39886679862 为 failure：`observe.fail-once` 在三平台都是 confirmed、exit 0；Ubuntu 上短链留下 3 个、损坏的链留下 2 个存活节点；各平台的报告损坏拒绝都有已启动而未入账的进程。
- **修复**（`8fea6c6`）：观察失败不可撤销，与清理的完成分开记录；报告的进程在记录到达时即核验登记，拒绝时未能登记的申报为未入账；在排他区间内被拒绝的检查先结束进程组、清扫区间内被收养的全部进程再离开区间；Linux 上留下本进程的子进程即为 invalid（§2.2、§2.4、§2.8）。隔离变异中，撤回每一项修复，对应的回归都会失败。
- **修复后的证据**（原型代码固定为 `8fea6c607b8e760ddd3428d7d674739e2d6b8197`）：三平台诊断 run 39887528435 为 success；Ubuntu（带 `-race`）38/38、macOS 27/27、Windows 29/29 个场景 confirmed，门禁回归 21/21、16/16、19/19 通过；没有 refuted、invalid 或 unsupported。作者本机 Windows 29/29 与 19/19，WSL 37 个 confirmed、1 个 unsupported（没有 pwsh）与 21/21（§3）。
- **旧的运行**（run 39815658463、39816938304、39835302375 与此前的常规 CI）保留为历史观测（§3.4）。
- **按门槛**：在原型的范围内，T1 / T6 的管道收尾与受控目的端、T2 的 Linux 排他区间候选、T3 的 macOS 退出观察与组身份、T4 的 Windows 启动链部分失败处理、T5 的所测适配器，都有修复后的证据；都只是**局部支持**，冻结之前仍缺的规格与原型见 §11。
- 14 项产品选择仍待负责人决定（§10）；C-4 与 D20 的关系见 §9。

## 1. 范围与隔离

**包含**：

- `prototype/wp25/`：一个独立的 Go 模块，只有 `_test.go` 文件，依赖只有 `golang.org/x/sys` v0.49.0（与 Specmetry 相同的版本）。`gate_test.go` 是有效性门禁本身的回归，`report_test.go` 是报告协议，`witness_*_test.go` 是外层门禁检查的独立核验（§2.8）。
- `.github/workflows/wp25-prototype.yml`：专用的三平台诊断 workflow（§2.5）。
- 本文，以及 review-resolution、AGENTS.md、实施计划与草案头部的状态记录。

**隔离**（可以直接核对）：

- 原型是独立模块，所以 Specmetry 的模块不包含它：根目录的 `go list ./...` 中没有 `prototype/` 下的包，`go vet`、`go test`、golangci-lint 的 `./...` 都不进入它；常规 CI 的日志中没有原型包。
- 只有测试文件，没有可导入的包；根模块的 `go.mod`、`go.sum` 没有改动，也无法在不修改它们的情况下导入原型。
- 生产代码、`schemas/`、迁移、SQLite 的保护、重试与错误映射都没有改动；没有 `check run` 命令；`ci.yml` 的 Job、名称、预算与保护步骤都没有改动。

**不做**：不运行用户配置的任何检查；不终止或回收归属未确立的进程；不使用付费服务；不自动批准任何保证变化。

## 2. 方法与有效性规则

### 2.1 辅助进程与夹具

- 每个场景只启动它自己的辅助进程：测试二进制本身，以环境变量 `WP25_HELPER` 选择模式（写出固定字节、按时退出、持有输出、建立进程链、离开进程组、写标记文件等）。
- **期限分两层**（Review 075 N3）：
  - 每个辅助进程有自己的 60 s 定时器，到时写一行 `limit` 并以 97 退出，这一场景即为无效。它**只在辅助进程运行 Go 代码之后才生效**，挂起创建（Windows `CREATE_SUSPENDED`）的区间里不起作用。
  - 场景自己的每一次等待都有界（父端的终止与确认、有时限的句柄等待、报告等待），不依赖辅助进程的定时器兜底。go test 的 15 分钟与 Job 的 20 分钟只是诊断的总上限：超时终止测试进程时，不保证 defer 与清理已经运行。
- 父辅助进程（持有者、成员、链）报告子进程之后，**等待 stdin 被关闭**才继续：场景在它们仍然活着时核验报告的后代（§2.2），再关闭 stdin 放行。
- 辅助进程自己的失败各有退出码：子进程起不来 90、记录写不出 91、等不到子进程的记录 92、输出写不出 93。场景核对辅助进程的退出状态，不一致即为负载没有建立。
- 测试输出只有每个场景一行的 `WP25RESULT`（结论、预测、观察、启动数、工作量、清理）、每项门禁回归一行的 `WP25GATE`，以及开头的 `WP25META`、结尾的 `WP25SUMMARY`。不保存任何输出内容：进入受控目的端的字节只在内存中用于比较，被丢弃的字节只计数。

### 2.2 进程的归属与清理

- **归属**：场景只操作它启动的进程（Go 的 `exec.Cmd`），以及它**核验为本次检查的**、由自己的辅助进程报告的后代。报告中的 PID 本身不是授权：
  - Unix：该进程的父进程链能追到场景启动的主进程，或者它在主进程的进程组中、并且主进程仍是本进程未回收的子进程；
  - Windows：该进程在本次检查的 Job 中（`IsProcessInJob`）。

  核验不了即为 invalid（“ownership not established”），不扩大清扫范围补救。
- **报告的进程在记录到达时即核验并登记**（`awaitOwned`，Review 076 B1-R2）：不等全部记录到齐。所以拒绝（负载不完整、报告损坏、归属核验不了）发生时，已经看到的每一个进程都已登记、会被清理；没能登记的（没有报告，或报告读不懂）申报为**未入账**，清理因此不称“已核验”，场景为 invalid。报告中的 PID 仍然只有经归属核验之后才可操作。
- **身份**：核验时记下身份（Linux 的 `/proc` 启动时间，macOS 的 `sysctl` 启动时间，Windows 打开并持有句柄）；每次操作之前再次核对，不一致就不再操作。
- **状态的三种读数**：可靠地不存在（Linux 的 ENOENT / ESRCH，macOS 的空回答）、存在、**未知**（其他读取失败、解析失败、回答长度不对）。未知从不当作“已结束”“不存在”或“空组”：清理不算核验，观察为 invalid，排他区间的基线或清扫停止。
- **观察失败不可撤销**（Review 076 B1-R1）：一个进程的第一次观察失败被保存，之后的成功观察不清除它；清理仍然结束并回收这个进程，但这次运行为 invalid。等待主进程退出、组长的状态、场景结束时组或 Job 不能确认为空、清扫中读不到的列表或状态，同样记入统一终判，之后的任何结果都不能恢复。“清理已经完成”与“这次运行有效”分开记录。
- **正常的“尚未”**：进程仍在运行、报告的记录尚未到齐、最后一行仍在写入，都是有期限的等待（期限见各场景，报告等待 5 s 或 20 s，清理 5 s，进程树确认 `T_kill`），不是失败；到期仍未满足才按负载没有建立或清理没有确认处理。读取错误从不当作“尚未”。
- **清理**：每个场景结束时终止仍存活的、身份仍成立的进程，再确认全部结束；本进程的僵尸子进程只按它自己的 PID 回收，并且只有确实回收之后才算结束。提前停止的场景，在组长仍被持有时先结束自己的进程组并确认为空；Windows 的检查在关闭 Job 句柄之前先终止并确认 Job 为空。在排他区间内被拒绝的检查（Linux），在离开区间之前先结束它的进程组，再清扫区间内被收养的全部进程：按区间的规则它们都属于这次检查，不论是否报告过（`abort`）。Linux 上，场景结束时本进程留下新的子进程（存活或未回收的僵尸）即为 invalid。确认不了 → 该场景无效。
- **句柄**（Review 075 N2）：Windows 的观察句柄在清理确认之后显式关闭；Runner 自己的 Job 与进程句柄以 `CloseHandle` 的结果核对，不再对已关闭的句柄调用任何 API（这种调用的行为没有定义）。
- 从不向不属于场景的 PID 或进程组发信号。`wait(-1)` 只出现一次：`T2.other-reaper` 有意扮演“其他回收者”，在排他区间内（本进程此时只有这一个子进程）调用它。

### 2.3 报告协议

辅助进程向场景给它的报告文件追加记录，每个事件一次写入一整行 `<kind> <pid>\n`：

| kind | 含义 |
|---|---|
| `pid` | 辅助进程正在运行（睡眠者、持有者、链的一层） |
| `epipe` | 持有者第一次写入失败（读端已关闭） |
| `fanout` | fanout 辅助进程的全部子进程都已报告 |
| `limit` | 辅助进程到达硬时限 |

- 其他一切都无效：其他 kind、多余的字段、非规范或非正的 PID、超过 64 字节的行、超过 64 KiB 的文件。
- **两个时点**：辅助进程仍可能写入时（场景等待记录），未结束的最后一行是正在进行的写入，留到下一次读；写者结束之后（场景结束时的终判），它就是截断。
- **规格**：每个报告在创建时声明它最终必须恰好包含的记录（例如 silent-holder 每一轮恰好 1 条 `pid`；fanout 恰好 30 条 `pid` 与 1 条 `fanout`；进程链恰好 depth 条 `pid`）。属于机制结果而不是负载的记录（F1 中持有者的 `epipe`）声明为“至多”。同一 (kind, PID) 只能出现一次；`limit` 永不允许。
- 读取失败、任何解析错误、等待超时、多于所需、重复、规格不符，都使场景 invalid；没有一条被跳过。

### 2.4 结论与有效性

每个场景给出预测，并与观察逐项比较：

| 结论 | 含义 |
|---|---|
| `confirmed` | 负载已经建立，观察与候选方案的预测一致 |
| `refuted` | 负载已经建立，观察不同：这是一项发现，不是通过；对应的候选方案按反例处理 |
| `unsupported` | 这里没有这一机制（例如 runner 上没有某个解释器）；不是通过，也不是失败 |
| `invalid` | 这次运行不能作为任何证据：辅助进程没有启动或崩溃、报告缺失或损坏或不符合规格、**场景声明的负载没有建立**（`needLoad`）、归属核验不了、观察读不到（**任何一次**，之后的成功不恢复）、拒绝时有已启动而未入账的进程、Linux 上留下本进程的子进程、到达硬时限、清理没有确认、或在预测检查之前失败 |

- **负载与结果分开**：场景先核对它需要的角色、数量与阶段（辅助进程的成功退出码、报告的每个后代、停止时阻塞的复制任务、Windows fanout 的 30 个后代与 31 个活动进程等），记为 `load:`；负载没有建立是 invalid，完整负载下的不同结果才是 refuted。“启动过一个进程”“有过一次工作”不能代替场景所需的负载；清理成功也不说明负载曾经完整执行。
- 场景分为**正常路径**、**反例**、**故障注入**与**拒绝路径**四类（结果中的 `kind`）。反例的“confirmed”表示反例被复现，即对应的朴素做法确实不成立。
- `refuted` 与 `invalid` 都使测试失败（workflow 显示为失败）；`unsupported` 单列，不计为通过。没有任何场景运行时，整次运行无效。
- **门禁回归**（§2.7）的通过另行计数（`WP25GATE`、`gate_passed`），不是任何机制的 confirmed。

### 2.5 诊断 workflow

`.github/workflows/wp25-prototype.yml`：

- 只在推送到 `proto/wp2.5-prototype` 分支、并且改动了 `prototype/wp25/**` 或该 workflow 时运行；只改文档不会触发；没有手动触发，所以没有盲目重跑；不取消进行中的运行，失败的历史保留。
- 三平台：`ubuntu-latest`（带 `-race`）、`macos-latest`、`windows-latest`。只运行原型模块：`go vet`（Linux 上另查格式），再 `go test -count=1 -v -timeout 15m`；最后列出结果行。仓库公开，标准 runner 不收费。
- macOS 的证据只来自这里的真实 macOS runner。

### 2.6 作者本地草稿运行中发现并修正的问题（第一轮，如实保留）

在第一次推送之前，作者在 Windows 11 与 WSL2 Ubuntu 上运行原型的草稿，发现并修正了以下问题。它们都是原型或测试基座的问题，不是机制的结论：

1. Linux 上对 pidfd 的 `poll` 用截断的毫秒数，等待提前 0.36 ms 返回；改为向上取整（C-1）。
2. 清理把本进程的僵尸子进程在回收成功之前就算作“已结束”；改为确实回收之后才算。
3. 停止读取的延迟测量没有先确认复制任务阻塞在 Read 中；现在先确认。
4. Go 在 Windows 上的单调时钟随时钟中断更新；延迟改用 QueryPerformanceCounter。
5. 场景的记录在并发调用时没有加锁；已加锁。

这五项没有覆盖 Review 075 的 B1：报告与负载门禁的缺陷是在推送之后由独立审查发现的（§2.7）。

### 2.7 Review 075 B1：门禁缺陷、回归先行、修复与变异

**缺陷**（审查第三节）：

- 报告的读取：打开失败返回空；字段不足、PID 解析失败直接跳过；`Scanner` 的错误被丢弃；未知事件不拒绝；最终门禁只看成功解析出来的 `limit`。
- 等待记录到期时返回已有集合，没有“未满足”的结果；fanout 辅助进程不检查子进程的记录是否到齐就报告 `fanout`。
- 场景专属的负载不是判定条件：要求 30 个后代的 Windows 场景只要求一条 `fanout` 记录，五个主进程提供的 `work = 5` 绕过了通用的 `work > 0`。
- 观察与清理：Linux 的 `allPIDs` 丢弃 ReadDir 错误，使组成员读不到时变成空集合；`pidProc.ended` 把一切读取失败当作已结束。

**回归先行**（`5140588e4cbb54eb224dc8131b113856ed43c781`，只有注入钩子与回归，门禁未改）：

- `TestGate` 为每一项在新的进程中运行一个真实场景（`-test.run` 限定，150 s 与父端 180 s 两层时限），注入经 `WP25_INJECT` 传给场景与它的辅助进程；正常运行不设任何注入。
- 判定：损坏的运行，真实诊断必须给出 invalid **并且**以非零码退出；正对照必须 confirmed 并以 0 退出。
- 旧门禁上的结果：

  | 环境 | 失败 / 总数 | 失败项的内层结论 |
  |---|---|---|
  | 本机 Windows 11 | 10 / 16 | 全部 confirmed、exit 0 |
  | 本机 WSL2 | 10 / 17 | 全部 confirmed、exit 0 |
  | run 39830995366 macOS | 9 / 13 | 全部 confirmed、exit 0 |
  | run 39830995366 Windows | 10 / 16 | 全部 confirmed、exit 0 |
  | run 39830995366 Ubuntu | 10 / 17 | 全部 confirmed、exit 0 |

  失败的是：Review 075 的两个反例（`report.malformed`、`load.fanout-zero`），`report.missing`、`report.unknown-kind`、`report.duplicate`、`report.truncated`、`report.overlong`、`report.deleted`、`load.parent-fails`、`load.fanout-partial`，以及 Unix 上的 `cleanup.state-unknown` 与 Linux 上的 `observe.readdir-fails`（审查第三节第 4 点的静态缺口，在这里动态复现）。
- 在旧门禁上就通过的：全部正对照；`cleanup.unconfirmed`（旧的清理循环已能拒绝）；`load.chain-short`，但原因是检查主进程的等待超时（“check main did not exit”），不是负载门禁。
- run 39830995366 中，普通场景仍全部 confirmed（macOS 27、Windows 29、Ubuntu 38）：这正是假通过的形态。同一提交的常规 CI（run 39831007212）为 success，它不运行原型。

**修复**（`a793cedfcef5596bad05612df7f4165490319f28`）：§2.1 ~ §2.4 所述，以及两项新的回归——规格之外的合法记录（`report.unexpected-kind`，只有终判能拒绝）与报告给出检查进程树之外的 PID（`ownership.foreign-pid`，只有归属核验能拒绝；那个 PID 属于场景自己启动的兄弟进程）。

**修复后的门禁回归**（run 39835302375 与作者本机，同一份原型代码）：

| 门禁回归 | 注入 | 应得 | Ubuntu | macOS | Windows | 本机 Win11 | 本机 WSL |
|---|---|---|---|---|---|---|---|
| `control.silent-holder` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.normal-drain` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.counterexample` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.refusal` | — | confirmed | ✓ confirmed/0 |  |  |  | ✓ confirmed/0 |
| `control.adoption-chain` | — | confirmed | ✓ confirmed/0 |  |  |  | ✓ confirmed/0 |
| `report.malformed` | `report-malformed` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.missing` | `report-skip` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.unknown-kind` | `report-unknown` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.duplicate` | `report-dup` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.unexpected-kind` | `report-extra` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.truncated` | `report-truncated` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.overlong` | `report-overlong` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.deleted` | `report-delete` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `load.parent-fails` | `parent-fail` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `load.chain-short` | `chain-short` | invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `ownership.foreign-pid` | — | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `cleanup.unconfirmed` | `cleanup-unconfirmed` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `cleanup.state-unknown` | `procinfo-unknown` | invalid | ✓ invalid/1 | ✓ invalid/1 |  |  | ✓ invalid/1 |
| `observe.readdir-fails` | `readdir-fail` | invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `control.fault-injection` | — | confirmed |  |  | ✓ confirmed/0 | ✓ confirmed/0 |  |
| `control.fanout` | — | confirmed |  |  | ✓ confirmed/0 | ✓ confirmed/0 |  |
| `load.fanout-zero` | `fanout-zero` | invalid |  |  | ✓ invalid/1 | ✓ invalid/1 |  |
| `load.fanout-partial` | `fanout-partial` | invalid |  |  | ✓ invalid/1 | ✓ invalid/1 |  |

`✓ invalid/1` 表示真实诊断把这次损坏的运行判为 invalid 并以 1 退出；`✓ confirmed/0` 是正对照。空白为该平台没有这一项。

**隔离变异**（作者本机，仓库外的原型副本，每次只撤回一道门禁，只运行它针对的回归与健康正对照）：

| 变异 | 撤回的门禁 | Windows | WSL |
|---|---|---|---|
| M1 | 解析器跳过读不懂的行 | `report.malformed`、`report.unknown-kind` 失败 | 同左 |
| M2 | fanout 不再要求 30 个后代与 31 个活动进程 | `load.fanout-zero` 仍通过（终判拒绝：两道门禁互为后备） | — |
| M2b | 同 M2，并撤回终判 | `load.fanout-zero`、`load.fanout-partial` 失败 | — |
| M3 | 撤回报告的终判 | `report.unexpected-kind` 失败 | 同左 |
| M4 | 读不到的状态当作已结束 | — | `cleanup.state-unknown` 失败 |
| M5 | 读不到的组成员当作空组 | — | `observe.readdir-fails` 失败 |
| M6 | 报告的 PID 不经核验即归属 | `ownership.foreign-pid` 失败 | 同左 |
| M7 | 不核对父辅助进程的退出码 | `load.parent-fails` 失败 | 同左 |

健康正对照 `control.silent-holder` 在每一个变异中都通过。M7 在 Windows 上的第一次变异写错，编译失败，那次运行不算数，日志保留；改正后重跑得到上表的结果。变异只在作者本机运行，没有进入 CI；macOS 没有变异。

### 2.8 Review 076：B1 的两项残余

**根因**（审查第三节）：

- **B1-R1**：`pidProc.ended` 在观察失败时只把错误写进进程对象并返回“未结束”；清理继续轮询，只要最后所有进程都报告结束，就称“已核验”，此前的失败没有进入终判。Unix 上之后读到“不存在”或回收成功、Windows 上之后的等待成功，都会这样覆盖它。原有的 `cleanup.state-unknown` 只注入持续的失败，发现不了“一次失败、之后成功”。
- **B1-R2**：`runCheck` 等全部 depth 条记录到齐之后才逐条核验归属；负载不完整时 `awaitReport` 直接停止场景，已存在的链节点没有进入清理列表，清扫没有执行；离开排他区间时 subreaper 已被清除；场景结束时的提前收尾只结束原来的进程组，而链在新会话中。所以链节点在内层结束之后仍然存活，只剩它们自己的 60 s 定时器。同一模式（记录到齐才登记）也出现在其他报告损坏的拒绝路径上：已启动的进程虽然被所在的组或 Job 结束了，却既没有登记，也没有被申报。

**外层的独立核验**（回归先行 `1df33df` 引入，门禁不变）：

- 外层门禁检查给内层运行一个**见证文件**。每个辅助进程启动时写下自己的 PID 与自己读到的启动时间（Linux 的 `/proc`、macOS 的 `sysctl`、Windows 的创建时间）；注入的故障到达注入点时留下一行（内层测试进程中为 `WP25INJECT` 行）。场景从不读取见证文件。
- 内层运行结束之后，外层按见证的身份核对：每个已启动的辅助进程都已结束（10 s 内）；仍存活的，报告出来，再由外层按同一身份结束——它们是本轮自己的夹具，不是未知进程。
- 每个已启动的辅助进程，必须在内层的已登记列表中（结果中的 `owned`），或被内层申报为未入账（`missing`）：已启动而两者都不是，就是“因拒绝而跳过登记的对象”。
- 需要注入命中的回归（`observe.fail-once`、`load.chain-short`）还要求见证或 `WP25INJECT` 中至少有一次命中，否则不算。

**新增回归**：

- `observe.fail-once`：在 `pidProc.ended` 中，每个被观察的进程的**第一次**观察被换成失败（Windows 换掉一次真实的 `WaitForSingleObject` 结果，Unix 把一次读数换成“未知”），之后的观察照常。注入点每命中一次打印一行 `WP25INJECT`。
- `load.chain-corrupt`：排他区间内的四层链，每一层的报告都读不懂（审查的“报告错误”同源路径）：链节点的 PID 无法从报告得知。
- `load.chain-short`（已有）：四层链少一层，现在还要求看到注入、核对已启动的链节点全部结束并全部入账。

**旧门禁上的结果**（`1df33df`）：

| 环境 | 失败 / 总数 | 主要失败 |
|---|---|---|
| 本机 Windows 11 | 9 / 19 | `observe.fail-once` 注入命中 3 次，内层 confirmed、exit 0；7 项报告损坏与部分 fanout 有已启动而未入账的进程 |
| 本机 WSL2 | 10 / 21 | `observe.fail-once` 注入命中 3 次，内层 confirmed、exit 0；7 项报告损坏有已启动而未入账的进程；`load.chain-short` 在内层结束后留下 3 个存活的链节点（PID 986、992、998），`load.chain-corrupt` 留下 4 个 |
| run 39886679862 Ubuntu | 10 / 21 | `observe.fail-once` confirmed、exit 0；短链留下 3 个、损坏的链留下 2 个存活节点；7 项报告损坏未入账 |
| run 39886679862 macOS | 8 / 16 | `observe.fail-once` confirmed、exit 0；7 项报告损坏未入账 |
| run 39886679862 Windows | 9 / 19 | `observe.fail-once` confirmed、exit 0；7 项报告损坏与部分 fanout 未入账 |

- run 39886679862：push，attempt 1，completed/failure（预期），每个 Job 检出 `1df33df`；普通场景仍全部 confirmed（38、27、29）。
- 留下的链节点都由外层按见证的身份结束；外层核对它们之后都已结束。没有操作任何未知进程。
- 在旧门禁上就通过的：全部正对照、`report.unexpected-kind`、`load.parent-fails`、`load.fanout-zero`、`ownership.foreign-pid`、持续的 `cleanup.unconfirmed` 与 `cleanup.state-unknown`、`observe.readdir-fails`。

**修复**（`8fea6c607b8e760ddd3428d7d674739e2d6b8197`）：§2.2 的“记录到达时即登记”“观察失败不可撤销”与区间内的收尾，§2.4 的终判。

**修复后的门禁回归**（run 39887528435 与作者本机，同一份原型代码）：

| 门禁回归 | 注入 | 应得 | Ubuntu | macOS | Windows | 本机 Win11 | 本机 WSL |
|---|---|---|---|---|---|---|---|
| `control.silent-holder` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.normal-drain` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.counterexample` | — | confirmed | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 | ✓ confirmed/0 |
| `control.refusal` | — | confirmed | ✓ confirmed/0 |  |  |  | ✓ confirmed/0 |
| `control.adoption-chain` | — | confirmed | ✓ confirmed/0 |  |  |  | ✓ confirmed/0 |
| `report.malformed` | `report-malformed` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.missing` | `report-skip` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.unknown-kind` | `report-unknown` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.duplicate` | `report-dup` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.unexpected-kind` | `report-extra` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `report.truncated` | `report-truncated` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.overlong` | `report-overlong` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `report.deleted` | `report-delete` | invalid | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 | ✓ invalid/1 未入账1 |
| `load.parent-fails` | `parent-fail` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `load.chain-short` | `chain-short` | invalid | ✓ invalid/1 未入账1 |  |  |  | ✓ invalid/1 未入账1 |
| `load.chain-corrupt` | `report-malformed` | invalid | ✓ invalid/1 未入账4 |  |  |  | ✓ invalid/1 未入账4 |
| `ownership.foreign-pid` | — | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `cleanup.unconfirmed` | `cleanup-unconfirmed` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `observe.fail-once` | `observe-fail-once` | invalid | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 | ✓ invalid/1 |
| `cleanup.state-unknown` | `procinfo-unknown` | invalid | ✓ invalid/1 | ✓ invalid/1 |  |  | ✓ invalid/1 |
| `observe.readdir-fails` | `readdir-fail` | invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `control.fault-injection` | — | confirmed |  |  | ✓ confirmed/0 | ✓ confirmed/0 |  |
| `control.fanout` | — | confirmed |  |  | ✓ confirmed/0 | ✓ confirmed/0 |  |
| `load.fanout-zero` | `fanout-zero` | invalid |  |  | ✓ invalid/1 未入账30 | ✓ invalid/1 未入账30 |  |
| `load.fanout-partial` | `fanout-partial` | invalid |  |  | ✓ invalid/1 未入账20 | ✓ invalid/1 未入账20 |  |

`✓ invalid/1` 表示真实诊断把这次损坏的运行判为 invalid 并以 1 退出；“未入账 k”是内层申报的未入账进程数；“存活 k”是外层发现的、内层结束后仍存活的辅助进程数（修复后没有）。每一项还核对了已启动的辅助进程都已结束、都已登记或被申报。

**拒绝之后的清理**（独立核验）：

- `load.chain-short`：已启动的 3 个链节点全部登记（未登记 0），另申报 1 个未入账（那一层没有启动）；内层结束之后，外层核对没有存活的辅助进程。
- `load.chain-corrupt`：报告读不懂，4 个链节点都无法从报告登记，内层如实申报 4 个未入账；它们在离开排他区间之前被清扫结束，外层核对没有存活的辅助进程。
- 报告损坏的 7 项：已启动的睡眠进程无法登记，内层申报 1 个未入账；它在组或 Job 结束时结束，外层核对已结束。
- 部分 fanout（Windows）：10 个已启动的后代都在记录到达时登记，另申报 20 个未入账（没有启动）。

**隔离变异**（作者本机，仓库外的原型副本，每次只撤回一项修复，只运行它针对的回归与健康正对照）：

| 变异 | 撤回的修复 | Windows | WSL |
|---|---|---|---|
| M8 | 观察失败不保存（之后的成功覆盖它） | `observe.fail-once` 回到 confirmed、exit 0 | 同左 |
| M9 | 链在全部记录到齐之后才登记 | — | `load.chain-short`：3 个已启动的节点未入账 |
| M10 | 排他区间内被拒绝时不收尾 | — | `load.chain-corrupt`：4 个节点存活；`load.chain-short` 仍通过（逐条登记与清理兜住：两层互为后备） |
| M12 | 不申报未入账的进程 | `report.missing` 失败；部分 fanout 仍通过（已启动的都已登记） | `report.missing`、`load.chain-corrupt` 失败 |

健康正对照（`control.silent-holder`，以及 Windows 的 `control.normal-drain` 或 Linux 的 `control.adoption-chain`）在每一个变异中都通过。变异只在作者本机运行，没有进入 CI；macOS 没有变异。

**计数分开**：门禁回归的通过（`WP25GATE`）、内层场景的 invalid（被注入故障的真实诊断，内层退出码 1）与机制的 confirmed（`WP25RESULT`）分别统计；预期的拒绝通过门禁回归，不计为任何机制的证据。

## 3. 运行与环境

### 3.1 运行

| 运行 | 对象 | 触发 | 结果 |
|---|---|---|---|
| [run 39887528435](https://github.com/XKHoshizora/specmetry/actions/runs/39887528435) | `8fea6c607b8e760ddd3428d7d674739e2d6b8197`（第二轮修复） | push，attempt 1 | **completed/success**；场景 Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；门禁回归 21/21、16/16、19/19 |
| [run 39886679862](https://github.com/XKHoshizora/specmetry/actions/runs/39886679862) | `1df33dff77ea252c4cae2535f82dd197ffd58697`（第二轮回归先行） | push，attempt 1 | **completed/failure**（预期）：门禁回归 Ubuntu 10/21、macOS 8/16、Windows 9/19 失败；场景 38、27、29 全部 confirmed |
| 本机 Windows 11 | 与 `8fea6c6` 相同的原型代码 | 作者手动 | 场景 29/29 confirmed；门禁回归 19/19 |
| 本机 WSL2 | 与 `8fea6c6` 相同的原型代码（交叉编译） | 作者手动 | 场景 37 confirmed、1 unsupported（没有 pwsh）；门禁回归 21/21 |
| run 39835302375 | `a793ced`（第一轮修复） | push | 历史观测（§3.4） |
| run 39830995366 | `5140588`（第一轮回归先行） | push | completed/failure（预期，§2.7） |
| run 39815658463、39816938304 | `bcf080f`、`4754142` | push | 历史观测（§3.4） |

- 第二轮修复后的运行：每个 Job 检出 `8fea6c6` 本身；没有重跑；唯一的 SKIP 是只供门禁回归内层运行的 `TestInnerForeignPID`；Ubuntu 的原型步骤带 `-race`，没有数据竞争的报告。各平台的场景 ID 集合与 run 39835302375 相同，并集为 53；38 + 27 + 29 是 94 次平台观察，不是 94 个独立场景。
- 不同运行的统计不混合：本节以下与 §4 ~ §8 的数值都来自 run 39887528435 或与它同一份代码的本机运行。

### 3.2 环境

| 环境 | 系统 | Go | 解释器（T5） |
|---|---|---|---|
| Ubuntu runner | Linux 6.17.0-1022-azure，x86-64；带 `-race` | 1.27.3 | `/bin/sh` → dash；bash 5.2.21；dash；pwsh 7.6.6 |
| macOS runner | macOS 26.6.2（Darwin 25.6.0），arm64 | 1.27.3 | `/bin/sh`（bash 3.2.57 的 sh 模式）；bash 3.2.57；dash；pwsh 7.6.6 |
| Windows runner | Windows 10.0.26100（Server 2025），x86-64 | 1.27.3 | Git for Windows 2.56.0（`usr\bin\sh.exe`、`usr\bin\bash.exe`、`bin\sh.exe`，bash 5.3.15）；Windows PowerShell 5.1.26100.33648；pwsh 7.6.7；`cmd` |
| 本机 Windows | Windows 11 Pro 10.0.26300 | 1.27.1 | Git for Windows 2.56.0；Windows PowerShell 5.1.26100.9711；pwsh 7.6.7 |
| 本机 WSL2 | Ubuntu 24.04，Linux 6.18.33.2-microsoft-standard-WSL2；普通用户 | 1.27.1 | dash；bash 5.2.21（没有 pwsh） |

runner 的镜像在各轮之间变化：Ubuntu 在 run 39816938304 为内核 7.0.0-1017-azure，此后各轮为 6.17.0-1022-azure；Windows runner 上的 Windows PowerShell 与 pwsh 在 run 39887528435 为 5.1.26100.33648 与 7.6.7（此前为 33438 与 7.6.6）。

### 3.3 逐场景的结论（第二轮修复后）

✓ 为 `confirmed`，— 为 `unsupported`，空白为该平台没有这一场景（平台专属）。没有任何 `refuted` 或 `invalid`。CI 列是 run 39887528435。

| 场景 | 门槛 | 类别 | Ubuntu | macOS | Windows | 本机 Win11 | 本机 WSL |
|---|---|---|---|---|---|---|---|
| `T1.O3.gate-lingering` | T1 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.F1-skip-drain` | T1 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.blocking-read-end` | T1 | 反例 | ✓ | ✓ |  |  | ✓ |
| `T1.pipe.escaped-holder` | T1 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.normal-drain` | T1 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.silent-holder` | T1 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.stop-read-latency` | T1 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `emain.bounded-timeout` | T1 / T3 | 正常 | ✓ | ✓ |  |  | ✓ |
| `emain.exit-unreaped` | T1 / T3 | 正常 | ✓ | ✓ |  |  | ✓ |
| `emain.waitid-wnowait` | T1 / T3 | 正常 | ✓ | ✓ |  |  | ✓ |
| `group.escape-boundary` | T1 / T3 | 正常 | ✓ | ✓ |  |  | ✓ |
| `group.zombie-leader` | T1 / T3 | 正常 | ✓ | ✓ |  |  | ✓ |
| `T6.arbitrary-writer` | T6 | 反例 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.dest.concurrent-writers` | T6 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.dest.doc-under-load` | T6 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.dest.owner-short-write` | T6 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.dest.seal-endless` | T6 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.pipe.alias-concurrent` | T6 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.pipe.capacity-discard` | T6 | 正常 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T6.pipe.capacity-stop` | T6 | 反例 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T2.F1.resources` | T2 | 注入 | ✓ |  |  |  | ✓ |
| `T2.adoption.chain` | T2 | 正常 | ✓ |  |  |  | ✓ |
| `T2.baseline.live-child` | T2 | 拒绝 | ✓ |  |  |  | ✓ |
| `T2.baseline.zombie-child` | T2 | 拒绝 | ✓ |  |  |  | ✓ |
| `T2.gate.bypass` | T2 | 反例 | ✓ |  |  |  | ✓ |
| `T2.gate.concurrent-checks` | T2 | 正常 | ✓ |  |  |  | ✓ |
| `T2.gate.exclusive-interval` | T2 | 正常 | ✓ |  |  |  | ✓ |
| `T2.identity.mismatch` | T2 | 注入 | ✓ |  |  |  | ✓ |
| `T2.other-reaper` | T2 | 反例 | ✓ |  |  |  | ✓ |
| `T2.pidfd.unsupported` | T2 | 拒绝 | ✓ |  |  |  | ✓ |
| `T2.subreaper.cleared` | T2 | 正常 | ✓ |  |  |  | ✓ |
| `sigchld.default` | T2 | 正常 | ✓ | ✓ |  |  | ✓ |
| `sigchld.ignored` | T2 | 反例 | ✓ | ✓ |  |  | ✓ |
| `sigchld.inherited` | T2 | 正常 | ✓ | ✓ |  |  | ✓ |
| `T4.chain.normal` | T4 | 正常 |  |  | ✓ | ✓ |  |
| `T4.start.fail-assign` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T4.start.fail-confirm` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T4.start.fail-create` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T4.start.fail-resume` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T4.teardown.query-fails` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T4.teardown.request-is-not-end` | T4 | 正常 |  |  | ✓ | ✓ |  |
| `T4.teardown.terminate-fails` | T4 | 注入 |  |  | ✓ | ✓ |  |
| `T5.adapter.bash-argv` | T5 | 正常 | ✓ | ✓ |  |  | ✓ |
| `T5.adapter.bash-msys-quoting` | T5 | 正常 |  |  | ✓ | ✓ |  |
| `T5.adapter.bin-sh-wrapper-msys-quoting` | T5 | 正常 |  |  | ✓ | ✓ |  |
| `T5.adapter.cmd-raw` | T5 | 正常 |  |  | ✓ | ✓ |  |
| `T5.adapter.dash-argv` | T5 | 正常 | ✓ | ✓ |  |  | ✓ |
| `T5.adapter.powershell-msvcrt-quoting` | T5 | 正常 |  |  | ✓ | ✓ |  |
| `T5.adapter.pwsh-argv` | T5 | 正常 | ✓ | ✓ |  |  | — |
| `T5.adapter.pwsh-msvcrt-quoting` | T5 | 正常 |  |  | ✓ | ✓ |  |
| `T5.adapter.sh-argv` | T5 | 正常 | ✓ | ✓ |  |  | ✓ |
| `T5.adapter.sh-msvcrt-quoting` | T5 | 反例 |  |  | ✓ | ✓ |  |
| `T5.adapter.sh-msys-quoting` | T5 | 正常 |  |  | ✓ | ✓ |  |

每个场景的预测、负载（`load:`）、观察、启动数、工作量与清理结果都在 workflow 日志的 `WP25RESULT` 行中。

### 3.4 历史运行与 B1 的影响

- run 39815658463（`bcf080f`）、run 39816938304（`4754142`）与常规 PR CI run 39818439605（`932b4a8`）的事实保留：都是 push / pull_request、attempt 1、completed/success。但它们的 confirmed 汇总**不再作为证据**，也不是通过了独立审查的证据：当时的门禁不能拒绝损坏的报告与缺失的负载。
- 没有证据表明这两轮中的正常记录实际损坏；失效的是“门禁已证明无效的实验绝不通过”这一总括。各场景的状况：
  - **依赖报告、负载或 Unix 清理核验，当时的结论不能单独成立，以修复后的运行为准**：`T1.pipe.F1-skip-drain`、`T1.pipe.silent-holder`；`group.zombie-leader`、`group.escape-boundary`；经 `runCheck` 的 `T2.gate.exclusive-interval`、`T2.gate.concurrent-checks`、`T2.gate.bypass`、`T2.adoption.chain`、`T2.identity.mismatch`、`T2.pidfd.unsupported`、`T2.F1.resources`，以及 `T2.subreaper.cleared`；Windows 的 `T4.teardown.request-is-not-end`（旧门禁接受零后代）。修复后这些场景在三平台都在负载门禁下 confirmed（§3.3）。
  - **Linux 上的进程树确认**：旧代码在 `/proc` 读不到时把组当作空，原则上影响所有 Linux 上的“树已确认”；没有证据表明当时发生过读取失败。修复后读不到即为 invalid。
  - **不依赖报告的场景**（受控目的端、E-main、waitid、阻塞读端、Windows 失败链、适配器、SIGCHLD 等）：原始观察仍可引用，结论在修复后的运行中不变；当前证据同样以修复后的运行为准。
- 本机与 CI 的旧统计（§3.1 之前的版本中的 37/27/29、38/27/29）作为历史观测，不再引用为通过。
- **Review 076 之后**：run 39835302375（`a793ced`）与其常规 CI 同样保留为历史观测。它的门禁回归在当时的范围内仍然成立（Review 076 第四节），但不能证明“任何一次观察失败都使运行无效”或“拒绝路径已收尾全部已启动的进程”：
  - 经 `pidProc` 观察进程结束的场景（报告了后代的全部场景），当时一次被覆盖的观察失败不会使运行无效；没有证据表明当时发生过这样的失败；
  - 经 `runCheck` 的 T2 场景与报告损坏的拒绝路径，当时的拒绝可能留下或不申报已启动的进程（Linux 的短链会留下节点，见 §2.8）；正常运行的完整链没有这一问题；
  - 修复后这些场景在三平台都 confirmed（§3.3），当前证据以 run 39887528435 为准。

## 4. T1 / T6：管道路径的收尾与受控目的端

**已证明**（在原型的范围内；数值来自 run 39887528435）：

- **正常排空**（`T1.pipe.normal-drain`）：负载为辅助进程写完全部字节并以 0 退出；262 144 + 131 072 字节经两个复制任务完整、按序到达（O1、O1），没有停止读取，文档在最后且完整。
- **D 阶段到期**（`T1.pipe.escaped-holder`）：一个不在已确认进程树中的写端持有者（停止之前写了 79 ~ 197 次）使 D 阶段用满 `T_drain`（1.0 s）；停止读取之后复制任务在 J 阶段立即返回（O2），持有者之后的写入失败（Linux / macOS 为 broken pipe，Windows 为 “The pipe is being closed”）。
- **F1 跳过 D 阶段**（`T1.pipe.F1-skip-drain`、`T1.pipe.silent-holder`）：持有管道的后代经核验属于本次检查，父辅助进程以预期的退出码结束；停止读取使两个阻塞的复制任务都返回（O2）；后代之后的写入失败并报告；封闭之后目的端只多出所有者的文档。
- **停止读取的延迟**（`T1.pipe.stop-read-latency`，每平台 50 次，每次先确认复制任务阻塞在 Read 中，这是负载门禁）：

  | 平台 | 最小 | 中位 | 最大 |
  |---|---|---|---|
  | Ubuntu runner | 11.5 µs | 21.5 µs | 36.0 µs |
  | macOS runner | 4.6 µs | 19.2 µs | 58.0 µs |
  | Windows runner | 36.7 µs | 40.0 µs | 81.5 µs |

  真实进程持有写端时（`T1.pipe.silent-holder`，每平台 3 次，每次两个复制任务都阻塞在 Read 中）为 23 ~ 137 µs。这些是所测样本的最大值，不是调度上界或实时保证；本文不据此改变提议的 `T_join`（草案 §8.3）。
- **封闭入口**（`T6.dest.seal-endless`、`T6.dest.doc-under-load`，后者 50 次）：一个永不结束、持续写入的复制任务在 D、J 两个阶段都没有返回（O3）；封闭之后目的端只多出所有者的文档，复制任务在下一次写入时被拒绝而返回。
- **O3 与可释放的闸门**（`T1.O3.gate-lingering`）：一个阻塞在 Read 中、忽略 Close 的复制任务为 O3；从封闭起它不能写出；释放闸门之后它的一次迟到写入被拒绝，任务结束，goroutine 数恢复。两种后续调用的策略都执行了：“拒绝”使下一次执行被拒绝，“允许”使下一次执行正常完成（O1、文档在最后），也收不到旧输出。原型释放了它设置的全部人为闸门；**没有**模拟一个真正永不返回的 Read 并声称它的资源已经释放。
- **受控目的端**：8 个复制入口与所有者并发写入（`T6.dest.concurrent-writers`），每次写入是一个完整的区段、每个入口的字节保持顺序、文档在最后；两个流别名到同一目的端并与所有者的 200 次进度写入并发（`T6.pipe.alias-concurrent`），每个流原样、按序到达；所有者短写（`T6.dest.owner-short-write`）被报告，不追加第二份文档。别名流可以先有检查的内容，文档整体写入不使整个流变成一份 JSON。
- **容量耗尽**（`T6.pipe.capacity-discard`）：上限 64 KiB、检查写出 1 MiB 时，复制任务继续读并丢弃 983 040 字节，检查正常退出，结果为 O2，目的端中正好是前 64 KiB。

**反例**（复现，对应的做法不成立）：

- `T1.pipe.blocking-read-end`（Linux、macOS）：读端为阻塞模式时，停止读取不能使 Read 返回（O3），描述符在 Close 之后仍然打开；持有者写入之后 Read 返回，它的迟到写入被已封闭的入口拒绝，此后描述符才真正关闭。所以 Read 返回不一定要等最后一个写端结束：一次数据也可以使它返回，之后的写入被封闭的入口拒绝（C-2、C-3）。
- `T6.pipe.capacity-stop`：复制任务在第一次写错误时停止，检查便阻塞在写满的管道上，被迫进入超时（C-5）。
- `T6.arbitrary-writer`：阻塞在 Write 中的复制任务无法停止，放弃它会让迟到的字节落在文档之后（C-7，D20 (i)）。

**O3 之后的归属与后续调用**：见候选规格 C-3、C-4，以及 C-4 与 D20 的关系（§9）。

### 4.1 E-main 与组身份（Linux）

E-main 与组身份的场景同时属于 T1（Linux）与 T3（macOS），这里说明 Linux 的结果，macOS 见 §6。

- **有时限、不回收的观察**（`emain.exit-unreaped`、`emain.bounded-timeout`）：Linux 上对主进程的 pidfd 做有时限的 `poll`，再以 `waitid(P_PID, WEXITED | WNOWAIT)` 取得状态。主进程在 300 ms 退出时，约 307 ms 观察到，状态为 `exit 7`，进程仍是僵尸、组不变、没有留下等待的 goroutine；不退出时在时限（500.6 ms）返回，之后对组发出 SIGKILL，观察到 `signal 9`。
- **组身份**（`group.zombie-leader`）：成员经核验在本组中；组长退出而组中仍有成员时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束，确认之后才回收组长；回收之后 `kill(-pgid, 0)` 为 ESRCH，原型此后拒绝再向这个组发信号。
- **边界 1**（`group.escape-boundary`）：`setsid` 的后代经核验属于本次检查、在另一个组中；组被确认为空时它仍存活（Linux 上由 subreaper 收回，见 §5）。

## 5. T2：Linux 的排他区间与身份

**已证明**（Ubuntu runner 带 `-race`，与本机 WSL2）：

- **闸门强制排他区间**（`T2.gate.exclusive-interval`）：四个 goroutine 不停地经**同一个闸门实例**启动子进程（其中一个是真实的 `git --version`：Ubuntu 上 182 次），三次检查在其间各自取得排他区间：每次基线为空，进程链的每一层都经核验属于检查，被收养的进程都在其中，全部经 pidfd 回收；共享租约与检查区间的重叠为 0；之后 subreaper 为 0，没有子进程。它不证明生产中的全部调用点已经接入闸门。
- **进程内并发的检查**（`T2.gate.concurrent-checks`）：两个调用方同时各请求两次检查，四次都干净地完成，检查区间两两不重叠。
- **重复收养**（`T2.adoption.chain`）：一条经核验的四层进程链（新会话）在组长退出之后逐层被收养，清扫用了 4 轮、经 pidfd 回收 4 个，身份不符 0 次，之后才回收组长。
- **拒绝路径**：基线中有存活的子进程或僵尸（`T2.baseline.live-child`、`T2.baseline.zombie-child`）→ 不设置 subreaper、不启动检查；基线读不到 → 同样拒绝；注入身份不符（`T2.identity.mismatch`）→ 候选被放弃、从未收到信号、清扫不收敛（F1）；注入 `pidfd_open` 的 ENOSYS（`T2.pidfd.unsupported`）→ 清扫停止，不回退到按 PID 的信号；进程状态读不到 → 清扫停止。
- **F1 之后的资源**（`T2.F1.resources`）：注入“清扫放弃”之后，subreaper 已清除、组长保持未回收、被收养的进程仍是本进程的子进程，下一次检查因基线不为空而被拒绝（C-15）。这里组长已经退出；仍在运行的主进程不在这一证据之内。
- **subreaper 只在区间内**（`T2.subreaper.cleared`）：区间之外产生的孤儿（在其父进程仍活着时核验）由 init 或其他 subreaper 收养（Ubuntu runner 上父进程为 1），不回到本进程。
- **SIGCHLD 前提**（`sigchld.*`，Linux 与 macOS）：`signal.Ignore(SIGCHLD)` 的进程，其子进程退出后被内核自动回收（`waitid` 返回 ECHILD；Linux 的 `SigIgn` 显示 SIG_IGN），`signal.Ignored` 为真，S0 的守卫能发现它；从父进程继承的 SIG_IGN 被 Go 运行时覆盖（Linux 内核显示“未忽略”），子进程保持未回收（C-10）。

**反例**：

- `T2.gate.bypass`：区间内不经闸门启动的子进程与被收养的进程一样是本进程的子进程，按父进程清扫会把它当作检查的进程（原型以只列不杀的方式确认，没有终止它）。
- `T2.other-reaper`：进程中的另一段代码以 `wait(-1)` 回收了组长；组信号之前的核对（启动时间一致且仍是本进程的子进程）发现了它并拒绝发信号，但核对与信号之间的回收无法发现。

两者都说明，运行时的闸门与核对之外，还需要静态规则（C-9）。

**仍缺**：见 §11。

## 6. T3：真实 macOS

全部来自真实的 macOS runner（macOS 26.6.2，Darwin 25.6.0，arm64），第二轮修复后的运行与此前各轮的观察相同；没有用其他平台的实验代替。这些证据绑定所测的版本与 arm64 ABI，不自动覆盖其他版本。

| 问题 | 结果 | 场景 |
|---|---|---|
| 有时限的退出观察 | kqueue `EVFILT_PROC`（`NOTE_EXIT \| NOTE_EXITSTATUS`）：主进程 300 ms 退出，约 304 ms 观察到 `exit 7`；不退出时在 501 ms 的时限返回；没有留下等待的 goroutine | `emain.exit-unreaped`、`emain.bounded-timeout` |
| `WNOWAIT` 与回收的顺序 | 观察之后主进程仍是僵尸（`sysctl` 的 `SZOMB`），组不变；`waitid(P_PID, WEXITED \| WNOWAIT)` 经原始系统调用（固定的 siginfo 偏移）可用，两次都给出 `exit 5` 而不回收；之后 `Wait` 回收，退出码一致 | `emain.waitid-wnowait`、`emain.exit-unreaped` |
| 组身份与成员结束的确认 | 成员经核验在本组中；组长退出、成员存活时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束；`sysctl kern.proc.pgrp` 中不含僵尸的成员为空即确认结束，组长仍未回收；回收之后 `kill(-pgid, 0)` 为 ESRCH | `group.zombie-leader` |
| 只剩僵尸组长时的组信号 | `kill(-pgid, 0)` 返回 **EPERM**（Linux 上成功）。这只是所测系统的观察：它不能解释为组已消失，也不能推广为“任何平台上的 EPERM 都无害”（C-12） | `emain.exit-unreaped` |
| 脱离进程组的后代 | `setsid` 的后代（经核验属于本次检查）在组被确认为空之后仍存活，没有 subreaper 可以发现它（草案 §8.2 的边界 1 在 macOS 上保留；D7 没有因此获批） | `group.escape-boundary` |
| 输出的关闭与复制的收尾 | 与 Linux、Windows 相同：停止读取在 5 ~ 58 µs 内返回阻塞的 Read；D / J 阶段、封闭、O1 ~ O3、容量、别名、最后文档的场景都成立；阻塞模式读端的反例也复现 | §4 的各场景 |
| SIGCHLD | 与 Linux 相同（`sigchld.*`） | §5 |
| 进程状态的读取 | `sysctl kern.proc.pid` 的空回答为不存在，调用失败或长度不对为未知；门禁回归 `cleanup.state-unknown` 在 macOS 上通过 | §2.2 |

**macOS 上不支持或没有测试的**：subreaper（macOS 没有对应机制，所以排他区间与收养清扫只在 Linux）；Specmetry 被 SIGKILL 之后组内进程的去向；真实终端中的信号；其他 macOS 版本与 Intel 机器；sysctl 枚举的重试预算与错误分支的生产规格。

## 7. T4：Windows 的失败链

全部在 Windows runner（Server 2025，26100）与本机 Windows 11（26300）上运行。失败由接缝注入：被注入的那一步不调用真实的 API。只覆盖下表的分支（Review 075 第五节）。

| 场景 | 失败的步骤 | 处理 | 结束 | 用户代码 | 句柄 |
|---|---|---|---|---|---|
| `T4.start.fail-create` | 创建 | 没有进程 | — | 没有运行 | 只有 Job 句柄，关闭成功；错误只记类别（not exist），不记文本 |
| `T4.start.fail-assign` | 加入 Job | 不恢复；`TerminateProcess`，对进程句柄有时限的等待 | 确认（进程不在 Job 中） | 没有标记文件 | Runner 的 Job 与进程句柄关闭成功；Go 的句柄由 `Wait` 释放；主线程句柄在创建时已由 Go 关闭 |
| `T4.start.fail-confirm` | 确认（`IsProcessInJob`） | 同上（实际已在 Job 中，按“未知”处理） | 确认 | 没有标记文件 | 同上 |
| `T4.start.fail-resume` | 恢复线程 | 不恢复；`TerminateJobObject`，确认 `ActiveProcesses = 0` | 确认 | 没有标记文件 | 同上 |
| `T4.chain.normal` | — | 负载：Job 中恰好一个进程；恢复之后用户代码运行；`TerminateJobObject` | 47 µs 内 `ActiveProcesses = 0` | 运行（有标记文件） | — |
| `T4.teardown.terminate-fails` | 终止请求 | 不确认结束（F1）；关闭 Job 句柄（`KILL_ON_JOB_CLOSE`，再次请求） | 协议不确认；场景以自己的句柄**观察到**进程随后结束 | 运行 | — |
| `T4.teardown.query-fails` | 结束查询 | 不确认结束（F1） | 协议不确认；场景以句柄观察到进程已经结束 | 运行 | — |
| `T4.teardown.request-is-not-end` | — | 负载：每轮 30 个后代经核验在 Job 中、31 个活动进程；5 轮 | 5 轮都确认；请求之后第一次查询已为 0，请求到确认 0.36 ~ 0.44 ms；清理核验 155 个进程 | 运行 | 观察句柄在清理之后关闭 |

- “终止请求成功不等于进程已结束”在本轮的负载下没有观察到异步的窗口（第一次查询已为 0），只由注入（终止失败、查询失败）证明协议不会把请求当作结束（C-17）。
- “没有标记文件”只说明所测的挂起进程没有运行到写标记的代码，不单独证明所有恢复失败的路径都没有运行用户代码。
- 没有覆盖：CreateJobObject、SetInformationJobObject、OpenProcess、线程枚举的失败；部分线程已恢复之后的失败；终止失败时仍挂起的进程；重负载下的分布；经服务、计划任务、WMI 或 COM 启动的、不在 Job 中的进程；真实控制台事件。

## 8. T5：解释器适配

复用作者 X5 的 23 个字符串，每个都放在引号定界的 here-document（PowerShell 为 here-string）中原样打印回来；先用 `plain` 做一次健全检查，失败即为无效（工具问题，不是引号问题）。

| 平台 | 适配器 | 结果 |
|---|---|---|
| Ubuntu、macOS、WSL | `sh`、`bash`、`dash`、`pwsh`（argv） | 23/23（WSL 没有 pwsh） |
| Windows runner、本机 | Git for Windows `usr\bin\sh.exe`、`usr\bin\bash.exe`、`bin\sh.exe`，MSYS 引号 | 23/23 |
| | 同一 `sh`，MSVCRT（Go 默认）引号（反例） | 21/23：`two\\` 与 `\\server\share\` 中的成对反斜杠各丢一个 |
| | Windows PowerShell 5.1、pwsh 7.6，MSVCRT 引号 | 23/23 |
| | `cmd /d /s /c "…"`（原样） | `exit /b 7`、`exit /b 9009` 原样返回；LF、CRLF 的多行命令都只执行第一行（D3） |

- 这些结果支持所测版本上这些脚本文本的往返，不穷尽所有 shell 文本、分隔符冲突、启动文件或参数。
- 安装布局：两个 Windows 环境都是 `C:\Program Files\Git`，由 `git --exec-path` 向上找到；没有测试其他版本、便携版、Cygwin 或独立 MSYS2。
- Review 075 的审查方在其环境中运行 Git for Windows `sh` 的健全检查时，进程以 3221225794 退出，两项被正确判为 invalid；本文不推断其原因。

## 9. 候选规格（待审查）

下列每一条都是对草案 §8 ~ §9 的细化候选，依据是本文的场景；冻结时由审查确认或否定。**它们不批准任何产品选择**，也不改变观测级、方案 B 或已冻结的契约。其中涉及可见行为的（例如拒绝之后的退出码 6）依赖对应的产品选择，在“前提与限制”一栏写明。“前提与限制”一栏按 Review 075 第六节收口。

| ID | 候选规格 | 依据（场景） | 前提与限制（Review 075） |
|---|---|---|---|
| C-1 | **有时限的等待不早于时限结束**：以毫秒为单位的系统调用（`poll`）对剩余时间向上取整；`kevent` 用纳秒的 `timespec` | `emain.bounded-timeout` | 超时的等待不提前，与事件先到时提前返回是两回事；不是实时保证 |
| C-2 | **管道路径的读端只用 Go 自建的、能被 Close 取消读取的管道**（`os.Pipe`）：关闭读端会使阻塞中的 Read 返回。不得包装外部给出的描述符，也不得把读端设为阻塞模式 | `T1.pipe.stop-read-latency`、`T1.pipe.silent-holder`；反例 `T1.pipe.blocking-read-end` | 按平台描述取消能力（Windows 不能套用 Unix 的 O_NONBLOCK）；不推广到任意 Reader；保留 O3 作为兜底 |
| C-3 | **O3 复制任务的归属**：任务（goroutine）持有它的读端对象（Specmetry 已调用 Close，真正的关闭推迟到 Read 返回）与已封闭的复制入口（仍引用目的端，但不能写出）；Runner 在进程级登记全部未确认结束的复制任务，直到它们返回。“不再写出”从封闭起成立；“资源已结束”要等 Read 返回 | `T1.O3.gate-lingering`、`T1.pipe.blocking-read-end` | Read 返回不一定要等最后一个写端结束：一次数据也可以使它返回，之后的写入被封闭的入口拒绝；登记不是释放，封闭不是任务回收 |
| C-4 | **O3 之后同一进程中的后续调用**：两种策略都已执行。候选推荐：登记中仍有未确认结束的复制任务时，新的检查执行在启动之前拒绝 | `T1.O3.gate-lingering` | **属于 D20 的待决范围**（Review 075 N1）：拒绝并返回 6 依赖调用方的支持契约；D20(a) 只决定当前批次停止，不决定宿主中新的调用；D20(b) 若允许继续，须明确与 C-4 的优先关系；区分单次 CLI 进程与长驻进程内的调用。拒绝的代价可能是长驻宿主之后都不能执行检查，不能由“样本中没有出现 O3”推为几乎没有代价。**未批准** |
| C-5 | **容量耗尽时继续读并丢弃**：复制入口返回容量错误之后，复制任务继续读取并丢弃其余数据（只计数，不保留内容），结果为 O2；检查不因管道写满而阻塞 | `T6.pipe.capacity-discard`；反例 `T6.pipe.capacity-stop` | 只针对受控目的端的管道路径；与真实文件的继承路径分开 |
| C-6 | **受控目的端的规则**：一个锁串行化全部写入；写入不阻塞；所有者入口与复制入口分开；封闭是锁下的一次设置；所有者写入成功时最后的文档完整、在最后；所有者写入失败或短写按 §8.6 C2 报告，不追加第二份文档 | `T6.dest.*`、`T6.pipe.alias-concurrent` | “不阻塞”限于没有外部 I/O、没有回调的有限追加，不承诺获得锁的实时上界；所有者写入出错时仍可能已写出一部分 |
| C-7 | **任意 Writer 在启动之前拒绝**：阻塞在 Write 中的复制任务无法停止，放弃它会让迟到的字节落在文档之后 | 反例 `T6.arbitrary-writer` | 是 D20 (i) 待批准的支持契约；原型没有实现生产的预检 |
| C-8 | **进程级闸门（排他区间的强制方式）**：Specmetry 启动的每一个子进程（`gitx` 的 git、`doctor` 的 `git --version`、macOS ACL 检查的 `/bin/ls`、检查本身）都经同一个闸门：普通启动持共享租约，直到子进程被回收；检查持排他租约，等待全部共享租约结束（有等待中的排他请求时，新的共享请求等待），确认本进程没有任何子进程（僵尸也算）、并且这一列表读得到之后才设置 subreaper，收尾完成之后清除 | `T2.gate.exclusive-interval`、`T2.gate.concurrent-checks`、`T2.baseline.*` | 只证明同一个闸门实例上的调度；生产中全局唯一的实例与所有者、所有启动路径、等待与取消、开始 / 结束上下文的 git 与区间的先后、F1 之后的共享租约策略、避免同一闸门内再申请共享租约形成自锁，都须定清 |
| C-9 | **闸门之外的静态规则**：只有受控的入口可以启动或回收子进程、改变自动回收的前提 | 反例 `T2.gate.bypass`、`T2.other-reaper`、`sigchld.ignored` | 规则须覆盖实际的启动与回收能力，而不只是 `os/exec` 的导入：`os.StartProcess`、`syscall` / `x/sys` 的 fork / exec 与 wait 包装、直接的 `os.Process.Wait`、`waitid` / `Wait4` 的各种形式、改变 SIGCHLD 处置的信号设置，以及内部与依赖中的调用；可用受控入口的白名单与按构建平台的扫描设计。原型没有实现，生产的 lint 接线在授权之后兑现 |
| C-10 | **SIGCHLD 前提**：S0 检查 `signal.Ignored(syscall.SIGCHLD)`，为真则不启动（退出码 6，依赖产品的错误约定）。从父进程继承的 SIG_IGN 不构成问题：Go 运行时启动时为 SIGCHLD 安装自己的处理函数 | `sigchld.*`（Linux 与 macOS） | 运行中不能由别处改变处置或自动回收的前提；`signal.Ignored` 不是宿主全部信号配置的证明 |
| C-11 | **每次组信号之前核对组长**：组长的 PID 仍是本进程未回收的子进程，并且启动时间与启动时记下的一致；否则不再向这个组发任何信号（F1） | `T2.other-reaper`、`emain.exit-unreaped` | 不是原子的“核对并发信号”；仍依赖没有其他回收者（C-9）；没有做强制复用 PID 的验证 |
| C-12 | **确认结束只看成员枚举，不看信号调用的返回**：Linux 用 `/proc` 的进程组成员（不含僵尸），macOS 用 `sysctl kern.proc.pgrp`（不含僵尸），Windows 用 `ActiveProcesses`；枚举读不到不是空组 | `emain.exit-unreaped`、`group.zombie-leader`；门禁回归 `observe.readdir-fails` | 结束的确认还依赖 E-main、保留的组身份与 Linux 的清扫；macOS 的 EPERM 只是所测系统的观察 |
| C-13 | **macOS 的 E-main**：对主进程注册 kqueue `EVFILT_PROC`（`NOTE_EXIT \| NOTE_EXITSTATUS`），以有时限的 `kevent` 等待，不回收、不留下等待任务；注册时进程已退出（ESRCH）则以 `sysctl` 确认它是僵尸。`waitid(P_PID, WEXITED \| WNOWAIT)` 经原始系统调用在所测 macOS 上可用 | `emain.*`（macOS） | 原始系统调用的 ABI 与 siginfo 偏移、注册与查询的失败、枚举的重试预算、支持的版本都须冻结；未测的版本不自动通过 |
| C-14 | **Linux 清扫的身份与有界等待**：每个候选经 pidfd 绑定后再次核对父进程与启动时间；信号经 `pidfd_send_signal`；回收先对 pidfd 做有时限的 `poll`，再 `waitid(P_PIDFD, WEXITED \| WNOHANG)`。身份不符 → 放弃该候选、不发信号；`pidfd_open` 不可用或进程状态读不到 → 停止，不回退到按 PID 的信号；都进入 F1 | `T2.adoption.chain`、`T2.identity.mismatch`、`T2.pidfd.unsupported` | 归属的前提（排他区间）独立于身份；所有系统调用的错误都须保守处理；没有 pidfd 的真实旧内核没有测试 |
| C-15 | **F1 之后的宿主资源**：关闭本次打开的 pidfd；清除 subreaper；组长保持未回收（僵尸），使组 ID 保持保留、基线保持不为空；释放闸门。被收养的进程仍是本进程的子进程，所以同一进程中下一次检查因基线不为空而被拒绝 | `T2.F1.resources` | 原型 F1 的组长已经退出，不能概括仍在运行的主进程；不是任意 F1 或任意嵌入宿主都安全；不等于自动回收 |
| C-16 | **Windows 启动链的失败处理**（只限已测试的分支）：创建失败 → 没有进程，只关闭 Job 句柄；加入 Job 或确认失败 → 绝不恢复，`TerminateProcess` 后对进程句柄做有时限的等待；恢复失败（已在 Job 中）→ `TerminateJobObject` 后确认 `ActiveProcesses = 0`。之后关闭 Runner 的 Job 与进程句柄并核对关闭的结果，`Wait` 释放 Go 的进程句柄 | `T4.start.*`、`T4.chain.normal` | 未测的原生失败点见 §7；父端独立于用户代码的期限、提前异常与所有权移交须在 T4 的规格中写明；没有标记文件不单独证明所有恢复失败的路径都没有运行用户代码 |
| C-17 | **Windows 的终止请求不等于结束**：只有 `ActiveProcesses = 0` 才确认结束；`TerminateJobObject` 失败或查询失败 → F1，即使进程事实上已经结束。关闭 Job 句柄（`KILL_ON_JOB_CLOSE`）是再次请求，不是确认 | `T4.teardown.*` | 没有观察到真实的异步窗口；不据此要求无目标的重负载压力 |
| C-18 | **命令传递的适配器**（草案 §5.5，按 runner 上的版本确认）：Unix 的 `sh` / `bash` / `dash` / `pwsh` 用 argv；Windows 的 Git for Windows `sh` / `bash`（含 `bin\sh.exe` 包装程序）用 MSYS 引号规则，`powershell` / `pwsh` 用 MSVCRT 规则，`cmd` 原样放在 `/d /s /c "…"` 中 | `T5.adapter.*` | 只是所测版本上这些脚本文本的往返；查找、PATH、worktree 排除、安装布局仍需规格；D1 ~ D3 仍待决定 |
| C-19 | **观察失败不可撤销**：一次检查中任何一次状态读取、等待或枚举失败，都使这次检查的结论保守（原型中为 invalid，生产中按 F1 或环境错误）；之后的成功读数不能恢复它。结束与回收仍照常进行，但“已确认结束”与“观察曾失败”分开报告 | 门禁回归 `observe.fail-once`；变异 M8 | 生产中的分类（F1 / 退出码 6）依赖 D19、D20 与错误约定；只证明了所测的观察点 |
| C-20 | **拒绝路径的收尾**：在排他区间内因负载、报告或归属被拒绝的检查，在离开区间之前先结束它的进程组，再清扫区间内被收养的全部进程（按区间的规则它们属于这次检查，不论是否报告过），然后才清除 subreaper、释放闸门；已知的进程在拒绝之前就登记 | 门禁回归 `load.chain-short`、`load.chain-corrupt`；变异 M9、M10 | 只在 Linux 的排他区间内成立；区间外（macOS、Linux 的 `group.escape-boundary`）报告读不懂的脱组后代无法找到，只能如实申报为未入账；生产 Runner 的拒绝路径尚未实现 |

## 10. 产品决定单（全部未批准）

按行为分组；“推荐”沿用草案 §17.3，“本轮证据”只写原型新增的部分。**没有一项因本文而获批**。

| 组 | 决定 | 推荐 | 代价 | 本轮证据 |
|---|---|---|---|---|
| **解释器与命令** | D1 默认解释器与不可用时的行为 | (a) 所有平台 `sh`，不回退；不可用时整批不执行，退出码 6 | 没有 Git for Windows `sh` 的机器上，省略 `shell` 的检查不可用 | Windows runner 与本机的 Git for Windows 2.56.0 都提供 `usr\bin\sh.exe` 与 `bin\sh.exe`，MSYS 引号 23/23（C-18） |
| | D2 Windows `sh` / `bash` 的入口、PATH 与来源 | W2；只取 `gitx` 所用 git 的 Git for Windows | 偏离 architecture §6.4 的环境规则，可能改变工具的解析顺序 | 由 `git --exec-path` 找到安装根，两个入口都可用；PATH 的构造本轮没有测试 |
| | D3 `cmd` 的多行命令 | (a) 拒绝 | 多行批处理要改写，旧配置要人工修订 | Windows Server 2025（26100）与 Windows 11（26300）上 LF、CRLF 都只执行第一行（`T5.adapter.cmd-raw`） |
| | D4 `sh` / `bash` 命令中的 CR | (a) 拒绝 | CRLF 写成的命令要人工修订；不能静默改变保存的值与哈希 | 本轮没有新证据（X4） |
| | D5 EV-04 与 `not_found` | 修订后的 A（按最终状态） | 被命令串或管道掩盖的失败不会被额外识别 | 本轮没有新证据（X3、X10） |
| **进程树与平台支持** | D7 进程树保证的范围；Linux 排他区间不成立时的处理 | 先完成 T1 ~ T4；在 (a) 如实限定与 (b) 不支持的平台不提供之间选择；Linux 排他区间不成立时不启动 | (a) 缩小保证；(b) 缩小平台 | macOS：组终止、不回收的观察与组身份规则有真实证据（C-12、C-13），`setsid` 的后代仍无法发现，没有 subreaper；Linux：排他区间可由闸门强制（C-8），但需要静态规则（C-9）；拒绝路径都已执行 |
| | D11 主进程正常退出之后仍在运行的后代 | (a) 终止 | 有意留下的后台进程会被终止 | 所有管道场景都在正常退出之后终止并确认进程树，再进入 S6 |
| | D19 `T_kill` 内没有确认结束 | (a) 尝试记录为不合格 | 扩展 `context_changed` 的使用情形；时间字段不是实际结束时刻；不是“已证明上下文变化” | F1 的输出与宿主资源（`T1.pipe.F1-skip-drain`、`T2.F1.resources`、C-15）；持久化的叠加本轮没有测试 |
| **记录与退出语义** | D6 结束上下文读取失败 | (a) 记录为不合格 | 环境故障产生一条不合格的执行事实 | 本轮没有新证据 |
| | D8 取消 | (a) 不记录，退出码 3；退出码优先级 1 > 6 > 3 > 10 > 0 | 被取消的一次在事实中看不到；不能从没有记录推断没有执行 | 本轮没有新证据（真实控制台没有测试） |
| | D9 批次汇总 `is_target_snapshot` | (b) 每一次的 `head_start` 都等于 `S` | 一个布尔值不能完整表达混合批次；不改变逐条的 `runs(S)` / `V` | 不属于机制原型 |
| | D13 配置的固定时刻 | (a) 命令开始时一次读取 | 前一次执行期间修订的 AC 仍按旧配置执行 | 不属于机制原型 |
| | D18 `--all` 选不到 AC 且 worktree 未登记 | (a) 只做登记的写事务 | 一次没有执行的命令写入一条事实与一个 `seq` | 不属于机制原型 |
| **输出** | D20 输出的支持契约与转发不完整 | (i) 继承句柄与受控目的端，其他 Writer 拒绝；(a) 转发不完整时照常记录、停止、退出码 6 | 一次转发故障中止整个批次；进程内调用的输出对象受限 | 受控目的端的规则、O1 ~ O3、迟到输出、别名、容量与最后文档在三平台成立（C-2 ~ C-7）。**同时须明确 C-4**：O3 之后同一宿主中新的调用是拒绝还是允许，区分单次 CLI 与长驻宿主（§9） |

## 11. 冻结之前仍缺的规格与原型

按 Review 075 第十节第 1 点。T1 ~ T6 只得到局部支持，不因门禁修好而宣布完成：

1. **B1 的独立复审**：两轮的修复、回归与修复后的证据（§2.7、§2.8、§3）待独立复审；复审之前不宣布 B1 关闭。
2. **候选的前提与措辞**（§9 的“前提与限制”）：特别是 O3 之后的后续调用（C-4 与 D20）、父端独立于用户代码的期限、Windows 未测的原生失败点、macOS 的原始 ABI 与枚举的错误和预算、Linux F1 之后的宿主状态。
3. **C-9 的规则设计**：覆盖实际的启动、回收与信号前提能力（不只是 `os/exec` 的导入）；完整的生产 lint 在授权之后兑现，不能先声称它已存在。
4. **闸门的生产拓扑**（C-8）：全局实例的所有者、上下文采集的 git 与排他区间的先后、取消与等待、F1 之后的资源策略、避免同一闸门内的自锁。
5. **T5 的冻结规格**：解释器的来源、实际路径、相对 PATH 项与 worktree 排除；字符串集的成功不能代替这些规格。
6. **负责人选择 14 项**（§10），再按草案 §18 同步相关的规范、验收与契约。D10、D12、D14 ~ D17、E1 ~ E9 仍按既有契约 / 工程细化处理。

## 12. 授权之后、本层验收之前（生产接入与测试）

原型证明的是机制在隔离夹具上可行，不是生产实现正确。获批之后、本层验收之前仍要：

- 把采纳的候选接入生产的 Runner / CLI，并逐分支测试：正常、超时、取消、启动失败、后代残留、确认失败、复制错误、短写、提交结果不确定，以及草案 §8.6 的每一行（含持久化的 B、C 表）。
- 闸门（C-8）与静态规则（C-9）的实现：`gitx`、`doctor`、macOS ACL 检查改为经闸门启动；lint 规则与导入者测试；进程内并发的 `Execute` 测试。
- 继承路径的进程级测试：真实终端与重定向、PTY、真实取消、`--json` 时 stdout 只有一份文档；继承句柄的限制（Windows 句柄列表、Unix close-on-exec）。
- 配置的固定、空批次、首次 worktree 登记与 `seq`、迁移 0009、`check-run.v1` 的 Schema 与编码、隐私、`dbdump` 的映射。
- 三平台 CI 上带故障注入的回归：本文的场景与门禁回归是起点，但生产测试要经生产代码路径，而不是原型的副本。

本原型没有覆盖记录、事务、取消、配置固定、空批次与持久化的组合，不能把“原型三平台成功”扩大为这些内容已验证。

## 13. 仍未验证的边界

- **真实的 F1**：处于不可中断睡眠、在 `T_kill` 内不结束的进程没有构造；F1 都由注入模拟，只证明分支的结果，不证明真实进程、句柄与属性的生存期。
- **真实控制台**：Ctrl+C、控制台关闭、SIGTERM 的竞争没有测试；`/dev/tty`、`CONIN$` 没有测试。
- **Specmetry 本身被强制结束**：Linux 子树的去向（subreaper 不覆盖）、macOS 的残留没有测试；Windows 的 `KILL_ON_JOB_CLOSE` 只有作者早先的 X2。
- **强制的 PID 复用**：身份不符只由注入模拟；真实复用下的信号误投只有推理。
- **Windows 的异步结束**：测试负载下没有观察到请求之后仍有活动进程的窗口；“请求不等于结束”只由注入证明。
- **其他版本与布局**：只有所列的 macOS 26.6.2（arm64）、Windows Server 2025（26100）与 Windows 11（26300）、Ubuntu runner（内核 6.17 与 7.0）与 WSL2（内核 6.18）、Git for Windows 2.56.0；其他 Git for Windows 版本与安装方式、Cygwin、独立 MSYS2、旧内核（没有 pidfd）没有测试。
- **静态规则（C-9）的实现**：本文只给出候选与反例，没有实现 lint。
- **闸门覆盖的完整性**：原型中的共享租约是辅助进程与 `git --version`，没有改动 `gitx`。
- **门禁本身**：门禁回归与变异覆盖了所列的故障；不是对一切可能的损坏的证明。变异只在作者本机运行（Windows 与 WSL），macOS 没有变异。Linux 上读 `/proc` 失败只由注入模拟；在 `hidepid` 等配置下，读不到的进程会使观察为 invalid（保守），没有测试。一次观察失败只在 `pidProc.ended` 一个观察点上做了“一次失败、之后成功”的注入；其他观察点（等待、组长、组与 Job 的确认、清扫）的失败直接记入终判，只有持续失败或源码层面的覆盖。
- **外层核验的范围**：见证只覆盖测试二进制自己的辅助进程（它们写见证）；内层直接启动的 `git`、解释器等不写见证，由内层自己的清理覆盖。外层核验辅助进程是否结束，不核验它们由谁回收；内层结束之后，它的僵尸子进程由 init 回收。区间外脱组的后代在报告读不懂时无法登记，只能申报为未入账（`group.escape-boundary` 的损坏报告没有回归）。
- **Review 076 N4**（NON-BLOCKER，本轮没有修复）：持有者写 `epipe` 记录失败时，F1 场景把“没有观察到预测的事件”记为 refuted，而不是把“观察工具失败”记为 invalid；没有形成假通过。建议把持有者的报告失败作为独立的工具故障状态传给场景，并增加分类回归。
- **`-race` 只在 Linux**；macOS 与 Windows 的原型没有开启竞态检测。
