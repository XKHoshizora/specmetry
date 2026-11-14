# Review 150：PR #56 逐步骤准入修复的独立定向复审

## 1. 裁定与范围

**固定对象在受审第一阶段范围内技术 GO。B-147-1-R-c/P 关闭，R-c 与 B-147-1 在该范围内整体关闭。未发现本轮新增 BLOCKER 或实现的直接回退。技术上建议负责人将该固定对象作为第一阶段成果考虑合并；本报告不产生合并授权。**

本轮有一项非阻塞意见 **N-150-1**：实现记录 §10.2 把新增回归统称为经过 `RunDomains → Finish`，超出了实际覆盖。部分测试也没有结束其内存记录写者。具体范围、影响和最小处理建议见 §5；这不改变本轮逐步骤准入的关闭裁定。

| 审查项 | 本轮结论 | 限定 |
|---|---|---|
| B-147-1-R-c/P | 关闭 | 每个受审的外部准备／修改调用分别以 W 准入，拒绝后不启动后续调用 |
| R-c | 整体关闭 | 结合 Review 149 已成立的外层登记、D 清理准入、持续持有与实际消费者 |
| B-147-1 | 整体关闭 | 受审第一阶段；R-a、R-b 沿用 Review 149，未发现本轮直接回退 |
| B-147-4、B-147-2、B-147-3、N-147-1～3 | 保持原限定结论 | 未重新全面审查未改动的方法与协议；检查了本轮接点 |
| `TestWriterExpiresMidway` 修正 | 保持 Review 149 结论 | 本轮未修改其代码或生产写者判定 |
| 第一阶段技术结论 | GO | 不等于完整工具可执行或正式取证有效 |
| 下一阶段 | 可交负责人作单独决定 | E-10(ii)、LE-5 ①／②、N2、真实调用和第三批生产实施边界不变 |

依据为 [Review 149 全文](codex-review-149.md)、[Review 148](codex-review-148.md) 与 [Review 147](codex-review-147.md) 的相关限定、[处理记录 §194](../review-resolution.md)、[实现记录 §10](../wp2.5-c9-evidence-tool.md)，以及[限定取证计划](../wp2.5-c9-limited-evidence-plan.md)的准入、三个证据域、持续失败与终判规则。上轮未关闭项是外层操作内部的后续文件调用绕过准入，不是要求改写标准库内部、可中断已准入系统调用或提供硬实时原子性。

## 2. 固定身份、登记与改动范围

### 2.1 开始身份

开始核验时刻为 2026-11-14 19:58:24 +09:00。本地分支、origin、远程分支及 GitHub PR #56 的 head/base 相符，工作区干净，目标报告不存在。

| 对象 | 实际值 |
|---|---|
| 分支 | `proto/wp2.5-c9-evidence-tool` |
| HEAD | `88c8636f18e7371c263a5bc9ad3a4d63a59e9b48` |
| HEAD tree | `cfdd92a87bbeb4782c0a7695e1a239990bab309e` |
| `prototype/c9evidence` subtree | `95d92292b7807d211eab6fdbe6f7cdd6c5ab32f5` |
| base／远程 master | `9137279554deed5a542dbd61b871d1cfa79bc323` |
| 上轮对象 | `23227b689b052f26cdd1205e3e1f46d767ec6055` |
| origin／PR | `XKHoshizora/specmetry`／[PR #56](https://github.com/XKHoshizora/specmetry/pull/56)，open、未合并 |

### 2.2 授权与归档

核对了提交顺序：`9d365d1` 单独归档 Review 149 → `be27020` 登记 §194 → `3dc38a9` 实现与测试 → `88c8636` 实现记录与状态同步。登记提交先于修复；§194 不含真实调用或正式运行授权。

归档 [Review 149](codex-review-149.md) 为 **26343 字节**，SHA-256 为 `62e4253578d1a248868d40955f60d7475726ec346fc165db663a53487da4d114`；工作树文件与归档提交 blob 一致。处理记录旧文件的 1369172 字节保持为当前 1379195 字节文件的逐字节前缀，§194 为追加；没有回改 §192／§193 或旧决定。

`23227b6 → 88c8636` 共 15 个文件：三份工具源码、三份工具测试、归档报告、实现记录、处理记录及状态同步。实际源码变化为 `controls.go`、`lc23.go` 与 `job.go` 的测试接缝接线；没有修改根生产模块、冻结正文、验收、依赖、workflow 或预算。`3dc38a9777f65cc9fd4d53cc42cfa6a86f566c82` 与最终 HEAD 的工具子树完全相同；其后是文档变化。该区间 `git diff --check` 没有输出。

实现记录与状态仍写“已修复，待 Review 150 独立复审”，没有提前宣布残余关闭。实施计划、方法稿和其他文档的本轮变化是状态同步，不产生新的方法或风险决定。

## 3. B-147-1-R-c/P：实际调用路径

以下源码位置均指固定提交 `88c8636f18e7371c263a5bc9ad3a4d63a59e9b48`；行号为该提交的行号。

### 3.1 新动作逐一准入

[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 第 314～323 行的 `step` 先调用 `Outer.Prepare(op)`。拒绝时返回带具体步骤名的 `ErrNotAdmitted`，不执行回调，也不把拒绝回写成完成。准入后才执行一个外部文件调用，并将其真实返回结果交给该登记项的 `done(err)`。

| 入口 | 实际接线 | 审查结果 |
|---|---|---|
| `prepare`，第 333～350 行 | `MkdirTemp` 自成一步；取得目录后立即 `Hold`；再进入受控 `Env.writeFiles` | 返回晚到也保留实际目录及结果；不能凭第一次准入放行后续文件调用 |
| `Env.writeFiles`，第 372～387 行 | 文件名排序；每个文件分别 `MkdirAll`、`WriteFile`，每一步重新调用 `step` | 跨调用、跨文件均无一次准入覆盖多个外部调用的残余 |
| `changeStep`／`change`，第 354～368 行 | 每个变更回调各为一步；首个错误／拒绝立即结束遍历 | 后续步骤不执行、不伪造完成 |
| LC-2，`lc23.go` 第 82～97 行 | 修改既有文件、增加文件、创建符号链接分别经 Env 的单调用接缝 | 未发现另一路直接绕过准备准入的修改 |
| LC-3，`lc23.go` 第 166～174 行 | 重读变体的写入为一步；文件改目录为 `Remove`、`Mkdir` 两步 | `Remove` 跨 W 后不再执行 `Mkdir` |
| 共用准备 | LC-2／3、`prepareRF` 的消费者及 LC-6 均经 `prepare` | 不以另一个夹具入口恢复一次性放行 |

Env 的新增接缝默认仍调用原有 `os.MkdirAll`、`os.WriteFile`、`os.Remove`、`os.Mkdir`、`os.Symlink`，权限参数等真实行为保持；没有默认成功或自动回退。`job.go` 第 208～218、455～458 行只把受控测试接缝传入 Env。正式作业的默认值仍走真实文件操作。

同名的裸 `writeFiles`（`controls.go` 第 393～409 行）仍不带作业准入，但实际调用来自 `FixtureRepo`，其调用者是测试准备代码；没有进入作业的 `prepare`／`change` 路径。本轮不能因此宣称所有仓库文件 I/O 都有 W 守卫；成立的是受审作业的外层准备与修改路径。

### 3.2 三个判定点没有混淆

| 判定点 | 依据 | 本轮影响 |
|---|---|---|
| 准备／修改准入 | `outerAdapter.Prepare`（`job.go` 第 374 行起）把 W 传给 `Outer.AdmitOp`；`outer.go` 第 165～181 行要求严格小于截止 | 每个新步骤单独判断；等于 W 拒绝 |
| 已准入外层操作的完成资格 | `Outer.Done/finish`（`outer.go` 第 204～219 行）按既有 D 与锁存保存真实结果、完成时刻、资格 | W 后、D 前完成可以是外层及时结果；不能据此推导工作及时 |
| 必要工作整体完成资格 | `job.go` 第 326～345 行在 F、R、LC-1、来源核验等必要工作完成后发布，仍要求 W 前且未放弃等待 | 晚到补齐不能恢复有效结论；父端晚醒也不抹掉已经及时成立的资格 |

清理经 `outerAdapter.Cleanup`（第 389～404 行）用原 D 准入，拒绝不会执行 RemoveAll；只有真实返回 nil 才记录对应目录释放。`setHeld`（第 414～423 行）没有把拒绝、失败或仅“尝试过”写成 Removed。根目录的创建和 defer 清理仍走第 425～455 行的同一套 W／D 接线。

W／D、120 秒总预算及原锁存机制均未变。检查与下一条外部调用之间不被宣称为硬实时原子操作；已经合法准入的系统调用可以晚返回，父端截止出口不以强行中断它为前提。

### 3.3 事件推演与结果消费

| 事件顺序 | 结果与下一步 | 终判／责任 |
|---|---|---|
| `MkdirTemp` W 前准入，D 后返回目录和 nil | 目录先进入持有集合，原操作真实成功、晚到资格保存；下一 `MkdirAll` 被拒绝；D 后根清理也被拒绝 | 不产生后续准备或清理调用；必要工作不完整／晚到，不能成为 3／4；目录责任保留 |
| `MkdirAll` 返回时已过 W | 当前真实结果保存；`WriteFile` 的新准入拒绝 | 部分目录仍有持有者；F 准备未完成，不冒充命中 |
| `WriteFile` 返回时已过 W | 当前文件真实结果保存；下一文件的 `MkdirAll` 不执行 | 跨文件停止，不把后续项伪记为成功 |
| LC-3 `Remove` 返回时已过 W | `Mkdir` 新准入拒绝；如仍小于 D，变体及根清理可正常准入 | 可以确认实际完成的清理，但不能改善既有拒绝／未完成结论 |
| 新准备恰好 W | `OpRefused`，无实际文件调用 | W 到 D 的清理仍可准入；恰好 D 或更晚的新清理拒绝 |
| 一个步骤返回真实错误 | 保存 `OpFailed` 及原错误，立即停止；不继续执行并制造新的拒绝来覆盖它 | F 的 `PrepErr`、真实外层错误均保留；清理的独立结果继续消费 |

实际消费不只依赖步骤函数返回非零：[controls.Judge](../../prototype/c9evidence/internal/controls/controls.go) 第 200～247 行把准备拒绝／真实失败、清理失败／未知分别保留；`publishVariant`（`job.go` 第 479 行起）保留结构化 F 结果；`copyState` 和 `Finish`（第 660～830 行）消费外层 `OpFailed/OpUnknown/OpRefused`、pending 与未确认移除的持有项。实际 `run/run_test.go` 的 TestMain 仍在 `m.Run` 后调用 `Finish` 并使用最终退出码。

准备中途失败时 `prepare` 可以返回 nil fixture；这不删除已经登记的目录。此时由根清理负责其文件系统子树，不能因局部没有可返回的 fixture 而丢失所有权。持有记录按确切目录更新，根 RemoveAll 成功不自动把每个子目录记录改写成“单独确认释放”。因此可能仍有保守的子目录未确认记录；这是既有责任／知识边界，不是证明该目录仍在磁盘，更不是本轮增加的假释放。失败运行保持无效。

## 4. 回归的实际范围与证据强度

### 4.1 新回归及旧测试调整

已逐项读取 [regress149_test.go](../../prototype/c9evidence/internal/job/regress149_test.go) 与 [regress149b_test.go](../../prototype/c9evidence/internal/job/regress149b_test.go)，没有运行它们。

| 回归 | 独立核对的关键断言 |
|---|---|
| `TestB149MkdirTempReturnsAfterD`，第 230 行起 | 实际 RunDomains；目录调用仍阻塞时经 Finish 取得无效出口；释放后实际目录、nil 返回、晚到 O 结果保留；下一 mkdir/write 和清理没有执行；持有与 F 未命中 |
| `TestB149MkdirAllCrossesW`，第 285 行起 | 一次 mkdirAll、零 write；拒绝记录和步骤名；已完成真实结果与部分目录持有 |
| `TestB149WriteCrossesW`，第 308 行起 | 第一次 mkdirAll/write 各一次，下一文件未进入；成功读数与后续拒绝分开 |
| `TestB149RemoveCrossesW`，第 325 行起 | Remove 一次、Mkdir 零次；准备拒绝；W 后 D 前两次真实 RemoveAll 与释放记录；清理成功不消除 F miss |
| `TestB149StepAtWIsRefusedCleanupAdmitted`，第 355 行起 | 等于 W 的准备拒绝及该时刻清理可以准入；没有扩大清理期限 |
| `TestB149RealFailureKeptApartFromRefusal`，第 375 行起 | W 前和 W 后返回的真实错误均可取得；不同于“真实成功后下一步拒绝”；停止点、操作数量和 F 原因分开 |
| `TestB149HealthyStepsAllDone`，第 428 行起 | 手写计数：根 1、夹具 4、mkdirAll 8、write 9、remove 1、mkdir 1、removeAll 5；真实结果均成功、持有项释放、F 命中；Finish 得到结论 4／exit 0 |
| 四项名称／顺序回归，regress149b 第 36／67／96／110 行起 | 固定写定的步骤名和顺序；拒绝步骤准确定位；Remove/Mkdir 分开；LC-2 两种写入修改各有独立步骤 |

健康计数和步骤清单没有从被测登记结果反算。文件接缝保留真实文件操作并可在其生效后阻塞；注入错误是接缝证据，不能称为自然发生的系统故障，也不覆盖未授权的真实工具调用。

受控阻塞由 Cleanup 释放，等待工作结束有 10 秒边界。测试用假时钟推进 W／D，不声称真实系统调用有该时长上界。实际外层结果、F 状态、责任记录是断言对象，并非仅检查 exit 非零。不过不是每项都进一步走到 Finish，见 §5。

[controls/regress148_test.go](../../prototype/c9evidence/internal/controls/regress148_test.go) 第 45 行附近把准入额度从 1 改为 5，与 LC-3 首个变体的两文件准备相符：1 次 MkdirTemp + 2 组 MkdirAll/WriteFile。额度只使原准备完成；随后的 Cleanup 仍拒绝，原 Pending、cleanup 原因和持有目录断言保留。它没有放宽为任意结论，也没有把“准备被拒绝”误当作原本要测的“清理被拒绝”。

### 4.2 旧实现与修复后结果不得混写

1. 第一份旧逻辑日志包含一项错误的测试资格预期：把 W 后 D 前返回的外层结果也要求为 Late。这个断言错误不是原缺陷证据；本轮仍保留该首次失败。
2. 第二次旧逻辑运行使用 `be27020` 加行为中立的文件接缝和测试补丁，不是无修改旧提交的运行。已读保存的 patch：它仍保留旧的一次性准入范围，接缝使各次文件调用可计数／阻塞。日志中后续 mkdir/write、Remove 后的 Mkdir 确实继续发生，支持旧残余反例。
3. 两份旧运行都是 6 个顶层失败、1 个通过；真实错误保留子路径和健康对照原已通过，不说它们因修复才由红变绿。`TestB149RealFailureKeptApartFromRefusal` 的旧失败来自其中的部分成功后拒绝路径。
4. 四项名称／顺序回归只有修复后证据，没有原缺陷的动态红绿对照。
5. 20 次重复、两条既有写者出口都出现，不是 `-race` 或全部调度覆盖，也不能替代本报告的源码事件推演。

## 5. N-150-1：回归终判覆盖表述及内存写者收尾（非阻塞）

**位置：**[实现记录](../wp2.5-c9-evidence-tool.md) §10.2 第 552 行；`internal/job/regress149_test.go` 第 204、230、428 行起；`regress149b_test.go` 四项测试；`regress148_test.go` 第 280～293 行；`internal/chain/writer.go` 第 401～430 行。

§10.2 称这些新增回归“都经过 RunDomains → Finish”。实际 11 个新增 B149 顶层测试中，只有 `TestB149MkdirTempReturnsAfterD` 和 `TestB149HealthyStepsAllDone` 调用了 `finish`。其他测试经过实际 RunDomains、工作完成和 O／F／Held 快照，但没有调用 Finish。`settle` 只释放受控阻塞并有界等待 WorkDone，不封口日志、不等待记录写者。

因此有两层限制：

- 这些测试动态验证的是步骤调用与中间消费状态，不能统称分别验证了最终封口、manifest、诊断终态和退出码。
- Begin 启动的内存记录写者在未封口且无后续变化时，可以仍停在 `writeUntilSeal` 的 `<-changed`。测试结束和真实夹具目录清理并不表示该内存任务也结束。这里是 memFS／进程内任务，不是已确认遗留真实文件句柄或子进程。

**为何不列本轮 BLOCKER：**新修复所改变的外部调用准入、返回结果和拒绝均有实际调用计数及状态断言；D 后目录返回的失败终判和健康终判另有实际 Finish 覆盖；此前到期／拒绝的 Finish 消费回归仍在，源码上本轮没有改变该消费规则。未发现这些少做的测试终判步骤能让正式 Job 越过 W 启动后续文件调用或得到虚假的有效结论。保留内存写者是测试生命周期不足，不等于本轮生产责任链丢失。

**最小建议：**下次获授权的收尾准确区分“两项完整 Finish 回归”与“RunDomains／外层状态回归”；若补测试，沿既有有界 helper 对后一类增加对应最终断言并结束记录写者，不重取预算、不提前释放被测阻塞、不放宽旧断言。本轮只报告，不修改文档或测试；无需为此重开已关闭的真实调用／平台协议设计。

## 6. 既有本机证据核验

### 6.1 原始日志与身份

读取了指定 `scratchpad/s194/logs/` 的七份原始日志，并独立核对字节数及 SHA-256；都与实现记录 §10 相符。这里是原始记录读取与静态核验，**不是审查者重新执行测试**。

| 日志 | 字节数 | SHA-256 |
|---|---:|---|
| pre-fix-regressions.log | 5446 | `d9f69a26bde21b1a32c817d187cf6abe11ff2c30f899bcc2f81c029aafd5c2a0` |
| pre-fix-regressions-v2.log | 5288 | `1611baacd47422b1578e8d9d6c7d013a10ddfc0944ffa5eda5b73f9e59b6b9b9` |
| targeted-test.log | 2197 | `9276ee8c9457d7d48370fdbb4ee90ca5e8cc25597db32f2550b023596983d25b` |
| affected-packages.log | 8016 | `7b2ac06ddabe499441a0195dd1583236c969aeb6f4b4629c13034c994bdc54ae` |
| full-test.log | 46998 | `58aba054334e5f0b2c65784122794f70edf080ad410bdff97b645a0f3debd1c8` |
| regress147-149-x20.log | 122279 | `a389964f3038b76d890b77507e6b365fe1c354c2c40c8404199346ff4b334232` |
| static.log | 2699 | `e6a4538319de64555c3930703be82b2f33a90e8541dc1a173d9a77d3a33c7e95` |

保存的旧逻辑接缝 patch 为 9247 字节，SHA-256 `3117974145e9d9b44a2ed277829d293648fccecc6f80690b1f35c875eed4f5a9`；旧回归第二版与最终对应测试文件的哈希一致。这支持记录中声明的修改范围，不把 patch 加载后的代码说成未经修改的旧提交。

修复后日志记录 HEAD `3dc38a9`、测试前后工作区、命令、退出状态和工具子树；该子树与固定最终 HEAD 相同。工具链记录为本机 Windows/amd64 的 go1.27.3，环境 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`。这些是时点关联，未建立不可变测试二进制与日志的密码学绑定，不能扩张为审查者独立复现或三平台运行。

### 6.2 可采的结果

| 记录 | 实际可采范围 |
|---|---|
| 定向测试 | 12 个顶层 PASS：11 个新增 B149 + 受影响旧拒绝测试；0 FAIL |
| 受影响包 | 67 个顶层、3 个子测试 PASS，2 个 ok 包；0 FAIL |
| 全量工具测试 | 198 个顶层、271 个子测试 PASS；1 个 SKIP；11 个 ok 包；0 FAIL |
| 重复回归 | 61 个独立顶层测试各 20 次，共 1220 个顶层 PASS；既有 writer 出口 a 出现 1 次、b 出现 19 次；不是竞态检测 |
| 静态记录 | gofmt 无待格式化输出；vet、带正式标签 vet、六配置交叉 vet、正式入口 `test -c`、模块校验等记录为 exit 0；正式入口只编译，不执行 |

唯一工具测试 SKIP 是 `TestTakeNegatives/symbolic_link` 的本机权限限制。正式 LC-2 的符号链接准备失败仍是未命中，不据单元测试 SKIP 判定正式对照成立。预期的类型检查诊断不能误计为顶层测试崩溃。导入闭包记录的第三方依赖为 x/mod 的 dirhash；本阶段没有为了形式添加未用的 x/sys。

历史 s192 首次全量失败、s193 的写者测试修正、旧 `fcf41b0` 只复现出口断言失败及空指针在 `ef1d9ad` 出现，继续按 [Review 149](codex-review-149.md) 的限定保留。本轮没有重新取得那些历史动态证据，也不把新成功抵消旧失败。500 次和本轮 20 次都不代表全部调度或 `-race`。

## 7. 常规 CI：单独核验，不作为工具模块验证

独立读取 [run 43584805200](https://github.com/XKHoshizora/specmetry/actions/runs/43584805200) 的正式元数据、八份作业日志和合并提交对象：`pull_request`、attempt 1、completed、success，8/8。八份日志均实际检出：

`c28889fc41d53523fba124a191d92159cd6afb3d`

父提交依次为 `9137279554deed5a542dbd61b871d1cfa79bc323`、`88c8636f18e7371c263a5bc9ad3a4d63a59e9b48`，tree 为 `cfdd92a87bbeb4782c0a7695e1a239990bab309e`，与受审 HEAD 相同。本核验不只采用 run 的 head_sha。

| 作业 ID | 作业 | 实际工具链与结果 |
|---|---|---|
| 132416891964 | Web | success；不安装 Go，不补写 Go 版本 |
| 132416892119 | Windows Go tests | go1.27.4 windows/amd64；27 个 ok 包，10 个 SKIP，0 FAIL |
| 132416892186 | Ubuntu Go tests | go1.27.4 linux/amd64；27 个 ok 包，15 个 SKIP，0 FAIL；race 与独立 root CrossUser 步骤分开 |
| 132416892204 | Go lint | go1.27.4 linux/amd64；0 issues，success |
| 132416892212 | macOS Go tests | go1.27.4 darwin/arm64；27 个 ok 包，7 个 SKIP，0 FAIL |
| 132417161027 | macOS dashboard build | go1.27.4 darwin/arm64；web 测试、构建与 embedded 回执成功 |
| 132417161059 | Ubuntu dashboard build | go1.27.4 linux/amd64；web 测试、构建与 embedded 回执成功 |
| 132417161099 | Windows dashboard build | go1.27.4 windows/amd64；web 测试、构建与 embedded 回执成功 |

三平台的 27 个包集合一致，含既有 `internal/checkrun`，不含嵌套工具模块。7 个共同 SKIP 为受环境控制的真实 fetching Git／旧 Git 测试与 M14 两项：`TestClaimWithARealGitThatFetches`、`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches`、`TestM14ReadOnlyProbe`、`TestM14OpenPath`。Ubuntu 常规步骤另有 8 个 CrossUser SKIP，随后独立 root 步骤实际 8 项 PASS；Windows 另有 3 个 unreadable 权限测试 SKIP。不能把 root 步骤说成三平台执行，也不能把常规步骤的 SKIP 覆盖掉。

`internal/cli` 耗时：Ubuntu 462.437 s、macOS 165.584 s、Windows 305.791 s，只是观测，不是上界、根因证据或维护事项关闭理由。

七个显示版本的作业均是 Go1.27.4，Web 没有 Go 安装。常规 workflow 运行根模块，八份日志没有执行 `prototype/c9evidence`；其成功不验证固定 Go1.27.3 工具模块、候选方法、正式 LC 或预算。Ubuntu 根模块的 race 也不能移作工具模块 race 证据。既有 CI 失败／单次重跑按此前报告范围保留，本轮不重新全量核验历史日志。

## 8. 直接回退、授权和后续准备

本轮对修改处与消费者的检查没有发现以下直接回退：来源核验移回父端；WorkDone 早于必要工作完成；W 放弃等待后被晚到成功恢复；文件真实 Close 错误被丢弃；未确认对象被误写释放；F 的真实失败被预期负向结果吞掉；或写者资格／TestWriterExpiresMidway 的生产判定改变。相关代码未因本轮接缝或分步改造被旁路。

可以交负责人作**第一阶段成果与下一阶段范围**的单独决定，但须保持：

- 技术 GO 不等于目前已有合并授权；本轮没有合并或改变 PR 元数据。
- 真实 invoker 仍停止；平台控制协议、LC-9、真实输入兼容性、真实三平台运行及 120 秒预算尚未交付／验证。这些不新增为本轮阻塞，也不能用它们导致的必然拒绝来掩盖范围内错误。
- E-10(ii) 未确认，LE-5 ①／② 与 N2 未批准。不能以第一阶段测试成功启动真实调用型夹具、建立运行分支或执行正式入口。
- 路径 1、T-1、L-b1a 保持既有决定；不扩大为 T-2／L-b1b 或相关原始路径／发送信号风险接受。H-1 未采纳，原方法停止。
- 没有精确例外获批，没有任何 G 项由本报告自动满足；第三批生产实现及 WP2.5／M2 验收未授权。

最小下一步是负责人审阅本报告并分别决定第一阶段的处置、N-150-1 的记录／测试收尾范围、是否准备下一阶段及其明确前置。若只授权归档和文档收尾，不应顺带实现真实调用者或触发新 workflow。

## 9. 实际操作范围与结束身份

审查者只读取源码、文档、Git 对象、哈希及已有本机／GitHub 日志，静态推演事件顺序；没有运行 Go、测试、构建、采集器、正式入口或实验，没有触发／重跑 CI、改代码、改 PR、提交、推送或合并。日志中已有 Go 命令是实现者的历史执行，不是本轮执行。

结束复核时刻为 2026-11-14 20:08:51 +09:00：本地 HEAD、tree、工具子树、当前分支与 §2 一致；重新查询的远程工作分支、远程 master 及 PR head/base 均与固定对象一致，PR 仍 open、未合并。已跟踪文件 diff 为空，工作区唯一新增为未提交的 `docs/reviews/codex-review-150.md`；报告内仓库相对链接均可解析。

另如实区分：本地未检出的 `master` 引用仍停在 `56d41b97e83abe85f59f4d7370f8effc617b8448`，其 reflog 最后更新时间为 2026-11-06 00:43:43 +09:00；它不是本次 PR base 的取证来源。受审当前分支、远程 master 与 PR base 没有漂移。本轮没有更新、切换或重置这个陈旧本地引用。

报告保持未提交。审查完成后暂停，等待负责人决定。
