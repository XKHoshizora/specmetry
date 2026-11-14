# Codex Review 149：PR #56 的 Review 148 残余与旧测试修正

审查日期：2026-11-14。对象为 `23227b689b052f26cdd1205e3e1f46d767ec6055`。本报告只作独立静态复审及已有证据核验，不修改实现，不运行 Go、测试、构建、采集器或实验。

## 1. 裁定

| 项目 | 裁定 |
|---|---|
| R-a：来源核验与父端截止出口 | **关闭**。核验在登记、准入后的工作任务中执行；父端只消费发布状态和记录计数。与 B-147-4 的同一残余合并裁定，不重复计数 |
| R-b：必要工作完成资格 | **关闭**。工作完成在 F/O、R、LC-1、来源核验之后发布；放弃等待与晚到资格持续进入终判 |
| R-c：实际外层操作准入 | **主体修复成立，仍有 B-147-1-R-c/P**：复合准备／修改内部的后续文件操作没有重新准入，前一操作跨过截止后仍可开始新动作 |
| `TestWriterExpiresMidway` | **本次测试修正成立**。保留先到期返回、再释放阻塞的顺序；两条合法出口分别检查；有界等待两个终态及资源断言均保留 |
| B-147-1 整体 | **不能整体关闭**，只剩上述准备内部的截止接点；R-a、R-b 不重开 |
| B-147-4 整体 | **在受审第一阶段范围内关闭**。Review 148 已认可的语义来源链和 F 域消费，加上本轮 R-a 的接线修复成立 |
| B-147-2、B-147-3、N-147-1～3 | 保持 Review 148 的限定关闭／完成结论，未发现本轮直接回退 |
| 第一阶段技术结论 | **NO-GO**，原因是一个范围内的具体截止缺陷，不是尚未交付真实调用或平台协议 |
| 合并建议 | **当前固定对象暂不建议合并**；先定向修复 R-c/P 并独立复审 |
| 下一阶段准备 | 可交负责人决定本报告所列最小修复及其验证；尚不建议据此直接进入下一阶段实现或正式运行授权 |

技术裁定、合并、真实调用、正式取证和第三批生产实施分别保留决定边界。E-10(ii)、LE-5 ①／②、N2 及既有执行限制不因本报告改变。

## 2. 固定身份、范围及登记

### 2.1 开始身份

开始核验时间为 `2026-11-14T18:50:55+09:00`。本地、远程引用和 PR 元数据一致，工作区干净，报告文件尚不存在。

| 身份 | 核验值 |
|---|---|
| 仓库／PR | `XKHoshizora/specmetry`，[PR #56](https://github.com/XKHoshizora/specmetry/pull/56)，open |
| 分支 | `proto/wp2.5-c9-evidence-tool` |
| HEAD | `23227b689b052f26cdd1205e3e1f46d767ec6055` |
| HEAD tree | `ab648cf2b0ad7952a3b5c0479428f12b230b8ef7` |
| 工具模块子树 | `c162da2cff482ee9bb3ab2be2e245928bc4f3ed5` |
| base／远程 master | `9137279554deed5a542dbd61b871d1cfa79bc323` |
| 上轮对象 | `fcf41b02a341ae35b59bcc43ad80464bd190e54c` |

读取当前 AGENTS.md、[Review 148 全文](codex-review-148.md)、[Review 147](codex-review-147.md) 的相关基线、[实现记录 §9](../wp2.5-c9-evidence-tool.md)、[处理记录 §192／§193](../review-resolution.md)，并沿 [限定取证计划](../wp2.5-c9-limited-evidence-plan.md) 和 [实施计划 §12.14](../wp2.5-implementation-plan.md) 的直接依赖检查。冻结规则、既有方法与执行边界沿用已审范围，没有重新全面审查底层方法、pidfd 或平台回收协议。

本轮实际差异为 21 个文件，2400 行增加、85 行删除。生产逻辑的修改位于 `chain/outer.go`、`collect/record.go`、`controls/controls.go`、`controls/lc23.go`、`job/job.go`；其余为回归、归档及文档记录／状态同步。检查了全部差异及相关调用者、消费者，不只检查测试名称或实现者说明。

### 2.2 顺序、归档和边界

已核对提交顺序：

1. `bfad05d` 只归档 Review 148。
2. `4c7e6f7` 登记 §192，先于 `ef1d9ad` 的实际修复。
3. `3362ac6` 记录修复和首次验证的结果。
4. `1e15714` 登记 §193，先于 `3b7d55d` 的测试修正。
5. `3b7d55d` 只修改 `writer_test.go`；最终 `23227b6` 是实现记录更新，工具子树与 `3b7d55d` 相同。

Review 148 的工作树文件与独立归档提交 blob 一致：28373 字节，SHA-256 `0b3f5e51c576c866dc8c2b994eafc2168b153ffb688d97c98773161c9d69cbba`，与登记值一致。处理记录从上轮对象、§192 登记、修复记录到 §193 的历史字节均保持为最终文件的前缀，未回改旧记录。状态仍为“已修复，待 Review 149 复审”，没有提前宣告审查关闭。

本轮没有改变冻结正文、验收项、根模块生产代码与依赖、常规 CI、取证 workflow、原 W/D 或 120 秒预算。`git diff --check fcf41b0 HEAD` 本轮没有输出；前次归档末尾空行的历史告警不能移植为本次新问题，也没有为消除告警修改历史报告。

## 3. R-a：来源核验与父端出口——关闭

下列源码位置均指固定提交 `23227b689b052f26cdd1205e3e1f46d767ec6055`。

### 3.1 实际接线

[job.go](../../prototype/c9evidence/internal/job/job.go) 290–339 行先在锁下登记 `WorkRecord`，再启动工作 goroutine；531–559 行的 `verifyRecords` 先登记 `ProvRecord`，再通过原工作预算准入。取得待核验记录、来源检查、哈希和解码都在工作任务中，`j.s.verify(items)` 在状态锁外执行。

核验结果发布时保存状态、覆盖条数、末项引用、问题集合、时刻和及时资格。未准入、未完成、晚到结果均不能作为有效核验。真实问题不依赖及时资格才保存：已发布的问题在 `copyState` 中持续进入 Defects。

父端的 `Copy`／`Level1`／`Finish` 现在消费状态；`copyState` 653–759 行只读取来源核验发布记录、`rec.Len()` 和已有错误，不再调用 `VerifyProvenance`、重新解码或同步遍历全部原始／派生记录。[record.go](../../prototype/c9evidence/internal/collect/record.go) 的 `Len` 为短锁读取。这里并不承诺状态复制或调度具备硬实时上界；排除的是原先在父端到期路径重新启动核验的外部工作依赖。

### 3.2 覆盖与实际记录绑定

本轮没有替换已经受审的来源语义检查。实际核验的是本次 recorder 保存的原始及派生记录，而非另造一个独立“正确列表”；记录采用只追加、私有保存的路径。核验完成后条数改变，681–683 行明确记缺陷，旧成功不能覆盖追加项。核验缺失、未运行、不完整、晚到以及来源问题分别有持续结果。

同一批 recorder 记录经既有 Sink 进入记录链。正常工作先完成记录再核验，最后才发布 WorkDone。异常追加在封口后仍按既有内存／artifact 边界保留，不能承诺失败运行的 artifact 自动追上全部晚到记录；该边界没有被本轮扩大为成功证据。

### 3.3 回归和限制

读取了 [regress148_test.go](../../prototype/c9evidence/internal/job/regress148_test.go)、[regress148b_test.go](../../prototype/c9evidence/internal/job/regress148b_test.go) 与 [provenance_test.go](../../prototype/c9evidence/internal/collect/provenance_test.go)：包含实际 Job 入口上的核验阻塞、父端 W/D 出口、晚到成功、晚到真实问题、覆盖后追加以及可解析但语义父项错误的反例。父端返回后仍持有工作记录、核验任务及结果，释放由测试自身有界收尾负责。

修复前日志中未建立“核验已进入”前提的失败，不算动态证明原缺陷；新增状态结构使旧版本无法编译的测试也不算红绿对照。修复后日志与源码提供的证据足以关闭本接点。R-a 同时完成 B-147-4 的共享核验器接线残余，按一个缺陷处理。

## 4. R-b：必要工作完成资格——关闭

### 4.1 Review 148 窗口不再恢复有效结论

[job.go](../../prototype/c9evidence/internal/job/job.go) 327–337 行的顺序是：F 及其 O 操作完成、R 与检查记录／LC-1 完成、来源核验完成，之后才发布 WorkDone。根夹具的 defer 收尾属于 `runControls` 返回之前的工作，不在 WorkDone 之后补做。

父端在 W 结束等待时，只有工作尚未 Done 才置 `Abandoned`。该事实保持到终判：即使后台随后在 W 与 D 之间完成，697–700 行仍保存放弃等待或晚到完成，不能恢复结论 3／4。LC-1 的命中也独立核对发布资格，晚到的 Hit 改为 Pending；最后一个 R 的发布时刻另有持续的预算失败记录。

因此，用户点名的顺序“最后一个 R 及时发布 → 工作暂停 → 父端 W 放弃 → 后台 W 后补齐 → Finish”不再被最后一份 R 的及时资格代替全部工作的完成资格。

### 4.2 边界与健康对照

完成资格在发布锁下读取原时钟，以 `now.Before(W) && !Abandoned` 判定；等于 W 属晚到。完成和计时器同时可读时，父端选择哪一分支都不能改变已经发布的 WorkDone 资格：及时 WorkDone 已存在，则计时器分支不补写 Abandoned；晚到 WorkDone 即使先被消费也仍不及时。

必要工作全部在 W 前完成，只是父端晚获调度的健康路径保留。这里没有要求取得锁、时钟检查和后续每条机器指令硬实时原子；采用的是既定发布逻辑判定点。

新增回归经过实际 `RunDomains`、`Finish`，分别覆盖最后 R 后暂停、晚到 LC-1、恰好 W、W 后完成、父端晚醒以及放弃后晚到真实问题。定向测试使用明确闸门和释放安排；健康、失败和持有断言均保留。未把结论 2 单独当作截止正确的充分证据。

## 5. R-c：主体成立，但准备内部仍有一个阻塞

### 5.1 已成立的修复

[outer.go](../../prototype/c9evidence/internal/chain/outer.go) 165–187 行的 `AdmitOp` 在原截止和锁存下区分准入／拒绝；外部调用在锁外。准备按 W、清理按 D，被拒绝的操作登记为 Refused，不伪装成实际执行失败或成功。

[job.go](../../prototype/c9evidence/internal/job/job.go) 419–449 行的根目录创建与 defer 清理已接入该协议；[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 316–323 行的 `fixture.cleanup` 也在 RemoveAll 前复核 D。创建后的目录由 Job 持有，清理失败或拒绝继续保留责任。分析跨过 D 才返回时，不再由 defer 无条件启动清理；W 之后、D 之前的收尾仍允许。这些主体修复不重开。

### 5.2 BLOCKER：B-147-1-R-c/P——一次准入覆盖了后续多次新文件操作

**位置**：固定提交 `23227b6`，[controls.go](../../prototype/c9evidence/internal/controls/controls.go) 261–307 行，尤其 `prepare` 267–274 行及 `writeFiles` 301–306 行；[lc23.go](../../prototype/c9evidence/internal/controls/lc23.go) 170–176 行；[job.go](../../prototype/c9evidence/internal/job/job.go) 376–380 行。相关实现说明为 [实现记录 §9.3](../wp2.5-c9-evidence-tool.md) 的“每个新的操作都在开始之前单独判定”。

**既定规则**：[处理记录 §192 第 5 项](../review-resolution.md) 要求“每次开始新的外部操作之前重新判断”，准备以 W 为截止；Review 148 §3.4 的最小修正同样要求每次新操作决策复核。允许已经开始的文件操作晚到返回，不等于允许随后再开始新的创建或写入。

**当前代码**：`prepare` 只在入口调用一次 `Outer.Prepare`，随后执行 `MkdirTemp`、`Hold` 和 `writeFiles`。`writeFiles` 每个文件依次调用 `MkdirAll`、`WriteFile`，各调用及下一文件之间没有新的 W 准入。`change` 也只在闭包入口准入；LC-3 的“handle not a regular file”闭包先 Remove，再 Mkdir，中间同样没有复核。把整段代码注释为“一次已准入的外层操作”，没有兑现已写定的逐次新动作守卫。

**可达顺序 A**：

1. 工作任务与一个真实夹具准备在 W 前准入，开始 `e.mkdirTemp`。
2. 此调用在 D 后才返回成功；父端已经可以按 W/D 返回。这是允许的已准入操作晚到返回。
3. 代码调用 `Outer.Hold` 保存目录，内部还读取了此时的时钟；该读取只记时，不作准入判断。
4. 随后无条件进入 `writeFiles`，在已过 W、甚至已过 D 时新启动 `MkdirAll` 和 `WriteFile`。第一个文件操作之后还可继续下一个文件。

**可达顺序 B**：LC-3 的修改闭包在 W 前准入；Remove 在 W 后返回 nil，闭包立即新调用 Mkdir。这里不是同一个已开始系统调用的继续返回，而是工具代码开始了另一个外部动作。

**影响**：父端有截止出口、最终非零、目录有人持有，都不能使这些后续动作自动获得准入。后台工作在父端退出后仍可能继续改变夹具文件，违反本轮明确授权的截止规则。它不依赖真实 go 调用、平台 helper 或尚未授权的机制；LC-2／3／5／6／8 的实际准备共用这段代码。既有回归在“整个准备入口”和“分析返回后的清理”处检查边界，没有覆盖准备／修改内部跨界，所以已有通过记录不能排除此路径。

**最小修复边界**：保持现有 W/D、目录持有及真实错误消费；在工具源码中的各个新的文件操作之前重新进行相应外层准入，或提供等价的逐步骤准备接口。前一操作晚到后，后续被拒绝的步骤不得启动，也不得记作真实失败／成功；已经产生的目录与部分文件继续由既有持有者负责，清理仍单独按 D。给复合准备和 Remove→Mkdir 接点增加可控的中途越界回归，保留健康对照和每一步的实际调用计数／拒绝／持有断言。

本要求不包括改写标准库内部、在 WriteFile／RemoveAll 内部每个系统调用间插入守卫，不要求中断已经准入的调用，也不要求时钟检查与下一条指令硬实时原子。缺陷是本工具明确分开的多个外部调用之间没有再次决策。该项为 R-c 的一个残余，不再按每个调用点重复计数。

## 6. TestWriterExpiresMidway：最小测试修正成立

`3b7d55d` 相对上一提交只改 [writer_test.go](../../prototype/c9evidence/internal/chain/writer_test.go) 的这个测试及专用 `fileNamed` 辅助函数，没有修改 writer、diagnostic、Job 或终判代码。

固定 HEAD 的 188–266 行明确保留顺序：等待 derived Close 进入阻塞 → 假时钟越过 D → `h.final(done)` 在 Close 仍阻塞时返回 → 检查返回 → `dc.free()` → 有界等待 writer 和 diagnostic 两个终态。

| 断言 | 实际检查 |
|---|---|
| 返回时机 | 释放前取得父端返回，决定时刻不早于 D；辅助等待有上界 |
| 结论／退出码 | 必须为 Incomplete（2）及其对应非零退出码 |
| 出口 (a) | PathConsumed、最终分支 a、原因含 diagnostic 的 expired-stopped，三者同时检查 |
| 出口 (b) | PathExpiredFinal、最终分支 b、原因是 D 时尚无 diagnostic 终态；其他出口失败 |
| 晚到结果 | 释放后等待两个终态，再核对 writer expired/late、diagnostic expired-stopped/late |
| 真实关闭 | raw 及时 nil、derived 晚到 nil；各仅关闭一次，CloseStarted／Returned／Released 均检查 |
| 未启动关闭 | controls、closeout 的关闭次数为零，仍由 writer 持有，不能误记释放 |
| manifest | 从未创建，既无 Create 结果也无该对象的持有记录 |
| 持续持有 | Held 集合恰为上述两个未关闭记录文件 |

专用 helper 缺少文件时返回空状态，但后续必需的 Close／Holder 断言会拒绝，未以跳过缺项换取成功。测试的故障闸门还由 Cleanup 负责释放；等待最终返回和两个终态均有有界失败出口。它没有提前释放 Close 来制造通过，也没有只把预期放宽成“任意结论 2”。

两条合法出口分别取决于父端最终消费前 diagnostic 是否已发布终态；修正反映既有协议允许的竞争，不改变工具判定。500 次记录确实同时出现两种出口，但不证明全部调度、不等于 `-race`。

## 7. 已关闭项、直接回退与授权

- B-147-2：本轮没有撤回记录文件及 manifest 的真实对象持有、唯一关闭和释放知识区分。测试在晚到关闭后仍要求 Released，未启动关闭的对象仍有实际持有者。其限定关闭保持。
- B-147-3：`ReadWhole` 及类型检查持续保存多项已观察错误的主体修复未回退；本轮来源核验、工作完成和外层拒绝都没有把真实错误改写为未评估或成功。其限定关闭保持。
- N-147-1：LC-1 从应检查集合判断的修复保持；本轮追加了发布资格，未使失败／未评估模块因缺少 Snapshot 消失。
- N-147-2：标签敏感回归及目标 Context 的既有范围保持，不宣称正式 LC 增项或真实 go list 证据。
- N-147-3：start/W/D 及偏移记录未改预算，保持原限定。块注释措辞没有本轮回退。

本轮新增的 R-c/P 不改变这些结论，也不是要求提前完成真实调用者、平台控制协议或 LC-9。正式入口仍使用停止调用者；LC-9 仍未执行，不具备正式有效取证出口。根模块常规 CI 排除独立工具模块。

取证 workflow 未改变，只匹配指定 `evidence/c9-run-1` 分支；只读远程引用查询没有该分支，按该分支查询既有运行结果为空。本轮没有创建分支或操作 workflow。这些检查的范围是当前远程状态和可取得的运行元数据，不据此声称任意外部副本不存在。

## 8. 既有本机证据：核验结果与限制

### 8.1 实际取得的记录

读取了用户提供目录下 s192／s193 的现有日志，并核对字节数、SHA-256、命令、退出记录、测试输出及所记前后身份，与 [实现记录 §9.5～§9.7](../wp2.5-c9-evidence-tool.md) 一致。本轮没有重跑任何命令或测试。

日志共同目录为 `C:\Users\hoshi\AppData\Local\Temp\claude\b96df3361c677053\67d963ef-6de7-4877-aebf-656e909830be\scratchpad\`；下表路径相对该目录。

| 日志 | 字节数 | SHA-256 |
|---|---:|---|
| s192/logs/pre-fix-regressions.log | 6311 | `59209644c642e1214421280460e7ab5eecb9572c3fc5eb3b8e6d139ada1964ce` |
| s192/logs/pre-fix-stuck-at-D.log | 1351 | `19fc1db12d5c02b754301a10a56e6ee382829a4b0e7b068e16eefaef2353f8c5` |
| s192/logs/final-full-test-run1.log | 46880 | `1cfb1b15064596ded1c7833f17e12737d0d2b91b9b7d42ad771ea409bda47d24` |
| s192/logs/final-full-test-run2.log | 46003 | `32e9a5ec1d866ae359ad26b35dda424c0b62c18604471cc32d69a5e93e24cb73` |
| s192/logs/final-static.log | 2699 | `703cb16b85af3bfa64acba74135780738b8d5de87ae4daefff6604f588f8684b` |
| s192/logs/final-regress148-x20.log | 100812 | `3473c11a909f75139e508967484c0f4d02312f99868318d295cd53309c613229` |
| s192/logs/flake-writer-expires-midway-fcf41b0.log | 1927 | `443500507da97a1547c3e67dad1c79cd2e04224bc1739f1d5637cb2ee1e92c81` |
| s192/logs/flake-writer-expires-midway-ef1d9ad.log | 2519 | `ef796f9a3dad68e37cb5d096ed61e66e6d680aca2c6f70d31a4332125924c920` |
| s193/logs/targeted-test.log | 1195 | `446b9d2a5b0846c0f886b51347c4c6ae99f5dfd53912c637de3779526629ff62` |
| s193/logs/repeat500.log | 72047 | `6c0c0ea2c88928e91a3f7fbbb03cd3de9e338cf1a0f24ac5f7dce59edb6d962a` |
| s193/logs/chain-test.log | 8060 | `1e0ba91e9a12ae6a09dc7b068d213d3313b4db0dd0d65ac70076db5b34a51ebf` |
| s193/logs/full-test.log | 46067 | `8e02d5c6db3c62f81817e8e54065fbe9e398e56ce179eefde046f864d6f31e39` |
| s193/logs/static.log | 2699 | `5e3f4f0df6aa279c0f888d7f3eb3e7911bfb32297f2554066832ea2bef954c85` |

### 8.2 不能合并抹平的结果

| 代码关联／记录 | 可采结论 |
|---|---|
| 修复前 `4c7e6f7` 加登记的测试／接缝 | pre-fix-regressions：9 个失败、6 个通过、退出 1；其中没有进入预定前提的失败不算原缺陷命中。单独 stuck-at-D 记录为有界等待超时失败。不能把不同工作树补丁当作无修改的旧提交运行 |
| `ef1d9ad`，子树 `31c11ad1d118c5045e09342d9f78ab43eeec0e68` 的 s192 第一次全量 | **失败，退出 1**；writer 测试出现空指针。后一次成功不抵消本次失败 |
| 同一 ef1d9ad 的 s192 第二次全量 | 187 个顶层 PASS、271 个子测试 PASS、1 个 SKIP、11 个 ok 包、退出 0 |
| 同一修复子树的 s192 定向重复 | 49 个不同顶层回归各 20 次，共 980 个 PASS，4 个包，退出 0；不是 race 或全调度覆盖 |
| 旧 `fcf41b0` 的 writer 探针 | 记录只复现出口断言失败；没有取得旧版本空指针的运行证据 |
| ef1d9ad 的 writer 探针 | 出口断言失败与空指针均有记录；“旧代码也可能空指针”只能是静态推断 |
| `3b7d55d`，s193 专项／chain | 专项 1 次通过；chain 包 41 个顶层、31 个子测试通过 |
| 同一 s193 修复子树的 repeat500 | 500 次 PASS；日志分支 a 出现 2 次、b 出现 498 次，无失败；这只是该组调度观测 |
| 同一 s193 修复子树的最终全量 | 187 个顶层 PASS、271 个子测试 PASS、1 个 SKIP、11 个 ok 包，退出 0 |

最终全量的 1 个 SKIP 是 `TestTakeNegatives/symbolic_link` 的权限条件，不写成符号链接负向断言通过。类型检查受控负向输出与测试进程的真实 panic 分开识别，不能仅检索 panic 字样混算。

s193 日志所记测试前后 HEAD 为 `3b7d55d7fe003ecd826212a41f9a92faa56d58b2`；其工具子树与最终 HEAD 完全相同。该关联支持把这些记录用于本轮固定工具代码，而不支持把日志当作不可变产物认证。

静态日志记载 gofmt、vet、带正式标签的 vet、六配置交叉 vet、正式入口仅编译、模块校验和导入闭包查询，均退出 0。它们不是三平台运行测试；编译正式入口不是执行正式入口。记录中的本机环境为 Windows/amd64、go1.27.3、GOTOOLCHAIN=local、GOPROXY=off、GOWORK=off；本轮只读核对了所指 go.exe 的 SHA-256，与记录相符，但不以当前文件哈希认证历史进程实际载入的每一字节。

上述是**审查者读取并核对实现者原始记录**，不是审查者独立复现。修复前无法编译、夹具未建立前提、不同导出树或测试补丁的记录均保留各自适用范围。首次失败、第一次夹具时钟／等待问题和后续修正不被后来的绿色改写。

## 9. 常规 CI：当前固定对象独立核验

[run 43579020189](https://github.com/XKHoshizora/specmetry/actions/runs/43579020189) 为 `pull_request`、attempt 1、completed／success，8/8。已读取八份实际作业日志，均检出：

`feddaac92dcae8aa900cdf4092a9f3f8b91d4793`

另外读取该 Git 提交对象：父提交依次为固定 base `9137279554deed5a542dbd61b871d1cfa79bc323`、固定 HEAD `23227b689b052f26cdd1205e3e1f46d767ec6055`；tree 为 `ab648cf2b0ad7952a3b5c0479428f12b230b8ef7`，与 HEAD 同树。该结论来自 checkout 日志与提交对象，不只来自作业 head_sha。

| 作业 | job ID | 实际版本／结果 |
|---|---:|---|
| Go lint | 132399535011 | go1.27.4 linux/amd64；0 issues，成功 |
| Go tests Ubuntu | 132399535127 | go1.27.4 linux/amd64；27 个 ok 包，15 个普通测试 SKIP；独立 root CrossUser 8 项 PASS |
| Go tests Windows | 132399535143 | go1.27.4 windows/amd64；27 个 ok 包，10 个 SKIP |
| Go tests macOS | 132399535213 | go1.27.4 darwin/arm64；27 个 ok 包，7 个 SKIP |
| Web | 132399535236 | 成功；没有安装／显示 Go 版本，不补写 Go 版本 |
| Build with dashboard Ubuntu | 132399742187 | go1.27.4 linux/amd64；构建与相关检查成功 |
| Build with dashboard macOS | 132399742213 | go1.27.4 darwin/arm64；构建与相关检查成功 |
| Build with dashboard Windows | 132399742223 | go1.27.4 windows/amd64；构建与相关检查成功 |

八份日志没有实际测试 FAIL；三个根模块测试作业的 27 个包集合相同，包含 internal/checkrun，不包含独立模块 `prototype/c9evidence`。Ubuntu 的 race 路径与独立 root CrossUser 步骤分别核对；Windows/macOS 没有执行该独立 root 步骤，不能记成跨平台 CrossUser 证据。

共同的 7 个 SKIP 为真实指定 Git 环境／M-14 探针：`TestClaimWithARealGitThatFetches`、`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches`、`TestM14ReadOnlyProbe`、`TestM14OpenPath`。Ubuntu 普通步骤另有 8 个 CrossUser SKIP，随后 root 步骤实际通过；Windows 另有 `TestResolveCommitsWithUnreadableStorage`、`TestClassifyWithUnreadableStorage`、`TestSourceListUnreadableSource` 的 3 个平台条件 SKIP。

CLI 耗时为 Ubuntu 380.390 s、Windows 387.788 s、macOS 226.561 s，仅是本次观测，不作上界或维护项已解决的证据。历史 CI 的失败与重跑事实按 Review 148 及前序已核验范围沿用，本轮没有重新全量核验所有历史作业，也不将本次称为旧运行的重跑。

根模块 go1.27.4 的常规 CI 没有运行独立工具模块，不能替代本机 go1.27.3 的工具测试，更不能证明候选方法、真实平台控制、正式取证或预算成立。

## 10. 最小下一步与结束核验

下一步建议仅限交负责人决定是否授权：修复 §5.2 的复合准备／修改内部准入接点，增加有界的中途跨界与健康回归，继续保持实际对象持有、拒绝与真实错误的区别。无需重做已经关闭的来源语义核验、整体方法或平台协议；不因本轮缺陷扩大对照范围、时间预算或运行权限。

R-a、R-b 和测试修正可以按本报告限定采纳；R-c/P 修复后仍应按固定对象复核。真实调用者、平台控制协议、LC-9、真实输入兼容性与实际预算证据继续是未交付的后续范围，不能用它们造成的必然拒绝掩盖本阶段问题，也不把它们本身列为本轮阻塞。

结束核验时间为 `2026-11-14T19:03:41+09:00`：再次读取远程分支、master、PR head/base 及固定提交树，与 §2.1 一致；PR 仍 open。本地 HEAD、tree、工具子树均未漂移。工作区唯一差异是未跟踪的 `docs/reviews/codex-review-149.md`；报告中的仓库相对链接均指向存在的文件。未修改受审代码、测试、文档、依赖或 workflow，没有运行 Go／测试／构建／实验、提交、推送、合并或操作 CI。本轮完成后暂停。
