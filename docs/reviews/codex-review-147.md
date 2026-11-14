# Review 147：PR #56 的 C-9 限定取证工具第一阶段实现

审查日期：2026-11-14（Asia/Tokyo）。性质：固定对象的独立静态实现审查及既有证据读取；不是正式取证、方法采纳、风险接受或第三批开工授权。

## 1. 结论

**第一阶段实现技术 NO-GO。当前不建议合并，也不建议以“本阶段技术通过”为前提进入真实调用／正式运行的授权阶段。** 发现四项范围内 BLOCKER：同步分析使父端截止出口不可达、manifest 在到期出口丢失持有者、两条路径遗失已经发生的错误、派生与对照记录没有兑现完整来源链。它们不依赖尚未授权的真实调用者或平台协议才能修正。

已交付的解码、保存字节上的 Hash1、条件分析、类型扫描、部分终判与停止入口有可采的局部实现和测试证据；不能据这些局部结论宣布整条工具链完整。LC-9 未实现及停止调用者使当前正式配置不能产生有效取证结论，这一点成立，但不能用这个永久拒绝条件掩盖其他范围内错误。

| 裁定层次 | 本轮结论 |
|---|---|
| 第一阶段实现 | NO-GO，四项 BLOCKER，见 §3 |
| 16 处实现解释 | 逐项见 §4；多数为已有规则或合理细化，另有文字／局部判定问题 |
| 完整工具可执行 | 否；真实调用者、平台控制、LC-9 尚未交付且未获相应运行授权 |
| 正式取证有效 | 未取得；本机夹具测试、交叉 vet 和根模块 CI 均不替代 |
| 下一阶段决定准备 | 可据本报告决定最小修复；不建议直接进入真实调用／取证。修复复审与后续授权分别进行 |
| 第三批生产、例外、G 项、验收 | 本轮不批准，也未由本 PR 自动满足 |

## 2. 固定身份、依据与授权范围

### 2.1 身份

| 项目 | 固定值及核验 |
|---|---|
| 仓库／PR | XKHoshizora/specmetry，[PR #56](https://github.com/XKHoshizora/specmetry/pull/56)，open、未合并 |
| 本地／远程分支 | proto/wp2.5-c9-evidence-tool |
| HEAD | `84c7718257921bb251326f753b23aed01fcac9b9` |
| HEAD tree | `216654c21e7ef89a283471dc6029c98e483a76b1` |
| base／远程 master | `9137279554deed5a542dbd61b871d1cfa79bc323` |
| CI 实际快照 | `d9a2d1573cd1c5ef805dd158c6e92988d1f42e91` |
| 快照父提交 | 依次为上述 base、HEAD；tree 与 HEAD 相同 |
| 开始 | 2026-11-14 13:23:57 +09:00；工作区干净，报告不存在 |
| 结束 | 见 §9；身份不变，唯一新增为本未提交报告 |

本地以 Git 核对，远程以 GitHub 只读 PR／refs／commit API 核对。一次 `git ls-remote` 因 DNS 失败，没有据此推测远程；随后使用连接器取得远程身份。没有 fetch、切换、重置或补齐对象。

### 2.2 读取与变更核验

已读取当前 [AGENTS.md](../../AGENTS.md)、[§190](../review-resolution.md)、[实现记录全文](../wp2.5-c9-evidence-tool.md)、[限定取证计划](../wp2.5-c9-limited-evidence-plan.md)、[G-4 方法规格](../wp2.5-c9-exception-identity-method.md)、[实施计划 §12](../wp2.5-implementation-plan.md) 的直接依赖，以及 [Review 143](codex-review-143.md)、[144](codex-review-144.md)、[145](codex-review-145.md)、[146](codex-review-146.md) 的相关裁定。审查覆盖全部独立模块源码、测试、workflow 和集成调用链，不仅是实现说明。

base→HEAD 为 88 文件、18228 行新增／11 行删除；其中独立模块 79 文件。授权提交 `52da47ea48f66b28409f6c5364e60505ac2bb6c1` 先于实现 `d8255c040de8572099e76a0bfea445bffb8d01bb`，后者先于 HEAD 的实现记录。d8255c0→HEAD 只新增实现记录文件。处理记录旧版 1332763 字节是新版 1343584 字节的逐字节前缀，§190 为追加；状态同步没有改冻结规则、验收项、根模块实现／测试／依赖、原 ci.yml 或既有预算。`git diff --check base HEAD` 无问题。

§190 已针对本 test-only 工具采纳 N1 十项，包括 LE-10 的三个域／四结论和退出映射；不能继续把它在本工具中的采用误写为仍未选择。其余政策边界不变：E-10(ii) 未确认，LE-5 的真实子进程夹具与采集器真实调用均未授权，N2 未决定；路径 1／T-1／L-b1a 不扩成 T-2／L-b1b。x/sys 未导入与本阶段不实现控制协议相容，不要求加入未使用依赖。

## 3. BLOCKER

本节行号均指固定 HEAD `84c7718257921bb251326f753b23aed01fcac9b9`。反例为源码事件推演，本轮没有运行夹具或变异。

### B-147-1：同步执行 F／R，父端无法按 W／D 脱离未返回的分析

**位置**：[run/run_test.go](../../prototype/c9evidence/run/run_test.go) 29–35；[job/job.go](../../prototype/c9evidence/internal/job/job.go) 156–169、208–225、236–246；[collect/b5.go](../../prototype/c9evidence/internal/collect/b5.go) 的 TypeCheck 调用；[typecheck/typecheck.go](../../prototype/c9evidence/internal/typecheck/typecheck.go) 737、858 附近的 parser／types 调用。

`TestMain` 同步等待 `m.Run()`；`TestEvidence` 同步调用 `RunDomains()`，后者同步执行每个 `v.Run(env)` 和 `collect.Collect`。开始单元时检查预算，不能使已进入的 `parser.ParseFile`、`types.Checker.Check` 或夹具操作返回。`Finish` 的 D 出口只在整条同步链返回后才可到达。

**可达事件**：某个已准入的类型检查在 W 前开始，至 D 仍未返回。测试 goroutine 仍在该检查中，`m.Run` 不返回，父端不能进入既有的收尾／D 出口。外层 go test／作业最终超时不是本工具按原 D 返回，也不能当作资源已确认收口。即使操作最终在 W 后、D 前返回，最后一个包没有后继准入点时，还缺少这次分析结果的 W 及时资格判定；不能仅以 A/B 已算完和记录链及时为依据采纳它。

**违反规则**：实施计划 §12.14“同步的类型检查”（1733–1737 行）明确要求工作 goroutine 与测试 goroutine 分开；测试端同时等结果与 W，W 先到即失败返回，工作端保留输入与晚到结果、不调用已结束的 t。限定计划继承该协议。本缺陷发生在本阶段已经实现的纯分析和集成入口，不是要求提前交付平台进程控制。

**最小方向**：给已交付的分析／夹具执行建立受控工作任务及父端 W 等待出口；先登记责任，父端只消费已发布状态，超时后持续保存任务和真实错误，禁止后续单元，晚到结果不得恢复有效性。针对实际 `RunDomains→Finish` 接线核对挂起、W 后完成及健康对照；无需增加预算或要求单次调用硬实时中断。现有链层七种推演不证明这个上游同步入口已兑现协议。

### B-147-2：manifest 在到期出口只有局部句柄，没有持续持有者

**位置**：[chain/writer.go](../../prototype/c9evidence/internal/chain/writer.go) 485–537，特别是 491、503–508、522–525；`ManifestSnapshot` 82–89 只保存结果，不持有文件对象。

**可达事件**：manifest 的 Create 在 D 前准入，返回文件后发布结果；下一次 `admit()` 因到期返回 false，507–508 直接返回。或者 Write 完成后在 522–523 到期返回。`mf` 只在 `writeManifest` 的局部变量中，既没有 Close，也没有转入 writer／持有登记。函数退出后没有计划规定的持续文件持有者；内存中的“到期／未完成”结果不等于持有该句柄。

这条路径会失败关闭，**不是发现了错误 exit 0**，但仍违反限定计划 §6.4 对“已创建文件、尚未关闭对象、到期后责任持续保留”的要求。不能依赖 GC finalizer 或进程退出替代已确认释放／明确持有；也不应为修复它在 D 后补一次同步 Close。

**最小方向**：取得 manifest 句柄后立即纳入唯一持有结构，分开真实 Close 结果、尝试状态与释放知识；正常路径唯一关闭，到期不新启动动作，未确认对象及晚到结果一直有持有者。Create 后到期和 Write 后到期都应沿实际 writer 路径核对。真实 Close 返回错误的释放未知也按同一原则处理，不据错误记录宣称已释放，不要求盲目重复 Close。

### B-147-3：已经发生的真实错误在聚合／到期路径被丢弃

**路径一**：[collect/guard.go](../../prototype/c9evidence/internal/collect/guard.go) 103–116 的 `readClosed`。它先执行 ReadAll 和真实 Close，但 ReadAll 出错时立即只返回 `rerr`，同时发生的 `cerr` 丢失。该 helper 不仅用于守卫 VERSION／可执行文件，也用于根输入读取。输入读取错误与关闭错误可以同时发生；J-1 主分类相同并不允许舍弃关闭结果。

**路径二**：[typecheck/typecheck.go](../../prototype/c9evidence/internal/typecheck/typecheck.go) 660–669、683–714、716–741。已经解析第一个文件并把错误存入局部 `own`／`header` 后，第二个文件的准入失败，`markBudget` 把包的 Load／Header 等写为未评估并直接返回，局部已观察错误没有进入报告。例：a.go 有语法错误或多条 go:build；处理 b.go 前 W 到期。最终留下预算失败，却遗失 a.go 的真实错误；类似地，先取得的模块语言版本错误也会丢失。

**违反规则**：阶段门可以阻止尚未执行的检查，不能把已经执行的错误回写成“未评估”；R／F／O 的全部真实错误持续保存，关闭失败和预算失败不能互相替代。两条反例均无需注入原生系统故障才能从代码确定。

**最小方向**：readClosed 保存并传递两个原始结果（例如结构化结果或完整错误链）；每个提前出口先发布已累积的检查结果，再仅对尚未执行的部分标未评估。主分类优先级可以保持，完整错误集合不能减少。分别核对“读错＋关错”和“已观察解析／约束错误＋下一单元到期”的实际调用者消费。

### B-147-4：派生记录与 F 域记录不满足已承诺的来源和完整消费

**位置**：[collect/stageb.go](../../prototype/c9evidence/internal/collect/stageb.go) 272；[collect/b5.go](../../prototype/c9evidence/internal/collect/b5.go) 220、297、355、400；[job/job.go](../../prototype/c9evidence/internal/job/job.go) 220–225、249–250；[collect/record.go](../../prototype/c9evidence/internal/collect/record.go) 35–43；[controls/lc5.go](../../prototype/c9evidence/internal/controls/lc5.go) 265–277；[controls/controls.go](../../prototype/c9evidence/internal/controls/controls.go) 110–139。

**确定的缺口**：闭包、Context、消费记录、findings、配置检查等派生项调用 `Derived(..., nil, ...)`，`parents` 又是 omitempty。只有比较记录连到 findings，不能补出 findings→消费字节／原调用的来源。健康构造场景可以完整写出并关闭这些文件、生成正确 manifest，但仍没有父记录名称和 SHA-256；缺少父链没有触发终判下降。限定计划 §6.2 第 459 行和实现记录 §2.7 第 111 行却明确要求每条派生记录保存这两项。

F 域还有直接接点：LC-5 的 `res/fake/rec` 最终只转换成 `Outcome.Entered/Lines`；原实例的全部检查、调用记录与 MemRecorder 未交给 Job 持有。`FailureLine` 刻意剔除 Site／Detail，Job 最后保存的也是这些扁平行和预期。这样不能兑现 §6.2 第 458 行的“被测实例全部结果与登记”；以 ID、类别、item、kind、source 相同的行匹配，也不足以证明计划要求的**站点**一致。并非要求把完整源码或敏感正文写进 artifact，而是不能把必要来源／站点从实际消费链删掉。

**影响**：文件完整性哈希不等于证据关联完整；当前健康集成测试的成功可在该缺口存在时成立。R 与 F 形式上分域，并未使每个 F 实例的原始结果持续可追溯。此处不宣称现有每个负向夹具都已出现假命中，但范围内记录承诺和站点判定尚未兑现。

**最小方向**：保留真实调用／输入记录的 Ref，给每项派生输出绑定实际父项；F 保存结构化实例结果、登记与来源，再从其生成展示行。判定保留必要站点、类型、来源、次数和完整失败集合，缺链不能被 manifest 或匹配行抵消。检查记录链的语义关联，而不只验证文件字节数／哈希；不需要新增密码学来源认证政策。

## 4. 实现记录 §3 的 16 项逐项裁定

| # | 裁定 | 理由与边界 |
|---|---|---|
| 1 | 合理实现细化 | 名字含 LF 使 Hash1 名字协议不成立，按来源异常 J-2 且不计算 h1；未把它当正常缺省。 |
| 2 | 符合已有规则 | import C 不算普通非标准库导入，生产变体专有 C 文件已由前一条件拒绝，不形成绕过。 |
| 3 | 需小幅文字澄清，非 BLOCKER | 固定 go/build 会接受已闭合块注释之后**下一行**的 go:build；同一行 `/*...*/ //go:build` 不能笼统说仍接受。代码按该处理顺序实现，未发现规则扩张。 |
| 4 | 符合已有规则 | I-7 保留原约束行、去除行尾 CR/LF，不把内部比较的 TrimSpace 偷换成记录字节。 |
| 5 | 符合已有规则，证据受 §3 限制 | Judged 记录候选，包括被文件名排除者；MatchFile 无须为已按名字排除者再读取头。Context 确实传给分析，不仅写标签。标签敏感集成回归仍有增强空间。 |
| 6 | 聚合优先级可采，细节需保留 | B4 有失败优先于未评估可以成立；不能因此省略逐模块未评估集合。与 LC-1 的实际判定有 N-147-1 接点。 |
| 7 | 类别可采，实际错误消费有缺陷 | 根 go.mod 的读取错误归 A4/J-1 合理；同时 Close 错误被 readClosed 丢弃见 B-147-3。 |
| 8 | 符合已决定的工具语义 | K 成立表示比较完成，consistent 分开；不一致可进入结论 3，不伪造为 J 错误，不自动扩大例外。 |
| 9 | 符合本轮发现边界 | 候选之外的对象发现及 WatchPackages 出现记差异；不把发现直接当违规／批准条目或执行证据。 |
| 10 | 有固定源码依据的实现细化 | cmd/go 对工具链与模块版本使用完整版本比较；类型检查使用的语言版本再按语言规则处理。不是把 go1.27.4 的常规 CI 当作工具版本。 |
| 11 | 前提传播规则可采，提前出口需修 | 未成立的包及传递导入者不交类型检查、记未评估合理；不能丢弃此前已取得的本包错误，见 B-147-3。 |
| 12 | 合理失败关闭 | 冻结对象出现在该表项未列平台时保留并报扫描异常 J-3，不静默扔掉，也不新增平台许可。 |
| 13 | 合理表示细化 | 对泛型 receiver 取原始类型／FullName 形式用于调用方标识；不推导通用泛型 absent 证明。 |
| 14 | 符合严格解码要求 | 第 3 调用空／全空白不能解释成健康的空闭包，J-3 与非零调用的 J-1 分开。 |
| 15 | 符合 Review 146 的资格规则 | 末次写入后终态可以是完成但 Late，不能因此保留 3／4；仍需下一次写入时走准入失败。链层源码实现该区别。 |
| 16 | 解释意图可采，实际实现未完全兑现 | “所有闭包模块都计算 h1”不能只枚举已成功放入 Snapshots 的项；见 N-147-1。 |

固定官方依据为 Go 1.27.3 commit `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`：只读核对 [src/go/build/build.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/go/build/build.go) 的 parseFileHeader（约 1640–1691 行）、[src/cmd/go/internal/modload/init.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/modload/init.go) 的版本比较（约 1931 行），以及 [go/types/check.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/go/types/check.go) 的同步检查入口。本轮没有运行这些源码或 Go 命令；此前方法规格已审部分仅按原限定沿用。

## 5. 集成路径、对照与非阻塞意见

### 5.1 可采的局部实现

实际入口从 FormalConfig→Job→collect 的 A/B→modsnap→cond→typecheck→scan→K→记录／verdict，已按调用者而非仅按包审查。R 明确为外部传入的根模块路径，独立工具模块和内存夹具不替代 R；FormalConfig 不从工作目录猜 R。三种调用经过 Invoker，正式配置只选 Stopped，未发现启动真实采集进程或自动回退。缺少 R、错误 T2、m.Run 非零、域未运行均有拒绝路径；配置不全退出 2 不是有效结论。

严格解码有重复键、字段类型、缺失语义字段、空流、尾部内容等拒绝；特殊包和主模块／依赖模块分支保持。I-4 的快照主体按文件读取、保存真实 Close 结果，Hash1 只消费已保存字节；I-5 的受控读取和 cond 的 Context 接线可静态确认，但不扩大成原子目录快照、认证官方来源或真实 cmd/go 输入证明。B-147-3 的公共读取 helper 是这条正面结论的明确例外。

闭包外候选不被按包名迁移；目标文件及约束行、对象引用与 X1/X4／运行时信号参数诊断分开。冻结对象发现不是新增规则的采纳。K 的不一致与检查失败不同，四类结论可在结构上区分；结论 3 的 exit 0 也不能被后续解释为候选一致或自动放行。

LC-5 八个投影的固定预期、进入检查和调用次数在源码中另行列出；未补实现 B′，未把不覆盖集合 N 写成实际执行。对照预期有手写固定项，不是单纯把运行结果复制为 Want。但完整实例／站点保存尚有 B-147-4，不能因此认定全部故障消费已经可靠。准备失败为 Miss、未终结为 Pending、清理失败不使命中有效；这些外层规则本身可采。

LC-9 在正式 All 中持续为适用但未执行；健康集成测试删减 LC-9 的函数位于测试文件，FormalConfig 没有该分支。Windows 符号链接准备失败在正式对照中是 Miss，不是平台不适用；单元测试自身 SKIP 与正式命中分开。LC-6 使用假调用者，不提供真实 go list 参数传递证据。

链层的状态锁不跨 writer／diagnostic 外部 I/O；逻辑判定点、到期锁存、及时资格不被父端晚消费撤销。末端先消费已发布终态，暂定 1／2 不因成功改善；正常 3／4 依赖 writer 和 diagnostic 的成功及时资格。源码和已有链测试覆盖七种推演／五状态的局部规则。**这些肯定不覆盖 B-147-1 的上游同步等待，也不消除 B-147-2 的 manifest 所有权缺口。**

### 5.2 非阻塞意见

- **N-147-1（LC-1 的正对照集合）**：job.go 272–283 接受 B4=Fail 后只遍历 `r.Snapshots`。若模块甲已计算 h1 但不匹配，模块乙因 I-2／I-3 失败未进入该 map，则 B4=Fail，现有 map 内项均有 H1，LC-1 被记 Hit，尽管并非所有闭包模块计算完成。R 的身份／哈希失败仍会阻止有效取证，所以不单列为本轮 BLOCKER。最小修正是从应检查的闭包模块集合核对“全部计算完成”，并保留每项未评估，不能只遍历成功产物。
- **N-147-2（标签敏感集成覆盖）**：目前可静态确认 Context 与 MatchFile 接线，并有 cond 单元反例；LC-6／Job 健康夹具尚不足以单独证明目标 ToolTags／ReleaseTags 改变确实进入实际消费者。适合在固定实现复审补一个有独立预期的标签敏感接线对照；不是要求本轮扩大正式 LC 清单或新增真实调用，未因“还可加测试”列为阻塞。
- **N-147-3（预算记录）**：job.go 134 将 `W` 写成 `cfg.Start.Sub(cfg.Start).String()`，恒为 `0s`。内部预算另行计算，不是实际 W 被设为零，但记录字段失真；应记录既有 W 的明确表示／相对偏移，不改预算。
- §3 第 3 项块注释后的措辞按 §4 收窄。未执行 -race 是证据限制，不是独立 BLOCKER；本报告只对具体可达路径提出阻塞。

## 6. workflow、执行边界与实现行为记录

[c9-evidence.yml](../../.github/workflows/c9-evidence.yml) 仅 `push` 到 `evidence/c9-run-1`，没有 PR／手动／定时触发；三个 runner、20 分钟外层兜底、Go 1.27.3、cache false、contents read、不中断进行中的运行及 artifact 设置都作为待审 workflow 内容存在，不代表 N2 已批准。读取远程 matching-refs 未发现运行分支；HEAD 的运行列表只有常规 CI，未发现新取证 workflow 运行。

根 ci.yml 未变，嵌套 go.mod 使根 `./...` 不纳入该模块；实际常规日志也没有 c9evidence 包。正式入口受 `c9evidence_run` 标签限制。workflow 的 checkout commit=push GITHUB_SHA 与打印 tree 是不同关系，不把 PR 合并快照冒充未来取证 C；授权 C/T 与完整 artifact 的复核仍是运行前后另行责任。

读取实现者本机会话的有限相关回执（不是重放执行），确认曾有三个子任务负责严格解码、记录收尾、装载扫描；其结果都在本轮受审源码内。没有因为使用子任务就认定越权，也没有豁免子任务代码。

相关 Python 回执并非全都“没有输出”：2026-11-14 03:55:40Z 有位置计算输出，03:56:35Z 有修改确认及 controls 测试结果，03:57:01Z 有修改确认，03:57:56Z 有 Python AssertionError 及当时 Job 测试失败，04:03:51Z 有文件／文档链接检查统计。它们属于实现过程的编辑／检查记录；不能以一次 shell 回执的外层成功标志抹去内部失败。本轮没有发现这些记录证明采集器启动了三种真实调用，也不能仅由 Python／子任务这一形式推断发生了未经授权的机制实验。此核对只覆盖取得的记录，不宣称审计了所有子任务的完整会话。

工具链下载在 §190 有单独决定。现有 go1.27.3 windows/amd64 的 go.exe 只读哈希为 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`，与登记一致；这不是对每次历史进程实际装载字节的独立证明。记录中确实有本地 Go 构建、测试、vet、mod verify 和导入闭包查询，因此“没有运行任何 Go 进程”不成立；准确说法是**未发现采集器启动三种真实调用**。本轮审查者没有运行上述命令。

## 7. 既有本机证据与独立核验的界限

取得并读取实现者原始测试日志：`...\scratchpad\s190\test-v.log`，38611 字节，SHA-256 `28d633707ab075502a05e2b3d70d5dfea567e7bb1c583cd16684cbbc26e831a6`。完整本机位置为 `C:\Users\hoshi\AppData\Local\Temp\claude\b96df3361c677053\67d963ef-6de7-4877-aebf-656e909830be\scratchpad\s190\test-v.log`（仓库外证据，不复制入仓库）。会话回执来源为 `C:\Users\hoshi\.claude\projects\C--Users-hoshi-Desktop-specmetry\67d963ef-6de7-4877-aebf-656e909830be.jsonl`。

独立读取统计：128 个顶层 PASS、268 个子测试 PASS、0 FAIL、1 SKIP；11 个有测试的包 ok，run 包在无标签时没有测试。唯一 SKIP 是 `TestTakeNegatives/symbolic_link`，权限不足。日志另可见 LC-2 符号链接准备失败按 Miss 保存，不冒称命中。

最终测试调用回执在 2026-11-14 04:01:43Z 返回成功，使用本机固定 go1.27.3；之后形成 d8255c0，HEAD 只加记录。可用时间顺序、提交差异与记录关联，但原日志本身没有嵌入被测 tree 或二进制哈希，**不能写成不可变产物级绑定，也不是审查者独立复现**。前面的失败测试及修正事实保留，不说首次全部通过。

另读取到 native vet、带正式标签 vet、三平台两架构交叉 vet、正式入口仅编译、mod verify 的既有回执。它们不是 Linux／macOS 运行测试；未取得 -race 运行证据。各包耗时只证明该次本机观测，不能证明两配置、真实采集和所有清理能在 120 秒内完成。真实输入兼容性、T2 单参数到达 cmd/go、平台控制、LC-9、runner 符号链接权限及 artifact 完整性均未由这些记录证明。

本轮证据分层：源码反例与接线为审查者静态确认；原始日志统计及工具回执为独立读取的既有证据；无产物绑定的历史执行细节、未取得的运行环境和所有未实施部分保留为限制，不用作者说明补成验证结果。

## 8. 常规 CI 的独立核验

[run 43554853341](https://github.com/XKHoshizora/specmetry/actions/runs/43554853341)：pull_request、attempt 1、completed/success，8/8。读取正式元数据和八份完整 job 日志，每份实际 checkout／git log 均为 `d9a2d1573cd1c5ef805dd158c6e92988d1f42e91`；另核对该对象父提交依次为固定 base、HEAD，tree 等于固定 HEAD tree。不是只使用 job head_sha。

| 作业 | job ID | 实际证据 |
|---|---|---|
| Go tests Windows | 132325283561 | go1.27.4 windows/amd64；27 个 ok 包、7 个无测试包；10 SKIP；CLI 390.144s |
| Go lint | 132325283784 | go1.27.4 linux/amd64；0 issues |
| Web | 132325283794 | success；没有安装／报告 Go，不补写版本 |
| Go tests Ubuntu | 132325283829 | go1.27.4 linux/amd64；27 个 ok 包、7 个无测试包；15 SKIP；CLI 435.655s；单独 root CrossUser 8 项 PASS |
| Go tests macOS | 132325283956 | go1.27.4 darwin/arm64；27 个 ok 包、7 个无测试包；7 SKIP；CLI 148.637s |
| Dashboard Windows | 132325550174 | go1.27.4 windows/amd64；构建与 embedded smoke 成功 |
| Dashboard Ubuntu | 132325550269 | go1.27.4 linux/amd64；构建与 embedded smoke 成功 |
| Dashboard macOS | 132325550375 | go1.27.4 darwin/arm64；构建与 embedded smoke 成功 |

主测试三个作业均无 FAIL／panic；八份日志没有 GitHub error 指令。Ubuntu 常规测试带 -race；root CrossUser 是独立步骤，不能把普通用户测试中的跳过写成运行通过。Windows 的 CLI 单独 20 分钟步骤与其他包分开。

三个平台共有的 7 个跳过：`TestClaimWithARealGitThatFetches`、`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches`、`TestM14ReadOnlyProbe`、`TestM14OpenPath`。Windows 另跳过 `TestResolveCommitsWithUnreadableStorage`、`TestClassifyWithUnreadableStorage`、`TestSourceListUnreadableSource`。Ubuntu 另有普通用户下 8 个 CrossUser 跳过，随后 root 步骤实际执行通过：PathSwapCannotRedirectWrites、CannotReplaceDataDirectoryUnderSafeParent、StickyParent、AncestorOwnedByAnotherUser、ScratchUnderAWritableParent、ScratchUnderASafeParent、ScratchAncestorOwnedByAnotherUser、ScratchStickyParent。

27 个 ok 是主测试包集合口径，不把额外的 root 复跑或 dashboard smoke 再加成新包。七个显示 Go 的作业都是 1.27.4，与方法固定的 1.27.3 分开。没有日志显示本次常规 CI 运行 prototype/c9evidence；因此它不反驳 §3 的源码问题，也不验证工具预算。CLI 耗时不是上界或根因证据。

前序 PR #55 的超时、单次重跑及 §190 所登记的 Windows 失败／单次重跑、master 成功按已有记录范围保留；本轮没有重新全量核验所有历史日志，不据当前绿色关闭 M-16 或其他维护事项。

## 9. 后续最小范围与结束记录

建议先由负责人决定是否授权修复 §3 的四项问题，并明确把修改限定在现有第一阶段的集成截止、文件持有、错误保存与记录链；修复及其固定回归再独立复审。N-147-1／3 宜一并作局部修正，N-147-2 可在固定实现复审决定适当的接线覆盖。此建议不是自动修复或扩大正式对照的授权。

后续真实调用／平台阶段仍需另行确认 E-10(ii) 或先审查替代方案，补齐唯一回收／pidfd／输出所有权／LC-9，审查固定实现，决定 N2 与运行 C/T，再决定运行分支首次推送。真实输入兼容性、三平台执行、预算、完整 artifact 与限定取证结论分别取证。未完成这些事项本身不是本轮四项 BLOCKER 的理由，也不能被本机测试或文档 GO 替代。

结束核验：2026-11-14 13:43:42 +09:00，再次读取本地 Git、远程 refs、PR 与 HEAD commit 对象；HEAD/tree、分支、master、PR head/base 均与 §2.1 和开始一致，PR open、未合并。本地已跟踪文件及暂存区无差异，唯一状态为 `?? docs/reviews/codex-review-147.md`。未发生身份漂移或无关修改。

本轮未运行 Go、测试、构建、采集器或实验；没有触发／重跑 CI、创建运行分支、修改 PR、提交、推送或合并。报告不接受 T-2／L-b1b、原始入口或发送信号风险，不批准任何条目／新 L／H-1，不宣布 G 项满足或 WP2.5／M2 验收。

