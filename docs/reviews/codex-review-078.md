# Codex Review 078：PR #33 的共享观察链、见证与外层盘点定向复审

## 1．结论

**PR #33 技术 NO-GO；目前不建议合并。**

- **B1-R1-S 可以关闭**：两项精确反例及本轮追踪的同源观察入口，已经把工具观察错误保留到内层终判；后续成功不恢复实验有效性。
- **B1-R2-W 的原始精确反例已经修复，但该项尚不能整体关闭**：创建记录、回执、严格解析、根记录、封条、独立普查与 Windows Job 计数提供了实质改进。不过，新增外层判定只要找到预期拒绝类别就返回成功，会同时忽略外层自身的观察、命中或补救失败。下文 B1-R2-W-R 给出真实内层进程链上的独立故障注入反例。
- **Review 075 B1 尚不能整体关闭。** 这里发现的是实验有效性门禁的残余，不是已经证明生产 Runner 会误杀、漏杀或产生错误交付事实。
- 现有三平台诊断的成功、正常场景的观察与已修复的原反例均予保留。它们不能证明“发现预期故障之后，外层工具自身再失败也必然使整次诊断失败”。

本报告不批准 14 项产品选择，不冻结规范，不授权生产实现，不宣布 WP2.5 或 M2 验收。M-14、M-15、M-16 保持未解决。

## 2．固定身份、范围与工作区

| 对象 | 本轮使用的完整身份 |
|---|---|
| PR | [XKHoshizora/specmetry#33](https://github.com/XKHoshizora/specmetry/pull/33) |
| base | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2` |
| 上轮对象 | `acf699b1df18343bf97e734fdb2a3af379637fbf` |
| 回归先行 | `8f42e58f43a31b433c540c8146fc3873dec3aa2c` |
| 修复代码 | `e77af946291862191759d3af6e94e372221748a2` |
| 本轮 HEAD | `f82dfb8fad7f6078af663d6cb68a8073b9f97169` |

审查开始和结束均通过远程 PR API 核对：HEAD/base 与上表相同，PR 为 open、未合并，`auto_merge=null`。完成核查时为 **2026-10-23 13:31:56 UTC**；报告落盘后于 **13:37:02 UTC** 再次核对，身份和状态仍一致。没有把其他提交纳入结论。

原工作区开始时干净，HEAD 为固定目标；完成核查、写报告之前再次确认干净且目标报告不存在。所有编译、运行和变异均在仓库外的 `review078` 隔离目录。原仓库只新增本报告，不改动其他任务的内容。

已完整阅读适用 AGENTS、[Review 077](codex-review-077.md)、[原型文档](../wp2.5-prototype.md) §2.9 两张共享链表、§3、§9～§13，以及 [review-resolution](../review-resolution.md) §92，并追踪本轮源码增量和必要调用方。

独立核对相对上轮的 **24 个文件，+2433/−278**：6 个文档文件、18 个原型文件。其中：

- 文档：AGENTS、implementation-plan、review-resolution、新增 Review 077、check-runner 草案头部、prototype 结果文档。
- 原型：adapters；新增两个 census 平台文件；exclusive_linux、gate、helper、helper_unix、job_windows、main、report；四个 tree 平台/共享文件；witness_check、witness 及两个 witness 平台文件。
- `e77af94 → f82dfb8` 只有五个文档文件变化。生产代码、Schema、迁移、根模块依赖、`ci.yml`、原型 workflow 均没有本轮增量。
- base 是目标的祖先；本轮没有改变已冻结契约或已验收模块。

五个提交分别是：`d3c9cb5` 归档、`8f42e58` 回归先行、`e77af94` 修复、`20486a3` 结果文档、`f82dfb8` 状态同步。Review 077 归档的 SHA-256 为 `c3d4b31072f44d39e06f06e07305b77ad7fcf12b805b59432582efb8abb3395b`，与登记一致；§92 只是追加，§84～§91 没有改写，原审查对象和 NO-GO 没有被后续状态替换。

## 3．BLOCKER

### B1-R2-W-R：预期拒绝类别会掩盖外层自己的补救或观察失败

**位置，均绑定本轮 HEAD：**

- [gate_test.go:312](../../prototype/wp25/gate_test.go#L312)，`judged`，尤其 320～323 行。
- 同文件 449～457 行记录补救失败与 `still-alive`；471～473 行记录未命中的注入；496～507 行把 `judged` 的结果作为门禁 PASS。

**预期：** 一项负向自检可以因正确发现它指定的故障而通过；但外层必须完成自身必需的观察、确认自有进程收尾，并确认注入命中。外层工具本身失败，不能因为列表里也存在一个预期的 `left:` 或 `witness-malformed:` 而得到绿色诊断。

**实际：** `c.reject != ""` 时，`judged` 找到任意一个匹配的前缀就立即返回 true。它不要求其他外层检查成功。于是同一个 `Problems` 可以包含预期故障和 `still-alive:`，甚至外层的 `census:`、`state-unknown:`、`hit:`，仍被接受。普通正对照要求 Problems 为空，负向自检却没有对应的基础设施成功门禁。

**独立有界复现：** 在固定目标的 Windows 隔离副本中，只在外层 `innerProc.remediate` 的非空负载分支注入一次有意义的失败结果：

```go
if len(ss) > 0 {
    return ss, errors.New("review078: remediation could not confirm its end")
}
```

这不修改内层 verdict、不修改见证、不修改最终判定函数；真实内层照常启动 helper，外层照常取得存活列表。保留专属外层 Job 的关闭路径，以便实验结束时结束本轮进程。运行真实场景、两个故意遗留回归及健康正对照：

```text
review078-mut.test.exe
  -test.run=^(TestPipeNormalDrain|TestGate)$/(witness.leak-unreceipted|witness.leak-unrecorded|control.normal-drain)$
  -test.v -test.count=1 -test.timeout=30s
```

观测结果：

| 项目 | 真实内层 | 补救前 | 外层补救结果 | 最终自检 |
|---|---|---|---|---|
| 健康 normal-drain | confirmed / 0 | 无残留 | 不需要补救 | PASS |
| leak-unreceipted | invalid / 1 | helper `33452` 存活 | 返回错误；`still_alive=[33452]` | **PASS** |
| leak-unrecorded | invalid / 1 | helper `67040` 存活，且无记录 | 返回错误；`still_alive=[67040]` | **PASS** |

两个负向自检均同时打印：

```text
left: ... alive when it was over
still-alive: the remediation could not confirm its end: review078: ...
still-alive: 1 processes still alive after the remediation
pass: true
```

最终 `gate_checks=3, gate_passed=3, gate_failed=0`，整个测试进程 **exit 0**。这不是只运行 `judged` 的单元反例，也不是只看到内层 invalid 就认为外层正确。实验确实走过真实 Start、内层退出、外层普查、故障补救和最终汇总。

专属外层 Job 随后关闭；本轮核查上述 PID 及隔离目录下的进程均已不存在。这里的故障是可控注入，**没有声称本机实际发生了原生 TerminateJobObject 故障**。它证明的是当前外层一旦获得这种合法的失败返回，确实会把它吞成绿色；最终 Job 关闭后的实际结束也不能倒推判定时已经确认收尾。

**影响：** 本轮新加入、作为 B1 关闭依据的外层工具在自己的关键失败分支上仍可假通过。CI 的健康运行没有触发此分支，因此此次发现不否认所记录的正常观察；但“预期故障被发现”还不能支撑外层有效性和清理保证。

**最小修正：** 将不可豁免的外层工具状态与预期故障类别分开判定。普查/身份观察失败、补救错误或仍存活、必需注入未命中等应先导致自检失败；只有这些门禁成功后，才判断指定负向故障是否被发现。允许故意遗留产生合理的连带问题（如 `unrecorded`、`unaccounted`、`witness-count`），不必粗暴要求 Problems 只有一项。增加“预期类别已存在，同时外层失败”的真实链路回归与正常正对照；同步 §2.9 对外层终判的说明。无需扩大为生产 Runner 或全部机制重审。

## 4．B1-R1-S：共享观察链的裁定

**本项可以关闭，限本轮原型门禁及已追踪调用链。**

| 入口 | 源码核查与证据 | 裁定边界 |
|---|---|---|
| pidfd 绑定后的第二次状态读取 | `sweep` 的 unknown 分支调用 `failed` 保存首错，关闭 fd，不按该读数发信号；之后回收成功也不清除 `obsErr` | 原精确反例关闭 |
| 场景起点子进程列表 | `ownChildren` 返回列表和错误，`begin` 保存 `baseErr`，`done` 据此 invalid；不再用 nil 冒充空基线 | 原精确反例关闭 |
| 清扫 poll、waitid | 非 EINTR 错误进入 `obsErr`；waitid 只有 `si_pid` 对应候选才计回收；零结果是尚未回收 | 不把暂未完成当作失败或不存在 |
| subreaper、childCount | 读取失败被 flag；subreaper 不再伪造 0 | 同源修复成立 |
| Unix 状态、父链、stop、Wait4 | unknown 与可靠消失/身份改变分开；必要观察错误进入 flag 或首错，cleanup 最终检查首错 | 结束确认与实验有效性分开 |
| Linux/macOS 退出观察 | poll/kevent 的错误返回；macOS 早退出的 waitid 错误不再写成普通状态字符串；调用方记录失败 | 未把 macOS 日志核验写成本机动态复验 |
| Windows Job 成员、等待、active、endJob | `IsProcessInJob`、句柄等待和 Job 查询的观察错误进入 flag/obsErr；结束后的正常读数不覆盖 | 原生 API 失败与被测接缝故障分开 |

EINTR 的重试重算原截止点，不另发一个完整等待预算；运行中、没有可回收结果和期限到达分别处理。这里不保证操作系统硬实时，也没有独立制造持续信号风暴。

修复前 `8f42e58` 的真实 Windows 场景 `observe.injob-once`，注入命中 1 次，仍是 confirmed/0；修复后的同场景为 invalid/1，注入仍命中 1 次，结束和清理正常。健康 normal-drain 在两版均 confirmed/0。

六个“失败后恢复”注入点的动态证据强度不同：本轮 Windows 独立执行了 `pidProc.ended` 和 Job 成员读取；其余 Linux 点由三平台诊断中的 Linux 完整日志及源码传播支持。没有把六点概括为所有系统调用都经过本轮动态故障注入。被测机制预期的 pidfd 不可用、身份不符、Job 查询接缝失败仍允许按正确响应 confirmed；没有改成一律 invalid。

C-19 的原型错误累积要求获得支持，生产 F1/退出码分类仍依赖未批准的选项。本轮新增 BLOCKER 位于外层最终门禁，不撤销内层以上精确修复。

## 5．B1-R2-W：见证、带外通报与盘点

### 5.1 原精确反例与真实调用链

以下修复成立：

1. `start` 在 Cmd.Start 之后，通过尚持有的子进程/句柄读取身份，写创建者的 `created`。辅助进程在等待到匹配创建记录并成功写 `launched` 后才执行场景工作或派生后代。Windows 挂起进程未能运行 helper 代码，也已由创建者记录。
2. 打开、写入、短写、关闭和身份读取失败，不再静默跳过。创建者失败时结束并等待自己的子进程，helper 的回执失败则返回专用退出码。helper 定时器在 Go 代码开始后才能生效，不是 CREATE_SUSPENDED 的父端兜底；父端仍有独立期限。
3. 根记录绑定外层实际启动的 PID/启动身份；封条核对本进程记录数及失败数。空文件、缺根、缺封条、错误身份、未知事件、重复、截断及旧格式都被拒绝，不再成为空集合成功。
4. 记录与回执按 PID 和启动身份关联；创建者也须有记录。`recordAlive` 只在平台身份合法后区分已结束/不同身份和观察失败，不再将 `unknown` 当成已结束。
5. Windows Job 的累计创建数独立检查已结束但没有记录的对象；Unix 不具备相同的历史计数能力，文档如实保留这个差异。

本轮在 `8f42e58` 和目标的隔离副本独立执行原反例：

| 故障 | 旧版 | 新版 |
|---|---|---|
| helper 写见证真实失败 | 命中 6 次，内层 confirmed/0；外层没有问题，不能得到所要求的拒绝 | 命中 1 次，内层 invalid/1；外层 `witness-raised` |
| helper 身份无法读取 | 命中 6 次，旧门禁接受 unknown 身份，内层 confirmed/0 | 命中 1 次，内层 invalid/1；外层 `witness-raised` |
| 健康 normal-drain | confirmed/0 | confirmed/0 |

旧版的门禁回归本身因此 FAIL/exit 1，恰恰揭示旧工具接受了不可信证据；不是把旧门禁自检失败写成生产故障。新版的正确拒绝是门禁自检 PASS，**不是机制 confirmed**。

### 5.2 带外通报的依据与支持前提

- Unix：外层在启动内层之前安装 SIGUSR1 接收，helper 从受控环境取得外层 PID；一次失败通报不依赖已经失败的见证文件。接收通道跨本测试进程保留，各顺序场景开始时排空，最终读取通报。它不是通用嵌入宿主的信号复用协议，也不证明并行场景之间任意延迟信号都能严格归属。
- Windows：外层逐项创建命名事件，helper 打开并置位，外层等待时将 API 错误保守视为拒绝，结束后关闭事件句柄。创建者记录、根封条和 Job 计数提供另外的检查。
- 通报本身失败不具备无限递归的后备通道，这一点已披露。当前支持范围是受控测试进程、正常同用户 IPC、helper 遵守协议并继承标记；不是抗任意恶意改写的证明。原型中 helper 失败还会停止工作，以专用退出状态沿场景负载检查传播。没有独立复现“这些前提成立，但仅通报丢失便使机制 confirmed”的新反例，不能只因有限信任前提就增设第二个 BLOCKER。
- 本报告也没有将上述前提批准为生产风险豁免。现有具体 BLOCKER 是已可复现的外层终判吞掉已知失败，不需要假设底层通报失效。

### 5.3 普查、归属与补救的顺序

源码顺序是：真实内层 Wait 返回 → 普查 → 按记录逐条读取状态 → 保存 Left/Unrecorded → 补救 → 另记 Still/错误 → 核对见证及登记 → 终判。没有旧的 10 秒补证宽限；通报的 200 ms 等待在这轮存活事实采集之后。这里是代码顺序保证，不是操作系统调度的零延迟保证。

- Linux：每项唯一的临时见证路径作为环境标记；筛选本用户和不早于内层的启动时间，读取状态、属主、环境，再复核身份。ENOENT/ESRCH 与权限/解析等错误分开，身份变化有有限重新核验。未知不应成为空集合。
- macOS：读取同用户进程和启动时间，再读取环境并复核状态；消失/已变成其他身份与无法读取分开。证据来自源码和真实 macOS CI 日志，本机没有执行 macOS。
- Unix 补救只针对已发现的本轮标记且启动身份仍成立的对象；无效见证 PID 不直接变成信号授权。标记可读、受控 helper 不主动清除标记是支持前提；已结束且绕过所有启动记录的历史对象，在 Unix 不能仅凭当前普查还原。
- Windows：外层在恢复内层之前加入专属 Job；普查及累计计数在外层 Job 关闭/终止之前进行。内层自己的 Job 是嵌套子集，没有为外层设置 breakaway 放行，也没有在盘点之前用关闭外层 Job 隐藏残留。嵌套本身改变了门禁下的运行条件，文档已披露；普通机制场景另行运行，不受这个外层 Job 约束。[微软的嵌套 Job 文档](https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs)与[继承/脱离规则](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)支持这里的机制依据，不证明未测试版本上的全部行为。

两项故意遗留在修复后的三平台完整日志及本轮 Windows 独立运行中，均在补救前看到了一个存活对象；没有任何记录的一项由普查发现。随后正常补救的 Still 为空。**这一正对照证明当前运行确实完成补救；不能代替 B1-R2-W-R 所需的补救失败门禁。**

`shortfall` 现在明确是预期登记缺额，包括未创建的预期节点；`unowned` 另列实际创建但未登记的对象及状态。数量容纳不再被描述为逐对象对应或结束证明。Review 077 N5 的术语问题可以关闭。

`TestStartPoints` 扫描当前包实际使用的 Start/Run/Output 等入口，逐行列出当前必要例外；本轮三平台日志和本机均实际执行通过。它不是生产 C-9，也不是对未来任意包装、别名或依赖内部启动的穷尽防御。

## 6．NON-BLOCKER 与计时裁定

### N4：epipe 分类继续保留

本轮没有修复 holder 写 epipe 报告失败后被归为 refuted 的问题。相关生产机制实验和分类代码没有新增将它放行为 confirmed 的变化。继续沿用 Review 076/077 的非阻塞裁定，不用它单独阻止本 PR，也不宣称已经修复。后续最小工作仍是传播报告工具故障并增加分类回归。

### N6：Windows 阶段 D 使用不同计时口径，能够导致随机拒绝

位置：[dest_test.go:245](../../prototype/wp25/dest_test.go#L245)、[clock_windows_test.go:34](../../prototype/wp25/clock_windows_test.go#L34)、[pipe_test.go:137](../../prototype/wp25/pipe_test.go#L137)。

独立核查：截止点/剩余预算使用 `time.Now`/`time.Until` 和 Go timer；报告时长却用显式 QPC stopwatch。安装的 Go 1.27.1 amd64 源码显示 `time.now`、`nanotime1` 读取 `_INTERRUPT_TIME`，与 QPC 是不同口径；[Go 官方对应源码](https://go.dev/src/runtime/time_windows_amd64.s)也体现这个实现方式。因此严格以 QPC 的 `>= 1s` 反查另一时钟的截止条件，存在边界不一致，有源码依据，不能只复述作者解释。

原诊断 Windows 日志确实是 **999.8195 ms**，其它主要预期（O2/O1、停止读取、封闭后写入失败、完整最后文档）成立，单因“phase D used T_drain”得到 refuted。最终诊断是 **1.0004497 s**。本轮本机只做有界的三次正常运行，分别 **1.0007471 s、1.0001121 s、1.0064479 s**，全部 confirmed/0，没有为了得到短值反复施压。

裁定：目前支持“时钟口径和严格比较导致随机拒绝”。按较精细时钟观察到少于 1 秒，不能再无条件说物理时间绝对没有提前；但现有证据也没有证明违反封闭/输出安全的实际提前收尾，更没有证明会产生绿色假通过。没有复原旧 CI 主机上那一次的全部时钟相位，因此“唯一根因已经独立证明”仍过强。

这是**非阻塞的计时可靠性事项**；不因作者称未授权修复就自动放行，也不据此否定已有非计时观察。最小建议是截止和测量统一时钟/定义，或者依据明确的分辨率制定容差，并单独保留边界回归；不能仅为绿色随意放宽。它不解决本报告的外层门禁 BLOCKER。

### N7：结果文档仍有旧运行来源的残留措辞

[原型文档](../wp2.5-prototype.md) 542 行说 §4～§8 数值来自 `39954957954`，但 642 行仍写 §4 数值来自 `39887528435`；629 行的“当前证据以旧 run 为准”也与紧接着的历史降级和新 run 适用说明不够清楚。

这是非阻塞的来源标注问题：本轮已从新 run 的完整日志核实当前场景重新执行，未发现需要靠旧绿色汇总才能成立的这些正常观察。最小修正是在活跃结果文档中明确区分“当时的证据”和“当前重跑证据”，替换 §4 过期的 run 引用；不改写历史审查。

Review 075 N1～N3 保持既有处理结论。没有新增其他确定机制反例或生产算法反例。

## 7．独立验证、失败与清理

本机：**Windows 11，10.0.26300，amd64；Go 1.27.1**。使用现有工具和缓存依赖，没有安装、升级或改动超时设置。固定对象通过 Git archive 导出到仓库外；没有切换原工作区。

| 独立操作 | 结果及适用性 |
|---|---|
| 目标代码编译 | 成功；单独 test-only 模块 |
| 目标：TestPipeNormalDrain、全部 TestGate、TestWitnessJudge、TestRecordAlive、TestStartPoints | exit 0；1 个正常机制 confirmed、32 个门禁回归通过；24 个 WitnessJudge 表用例实际执行，另外两个直接测试通过 |
| 旧 `8f42e58`：normal-drain 及三项精确反例 | 健康对照通过；injob-once、write-fail、identity-unknown 三个门禁回归失败，整个进程 exit 1，失败理由对应原缺陷 |
| 隔离补救故障注入：两个真实遗留回归及 normal-drain | 真实存活与补救错误仍被报告 PASS/exit 0，形成 B1-R2-W-R；没有改最终判定来伪造结果 |
| 未变 escaped-holder，限定三次 | 三次 confirmed/0；没有本机重现旧的短于 1 秒读数，不把未复现当成问题消失 |

24 个 `TestWitnessJudge` 用例的预期是固定表项，包括三个健康正例，不由被检查实现生成；覆盖空/缺根/缺封条、读失败、格式/身份/重复/截断、回执和创建者不一致、带外通报及 Job 计数。它们没有覆盖“预期拒绝类别 + 额外外层工具失败”的组合，这正是新增残余的证据缺口。

第一次准备变异编译时，指定的隔离工作目录尚未创建，执行器在启动命令前拒绝；创建目录后首次实际编译成功。这是审查准备错误，不是原型缺陷，也不计为变异检出。其余上述运行均按记录结果报告，没有隐藏失败后改称首次通过。

隔离实验在自己的 30/60/120 秒测试期限内完成；没有对未知进程发信号。故障注入保留专属外层 Job 的最终释放，正常场景使用自身清理。结束时按本轮可识别路径/PID查询，未发现本轮测试进程和上述已知 helper 残留。该核查不宣称系统中没有其他任务的进程。

Linux/macOS 本轮没有本机动态执行，相关判断是源码、正式 CI 完整日志和必要官方机制依据；没有安装工具或绕过环境限制。作者的 M13～M24 变异属于作者本机证据，本轮没有冒称重做全部变异。没有强制 PID 复用、不可中断睡眠或真实硬件故障实验。

隔离证据保存在仓库外 `review078` 下的 `fixed-run.log`、`old-run.log`、`mutation-run.log`、`timing-run.log`，不作为原仓库额外新增文件。

## 8．CI 的独立核验

通过 run API、job API、各作业实际 checkout 和日志交叉核对；以下均 **attempt 1**。没有重跑或触发 workflow。

| Run | 事件 / HEAD | 正式结果 | 实际检出与适用性 |
|---|---|---|---|
| [39951830770](https://github.com/XKHoshizora/specmetry/actions/runs/39951830770) | push / `8f42e58` | completed/failure，三作业均 failure | 诊断直接检出回归先行代码 |
| [39954957954](https://github.com/XKHoshizora/specmetry/actions/runs/39954957954) | push / `e77af94` | completed/success，三作业均 success | 诊断直接检出修复代码；不等同最终文档 HEAD 的新诊断 |
| [39951833625](https://github.com/XKHoshizora/specmetry/actions/runs/39951833625) | pull_request / `8f42e58` | completed/success，8/8 | 合并快照 `25d548906c94d1df09160fe9ce94c2d06e79e3f7` |
| [39954960926](https://github.com/XKHoshizora/specmetry/actions/runs/39954960926) | pull_request / `e77af94` | completed/success，8/8 | 合并快照 `71d980ed26ef80ea344017c419b4a1b092cc974f` |
| [39956896177](https://github.com/XKHoshizora/specmetry/actions/runs/39956896177) | pull_request / `f82dfb8` | completed/success，8/8 | 合并快照 `716686b4396f380dfff300924df0c3cd154d9679` |

三个常规合并快照的父提交依次均为固定 base 和对应 HEAD；内容树分别为：

```text
25d5489 -> 87ef7222663cb66e9ae7d50b8f659145d8493924
71d980e -> 0853da06d80645b1508a44e2dfce14126ed87a79
716686b -> d38c7c70ad2c5215be21bc8e127f21c23cd522a9
```

每棵树与对应 HEAD 相同。最终八个作业为三平台 Go tests、Go lint、Web、三平台 Build with dashboard，全部正式 success，全部实际检出最终合并快照；不是仅凭 PR 正文或 HEAD 元数据判断。

### 8.1 原型日志的实际计数

| 诊断 | 平台 | 唯一机制场景 | 门禁自检 | 额外事实 |
|---|---|---|---|---|
| 39951830770 | Ubuntu | 38 confirmed | 23/35 PASS，12 FAIL | 四项新增 Linux 观察故障仍为内层 confirmed |
| 同上 | macOS | 27 confirmed | 18/26 PASS，8 FAIL | 旧见证反例确实未被拒绝 |
| 同上 | Windows | 28 confirmed、1 refuted | 21/31 PASS，10 FAIL | refuted 是独立的 999.8195 ms 计时，不是 B1 回归预期失败 |
| 39954957954 | Ubuntu | 38 confirmed | 36/36 PASS | 新观察回归 invalid/1；带 -race |
| 同上 | macOS | 27 confirmed | 27/27 PASS | 真 macOS 26.6.2 / Darwin 25.6.0 arm64 |
| 同上 | Windows | 29 confirmed | 32/32 PASS | Windows 10.0.26100 amd64 |

修复后的三平台均使用 Go 1.27.3；本机 Go 1.27.1 与其版本差异保留。Ubuntu 原型命令是 Linux 条件设置 `race=-race` 后执行实际的 `go test`，不是普通日志中没有竞态文字就推定启用了竞态检测。macOS、Windows 原型未开启 -race。

Results 步骤会重印结果；逐 ID 去重后，三平台场景集合的并集是 **53**，不是把 38+27+29 算成 94 个独立场景。门禁 PASS、内层 verdict 和外层 Problems 独立读取，没有把故障内层 invalid 算成正常机制 confirmed。

`TestWitnessJudge`、`TestRecordAlive`、`TestStartPoints` 三平台均有实际 PASS。顶层唯一两项设计跳过是 `TestInnerForeignPID` 和 `TestInnerLeak`：只供内层运行。对应 Gate 的真实子进程结果、退出码、命中和存活盘点记录证明它们在内层实际执行；顶层 SKIP 不算机制通过。

最终常规 CI 的 Ubuntu 普通 Go 测试带 -race；另行编译运行的 root CrossUser 步骤是独立证据，普通步骤中因非 root 跳过的跨账户用例在那里实际运行。root 步骤不是原型测试，也不是自动带 -race 的同一步。平台专属/权限/真实旧 Git 的既有跳过不计为通过；本轮不据常规 CI 重新验收旧模块。

常规 CI 不运行独立原型模块；原型成功也不证明生产 Runner。旧失败、跳过和作者本地历史均保留，没有把整次旧诊断失败全归因于 B1，亦没有推断 M-14/M-15/M-16 的根因解决。

## 9．PRODUCT DECISION、ACCEPTED RISK 与后续门槛

**PRODUCT DECISION：本轮无新增需负责人代替技术修复作出的选择。** 原有 14 项仍待批准：D1～D9、D11、D13、D18、D19、D20。C-4 的同宿主后续调用策略属于 D20 的待决范围；当前批次停止与宿主以后拒绝/继续不同，单次 CLI 与长驻宿主的代价不同。

**ACCEPTED RISK：无新增。** 既有 WP2.3 方案 B 的局部边界保持原适用范围，不豁免 Log 完整性、误操作无关进程或外层工具已知失败。本轮的 BLOCKER 不能通过将其称为工具边界自动转成已接受风险。

冻结及实现授权之前的最小工作分两类：

1. **本 PR 必需修正：** 关闭 B1-R2-W-R，加入负向故障与外层工具失败叠加的回归，保留健康、原反例、无记录遗留的对照；只恢复受影响证据，不重做所有 T1～T6。
2. **既有冻结门槛：** 产品 14 项选择；候选支持条件与失败协议；Linux 全局闸门拓扑/取消/F1 后宿主资源；C-9 覆盖启动、等待、回收及信号前提的规则设计；macOS ABI/支持版本/枚举错误及预算；Windows 父端期限及尚未覆盖的启动失败规格；T5 来源、实际路径、PATH/worktree 排除与安装布局。计时事项独立收口。必要隔离原型与规格先定清，不要求先完成生产实现才能授权实现。

授权之后、工作包验收之前才是生产 Runner/CLI 接线、全局闸门和生产静态规则实现、迁移 0009、正式 check-run 契约、配置固定/空批次/worktree/seq/事务/取消、§8.6 持久化组合、终端与 JSON 输出、生产隐私及三平台/E2E 测试。本原型没有验证这些内容，不能借门禁修复一并宣布完成。

## 10．交接裁定

| 问题 | 答复 |
|---|---|
| B1-R1-S 能否关闭？ | **可以**，按本报告的原型与证据边界。 |
| B1-R2-W 能否关闭？ | **暂不能整体关闭**；原精确反例已修复，剩余外层终判 B1-R2-W-R 必须处理。 |
| B1 能否整体关闭？ | **不能。** |
| PR #33 技术结论及合并建议？ | **NO-GO，暂不建议合并。** 正式 CI 成功条件已满足，但它没有覆盖已独立复现的外层叠加失败。 |
| 新见证/盘点是否有用？ | **有实质支持。** 必须保留其受控 helper、可读环境、同用户 IPC、Windows 嵌套 Job 及当前启动入口的适用前提；不是生产 C-9 或任意宿主保证。 |
| 计时问题是否另一个 BLOCKER？ | **不是。** 已确认随机拒绝风险及口径差异，未证明输出安全失效；按 N6 独立处理，不关闭或掩盖历史 refuted。 |
| 冻结/产品批准/生产授权？ | **均未发生。** 修正门禁后交独立会话定向核验，再由负责人决定后续；不启动生产 WP2.5。 |

归档说明：本报告是本轮新写的审查原文，仓库文件链接使用从报告位置出发的相对路径；外部来源链接指向实际 PR、run 或官方材料。没有转换或改写历史审查原文。
