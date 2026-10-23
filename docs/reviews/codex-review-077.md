# Codex Review 077：PR #33 的观察错误与拒绝收尾定向复审

## 1．结论

**PR #33 技术 NO-GO，暂不建议合并；Review 075 B1 尚不能整体关闭。**

本轮修复确实解决了 Review 076 的两个具体触发点：`pidProc.ended` 的首次观察错误不再被清理成功覆盖；Linux 短链拒绝会逐条登记已有节点，并在排他区间内执行组终止及收养清扫。回归先行运行也确实检出了旧行为。

但共享调用链与新增证据仍有两项阻塞：

1. **B1-R1-S：Linux 清扫的第二次状态读取，以及新增的场景初始子进程枚举，仍能丢失观察错误。** 有具体调用方的源码级反例；没有冒称本机 Linux 动态复现。
2. **B1-R2-W：外层见证不是 fail-closed。** 写入失败可以留下空见证，外层仍 PASS；无效的启动身份会被当作原进程已结束。已在 Windows 隔离副本中独立复现，包括真实内层场景与真实外层门禁。

原短链缺陷的实现修复得到支持，但“外层能独立发现所有遗漏”的新增门槛尚未成立，B1-R2 的完整证据暂不关闭。Review 075 N1～N3 的已确认处理没有回退；Review 076 N4 继续为 NON-BLOCKER。另有 `missing` 的术语与账户含义问题，列为 N5，不将纯措辞问题升级为阻塞。

本结论只涉及 test-only 原型和证据有效性，不是生产 Runner 的缺陷裁定，不重开已验收模块，也不批准产品选择、规范冻结或生产实现。

## 2．固定身份与范围核验

审查日期：2026-10-23（Asia/Tokyo）。仓库：[XKHoshizora/specmetry](https://github.com/XKHoshizora/specmetry)，[PR #33](https://github.com/XKHoshizora/specmetry/pull/33)。

| 身份 | 完整 SHA |
|---|---|
| base | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2` |
| 本轮 HEAD | `acf699b1df18343bf97e734fdb2a3af379637fbf` |
| 修复代码 | `8fea6c607b8e760ddd3428d7d674739e2d6b8197` |
| 回归先行 | `1df33dff77ea252c4cae2535f82dd197ffd58697` |
| 上轮对象 | `3ddd14fb1d478e333ad922310f9bcfc2b70033d9` |

开始、结束均通过 GitHub API 核对：HEAD/base 与上表一致，PR open、未合并，`auto_merge=null`；没有对象漂移。本地 HEAD 同为固定目标，开始时工作区干净，报告路径不存在。读取了适用 AGENTS.md、完整 [Review 076](codex-review-076.md)、本轮全部差异、[原型结果文档](../wp2.5-prototype.md)和 [review-resolution](../review-resolution.md) §91，并追踪必要的共享调用链。

实际为 **22 个文件，+1104/−100，其中 16 个原型文件**。新增的三个原型文件为 `witness_test.go`、`witness_unix_test.go`、`witness_windows_test.go`。提交顺序：

1. `50156ece8e86dfef23fd6d7a9562ab6bc8b28db0`：归档 Review 076。
2. `1df33dff77ea252c4cae2535f82dd197ffd58697`：回归和见证先行，旧门禁未修。
3. `8fea6c607b8e760ddd3428d7d674739e2d6b8197`：原型修复。
4. `bf392953a146b671dc0cf3a0a4a02bd5ec8a4883`：结果文档。
5. `acf699b1df18343bf97e734fdb2a3af379637fbf`：§91 和状态同步。

`8fea6c6 → HEAD` 只有 AGENTS、implementation-plan、review-resolution、准备草案头部和原型报告五份文档变化。本轮没有改生产代码、Schema、迁移、根模块依赖、`ci.yml`、原型 workflow、architecture、data-model 或正式验收条文。`git diff --check` 通过。

review-resolution 的旧文本完整保留为新文本前缀，§84～§90 未改写。Review 076 归档 SHA-256：

`5d5e58b5609e365de78038dd0a20411826fbe3bb1942c641ef243a57b95f11a6`

与 §91 一致，原审查身份仍为 `3ddd14f`，NO-GO 没有被实施方“已修复、待复审”的状态替换。

## 3．B1-R1：精确反例修复成立，共享错误传播仍有残余

### 已确认的修复

Windows 与 Unix 的 `pidProc` 增加了独立 `obsErr`；后续成功不清除它。[通用清理](../../prototype/wp25/main_test.go#L336)在对象都结束后仍检查 `observeErr()`，将实验记为 invalid。清理是否完成和实验是否有效已分开。

本机 Windows 对回归先行版和固定目标分别运行真实外层：

```text
1df33df / observe.fail-once:
  inject_hits=3, got=confirmed, inner_exit=0, pass=false

固定目标 / observe.fail-once:
  inject_hits=3, got=invalid, inner_exit=1, pass=true
  launched=6, unowned=0, missing=0, left=null
```

健康 `control.silent-holder` 在两版均 confirmed/0。注入位于真实 `pidProc.ended`：Windows 先调用真实等待，再替换一次返回结果；Unix 替换一次状态读数。它没有直接修改 verdict。新版的后续结束及清理成功没有恢复实验有效性，精确的 Review 076 反例已经关闭。

共享调用链也有正确增量：主进程等待出错、组长状态 unknown、组成员读取失败、真实 Job 查询失败，以及清扫第一次列表/状态读取失败，会记入 `flag` 或 `obsErr`。正常超时轮询、尚未退出、报告尚未齐和真正可靠不存在，没有被一律当作观察错误。

### BLOCKER B1-R1-S：清扫的第二次读取仍把 unknown 当成可重试的不匹配

**位置**：[sweep 第二次 procInfo](../../prototype/wp25/exclusive_linux_test.go#L222)、[runCheck 的错误传播](../../prototype/wp25/exclusive_linux_test.go#L284)、[TestExclusiveGate](../../prototype/wp25/exclusive_linux_test.go#L345)。

`sweep` 在 pidfd 绑定前的 `procInfo` 对 unknown 设置 `r.obsErr`；绑定后的第二次读取却采用：

```go
st2, pr2 := procInfo(pid)
// ...
if pr2 != present || st2.ppid != self || start != st.start {
    r.mismatched++
    unix.Close(fd)
    continue
}
```

这里没有区分可靠不存在、真实身份变化与读取失败，也没有设置 `obsErr`。一次读取失败后，下一轮恢复成功可以完成清扫，`converged=true`、全部 `reaped`，而 `obsErr=nil`。

**具体调用方反例（源码级，不是本机动态运行）**：在 `TestExclusiveGate` 的一个完整三层检查中，令一个候选在 pidfd 绑定后的那次状态读取为 unknown，其后读取全部恢复。该轮只多一次 `mismatched`；它不影响下一轮回收全部三个节点。该测试计算 `good` 的条件检查收敛、回收数、集合包含和组长回收，**不检查 mismatched**；`runCheck` 也只对 `obsErr` 调用 `flag`。因此三个检查都可以计为 good，其他预测保持成立，最终 confirmed，而不是契约要求的 invalid。

这与原型文档 §2.2、§2.4 和 C-19 的“任何一次读取失败不可撤销”直接冲突。它不是要求把真实身份变化当错误，也不是认定发生了误杀：代码在这一次没有发信号，问题是错误被分类成不匹配并从终判丢失。现有一次失败注入只落在 `pidProc.ended`，检测不到这一点。

**同类直接增量**：[ownChildren](../../prototype/wp25/tree_linux_test.go#L215) 的首次枚举失败返回 nil；[begin](../../prototype/wp25/main_test.go#L149) 没有保留错误。若只有这次枚举失败，而结束时 `newChildren` 读取成功且没有子进程，普通正常场景可以继续 confirmed。不能将“初始清单未知”静默转成“已取得空清单”。`procGate.exclusive` 自己的基线检查能拒绝错误，不等于新增的场景级检查也已处理它。

**最小修正**：将清扫第二次 unknown 单独传播为不可撤销的观察错误；初始子进程枚举保留错误，而不是只返回 nil。沿受影响调用链检查尚未进入统一终判的状态读取/等待错误，明确正常等待和允许重试的条件。增加相应真实路径的单次失败后恢复测试；不用把原型 invalid 直接照搬成生产 F1 或退出码。

本轮 Linux 入口为 `Wsl/E_ACCESSDENIED`，没有运行 Linux 夹具或绕过拒绝。上述裁定依靠具体分支、字段和调用方预测的源码证据，不冒称 Linux 故障注入已经跑过。

## 4．B1-R2：区间内收尾有实质修复，但新见证仍不能兜住遗漏

### 原短链与损坏链的实现修复

[awaitOwned](../../prototype/wp25/report_test.go#L191) 每轮核验并登记已经到达的合法 PID，不再等完整集合；归属失败仍拒绝，不直接操作报告中的未知 PID。

[runCheck](../../prototype/wp25/exclusive_linux_test.go#L261) 在等待负载之前安装了拒绝收尾 defer。`FailNow` 栈展开先进入 [abort](../../prototype/wp25/exclusive_linux_test.go#L302)，此时仍在 `g.exclusive` 的回调中：先按持有组长的条件终止原组，再按空基线/唯一子进程来源的前提清扫被收养的节点；之后才离开回调、清除 subreaper、释放租约。报告损坏不再必然绕过对未报告节点的收养清扫。主进程及已登记对象还由通用清理结束/回收，Linux 场景结束新增了存活或僵尸子进程检查。

这是原型排他区间内的归属证明，没有把“全系统新出现的进程”视为自己，也没有为了收尾而扩大组信号或操作未核验的报告 PID。不能推广到区间外脱组且报告无法读取的后代；文档已披露这一边界。

已核验的 Ubuntu 正式日志：

| 情形 | 回归先行 `1df33df` | 修复 `8fea6c6` |
|---|---|---|
| 短链 | 内层 invalid/1，但外层发现 3 个存活节点，PASS=false | 内层 invalid/1；见证启动 4 个 helper（主进程+3节点），unowned=0、missing=1、left=null，PASS=true |
| 损坏链 | 内层 invalid/1，外层发现 2 个存活节点，PASS=false | 内层 invalid/1；见证启动 5 个 helper，unowned=4、missing=4、left=null，PASS=true |

这里不把旧版外层补救后的结果算成内层成功：`left` 在补救前已经保存，非空仍使该回归失败。没有把 helper 的 60 秒定时退出算成父端收尾。修复后的观察支持原具体短链/损坏链路径，但其独立见证还存在下面的直接问题。

### BLOCKER B1-R2-W：见证丢失和无效身份可以使外层假通过

**位置**：

- [witness](../../prototype/wp25/witness_test.go#L32)：打开失败直接返回；写入数量、写入错误和关闭错误被忽略。
- [readWitness](../../prototype/wp25/witness_test.go#L52)：空文件成功；只验证字段数及正 PID，不验证事件白名单、启动身份、重复或完整尾行。
- [checkGate](../../prototype/wp25/gate_test.go#L213)：对空见证没有完整性拒绝；只遍历读到的 launched，`Unowned <= Missing` 即可通过。
- [Windows openWitnessed](../../prototype/wp25/witness_windows_test.go#L35) / [Unix witnessAlive](../../prototype/wp25/witness_unix_test.go#L18)：记录中的无效身份与当前身份不同，就被当作原对象已结束。

#### 独立反例一：真实写入失败，外层仍以零个对象 PASS

在固定目标的仓库外副本，只在 `witness` 的写入前关闭其文件，使原 `fmt.Fprintf` 真实返回 `n=0` 和错误；不改内层预测、verdict、进程清理或外层判定。随后用独立侧写记录确认故障确实执行：六个真实 helper 都留下 `n=0 write_failed=true`，不是只创建了一个未使用的注入开关。

真实 `TestGate/control.silent-holder` 仍得到：

```text
got=confirmed inner_exit=0 launches=3 work=3
launched=0 unowned=0 missing=0 left=null pass=true
exit 0
```

同一故障下，`report.malformed` 也仍外层 PASS：内层正确 invalid/1，但 `launched=0`。空见证使“逐个核对已启动对象”变成空集合上的通过；它不能证明所有对象都被收尾。上述实验没有制造实际进程残留，所有对象仍按原 Job/持有句柄回收；无需故意留逃逸进程才能证明这条证据门禁失效。

实际故障可以是打开/追加失败；当前实现同样忽略它们。只有读取文件本身失败会进入 `Problems`，**不能将这一正确处理扩大成写入失败也已被处理**。

#### 独立反例二：unknown 身份被误当作可靠结束

在未修改原函数的额外小测试中，取本测试进程的真实 PID，只把见证启动身份设为代码本身可能产生的 `"unknown"`：

```text
witnessAlive({kind:launched, pid:本测试进程, start:unknown})
=> alive=false, err=nil
```

进程显然仍在执行此测试；健康的同 PID、真实 `selfStart()` 对照返回 alive=true。`selfStart` 失败本来就返回字符串 `unknown`，解析器接受它，后续“身份不同”路径便绕过 unknown。

同一组直接检查还确认未知 kind、重复 launched、未结束尾行及 `unknown` 身份均被解析器接受。未知 kind 会被后续遍历跳过。重复和缺少换行单独是否足以阻塞，要结合协议定义判断；本项阻塞依据是**丢失见证变成空集合通过，以及无法核验身份变成已结束**，不是机械放大格式细节。

**启动覆盖问题**：见证由 helper 执行到 `runHelper` 时自写；进程在创建后、写见证前即被终止或仍挂起时，不会留下 launched。一次本机健康归属拒绝中 `launches=2` 而见证 `launched=1`，也显示两种口径不能等同。它不自动证明发生了残留，但足以说明“每个实际创建的对象必然有自写记录”尚未实现。不能仅靠声明预期数量或内层 Owned，补出独立证据的完整性。

**最小修正**：

1. 见证建立、身份取得、写入/关闭失败必须可传播到外层的失败结果；空见证只有在独立确认未创建对象时才有合法含义。
2. 建立可核验的创建/登记与见证完整性关系，分别处理创建后未进入 helper 的对象、已执行 helper 和尚未产生的预期节点；不只相信内层汇总数量。可在原型的受控启动点及 helper 间建立可靠回执，不要求生产 Runner 提前接入。
3. 明确并校验事件、各平台身份格式、记录键与完整结束；`unknown`/损坏身份不得走“可靠身份变化，所以原对象结束”。
4. 对已知对象保留安全补救，但必须先保存内层结束时的事实；补救成功不能恢复该回归 PASS。无法确认归属时不得猜 PID 或扩大信号范围。
5. 增加见证写失败、缺失/损坏/重复/截断、未知身份及启动早退的外层正反例。

因此 B1-R2 的原实现缺陷得到修复支持，但本轮要求的独立收尾门禁仍不充分；暂不作完整关闭。没有据此否定日志中健康运行确实取得的观察，也没有要求全仓库或生产进程监管在本 PR 内完成。

## 5．NON-BLOCKER 与既有意见

### N5：missing 是声明缺额，不是已启动对象的身份账户

[awaitOwned](../../prototype/wp25/report_test.go#L199) 的 `missing = n - len(got)` 可以同时包含：

- 从未创建的预期节点：短链 missing=1；部分 fanout missing=20。
- 已创建但报告不可用、未能登记的节点：损坏链 missing=4。

外层计算的 `Unowned` 才是**见证中实际出现、却不在内层 Owned 列表中的记录数**；它不与 Missing 逐对象匹配。`Unowned <= Missing` 只是数值容纳条件，不是身份对应，也不是结束证明。

文档局部已解释“那一层没有启动”，但表头及“每个已启动对象已登记或被申报”的总述仍易误读。建议改称预期登记缺额，并分别报告实际已启动未登记数及观察状态。该术语问题本身是 NON-BLOCKER；独立清理门禁真正失效的部分已单列 B1-R2-W，不能靠改名关闭它。

### Review 076 N4：继续保留，不因此 NO-GO

holder 仍忽略 `report("epipe")` 的写错误。F1 场景的 `epipe` 最终规格仍为至多一条，而预测要求恰好观察一条；报告失败没有事件时，预测不符导致 refuted/非零，或若另有工具错误则 invalid，不能据此得到 confirmed。Review 076 已独立复现的分类问题未修复；本轮核对其错误分支、预测和终判仍相同，没有再次冒称新动态复现。

建议继续把报告工具失败与机制预测不同分开处理。它没有形成绿色假通过，保持 NON-BLOCKER。

### Review 075 N1～N3：不重开

C-4 仍在 D20 的待决范围，同一宿主后续调用与当前批次停止分开；Windows 观察句柄仍显式释放，没有恢复在已关闭句柄上调用验证 API；内部定时器仅在 Go 代码执行后生效，与父端期限分开。版本和运行身份的记录没有直接回退。新的见证问题不能用这些旧结论豁免，也不反过来否定它们已成立的处理。

## 6．独立验证、失败与未验证边界

本机：Windows 11 build 26300、amd64，已有 Go 1.27.1；未安装/升级工具。使用固定 Git 对象导出的仓库外副本；原仓库没有切换提交或生成测试文件。测试设 40～90 秒 Go 期限，外层自身还有子进程等待期限；只运行自己创建的合成夹具。

| 本轮实际运行 | 结果 |
|---|---|
| 回归先行版的 observe.fail-once + 健康 silent-holder | 故障命中三次，旧内层 confirmed/0，外层 FAIL；健康对照 PASS。 |
| 固定目标：observe.fail-once、report.malformed、foreign-pid、健康 silent-holder | 四项外层 PASS；故障分别由相应真实路径拒绝，观察失败为 invalid/1，命中三次。 |
| 固定目标：正常 fanout、部分 fanout、parent-fail、cleanup-unconfirmed、capacity-stop 合法反例 | 五项外层 PASS；正常 fanout 见证 155 个对象、work=155；部分 fanout 见证 11 个、unowned=0、missing=20、left=null。 |
| 固定目标：assign/confirm/resume 失败链、query-fails、normal-drain、F1-skip-drain | 六项根场景 confirmed/0；未再现关闭句柄检查的原生异常。 |
| 见证写错误变异 | 原始输出中仍 PASS；补充独立侧写确认六次真实写错误后，仍 launched=0/PASS/exit 0。 |
| 直接调用原见证解析及身份函数的额外断言 | 未知事件、重复、未结束尾行、unknown 身份及活进程 unknown 身份五项拒绝断言首次运行即 FAIL；真实身份的健康对照 PASS。与正常场景同跑，失败不是零场景导致。 |

上述九项外层定向回归与机制根场景、额外断言分别计数；没有把内层 invalid 加到机制 confirmed，也没有将重复的 normal-drain 计为新场景。未在本机重跑全部 T1～T6 或所有 19 项 Windows 门禁。

首次环境/夹具边界如实记录：仓库外导出和已有 Go 查询成功；WSL 入口随后返回 `Wsl/E_ACCESSDENIED`，该准备命令退出非零，但没有 Linux 夹具开始，未绕过拒绝。见证写错误的早期可见标记写到了 helper 的被测输出管道，不在外层结果流中；没有把外层 `inject_hits=0` 当成命中证明，随后用独立侧写核实实际 Write 的数量与错误。这个补证过程没有改变 verdict 或制造孤儿来取得结论。

Linux 短链与损坏链采用完整 CI 日志及源码核验，未声称本机动态复现；B1-R1-S 同样是具体调用方源码证据。macOS 只核对真实 runner 日志和对应源码。没有运行未知用户检查、强制 PID 复用、制造不可中断睡眠或终止未知进程。结束时本轮隔离路径下无存活测试程序；本轮 Windows 场景的已知对象通过 Job/进程句柄收尾。

作者 M8/M9/M10/M12 的变异及 WSL 结果仍为作者证据；本轮没有把它们全部重跑，也没有将 CI 绿色扩张成那些未提交变异的三平台验证。

## 7．CI 独立核验

读取五组 run 的正式元数据、所有 Job 状态及相关完整日志，包括六份原型 Job 和三组常规 CI 的 checkout。没有触发或重跑 workflow。

| run | 事件 / SHA / attempt | 正式结果与适用性 |
|---|---|---|
| [39886679862](https://github.com/XKHoshizora/specmetry/actions/runs/39886679862) | push / `1df33df` / 1 | completed/failure，三平台诊断均失败；回归先行的预期失败。 |
| [39886688329](https://github.com/XKHoshizora/specmetry/actions/runs/39886688329) | pull_request / `1df33df` / 1 | completed/cancelled；7 个 Job success，Windows Go tests cancelled。不能写成 8/8，也不能因没有测试 FAIL 说它完整通过。 |
| [39887528435](https://github.com/XKHoshizora/specmetry/actions/runs/39887528435) | push / `8fea6c6` / 1 | completed/success，三平台诊断均成功。 |
| [39887533937](https://github.com/XKHoshizora/specmetry/actions/runs/39887533937) | pull_request / `8fea6c6` / 1 | completed/success，8/8。 |
| [39888913896](https://github.com/XKHoshizora/specmetry/actions/runs/39888913896) | pull_request / `acf699b` / 1 | completed/success，8/8。 |

常规 CI 的八 Job 为 Web、Go lint、三平台 Go tests、三平台 Build with dashboard。实际 checkout 与 Git commit API 核验：

| run | 合并快照 | 父提交：base、对应 HEAD | tree（与对应 HEAD 相同） |
|---|---|---|---|
| 39886688329 | `64fc257c2b4dd4c05fd6513caff07142ba30cfa3` | `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`、`1df33dff77ea252c4cae2535f82dd197ffd58697` | `a276b636d9d600f832030cfecacbe9c513f8bdc0` |
| 39887533937 | `2c76407c982d92af940295b3c6618b9ab3d93694` | 同一 base、`8fea6c607b8e760ddd3428d7d674739e2d6b8197` | `f28fa5c1a224eac7ed0931c1b792074d0ea99b2d` |
| 39888913896 | `393416d28ffdaa0bc1d5ae594093cf5d57267ca8` | 同一 base、`acf699b1df18343bf97e734fdb2a3af379637fbf` | `54c654cf522c5fa518fb4d4148c36d34feb2ec53` |

最终实际快照与用户预期一致。push 诊断则分别直接检出 `1df33df` 与 `8fea6c6` 本身；没有把合并快照与直接 HEAD 混用。

### 原型日志与计数

| 平台 | 回归先行：普通场景 / 门禁失败 | 修复后：普通场景 / 门禁通过 |
|---|---|---|
| Ubuntu | 38 confirmed；10/21 失败 | 38 confirmed；21/21 通过 |
| macOS | 27 confirmed；8/16 失败 | 27 confirmed；16/16 通过 |
| Windows | 29 confirmed；9/19 失败 | 29 confirmed；19/19 通过 |

旧 observe.fail-once 在三平台均命中三次，内层 confirmed/0；新版均 invalid/1。旧短链 `left=[3849,3856,3864]`、旧损坏链 `left=[3888,3895]` 在外层补救之前已登记，均使门禁失败；不能在补救后改写为内层清理成功。新版两项均 left=null。

修复后三个平台唯一场景数为 38、27、29，并集 **53**，共 94 次平台观察。Results 步骤重印导致原始 `WP25RESULT` 行数为 76、54、58；按实际执行阶段和 ID 去重，不能将重印或内层运行重复计数。顶层唯一内层专用 SKIP 为 `TestInnerForeignPID`，它经相应门禁在内层实际运行；平台专属测试没有编入别的平台，不算跨平台通过。

Ubuntu 原型的实际命令按 Linux 分支带 `-race`；macOS/Windows 原型没有 race。常规 Ubuntu 的普通 Go tests 同样带 race；八项 root 跨账户检查在其后单独编译测试程序并 `sudo ... -test.run CrossUser`，确实通过，不能称为 root race。真实旧 Git/权限等设计跳过保持跳过，不据此重新验收旧模块。

常规 CI 不运行原型模块；诊断成功不能证明新增 witness 的错误分支已经被检出。既有失败、取消和历史观测保留，M-14/M-15/M-16 均未因此关闭。

## 8．候选、冻结门槛与接手

**C-19** 正确把原型 invalid 与生产 F1/环境错误分开，并承认只动态测试了一个观察点。但正文“其他观察点均直接记入终判”的概括不成立，需按 B1-R1-S 补齐；不能只加“未验证”标签而保留错误的无条件描述。

**C-20** 的收养归属只限 Linux 的受控排他区间；没有推广到其他宿主。区间外脱组且报告损坏的后代无法据报告找到，文档如实保留。候选没有并入冻结规范，不因本轮 CI 自动获批。它的拒绝收尾证据仍受 B1-R2-W 限制。

**PRODUCT DECISION**：没有新增由本审查批准的事项。14 项 D1～D9、D11、D13、D18、D19、D20 仍待负责人决定；C-4 仍属于 D20 的宿主准入选择。不得把上述原型错误的修复包装成需要负责人接受的新风险。

**ACCEPTED RISK**：本轮无新增。方案 B 的既有局部边界不变，不豁免 Log 完整性、其他账户保护、正常合作并发或误杀无关进程。原型见证丢失和 unknown 假结束没有获准作为接受风险。

冻结/实现授权前的最小工作：

1. 补齐共享观察错误的不可撤销传播；修复见证写入、身份及完整性门禁，补实际内层/外层错误回归和健康对照；同步证据说明。N5 术语收口，N4 继续独立登记。
2. 完成此前已列明的机制规格/必要原型：C-9 对全部启动、等待、回收及信号能力的规则设计；全局闸门拓扑、上下文 Git 的先后与自锁/取消/F1 策略；macOS ABI 与枚举错误预算；Windows 父端期限和未覆盖失败点；T5 路径、PATH、解释器来源与支持范围。
3. 负责人选择产品项并同步相关规范、契约及验收；本审查不代替批准。

获准实施后、生产验收前再完成 Runner/CLI 接线、生产闸门与 C-9 检查、真实取消与继承输出、配置固定、空批次、worktree/seq、迁移/Schema、事务及 §8.6 的组合失败和三平台生产测试。原型未验证这些内容，不把它们自动增列为本 PR 的阻塞。

## 9．最终回答

**A．B1-R1 / B1-R2 / B1：** pidProc 精确反例和原短链实现已实质修复；共享观察仍有 B1-R1-S，新见证仍有 B1-R2-W，因此本轮不作两项的完整关闭，B1 不能整体关闭。

**B．PR #33：技术 NO-GO，暂不建议合并。** 两组最终 CI 均正式成功，但不能替代对这些错误路径的验证。

**C．证据强度：** 真实外层与内层能检出旧精确反例；Linux 区间内收尾有源码及真实 CI 支持；新见证能在其完整且身份有效时发现旧残留，并在补救前保存失败。它尚不能保证见证失败时仍拒绝。Windows 的新反例已独立动态复现；Linux 新残余是具体源码路径证明，未冒称本机运行。

**D．范围：** 仅需针对原型门禁及证据补齐最小修复；不要求本 PR 提前完成生产 Runner。14 项产品选择未批准、规范未冻结、WP2.5 生产实现未授权、WP2.5/M2 未验收；维护项保持原状。

结束时远程身份和 PR 状态与开始一致。原仓库只新增本未提交报告；没有修改其他文件、提交、推送、合并、改基、重跑或触发 workflow。审查到此停止，等待负责人决定。
