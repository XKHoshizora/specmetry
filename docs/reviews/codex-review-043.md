# Codex Review 043：PR #19 第 4 版定向独立复审

- 日期：2026-10-17（Asia/Tokyo）。
- 仓库：`XKHoshizora/specmetry`；[PR #19](https://github.com/XKHoshizora/specmetry/pull/19)。
- 基准 master：`f904ab6fdd3cbabd2025318e9e7b6ca0c707b7e9`。
- Review 042 目标：`be660dfe041d09b413957265fb20bde7367a9257`。
- 本轮固定目标：`e1d9000788521c110a2b806a56c13953dbe484d3`。
- CI：[run 39039844812](https://github.com/XKHoshizora/specmetry/actions/runs/39039844812)，Attempt 1，8/8 SUCCESS。
- 范围：完整阅读 [Review 042](codex-review-042.md)，只复核共享 seq 的残余 BLOCKER、验收映射 NON-BLOCKER 及本轮直接关联增量。不重新验收 M1，不重开已通过的恢复、Shell、历史完整性或复核器谓词审查。
- 方法：静态文档、规范、迁移与调用链核查，以及提交和 CI 元数据／日志核查。未运行动态实验或 M2 测试程序。

## 结论

**第 4 版准备报告：GO。Review 042 的共享 seq 阻塞及验收映射遗漏已经解决，没有新的 BLOCKER。** PR #19 可以作为待负责人决定的准备报告交由负责人合并。

报告现在区分序号分配与记录身份，明确保留共享 seq 的合法历史，也没有把一个事务等同于一次分配。两处概括性措辞可以在后续 F4／F5 规格中收紧，列为 NON-BLOCKER；不需要因此再开启一轮准备报告审查。

**完整首批与受限首批目前仍不具备开工前提。** 准备报告 GO 只确认其依据、待决事项和门槛已经足够准确；不表示 Q1～Q10、局部例外或新退出结果获批，不表示 F4／F5 已冻结，不授权 M2 实现。

## 一、固定身份、增量与归档

开始核查时，本地 HEAD 为指定目标，工作区干净，`docs/reviews/codex-review-043.md` 不存在。开始及结束核查时，PR HEAD 都是 `e1d9000788521c110a2b806a56c13953dbe484d3`，base 为 `f904ab6fdd3cbabd2025318e9e7b6ca0c707b7e9`，PR 保持 Open；没有混入其他提交。

本轮提交为：

| 提交 | 内容 |
|---|---|
| `bcb5a2fe447193460709419d19f9a7e2ac0a7f90` | 归档 Review 042 原文 |
| `e1d9000788521c110a2b806a56c13953dbe484d3` | 第 4 版准备报告、验收映射与活跃记录 |

`be660df → e1d9000` 实际只有四个文档文件：

- `docs/m2-readiness.md`；
- `docs/implementation-plan.md`；
- `docs/review-resolution.md`；
- 新增的 `docs/reviews/codex-review-042.md`。

没有生产代码、测试、Schema、迁移或正式验收标准变化。AGENTS、README、architecture、data-model 与 acceptance-criteria 在本轮也没有修改。`git diff --check` 通过。

Review 042 在 `bcb5a2f` 归档后没有被 `e1d9000` 改写，受审身份仍指向 `be660df`。活跃记录 [§53](../review-resolution.md#L2637)保留第 3 版 NO-GO、审查边界和证据身份；处理状态仍是“已修订，待复审”，没有自行宣布审查关闭。§52 只是追加带来源的后续说明，历史审查原文没有重写。

## 二、CI 身份与证明范围

| 项目 | 独立核对结果 |
|---|---|
| Workflow / event | CI / `pull_request` |
| HEAD | `e1d9000788521c110a2b806a56c13953dbe484d3` |
| Attempt | 1；`previous_attempt_url = null` |
| 最终状态 | completed / success，8 个 Job 全部 success |
| 实际检出的 PR 合并快照 | `4d55e43ea2c59c49d5f54d66aba21e9462634008` |
| 合并快照父提交 | `f904ab6fdd3cbabd2025318e9e7b6ca0c707b7e9`、`e1d9000788521c110a2b806a56c13953dbe484d3` |
| 合并快照及目标的树 | 均为 `98d111eac79abf8726bad99bafcf0fd939e77ae8` |

核对了 Windows、Ubuntu、macOS 三个 Go Job 的 checkout 日志，均明确检出上述合并快照。三个 Go tests、Web、Go lint 和三个 dashboard build Job 均成功。

Ubuntu Go Job 按工作流启用 Race Detector；四项 CrossUser 测试在专门的 sudo 步骤实际运行并 PASS。需要 root 的普通测试 SKIP 与之后的补跑应分别记录，不能用其中一部分代替全部证据。

**这组 CI 只验证已提交仓库。** 它没有运行未提交的 M2 实验，也没有证明恢复候选方案、U1-a、shell 适配器、生成器或复核器转换器已经实现或通过三平台验证。既有 Windows 首次建库间歇性问题等维护事项不因此关闭。

## 三、BLOCKER

**无。Review 042 的唯一共享 seq BLOCKER 可以关闭，限准备报告层面。**

### 1. 分配唯一与事实共享已经分开

[第 813 行](../m2-readiness.md#L813)不再要求每条事实的 seq 两两不同；[第 843 行](../m2-readiness.md#L843)也明确禁止仅因 seq 相同就判为畸形。畸形例子改为违反具体写入约束的情形，而不是笼统排除重复值。

这里的“同一次逻辑写入”应按已有调用链中的**一次序号分配及其明确复用范围**理解，不等于一条 SQL、一行记录、整个 CLI 命令或整个事务。报告已用共享表和一个事务多次分配的反例明确这一点，不需要新增“逻辑写入 ID”或全局排序字段。

序号唯一和递增讨论的是已提交事实历史。预演或失败事务会回滚计数器，不构成两份重复的已提交事实，也不要求失败操作消耗 seq；[NextSeq](../../internal/store/sqlite/store.go#L421)及 [review 的预演／正式事务](../../internal/cli/review.go#L112)保持这一行为。本轮没有修改它们。

### 2. 共享表与真实写入相符

对[共享表](../m2-readiness.md#L822)逐项核对：

| 路径 | 现有依据与判断 |
|---|---|
| review | [CLI](../../internal/cli/review.go#L120)一次分配 plan.Seq；[Version](../../internal/baseline/apply.go#L148)、[Changes](../../internal/baseline/apply.go#L272)、本次新写入的引用、审核字段和[附带 Scope](../../internal/review/scope.go#L82)复用该值。共享关系正确。 |
| propose --apply | [人工编辑应用](../../internal/cli/apply.go#L125)把 Proposal Receipt 的 Seq 放入 Baseline Plan，提案和本次应用复用同一分配值。 |
| propose | [Store](../../internal/proposal/store.go#L325)分配提案序号；条目归属提案，问题 created_seq 和问题的来源引用使用同一个序号。不是要求条目表新增 seq 列。 |
| target set | [SetTarget](../../internal/scope/target.go#L106)一次分配，旧目标 archived_seq 与新目标 created_seq 使用该值；这本来就是合法的跨行共享。 |
| 同一事务多次分配 | [init](../../internal/project/init.go#L449)分别登记主／关联 worktree 和默认 Target；[worktree 插入](../../internal/project/init.go#L594)自己分配 registered_seq。提案 [EnsureWorktree → NextSeq](../../internal/proposal/store.go#L321)也明确区分登记与提案分配。报告没有把事务视为只能有一个 seq。 |

相关 UNIQUE 列与迁移相符：proposals.seq、baseline_versions.seq、scope_confirmations.seq、source_acks.seq，以及 worktrees.registered_seq、delivery_targets.created_seq、sources.registered_seq／removed_seq，各有相应列唯一约束。baseline_changes、source_refs、code_refs 的 seq 没有这样的逐列唯一限制，可以出现多行同值。

这些是按表／列已有约束限定的共享关系，不能据此允许任意独立分配复用同值。也不能把它反向理解为“所有事务在每张表只改一行”，具体措辞建议见 NON-BLOCKER N1。

### 3. 生成器、转换器和顺序要求已经同步

[第 850～862 行](../m2-readiness.md#L850)及 [WP2.2 测试规划](../m2-readiness.md#L924)明确要求：

- 生成共享 seq 的合法历史，让它们参与 MVP／通用合法域的双向对照，不再当作畸形排除。
- 保留全部关联记录，不以 seq 去重、合并或丢弃；不提前选择当前决定或计算适用性／合格性。
- 两个实现各自处理 as_of_seq；到达 n 时，保留该序号的全部相关事实，而不是只保留一行。
- 同 seq 内沿用 change_index、item_index 和 data-model §5.5 的应用规则；不以 ULID、输入排列或新造的全局顺序判断业务先后。
- 合格执行先于正常验收决定、拒收解除需要后续新证据等具体时序仍严格按既有契约比较，没有统一放宽为小于等于。
- 乱序测试现在明确包含共享 seq 记录之间的排列；合法但不能 ready 的历史和足够的有效 ready 正例继续保留。

事实身份必须沿用各表的实际主键或标识，包括父子关系与复合主键。第 852 行的 ID 应理解为记录标识的统称，不是“所有表必须新增独立 id 列”；此处有可收紧的措辞，见 N2。

### 4. 没有冒充 M2 新事实分配方式已冻结

[F4／F4-a、F5／F5-a](../m2-readiness.md#L737)、受限子集、最小澄清和测试规划已纳入共享 seq。M2 的 Claim Batch／Claim、Check Run、Decision、Commit／Link 如何分配或共享，仍按数据模型和对应工作包设计明确，没有从 M1 共享表直接推定所有新表的行为。

“按对应工作包设计写明”不能解释为允许先实现依赖这些事实的推导器，再到后续工作包才确定其输入：首批实际需要的分配方式、记录身份、时序和引用结构，仍须在 F4／F4-a 冻结时明确。报告已保留这一冻结门槛，当前不是已完成状态。

不需要本 PR 现在实现生成器、转换器、M2 迁移或新增测试。准备报告对后续设计与验证要求的说明已经足够。

## 四、NON-BLOCKER

### N1：共享范围表的简称可按具体列和实际变更条件收紧

**位置**：[第 824、830 行](../m2-readiness.md#L824)。

“一次写入在这些表中至多写一行”最好改成“对同一个分配值，每个声明 UNIQUE 的序号列至多有一行”。这是列约束，不是事务修改行数约束：target set 同时修改旧行 archived_seq 和新行 created_seq；init 可以在同一事务以不同 seq 插入多个 worktree。

共享表也应按实际发生的写入理解：全拒绝审核不创建空 Baseline Version；revision_seq 仅在新建／相应实质修订等既有规则要求时刷新，[editorial 更新](../../internal/baseline/apply.go#L562)不会因此自动改 revision_seq。建议用“本次实际写入或更新的序号字段”概括，而不是要求每次审核重写列出的所有字段。

判为 NON-BLOCKER 的原因是：表内列出的共享关系正确，报告自身已经给出 target set 跨行共享及一个事务多次分配的例子，没有再次把合法共享历史排除。这里是概括性文字的范围需要收紧，不是本轮引入了新的迁移或业务限制。可在 F4／F5 规格整理时直接落实，无须为此重复准备报告审查。

### N2：将“各自的 ID”明确为“各表既有主键或标识”

**位置**：[第 852 行](../m2-readiness.md#L852)。

不按 seq 标识行的方向正确，但不必要求所有表有独立 id 列。例如 [scope_items](../../internal/store/sqlite/migrations/0006_scope.sql#L44)已有 `(confirmation_id, requirement_id)` 复合主键，并通过 confirmation 关联序号；proposal_items 的创建归属也通过 proposal 确定。

建议把“事实行由各自的 ID 标识”改为“按各表既有主键、复合键或约定标识保留记录”。本 PR 没有提出增加 id 列或修改这些表，因此不构成需要拒绝准备报告的 Schema 变更；后续转换器不能把这一泛称实现成新的列要求。

## 五、Review 042 的验收映射问题

**已解决。** [§6.2](../m2-readiness.md#L695)现在列出 STO-08、09a、09b、10～13、13b，并把 STO-14 单列为“只在 E4 获批时加入”。它们与正文草案对应，也明确须批准并写入规范后才计入 M2。

正式 acceptance-criteria 没有修改，恢复草案没有提前计入退出集合。Review 042 的该项 NON-BLOCKER 可以关闭。

## 六、PRODUCT DECISION 与 ACCEPTED RISK

**PRODUCT DECISION：本轮没有批准任何方案，也没有新增必须重决的产品问题。** 仍待负责人决定／冻结的内容包括：

- Q1 的默认解释器、EV-04 取舍、解释器不可用和 cmd 多行行为；Q2 的输入身份比较；Q5 的指标；Q8 的标记及警告；这些按各自对首批的影响处理。
- Q3 的 U1／U2、E1～E3／E5、E4 普通 Open 扩展和收尾结果；Q4、Q6、Q7、Q9 的已有待决事项。
- 畸形输入选项 (a)／(b) 的范围、F4／F5 或 F4-a／F5-a 的具体规格。
- Q10 的截止点处理及完整／受限首批的明确授权。

U1／U2 的原“M2 开工前”截止点继续有效；只有负责人明确批准 Q10，才可以按批准范围顺延。技术验证不能由批准替代。共享 seq 的既有关系不需要负责人重新决定；M2 新事实结构仍须按门槛写明和审查。

**ACCEPTED RISK：没有新增。** 仅保留既有方案 1 的局部旧名称兼容限制。未批准的恢复例外、历史完整性风险、临时副本与清理失败、进程树边界都不是本轮接受的风险。M-1、M-2、M-4～M-10、M-13 及 M0 N1 继续按既有记录保留，不因报告或 CI GO 自动关闭。

## 七、证据与未验证边界

本轮实际完成：

1. 阅读完整 Review 042，检查本轮四个文档的实际增量及归档身份。
2. 核对已有迁移的序号列 UNIQUE、允许多行的 seq 及复合主键；核对 review、Baseline、附带 Scope、人工编辑、提案、Target、worktree 登记的必要调用链。
3. 核对 F4／F5、受限方案、转换器和测试规划的共享 seq 要求，以及 STO 映射。
4. 独立核对 CI HEAD、Attempt、全部 Job 结果、三平台 Go checkout 日志、合并父提交和树；检查目标未变。
5. 本地只读差异检查通过，历史 Review 042 自归档后未改。

没有运行新的 Go 测试、生成器、恢复／锁实验、shell 实验或跨平台动态复验。对本轮文档描述的共享关系，以现有规范、迁移和源码为证；这不是宣称 M2 生成器或复核器已经通过测试。此前作者实验与独立审查证据的归属不变。

本报告使用仓库相对链接；`#L` 对应固定目标 `e1d9000`。旧报告行号仍属于各自受审提交，不改写历史原文来更新定位。

## 八、开工裁定与下一步

| 问题 | 结论 |
|---|---|
| Review 042 共享 seq 阻塞 | **已解决，准备报告层面关闭** |
| Review 042 验收映射遗漏 | **已解决** |
| 新的 BLOCKER | **无** |
| 第 4 版准备报告 | **GO；可以由负责人合并 PR #19** |
| 完整首批是否可以开工 | **尚不具备**；F1～F5 冻结／审查和 F6 仍未完成 |
| 受限首批是否可以开工 | **尚不具备**；F4-a／F5-a 须先冻结并审查，再完成 F6；现有截止点未获顺延 |

建议产品负责人下一步决定 Q1～Q10、局部例外、首批完整／受限范围与截止点。Claude Code 随后只把获批内容同步至规范、Schema、验收项和计划，并写出首批需要的 F4／F5 规格，纳入本报告两项措辞澄清。规格冻结与开工授权应明确记录，不能从本报告 GO 推定。

可交给 Claude Code 的后续提示词（仅在负责人决定相应范围后执行）：

> 请依据产品负责人明确批准的 M2 决定，同步对应规范、交换契约、验收项和实施计划，并写出首批 F4／F5 或受限 F4-a／F5-a 规格。序号分配与共享按已有调用链及各表具体约束，记录身份沿用既有主键／复合键；保留共享 seq 的全部合法事实，按既有 change_index、item_index 与时序规则处理，不推定所有事务只有一次分配。明确 M2 新事实的分配、引用和转换器输入域；未批准事项继续保留。不要据准备报告 GO 自行实现 M2，不改历史审查，不合并或验收；整理证据、未决门槛和下一步后暂停。

**最终结论：准备报告 GO，M2 尚未授权实现。** 不需要再重复本轮已经关闭的准备报告问题，也不重审 M1。后续产品决定和规格冻结是独立阶段。

本轮仅新增 `docs/reviews/codex-review-043.md`，没有修改其他仓库文件，不提交、推送或合并，不批准方案或新增接受风险。
