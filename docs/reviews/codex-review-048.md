# Codex Review 048：WP2.1／WP2.2 合法域修复定向独立复审

审查日期：2026-10-18。审查者：Codex。

**结论：PR #21 为 NO-GO；PR #22 为 NO-GO。** B1-R1、B1-R3 可以关闭；B1-R2 的原提交树矛盾及随机生成路径已修复，但仍遗漏“没有声明 inputs 就不能产生 input_missing”的约束，并有实际手写样例因此被误计为合法历史。仅需补这一处校验和样例，不需要修改生产推导算法或重新展开整个工作包。

## 一、固定身份、范围与原仓库状态

| 对象 | 上轮目标 | 本轮目标 | 本轮 base |
|---|---|---|---|
| [PR #21](https://github.com/XKHoshizora/specmetry/pull/21)，WP2.1 | `b3af21b076115c7934e962e068c8732f96717739` | `ca2aec2309e0e3b7e556e927f62bde7cfb9b4b88` | `666efe2c907d518b91cd86710a18989efa5dfdb4` |
| [PR #22](https://github.com/XKHoshizora/specmetry/pull/22)，WP2.2 | `bd883721df8e7c4551143b1b1c77f71945aa4a26` | `53a186c3a286ce7269126fcc9d7a619be347cb69` | `ca2aec2309e0e3b7e556e927f62bde7cfb9b4b88` |

开始和结束均通过 GitHub API 核对：两个 PR 仍为 Open，HEAD／base 均与上表一致，没有混入其他提交。冻结规范基准仍为 `666efe2`。

原工作区开始时干净，HEAD 为 `53a186c`，本报告此前不存在。只在原仓库新增本报告；所有新增校准测试、扫描和变异都在仓库外的隔离副本运行。

完整阅读 [Review 047](codex-review-047.md)，复用其对原 seed 2、3、5、N1／N2、未变算法、literal readings、独立复核器及两套转换器的结论。本轮没有重新验收 M1、审计整个 M2 或开发后续工作包。

### 实际增量

- #21：`b3af21b → ca2aec2`，16 个文件，涉及文档、Review 047 归档、共享 History／Random／legality，以及推导包测试；生产推导实现、Schema、迁移未变化。
- 其中 `42fe239` 为失败回归，`63c0e4d` 为新校验规则，`1c6705f` 为构造器与生成器修复，`4e82438` 为例外验收生成概率调整，`ca2aec2` 为状态和负责人确认记录。
- #22：`bd88372 → 53a186c` 包含继承 #21 的共享修复；自身增量是 `checker_test.go` 的级联历史检查和 `compare_test.go` 接入 CheckTrees。复核器模型、算法和转换器没有修改。
- #22 固定目标中的 derive／factgen 文件与 #21 固定目标一致。不能把 #22 的 base 记为 master，也不能把继承的修复算作第二套独立修复。

## 二、BLOCKER

### B1-R2 残余：未声明 inputs 的 input_missing 仍通过两个门禁，且实际用于“合法”决策表测试

**位置（#21 `ca2aec2`；#22 原样继承）：**

- [legality.go](../../internal/testutil/factgen/legality/legality.go) 861～872 行：只有 `len(declared) > 0` 时才检查 input_missing；随后又允许这种执行没有 inputs_json。
- 同文件 1008～1011 行：CheckTrees 对 `len(inputs) == 0` 直接跳过。
- [criterion_test.go](../../internal/domain/derive/criterion_test.go) 53～55 行：现有“4: input_missing counts as a failure”直接对默认检查写 input_missing。
- [helpers_test.go](../../internal/domain/derive/helpers_test.go) 54 行：这个默认检查是 `check("go test ./orders")`，没有声明任何 inputs。

**契约与影响：**[data-model §7.4 第 2 步](../data-model.md) 在声明 inputs 后读取这些路径；路径不存在才记录 input_missing。零个声明输入不可能存在“至少一个声明输入缺失”。补 actor、确认信息或树见证都无法把上述历史补全为合法写入。

这不是新发现的生产算法错误，也不是说当前 400／600 个随机样本都非法。问题是本轮明确要求的 input_missing 合法性规则没有完整实现；实际手写决策表仍在合法性门禁通过后使用不可合法产生的记录。因此不能把“所有手写样例合法、该规则已有可靠门禁”作为完整验收证据。

**独立复现：**在 `53a186c` 的隔离副本中：

1. 创建带一个自动化 AC 的历史；Check 有合法 command、workdir、timeout，但 Inputs 为空。
2. 在一个非空 HEAD 上记录 `error / input_missing`，不记录 inputs_json。
3. 对 Dump 调用 legality.Check；再提供该 HEAD 的空树调用 CheckTrees。
4. 两者都返回零问题。审查用例按“应拒绝”断言，实际 FAIL：

```text
both gates accept input_missing although the check declares NO inputs;
no missing path can exist
```

另在隔离副本给 Check 增加唯一一条“input_missing 且声明列表为空则拒绝”的校准规则，再运行原仓库的该决策表子用例，实际在合法性门禁处失败：

```text
TestCriterionDecisionTable/4:_input_missing_counts_as_a_failure
the fixture is not a legal history:
seq 4: commit tree: review048 mutation: input_missing without a declared input
```

因此不是只有审查者临时捏造的输入才会触发：已交付测试确实依赖这一漏检。变异只存在于隔离副本，没有修改原仓库。

**最小修复：**

1. Check 对 input_missing 无条件要求至少一个声明输入；非空列表再按现有规则核对可见树矛盾。联合门禁不能依赖 CheckTrees 的“无输入直接跳过”来补这个前提。
2. 给上述决策表样例明确声明一个缺失路径，再记录 input_missing；保持其 `failing / check_failed` 的业务断言。
3. 补“零声明输入＋input_missing 必须拒绝”的负例，以及“确有声明路径缺失应接受”的正例。CheckTrees 可以明确以通过 Check 的记录为前提，或独立拒绝这种不可能的记录，不要求重复全部生产校验。
4. 重跑受影响包与 CI。保留当前合法随机样本、种子和门槛，不必修改生产状态规则或降低现有断言。

**裁定：**B1-R2 原 seed 4、同提交条目冲突和随机生成修复有效；上述小范围残余仍阻止其整体关闭。#22 自身两处测试改动未发现算法错误，但依赖相同共享门禁与尚未通过的 #21，本轮联合验收仍为 NO-GO。

## 三、B1-R1：可以关闭

核对 [review.go](../../internal/testutil/factgen/review.go)、[random.go](../../internal/testutil/factgen/random.go)、legality，以及生产 [baseline.Apply](../../internal/baseline/apply.go) 的必要调用链：

- 显式 target 唯一性继续按同一个 Proposal 判断，并覆盖分次审核；级联行不再被当作第二个显式 target。
- 每条变更在前面变更留下的状态上检查。Goal 退役后，Requirement 的清空 Goal 级联保持 editorial／原修订，随后同一 Requirement 可以接受显式 material 更新，或退役并级联退役 AC。
- 所欠级联由当时 active 状态计算，必须紧跟原因并完整写出。级联／退役保持业务取值；重新关联已退役 Goal、修改被级联退役的 AC 等仍拒绝。
- change_index、after_json、共享 seq 和父子修订全部保留；没有改用输入排列或 ULID 判断应用先后。
- “每实体至多一次条目变更、一次级联、一次后续处理”在现有操作模型下有依据：同一审核只处理一个 Proposal，V-P03 限制显式 target；Requirement 只有一个 Goal，AC 只有一个父需求；follow-up 本身不得重复，也不能再处理已由接受条目处理的 AC。没有把该概括重新扩大为“一个实体只有一条 change”。

独立复用了上轮按字面原始行构造的共享 seq／级联控制，用当前门禁验证：正常共享历史、Goal 级联后显式更新均接受；History 不再 panic；原始行直接推导得到 Requirement revision=2、goal=NULL、AC 原因 parent_revised，两边均不 ready。

本轮新增的原始行正反例另覆盖“级联后退役”、缺失或延后级联、重复显式 target、已退役实体操作、改变级联取值及重复 follow-up。其断言检查具体规则，并对结构性连带错误列出许可集合，非只检查出错或不 panic。

两套新增算法测试实际检查：

| 截止点 | 推导与等级 | 复核器 |
|---|---|---|
| 编辑前 | ready，observed | ready，无失败条件 |
| 同次编辑后 | R revision=2、goal=NULL；AC requirement_revision=1，parent_revised；needs_reverification，none | 仅 R6 失败，none |
| reaffirm 后并重新确认范围 | AC requirement_revision=2，原接受重新适用；ready，observed | ready，observed |

这与 §5.2、§5.5、§6.3 的父修订优先规则一致。不是仅凭“两边一致”认定正确。

## 四、B1-R2 已修复部分、树见证与证据边界

### 1. 生成与记录

Random 在任何历史操作之前，为五个提交分别创建固定 Trees。run 从 head_start 对应树读取输入；missingAt 对声明路径逐一判断，不再独立随机抽取 input_missing。空声明列表不会被 Random 生成 input_missing。

deliver 不修改既有树，也不给同一个提交补路径；它通过一次合法人工编辑，把在 S 上缺失输入的 AC 改成 S 可执行的 Check。该操作产生正常 material 更新与修订，然后才能生成执行或合法例外验收。没有修改过去执行或根据最终推导结论修补记录。

### 2. 校验与正反对照

entries 按完整提交和规范化查找路径比较 mode／type／OID，不限同一 AC 或同一次执行。input_missing 对已声明路径检查是否全部被其他执行证明存在。规范化为反斜杠转斜杠后 path.Clean；保留原声明路径的持久化语义。

现有正反例实际覆盖另一提交存在、另一声明路径缺失、跨 AC、./a 与 a、脏工作区及 context_changed。脏工作区和最终 HEAD 变化不改变开始提交的树，故没有被误当成条目可变的理由。

审查方另外独立修改：

- 树见证的 mode、type、OID；
- 删除见证中的实际存在路径；
- 给 input_missing 的见证补齐原来缺失的路径；
- 单独篡改执行的输入记录。

全部被 CheckTrees 检出。未篡改的别名路径和合法缺失正例通过。说明 CheckTrees 确实比较见证与记录，不只是检查 map 存在。

### 3. 两类保证不能混用

- 不带树见证的转储：可检出记录之间的可见矛盾，未被观测的路径仍需存在一种合法补全；不声称获得真实完整提交树。
- 带 Trees 的生成历史：可核对该固定树是否解释执行记录及缺失原因。
- Trees 是生成器声明的见证，不是实际 Git 读取结果；本轮没有验证生产 Git Reader、Git 对象可用性或后续真实执行前端。

上述修复和证据成立；第二节的零声明输入漏检不因此消失。

## 五、B1-R3：可以关闭

- Random 把有 Question 指向的新增 Item 限制在 Requirement／AC 阶段；History 对最终接受为 Goal 或非 add 的局部关联拒绝。
- legality 检查 Question 的 Item 属于同一 Proposal；接受后必须对应 Requirement／AC add；仍 open 时按接受结果回填，先关闭后接受则按既有持久化与读取规则处理。
- 关于既有实体的问题要求 active Requirement／AC。待审、拒绝 Item 的问题保持 open、无伪造正式关联的合法控制仍通过。
- TestStateAtCutoff 已改为合法 Requirement 局部引用。新增测试覆盖 Requirement／AC 局部引用、拒绝及待审、既有 Requirement、其他 Proposal、Goal 局部引用和既有 Goal。
- 审查方直接调用生产 proposal.CheckStatic，独立确认 Requirement／AC 局部 about 可接受，Goal 局部 about 拒绝；同时核对 proposal.v1 的 about 只允许 Requirement／AC key 或局部 ref。这提供了不依赖新 legality 判定的写入规则依据。
- “about a goal”保留为域外鲁棒性用例，未扩大合法写入语义。fixture 的域外标签反向检查只是辅助；其域外性质另由上述 Schema／CheckStatic 证明，不靠同一个错误假设循环论证。

Dump 未携带待审／拒绝 Item 的原 op、type、target，这个限制在注释、§59 和 PR 描述中均披露。当前投影可以把这些 Item 补全成合法 Requirement／AC add；本轮未发现与其他可见事实冲突、无法合法补全的这类样本。不要求提前实现生产前端，也不将“字段缺失”表述为已验证。

## 六、修复前后及统计含义

### 1. 本轮独立重放

在隔离归档 `42fe239` 上执行回归：

- 原始行级联正例被旧校验器误拒：one change per entity；
- History 的级联后显式 update 发生 `REQ-001 changes twice in one write`；
- seed 4 保留原 seq 27、head 686780ca、唯一输入 a 的树矛盾；
- seed 7 保留原 Goal Item 的非法 Question 关联。

在隔离归档 `63c0e4d` 上：

- 原始行级联正例、新增级联／提交树／Question 规则控制通过；
- 旧生成器的 seed 4、7 现在被新校验器明确拒绝；
- 全量扫描仍失败，符合“先加强校验器，后修生成器”的提交顺序。

最终版本的上述原回归和控制全部通过；第二节是额外发现且实际失败的剩余校准用例。

### 2. 178／1204、76、103

审查方另写只读扫描，直接从 after_json 取得声明输入、从执行记录建立提交路径元数据集合、从 Item 对应变更判定 Question 类型；不读取 derive／readycheck 结果，不调用作者测试内的扫描函数。

对 `63c0e4d` 的旧生成器实际得到：

| 组 | 不可合法补全历史／样本数 |
|---|---:|
| derivation | 60／400 |
| comparison | 93／600 |
| shuffle | 1／4 |
| malformed base | 24／200 |
| 合计 | 178／1204 |

其中树矛盾 76 个，Goal Item 问题 103 个，**有 1 个历史同时命中**，所以并集为 76+103-1=178。1204 是四个参数集合中的样本次数，不能当成 1204 个互异 seed。

本轮扫描与新校验器在这 1204 个样本上的分类零分歧。该数字证明原两类违规在这些样本上被找到；不单凭计数相同证明规则正确，也不表示旧校验器穷尽全部写入不变量。接受／拒绝控制与第二节反例说明独立校准仍必要。

### 3. 0.2 → 0.3

`4e82438` 只改变 deliver 中“没有合格执行、允许例外接受”分支的抽样概率。该分支仍受 §7.3 前提及 legality 门禁约束。没有删除种子、降低样本／覆盖门槛、循环生成直到 ready、按最终结果过滤或重试样本；生成器不调用两套算法。

因此是合法操作分布调整，可以保留。为达到最低覆盖数量而调整概率应如实披露；这些数字不能解释为生产分布、可靠性比例或穷尽证明。

## 七、实际测试与恢复后的覆盖

本机：Windows／PowerShell，Go 1.27.1 windows/amd64；使用 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local、-mod=readonly，没有下载依赖或访问真实远端。

固定 `53a186c` 组合树执行：

```text
go test -tags nowebui -mod=readonly ./internal/domain/derive ./internal/testutil/factgen/... ./internal/testutil/readycheck/... -count=1
```

| 包 | 本机结果 |
|---|---|
| domain/derive | PASS，24.318s |
| factgen | PASS，20.337s |
| factgen/legality | PASS，21.367s |
| factgen/toderive | PASS，13.122s |
| readycheck | PASS，0.131s |
| readycheck/fromdump | PASS，16.430s |

derive／factgen 文件与 #21 固定源码一致，WP2.2 部分绑定 #22 固定源码。这些是仓库已有测试通过；第二节新增审查断言确实失败，二者不能相互覆盖。

隔离校准的结果：

| 用例 | 结果 |
|---|---|
| 字面原始行共享 seq、级联后显式更新；History 构造；两算法父修订状态 | PASS |
| 独立更改树见证 mode／type／OID／存在性，以及执行记录 | PASS，矛盾均检出 |
| 生产 CheckStatic 的 Question 局部引用类型对照 | PASS |
| 无声明 inputs 的 input_missing 应拒绝 | FAIL，两个门禁均接受 |
| 只补该拒绝条件后运行原决策表样例 | 预期 FAIL，暴露真实样例不合法 |

400 个性质样本、600 个主对照样本都在进入计算前运行 Check 和 CheckTrees；非法时 Fatal，不是 skip。手写 fixture 使用 Check；域外／畸形用例不计入合法域统计。后者仍需第二节的小修正才能完成“所有手写正例合法”的声明。

三平台最新日志均重新取得以下分布：

| 测试／域 | 非 ready | ready observed | ready none | ready snapshot_isolated | 合计 |
|---|---:|---:|---:|---:|---:|
| WP2.1／MVP | 198 | 61 | 7 | 0 | 266 |
| WP2.1／通用 | 87 | 14 | 4 | 29 | 134 |
| WP2.2／MVP | 209 | 79 | 12 | 0 | 300 |
| WP2.2／通用 | 194 | 29 | 12 | 65 | 300 |

WP2.2 非 ready 失败归因：

| 域 | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| MVP | 5 | 18 | 121 | 35 | 97 | 105 | 7 | 64 | 29 |
| 通用 | 5 | 16 | 118 | 31 | 84 | 99 | 12 | 62 | 25 |

生成组合也保留：400 个性质样本分别有 12 个级联后条目变更、35 个 input_missing、81 个接受后回填、31 个拒绝 Item 问题；600 个对照分别为 21、58、109、56。计数单位是“包含此组合的历史”，不是操作条数，各列可重叠。

乱序、重放、截止点、ChangesSince、设置不改变 ready、R1～R9 定向正反例、各样本 GradeConsistent 和原 seed 2／3／5 均实际运行通过。MVP 不含隔离级执行；通用域合成隔离级不被宣传为 MVP 能力。旧分布只保留为历史记录，不继续支持合法域验收。

未发现本轮随机生成历史重现原提交树／Goal Item 矛盾；本报告的剩余阻塞不等于推翻所有恢复后的测试证据。

## 八、CI、归档与状态登记

分别核对两次运行、八个 Job 元数据及各自日志：

| CI | workflow HEAD | Attempt／结果 | 实际检出合并快照 | 父提交 | 内容树 |
|---|---|---|---|---|---|
| [#21 run 39079396186](https://github.com/XKHoshizora/specmetry/actions/runs/39079396186) | `ca2aec2` | 1，8／8 SUCCESS | `1a0e77551f996b99e788d3f7173c79cf314ce914` | `666efe2`、`ca2aec2` | `34b33782944124bf34946bb239749a98ad036e2a` |
| [#22 run 39079421743](https://github.com/XKHoshizora/specmetry/actions/runs/39079421743) | `53a186c` | 1，8／8 SUCCESS | `41eb507a8781317478829ad17d7c41cada3344e4` | `ca2aec2`、`53a186c` | `aed530d2c9ee629882cc06f78abe4717f459be4d` |

两棵合并快照树分别与固定 HEAD 相同；16 个 Job 的日志均指向对应快照。八类 Job 为三个平台 Go tests、三个平台 embedded-dashboard Build、Go lint、Web。

Windows、Ubuntu、macOS 实际运行并通过新增级联／树／Question 测试及原 seed 回归，关键测试没有 SKIP；#22 的级联状态／等级检查和 600 个对照也均实际通过。Ubuntu 日志中的 Linux 分支设置 -race，并完成 Go 测试。本机没有运行 Linux Race Detector，不冒称本机三平台复验。

保留失败与取消历史，不把最终 HEAD 首轮成功写成整个 PR 从未失败：

- [39066356194](https://github.com/XKHoshizora/specmetry/actions/runs/39066356194)，旧 `2e6e02b`：lint 失败，Windows Go tests 后续取消。
- [39071770560](https://github.com/XKHoshizora/specmetry/actions/runs/39071770560)，`bc27868`：lint unparam 失败，其他 7 个 Job 成功。
- [39071848979](https://github.com/XKHoshizora/specmetry/actions/runs/39071848979)，`b7d4072`：同一共享 lint 问题，其他 7 个 Job 成功。

Review 047 归档在 `da40a05` 与 `ca2aec2` 的 blob 相同（`098ac5347c5e2becd5849bc7fc2c3cf905781a17`），受审身份仍指向上轮固定目标，没有改写历史结论。

[review-resolution §58／§59](../review-resolution.md) 正确把旧分布标为被新证据取代，N1／N2 已关闭，R1～R3 为修订待复审，未自行宣布 B1 关闭或工作包验收。P1 的负责人确认仅覆盖无 Scope 时条目差距与项目级差距的措辞澄清，不改变 ready 谓词、不批准其他草案；与本次用户确认范围一致。

## 九、NON-BLOCKER、PRODUCT DECISION、ACCEPTED RISK

- **NON-BLOCKER：**没有新增需要扩大本轮范围的独立改造。零输入漏检、对应手写样例和覆盖声明的修正属于同一个 B1-R2 残余，不能另归为已接受风险。
- **PRODUCT DECISION：**P1 已按负责人确认登记，不再要求重复决定。三个合法域修复无需新增产品审批；本轮未发现新的必要产品决定。
- **ACCEPTED RISK：**没有新增。沿用已批准的 MVP observed／通用合成隔离级区分及畸形输入选项 (a)。Dump 缺失的写入字段、生成器树见证和有限样本属于证据边界，不是擅自接受新的核心风险。

CLI、真实数据库读快照／高水位、Git／磁盘读取、生产写入前置条件、Schema 输出、SQLite 复核器前端和 WP2.9 E2E 均未在本轮动态验证，仍归后续工作包。本轮没有关闭 M-1、doctor GIT_TRACE 等既有维护项，也没有将纯函数 CI 扩展为集成验收。

## 十、最终回答与接手范围

1. **PR #21：NO-GO；PR #22：NO-GO。** #22 自身修改未发现直接算法回归；其合并与联合验收依赖共享基座的剩余修复。
2. **B1-R1、B1-R3 关闭；B1-R2 部分关闭。** 原 seed 4 和不可变树修复有效，剩余是零声明输入的 input_missing 被误接纳，并已用于实际合法域样例。
3. **主要生成／双向对照证据已恢复，但尚不足以宣称两个工作包完整纯函数层验收条件全部满足。** 只需修正第二节的门禁和样例，保留当前其他有效证据，不要求重新构造整套测试体系。
4. **没有发现确定的生产推导算法反例或复核器新算法缺陷。** 发现的是具体、已复现的合法域证据缺口，不是未来 Git Reader／CLI 未实现的问题。
5. **当前不建议合并。** 最小接手任务：拒绝零声明输入的 input_missing，把原决策表样例改为真实声明路径缺失，补正反控制，运行相关包与对应新 HEAD CI，再作这一点的定向确认；不修改冻结规则或降低覆盖门槛。

取得 GO 后，先由负责人合并 #21；再将 #22 base 改为实际 master。届时核对 #21 实际合并提交与已审共享代码一致，#22 只剩 WP2.2 自身内容、继承的 History／Random／legality／toderive 没有丢失或冲突改写，两套算法及转换器未出现未审增量；核对新 HEAD、合并快照、父提交、树适用性和完整 CI（含 Ubuntu Race Detector）。等价改基不必机械重做 M1 或全部 M2；有实际新差异则只审相应影响。

**本轮结束并暂停。除新增本报告外原仓库不变；未提交、推送或合并，未宣布工作包验收，未授权或启动 WP2.3。**

