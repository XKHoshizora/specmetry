# Codex Review 045：PR #20／Review 044 B1 定向独立复审

- 日期：2026-10-18（Asia/Tokyo）。
- 仓库：`XKHoshizora/specmetry`；[PR #20](https://github.com/XKHoshizora/specmetry/pull/20)。
- PR 固定基准：`258fc50f5e94354892737614ab61eea564178e7c`。
- Review 044 目标／本轮增量起点：`4ef79c8f2c309393a94f58897d1f48cad15fee6f`。
- 本轮固定目标：`d4ba0a9b29995e21536efb8fcd7a089c17f96703`。
- CI：[run 39056682275](https://github.com/XKHoshizora/specmetry/actions/runs/39056682275)，Attempt 1，8/8 SUCCESS。
- 范围：完整阅读 [Review 044](codex-review-044.md)，仅复核 B1 的修订及直接关联的规范、Schema、示例、验收和状态记录。复用已通过的 F1～F3、seq／主要历史规则、F5 ready 谓词；不重新验收 M1。
- 方法：固定差异、规范与必要调用链的静态核查，GitHub 身份及 CI 日志核查，Windows 本机现有契约测试。没有实现 M2。

## 结论

**GO。Review 044 B1 可以关闭。PR #20 的首批规范与契约具备合并条件；完整 F4／F5 具备冻结条件。**

本轮补齐了参与推导的三项项目设置、一次查询的取值边界、原输入重放、差异计算参数及两种输出的白名单字段。原来未输出 skip 设置、因此无法重放设置变化前结果的反例，已在规格、Schema 和验收预期上得到对应修正。

没有新的 BLOCKER。保留一项非阻塞措辞建议：设置不会改变阻塞差距的存在性与类型，但 stage 可以改变诊断原因码，不能把这句话扩张为完整差距 JSON 逐字段不变。

**本结论是规范与契约审查 GO，不是 M2 实现验收，也不授权开工。** 除合并结果核验、负责人登记规格冻结并明确授权外，没有发现新的完整首批规范门槛。获授权后的实施仍须遵守 F6／测试先行及独立复核器要求；本轮没有提前完成这些实现工作。

## 一、身份、实际增量与归档

开始及结束核查时，PR HEAD 均为指定目标，base 均为指定基准，PR 保持 Open；本地 HEAD 也为指定目标。没有混入新增提交。开始时工作区干净，`docs/reviews/codex-review-045.md` 不存在。

增量包含三个提交：

| 提交 | 内容 |
|---|---|
| `a3767776c461654dc779842306cb764d42325ab5` | 归档 Review 044 |
| `d8b61e7db144036ff7009b0cf56134b1f73f7d46` | Schema、示例、SCH-157～165 |
| `d4ba0a9b29995e21536efb8fcd7a089c17f96703` | 完整输入、重放、验收、TD-27、既有 N1 及活跃记录 |

`4ef79c8 → d4ba0a9` 实际修改 18 个文件：AGENTS，七份规范／状态文档（acceptance-criteria、architecture、data-model、implementation-plan、mvp、prd、review-resolution），Review 044 归档，两份 Schema，以及七个示例／反例文件。PR 相对 master 基准累计为 20 个文件；这两个统计口径没有混用。

没有生产代码、测试实现、迁移或工作流改动；`init.v1`、go.mod／go.sum 也没有变化。`git diff --check` 通过。

Review 044 的全文与受审身份保持，归档后至本轮目标没有改动；归档文件的 Git blob 为 `001325d0a7982a24c568a6962665c4a2e178ada2`。[§56](../review-resolution.md#L2747)保留原 NO-GO 及其适用目标 `4ef79c8`，处理状态是“已修订，待复审”，没有以作者修订代替独立关闭 B1。历史审查原文未被改写。

## 二、BLOCKER

**无。Review 044 B1 已解决，限本轮规范和契约层面。**

### 1. 完整显式输入已补齐

[data-model §6](../data-model.md#L555)、[§6.13](../data-model.md#L838)、[architecture §7](../architecture.md#L759)现在一致列出：事实、S、来源观测、项目设置，以及仅用于工作区视图的上下文。参数也明确包含截止点、diff 起点和适用的 session 选项。

三项设置覆盖现有推导规则中参与计算的项目设置：

| 设置 | 规则中的影响 |
|---|---|
| stage | AC 进度；由进度产生的部分原因码和 conflict 警告；INVENTORY_INCOMPLETE；初始化进度 |
| sources_skipped | 初始化进度 |
| inventory_skipped | 初始化进度与 INVENTORY_INCOMPLETE |

excerpt_storage 决定保存行为；commit_ref_mode 作用于扫描时形成的事实，不由推导函数重新解释。这两项不需要透传为当前推导设置。stage 对完整诊断对象的措辞建议见本轮 N1，但其不改变交付状态、ready 及阻塞差距存在性的结论正确。

[§6.13.1 第 925～929 行](../data-model.md#L925)要求调用方一次命令只取得一份当前值；设置与事实、as_of_seq 在同一读事务中取得，后续推导与输出复用该值，不能再次查询设置。数据库中未设置的 skip 键由调用方读成 false，与“重放文档缺少这个字段”不是一回事；推导端不能自行用 false 填补缺项。

SQL 读事务保证的是数据库输入的一致读快照。Git 和磁盘由调用方另行观测，规定只取一次不等于它们与 SQL 有共同原子时刻；修订没有新增跨系统原子快照或持续锁定磁盘的承诺。实现时不得从“同一读事务”推导出此类额外保证。

来源观测的缺项与 NULL 已明确分开：缺少某来源的观测是输入错误；有该来源记录且 current_sha256=null 才表示观测到文件缺失，并产生相应 SOURCE_CHANGED。两种输出也要求观测字段存在。

### 2. 正常查询与重放形成了闭合契约

[重放表及拒绝规则](../data-model.md#L931)逐项明确了输入来源：

- project.id 和 as_of_seq 定位原项目的事实历史；原状态按 F4 重建，不能拿当前投影冒充历史。
- S 使用原输出记录的 snapshot，不重新解析 ref。
- 来源使用原输出中的 ID／观测哈希，覆盖该截止点的来源集合；没有观测不能改成 NULL。
- 设置使用原输出的 stage／两个 skip 值，不读取当前设置。
- 需要工作区视图时，保留原 worktree、HEAD、tree_state 和适用的 compare_from／changed_paths。
- diff 起点由 diff_since_seq 保留。

缺失项目、必要事实、设置或观测时报告缺项；不自动补当前值，不降低截止点。输出是计算上下文和读取模型，不是事实备份；项目已 purge 或只有较早备份时，不能声称单凭输出恢复了事实。

保证对象是业务推导结果及适用的工作区视图，不是展示字段或数据库备份的逐字节复制。project.name 等展示字段不参与判定；gaps 不需要工作区上下文。`init.v1` 的引导提示继续保持 M1 的既有契约，没有被偷偷纳入需要新增历史的重放接口。

Review 044 的具体反例现在有对应处理：设置变更后仍可使用原输出记录的 sources_skipped=false 重放 initialized；一次新的查询使用 true 得到 sources_registered。两者不再被错误称为同一组完整输入。M1 的 [init 更新路径](../../internal/project/init.go#L474)没有改动，也没有为这次设置变化虚构 seq。

### 3. “同一版本的推导规则”没有暗中新增跨版本重放系统

[第 940 行](../data-model.md#L940)限定同一规则版本，是固定同一纯函数的前提。原来的确定性要求针对同一推导规则和输入，并没有明确承诺任意未来版本都能执行旧规则。NFR-11／ST-14 也没有被改为允许同一函数在同一输入下给出不同结果。

WP2.1／WP2.2 的重放测试应在同一测试构建、同一套固定规则下计算原结果和重放结果；不能在两次计算之间换用另一套规则。对外引用重放证据时，也应保留对应的二进制／源码版本身份，不能跨版本套用结论。

规范已经把这一前提写入 ST-14 所引用的 §6.13.1。没有必要为关闭本轮 B1 新建跨版本规则存档服务、版本兼容转换或重放命令；也没有因此产生需要负责人额外批准的产品取舍。

## 三、--diff-since 的专门裁定

### 1. 两个截止点共用输入，设置变化不进入事实历史

[第 967～971 行](../data-model.md#L967)和 [ST-11](../acceptance-criteria.md#L212)要求同一次命令固定 S、项目设置及其他已取得的当前值，只改变事实截止点。比较期间即使另一连接修改了设置，两边与输出也不重读。

只有设置变化时，当前事实序号不变；以该序号为起点，changes_since 为空。它表示在同一组本次当前值下重算事实差异，不是回放过去实际展示过的页面；原结果重放则必须使用原来那组当前值。三种操作没有混为一谈。

### 2. 起点不需要旧来源观测，成立条件是只计算 AC 投影

[第 968 行](../data-model.md#L968)已将 changes_since 明确限定为 **AC 的 delivery_status／progress**。结合 §6.3 与 §6.5，这两个结果不读取来源观测和工作区上下文；来源变化影响的是目标层面的差距／交付聚合，不能据此反向改变 AC 状态表。

因此这里定义的是差异所需的 AC 投影语义：起点只需历史 AC／相关事实、共用 S 和项目设置，不要求生成起点完整的 status／gaps。虽然正文沿用“两次推导”这一简称，第 968 行已明确其所需结果和不需要旧来源观测的依据；无需为术语再要求另一次阻塞修订或指定内部函数名称。

**合法历史核对：**

1. 起点 t1 的来源集合为 A；
2. 之后移除 A、登记 B；当前 t2 的来源集合为 B；
3. 本次输出只记录 B 的观测，不保留 A 的观测；
4. t2 的完整推导使用 B；t1 的 AC 投影使用 t1 的事实与共用 S／设置，不读取 A 的观测；changes_since 仍可计算和重放。

这不构成缺项容错。如果实现改为调用“生成完整状态／差距并检查全部来源观测”的推导入口，那么仅传 B 的观测给 t1 会同时存在缺 A／多 B，必须被完整输入校验拒绝。实现只能选择必要的 AC 投影，或另行满足完整入口的输入要求；**不能捕获缺项错误后忽略，也不能把缺项当 NULL。**

另外，重放整个 t2 的 status／gaps 仍然需要 B 的观测；第 971 行列出的较小输入集合只足以重算 changes_since，不能用来重放完整输出。该区分使“完整推导缺项拒绝”与“起点不读取旧来源观测”同时成立。

这是一项源码未实现前的规格审查与逻辑核对，没有声称动态执行了上述 M2 场景。实现 ST-11 时应保留这条来源集合变化用例，防止错误接线到完整入口。

## 四、NON-BLOCKER

### N1（本轮）：收紧“设置不改变阻塞差距”的措辞

[data-model 第 850 行](../data-model.md#L850)及 [F5 测试前端说明](../acceptance-criteria.md#L375)说设置不改变阻塞差距。该结论应理解为：不改变差距的存在性、类型和阻塞性；不应扩展为完整诊断对象逐字段不变。

严格的规则反例：已有自动化 AC 和检查配置，S 有效，没有 Claim、关联提交、执行或决定。stage=docs_only 时，§6.5 给出 not_started；stage=in_development 时给出 unknown。按 §6.3 第 11 条，AC 仍为 in_progress，但原因码分别是 not_started／unknown；状态差距类型仍是 NOT_VERIFIED。携带该诊断原因时，差距对象也不应被要求逐字段相等。

最小建议是将文字收紧为“设置不改变交付状态与阻塞差距的存在性、类型和阻塞性，但可改变诊断原因码”。F5 的实际断言仍只要求设置不同的输入对具有相同 ready 结论，这项断言正确；不要新增完整 gaps JSON 必须相同的断言。

该处不导致错误 ready，不破坏重放输入闭合性，也不需要改变已通过的 ready 谓词或产品语义，故不阻塞本次 GO。

**Review 044 的既有 N1 是另一事项。** 提交关联生命周期已登记到 [implementation-plan §3](../implementation-plan.md#L40)与 data-model §4.12／§6.13，在 WP2.3／Q7 冻结时处理。本轮没有将它升级为首批阻塞，也没有提前批准补写方案。

## 五、Schema、示例与验收核查

### 1. 输出结构与白名单

- [status.v1](../../schemas/status.v1.schema.json#L30)的 project 保留既有字段，并必填 sources_skipped、inventory_skipped；stage 为限定枚举，skip 值为布尔值。
- [diff_since_seq](../../schemas/status.v1.schema.json#L218)为非负整数，与 changes_since 通过双向 dependentRequired 成对出现；任一单独存在都会拒绝。
- [gaps.v1](../../schemas/gaps.v1.schema.json#L8)新增必填 project（ID 与三项设置）和 sources（每个来源的 ID、必填且可为 NULL 的观测哈希）。
- 两种 project 及新的观测条目均拒绝额外属性，不输出 settings_json、excerpt_storage 或其他无关设置。
- Schema 只能验证结构；来源 ID 集合与历史来源集合的一致、观测不遗漏／不冲突、设置与实际计算相符、diff 两个截止点的关系及结果正确，仍由语义校验和对应测试负责。

### 2. 示例不能代替同一完整输入的语义测试

核对了新增 status.diff-since、两份 gaps 和已修改 status 示例。新 diff 示例记录了起点 400、当前 412、AC-002-2 从 in_progress 到 ready_for_review；展示的检查执行位于起点之后，存在符合该变化的合法事实历史。gaps 的 failing／claimed_without_evidence 与其非 ready 结果相容；ready-observed 无阻塞差距并保留 observed 等级。

同项目、as_of_seq 和 S 相同，不足以证明示例具有相同的完整输入。这里的 JSON 不是完整事实备份；status.minimal 与 status.exception 本身也展示了不同的决定事实。本轮没有把独立示例视为同一个数据库历史的多份同时输出，也没有通过 gaps.json 改用另一个项目 ID 来替代业务语义核查。

示例与反例证明结构可表达、约束有效；真正的同事实／同当前值重放对照仍须由 ST-14 等测试提供，不能从示例全部合法推断推导实现已正确。

### 3. 验收预期覆盖本轮保证

| 规格／验收 | 实际证明目标 |
|---|---|
| ST-11 | 同命令共用当前设置；中途修改不影响结果；设置单独变化不增加 seq／changes_since；保留 diff 起点 |
| ST-14 | 相同完整输入的纯度；设置变化后按原输入重放；缺设置、观测或事实拒绝，不用当前值补齐 |
| ST-15 | 设置和事实同一读快照；输出前修改不改变已取值；新查询反映新设置；白名单输出 |
| E2E-6 | 用原完整输入与原截止事实重放，不只凭序号声称可复现 |
| F5 测试前端 | 同一份当前值记录供两套独立实现使用；生成设置取值组合；双方不自行观测／查询；设置不同但其余输入相同时 ready 一致 |

这些是未来 WP2.1／WP2.2 及集成工作的可执行验收预期，不是本轮已运行的 M2 测试。R1～R10 的业务判定没有被修改。

## 六、实际运行与 CI 证据

### 1. 本轮独立本机验证

Windows／amd64，使用已经安装的 Go 1.27.1，关闭模块下载与自动工具链切换，直接调用真实 go.exe 运行：

```text
go test -tags nowebui -mod=readonly ./internal/contract -count=1 -v
```

**PASS，退出码 0。** 全部 Schema 编译通过，40 个合法示例通过，165 个非法反例被拒绝；该包现有错误信封、Pointer 和不回显输入测试也通过。没有新增测试程序、下载工具链或修改代码／配置。

[TestInvalidCasesFail](../../internal/contract/contract_test.go#L126)在每个 patch 前先验证基础示例合法，然后要求 patch 后返回带 details 的 invalid_input。本轮逐项核对了新增反例与对应约束：

| 反例 | 基础／变异 | 拒绝依据 |
|---|---|---|
| SCH-157 | status.minimal；删除 sources_skipped | project.required |
| SCH-158 | status.minimal；inventory_skipped 改为字符串 | boolean 类型 |
| SCH-159 | status.minimal；增加 settings_json | additionalProperties=false |
| SCH-160 | status.diff-since；删除 diff_since_seq | changes_since 的 dependentRequired |
| SCH-161 | status.minimal；单独增加 diff_since_seq | 反向 dependentRequired |
| SCH-162 | gaps.ready-observed；删除 project | 根 required |
| SCH-163 | gaps.ready-observed；删除 sources | 根 required |
| SCH-164 | gaps.ready-observed；删除 current_sha256 | 观测条目的 required；不等同于允许的 NULL |
| SCH-165 | gaps.ready-observed；增加 excerpt_storage | project.additionalProperties=false |

现有测试没有逐例断言验证器内部 keyword。本轮用基础示例预校验、单字段变异及对应 Schema 的静态检查确认拒绝依据，没有冒称运行了作者未提交的临时检查程序。

### 2. CI 身份与日志

| 项目 | 核对结果 |
|---|---|
| Event／HEAD | pull_request／`d4ba0a9b29995e21536efb8fcd7a089c17f96703` |
| Attempt | 1，previous_attempt_url=null |
| 最终结果 | completed／success，8/8 Job success |
| 实际检出的合并快照 | `533b8d67601a12f87dab208bd1be5899e30c040d` |
| 父提交 | `258fc50f5e94354892737614ab61eea564178e7c`、`d4ba0a9b29995e21536efb8fcd7a089c17f96703` |
| 合并快照与目标内容树 | 均为 `d0b660f628fb61a33cf8c8cc12cc3363a4100f3e` |

核对了全部八个 Job 的日志，checkout 均指向上述合并快照：Web、Go lint、Windows／Ubuntu／macOS Go tests，以及三平台 Build with dashboard。

三个 Go Job 都实际记录 40 个合法示例、165 个非法反例；Ubuntu 按工作流使用 Race Detector。四项 CrossUser 测试在普通步骤因非 root 跳过，在专门 sudo 步骤实际通过，两类记录没有混用。其余按平台设计跳过的已有测试不被描述为本机或三平台全部独立实测。

这组 CI 验证已提交的规范／Schema 与既有实现检查，不证明尚未实现的 M2 重放、读快照接线、diff 投影或独立复核器已经运行通过。

### 3. 作者本地记录与本轮证据分开

按本次任务提供的本地验证记录，分别保留：首次 gitx 超时、sqlite 的既有 M-1 失败、相关单独重跑成功、internal/cli 未在本地运行；CI 则是上述固定目标的首轮 8/8 成功。本轮独立执行的是契约包，不是作者的这些重跑。

没有取得作者本地失败的完整原始过程日志，不能据此独立认定超时或 sqlite 失败的根因；也不能把 PR 检查清单中的最终通过改写成全部本地测试首次通过。CI 绿色及重跑成功均不关闭 M-1、M-10 或其他维护项。本轮不要求修复这些实现，也没有重跑全套 M1。

## 七、PRODUCT DECISION、ACCEPTED RISK 与状态同步

**PRODUCT DECISION：没有新增必须由本轮负责人裁决的事项。**

PRD NFR-11、MVP 的重放说明及 architecture 已同步完整输入；它们通过补齐上下文保持可复现保证，没有将“缺项时使用当前值”改成合法行为。没有增加重放命令、API 或持久化表，没有改变 M1 init、事实序号或已批准的交付判定。

TD-27 正确拆分为：当前输入／输出／重放语义须在 WP2.1／WP2.2 前明确；未来是否提供设置历史独立待决。将来若改变 diff-since 使用哪一组设置，仍须另行决定并同步规范，不能用本轮 GO 预先批准。

**ACCEPTED RISK：本轮没有新增。** 输出不是事实备份、同一规则版本下重放和 AC 投影的依赖范围，是此次明确的契约前提，不是以接受新风险换取 GO。未批准的 Q1、Q3、Q4、Q6、Q7、Q9、E1～E5、恢复退出码／交互仍未获批；Q10 的既有截止调整不变。M0 N1、Windows 数据库竞争、doctor Git Trace 等维护事项保持原状态。

AGENTS、implementation-plan 与 §56 都保留“待定向复审／未授权 M2”的状态，没有作者自行宣布关闭。原 Review 044 保持其固定身份和历史结论；本报告承担对新目标关闭 B1 的独立判断。

## 八、交付结论与接手建议

| 问题 | 结论 |
|---|---|
| Review 044 B1 是否关闭？ | **可以关闭。** 完整输入、设置输出、原输入重放和 diff 的保证已经一致。 |
| PR #20 是否 GO／可合并？ | **GO，可以交由负责人合并。** 本轮不执行合并。 |
| 完整 F4／F5 是否具备冻结条件？ | **具备。** 复用 Review 044 已通过部分，本轮直接关联修订没有破坏它们。本文 N1 是非阻塞措辞建议。 |
| 除合并核验和负责人明确授权外，首批还缺什么？ | 没有新的实质性规范或产品待决门槛；获授权后仍须落实 F6／测试先行、两套独立转换器和既定反例。不能把本次规格 GO 当成这些实现已完成。 |
| 是否授权实现或批准其他草案？ | **否。** 冻结、合并、实现授权仍由负责人决定；本报告不启动 M2。 |

建议接手顺序：负责人核对固定目标及合并结果／master CI，登记本轮审查和规格冻结决定，再明确是否授权完整 WP2.1／WP2.2。可顺手收紧本轮 N1 的措辞；既有提交关联 N1 留在 WP2.3／Q7，不因此增加一次全面审查。

可交给接手方的提示：

> 先按 Review 045 原样归档与登记，保留受审 SHA、CI 和验证边界；不得据此自行开始 M2。负责人明确授权后，再以冻结的 F4／F5 组织完整 WP2.1／WP2.2 的测试先行工作，特别保留原设置重放、同命令设置快照，以及来源集合变化时 AC 差异投影的用例。未批准的后续产品草案保持原状态。

本轮仅新增本报告，使用仓库相对链接。未修改其他文件或历史审查，未提交、推送、合并，未批准产品取舍或新增接受风险。审查完成后暂停，不启动 M2。
