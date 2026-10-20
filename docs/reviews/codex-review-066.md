# Codex Review 066：PR #29 Claim 规范定向复审

## 一、结论

**GO。Review 065 B1 可以关闭；N1、N2 在规范层面已收口；E4、E6 已忠实落实负责人的批准。未发现新的实质性 BLOCKER。**

PR #29 可以由负责人合并，WP2.4 Claim 规范及 `claim-receipt.v1` 具备供负责人冻结的条件。本轮范围内没有尚需负责人裁定的产品取舍。

保留一项非阻塞说明建议：并发测试中的“seq 计数器不变”应明示比较快照的基点，不能把另一连接的合法提交算作被拒 Claim 的写入，详见第七节。该建议不改变已经明确的“本次 Claim 不提交事实或 seq”契约，不构成新的产品决定或风险接受。

本结论是规范复审 GO，不是宣布规范已经冻结、Claim 实现已经通过、WP2.4 已验收或已获实现授权。指定 CI attempt 1 已正式 8/8 成功；实际合并、合并核验、规范冻结和开工授权仍由负责人处理。

## 二、固定身份与范围

- 审查日期：2026-10-20，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#29](https://github.com/XKHoshizora/specmetry/pull/29)。
- 固定 base：`7a361663ccb53f6a7db594a784c08ad54a24e4c8`。
- 上轮对象：`058b9a07ef3d62acae6ab51e45fb338f7e5820e3`。
- 本轮 HEAD：`29834d1fb0c70d3abd95b0b3efba777c98bdba9c`。
- HEAD tree：`7619127b74e02d8dc67b1d2b74ada7ed5536d26f`。
- 开始及结束远程核查均为上述 HEAD/base，PR Open、未合并、`auto_merge = null`；没有把其他提交纳入结论。

已读取适用的根 [AGENTS.md](../../AGENTS.md) 和完整 [Review 065](codex-review-065.md)，按固定 Git 对象核对两个提交：

1. `ca3e4950d274f9fa05b3eb1157df8372f19f4a79`：原样归档 Review 065。
2. `29834d1fb0c70d3abd95b0b3efba777c98bdba9c`：版本优先、N1/N2、E4/E6 及活跃记录同步。

增量确实只有六份文档，446 行新增、45 行删除：

| 文件 | 核查内容 |
|---|---|
| [wp2.4-claim.md](../wp2.4-claim.md) | 处理顺序、反例、输出故障、既定决定与实现测试要求 |
| [acceptance-criteria.md](../acceptance-criteria.md) | INV-01、INV-02、CLM-02 的关联澄清 |
| [data-model.md](../data-model.md) | V-C06 的 E4 分类边界 |
| [implementation-plan.md](../implementation-plan.md) | 记录 Review 065、修订及待复审状态 |
| [review-resolution.md](../review-resolution.md) | 新增 §78：历史审查、修订及负责人决定 |
| [codex-review-065.md](codex-review-065.md) | 历史原文归档 |

`internal/`、`schemas/`、`.github/`、`go.mod`、`go.sum`、`web/` 与上轮对象的 diff 为零；没有代码、测试、示例、反例、迁移或工作流修改。没有改动冻结的推导规则，也没有开始 Claim 生产实现。

### Review 065 原样归档

当前文件、归档提交的导出文件与上一轮交付原文的 SHA-256 均为：

`109c823528dc4ecd9c352840f73af143eca2e5167e17ec00113a0e91b29fc0f3`

归档 blob 为 `7c24fafc28e70a58afe461ae732068b73700641f`，当前文件按原始字节计算的 Git blob ID 与之相同；归档提交之后该文件没有变化。

原文仍审查 `058b9a0` / 固定 base，保留技术 NO-GO、唯一 B1、当时 E4/E6 待决定及原验证边界。§78 没有把历史对象改成新 HEAD，也没有用实施方的“已修订”替换独立审查关闭；末尾明确保留“待复审、未冻结、未授权实现”。

## 三、B1：版本优先已修正

**裁定：Review 065 B1 在规范层面可以关闭。**

核对位置及结果：

| 位置 | 现在的要求 |
|---|---|
| [§4 第5步](../wp2.4-claim.md#L62) | manual 确认后重新检查仍按版本优先；合法基准变化含退役时先 exit 5 |
| [§5 写事务](../wp2.4-claim.md#L83) | `BEGIN IMMEDIATE` 后首先比较 `base_version`；不一致返回 exit 5 和当前版本，不进入后续 exit 2 检查 |
| [§5 第10步的业务检查](../wp2.4-claim.md#L84) | 版本通过后才按 V-C03、D2、D3 判断 active/范围，并保留精确 Pointer |
| [§12 映射](../wp2.4-claim.md#L255) | CLM-02 区分当前版本下的无效目标与读取后发生的合法退役 |
| [§13 反例](../wp2.4-claim.md#L294) | 当前版本无效输入 exit 2；合法并发退役导致旧版本输入 exit 5；确认期间同样如此 |
| [§14 决定](../wp2.4-claim.md#L322) | 明确 D2/D3 只在基准版本前置条件通过后适用，不撤销批准 |
| [§15 测试计划](../wp2.4-claim.md#L352) | 两连接通过正常审核退役、推进版本；旧输入 exit 5；另测当前版本的无效实体 exit 2 |
| [acceptance-criteria.md](../acceptance-criteria.md) | INV-01、CLM-02 增加相同的优先级说明 |

规范级对照如下，没有使用非法直接改库或“AC 改换父需求”的场景：

1. A 读取 v1；B 经正常审核退役需求或 AC 并提交 v2；A 之后取得写锁。A 必须在版本检查返回 `stale_precondition / exit 5`，`error.current.base_version = 2`，Pointer 为 `/base_version`，不能再改报 exit 2。
2. 输入已经使用当前 v2，但仍引用已退役／不存在的 AC 或需求。版本检查通过后返回 exit 2，分别指向 `/claims/<i>/target` 或 `/audit_scope/requirements/<j>`。
3. 输入使用当前版本、目标 AC active，但所属需求在声明的 audit 范围之外。D3 返回 exit 2，指向 `/claims/<i>/target`。
4. manual 演练通过之后、确认期间发生 B 的合法退役，正式事务仍先返回 exit 5，不复用演练时的版本和有效性结论。

这些规则可同时满足冻结的 data-model §7.1、V-C02 与 D2/D3。局部覆盖、`{"all":true}`、空 claims 正例仍保留，没有把局部盘点变成必须覆盖全部 AC 的操作。

### 并发事实与 seq 的比较边界

必须保留 B 已提交的基准版本、退役、级联、历史及其 seq；拒绝 A 不能撤销另一个事务。

后续实现测试可采用明确调度：A 暂停在取得写锁／确认之前；B 完整提交；读取 **B 提交后的快照 S1**；再放行 A；A 返回 exit 5 后与 S1 比较。此时应无 A 的 batch、claims、unmapped_work、首次 worktree 登记或额外 seq，B 的全部事实保持。

若还要与 B 提交前的 S0 比较，必须允许并精确核对 B 的合法增量，不能断言 S0 与最终数据库相同。该边界来自已有事务和 seq 契约，不是新接受的副作用。当前增量的总则和 §78 明确约束“本次 Claim”；§15 的简写不应被解读成全局冻结，相关说明建议见第七节。

上述是本轮对规范逻辑和测试设计的核查；没有 Claim 实现，不能声称已经动态通过这组数据库竞争测试。

## 四、N1、N2 的收口

### N1：收口

[§5 总述](../wp2.4-claim.md#L68)、§13 反例总述和 §78 已改为“不提交本次 Claim 的业务事实、不分配本次写入的 seq”，并明确不是数据目录逐字节不变。

普通 Open 的既有版本检查、辅助文件行为，以及方案 B 的创建、可信清理、失败保留均保持原契约；本轮没有通过修改文字承诺数据库未打开、辅助文件零变化或所有私有目录都能删除。提交前失败的限制不适用于 §9.3 明确区分的提交后输出故障。

### N2：收口

[§9.3](../wp2.4-claim.md#L205) 与 [§9.4](../wp2.4-claim.md#L222) 已明确：

1. 写事务中从写入行构造回执，在 COMMIT 前校验并编码输出字节。
2. 校验／编码失败是内部错误，事务回滚；同事务的首次 worktree 登记及其 seq 也不能提交。§5 的事务回滚和 §15 对首次登记的检查与此一致。
3. COMMIT 后只写出已编码字节，不再校验或编码。
4. 写入错误和短写均 exit 6；批次及相关事实已提交，不能回滚。
5. stdout 可能已有部分回执，不追加第二份 JSON，不再保证故障流中有完整 error.v1。
6. 通过仍可用的 stderr 尽力说明“可能已经提交，不要盲目重试”；不把重新读取状态视为原提交不存在的证明。

§9.4 已为完整错误 JSON 总则加上提交后输出故障的例外，因此不会要求对截断 stdout 作不可实现的撤回。§5 第10步引用 §9.3，COMMIT 仍在该构造／校验阶段之后；不是把编码挪到提交后。

§15 增加了提交后显式错误／短写的测试要求。实施时还须覆盖提交前编码故障的回滚，以及命令上层错误处理不能再次向部分回执追加 error.v1；不能仅修改底层 writer 就假定整条链满足要求。这些属于已经写明的实现验收责任，不要求本规范 PR 提前提供 Claim 测试或改动其他 CLI 的 `writeJSON`。

## 五、E4/E6 和其他细化

### E4：忠实落实，已批准，不再是待决事项

[§7](../wp2.4-claim.md#L141)、data-model V-C06、INV-02、§13～§15 与 §78 一致区分：

| 条件 | 结果 |
|---|---|
| 格式合法、唯一指向可读 commit | resolved=true |
| 可靠确认不存在、非提交对象、真实歧义 | resolved=false，照常提交，沿用 unknown_commit，不增加原因字段 |
| 格式非法 | exit 2 |
| 损坏、读取失败、无法可靠分类、PD-26 拒绝 | exit 6，不以 false 掩盖 |

新函数必须使用受控入口及结构化分类，不能把全部非零退出转换为 false，也不能匹配原始 stderr 猜测。非 Git 项目的 false 不触发 Git 对象读取。存在性仍不代表验证、证据或验收。

这没有谎称现有 `CommitOf` 或只接受完整 commit ID 的 `Classify` 已能完成全部新语义；§7 明确保留新解析函数的实现任务。对象“缺失”的用例应按可靠不存在与损坏／不完整导致无法可靠分类的区别落实，而非一概返回 false 或一概拒绝所有真正不存在的前缀。

### E6：忠实落实，已批准，不再是待决事项

[§9.1](../wp2.4-claim.md#L182) 与 §78 明确 tests 只输出数量，回执省略 audit_scope，但 audit_scope_json 仍是批次事实；D2/D3 及逐项 mapped/unmapped 对应没有取消。

回执用于确认写入，不是完整输入备份，调用方保留原输入；它不能单独恢复测试名或声明范围。该决定没有改变 status/gaps 的显式输入、事实历史或重放保证。Schema 的严格字段及现有例子原本就表达这些输出边界，因此本次不修改 Schema 是一致的，不是遗漏批准字段。

### E1/E2/E3/E5 和实现要求

- E1 的确认值仍是 `v<base_version>`；提示补齐 kind、数量、逐项信息及 unmapped 的数量／引用，不显示摘要。
- E2 的 manual/channel 数据库约束作为第二道防线；没有升级为身份认证，迁移仍待实现。
- E3 准确选择同项目、同 worktree、排除 manual 后 seq 最大的 sync/audit；只看那一批。它若 dirty、unknown 或 head=null，M2 结果为 none，不跳过它寻找更早 clean，也不产生 session_start。§6.3 与 §14 的 A/B/C 例子一致。
- E5 只约束本工作包新的 Claim 回执顺序，没有修改或豁免 propose。

新增“实现时须兑现的要求”均可追溯到 Review 065：固定 head 的文件检查、新前缀解析器、迁移防线、同事务回滚、输入索引映射、actor 扫描、termtext、短写和进程级通道验证。没有批准新认证、完整盘点门槛、幂等系统或其他工作包实现。

CLM-01 的夹具说明要求存在已确认检查但无执行证据，才期待 claimed_without_evidence；无检查保持 no_check。该说明落实既有推导规则，没有修改冻结算法。

## 六、契约证据适用性与实际验证

已先确认源码、Schema、示例、反例和工作流均未变。因此 Review 065 对 24 个 Schema、43 个合法示例、190 个非法反例，以及 SCH-166～190 实际拒绝原因的独立证据仍适用于本轮相同内容。原报告对重复 index/ID、M2 不得产生 session_start 等业务测试边界的说明也继续有效，没有被本次文档修订悄悄取消。

本轮另从固定 HEAD 在仓库外导出隔离副本，仅运行必要的契约回归：

```text
go test -tags nowebui -count=1 -timeout 3m -v ./internal/contract
```

- 平台：Windows/amd64，Go 1.27.1；使用已有依赖缓存，GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local。
- 结果：PASS，包 0.549 秒；Schema 名单与编译检查、43 个合法示例、190 个非法反例及其余契约测试均通过。
- 导出包 SHA-256：`ace7cf87817d821ee34002711ca2b915d92aa9dd6af9db9edb8c9ba2b14a4335`。
- 未新增临时业务实现或运行虚构 Claim 命令，未重复上轮逐例临时程序、整仓、TTY、真实数据库竞争或 Git 解析压力实验。

本轮新增的是测试计划和文档约束，没有新增 Claim 实现测试；其计划不能因 CI 绿色被登记为运行成功。

## 七、问题分类

### BLOCKER

无。Review 065 的退出码冲突已经消除；未发现本轮直接引入的新技术、安全、隐私或事实一致性阻塞。

### NON-BLOCKER N-doc：建议明示并发检查的快照基点

位置：[wp2.4-claim.md:350](../wp2.4-claim.md#L350)、[352](../wp2.4-claim.md#L352)。两处仍使用“数据库没有任何新行／seq 计数器不变”的简写，没有直接写明快照取自另一连接提交之后。

全文 §5、§13 总述及 §78 已把保证限定为本次 Claim，版本竞争又明确要求另一审核提交新的基准版本，因此存在一致且可执行的解释：以对方提交后的 S1 为基点，或单独核对它的合法增量。文档没有明确要求以 S0 为基点撤销别人的提交，不据这句简写重新判 B1 未关闭。

建议在实施测试说明中加一句：“先保留对方提交后的快照，再比较被拒 Claim 的额外写入与 seq；对方合法变更不要求回退。”第三节已给出具体时序。若后续测试真的比较 S0 并要求全局不变，应判为测试缺陷，不能用非法历史或弱化事务来满足它。

这是非阻塞的可读性／测试落地说明，不新增产品决定，不推迟核心保证，也不要求本 PR 为此实现竞争程序。

### PRODUCT DECISION

本轮范围内无剩余产品待决事项。E4/E6 已由负责人批准，规范和 §78 落实忠实；Q6/Q9、D1～D3 不重开。其他 M2 未决内容没有因本 PR 获得批准或实现授权。

### ACCEPTED RISK

没有新增风险接受。仅继续引用既有方案 B 的 Git 私有临时区局部边界，未扩大到其他账户、正常合作并发、消费者崩溃保护、数据库、来源权限或隐私。M-15、M-16 及其他维护事项仍未解决，不是接受风险。

## 八、CI 与历史失败

[run 39433229408](https://github.com/XKHoshizora/specmetry/actions/runs/39433229408)：pull_request，HEAD 为固定 `29834d1`，attempt 1，正式 completed/success，8/8。

已读取八个 Job 的日志，实际 checkout 均为 `0c3e10c89a46c25bc52938d23d8f26ffa046eaf4`：

| Job | ID | 结果 |
|---|---|---|
| Web | 118663411166 | success |
| Go lint | 118663411501 | success |
| Go tests (windows-latest) | 118663411525 | success |
| Go tests (ubuntu-latest) | 118663411679 | success |
| Go tests (macos-latest) | 118663412020 | success |
| Build with dashboard (ubuntu-latest) | 118663620938 | success |
| Build with dashboard (windows-latest) | 118663620959 | success |
| Build with dashboard (macos-latest) | 118663622414 | success |

独立读取 Git commit 元数据确认合并快照第一父为固定 base、第二父为固定 HEAD；其 tree `7619127b74e02d8dc67b1d2b74ada7ed5536d26f` 与 HEAD 相同。

三平台实际 PASS `TestAllSchemasCompile`、`TestValidExamplesPass`、`TestInvalidCasesFail`，日志明确列出 43/190，未跳过这些契约检查。Ubuntu 普通步骤使用 -race；单独 sudo 跨账户步骤八项 PASS，不带 -race。该 root 步骤在 Windows/macOS 按设计不运行；普通 Linux 步骤跳过 root 用例，另外仍有真实旧/unsafe Git、显式诊断、Windows 不可读权限等设计跳过。没有将整仓成功写成所有环境条件均已执行。

保留 Review 065 已独立核对的旧 run `39307739451` attempt 1 失败：Windows CLI 包达到默认 600 秒超时，两个 seq 并发测试出现 SQLITE_BUSY；旧 HEAD `91e7e5f` 与快照 `60f8820` 的身份不改写。本轮六文件增量未修改维护登记，§78 继续明确 M-15/M-16 未解决。新 CI 不证明旧失败根因消失，也不能据此归因为资源竞争。本轮没有重跑任何 CI。

## 九、最终回答与接手

| 问题 | 裁定 |
|---|---|
| Review 065 B1 能否关闭 | **可以**，版本优先的规范冲突已修正 |
| N1/N2 是否收口 | **是，规范层面收口**；实现阶段仍须验证所有失败路径和输出故障 |
| E4/E6 是否忠实落实 | **是**，已批准，不再列为待决定 |
| 是否仍有本轮产品待决事项 | **无** |
| 技术审查 GO/NO-GO | **GO** |
| PR #29 是否可以合并 | **可以**，由负责人决定并执行合并，随后核对实际结果 |
| 规范是否具备冻结条件 | **具备**，冻结动作及身份确认由负责人作出；本报告不宣布已经冻结 |

下一步由负责人处理 PR 合并、合并结果与对应检查核验，再决定规范冻结及是否明确授权 WP2.4。后续实施应按现规范兑现前缀解析、固定 head、迁移、事务与输出故障、通道等测试；无需为本次纯文档收尾重开 M1、已验收算法或 WP2.3 审查。

原工作区起始干净，本轮只新增此未提交报告。没有修改其他原仓库文件、历史审查、生产代码、Schema、测试或迁移；没有提交、推送、合并、改基、重跑 CI、关闭维护事项或启动 WP2.4/WP2.5。
