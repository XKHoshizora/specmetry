# WP2.5 冻结之前的机制原型（test-only）

> 状态：**原型，PR #33，不合并，待独立审查**（2026-10-23；[review-resolution.md §89](review-resolution.md)）。依据 [review-resolution.md §88](review-resolution.md) 的授权：PR #32 合并核验与 master push CI 全部成功之后，从已核验的 master 另开 PR，把 [WP2.5 规范草案](wp2.5-check-runner.md) §17.2 中 T1 ~ T6 冻结之前的关键可行性问题转成可审查的证据。
>
> - 这**不是**规范冻结：草案仍是第 4 版，14 项产品选择（草案 §17.3）都没有批准。
> - 原型成功**不是**生产验收：生产 Check Runner、迁移 0009、CLI 接线都没有授权，WP2.5 没有实现或验收，M2 没有整体验收。
> - 本文中的“候选规格”都待审查，不改变产品行为；本文不新增接受风险，也不批准任何保证调整。

## 0. 摘要

- **对象**：`prototype/wp25/` 中的 53 个场景（正常、反例、故障注入、拒绝四类），在三平台的 GitHub runner 与作者的两台本地环境上运行（§3）。证据所依据的原型代码固定为 `475414228b995c8dbabeb35d78ce6c1dd9bd9a95`。
- **结果**：诊断 workflow 的第二轮（run 39816938304，对 `4754142`）在 Ubuntu 38/38、macOS 27/27、Windows 29/29 都是 `confirmed`；没有 `refuted`、`invalid`，也没有跳过。第一轮（run 39815658463，对 `bcf080f`）在三平台也全部 `confirmed`（37、27、29）。本地 WSL 上没有 pwsh，记为 `unsupported`。作者的本地草稿运行发现了 5 个原型或测试基座的问题，修正之后才推送（§2.5）。
- **T1 / T6**：管道路径的时间线（D 阶段、停止读取、独立的 J 阶段、封闭入口）在三平台成立；`os.Pipe` 的读端上，停止读取使阻塞的 Read 在 0.4 ms 以内返回，没有出现 O3。O3 只在读端为阻塞模式时出现（反例），这时“不再写出”从封闭起成立，“资源已结束”要等 Read 返回。受控目的端的同步、别名、并发、容量与最后文档的规则成立；容量耗尽时应继续读并丢弃（反例表明否则检查会被迫超时）。
- **T2**（Linux）：排他区间可以由进程级闸门**强制**，不只靠测试纪律：共享租约（含真实的 `git --version`）与检查的区间从不重叠，两个并发的检查被串行化，基线为空（含僵尸）才设置 subreaper；收养清扫逐层进行、经 pidfd 回收；身份不符、pidfd 不可用、基线不为空都走拒绝路径；F1 之后下一次检查被拒绝。旁路启动的子进程与进程内的其他回收者是**反例**：运行时只能部分发现，需要静态规则。
- **T3**（真实 macOS 26.6.2，arm64）：kqueue `NOTE_EXIT` 给出有时限、不回收的退出观察；`waitid(WNOWAIT)` 经原始系统调用可用；未回收的组长使组信号仍指向本组，回收之后组消失；`setsid` 的后代无法被组终止发现（边界保留）。新发现：组里只剩未回收的组长时，`kill(-pgid, 0)` 返回 EPERM（Linux 上成功），所以确认结束只能看成员枚举。
- **T4**（Windows）：启动链每一步的失败都不恢复进程、确认结束、句柄都已关闭、用户代码没有运行；终止请求失败或查询失败时不确认结束。测试负载下请求之后第一次查询 `ActiveProcesses` 已为 0，“请求不等于结束”只由注入证明。
- **T5**：所测解释器上命令文本 23/23 原样到达；MSVCRT 引号交给 MSYS 时 21/23 的反例复现；`cmd` 的多行只执行第一行。
- **仍缺**（§12）：真实的 F1（不可中断睡眠）、真实控制台的取消、Specmetry 被强制结束、强制的 PID 复用、Windows 的异步结束窗口、静态规则的实现、生产接线与全部生产测试（§11）。14 项产品选择仍待负责人决定（§10）。

## 1. 范围与隔离

**包含**：

- `prototype/wp25/`：一个独立的 Go 模块，只有 `_test.go` 文件，依赖只有 `golang.org/x/sys` v0.49.0（与 Specmetry 相同的版本）。
- `.github/workflows/wp25-prototype.yml`：专用的三平台诊断 workflow（§2.4）。
- 本文，以及 review-resolution、AGENTS.md、实施计划与草案头部的状态记录。

**隔离**（可以直接核对）：

- 原型是独立模块，所以 Specmetry 的模块不包含它：根目录的 `go list ./...` 中没有 `prototype/` 下的包，`go vet`、`go test`、golangci-lint 的 `./...` 都不进入它。
- 只有测试文件，没有可导入的包；根模块的 `go.mod`、`go.sum` 没有改动，也无法在不修改它们的情况下导入原型。
- 生产代码、`schemas/`、迁移、SQLite 的保护、重试与错误映射都没有改动；没有 `check run` 命令；`ci.yml` 的 Job、名称、预算与保护步骤都没有改动。

**不做**：不运行用户配置的任何检查；不终止或回收归属未知的进程；不使用付费服务；不自动批准任何保证变化。

## 2. 方法与有效性规则

### 2.1 辅助进程与夹具

- 每个场景只启动它自己的辅助进程：测试二进制本身，以环境变量 `WP25_HELPER` 选择模式（写出固定字节、按时退出、持有输出、建立进程链、离开进程组、写标记文件等）。
- 每个辅助进程都有**硬时限**（60 s）：到时它在报告文件中写一行 `limit` 并以 97 退出，这一场景即为无效。场景在数秒内结束自己的辅助进程，所以到达时限意味着清理失败。
- 后代进程的 PID 经报告文件（只有 PID，没有内容）告诉场景；它们的标准输出、错误输出在不受测时接到空设备，避免 X13 那样意外继承管道。
- 测试输出只有每个场景一行的 `WP25RESULT`（结论、预测、观察、启动数、工作量、清理）以及开头的 `WP25META`、结尾的 `WP25SUMMARY`。不保存任何输出内容：进入受控目的端的字节只在内存中用于比较，被丢弃的字节只计数。

### 2.2 进程的归属与清理

- **归属**：场景只操作它启动的进程（Go 的 `exec.Cmd`）与它自己的辅助进程报告的后代。
- **身份**：报告的后代在得知时记下身份（Linux 的 `/proc` 启动时间，macOS 的 `sysctl` 启动时间，Windows 在得知时打开并持有句柄）；每次操作之前再次核对，不一致就不再操作。
- **清理**：每个场景结束时终止仍存活的、身份仍成立的进程，再确认全部结束；本进程的僵尸子进程只按它自己的 PID 回收，并且**只有确实回收之后**才算结束（§2.5 的缺陷 2）。确认不了 → 该场景无效。
- 从不向不属于场景的 PID 或进程组发信号。`wait(-1)` 只出现一次：`T2.other-reaper` 有意扮演“其他回收者”，在排他区间内（本进程此时只有这一个子进程）调用它；此外从不调用。

### 2.3 结论与有效性

每个场景给出预测，并与观察逐项比较：

| 结论 | 含义 |
|---|---|
| `confirmed` | 观察与候选方案的预测一致 |
| `refuted` | 观察不同：这是一项发现，不是通过；对应的候选方案按反例处理 |
| `unsupported` | 这里没有这一机制（例如 runner 上没有某个解释器）；不是通过，也不是失败 |
| `invalid` | 这次运行不能作为任何证据：辅助进程没有启动或崩溃、结果缺失或损坏、没有启动任何进程或没有做任何工作（按场景声明）、到达硬时限、清理没有确认、或在预测检查之前失败 |

- 场景分为**正常路径**、**反例**、**故障注入**与**拒绝路径**四类（结果中的 `kind`）。反例的“confirmed”表示反例被复现，即对应的朴素做法确实不成立。
- `refuted` 与 `invalid` 都使测试失败（workflow 显示为失败）；`unsupported` 单列，不计为通过。
- 没有任何场景运行时，整次运行无效。

### 2.4 诊断 workflow

`.github/workflows/wp25-prototype.yml`：

- 只在推送到 `proto/wp2.5-prototype` 分支、并且改动了 `prototype/wp25/**` 或该 workflow 时运行；只改文档不会触发；没有手动触发，所以没有盲目重跑；不取消进行中的运行，失败的历史保留。
- 三平台：`ubuntu-latest`（带 `-race`）、`macos-latest`、`windows-latest`。只运行原型模块：`go vet`（Linux 上另查格式），再 `go test -count=1 -v -timeout 15m`；最后列出结果行。仓库公开，标准 runner 不收费。
- macOS 的证据只来自这里的真实 macOS runner。

### 2.5 作者本地草稿运行中发现并修正的问题（如实保留）

在推送之前，作者在 Windows 11 与 WSL2 Ubuntu 上运行原型的草稿，发现了以下问题。它们都是原型或测试基座的问题，不是机制的结论；修正之后才推送：

1. **有时限等待提前结束**：Linux 上对 pidfd 的 `poll` 用截断的毫秒数，等待提前 0.36 ms 返回（场景 `emain.bounded-timeout` 为 `refuted`）。改为向上取整，使有时限的等待不早于时限结束。这一点对生产实现同样适用（候选规格 C-1）。
2. **清理的假通过**：清理把一个本进程的僵尸子进程在回收成功之前就算作“已结束”，场景报告“清理已核验”，遗留的僵尸却使之后的排他区间都因基线不为空而被拒绝（4 个 T2 场景为 `invalid`）。改为只有按 PID 确实回收之后才算结束，清理结果中列出每个进程。
3. **停止读取的前提**：停止读取的延迟测量原来没有确认复制任务在停止之前确实阻塞在 Read 中；一个早已返回的任务会被误读为“立即被打断”。现在先确认阻塞，否则这一次不计。
4. **Windows 的计时粒度**：Go 在 Windows 上的单调时钟随时钟中断更新，低于一个周期的延迟读作 0。延迟改用 QueryPerformanceCounter 测量。
5. 修正 2 之后又发现，场景的记录在并发调用（两个调用方同时请求排他区间）时没有加锁；已加锁，并使非测试 goroutine 中的无效标记只结束该 goroutine。

## 3. 运行与环境

### 3.1 运行

| 运行 | 对象 | 触发 | 结果 |
|---|---|---|---|
| [run 39815658463](https://github.com/XKHoshizora/specmetry/actions/runs/39815658463) | `bcf080f4caebdc3784fc5976cbd188a64f6a3895` | push，attempt 1 | completed/success；Ubuntu 37/37、macOS 27/27、Windows 29/29 `confirmed` |
| [run 39816938304](https://github.com/XKHoshizora/specmetry/actions/runs/39816938304) | `475414228b995c8dbabeb35d78ce6c1dd9bd9a95` | push，attempt 1 | completed/success；Ubuntu 38/38、macOS 27/27、Windows 29/29 `confirmed` |
| 本机 Windows 11 | 与 `4754142` 相同的原型代码 | 作者手动 | 29/29 `confirmed` |
| 本机 WSL2 | `4754142`（交叉编译的测试二进制） | 作者手动 | 37 `confirmed`、1 `unsupported`（没有 pwsh） |

- 两轮之间只增加了 `T2.other-reaper`、共享租约中的 `git --version`、Windows 失败链的句柄核对，并改正两处读起来方向相反的观察项措辞；第一轮已有的场景在第二轮的结论不变。
- 每个 Job 都检出被测提交本身（push 事件），没有重跑，没有跳过的 Job。
- 第二轮的 Ubuntu runner 换了镜像（内核 6.17 → 7.0，pwsh 7.6.6 → 7.6.7）；结论相同。

### 3.2 环境

| 环境 | 系统 | Go | 解释器（T5） |
|---|---|---|---|
| Ubuntu runner | Linux 7.0.0-1017-azure（第一轮 6.17.0-1022-azure），x86-64；带 `-race` | 1.27.3 | `/bin/sh` → dash；bash 5.2.21；dash；pwsh 7.6.7 |
| macOS runner | macOS 26.6.2（Darwin 25.6.0），arm64 | 1.27.3 | `/bin/sh`（bash 3.2.57 的 sh 模式）；bash 3.2.57；dash；pwsh 7.6.6 |
| Windows runner | Windows 10.0.26100（Server 2025），x86-64 | 1.27.3 | Git for Windows 2.56.0（`usr\bin\sh.exe`、`usr\bin\bash.exe`、`bin\sh.exe`，bash 5.3.15）；Windows PowerShell 5.1.26100.33438；pwsh 7.6.6；`cmd` |
| 本机 Windows | Windows 11 Pro 10.0.26300 | 1.27.1 | Git for Windows 2.56.0；Windows PowerShell 5.1.26100.9711；pwsh 7.6.7 |
| 本机 WSL2 | Ubuntu 24.04，Linux 6.18.33.2-microsoft-standard-WSL2；普通用户 | 1.27.1 | dash；bash 5.2.21（没有 pwsh） |

### 3.3 逐场景的结论

✓ 为 `confirmed`，— 为 `unsupported`，空白为该平台没有这一场景（平台专属）。没有任何 `refuted` 或 `invalid`。CI 列是第二轮。

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

每个场景的预测、观察、启动数、工作量与清理结果都在 workflow 日志的 `WP25RESULT` 行中。

## 4. T1 / T6：管道路径的收尾与受控目的端

**已证明**（在原型的范围内）：

- **正常排空**（`T1.pipe.normal-drain`）：262 144 + 131 072 字节经两个复制任务完整、按序到达（O1、O1），没有停止读取，文档在最后且完整。
- **D 阶段到期**（`T1.pipe.escaped-holder`）：一个不在已确认进程树中的写端持有者使 D 阶段用满 `T_drain`（1.0 s）；停止读取之后复制任务在 J 阶段立即返回（O2），持有者之后的写入失败（Linux / macOS 为 broken pipe，Windows 为 “The pipe is being closed”）。
- **F1 跳过 D 阶段**（`T1.pipe.F1-skip-drain`、`T1.pipe.silent-holder`）：仍有后代持有管道时，停止读取使两个阻塞的复制任务都返回（O2）；后代之后的写入失败并报告；封闭之后目的端只多出所有者的文档。
- **停止读取的延迟**（`T1.pipe.stop-read-latency`，每平台 50 次，每次先确认复制任务阻塞在 Read 中）：

  | 平台 | 最小 | 中位 | 最大 |
  |---|---|---|---|
  | Ubuntu runner | 12.6 µs | 23.2 µs | 50.4 µs |
  | macOS runner | 20.1 µs | 29.0 µs | 152.0 µs |
  | Windows runner | 62.9 µs | 78.0 µs | 116.7 µs |
  | 本机 Windows 11 | 56.3 µs | 76.8 µs | 335.4 µs |
  | 本机 WSL2 | 35.6 µs | 75.8 µs | 159.1 µs |

  真实进程持有写端时（`T1.pipe.silent-holder`，每平台 3 次）为 45 ~ 174 µs。所以提议的 `T_join` = 2 s 有三到四个数量级的余量；本文不据此改变提议值（草案 §8.3）。
- **封闭入口**（`T6.dest.seal-endless`、`T6.dest.doc-under-load`，后者 50 次）：一个永不结束、持续写入的复制任务在 D、J 两个阶段都没有返回（O3）；封闭之后目的端只多出所有者的文档，复制任务在下一次写入时被拒绝而返回。
- **O3 与可释放的闸门**（`T1.O3.gate-lingering`）：一个阻塞在 Read 中、忽略 Close 的复制任务为 O3；从封闭起它不能写出；释放闸门之后它的一次迟到写入被拒绝，任务结束，goroutine 数恢复。两种后续调用的策略都执行了：“拒绝”使下一次执行被拒绝，“允许”使下一次执行正常完成（O1、文档在最后），也收不到旧输出。
- **受控目的端**：8 个复制入口与所有者并发写入（`T6.dest.concurrent-writers`），每次写入是一个完整的区段、每个入口的字节保持顺序、文档在最后；两个流别名到同一目的端并与所有者的进度写入并发（`T6.pipe.alias-concurrent`），每个流原样、按序到达；所有者短写（`T6.dest.owner-short-write`）被报告，不追加第二份文档。
- **容量耗尽**（`T6.pipe.capacity-discard`）：上限 64 KiB、检查写出 1 MiB 时，复制任务继续读并丢弃 983 040 字节，检查正常退出，结果为 O2，目的端中正好是前 64 KiB。

**反例**（复现，对应的做法不成立）：

- `T1.pipe.blocking-read-end`（Linux、macOS）：读端为阻塞模式时，停止读取不能使 Read 返回（O3），描述符在 Close 之后仍然打开；持有者写入之后 Read 返回，它的迟到写入被已封闭的入口拒绝，此后描述符才真正关闭。这就是“不再写出”与“资源已结束”的区别（C-2、C-3）。
- `T6.pipe.capacity-stop`：复制任务在第一次写错误时停止，检查便阻塞在写满的管道上，被迫进入超时（C-5）。
- `T6.arbitrary-writer`：阻塞在 Write 中的复制任务无法停止，放弃它会让迟到的字节落在文档之后（C-7，D20 (i)）。

**O3 之后的归属与后续调用**：见候选规格 C-3、C-4。

**仍缺**：

- 生产的 `cli.Execute` 接线与继承路径的进程级测试（终端、重定向）；
- 任意不可中断 Writer 的生命周期仍没有方案，按 D20 (i) 拒绝；
- macOS 与 Windows 的其他版本上 `os.Pipe` 读端的停止行为。

### 4.1 E-main 与组身份（Linux）

E-main 与组身份的场景同时属于 T1（Linux）与 T3（macOS），这里说明 Linux 的结果，macOS 见 §6。

- **有时限、不回收的观察**（`emain.exit-unreaped`、`emain.bounded-timeout`）：Linux 上对主进程的 pidfd 做有时限的 `poll`，再以 `waitid(P_PID, WEXITED | WNOWAIT)` 取得状态。主进程在 300 ms 退出时，约 304 ~ 312 ms 观察到，状态为 `exit 7`，进程仍是僵尸、组不变、没有留下等待的 goroutine；不退出时在时限（500.6 ms）返回，之后对组发出 SIGKILL，观察到 `signal 9`。
- **组身份**（`group.zombie-leader`）：组长退出而组中仍有成员时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束，确认之后才回收组长；回收之后 `kill(-pgid, 0)` 为 ESRCH，原型此后拒绝再向这个组发信号。
- **边界 1**（`group.escape-boundary`）：`setsid` 的后代离开了组，组被确认为空时它仍存活（Linux 上由 subreaper 收回，见 §5）。

## 5. T2：Linux 的排他区间与身份

**已证明**（Ubuntu runner 带 `-race`，与本机 WSL2）：

- **闸门强制排他区间**（`T2.gate.exclusive-interval`）：四个 goroutine 不停地经闸门启动子进程（其中一个是真实的 `git --version`：Ubuntu 上 160 次、WSL 上 81 次），三次检查在其间各自取得排他区间：每次基线为空，被收养的进程都属于检查（都在检查的辅助进程报告的集合中），全部经 pidfd 回收；共享租约与检查区间的重叠为 0；之后 subreaper 为 0，没有子进程。
- **进程内并发的检查**（`T2.gate.concurrent-checks`）：两个调用方同时各请求两次检查，四次都干净地完成，检查区间两两不重叠。
- **重复收养**（`T2.adoption.chain`）：一条四层的进程链（新会话）在组长退出之后逐层被收养，清扫用了 4 轮、经 pidfd 回收 4 个，身份不符 0 次，之后才回收组长。
- **拒绝路径**：基线中有存活的子进程或僵尸（`T2.baseline.live-child`、`T2.baseline.zombie-child`）→ 不设置 subreaper、不启动检查；注入身份不符（`T2.identity.mismatch`）→ 候选被放弃、从未收到信号、清扫不收敛（F1）；注入 `pidfd_open` 的 ENOSYS（`T2.pidfd.unsupported`）→ 清扫停止，不回退到按 PID 的信号。
- **F1 之后的资源**（`T2.F1.resources`）：注入“清扫放弃”之后，subreaper 已清除、组长保持未回收、被收养的进程仍是本进程的子进程，下一次检查因基线不为空而被拒绝（C-15）。
- **subreaper 只在区间内**（`T2.subreaper.cleared`）：区间之外产生的孤儿由 init 或其他 subreaper 收养（Ubuntu runner 上父进程为 1，WSL 上为 427），不回到本进程。
- **SIGCHLD 前提**（`sigchld.*`，Linux 与 macOS）：`signal.Ignore(SIGCHLD)` 的进程，其子进程退出后被内核自动回收（`waitid` 返回 ECHILD；Linux 的 `SigIgn` 显示 SIG_IGN），`signal.Ignored` 为真，S0 的守卫能发现它；从父进程继承的 SIG_IGN 被 Go 运行时覆盖（Linux 内核显示“未忽略”），子进程保持未回收（C-10）。

**反例**：

- `T2.gate.bypass`：区间内不经闸门启动的子进程与被收养的进程一样是本进程的子进程，按父进程清扫会把它当作检查的进程（原型以只列不杀的方式确认，没有终止它）。
- `T2.other-reaper`：进程中的另一段代码以 `wait(-1)` 回收了组长；组信号之前的核对（启动时间一致且仍是本进程的子进程）发现了它并拒绝发信号，但核对与信号之间的回收无法发现。

两者都说明，运行时的闸门与核对之外，还需要静态规则（C-9）。

**仍缺**：静态规则的实现；生产代码（`gitx`、`doctor`、macOS ACL 检查）改为经闸门；内核不支持 pidfd 的真实环境；真实的 PID 复用；处于不可中断睡眠的进程。

## 6. T3：真实 macOS

全部来自真实的 macOS runner（macOS 26.6.2，Darwin 25.6.0，arm64），两轮结论相同；没有用其他平台的实验代替。

| 问题 | 结果 | 场景 |
|---|---|---|
| 有时限的退出观察 | kqueue `EVFILT_PROC`（`NOTE_EXIT \| NOTE_EXITSTATUS`）：主进程 300 ms 退出，约 310 ~ 315 ms 观察到 `exit 7`；不退出时在 501 ~ 503 ms 的时限返回；没有留下等待的 goroutine | `emain.exit-unreaped`、`emain.bounded-timeout` |
| `WNOWAIT` 与回收的顺序 | 观察之后主进程仍是僵尸（`sysctl` 的 `SZOMB`），组不变；`waitid(P_PID, WEXITED \| WNOWAIT)` 经原始系统调用可用，两次都给出 `exit 5` 而不回收；之后 `Wait` 回收，退出码一致 | `emain.waitid-wnowait`、`emain.exit-unreaped` |
| 组身份与成员结束的确认 | 组长退出、成员存活时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束；`sysctl kern.proc.pgrp` 中不含僵尸的成员为空即确认结束，组长仍未回收；回收之后 `kill(-pgid, 0)` 为 ESRCH | `group.zombie-leader` |
| 只剩僵尸组长时的组信号 | `kill(-pgid, 0)` 返回 **EPERM**（Linux 上成功）。安全上没有问题：只有 PID 等于组 ID 的进程才能建立这个组，而未回收的组长占着这个 PID；但实现不能把 EPERM 解释为终止失败或组已消失（C-12） | `emain.exit-unreaped` |
| 脱离进程组的后代 | `setsid` 的后代在组被确认为空之后仍存活，没有 subreaper 可以发现它（草案 §8.2 的边界 1 在 macOS 上保留，D7） | `group.escape-boundary` |
| 输出的关闭与复制的收尾 | 与 Linux、Windows 相同：停止读取在 20 ~ 152 µs 内返回阻塞的 Read；D / J 阶段、封闭、O1 ~ O3、容量、别名、最后文档的场景都成立；阻塞模式读端的反例也复现 | §4 的各场景 |
| SIGCHLD | 与 Linux 相同（`sigchld.*`） | §5 |

**macOS 上不支持或没有测试的**：subreaper（macOS 没有对应机制，所以排他区间与收养清扫只在 Linux）；Specmetry 被 SIGKILL 之后组内进程的去向；真实终端中的信号；其他 macOS 版本与 Intel 机器。

## 7. T4：Windows 的失败链

全部在 Windows runner（Server 2025，26100）与本机 Windows 11（26300）上运行。失败由接缝注入：被注入的那一步不调用真实的 API。

| 场景 | 失败的步骤 | 处理 | 结束 | 用户代码 | 句柄 |
|---|---|---|---|---|---|
| `T4.start.fail-create` | 创建 | 没有进程 | — | 没有运行 | 只有 Job 句柄，已关闭；错误只记类别（not exist），不记文本 |
| `T4.start.fail-assign` | 加入 Job | 不恢复；`TerminateProcess`，对进程句柄有时限的等待 | 确认（进程不在 Job 中） | 没有运行（没有标记文件） | Job 与进程句柄已关闭；Go 的句柄由 `Wait` 释放；主线程句柄在创建时已由 Go 关闭 |
| `T4.start.fail-confirm` | 确认（`IsProcessInJob`） | 同上（实际已在 Job 中，按“未知”处理） | 确认 | 没有运行 | 同上 |
| `T4.start.fail-resume` | 恢复线程 | 不恢复；`TerminateJobObject`，确认 `ActiveProcesses = 0` | 确认 | 没有运行 | 同上 |
| `T4.chain.normal` | — | 恢复之后用户代码运行；`TerminateJobObject` | 52 ~ 91 µs 内 `ActiveProcesses = 0` | 运行 | — |
| `T4.teardown.terminate-fails` | 终止请求 | 不确认结束（F1）；关闭 Job 句柄（`KILL_ON_JOB_CLOSE`，再次请求） | 协议不确认；场景以自己的句柄**观察到**进程随后结束 | 运行 | — |
| `T4.teardown.query-fails` | 结束查询 | 不确认结束（F1） | 协议不确认；场景以句柄观察到进程已经结束 | 运行 | — |
| `T4.teardown.request-is-not-end` | — | 31 个进程的 Job，5 次 | 5 次都确认；请求之后第一次查询已为 0，请求到确认 0.46 ~ 1.25 ms | 运行 | — |

- “终止请求成功不等于进程已结束”在本轮的负载下没有观察到异步的窗口（第一次查询已为 0），只由注入（终止失败、查询失败）证明协议不会把请求当作结束（C-17）。
- 没有测试：重负载下的分布；经服务、计划任务、WMI 或 COM 启动的、不在 Job 中的进程；真实控制台事件。

## 8. T5：解释器适配

复用作者 X5 的 23 个字符串，每个都放在引号定界的 here-document（PowerShell 为 here-string）中原样打印回来；先用 `plain` 做一次健全检查，失败即为无效（工具问题，不是引号问题）。

| 平台 | 适配器 | 结果 |
|---|---|---|
| Ubuntu、macOS、WSL | `sh`、`bash`、`dash`、`pwsh`（argv） | 23/23（WSL 没有 pwsh） |
| Windows runner、本机 | Git for Windows `usr\bin\sh.exe`、`usr\bin\bash.exe`、`bin\sh.exe`，MSYS 引号 | 23/23 |
| | 同一 `sh`，MSVCRT（Go 默认）引号（反例） | 21/23：`two\\` 与 `\\server\share\` 中的成对反斜杠各丢一个 |
| | Windows PowerShell 5.1、pwsh 7.6，MSVCRT 引号 | 23/23 |
| | `cmd /d /s /c "…"`（原样） | `exit /b 7`、`exit /b 9009` 原样返回；LF、CRLF 的多行命令都只执行第一行（D3） |

- 安装布局：两个 Windows 环境都是 `C:\Program Files\Git`，由 `git --exec-path` 向上找到；没有测试其他版本、便携版、Cygwin 或独立 MSYS2。
- 没有测试：路径规范化（符号链接、联接、短文件名）、D2 的 PATH 构造、解释器的查找顺序。

## 9. 候选规格（待审查，不改变产品行为）

下列每一条都是对草案 §8 ~ §9 的细化候选，依据是本文的场景；冻结时由审查确认或否定。它们**不批准**任何产品选择，也不改变观测级、方案 B 或已冻结的契约。

| ID | 候选规格 | 依据（场景） | 涉及 |
|---|---|---|---|
| C-1 | **有时限的等待不早于时限结束**：以毫秒为单位的系统调用（`poll`）对剩余时间向上取整；`kevent` 用纳秒的 `timespec` | `emain.bounded-timeout`（§2.5 的缺陷 1） | T1、T3 |
| C-2 | **管道路径的读端只用 `os.Pipe` 建立**（非阻塞、可轮询）：关闭读端会使阻塞中的 Read 返回。不得包装外部给出的描述符，也不得把读端设为阻塞模式；阻塞模式下停止读取无效，描述符在 Close 之后仍然打开，直到 Read 返回 | `T1.pipe.stop-read-latency`、`T1.pipe.silent-holder`；反例 `T1.pipe.blocking-read-end` | T1、T6 |
| C-3 | **O3 复制任务的归属**：任务（goroutine）持有它的读端对象（Specmetry 已调用 Close，真正的关闭推迟到 Read 返回）与已封闭的复制入口（仍引用目的端，但不能写出）；Runner 在进程级登记全部未确认结束的复制任务。“不再写出”从封闭起成立；“资源已结束”要等 Read 返回，在生产中即最后一个写端持有者结束（或 Specmetry 进程结束） | `T1.O3.gate-lingering`、`T1.pipe.blocking-read-end` | T1、T6 |
| C-4 | **O3 之后的后续调用**：推荐“拒绝”——登记中仍有未确认结束的复制任务时，同一进程中新的检查执行在启动之前拒绝（退出码 6，与 F1 之后“残留时不复用”一致）。备选“允许”：输出上安全（入口已封闭，后续调用收不到旧输出），但任务与描述符会累积。按 C-2，`os.Pipe` 的读端在三个平台上都没有出现 O3，所以“拒绝”几乎没有代价 | `T1.O3.gate-lingering`（两种策略都执行） | T1、T6 |
| C-5 | **容量耗尽时继续读并丢弃**：复制入口返回容量错误之后，复制任务继续读取并丢弃其余数据（只计数，不保留内容），结果为 O2；检查不因管道写满而阻塞。“在第一次写错误时停止复制”会使检查被迫超时（反例） | `T6.pipe.capacity-discard`；反例 `T6.pipe.capacity-stop` | T6 |
| C-6 | **受控目的端的规则成立**：一个锁串行化全部写入；写入不阻塞；所有者入口与复制入口分开；封闭是锁下的一次设置；所有者写入成功时最后的文档完整、在最后；所有者写入失败或短写按 §8.6 C2 报告，不追加第二份文档 | `T6.dest.*`、`T6.pipe.alias-concurrent` | T6 |
| C-7 | **任意 Writer 在启动之前拒绝**（D20 (i) 的技术依据）：阻塞在 Write 中的复制任务无法停止，放弃它会让迟到的字节落在文档之后 | 反例 `T6.arbitrary-writer` | T6、D20 |
| C-8 | **进程级闸门（排他区间的强制方式）**：Specmetry 启动的每一个子进程（`gitx` 的 git、`doctor` 的 `git --version`、macOS ACL 检查的 `/bin/ls`、检查本身）都经同一个闸门：普通启动持共享租约，直到子进程被回收；检查持排他租约，等待全部共享租约结束（有等待中的排他请求时，新的共享请求等待，避免饥饿），确认本进程没有任何子进程（僵尸也算）之后才设置 subreaper，收尾完成之后清除。两个进程内调用方同时请求时被串行化 | `T2.gate.exclusive-interval`、`T2.gate.concurrent-checks`、`T2.baseline.*` | T2、D7 |
| C-9 | **闸门之外的静态规则**：只有闸门所在的包可以导入 `os/exec`、调用 `os.StartProcess` 或 `syscall.ForkExec`；任何代码都不得调用 `wait(-1)` / `Wait4(-1)` 或忽略 SIGCHLD（`signal.Ignore(syscall.SIGCHLD)`）。由 lint（例如 depguard / forbidigo）与一个列出导入者的测试强制。运行时只能部分发现违规：旁路启动的子进程与被收养的进程按父进程无法区分；其他回收者在“核对组长”与“发信号”之间的动作看不到 | 反例 `T2.gate.bypass`、`T2.other-reaper`、`sigchld.ignored` | T2 |
| C-10 | **SIGCHLD 前提**：S0 检查 `signal.Ignored(syscall.SIGCHLD)`，为真则不启动（退出码 6）。从父进程继承的 SIG_IGN 不构成问题：Go 运行时启动时为 SIGCHLD 安装自己的处理函数（只有 SIGHUP、SIGINT 保留继承的 SIG_IGN），Linux 上内核随之显示“未忽略” | `sigchld.*`（Linux 与 macOS） | T2、T3 |
| C-11 | **每次组信号之前核对组长**：组长的 PID 仍是本进程未回收的子进程，并且启动时间与启动时记下的一致；否则不再向这个组发任何信号（F1）。这只是发现手段，不能代替 C-9 | `T2.other-reaper`、`emain.exit-unreaped` | T1、T2、T3 |
| C-12 | **确认结束只看成员枚举，不看信号调用的返回**：Linux 用 `/proc` 的进程组成员（不含僵尸），macOS 用 `sysctl kern.proc.pgrp`（不含僵尸），Windows 用 `ActiveProcesses`。macOS 上组里只剩未回收的组长时，`kill(-pgid, 0)` 返回 EPERM（Linux 上成功），所以 EPERM 既不能当作“终止失败”，也不能当作“组已消失” | `emain.exit-unreaped`（macOS 与 Linux 的差别）、`group.zombie-leader` | T1、T3 |
| C-13 | **macOS 的 E-main**：对主进程注册 kqueue `EVFILT_PROC`（`NOTE_EXIT \| NOTE_EXITSTATUS`），以有时限的 `kevent` 等待，不回收、不留下等待任务；注册时进程已退出（ESRCH）则以 `sysctl` 确认它是僵尸。`waitid(P_PID, WEXITED \| WNOWAIT)` 经原始系统调用在所测 macOS 上可用，可作为取得状态的第二条途径 | `emain.*`（macOS） | T3 |
| C-14 | **Linux 清扫的身份与有界等待**：每个候选经 pidfd 绑定后再次核对父进程与启动时间；信号经 `pidfd_send_signal`；回收先对 pidfd 做有时限的 `poll`，再 `waitid(P_PIDFD, WEXITED \| WNOHANG)`，不使用会无限阻塞的 `waitid`。身份不符 → 放弃该候选、不发信号；`pidfd_open` 不可用 → 停止，不回退到按 PID 的信号；都进入 F1 | `T2.adoption.chain`、`T2.identity.mismatch`、`T2.pidfd.unsupported` | T2 |
| C-15 | **F1 之后的宿主资源**（草案 §8.5 F1 候选的细化）：关闭本次打开的 pidfd；清除 subreaper；组长保持未回收（僵尸），使组 ID 保持保留、基线保持不为空；释放闸门。被收养的进程仍是本进程的子进程，所以同一进程中下一次检查因基线不为空而被拒绝。这些僵尸与残留进程在 Specmetry 进程结束时交给系统 | `T2.F1.resources` | T1、T2、D19 |
| C-16 | **Windows 启动链的失败处理**（草案 §8.5 S1 的细化）：创建失败 → 没有进程，只关闭 Job 句柄；加入 Job 或确认失败 → 绝不恢复，`TerminateProcess` 后对进程句柄做有时限的等待；恢复失败（已在 Job 中）→ `TerminateJobObject` 后确认 `ActiveProcesses = 0`。之后关闭本 Runner 的 Job 与进程句柄，`Wait` 释放 Go 的进程句柄（主线程句柄在创建时已由 Go 关闭）。所有这些情形中用户代码都没有运行 | `T4.start.*`、`T4.chain.normal` | T4 |
| C-17 | **Windows 的终止请求不等于结束**：只有 `ActiveProcesses = 0` 才确认结束；`TerminateJobObject` 失败或查询失败 → F1，即使进程事实上已经结束。关闭 Job 句柄（`KILL_ON_JOB_CLOSE`）是再次请求，不是确认 | `T4.teardown.*` | T4 |
| C-18 | **命令传递的适配器**（草案 §5.5，按 runner 上的版本确认）：Unix 的 `sh` / `bash` / `dash` / `pwsh` 用 argv；Windows 的 Git for Windows `sh` / `bash`（含 `bin\sh.exe` 包装程序）用 MSYS 引号规则，`powershell` / `pwsh` 用 MSVCRT 规则，`cmd` 原样放在 `/d /s /c "…"` 中。Go 默认（MSVCRT）规则交给 MSYS 时丢失成对反斜杠中的一个（反例） | `T5.adapter.*` | T5、D1 ~ D3 |

## 10. 产品决定单（全部未批准）

按行为分组；“推荐”沿用草案 §17.3，“本轮证据”只写原型新增的部分。**没有一项因本文而获批**。

| 组 | 决定 | 推荐 | 代价 | 本轮证据 |
|---|---|---|---|---|
| **解释器与命令** | D1 默认解释器与不可用时的行为 | (a) 所有平台 `sh`，不回退；不可用时整批不执行，退出码 6 | 没有 Git for Windows `sh` 的机器上，省略 `shell` 的检查不可用 | Windows runner 与本机的 Git for Windows 2.56.0 都提供 `usr\bin\sh.exe` 与 `bin\sh.exe`，MSYS 引号 23/23（C-18） |
| | D2 Windows `sh` / `bash` 的入口、PATH 与来源 | W2；只取 `gitx` 所用 git 的 Git for Windows | 偏离 architecture §6.4 的环境规则 | 由 `git --exec-path` 找到安装根，两个入口都可用；PATH 的构造本轮没有测试 |
| | D3 `cmd` 的多行命令 | (a) 拒绝 | 多行批处理要改写 | Windows Server 2025（26100）与 Windows 11（26300）上 LF、CRLF 都只执行第一行（`T5.adapter.cmd-raw`） |
| | D4 `sh` / `bash` 命令中的 CR | (a) 拒绝 | CRLF 写成的命令要人工修订 | 本轮没有新证据（X4） |
| | D5 EV-04 与 `not_found` | 修订后的 A（按最终状态） | 被命令串掩盖的失败不会被识别 | 本轮没有新证据（X3、X10） |
| **进程树与平台支持** | D7 进程树保证的范围；Linux 排他区间不成立时的处理 | 先完成 T1 ~ T4；在 (a) 如实限定与 (b) 不支持的平台不提供之间选择；Linux 排他区间不成立时不启动 | (a) 缩小保证；(b) 缩小平台 | macOS：组终止、不回收的观察与组身份规则都有了真实证据（C-12、C-13），`setsid` 的后代仍无法发现（`group.escape-boundary`），没有 subreaper；Linux：排他区间可由闸门强制（C-8），但需要静态规则（C-9）；拒绝路径都已执行（`T2.baseline.*`） |
| | D11 主进程正常退出之后仍在运行的后代 | (a) 终止 | 有意留下的后台进程会被终止 | 所有管道场景都在正常退出之后终止并确认进程树，再进入 S6 |
| | D19 `T_kill` 内没有确认结束 | (a) 尝试记录为不合格 | 扩展 `context_changed` 的使用情形；时间字段不是实际结束时刻 | F1 的输出与宿主资源（`T1.pipe.F1-skip-drain`、`T2.F1.resources`、C-15）；持久化的叠加本轮没有测试 |
| **记录与退出语义** | D6 结束上下文读取失败 | (a) 记录为不合格 | 环境故障产生一条不合格的执行事实 | 本轮没有新证据 |
| | D8 取消 | (a) 不记录，退出码 3；退出码优先级 1 > 6 > 3 > 10 > 0 | 被取消的一次在事实中看不到 | 本轮没有新证据（真实控制台没有测试） |
| | D9 批次汇总 `is_target_snapshot` | (b) 每一次的 `head_start` 都等于 `S` | 一个布尔值不能完整表达混合批次 | 不属于机制原型 |
| | D13 配置的固定时刻 | (a) 命令开始时一次读取 | 前一次执行期间修订的 AC 仍按旧配置执行 | 不属于机制原型 |
| | D18 `--all` 选不到 AC 且 worktree 未登记 | (a) 只做登记的写事务 | 一次没有执行的命令写入一条事实 | 不属于机制原型 |
| **输出** | D20 输出的支持契约与转发不完整 | (i) 继承句柄与受控目的端，其他 Writer 拒绝；(a) 转发不完整时照常记录、停止、退出码 6 | 一次转发故障中止整个批次；进程内调用的输出对象受限 | 受控目的端的规则、O1 ~ O3、迟到输出、别名、容量与最后文档在三平台成立（C-2 ~ C-7）；O3 之后的策略另见 C-4（技术规格，不是产品选择） |

## 11. 留给生产阶段的测试

原型证明的是机制在隔离夹具上可行，不是生产实现正确。获批之后、本层验收之前（草案 §17.2 的第二个时点）仍要：

- 把 C-1 ~ C-18 接入生产的 Runner / CLI，并逐分支测试：正常、超时、取消、启动失败、后代残留、确认失败、复制错误，以及草案 §8.6 的每一行（含持久化的 B、C 表）。
- 闸门（C-8）与静态规则（C-9）的实现：`gitx`、`doctor`、macOS ACL 检查改为经闸门启动；lint 规则与导入者测试；进程内并发的 `Execute` 测试。
- 继承路径的进程级测试：真实终端与重定向、`--json` 时 stdout 只有一份文档；继承句柄的限制（Windows 句柄列表、Unix close-on-exec）。
- 迁移 0009、`check-run.v1` 的 Schema 与示例、事实一致性、`dbdump` 的映射。
- 三平台 CI 上带故障注入的回归：本文的场景是它们的起点，但生产测试要经生产代码路径，而不是原型的副本。

## 12. 仍未验证的边界

- **真实的 F1**：处于不可中断睡眠、在 `T_kill` 内不结束的进程没有构造；F1 都由注入（放弃清扫、跳过终止、注入失败）模拟，只证明分支的结果，不证明真实进程、句柄与属性的生存期（草案 §8.5 F1 的说明）。
- **真实控制台**：Ctrl+C、控制台关闭、SIGTERM 的竞争没有测试；`/dev/tty`、`CONIN$` 没有测试。
- **Specmetry 本身被 SIGKILL**：Linux 子树的去向（subreaper 不覆盖）、macOS 的残留没有测试；Windows 的 `KILL_ON_JOB_CLOSE` 只有作者早先的 X2。
- **强制的 PID 复用**：身份不符只由注入模拟；真实复用下的信号误投只有推理。
- **Windows 的异步结束**：测试负载下请求之后立即查询 `ActiveProcesses` 已为 0，没有观察到大于 0 的窗口；“请求不等于结束”只由注入证明。重负载下的分布没有测试。
- **其他版本与布局**：只有所列的 macOS 26.6.2（arm64）、Windows Server 2025（26100）与 Windows 11（26300）、Ubuntu runner（内核 6.17）与 WSL2（内核 6.18）、Git for Windows 2.56.0；其他 Git for Windows 版本与安装方式、Cygwin、独立 MSYS2、旧内核（没有 pidfd）没有测试。
- **静态规则（C-9）的实现**：本文只给出候选与反例，没有实现 lint。
- **闸门覆盖的完整性**：原型中的共享租约是辅助进程与 `git --version`，没有改动 `gitx`；生产代码中是否还有其他启动子进程的路径，要在实现时由 C-9 的导入者测试列出。
- **`-race` 只在 Linux**；macOS 与 Windows 的原型没有开启竞态检测。
