# Specmetry MVP 实施计划

> 状态：Draft v0.3 · 2026-10-14（含 codex-review-002 的处理；M1 开工准备见 [review-resolution.md §13](review-resolution.md#13-m1-开工准备产品决策与设计收尾2026-10-14)）
> 依据：[mvp.md](mvp.md)（范围）、[data-model.md](data-model.md)（模型与推导）、[`schemas/`](../schemas/)（契约）、[acceptance-criteria.md](acceptance-criteria.md)（验收）、[review-resolution.md](review-resolution.md)（审查处理）。
> 本计划只排顺序、依赖和退出条件，不估工期。

## 1. 实施原则

1. **契约先行**：`schemas/` 是 CLI 和 API 输入输出的规范来源。实现不能先于 schema 改变格式；要改格式，必须先改 schema 和文档。
2. **推导引擎测试先行**：`domain/derive` 先把 data-model §6 的决策表写成表驱动测试，再实现。每条规则都要有对应的反例（CX）。
3. **独立的检查器**：测试代码中实现一个不复用推导引擎的 `ready` 复核器（不变量 #2），在全部 E2E 中运行。
4. **每个里程碑都可以独立验收**：退出条件是该里程碑全部的 P0 验收项和 CX 项（见 acceptance-criteria 各表的"里程碑"列）。P1 项不阻塞。
5. **范围纪律**：实施中发现需要新功能时，先修改 mvp.md 并获得确认；不能"顺手实现"。
6. **设计变更流程**：修改文档 → 修改 schema（如果涉及，按 TD-20 判断是否升级版本）→ 修改或新增验收项 → 实现。

## 2. 开工前提条件

| 门槛 | 内容 | 阻塞哪个里程碑 | 状态 |
|---|---|---|---|
| G0 技术决策 | 确认 TD-02（数据目录）、TD-04（SQLite 驱动；需要先做一个 spike，验证 `VACUUM INTO` 和 WAL 检查点）、TD-10（CLI 框架）、TD-16（迁移策略）、TD-22（Windows ACL 授权对象） | M0 | **已满足**（2026-10-13；spike 通过；TD-22 选择当前用户 + SYSTEM） |
| G1 隐私契约 | 确认 PD-20（C1 允许的内容） | M0 中的存储部分 | **已满足**（2026-10-13；附加的采集策略与用户控制见 architecture §10.7） |
| G2 交付模型 | 确认 PD-16（沿用，仅限隔离级证据）、PD-17（例外验收）、PD-19（交付目标形态）、PD-21（范围粒度）；确认 PD-03、PD-04、PD-12 的建议默认；同时确认截止在 M1 的 PD-07、PD-13、PD-14、PD-24，以及技术决策 TD-01、TD-05、TD-14、TD-18、TD-20、TD-21 | M1（冻结 `proposal`、`review`、`scope` 的契约和 M1 读取模型；`decision` 契约在 G2b 冻结，这里只确认它的语义前提 PD-03、PD-16、PD-17） | **已满足**（2026-10-14）：上述 PD 与 TD 全部已确认（PD-04 采用方案 A；PD-18 一并确认）；补充设计见 [review-resolution.md §13](review-resolution.md#13-m1-开工准备产品决策与设计收尾2026-10-14) |
| G2b 证据等级 | 确认 PD-22（MVP 只提供观测级证据，交付判定标为观测级，人验收时确认风险）；codex 对 B1、B2 的复审通过 | M2（Check Runner 与人工决定的契约在此冻结） | **已满足**（PD-22 已确认；codex-review-003 结论为 GO） |
| G3 许可证 | PD-10 | 接受外部贡献之前（不阻塞内部编码） | 未满足 |

## 3. 设计细化截止点

以下项目已经在模型层面关闭，只剩实现细节需要在对应模块开工前确定。**这些细节都不会改变数据模型或交换契约。**

| 项 | 内容 | 截止 | 关联 |
|---|---|---|---|
| Windows ACL | 授权对象（当前用户；是否加入 SYSTEM）、实现方式、`doctor` 的检测方法 | M0 开工前 | 审查 #12、TD-22 |
| SQLite 驱动 spike | 验证 `VACUUM INTO`、`wal_checkpoint(TRUNCATE)`、`BEGIN EXCLUSIVE`、触发器 | M0 开工前 | TD-04 |
| 密钥规则集 | 内置规则清单、正反样本、配置扩展方式 | M1 开工前 | TD-21；**已确认**（2026-10-14）：architecture §10.8 |
| 人类读取模型的 schema | `proposal show`（含三方对比和 `item_id`）、`baseline show/history`、`source list` 的 `--json` 输出 | M1 开工前 | 第二轮自检 #5；**已补齐**（连同 M1 的其他读取模型和 Agent 结果输出，data-model §8） |
| `doctor --json` 的 schema | 版本、路径、权限、Hook 与 Skill、spool、兼容版本 | M3 开工前 | 第二轮自检 #5 |
| 隔离执行器的后端 | TD-24：物化方式、只读强制机制、写入探测、执行后比对、子模块和 LFS | M7 开工前（MVP 之后） | codex-002 B1 |
| Windows 上执行检查的 shell | 默认 shell 的探测顺序；`shell` 字段的取值映射。**已决定**（2026-10-25，WP2.5 D1 ~ D4，review-resolution §106）：所有平台默认 `sh`，不回退 `cmd`；Windows 只取 `gitx` 所用 Git 的 Git for Windows 的 `usr\bin` 入口。实际路径、安装布局与 PATH 的构造是 WP2.5 规范冻结之前的 T5 规格 | WP2.5 开工前（2026-10-18 调整） | TD-06、M2-Q1 |
| `inputs` 的对象 ID | 比较语义与错误边界：**已确认**（2026-10-18，data-model §4.11.2）；Git 读取机制（命令与调用方式）：方案已写出（[WP2.3 规范 §4](wp2.3-git-reader.md)）；**已冻结**（2026-10-19，审查 057 技术 GO，产品负责人批准） | Git 读取机制：WP2.3 开工前（2026-10-18 调整） | TD-23、M2-Q2 |
| 提交关联的生命周期 | **已决定**（2026-10-18，产品负责人）：已入库 SHA 的关联在首次入库时固定，不重算、不补写；首次满足条件的未入库提交按入库时的 `seq` 写入，不回填旧序号；没有新入库提交的扫描不分配 `seq`，可以更新游标（data-model §4.12）。扫描的范围、写入事务与待决的细节见 [WP2.3 规范](wp2.3-git-reader.md)，**已冻结**（2026-10-19） | WP2.3 开工前，与 M2-Q7 一起冻结 | 审查 044 N1、M2-Q7、TD-17 |
| 备份恢复命令（P1） | `backup restore` 的锁、原子替换、恢复后的墓碑处理 | WP2.8 中该 P1 工作开工前（2026-10-18 调整） | 审查 #12 |
| 数据库恢复命令 | `db recover` 的备份方式、墓碑顺序、一致性检查、拒绝路径的文件副作用；§8.1 表中第 1 ~ 4 项的确认；补齐需求与验收项；U1、U2 的明确结论 | WP2.10 开工前（2026-10-18 调整；仍不迟于 M4 自测） | PD-23、M2-Q3 |
| Claude Code Hook 核实 | 事件名、输入字段、PreToolUse 拒绝协议、PowerShell 工具名；固定经过测试的版本范围；更新 architecture §12.1 | M3 开工前 | 审查 #10、TD-12 |
| 其他 Agent 的约束 | Codex 等的沙箱可写路径、是否支持 Skill、受保护的 git 路径；更新兼容矩阵 | M4 开工前 | 审查 #11 |
| Dashboard 技术细节 | OpenAPI 组织方式、SSE、UI 方案、端口 | M5 开工前 | TD-07、08、09、19 |

## 4. 依赖关系

```mermaid
flowchart LR
    G0([G0 技术决策]) --> M0
    G1([G1 隐私契约]) --> M0
    M0[M0 骨架与存储基础] --> M1[M1 基准、来源、目标与范围]
    G2([G2 交付模型决策]) --> M1
    M1 --> M2[M2 事实、检查与交付推导]
    G2b([G2b 证据等级确认 + B1/B2 复审]) --> M2
    M0 --> M3a[M3 Hook 入口与 ingest]
    M2 --> M3b[M3 sync 联调与 setup]
    M3a --> M3b
    M2 --> M5[M5 Dashboard]
    M3b --> M4[M4 Agent Skill]
    M2 --> M4
    M4 --> M6[M6 加固与发布]
    M5 --> M6
    M3b --> M6
    M6 -.MVP 之后.-> M7[M7 隔离执行器]
```

**可以并行**的部分：

- M3 的 Hook 入口和 ingest（只依赖 M0），可以与 M1、M2 并行；
- M5 可以在 M2 的读取模型（`status.v1`）稳定后开始，与 M3、M4 并行。

## 5. 里程碑

### M0 工程骨架与存储基础

**目标**：后续所有模块依赖的基础设施，以及审查 #6（`seq`）和 #12（存储生命周期）中的基础部分。

| 工作包 | 内容 |
|---|---|
| WP0.1 | Go module、cobra CLI 骨架、`version` 与 `doctor`（框架）、lint、三平台 CI |
| WP0.2 | 数据目录与权限：Unix `0700` / `0600`，Windows 受保护的 DACL；`SPECMETRY_HOME`；`config.toml` |
| WP0.3 | SQLite 存储：WAL、`busy_timeout`、单一写连接；迁移框架（`.migrate.lock`、`BEGIN EXCLUSIVE`、`VACUUM INTO` 备份、`schema_version` / `min_binary_version`）；`seq` 计数器；只追加触发器辅助；墓碑检查的挂载点 |
| WP0.4 | 内嵌 `schemas/`，选定 JSON Schema 2020-12 校验库；CI 校验所有 schema 本身，`schemas/examples/valid/` 全部通过，`invalid-cases.json`（SCH-01 ~ 64）全部失败；`error.v1` 输出。设计阶段只用一个精简的自制校验器做过自检，正式校验库的结果以 M0 为准。 |
| WP0.5 | 测试基座：临时 HOME、夹具仓库构造器（用脚本创建 git 仓库、分支、worktree）、golden 工具、伪终端辅助、canary 搜索工具 |
| WP0.6 | 前端工程骨架（Vite + React + TS），通过 `go:embed` 内嵌，并提供跳过前端的构建标签 |

**退出条件（P0）**：MVP-PLT-01、MVP-PRV-05、MVP-PRV-06、MVP-STO-01 ~ 04、MVP-STO-06、MVP-STO-07（后两项由 codex-review-004 增补）。
**交付物**：可以运行的 `specmetry version`、`specmetry doctor`；空库的迁移 v1；CI 通过。
**状态（2026-10-14）**：**已验收**（产品负责人，2026-10-14）。代码审查经过五轮（codex-review-004 ~ 008），最终结论 GO，BLOCKER 为 0；GitHub Actions 第二轮 CI 8/8 通过，Windows、Linux（race detector，以 root 运行的跨用户测试）、macOS（含 ACL 检测）均在实际平台上运行；九项退出条件全部满足。记录见 [review-resolution.md §12](review-resolution.md#12-m0-验收github-actions-ci-与产品负责人决定)。**后续事项（不属于 M0 退出条件，尚未完成）**：Windows 真实双用户验证未执行；N1 部分修复，最晚在 M2 的 `purge` 验收前完成；`specmetry db recover`（PD-23）在 M2（WP2.10）实现，最晚在 M4 自测前完成。

### M1 基准、来源、交付目标与范围

**目标**：审查 #4、#5、#8 和中-1 的实现；#3、#6 中属于写入侧的部分。

**状态（2026-10-17）**：设计收尾完成，G2 已满足（[review-resolution.md §13](review-resolution.md#13-m1-开工准备产品决策与设计收尾2026-10-14)）。产品负责人已授权开发（codex-review-009 GO 之后）。**WP1.3 已验收**（codex-review-011 GO，PR #3，合并提交 `cee87b4`；MVP-PRV-07 的 CLI 写入集成随 WP1.5 验证，见 [review-resolution.md §16](review-resolution.md#16-wp13-最终复审codex-review-011与验收)）；**WP1.0 已验收**（codex-review-013 GO，PR #4，合并提交 `84e6e70`，见 [review-resolution.md §18](review-resolution.md#18-wp10-最终复审codex-review-013与验收)）；**WP1.2 已验收**（codex-review-014 GO，PR #5，合并提交 `4c5a038`；真实人类专属命令的 MVP-CH-01、CH-02 验证随 WP1.4、WP1.6、WP1.8 完成，见 [review-resolution.md §19](review-resolution.md#19-wp12-代码审查codex-review-014与验收)）；**WP1.1 已验收**（codex-review-017 GO，PR #6，合并提交 `568432d`；审查经过三轮，015、016 为 NO-GO，处理记录见 [review-resolution.md §20](review-resolution.md#20-wp11-代码审查codex-review-015的处理) ~ [§22](review-resolution.md#22-wp11-最终复审codex-review-017与验收)；`init` 的登记规则见 [architecture.md §6.1.2](architecture.md#612-init-的登记规则m1-实现)）；**WP1.4 已验收**（codex-review-019 GO，PR #7，合并提交 `35d060f`；审查 018 为 NO-GO，处理记录见 [review-resolution.md §23](review-resolution.md#23-wp14-代码审查codex-review-018的处理) ~ [§24](review-resolution.md#24-wp14-最终复审codex-review-019与验收)；Source Ref 相关的验收随 WP1.5 / M2；命令行为见 [architecture.md §6.1.3](architecture.md#613-来源的登记与确认m1-实现)；MVP-SRC-06 的含义按已确认的 PD-25）；**WP1.5 已验收**（codex-review-024 GO，PR #8，合并提交 `429e94e`；审查经过五轮，020 ~ 023 为 NO-GO，其中 022 按已确认的 PD-26（方案 A）处理：运行时探测 git 安全读取对象的能力，不具备能力时拒绝读取对象；处理记录见 [review-resolution.md §25](review-resolution.md#25-wp15-代码审查codex-review-020的处理) ~ [§30](review-resolution.md#30-wp15-最终定向复审codex-review-024与验收)；SHA-256 仓库按 TD-26 拒绝；命令行为见 [architecture.md §6.1.4](architecture.md#614-提案的提交m1-实现)；审核、`propose --apply` 与编号的正式分配在 WP1.6）；**WP1.6 已验收**（codex-review-026 GO，PR #9，合并提交 `c9c9cf8`；审查 025 为 NO-GO，处理记录见 [review-resolution.md §31](review-resolution.md#31-wp16-代码审查codex-review-025的处理) ~ [§32](review-resolution.md#32-wp16-最终定向复审codex-review-026与验收)；命令行为见 [architecture.md §6.1.5](architecture.md#615-提案的审核与人工编辑m1-实现)；附带在审核中的范围确认随 WP1.8；保留两项 NON-BLOCKER：确认预览的变更类别、历史中的后续处理理由）；**WP1.7 已验收**（codex-review-028 GO，PR #10，合并提交 `867bd60`；审查 027 为 NO-GO，处理记录见 [review-resolution.md §33](review-resolution.md#33-wp17-代码审查codex-review-027的处理) ~ [§34](review-resolution.md#34-wp17-最终定向复审codex-review-028与验收)；命令行为见 [architecture.md §6.1.6](architecture.md#616-问题的查看回答与驳回m1-实现)；问题阻塞的推导随 M2；审查中登记的两项遗留安全问题见 M6"发布前必须关闭的遗留安全问题"）；**WP1.8 已验收**（codex-review-035 GO，PR #14，合并提交 `19600ec`；审查 034 为 NO-GO，处理记录见 [review-resolution.md §43](review-resolution.md#43-wp18-交付目标与交付范围的实现) ~ [§45](review-resolution.md#45-wp18-最终定向复审codex-review-035与验收)；命令行为见 [architecture.md §6.1.7](architecture.md#617-交付目标与交付范围m1-实现)；MVP-TGT-09、`NO_SCOPE` 的推导和真实豁免的生命周期随 M2）。**WP1.0 ~ WP1.8 全部验收完成。** 证据核对第 1 版的 6 项缺口由 PR #16 补齐（codex-review-036 GO，合并提交 `01bd50f`）；M1 最终验收前的项目名称安全修复已通过审查并合并（PR #17，codex-review-037 GO，合并提交 `5aad555`）。退出条件的证据核对第 2 版见 [m1-exit-evidence.md](m1-exit-evidence.md)：59 个 P0 项全部 PASS。**M1 最终退出条件审查 GO**（codex-review-038：PR #15 GO；固定代码快照 `5aad555` 的 M1 退出条件 GO，59 PASS、0 INCOMPLETE；review-resolution §49）。**M1 已验收**（产品负责人，2026-10-17；验收对象 `7e77fd6`，即 PR #15 的合并提交；依据为审查 038、审查 039 与合并后 master CI 39025525139；review-resolution §50）。未解决的维护事项保留，见下文与 [m1-exit-evidence.md](m1-exit-evidence.md) 第 9 节。M2 尚未开始，开工需要另行明确授权。

| 工作包 | 内容 |
|---|---|
| WP1.0 | M1 夹具：`fx-docs-only`（含录制的 `proposal.json`）、`fx-in-dev` 的文件与分支脚本、`fx-incomplete`（acceptance-criteria §1.1；原属 WP2.9，提前到 M1，因为 INIT-03、INIT-04、BL-01 需要它们） |
| WP1.1 | 项目与 worktree 的解析和登记（git 公共目录）、marker、`init`（输出 `init.v1`，包括 `init_progress` 和 `baseline_version`）、阶段识别与候选来源（architecture §6.1.1）、默认交付目标 |
| WP1.2 | 通道识别：TTY、已知的 Agent 环境变量、交互式确认组件（architecture §5.1） |
| WP1.3 | 密钥规则与字符串扫描（拒绝写入；architecture §10.8）；`config.toml` 的 `[secrets]` 追加规则 |
| WP1.4 | 来源：`add` / `list` / `remove`（含 `--erase-excerpts`，M1 只有 `hash_only` 引用）/ `ack`，哈希，来源集合哈希，变化检测；`excerpt_storage` 默认 `hash_only`（PD-24）；输出 `source-add.v1`、`source-list.v1`。`full` 模式和擦除后的整理在 WP2.11。人类专属命令在写入之前调用 `channel.Guard`，并在本工作包中完成 MVP-CH-01、CH-02 对这些命令的验证 |
| WP1.5 | 提案：schema 校验、版本前置条件、引用片段校验、代码引用、编号分配（AC 按需求的作用域分配）；输出 `proposal-receipt.v1` |
| WP1.6 | 审核：逐条决定（接受 `update` 条目必须选择 `change_class`，V-R01）、冲突检测与三方对比、版本与变更记录、级联、父需求修订与 AC 后续处理、`reaffirm`、`propose --apply`；输出 `proposal-list.v1`、`proposal-show.v1`、`baseline-show.v1`、`baseline-history.v1`。人类专属命令在写入之前调用 `channel.Guard`，并在本工作包中完成 MVP-CH-01、CH-02 对这些命令的验证 |
| WP1.7 | 待澄清问题（只能由人回答或驳回）；输出 `question-list.v1` |
| WP1.8 | `target set`、`scope confirm`（独立提交，或附带在审核中）、声明校验、已有豁免的确认（V-T01）、读取时范围确认的校验（V-T02）；输出 `target-scope.v1`。人类专属命令在写入之前调用 `channel.Guard`，并在本工作包中完成 MVP-CH-01、CH-02 对这些命令的验证 |

**建表约束**（codex-review-004 N1）：见 [data-model.md §1.1](data-model.md#11-建表约束m1-起)；由 schema 不变量测试强制。

**输出契约的语义约束**（codex-review-009 NON-BLOCKER 2）：实现某个输出接口时，必须同时为该输出的 schema 补充字段之间的条件约束和对应反例；跨 ID、数据库状态等 schema 无法表达的关系，由输出一致性测试验证。这是相应工作包的验收任务，不提前实现。M1 的已知缺口已补齐：`proposal-receipt.v1`（WP1.5，MVP-BL-28；SCH-113 ~ 125）、`proposal-show.v1`（WP1.6，MVP-BL-14；SCH-126 ~ 129）、`question-list.v1`（WP1.7，MVP-BL-17；SCH-131 ~ 136），另有 `target-scope.v1`（WP1.8；SCH-140 ~ 147）。尚待补齐：`status.v1` 中拒收决定的快照（WP2.7，MVP-ACC-07）。

**退出条件（P0）**：acceptance-criteria 中标注 M1 的全部 P0 项（包括 CH-01、CH-02，以及 SRC-08、TGT-10、WT-01 的 M1 部分），以及 CX-14、CX-17（CLI 部分）。标注为"M1 / M2"的项，在 M1 中只验证写入侧（退出码、写入的数据、`init --json` 的输出），推导侧（差距、交付状态）在 M2 中验证。
**可选（P1）**：MVP-INIT-08、INIT-10、INV-05。

**M1 最终验收之前必须完成的独立安全修复**（产品负责人 2026-10-17 确定方向；单独的最小修复 PR，须经过代码审查；M1 退出证据的补充 PR 中不实现）：

- **项目名称禁止代码围栏**，与交付目标名称一致（architecture §10.2；参照 codex-review-034 B2）。检查范围：
  - `init --name`；
  - 由目录名生成的默认项目名称；
  - `project.validateName`；
  - `init.v1` 中 `project.name` 的 schema；
  - 已存在的项目中不合法名称的兼容处理。
- 不能静默修改已有项目的名称、marker 或数据库中的值。
- **已完成**：PR #17，合并提交 `5aad555`，codex-review-037 GO（review-resolution §47、§48）。旧项目按产品负责人批准的方案 1 处理：`init` 有说明地拒绝（退出码 2），不静默改名，不改写事实。已接受的局部限制：这类项目暂时不能通过 `init` 修改阶段、补写 marker 或登记关联 worktree。M2 `status.v1` 中名称的处理留到 M2 设计。

### M2 事实、检查执行与交付推导

**目标**：审查 #1（检查侧）、#2、#3（执行与扫描）、#6（Claim、决定、检查）、#7、#9 和 #12（`purge`）的实现。**这是 MVP 的核心里程碑。**

**状态（2026-10-18）**：开工准备报告已合并（PR #19，合并提交 `258fc50`；审查 043 GO，见 [m2-readiness.md](m2-readiness.md)）。产品负责人 2026-10-18 的决定（review-resolution §55）：首批为完整的 WP2.1、WP2.2；批准 M2-Q2 的比较语义、M2-Q5（PD-02）、M2-Q8、畸形输入的选项 (a)；按 M2-Q10 调整了 §3 中的截止点（见下表）。其余草案（M2-Q1、Q3、Q4、Q6、Q7、Q9，E1 ~ E5，恢复的退出码与交互）没有获批。首批规范 F4（data-model §6.13、§6.13.1，与 §4.10 ~ §4.13 中的 M2 事实结构）、合法事实域（data-model §6.14）与 F5（acceptance-criteria §3.1）在 PR #20 中写出：审查 044 为 NO-GO（B1：当前值设置输入与输出的重放契约，review-resolution §56），修订后经审查 045 GO（§57）。**F1 ~ F5 已冻结**（2026-10-18，产品负责人），是完整首批的依据。**已授权**：完整的 WP2.1、WP2.2，在 PR #20 合并、master CI 核验之后开始，每个工作包一个 PR；**其余 M2 工作包尚未授权**。实现 PR：WP2.1 为 PR #21，WP2.2 为 PR #22（叠加在 #21 之上）。审查 046 两者都为 NO-GO（B1：共享生成器把违反 V-P03 与 §5.5 的历史当作合法历史）；已按审查修复并重新取得合法域证据（review-resolution §58）。审查 047 两者仍为 NO-GO：原三个反例已修复，但合法级联被误拒、不可变提交树的矛盾与非法的问题引用被接纳（B1-R1 ~ R3）；N1、N2 已关闭；P1 的澄清已由负责人确认。已按审查修复并重新取得证据（§59）。审查 048 两者仍为 NO-GO：B1-R1、B1-R3 已关闭；B1-R2 只剩“没有声明 inputs 却记录 input_missing”，已修复两个合法性门禁与对应的手写样例（§60）。审查 049 两者 GO，B1 整体关闭。PR #21、#22 已于 2026-10-18 以 merge commit 合并（`1364af1`、`8cf54e0`），master CI 通过；**WP2.1、WP2.2 的纯函数层交付已由产品负责人正式验收**（review-resolution §61）。CLI、真实数据库读取、Git、输出 Schema 与 WP2.9 的 E2E 不在这次验收之内，由对应的工作包验证；**WP2.3 的实现获得条件授权（2026-10-19，见下文）；其余 M2 工作包尚未授权，M2 尚未整体验收**。审查 044 的 N1 登记在 §3 的"提交关联的生命周期"，随 WP2.3 与 M2-Q7 冻结。

**WP2.3 的准备（2026-10-18）**：

- 产品负责人决定了 M2-Q7 / TD-17、关联的生命周期与没有新入库提交的扫描（review-resolution §62）。
- WP2.3 规范（[wp2.3-git-reader.md](wp2.3-git-reader.md)）已写出，包括 TD-23 的读取机制与实验、`internal/gitx` 的新接口、扫描层、迁移设计、正反例、验收映射与测试计划，待独立审查。
- 规范中列出了需要负责人决定的冲突（D1、D2）与需要审查的细节（D3 ~ D8）。
- 审查 050 为 NO-GO（B1：`--ignore-missing` 吞掉必需的正向 tip；B2：旧 tip 缺失的判断；B3：`status` 会执行仓库配置的过滤器）。产品负责人 2026-10-18 决定了 D1、D2、D7；规范已按审查修订，新增 MVP-EV-18，待复审（review-resolution §63）。其余没有获批的草案保持原状。
- 审查 052 仍为 NO-GO（B1-R：commit-graph 掩盖排除对象的消失；B2-R：损坏的包索引被判为可靠缺失；B3-R：过滤器下的错误 `clean`，稀疏索引借 alternates 改变原对象的修改时间）。规范已按审查修订，待复审；由此引出的取舍 P1 ~ P3 待负责人决定（review-resolution §64）。
- 审查 055 仍为 NO-GO（B2-R2：`.pack` 失去 `.idx` 时被当作可靠缺失；B3-R2：稀疏门禁晚于 `status`；另需决定浅克隆加深后的生命周期）。产品负责人 2026-10-19 决定了 P1（`unknown`）、P2（含稀疏目录条目的索引一律不可判定，门禁前置）、P3（MVP 不支持浅克隆的提交扫描）与 `worktree list -z` 的能力要求。规范已按审查修订，待复审（review-resolution §66）。下一轮审查的基准为已核验的 master `46a4b26`。
- 审查 057 为技术 GO（B2-R2、B3-R2 与浅克隆的生命周期都关闭；N1：主索引不存在的正常分支写进实现测试）。产品负责人 2026-10-19 **冻结**了 WP2.3 规范与此前确认的产品范围，并**条件授权 WP2.3 的实现**：PR #24 合并、合并核验与对应的 master push CI 全部通过之后开工。实现约束见 WP2.3 规范 §11 与 review-resolution §67。
- WP2.3 的实现 PR #27：审查 058 为 NO-GO（B1 私有目录的祖先路径与身份、B2 截断位图 panic、B3 `.specmetry/` 的排除；N1 测试快照），已修复（review-resolution §68）；审查 059 为 NO-GO，关闭 B2、B3、N1，B1 未关闭（删除重试、替代锁文件、根的身份），B1 已修复后经审查 060 复审仍为 NO-GO：原三个持续替换反例已修复，B1 因精确窗口及 Log 漏读却成功未关闭，B2、B3、N1 保持关闭。当时未批准保证调整（历史记录 §69、§70）；之后负责人明确批准 B1-D1～D4 和本轮方案 B 实施（§71）。代码 `979f705` 已提交推送，实施证据见 `docs/wp2.3-b1-implementation.md`；经审查 061、062 的定向修复（§72、§73），审查 063 对 `c4dbfb1` 技术 GO，B1 在已批准的方案 B 下关闭（§74）。已合并为 `5985b31`（master CI 39287153098 的 attempt 2 正式成功，attempt 1 的 Windows 失败作为 M-15 的新观测保留）；**WP2.3 库层已于 2026-10-20 由产品负责人正式验收**（对象 `5985b31`，§75）。CLI 接入、输出集成与 E2E 由对应工作包完成。实现细化写在 WP2.3 规范 §14。
- **M-7**：须在 WP2.4、WP2.5、WP2.7 的新输出接口之前修复。它不是 WP2.3 的开工条件，建议在 WP2.3 规范冻结后单独修复（WP2.3 规范 §12）。**已关闭**（2026-10-20）：修复 PR #28 经审查 064 技术 GO，合并为 `7a36166`，master CI 正式成功（§76）。
- **M2-Q6、M2-Q9**：2026-10-20 由产品负责人批准（Q6：TD-13 的建议默认，Claim 中只有 `manual` 批次是人类专属；Q9：`claim-receipt.v1` 的独立回执方向；§76）。WP2.4 规范（`docs/wp2.4-claim.md`、`claim-receipt.v1`）的 D1 ~ D3 于 2026-10-20 由负责人决定（§77）；审查 065（对 `058b9a0`）为 NO-GO，唯一 BLOCKER B1（并发退役的退出码与版本优先冲突）已按版本优先修正，E4、E6 由负责人决定（§78）；审查 066（对 `29834d1`）GO，B1 关闭。产品负责人 2026-10-20 **冻结**了 WP2.4 规范，并**条件授权 WP2.4 的实现**：PR #29 合并、合并核验与对应的 master push CI 全部成功之后开工，另开 PR；不授权合并实现 PR、验收工作包或 WP2.5 等其他工作包（§79）。实现 PR #30：审查 067（对 `69d2f86`）为 NO-GO，实现 B1（提交前缀的候选未全部验证）、B2（文本模式未在提交前校验回执），已修复（§80）；审查 068（对 `e70c09b`）关闭 B1、B2，仍为 NO-GO：B3（文件引用的对象完整性，含共用的 `propose` 路径）**已修复，待独立复审**（§81）。合并 PR #31 后的 master `8b45ef5` 已整合进 PR #30，源码、测试、Schema、迁移与依赖不变（§82）。审查 070（对 `b8606b5`）GO，B3、N1 关闭，B1、B2 没有直接回归；**技术审查通过，待合并核验**。负责人 2026-10-22 条件授权合并，并在合并核验与 master push CI 全部成功之后验收 WP2.4 本工作包范围（§83）。已合并为 `bcbb21c`，合并核验与 master push CI 正式成功；**WP2.4 本工作包范围已于 2026-10-22 由产品负责人正式验收**（对象 `bcbb21c`，§84；`status` / `gaps` 中的展示、检查执行、M3 会话与 WP2.9 的 E2E 留在原工作包）。
- **M-16**（2026-10-20 登记）：Windows CI 上 `internal/cli` 超过默认的每包 600 秒超时，耗时余量不足（§77）；2026-10-21 在 PR #30 的 CI 中又观测到 555 s。负责人 2026-10-21 批准风险缓解：Windows CI 单独以 20 分钟的包级超时执行 `internal/cli`，其他包与 Ubuntu、macOS 不变（CI 调整 PR #31，分支 `ci/m16-windows-cli-timeout`）；代价是挂起最晚 20 分钟才被发现。审查 069（对 `0de32c5`）GO，负责人 2026-10-22 条件授权合并 PR #31；已合并为 `8b45ef5`，master push CI 正式成功（§82）。这是风险缓解，根因未验证，本项未解决（`docs/m1-exit-evidence.md` §9）。Ubuntu 上 `internal/cli`（带 `-race`，默认 600 s）在 2026-10-22 的 7 次 CI 中为 307 ~ 530 s，都没有超时；Windows 的缓解不覆盖 Ubuntu，不调整 Ubuntu 的超时或并行度（§85）。
- **WP2.5 的规范准备**（2026-10-22 授权，§84）：规范草案 [wp2.5-check-runner.md](wp2.5-check-runner.md)（PR #32，未冻结）。第 1 版 `bbef175` 经审查 071 为 NO-GO（B1 `Clean` 的保证被概括得过强、B2 批次标记限制逐条执行、B3 进程与输出管道的收尾没有闭环、B4 EV-04 改写的过度承诺）；已按审查修订为第 2 版（§85）；第 2 版 `be3f4f6` 经审查 072 复审，B1、B2、B4 在准备文档层面关闭，B3 残余三项（迟到输出与 Writer 生命周期、进程的归属与身份、组合失败策略），已修订为第 3 版（§86）；第 3 版 `78a96dd` 经审查 073 复审，B3-R2 在准备文档层面关闭，B3-R1（排空与复制确认的预算、未确认结束的出口）、B3-R3（持久化失败的状态叠加）未关闭，已修订为第 4 版（§87）；第 4 版 `a53f2d3` 经审查 074 **准备文档技术审查通过**，B3 在准备文档层面整体关闭，两项非阻塞意见已按措辞收口；负责人 2026-10-23 条件授权合并 PR #32，并在合并核验成功之后授权冻结之前的 test-only 机制原型（另开 PR，§88）。PR #32 已合并为 `ecb83ba`，合并核验与 master push CI 正式成功；冻结之前的机制原型在 PR #33（[wp2.5-prototype.md](wp2.5-prototype.md)，test-only）：审查 075 为 NO-GO（B1：有效性门禁的假通过），已定向修复；审查 076 仍为 NO-GO（B1 的两项残余），已再次定向修复；审查 077 仍为 NO-GO（共享观察错误与见证的有效性），已第三次定向修复；审查 078 关闭 B1-R1-S，仍为 NO-GO（外层自检的组合判定），已第四次定向修复；审查 079 关闭 B1-R2-W-R，仍为 NO-GO（B2：保险清理的归属），已定向修复；审查 080 关闭 B2，技术 GO；PR #33 已合并为 `5e9a42c`，合并核验与 master push CI 正式成功；原型的 N4、N6 在独立小 PR #34 中修复，审查 081、082 为 NO-GO（B1：停止读取可能在阶段 J 之前阻塞，及其停止结果的生命周期；N4-R），N6 关闭；审查 083 关闭 B1-Ra、B1-Rb，N4-R-S 与 N-MAC 已定向修复；审查 084 为 NO-GO（B-N4），已定向修复；审查 085 关闭 B-N4 与 N1，仍为 NO-GO（B-N4-R），已定向修复；审查 086 仍为 NO-GO（B-N4-R 的两项残余），已定向修复；审查 087 技术 GO（B-N4-R/T、B-N4-R/I 关闭），已以 merge commit 合并为 `cc4514a`，合并后核验完成：master push CI run 40122051541 的 attempt 1 为 7/8（Windows 的 `TestSourceRemoveInTerminal` 失败，登记为 M-17），按负责人的单独授权只重跑失败作业的 attempt 2 正式成功，只有 Windows 实际重跑，其余七项沿用 attempt 1 的结果（§104）；T1 ~ T6 只得到局部支持（§89 ~ §105）。**产品负责人 2026-10-25 决定了负责人选择的 14 项（D1 ~ D9、D11、D13、D18 ~ D20）与 C-4**（§106），TD-06 / M2-Q1 与 EV-04 因此已决定；规范第 5 版草案与受影响契约的同步在独立 PR #36 中；审查 089 为技术 NO-GO（B1 ~ B3：O3 之后的宿主拒绝、阶段 J 与停止的先后、D19 的时长），已定向修订（§107）；审查 090 技术 GO，B1 ~ B3 在准备文档层面关闭（§108）；PR #36 已合并为 `f3fbe83`，合并核验完成（§109）。负责人决定 D21（§109）；规范第 6 版草案（冻结之前的技术规格，§19）在独立 PR #37 中，审查 091 为技术 NO-GO（B1、B2），已定向修订（§110）；审查 092 关闭 B1，B2 的残余 B2-R1、B2-R2 仍为 NO-GO，已再次定向修订（§111）；审查 093 关闭 B2-R1，B2-R2 仍为 NO-GO，已撤回按 S3 前后丢弃可靠退出码的写法并定向修订（§112）；审查 094 技术 GO，B2 在准备文档层面整体收口，PR #37 已合并为 `2c5cf23`，合并核验完成（§113、§114）；负责人 2026-10-27 决定 Q-1 与 Q-2 (a)，规范第 7 版草案的同步与 P-1、P-2、P-4 的隔离原型、P-3 的整理在独立 PR #38 中（§114）；审查 095 为技术 NO-GO（B1），已定向修复，固定代码点 `85ec2ca`（§115）；审查 096 为技术 NO-GO（B1-R：Linux 排他区间退出时清除 subreaper 的错误被丢弃），已定向修复，固定代码点 `48ff0cd`（§116）；审查 097 技术 GO（B1-R 关闭，审查 095 B1 在受审原型范围内整体收口；N-096-2 非阻塞，待独立修复），PR #38 已合并为 `27ce69d`，合并核验完成；N-096-2 在独立小 PR #39 中修复，固定代码点 `18bc6a4`（§117、§118）；审查 098 技术 GO，N-096-2 在受审的受控原型范围内关闭，PR #39 已合并为 `614ac4b`，合并核验完成（§119、§120）；负责人 2026-10-29 授权冻结之前的文档整理，规范第 8 版草案（G5 的登记、P-3 的完整计划、§20 的分类与审查清单）在独立文档 PR #40 中（§120）；审查 099 为技术 NO-GO（B1、B2），第 8 版第二稿已定向修订，两项阻塞已修订，待独立复审；新增负责人待决的 Q-3（§121）；负责人 2026-10-29 决定 Q-3 选 (a)，已在 PR #40 中落实（§122）；审查 100 技术 GO，B1、B2 关闭，负责人条件授权合并 PR #40（§123）；PR #40 已合并为 `dbcd5d4`，合并核验完成；P-3 在独立原型 PR #41 中执行（固定代码点 `4e690d8`，诊断 run 40751152882 四个作业 success）；审查 101 为技术 NO-GO（B1 ~ B3），已定向修复（固定代码点 `eb2317f`，诊断 run 40817305419 为 failure：arm64 一项既有门禁失败，原因未知）；审查 102 认定 B1 ~ B3 关闭、修复技术 GO、暂不建议合并；最小观测与补证在固定代码点 `9d45036`（诊断 run 40847347022 四个作业 success）；审查 103 技术 GO，N-102-1 关闭，PR #41 已合并为 `03a6f6e`，合并核验完成；冻结候选稿与审查包（规范第 9 版 §21，推荐 macOS 生产入口 E2，负责人待决 Q-4）在独立文档 PR #42 中；审查 104 为技术 NO-GO（B1、B2），第 9 版第二稿已定向修订并补上 P-5 的计划草稿（只在 Q-4 (a) 获批之后适用）；审查 105 认定 B1、B2 关闭，新增 B-P5-1，第三稿已定向修订；审查 106 技术 GO（B-P5-1 关闭）；负责人 2026-10-30 决定 Q-4 选 (a)，条件授权合并 PR #42，P-5 的计划另开文档 PR 定稿；G5 未满足（§124 ~ §131）；其中 G3、G6、G7 的最小原型（P-1、P-2、P-4）已在隔离原型中执行，审查 097 认定只取得局部支持，G5（P-3）已执行；审查 101 为 NO-GO，已修复；审查 102 认定修复技术 GO、暂不建议合并；审查 103 认定补证技术 GO，PR #41 已合并；G5 的冻结审查包在规范第 9 版 §21，没有满足。这**不是**规范冻结：实现细化（D10、D12、D14 ~ D17、E1 ~ E9）与原型的候选规格（C-1 ~ C-20）只有采纳建议、待审查；剩余技术门槛（第 5 版 §17.5 的 G1 ~ G15）中冻结之前必需的规格尚未全部写定；生产实现没有授权，WP2.5 没有验收。除已授权的 P-1、P-2、P-4 隔离原型之外，不授权 WP2.5 的实现、新的机制实验、规范冻结或 WP2.5 验收。

| 工作包 | 内容 |
|---|---|
| WP2.1 | **推导引擎（测试先行）**：适用性、AC 决策表、进度、标记、需求聚合、目标交付状态、差距、指标、初始化进度、保守规则；性质测试 |
| WP2.2 | 独立的 `ready` 复核器（测试代码） |
| WP2.3 | Git Reader：解析 ref、porcelain 状态、对象 ID、按 ref 维护游标的增量扫描、提交关联（只保存 SHA） |
| WP2.4 | Claim：前置条件、代码上下文、`audit_scope`、覆盖率、会话关联（在 M3 之前，只按"上一批次"取比较起点） |
| WP2.5 | Check Runner（观测级）：开始时固定上下文、在调用者的 worktree 中执行、进程树、输出转发（`--json` 时转发到 stderr，stdout 只有 JSON）、合格判定、`context_changed`、`inputs`、标注 `evidence_grade = observed`；`--isolation required` 返回退出码 6，不写记录。**不实现**隔离执行器。 |
| WP2.6 | 决定：前置条件（含 `seen_decision_id`，在写事务中原子比较）、风险推导与确认（V-D01）、`normal` / `manual` / `exception`、交互式的 `accept` / `reject` / `waive`（收到退出码 5 后重新读取，不自动重试） |
| WP2.7 | `status --json`（`status.v1`）、`gaps`、人类可读输出、`--diff-since <seq>` |
| WP2.8 | `purge` 与墓碑生效；（P1）`backup create`、`backup restore` |
| WP2.10 | 数据库恢复命令 `specmetry db recover`（PD-23，见 §8.1）；**必须在 M4 自测开始前完成**。需求与验收项在 WP2.10 开工前补齐，并先确认 §8.1 表中第 1 ~ 4 项的调整，U1、U2 有明确结论（2026-10-18 调整截止，不移除交付义务） |
| WP2.9 | 夹具与 oracle：在 WP1.0 的基础上补齐 `fx-in-dev` 的失败测试和录制的 Claim，加上 `fx-scale` 和 oracle；完整的 CX 测试套件 |
| WP2.11 | 片段原文的保存与擦除：`project set excerpt-storage`（仅限人）、`full` 模式；`source remove --erase-excerpts` 之后的整理（`wal_checkpoint(TRUNCATE)` 与 `VACUUM`，读取 checkpoint 的结果，与 `purge` 共用整理步骤和 `.compaction-pending`）；已保存原文的累计总量上限（V-P11，MVP-BL-26）。**依赖 N1 的修复**，与 WP2.8 一起完成 |

**退出条件（P0）**：标注 M2 的全部 P0 项（包括 M1 / M2 项的推导侧；INIT-07、INIT-09、BL-18、WT-02 ~ 06、INV-01 ~ 04、CLM、EV（含 EV-12 ~ EV-18）、ACC（ACC-03 除外，含 ACC-10、ACC-11）、ST（ST-06 除外）、PRV-03、PLT-05、COL-12），以及 **CX-01 ~ CX-22、CX-24 的检查部分、CX-25 ~ CX-28 的 MVP 部分**（CX-08 的 UI 部分和 CX-23 除外）。
SRC-08、TGT-10 的 M2 部分，以及 PRV-10，也在本里程碑验收。
**可选（P1）**：INV-06、STO-05。
**PD-23**：`db recover`（WP2.10）在 M2 完成；它的验收项补齐后加入本里程碑的退出条件。
**环境失败（PD-26，2026-10-15，产品负责人）**：正确的进度推导依赖 Git 对象，而 Git 无法安全读取时（不具备 architecture §6.1.2 所述的能力，或对象无法读取），返回 environment（退出码 6），不把环境失败转换为正常的 `not_assessed`。Git Reader（WP2.3）、`status`、`gaps` 中的对象读取都经过 `internal/gitx` 的统一入口。M2 实现时据此细化 data-model §6 中"git 不可用"一行的结论，并补齐测试；M1 中不提前开发。
**里程碑评审**：M2 结束时，用 CX 结果和 E2E-1、E2E-2 的 CLI 版本做一次设计复核，确认推导规则在真实夹具上没有产生意外结论，然后再进入 M4 和 M5。

### M3 Hook 采集与 setup

**目标**：审查 #10 的实现，以及 #1 在 Hook 侧的部分。

| 工作包 | 内容 |
|---|---|
| WP3.0 | （开工前）核实 Hook 事件与字段，固定版本范围，更新 architecture §12.1 |
| WP3.1 | Hook 入口：输入上限、内部预算、不访问数据库的项目与 worktree 解析、读文件获取 HEAD、白名单、原子写 spool、drop 标记 |
| WP3.2 | PreToolUse 拦截（Bash、PowerShell），宁可误拦 |
| WP3.3 | ingest：`seq`、去重、会话投影、drop 汇总、死信、30 天保留 |
| WP3.4 | `setup` / `setup --uninstall`：用户级或项目级、展示改动、交互式确认 |
| WP3.5 | `sync` 的比较起点（加入 `session_start`）和 `changed_paths` |
| WP3.6 | `doctor`：Claude Code 版本、权限、Hook 与 Skill 状态 |

**退出条件（P0）**：标注 M3 的全部 P0 项（INIT-05、INIT-06、CLM-04 的会话部分、WT-03 的会话与事件部分、ACC-03、COL-01 ~ 11、COL-13、COL-15、PRV-04、PLT-04），以及 CX-24 的 `setup` 部分。

### M4 Agent Skill

**目标**：审查 #9（Skill 侧）和 #11（禁止事项与兼容）的实现。**开始 dogfooding。**

| 工作包 | 内容 |
|---|---|
| WP4.0 | （开工前）核实其他 Agent 的约束，更新兼容矩阵 |
| WP4.1 | Skill 包：`SKILL.md`；参考文档（粒度指南、交换格式、版本前置条件与退出码 5 的处理、禁止事项、worktree 与交付视图的区别） |
| WP4.2 | Skill 静态检查（命令树、禁止事项、schema 一致性） |
| WP4.3 | 评测框架，在三个项目夹具上运行 |
| WP4.4 | dogfooding：把 acceptance-criteria 导入为 Specmetry 自身的基准 |

**退出条件（P0）**：MVP-SK-01 ~ 07。
**可选（P1）**：MVP-SK-08。

### M5 Dashboard

| 工作包 | 内容 |
|---|---|
| WP5.1 | OpenAPI（请求体引用 `schemas/`），并审查"没有设置状态的接口"（MVP-ST-06） |
| WP5.2 | 服务端：令牌（不输出到 stdout）、Host 与 Origin、CSP、SSE、各端点、409 |
| WP5.3 | 页面：项目列表、概览（目标、快照、交付状态、阻塞差距）、基准树、审核队列（含 AC 后续处理和范围确认）、需求详情（含决定）、差距、健康 |
| WP5.4 | Playwright 测试套件（含页面过期时的处理、没有外部请求） |

**退出条件（P0）**：标注 M5 的全部 P0 项（BL-05 的 UI 部分、ACC-01 的 API 部分、ACC-07 的 API 部分、ST-06、COL-14、UI-01 ~ 14），以及 CX-08 的 UI 部分、CX-23。

### M6 加固与发布

| 工作包 | 内容 |
|---|---|
| WP6.1 | E2E-1 ~ 8，三平台运行，断言全部不变量 |
| WP6.2 | 隐私：在禁止出站网络的环境中运行（PRV-01）；C0 canary 综合测试（PRV-02） |
| WP6.3 | 在发布候选版本上回归性能基准（ST-08、COL-01、UI-04 已分别在 M2、M3、M5 首次通过），并验证路径健壮性（PLT-02） |
| WP6.4 | 打包分发（TD-15）；README：快速上手、数据位置、隐私契约摘要、能力边界 |
| WP6.5 | 3 个真实项目试用，并输出 dogfooding 报告 |

**退出条件**：[mvp.md §7](mvp.md#7-mvp-完成定义) 中的全部条件；发布 v0.1.0。

**发布前必须关闭的遗留安全问题**：

- `doctor` 的 git 版本查询（`git --version`）仍继承完整环境：设置 `GIT_TRACE` 等变量时会在外部写入跟踪文件，违反 `doctor` "非破坏性诊断" 的承诺（codex-review-016 NON-BLOCKER 1）。修复方向：改用 `internal/gitx` 的受控环境运行，并对低于 2.32 的 git 报告失败。已作为独立的安全修复任务登记，不属于 WP1.1，但**不是已接受的产品限制**，最晚在 v0.1.0 发布之前关闭。
- ~~**CLI 错误回显不可信参数**~~ **已完成**（2026-10-16；PR #11，合并提交 `a91016e`；codex-review-030 GO）。原登记内容保留如下，作为证据：codex-review-027 NON-BLOCKER N1，与 codex-review-025 B1 同类，是隐私缺陷，不是已接受的风险。
  - 原因：cobra 和 pflag 的错误被包装后保留原因。`printError` 的文本，以及 `error.v1` 的 `message`（`apperr.ToEnvelope` 使用包含原因的 `Error()`），都会原样引用参数。
  - 已复现的入口（以合成的 GitHub Token 作为参数）：
    - 根命令的未知子命令：`specmetry <x>`，带 `--json` 时同样回显；
    - `cobra.NoArgs` 命令多出的参数：`proposal list <x>`，带 `--json` 时 `<x>` 进入 `error.v1.error.message`；
    - 无法解析的 flag 值和未知 flag：`proposal list --pending=<x>`（`--json` 在前、在后都回显）、`source list --<x>`、`review -<x>`。
  - 机制相同、应一并处理的入口：其他使用 `cobra.NoArgs` 的命令（`init`、`doctor`、`version`、`source list`、`propose`、`review`、`baseline show`、`baseline history`），以及 `question` 之外的全部命令的 flag 错误。
  - 修复方向：在根命令上设置不引用参数的 flag 错误函数和 `Args` 校验，与 `question` 命令的 `noArgs`、`flagError` 相同；`error.v1` 的 `message` 不包含 cobra 的原因；补充覆盖全部命令、文本与 `--json` 两种模式的回归测试。
  - 关闭依据：
    - 全部命令由 `secureCommandLine` 统一处理（architecture §9.1"命令行错误"）；
    - cobra 的隐藏补全命令在 cobra 执行之前被拒绝（codex-review-029 B1）；
    - 回归测试 `TestCommandLineErrorsQuoteNothing`（63 种命令行）、`TestShellCompletionIsRefused` 与 `TestShellCompletionWordIsData`（真实二进制的子进程）在三平台 CI 中运行通过；
    - 处理记录见 review-resolution.md §35 ~ §37。
  - 保留的 NON-BLOCKER：codex-review-029 N1，成功的 JSON 输出测试只检查以 `{` 开头，待改为按具体 schema 校验。
- ~~**其他命令的文本输出没有转义终端控制字符**~~ **已完成**（2026-10-16；PR #12，合并提交 `26c787b`；codex-review-032 GO）。原登记内容保留如下，作为证据：与 codex-review-027 B1 同类。
  - Agent 提交的提案文本可以含有 ESC、CR、BEL 等控制字符。
  - 已按 027 B1 的方法在内存中确认原样输出的命令：`proposal list`、`proposal show`、`review` 的确认摘要、`baseline show`、`baseline history`。`propose --apply` 的确认摘要和 `source` 命令的文本输出可能同样受影响，修复时一并检查。
  - 修复方向：这些命令的文本输出和确认摘要统一使用 `internal/termtext`。
  - 关闭依据：
    - 命令的 stdout、stderr 都经过 `termtext.Writer`，人工确认摘要由 `channel.Guard` 转义，JSON 文档保持标准编码（architecture §9.1"终端显示"）；
    - `termtext.Writer` 会报告短写入，确认提示没有完整显示时 `Guard` 仍返回退出码 6（codex-review-031 B1）；
    - 回归测试 `TestTextOutputIsShownSafely`、`TestInitTextIsShownSafely`、`TestGuardShowsTheSummarySafely`、`TestWriterReportsShortWrites`、`TestConfirmationNotShownWholeWritesNothing` 在三平台 CI 中运行通过；
    - 处理记录见 review-resolution.md §38 ~ §40。
  - 保留的独立维护事项：codex-review-031 N1、N2（见下文"独立维护事项"）。

**独立维护事项**（不是安全修复，不属于任何工作包，各自单独处理；不是已接受的风险）：

- **首次登记特殊路径的非文档来源，确认可能无法输入**（codex-review-031 NON-BLOCKER N1）。
  - `source add --allow-any-type` 登记一个尚未登记的来源时，还没有来源 ID 可用，`confirmText` 回退为原始路径（`internal/cli/source.go`）。
  - 审查方的复现：路径含双向控制字符的非文档文件，输入屏幕上显示的转义文本时返回退出码 3、不写入；输入实际的原始路径时可以登记。输出中没有危险控制字符，没有错误放行。
  - 对 POSIX 允许的含换行的文件名，单行读取的确认无法输入这样的路径（未在真实 POSIX 文件系统上动态复验）。
  - 修复方向：对无法单行确认的路径给出明确提示，或改用一个可以输入的确认标识。
- ~~**人工交互取消后，`--json` 的 stdout 混入确认提示**~~ **已完成**（2026-10-17；PR #13，合并提交 `fbd6dff`；codex-review-033 GO）。带 `--json` 时确认显示在 stderr，stderr 不是终端时返回退出码 6；规则见 architecture §5.1 规则 2、§9.1；处理记录见 review-resolution.md §41、§42。原登记内容（codex-review-031 NON-BLOCKER N2）保留如下：
  - 在 human 通道执行 `question answer Q-001 -m answer --json`，取消确认后，stdout 依次是确认提示和 `error.v1`，整个 stdout 不是一个完整的 JSON 文档（`internal/cli/question.go` 调用 `channel.Guard` 时写到 stdout；`internal/cli/root.go` 把 `error.v1` 也写到 stdout）。`review`、`propose --apply`、`source ack` 等人类专属命令的确认使用同一机制，也会这样（按代码确认，未逐一复现）。
  - 基准提交 `a91016e` 与 `db81f5a` 都能复现，取消时没有写入，不是 PR #12 引入的回归。
  - 修复方向：`--json` 模式下把确认提示与机器可读输出分流（例如提示写到 stderr 或终端），并补充对整个 stdout 做 JSON 解析的测试。
- **自定义 Writer 与进程 stderr 的终端属性可能不一致**（codex-review-033 NON-BLOCKER N1；不是已接受的风险）。
  - `cli.Execute` 接受调用者传入的 stdout、stderr Writer，而 `confirm` 判断的是进程自己的 `os.Stderr` 是否为终端（`internal/cli/root.go`、`internal/cli/confirm.go`）。进程内的调用者把 stderr 换成另一个 Writer 时，带 `--json` 的确认提示写进该 Writer，人看不到，确认输入却仍会被读取。
  - 审查方的复现：在真实 ConPTY 子进程中调用 `cli.Execute(ctx, args, os.Stdout, &hidden)` 执行 JSON 模式的 `question answer`，提示进入 `hidden`，预先输入的正确确认仍使操作成功。
  - 正式二进制不受影响：唯一的生产调用 `cmd/specmetry/main.go` 传入的就是 `os.Stdout`、`os.Stderr`，检测对象与实际输出一致，没有命令行参数能替换 Writer。
  - 使用约束：`cli.Execute` 用于交互时，stdout、stderr 必须是进程自己的终端流；不能把"进程 stderr 是终端"推广为"任意传入的 Writer 都能被人看到"。
  - 修复方向：将来支持嵌入或包装输出时，把终端能力绑定到实际显示确认的 Writer。
- **跨项目需求外键的测试可能由另一条外键先拒绝**（codex-review-034 NON-BLOCKER N1；测试覆盖的准确性问题，不是生产缺陷）。
  - `internal/store/sqlite/scope_test.go` 的这个用例，把属于项目 A 的确认和需求都配上项目 B，可能先因确认的项目外键失败，不能单独证明需求的复合外键有效。
  - 审查方已独立构造前置条件，数据库正确拒绝，实际约束没有缺陷。
  - 修复方向：增加真实属于项目 B 的需求，再用项目 A 的合法确认引用它。
  - codex-review-035 N1 沿用此项。
- **Windows 首次创建数据库期间，数据目录权限检查偶发 `Access is denied`**（codex-review-035 NON-BLOCKER N2；可靠性问题，不是已接受的风险）。
  - 证据：PR #14 的 CI [run 38957016420](https://github.com/XKHoshizora/specmetry/actions/runs/38957016420) attempt 1 中，`Go tests (windows-latest)` 的 `internal/store/sqlite` `TestCheckWhileTheDatabaseIsCreated` 在 round 17、check 11 失败（`datadir.Check` 返回 `Access is denied`）。只重跑失败的 job（attempt 2），8/8 通过；本地 Windows 上连续 30 次运行（每次 20 轮）没有复现。该 PR 没有改动存储层、数据目录检查或这个测试。
  - 再次观测（2026-10-28）：PR #38 的常规 CI [run 40656041186](https://github.com/XKHoshizora/specmetry/actions/runs/40656041186) attempt 1 中，`Go tests (windows-latest)`（作业 122707197257）的同一测试在 round 11、check 9 失败（`openrace_test.go:155`，`Access is denied`）；该 PR 没有改动根模块。只确认是同一测试、相似的错误签名，不确认同根因；没有重跑，之后的通过不关闭本事项（review-resolution §117，登记见 [m1-exit-evidence.md §9](m1-exit-evidence.md)）。
  - 初步分析（未在 Windows 上复现验证）：
    - `internal/datadir/perm_windows.go` 的 `checkPermissions` 遍历数据目录，对每个条目调用 `GetNamedSecurityInfo`。
    - `vanished()`（`internal/datadir/datadir.go`）只把 `fs.ErrNotExist` 视为检查期间消失的条目。
    - SQLite 建库时的临时文件（例如回滚日志）被删除、句柄尚未关闭时处于删除挂起状态；此时打开它，Windows 返回 `ERROR_ACCESS_DENIED`，而不是"不存在"。
    - 这与提交 `8ef7dc3`（检查期间消失的条目）是同一类竞争。
  - 可能的影响：Windows 上并发首次建库（例如两个 `init` 同时执行）时，打开数据库前的权限检查偶发失败（退出码 6，不写入）。失败方向是安全的，但会错误地报告环境问题。
  - 约束：真实的权限问题同样表现为 `Access is denied`，不能把它一律当作"已消失"，也不能只靠重试掩盖。修复必须先确认条目确实已经消失或正在删除，才能跳过；否则照常报告错误。
  - 修复方向：在 Windows 上复现并确认根因（例如在测试钩子中保持一个删除挂起的文件），再按条目的实际状态处理，并补充确定性的回归测试。
  - 建议：M3 之前处理。M3 的 Hook 与采集会让并发进程成为常态。
- **旧项目名称测试的无写入摘要可以加强**（codex-review-037 NON-BLOCKER N2；测试强度，不是生产缺陷）。
  - `internal/cli/projectname_test.go` 的 `TestInitWithARegisteredNameThatHoldsACodeFence`，只比较名称、阶段、worktree 数量、`seq` 与 marker，没有直接覆盖完整的 settings、worktree 明细与 exclude。
  - 生产检查的顺序正确，审查方已另行逐表、逐字段独立验证。
  - 修复方向：把完整的 settings、worktree 明细与文件不变性断言加入仓库测试。

### M7 隔离执行器（MVP 之后，不属于 v0.1.0）

**目标**：提供 `snapshot_isolated` 证据，让沿用（PD-16）在实际中生效。开工前确定 TD-24。

| 工作包 | 内容 |
|---|---|
| WP7.1 | 私有快照物化与树一致性验证 |
| WP7.2 | 至少一个平台上的只读强制后端（由隔离后端强制，不依赖文件权限位），以及执行前写入探测 |
| WP7.3 | 执行后树比对；`--isolation required` 与 `auto` 的完整语义 |
| WP7.4 | 追加 CX-26 ~ CX-28 的隔离级部分，以及沿用相关的集成测试（ACC-09 的隔离级部分） |

在 M7 完成之前，任何文档、界面和输出都不得声称"快照已验证"。

## 6. 审查问题的关闭计划

| 审查问题 | 设计状态（codex-review-003 GO 后全部冻结） | 实现验证（里程碑；M0 部分已验证，其余未开始） | 验证依据 |
|---|---|---|---|
| #1 隐私 | 本轮 | M2（检查）、M3（Hook）、M6（综合） | EV-05、EV-11、COL-05、PRV-02 |
| #2 交付代码绑定 | 主体：v0.2；剩余部分见 B1 | M2 | CX-01、CX-02、CX-19 ~ 22 |
| codex-002 B1 证据与快照一致性 | 已冻结（codex-review-003 GO；分级证据，MVP 只有观测级） | M2（观测级、不沿用、风险确认）；M7（隔离级） | EV-13、ACC-09 ~ 10、CX-26 ~ 28 |
| #3 worktree | 本轮 | M1（登记）、M2（执行与扫描） | WT-*、CX-03、CX-04 |
| #4 交付范围 | 本轮 | M1（确认）、M2（推导） | TGT-*、CX-05、CX-06、CX-18 |
| #5 父需求依赖 | 本轮 | M1（后续处理）、M2（推导） | BL-18 ~ 20、CX-07 |
| #6 前置条件与 `seq` | 主体：v0.2；剩余部分见 B2 | M0（`seq`，已验证）、M1（写入侧）、M2（Claim、决定、检查） | STO-04、BL-03、BL-10、CX-08、CX-09、CX-13 ~ 15 |
| codex-002 B2 旧页面覆盖拒收 | 已冻结（codex-review-003 GO；`seen_decision_id`） | M2（CLI）、M5（API 与界面） | ACC-11、CX-08 (e)(f) |
| #7 检查语义 | 本轮 | M2 | ACC-04 ~ 06、CX-19 ~ 21 |
| #8 来源确认 | 本轮 | M1 | SRC-03、SRC-04、CX-17 |
| #9 盘点 | 本轮 | M2（模型）、M4（Skill） | INV-*、CX-10、SK-06 |
| #10 Hook | 模型：本轮；细节：M3 开工前 | M3 | COL-*、ACC-03 |
| #11 跨 Agent | 措辞：本轮；核实：M4 开工前 | M3、M4、M5 | SK-03、UI-14、CX-23 |
| #12 数据生命周期 | 模型：本轮；ACL：M0 开工前；恢复：`db recover` 在 WP2.10 开工前、`backup restore`（P1）在 WP2.8 的 P1 工作开工前（2026-10-18 调整，原为 M2 开工前） | M0（已验证：PRV-05、PRV-06、STO-01 ~ 04、STO-06、STO-07）、M2 | PRV-03、05、06、STO-*、CX-11 |
| 中-1 引用归属 | 本轮 | M1 | BL-04、BL-22 |
| 中-2 退出条件 | 本轮 | — | 本计划的结构 |
| 中-3 离线边界 | 本轮 | M6 | PRV-01 |

## 7. 风险跟踪

| 风险 | 触发信号 | 应对 |
|---|---|---|
| 推导规则在真实项目上过于保守，导致很难得到 `ready` | M2 评审或 dogfooding 中，大量 `needs_reverification` 被认为不合理 | 先确认是否是规则缺陷，再修改 data-model §6 与对应的 CX；**不能**为了"好看"放宽保守规则 |
| 检查会弄脏工作区，导致执行普遍不合格 | dogfooding 中 `context_changed` 频繁出现 | 完善文档与 `doctor` 的提示；不放宽合格判定 |
| Claude Code Hook 接口变化 | WP3.0 核实结果与 §12.1 不一致 | 只修改适配器映射；`spool-event.v1` 不变 |
| JSON Schema 校验库对 2020-12 支持不完整 | WP0.4 选型时发现 | 在 M0 中更换库，或调整 schema 写法（不改变语义） |

## 8. 已决策、待排期的功能

### 8.1 数据库恢复命令 `specmetry db recover`（PD-23）

**决策**（2026-10-13，产品负责人）：采用 [review-resolution.md §8](review-resolution.md#8-m0-第二次代码审查codex-review-005的处理) 中的方案 B，并做如下调整：

1. 普通打开和 `doctor` 保持严格：遇到无法只读确认版本的状态，继续拒绝（退出码 6）或报告版本未知。
2. 恢复是独立命令 `specmetry db recover`，不放进 `doctor`；`doctor` 必须保持只读。
3. 恢复必须由用户明确授权（仅限人，交互式确认），并实际检查 schema 版本、`min_binary_version` 和数据库一致性；不能仅凭用户声明"正在使用最新版本"就允许恢复。
4. 恢复前必须已有一致性备份；不得直接删除 `-wal` / `-shm`；不得绕过版本兼容限制。
5. M0 不开发；提交 `3e0b0bc` 的代码保持不变，先完成 Codex 对 B1 ~ B3 的复审。
6. **排期**（2026-10-13 确认）：在 M2 实现（WP2.10），必须在 M4 自测（dogfooding）开始前完成。

**适用的数据库状态**（[architecture.md §11.5](architecture.md#115-只读检查) 中报告"版本未知"、普通打开会被拒绝的状态）：WAL 模式数据库缺少 `-wal` 与 `-shm`；只有其中一个；存在非空的 `-journal`。

**功能依赖**：

| 依赖 | 现状 | 说明 |
|---|---|---|
| 通道识别与交互式确认 | M1 WP1.2 | 恢复仅限人；Agent 通道返回退出码 3 |
| 打开锁 | M0 已实现 | 恢复全程持有，与其他进程的打开和迁移互斥 |
| 版本保护规则 | M0 已实现（`guardVersion`） | 与普通打开共用同一规则，不另设规则 |
| 墓碑 | M0 已有挂载点；完整的 `purge` 在 M2 | 恢复产生的备份不得含有已删除项目的数据（见下表第 2 项） |
| 一致性备份 | M0 有迁移前备份（`VACUUM INTO`）；`backup create/restore` 是 P1，在 M2（MVP-STO-05） | 见下表第 1、6 项 |
| 数据库一致性检查 | 尚未设计（例如 `PRAGMA integrity_check`） | 必须在不改动原文件的前提下完成 |
| N1 的剩余部分 | 最晚在 M2 的 `purge` 验收前解决 | 恢复同样要合并 WAL，不能在 checkpoint 返回 busy 时误报完成 |

**排期评估**（排期已确认为 M2，评估保留作记录）：现有验收项中没有与 `db recover` 直接矛盾的条目。无论放在哪个里程碑，都需要**新增**需求与验收项，并调整下列已批准的内容。这些调整需要产品负责人在 M2 开工前确认，目前没有改动任何已批准的验收条件：

| # | 涉及的已批准内容 | 冲突或需要的调整 |
|---|---|---|
| 1 | architecture §11.3："只使用一致性备份机制（`VACUUM INTO`），不直接复制正在使用的数据库文件" | 在适用状态下，对原文件执行 `VACUUM INTO` 必须先用 SQLite 打开它，而这正是会修改文件的操作（§11.5）。要在不改动原文件的前提下完成检查与备份，可行做法是先在数据目录内复制数据库文件，在副本上检查。这需要修改 §11.3 的表述，例如：持有打开锁、并确认复制期间文件没有变化时，允许复制静止的数据库文件 |
| 2 | architecture §11.2、§11.4，CX-11，MVP-PRV-03："墓碑在备份之前执行，新生成的备份中不含已删除项目的数据" | 恢复前的备份如果直接取自原文件，墓碑还没有执行，可能含有已删除项目的数据，与 CX-11 的承诺冲突。设计必须先在副本上执行墓碑，再生成要保留的备份，并删除中间副本 |
| 3 | MVP-STO-02 的严格标准（拒绝时数据库、WAL、SHM 和目录项都不变） | 恢复命令拒绝时（版本不兼容、一致性检查失败、用户未授权）同样必须满足。如果检查在数据目录内的副本上进行，拒绝后副本必须删除，目录项最终不变，但检查期间会短暂出现副本；是否接受，需要在新验收项中写明 |
| 4 | mvp.md §2（范围的唯一依据）、prd §6 CLI 概览、AGENTS.md 中的人类专属操作 | 新命令需要写入范围、新增需求（FR）和 CLI 概览，并列为人类专属操作。权限矩阵（prd §3.3）已按本决定加入"仅限人"，标为待排期 |
| 5 | M1 的退出条件（标注 M1 的全部 P0 项） | 放在 M1 会增加 M1 的 P0 验收项，扩大 M1 的退出条件，并且必须排在 WP1.2 之后。另外 G2 尚未满足，M1 本身还没有开工 |
| 6 | MVP-STO-05 / FR-STO-04（`backup create/restore` 是 P1，在 M2） | 如果恢复是 P0，它依赖的备份能力就要提前到 P0，与"备份命令是 P1"的优先级不一致。可以复用 M0 的内部备份函数，而不提前实现 `backup` 命令 |
| 7 | M6 的退出条件（mvp.md §7） | 放在首次公开发布前不冲突，但需要写入 mvp.md 的范围和 §7 的完成清单。风险：从 M4 开始的 dogfooding 期间没有恢复手段 |

**结论**：排期为 M2，不晚于 M4 自测。它与 M2 的 `purge`、`backup create/restore` 共用墓碑、备份和人工确认。WP2.10 开工前仍需确认上表第 1 ~ 4 项的调整（2026-10-18 产品负责人把截止由"M2 开工前"调整为 WP2.10 开工前；不移除交付义务，不批准降低安全保证，U1、U2 必须在那之前有明确结论）；第 5 项（M1 退出条件）不再涉及，第 7 项的风险因排在 M4 之前而消除。

**PR #27 B1 后续授权（2026-10-19）**：负责人明确批准 B1-D1～D4，方案 B 的局部边界、扫描空覆盖及实施范围见 review-resolution §71；授权本轮实施、测试、提交和推送现有 PR，未授权合并或后续工作包。审查 060 的 NO-GO 是历史结论；实现结果待独立复核，不由实施方宣布 GO 或验收。

**Review 061 后续（2026-10-19）**：`8e089ab` 的定向独立复审为 NO-GO，B1 未关闭；owner 缺失预检与调用方崩溃后的 Git 生存期两项阻塞按 §72 修复，空覆盖与 Windows 子机制保持，B2／B3／已关闭 N1 不重开。负责人授权修改、测试、提交和推送 PR #27，未授权合并、后续工作包或实施方作验收结论。交付证据接续 [实施记录](wp2.3-b1-implementation.md)，历史报告不改写。

**Review 062／063 后续（2026-10-20）**：审查 062 对 `d121bfb` 为 NO-GO，B1-R1 关闭，唯一残余 B1-R2-S 按 §73 修复（`a2f7bd6`）；审查 063 对 `c4dbfb1` 技术 GO，B1-R2-S 关闭，B1 在已批准的方案 B 下关闭（§74）。Unix 的局部窗口与 Clean 不能排除 false-clean 的边界保留，Log 完整性没有豁免。负责人授权纯文档收尾、推送，以及在新最终 HEAD 的 CI 正式成功后以 merge commit 合并 #27 并核验；未授权修改代码、测试、Schema、迁移或冻结规则，未授权后续工作包，WP2.3 与 M2 的验收由负责人决定。合并与合并后核验已完成，WP2.3 库层已于 2026-10-20 验收（§75）。
