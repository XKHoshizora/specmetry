# Review 148：PR #56 的 Review 147 定向修复复审

审查日期：2026-11-14。审查方式：只读源码、Git、已有日志与 GitHub 元数据；没有运行 Go、测试、构建、扫描器、采集器或实验。

## 1. 结论

**第一阶段实现仍为技术 NO-GO，当前不建议合并，也不建议以本阶段已经技术通过为依据进入真实调用或正式取证阶段。** B-147-2、B-147-3 可以在原受审范围关闭；B-147-1 的工作任务主体修复成立，但截止接线仍有三处具体残余。B-147-4 的来源记录与 F 域修复成立，其新增核验器接入父端的方式与 B-147-1 共享一处阻塞，不能以记录功能已经补上宣布整体接线通过。

| 项目 | 本轮裁定 |
|---|---|
| B-147-1 | 不关闭。R-a：父端到期出口重新执行来源核验；R-b：放弃等待／工作完成资格没有持续进入终判，LC-1 晚到仍可命中；R-c：后台夹具的准备／清理缺少外层截止准入 |
| B-147-2 | 关闭：manifest 与记录文件的实际句柄持续持有、唯一关闭、错误与释放知识分开成立 |
| B-147-3 | 关闭：ReadWhole 的全部调用者消费读取与关闭错误；类型检查的预算出口保存此前真实失败 |
| B-147-4 | 来源与 F 域主体修复可采；整体暂不关闭，剩余的是与 B-147-1 共计一次的 R-a，而不是再列一项相同缺陷 |
| N-147-1／2／3 | 在原限定范围完成；LC-1 集合修复与其发布截止残余是不同问题 |
| 16 项解释与此前已认可实现 | 沿用 [Review 147](codex-review-147.md) §4、§5 的限定；本轮不重新审判未受影响的规则 |
| 下一阶段准备 | 可以交负责人决定是否授权本报告列出的最小修复；尚不适合直接转入真实调用／正式运行授权 |

这些结论不以 LC-9 尚未实施、真实调用仍停止、尚无三平台运行或预算测量为阻塞理由。反过来，正式配置因这些未交付项必然拒绝，也不能遮住健康构造配置中已经可达的错误出口。

## 2. 固定身份、授权及变更范围

受审仓库为 `XKHoshizora/specmetry`，PR 为 [#56](https://github.com/XKHoshizora/specmetry/pull/56)。开始核验时间为 2026-11-14 16:07:22 +09:00。

| 身份 | 核验值 |
|---|---|
| 分支 | `proto/wp2.5-c9-evidence-tool` |
| 本地 HEAD、远程分支、PR head | `fcf41b02a341ae35b59bcc43ad80464bd190e54c` |
| HEAD tree | `9fa336615ccf3050026d140f446a530c3f6f2764` |
| PR base、远程 master | `9137279554deed5a542dbd61b871d1cfa79bc323` |
| 上轮对象 | `84c7718257921bb251326f753b23aed01fcac9b9` |
| 工具模块子树 | `3b6b26ddafad5beae05a8bafca60ca371c84eb55` |
| 开始工作区 | 干净；Review 148 文件不存在 |

实际提交顺序为 `a390fda → b44203e → bf0a01c → ec07dec → fcf41b0`。归档、§191 授权登记先于修复；`ec07dec` 只改 controls 的回归测试，最终提交只作记录与状态同步。独立核对 `ec07dec:prototype/c9evidence` 与最终 HEAD 的对应子树完全相同。

本轮完整增量为 34 个文件，2331 行加入、236 行删除：工具模块代码／测试、Review 147 归档、实现记录及必要状态说明。根模块实现、测试、依赖、冻结正文、验收项、workflow 未在本轮增量改变。技术规则未因状态同步被改写。

[§191](../review-resolution.md) 的旧历史前缀逐字节保持：旧文件 1,343,584 字节是当前 1,352,346 字节文件的精确前缀。Review 147 的归档提交只增加该报告；当前文件与 Git blob 均为 28,760 字节，SHA-256 为 `9e337d989167844dfd9c745306f2ac14be3d98ad54d1c1edf5ad681ae661ce25`，与原文登记一致。`git diff --check 84c7718 HEAD` 唯一提示是归档报告第 196 行的末尾空行；保持原文，不据此修改历史，也没有把这条提示当作其他差异检查已通过。

授权仍是第一阶段定向修复。§190 已采纳本工具的 N1／LE-10 退出映射，不能倒退写成全部 LE 都未采纳；E-10(ii)、LE-5 ①／②、N2 和正式运行仍未获授权。路径 1／T-1／L-b1a 不扩大到 T-2／L-b1b 或原始入口／发送信号风险。

## 3. B-147-1 的成立部分与残余

以下源码位置均指固定 HEAD `fcf41b02a341ae35b59bcc43ad80464bd190e54c`，链接不指向本机绝对路径。

### 3.1 已成立的主体修复

[job.go](../../prototype/c9evidence/internal/job/job.go) 209–253 行先登记 WorkRecord 再启动 goroutine；测试函数只等完成通道或 W。未返回的类型检查不再直接把 `m.Run` 卡在同一调用栈。任务闭包持有 Job、配置和执行中的输入，真实晚到结果继续进入 R／F 与内存记录，不调用已结束的 testing.T。

320–359 行在 R／F 发布的同一次锁下读取时刻，要求早于 W 且尚未放弃等待；等于 W 按晚到处理。消费时晚到的 F Hit 降为 Pending，晚到 R 加预算类失败，原真实失败仍在。完成通道与 W 同时可读时，单个 R／F 的资格不依赖 select 偶然选择；全部已经及时发布的结果也不会仅因父端晚调度被修改其历史资格。

这些修复真实解决了上轮的同步入口与最后一个配置结果晚到问题，但不覆盖下面三个接点。

### 3.2 B-147-1-R-a：到期出口重新同步执行整条来源核验（与 B-147-4 共用）

**位置**：[job.go](../../prototype/c9evidence/internal/job/job.go) 445–454、534–535、559–569 行；[provenance.go](../../prototype/c9evidence/internal/collect/provenance.go) 27–53、85–108 行；[closeout.go](../../prototype/c9evidence/internal/chain/closeout.go) 238、250、254、357–370 行。

`copyState` 在持有 `j.mu` 时调用 `VerifyProvenance(j.rec.Items())`。这不是读取一个已发布的核验结果：它重新对记录字节计算 SHA-256、解码每条派生 JSON、扫描此前记录集合并检查父项。输入包含采集记录，其处理量随记录数量和字节量增长。`Finish` 同时把它接到 `Copy` 和 `Level1`。

**可达顺序**：

1. 工作在 W 被父端放弃等待，或 `m.Run` 返回时父端才获调度，当前已经到 D。
2. `Closeout` 已经识别 `Late`，进入 `expiredLocked`。
3. `expiredLocked` 在外层锁下调用 `Level1`；后者取得 `j.mu`，复制记录并重新核验整条来源链，完成后才能返回。

另一条正常收尾路径在 `cfg.Copy()` 中做同样处理，完成后才重新检查 D。将分析移到工作 goroutine，并未使这些新增分析获得父端截止出口；外层状态锁还被整个核验占用。

**违反规则与影响**：限定计划 §6.4 的统一到期出口只消费已发布状态；实施计划 §12.14 的同步分析不能决定父端何时脱离。此处不是要求短临界区或一次时钟读取具有硬实时上界，而是在已经知道过期之后重新开始一项完整输入核验。现有小记录回归即使迅速完成，也不能证明这个接线符合该规则。

**最小修正**：把来源核验放进已登记、受工作截止约束且有持续持有者的工作阶段，发布核验结果、完整性与资格；父端的 Copy／Level1 只消费这些已发布状态。核验未完成必须拒绝，不能简单在到期时跳过核验并保留有效结论；晚到核验错误继续保留。新回归应核对实际 `RunDomains → Finish` 路径，而不只测独立核验函数。

### 3.3 B-147-1-R-b：工作放弃／晚到资格在 Done 后不再进入终判

**位置**：[job.go](../../prototype/c9evidence/internal/job/job.go) 229–233、248–250、355–368、457–465、491–492 行。

WorkRecord 保存 `Abandoned` 和 `Timely`，但 `copyState` 仅在 `State != WorkDone` 时消费工作未完成状态。Done 后两个字段均不影响终判。LC-1 又直接由 `lc1(j.rRecords)` 写入 `j.lc1`，没有其自身发布资格；消费时不经过 FRecord 的晚到降级。

**可达顺序**（采用已有健康集成测试所用的构造配置，不借正式 LC-9 的必然拒绝掩盖问题）：

1. F 结果及最后一个 R 结果都在 W 前发布；工作端在最后的 checks 记录／LC-1 发布之前被暂停。
2. 父端在 W 消费计时器，确认 Work 尚非 Done，写入 `Abandoned=true`，从 RunDomains 返回；随后父端暂未进入 Finish。
3. 工作端在 W 后、D 前恢复，追加最后的记录，计算并发布 LC-1=Hit，再发布 `WorkDone, Timely=false`。
4. Finish 看到工作已 Done，`allTerminal` 可成立；`copyState` 不再保留“已经放弃等待”的必要失败，LC-1 也按 Hit 消费。此前 R／F 资格都为及时，记录链及诊断若在 D 前成功，现有规则可以返回 3／4、退出 0。

等价变体是父端晚调度时完成通道和计时器同时可读：工作完成资格仍是 false，但消费端不看它。这个反例不同于“所有必要结果已经及时完成、仅父端晚醒”——本例在 W 时确有尚未完成的工作和未发布的 LC-1，且第一种顺序已经明确放弃等待。

**违反规则**：§191 第 2 项及 WorkRecord 自身承诺，W 先到失败返回，之后成功不能恢复有效性；不能只保存资格而不消费。当前 `TestB1471LastUnitCompletesAfterW` 检查配置本身晚到，`RealErrorAfterTheParentStopped` 在放行后台前已完成 Finish，均未覆盖这个“放弃之后、快照之前补齐”的窗口。

**最小修正**：将已成立的放弃／未及时完成事实持续进入最终输入；给必要的 LC-1／最终工作发布完整接上 W 资格，或用包含它们的工作完成资格统一守卫。保留真正及时完成、父端晚消费的健康分支。回归应精确暂停在最后 R 发布之后、必要的后续发布之前，断言恢复后台不能改善已经成立的失败。

### 3.4 B-147-1-R-c：后台夹具的新准备／清理仍能在已过 D 后开始

**位置**：[job.go](../../prototype/c9evidence/internal/job/job.go) 273–297 行；[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 219–255 行；[lc5.go](../../prototype/c9evidence/internal/controls/lc5.go) 298–311 行；[outer.go](../../prototype/c9evidence/internal/chain/outer.go) 125–132 行。

`outerAdapter.Begin` 只登记操作，不做准入；RegisterOp 也不检查截止。根夹具的 `MkdirTemp` 在首个 `budget.AdmitUnit` 之前执行。各夹具的 `cleanup()` 以及根目录的 defer 清理，直接在登记后调用 RemoveAll，未复核原 D。

**可达顺序一**：先登记工作任务；工作 goroutine 迟迟未调度。父端在 W 返回、进入 D 出口。后台此后才进入 runControls，仍会创建根临时目录，然后才开始拒绝各对照单元。

**可达顺序二**：一个 LC-5 实例在 W 前准入，其类型检查直到 D 后才返回；父端已退出等待。实例随后执行 `fx.cleanup()`，登记并开始新的 RemoveAll；最后的根目录 defer 还可再启动一次清理。它们不是此前已经开始的系统调用的晚到返回。

**影响**：将同步工作移入后台后，父端已可停止等待，但后台没有完整兑现“到期之后不再启动后续动作、未确认责任持续持有”的接线。O 域中记录了结果不等于动作已获得准入；最终非零也不能修复越过既定截止才开始的动作。LC-5 内部的假时钟用于模型预期，不是外层真实 W／D 的替代。

**最小修正**：为这两类实际 O 操作接上外层准入：准备按工作截止，收尾按既有 D；每次新操作决策都复核。到期不再创建／清理，保留已创建目录、未完成实例及清理未知的责任和真实结果。不要把准入失败表示为成功 Cleanup，也不要以重新取得一份模拟预算允许真实外部动作。无需改成可中断解析器或要求已经开始的 RemoveAll 硬实时返回。

以上三项都是当前受授权第一阶段的接点，不是要求实现尚未授权的平台子进程控制。它们按 B-147-1 的残余处理；R-a 同时阻止 B-147-4 的新增核验接线整体收口，不重复计数。

## 4. B-147-2：关闭

[writer.go](../../prototype/c9evidence/internal/chain/writer.go) 160–169、475–510、529–608 行可确认：

- manifest 的真实 File 在 Create 成功返回后立即保存到 `writer.task.manifest`，不是仅写一条 Holding 描述；外层和 Job 持有 writer，writer 终态不删除该任务结构。
- Create 后到期、Write 后到期均不启动新的 Close，实际对象仍可达；Write 错误、真实 Close 结果、CloseStarted 与 Released 分开记录。
- 正常路径唯一 Close，只有 nil 才清空实际句柄并标 Released；错误则保留对象及错误，不盲目重试。记录文件也按相同原则处理，`open=false` 是不再尝试关闭，不是已释放。
- 判定消费真实操作结果及资格，Holding 不替代关闭成功证据。GC、测试进程退出、artifact 存在均不被当作已确认释放。

[chain 回归](../../prototype/c9evidence/internal/chain/regress147_test.go) 覆盖 Create 跨 D、Write 跨 D、manifest Close 失败、记录 Close 失败和正常唯一关闭。其放宽后的到期断言并非只有“结论 2”：仍要求合法出口集合、writer=Expired/Late、无 Close、Holding 与 Created 结果保留；Write 例另检查写入结果。分支 (a) 只有诊断已经发布到期停止时可消费，分支 (b) 是尚无终态的到期出口。该放宽不构成降低所有权要求。

测试主要检查公开持有状态；实际对象而非描述的持有由上述源码直接确认。它们是修复后回归，不是修复前已经能编译的红绿对照，也不是原生文件关闭故障取证。

## 5. B-147-3：关闭

[guard.go](../../prototype/c9evidence/internal/collect/guard.go) 84–99、105–149 行的 ReadResult 分别保存 OpenErr、ReadErr、CloseErr、Data 与关闭尝试状态。`ReadWhole` 真实读取结束后唯一关闭，即使读取失败也保存 Close 返回值。全体实际调用者已核对：

| 消费者 | 结果 |
|---|---|
| ReadGuard 的 VERSION 与候选可执行文件 | 遍历 Errors，分别保留读取与关闭错误；不是只返回第一个 |
| engine.stageA 的 go.sum／go.mod | 每个操作错误单独进入 A4 J-1；原始 repo-files 记录同时保存 fileIdentity 的错误集合 |
| 字段校验、汇总、Job | 主分类不替代细节；checks 快照与序列化保留失败集合，预算失败独立存在 |

[typecheck.go](../../prototype/c9evidence/internal/typecheck/typecheck.go) 660–737 行的 `markBudgetKeeping` 在包／文件拒绝时保留已经得到的导入环、语言版本、读取、解析、包名与约束错误，只将尚未完成的检查标为预算导致的未评估。修改后的所有相关提前返回及向 collect 的转换都已追踪，没有发现再次以预算覆盖此前失败的直接回退。

现存原始日志提供两个读取调用者及三种类型检查错误的修复前失败、修复后通过；健康预算对照也保留。这里的错误来自受控读取／关闭接缝，不表述为自然发生的原生系统故障。关闭未知不等于释放确认；此结论不扩大到新的文件系统或事务保证。

## 6. B-147-4：主体修复成立，保留共享截止残余

实际生产者及消费者已逐项核对：

| 派生／消费路径 | 本轮结果 |
|---|---|
| closure → 第 3 种调用 | stageb 保存实际调用 Ref，核验要求本配置此前的 call3-list |
| 各 Context → 第 2 种调用 | b5 使用实际 call2-context Ref；不是以一个任意可解析父项代替 |
| consumption → 第 3 种调用及本配置 I-4 | 按确定次序加入原始模块记录；受控读取结果本身保留消费内容的摘要与相关字段 |
| findings → consumption／call3，comparison → findings | 两条链不再止于空 parents |
| checks → repo-files、toolchain、调用及已产生的派生记录 | Refs.All 连接已取得项；提前失败不虚构未发生调用 |
| 核验到 Job | 缺父项、错误摘要、其他配置、错误调用、后项引用、记录字节与 Ref 不符都会产生 defect；但是当前在父端重做，须修 §3.2 |

核验器与 artifact 使用同一个 MemRecorder 生成的 Line；Sink 交给记录链的是同一条内容（去 LF 再由链补 LF）。正常终态后不再有 R／F 生产者，因此正常封口有同一输入基础。后台未终态或封口后到达的记录仍在内存中保留，不能反过来承诺已经写入 artifact。这些是来源关联，不是源码来源认证，也不是对任意伪造输入的密码学证明。

[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 40–73、118–193 行及各 LC 生产者现在保存 Instance：Checks、Entries、Calls、Records、Refused、Extra；FRecord 保存完整 Outcome。Job 将结构化实例与完整预期一并写到 controls.jsonl。展示行由结构化检查生成，细节仍在实例中；站点参与比较，路径／配置只按固定夹具关系规范化，失败数量和来源／注入序号不被删去。

`ec07dec` 的 [OnlyTheSiteDiffers 回归](../../prototype/c9evidence/internal/controls/regress147_test.go) 58 行起，确实把 LC-4 的两个变体固定为相同检查、J-3、call3-list、syntax、real，只有 `$[0]`／`$[1]` 站点不同；自己的预期 Hit，交叉预期 Miss。原 LC5-19b／20a 测试同时改变检查与站点，只保留其原覆盖，不再用它单独证明站点维度。额外真实失败没有被展示转换去重或过滤；真实清理失败仍阻止有效命中。

以上来源与完整实例修复可采。**整体 B-147-4 暂不关闭的原因是核验实际接入终判时新增的 §3.2 截止问题**，不要求本轮增加源码全文、签名认证或正式调用。

非阻塞覆盖建议：当前 wrong-call／cross-config 单元反例还同时具有错误或空摘要。核验源码确实另行检查必要父项及配置；可以补一个所有摘要都正确、只交换语义父项的独立反例，使运行证据更精确。这不是据“还能加测试”新增 BLOCKER。

## 7. 非阻塞项及回归质量

- **N-147-1 完成**：Result.Modules 在快照成功与否之前记录 B4 应检查的闭包模块，LC-1 按该集合查 Snapshots/H1，未评估或失败而没产生快照的模块不会消失。正负对照覆盖缺少快照、H1 缺失、闭包未知及已算出但不匹配。§3.3 只涉及这个结果何时发布和消费，不否定集合修复。
- **N-147-2 完成**：`TestN1472TargetTagsReachTheConsumer` 用实际 collect 接线、假调用者、三组独立固定预期，检查目标 ToolTags 对文件选择的影响及所记录 Context。是接线单元／集成证据，不是正式 LC 扩展、真实 go list 调用或真实输入兼容性证据。
- **N-147-3 完成**：identity/run 保存实际 start、W、D 的 RFC3339Nano 及相对偏移，来自同一计算值；偏移为 1m50s／2m0s，预算未改变。
- 块注释之后同一行／下一行的说明与 cond、typecheck 的固定用例相容；措辞不再把两者混成同一行为。
- 工作回归等待均有十秒失败边界，并以 Cleanup 释放受控阻塞；它们不等于证明所有后台任务已在任何失败出口全部结束。最早的两份红色日志确有 TempDir RemoveAll 的 sharing violation，须保留，不能写成那次夹具收尾全部成功。
- manifest 的放宽断言见 §4；代码保留了比实现记录“只断言结论 2”更多的责任与结果断言，这一记录概括可精化，但不是新的阻塞。
- §8 已区分开发期间运行 Go 与采集器启动真实三种调用。Python／子任务等历史按已取得回执及 Review 147 范围保留；本轮不从“无输出”推出没有执行，也没有重新审计所有历史会话或把作者陈述升格为独立复现。

## 8. 本机已有证据核验

读取目录为 `C:\Users\hoshi\AppData\Local\Temp\claude\b96df3361c677053\67d963ef-6de7-4877-aebf-656e909830be\scratchpad\s191\logs`。本轮只读取日志和计算哈希，没有执行其中任何命令。

| 日志 | 字节数 | 本轮 SHA-256 |
|---|---:|---|
| final-full-test.log | 43053 | `1f3ac1413d0ffd12896ec9c9eeade1ead9f3658935b7a1590cd94f675defd099` |
| final-static.log | 2771 | `e0546c77894c63b426db63cf26b3081ab56449ed437569aea8a9fd1b71462996` |
| final-regress147-x20.log | 2136 | `bc53f6ce77c365e208b97ce24a2d547c2d1cba2c6d69bdefea636339b411793f` |
| pre-fix-regressions.log | 4515 | `089c92e3c6bafdfdb519edd20c644a495b1a4f388339e8f48732ae55b86024a9` |
| pre-fix-B1471.log | 2114 | `99209fedcab6f129ccbce8a3256838fc87fc98e61def9f5c2ce24ddf136756f1` |
| pre-fix-site-only.log | 1237 | `5d5d8a6e1c463a8c9412497e9e38724d38a8569807e5111170b66468364a48f6` |
| post-fix-full-test.log | 42748 | `f24e49842f6193f2530ff0b582b82bc9b273f81c734ea8a1d8f37035e4e7b46b` |
| post-fix-static.log | 2699 | `e0b0fd5aff53caf833a1a01b59c6304f17fee7e7fadc62f5d7725cbf8b6a3b24` |

八份长度／哈希与实现记录 §7.6／§7.7 一致。最终三份记录的代码点是 `ec07dec8aa5c56f00d861a6b2a07f0ff66cd9a39`，树为 `1baced68a771a0cd16bc28777cf71afd14837d77`，模块子树与本轮相同。测试前后工作区只显示实现记录文档未提交；工具模块无修改。记录给出命令、环境、工具链及退出 0。工具为本机 go1.27.3 windows/amd64，日志中的 go.exe SHA-256 是 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`。这些是日志中的身份与时点关联，不是测试二进制的不可变产物绑定，更不是本审查者重新运行所得。

直接从逐项 PASS／SKIP 行复算：最终全量有 **158 个顶层 PASS、271 个子测试 PASS、0 FAIL、1 SKIP、11 个 ok 包**；跳过项是 `TestTakeNegatives/symbolic_link`，本机无建符号链接权限。日志在 `TestStripBodiesCounterExamples` 中含被测类型检查的 panic 诊断，所属反例测试随后通过；不能把这条诊断漏写成未发生，也不能把它误计为 Go 测试进程的失败。run 包无普通测试文件。

静态日志包含 gofmt、普通及带运行标签的 vet、六个 GOOS/GOARCH 的交叉 vet、正式入口只编译、mod verify 与工具自身依赖查询；均记录退出 0。第三方导入记录只有 x/mod/sumdb/dirhash，未导入 x/sys，与本阶段不实现平台控制相容。交叉 vet 不是 Linux／macOS 运行。

20 次回归日志记录 `-count=20` 与选定正则、包级退出 0；chain、collect、controls、job、typecheck 有匹配测试，cond 等六包明确 `[no tests to run]`，不能把 11 个 ok 都写成执行了匹配回归。它没有逐次详细输出，不是 `-race` 或全调度覆盖。此前无日志的 30 次／20 次重复仅保留作者陈述。

修复前第一份记录为 8 FAIL、1 PASS；B-147-1 当时失败在未进入分析，不能采作原同步缺陷的反证。修正夹具时钟后的单独记录才失败在 RunDomains 未于 W 返回。两次均还有已记录的测试目录清理错误。站点旧实现探针在导出的旧模块加临时测试上交叉误命中，退出 1；其来源不是旧提交原封不动的测试集。B-147-2 与新核验器在旧实现不可编译，只有修复后验证，不补写红绿对照。

## 9. 常规 CI、执行边界与维护事实

独立读取 [run 43561012029](https://github.com/XKHoshizora/specmetry/actions/runs/43561012029) 元数据和八份完整作业日志：`pull_request`、attempt 1、completed/success，8/8。每份 checkout 均检出 `a130058ff605e7c3018568bdedc1f1824d400003`，不是只取 job.head_sha。GitHub commit 对象显示父提交依次为固定 base、固定 HEAD，tree 为 `9fa336615ccf3050026d140f446a530c3f6f2764`，与受审树相同。

| 作业 | job ID | 已核验回执 |
|---|---:|---|
| Web | 132344466642 | success；不安装／报告 Go，不补写版本 |
| Go tests Windows | 132344466849 | go1.27.4 windows/amd64；27 ok 包，10 SKIP；CLI 417.018s |
| Go tests Ubuntu | 132344466920 | go1.27.4 linux/amd64；27 ok 包，普通测试 15 SKIP；CLI 401.356s；独立 root CrossUser 8 项 PASS |
| Go lint | 132344467028 | go1.27.4 linux/amd64；tidy、vet、相对链接检查成功，lint 0 issues |
| Go tests macOS | 132344467032 | go1.27.4 darwin/arm64；27 ok 包，7 SKIP；CLI 129.552s |
| Dashboard macOS | 132344722018 | go1.27.4 darwin/arm64；构建、web 测试与 smoke 成功 |
| Dashboard Ubuntu | 132344722039 | go1.27.4 linux/amd64；构建、web 测试与 smoke 成功 |
| Dashboard Windows | 132344722074 | go1.27.4 windows/amd64；构建、web 测试与 smoke 成功 |

三个主测试作业的 ok 包集合相同：apperr、baseline、channel、checkrun、claim、cli、config、contract、datadir、domain/derive、gitscan、gitx、project、proposal、question、scope、secrets、source、store/sqlite、termtext、testutil、factgen、factgen/legality、factgen/toderive、readycheck、readycheck/fromdump、ulid；无 FAIL。普通 Ubuntu 测试使用 -race，但仅针对根模块，不能给独立工具补写 race 证据。

三平台共有七个 SKIP：`TestClaimWithARealGitThatFetches`、`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches`、`TestM14ReadOnlyProbe`、`TestM14OpenPath`。Windows 另有三项不可读存储／来源测试；Ubuntu 另有八项普通用户 CrossUser 跳过，随后独立 root 步骤实际通过 PathSwapCannotRedirectWrites、CannotReplaceDataDirectoryUnderSafeParent、StickyParent、AncestorOwnedByAnotherUser、ScratchUnderAWritableParent、ScratchUnderASafeParent、ScratchAncestorOwnedByAnotherUser、ScratchStickyParent。不能把 root 步骤算成又一套 27 包或三平台共同覆盖。

八份日志没有执行 `prototype/c9evidence`。根模块常规 CI 的 go1.27.4 回归与本机 go1.27.3 工具测试分别采证。耗时只作观测，不据此建立上界、诊断争用根因或关闭 M-16。

取证 workflow 本轮未改，仍仅匹配 `evidence/c9-run-1` 的 push，没有 PR／手动／定时触发。远程匹配 `heads/evidence/` 为空，按该运行分支查询现有 Actions 为 0；按固定 HEAD 查询只有上述常规 CI。源码 FormalConfig 仍使用 Stopped，LC-9 适用但未执行；没有自动回退到真实调用。开发期间确曾运行 Go，与采集器未启动三种真实调用不矛盾。

此前 CI 失败、单次重跑及维护事项按 Review 147 已取得的历史范围保留，本轮不重新全量读取历史日志，不以当前绿色改写失败或原因未知。

## 10. 最小下一步与结束边界

建议把后续最小修复限定在 §3 三个接点：来源核验的工作阶段与发布消费；工作放弃／最终资格的持续消费；后台夹具 O 操作的真实截止与持有。先固定对应事件顺序和独立预期，再按负责人另行授权修复及复审；不用扩大预算、正式 LC 清单或提前实现平台进程控制。

第一阶段修复通过之后，真实调用与下一阶段仍需要 E-10(ii)、LE-5 ①／②、N2、固定实现复审和运行 C/T 的分别决定。当前报告不作这些决定，不宣布清单生效、G 项满足、第三批开工或 WP2.5／M2 验收。

本轮独立完成的是源码与调用链静态审查、日志内容／长度／哈希核对、Git 与远程对象／CI checkout 核对。没有独立动态复现本报告反例；反例来自当前代码允许的事件序列。现存本机日志是可读取的实现者证据，常规 CI 是根仓库回归证据，两者均不消除上述未覆盖路径。

结束核验于 2026-11-14 16:20:34 +09:00 完成：本地 HEAD、远程分支与 PR head 仍为 `fcf41b02a341ae35b59bcc43ad80464bd190e54c`，tree 仍为 `9fa336615ccf3050026d140f446a530c3f6f2764`；PR base、远程 master 与本地 origin/master 仍为 `9137279554deed5a542dbd61b871d1cfa79bc323`；工具模块子树不变。PR open、未合并，origin 仍指向指定仓库。没有身份漂移。

工作区只有 `?? docs/reviews/codex-review-148.md`，已跟踪文件与暂存区均无差异。本报告保持未提交。本轮未运行被禁止的工具或实验，未操作 CI、PR 元数据、运行分支或其他仓库文件。报告后暂停。
