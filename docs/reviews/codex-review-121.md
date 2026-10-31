# Codex Review 121 — PR #47 / WP2.5 第一批实现

**结论：固定对象技术 GO，建议负责人决定合并。第一批已在授权范围内兑现契约、迁移、存储映射及纯函数交付，未发现 BLOCKER。** 这不是完整 Runner 的通过结论，也不授权后续批次、合并操作或 WP2.5／M2 验收。

**证据分层**：本轮独立通过契约测试、九项纯函数顶层测试及 135 次推导对照；独立的内存 SQLite 核对中，168 个 D14 组合与固定期望一致。Go 存储／迁移测试在本机被迁移锁的访问权限拒绝，未进入目标断言；其生产驱动下的动态结果来自已独立读取的三平台 CI，不冒称本机复现成功。直接 SQL 的 NULL 主键防线及映射测试的存在性断言列为非阻塞建议。

| 问题 | 裁定 |
|---|---|
| 授权范围及先后 | 成立：条件已生效，登记先于代码，契约／迁移先于实现，纯函数固定预期先提交 |
| D14 与兼容 | 成立：条件、允许边界、新写入／历史读取分开；未增加全局退出码上限 |
| 0009 与写入 | 成立；D14 的 NULL 行为、同项目外键、只追加、事务与版本保护可采纳，独立存储验证限制见 §7 |
| 纯函数与推导对照 | 成立；独立固定预期与独立实现对照分别提供证据，derive 语义未改 |
| 当前 CI 条件 | attempt 1 正式 success，8/8，实际 checkout 和同树关系成立 |
| 新 BLOCKER | 未发现 |
| 第一批整体／工作包验收 | 第一批可供负责人作合并决定；WP2.5／M2 未验收 |

## 1. 固定身份与审查范围

开始核验时间：2026-10-31 21:29:17 +09:00。读取当前 AGENTS.md；本地分支 `feat/wp2.5-batch1`，开始时工作区及暂存区干净。没有切换分支、覆盖或清理其他会话内容。

| 项 | 独立核验值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #47](https://github.com/XKHoshizora/specmetry/pull/47)，open、未合并 |
| HEAD | `4d5dabee790563fdb320d65c838ec17ee69384ce` |
| HEAD 树 | `428286e11b1b54401653d09bf69165398150ea5c` |
| base／远程 master | `ad592c33306828e96b59f6e7be77dc4af8995bd9` |
| CI 合并快照 | `bc7988a5cbf5c382fe10602eec0a1d428147ee51` |
| 快照父提交，依次 | 上述 base、上述 HEAD |
| 快照树 | `428286e11b1b54401653d09bf69165398150ea5c`，与 HEAD 相同 |

全部增量为 21 个文件，2308 行增加、7 行删除：四份状态／计划文档；`internal/checkrun` 的四份实现与六份测试；契约专项测试；迁移与迁移测试；dbdump；Schema、非法例子集及新的合法例子。逐份读取，沿写事务、worktree、迁移、墓碑清理、契约验证、toderive 与 derive 的直接调用链核对。

依据冻结提交 `f99d648`／技术基准 `f2b5c87`，按规范 §22 的优先关系，对照 §6.2、§7.2／§7.4、§10.1～10.3、§11.2、§19.10、data-model、验收约定，以及生产计划 §4.1／§10、Review 120 §4、处理记录 §152／§153。

冻结规范、数据模型、验收正文、工具链、依赖、workflow、原型及 derive／ready 实现均未变化。没有 CLI 注册、检查进程、全局闸门、平台管理或 E2 接线；没有修改维护事项或预算。旧处理记录完整保留为字节前缀，只追加 §153。

## 2. 授权、归档和实施顺序

Review 120 单独归档于 `7108e85baf96cdfc89fc88e707aa3308be41f616`，只有该报告文件，共 28573 字节，SHA-256：

`56ee4b7f440f935d15bd186ec15c486e9962843ecf9c01bc87be41b3fa38a3c2`

当前文件与归档 blob 及上一轮交付一致。之后的 `c9b4c432f2b7d5e71164e9deea6052052bf2f203` 仅落实 §152 的负责人决定、计划精化及状态：维持末批注册 CLI，第一批不改工具链／CI；N-120-1～3 的最迟点和限制准确保留，没有改变冻结语义。

PR #46 已合并为 base `ad592c3`，父提交依次为 `1d3f456b54214d2206360de7e81e761a7b65f2e2`、`c9b4c432f2b7d5e71164e9deea6052052bf2f203`；树 `7195eda32a9ef35eb6510c8a17be1157d91e9739` 与其最终 HEAD 相同。独立核验远程 PR、提交对象及 master push CI 41195787296 的正式成功和八份实际 checkout。因此 §152 的第一批条件授权确已生效。PR #46 最终常规 CI 41194799490 的正式元数据也为 attempt 1 success；本轮对合并后基线另读了完整八份日志，见 §8。

本 PR 的提交顺序：

| 提交 | 内容及裁定 |
|---|---|
| `ddb1fc74761bc1157a849c3dd76363ac7b1e7c04` | 仅实施登记／§153 与状态，早于任何生产变更 |
| `5a374b1378c6c8a05a3670ff0bd876d817ee7e55` | D14 Schema、示例及契约固定预期 |
| `f13e493a0753010d9d1d8d88221d888d545c212d` | 0009 与迁移测试 |
| `d123ba1a737cece9f8d8592f405f4206fd6a8dba` | 纯函数测试，尚无实现，按登记此时不能编译；这是提交顺序证据，不是一次已通过的红测试运行 |
| `54c5957b6e3719aed81b8a213ab1b86253c473dd` | 纯函数、存储写入、dump 映射 |
| `4d5dabee790563fdb320d65c838ec17ee69384ce` | 存储集成和推导对照 |

测试先行提交到实现提交之间，`classify_test.go` 只把 `ContextChanged(...)` 调用及显示名称改为 `Changed(...)`；固定预期未改。该命名避让同名 failure-kind 常量，不改变 §7.2 或覆盖。

## 3. D14 契约与合法性

`schemas/check-run.v1.schema.json` 的七项条件在每个 `runs.items` 内执行；既有 required、类型、枚举、additionalProperties 仍有效。因此不是只校验顶层或第一项，缺失字段也不能利用 `if` 的 properties 语义绕过 required。

| 冻结约束／允许边界 | 实现与证据 |
|---|---|
| pass → NULL kind、exit=0 | 对应条件及错误字段断言；NULL 退出码、非零与带 kind 均拒绝 |
| fail → nonzero_exit、exit≠0，可 NULL | `not const 0` 叠加原有 integer/null 类型；信号 NULL 合法，不会当作数值 0 |
| error → 指定 kind 集合 | 包含历史 `not_found`，不包含 NULL 或 nonzero_exit；新写入另由 SQL 排除 not_found |
| timeout／spawn_error／input_missing → NULL exit | 三个受约束 kind 分别有反例，spawn/input 的反例先满足 duration=0，避免由无关 duration 约束碰巧拒绝 |
| spawn_error／input_missing → duration=0 | 反例先满足 NULL exit，确实核对 duration 字段 |
| context_changed → qualifying=false | exit=0、NULL、256 均可合法，错误反例指向 qualifying |
| qualifying → 非 NULL head_start、clean start | NULL／dirty／unknown 的字段级拒绝；不合格的 pass 与合格的 error 仍可合法 |

`TestCheckRunD14` 的预期来自固定条文，拒绝时检查目标 JSON pointer；`TestInvalidCasesFail` 先验证未改的 base 合法，再应用 patch，并检查 invalid_input 与非空 details。SCH-191～208 覆盖 D14 的反例，包括 pass+timeout+0；其中该冲突可能同时触发多个相关条件，不要求只产生一个错误，专项测试确认所需字段已被拒绝。

新增合法例子包含八个执行项：合格 pass、不合格 pass、非零 fail、信号 fail、spawn_error、input_missing、context_changed+0 与 context_changed+NULL。现有例子仍通过；专项测试另保留 not_found 读取与 Windows `3221225477`。没有对 Windows 执行 macOS 低八位规范化，没有全局 255 上限，也没有把所有 error 强制变为不合格。

Schema 不具有 head_end，不能单独表达完整 qualifying 或 D19 时间区间；对应业务规则由下文纯函数及存储对照覆盖，没有被 Schema 通过替代。

## 4. 0009、约束与升级

`internal/store/sqlite/migrations/0009_check_runs.sql` 与冻结 §10.1 的列对应：ID／seq／项目／AC 与修订、配置哈希、输入记录、worktree／workdir、observed/off 与隔离空列、起止 HEAD／tree、结果／kind／退出码／duration、通道／session、展示时间。没有保存命令、输出、环境或额外运行时错误文本。

项目、AC 与 worktree 的引用使用延迟外键；AC／worktree 是含 project_id 的复合外键。未复制 0008 的当前修订／活跃状态写入限制。新增索引为 `(project_id, criterion_id, seq)`；seq 表内唯一，由存储接口的事务内全局 NextSeq 分配。

### 4.1 三值逻辑与其他约束

结果的双向约束使用确定布尔值的 `failure_kind IS NULL`／`IS 'nonzero_exit'`，pass 使用 `exit_code IS 0`，fail 使用 `exit_code IS NOT 0`。nullable kind 的其他条件先显式允许 NULL，再判断受约束 kind。就 D14 的结果组合而言，NULL 不会使非法行从 CHECK 的未知值出口通过。

独立用 Python 3.14.1 内置 SQLite 3.50.4，在内存中执行本仓库 0001～0009 原始迁移，复用已有存储测试的基础项目／AC／worktree 数据，再按冻结规则独立计算期望：3 类 result × 7 个 kind（含 NULL、not_found）× 4 个 exit（NULL、0、1、3221225477）× 2 个 duration（0、1），共 168 组，20 组允许，零差异。每组 savepoint 回滚，数据库最后关闭；这不是 modernc 驱动、升级文件或 Go 写入接口的复现。

该夹具第一次因把有循环延迟外键的基础数据按自动提交执行而失败，尚未到达目标检查；改为与现有 Go 测试相同的一次事务后才得到上述有效结果。首次无效准备不计作产品失败或有效检出。

observed、off、隔离 backend/fallback 为 NULL、tree／channel 枚举、修订≥1、哈希／HEAD 格式及输入 JSON 数组均有明确约束和 CI 正反例。没有把一项 SQL 类型亲和性当作完整业务输入验证；内部 `Run` 的类型及后续预检仍各负其责。NULL 主键的额外防线见 N-121-1。

### 4.2 只追加、清理与版本

迁移文件包含 UPDATE／DELETE 触发器的实际定义，测试与 `AppendOnlyTriggers` 生成内容比较；REPLACE 按 ID／seq、upsert 的更新分支均由触发器拒绝。`DO NOTHING` 不改变事实，不等同于可更新事实。既有墓碑机制在维护事务内发现所有含 project_id 的表，延迟核对外键、删除对应项目并完成整理；测试确认清理 A 不删 B，没有为新表加额外删除豁免。

`TestCheckRunsMigrationUpgradesV8` 明确从八项迁移创建旧库，升级到九项、保留原数据、生成备份、新表可写、min_binary_version 仍空；再用只认识 v8 的打开选项拒绝 v9，并核对文件不变。通用的迁移失败回滚、旧版本和 min_binary_version 保护仍在三平台套件中执行；不将通用回滚测试说成特意模拟了 0009 每个 DDL 点的原生故障。

**不增加 min_binary_version 的决定可采纳**：新增表不改变旧表，当前 schema guard 已使 v8 程序拒绝 v9，写锁后的 checkWritable 也重新检查版本；额外 min-binary 门槛不是为本批凭空新增的产品政策。将来升级或旧进程并存仍使用现有保护，不新增重试或错误映射。

## 5. 存储、事务及映射

`internal/checkrun/store.go:70` 的 `Write` 接受调用者的 `*sql.Tx`，实际顺序为 EnsureWorktree → 输入按 path 排序并编码 → NextSeq → INSERT → 同事务读回。API 注释明确任何错误由调用者回滚；函数返回 Row 不代表外层 Commit 已成功。

`DB.WriteTx` 在写锁下先检查版本与连接保护，回调错误回滚，成功才提交；worktree 首次登记、自己的 seq、执行 seq 和事实插入在同一事务内。既有 worktree 更新 last_seen_at 而不再分配登记 seq；空批次的零写入路径属于后续批次，不能将此函数误用于没有事实要写的已有 worktree。

| 场景 | 实际检查与证据范围 |
|---|---|
| 首次 worktree／再次执行／主 worktree | `TestWrite` 检查登记 seq、执行 seq、后续每次一个 seq、字段读回和 NULL／排序 JSON |
| 基准之后修订／AC 退休 | `TestWriteKeepsTheRevisionOfItsStart` 通过同一接口保留开始时两种修订与 hash，不按当前版本拒写 |
| 首登失败 | 测试数据库真实 BEFORE INSERT trigger 拒绝；行、worktree、全局 counter 不变 |
| 事实 CHECK 失败 | 同一接口 pass+exit1，被 CHECK 拒绝；首登与已分配 seq 同时回滚 |
| 延迟外键在 Commit 失败 | 外项目 AC 经过 INSERT 后被 Commit 拒绝，所有状态不变 |
| 调用者主动回滚 | 同一接口成功后回调返回错误，行与 seq 不留下；不是单独造一个不经 Write 的回滚路径 |
| 健康恢复 | 移除首登拒绝 trigger 后使用相同 Write，得到一项 worktree、一项事实及两个 seq |

上述存储测试在三平台 CI 中实际通过。本机未越过锁文件创建阶段，不能冒称独立验证了 Commit／备份／文件清理。

dbdump 新增明确的 project_id 筛选 SELECT，映射冻结 §10.2 所列全部推导输入字段；不读 exit_code／duration 等展示字段，因为 derive 本来不消费它们。NULL 字段保留指针空值，inputs_json 保留原文本，不过滤 cutoff、不选“当前行”、不计算 qualifying。toderive 仍逐行转换，derive 实现未改；不存在为使新规则通过而修改已验收推导语义的变更。映射测试的小范围增强见 N-121-2。

存储接口依赖调用方已经取得并验证的项目、工作目录、配置和输入元数据；数据权限、worktree 路径密钥扫描及执行前固定配置继续由后续编排负责。这里不要求第一批先实现第二批，也不能由字段类型正确推导出隐私预检已经完成。

## 6. 纯函数、观察边界与固定期望

### 6.1 分类和合法序列

`Classify` 在校验后按 F1/D19 → 写前取消 → 恢复异常 A12 → D6／上下文变化 → input_missing → spawn_error → timeout → 正常 0／非零／信号依次裁定。A12 不读结束上下文，D8 在树已确认时仍优先；F1 叠加取消继续尝试 D19。这里计算的是当前项记录，不声称覆盖尚未实现的所有 CLI 退出／持久化叠加。

核对的代表序列包括：

| 合法状态 | 结果 |
|---|---|
| 退出 0，但上下文变化 | error/context_changed，保留 0，不合格 |
| 超时、树确认结束、上下文稳定 | error/timeout，exit=NULL；仍按公式可合格 |
| 超时再叠加 F1 | D19/context_changed；可靠取得的主进程数值保留，不因原 timeout 强制丢弃 |
| 主进程结束状态已知，后代未确认 | 同上；已知主进程不证明整树结束 |
| D6：同一 HEAD、开始 dirty、结束读取失败 | error/context_changed，保留读取到的部分，不要求“确实发生变化” |
| 没有进程的 input_missing／spawn_error，后有上下文变化／D6 | 优先 context_changed，duration=0，exit=NULL |
| 恢复异常 A6／A12，有或没有退出状态 | 保留已知数值或 NULL；不反推实际执行／开始时刻；结束上下文未知 |
| 树确认且写前取消，A12 亦适用 | Record=false；不能误写一项普通结果 |

`Observation.check` 拒绝无进程却有退出状态／进程超时、带信号却未取得状态、D19/A12 声称已读结束上下文等矛盾。正常 Started、树已确认、既无 timeout／cancel 又没取得退出状态，不构成已描述的 S2 结果，返回 ErrObservation 合理。已取消且树已确认允许 End 尚未读取；A12、F1 无退出码的合法状态也保留。未发现它拒绝本批所需的冻结合法序列。

这不是对任意调用方输入的完整认证：后续编排必须如实构造这些知识维度，不能用纯函数接受输入来证明实际 API 或进程树已经成立。

### 6.2 时间、可靠退出码、qualifying 与 D9

`Timing` 分开正常启动、恢复异常与无进程：正常已取得退出状态取 t_main−t_start；否则取 t_S3−t_start 并标 Interval；恢复异常始终取 t_S3−t_anom；无进程取 item 开始且 duration=0。finished_at 是 Attempt 的 Unix 毫秒，不是实际退出时刻。没有按 S3 前后丢弃退出码，也没有把定义区间变回运行时间的界限；事件顺序仍须由 Runner 持有，不能从相等读数推断因果。

`Qualifying` 检查非空且相等 HEAD、两端 clean、kind 非 context_changed、MVP observed；不以 result 或退出码代替公式。`TargetSnapshot` 仅汇总每项 head_start 是否为 S，空批次则比较命令开始 HEAD；没有 S 时为 false，不影响逐条 qualifying 或推导。

固定预期分别由分类表、上下文表、时间表、qualifying 和 D9 的允许／拒绝边界及 §19.10 的 16 行记录状态给出。九项纯函数顶层测试独立运行通过。`TestTruthTable` 同时核对 result、kind、数值／NULL、duration、Interval、结束上下文与 qualifying，并非仅比较两个实现。

135 次 derive 对照另行验证 WorkspaceRun.Qualifying 和 SnapshotRun 是否对应。它包含 3×5×7 的正常／上下文构造及重复排列的 F1/A12，不是 135 个互不重复的机制场景，更不是全部可能历史的穷举。两个实现不共用公式；独立固定预期和推导一致性分别成立，不能仅靠其中一项自证。

## 7. 本轮独立执行、失败和限制

环境为 Windows/amd64；直接调用已有 Go 1.27.1，不是 CI 的 1.27.3。没有 E2 或平台机制代码，本批不需要为运行这些测试升级工具链。直接调用时设置 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`，未下载依赖。

| 次序／检查 | 实际结果及解释 |
|---|---|
| 默认 `go` 入口准备 | 命中 mise shim，意外触发用户级版本解析／自动安装尝试，网络离线且日志为 installed 0，未成功安装或升级；测试未开始。随后不再用该 shim |
| 三包定向 Go 测试 | `go test -tags nowebui -count=1 -timeout 90s ./internal/checkrun ./internal/contract ./internal/store/sqlite -run '<本批及直接关联测试名单>'`；contract 通过（0.401s），checkrun 的四项存储测试及 sqlite 的锁相关测试在 `.migrate.lock` 创建时 Access is denied，命令非零。未进入这些测试的业务断言，不确认产品根因 |
| 一次分离输出的复验调用 | 没有保留完整结束回执，不计作额外独立成功。没有据无输出判通过 |
| 取得明确回执的纯函数复验 | `go test -tags nowebui -count=1 -timeout 60s -v ./internal/checkrun -run '^(TestClassify\|TestClassifyRefusesInconsistentObservations\|TestContextChanged\|TestTiming\|TestTimingRefusesInconsistentEvents\|TestTruthTable\|TestQualifying\|TestTargetSnapshot\|TestQualifyingMatchesTheDeriveEngine)$'`；九项顶层 PASS，135 rows compared，包 0.075s、退出 0。此处竖线在 Markdown 单元格中作转义，实际使用正则 alternation |
| 原始 SQL 静态／内存核对 | Python 3.14.1／SQLite 3.50.4，加载原迁移、外键与 recursive_triggers 开启；第一次基础数据事务夹具无效，修正后 168 组零差异，另取得 NULL 主键观测。不是 Go 驱动存储集成通过 |
| 范围／归档／差异检查 | `git diff --check`、旧处理记录前缀、Review 120 blob／字节 hash、冻结文件与受保护目录不变均成立 |

没有解除权限检查、以其他账户运行、修改测试夹具或申请绕过环境拒绝。没有执行用户项目检查、原型、平台机制实验或本机整包测试，也没有复做作者的未提交变异。只读查询 CIM 进程信息被环境拒绝；随后普通进程列表未见 Go／checkrun／contract／sqlite 测试进程仍运行，未终止任何未知进程。内存数据库已关闭；不以测试退出证明任何未执行的生产清理保证。

作者的未提交反向检查（移除 Schema 条件、naive NULL CHECK、分类／时间／qualifying 变异）仅为作者证据，本轮未独立复现。作者本机的首次全量尝试有 gitx 的 10 分钟超时，随后 gitx 单跑 553.9s、CLI 单跑 929.0s（其本机命令使用 30m），以及各包单独通过，不能合写成一次在原整包预算内全量通过。CI 的标准预算未修改；本机慢运行的原因和 M-16 根因没有据此确认或关闭。

## 8. CI 正式状态、检出与计数

### 8.1 PR #47

独立读取 [run 41199140442](https://github.com/XKHoshizora/specmetry/actions/runs/41199140442)：pull_request、attempt 1、completed/success；八份作业原始日志均在项目 checkout 后打印 `bc7988a5cbf5c382fe10602eec0a1d428147ee51`。不是仅引用 job.head_sha；完整父提交、树见 §1。

| 作业 | Job ID | 正式结果 |
|---|---:|---|
| Go lint | 124531294132 | success |
| Go tests / Ubuntu | 124531294291 | success |
| Go tests / Windows | 124531294296 | success |
| Web | 124531294298 | success |
| Go tests / macOS | 124531294323 | success |
| Build / Windows | 124531517769 | success |
| Build / macOS | 124531517778 | success |
| Build / Ubuntu | 124531517820 | success |

各 Go 作业 27 个去重的 `ok` 包，比基线多 `internal/checkrun` 一个。17 个新增顶层测试均实际 PASS；既有合法／非法例子测试新增了输入，故作者“19 个新增或扩展测试”应按 17＋2 理解，不是 19 个新函数。日志另列 135 次推导对照、46 个合法例子、208 个非法例子；不把子测试或重复输出再计为新测试。

逐测试名比较 SKIP 集合，与 master 基线完全一致：Ubuntu 15、macOS 7、Windows 10；没有新增本批跳过或 FAIL。Ubuntu 普通步骤使用 race，八个 CrossUser 在该步骤跳过、在单独编译并以 root 运行的步骤中 PASS；root 步骤不宣称含 race。macOS 为 arm64；第一批没有 E2，实现两架构生产测试仍属于后续批次。

| 新增／相关包时长 | Ubuntu race | macOS | Windows |
|---|---:|---:|---:|
| checkrun | 2.089 s | 0.226 s | 1.158 s |
| sqlite | 72.477 s | 27.213 s | 46.484 s |
| contract | 2.490 s | 0.590 s | 0.577 s |
| CLI（未改） | 391.671 s | 153.412 s | 409.071 s |

这些是观测值，不是上界。实际 Go 为 1.27.3，不能由当前浮动选择恰好取得该值认定 N-120-1 的未来固定配置已经实现。本次没有原型诊断；常规 CI 不证明后续平台机制或完整 Runner 成立。

### 8.2 合并基线与授权生效

[master run 41195787296](https://github.com/XKHoshizora/specmetry/actions/runs/41195787296) 是 push、attempt 1、completed/success，八份原始日志实际 checkout 均为 `ad592c33306828e96b59f6e7be77dc4af8995bd9`：

- Go lint：124521546602；
- Go tests Windows／Ubuntu／macOS：124521546752、124521546765、124521546771；
- Web：124521546776；
- Build Ubuntu／Windows／macOS：124521696060、124521696078、124521696084。

三个 Go 作业各 26 个包；SKIP 集合与本 PR 相同，无 FAIL，Ubuntu root 八项通过。正式成功及实际检出支持“第一批授权条件已生效”，不等于 PR #47 已获准合并。

## 9. 非阻塞项与后续阶段

### N-121-1：NULL 主键的数据库防线

位置：`internal/store/sqlite/migrations/0009_check_runs.sql:16`。`id TEXT PRIMARY KEY CHECK(length(id)=26)` 在普通 SQLite rowid 表上没有隐含 NOT NULL，length(NULL) 的 CHECK 也不是拒绝。本轮在完整迁移及合法外键夹具下把唯一差异设为 id=NULL，直接 SQL 插入成功；[SQLite 官方说明](https://www.sqlite.org/lang_createtable.html#the_primary_key)与此一致。

这是确定的数据库防线缺口，但**不是当前 Write 的可达输入**：`store.go` 自行调用 ULID 生成器，调用方不能提供 ID，生成器固定返回 26 字符字符串；没有外部导入或原始 SQL 接口在本 PR 暴露这一能力。因此不据此断言正常写入会丢失身份，也不重开全库历史设计。建议显式 NOT NULL，并加直接 NULL ID 拒绝与正常写入对照；不要求改成 STRICT、增加新产品政策或扩大本轮修复范围。此建议不阻塞当前第一批合并，由负责人决定是否纳入收尾。

### N-121-2：映射测试可显式断言转换结果存在

位置：`internal/checkrun/store_test.go:299` 起的 toderive 比较循环。dump 部分已经核对两行及字段；转换后只在碰到第一行 ID 时比较，没有在本测试中显式断言转换后的行数、该 ID 必定出现及第二行全部字段。当前 toderive 源码逐行 append，135 次单行对照也会拒绝丢失对应行，未发现当前丢行缺陷；但该多行集成测试可改为按 ID 建表、核对完整期望及数量，避免单个循环的空过。属于覆盖增强，不是新 BLOCKER。

### factgen/legality：已披露的测试工具差额

计划 §10.7 的描述准确：其旧规则把 context_changed 与确实观察到的变化作双向等价。冻结 D6、D19、A12 允许“不能证明稳定”也使用该 kind；非 git 的 D19、同 HEAD 且 dirty 起点的 D6 可触发误拒绝。

本批没有修改该工具、降低生产校验或伪造合法历史。直接 toderive／derive 对照仍有价值，D14、原始 SQL、写事务、固定分类／时间／qualifying 各有独立测试，不依赖绕过 legality 来使非法 SQL 通过。但 135 次对照不能声称已经验证完整历史合法性或全部输入元数据关系。

最迟在后续批次首次用 legality 校验 Runner 写出的历史之前，提出针对已批准 D6／D19／A12 的工具修订并审查；须保留真正上下文变化的正反例、非 context_changed 的非法历史及其他原有合法性约束。不是要求现在重构 WP2.1／WP2.2，也不授权直接放宽或忽略整条规则。

## 10. 最终意见与结束核验

**第一批授权交付完整，技术 GO，建议负责人决定合并固定对象。** 当前证据足以支持第一批的契约与纯函数、生产驱动下的存储／迁移 CI，以及本轮明确限定的独立验证；不覆盖进程、宿主禁用、闸门、取消事务、平台 E2 或完整生产 Runner。

后续第一项工作应是本 PR 的负责人合并决定及相应核验，不自动开始第二批。后续批次的快照读取、隐私预检、空批次、C-9、固定工具链、解释器与两个架构生产测试继续按计划的最迟点另行授权／审查。M-1、M-14～M-17、N-087-1 和历史未知不因本轮通过消失。

本轮只新增本报告；没有修改受审代码／测试或冻结规范，没有触发／重跑 CI，没有提交、推送或合并。WP2.5／M2 未验收。

结束身份核验：2026-10-31 21:38:19 +09:00。远程 PR 仍为 open、未合并，HEAD/base、master 与开始时一致；本地仍为 `feat/wp2.5-batch1`、HEAD `4d5dabee790563fdb320d65c838ec17ee69384ce`、树 `428286e11b1b54401653d09bf69165398150ea5c`。CI 仍为 attempt 1、completed/success。已跟踪文件及暂存区无变化，工作区仅新增本报告，保持未提交。完成后暂停，等待负责人决定。
