# Codex Review 054：PR #25 诊断工具有效性定向复审

审查日期：2026-10-19。审查者：Codex。

**NO-GO。Review 051 B1 尚不能整体关闭。** 崩溃、空结果、零工作量和期限等原路径已得到有效修复，N1～N4 已合理处理；但结果解析仍接受非法 JSON 尾部，且已通过真实诊断父进程复现“损坏结果被报告有效并 PASS”。这是诊断工具的剩余缺陷，不是要求修复生产 SQLite。

## 一、固定身份与只读边界

- [PR #25](https://github.com/XKHoshizora/specmetry/pull/25)。
- 固定基准：`a236e5350ff92dcffd9bc05d09b1f3aac3996704`。
- 本轮固定 HEAD：`d2f1d35a5d38564893d991b7c6d872b854e5e8b0`。
- Review 051 对象：`db8d7bc29269727448f5dba893e3dce2a165a460`；原诊断代码：`1d8b73c9e637e339f2e1ebac4c81b5bd029480de`。
- 已完整阅读目标中归档的 [Review 051](https://github.com/XKHoshizora/specmetry/blob/d2f1d35a5d38564893d991b7c6d872b854e5e8b0/docs/reviews/codex-review-051.md)。

开始和收尾核对，远程 HEAD 均为上述固定对象，PR 为 Open；PR API 的 base.sha 仍报告 `a236e53`。另外独立读取 master ref，已经前进到 `46a4b26855cc582b6387883260433c7badb1f2f9`。这两个身份分别记录；本轮仍按固定基准计算差异，没有改基或混入 PR #26。旧基准上的 CI 不等于与新 master 整合后的验证。

相对 Review 051 仅有四个文件增量：诊断测试、有效性回归、运行说明、Review 051 归档。相对共同基准共六个文件，另含原有工作流和 TestMain 登记。生产 Open、只读／权限／版本保护、分类与重试未变。未重新审计 M0／M1。

原工作区开始时为上述新 master，干净；054／055 报告均不存在。未切换或清理工作区。固定树解包到仓库外运行，独立测试和故障注入只改隔离副本。本文源码行号均指固定 HEAD。

## 二、BLOCKER

### B1-R：More() 不是顶层 EOF 检查，损坏结果仍可变成绿色诊断

**位置：**[m14_diag_test.go](https://github.com/XKHoshizora/specmetry/blob/d2f1d35a5d38564893d991b7c6d872b854e5e8b0/internal/store/sqlite/m14_diag_test.go#L412) 第 414～428 行；汇总调用链第 393、434～463、715～769 行。

解析器使用：

```go
dec.DisallowUnknownFields()
if bytes.ContainsRune(line, '\n') || dec.Decode(&o) != nil || dec.More() {
    // malformed
}
```

More 用来判断数组／对象内部是否还有元素，遇到 `]` 或 `}` 会返回 false，不能证明顶层 JSON 后已经 EOF。因此以下两份字节串都被接受：

```text
{"helper":"m14open","stage":"done","open_called":true,"committed":1}]
{"helper":"m14open","stage":"done","open_called":true,"committed":1}}
```

**独立复验：**

1. 在隔离副本新增纯解析测试，对合法原串及上述两个尾缀调用原 m14Parse，再调用原 m14CheckOpen。合法对照通过；两个非法串均满足 json.Valid=false，但解析错误和最终 Invalid 都为空。
2. 为验证不是只测辅助函数，在另一隔离副本仅修改 m14Print：真实辅助结果序列化之后附加一个 `]`；不改变 Open、实际提交、父进程解析、数据库检查或有效性判定。
3. 经原 TestM14OpenPath 运行 1 轮、2 个辅助进程、各 5 条事实。两个进程都真实 Open 并提交；辅助 stdout 都是损坏 JSON。父测试仍输出：

```text
planned_helpers=2, started_helpers=2, open_calls=2, opens_succeeded=2
facts_committed=10, invalid_count=0, valid=true
TestM14OpenPath PASS，进程 exit 0
```

这直接违反“结果必须恰好是一份合法 JSON；损坏结果使实验无效”的本轮修复契约。现有 corrupt 用例仅输出 not-json，能覆盖首个 Decode 失败，无法发现“首个对象合法、后部损坏”的路径。

**影响：**辅助输出已损坏时，诊断仍可声称实验有效。现有 CI 的完整结果可单独核对并保留，不因此推断它们发生过损坏；但工具的通用拒绝保证不能关闭。

**最小修正：**首个严格 Decode 成功后再次 Decode，并且必须明确得到 io.EOF；任何尾随 token、第二份 JSON 或语法错误均拒绝。保留当前未知字段和单行约束，增加合法／空结果／未知字段／第二对象／尾随 `]` 与 `}` 的对照，并让真实诊断父进程对尾缀故障报告 valid=false、非零退出。无需增加生产重试或重构存储层。

## 三、其他修复与 NON-BLOCKER 裁定

| 项目 | 本轮判断与依据 |
|---|---|
| 辅助进程启动／崩溃／非零退出 | m14Spawn 分开 Started 和 Invalid，Start、Wait、取消错误不会混入 SQLite 观测；现有真实子进程回归通过 |
| 准备失败／未调用 Open | setup、resolve、migrations、OpenCalled 都受检查，不能把准备失败当成有效 Open 样本 |
| 零工作量 | m14Int 明确范围为 1～上限，不再静默截断或接受零；控制组的 0 writer 是代码显式选择，不是接受用户零写入参数 |
| 探测／提交负载 | 每个有效 prober 至少一次；计数需相加一致；during commits 要求提交数和 overlap_ms 为正。没有提交或只有无效 writer 时不能绿色通过 |
| 真实 SQLite 失败 | 不强求每次提交成功。真实 Open 后由触发器拒绝所有写入，仍是有效观测，记录 write/code 1811；本轮独立对照通过 |
| N1：seq 下界 | m14Check 对非空要求 min=1、max=n，同时唯一、计数器与提交数一致；空表计数器为 0；独立四组回归通过 |
| N2：期限与回收 | helper 自身有 deadline context；父进程在 deadline+3s 杀仍未结束的直接子进程，Wait 并设置 3s WaitDelay；挂起用例在约 5.39s 完成拒绝，未靠等待原来的 20s 睡眠结束 |
| N3：阶段与计数 | 只有完整 query 包装才标只读查询；其他版本读取标 step unknown。probes、retries、probe_calls 和 planned／started／open_calls 分开，静态与回归均成立 |
| N4：保证边界 | 已区分目录准备、打开锁及批准的 WAL／SHM 例外；没有再称数据目录绝对零变化，数据库检查不冒充 payload 核对或失败 Open 分支的动态证明 |

**本轮没有另列新的 NON-BLOCKER。** 上述 N1～N4 可按其原范围关闭；这不关闭 B1-R。

overlap_ms 是首末调用区间的交集，不证明某次 probe 与某次 COMMIT 临界区相交，也不证明三个 writer 全程持续并发。说明已明确限制，不能把有该数值扩大为精确锁竞争时间线。

默认两个诊断仍必须 SPECMETRY_M14=1 才执行；常规有效性测试会在自己的隔离子进程中显式启用。工作流未改变生产或常规 CI，权限仍为 contents: read，参数有界。SQLite 错误观测与进程／工具故障分开这一方向正确；剩余问题仅是 B1-R 的解析完备性。

## 四、本轮独立运行及边界

平台：Windows/amd64、Go 1.27.1、SQLite 3.53.4、modernc.org/sqlite v1.60.1。

- 离线构建固定树的 SQLite 测试二进制。隔离树额外加入上述纯 JSON 测试；原诊断和有效性回归代码保持固定版本。
- 原有 TestValidityOfM14Diagnostics 的 18 个场景全部通过：3 个有效场景、15 个应使子诊断失败的场景，分别核查判定与结果记录；并通过 seq 缺口、阶段、区间交集测试。
- 未设置启用变量直接运行两个 TestM14，均 SKIP；这只证明默认不开启，不称为压力诊断通过。
- 纯解析对照发现两个非法尾缀仍被接受，测试按预期失败。
- 修改 m14Print 的隔离故障注入经真实诊断汇总复现错误 PASS，见 B1-R。
- 首次在受限沙箱运行涉及数据库的回归，统一在临时目录打开锁时 Access is denied，未到被测 helper；不计为有效覆盖，也不归因于 M-14／M-15。改在正常用户隔离环境运行后通过，没有改生产或降低断言。
- 最初一次命令参数及隔离编译工作目录配置错误均已更正，不作为代码失败或通过证据。

没有本机复现 264／1288，没有为了追求复现重复施压，没有运行 macOS／Linux 启用压力诊断或本机 Race Detector。Review 051 已建立的旧版崩溃／零负载反例作为既有独立证据复用；本轮未重跑作者 2347678 的全部桩阶段，因此其 15 项修复前统计仍标为作者证据。

## 五、分 SHA 的 CI 与观测核查

### 1. 固定 HEAD 的常规 CI

[run 39125973101](https://github.com/XKHoshizora/specmetry/actions/runs/39125973101)，Attempt 1，8/8 success。

八个 Job 日志实际检出 `eaf6a16c3d9edbe0761b7dbdd6a7c49b63fbffe9`；父提交依次为固定 base 和本轮 HEAD；内容树 `20dce350b100828d7617907db37687b79754c8cb` 与 HEAD 相同。三平台有效性回归均实际 RUN／PASS，没有 SKIP；直接诊断默认 SKIP。Ubuntu 按工作流启用 -race，有效性回归在该运行中也执行；这不是在 Linux 上跑过 60 秒／200 轮的诊断压力规模。

此前 e52d6f5 的 [39124917580](https://github.com/XKHoshizora/specmetry/actions/runs/39124917580) 也为 Attempt 1、8/8，检出 `f38857eba6cee7a82fd64b34479d7b2d48929166`；有效性回归三平台通过。e52d6f5 → d2f1d35 只有运行说明变化，测试代码与工作流相同。

保留 dce8b67 的 39124717515 lint/unparam 失败及随后推送取消测试的历史，不写成整个 PR 从未失败。其修正到 e52d6f5 只把各参数共同的下界固定为 1，没有据此修改生产行为。

### 2. 两次独立 push 诊断，不合并统计

分别读取两个 run 的元数据、两平台完整日志、实际 checkout 和三条唯一结果。Results 展示步骤再次打印相同结果，不重复计数。

| SHA／run | 平台 | 控制首轮探测 | 并发首轮／重试；错误 | 并发提交／overlap_ms | 真实 Open／事实／写失败 |
|---|---|---:|---|---|---|
| dce8b6784505227b28394243e1a4eb393e627f29／[39124713860](https://github.com/XKHoshizora/specmetry/actions/runs/39124713860) | macOS | 596452 | 390918／31；27×264、4×1288，重试均成功 | 213033／59863 | 600／60000／0 |
| 同 SHA／run | Windows | 77557 | 25139／0；无首轮错误 | 16484／59956 | 600／59900／1 次 code 5，1 轮 |
| e52d6f5797b10f5cb21b1f1c641b8bb76aac441b／[39124915738](https://github.com/XKHoshizora/specmetry/actions/runs/39124915738) | macOS | 372613 | 380181／20；17×264、3×1288，重试均成功 | 193228／59931 | 600／60000／0 |
| 同 SHA／run | Windows | 75574 | 16695／0；无首轮错误 | 9508／59955 | 600／59636／4 次 code 5，3 轮 |

两次均为 push、Attempt 1，各自两个 Job success；实际检出和结果 platform.SHA 均为对应完整 SHA，非 PR 合并快照。每次每平台三个结果均自报 valid=true、invalid_count=0，且记录了真实非零负载、全部 600 个 planned／started／open_calls 和数据库检查。

这四份具体日志未出现本轮非法尾缀，结果行可完整解析；因此其具体计数仍有证据价值。工具通用门禁未完备，不等于这些实际观测必须丢弃；反之，四份正常日志也不能关闭 B1-R。

dce8b67 与 e52d6f5 的诊断文件 blob 不同（参数下界重构），分别陈述；e52d6f5 与本轮 HEAD 的该 blob 均为 `34f9684e2cfb1678abe7bc27069e72d7d4c6fd99`。本轮没有在目标 HEAD 又做新的 CI 压力实验，不能把文档 CI当作新增诊断运行。

两平台均有 writer 以 SQLITE_BUSY 提前停止，有的从未提交。区间与总数能证明有负载及跨度交集，不证明全程多 writer 并发。所有 264／1288 仍来自辅助无打开锁探测，不能改称真实 Open 的同根因复现；真实 Open 的四组各 600 次均无打开失败。Windows 的 code 5 明确在 write 阶段、分类 internal，仍与 M-14 分开。

## 六、最终裁定与接手建议

- **B1：不能整体关闭，剩余 B1-R。**
- **N1～N4：原范围内已合理处理。**
- **PR #25：NO-GO。** 最小下一步是修复顶层 JSON EOF 校验及补尾缀父进程回归；不需重新做生产存储审计或反复压力诊断。
- **PRODUCT DECISION：**该修正不需要放宽产品契约；进一步 SQLite 插桩、错误映射或重试另行授权。
- **ACCEPTED RISK：**未新增。已有狭窄辅助文件例外不扩大；M-14／M-15 未解决。
- 与新 master 的整合和 CI 留待另行核验。本轮不改基、不解冲突、不合并、不授权 WP2.3。

报告完成后暂停。除新增本报告及同时受托的 Review 055 外，没有修改原仓库文件，未提交、推送或合并。

