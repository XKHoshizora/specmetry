# Codex Review 067：PR #30 WP2.4 Claim 实现独立审查

## 一、结论

**NO-GO。发现 2 项 BLOCKER，PR #30 当前不建议合并，尚不足以支持 WP2.4 本工作包范围的交付验收。**

1. **B1：损坏对象落入非提交／歧义分支时，未做完整性检查，被当作普通 `resolved=false` 成功提交。** 已在 Windows Git 2.56 和 WSL Git 2.43 独立复现，并通过公开 CLI 执行入口确认新增批次、Claim 和 seq。这违反已冻结的 E4／V-C06，不是要求重新决定歧义前缀的产品行为。
2. **B2：默认文本模式跳过提交前回执 Schema 校验。** 使用本 PR 已有的回执故障注入钩子，同一个非法回执在 JSON 模式被拒绝并回滚，文本模式却成功提交、打印错误的 seq。这是冻结规范要求的防线缺失；本报告不声称普通输入可以直接修改内部回执。

未发现需要重新决定的本轮产品取舍。版本优先、D2/D3、正常回执与落库对应、manual 通道、固定 HEAD 文件检查、提交后输出故障及迁移的主要约束已有有效实现和测试证据，不能用这两项发现否定所有已通过路径；同样，CI 全绿和双方一致的测试不能豁免这两项缺口。

最终 CI 已正式 completed/success、attempt 1、8/8。**合并检查成功与业务技术审查 NO-GO 分开成立。**

## 二、固定身份与范围

- 日期：2026-10-21，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#30](https://github.com/XKHoshizora/specmetry/pull/30)。
- base：`d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`。
- HEAD：`69d2f86bb1c9f2b3f842e127576ef04b7d886d7a`。
- HEAD tree：`e828b9024862e6cfdeb9c7d75cde281c2e97554d`。
- 开始及收尾远程核查均与指定 HEAD/base 相同，PR Open、未合并，`auto_merge = null`。没有自动纳入其他提交。
- 原工作区起始干净，报告文件起始不存在。按固定对象读取，并在仓库外导出隔离副本；没有切换工作区或修改实现。
- 已读取适用的根 [AGENTS.md](../../AGENTS.md)、冻结的 [WP2.4 规范](../wp2.4-claim.md)、相关模型／架构／验收及两个 Schema，并完整追溯 [Review 065](codex-review-065.md)、[Review 066](codex-review-066.md) 的实现要求。固定树中未发现下级 AGENTS.md。

本轮是首次实现审查，覆盖整个 base→HEAD 的 21 文件、4060 行新增／16 行删除：

| 范围 | 文件 |
|---|---|
| Claim 库 | [claim.go](../../internal/claim/claim.go)、[claim_test.go](../../internal/claim/claim_test.go)、[receipt.go](../../internal/claim/receipt.go)、[refs.go](../../internal/claim/refs.go)、[store.go](../../internal/claim/store.go) |
| CLI | [claim.go](../../internal/cli/claim.go)、[root.go](../../internal/cli/root.go)、[claim_test.go](../../internal/cli/claim_test.go)、[claim_manual_test.go](../../internal/cli/claim_manual_test.go)、[claim_terminal_test.go](../../internal/cli/claim_terminal_test.go)、[claim_realgit_test.go](../../internal/cli/claim_realgit_test.go)、[claim_derive_test.go](../../internal/cli/claim_derive_test.go) |
| 提交解析 | [resolve.go](../../internal/gitx/resolve.go)、[resolve_test.go](../../internal/gitx/resolve_test.go) |
| 共享引用规则 | [proposal.go](../../internal/proposal/proposal.go)、[coderef.go](../../internal/proposal/coderef.go)、[coderef_test.go](../../internal/proposal/coderef_test.go) |
| 数据库 | [0008_claims.sql](../../internal/store/sqlite/migrations/0008_claims.sql)、[claims_test.go](../../internal/store/sqlite/claims_test.go) |
| 测试转储与说明 | [dbdump.go](../../internal/testutil/dbdump/dbdump.go)、[development.md](../development.md) |

提交顺序：

1. `e45171ea1f54eabec903fb325b855eaae27858e8`：迁移 0008。
2. `186b6966612245de27a4ad3f58e311504d7905d6`：ResolveCommits。
3. `3eb7b29490671015939d16a038a6ef1d73ce4b2c`：固定 HEAD 与共享静态规则。
4. `cb224b1219a84e52ceb4b2580d446d8a9128d070`：Claim 库、CLI 与测试。
5. `09596fe95a7a90c6a92c53b741cd0a0d022d50ad`：首轮 lint 修正。
6. `69d2f86bb1c9f2b3f842e127576ef04b7d886d7a`：终端测试采用非 Git 项目。

旧迁移、冻结推导包、Schema、工作流、SQLite Open／错误分类／重试和 WP2.3 私有临时区实现没有变化。Proposal 的修改是提取原有 Code Ref 静态规则、增加显式 HEAD 接口和调整通用错误文字，未发现借此改变原有提案业务。没有提前实现 WP2.5、M3 会话或 status/gaps 命令。

## 三、BLOCKER

### B1：完整对象校验只覆盖 resolved=true 候选，损坏对象可被普通 false 掩盖

**位置：**

- [resolve.go:123](../../internal/gitx/resolve.go#L123)：先设置 `got[p] = len(ids) == 1 && types[ids[0]] == "commit"`。
- [resolve.go:124](../../internal/gitx/resolve.go#L124)：只有 `got[p]` 为 true 的对象加入 `read`。
- [resolve.go:128](../../internal/gitx/resolve.go#L128)：完整读取并重新计算对象 ID 仅遍历 `read`。
- [resolve.go:147](../../internal/gitx/resolve.go#L147)：后续存储检查不会完整解压、校验这些被跳过的 loose 对象正文。
- 可达调用：[claim/refs.go:105](../../internal/claim/refs.go#L105) → CLI 写事务。

**规则依据：** [wp2.4-claim.md:148](../wp2.4-claim.md#L148)、[data-model.md:1246](../data-model.md#L1246) 的 V-C06：真正不存在、非提交或真实歧义可标记 false；**对象损坏、读取失败或不能可靠分类必须 exit 6，整个 Claim 不写入**。E4 已批准，不需要再讨论是否拒绝正常歧义。

**独立反例 A：类型头可读、正文损坏的 blob。**

1. 在隔离普通 Git 仓库中，用 Git 写入正文为 `independent blob body\n` 的正常 blob，得到完整 ID `d032dba1959319bde023699cbc25cc2620c72717`。
2. 正对照：ResolveCommits 对此完整 ID 返回 false，无错误，符合“非提交”的规则。
3. 只替换该对象文件内容为合法 zlib 流，其解压内容是 `blob 500\0short`。对象声明 500 字节却没有相应正文，且与对象文件名的 SHA 不符；这不是另一个正常 blob。Git 仍可从头部报告 blob／500。
4. 再调用 ResolveCommits，实际得到上述 ID → false、`err=nil`、exit 0。对象库前后内容及 mtime 对照不变，证明读取没有修复它，也不是测试先把损坏消除了。

**公开 CLI 实际影响：** 在已初始化、基准 v1 有 active `AC-001-1` 的同一隔离项目中，提交合法 sync Claim，`refs.commits` 指向这个损坏对象。经公开 Execute 命令入口、真实 Git 和真实 SQLite 执行：

```text
期望：exit 6 / environment；没有本次批次、Claim 或额外 seq
实际：exit 0；claim-receipt.v1 的 resolved=false
      seq 6 -> 7；批次总数 1 -> 2；新增对应 Claim
```

第一批是正常 blob 的正对照，第二批是损坏后本应拒绝的提交。全部事实表快照确认第二次确实持久化，不是只打印了一份成功回执。

**独立反例 B：真实歧义中的损坏 commit 候选。**

先构造两个实际 ID 前七位同为 `585bc06` 的 commit，正常对照为 false；随后将其中一个 loose commit 改成头部仍可读、正文不完整的对象。由于 `len(ids) > 1`，两个候选均不进入完整读取，仍返回 false、nil。现有“损坏 blob 位于歧义前缀后”的用例用 `not zlib` 破坏头部，会在 batch-check 阶段被拒绝，不能覆盖本反例。

上述两个库层反例在 Windows Git `2.56.0.windows.2` 与 WSL Ubuntu Git `2.43.0` 均独立复现；公开 CLI 的新增事实也在这两种环境复现。测试只使用临时对象，没有网络或外部 Helper 实验。

**影响与界限：** 这不是把 false 宣称为验收通过，但它仍把本应使整批拒绝的对象损坏当成正常未知引用，违反冻结的失败和无写入契约。当前未发现 unique commit 的既有全量哈希检查被绕过；缺口在该检查被条件分支跳过。

**最小修正：**

- 在发布任何 true/false 结果之前，对前缀枚举出的、参与分类的对象建立足够的完整可读性证明；按实际类型、声明长度、完整字节和重新计算的对象 ID 检查，不能只校验将返回 true 的 commit。
- 保留合法非提交／真实歧义的 false，不改为一律拒绝，也不解析原始 stderr 猜测分类。
- 继续采用受控对象入口、PD-26、流式读取、不保存对象正文和失败时无部分结果。
- 补上述两种头部可读的损坏用例，以及公开 CLI 的 exit 6、全部业务事实／seq 不变正反对照。

### B2：文本成功路径绕过冻结的提交前回执校验

**位置：** [internal/cli/claim.go:220](../../internal/cli/claim.go#L220)～228；调用点 [158](../../internal/cli/claim.go#L158)。

`claimOutput` 在 `!asJSON` 时立即返回 `claimText(r)`，因此只有 `--json` 才加载并执行 `claim-receipt.v1` 校验。事务随后可以 COMMIT。

**规则依据：** [wp2.4-claim.md:93](../wp2.4-claim.md#L93) 的写事务第10步、[§9.3](../wp2.4-claim.md#L205)、§15 的“回执校验失败的注入”均要求从刚写入行构造回执，提交前验证，失败回滚，包括首次 worktree 登记。这一要求没有限定为用户选择 JSON 时才有效。文本输出方式不会取消事务防线。

**独立精确验证：** 仅在隔离副本使用本 PR 已存在的 `testHookClaimReceipt`，与 [TestClaimRegistersTheWorktreeOrNothing:859](../../internal/cli/claim_test.go#L859) 相同，将回执 `Seq` 改为 0。没有修改生产函数、Schema 或事务代码。

| 同一非法回执 | 期望 | 实际 |
|---|---|---|
| `claim ... --json` | exit 1，回滚 | 正确：`/seq` 不满足最小值，业务快照不变 |
| `claim ...` | exit 1，回滚 | **exit 0，新增批次／Claim／seq，文本打印 `stored at seq 0`** |

Windows 与 WSL 上结果一致。两次均使用正常完整输出 Writer，没有用输出短写或数据库打开失败替代目标分支。该注入专门证明已承诺的构造后校验防线；**没有证明，也不宣称，普通用户可通过 claims.v1 输入直接设置回执 seq。**

**实际影响：** 默认命令在回执构造产生不合法结果时不会按冻结规则回滚，可提交事实并向人显示错误结果。现有故障测试只带 `--json`，因此 CI 能通过但不能证明默认路径的防线存在。这是明确的实现／验收契约缺口，不是要求所有文本输出自身成为 JSON。

**最小修正：** 将回执的 Schema 校验放在文本／JSON 分支之前；验证通过后再编码所选输出。两种模式均在 COMMIT 前完成；保留提交后只写已编码字节、失败 exit 6 且不追加第二份 JSON 的逻辑。将现有回执故障用例扩展到文本、JSON 和首次 worktree 的回滚，manual 演练也应共用此检查。不要求顺带修改其他 CLI 的 writeJSON。

## 四、其余核心路径的审查结果

### 1. 输入、通道与人工确认

[CLI Claim](../../internal/cli/claim.go) 先以 8 MiB 上限读取输入、严格校验 claims.v1，再识别 kind。manual 在配置加载、数据库／Git 读取之前执行 RequireHuman；格式错误仍先 exit 2。这符合批准的 Q6，不是要求在尚不知道 kind 时拒绝读取输入。

重复 AC、摘要围栏／空段、路径逃逸、行号规则由静态检查处理；输入每个字符串与待登记 worktree 路径／GitDir 在写入前扫描。Human actor 复用经扫描的显示名；Agent 变量只用于识别来源，不作为可信身份。静态错误给 Pointer 和约束，不带原值。

manual 的回滚演练复用正式 Store；确认经 cli.confirm／channel.Guard，JSON 提示转 stderr，非终端 stderr 拒绝。错误确认、取消、EOF、不可见或短写提示均没有业务写入；短写用例明确确认输入读取次数为 0。除 B2 的校验覆盖缺口外，确认后重开写事务而非复用演练结果。

真实进程＋PTY 用例覆盖两个 Claude 环境变量、stdout 重定向、管道输入、stderr 重定向、取消和成功确认。最终终端用例确实已改为非 Git 项目：它证明 OS 通道接线，而不证明 Git 上下文；后者由固定 HEAD、D1、PD-26、worktree、compare_from 等真实 Git 用例分别证明。不能把非 Git PTY 本身写成 Git 验证。

### 2. 写锁、版本优先、完整回滚与 seq

[Store:63](../../internal/claim/store.go#L63) 在调用方 BEGIN IMMEDIATE 后先读取当前版本：base 不同先 exit 5，`current.base_version` 和 `/base_version` 正确；之后才汇集 active AC、D2 需求、D3 归属的 exit 2，最后比较 target_revision。没有用退役而不推进版本的非法历史满足测试。

[版本竞争测试](../../internal/cli/claim_manual_test.go#L190) 使用另一数据库调用通过正常 Proposal/Review 退役 AC／需求，先保留对方提交后的 S1，再放行 Claim；manual 则在实际读取确认输入前提交另一事务。断言最终等于 S1，保留对方的级联、版本、历史及 seq，而没有误要求回退对方事实。首次关联 worktree 拒绝和确认后重新检查都被覆盖。Review 066 的快照基点建议已落实。

worktree 首次登记在 [138](../../internal/claim/store.go#L138) 另取 seq，之后 [151](../../internal/claim/store.go#L151) 为批次、全部 claims 和 unmapped 共取一个 seq；全部属于一个写事务。空批次、仅未映射工作、局部 audit、audit 空 claims 均保留合法性，没有增加完整盘点门槛。JSON 回执注入失败的首次登记回滚通过；文本模式的同类保证须修 B2。

### 3. 迁移 0008 与直接 SQL

核查并实际运行 [TestClaimTablesConstraints](../../internal/store/sqlite/claims_test.go)：

- project/worktree、batch/claim、criterion 的复合归属外键；批次 seq 唯一、Claim 与批次 seq 相同；同批次 AC 唯一。
- manual 只能 human_cli，actor/channel 组合一致；unknown 只允许 audit。
- criterion 必须本项目 active 且为当前 revision；未映射工作的 criterion、revision、assertion 均为 SQL NULL。
- kind、scope presence、SHA、compare_from/source、summary、JSON 对象等检查。
- UPDATE、DELETE、REPLACE（按 ID 或唯一键）及 upsert 的拒绝理由；递归触发器保护沿用既有连接检查。
- maintenance 不能留下悬挂 batch/claim 外键；合法 purge 删除本项目而保留另一项目。

该测试既有成功写入，也核查各错误的实际约束原因，非“任意 SQL 错误均算通过”。INSERT 时的 active/revision 检查不会反过来要求历史 Claim 随后更新；实体退役保留事实，与只追加模型一致。应用层仍负责版本优先、D2/D3 等输入业务检查，不把有限 SQL 约束宣称为完整 claims.v1 校验器。

0001～0007 未改，0008 由原迁移框架升级；没有新增生产 SQLite 重试或改映射。没有因本轮正常打开成功而关闭既有 Open／建库并发维护项。

### 4. 上下文、引用和比较起点

[Capture](../../internal/claim/refs.go) 保存一次 CommitOf 的 HEAD，文件引用经 CheckCodeRefsAt 使用该值，不再次解析 HEAD；测试确实在捕获后推进 HEAD，再验证旧文件存在／新文件缺失和落库 checked_head 一致。非 Git、unborn、正常 undecidable→unknown 与实际环境错误→exit 6 分开。没有承诺 Git、工作区和 SQL 形成跨系统原子快照。

除 B1 外，ResolveCommits 使用结构化候选枚举／类型头，不把任意非零退出映射为 false；unique commit 进行完整流式哈希，存储检查涵盖 pack/idx 双向配对及 alternates；取消、异常输出、受禁和读取失败保守拒绝。PD-26 经 scanCall／既有能力探测；Claim 的部分克隆引用另行拒绝。没有输出原始 Git 错误。

[previousBatch:192](../../internal/claim/store.go#L192) 查询同项目、同 worktree、排除 manual 后 seq 最大的 sync/audit，只检查那个批次是否 clean 且有 head。最新 dirty/unknown/null 不向更早 clean 回溯；audit/manual 没有 compare_from，M2 没有 session_start。现有 Git 用例验证 manual 不参与和关联 worktree 隔离。

CheckCodeRefsAt 沿用普通文件／行号／符号链接／子模块规则；非 Git 仍受根内解析和普通文件检查。CodeRefProblems 是原静态逻辑的直接提取，完整 proposal 包回归通过，未发现本 PR 导致 propose 规则变化。

### 5. 回执对应与输出故障

[Store/readReceipt](../../internal/claim/store.go#L258) 保存输入下标对应的 ID 数组，再按每个 ID 回读写入行，不靠 SQL 未排序返回或 ULID 排序。正常路径的测试将批次、索引、数量、ID、修订、断言、refs_json、refs_check_json 和回执逐项对照；tests 取已存测试名数量，audit_scope 保存在批次但不回显。空引用和空数组明确，unmapped 不伪造 AC 或断言。

JSON 路径提交前验证和编码；B2 是文本路径的例外。COMMIT 后 Writer 显式错误、短写均 exit 6，事实保留，root 的 errOutputCut 链阻止第二份 error.v1 混入 stdout；可用 stderr 给出已存批次与不要盲目重提的说明。文本及 JSON 各有故障测试，部分 stdout 不被伪装成完整文档。没有新增自动重试或幂等承诺。

### 6. 方案 B 合作竞争

Claim 复用未变的 Scratch/Clean/扫描能力探测生命周期，在写事务前完成私有读取及显式 Close；没有更换临时协议、放宽消费者屏障或增加等待／重试。

四个独立进程的测试：非 Git 四批全部成功；Git 情形各批完整提交，或因明确的 namespace lease 竞争 exit 6，成功批次 seq 连续，拒绝方没有额外批次／Claim／seq。不能把这个已批准的竞争结果机械列为新的产品决定。

另在 Windows 独立持有与生产协议相同位置的 LockFileEx 排他锁，运行公开 Claim：exit 6、全部业务快照不变，协议目录内容及 mtime 不变；释放锁后同一输入正常成功。该验证不等于重做整个跨账户／崩溃消费者审查，也没有模拟不合作同用户替换；那些边界继续按已批准方案 B 适用。

### 7. dbdump 与推导接点

dbdump 按项目、在同一 SQL 事务内映射当前工作包需要的事实行及全库 seq，不预选最新 Claim、决定或适用性；JSON 列保留文本，交给原 factgen/toderive。它是推导输入投影，不是含全部字段的数据库备份，也尚不读取未实现的 check_runs/decisions。

无写入断言另由 [factState:461](../../internal/cli/claim_test.go#L461) 动态读取所有业务表、所有字段和 meta 完成，并非只靠 dbdump。TestClaimFactsDerive 使用实际已写行，正确选择有检查的自动化 AC、manual AC 和开发阶段项目，验证 claimed_without_evidence、manual_claimed、局部／完整 audit；未修改推导或把 Claim 当成 accepted/ready。真实 status/gaps 输出、检查执行、Decision、会话及 E2E 仍由后续对应工作包完成。

## 五、NON-BLOCKER、产品决定及接受风险

### NON-BLOCKER N1：PR 验证说明应与最终测试口径对齐

开始审查时的 PR 正文 Findings 第3项仍把 PTY 列入“每个 Git 相关用例都在 Git 项目中”的清单；最终 `69d2f86` 明确已将 TestClaimManualInTerminal 改为非 Git。源码注释如实说明这一点，且有独立的 Git 功能用例，因此不构成通道证据缺失。

建议更新 PR 说明，分别陈述“PTY 证明通道”“真实 Git 用例证明上下文”，不要继续把最终 PTY 当作 Git 验证。不需要为此把通道测试机械迁回 Git 或改动产品行为。

### PRODUCT DECISION

本轮没有新增必须由负责人决定的产品事项。Q6/Q9、D1～D3、E4/E6 均为冻结输入；B1/B2 的最小修正属于落实现有保证。PR 正文提出未来是否等待合作锁的想法，不是当前实现成立所必需的决定；本轮不批准等待策略，也不授权修改方案 B。

### ACCEPTED RISK

没有新增接受风险。仅沿用 review-resolution §71 的 Git 私有临时区方案 B 局部排除：不合作同用户干预及已明确的 Unix 窗口／Clean 影响。它不豁免正常合作竞争、其他账户、消费者崩溃保护、Log 完整性、数据库或隐私。

M-14、M-15、M-16 及其他维护事项继续未解决，不是本轮接受的风险；M2 未整体验收。

## 六、实际独立验证及边界

固定 HEAD 导出包 SHA-256：

`a6dbfc04ac4b3baa414b26f8209705ad2f9b7e20b251b9897eef97ab3114a2be`

所有新增临时反例和故障注入都在仓库外隔离副本；原仓库没有补入测试或实现。使用缓存依赖，GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，Go `1.27.1`。本机 Windows/amd64，Git `2.56.0.windows.2`；另以交叉编译测试二进制在 WSL Ubuntu/amd64、UID 1000、真实 Git `2.43.0`、Linux 临时目录运行。

| 验证 | 结果及范围 |
|---|---|
| internal/claim 的 ParseExamples、CheckStatic | PASS |
| sqlite 的 TestClaimTablesConstraints | PASS，含直接 SQL、REPLACE/upsert、purge 正反例 |
| Windows 全部 TestClaim* | PASS，CLI 包 147.530 秒；真实 unsafe Git 条件未启用，明确 SKIP |
| ResolveCommits*、CheckCodeRefsAt、CodeRefProblems | PASS；gitx 52.605 秒、proposal 6.987 秒；Windows 不可读权限与真实 unsafe Git 用例 SKIP |
| 完整 internal/proposal 包 | PASS，15.533 秒；覆盖共享规则直接回归 |
| internal/contract | PASS；另一次详细运行 0.775 秒，实际列出 43 合法示例、190 非法反例，24 Schema 编译 |
| 命令参数脱敏、正常命令、隐藏补全拒绝、Question 参数错误、确认短写五项定向回归 | PASS，52.745 秒 |
| B1 两个独立库反例 | Windows 与 WSL 均 FAIL 于预期的 E4 断言：实际错误地返回 false/nil |
| B1 公开 Execute＋实际 Git/SQLite 反例 | Windows 与 WSL 均复现 exit 0 和新增事实／seq；并非正式发布二进制测试 |
| B2 同一非法回执的文本／JSON 对照 | Windows 与 WSL 均复现：JSON 回滚，文本错误提交；使用已有构造后钩子，非硬件故障 |
| Windows 实际协议字节锁竞争、协议目录快照、释放后成功 | PASS，最后含目录快照的运行 14.242 秒 |
| WSL 不可读 loose 文件／目录 | 非 root 实际 PASS，非 Windows 跳过的替代记法 |
| WSL 真实 PTY 与四进程 Claim 对照 | 绝对路径启动后 PASS：并发 3.15 秒，终端 4.58 秒 |

验证过程中的非产品失败也保留：

- 初次租约校准仅创建 NewScratch，没有持有实际读取租约，却预期 busy，断言失败。核对 acquire 的真实生命周期后改为持有生产协议同位置的排他字节锁；该初次失败不算产品缺陷或有效竞争证据。
- 首次 WSL 用相对路径启动交叉编译测试二进制，测试改变 cwd 后辅助进程找不到 `./cli-linux.test`，PTY／并发对照失败；改用绝对路径、未改测试或实现后通过。同期 B1/B2 不依赖辅助进程的精确反例仍实际复现。没有把环境启动失败算成产品检出。
- 本轮产品测试没有超时。隔离测试命令使用有界的命令级期限；没有修改仓库脚本、Go 默认配置或 CI 超时，完整 Claim 定向包耗时低于既有 600 秒默认包期限。没有为失败重跑 CI。

未独立重跑作者的桩阶段红灯、全部变异或真实 Git 2.32 实验。交接提到20项变异；本次读取的 PR 正文可逐条对应迁移4项、解析器4项、CLI8项，未取得其余项的独立记录，不将计数或作者“全部检出”作为穷尽证明。作者关于 WSL Git 2.32 的两项真实测试仍是作者本地证据，不能写成三平台 CI 或本轮独立复验。

本机没有 macOS；macOS 结果来自下节 CI。没有重跑整仓、M1、全部 WP2.3 安全实验或生产工作流，不证明断电／物理磁盘故障、所有文件系统或未来命令集成。

## 七、CI 身份、实际运行和失败历史

### 最终运行

[run 39464343744](https://github.com/XKHoshizora/specmetry/actions/runs/39464343744)：pull_request、attempt 1、正式 completed/success，HEAD 为固定 `69d2f86`。

已读取八个 Job 日志，全部实际 checkout：

`1a2caaf225b63b75ad831d70c400a05132d93041`

独立 Git commit 元数据核对：

- 第一父：固定 base `d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`。
- 第二父：固定 HEAD `69d2f86bb1c9f2b3f842e127576ef04b7d886d7a`。
- tree：`e828b9024862e6cfdeb9c7d75cde281c2e97554d`，与本地固定 HEAD tree 完全相同。

| Job | ID | 正式结果 |
|---|---|---|
| Go lint | 118771688308 | success |
| Go tests (macos-latest) | 118771688394 | success |
| Go tests (windows-latest) | 118771688692 | success |
| Go tests (ubuntu-latest) | 118771688693 | success |
| Web | 118771688881 | success |
| Build with dashboard (macos-latest) | 118771914396 | success |
| Build with dashboard (windows-latest) | 118771914402 | success |
| Build with dashboard (ubuntu-latest) | 118771914410 | success |

三平台日志各有45条 TestClaim* 的 PASS 记录，**含父测试及子测试**；不是45个独立顶层测试。16个顶层 Claim 测试通过，真实 unsafe Git 的另一个顶层测试跳过。ResolveCommits 的 PASS 记录 Unix 为21条、Windows为18条，差额来自 Unix 不可读存储测试及两个子例；不能把 Windows 跳过计为通过。

实际 PASS 包括固定 HEAD、版本竞争、四进程提交、manual PTY、提交后输出故障、首次 worktree 回滚及 SQL 约束。CLI 包耗时：Ubuntu 367.414 秒、macOS 169.986 秒、Windows 453.444 秒。它们通过的是本 PR 已提交的断言，不包含本轮新增 B1/B2 反例。

Ubuntu 普通 Test 步骤确实采用 `-race`。另一个 sudo root 步骤编译存储测试二进制，实际通过8项跨账户／Scratch 测试，**不带 -race**；不能把这两个步骤合称 root 下 Race 证明。Windows/macOS 的该 root 步骤按设计不适用。真实 unsafe Git 条件测试在三平台均 SKIP；Windows 不可读文件权限测试 SKIP；显式诊断等其他条件性跳过继续按日志理解。

### 必须保留的首轮失败

[run 39461898893](https://github.com/XKHoshizora/specmetry/actions/runs/39461898893)：attempt 1，HEAD `cb224b1219a84e52ceb4b2580d446d8a9128d070`，completed/failure。

- 实际旧合并快照 `97fdb7ba4316bb67e69681e778921755d751992e`，父提交为固定 base 和 `cb224b1`；tree `6cdea6ad362f2ea330edef5b845cb2796ff51ae4`。
- Go lint job `118763216116` 失败：gocritic unlambda 1项、unused nolint 3项、staticcheck QF1011 1项，日志确有5项。
- 其他七个 Job 均正式 completed/success，不只是搜索不到 FAIL。三平台 Go tests 的 CLI 包实际完成：Ubuntu 455.032秒、macOS 147.292秒、Windows 533.842秒。没有将取消／未完成记为通过。
- `09596fe` 修 lint，最终提交另调了终端夹具。最终成功不改写首轮历史，也不能证明 M-14/M-15/M-16 的根因、耗时来源或风险已经解决。本轮未重跑 CI、调整生产重试或关闭维护项。

## 八、最终回答与最小接手范围

| 问题 | 裁定 |
|---|---|
| PR #30 技术 GO/NO-GO | **NO-GO，2项 BLOCKER** |
| 是否可以合并 | **当前不建议**；CI 8/8 的合并检查已满足，但 B1/B2 尚未修复 |
| 是否足以支持 WP2.4 本工作包验收 | **尚不足**；正常路径证据有效，E4 和默认模式的回执防线仍有实质缺口 |
| Review 065/066 的版本优先要求 | 实现及合法竞争证据通过，没有重开原规范结论 |
| 是否需要新的产品决定 | **无**；按冻结契约修复，不要求改变 E4、D1～D3 或方案 B |
| 是否新增接受风险 | **无** |

最小后续任务：

1. 实施方修 B1，补头部可读但正文／对象 ID 错误的非提交和歧义候选，以及 CLI 不写入反例；合法 false 对照继续通过。
2. 实施方修 B2，所有显示模式均在 COMMIT 前校验内部回执；扩展故障注入到文本／JSON、首次 worktree 及 manual 演练，保留提交后输出故障语义。
3. 修正 PR 的最终测试口径说明，运行受影响定向回归及对应 CI，再由独立审查确认修复。无需重开未变的整个 M1、推导引擎或 WP2.3 安全审查。

本轮仅新增本未提交报告，链接使用仓库相对路径；未修改其他原仓库文件、历史审查、生产代码、测试、Schema、迁移或冻结规范。未提交、推送、合并、改基、重跑 CI、宣布工作包／M2 验收或启动 WP2.5。M-14/M-15/M-16 等维护事项保持原状，后续决定与合并权仍属于负责人。
