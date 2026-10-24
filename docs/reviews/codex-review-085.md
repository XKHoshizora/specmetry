# Codex Review 085：PR #34 证据收集、最终分类与交接完整性的定向独立复审

## 1．结论

**技术 NO-GO，当前不建议合并 PR #34。**

Review 084 的 **B-N4 精确冲突反例已修复**：完整写出的 ok 与 epipe 不再因到达轮次或记录顺序而绕过检查；N1 的诊断标签也已准确收紧，可以关闭。

但 N4 系列尚未收口，发现同一收集链的新 **BLOCKER B-N4-R：把持有者被终止后的报告稳定，等同于工具已经完成交接**。完整 epipe 记录存在时，报告操作随后失败，或者本次已到达的工具失败报告被终止打断，内层仍可 **confirmed / exit 0**。统一分类正确处理了“已经收集到的错误”，没有证明错误都已收集。详见 §4。

| 问题 | 裁定 |
|---|---|
| 原 B-N4 能否关闭 | **原有完整记录的冲突旁路可以关闭**；这不等于整个 N4 收集链已无阻塞 |
| N4 系列分类是否收口 | **否**；B-N4-R 仍会使工具无效的实验得到绿色结论 |
| N1 能否关闭 | **可以**；标签、函数及文档明确为短命令名前缀启发式，不是身份或归属证明 |
| 观察终点表述 | 需要修正：观察期间已发生的失败、报告尚未完成与观察后的无关行为不能合并排除 |
| 新 BLOCKER | **B-N4-R：缺少工具完成证明，终止可能截断证据** |
| B1、N6 | 未发现本轮直接回退，保留已有关闭范围 |
| 最终原型诊断 / 常规 CI | 均正式 success；不覆盖本轮独立组合反例 |
| 合并建议 | **暂不合并**，先修本轮明确的交接缺口并补相应回归 |

没有新的产品待决取舍需要由本审查代批，也没有新增接受风险。14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14、M-15、M-16 保持未解决。

## 2．身份、范围与记录

| 对象 | 身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| Review 084 对象 | `2414ae939b9a47c73fc7b440bf73d7afa520463a` |
| 归档 Review 084 | `df819911d96a75b5e40c104094b406ec01d4cb83` |
| 回归先行 | `7307d5d20835d0947caf21e3216f1130e75b359f` |
| B-N4 修复 | `173d2ebfa3745ca1a800de1bdf2d91d284973d8d` |
| 最终代码 / N1 | `1e82fbdc21edf12255c7e72a2e981e5c0a6a648d` |
| 文档 | `711c0d3e818281ce787e1641e68a960fffddc3ed` |
| HEAD | `4d1e5754175a9b93d7c73faf9999f13ee6f2b3f0` |

**开始核验：2026-10-24 13:47:07 UTC。** 远程 HEAD/base 与固定对象一致，PR open、未合并、auto_merge=null。本地 HEAD 同为固定对象，工作区干净，报告不存在。**13:58:12 UTC** 技术核查后复核身份和两套最终 CI，均未变化。没有切换对象、改基、fetch 到原仓库或改动其他任务文件。落盘后另记最终核验。

完整读取适用 [AGENTS](../../AGENTS.md)、[Review 084](codex-review-084.md)、本轮增量、[原型文档](../wp2.5-prototype.md) §2.16、[处理记录](../review-resolution.md) §100，以及观察、报告、helper、管道、进程结束、场景终判、门禁和相关清理调用链。

相对上轮 **12 文件，+899/−109**：

- 文档 5 份：AGENTS、implementation-plan、review-resolution、新增 Review 084、wp2.5-prototype。
- 原型 7 份：新增 f1_check；修改 census_check_darwin、census_darwin、gate、helper、pipe、report。

**1e82fbd → HEAD 两个提交只有四份文档变化**：AGENTS、implementation-plan、review-resolution、wp2.5-prototype。prototype/.github 差异为空，原型子树均为 `dcb86c5311e7b8ad94ef8477a2befe9e4998a2c5`。

生产代码、Schema、迁移、根依赖、ci.yml、原型 workflow 与既有冻结契约未改。review-resolution 只追加 §100（92 行），§84～§99 未改写。Review 084 SHA-256 为 `a97d7515c4833a86cb99dac7a48d288b1404c534ed3dffa92afc6ae814a6f41a`，与归档登记一致。旧受审 SHA、NO-GO 和各项边界完整保留，实施方“已修复”仍明确待独立复审。

## 3．原 B-N4：完整记录的冲突与分类旁路已修复

### 3.1 调用链与实际检查

| 环节 | 本轮判断 |
|---|---|
| 初始归属 | `awaitOwned` 经原有树/Job 等核验得到持有者；后续使用该 holder 的 PID |
| 收集 | [pipe_test.go:269](../../prototype/wp25/pipe_test.go#L269) 的 `observeF1` 等待记录；没有时询问、等回答、共享 2 s 读两路 EOF，收集到的错误持续追加 |
| 结束未确认 | [finalRead:319](../../prototype/wp25/pipe_test.go#L319) 增加失败；即使随后读到合法记录也不能恢复成功 |
| 最终读取 | 持有者结束后调用严格解析；截断、畸形、重复、limit 等错误保留 |
| 记录关联 | [375 行](../../prototype/wp25/pipe_test.go#L375) 核对每条 epipe/ok 的 PID 等于已核验 holder；这不是写者认证 |
| 语义冲突 | [383 行](../../prototype/wp25/pipe_test.go#L383) 统一拒绝 ok/epipe 共存，不依赖记录排列或当初在哪次轮询出现 |
| 合法反证 | 仅 ok 时仍要求被询问、每路 EOF、读端齐全和每路目标时段迟到字节 |
| 失败优先 | [404 行](../../prototype/wp25/pipe_test.go#L404) 在检查完各规则后先返回 invalid；已收集错误不会被成功最终读取清除 |
| 全场景终判 | `scen.done` 还严格检查报告允许种类/数量及清理、观察、遗留任务；不允许的其他记录不能靠中途分类恢复最终成功 |

两路 EOF 仍使用一个共同 `endBy`，没有逐流续预算。迟到字节仍从停止时待取字节数扣除旧缓冲，不以任意非零计数代替逐流证据。正常受控 holder 收到询问后关闭两路输出，再给唯一 ok；保留读端不会被误作写端。

统一分类解决的是**已经形成并收集到的证据之间如何裁定**；§4 是独立的收集完成前提，不能由此表外推。

### 3.2 精确反例、记录顺序与正对照

在仓库外，独立导出回归先行 `7307d5d` 与固定目标：

| 真实 helper 路径 | 旧分类 | 固定目标 |
|---|---|---|
| Review 084 精确反例：ok 后真实 stderr 写失败，epipe 报告成功 | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| 关闭输出后真实写失败，先报告 epipe，再回答 ok | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| 未询问的 ok 旁边出现真实 epipe | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| epipe 写的是其他 PID | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| ok 写的是其他 PID | refuted/1，外层 FAIL | invalid/1，外层 PASS |
| 健康 F1 | confirmed/0 | confirmed/0 |
| 合法保留读端反证 | refuted/1 | refuted/1 |
| 正常排空 | 既有对照保留 | confirmed/0 |

旧树五项失败独立复现；新树相关门禁均实际执行并通过。真实不同到达顺序由前三项 helper 路径支持；没有用反序表测试冒充动态到达顺序。

[TestF1Classify](../../prototype/wp25/f1_check_test.go) 的 **19 行固定期望**覆盖正常记录、两种冲突、错误叠加、PID、重复、未结束和不完整反证等；每行另以记录逆序计算。期望不是由被测分类器生成，三种输出均保留。它足以支持分类规则本身，不能证明输入 `ev.failures` 已包括 helper 内部的一切相关失败。

## 4．BLOCKER B-N4-R：结束进程不能补足工具完成证明

### 4.1 具体缺口与依据

[pipe_test.go:203～212](../../prototype/wp25/pipe_test.go#L203) 的顺序是：

`observeF1 → tr.kill → tr.confirm → reap → finalRead → classifyF1`

如果第一轮已经看到 epipe，[278 行](../../prototype/wp25/pipe_test.go#L278) 不再询问或执行 EOF 交接。随后主动结束持有者，仅凭进程结束使 `finalRead` 的 `ended` 为 true。

[315～317 行](../../prototype/wp25/pipe_test.go#L315) 因而宣称“不再有写者，所以报告包含观察可能有的全部”。前半句只能证明**此后不会再追加**，不能证明：

- 产生现有 epipe 的报告调用已经成功返回；
- helper 的工具失败没有发生；
- 当时尚待写出的必要失败报告已完成。

[helper_test.go:341～349](../../prototype/wp25/helper_test.go#L341) 本来会返回报告 Write/Close 错误；[176～198 行](../../prototype/wp25/helper_test.go#L176) 也把报告失败与正常 epipe 结束区分为 `exitReport` 和 3。但父端没有把该工具完成/错误结果作为接受证据的必要条件；读取到完整 epipe 后，进程自然错误退出或被父端杀掉都能留下同一份“干净报告”。

本轮已有 `report.epipe-limit` 明确规定 **epipe 旁的工具失败要 invalid**。下面的组合只让该报告变迟或真实失败，不能把它改成观察完成后的无关行为。原型整体的有效性定义同样要求工具失败不计 confirmed。

### 4.2 独立反例一：完整记录之后报告 Close 失败

外部副本沿既有 `epipe-limit` 注入选择运行真实 holder：

1. 保留真实晚到输出写失败，及真实 epipe 记录的完整 Write。
2. 在 `report("epipe")` 的 Close 位置，先正常 Close，再调用一次 Close 得到真实 `file already closed` 错误，并按原有返回路径把错误返回。
3. 只用外部夹具标记把调度固定在**错误已经发生、父端尚未终止/分类**；标记只用于确认负载和时序，不注入 verdict、不写入 `ev.failures`。
4. 保持原分类、门禁预期、确认与清理不变。

| 证据 | 结果 |
|---|---|
| 报告 | 一条合法完整 epipe，PID 属于已核验持有者 |
| 实际错误 | 报告 Close 返回 `file already closed`；故障发生在终止/分类前 |
| 内层结果 | **confirmed / exit 0** |
| 内层清理 | 父 36360、holder 53788 均核实结束 |
| 外层原门禁 | 要求 invalid，得到 confirmed/0，**正确 FAIL** |

这不是原生 Close 自发失败的复现，而是**人为第二次 Close 触发真实错误返回**，用于核验现有 API 错误通路；与原型已有关闭错误故障模型同类。没有声称操作系统自然发生了该故障。未读取持有者的自然退出码，不把源码的 `exitReport` 分支说成已动态取得其退出码。

完整字节可读与整个报告操作成功是不同事实。原型当前把前者误用为后者，形成绿色假通过。

### 4.3 独立反例二：工具状态已失败，失败报告被终止打断

沿**现有 `epipe-limit` 工具期限故障点**，保持真实 epipe 产生与报告：

- 已到达该既有模拟工具期限故障点并写本轮夹具标记；
- 在随后真实 `report("limit")` 调用前等待可释放闸门；
- 父端确认该点已到达，继续原有主动终止、确认结束、最后读取与分类。

结果：报告只剩完整 epipe，内层 **confirmed/0**，外层要求 invalid 因而 FAIL；父 20904、holder 16324 均核实结束。是**被父端终止阻止了后续报告**，不是失败尚未发生。

这是现有期限故障点的确定性调度，不是声称真实 60 s 时钟已到期；没有伪造最终结论。调度有外部硬期限，闸门可释放，所有自有进程得到收尾。

同一夹具的反向对照：终止前放开闸门，让真实 limit 报告完成，再继续同一最终分类。此时内层 **invalid/1**、外层 PASS，正常排空也 confirmed/0，整个补跑 exit 0。**只改变工具报告是否被终止截断，分类便从错误绿色变为正确拒绝。**

### 4.4 独立反例三：epipe 已在，后续工具报告真的写失败

保留现有 `epipe-limit` 场景，把其中 limit 报告以只读方式打开，执行真实写入得到错误；父端标记确认该错误已经发生才继续原流程。此前 epipe 正常写入。

内层仍 **confirmed/0**；外层期望 invalid，正确 FAIL；父 12852、holder 70660 均核实结束。这组合了已有“工具期限失败”与“报告写失败”模型，不是任意不可信程序协议。真实写入错误不因没有成功写成 limit 记录而不存在。

三项实验均保留健康 confirmed、可靠反证 refuted 对照。没有修改内层 verdict、expected、分类器或外层判定来取得结果。**错误的是内层绿色，外层发现它而失败是正确的**；没有声称整份变异后的诊断仍绿色。

### 4.5 观察终点与最小修正

[原型 §2.16 的边界](../wp2.5-prototype.md) 称：“持有者在被结束之前没有写出的记录不在报告中……按协议此后的行为不属于这次观察。”该句不能覆盖：

| 类别 | 本轮判断 |
|---|---|
| A．明确完成交接之后的无关行为 | 可以在受控协议中限定范围；没有要求抵御任意未来恶意程序 |
| B．观察/交接期间已发生，但报告尚未写出或报告调用失败 | **不能排除**。§4.2、§4.4 为具体反例 |
| C．父端终止阻止报告完成 | **不能用杀后报告干净证明完成**。§4.3 是具体调度 |

本轮 epipe 路径甚至没有回答+EOF 的正常终点，不能把只在 ok 分支成立的终点用于豁免它。EOF 关闭输出流，不等于其他报告操作成功；最终读文件的格式/数量正确也补不出失去的语义事实。

**最小修正范围**是当前 helper/report/F1 收集交接：为成功接受证据定义可验证的工具完成/失败状态，在 epipe 与 ok 两条路径都适用；主动终止只能作为清理，不能自动认证观察有效。没有取得该状态、报告调用失败或仍可能被终止截断的必要报告，应保守 invalid。

可以利用受控 helper 的完成状态、可核验退出结果或明确收尾握手，具体实现由实施方选择；不能只增加一次读取、固定延迟或把未报告错误重新划出区间。无需建立无限层证明、生产服务、任意恶意程序认证或提前完成生产 Runner。

必要回归是上述完整记录+报告失败、已到达失败点+报告未完成两组，加上成功交接、真实 epipe、合法 ok/EOF/字节反证。当前新 BLOCKER 针对确定的绿色误判，不是因为“还有一般性未知”。

## 5．N1、macOS 历史与直接回归

**N1 可以关闭。** [census_darwin_test.go:76](../../prototype/wp25/census_darwin_test.go#L76) 已改为 `short command name a prefix of this test's (heuristic, not identity)`，函数改名 `commPrefix`，注释明确不同程序也可匹配。相应直接检查和活跃文档同步。没有增加程序名、参数或环境回显，没有把匹配值用于归属或信号授权；普查失败仍保守返回。

macOS 旧 PID 2088（被测普查失败）与 1989（保险普查失败）的归属、EINVAL 原因和当时完整清理继续未知。本轮代码与文档没有用当前绿色回填；没有要求反复运行至 EINVAL 再现，也不把不可追溯当永久合并门槛。

本机无法运行 macOS；N1 的动态证据来自当前 macOS CI，其两项 `TestCensusUnreadable` 实跑通过。它们仍是自有 helper 上的错误接缝检查，不是原生 EINVAL 重现，亦不关闭旧观测。

本轮抽查 B1/N6 直接回归：

- 晚到停止错误、挂起任务移交、Windows Cancel/Close API 错误检查实际通过；没有恢复阶段 J 前同步阻塞或丢弃任务。
- 同一阶段时钟、固定 D/J 预算、恰好截止/差 1 ns/提前唤醒保持；没有用 stopwatch 替代阶段判定或增加容差。
- forge-owner 对照仍拒绝非内层对象，零动作/零信号；混合场景本轮自有遗留仍实际清理。
- 补救失败、观察错误、状态未知、保险矛盾等原有组合仍使被测自检失败；成功清理不能恢复失败结论。

保留此前 B1、N6、归属与组合门禁的关闭范围；不重新审查全部 T1～T6。§4 是本轮收集/最终分类链的直接缺口，不是重开整个 PR #33。

## 6．独立运行、失败与验证边界

实际平台：**Windows 11，10.0.26300，amd64，Go 1.27.1**。使用已安装工具，`GOTOOLCHAIN=local`、`GOPROXY=off`；所有副本位于仓库外本轮独立目录，从固定 Git 对象导出，不改原工作区。

| 本轮独立运行 | 结果 |
|---|---|
| 固定目标，240 s 硬期限 | **PASS**：3 个顶层机制场景、63 门禁、13 组合；F1Classify、StopRead、StopAPIErrors、LingeringFailedStop、PhaseClock、Candidates、GateJudge 等直接检查通过 |
| 回归先行 7307d5d，45 s | 五项预期失败独立复现：四项错误 confirmed/0、一项错误 refuted/1；健康/合法反证正确 |
| 延迟报告夹具，30 s | 内层错误 confirmed/0，外层 FAIL；自身清理完成 |
| 工具报告真实写失败夹具，30 s | 同上 |
| 完整报告后 Close 失败夹具，30 s | 同上；实际 Close 错误在终止/分类前发生 |
| 放行报告的反向对照，20 s | 内层 invalid/1，外层 PASS；另带正常排空，整个补跑 PASS/0 |

固定目标定向选择：

`^(TestGate|TestGateCombos|TestF1Classify|TestStopRead|TestStopAPIErrors|TestLingeringFailedStop|TestPhaseClock|TestCandidates|TestGateJudge|TestPipeNormalDrain|TestPipeEscapedHolder|TestPipeF1SkipDrain)$`

只把本机 3 个机制场景记为 3，未说成全部 29；门禁 PASS、内层 invalid、机制 confirmed、组合与固定表测试各自统计。实验启动/报告均沿真实 helper 链路；夹具标记只证明注入命中与顺序。所有操作对象归属已由本轮协议确认，未向未知 PID 动作；结束核查未见 review085 测试进程残留。

**准备失败与补跑如实保留**：

- 第一次编译误在仓库根目录执行，因无 Go 文件 setup failed；没有运行场景或生成实现变更。随后在外部原型模块正确编译。
- 初次延迟夹具文本锚点未匹配，脚本在修改前拒绝；调整定位后运行。不是检出产品问题。
- 放行对照首次补接文本缺一个换行，编译失败，不计有效检出；修正夹具后实际执行。
- 放行对照第一次只选门禁，虽该门禁 PASS，汇总因无顶层机制场景而 exit 非零；没有把它报成整次通过。保留该日志，补选正常排空后整体 PASS/0。
- 一次未加引号的 Git tree 参数被 PowerShell 错误解析，未用其输出作证；改用正确引用读取 tree。
- 未安装工具、未运行用户配置，未触发或重跑远端 CI。

仓库外证据 SHA-256：

| 日志 | SHA-256 |
|---|---|
| fixed-run.log | `69ec90783f4bd02d2825b5e6dfd5e2a1abb3b2676c9552a6ff540680a7237606` |
| old-run.log | `0e6f3906c0ccf5e5a56b9bf054f85e151913241da44e078b58569499954fd3d2` |
| delayed-run.log | `5691971079952be8f97f07eca11426b14739a0bb191ae241ed637ecbaa57b410` |
| reportfail-run.log | `19dd11684eb123fe2fd1d7053b6b4a5cea0a55a32c03fc6f512f5e21a9255653` |
| closefail-run.log | `ef466bf9083fa4392d294e6d48ad2dccfaf515c74f606381818adb695b5165c1` |
| released-run.log（补跑） | `58400b1602c9751d6859207c6acd7c1d58750de785f420424e464b164c9b9c75` |

作者 M60～M62 的 Windows/WSL 撤回检验属于辅助证据，未冒称本轮独立重做。作者披露的“各自检出”只支持对应已输入的冲突、PID 和收集错误；M62 下最终读取失败导致没有记录，本身仍 invalid，文档没有误算其为 M62 检出。本轮未把任何编译失败计成变异检出。

没有本机 Linux/macOS 动态复现、原生存储故障、任意 PID 复用或生产 Runner 验证。Close 反例为明确错误注入，期限反例为已有模拟故障点调度；真实报告写失败与这些注入的性质分别记录。

## 7．CI 与提交适用性

已取得两套最终运行全部 11 个作业的完整日志及正式汇总，核对实际 checkout、事件、attempt、相关用例和范围。没有仅以作者回执、PASS/SKIP 数量相同或 Job 看起来绿色代替正式状态与具体测试名。

### 7.1 固定代码原型诊断

[run 40096356396](https://github.com/XKHoshizora/specmetry/actions/runs/40096356396)：**push，attempt 1，completed/success**；三个 Job 实际检出均为 `1e82fbdc21edf12255c7e72a2e981e5c0a6a648d`。

| Job / ID | 机制场景 | 门禁 | 组合 |
|---|---:|---:|---:|
| Ubuntu / 120879325028 | 38/38 confirmed | 67/67 PASS | 13/13 PASS |
| Windows / 120879325082 | 29/29 confirmed | 63/63 PASS | 13/13 PASS |
| macOS / 120879324838 | 27/27 confirmed | 58/58 PASS | 13/13 PASS |

场景 ID 去重并集为 **53**，不是平台数相加；Results 重复打印已去重。Ubuntu 实际 Linux 分支设置 `race=-race` 并执行对应 go test，未报告竞态。另两平台不是 Race Detector 证据。

三个顶层内层专用 SKIP 为 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError`，相应子进程场景确有实际运行。没有把 SKIP 计为通过。`TestF1Classify` 的 19 行、StopRead、PhaseClock，以及适用平台 StopAPIErrors / CensusUnreadable 等直接检查实际通过。

本轮新增 12 项门禁实际得到 invalid/非零，外层接受；健康 confirmed、合法 refuted、epipe-late confirmed 均保持。它们没有覆盖 §4 的“工具操作失败但完整记录仍在”或“必要报告被终止截断”的组合。

### 7.2 最终常规 CI

[run 40098220152](https://github.com/XKHoshizora/specmetry/actions/runs/40098220152)：**pull_request，attempt 1，completed/success，8/8**，HEAD 为固定目标。

八个作业实际检出完整合并快照：

`62ea3421481c161fd675d22a10795cf2fd7fed55`

GitHub commit 元数据父提交依次为固定 base 与固定 HEAD，树：

`00481cff45f3aea8ebe30ee346dcc404c4b2f698`

与本地固定 HEAD 树一致。不是只核对缩写或作者所报树。

| Job | ID | 结果 |
|---|---|---|
| Web | 120884831232 | success |
| Go tests Windows | 120884831326 | success |
| Go tests macOS | 120884831370 | success |
| Go tests Ubuntu | 120884831379 | success |
| Go lint | 120884831404 | success |
| Build Windows | 120884915332 | success |
| Build macOS | 120884915363 | success |
| Build Ubuntu | 120884915364 | success |

Ubuntu 普通测试执行带 -race 的命令；单独 root 跨账户步骤执行并通过 8 项，与普通步骤因非 root 跳过这些用例分开。平台/旧 Git/默认关闭诊断等设计 SKIP 不记为本轮新通过。

**常规 CI 不运行原型独立模块**。固定代码到 HEAD 只有文档变化，故代码点诊断适用于原型内容，但两套正式绿色都不覆盖本轮独立组合反例。

### 7.3 回归先行、取消与历史

[run 40095937741](https://github.com/XKHoshizora/specmetry/actions/runs/40095937741)：7307d5d，push，attempt 1，**failure**。独立核对三个 Job 的实际 checkout 与完整日志：均只有上述五项新回归失败，Ubuntu 60/65、macOS 51/56、Windows 56/61；机制 38、27、29 confirmed，组合各 13/13。未把这些预期失败写成从未失败。

同代码点的常规 [40095938462](https://github.com/XKHoshizora/specmetry/actions/runs/40095938462) 为 **cancelled**。最终代码常规 [40096359125](https://github.com/XKHoshizora/specmetry/actions/runs/40096359125) 正式 success；最终 HEAD 适用性仍以 40098220152 为准，不混记 SHA。没有声称中间修复 173d2eb 有另一次独立核实的专属诊断。

此前失败、取消、旧 macOS 未知与旧 N6 读数均保留，不用本次成功关闭维护事项或确认其根因。

## 8．合并建议与后续范围

**当前原型必要修正**：处理 §4 的工具完成证明与终止次序，不能仅让已经落到文件里的记录一致；增加完整记录+报告操作失败、已到达失败点+报告被终止两组端到端回归，保留正常 confirmed/refuted。相应更正 §2.16“没有写出的记录不属于观察”的表述，明确哪些是已完成后的无关行为，哪些是未完成的本次报告。

**非阻塞历史观测**：旧 macOS 2088、1989 的归属、根因、完整清理继续未知。N1 已关闭，不需重开 macOS 归属规则；不要求反复追逐偶发 EINVAL，不批准忽略错误、自动重试或接受未知清理风险。

**冻结前规格/必要原型**：14 项产品选择、C-4/D20 的挂起资源与宿主后续调用、Linux 全局闸门/C-9 拓扑、平台支持及 macOS ABI、T5 入口/PATH 等既有剩余门槛继续分别处理。没有因门禁修复而认定 T1～T6 全部成立。

**授权后、生产验收前**：生产 Runner 接线、记录与事务、取消、配置固定、空批次、§8.6 持久化组合、CLI 与 E2E 仍属于后续工作；没有将这些未授权生产任务自动升级为本 PR 的阻塞。

最终裁定是：**原 B-N4 的具体冲突修复成立，N1 关闭；N4 系列仍有确定绿色反例，新增 B-N4-R，技术 NO-GO，当前不建议合并。** 当前 CI 状态正式成功，但技术条件未满足。原型缺陷、历史未知、冻结门槛与生产工作已分开，不批准保证调整、接受风险、规范冻结或生产实施。

本轮仅新增本报告，未修改原仓库其他文件，未提交、推送、合并、改基、触发/重跑 CI、安装升级工具或执行用户项目配置。未宣布 WP2.5/M2 验收，维护事项保持原状。

**落盘后最终核验（2026-10-24 14:02:26 UTC）**：远程 HEAD/base 仍与固定对象一致，PR open、未合并、auto_merge=null。本地仅新增未跟踪的 `docs/reviews/codex-review-085.md`，跟踪文件无差异；Review 084 哈希未变，报告仓库相对链接目标均存在，未发现本轮 review085 测试进程残留。没有扩大审查对象。

