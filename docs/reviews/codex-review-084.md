# Codex Review 084：PR #34 最终交接与 macOS 诊断补充的定向独立复审

## 1．结论

**技术 NO-GO，当前不建议合并 PR #34。** Review 083 的精确反例已被拒绝，但发现新的 **BLOCKER B-N4：有效 ok 与真实 epipe 记录同时到达时，冲突检查被跳过，内层场景错误地 confirmed / exit 0**。因此 N4-R-S 尚不能整体关闭。见 §3 的源码路径、真实 helper 复现及最小修正对照。

**当前两套 CI 均正式成功。** 固定代码点的三平台原型诊断及最终 HEAD 的常规 CI 已满足本次查询所对应的运行条件；它们没有覆盖本轮独立发现的组合反例，不能替代技术修正。

**旧 macOS 失败本身不构成永远不得合并的门槛。** 新增信息与保守失败路径基本成立，旧 PID 的归属、EINVAL 原因及当时完整清理结果仍未知。当前不建议合并的依据是可复现的绿色假通过，不是要求反复运行直到旧 EINVAL 再现。

| 问题 | 本轮裁定 |
|---|---|
| N4-R-S 能否关闭 | **不能整体关闭**。旧反例、输出未最终关闭与关闭失败已拒绝；另有终态冲突的绿色假通过 |
| N-MAC 已完成什么 | 增加关联到原列表实例及复核状态的有限诊断；保留原始错误与保守失败；更正历史归属推论 |
| N-MAC 仍未知什么 | 旧两次失败的对象归属、根因、完整清理；真实 EINVAL 尚未由新诊断重现 |
| 新 BLOCKER | **B-N4：ok + epipe 共存绕过交接验证** |
| 新 NON-BLOCKER | N1：`same program` 实际只是短命令名前缀启发式，标签比证据强 |
| Review 083 已关闭的 B1、N6 | 未发现本轮直接回退；保留原裁定与适用范围 |
| 当前合并建议 | **不建议合并**；修正 B-N4 并取得适用的针对性证据后重新判断 |

没有批准新的产品选择、保证调整或接受风险。14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14、M-15、M-16 保持未解决。

## 2．固定身份、核验与范围

| 对象 | 完整身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| Review 083 对象 | `b51cc5e571dc7f1cad00f49055d862e08a5833f0` |
| Review 083 归档 | `ab399e58c41beaa1917f01a8c11eb9cafb6bc8bd` |
| 回归先行 | `61a43db7f86ff6ceb4e8b1259cba07d7e63746cd` |
| N4-R-S 修复 | `419e4f388ecc7cbdee2c853a337f4c13303a055e` |
| 最终代码 / N-MAC | `56234c364eaffc109f1ddf1cd0bd837420afcbf7` |
| 文档收尾 | `f7432f0485c55a8888e8bcc279e9d6449f85efb8` |
| 最终 HEAD | `2414ae939b9a47c73fc7b440bf73d7afa520463a` |

开始于 **2026-10-24 12:05:05 UTC** 核对：远程 HEAD/base 与固定对象一致，PR open、未合并、auto_merge=null；本地 HEAD 为固定目标，工作区干净，报告不存在。技术核查后于 **12:17:44 UTC**、**12:20:47 UTC** 再查，身份及 PR 状态未变化。没有切换对象或原工作区、fetch 到原仓库、改基或覆盖其他任务内容。落盘后另记最终核验。

已完整读取适用 [AGENTS](../../AGENTS.md)、[Review 083](codex-review-083.md)、本轮完整增量、[原型文档](../wp2.5-prototype.md) §2.15 及更正、[处理记录](../review-resolution.md) §99，以及 helper、管道、保留读端、报告终判、门禁、普查和保险清理的必要调用链。没有重做整套 T1～T6 或已验收模块审计。

相对上轮为 **13 文件，+986/−30**：

- 文档 5 份：AGENTS、implementation-plan、review-resolution、新增 Review 083、wp2.5-prototype。
- 原型 8 份：新增 census_check_darwin；修改 census_darwin、gate、helper、keep_unix、keep_windows、pipe、report。

**56234c3 → HEAD 只有两个提交、四份文档变化**；prototype/.github 差异为空。原型子树均为 `288579a06b360aab452a240910298aeb9d3205bf`。生产代码、根模块依赖、Schema、迁移、ci.yml、原型 workflow、正式契约及 check-runner 草案未变。

Review 083 归档 SHA-256 为 `1989204ff4f8d539ee04660e557e6390aaf35e67a1b95e81d31de7737efef8d4`，与登记一致。review-resolution 只追加 §99（164 行）；§84～§98 及历史审查未改写。活跃文档保留旧记录并加更正，没有替换原受审身份或把实施方“已修复”写成审查关闭。

## 3．BLOCKER B-N4：终态冲突被分支跳过，形成绿色假通过

### 3.1 预期与实际

预期依据是本轮交接协议及文档 §2.15：回答只能是观察的最终回答；**回答与失败记录同时存在应 invalid**，不能把失效的回答或未闭合的观察重新当作有效机制证据。

实际路径，位置均对应固定 HEAD：

1. [pipe_test.go:150](../../prototype/wp25/pipe_test.go#L150) 允许各至多一条 `ok` 和 `epipe`；结构校验没有规定两者互斥。
2. 第一次等不到 epipe，进入 [201 行](../../prototype/wp25/pipe_test.go#L201) 的询问路径。
3. [220 行](../../prototype/wp25/pipe_test.go#L220) 得到有效 ok；等待期间或之后出现真实写入失败及有效 epipe。
4. [234 行](../../prototype/wp25/pipe_test.go#L234) 重读已得到 epipe。
5. **[237 行](../../prototype/wp25/pipe_test.go#L237) 只有 epipe 仍为空才执行 EOF、迟到字节和终态冲突检查。** 此时已有 epipe，整个保护分支被跳过。
6. [267～268 行](../../prototype/wp25/pipe_test.go#L267) 的“回答并报告失败”检查因此没有执行。
7. [295 行](../../prototype/wp25/pipe_test.go#L295) 恰好把一条 epipe 判作预测成立。其他正常条件满足后，内层 **confirmed / 0**。
8. [report_test.go:129](../../prototype/wp25/report_test.go#L129) 的最终报告校验只核对各事件数量、重复与格式，不拒绝这组语义矛盾；正常清理也不能恢复观察的有效性。

这不是“错误后仍 refuted”的旧非阻塞残余，而是本 PR 要修的有效性门禁直接形成了绿色机制结论，故列为 BLOCKER。没有重开已关闭的停止任务 B1。

### 3.2 独立真实 helper 复现

在仓库外固定目标副本中，沿已有 `report.fail-after-ok` 的真实 helper 调度，仅改变一个故障维度：

- 保留 helper 真实写 ok、关闭 stderr、下一次真实写 stderr 失败的顺序。
- 从 [helper_test.go:261](../../prototype/wp25/helper_test.go#L261) 的“故意以只读方式写 epipe”选择条件中去掉 `fail-after-ok`，**让这次真实失败正常报告出来**。
- 没有修改 verdict、预测条件、gate judge、退出码、S6 或清理逻辑；只另加观察打印以保留内层完整输出。

结果：

| 观察 | 实际结果 |
|---|---|
| 初次失败记录 | `[]` |
| 回答后重读 | `[30392]`，为本轮已核验 holder |
| 内层预测 | `holder's late writes fail: 1 epipe reports` |
| 内层结果 | **confirmed / exit 0** |
| 实际工作量 | 198；真实启动父 helper 与 holder |
| 清理 | 两个自有对象（父 55356、holder 30392）均核实结束 |
| 更外层原有门禁 | 想要 invalid，实际得到 confirmed/0，故 `pass=false`、整个此次定向 go test 失败 |

**内层错误绿色与外层检出这个错误必须分开。** 本轮没有声称修改后的整份诊断仍绿色；现有外层断言正确指出反例。但仓库现有故障组合把 epipe 写入也一并破坏，没有覆盖“报告成功”的这一合法组合。

原有 `.failed` 标记只在 epipe 报告本身失败时写入；本次报告成功，所以其打印为 false，父端按现有代码等待满 3 s。没有把该标记误当作注入证据：真实 ok、真实关闭后的写失败、真实 epipe 记录、对应注入命中及内层完整输出构成证据。该等待也给了错误足够的到达时间，不依赖抢到极小窗口。

同时运行的健康预期保持 confirmed/0，可靠反证保持 refuted/1，正常排空保持 confirmed/0；不是把所有路径改成 invalid。

这是本轮已有受控 helper 故障模型内的组合，不是要求协议抵御任意恶意报告程序。

### 3.3 最小修正与独立对照

最小方向：**终态一致性不能放在“当前没有失败记录”的保护条件内。** 在已得到回答的路径中，无论失败记录在询问前后哪次读取出现，都必须先处理矛盾，再决定是否接受预测。保留：

- 只有真实 epipe、没有矛盾回答：仍可证明预测现象；
- 最终回答 + 两路 EOF + 每路目标时段字节、无错误：可靠 refuted；
- 回答与失败共存、关闭/报告/观察失败：invalid。

隔离副本仅在重读后、该条件分支之前增加“ok 与 epipe 同时存在则 invalid”的检查，同一真实反例变为 **invalid/1**，外层原有门禁 PASS，三个健康/反证对照保持正确。此对照只验证精确反例的最小修正方向，**不是交付实现或证明所有到达次序已修好**。

必要的后续范围是终态归并及对应回归：至少覆盖失败在等待回答期间到达、回答之后最终重读时到达，以及只有 epipe / 只有最终 ok 的对照。不得以补一项同时破坏报告的注入替代报告成功的组合。

## 4．N4-R-S 其他交接环节的证据与限制

| 环节 | 本轮判断 |
|---|---|
| 停止继续写入 | [helper_test.go:110](../../prototype/wp25/helper_test.go#L110) 的正常分支收到询问后不再进入写循环；两路 Close 都尝试，只有两者成功才写 ok |
| 唯一回答 | 正常受控 helper 只写一次并随后等待被结束；重复事件另由报告重复检查拒绝。没有推广为不可信程序协议 |
| 关闭失败 | `output-close-fails` 通过真实第二次 Close 得到错误，不给 ok，场景 invalid；这是错误返回处理验证，不是原生部分关闭失败的穷尽验证 |
| 回答写不出 | 无有效完整 ok，现有故障对照 invalid；没有把缺失回答当作 refuted |
| 对应管道与写端 | EOF 从本轮创建的两路管道的保留读端读取；父进程写副本已关闭，holder 是该布局的剩余写者，保留副本是不可继承的读端 |
| EOF 预算 | [pipe_test.go:248](../../prototype/wp25/pipe_test.go#L248) 只创建一次 `endBy`，两路共用 **2 s**，没有逐路重获完整预算；这是观察协议预算，不是 D/J 预算变更或硬实时保证 |
| 读取方式 | Unix 非阻塞 read，以 EAGAIN 等待；Windows PeekNamedPipe 后仅取现有字节，以 broken pipe 表示 EOF。只支持当前受控、有限输出布局，不据此保证任意持续写者的实时上界 |
| 迟到字节 | 基于停止时每路待取字节数与之后实际读到的总数之差；正常复制任务已停止，避免把旧缓冲计为新增目标字节 |
| 最后报告检查 | 在“仅 ok”分支读取两路 EOF 后重读；正常 helper 的输出关闭和唯一最终 report 是前提。**不能据此断言所有分支均闭合**，§3 正是提前旁路 |
| 工具错误与成功 | 正常已覆盖的故障保持 invalid；成功回答或清理成功不会覆盖它们。但终态矛盾当前可以绕过该规则 |

独立在回归先行代码 `61a43db` 上执行新门禁的两个精确对照：`report.fail-after-ok` 与 `report.ok-not-final` 都仍是 refuted/1，外层要求 invalid 因而失败；在固定最终代码中两者均 invalid。正常 `control.f1-skip-drain` 为 confirmed，`control.f1-read-kept` 为 refuted。**旧机制失败、新机制拒绝的证据成立，但不能外推覆盖新组合。**

## 5．N-MAC：信息补充成立，旧未知仍保留

### 5.1 同一实例、失败和动作边界

[census_darwin_test.go:25](../../prototype/wp25/census_darwin_test.go#L25) 的列表条目提供 PID、启动时间、状态、P_WEXIT、父 PID、PGID。读取环境失败后，重新读取状态并比较启动身份：

- 可靠不存在、已成僵尸，或启动时间不同：按原有逻辑不再把旧列表实例当作当前存活实例。
- 同一实例仍在，或复核本身无法确认：仍返回普查错误。
- 列表信息与复核信息分开标示；后者无法读取明确为 unreadable，不凭旧快照补成当前状态。

[45 行](../../prototype/wp25/census_darwin_test.go#L45) 保留 `errCensus` 及原始错误的包装关系。新增信息没有新增信号、动作授权、自动重试或归属放宽。诊断输出不含环境内容、参数或进程名；短名仅在内存中形成布尔启发式。

这些字段是原列表实例的事实与一次后续复核，**不是原子跨时刻快照**；没有把二次读取的身份变化误当成原实例仍存活，更不能用于补证旧运行的归属。

### 5.2 TestCensusUnreadable 实际证明范围

[census_check_darwin_test.go](../../prototype/wp25/census_check_darwin_test.go) 的两项检查：

1. 创建自己的真实 helper，列表限制为该 helper，环境读取接缝返回 EINVAL；使用真实状态复核，要求普查失败、原错误可追溯、诊断含身份/状态等字段，且不回显 token。
2. 同一环境错误并令状态复核 unknown，要求仍失败并明确 unreadable。

CI 的 macOS Job 实际执行并通过两项。它们证明**注入错误的传播与信息格式**，没有证明真实原生 EINVAL 根因已复现，也没有动态覆盖 PID 复用、所有退出时序或所有字段一致性。第二项对错误链的细致断言少于第一项；源码的统一返回路径仍保留原错误。本机为 Windows，未冒称本机执行 macOS 用例。

### 5.3 NON-BLOCKER N1：短名匹配不能命名为程序身份

[census_darwin_test.go:81](../../prototype/wp25/census_darwin_test.go#L81) 的 `sameProgram` 实际判断是：

`strings.HasPrefix(filepath.Base(self), comm)`

另一个不同程序只要短命令名等于该前缀，也会给出 yes；不同可执行内容的同名文件同样如此。故 [75 行](../../prototype/wp25/census_darwin_test.go#L75) 的 `same program as this test` 容易把启发式误读成程序身份。

建议只把标签和文档改成“短命令名前缀匹配（启发式）”，继续不输出原名；无需新增侵入式进程路径采集或把它纳入动作授权。此项由源码即可成立，未冒称 macOS 原生动态反例。当前错误分支仍失败、没有据此信号或认领，所以为 NON-BLOCKER。

### 5.4 旧 macOS 运行：事实、推断、未知

| 旧观测 | 可以确认 | 不能确认 |
|---|---|---|
| run 40072151282，PID 2088 | 被测普查失败，结果 final=unknown；后续保险报告 nothing left | PID 归属、EINVAL 根因、当时全部自有对象是否均被覆盖并清理 |
| run 40074633075，PID 1989 | 保险清理普查失败；被测路径此前为 final=ended；整次诊断失败 | 保险普查完整性、该 PID 归属、当时完整清理及与前次是否同根因 |

缺少完整创建记录不能证明“无关进程”；同用户、时间重叠也不能证明本轮归属。两次发生阶段不同，不能用相同 errno 合并根因。本轮新增的更正忠实保留上述未知，未用当前绿色回填过去。

按本次要求分别裁定：

A. **旧失败没有提供当前仍存在的具体不安全动作缺陷证据。** 它们证明过去诊断保守失败与信息不足；不能据此断言误杀、当时已安全清理或必定与改动无关。

B. **当前受控原型对无法读取环境的保守失败契约保持。** 新接缝与源码支持错误传播；未知不是空集合，不忽略 EINVAL，不自动重试。

C. **最新固定代码的三平台诊断确已正式成功，适用于其覆盖的机制与门禁。** 旧最终诊断失败这一运行条件已有新的适用证据；但该证据不能抵消 §3 的独立反例，故不足以支持当前 PR 整体合并。

D. **不要求反复施压直到偶发 EINVAL 再现。** 必要新增验证是 §3 明确的终态冲突假设及其最小回归；不把无法追溯旧进程当作永久合并门槛。今后若自然再遇到该错误，可据新增字段单独诊断，仍不得凭短名或旧记录缺失推定归属。

## 6．独立运行、直接回归与证据边界

本轮实际平台：**Windows 11 / 10.0.26300 / amd64，Go 1.27.1**。使用既有工具，GOTOOLCHAIN=local、GOPROXY=off，没有安装升级。实验副本均在仓库外独立目录，使用固定 Git 对象导出，不切换或修改原工作区。

| 验证 | 本轮独立结果 |
|---|---|
| 固定目标定向测试，240 s 硬期限 | PASS；顶层 3 个机制场景 confirmed、51 个门禁、13 个组合全部通过，及 StopRead、StopAPIErrors、LingeringFailedStop、PhaseClock、Candidates、GateJudge 等直接检查 |
| 回归先行旧机制，60 s | 两个新增交接回归均得到错误 refuted，外层 FAIL；两个健康分类及正常排空对照正确 |
| §3 新组合，25 s | 内层 confirmed/0、外层正确 FAIL；完整日志保留，真实进程与故障命中 |
| 精确最小修正对照，25 s | 内层 invalid/1、外层 PASS；健康/可靠反证/正常排空保持正确 |
| 归属与清理直接回归 | 两项 forge-owner 对照保持零动作/零信号；混合场景自有遗留实际清理；补救失败、工具失败与未知状态的组合仍不通过被测自检 |
| N6 | PhaseClock、真实正常/escaped-holder 管道、提前唤醒等适用检查通过；同一阶段时钟、固定截止、原预算未变 |
| B1 | 相关停止错误、晚到错误与接手检查通过；此次修改没有回退到同步阻塞或抛弃移交任务 |

定向测试选择为：

`^(TestGate|TestGateCombos|TestStopRead|TestStopAPIErrors|TestLingeringFailedStop|TestPhaseClock|TestCandidates|TestGateJudge|TestPipeNormalDrain|TestPipeEscapedHolder|TestPipeF1SkipDrain)$`

只统计顶层场景一次；门禁 PASS、内层 invalid/confirmed、组合及直接表测试各自独立。没有把本机 3 个机制场景说成全部 29 个；没有本机 Linux/macOS、原生断电、不可控阻塞、任意 PID 复用或双账户新动态验证。

实验均为自己的 helper，使用现有身份/Job/清理协议；新反例的内层和外层均确认已启动的自有对象结束，结束后未发现本轮 review084 测试进程残留。没有对未知 PID 发信号。

**首次失败与准备边界**：旧机制及新组合的预期运行失败均如实保留。临时最小修正第一次文本定位因已加入的观察打印而未匹配，脚本在写入/编译前拒绝，不能计为检出；随后按实际分支定位并成功编译运行。原仓库没有本地合并快照对象，故其身份通过 GitHub 只读 commit API 核验而非向原仓库 fetch。日志初次按错误目录读取失败，随后从实际副本目录读取；这不是产品失败或新增运行。PowerShell 交互 profile 的工具路径警告与原型无关，后续读取使用无 profile 方式。

仓库外日志 SHA-256（留作本轮证据识别，不添加仓库文件）：

| 日志 | SHA-256 |
|---|---|
| fixed-run.log | `29bfc99de90bb3334c234cfabe65f5e1b15eb750595bac49d934f27e26e84b06` |
| old-run.log | `8923bcb4315b533d7fd1fb9ae02081cfe8a733c5f7cd78d2c5a1d33721fb78c8` |
| both-run.log | `cd8dff934e521b95f9cbbef8828213399ffc09583c1d3b2e2d7e26e0027cf507` |
| repaired-run.log | `bc91655fe761c42dd124b89c8eafbcf541982a7d4f5bd66ce95b75ad07d36eef` |

作者 Windows/WSL 全量运行与 M51～M59 撤回实验属于作者证据，没有冒称本轮独立重做；M59 首版编译失败不计有效检出。没有通过本机临时修正交付实现。

## 7．CI 独立核验

已读取正式 run/job 汇总及两套最终运行全部 11 个作业日志，核对事件、attempt、实际 checkout、测试结果与范围；没有触发或重跑 workflow。

### 7.1 固定代码原型诊断

[run 40084692941](https://github.com/XKHoshizora/specmetry/actions/runs/40084692941)：**push，attempt 1，completed/success**。三个 Job 实际检出均为 `56234c364eaffc109f1ddf1cd0bd837420afcbf7`。

| Job / ID | 机制场景 | 门禁回归 | 组合 |
|---|---:|---:|---:|
| Ubuntu / 120845104188 | 38/38 confirmed | 55/55 PASS | 13/13 PASS |
| Windows / 120845104367 | 29/29 confirmed | 51/51 PASS | 13/13 PASS |
| macOS / 120845104374 | 27/27 confirmed | 46/46 PASS | 13/13 PASS |

场景 ID 去重并集为 **53**，没有累加三平台数或 Results 重复打印。Ubuntu 实际执行 `go test -count=1 -v $race -timeout 15m ./...`，Linux 分支设置 `race=-race`；未报告竞态。另两平台没有该参数，不等同于也完成 Race Detector。

各平台三个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError` 在顶层 SKIP；对应门禁/组合确有实际子进程执行，不计 SKIP 为通过。没有其他顶层场景 SKIP。macOS 两项 CensusUnreadable、Windows 三项 StopAPIErrors 及其他适用直接检查实际通过。

现有 `report.fail-after-ok`、`report.ok-not-final`、`report.close-fails` 均 invalid；`report.epipe-late` confirmed；健康与反证对照分别 confirmed/refuted。**CI 中没有 §3“报告成功”的新组合。**

### 7.2 最终常规 CI 与内容适用性

[run 40087043315](https://github.com/XKHoshizora/specmetry/actions/runs/40087043315)：**pull_request，attempt 1，completed/success，8/8**，HEAD 为固定目标。

所有作业实际检出：

`11490a4d538bfc111960ef9b5a414c1d8e3228df`

其父提交依次为固定 base、固定 HEAD；内容树：

`ead9125ea6f0aef61bc0bc1901b6568a4de52770`

与本地固定 HEAD 内容树相同。合并对象元数据经 GitHub 只读 API 核验，不仅依据 PR 正文。

| Job | ID | 最终结果 |
|---|---|---|
| Web | 120851998465 | success |
| Go tests macOS | 120851998590 | success |
| Go lint | 120851998636 | success |
| Go tests Windows | 120851998651 | success |
| Go tests Ubuntu | 120851998678 | success |
| Build Ubuntu | 120852074420 | success |
| Build Windows | 120852074433 | success |
| Build macOS | 120852074455 | success |

Ubuntu 普通 Go 测试带 `-race`；单独 root 跨账户步骤实际执行并通过，与普通步骤因非 root 跳过的跨账户用例分开。旧 Git、unsafe Git、平台权限及默认关闭的诊断等按环境/设计跳过，不记作本轮新验证通过。

**常规 CI 不运行独立 prototype 模块**；不能用 8/8 替代原型或 §3 新反例。纯文档收尾没有新的原型运行，但原型子树和 workflow 相同，故固定代码诊断仍适用。

### 7.3 首轮、取消与历史

| 运行 | 身份 / 事件 / attempt | 结果与适用性 |
|---|---|---|
| [40084348430](https://github.com/XKHoshizora/specmetry/actions/runs/40084348430) | 61a43db / push / 1 | **failure**。三平台回归先行拒绝未成立；macOS 另有两项诊断字段/错误链检查失败。完整 macOS 日志未出现新的自发 EINVAL |
| [40084351128](https://github.com/XKHoshizora/specmetry/actions/runs/40084351128) | 61a43db / pull_request / 1 | **cancelled**，不算通过 |
| [40084692941](https://github.com/XKHoshizora/specmetry/actions/runs/40084692941) | 56234c3 / push / 1 | 最终代码原型 **success** |
| [40084696154](https://github.com/XKHoshizora/specmetry/actions/runs/40084696154) | 56234c3 / pull_request / 1 | 常规代码 CI **success**，不冒称文档 HEAD 的运行 |
| [40087043315](https://github.com/XKHoshizora/specmetry/actions/runs/40087043315) | 2414ae9 / pull_request / 1 | 最终 HEAD 常规 CI **success** |

没有把中间提交 419e4f3 写成另有已核实的专属 CI。旧 40072151282、40074633075 的失败及更早 N6 读数、常规取消记录继续按历史保存；本轮绿色不关闭 M-14/M-15/M-16，不推定其根因消失。

## 8．状态、合并条件与最小下一步

**技术修正**：只需先修 §3 的终态合并/互斥判断及精确组合回归，保留现有可靠 confirmed/refuted/invalid 分类。不要通过取消合法反证、让所有输入 invalid、扩大归属或忽略报告失败关闭问题。

**证据补齐**：增加“ok 后真实写入失败、epipe 写成功”的实际 helper 路径，核对内层 invalid、非零退出、外层正确通过、正常清理及健康对照。到达次序应覆盖等待期间与最终检查；修复后再核对适用诊断和常规 CI。这里是后续实施/独立核验建议，本会话没有重跑远端或授权实施。

**非阻塞收尾**：N1 诊断标签可按真实启发式收紧；旧 macOS 2088/1989 继续登记为未解决历史观测。若未来出现新的原生失败，再分析当次身份与阶段，不接受风险、不放宽保守契约，也不要求本 PR 为不可恢复的旧事实制造答案。

**冻结前门槛**仍包括 14 项负责人产品选择、C-4/D20 的挂起资源与宿主后续调用策略、平台机制支持范围、Linux 全局闸门/C-9 生产拓扑、macOS 机制与 ABI、Windows 停止/句柄前提及 T5 入口/PATH 等尚待定清的规格与必要原型。没有把本轮门禁修复自动视为 T1～T6 全部满足或 C-1～C-20 全部冻结。

**获准生产实施后、工作包验收前**仍需生产 Runner 接线、记录/事务、取消、固定配置、空批次、§8.6 持久化组合及真实 CLI/跨平台验收。本原型没有证明这些能力；也没有把这些未授权生产工作增列为当前原型 PR 的 BLOCKER。

当前两个最终 CI 的正式状态均成功；**合并的技术条件仍未满足，PR #34 不建议合并**。技术 NO-GO 的最小理由是已复现的 B-N4。不能以旧 macOS 历史未知永久扣留，也不能以本次绿色放行新的确定反例。

本审查仅新增本报告，未修改实现、被审文档或历史审查，未提交、推送、合并、改基、触发/重跑 CI、安装工具或执行用户项目配置。未批准产品方案、保证调整、接受风险、规范冻结或生产实施；未宣布 WP2.5/M2 验收，维护事项保持原状。

**落盘后最终核验（2026-10-24 12:24:58 UTC）**：远程 HEAD/base 仍为本报告的固定对象，PR open、未合并、auto_merge=null。本地仅有未跟踪的 `docs/reviews/codex-review-084.md`；原有跟踪文件无差异，Review 083 哈希未变，仓库相对链接目标均存在，未发现本轮测试进程残留。该核验没有扩展审查对象。

