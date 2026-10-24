# WP2.5 冻结之前的机制原型（test-only）

> 状态：**原型，PR #33 已合并为 `5e9a42c`**，合并核验与 master push CI 正式成功。之后修复 N4、N6 的独立小 PR #34 经 Review 081、082 为 **NO-GO**（B1：停止读取可能在阶段 J 之前阻塞，及其停止结果的生命周期；N4-R），N6 关闭；**已定向修复，待独立复审**（[review-resolution.md §89 ~ §98](review-resolution.md)）。依据 [§88](review-resolution.md) 的授权：PR #32 合并核验与 master push CI 全部成功之后，从已核验的 master 另开 PR，把 [WP2.5 规范草案](wp2.5-check-runner.md) §17.2 中 T1 ~ T6 冻结之前的关键可行性问题转成可审查的证据。
>
> - [Review 075](reviews/codex-review-075.md) 对 `932b4a8` 为**技术 NO-GO**：B1，实验的有效性门禁没有兑现它声明的拒绝规则（畸形或缺失的报告、缺失的目标负载仍可得到 confirmed）。第一轮修复见 §2.7。
> - [Review 076](reviews/codex-review-076.md) 对 `3ddd14f` 仍为**技术 NO-GO**：原来的两个反例已修复，但 B1 有两项残余——B1-R1，一次观察失败被之后的成功覆盖，仍得 confirmed；B1-R2，Linux 短链的提前拒绝没有收尾已经启动、脱离原组的链节点。第二轮修复见 §2.8。
> - [Review 077](reviews/codex-review-077.md) 对 `acf699b` 仍为**技术 NO-GO**：两个精确反例已有实质修复，但还有两项阻塞——B1-R1-S，共享的观察错误仍可能丢失（Linux 清扫在 pidfd 绑定之后的读取、场景起点的子进程枚举）；B1-R2-W，新的外层见证不是 fail-closed（写入失败留下空见证仍通过，无效身份被当作已结束）。第三轮修复见 §2.9。
> - [Review 078](reviews/codex-review-078.md) 对 `f82dfb8`：**B1-R1-S 关闭**；B1-R2-W 的原反例已修复，但残余 B1-R2-W-R——负向自检找到预期的拒绝类别即通过，忽略同时发生的外层自身失败（补救失败、仍存活、普查或读取失败、注入未命中）；**技术 NO-GO**。第四轮修复见 §2.10。
> - [Review 079](reviews/codex-review-079.md) 对 `8910825`：**B1-R2-W-R 关闭**，B1-R1-S 保持关闭；新增 **B2**——Unix 的保险清理可能向归属没有被确认的见证 PID 发信号；**技术 NO-GO**；最终常规 CI 为 7/8，合并条件没有满足。本版已按 B2 修复（§2.11）。
> - [Review 080](reviews/codex-review-080.md) 对 `0621316`：**B2 关闭**，B1-R1-S、B1-R2-W-R 保持关闭，没有新的 BLOCKER；**技术 GO**。最终常规 CI run 39979587006 为 attempt 1、completed/success、8/8，合并条件满足，可以建议负责人合并。结论限定在本原型声明的对象来源与受控继承 / Job 前提内（§2.11、§13）。Review 075 N1 ~ N3 保持审查的结论；Review 076 N4 与 Review 078 N6 仍没有修复（§13）；Review 077 N5 与 Review 078 N7 关闭。
> - **合并之后**（§96）：PR #33 以 merge commit 合并为 `5e9a42c`（锁定收尾 HEAD `38d15ef`），合并树与收尾 HEAD 相同，master push CI run 39988732786 为 8/8。Review 076 N4 与 Review 078 N6 在分支 `test/wp2.5-prototype-n4-n6` 的独立小 PR 中修复（§2.12），**待独立审查**；不宣布关闭。
> - [Review 081](reviews/codex-review-081.md) 对 `4e009a6`：**技术 NO-GO**。B1（BLOCKER）：同步的 Close 可能在阶段 J 开始之前阻塞（Windows 上可能一直到有数据或 EOF），阶段 J、封闭与 O3 都没有自身的出口；N4-R：“持有者仍在运行”不能证明写入没有失败；N6 在原时钟口径的范围内关闭。已定向修复（§2.13），**待独立复审**。
> - [Review 082](reviews/codex-review-082.md) 对 `0f1f20b`：**技术 NO-GO**。B1 的原反例已修复，但完整契约不能关闭：B1-Ra，停止在运行返回之后才出错时，错误随遗留丢失，场景仍可 confirmed；B1-Rb，Windows 的取消与关闭自有句柄的错误被丢弃。N4-R 的组合故障（一路迟到字节、另一路写入失败、报告失败）仍得到 refuted（NON-BLOCKER）。N6 保持关闭。已定向修复（§2.14），**待独立复审**。
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
- **第二轮修复后的证据**：原型代码 `8fea6c6`，三平台诊断 run 39887528435 为 success，现为历史观测（§3.4）。
- **Review 077 的两项残余**（第三轮，§2.9）：
  - **B1-R1-S**：清扫在绑定之后读不到状态时记为“身份不符”，下一轮成功就回收；场景起点的子进程列表读不到时当作空列表。同源的还有 subreaper、子进程计数、清扫的 `poll` / `waitid`、Windows 的 Job 成员与句柄等待、Unix 清理的回收、macOS 早退出的状态。
  - **B1-R2-W**：见证由辅助进程自己写，写入、关闭错误与短写都被忽略；空见证、未知 kind、重复、截断的尾行与 `unknown` 身份都被接受；身份不同就当作已结束；进入辅助进程代码之前被结束的进程没有记录。
- **回归先行**（`8f42e58`）：一次性的观察故障注入在其余共享入口；见证本身的写入、短写、身份、截断、重复、未知 kind、删除、故意的遗留与早退。旧门禁上，三平台诊断 run 39951830770 为 failure：Ubuntu 35 项中 12 项、macOS 26 项中 8 项、Windows 31 项中 10 项失败，都是新增的回归。
- **修复**（`e77af94`）：
  - 对应表 A（§2.9）中每一个共享入口的失败都进入统一终判；
  - 见证改为创建者记录、辅助进程回执，每次写入都核对，失败带外通报；
  - 内层有 `root` 记录与封条；外层严格解析并核对身份；
  - 外层在任何补救之前，按标记或 Job 普查存活进程，Windows 上另核对 Job 的已创建进程数；
  - 登记按 PID 与身份匹配，`missing` 改称 `shortfall`。
  - 隔离变异中，撤回每一项修复，对应的回归都会失败。
- **第三轮修复后的证据**：原型代码 `e77af94`，三平台诊断 run 39954957954 为 success，现为历史观测（§3.4）。
- **Review 078 的残余**（第四轮，§2.10）：B1-R1-S 关闭。B1-R2-W-R：负向自检的判定只看预期的拒绝类别，同时发生的补救失败、仍存活、普查或读取失败都被忽略。
- **回归先行**（`4efc9fc`）：`TestGateCombos` 在真实的门禁回归上，只对外层被测路径的一步注入故障，并加一项独立于被测路径的保险清理。旧判定上，三平台诊断 run 39964760402 为 failure：三平台都是 10 个组合中同样的 6 个失败，其中被测自检在外层失败的同时报告 PASS。
- **修复**（`31c0b29`）：
  - 判定拆成分别保存的维度：预期拒绝、证据的其他问题、外层自己的失败、补救、本轮进程的最终状态；全部满足才 PASS（§2.10 的判定表）；
  - 保险清理在判定之后，只会使结论变差。
  - 隔离变异中，撤回关键的组合判定，组合或判定表都会失败。
- **第四轮修复后的证据**：原型代码 `31c0b29`，三平台诊断 run 39966556540 为 success，现为历史观测（§3.4）。
- **Review 079 的 B2**（第五轮，§2.11）：Unix 的保险清理接收了只凭见证记录认出的进程，只核对启动时间就发信号；身份准确不等于属于本轮。
- **回归先行**（`f1b8566`）：外层的信号经一个可以只记录的接缝；一条伪造、格式合法的 `created` 记录指向组合自己的夹具进程。旧代码上，三平台诊断 run 39976843072 为 failure：夹具在三平台都被交给动作入口，Unix 上保险清理向它发出数百次信号意图（只记录，没有发送）。
- **修复**（`8a2b97c`）：
  - 只对已证明归属的进程按 PID 动作：普查找到的，或只凭记录存活、但在那一刻被证明的（Unix：同一进程且带本次检查的标记；Windows：同一进程且在外层 Job 中）；
  - 其余的拒绝，记为 `ownership` 工具错误，最终状态为未知；
  - 保险清理只复用已证明的身份。
  - 隔离变异中，撤回关键的归属检查，组合或分拣表都会失败。
- **修复后的证据**（原型代码固定为 `8a2b97ccd521c6af587baf4e3f45421d68dafcd9`）：三平台诊断 run 39977737240 为 success。
  - 场景：Ubuntu（带 `-race`）38/38、macOS 27/27、Windows 29/29 confirmed，没有 refuted、invalid 或 unsupported。
  - 门禁回归：36/36、27/27、32/32 通过。
  - 组合：三平台各 13/13；夹具在三平台都没有被动作。
  - 直接检查：分拣表 5 行、判定表 13 行、见证判定 24 例、身份、启动点，三平台都通过。
  - 作者本机见 §3.1。
- **Review 080**（对 `0621316`）：B2 关闭，技术 GO；最终常规 CI 8/8。审查方在 Windows 的隔离副本上复验了旧、新门禁的组合，并在真实调用链上注入一次归属读取错误（不是原生故障的动态复现）；Unix 的结论来自诊断日志。N4、N6 仍没有修复（§13）。
- **N4 / N6**（PR #33 合并之后的独立小 PR，§2.12）：
  - N4：持有者的 `epipe` 记录写不出时，不再记为 refuted；没有记录时先读持有者的状态，仍在运行为 refuted，已结束为 invalid。
  - N6：阶段 D、阶段 J 在同一个时钟上限定与计量；早到的唤醒不结束阶段；stopwatch 的读数另记，不作判定。
  - 回归先行 `91229b5`，三平台诊断 run 39991866363 为 failure（预期）；修复代码固定为 `ac3b3be`，三平台诊断 run 39994591073 为 success。
  - `5054e54` 的直接检查在 Windows 上有一个自身的缺陷（操作系统管道的关闭时序），已改用内存管道（§2.12）。
- **Review 081**（§2.13）：
  - B1：停止读取纳入阶段 J，由运行持有的停止任务完成；阶段 J 结束即封闭，未完成的停止与复制任务交给调用方。Windows 上停止任务在 Close 等待期间重复取消。
  - N4-R：没有记录时，只有“持有者的迟到字节确实到达了保留的读端”才是 refuted；其余为 invalid。
  - N6 经审查关闭。回归先行 `bde00fc`（run 40066182430 为 failure，预期）；修复代码固定为 `125c4f7`，run 40066910980 为 success。
  - T1 / T6 的结论随之收窄：读取始终不进入系统调用时停止挂起，资源没有上界（§2.13）。
- **Review 082**（§2.14）：
  - B1-Ra：停止的结果随任务移交，接手者在结论之前读取；运行返回之后才到的错误也使观察失败；O3 登记保留带错误的结束。
  - B1-Rb：Windows 的 `CancelIoEx` 只把 `ERROR_NOT_FOUND` 当作预期竞态，其他失败与关闭自有句柄的失败都是停止的错误。
  - N4-R：refuted 需要持有者被询问时的确认，与每一路经保留读端的迟到字节。
  - 回归先行 `e4eadfa`（run 40072151282 为 failure，预期；macOS 另有一项普查的 EINVAL）；修复代码固定为 `2413634`，run 40074633075 中 Ubuntu、Windows 为 success，macOS 因同类的普查 EINVAL 有一项旧门禁回归不通过（§2.14，没有重跑）。
- **旧的运行**（run 39815658463、39816938304、39835302375、39887528435、39954957954、39966556540 与此前的常规 CI）保留为历史观测（§3.4）。
- **按门槛**：在原型的范围内，T1 / T6 的管道收尾与受控目的端、T2 的 Linux 排他区间候选、T3 的 macOS 退出观察与组身份、T4 的 Windows 启动链部分失败处理、T5 的所测适配器，都有修复后的证据；都只是**局部支持**，冻结之前仍缺的规格与原型见 §11。
- 14 项产品选择仍待负责人决定（§10）；C-4 与 D20 的关系见 §9。

## 1. 范围与隔离

**包含**：

- `prototype/wp25/`：一个独立的 Go 模块，只有 `_test.go` 文件，依赖只有 `golang.org/x/sys` v0.49.0（与 Specmetry 相同的版本）。`gate_test.go` 是有效性门禁本身的回归，`report_test.go` 是报告协议，`witness_*_test.go` 与 `census_*_test.go` 是外层门禁检查自己的证据（见证、带外通报、普查，§2.8、§2.9），`gate_combo_test.go` 是外层自检的组合判定（§2.10），`witness_check_test.go` 是外层规则与判定的直接检查，`clock_check_test.go` 是输出步骤时钟的直接检查（§2.12），`keep_*_test.go` 为 F1 场景的对照另持读端。
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
- 辅助进程自己的失败各有退出码：子进程起不来 90、记录写不出 91、等不到子进程的记录 92、输出写不出 93、门禁检查之下见证记录做不成 94（§2.9）。场景核对辅助进程的退出状态，不一致即为负载没有建立。
- 测试输出只有每个场景一行的 `WP25RESULT`（结论、预测、观察、启动数、工作量、清理）、每项门禁回归一行的 `WP25GATE`，以及开头的 `WP25META`、结尾的 `WP25SUMMARY`。门禁回归的内层运行另外打印注入命中的 `WP25INJECT` 与见证的封条 `WP25WITNESS`。不保存任何输出内容：进入受控目的端的字节只在内存中用于比较，被丢弃的字节只计数。

### 2.2 进程的归属与清理

- **归属**：场景只操作它启动的进程（Go 的 `exec.Cmd`），以及它**核验为本次检查的**、由自己的辅助进程报告的后代。报告中的 PID 本身不是授权：
  - Unix：该进程的父进程链能追到场景启动的主进程，或者它在主进程的进程组中、并且主进程仍是本进程未回收的子进程；
  - Windows：该进程在本次检查的 Job 中（`IsProcessInJob`）。

  核验不了即为 invalid（“ownership not established”），不扩大清扫范围补救。
- **报告的进程在记录到达时即核验并登记**（`awaitOwned`，Review 076 B1-R2）：不等全部记录到齐。所以拒绝（负载不完整、报告损坏、归属核验不了）发生时，已经看到的每一个进程都已登记、会被清理；没能做成的预期登记（没有报告，或报告读不懂）计为**缺额**（`shortfall`，第二轮中称“未入账”，Review 077 N5），清理因此不称“已核验”，场景为 invalid。缺额是数量：它也计入从未创建的预期进程，不是已启动对象的名单（那由外层的见证与普查给出，§2.9）。报告中的 PID 仍然只有经归属核验之后才可操作。
- **身份**：核验时记下身份（Linux 的 `/proc` 启动时间，macOS 的 `sysctl` 启动时间，Windows 打开并持有句柄）；每次操作之前再次核对，不一致就不再操作。
- **状态的三种读数**：可靠地不存在（Linux 的 ENOENT / ESRCH，macOS 的空回答）、存在、**未知**（其他读取失败、解析失败、回答长度不对）。未知从不当作“已结束”“不存在”或“空组”：清理不算核验，观察为 invalid，排他区间的基线或清扫停止。
- **观察失败不可撤销**（Review 076 B1-R1，Review 077 B1-R1-S）：
  - 一个进程的第一次观察失败被保存，之后的成功观察不清除它；清理仍然结束并回收这个进程，但这次运行为 invalid。
  - 同样记入统一终判、之后的任何结果都不能恢复的，见 §2.9 的对应表 A：
    - 等待主进程退出、组长的状态；
    - 场景结束时组或 Job 不能确认为空；
    - 清扫的列表、绑定前后的两次状态读取、`poll` 与 `waitid`；
    - 场景起点的子进程列表、subreaper 与子进程计数的读取；
    - Windows 的 Job 成员读取与句柄等待；
    - Unix 清理中的回收与 `stop` 的读取；
    - macOS 早退出的状态。
  - “清理已经完成”与“这次运行有效”分开记录。
- **正常的“尚未”**：
  - 以下都是有期限的等待，不是失败：进程仍在运行、报告的记录尚未到齐、最后一行仍在写入、`waitid` 的回答里还没有进程、门禁之下辅助进程等待自己的创建记录。期限见各场景：报告等待 5 s 或 20 s，清理 5 s，进程树确认与清扫 `T_kill`，创建记录 10 s。
  - 唯一允许重复的是被 EINTR 打断的 `poll` 与 `kevent`，也只到期限为止。
  - 到期仍未满足，才按负载没有建立或清理没有确认处理。读取错误从不当作“尚未”。
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
| `invalid` | 这次运行不能作为任何证据：<ul><li>辅助进程没有启动或崩溃；</li><li>报告缺失、损坏或不符合规格；</li><li>**场景声明的负载没有建立**（`needLoad`）；</li><li>归属核验不了；</li><li>观察读不到（**任何一次**，任何共享入口，之后的成功不恢复，§2.9 表 A）；</li><li>拒绝时有预期的登记没有做成（缺额）；</li><li>Linux 上留下本进程的子进程，或起点的子进程列表读不到；</li><li>到达硬时限、清理没有确认，或在预测检查之前失败</li></ul> |

- **负载与结果分开**：场景先核对它需要的角色、数量与阶段（辅助进程的成功退出码、报告的每个后代、停止时阻塞的复制任务、Windows fanout 的 30 个后代与 31 个活动进程等），记为 `load:`；负载没有建立是 invalid，完整负载下的不同结果才是 refuted。“启动过一个进程”“有过一次工作”不能代替场景所需的负载；清理成功也不说明负载曾经完整执行。
- 场景分为**正常路径**、**反例**、**故障注入**与**拒绝路径**四类（结果中的 `kind`）。反例的“confirmed”表示反例被复现，即对应的朴素做法确实不成立。
- `refuted` 与 `invalid` 都使测试失败（workflow 显示为失败）；`unsupported` 单列，不计为通过。没有任何场景运行时，整次运行无效。
- **门禁回归**（§2.7）的通过另行计数（`WP25GATE`、`gate_passed`），不是任何机制的 confirmed。
- **外层证据的有效性**（§2.9）：外层门禁检查也判断它自己的证据，即见证、封条、带外通报与普查。证据无效时，这项门禁回归不通过，整次诊断失败；只看内层的 confirmed、exit 0 不够。只有专门注入见证故障的自检，以“正确地判定证据无效”为通过。
- **外层自检的组合判定**（§2.10）：每项门禁回归按分开保存的维度判定，全部满足才通过：
  - 预期拒绝；
  - 证据的其他问题；
  - 外层自己的失败；
  - 补救是否需要、是否执行、是否成功；
  - 本轮进程最终是否已确认结束。

  找到预期的拒绝理由只证明第一项。外层自己的失败（普查、读取、注入命中、补救与结束确认）不能被任何预期拒绝豁免。判定之后的保险清理永远不能使检查通过。

### 2.5 诊断 workflow

`.github/workflows/wp25-prototype.yml`：

- 只在推送到 `proto/wp2.5-prototype` 分支或 N4、N6 修复的分支 `test/wp2.5-prototype-n4-n6`（§2.12）、并且改动了 `prototype/wp25/**` 或该 workflow 时运行；只改文档不会触发；没有手动触发，所以没有盲目重跑；不取消进行中的运行，失败的历史保留。
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

### 2.9 Review 077：共享观察错误与见证的有效性

**审查结论**（第一、三、四节）：Review 076 的两个精确反例已有实质修复，但还有两项阻塞：

- **B1-R1-S**：Linux 清扫在 pidfd 绑定之后的那次状态读取，以及新增的场景起点子进程枚举，仍会丢失观察错误。
- **B1-R2-W**：外层见证不是 fail-closed。写入失败留下空见证，外层仍 PASS；无效的启动身份被当作原进程已结束。

N5（`missing` 的术语）为 NON-BLOCKER；Review 076 N4 保持。

**根因**：

- **B1-R1-S**：共享的观察入口在若干调用点，把“读不到”变成了另一种读数。
  - 清扫在绑定之后读不到状态时记为身份不符（`mismatched`），下一轮成功就回收，终判不知道这次失败。
  - 场景起点的子进程列表读不到时返回空列表，结束时的“留下的子进程”检查便与空集合比较。
  - 同源的写法：
    - subreaper 的读取失败返回 0；
    - 子进程计数读不到时记为“不同”（refuted 而不是 invalid）；
    - 清扫中 `poll` 与 `waitid` 的错误被丢弃（`waitid` 失败一次，下一轮成功回收）；
    - `waitid` 的成功不核对 `si_pid`；
    - Windows 失败链中 `IsProcessInJob` 与句柄等待的错误被丢弃；
    - Unix 清理中回收（`Wait4`）的失败、以及 `stop` 读不到状态，没有保存；
    - macOS 早退出时 `waitid` 的失败被写成一个状态字符串。
- **B1-R2-W**：
  - 见证由辅助进程运行到 `runHelper` 时自己写入。打开、写入、关闭的错误与短写都被忽略；自己的身份读不到时写下 `unknown`。
  - 读取端接受空文件、未知的 kind、重复、没有换行的尾行与 `unknown` 身份。
  - 外层只遍历读到的记录，空集合即通过；身份不同就当作原进程已结束，`unknown` 也走这一分支。
  - 创建之后、进入辅助进程代码之前就被结束的进程，没有任何记录。
  - 外层给内层结束之后 10 s 的宽限，遗留进程可以在宽限内按自己的定时器结束，而不算遗留。

**两条共享链的对应表**（审查要求的交付内容）：

A. 观察入口 → 返回状态／错误 → 调用者 → 场景失败的累积 → 最终结论

| 共享入口 | 返回 | 调用者与失败的传播 | 允许的等待或重试（条件、期限） | 最终由谁拒绝 |
|---|---|---|---|---|
| `procInfo`（Linux `/proc/<pid>/stat`；macOS `sysctl kern.proc.pid`） | 存在、可靠不存在（Linux 的 ENOENT / ESRCH，macOS 的空回答）、未知（其他错误、解析失败、回答长度不对） | <ul><li>`pidProc.ended`、`pidProc.stop`：首次失败存入 `obsErr`，不清除；</li><li>`ownDescendant`（含父进程链的每一级）、`startTree`：`invalidf`；</li><li>组长的检查（`leaderHeld` 等）：`flag`；</li><li>清扫的两次读取：`sweepResult.obsErr`；</li><li>外层的普查：普查错误</li></ul> | 进程仍在运行是“尚未”：清理 5 s，确认 `T_kill`；“未知”从不重试成“已结束”或“另一个进程” | 场景 `done()`：清理中的 `obsErr`、`flag` 的记录、`runCheck` / `abort` 对清扫 `obsErr` 的 `flag`，结论为 invalid、exit 1 |
| `allPIDs` / `processes` / `childrenOf` / `groupLive`（Linux） | 列表或错误；读不到不是空列表 | <ul><li>进程树确认 `confirm`：`invalidf`；</li><li>提前收尾、`abort`：`flag`；</li><li>清扫：`obsErr`；</li><li>场景起点的子进程列表：`baseErr`；</li><li>`childCount`：`flag`；</li><li>排他闸门的基线：拒绝区间（被测机制自己的保守响应）</li></ul> | 无重试 | 同上 |
| `exitObserver.wait`（Linux pidfd `poll` + `waitid WNOWAIT`；macOS `kevent`，早退出时 `waitid`） | （已退出，状态，错误） | `waitExit`：`flag`；macOS 早退出的 `waitid` 失败返回错误，不再写成状态 | EINTR 在期限内重复；期限到仍未退出是“尚未” | 同上 |
| 清扫的 `pidfd_open`、`pidfd_send_signal`、`poll`、`waitid` | 错误；`waitid` 的 `si_pid` | <ul><li>`pidfd_open`、发信号失败：被测机制停止（`err`，F1，不回退到按 PID 的信号）；</li><li>`poll`（EINTR 之外）、`waitid` 失败：观察失败（`obsErr`）；</li><li>`si_pid` 为 0：没有可回收的，“尚未”</li></ul> | `poll` 的 EINTR 在期限内重复；没有回收的僵尸在下一轮再试，到 `T_kill` 为止，但此前的失败已保存 | 同上 |
| `prctl(PR_GET_CHILD_SUBREAPER)` | 值或错误 | `subreaper(s)`：`flag`，不再返回 0 | 无 | 同上 |
| Windows `WaitForSingleObject`、`GetExitCodeProcess`、`QueryInformationJobObject`、`IsProcessInJob` | 结果或错误 | <ul><li>`pidProc.ended`：`obsErr`；</li><li>`waitExit`、`endJob`、`waitHandle`、失败链的 Job 成员读取、`active` 的读取：`flag`；</li><li>`confirm`：`invalidf`（注入的查询故障除外：那是被测机制的 F1，属于预测）</li></ul> | 等待有期限；`ActiveProcesses` > 0 是“尚未” | 同上 |

区分两类：

- **被测机制预期的故障**，例如注入的 Job 查询失败、`pidfd_open` 不可用、绑定之后的身份不符：场景预测了机制对它的响应（F1、不发信号、不回退），证据完整时 confirmed。
- **实验工具没能取得必要的观察**：上表中的读取、等待、回收、枚举失败。这次运行 invalid，不能计为机制的预测成立，也不能计为机制被推翻。

B. 辅助进程的启动 → 见证写入 → 外层读取 → 身份验证 → 结束核验 → 外层判定

| 环节 | 责任方 | 失败时 | 允许的等待 | 最终由谁拒绝 |
|---|---|---|---|---|
| 内层测试进程的启动 | 外层（`launchInner`）：读内层的身份；Windows 上挂起创建、加入外层自己的 Job 并确认、再恢复 | 读不到身份或加入不了 Job：结束内层，这项检查失败 | 无 | 外层 |
| `root` 记录 | 内层测试进程，第一件事 | 身份或写入失败：带外通报，封条计数 | 无 | 外层：`witness-raised`、`witness-seal`、`witness-root` |
| 每一个进程的启动 | 创建者，经 `start`：`cmd.Start` 之后读子进程的身份（本进程未回收的子进程 / Go 持有句柄的进程），写 `created` 记录 | 读不到或写不成：结束并回收这个子进程，带外通报，返回错误；场景 `invalidf`，辅助进程以 90 退出 | 无 | 外层 `witness-raised`；内层 invalid |
| 辅助进程的回执 | 辅助进程，在做任何事之前：读自己的身份，等待自己的 `created` 记录，写 `launched` 记录 | 任何一步失败：带外通报，以 94 退出，不做任何工作、不启动子进程 | 等待创建记录最多 10 s，到期即失败 | 外层 `witness-raised` |
| 每一次写入 | 写者 | 打开、写入、短写、关闭失败都是失败，带外通报（Unix 是对外层的信号，Windows 是外层的命名事件），不只记在已经失败的文件里 | 无 | 外层 |
| 封条 | 内层测试进程，结束时打印在标准输出（`WP25WITNESS`） | 写成的记录数与遇到的失败数 | 无 | 外层 `witness-seal` |
| 外层的读取与解析 | 外层（`parseWitness`） | 读不到：`witness-unreadable`；任何格式问题（尾行截断、未知 kind、重复、PID 非规范、身份不是本平台的形式、字段不对）：`witness-malformed`，不跳过任何一行 | 无 | 外层 |
| 身份的核对 | 外层（`judgeWitness`） | <ul><li>`root` 不是外层启动的那个进程：`witness-root`；</li><li>`launched` 没有身份相同的 `created`：`witness-receipt`；</li><li>创建者没有自己的记录：`witness-creator`；</li><li>Windows 上 Job 计数的已创建进程数与 `created` 数不同：`witness-count`</li></ul> | 无 | 外层 |
| 结束的核验 | 外层，内层结束之后、任何补救之前：先普查，再按每条 `created` 记录的身份读取 | <ul><li>存活：`left`；</li><li>存活而没有创建记录：`unrecorded`；</li><li>状态读不到：`state-unknown`；</li><li>普查读不到，或记录存活而普查没有它：`census`</li></ul> | 不等待：不给遗留进程按自己的定时器结束的机会 | 外层 |
| 补救 | 外层：Windows 终止外层的 Job（成员就是本轮的进程）；Unix 只结束仍带本次检查的标记、身份仍成立的进程 | 补救之后仍存活：`still-alive`。补救永远不使这项检查通过 | 10 s | 外层 |
| 登记 | 外层：每个已创建的进程按 PID 与身份在内层的 `owned` 中，否则计为未登记（附结束状态） | 未登记数多于内层申报的缺额：`unaccounted` | 无 | 外层 |

**普查**（census）不依赖任何记录：

- Linux 与 macOS：外层读本用户、启动不早于内层的全部进程的环境（`/proc/<pid>/environ`，`sysctl kern.procargs2`），找出带本次检查唯一标记的存活进程。标记即 `WP25_WITNESS=<本项检查的文件>`，本轮的每个进程都从内层继承它。读不到即普查失败，不当作不存在。
- Windows：外层 Job 的活动进程列表，以及它的累计创建数 `TotalProcesses`；进程无法脱离这个 Job。

所以见证写入全部失败，也不能让存活的遗留进程看起来不存在；Windows 上连已经结束、没有任何记录的进程也能从计数中发现。

**新增回归**（回归先行 `8f42e58f43a31b433c540c8146fc3873dec3aa2c`，门禁的判定未改）：

- **B1-R1-S**：每个注入只作用一次，之后的调用照常；注入点打印 `WP25INJECT`。
  - `observe.sweep-recheck-once`：`TestExclusiveGate` 中，绑定之后的那次状态读取。
  - `observe.sweep-wait-once`：同一场景中，一次 `waitid` 失败而不回收，下一轮回收。
  - `observe.baseline-once`：`TestPipeNormalDrain` 起点的子进程列表。
  - `observe.prctl-fails`：`TestExclusiveBaselineLive` 的 subreaper 读取。
  - `observe.injob-once`（Windows）：`TestJobFailResume` 中，进程是否在 Job 中的读取。
  - 前四项为 Linux。
- **B1-R2-W**：在真实的辅助进程与真实的外层上注入见证本身的故障。外层必须以指定的理由拒绝这份证据，这项门禁自检才通过；内层的结论只记录、不作为条件。
  - `witness.write-fail`：辅助进程的写入真实失败，文件在写之前被关闭，即审查的反例一。
  - `witness.short-write`：只写出半条记录。
  - `witness.identity-unknown`：辅助进程读不到自己的身份，即审查的反例二。
  - `witness.truncated`、`witness.duplicate`、`witness.unknown-kind`、`witness.deleted`：内层结束时，真实地截断、重复、追加未知 kind 的记录，或删除见证文件。
  - `witness.leak-unreceipted`：内层故意留下一个不写自己记录的辅助进程（`TestInnerLeak`，只在门禁的内层运行，自身期限 8 s）。
  - `witness.leak-unrecorded`：故意留下一个绕过 `start`、没有任何记录的辅助进程。
  - `witness.unrecorded-ended`（Windows）：同样没有任何记录，但在外层查看之前已经结束。
  - `witness.early-exit`（正例）：辅助进程在留下自己的记录之前就结束了。外层必须看到它的创建记录、确认它已结束，并且接受这份证据。
  - 修复中新增 `witness.root-write-fail`：内层测试进程自己的创建记录写不成。它在旧代码中没有对应的写入点，回归先行中没有这一项。

**旧门禁上的结果**（`8f42e58`，旧的见证与判定）：

| 环境 | 失败 / 总数 | 主要失败 |
|---|---|---|
| 本机 Windows 11 | 10 / 31 | <ul><li>`observe.injob-once` 内层 confirmed、exit 0；</li><li>`witness.write-fail`（命中 6 次，见证为空）、`identity-unknown`（6 条 `unknown` 身份）、`truncated`、`duplicate`、`unknown-kind`、`leak-unreceipted`、`leak-unrecorded`、`unrecorded-ended`：外层没有任何问题，即证据被接受；</li><li>`early-exit`：外层看不到那个进程</li></ul> |
| 本机 WSL2 | 12 / 35 | <ul><li>四项 `observe.*` 内层 confirmed、exit 0；</li><li>见证类同上；</li><li>`leak-unrecorded`：遗留进程在 10 s 宽限内按自己的 8 s 定时器结束，外层只报告“未登记”，没有报告遗留</li></ul> |
| [run 39951830770](https://github.com/XKHoshizora/specmetry/actions/runs/39951830770) Ubuntu | 12 / 35 | 同本机 WSL |
| run 39951830770 macOS | 8 / 26 | 见证类 8 项 |
| run 39951830770 Windows | 10 / 31 | 同本机 Windows |

- run 39951830770：push，attempt 1，completed/failure（预期），每个 Job 检出 `8f42e58` 本身。
- 普通场景：Ubuntu 38、macOS 27 confirmed；Windows 28 confirmed、1 refuted。refuted 的是 `T1.pipe.escaped-holder`，阶段 D 计得 999.82 ms，小于 `T_drain` 1 s。这是一项与本轮无关的计时问题，见 §13；作者本机在同一提交上也出现一次（999.92 ms）。
- 在旧门禁上就通过的：`witness.short-write`（半条记录与下一条连成畸形行，旧解析恰好拒绝）、`witness.deleted`（打开失败）。

**修复**（`e77af946291862191759d3af6e94e372221748a2`）：

- **B1-R1-S**（对应表 A 的每一行）：
  - 清扫绑定之后读不到状态，记为观察失败、不发信号，不再是身份不符；
  - `poll`、`waitid` 的失败保存；`waitid` 只在 `si_pid` 指名这个进程时才算回收；
  - 发信号失败为机制停止；
  - 起点的子进程列表读不到即 invalid；
  - subreaper 与子进程计数的读取失败记入终判；
  - Windows 失败链的 Job 成员读取与句柄等待的错误记入终判；
  - Unix 清理中回收的失败与 `stop` 读不到状态保存到 `obsErr`；
  - macOS 早退出的 `waitid` 失败返回错误。
- **B1-R2-W**（对应表 B）：
  - 只有 `start` 可以启动进程（`TestStartPoints` 扫描源码，例外只有 `start` 本身、外层启动内层、`TestInnerLeak` 中故意的绕过）；
  - 创建者写 `created`，辅助进程等到它才写 `launched` 并做事；
  - 每次写入都核对；任何失败都带外通报，写者停止；
  - 内层有 `root` 记录与封条；
  - 外层严格解析并核对 `root`、封条、回执与创建者，Windows 上还核对 Job 计数；
  - 内层结束之后立即普查，再按身份逐条读取，最后才补救；
  - 登记按 PID 与身份匹配；`missing` 改为它实际的含义 `shortfall`（N5）。

**修复后的门禁回归**（[run 39954957954](https://github.com/XKHoshizora/specmetry/actions/runs/39954957954) 与作者本机，同一份原型代码）：

| 门禁回归 | 应得 | Ubuntu | macOS | Windows | 本机 Win11 | 本机 WSL |
|---|---|---|---|---|---|---|
| `observe.sweep-recheck-once` | 内层 invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `observe.sweep-wait-once` | 内层 invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `observe.baseline-once` | 内层 invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `observe.prctl-fails` | 内层 invalid | ✓ invalid/1 |  |  |  | ✓ invalid/1 |
| `observe.injob-once` | 内层 invalid |  |  | ✓ invalid/1 | ✓ invalid/1 |  |
| `witness.write-fail` | 外层 `witness-raised` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.short-write` | 外层 `witness-malformed` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.identity-unknown` | 外层 `witness-raised` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.truncated` | 外层 `witness-malformed` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.duplicate` | 外层 `witness-malformed` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.unknown-kind` | 外层 `witness-malformed` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.deleted` | 外层 `witness-unreadable` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.root-write-fail` | 外层 `witness-seal`；内层 invalid | ✓ | ✓ | ✓ | ✓ | ✓ |
| `witness.leak-unreceipted` | 外层 `left`；内层 invalid | ✓ 存活 1 | ✓ 存活 1 | ✓ 存活 1 | ✓ 存活 1 | ✓ 存活 1 |
| `witness.leak-unrecorded` | 外层 `left`；内层 invalid | ✓ 存活 1，无记录 | ✓ 存活 1，无记录 | ✓ 存活 1，无记录 | ✓ 存活 1，无记录 | ✓ 存活 1，无记录 |
| `witness.unrecorded-ended` | 外层 `witness-count`；内层 invalid |  |  | ✓ | ✓ |  |
| `witness.early-exit` | 外层接受；内层 invalid；看到无回执的已结束进程 | ✓ | ✓ | ✓ | ✓ | ✓ |

- Review 075、076 的全部门禁回归与正对照在修复后仍然通过，见 §2.8 的表。各平台合计：Ubuntu 36/36、macOS 27/27、Windows 32/32；本机 Windows 32/32、WSL 36/36。
- 每一项都另外核对了：
  - 注入确实命中：命中来自内层测试进程的输出与辅助进程自己的日志，不来自见证文件；
  - 后续调用照常：每项注入只作用一次，场景之后的读取与清理照常完成；
  - 正对照与合法的反例、注入、拒绝场景仍为 confirmed、exit 0，见证完整、没有遗留。

**拒绝之后的清理**（外层在补救之前记下的状态，run 39954957954 与作者本机）：

- 除了故意遗留的两项，所有门禁回归在内层结束时：
  - 普查没有存活进程，逐条记录也都已结束，没有状态读不到的对象，补救没有执行；
  - Linux 的内层自己也核对了没有留下子进程（存活或未回收的僵尸），否则内层的结论会带上这一理由。
- **已创建而未登记的进程**（外层列出，附结束状态）：
  - 报告损坏的 7 项各 1 个睡眠进程，已结束，缺额 1；
  - `load.chain-corrupt` 4 个链节点，已结束，缺额 4；
  - 未登记数都不多于缺额。这是数量上的容纳，不是逐个对象的对应：缺额也计入从未创建的预期进程，例如 `load.chain-short` 的缺额 1 是没有启动的那一层，`load.fanout-partial` 的缺额 20 是没有启动的 20 个后代。
- **故意遗留的两项**：
  - 补救之前，`left` 中各有 1 个存活进程。`leak-unrecorded` 的那一个没有任何记录（`unrecorded`），只有普查能看到它。
  - Linux 的内层自己的“留下的子进程”检查也报告了它。
  - 外层随后按本轮的标记与身份（Windows 按 Job）结束了它，补救之后没有存活进程。
  - 这两项因正确发现遗留而通过：补救成功没有使任何一项检查通过，也没有依赖辅助进程自己的 8 s 定时器补证。

**隔离变异**（作者本机，仓库外的原型副本，每次只撤回一项修复，只运行它针对的回归与健康正对照）：

| 变异 | 撤回的修复 | Windows | WSL |
|---|---|---|---|
| M13 | 绑定之后读不到状态，又记为身份不符 | — | `observe.sweep-recheck-once` 回到 confirmed、exit 0 |
| M14 | 清扫中 `waitid` 的失败不保存 | — | `observe.sweep-wait-once` 回到 confirmed、exit 0 |
| M15 | 起点的子进程列表读不到，又当作空列表 | — | `observe.baseline-once` 回到 confirmed、exit 0 |
| M16 | subreaper 读取失败，又返回 0 | — | `observe.prctl-fails` 回到 confirmed、exit 0 |
| M17 | Job 成员读取失败，又走“不在 Job 中”的分支 | `observe.injob-once` 回到 confirmed、exit 0 | — |
| M18 | 不带外通报 | `witness.write-fail`、`witness.identity-unknown` 的证据被接受（外层没有任何问题） | 同左 |
| M19 | 宽松解析（接受截断的尾行，跳过未知 kind，接受重复） | `witness.truncated`、`duplicate`、`unknown-kind` 的证据被接受 | 同左 |
| M20 | 不普查 | `witness.leak-unrecorded` 看不到遗留，只剩 `witness-count`；`leak-unreceipted` 仍被逐条记录发现（互为后备） | `witness.leak-unrecorded` 外层没有任何问题；`leak-unreceipted` 同左 |
| M21 | 不核对 Job 计数 | `witness.unrecorded-ended` 的证据被接受 | — |
| M22 | 不核对封条 | `witness.root-write-fail` 缺少 `witness-seal`（证据仍因带外通报与 Job 计数被拒绝） | 缺少 `witness-seal`（仍因带外通报被拒绝） |
| M23 | 身份读不到时写 `unknown` 并接受 | `witness.identity-unknown` 内层回到 confirmed、exit 0，外层只靠创建者的记录（`witness-receipt`）发现身份不符 | 同左 |
| M24 | 普查之前等待 9 s（旧的宽限） | `leak-unreceipted`、`leak-unrecorded` 的遗留按自己的定时器结束，外层不再报告遗留 | 同左 |

- 健康正对照在每一个变异中都通过：`control.silent-holder`，以及 Windows 的 `control.normal-drain` 或 Linux 的 `control.adoption-chain`。
- 变异只在作者本机运行，没有进入 CI；macOS 没有变异。

**直接检查**（不是门禁回归，也不是机制证据，三平台 CI 与本机都通过；WSL 上的 `TestStartPoints` 因测试二进制旁没有源码而 SKIP）：

- `TestWitnessJudge`：24 个构造的见证与封条，期望写死在表中，不由被检查的代码计算。包括：空文件（不论封条）、读取失败、截断、未知 kind、重复、`unknown` 身份、非规范 PID、旧的四字段记录、没有创建记录的回执、身份与创建记录不同、没有记录的创建者、别的进程的 `root`、封条报告失败、封条计数不符、没有封条、两个封条、带外通报、Job 计数不符，以及 3 个健康的正例。
- `TestRecordAlive`：自己的真实身份为存活，同一 PID 的另一身份为已结束，`unknown` 身份不被接受。
- `TestStartPoints`：源码中除列出的例外，没有绕过 `start` 的启动。

**计数分开**，三个层次互不替代，也不互相引用同一结果：

| 计数 | 来源 | 含义 |
|---|---|---|
| 门禁自检的通过 | `WP25GATE` | 这项自检得到了它要求的结果，可以是“正确地拒绝” |
| 内层机制的结论 | 内层的 `WP25RESULT` | 被注入故障的那次运行的结论 |
| 外层证据是否有效 | `WP25GATE` 中的 `problems` | 外层对它自己证据的判断 |

- 整次诊断的退出状态，只在三者都符合要求时为成功。见证故障类的自检通过，表示外层**正确地判定那份证据无效**，不是任何机制的 confirmed。
- 没有注入时，见证出现任何问题，都会使对应的门禁回归失败、整次诊断失败。
- 直接检查的期望不来自门禁回归，门禁回归的命中也不来自见证文件。

### 2.10 Review 078：外层自检的组合判定

**审查结论**（第一、三、四节）：

- **B1-R1-S 关闭**，限原型门禁与已追踪的调用链。
- **B1-R2-W 的原精确反例已修复**，但留下残余 **B1-R2-W-R**：负向自检只要找到预期的拒绝类别就判为通过，同时发生的外层自身失败被忽略。被忽略的包括补救失败、补救之后仍存活、普查或身份读取失败、必需的注入没有命中。
  - 审查方在 Windows 隔离副本中复现：只让外层补救在有存活对象时返回错误。两项故意遗留的自检在 `still-alive` 的同时报告 PASS，整个测试进程 exit 0。
- **其余意见**：
  - Review 077 N5（术语）关闭；Review 076 N4 保持非阻塞。
  - 新增 **N6**：Windows 阶段 D 的截止与计时口径不同，会导致随机拒绝，非阻塞，见 §13。
  - 新增 **N7**：本文残留旧运行来源的措辞，非阻塞，本轮已改：§3.4 区分“当时的证据”与“当前证据”，§4 ~ §8 的数值改为当前重跑的数值。

**根因**：

- 负向自检只回答了一个问题：“预期的拒绝理由出现了没有”。外层自己必须完成的步骤，没有作为通过的前提。所以同一份问题列表中，预期的 `left:` 与 `still-alive:` 可以同时存在，自检照样通过。
- 判定只有一个布尔值。补救是否需要、是否执行、是否成功，以及本轮进程最终的状态，都没有分开保存。

**判定表**（修复后的 `judge`；每一维单独保存在 `WP25GATE` 中）：

| 维度 | 取值 | 对 PASS 的作用 |
|---|---|---|
| 内层结论 | 符合 / 不符合用例的要求 | 不符合即 FAIL |
| 预期拒绝（`expected_found`） | 找到 / 没找到；用例没有指定类别时，要求证据没有任何问题 | 没找到即 FAIL |
| 证据的其他问题（`unexpected`） | 用例允许的连带类别之外的证据问题 | 有即 FAIL |
| 外层自己的失败（`tool_errors`） | `inner`、`census`、`state-unknown`、`hit`、`still-alive`、`insurance` | 有即 FAIL；任何预期拒绝都不能豁免 |
| 补救（`remedy`） | 不需要 / 成功 / 失败 / 没有执行 | 失败或没有执行即 FAIL |
| 本轮进程的最终状态（`final`，被测路径自己的事实） | 已确认结束 / 仍存活 / 未知 | 不是“已确认结束”即 FAIL；未知从不当作结束 |
| 保险清理（`insurance`，判定之后） | 什么也没剩 / 结束了 N 个 / 失败 | 永远不能使检查通过；被测路径说已结束、它却还找到进程或失败，即记为 `insurance` 而 FAIL |

- **PASS 的条件**：上表全部满足，并且用例要求时看到无回执而已结束的进程，才是 PASS。找到预期拒绝只证明第二行。
- **工具错误与资源事实分开保存**：最终确认对象已结束，不消除此前的工具失败；发现了预期错误，也不证明资源已经结束。
- **允许的连带类别**逐用例写明：
  - `witness.leak-unreceipted` 允许 `unaccounted`；
  - `witness.leak-unrecorded` 允许 `unrecorded`、`witness-count`；
  - `witness.root-write-fail` 允许 `witness-raised`、`witness-count`；
  - 其余用例都不允许。
- **允许的重试**：仍只有原有协议中的那些（§2.2 的“正常的尚未”）。本轮没有新增忽略错误的规则。

**组合回归**（`TestGateCombos`，回归先行 `4efc9fcfb3cda5e10a765c7c35c864c7f8d6673e` 引入，判定未改）：每一项都是真实的门禁回归——真实的内层启动辅助进程，外层做普查——只在外层被测路径的一步上注入一个故障：

| 注入 | 作用的位置 |
|---|---|
| `remedy-error` | 补救在有存活对象时返回这些对象与一个错误，什么也不做：审查的原注入 |
| `remedy-unconfirmed` | 真实的补救执行，它的结束确认被换成错误 |
| `census-error` | 真实的普查执行，它的结果被换成错误 |
| `record-unknown` | 一次真实的记录状态读取，它的回答被换成“读不到” |

这些注入都不改内层结论、不改见证，也不改最终的 `pass` 字段。

| 组合 | 门禁回归 | 外层注入 | 被测自检应得 | 旧判定 `4efc9fc` | 新判定 `31c0b29` |
|---|---|---|---|---|---|
| `combo.reject-remedy-ok` | `witness.leak-unreceipted` | — | PASS（补救成功，已结束） | PASS | PASS |
| `combo.reject-remedy-error` | `witness.leak-unreceipted` | `remedy-error` | FAIL（`still-alive`；补救失败；仍存活） | **PASS** | FAIL |
| `combo.reject-unrecorded-remedy-error` | `witness.leak-unrecorded` | `remedy-error` | FAIL（同上） | **PASS** | FAIL |
| `combo.reject-end-unconfirmed` | `witness.leak-unreceipted` | `remedy-unconfirmed` | FAIL（`still-alive`；补救失败；未知） | **PASS** | FAIL |
| `combo.reject-census-error` | `witness.deleted` | `census-error` | FAIL（`census`；未知） | **PASS** | FAIL |
| `combo.reject-state-unknown` | `witness.write-fail` | `record-unknown` | FAIL（`state-unknown`；未知） | **PASS** | FAIL |
| `combo.reject-with-unexpected` | `witness.leak-unrecorded`，不允许连带类别 | — | FAIL（其他问题 `unrecorded`） | **PASS** | FAIL |
| `combo.expected-missing` | `witness.truncated`，不注入 | — | FAIL（预期拒绝没找到） | FAIL | FAIL |
| `combo.healthy` | `control.silent-holder` | — | PASS（不需要补救，已结束） | PASS | PASS |
| `combo.healthy-state-unknown` | `control.silent-holder` | `record-unknown` | FAIL（`state-unknown`；未知） | FAIL | FAIL |

- **组合的判定**：被测自检的结论与每一维都符合要求，并且保险清理确认本轮没有留下进程，这一组合才通过。组合计在 `WP25COMBO` 行与 `combo_*` 计数中，与门禁回归分开。
- **两层分开**：被测自检“正确地拒绝”，是这一组合的通过；它本身从不计为一项门禁回归的通过。没有把“预期的补救失败”加进任何豁免列表。
- **旧判定上**：粗体的 6 项，被测自检都在外层失败的同时报告 PASS。

**旧判定上的结果**（`4efc9fc`）：

- **作者本机**：Windows 与 WSL 各 10 个组合，6 个失败（上表粗体）；门禁回归 32/32、36/36 仍全部通过。
- **三平台诊断 run 39964760402**（push，attempt 1，completed/failure，预期；每个 Job 检出 `4efc9fc`）：
  - Ubuntu、macOS、Windows 都是同样的 6 个组合失败；
  - 门禁回归 36/36、27/27、32/32 通过；
  - 场景：Ubuntu 38、macOS 27 confirmed；Windows 28 confirmed、1 refuted（`T1.pipe.escaped-holder`，N6：阶段 D 计得 999.9851 ms，没有重跑）。
- **审查原反例**（`combo.reject-remedy-error`，各平台相同）：
  - 内层 invalid、exit 1；注入命中；补救之前 `left` 中有 1 个存活进程；
  - 补救返回错误，`still_alive` 为同一个进程；
  - 被测自检 PASS；
  - 随后保险清理结束了它（`ended 1`）。

**首次失败（macOS 上的保险清理）**：

- **现象**：run 39964760402 的 macOS Job 中，`combo.reject-unrecorded-remedy-error` 的保险清理报告 FAILED。它重新普查时，读不到遗留进程 1549 的环境（`sysctl kern.procargs2` 返回 EINVAL），而约 0.2 s 之前被测路径的普查读到了它。
- **处置**：保险清理因此没有对它动作。它最迟在自己的 8 s 期限结束；这既不计为清理，在 runner 上也没有核验。
- **原因没有查明**：进程正处于 exec 或退出的过渡，只是一个猜测，没有证明。
- **修复**：保险清理改为先结束被测路径已经认出的对象（PID 与启动时间，动作之前再核对启动时间），不再重新读它们的环境；然后再结束新的普查找到的进程。
- **被测路径没有放宽**：活着的进程读不到环境，仍是普查错误。修复后的 run 39966556540 中，同一组合在 macOS 上为 `ended 1`。

**修复**（`31c0b290aed741f2506503aa0bb73790779a1a16`）：

- `judge` 按上表判定；`remedy`、`final`、`tool_errors`、`unexpected`、`expected_found` 分别保存。
- 补救的“没有执行、失败、成功、不需要”分开记录。
- 最终状态由被测路径自己的事实决定，按“仍存活 > 未知 > 已确认结束”取值。
- 保险清理在判定之后、独立于被测路径，只会使结论变差。
- `TestGateJudge` 以 13 行固定的期望，直接检查判定，包括真实路径不易到达的状态：补救没有执行、保险清理的矛盾、只有最终状态未知。

**修复后的结果**：

- **三平台诊断 run 39966556540**（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `31c0b29` 本身）：
  - 组合：三平台各 10/10，每一维都符合上表；
  - 门禁回归：Ubuntu 36/36、macOS 27/27、Windows 32/32；
  - 场景：38/38、27/27、29/29 confirmed，Ubuntu 带 `-race`、没有数据竞争的报告；
  - 直接检查：`TestGateJudge` 13 行、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过；SKIP 只有两个只供内层运行的测试。
- **作者本机**（`31c0b29` 的工作树）：结果见 §3.1。

**被测补救与保险清理分别记录**（run 39966556540，三平台相同）：

| 组合 | 被测路径：补救、最终状态 | 被测自检 | 保险清理（判定之后） |
|---|---|---|---|
| `reject-remedy-ok` | 成功；已结束 | PASS | 什么也没剩 |
| `reject-remedy-error` | 失败；仍存活（1 个） | FAIL | 结束了那 1 个 |
| `reject-unrecorded-remedy-error` | 失败；仍存活（1 个，无记录） | FAIL | 结束了那 1 个 |
| `reject-end-unconfirmed` | 失败（结束确认被换成错误）；未知 | FAIL | 什么也没剩（真实的补救已结束它） |
| `reject-census-error`、`reject-state-unknown`、`healthy-state-unknown` | 不需要；未知 | FAIL | 什么也没剩 |
| `reject-with-unexpected` | 成功；已结束 | FAIL（其他问题） | 什么也没剩 |
| `expected-missing`、`healthy` | 不需要；已结束 | FAIL、PASS | 什么也没剩 |

- 保险清理的结果没有使任何一项被测自检通过。
- 保险清理只操作本轮的对象：Windows 是外层 Job 的成员；Unix 是被测路径已认出、启动时间仍成立的进程，以及仍带本次检查标记的进程。它不使用无效见证中的 PID，也不依赖辅助进程自己的定时器补证。
  - **更正**（Review 079 B2，§2.11）：在 `31c0b29` 上，这句对 Unix 不成立。“被测路径已认出”的清单里，混有只凭见证记录认出、归属没有被证明的进程；保险清理只核对启动时间，就会向它发信号。修复见 §2.11。

**隔离变异**（作者本机，仓库外的副本，每次只撤回一项；运行全部组合、判定表与健康的门禁正对照）：

| 变异 | 撤回 | 组合 | 判定表 | 健康对照 |
|---|---|---|---|---|
| M25 | 恢复旧规则：找到预期理由即通过 | 6 个失败（上表粗体） | 8 行失败 | 通过 |
| M26 | 不看证据的其他问题 | `reject-with-unexpected` 失败 | 2 行失败 | 通过 |
| M27 | 不看外层自己的失败 | 全部通过（补救与最终状态兜住） | 2 行失败（注入没有命中、保险清理的矛盾） | 通过 |
| M28 | 不看补救与最终状态 | 全部通过（外层失败兜住） | 2 行失败（补救没有执行、只有最终状态未知） | 通过 |
| M29 | M27 与 M28 同时 | 6 个失败（外层失败的那 6 个） | 8 行失败 | 通过 |
| M30 | 未知的最终状态当作结束 | 4 个失败（维度不符） | 0 行失败 | 通过 |

- M27 与 M28 单独撤回时组合仍通过，这是两维互为后备，由判定表检出；同时撤回（M29）即重现审查的假通过。
- M25 第一次准备时变异不能编译（留下未使用的变量），这次记为无效，改正之后重做。
- 结果在 Windows 与 WSL 上相同。变异只在本机运行，macOS 没有变异。

**计数分开**：

| 计数 | 来源 | 含义 |
|---|---|---|
| 门禁回归的通过 | `WP25GATE` | 外层自检得到它要求的结果 |
| 组合的通过 | `WP25COMBO` | 被测自检在组合下给出要求的结论与维度；其中被测自检的拒绝不计为门禁回归 |
| 内层机制的结论 | `WP25RESULT` | 被注入的那次运行的结论 |
| 直接检查 | 普通 Go 测试 | 期望写死在表中 |

整次诊断的退出状态，只在四者都符合要求时为成功。

### 2.11 Review 079：保险清理的归属（B2）

**审查结论**（第一、三节）：

- **B1-R2-W-R 关闭**；B1-R1-S 保持关闭；原始 B1 的已知判定缺陷可按既有的原型门禁范围收口。这不等于 B2 也已通过。
- **BLOCKER B2**：Unix 的保险清理可能向归属没有被确认的见证 PID 发信号。
  - 调用方把只由见证记录提供、没有被本轮普查确认归属的身份放进 `known`；保险清理只凭 PID 与启动时间相符就发 SIGKILL。
  - 见证即使已被判为无效，也没有阻止这一动作。
  - 身份准确，不等于属于本轮检查。
  - 审查方在 Windows 上安全验证了共用的数据路径：一条 `created` 记录被换成外层自身的身份后，它进入了保险清理的入口，但不在内层的 Job 中，所以没有被动作。Unix 发信号的分支由源码证明，没有在 Unix 上实际执行。
- **其余**：
  - N7 关闭；N4、N6 保持未修复、非阻塞。
  - 最终常规 CI run 39968755313 为 7/8，合并条件没有满足。
  - 那次 SQLITE_BUSY 的特征与 M-15 相似；M-14 是版本探测路径的 `SQLITE_READONLY_RECOVERY`（264），不能因测试同名就把它并入 M-14。

**信任边界与根因**：

- PID 与启动时间只能证明“还是那个进程”，不能证明“这个进程属于本轮检查”。归属有两种可以核对的依据：
  - Unix：进程的环境中带本次检查唯一的标记（`WP25_WITNESS=<本项检查的文件>`），只有内层运行的进程从内层继承它；
  - Windows：进程是外层 Job 的成员，内层运行的进程都在其中、不能脱离。
- 见证记录是内层运行自己的声明，见证本身可能无效。
- 修复之前，`runGate` 把“记录说它存活、普查却没有找到”的进程也放进 `known`，交给补救与保险清理：
  - 被测的 Unix 补救在动作前还核对标记，所以不会动作；
  - 保险清理只核对启动时间，就会动作；
  - Windows 两个入口都只终止外层 Job，不会动作，但数据路径相同。

**候选来源、身份、归属与动作的对应**（修复后）：

| 候选来源 | 身份如何取得 | 归属凭什么成立 | 允许动作的入口 | 失败如何上报 |
|---|---|---|---|---|
| 被测路径的普查（Unix） | 普查时读到的 PID 与启动时间 | 普查时读到本次检查的标记，并且同一用户、启动不早于内层 | 补救：动作前再核对启动时间与标记。保险清理：动作前再核对启动时间 | 读不到即 `census` 工具错误 |
| 被测路径的普查（Windows） | 外层 Job 的成员列表，经句柄读创建时间 | 外层 Job 的成员 | 补救与保险清理都只终止外层 Job | 查询失败即 `census` |
| 只凭见证存活的 `created` 记录 | 记录中的 PID 与启动时间，`recordAlive` 核对仍是同一进程 | **记录本身不证明归属**；只有 `ownedNow` 在那一刻证明才成立（Unix：同一进程且带标记；Windows：同一进程且在外层 Job 中） | 证明成立，与普查所得同等对待；否则不交给任何按 PID 动作的入口 | 证明不成立或读不到，即 `ownership` 工具错误，最终状态为未知 |
| 外层直接创建的对象（内层测试进程） | 外层持有的子进程与句柄 | 外层自己创建 | 它由外层的 `Wait` 回收，不在清理之列 | 启动失败即 `inner` |
| 补救交给保险清理的清单 | 只来自上面两类已证明的 | 沿用当时的证明，动作前再核对启动时间 | 保险清理 | 失败或矛盾即 `insurance`，永远不使检查通过 |
| 保险清理自己的新普查（Unix） | 新普查读到的 PID 与启动时间 | 新普查读到标记 | 保险清理 | 读不到时保险清理报告失败 |
| 组合回归自己的夹具 | 组合持有的子进程，在它仍是子进程时读身份 | 不属于内层运行，由组合自己负责 | 只由组合自己结束（`Process.Kill` 与 `Wait`），不经外层检查的任何入口 | 结束不了即这一组合失败 |

- **外层的信号路径**：
  - Unix：补救与保险清理，都经同一个接缝 `outerSignal`；
  - Windows：补救与保险清理只终止外层 Job，不按 PID 发信号；
  - 带外通报（辅助进程向外层自身发 SIGUSR1，PID 来自外层自己设置的环境）不是清理路径。
- **保险清理的直接调用者只有 `runGate`**。修复后交给它的只有已证明归属的身份，不论被测路径判定成功与否。

**回归先行**（`f1b8566bde691d738b804c4defd129605367c14e`，谁可以被动作没有改）：

- **信号接缝**：外层的信号都经 `outerSignal`，组合回归可以让它对夹具只记录、不发送。
- **外层注入 `forge-owner`**：在外层读取见证之前，追加一条格式合法、以内层名义写的 `created` 记录，指向组合自己启动、持有的夹具进程；夹具不属于内层运行，也不在外层 Job 中。
- **三个组合**：
  - `combo.forge-owner`：健康运行；
  - `combo.forge-owner-mixed`：同时有本轮自己遗留的进程；
  - `combo.forge-stale`：夹具的 PID 配另一个启动时间，即身份已经变化。
- **每个组合要求**：夹具不被交给任何按 PID 动作的入口，没有发向它的信号，之后仍存活；随后由组合自己结束。门禁结果记下交给这些入口的身份（`acted`）。
- **旧代码上的结果**：

  | 环境 | `forge-owner` / `forge-owner-mixed` | `forge-stale` |
  |---|---|---|
  | 作者本机 Windows 11 | 夹具被交给动作入口；没有信号（入口只终止 Job） | 通过 |
  | 作者本机 WSL2 | 夹具被交给动作入口；保险清理向它发出 477 / 471 次信号意图，全部只记录、没有发送 | 通过 |
  | [run 39976843072](https://github.com/XKHoshizora/specmetry/actions/runs/39976843072) Ubuntu | 同上，401 / 400 次 | 通过 |
  | run 39976843072 macOS | 同上，423 / 435 次 | 通过 |
  | run 39976843072 Windows | 夹具被交给动作入口；没有信号 | 通过 |

  - run 39976843072：push，attempt 1，completed/failure，预期；每个 Job 检出 `f1b8566`。
  - 门禁回归 36/36、27/27、32/32 通过；场景 38、27、29 confirmed；组合各 11/13。
  - 夹具在所有平台都仍存活，然后由组合自己结束。没有向任何未知进程或夹具实际发送过信号。

**修复**（`8a2b97ccd521c6af587baf4e3f45421d68dafcd9`）：

- `sortCandidates`：普查找到的直接可动作；只凭记录存活的，交给 `ownedNow` 此刻证明，证明不成立或读不到即拒绝。
- 被拒绝的：不进入 `left`、不交给补救或保险清理；记为 `ownership` 工具错误，任何预期拒绝都不能豁免；最终状态为未知。
- 保险清理只复用已证明的身份，动作前再核对启动时间。补救在 Unix 上仍在动作前核对标记。
- `TestCandidates` 以 5 行固定的期望直接检查分拣，包括“归属读不到”这一真实路径难以构造的情形。

**修复后的结果**：

- **三平台诊断 [run 39977737240](https://github.com/XKHoshizora/specmetry/actions/runs/39977737240)**（push，attempt 1，completed/success，没有重跑；每个 Job 检出 `8a2b97c` 本身）：
  - 组合：三平台各 13/13；
  - 门禁回归：Ubuntu 36/36、macOS 27/27、Windows 32/32；
  - 场景：38/38、27/27、29/29 confirmed，Ubuntu 带 `-race`；
  - 直接检查：`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过；
  - 两项 `forge-owner` 组合中，夹具在三平台都被拒绝（`refused`），没有被交给动作入口，信号意图为 0，之后仍存活，再由组合自己结束；
  - `forge-stale` 中，夹具不是候选（记录所指的进程已不存在），没有动作。
- **作者本机**（`8a2b97c` 的工作树）：结果见 §3.1。

**被测补救与保险清理分别记录**（修复后，三平台相同）：

| 组合 | 被测路径 | 被测自检 | 保险清理 | 夹具 |
|---|---|---|---|---|
| `forge-owner` | 夹具被拒绝（`ownership`）；不需要补救；最终状态未知 | FAIL | 什么也没剩 | 存活，没有被动作；随后由组合结束 |
| `forge-owner-mixed` | 本轮遗留的进程被补救（成功）；夹具被拒绝；最终状态未知 | FAIL | 什么也没剩 | 同上 |
| `forge-stale` | 记录所指的进程已不存在，不是候选；不需要补救；已结束 | FAIL（见证的其他问题） | 什么也没剩 | 同上 |
| `reject-remedy-error` 等（Review 078 的组合） | 与 §2.10 相同 | 与 §2.10 相同 | 只结束本轮已证明的进程 | — |

**隔离变异**（作者本机，仓库外的副本，每次只撤回一项；运行全部组合、分拣表、判定表与健康的门禁正对照）：

| 变异 | 撤回 | Windows | WSL |
|---|---|---|---|
| M31 | 只凭记录存活的进程不经证明就动作 | 两项 `forge-owner` 组合失败（夹具被交给动作入口）；分拣表 3 行失败 | 同左，另有 469 / 476 次信号意图 |
| M32 | 归属只凭身份（不看标记或 Job） | 两项组合失败；分拣表通过（它用固定的验证函数），只有真实路径检出 | 同左，481 / 475 次信号意图 |
| M33 | 保险清理也拿到只凭记录的进程 | 没有检出：Windows 的保险清理只终止 Job，不用这份清单 | 两项组合失败，480 / 479 次信号意图 |
| M34 | 拒绝之后最终状态不再记为未知 | 两项组合的维度不符 | 同左 |

- 健康正对照每一个变异都通过。
- 所有信号意图都只被记录，没有发送；变异中夹具始终存活，由组合自己结束。
- 变异只在作者本机运行，macOS 没有变异。

**仍未验证的边界**：

- **身份核对与动作之间的时间窗口**：在启动时间核对与信号之间，PID 仍可能被复用；本文没有构造这种情形，也不宣称已经解决。
- **归属的前提**：
  - Unix：受控的辅助进程不清除、不伪造自己的环境，标记只从内层继承；一个故意复制标记的进程会被当作本轮的。
  - Windows：Job 不允许脱离。
  - 这些是原型受控环境下的前提，不是对任意宿主的保证。
- **只由直接检查覆盖的**：“归属读不到”只有分拣表覆盖，真实路径中没有构造读不到环境、却仍存活的同用户进程。Review 080 的审查方另在 Windows 的真实外层调用链上，于真实的 Job 查询之后注入一次归属读取错误，错误不可豁免地传播（Review 080 §5.1）；这不是原生查询失败或 Unix 环境读取失败的动态复现。
- **夹具的信号**：Unix 上发向夹具的信号只由接缝记录，没有实际发送；修复后的路径中也没有发送。

### 2.12 Review 076 N4 与 Review 078 N6（PR #33 合并之后的独立小 PR）

**范围**（[review-resolution.md §95、§96](review-resolution.md)）：PR #33 合并核验与 master push CI 全部成功之后，从已核验的 master `5e9a42c` 另开分支 `test/wp2.5-prototype-n4-n6`，只修原型的 N4 与 N6。仍是 test-only：生产代码、根模块的依赖、Schema、迁移与 `ci.yml` 都没有改动。

**N4：报告工具失败的分类**

- **问题**：持有者（`holder`）在写入失败之后写一条 `epipe` 记录。记录写不出时，它忽略这个错误，照常以 3 退出。F1 场景（`T1.pipe.F1-skip-drain`）等不到记录，就把“持有者的后续写入失败”记为没有发生，结论为 refuted。实际上是观察工具没能报告，应为 invalid。
- **修复后的判定**：没有记录时，先读持有者的状态，再读一次报告（结束之前刚写入的记录仍然算数）。

  | `epipe` 记录 | 持有者 | 结论 |
  |---|---|---|
  | 有 | — | 这一项预测成立（其余预测也成立时为 confirmed） |
  | 没有 | 仍在运行 | 写入没有失败：预测不成立，refuted |
  | 没有 | 已结束 | 它没能报告看到的事件：观察失败，invalid |
  | — | 状态读不到 | 观察失败，invalid（既有规则，在清理时记入） |

- **持有者自己的退出状态**：记录写不出时，以 `exitReport`（91）退出，与其他辅助进程的约定一致。Unix 上持有者不是测试进程的子进程，外层读不到它的退出码；场景依据的是“已结束而没有记录”，不是退出码。
- **门禁回归**（真实的辅助进程与场景路径，不是单独的分类函数）：
  - `report.epipe-unwritable`：持有者的报告文件以只读方式打开，真实的写入失败；要求 invalid，并且注入命中。
  - `control.f1-read-kept`：场景对两个读端另持一个句柄（Unix 为 `F_DUPFD_CLOEXEC`，Windows 为不可继承的 `DuplicateHandle`），停止读取不会关闭管道，持有者的写入不失败。工具正常、预测不成立，要求 refuted。
  - `control.f1-skip-drain`：健康的运行，要求 confirmed。
- **错误的传播**：invalid 与 refuted 都使内层测试失败、退出码非零；门禁回归同时核对结论与退出码。
- **更正**（Review 081 N4-R，§2.13）：表中“没有记录、仍在运行 → refuted”一行不成立：报告写失败之后、退出之前的持有者也仍在运行。这一判定已由 §2.13 的表取代。

**N6：阶段 D 的时钟口径**

- **问题**：阶段 D 的截止由 Go 的时钟（`time.Now` 的单调读数与运行时的定时器）给出，时长却由 `stopwatch` 计量，在 Windows 上是性能计数器。Go 的时钟在 Windows 上随时钟中断前进，所以一个按 Go 的时钟准时结束的阶段 D，在性能计数器上可以短最多一个中断周期。严格的“≥ 1 s”比较于是随机拒绝（999.5519 ms 等，§13）。
- **修复**：
  - 阶段 D 与阶段 J 都在同一个 `phaseClock` 上限定与计量。场景中它是 Go 的时钟，也就是 `T_drain`、`T_join` 所定义的时钟。
  - 等待只在这个时钟读到截止时刻之后，才以截止结束；唤醒只是再读一次时钟的提示，早到的唤醒不结束阶段。
  - 时长在同一个时钟上读取。`stopwatch` 的读数另行记录（`drainFine`、`joinFine`），用于与此前各轮的读数对照，不作判定。
  - `T_kill`、`T_drain`、`T_join` 与各项超时都没有改动，也没有引入容差：判定所用的时钟与截止所用的时钟相同，不需要换算。
- **确定性的边界回归** `TestPhaseClock`（直接检查）：`runS6On` 在测试自己推进的时钟上运行，复制器读的是内存管道，写端由测试持有，所以复制器不会自行结束。
  - **恰好到截止**：时钟恰好推进 `T_drain`，阶段 D 结束，时长正好是 `T_drain`，计为用满。
  - **差 1 ns**：阶段 D 不结束；再推进 1 ns 才结束。
  - **早到的唤醒**：唤醒比截止早 450 µs（与 Windows 上观察到的短缺相近），阶段 D 不结束，按剩余的 450 µs 再等一次。
  - 三行都要求读取被停止、停止时复制器仍阻塞、结果为 O2。
- **实际管道的对照**仍是 `T1.pipe.escaped-holder`：保留严格的“阶段 D 用满 `T_drain`”，现在在阶段自己的时钟上比较；观察中同时列出 stopwatch 的读数。
- **不是硬实时保证**：“用满 `T_drain`”只表示阶段 D 在它的时钟读到截止之前没有结束。时钟的推进与调度的延迟不在本原型的证明范围内。

**回归先行**（`e7dbcdc` 只改 workflow；回归在 `91229b5beb7b6b06fe3a2d97b3e99f25d1a9e882`，判定没有改）：

- 旧代码上，三平台诊断 [run 39991866363](https://github.com/XKHoshizora/specmetry/actions/runs/39991866363)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `91229b5`。
  - `report.epipe-unwritable` 在三平台都得到 refuted，失败；两项对照通过。门禁回归 Ubuntu 38/39、macOS 29/30、Windows 34/35。
  - `TestPhaseClock` 的三行在三平台都失败：时长读的是 stopwatch（远小于 `T_drain`）；早到的唤醒结束了阶段 D。
  - 场景 38、27、29 confirmed；组合 13/13；SKIP 只有两个只供内层运行的测试。
- 作者本机 Windows 11 与 WSL2 的结果相同：门禁回归 34/35、38/39；场景 29 confirmed，37 confirmed + 1 unsupported（没有 pwsh）；组合 13/13；`TestPhaseClock` 三行失败。
- 那时的 `TestPhaseClock` 用的是操作系统的管道（见下文“直接检查自己的缺陷”）；三行的失败都是断言本身（时长与早到的唤醒），不是超时。

**修复**：

- N4：`3fe5701f0c602ffdcf42200ce93b793341603da3`；
- N6：`5054e54a14d3e9084aa675880175232e61361c17`；
- 直接检查改用内存管道：`ac3b3bed4489404124d51181c1e8ed902d55c82e`。**修复代码固定为 `ac3b3be`。**

**直接检查自己的缺陷**（如实保留）：

- `5054e54` 的三平台诊断 [run 39992975299](https://github.com/XKHoshizora/specmetry/actions/runs/39992975299)（push，attempt 1，completed/success；每个 Job 检出 `5054e54`）：场景 38/38、27/27、29/29 confirmed，门禁回归 39/39、30/30、35/35，组合 13/13，直接检查都通过。作者本机对同一棵树的完整运行（Windows、WSL）也都通过。
- 但在随后的隔离变异中，M35 只改 `pipe_test.go`，本机 Windows 上 `TestPhaseClock/deadline-exact` 却以“runS6On did not return within 5 s”失败。对 `5054e54` 的二进制重复 60 次，180 个子检查中有 12 次同样失败。WSL 的各次运行中没有出现，但没有在 WSL 上重复测量。
- **原因在检查本身**：假时钟下阶段 D 不占用真实时间。Windows 上关闭操作系统管道的读端，只取消已经发出的读取（`CancelIoEx`），然后等待进行中的读取结束（Go 的 `internal/poll` 源码如此）。复制器的读取若在关闭之后才发出，就会一直阻塞，关闭也随之等待，直到测试关闭写端。
  - **更正**（Review 081 B1，§2.13）：这不只是检查自身的缺陷。真实的 `runS6` 同样在调用者中同步关闭读端，在 Windows 上也可能在阶段 J 之前阻塞；改用内存管道只隔离了时钟检查，没有修复真实路径。
- **修正**（`ac3b3be`）：`TestPhaseClock` 改用 `io.Pipe`，关闭读端会立即结束挂起的或之后的读取。重复 200 次，600 个子检查 0 次失败。检查的内容没有改变；真实的管道仍由各管道场景使用。
- 这次失败没有被重跑覆盖：run 39992975299 是 `5054e54` 的诊断，保留为历史观测；当前证据是 `ac3b3be` 的诊断。

**修复后的证据**（`ac3b3be`）：

- **三平台诊断** [run 39994591073](https://github.com/XKHoshizora/specmetry/actions/runs/39994591073)：push，attempt 1，completed/success，没有重跑；每个 Job 检出 `ac3b3be` 本身。
  - 场景：Ubuntu（带 `-race`，没有数据竞争的报告）38/38、macOS 27/27、Windows 29/29 confirmed；场景 ID 的并集仍为 53。
  - 门禁回归：39/39、30/30、35/35（各新增上述 3 项）。三项 F1 门禁回归分别得到 confirmed、refuted、invalid。
  - 组合：三平台各 13/13。两项 `forge-owner` 中，夹具被拒绝、没有交给动作入口、没有信号意图，之后仍存活，再由组合结束；`forge-owner-mixed` 中本轮自己遗留的进程照常由被测补救结束。
  - 直接检查：`TestPhaseClock`、`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过。
  - SKIP 只有两个只供内层运行的测试。
  - `T1.pipe.escaped-holder` 的阶段 D（自身时钟 / stopwatch）：Ubuntu 1.000342361 s / 1.000342312 s，macOS 1.000062958 s / 1.000062959 s，Windows 1.0002407 s / 1.0000222 s。
- **作者本机**，对 `ac3b3be` 的树：
  - Windows 11：场景 29/29 confirmed；门禁回归 35/35；组合 13/13；直接检查都通过。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 39/39；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码），其余直接检查通过。
- **N6 的现象再次出现过**：作者本机对 `5054e54` 的完整 Windows 运行中，`T1.pipe.escaped-holder` 的阶段 D 在自身时钟上为 1.0001495 s，stopwatch 读数为 **999.6671 ms**。旧口径下这一次会被判为 refuted；新口径下是 confirmed。
- **隔离变异**（作者本机 Windows 与 WSL，对 `ac3b3be`；每次只撤回一处，运行 `TestPhaseClock`、三项 F1 门禁回归、健康对照 `control.silent-holder` 与两个真实管道场景）：

  | 变异 | 撤回的内容 | Windows | WSL |
  |---|---|---|---|
  | M35 | 没有记录时不看持有者的状态，一律 refuted | `report.epipe-unwritable` 失败（得到 refuted） | 同左 |
  | M36 | 没有记录时一律 invalid，不看持有者是否仍在运行 | `control.f1-read-kept` 失败（得到 invalid） | 同左 |
  | M37 | 唤醒即结束等待，不再读时钟 | `TestPhaseClock/early-wake` 失败 | 同左 |
  | M38 | 阶段 D 的时长改回在 stopwatch 上读取 | `TestPhaseClock` 三行都失败 | 同左 |

  - 每个变异只由对应的检查检出；其余检查与健康对照都通过。
  - M37、M38 下真实管道场景 `T1.pipe.escaped-holder` 仍通过：Go 的定时器不会早于它的时钟触发，stopwatch 的短缺又是随机的；所以只有确定性的直接检查能稳定检出它们。
  - 对 `5054e54` 的第一次变异运行中，M35 在 Windows 上另有上述 `TestPhaseClock` 超时；它是直接检查自己的缺陷，不是变异的效果，原始记录保留。

**诊断 workflow 的变化**（单独列出）：`e7dbcdcf5690e6cbbca50515f37cefe1c49c2845` 只在 push 的分支列表中加了 `test/wp2.5-prototype-n4-n6` 一项精确匹配，并相应改了一句说明。路径过滤、权限、并发策略、平台、`-race`、期限与失败传播都不变；`ci.yml` 没有改动。

**保留的历史**：

- 此前的 refuted 与旧读数都保留：run 39951830770（999.8195 ms）、run 39964760402（999.9851 ms）、作者本机的 999.92 ms 与 999.5519 ms。那些运行的阶段 D 是在 stopwatch 上计量的，不改写为通过。
- 此前各轮 `T1.pipe.F1-skip-drain` 的 confirmed 不受 N4 影响：它们都有 `epipe` 记录，N4 只改变“没有记录”时的分类。

**新的观察：Windows 上停止读取的时序前提**（不是本轮的修复对象，登记待审查）：

- 关闭读端只取消已经发出的读取；在复制器的读取到达系统之前关闭，关闭会等这次读取返回。
- 原型的管道场景中，停止读取之前都有足够的等待（阶段 D、F1 场景的等待、`T1.pipe.stop-read-latency` 的 5 ms），所以复制器的读取先到达；这一点没有被单独证明。按同一机制推断，`T1.pipe.stop-read-latency` 若遇到这种时序，`Close` 会阻塞到测试的期限；没有观察到。
- 生产设计中，“停止读取会使复制器返回”在 Windows 上依赖这一时序；是否需要另外的保证（例如停止之前确认读取已经发出，或使用可取消的读取），属于冻结之前要明确的技术前提，本轮没有决定。
- **更正**（Review 081 B1）：以“停止之前有足够的等待”推定读取已经发出，依据不足，每一次复制循环都可能重新进入这个窗口。这一缺口已在 §2.13 修复，结论随之收窄。

**仍未验证的边界**：

- **持有者的结束**：Unix 上外层读不到持有者的退出码；“已结束而没有记录”是推断工具失败的依据。持有者若在检查之前因为别的原因结束，同样记为 invalid，这是保守的方向；本场景中没有这样的路径，硬时限另有 `limit` 记录。
- **refuted 的对照**：`control.f1-read-kept` 用人为保留的读端构造“预测不成立”，只覆盖一种成因。
- **Windows 上 Go 时钟的中断周期**会在运行中变化；本修复不依赖它的大小。stopwatch 与阶段时钟之间的差只作为观察，没有定量的上界检验。
- **假时钟**：`TestPhaseClock` 证明的是 `runS6On` 的规则；真实时钟上的行为由 `T1.pipe.escaped-holder` 与其他管道场景检验。

**状态**：N4、N6 已修复，待独立审查；不宣布关闭。（其后：Review 081 关闭 N6，认定 N4 的残余与 B1，见 §2.13。）

### 2.13 Review 081：停止读取的生命周期（B1）与 N4 的残余（N4-R）

**审查结论**（[Review 081](reviews/codex-review-081.md) 第一、三 ~ 五节，对 `4e009a6`）：**技术 NO-GO**。

- **B1（BLOCKER）**：`runS6On` 在调用者中逐个同步关闭读端，全部关闭返回之后才开始阶段 J，阶段 J 之后才封闭。
  - Close 会等待持有该文件的读取。Windows 上，Close 只取消已经发出的读取；之后才进入系统调用的读取会一直阻塞，直到有数据或 EOF。
  - 于是阶段 J、封闭与 O3 都可能被一个没有自身出口的 I/O 等待挡住。审查方用受控的 FD 调度，在真实的 `runS6` 与真实的 `os.File.Close` 上复现：默认 `T_join` 下，2 s 之后仍没有开始阶段 J。
  - §2.12 把 `5054e54` 的超时主要归为“检查本身”的缺陷，又以“停止之前有足够的等待”推定读取已经发出；这两点依据不足（§2.12 的更正）。
- **N4-R（NON-BLOCKER）**：“没有记录、持有者仍在运行，所以写入没有失败”不成立。报告写失败之后、退出之前的持有者也仍在运行；审查方让它延后退出，就得到 refuted。
- **N6 可以关闭**（原时钟口径的范围内）：截止、剩余预算与阶段时长已统一在 `phaseClock` 上；这不包括 Close 的生存期，也不包括真实 I/O 的有界保证。
- **其余**：PR #33 的 B1、B2 的关闭范围保持；正式 CI 条件已满足；没有新增产品决定或接受风险。

**三个时序**（Review 081 §3.2）：

1. 读取还没有取得文件的读引用：先关闭，之后的读取直接得到“已关闭”。
2. 读取已经取得读引用、还没有进入系统调用：Close 的取消找不到要取消的请求，稍后才发出的读取（Windows 的同步 `ReadFile`）会等数据，Close 等它释放引用。
3. 系统读取已经发出：取消针对它；取消请求本身不是完成的证明。

**回归先行**（`bde00fc30cae12c0827a6d3125d04273d10af164`，判定没有改）：

- **`TestStopRead`**（直接检查）：真实的 `os.Pipe`、真实的 `File.Close`；`runS6On` 在测试推进的假时钟上运行，跳过阶段 D（F1 的路径）；写端由测试持有，从不写入。读取经文件的 RawConn 进行：先取得与 `File.Read` 相同的读引用，在测试控制的闸门前等待，再做真实的系统读取。
  - `read-in-system-call`（对照）：读取已在系统调用中。要求停止完成、结果 O2，时钟不推进。
  - `window-released-in-J`：停止时读取持有引用。要求阶段 J 照常开始（假时钟上请求了 `T_join` 的唤醒），并确认有 `(*FD).Close` 在等这个引用；随后开闸，读取在阶段 J 内做系统读取，要求停止自行完成：O2、停止完成、没有遗留，时钟不推进。
  - `window-held-past-J`：阶段 J 结束时读取仍在窗口中。要求照常封闭、O3、停止挂起，并把停止与复制器交给调用方；两种后续调用策略都看到挂起。开闸之后，不写任何字节，交出的两项须自行结束。
- **`report.epipe-unwritable-alive`**（门禁回归）：持有者的 `epipe` 记录真实写入失败，之后它再存活 5 s 才退出；要求 invalid。
- **旧代码上**：
  - 三平台诊断 [run 40066182430](https://github.com/XKHoshizora/specmetry/actions/runs/40066182430)（push，attempt 1，completed/failure，预期；每个 Job 检出 `bde00fc`）：`TestStopRead` 的两个窗口行在三平台都失败（停止挡住了阶段 J），对照通过；`report.epipe-unwritable-alive` 在三平台都得到 refuted。门禁回归 Ubuntu 39/40、macOS 30/31、Windows 35/36；场景 38、27、29 confirmed；组合 13/13。
  - 作者本机 Windows 结果相同。
  - **检查自己的错误**（提交之前发现，如实记录）：对照行的第一版要求“阶段 J 请求了唤醒”，在旧代码上也失败。原因是复制器在阶段 J 等它之前就已返回时，不会请求唤醒。改为“时钟不推进即返回”之后才提交。

**修复 B1**（`1a6fec5a988f29332c871ef04ff7ac92d5aedc17`）：停止读取纳入阶段 J。

| 问题 | 修复之后 |
|---|---|
| 停止请求 | 阶段 J 一开始，就为每个被停止的流启动一个停止任务（关闭它的读端）；运行等待复制器与停止任务，共用 `T_join` 的截止 |
| 停止是否完成 | `s6Result.stop`：done（关闭已返回，错误另记）或 pending |
| 复制任务是否结束 | 结束为 O1 / O2，否则为 O3（`s6Result.out`） |
| 入口是否封闭 | 阶段 J 结束即封闭，不论什么仍在运行；之后这次运行不再有字节到达目的端 |
| 资源由谁持有 | 未完成的停止（持有读端）与未返回的复制任务（持有读端与已封闭的入口）交给调用方（`s6Result.left`）。场景以 `adopt` 接手，在自己的进程结束之后等待它们，期限内没有结束即 invalid（cleanup not verified）；停止出错是观察失败（**更正**，Review 082 B1-Ra：当时只对运行返回之前的错误成立，§2.14） |
| 之后的调用是否允许 | O3 登记（`lingering`）同时持有挂起的停止与复制任务；拒绝还是允许由策略决定（C-4，属于 D20 的待决范围，**未批准**） |

- **停止任务的身份与出口**：每个被停止的流一个，由启动它的运行持有；关闭返回即结束，并保留关闭的错误；运行返回时没有结束的，交给调用方，不被丢弃。
- **Unix**：读端在 Go 的 poller 中。Close 设置“正在关闭”，并唤醒等在 poller 上的读取；还没到 poller 的读取会在那里看到这个标记（`internal/poll` 的 `prepareRead`、`waitRead`）。停止只需等读取离开它的系统调用。
- **Windows**：`os.Pipe` 的句柄是同步的。Close 等待期间，停止任务每毫秒再调用一次 `CancelIoEx`，直到 Close 返回；读取一旦进入系统调用就被取消。
  - 取消经停止任务自己持有的、指向同一管道的句柄（`DuplicateHandle`，不可继承）。这个句柄在 Close 之前取得，所以从不指向 Close 可能已经释放的句柄值；Close 返回之后关闭。
  - 作者在仓库外的探针中核实：经复制的句柄取消，会取消经原句柄发出的同步 `ReadFile`；不取消时，它一直阻塞到有人写入。
- **两个平台**：读取若始终不进入系统调用（持有引用、在别处等待），停止就一直挂起，由调用方持有。
- `T_kill`、`T_drain`、`T_join` 与各项超时都没有改动；没有容差。

**修复 N4-R**（`079f9c72d57a0b182d970b0f8455a80d580e04c1`）：

| 情形 | 依据 | 结论 |
|---|---|---|
| 预期的现象发生 | `epipe` 记录到达 | 这一项预测成立（其余也成立时为 confirmed） |
| 预期的现象没有发生，工具正常 | 没有记录，**并且**在场景另持的读端上，停止之后有迟到的字节在等待（不读取地计数：FIONREAD、PeekNamedPipe），即持有者的后续写入确实成功 | refuted |
| 报告尚未到达 | 没有记录，也没有上述证据 | invalid（观察缺失） |
| 报告失败 | 同上；持有者已结束或仍在运行都一样 | invalid |
| 状态不可读 | 计数读不到，或持有者的状态读不到 | invalid |

- “持有者仍在运行”不再作为任何结论的依据，只写进理由。
- 工具失败不会降为 refuted，也不会被之后的成功观察覆盖：判为 invalid 时，场景即停止。
- **更正**（Review 082 N4-R，§2.14）：上表的“报告失败 → invalid”与上一句当时超出了实现：一路的迟到字节与另一路的写入失败、报告失败可以共存，结论仍为 refuted。已由 §2.14 的判定取代。
- **对照**：`control.f1-skip-drain` 为 confirmed；`control.f1-read-kept` 为 refuted，依据是它的迟到字节；`report.epipe-unwritable` 与 `report.epipe-unwritable-alive` 为 invalid。

**补充的回归**（`125c4f70114c312828f7f8fc74734761e14c343a`，在修复之后补上）：

- `cleanup.leftover-running`：`T1.O3.gate-lingering` 从不放开它的闸门，O3 复制任务在场景结束时仍在运行；要求 invalid。
- 它是在定向撤回时发现“场景等待交出的遗留”没有检查覆盖之后补的。在 `bde00fc` 的树上加入同样的注入与用例，场景为 refuted（门禁回归失败）；在当前树上为 invalid。
- **修复代码固定为 `125c4f7`。**

**修复后的证据**（`125c4f7`）：

- **三平台诊断** [run 40066910980](https://github.com/XKHoshizora/specmetry/actions/runs/40066910980)：push，attempt 1，completed/success，没有重跑；每个 Job 检出 `125c4f7` 本身。
  - 场景：Ubuntu（带 `-race`，没有数据竞争的报告）38/38、macOS 27/27、Windows 29/29 confirmed；并集 53。
  - 门禁回归：41/41、32/32、37/37（相对上一轮各新增 2 项）。F1 的四项分别为 confirmed、refuted、invalid、invalid；`cleanup.leftover-running` 为 invalid。
  - 组合：三平台各 13/13。两项 `forge-owner` 中夹具零动作、零信号，之后仍存活，再由组合结束；`forge-owner-mixed` 中本轮自己遗留的进程照常由被测补救结束；补救失败、结束未知的组合仍使被测自检不通过。
  - 直接检查：`TestStopRead`、`TestPhaseClock`、`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过。
  - SKIP 只有两个只供内层运行的测试。
  - `T1.pipe.escaped-holder` 的阶段 D（自身时钟 / stopwatch）：Ubuntu 1.000377313 s / 1.000377383 s，macOS 1.000251333 s / 1.000251375 s，Windows 1.0002235 s / 1.0002329 s。
- **中间提交的诊断**（历史观测）：[run 40066611501](https://github.com/XKHoshizora/specmetry/actions/runs/40066611501)（`079f9c7`，没有补充的回归）：completed/success；场景 38/38、27/27、29/29；门禁回归 40/40、31/31、36/36；组合 13/13。
- **作者本机**：对 `125c4f7` 的树：
  - Windows 11：场景 29/29 confirmed；门禁回归 37/37；组合 13/13；直接检查都通过（含 `TestStopRead`）。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 41/41；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码），其余直接检查通过。
  - 对 `079f9c7` 的 Windows 完整运行：场景 29/29、门禁回归 36/36、组合 13/13。
- **定向撤回**（作者本机 Windows 与 WSL，对 `125c4f7`；每次只撤回一处；运行 `TestStopRead`、`TestPhaseClock`、F1 与遗留的门禁回归、健康对照 `control.silent-holder` 与真实管道场景）：

  | 变异 | 撤回的内容 | Windows | WSL |
  |---|---|---|---|
  | M39 | 停止改回在阶段 J 之前同步完成 | 两个窗口行失败（停止挡住了阶段 J） | 同左 |
  | M40 | Windows 上 Close 等待期间不再取消 | `window-released-in-J` 失败（停止没有完成）；`window-held-past-J` 中交出的两项不自行结束 | 不受影响（Unix 不用这一步） |
  | M41 | 挂起的停止不交给调用方 | `window-held-past-J` 失败（只交出 1 项） | 同左 |
  | M42 | 场景不等待接手的遗留 | `cleanup.leftover-running` 失败（得到 refuted） | 同左 |
  | M43 | 没有记录时不要求证据 | 两项 `report.epipe-*` 失败（得到 refuted） | 同左 |
  | M44 | 证据被忽略，没有记录一律 invalid | `control.f1-read-kept` 失败（得到 invalid） | 同左 |

  - 每个变异只由对应的检查检出；其余检查与健康对照都通过（真实管道场景 5 个 confirmed）。
  - N6 的三项时钟回归在每次撤回中都通过。

**对 T1 / T6 当前证据的影响**：

- **`1a6fec5` 之前**：管道路径的“停止 → 阶段 J → 封闭 → O3”，只在读取已经进入系统调用时有界成立。读取持有文件、尚未进入系统调用时，三平台的停止都会挡住阶段 J：Unix 上挡到读取离开窗口为止，Windows 上可能一直挡到有数据或 EOF。
  - 此前各轮场景的结论都是在没有遇到这一时序的运行中得到的，作为观测保留；它们不证明有界的输出收尾。
- **现在**：在任何读取状态下，阶段 J 的结束、封闭与 O3 的分类都在阶段 J 的截止之内到达（`TestStopRead`，假时钟）。停止本身在 Unix 上于读取离开系统调用时完成，在 Windows 上于读取进入系统调用时完成。
- **仍然只是受限的正例**：读取始终不进入系统调用时，停止挂起；读端、复制任务与停止任务在它结束之前一直被持有，没有上界。这是否可以接受、长驻宿主中怎样处理（C-4、D20），属于冻结之前的门槛，本轮不决定。
- **候选的变化**：Windows 的重复取消是一项新的候选机制（C-2），待审查。它把“写端何时有数据或关闭”从停止的完成条件中去掉，但没有让读取的调度变得有界。C-2、C-3 与 D20 的证据概括已按此改写（§9、§10）。

**仍未验证的边界**：

- **生产路径**：Runner 中的停止，以及它与取消、F1、`T_kill` 和 §8.6 持久化的组合，都没有接入；原型的 `adopt` 与 `lingering` 不是生产的登记。
- **重复取消的代价**：挂起期间每毫秒一次系统调用；读取长时间不进入系统调用时，这一代价随之持续，没有构造。
- **窗口**：读取停在窗口中只由闸门（RawConn）构造；真实调度下的长时间停顿没有构造。
- **迟到字节的证据**只在 `control.f1-read-kept` 这一种构造下使用；它证明写入成功，不证明其他成因。
- 不是硬实时保证。

**状态**：B1、N4-R 已修复，**待独立复审**；不宣布关闭或 GO。N6 经 Review 081 关闭（原时钟口径的范围内）。（其后：Review 082 认定 B1-Ra、B1-Rb 与 N4-R 的组合故障，见 §2.14。）

### 2.14 Review 082：停止结果的生命周期（B1-Ra、B1-Rb）与 N4-R 的组合故障

**审查结论**（[Review 082](reviews/codex-review-082.md) 第一、三、四节，对 `0f1f20b`）：**技术 NO-GO**。

- **B1 的原反例已修复**：同步 Close 不再挡住阶段 J；所有流共用一个截止，之后封闭、移交。
- **B1 的完整契约仍不能关闭**：
  - **B1-Ra**：停止在运行返回之后才出错时，错误随遗留接口丢失。`stopTask.until` 只给“是否结束”，`leftover` 只有 what / until，`adopt` 只看返回时的 `stopErr`，`done` 只看是否结束，O3 登记也直接丢掉已结束的项。审查方在仓库外以受控的 Closer 复现：同一错误早到时 invalid，晚到时 **confirmed / exit 0**。
  - **B1-Rb**：Windows 的重复取消与关闭自有句柄的错误被静默丢弃（`_ = cancelOwn`、`_ = closeOwn`）。审查方以 API 接缝注入 `ERROR_ACCESS_DENIED`，两处 `closeRead` 都返回 nil。
- **N4-R 不能整体关闭（NON-BLOCKER）**：迟到字节与同次的报告失败可以共存。审查方只保留 stdout 的读端，让持有者在停止计数之后写 stdout（成功）、再写 stderr（失败），报告也失败，结论仍为 refuted。
- **N6** 没有直接回退，保持原关闭范围；Windows 的重复取消只在受控的同步管道布局、API 成功与可释放的调度窗口下得到支持。
- 没有新增产品决定或接受风险。

**回归先行**（`e4eadfa9ae3e6e4d21a7532bdd59ee33494f87cd`，判定没有改）：

| 回归 | 内容 | 旧代码上 |
|---|---|---|
| 门禁回归 `stop.*`（7 项，内层 `TestInnerStopError`） | 场景控制的停止 Closer：干净地结束于运行之内、运行之后（要求 confirmed）；出错于运行之内、运行之后、复制器结束之后、两路中的一路；永不结束（要求 invalid） | 运行之后、复制器之后、两路之一这 3 项得到 **confirmed**，失败；其余 4 项通过 |
| 直接检查 `TestLingeringFailedStop` | 停止在运行返回之后带错误结束；拒绝策略下，登记不得放行后续运行 | 放行，失败 |
| 直接检查 `TestStopAPIErrors`（Windows） | 经接缝让 `CancelIoEx` 返回 `ERROR_ACCESS_DENIED`、让关闭自有句柄返回同样的错误（不关闭，测试自己收尾） | 停止的结果都是 nil，失败 |
| `TestStopRead` 的正常行 | 另要求停止没有错误，使“没有可取消的请求”保持为正常竞态 | 通过 |
| 门禁回归 `report.epipe-mixed-fault` | 审查方的组合故障：只保留 stdout 的读端；停止计数之后，持有者写 stdout（成功）、写 stderr（失败），`epipe` 报告也失败，之后再存活 | 得到 **refuted**，失败 |

- **三平台诊断** [run 40072151282](https://github.com/XKHoshizora/specmetry/actions/runs/40072151282)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `e4eadfa`。三平台都是上表中预期的失败（Windows 另有 `TestStopAPIErrors`）；场景 38、27、29 confirmed；组合 13/13。
- **macOS 另有一项非设计内的失败**（如实保留，没有重跑）：`report.epipe-unwritable-alive` 的内层结论正确（invalid），但外层普查读进程 2088 的环境时得到 EINVAL，判为普查工具错误，这项门禁回归因此不通过。2088 不在这次内层运行的创建记录中（`created` 为 2，持有者是 2072）；按普查的条件，它是同一用户、在内层运行期间启动的其他进程。这是 §13 已登记的普查前提（同一用户、启动不早于内层、读不到环境的进程使普查失败，保守），与本轮的改动无关，本轮不改普查。
- **作者本机 Windows**：与上表相同。

**修复 B1-Ra 与 B1-Rb**（`920220c2b3c7fee89b9a30371ac62734d0307c34`）：

- **停止任务保留它的结果**：`done` 只表示关闭已返回，不等于成功；任务在一个加锁的列表中保留它遇到的全部错误，随到随记，之后的成功不清除。
- **遗留接口带结果**：`leftover.until` 返回“是否结束”与“迄今的错误”。移交的是任务本身，接手者在结果到达时读取，不在移交时复制快照。挂起期间已经出现的错误也立即可见。
- **接手者消费结果**：`done` 读取每一项的结果；运行返回之后才到的错误、仍在运行的停止已经遇到的错误，都使观察失败（invalid），之后的复制结束或清理成功都不清除它。
- **O3 登记**：带错误结束的项留在 `failed` 中；拒绝策略下，挂起或带错误结束都使后续运行被拒绝；允许策略下照常放行，失败仍在记录中（`failures`）。
- **阶段 D 中结束的流**：它的关闭错误不再被丢弃，与停止的错误同样处理。
- **Windows 的 API 结果**（按微软的契约）：
  - `CancelIoEx` 返回 `ERROR_NOT_FOUND`：没有未完成的请求，是重复取消预期的竞态，不是错误。
  - `CancelIoEx` 成功：只表示已请求取消，不是结束；结束仍以 `File.Close` 返回为准。
  - `CancelIoEx` 的其他失败：立即记入停止的错误（挂起的停止也看得到），循环继续（Close 仍可能返回），之后的失败只计数、不重复记录。
  - 关闭自有句柄失败：它的释放未经确认，即使 Close 成功，停止也带错误；不再关闭第二次（句柄值可能已属于别人）。
  - `DuplicateHandle` 失败：与此前一样，停止带错误，只剩 Close。
  - 错误只写调用名与 Windows 的错误，不写句柄的值。
- 没有恢复同步 Close，没有增加预算或容差，没有丢弃任务。

**错误传播对应表**（B1）：

| 错误或事件 | 产生于 | 由谁保存 | 由谁消费 | 何时影响最终结论 |
|---|---|---|---|---|
| `File.Close` 的错误（两个平台） | 停止任务 | 停止任务的错误列表 | 运行返回时已结束：`runS6On` 放入 `stopErr`，`adopt` 消费；仍挂起：交出的遗留，`done` 消费 | 场景的 `done` 定结论之前：invalid |
| `CancelIoEx` 的非 `ERROR_NOT_FOUND` 失败（Windows） | 停止任务的取消循环 | 同上，发生时立即记入 | 同上；挂起时也能读到 | 同上，即使停止始终没有结束 |
| `CancelIoEx` 的 `ERROR_NOT_FOUND` | 同上 | 不保存 | — | 不影响（预期的竞态） |
| 关闭自有句柄失败（Windows） | 停止任务，在 Close 返回之后 | 停止任务的错误列表 | 同第一行 | 同第一行 |
| `DuplicateHandle` 失败（Windows） | 停止任务开始时 | 停止任务的错误列表 | 同第一行 | 同第一行 |
| 阶段 D 中结束的流的关闭错误 | `runS6On` | `stopErr` | `adopt` | 同第一行 |
| 停止到期仍未结束 | — | 交出的遗留 | `done`（统一期限 5 s） | invalid（cleanup not verified），它已有的错误另记 |
| 复制器的结束（停止之后读失败、写到已封闭的入口） | 复制任务 | 复制结果 | `runS6On`（O1 / O2 / O3） | 不是错误；只决定 O 的分类 |
| 带错误结束的遗留（登记中） | 停止任务 | O3 登记的 `failed` | 后续运行的策略（`admit`） | 拒绝策略下拒绝；允许策略下放行，失败仍在记录 |

- **未完成的任务由谁负责**：运行返回之前是运行；之后是接手它的调用方（场景以 `adopt` 接手，O3 登记以 `add` 接手），直到任务结束、结果被读取。
- **区分的状态**：
  - 停止请求尚未完成：任务的 done 未关闭（pending）；
  - 完成且成功：done 已关闭，结果为空；
  - 完成但失败：done 已关闭，结果不为空；
  - 复制结束：复制结果存在（O1 / O2），或交出的复制器已结束；
  - 资源释放：Close 已返回，Windows 上自有句柄也已成功关闭（失败计入结果）；
  - 入口封闭：阶段 J 结束时，总是。

**修复 N4-R**（`63b74324ee1e61efdcba9d98f3cd7ed810b291b6`）：

| 情形 | 依据 | 结论 |
|---|---|---|
| 预期的现象发生 | `epipe` 记录到达（等待回答期间才到达的也算） | 这一项预测成立 |
| 预期的现象没有发生，工具正常 | 没有 `epipe` 记录；场景发出询问之后，持有者回答 `ok`（只有迄今全部写入都成功的持有者才写）；**并且**停止之后每一路都经场景保留的读端有迟到字节 | refuted |
| 报告尚未到达、报告失败、工具失败 | 没有 `epipe`，也没有 `ok` | invalid |
| 证据不全 | 有 `ok`，但有一路没有保留读端或没有迟到字节 | invalid |
| 计数或状态不可读 | — | invalid |

- 迟到字节只证明相应那一路的写入，不能豁免同时发生的其他失败。
- 报告协议增加 `ok`。
- 运行中的其他错误（停止链、观察、清理）都记入同一个 invalid 列表，由 `done` 统一判定，不因已经找到 refuted 的证据而跳过。
- 两个“报告失败之后仍存活”的注入，存活期由 5 s 改为 10 s，使判定时持有者确实仍在运行；场景判定之后照常结束它。

**补充的回归**（`ba32dbe2c378cad96a18acaadfd6f7095f9726a0`，修复之后补上）：

- 判定的两部分（持有者的确认、每一路的迟到字节）都能拦住组合故障，所以单独撤回任一部分都没有检查发现。
- 于是补了两项，各自只由一部分拦住：
  - `report.ok-unwritable`：两路都保留、写入都成功，但持有者的 `ok` 写不出；
  - `report.one-stream-seen`：只保留 stdout，持有者只写 stdout 并确认。
- 旧代码上没有这两个注入，不声称旧代码上的对照。

**再补充的检查**（`2413634ba21838f7275ee4ddd523779f1409c9b6`，修复之后补上）：对 `ba32dbe` 的定向撤回中，又有两处只被计时偶然检出或完全没有检出。

- 把 `ERROR_NOT_FOUND` 当作失败（M50）只被 `control.f1-read-kept` 偶然检出：`TestStopRead` 中重复取消的第一次触发通常晚于读取进入系统调用，“没有可取消的请求”不一定出现。`TestStopAPIErrors` 增加 `nothing-to-cancel`：用真实的 `CancelIoEx` 并计数，读取停在窗口中，直到停止确实遇到过 `ERROR_NOT_FOUND`；要求停止没有错误。
- 等待回答之后再读一次失败记录（M53）没有检查覆盖。增加门禁回归 `report.epipe-late`：持有者在写入失败之后 3.5 s 才写记录，正好落在场景等待回答的期间；要求 confirmed。
- **修复代码固定为 `2413634`。**

**修复后的证据**（`2413634`）：

- **三平台诊断** [run 40074633075](https://github.com/XKHoshizora/specmetry/actions/runs/40074633075)：push，attempt 1，**completed/failure**，没有重跑；每个 Job 检出 `2413634` 本身。

  | 平台 | Job 结论 | 场景 | 门禁回归 | 组合 |
  |---|---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | success | 38/38 confirmed | 52/52 | 13/13 |
  | Windows | success | 29/29 confirmed | 48/48 | 13/13 |
  | macOS | **failure** | 27/27 confirmed | **42/43** | 13/13 |

  - **macOS 的失败**：本轮没有改动的旧门禁回归 `report.truncated`。内层结论正确（invalid），但判定之后的保险清理在普查中读进程 1989 的环境得到 EINVAL，于是记为“被测路径认为已结束、保险清理却失败”的矛盾，这项不通过。
    - 1989 不在这次内层运行的创建记录中（`created` 为 2，其一是 1969）；按普查的条件，它是同一用户、在内层运行期间启动的其他进程。
    - 这与回归先行那次 macOS 的失败是同一类（§13 的普查前提）；普查与保险清理的代码本轮没有改动；方向是保守的（只会使检查不通过）。
    - 按要求保留，没有重跑；是否单独处理这一前提，交负责人决定（§13）。
  - **本轮新增的回归在三平台都通过**：`stop.*` 7 项、`report.epipe-mixed-fault`、`report.ok-unwritable`、`report.one-stream-seen`、`report.epipe-late`、`TestStopRead`、`TestLingeringFailedStop`、`TestStopAPIErrors`（Windows，3 行）。
  - **组合**：B2 的两项 `forge-owner` 中夹具零动作、零信号，之后仍存活，再由组合结束；`forge-owner-mixed` 中本轮自己遗留的进程照常由被测补救结束；补救失败、观察失败、结束未知的组合仍使被测自检不通过。
  - **直接检查**：`TestStopRead`、`TestLingeringFailedStop`、`TestPhaseClock`、`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台都通过；`TestStopAPIErrors` 在 Windows 通过。
  - **SKIP**：只有三个只供内层运行的测试（`TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError`）。场景 ID 的并集仍为 53。
  - **`T1.pipe.escaped-holder` 的阶段 D**（自身时钟 / stopwatch）：Ubuntu 1.000781848 s / 1.000781889 s，macOS 1.0010885 s / 1.0010885 s，Windows 1.0001323 s / 1.0001563 s。
- **中间提交的诊断**（历史观测）：
  - `63b7432`，[run 40072621588](https://github.com/XKHoshizora/specmetry/actions/runs/40072621588)：success；门禁回归 49/49、40/40、45/45。
  - `ba32dbe`，[run 40072845216](https://github.com/XKHoshizora/specmetry/actions/runs/40072845216)：success；门禁回归 51/51、42/42、47/47。
- **作者本机**（`2413634` 的树）：
  - Windows 11：场景 29/29 confirmed；门禁回归 48/48；组合 13/13；直接检查都通过。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 52/52；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码）。
- **定向撤回**（作者本机 Windows 与 WSL，对 `2413634`；每次只撤回一处，都编译通过）：

  | 变异 | 撤回的内容 | Windows | WSL |
  |---|---|---|---|
  | M45 | 场景不读接手项的错误 | `stop.error-after-run`、`-after-copier`、`-one-of-two` 得到 confirmed | 同左 |
  | M46 | 挂起的停止交出时不带错误 | 同上三项，另有 `TestLingeringFailedStop` | 同左 |
  | M47 | 登记忘记带错误结束的项 | `TestLingeringFailedStop` | 同左 |
  | M48 | 取消失败不报告 | `TestStopAPIErrors/cancel-fails` | 不受影响（Unix 不用这一步） |
  | M49 | 关闭自有句柄失败被忽略 | `TestStopAPIErrors/own-handle-close-fails` | 不受影响 |
  | M50 | “没有可取消的请求”记为失败 | `TestStopAPIErrors/nothing-to-cancel`、`TestStopRead/window-held-past-J` | 不受影响 |
  | M51 | 不要求持有者的确认 | `report.ok-unwritable` 得到 refuted | 同左 |
  | M52 | 不要求每一路的迟到字节 | `report.one-stream-seen` 得到 refuted | 同左 |
  | M53 | 等待回答之后不再读失败记录 | `report.epipe-late` 得到 invalid | 同左 |

  - 每个变异只由对应的检查检出；健康对照、N6 的时钟回归与真实管道场景都通过。
  - 对 `ba32dbe` 的第一次撤回中，M50 只被 `control.f1-read-kept` 偶然检出、M53 没有被检出；原始记录保留，并因此补了 `2413634` 的两项检查（见上）。

**对 T1 / T6 证据的影响**：

- 停止的结果现在随任务移交，并在场景的最终结论之前消费；O3 登记保留带错误的结束。此前 `125c4f7` 的诊断中，停止晚到的错误不会进入结论，那时的“停止出错是观察失败”只对运行返回之前的错误成立。
- Windows 的重复取消现在区分预期的竞态与实际的 API 失败；它仍只在受控的同步管道布局、可释放的调度窗口下得到支持，不是任意取消失败、驱动或调度下的完成保证。
- N4：refuted 现在依据持有者的确认与每一路的迟到字节；只有这一种构造（`control.f1-read-kept`）得到 refuted。
- 受限正例与资源无上界（§2.13）不变，仍是冻结之前的门槛。

**仍未验证的边界**：

- API 失败只由接缝注入；原生的取消失败、句柄关闭失败没有构造。
- 持有者的确认是它自己的声明：一个写入成功、却错误地认为自己没有失败的持有者不在模型中。
- macOS 普查读不到同一用户其他进程的环境（EINVAL）时判为普查错误，这一保守的前提照旧；本轮在 macOS 上因此出现两次门禁回归不通过（回归先行的 `report.epipe-unwritable-alive`、最终诊断的 `report.truncated`），都没有重跑。
- 不是硬实时保证；生产路径没有接入。

**状态**：B1（B1-Ra、B1-Rb）与 N4-R **已修复，待独立复审**；不宣布关闭或 GO。N6 保持经 Review 081 的关闭范围。

## 3. 运行与环境

### 3.1 运行

| 运行 | 对象 | 触发 | 结果 |
|---|---|---|---|
| [run 40074633075](https://github.com/XKHoshizora/specmetry/actions/runs/40074633075) | `2413634ba21838f7275ee4ddd523779f1409c9b6`（Review 082 的修复，§2.14） | push，attempt 1 | **completed/failure**：Ubuntu、Windows success；macOS failure（旧门禁回归 `report.truncated` 的保险清理遇到同一用户其他进程的 EINVAL，§2.14，没有重跑）。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：52/52、42/43、48/48；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过</li></ul> |
| 本机 Windows 11 / WSL2 | `2413634` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 48/48、52/52；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M45 ~ M53 各自只被对应的检查检出 |
| [run 40072845216](https://github.com/XKHoshizora/specmetry/actions/runs/40072845216) | `ba32dbe2c378cad96a18acaadfd6f7095f9726a0` | push，attempt 1 | completed/success；门禁回归 51/51、42/42、47/47；历史观测 |
| [run 40072621588](https://github.com/XKHoshizora/specmetry/actions/runs/40072621588) | `63b74324ee1e61efdcba9d98f3cd7ed810b291b6` | push，attempt 1 | completed/success；门禁回归 49/49、40/40、45/45；历史观测 |
| [run 40072151282](https://github.com/XKHoshizora/specmetry/actions/runs/40072151282) | `e4eadfa9ae3e6e4d21a7532bdd59ee33494f87cd`（Review 082 的回归先行） | push，attempt 1 | **completed/failure**（预期）：设计中的回归在三平台都失败；macOS 另有一项普查的 EINVAL（`report.epipe-unwritable-alive`）；场景 38、27、29 confirmed；组合 13/13 |
| 本机 Windows 11 | `e4eadfa`（回归先行） | 作者手动 | 与 run 40072151282 中设计内的失败相同 |
| [run 40066910980](https://github.com/XKHoshizora/specmetry/actions/runs/40066910980) | `125c4f70114c312828f7f8fc74734761e14c343a`（Review 081 的修复，§2.13） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：41/41、32/32、37/37；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过</li></ul> |
| 本机 Windows 11 / WSL2 | `125c4f7` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 37/37、41/41；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M39 ~ M44 各自只被对应的检查检出 |
| [run 40066611501](https://github.com/XKHoshizora/specmetry/actions/runs/40066611501) | `079f9c72d57a0b182d970b0f8455a80d580e04c1`（没有补充的回归） | push，attempt 1 | completed/success；场景 38/38、27/27、29/29；门禁回归 40/40、31/31、36/36；组合 13/13 |
| [run 40066182430](https://github.com/XKHoshizora/specmetry/actions/runs/40066182430) | `bde00fc30cae12c0827a6d3125d04273d10af164`（Review 081 的回归先行） | push，attempt 1 | **completed/failure**（预期）。<ul><li>`TestStopRead` 的两个窗口行三平台都失败；</li><li>`report.epipe-unwritable-alive` 三平台都得到 refuted；</li><li>门禁回归 39/40、30/31、35/36；场景 38、27、29 confirmed；组合 13/13</li></ul> |
| 本机 Windows 11 | `bde00fc`（回归先行） | 作者手动 | 与 run 40066182430 相同 |
| [run 39994591073](https://github.com/XKHoshizora/specmetry/actions/runs/39994591073) | `ac3b3bed4489404124d51181c1e8ed902d55c82e`（N4、N6 的修复，§2.12） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：39/39、30/30、35/35；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过</li></ul> |
| 本机 Windows 11 / WSL2 | `ac3b3be` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 35/35、39/39；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M35 ~ M38 各自只被对应的检查检出 |
| [run 39992975299](https://github.com/XKHoshizora/specmetry/actions/runs/39992975299) | `5054e54a14d3e9084aa675880175232e61361c17`（N6 的修复） | push，attempt 1 | completed/success；场景 38/38、27/27、29/29；门禁回归 39/39、30/30、35/35；组合 13/13。历史观测：其直接检查后来在本机 Windows 上发现自身的缺陷（§2.12） |
| 本机 Windows 11 / WSL2 | `5054e54` 的树 | 作者手动 | 完整运行都通过；Windows 的 `T1.pipe.escaped-holder` 阶段 D 自身时钟 1.0001495 s、stopwatch 999.6671 ms。变异运行中 `TestPhaseClock` 在 Windows 上超时一次；重复 60 次，180 个子检查失败 12 个（§2.12） |
| [run 39991866363](https://github.com/XKHoshizora/specmetry/actions/runs/39991866363) | `91229b5beb7b6b06fe3a2d97b3e99f25d1a9e882`（N4、N6 的回归先行） | push，attempt 1 | **completed/failure**（预期）。<ul><li>`report.epipe-unwritable` 三平台都得到 refuted；</li><li>`TestPhaseClock` 三行三平台都失败；</li><li>门禁回归 38/39、29/30、34/35；场景 38、27、29 confirmed；组合 13/13</li></ul> |
| 本机 Windows 11 / WSL2 | `91229b5`（回归先行） | 作者手动 | 与 run 39991866363 相同：门禁回归 34/35、38/39；`TestPhaseClock` 三行失败 |
| [run 39977737240](https://github.com/XKHoshizora/specmetry/actions/runs/39977737240) | `8a2b97ccd521c6af587baf4e3f45421d68dafcd9`（第五轮修复） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：36/36、27/27、32/32；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过</li></ul> |
| [run 39976843072](https://github.com/XKHoshizora/specmetry/actions/runs/39976843072) | `f1b8566bde691d738b804c4defd129605367c14e`（第五轮回归先行） | push，attempt 1 | **completed/failure**（预期）。<ul><li>组合：三平台都是 13 个中 2 个失败（两项 `forge-owner`）；</li><li>门禁回归：36/36、27/27、32/32；</li><li>场景：38、27、29 confirmed</li></ul> |
| 本机 Windows 11 | `8a2b97c` 的工作树 | 作者手动 | 场景 28 confirmed、1 refuted（`T1.pipe.escaped-holder`，N6，阶段 D 999.5519 ms，没有重跑）；门禁回归 32/32；组合 13/13；直接检查都通过 |
| 本机 WSL2 | `8a2b97c` 的工作树（交叉编译） | 作者手动 | 场景 37 confirmed、1 unsupported（没有 pwsh）；门禁回归 36/36；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码），其余直接检查通过 |
| 本机 Windows 11 / WSL2 | `f1b8566`（回归先行） | 作者手动 | 组合 13 个中 2 个失败（两项 `forge-owner`）；场景 1/1 confirmed |
| run 39966556540 | `31c0b29`（第四轮修复） | push，attempt 1 | 历史观测（§3.4）：completed/success；场景 38/38、27/27、29/29；门禁回归 36/36、27/27、32/32；组合 10/10 |
| run 39964760402 | `4efc9fc`（第四轮回归先行） | push，attempt 1 | completed/failure（预期，§2.10）；Windows 有 1 个 refuted（N6，999.9851 ms）；macOS 一项组合的保险清理失败 |
| 本机 Windows 11 / WSL2 | `31c0b29` 的工作树 | 作者手动 | 历史观测：场景 29/29、37 + 1 unsupported；门禁回归 32/32、36/36；组合 10/10 |
| 本机 Windows 11 / WSL2 | `4efc9fc`（回归先行） | 作者手动 | 组合 10 个中 6 个失败；门禁回归 32/32、36/36 |
| run 39954957954 | `e77af94`（第三轮修复） | push，attempt 1 | 历史观测（§3.4）：completed/success；场景 38/38、27/27、29/29；门禁回归 36/36、27/27、32/32 |
| run 39951830770 | `8f42e58`（第三轮回归先行） | push，attempt 1 | completed/failure（预期，§2.9）；Windows 有 1 个 refuted（N6，999.82 ms） |
| 本机 Windows 11 | `8f42e58`（回归先行） | 作者手动 | 门禁回归 10/31 失败；另一次只跑场景（不含门禁）的运行中，`T1.pipe.escaped-holder` refuted（阶段 D 999.92 ms），其余 28 个 confirmed |
| 本机 WSL2 | `8f42e58`（回归先行） | 作者手动 | 门禁回归 12/35 失败 |
| run 39887528435 | `8fea6c6`（第二轮修复） | push | 历史观测（§3.4） |
| run 39886679862 | `1df33df`（第二轮回归先行） | push | completed/failure（预期，§2.8） |
| run 39835302375 | `a793ced`（第一轮修复） | push | 历史观测（§3.4） |
| run 39830995366 | `5140588`（第一轮回归先行） | push | completed/failure（预期，§2.7） |
| run 39815658463、39816938304 | `bcf080f`、`4754142` | push | 历史观测（§3.4） |

- **第五轮修复后的运行**（run 39977737240）：
  - 每个 Job 检出 `8a2b97c` 本身；没有重跑。
  - SKIP 只有两个只供门禁回归内层运行的测试：`TestInnerForeignPID` 与 `TestInnerLeak`。
  - Ubuntu 的原型步骤带 `-race`，没有数据竞争的报告。
  - 各平台的场景 ID 集合与此前各轮相同，并集为 53。38 + 27 + 29 是 94 次平台观察，不是 94 个独立场景；结果行在 Results 步骤中重印，按 ID 去重计数。
- **本机的两次修复后运行**都在提交 `8a2b97c` 之后，对同一棵树进行。
- **N4、N6 修复之后的运行**（run 39994591073）：每个 Job 检出 `ac3b3be` 本身；没有重跑；SKIP 同上；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53。§3.3 的各格在这次运行中相同。
- **Review 081 修复之后的运行**（run 40066910980）：每个 Job 检出 `125c4f7` 本身；没有重跑；SKIP 同上；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53；§3.3 的各格相同。
- **Review 082 修复之后的运行**（run 40074633075）：每个 Job 检出 `2413634` 本身；没有重跑；SKIP 只有三个只供内层运行的测试；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53。macOS 的 Job 因一项旧门禁回归的普查 EINVAL 失败（§2.14）。
- **不同运行的统计不混合**：§3.3 的结论来自 run 39977737240。§4 ~ §8 的数值仍来自 run 39966556540（`31c0b29`）。第五轮只改动外层门禁的文件，场景代码没有变化，run 39977737240 中这些场景的结论相同；单次运行的数值以各自的日志为准。

### 3.2 环境

| 环境 | 系统 | Go | 解释器（T5） |
|---|---|---|---|
| Ubuntu runner | Linux 6.17.0-1022-azure，x86-64；带 `-race` | 1.27.3 | `/bin/sh` → dash；bash 5.2.21；dash；pwsh 7.6.6 |
| macOS runner | macOS 26.6.2（Darwin 25.6.0），arm64 | 1.27.3 | `/bin/sh`（bash 3.2.57 的 sh 模式）；bash 3.2.57；dash；pwsh 7.6.6 |
| Windows runner | Windows 10.0.26100（Server 2025），x86-64 | 1.27.3 | Git for Windows 2.56.0（`usr\bin\sh.exe`、`usr\bin\bash.exe`、`bin\sh.exe`，bash 5.3.15）；Windows PowerShell 5.1.26100.33648；pwsh 7.6.7；`cmd` |
| 本机 Windows | Windows 11 Pro 10.0.26300 | 1.27.1 | Git for Windows 2.56.0；Windows PowerShell 5.1.26100.9711；pwsh 7.6.7 |
| 本机 WSL2 | Ubuntu 24.04，Linux 6.18.33.2-microsoft-standard-WSL2；普通用户 | 1.27.1 | dash；bash 5.2.21（没有 pwsh） |

runner 的镜像在各轮之间变化：Ubuntu 在 run 39816938304 为内核 7.0.0-1017-azure，此后各轮为 6.17.0-1022-azure；Windows runner 上的 Windows PowerShell 与 pwsh 在 run 39887528435 为 5.1.26100.33648 与 7.6.7（此前为 33438 与 7.6.6）。

### 3.3 逐场景的结论（第五轮修复后）

✓ 为 `confirmed`，— 为 `unsupported`，空白为该平台没有这一场景（平台专属）。CI 列是 run 39977737240，没有任何 `refuted` 或 `invalid`，与此前三轮修复后（run 39887528435、39954957954、39966556540）的表逐格相同。本机列是 `8a2b97c` 的工作树；本机 Windows 的 `T1.pipe.escaped-holder` 这一次为 refuted（N6，阶段 D 999.5519 ms，§13），其余逐格相同。N4、N6 修复之后的 run 39994591073 与作者本机对 `ac3b3be` 的运行逐格相同，本机 Windows 的 `T1.pipe.escaped-holder` 也为 confirmed（阶段 D 在它自身的时钟上判定，§2.12）；上表的本机列照原样保留。

| 场景 | 门槛 | 类别 | Ubuntu | macOS | Windows | 本机 Win11 | 本机 WSL |
|---|---|---|---|---|---|---|---|
| `T1.O3.gate-lingering` | T1 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.F1-skip-drain` | T1 | 注入 | ✓ | ✓ | ✓ | ✓ | ✓ |
| `T1.pipe.blocking-read-end` | T1 | 反例 | ✓ | ✓ |  |  | ✓ |
| `T1.pipe.escaped-holder` | T1 | 正常 | ✓ | ✓ | ✓ | refuted（N6） | ✓ |
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
  - 当时修复后的证据是 run 39887528435，这些场景在三平台都 confirmed；它现在也是历史观测（见下一条），当前证据见 §3.1。
- **Review 077 之后**：run 39887528435（`8fea6c6`）与其常规 CI 同样保留为历史观测。它的门禁回归在当时的范围内成立（Review 077 第三、四节确认精确反例的修复），但有两点没有证明。
  - **没有证明其余共享入口的一次失败必使运行无效**：
    - 影响经 Linux 清扫的 T2 场景：绑定之后读不到状态时记为身份不符，只有 `T2.adoption.chain` 核对了不符数为 0；
    - 影响 Linux 全部场景的留下子进程检查：起点的列表读不到当作空列表；
    - 影响核对 subreaper 的 T2 场景；
    - 影响 Windows 的 `T4.start.fail-resume`：Job 成员读不到时走另一分支，而预测不看这一分支。
    - 没有证据表明当时发生过这些失败。
  - **没有证明外层的独立核验在见证本身失败时仍拒绝通过**：当时的“已启动的辅助进程都已结束、都已登记”只在见证完整、身份有效时成立；当时的门禁回归中，见证写入是否全部成功没有被核对。
  - 当时修复后的证据是 run 39954957954，这些场景与全部门禁回归在三平台都通过；它现在也是历史观测（见下一条），当前证据见 §3.1。
- **Review 078 之后**：run 39954957954（`e77af94`）与其常规 CI 同样保留为历史观测。它的场景与门禁回归在当时的范围内成立（Review 078 第四、五节），但没有证明“外层的负向自检在它自己的补救、结束确认、普查或读取失败时不会通过”：当时的判定只要找到预期的拒绝类别就通过（§2.10）。
  - 健康的正对照与已修复的原反例不受影响：它们的外层没有任何问题，补救没有发生或已成功。
  - 当时修复后的证据是 run 39966556540（`31c0b29`）：场景、门禁回归、组合判定与直接检查在三平台都通过；它现在也是历史观测（见下一条），当前证据见 §3.1。
- **Review 079 之后**：run 39966556540（`31c0b29`）与其常规 CI 保留为历史观测。它的场景、门禁回归与组合在当时的范围内成立（Review 079 第四、五节），但没有证明“保险清理只对归属已证明的进程动作”。
  - 在 Unix 上，一条格式合法、归属无效的见证记录，可以让保险清理向本轮之外的进程发信号（§2.11）。
  - 当时的健康运行中没有这样的记录；没有证据表明当时发生过这样的动作。
  - 当时的证据是 run 39977737240（`8a2b97c`）：场景、门禁回归、组合与直接检查在三平台都通过（§3.1、§2.11）。
- **N4、N6 之后**（§2.12）：`ac3b3be` 之前的各次运行中，阶段 D 的时长是在 stopwatch 上计量的；其中的 refuted 照原样保留，不改写为通过。此前 `T1.pipe.F1-skip-drain` 的 confirmed 都有 `epipe` 记录，不受 N4 影响。run 39992975299（`5054e54`）保留为历史观测；当前证据是 run 39994591073（`ac3b3be`）。
- **Review 081 之后**（§2.13）：`1a6fec5` 之前的各次运行中，停止读取在调用者中同步进行，阶段 J 之前可能被一次读取挡住。那些运行没有遇到这一时序，场景的结论作为观测保留，但不证明有界的输出收尾。`control.f1-read-kept` 在 `079f9c7` 之前的 refuted 以“持有者仍在运行”为依据，现在以迟到字节为依据。run 39994591073、39992975299 与 run 40066611501 保留为历史观测；当前证据是 run 40066910980（`125c4f7`）。
- **Review 082 之后**（§2.14）：`920220c` 之前的各次运行中，停止在运行返回之后才出现的错误、Windows 取消与关闭自有句柄的错误都不会进入结论；那些运行中没有这样的错误被注入，场景的结论作为观测保留。`control.f1-read-kept` 在 `63b7432` 之前的 refuted 只以迟到字节为依据，现在另需持有者的确认。当前证据是 run 40074633075（`2413634`），其中 macOS 的那项失败如实保留。

## 4. T1 / T6：管道路径的收尾与受控目的端

**已证明**（在原型的范围内；本节与 §4.1 ~ §8 的数值来自当前的重跑 run 39966556540，`31c0b29`；此前各轮当时的数值在它们自己的日志中，不再引用）：

- **正常排空**（`T1.pipe.normal-drain`）：负载为辅助进程写完全部字节并以 0 退出；262 144 + 131 072 字节经两个复制任务完整、按序到达（O1、O1），没有停止读取，文档在最后且完整。
- **D 阶段到期**（`T1.pipe.escaped-holder`）：一个不在已确认进程树中的写端持有者（停止之前写了 80 ~ 198 次）使 D 阶段用满 `T_drain`（1.0 s）；停止读取之后复制任务在 J 阶段立即返回（O2），持有者之后的写入失败（Linux / macOS 为 broken pipe，Windows 为 “The pipe is being closed”）。
- **F1 跳过 D 阶段**（`T1.pipe.F1-skip-drain`、`T1.pipe.silent-holder`）：持有管道的后代经核验属于本次检查，父辅助进程以预期的退出码结束；停止读取使两个阻塞的复制任务都返回（O2）；后代之后的写入失败并报告；封闭之后目的端只多出所有者的文档。
- **停止读取的延迟**（`T1.pipe.stop-read-latency`，每平台 50 次，每次先确认复制任务阻塞在 Read 中，这是负载门禁）：

  | 平台 | 最小 | 中位 | 最大 |
  |---|---|---|---|
  | Ubuntu runner | 5.3 µs | 10.7 µs | 27.1 µs |
  | macOS runner | 15.7 µs | 32.0 µs | 140 µs |
  | Windows runner | 65.8 µs | 74.9 µs | 128.1 µs |

  真实进程持有写端时（`T1.pipe.silent-holder`，每平台 3 次，每次两个复制任务都阻塞在 Read 中）为 26 ~ 163 µs。这些是所测样本的最大值，不是调度上界或实时保证；本文不据此改变提议的 `T_join`（草案 §8.3）。
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

- **有时限、不回收的观察**（`emain.exit-unreaped`、`emain.bounded-timeout`）：Linux 上对主进程的 pidfd 做有时限的 `poll`，再以 `waitid(P_PID, WEXITED | WNOWAIT)` 取得状态。主进程在 300 ms 退出时，约 305 ms 观察到，状态为 `exit 7`，进程仍是僵尸、组不变、没有留下等待的 goroutine；不退出时在时限（500.5 ms）返回，之后对组发出 SIGKILL，观察到 `signal 9`。
- **组身份**（`group.zombie-leader`）：成员经核验在本组中；组长退出而组中仍有成员时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束，确认之后才回收组长；回收之后 `kill(-pgid, 0)` 为 ESRCH，原型此后拒绝再向这个组发信号。
- **边界 1**（`group.escape-boundary`）：`setsid` 的后代经核验属于本次检查、在另一个组中；组被确认为空时它仍存活（Linux 上由 subreaper 收回，见 §5）。

## 5. T2：Linux 的排他区间与身份

**已证明**（Ubuntu runner 带 `-race`，与本机 WSL2）：

- **闸门强制排他区间**（`T2.gate.exclusive-interval`）：四个 goroutine 不停地经**同一个闸门实例**启动子进程（其中一个是真实的 `git --version`：Ubuntu 上 239 次），三次检查在其间各自取得排他区间：每次基线为空，进程链的每一层都经核验属于检查，被收养的进程都在其中，全部经 pidfd 回收；共享租约与检查区间的重叠为 0；之后 subreaper 为 0，没有子进程。它不证明生产中的全部调用点已经接入闸门。
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

全部来自真实的 macOS runner（macOS 26.6.2，Darwin 25.6.0，arm64）；结论在各轮的运行中相同，数值来自 run 39966556540；没有用其他平台的实验代替。这些证据绑定所测的版本与 arm64 ABI，不自动覆盖其他版本。

| 问题 | 结果 | 场景 |
|---|---|---|
| 有时限的退出观察 | kqueue `EVFILT_PROC`（`NOTE_EXIT \| NOTE_EXITSTATUS`）：主进程 300 ms 退出，约 316 ms 观察到 `exit 7`；不退出时在 503 ms 的时限返回；没有留下等待的 goroutine | `emain.exit-unreaped`、`emain.bounded-timeout` |
| `WNOWAIT` 与回收的顺序 | 观察之后主进程仍是僵尸（`sysctl` 的 `SZOMB`），组不变；`waitid(P_PID, WEXITED \| WNOWAIT)` 经原始系统调用（固定的 siginfo 偏移）可用，两次都给出 `exit 5` 而不回收；之后 `Wait` 回收，退出码一致 | `emain.waitid-wnowait`、`emain.exit-unreaped` |
| 组身份与成员结束的确认 | 成员经核验在本组中；组长退出、成员存活时，组长保持为僵尸，`kill(-pgid, 0)` 成功，组终止使成员结束；`sysctl kern.proc.pgrp` 中不含僵尸的成员为空即确认结束，组长仍未回收；回收之后 `kill(-pgid, 0)` 为 ESRCH | `group.zombie-leader` |
| 只剩僵尸组长时的组信号 | `kill(-pgid, 0)` 返回 **EPERM**（Linux 上成功）。这只是所测系统的观察：它不能解释为组已消失，也不能推广为“任何平台上的 EPERM 都无害”（C-12） | `emain.exit-unreaped` |
| 脱离进程组的后代 | `setsid` 的后代（经核验属于本次检查）在组被确认为空之后仍存活，没有 subreaper 可以发现它（草案 §8.2 的边界 1 在 macOS 上保留；D7 没有因此获批） | `group.escape-boundary` |
| 输出的关闭与复制的收尾 | 与 Linux、Windows 相同：停止读取在 15.7 ~ 140 µs 内返回阻塞的 Read；D / J 阶段、封闭、O1 ~ O3、容量、别名、最后文档的场景都成立；阻塞模式读端的反例也复现 | §4 的各场景 |
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
| `T4.chain.normal` | — | 负载：Job 中恰好一个进程；恢复之后用户代码运行；`TerminateJobObject` | 130 µs 内 `ActiveProcesses = 0` | 运行（有标记文件） | — |
| `T4.teardown.terminate-fails` | 终止请求 | 不确认结束（F1）；关闭 Job 句柄（`KILL_ON_JOB_CLOSE`，再次请求） | 协议不确认；场景以自己的句柄**观察到**进程随后结束 | 运行 | — |
| `T4.teardown.query-fails` | 结束查询 | 不确认结束（F1） | 协议不确认；场景以句柄观察到进程已经结束 | 运行 | — |
| `T4.teardown.request-is-not-end` | — | 负载：每轮 30 个后代经核验在 Job 中、31 个活动进程；5 轮 | 5 轮都确认；请求之后第一次查询已为 0，请求到确认 0.50 ~ 0.79 ms；清理核验 155 个进程 | 运行 | 观察句柄在清理之后关闭 |

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
| C-2 | **管道路径的读端只用 Go 自建的管道（`os.Pipe`），停止读取在阶段 J 内由停止任务完成**（Review 081 B1）：Unix 上读端在 poller 中，关闭使阻塞中与之后的读取返回；Windows 上句柄是同步的，关闭只取消已经发出的读取，停止任务在关闭等待期间经自有的复制句柄重复取消，读取一旦进入系统调用即返回。不得包装外部给出的描述符，也不得把读端设为阻塞模式 | `T1.pipe.stop-read-latency`、`T1.pipe.silent-holder`、直接检查 `TestStopRead`；反例 `T1.pipe.blocking-read-end` | 按平台描述取消能力（Windows 不能套用 Unix 的 O_NONBLOCK）；读取始终不进入系统调用时停止挂起，交给调用方（C-3）；Windows 的重复取消是新增的候选，待审查，它的 API 结果按契约区分（`ERROR_NOT_FOUND` 是预期竞态，其他失败都是停止的错误，§2.14），只在受控的同步管道布局与可释放的调度窗口下得到支持；不推广到任意 Reader；保留 O3 作为兜底 |
| C-3 | **O3 复制任务与挂起的停止的归属**：复制任务（goroutine）持有它的读端对象（Specmetry 已调用 Close，真正的关闭推迟到 Read 返回）与已封闭的复制入口（仍引用目的端，但不能写出）；挂起的停止任务持有读端，直到关闭返回（Review 081 B1）。运行返回时把两者交给调用方（`s6Result.left`）；Runner 在进程级登记全部未确认结束的复制任务与停止任务，直到它们结束。“不再写出”从封闭起成立；“资源已结束”要等读取与关闭返回 | `T1.O3.gate-lingering`、`T1.pipe.blocking-read-end`、直接检查 `TestStopRead`、门禁回归 `cleanup.leftover-running` | Read 返回不一定要等最后一个写端结束：一次数据也可以使它返回，之后的写入被封闭的入口拒绝；登记不是释放，封闭不是任务回收；挂起的停止没有资源上界，是冻结之前的门槛；停止的结果（含运行返回之后才到的错误）随任务移交，接手者在结论之前读取，登记保留带错误的结束（§2.14） |
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
| C-19 | **观察失败不可撤销**：一次检查中任何一次状态读取、等待、回收或枚举失败，都使这次检查的结论保守（原型中为 invalid，生产中按 F1 或环境错误）；之后的成功读数不能恢复它；“读不到”不能变成“不存在”“空列表”“另一个进程”或“0”。结束与回收仍照常进行，但“已确认结束”与“观察曾失败”分开报告。被测机制预期的故障（身份不符、pidfd 不可用、注入的 Job 查询失败）不在此列 | 门禁回归 `observe.fail-once`、`observe.sweep-recheck-once`、`observe.sweep-wait-once`、`observe.baseline-once`、`observe.prctl-fails`、`observe.injob-once`；变异 M8、M13 ~ M17 | 生产中的分类（F1 / 退出码 6）依赖 D19、D20 与错误约定。“一次失败、之后成功”的注入只在所列的 6 个观察点上做了；其余共享入口（Windows 的等待与 Job 确认、macOS 的退出观察、Unix 清理的回收）只有持续失败的注入或源码层面的覆盖（§2.9 表 A），不能概括为全部已动态验证。Review 078 关闭了原型门禁中的 B1-R1-S，确认原型的错误累积得到支持，但这不批准生产的分类；候选仍待审查 |
| C-20 | **拒绝路径的收尾**：在排他区间内因负载、报告或归属被拒绝的检查，在离开区间之前先结束它的进程组，再清扫区间内被收养的全部进程（按区间的规则它们属于这次检查，不论是否报告过），然后才清除 subreaper、释放闸门；已知的进程在拒绝之前就登记 | 门禁回归 `load.chain-short`、`load.chain-corrupt`；变异 M9、M10 | 只在 Linux 的排他区间内成立；区间外（macOS、Linux 的 `group.escape-boundary`）报告读不懂的脱组后代无法找到，只能如实申报为缺额；生产 Runner 的拒绝路径尚未实现。原型中拒绝之后的状态由外层的普查独立核验（§2.9），那是测试基座的手段，不是生产 Runner 的候选；外层负向自检在补救失败时不通过（§2.10） |

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
| **输出** | D20 输出的支持契约与转发不完整 | (i) 继承句柄与受控目的端，其他 Writer 拒绝；(a) 转发不完整时照常记录、停止、退出码 6 | 一次转发故障中止整个批次；进程内调用的输出对象受限 | 受控目的端的规则、O1 ~ O3、迟到输出、别名、容量与最后文档在三平台成立（C-2 ~ C-7）；阶段 J 的结束、封闭与 O3 在 `T_join` 之内到达，但读取始终不进入系统调用时停止挂起，资源没有上界（§2.13）。**同时须明确 C-4**：O3 之后同一宿主中新的调用是拒绝还是允许，区分单次 CLI 与长驻宿主（§9） |

## 11. 冻结之前仍缺的规格与原型

按 Review 075 第十节第 1 点。T1 ~ T6 只得到局部支持，不因门禁修好而宣布完成：

1. **门禁的独立复审**：已完成。
   - B1-R1-S 经 Review 078、B1-R2-W-R 经 Review 079、B2 经 Review 080 关闭；Review 080 对门禁与清理给出技术 GO，限定在本原型声明的对象来源与受控继承 / Job 前提内。
   - 这不等于 T1 ~ T6 完成。Unix 上身份复核与信号之间的窗口、归属读取失败的原生动态验证、任意宿主的适用性仍未验证（§2.11、§13）。
2. **候选的前提与措辞**（§9 的“前提与限制”）：特别是 O3 之后的后续调用（C-4 与 D20）、父端独立于用户代码的期限、Windows 未测的原生失败点、macOS 的原始 ABI 与枚举的错误和预算、Linux F1 之后的宿主状态、挂起的停止的资源边界及它与 C-4 / D20 的关系、Windows 重复取消这一候选（§2.13）。Review 080 第九节另列出：T1 / T6 的目的端、独立预算与 O3 的资源归属，Windows 父端的提前失败与所有权移交，macOS 的支持版本。
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
- **门禁本身**：
  - 门禁回归与变异覆盖了所列的故障，不是对一切可能的损坏的证明。变异只在作者本机运行（Windows 与 WSL），macOS 没有变异。
  - Linux 上读 `/proc` 失败只由注入模拟；在 `hidepid` 等配置下，读不到的进程会使观察为 invalid（保守），没有测试。
  - “一次失败、之后成功”的注入只在 6 个观察点上做了（`pidProc.ended`、清扫绑定之后的读取、清扫的 `waitid`、起点的子进程列表、subreaper、Windows 的 Job 成员读取）。其余共享入口的失败按 §2.9 表 A 记入终判，只有持续失败的注入或源码层面的覆盖。
- **外层核验的范围**（§2.9）：
  - **见证**：覆盖内层启动的每一个进程，包括 `git`、解释器（它们只有创建记录，没有回执）；回执只来自测试二进制自己的辅助进程。`TestStartPoints` 是对本包源码的文本扫描，不覆盖依赖库内部启动的进程（本原型没有这样的依赖）。
  - **带外通报**：Unix 是对外层进程的 `SIGUSR1`，Windows 是外层的命名事件。通报本身失败（发信号或打开事件失败）不能再被检出；在 Unix 上，这时只剩写者自己的退出码。Windows 上 Job 的已创建进程数另外兜住缺少创建记录的进程，Unix 没有对应的计数。
  - **Unix 上计数不到的情形**：一个绕过 `start` 的启动点，如果它启动的进程在外层查看之前已经结束，Unix 上发现不了，只能靠 `TestStartPoints`。仍存活的，普查能发现。
  - **普查的前提**：本轮的进程从内层继承唯一的标记，并且可读其环境（同一用户、可 dump）。启动不早于内层、属于同一用户、却读不到环境的其他进程，会使普查失败（保守，这项检查不通过）。Review 082 这一轮在 macOS 上实际出现两次（EINVAL：回归先行 run 40072151282 的 `report.epipe-unwritable-alive`、最终诊断 run 40074633075 的 `report.truncated`；出错的进程都不在内层的创建记录中），都没有重跑。是否单独处理这一前提（例如在 macOS 上区分正在退出的进程），交负责人决定。一个故意清除自己环境的进程，普查看不到；本原型的辅助进程没有这样做。
  - **测试条件的变化**：Windows 的内层运行在外层自己的 Job 中，内层每次检查的 Job 嵌套在它下面；门禁之下，辅助进程先等待自己的创建记录再做事。正常的诊断运行（不在门禁之下）不受影响。
  - **补救**只针对普查找到的本轮进程（Windows 按 Job 成员；Unix 按标记与身份），从不使用无效见证中的 PID；补救之后的状态另行记录。区间外脱组的后代在报告读不懂时无法登记，只能申报为缺额（`group.escape-boundary` 的损坏报告没有回归）。
- **Review 076 N4**（NON-BLOCKER，Review 080 保持；第一次修复见 §2.12；Review 081 认定残余 N4-R，Review 082 认定其组合故障，**已再修复，待独立复审**，§2.13、§2.14）：持有者写 `epipe` 记录失败时，F1 场景把“没有观察到预测的事件”记为 refuted，而不是把“观察工具失败”记为 invalid；没有形成假通过。建议把持有者的报告失败作为独立的工具故障状态传给场景，并增加分类回归。修复：没有记录时先读持有者的状态，已结束而没有记录为 invalid（§2.12 的表）。这一判定已由 §2.13 取代：没有记录时，只有迟到字节的证据才是 refuted，其余为 invalid。其后又由 §2.14 取代：refuted 需要持有者的确认与每一路的迟到字节。
- **Review 077 N5**（经 Review 078 关闭）：`missing` 改称 `shortfall`（预期登记的缺额），外层另列出已创建而未登记的进程及其结束状态（`unowned`）；“未登记数不多于缺额”只是数量上的容纳，不是逐个对象的对应，也不是结束的证明（结束由普查与逐条记录证明）。
- **Review 078 N7**（经 Review 079 关闭）：§3.4 区分“当时的证据”与“当前证据”，§4 ~ §8 指向重跑的运行并更新了数值。
- **Review 078 N6：Windows 上阶段 D 的计时**（NON-BLOCKER，Review 079、080 保持；已修复，**经 Review 081 关闭**（原时钟口径的范围内），§2.12）：
  - **现象**：`T1.pipe.escaped-holder` 预测“阶段 D 用满 `T_drain`”（计得的时长 ≥ 1 s）。截止与剩余预算由 Go 的 `time.Now` / `time.Until` 与定时器算出，它们在 Windows 上读中断时间（审查核对了 Go 的源码）；计时用的是性能计数器（`stopwatch`）。两种口径之间，严格的 ≥ 1 s 比较存在边界不一致。
  - **观察**：
    - 回归先行的诊断 run 39951830770 的 Windows Job（999.8195 ms）、作者本机同一提交的一次运行（999.92 ms），以及第四轮回归先行的 run 39964760402（999.9851 ms），都因此判为 refuted；
    - 第五轮中，作者本机对 `8a2b97c` 的完整运行（999.5519 ms）也因此判为 refuted；
    - 其余各轮与修复后的 run 39954957954（1.00045 s）、run 39966556540（1.00043 s）、run 39977737240 都 ≥ 1 s；
    - 审查方本机限定的三次运行都 ≥ 1 s；Review 080 的审查方本机没有运行这一场景。
    - 修复之前的最后一次本机观察：对 `5054e54` 的完整运行中，stopwatch 读数 999.6671 ms，阶段自身的时钟 1.0001495 s；新口径下为 confirmed。
  - **审查的裁定**：支持“时钟口径与严格比较导致随机拒绝”。没有证明输出安全失效或假通过；“唯一根因已经证明”仍过强。
  - **建议**：截止与计时用同一时钟或同一定义，或依据明确的分辨率制定容差，并单独保留边界回归；不能只为取得绿色而放宽。每一次 refuted 都如实保留，没有为取得绿色而重跑。
  - **修复**（§2.12）：截止与计量都在阶段自己的时钟上；没有容差，参数不变；确定性的边界回归为 `TestPhaseClock`。
- **停止读取的生命周期**（Review 081 B1，§2.13；此前登记为“Windows 上停止读取的时序前提”）：停止已纳入阶段 J，阶段 J 的结束、封闭与 O3 在 `T_join` 之内到达；停止本身在 Unix 上于读取离开系统调用时、在 Windows 上于读取进入系统调用时完成。读取始终不进入系统调用时停止挂起，读端、复制任务与停止任务一直被持有，没有上界；挂起期间 Windows 每毫秒一次取消。生产中怎样处理（C-4、D20）属于冻结之前的门槛。停止的结果随任务移交并在结论之前消费，Windows 的 API 结果按契约区分（Review 082，§2.14）。
- **组合判定的范围**（§2.10）：
  - 组合回归只在外层被测路径的四个位置注入故障（普查、一次记录读取、补救、补救的结束确认），并只在两项故意遗留、一项见证删除、一项写入失败与一项健康对照上组合；其余组合由判定表（`TestGateJudge`）在固定的输入上检查，不是真实路径。
  - 原生的 `TerminateJobObject` 失败、原生的 `/proc` 或 `sysctl` 失败都没有构造，只有注入。
- **保险清理的范围**：
  - 它独立于被测路径的注入，但与被测路径用同样的身份核对与普查方法，不是另一套机制；它不能弥补普查方法本身的缺陷。
  - 在 Unix 上，被测路径没有认出、又读不到环境的进程，它也找不到。
  - 它只对已证明归属的进程动作（Review 079 B2，§2.11）：普查找到的，或只凭记录存活、但在核对时被证明的。只凭记录、归属读不到或不成立的，不动作，记为 `ownership` 工具错误。
  - 身份核对与信号之间的 PID 复用没有构造，也没有解决；归属依据是受控原型下的标记继承（Unix）与 Job 成员（Windows），不是对任意宿主的保证。
  - macOS 上一次重新普查读不到遗留进程的环境（EINVAL，§2.10），原因没有查明。修复后保险清理不再依赖重读已认出的进程；被测路径的普查遇到同样情况仍判为普查错误（保守）。
- **`-race` 只在 Linux**；macOS 与 Windows 的原型没有开启竞态检测。
