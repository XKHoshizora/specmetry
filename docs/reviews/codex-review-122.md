# Codex Review 122 — PR #47 / N-121-1、N-121-2 定向复核

**结论：N-121-1、N-121-2 均可关闭。固定对象技术 GO，建议负责人决定合并；未发现新增 BLOCKER 或直接回退。** 沿用 Review 121 对第一批其余范围的技术 GO 和证据限定，不将本轮结论扩大为完整 Runner、第二批授权或 WP2.5／M2 验收。

本轮独立核对源码、归档、正式 CI 和八份实际 checkout。一次本机 Go 定向运行仍在创建 `.migrate.lock` 时遇到 Access is denied，未进入存储断言；不计为本机验证通过，也不绕过权限保护。两项修正的生产驱动动态证据来自本轮三平台 CI。

## 1. 固定身份与范围

开始核验：2026-10-31 23:45:01 +09:00。已读取当前 AGENTS.md、Review 121、处理记录 §154 和本轮完整差异。本地分支为 `feat/wp2.5-batch1`，开始时工作区及暂存区干净，没有覆盖其他会话内容。

| 项 | 独立核验值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #47](https://github.com/XKHoshizora/specmetry/pull/47)，open、未合并 |
| HEAD | `8523aa04c1609a0244e9d5bab44e725495887bcb` |
| HEAD 树 | `32a5899e83435f326c3c0bb38cdf80897ac6c862` |
| base／远程 master | `ad592c33306828e96b59f6e7be77dc4af8995bd9` |
| 上轮受审 HEAD | `4d5dabee790563fdb320d65c838ec17ee69384ce` |
| CI 合并快照 | `64791b2e2491c252f1b137a956864e1f21f37536` |
| 快照父提交，依次 | 上述 base、上述 HEAD |
| 快照树 | `32a5899e83435f326c3c0bb38cdf80897ac6c862`，与 HEAD 相同 |

相对上轮为七文件、374 行增加、32 行删除：`AGENTS.md`、`docs/implementation-plan.md`、`docs/review-resolution.md`、新增 Review 121 归档、`internal/store/sqlite/migrations/0009_check_runs.sql`、`internal/store/sqlite/checkruns_test.go`、`internal/checkrun/store_test.go`。

提交顺序为：

1. `4f38909e1fcc3ae67cc2f0e041c3058ff5f983d9`：单独归档 Review 121。
2. `208a6b09c51590151d35425bcf661bb40a87b2c1`：§154 的负责人决定与状态同步，先于两项修正。
3. `4c2969e83bb75375dd00e365a972f4f1e8389c1a`：NOT NULL 及相应测试。
4. `8523aa04c1609a0244e9d5bab44e725495887bcb`：映射测试增强。

Review 121 归档为 26839 字节，SHA-256 为 `3252c33da47f3c2d39a0641ec8ddbaefcbcecc8ad71eca0344907f30bc8b899c`，与上一轮交付、归档 blob 和当前文件一致。旧处理记录保留为字节前缀，只追加 §154；没有改写其他历史报告。§154 正确保留 Review 121 的 GO、本机限制及 factgen/legality 差额，只授权两项修正和验证，没有提前宣布关闭或授权合并。

忽略 SQL 注释后，生产差异严格只有 `id` 定义增加 `NOT NULL`。其他实现变化仅在上述两份测试中。冻结规范、Schema／示例、其他迁移、Write、分类／时间／qualifying、derive／ready、toderive、dbdump、工具链、依赖、workflow、预算及维护状态未变；没有命令注册或后续能力接入。`git diff --check` 通过。

## 2. N-121-1：关闭

### 2.1 实际约束与目标用例

当前 `0009_check_runs.sql:18` 为：

```sql
id TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 26)
```

保留原主键与长度约束，不使用 STRICT，不改变 ID 生成或其他列、外键、索引和触发器。`Write` 仍自行生成 ULID；不是把调用方提供 ID 的能力加入存储接口。

`checkruns_test.go:197`～201 先取得一份正常 `first` 行，再值复制为 `withoutID`，唯一差异是 `nullID=true`。`insertRunWith` 仅据此把第一个绑定参数改为真正的 SQL NULL，其余参数仍来自相同的行和既有合法默认值。

该行的 seq 尚未使用，项目、AC、worktree 已在同一测试的合法基础数据中建立，修订、hash、observed/off、result／exit／duration 均合法。拒绝断言要求错误包含 **`NOT NULL constraint failed: check_runs.id`**，不能由其他列的 CHECK、唯一性或外键错误冒充。随后同一 `first`、同一 seq 仅恢复正常 ID 即成功；既证明夹具其余部分可写，也没有用另一条绕过路径充当健康对照。

该检查在 `TestCheckRunsConstraints` 中是无条件执行的普通步骤，没有 SKIP／未命中分支。若 NULL 被接受，`refused` 会标记测试失败；若错误原因不符，也会失败。三平台上该测试均实际 PASS。

### 2.2 机制对照与直接回归

`TestCheckRunsNullCannotSlipThrough` 新增临时 `naive_key`，定义为没有显式 NOT NULL 的 `TEXT PRIMARY KEY CHECK(length(id)=26)`，并要求它接受 NULL。这仅固定朴素定义的 SQLite 行为，**不能单独证明实际 0009 拒绝了 NULL**；后一个结论由上一节实际表的目标用例和健康对照提供。

用例 helper 默认 `nullID=false`，保持其他现有插入行为。升级、正常 Write、旧修订保留、事务失败回滚、项目隔离、只追加及 purge 的原有断言仍存在，未被删除或放宽；相应三平台测试通过。没有发现增加 NOT NULL 对当前正常 ULID 写入或 v8→v9 升级造成直接回退，也没有借机处理全库主键或既有历史问题。

**裁定：N-121-1 在当前未合并的 0009 及其实际写入范围内关闭。** 本机未独立跑到目标 SQL 断言，不能把这项 CI 证据写成本机复现。

## 3. N-121-2：关闭

### 3.1 exactlyByID 的失败条件

`internal/checkrun/store_test.go:329` 的 helper 同时用于两处：第 294 行的 dump，和第 314 行的 `toderive.Facts(d).CheckRuns`。预期不是从被测映射的输出构造，而是来自已写入夹具的身份／seq 与独立写定的契约字段。

| 情形 | 必须失败的接点 |
|---|---|
| 缺失某个期望 ID | 遍历 want 时检查 got，不存在则 `t.Errorf`；数量减少另触发 Fatal |
| 重复某 ID | 在覆盖 map 前检查该 key 已存在，持续标记 `t.Errorf` |
| 额外 ID | 不属于 want 时立即 `t.Errorf`；数量变化另触发 Fatal |
| 数量相同但替换／重复 | missing、unexpected 或 duplicate 的检查仍生效，不能靠 len 相同通过 |
| 全空／多一行／少一行 | 数量检查 `t.Fatalf`，没有“遍历不到目标就成功”的出口 |

例如期望 `{a,b}`、实际 `[a,a]`，即使最终 map 的 a 被覆盖，第二个 a 已记录重复，b 也被报缺失；后续字段比较成功不会清除 `testing.T` 的失败状态。因此覆盖 map 不会恢复通过。

测试在构造期望 map 前另断言两次实际写入的 ID 不相等，避免预期集合自身因相同 key 缩成一项。输入顺序不作不必要限制，符合按 ID 比较的要求。以上是源码控制流证明；本轮未声称额外独立执行了作者的负向变异。

### 3.2 两行的完整字段与 NULL

dump 的 `wantDump` 分别写定两份完整 `factgen.CheckRunRow`，按固定 ID 取结果后 `reflect.DeepEqual`；转换的 `wantFacts` 分别写定两份完整 `derive.CheckRun`，同样整体比较。固定值来自夹具和契约，不调用 dbdump／toderive 来反算期望；随机 ID、事务分配的 seq 和首次登记的 worktree 身份取自写入回执是关联该对象所必需的信息，不是从映射结果自证。

第二行明确为 error/context_changed，保留开始 HEAD、clean 起点、unknown 终点、非空 kind；dump 中 InputsJSON／HeadEnd 为 nil，转换后 Inputs 为 nil、HeadEnd 为空字符串。整体比较已覆盖全部字段，另有显式 NULL／零值断言；nil 输入不会被空但非 nil 的 slice 混过，kind 也不能丢成 NULL。第一行的 JSON 输入元数据、版本、hash、HEAD、结果等继续完整核对。

**裁定：N-121-2 关闭。** 新增的是测试集合与字段断言，没有改变任何映射、推导或记录语义，不将该测试增强扩大成整个合法性工具的修复。

## 4. 独立执行与限制

先以绝对路径检查并调用已安装的 `C:\Users\hoshi\AppData\Local\mise\installs\go\1.27.1\bin\go.exe`，实际为 **Go 1.27.1、Windows/amd64**。没有调用 shim、安装或升级工具。设定 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off` 后，仅运行一次以下定向命令：

```text
go test -tags nowebui -count=1 -timeout 60s ./internal/store/sqlite ./internal/checkrun -run '^(TestCheckRunsConstraints|TestCheckRunsNullCannotSlipThrough|TestCheckRunsMigrationUpgradesV8|TestDumpMapsCheckRuns|TestWrite|TestWriteKeepsTheRevisionOfItsStart|TestWriteFailuresLeaveNothing)$'
```

**实际结果：退出 1。** 七个目标测试都在 Open 创建 `.migrate.lock` 时遇到 Access is denied，未进入业务断言；sqlite 包 0.120s、checkrun 包 0.117s。没有将其解释为修正失败，也没有将测试已编译或库打开失败计为断言通过。没有再试图以其他账户、权限放宽、替换锁或修改夹具绕过拒绝。

只读静态检查另确认归档 hash／旧记录前缀、受保护路径不变，以及移除注释后生产 SQL 只有 NOT NULL 的差异。未运行原型、平台机制、用户检查或整个第一批测试包的额外压力运行。测试命令已退出，普通进程列表未见 Go／checkrun／sqlite 测试进程仍运行；未终止任何未知对象。

作者报告的本机移除 NOT NULL、删除／重复／增加转换行及错误字段的未提交反向检查，只作为作者证据保留。本轮没有复现这些变异，不以反向检查文字代替实际源码的失败条件。Review 121 的独立纯函数、契约及内存 SQL 证据仍只适用于当时对象，不冒称本轮重新取得；本轮 changed test 的正常动态执行另见 CI。

## 5. CI 正式状态、实际 checkout 与结果归属

独立读取 [run 41204828884](https://github.com/XKHoshizora/specmetry/actions/runs/41204828884) 的正式元数据、八个作业及各自原始日志：pull_request、attempt 1、completed/success。所有作业在项目 checkout 后的 `git log -1 --format=%H` 均为 `64791b2e2491c252f1b137a956864e1f21f37536`。提交 API 的完整父提交及同树关系见 §1；不是用 job.head_sha 替代实际检出。

| 作业 | Job ID | 正式结论 |
|---|---:|---|
| Go lint | 124547678600 | success |
| Go tests / macOS | 124547678641 | success |
| Web | 124547678714 | success |
| Go tests / Windows | 124547678766 | success |
| Go tests / Ubuntu | 124547678844 | success |
| Build / Ubuntu | 124547893869 | success |
| Build / macOS | 124547893871 | success |
| Build / Windows | 124547893874 | success |

三个 Go 作业均有 27 个按包路径去重的 `ok` 包，无 FAIL。SKIP 逐名称比较与 Review 121 的 run 41199140442 原始日志相同：Ubuntu 15、macOS 7、Windows 10，不仅数量相同。新修改的目标测试没有 SKIP：

- `internal/store/sqlite`：`TestCheckRunsConstraints`、`TestCheckRunsNullCannotSlipThrough`、`TestCheckRunsMigrationUpgradesV8` 均 PASS，覆盖实际 NULL ID 拒绝／同一行健康对照、朴素主键机制对照及升级。
- `internal/checkrun`：`TestDumpMapsCheckRuns`、`TestWrite`、`TestWriteKeepsTheRevisionOfItsStart`、`TestWriteFailuresLeaveNothing` 均 PASS，覆盖增强映射与直接回归。
- 两个包各自的测试输出按结尾的 `ok <完整包路径>` 归属。特别是 `internal/checkrun.TestClassify` 与 `internal/gitx.TestClassify` 均存在，分别核对包块和子测试，未把 gitx 的同名 PASS 充作本批分类测试。

Ubuntu 普通 Go 测试实际带 race；八个 CrossUser 在普通步骤跳过、在独立 root 步骤实际 PASS。root 步骤不是 race 证据，macOS／Windows 也不称作 race 运行。三个测试作业实际 Go 均为 1.27.3，与本机独立运行分开记录。

| 本轮相关包时长 | Ubuntu race | macOS | Windows |
|---|---:|---:|---:|
| checkrun | 1.904 s | 0.173 s | 0.929 s |
| sqlite | 72.669 s | 20.876 s | 46.579 s |

这些只是该运行观测，不是上界或 M-16 关闭依据。常规 CI 不验证后续进程机制或 E2；没有触发新原型诊断。本轮没有重跑 CI；旧运行、首轮超时及维护观测按 Review 121 的证据身份保留。

## 6. 最终裁定与保留事项

| 项 | 裁定 |
|---|---|
| N-121-1 | 关闭：实际表的 NOT NULL 与长度／主键同时保留，目标拒绝和同一行健康对照有效 |
| N-121-2 | 关闭：两处集合检查及两行字段／NULL 比较消除原有空过 |
| 新 BLOCKER／直接回退 | 未发现 |
| 固定 PR | 技术 GO，建议负责人决定合并；不代为操作或产生合并授权 |
| Review 121 其余范围 | 沿用限定技术 GO，不重做第一批或后续机制审计 |

factgen/legality 的旧 context_changed 等价规则没有修正；在后续首次使用它验证 Runner 历史之前，仍须按计划 §10.7 另行提出工具修订。不能以本轮集合断言或绿色 CI 关闭这个差额。Review 121 的本机锁访问限制、作者首次本机全量超时、既有维护事项与历史未知继续保留。

下一步由负责人决定 PR #47 是否合并并安排相应核验；第二批及其后续工作没有因此获授权。没有修改冻结规范、代码或测试，没有提交、推送、合并或触发／重跑 CI；WP2.5／M2 未验收。

结束身份核验：2026-10-31 23:47:54 +09:00。远程 PR 仍为 open、未合并，HEAD/base 与开始时一致，远程 master 仍为固定 base；本地仍为 `feat/wp2.5-batch1`、HEAD `8523aa04c1609a0244e9d5bab44e725495887bcb`、树 `32a5899e83435f326c3c0bb38cdf80897ac6c862`。CI 仍为 attempt 1、completed/success。已跟踪文件和暂存区无变更，工作区仅新增本报告，保持未提交。完成后暂停，等待负责人决定。
