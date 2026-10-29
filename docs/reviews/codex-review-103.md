# Codex Review 103：PR #41 最小观测与补证的定向独立复审

## 1．结论

**技术 GO，建议合并本报告的固定对象。N-102-1 的最小观测缺口可以关闭；Review 102 留下的当前合并证据差额已补齐。没有发现新的 BLOCKER。**

本次结论由三部分共同支持：新增列表保存同一内层运行中已成功解析的最终结果，不改原门禁判定；两个故障组合及健康对照验证信息确实得到保存；同一代码点的两种 macOS 架构均重新取得了布局有效、目标执行、K1 接缝命中并按预期拒绝的完整证据。四作业原型诊断及最终常规 CI 正式成功，且适用于当前 HEAD 的内容。

| 问题 | 裁定 |
|---|---|
| A．N-102-1 | **关闭**，范围是最小可追溯性缺口，不是任意原始输出的无损归档 |
| B．新增观测及回归 | **技术 GO**。故障组合通过证明原门禁正确拒绝且证据保留，不算原生机制 confirmed |
| C．固定 PR 合并建议 | **建议合并**。此前缺少的有效 arm64 K1 验证已取得；旧运行仍为 failure、旧原因仍未知，不要求永久追溯不存在的数据 |
| D．下一步冻结审查 | **证据足以作为输入进入下一步冻结审查**；本轮不代负责人启动该审查或批准冻结 |
| E．G5 整体 | **不认定整体满足**。生产入口 E1/E2、支持与已测组合差额，以及其他 F 类规格/候选仍须冻结审查 |

Review 101 的 B1/B2/B3 按 Review 102 的范围保持关闭；N1（显式结果集合/重复拒绝）、N2（改写前原始值结构）仍未实现，保持原非阻塞裁定。没有新增产品决定、保证调整或风险接受。Q-3(a) 及其他已批准决定保持不变。

## 2．身份、归档与完整增量

开始核验：**2026-10-29 20:03:42 JST（11:03:42 UTC）**。

| 对象 | 固定值及核验 |
|---|---|
| PR | [XKHoshizora/specmetry#41](https://github.com/XKHoshizora/specmetry/pull/41)，open、未合并、`auto_merge=null` |
| HEAD | `ab4b7dd27bdc66d1e393cb6b8818244772a6724c` |
| base / master | `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b` |
| HEAD 内容树 | `a29b84ef1764ef3bd6470a93e375fd3b9790bd34` |
| 固定原型代码点 | `9d450365cb11f343a634f47d21cf962c9dddec35` |
| 上轮对象 | `fbcef6fd230cedd60bc4943a6192e3eb1a875698` |

远程 PR、master 与本地对象相符。本地分支 `test/wp2.5-prototype-p3`，开始时工作区、跟踪文件与暂存区均无改动。读取了根 `AGENTS.md`、完整 Review 102、本轮全部九文件差异、原型文档 §15.12、处理记录 §126 和受影响调用链。仓库内没有更深层 AGENTS.md。

| 提交 | 实际范围 |
|---|---|
| `5e16cb64f98ae31390640d6f8942cc71172484ff` | 仅原样归档 Review 102 |
| `9d450365cb11f343a634f47d21cf962c9dddec35` | 三份 test-only 文件：`gate_test.go`、`gate_combo_test.go`、`p3_darwin_test.go`，共 +151/−3 |
| `ab4b7dd27bdc66d1e393cb6b8818244772a6724c` | 五份文档：AGENTS、implementation-plan、review-resolution、wp2.5-check-runner、wp2.5-prototype |

Review 102 为 **31,064 字节**，SHA-256 `84d75601bb0cb640b41bb32a48ae856c809aaafacc60aa408925deb93c4abcbb`，与归档登记一致。§126 仅追加在旧处理记录之后，没有改写 §125 或历史报告。代码点到 HEAD 的 `prototype/`、`.github/` 差异为空。

本轮没有修改 workflow、平台、权限、并发规则、次数、期限/预算、生产代码、根依赖、Schema、迁移或产品语义。规范正文仅同步状态，没有以新证据自动批准技术候选或更新支持承诺。

## 3．inner/all 的数据与判定边界

主要位置：`prototype/wp25/gate_test.go:138`、`:523`、`:537`、`:588`、`:736`；结果结构在 `main_test.go:101`，最终生成在 `scen.done`（`:317`）。

### 3.1 保存的是哪些结果

`runInner` 仍按捕获输出的行序扫描。每条 WP25RESULT 成功解码为既有 `result` 后，同时：

- 写入原有 `results[ID]` 映射；
- 追加到新 `all []result`，不排序、不去重、不按目标或 verdict 过滤。

`runGate` 在检查目标、because、命中、exact 之前，直接将 `r.all` 赋给 `gateResult.Inner`。因此 because 不符、目标 invalid 或未命中，都不再造成前提场景的信息从外层结果中消失。重复 ID 在列表中按打印顺序可见，映射仍是后值覆盖；**可见不等于拒绝**，N1 没有因此实现。

保存的既有结构包括 ID、Gate、Kind、Platform、Verdict、Expected、Observed、Launches、Work、Cleanup、Reason、Millis、Owned、Shortfall、Invalid；没有只挑出目标的摘要。`omitempty` 仍按既有 JSON 规则省略空字段，不等于非空错误被删除。

来源仅为收尾后的 **WP25RESULT**。`runP3` 先打印 WP25P 的提前证据，再调用 `s.done()`，最终关闭错误进入后者；新读取路径不消费 WP25P.T 来替代最终结果。布局最终 verdict 写入前提状态的次序也没有改变。

### 3.2 “全部、原样、最终”的准确口径

本轮认可的口径是：**同一次内层运行中，所有已成功解析的既有 result 字段值，依打印顺序保存，来源为最终 WP25RESULT。**

它不是原始行字节归档：JSON 会重新编码，未知 JSON 字段不进入当前结构；没有打印的结果、解析失败的原始行和完整工具输出也不在列表内。当前生产者正是既有 `result`，没有发现本次遗漏实际字段。§15.12 末尾已披露“只保存打印的 WP25RESULT”、未打印/超时的限制；“原样”不能扩大解释为未知字段或原始字节无损保留。以后精化这一词语是非阻塞建议，不妨碍当前缺口关闭。

畸形 JSON、Scanner 超长/读取错误仍使 `r.err` 非空；`runGate` 的错误分支令判定失败，已解析的部分列表不能恢复成功。目标结果缺失仍拒绝。新列表没有改变这些分支，也没有取消原有 1MiB Scanner 限制。语义上的完整集合/重复检查仍按 N1 保留。

新增字段的内容都是既有字符串、整数及切片，非递归、不含不支持 JSON 的值类型，没有新引入 Marshal 错误类型。沿用了既有序列化调用；这不是为序列化失败增加豁免。本轮独立回读检查也确认已保存字段经过 JSON 往返保持一致。

### 3.3 未改变原门禁

所有 `Inner` / `all` 的引用已追踪：生产者是 `runInner`，外层只赋值并输出，新增 `traced` 只检查三个观测组合。原 `runGate`、`judge`、exact、Owned/Shortfall、归属核验、补救、Final 与保险清理没有据此分支或放宽。

`outerFault.env` 只附加给指定内层 Cmd。普通 `TestGate` 仍传零值；witness 文件仍是传入环境的第一项，没有更换登记或动作授权。PATH 存根没有通过 `os.Setenv` 修改宿主或其他场景的环境。

## 4．三个观测回归及直接影响

主要位置：`gate_combo_test.go:147`、`:202`、`:268`、`:311`；`p3_darwin_test.go:597` 的 `lateClose`。

### 4.1 构造、命中与结果

| 组合 | 构造及固定代码点的实际结果（两种 macOS 相同） |
|---|---|
| `combo.trace-layout-tool` | 仅该内层 PATH 最前放 `xcrun` 脚本，真实执行并 exit 1。布局 invalid，自己的理由为 `the tool step xcrun-find-clang ended with exit 1: stub xcrun: exits 1`；布局 launches=1、work=0、进程清理 verified。目标因布局前提不成立未执行，launches/work/hits=0。原门禁 **pass=false**，保留预期的 `hit: the fault never reached its site`；Final ended、补救 not-needed、保险 nothing left。组合 PASS |
| `combo.trace-layout-late` | 将该内层的注入名替换为 `p3-layout-close-late`，不武装 K1。布局取得额外 `/dev/null` 文件，交既有关闭器持有；收尾实际关闭成功后，接缝返回错误。最终布局 invalid，含唯一关闭错误，Cleanup 为 `NOT verified: closing failed ...; verified: 5 processes gone`，launches=5、work=55。目标未执行，命中恰好1且站点为晚到关闭；原门禁 **pass=false**，无额外外层工具错误、Final ended、保险 nothing left。组合 PASS |
| `combo.trace-healthy` | 原 `control.p3-k-exit7`，布局和目标都 confirmed，清理 verified；目标 launches=1、work=1；无注入、无额外问题，原门禁 **pass=true**，组合 PASS |

两个故障组合的 PASS 是对自检 FAIL 及信息保存的确认，不是把故障场景算成机制 confirmed。它们与根层 63 次 P-3 分开。

`lateClose` 同时要求 inner 环境、准确注入名、`layout`、第1次；正常原生场景及其他注入均直接返回。文件成功取得后立即由已有 `s.closer` 接管，只实际关闭一次；实际 Close 若失败，保留原错误，不用计划错误覆盖它。它不调用原始关闭来绕过已有错误保存，也不再额外关闭同一文件。

### 4.2 断言是否有效

组合使用现有 `runGate`，不伪造内层 verdict 或局部 pass。`traced` 的固定期望来自上述三种构造，不由被检查结果反算：

- 声明的两个 ID 必须各出现一次，列表总数恰好2，保存自己的 verdict、Reason 和 Cleanup；
- 目标必须确实执行或确实没有启动/工作，不能用另一个场景的负载代替；
- 命中数量和站点按组合声明核对；健康为零，晚到关闭为一次，工具失败时 K1 为零；
- 原门禁 pass、预期拒绝维度、工具类别、Unexpected、补救、Final 及保险结果同时检查。

工具失败组合特定地允许“目标未到达接缝”的 hit 结果；它没有把这一允许加入原 `judge`。其他两个新组合要求没有工具错误。未知状态、补救失败或保险清理不确认仍使组合失败；没有因保存了布局理由而恢复原门禁的 PASS。

`traced` 按 ID 检查结果，不单独检验列表顺序；顺序来自 `runInner` 的直接追加。本轮独立共享链检查另核验了顺序和序列化往返。它不是 N1 在所有门禁上的实现。

新增组合借用现有门禁，但没有增加普通 `TestGate` 中 K1 的运行次数。它们按所借门禁的平台条件只在 macOS 执行；Linux/Windows 没有执行这三个组合，不能把那里的13个旧组合写成16个。

## 5．补齐的两架构 K1 证据

[原型诊断 40847347022](https://github.com/XKHoshizora/specmetry/actions/runs/40847347022)：**push、attempt 1、正式 completed/success**。四份完整作业日志的 checkout 与 WP25META 均为 `9d450365cb11f343a634f47d21cf962c9dddec35`。

| 平台 | job ID | 根层场景 | 门禁 | 组合 | 正式结果 |
|---|---|---|---|---|---|
| arm64 | 123357986540 | 96/96 confirmed | 99/99 | 16/16 | success |
| Intel | 123357986481 | 96/96 confirmed | 99/99 | 16/16 | success |
| Ubuntu | 123357986066 | 53/53 confirmed | 111/111 | 13/13 | success，Prototype 步骤带 -race |
| Windows | 123357986509 | 53/53 confirmed | 92/92 | 13/13 | success |

### 5.1 同一内层的完整链

逐项读取两个作业中普通 TestGate 的 `p3.k-nostatus-once`，不是借其他运行的布局结果：

| 环节 | arm64 | Intel |
|---|---|---|
| `inner` 列表 | 先 `P3.layout.1`，后 `P3.k.exit7.1` | 同左 |
| 布局最终结果 | confirmed，launches=5、work=55，5个进程的 Owned，Cleanup verified | 同左 |
| 目标实际执行 | launches=1、work=1，1个进程的 Owned，Cleanup verified | 同左 |
| 命中 | 恰好1；`P-3 kevent: fflags 0x84000000 -> 0x80000000, data 0x700 -> 0x700` | 同左 |
| 目标 verdict / 退出 | invalid / 内层退出1 | 同左 |
| 唯一 Invalid | `planned corruption detected by K1: K1: the returned fflags 0x80000000 lack NOTE_EXIT or NOTE_EXITSTATUS` | 同左 |
| 外层登记与状态 | created=6、receipts=1、无 unowned/shortfall 缺口；无问题项、无工具错误，补救 not-needed、Final ended、insurance nothing left | 同左 |
| 原门禁 | **PASS**，约550ms | **PASS**，约679ms |

这补齐了 Review 102 当时缺少的目标执行与真实接缝命中证据。它是对原生 kevent 返回后作计划改写的门禁证据，不是原生内核自行返回缺失 NOTE_EXITSTATUS 的证据。

### 5.2 集合、重印与直接检查

两个 macOS 的根层均为既有33 + P-3 63；后者是 **21个场景族各三次**，不是63种独立机制。63条最终结果全部 confirmed、Cleanup verified；对应63条 WP25P 没有 T 错误。根层正常、计划机制故障、门禁内层 invalid、组合与直接检查分别计数。

Results 步骤会重印根层结果；本轮按根行前缀及 ID 去重，没有递归把 `inner` 中的结果再计作根层。所有平台的每条门禁行均有非空 `inner`，已核对保存值与外层目标摘要的对应。两个 macOS 的三个新组合均实际执行。B1/B2/B3 的九项门禁保持通过；`TestP3Decode`、`TestP3Parse`、`TestEnumBudget`、`TestEnumChain`、`TestKqWaitDeadline` 五项直接检查实际 PASS。

三个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError` 在主运行设计 SKIP，相应内层由原门禁选择，不把 SKIP 算通过。Darwin 专属测试不编译进 Linux/Windows。Ubuntu 确实执行 Linux 分支将 `-race` 传给 go test；不能仅凭无竞态告警概括为所有并发机制已验证。

arm64 实测为 Go1.27.3、macOS26.6.2/25G83、`xnu-12377.161.14~5`（VMAPPLE）、SDK26.5；Intel 为 Go1.27.3、macOS26.6.1/25G76、`xnu-12377.161.13~4`、SDK26.5。架构、SDK布局与实际内核行为是不同层证据，未据此确认固定公开源码就是运行内核。

## 6．旧失败、哈希及当前限制

### 6.1 旧运行没有被修写

[40817305419](https://github.com/XKHoshizora/specmetry/actions/runs/40817305419) 仍为 **attempt1、completed/failure**，代码点 `eb2317fa0be728202ceb705a5bf5aa72eb6ab012`；artifact 列表仍为0。旧 arm64 门禁的 layout 具体原因仍无法恢复，不推定为环境、镜像、本轮修复或观测改动造成。

新 arm64 镜像为 **20261013.0424.1**，失败那次为 **20261021.0445.8**。这证明环境身份不同，不证明镜像是根因，也不能证明错误不会再发生。过去更短的用时、created=5/receipts=0 只支持目标未启动，与 launches=0 一致，不解释布局为何 invalid。

本轮重新只读取得四份旧作业日志，与 Review 102 已读内容相同。按返回日志的 UTF-8 文本（包括其 BOM/换行）计算的 SHA-256 如下，均与文档所列缩略哈希对应：

| 旧作业 | 本轮计算 SHA-256 |
|---|---|
| arm64 /123261288396 | `25ecd74cf52e469bf95f07f79477a93baee13231bdc6cce3a550bb6435892a01` |
| Intel /123261288366 | `175acdaf757b213539fd78613386a8b25b3abb3c60057e6938b3feea99cc619a` |
| Ubuntu /123261288385 | `bac2e8543001f79bf979ae90678265bd84d618b7c28f98122e152bea8fe57c90` |
| Windows /123261288091 | `d21f31497dff0a9cab403e6d8bac57a01e7e08bec76a1c9a57ea5b7f06cc8b92` |

新四份日志的计算结果也分别匹配 §15.12 的 `7686260a…e537`、`6d8403b9…cbe9`、`deb9c5cd…5939`、`7d58110e…fefa`。这核验了可读取的日志内容；**没有读取作者会话临时目录的原件**。run/job/attempt/artifact 的正式字段由 API 独立核对，但作者保存的元数据序列化文件及其字节哈希未独立核验，不能以重新取得一个同义 JSON 响应冒称核对过原文件哈希。

因此可保留旧失败与未知，而不把不可追溯历史作为永久合并门槛。新成功满足的是新固定对象的验证条件，没有回填旧原因或旧完整资源释放结果。

### 6.2 非阻塞的观测限制

| 限制 | 对本次结论的影响 |
|---|---|
| 内层尚未打印最终结果即退出/超时 | 只能保存已解析的部分，不能恢复未发送的信息；原超时/解析/结果缺失及退出检查继续拒绝，不因已有部分 `inner` 放行。本轮不要求建立任意崩溃恢复协议 |
| 工具 stderr 仅保留既有40字符摘要 | 可能不足以诊断底层工具全部原因，但当前已保留工具步骤、退出状态、最终错误；没有削弱 invalid 判定。扩大受控诊断可另议，不要求无差别持久化/回显工具输出 |
| 注入命中按内层计数，以站点文字区分 | 当前单目标布局+K1 拓扑可由源接缝、唯一命中、站点与唯一规则相互核对。未推广为一般性的写者认证或任意并发拓扑绑定；以后扩展时可增加结构化站点关联 |
| N1 显式集合/重复结果拒绝 | 列表让重复可见，但没有改变映射覆盖或原 gate 的集合验证；仍是非阻塞增强，未宣称关闭 |
| N2 改写前 R 结构 | 没有变化；原生读数与改写接缝仍按 Review 101/102 的范围采纳，不把列表保存误称为 R_before 已实现 |

没有发现这些限制在当前新增三种构造或普通 K1 路径上形成新的绿色假通过。没有将“已披露”当作接受新风险，也没有要求抵御任意不可信程序或无限增加证明层。

## 7．最终常规 CI

[40850097986](https://github.com/XKHoshizora/specmetry/actions/runs/40850097986)：**pull_request、attempt1、正式 completed/success，8/8**。

八份完整作业日志实际 checkout 均为：

`a5359d2da17c35869ed69305ee7a74501381b782`。

Git commit API 核对父提交依次为 `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b`、`ab4b7dd27bdc66d1e393cb6b8818244772a6724c`，树 `a29b84ef1764ef3bd6470a93e375fd3b9790bd34` 与 HEAD 相同。

| 作业 | job ID |
|---|---|
| Web | 123367148897 |
| Go tests macOS | 123367149184 |
| Go tests Ubuntu | 123367149193 |
| Go tests Windows | 123367149194 |
| Go lint | 123367149508 |
| Build Ubuntu | 123367318817 |
| Build macOS | 123367318834 |
| Build Windows | 123367318868 |

Ubuntu 普通测试带 -race，root CrossUser 步骤另行编译并运行，八项实际 PASS，不混写为 root race。普通步骤需要 root、真实旧 Git、平台权限及默认关闭诊断等已有跳过按原范围保留。常规 CI 不运行嵌套原型模块，不能替代 §5 的原型诊断。

代码点 CI/诊断适用原型，最终 HEAD CI 适用文档收尾；没有拿其他 SHA 或重复 Results 输出合并统计。没有触发、重跑或取消任何远程作业。

## 8．本轮独立执行、首次失败与边界

环境：Windows/amd64，已安装 Go1.27.1；`GOTOOLCHAIN=local`、`GOPROXY=off`。固定 HEAD 通过 git archive 导出至：

`C:\Users\hoshi\.codex\visualizations\2026\10\12\01a13137-991b-7b42-8ea2-3c24008343a0\review-103`。

本轮没有原生 macOS 环境，三个已提交的 macOS 观测组合以源码和完整 CI 日志核查，未冒称本机运行。另以仓库外补充测试调用现有共享 `runGate`，检查跨进程采集与输出，不实现/扩展 P-3：

| 执行 | 次数、期限及结果 |
|---|---|
| 原样副本 | 一次，go test 90s；`TestPipeNormalDrain`、`TestGateJudge`、`TestWitnessJudge`、`TestStartPoints`、`TestRecordAlive` 均 PASS，包6.429s，退出0 |
| 共享采集探针首轮 | 一次，120s；三个分支均在探针自己的 `Observed` 断言失败，包7.329s、退出1。原因是本审查把同时含 note 和预测摘要的字段误当成纯 note。不是产品缺陷，不计为有效反例 |
| 更正后探针 | 一次，120s；检查提前错误、收尾关闭错误、健康对照，三项均 PASS，包6.764s、退出0。两个故障的真实内层退出1、原 gate pass=false，仍有按顺序保存的前提/目标；健康内层退出0、原 gate pass=true。Invalid/Observed/Cleanup 等及 JSON 往返核对通过 |

探针使用本轮自有文件，晚到关闭错误由真实关闭成功后返回接缝错误产生；不是原生 Close 故障。没有伪造内层 verdict 或修改 gate 判定。每轮均含真实 normal-drain 根层场景，避免 TestMain 零场景导致无效汇总；更正只发生在隔离副本的断言，没有改原仓库或被审实现。

日志为 `pristine.log`、`trace-first-invalid-probe.log`、`trace-corrected.log`。三轮 normal-drain 均确认自有子进程结束，结束时只读检查其 PID 7908、18396、46484 均不存在；探针内层均已等待退出、Final ended、insurance nothing left。没有对未知对象发信号，未留下本轮已知进程或未关闭的实验文件。

作者临时新旧副本、Windows/WSL 全包及交叉 vet 是作者证据，不是本轮重跑；其只选内层测试导致零根场景、包退出1的记录按文档保留。作者临时元数据原件不可访问的部分已在 §6 限定。本轮最初的日志提取曾误把 WP25P 的“ID+JSON”格式当作纯 JSON，改正提取后重新核对集合；不是 CI/产品解析失败。没有把分析工具错误、编译失败或无效夹具计为产品检出。

未安装工具、申请资源、执行用户项目配置或新的原生机制实验。已关闭 B1～B3 只检查必要直接回归，没有重做全部 T1～T6。

## 9．G5 与冻结前事项的分层

本轮可以采纳的新增支持是：**同一固定代码点，两种原生架构的计划场景、既有观察器与门禁适用；K1 的布局前提、目标执行、唯一规则和收尾现在可逐项追溯；最小错误保存回归有效。** 可以据此将原型交付及其证据送入下一步冻结审查。

它没有证明生产 ABI 稳定、所有 macOS 版本、任意宿主/程序或全部原生故障；也没有把 SDK 布局探针升级成运行内核宏分支证明。

| 类别 | 仍须处理的内容 |
|---|---|
| **F：G5 冻结前必需** | 在冻结审查中明确采纳本次两架构证据并同步“已测组合”；选择生产 E1/E2，说明选择所需兼容性依据、支持承诺与已测组合差额。不能因 P-3 只验证 E1 就自动选 E1，也不能自动删除15.x Intel支持；是否需额外最小证据应由具体入口/兼容性假设决定 |
| **F：其他规格/候选审查** | §20 的 G1资源移交/永久禁用、G2时间、G3/P-1责任、G4闸门、G6能力探测、G7恢复、G8/G11解释器/PATH、G9取消、G10输出、G12/D14事实约束等仍按各自证据逐项审查；§20.2 的 D/E/C 技术候选仍须采纳或否决。当前只审最小增量，不把原型 GO 自动当作这些项已批准或已实现 |
| **L：如实保留的证据限制** | macOS26.6.2 arm64虚拟机与26.6.1 Intel各一次作业，每族三次；两者实际 xnu-12377.161.x 没有公开源码；15.x Intel只有所列静态依据；只测信号9，没有原生 core/ENOMEM/EINTR；旧布局失败仍未知。G1挂起资源无统一上界、G3/G6/G7受控构造的既有局部限制保持，不因本轮移类或接受新风险 |
| **P：授权后的生产工作** | Runner/CLI、迁移0009及D14约束、真实生产入口接线、事务/取消/配置固定/空批次和输出叠加、C-9生产静态覆盖、所选入口的生产平台测试。不是要求本原型 PR 提前实现 |
| **N：非阻塞增强** | N1显式结果集合/重复拒绝、N2 R_before、结构化站点关联与受控诊断精化；不因当前未实现而重新否决本次最小观测 |

**没有新的产品待决事项由本轮引入。** 生产入口与支持差额仍需明确评价；若未来选型要求改变已批准支持/信息/风险保证，交负责人另行决定，本报告不代为批准。维护事项 M-1、M-14～M-17、N-087-1 保持原范围。

## 10．结束核验与交付边界

结束核验：**2026-10-29 20:17:20 JST（11:17:20 UTC）**。再次读取 PR、master 和两组 CI：HEAD/base、本地 HEAD/树仍为固定身份；PR open、未合并、auto_merge=null；新诊断及最终常规 CI 都仍 attempt1 success。原型代码点之后仍仅文档变化，没有将其他对象纳入结论。报告落盘后核对仅有本报告未跟踪，跟踪文件与暂存区差异为空。

原仓库仅新增本未提交报告，其他跟踪文件、暂存区与其他会话内容保持不变。未提交、推送、合并、改基、触发/重跑/取消 CI，未扩展原型或申请资源。

**建议合并是独立审查建议，不是合并操作、规范冻结、产品批准、生产入口选择或实施授权；本报告不宣布 WP2.5/M2 验收。**
