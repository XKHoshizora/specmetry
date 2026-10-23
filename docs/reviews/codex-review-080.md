# Codex Review 080：PR #33 保险清理归属修复的定向独立复审

## 1．结论分层

**A．Review 079 B2 可以关闭；原型 PR #33 技术 GO。**

修复已经把“见证声称某身份存在”与“独立证明该身份属于本轮检查”分开。仅由记录提供的存活身份，必须通过 `ownedNow`，才可进入补救及保险清理共用的动作清单；归属不成立或无法读取时，拒绝动作、记录不可豁免的 `ownership` 工具错误，并保留未知的最终状态。此前可信普查取得的身份仍可传给保险清理，在动作前复核启动身份，不要求再次读取每个对象的环境。这个来源边界符合 Review 079 的最小修正，也保留了 macOS EINVAL 处置的正确方向。

本轮独立复验了旧门禁的两个错误动作清单、新门禁的三个真实组合、全部 Windows 门禁回归及直接判定测试，并额外在真实外层调用链注入归属读取错误。结合 Unix 源码和三平台完整诊断日志，未发现新的直接 BLOCKER。B1-R1-S、B1-R2-W-R 等既有关闭结论保持。

**B．当前固定对象的 CI 合并条件已经满足。** 最终 HEAD 的 run `39979587006` 为 attempt 1、completed/success、8/8；修复代码的常规 CI 与三平台原型诊断也均正式成功。可以建议负责人合并本原型 PR。上轮 `39968755313` 的 SQLITE_BUSY 失败继续保留，未重跑；本次绿色不证明其根因消失，也不关闭 M-15。

**C．冻结与实施条件没有因此满足。** 本结论仅支持 test-only 原型及其门禁、清理在已声明前提下的交付。T1～T6 仍只得到局部支持，14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收。没有新增接受风险；M-14、M-15、M-16 保持未解决。

## 2．固定身份、范围与工作区

| 对象 | 固定身份 |
|---|---|
| PR | [XKHoshizora/specmetry#33](https://github.com/XKHoshizora/specmetry/pull/33) |
| base | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2` |
| Review 079 对象 | `8910825c8145f7eea0cec6451ff8a84b885e48ab` |
| 回归先行 | `f1b8566bde691d738b804c4defd129605367c14e` |
| 修复代码 | `8a2b97ccd521c6af587baf4e3f45421d68dafcd9` |
| 最终 HEAD | `06213169f4f5cbe9758943a64eeebd2447f748e4` |

开始核验为 **2026-10-23 15:55:26 UTC**；完成技术核查时于 **16:06:41 UTC** 再次核验，报告落盘后于 **16:11:10 UTC** 最终复核。远程 HEAD/base 均与上表一致，PR open、未合并、`auto_merge=null`。完成技术核查时 `mergeable=true`、`mergeable_state=clean`，最终复核时常规 CI 仍正式成功。没有扩大到其他提交。

开始时本地 HEAD 为目标，工作区干净，目标报告不存在。读取了适用 AGENTS、完整 [Review 079](codex-review-079.md)、本轮源码及文档差异、[原型文档](../wp2.5-prototype.md) §2.11 和 [处理记录](../review-resolution.md) §94。编译、运行和注入均在仓库外 `review080` 隔离副本中进行，没有切换、重置或清理原工作区。

独立核对相对上轮 **13 文件，+905/−74**：

- 六份文档：AGENTS、implementation-plan、review-resolution、新增 Review 079、check-runner 草案头部、prototype 结果文档。
- 七份原型文件：`census_darwin_test.go`、`census_linux_test.go`、`gate_combo_test.go`、`gate_test.go`、`witness_check_test.go`、`witness_unix_test.go`、`witness_windows_test.go`。
- `8a2b97c → 0621316` 只有五份文档变化。两者的 `prototype/wp25` 子树均为 `e70f83a0be9492805fbee8e692382ae7ee1c5cb7`。
- 相对上轮，生产代码、Schema、迁移、根模块依赖、`ci.yml`、原型 workflow 均未变；check-runner 草案只更新头部状态，冻结契约没有变化。

提交顺序为 `325c646` 归档、`f1b8566` 回归先行、`8a2b97c` 修复、`8aeac2e` 结果文档、`0621316` 状态同步。Review 079 文件 SHA-256 为 `6e86a9ad1dd40107984a6dccde917b8c7d3d9dd8a212a700b59eb5d47c5bb405`，与 §94 登记一致。review-resolution 的本轮变化仅追加 110 行；历史审查原文和 §84～§93 没有被重写。

## 3．B2：候选、归属与动作链

以下位置均绑定本轮固定 HEAD；链接为仓库相对链接。

| 环节 | 实际规则与证据 | 裁定 |
|---|---|---|
| 候选来源 | [gate_test.go:507](../../prototype/wp25/gate_test.go#L507) 先独立普查，再读取见证；[535 行](../../prototype/wp25/gate_test.go#L535) 只把记录确认存活、但不在普查中的对象收集为 `claimed`。 | 身份存在没有直接成为动作授权。 |
| 候选分拣 | [sortCandidates:384](../../prototype/wp25/gate_test.go#L384) 只直接接受普查结果；每个 `claimed` 都调用真实归属检查。false 或 error 均进入 `refused` 和 `ownership` 问题，不能进入 `act`。 | 解决旧 `known` 混合来源的缺陷。 |
| Unix 归属 | [ownedNow:156](../../prototype/wp25/witness_unix_test.go#L156) 核对进程状态、启动身份、非僵尸，再读取本轮标记；读取失败返回错误。Linux 使用 `/proc/.../environ`，macOS 使用 `kern.procargs2`。 | 独立于见证；成立依赖受控原型的唯一标记继承和可读性。 |
| Windows 归属 | [ownedNow:263](../../prototype/wp25/witness_windows_test.go#L263) 经 `openRecorded` 核对真实创建身份，再以同一打开句柄调用 `IsProcessInJob`，最后关闭句柄。 | 准确身份与属于本轮 Job 分开验证。 |
| 清单传递 | [gate_test.go:557](../../prototype/wp25/gate_test.go#L557) 接入分拣；[585 行](../../prototype/wp25/gate_test.go#L585) 的 `known` 仅为 `act`，交给被测补救和 [671 行](../../prototype/wp25/gate_test.go#L671) 的保险清理。没有第二个把纯见证身份追加回去的入口。 | “曾被认出”可以追溯到普查或本次独立证明。 |
| Unix 动作 | [remediate:125](../../prototype/wp25/witness_unix_test.go#L125) 动作前仍查启动身份和标记；[insure:179](../../prototype/wp25/witness_unix_test.go#L179) 对此前可信清单查启动身份，并加入新普查确认的对象。两者的致命信号均经 `outerSignal`。 | 保留可信旧普查结果，未恢复旧的纯见证授权。 |
| Windows 动作 | [remediate:209](../../prototype/wp25/witness_windows_test.go#L209)、[insure:234](../../prototype/wp25/witness_windows_test.go#L234) 实际终止始终只针对本轮 Job，不按见证 PID 终止任意进程。 | 候选分拣与平台动作边界一致。 |
| 错误与终判 | [toolClasses:372](../../prototype/wp25/gate_test.go#L372) 将 ownership 列为工具故障；[judge:415](../../prototype/wp25/gate_test.go#L415) 先检查工具类，再检查预期/允许类别；[628 行](../../prototype/wp25/gate_test.go#L628) 令拒绝归属保留 unknown。 | 预期拒绝、补救成功和保险清理成功均不能豁免它。 |

“无效见证中的 PID 不得操作”现在有两层含义：仅有见证不能授权；若独立普查或当前归属检查另外证明该对象确属本轮，仍可以负责收尾，见证无效本身不会迫使工具遗留已知自有对象。这与最小修正一致，并非简单关闭所有失败路径的清理。

本轮还追踪了外层直接动作：内层启动/超时失败时对自己持有的 `exec.Cmd` 结束和等待；组合夹具由创建它的组合单独结束；见证启动失败对本次直接创建的子进程收尾。它们没有改成接受见证提供的操作对象。Unix 的 SIGUSR1 带外通报不是保险清理授权。未发现绕开新清单的外层致命信号入口。

### 3.1 可信旧普查与剩余窗口

Linux 普查同时检查用户、起始时间、环境标记及读取前后的身份；macOS 普查结合用户、起始时间、`kern.procargs2` 和再次状态读取。只由这些结果、或独立 `ownedNow` 证明得到的身份，才可复用到保险清理。先前 macOS 重新读取环境遇到 EINVAL，不要求因此机械撤回已经取得的归属证明；新一轮普查或独立归属读取失败仍须保守报告。

**本修复没有解决 Unix 身份复核到按 PID 发信号之间的原子性窗口，也没有动态构造 PID 复用。** 标记是本隔离原型受控继承的归属前提，不是任意宿主上的身份认证或对恶意环境伪造的完整防线。Windows 的本轮 Job 限制也不能替代其他平台的证明。文档 §2.11、§13 正确保留这些边界；本结论没有新增豁免。

## 4．真实回归：旧动作清单与新拒绝路径

[gate_combo_test.go:144](../../prototype/wp25/gate_combo_test.go#L144) 创建并持有独立夹具，在真实内层结束后、外层读取见证前，注入格式合法的 `created` 记录。夹具没有继承本次内层标记，也不在外层为内层建立的 Job 中。记录包含其准确身份；`forge-stale` 则只改变记录中的启动身份。

这条注入没有直接改写内层 verdict 或组合 PASS。组合核对被测自检的多个维度、清单是否包含夹具、信号意图、夹具当时存活及组合自己的最终收尾。没有以“夹具仍活着”单独证明拒绝。

### 4.1 本轮 Windows 隔离复验

在 `f1b8566` 与固定目标的仓库外副本中，使用同一现有 Go 工具编译后运行真实进程：

| 场景 | 旧门禁：回归先行提交 | 修复后固定目标 |
|---|---|---|
| forge-owner | 夹具进入可动作清单；被测路径错误报 remedy=succeeded、final=ended；组合 FAIL。 | 夹具在 refused，未进入动作清单；ownership 工具错误、final=unknown；夹具仍活着，由组合结束；组合 PASS。 |
| forge-owner-mixed | 夹具和真实自有遗留一起进入动作清单；组合 FAIL。 | 动作清单只含真实自有遗留；补救实际成功，夹具被拒绝，final=unknown；组合 PASS。 |
| forge-stale | 记录身份变化，没有成为动作候选；组合 PASS。 | 同样不动作；夹具真实身份仍存活，后由组合结束；组合 PASS。 |

旧版运行 exit 1，13 个组合中恰好两个 forge-owner 组合失败，其他 11 个通过；正常排空对照 confirmed。新版运行 exit 0，13 个组合全部通过。旧 Windows 不会真的向夹具发 PID 信号：其 Job 限制已经阻止终止外部对象；本机证明的是共用清单的旧错误/新拒绝，不能冒称 Windows 动态复现了 Unix 的误杀。

`forge-stale` 的 ended 是对记录所指的旧身份的判断，不是宣称当前夹具进程已经结束。组合另外检查其真实身份仍存活，再由自己结束它。见证不一致仍使被测自检失败。

### 4.2 Unix 信号接缝及正式诊断

已读取回归先行和修复后三平台的完整诊断日志。去除 Results 步骤的重复打印后：

| 平台 | 旧 forge-owner / mixed | 新 forge-owner / mixed |
|---|---|---|
| Ubuntu | 夹具进入动作清单；401 / 400 次信号意图；两个组合 FAIL。 | 拒绝夹具；0 / 0 次信号意图；两个组合 PASS。 |
| macOS | 夹具进入动作清单；423 / 435 次信号意图；两个组合 FAIL。 | 拒绝夹具；0 / 0 次信号意图；两个组合 PASS。 |
| Windows | 夹具进入动作清单；实际入口只操作 Job；两个组合 FAIL。 | 夹具未进入动作清单；两个组合 PASS。 |

上述 Unix 信号意图均由接缝记录，没有向夹具实际发送。修复后的每个平台均记录夹具核验期间存活、随后由组合结束。mixed 的真实遗留具有不同身份，确实进入动作清单并被补救结束；保险清理随后记录 nothing left。

[recordSignals:234](../../prototype/wp25/witness_unix_test.go#L234) 只拦截组合自己的那个夹具 PID，对其他对象继续调用原来的真实信号函数，结束后恢复；当前测试没有 `t.Parallel` 并行使用这一全局接缝。因而没有通过全局屏蔽信号，让“自有遗留已清理”虚假成立。Windows 的零信号计数是平台实现的“不按 PID 发信号”，不是对 Unix 的动态证明。

这些 Unix/macOS 结果是本轮独立核验的 CI 日志证据，不是本机复现。作者 WSL 的 477 / 471 次旧信号意图、M31～M34 变异只作为作者辅助证据，没有计入本轮独立实验。

## 5．归属读不到、终判与直接回归

### 5.1 固定表与真实调用链

[TestCandidates:191](../../prototype/wp25/witness_check_test.go#L191) 的五行固定期望覆盖普查对象、额外证明成功、归属不成立、归属读取失败和混合清单。期望动作/拒绝清单由表给出，不由分拣器自己计算；还把拒绝错误送入真实 `judge`，确认预期的 left 拒绝不能豁免 ownership。

表测试本身不证明原生 `/proc`、sysctl 或 Job 查询故障已发生。因此本轮在仓库外 Windows 副本额外注入一次**真实查询之后的错误返回**：`ownedNow` 先实际打开/核对夹具身份并调用 `IsProcessInJob`；查询得到不属于本轮 Job 后，打印命中标记，再返回模拟读取错误。没有改变候选、终判、信号入口或组合期望。

两项 owner 组合均实际命中，产生 `ownership: ... cannot be shown ... (review080 injected: ownership cannot be read)`；被测自检 pass=false、final=unknown，夹具未进入动作清单、仍存活并被组合结束。mixed 的真实自有遗留照常补救。全部 13 个组合和正常排空对照通过，测试进程 exit 0。

这支持“错误沿真实调用链不可豁免地传播”，**不是原生 Windows 查询失败或 Unix 环境读取失败的动态复现**；没有把模拟故障改名为原生证据。

### 5.2 既有终判没有被放松

本轮本机与 CI 均确认此前十个组合仍运行：补救失败/未确认、未知状态、额外证据问题、未命中预期、健康及健康中观察失败分别保留各自判定。组合因正确检出被测自检 FAIL 而 PASS，不等于该自检成功，也不计为机制 confirmed。

`ownership` 先于 expected/also 分支处理，类别使用完整匹配。已证明归属的对象清理成功不会清除拒绝清单或工具错误；保险清理成功不能恢复被测 PASS，保险清理矛盾仍能推翻原有 ended。新增 `Final=unknown` 还避免 Windows Job 已空后把无法授权的记录身份伪装成已经结束。

未发现对 B1-R1-S 的观察错误累积、B1-R2-W-R 的组合终判、短链/损坏链、见证故障或原有健康对照的直接回退。Linux 特有路径的本轮运行证据来自正式诊断，本机未运行 Linux。

## 6．本轮实际运行、失败及验证边界

平台：Windows 11 build 26300、amd64；现有 Go `1.27.1`。没有安装或升级工具。使用固定 Git 对象的仓库外副本，`GOTOOLCHAIN=local`、`GOPROXY=off` 编译；编译均成功。

| 独立运行 | 硬时限 | 结果 |
|---|---|---|
| 回归先行 f1b8566：TestGateCombos、TestPipeNormalDrain | 120 s | exit 1；13 组合中两个 owner 组合按预期失败，stale 和原十项通过；正常排空 confirmed。 |
| 固定目标：TestGate、TestGateCombos、TestCandidates、TestGateJudge、TestWitnessJudge、TestRecordAlive、TestStartPoints、TestPipeNormalDrain | 180 s | exit 0；32 项 Windows 门禁回归、13 组合及直接测试通过；一项外层机制对照 confirmed。TestGate 用时 95.47 s，组合 12.94 s。 |
| 真实归属检查返回错误的定向注入：13 组合与正常排空 | 60 s | exit 0；两个 owner 命中标记、被测自检失败且保留 unknown，无误动作；其他对照通过。 |

没有为绿色重跑失败场景，也没有把旧版 exit 1 算作产品修复失败或机制反例。内层预期 invalid/非零与外层门禁 PASS 分开读取；外层 summary 的 invalid=0 不表示没有执行这些内层负向场景。本机没有重跑全部 29 项 Windows 机制场景，不能将一项正常排空对照写成全部 T1～T6 通过。

所有场景只操作本轮创建/持有或按本轮归属证明识别的对象。组合的 fixture_alive、fixture_ended、混合自有清单、Job 空状态与保险结果均核验。结束盘点首次使用 CIM 被当前环境拒绝，该查询无效；其后以成功的 `Get-Process` 枚举复核，本轮 `review080*` 测试/helper 进程为 0。没有根据失败的 CIM 查询宣称无残留，没有终止未知 PID。

Linux/macOS 本机动态复验未执行；没有据 Windows 结果外推 Unix 信号行为。没有制造原生权限/I/O 错误、强制 PID 复用、不可中断睡眠或任意宿主干预；相应前提和未验证项继续保留。故障注入副本已恢复原源码，原仓库未受影响。

隔离日志 SHA-256（日志留在仓库外，不新增入库文件）：

| 日志 | SHA-256 |
|---|---|
| old-run.log | `959fb9802e2ecb98d54e2ed7bfe36bbac829e89ee402f854186b16f54889ad71` |
| fixed-run.log | `9d0f573a4e196b363163e854a305a0827a57d2a746c612780942644c4b91629d` |
| owned-error-run.log | `866597f4a0580abb602e91a53da9cf552f1abc170a0a464c513ddec28cf14091` |

## 7．CI 正式身份、日志与适用性

通过 GitHub run、job API、完整 job 日志和 commit API 分别核验；不是只采信 PR 正文。下表均为 attempt 1。

| Run | 事件 / HEAD | 正式结论 | 适用范围 |
|---|---|---|---|
| [39976843072](https://github.com/XKHoshizora/specmetry/actions/runs/39976843072) | push / f1b8566 | completed/failure，三平台 Prototype 步骤均失败 | 回归先行。各平台只有两个 owner 组合失败；其他场景/门禁保持原有结果。 |
| [39977737240](https://github.com/XKHoshizora/specmetry/actions/runs/39977737240) | push / 8a2b97c | completed/success，3/3 | 修复代码原型诊断。各 Job 实际检出代码提交本身。 |
| [39977743051](https://github.com/XKHoshizora/specmetry/actions/runs/39977743051) | pull_request / 8a2b97c | completed/success，8/8 | 修复代码常规 CI，检出合并快照 `170d53c8210f8c638ccec2a8e2806e50df329cc7`。 |
| [39979587006](https://github.com/XKHoshizora/specmetry/actions/runs/39979587006) | pull_request / 0621316 | completed/success，8/8 | 最终固定对象常规 CI，检出 `d3841c7920e076f1af83e136f5456377066a3b2f`。 |

### 7.1 合并快照与内容树

- 最终合并快照的两个父提交准确为固定 base 和最终 HEAD；其树及最终 HEAD 树均为 `e55e4f41d76a35fb59f755fb514a1421f0932e11`。
- 修复代码 CI 快照 `170d53c` 的两个父提交为固定 base 和 `8a2b97c`；其树与代码提交树均为 `080c17d1529530847380f016adaff49b46bf4b50`。
- 两次常规 CI 的各八个 Job 完整日志中的实际 checkout 均分别指向上述快照；没有把 run 元数据的 PR HEAD 当作直接检出对象。
- 原型诊断检出 `8a2b97c` 本身，最终 HEAD 只加文档且原型子树相同，因此该代码诊断适用于最终原型代码；不能把最终常规 CI 当成额外一次原型运行。

### 7.2 原型：唯一计数与平台集合

修复后诊断各 Job：Ubuntu `120496651882`、macOS `120496651878`、Windows `120496651688`。均核对 Vet、Prototype 步骤与完整结果行：

| 平台 | 唯一机制场景 | confirmed | 门禁回归 PASS | 组合 PASS |
|---|---:|---:|---:|---:|
| Ubuntu | 38 | 38 | 36 | 13 |
| macOS | 27 | 27 | 27 | 13 |
| Windows | 29 | 29 | 32 | 13 |

场景 ID 并集为 **53**，没有把三平台计数相加。Results 步骤重复打印的结果去重；门禁、组合、直接判定表均不计入机制场景。`TestCandidates`、`TestGateJudge`、`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台均实际 PASS。

两个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak` 在外层普通枚举时按设计 SKIP；相应门禁/组合确实另启内层，日志含对应 invalid/非零、自述和结束观察。SKIP 没有当作直接通过，也没有因外层汇总成功忽略内层结果。

Ubuntu 原型执行的 Linux 分支设置 `race=-race`，运行独立模块的 `go test -count=1 -v $race -timeout 15m ./...`；macOS/Windows 未开原型竞态检测。正式日志的平台为 Go 1.27.3，Ubuntu kernel 6.17.0-1022-azure/amd64、macOS 26.6.2（Darwin 25.6.0）/arm64、Windows build 26100/amd64。不能外推其他版本、原生故障或生产接入。

### 7.3 最终常规 CI

八个 Job 全部正式 completed/success：

| Job | ID |
|---|---|
| Web | `120503042952` |
| Go tests macOS | `120503043213` |
| Go tests Ubuntu | `120503043253` |
| Go tests Windows | `120503043353` |
| Go lint | `120503043368` |
| Build with dashboard Windows | `120503209402` |
| Build with dashboard Ubuntu | `120503209500` |
| Build with dashboard macOS | `120503209674` |

常规 CI 没有运行原型模块。Ubuntu 普通测试使用 `-race`；随后单独编译存储测试二进制，以 sudo 运行八项 `TestCrossUser*`，八项实际 PASS。root 步骤不是普通 `-race` 步骤，也不是跨账户原型归属实验。Windows/macOS 的 root 步骤按平台 skipped。

Windows 按已有动态分组执行根模块其他包与 `internal/cli`；CLI 保留 `-timeout 20m`。真实旧 Git、真实会隐式取对象的 Git 夹具、Windows 权限语义不适用项和 opt-in M-14 诊断均存在设计跳过；没有把这些跳过写成通过。当前存储测试成功只证明本次运行，不能关闭历史维护事项。

### 7.4 失败、取消与作者本地证据

- 归档常规 run [39976810964](https://github.com/XKHoshizora/specmetry/actions/runs/39976810964)（325c646）和回归先行常规 run [39976851878](https://github.com/XKHoshizora/specmetry/actions/runs/39976851878)（f1b8566）均正式 cancelled。后者部分 Job 成功、Ubuntu/Windows Go Job cancelled，不是整次 8/8。保留并发取消的历史登记，不用最终绿色改写。
- 上轮 [39968755313](https://github.com/XKHoshizora/specmetry/actions/runs/39968755313) 仍为 attempt 1、completed/failure。再次核对 Windows job `120466046999` 的日志：`TestSeqIsUniqueAcrossProcesses`，`store_test.go:571`，`database is locked (5) (SQLITE_BUSY)`。未重跑。错误特征与 M-15 相似，不能据路径未变或新测试通过认定唯一根因、绝对无关或已经解决；也不能把它等同 M-14 的版本探测/READONLY_RECOVERY。
- 作者对 8a2b97c 的本机 Windows 完整运行仍保留 **28 confirmed、1 refuted**：N6 的阶段 D 为 **999.5519 ms**，未重跑。本轮独立定向测试没有运行该计时场景，不能称本机验证已替代其失败。
- 作者 WSL2 的 37 confirmed、1 unsupported（缺少 pwsh）和 M31～M34 变异，只是作者本地证据。特别是 M33 在 Windows 未检出，因为 Windows 保险只终止 Job、不使用该清单；这个差异如实保留。

本轮没有触发、重跑或取消任何 workflow。

## 8．BLOCKER、NON-BLOCKER、产品与风险

**BLOCKER：无新增；Review 079 B2 关闭。** 修复与真实回归满足其最小要求，未发现本轮确定的绿色假通过或非本轮对象动作路径。结论限定在本原型声明的对象来源与受控继承/Job 前提内。

**NON-BLOCKER：**

- **N4 保持未修复、非阻塞**：epipe 报告失败仍可能被分为 refuted 而不是 invalid；没有本轮新增 confirmed/绿色放行的证据。建议独立传递报告工具错误并增加分类回归。
- **N6 保持未修复、非阻塞**：Windows 截止/计时口径及严格边界比较导致随机拒绝的既有裁定不变。建议统一时钟定义或明确分辨率规则，保留边界回归；不能只为绿色加任意容差。
- **N5、N7 保持已关闭的既有范围**：shortfall 仍是数量缺额，不是身份匹配/结束证明；当前与历史证据仍按 SHA、运行区分。没有直接回退。
- 归属读取故障的原生动态验证、Unix PID 复用窗口和任意宿主适用性仍未验证/未解决，已在上述证据边界内明确，不被本轮改写为代码已经阻止。

没有新的非阻塞维护任务需要借本轮扩大原型范围；前述既有事项继续独立登记。

**PRODUCT DECISION：没有新增待决项来代替 B2 修复。** 原有 **D1～D9、D11、D13、D18、D19、D20，共 14 项**仍待负责人选择。C-4 属于 D20 的宿主后续调用支持契约；C-19/C-20 的原型技术规则没有自动批准生产 F1/退出码或其他平台的收养归属。

**ACCEPTED RISK：本轮没有新增。** WP2.3 方案 B 的既有局部边界保持原状，未放宽 Log 完整性、其他账户保护或本原型禁止误操作无关对象的要求。未解决问题和实验前提不被自动归类为接受风险。

## 9．冻结前与生产验收前分别还缺什么

本次门禁与清理的技术 GO，不等于 T1～T6 全部完成。

**冻结/实现授权之前：**

1. 负责人决定上述 14 项产品选择，明确进程树保证范围、O3 后宿主策略、输出支持、记录/取消和解释器行为，再同步正式规范、契约与验收。
2. 收口 T1/T6 的目的端、独立预算、O3 资源归属与后续调用；T2 的生产闸门全局拓扑、启动/回收能力的 C-9 规则设计、上下文 Git 与排他区间的顺序、取消和 F1 之后策略。
3. 定清 macOS ABI/枚举错误及预算和支持版本、Windows 父端独立期限/提前失败及所有权移交、T5 解释器来源/实际路径/PATH/worktree 排除。必要隔离原型只需补这些具体主张，不要求为授权先完成生产 Runner。

**获得授权之后、本工作包验收之前：**

- 实际 Runner/CLI、静态规则与闸门接入，以及三平台经生产路径的正反例；原型副本不能替代接线测试。
- 真实终端/重定向/PTY/取消、输出继承与 JSON、F1/O1～O3、故障/短写/提交不确定及草案 §8.6 的持久化叠加。
- 迁移 0009、check-run 契约、配置固定、空批次、首次 worktree/seq、记录/事务、隐私与 dbdump 一致性。上述内容本原型没有验证。

不自动增加无目标压力、强制 PID 复用、生产开发或其他工作包作为本 PR 的阻塞任务。

## 10．最终答复

| 问题 | 本轮结论 |
|---|---|
| A．Review 079 B2 能否关闭？ | **可以。** 纯见证身份不再成为动作授权；真实归属错误不可豁免。 |
| B．PR #33 技术与合并建议？ | **技术 GO；可以建议负责人合并本原型 PR。** 本轮没有自动合并或替负责人授权。 |
| C．最终 CI 条件？ | **当前固定 HEAD 已满足：attempt 1，正式 success，8/8。** 原型诊断另有代码身份一致的三平台成功证据。 |
| D．清理与直接回归？ | 自有对象实际清理与非本轮对象拒绝同时成立；原 B1 的关闭结论保持。Unix 动作窗口、受控标记/Job 前提和原生故障未验证边界如实保留。 |
| E．冻结/生产实施？ | **仍不具备整体冻结与生产开工条件。** 14 项产品选择和具体技术规格尚待收口；生产接入及本层验收仍另行完成。 |

建议接手顺序是：负责人处理原型 PR 合并决定及合并结果核验，再按第九节组织规格/产品决定；本报告不授权生产实施，也不宣布 WP2.5 或 M2 验收。M-14、M-15、M-16 均保持未解决。

本报告为独立审查原文，仅新增本地未提交文件；未修改其他原仓库文件，未提交、推送、合并、改基或重跑 CI。
