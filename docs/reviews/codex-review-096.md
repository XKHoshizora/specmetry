# Codex Review 096 — PR #38 定向独立复审

## 1．结论

**技术 NO-GO，当前不建议合并。** Review 095 点名的 P-4、P-2、P-1 反例已得到针对性修复与新证据支持；但 B1 **不能整体关闭**：P-1 新调用链使用的 Linux 排他区间仍丢弃退出时清除 subreaper 的错误，屏障可以把该失败报告为成功进入，最终保持 confirmed / 0（本报告 B1-R，源码条件反例）。

Review 095 **N1、N2 可以关闭**。P-2 新增的“报告—核验登记—确认—回收”顺序成立，恢复了本轮日志所示的正常组与 SET 0 拒绝组的登记证据；不等于所有失败分支或生产 G6 已穷尽验证。

最终常规 CI **attempt 1 正式 success，8/8**；固定代码诊断三平台正式 success。CI 条件已满足，但不能替代尚未满足的技术合并条件。没有新增产品待决或风险接受；Q-1、Q-2(a) 保持已批准。

| 问题 | 裁定 |
|---|---|
| Review 095 B1 | 原点名路径修复成立；共享退出错误仍有 B1-R，不能整体关闭 |
| N1 | 关闭：第二次准入采样已移到资源释放之后 |
| N2 | 关闭：H 维度、§19 摘要及 data-model 已同步 |
| 新 P-2 登记与门禁变化 | 已执行路径成立；有一项非阻塞的 unsupported 分类边界，见 N-096-2 |
| 固定 PR 是否建议合并 | 否，先修 B1-R；CI 绿色保持有效但不足以解除技术阻塞 |
| G3 / G6 / G7 | 分别取得下文列出的局部支持，不能整体标记满足 |
| G5 / P-3 | G5 未满足，P-3 未执行，不自动缩减平台支持 |

## 2．固定身份、范围与工作区

审查开始核验：**2026-10-28 07:10:12 UTC（16:10:12 JST）**。结束身份核验：**07:31:18 UTC（16:31:18 JST）**。

| 对象 | 实际核验 |
|---|---|
| PR | [XKHoshizora/specmetry#38](https://github.com/XKHoshizora/specmetry/pull/38)，两次均 open、未合并，auto_merge 为 null |
| base/master | `2c5cf23a8c5a025be49b895f353af77b70a6248f` |
| 固定 HEAD | `b80982c2bbf4f7db1c6af179ad6cd58139b1cb8b` |
| HEAD 树 | `012a214d36ccfa833de0f7a51b37fe1da3243206` |
| 上轮对象 | `1be48c328baea7b7c1664849dfedcc9f9d6d5107` |
| 回归先行 | `0a9a01a95aa67d200f98315109819276956743f7` |
| 修复代码点 | `85ec2ca7857f7770d309edc7368d902b10f57902` |
| 常规 CI 合并对象 | `e654e171dbdeebe607147138912d6fbcdd9cb106` |

远程与本地 HEAD、树一致，结束时没有身份变化。开始及写报告前工作区均干净，没有覆盖其他会话内容；本报告原先不存在。原仓库唯一新增文件为本报告，保持未提交。

已完整阅读实际适用的根 AGENTS.md、Review 095、全部增量、§115、原型文档 §14.11～§14.14，以及必要的共享观察、报告、登记、终判和清理调用链。没有重审全部 T1～T6 或已验收模块。

### 提交与内容隔离

上轮对象之后恰有四个提交：

1. `9d80431df057c6010406829336aafbcb5fb711c2`：归档 Review 095。
2. `0a9a01a95aa67d200f98315109819276956743f7`：回归先行。
3. `85ec2ca7857f7770d309edc7368d902b10f57902`：原型修复。
4. `b80982c2bbf4f7db1c6af179ad6cd58139b1cb8b`：文档及状态。

相对上轮为 **17 文件，+931/-94**：7 份文档（含新增 Review 095），10 份原型测试文件。代码点到 HEAD 仅六份文档：AGENTS.md、data-model、implementation-plan、review-resolution、wp2.5-check-runner、wp2.5-prototype；`prototype/` 与 `.github/` 无差异。

独立确认 `internal/`、Schema、迁移、根依赖、根 ci.yml、诊断 workflow 未变。§115 为新增 78 行，之前的处理记录没有被改写。Review 095 归档提交与 HEAD 的 blob 均为 `48098abc1afecfd8a1819ae2954475293b49457d`；当前文件 SHA-256 为 `0b2d2ce92870721f50935a5ed9ef70ed9ff6603cfcdba2d3eb9ea4d8c3cef199`，与登记一致。能够核对入库对象及登记值；不据此声称重新取得了归档前的历史临时源文件。

回归先行提交增加一次性故障、门禁用例，并去掉 P 测试的 inner 跳过以真正运行目标链；没有提前修正被测错误处理或分类。去掉跳过确实改变了执行覆盖，不能把该提交描述为只有判定表数据变化。

## 3．BLOCKER：B1-R — P-1 屏障区间的退出错误仍被吞掉

### 位置与规则

固定 HEAD：

- `prototype/wp25/exclusive_linux_test.go:89`，尤其 **119～120 行**：
  `defer func() { _ = unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 0, 0, 0, 0) }()`，随后只返回 `fn()`。
- `prototype/wp25/p1_linux_test.go:29`：`p1Barrier` 只处理 `exclusive` 返回的错误；回调把 entered 设为 true。
- `prototype/wp25/p1_test.go:118`、165 行：healthy 的屏障要求是 entered=true。
- `prototype/wp25/p1_unix_test.go:124`：subAfter 的取样属于**前一次**内层区间，发生在调用新屏障之前。
- `prototype/wp25/main_test.go:343`：没有 invalid 或断言差异即可 confirmed。

依据是原型 §14.2、§14.11 的“计划外工具失败使实验 invalid，之后成功不豁免”，以及 §14.11 对 P-1 屏障“其他错误→观察失败”的声明。清除 subreaper 是区间退出管理，不属于 C-17 所允许以最终结束确认裁定的终止请求。P-2 对 SET 0 拒绝的**计划内**负向实验也不能豁免 P-1 的计划外失败。

### 可逐行推出的反例

使用当前已有的 `P1.healthy.1` / `control.p1-healthy` 调用拓扑，考虑以下返回序列：

1. 内层 healthy 区间、进程观察、回收和第一次 SET 0 均正常；r.subAfter=0。
2. 外层随后调用 `p1Barrier`；基线可靠为空，SET 1 成功，回调确实 entered=true。
3. **该屏障退出时的 SET 0 返回错误**。
4. deferred 调用丢弃错误，`exclusive` 返回回调的 nil；`p1Barrier` 不 flag，返回 entered=true。
5. 先前的 subAfter=0 不会重新取样；剩余释放与进程清理可正常完成。healthy 的全部预测仍一致，因此可 confirmed / 0。

若原生清除确实失败，属性还可能保留为 1；进程后来退出不等于该次清除成功。即使故障接缝是在真实清除成功之后替换其返回错误，该次**已观察到的错误**仍被丢弃，也违反本轮持续保存错误的契约。

**证据等级：源码条件证明，未在本机制造新的 Linux SET 0 故障。** 本机为 Windows，本轮没有新增该注入或开展新的机制实验，也没有声称当前 CI 实际遇到此故障。这里不是要求生产 Runner 或新产品政策，而是当前被审 P-1 有效性链自己的可达返回分支没有消费错误。P-2 的既有计划及本次 Linux 日志已经明确允许 SET 0 失败；不能假定该管理调用没有错误出口。

### 影响与最小修正

本轮十二项点名回归的成功不能证明共享调用链已完整失败关闭。这个遗漏允许工具管理失败被当成成功屏障；因此 **B1 不整体关闭，技术 NO-GO**。不撤销已独立支持的 P-4 原反例修复或把整套旧原型重开。

最小修正：

- 让区间退出错误返回并保留；与回调错误同时存在时不得覆盖其中任一个。P-1 调用者必须把计划外清除失败持续记入 invalid。
- 对**第二次屏障区间退出**增加一次性故障回归，核对命中、理由、内层 invalid/非零、健康对照，以及实际清理/残留状态；仅检查前一次 subAfter 不够。
- 同步错误传播表，不能在仍忽略该返回时写成此调用链所有错误已消费。无需改生产代码、产品决定或无关事务。

## 4．已成立的修复、N1/N2 与门禁

### P-4：原点名路径成立

`p4Exit` 不再把读取失败压成空值；`p4ReadExit` 将合法等待超时、等待失败、可靠取得状态分开。等待或读取失败直接 `s.flag`，后续外层读数、合法 NULL、终止与回收不清除它。

外层零等待、外层退出码读取及移交句柄读取都检查错误；加入 Job、确认成员、线程快照、打开线程的计划外真实失败也 flag。计划内恢复异常、Job 查询接缝或被吞掉的终止请求仍作为相应机制负载，不一律 invalid。Runner 取得状态时必须与外层一致，只有明确等待超时允许合法 NULL。

本机独立复验的原四条错误放行均由旧 confirmed/0 变为新 invalid/1：Runner 退出码、外层退出码、异常等待、外层等待。另三个旧 refuted 路径变为 invalid；healthy 观察失败如今理由明确指出原错误，不能只靠附带的负载失败过关。各条新门禁均命中一次故障，之后清理及最终状态正常。没有把接缝返回当作原生 API 已自然失败。

### P-2：步骤表与原反例成立，仍保留支持限制

`p2Steps` 事先列出正常、各 seccomp 拒绝、SET 0 拒绝的完整序列，覆盖目标拒绝之后的关闭及末尾 GET。`p2Judge` 对匹配步骤检查预期 errno 与观测值，整个序列也需匹配；计划拒绝未命中不能 confirmed。正常组不再完全跳过错误检查。

检查进程 poll、waitid、pidfd 关闭及状态读数有实际消费者；poll 错误不再继续到无条件 waitid。EINTR 重试沿用同一 deadline，没有每次重置预算。waitid 在本轮自身、仍未回收的子进程 pidfd 已报告可读之后使用；这依赖受控 helper 拓扑及没有另一个回收者，不能宣传为任意进程的可中断等待保证。后续 Wait 有 5 s 观察窗口；不因 Wait 任务启动便宣称回收完成。

Linux 修复诊断的三个新增故障分别 invalid/1，理由对应 poll、own pidfd close、外层 status；正常、计划拒绝、SET 0 拒绝对照保持 confirmed/0。SET 0 组记录 EPERM、SubEnd=1、后续探测拒绝、外层最后终止实验进程；**实验进程退出不是内层清除成功**。

这些是实际执行路径证据，不是所有 API 故障已动态覆盖；B1-R 也不是由这张 P-2 表自动解决。

### P-1、N1、N2

- `errBaselineUnread` 与可靠非空基线分开；Linux 的 unknown 基线不能再当作正常屏障拒绝。healthy 要求实际 entered。
- healthy 的 E-main 观察失败、Unix exited 夹具未知、Windows 移交句柄读取失败都显式进入 invalid。外层分支、补救和清单释放错误保留。
- **N1 关闭**：`p1_test.go:172～195` 先执行补救、`h.release()` 和逐项 release，再取得 acc2；断言对应“结束和释放之后仍拒绝”。若释放错误，也会另行 flag，而非用拒绝掩盖。
- **N2 关闭**：规范 H 维度加入 Q-1 的 F1/A5/A6 及未确认进程资源；§19.1 引言、状态说明、§19.11 原型状态与 data-model 的 A12/context_changed 原因一致。没有改逐条推导、可靠退出码保留或取消优先级。§19.1 的 T_kill 措辞精化仍按原建议留作冻结前工作，没有冒称本轮已完成。

### because 与外层判定

`gate_test.go:641` 的 because 是**附加条件**：必须已有正确内层 verdict/退出码，还必须在理由中找到目标错误。命中次数、见证、归属、普查、补救、结束状态和保险清理继续单独裁定。

`judge` 的 toolClasses 优先于预期拒绝类别；最后要求没有未允许问题/工具错误、补救 not-needed 或 succeeded、Final=ended。`finishGate` 的保险结果仍能否决通过。找到预期理由不会豁免另一个外层错误。未发现本次 because 变化绕过原组合终判。

## 5．P-2 新登记协议

实际顺序：

1. 实验进程通过共享 start 创建 `report-exit`；在门禁内，创建记录与 helper 回执继续使用已有见证协议。
2. 检查进程报告自身 PID，报告失败返回 exitReport，随后退出；实验进程保持它未回收。
3. 外层 `runP2:594～603` 逐条读取报告，以实际 procInfo 验证 present、PPID 等于本轮实验进程 PID，保存启动身份并 `s.own`。
4. **只有登记后**外层才写本场景新临时目录下的 ack 文件。
5. 实验进程不论观察是否成功，都先等 ack（10 s）。再按观察结果结束自有未回收子进程，并在 5 s 内等 Wait；之后发布实验报告。
6. 外层读取最终报告、自/外 status、子进程列表与 pidfd 数，释放实验进程，确认回收，再进入预测和场景终判。

这里可登记的是已退出但尚未回收、身份与父子关系仍可读的子进程；不能要求它必须还在执行用户代码。未登记时不得把报告 PID 当作操作授权。`pidProc` 动作前仍核对启动身份，unknown 会持续保存观察错误。报告缺失/损坏、身份无法验证、ack 写失败及最终报告读取失败都不能成为绿色证据。

检查进程是实验进程通过自己的 Cmd 创建、保持未回收的直接子进程；实验侧对它的失败收尾不依赖外层转述的 PID。外层只有经父子关系验证后才取得 PID 动作权。外层提前拒绝时，场景与门禁仍保留短缺、未知或清理失败，不能用最终空集合代替成功登记。

修复诊断中 `P2.normal.1` 实验 PID=6953、检查 PID=6960（登记身份 44975）；步骤含等待登记、reap exit 0，末尾 children=0、pidfds=0、SubEnd=0。SET 0 组实验 PID=7025、检查 PID=7032（身份 45601），登记及回收同样完成，SubEnd=1 如实保留。它们在回归先行代码中曾是内层 confirmed/0、外层 unaccounted 而拒绝；修复后两层均通过。证据来自 GitHub 既有日志，不是本机 Linux 重跑。

新增握手延长了检查子进程的**未回收保留期**，文档 §14.11、§115 已披露；没有让被测 poll、waitid 或 SET 0 预期依赖生产推导结果。其失败或 timeout 不是“正常回收已完成”。

## 6．NON-BLOCKER

### N-096-1：macOS PID 2668 的归属措辞超出证据

位置：固定 HEAD `docs/wp2.5-prototype.md:2672`、`docs/review-resolution.md:7012`。

回归先行诊断 macOS job 122660646495 的真实行：

- `report.epipe-unwritable` 内层 invalid/1，预期的报告失败拒绝已出现。
- 外层 **tested census** 读取 PID 2668 的环境返回 EINVAL；状态读数：启动身份 `1793167516.433196`、parent=1、group=2668、名称前缀启发式=no；失败后该身份仍 present、非僵尸。
- 外层将 census 错误列入 tool_errors，**Final=unknown、Pass=false**。
- Insurance 写的是“nothing left”；不能由此把普查未能判定归属的 PID 2668 解释为已完整清理。

故文档“一个无关进程”“与 B1 无关”的排他表述过强。能说的是：这是既有普查读取失败的又一次观测，**不属于十二个 B1 目标故障的预期命中**；PID 的归属、EINVAL 根因及该运行对所有对象的完整清理仍未知。PPID=1、名称不匹配不能证明无关，也不能反过来证明本轮所有。

最小修正：把上述两处归属结论收窄为未知，保留原失败、线索与实际最终状态。这是非阻塞措辞项：本次门禁已经正确拒绝，没有绕过 EINVAL；历史不可追溯本身不构成永久合并门槛，后来的成功也不能回填旧运行。

### N-096-2：新 P-2 握手可先于 unsupported 判定超时

位置：`p2_linux_test.go:335～344`、590～630；文档 §14.11 的“过滤器装不上→unsupported（不变）”。

`prctl.set0` 预期有检查进程，故 checks=1。但若架构或安装过滤器不支持，实验进程先发布 Unsupported 并等待释放，不创建检查；外层却在读取该最终报告前等待 10 s 的 PID 记录，因缺记录 invalid，无法走到后面的 unsupported。

这是新握手引入的确定分类顺序问题，不是 confirmed 假通过；本次 CI 环境已成功安装过滤器，未走该分支，本轮未另造原生安装失败。最小建议是在等登记时也识别经核验的“不支持且未启动检查”的实验结果，保留未知/损坏报告拒绝；补不支持对照。不能把它归因于已启动检查后正常结束，也不必修改生产支持范围。

另有分类边界应与表述同步：`p2Probe:220～225` 把 self procInfo 的 unknown 压成 /proc=false，后续可变成 refuted，而不是保存读取失败。它不会在当前正常/拒绝表里成为 confirmed，但“所有计划外观察失败均 invalid”的概括仍应收窄或补齐。这里只登记错误分类与证据范围，不把非绿色的保守拒绝机械升级为另一个合并阻塞。

## 7．本轮独立运行与首次失败

平台：**Windows/amd64，Go 1.27.1**。通过固定 Git 对象导出到仓库外：

`C:/Users/hoshi/.codex/visualizations/2026/10/12/01a13137-991b-7b42-8ea2-3c24008343a0/review096/`

new 为固定 HEAD，old 为回归先行 0a9a01a。没有修改导出的被测代码；使用已有场景、已有注入、离线依赖和独立 GOCACHE。预先限定每组选定测试各一次、每次 120 s；没有压力循环、原生机制新实验或 CI 重跑。

| 运行 | 实际结果 |
|---|---|
| 首次 new，只选择 13 项 TestGate | 各门禁均 PASS，但整个包 **FAIL**：TestMain 发现根层 scenarios=0，按既有规则拒绝。是本次选择范围无效，不能冒称整组首次通过，也不是产品缺陷 |
| new 加入 P1.healthy.1，重做同一 13 项门禁 | **PASS，19.260 s**；1 个根层场景 confirmed，13/13 门禁 |
| old，同一修正后的选择 | **预期 FAIL，25.174 s**；1 个根层 healthy confirmed，门禁 6/13，七项失败 |
| new 的 P1/P4 + Q12Classify、GateJudge、WitnessJudge、StartPoints、PhaseClock | **PASS，21.362 s**；P1 六次、P4 十八次，共 24 个机制观察；直接测试另计 |

十三项门禁包括 5 个正常/计划负向对照、8 个观察故障：

- 五个对照在 old/new 均 confirmed/0。
- old 的四项 P-4 错误为 confirmed/0；两项移交句柄及 healthy 退出码故障为 refuted/1；healthy E-main 故障虽 invalid/1，但旧理由只有负载不符。
- new 的八项故障均 invalid/1、一次命中，理由指向目标观察；全部外层 Final=ended，未有 tool_errors、未知清理或保险补救掩盖。

本地日志为 `new-gate.log`、`new-gate-control.log`、`old-gate-control.log`、`new-direct.log`。根层机制结果、门禁正确拒绝、纯判定表不混计；没有在本机执行组合套件、Linux P-2 或 macOS 场景。

收尾：各被运行门禁记录 Final=ended，独立机制场景记录清理完成；结束时 Get-Process 未发现 wp25.test。一次更广的 CIM 查询因本机权限不可用，未将其当作产品错误，也没有据此绕过环境限制或操作未知进程。没有终止任何非本轮对象。

作者 M1～M12、作者 Windows 完整套件和 WSL2 结果只作作者证据；本轮没有重复这些变异，编译失败不计检出。本报告 B1-R 是源码条件证明，不能与上述已动态复验的旧/新八项故障混为一谈。

## 8．CI 独立核验

读取 GitHub API 的正式 run/job 状态与十四份 job 完整解码日志，核对 checkout，并按 ID 去重 Results 步骤重复打印。没有只依赖 PR 正文或作者汇总。

### 原型：与常规 CI 分开

| run / 对象 | 事件、attempt、正式结论 | macOS | Ubuntu | Windows |
|---|---|---|---|---|
| [40641559468](https://github.com/XKHoshizora/specmetry/actions/runs/40641559468) / 0a9a01a | push，1，failure | 33 confirmed；门禁75/76；组合13/13 | 53 confirmed；86/92；13/13 | 53 confirmed；84/91；13/13 |
| [40643975558](https://github.com/XKHoshizora/specmetry/actions/runs/40643975558) / 85ec2ca | push，1，success | 33 confirmed；76/76；13/13 | 53 confirmed；92/92；13/13 | 53 confirmed；91/91；13/13 |

回归先行三个 job：macOS **122660646495**、Windows **122660646732**、Ubuntu **122660646763**，都实际 checkout 完整 0a9a01a。不能把 macOS 的计划外 EINVAL 失败也写成预期 B1 检出。

修复三个 job：Windows **122668387092**、macOS **122668387295**、Ubuntu **122668387312**，都实际 checkout 完整 85ec2ca。日志 META 为 Go 1.27.3；Linux/amd64 为 6.17.0-1022-azure，darwin/arm64 为 macOS 26.6.2/Darwin 25.6.0，Windows/amd64 为 10.0.26100。Ubuntu 的 RUNNER_OS=Linux 路径设置并传入 **-race**，其 Prototype 步骤及最终测试成功；这不是根 CI 的 root 步骤。

固定诊断场景 ID 并集为 **86**：原有 53 加 P1 六个、P2 九个、P4 十八个；三平台 33/53/53 是跨平台观察次数，不是 139 个独立机制。门禁、组合及直接测试分开统计。各平台三个内层专用测试在根层设计 SKIP：TestInnerForeignPID、TestInnerLeak、TestInnerStopError；它们经门禁/组合的内层使用不算根层通过。Linux/Windows 专属 P-2/P-4 在 macOS 不编译，不计 macOS 通过。

TestPhaseClock、TestF1Classify、TestQ12Classify、TestStartPoints、TestWitnessJudge、TestGateJudge、TestRecordAlive 等直接检查在适用日志实际通过；纯表与源启动点检查不替代真实 API 观察。P-2 新登记的实际步骤已在 §5 单列。

### 最终常规 CI

[40645291628](https://github.com/XKHoshizora/specmetry/actions/runs/40645291628)：pull_request，attempt 1，**completed/success，8/8**。run 的 head_sha 为固定 HEAD。

| Job | ID | 正式结论 / 实际 checkout |
|---|---:|---|
| Web | 122672580180 | success / e654e171… |
| Go tests Windows | 122672580399 | success / e654e171… |
| Go tests macOS | 122672580420 | success / e654e171… |
| Go lint | 122672580461 | success / e654e171… |
| Go tests Ubuntu | 122672580479 | success / e654e171… |
| Build Ubuntu | 122672690830 | success / e654e171… |
| Build macOS | 122672690879 | success / e654e171… |
| Build Windows | 122672690889 | success / e654e171… |

八份日志 checkout 都是完整 `e654e171dbdeebe607147138912d6fbcdd9cb106`。独立读取该 Git commit：父提交依次为固定 base、固定 HEAD；树为 `012a214d36ccfa833de0f7a51b37fe1da3243206`，与 HEAD 相同，故内容适用性成立。

Ubuntu 常规 Test 路径带 -race；Cross-user path tests 为**另一个**成功步骤，独立编译测试二进制后 sudo 运行，不能称作同一个 -race 运行。普通步骤的 root 专属跳过按设计保留，后面的 root 实际执行另计。其余平台/旧 Git/权限设计跳过不当作通过。根 ci.yml 不运行嵌套的原型模块。

保存既有失败及取消历史，不通过新绿状态改写原运行；本轮没有触发、重跑或刷新 workflow。常规存储测试成功不关闭 M-14～M-17，原型成功也不能豁免 B1-R。

## 9．证据对门槛的支持与最小下一步

| 门槛 | 本轮支持 | 冻结前仍缺 |
|---|---|---|
| G3 / P-1 | 正常、alive、exited 拓扑及 Cmd/句柄移交、禁用、释放后仍拒绝；unknown 基线和已点名读取故障会拒绝 | 先补 B1-R；原有生产拓扑、宿主资源生命周期与并发接线限制继续保留。Windows alive 额外持有 Job 句柄改变最后句柄关闭前提，只支持受控构造下的所有权/不阻塞，不能推出任意真实 F1 |
| G6 / P-2 | 九组预定序列、seccomp 参数对照、自/外读数、检查实际创建/登记/回收；SET 0 失败及属性保留如实表现 | 只适用已测内核/架构和过滤器前提；改正 unsupported 分类边界。不能从隔离实验退出推导生产清除成功；生产运行中失败与能力探测接线仍需兑现 |
| G7 / P-4 | 九种构造/接缝各两次，异常恢复知识状态、可靠退出码、父端确认窗口与移交；原 B1 点名错误持续拒绝 | 返回值、改变前提和接缝仍须分开；不等于真实不可终止故障、任意宿主、全部 Windows 布局已验证。生产 API 失败链及资源接线仍待授权后验证 |
| G1 / G12 | 已批准决定及模型判定的既有结论保留，N1/N2 修订正确 | 模型零访问与模型行不是数据库、Git、seq、事务或准入的生产证明 |
| G5 / P-3 | 仅整理状态 | 未执行，门槛不满足，不自动取消 macOS 支持 |

**当前 PR 必需的最小修正**是 §3 的共享区间清除错误传播与定向回归；文档应一并收窄 PID 2668 的归属结论。N-096-2 不形成绿色假通过，按非阻塞分类/证据说明处理，不要求另开生产工作。

冻结前继续保留第 7 版已列的规格与必要证据缺口。生产授权后才可做生产 Runner、CLI、数据库写入/seq、完整取消及持久化组合、配置固定、输出接线与最终工作包验收测试；本审查不要求这些提前成为原型合并的条件。

**PRODUCT DECISION：没有新增；ACCEPTED RISK：没有新增。** Q-1、Q-2(a)、C-4、D21 和已批准方案 B 边界不重选、不扩大；Log 完整性、归属未确认零动作等保证不豁免。旧 macOS 2088/1989 及本轮历史 PID 2668 不回填为已解决。M-14～M-17、N-087-1 保持原状态。

本报告仅给独立审查结论，不宣布规范冻结、生产授权、WP2.5/M2 验收或风险接受。

