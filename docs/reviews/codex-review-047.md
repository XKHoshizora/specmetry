# Codex Review 047：PR #21／#22 的 Review 046 修复定向独立复审

- 日期：2026-10-18（Asia/Tokyo）。
- 仓库：`XKHoshizora/specmetry`。
- 冻结规范基准：`666efe2c907d518b91cd86710a18989efa5dfdb4`。
- PR #21：[链接](https://github.com/XKHoshizora/specmetry/pull/21)，原目标 `1dc17740a44c02f170dcf066d480a77fde4f2e24`；本轮目标 `b3af21b076115c7934e962e068c8732f96717739`，base 为冻结规范基准。
- PR #22：[链接](https://github.com/XKHoshizora/specmetry/pull/22)，原目标 `221df8877f331e6726bfec033fc20151450dbab3`；本轮目标 `bd883721df8e7c4551143b1b1c77f71945aa4a26`，base 为 **`b3af21b076115c7934e962e068c8732f96717739`**，不是 master。
- 范围：完整阅读 [Review 046](codex-review-046.md)，复审 B1、N1、N2、P1、受影响的测试证据及直接回归。保留未变推导算法、12 条 literal readings、复核器独立性的既有裁定，不重做 M1 或整个 M2 审查。
- 方法：固定增量及必要写入调用链的静态审查；GitHub 身份、CI 元数据与日志核查；Windows 定向包测试；隔离副本中的修复前后回归、原始行及约束审计。
- 归档：仅新增本报告，使用仓库相对链接。下述源码位置对应本轮固定 SHA；未修改其他原仓库文件，未提交、推送或合并。

## 结论

| 对象 | 本轮结论 | 原因 |
|---|---|---|
| PR #21／WP2.1 | **NO-GO** | 原三个 seed 的问题已修复，但新合法域门禁误拒绝合法级联历史；当前生成样本中仍存在不可合法补全的输入及 Question 引用历史。B1 尚不能整体关闭。 |
| PR #22／WP2.2 | **NO-GO** | 自身 N1／N2 修复成立，未发现新的 ready 算法缺陷；合法域双向对照仍继承 #21 的上述问题，验收证据尚不足。 |

逐项裁定：

| Review 046 意见 | 本轮结果 |
|---|---|
| B1 | **部分修复，未关闭。** 原 seed 2、5、3 的 V-P03／应用顺序反例修复有效；下文 B1-R1～R3 是当前合法域证据的三个具体缺口。 |
| N1 | **可以关闭。** 两套模型／转换器保留了本次要求的前驱决定 ID、执行上下文与修订字段，NULL 与具体前驱 ID 得到区分。 |
| N2 | **可以关闭。** GradeConsistent 在每个主对照样本上执行，包含非 ready；另有错误等级必须失败的直接测试。 |
| P1 | **技术澄清符合最小修订，负责人确认仍待登记。** 行为未变，规范与测试一致；本审查不代替产品负责人确认。 |

**没有发现本轮改动直接引入的确定生产推导／ready 谓词算法反例。** 当前 NO-GO 的依据是可复现的合法事实域错误及关键验收证据缺口。两个实现一致、校验器返回无问题、CI 全绿，都不能把这些具体非法历史变成合法历史。

## 一、提交身份与增量归属

开始及结束核查时，两个 PR 均为 Open，HEAD／base 与上述固定身份一致，没有混入未审查提交。开始时本地 HEAD 为 `bd88372`，工作区干净，`codex-review-047.md` 不存在。

- #21 的 `1dc1774 → b3af21b`：23 个文件，主要为 Review 046 归档、factgen／legality、Review 构造器、测试调整、N1 字段与映射、P1 及活跃记录同步。
- #22 的 `221df88 → bd88372`：包含上述继承修复，另有 WP2.2 的模型、转换器、GradeConsistent 及相关测试增量。不能把整个增量误记为 WP2.2 自己实现。
- 按 #22 当前 base 比较 `b3af21b → bd88372`：7 个文件，仍为 readycheck 及其测试、开发文档；#21 的 derive／factgen 源码在组合树中没有再次改变。
- 生产推导算法的 criterion、history、derivation、derive、inputrecord、keys、result 文件没有本轮逻辑修改；derive 的生产变化为保留 SeenDecisionID 的输入字段。readycheck 的谓词／历史重建算法未改，本轮计算逻辑增量是输出等级一致性帮助函数。
- 没有迁移、CLI 业务实现或 Schema 改动，没有实现 WP2.3。Review 046 原文自归档提交 `6d35f05` 起未被改写；§58 保留原审查身份与待复审状态，没有自行宣布关闭或工作包验收。

主要修复提交：`e1498f8` 增加校验及回归，`23b3e9b` 调整构造器与生成器，`8d92fc8` 补 N1，`765413e` 同步文档，`bc27868`／`b3af21b` 做 lint 修订。#22 通过合并继承共享修复，自身修复为 `0c329f6`，最终目标为 `bd88372`。

## 二、BLOCKER：B1 的残余缺口

以下三项的修复主体均在 #21 的共享基座；#22 的合法域验收依赖它们。无需重构推导器，也不要求提前实现 M2 写入命令或 SQLite 前端。

### B1-R1：把显式 Target 唯一性扩大成所有变更行的实体唯一性，误拒绝合法级联

**位置：**

- `b3af21b` 的 [factgen/review.go](../../internal/testutil/factgen/review.go) 第 454～455、502～503 行：任何实体在同次写入出现第二次 change 就 panic。
- [factgen/legality/legality.go](../../internal/testutil/factgen/legality/legality.go) 第 420～423 行：无论显式条目还是级联，统一报告 `one change per entity`。
- [factgen/random.go](../../internal/testutil/factgen/random.go) 第 346～348 行：排除本次退役 Goal 下 Requirement 的显式操作，使这种合法组合不能通过 Random 产生。
- [review-resolution §58](../review-resolution.md) 也把“同一写入中每个实体至多一次变更”登记为被检查的规则，但它不是 V-P03 的含义。

**合法反例：**已有 Goal G、其 Requirement R 及 AC。一次人工编辑的两个显式 Item 分别为：退役 G；material 更新 R 的 title，省略 goal 字段。

按冻结 §5.5，合法变更顺序为：

| change_index | 变更 | Proposal Item |
|---|---|---|
| 0 | Goal G retire | Item A |
| 1 | Requirement R update，goal 清空，editorial，revision 仍为 1 | NULL，Goal 退役级联 |
| 2 | Requirement R update，title 改变，material，revision 为 2 | Item B |

显式 Target 只有 G 和 R，各一次；R 的两行中一行是级联，因此不违反 V-P03。AC 未跟进时成为 parent_revised，也是合法但不能 ready 的历史。

**源码级依据：**

- [data-model §5.2、§5.5](../data-model.md) 规定 Goal 退役级联先执行，之后执行 Requirement 步骤。
- [baseline/apply.go](../../internal/baseline/apply.go) 第 618～636 行实际记录 Goal 的清空关联级联；之后 update 从当前记录加载 R，R 仍 active，可以更新。
- 同文件第 259～282 行按 change_index 追加每次变更，没有禁止同版本同实体第二行。
- [0005_baseline_changes.sql](../../internal/store/sqlite/migrations/0005_baseline_changes.sql) 的唯一约束是 `(version_id, change_index)`，不是 `(version_id, entity_id)`。

**独立验证：**在隔离副本中以字面原始行构造上述历史，不调用 History／Random：`legality.Check` 仅因 `REQ-001 changes twice` 拒绝。独立使用新 History 构造相同操作也 panic。去掉该新门禁、直接把同一原始行投影交给两个既有算法，均不 ready；生产推导重建出 R revision=2、goal=NULL、AC reason=parent_revised。这说明被挡住的是合法历史，不是算法已经出现错误 ready。

尝试实际 CLI 写库的额外对照未能完成：隔离环境先在 Git 临时夹具创建时报权限错误，改为非 Git 项目后在数据库锁创建阶段 Access is denied。均未进入目标业务写入，不能计作生产 CLI 动态通过，也不是本项的业务失败证据。本项依据上述独立行级复现及完整源码路径，不声称已经动态执行成功的 M1 数据库操作。

**影响与最小修复：**当前门禁缩窄了合法共享历史，使两边都在排除真实生产组合后的集合上通过。应只对显式 Proposal Target、Item 应用次数及确有规定的 follow-up 限制做唯一性校验，按 change_index 保留合法级联与后续显式变更；移除笼统的实体唯一限制。补一个上述合法正例，并保留重复显式 Target 必须拒绝的反例。对已经被级联退役、因而不再 active 的 AC 的非法显式操作仍应拒绝，不能为修复此项放宽 active 前提。

### B1-R2：同一不可变提交的同一路径先缺失、后存在，仍被作为合法样本

**位置：**

- `b3af21b` 的 [factgen/random.go](../../internal/testutil/factgen/random.go) 第 590～596 行：对声明了 inputs 的检查随机生成 input_missing，不与该提交的输入树身份保持一致；正常执行和 deliver 又为相同提交、路径生成元数据。
- [legality.go](../../internal/testutil/factgen/legality/legality.go) 第 725～747 行：检查声明路径与 InputsJSON 的局部形状，但没有捕获本例跨记录的存在性矛盾。

**固定、可复现样本：**使用 #22 当前原参数：

```text
factgen.Random(4, Options{
    Steps: 40, Deliver: 0.55, Cutoff: 0.15, Isolated: false,
})
```

截止点为 41。在同一 `head_start = 686780cab026a3d61fc1ba4c33dd7a1ff49274e7`、精确路径 `a` 上：

- seq 27，run `01JAAAAAAA0000000000000038`：该检查声明的输入只有 `a`，记录为 error／input_missing，没有输入记录。
- seq 38，run `01JAAAAAAA000000000000003V`：同一提交、同一路径存在输入元数据。
- 两条记录均不晚于截止点，`legality.Check` 返回空问题列表。该 seed 属于 **MVP 域**，不涉及合成隔离级。

独立检查只解析原始 Check 配置与 inputs_json，按 `(head_start, path)` 关联存在性，没有用 derive 或 readycheck 结果判定是否合法。它在上述 seed 明确失败。

**与冻结规则的冲突：**[data-model §4.11.2、§7.4](../data-model.md) 规定从固定的完整 head_start 提交树中精确查找输入。提交树不因工作区变化而改变。既然同一树同一路径存在，就不能在另一次读取中把它作为“路径不存在”记录 input_missing。若读取所需对象暂时缺失或损坏，应 environment／exit 6，不写执行记录；叶对象缺失也不能当成路径缺失。

因此这组事实不能靠补写 actor、风险确认或缺失的 Scope 字段变成合法生产历史。它不是仅仅缺少校验器对某个外围字段的穷尽检查，而是当前被计入对照的具体矛盾样本。

**最小修复：**生成器为 `(完整提交, 规范化查找路径)` 维护稳定的存在性及元数据模型；input_missing 必须来自该模型的缺失路径。后续 deliver 也遵守同一模型，不能给同一个提交的原缺失路径补出条目。保留合法的 input_missing 历史，可选择其他提交或配置构造通过；不要删除该 seed 或一概取消缺失用例。为校验器补本例可由现有原始行判断的矛盾检查，并保留“其他提交上存在”及“另一条声明路径确实缺失”的合法对照。

### B1-R3：Question 关联新 Goal 的局部 ref 不符合生产提案校验，却通过合法性门禁

**位置：**

- `b3af21b` 的 [factgen/random.go](../../internal/testutil/factgen/random.go) 第 334～338 行：有 Question 关联的 addOnly slot 仍可以进入 goal 阶段并添加 Goal。
- [legality.go](../../internal/testutil/factgen/legality/legality.go) 第 585～610 行：只检查关联 Item 是否 add 及回填，没有检查新增条目的允许类型。
- 受影响手写用例：[derive/history_test.go](../../internal/domain/derive/history_test.go) 第 60～61 行也把 AboutItem 问题关联到随后接受的 Goal add，再经 legalDump 宣称合法。此构造原已存在，本轮接入的门禁没有发现它；不据此重审 M1。

**独立复现：**#22 当前原参数的 **seed 7**，截止点 27：

- Question `01JAAAAAAA000000000000001X` 于 seq 16 创建；
- 关联 Item `01JAAAAAAA000000000000001V` 于 seq 20 被接受为 Goal；
- `legality.Check` 返回空问题列表。

生产的 [proposal.CheckStatic](../../internal/proposal/proposal.go) 第 218 行明确将 Question 的局部 about 限定为 Requirement／Criterion。独立纯函数对照调用 CheckStatic，Goal 局部 ref 被拒绝，测试通过。不能通过把该 Question 改说成关联现有 Goal 来补全：原始行已经保存 proposal_item_id；关联现有正式编号会走另一种持久化关系。accepted_with_edits 也不能把同一 Item 的类型从 Requirement 改成 Goal。

**影响与最小修复：**这是现有对照样本及手写历史的非法引用，不是校验器可忽略的 actor／channel 缺项。应让有局部 Question 引用的 slot 只生成允许的新增实体类型；按实际生产类型规则检查已关联 Item，并把上述手写样例改为合法 Requirement／AC 关联。保留现有正式条目关联的既有语义，不扩大成“所有 Goal 相关问题都非法”。补 Goal 局部 ref 的拒绝反例及合法 Requirement／AC 的正例。

### B1 的整体处理要求

原三个违规种子已经修复，不需要重做那一轮算法审查。必须修的是本轮具体的误拒与漏检：

1. 不把唯一 Target 扩大到所有变更行；
2. 不把不可满足的输入身份历史计为合法；
3. Question 的局部引用类型与真实提案规则一致。

之后重跑原种子、400 个性质样本、600 个双向对照及受影响手写正反例，重新取得覆盖分布。当前两组分布只能称为“通过当前校验器的样本分布”，还不能称为全部合法事实历史的验收证据。

## 三、N1、N2、P1 的逐项复核

### N1：可以关闭

- [derive/input.go](../../internal/domain/derive/input.go) 的 Decision 新增 SeenDecisionID，空字符串表示原 NULL；ULID 前驱为非空字符串，不会与无前驱混淆。推导不读取该元数据。
- [toderive.go](../../internal/testutil/factgen/toderive/toderive.go) 第 119～128 行原样映射该字段。
- [readycheck/model.go](../../internal/testutil/readycheck/model.go) 与 [fromdump.go](../../internal/testutil/readycheck/fromdump/fromdump.go) 第 57～71 行补齐 SeenDecision、执行 worktree、执行与决定的 RequirementRevision；原有 head、tree state、结果、等级、修订、配置及证据引用仍保留。
- [toderive_test.go](../../internal/testutil/factgen/toderive/toderive_test.go) 逐表核对行数和时序；[TestConverterKeepsTheTiming](../../internal/testutil/readycheck/fromdump/checker_test.go) 检查 NULL／有前驱两种情况及执行上下文。现有包测试与三平台 CI 均通过。

字段映射的正确性不依赖生成历史在业务上是否合法；本次实际映射代码及断言足以支持关闭 N1。没有要求这些元数据加入 ready 谓词，也没有把转换器扩张为生产写入校验器。

### N2：可以关闭

[readycheck/check.go](../../internal/testutil/readycheck/check.go) 第 389 行的 GradeConsistent 无 ready 条件地比较等级。[compare_test.go](../../internal/testutil/readycheck/fromdump/compare_test.go) 第 70～73 行在每一个主样本执行，非 ready 分支也不会跳过。

[TestGradeOfADeliveryThatIsNotReady](../../internal/testutil/readycheck/fromdump/checker_test.go) 用 normal 接受仍适用、仅来源变化导致非 ready 的历史，明确断言 grade=observed；输入 none 时必须失败。它检查的是实际错误等级，不是仅检查函数运行成功。

R1～R9 与输出一致性仍分开；没有为统一两边结果改变 ready 的业务条件。N2 的直接缺口已补齐，B1 中的样本域问题不改变该帮助函数及定向断言本身的结论。

### P1：技术澄清成立，负责人确认仍待登记

[data-model §6.10](../data-model.md) 把绝对的“只有 NO_SCOPE”限定为条目级状态差距，其他项目级差距按项目级表处理。这与原项目表、§6.8、F5 R9 及原有实现相容。

新增 [TestGaps 的无 Scope／来源变化用例](../../internal/domain/derive/target_test.go) 明确要求同时有阻塞的 NO_SCOPE 与 SOURCE_CHANGED，目标仍为 not_assessed。生产 gaps 算法没有变化。因此本次是 Review 046 建议的最小措辞澄清，没有暗中放宽 ready 或引入新的屏蔽规则。

[review-resolution §58](../review-resolution.md) 如实保留“措辞已澄清，待负责人确认”。本轮只能确认技术一致性，不能把作者对“原意”的说明当作负责人已经批准。无需新增产品功能或安全例外；保留负责人对该窄范围文字的确认事项。

## 四、合法性校验器的独立性、覆盖与可补全边界

### 1. 独立实现的证据

legality.Check 实际读取 Dump 原始行，自行解码 after_json、按 seq/change_index 重建，检查审核、提案、问题、范围、Claim、Run 和 Decision 的前提。没有调用 History／Review 的构造方法，没有导入 derive／readycheck 或使用它们的 ready 结论。

共享 Dump 类型本身不是复用生成算法。baseline.CheckConfigHash、source.SetHash 用于固定规范公式；这不是来源哈希的第三套独立算法证明，既有 readycheck 仍自行重算。形式与调用依赖的独立性成立，但本轮反例说明其若干规则仍不正确／不完整。

新构造器按阶段再按 item_index 应用，显式 Target 按 Proposal 跟踪，能够拒绝分多次审核同一 Proposal 的重复 Target。pending Item 与审核绑定、Question 回填、附带 Scope 绑定新版本、父修订后的 follow-ups 等直接测试保留。未发现删 seed、Skip 非法样本、调用推导结果筛选样本的代码。

同时，B1-R1 是确定的输入域缩窄；B1-R2／R3 是确定的非法输入漏入。不能以没有主动筛选 ready 结果为由忽略它们。

### 2. 作者披露的缺失字段

校验器注释与 §58 披露未携带：Scope 的 acknowledged_waivers、seen_scope_confirmation_id，Decision 的 acknowledged_risks，以及人类写入的 actor／channel。这一披露本身准确。

对符合其余规则的事实投影，这些缺项不必自动造成非法历史：可以补当前前驱 Scope ID、当时有效的豁免集合、正常观测级验收所需的风险确认及合法人类通道。不能把“未携带”当作“已经验证”，但也无需为本轮纯函数验收提前实现生产命令或穷尽所有授权字段。

本轮没有仅凭上述遗漏提出 BLOCKER。B1-R2／R3 的样本无法靠补这些字段消除矛盾，因而是具体的不可合法补全实例。校验器无问题返回值不能作为完整生产写入合法性的穷尽证明。

### 3. 共享 seq 与非 ready 历史

共享 seq 的逐行保留、表内唯一列及各种合法写入组合检查没有被简单改成“任意重复 seq 都非法”。代表性的共享 Version／Changes／Scope 原始行控制用例通过；现有阶段、级联、reaffirm／retire／defer 测试也实际执行通过。

合法非 ready 的拒收、接受后失败、parent_revised、过期范围、来源变化等仍保留。问题在于额外排除了 B1-R1 的合法组合，以及把 B1-R2／R3 的非法组合也列入了同一个“合法”集合；不是要求每个样本都 ready，也不是因缺少 M2 集成而阻塞。

## 五、独立运行与修复前后证据

平台：Windows／PowerShell，Go 1.27.1 windows/amd64。使用真实 Go 可执行文件，设置 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，使用 -mod=readonly。没有下载依赖或访问真实远端。

### 1. 本轮现有相关包

在固定 `bd88372` 组合树执行：

```text
go test -tags nowebui -mod=readonly ./internal/domain/derive ./internal/testutil/factgen/... ./internal/testutil/readycheck/... -count=1
```

进程 exit 0：

| 包 | 本机结果 |
|---|---|
| domain/derive | PASS，2.050s |
| factgen | PASS，0.132s |
| factgen/legality | PASS，1.034s |
| factgen/toderive | PASS，0.172s |
| readycheck | PASS，0.185s |
| readycheck/fromdump | PASS，1.480s |

组合树中 #21 的源文件与 `b3af21b` 相同，因此相应包测试绑定 #21 固定源码；WP2.2 部分绑定 `bd88372`。不把组合测试误记为 #22 以 master 为基准，也不声称本机另行执行了 Linux Race Detector。

### 2. 原反例在修复前后

隔离归档 `e1498f8`，运行其三项 Review046Seed 回归：

- seed 2：seq 15，V-P03，同 Proposal 再次指定 AC-001-1，FAIL；
- seed 5：包括 seq 20 同 Proposal 再次指定 Requirement 与 AC，V-P03，FAIL；
- seed 3：seq 10，Goal add 的第 0 阶段在 Requirement 第 1 阶段之后，§5.5 order，FAIL。

断言过滤的是对应 V-P03／§5.5 规则，失败原因与 Review 046 的原始行反例一致，不是新校验器把任意共享 seq 误判。目标版本三项回归通过，相关三平台日志均显示 PASS。

本轮没有独立完整重跑作者所称旧版本全部 1,204 个非法历史统计，也没有重放最早的引擎／复核器桩阶段。相关历史数字不冒称本轮独立复验。

### 3. 隔离的独立校准用例

| 验证 | 实际结果 |
|---|---|
| 字面原始行的合法共享 seq、无接受决定控制 | PASS：legality 接受，两个算法均不 ready。 |
| 合法 Goal 级联＋显式 Requirement 更新应被接受 | FAIL：legality 报 one change per entity，B1-R1。 |
| History 应能构造同一合法组合 | FAIL：REQ-001 changes twice in one write，B1-R1。 |
| 同一字面历史直接交给两个既有算法 | PASS：R revision=2、goal=NULL、AC parent_revised，均不 ready。 |
| 当前原参数生成输入身份应能由同一不可变树解释 | FAIL：seed 4，B1-R2；legality 没有报错。 |
| 当前原参数生成的 Question 局部引用应能合法补全 | FAIL：seed 7，B1-R3；legality 没有报错。 |
| 生产 CheckStatic 拒绝 Question 的 Goal 局部 ref | PASS：确实拒绝。 |

上述期望失败是审查用例捕获缺陷的结果，不被算作项目包测试通过。临时文件只在隔离副本中创建，没有带回原仓库。

额外尝试的 CLI／SQLite 对照因隔离环境权限而在准备阶段失败，未动态验证目标写库，边界见 B1-R1。不把这些环境失败与已登记的 Windows 数据库创建竞争问题认定为同一根因，不据此关闭或升级维护项。

## 六、重新统计的证据及其适用范围

400 个性质样本、600 个主双向对照样本在进入推导之前都调用 legality.Check；非法时 Fatal 并给出 seed／trace，不是跳过。乱序、重放、diff 投影、设置不改变 ready 等性质也改用门禁后的历史。畸形测试从门禁通过的基底显式破坏，不计入合法域分布。缺少 inputs 记录的未知状态用例有单独 outsideLegalDomain 标注，没有偷算为合法正例。

CI 中实际重新统计如下，不沿用 Review 046 之前的旧分布：

| 测试及域 | 非 ready | ready observed | ready none | ready snapshot_isolated | 合计 |
|---|---:|---:|---:|---:|---:|
| WP2.1／MVP | 185 | 72 | 9 | 0 | 266 |
| WP2.1／通用 | 98 | 10 | 4 | 22 | 134 |
| WP2.2／MVP | 202 | 87 | 11 | 0 | 300 |
| WP2.2／通用 | 192 | 36 | 12 | 60 | 300 |

WP2.2 非 ready 的失败归因：

| 域 | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| MVP | 3 | 15 | 116 | 34 | 81 | 99 | 7 | 66 | 35 |
| 通用 | 5 | 14 | 108 | 31 | 81 | 94 | 10 | 48 | 36 |

这些数字已从三平台日志核对；主对照的 ready 结论与等级在所有样本上一致。它们证明当前测试确实执行、没有对照空转，**不能证明全部样本合法**：seed 4、7 已是反证。当前分布须在修复本报告问题后重新取得。

MVP 样本没有隔离级；通用域的合成隔离级用例没有被宣传为实际 MVP 能力。既有定向正反例仍有价值，不能因两个非法 seed 否定全部测试；也不能用样本总数及 ready 比例覆盖已发现的合法性漏洞。

## 七、CI 身份、实际检出树与失败历史

两个最新 CI 均为 pull_request，Attempt 1，8／8 SUCCESS。分别检查了八个 Job 的元数据及日志，实际 checkout 如下：

| CI | workflow HEAD | 实际合并快照 | 父提交 | 内容树 |
|---|---|---|---|---|
| [#21 run 39072862769](https://github.com/XKHoshizora/specmetry/actions/runs/39072862769) | `b3af21b` | `ec7e66a5116951dfb19b3cd38a034c303db910aa` | `666efe2`、`b3af21b` | `5c07bd7ad791a9b40d6025db98e0455d3acc1e41` |
| [#22 run 39072884994](https://github.com/XKHoshizora/specmetry/actions/runs/39072884994) | `bd88372` | `ed8203dfe96575e016a65632dce0ec18381e4926` | `b3af21b`、`bd88372` | `aa027eb2a5d8043e8e4617fb0e4511291e5a0699` |

合并快照的树分别与固定 HEAD 的树相同；16 个 Job 均检出了对应快照。八类 Job 为 Web、Go lint、三个平台 Go tests、三个平台 embedded-dashboard Build。

Ubuntu 实际脚本在 Linux 设置 race=-race 并执行 Go 测试；相关包通过。Windows、Ubuntu、macOS 均实际运行并通过原 seed、RandomHistories、规划顺序、follow-ups 等测试；#22 的两个新 N1／N2 测试也均实际运行，未 SKIP。这是 GitHub 平台证据，不冒称本机三平台复验。

保留失败历史：

| 运行 | HEAD | 结果 |
|---|---|---|
| [39066356194](https://github.com/XKHoshizora/specmetry/actions/runs/39066356194)，Review 046 前的旧历史 | `2e6e02b` | Go lint 失败，Windows Go tests 取消；其余历史按 Review 046 保留。 |
| [39071770560](https://github.com/XKHoshizora/specmetry/actions/runs/39071770560) | `bc27868` | Attempt 1：Go lint 失败，其余 7 个 Job 成功。 |
| [39071848979](https://github.com/XKHoshizora/specmetry/actions/runs/39071848979) | `b7d4072` | Attempt 1：Go lint 失败，其余 7 个 Job 成功。 |

本轮两次失败日志均为 review_test.go:11:46 的 unparam：base 返回的 goal 未使用；`b3af21b` 修订后 #22 合入。不能把最终 HEAD 首轮成功写成两个 PR 整个过程从未失败。

## 八、NON-BLOCKER、PRODUCT DECISION、ACCEPTED RISK

- **NON-BLOCKER：**Review 046 N1、N2 本轮可以关闭；没有另列需要扩大开发范围的新非阻塞改造。合法性校验覆盖表、全合法分布及“每实体一次变更”的文档应随 B1-R1～R3 修正，属于对应修复范围。
- **PRODUCT DECISION：**P1 的最小澄清技术上成立，但负责人确认仍待登记。本轮不批准新的产品取舍，不要求为上述三个测试基座缺陷增加产品审批，也不把修改冻结规则当作修复方案。
- **ACCEPTED RISK：**没有新增接受风险。校验器未携带字段是本轮证据的明确边界，不能因此把已证实的非法样本归为接受风险。沿用已批准的 MVP observed／通用合成隔离级区分及畸形输入选项 (a)。

未变实现、12 条 literal readings、R7／R8 的既有裁定与复核器导入／计算独立性，继续复用 Review 046。没有发现需要重开整套算法或 M1 审查的直接证据。

CLI、真实数据库读快照及高水位、Git／磁盘观测、实际写入前提与并发、Schema 输出、SQLite 复核器前端和 WP2.9 E2E 仍由后续工作包验证。本轮没有把纯函数测试或 CI 通过等同于这些集成条件通过，没有关闭任何既有维护事项。

## 九、最终回答与下一步

1. **PR #21：NO-GO；PR #22：NO-GO。** 修复主要位于共享基座，#22 自身的 N1／N2 改动没有发现直接算法回归，但依赖的合法域证据未成立。
2. **B1 未整体关闭，N1／N2 可关闭，P1 技术澄清通过且等待负责人确认。** 原三个反例修复有效，不需要重新修复已经解决的部分。
3. **当前证据不足以支持两个工作包完整的纯函数层验收。** 问题是合法域误拒／漏检，不是要求提前交付真实 CLI、数据库或 Git 模块。
4. **没有发现新的确定 ready 算法反例。** 发现三个具体证据缺口：合法级联被排除、不可变输入身份矛盾被接纳、非法 Question 局部引用被接纳。无需为此底层重构。
5. **当前不建议合并。** 最小接手任务是修复上述三处生成／校验规则，补独立原始行或约束反例，保留 seed 2、5、3、4、7 及合法共享 seq／follow-ups 控制，重跑受影响包和 CI，更新真实覆盖分布及 §58。不要删除种子、放宽规范或按双方推导结果筛样本。

取得修订审查 GO 后，才考虑先合并 #21，再把 #22 的 base 改为实际 master。届时核对实际 #21 合并内容、#22 仅剩 WP2.2 差异、共享基座与已审版本一致、没有冲突带入新逻辑，并要求对应新 HEAD／合并快照的 CI 完成；若只是无冲突的等价改基，核对树与适用性即可，不机械重做整个 M1／M2 审查。

**本轮审查结束。除新增本报告外原仓库保持不变；未提交、推送、合并，未宣布工作包验收，未授权或启动 WP2.3。下一步等待负责人安排定向修复与复审。**
