# WP2.5 冻结之前的机制原型（test-only）

> 状态：**原型，PR #33 已合并为 `5e9a42c`**，合并核验与 master push CI 正式成功。之后修复 N4、N6 的独立小 PR #34 经 Review 081、082 为 **NO-GO**（B1：停止读取可能在阶段 J 之前阻塞，及其停止结果的生命周期；N4-R），N6 关闭；Review 083 关闭 B1-Ra、B1-Rb（修复技术 GO，当前不建议合并），N4-R 的残余（N4-R-S）与 macOS 普查失败的措辞和信息（N-MAC）已定向修复；Review 084 为 **NO-GO**（B-N4：有效的回答与真实的失败记录并存时，冲突检查被跳过，内层 confirmed / 0），已定向修复；Review 085 关闭 B-N4 与 N1，仍为 **NO-GO**（B-N4-R：持有者被结束之后报告的稳定被当作工具完成），已定向修复；Review 086 仍为 **NO-GO**（B-N4-R/T：期限回调已进入时仍能发布完成；B-N4-R/I：被忽略的报告失败之后仍能发布完成），已定向修复；Review 087 **技术 GO**（B-N4-R/T、B-N4-R/I 关闭，N4 系列只在受控 helper 协议范围内收口），PR #34 已以 merge commit 合并为 `cc4514a`，合并后核验完成（[review-resolution.md §89 ~ §104](review-resolution.md)）。依据 [§88](review-resolution.md) 的授权：PR #32 合并核验与 master push CI 全部成功之后，从已核验的 master 另开 PR，把 [WP2.5 规范草案](wp2.5-check-runner.md) §17.2 中 T1 ~ T6 冻结之前的关键可行性问题转成可审查的证据。
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
> - [Review 083](reviews/codex-review-083.md) 对 `b51cc5e`：**B1-Ra、B1-Rb 可以关闭**（Review 081 B1 在当前原型范围内整体关闭），没有新的 BLOCKER；修复技术 GO，但最终代码点的原型诊断为 failure，**当前不建议合并**。N4-R 不能整体关闭（N4-R-S：真实的 `ok` 之后发生的写入与报告失败仍得到 refuted，NON-BLOCKER）。N-MAC：§2.14 关于 macOS 普查失败的归属与因果措辞过强，相关 PID 的归属与完整清理的结果未知。已定向修复（§2.15），**待独立复审**。
> - [Review 084](reviews/codex-review-084.md) 对 `2414ae9`：**技术 NO-GO**。BLOCKER B-N4：有效的 `ok` 与真实的 `epipe` 记录并存时，冲突检查被跳过，内层 confirmed / exit 0；N4-R-S 不能整体关闭。N-MAC 的诊断补充成立，旧未知保留，但旧 macOS 失败本身不构成永久的合并门槛。NON-BLOCKER N1：短命令名前缀匹配被标成“同一程序”。B1、N6 没有回退。已定向修复（§2.16），**待独立复审**。
> - [Review 085](reviews/codex-review-085.md) 对 `4d1e575`：**原 B-N4 的冲突修复成立，N1 关闭**；**技术 NO-GO**，BLOCKER B-N4-R：缺少工具完成的证明，结束持有者可能截断证据。完整的 `epipe` 记录之后报告操作失败，或已到达的工具失败的报告被结束打断，内层仍为 confirmed / exit 0。N4 系列没有整体关闭；B1、N6 没有回退。已定向修复（§2.17），**待独立复审**。
> - [Review 086](reviews/codex-review-086.md) 对 `bc5ad78`：Review 085 的三类原反例已修；**技术 NO-GO**，B-N4-R 不能整体关闭：B-N4-R/T，期限回调已进入、报告仍在进行时主路径仍能发布完成；B-N4-R/I，故障分支忽略报告返回值之后仍能发布完成。NON-BLOCKER N-RENAME：改名的原子性措辞过宽。B1、N6 与已关闭的 B-N4、N1 没有回退。已定向修复（§2.18），**待独立复审**。
> - [Review 087](reviews/codex-review-087.md) 对 `245b51b`：**技术 GO**。B-N4-R/T、B-N4-R/I 关闭；N4 系列只在已审查的受控协议范围内收口；没有新的 BLOCKER；NON-BLOCKER N-087-1（`TestToolState` 第 5 组没有确证回调已进入）登记为后续事项，N-087-2（“没有无界等待”过宽）已更正措辞；N-RENAME 关闭。最终常规 CI run 40116851180：attempt 1 为 failure、7/8（Windows 的 M-15 新观测），attempt 2 只重跑 Windows，正式成功。**技术审查通过，待合并核验**（§2.19）。
> - **合并之后**（§104、§105）：PR #34 合并为 `cc4514a`；master push CI run 40122051541 的 attempt 1 为 7/8（Windows 的 `TestSourceRemoveInTerminal`，登记为 M-17），按负责人的单独授权只重跑失败作业的 attempt 2 正式成功。M-17 的登记与诊断（PR #35）已合并为 `bd45583`；M-17 未解决。
> - **2026-10-25**（§106）：负责人决定了草案 §17.3 的 14 项产品选择与 C-4。规范第 5 版草案对 C-1 ~ C-20 逐项给出采纳建议（草案 §17.4），都待审查。Review 089 之后，草案的 S6 按本文 §2.13、§2.14 的停止结果生命周期改写（先建立阶段 J 的截止，再启动停止任务），C-4 落实为 O3 之后宿主的持续禁用（§107）；Review 090 技术 GO（§108），PR #36 已合并为 `f3fbe83`；第 6 版草案增加 D21 与冻结之前的规格草案（草案 §19，§109），其中 G3、G5、G6、G7 的最小原型只有计划，没有实施。本文 §9 ~ §12 是决定之前的记录，保持原样：§10 标题中的“全部未批准”是当时的状态。
> - **2026-10-27**（§114）：PR #37 合并为 `2c5cf23` 之后，负责人决定 Q-1、Q-2 (a)，并授权 P-1、P-2、P-4 的隔离原型与 P-3 的整理。计划在写代码之前固定（§14.1 ~ §14.6，`dcdb2b2`），固定代码点 `d45a319` 的诊断 run 40439516254 三平台 success（P-1 18/18、P-2 9/9、P-4 18/18 confirmed，§14.7）；结果与门槛的状态（§14.10）**待独立审查**。
> - [Review 095](reviews/codex-review-095.md) 对 `1be48c3` 为**技术 NO-GO**（B1：新增观察入口没有完整失败关闭；N1、N2 非阻塞）。已定向修复（§14.11 ~ §14.14）：回归先行 `0a9a01a`（诊断 run 40641559468 为 failure，预期），修复与固定代码点 `85ec2ca`（诊断 run 40643975558 三平台 success）；`d45a319` 的证据降为历史观测。**已修复，待独立复审**；不宣布 B1 关闭。
> - [Review 096](reviews/codex-review-096.md) 对 `b80982c` 为**技术 NO-GO**：Review 095 点名的路径修复成立，N1、N2 关闭；B1 不能整体关闭（B1-R：Linux 排他区间退出时清除 subreaper 的错误被丢弃）。已定向修复（§14.15 ~ §14.17）：回归先行 `b641e17`（诊断 run 40656037766 为 failure，预期），修复与固定代码点 `48ff0cd`；`85ec2ca` 的 Linux P-1 与 T2 证据降为历史观测。**已修复，待独立复审**；不宣布 B1 关闭。
> - [Review 097](reviews/codex-review-097.md) 对 `800b5e8`（固定的原型代码点 `48ff0cd`）为**技术 GO**：B1-R 关闭；结合 Review 096 对原点名路径的关闭，Review 095 B1 **在受审原型范围内整体收口**；N-096-1 的文档意见关闭（PID 2668 的历史未知保留）；N-096-2 仍未修，非阻塞，待另开独立 PR 修复；没有新的 BLOCKER（§14.18）。技术 GO 不是规范冻结、产品批准、生产授权或验收。
> - **2026-10-28**（review-resolution §118）：PR #38 以 merge commit 合并为 `27ce69d`，合并核验与 master push CI 正式成功。N-096-2 在独立小 PR（分支 `test/wp2.5-prototype-n-096-2`）中修复（§14.19、§14.20）：回归先行 `e23a7ef`（诊断 run 40689865305 为 failure，预期），修复与固定代码点 `18bc6a4`（诊断 run 40691692731 三平台 success）。**已修复，待独立审查**；不宣布关闭。
> - [Review 098](reviews/codex-review-098.md) 对 `8b6aca1`（固定的原型代码点 `18bc6a4`）为**技术 GO**：N-096-2 的两处问题在受审的受控原型范围内关闭；N-098-1 的两处措辞已在 §14.19 收窄；没有新的 BLOCKER（§14.21）。技术 GO 不是规范冻结、产品批准、生产授权或验收。
> - **2026-10-29**（review-resolution §120）：PR #39 以 merge commit 合并为 `614ac4b`，合并核验与 master push CI 正式成功。负责人授权冻结之前的文档整理：规范第 8 版草案登记了 G5 的 ABI 依据、状态的解码规则与 P-3 的完整计划（草案 §19.5、§19.12、§20），**待独立审查**。P-3 只有条件授权（须经独立审查明确允许实施，另开原型 PR）；本文的原型代码没有改动。
> - [Review 099](reviews/codex-review-099.md) 对 `657ffc9`：**技术 NO-GO**（B1：错误套用用户态的 `WEXITSTATUS`，waitid 的退出值可带 exit 参数的第 8 ~ 23 位；B2：P-3 对同一原生反证同时规定 refuted 与 invalid）。规范第 8 版第二稿定向修订（草案 §19.5、§19.12、§19.13、§20；review-resolution §121），**两项阻塞已修订，待独立复审**；新增负责人待决的 Q-3；P-3 的启动条件仍未满足。本文的原型代码没有改动。
> - **2026-10-29**（review-resolution §122）：负责人决定 Q-3 选 (a)：macOS 的正常退出在核对成立之后取原始 `si_status` 的低 8 位，waitid 的第 8 ~ 23 位不进入事实。P-3 的预测已按它收敛（草案 §19.12）；启动条件仍须独立复审明确允许。本文的原型代码没有改动。
> - [Review 100](reviews/codex-review-100.md) 对 `344cbca`：**技术 GO**，Review 099 的 B1、B2 在规范与计划层面关闭；P-3 可按既有条件授权启动（review-resolution §123），尚未执行。本文中 macOS 的 T3 证据仍**不**覆盖第 8 版的 W / K / C 核对。
> - **2026-10-29**（PR #40 合并为 `dbcd5d4` 之后）：P-3 在分支 `test/wp2.5-prototype-p3` 的独立原型 PR 中实施。实施的计划在写任何原型代码之前登记（§15），固定 base `dbcd5d4`、受审的文档提交 `344cbca` 与计划的版本。固定代码点 `4e690d8` 的诊断 run 40751152882 四个作业 success：arm64 与 x86_64 的 P-3 各 63/63 confirmed、门禁 13/13，既有的 macOS 场景在新的核对下 33/33（§15.7）。**结果待独立审查**，不宣布 G5 满足（§15.10）。
> - [Review 101](reviews/codex-review-101.md) 对 PR #41（HEAD `c776f6a`，代码点 `4e690d8`）：**技术 NO-GO**（B1：计划故障覆盖了同一次真实的枚举错误；B2：工具文件与观察器的关闭错误没有进入终判；B3：枚举依赖内部的 `ENOMEM` 重试不受期限约束）。定向修复在 PR #41 中（§15.11）：
>   - 回归先行 `7f9fda0`：诊断 run 40816447439 为 failure，6 个新门禁与 8 个直接检查子项按预测失败；
>   - 固定代码点 `eb2317f`：诊断 run 40817305419 为 **failure**。Intel、Ubuntu、Windows 成功，B1 ~ B3 的新门禁与直接检查在两个架构上都通过；arm64 的既有门禁 `p3.k-nostatus-once` 因同一内层运行的 `layout.1` 为 invalid 而失败，原因没有记录，未知。
>
>   **已修复，待独立复审**。
> - [Review 102](reviews/codex-review-102.md) 对 PR #41（HEAD `fbcef6f`，代码点 `eb2317f`）：B1 ~ B3 在受审的原型调用链范围内**关闭**，修复技术 GO；当前**暂不建议合并**（诊断失败；arm64 的 `p3.k-nostatus-once` 没有执行目标、没有命中接缝；同一内层运行中布局场景的失效详情丢失，N-102-1 非阻塞）。按负责人授权的最小观测与补证（§15.12）：门禁保留内层运行的全部最终结果，3 个组合回归；固定代码点 `9d45036` 的诊断 run 40847347022 四个作业 success。run 40817305419 中布局为何 invalid 仍**未知**。**待独立审查**。
> - [Review 103](reviews/codex-review-103.md) 对 PR #41（HEAD `ab4b7dd`，代码点 `9d45036`）：**技术 GO，建议合并固定对象**；N-102-1 关闭；B1 ~ B3 按 Review 102 的范围保持关闭；N1、N2 仍为非阻塞；run 40817305419 的 failure 与布局失效的未知原因保留；证据足以作为冻结审查的输入，G5 **不认定整体满足**（§15.13）。负责人条件授权以 merge commit 合并 PR #41（review-resolution §127）。
> - **2026-10-29**（review-resolution §128）：PR #41 以 merge commit 合并为 `03a6f6e`（锁定最终 HEAD `ee83482`），合并树与最终 HEAD 相同，master push CI run 40885279060 正式成功。G5 的冻结审查包在规范第 9 版 §21（独立文档 PR），**待冻结审查**；本文的原型代码没有改动。
> - **2026-10-30**（review-resolution §131、§132）：负责人决定 Q-4 选 (a)：入口 E2 为拟采用的入口，先完成 P-5 及其独立审查再决定冻结。PR #42 已以 merge commit 合并为 `0442f14`，合并核验与 master push CI run 40916689386 正式成功。P-5 的计划在 §16 定稿，**待 Review 107**；P-5 的执行未授权，本文的原型代码没有改动。
> - **2026-10-30**（review-resolution §133）：Review 107 为技术 NO-GO：B1 是门禁内层的期限与准入冲突；N1、N2 非阻塞。§16 的第二稿在同一 PR 中定向修订：
>   - §16.8 分开三种角色的准入、观察、收尾与截止，并给出两条时间线；
>   - §16.7 写定返回值与 errno 互斥的分类，以及调用次数的规则；
>   - §16.11 保留 `kind`，另设 `evidence` 与 `calls`；
>   - §16.5 的 `P5.c` 明确增加第二次读取。
>
>   第二稿**待定向复审**，不合并；P-5 的执行未授权，本文的原型代码没有改动。
> - **2026-10-30**（review-resolution §134）：Review 108 为技术 NO-GO：准入冲突的修复成立，B1 整体仍有残余 B1-R；Review 107 的 N1、N2 在计划规格层面收口。§16 的第三稿在同一 PR 中最小修订：撤回由 `WaitDelay` 推出的返回上界；分开终止请求、可靠观察退出、`Cmd.Wait` 完成、管道关闭与复制任务结束；写定截止、发信号的规则、提前出口与所有权表；重写时间线二。第三稿**待定向复审**，不合并；P-5 的执行未授权，本文的原型代码没有改动。
> - **2026-10-30**（review-resolution §135）：Review 109 为技术 NO-GO：B1-R 的主要修订成立，残余 B1-R-S 是发信号与回收共用锁的持有边界及其与截止查询的关系；N-108-1、N-108-2 关闭。§16 的第四稿在同一 PR 中只修订 B1-R-S：写定控制锁与状态锁的范围、Wait 任务与请求任务的加锁、发布、解锁与外部调用的顺序、查询的规则，并同步所有权表、提前出口与时间线。第四稿**待定向复审**，不合并；P-5 的执行未授权，本文的原型代码没有改动。
> - **2026-10-30**（review-resolution §136）：[Review 110](reviews/codex-review-110.md) 对第四稿（`35ecd79`）为**文档技术 GO**：B1-R-S 关闭，B1-R 在计划规格层面整体收口，没有新的 BLOCKER；计划具备供负责人单独授权实现和执行的条件。负责人条件授权以 merge commit 合并 PR #43，并条件授权在其合并核验全部成功之后另开 test-only 原型 PR 按 §16 第四稿实现并执行 P-5。计划通过不代表协议已实现或动态验证；§16 的技术内容没有改动。
> - **2026-10-30**（review-resolution §137、§138）：PR #43 合并为 `021c956`，合并核验完成。P-5 在分支 `test/wp2.5-prototype-p5`（PR #44）中实施，计划先于代码登记（§16.14、§16.15）。固定代码点 `0213fb7` 的诊断 run 41055799521 为 **failure**：两个 macOS 架构的原生 39/39 confirmed、接缝门禁 2/2、模型 10/10、布局 3/3，唯一的 T 是 `P5.static.symbols`（§16.16），终判 invalid（T），**交负责人决定**，不重跑；PR #44 不合并。
> - **2026-10-30**（review-resolution §139）：[Review 111](reviews/codex-review-111.md) 对 PR #44 的实现为**技术 NO-GO**，有两项阻塞：
>   - B1：正常 EOF 的路径没有关闭门禁的父端读端；
>   - B2：新清理链会遗失必要的观察失败。
>
>   首次诊断仍为 failure / invalid（T）；N1 ~ N3 为非阻塞项。负责人只授权纯文档的修订准备：§16.17 写定 B1、B2 的最小修复与回归，以及符号检查的第二版（§16.4 的方法修订），**待独立审查**。实现、workflow 与新的诊断都未授权，本文的原型代码没有改动。
> - **2026-10-31**（review-resolution §140）：[Review 112](reviews/codex-review-112.md) 对 §16.17 第一稿（`691435a`）为**计划技术 NO-GO**，有三项计划阻塞：
>   - B-P112-1：带错完成的关闭任务可能不移交；
>   - B-P112-2：绑定只核对起始地址，没有核对完整的指针范围与逐次展开；
>   - B-P112-3：真实子进程回归没有写定自己的截止与失败出口的收尾。
>
>   负责人只授权修订 §16.17 的计划：第二稿修这三项与列明的精化，**待定向复审**。原实现的 B1、B2 不关闭；首次诊断保持 failure / invalid（T）。代码修复与新的诊断都未授权，本文的原型代码没有改动。
> - **2026-10-31**（review-resolution §141）：[Review 113](reviews/codex-review-113.md) 对 §16.17 第二稿（`598bb2f`）为**计划技术 NO-GO**：B-P112-1、B-P112-2 在计划层面关闭；B-P112-3 的主要修订成立，残余 B-P112-3-R。B-P112-3-R 有三处：
>   - 持有者的终止请求没有任务化；
>   - 每侧 Cleanup 的实际时机没有写定；
>   - 截止之后没有持有者。
>
>   负责人只授权定向修订 §16.17：第三稿收口 B-P112-3-R，并完成列明的小范围精化，**待定向复审**。原实现的 B1、B2 不关闭；首次诊断保持 failure / invalid（T）。代码修复与新的诊断都未授权，本文的原型代码没有改动。
> - **2026-10-31**（review-resolution §142）：[Review 114](reviews/codex-review-114.md) 对 §16.17 第三稿（`7930a93`）为**计划技术 NO-GO**：
>   - 请求任务化、每侧 Cleanup、持续持有三项主体修订成立；
>   - B-P112-3-R 尚余 B-P112-3-R/D：等待或同步读取之后缺少统一的截止复核；
>   - B-P112-1、B-P112-2 保持计划层面关闭。
>
>   负责人只授权第四稿的纯文档定向修订：写定新任务的准入与统一的到期出口，并完成 N-114-1 ~ 3 的精化，**待定向复审**。原实现的 B1、B2 不关闭；首次诊断保持 failure / invalid（T）。代码修复与新的诊断都未授权，本文的原型代码没有改动。
> - **2026-10-31**（review-resolution §143）：[Review 115](reviews/codex-review-115.md) 对 §16.17 第四稿（`cd0d4f7`）为**计划技术 GO**：B-P112-3-R/D 关闭，B-P112-3-R、B-P112-3 在计划层面整体收口，N-114-1 ~ 3 关闭，没有新的 BLOCKER。负责人授权按第四稿实施固定范围的修复，选择流程 (i)（实现完成之后一次推送、一次诊断），不另取回归先行的诊断，不授权合并。实施在写代码之前登记（§16.18）。原实现的 B1、B2 仍待实际修复与独立复审；首次诊断保持 failure / invalid（T）。
> - 这**不是**规范冻结：草案为第 7 版，待独立审查和冻结；14 项产品选择、D21、Q-1 与 Q-2 已决定，技术候选仍待审查。T1 ~ T6 按 Review 075 的裁定只得到**局部支持**（§11；草案 §17.5 的 G1 ~ G15 与 §19）。
> - 原型成功**不是**生产验收：生产 Check Runner、迁移 0009、CLI 接线都没有授权，WP2.5 没有实现或验收，M2 没有整体验收。
> - 本文中的“候选规格”都待审查；其中涉及可见行为的（例如 C-4、C-7、C-10）依赖对应的产品选择（§9、§10）。C-4 的产品语义与 C-7（D20 (i)）已于 2026-10-25 决定，其技术部分仍待审查。本文不新增接受风险，也不批准任何保证调整。

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
- **Review 083**（§2.15）：
  - B1-Ra、B1-Rb 经审查可以关闭。
  - N4-R-S：询问即观察的终点。持有者不再写，先关闭自己的输出，再给出最终回答 `ok`；场景以每个保留读端在 2 s 内的 EOF 证实观察已结束，然后最后读一次报告。没有终点、没有回答、回答旁有失败记录，都是 invalid。
  - N-MAC：更正 §2.14 的措辞，2088 与 1989 的归属、EINVAL 的原因与清理结果都未知。macOS 普查读不到环境时，失败信息补充该进程的启动时间、状态、是否正在退出、父进程、进程组、是否同一程序，以及失败之后的状态；归属规则不变。
  - 回归先行 `61a43db`（run 40084348430 为 failure，预期）；修复代码固定为 `56234c3`，run 40084692941 为 success。旧运行中 macOS 的未知结果不因此消除；就此 PR #34 当前不具备合并条件（§2.15）。
- **Review 084**（§2.16）：
  - B-N4：证据先完整收集（等待记录、询问、读到 EOF），持有者结束之后再最后读一次报告，然后整体分类。`ok` 与 `epipe` 互斥，并存即 invalid（不论先后）；记录须是持有者的 PID；收集中的任何失败使任一记录旁的结论为 invalid；成功不提前返回。
  - N1：普查的标签改为短命令名前缀匹配（启发式）。
  - 回归先行 `7307d5d`（run 40095937741 为 failure，预期）；修复代码固定为 `1e82fbd`，run 40096356396 为 success。
  - 旧 macOS 失败的未知项保留，不称为已解决；按 Review 084 与负责人本轮的指示，不可追溯本身不再作为永久的合并门槛（更正上一条的最后一句）。
- **Review 085**（§2.17）：
  - 原 B-N4 与 N1 经审查可以关闭。
  - B-N4-R：持有者在全部报告操作成功、没有工具失败之后，以原子改名提交完成（`<report>.done`）；场景在两条路径上都等待它，confirmed 与 refuted 都需要它；结束持有者只是清理。完成协议与错误传播见 §2.17 的两张表。
  - 更正 §2.16 的观察终点：只有完成提交之后的行为不属于观察。
  - 回归先行 `acfee1f`（run 40102730930 为 failure，预期）；修复代码固定为 `fb3f56d`，run 40103135762 为 success。
- **Review 086**（§2.18）：
  - 持有者的全部入口（主路径、期限回调、模拟期限点、报告操作、完成提交）经过同一个工具状态与同一把锁：报告失败在报告层保存，不论调用者是否检查，之后的成功不清除；期限回调第一步在锁下裁定；完成在整个发布期间持有锁，只在没有失败、期限没有进入时发布；发布之后不再写入属于观察的报告。
  - 改名的原子性收窄为场景临时目录内的同目录改名加严格的内容校验（N-RENAME）。
  - 回归先行 `750e139`（run 40109659805 为 failure，预期）；修复 `9e3b7ca` 的诊断 run 40110180232 中 `report.limit-after-done` 在两个平台上空通过，夹具更正为 `f728e11`，run 40110917799 为 success。
- **Review 087**（§2.19）：技术 GO，B-N4-R/T、B-N4-R/I 关闭，N4 系列只在受控 helper 协议范围内收口；N-087-2 的措辞已更正，N-087-1 登记为后续事项；最终常规 CI 的 attempt 1 为 7/8（M-15 新观测），attempt 2 只重跑 Windows 并成功。技术审查通过，待合并核验。
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
- **macOS 另有一项非设计内的失败**（如实保留，没有重跑）：`report.epipe-unwritable-alive` 的内层结论正确（invalid），但外层普查读进程 2088 的环境时得到 EINVAL，判为普查工具错误，这项门禁回归因此不通过。2088 不在这次内层运行的创建记录中（`created` 为 2，持有者是 2072）；按普查的条件，它是同一用户、在内层运行期间启动的其他进程。这是 §13 已登记的普查前提（同一用户、启动不早于内层、读不到环境的进程使普查失败，保守），与本轮的改动无关，本轮不改普查。**更正**（Review 083 N-MAC，§2.15）：“不在创建记录中”“同一用户的其他进程”“与本轮的改动无关”的措辞过强。2088 是否属于本轮、EINVAL 的原因都未知，与本轮改动的因果没有证明；这是被测普查的失败（`final=unknown`），与最终诊断中保险清理的那一次阶段不同。
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
- **更正**（Review 083 N4-R-S，§2.15）：上表的 `ok` 只证明回答以前的写入；回答之后、场景判定之前发生的写入与报告失败不会进入结论，仍可得到 refuted。“报告失败、工具失败为 invalid”只对回答以前成立。已由 §2.15 的判定取代。
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
    - **更正**（Review 083 N-MAC，§2.15）：上面两点中“不在创建记录中”“其他进程”的措辞过强。1989 是否属于本轮、EINVAL 的原因、保险清理之后普查范围内是否清理完整，都未知；`final=ended` 是保险清理之前被测路径的结论；源码没有改动不构成因果排除；与回归先行那次同为 EINVAL，不证明同一根因。
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

**状态**：B1（B1-Ra、B1-Rb）与 N4-R **已修复，待独立复审**；不宣布关闭或 GO。N6 保持经 Review 081 的关闭范围。（其后：Review 083 关闭 B1-Ra、B1-Rb，认定 N4-R-S 与 N-MAC；本节关于 macOS 普查失败的措辞由 §2.15 更正。）

### 2.15 Review 083：回答之后的工具失败（N4-R-S）与 macOS 普查的诊断（N-MAC）

**审查结论**（[Review 083](reviews/codex-review-083.md) 第一、三 ~ 五节，对 `b51cc5e`）：

- **B1-Ra、B1-Rb 可以关闭**：停止的结果随任务移交，挂起时与完成后的错误都保留；Windows 的 API 结果按契约区分。Review 081 B1 在当前原型的范围内可整体关闭；挂起资源的上界与后续宿主策略仍是冻结之前的门槛。
- **没有新的 BLOCKER**。
- **N4-R 不能整体关闭（NON-BLOCKER，N4-R-S）**：真实的 `ok` 只证明回答以前的写入。审查方让持有者先给出真实的 `ok`，随后它的 stderr 断开、下一次写入失败、报告也失败，都发生在场景最终判定之前，结论仍为 refuted。
- **N6** 没有直接回退。
- **macOS 最终诊断的失败**是真实的外层工具错误，不是预期的截断拒绝本身失败；相关 PID 的归属与完整清理的结果**未知**。
  - **N-MAC（NON-BLOCKER）**：§2.14、§98 中“不在内层的创建记录中”“同一用户的其他进程”“与本轮的改动无关”的措辞过强，应收敛为“既有、未改动的调用链出现读取失败，归属未确认”。
- **修复技术 GO，但当前不建议合并**：最终代码点的原型诊断为 completed/failure，最终常规 CI 的 8/8 不能替代它。
- 没有新增产品决定、保证调整或接受风险。

**更正 §2.14 与 §98 的措辞**（N-MAC；§98 原文不改，更正记入 §99）：

- 2088（回归先行，被测普查）与 1989（最终诊断，保险清理）的环境读取得到 EINVAL。这两个进程是否属于本轮**未知**：同一用户、启动时间在范围内，只是普查的筛选条件，不是归属证明；“不在创建记录中”是从 `created` 计数与 `unowned` 列表推出的，日志没有打印完整的创建记录，不能据此独立确认，即使确认也不能排除未登记的后代。
- EINVAL 的原因**未知**；不能排他地归为“正在退出”或某种系统机制。
- 1989 那一次，`final=ended` 是保险清理之前被测路径的结论，不能升级为保险清理之后的全局确认；普查范围内的清理是否完整**未知**。
- 与本轮改动的因果**未证明**：普查与保险清理的源码没有改动是事实，但不构成因果排除。
- 两次的阶段不同：2088 是被测普查（`final=unknown`），1989 是判定之后的保险清理（此前 `final=ended`）；同为 EINVAL 不证明同一根因。
- 两次失败都是保守的方向，没有据此向任何 PID 发信号，没有形成绿色的假通过；都没有重跑。

**macOS 旧失败：还能恢复与已无法追溯的事实**：

| 事项 | 能否恢复 | 依据 |
|---|---|---|
| 失败的阶段（被测普查 / 保险清理）与错误文本 | 能 | 作业日志中的 `WP25GATE` 行（problems、insurance） |
| 内层结论、`created` / `receipts` 计数、`unowned` 列表、`remedy`、`final` | 能 | 同上 |
| 完整的创建记录（每个 PID 与身份） | 不能 | 见证文件在 runner 的临时目录，没有打印 |
| 内层运行自己的输出 | 不能 | 门禁回归不打印内层的输出 |
| 2088 / 1989 当时的身份、状态、父进程、进程组、是否正在退出、是否带本轮标记 | 不能 | 当时没有记录；进程与 runner 都已不在，现在重查不能恢复过去的状态 |
| 它们是否属于本轮 | 未知 | 见上 |
| EINVAL 的原因 | 未知 | — |
| 保险清理失败之后，普查范围内是否还有本轮的进程 | 未知 | 没有完成的新普查 |

**回归先行**（`61a43db7f86ff6ceb4e8b1259cba07d7e63746cd`，判定没有改）：

| 回归 | 内容 | 旧代码上 |
|---|---|---|
| 门禁回归 `report.fail-after-ok` | 审查方的精确反例：两路读端都保留；持有者给出真实的 `ok`；随后它的 stderr 断开，下一次写入真实失败，`epipe` 报告也真实失败（只读打开），它仍存活；场景在取得 `ok` 之后，限时等待测试自己的标记，确认失败已发生，再按原代码判定 | **refuted**，失败 |
| 门禁回归 `report.ok-not-final` | 持有者回答之后不再写，但输出仍然开着：回答不是观察的终点 | **refuted**，失败 |
| 直接检查 `TestCensusUnreadable`（darwin） | 普查只覆盖测试自己的辅助进程，它的环境读取经接缝失败为 EINVAL：普查必须失败，并说出该进程的身份、状态、是否正在退出、父进程、进程组、是否运行本测试的程序，不得出现环境内容；状态也读不到时，要写明 | 失败（旧信息只有 PID 与错误） |

- **三平台诊断** [run 40084348430](https://github.com/XKHoshizora/specmetry/actions/runs/40084348430)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `61a43db`。三平台都只有上表中预期的失败（darwin 的直接检查只在 macOS 运行）：门禁回归 Ubuntu 52/54、macOS 43/45、Windows 48/50；场景 38、27、29 confirmed；组合 13/13。这次 macOS 没有出现计划外的 EINVAL。
- **作者本机 Windows**：两项门禁回归得到 refuted，与上表相同。

**修复 N4-R-S**（`419e4f388ecc7cbdee2c853a337f4c13303a055e`）：询问即观察的终点。

- **观察区间**：从停止到持有者的最终回答。
- **持有者**：看到询问之后不再写；**先关闭自己的输出**，使它之后做的任何事都不属于这次观察；然后给出唯一的最终回答 `ok`（它的写入都没有失败）；之后只等待被结束。输出关不上，或回答写不出，就没有回答。
- **场景的完成边界**：收到回答之后，要求每个保留读端在 2 s 内读到 EOF（不等待地读：Unix 上是非阻塞读，Windows 上是 PeekNamedPipe 加 ReadFile），即每一路的所有写端都已关闭，此后不可能再有属于这次观察的写入。迟到字节改为每一路从停止到 EOF 实际携带的字节。
- **最后一次读报告**：在 EOF 之后；回答旁边若有失败记录，就不是回答。
- **判定**：

  | 情形 | 依据 | 结论 |
  |---|---|---|
  | 预期的现象发生 | `epipe` 记录到达（等待回答期间才到达的也算） | 这一项预测成立 |
  | 预期的现象没有发生，工具正常 | 没有 `epipe`；最终回答 `ok`；每个保留读端都读到 EOF；每一路都保留并有迟到字节 | refuted |
  | 回答之前的写入或报告失败 | 没有回答（或有失败记录） | invalid（或预测成立） |
  | 回答之后、观察完成之前仍有写端（回答不是终点） | 2 s 内没有 EOF | invalid |
  | 完成交接附近的故障：输出关不上、回答写不出 | 没有回答 | invalid |
  | 回答与失败记录并存 | — | invalid |
  | 计数、读取或状态读不到 | — | invalid |

- 中途的 `ok` 不再是最终成功的凭据：只有在输出关闭之后给出、并由 EOF 证实观察已结束的回答才算。
- 只调整原型协议，不决定生产的产品规则。
- **随修复增加** `report.close-fails`：持有者关不上输出（第二次关闭的真实错误），没有回答，要求 invalid；旧代码没有这一步，不声称旧代码上的对照。

**修复 N-MAC 的诊断**（`56234c364eaffc109f1ddf1cd0bd837420afcbf7`）：

- 普查找到的进程，若 `kern.procargs2` 读取失败、复核又不能证明它已结束，普查仍然失败（不跳过、不重试、不改变归属规则）；错误改为包装原错误，并附上：
  - 从普查读到的列表项：启动时间、状态、是否正在退出（`P_WEXIT`）、父进程、进程组、短命令名是否与本测试的程序相同（只记“是 / 否 / 未知”）；
  - 失败之后再读一次状态的结果，读不到时写明“unreadable”。
- 不记录参数、环境或进程名。新增的信息来自普查已经读到的列表项与既有的复核；复核读不到时普查照样失败。
- 这只是给以后的同类失败留下可追溯的事实，不是归属的证明，也不改变保险清理。

**修复后的证据**（`56234c3`）：

- **三平台诊断** [run 40084692941](https://github.com/XKHoshizora/specmetry/actions/runs/40084692941)：push，attempt 1，**completed/success**，没有重跑；每个 Job 检出 `56234c3` 本身。

  | 平台 | 场景 | 门禁回归 | 组合 |
  |---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | 38/38 confirmed | 55/55 | 13/13 |
  | Windows | 29/29 confirmed | 51/51 | 13/13 |
  | macOS | 27/27 confirmed | 46/46 | 13/13 |

  - **N4 的各项**：`control.f1-skip-drain` 为 confirmed；`control.f1-read-kept` 为 refuted；`report.fail-after-ok`、`report.ok-not-final`、`report.close-fails` 与此前各项 F1 故障为 invalid；`report.epipe-late` 为 confirmed。
  - **macOS 的直接检查** `TestCensusUnreadable` 两行都通过。这次运行中没有出现普查的 EINVAL。
  - **其余**：
    - B1 的停止回归、`TestLingeringFailedStop`、`TestStopRead`、`TestStopAPIErrors`（Windows）都通过。
    - B2 的两项 `forge-owner` 中夹具零动作、零信号；本轮自有的遗留照常由被测补救结束；补救失败、观察失败、结束未知的组合仍使被测自检不通过。
    - N6 的时钟回归通过。
    - SKIP 只有三个只供内层运行的测试；场景 ID 的并集仍为 53。
  - **这次成功不能替代旧运行的未知结果**：run 40074633075 的 macOS 失败与 run 40072151282 的那一项，归属、原因与清理结果仍然未知（见上表）。
- **作者本机**（`56234c3` 的树）：
  - Windows 11：场景 29/29 confirmed；门禁回归 51/51；组合 13/13；直接检查都通过。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 55/55；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码）。
- **定向撤回**（作者本机 Windows 与 WSL，对 `56234c3`）：

  | 变异 | 撤回的内容 | 检出（两边相同） |
  |---|---|---|
  | M51 | 不要求最终回答 | `report.ok-unwritable`、`report.close-fails` 得到 refuted |
  | M52 | 不要求每一路的迟到字节 | `report.one-stream-seen` 得到 refuted |
  | M53 | 等待回答之后不再读失败记录 | `report.epipe-late` 得到 invalid |
  | M54 | 不要求观察结束（EOF） | `report.fail-after-ok`、`report.ok-not-final` 得到 refuted |
  | M56 | 输出关不上仍然回答 | `report.close-fails` 得到 refuted |
  | M59 | 迟到字节不计到结束为止 | `control.f1-read-kept` 得到 invalid |

  - 每个变异只由对应的检查检出；健康对照与真实管道场景都通过。
  - M59 的第一次写法编译失败（未使用的变量），没有计为检出，改为能编译的写法之后单独重跑。
  - macOS 的诊断（N-MAC）只能在 CI 的 macOS 上运行，没有本机的撤回验证；它的直接检查在回归先行时失败、修复之后通过。

**对 T1 / T6 证据的影响**：

- N4：refuted 现在依据一个有界、完整结束的观察段：持有者在关闭输出之后给出的最终回答，加上场景对每一路 EOF 的核实与每一路的迟到字节。只有 `control.f1-read-kept` 这一种构造得到 refuted。
- B1 在当前原型范围内关闭；挂起停止的资源上界与 C-4 / D20 仍是冻结之前的门槛。
- macOS：旧失败的归属与清理结果保持未知；本轮只增加了诊断信息，没有改变归属门禁。

**仍未验证的边界**：

- 持有者的回答仍是它自己的声明；EOF 证明的是写端都已关闭，不证明持有者没有在别处出错。
- macOS 普查读不到同一用户其他进程环境时判为失败，这一保守前提不变；若要区分“正在退出”或忽略 EINVAL，属于需要另行论证与批准的保证取舍，本轮没有实施。
- 不是硬实时保证；生产路径没有接入。

**合并条件**（其后由 §2.16 更正：不可追溯本身不再作为永久的合并门槛）：最终代码点的三平台诊断为 success，但旧 macOS 失败本身没有得到有效诊断（当时的事实不可恢复，新的诊断信息尚未遇到真实的同类失败）。按本轮指令，就此交付为 PR #34 **不具备合并条件**，不以最终诊断的成功或常规 CI 自行豁免；是否足够，由独立复审与负责人判断。

**状态**：B1-Ra、B1-Rb 经 Review 083 可以关闭（Review 081 B1 在当前原型范围内整体关闭）。N4-R-S 与 N-MAC **已修复，待独立复审**；不宣布关闭或 GO。N6 保持关闭范围。修复技术 GO 不等于合并条件满足。（其后：Review 084 认定 B-N4，`report.fail-after-ok` 等各项的证据只覆盖报告失败的组合，见 §2.16。）

### 2.16 Review 084：回答与失败记录的冲突（B-N4）与普查标签（N1）

**审查结论**（[Review 084](reviews/codex-review-084.md) 第一、三 ~ 五、八节，对 `2414ae9`）：

- **技术 NO-GO，BLOCKER B-N4**：有效的 `ok` 与真实的 `epipe` 记录同时存在时，冲突检查被跳过，内层 **confirmed / exit 0**。
  - **路径**：第一次等不到 `epipe`，场景发出询问，得到有效的 `ok`；之后重读得到 `epipe`。EOF、迟到字节与“回答旁有失败记录”的检查都在“`epipe` 仍为空”的条件之内，于是整段被跳过；恰好一条 `epipe` 被判为预测成立。
  - **报告的最终校验**只核对各类记录的数量、重复与格式，不拒绝这组语义矛盾；正常的清理也不能恢复观察的有效性。
  - **审查方的复现**（仓库外，真实 helper）：持有者写出真实的 `ok`，随后它的 stderr 被关闭，下一次写入真实失败，`epipe` 报告**写成功**，内层 confirmed / 0。外层原有的门禁正确检出了它。
  - 这是本 PR 要修的有效性门禁直接形成的绿色结论，所以列为 BLOCKER，不是旧的非阻塞残余；没有重开 B1。
- **N4-R-S 不能整体关闭**：旧反例、输出未最终关闭、关闭失败都已拒绝。
- **N-MAC**：
  - 新增的有限诊断与保守失败路径基本成立；旧两次失败的归属、根因与完整清理仍未知，真实的 EINVAL 还没有由新诊断重现。
  - **旧 macOS 失败本身不构成永远不得合并的门槛**；不要求反复运行直到旧的 EINVAL 再现。
- **NON-BLOCKER N1**：`same program as this test` 实际只是短命令名的前缀匹配（启发式），标签比证据强。
- B1、N6 没有直接回退。
- 两套 CI 正式成功（run 40084692941、40087043315），但都没有覆盖 B-N4 的组合，不能替代技术修正。
- 没有新增产品决定、保证调整或接受风险。

**对 §2.15 的更正**（§2.15 原文不改）：

- §2.15 的“合并条件”一段，把“旧 macOS 失败本身没有得到有效诊断”作为“不具备合并条件”的依据。按 Review 084 与负责人本轮的指示，不可追溯本身不再作为永久的合并门槛。
- 旧的未知项继续保留，不称为已解决：2088 与 1989 的归属、EINVAL 的原因与当时的完整清理，仍然未知。
- §2.15 中“`report.fail-after-ok` 等各项为 invalid”的证据成立，但只覆盖报告失败的组合；回答与报告成功的失败记录并存的组合，当时没有回归。

**回归先行**（`7307d5d20835d0947caf21e3216f1130e75b359f`，判定没有改；真实 helper、写入与报告路径）：

| 门禁回归 | 持有者做什么 | 旧代码上 |
|---|---|---|
| `report.ok-epipe-reported` | 审查方的精确反例：真实的 `ok`；随后 stderr 断开，下一次写入失败，`epipe` 报告写成功 | **confirmed / 0**，失败 |
| `report.epipe-then-ok` | 相反的顺序：关闭输出之后的一次写入失败并报告 `epipe`，然后仍回答 `ok` | **confirmed / 0**，失败 |
| `report.ok-unasked` | 未被询问就写 `ok`，之后的写入失败并报告 `epipe`（第一次读取就找到 `epipe` 的路径） | **confirmed / 0**，失败 |
| `report.epipe-foreign` | `epipe` 记录写的是另一个进程的 PID | **confirmed / 0**，失败 |
| `report.ok-foreign` | `ok` 记录写的是另一个进程的 PID | **refuted / 1**，失败 |
| `report.ok-duplicate`、`report.ok-truncated`、`report.ok-malformed` | 重复、截断、畸形的 `ok` | invalid（旧代码已拒绝） |
| `report.epipe-limit`、`report.ok-limit` | 任一记录旁边，持有者报告到达自己的硬期限（工具失败） | invalid（旧代码已拒绝） |

- **场景的设置**：新故障与旧故障一样保留两路读端，并在回答之后等待持有者自己的标记；这只控制调度，不改判定。
- **三平台诊断** [run 40095937741](https://github.com/XKHoshizora/specmetry/actions/runs/40095937741)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `7307d5d`。
  - 三平台都只有上表中预期的 5 项失败：门禁回归 Ubuntu 60/65、macOS 51/56、Windows 56/61。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- **作者本机 Windows**：与上表相同（前 4 项 confirmed / 0，`report.ok-foreign` refuted / 1，其余 5 项 invalid）。

**修复 B-N4**（`173d2eb`）：证据的收集与最终分类分开。

- **协议**（`report_test.go`）：
  - `epipe`：持有者的一次迟到写入失败；
  - `ok`：只在被询问时、只在输出已经关闭之后给出，表示它的写入都没有失败；
  - 两者互斥：不论先后，两者并存都违反协议，任何一个都不能作为成功的依据。
- **收集**（`observeF1`，持有者可能仍在运行）：
  - 等待失败记录；没有时发出询问，等待回答，并在一个共用的 2 s 预算内把每个保留读端读到 EOF，记下迟到字节；
  - 途中的失败都记下，收集照常继续，不因某一条记录或回答而提前结束。
- **最终读取**（`finalRead`，持有者结束之后）：
  - 不再有写者，报告就是这次观察可能有的全部；严格读取，截断的记录现在是错误；
  - 持有者的结束没有确认时，不算最终读取。
- **分类**（`classifyF1`，在全部证据之后）：每条规则都检查之后才决定，成功不提前返回。

  | 证据 | 结论 |
  |---|---|
  | 只有持有者的 `epipe`（包括等待回答期间到达的），没有失败 | 预测成立 |
  | 只有持有者被询问之后的 `ok`；每个保留读端都到 EOF；每一路都保留并有迟到字节；没有失败 | refuted |
  | `ok` 与 `epipe` 并存（不论先后） | invalid |
  | 记录不是持有者的 PID；同类记录多于一条 | invalid |
  | 收集或最终读取中的任何失败（读取、写入、报告的格式、硬期限），不论旁边有哪种记录 | invalid |
  | 两种记录都没有；`ok` 而观察没有结束、读端不全或缺迟到字节；没有被询问的 `ok` | invalid |

- **同一分类链的扫描**：
  - 第一次读取就找到 `epipe` 时，从不看 `ok`：由统一的分类覆盖（`report.ok-unasked`）；
  - 回答路径中的检查只在 `epipe` 为空时执行：同上（`report.ok-epipe-reported`、`report.epipe-then-ok`）；
  - 记录的 PID 没有与已核验的持有者核对：现在核对（`report.epipe-foreign`、`report.ok-foreign`）；
  - 分类时，持有者仍可能写入：现在在持有者结束之后最后读一次；
  - 其余的提前结束都是 `invalidf`，只能得到 invalid。没有发现其他同源的成功提前返回；没有扩大到其他场景。
- **随修复增加**：
  - 直接检查 `TestF1Classify`：固定的证据行（健康的 `epipe`、健康的最终回答、两种顺序的冲突、失败与各种记录并存、他人的记录、重复、观察未结束、读端不全、缺迟到字节、未被询问的回答），每一行都另以记录逆序再判一次，结论不得改变；
  - 门禁回归 `report.epipe-read-fails`（健康的 `epipe` 旁，场景的最终读取失败）与 `report.ok-drain-fails`（完整的回答旁，保留读端的读取失败）。旧代码没有最终读取，不声称旧代码上的对照。
- 只调整原型协议，不决定生产的产品规则。

**修复 N1**（`1e82fbd`）：标签改为“短命令名是本测试名称的前缀（启发式，不是身份）”，函数改名为 `commPrefix`。仍不输出原名；普查照样失败，不据此动作。

**修复后的证据**（`1e82fbd`）：
- **三平台诊断** [run 40096356396](https://github.com/XKHoshizora/specmetry/actions/runs/40096356396)：push，attempt 1，**completed/success**，没有重跑；每个 Job 检出 `1e82fbd` 本身。

  | 平台 | 机制场景 | 门禁回归 | 组合 |
  |---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | 38/38 confirmed | 67/67 | 13/13 |
  | Windows | 29/29 confirmed | 63/63 | 13/13 |
  | macOS | 27/27 confirmed | 58/58 | 13/13 |

  - **F1**：`control.f1-skip-drain` 为 confirmed / 0，`control.f1-read-kept` 为 refuted / 1，`report.epipe-late` 为 confirmed / 0；其余各项 F1 故障（含本轮新增的 12 项）都是 invalid / 1。正确拒绝无效证据是门禁回归的通过，不计为机制的 confirmed。
  - **直接检查**都通过：`TestF1Classify`（19 行，三平台）、`TestCensusUnreadable`（macOS）、`TestStopAPIErrors`（Windows）、`TestStopRead`、`TestLingeringFailedStop`、`TestPhaseClock`、`TestGateJudge`、`TestCandidates`。
  - **B2 与归属保护**：
    - `forge-owner` 拒绝伪造的 PID，零动作；
    - `forge-owner-mixed` 结束本轮自有的遗留，拒绝伪造的 PID；
    - 补救失败、结束未确认、普查错误、状态未知的组合仍使被测自检不通过。
  - **N6**：时钟回归通过。
  - **SKIP**：只有三个只供内层运行的测试；场景 ID 的并集为 53。
  - 没有出现普查的 EINVAL。
- **作者本机**（`1e82fbd` 的树）：
  - Windows 11：场景 29/29 confirmed；门禁回归 63/63；组合 13/13；直接检查都通过。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 67/67；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码）。
- **定向撤回**（作者本机 Windows 与 WSL，对 `1e82fbd`；每次只撤回一处，都编译通过）：

  | 变异 | 撤回的内容 | 检出（两边相同） |
  |---|---|---|
  | M60 | 冲突检查（`ok` 与 `epipe` 并存） | `report.ok-epipe-reported`、`report.epipe-then-ok` 得到 confirmed；`TestF1Classify` 的两种顺序 |
  | M61 | 记录 PID 与持有者的核对 | `report.epipe-foreign` 得到 confirmed、`report.ok-foreign` 得到 refuted；`TestF1Classify` 的两行 |
  | M62 | 收集中的失败不计入分类 | `report.ok-drain-fails` 得到 refuted；`TestF1Classify` 的两行 |

  - 每个变异只由对应的检查检出。健康的 `control.f1-skip-drain`（confirmed）、合法的 `control.f1-read-kept`（refuted）、`report.epipe-late`（confirmed）与 5 个顶层机制场景都保持通过。
  - 在 M62 下，`report.epipe-read-fails` 仍为 invalid：最终读取失败时没有记录可分类，结论仍是 invalid。它不是这一撤回的检出。

**对 T1 / T6 证据的影响**：

- F1 的 confirmed 与 refuted 现在都依据完整收集的证据，与持有者结束之后的最终报告；`epipe` 与 `ok` 互斥。
- `173d2eb` 之前的各次运行中，回答与报告成功的失败记录并存、他人 PID 的记录都没有被注入；场景的结论作为观测保留。

**仍未验证的边界**：

- 持有者的回答与记录仍是受信任 helper 的声明。PID 核对只排除指向其他进程的记录，不是对写者的认证。
- 持有者在被结束之前没有写出的记录不在报告中：观察以它的回答与 EOF 为终点，按协议此后的行为不属于这次观察。（这一句不成立，由 §2.17 更正：它排除了观察期间已经发生、但尚未报告或报告失败的错误，`epipe` 路径也没有这样的终点。）
- 原生的读取或报告失败没有构造，场景一侧的两项工具失败只由注入模拟。
- macOS 的旧未知项保持；N1 只改标签。
- 不是硬实时保证；生产路径没有接入。

**状态**：B-N4 与 N1 **已修复，待独立复审**；不宣布关闭或 GO。（其后：Review 085 关闭 B-N4 与 N1，认定 B-N4-R，见 §2.17。）

### 2.17 Review 085：工具完成的证明（B-N4-R）

**审查结论**（[Review 085](reviews/codex-review-085.md) 第一、三、四、五、八节，对 `4d1e575`）：

- **原 B-N4 的冲突旁路可以关闭**：完整写出的 `ok` 与 `epipe` 不再因到达轮次或记录顺序绕过检查；这不等于整个 N4 收集链已无阻塞。
- **N1 可以关闭**：标签、函数与文档都已明确为短命令名前缀的启发式。
- **技术 NO-GO，BLOCKER B-N4-R**：把持有者被结束之后报告不再变化，等同于工具已经完成交接。
  - **路径**：第一次读取就看到 `epipe` 时，不询问、也没有 EOF 的交接；场景随即结束持有者，只凭进程已结束，就把最终读取当作“报告包含观察可能有的全部”。这只证明此后不会再追加，不证明产生 `epipe` 的报告调用已成功返回，也不证明没有工具失败、或尚待写出的失败报告已完成。
  - **审查方的三个反例**（仓库外，真实 helper；内层都是 **confirmed / exit 0**，外层原有门禁正确检出）：
    - 完整的 `epipe` 记录之后，报告的 Close 失败（第二次 Close 的真实错误）；
    - 已到达工具期限的故障点，它的报告被父端的结束截断；放开闸门、让报告完成的反向对照为 invalid / 1；
    - `epipe` 之后，工具期限的报告真实写入失败。
  - **观察终点的表述需要更正**：§2.16 的“持有者在被结束之前没有写出的记录不在报告中……此后的行为不属于这次观察”不能覆盖观察或交接期间已经发生、但报告尚未写出或报告调用失败的错误，也不能用结束之后干净的报告证明完成。只有明确完成交接之后的无关行为，才可以在受控协议中排除。
  - **最小修正**：为接受证据定义可验证的工具完成 / 失败状态，在 `epipe` 与 `ok` 两条路径都适用；主动结束只能作为清理，不能自动认证观察有效。
- B1、N6 没有直接回退；两套 CI 正式成功，但不覆盖本轮的组合反例。
- 没有新增产品决定、保证调整或接受风险。

**对 §2.16 的更正**（§2.16 原文不改）：

- §2.16“仍未验证的边界”中“持有者在被结束之前没有写出的记录不在报告中：观察以它的回答与 EOF 为终点，按协议此后的行为不属于这次观察”的说法不成立：
  - 它把观察期间已经发生、但报告尚未写出或报告调用失败的错误也排除在外；
  - `epipe` 路径根本没有回答与 EOF 的终点。
- 观察的终点改为持有者的完成提交（下文）；只有完成提交之后的行为不属于这次观察。
- §2.16 `finalRead` 的注释“不再有写者，报告就是这次观察可能有的全部”同样过强，已改为：结束之后报告不会再追加；记录是否是观察的全部，只由持有者的完成证明。

**完成协议**（状态与证据）：

| 阶段 | 持有者做什么 | 证据 | 失败或缺失时 |
|---|---|---|---|
| 目标写入 | 停止之后继续写两路 | 写入失败：`epipe` 的依据。写入成功：保留读端上的迟到字节 | — |
| 报告写入 | 追加一条 `epipe` 或 `ok` 记录（一次 Write） | 最终报告中持有者 PID 的完整记录 | Write 失败：不提交完成 |
| 报告关闭 | Close | 只由完成提交证明 | Close 失败：不提交完成 |
| 其他工具操作 | 例如到达硬期限并报告 `limit` | 写出的 `limit` 记录由最终读取拒绝 | 任一工具失败：不提交完成 |
| 观察完成 | `ok` 路径：先关闭输出再回答。`epipe` 路径：写入失败已经发生 | `ok` 路径：每个保留读端的 EOF 与迟到字节 | 缺：invalid |
| **完成提交** | 最后一步：把 `done <pid>` 写入自己的临时文件并 Close，再改名为 `<report>.done`（原子：全有或全无） | `.done` 存在，内容严格为已核验持有者的 PID | 任一步失败，或之前被结束：没有 `.done` |
| 自然结束 | 提交之后：`epipe` 路径以 3 退出；`ok` 路径等待被结束 | 不作为证据（持有者的退出码不能跨平台读取） | — |
| 被结束 | 场景在完成提交之后、或 3 s 期限之后结束它 | 结束确认 | 结束未确认：invalid |
| 最终读取 | — | 结束之后严格读取报告：截断、畸形、重复、`limit` 都是错误 | 错误：invalid |
| 最终分类 | — | 收集的全部证据、最终报告与完成提交 | 见下表 |

- **证明预测现象发生的证据**：最终报告中只有一条属于已核验持有者的完整 `epipe` 记录。
- **证明观察工具正常完成的独立证据**：持有者的完成提交（`.done`）。它在全部报告操作成功、没有工具失败之后才写，并以改名原子地出现。`ok` 路径另需每个保留读端的 EOF 与每一路的迟到字节。
- **报告失败由谁保存、经什么路径传给最终判定**：
  - 持有者遇到报告或工具失败时，不提交完成：`epipe` 路径以 `exitReport` 退出（或按注入存活一段有界的时间），`ok` 路径等待被结束；
  - “没有完成提交”由场景的完成等待记入证据（`ev.done == 0`），分类判 invalid；
  - 写出的失败记录（`limit`）与格式错误，由最终读取记入 `failures`；
  - 场景自身的读取、写入与结束确认的失败，也记入 `failures`。
- **强制结束是否可能截断仍属于本次观察的报告**：可能，但被截断的观察不会被接受。场景只在看到完成提交之后、或 3 s 期限之后才结束持有者。完成提交之前的结束，结果就是没有提交，于是 invalid。完成提交之后，持有者按协议不再有属于观察的操作。
- **缺少完成证据时**：`classifyF1` 对 `done == 0` 与 `done` 不是持有者的情形都给出 invalid 理由；confirmed 与 refuted 都需要它。
- **完成与错误的交接**：提交是持有者最后一个可能失败的操作，并且原子可见。在它之前发生的任何失败都会使它缺失，场景对缺失判 invalid；在它之后，没有属于观察的操作。因此不存在无人收集的窗口。（这一条只对主路径上返回值受检查的串行操作成立，不覆盖并发的期限回调与被忽略的报告失败；“原子可见”也过宽。由 §2.18 更正。）
- **期限**：等待失败记录 3 s、等待回答 2 s、读到 EOF 共用 2 s、等待完成提交 3 s，结束确认按既有的 `tKill`；没有无限等待。

**错误传播**：

| 错误 | 发生在 | 保存者与路径 | 最终结论 |
|---|---|---|---|
| 目标写入失败 | 持有者 | `epipe` 记录 | 预测成立（另需完成提交） |
| 报告的 Write / Close 失败 | 持有者 | 不提交完成 → `ev.done == 0` | invalid |
| 其他工具失败（硬期限） | 持有者 | 不提交完成；若已写出，最终读取的 `limit` 错误 → `failures` | invalid |
| 失败的报告被结束截断 | 持有者 / 场景的结束 | 没有完成提交；截断的记录由最终读取拒绝 | invalid |
| 完成提交本身失败（写、Close、改名） | 持有者 | 没有 `.done` | invalid |
| 完成提交畸形或不是持有者的 PID | 持有者 | `awaitDone` 的错误 → `failures`；或 `ev.done` ≠ 持有者 | invalid |
| 询问写不出、保留读端读取失败 | 场景 | `failures` | invalid |
| 结束未确认 | 场景 | `finalRead` → `failures` | invalid |
| 最终读取失败、截断、畸形、重复 | 场景 | `finalRead` → `failures` | invalid |
| 回答与失败记录并存、他人的记录 | 持有者 | 最终报告 → `classifyF1` | invalid |
| 停止、观察、清理的错误 | 场景 | 既有的 `adopt` / `done` / `flag` | invalid |

**回归先行**（`acfee1fa9d2a4665b2f14858aa526b833dc63130`，判定没有改；真实 helper、报告、结束确认与场景分类路径）：

| 门禁回归 | 持有者做什么 | 旧代码上 |
|---|---|---|
| `report.epipe-close-fails` | 完整写出 `epipe`，然后报告的 Close 失败（第二次 Close） | **confirmed / 0**，失败 |
| `report.epipe-limit-cut` | `epipe` 之后到达硬期限的故障点；它的报告 10 s 之后才写，在此之前被场景结束 | **confirmed / 0**，失败 |
| `report.epipe-limit-unwritable` | `epipe` 之后，硬期限的报告真实写入失败 | **confirmed / 0**，失败 |
| `report.ok-close-fails` | 完整写出 `ok`，然后报告的 Close 失败 | **refuted / 1**，失败 |
| `report.ok-limit-cut` | `ok` 之后同样被截断 | **refuted / 1**，失败 |
| `report.ok-limit-unwritable` | `ok` 之后同样写入失败 | **refuted / 1**，失败 |
| `report.epipe-cut` | 记录本身写到一半时被场景结束 | invalid（旧代码已拒绝） |

- **调度**：在新故障下，场景先限时等待持有者自己的标记，确认故障已经发生，再结束它。这只控制调度，不改判定。作者本机对 6 项的直接运行都记录到标记已出现。
- **三平台诊断** [run 40102730930](https://github.com/XKHoshizora/specmetry/actions/runs/40102730930)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `acfee1f`。
  - 三平台都只有上表中预期的 6 项失败：门禁回归 Ubuntu 68/74、macOS 59/65、Windows 64/70。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- **作者本机 Windows**：与上表相同。

**修复 B-N4-R**（`fb3f56d`）：

- **持有者**：`complete()` 是最后一步。只有记录与其后的工具操作都成功时才调用；它写 `done <pid>`、Close、改名为 `<report>.done`。任一步失败都没有完成提交。
- **场景**：
  - 在两条路径上，结束持有者之前都在 3 s 内等待完成提交，并严格读取（`awaitDone`）；
  - 最终读取的注释更正为“结束之后不再追加；是否是全部，只由完成证明”；
  - `classifyF1` 对没有完成提交、或完成提交不是持有者的，判 invalid；confirmed 与 refuted 都需要它。
- **协议**写入 `report_test.go` 的头注释：完成提交不是记录，是持有者的最后一步。
- **随修复增加**：
  - 门禁回归 `report.done-close-fails`（健康的 `epipe` 旁，完成提交的 Close 失败）、`report.done-foreign`（完整的回答旁，完成提交写的是另一个进程）。旧代码没有完成提交，不声称旧代码上的对照。
  - `TestF1Classify` 增加 4 行：两条路径上没有完成提交、或完成提交属于其他进程。
- **不是**增加一次重读或延迟，也不是针对注入名称的判定：完成证明对所有 F1 运行都适用，注入只决定持有者在哪一步失败。
- 只调整原型协议，不决定生产的产品规则。

**修复后的证据**（`fb3f56d`）：
- **三平台诊断** [run 40103135762](https://github.com/XKHoshizora/specmetry/actions/runs/40103135762)：push，attempt 1，**completed/success**，没有重跑；每个 Job 检出 `fb3f56d` 本身。

  | 平台 | 机制场景 | 门禁回归 | 组合 |
  |---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | 38/38 confirmed | 76/76 | 13/13 |
  | Windows | 29/29 confirmed | 72/72 | 13/13 |
  | macOS | 27/27 confirmed | 67/67 | 13/13 |

  - **F1**：
    - `control.f1-skip-drain` 为 confirmed / 0，`control.f1-read-kept` 为 refuted / 1，`report.epipe-late` 为 confirmed / 0。这三项都在持有者的完成提交之后得出。
    - 其余各项 F1 故障，包括本轮新增的 9 项，都是 invalid / 1。正确拒绝无效证据是门禁回归的通过，不计为机制的 confirmed。
  - **直接检查**都通过：`TestF1Classify`（23 行，三平台）、`TestCensusUnreadable`（macOS）、`TestStopAPIErrors`（Windows）、`TestStopRead`、`TestLingeringFailedStop`、`TestPhaseClock`、`TestGateJudge`、`TestCandidates`。
  - **B2 与归属保护**：
    - `forge-owner` 拒绝伪造的 PID，零动作；
    - `forge-owner-mixed` 结束本轮自有的遗留，拒绝伪造的 PID；
    - 补救失败、结束未确认、普查错误、状态未知的组合仍使被测自检不通过。
  - **N6**：时钟回归通过。**B1**：停止的回归都通过。
  - **SKIP**：只有三个只供内层运行的测试；场景 ID 的并集为 53；没有出现普查的 EINVAL。
  - **耗时**：Ubuntu 的 `go test` 为 358 s（上限 15 min），Job 6.2 min（上限 20 min）；预算没有改。
- **作者本机**（`fb3f56d` 的树）：
  - Windows 11：场景 29/29 confirmed；门禁回归 72/72；组合 13/13；直接检查都通过。
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 76/76；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码）。
- **定向撤回**（作者本机 Windows 与 WSL，对 `fb3f56d`；每次只撤回一处，都编译通过）：

  | 变异 | 撤回的内容 | 检出（两边相同） |
  |---|---|---|
  | M63 | 场景不等待完成提交，直接认定已完成 | 记录之后报告失败或被截断的 6 项（`epipe` 的 3 项为 confirmed，`ok` 的 3 项为 refuted），以及 `report.done-close-fails`（confirmed）、`report.done-foreign`（refuted） |
  | M64 | 持有者在记录或工具操作失败之后仍提交完成（两条路径） | `report.epipe-close-fails`、`report.epipe-limit-unwritable`（confirmed）；`report.ok-close-fails`、`report.ok-limit-unwritable`（refuted） |
  | M65 | 接受其他进程的完成提交 | `report.done-foreign`（refuted）；`TestF1Classify` 的两行 |

  - 每个变异只由对应的检查检出。健康的 `control.f1-skip-drain`（confirmed）、合法的 `control.f1-read-kept`（refuted）、`report.epipe-late`（confirmed）与 5 个顶层机制场景都保持通过。
  - M64 之下，被截断的两项（`...-limit-cut`）与 `report.done-close-fails` 仍为 invalid：截断时持有者还没有走到提交，提交本身失败时也没有提交。它们不是这一撤回的检出，由 M63 检出。

**对 T1 / T6 证据的影响**：

- F1 的 confirmed 与 refuted 现在都需要持有者的完成提交：记录的字节完整，不再被当作报告操作成功的证明；结束持有者只是清理。
- `fb3f56d` 之前的各次运行中，记录之后报告失败或被截断的组合都没有注入；场景的结论作为观测保留。

**仍未验证的边界**：

- **信任边界**：持有者仍是受信任的 helper。完成提交证明的是它自己声明的“全部报告操作成功”；一个在失败之后仍然提交的持有者不在模型中（定向撤回 M64 检查本 helper 不这样做）。
- **注入的性质**：报告的 Close 失败是第二次 Close 的真实错误，期限故障点是既有的模拟故障点，不是原生的存储故障或真实到期。
- **原子性的前提**：完成提交依赖同一目录内改名的原子性（Unix 的 `rename`、Windows 的 `MoveFileEx`），只在受控的临时目录中使用。
- **观察的范围**：完成提交之后的行为不属于观察；这是受控协议的限定，不是对任意程序的保证。
- macOS 的旧未知项保持，不作为永久合并门槛；不是硬实时保证；生产路径没有接入。

**状态**：B-N4-R **已修复，待独立复审**；不宣布关闭或 GO。（其后：Review 086 认定 B-N4-R/T、B-N4-R/I 与 N-RENAME，见 §2.18。）

### 2.18 Review 086：持有者各入口的统一完成协议（B-N4-R/T、B-N4-R/I）

**审查结论**（[Review 086](reviews/codex-review-086.md) 第一、三 ~ 六、九节，对 `bc5ad78`）：

- **Review 085 的三类原反例已经修复**：完成标记正确拒绝它们；健康的 `epipe`、合法的 `ok` / EOF / 迟到字节反证、`epipe-late` 保持正确结果。
- **技术 NO-GO，B-N4-R 不能整体关闭**，N4 系列尚未收口。两项确定的残余：
  - **B-N4-R/T（BLOCKER）**：真实的期限回调已经进入、`limit` 报告仍在进行时，主路径仍能发布 `done`。
    - 根源：`time.AfterFunc` 的回调在独立的 goroutine 中执行，`complete` 不读取它的状态，也不与它同步。
    - 审查方的复现：真实计时器缩短到 100 ms，回调进入后在可释放闸门处等待。结果 `epipe` 路径为 confirmed / 0，`ok` 路径为 refuted / 1；先放行回调的反向对照为 invalid / 1。
  - **B-N4-R/I（BLOCKER）**：已有的 helper 故障分支忽略报告的返回值。
    - `ok` 报告真实写入失败之后，仍能生成 `epipe` 与有效的 `done`，内层 confirmed / 0。
    - 反序分支同样忽略 `epipe` 报告的返回值；审查方只指出了源码风险，没有动态复现。
- **完成标记目前支持的范围**：正常串行路径上的报告、后处理与完成文件的交接，以及缺失、错误 PID、格式错误的保守拒绝。它还不能证明这个 helper 中所有相关的写者都已结束、也没有被忽略的早先报告错误。
- **NON-BLOCKER N-RENAME**：§2.17 把同目录改名直接表述为“原子：全有或全无”。Go 对非 Unix 平台不作这种无条件承诺；措辞应限定在已验证的条件之内。
- B1、N6 与已关闭的 B-N4、N1 没有直接回退；两套 CI 正式成功，但不覆盖本轮的组合。
- 没有新增产品决定、保证调整或接受风险。

**对 §2.17 的更正**（§2.17 原文不改）：

- §2.17 中“在它之前发生的任何失败都会使它缺失”“提交是持有者最后一个可能失败的操作”只对**主路径上返回值受检查的串行操作**成立。它没有覆盖并发的期限回调，也没有覆盖调用者忽略返回值的报告。
- §2.17 中“完成提交以改名原子地出现（全有或全无）”收窄为：在场景自己的临时目录内同目录改名，并对内容做严格的完整校验；这是本原型在三平台 CI 与作者本机（Windows NTFS、WSL2）上已验证的布局，不推广为任意文件系统或平台的原子性保证。

**持有者的入口与共享状态**：

| 入口 | 执行者 | 经过工具状态的哪一步 |
|---|---|---|
| 主路径（目标写入、询问的回答、失败路径） | 持有者的主 goroutine | 记录与输出关闭都经 `report` / `failed` |
| 期限回调（`time.AfterFunc`） | 计时器的独立 goroutine | 第一步 `limitEntered` |
| 模拟期限点（门禁回归） | 主 goroutine | 同上，`limitEntered` |
| 报告操作（`report`，任何种类） | 主路径或回调 | `tool.report`：执行并保存失败 |
| 完成提交（`complete`） | 主 goroutine | `tool.complete`：一次性取得终态 |

- **共享状态**（`toolState`，`tool_test.go`）：第一个失败 `err`（只设一次，从不清除）、期限已进入 `limit`、完成已取得终态 `final`、完成已发布 `published`。一把锁保护全部字段。
- **所有权**：状态归 helper 进程所有；只能经过上述四个入口修改。场景不读取它，只读取完成标记与报告。

**统一的完成协议**：

| 问题 | 规则 |
|---|---|
| 状态由谁修改 | 只有 `failed`、`report`、`limitEntered` 与 `complete`，都在同一把锁之下 |
| 错误由谁保存 | `report` 在操作返回之后保存它的失败，不论调用者是否检查；持有者的输出关闭经 `failed` 保存；期限进入时保存 `errLimit`；完成发布本身的失败由 `complete` 保存。只保存第一个，之后的成功不清除 |
| 期限回调尚未进入 | 对完成没有影响 |
| 期限回调已进入 | 第一步在锁下裁定：完成已发布，则只结束进程；否则 `limit = true`、工具失败，此后不能再发布完成，再去报告 `limit` 并结束进程 |
| 回调进入时完成正在发布 | `complete` 在整个发布期间持有锁，回调等到完成取得终态之后才裁定，因此只会看到“已发布”或“未发布”，不会看到中间状态 |
| 何时允许发布完成 | 只在 `complete` 第一次取得终态时，并且没有保存的失败、期限没有进入；之后再调用都拒绝 |
| 完成之后 | `report` 不再写入（返回 `errAfterCompletion`）；回调只结束进程，不报告 |
| 已进入回调的收尾 | 赢得裁定的回调报告 `limit` 并以 `limitExit` 结束；被场景结束时，因为没有完成，观察无效。输给完成的回调直接以 `limitExit` 结束。不依赖 `Timer.Stop`，也不把 Stop 当作 join |
| 谁消费最终结果 | 场景：在 3 s 内等待完成标记并严格读取（`awaitDone`）；结束持有者之后严格读取报告（`finalRead`）；`classifyF1` 要求已核验持有者的完成 |
| 超期或未知 | 3 s 内没有完成标记：`done = 0`，invalid。完成标记畸形、不可读或不是持有者的：invalid。结束未确认：invalid。场景从不等待回调本身 |
| 同步的代价 | 只有完成发布在它的文件操作期间持有锁；报告操作只在检查与保存时短暂持有，执行操作本身时不持有。回调最多等待一次完成发布；锁内不调用其他入口，没有无界等待、死锁或无人负责的任务，期限与预算没有改变。（措辞经 Review 087 N-087-2 更正：没有引入协议级的循环等待，父端期限不变；文件 I/O 与系统调度不提供硬实时保证，§2.19。） |

- 同一状态不会既被确认为“正常完成”，又被确认为“期限失败”：`published` 与 `limit` 在锁下互斥地设置，一旦其中一个成立，另一个就不能再成立。
- 已经发生的失败，不靠把它划到观察范围之外来解决；完成之后真正无关的行为，只是不再启动属于本次观察的报告。

**错误传播**（在 §2.17 的表上补充）：

| 错误 | 发生在 | 保存者与路径 | 最终结论 |
|---|---|---|---|
| 被调用者忽略的报告失败 | 主路径的任一分支 | `tool.report` → `err` → `complete` 拒绝 → 没有完成标记 | invalid |
| 先失败后成功 | 主路径 | `err` 只设一次，之后的成功不清除 | invalid |
| 输出关闭失败 | 主路径 | `tool.failed` → `err` | 不回答；没有完成 |
| 期限回调已进入，`limit` 报告仍在进行或被截断 | 回调 | `limitEntered` → `limit`、`err` → `complete` 拒绝 | invalid |
| 期限回调在完成发布之后进入 | 回调 | 裁定为“已发布”，只结束进程 | 不改变已完成的观察 |
| 完成发布失败（写入、Close、改名） | 主路径 | `complete` 保存失败 → 没有完成标记 | invalid |

**回归先行**（`750e1390d6cc8cc211c144469d582cad239aca77`，判定没有改；真实 helper、报告、完成发布与场景分类路径）：

| 门禁回归 | 持有者做什么 | 旧代码上 |
|---|---|---|
| `report.limit-race-epipe` | 真实的期限计时器提前到 100 ms；回调进入后留下标记，在可释放闸门处等待（有界）；`complete` 等到这个标记之后才发布，`epipe` 路径 | **confirmed / 0**，失败 |
| `report.limit-race-ok` | 同上，回答路径（保留两路读端） | **refuted / 1**，失败 |
| `report.ok-fails-epipe-reported` | 回答的报告以只读方式打开、真实写入失败，调用者忽略；之后 `epipe` 报告成功并发布完成 | **confirmed / 0**，失败 |
| `report.epipe-fails-then-ok` | 相反方向：被忽略的 `epipe` 报告失败，之后回答成功并发布完成 | **refuted / 1**，失败 |
| `report.done-rename-fails` | 完成标记的改名失败（目标是目录）：对照 | invalid（旧代码已拒绝） |

- **闸门固定顺序**：竞争的两项由回调的“已进入”标记与 `complete` 对它的等待决定先后，不靠随机延迟；闸门由文件释放，或 10 s 之后自行放开，持有者在此之前被场景结束。
- **第一次运行的夹具错误**（如实保留，不计为复现）：`report.ok-fails-epipe-reported` 第一次没有加入保留读端的列表，持有者在被询问之前就因写入失败走了健康路径（命中数 0）。修正夹具之后，这一项命中 3 次，并得到 confirmed / 0。
- **三平台诊断** [run 40109659805](https://github.com/XKHoshizora/specmetry/actions/runs/40109659805)：push，attempt 1，completed/failure（预期）；每个 Job 检出 `750e139`。
  - 三平台都只有上表中预期的 4 项失败：门禁回归 Ubuntu 77/81、macOS 68/72、Windows 73/77。
  - 场景 38、27、29 confirmed；组合 13/13。
  - 没有出现普查的 EINVAL。
- **作者本机 Windows**：与上表相同。

**修复**（`9e3b7ca`）：

- **新增 `tool_test.go`**：`toolState` 与它的四个入口（`failed`、`report`、`limitEntered`、`complete`），规则见上表。
- **helper**：
  - `report` 成为通往报告的唯一入口，把每次操作交给 `tool.report`；原来的写入逻辑移到 `reportOnce`；
  - 期限回调第一步调用 `tool.limitEntered()`，只有赢得裁定时才报告 `limit`；
  - 门禁回归的模拟期限点也经过 `limitEntered`；
  - 持有者的输出关闭经 `tool.failed`；
  - `complete` 经 `tool.complete` 发布。
- **扫描所有 `report` 调用**：`pid`（4 处）、`fanout`、`epipe`、`ok`（含两个故障分支中被忽略的调用）、`limit`（回调与模拟点）。由于失败在报告层保存，调用点是否检查返回值都不再影响完成；不按注入名称决定结论。
- **随修复增加**：
  - 门禁回归 `report.limit-after-done`（要求 confirmed）：完成先发布，然后真实的回调进入并裁定，只结束持有者。命中数为 2，证明回调确实在持有者结束之前裁定。旧代码没有裁定，它的回调会在完成之后报告 `limit`，所以不声称旧代码上的对照。
  - 直接检查 `TestToolState`（5 组）：健康的发布、被忽略的失败之后的成功、期限先进入、发布失败，以及发布进行中回调进入（回调必须等到终态并输给已发布的完成）。（Review 087 N-087-1：这一组检查锁与终态的规则，但没有确证回调已在发布期间进入，§2.19。）
- 分类器没有改动。只调整原型协议，不决定生产的产品规则。

**夹具更正**（`f728e11498a8b4bc5b71ec3cbff1618c795f6969`）：

- **发现**：`9e3b7ca` 的诊断 [run 40110180232](https://github.com/XKHoshizora/specmetry/actions/runs/40110180232)（push，attempt 1，completed/success）中，`report.limit-after-done` 在 Ubuntu 与 Windows 上只有 1 次命中，macOS 为 2 次。场景看到完成之后就结束了持有者，回调还没有裁定；这一项所要检验的顺序（完成在先、回调在后）在那两个平台上没有发生，它的通过是空的。作者本机的 WSL 运行中也出现了同样的 1 次命中。
- **更正**：在这一故障下，场景结束持有者之前，限时等待回调“已裁定”的标记；没有这个标记，负载就没有建立，这次运行判 invalid，而不是通过。这只决定先后，不改变分类。
- run 40110180232 保留为历史观测；它的其余结果与之后的运行相同。
- **修复代码固定为 `f728e11`**（相对 `9e3b7ca` 只有这一处夹具差异）。

**修复后的证据**（`f728e11`）：
- **三平台诊断** [run 40110917799](https://github.com/XKHoshizora/specmetry/actions/runs/40110917799)：push，attempt 1，**completed/success**，没有重跑；每个 Job 检出 `f728e11` 本身。

  | 平台 | 机制场景 | 门禁回归 | 组合 |
  |---|---|---|---|
  | Ubuntu（带 `-race`，没有数据竞争的报告） | 38/38 confirmed | 82/82 | 13/13 |
  | Windows | 29/29 confirmed | 78/78 | 13/13 |
  | macOS | 27/27 confirmed | 73/73 | 13/13 |

  - **F1**：
    - `control.f1-skip-drain` 为 confirmed / 0，`control.f1-read-kept` 为 refuted / 1，`report.epipe-late` 为 confirmed / 0；
    - `report.limit-after-done` 为 confirmed / 0，三平台都是 2 次命中（回调确实在持有者结束之前裁定）；
    - 本轮的 4 项回归与此前的各项 F1 故障都是 invalid / 1。正确拒绝无效证据是门禁回归的通过，不计为机制的 confirmed。
  - **直接检查**都通过：`TestToolState`（5 组）与 `TestF1Classify`（23 行）在三平台通过；另有 `TestCensusUnreadable`（macOS）、`TestStopAPIErrors`（Windows）、`TestStopRead`、`TestLingeringFailedStop`、`TestPhaseClock`、`TestGateJudge`、`TestCandidates`。
  - **没有直接回退**：已关闭的冲突检查、PID 关联、B1、N6 与归属保护：
    - `forge-owner` 拒绝伪造的 PID，零动作、零信号；
    - `forge-owner-mixed` 结束本轮自有的遗留，拒绝伪造的 PID；
    - 补救失败、结束未确认、普查错误、状态未知的组合仍使被测自检不通过。
  - **SKIP**：只有三个只供内层运行的测试；场景 ID 的并集为 53；没有出现普查的 EINVAL。
  - **耗时**：Ubuntu 的 `go test` 为 386 s（上限 15 min），Job 6.7 min（上限 20 min）；预算没有改。
- **`9e3b7ca` 的诊断** run 40110180232（success）保留为历史观测：其中 `report.limit-after-done` 在 Ubuntu 与 Windows 上的通过是空的（见上文“夹具更正”）。
- **作者本机**（`f728e11` 的树）：
  - Windows 11：场景 29/29 confirmed；门禁回归 78/78；组合 13/13；直接检查都通过；
  - WSL2：37 confirmed、1 unsupported（没有 pwsh）；门禁回归 82/82；组合 13/13；`TestStartPoints` SKIP（二进制旁没有源码）；
  - 两边的 `report.limit-after-done` 都是 2 次命中。
- **定向撤回**（作者本机 Windows 与 WSL；每次只撤回一处，都编译通过）：

  | 变异 | 撤回的内容 | 检出（两边相同） |
  |---|---|---|
  | M66 | 报告层不保存失败 | `report.ok-fails-epipe-reported`（confirmed）、`report.epipe-fails-then-ok`（refuted）；`TestToolState` 的“被忽略的失败之后的成功” |
  | M67 | 期限进入不使工具失败 | `report.limit-race-epipe`（confirmed）、`report.limit-race-ok`（refuted）；`TestToolState` 的“期限先进入” |
  | M69 | 之后的成功清除先前的失败 | `report.ok-fails-epipe-reported`、`report.epipe-fails-then-ok`；`TestToolState` 的同一行 |
  | M70 | 完成发布期间不持有锁 | 只有 `TestToolState` 的“发布进行中回调进入”（这一顺序没有真实 helper 的回归） |
  | M68 | 回调在完成发布之后仍赢得裁定 | `TestToolState` 的“健康”与“发布进行中回调进入”；真实链路仍由 `report` 的拒绝保护 |
  | M71 | `report` 在完成发布之后仍写入 | `TestToolState` 的“健康”；真实链路仍由回调的裁定保护 |
  | M72 | M68 与 M71 同时撤回 | `report.limit-after-done` 得到 invalid（回调在完成之后写出 `limit`）；`TestToolState` 的两行 |

  - M66、M67、M69、M70 在 `9e3b7ca` 上运行；M68、M71、M72 在 `f728e11` 上运行。两者之间只有 `limit-after-done` 的夹具差异，`tool_test.go` 与 `helper_test.go` 相同。M68 在 `9e3b7ca` 上也运行过，结果相同。
  - 每个变异只由对应的检查检出。健康的 `control.f1-skip-drain`（confirmed）、合法的 `control.f1-read-kept`（refuted）、`report.epipe-late`（confirmed）与 5 个顶层机制场景都保持通过。
  - M71 的第一种写法编译失败（变量未使用），没有运行，不计为检出；改为能编译的写法之后单独运行。

**对 T1 / T6 证据的影响**：

- F1 的 confirmed 与 refuted 现在都需要一个只在 helper 所有入口都没有失败、期限没有进入时才发布的完成。
- `9e3b7ca` 之前的各次运行中，期限回调与完成的竞争、被忽略的报告失败都没有注入；场景的结论作为观测保留。
- 完成发布之后回调不报告，由两层保护共同保证：回调在锁下的裁定，与 `report` 在完成发布之后拒绝写入。真实链路的 `report.limit-after-done` 只在两层都撤回时才会失败（M72）；只撤回其中一层（M68、M71）时，由 `TestToolState` 检出。

**仍未验证的边界**：

- **信任边界**：持有者仍是受信任的 helper。工具状态只覆盖它经过 `report`、`failed`、`limitEntered`、`complete` 的入口；绕过这些入口直接写文件的代码不在模型中（本原型的 helper 没有）。
- **竞争的构造**：期限竞争由提前的真实计时器加闸门构造，不是默认 60 s 的自然长跑。发布期间回调进入只由 `TestToolState` 在进程内验证，没有真实 helper 的门禁回归。（Review 087 N-087-1：而且那一组没有确证回调已进入；收紧登记为后续事项，§13。）
- **改名的范围**：完成标记的改名只在场景自己的临时目录内、同一目录中使用，内容严格校验；不是对任意文件系统的原子性保证（N-RENAME）。
- macOS 的旧未知项保持，不作为永久合并门槛；不是硬实时保证；生产路径没有接入。

**状态**：B-N4-R（B-N4-R/T、B-N4-R/I）**已修复，待独立复审**；不宣布关闭或 GO。（其后：Review 087 技术 GO，B-N4-R/T、B-N4-R/I 关闭，见 §2.19。）

### 2.19 Review 087：技术 GO 与收尾

**审查结论**（[Review 087](reviews/codex-review-087.md)，对 `245b51b`）：

- **技术 GO**：B-N4-R/T 与 B-N4-R/I 分别关闭。
- **N4 系列只在已审查的受控协议范围内收口**：受信任的 helper、固定的入口、受控的文件布局。完成标记不是不可信写者的认证，也不是断电持久性或任意宿主协议的证明。
- **安全的依据是实际的调用拓扑，不是“类型里有 mutex”**：
  - 主 goroutine 同步完成报告之后，才尝试完成；
  - 唯一并行的真实期限回调，要先在锁下赢得裁定，才会报告；
  - 完成已发布时，回调输掉裁定、不报告；回调在发布期间到达时，等待终态。

  以后若改变拓扑，例如新增并行的 `report` 或 `complete` 调用者，须重新核查这一前提。
- **“回调进入”须分成三个事件**：回调函数开始执行 → 取得锁并作 limit 裁定 → 实际开始报告。有效性边界由第二个事件线性化；不能把 goroutine 已创建或函数已开始，写成 limit 已经设置。
- **没有新的 BLOCKER。**
- **NON-BLOCKER N-087-1**：`TestToolState` 的“发布进行中回调进入”一组，没有确证回调已经进入。发布入口有显式通知，回调只有 goroutine 启动与 100 ms 等待。审查方把该 goroutine 改为等到发布释放之后才调用 `limitEntered`，这一组仍然 PASS。建议用入口握手收紧，或把证据说明限定为它实际证明的范围。
- **NON-BLOCKER N-087-2**：不能把“没有协议循环”写成无条件的“没有无界等待”。`complete` 持锁执行文件的 Open、Write、Close、Rename，源码没有为这些系统调用提供硬截止。
- **N-RENAME 可以关闭**（措辞事项）。
- **没有新增产品决定，也没有新增接受风险。** 方案 B 的局部范围不变。旧 macOS PID 2088 / 1989 的归属、EINVAL 的根因、当时的完整清理继续未知。
- **审查时的 CI**：最终常规 CI 的合并条件已经满足（见下表）。这不能改写为首轮成功，也不表示存储层的维护问题已经解决。

**两项非阻塞建议的处理**：

- **N-087-2（措辞，本轮更正）**：§2.18 统一完成协议表中的“没有无界等待、死锁或无人负责的任务”，更正为：
  - 没有引入协议级的循环等待，锁内不调用其他入口；
  - 父端的期限不变；
  - 文件 I/O 与系统调度不提供硬实时保证。

  处理记录 §102 中同样的措辞，更正记入 §103。§2.18 原文保留，在原处加更正说明。
- **N-087-1（涉及测试代码，本轮只登记，不实现）**：§2.18 对 `TestToolState` 第 5 组的描述收窄为：它检查锁与终态的规则，没有确证回调已在发布期间进入。“发布期间回调进入”的证据来自以下几项，相互支持：
  - 源码中的互斥关系；
  - 两项真实的期限先赢回归（`report.limit-race-*`）；
  - 完成之后的真实回调（`report.limit-after-done`）；
  - 审查方的显式入口对照。

  用入口握手收紧这一组，登记为后续事项（§13），不写成已经实现。
- **N-RENAME**：经 Review 087 关闭。

**最终常规 CI 的历史**（[run 40116851180](https://github.com/XKHoshizora/specmetry/actions/runs/40116851180)，对 `245b51b`，不改写）：

| 作业 | attempt 1 | attempt 2 |
|---|---|---|
| Go tests (windows-latest) | 120939788019：**failure** | 120946480814：**实际重跑**，success |
| Go tests (macos-latest) | 120939788110：success | 120946481556：沿用 attempt 1 的结果 |
| Go tests (ubuntu-latest) | 120939788118：success | 120946505783：沿用 |
| Go lint | 120939787867：success | 120946518491：沿用 |
| Web | 120939788121：success | 120946482382：沿用 |
| Build with dashboard（Windows / macOS / Ubuntu） | 120939862188 / 120939862209 / 120939862292：success | 120946482338 / 120946507581 / 120946481904：沿用 |

- **attempt 1**：正式 completed/failure、7/8。唯一的失败是 Windows 的 `TestSeqIsUniqueAcrossConnections`（`store_test.go:546: database is locked (5) (SQLITE_BUSY)`，11.97 s）；同一作业的 `TestSeqIsUniqueAcrossProcesses` 通过。
- **attempt 2**：按负责人的一次授权执行 `gh run rerun 40116851180 --failed`，正式 completed/success。
  - 只有 Windows 的 Go 测试实际重新执行（2026-10-24 18:05:16 至 18:18:19 UTC，新的 runner）；其余七项沿用 attempt 1 的结果，开始与结束时间、runner 都与 attempt 1 相同。
  - 重跑检出同一个合并快照 `c0ec2e4`。Windows 全部测试 2999 PASS、10 SKIP、0 FAIL。
- **不能写成**：首轮 8/8，或三平台全部重跑。
- 常规 CI 不运行原型；原型诊断仍是 run 40110917799（`f728e11`），HEAD 之前的文档提交不改变它的适用性。

**M-15 的新观测**：attempt 1 的失败与维护事项 M-15 已登记的签名一致（同一测试、同一行，review-resolution §77）。

- 根因与是否同根因都没有验证；
- 重跑成功不关闭这一事项，它也不是已接受的风险；
- 已补入维护登记（`m1-exit-evidence.md` 的 M-15）与处理记录 §103。

**收尾**：

- 本节与 §103 的收尾提交只改文档。源码、原型、测试、依赖、Schema、迁移与 workflow，与受审对象 `245b51b` 完全相同。
- 合并与合并之后的核验，结果写在 PR #34 的描述中，不为登记它们而向 master 追加提交。

**状态**：

- 技术审查通过，待合并核验。
- PR #34 合并只表示原型修复入库：14 项产品选择未批准，规范未冻结，生产 WP2.5 未授权，WP2.5 / M2 未验收；M-14、M-15、M-16 未解决；旧 macOS 的未知项保留。

## 3. 运行与环境

### 3.1 运行

| 运行 | 对象 | 触发 | 结果 |
|---|---|---|---|
| [run 40110917799](https://github.com/XKHoshizora/specmetry/actions/runs/40110917799) | `f728e11498a8b4bc5b71ec3cbff1618c795f6969`（Review 086 的修复与夹具更正，§2.18） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：82/82、73/73、78/78；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过（含三平台的 `TestToolState`、`TestF1Classify`）；`report.limit-after-done` 三平台 2 次命中</li></ul> |
| 本机 Windows 11 / WSL2 | `f728e11` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 78/78、82/82；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M68、M71、M72 |
| [run 40110180232](https://github.com/XKHoshizora/specmetry/actions/runs/40110180232) | `9e3b7ca953357273ef4638d7200b5b4df2f4b88d`（Review 086 的修复，夹具更正之前） | push，attempt 1 | completed/success；门禁回归 82/82、73/73、78/78；历史观测：`report.limit-after-done` 在 Ubuntu 与 Windows 上只有 1 次命中，通过是空的 |
| 本机 Windows 11 / WSL2 | `9e3b7ca` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 78/78、82/82；组合 13/13；变异 M66 ~ M70（WSL 中 `limit-after-done` 也出现 1 次命中） |
| [run 40109659805](https://github.com/XKHoshizora/specmetry/actions/runs/40109659805) | `750e1390d6cc8cc211c144469d582cad239aca77`（Review 086 的回归先行） | push，attempt 1 | **completed/failure**（预期）：设计中的 4 项回归在三平台都失败（门禁回归 77/81、68/72、73/77）；没有出现普查的 EINVAL；场景 38、27、29 confirmed；组合 13/13 |
| 本机 Windows 11 | `750e139`（回归先行） | 作者手动 | 2 项 confirmed / 0、2 项 refuted / 1，与 run 40109659805 中设计内的失败相同；`report.ok-fails-epipe-reported` 第一次的夹具漏了保留读端（命中 0），不计 |
| [run 40103135762](https://github.com/XKHoshizora/specmetry/actions/runs/40103135762) | `fb3f56d8caebc1bc5d464f1db4cbec29a8faf24b`（Review 085 的修复，§2.17） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：76/76、67/67、72/72；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过（含三平台的 `TestF1Classify`）</li></ul> |
| 本机 Windows 11 / WSL2 | `fb3f56d` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 72/72、76/76；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M63 ~ M65 各自只被对应的检查检出 |
| [run 40102730930](https://github.com/XKHoshizora/specmetry/actions/runs/40102730930) | `acfee1fa9d2a4665b2f14858aa526b833dc63130`（Review 085 的回归先行） | push，attempt 1 | **completed/failure**（预期）：设计中的 6 项回归在三平台都失败（门禁回归 68/74、59/65、64/70）；没有出现普查的 EINVAL；场景 38、27、29 confirmed；组合 13/13 |
| 本机 Windows 11 | `acfee1f`（回归先行） | 作者手动 | `epipe` 的 3 项 confirmed / 0、`ok` 的 3 项 refuted / 1，与 run 40102730930 中设计内的失败相同 |
| [run 40096356396](https://github.com/XKHoshizora/specmetry/actions/runs/40096356396) | `1e82fbdc21edf12255c7e72a2e981e5c0a6a648d`（Review 084 的修复，§2.16） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：67/67、58/58、63/63；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过（含三平台的 `TestF1Classify`）</li></ul> |
| 本机 Windows 11 / WSL2 | `1e82fbd` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 63/63、67/67；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M60 ~ M62 各自只被对应的检查检出 |
| [run 40095937741](https://github.com/XKHoshizora/specmetry/actions/runs/40095937741) | `7307d5d20835d0947caf21e3216f1130e75b359f`（Review 084 的回归先行） | push，attempt 1 | **completed/failure**（预期）：设计中的 5 项回归在三平台都失败（门禁回归 60/65、51/56、56/61）；没有出现普查的 EINVAL；场景 38、27、29 confirmed；组合 13/13 |
| 本机 Windows 11 | `7307d5d`（回归先行） | 作者手动 | 4 项 confirmed / 0、`report.ok-foreign` refuted / 1，与 run 40095937741 中设计内的失败相同 |
| [run 40084692941](https://github.com/XKHoshizora/specmetry/actions/runs/40084692941) | `56234c364eaffc109f1ddf1cd0bd837420afcbf7`（Review 083 的修复，§2.15） | push，attempt 1 | **completed/success**。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：55/55、46/46、51/51；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过（含 macOS 的 `TestCensusUnreadable`）；</li><li>不替代旧运行中 macOS 的未知结果（§2.15）</li></ul> |
| 本机 Windows 11 / WSL2 | `56234c3` 的树 | 作者手动 | 场景 29/29、37 + 1 unsupported；门禁回归 51/51、55/55；组合 13/13；直接检查都通过（WSL 的 `TestStartPoints` SKIP）；变异 M51 ~ M54、M56、M59 各自只被对应的检查检出 |
| [run 40084348430](https://github.com/XKHoshizora/specmetry/actions/runs/40084348430) | `61a43db7f86ff6ceb4e8b1259cba07d7e63746cd`（Review 083 的回归先行） | push，attempt 1 | **completed/failure**（预期）：设计中的回归在三平台都失败（门禁回归 52/54、43/45、48/50；macOS 另有 `TestCensusUnreadable` 两行）；没有计划外的 EINVAL；场景 38、27、29 confirmed；组合 13/13 |
| 本机 Windows 11 | `61a43db`（回归先行） | 作者手动 | 两项门禁回归得到 refuted，与 run 40084348430 中设计内的失败相同 |
| [run 40074633075](https://github.com/XKHoshizora/specmetry/actions/runs/40074633075) | `2413634ba21838f7275ee4ddd523779f1409c9b6`（Review 082 的修复，§2.14） | push，attempt 1 | **completed/failure**：Ubuntu、Windows success；macOS failure（旧门禁回归 `report.truncated` 的保险清理遇到同一用户其他进程的 EINVAL，§2.14，没有重跑；“其他进程”的措辞经 Review 083 更正：该进程的归属与清理结果未知，§2.15）。<ul><li>场景：Ubuntu 38/38、macOS 27/27、Windows 29/29 confirmed；</li><li>门禁回归：52/52、42/43、48/48；</li><li>组合：13/13、13/13、13/13；</li><li>直接检查都通过</li></ul> |
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
- **Review 083 修复之后的运行**（run 40084692941）：每个 Job 检出 `56234c3` 本身；没有重跑；SKIP 只有三个只供内层运行的测试；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53；§3.3 的 CI 各格相同。这次没有出现普查的 EINVAL；它不替代 run 40072151282、40074633075 中 macOS 的未知结果（§2.15）。
- **Review 084 修复之后的运行**（run 40096356396）：每个 Job 检出 `1e82fbd` 本身；没有重跑；SKIP 只有三个只供内层运行的测试；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53；§3.3 的 CI 各格相同；没有出现普查的 EINVAL。
- **Review 085 修复之后的运行**（run 40103135762）：每个 Job 检出 `fb3f56d` 本身；没有重跑；SKIP 只有三个只供内层运行的测试；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53；§3.3 的 CI 各格相同；没有出现普查的 EINVAL。
- **Review 086 修复之后的运行**（run 40110917799）：每个 Job 检出 `f728e11` 本身；没有重跑；SKIP 只有三个只供内层运行的测试；Ubuntu 带 `-race`，没有数据竞争的报告；场景 ID 集合不变，并集 53；§3.3 的 CI 各格相同；没有出现普查的 EINVAL。
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
- **Review 083 之后**（§2.15）：`419e4f3` 之前的各次运行中，F1 场景的 refuted 不要求观察已经结束，持有者回答之后的写入与报告失败不进入结论；那些运行中没有注入这样的故障，场景的结论作为观测保留。`control.f1-read-kept` 的 refuted 现在另需每个保留读端的 EOF。run 40072151282、40074633075 中 macOS 的失败保持原样，归属、原因与清理结果仍未知，不由之后的成功运行消除。当前证据是 run 40084692941（`56234c3`）。
- **Review 084 之后**（§2.16）：`173d2eb` 之前的各次运行中，F1 场景找到一条 `epipe` 记录就判为预测成立，不看旁边是否有回答、记录是否属于持有者；那些运行中没有注入这样的组合，场景的结论作为观测保留。run 40084692941 的成功只覆盖报告失败的组合。当前证据是 run 40096356396（`1e82fbd`）。
- **Review 085 之后**（§2.17）：`fb3f56d` 之前的各次运行中，F1 的结论不要求持有者的完成提交，结束持有者之后干净的报告被当作工具完成；那些运行中没有注入记录之后报告失败或被截断的组合，场景的结论作为观测保留。run 40096356396 的成功只覆盖已经写出的记录之间的裁定。当前证据是 run 40103135762（`fb3f56d`）。
- **Review 086 之后**（§2.18）：`9e3b7ca` 之前的各次运行中，完成只看主路径自己的返回值，不与期限回调同步，也不知道被忽略的报告失败；那些运行中没有注入这些组合，场景的结论作为观测保留。run 40103135762 的成功只覆盖主路径上返回值受检查的串行操作。当前证据是 run 40110917799（`f728e11`）。

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
  - **普查的前提**：本轮的进程从内层继承唯一的标记，并且可读其环境（同一用户、可 dump）。启动不早于内层、属于同一用户、却读不到环境的其他进程，会使普查失败（保守，这项检查不通过）。Review 082 这一轮在 macOS 上实际出现两次（EINVAL：回归先行 run 40072151282 的 `report.epipe-unwritable-alive`、最终诊断 run 40074633075 的 `report.truncated`；出错的进程都不在内层的创建记录中），都没有重跑。**更正**（Review 083 N-MAC，§2.15）：括号中“都不在内层的创建记录中”不能由日志独立确认（日志没有打印完整的创建记录），即使成立也不能排除未登记的后代；两个进程的归属、EINVAL 的原因与保险清理之后的完整性都未知，两次的阶段也不同（被测普查 / 保险清理）。普查失败的信息现在包含该进程的身份与状态（§2.15），归属规则不变。其中短命令名的比较只是前缀匹配的启发式，不是程序身份（Review 084 N1，§2.16）。N1 经 Review 085 关闭。是否单独处理这一前提（例如在 macOS 上区分正在退出的进程），交负责人决定。一个故意清除自己环境的进程，普查看不到；本原型的辅助进程没有这样做。
  - **测试条件的变化**：Windows 的内层运行在外层自己的 Job 中，内层每次检查的 Job 嵌套在它下面；门禁之下，辅助进程先等待自己的创建记录再做事。正常的诊断运行（不在门禁之下）不受影响。
  - **补救**只针对普查找到的本轮进程（Windows 按 Job 成员；Unix 按标记与身份），从不使用无效见证中的 PID；补救之后的状态另行记录。区间外脱组的后代在报告读不懂时无法登记，只能申报为缺额（`group.escape-boundary` 的损坏报告没有回归）。
- **Review 076 N4**（NON-BLOCKER，Review 080 保持；第一次修复见 §2.12；Review 081 认定残余 N4-R，Review 082 认定其组合故障，Review 083 认定回答之后的残余 N4-R-S，Review 084 认定回答与失败记录并存时的绿色假通过 B-N4（经 Review 085 关闭），Review 085 认定缺少工具完成证明的 B-N4-R，Review 086 认定它的两项残余（期限回调、被忽略的报告失败），经 Review 087 关闭，N4 系列只在受控 helper 协议范围内收口，§2.13 ~ §2.19）：持有者写 `epipe` 记录失败时，F1 场景把“没有观察到预测的事件”记为 refuted，而不是把“观察工具失败”记为 invalid；没有形成假通过。建议把持有者的报告失败作为独立的工具故障状态传给场景，并增加分类回归。修复：没有记录时先读持有者的状态，已结束而没有记录为 invalid（§2.12 的表）。这一判定已由 §2.13 取代：没有记录时，只有迟到字节的证据才是 refuted，其余为 invalid。其后又由 §2.14 取代：refuted 需要持有者的确认与每一路的迟到字节。其后又由 §2.15 取代：询问即观察的终点，refuted 需要持有者在关闭输出之后的最终回答、每个保留读端的 EOF 与每一路的迟到字节。其后又由 §2.16 补充：证据完整收集之后整体分类，`ok` 与 `epipe` 互斥，记录须是持有者的，收集中的失败使任一记录旁的结论为 invalid。其后又由 §2.17 补充：confirmed 与 refuted 都需要持有者的完成提交，结束持有者只是清理。其后又由 §2.18 补充：完成只经过 helper 的统一工具状态发布，报告失败在报告层保存，期限回调与完成在同一把锁下裁定。
- **Review 087 N-087-1**（NON-BLOCKER，登记为后续事项，没有实现）：`TestToolState` 的“发布进行中回调进入”一组只有 goroutine 启动与 100 ms 等待，没有确证回调已进入；审查方把回调改为发布释放之后才调用，这一组仍然通过。建议用入口握手收紧，或保持 §2.18 中已收窄的说明。发布期间回调进入的现有证据见 §2.19。
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
- **停止读取的生命周期**（Review 081 B1，§2.13；此前登记为“Windows 上停止读取的时序前提”）：停止已纳入阶段 J，阶段 J 的结束、封闭与 O3 在 `T_join` 之内到达；停止本身在 Unix 上于读取离开系统调用时、在 Windows 上于读取进入系统调用时完成。读取始终不进入系统调用时停止挂起，读端、复制任务与停止任务一直被持有，没有上界；挂起期间 Windows 每毫秒一次取消。生产中怎样处理（C-4、D20）属于冻结之前的门槛。停止的结果随任务移交并在结论之前消费，Windows 的 API 结果按契约区分（Review 082，§2.14）。Review 083 认定 B1-Ra、B1-Rb 可以关闭（§2.15）；挂起资源的上界与后续宿主策略仍是冻结之前的门槛。
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

## 14. P-1、P-2、P-4 的隔离原型与 P-3 的整理（2026-10-27 授权）

依据 [review-resolution.md §114](review-resolution.md) 记录的授权：PR #37 合并核验完成之后，从已核验的 master `2c5cf23` 另开 PR，只在本原型模块中实施草案 §19.12 的 P-1、P-2、P-4；P-3 只整理。本节的 14.1 ~ 14.5 在写任何实验代码**之前**提交，固定场景、次数、预期、期限与清理预算；结果写在 14.7 之后，不回改 14.1 ~ 14.5。

### 14.1 边界

- 只新增 `prototype/wp25/` 中的 `_test.go` 文件（夹具、模型与测试），以及诊断 workflow 中这个分支的推送触发（`proto/wp2.5-p1-p2-p4`）。不改 workflow 的权限、预算、步骤与根 `ci.yml`；不改生产代码、迁移、推导规则与 schema；不修 M-14 ~ M-17、N-087-1。
- 原型中的“Runner”是测试代码里的最小模型，只实现被测的分支；它不是生产实现，也不接入生产存储或迁移。Q-2 的“记录”只是模型给出的记录判定，生产事务的保证留待生产授权之后验证。
- seccomp 只装在 P-2 的隔离实验子进程（测试二进制再次启动的辅助进程）上，不改 runner 的系统配置。
- 只操作本轮创建、归属与身份已确认的对象。不向归属未确认的 PID 发信号。

### 14.2 共同规则

- **预期在实施之前固定**（本节）；每次运行只执行一次，失败、invalid、unsupported 都如实保留，不为取得绿色而重跑（诊断 CI 也不重跑）。
- **内层与外层分别裁定**：内层是模型的一次调用（被测的机制）；外层是场景自己的代码，在内层返回之后、任何补救之前，用自己的句柄与读数（不用内层的对象）独立确认目标分支确实命中，然后先保存内层的真实状态（写进结果行），再补救。
  - 内层的预期与观察不符 → refuted；
  - 目标分支没有命中、观察失败（任何一次，之后的成功不恢复）、外层自己的失败、补救失败、清理没有确认 → invalid；
  - 补救成功只说明收尾，不能恢复内层的结论；
  - 环境不具备（例如装不上过滤器）→ unsupported，不是通过。
- **期限**：每次运行有单独的上限，由各阶段的有界等待组成；每项实验另有一个**总期限**。开始每一次运行之前，剩余的总期限小于单次上限 → 这一次及之后的不执行，记为 invalid（“未执行：总期限”），不算通过。
- **证据的形式**：每次运行一行 `WP25RESULT`（结论、预测、观察、清理）；内层的状态快照、命中记录、外层的独立确认与补救结果写在 `observed` 中，按“inner / branch / admission / remedy / final”分开。
- **goroutine 的证据**（P-1、P-4）：按栈判断，不按总数。在内层返回之后取全部 goroutine 的栈：
  - 任何一个 goroutine 处在进程等待的帧中（`os.(*Process).Wait`、`os.(*Process).wait`、`os/exec.(*Cmd).Wait`、`syscall.Wait4`、`syscall.wait4`、`os.(*Process).blockUntilWaitable`、`os.(*Process).pidfdWait`）→ 禁止的阻塞 `Wait`；
  - 本次运行开始之后新出现、仍在运行原型代码的 goroutine 逐个列出；预期为没有。
  - 不能归属的等待帧（运行开始之前就存在的）使这次运行 invalid，不当作通过。
- 用到的接缝都记录命中（地点、PID、次数）；没有命中记录的“故障”不算发生过。
- **区分三种来源**：接缝注入（真实调用没有发生，返回值由接缝给出）、改变前提的构造（测试自己用真实 API 改变了被测调用的前提）、真实 API 的返回。结果行逐项标明。

### 14.3 P-1（G3，Q-1；三平台）

**变体**（每个平台各 2 次，共 6 次）：

| 变体 | 主进程 | 接缝与构造 | 外层要独立确认的目标分支 |
|---|---|---|---|
| `alive`（目标） | `sleep`：报告自己的 PID 后一直运行 | 第一次 E-main 观察返回注入的错误（一次）；内层的全部终止请求被吞掉（返回成功，没有发出）。Windows 另有改变前提的构造：外层在内层关闭 Job 句柄之前另持被测 Job 的一个句柄，使 `KILL_ON_JOB_CLOSE` 不在内层关闭时触发 | 主进程**仍存活**：Unix 按 `/proc` / `sysctl` 读到存在、不是僵尸、启动时间与启动时记下的相同、父进程是本进程；Windows 按外层自己的进程句柄没有发出信号、并且在被测 Job 中 |
| `exited`（对照） | `exit-after 0 7`：立即以 7 退出 | 夹具先以自己的读数确认主进程已经退出，再放行内层的观察；第一次 E-main 观察返回注入的错误（一次）；终止请求是真实的 | 主进程**已退出而观察失败**：Unix 为本进程未回收的僵尸、启动时间相同；Windows 为外层的进程句柄已发出信号、退出码为 7 |
| `healthy`（反向对照） | `exit-after 100 0` | 没有接缝 | 不适用：E-main 被可靠观察，进程树确认结束 |

**内层**（模型的一次调用）：宿主准入 → 启动（Unix：独立进程组与 E-main 观察者，Linux 在排他区间之内并设置 subreaper；Windows：挂起创建、加入 Job、确认、恢复）→ 夹具的放行点 → S2 → 进入 S3 → S4 / S5，在 `T_kill`（5 s）内确认 → 分类：观察失败不可撤销，`alive` 与 `exited` 为 F1；`healthy` 为正常结束。F1 时按草案 §19.3：不调用 `Wait`；关闭 E-main 的观察者（Linux 的 pidfd、macOS 的 kqueue）；Linux 清除 subreaper、释放排他区间；Windows 关闭 Runner 的 Job 句柄，Runner 的进程句柄不关闭；把“未确认结束的进程”一项（`*exec.Cmd`、Windows 的 Runner 进程句柄）移交宿主的资源清单，并在同一个临界区中置位禁用（原因 `unconfirmed_end`，Q-1）。

**内层的预期**：

| 断言 | `alive` | `exited` | `healthy` |
|---|---|---|---|
| 分类 | F1 | F1 | 正常结束，退出 0 |
| 观察的接缝命中 | 1 次，主进程的 PID | 1 次，主进程的 PID | 0 |
| 终止的接缝命中 | ≥ 1 | 0 | 0 |
| 资源清单 | 恰好 1 项“未确认结束的进程”，持有的正是内层启动的那个 `*exec.Cmd`（指针相同）；Windows 另持 Runner 的进程句柄（句柄值相同，对它的零等待不出错） | 同左 | 空 |
| 禁止的阻塞 `Wait` | 没有 | 没有 | 不适用（正常结束之后内层回收主进程，等待已经返回） |
| 本次运行新出现的原型 goroutine | 没有 | 没有 | 没有 |
| 宿主禁用 | 是，原因 `unconfirmed_end` | 是 | 否 |
| Linux：内层返回之后的 subreaper | 0 | 0 | 0 |
| Linux / macOS：E-main 观察者的描述符 | 已关闭 | 已关闭 | 已关闭 |

**外层**（内层返回之后、补救之前）：

- 目标分支的独立确认（上表）；没有命中 → invalid（负载没有建立），不是 refuted。
- **Q-1 的准入**：再请求一次检查 → `alive`、`exited` 被拒绝，被拒绝的请求中数据库、git、进程启动的计数都为 0；`healthy` 照常准入。
- **Linux 的基线屏障**（单独记录，不是 Q-1 的依据）：绕过宿主准入，直接请求排他区间 → `alive`、`exited` 因本进程有子进程而被拒绝，区间内的函数没有被调用。macOS 与 Windows 记为“没有这样的屏障”。
- 先把内层的快照与上述外层证据写进结果，再补救。
- **补救**（上限 20 s）：Unix 在身份核对之后结束主进程（组长仍被持有时结束整组）并回收；Windows 经外层另持的 Job 句柄终止并确认 `ActiveProcesses = 0`，再关闭外层的句柄并回收。然后模拟宿主退出：关闭资源清单中 Runner 的进程句柄（Windows），记录关闭的结果。
- **补救之后再请求一次检查**：`alive`、`exited` 仍被拒绝（Q-1 不因之后的结束、资源释放而解除）——这一条属于内层的预期。
- **清理核验**：场景的全部进程确认结束；Linux 上本进程没有留下子进程。Windows 不嵌套外层 Job（嵌套会改变被测 Job 的行为，草案 §19.12 P-1 的风险项）；主进程不创建后代，外层以自己的进程句柄确认它的结束。

**期限**：每次 60 s（内层 30 s、证据 10 s、外层补救与清理 20 s）；P-1 在每个作业中的总期限 3 min。

### 14.4 P-2（G6；Linux）

每一组在一个隔离的实验子进程中运行：测试二进制以辅助模式再次启动。子进程先（按组）安装 seccomp 过滤器，再按草案 §19.6 的顺序探测，探测成立时设置 subreaper、启动并回收一个最小的检查进程、清除 subreaper；它把每一步的原生返回写进报告，然后等待 stdin 关闭。过滤器使用 `SECCOMP_FILTER_FLAG_TSYNC`（作用于子进程的全部线程），之前设置 `PR_SET_NO_NEW_PRIVS`；只作用于这个子进程及其后代。

**组与预期**（正常组 3 次，其余每组 1 次，共 9 次）：

| 组 | 过滤器 | 探测的预期 | 检查进程 | subreaper 的终值 | 子进程的退出码 |
|---|---|---|---|---|---|
| `normal` ×3 | 无 | GET 为 0；`pidfd_open` 成功；`pidfd_send_signal(self, 0)` 返回 0；`waitid(P_PIDFD, self, WEXITED \| WNOHANG)` 为 `ECHILD`；`poll` 返回 0；`/proc` 可读 | 启动、E-main 观察到 `exit 0`、回收 | 0 | 0 |
| `pidfd_open.EPERM` | `pidfd_open` → `EPERM` | 在 `pidfd_open` 拒绝，`EPERM` | 没有 | 0 | 6 |
| `pidfd_open.ENOSYS` | `pidfd_open` → `ENOSYS` | 在 `pidfd_open` 拒绝，`ENOSYS` | 没有 | 0 | 6 |
| `pidfd_send_signal.EPERM` | `pidfd_send_signal` → `EPERM` | 在 `pidfd_send_signal` 拒绝；自己的 pidfd 已关闭 | 没有 | 0 | 6 |
| `waitid.EPERM` | `waitid` → `EPERM` | 在 `waitid` 拒绝；自己的 pidfd 已关闭 | 没有 | 0 | 6 |
| `prctl.set1` | `prctl(PR_SET_CHILD_SUBREAPER, 1)` → `EPERM`；GET 允许 | 探测成立（GET 为 0）；设置失败，在 `Start` 之前拒绝 | 没有 | 0 | 6 |
| `prctl.set0` | `prctl(PR_SET_CHILD_SUBREAPER, 0)` → `EPERM`；`SET 1` 与 GET 允许 | 探测成立；设置与核对成功（GET 为 1） | 启动、观察、回收 | **1**：清除失败，如实报告“属性保留”；之后的一次探测因 GET 为 1 而拒绝 | 6 |

**命中的证据**（每一组都要）：

- 过滤器确实装上：子进程自己读 `/proc/self/status`，外层另行读 `/proc/<子进程>/status`：`NoNewPrivs: 1`、`Seccomp: 2`、`Seccomp_filters: 1`（正常组为 0 / 0 / 0）。
- 故障确实命中：被过滤的调用返回规则给出的 errno；同一个过滤器下的参数对照照常成功（`prctl.set1` 的 GET，`prctl.set0` 的 `SET 1` 与 GET）。pidfd 与 `waitid` 的组以同一作业中正常组的同一调用成功为对照。
- 外层在子进程等待 stdin 时核对：子进程没有子进程（检查进程已回收或从未创建）；子进程的描述符中没有遗留的 pidfd（`/proc/<pid>/fd` 的链接不是 `anon_inode:[pidfd]`）；检查进程、子进程自己的 pidfd 与设置过滤器的子进程按 PID 与描述符分开记录。
- `prctl.set0`：子进程报告清除失败之后 GET 仍为 1、之后的探测被拒绝；子进程随后退出时属性随进程消失，这**不**算内层清除成功，结果中明写。
- 装不上过滤器（`seccomp` 或 `PR_SET_NO_NEW_PRIVS` 失败）→ 这一组 unsupported。

**期限**：每次 30 s；P-2 的总期限 3 min。**清理**：外层关闭 stdin，子进程退出，外层回收并核对退出码；场景结束时本进程没有留下子进程。

### 14.5 P-4（G7，Q-1、Q-2；Windows）

**场景**（每个 2 次，共 18 次；“来源”一栏见 14.2）：

| # | 场景 | 来源 | 预期的知识状态 | §8.6 | 记录判定（Q-2 的模型） | 宿主禁用（Q-1） |
|---|---|---|---|---|---|---|
| 1 | `healthy`：Runner 的 `ResumeThread` 返回 1，主进程写标记后以 0 退出 | 真实 API | 按协议视为已启动；有 `t_start`，没有 `t_anom` | A10 | 照常：`pass`，退出码 0 | 否 |
| 2 | `resume.0`：夹具在 Runner 之前自己恢复主线程（夹具事件：它的 `ResumeThread` 返回 1），Runner 的 `ResumeThread` 返回 0 | 改变前提的构造 + 真实 API | 可能已运行；`t_anom`，没有 `t_start` | A12 | `error` / `context_changed`，不合格；`head_end` NULL，`tree_state_end = unknown`；`duration_ms = t_S3 − t_anom`；`exit_code` 为 Runner 可靠读到的值（与外层自己的读数相同），读不到为 NULL | 否 |
| 3 | `resume.gt1`：夹具在 Runner 之前对主线程多挂起一次（夹具事件：它的 `SuspendThread` 返回 1），Runner 的 `ResumeThread` 返回 2 | 改变前提的构造 + 真实 API | 可能已运行 | A12 | 同 2 | 否 |
| 4 | `resume.-1`：接缝把 Runner 用于恢复的线程句柄换成只有 `THREAD_QUERY_LIMITED_INFORMATION` 的句柄，真实的 `ResumeThread` 返回 `(DWORD) -1`、`ERROR_ACCESS_DENIED` | 接缝改变前提 + 真实 API 的失败 | 可能已运行（未确认恢复成功） | A12 | 同 2 | 否 |
| 5 | `pre.assign`：加入 Job 失败 | 接缝注入 | 确定未运行 | A4（`TerminateProcess`，进程句柄确认） | 不记录 | 否 |
| 6 | `pre.snapshot`：线程快照失败 | 接缝注入 | 确定未运行 | A4（`TerminateJobObject`，`ActiveProcesses = 0`） | 不记录 | 否 |
| 7 | `pre.openthread`：打开线程失败 | 接缝注入 | 确定未运行 | A4 | 不记录 | 否 |
| 8 | `pre.unconfirmed`：加入 Job 失败，且 `TerminateProcess` 被吞掉（没有发出） | 接缝注入 | 确定未运行；挂起的进程未确认终止 | A5 | 不记录；说明“一个挂起的进程可能仍然存在” | **是** |
| 9 | `anomaly.unconfirmed`：同 3 的构造（返回 2），Job 的查询失败 | 构造 + 真实 API + 接缝注入 | 可能已运行；未确认 | A6 | `error` / `context_changed`，字段同 2 | **是** |

**内层的预期**（逐场景）：

- 知识状态与 §19.7 的矩阵一致：异常路径从不报告“检查没有运行”，也不报告“已经执行”；返回 1 只报告按协议已启动；恢复之前的失败报告确定未运行。
- 时间：正常为 `t_start`；异常为 `t_anom`，`duration_ms = t_S3 − t_anom`（模型自己的时钟，非负）。
- 终止的确认只看 `ActiveProcesses = 0`（在 Job 中）或进程句柄（不在 Job 中），终止请求的成功不算确认（C-17）。
- 所有权按草案 §19.7 的分支表：
  - A4、A12：Job 句柄、Runner 的进程句柄关闭成功，线程句柄关闭，Go 的进程对象经 `Wait` 释放；
  - A5、A6：Job 句柄关闭，Runner 的进程句柄**不关闭**、移交资源清单并仍有效（外层在补救之前以零等待核对），Go 的进程对象不 `Wait`（没有 goroutine 处在进程等待的帧中）。
- 记录判定按上表；Q-1 的禁用按上表。写事务之前取消的 A12（D8，不记录）与写事务之后的取消，只在直接检查中按判定表核对（没有进程）。

**外层**：

- 夹具事件的命中：2 与 3 的夹具调用返回值；4 的句柄权限（打开时只请求 `THREAD_QUERY_LIMITED_INFORMATION`）与真实的错误码；5 ~ 9 的接缝命中记录。
- 外层自己在创建之后、Runner 恢复之前打开主进程与主线程的句柄（只读与同步权限）：
  - 用进程句柄独立确认结束与退出码（2 ~ 4 的退出码与 Runner 读到的比较）；
  - 8：补救之前进程仍存在（句柄没有发出信号）；
  - 线程的 `QueryThreadCycleTime` 只作为辅助读数原样记录，**不**作为执行或未执行的证据；当前的挂起计数不读取、不使用；调用失败时这一辅助读数记为 unsupported，场景照常裁定；不以“没有标记文件”单独证明没有运行。
- 先保存内层的状态，再补救：以外层的进程句柄 `TerminateProcess` 并等待确认，关闭外层的句柄；模拟宿主退出时关闭资源清单中的句柄，记录结果。
- 清理核验：外层以自己的进程句柄确认主进程结束（辅助进程不创建后代）；不嵌套外层 Job（同 P-1）。草案 §19.12 P-4 写的“外层 Job 的活动进程最终为 0、已创建进程数与见证一致”只适用于门禁之下的内层运行，P-4 不在门禁之下运行，以上面的方法代替，并在结果中写明。

**期限**：每次 30 s（内层 20 s、外层 10 s）；P-4 的总期限 4 min。

**直接检查**（三平台，没有进程）：Q-1 的触发表（F1、A5、A6 触发；没有进程、启动之前的拒绝、A4、A12、正常都不触发；O3、D21 保持原规则）；Q-2 的记录判定表（A12 在写事务之前取消 → 不记录；没有取消 → 字段如上，可靠取得与没有取得两种；写事务开始之后的取消 → 按实际提交结果）。

### 14.6 P-3：macOS 的支持矩阵、ABI 来源与证据缺口（只整理，不执行）

本轮不执行 Intel 原型，不缩小 D7 保留的 macOS 支持范围，也不因缺少 runner 宣布 G5 满足。

**支持矩阵**：

| 平台 | 支持声明（D7） | 动态证据 | 静态依据 | 状态 |
|---|---|---|---|---|
| macOS 26.6.2，arm64（`macos-latest` runner） | 保留 | 本原型的 T3 场景（`emain.exit-unreaped`、`emain.bounded-timeout`、`emain.waitid-wnowait`、`group.zombie-leader`、`cleanup.state-unknown` 等，§6）；本轮的 P-1 结果见 14.7 | 同下 | 部分：只有这一个版本 |
| macOS，x86_64（Intel） | 保留，不缩小 | **没有** | 见下面的 ABI 来源；不是动态证据 | 门槛没有满足：静态依据待审查接受；接受之后作为证据限制 |
| 其他 macOS 版本 | 保留 | 没有 | 同上 | 没有证据 |

**ABI 来源**（原型使用的接口；Specmetry 的依赖只有 `golang.org/x/sys` v0.49.0）：

| 接口 | 原型的读取方式 | 来源 | arm64 与 x86_64 |
|---|---|---|---|
| `kern.proc.pid`、`kern.proc.pgrp`（`KinfoProc`，长度 `0x288`） | `unix.SysctlRaw`、`unix.SysctlKinfoProcSlice`，按 `unsafe.Sizeof(unix.KinfoProc{})` 核对长度 | x/sys 的 `ztypes_darwin_*.go`（由 `cgo -godefs` 从 SDK 头文件生成） | 作者核对：`ztypes_darwin_amd64.go` 与 `ztypes_darwin_arm64.go` 在替换架构名之后逐行相同（0 行差异） |
| kqueue（`Kevent_t`、`EVFILT_PROC`、`NOTE_EXIT`、`NOTE_EXITSTATUS`、`EV_ADD`、`EV_ONESHOT`） | `unix.Kevent` | x/sys 的 `ztypes_darwin_*.go`、`zerrors_darwin_*.go` | 两个架构的 `zerrors_darwin_*.go` 在替换架构名之后逐行相同 |
| `waitid`（`SYS_WAITID`、`WEXITED`、`WNOWAIT`、`P_PID = 1`） | 原始系统调用 | x/sys 的 `zsysnum_darwin_*.go`、`zerrors_darwin_*.go` | `SYS_WAITID` 都是 173；常量相同 |
| `siginfo_t` 的前六个 32 位整数（`si_signo` 0、`si_errno` 4、`si_code` 8、`si_pid` 12、`si_uid` 16、`si_status` 20） | 128 字节的原始缓冲区，只读前 24 字节 | x/sys 没有 darwin 的 `Siginfo`；依据是 XNU 的 `bsd/sys/signal.h`（`__siginfo`），Review 091 查阅的是上游 `main` 分支 | 只有静态推理：开头的 32 位整数在两种 LP64 ABI 下偏移相同；具体的 SDK 或源码版本**没有登记** |

**证据缺口**：

1. x86_64 上没有动态运行：草案 §19.12 P-3 的场景（退出而不回收、超时、组长保留、枚举未知、`si_pid` 不符的正反对照、`exit 7` 与 `signal 9` 的解码）都没有执行。本轮没有核实可用的 Intel runner 及其 macOS 版本。
2. `siginfo_t` 的依据没有登记具体的 SDK 或 XNU 版本；x/sys 生成文件所用的 SDK 版本也没有登记。
3. 运行时核对只覆盖 `si_signo`、`si_pid` 与 `si_code` 的集合，不证明 `si_status` 的偏移或整个结构的布局（草案 §19.5）。
4. `ENOMEM` 的重试、`EINTR`、核对不一致的注入属于授权之后的生产测试，没有运行。
5. x/sys 的两个架构逐行相同只说明 Go 侧的定义一致，不证明对应 SDK 的头文件或内核的行为相同。

**G5 的状态**：arm64 部分有动态证据；Intel 的静态依据（上表）待审查接受，`siginfo_t` 的版本登记仍缺；**G5 没有满足**。是否把支持范围限定为有证据的版本，属于改变已批准的支持范围，本轮不提出。

**第 8 版之后**（规范草案 §19.5、§19.12，待独立审查；本节保留原写法）：

- 缺口 2 的版本登记：固定为 `xnu-12377.121.6`（macOS 26.5）与 `xnu-11417.140.69`（macOS 15.6）；它们不是 runner 实际运行的内核。x/sys 生成文件所用的 SDK 仍无法核实。
- 缺口 3：第 8 版写定状态的解码规则（W1 ~ W5、K1 ~ K2、C1；第二稿把 W3 拆为 W3a 与 W3b，W3b 按 Q-3 (a) 取低 8 位）。本原型的 waitid 还没有实现它们（只解码 `si_code` 与 `si_status`，按有符号数读取），所以上表的 arm64 证据不覆盖第 8 版的核对。
- 上表 `siginfo_t` 一行的“只有静态推理”：第二稿按固定提交的源码推演了退出值的位段（草案 §19.5）；waitid 的退出值不限于 0 ~ 255。
- 缺口 1：P-3 的完整计划写在草案 §19.12，只有条件授权，没有执行。

### 14.7 结果（固定代码点 `d45a319`）

> Review 095 B1 之后，本节与 §14.8 是修复之前的**历史观测**，保留原文；有效门禁之下的替代证据见 §14.12、§14.14。

**固定的代码点**：`d45a319dd3f61b5e98ff7bb95249c082fb2541dd`（`test: add the P-1, P-2 and P-4 isolated prototypes`）。14.1 ~ 14.6 在它之前的提交 `dcdb2b2` 中固定；之后只有文档的提交，不触发诊断，也不改变原型代码。

**诊断 CI**（`.github/workflows/wp25-prototype.yml`，与常规 CI 分开）：run 40439516254，push，attempt 1，completed/success，没有重跑、没有取消。

| 作业 | Job ID | 结果 | 实际检出 | 环境 | 本作业的场景 | 门禁回归 | 组合 |
|---|---|---|---|---|---|---|---|
| macOS | 121976202174 | success | `d45a319` | macOS 26.6.2（Darwin 25.6.0），arm64，Go 1.27.3 | 33/33 confirmed | 73/73 | 13/13 |
| Ubuntu（`-race`） | 121976202347 | success | `d45a319` | Linux 6.17.0-1022-azure，amd64，Go 1.27.3 | 53/53 confirmed | 82/82 | 13/13 |
| Windows | 121976202551 | success | `d45a319` | Windows 10.0.26100，amd64，Go 1.27.3 | 53/53 confirmed | 78/78 | 13/13 |

三份日志中都没有 `--- FAIL`，也没有 refuted、invalid 或 unsupported。场景数中包含既有的原型场景（macOS 27、Ubuntu 38、Windows 29）；它们没有回退。

**P-1 的原始结论**（每行 2 次，都为 confirmed；时长为单次的毫秒数）：

| 变体 | macOS | Ubuntu | Windows | 外层独立确认的分支 |
|---|---|---|---|---|
| `alive` | 5020、5020 | 5039、5036 | 5029、5028 | 仍存活：Unix 为同一启动时间、父进程是本进程、不是僵尸；Windows 为外层句柄没有发出信号且在被测 Job 中（外层另持 Job 句柄的构造） |
| `exited` | 15、15 | 34、35 | 19、20 | 已退出而观察失败：Unix 为本进程未回收的僵尸；Windows 为外层句柄已发出信号、退出码 7 |
| `healthy` | 109、109 | 1131、1131 | 118、120 | 不适用：E-main 被可靠观察（`exit 0`），内层回收 |

- 每一次 F1 运行：分类为 F1；观察的接缝命中 1 次；资源清单恰好 1 项，持有的正是内层启动的那个 `*exec.Cmd`，Windows 另持 Runner 的进程句柄（零等待不出错）；没有 goroutine 处在进程等待的帧中，没有新出现的原型 goroutine；宿主禁用，原因 `unconfirmed_end`；下一次检查被拒绝（数据库、git、进程启动都为 0）；补救与清单释放之后仍被拒绝。`alive` 中终止请求被吞掉的次数为数百次（每 5 ms 一次，持续 `T_kill`）。
- Linux：内层返回之后 subreaper 为 0；E-main 的 pidfd 已关闭；绕过宿主准入直接请求排他区间 → 因本进程有子进程被拒绝，区间内的函数没有被调用（F1 两种变体）；`healthy` 不被拒绝。macOS：kqueue 描述符已关闭（`F_GETFD` 为 `EBADF`）；没有基线屏障。Windows：Runner 的 Job 句柄关闭成功。
- `healthy`：宿主不禁用，下一次检查照常准入；资源清单为空。
- 外层的补救与清理：每一次都 “verified: 1 processes gone”；Ubuntu 上本进程没有留下子进程。

**P-2 的原始结论**（Ubuntu，都为 confirmed）：

| 组 | 时长（ms） | 过滤器（内 / 外的读数） | 被拒绝的调用与 errno | 之前的调用 | 检查进程 | subreaper 终值 | 退出码 |
|---|---|---|---|---|---|---|---|
| `normal` ×3 | 2040、2040、2037 | 0 / 0 / 0（两侧相同） | —；`waitid(P_PIDFD, self)` 为 `ECHILD` | 全部成立 | 启动、`exit 0`、回收 | 0 | 0 |
| `pidfd_open.EPERM` | 25 | 1 / 2 / 1 | `pidfd_open` `EPERM` | GET 0 | 没有 | 0 | 6 |
| `pidfd_open.ENOSYS` | 25 | 1 / 2 / 1 | `pidfd_open` `ENOSYS` | GET 0 | 没有 | 0 | 6 |
| `pidfd_send_signal.EPERM` | 24 | 1 / 2 / 1 | `pidfd_send_signal` `EPERM`；自己的 pidfd 已关闭 | GET、`pidfd_open` | 没有 | 0 | 6 |
| `waitid.EPERM` | 24 | 1 / 2 / 1 | `waitid(P_PIDFD)` `EPERM`；自己的 pidfd 已关闭 | GET、`pidfd_open`、`pidfd_send_signal` | 没有 | 0 | 6 |
| `prctl.set1` | 25 | 1 / 2 / 1 | `PR_SET_CHILD_SUBREAPER 1` `EPERM`（GET 照常为 0） | 探测全部成立 | 没有 | 0 | 6 |
| `prctl.set0` | 1032 | 1 / 2 / 1 | `PR_SET_CHILD_SUBREAPER 0` `EPERM`（`SET 1` 与 GET 照常） | 探测、设置、核对都成立 | 启动、`exit 0`、回收 | **1**，之后的探测因 GET 为 1 而拒绝 | 6 |

- 过滤器的读数为 `NoNewPrivs / Seccomp / Seccomp_filters`；实验子进程自己读与外层读 `/proc/<pid>/status` 都相同。
- 每一组在子进程等待释放时：子进程没有子进程，描述符中没有 pidfd。
- `prctl.set0`：清除失败之后属性保留为 1，直到实验子进程退出；那次退出**不是**内层的清除成功。
- 作者本机（WSL2，内核 6.18）上同样 9/9 confirmed；作者本机没有 `-race`（交叉编译需要 cgo）。

**P-4 的原始结论**（Windows，每个场景 2 次，都为 confirmed）：

| 场景 | 来源 | Runner 的 `ResumeThread` | 知识状态 | §8.6 | 夹具事件 / 接缝命中 | Runner 读到的退出状态（= 外层） | 记录判定 | 宿主 |
|---|---|---|---|---|---|---|---|---|
| `healthy` | 真实 API | 1 | 按协议视为已启动 | A10 | — | `exit 0` | `pass`，退出码 0 | 不禁用 |
| `resume.0` | 构造 + 真实 API | 0 | 可能已运行 | A12 | 夹具自己的 `ResumeThread` 返回 1 | `exit 1` | `error` / `context_changed`，退出码 1 | 不禁用 |
| `resume.gt1` | 构造 + 真实 API | 2 | 可能已运行 | A12 | 夹具自己的 `SuspendThread` 返回 1 | `exit 1` | 同上 | 不禁用 |
| `resume.-1` | 接缝改变前提 + 真实 API 的失败 | `-1`（`ERROR_ACCESS_DENIED`） | 可能已运行 | A12 | 线程句柄的权限换为 `0x800` | `exit 1` | 同上 | 不禁用 |
| `pre.assign` | 接缝注入 | 没有调用 | 确定未运行 | A4 | 加入 Job：注入，没有调用 | —（外层：`exit 1`） | 不记录 | 不禁用 |
| `pre.snapshot` | 接缝注入 | 没有调用 | 确定未运行 | A4 | 快照：注入 | —（外层：`exit 1`） | 不记录 | 不禁用 |
| `pre.openthread` | 接缝注入 | 没有调用 | 确定未运行 | A4 | 打开线程：注入 | —（外层：`exit 1`） | 不记录 | 不禁用 |
| `pre.unconfirmed` | 接缝注入 | 没有调用 | 确定未运行 | A5 | 加入 Job 注入；`TerminateProcess` 被吞掉 | —（补救之前外层确认进程仍存在） | 不记录 | **禁用** |
| `anomaly.unconfirmed` | 构造 + 真实 API + 接缝注入 | 2 | 可能已运行 | A6 | 夹具的 `SuspendThread` 返回 1；Job 的查询注入失败 | `exit 1` | `error` / `context_changed`，退出码 1 | **禁用** |

- 记录判定中，恢复异常的行都为 `started_at` 取 `t_anom`、`duration_ms = t_S3 − t_anom`（两次都为 0 ms，进入 S3 紧接在异常之后）、`head_end` NULL、`tree_state_end = unknown`、不合格、停止、退出码 6；`exit_code` 为 Runner 可靠读到的值，与外层自己的读数相同。`resume.0` 与 `anomaly.unconfirmed` 的退出码 1 来自终止（`TerminateJobObject(job, 1)`），按 §10.1 保留数值、不推断因果。
- 所有权：A4、A10、A12 的 Job 句柄、Runner 的进程句柄、线程句柄都关闭成功，Go 的进程对象经 `Wait` 释放；A5、A6 的 Job 句柄与线程句柄关闭成功，Runner 的进程句柄移交资源清单并仍有效，Go 的进程对象没有 `Wait`，没有 goroutine 处在进程等待的帧中。
- Q-1：只有 A5、A6 禁用宿主（原因 `unconfirmed_end`），下一次检查被拒绝，清理与清单释放之后仍被拒绝；A4、A10、A12 照常准入。
- **辅助读数**（不作证据，原样记录）：主线程的 `QueryThreadCycleTime`：`healthy`（恢复并运行到退出）约为 1792 万 ~ 1812 万；其余场景在终止之后读取都在约 153 万 ~ 192 万之间，其中包括主线程**从未被恢复**的 `pre.*`、`resume.gt1`、`resume.-1`、`anomaly.unconfirmed`，以及夹具恢复之后很快被终止的 `resume.0`；未恢复、也未终止时（`pre.unconfirmed`）约为 2.1 万。所以周期时间不能区分是否执行过用户代码，这印证了只把它当作辅助读数。标记文件只有 `healthy` 写出；`resume.0` 中夹具恢复了线程，却没有标记文件（进程在到达用户代码之前就被终止）——没有标记文件不证明没有运行。
- 外层不嵌套 Job，以自己的进程句柄确认结束（辅助进程不创建后代）；每一次的清理都 “verified: 1 processes gone”。

**直接检查** `TestQ12Classify`：三平台都通过（Q-1 的触发表 10 行与宿主的拒绝；Q-2 的判定表 5 行；goroutine 证据的识别）。

**总期限的使用**：P-1 在每个作业中约 10 ~ 12 s（总期限 3 min），P-2 约 7 s（3 min），P-4 约 10 s（4 min）；没有一次因总期限而不执行。三个作业的总时长为 5.6 ~ 7.4 min，在 workflow 的 20 min 与 `go test` 的 15 min 之内。

### 14.8 作者本地运行与变异（作者证据，与诊断 CI 分开）

- **本地运行**：Windows 11（26300）上 P-1 6/6、P-4 18/18、`TestQ12Classify` 通过；WSL2（内核 6.18，不带 `-race`）上 P-1 6/6、P-2 9/9。完整的原型套件：WSL 上 53 个场景中 52 个 confirmed、1 个 unsupported（既有的 `T5.adapter.pwsh-argv`，没有 pwsh），门禁回归 82/82、组合 13/13。
- **本地发现并修正的问题**（如实保留）：
  - 新类型 `outcome` 与既有的同名类型冲突，编译失败；改名为 `caseOut`。
  - 第一次本地完整运行（Windows）中 `TestStartPoints` 失败：P-2 的检查进程直接调用 `cmd.Start()`，绕过了统一的启动入口 `start`；改为 `start(cmd)`，规则本身没有改。
  - P-4 的初稿把夹具的构造放在 Runner 的线程快照之前；那样被恢复的 Go 程序立即创建更多线程，Runner 的快照会看到多于一个线程（§19.7 的 5'），得不到真实的 0。改为在 Runner 打开线程之后、调用 `ResumeThread` 之前构造（仍是 14.5 写的“在 Runner 恢复之前”）。这一修正发生在第一次运行 P-4 之前。
- **隔离变异**（只在作者本机，变异之后已撤回，没有提交）：
  - 在 P-1 的 F1 分支中加入 `go func() { _ = cmd.Wait() }()` → Windows 与 WSL 上 `P1.alive.1` 都为 refuted（“no goroutine waits for a process”“no new prototype goroutine”）；
  - 让 P-2 的过滤器不安装 → `P2.pidfd_open.EPERM` 与 `P2.prctl.set0` 都为 refuted（过滤器的两侧读数、被拒绝的 errno、退出码等）。
  - macOS 没有变异。

### 14.9 没有验证的边界

- **P-1**：
  - F1 都由接缝构造（观察失败一次、终止请求被吞掉），不是真实的不可中断睡眠或真实的观察故障；
  - Windows 的 `alive` 依赖外层另持被测 Job 句柄的构造：真实 F1 中关闭 Job 句柄会再次请求终止，进程通常随之结束，所以这一变体只证明这一前提下的所有权、不阻塞与准入，不证明真实 F1 中进程仍存活；
  - goroutine 的证据只覆盖所列的等待帧与原型代码中的 goroutine，不覆盖依赖库内部未来可能新增的等待方式；
  - 宿主对象是测试中的最小模型，不是生产的宿主对象；被拒绝调用的“数据库、git、进程”计数只是模型的接缝，生产中的零访问须在授权之后的生产测试中证明（§19.1）。
- **P-2**：
  - 只有 x86_64 的 Ubuntu runner 与 WSL2；没有 arm64 的 Linux、旧内核或真实受限的容器；
  - 每个过滤器只拒绝一个调用的一种 errno，不覆盖组合；
  - “命中”的证据是过滤器的读数、被拒绝的调用返回规则的 errno 与参数对照；不是内核层面的命中计数；
  - 检查进程只是最小的 `exit 0`，没有组、清扫与 F1 的组合。
- **P-4**：
  - 返回 0 与大于 1 是测试有意构造的前提变化，不称为自然发生的故障；`-1` 的失败是真实的 API 失败，但前提由接缝改变；
  - 5'（快照显示多于一个线程）、返回 1 之后关闭线程句柄出错、`CreateJobObject` / `SetInformationJobObject` / `OpenProcess` 的原生失败都没有构造；
  - 恢复之前失败的三个步骤与终止未确认都是接缝注入，不是原生故障；
  - Q-2 的记录只是模型的判定，没有接入生产存储、事务或迁移；写事务之前与之后的取消只在直接检查的判定表中覆盖；
  - 只有 Windows Server 2025（26100）的诊断与作者本机 Windows 11（26300）。
- **P-3**：没有执行（14.6）。
- 原型成功不是生产实现正确；生产的 Runner、宿主对象、记录与事务都没有授权（§12）。

### 14.10 门槛的状态（待审查）

| 门槛 | 本轮之前 | 本轮 | 状态 |
|---|---|---|---|
| G1 | 规格写定（O3、D21），没有最小原型的需要 | Q-1 加入禁用条件（规范 §19.1 规则 2、8）；P-1、P-4 驱动了模型的准入与置位 | 规格待审查；生产测试待授权 |
| G2 | 规格写定 | 不变 | 不变 |
| G3 | 需要 P-1 | P-1 在三平台执行，18 次都为 confirmed（14.7） | 原型完成，待审查；限制见 14.9 |
| G4 | 规格写定 | 不变 | 不变 |
| G5 | 需要 P-3 | 只整理支持矩阵、ABI 来源与证据缺口（14.6） | **没有满足**：Intel 没有动态证据，`siginfo_t` 的版本没有登记 |
| G6 | 需要 P-2 | P-2 在 Ubuntu 执行，9 次都为 confirmed | 原型完成，待审查；只有 x86_64 |
| G7 | 需要 P-4 | P-4 在 Windows 执行，18 次都为 confirmed；A12 的记录判定按 Q-2 (a) | 原型完成，待审查；未构造的失败点见 14.9 |
| G8、G11 | 规格方向成立 | 不变 | 不变 |
| G9 | 继承支持 | 不变 | 不变 |
| G10 | 规格方向成立 | 不变 | 不变 |
| G12、D14 | 记录语义待 Q-2 | Q-2 (a) 已决定，A12 两行写入真值表（规范 §19.10）；模型的判定表由直接检查覆盖 | 规格待审查；D14 的约束与迁移待授权 |
| G13 | 只有注入或推理 | 不变（真实的 F1 没有构造） | 不变 |
| G14 | 非阻塞 | 不变 | 不变 |
| G15 | 生产阶段 | 不变 | 不变 |

这些状态都是作者的登记，待独立审查；不宣布任何门槛满足，不冻结规范。

### 14.11 Review 095 B1：新增观察入口的错误传播（修复）

> Review 096 之后：本表 P-1 屏障一行的“其他错误 → 观察失败”遗漏了区间退出时清除 subreaper 的错误，那时 `exclusive` 自己丢弃它（B1-R）。“计划外的观察失败使运行 invalid”与“过滤器装不上 → unsupported（不变）”另有两处例外（N-096-2）。见 §14.15、§14.17。本节保留原文。

[Review 095](reviews/codex-review-095.md) 对 `1be48c3` 为**技术 NO-GO**。B1：P-4、P-2、P-1 的新增观察入口没有完整接入 §14.2 的失败关闭规则——P-4 的退出码读取错误被吸收为合法的 NULL，外层的一次读取失败被之后的成功掩盖，仍可 confirmed / 0；P-2 正常组的步骤错误只被打印；P-1 读不到的基线被当作“有子进程而拒绝”。修复限定在这三个原型的观察链与它们的调用者；预测、预算、归属规则与清理要求不变，没有减少断言、扩大允许的错误或增加重试（EINTR 在原有期限之内的重试除外，见 P-2）。

**区分两类失败**（§14.2 的细化，不改变计划）：被测机制按计划返回的拒绝或故障（seccomp 的 errno、恢复异常、计划的 Job 查询失败、被吞掉的终止请求）照常按机制判定；用来观察机制的工具发生计划外的失败（读取、等待、关闭、状态读数），使这次运行 invalid，之后的成功、补救或其他成功的读数都不豁免它。机制把观察失败当作 F1 处理（C-19）这一点不变：在内层，它是被测分支；在实验的有效性上，它是计划外的工具失败。

**逐项的错误传播**（“新”为本轮的修复；其余为审计之后保持不变的入口）：

| 入口 | 返回的状态与错误 | 保存者 | 调用者 | 最终判定 |
|---|---|---|---|---|
| **P-1** 起点的 `procInfo`、E-main 观察者的创建、`awaitReport` | 状态 / error | `p1Inner` | `p1Inner` | 不是 present、错误 → invalid（不变） |
| P-1 `exited` 夹具：`p1AwaitZombie` | 是否僵尸、**读取错误**（新） | `p1Inner` | `p1Inner` | 读不到 → 观察失败（新，原来与“没有退出”混同）；没有看到退出 → invalid |
| P-1 `healthy` 的 E-main（Unix `obs.wait`；Windows 的等待与 `GetExitCodeProcess`） | 是否退出、状态、error | `p1Inner` | `p1Inner` | 计划外的错误 → **观察失败**（新）；机制照 C-19 走 F1 |
| P-1 S5：`groupLive` / Job 的 `ActiveProcesses` | 成员 / 计数、error | `p1Inner` | `p1Inner` | 错误 → 观察失败（不变） |
| P-1 终止请求（`kill`、`TerminateJobObject`） | error | — | `p1Inner` | 机制的请求，不是观察；结束由 S5 确认（C-17，不变） |
| P-1 E-main 观察者的关闭 → `fdClosed`（Unix） | 已关闭、能否判断 | 驱动 | 驱动 | 不能判断 → 观察失败（不变） |
| P-1 外层的分支确认（Unix `procInfo`；Windows 零等待、`IsProcessInJob`、`GetExitCodeProcess`） | 状态、error | 驱动 | 驱动 | 读不到 → 观察失败；没有命中 → invalid（不变） |
| P-1 清单项的核对（Windows 对移交句柄的零等待） | 事件、error | 驱动 | 驱动 | 错误 → **观察失败**（新，原来只使断言不符，判 refuted） |
| P-1 `p1Barrier` → `exclusive` 的基线读取 | `errBaseline`、**`errBaselineUnread`**（新）、其他错误 | `exclusive` | `p1Barrier` | 读不到、其他错误 → **观察失败**（新）；只有读到非空的基线才算屏障的拒绝；`healthy` 必须实际进入区间（新） |
| P-1 subreaper 的读取、补救、清单的释放 | 值 / error | 驱动 | 驱动 | 错误 → 观察失败（不变） |
| P-1 释放之后的准入（Review 095 N1） | — | 宿主模型 | 驱动 | 取样移到释放之后（新） |
| **P-2** 实验子进程的每一步（探测、设置、清除） | errno、值 | 报告的步骤 | 外层 `p2Judge`（新） | 与该组声明的**完整**步骤序列逐步比较：计划外的 errno → 观察失败；计划的拒绝没有发生 → 故障没有命中（invalid）；观察步骤的值不符 → 负载没有建立；序列或探测值不同 → refuted（机制走了另一条路）。原来只扫描被拒调用之前的前缀，正常组完全不扫 |
| P-2 检查进程的观察（`pidfd_open`、`poll`、`waitid WNOWAIT`、关闭） | errno、值；**EINTR 在剩余期限内重试**（新） | 报告 | `p2Judge` | 同上；失败之后不再落入没有期限的 `waitid` 或 `Wait`：登记之后结束并在 5 s 内回收（新） |
| P-2 检查进程的登记（新） | 检查进程自己的 `pid` 记录 | 场景（`awaitOwned`） | `runP2` | 外层在它仍是实验子进程的子进程时核对归属并登记，然后写确认文件；实验子进程收到确认之后才回收它。核对不了 → invalid |
| P-2 实验子进程自读 `/proc/self/status` | 值、**error**（新） | 报告 | `runP2` | 错误 → 观察失败（新，原来变成字符串，判 refuted） |
| P-2 外层读 `/proc/<pid>/status` | 值、**error**（新） | `runP2` | `runP2` | 错误 → 观察失败（新） |
| P-2 实验进程的子进程、pidfd、报告的读取、回收 | 列表 / error / bool | `runP2` | `runP2` | 错误 → 观察失败或 invalid（不变） |
| P-2 过滤器装不上 | 原因 | 报告 | `runP2` | unsupported（不变；不算通过） |
| **P-4** 启动链（加入 Job、`IsProcessInJob`、快照、`OpenThread`） | error | `p4Run` | `p4Run` | 计划的接缝 → 被测分支；**计划外的真实失败 → 观察失败**（新，原来落入 A4 后判 refuted） |
| P-4 Runner 的 `ResumeThread` | 返回值、error | `p4Inner.Resume` | `runP4` | 与这一行的计划不符 → 负载没有建立（新，原来判 refuted） |
| P-4 Runner 对进程句柄的等待（正常启动与恢复异常）与读退出码 | 事件、error | `p4ReadExit`（新） | `p4Run` | 超时 → 合法的“没有取得”（记录 NULL）；**等待或读取失败 → 观察失败**（新，原来被吸收为 NULL，异常路径的等待错误被忽略） |
| P-4 `p4Exit`（`GetExitCodeProcess`） | 值、**error**（新） | 调用者 | `p4ReadExit`、`runP4` | 错误不再变成空值 |
| P-4 Job 的查询（`p4Confirm`） | 是否确认、error | `p4Run` | `p4Run` | 真实错误 → 观察失败；计划的查询故障 → 不确认（不变） |
| P-4 外层的零等待与读退出码 | 事件 / 值、error | `runP4` | `runP4` | 错误 → **观察失败**（新） |
| P-4 清单句柄的零等待 | 事件、error | `runP4` | `runP4` | 错误 → **观察失败**（新） |
| P-4 两侧退出码的对照 | — | `runP4` | `runP4` | Runner 取得 → 必须与外层相同；Runner 合法地没有取得（超时）→ 记下理由；不再因一方为空而豁免（新） |
| P-4 夹具的构造（`ResumeThread` / `SuspendThread`）与接缝命中 | 返回值 | 夹具事件 | `runP4` | 不符 → 负载没有建立（不变） |
| P-4 辅助读数（周期时间、标记文件） | 值或 unsupported | `runP4` | — | 不参与任何判定（不变） |
| P-4 外层的清理 | 事件、error | `runP4` | `runP4` | 外层没有亲见结束就终止（新，原来只在“仍存活”时终止）；没有确认结束 → 观察失败 |
| 共享模型（宿主、栈证据、总期限） | 没有错误返回 | — | — | 不变 |

**本轮新发现的登记缺口**（回归先行时暴露，与 B1 同源，一并修复）：P 测试原来跳过门禁的内层运行，所以 P-2 从未在门禁之下运行。本轮让它们也作为门禁的内层运行之后，外层普查报告 P-2 的检查进程“已创建、没有被内层登记”（`control.p2-normal`、`control.p2-set0` 的内层 confirmed / 0，门禁不通过）。检查进程由实验子进程创建并回收，场景从未登记它，违反 §2.2 的归属规则。修复见上表的“检查进程的登记”。拒绝组的报告声明 `pid` 恰好 0 条：机制错误地启动检查时，判为 invalid（保守），不是 refuted。

**门禁回归的判定**（新）：门禁用例可以要求内层的理由包含被注入的那次失败本身（`because`）。B1 的用例都这样要求，所以一次故障通过其他后果（例如使负载没有建立）碰巧得到 invalid，不算检出。

### 14.12 回归先行、修复与诊断

**提交的顺序**：

1. `9d80431`：原样归档 Review 095。
2. `0a9a01a`（回归先行，旧实现没有改动）：
   - 新增观察入口各加一个一次性故障（只失败一次，之后的读取照常）；
   - 增加 12 个 B1 门禁用例与 8 个对照；
   - P 测试也作为门禁的内层运行。
3. `85ec2ca`（修复，**固定的原型代码点**）：14.11 的修复，以及门禁用例的 `because`。
4. 之后只有文档的提交：原型代码与 workflow 与 `85ec2ca` 相同（14.14）。

**B1 用例**（每个都是一次失败、之后读取成功；“旧”为 `0a9a01a`，“新”为 `85ec2ca`）：

| 用例 | 平台 | 旧实现的内层结论 / 退出码 | 新实现 |
|---|---|---|---|
| `p4.exitcode-fail-once`（审查的原反例：A12 中 Runner 读退出码失败） | Windows | **confirmed / 0** | invalid / 非零：the exit code of the main process |
| `p4.outer-wait-fail-once`（审查的外层变体：`pre.assign` 中外层零等待失败） | Windows | **confirmed / 0** | invalid：the outer's wait |
| `p4.anomaly-wait-fail-once`（异常路径的等待错误） | Windows | **confirmed / 0** | invalid：waiting for the main process |
| `p4.outer-exitcode-fail-once`（外层读退出码失败） | Windows | **confirmed / 0** | invalid：the outer's read of the exit code |
| `p4.exitcode-fail-once-healthy`（正常启动路径） | Windows | refuted（把工具失败当作发现） | invalid |
| `p4.item-wait-fail-once`（移交句柄的读取） | Windows | refuted | invalid |
| `p1.item-wait-fail-once`（移交句柄的读取） | Windows | refuted | invalid |
| `p1.observe-fail-once`（`healthy` 的 E-main） | 三平台 | invalid，但理由是“负载没有建立”（碰巧，不是观察失败本身） | invalid：the healthy control's（理由点名观察失败） |
| `p1.baseline-unread-once`（审查的原反例：屏障的基线读不到） | Linux | **confirmed / 0** | invalid：the barrier's baseline |
| `p2.check-poll-fail-once`（审查的原反例：正常组检查进程的 poll） | Linux | **confirmed / 0**（另有登记缺口） | invalid：unplanned result of poll(check pidfd |
| `p2.close-fail-once`（拒绝分支中关闭自有 pidfd） | Linux | **confirmed / 0** | invalid：unplanned result of close(own pidfd) |
| `p2.status-unreadable-once`（外层读状态） | Linux | refuted | invalid |

- 对照 `control.p1-alive`、`control.p1-healthy`、`control.p2-denied`、`control.p4-resume-gt1`、`control.p4-pre-unconfirmed`、`control.p4-anomaly-unconfirmed` 在旧、新实现上都是 confirmed / 0。其中包括计划内的 seccomp 拒绝、恢复异常与计划的 Job 查询失败：它们没有被一律判为 invalid。
- `control.p2-normal`、`control.p2-set0` 在旧实现上，内层 confirmed / 0，但门禁不通过：外层普查报告检查进程没有被登记（14.11 的登记缺口）。新实现上两者都通过。
- 每个 B1 用例的故障都有命中记录（hits 1）。

**作者本机**（作者证据）：

- Windows 11 26300：上表的 Windows 用例在旧实现上如上表；新实现上全部通过。完整原型套件为场景 53/53 confirmed、门禁回归 91/91、组合 13/13。
- WSL2 6.18（不带 `-race`）：Linux 用例同上表；新实现上全部通过。完整原型套件为场景 52 confirmed、1 unsupported（既有的 `T5.adapter.pwsh-argv`，没有 pwsh），门禁回归 92/92，组合 13/13。

**诊断 CI**（`.github/workflows/wp25-prototype.yml`；与常规 CI 分开，都没有重跑或取消）：

| run | 代码点 | 结论 | macOS 26.6.2 arm64 | Ubuntu 6.17（`-race`） | Windows 10.0.26100 |
|---|---|---|---|---|---|
| 40641559468 | `0a9a01a`（回归先行） | **failure**（预期） | failure：场景 33/33，门禁回归 75/76 | failure：场景 53/53，门禁回归 86/92 | failure：场景 53/53，门禁回归 84/91 |
| 40643975558 | `85ec2ca`（修复） | **success** | 场景 33/33，门禁回归 76/76，组合 13/13 | 场景 53/53，门禁回归 92/92，组合 13/13 | 场景 53/53，门禁回归 91/91，组合 13/13 |

- 两个 run 的三个作业都实际检出各自的代码点，WP25META 声明相同的 SHA。
- 40641559468 的门禁失败：
  - Ubuntu 6 项：`p1.baseline-unread-once`、`control.p2-normal`、`control.p2-set0`、`p2.check-poll-fail-once`、`p2.close-fail-once`、`p2.status-unreadable-once`；
  - Windows 7 项：`p1.item-wait-fail-once` 与上表 6 个 P-4 用例；
  - macOS 1 项：既有的 `report.epipe-unwritable`（Review 096 N-096-1 收窄了此处原来的写法，见 §14.17）：
    - 内层为 invalid / 1，预期的报告失败拒绝已出现；
    - 外层的普查读 PID 2668 的环境得到 EINVAL，列入 `tool_errors`，Final=unknown、Pass=false。

    这**不属于**十二个 B1 目标故障的预期命中，是 §2.14、§2.15 记录的 macOS 普查读取失败的又一次观测。PID 2668 的归属、EINVAL 的根因与该运行对所有对象的完整清理**仍未知**。没有重跑。
- 40643975558 中，B1 用例在各自的平台上都是 invalid / 非零，理由点名被注入的失败；对照都是 confirmed / 0。P 场景：
  - Windows：P-1 6、P-4 18，都 confirmed；
  - macOS：P-1 6，都 confirmed；
  - Ubuntu：P-1 6、P-2 9，都 confirmed。

**常规 CI**（`ci.yml`，不运行原型模块）：

- 回归先行的 `0a9a01a` 上，run 40641564083 为 success 8/8，而同一代码点的原型诊断为 failure。这说明常规 CI 不能代替原型诊断。
- 最终 HEAD 的常规 CI 写在 PR 描述中。

### 14.13 定向撤回（作者本机，变异之后已恢复，没有提交）

每次只撤回一项关键修复，运行应检出它的 B1 用例：

| 撤回 | 应检出的用例 | 结果 |
|---|---|---|
| M1 P-4 Runner 读退出码失败不报告 | `p4.exitcode-fail-once`、`p4.exitcode-fail-once-healthy` | 检出（前者 confirmed / 0，后者 refuted；理由都没有点名观察失败） |
| M2 P-4 异常路径的等待失败不报告 | `p4.anomaly-wait-fail-once` | 检出（confirmed / 0） |
| M3 P-4 外层的零等待失败不报告 | `p4.outer-wait-fail-once` | 检出（confirmed / 0） |
| M4 P-4 外层读退出码失败不报告 | `p4.outer-exitcode-fail-once` | 检出（confirmed / 0） |
| M5 P-4 移交句柄的读取失败不报告 | `p4.item-wait-fail-once` | 检出（refuted） |
| M6 P-1 `healthy` 的观察失败不报告（Windows） | `p1.observe-fail-once` | 检出（invalid，但理由只是“负载没有建立”） |
| M7 P-1 移交句柄的读取失败不报告 | `p1.item-wait-fail-once` | 检出（refuted） |
| M8 P-1 读不到的基线不报告 | `p1.baseline-unread-once` | 检出（refuted） |
| M9 P-1 `healthy` 的观察失败不报告（Unix） | `p1.observe-fail-once` | 检出（同 M6） |
| M10 P-2 不逐步判定 | `p2.check-poll-fail-once`、`p2.close-fail-once` | 检出（前者 refuted，后者 confirmed / 0） |
| M11 P-2 外层读状态失败不报告 | `p2.status-unreadable-once` | 检出（refuted） |
| M12 P-2 观察失败之后不等登记就结束检查进程 | `p2.check-poll-fail-once` | 检出（invalid，但理由是归属无法确认，不是 poll 失败） |

- Windows 的 M1 ~ M7 在 Windows 11 上运行，Linux 的 M8 ~ M12 在 WSL2 上运行；macOS 没有变异。
- M6、M9、M12 说明了为什么需要 `because`：只看结论，这三项撤回都检不出来，因为那次故障还通过其他后果使运行 invalid。
- 修复过程中如实保留的问题：
  - 修复的第一稿中，P-2 在观察失败之后没等外层登记就结束了检查进程，`p2.check-poll-fail-once` 的理由因此是归属失败。已改为不论观察结果都先等登记（即 M12 撤回的那一项）。
  - P-4 正常启动路径原有一处对同一等待错误的报告，与新的 `p4ReadExit` 重复，已删去。

### 14.14 受影响的证据范围与门槛的状态

- **降为历史观测**：§14.7 中固定代码点 `d45a319` 的诊断 run 40439516254，以及 §14.8 的作者本地运行，是 B1 修复之前的观测，保留原文。新增观察入口当时没有失败关闭，所以它们的 P-1、P-2、P-4 结论只是“在没有发生这些读取错误的那次运行中观察到的结果”，不是有效门禁之下的证据。P-2 的检查进程当时也没有被登记。
- **替代的证据**：固定代码点 `85ec2ca` 的诊断 run 40643975558（14.12）。§14.7 的结论表在新代码点上逐项重现：P-1 18/18、P-2 9/9、P-4 18/18 都为 confirmed。各项的数值（构造的返回值、记录判定、所有权、Q-1 的准入）以 40643975558 的日志为准。
- **证据中不变的部分**：§14.9 的未验证边界、P-3 的整理（§14.6）不受影响。
- **固定点之后**：之后只有文档的提交；`git diff 85ec2ca HEAD -- prototype .github` 为空（最终核对见 review-resolution §115 与 PR 描述）。

**门槛的状态**（作者的登记，待独立复审；§14.10 是修复之前的登记，保留原文）：

| 门槛 | 状态 |
|---|---|
| G1 | Q-1 的规格写定；模型的准入、置位与“释放之后仍拒绝”（N1 更正之后按实际顺序取样）由 P-1、P-4 驱动；生产接线待授权 |
| G3 | P-1 在有效门禁之下三平台 18/18 confirmed；Windows `alive` 依赖外层另持 Job 句柄的构造；不宣布整体满足 |
| G5 | **没有满足**（本轮不补做 P-3） |
| G6 | P-2 在有效门禁之下 Ubuntu 9/9 confirmed，检查进程已登记；只有 x86_64；不宣布满足 |
| G7 | P-4 在有效门禁之下 Windows 18/18 confirmed；未构造的失败点见 §14.9；不宣布满足 |
| G12、D14 | Q-2 (a) 的规格写定；生产记录、事务与迁移待授权 |
| 其余 | 不变 |

### 14.15 Review 096 B1-R：排他区间自身步骤的错误传播（修复）

[Review 096](reviews/codex-review-096.md) 对 `b80982c` 为**技术 NO-GO**：

- Review 095 点名的 P-4、P-2、P-1 路径修复成立；N1、N2 关闭。
- B1 不能整体关闭（**B1-R**，源码条件反例）：Linux 排他区间 `exclusive` 退出时清除 subreaper（`PR_SET_CHILD_SUBREAPER 0`）的错误被 `_ =` 丢弃，`exclusive` 只返回回调的结果。P-1 的屏障因此可以把清除失败报告为成功进入，`healthy` 的预测全部一致，得到 confirmed / 0。
- §14.11 中 P-1 屏障一行的“其他错误 → 观察失败”因此不成立：`exclusive` 自己就不返回这个错误。

修复只改原型的 `exclusive` 与它的直接调用者。规范 §19.6 已写定生产的处理：收尾时 SET 0 失败作为环境错误，批次停止、退出码 6，租约照常释放，之后的探测因读取为 1 而拒绝启动。本轮不改规范，不增加宿主禁用条件、重试或恢复政策；原型也不实现这些生产行为。

**修复**：

- `exclusive` 现在接受场景 `s`。区间自身步骤的失败，都在 `exclusive` 内部记入 `s.flag`，作为持续保存的观察失败：
  - 读不到的基线；
  - SET 1 失败；
  - SET 0 失败。
- 因此，不论调用者检查、计数还是丢弃返回值，不论回调正常返回、还是经 `invalidf` / `FailNow` 结束本 goroutine，这些错误都有消费者。
- 错误同时返回：清除失败与回调的错误用 `errors.Join` 合并，互不覆盖。
- 只有读到而且不为空的基线是拒绝（`errBaseline`），不记入 flag。这是计划内的机制结论，按预测判定。
- 释放租约的 defer 先注册、后执行：清除与 flag 在前，释放在后，任何结束方式下都执行。flag 只取场景的锁，不取闸门的锁，所以不会遗漏释放，也不增加死锁或等待。
- **审计直接调用者时一并处理的两项**（与 B1-R 同在 `exclusive` 的结果中）：
  - T2 的基线场景把读不到的基线也当作拒绝，可以得到 confirmed / 0；
  - T2 的两个计数调用者把 SET 1 失败当作检查被拒，得到 refuted。

**区间的步骤**：

| 步骤 | 返回 | 保存者 | 何时消费 | 最终判定 |
|---|---|---|---|---|
| 基线读取（`childrenOf`） | 列表 / error | `exclusive`：`s.flag`（新），并返回 `errBaseline` + `errBaselineUnread` | flag 立即记入，`done` 汇总 | invalid（新；原来 T2 的基线场景当作拒绝） |
| 基线不为空 | `errBaseline` | 调用者 | 返回时 | 计划的拒绝，按预测判定（不变） |
| SET 1 | error | `exclusive`：flag（新），并返回 | 立即 | invalid（新；原来 T2 的计数调用者为 refuted） |
| 回调 `fn` | error；或经 `invalidf` 结束 goroutine | 调用者；或 `invalidf` 已记入场景 | 返回时；或已记入 | 不变。回调提前结束时，`exclusive` 的两个 defer 照常执行 |
| SET 0（defer） | error | `exclusive`：flag（新），并 `errors.Join` 进返回值（新） | flag 立即记入，回调经 Goexit 结束、没有人接收返回值时也是；返回时 | invalid（新；原来丢弃） |
| 释放租约（defer） | 没有错误 | — | 每次都执行 | 不变 |

**直接调用者**（13 处）：

| 调用者 | 对返回值 | 清除失败：修复之前 | 修复之后 |
|---|---|---|---|
| P-1 内层 `p1Inner`（经 `p1Exclusive`） | 检查 → `invalidf` | 返回 nil；之后读到 subreaper 为 0 → confirmed / 0 | flag，加 `invalidf` → invalid |
| P-1 屏障 `p1Barrier` | 分类；`healthy` 要求进入 | 进入、nil → confirmed / 0 | flag，加屏障自己的 flag → invalid。之后读到的设置（新增的读数）单独记录 |
| `T2.gate.exclusive-interval` | 计数（nil 才算成功） | confirmed / 0 | invalid |
| `T2.gate.concurrent-checks` | 计数（错误算被拒） | 清除错误不出现 | invalid |
| `T2.adoption.chain` | 检查 → `invalidf` | confirmed / 0 | invalid |
| `T2.gate.bypass`、`T2.other-reaper`、`T2.identity.mismatch`、`T2.pidfd.unsupported`、`T2.F1.resources` 的第一次 | 丢弃（`_ =`） | confirmed / 0 | invalid：flag 在 `exclusive` 内部，丢弃返回值也不漏判 |
| `T2.baseline.live-child`、`T2.baseline.zombie-child`、`T2.F1.resources` 的第二次 | 期望拒绝；回调不运行 | 没有设置，也就没有清除；读不到的基线原来得到 confirmed | 读不到的基线 → invalid |

**不适用的退出路径**（不制造对应的测试）：

- **拒绝之后的清除**：基线不为空、读不到的基线与 SET 1 失败，都在注册清除的 defer 之前返回。这时 subreaper 从未设置，没有可清除的东西。
- **回调返回错误、同时清除失败**：13 处回调都不返回非 nil 的错误。合并已实现，但没有调用者走到这里。可达的“回调失败”是经 `invalidf` 提前结束，由组合用例覆盖。
- **回调 panic**：原型的回调不 panic。panic 时 defer 照常执行、flag 照常记入，测试进程随后失败，不可能得到绿色。
- **macOS、Windows**：macOS 的 `p1Exclusive` 只运行回调，没有 subreaper；Windows 没有 `exclusive`。

**注入与实际状态**：

- **清除的故障**：真实清除照常执行，只替换它的返回值。每次命中都记下真实清除的返回：本轮所有命中都是 `<nil>`，即真实清除成功、属性为 0。之后的读数也都是 0：
  - `subreaper after the barrier's interval=0`；
  - `subreaper after the inner call=0`（旧实现上）；
  - `subreaper after the intervals=0`。

  所以这些运行**没有**验证“清除失败之后属性保持为 1”，也没有验证真实的原生清除失败之后会怎样。
- **属性可能残留**：证据是 P-2 `prctl.set0` 组在隔离的实验进程中的既有结果（run 40643975558）：SET 0 被 EPERM 拒绝，之后读到 1（SubEnd=1），外层最后结束实验进程。实验进程的退出不是清除成功。本轮没有新增实验。
- **SET 1 的故障**：不执行真实调用（命中记录写明），没有设置，也就没有可清除的东西。
- **测试进程中真实的清除失败**：原型不重试、不恢复，属性可能在同一测试进程之后的场景中保持为 1。这次运行已经 invalid；之后读取属性的 T2 场景也会不符。不可能得到绿色。
- **补救与清理单独记录**：门禁结果现在带有内层自己的清理（`inner_cleanup`）、命中的位置（`inject_sites`）与故障之后的读数（`inner_observed`）。本轮全部用例的结果相同：
  - 内层的清理已核实（进程都已结束）；
  - 外层的补救为 not-needed，保险清理为 nothing left，Final=ended；
  - 运行仍然 invalid。

  清理只作用于场景登记的进程，不对归属未确认的对象动作。

**本节对 §14.11 的更正**：

- P-1 屏障一行的“其他错误 → 观察失败”此前不成立。
- “subreaper 的读取、补救、清单的释放”一行只覆盖读取，不覆盖区间自身的设置与清除；后者见上表。

### 14.16 回归先行、修复、诊断与定向撤回

**提交的顺序**：

1. `73c8eb3`：原样归档 Review 096。
2. `b641e17`（回归先行，旧的错误处理没有改动）：
   - 区间的各个步骤加一次性故障：
     - P-1 的屏障与内层的清除；
     - 内层回调经观察者失败而提前结束，同时清除失败（组合用例）；
     - T2 的清除、读不到的基线与 SET 1。
   - 增加 8 个 B1-R 门禁用例与 3 个对照（`control.p1-exited`、`control.t2-gate`、`control.t2-bypass`）。下表的最后一行是 Review 095 既有的用例。
   - 测试工具：
     - `injectOnceAt`（一个故障在几个位置各作用一次）；
     - 闸门的 `label`，取代 `p1Probing`，既有基线故障的条件不变；
     - 门禁用例的 `alsoBecause`；
     - 门禁结果中的命中位置与内层读数；
     - P-1 屏障之后对设置的读数（预期 0），作为与区间结果分开的记录。
3. `be25df5`（测试工具，旧的错误处理仍未改动）：门禁结果也带有内层自己的清理。
4. `48ff0cd`（修复，**固定的原型代码点**）：14.15 的修复。
5. 之后只有文档的提交（14.17）。

**B1-R 用例**（Linux；“旧”为旧的错误处理，即 `b641e17` 与 `be25df5`，“新”为 `48ff0cd`；每个故障都有命中记录）：

| 用例 | 调用者对结果 | 旧实现的内层结论 / 退出码 | 新实现 |
|---|---|---|---|
| `p1.barrier-clear-fail-once`（审查的原反例） | 屏障分类 | **confirmed / 0** | invalid / 1。理由点名区间的清除失败与屏障自己的 flag |
| `p1.inner-clear-fail-once` | 检查 → `invalidf` | **confirmed / 0**（之后读到 subreaper 为 0） | invalid / 1。区间的 flag，加 `the exclusive interval: …` |
| `p1.observer-and-clear-fail-once`（回调失败，加清除失败） | 回调经 `invalidf` 结束 | invalid / 1，但理由只有观察者的失败，清除的失败丢失 | invalid / 1。两项都在理由中（命中 2） |
| `t2.clear-fail-once-counted`（`TestExclusiveGate`） | 计数 | **confirmed / 0**（之后读到 0） | invalid / 1 |
| `t2.clear-fail-once-checked`（`TestAdoptionChain`） | 检查 → `invalidf` | **confirmed / 0** | invalid / 1 |
| `t2.clear-fail-once-dropped`（`TestExclusiveBypass`） | 丢弃 | **confirmed / 0** | invalid / 1 |
| `t2.baseline-unread-once`（`TestExclusiveBaselineLive`） | 期望拒绝 | **confirmed / 0**（读不到的基线被当作拒绝） | invalid / 1 |
| `t2.set1-fail-once`（`TestExclusiveGate`） | 计数 | refuted / 1（把工具失败当作检查被拒） | invalid / 1 |
| `p1.baseline-unread-once`（既有，Review 095） | 屏障分类 | invalid / 1 | invalid / 1。区间的 flag 加在屏障的 flag 之前 |

- 每个用例的理由都须点名被注入的那次失败（`because`）。三个用例还须点名第二项（`alsoBecause`）：
  - 屏障与内层：调用者收到了返回的错误；
  - 组合用例：回调自己的失败。
- 对照在旧、新实现上都是 confirmed / 0：
  - `control.p1-healthy`、`control.p1-alive`、`control.p1-exited`（可靠的非空基线使屏障拒绝）；
  - `control.refusal`（T2 的可靠非空基线）；
  - `control.adoption-chain`、`control.t2-gate`、`control.t2-bypass`。
- 可靠的非空基线（`control.p1-exited`、`control.refusal`）与读不到的基线（`p1.baseline-unread-once`、`t2.baseline-unread-once`）仍然分开：前者 confirmed，后者 invalid。

**定向撤回**（作者本机 WSL2，每次只撤回一项，与对照一起运行；变异之后已恢复，没有提交）：

| 撤回 | 应检出的用例 | 结果 | 对照 |
|---|---|---|---|
| R1 清除失败只返回、不 flag | 6 个清除用例 | 全部检出。`t2.clear-fail-once-dropped` 退回 confirmed / 0，`t2.clear-fail-once-counted` 为 refuted，组合用例丢失清除的失败；屏障、内层、checked 只剩调用者的消息（`because` 检出） | 全部通过 |
| R2 清除失败只 flag、不返回 | 屏障、内层 | 检出（`alsoBecause`：调用者没有收到错误）。`t2.clear-fail-once-counted` **没有**检出：只靠 flag 它也是 invalid，返回值只多一项计数不符 | 全部通过 |
| R3 读不到的基线不 flag | `t2.baseline-unread-once` | 检出（confirmed / 0）。`p1.baseline-unread-once` 仍通过：屏障自己的 flag（Review 095）仍在 | 全部通过 |
| R4 SET 1 失败不 flag | `t2.set1-fail-once` | 检出（refuted） | 全部通过 |

macOS 没有变异：这些路径只在 Linux 上存在。

**作者本机**（作者证据，`48ff0cd` 的树）：

- Windows 11 26300：完整原型套件为场景 53/53 confirmed、门禁回归 92/92、组合 13/13。
- WSL2 6.18（不带 `-race`）：场景 52 confirmed、1 unsupported（既有的 `T5.adapter.pwsh-argv`，没有 pwsh），门禁回归 103/103，组合 13/13。
- B1-R 用例与对照在 `be25df5`（旧）与 `48ff0cd`（新）上各运行一次，结论如上表。

**诊断 CI**（`.github/workflows/wp25-prototype.yml`；与常规 CI 分开，都没有重跑或取消）：

| run | 代码点 | 结论 | macOS 26.6.2 arm64 | Ubuntu 6.17（`-race`） | Windows 10.0.26100 |
|---|---|---|---|---|---|
| 40656037766 | `b641e17`（回归先行） | **failure**（预期） | success：场景 33/33，门禁回归 77/77，组合 13/13 | failure：场景 53/53，门禁回归 95/103，组合 13/13 | success：场景 53/53，门禁回归 92/92，组合 13/13 |
| 40658194158 | `48ff0cd`（修复） | **success** | 场景 33/33，门禁回归 77/77，组合 13/13 | 场景 53/53，门禁回归 103/103，组合 13/13 | 场景 53/53，门禁回归 92/92，组合 13/13 |

- 40656037766：
  - 三个作业都实际检出 `b641e17`，WP25META 声明相同的 SHA。
  - Ubuntu 失败的门禁用例正是 8 个新增的 B1-R 用例；结论、退出码与命中位置与上表的旧实现相同。
  - Windows、macOS 上新增的只有对照 `control.p1-exited`，通过。
- 40658194158：
  - 三个作业都实际检出 `48ff0cd`，WP25META 声明相同的 SHA；Ubuntu 作业按既有的 `RUNNER_OS` 判断传入 `-race`。
  - B1-R 用例都为 invalid / 1，理由点名被注入的失败。清除的命中都记下真实清除返回 `<nil>`；SET 1 的命中记下没有执行真实调用。
  - 对照都为 confirmed / 0，Final=ended，保险清理为 nothing left。
  - P 场景都为 confirmed：Windows P-1 6、P-4 18；Ubuntu P-1 6、P-2 9（另有 T2 11）；macOS P-1 6。

**常规 CI**（`ci.yml`，不运行原型模块）：

- 回归先行的 `b641e17` 上，run 40656041186（pull_request，attempt 1）为 **failure，7/8**。
  - 八个作业都检出合并快照 `c9ea84f`（父提交 `2c5cf23`、`b641e17`）。
  - 失败的是 Windows 的 `internal/store/sqlite` `TestCheckWhileTheDatabaseIsCreated`（round 11、check 9，`Access is denied`）。与 M-1（codex-review-035 N2）是同一测试、相似的错误签名，登记为 M-1 的观测；不确认同根因，也不能据此认定与本轮增量无关（Review 097 §7，§14.18）。
  - 本轮没有改根模块；M-1 未解决，根因仍未确认。没有重跑。
- 修复之后与最终 HEAD 的常规 CI 写在 PR 描述中。

### 14.17 受影响的证据、两项非阻塞意见与门槛的状态

**降为历史观测**（保留原文）：

- 修复之前的各个代码点上，`exclusive` 的清除结果都没有被观察。受影响的是：
  - `85ec2ca` 的诊断 run 40643975558 中 Ubuntu 的 P-1 与 T2 结论；
  - §14.12 中 WSL2 的作者本地运行；
  - 更早的 T2 证据。
- 这些结论只是“在没有观察清除结果的那次运行中观察到的结果”。
  - 内层区间之后读到 0，只说明读取的那一刻属性为 0；
  - P-1 屏障区间之后原来没有读数。

**替代的证据**：固定代码点 `48ff0cd` 的诊断 run 40658194158（14.16）。

**不受这一缺陷影响**（不经 `exclusive`）：P-2、P-4、Windows 与 macOS 的结论。它们也由同一诊断在新代码点上重新运行；§14.14 对它们的登记不变。

**N-096-1**（措辞收窄，已处理）：

- §14.12 中“与 B1 无关”“一个无关进程”的写法已就地收窄。审查原文的线索：
  - PID 2668 的启动身份 `1793167516.433196`，父进程为 1，进程组 2668，名称前缀的启发式为 no；
  - 失败之后该身份仍然 present、不是僵尸。
- 父进程为 1、名称不匹配，既不能证明它与本轮无关，也不能证明它属于本轮。保险清理的“nothing left”不能解释为普查未能判定归属的 PID 2668 已完整清理。
- 门禁当时已经正确拒绝（Final=unknown、Pass=false），没有绕过 EINVAL。后来的成功不回填这次运行。

**N-096-2**（只登记，本轮不改代码）：

- **unsupported 的分类顺序**：`prctl.set0` 组预期有检查进程（checks=1）。过滤器不支持时，实验进程先发布 unsupported、不创建检查，外层却先等 10 s 的 PID 记录，得到 invalid，走不到 unsupported。
  - 这是保守的拒绝，不是 confirmed 的错误放行。
  - CI 的环境装上了过滤器，没有走到这个分支。
- **自读的 unknown**：`p2Probe` 把自读 `procInfo` 的 unknown 压成 `/proc=false`，之后可能得到 refuted，而不是保存读取失败。它在当前的正常组与拒绝组中不会成为 confirmed。
- **概括的收窄**：§14.11 的“计划外的观察失败使运行 invalid”与“过滤器装不上 → unsupported（不变）”，不覆盖这两处：
  - unsupported 的分类顺序先得到 invalid；
  - 自读的 unknown 可能得到 refuted。

  两处都不会得到 confirmed。
- **推荐的修正**（待负责人决定是否放进本 PR）：
  - 等登记时同时识别经核验的“不支持、没有启动检查”的实验结果，补不支持的对照；
  - 自读失败记为观察失败。

**门槛的状态**（作者的登记，待独立复审；§14.14 是本轮之前的登记，保留原文）：

| 门槛 | 状态 |
|---|---|
| G1 | 不变（§14.14） |
| G3 | P-1 在新代码点的有效门禁之下：Linux 的屏障与内层区间的清除结果都被观察；三平台 P-1 18/18 confirmed，以 40658194158 为准。Windows `alive` 依赖外层另持 Job 句柄的构造。不宣布整体满足 |
| G5 | **没有满足**（本轮不补做 P-3） |
| G6 | 不变：P-2 不经 `exclusive`。N-096-2 的两处分类边界登记待处理；只有 x86_64。不宣布满足 |
| G7 | 不变（§14.14）；不宣布满足 |
| G12、D14 | 不变 |
| 其余 | 不变 |

**固定点之后**：之后只有文档的提交，`git diff 48ff0cd HEAD -- prototype .github` 为空（最终核对见 PR 描述）。

### 14.18 Review 097 的结论与收尾

[Review 097](reviews/codex-review-097.md) 对 `800b5e8`（固定的原型代码点 `48ff0cd`）为**技术 GO**（review-resolution §117）：

- **B1-R 关闭**。结合 Review 096 对原点名路径的关闭，Review 095 B1 **在受审原型范围内整体收口**，限定为已审原型及其受控前提。没有新的 BLOCKER、confirmed 的假通过或直接回归；Review 095 N1、N2 保持关闭。
- **N-096-1**：§14.12、§14.17 的措辞修正成立，这项文档意见关闭；PID 2668 的归属、EINVAL 的根因与该次运行的完整清理仍是未知。
- **N-096-2**：仍未修，非阻塞，待另开独立 PR 修复（§14.17 的推荐修正）。修复并经审查之前：
  - §14.11 中 P-2 的两条分类概括只按 §14.17 收窄后的范围适用，不能宣传为完整的失败分类保证；
  - G6 的证据只适用于实际装上过滤器、完整执行了正常与拒绝序列的环境。

**证据的范围**（照审查原文，补充 §14.15 ~ §14.17）：

- 支持的是受控原型的错误传播，不是任意宿主、原生不可终止的故障或生产事务。
- 回调错误与清除错误的合并按源码流程核对（当前 13 处回调都不正常返回非 nil），没有原生的动态用例；经 Goexit 结束时，两项失败由两条 invalid 理由共同保存，组合回归实际覆盖了它。
- 原有的等待与共享持有者的生命周期没有改成统一有界，也没有增加超时政策。本轮只确认新增的错误保存没有引入新的等待依赖，不推出生产拓扑或系统调用的硬实时保证。
- 门禁结果的补充字段 `inject_sites`、`inner_observed`、`inner_cleanup` 是**可选的诊断字段**：
  - 不是外层结束或归属的证明，也不是写者认证；
  - site 的 JSON 解析失败时不追加位置，但标记仍可能计入命中；缺失时保留空值；
  - 它们没有被用来放宽判定。将来若升级为硬性的证据门禁，须同步严格校验，不能沿用当前的容错。
- `injectOnceAt` 的“每个 key 一次”只在本轮的调用拓扑中核对过没有冲突，不推广到将来任意的 key。
- 撤回 R1 ~ R4 只是作者的变异证据，审查方没有独立撤回。
- 审查方本机读取 WSL 时得到 `Wsl/E_ACCESSDENIED`，没有启动 Linux 实验；Linux 八项故障的旧 / 新对照来自独立读取的 CI 日志，不是本机复现。

**门槛的状态**（Review 097 的裁定；§14.10、§14.14、§14.17 是之前的登记，保留原文）：

| 门槛 | 状态 |
|---|---|
| G3 | 局部支持：共享区间的错误传播恢复可信；P-1 的真实 Cmd / 句柄移交、F1 的返回、释放之后的持续禁用得到局部支持。Windows `alive` 另持 Job 句柄的受控构造改变了最后句柄关闭的前提；不证明真实的不可终止故障；宿主的生产拓扑与资源接线仍须兑现 |
| G6 | 局部支持：P-2 的九组证据（过滤器、能力、登记、回收、SET 0 被拒与属性保留）继续适用，不经 `exclusive`。边界：受测的内核与架构、seccomp 前提、N-096-2 的分类边界；实验进程退出不算清除成功；生产探测的接线没有验证 |
| G7 | 局部支持：P-4 九类各两次、恢复的知识状态、退出码、期限与移交。真实 API 结果、改变的前提与接缝分开；不推出任意 Windows 宿主、真实的不可终止情形或生产事务 |
| G5 | **没有满足**：P-3 没有执行，不自动缩小 macOS 的支持 |
| G1、G12 与其余 | 既有决定与模型规则不变；模型的零访问与记录不是生产准入、数据库、`seq`、事务、CLI 或完整持久化组合的证据 |

**M-1**：§14.16 中常规 CI run 40656041186 的 Windows 失败，按 Review 097 §7 只确认与 M-1 是同一测试、相似的错误签名（§14.16 已就地收窄），不确认同根因；已补入 M-1 的观测登记（[m1-exit-evidence.md §9](m1-exit-evidence.md)）。M-1 未解决。

**收尾的范围**：相对受审 HEAD `800b5e8` 只有文档；原型代码仍与 `48ff0cd` 相同，不为纯文档收尾重跑原型诊断。技术 GO 不是规范冻结、产品批准、生产授权或 WP2.5 / M2 验收。

### 14.19 N-096-2：P-2 自身结果的分类（修复）

**背景**：

- [Review 096](reviews/codex-review-096.md) 登记了两项非阻塞意见（§14.17）；[Review 097](reviews/codex-review-097.md) 维持非阻塞（§14.18）。
- PR #38 合并之后，负责人授权从核验成功的 master 另开独立小 PR 修复（review-resolution §118）。
- 两处问题：
  - **unsupported 的分类顺序**：过滤器装不上时，实验进程先发布 unsupported、不创建检查；外层却先等 10 s 检查的 PID 记录，得到 invalid，走不到 unsupported。
  - **自读的 unknown**：探测把读自身 `/proc` 条目得到的 unknown 压成 `/proc=false`，之后得到 refuted，而不是保存读取失败。
- 两者都不会得到 confirmed。

修复只改原型：P-2 的外层与实验进程，以及报告协议中登记等待的一个入口。规范、生产代码、Schema、迁移、根 `ci.yml` 与已批准的产品语义都不改。不增加宿主禁用条件、重试或恢复政策。

**修复**：

- **登记等待可以因报告结束**：`awaitOwnedUnless(path, n, d, own, ended)` 是 `awaitOwned` 的扩展。`awaitOwned` 调用它、`ended` 为 nil，行为不变。
  - 每一轮先读取并登记已到的记录，再询问 `ended`。
  - 提前返回时，没有登记的仍记为短缺，直到调用者核验它从未创建。
- **P-2 等登记时**，同时读取实验进程的报告。只有报告存在、能解析、而且声称 unsupported 时，等待才结束：
  - 报告不存在：继续等，超时仍是 invalid；
  - 读取失败或不能解析：invalid；
  - 不声称 unsupported 的报告不结束等待。
- **每个声称 unsupported 的报告都要核验**，不论在等待中读到、还是登记之后读到（`p2NoCheck`）。核验的内容：
  - 报告属于本实验进程、本组（既有的检查）；
  - 没有任何步骤、没有检查、没有安装之后的任何结果，报告的退出码为 0；
  - 外层自己看到：检查的报告文件中没有任何记录，实验进程没有子进程。读取失败记为观察失败，读不到的不当作空。
- **核验通过**之后的处理：
  - 声明该报告文件应为空（`noneExpected`），短缺清零；`done` 在写者可能已结束之后，仍按“没有任何记录”严格检查它；
  - 之后得到 unsupported。unsupported 不计为机制的 confirmed。
- **核验不通过**之后的处理：
  - 记为观察失败（`s.flag`），运行继续；
  - 因此已启动的检查仍被判定、释放与清理；
  - 报告文件仍按原声明（1 条记录）检查。
- **自读 unknown**：
  - 实验进程在报告中保存 `self_read_error`，`/proc` 步骤带上错误；
  - 外层把它记为观察失败；
  - 探测照常拒绝。不改探测的语义，不重试。

**分类的传播**：

| 情况 | 实验进程的报告 | 修复之前 | 修复之后 |
|---|---|---|---|
| 过滤器装不上，预期有检查、而且安装过滤器的组（`prctl.set0`；`normal` 对照不安装过滤器，只预期一条检查登记） | unsupported，没有步骤、没有检查，退出 0 | 先等 10 s 的登记 → invalid，走不到 unsupported | 等登记时读到报告即停止；核验通过 → 报告文件声明为空 → unsupported |
| 过滤器装不上，没有检查的组 | 同上 | 读报告 → unsupported，不核验 | 读报告 → 同一核验 → unsupported |
| 报告缺失（不写） | — | 登记超时 → invalid | 不变：没有报告就不结束等待 → invalid |
| 报告畸形（截断） | 不能解析 | 登记超时 → invalid，理由不点名报告 | 等待中读到 → invalid（does not parse） |
| 他人的报告（PID 不符） | PID 为父进程 | 登记超时 → invalid | 等待中读到 → invalid（a report of pid …） |
| 探测之后声称不支持 | unsupported，另有 7 个步骤与过滤器状态 | 登记超时 → invalid | 核验不通过 → flag → invalid |
| 已启动检查之后声称不支持 | unsupported，另有 16 个步骤、检查及其结果；检查的报告文件有 1 条记录 | 检查已登记、清理，但判为 **unsupported / 0**，绕过了判定 | 核验不通过 → flag；继续判定、释放与清理 → invalid |
| 自读 unknown（`prctl.set1`） | `/proc` 步骤成功、值 false；探测拒绝 | **refuted** | 报告保存读取失败，步骤带错误，外层 flag → invalid；探测照常拒绝 |
| 自读 unknown，有检查的组 | 探测拒绝，没有检查 | 登记超时 → invalid | 不变（见下） |

**不适用或不变的路径**（不制造对应的测试）：

- **有检查的组中的自读 unknown**：
  - 探测在检查之前拒绝，报告不是 unsupported，所以外层仍等 10 s 的登记，得到 invalid，理由是登记超时。
  - 读取失败保存在报告中。等待中的回调仍读取并解析报告，寻找 unsupported；但报告不声称 unsupported，等待不结束，进不到之后消费 `self_read_error` 的判定分支（Review 098 N-098-1）。
  - 这是保守的 invalid，不是 confirmed。本轮按授权只让经核验的“不支持”结束等待，不扩大到其他提前结束；那会改变机制拒绝的分类。
- **“之后成功”**：
  - P-2 中自读只在探测中出现一次；`prctl.set0` 的第二次探测在 GET=1 处先拒绝，到不了自读。“同一读取之后又成功”在 P-2 中没有路径。
  - 用例中，自读失败之后的读取都成功：结束时的 GET，外层对状态、子进程与描述符的读取。运行仍为 invalid，理由点名自读的失败。
- **没有检查的组中的注入**：门禁用例只在 `prctl.set0` 上注入。没有检查的组走同一核验函数，不另设用例。
- **原生的“装不上”**：
  - CI 与作者本机都装上了过滤器，`p2Install` 的真实失败与缺少审计架构都没有出现。
  - 有效的 unsupported 只由注入产生，验证的是分类路径，不是“过滤器在某个环境中确实装不上”。
- **macOS、Windows**：P-2 只在 Linux 上。

**注入与实际状态**：

| 故障 | 实际执行的部分 |
|---|---|
| `p2-unsupported-once` 与 withheld、truncated、foreign 三种变体 | 不执行真实的安装（命中记录写明）：实验进程没有过滤器、没有步骤、没有检查。三种变体之后分别不写报告、截掉报告的最后一个字节、以父进程的 PID 写报告 |
| `p2-unsupported-after-probe-once` | 过滤器真实安装，探测真实执行（7 个步骤），之后什么都不做 |
| `p2-unsupported-after-check-once` | 真实的检查已启动、登记、观察并回收；之后不执行清除 SET 0，属性在实验进程退出之前保持设置。实验进程是隔离的，它的退出不算清除 |
| `p2-self-read-unknown-once` | 真实读取照常执行（命中记录写明返回 present），只把结果替换为 unknown |

**三类事实分开记录**：

- **机制结论**：内层的判定。unsupported 不计为 confirmed。
- **证据有效性**：门禁是否通过、理由是否点名失败。
- **最终清理**：内层的清理、外层的补救、Final、保险清理。

补救或清理成功不恢复无效的证据，清理只作用于场景登记过的进程。逐项结果见 §14.20。

### 14.20 回归先行、修复、诊断与定向撤回（N-096-2）

**提交的顺序**（分支 `test/wp2.5-prototype-n-096-2`，从核验成功的 master `27ce69d` 开始）：

1. `e880902`：诊断 workflow 的分支列表增加本分支的精确匹配。权限、平台、预算、路径过滤与失败传播不变。
2. `e23a7ef`（回归先行，旧的分类处理没有改动）：
   - 实验进程中的故障接缝（§14.19 的注入表）；
   - 7 个 N-096-2 门禁用例与对照 `control.p2-set1`；
   - 测试工具：
     - 门禁用例可以要求 unsupported：退出码 0，不计为 confirmed；
     - 门禁结果也记录辅助进程自己的命中位置，前缀为 `helper: `。它与 Review 097 指出的补充字段一样只用于诊断：解析失败时不追加，不用来放宽判定。
3. `18bc6a4`（修复，**固定的原型代码点**）：§14.19 的修复。
4. 之后只有文档的提交。

**用例**（Linux；“旧”为 `e23a7ef`，“新”为 `18bc6a4`；每个故障都有命中记录）：

| 用例 | 组 | 旧实现的内层结论 / 退出码 | 新实现 | 性质 |
|---|---|---|---|---|
| `p2.unsupported-once` | `prctl.set0` | invalid / 1（登记超时） | **unsupported / 0** | 回归：有效的不支持 |
| `p2.unsupported-withheld-once` | 同上 | invalid / 1（登记超时） | invalid / 1（登记超时） | 新路径的防护，旧实现上也通过 |
| `p2.unsupported-truncated-once` | 同上 | invalid / 1，理由是登记超时 | invalid / 1（does not parse） | 新路径的防护 |
| `p2.unsupported-foreign-once` | 同上 | 同上 | invalid / 1（a report of pid …） | 新路径的防护 |
| `p2.unsupported-after-probe-once` | 同上 | 同上 | invalid / 1（contradicts it：7 个步骤、安装之后的结果） | 新路径的防护 |
| `p2.unsupported-after-check-once` | 同上 | **unsupported / 0**：检查已登记、清理，判定被绕过 | invalid / 1（contradicts it：16 个步骤、检查、检查的报告文件有 1 条记录） | 回归 |
| `p2.self-read-unknown-once` | `prctl.set1` | **refuted / 1** | invalid / 1：自读失败，`/proc` 步骤带错误 | 回归 |

- **新路径的防护**：截断、外来、矛盾三个用例在旧实现上已是 invalid，只因理由不点名报告本身（`because`）而失败，**不是**旧实现的假通过。它们防止新的提前结束把这类报告当作 unsupported。缺失报告的用例在旧、新实现上都通过。
- **对照**：在旧、新实现上都是 confirmed / 0。
  - `control.p2-normal`；
  - `control.p2-denied`（`waitid.EPERM`，计划的拒绝）；
  - `control.p2-set0`（SET 0 被拒）；
  - `control.p2-set1`（新增）。
- Review 095 的三个既有 P-2 用例保持 invalid / 1。

**机制结论、证据有效性与最终清理**（诊断 run 40691692731 的 Ubuntu 作业；作者本机的结论相同）：

| 用例 | 机制结论（内层） | 证据有效性（门禁） | 内层的清理 | 外层 |
|---|---|---|---|---|
| `p2.unsupported-once` | unsupported，不计为 confirmed | 通过 | verified：1 个进程（实验进程）已结束 | 补救 not-needed，Final=ended，保险清理 nothing left |
| withheld、truncated、foreign、after-probe | invalid | 通过（正确拒绝） | NOT verified：声明的 1 个检查从未登记；实验进程已结束 | 同上 |
| `p2.unsupported-after-check-once` | invalid | 通过 | verified：2 个进程（实验进程与检查）已结束 | 同上 |
| `p2.self-read-unknown-once` | invalid | 通过 | verified：1 个进程已结束 | 同上 |

内层的“NOT verified”来自严格的声明：没有通过核验的报告不能免除声明的检查，它按短缺计入。它不表示有进程遗留：外层普查没有发现本轮遗留的进程。

**定向撤回**（作者本机 WSL2，基于 `18bc6a4` 的树；每次只撤回一项，与 P-2 的全部门禁用例和对照一起运行；变异之后已恢复，源文件的 SHA-256 不变，没有提交）：

| 撤回 | 应检出的用例 | 结果 | 对照 |
|---|---|---|---|
| R1 登记等待不因报告结束 | 有效的不支持，以及截断、外来、矛盾 | 4 个都检出：有效的不支持退回登记超时（invalid），另三个的理由退回登记超时 | 全部通过 |
| R2 unsupported 不经核验 | 探测之后、检查之后 | 都检出：探测之后的报告得到 **unsupported / 0**；检查之后的报告由 `done` 的报告文件检查拒绝（“1 pid records, want 0”），理由不点名矛盾 | 全部通过 |
| R3 自读 unknown 再次被压成 false（两个消费者都去掉） | 自读 | 检出（refuted） | 全部通过 |
| R4 只去掉外层的 flag | 自读 | 检出：仍为 invalid，但理由只有 `/proc` 步骤的错误（`because`） | 全部通过 |
| R5 只去掉 `/proc` 步骤的错误 | 自读 | **没有检出**：外层的 flag 单独就使运行 invalid，并点名失败 | 全部通过 |
| R6 核验通过之后不声明报告文件为空 | 有效的不支持 | 检出：`done` 按原声明拒绝（0 条记录，要 1 条） | 全部通过 |

R2 也说明 `done` 的报告文件检查是第二道防线：即使没有核验，已登记的检查也不能被声明为不存在。

**作者本机**（作者证据）：

- `18bc6a4` 的树：
  - Windows 11 26300：完整原型套件为场景 53/53 confirmed、门禁回归 92/92、组合 13/13；
  - WSL2 6.18（不带 `-race`）：场景 52 confirmed，另有 1 unsupported（既有的 `T5.adapter.pwsh-argv`，没有 pwsh）；门禁回归 111/111，组合 13/13。
- P-2 的门禁用例与对照在 `e23a7ef`（旧）上运行一次，结论如上表。
- 在新树上运行三次：
  - 第一次：两个矛盾用例因修复中的消息格式不合预期的子串而失败。只改了消息格式，没有改回归的预期。
  - 第二次：其余都通过。但 `control.p2-set0` 的内层为 confirmed / 0，外层的保险普查读取 PID 641 的环境时得到 permission denied，记为外层工具错误，门禁失败关闭。
    - 之后该 PID 已不存在，它的归属与原因未知，不归因。
    - 这次失败保留，不以之后的通过回填。
  - 第三次：完整套件，全部通过。

**诊断 CI**（`.github/workflows/wp25-prototype.yml`；与常规 CI 分开；都没有重跑或取消）：

| run | 代码点 | 结论 | macOS 26.6.2 arm64 | Ubuntu（`-race`） | Windows 10.0.26100 |
|---|---|---|---|---|---|
| 40689865305 | `e23a7ef`（回归先行） | **failure**（预期） | success：场景 33/33，门禁回归 77/77，组合 13/13 | failure（6.17.0-1022-azure）：场景 53/53，门禁回归 105/111，组合 13/13 | success：场景 53/53，门禁回归 92/92，组合 13/13 |
| 40691692731 | `18bc6a4`（修复） | **success** | 场景 33/33，门禁回归 77/77，组合 13/13 | 7.0.0-1017-azure：场景 53/53，门禁回归 111/111，组合 13/13 | 场景 53/53，门禁回归 92/92，组合 13/13 |

- **检出**：两个 run 的三个作业都实际取得并检出各自的完整 SHA，WP25META 声明相同的 SHA。Ubuntu 作业按既有的 `RUNNER_OS` 判断传入 `-race`。
- **40689865305**：Ubuntu 失败的门禁用例正是上表中旧实现上失败的 6 个，结论与本机相同。Windows、macOS 没有 P-2，没有新增的用例。
- **40691692731**：Ubuntu 上 P-2 的 15 个门禁用例都通过，P-2 的 9 组都是 confirmed。
- **内核**：两个 run 的 Ubuntu 内核不同（runner 镜像的差异）。

**受影响的证据**：

- **§14.11 中 P-2 的两条概括**：
  - 原文是“计划外的观察失败使运行 invalid”与“过滤器装不上 → unsupported（不变）”；
  - 在 `18bc6a4` 上按 §14.19 的范围成立：经核验的不支持 → unsupported；自读 unknown → invalid（有检查的组中，经登记超时得到 invalid）；
  - **待独立审查**。
- **之前的 P-2 结论**（`85ec2ca` 的 run 40643975558、`48ff0cd` 的 run 40658194158）：
  - 九组都装上了过滤器、自读为 present，没有走到改动的分支，所以结论不因本修复而降级；
  - 新代码点的证据是 run 40691692731。
- **共享的改动**：
  - `awaitOwned` 改为调用 `awaitOwnedUnless(…, nil)`，行为不变；
  - 门禁的 unsupported 退出码规则只影响要求 unsupported 的用例（只有新增的一个）；
  - 辅助进程的命中位置只用于显示；
  - 其他场景与门禁由同一诊断在新代码点重新运行。

**门槛的状态**（作者的登记，待独立审查；§14.18 是 Review 097 的裁定，保留原文）：

| 门槛 | 状态 |
|---|---|
| G6 | N-096-2 的两处分类边界在原型中已修复，**待独立审查**。有效的 unsupported 只由注入产生，没有原生的“装不上”；只有 x86_64。仍只有局部支持，不宣布满足 |
| G5 | **没有满足**（本轮不补做 P-3） |
| G3、G7 与其余 | 不变（§14.18） |

**固定点之后**：之后只有文档的提交，`git diff 18bc6a4 HEAD -- prototype .github` 为空（最终核对见 PR 描述）。

### 14.21 Review 098 的结论与收尾

[Review 098](reviews/codex-review-098.md) 对 `8b6aca1`（固定的原型代码点 `18bc6a4`）为**技术 GO**（review-resolution §119）：

- **N-096-2 的两处问题关闭**，限定在受审的受控原型范围内：
  - 经核验、尚未探测或启动检查的“不支持”得到 unsupported；
  - 自读 unknown 保留为工具的观察失败，不再按可靠的能力不满足得到 refuted。
- 没有新的 BLOCKER、confirmed 的假通过或直接回归；Review 095 B1 维持 Review 097 的关闭范围。
- 共享的报告协议与门禁的改动成立：
  - 原来调用 `awaitOwned` 的地方都走 `ended` 为 nil 的严格规则；
  - unsupported / 0 只匹配明确要求这一分类的门禁用例。
- **N-098-1**（非阻塞，两处措辞）已在 §14.19 就地收窄，不改语义：
  - 分类表的第一行原写“过滤器装不上，有检查的组（`normal`、`prctl.set0`）”。`normal` 对照不安装过滤器，原生的安装失败只会发生在有过滤规则的组；该行现在只写 `prctl.set0`，`normal` 只作为“预期一条检查登记”的组说明。
  - 原写“这条路径上外层不读报告”。实际上等待中的回调仍读取并解析报告，寻找 unsupported；只是进不到之后消费 `self_read_error` 的判定分支。现在按后者写。

**证据的范围**（照审查原文，补充 §14.19、§14.20）：

- **R5 没有检出**：只去掉 `/proc` 步骤的错误时，外层的 flag 单独就使运行 invalid，并满足理由。两个消费者在终判上是冗余的。
  - R3 同时去掉两者之后变成 refuted，支持“两者共同必要”的分类性质；但这不等于分别证明每个字段都有独立的防回归断言。
  - `/proc` 步骤错误的独立输出还没有同等强度的回归保护。这是测试证据的限制，不是已发现的分类漏洞。
- **只有注入，没有原生证据**：有效的 unsupported 只由注入产生，没有原生的安装失败或缺少审计架构。
- **测试环境**：只有现有的 x86_64 与已测的内核；两个 Ubuntu 作业的内核不同（6.17.0-1022-azure、7.0.0-1017-azure），不是同一原生环境的前后对照。
- **其他提前拒绝**：在有检查的组中，其他提前拒绝仍可能先在登记等待中超时。这是原因的精度限制，结果是保守的 invalid，不形成 confirmed。
- **报告的发布与解析**：
  - 发布与可见性只在受控的 Linux 临时目录中核对：`.tmp` 写完之后在同一目录改名，父端只读已发布的文件；
  - 不新增断电持久性、任意写者认证或所有文件系统的原子性保证；
  - `json.Unmarshal` 不是拒绝未知字段的 Schema，不是任意不可信 JSON 的完整验证器。
- **内层与外层分开**：
  - 内层的“NOT verified”与短缺按严格的声明保留；外层另行核对实验进程已结束、没有本轮遗留；
  - 不能把外层的结束证据改写成内层登记完整，也不能把声明的短缺一律解释为真实遗留。
- **审查方的本机条件**：
  - 读取 WSL 时得到 `Wsl/E_ACCESSDENIED`，没有启动 Linux 实验；
  - 新旧行为的依据是源码与完整的 CI 日志；
  - 没有本机复现 P-2 的故障、原生的安装失败或作者的 R1 ~ R6，macOS 也没有本机动态运行。
  - 审查方的本机运行：Windows 上十项既有的共享报告门禁与一个根层场景，首次即通过（19.928 s）。
- **作者证据**：R1 ~ R6、新树上第一次的消息子串不匹配、第二次 PID 641 的 permission denied 都只是作者证据，保留原样。PID 641 的归属、原因与当时的完整状态未知。
- **PR #38 首轮的 Windows**：
  - 审查方读取该日志同样得到 404；
  - 失联注释按实施方保存的记录列示，审查方没有独立取得注释的内容；
  - 测试结果、实际检出与失败原因仍未知，之后的绿色不回填。

**门槛的状态**（Review 098 的裁定；§14.18、§14.20 是之前的登记，保留原文）：

| 门槛 | 本轮增加的支持 | 仍不能推出 |
|---|---|---|
| G6 | 受控 Linux 原型中，经核验的安装之前的 unsupported 能正确退出；自读 unknown 不再当作能力的反证；矛盾、缺失、损坏与外来身份的报告继续被拒绝 | 有效 unsupported 只有注入证据；只有现有 x86_64 与已测内核；其他提前拒绝仍可能先登记超时；不是生产能力探测的全路径。**不宣布满足** |
| G3 | 维持既有的所有权、独立的外层普查、补救与终判 | Windows `alive` 保持 Job 句柄的构造限制、真实的不可终止故障、任意宿主与生产接线 |
| G7 | 维持既有恢复异常、模型与错误传播的局部支持 | 没有新增 Windows 恢复实验；不证明生产恢复、事务或所有原生失败 |
| G5 / P-3 | 没有新增 | **没有满足**，P-3 没有执行；不因本轮的绿色自动缩小支持平台 |

**收尾的范围**：相对受审 HEAD `8b6aca1` 只有文档；原型与 workflow 仍与 `18bc6a4` 相同，不为纯文档收尾重跑原型诊断。技术 GO 不是规范冻结、产品批准、生产授权或 WP2.5 / M2 验收。

## 15. P-3：macOS 的 ABI 与 x86_64 的最小原型（2026-10-29 起实施）

### 15.1 授权、固定的身份与计划的版本

- **授权**：负责人 2026-10-29 的条件授权（review-resolution §120）；[Review 100](reviews/codex-review-100.md) 明确允许实施（§123）。另开原型 PR，不合并，不宣布 G5 满足。
- **固定的身份**：
  - base：核验之后的 master `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b`（PR #40 的合并）；
  - 受审的文档提交：`344cbcae2f2ac0903c1a1052a0266cbb68cd03ef`（Review 100 的对象）；
  - 计划的版本：规范第 8 版第二稿 §19.5、§19.12，以 `b359b0cb02848434e3c38475b42f157dff1b9014` 的内容为准（Review 100 收尾时只做了不改变语义的措辞精化，随 PR #40 合并进 `dbcd5d4`）。
- **本节在写任何原型代码之前提交**。实施若须偏离本节，先在这里登记偏离与理由，再改代码。不为取得绿色改动预测。

### 15.2 范围

- 只改 test-only 的 `prototype/wp25`，以及诊断 workflow。workflow 只增加三项：
  - 本分支 `test/wp2.5-prototype-p3` 的精确匹配；
  - 标准的 `macos-26-intel`；
  - 受控的环境元数据。
- 既有平台、只读权限、失败传播、作业 20 min 与 `go test` 15 min 的期限都不变。不增加 `macos-15-intel`、larger runner、root、付费资源或工具安装。
- 不改生产代码、根依赖、Schema、迁移或根 `ci.yml`。
- 只验证 E1（直接陷入），不新增 E2 的原型，不选择生产调用入口。

### 15.3 实施的落点（按 §19.12 写定，不改变其语义）

**被测的解码器 M**（`tree_darwin_test.go`，darwin）：

- waitid：
  - 原始读取 `readWaitid` 返回前六个字段（按 `uint32`）；
  - 解码 `decodeW` 依次检查 W1、W3a、W4，按 W3b（Q-3 (a)）取值；W2 由按 `uint32` 读取实现；
  - 读两次时，第一次的解码通过之后按 W5 比较两次的全部六个字段。
- kqueue：
  - 原始读取 `kqWaitRaw` 返回事件的 `ident`、`filter`、`flags`、`fflags` 与有符号的 `data`；
  - 解码 `decodeK` 按 K1、K2。
- C1：比较两条路径规范化之后的结果。
- M 以第一条拒绝的规则为结论，之后的读取不改变它（C-19）。
- 事件的 `ident` 不是所查的进程、或 `filter` 不是 `EVFILT_PROC`，是调用本身的错误（观察失败），不属于 K 规则。
- 既有的 `waitidNoWait` 与 `exitObserver.wait` 改为经这些解码，拒绝时返回错误（观察失败）。
  - 所以既有的 macOS 场景（T3、P-1）也受新的核对约束，须在同一个固定代码点重新取得证据；之前代码点的 macOS 证据降为历史观测。
  - `waitidNoWait` 在注册时已退出的路径上只读一次。

**独立的规范化 N**（`p3_darwin_test.go`）：

- 按 §19.5 的规则另行实现，不调用 `decodeW`、`decodeK`：W1、W3a、W4 与 K1、K2 不成立时为“无”。

**场景**：

- `TestP3` 的子测试按声明顺序运行：先 `layout.1` ~ `layout.3`，再其余场景。
- 场景 ID 为 `P3.<路径>.<名字>.<n>`，子测试名为 `<路径>.<名字>.<n>`。
- 每次运行的证据另以 `WP25P` 行输出，在任何补救之前打印：
  - R、N、M、F、E 的各项，以及到那时为止的工具失效；
  - 最终的 T、C 与 V 在 `WP25RESULT` 中：`invalid` 的全部理由、`cleanup` 与 `verdict`。

**前提的判定**（每次运行开始时）：

- **原生架构**：
  - `uname -m` 由 `uname(2)` 读取（不启动进程），另记录 `hw.machine`；
  - `GOARCH` 为 amd64 时须 `uname -m` 为 `x86_64`，arm64 时须为 `arm64`；
  - `proc_translated` 为 1，或 `uname -m` 与 `GOARCH` 不符 → E 不具备；
  - `proc_translated` 为 0 或不存在（`ENOENT`）→ 原生；其他读取错误 → T 失效。
- **布局前提**（`layout` 以外的场景）：只在同一进程中已运行的布局场景都为 confirmed 时成立。
  - 都没有运行 → T 失效；
  - 有 refuted 或 invalid → T 失效；
  - 都为 unsupported → E 不具备。
  - 所以门禁的内层运行同时选中 `layout.1`。
- **可靠确认没有 Xcode / SDK**：只有在 `PATH` 中找不到 `xcrun`。`xcrun` 本身的失败与无法取得 SDK 都无法与工具失效可靠区分，按 T 失效（invalid），不按 unsupported。这比“没有 Xcode 或 SDK 为 unsupported”保守，不会产生通过。

**观察改写类的接缝**（只在门禁的内层运行中按 `WP25_INJECT` 生效；`readWaitid` 与 `kqWaitRaw` 读完真实结果之后，经 P-3 设置的钩子改写缓冲区或事件）：

- 改写的值：
  - `p3.w-sipid-once`：第一次读取的 `si_pid` 改为 `pid + 1`；
  - `p3.w-reread-once`：第二次读取的 `si_status` 改为 8，在 W1 ~ W4 下合法，只有 W5 能检出；
  - `p3.k-stopped-once`：`data` 改为 `0x117f`（被 SIGSTOP 停止）；
  - 其余按规范 §19.12 的表。
- 命中记录：`WP25INJECT` 行的 `site` 写明读取的序号、字段、真实值与改写之后的值。
- 接缝模式下，场景不做原生的预测比较：
  - M 以预期的规则拒绝 → `s.flag("planned corruption detected by <规则>: …")`（invalid）；
  - M 接受，或以其他规则拒绝 → `s.expect` 不符（refuted）；
  - 没有命中 → T 失效。

**计划的故障 `group.enum-unknown`**（不由 `WP25_INJECT` 控制，在主诊断中运行）：

- P-3 自己的确认循环中，第一次 `kern.proc.pgrp` 的读取先真实进行，再以一个注入的错误（不是 `ENOMEM`）代替结果。按 §19.5 的枚举预算，这是“未知 → 不确认结束”，不重试。
- 命中写入 `WP25P` 的 F。之后钩子关闭，清理用既有的确认（读不到即为 invalid）。
- 不调用既有的 `confirm` 来判定故障的响应：它把读取失败直接记为 invalid。

**更严格的门禁**：

- 场景的结果（`WP25RESULT`）增加 `invalid` 字段，列出全部 invalid 的理由。
- 门禁用例增加 `exact`，要求：
  - 理由恰好一条，并且包含预期的检出原因；
  - 内层场景的清理以 `verified` 开头；
  - 同一次内层运行中的其他场景（`layout.1`）都为 confirmed。
- 外层的既有条件照常。
- 内层运行含两个场景，所以登记的核对改为合并这次内层运行全部结果的 `owned` 与 `shortfall`。只有一个场景时与原来相同。

**工具进程**：

- `xcrun`、Clang 与探针经 `start` 直接启动，由场景登记（`childProc`）。环境中带本次运行的标记 `WP25_P3TOOL=<随机值>`；门禁的内层运行中，它们也继承见证的标记。
- 输出写到临时文件，不经管道。
- 每一步结束之后，按本次的标记普查（`censusTag`）。Clang 内部的后代也由普查找到：
  - 找到存活的、或普查失败 → T 失效；
  - 对找到的进程，只在标记与启动时间都核实时结束它们，归属未确认的零动作；补救不恢复有效性。
- `xcrun --find clang`、`xcrun --sdk macosx --show-sdk-path` 与 `--show-sdk-version` 各一次；之后 `clang -arch <arm64 | x86_64> -isysroot <SDK>` 编译，再运行探针。退出码不为 0 → T 失效。

**探针的键集**：

- `siginfo_t` 的大小与六个字段的偏移、大小；
- `struct kevent` 的大小与六个字段的偏移、大小；
- `struct kinfo_proc` 的大小，与 §19.12 所列字段的偏移；
- §19.12 所列的常量。

比较的来源：

- `unsafe.Sizeof` / `Offsetof`；
- x/sys 的常量；
- x/sys 没有的常量（`P_PID`、`CLD_*`、`SZOMB`、`P_WEXIT`、`KERN_PROC_*`、`KERN_PROCARGS2`）：与原型自己的常量，或规范 ABI 表登记的值比较；
- `sizeof(siginfo_t)` 与 §19.5 的 104 比较，另检查不大于 128（只证明容量）。

值按键的符号与位宽解析，并须是规范的十进制写法。

**期限**：

- 每次运行 60 s：内层 ≤ 20 s，外层核验 ≤ 10 s，清理 ≤ 20 s，余量 10 s。各段超出 → T 失效。
- 工具各步的期限为 `min(单步上限, 内层剩余, 到 t.Deadline() 的剩余减 60 s)`。单步上限：`xcrun` 10 s，编译与链接 30 s，探针 5 s。
- **准入**：每次运行开始之前检查两项，否则记为未执行（invalid）：
  - P-3 的剩余预算 ≥ 60 s；
  - 到 `t.Deadline()` 的剩余 ≥ 120 s。
- P-3 的总预算 15 min 计 `TestP3` 的 63 次运行。门禁的内层运行在各自的进程中，按既有的门禁期限（内层 150 s、外层 180 s），另行计数。

**环境元数据**：

- WP25META 在 darwin 上加入 `kern.version`、`hw.machine`、`uname -m` 与 `proc_translated`。读到的值照实记录，或记为“不存在”“未知”。
- workflow 在 macOS 作业的原型步骤之前加一个只打印的步骤：
  - 记录 `sw_vers`、`uname -a`、`sysctl` 的几项、`xcodebuild -version`、`xcrun --show-sdk-version` 与 `--show-sdk-path`、Clang 的版本、`go env GOARCH GOHOSTARCH`；
  - 每条命令失败也照实打印，不影响作业；
  - 只记录，不参与判定。

### 15.4 场景、构造与预测（每个场景 3 次）

- 子进程都是本测试二进制的 helper，在自己的进程组中启动，由场景登记。`exit-after <ms> <code>` 以 `os.Exit(code)` 退出。
- 原始值的预测依据规范 §19.5 的源码推演；N 按 Q-3 (a)；M 须等于 N。
- 回收之后的 `ExitCode()` 按 §19.5 推演表的最后一列。

| 场景 | 构造 | R 的预测 | N（M 须相同） |
|---|---|---|---|
| `layout` | `xcrun` 取得 Clang 与 SDK，编译、运行探针，严格解析，与 Go 侧逐项比较 | 键集完整、每键一次，全部与 Go 侧相同 | — |
| `k.exit7` | `exit-after 200 7`；启动之后立即注册 kqueue，等事件（≤ 3 s） | 返回的 `fflags` 含 `NOTE_EXIT \| NOTE_EXITSTATUS`；`data` = `0x0700`；之后为本进程未回收的僵尸；回收之后 `ExitCode()` 为 7 | （退出，7） |
| `k.signal9` | `sleep`；注册之后有期限地等 300 ms，之后 `SIGKILL` | 300 ms 内没有事件；之后 `data` 的低 7 位为 9；僵尸；回收之后为信号 9 | （被信号终止，9） |
| `k.exit-100` / `-107` / `-ff` / `-7f` / `-80` | `exit-after 200` 加 256 / 263 / −1 / 2130706439 / −2147483648 | `data` = `0x0000` / `0x0700` / `0xFF00` / `0x0700` / `0x0000`；僵尸；`ExitCode()` 为 0 / 7 / 255 / 7 / 0 | （退出，0 / 7 / 255 / 7 / 0） |
| `k.esrch` | `exit-after 0 7`；确认为本进程未回收的僵尸之后，以候选的观察器注册 kqueue | 注册得到 `ESRCH`（推理的预测）；候选转入 waitid，再读第二次，同 `w.exit7` | （退出，7） |
| `w.exit7` | `exit-after 0 7`；确认为僵尸之后直接 waitid 两次（不注册 kqueue） | `si_signo` 20、`si_pid` 等于子进程、`si_code` 1、`si_status` 7；两次相同；仍为僵尸；回收之后为 7 | （退出，7） |
| `w.signal9` | `sleep`；`SIGKILL`；确认为僵尸之后 waitid 两次 | `si_code` 2、`si_status` 9；两次相同；回收之后为信号 9 | （被信号终止，9） |
| `w.exit-100` / `-107` / `-ff` / `-7f` / `-80` | `exit-after 0` 加上面的值；确认为僵尸之后 waitid 两次 | `si_status` = `0x100` / `0x107` / `0x00FFFFFF` / `0x7` / `0x0`；`si_code` 1；两次相同；`ExitCode()` 同上 | （退出，0 / 7 / 255 / 7 / 0） |
| `c.exit7` / `c.exit-107` / `c.exit-ff` | `exit-after 200` 加 7 / 263 / −1；注册 kqueue，取得 K；确认为未回收的僵尸之后 waitid 一次，取得 W | K 的 `data` 与 W 的 `si_status` 分别如上（`-107`、`-ff` 的原始值不同） | K 与 W 都为（退出，7 / 7 / 255），C1 相同 |
| `group.zombie-leader` | 组长 `member-parent`；成员报告之后放行，组长以 0 退出 | 组长为未回收的僵尸；不含僵尸的成员只有那个成员；结束成员并确认之后组为空；最后回收组长 | 确认结束 |
| `group.enum-unknown` | 同上；P-3 的确认循环中第一次 `kern.proc.pgrp` 的读取被注入为失败 | 命中 1 次 | 不确认结束；之后钩子关闭，清理照常 |

- 合计 21 个场景 × 3 次 = 63 次运行，每个 macOS 作业相同。
- 门禁回归与对照（只在 darwin 上；内层的选择为 `TestP3/^(layout.1|<基于的场景>.1)`）：
  - 10 个接缝用例：
    - waitid：`p3.w-sipid-once`、`p3.w-signo-once`、`p3.w-code-once`、`p3.w-highbyte-once`、`p3.w-negative-once`、`p3.w-reread-once`；
    - kqueue：`p3.k-nostatus-once`、`p3.k-range-once`、`p3.k-stopped-once`；
    - 两条路径：`p3.c-mismatch-once`。
  - 预期的规则见规范 §19.12 的表（`p3.w-negative-once` 由 W3a 拒绝）；每个都按 `exact`，`because` 为 `planned corruption detected by <规则>`。
  - 3 个对照：`control.p3-k-exit7`、`control.p3-w-exit7`、`control.p3-c-exit7`，都为 confirmed / 0，同样要求 `layout.1` 为 confirmed。

### 15.5 判定、命中与清理

- **判定**按规范 §19.12 的优先级：
  1. 工具失效 → invalid；
  2. 环境不具备 → unsupported；
  3. 观察改写被预期规则检出 → invalid（由门禁裁定）；
  4. 没有检出 → refuted；
  5. 机制故障类按预测；
  6. 原生运行按预测为 refuted 或 confirmed。

  对应到原型：工具失效以 `s.flag` 记入，不清除；预测以 `s.expect` 比较；E 以 `s.unsupported`。`done` 的既有优先级为 invalid、unsupported、refuted、confirmed，与上面一致。
- 解码器对原生值的拒绝只作为 M 与预测比较，不记为工具失效。
- 工具失效、清理没有核实或结束未知、计划的故障没有命中，都不得到 confirmed；找到预期的理由不豁免其他失败。
- **命中**：每个接缝与计划的故障都有独立的命中记录（真实值、改写的位置）；门禁要求命中。
- **清理**：
  - 场景结束时，场景登记的进程都须核实已结束，工具的标记普查为空；
  - 先保存被测的结论（`WP25P` 行），再补救；补救不恢复有效性；
  - 外层门禁的既有条件（Final 为 ended、保险清理为 nothing left、归属核实）照常。
- **结果的报告**：
  - 诊断 CI 与常规 CI 分开；
  - 失败、取消、unsupported、未执行、未知与夹具缺陷都保留，不擅自重跑；
  - 不宣布 G5 满足。

### 15.6 写代码时登记的补充与偏离（在代码提交之前）

下面各项是写代码时发现的，在代码提交之前登记。它们都不放宽 §15.3 ~ §15.5 的判定，也不改变任何预测。

- **偏离 1：`uname -m` 的读取**。macOS 没有 uname 系统调用：x/sys 的 `Uname` 与 C 库的 `uname` 都按 sysctl（`CTL_HW`、`HW_MACHINE`）读取机器名。
  - 进程内记录两项：`uname.machine`（x/sys 的 `Uname`）与 `hw.machine`（按名字的 sysctl）。两者来自同一个 sysctl，**不是**互相独立的来源。
  - 架构的判定用 `uname.machine`，规则不变。作业中实际的 `uname -a` 由 workflow 的元数据步骤打印。
- **补充 1：纯函数的直接检查**（`p3_check_darwin_test.go`；只在 darwin 上运行，不计为场景，不是原生证据）：
  - `TestP3Decode`：M 与 N 在三类值上的结果，即 §19.5 的边界值、前提不成立时的值，以及每个接缝的改写值；
  - `TestP3Parse`：探针解析器拒绝以下输出：
    - 截断、空行、不是 `键=值` 的行；
    - 重复、未知与缺少的键；
    - 非规范的写法、超出位宽的值、负的无符号值。

    另检查真实的键集每键一次，探针每键打印一行。
  - 它们失败会使诊断失败。
- **补充 2：`k.esrch` 中候选没有走提前路径**（注册没有得到 `ESRCH`）：只记录注册的结果，不做 waitid 的读取；注册结果与预测不符，即为 refuted（规范 §19.12 的“会改变设计的结果”）。
- **补充 3：分组场景中组长的退出**由候选的 kqueue 观察器取得：
  - 它被解码器拒绝时，作为 M 与“`exit 0`”的预测比较，不记为工具失效；
  - 调用本身的错误仍为工具失效。
- **补充 4：各段的计时**：
  - “观察”从运行开始，到本场景的读取与解码结束；布局场景到探针运行结束；
  - “之后的核验”从那时到 `WP25P` 行；
  - “收尾”是 `done` 自身（运行 `defer`、清理与核实），超出 20 s 记为 invalid。
- **补充 5：工具步骤到期**：结束直接启动的进程并普查之后，停止本次运行（invalid）。
- **补充 6：门禁内层的选择**写作 `TestP3$/^(layout.1|<基于的场景>.1)`。§15.4 中的 `TestP3/…` 在第一层也会选中补充 1 的直接检查；它们不产生场景结果，但会在内层运行。

### 15.7 结果（固定代码点 `4e690d8`，诊断 run 40751152882）

**身份**：

- 固定代码点：`4e690d813b88af3005da6e6ca25d9cc851ee6f92`（`test(prototype): implement the minimal P-3 on macOS arm64 and x86_64`），本轮唯一的原型代码提交，父提交 `34342df`。
- 它之前的三个提交只有本文：
  - `acb25bc`：§15.1 ~ §15.5 的计划，写任何原型代码之前提交并推送；
  - `692f2ad`：§15.6 的补充 1 ~ 5；
  - `34342df`：§15.6 的补充 6。
- 之后的提交只有文档，原型与 workflow 与 `4e690d8` 相同（`git diff 4e690d8 HEAD -- prototype .github` 为空），诊断不会因它们重跑。
- 诊断 run 40751152882：push 触发，attempt 1，completed / success，四个作业都成功。没有取消、重跑或手动触发。

**四个作业的实际检出与环境**（每个作业的检出步骤 `git log -1 --format=%H` 都打印 `4e690d8…`，WP25META 的 `sha` 相同）：

| 作业（job ID） | 镜像 | 系统与内核 | 工具 | 结果 |
|---|---|---|---|---|
| `macos-latest`（123037855426） | `macos-26-arm64` 20261013.0424.1 | macOS 26.6.2（25G83），`xnu-12377.161.14~5/RELEASE_ARM64_VMAPPLE`；`hw.machine` 与 `uname.machine` 为 `arm64`；`proc_translated` 为 0 | Go 1.27.3 darwin/arm64；Xcode 26.6（17F113），SDK 26.5，Apple clang 21.0.0（clang-2100.1.1.101） | 96 个场景 96 confirmed（P-3 63，既有 33）；门禁 90/90（P-3 13）；组合 13/13；`TestP3Decode`、`TestP3Parse` PASS；295.7 s |
| `macos-26-intel`（123037855377） | `macos-26` 20260824.0517.1 | macOS 26.6.1（25G76），`xnu-12377.161.13~4/RELEASE_X86_64`；`hw.machine` 与 `uname.machine` 为 `x86_64`；`proc_translated` 不存在（元数据步骤：`unknown oid`；进程内 `ENOENT`） | Go 1.27.3 darwin/amd64；Xcode、SDK 与 Clang 同上 | 同上：96/96，门禁 90/90，组合 13/13，两项直接检查 PASS；307.5 s |
| `ubuntu-latest`（123037855444） | `ubuntu-24.04` 20261025.353.1 | Linux 7.0.0-1017-azure | Go 1.27.3，`-race` | 53/53；门禁 111/111；组合 13/13；472.9 s |
| `windows-latest`（123037854710） | `windows-2025-vs2026` 20261021.278.1 | Windows 10.0.26100 | Go 1.27.3 | 53/53；门禁 92/92；组合 13/13；356.0 s |

- 两个 macOS 作业都按 §15.3 判定为原生：机器名与 `GOARCH` 相符，`proc_translated` 为 0 或不存在。
- 两个内核都是 `xnu-12377.161.x`，**不是**规范固定的源码 `xnu-12377.121.6`；它们没有公开源码。
- arm64 的内核是 `VMAPPLE`（虚拟机）。

**场景的计数**（每个 macOS 作业相同）：

- P-3 场景 21 个 × 3 次 = 63 次运行，都为 confirmed：
  - `layout`：3；
  - `k.*`：8 个场景，24 次；
  - `w.*`：7 个场景，21 次；
  - `c.*`：3 个场景，9 次；
  - `group.*`：2 个场景，6 次。
- 63 行 `WP25P` 都没有工具失效（`T` 为空）。
- 63 个结果的清理都以 `verified` 开头：
  - 布局场景各 5 个工具进程（`xcrun` ×3、Clang、探针）；
  - 分组场景各 2 个进程（组长、成员）；
  - 其余各 1 个。
- 工具的标记普查一次也没有找到存活的进程，没有补救。
- 既有的 macOS 场景 33 个在新的解码器下都为 confirmed，门禁 77 个都通过（之前的代码点 `18bc6a4` 为 33 个场景、77 个门禁）。所以 T3 与 P-1 的 macOS 证据在 `4e690d8` 上重新取得，`18bc6a4` 的 macOS 证据降为历史观测。

**原始值、规范化与预测**（两个平台逐项相同，同一场景的 3 次运行 R、N、M 都相同）：

| 场景 | R（原始值） | N | M | 预测 |
|---|---|---|---|---|
| `layout` | 55 个键，每键一次；例：`siginfo.size` 104，`kevent.size` 32，`kinfo_proc.size` 648，`kinfo_proc.e_ucred.cr_uid.off` 420，`SYS_waitid` 173，`NOTE_EXITSTATUS` 67108864，`EVFILT_PROC` −5 | — | — | 55 项与 Go 侧相同，`siginfo_t` 不大于 128 字节：56 项都相同 |
| `k.exit7` | `fflags` `0x84000000`，`data` `0x700` | 退出，7 | 退出，7 | 相同；回收之后 `exit 7` |
| `k.signal9` | 300 ms 内没有事件（约 301 ~ 302 ms 返回）；之后 `data` `0x9` | 被信号终止，9 | 同 N | 相同；回收之后 `signal 9` |
| `k.exit-100` / `-107` / `-ff` / `-7f` / `-80` | `data` `0x0` / `0x700` / `0xff00` / `0x700` / `0x0` | 退出，0 / 7 / 255 / 7 / 0 | 同 N | 相同；回收之后 `exit` 0 / 7 / 255 / 7 / 0 |
| `k.esrch` | 注册得到 `ESRCH`；候选的 waitid 与第二次读取都为 `si_code` 1、`si_status` `0x7` | 退出，7 | 同 N（候选的提前路径） | 相同 |
| `w.exit7` | `si_signo` 20、`si_pid` 等于子进程、`si_code` 1、`si_status` `0x7`；两次相同 | 退出，7 | 同 N | 相同；仍为僵尸；回收之后 `exit 7` |
| `w.signal9` | `si_code` 2、`si_status` `0x9`；两次相同 | 被信号终止，9 | 同 N | 相同 |
| `w.exit-100` / `-107` / `-ff` / `-7f` / `-80` | `si_status` `0x100` / `0x107` / `0xffffff` / `0x7` / `0x0`；两次相同 | 退出，0 / 7 / 255 / 7 / 0 | 同 N | 相同 |
| `c.exit7` / `c.exit-107` / `c.exit-ff` | K 的 `data` `0x700` / `0x700` / `0xff00`；W 的 `si_status` `0x7` / `0x107` / `0xffffff` | K 与 W 都为退出，7 / 7 / 255 | 同 N；C1 相同 | 相同 |
| `group.zombie-leader` | 组长的事件 `data` `0x0` | — | 组长退出 0；确认结束 | 相同；组长到回收为止都是僵尸 |
| `group.enum-unknown` | 同上；计划的故障命中 1 次（真实的读取得到 1 个存活的成员，没有错误，结果被注入的错误代替） | — | 不确认结束 | 相同；之后清理照常确认了组的结束 |

- **内核宏分支的区分值**：`0x107` 在 waitid 上读到 263，在 kqueue 与 `wait4` 上为 7；`-ff` 在 waitid 上读到 `0x00ffffff`，最高字节为 0。这与 §19.5 按内核分支的推演一致，也说明 P1 ~ P4 下最高字节已经移走。`0x7f000007` 读到 7，没有最高字节。
- **原始值不同、规范化相同**：例如 `c.exit-107` 中 K 为 `0x700`、W 为 263，规范化都为 7。这是预测的现象（§19.12）；原始值仍逐项与推演比较，也都相同。
- **预测之外的观察**（照实记录，判定与 N 都不依赖它们）：
  - 每次 waitid 读取的 `si_errno` 与 `si_uid` 都为 0；本轮没有记录 runner 用户的 uid，不据此推断内核是否写入 `si_uid`；
  - kqueue 事件返回的 `flags` 为 `0x8031`。

**门禁回归与对照**（每个 macOS 作业都 13/13 通过）：

| 用例 | 命中（真实值 → 改写之后的值） | 内层 | 理由（唯一的一条） |
|---|---|---|---|
| `p3.w-sipid-once` | waitid 第 1 次读取：`si_pid` 改为加 1 | invalid / 1 | `planned corruption detected by W1: W1: si_pid … is not …` |
| `p3.w-signo-once` | 第 1 次：`si_signo` `0x14` → `0x0` | invalid / 1 | `… by W1: W1: si_signo 0 is not SIGCHLD` |
| `p3.w-code-once` | 第 1 次：`si_code` `0x1` → `0x4` | invalid / 1 | `… by W1: W1: si_code 4 is not …` |
| `p3.w-highbyte-once` | 第 1 次：`si_status` `0x7` → `0x1000007` | invalid / 1 | `… by W3a` |
| `p3.w-negative-once` | 第 1 次：`si_status` `0x7` → `0xff0000ff` | invalid / 1 | `… by W3a` |
| `p3.w-reread-once` | 第 2 次：`si_status` `0x7` → `0x8` | invalid / 1 | `… by W5` |
| `p3.k-nostatus-once` | `fflags` `0x84000000` → `0x80000000` | invalid / 1 | `… by K1` |
| `p3.k-range-once` | `data` `0x700` → `0x10000` | invalid / 1 | `… by K1` |
| `p3.k-stopped-once` | `data` `0x700` → `0x117f` | invalid / 1 | `… by K2` |
| `p3.c-mismatch-once` | K 的 `data` `0x700` → `0x800` | invalid / 1 | `… by C1: the kqueue end exit 8 is not the waitid end exit 7` |
| `control.p3-k-exit7`、`control.p3-w-exit7`、`control.p3-c-exit7` | 没有注入 | confirmed / 0 | — |

- 10 个接缝用例都满足以下条件：
  - 命中 1 次；
  - 内层的理由恰好一条；
  - 内层的清理为 `verified: 1 processes gone`；
  - 同一内层运行的 `layout.1` 为 confirmed；
  - 外层的补救为 not-needed，Final 为 ended，保险清理为 nothing left，没有问题。
- 3 个对照同样满足 `exact`，内层都为 confirmed、退出 0。

**期限与预算**：

| 作业 | `TestP3` | 每次运行最长 | 63 次运行合计 | `TestGate`（含 P-3 的 13 个） | 包 |
|---|---|---|---|---|---|
| arm64 | 7.77 s | 359 ms（`layout.1`） | 7.7 s | 253.6 s | 295.7 s |
| x86_64 | 8.21 s | 308 ms | 8.2 s | 255.0 s | 307.5 s |

- 都远在每次 60 s、P-3 的 15 min、`go test` 的 15 min 与作业的 20 min 之内。
- 没有“未执行”，没有阶段超时。

**补救之前的状态与失败的历史**：

- 本轮所有运行都不需要补救：
  - `WP25P` 中没有工具失效；
  - 工具普查没有找到存活的进程；
  - 门禁的外层补救都为 not-needed。
- 本轮在 CI 上没有失败、取消、unsupported、未执行、未知或夹具缺陷；固定代码点是本轮推送的唯一原型代码点，没有重跑。

### 15.8 作者本地的证据（与诊断 CI 分开）

- **静态检查**：`go vet` 在 darwin/arm64、darwin/amd64、linux、windows 上通过；`gofmt` 干净。
- **Windows 11（本机）**：`go test -count=1 -v -timeout 15m ./...` PASS，53/53 confirmed，门禁 92/92，组合 13/13，408.8 s。
- **WSL Ubuntu**：交叉编译的 linux/amd64，**没有** `-race`。52 confirmed、1 unsupported（`T5.adapter.pwsh-argv`：WSL 中没有 pwsh，与本轮无关）；门禁 111/111，组合 13/13。
- 两次本地运行之后，只改了两处 darwin 专用的代码：
  - 门禁内层的选择改为 `TestP3$/…`；
  - 分组场景中组长等待超时的处理，以及它的事件记录。

  所以本地运行与 `4e690d8` 在这两处之外相同；本机不能运行 darwin 的部分。
- **纯函数的临时副本**：解码器、N 与解析器复制到 Windows 上的临时包运行，`TestP3Decode`、`TestP3Parse` 通过。
  - 变异检查共 9 处：去掉 W3a、W5、K1 的下界、K2 的“信号旁的第 8 ~ 15 位”、解析器的重复键检查与有符号、无符号的规范写法检查，以及 N 中的最高字节与 `NOTE_EXITSTATUS` 检查。
  - 有符号值的规范写法第一次存活，补上用例之后重跑检出；其余各运行一次，都检出（共 10 次运行）。
  - 这只是作者证据，不是原生证据。

### 15.9 证据的限制（P-3 不证明的）

- **内核**：只有两个内核，即 26.6.2 arm64（`VMAPPLE`，虚拟机）与 26.6.1 x86_64。两者都是 `xnu-12377.161.x`，没有公开源码。观察与固定源码的推演一致，**不**证明这两个内核与公开源码相同，也不覆盖其他版本，包括 15.x 的 Intel。
- **调用入口与运行器**：只验证了 E1（直接陷入）。没有 E2 的原型，没有选择生产调用入口，也不证明直接陷入在以后的版本中继续可用。不是生产 Runner。
- **布局探针**：比较的是 SDK 26.5 的用户态头文件与 Go 侧，**不能**证明内核采用了哪一个宏分支；那只由上面的原生值间接支持。
- **前提**：
  - P1 ~ P4 由构造满足（本进程的直接子进程、父进程存活、没有其他等待者），没有在它们不成立的情形下测试；
  - 只有信号 9；没有生成 core 的终止，也没有原生的停止状态；
  - 原生的 `ENOMEM`、`EINTR` 没有构造。
- **注入的证据**：
  - `group.enum-unknown` 的故障是注入的，不是原生的读取失败；
  - 接缝只证明解码器在改写之后的值上的处理，不证明内核会产生那些值。
- **次数与规模**：每个场景 3 次，每个架构一个作业，小样本。
- **普查**：工具的标记普查覆盖继承了标记的后代，不覆盖改写自身环境的进程，也不是对任意程序的认证。

### 15.10 对 G5 的影响（作者的证据陈述，由独立审查裁定）

- **本轮增加的证据**：在**同一个**固定代码点 `4e690d8` 上，原生 arm64（macOS 26.6.2）与原生 x86_64（macOS 26.6.1）按 §15 的计划运行了全部 63 次 P-3。
  - 原始值、规范化结果与解码器的输出都与预测相同；
  - 10 个接缝门禁与 3 个对照都按 `exact` 通过；
  - 既有的 macOS 场景在新的核对下也都为 confirmed。
- **作者不宣布 G5 满足**。按规范 §20，下面几项仍是冻结审查的内容：
  - 生产调用入口 E1 / E2 的选择，以及支持声明与已测组合之间的差额（15.x 的 Intel 只有静态依据，没有公开源码的内核）；
  - 其余 F 类门槛的冻结审查。
- **没有改动规范**：规范的支持声明与“已验证的组合”没有因本轮改动，由独立审查判断是否、以及怎样写入；规范只同步了“P-3 已执行、待审查”的状态。
- **会改变设计的结果**（§19.12）：本轮没有出现。没有架构差异、最高字节或中间位与推演不同、重复读取不同、K 与 W 的低 8 位不一致，`k.esrch` 也得到了 `ESRCH`。

### 15.11 Review 101 的定向修复（已修复，待独立复审）

[Review 101](reviews/codex-review-101.md) 对 PR #41（HEAD `c776f6a`，base `dbcd5d4`，代码点 `4e690d8`）为**技术 NO-GO**，有三项 BLOCKER：

- **B1**：计划内的枚举故障覆盖了同一次真实读取的错误，运行可以得到 confirmed；
- **B2**：工具文件与观察器的关闭错误被丢弃，进不了终判；
- **B3**：成员枚举经 x/sys 的 `SysctlKinfoProcSlice`，它在 `ENOMEM` 时于函数内部无界重试，所声明的预算没有覆盖实际的调用链。

本节只处理这三项及其直接关联的共享调用链；Review 101 的 N1 ~ N3（非阻塞）本轮没有处理。

**起点、归档与提交**：

- 起点核对：远程、本地与工作区一致；HEAD `c776f6a5f0e9a1fd09d4a1959e18122fe99b8307`，base / master `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b`，代码点 `4e690d813b88af3005da6e6ca25d9cc851ee6f92`。
- Review 101 原样归档为单独的提交 `6ae979778775e3a64e19166967a814b458d1789c`（只有该文件，26685 字节）。源文件、暂存区与提交 blob 的 SHA-256 都为 `1c27916733ee2f4a5f8726e8e689b0989327855bdc37f5c7e7b6750d71c40cac`。
- 历史审查原文与处理记录的旧内容没有改动。

| 提交 | 内容 |
|---|---|
| `6ae9797` | Review 101 的归档 |
| `7f9fda0871511f0586d6377b91f70006129f740c` | **回归先行**：回归与必要的接缝，旧的错误处理不变（x/sys 的循环原样移入 `kinfoProcs`，关闭结果仍被丢弃，计划故障仍覆盖真实错误） |
| `eb2317fa0be728202ceb705a5bf5aa72eb6ab012` | **修复，新的固定代码点** |
| 之后 | 只有文档：本节、review-resolution §125 与状态的同步 |

- 范围：只改 test-only 的 `prototype/wp25`（14 个文件，相对 `c776f6a` +728 / −97）。
- workflow 没有改：平台、只读权限、失败传播、作业 20 min 与 `go test` 15 min 的期限、P-3 的次数与预算都不变。
- 生产代码、根依赖、Schema、迁移与根 `ci.yml` 没有改；规范只同步状态行。

**B1 的修复**（`confirmEnd`）：

- 每次真实读取的结果先保存，再看计划的故障：
  - R 记录第 1 次、每次失败与最后一次读取（`kern.proc.pgrp <组>, read <n>`，存活的成员或错误）；
  - 读取失败立即 `s.flag`（T），在计划的故障作用之前。
- 计划的故障只替换**候选的输入**，命中记入 F 与 `hits`；它不覆盖、不替换真实的结果，也不豁免它的失败。
- 候选没有确认结束时（计划的故障，或读取失败），回收组长之前先结束组。
- 四个维度分开保存：
  - 真实调用的结果及错误：R 与 T；
  - 计划的故障是否命中：F 与 `hits`；
  - 被测候选的响应：M（`the group's end: …`）与预测的比较；
  - 工具的有效性与最终结论：`WP25RESULT` 的 `invalid`、`cleanup` 与 `verdict`。
- **同源审计**（“先读真实值、再改写为计划的故障”）：
  - P-3 中这样的路径只有 `confirmEnd`。
  - 观察改写类的接缝（`rewriteW`、`rewriteK`）只在真实调用**成功**之后作用；真实调用的错误先返回，不经过接缝。
  - P-1、P-2、P-4、T2 与外层组合中的“真实调用之后替换结果”都是门禁的接缝，不是预期 confirmed 的计划故障：注入的错误本身就使运行 invalid，所以不会产生 confirmed。
  - 但在这些路径上，同时发生的真实错误会被注入的错误代替，不出现在理由中。这是理由文本的限制，不是假通过；它们不在 B1 的调用链上，本轮没有改动。

**B2 的修复**（`scen.closer`）：

- 场景从句柄存在的时刻起成为它的所有者。
- 关闭只做一次（`sync.Once`），由所有者的代码或场景的收尾先调用的一方完成。
- 关闭的错误：
  - 记入 `s.flag`，不被之后的成功、正常退出或清理完成清除；
  - 场景的 `cleanup` 以 `NOT verified: closing failed (…)` 开头；
  - 不重试（失败之后的状态未知，描述符号可能已被复用），不写成“已释放”，也不写成“已泄漏”。
- 收尾才发生的错误（`done` 中运行的关闭）同样进入最终的 `invalid` 与 `verdict`。
  - `WP25P` 行在收尾之前打印，它的 T 是到那时为止的工具失效（§15.3）；
  - 最终的 T、C、V 以 `WP25RESULT` 为准，其中包含收尾的错误。

**错误传播表**：

| 来源 | 创建者 | 所有者（自取得起） | 关闭或保存的入口 | 错误的保存者 | 最终的消费者 | 对终判的影响 |
|---|---|---|---|---|---|---|
| 成员读取的真实错误 | `groupLive` → `kinfoProcs` | — | `confirmEnd`，每次读取之后立即 | `s.flag`（T）与 R | `done` → `invalid`、`verdict` | invalid；计划的故障不改变它 |
| 计划的故障 | `confirmEnd` | — | 只替换候选的输入 | F 与 `hits` | M 与预测的比较 | 单独发生时按预测 confirmed |
| kqueue 观察器（P-3 的 `observer`、`startTree`） | `newExitObserver` | 场景（`s.closer`，创建之后立即） | `done` 的收尾 | `s.flag` 与 `closeFail` | `done` → `invalid`、`cleanup` 的前缀、`verdict` | invalid，收尾的错误也计入 |
| 注册失败时的 kqueue | `newExitObserver` | 该调用本身 | 返回错误之前关闭 | 与注册的错误合并（`errors.Join`） | 调用者的 `invalidf` | invalid |
| P-1 的观察器（macOS、Linux） | `newExitObserver` | 场景（`s.closer`） | 三处显式关闭，或 `done` | 同上 | 同上 | 同上；不重复关闭 |
| Linux 的 pidfd 观察器（共享的 `startTree`） | `newExitObserver` | 场景 | `done` | 同上 | 同上 | invalid |
| 工具步骤的 `.out` 与 `.err` | `toolRaw` | 场景（`s.closer`，各自创建之后立即） | 启动之后显式关闭；后一个创建失败时，立即关闭已创建的；或 `done` | 同上 | 同上 | invalid |
| 读回的文件 | `readBack` | 场景（`s.closer`） | 读完立即关闭（`os.ReadFile` 会丢弃关闭的错误，不再使用） | 同上 | 同上 | invalid |

- 管道排空的夹具中也有被忽略的关闭：`blocking_unix_test.go` 的 `wfd`，以及 `keep_unix_test.go` 的 `kept`（T5、S6 的夹具）。它们不在 P-3 的文件与观察器生命周期上，本轮没有改动，只在此登记。

**B3 的修复**：

- `kinfoProcs` 按 §19.5 的枚举预算读取 kinfo_proc 列表：
  - `ENOMEM` 最多重试 3 次，每次只在调用者的截止之前重试；截止属于调用者，全部重试共用它；
  - 超出任一条件，或其他错误，都返回错误（未知），不当作空列表；
  - 第一次调用总会进行，它的结果照常保留（即使已过截止）。
- 成员的读取与普查都不再使用 x/sys 的无界循环。
- `kqWaitRaw`：`EINTR` 只在截止之前重试；截止时或之后的 `EINTR` 为观察失败，不当作“没有事件”。
- 工具步骤：截止在创建文件与启动之前固定，等待用同一个绝对截止，不再在启动之后重新给出完整的时长。
- 单次系统调用（sysctl、kevent、close）没有硬实时的中断：期限约束的是是否开始下一次重试，不是一次调用的耗时。这不是实时保证（§19.5）。

**期限覆盖表**：

| 重试或等待 | 实际重试的最内层 | 重试的条件（受审规格） | 截止的所有者 | 到期时 |
|---|---|---|---|---|
| `kern.proc.pgrp`（组的成员） | `kinfoProcs`（原为 x/sys 内部的无界循环） | `ENOMEM`，最多 3 次（§19.5） | 调用者：`confirmEnd` 与 `startGroup` 的收尾（`T_kill`）；`confirm(d)`；P-3 分组场景的读取（观察段的截止）；P-1、`emain`（`T_kill`）；T2 的 `abort` | 返回错误（未知），不确认结束 |
| `kern.proc.all`（普查） | 同上 | 同上 | P-3 的工具普查：观察段的截止；外层门禁的普查：`T_kill`；保险清理：它自己的 10 s（现在也覆盖它的第一次普查） | 普查失败（T 失效，或外层的 `census` 问题） |
| `kern.proc.pid`（`procInfo`） | x/sys 的 `SysctlRaw`，没有内部重试 | 不重试（在“最多 3 次”之内，更保守） | — | 未知 |
| `kern.procargs2`（普查） | `SysctlRaw`，没有内部重试 | 不重试 | — | 普查错误 |
| kqueue 的等待 | `kqWaitRaw` 自己的循环 | `EINTR`，在剩余的期限之内（§19.5） | 调用者给出的时长，进入时换算为绝对截止 | 截止时或之后的 `EINTR` → 观察失败；超时 → 没有事件 |
| 工具步骤 | `toolRaw` | — | 步骤开始之前固定：min(单步上限, 观察段的截止) | 结束直接启动的进程、普查之后停止（invalid） |

- Linux 的 `groupLive` 与 `censusTag` 接受截止但不使用：/proc 的读取没有无界重试（`allPIDs` 读一次，`censusOne` 最多 3 次）。
- **同类模式，登记，本轮没有改动**：Linux 的 pidfd 等待在 `EINTR` 之后同样无条件重试（截止之后以 0 超时）。它不在 P-3 的调用链上。
- Go 运行时与标准库内部的 `EINTR` 重试（`wait4`、文件 I/O）不是枚举，本轮没有改动：
  - 等待由 `childProc.reap` 的期限约束；
  - Wait 的 goroutine 由 `childProc` 持有，由 `stop` 结束并由清理核实。

**回归与门禁**（只在 darwin 上；内层的选择同 §15.6 补充 6）：

| 用例 | 注入（都是接缝的构造，不是原生的 API 故障） | 预期 |
|---|---|---|
| `control.p3-enum-unknown` | 没有，只有计划的故障 | confirmed / 0 |
| `control.p3-enum-enomem-retried` | 成员的第一次读取：3 次构造的 `ENOMEM`（调用不进行），第 4 次为真实调用 | confirmed / 0，命中 3 次（合法的重试） |
| `p3.enum-error-once` | `group.enum-unknown` 中成员的第一次读取由注入的错误回答（调用不进行），同时有计划的故障 | invalid / 1，理由恰好一条，含 `observation failed: the members of group` 与注入的文本 |
| `p3.enum-error-once-healthy` | 同上，在没有计划故障的 `group.zombie-leader` 中；之后的读取都成功 | 同上 |
| `p3.enum-enomem-exhausted` | 4 次构造的 `ENOMEM` | invalid / 1，命中 4 次，理由含 `the 3 retries are used up` |
| `p3.close-out-once` | 第一个工具步骤的 `.out` 真实关闭之后，结果被替换为注入的错误 | invalid / 1，`cleanup` 以 `NOT verified: closing failed` 开头 |
| `p3.close-kq-once` | `k.exit7` 的主路径得到正常的结果之后，收尾时观察器的真实关闭结果被替换 | 同上 |
| `p3.create-err-once` | 第一个工具步骤的 `.err` 不创建，由注入的错误回答 | invalid / 1，`cleanup` 为 `no processes`，并观察到已创建的 `.out` 被关闭 |
| `p3.enum-error-and-close-once` | 读取的错误与观察器的关闭错误在同一次运行中 | invalid / 1，**两条**理由都保留 |

- 门禁的新字段：
  - `hits`：命中次数须恰好等于给定的值；
  - `sees`：内层须观察到给定的文本；
  - `reasons` 与 `cleanup`：细化 `exact` 的理由条数与清理的开头。
- 每个用例都核对注入的文本确实进入理由（`because`、`alsoBecause`），不只核对 invalid。
- **直接检查**（`enum_check_darwin_test.go`；替换了系统调用，不是原生证据，不计为场景）：
  - `TestEnumBudget`（8 项）：
    - `ENOMEM` 连续直到用尽 3 次重试；
    - 慢的 `ENOMEM` 直到截止；
    - 边界处到期（第一次调用结束时已过截止，不重试）；
    - 截止之前成功；
    - 截止之后成功（保留）；
    - 其他错误、空列表、不完整的条目。
  - `TestEnumChain`（3 项）：`groupLive`、`procList`、`censusTag` 都经过预算。
  - `TestKqWaitDeadline`（4 项）：`EINTR` 不断、边界处的 `EINTR`、`EINTR` 之后超时、`EINTR` 之后得到事件；并核对每次调用的超时不超出等待的结束。
  - 每个调用都在预先设置的硬期限下运行。到期即为失败，之后它的桩立即回答一个终止的结果，调用返回，没有遗留的 goroutine。复现旧的无界实现不依赖 CI 的总期限。

**修复之前**：回归先行 `7f9fda0`，诊断 run 40816447439（push，attempt 1，completed / **failure**）。

| 作业（job ID） | 镜像 | 结果 |
|---|---|---|
| `macos-latest`（123258520722） | `macos-26-arm64` 20261013.0424.1；macOS 26.6.2（25G83），`xnu-12377.161.14~5`（`VMAPPLE`） | failure：场景 96/96 confirmed（P-3 63）；门禁 93/99；组合 13/13；`TestP3` PASS |
| `macos-26-intel`（123258520785） | `macos-26` 20260824.0517.1；macOS 26.6.1（25G76），`xnu-12377.161.13~4` | 同上 |
| `ubuntu-latest`（123258520783） | `ubuntu-24.04` | success：53/53；门禁 111/111；组合 13/13 |
| `windows-latest`（123258520481） | `windows-2025-vs2026` | success：53/53；门禁 92/92；组合 13/13 |

- 四个作业的检出与 WP25META 都为 `7f9fda0`。
- 两个 macOS 作业的失败与预测完全一致，并且各自因为对应的缺陷本身而失败：
  - 6 个新门禁失败：
    - `p3.enum-error-once`：内层 confirmed / 0，即审查的反例；
    - `p3.enum-enomem-exhausted`：confirmed / 0，4 次 `ENOMEM` 被无界循环吸收；
    - `p3.close-out-once`、`p3.close-kq-once`、`p3.enum-error-and-close-once`：confirmed / 0，关闭的错误被丢弃；
    - `p3.create-err-once`：invalid，但没有关闭已创建文件的记录。
  - 3 个保护性用例通过：`control.p3-enum-unknown`、`control.p3-enum-enomem-retried`、`p3.enum-error-once-healthy`。
  - 直接检查失败的子项共 8 个：
    - `TestEnumBudget` 的 3 项（用尽、直到截止、边界）；
    - `TestEnumChain` 的 3 项；
    - `TestKqWaitDeadline` 的 2 项（`EINTR` 不断、边界）。

    每项都在硬期限处报告（例：2 s 内无界重试约 1.4 亿次），随后桩被停止，调用返回。其余的直接检查子项通过。
- 同一次提交的常规 CI run 40816451875 为 cancelled：推送 `eb2317f` 时，根 `ci.yml` 的 `concurrency`（`cancel-in-progress: true`）自动取消了它。当时 Go tests（Windows）仍在运行，其余 7 项 success。这不是手动取消，它也不是本轮的证据。

**修复之后**：固定代码点 `eb2317f`，诊断 run 40817305419（push，attempt 1，completed / **failure**）。

| 作业（job ID） | 镜像 | 结果 |
|---|---|---|
| `macos-latest`（123261288396） | `macos-26-arm64` **20261021.0445.8**（比修复之前的新）；macOS 26.6.2（25G83），`xnu-12377.161.14~5`（`VMAPPLE`）；Xcode 26.6（17F113）、SDK 26.5、Apple clang 21.0.0 与之前相同 | **failure**：场景 96/96 confirmed（P-3 63，既有 33）；门禁 **98/99**（P-3 21/22）；组合 13/13；5 项直接检查 PASS；`TestP3` 7.66 s；包 300.4 s |
| `macos-26-intel`（123261288366） | `macos-26` 20260824.0517.1；macOS 26.6.1（25G76），`xnu-12377.161.13~4`；工具同上 | success：96/96；门禁 99/99（P-3 22/22）；组合 13/13；5 项直接检查 PASS；`TestP3` 8.07 s；包 320.7 s |
| `ubuntu-latest`（123261288385） | `ubuntu-24.04`，`-race` | success：53/53；门禁 111/111；组合 13/13 |
| `windows-latest`（123261288091） | `windows-2025-vs2026` | success：53/53；门禁 92/92；组合 13/13 |

- 四个作业的检出与 WP25META 都为 `eb2317f`；arm64 与 Intel 是同一个固定代码点。
- **P-3 的场景**（两个 macOS 作业相同）：
  - 63/63 confirmed；63 行 `WP25P` 都没有工具失效；结果中没有 `invalid` 的理由；清理都以 `verified` 开头；
  - 每次运行最长 408 ms（arm64 的 `layout.1`）与 308 ms（Intel），63 次合计 7.6 s 与 8.0 s；
  - 没有补救、未执行、unsupported 或未知。
- **原始值、规范化与预测**：
  - 每个架构上，21 个场景族中的 19 个，其 R（PID 之外的全部字段）、N 与 M 与 `4e690d8` 逐项相同，3 次运行也相同；布局的 55 个键与 `4e690d8` 相同；
  - `group.*` 只多了按 B1 新增的成员读取记录：
    - `group.zombie-leader`：第 1 次读到那个成员，第 2 次为空；
    - `group.enum-unknown`：第 1 次读到那个成员，没有错误，计划的故障之后不再读取。

    N 与 M 不变。
- **门禁**：
  - Intel 22/22；
  - arm64 21/22：9 个新用例在两个架构上都按预期通过（命中次数、理由、清理的开头、`sees` 都符合），外层的补救都为 not-needed，Final 为 ended，保险清理为 nothing left。
- **arm64 唯一的失败**是既有的接缝用例 `p3.k-nostatus-once`：
  - 外层的记录：内层退出 1；内层只创建了 5 个工具进程（布局场景的 5 步），命中 0；
  - 目标 `k.exit7.1` 唯一的理由是 `not executed: the layout premise does not hold: the probes of this process gave [invalid]`，即同一内层运行中的 `layout.1` 为 invalid。
  - 外层的 Final 为 ended，保险清理为 nothing left，登记没有缺口。
  - `k.exit7.1` 的理由只有这一条，所以元数据（环境）的读取没有失败。
  - **`layout.1` 为什么 invalid，没有记录**：门禁只保存目标场景的结论与理由，内层的输出不进入外层的日志。所以原因**未知**：
    - 不能归因于 B1 ~ B3 的修复，也不能证明与它们无关；
    - 能在 5 个工具都已启动之后使布局场景 invalid 的路径，有工具的普查（存活的进程，或普查失败，包括新的有界枚举）、关闭的失败、读回的失败、探针的退出或解析，以及工具步骤的期限。哪一条成立，现有证据不能区分。
  - 同一个用例在以下运行中都通过：同一代码点的 Intel；arm64 的 `7f9fda0`（另一个镜像构建）与 `4e690d8`。同一作业中 `TestP3` 的 3 次布局与其余 21 个内层的布局都为 confirmed。
  - 按本轮的规则，这项失败照实保留：没有重跑，也没有为取得绿色而推送新的代码点。是否修改夹具，使门禁也保存同一内层运行中其他结果的理由并重新诊断，由负责人决定。
- **常规 CI**：见 review-resolution §125（最终 HEAD 的合并快照）。它不运行原型模块，不是 P-3 的证据。

**作者本地的证据**（与诊断 CI 分开；不是原生证据）：

- **静态检查**：`go vet` 在 darwin/arm64、darwin/amd64、linux、windows 上通过（`7f9fda0` 与 `eb2317f`）；`gofmt` 干净。
- **WSL Ubuntu**（交叉编译，没有 `-race`）：`7f9fda0` 与 `eb2317f` 都是 52 confirmed、1 unsupported（WSL 中没有 pwsh，与本轮无关），门禁 111/111，组合 13/13。
- **Windows 11 本机**：`eb2317f` 为 53/53，门禁 92/92，组合 13/13。
- **临时副本中的变异与定向撤回**（仓库之外，Windows 上运行；循环与关闭的代码从工作区逐字抽出）：
  - **B3**：抽出 `kinfoProcs`、`kqWaitRaw` 与直接检查，以 x/sys 名字的替身编译。共 6 个变体：
    - 修复版 PASS，并重复 20 次 PASS；
    - 旧实现：检出 5 项；
    - 去掉重试的上限：检出 1 项；
    - 去掉截止的检查：检出 2 项；
    - `EINTR` 无条件重试：检出 2 项；
    - 每次重试重新取得完整的期限：检出 3 项。
    - 这一项**第一次存活**：初版的检查只比较相邻两次的超时，检不出它。之后改为核对每次调用的超时不超出等待的结束，才检出；加强的检查包含在 `eb2317f` 中。
    - 在 Windows 上，两次时钟读取之间约有 1.5 ms 的差，所以容差取 5 ms；这一变体超出 10 ms 以上。
  - **B2**：完整的包复制到临时目录，加入只在副本中存在的场景：失败的显式关闭、只在收尾时失败的关闭、成功的关闭，以及关闭失败与另一项失败并存。它们在子进程中经 `s.done` 结束，由驱动读取 `WP25RESULT`。
    - 修复版 PASS；
    - 去掉 `s.flag`、去掉 `sync.Once`（重复关闭）、去掉收尾时的关闭、去掉 `cleanup` 的前缀，4 个变异各自检出。
  - **B1**：`confirmEnd` 的函数体逐字抽出，只把接收者换成替身，读取由给定的返回值代替（不是 Darwin 的原生错误），同样经 `s.done` 判定：
    - 修复版：健康的计划故障 confirmed，真实错误 invalid，真实错误加计划故障 invalid；
    - 旧的函数体：真实错误加计划故障得到 **confirmed**，复现了 Review 101 的反例；
    - 变异“计划故障豁免第一次读取的错误”：检出。
  - 场景层面的整体撤回由修复之前的诊断给出：每个回归各自运行，失败的理由都是对应的缺陷本身，没有被其他错误碰巧挡住。

**对旧证据的影响**（`4e690d8`，诊断 run 40751152882，§15.7）：

- **原生的原始值（R）、N、M 与预测的比较**：作为历史观测保留。三项缺陷影响的是有效性的判定与收尾，不改变读到的值。`eb2317f` 上两个架构重新取得，逐项相同（`group.*` 只多了新增的记录）。
- **工具的有效性**（“`WP25P` 中没有工具失效”“清理都以 `verified` 开头”）：不能继续当作成立的证明。在 `4e690d8` 上：
  - 关闭的错误进不了 T；
  - 枚举的 `ENOMEM` 被无界重试吸收；
  - 所以这些只说明没有记录到失败，不说明不存在。由 `eb2317f` 的运行取代。
- **`group.enum-unknown` 的故障分类**：6 次运行的 F 显示真实的读取没有错误；但在 `4e690d8` 上，计划的故障可以掩盖并发的真实错误，所以这些运行不能证明门禁会保留它。由 `eb2317f` 与新的门禁取代。
- **有界的收尾**：`4e690d8` 上枚举与 `EINTR` 的重试不受期限约束，“各段都在期限之内”只是实际的读数，不是有界的证明。
- **13 个门禁的接缝响应**：接缝作用于解码器，不经过 B1 ~ B3 的路径，所以按规则检出的结论可作为历史观测；但 `exact` 中“清理已核实”同样受 B2 影响。
- **整体的结论**：旧运行整体的 success 不说明这三项缺陷不存在。
- **N3 的措辞**：§15.7 的“本轮所有运行都不需要补救”只指 P-3 的运行；套件中既有的门禁与组合有故意的遗留与补救。

**证据的限制**（在 §15.9 之外）：

- B1 ~ B3 的回归都是接缝的构造，没有原生地构造读取错误、`ENOMEM`、`EINTR` 或关闭错误；直接检查替换了系统调用。
- 关闭失败之后的状态未知：原型不重试，不声称已释放。
- 单次系统调用没有硬实时的期限（见上）。
- 作者本地的变异只覆盖临时副本中的循环、关闭的所有者，以及 `confirmEnd` 的控制流模型。
- 修复之后的诊断为 failure：arm64 一项既有门禁的内层布局场景 invalid，原因未知（见上）。

**对 G5 的影响**（作者的证据陈述，由独立复审裁定）：

- 本轮增加的：同一代码点 `eb2317f` 上，B1 ~ B3 的回归在 arm64 与 x86_64 上都通过；两个架构的 P-3 原生场景重新取得，与之前相同。
- 未解决的：固定代码点的诊断正式为 failure，arm64 一项门禁失败，原因未知。
- 作者**不宣布** G5 满足，也不宣布 B1 ~ B3 关闭；规范的支持声明与“已验证的组合”没有改动。
- §15.10 所列冻结审查的内容不变。

### 15.12 Review 102 的最小观测与补证（待独立审查）

[Review 102](reviews/codex-review-102.md) 对 PR #41（HEAD `fbcef6f`，base `dbcd5d4`，修复代码点 `eb2317f`）的结论分两层：

- Review 101 的 B1、B2、B3 在受审的原型调用链范围内分别**关闭**，修复本身**技术 GO**；没有发现有具体反例支持的新 BLOCKER。
- 固定的 PR **暂不建议合并**：固定代码点的诊断正式为 failure；arm64 的 `p3.k-nostatus-once` 没有执行目标观察、没有命中接缝；同一内层运行中 `P3.layout.1` 的具体错误在外层输出中丢失。
- **N-102-1**（NON-BLOCKER，诊断可追溯性缺陷）：`runInner` 解析了内层全部的 `WP25RESULT`，但 `runGate` 主要只复制目标的理由；`because` 不符时 `exact` 分支不运行，即使运行也只列 ID 与 verdict。

本节只处理 Review 102 列出的最小观测（§6.3 的 1、2）与授权的一次补证；三项原缺陷不重开。修复的 GO 不是整个 PR 的合并建议。Review 101 的 N1（显式的结果集合与重复的拒绝）与 N2（R_before 的结构）仍为非阻塞，本轮没有实现。

**起点、归档与提交**：

- 起点核对：远程、本地与工作区一致；HEAD `fbcef6fd230cedd60bc4943a6192e3eb1a875698`，base / master `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b`，修复代码点 `eb2317fa0be728202ceb705a5bf5aa72eb6ab012`；工作区只有未跟踪的 Review 102。
- Review 102 原样归档为单独的提交 `5e16cb64f98ae31390640d6f8942cc71172484ff`（只有该文件，31064 字节）。源文件、暂存区与提交 blob 的 SHA-256 都为 `84d75601bb0cb640b41bb32a48ae856c809aaafacc60aa408925deb93c4abcbb`。
- 历史审查原文与处理记录的旧内容没有改动。

| 提交 | 内容 |
|---|---|
| `5e16cb6` | Review 102 的归档 |
| `9d450365cb11f343a634f47d21cf962c9dddec35` | **观测修正与回归，新的固定代码点** |
| 之后 | 只有文档：本节、review-resolution §126 与状态的同步 |

- 范围：只改 test-only 的 `prototype/wp25` 的 3 个文件（`gate_test.go`、`gate_combo_test.go`、`p3_darwin_test.go`，相对 `eb2317f` +151 / −3）。
- workflow 没有改：平台、只读权限、失败传播、作业 20 min 与 `go test` 15 min 的期限、P-3 的次数与预算都不变。
- 生产代码、根依赖、Schema、迁移与根 `ci.yml` 没有改；规范只同步状态行。

**观测修正**（`gate_test.go`）：

- `runInner` 在解析每一条 `WP25RESULT` 时，除了原有的按 ID 的映射，另按打印的次序保存全部结果（`innerRun.all`）。
- `gateResult` 增加 `inner`（JSON 键 `inner`）：该内层运行打印的**全部最终结果**，原样、按次序，不论目标的 `because`、命中或 `exact` 是否满足。每条都有：
  - `id`、`verdict`、`reason` 与 `invalid`（每条理由）；
  - `cleanup`；
  - `expected` 与 `observed`（既有的预测与观测摘要）；
  - `launches`、`work`、`owned`、`shortfall` 与 `ms`。
- 命中与站点仍在门禁这一层（`inject_hits`、`inject_sites`），与之前相同；目标是否执行，由目标记录的 `launches`、`work` 与理由给出。
- 来源是最终的 `WP25RESULT`，不是 `WP25P` 中提前打印的 T。工具的原始 stdout、stderr 不回显；理由中只有既有的 40 字符截断（`clip`）。
- 只保存，不判定：
  - 判定、解码、预测、故障注入的时点、判定的优先级、准入、预算、所有权与清理都不变；
  - `exact` 分支及其理由文本不变；
  - 按 ID 的映射与以前一样（后出现的覆盖先出现的），重复的结果只是在 `inner` 中按次序都可见，不被拒绝——这不是 N1 的实现。
- `outerFault` 增加 `env`：组合检查给内层运行附加的环境。`TestGate` 传零值，内层环境与之前相同；witness 仍是第一项。

**确定性回归**（`gate_combo_test.go`，3 个组合；它们使用的门禁只在 macOS 上运行，所以只在两个 macOS 作业中执行）：

| 组合 | 构造 | 必须满足（全部） |
|---|---|---|
| `combo.trace-layout-tool` | 门禁 `p3.k-nostatus-once` 原样（K1 的接缝已武装）；组合在内层运行的 PATH 最前面放一个 `xcrun` 存根（shell 脚本，向 stderr 写一行后 exit 1）。这是**构造的真实工具失败**，不是接缝，也不是原生故障 | 门禁 `pass=false`；门禁的理由含 `not executed: the layout premise does not hold`；工具错误为 `hit` 类（`hit: the fault never reached its site`）；补救 not-needed，Final ended，保险清理确认结束；`inner` 恰为 2 条：`P3.layout.1` invalid、理由含 `the tool step xcrun-find-clang ended with exit 1`、清理以 `verified` 开头，`P3.k.exit7.1` invalid、理由含布局前提不成立、清理为 `no processes`；目标 launches 0、work 0；命中 0，没有站点 |
| `combo.trace-layout-late` | 同一门禁，内层的故障换成新的接缝 `p3-layout-close-late`（K1 没有武装）：布局场景在开始时多取得一个文件 `/dev/null`，交给场景的关闭器，只在收尾中关闭；真实关闭之后，接缝把结果换成注入的错误。失败发生在布局的 `WP25P` 行之后，只在它的最终结果中 | 门禁 `pass=false`、理由同上；没有工具错误；补救、Final 与保险同上；`inner` 恰为 2 条：`P3.layout.1` invalid、理由含 `closing the seam's late file`、清理以 `NOT verified: closing failed` 开头，`P3.k.exit7.1` 同上；目标 launches 0、work 0；命中恰为 1，站点为该接缝（`the seam's late file`） |
| `combo.trace-healthy`（对照） | 门禁 `control.p3-k-exit7` 原样 | 门禁 `pass=true`；没有工具错误；`inner` 恰为 2 条，都为 confirmed，清理都以 `verified` 开头；目标 launches 与 work 都大于 0；命中 0 |

- 新的接缝 `p3-layout-close-late`（`p3_darwin_test.go` 的 `lateClose`）只在内层运行、这个注入名、第 1 次布局场景中作用；它记录命中，不称为原生 API 故障。其他注入名与非内层运行中，它直接返回，P-3 的行为不变。
- 这些组合检查的是“保存了什么”，不重新判定：内层与门禁本身的判定仍由原有的 judge 作出；它们的通过只证明信息得到保存，**不是新的原生机制证据**。
- 既有的门禁、组合与健康的多场景、双错、外层错误的对照都不变。
- 两个 `trace-layout-*` 组合借用 `p3.k-nostatus-once` 的选择，但内层的条件不同（存根或另一个接缝），目标都不执行。布局前提成立、K1 命中的完整链，仍只在 `TestGate` 的那一次运行中按原有的严格条件判定；补证没有增加它的运行次数。
- Linux 与 Windows 上，这 3 个组合按其门禁的平台跳过；`inner` 的保存由这些平台上的每一条门禁行实际经过（见下文对日志的核对）。

**作者本地的证据**（与诊断 CI 分开；不是原生证据）：

- **静态检查**：`go vet` 在 darwin/arm64、darwin/amd64、linux、windows 上通过；`gofmt` 干净。
- **旧保存路径与新路径的对照**（只在仓库之外的临时副本中，没有提交，也没有为它触发旧代码的诊断）：
  - 副本中加入一个只作内层运行的测试：前提场景以自己的理由（`R102-SCRATCH-PREMISE-REASON`）invalid，目标场景以“前提不成立”未执行。外层对目标调用真实的 `runGate`（`because` 指向一个不存在的理由，`exact`）。
  - 旧路径（`fbcef6f` 的原型树）：门禁 `pass=false`，命中 0，Final ended，保险清理 nothing left；门禁的整行 JSON 中**没有**前提的理由。
  - 新路径（`9d45036` 的原型树）：门禁的结论与上面相同；`inner` 中原样有两条结果，前提的理由与 `invalid` 都在。
  - Windows 11 本机与 WSL Ubuntu（交叉编译，没有 `-race`）上结论相同。日志的 SHA-256：Windows 旧 `1f12a08a…6f3`、新 `028e868a…0e2`；WSL 旧 `2d16bf19…566e`、新 `308b653e…8b19`；临时测试文件 `d14f30c1…8955`。
  - 只运行所选测试时没有根层场景，`summarize` 按设计返回 1，所以两份副本的包都以 1 退出；判断取自该测试自己的 PASS / FAIL。
- **完整的包**（`9d45036` 的工作树）：
  - Windows 11 本机：53/53，门禁 92/92，组合 13/13；
  - WSL Ubuntu：52 confirmed、1 unsupported（WSL 中没有 pwsh，与本轮无关），门禁 111/111，组合 13/13；
  - 与 `eb2317f` 的本地结果相同；每条门禁行都带 `inner`，其中目标的记录与该行的 `got`、`launches`、`work` 及故障用例的 `inner_cleanup` 一致。
  - 3 个新组合只在 macOS 上运行，本机不能执行。

**旧失败的保存**（run 40817305419，只读取，没有重跑）：

- 重新取得并保存（作者本机的会话临时目录）：run、作业列表、attempt 1 与 artifact 列表的元数据，四个作业的完整日志与注释。四份日志与上一轮保存的副本逐字节相同。
- SHA-256：
  - 日志：arm64（123261288396）`25ecd74c…2a01`，Intel（123261288366）`175acdaf…619a`，Ubuntu（123261288385）`bac2e854…7c90`，Windows（123261288091）`d21f3149…8b92`；
  - 元数据：run `d5486d87…6a08`，作业 `d00ff584…d403`，attempt 1 `a2718fe3…5025`，artifact 列表 `244563f9…856c`（0 个 artifact）。
- 注释：arm64 作业为 `Process completed with exit code 1.`（failure）与一条关于 arm64 runner 排队时间的通知；Ubuntu 为 ubuntu-latest 标签迁移的通知；Intel、Windows 没有注释。
- 日志与 artifact 中**没有** `P3.layout.1` 那次内层结果的理由。日志中可读到的事实：
  - 失败的那次门禁：created 5、receipts 0，用时 352 ms；
  - 同一用例通过的运行中：created 6、receipts 1，用时 550 ~ 716 ms（`4e690d8` 与 `7f9fda0` 的两个架构、`eb2317f` 的 Intel、`9d45036` 的两个架构，共 7 次）。

  这只说明失败的那次门禁用时较短、没有启动目标的 helper（与目标的 launches 0 一致），**不说明** `layout.1` 为什么 invalid。
- `layout.1` 的原因保持**未知**：
  - 新的运行成功不能补写旧的原因；
  - 镜像的变化不作为因果结论；
  - 不声称与 B1 ~ B3 的修复或本轮的观测修正有关或无关。
- 原有的失败记录、未命中的事实与未知项都不删改（§15.11、review-resolution §125）。新的观测只使之后的同类失败可以追溯，不保证复现或查明旧失败。

**新固定代码点的诊断**：`9d45036`，run 40847347022（push，attempt 1，completed / **success**，09:08:52 ~ 09:17:00 UTC，0 个 artifact）。

| 作业（job ID） | 镜像 | 结果 |
|---|---|---|
| `macos-latest`（123357986540） | `macos-26-arm64` **20261013.0424.1**（与回归先行 `7f9fda0` 那次相同，**不是**失败那次的 20261021.0445.8）；macOS 26.6.2（25G83），`xnu-12377.161.14~5`（`VMAPPLE`）；Xcode 26.6（17F113）、SDK 26.5、Apple clang 21.0.0 | success：场景 96/96 confirmed（P-3 63，既有 33）；门禁 99/99（P-3 22/22）；组合 16/16；5 项直接检查 PASS；`TestP3` 7.67 s；包 302.1 s |
| `macos-26-intel`（123357986481） | `macos-26` 20260824.0517.1；macOS 26.6.1（25G76），`xnu-12377.161.13~4`；工具同上 | success：96/96；门禁 99/99（P-3 22/22）；组合 16/16；5 项直接检查 PASS；`TestP3` 7.59 s；包 307.1 s |
| `ubuntu-latest`（123357986066） | `ubuntu-24.04` 20261025.353.1，工作流在 Linux 上加 `-race`（脚本未改） | success：53/53；门禁 111/111；组合 13/13；包 467.8 s |
| `windows-latest`（123357986509） | `windows-2025-vs2026` 20261021.278.1 | success：53/53；门禁 92/92；组合 13/13；包 344.0 s |

- 四个作业都取得并检出 `9d450365cb11f343a634f47d21cf962c9dddec35`（`fetch … +9d45036…`，`checkout -B test/wp2.5-prototype-p3`），WP25META 的 `sha` 也是它；arm64 与 Intel 是同一个固定代码点。
- 没有重跑、取消或手动触发。各作业只有 3 个设计内的 SKIP（`TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError` 只作内层运行），它们不计为通过；根层没有 refuted、invalid、unsupported 或未执行的结果。
- 注释：arm64 作业仍有那条排队时间的通知，Ubuntu 有标签迁移的通知，都不是失败。
- 日志与元数据已保存（同一临时目录），SHA-256：arm64 `7686260a…e537`，Intel `6d8403b9…cbe9`，Ubuntu `deb9c5cd…5939`，Windows `7d58110e…fefa`；run `f6c695a1…027a`，作业 `9704fe23…9bde`。

**完整链**（`TestGate` 中那一次 `p3.k-nostatus-once`，由门禁行的 `inner` 逐项读出；两个架构相同）：

| 环节 | arm64（123357986540） | Intel（123357986481） |
|---|---|---|
| 布局前提 | `P3.layout.1` confirmed：launches 5、work 55，清理 `verified: 5 processes gone` | 同左 |
| 目标实际执行 | `P3.k.exit7.1`：launches 1、work 1，清理 `verified: 1 processes gone` | 同左 |
| 接缝命中 | 命中 1：`P-3 kevent: fflags 0x84000000 -> 0x80000000, data 0x700 -> 0x700` | 同左 |
| 被预期的 K1 拒绝 | invalid，恰好一条理由：`planned corruption detected by K1: K1: the returned fflags 0x80000000 lack NOTE_EXIT or NOTE_EXITSTATUS` | 同左 |
| 外层的严格条件 | pass；内层退出 1；没有问题项与工具错误；created 6、receipts 1，登记没有缺口；补救 not-needed，Final ended，保险清理 nothing left；550 ms | pass；同左；679 ms |

- 没有其他工具错误、额外的拒绝理由、清理失败或未知被预期的 K1 拒绝豁免：`exact` 要求的唯一理由、已核实的清理与布局的 confirmed 都成立，外层的 judge 没有任何问题项。

**三个组合回归**（两个架构结果相同）：

| 组合 | 组合的结论 | 自检（门禁）的实际结果 |
|---|---|---|
| `combo.trace-layout-tool` | pass（自检按预期**失败**） | `pass=false`；理由：目标的理由不含 K1、是布局前提不成立；工具错误 `hit: the fault never reached its site`；命中 0；`inner`：`P3.layout.1` invalid `the tool step xcrun-find-clang ended with exit 1: stub xcrun: exits 1`，launches 1、work 0，清理 `verified: 1 processes gone`；`P3.k.exit7.1` invalid、launches 0、work 0、`no processes`；created 1；补救 not-needed，Final ended，保险 nothing left |
| `combo.trace-layout-late` | pass（自检按预期**失败**） | `pass=false`；理由同上；没有工具错误；命中 1，站点为 `p3-layout-close-late`；`inner`：`P3.layout.1` invalid `cleanup not verified: closing the seam's late file /dev/null failed: injected: closing the late file failed at the end (the real close was made)`，launches 5、work 55，清理 `NOT verified: closing failed (the seam's late file /dev/null); verified: 5 processes gone`；`P3.k.exit7.1` 同上；created 5；其余同上 |
| `combo.trace-healthy` | pass（自检按预期通过） | `pass=true`；没有问题项；命中 0；`inner` 两条都为 confirmed、清理已核实；目标 launches 1、work 1 |

- `trace-layout-tool` 的形状与 run 40817305419 中 arm64 的失败相同（目标未执行、命中 0、`hit` 类工具错误、外层其他维度正常），但这里布局的理由被保存了下来。`trace-layout-late` 中布局的失败只在收尾中发生，门禁保存的是包含它的最终结果。
- 这些是构造（存根与接缝）的结果，只证明信息得到保存，不是新的原生机制证据。

**全部门禁行与根层的 P-3**：

- 四个平台的每一条门禁行（99、99、111、92 条）都带 `inner`；其中目标的记录与该行的结论、launches、work 及故障用例的清理一致；所有门禁都通过。
- 两个 macOS 作业的 P-3 根层：
  - 63/63 confirmed（60 次 normal、3 次 fault-injection）；63 行 `WP25P` 都没有工具失效；清理都以 `verified` 开头；
  - 每次运行最长 358 ms（arm64 的 `layout.1`）与 305 ms（Intel 的 `k.signal9.1`），合计 7.6 s 与 7.5 s；
  - 21 个场景族的 R（PID 之外）、N 与 M 与 `eb2317f` 逐项相同；与 `4e690d8` 相比，19 个族相同，`group.*` 只多了 B1 新增的成员读取记录（与 §15.11 相同）；布局的 55 个键相同。
- Review 101 的 9 个门禁与 5 项直接检查在两个架构上再次通过。

**常规 CI**：`9d45036` 的 run 40847351213（pull_request，attempt 1）8/8 success。最终文档 HEAD 的常规 CI 见 review-resolution §126；它们都不运行原型模块，不是 P-3 的证据。

**对旧证据的影响**：

- run 40817305419 的正式 failure、它的失败记录与未知项不变；不因 `9d45036` 的 success 改写。
- `eb2317f` 上 B1 ~ B3 的门禁、直接检查与 P-3 的根层场景，按 Review 102 的采纳保留。`9d45036` 与 `eb2317f` 只差观测字段、3 个组合与一个只在新注入名下作用的接缝，它上面的同一组结果是第二次观测，不替代第一次。
- K1 门禁以前的通过（`4e690d8`、`7f9fda0`、`eb2317f` 的 Intel）在 `exact` 下已隐含布局为 confirmed；`9d45036` 首次在外层的输出中逐项留下这条链。

**证据的限制**（在 §15.9 与 §15.11 之外）：

- 这是一次运行。它的成功不说明那项失败不会再出现，也不说明它的原因；而且 arm64 这次的镜像与失败那次不同，不能对那个镜像作出任何结论。
- 两个回归的构造都不是原生故障：存根是测试构造的真实工具失败；晚到的关闭失败是接缝在真实关闭之后替换结果。
- 观测只保存内层打印的 `WP25RESULT`：内层没有打印结果（启动失败、超时被结束）时，仍只有 `inner run: …` 的错误与已解析的部分；工具 stderr 只有理由中的 40 字符；命中仍按整次内层运行计数，只能由站点的文字区分。
- Review 101 的 N1、N2 没有实现。

**对 G5 的影响**（作者的证据陈述，由独立审查裁定）：

- 本轮增加的：同一固定代码点 `9d45036` 上，arm64 与 x86_64 的 K1 门禁完整链可以逐项追溯并通过；整套诊断四个作业 success。
- 未改变的：run 40817305419 的失败原因未知；§15.10 所列冻结审查的内容。
- 作者**不宣布** PR 整体 GO、G5 满足、规范冻结或生产授权；规范的支持声明与“已验证的组合”没有改动。

**状态**：Review 102 的最小观测与补证**待独立审查**（Review 103）。PR #41 不合并，Auto-fix、Auto-merge 关闭。规范第 8 版第二稿未冻结；G5 未满足；生产 WP2.5 未授权；WP2.5 与 M2 未验收。（其后：Review 103 技术 GO，N-102-1 关闭，见 §15.13。）

### 15.13 Review 103 的结论与收尾

[Review 103](reviews/codex-review-103.md) 对 PR #41（HEAD `ab4b7dd`，固定的原型代码点 `9d45036`）为**技术 GO，建议合并固定对象**（review-resolution §127）：

- **N-102-1 关闭**。关闭范围是最小的可追溯性缺口，不是任意原始输出的无损归档。
- **观测修正与回归技术 GO**。两个故障组合的通过证明原门禁正确拒绝且信息得到保存，不算原生机制的 confirmed。
- **合并的证据差额已补齐**：同一代码点的两个 macOS 架构上，布局有效、目标执行、K1 接缝命中并按预期拒绝。
- **Review 101 的 B1 ~ B3** 按 Review 102 的范围保持关闭。
- **N1**（显式的结果集合、拒绝重复）、**N2**（改写之前原始值的结构）没有实现，仍为非阻塞。
- 没有新的 BLOCKER；没有新增产品决定、保证调整或风险接受；Q-3 (a) 与其他已批准的决定不变。

**口径的确认**（照审查原文，§15.12 的写法不改）：

- `inner` 认可的口径是“同一次内层运行中，所有已成功解析的既有 `result` 字段值，依打印顺序保存，来源为最终的 `WP25RESULT`”。
- 它不是原始行字节的归档：JSON 会重新编码，未知的 JSON 字段不进入当前结构；没有打印的结果、解析失败的原始行与完整的工具输出不在列表内。“原样”不能扩大解释为未知字段或原始字节的无损保留。以后精化这一措辞是非阻塞建议。
- 重复的 ID 在列表中按打印顺序可见，映射仍是后值覆盖；**可见不等于拒绝**，N1 没有因此实现。
- 畸形 JSON、Scanner 的超长或读取错误仍使门禁失败；已解析的部分列表不能恢复成功。
- 新增的三个组合只在 macOS 上执行；Linux 与 Windows 的 13 个是既有组合，不能写成 16 个。

**证据的范围**（照审查原文）：

- 完整链是对原生 kevent 返回之后作计划改写的门禁证据，**不是**原生内核自行返回缺少 `NOTE_EXITSTATUS` 的证据。
- P-3 的 63 次是 21 个场景族各 3 次，不是 63 种独立机制。
- 两个架构的实测内核为 `xnu-12377.161.14~5`（arm64，VMAPPLE）与 `xnu-12377.161.13~4`（Intel），都没有公开源码；SDK 布局与实际内核行为是不同层的证据，没有据此确认固定的公开源码就是运行内核。
- Ubuntu 的 `-race` 不能概括为所有并发机制都已验证。
- run 40817305419 仍为 attempt 1 的 failure；布局为何 invalid 仍未知，不推定为环境、镜像、本轮修复或观测改动造成。新镜像的不同只证明环境身份不同。
- 审查方独立计算的四份旧日志与四份新日志的 SHA-256 与 §15.12 的登记一致；审查方**没有**读取作者会话临时目录中的原件，作者保存的元数据序列化文件及其字节哈希没有独立核验。
- 审查方本机（Windows，Go 1.27.1）：原样副本一次通过；仓库外的共享采集探针首轮因审查方自己的断言错误失败（不计为反例），更正之后一次通过。没有原生 macOS，三个 macOS 组合以源码与完整的 CI 日志核查。
- 非阻塞的观测限制（审查 §6.2）：内层没有打印最终结果即退出或超时；工具 stderr 只有 40 字符的摘要；命中按整次内层运行计数、以站点文字区分；N1；N2。没有发现这些限制在三种构造或普通 K1 路径上形成新的绿色假通过。

**门槛的状态**（Review 103 的裁定；§15.10 ~ §15.12 是之前的登记，保留原文）：

| 门槛 | 本轮增加的支持 | 仍不能推出 |
|---|---|---|
| G5 / P-3 | 同一固定代码点、两种原生架构上，计划场景、既有观察器与门禁适用；K1 的布局前提、目标执行、唯一规则与收尾可以逐项追溯；最小的错误保存回归有效。证据**足以作为输入进入冻结审查** | 生产 ABI 的稳定、所有 macOS 版本、任意宿主或程序、全部原生故障；SDK 布局探针不升级为运行内核的宏分支证明。生产入口 E1 / E2 的选择、支持与已测组合的差额与其他 F 类须在冻结审查中处理。**不认定整体满足** |
| G3、G6、G7 | 没有新增 | 维持各自的局部支持 |

**收尾的范围**：相对受审 HEAD `ab4b7dd` 只有 Review 103 的归档与文档；原型与 workflow 仍与 `9d45036` 相同，不为纯文档收尾重跑原型诊断。技术 GO 与合并建议不是规范冻结、产品批准、生产入口的选择、实施授权或 WP2.5 / M2 验收。

## 16. P-5：macOS 入口 E2 的最小原型（计划第四稿，Review 110 技术 GO）

### 16.1 授权、依据与固定的身份

- **授权**：
  - 负责人 2026-10-30 决定 Q-4 选 (a)（review-resolution §131）：入口 E2 为拟采用的入口，先完成 P-5 及其独立审查，再决定冻结。
  - 同日授权在 PR #42 合并核验成功之后，另开纯文档 PR 定稿本计划，交 Review 107。
  - Review 107 为技术 NO-GO（B1：门禁内层的期限与准入冲突；N1、N2 非阻塞）。负责人授权在同一 PR 中定向修订（review-resolution §133）。第二稿改动了 §16.3 的返回值解释、§16.5 的 `P5.c`、§16.6、§16.7、§16.8、§16.11 与 §16.12；第一稿在提交 `f7cd0a5` 中。
  - Review 108 为技术 NO-GO：准入冲突的修复成立，B1 整体仍有残余 B1-R（终止请求之后等待的上界没有依据，未确认的分支没有截止出口与移交）；Review 107 的 N1、N2 在计划规格层面收口。负责人授权在同一 PR 中最小修订（review-resolution §134）。第三稿改动了 §16.8（撤回由 `WaitDelay` 推出的返回上界，写定门禁内层的结束、等待与移交，重写时间线二，N-108-2 的启动边界）、§16.9、§16.11 与 §16.12；第二稿在提交 `18eb2cf` 中。
  - Review 109 为技术 NO-GO：B1-R 的主要修订成立，残余 B1-R-S（发信号与回收共用锁的持有边界没有写定，等待可能在锁上绕过截止）；N-108-1、N-108-2 关闭。负责人授权在同一 PR 中只修订 B1-R-S（review-resolution §135）。第四稿改动了 §16.8（控制锁、请求任务与状态锁；五个事件、启动、截止、提前出口、所有权表、不变式、最终收尾与时间线的同步）、§16.9、§16.11 与 §16.12；第三稿在提交 `096fc05` 中。
  - Review 110 对第四稿（`35ecd79`）为文档技术 GO：B1-R-S 关闭，B1-R 在计划规格层面整体收口。负责人条件授权合并 PR #43，并条件授权在其合并核验全部成功之后，另开 test-only 原型 PR 按本节实现并执行 P-5（review-resolution §136）。本节的技术内容没有因此改动。
- **本节只是计划**：
  - 固定计划、检查的办法与预期；不写原型代码，不构建或运行入口 E2。
  - 本计划的提交先于任何原型代码。
  - P-5 的执行须在本计划经独立复审之后另行明确授权。
- **依据**：
  - 规范第 9 版 §21.5（H1 ~ H6）、§21.6（Q-4 的决定）、§21.9（计划草稿、分类与期限）；
  - Review 104 §8、Review 105 §6、Review 106 §4 ~ §6。
  - 规范 §21.9 的草稿与本节不一致时，以本节为准；本节不改变规范的规则。

**固定的身份**：

| 项 | 固定值 |
|---|---|
| base 与所依赖的规范 | master `0442f14ebfc985d2dfadd9684efeb44d821763e4`（PR #42 的合并，规范第 9 版收尾稿） |
| Go 工具链 | `go1.27.3`，标签解引用为 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`（官方仓库 `golang/go`） |
| 依赖 | `golang.org/x/sys v0.49.0`（提交 `b6b8557e8e2974925aec7a136ba0e323eedc5f43`；`prototype/wp25/go.sum`：`h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=`）；不新增依赖 |
| 静态依据 | xnu 26.5 `ac9718fb1af618d5ce8678d0dc6e8a58f252216f`（规范 §19.5、§21.4、§21.5） |
| 计划的版本 | 本节所在的提交（PR 描述登记它的 SHA） |
| 平台 | `macos-latest`（arm64）与 `macos-26-intel`（x86_64），同一个固定代码点 |

### 16.2 Go 1.27.3 的过滤逻辑与调用桩（重新核对）

规范 §21.5、§21.9 引用的是本机的 Go 1.27.1。本计划固定 `go1.27.3`，所以按 `2aae1000` 重新取得下列文件。它们与 1.27.1 逐字节相同（SHA-256 相同），所以引用的行号不变：

| 文件 | SHA-256 | 核对的内容 |
|---|---|---|
| `src/syscall/syscall_darwin.go` | `07eea080aae7b97b37943bb0964594a198c73b8ec35697b2b8912c0323cd129a` | `:330` ~ `335` 的 `errno(r, e)`：`int32(r) == -1` 时返回 e，否则返回 0。`:403` ~ `408` 的 `syscall6`：带 `//go:linkname`、`//go:nosplit`、`//go:uintptrkeepalive`，先调用 `syscalln`，再经 `errno(r1, err)` 返回 |
| `src/runtime/sys_darwin.go` | `512814eff1c543337b9e5e57febf25641ff0b2520f2326ebaa9c9252ab393090` | `:23` ~ `31` 的 `syscall_syscalln`（`entersyscall` / `exitsyscall`）；`:38` 起的 `syscall_rawsyscalln`，`:49` 经 `libcCall` 进入 `syscallN_trampoline` |
| `src/runtime/sys_darwin_arm64.s` | `4a9b539aca0682b4f8f90257fff4a2be645befb5a695623dd2f77c8ef853d17d` | `:484` 起的 `syscallN_trampoline`；目标调用之后，`:568` ~ `569` 读取线程的 errno |
| `src/runtime/sys_darwin_amd64.s` | `4dc648f7324975e4a57fdb8d793844d798bd4db8d1c65aef42fe7502372bee6d` | `:510` 起的 `syscallN_trampoline`；`:571` ~ `572` 读取 errno |
| `src/runtime/os_darwin.go` | `5fbeed78977bcae998e0b48dea235690be222cef8713c29397ae941cd1ea9685` | `:397` 信号处理带 `SA_RESTART` |
| `src/runtime/signal_darwin.go` | `ec4119fe65a45c77833150bc11107e4a11bf1b062c04cd1c87c9731a01935d9b` | `:20` SIGSYS 为 `_SigThrow` |
| `src/runtime/proc.go` | `40d15ce858cf058cff92606001a29ee1689db88c121a2f0c283163a0d64a7375` | `:5658` ~ `5660` `LockOSThread` 只绑定 goroutine 与线程 |
| `src/cmd/compile/internal/noder/noder.go` | `2aa3493281bfced8419994a553606fe074f3dbde6a46cf7a66cff0519cb68e83` | `:315` 起：带库名的 `//go:cgo_import_dynamic` 允许在普通包中使用 |
| `src/cmd/link/internal/ld/macho.go` | `665c2270cec3df9bc9b77931d7e87e50ad5726da147e8df651cead6230c1ef17` | `:449` macOS 的最低版本 13.0.0 |
| `src/syscall/zsyscall_darwin_amd64.go` | `b7d5fe4a69c51b0cd8ecab53ae76ff599ac429dadd5fa43c3f5be0c8dda56f8a` | `:44` `wait4` 经 libSystem |
| `src/syscall/zsyscall_darwin_arm64.go` | `941547c92d1204cf9666c40299b1685f393cb5546369b9238ec534dcd7a81201` | 同上（arm64） |

- **由此推出**（Review 106 §4.1）：对这份过滤逻辑，`int32(r1)` 不为 −1 时，不论过滤之前的 e 是什么，输出的 err 都为 0。
- **适用的前提**：入口 E2 确实调用这一个 `syscall6`，不改用未过滤的入口或自制的过滤器。
- **作业的实际版本**：作业中 `go version` 不是 `go1.27.3` 时，记为 T，不继续。

### 16.3 入口 E2 的实现方案

**新的 test-only 子包** `prototype/wp25/e2waitid`（package `e2waitid`）：

| 文件 | 内容 |
|---|---|
| `doc.go`（没有构建约束） | 只有包说明，使非 darwin 平台的 `./...` 不因没有可构建的文件而出错 |
| `waitid_darwin.go`（`//go:build darwin`） | 写法见下面的说明 |
| `waitid_darwin_arm64.s`、`waitid_darwin_amd64.s` | `#include "textflag.h"`；`TEXT libc_waitid_trampoline<>(SB),NOSPLIT,$0-0` 之后 `JMP libc_waitid(SB)`；`GLOBL ·libc_waitid_trampoline_addr(SB), RODATA, $8`；`DATA ·libc_waitid_trampoline_addr(SB)/8, $libc_waitid_trampoline<>(SB)`。与 x/sys `unix/zsyscall_darwin_arm64.s:96` ~ `99` 中 kevent 的写法相同 |

`waitid_darwin.go` 的写法：

- import `syscall`、`unsafe`；
- `//go:cgo_import_dynamic libc_waitid waitid "/usr/lib/libSystem.B.dylib"`，以及 `var libc_waitid_trampoline_addr uintptr`；
- `//go:linkname syscall_syscall6 syscall.syscall6`，加上没有函数体的声明 `func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)`，与 x/sys `unix/syscall_darwin_libSystem.go:13`、`:21` 相同；
- `func Waitid(idtype, id uint32, info unsafe.Pointer, options uint32) (r1 uintptr, err syscall.Errno)`，函数体只有一次调用：`syscall_syscall6(libc_waitid_trampoline_addr, uintptr(idtype), uintptr(id), uintptr(info), uintptr(options), 0, 0)`。

**ABI 与指针的生命周期**：

- **参数**：C 的声明为 `int waitid(idtype_t, id_t, siginfo_t *, int)`，前两个与最后一个都是 32 位。四个参数经寄存器传递：arm64 为 x0 ~ x3，x86_64 为 rdi、rsi、rdx、rcx。动态导入的目标是 libSystem 的 `waitid`（C 的符号 `_waitid`），与 x/sys 对其他函数的写法相同，不用 `$NOCANCEL` 等变体。
- **缓冲区**：
  - 调用者用自己的局部数组 `[siSize]byte`，传入 `unsafe.Pointer(&buf[0])`。
  - 向 uintptr 的转换只出现在没有函数体的函数的调用参数表中，适用 unsafe 的第 4 条规则。
  - 调用之后，调用者 `runtime.KeepAlive(&buf)`。
  - 这些是对源码的静态核对，不是运行的证明。
- **返回值的解释**：消费者按 `int32(r1)` 与 err 的组合解释返回值。五种形式与每一次调用唯一的分类见 §16.7。第二稿按 Review 107 N1 写定：第一稿此处的三条与 §16.7 的写法有重叠，以 §16.7 为准。

**原型的接线**（`prototype/wp25/tree_darwin_test.go`）：

- 新的低层函数 `waitidCall(idtype, id, options uint32) (ret int32, err syscall.Errno, si [siSize]byte, calls int)`：
  - 调用之前写入哨兵：`si_pid` 置 0，其余字节为固定的模式；
  - 调用 `e2waitid.Waitid`，之后 `runtime.KeepAlive`；
  - 记录这一次调用。它是原型中 `e2waitid.Waitid` 唯一的调用者。
- `readWaitid(pid)` 改为调用 `waitidCall(P_PID, pid, WEXITED|WNOWAIT)`，判定按上面的返回值解释。三个调用点都经它：
  - `exitObserver.wait` 的提前路径（`:191`）；
  - `waitidNoWait`（`:434`）；
  - `p3Run.readW`（`p3_darwin_test.go:666`）。
- 删去 `unix.Syscall6(unix.SYS_WAITID, …)`（`:310`）。P-5 的代码点中不再有入口 E1。P-3 的入口 E1 证据（`9d45036`）保留为历史。
- `waitid2`：第一次读取失败之后不再读第二次。这与 §19.5 的错误出口一致：观察失败之后不再观察。
  - P-3 的 W 门禁改写的是成功读取的字段，第一次调用本身是成功的，所以不受影响；
  - `p3.w-reread-once` 改写的是第二次读取，同样不受影响。

### 16.4 构建、符号与静态检查（办法与预期）

都在两个 macOS 作业中执行；在 `prototype/wp25` 下，带 `CGO_ENABLED=0`：

| ID | 检查 | 办法 | 预期 | 不成立时 |
|---|---|---|---|---|
| `P5.static.toolchain` | 工具链 | `go version`、`go env GOVERSION CGO_ENABLED GOTOOLCHAIN` | `go1.27.3`、`0`、`local` | T，不继续 |
| `P5.static.vet` | vet（含 asmdecl） | `go vet ./...` | 通过 | T |
| `P5.static.build` | 构建 | `go test -c -o "$RUNNER_TEMP/wp25.test" .` | 成功 | T（H1 不成立） |
| `P5.static.settings` | 构建设置 | `go version -m "$RUNNER_TEMP/wp25.test"` | 含 `build	CGO_ENABLED=0` | T |
| `P5.static.symbols` | 跳板与动态导入 | `go tool nm "$RUNNER_TEMP/wp25.test"`；Xcode 的 `nm -m "$RUNNER_TEMP/wp25.test"` | 前者有名字含 `e2waitid.libc_waitid_trampoline` 的符号，以及 `waitid` 的动态导入；后者 `_waitid` 为 `(undefined) external`，来自 `libSystem`。第一次运行时原样记录两份输出中的这几行 | T（H1 不成立） |
| `P5.static.entry` | 入口的覆盖 | darwin 的测试用 `go/parser` 解析模块中的 Go 源文件，并把 `.s` 当作文本检查 | 见下面的四条断言 | T（不是机制的反证） |
| `P5.static.load` | 加载 | 测试二进制在目标系统上启动，并执行到 `TestP5` | `TestP5` 开始运行 | T（H2 不成立） |

`P5.static.entry` 的四条断言：

1. `cgo_import_dynamic` 指令与 `libc_waitid` 只出现在 `e2waitid` 中；
2. 没有参数中出现 `SYS_WAITID` 的 `unix.Syscall*`、`unix.RawSyscall*`、`syscall.Syscall*` 或 `syscall.RawSyscall*` 调用；
3. `e2waitid.Waitid` 只被 `waitidCall` 调用；
4. `readWaitid` 与 `P5.err.*` 都经 `waitidCall`。

这些静态检查每个作业 1 次，计入“静态”证据，不计入原生的 confirmed。

**方法的版本**（Review 111 之后补记；上表本身不改）：

- 上表是**第一版**：受审于 `35ecd79`，固定代码点 `0213fb7` 的诊断使用它。
  - 那一次 `P5.static.symbols` 为 T，终判 invalid（T），见 §16.16。
  - 这一结果不回改，也不按第二版重判。
- **第二版**见 §16.17.4：
  - 修订 `settings` 与 `symbols` 的写法；
  - 新增 `P5.static.method`、`P5.static.artifact` 与 `P5.static.binding`；
  - 以 `_waitid` 的每符号绑定记录取代“`nm -m` 显示来自 libSystem”这一条件。

  第二版只是计划，待独立审查。它只适用于获得授权之后的新固定代码点。

### 16.5 原生场景、受控对象与预测（每个场景 3 次）

- **身份前提**：每个场景的被读对象，都是本进程以 `startGroup` 启动、在自己进程组中的直接子进程。读取之前，`r.zombie` 确认它是本进程未回收的僵尸，且启动时间相符（3 s 内）；不成立为 T。
- **独立的预测**：N 用 `p3NormW` 与 `p3Pred` 由原始值 R 计算，不调用 M 的解码器。Review 101 的 N2 仍是限制。
- **收尾顺序**：读取完毕 → `r.reap`（先确认组结束，再回收组长，预测 `wait4` 的状态）→ 普查，与 P-3 相同。

| 场景 ID | 构造 | 经 E2 的 waitid 调用次数 | 预测（R / N） |
|---|---|---|---|
| `P5.layout` | P-3 的布局探针 | 0 | 55 项与 Go 侧相同 |
| `P5.w.exit7`、`-100`、`-107`、`-ff`、`-7f`、`-80` | P-3 的 `p3W`（helper `exit-after`） | 2（`waitid2`） | R 与原型文档 §15.7 相同（`si_status` 依次为 `0x7`、`0x100`、`0x107`、`0xffffff`、`0x7`、`0x0`）；N 依次为 7、0、7、255、7、0；两次读取相同 |
| `P5.w.signal9` | P-3 的 `p3WSignal` | 2 | `si_code` 2、`si_status` 9；N 为被信号终止、9 |
| `P5.c.exit7`、`-107`、`-ff` | 新的 `p5C`。对象的建立与 `p3C` 相同：helper `exit-after 200 <code>`，在运行中注册 kqueue，等待事件。之后对同一个未回收的对象做经 E2 的 `waitid2`，读两次。`p3C` 只读一次，不改，仍供 `TestP3` 的回归使用；所以复用 `p3C` 不能算作两次读取 | 2 | K 的预测与 P-3 相同。两次 W 读取各自与预测相同，彼此也相同（`predictW(a, &b, …)`）。C1：K 的结束与两次读取的解码结果（`decodeW2`）相同。R 记录两次读取 |
| `P5.k.esrch` | P-3 的 `p3KEsrch`：注册得到 `ESRCH` 之后的提前路径 | 2（提前路径 1 次加第二次读取） | 退出，7 |
| `P5.err.echild` | 整个场景 `runtime.LockOSThread`，结束时释放。先建立受控的僵尸子进程（`exit-after 0 7`），再调用 `waitidCall(P_PID, getpid(), WEXITED\|WNOHANG\|WNOWAIT)`；随后对子进程做 `waitid2` | 3（错误 1 次、健康 2 次） | 错误：`int32(r1)` 为 −1、err 为 `ECHILD`，不读取缓冲区（依据：固定源码 `kern_exit.c:3501` ~ `3503`，本进程不是自己的子进程）。健康：每次 0 / 0，R 与 `w.exit7` 相同 |
| `P5.err.einval` | 同上，错误调用改为 `waitidCall(P_PID, 子进程, 0)` | 3 | 错误：−1、`EINVAL`（`kern_exit.c:3316` ~ `3318`，在遍历子进程之前返回，不消费子进程的状态）。健康：同上 |

- **计数**：原生共 21 + 9 + 3 + 6 = 39 次，布局 3 次。
- **线程绑定**：只提供线程局部的语境（规范 §21.9）。不宣称残留负载已经建立。
- **两个真实错误**：都不作用于其他进程，也不改变子进程的状态。

### 16.6 门禁与对照（内层运行）

| ID | 内层 | 注入（接缝） | 预期 | 经 E2 的调用次数 |
|---|---|---|---|---|
| `p5.eintr-once` | `TestP5$/^(layout.1\|w.exit7.1)`，目标 `P5.w.exit7.1` | `p5-eintr-once`，见下面的说明 | 命中恰好 1 次；内层 invalid（非零），理由恰好一条：`observation failed: waitid 1:` 加 EINTR 的错误文本；不读第二次；清理已核实 | 1 |
| `control.p5-w-exit7` | 同上 | 不注入 | 内层 confirmed（零）；两次读取相同 | 2 |

`p5-eintr-once` 的注入：

- 位置：`readWaitid` 中、`waitidCall` 返回之后。
- 先保存真实的结果（`ret`、err 与 R），记入证据。
- 真实的结果是成功时，经 `injectOnce` 把这一次的结果换成（−1，`EINTR`）。站点文字为 `P-5 waitid: (0, 0) -> (-1, EINTR)`。
- 真实的结果不是成功时，不改写，见 §16.7。

两个门禁都按 P-3 门禁的 `exact` 判定，外层的见证、普查、补救与保险清理照旧。第二稿补充以下几点：

- **位置**：两个门禁是 `TestP5` 的子测试，在 39 次原生之后运行（§16.8 的角色 B′），不在 `TestGate` 中。`TestGate` 的门禁列表不变。
- **截止**：内层与外层的截止按 §16.8 的角色 B 与 B′。
- **`exact` 增加的条件**（Review 107 N2）：
  - 内层的 `P5.layout.1` 与目标都有最终结果（收尾之后的 `WP25RESULT`）；
  - 布局的 `evidence` 为 `layout`，`calls` 存在且为 0；
  - 目标的 `evidence` 为 `seam`，`calls` 存在，且 `p5.eintr-once` 为 1、`control.p5-w-exit7` 为 2；
  - `calls` 缺失时不当作 0，门禁失败。

  期限的修订不取消 `exact` 的任何要求：非目标 confirmed、命中次数、目标唯一的理由、退出码，以及内外层的清理与最终状态，都仍须成立。

**另列为 P-5 相关的回归**（不计入上表，也不计入 39 次原生）：

- P-3 的 W 与 C 门禁：`p3.w-sipid-once`、`p3.w-signo-once`、`p3.w-code-once`、`p3.w-highbyte-once`、`p3.w-negative-once`、`p3.w-reread-once`、`p3.c-mismatch-once`，以及对照 `control.p3-w-exit7`、`control.p3-c-exit7`。
- 它们在 `TestGate` 中照常运行，因为入口替换而经过 E2。
- P-3 的根层场景（`TestP3`）同样经 E2 运行。它们也只作回归另列，不再是入口 E1 的证据。

### 16.7 注入之前真实结果的消费与唯一的分类

- **消费**：`p5-eintr-once` 的真实调用不是成功时：
  - 不改写，原样返回真实的错误；
  - 同时记录 T：`the real waitid at the seam's site failed: …`，这一记录持续到终判；
  - 这时注入没有命中，内层理由不止一条，门禁失败。
  - 只保存而不消费，不算失败关闭。
  - 第二稿不改变这一规则。真实调用的形式按下表记录，但不论哪一种非成功的形式，在接缝站点上都持续记为 T。

**每一次经 E2 的调用的形式**（第二稿按 Review 107 N1 写定，取代第一稿 §16.3 的三条解释）：

r 为 `int32(r1)`。下面五种形式互斥，并且覆盖全部组合：

| 形式 | 条件 | 说明 |
|---|---|---|
| X1 成功的形式 | r = 0 且 err = 0 | — |
| X2 错误的形式 | r = −1 且 err ≠ 0 | — |
| X3 协议失效：错误没有 errno | r = −1 且 err = 0 | C 的契约是返回 −1 时设置 errno |
| X4 协议失效：成功带 errno | r = 0 且 err ≠ 0 | 固定的 `syscall6` 只在 `int32(r1) == −1` 时返回 e（§16.2），所以这种组合只能来自另一条调用链 |
| X5 协议失效：契约之外的返回值 | r 不是 0 也不是 −1（不论 err） | C 的契约只返回 0 或 −1 |

**每一次调用唯一的分类**（由形式与这一次调用的预期共同决定）：

| 形式 | 健康的调用（预期 0 / 0）：原生场景与门禁中的读取，以及 `P5.err.*` 之后的恢复读取 | 构造的错误（预期 −1 / `ECHILD` 或 −1 / `EINVAL`） | 接缝站点上的真实调用（`p5-eintr-once`，注入之前） |
|---|---|---|---|
| X1 | 与预测比较 R：相同为**一致**，不同为**原生反证**（R 不符） | **原生反证**：应当失败而成功 | **一致**：接缝把这一次的结果换成（−1，`EINTR`） |
| X2 | **原生反证**：健康调用失败；在 `P5.err.*` 中写作“错误之后的健康恢复失败” | err 与预期相同为**一致**，不同为**原生反证**（错误码不符） | 不改写，记为 **T**：`the real waitid at the seam's site failed: …`，持续到终判（规则不变） |
| X3、X4、X5 | **协议失效（T）** | **协议失效（T）** | 不改写，记为**协议失效（T）**，同样持续到终判 |

- **分类的对象**：只对真实的元组分类。
  - 接缝换上的（−1，`EINTR`）不是原生的结果，不再分类。消费者按 §19.5 的错误出口处理它，得到门禁预期的唯一理由 `observation failed: waitid 1: …`。
  - 这与 P-3 门禁的 `r.seam` 路径相同：改写之后判的是消费者的拒绝，不是原生的预测。
- **没有接缝时**（根层的原生运行与对照的目标）：
  - 健康调用的 X2 记为原生反证，`waitid2` 停止。
  - 消费者因此得到的观察失败（§19.5 的错误出口）是它对错误的正确处理，记入 M，不另记为 T。
  - 这与 P-3 不同：P-3 把失败的读取记为观察失败（T），因为那里的被测对象是解码与组合的规则；P-5 的被测对象是入口 E2 本身。角色 C 的回归仍按 P-3 的规则。
- **唯一**：同一个真实的元组只得到上表中的一个分类，不同时作为两种理由。
- **保留**：每一次调用都记录 r、err（数值与文字）、缓冲区的原始字节（包括哨兵是否仍在）、分类与具体的原因。X2 ~ X5 的缓冲区不解码，原始字节照样保留。
- **协议失效为什么是 T 而不是原生反证**：
  - 消费者依赖的契约有两条：C 只返回 0 或 −1，返回 −1 时设置 errno；固定的过滤在返回值不为 −1 时给出 err 0。X3 ~ X5 落在这个契约之外，消费者无法据此区分成功与失败，所以不能与预测比较。
  - 这是入口 E2 自身调用或报告协议的失效（H3 / H4 不成立），不是外部工具的故障。原因以 `E2 protocol:` 开头，与 §19.12 其他层的 T 分开记录。
  - 它属于 §16.12 的停止条件，不被任何预期的理由豁免，也不计为 confirmed。
- **之后的读取**：
  - 健康的第一次读取不是 X1 时，`waitid2` 不读第二次（§16.3）。
  - `P5.err.*` 的错误调用不论是哪一种形式，之后都照样做 `waitid2`，以保留恢复读取的原始值。

**调用次数**（Review 107 N2）：

- **计划次数**：
  - `P5.static.entry` 与布局 0；
  - `p5.eintr-once` 的目标 1（注入的失败使 `waitid2` 停止）；
  - 健康的 W、C、K 场景与对照的目标 2；
  - `P5.err.*` 3。
- **规则次数**：按消费者的规则，由消费者实际看到的形式推出；接缝改写的那一次，按改写之后的形式。`waitid2` 在第一次读取不是 X1 时停止；`P5.err.*` 先做错误调用，再做 `waitid2`。
- **判定**：
  - 次数是原型自己的调用产生的，不是原生的结果。所以实际次数与规则次数不同时记为 T（`calls:`），不是原生反证。
  - 实际次数等于规则次数、但不等于计划次数时，偏差已经由那一次调用的分类记录，不另加理由。
  - confirmed 要求实际次数等于计划次数。
  - 必需的结果缺少 `calls` 时记为 T，不当作 0。

**运行的终判**（优先级沿用 §19.12 的层次）：

| 状态 | 终判 |
|---|---|
| 有 T：§16.4 的任何静态检查不成立；工具的报告、解析、关闭或身份观察出错；必要的元数据未知；清理失败或最终状态未知；协议失效；接缝站点的真实失败；调用次数与规则次数不同 | invalid（T）。列出全部原因，原生反证的原因也照样列出；之后的成功与补救的成功都不恢复结论 |
| 可靠确认是 Rosetta、架构不符，或缺少 Xcode / SDK | unsupported，不计 confirmed；读不到元数据归 T |
| 布局 invalid 或 refuted、截止不足、未执行、没有命中 | 依赖它的目标不能 confirmed，保留前提场景自己的理由 |
| 没有 T，至少一次调用为原生反证，或其他原生预测不符 | refuted |
| `p5.eintr-once` 被预期的规则拒绝 | 内层 invalid；门禁按 §16.6 的 `exact` 判定 |
| 健康的原生场景中，每一次调用都一致、预测全部相同、实际次数等于计划次数 | 只有这时才可 confirmed |

### 16.8 期限与截止的传递（N-106-2；第二稿按 Review 107 B1 修订）

**第一稿的冲突**（Review 107 B1）：

- 第一稿把根层的单次准入（距截止至少 60 s）也用在门禁的内层上。内层的截止 d_i = d_g − 30 s，最多只有约 30 s，所以布局与目标都不能准入，门禁拿不到有效的证据。
- 直接复用 P-3 的入口同样不成立：
  - `admit` 要求距 `t.Deadline` 至少 120 s；
  - `obsEnd` 不晚于 `t.Deadline` − 1 min，在内层会落在场景开始之前；
  - `runInner` 固定为 150 s / 180 s，没有落实 60 s 的门禁窗口。
- 第二稿把三种角色分开，各自写定准入、观察、收尾与截止。

**不变的部分**：

- **总预算**：D = min(S + 180 s, T − 30 s)，只建立一次。S 为 `TestP5` 开始的时刻，在第一个运行（`P5.static.entry`）之前；T 为 `TestP5` 的 `t.Deadline()`（`go test -timeout 15m`）。
- **D 覆盖**（不按 ID 的前缀区分）：`P5.static.entry`；`P5.layout` 的 3 次；39 次原生；`p5.eintr-once` 与 `control.p5-w-exit7` 的内层运行与外层核验。
- **D 不覆盖**：
  - `P5.static.toolchain`、`vet`、`build`、`settings`、`symbols`：它们是 `go test` 之外的 workflow 步骤，以作业的 20 min 为界。
  - 旧 P-3 的回归（下面的角色 C）：沿用既有的期限，只受同一个 `go test` 截止约束。它们不计入 P-5 的 3 min，也不会每项各得 15 min。
- **门禁的窗口**：每个门禁仍为 60 s（d_g = s + 60 s）；内层的截止仍为 d_i = d_g − 30 s；外层仍在 d_i + 5 s 请求终止（第三稿起，这只是请求，见“门禁内层的结束、等待与移交”）。
- 场景次数、布局前提、`exact` 与失败关闭都不变。

**通则**：

1. 每一种角色只有一个截止的来源（见下表）。
2. 各步骤消费同一个剩余的预算：每一处有界等待取 min(自身的上限, 所在步骤的截止)，不在步骤开始时重新取得完整的期限。
3. 截止已过时，不再开始等待：
   - 准入不成立的，记为未执行；
   - 等待中的核实记为未核实（T）；
   - 补救的成功不恢复结论。
4. 超期、未执行、没有命中与清理未知都失败关闭。不重跑、不减少次数、不删布局、不扩大预算，也不借用 D 之后的 30 s。
5. 用户态的期限不是单次系统调用的硬实时保证。阻塞在系统调用中的调用不受用户态期限的中断，只能由外层请求终止，并按截止停止等待、移交来处理（时间线二甲、二乙）。
6. 跨进程只能用墙上时钟传递 d_i。墙上时钟的跳变会使内层提前或推后到期，但外层按自己的单调时钟在 k = d_i + 5 s 请求终止、在 w = d_i + 10 s 停止等待。所以两种情形都失败关闭，不会延长外层的窗口。

**三种角色**（B′ 是门禁在外层进程中的部分，与 B 同属新门禁）：

| 角色 | 运行在哪里 | 截止的来源 | 准入与单次截止 d | 观察的截止 obsEnd | 观察之后（检查与收尾） | 最终收尾 |
|---|---|---|---|---|---|---|
| A 根层原生运行 | 外层 `go test` 进程中 `TestP5` 的 `P5.static.entry`、`P5.layout` 3 次与 39 次原生 | D | 准入要求 D − s ≥ 60 s；d = min(s + 60 s, D)，准入的运行即为 s + 60 s | min(开始 + 20 s, d − 30 s)，即开始 + 20 s | 检查不超过 10 s，收尾不超过 20 s，运行在 d 之前结束 | `TestP5` 在 S 注册的收尾，截止 D + 30 s，不越过 T |
| B 门禁内层的布局与目标 | 门禁启动的子进程中的 `TestP5`，只选中 `layout.1` 与 `w.exit7.1` | 外层传入的 d_i；不取 S + 180 s，也不取 T − 30 s | 准入要求 d_i − s ≥ 12 s；d = min(s + 12 s, d_i)，准入的运行即为 s + 12 s | min(开始 + 6 s, d − 6 s)，即开始 + 6 s | 检查与收尾共用剩余的 6 s，运行在 d 之前结束 | 内层 `TestP5` 注册的收尾，截止 d_i + 3 s |
| B′ 门禁的外层 | 外层进程中 `TestP5` 的两个门禁子测试 | D 下的 d_g | 准入要求 D − s ≥ 60 s；d_g = min(s + 60 s, D)，即 s + 60 s；d_i = d_g − 30 s | — | `runInner` 的变体在 k = d_i + 5 s 请求终止，在 w = d_i + 10 s 停止等待；之后的普查、补救、见证与保险清理共用到 d_g 的剩余 | 门禁交来的剩余进程归角色 A 的最终收尾 |
| C 旧 P-3 回归 | `TestGate` 的 P-3 门禁与对照，以及 `TestP3` | 既有的：P-3 的总预算 15 min，以及 `go test` 的截止 | 既有的 `admit` | 既有的 `obsEnd` | 既有的 20 s / 10 s / 20 s | 既有 |

**逐个接点的规则**（括号中是现有代码的位置，原型代码自 base 以来没有改动；角色 C 一列全部不变）：

| 接点 | A 根层原生运行 | B 门禁内层 | B′ 门禁的外层 | C 旧 P-3 回归 |
|---|---|---|---|---|
| `admit`（`p3_darwin_test.go:188` ~ `197`：P-3 的总预算；距 `t.Deadline` 至少 `p3Reserve` 120 s，`:56`） | 不用 P-3 的总预算与 120 s，改为 D − s ≥ 60 s。因为 D ≤ T − 30 s，准入时距 T 至少 90 s：本次的 60 s，加上 D 之后 30 s 的收尾 | 不用，改为 d_i − s ≥ 12 s；内层的 `t.Deadline` 不参与准入 | 门禁自身的准入：D − s ≥ 60 s | 不变 |
| `obsEnd`（`:216` ~ `222`：开始 + `p3Inner` 20 s，且不晚于 `t.Deadline` − `p3Margin` 1 min，`:53`、`:57`） | min(开始 + 20 s, d − 30 s)。因为 d ≤ D ≤ T − 30 s，所以仍不晚于 T − 1 min | min(开始 + 6 s, d − 6 s)。不参照内层的 `t.Deadline`，否则会落在场景开始之前 | — | 不变 |
| 阶段的上限（`r.limits` `:201` ~ `212`：观察 20 s、检查 `p3Outer` 10 s；收尾 `cleanLimit` = `p3Clean` 20 s，`main_test.go:380`） | 20 s、10 s、20 s；运行在 d 之后结束另记为 invalid | 观察 6 s；检查与收尾在 d 之前结束，否则为 invalid | — | 不变 |
| 运行内的有界等待（`zombie` 3 s `:679`；观察器 3 s `:861` 等；布局的工具步骤与普查（已截取到 obsEnd）；`reap` 中组的确认 `tKill` 5 s 与回收 1 s，`:700`、`:703`，`tKill` 见 `pipe_test.go:24`；`done` 中的收尾 5 s 与读取者 5 s，`main_test.go:429`、`:335`） | min(自身的上限, d) | min(自身的上限, d) | — | 不变 |
| `runInner`（`gate_test.go:537` ~ `594`：`-test.timeout=150s` `:542`，`WaitDelay` 5 s `:544`，父端等待 180 s `:556`） | 不调用 | 不调用 | 带截止的变体，不用 `WaitDelay`，不无期限地等待，见表下的说明 | 原样调用既有的 `runInner` |
| 普查（`witness_unix_test.go:113` ~ `116`：`tKill` 5 s） | — | — | min(当前 + 5 s, d_g) | 不变 |
| 记录的读取、归属的核实、见证的判定、命中与登记（`gate_test.go` 的 `runGate`） | — | — | 只读取，不等待；在 d_g 之前完成 | 不变 |
| 补救（`witness_unix_test.go:126` ~ `149`：10 s，`:132`） | — | — | min(当前 + 10 s, d_g)；内层进程本身不按 PID 动作，只经发信号的规则 | 不变 |
| 保险清理（`:180` 起：10 s，`:188`，其中的普查共用这一截止） | — | — | 见表下的说明 | 不变 |
| 最终收尾 | 截止 D + 30 s，不越过 T，见表下的说明 | 截止 d_i + 3 s，在外层于 d_i + 5 s 请求终止之前，留 2 s 给输出与退出 | 归角色 A | 既有 |
| 角色的判定 | 没有 `innerEnv` | 有 `innerEnv`，且 `WP25_P5_DEADLINE` 可读、在未来 | — | 不设 `WP25_P5_DEADLINE` |

表中较长的几项：

- **`runInner` 的带截止变体**（B′）：
  - 环境变量 `WP25_P5_DEADLINE`（Unix 纳秒）为 d_i。
  - `-test.timeout` 为 ⌈d_i − 当前时刻⌉ + 5 s，所以内层的 T 不早于 d_i + 5 s。
  - 启动、等待、终止请求与移交见下面的“门禁内层的结束、等待与移交”。
  - 第二稿写的“父端至多等到 d_i + 5 s 结束内层，`WaitDelay` 5 s，所以返回不晚于约 d_i + 10 s”撤回（Review 108 B1-R）。
- **保险清理**（B′）：截止为 min(当前 + 10 s, d_g)。
  - 已证明属于本轮的进程照样发信号，不需要等待；内层进程本身只经发信号的规则，由请求任务执行（R0，见下）。
  - 到 d_g 仍未确认结束时为 T（`insurance`），剩余的本轮进程交给角色 A 的最终收尾。
- **角色 A 的最终收尾**：
  - 收尾所有仍登记在 P-5 名下的进程、文件与观察器，包括门禁移交的项。
  - 只对已证明属于本轮的进程动作。
  - 对移交的任务只按截止查询，不再同步等待（见下面的“最终收尾对移交项的处理”）。
  - 没有确认的记为 invalid / 未知，写入汇总；不另取期限。
- **角色判定的失败**：
  - 外层（没有 `innerEnv`）的环境中有 `WP25_P5_DEADLINE`：T，`TestP5` 不运行。
  - 内层的 `WP25_P5_DEADLINE` 缺失、不可读或已过：选中的两个运行都记为 invalid（`not executed: …`），所以门禁失败。

**门禁内层的结束、等待与移交**（第三稿按 Review 108 B1-R 写定；锁与调用的顺序由第四稿按 Review 109 B1-R-S 写定）

沿用已审的持有原则：有完成信号与累积错误的任务，按绝对截止查询，未完成即移交（`stopTask` / `leftover` / `lingering`，`dest_test.go:286` ~ `352`、`destscen_test.go:20` ~ `83`）。这里只用于 P-5 门禁的内层，不扩展为生产 Runner 或通用的后台任务系统。

**撤回的推论与依据**：第二稿从“到 d_i + 5 s 发 SIGKILL，加 `WaitDelay` 5 s”推出 `runInner` 不晚于约 d_i + 10 s 返回，把终止请求、可靠观察退出与等待返回混成了一个事件。固定的 Go 1.27.3（`2aae1000`）源码：

- `os/exec/exec.go:289` ~ `305`：`WaitDelay` 的计时从 Context 完成或 `Wait` 观察到退出开始；直接调用 `Kill` 不是起点。
- `:936` ~ `970`：`Cmd.Wait` 先执行 `Process.Wait`（`:944`），之后才等待复制任务（`:981` 起）。所以终止请求之后，`Process.Wait` 可以一直不返回，`WaitDelay` 也还没有开始。
- 改用 `CommandContext` 也一样：取消只是请求（`Cancel` 默认为 `Kill`），不能证明 `Process.Wait` 已经返回。
- darwin 没有实现 `blockUntilWaitable`（`os/wait_unimp.go:8` ~ `21`）：`Process.Signal` 与 `Wait` 并发时，在极少的情形下可能把信号发给错误的进程（源码注释所指的 Go issue #13987）。

所以本计划不依赖 `WaitDelay` 或 `CommandContext` 取得任何上界，也不把 `Kill` 当作退出的证明。

**五个事件**（分开记录，各自有时刻、结果与错误）：

| 事件 | 含义 | 由谁产生 | 不能由它推出 |
|---|---|---|---|
| 终止请求 | 向内层发出 SIGKILL，记下时刻与请求的错误 | 请求任务（步骤 R0 ~ R5），由门禁或之后的持有者登记；持有者自己不发信号 | 进程已经退出；请求没有错误也不是退出 |
| 可靠观察退出 | 唯一 Wait 任务以不回收的读取看到内层成为本进程的僵尸（`procInfo`，与 `r.zombie` 相同的读取，`p3_darwin_test.go:678` ~ `694`）；读取失败的错误累积，之后的成功不清除 | 唯一 Wait 任务（步骤 W1 ~ W3） | `Cmd.Wait` 已完成；输出已完整 |
| `Cmd.Wait` 完成 | 唯一 Wait 任务调用的 `Cmd.Wait` 返回，取得 wait4 的状态，即回收；结果在关闭完成信号之前写入 | 唯一 Wait 任务（步骤 W4 ~ W6） | 管道已关闭；后代已结束 |
| 管道关闭 | 复制任务从读端读到 EOF，即所有写端都已关闭（内层与继承了写端的后代都已关闭或结束）；或者持有者关闭了读端 | 复制任务；持有者 | 进程已退出：后代可以持有写端 |
| 输出复制任务结束 | 复制任务返回，它的缓冲区从此稳定；结果（字节与错误）在关闭完成信号之前写入 | 复制任务 | 输出完整：持有者关闭读端时，只得到已读到的部分 |

**启动与资源**（带截止的变体自己创建管道，不复用 `launchInner` 的同步分支）：

1. 门禁开始时记下时刻 g0，作为之后普查的时间下界。
2. 用 `os.Pipe` 取得读端 r 与写端 w；`Cmd` 的 `Stdout`、`Stderr` 都设为 w（`*os.File`）。
   - 固定源码中，`*os.File` 直接交给子进程，`Cmd` 内部不建管道、不启动复制任务（`exec.go:591` ~ `618`，`:601`）；所以 `Cmd.Wait` 没有复制任务要等（`:990`）。
   - 不设 `WaitDelay`，不用 `CommandContext`。
3. `Cmd.Start` 返回错误时，没有子进程：只有 `os.StartProcess` 成功之后才置 `started`（`exec.go:744` ~ `753`），本配置之后的步骤不会失败（`:797`）。这时关闭 r 与 w（错误记 T），记 T，没有要移交的任务。
4. `Cmd.Start` 成功的那一刻起，在任何其他步骤之前，依次：
   - 关闭父进程的 w（错误记 T；门禁同步关闭，不持有任何锁，依据见下面的“锁外的其余调用”）；
   - 从 `cmd.Process.Pid` 读取一次 pid，建立该内层的控制锁与回收标记（初值“未开始”）；
   - 启动唯一 Wait 任务与复制任务，登记到门禁的持有表。

   之后才读取内层的启动身份。

**发信号的规则**（所有权与动作授权分开；锁与调用的顺序见下面的“控制锁、请求任务与状态锁”）：

- **回收的前提**（Review 109 §3.2）：外层进程中只有唯一 Wait 任务回收内层。
  - 其他代码不对内层调用 `Wait`、`Process.Wait` 或 `wait4`，不调用 `wait4(-1)`；内层不登记为场景的进程。
  - 外层不忽略 SIGCHLD。现有原型中唯一的 `signal.Ignore(syscall.SIGCHLD)` 在 helper 的 `sigchld-probe` 分支（`helper_unix_test.go:86`），运行在 helper 子进程中；`tree_unix_test.go:188` 的 `Wait4` 只针对场景登记的进程。P-5 的实现不新增这类调用。
- 向内层本身发信号，只由请求任务进行：它在控制锁内检查回收标记，并在同一个临界区中实际发出信号；唯一 Wait 任务在同一把锁内发布回收标记。
- 标记之前，内层没有被本进程回收，它的 PID 只能指本轮的子进程（活着或僵尸），所以信号不会落到别的进程。标记之后不再向它发信号，记为“已观察到退出，不需要请求”。
- 这条规则不依赖内层的启动身份，所以身份读取失败时仍然适用。
- 内层进程本身不由普查、补救或保险清理按 PID 动作。其余进程的规则不变：只在普查以本轮标记证明、且动作之前身份与标记重新核对成立时动作；归属未知的不动作。

**控制锁、请求任务与状态锁**（第四稿按 Review 109 B1-R-S 写定）

第三稿只写了“在锁内标记”“在同一把锁内检查”，没有写定锁的范围。按直接的实现顺序，Wait 任务可能持锁调用 `Cmd.Wait`；门禁在 k、补救或最终收尾时先等这把锁，到 w、d_g 或 D + 30 s 仍停在加锁上，截止出口失效（Review 109 §4.1）。第四稿写定获取锁、发布状态、释放锁、外部调用与发布完成的顺序。

**两类锁**：

| 锁 | 数量 | 保护的状态 | 谁获取 | 临界区内允许的动作 |
|---|---|---|---|---|
| 控制锁 | 每个内层一把 | 回收标记：初值“未开始”，只变一次，变为“开始回收” | 只有唯一 Wait 任务（步骤 W2）与请求任务（步骤 R1 ~ R3） | W2：一次赋值。R2：读取标记，未开始时调用一次 `kill` |
| 状态锁 | 每个任务一把（Wait、请求、复制、停止） | 该任务自己的结果字段：时刻、累积的错误、完成标志、取得的状态 | 该任务写入；持有者的 `until` 与快照读取 | 只读写这些字段 |

- 门禁（B′）、普查、补救、保险清理、最终收尾与所有 `until` 查询**都不获取控制锁**。
- 持有状态锁时，不调用系统调用（`procInfo`、`kill`、`Close`、`Read`、`Cmd.Wait`），不发送、接收或等待通道，不获取控制锁或另一把状态锁。
- 两类锁不嵌套：持有任何一把锁时，不获取另一把。
- 控制锁的临界区中，唯一的外部调用是 R2 的一次 `kill`。`procInfo`、`Close`、`Cmd.Wait` 与任务完成的等待都在锁外。

**唯一 Wait 任务的顺序**（Start 成功之后立即启动，见“启动与资源”）：

| 步骤 | 持有的锁 | 动作 |
|---|---|---|
| W1 | 无 | 以 `procInfo` 不回收地读取内层（与 `r.zombie` 相同的读取）。读取失败时，取状态锁追加错误，释放，按间隔重试；间隔的等待不持有锁。仍在运行、未知、外来对象或缺失，都不是观察到退出 |
| W2 | 控制锁 | 看到内层是本进程未回收的僵尸之后：取控制锁，把回收标记置为“开始回收”，**释放控制锁**。临界区内只有这一次赋值 |
| W3 | 状态锁 | 记下“可靠观察退出”的时刻与“开始回收”，释放 |
| W4 | 无 | 调用唯一一次 `Cmd.Wait`。这时不持有任何锁；W4 不返回时，停住的只有 Wait 任务自己 |
| W5 | 状态锁 | 写入 `Cmd.Wait` 的错误、取得的 `ProcessState`（只有 Wait 任务在这里读取 `cmd.ProcessState`）与完成标志，释放 |
| W6 | 无 | 关闭完成信号（发布完成） |

**请求任务的顺序**（终止请求的唯一执行者）：

| 步骤 | 执行者与锁 | 动作 |
|---|---|---|
| R0 | 持有者，无锁 | 决定是否需要请求：<br>• 最近一次对 Wait 任务的快照已显示“开始回收”：不启动，记“已观察到退出，不需要请求”。<br>• 本内层先前的请求任务仍 pending：不启动新的任务，只查询它。所以每个内层同时至多一个请求任务。<br>• 否则新建请求任务（完成信号、状态锁、结果字段），先登记到持有者的持有表，再启动。没有无主的请求 |
| R1 | 请求任务 | 取控制锁。在这里等待的只有请求任务本身 |
| R2 | 请求任务，控制锁 | 读取回收标记：<br>• 已开始回收：结果为“不需要”，不发信号；<br>• 未开始：调用 `syscall.Kill(pid, SIGKILL)`，pid 为 Start 成功之后读取的那一个值；记下返回的原始 errno |
| R3 | 请求任务 | **释放控制锁** |
| R4 | 请求任务，状态锁 | 写入结果（“已发出”或“不需要”）、R2 的时刻与错误、完成标志，释放 |
| R5 | 请求任务，无锁 | 关闭完成信号（发布完成） |

- **为什么不用 `Process.Signal`**：它的内部状态检查与 `sigMu`（`os/exec_unix.go:93` ~ `112`）不能代替本计划的回收标记（Review 109 §3.1），而且会把 `ESRCH` 换成 `ErrProcessDone`。直接调用 `kill` 保留原始的 errno。
- **请求的结果都不是退出的证明**：没有错误、`ESRCH` 或其他错误，都只按请求记录。退出只由 Wait 任务确认（W2、W3 与 W5）。
- **串行的保证**：检查标记与实际的 `kill` 在同一个临界区中，W2 的赋值也要先取得这把锁。所以 `kill` 只会发生在标记之前，那时 PID 仍属于本轮未回收的子进程；不会出现解锁之后再按 PID 发信号。
- **挂起时**：R1 或 R2 不返回时（在等控制锁，或 `kill` 没有返回），停住的只是请求任务；W2 也可能因此等锁，`Cmd.Wait` 不开始。持有者不获取控制锁，所以照常在截止返回：请求任务与 Wait 任务都记为 pending，退出码 unknown，两者都移交（下面“时间线二乙的变体”）。

**查询**（`until` 与快照）：

- `until(截止)` 的顺序：
  1. 先不等待地检查完成信号；
  2. 未完成时，在完成信号与截止的计时器之间选择一次；
  3. 取一次该任务的状态锁，复制字段，释放；
  4. 返回“是否完成，以及至今的结果与错误”。

  这沿用 `stopTask.until`（`dest_test.go:338` ~ `350`）的模式。它成立的条件是状态锁只保护短的字段访问；这里的状态锁满足这一条件，而且不是控制锁。
- 快照不会等在 `Cmd.Wait`、`kill`、`Close`、`Read` 或 `procInfo` 之后：这些调用都不在状态锁内。
- 到截止返回 pending、unknown 与至今的错误，按规则记 T。不因为取得了快照而再等一次，也不重新等待未完成的任务。
- 同一阶段的多个查询共用同一个绝对截止，依次进行；截止过后，每一次都只是不等待的快照。
- 快照中的状态只在完成标志为真时存在（W5 一并写入）。持有者不直接读取 `cmd.ProcessState`，也不在复制任务结束之前读取它的缓冲区。

**锁外的其余调用**：

- 门禁同步执行 `os.Pipe`、`Cmd.Start`、关闭 w、读取启动身份与普查，都不持有任何锁。
- **w 的关闭**：
  - 固定源码中，`Close` 只在仍有其他引用时等待（`internal/poll/fd_unix.go:89` ~ `118`；引用归零时由 `:75` ~ `86` 释放）。
  - 父进程对 w 没有其他使用者：`Cmd` 把调用者提供的 `*os.File` 直接交给子进程，不登记为自己要关闭的文件（`os/exec/exec.go:601` ~ `603`），也没有任务读写 w。
  - 所以 w 由门禁同步关闭。
- **r 的关闭**：r 有复制任务的并发读取，它的关闭由停止任务执行（Review 109 §3.1 引用的同一段源码）。复制任务的 `Read` 与停止任务的 `Close` 都不持有锁。

**门禁内的等待与截止**（只用由 d_i、d_g 派生的绝对截止，不另取期限，也不无期限地接收完成信号）：

| 截止 | 时刻 | 到期时做什么 |
|---|---|---|
| k | d_i + 5 s | 门禁以 `until(k)` 查询，Wait 任务的快照仍没有“开始回收”：按 R0 登记并启动请求任务，门禁自己不取控制锁；记 T（`inner: the inner run did not end by itself by d_i + 5 s`）。之后以 `until(w)` 查询请求任务的结果 |
| w | d_i + 10 s（第二稿误作 `runInner` 的必然返回时刻；第三稿改为等待的截止） | 停止等待 Wait 任务、复制任务与请求任务。复制任务未结束时，登记并启动停止任务（关闭读端，错误累积），只做一次不等待的查询。仍未确认的记 T / unknown；请求任务仍 pending 的记“请求未完成”（可能在等控制锁或系统调用）；都移交 |
| d_g | s + 60 s | 普查、补救、见证、保险清理的共同截止（规则不变）；到期仍未确认的记 T / unknown，并移交 |

- **怎样等**：门禁在截止之前按上面的“查询”等待完成信号；到期即返回“未结束，以及至今的错误”，不阻塞，也不获取控制锁。
- **不读未稳定的状态**：Wait 任务完成之前，不读取 `ProcessState`，也不调用 `ExitCode`；复制任务结束之前，不读取缓冲区。
- **退出码**：退出未确认时没有退出码。门禁行记为 unknown，门禁的退出码条件不成立，不以 −1 或 0 代替。
- **输出**：
  - 复制任务在 w 之前读到 EOF：输出完整，由门禁解析（`WP25RESULT`、`WP25INJECT`、`WP25WITNESS` 行）。
  - 在 w 被停止：已读到的部分是安全取得的证据，照样保存并解析；但输出不完整本身是 T（`inner: the output was not complete by d_i + 10 s`），缺少的结果不推定为任何结论。
  - 复制任务在 w 仍未返回：没有可以安全读取的输出，记 T；复制任务与停止任务移交。

**提前出口与每个出口的处理**：

| 出口 | 当时已有的资源 | 处理 |
|---|---|---|
| `os.Executable` 或 `os.Pipe` 失败 | 无，或刚创建的管道两端 | 关闭已创建的端（错误记 T）；记 T；没有要移交的 |
| `Cmd.Start` 失败 | 管道两端 | 同上；没有子进程 |
| Start 成功之后关闭 w 失败 | `Cmd`、r（第四稿更正：按启动步骤 4 的顺序，Wait 任务与复制任务在关闭 w 之后才启动） | 记 T；照常继续启动步骤 4 的其余部分：读取 pid，建立控制锁与回收标记，启动并登记 Wait 任务与复制任务 |
| Start 成功之后读取内层的启动身份失败 | `Cmd`、r、控制锁与回收标记、Wait 任务、复制任务 | 记 T（`the inner run's identity: unknown`）；立即按 R0 登记并启动请求任务（不依赖身份，门禁不取控制锁）；之后与正常路径相同：等到 w，再做普查（时间下界用 g0）、补救、见证与保险清理，移交未确认的部分。不再像 `launchInner`（`witness_unix_test.go:94` ~ `107`）那样 `Kill` 之后同步 `Wait` |
| k 到期、w 到期仍未确认 | 同上；k 之后还有请求任务 | 见上表 |
| d_g 到期仍未确认 | 同上，以及普查证明属于本轮的进程 | 记 T / unknown，移交 |

**所有权表**：

| 对象 | 何时成为本轮的 | 门禁中的所有者 | 移交给 | 谁可以动作、凭什么 | 晚到的结果 |
|---|---|---|---|---|---|
| `Cmd` / `Process`（内层） | `Cmd.Start` 成功 | 门禁（B′） | 角色 A 的最终收尾 | 只经请求任务（R0 ~ R5）；不按别处得到的 PID 动作 | 由持有者读取 |
| 控制锁与回收标记 | Start 成功之后，与 Wait 任务一起建立 | 门禁（随 `Cmd`） | 随 `Cmd` | 只有 Wait 任务（W2）与请求任务（R1 ~ R3）获取；持有者从不获取 | 标记只变一次；变为“开始回收”之后不再发信号 |
| 唯一 Wait 任务 | Start 成功之后立即 | 门禁 | 同上 | 只有它调用 `Cmd.Wait`，在 W4 不持锁调用一次；控制锁只在 W2 持有，只做一次赋值；其他代码不调用 `Wait` / `Process.Wait`，不读取 `cmd.ProcessState` | 晚到的退出与状态记为“晚到”，不改变门禁 |
| 请求任务 | R0 登记时（先登记，后启动） | 登记它的持有者：门禁，或最终收尾 | 门禁登记而仍 pending 的，交给最终收尾 | 只有它向内层发信号，且只在控制锁内、标记之前 | 晚到的结果（已发出、不需要、错误）记为“晚到”，不改变门禁；错误持续保留 |
| 父进程的写端 w | `os.Pipe` | 门禁 | 不移交：Start 之后立即关闭 | — | 关闭的错误记 T |
| 读端 r | `os.Pipe` | 复制任务读取，门禁持有关闭的权利 | 与复制任务一起 | 门禁或持有者关闭（停止） | 关闭的错误累积 |
| 复制任务与它的缓冲区 | Start 成功之后立即 | 门禁 | 同上 | 缓冲区只在任务结束之后读取 | 晚到的字节不再解析进门禁的判定，只记字节数与错误 |
| 读端的停止任务 | 在 w 需要时 | 门禁 | 同上 | — | 错误累积 |
| 解析的责任 | 复制任务结束之后 | 门禁：只在复制任务于 w 之前结束，或在 w 停止之后的那一次查询中已结束时 | 不移交 | — | — |
| 见证文件、命中日志、raise 的观察 | 门禁开始时 | 门禁（子测试的 `TempDir`） | 不移交 | — | 目录的删除不证明进程已结束 |
| 普查证明属于本轮的其余进程 | 普查 | 门禁 | 最终收尾，只交已证明的 | 补救、保险清理与最终收尾；动作之前重新核对身份与标记 | — |
| 各任务晚到的错误 | — | 任务自己累积 | 随任务 | — | 之后的成功不清除，不覆盖已经成立的失败 |

**不变式**：

- 移交不等于释放：移交的任务仍由持有者读取，直到它结束或最终收尾的截止。
- `Kill` 成功不等于退出确认；补救成功不恢复门禁；晚到的成功不改变已经成立的 T。
- 持有者（门禁、补救、保险清理、最终收尾）从不获取控制锁。控制锁的临界区只有 W2 的一次赋值，或 R2 的标记检查加至多一次 `kill`。
- 状态锁不跨外部调用；到截止只取一次快照，不重新等待未完成的任务。
- 之后的运行照常准入（沿用 `lingering` 的 allow 策略）。交出的未完成项与它们的错误留在记录中，P-5 不算完整通过。

**最终收尾对移交项的处理**（截止 D + 30 s，不越过 T）：

- 截止只建立一次，所有移交项共用：
  - 对尚未开始回收的内层按 R0 处理：没有 pending 的请求任务时，登记一个新的请求任务到最终收尾的持有表，再启动；有 pending 的，只查询它；
  - 最终收尾不获取控制锁；对每个任务（包括请求任务）只用 `until(D + 30 s)` 查询，不再同步等待同一任务；
  - 按各门禁的标记普查（同一截止），只对已证明的进程动作。
- 确认结束的，记为“晚到的结束”（时刻、状态与错误），不改变原门禁。
- 到截止仍未确认的，汇总记 unknown，P-5 为 T，按 §16.12 报告。`TestP5` 随即返回；仍未结束的任务留在进程中，不再等待。测试进程退出之后，未回收的内层由系统收养。

**角色 B 的算术**（第三稿按 Review 108 N-108-2 精化）：

记号：s 为门禁的开始，d_i = s + 30 s；δ 为从 s 到 `P5.layout.1` 开始的时间，包括内层进程的启动；b 为布局的耗时；h 为布局结束到目标准入之间的开销。

- **布局**：准入要求 δ ≤ 18 s；它的 d = s + δ + 12 s。
- **目标**：
  - 布局在自己的 d 之前结束，否则为 invalid，目标因布局前提不成立而不执行。所以 b ≤ 12 s。
  - 目标在 s + δ + b + h 准入，要求 d_i 减去这一时刻至少 12 s，即 δ + b + h ≤ 18 s。布局用满 12 s 时，条件为 δ + h ≤ 6 s。目标的 d 不晚于 d_i。
  - 实现按准入时实际的当前时刻判断，不把 δ ≤ 6 s 当作充分的保证。
  - 例（理想的 h = 0）：δ = 6 s、b = 12 s，则布局为 s + 6 ~ s + 18，目标在 s + 18 准入，d = d_i。h > 0 时目标不准入，记为未执行（失败关闭）。
- **收尾与结束**：内层的最终收尾在 d_i + 3 s 之前完成；外层到 k = d_i + 5 s 仍未看到退出时，按 R0 登记并启动请求任务。
- **与名义上限的关系**：
  - 角色 B 的单次 12 s 小于既有各项名义上限之和。所以名义上限之内的最坏耗时会被截取，并失败关闭。
  - 正常的耗时远低于 12 s：§15 中入口 E1 的 P-3 观测，单次运行最长 408 ms，门禁的整次内层 550 ~ 716 ms。这些只作参考，不是对入口 E2 的保证。
  - 角色 A 的名义上限之和为 20 + 10 + 20 = 50 s，小于单次的 60 s，所以只有超过名义上限时才会被截取。

**角色 B′ 的算术**：

- **d_g − d_i 的 30 s 是截止的分配，不是对 Go 等待时长的推断**：到 k 的 5 s 是内层的宽限；k 到 w 的 5 s 是终止请求之后的等待；w 到 d_g 的 20 s 给外层的步骤。
- **外层始终使用实际的剩余量，不保证有 30 s 以上**：内层正常结束时也可以晚到约 d_i + 5 s（最终收尾到 d_i + 3 s，输出与退出到 d_i + 5 s），这时剩余约 25 s。
- **最坏的情形**：在 w 才进入外层的步骤，剩余 20 s。普查（名义 5 s）、补救（10 s）与保险清理（10 s）的名义合计 25 s，所以保险清理至多得到 5 s。各步骤不另取期限；到 d_g 仍未确认的部分记 T / unknown，交给最终收尾（时间线二乙）。

**时间线一：布局与目标都准入并完成**

设 S = 0、T = 600，所以 D = min(180, 570) = 180，最终收尾的截止为 210。时刻是相对时刻，单位 s；各次的耗时是为推演所设的值，不是预测。

| 时刻 | 角色 | 事件 | 截止与剩余 |
|---|---|---|---|
| 0.0 | A | `TestP5` 开始：D = 180；注册最终收尾 | 距 D 180 |
| 0.0 ~ 0.5 | A | `P5.static.entry`：准入 180 − 0 ≥ 60；d = 60 | — |
| 0.5 ~ 21.5 | A | 3 次布局与 39 次原生，设每次 0.5 s。每次准入时距 D 至少 159，d = s + 60，obsEnd = s + 20 | 21.5 时距 D 158.5 |
| 22.0 | B′ | `p5.eintr-once`：准入 180 − 22.0 = 158 ≥ 60；d_g = 82.0，d_i = 52.0，k = 57.0，w = 62.0；`-test.timeout` = 30 + 5 = 35 s。记 g0，建管道；Start 成功，关闭 w，读取 pid，建立控制锁与回收标记，启动唯一 Wait 任务与复制任务 | 距 d_g 60.0 |
| 22.4 | B | 内层进程开始；内层的 T 不早于 57.4 | 距 d_i 29.6 |
| 22.5 | B | `P5.layout.1` 开始（δ = 0.5 s）：准入 52.0 − 22.5 = 29.5 ≥ 12；d = 34.5；obsEnd = 28.5 | — |
| 23.0 | B | 布局 confirmed，`calls` 0，收尾已核实 | 距 d_i 29.0 |
| 23.0 | B | `P5.w.exit7.1`（b = 0.5 s，h ≈ 0）：准入 52.0 − 23.0 = 29.0 ≥ 12；d = 35.0；obsEnd = 29.0。第一次真实调用为 X1，接缝换成（−1，`EINTR`）；不读第二次；`calls` 1 | — |
| 23.3 | B | 目标 invalid，理由恰好一条；组的确认与回收截取到 35.0；收尾已核实 | 距 d_i 28.7 |
| 23.3 ~ 23.4 | B | 内层的最终收尾（截止 55.0）：没有剩余；内层退出 1 | — |
| 23.4 | B′ | Wait 任务在锁外看到僵尸（W1）；取控制锁，置“开始回收”，释放（W2）；记下观察（W3）；不持锁调用 `Cmd.Wait`，返回退出码 1（W4）；写入状态，发布完成（W5、W6）。复制任务读到 EOF 并结束（没有后代持有写端）。门禁的 `until(57.0)` 返回已完成，不需要终止请求；此后才从快照读取退出码，读取输出并解析 | 距 d_g 58.6 |
| 23.4 ~ 23.5 | B′ | 普查（截止 min(28.4, 82.0) = 28.4）：没有存活的进程 | — |
| 23.5 ~ 23.6 | B′ | 记录的读取、见证、命中 1、登记；补救 not-needed | — |
| 23.6 ~ 23.7 | B′ | 保险清理（截止 min(33.6, 82.0) = 33.6）：nothing left；没有移交项；门禁 pass | 距 d_g 58.3；距 D 156.3 |
| 23.8 ~ 25.3 | B′、B | `control.p5-w-exit7`：准入 180 − 23.8 = 156.2 ≥ 60；d_g = 83.8，d_i = 53.8。内层同上，目标两次读取都为 X1，`calls` 2，confirmed，内层退出 0；外层同上；门禁 pass | 25.3 时距 D 154.7 |
| 25.3 ~ 25.4 | A | `TestP5` 结束；最终收尾（截止 210）：没有移交项 | — |

**时间线二甲：可靠退出与 Wait 都已完成（输出在 w 被停止）**

前提同上，门禁在 22.0 开始：d_g = 82.0，d_i = 52.0，k = 57.0，w = 62.0。本时间线的条件写在事件中；它不是终止请求之后的无条件上界。

| 时刻 | 角色 | 事件 | 截止与剩余 |
|---|---|---|---|
| 22.4 ~ 23.0 | B | 内层开始；`P5.layout.1` confirmed，它的最终结果已写入输出 | 距 d_i 29.0 |
| 23.0 | B | `P5.w.exit7.1` 开始，d = 35.0。为推演，设第一次经 E2 的调用阻塞在系统调用中，不返回（原因不论）；接缝没有到达 | — |
| 35.0 | B | 目标的 d 到期。阻塞的调用不受用户态期限的中断：目标没有结束，也没有最终结果 | — |
| 52.0 | B | d_i 到期。测试的 goroutine 仍阻塞，内层的最终收尾不能开始 | — |
| 57.0 | B′ | k：门禁的 `until(57.0)` 到期，Wait 任务的快照没有“开始回收”。门禁按 R0 登记并启动请求任务 R1，自己不取控制锁；记 T（`inner: … did not end by itself by d_i + 5 s`）。R1 取控制锁，标记未开始，`kill` 返回 0，释放控制锁，写入结果并发布。门禁以 `until(62.0)` 查询 R1：已完成，请求没有错误。这只是请求 | 距 d_g 25.0 |
| 57.1 | B′ | 条件：Wait 任务在锁外看到内层成为僵尸；取控制锁，置“开始回收”，释放；不持锁调用 `Cmd.Wait`，返回内层被信号 9 终止；写入状态并发布。门禁以 `until(62.0)` 查询 Wait 任务：已完成。退出与 Wait 都已确认 | — |
| 57.1 ~ 62.0 | B′ | 条件：内层留下的后代 X 继承了写端，复制任务没有 EOF；门禁以 `until(62.0)` 查询复制任务 | — |
| 62.0 | B′ | w：复制任务未结束。登记并启动停止任务（关闭读端），做一次不等待的查询：复制任务已返回（`file already closed`），停止任务已完成。已读到的输出照样保存并解析：其中有布局的最终结果，没有目标的最终结果。输出不完整记 T | 距 d_g 20.0 |
| 62.0 ~ 62.3 | B′ | 普查（截止 min(67.0, 82.0) = 67.0）：找到带本轮标记的 X | 距 d_g 19.7 |
| 62.3 ~ 62.4 | B′ | 记录的读取；X 由普查证明属于本轮 | — |
| 62.4 ~ 62.6 | B′ | 补救（截止 min(72.4, 82.0) = 72.4）：重新核对身份与标记之后结束 X，确认结束；补救 succeeded，不恢复结论 | 距 d_g 19.4 |
| 62.6 ~ 62.7 | B′ | 见证、命中 0（`hit: the fault never reached its site`）、登记 | — |
| 62.7 ~ 62.9 | B′ | 保险清理（截止 min(72.7, 82.0) = 72.7）：nothing left。门禁 pass = false：工具错误 `inner` 与 `hit`，输出不完整，目标没有结果；Final ended；没有移交项 | 距 d_g 19.1；距 D 117.1 |
| 63.0 | B′ | 对照照常准入：180 − 63.0 = 117.0 ≥ 60（d_g = 123.0）；第一个门禁不重跑 | — |

**时间线二乙：到截止仍未确认退出，Wait 未完成**

前提同上。从 57.0 起：

| 时刻 | 角色 | 事件 | 截止与剩余 |
|---|---|---|---|
| 57.0 | B′ | k：同时间线二甲：门禁登记并启动请求任务 R1，自己不取控制锁；R1 在控制锁内发出 SIGKILL，没有错误，释放之后发布结果；记 T | 距 d_g 25.0 |
| 57.0 ~ 62.0 | B′ | 条件：Wait 任务一直没有看到僵尸（停在 W1 的读取与重试中，不持有锁），`Cmd.Wait` 没有开始；内层仍持有写端，复制任务没有 EOF。门禁以 `until(62.0)` 查询，不阻塞在完成信号上，也不获取控制锁 | — |
| 62.0 | B′ | w：退出未确认，记 T（`inner: exit not confirmed by d_i + 10 s; SIGKILL requested at 57.0 without error`）。不读取 `ProcessState`，退出码 unknown。登记并启动停止任务（关闭读端），做一次不等待的查询：设复制任务已返回，已读到的部分保存并解析，输出不完整记 T；若未返回，复制任务与停止任务也待移交。`Cmd` / `Process` 与控制锁、唯一 Wait 任务待移交；R1 已完成，结果留在记录中 | 距 d_g 20.0 |
| 62.0 ~ 62.3 | B′ | 普查（截止 67.0）：只找到内层本身；它不按 PID 动作 | 距 d_g 19.7 |
| 62.3 ~ 62.4 | B′ | 记录的读取 | — |
| 62.4 ~ 72.4 | B′ | 补救（截止 72.4）：按 R0，R1 已完成、标记仍未开始，登记并启动请求任务 R2；R2 在控制锁内再发一次 SIGKILL（标记之前，所以安全），没有错误，释放之后发布结果。然后以 `until(72.4)` 查询 R2 与 Wait 任务。到期 Wait 任务仍未完成：补救 failed（still-alive），Final unknown | 距 d_g 9.6 |
| 72.4 ~ 72.5 | B′ | 见证、命中 0、登记 | — |
| 72.5 ~ 82.0 | B′ | 保险清理（截止 min(82.5, 82.0) = 82.0）：普查仍只找到内层。按 R0，R2 已完成、标记仍未开始，登记并启动请求任务 R3，过程同 R2；以 `until(82.0)` 查询 R3 与 Wait 任务。到 82.0 仍未确认，记 T（`insurance`） | 距 d_g 0 |
| 82.0 | B′ | 门禁返回 pass = false，Final unknown。移交：`Cmd` / `Process` 与控制锁、唯一 Wait 任务，以及未结束的复制或停止任务，交给最终收尾。R1 ~ R3 都已完成，结果留在记录中。没有其他已证明的进程 | — |
| 82.1 | B′ | 对照照常准入：180 − 82.1 = 97.9 ≥ 60（d_g = 142.1）；第一个门禁不重跑 | — |
| 84.0 | A | 设对照在 84.0 结束。`TestP5` 的运行都已结束，最终收尾开始（截止 210）。按 R0，标记仍未开始、没有 pending 的请求任务：登记请求任务 R4 到最终收尾的持有表，再启动。以 `until(210)` 查询 R4 与移交的任务；按它的标记普查（同一截止）。最终收尾不获取控制锁 | 距最终截止 126.0 |
| 设 95.0（结局一） | A | 条件：Wait 任务在锁外看到僵尸；取控制锁，置“开始回收”，释放；不持锁调用 `Cmd.Wait` 并返回；写入状态并发布。最终收尾对 Wait 任务的 `until(210)` 返回已完成：记为“晚到的结束”（时刻与状态），不改变第一个门禁的 T；之后不再向它发信号。其余移交项按同一截止处理 | 距最终截止 115.0 |
| 210（结局二） | A | 条件：到 210 仍未确认：快照仍为 pending，汇总记 unknown，P-5 为 T，按 §16.12 报告 | 0 |
| 两种结局之后 | A | `TestP5` 随即返回，不再等待；仍未结束的任务留在进程中 | — |

最坏的变体：外层每一步都用满名义上限时，普查 62.0 ~ 67.0，补救 67.0 ~ 77.0，保险清理 77.0 ~ 82.0，只得到 5 s；结局与上表相同。门禁不在 d_g 之后另取期限。

**时间线二乙的变体：请求任务停在控制锁内**

前提同上。这一行的条件是为推演所设，不是预测；它说明请求在等锁或系统调用时，持有者仍按既有的截止返回。

| 时刻 | 角色 | 事件 | 截止与剩余 |
|---|---|---|---|
| 57.0 | B′ | k：门禁按 R0 登记并启动 R1；记 T。条件：R1 取得控制锁，进入 `kill` 之后不返回，一直持有控制锁 | 距 d_g 25.0 |
| 57.0 ~ 62.0 | B′ | 门禁以 `until(62.0)` 依次查询 R1、Wait 任务与复制任务。门禁不取控制锁，所以不会停在加锁上。条件：若 Wait 任务此间看到僵尸，它停在 W2 等控制锁，`Cmd.Wait` 不开始 | — |
| 62.0 | B′ | w：三个查询都到期，只取快照：R1 pending（请求未完成），Wait 任务 pending，退出码 unknown；记 T。登记并启动停止任务（关闭读端），做一次不等待的查询 | 距 d_g 20.0 |
| 62.0 ~ 82.0 | B′ | 普查、补救、见证与保险清理的顺序与截止同二乙。补救与保险清理按 R0：R1 仍 pending，不启动新的请求任务，只以各自的截止查询 R1 与 Wait 任务。到 82.0 仍 pending | 距 d_g 0 |
| 82.0 | B′ | 门禁返回 pass = false，Final unknown。移交：`Cmd` / `Process` 与控制锁、R1、唯一 Wait 任务，以及未结束的复制或停止任务 | — |
| 84.0 ~ 210 | A | 最终收尾按 R0：R1 仍 pending，不启动新的请求任务，只以 `until(210)` 查询 R1、Wait 任务与其余移交项。R1 晚到完成、Wait 任务随后完成的，记为“晚到”，不改变门禁；到 210 仍 pending 的，记 unknown，P-5 为 T。`TestP5` 随即返回 | — |

**不完整**：有未执行的运行时，P-5 不算完整通过；不减少次数、不扩大预算、不重跑。

### 16.9 归属、收尾与失败记录

- 复用 P-3 的工具与子进程登记、身份核对与独立的外层普查；每个 `Cmd`、文件、观察器与后代都有所有者。
- 先保存被测的状态（`WP25P` 行），再补救；归属未知时不做任何动作；补救的成功不恢复结论。
- 内层没有打印最终结果时，保留错误与正式的非零结果。
- P-5 门禁内层的结束、等待与移交（五个事件、截止、发信号的规则与所有权表）见 §16.8（第三稿）；控制锁、请求任务与状态锁的顺序见 §16.8（第四稿）。

### 16.10 workflow 与元数据（在 P-5 的原型 PR 中改动）

- **工具链**：setup-go 由 `go-version-file` 改为 `go-version: 1.27.3`（四个作业都改）；作业的环境加 `GOTOOLCHAIN=local`。
- **macOS 的两个作业**：
  - 测试步骤加 `CGO_ENABLED=0`；
  - 增加 §16.4 的步骤（vet、`go test -c`、`go version -m`、`go tool nm`、`nm -m`），各自的输出原样进入日志。
- **Ubuntu、Windows**：只随工具链的固定而变。
- **元数据**：
  - 沿用 WP25META：OS 与内核、镜像、Xcode 与 SDK、`hw.machine`、`proc_translated`、实际检出的 SHA；
  - 增加 `go version`、`CGO_ENABLED`、`GOTOOLCHAIN`、构建设置与跳板的符号。
  - 环境未知时不当作健康。
- **触发**：workflow 的触发、并发与“没有手动触发”的政策不变；一个代码点只有一次诊断，不重跑。

### 16.11 结果模板

- **根层**：每次运行一行 `WP25RESULT`（verdict、R、N、M、理由、清理、启动与工作量）。
- **结果的字段**（第二稿按 Review 107 N2 修订）：第一稿写的“增加 `kind`（`static` / `native`）”会与既有的字段冲突，撤回。
  - **`kind` 保留原义**：实验的种类，取值仍为 `normal` / `counterexample` / `fault-injection` / `refusal`（`main_test.go:83` ~ `90`），不扩展，也不覆盖。
  - **新字段 `evidence`**（证据的类别，JSON 名 `evidence`），取值：
    - `static`：`P5.static.entry`；
    - `layout`：`P5.layout.*`，根层与内层都是；
    - `native`：39 次原生；
    - `seam`：两个门禁内层的目标，包括对照。

    其余实验与角色 C 的结果不带这个字段（省略），解析时不当作任何一类。workflow 的静态步骤（`P5.static.toolchain` 等）不产生 `WP25RESULT`，它们的结论只进入日志与 `WP25P5`。
  - **新字段 `calls`**（经 E2 的调用次数，JSON 名 `calls`）：
    - 进入共享的最终结果类型 `result`（`main_test.go:101` 起），类型可以区分缺失与 0（例如指针）；
    - 每次运行开始时清零的原子计数，由 `waitidCall` 递增；
    - 在收尾之后、打印最终 `WP25RESULT` 之前读取，所以只有收尾之后的最终结果携带它。`WP25P` 行可以打印，但不作为依据。
  - **内外层的消费**：
    - 带截止的 `runInner` 变体按既有的路径把 `WP25RESULT` 解码为 `result`，所以 `calls` 与 `evidence` 进入 `results`、`all`，并随门禁行的 `inner` 保存；
    - 门禁按 §16.6 的 `exact` 断言它们。
  - **必需的结果缺少 `calls` 时**，记为 T（§16.7），不当作 0。
  - **逐项核对的计划次数**：布局 0、门禁的故障目标 1、健康 2、错误之后的恢复 3；`P5.static.entry` 0。规则次数与计划次数的关系见 §16.7。
  - 角色 C 的结果也经过 `waitidCall`，可以带 `calls`；旧 P-3 的门禁不断言它，判定不变。
- **门禁**：沿用门禁行（含 `inner`）。增加 d_g、d_i、k、w，以及内层的知识状态（第三稿）：终止请求（是否发出、时刻、错误）、退出是否确认（状态或 unknown）、`Cmd.Wait` 是否完成、输出是否完整，以及移交的项。第四稿补充：每个请求任务的结果（已发出、不需要或 pending）与登记它的持有者。
- **汇总**：每个作业一行 `WP25P5`，内容为：
  - 工具链与 cgo；
  - 静态检查的逐项结论；
  - 原生的 confirmed 数与应有的 39；
  - 门禁与对照的结论；
  - 模型检查（`TestP3Decode`、`TestP3Parse` 等纯函数测试）的结论；
  - 未执行数、D 的值；
  - H4、H6 的限制声明。
- **报告**（原型文档的结果节）：
  - 两个架构分开；
  - 静态、原生、接缝、模型四类分开计数，不合计；布局（`evidence` 为 `layout`）另列为前提，不计入这四类；
  - 下面两条声明原样写入：

    > H4：静态依据是固定的 Go 1.27.3 过滤源码；动态覆盖的是所测的真实错误（`ECHILD`、`EINVAL`）与错误之后的健康恢复；没有动态取样的是健康调用时过滤之前的 errno 是否非零。

    > H6：只有接缝证据（消费者的不重试与失败）；真实的 EINTR 穿过跳板没有原生验证。

### 16.12 会改变设计的结果与停止的条件

按负责人 Q-4 的决定：P-5 出现反证、或无法取得有效证据时，报告并交负责人重新决定。不自行切换入口、扩大预算、降低判定条件或重跑。下列结果属于这种情况：

- H1 或 H2 不成立（构建、链接、符号或加载失败）；
- H3 ~ H5 被反证（原生值、真实错误或健康恢复与预测不同）；
- 调用次数与计划不同。
- 第二稿补充（Review 107）：
  - 经 E2 的调用出现协议失效（§16.7 的 X3 ~ X5）；
  - 实际次数与规则次数不同，或必需的结果缺少 `calls`；
  - 因截止而未执行；内层到 k 仍未自行结束，或到 w 仍未确认退出、输出不完整；保险清理到 d_g 仍未确认；最终收尾到截止仍有未确认的移交项（第四稿补充：包括仍 pending 的请求任务）。

### 16.13 不在范围之内

- 生产 Runner、生产的 C-9 lint；
- 与入口 E1 的对照运行；
- 原生的 EINTR、pthread 的取消路径、部署的 libSystem 的逐字节核对；
- 过滤之前 errno 的取样（负责人决定 Q-4 (a) 时认可了不做）；
- 其他 macOS 版本与物理的 Apple Silicon；
- Review 101 的 N1、N2 与维护事项。Review 107 的 N1、N2 已在第二稿的 §16.7、§16.11 中处理，与 Review 101 的 N1、N2 无关。

### 16.14 P-5 的实施：登记（在写任何原型代码之前提交）

- **授权**：负责人 2026-10-30 采纳 Review 110 之后的条件授权（review-resolution §136 的第二项）。它在第一阶段全部成功之后生效：
  - PR #43 以 merge commit 合并为 `021c956c33a65cc17ad050bda34883df355adcd0`，锁定最终 HEAD `13cd4b9`；父提交依次为 `0442f14`、`13cd4b9`，树 `c6b2654` 与最终 HEAD 相同，受审提交 `35ecd79` 是它的祖先；
  - master push CI run 41050553619：push，attempt 1，正式 success（8/8），八个作业都取得并检出 `021c956`。

  另开独立的 test-only 原型 PR，不合并；Auto-fix 与 Auto-merge 保持关闭；等待代码与结果的独立审查。
- **固定的身份**：
  - **实现起点**：master `021c956c33a65cc17ad050bda34883df355adcd0`（树 `c6b26542bb095cc0e6be41d5227d334dcc79178d`），分支 `test/wp2.5-prototype-p5`；
  - **所依赖的计划**：本节之前的 §16（第四稿），受审提交 `35ecd7984f4d4624f43b32e2e618c14e0257ee80`（Review 110 的对象）。PR #43 的收尾 `13cd4b9` 只改了 §16 的标题与 §16.1 的一条状态（`git diff 35ecd79 021c956 -- docs/wp2.5-prototype.md` 只有开头一行、§16 的标题与 §16.1 一行），§16.2 ~ §16.13 的技术内容逐字未变；
  - **Review 110 的限定**：唯一回收与 PID 的安全是受控拓扑中的条件证明；短临界区不是硬实时的保证；移交、测试返回或系统收养都不等于资源已经释放；实现审查须核对新增的实际调用点兑现这些前提（审查 §3 ~ §5、§8.3）；
  - **工具链与依赖**：§16.1 的表（`go1.27.3`、`golang.org/x/sys v0.49.0`、不新增依赖），两个 macOS 作业 `CGO_ENABLED=0`；
  - **平台**：`macos-latest`（arm64）与 `macos-26-intel`（x86_64），同一个固定代码点。
- **授权的范围**（§136）：
  - `prototype/wp25` 下既定的 `e2waitid` 子包、两个架构的跳板、`waitidCall` 的接线，以及计划要求的辅助协议（§16.3 ~ §16.11）；
  - 现有诊断 workflow 中 §16.10 明定的改动，以及触发分支的精确匹配（与 P-3 相同的既有政策：只有推送到本分支、且改动原型或 workflow 时运行，没有手动触发）；
  - 必要的结果记录与状态文档。
- **必须兑现**（§136 与 Review 110 §8.3）：W / R 的顺序、两类锁的边界、唯一回收、请求先登记、每个内层最多一个 pending 请求、状态发布、稳定输出的读取、持续错误与所有权移交；预算、期限、次数、角色 C、`exact` 与归属规则都按 §16.5 ~ §16.8。
- **诊断的政策**：
  - 代码只在完成本地可做的核对之后推送一次；那一次推送的头提交即为固定代码点，由既有的触发方式取得一次诊断。不手动触发，不重跑。
  - 出现反证、工具失效、未执行、无命中、补救失败或 unknown 清理时，按计划记录与收尾，然后报告，交负责人决定；不扩大预算、更换入口、减少次数或放宽判定，也不为取得绿色改动预测或代码。
  - 结果回填只增加文档提交，并核对原型与 workflow 相对固定代码点没有变化。
- **本节在写任何原型代码之前提交**。实施若须偏离 §16，先在 §16.15 登记偏离与理由（在代码提交之前），再改代码。
- **不授权**：规范冻结、生产实现、G5 整体满足、WP2.5 / M2 的验收。

### 16.15 写代码时登记的补充与偏离（在代码提交之前）

下面各项是写代码时确定的，在代码提交之前登记。它们都不改变 §16.4 ~ §16.12 的预测、次数、期限、角色 C、`exact`、归属规则或判定，也不放宽失败关闭。没有偏离 §16 的技术方案。

- **补充 1：协议核心与本机的模型检查**。
  - 门禁内层协议的平台无关部分（两类锁、W1 ~ W6、R0 ~ R5、`until`、复制与停止任务、调用形式的分类与调用次数的判定）在 `p5proto_test.go`；darwin 的接线（`procInfo`、`Cmd.Wait`、`syscall.Kill`）在 `p5gate_darwin_test.go`。
  - 两者带构建约束 `darwin || wp25p5local`：诊断的 macOS 作业照常构建；Ubuntu、Windows 作业不构建，§16.10 的“只随工具链的固定而变”不受影响。标签 `wp25p5local` 只供作者在本机用假对象运行模型检查（`p5proto_check_test.go`），不是诊断的证据。
  - 模型检查：`TestP5ProtoUntil`、`TestP5ProtoWaitOrder`（W2 先解锁再 W4；W4 阻塞时请求立即取得控制锁并判为“不需要”；快照不等待；失败的读取保留）、`TestP5ProtoRequestHung`（请求停在控制锁内时，查询按截止返回，R0 不另起请求，W2 等锁，`Cmd.Wait` 不开始）、`TestP5ProtoRequestSerial`（请求先登记、后执行）、`TestP5ProtoCopyStop`、`TestP5ProtoErrs`、`TestE2Forms`、`TestP5CallsCheck`。它们与 `TestP3Decode`、`TestP3Parse` 一起登记结论，进入 `WP25P5` 的模型一项。
- **补充 2：符号检查的写法**（作者在本机用 Go 1.27.1 交叉构建的两个架构的测试二进制上取得，不是 1.27.3 的证据）：
  - `go tool nm` 中跳板的地址变量为 `github.com/…/e2waitid.libc_waitid_trampoline_addr`（另有静态的 `_libc_waitid_trampoline`），动态导入为 `U _waitid`。检查取“名字含 `e2waitid.libc_waitid_trampoline` 的行”与“`U _waitid` 的行”，即 §16.4 的预期。
  - Xcode 的 `nm -m` 按 §16.4 取“`(undefined) external _waitid (from libSystem)`”；本机无法预先核对，以作业的输出为准，不成立为 T。
- **补充 3：静态步骤的落点**（§16.4、§16.10）：
  - macOS 的两个作业增加一个步骤 `P-5 static checks`：带 `CGO_ENABLED=0`，依次做工具链、vet、`go test -c`、`go version -m`、`go tool nm`、`nm -m`，各自的输出原样进入日志，结论写入一个文件，经 `WP25_P5_STATIC` 交给测试的 `WP25P5` 行。
  - 工具链不成立时步骤失败，之后的步骤不运行（T，不继续）；其余不成立时记 T，测试照常运行，由最后的步骤 `P-5 static verdict` 使作业失败。
  - 既有的 `Vet` 步骤不变。
  - 静态检查的对象是 `go test -c` 的二进制；测试步骤仍按 §16.10 用 `go test`（同一工具链与构建设置），`P5.static.load` 是 `TestP5` 在这个二进制中开始运行。
- **补充 4：共享代码的改动**（行为不变）：
  - `gate_test.go`：把内层输出的解析（`parseInner`）与内层判定的一维（`verdictOf`）从 `runInner`、`runGate` 中提出来，P-5 的门禁复用；`gateResult` 增加省略的 `p5` 字段（P-5 门禁自己的记录）。角色 C 的判定与输出不变。
  - `main_test.go`：`result` 增加省略的 `evidence` 与指针 `calls`；`scen` 增加 P-5 运行的截止 `cap`（结束阶段的有界等待取 min(自身上限, d)）与收尾之后、判定之前执行的 `final`；`TestMain` 在外层调用 `p5Summary`（非 darwin 为空）。其他运行的这些字段为零值，行为不变。
  - `p3_darwin_test.go`：`p3Run` 增加 `p5`；`admit`、`obsEnd`、`limits`、`layoutPremise`、`zombie` 与 `reap` 的等待只在 `p5` 非空时按 §16.8 的角色走另一分支，P-3 自己的运行不变。证据行增加省略的 `E2`。
  - `witness_check_test.go`：`TestStartPoints` 允许 P-5 门禁启动内层的那一行。
- **补充 5：结果与记录的写法**（§16.11）：
  - 每一次经 E2 的调用记在运行的 `WP25P` 行的 `E2` 中：参数、r、errno（数值与文字）、缓冲区的原始字节、哨兵是否仍在、形式、接缝的标记、分类与原因。
  - 健康调用的原生反证记为预测“`0 / 0 / X1`”与实际的不符；构造的错误记为预测“`-1 / errno / X2`”与实际的比较；协议失效与接缝站点的真实失败记为 T，持续到终判。
  - 每个作业另有 `WP25P5META`（工具链、`CGO_ENABLED`、`GOTOOLCHAIN`、S / T / D、构建设置）、`WP25P5FINAL`（最终收尾）与 `WP25P5`（汇总）行；静态步骤打印 `WP25P5STATIC` 行。门禁行增加 `p5`（d_g、d_i、k、w、内层的身份、知识状态、退出、输出与移交的项）。
- **补充 6：P-5 的布局前提只看 P-5 自己的布局运行**（`p5Layouts`，与 P-3 的列表分开），所以 `TestP3` 的布局结论不进入 P-5 的前提。
- **补充 7：唯一 Wait 任务的细节**：
  - W1 的读取间隔为 5 ms，不持有锁；同一文字的读取失败只记一次并计数，不丢失、不清除。
  - `Cmd.Wait` 返回的 `*exec.ExitError` 是退出状态，不是调用的失败；状态在 W5 由 Wait 任务读取 `cmd.ProcessState` 取得。
- **补充 8：截止与移交的细节**：
  - 在 w 启动停止任务之后只做一次不等待的查询；复制任务这时尚未返回的，没有可以安全读取的输出（T），复制任务与停止任务移交。
  - 补救与保险清理是 P-5 自己的（`p5Remediate`、`p5Insure`）：截止为 min(当前 + 10 s, d_g)，不按 PID 动作内层；保险清理到 d_g 仍未确认时，除了 `runGate` 既有的矛盾规则，另记 `insurance` 的问题（§16.8 的“到 d_g 仍未确认结束时为 T”）。
  - 只有未确认的部分才移交最终收尾（时间线一的“没有移交项”）；最终收尾按移交门禁的标记普查。
  - 内层的 `TestP5` 也注册截止 d_i + 3 s 的最终收尾；内层没有移交项，它只打印 `WP25P5FINAL`。

### 16.16 结果（固定代码点 `0213fb7`，诊断 run 41055799521）：invalid（T），交负责人决定

**身份**：

| 项 | 值 |
|---|---|
| 所依赖的计划 | 受审提交 `35ecd7984f4d4624f43b32e2e618c14e0257ee80` 的 §16 第四稿（合并进 master `021c956` 之后技术内容逐字未变，§16.14） |
| 实现起点 | master `021c956c33a65cc17ad050bda34883df355adcd0` |
| 登记 | `83854cb`（§16.14）、`aeb0a4c`（§16.15），都在代码提交之前 |
| 取证代码点 | `0213fb706e29130110af7ad8971dd8bbd94dd1d4`（树 `b097f99421c58ae8c3eaba8ad82bdea8e1e8f624`），一次推送的头提交 |
| 诊断 | WP2.5 prototype run 41055799521，push，attempt 1，**failure**；没有手动触发，没有重跑 |
| 实际检出 | 四个作业都检出 `0213fb7`（`checkout -B test/wp2.5-prototype-p5`；`WP25_SHA` 与 `WP25META` 的 `sha` 相同） |
| 工具链 | 四个作业 `go version go1.27.3`；两个 macOS 作业 `go env`：`go1.27.3`、`0`、`local` |
| 与最终 PR HEAD 的关系 | 本节与状态的回填只增加文档提交；`prototype/` 与 `.github/` 相对 `0213fb7` 没有变化（PR 回执登记核对的命令与结果） |

**作业的结果**：

| 作业 | 结论 | 说明 |
|---|---|---|
| `macos-latest`（arm64） | failure | “Prototype”（`go test`）步骤 success；最后的“P-5 static verdict”失败：`P5.static.symbols` 为 T |
| `macos-26-intel`（x86_64） | failure | 同上 |
| `ubuntu-latest` | success | 53 个场景 confirmed，门禁 111/111，组合 13/13（只随工具链的固定而变） |
| `windows-latest` | success | 53 个场景 confirmed，门禁 92/92，组合 13/13 |

**唯一的 T：`P5.static.symbols`（两个架构相同）**

- 两份输出的原样记录（arm64；x86_64 只有地址不同）：
  - `go tool nm`：`1001ad580 T _libc_waitid_trampoline`、`U _waitid`、`100424058 R github.com/XKHoshizora/specmetry/prototype/wp25/e2waitid.libc_waitid_trampoline_addr`；
  - `nm -m`：`0000000100424058 (__DATA_CONST,__rodata) non-external _github.com/…/e2waitid.libc_waitid_trampoline_addr`、`00000001001ad580 (__TEXT,__text) non-external _libc_waitid_trampoline`、`(undefined) external _waitid`。
- 检查记为 T 的原因有两处，都在检查与预期的写法，不是被测入口给出的反证：
  1. **检查脚本的缺陷**：步骤以 `bash -e -o pipefail` 运行。条件 `printf '%s\n' "$n1" | grep -q …` 中，`grep -q` 找到第一行就退出，`printf` 写入 `go tool nm` 的大段输出时得到 `Broken pipe`（日志：`printf: write error: Broken pipe`），`pipefail` 使管道的结果为失败，所以第一个条件即为假。作者本机的模拟没有覆盖 `pipefail`。
  2. **`nm -m` 的输出没有库的归属**：§16.4 预期“`_waitid` 为 `(undefined) external`，来自 libSystem”，检查写作 `(undefined) external _waitid (from libSystem)`；实际输出是 `(undefined) external _waitid`，没有 `(from libSystem)`。所以即使没有第 1 处，第三个条件也不成立。为什么没有库的归属（例如二进制是否为两级命名空间），本次诊断没有取得，原因未知。
- 按 §16.4，这一项不成立为 T（H1 的符号部分没有按计划取得证据）；按 §16.7 的终判，有任何静态检查不成立即为 invalid（T）。原始输出中可以看到的符号不改变这一判定，也不由作者改判。
- 这属于 §16.12 的停止条件（构建、链接、符号或加载的证据不成立）。按 Q-4 (a) 与 §136：报告，交负责人决定；不改代码或检查重跑，不放宽判定。

**其余结果**（两个架构分开；四类证据分开计数，不合计；布局另列）：

| 类别 | arm64 | x86_64 |
|---|---|---|
| 静态（§16.4，7 项） | 6 项成立：`toolchain`、`vet`、`build`、`settings`（`build CGO_ENABLED=0`）、`entry`（`P5.static.entry` confirmed，`calls` 0）、`load`（`TestP5` 开始运行）；**`symbols` 为 T** | 同左 |
| 原生（§16.5） | 39/39 confirmed：`calls` 2 的 33 次、`calls` 3 的 6 次；未执行 0 | 39/39 confirmed（同左） |
| 接缝（§16.6） | `p5.eintr-once` pass：内层 exit 1，命中 1（`P-5 waitid: (0, 0) -> (-1, EINTR)`），目标 invalid、理由恰好一条 `observation failed: waitid 1: interrupted system call`，`evidence` seam、`calls` 1，清理 verified；`control.p5-w-exit7` pass：exit 0，目标 confirmed、`calls` 2 | 同左 |
| 模型（§16.11） | 10/10 pass：`TestP3Decode`、`TestP3Parse`、`TestP5ProtoUntil`、`TestP5ProtoWaitOrder`、`TestP5ProtoRequestHung`、`TestP5ProtoRequestSerial`、`TestP5ProtoCopyStop`、`TestP5ProtoErrs`、`TestE2Forms`、`TestP5CallsCheck` | 10/10 pass |
| 布局（前提，另列） | `P5.layout.1` ~ `3` confirmed，`calls` 0；两个门禁内层的 `P5.layout.1` 也 confirmed、`evidence` layout、`calls` 0 | 同左 |

- **真实错误与健康恢复**（`P5.err.*`，每个架构各 3 次）：错误调用都是 X2 并与预测一致——`ECHILD`（10，`waitid(P_PID, 本进程, 0x25)`）与 `EINVAL`（22，`waitid(P_PID, 子进程, 0)`），缓冲区的哨兵仍在；之后的两次健康读取都是 X1，与 `w.exit7` 的预测相同。
- **门禁内层的协议**（两个架构共四次门禁）：都没有请求任务（内层在 k 之前自行结束，Wait 任务在 W3 记下观察并完成 `Cmd.Wait`），输出完整（复制任务在 w 之前读到 EOF），补救 not-needed，Final ended，保险清理 nothing left，没有移交项；最终收尾（截止 D + 30 s）为 “nothing handed over”。这只是正常路径的执行，挂起、未确认与移交的分支没有在原生运行中出现，只有模型检查覆盖（补充 1）。
- **期限**：arm64 的 D = S + 180 s（S 12:35:23Z，T 12:45:35Z）；x86_64 同样取 S + 180 s。39 次原生合计约 2.0 s（arm64）与 2.4 s（x86_64），单次最长 213 ms 与 290 ms；门禁 320 ~ 536 ms。都不是入口 E2 的时限保证。
- **回归**（另列，不计入上表）：P-3 的门禁 22/22 pass，其中 W 与 C 的 7 个门禁与 2 个对照都 pass；`TestP3` 63/63 confirmed，都经入口 E2。WP25SUMMARY：场景 139/139 confirmed，门禁 101/101，组合 16/16。

**限制声明**（§16.11，原样）：

> H4：静态依据是固定的 Go 1.27.3 过滤源码；动态覆盖的是所测的真实错误（`ECHILD`、`EINVAL`）与错误之后的健康恢复；没有动态取样的是健康调用时过滤之前的 errno 是否非零。

> H6：只有接缝证据（消费者的不重试与失败）；真实的 EINTR 穿过跳板没有原生验证。

**结论与待决**：

- 本次 P-5 **不是完整的通过**：终判为 invalid（T），唯一的 T 是 `P5.static.symbols`，原因是检查脚本的缺陷与 `nm -m` 输出没有库归属这两处；原生、接缝、模型与其余静态检查在两个架构上都按预测成立。
- 作者不宣布 G5、H1 或 P-5 满足，也不据此推断入口 E2 的符号与库的归属已经验证。
- 需要负责人决定（作者的建议见 review-resolution §138）：是否授权修正 `P5.static.symbols` 的检查（去掉 `pipefail` 下 `grep -q` 的提前退出，并重新确定“来自 libSystem”的证据形式，例如增加取得二进制标志与加载命令的步骤），以及是否在新的固定代码点取得一次新的诊断。修正之前不推送任何原型或 workflow 的改动。

### 16.17 Review 111 之后的修订准备：B1、B2 的最小修复与回归，符号检查的第二版（计划第四稿，Review 115 技术 GO）

#### 16.17.1 授权、依据与固定的身份

- **授权**：负责人 2026-10-30 采纳 Review 111（review-resolution §139）。本轮只授权纯文档的修订准备。
  - 不授权修改实现或 workflow，不授权取得新的诊断。
  - 本节通过审查，不代表 B1、B2 已在实现中关闭。
- **第二稿**（review-resolution §140）：
  - Review 112 对第一稿（提交 `691435a`，HEAD `691435af8db0fdbe387353f28481f597b89253d5`，树 `113d1ed470c29e0c457f92b756dba074edc37787`）为计划技术 NO-GO，有三项计划阻塞：
    - B-P112-1：带错完成的关闭任务可能不移交；
    - B-P112-2：A7 只核对绑定的起始地址；
    - B-P112-3：真实子进程回归没有写定自己的截止与失败出口的收尾。
  - 负责人只授权修订本节的计划。第二稿只修这三项与 Review 112 列明的精化（N-P112-1 等）。改动的位置：
    - §16.17.2：移交谓词、真实与报告的关闭结果，以及 R-B1-1、R-B1-2、R-B1-4；
    - §16.17.3：R-B2-2 的执行协议；
    - §16.17.4：A3、A4、A5、A7、A9、工具的身份、M5、共享缓存的说明与新的对照；
    - §16.17.5、§16.17.7、§16.17.8：同步。
  - 第一稿的全文在提交 `691435a` 中。原实现的 B1、B2 不因本稿关闭；首次诊断保持 failure / invalid（T）。
- **第三稿**（review-resolution §141）：
  - Review 113 对第二稿（提交 `598bb2f`，HEAD `598bb2f50e9d0adf44c28429f71e5a0a59af292b`，树 `51d638f9586c60e2f42ddf9df818622cabb96ccb`）为计划技术 NO-GO：
    - B-P112-1、B-P112-2 在计划层面关闭；
    - B-P112-3 的主要修订成立，残余 **B-P112-3-R**：持有者的终止请求没有任务化，每侧 Cleanup 的实际时机没有写定，截止之后没有持有者。
  - 负责人只授权定向修订本节，形成第三稿。改动的位置：
    - §16.17.3 R-B2-2：
      - 两个非并行的子测试；
      - 适用的前提；
      - 请求任务与 Wait 的串行；
      - 每侧收尾的时机与只执行一次；
      - 持有登记 `p5RealHeld`；
      - 出口表；
      - 五条时间线与没有 `t.Deadline` 时的可用量。
    - §16.17.4：Review 113 §4、§6.1 列明的小范围精化：
      - 基准映像与 scaled 指令的符号状态；
      - 段尾七例的偏移与剩余量；
      - 溢出例最早失败的规则；
      - count = 2 的两个合法位置；
      - 输出结尾的换行与数值行的切分。
    - §16.17.5、§16.17.8：同步。
  - B-P112-1、B-P112-2 的计划层面关闭保持，本稿不重新展开符号方法。第二稿的全文在提交 `598bb2f` 中。
- **第四稿**（review-resolution §142）：
  - Review 114 对第三稿（提交 `7930a93`，HEAD `7930a934ea2aed49ceed95d295fe3a38ba9798dc`，树 `7bd35cc625b675e94931fb19e7cdb7d6287efe50`）为计划技术 NO-GO：
    - 请求任务化、每侧 Cleanup、持续持有这三项主体修订成立；
    - B-P112-3-R 尚余 **B-P112-3-R/D**：收尾在等待或同步读取返回之后，缺少统一的截止复核，可能在已知过期之后新建请求或启动 Wait；
    - B-P112-1、B-P112-2 保持计划层面关闭。
  - 负责人只授权第四稿的纯文档定向修订。改动的位置：
    - §16.17.3 R-B2-2：
      - 新任务的准入与统一的到期出口；
      - 正常路径的第 3、4 步与收尾函数的步骤；
      - 出口表；
      - 时间线：甲、乙、丙、戊的措辞，新增己、庚；
      - 名义窗口与可用量（N-114-1）；
      - 持有登记的两种视图与回收的四种状态（N-114-2）。
    - 新增 R-B2-3 的模型预期（待实施）。
    - §16.17.4：scaled 不唯一对照与 vmsize < 8 对照的构造（N-114-3）。
    - §16.17.5、§16.17.8：同步。
  - 其余已认可的协议与符号方法保持，不增加实验类型、次数或预算。第三稿的全文在提交 `7930a93` 中。
- **Review 115**（review-resolution §143）：对第四稿（提交 `cd0d4f7`，树 `d844e67`）为计划技术 GO；B-P112-3-R/D、B-P112-3-R、B-P112-3 在计划层面收口，N-114-1 ~ 3 关闭。负责人授权按第四稿实施固定范围的修复与一次诊断（流程 (i)），实施的登记见 §16.18。本节的技术内容没有因此改动。
- **固定的起点**：

  | 项 | 值 |
  |---|---|
  | PR | #44（不合并） |
  | HEAD | `067526760e07be3da1d054c3da1939b39bf331dc` |
  | base | `021c956c33a65cc17ad050bda34883df355adcd0` |
  | 树 | `1ef9f0bb70269b9c501dcc1b37ed6443f94d1c77` |
  | 取证代码点 | `0213fb706e29130110af7ad8971dd8bbd94dd1d4`（树 `b097f99421c58ae8c3eaba8ad82bdea8e1e8f624`） |

  本轮的提交只改文档；`prototype/` 与 `.github/` 相对 `0213fb7` 没有变化。
- **依据**：
  - Review 111 §4（B1）、§5（B2）、§7（符号检查）、§8（N1 ~ N3）；
  - Review 112 §3（B-P112-1）、§4（B-P112-3）、§5（B-P112-2、N-P112-1、M5）、§6（计数与预算）、§7（流程）；
  - 本节之前的 §16.4、§16.7 ~ §16.9（第四稿）、§16.15 与 §16.16。
- **首次结果不变**：固定代码点 `0213fb7` 的诊断 run 41055799521 仍为 failure，终判 invalid（T）。§16.16 的局部证据及其限制保留原样，不回改。
- **不变的部分**：
  - 39 次原生、3 次布局与两个门禁，以及它们的预测与判定；
  - 预算（D、角色 B 与 B′ 的截止）、调用次数、角色 C、`exact`、归属规则；
  - Q-3 (a)、Q-4 (a)、H4 / H6 的限制声明与支持承诺。

  本节新增的检查与回归另列，不计入上面任何一项。

#### 16.17.2 B1：父端读端的关闭

**问题**（Review 111 §4，源码位置为 `0213fb7`）：

- `p5Launch` 创建管道（`p5gate_darwin_test.go:89`），把读端交给复制任务（`:150`）。复制任务读到 EOF 后返回，不关闭读端（`p5proto_test.go:297` ~ `307`）。
- 门禁只在复制任务到 w 仍未结束时，才启动停止任务（`p5gate_darwin_test.go:219` ~ `226`）。那里的一次快照，结果也被丢弃（`:224`）。
- 正常 EOF 的路径上没有任何关闭，也没有关闭错误的消费；模型对照却手工关闭了读端（`p5proto_check_test.go:266`）。
- §16.8 第四稿的 w 一行只写了“复制任务未结束时”启动停止任务。本节补正：每一条出口都关闭读端。

**修复**：

1. **唯一的关闭责任**：
   - 读端由 `p5Inner` 持有（新字段），不再只是门禁的局部变量。
   - 关闭只经 `p5Inner` 的唯一停止任务：
     - 方法 `closeRead()` 在停止任务已存在时，返回已有的任务；
     - 否则新建停止任务（完成信号、状态锁、结果字段），先登记到 `p5Inner`，再启动。
   - 持有者依次使用 `p5Inner`（§16.8 的持有原则），所以读端至多关闭一次。
   - 停止任务只调用一次 `Close`，额外记录调用次数与开始、结束的时刻，供回归核对。
   - **两个结果分开保存**（第二稿，B-P112-1）：停止任务在自己的状态锁内保存以下字段：
     - **真实结果**：真实的 `Close` 返回的错误；
     - **报告的结果**：通常与真实结果相同；只有 R-B1-2 的接缝在真实的 `Close` 返回 nil 之后，才把它换成注入的错误；
     - **是否注入**；
     - **完成标志**。

     由此定义**释放的知识**：
     - **已释放**：已完成，且真实结果为 nil；
     - **释放未确认**：未完成；或者已完成，但真实结果不为 nil；或者停止任务不存在。

     注入的错误只改变报告的结果，不改变释放的知识。
2. **每条出口的关闭**：

   | 出口 | 关闭由谁、何时 | 截止 | 结果的消费 |
   |---|---|---|---|
   | `os.Executable`、`os.Pipe` 或 `Cmd.Start` 失败（还没有复制任务） | `p5Launch` 同步关闭已创建的端（现状，不变） | — | 关闭错误记为 `inner:` 的 T；没有要移交的 |
   | 复制任务在 w 之前读到 EOF | 门禁对复制任务的 `until(w)` 查询返回、且复制任务已完成之后（这个时刻可能早于 w），立即调用 `closeRead()` | 以 `until(w)` 查询停止任务 | 见下面第 3 条 |
   | 复制任务在 w 之前因读错误返回 | 同上；读错误照旧记 T（`inner: reading the output failed`） | 同上 | 同上 |
   | 复制任务到 w 仍未结束 | 门禁在 w 调用 `closeRead()`（现状） | 对复制任务与停止任务各做一次不等待的查询（现状） | 同上 |
   | 启动身份读取失败 | 照常走上面三行之一（§16.8 的提前出口，不变） | 同上 | 同上 |
   | 移交给最终收尾 | 停止任务已在门禁中启动；万一没有，最终收尾先调用 `closeRead()` | `until(D + 30 s)` | 见下面第 5 条 |

   门禁对复制任务的 `until(w)` 查询返回之后（这个时刻可能早于 w），**总是**调用一次 `closeRead()`，不论复制任务是否完成。这里指的是这次查询返回之后，不是绝对时刻 w。所以 Start 成功之后的每一条路径，都有一个已登记的停止任务。
3. **门禁先消费关闭结果，再判定**：消费在复制与停止的查询之后、`parseInner` / `verdictOf` / `finishGate` 之前。
   - 停止任务已完成，报告的结果为 nil：记“已关闭”。
   - 停止任务已完成，报告的结果不为 nil（真实的错误或注入的错误）：记问题 `inner: closing the read end failed: <错误>`。
     - `inner` 属于 `toolClasses`，任何预期的拒绝都不豁免它；
     - 这一问题持续到终判，之后的成功不清除。
   - 停止任务未完成：记问题 `inner: the read end's close did not return by d_i + 10 s`。
   - **晚到的结果**：判定之后、移交之前停止任务才完成的，结果记入门禁行的 `p5` 记录，不改变已经成立的 T。晚到的成功也不恢复结论。

   门禁对关闭结果的消费，提为一个小函数（平台无关，见第 9 条），门禁与模型检查都调用它。
4. **移交谓词**（第二稿，B-P112-1；取代 `p5gate_darwin_test.go:476` 的 `pending`）：
   - **何时计算**：在保险清理之后、门禁返回之前，以当时的新快照计算（现在的位置）。
   - **移交的条件**：下面任何一条成立，`p5Inner` 都随移交项交给最终收尾：
     - Wait 任务未完成；
     - 复制任务未完成；
     - 有 pending 的请求任务；
     - **读端的释放未确认**（第 1 条的定义）：停止任务不存在、未完成，或者已完成但真实结果不为 nil。
   - **因此**：
     - 真实的 `Close` 错误，即使任务已经完成，也保留责任；
     - 判定时 pending、移交之前带着真实错误完成的，按新快照同样移交；
     - 只有真实结果为 nil 的完成才解除读端的责任。
   - **移交的内容**：移交的是停止任务及其持续的结果，不再调用 `Close`，也不新建停止任务（`closeRead()` 返回已有的任务）。
   - 这一谓词提为一个小函数（平台无关），门禁与模型检查都调用它。
5. **最终收尾对读端的消费**（同一个持续的结果）：对移交项的停止任务以 `until(D + 30 s)` 查询。

   | 截止时的快照 | 记录 | 计入 `final_T` |
   |---|---|---|
   | 仍未完成 | 该项的 `unknown`：`the read end's close has not returned` | 是 |
   | 已完成，真实结果不为 nil | 该项的 `unknown`：`the read end's close failed, release not confirmed: <错误>` | 是 |
   | 已完成，真实结果为 nil（晚到的成功） | 该项的 `outcome`：晚到的关闭成功，不改变门禁已成立的 T | 否 |
   | 已完成，真实结果为 nil，报告的结果为注入的错误（只在这一项因其他原因被移交时出现） | 该项的 `outcome`：被测的注入错误，真实的关闭已成功 | 否 |

   - 释放未确认的读端一定被移交（第 4 条），所以最终收尾不会在遗漏它的情况下给出“nothing handed over”或 `final_T` 为 false。
   - 现在被丢弃的错误（`p5_darwin_test.go:765`）改为消费。这部分提为一个小函数（平台无关），最终收尾与模型检查都调用它。
6. **锁**：
   - 停止任务在不持有任何锁时调用 `Close`，之后只在自己的状态锁内写入结果；
   - 持有者只经 `until` 与快照读取，不取控制锁。
7. **截止**：只用既有的 w 与 D + 30 s。不新增期限，不改变预算。
8. **记录**（§16.11 的补充）：门禁行的 `p5` 增加 `read_close`：
   - 关闭从何而起：EOF 之后、读错误之后或 w 的查询之后；
   - 是否完成；
   - 真实结果、报告的结果与是否注入；
   - 释放的知识（已释放 / 释放未确认）；
   - 调用次数。
9. **代码的位置**：第 3 ~ 5 条的三个小函数（门禁对关闭结果的消费、移交谓词、最终收尾对读端的消费）写在平台无关的协议核心中（`p5proto_test.go`，构建约束 `darwin || wp25p5local`，同 §16.15 的补充 1）。`runP5Gate`、`p5FinalCleanup` 与模型检查调用同一组函数。
10. **不变的部分**：
    - 五个事件与复制任务；
    - 写端由门禁同步关闭（§16.8）；
    - 移交给最终收尾的其余规则（第 4 条只增加读端的条件）；
    - §16.8 所有权表中“读端 r”一行：内容不变，现在在 EOF 的路径上也兑现。

**回归**：

- **R-B1-1 真实接线的正常 EOF 关闭**：两个既有的 P-5 门禁（`p5.eintr-once`、`control.p5-w-exit7`）增加必要条件，不增加运行。
  - 条件：
    - 停止任务恰好调用一次 `Close`；
    - 它在门禁对复制任务的 `until(w)` 查询返回、且复制任务已完成之后启动（这个时刻可能早于 w）；
    - 它在 w 之前完成，真实结果与报告的结果都为 nil，释放的知识为已释放；
    - 它的结果在判定之前被消费；
    - 没有因读端而移交。
  - 核对的依据是停止任务自己记录的调用次数与时刻，不是门禁的叙述。
  - 任何一条不成立时，记 `inner:` 的问题，门禁失败。
  - 在 `0213fb7` 的代码上这些条件不成立（EOF 的路径上没有关闭）。这是按源码推出的回归方向，本轮没有执行。
- **R-B1-2 关闭返回错误的拒绝**：新增一个 P-5 组合 `p5combo.close-error`。它是一次真实的 `runP5Gate` 运行，内层与 `control.p5-w-exit7` 相同。
  - **接缝**：外层的接缝只作用于这一次运行的停止任务。
    - 先执行真实的 `Close`。它返回 nil 时，把结果换成 `injected: the read end's close failed (after the real close)`，只换一次，命中记入停止任务的状态。
    - 真实的 `Close` 本身失败时，保留真实的错误，不替换；这时组合不成立。
  - **预期**（全部成立才通过）：
    - 接缝命中 1 次；
    - 门禁 `Pass` 为 false，问题恰好一条：`inner: closing the read end failed: injected: …`（`ToolErrors` 1，`Unexpected` 0）；
    - 内层的判定成立：目标 confirmed、`calls` 2，布局 `calls` 0，内层退出 0；
    - 补救 not-needed，Final ended，保险清理 nothing left；
    - `read_close` 分开显示两个维度：真实结果为 nil（释放的知识为已释放），报告的结果为注入的错误，是否注入为是；
    - 按第 4 条的移交谓词，读端不移交，没有移交项。
  - **两个维度的限定**：
    - “已释放”只来自这一次真实 `Close` 返回的 nil，不推广到真实的 `Close` 错误：真实错误一律为释放未确认，按第 4、5 条移交并计入 `final_T`。
    - 选择分开两个维度，而不是统一的保守移交，是因为统一移交会使这个安全接缝的组合进入最终收尾的 unknown，`final_T` 因而为 true。那样的结果不是被测的拒绝，而是使整个 P-5 为 T。
  - **记录**：
    - 记为 `WP25COMBO` 行：`id` 为 `p5combo.close-error`，`gate` 为 `control.p5-w-exit7`，`fault` 为 `close-error`；
    - 计入组合（macOS 由 16 增为 17），不计为门禁的通过。
  - **位置**：在 `TestP5` 中、两个门禁之后，按角色 B′ 准入（D − s ≥ 60 s），截止的推导不变。因截止而未执行时记为未执行，P-5 不算完整（失败关闭）。
- **R-B1-3 模型**：只作补充，不代替上面两项。
  - `TestP5ProtoCopyStop` 改为经 `p5Inner.closeRead()` 关闭，不再手工 `Close`；
  - 新增 `TestP5ProtoCloseOnce`：调用两次 `closeRead()` 得到同一个任务，关闭函数只被调用一次；
  - 按 §16.17.5 做命中握手、释放与完成核对。
- **R-B1-4 模型：初次 pending、随后带错完成、移交之前已完成**（第二稿，B-P112-1；只用假的关闭函数，不增加原生的关闭故障实验）：
  - **构造**：
    - 假的关闭函数在进入时经通道发出信号（命中握手），然后等待释放闸门；闸门在 `t.Cleanup` 中也会关闭，失败的出口同样释放；
    - 释放之后返回错误 E，作为真实结果；
    - 其余（Wait、复制、请求）都用已完成的假对象。
  - **步骤与预期**：
    1. 调用 `closeRead()`，等到进入的信号；以一个很近的截止 `until` 查询：未完成。门禁的消费函数给出 `did not return` 的问题。
    2. 打开释放闸门，以截止等停止任务完成（完成核对）。
    3. 新的快照：已完成，真实结果为 E。移交谓词为真（释放未确认）。
    4. 最终收尾对读端的消费函数：该项的 `unknown` 含 E，计入 `final_T`。
    5. 关闭函数只被调用一次；第二次 `closeRead()` 返回同一个任务，不重新关闭。
  - **对照**：同样的步骤，假的关闭函数在释放之后返回 nil。这时：
    - 第 3 步的移交谓词对读端为假（已释放）；
    - 第 1 步的 `did not return` 问题仍在消费结果中；
    - 第 4 步记为晚到的成功，不计入 `final_T`。

#### 16.17.3 B2：新清理链的错误

**问题**（Review 111 §5，源码位置为 `0213fb7`）：

- `p5EndProven`（`p5_darwin_test.go:821` ~ `857`）的问题：
  - 初始读取为 unknown 的对象被跳过（`:824` ~ `827`），这次读取失败没有记录；
  - 每一轮重新建立 `rerr`（`:838`）：一轮读取失败之后，下一轮的成功会把它覆盖；
  - 动作之前 `procInfo`、`tagged` 的失败被布尔条件吞掉；
  - `kill` 的错误被忽略（`:841`）；
  - 普查的错误只在没有读取错误时返回（`:852`）。
- `p5Remediate`（`p5gate_darwin_test.go:547` ~ `568`）有同样的问题：动作之前的检查失败被吞掉，`outerSignal` 的错误被忽略（`:550`）。
- 结果：保险清理的说明、门禁的问题与最终收尾的 `final_T`，都可能漏掉实际发生过的工具失败。

**范围**：

- 改的只有本轮新增的 `p5EndProven`、`p5Remediate`，以及它们的直接调用者：`runP5Gate` 中补救与保险清理的消费、`p5Insure`、`p5FinalCleanup`。
- 不改旧机制：
  - `innerProc.insure`（`witness_unix_test.go:180` 起）、`outerSignal`、`recordAlive`、`censusTag`、`procInfo`、`tagged`；
  - P-3 的清理与 `runGate`。

**每一次读取的分类**：对已记录的身份 v =（pid，启动时刻），以 `procInfo` 读取一次：

| 读数 | 条件 | 含义 | 动作的资格 | 错误 |
|---|---|---|---|---|
| 可靠 absent | `procInfo` 为 absent | v 已结束 | 无 | — |
| 身份变化 | present，启动时刻不同 | v 已结束，PID 已被另一个进程使用 | 永不对它动作 | — |
| 僵尸 | present，同一身份，僵尸 | v 不再运行（与 `recordAlive` 的规则相同） | 无 | — |
| 仍存活 | present，同一身份，不是僵尸 | v 仍在运行 | 只有同一步的标记检查成立之后 | — |
| unknown | `procInfo` 为 unknown | 不知道 | 无 | 持续保存 |

**动作之前的标记检查**：

- 新的辅助函数直接读取 `environOf`（旧的 `tagged` 不变）：
  - 读到本轮的标记：可以动作；
  - 读取成功但没有标记：不动作，记错误（已证明属于本轮的进程不再带有标记，属于矛盾）；
  - 读取失败：立即重读身份。可靠结束（absent、身份变化或僵尸）时，记为“读取期间结束”，不动作，不记错误；否则记错误，不动作。
- 读取失败后的重读，与已审的 `censusTag` 规则相同（`census_darwin_test.go:41` ~ `51`）。它在同一步内给这次失败分类，不是“后来的结束清除先前的失败”。

**终止请求**（`kill(pid, SIGKILL)`）：

- 返回 0：只是请求，不是结束的证明。
- `ESRCH`：允许的竞争（检查之后、发信号之前已结束），记为说明，不记错误。结束仍只由之后的读取确认。
- 其他 errno：工具失败，持续保存。

**责任集合与轮询**：

- **进入集合**：
  - 已知的对象都进入，除非初始读取已可靠结束；
  - 初始读取为 unknown 的，以 unknown 进入，并保存这次失败；
  - 普查证明属于本轮的对象进入；普查的错误同样保存。
- **unknown 的限制**：unknown 永不增加对象，也不授予动作。动作要求同一步内身份仍存活、标记成立。
- **每一轮**，对集合中的每个对象读取一次：
  - 仍存活：做标记检查，然后发出终止请求；
  - 已结束：移出集合，保留它的分类；
  - unknown：留在集合中，保存错误。
- **结束条件**：集合中既没有存活也没有 unknown 的对象，或者到截止。到截止时，存活与 unknown 的对象连同最后的分类一起返回，供移交。
- **错误**：整个调用共用一个累积器（`p5Errs`：按文字去重、计数），不按轮重置。只要发生过错误就返回，即使所有对象最终都已结束。
- **结果分开保存**：
  - 结束的知识：每个对象的分类，到截止仍存活、仍 unknown 的；
  - 实验的有效性：累积的错误。
- **截止与间隔**：都不变。补救与保险清理为 min(当前 + 10 s, d_g)，最终收尾为 D + 30 s，轮询间隔 20 ms。
- **`p5Remediate`**：同一套规则，没有普查。每一轮对存活的对象做动作之前的检查与终止请求；返回仍存活的、unknown 的与累积的错误。

**错误怎样被消费**：

- **保险判定**（`p5Insure`）：
  - 内层退出未确认、仍有存活或 unknown 的对象、或者累积的错误非空（即使对象都已结束）时，说明都以 `FAILED:` 开头，并附上错误；
  - 只有没有任何错误时，才是 `ended n` 或 `nothing left`；
  - 存活与 unknown 的对象都移交。
- **门禁不可豁免的问题**（`runP5Gate`）：
  - 补救的错误：补救 failed，记下 `RemedyErr`，问题 `still-alive: the remediation could not confirm its end: …`，Final unknown（现有的分支）；
  - 补救之后仍为 unknown 的对象：问题 `still-alive: …`（列出身份），Final unknown；
  - 保险清理 `FAILED`：问题 `insurance: not confirmed by d_g: …`，并重新判定（现有的分支）；
  - Final 为 ended 而保险清理不是 nothing left：矛盾的问题（现有的分支）。

  `still-alive`、`insurance` 都属于 `toolClasses`，预期的拒绝不豁免它们。
- **最终汇总**（`p5FinalCleanup`）：
  - 错误记入该项的 `unknown`（`the processes of the round: …`）；存活与 unknown 的对象记入 `processes` 与 `unknown`；
  - `unknown` 非零时，`final_T` 为 true，`t.Errorf` 使 P-5 为 T；
  - 结果见 `WP25P5FINAL` 与 `WP25P5` 的 `final_T`。
- **可测的写法**：上面几处消费提为小函数，调用者直接使用它们，供回归核对：
  - 保险清理的说明；
  - 保险清理之后的判定；
  - 补救的结局；
  - 最终收尾中进程的部分。

  除 B2 的规则之外，行为不变。

**代码的位置**：

- 判定逻辑写成平台无关的代码（构建约束 `darwin || wp25p5local`，同 §16.15 的补充 1）：每次读取的分类、错误的累积、动作的资格、轮询与说明的生成。
- 它经一组操作取得读数：读取身份、读取标记、普查、发信号。darwin 绑定真实的函数。
- 回归的接缝包在真实的读取外面：真实的读取照样进行，只把它的答案换一次。

**回归**：

- **R-B2-1 模型**（本机与 darwin，假对象）：
  1. 轮询的读取 unknown 一次，下一轮为 absent：错误保留，没有遗留，结束 1 个。
  2. 初始读取 unknown，下一轮仍存活：对象留在集合中；只在恢复的那一次读取之后动作（终止请求 1 次）；之后结束；错误保留。
  3. 身份仍存活而标记读取失败一次，重读仍存活：那一步不动作，错误保留；下一步动作。
  4. 标记读取失败，重读为 absent：不记错误（`censusTag` 的规则）。
  5. `kill` 返回 `ESRCH`：只记说明，不记错误；返回 `EPERM`：记错误。
  6. 到截止仍为 unknown：作为 unknown 返回、移交，错误保留。

  每一项都核对命中：假对象记录每一次调用，注入的失败恰好命中一次，而且在预期的那一步。
- **R-B2-2 darwin 的真实进程**（直接检查，不用 D；第二稿按 B-P112-3 写定执行协议，第三稿按 B-P112-3-R 收口请求任务、每侧收尾的时机与截止之后的持有）：
  - **数量与结构**：
    - 一个顶层测试 `TestP5CleanupReal`（darwin）。两侧各是它的一个**非并行子测试**，依次执行：`t.Run("insurance", …)` 测 `p5EndProven` 链，`t.Run("remediation", …)` 测 `p5Remediate` 链。子测试中不调用 `t.Parallel`，也不再嵌套子测试。
    - 每侧恰好一个受控的子进程，执行一次，不复用，不重试。共 2 个子进程、2 次执行。
    - 门禁的内层运行只选 `TestP5` 的子测试，不会运行它。
  - **适用的前提**（第三稿，Review 113 §6.2）：每侧调用 `start` 之前核对下列三条，任何一条不成立，本侧记为未执行（失败）：
    - 不是内层运行（`innerEnv` 为空）；
    - `witnessOn()` 为 false，所以 `start` 在 `Cmd.Start` 之后立即返回，不进入见证的身份读取与同步 `Wait`（`witness_test.go:170` ~ `186`）；
    - `os.Executable` 可用。

    不绕过统一的 `start`，不新增启动的例外，也不改动旧的启动链。在这些前提下，`start` 失败时没有子进程。
  - **受控的子进程**：
    - 经 `start` 启动直接子进程 `helper("exit-after", …, "60000", "0")`；
    - 环境中带本侧独有的标记（一个只属于本侧的环境项）。
  - **截止**：都是按单调时钟的绝对时刻，与第二稿相同。

    | 名称 | 定义 | 用途 |
    |---|---|---|
    | S_r | `TestP5CleanupReal` 开始的时刻 | 总的起算点 |
    | T_bin | `t.Deadline()`，即根测试二进制 `-timeout` 的截止；没有时取 S_r + 60 s | 套件的截止 |
    | T_r | min(S_r + 60 s, T_bin − 30 s) | 本检查的总截止 |
    | s_k | 第 k 侧子测试开始的时刻 | 本侧的起算点 |
    | 准入 | T_r − s_k ≥ 25 s；不成立时本侧记为未执行（失败），不启动子进程 | — |
    | c_k | s_k + 10 s | 交给被测清理链的截止；也是启动被测链的最晚时刻 |
    | q_k | s_k + 12 s | 查询被测链的任务是否返回 |
    | w_k | s_k + 20 s | 正常路径上查询请求任务与唯一 Wait 任务；也是正常路径上发出请求、启动 Wait 的最晚时刻 |
    | d_k | s_k + 25 s（不超过 T_r） | 本侧收尾的最后截止；w_k 到 d_k 的 5 s 是清理余量 |

    - **第二侧的准入**在第一侧的 `t.Run` 返回之后才计算。那时第一侧的收尾已经完成（见下面“每侧收尾的时机”），所以两侧不重叠。
    - **名义窗口与实际耗时**（第四稿，N-114-1）：
      - 每侧 25 s 是名义窗口，不是无条件的实际耗时上界。
      - 同步的操作或调度的延迟可以使本侧的实际结束晚于 d_k（例如时间线戊中第一侧结束于 27 s，第二侧最晚可到 52 s）。这时按实际时刻记录，不截写为 25 s，也不重新取得预算。
      - T_r 与各截止是准入与查询的边界，不是能够中断系统调用的硬上限。
      - 本稿不为这一已披露的延迟模型增加超时或新的机制。
    - **可用量**：
      - 根测试二进制由 `go test` 运行时总有 `-test.timeout`（工作流中为 15m，`go test` 没有指定时默认 10m），所以 `t.Deadline()` 存在，T_r 通常为 S_r + 60 s。
      - 没有 `t.Deadline()` 时（例如直接运行测试二进制而不带 `-test.timeout`，或传入 0），按公式 T_bin = S_r + 60 s，T_r = S_r + 30 s。这时两侧都按各自实际的准入点判断：
        - 第一侧只在 T_r − s_1 ≥ 25 s，即 s_1 ≤ S_r + 5 s 时准入。通常 s_1 接近 S_r，但这不是承诺；子测试的调度已耗掉超过 5 s 时，第一侧也记为未执行（失败关闭）。
        - 第二侧同样只在 s_2 ≤ S_r + 5 s 时准入（等号成立也准入），否则记为未执行（失败关闭）。
        - 不宣称这时一定有完整的 50 s。
    - 不增加作业的 20 min，也不延长根测试二进制的上限。
  - **先登记**：`start` 成功之后、任何其他观察与断言之前，依次：
    1. 从 `cmd.Process.Pid` 读取一次 pid；
    2. 建立本侧的持有者：本侧的标识、Cmd、pid、控制锁与回收标记、请求表（空）、唯一 Wait 任务（尚未启动）、被测链的任务（尚未启动）、收尾的标记（未进入）；
    3. 立即以**本侧子测试的** `t.Cleanup` 登记持有者的收尾函数。不登记到顶层的 `t`。

    之后才读取身份、运行被测链与断言。持有者复用门禁内层的同一组协议结构：§16.8 第四稿的控制锁、请求任务（R0 ~ R5）与唯一 Wait 任务（W1 ~ W6）。它的操作为：
    - 观察：`procInfo`，判断是否是本进程未回收的僵尸；
    - 回收：`Cmd.Wait`；
    - 状态：`ProcessState`；
    - 终止请求：`syscall.Kill(pid, SIGKILL)`。

    R0 在 Wait 任务尚未启动时，按“未开始回收”处理。
  - **顺序与安全协议**（第三稿，B-P112-3-R 第一项；采用已审的请求 / 回收串行规则）：
    1. **被测链**：在自己的任务中运行，先登记后启动。只在当前时刻早于 c_k 时启动；否则不启动，记为未执行（失败）。
       - 被测链按 PID 发信号，不经控制锁。所以**被测链的任务返回之前，不发持有者的请求，也不启动 Wait 任务**。子进程在被回收之前，它的 PID 不会被重用；被测链运行期间，子进程不被回收。
    2. **持有者的终止请求**：只经请求任务（R0 ~ R5）发出。
       - R0：最近的 Wait 快照显示“开始回收”时不需要；先前的请求任务仍 pending 时，只查询它，不另发第二次；否则新建请求任务，先登记到本侧的请求表，再启动。所以每侧同时至多一个 pending 的请求。
       - R1 ~ R3：在控制锁内读取回收标记；未开始回收时，向持有者自己读取的 pid 调用一次 `kill`，记下原始 errno；然后释放控制锁。
       - R4、R5：在请求任务的状态锁内写入完成与原始错误，然后发布完成。完成状态与原始错误持续保存，之后的成功不清除。
       - 父端（本侧的测试 goroutine 与收尾函数）**从不同步调用 `kill`，也从不取控制锁**，只按既定的截止以 `until` 查询请求任务。
    3. **唯一 Wait 任务**：在被测链的任务返回之后（或者被测链没有启动，例如身份读取失败），可以启动。
       - 它与请求任务按控制锁串行（W2 在同一把锁内置“开始回收”）。请求任务停在 `kill` 中、持有控制锁时，W2 等锁，`Cmd.Wait` 不开始。所以不会出现“晚到的 PID 动作”与回收的竞争。
       - 请求任务未返回时，不另发请求；Wait 任务照常按截止查询。
    4. **权利的范围**：持有者的请求只针对自己读取的那个 pid，并且只在回收标记之前（R2 的规则）。这一权利不依赖身份的读取，也不扩展到任何其他 PID。
  - **新任务的准入与统一的到期出口**（第四稿，B-P112-3-R/D）：
    - **截止**：都是既有的绝对截止，不新增：
      - 正常路径用 w_k；
      - 收尾函数用 d_k；
      - 被测链的启动沿用 c_k。
    - **决策点**：父端准备做下面任何一件事之前，都是一个决策点：
      - 新建持有者的请求任务：R0 的“新建”分支。只查询已有的 pending 请求不是新任务，不受此限；
      - 启动尚未启动的唯一 Wait 任务；
      - 启动被测链。
    - **复核的规则**：
      - 在每个决策点上，先重新读取当前时刻（单调时钟），再与该路径的截止比较；
      - 在此之前发生的任何等待（`until`）或同步读取（`procInfo` 等），返回之后都不能代替这次读取：
        - `until` 返回 Done = true，不代表还有剩余的时间；
        - 一次读取开始时尚未到期，不代表返回时仍未到期。
    - **当前时刻达到或超过截止**（`now ≥ 截止`）：走统一的到期出口，不新建请求，不启动 Wait，不启动被测链，不重新取得等待的预算。
      - **正常路径的到期出口**：
        - 只取各任务的快照，记 T（`… deadline w_k passed at the decision point`），返回；
        - 未确认的责任仍由本侧持有者持有，交本侧收尾按 d_k 处理；
        - 收尾在 d_k 之前进入时，在自己的决策点上按 d_k 复核。
      - **收尾函数的到期出口**：只取各任务的快照，按规则记 T，把未确认的责任移交 `p5RealHeld`，返回。
      - **共用一个出口**：“首次进入收尾时已过期”与“执行步骤中越过截止”是同一个到期出口，不是两条规则。
    - **截止之前已经启动的任务**：被测链、请求、Wait 可以在截止之后晚到完成原有的步骤，结果持续保存在任务自己的状态中。这是任务的晚到完成，不是父端在截止之后启动新任务的依据。
    - **守卫的范围**：
      - 这一守卫针对已经知道过期之后的决策；
      - 它不要求“读取时钟”与“登记并启动任务”之间具备硬实时的原子性，也不声称能中断系统调用；
      - 时钟读取之后到任务登记之间的短暂间隔照旧存在，不当作截止的扩展。
  - **正常路径**：
    1. 读取身份（`procInfo`）。
    2. 运行被测链（截止 c_k），以 `until(q_k)` 查询它。
    3. 被测链返回之后，按 `procInfo` 判断：子进程仍存活（不是僵尸）或状态 unknown 时，记为“清理链没有结束它”，本侧失败；需要发出持有者的请求。**决策点**：按上面的规则以 w_k 复核，未到期时按 R0 新建请求，已到期走正常路径的到期出口。
    4. **决策点**：以 w_k 复核，未到期时启动 Wait 任务，已到期走正常路径的到期出口。然后以 `until(w_k)` 查询 Wait 任务，有请求时也查询请求任务。
    5. 断言：
       - Wait 已完成，状态为信号 9；
       - 请求表为空，即持有者的终止请求为 0 次：这样信号 9 才独立证明是清理链终止了子进程；
       - 接缝恰好命中 1 次；
       - 链的结果与调用者的消费（下面）。
  - **每侧收尾的时机**（第三稿，B-P112-3-R 第二项）：
    - 收尾函数登记在本侧子测试的 `t` 上。固定的 Go 1.27.3（`2aae1000` 的 `src/testing/testing.go`，SHA-256 `27c59df1832eae458ab8d2d4b6a6252bbd1ea2a3001734910d1ce2decff16f81`）中，没有子测试的测试，在它的函数返回或因 FailNow 而 `Goexit` 之后，由 `tRunner` 的延迟调用执行其 Cleanup（`testing.go:2187` ~ `2191`），然后才向父测试发出完成信号（`:2102`）；`t.Run` 在收到这个信号之前不返回（`:2268`）。Cleanup 的语义见 `:1438` ~ `1440`。
    - 所以：
      - **正常出口**：本侧函数返回 → 本侧收尾 → `t.Run` 返回；
      - **FailNow**：只结束本侧子测试 → 本侧收尾 → `t.Run` 返回 false。

      两种出口都在第二侧准入之前完成第一侧的收尾。
    - **只执行一次**：Go 对每次登记的 Cleanup 只调用一次。收尾函数另有“已进入”的标记，第二次进入时直接返回（防御）。正常路径不显式调用它，只经 `t.Cleanup` 进入。
    - **任务不触碰 `t`**：被测链、请求与 Wait 任务都不调用 `t` 的任何方法。只有本侧的测试 goroutine 与本侧的收尾函数调用本侧的 `t`。
  - **收尾函数**（第四稿：每个决策点都按 d_k 复核，到期即走收尾函数的到期出口）：
    1. **进入**：读取当前时刻；已达到或超过 d_k 时，走到期出口（这就是“首次进入时已过期”）。
    2. **被测链的任务已启动而未返回**：以 `until(d_k)` 再查一次。
       - 仍未返回：不发请求，不启动 Wait，移交持有登记，记 T，返回。
       - 已返回（Done = true）：不据此推定还有时间，继续第 3 步，在决策点上复核。
    3. **Wait 任务尚未启动**：
       1. 以 `procInfo` 判断子进程是不是可靠的僵尸（同步读取）。
       2. 不是可靠的僵尸、且没有 pending 的请求时，需要新建请求。**决策点**：重新读取当前时刻，以 d_k 复核；未到期时按 R0 新建请求，已到期走到期出口。有 pending 的请求时只查询它，不新建。
       3. **决策点**：重新读取当前时刻，以 d_k 复核；未到期时启动 Wait 任务，已到期走到期出口。
    4. 以同一个截止 `until(d_k)` 依次查询 Wait 任务与请求任务。这一步只查询，不启动新任务。
    5. **全部确认**（Wait 已完成，没有 pending 的请求）：记录回收的状态，返回。
    6. **否则**：把未确认的部分移交持有登记，以 `t.Errorf` 记 T，返回。不同步等待，不重试。

    收尾函数的到期出口：只取各任务的快照，按规则记 T（已经成立的 T 与失败照样保留），把未确认的责任移交持有登记，返回。不新建请求，不启动 Wait，不重新取得预算。
  - **截止之后的持续持有**（第三稿，B-P112-3-R 第三项）：
    - **持有登记**：一个根测试进程内的登记 `p5RealHeld`（受互斥锁保护的列表），与 `p5.handed` 分开。原因是 `TestP5` 的最终收尾与本检查的先后不固定。
    - **登记项的内容**：
      - 本侧的标识、Cmd、pid、移交的时刻与原因；
      - 被测链的任务；请求表与各请求任务；
      - Wait 任务，或“尚未启动”；
      - 各任务持续的结果：完成标志、原始错误、晚到的错误；
      - **移交时的知识**（第四稿，N-114-2）：移交那一刻的一次快照，此后不改写。

      登记项保持对这些对象的引用，直到测试进程结束。Cleanup 返回之后，持有者就是这个登记，不是一条 unknown 文字。
    - **回收的四种状态**（第四稿，N-114-2；由 Wait 任务的状态推出，分开记录）：

      | 状态 | 条件 |
      |---|---|
      | Wait 尚未启动 | Wait 任务没有启动；回收的责任未开始 |
      | Wait 已启动、未开始回收 | W1 中：还没有看到僵尸，或者 W2 在等控制锁；回收标记仍为“未开始” |
      | 已开始回收、未确认 | W2 已置“开始回收”，W4 的 `Cmd.Wait` 尚未完成 |
      | 已可靠回收 | W4 的 `Cmd.Wait` 已返回，W5 已写入状态并发布完成 |

    - **两种视图分开**：
      - **移交时的知识**：移交那一刻的回收状态、请求与被测链的状态，以及移交的原因。
      - **当前任务快照**：读取时（例如 `TestMain` 中）各任务的状态。之前已启动的 Wait 可能在之后完成 W4，所以当前快照可以是“已可靠回收（晚到）”。
    - **登记的含义**：
      - 登记项继续保留，不等于当前仍未回收；
      - 晚到的回收不恢复移交时已成立的 T，也不恢复本侧或总体的通过。
    - **晚到的结果**：只写入各任务自己的状态（状态锁内），任务不调用已经结束的 `t`。也不触发新的动作：到期之后不继续等待，不自动重试，不借用 D，也不启动尚未启动的请求或回收（与上面的统一到期出口是同一条规则）。
    - **最后的观察**：根测试进程的 `TestMain` 在所有测试结束之后（外层），把持有登记各项写入 `WP25P5` 的 `real_held`：移交时的知识与一次不等待的当前快照，都只打印、不裁定。判定已由移交时的 `t.Errorf` 决定，N3 的限制照旧。
    - **展示标签的用途**：只用于记录。PID 动作只按控制锁内的回收标记（R2 的规则），有效性只按移交时已经记录的 T；都不读取展示标签。
    - **不当作回收**：下列情形都不当作已回收；只有 W4 可靠完成才是“已可靠回收”：
      - helper 自己在 60 s 之后退出；
      - 测试进程退出之后由系统收养；
      - `t.TempDir` 被删除。
  - **各出口的所有者与处理**：

    | 出口 | 所有者 | 处理 | 结论 |
    |---|---|---|---|
    | 未准入或前提不成立 | 本侧子测试 | 不启动子进程 | 本侧失败（未执行） |
    | 正常 | 本侧持有者 | 链终止子进程，Wait 回收，信号 9，请求 0 次；收尾确认 | 按上面断言 |
    | 身份读取失败 | 本侧持有者 | 不运行被测链；在决策点按 w_k 复核之后按 R0 请求、启动 Wait，`until(w_k)`；收尾按截止确认 | 本侧失败 |
    | 接缝未命中 | 本侧持有者 | 链照常返回；子进程仍存活时按 R0 请求；Wait 回收 | 本侧失败（命中 0） |
    | 终止失败：链没有结束子进程（没有动作、发信号出错，或到 c_k 仍存活） | 本侧持有者 | 按 R0 请求，Wait 回收 | 本侧失败 |
    | 被测链到 q_k 仍未返回 | 本侧持有者，之后为收尾函数 | 不请求，不启动 Wait；收尾到 d_k 再查一次；仍未返回则移交持有登记 | 本侧失败 |
    | 持有者的请求未返回 | 本侧持有者，之后为持有登记 | 不另发请求；Wait 按控制锁等待；到 d_k 未确认则移交持有登记 | 本侧失败 |
    | Wait 到 w_k、d_k 仍未完成 | 本侧持有者，之后为持有登记 | 不同步等待；到 d_k 移交持有登记 | 本侧失败 |
    | FailNow | 本侧子测试的收尾函数 | 按上面的收尾步骤 | 本侧失败 |
    | 在决策点上已达到或超过截止：首次进入收尾时已过期，或执行步骤中越过截止（包括 `until` 返回 Done = true 而决策时已过期、同步读取返回时已过期） | 正常路径：本侧持有者，之后为收尾函数；收尾：收尾函数，之后为持有登记 | 统一的到期出口：只取快照，不新建请求，不启动 Wait；正常路径交收尾，收尾移交持有登记 | 本侧失败 |

  - **时间线**（S_r = 0；设 T_bin 远在之后，所以 T_r = 60。事件中的“设”是为推演所设的条件，不是预测）：

    **时间线甲：第一侧的链未及时返回，之后返回，再进入本侧收尾**

    | 时刻 | 事件 |
    |---|---|
    | 0.0 | 第一侧子测试开始：s_1 = 0，准入 60 − 0 ≥ 25；c_1 = 10，q_1 = 12，w_1 = 20，d_1 = 25。前提成立；`start` 成功，登记持有者与本侧的 Cleanup |
    | 0.1 | 身份读取成功；被测链的任务登记并启动（截止 10） |
    | 12.0 | q_1：`until(12)` 到期，设链的任务未返回（例如其中一次系统调用没有返回）。记 T；不请求，不启动 Wait；本侧函数返回 |
    | 12.0 | 本侧收尾（首次进入，早于 25）：以 `until(25)` 再查链的任务 |
    | 14.0 | 设链在此返回，结果写入它的状态。决策点上重新读取时刻（14.0，早于 25）：子进程不是可靠的僵尸时按 R0 新建请求；再次复核（仍早于 25）之后启动 Wait。若链在接近 d_1 时才返回，见时间线己 |
    | 14.1 | `until(25)`：Wait 已完成（信号 9），请求已完成。全部确认，不移交；本侧仍为失败（q_1 的 T 不恢复）。收尾返回 |
    | 14.1 | 第一侧的 `t.Run` 返回。第二侧准入：60 − 14.1 = 45.9 ≥ 25；s_2 = 14.1，c_2 = 24.1，q_2 = 26.1，w_2 = 34.1，d_2 = 39.1 |
    | 约 15.0 | 第二侧正常路径与收尾完成，`t.Run` 返回。持有登记为空 |

    **时间线乙：持有者的请求未返回**

    | 时刻 | 事件 |
    |---|---|
    | 0.5 | 第一侧：链在截止之前返回，但子进程仍存活（链没有结束它），记失败。决策点（0.5，早于 w_1 = 20）：按 R0 登记并启动请求任务 R1；再次复核之后启动 Wait 任务 |
    | 0.5 | 设 R1 取得控制锁后，`kill` 不返回，一直持有控制锁。Wait 的 W1 读到子进程仍在运行（不是僵尸），按间隔重读 |
    | 20.0 | w_1：Wait 与 R1 都 pending，记 T；本侧函数返回 |
    | 20.0 | 本侧收尾：链已返回；Wait 已启动；按 R0，R1 仍 pending，不另发请求；`until(25)` 查询 Wait 与 R1 |
    | 25.0 | d_1：仍都 pending。移交持有登记：第一侧、Cmd、pid、R1（pending）、Wait 任务（pending）。移交时的知识：Wait 已启动、未开始回收（R1 持有控制锁，W2 没有置标记）；R1 pending。`t.Errorf`，收尾返回 |
    | 25.0 | 第一侧的 `t.Run` 返回。第二侧准入：60 − 25 = 35 ≥ 25；s_2 = 25，d_2 = 50 |
    | 之后 | 设 R1 晚到返回：结果写入 R1 的状态；W1 若随后看到僵尸，W2 取锁置标记，W4 回收，结果写入 Wait 的状态。这些都只进入持续的状态，不调用 `t`，不恢复通过 |
    | 测试结束之后 | `TestMain` 把登记项写入 `real_held`：移交时的知识仍是“Wait 已启动、未开始回收”，当前快照可以是“已可靠回收（晚到）”。原 T 不恢复。最终的持有者是持有登记 |

    **时间线丙：Wait 未返回**

    | 时刻 | 事件 |
    |---|---|
    | 0.5 | 第一侧：链返回，子进程已成为僵尸；不请求；启动 Wait。W1 看到僵尸，W2 置标记，设 W4 的 `Cmd.Wait` 不返回 |
    | 20.0 | w_1：Wait pending，记 T；本侧函数返回 |
    | 20.0 ~ 25.0 | 本侧收尾：`until(25)`，仍 pending；移交持有登记。移交时的知识：已开始回收、未确认（W2 已置标记，W4 未完成）。`t.Errorf` |
    | 25.0 | 第二侧准入：s_2 = 25，d_2 = 50 |

    **时间线丁：FailNow**

    | 时刻 | 事件 |
    |---|---|
    | 0.5 | 第一侧：链返回；对链结果的一项断言以 `t.Fatalf` 失败（例如接缝命中 0），Wait 尚未启动。本侧子测试 `Goexit` |
    | 0.5 | Go 在 `t.Run` 返回之前执行本侧的收尾：链已返回；Wait 未启动；子进程不是可靠的僵尸时按 R0 请求，然后启动 Wait；`until(25)` |
    | 0.6 | Wait 已完成，记录回收的状态；本侧为失败 |
    | 0.6 | 第一侧的 `t.Run` 返回 false；第二侧准入：s_2 = 0.6 |

    **时间线戊：首次进入收尾时已超过 d_k**

    | 时刻 | 事件 |
    |---|---|
    | 0.0 | 第一侧：`start` 成功，登记持有者 |
    | 0.1 | 设身份读取（本侧测试 goroutine 中的同步读取）直到 27.0 才返回 |
    | 27.0 | 已晚于 c_1 = 10：不启动被测链，记为未执行（失败）；本侧函数返回 |
    | 27.0 | 本侧收尾首次进入，读取时刻 27.0，已晚于 d_1 = 25：走统一的到期出口。不请求，不启动 Wait，只取快照；移交持有登记。移交时的知识：被测链未启动、没有请求、Wait 尚未启动。`t.Errorf` |
    | 27.0 | 第一侧的 `t.Run` 返回。第二侧准入：60 − 27 = 33 ≥ 25；s_2 = 27，d_2 = 52（不超过 T_r = 60） |
    | 之后 | 第一侧的子进程（仍在运行）由持有登记持有；helper 在 60 s 时自行退出，不当作回收；测试进程退出之后的系统收养也不当作回收 |

    **时间线己：`until(d_k)` 返回 Done = true，但决策时已在 d_k 之后**（第四稿，B-P112-3-R/D）

    | 时刻 | 事件 |
    |---|---|
    | 0.0 | 第一侧：s_1 = 0；被测链的任务启动（截止 10） |
    | 12.0 | q_1：链未返回，记 T；本侧函数返回 |
    | 12.0 | 本侧收尾进入：读取时刻 12.0，早于 25，继续。第 2 步：`until(25)` |
    | 24.999 | 设链的任务在此写入结果（含错误）并发布完成 |
    | 25.0 | `until(25)` 返回 Done = true。链的结果持续保存。Done = true 不推定还有时间 |
    | 25.001 | 第 3 步：设 `procInfo` 判断子进程不是可靠的僵尸。决策点：重新读取时刻 25.001 ≥ 25，走统一的到期出口：不新建请求，不启动 Wait，只取快照 |
    | 25.001 | q_1 的 T 保留。移交持有登记：Cmd、pid、被测链的任务（已返回，结果与错误都保存）、没有请求、Wait 尚未启动。移交时的知识：Wait 尚未启动。`t.Errorf` |
    | 25.001 | 第一侧的 `t.Run` 返回。第二侧准入：60 − 25.001 ≥ 25；s_2 = 25.001，d_2 = 50.001 |
    | 之后 | 第一侧的子进程（可能仍在运行）由持有登记持有，不再有新的动作 |

    **时间线庚：同步读取开始于截止之前、返回于截止之后**（第四稿，B-P112-3-R/D）

    | 时刻 | 事件 |
    |---|---|
    | 0.5 | 第一侧：链返回。正常路径第 3 步的 `procInfo` 在 0.5 开始 |
    | 21.0 | 设这次同步读取在此返回（不是可靠的僵尸）。决策点：重新读取时刻 21.0 ≥ w_1 = 20，走正常路径的到期出口：不新建请求，不启动 Wait；记 T，返回。责任仍由本侧持有者持有 |
    | 21.0 | 本侧收尾进入：读取时刻 21.0，早于 25，继续。第 2 步：链已返回。第 3 步的 `procInfo` 在 21.0 开始 |
    | 26.0 | 设这次读取在此返回。决策点：重新读取时刻 26.0 ≥ d_1 = 25，走收尾函数的到期出口：不新建请求，不启动 Wait，只取快照 |
    | 26.0 | 已有的 T 与失败保留。移交持有登记：Cmd、pid、被测链的任务（已返回）、没有请求、Wait 尚未启动。移交时的知识：Wait 尚未启动。`t.Errorf` |
    | 26.0 | 第一侧的 `t.Run` 返回。第二侧准入：60 − 26 = 34 ≥ 25；s_2 = 26，d_2 = 51 |

    七条时间线中，第二侧都在第一侧的 `t.Run` 返回之后才准入。最终的持有者，在全部确认时是本侧持有者（收尾时已解除），否则是持有登记。
  - **保险清理一侧**：以接缝把这个子进程第一次轮询读取的答案换成 unknown（真实的读取照样进行）。预期：
    - 接缝命中 1 次；
    - 子进程被清理链终止：持有者的终止请求为 0 次，Wait 回收的状态为信号 9；
    - 链的结果：没有遗留，结束 1 个，错误中有注入的文字。
  - **调用者的消费**：把这个结果交给上面提出的小函数：
    - 保险清理的说明以 `FAILED` 开头，并带上这段文字；
    - 对一个原本通过的门禁结果做保险清理之后的判定：`Pass` 为 false，恰好一条 `insurance:` 的工具问题；
    - 最终收尾中进程的部分给出 `unknown` ≥ 1（会使 `final_T` 为 true）。
  - **补救一侧**：同一协议，把动作之前的身份读取换成 unknown 一次。预期：
    - 接缝命中 1 次；
    - 补救在下一轮动作，子进程被终止：持有者的终止请求为 0 次，信号 9；
    - 错误保留；
    - 补救的结局：failed、`still-alive:` 的问题、Final unknown。
  - **归类**：这些是回归，另列，不计入四类证据、布局、39 次或门禁。
- **R-B2-3 模型：统一的到期出口**（第四稿，B-P112-3-R/D；待实施，本轮不写代码、不运行）：
  - **代码的位置**：持有者的决策步骤（正常路径的第 3、4 步与收尾函数的第 1 ~ 6 步）写在平台无关的协议核心中（构建约束 `darwin || wp25p5local`）。它经一组操作取得当前时刻与子进程的读数；darwin 绑定 `time.Now` 与 `procInfo`，模型检查注入假的时钟与假的读取。真实的 R-B2-2 与模型调用同一组步骤。
  - **对照一（时间线己）**：
    - **构造**：假的被测链在接近 d_k 时发布完成；`until(d_k)` 返回 Done = true；注入的时钟在决策点上给出 d_k + 1 ms。
    - **预期**：
      - 请求表为空，Wait 任务没有启动；
      - 被测链的结果（含错误）保留，q_k 的 T 与到期的 T 都在；
      - 持有登记恰有本侧一项，移交时的知识为“Wait 尚未启动”，Cmd 与 pid 由它持有。
  - **对照二（时间线庚）**：
    - **构造**：假的 `procInfo` 进入时经通道发出信号，然后等待释放闸门（闸门在 `t.Cleanup` 中也会关闭）。检查等到进入的信号之后，把注入的时钟推进到截止之后，再打开闸门。正常路径（w_k）与收尾函数（d_k）各做一次。
    - **预期**：
      - 两次都不新建请求，不启动 Wait；
      - 已有的结果与失败保留；
      - 正常路径的那一次，责任仍在本侧持有者，交给收尾；收尾的那一次，未确认的责任进入持有登记。
  - **对照三（未到期的决策照常进行）**：同样的步骤，注入的时钟在决策点上早于截止。预期照常新建请求（需要时）并启动 Wait，用来确认守卫没有误挡正常的决策。
  - 按 §16.17.5 做命中握手、释放与完成核对。预期由上面的规则预先确定，不调用被测的步骤生成预期。
  - 这些只是模型证据，不当作原生调度延迟或系统调用挂起的动态复现。

#### 16.17.4 符号检查的第二版（§16.4 的方法修订）

**第一版的结果说明了什么**（§16.16）：

- 原始输出中，有跳板与未定义的 `_waitid`（`go tool nm` 的 `U _waitid`，`nm -m` 的 `(undefined) external _waitid`）。
- 第一版记为 T 有两处原因：
  1. `pipefail` 下的提前退出；
  2. `nm -m` 没有给出库的归属。
- 它既不证明 `_waitid` 来自错误的库，也不证明来自 libSystem（Review 111 §7.2）。

**固定的依据**（公开源码，不声称与运行器上的工具或 dyld 是同一构建）：

| 来源 | 位置 | 内容 |
|---|---|---|
| Go 1.27.3 `src/cmd/link/internal/ld/macho.go`（`2aae1000`，SHA-256 `665c2270cec3df9bc9b77931d7e87e50ad5726da147e8df651cead6230c1ef17`，与 §16.2 相同） | `:326` ~ `333` | 内部链接的头标志：`MH_NOUNDEFS`（没有未定义符号时）、`MH_PIE` 与 `MH_DYLDLINK`（PIE 且内部链接时），**没有** `MH_TWOLEVEL` |
| 同上 | `:806` ~ `818` | `LC_DYLD_INFO_ONLY`：rebase 与 bind；weak、lazy、export 都为 0 |
| 同上 | `:833` ~ `840` | 每个动态库一条 `LC_LOAD_DYLIB`，按 `dylib` 列表的顺序 |
| 同上 | `:1093` ~ `1097` | 动态导入的 nlist：`N_EXT`，`n_desc` 为 0 |
| 同上 | `:1428` ~ `1486` | bind 记录：库 ordinal 为该库在列表中的位置加 1；找不到库时为 `BIND_SPECIAL_DYLIB_FLAT_LOOKUP` |
| cctools `e0d56624`（Review 111 §7.2） | `misc/nm.c:1804` ~ `1821` | `nm -m` 只在 `MH_TWOLEVEL` 且 `n_desc` 的库 ordinal 非零等条件下打印 `(from …)` |
| dyld-1378（`fd8d0c4d52320ebf64db34f3cb280310d905c5ae`）`common/MachOAnalyzer.cpp`（SHA-256 `1ca57bff3e391f2a74c20cf0f6bb84eeb1bfdcb195f51483e176b7c06968d9dd`） | `:1269` ~ `1273` | `MH_TWOLEVEL` 只用于由 nlist 的 `n_desc` 取得 ordinal 的路径：没有它时为扁平查找 |
| 同上 | `:1415` ~ `1480` | opcode 形式的 bind：ordinal 直接取自 `BIND_OPCODE_SET_DYLIB_ORDINAL_*` 或特殊值 |
| dyld-1378 `dyld/Loader.cpp`（SHA-256 `82de7dbe69cfd805927b7b416b5af4bffc7e1e571ef9a31cb6056f747400a9a2`） | `:2853` ~ `2858` | ordinal 为正时，目标是第 ordinal − 1 个依赖库 |
| 同上 | `:2866` 起 | 只有 `BIND_SPECIAL_DYLIB_FLAT_LOOKUP` 才按加载顺序扁平查找 |
| 同上 | `:3109` ~ `3113`、`:3445` ~ `3446` | 在目标库中查找导出（staticLink），并沿 re-export 的依赖继续 |

由此：

- 对 Go 内部链接的产物，`nm -m` 不能显示归属：没有 `MH_TWOLEVEL`，`n_desc` 为 0。所以缺少 `(from libSystem)`，不说明归属错误。
- 产物中真正的每符号绑定表示，是 `LC_DYLD_INFO_ONLY` 的 bind 记录及其库 ordinal。按上面的 dyld 源码，ordinal 为正时直接指向对应的 `LC_LOAD_DYLIB`，与 `MH_TWOLEVEL` 无关。
- 这些是固定公开源码的推演。运行器上 dyld 与工具的实际构建可能不同，列为限制，不升级为运行时的证明。

**脚本的规则**（所有静态检查，替换第一版的管道写法）：

- **执行**：每个工具执行一次，输出（stdout 与 stderr）完整写入 `$RUNNER_TEMP/p5static/` 下自己的文件，并独立取得退出状态。
  - 写法为 `if cmd >file 2>&1; then s=0; else s=$?; fi`；
  - `-e` 既不使脚本中止，也不吞掉状态。
- **匹配**：只对文件进行，例如 `grep -c -E -e PATTERN -- FILE`，同样以 `if … then … else … fi` 取得状态：
  - 0 为匹配（并取得行数）；
  - 1 为未匹配；
  - ≥ 2 为匹配器失败。

  不把输出经管道交给会提前退出的匹配器。不用 `|| true`，不忽略 SIGPIPE，不去掉失败的传播。
- **打印**：打印摘要时可以容忍“没有匹配”，但判定只用真实的退出状态与匹配结果。
- **工具的身份与执行**（第二稿，N-P112-1）：
  - 绝对路径：`go` 取 `command -v go`，otool 与 nm 取 `xcrun --find otool`、`xcrun --find nm`。每个路径与它的 `shasum -a 256` 都记入 `P5.static.artifact`。
  - 之后所有调用都经这些已记录的绝对路径执行（例如 `"$OTOOL" -h "$bin"`），不再经 `PATH` 查找同名的工具。
  - `p5bind` 取自本步骤刚构建的 `$RUNNER_TEMP/p5bind`。
  - 路径为空、不是可执行文件或求哈希失败，为 T(tool)。
- **持续的失败标记**：
  - 任何一项为 T，都把步骤的 `failed` 置为 1，之后不再清除：工具非零、匹配器失败、解析失败、未知的表示、对照矛盾、归属不成立。
  - `WP25P5` 的打印汇总不代替这个标记（N3）。
- **类别**：每项结论都写明类别，并与类别一起打印为 `WP25P5STATIC <ID> v2 <结论>`：
  - `ok`；
  - `T(tool)`：工具失败，附命令与退出状态；
  - `T(unmatched)`；
  - `T(matcher)`：附 grep 的状态；
  - `T(parser)`：解析器失败；
  - `T(contradiction)`；
  - `T(attribution)`。

**第二版的检查**（两个 macOS 作业，在 `prototype/wp25` 下，带 `CGO_ENABLED=0`）：

| ID | 工具与输入 | 判定 | 不成立时 |
|---|---|---|---|
| `P5.static.toolchain`、`vet`、`build` | 与第一版相同 | 与第一版相同 | 与第一版相同（toolchain 不成立时不继续） |
| `P5.static.method`（新） | 本步骤自己的执行与匹配函数，作业的 `bash -e -o pipefail` | 下面的对照 M1 ~ M5 都得到预期的类别 | T（方法本身不可信）；其余检查照样执行并记录 |
| `P5.static.artifact`（新） | 产物 `$RUNNER_TEMP/wp25.test` 的大小（`stat -f %z`）、`shasum -a 256`、`"$GO" tool buildid`；`go`、otool、nm 的绝对路径及这三个文件的 `shasum -a 256`；`xcodebuild -version` | 每条命令退出 0，输出非空；大小、SHA-256、build ID 与三个工具的路径各为单一的值 | T(tool) 或 T(parser) |
| `P5.static.settings`（第二版） | `go version -m "$bin"` | 退出 0；文件中恰有一行匹配 `^[[:space:]]*build[[:space:]]+CGO_ENABLED=0$` | T |
| `P5.static.symbols`（第二版，范围收窄为跳板与未定义导入的存在） | `go tool nm "$bin"` | 退出 0；至少一行的名字含 `e2waitid.libc_waitid_trampoline`；恰有一行匹配 `^[[:space:]]+U _waitid$` | T |
| `P5.static.binding`（新） | 解码器 `p5bind`（见下）；`otool -h`、`otool -hv`、`otool -l`、`otool -L`、`nm -m`，都对 `"$bin"` | 下面的规则 A1 ~ A9 全部成立 | T，附类别 |
| `P5.static.entry`、`load` | 不变 | 不变 | 不变 |

**解码器 `p5bind`**（新的 test-only 目录 `prototype/wp25/p5bind`）：

- **组成**：
  - `package main`，只用 Go 1.27.3 的标准库（`debug/macho`、`encoding/binary`、`crypto/sha256`），平台无关，不新增依赖；
  - 模型测试 `bind_test.go` 带构建约束 `darwin || wp25p5local`。所以 Ubuntu、Windows 作业只编译这个包，不运行它的测试；这是对 §16.10 “只随工具链的固定而变”的唯一补充。
- **执行**：静态步骤先以 `go build -o "$RUNNER_TEMP/p5bind" ./p5bind` 构建（失败为 T(tool)），再运行。
  - 参数：产物路径、预期的架构、shell 记录的大小与 SHA-256，以及 otool 与 `nm -m` 的输出文件；
  - 输出：一行 `WP25P5BIND` JSON，以及人可读的行；
  - 退出状态：0 为 ok，1 为 T（附类别），其他为用法错误（T(tool)）。
- **解码器的身份**：固定代码点上的源码，加上 `go1.27.3`；输出中带格式版本 `p5bind/1`。

**规则 A1 ~ A9**（全部在解码器中，依据 `<mach-o/loader.h>` 的定义与上面的固定源码）：

- **A1 产物的身份**：解码器对自己读到的字节计算大小与 SHA-256，须与 `P5.static.artifact` 的记录相同；不同为 T(contradiction)。
- **A2 格式**：须为单一架构（不是 fat）、64 位、`MH_EXECUTE`，CPU 与预期的架构相符（arm64 / x86_64）；否则为 T(parser)。头标志原样记录（十六进制与名字），其中 `MH_TWOLEVEL` 只记录，不作判定（依据见上面 dyld 的源码）。
- **A3 加载命令、段与受支持的流**（第二稿补全段、节与流的范围）：
  - **段**：
    - `LC_SEGMENT_64` 按出现的顺序编号，从 0 开始，即 bind 中的段号；
    - 逐个记录 segname、vmaddr、vmsize、fileoff、filesize、maxprot、initprot 与各节（sectname、segname、addr、size、offset、flags）；
    - vmaddr + vmsize 不得溢出；每个节须完全落在所属的段内（运算不溢出），节的 segname 与段相同。不成立为 T(parser)。
  - **`__LINKEDIT`**：恰有一个，记它的段号 L。
  - **库加载命令**：按出现的顺序编号，从 1 开始，即库 ordinal；它们都须是 `LC_LOAD_DYLIB`。出现 `LC_LOAD_WEAK_DYLIB`、`LC_REEXPORT_DYLIB`、`LC_LAZY_LOAD_DYLIB`、`LC_LOAD_UPWARD_DYLIB` 或不认识的库加载命令，为 T(parser)（未知的表示）。
  - **其余的命令**：`LC_DYLD_INFO_ONLY` 恰有一条；`LC_DYLD_INFO` 与 `LC_DYLD_CHAINED_FIXUPS` 都不得出现（出现即未知的表示，T）；`LC_SYMTAB` 与 `LC_DYSYMTAB` 各恰有一条。缺少或重复为 T(parser)。
  - **受支持的流**：只支持普通的 bind 流。
    - `weak_bind_size` 与 `lazy_bind_size` 须为 0：固定的 Go 1.27.3 把它们写为 0（macho.go:`812` ~ `815`）。不为 0 即未知的表示，T。
    - 所以 weak 流（隐含的弱查找 ordinal）与 lazy 流（指针类型的初值、以 `DONE` 为条目的边界）的语法（dyld-1378 MachOAnalyzer.cpp:`1535` 起）都不需要解码，也不声称支持。
  - **bind 流的范围**：以不溢出的运算核对，否则为 T(parser)：
    - bind_off + bind_size 不溢出，bind_size > 0；
    - 范围落在文件之内，也落在 `__LINKEDIT` 的 [fileoff, fileoff + filesize) 之内。
- **A4 普通 bind 流的受支持语法**（第二稿，B-P112-2；依据固定 Go 的生成 macho.go:`1458` ~ `1489` 与 dyld-1378 MachOAnalyzer.cpp:`1440` ~ `1528`）：
  - **初始状态**：类型未设（0）；段未设，偏移 0；ordinal 未设；符号未设；标志 0；加数 0。
  - **字节的拆分**：opcode = 字节 & 0xF0，立即数 = 字节 & 0x0F。
  - **受支持的 opcode 与状态的变化**：

    | opcode | 动作 |
    |---|---|
    | `0x00` DONE | 流结束。范围内之后的字节须都为 0x00（Go 按 16 字节对齐的填充，macho.go:`1487` ~ `1489`），有非零字节为未知的表示（T）。到范围末尾仍没有 DONE，为 T(parser) |
    | `0x10` SET_DYLIB_ORDINAL_IMM | ordinal = 立即数 |
    | `0x20` SET_DYLIB_ORDINAL_ULEB | ordinal = ULEB；大于 2^31 − 1 为 T(parser) |
    | `0x30` SET_DYLIB_SPECIAL_IMM | 立即数为 0 时 ordinal = 0；否则把 0xF0 \| 立即数 当作有符号的 8 位数（−1 ~ −15） |
    | `0x40` SET_SYMBOL_TRAILING_FLAGS_IMM | 标志 = 立即数；符号 = 之后到 NUL 为止的字节，NUL 须在范围之内 |
    | `0x50` SET_TYPE_IMM | 只支持立即数 1（`BIND_TYPE_POINTER`）；其他为未知的表示（T） |
    | `0x60` SET_ADDEND_SLEB | 加数 = SLEB |
    | `0x70` SET_SEGMENT_AND_OFFSET_ULEB | 段号 = 立即数；偏移 = ULEB |
    | `0x80` ADD_ADDR_ULEB | 偏移 += ULEB |
    | `0x90` DO_BIND | 在当前偏移产生一条记录；偏移 += 8 |
    | `0xA0` DO_BIND_ADD_ADDR_ULEB | 产生一条记录；偏移 += ULEB + 8 |
    | `0xB0` DO_BIND_ADD_ADDR_IMM_SCALED | 产生一条记录；偏移 += 立即数 × 8 + 8 |
    | `0xC0` DO_BIND_ULEB_TIMES_SKIPPING_ULEB | 见表后 |
    | `0xD0` THREADED、`0xE0`、`0xF0` | 未知的表示（T） |

    `DO_BIND_ULEB_TIMES_SKIPPING_ULEB` 的展开：count = ULEB，skip = ULEB。
    - count = 0：不产生记录，偏移不变。
    - count ≥ 1：在 偏移 + i × (skip + 8)（i = 0 … count − 1）处各产生一条记录，然后 偏移 += count × (skip + 8)。
    - 产生任何一条之前，先以不溢出的运算求出最后一条的偏移，并按下面的有效性规则核对它。溢出或越界为 T(parser)，不逐条展开。
  - **记录的有效性**：对每一条展开出的记录，不只目标记录（依据 dyld-1378 MachOAnalyzer.cpp:`1306` ~ `1347` 与 Review 112 引用的 BindOpcodes.cpp:`84` ~ `150`）：
    1. 段已设，段号 < L，且小于段的条数；
    2. 指针按完整的 8 字节核对：先要求段的 vmsize ≥ 8，再要求 偏移 ≤ vmsize − 8；
    3. 符号与 ordinal 已设；ordinal ≥ −3，且不大于库加载命令的条数；
    4. 类型为 `BIND_TYPE_POINTER`；段的 initprot 含写、不含执行；
    5. 地址 = 段的 vmaddr + 偏移（不溢出）；[地址, 地址 + 8) 完全落在该段的某一个节之内：地址 + 8 不溢出，且不超过 节的 addr + size。

    任何一条不成立，为 T(parser)：流本身不能按受支持的语法解释。
  - **算术**：都以 64 位无符号、不溢出的运算进行，截断或溢出为 T(parser)：
    - ULEB 与 SLEB 至多 10 个字节，值须能放进 64 位（ULEB 的第 10 个字节只能是 0 或 1；SLEB 另按符号的规则），且在范围之内结束；
    - 偏移的递增、count × (skip + 8) 的乘加、地址与文件范围的换算。
  - **记录的内容**：
    - 所在的段名与节名、偏移、地址；
    - ordinal、符号、标志、类型、加数；
    - 产生它的 opcode 与重复的序号 i。

    另记录展开之后的记录总数。
- **A5 唯一匹配**：按展开之后实际的绑定位置计数，不按 opcode 计数。
  - 展开出的记录中，符号恰为 `_waitid` 的恰有一条；
  - 没有其他以 `_waitid` 开头的符号（例如 `_waitid$NOCANCEL`）；
  - 所以：
    - 目标状态之后接 count = 2 的重复形式，就是两个绑定位置，不唯一；
    - count = 0 不产生记录，不能凭这条 opcode 算作一条绑定；
  - 没有为“缺失”，多于一条为“不唯一”，都是 T(attribution)。
- **A6 归属**：这一条记录须满足：
  - 类型为 `BIND_TYPE_POINTER`，没有 `WEAK_IMPORT` 与 `NON_WEAK_DEFINITION` 标志，加数为 0；
  - ordinal 为正，且不大于库加载命令的条数；
  - 它所指的加载命令是 `LC_LOAD_DYLIB`，路径恰为 `/usr/lib/libSystem.B.dylib`；并且只有一条加载命令是这个路径。

  不成立的情形：
  - ordinal 为特殊值：0（本镜像）、−1（主程序）、−2（扁平 / 动态查找）、−3（弱定义合并）或其他。这时归属没有建立，为 T(attribution)。
  - ordinal 越界：T(parser)。
  - 路径不同或重复：T(attribution)。
- **A7 绑定的位置**（第二稿）：
  - 目标记录已按 A4 的有效性核对：完整 8 字节的范围、段的权限与节的对应。这里记录它的段名与节名。
  - 固定 Go 的预测是 `__DATA_CONST,__got`（macho.go:`504`、`630`、`721` ~ `732`）。实际的节与预测不同时只记录，不单独作判定；归属由 A6 判定。
  - 跳板的 `JMP` 指令与这个位置的对应不做反汇编核对，列为限制。
- **A8 符号表**：恰有一个未定义的外部 nlist 名为 `_waitid`，记录它的 `n_desc`（只记录：没有 `MH_TWOLEVEL` 时不携带归属）。没有或多于一个为 T(contradiction)。
- **A9 与 Apple 工具的对照**（第二稿，N-P112-1）：
  - **语法的依据**：cctools `e0d56624` 的打印代码：
    - `otool/main.c:1256` ~ `1265`：文件名行；
    - `otool/ofile_print.c:1749` ~ `2353`：头；
    - `:2421`：`Load command`；
    - `print_dylib_command`、`print_dyld_info_info_command`（`:4633` 起）：`-l` 的块；
    - `:3014` ~ `3038`：`-L` 的行。

    运行器上 `xcrun` 找到的 otool 不声称与这份源码同构建。它的输出不符合下面的语法时为 T(parser)，不放宽语法去迁就。
  - **共同的规则**：
    - 每个工具的输出文件都已独立取得退出状态，非零为 T(tool)；
    - 解析之前，完整的原文已保存在文件中，并原样出现在日志里；
    - **行的切分**（第三稿）：
      - 输出须以 `\n` 结尾。最后这个 `\n` 是最后一行的结束符，不是额外的空行；按 `\n` 切分之后，丢弃它之后的那个空串。
      - 非空的输出不以 `\n` 结尾、中间出现空行，或者输出为 0 字节，都为 T(parser)。
    - 除了下面明说可以变化的地方，空白须完全相同；
    - 缺失、重复或无法识别的行，为 T(parser)；与解码器不同，为 T(contradiction)。
  - **文件名行**：每个 otool 输出的第一行须恰为 `<产物路径>:`。路径与传给它的参数逐字节相同，不带架构或归档成员的后缀。
  - **`otool -h`**：
    - 第 2 行恰为 `Mach header`；
    - 第 3 行恰为 `      magic  cputype cpusubtype  caps    filetype ncmds sizeofcmds      flags`；
    - 第 4 行是唯一的数值行。它的打印形状为 `" 0x%08x %7d %10d  0x%02x  %10u %5u %10u 0x%08x\n"`（`ofile_print.c:2347`），所以（第三稿）：
      - 行首至少有一个空格；字段之间是一个或多个空格；`\n` 之前没有空白；不含制表符；
      - 去掉行首的空格之后，按连续的空格切分，恰好 8 个字段，依次为：
        - magic：`0x` 加 8 位十六进制；
        - cputype：十进制；
        - cpusubtype：十进制；
        - caps：`0x` 加 2 位十六进制；
        - filetype、ncmds、sizeofcmds：十进制；
        - flags：`0x` 加 8 位十六进制；
    - 之后没有其他行。
    - **核对**：magic 为 `0xfeedfacf`（MH_MAGIC_64）；cputype 为 16777228（arm64）或 16777223（x86_64），与作业的架构相符；filetype 为 2；ncmds、sizeofcmds 与 flags 都与解码器相同。
  - **`otool -hv`**：只原样记录；须退出 0，第一行为文件名行。
  - **`otool -l`**：
    - 文件名行之后，块的边界是恰为 `Load command <N>` 的行；N 从 0 起连续，块数须等于 `otool -h` 的 ncmds。
    - 每块的第一行须匹配 `^ *cmd (\S+)$`。其余的行保留原文，只解析下面两种块：
      - 库加载命令的块（`cmd LC_LOAD_DYLIB` 等）：恰有一行匹配 `^ *name (.+) \(offset ([0-9]+)\)$`，第一组为路径；`name ?(bad offset …)` 为 T(parser)。
      - `cmd LC_DYLD_INFO_ONLY` 的块：`rebase_off`、`rebase_size`、`bind_off`、`bind_size`、`weak_bind_off`、`weak_bind_size`、`lazy_bind_off`、`lazy_bind_size`、`export_off`、`export_size` 各恰有一行，匹配 `^ *<名字> ([0-9]+)$`。带 ` (past end of file)` 或其他后缀，为 T(parser)。
    - **核对**：`cmd` 的顺序、库加载命令的路径与顺序、上面十个数值，都与解码器相同。
  - **`otool -L`**：
    - 文件名行之后，每行须匹配 `^\t(.+) \(compatibility version ([0-9]+)\.([0-9]+)\.([0-9]+), current version ([0-9]+)\.([0-9]+)\.([0-9]+)(, (weak|reexport|upward|lazy))?\)$`。第一组为路径；三段版本号保留，不作判定。
    - 带 `, weak`、`, reexport`、`, upward` 或 `, lazy` 的后缀，与 A3（只允许 `LC_LOAD_DYLIB`）矛盾，为 T(contradiction)。
    - 路径的顺序须与解码器的库加载命令相同。
    - 不匹配的行为 T(parser)。
  - **`nm -m`**（Xcode 的 nm）：
    - 只解析以 `_waitid` 为一个完整的空白分隔字段的行；其余的行只保留原文。
    - 这样的行须恰有一行，并整行匹配 `^ *\(undefined\) external _waitid( \(from ([^)]+)\))?$`；有 `(from …)` 时，名字须为 `libSystem`。
    - 没有这一行、多于一行、不匹配或名字不同，为 T(contradiction)。
  - 这些工具只作对照：“存在 libSystem 依赖”（`otool -L`）不替代 A5、A6。

**允许的 re-export 关系**：

- 静态的归属止于 `/usr/lib/libSystem.B.dylib`。
- 按 dyld-1378 的 `Loader.cpp:3109` ~ `3113` 与 `:3445` ~ `3446`，dyld 在 libSystem 的导出中查找，并沿它 re-export 的子库继续（例如 `/usr/lib/system/` 下的库）。所以定义可以位于 libSystem 再导出的子库中。这是允许的关系。
- 子库是哪一个，不在静态检查中观察。
  - 在所测的环境（运行器的 macOS 26 映像）上，libSystem 位于 dyld 的共享缓存中，磁盘上通常没有它单独的文件。
  - 这只是所测现代系统的布局前提，不推广为所有受支持的环境都不可能取得磁盘上的文件（第二稿按 Review 112 §5.4 收窄）。
  - 静态方法本来止于 libSystem 这个导入目标，不为此新增运行时的镜像观察。
- 运行时的镜像或地址观察是另一种办法，须另行明确所测的对象并另获授权（Review 111 §7.3）。本计划不包括。

**方法的对照**（在同一步骤中、真实检查之前执行；用同一组执行与匹配函数）：

| 对照 | 构造 | 预期的类别 |
|---|---|---|
| M1 大输出匹配 | 生成 200 000 行的文件，匹配行在第 1 行与最后一行；另一份只在第 1 行 | 匹配，行数分别为 2 与 1；没有 `Broken pipe` |
| M2 工具非零退出 | 一个 shell 函数，输出含匹配的文字，然后以状态 3 返回 | T(tool)，即使文件匹配 |
| M3 未匹配 | 工具退出 0，输出不含匹配 | T(unmatched) |
| M4 匹配器失败 | 匹配一个不存在的文件 | T(matcher)（grep 状态 ≥ 2） |
| M5 解码器的失败出口 | 以 `prototype/wp25/go.sum` 作为产物运行 `p5bind`。传给它的大小与 SHA-256 由同一组命令对 go.sum 本身求出，所以 A1 成立、检查到达 A2（第二稿，Review 112 §5.5） | 退出 1，T(parser)，失败的规则为 A2。得到 A1 的 contradiction 时，对照不成立 |

任何一项得不到预期的类别，`P5.static.method` 为 T。

**解码器的模型对照**（`bind_test.go`，以代码构造最小的 Mach-O 映像）：

- **基准映像**（第三稿写定，下面各例只说明与它的不同）：
  - **段**：
    - 0 `__PAGEZERO`；
    - 1 `__TEXT`（initprot r-x）；
    - 2 `__DATA_CONST`（initprot rw-，vmaddr `0x100004000`，vmsize `0x40`），只有一个节 `__got`，覆盖 [`0x100004000`, `0x100004040`)；
    - 3 `__LINKEDIT`。
  - **库**：一条 `LC_LOAD_DYLIB /usr/lib/libSystem.B.dylib`（ordinal 1）。
  - **符号表**：一个未定义的外部 `_waitid`。
  - **bind 流**（偏移都相对段 2，记为 X）：
    1. `SET_TYPE_IMM 1`；
    2. `SET_SEGMENT_AND_OFFSET_ULEB` 段 2、偏移 X；
    3. `SET_DYLIB_ORDINAL_IMM 1`；
    4. `SET_SYMBOL_TRAILING_FLAGS_IMM 0 "_waitid"`；
    5. `DO_BIND`；
    6. `DONE`，然后是全零的填充。
  - otool 与 `nm -m` 的文本按 A9 的语法与它一致。
- **正例**：
  - X = `0x00`，得 ok：一条 `_waitid` 记录，地址 `0x100004000`。
  - （第二稿）最后一个完整的指针：X = `0x38`（剩余 vmsize − X = 8 字节），节在段尾结束，得 ok。
  - （第二稿）X = `0x00`，以 `DO_BIND_ULEB_TIMES_SKIPPING_ULEB` count = 1、skip = 0 代替 `DO_BIND`，得 ok：一条记录，地址 `0x100004000`。
  - （第三稿写全符号的状态）scaled 指令：
    - 流为：X = `0x00`；ordinal 1；`SET_SYMBOL_TRAILING_FLAGS_IMM 0 "_other"`；`DO_BIND_ADD_ADDR_IMM_SCALED` 立即数 2；`SET_SYMBOL_TRAILING_FLAGS_IMM 0 "_waitid"`；`DO_BIND`；`DONE`。
    - 按规则手算：第一条记录为 `_other`，偏移 `0x00`；偏移随后 += 2 × 8 + 8 = `0x18`。第二条记录为 `_waitid`，偏移 `0x18`，地址 `0x100004018`。
    - 预期 ok：展开出的记录共 2 条，其中 `_waitid` 恰有一条。
    - 对照（第四稿更正）：把第一条 `SET_SYMBOL` 改为 `"_waitid"`，并去掉第二条 `SET_SYMBOL`。两个合法的位置（偏移 `0x00` 与 `0x18`）都绑定 `_waitid`，预期 T(attribution)（A5，不唯一）。第三稿写的“只去掉第二条”会使两条都是 `_other`，那是目标缺失，不是不唯一。
- **反例**，各得规定的类别与失败的规则：
  - ordinal 为特殊值：−2、0、−1、−3；
  - ordinal 2 越界；
  - 路径为 `/usr/lib/libSystem.dylib`；同一路径两条；目标为 `LC_LOAD_WEAK_DYLIB`；
  - 带 `WEAK_IMPORT`；加数非零；
  - bind 中两条 `_waitid` 记录（两条 `DO_BIND`）；没有记录；`_waitid$NOCANCEL`；
  - `THREADED`；opcode `0xE0`；被截断的 ULEB；
  - 带 `LC_DYLD_CHAINED_FIXUPS`；带 `LC_DYLD_INFO`（不是 `_ONLY`）；fat 文件；
  - 符号表没有未定义的 `_waitid`；
  - `otool -l` 的路径不同；`otool -h` 的 flags 不同；`nm -m` 为 `(from libFoo)`。
- **第二稿增加的反例**（B-P112-2；第三稿逐项写清构造与最早失败的规则）：
  - **段尾的七例**（vmsize = `0x40`，节覆盖整段；每例都是 T(parser)，最早失败的规则为 A4 有效性的第 2 条）：

    | X | 剩余 vmsize − X |
    |---|---|
    | `0x39` | 7 |
    | `0x3a` | 6 |
    | `0x3b` | 5 |
    | `0x3c` | 4 |
    | `0x3d` | 3 |
    | `0x3e` | 2 |
    | `0x3f` | 1 |

  - **其余的范围**（各为 T(parser)）：
    - 段 2 的 vmsize 为 4（< 8），节 `__got` 同步缩为 [`0x100004000`, `0x100004004`)，使它仍在段内，X = `0x00`：最早失败的规则为 A4 有效性的第 2 条（第四稿更正）。若不同步缩小节，节会超出所属的段，最早失败的规则是 A3；
    - 段 2 分为两个节，`__got` [`0x00`, `0x20`) 与 `__other` [`0x20`, `0x40`)，X = `0x1c`，[`0x1c`, `0x24`) 跨过节的末尾但仍在段内：A4 有效性的第 5 条；
    - 段 2 的 initprot 为 r--：A4 有效性的第 4 条；
    - 段 2 的 initprot 为 rwx：A4 有效性的第 4 条。
  - **算术**（各为 T(parser)，写明最早失败的规则）：
    - 段 2 的 vmaddr 为 2^64 − `0x20`、vmsize `0x40`：vmaddr + vmsize 溢出。最早失败的规则为 **A3**（段的范围）。A4 中“地址 = vmaddr + 偏移不溢出”这一核对，在 A3 与 A4 有效性第 2 条都成立时不可能单独到达；本例不声称覆盖它，它是防御性的核对。
    - X = `0x08` 之后接 `ADD_ADDR_ULEB` 2^64 − 4：偏移的加法溢出，在任何 `DO_BIND` 之前。最早失败的规则为 A4 的算术。
    - `SET_SEGMENT_AND_OFFSET_ULEB` 的 ULEB 为 11 个字节：A4 的 ULEB 规则。
    - ULEB 的第 10 个字节为 `0x02`（值超过 64 位）：A4 的 ULEB 规则。
    - X = `0x00`，`DO_BIND_ULEB_TIMES_SKIPPING_ULEB` count = 2^62、skip = 8：(count − 1) × 16 溢出。最早失败的规则为 A4 展开的预检，不逐条展开。
    - X = `0x30`，count = 3、skip = 0：最后一条的偏移 `0x30` + 2 × 8 = `0x40` > vmsize − 8 = `0x38`。最早失败的规则为 A4 展开的预检，不产生任何记录。
  - **展开与计数**（各为 T(attribution)，规则 A5）：
    - X = `0x00`，以 count = 2、skip = 0 代替 `DO_BIND`：两个位置 `0x00` 与 `0x08`，各自的 8 字节都在 `__got` 之内，都合法。两条 `_waitid` 记录，不唯一。
    - X = `0x00`，以 count = 0 代替 `DO_BIND`，没有其他 `_waitid` 记录：不产生记录，缺失。
  - **流的语法**：
    - DONE 之后有非零字节；没有 DONE；
    - `weak_bind_size` 不为 0；`lazy_bind_size` 不为 0；
    - `SET_TYPE_IMM` 为 2。
  - **Apple 输出的语法**（以文本构造）：
    - 文件名行带 ` (architecture arm64):`；
    - `-L` 的行带 `, weak`；`-L` 的行缺少版本的后缀；
    - `-l` 的块数与 ncmds 不同；
    - `-h` 的数值行只有 7 个字段。
- **预期的来源**：每个对照的预期类别与失败的规则（A1 ~ A9）都写在测试表中，按上面的规则预先确定，不调用被测的解析器生成预期。
- **分开记录**：下面四项分开记录、分开计数：
  - 方法的对照（M1 ~ M5）；
  - 解码器的模型对照；
  - 实际产物的解码；
  - 原生的 E2 调用。
- **作者的本机核对**（实施时，不是诊断的证据）：在交叉构建的两个架构的测试二进制上运行解码器。本机没有 macOS，otool 与 `nm -m` 的对照不能在本机执行。

**时间与预算**（第二稿，Review 112 §6）：

- **静态步骤**：构建、`p5bind` 的构建与运行、otool 与 nm，都在 `go test` 之外，受作业的 20 min（`timeout-minutes: 20`）约束。
- **测试二进制**：`go test ./...` 为每个包分别构建并运行一个测试二进制，`-timeout 15m` 对每个测试二进制分别计时（Go 1.27.3 `cmd/go/internal/test/test.go:345` 起）。
  - 根包的原生、旧回归、新的直接检查与 R-B2-2，共享根测试二进制的 15 min；
  - **`p5bind` 包是另一个测试二进制**，有它自己的 15 min 上限，不在根包的截止之内。
- **`p5bind` 的模型测试**：只处理以代码构造的小映像，预期远少于 1 s；不使用 D，也不影响 `TestP5` 的 D。
- **总上限**：作业的 20 min 是全部步骤唯一的总上限，不因新的解码器或回归而增加。作业超时即为失败，不重跑；不为此删除旧测试或放宽 `exact`。

**版本的关系**：

- 第一版（§16.4 的原表）与它在 `0213fb7` 上的 T，保持原样。
- 第二版只适用于获得授权之后的新固定代码点。记录带 `v2`，`WP25P5` 汇总的静态一项随之增加新的 ID（只打印；N3 见 §16.17.6）。
- `P5.static.symbols` 沿用 ID，但第二版只判定跳板与未定义导入的存在；归属由 `P5.static.binding` 判定。

**由固定源码推出的预测**（只是预测，不用于判定；实际产物与它不同时，按上面的规则判定）：

- 头标志没有 `MH_TWOLEVEL`，有 `MH_DYLDLINK` 与 `MH_PIE`；
- 库加载命令都是 `LC_LOAD_DYLIB`，其中有 `/usr/lib/libSystem.B.dylib`；
- `_waitid` 的 bind 记录恰有一条，ordinal 为 libSystem 的位置，标志 0，类型为指针，加数 0，位于数据段的节中；
- 符号表中未定义的 `_waitid` 的 `n_desc` 为 0；
- `nm -m` 为 `(undefined) external _waitid`，没有 `from`。

**限制**：

- 静态检查读取的是 `go test -c` 的产物；实际执行的测试二进制由 `go test` 另行构建。第二版记录前者的大小、SHA-256 与 build ID，不与后者比较（N1）。
- 运行器上的 dyld 与工具不声称与上面的公开源码相同。
- libSystem 内部 re-export 的解析与跳板指令的反汇编，都不在范围之内。

#### 16.17.5 新回归复用模型时的命中握手、释放与完成核对

适用于 §16.17.2、§16.17.3 中新增的模型检查与直接检查（Review 111 N2 的建议只用于新增的检查）：

- **命中握手**：会阻塞或注入失败的假对象，在进入时经通道发出信号，或记录调用的序列。检查先以截止等到这个信号，或核对调用的序列，再断言阻塞或失败的状态；不以 `Sleep` 代替。
- **释放**：阻塞的假对象都有一个释放闸门，在 `t.Cleanup` 中关闭，所以失败的出口也会释放。释放闸门只适用于假对象。
- **完成核对**：检查启动的每个任务，在检查返回之前以截止等它的完成信号，未完成即失败。
- **真实的子进程**（第二稿；第三稿补全）：直接检查自己启动的真实子进程，按 §16.17.3 R-B2-2 的执行协议收尾：
  - 先登记，再观察与断言；
  - 每侧一个非并行的子测试，收尾登记在本侧的 `t` 上，只执行一次；
  - 持有者的终止请求只经请求任务（R0 ~ R5），与唯一 Wait 任务（W1 ~ W6）按控制锁串行；被测链返回之前，两者都不开始；
  - 按截止查询，父端不同步调用 `kill`；
  - 未确认为 T / unknown，并移交持有登记 `p5RealHeld`；
  - （第四稿）每个新建请求或启动 Wait 的决策点都重新读取时刻、按该路径的截止复核，已到期即走统一的到期出口；R-B2-3 的模型以注入的时钟与可释放的假读取核对这一点；
  - 不同步无限等待。

既有的 `TestP5ProtoRequestHung`、`TestP5ProtoRequestSerial` 不在本次修复中改动，N2 不因此关闭。

#### 16.17.6 Review 111 的非阻塞项（只登记，不关闭）

| 项 | 内容 | 本计划的处理 |
|---|---|---|
| N1 | 静态检查的产物 `$RUNNER_TEMP/wp25.test` 与实际执行的测试二进制不是同一个文件；只有相同的源码、版本、`CGO_ENABLED`、架构与构建元数据，没有字节哈希或 build ID 的相等证明 | 保留这条限制。第二版记录静态产物的大小、SHA-256 与 build ID，不把“同源码、同设置”写成同一个文件；不为统一文件而减少验证 |
| N2 | `TestP5ProtoRequestHung` 没有确定的命中握手；`TestP5ProtoRequestSerial` 的假 Wait 循环没有释放到完成；若干失败出口没有统一的释放 | 新增的检查按 §16.17.5；既有的两项不改，N2 保持未关闭 |
| N3 | `p5StaticFile` 忽略 Scanner 与 Close 的错误，容许覆盖与未知项；`p5Summary` 只打印、不裁定 | 不改。第二版只把新 ID 加入打印的列表；有效性仍由 workflow 的真实状态与 `failed` 标记决定。以后若要用这个文件决定有效性，须先收紧解析与错误传播 |

#### 16.17.7 实施的顺序与待负责人决定的事项

**实施的顺序**（获得授权之后）：

1. 先提交登记：授权、本计划的提交 SHA、实现起点。
2. 再写代码：
   - B1、B2 的修复与回归；
   - 第二版的静态步骤与 `p5bind`；
   - 本机可以做的核对：
     - 模型检查（标签 `wp25p5local`）；
     - 解码器在交叉构建产物上的运行；
     - Windows 与 WSL Linux 的全量。
3. 推送一次，头提交即为新的固定代码点。
   - 不在触发分支上逐次推送实现的中间状态；
   - 不把多次诊断解释为同一次授权（Review 112 §7）。

**待负责人决定**：

- **授权实现**：B1、B2 的修复与回归，第二版的符号检查，以及 workflow 的相应改动。
- **推送即诊断**：workflow 在本分支上、改动原型或 workflow 的推送即运行（§16.10，没有手动触发）。所以代码的推送本身就是一次诊断，实现与诊断的授权不能分开推送。可选的做法：
  - (i) 一并授权实现与一次诊断（作者推荐）；
  - (ii) 先在不在触发列表中的分支上提交代码供审查，之后再授权推送到本分支取得诊断。
    - 那个分支须确实不在触发列表中；不为审查而改动触发的规则；
    - 推送到触发分支时，核对源码与 workflow 和受审的对象相同，并重新记录取证点；
    - 常规的 PR CI 与原型诊断是两套流程。

    这改变的是取证的流程，须负责人明确。
- **回归先行**：是否先以只含回归的代码点另取一次诊断，确认回归在 darwin 上按预测失败。
  - **前提**：
    - 须另一次诊断的授权；
    - 须固定旧代码的最小适配：R-B1-1 依赖的停止任务状态与 B2 的消费函数，在旧实现中都不存在。适配只为使回归可以编译、真实的路径可以命中，不提前修正判定。
  - **无效的检出**：编译失败、方法的静态步骤先失败、目标没有运行，都不是有效的检出（Review 112 §7）。
  - **作者的建议：不另取**。B1、B2 由源码证明，回归方向按源码推出。
  - 这是证据路径的选择，不冒称已动态验证旧代码会失败。

#### 16.17.8 新诊断（如获授权）的停止条件

- §16.12 的停止条件照旧适用。
- 另加以下情形，出现时按计划记录与收尾，然后报告，交负责人决定：
  - 第二版的任何静态检查为 T（包括 `P5.static.method` 的对照）；
  - R-B1-1、R-B1-2、R-B1-4、R-B2-1、R-B2-2 或 R-B2-3 的任何一项不成立；
  - `p5combo.close-error` 因截止而未执行；
  - 最终收尾中，读端的关闭未返回或返回错误（释放未确认）；
  - （第二稿）R-B2-2 的任何一侧：
    - 未执行（包括前提不成立与未准入）；
    - 被测链到 q_k 仍未返回；
    - 收尾函数到 d_k 仍未确认回收（子进程未回收、责任没有解除）；
  - （第三稿）测试结束时 `p5RealHeld` 不为空，即有移交而未确认的一侧；
  - （第四稿）R-B2-2 的任何一侧在决策点上走了到期出口（正常路径或收尾）；R-B2-3 的模型对照有任何一项不成立；
  - （第二稿）`p5bind` 的模型对照有任何一项不成立。

不扩大预算、更换入口、减少次数或放宽判定，不重跑。

### 16.18 修复的实施：登记（在写任何代码之前提交）

- **授权**：负责人 2026-10-31 采纳 Review 115（review-resolution §143）：
  - 授权按 §16.17 第四稿实施固定范围的修复；
  - 选择 §16.17.7 的流程 (i)：实现完成之后，一次推送取得一个新固定代码点的一次诊断；
  - 不另取只含回归的先行诊断：旧代码的缺陷保留源码证明，不宣称已动态验证旧代码会失败；
  - 不授权合并 PR #44。
- **固定的身份**：

  | 项 | 值 |
  |---|---|
  | 所依赖的计划 | §16.17 第四稿，受审提交 `cd0d4f7a7a5243e1ee1cbe24cde327db7fd30af8`（树 `d844e6780df8e4785298100d58ca4da3e4758984`，Review 115 的对象） |
  | 实施起点 | PR #44 的 HEAD `cd0d4f7`，分支 `test/wp2.5-prototype-p5`；之上只有 Review 115 的归档（`5e6169c`）与本登记 |
  | base | master `021c956c33a65cc17ad050bda34883df355adcd0` |
  | 原取证代码点 | `0213fb706e29130110af7ad8971dd8bbd94dd1d4`（首次诊断 run 41055799521，failure / invalid（T），不回改） |
  | 工具链与依赖 | `go1.27.3`、`golang.org/x/sys v0.49.0`，不新增依赖；两个 macOS 作业 `CGO_ENABLED=0` |
  | 平台 | `macos-latest`（arm64）与 `macos-26-intel`（x86_64），同一个固定代码点；Ubuntu、Windows 作业照常运行既定的回归 |

- **实施的范围**（只按 §16.17 第四稿）：
  - **Review 111 B1**（§16.17.2）：`p5Inner` 持有读端，唯一的停止任务；真实与报告的关闭结果分开，释放的知识；门禁先消费关闭结果再判定；移交谓词；最终收尾对读端的消费；`read_close` 记录。
  - **Review 111 B2**（§16.17.3）：每次读取的分类、动作之前的标记检查、终止请求的 errno 分类、责任集合与轮询、持续累积的错误；保险判定、门禁的问题、补救的结局与最终汇总这几处消费，以及所有直接调用者。
  - **回归与模型**：
    - R-B1-1：两个既有门禁的必要条件（不增加运行）；
    - R-B1-2：组合 `p5combo.close-error`（一次真实的 `runP5Gate` 运行）；
    - R-B1-3、R-B1-4：模型；
    - R-B2-1：模型；
    - R-B2-2：darwin 的真实子进程（`TestP5CleanupReal`，两个非并行子测试，各一个受控子进程）；
    - R-B2-3：模型（注入的时钟与可释放的假读取）。
  - **符号检查的第二版**（§16.17.4）：test-only 的 `p5bind`（`prototype/wp25/p5bind`）及其模型对照、方法的自检 M1 ~ M5，以及 workflow 中两个 macOS 作业的静态步骤。
- **必须兑现**（Review 115 §7 与负责人的决定）：
  - 请求与回收按控制锁串行、唯一 Wait、请求先登记、每侧（每个内层）至多一个 pending 请求，父端与持有者不取控制锁、不同步调用 `kill`；
  - 两个非并行子测试，Cleanup 登记在本侧的 `t` 上，只执行一次；
  - `p5RealHeld` 的持续持有、移交时的知识与当前快照分开、晚到结果只写入任务自己的状态；
  - 每个新任务的决策点（新建请求、启动 Wait、启动被测链）都重新读取时刻，按 c_k / w_k / d_k 复核，已到期即走统一的到期出口；
  - 模型与真实路径调用同一组决策与消费步骤（平台无关的协议核心，构建约束 `darwin || wp25p5local`）。
- **不变的部分**：
  - 原生 39 次、布局 3 次、既有门禁 2 个及其预测与判定；
  - 旧回归、D、角色 B / B′ 的截止、角色 C、`exact`、归属规则、调用次数与总预算；
  - Q-3 (a)、Q-4 (a)、H4 / H6 的限制声明与支持承诺。
- **另列的计数**（不计入上面任何一项）：
  - 组合：macOS 由 16 增为 17（`p5combo.close-error`），不计为门禁的通过；
  - 真实回归：R-B2-2 的两侧；
  - 模型：R-B1-3、R-B1-4、R-B2-1、R-B2-3，以及 `p5bind` 的模型对照；
  - 方法的自检：M1 ~ M5（`P5.static.method`）。
- **本机的核对**（作者的证据，与 CI 的原生证据分开）：
  - 模型检查（标签 `wp25p5local`）；
  - 两个架构的交叉构建，以及 `p5bind` 对产物的解码（本机没有 macOS，otool 与 `nm -m` 的对照不能在本机执行）；
  - Windows 与 WSL Linux 的全量；
  - 本机的 Go 是 1.27.1，不是固定的 1.27.3，本机结果不作 1.27.3 的证据。
- **诊断的政策**：
  - 所有实现、回归与本机核对完成之后，一次推送到本分支；该次推送的头提交即为新的固定取证代码点，由既有的触发方式取得一次诊断。
  - 不逐次推送实现的中间状态，不手动触发，不重跑旧 run，不把多次诊断算作一次授权。
  - 停止条件按 §16.12 与 §16.17.8：出现时按计划记录与收尾，然后报告，交负责人决定；不自动修改之后再推送取证，不重跑，不扩大预算、更换入口、减少次数或放宽判定。
  - 结果回填只增加文档提交，写明计划 SHA、新取证代码的 SHA 与树、run 与 attempt、各作业实际检出，以及最终 PR HEAD 与取证点的关系；并核对 `prototype/` 与 `.github/` 相对取证点没有变化。
- **本节在写任何代码之前提交**。不偏离方案的补充，在代码提交之前登记于 §16.19；实施若须偏离 §16.17 第四稿，先报告负责人，不自行改变方案。
- **不授权**：合并 PR #44、规范冻结、生产实现、G5 整体满足、WP2.5 / M2 的验收；原实现的 B1、B2 只有在实际修复与独立复审之后才可能关闭，本节不宣布关闭。

### 16.19 写修复代码时登记的补充（在代码提交之前）

下面各项是写代码时确定的，在代码提交之前登记。除了补充 9（负责人另行决定的偏离），它们都不改变 §16.17 第四稿的规则、预测、次数、期限、角色 C、`exact`、归属或判定，也不放宽失败关闭。

- **补充 1：代码的位置**。
  - B1 的三个小函数（`p5CloseProblem`、`p5HandOver`、`p5FinalRead`）、`p5Inner.closeRead` 与停止任务的两个结果，按 §16.17.2 第 9 条写在 `p5proto_test.go`。另有 R-B1-1 的核对 `p5CloseNormal` 与 `read_close` 的记录。
  - B2 的清理链（`p5EndChain`）与它的消费函数（`p5InsureAccount`、`p5AfterInsurance`、`p5RemedyOutcome`、`p5FinalProcs`），以及 R-B2-2 的持有者（`p5Holder`：R0 的决策、`chainStep`、正常路径第 3、4 步的 `tail`、收尾函数 `cleanup`）与持有登记 `p5RealHeld`，写在新文件 `p5clean_test.go`。它与 `p5proto_test.go` 同属平台无关的协议核心，构建约束同为 `darwin || wp25p5local`。
  - darwin 的接线：门禁与组合在 `p5gate_darwin_test.go`，最终收尾与汇总在 `p5_darwin_test.go`，R-B2-2 在新文件 `p5real_darwin_test.go`。
  - 模型：R-B1-3、R-B1-4 在 `p5proto_check_test.go`（`TestP5ProtoCopyStop` 改经 `closeRead`，新增 `TestP5ProtoCloseOnce`、`TestP5ProtoCloseLate`）；R-B2-1、R-B2-3 在新文件 `p5clean_check_test.go`（`TestP5CleanChain` 七个子项，`TestP5HoldExpiry` 三个对照）。
  - 真实的 R-B2-2 与 R-B2-3 的模型调用同一组 `p5Holder` 方法；门禁、最终收尾与模型调用同一组消费函数。
- **补充 2：错误累积器的去重**。清理链轮流读取多个对象，相同的文字不一定相邻。所以 `p5Errs` 增加方法 `note`：按文字在已有的全部项中去重并计数（§16.17.3 的“按文字去重、计数”）。Wait 任务 W1 的 `add`（只合并相邻的相同文字）不变，`TestP5ProtoErrs` 不变。
- **补充 3：回收的四种状态由 Wait 任务自己的状态推出**。持有者从不取控制锁，所以“已开始回收”取自 W3 在状态锁内的记录。W2 置标记与 W3 记录之间的极短时刻，显示为“Wait 已启动、未开始回收”。这只影响记录的措辞，不影响 PID 动作（只按控制锁内的标记）或有效性的裁定。
- **补充 4：R-B2-2 两侧的接缝**。两侧的接缝都替换这个子进程在**轮询中第一次读取**的答案（`p5EndChain` 的读取分为进入、轮询与标记读取失败之后的重读三种）。保险清理一侧即“第一次轮询读取”；补救一侧，这次读取就是“动作之前的身份读取”。真实的读取照样进行，它的答案另外记录。
- **补充 5：R-B2-2 的细节**。
  - 身份读取要求子进程存在、是本进程的子进程、不是僵尸；否则走身份读取失败的出口（不运行被测链；在决策点按 w_k 复核之后按 R0 请求、启动 Wait）。
  - 每侧打印一行 `WP25P5REAL`（截止、身份、接缝的命中与真实的答案、链的结果、调用者的消费、收尾之后的知识、回收的状态、到期出口、移交与失败），由本侧的收尾函数在最后打印。
  - `WP25P5` 增加 `combos`（`p5combo.close-error`）、`real`（两侧）与 `real_held`（持有登记的两种视图，没有移交时为空列表）。这些只打印，不裁定（N3）。
- **补充 6：最终收尾的记录**。复制任务已返回时，总是记下“the copy returned …”（原先只在有停止任务或有字节时记录）。读端由 `p5FinalRead` 消费。
- **补充 7：静态步骤的写法**（§16.17.4）。
  - 步骤仍写在 workflow 中（不另设脚本文件），在作业的 `bash -e -o pipefail` 下运行。命令替换都写在 `if` 的条件中，不用 `|| true`。
  - **文件名**：otool 的 `-l` 与 `-L` 的输出分别存为 `otool-loads.txt` 与 `otool-libs.txt`。原因：macOS 运行器的文件系统默认不区分大小写，`otool-l.txt` 与 `otool-L.txt` 会是同一个文件。本机演练（NTFS 同样不区分大小写）中，`-L` 的输出覆盖了 `-l`，A9 因此为 T(parser)。这是在推送之前发现并修正的脚本缺陷。
  - `P5.static.method` 不成立时的类别记为 `T(contradiction)`（对照没有给出预期的类别）。
  - `p5bind` 的预期架构取自 `go env GOARCH`（`amd64` 写作 `x86_64`）。
  - `p5bind` 在最后打印一行 `WP25P5BIND-VERDICT <类别> <规则>`，供步骤自己的匹配函数读取，不依赖 JSON 工具。
  - 各工具的退出状态经参数交给 `p5bind`，A9 的“非零为 T(tool)”在解码器中判定。
  - 工具的路径、哈希、产物的大小、SHA-256 与 build ID 都写入 `P5.static.artifact` 的结论。
  - A9 所解析的五份输出完整打印在日志中；`go tool nm` 只打印含 `waitid` 的行（判定只用文件上的匹配）。
- **补充 8：解码器的写法与规则名**。
  - 解码器自己按 `<mach-o/loader.h>` 的布局读取头与加载命令（`encoding/binary`），`debug/macho` 只提供常量。
  - A4 的子规则记为 `A4.v1` ~ `A4.v5`（有效性的五条）、`A4.leb`（ULEB / SLEB）、`A4.arith`（算术）、`A4.expand`（展开的预检）、`A4.stream`（流的语法与未知的表示）。
  - A3 的“不认识的库加载命令”按保守的写法实现：不在 `<mach-o/loader.h>` 已知列表中的任何加载命令都记为未知的表示（T(parser)），因为无法判断它是否加载库。加载命令的 cmdsize 须为 8 的倍数并落在 sizeofcmds 之内，否则解析失败（T(parser)）。所测的产物只含已知的命令。
  - `p5bind` 的模型对照是另一个测试二进制，它的结果不进入 `WP25P5` 的模型一项（那里只有根包的模型），而是 `go test` 中 `p5bind` 包自己的结果；它失败时作业失败，属于 §16.17.8 的停止条件。
- **补充 9：A3 与不映射的 `__DWARF` 段**（偏离，作者先报告，负责人 2026-10-31 决定）：
  - **发现**：本机交叉构建的两个架构的测试二进制中，`__DWARF` 段的 vmsize 为 0，它的七个节大小不为 0、标志为 `S_ATTR_DEBUG`（`0x02000000`）。本机的 Go 是 1.27.1，但 `src/cmd/link/internal/ld/macho.go` 与固定的 1.27.3 逐字节相同（SHA-256 `665c2270…`，§16.2）：`:753` ~ `757` 把 `__DWARF` 写为不映射的段（`ms.vsize = 0`），`:651` ~ `653` 给它的节加 `S_ATTR_DEBUG`。
  - **后果**：按 §16.17.4 A3 的字面（每个节须完全落在所属的段内），真实产物在两个架构上都会确定地在 A3 得到 T(parser)，`P5.static.binding` 为 T，新诊断因这一已知原因 invalid（T）。
  - **负责人的决定**：放过不映射的调试段。
    - 包含的规则只用于 vmsize 大于 0 的段；
    - vmsize 为 0 的段，只在它的每个节都带 `S_ATTR_DEBUG` 时接受，否则仍是 A3 的 T(parser)；
    - 这样的段不能容纳 bind 记录（A4 有效性的第 2 条要求段的 vmsize 至少为 8），所以 A4 ~ A9 不变。
  - **新增的模型对照**（在基准映像的 `__LINKEDIT` 之前加一个不映射的段）：它的节带 `S_ATTR_DEBUG` 时为 ok；不带时为 T(parser)，规则 A3。`TestBind` 的子项因此由 52 增为 54。
  - **作者本机的核对**（不是诊断的证据）：用这一写法的解码器，两个架构的产物 A1 ~ A8 都成立：`_waitid` 的 bind 记录恰有一条，ordinal 1（`/usr/lib/libSystem.B.dylib`），位于 `__DATA_CONST,__got`，标志 0、类型为指针、加数 0；符号表中未定义的 `_waitid` 的 `n_desc` 为 0。A9 需要 Apple 的工具，本机不能执行。
