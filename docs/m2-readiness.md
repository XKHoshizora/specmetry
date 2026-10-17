# M2 开工准备报告

> 状态：**草案，开工准备**（2026-10-17）。本文只做资料核对、设计细化与任务规划，**不授权 M2 业务实现**。
> 基线：master `f904ab6`（PR #18 的合并提交，树 `705ce55`）；M1 验收对象 `7e77fd6`。审查 038、039 的结论保持有效，本文不重新打开 M1 验收。
> 依据：[mvp.md](mvp.md)（范围）、[implementation-plan.md](implementation-plan.md)（里程碑、§3 设计细化截止点、§8.1 `db recover`）、[architecture.md](architecture.md)、[data-model.md](data-model.md)、[prd.md](prd.md)、[acceptance-criteria.md](acceptance-criteria.md)、[`schemas/`](../schemas/)、[m1-exit-evidence.md](m1-exit-evidence.md) 第 9 节。

本文标为"推荐"或"草案"的内容都**没有经过批准，不是冻结契约**。产品负责人批准之后，才按第 6 节写入规范文档、schema 与验收项。

文中每项内容按来源分为三类：

| 分类 | 含义 |
|---|---|
| **既有批准决定** | 产品负责人已经确认的决定，本文只引用，不重新讨论 |
| **实施细节** | 不改变数据模型、交换契约和用户可见语义，按 implementation-plan §3 在对应模块开工前写明即可 |
| **新增产品取舍** | 会改变用户行为、范围、交换契约或已批准的验收，需要产品负责人决定 |

## 0. 摘要

- **基线**：远端 master 没有前进，仍是 `f904ab6`；本地与远端一致，工作区没有已跟踪文件的改动（第 1 节）。
- **首批实现**：推荐 WP2.1（推导引擎）与 WP2.2（独立 `ready` 复核器）。两者是纯函数和测试代码，不需要迁移、CLI 或 schema 变化，也不依赖本文的任何待决事项开工；验收前需要 M2-Q5、M2-Q8 有结论（第 8 节）。
- **需要负责人决定的事项**：共 10 项，见下表；详细分析在第 5 节。

| 编号 | 事项 | 推荐 | 分类 | 最晚决定于 |
|---|---|---|---|---|
| M2-Q1 | TD-06：检查使用的 shell | 省略 `shell` 时所有平台都用 `sh`，不回退到 `cmd`；找不到解释器时不执行、不记录，退出码 6；Windows 上自行构造命令行 | 新增产品取舍 + 实施细节 | WP2.5 开工前 |
| M2-Q2 | TD-23：`check.inputs` 的对象 ID | 记录每个路径在 `head_start` 上的 mode、类型与对象 ID；路径必须精确存在于提交树中，否则 `input_missing`；对象读取失败按 PD-26 返回退出码 6 | 实施细节（附两项行为确认） | WP2.3 的 `inputs` 部分开工前 |
| M2-Q3 | PD-23：`db recover` 的 §8.1 第 1 ~ 4 项 | 在数据目录内的私有副本上完成恢复与检查，由副本生成去除墓碑数据的备份，确认后才由 SQLite 打开原数据库 | 新增产品取舍（PD-23 本身已批准） | WP2.10 开工前 |
| M2-Q4 | `status.v1` 中旧项目名称的兼容 | `project.name` 输出 `null`，不输出含代码围栏的名称，`status` 照常推导 | 新增产品取舍 | WP2.7 开工前 |
| M2-Q5 | PD-02：主进度指标 | 采用建议默认（data-model §6.11 已按它书写） | 待确认的既有 PD | WP2.1 验收前 |
| M2-Q6 | TD-13：人类专属操作的防护强度 | 采用建议默认，并把 M2 的新人类专属命令纳入 | 待确认的既有 TD | WP2.6 开工前 |
| M2-Q7 | TD-17：提交关联规则 | trailer 加正文编号，MVP 不提供关闭开关 | 待确认的既有 TD | WP2.3 的提交关联开工前 |
| M2-Q8 | 条目级标记的表达（`source_removed`、需求的 `inferred`） | `status.v1` 的需求加 `flags`；增加不阻塞的 `SOURCE_REMOVED` 警告 | 新增产品取舍（契约补充） | WP2.1 验收前 |
| M2-Q9 | `claim --json` 的输出契约 | 新增 `claim-receipt.v1` | 新增产品取舍（契约补充） | WP2.4 开工前 |
| M2-Q10 | implementation-plan §3 中"M2 开工前"截止点的处理 | 本轮一并确认 M2-Q1 ~ Q3；否则按工作包顺延 | 流程 | 授权 M2 实现时 |

## 1. 基线核对

| 项 | 结果 |
|---|---|
| 远端 master | `f904ab6fdd3cbabd2025318e9e7b6ca0c707b7e9`（`git fetch` 之后），与已知值相同，没有新增差异 |
| 本地 | `master` 与 `origin/master` 一致；本报告在分支 `docs/m2-readiness` 上撰写 |
| 工作区 | 没有已跟踪文件的改动。本轮期间出现了两个未跟踪文件 `ACP.md`、`docs/agent-collaboration-protocol.md`，不是本轮产生的，本轮不修改、不提交它们 |
| master CI | [run 39028673308](https://github.com/XKHoshizora/specmetry/actions/runs/39028673308)，提交 `f904ab6`，首轮 8/8 成功 |

阅读范围：AGENTS.md、mvp.md、implementation-plan.md、architecture.md、data-model.md、prd.md、acceptance-criteria.md，M2 相关的 schema（`status.v1`、`gaps.v1`、`check-run.v1`、`claims.v1`、`decision.v1`、`common.v1`），m1-exit-evidence.md 的第 5、8、9、11 节，review-resolution §8、§9（PD-23 的原始决定），以及存储层（打开顺序、墓碑、备份、只读检查）和 `gitx` 的现有实现。

核对中发现的、需要决定的契约缺口（详见第 5 节）：

1. `status.v1` 的 `project.name` 与 `target.name` 没有禁止代码围栏，与 architecture §10.2、`init.v1`、`target-scope.v1` 不一致；旧项目名称的处理尚未决定（M2-Q4）。
2. MVP-SRC-04 要求引用"以 `source_removed` 的形式出现在状态与差距中"，MVP-BL-05 要求 `status --json` 带 `inferred` 标记，但 `status.v1` 只有 AC 有 `flags`，也没有 `source_removed` 标记或差距类型（M2-Q8）。
3. `claim` 提交成功时的 `--json` 输出没有 schema。AGENTS.md 要求面向 Agent 的输出都通过 schema 校验，MVP-INV-02 的存在性标记也需要一个让 Agent 看到的位置（M2-Q9）。

## 2. M2 目标、范围与不做的内容

**目标**（implementation-plan §5 M2）：实现审查 #1（检查侧）、#2、#3（执行与扫描）、#6（Claim、决定、检查）、#7、#9 和 #12（`purge`）。这是 MVP 的核心里程碑：从 M1 的基准、来源、目标与范围出发，产生三层事实（Claim、检查执行、人工决定），并推导出交付状态与差距。

**范围**（既有批准决定）：WP2.1 ~ WP2.11（第 3 节）。退出条件是标注 M2 的全部 P0 项，包括：

- "M1 / M2"项的推导侧：INIT-11、TGT-01、BL-01、SRC-04、SRC-07、TGT-10、WT-01、BL-05、BL-25；
- INIT-07、INIT-09、BL-18、BL-26、TGT-09、WT-02 ~ 05、INV-01 ~ 04、CLM-01 ~ 05、EV-01 ~ 13、ACC（ACC-03 除外）、ST（ST-06 除外）、PRV-03、PRV-09（CLI 部分）、PRV-10、PLT-05、COL-12、SRC-08（M2 部分）；
- CX-01 ~ CX-22、CX-24 的检查部分、CX-25 ~ CX-28 的 MVP 部分（CX-08 的 UI 部分和 CX-23 除外）。其中 CX-14 已在 M1 通过，CX-17 在 M2 验证推导侧；
- `db recover` 的验收项（第 5.3 节的草案）在批准后加入（PD-23，既有批准决定）。

可选（P1）：INV-06、STO-05（`backup create`、`backup restore`）。

**里程碑评审**（既有批准决定）：M2 结束时，用 CX 结果与 E2E-1、E2E-2 的 CLI 版本做一次设计复核，再进入 M4、M5。

**明确不做**：

| 不做 | 归属 |
|---|---|
| 隔离执行器、`snapshot_isolated` 证据、`--isolation auto`、沿用的实际发生 | M7（TD-24、FR-EV-09）。M2 中 `--isolation required` 一律退出码 6 |
| Hook、spool、ingest、会话投影、`setup`、`session_start` 比较起点、PreToolUse 拦截 | M3。M2 中 `sync` 的比较起点只有 `previous_batch` 与 `none` |
| `doctor --json` 的 schema | M3 开工前 |
| Agent Skill | M4 |
| HTTP API、Dashboard、ACC-01 / 07 / 10 / 11 的 API 部分、CX-08 的 UI 部分、CX-23 | M5 |
| E2E-1 ~ 8 的三平台全量运行、PRV-01、PRV-02 的综合 canary、打包分发 | M6 |
| 项目位置迁移（TD-25）、SHA-256 仓库（TD-26） | M4 之前 / v0.1.0 之后 |
| 导入 CI 结果、趋势、报告导出等 P2 项 | MVP 之后 |
| 项目改名命令 | 方案 1 的第 6 条（既有批准决定）；M2-Q4 的备选 C 需要另行决定 |
| 维护事项中不在 M2 路径上的修复 | 第 7 节 |

## 3. 工作包、依赖与实施顺序

### 3.1 工作包

| 工作包 | 交付物 | 依赖 | 需求 | 验收项 / CX | 待决事项 |
|---|---|---|---|---|---|
| **WP2.1** 推导引擎（测试先行） | `internal/domain/derive`：适用性、AC 决策表、进度、标记、需求聚合、目标交付状态、差距、指标、初始化进度、保守规则；性质测试 | 无（纯函数；输入类型在本包内定义） | FR-ST-01 ~ 05、07 ~ 09，FR-CLM-02，FR-ACC-09，FR-TGT-03、05，FR-INV-07 | ST-01 ~ 05、ST-12（单元部分）、ST-13、ST-14、ACC-09（单元）；CX-01、03、05 ~ 07、09、10、12、18 ~ 22、25 ~ 27 的推导层用例；BL-01、BL-18、BL-25、TGT-01、TGT-09、SRC-04、SRC-07、INIT-09、INIT-11 的推导规则 | M2-Q5、M2-Q8（验收前） |
| **WP2.2** 独立 `ready` 复核器 | 测试代码中的复核器，不复用推导引擎；自身的反例测试 | 无 | 不变量 #2（acceptance-criteria §3） | 不变量 #2、#8、#9 的断言；之后在全部 CX 集成测试与 E2E 中运行 | — |
| **WP2.3** Git Reader | ref 解析、工作区状态（`tree_state`）、`inputs` 的对象 ID、按 ref 的增量扫描与提交关联；全部对象读取经过 `internal/gitx` 的统一入口（PD-26） | 迁移：`commits`、`commit_links`、`git_scan_cursors` | FR-WT-04、FR-EV-06、FR-TGT-03、FR-WT-05 | WT-04、EV-11、CX-24 的 `tree_state` 部分；PD-26 的 M2 原则（环境失败返回退出码 6） | M2-Q2、M2-Q7 |
| **WP2.4** Claim | `claim --file`：前置条件、代码上下文、`audit_scope`、覆盖率、引用存在性标记；`manual` 批次仅限人 | WP2.3、M-7；迁移：`claim_batches`、`claims` | FR-CLM-01 ~ 05、FR-INV-01、02、07、FR-WT-02 | CLM-01 ~ 05、INV-01、02、04、INV-06（P1）、WT-03 的 Claim 部分、CX-10、CX-13 | M2-Q9 |
| **WP2.5** Check Runner（观测级） | `check run`：开始时固定上下文、在调用者的 worktree 中执行、进程树、输出转发、合格判定、`inputs`、`evidence_grade = observed`；`--isolation required` 退出码 6 | WP2.3、M-7；迁移：`check_runs` | FR-EV-01 ~ 05、08、FR-WT-03、FR-INV-04 | EV-01 ~ 10、12、13，WT-02、WT-03 的执行部分，INV-03，CX-02、04、15、16、24（检查部分）、26、27、28(a) | M2-Q1、M2-Q2 |
| **WP2.6** 决定 | `accept`、`reject`、`waive`：前置条件（含 `seen_decision_id`，写锁下比较）、风险推导与确认（V-D01）、`normal` / `manual` / `exception` | WP2.1、WP2.5；迁移：`decisions` | FR-ACC-01 ~ 08 | ACC-01、02、04 ~ 08、10、11（CLI 部分）、TGT-10 的 M2 部分，CX-01、08（CLI）、19 ~ 21、25 ~ 27 的验收步骤 | M2-Q6 |
| **WP2.7** `status`、`gaps` | `status --json`（`status.v1`）、`gaps --json`（`gaps.v1`）、人类可读输出、`--diff-since <seq>`；输出 schema 的条件约束（含拒收决定的快照，MVP-ACC-07） | WP2.1、WP2.3、M-7 | FR-ST-06、FR-WT-05、FR-PRV-09、FR-SRC-02、FR-INIT-05 | ST-07 ~ 11、WT-01、WT-05、INIT-07、09、11，BL-01、05、18、25，TGT-01、09，SRC-04、07，PRV-09（CLI），ACC-05 的显示，CX-05、06、18、22 | M2-Q4、M2-Q5、M2-Q8 |
| **WP2.8** `purge` 与墓碑；（P1）备份命令 | `purge`、墓碑生效、整理；P1 的 `backup create`、`backup restore` | N1 修复 | FR-PRV-03、FR-STO-03、FR-STO-04（P1） | PRV-03、CX-11、STO-05（P1） | — |
| **WP2.10** `db recover` | 第 5.3 节的流程 | N1 修复；M2-Q3 批准 | FR-STO-06（草案） | STO-08 ~ 13（草案） | M2-Q3 |
| **WP2.11** 片段原文的保存与擦除 | `project set excerpt-storage`（仅限人）、`full` 模式、擦除后的整理、累计上限（V-P11） | N1 修复；与 WP2.8 共用整理步骤 | FR-PRV-08、FR-SRC-07、FR-BL-02 | PRV-10、SRC-08（M2 部分）、BL-26 | — |
| **WP2.9** 夹具与 oracle | `fx-in-dev` 的失败测试与录制的 Claim、`fx-scale`、`expected-*.json`；完整的 CX 套件 | 随各工作包补齐 | — | ST-07、ST-08、COL-12、E2E-1 / 2 的 CLI 版本 | — |

跨工作包的验收：EV-07（新增事实表的只追加触发器，随各自的迁移）、PLT-05（data-model 与迁移一致，M2 结束时审查）、PRV-09 的 golden 静态检查（随 WP2.7）。

### 3.2 建议的实施顺序

| 批次 | 内容 | 理由 |
|---|---|---|
| 1 | **WP2.1、WP2.2** | 纯函数与测试代码，决定其余工作包的输入形态；不需要迁移、CLI 或 schema。AGENTS.md 要求推导引擎测试先行 |
| 2 | M-7、**WP2.3** | Git Reader 是 Claim、检查与状态的共同前提；M-7 在批次 3 新增输出接口之前完成 |
| 3 | **WP2.4**、**WP2.5** | 产生前两层事实；分别需要 M2-Q9、M2-Q1 / Q2 |
| 4 | **WP2.6**、**WP2.7** | 第三层事实与读取模型；WP2.7 把推导接到真实数据上 |
| 5 | N1 修复 → **WP2.8**、**WP2.11**、**WP2.10** | 三者共用墓碑、整理与人工确认；N1 必须在它们兑现保证之前修复。WP2.10 必须在 M4 自测开始前完成，排在 M2 内满足这一要求 |
| 6 | **WP2.9** 收尾、CX 全集、E2E-1 / 2 的 CLI 版本、M2 设计复核与退出证据 | 夹具从批次 3 起随工作包逐步补齐 |

批次 5 与批次 2 ~ 4 没有代码依赖，可以视需要提前；推导与事实相关的工作包按 1 → 4 的顺序进行。与 M1 相同，每个工作包（或一次修复，例如 M-7、N1）各自一个 PR，经独立审查后由产品负责人合并。

## 4. 开工条件逐项状态

状态：**已满足**；**待设计**（需要补充设计，本文已给出草案）；**待负责人确认**；**到对应模块之前解决**（不阻塞 M2 开工，阻塞对应的工作包）。

| # | 条件 | 状态 | 说明 |
|---|---|---|---|
| 1 | M1 已验收 | **已满足** | 产品负责人 2026-10-17 验收，对象 `7e77fd6`（review-resolution §50） |
| 2 | G2 交付模型（PD-16、17、19、21 等） | **已满足** | implementation-plan §2 |
| 3 | G2b 证据等级（PD-22，审查 003 GO） | **已满足** | Check Runner 与人工决定的契约已冻结 |
| 4 | master 基线与 CI | **已满足** | `f904ab6`，CI 首轮 8/8 |
| 5 | 本报告经独立审查 | **待负责人确认** | 建议交给 Codex 审查（第 10 节） |
| 6 | 产品负责人授权 M2 业务实现 | **待负责人确认** | 本轮不是授权 |
| 7 | TD-06（§3 截止点"M2 开工前"） | **待负责人确认** | 草案见 M2-Q1 |
| 8 | TD-23（§3 截止点"M2 开工前"） | **待负责人确认** | 草案见 M2-Q2 |
| 9 | `db recover` 的 §8.1 第 1 ~ 4 项、需求与验收项（§3 截止点"M2 开工前"） | **待负责人确认** | 草案见 M2-Q3；批准后补入规范文档 |
| 10 | `backup restore`（P1）的锁、原子替换、恢复后的墓碑（§3 截止点"M2 开工前"） | **到对应模块之前解决** | P1，不阻塞 M2 退出；在 WP2.8 的 P1 部分开工前设计，复用 M2-Q3 的机制 |
| 11 | PD-02、TD-13、TD-17（标注"M2"的待决事项） | **待负责人确认** | M2-Q5 ~ Q7 |
| 12 | `status.v1` 的旧项目名称 | **待负责人确认** | M2-Q4 |
| 13 | 条目级标记、`claim` 的输出契约 | **待设计** / **待负责人确认** | M2-Q8、M2-Q9 |
| 14 | M-7（成功 JSON 输出只检查 `{` 开头） | **到对应模块之前解决** | 在 WP2.4、2.5、2.7 新增输出接口之前 |
| 15 | M0 N1（checkpoint busy 时误删整理标记） | **到对应模块之前解决** | 在 WP2.8、2.10、2.11 兑现保证之前 |
| 16 | 夹具与 oracle | **到对应模块之前解决** | WP2.9，随各工作包补齐 |
| 17 | G3 许可证（PD-10） | 不适用于 M2 | 只在接受外部贡献之前阻塞 |

## 5. 待决事项与推荐方案

每项都给出：当前约束、推荐方案与备选、影响、建议的验收标准。实测、推断与未验证的区分见第 9 节；正文中的"（实测）"指第 9 节记录的实验结果。

### 5.1 M2-Q1：TD-06 Windows 上执行检查的 shell（草案）

**当前约束**（既有批准决定与已有契约）：

- `checkSpec.shell` 的取值为 `sh`、`bash`、`cmd`、`pwsh`、`powershell`，可以省略（common.v1）。
- `check_config_hash` 包含 `shell`（省略时为 null），`shell` 是强制 `material` 的字段（data-model §4.4、§7.2）。
- 检查继承用户的环境，注入 `SPECMETRY_CHECK=1`；超时终止整个进程树（Unix 进程组，Windows Job Object）；输出只转发、不保存（architecture §6.4，FR-EV-01、02）。
- MVP-EV-04："检查命令不存在 → `error` / `not_found`"。
- TD-06 现在的建议默认是"优先 `sh`（Git for Windows），找不到时用 `cmd /C`；AC 可以指定 `shell`"，尚未确认。

**实测要点**（Windows 11，Git for Windows 2.56）：

- 在普通的 PowerShell 会话中，`sh` 不在 PATH 上；`bash` 解析到 `%LOCALAPPDATA%\Microsoft\WindowsApps\bash.exe`，即 WSL 的启动器。Git for Windows 的 `sh.exe` 位于 `<Git>\usr\bin\` 和 `<Git>\bin\`，`git --exec-path` 给出 `<Git>/ucrt64/libexec/git-core`。
- 用 Go 默认的参数转义启动 Git for Windows 的 `sh`：不含空格的参数会被它的 MSYS 运行时展开通配符（`a*.txt` 变成两个文件名）和花括号（`{a,b}`），引号也会被改写（`x'y` 变成 `xy`）。自行构造命令行、每个参数都按 MSVCRT 规则加引号，11 个测试字符串全部原样到达。
- `cmd`：Go 默认的转义会改坏引号（`echo "a b"` 输出 `\"a b\"`）；用 `cmd.exe /d /s /c "<命令>"` 原样构造则正常。**多行命令只执行第一行，不报错**。命令不存在时退出码为 1，无法与普通失败区分。
- `pwsh` 7.6 与 Windows PowerShell：`-NoLogo -NoProfile -NonInteractive -Command <命令>` 正常，退出码可以传出；`-EncodedCommand` 在输出被重定向时会把 CLIXML 记录混进转发的输出。`pwsh` 中原生命令的非零退出码不会原样传出（`cmd /c exit 7` 得到 1）。
- `sh -c` 中命令不存在时退出码为 127。

**推荐方案**：

1. **默认（新增产品取舍）**：省略 `shell` 时，**所有平台都用 `sh`**，不回退到 `cmd`。
   - 理由：同一个检查配置（同一个 `check_config_hash`）在不同机器上应当是同一个程序。`sh` 与 `cmd` 的引号、变量、多行和内建命令都不同，静默回退会让同一条已确认的检查在有、没有 Git Bash 的机器上执行不同的程序。
   - Windows 上 Specmetry 本来就需要 git，Git for Windows 自带 `sh`。
2. **字段映射**（实施细节）：

   | `shell` | Linux / macOS | Windows |
   |---|---|---|
   | 省略、`sh` | `/bin/sh -c <命令>` | Git for Windows 的 `sh.exe -c <命令>` |
   | `bash` | PATH 上的 `bash -c <命令>` | Git for Windows 的 `bash.exe -c <命令>` |
   | `cmd` | 不可用 | `<系统目录>\cmd.exe /d /s /c "<命令>"` |
   | `pwsh` | PATH 上的 `pwsh -NoLogo -NoProfile -NonInteractive -Command <命令>` | 同左（`pwsh.exe`） |
   | `powershell` | 不可用 | `<系统目录>\WindowsPowerShell\v1.0\powershell.exe`，参数同上 |

   - `/d` 关闭 cmd 的 AutoRun，`-NoProfile` 不加载 PowerShell 配置文件：这些不属于经人确认的检查配置。
   - `sh`、`bash` 以非交互方式运行，不读启动文件；继承的 `BASH_ENV`、`ENV` 照常生效，属于"继承用户环境"的边界。
3. **解释器的查找**（实施细节）：
   - 只使用绝对路径，不在当前目录或 worktree 中查找，忽略 PATH 中的相对路径项。检查在 worktree 中执行，仓库中放置的 `sh.exe` 不能被当作解释器。
   - Windows 上的 `sh`、`bash`：先取 `internal/gitx` 所用的 git 所在的 Git for Windows 安装中的 `usr\bin\`（由 `git --exec-path` 推出安装根）；其次是 PATH 上的绝对路径项，但排除 Windows 目录之下和 `WindowsApps` 中的启动器（WSL）。
   - 系统目录由 Windows API（`GetSystemDirectory`）取得，不使用 `ComSpec`、`SystemRoot` 等环境变量指定解释器的位置。
4. **找不到解释器时（新增产品取舍）**：执行前对所有要执行的 AC 预检解释器。任何一个不可用，**不执行任何检查、不写记录**，返回退出码 6（`environment`），`error.details` 给出 `shell_unavailable` 和 AC 编号，不回显路径。
   - 与 `--isolation required` 的处理一致：环境问题不写成检查结果，否则一条"解释器缺失"会被记为 `fail`，让已验收的 AC 变为 `failing`（`regressed`）。
   - 平台上不存在的取值（Unix 上的 `cmd`、`powershell`）同样按不可用处理。
5. **命令行与参数**（实施细节）：
   - Linux / macOS：直接以 argv 传入（`[sh, -c, 命令]`），没有转义问题。
   - Windows：一律自行构造完整的命令行（`SysProcAttr.CmdLine`），不使用 Go 的默认转义：`sh`、`bash`、`pwsh`、`powershell` 的每个参数按 MSVCRT 规则总是加引号；`cmd` 用 `/d /s /c "<命令>"`。不使用 `-EncodedCommand`。
   - **`cmd` 与多行命令（新增产品取舍）**：`shell = cmd` 时命令不能含 CR 或 LF。推荐在提交、审核和人工编辑时拒绝（扩展 V-P04，退出码 2）；已经保存的此类检查，`check run` 在执行前拒绝（退出码 2），不写记录。
6. **结果映射**（实施细节）：
   - 退出码 0 → `pass`；
   - 非零 → `fail` / `nonzero_exit`，记录实际退出码；
   - `sh`、`bash` 的 127 → `error` / `not_found`（MVP-EV-04）；`cmd`、`pwsh`、`powershell` 无法可靠区分，按 `nonzero_exit` 处理；
   - 解释器已找到、但进程无法启动（包括工作目录不存在）→ `error` / `spawn_error`。
7. **路径**（实施细节）：
   - 工作目录是 `W/<workdir>`，按平台转换分隔符，开始时必须是存在的目录。
   - 命令文本原样传入，Specmetry 不做路径转换；Git Bash 对它所启动的程序做的 MSYS 路径转换，属于所选 shell 的语义。
   - 路径含空格和中文时正常（实测）。

**备选**：

- A：保留 TD-06 现在的建议（`sh` 找不到时回退到 `cmd /C`）。不推荐：静默回退，同一配置执行不同的程序；实测 `cmd` 对多行命令只执行第一行。
- B：Windows 默认 `cmd`，其他平台默认 `sh`。同一配置在不同平台上是不同的程序。
- C：Windows 上必须显式指定 `shell`。改变已登记数据的含义（M1 中省略 `shell` 的检查都要改），对使用者负担大。
- D：解释器缺失时记录 `error` / `spawn_error`。会把环境问题写成检查结果。

**影响**：

- 用户行为：没有 Git for Windows `sh` 的 Windows 机器，省略 `shell` 的检查返回退出码 6，需要安装 Git for Windows 或显式指定 `cmd` / `pwsh`。`cmd` 检查不能写多行命令。
- 现有数据：没有检查执行记录；M1 中已确认、省略 `shell` 的检查，含义统一为 `sh`。不需要迁移。
- Schema：取值不变；可以更新 `common.v1` 中 `checkSpec.shell` 的描述。V-P04 的扩展是校验规则的变化（data-model §8.3）。
- 测试：每种 shell 在每个平台上都要有测试。CI 运行器（Windows 有 Git Bash、cmd、两种 PowerShell；Ubuntu、macOS 有 sh、bash、pwsh）上解释器缺失时测试应当失败，不能静默跳过。
- 后续模块：M4 的 Skill 需要说明如何选择 shell；M3、M5 不受影响。

**建议的验收标准**（草案，可作为 EV-14、EV-15 或并入 EV-01、EV-04）：

1. 每个平台上，每种可用的 shell 用同一组测试字符串（含空格、引号、反斜杠、通配符、花括号、中文、`$`、`%`）经 `printf` 回显，结果与输入完全一致。
2. 省略 `shell` 的检查在三个平台上都由 `sh` 执行；Windows 上 PATH 中只有 WSL 的 `bash.exe` 时不使用它；worktree 中放置的 `sh.exe` 不被执行。
3. 解释器不可用（包括 Unix 上的 `cmd`）→ 退出码 6，`error.details` 中有 `shell_unavailable`，`check_runs` 不增加任何行，没有执行任何检查。
4. `shell = cmd` 的多行命令在提案和审核中被拒绝（退出码 2）；已保存的此类检查在执行前被拒绝，不写记录。
5. `sh` 中命令不存在 → `error` / `not_found`；超时后各 shell 启动的整个进程树都被终止（EV-03，含 Git Bash 启动的子进程）。

### 5.2 M2-Q2：TD-23 `check.inputs` 的对象 ID（草案）

**当前约束**（既有批准决定）：

- `inputs_json` 是开始时 `inputs` 中每个路径在 HEAD 上的 git 对象 ID；路径不存在时结果为 `error` / `input_missing`，计为失败（data-model §4.11、§7.4 第 2 步、§6.9）。
- 沿用要求 `S` 上这些路径的对象 ID 等于 `D.inputs_json`（§6.2）。MVP 没有沿用，但第 7 条规则的原因码 `inputs_changed` 照常推导（CX-19）。
- `inputs` 是仓库相对路径，最多 50 个，不重复；V-P04 做字面规范化；`check_config_hash` 包含排序后的 `inputs`。
- 对象读取一律经过 `internal/gitx` 的统一入口；git 无法安全读取时返回退出码 6，不把环境失败转换为正常结论（PD-26 的 M2 原则）。只支持 SHA-1（TD-26）。

**实测要点**（git 2.56）：

- `ls-tree -z --full-tree <提交> -- <路径>` 给出 mode、类型与对象 ID：普通文件 `100644 blob`，目录 `040000 tree`，子模块 `160000 commit <子模块的提交>`，符号链接 `120000 blob`。
- 链接文本与某个普通文件内容相同的符号链接，对象 ID 与该文件**相同**，只有 mode 不同。
- 目录路径末尾加 `/` 时，列出的是目录的内容而不是目录本身；普通文件末尾加 `/` 时没有输出。
- 子模块内部的路径、不存在的路径、未跟踪的空目录、大小写不同的路径（`up.txt` 对 `Up.txt`，即使 Windows 文件系统不区分大小写），都没有输出。
- 一次调用中给出互为前缀的两个路径时，只列出较深的那个。
- `ls-tree` 的路径按字面匹配（`src/*` 不匹配任何条目）。
- 子模块检出的提交与记录的不同时，`git status --porcelain` 报告 ` M sub`，工作区不是 `clean`。

**推荐方案**（实施细节，附两项行为确认）：

1. **读取**：在固定 `head_start` 之后、执行之前，对每个输入在 `head_start`（完整的提交 ID，而不是 `HEAD`）上用 `ls-tree` 查找**路径完全相同**的条目，经过 `gitx` 的对象读取入口。不把互为前缀的路径放进同一次调用。
2. **记录的内容**：每个输入记录路径（按声明的写法）、mode、类型与对象 ID，按路径排序，例如 `[{"path":"src/a.go","mode":"100644","type":"blob","oid":"…"}]`。比较时四项都要相同。
   - 理由：只记录对象 ID 时，符号链接与内容相同的普通文件无法区分（实测）。
   - `inputs_json` 是数据库内部的列，`check-run.v1`、`decision.v1`、`status.v1` 都不包含它，所以不改变交换契约；data-model §4.11、§4.13 的文字需要相应更新。
3. **各类路径**：

   | 路径 | 结果 | 说明 |
   |---|---|---|
   | 普通文件 | blob（`100644` / `100755`） | 可执行位的变化也算变化 |
   | 目录 | tree | 目录下任何受跟踪文件的变化都会改变它 |
   | 符号链接 | blob（`120000`） | 只反映链接文本；链接指向的文件变化不会被发现，需要时另行声明该文件 |
   | 子模块 | commit（`160000`） | 子模块的提交决定了它的内容；检出的提交不一致时，工作区不是 `clean`（实测），执行不合格 |
   | 子模块内部的路径 | `error` / `input_missing` | 上级仓库的树中没有这个路径；提示改为声明子模块本身 |
   | LFS 管理的文件 | 指针文件的 blob | 指针含有内容的 sha256 和大小，所以内容变化会改变指针；Specmetry 不运行 LFS 或任何过滤器 |
   | 不存在、未跟踪、只有被忽略的文件、空目录 | `error` / `input_missing` | data-model §7.4 第 2 步；推荐此时不再执行命令 |
   | 末尾带 `/` | 必须是目录，否则 `input_missing` | 比"去掉 `/` 再查"更严格 |
   | 大小写与提交树中不同 | `input_missing` | git 的路径区分大小写 |

4. **行为确认 1（对象读取失败）**：git 不具备安全读取对象的能力、对象缺失（例如部分克隆）或读取出错时，返回退出码 6，**不执行、不写执行记录**。这是 PD-26 的 M2 原则的应用，与"路径不存在"（`input_missing`，写记录）区分。
5. **行为确认 2（没有提交或非 git 项目）**：`head_start` 为 NULL（分支还没有提交）而声明了 `inputs` 时，记为 `input_missing`；非 git 项目不记录 `inputs_json`，执行本来就不合格（INIT-09）。

**备选**：

- 只记录对象 ID（`git rev-parse <提交>:<路径>`）：实现更简单，但符号链接与普通文件、子模块与其他类型的区别只能从对象类型间接看出，mode 的变化（例如可执行位）也看不到。
- 把末尾的 `/` 去掉再查：对使用者更宽松，但 `foo/` 可能匹配到普通文件 `foo`。
- 对象读取失败记为 `input_missing`：会把环境失败写成检查结果，违反 PD-26 的 M2 原则。

**影响**：

- 用户行为：声明了子模块内部路径或大小写不符的路径时，执行结果是 `input_missing`，需要改写检查配置（审核后生效）。
- 现有数据：没有执行记录，不需要迁移。
- Schema：不变。
- 测试：EV-10 扩展到上表各类路径；推导层的 `inputs_changed` 比较 mode。
- 后续模块：M7 的隔离执行器沿用同一份记录；M5 不显示 `inputs_json`。

**建议的验收标准**（草案，扩展 MVP-EV-10）：上表每一行各一个用例；声明的 `inputs` 在 A、B 两个提交上 mode 不同、对象 ID 相同 → 推导为 `inputs_changed`（CX-19 的变体）；git 不具备安全读取能力或对象缺失 → 退出码 6，`check_runs` 不增加任何行，没有运行检查命令。

### 5.3 M2-Q3：PD-23 `db recover`（草案）

**既有批准决定**（PD-23，2026-10-13；implementation-plan §8.1），不重新讨论：

1. 普通打开和 `doctor` 保持严格。
2. 恢复是独立命令 `specmetry db recover`，`doctor` 保持只读。
3. 仅限人、交互式确认，并实际检查 schema 版本、`min_binary_version` 和数据库一致性。
4. 恢复前必须已有一致性备份；不得直接删除 `-wal` / `-shm`；不得绕过版本兼容限制。
5. 排期 M2（WP2.10），必须在 M4 自测开始前完成。

适用状态（architecture §11.5）：WAL 模式数据库缺少 `-wal` 与 `-shm`；只有其中一个；存在非空的 `-journal`。

需要确认的是 §8.1 表中第 1 ~ 4 项的调整。

**推荐流程**（草案）：

1. **通道**：命令最先调用 `RequireHuman`。`agent_cli` 返回退出码 3，不读取任何文件，也不创建数据目录或锁文件。
2. **数据目录与锁**：按 §11.1 检查数据目录（不存在时说明没有数据库，退出码 0）；取得 `.migrate.lock`，持有到命令结束。
3. **清理遗留**：删除此前崩溃留下的恢复工作目录（见第 6 步）。
4. **判定状态**：与只读检查（`readonly.go`）使用同一套文件状态分类。不属于适用状态时说明无需恢复，退出码 0，什么都不改。
5. **外部活动检测**：
   - 用不写文件的锁探测，确认没有其他进程持有这个数据库的 SQLite 锁。Windows 上实测可行；Linux、macOS 用 `F_GETLK` 一类的查询，尚未验证。检测到 → 退出码 6，提示关闭其他程序。
   - 记录原数据库各文件的存在情况、大小、修改时间与 sha256。
6. **复制**：在数据目录中新建私有工作目录 `recover-<ULID>/`，把数据库文件和存在的辅助文件复制进去。复制之后重新计算原文件的状态，与第 5 步不同 → 删除工作目录，退出码 6（文件在复制期间变化，请重试）。
7. **在副本上恢复与检查**：
   - 以读写方式打开副本，由 SQLite 在副本上回放回滚日志或重建 WAL 索引（实测四种状态都可以）；
   - `PRAGMA integrity_check`、`PRAGMA foreign_key_check`；
   - 读取 `schema_version`、`min_binary_version`，用与普通打开相同的 `guardVersion` 判断；
   - 读取并校验 `tombstones.json`。

   任何一项失败 → 删除工作目录，退出码 6，原文件不变。
8. **在副本上执行墓碑**：删除墓碑中项目的行，然后整理（读取 checkpoint 的结果，依赖 N1 的修复），再做一次完整性检查。
9. **确认**（`channel.Guard`，不另设确认机制）：
   - 摘要包括：发现的状态与涉及的文件名，副本上的检查结果（schema 版本、完整性、墓碑将删除的行数），将生成的备份，"使用该数据库的其他程序必须先关闭"，以及恢复将由 SQLite 以读写方式打开原数据库一次。
   - 确认内容建议为 `recover`。取消 → 删除工作目录，退出码 3。
10. **确认之后复核**：重新比较原文件的状态与第 5 步。不同 → 删除工作目录，退出码 5（确认期间状态已变化），不生成备份。
11. **生成保留的备份**：对第 8 步之后的副本执行 `VACUUM INTO`，先写入 `backups/` 中的临时名称，完成后改名为正式名称。备份与迁移前备份同目录、同前缀、同轮换（保留 3 份），`purge` 时同样处理。
12. **恢复原数据库**：用与普通打开相同的读写连接配置（保留 WAL、`journal_size_limit = 0`）打开原数据库，由 SQLite 完成日志回放或 WAL 索引重建。在这个连接上重新做版本检查、完整性检查，并执行墓碑（同普通打开）。
    - 失败 → 退出码 6，报告原数据库已被 SQLite 打开、检查未通过，以及备份的位置；不删除备份。
13. **收尾**：关闭连接（保持 Specmetry 的持久 WAL 状态），删除工作目录，报告备份位置。之后普通打开与 `doctor` 照常工作。

`db recover` 不迁移：数据库版本比程序旧时，恢复之后由下一次普通打开迁移，并生成迁移前备份。

**§8.1 第 1 项：恢复前的检查与一致性备份**

- 当前约束：architecture §11.3"只使用一致性备份机制（`VACUUM INTO`），不直接复制正在使用的数据库文件"。在适用状态下，对原文件执行 `VACUUM INTO` 必须先用 SQLite 打开它，而打开本身就会修改文件。
- 推荐调整（§11.3 增加有限的例外）：
  - 只有 `db recover` 可以复制数据库文件，前提是：持有打开锁；锁探测没有发现其他进程；复制前后文件的状态（存在、大小、修改时间、sha256）一致。
  - 复制只放在数据目录内的私有工作目录中，只用于在副本上恢复和检查，不是备份，用完即删。
  - 保留的备份仍然只由 `VACUUM INTO` 生成。
  - "一致性检查"指 `integrity_check` 加 `foreign_key_check`，在恢复之后的副本上进行；版本检查使用同一个 `guardVersion`。
- 备选：
  - 用 `immutable=1` 只读打开原文件：不读 WAL，也不回放日志，在"只有 `-wal`"与"热日志"状态下读到的不是已提交的状态。不推荐。
  - 自行实现带写时复制的 VFS 或解析 WAL：审查 005 已证明自行解析会读到未提交的数据，并删除了那个方案。不推荐。
  - 只备份原始文件（不经 SQLite）：备份可能含已删除项目的数据，与第 2 项冲突。
- 影响：数据目录短暂出现工作目录（第 3 项）；磁盘需要约两倍于数据库的空间；大数据库的复制耗时与文件大小成正比。

**§8.1 第 2 项：墓碑与可保留备份的顺序**

- 当前约束：墓碑在任何备份之前执行，新生成的备份中不含已删除项目的数据（architecture §11.2、§11.4，CX-11，MVP-PRV-03）。
- 推荐调整：保留的备份只由第 8 步之后的副本生成。
  - `VACUUM INTO` 写出一个新的、紧凑的文件，只包含现存的行，所以备份中不会有已删除项目的数据，也没有空闲页。
  - 含有原始数据的中间副本只存在于私有工作目录中：正常结束和每一条拒绝路径都会删除它。
  - 崩溃留下的工作目录，由下一次 `db recover` 删除，并建议下一次兼容的普通打开（在版本检查通过之后、墓碑之前）和 `purge` 也删除它，`doctor` 报告它的存在。
  - 承诺边界与 §11.4 相同：不承诺物理擦除。
- 备选：
  - 同时保留原始文件的副本，作为"恢复之前的原样"：可能含已删除项目的数据，与 CX-11 冲突。只在没有墓碑时保留，可以作为备选，但需要再写一条例外。
  - 不保留备份，只依靠副本上的检查：违反 PD-23 第 4 条。

**§8.1 第 3 项：拒绝路径的临时文件与清理**

- 当前约束：MVP-STO-02 的严格标准（拒绝时数据库、WAL、SHM 和目录项都不变），以及两项已批准的例外（`.migrate.lock`；外部 SQLite 并发关闭时只读连接新建的空 `-wal` / `-shm`）。
- 推荐调整（写入新的验收项，不修改 MVP-STO-02）：
  - Agent 通道：任何文件都不变，也不创建 `.migrate.lock`。
  - 其他拒绝（不适用的状态、锁被持有、版本不兼容、完整性失败、墓碑文件无效、取消、复制期间变化、确认期间变化）：原数据库的各文件逐字节不变；命令结束时数据目录的条目集合与开始时相同（`.migrate.lock` 的例外除外）；不生成备份。
  - 命令运行期间会短暂出现工作目录。因为创建和删除了目录项，数据目录本身的修改时间可能改变。
  - 崩溃时工作目录可能留下，按第 2 项清理。
  - 外部进程自己造成的文件变化不算 Specmetry 的副作用（与 MVP-STO-02 的表述一致）。
- 备选：
  - 工作目录放在系统临时目录：系统临时目录可能授予其他账户权限（本仓库在 Windows 上的测试已经遇到过，见 development.md §6），也违背"数据只在受保护的数据目录中"。不推荐。
  - 在内存中完成检查：WAL 回放需要文件或自定义 VFS（见第 1 项）。不推荐。

**§8.1 第 4 项：范围、需求、CLI、人类专属权限与验收项**

- 范围：mvp.md §2"隐私与存储"一行加入"数据库恢复（`db recover`）"，需求加入 FR-STO-06。
- 需求（草案）：**FR-STO-06**（P0）：`specmetry db recover` 只能由人在交互式终端执行，只处理普通打开会拒绝的三类状态。恢复之前，在数据目录内的私有副本上完成日志恢复、完整性检查、版本检查与墓碑，并由副本生成一致性备份；确认之后才由 SQLite 以读写方式打开原数据库。拒绝时原数据库文件不变。没有跳过确认的参数。
- CLI：`specmetry db recover`，没有参数，不提供 `--yes`；成功时只输出文本，出错时带 `--json` 输出 `error.v1`（与其他人类专属命令相同）。
- 人类专属：AGENTS.md 约束 3 的列表加入"恢复数据库"；prd §3.3 中这一行去掉"待排期"；architecture §5 的拦截规则把 `db` 加入有二级子命令的命令组，拒绝 `db recover`；MVP-ACC-03 的"必须拒绝"清单加入它（M3 验证）；M4 的 Skill 禁止事项包括它。
- 新增验收项（草案，编号待定）：

  | 草案编号 | 前置 → 操作 → 期望 |
  |---|---|
  | STO-08 | Agent 通道、非 TTY、设置了已知的 Agent 环境变量 → `db recover` → 退出码 3，数据目录逐字节不变，不创建 `.migrate.lock`；必须有进程级或正式二进制的证据（acceptance-criteria §1） |
  | STO-09 | 用真实的 SQLite 程序构造四种状态（外部程序关闭后缺少两者；只有 `-wal`，含未合并的已提交帧；只有 `-shm`；崩溃留下的热日志）→ 确认恢复 → 退出码 0；已提交的数据保留（包括 WAL 中的帧），热日志被回滚；`backups/` 中有新备份，权限符合 §11.1，完整性检查通过，能被版本检查读取；之后普通打开成功，`doctor` 报告版本 |
  | STO-10 | 版本更新的数据库、`min_binary_version` 更高或无法解析、完整性检查失败、`tombstones.json` 无效、取消、不适用的状态 → 分别为退出码 6、6、6、6、3、0；原数据库的各文件逐字节不变，数据目录的条目集合不变（`.migrate.lock` 的例外除外），没有生成备份，没有留下工作目录 |
  | STO-11 | 适用状态的数据库中有已写入墓碑的项目（含 canary）→ 恢复 → 新备份、恢复后的数据库及其 WAL 中都找不到 canary；其他项目不受影响 |
  | STO-12 | 外部并发：另一进程持有该数据库的 SQLite 连接 → 退出码 6，文件不变；复制期间文件变化（测试钩子）→ 退出码 6，不生成备份；确认期间文件变化 → 退出码 5，不生成备份；另一个 Specmetry 进程在等待或超时于打开锁 |
  | STO-13 | 在复制之后、备份之后、恢复原数据库期间分别终止进程 → 下一次 `db recover` 或兼容的普通打开删除遗留的工作目录；不完整的备份从不带正式名称；原数据库要么未被改动，要么已由 SQLite 恢复，从不被副本替换 |

**外部 SQLite 进程的并发活动**（如实声明）：

- `.migrate.lock` 只协调 Specmetry 进程，外部的 SQLite 程序不认识它，不能把它当作能阻止外部写入的锁。
- 外部程序只受 SQLite 自身文件锁的约束。为了不修改原文件，恢复在第 12 步之前不在原数据库上持有任何 SQLite 锁。
- 推断：也不能在进程内另外持有原始的文件锁来阻挡外部程序。Unix 上 POSIX 锁在进程关闭该文件的任何描述符时都会全部释放；Windows 上按句柄的锁会挡住本进程自己随后的 SQLite 连接。
- 因此采用"检测而不阻止"：锁探测（第 5 步）、复制前后的比较（第 6 步）、确认之后的复核（第 10 步）；第 12 步起，由 SQLite 自身的锁协调外部访问，并在恢复后的连接上重新检查版本与完整性。
- 残余窗口：第 10 步复核之后、第 12 步建立连接之前，外部程序仍可能打开并修改数据库。此时 SQLite 对原数据库所做的，就是任何 SQLite 程序打开它时都会做的恢复；第 12 步的检查会发现不兼容或损坏，备份保留确认时的状态。这一窗口作为已知边界写入文档与命令说明，并提示用户先关闭其他程序。

**影响**：

- 用户行为：新增一个人类专属命令；恢复需要额外的磁盘空间，大数据库需要等待复制。
- 现有数据：不需要迁移；恢复后的数据库在下一次普通打开时按常规迁移。
- Schema：不变（只输出文本与 `error.v1`）。
- 测试：需要真实的外部 SQLite 进程与崩溃辅助进程（存储包已有同类基础）。
- 后续模块：M3 拦截规则与 `doctor`（报告遗留的工作目录）；M4 Skill 禁止事项；`backup restore`（P1）复用同一套检查与墓碑顺序。

### 5.4 M2-Q4：`status.v1` 中旧项目名称的兼容（草案）

**当前约束**：

- 方案 1（既有批准决定，review-resolution §48）**只针对 `init`**：已登记的名称含代码围栏时，`init` 返回退出码 2，不改名、不改写事实；不提供改名命令（第 6 条）。它不代表 `status` 或其他读取接口采用同样的策略（m1-exit-evidence 第 11 节）。
- architecture §10.2：所有自由文本（包括项目名称、交付目标名称）都不能包含代码围栏，由 schema 强制。`init.v1` 的 `project.name`（SCH-150）与 `target-scope.v1` 的 `target.name` 都已禁止；`status.v1` 的 `project.name` 和 `target.name` 只有长度限制。
- 交付目标名称不存在旧数据的问题：WP1.8 之前目标只能由 `init` 以 `default` 创建；合并进 master 的 `target set`（PR #14）始终拒绝代码围栏。
- `status --json` 是 Skill 的 `init`、`status`、`sync` 每一步都要读取的接口。

**推荐方案（B，新增产品取舍）**：`status` 照常推导，但不输出旧名称。

- `status.v1` 的 `project.name` 改为"不含代码围栏的名称，或 `null`"。`null` 只表示"登记的名称不被本版本接受，因此不输出"（项目名称本来非空，所以 `null` 没有歧义）。`target.name` 同时加上代码围栏的禁止。
- 人类可读输出显示项目 ID，并说明登记的名称不再被接受、因此不显示。
- 名称不是推导的输入，推导结果、差距与其他字段与名称合法时完全相同。
- `init` 的行为不变（方案 1）。

**备选**：

- A：`status`、`gaps` 像 `init` 一样拒绝（退出码 2）。实现最简单，但这类项目的交付推导完全不可用，又没有改名命令，唯一的出路是 `purge` 后重新初始化，丢失全部基准与事实。不推荐。
- C：新增人类专属的改名命令（例如 `project set name`），`status` 在改名之前拒绝。它同时解除方案 1 的局部限制，但违背方案 1 的第 6 条，需要负责人另行决定；还要修改 marker"只创建、不替换"的规则（architecture §10.6），并新增拦截、确认与验收项。
- D：原样输出旧名称。违反 §10.2 与 `init.v1` 的一致性，不推荐。

**影响**：

- 用户行为：只影响早期开发版本登记、名称含三个连续反引号的项目；它们可以正常使用 `status`、`gaps` 与后续命令，只是看不到名称。
- 现有数据：不改写，不迁移。
- Schema：`status.v1` 的 `project.name`（可为 null、禁止代码围栏）与 `target.name`（禁止代码围栏），补充示例与 SCH 反例。
- 测试：按 M1 的旧名称测试方式（用 SQL 写入旧名称）构造项目。
- 后续模块：
  - WP2.8 的 `purge` 要求输入项目名确认，对这类项目改为输入项目 ID（与 `source ack` 对无法输入的路径改用 ID 的做法相同）；
  - M3 的 `doctor --json`、M5 的项目列表采用同一规则。

**建议的验收标准**（草案）：

1. 名称含代码围栏的旧项目 → `status --json` 退出码 0，输出通过 `status.v1` 校验，`project.name = null`，其余字段与名称合法的同一数据库完全相同；文本与 JSON 模式的 stdout、stderr 中都找不到这个名称（canary）。
2. `gaps --json` 不受影响；`init` 仍按方案 1 返回退出码 2；整个过程不写入业务事实。
3. `status.v1` 拒绝含代码围栏的 `project.name` 与 `target.name`（新的 SCH 反例）。

### 5.5 M2-Q5：PD-02 主进度指标

- 当前约束：PD-02 的建议默认是"范围内需求的验收率 + AC 状态计数"，最晚 M2 决定。data-model §6.11 的 `scope_acceptance_ratio`（`RS` 中 `accepted` 的需求数 ÷ `RS` 中不是 `waived` 的需求数，分母为 0 时为 null）与 `ac_status_counts`，以及 `status.v1` 的 `metrics`，都已按它书写。
- 推荐：确认建议默认，不改 schema。
- 备选：按 AC 计算的验收率（`AS` 中 `accepted` 的 AC ÷ 不是 `waived` 的 AC）；按优先级加权。都需要修改 data-model §6.11 与 `status.v1`。
- 影响：WP2.1 的指标、WP2.7 的输出、M5 概览页的主指标。没有数据影响。
- 验收：指标的单元测试（含分母为 0）；ST-07 的 oracle。

### 5.6 M2-Q6：TD-13 人类专属操作的防护强度

- 当前约束：建议默认是"启发式通道识别 + 拦截 + 审计记录，并如实声明能力边界"，最晚 M2。M1 已实现通道识别（`channel.Detect`）、唯一的确认入口（`cli.confirm` → `channel.Guard`）、操作者与通道的记录；拦截在 M3。通道识别不是身份认证。
- 推荐：确认建议默认，并明确 M2 新增的人类专属命令都沿用同一机制，不另设确认机制：
  - `accept`、`reject`、`waive`，`claim` 的 `manual` 批次，`purge`，`project set excerpt-storage`，`db recover`，以及 P1 的 `backup restore`；
  - 每个命令最先调用 `RequireHuman`，经 `confirm` 确认；
  - data-model §4.13 规定 `decisions` 的通道只能是 `human_cli` 或 `dashboard`，迁移中用 CHECK 约束兜底。
- 备选：更强的隔离（例如由另一个操作系统用户执行人类专属操作、确认签名）。超出 MVP 的范围。
- 影响：WP2.4、2.6、2.8、2.10、2.11 的通道测试必须有进程级或正式二进制的证据（acceptance-criteria §1）。
- 验收：ACC-02、CLM-05、PRV-03、PRV-10 的非 TTY 部分，以及 STO-08（草案），各自带 Agent 环境变量与重定向的进程级用例。

### 5.7 M2-Q7：TD-17 提交关联规则

- 当前约束：建议默认是"trailer 优先；正文中的编号也匹配，可以关闭"，最晚 M2。只保存 SHA、提交时间和关联关系，正文在内存中处理（architecture §6.7、§10.2 第 6 条，FR-EV-06，EV-11）。按 ref 维护游标，历史被改写时重新扫描该 ref、按 SHA 去重（WT-04）。`settings_json` 预留了 `commit_ref_mode`。提交关联只影响进度，不影响交付状态。
- 推荐：
  1. trailer 的键为 `Specmetry-Ref`（按 git trailer 的惯例不区分大小写），值是逗号或空格分隔的编号。用 git 的 `%(trailers:key=…,valueonly)` 读取，不另外解析（git 2.56 上实测可用，键不区分大小写；最低版本 2.32 上未验证）。
  2. 正文中按完整单词匹配 `REQ-nnn` 与 `AC-nnn-n`。在 trailer 中出现的，`link_source = trailer`，否则为 `message`。
  3. 只关联本项目中存在的编号（包括已退役的，历史保留；进度只看 `active` 条目），其他编号忽略。
  4. MVP 不提供关闭正文匹配的开关，`commit_ref_mode` 固定为 `trailer_and_message`。误匹配（例如回滚提交中提到编号）只影响没有 Claim 的 AC 的进度。
  5. 扫描间隔（同一项目至少 30 秒）的记录方式是实施细节。
- 备选：提供 `trailer_only` 模式，通过人类专属的 `project set commit-ref-mode` 切换。需要新增命令、范围与验收项。
- 影响：WP2.3 的提交关联、进度的"存在关联提交"条件（data-model §6.5）、Skill 的提交说明（M4）。没有数据影响。
- 验收：WT-04、EV-11；trailer 与正文的解析、未知编号、已退役编号、改写历史的单元测试。

### 5.8 M2-Q8：条目级标记的表达（契约补充，草案）

- 当前约束：
  - MVP-SRC-04（M2 部分）要求引用"以 `source_removed` 的形式出现在状态与差距中"，FR-SRC-04、data-model §4.3 也这样写。但 data-model §6.6 的标记、§6.10 的差距类型、`status.v1` 与 `common.v1` 中都没有 `source_removed`。
  - MVP-BL-05（M2 部分）要求 `status --json` 带 `inferred` 标记，FR-BL-03 要求反推条目有醒目标识。反推的条目也可以是需求，而 `status.v1` 的需求没有 `flags`；`ref_not_found` 也只能标在 AC 上。
- 推荐：
  - `status.v1` 的需求增加 `flags`（取值 `inferred`、`ref_not_found`、`source_removed`）；
  - AC 的 `flags` 增加 `source_removed`；
  - data-model §6.6 增加 `source_removed`："当前修订的来源引用中，有指向已移除来源的"；
  - 差距增加不阻塞的标记类警告 `SOURCE_REMOVED`（`target_key` 为目标、需求或 AC 的编号），与 `CONFLICT` 等警告并列，所以目标的引用也能表达。
  - 这些都是标记，永不阻塞，不改变交付状态。
- 备选：
  - 只加标记、不加差距类型，同时把 SRC-04 的文字改为"出现在状态中"；
  - 只加差距警告、不给需求加 `flags`（BL-05 对需求的标记仍无处表达）。
- 影响：
  - Schema：`status.v1` 的需求结构、`common.v1` 的 `gapType`，补充示例与反例。
  - 推导：WP2.1 增加一条标记规则和一类警告。
  - 后续模块：M5 的标识（FR-UI-11、UI-12）可以直接使用。
  - 没有数据影响。
- 验收：SRC-04、BL-05 的 M2 部分分别覆盖需求与 AC；标记不改变交付状态，警告不阻塞（ST-13 的性质测试覆盖）。

### 5.9 M2-Q9：`claim --json` 的输出契约（契约补充，草案）

- 当前约束：prd §6 与 architecture §9.1 列出的面向 Agent 的输出中没有 `claim`；AGENTS.md 要求面向 Agent 的输出都通过 `schemas/` 校验；V-C06 的存在性校验"照常写入并标记"，MVP-INV-02 期望标记为 `missing` 或 `unknown_commit`，但 Agent 目前看不到这些标记。
- 推荐：新增 `claim-receipt.v1`，内容为：
  - 批次 ID、`seq`、`kind`、`base_version`；
  - 采集到的代码上下文（`worktree_id`、`head`、`tree_state`），`sync` 批次的比较起点；
  - 每条 Claim 的 AC、修订、断言，以及引用的存在性结果（文件的 `exists`、提交是否能被解析）；
  - 不回显 Claim 的摘要。

  在 WP2.4 中实现，并按"输出契约的语义约束"补充条件约束与反例。
- 备选：
  - `claim --json` 成功后输出整份 `status.v1`：职责混杂，输出很大；
  - 成功时不输出 JSON，只用退出码：Agent 看不到存在性标记。
- 影响：新增 schema、示例与反例；data-model §8、prd §6、architecture §9.1 与 AGENTS.md 的输出列表；M4 的 Skill 使用它。没有数据影响。
- 验收：CLM-01 ~ 05、INV-01、02 的成功输出按 `claim-receipt.v1` 校验；标记与数据库中的 `refs_check_json` 一致。

### 5.10 M2-Q10：§3 截止点的处理

- 当前约束：implementation-plan §3 把 TD-06、TD-23、`backup restore`（P1）、`db recover` 的截止都写为"M2 开工前"。
- 推荐：本轮一并确认 M2-Q1 ~ Q3，满足原截止点，不改计划。`backup restore` 是 P1，建议按第 4 节第 10 项把它的设计顺延到 WP2.8 的 P1 部分开工前；这是对原截止点的调整，需要负责人同意。
- 备选：如果某一项本轮不能决定，由负责人在授权首批实现时，把它的截止改为"对应工作包开工前"（TD-06 → WP2.5；TD-23 → WP2.3 的 `inputs` 部分；`db recover` → WP2.10），并在 implementation-plan §3 中记录这次调整。首批的 WP2.1、WP2.2 不依赖这三项。

### 5.11 实施细节（到对应模块之前写明，不需要产品决定）

| 工作包 | 事项 | 推荐 |
|---|---|---|
| WP2.3 | 工作区状态 | 按 data-model §1 的 `git status --porcelain --untracked-files=all -- . ':(exclude).specmetry'`，经过 `gitx` 的对象读取入口（它会比较索引与树），`GIT_OPTIONAL_LOCKS=0`；不忽略子模块 |
| WP2.4 | `manual` 批次 | 人类专属：最先调用 `RequireHuman`，再经 `confirm` 显示批次摘要并确认 |
| WP2.5 | `check run --all` 的对象 | 全部 `active`、`automated`、有检查的 AC，与范围无关；串行执行；任一不是 `pass` 时退出码 10 |
| WP2.6 | CLI 中的验收交互 | 像 `scope confirm` 一样由读到的当前状态构造 `decision.v1`，按 schema 校验后再确认（审查 034 B1 的教训）；必须确认的风险逐条显示 data-model §7.3 规定的文字，确认内容包括 AC 编号和每个风险的名称，人必须逐项输入；仍然只调用一次 `channel.Guard`，不另设确认机制。这一交互是用户可见的，建议在 WP2.6 开工前请负责人过目 |
| WP2.7 | `open_questions[].blocking` | 输出推导后的阻塞（关联条目已拒绝的问题不阻塞），与 `question list` 中记录的值区分，并在 schema 描述中写明 |
| WP2.8 | `purge` 的选择 | 保留备份、删除 `.specmetry/` 用参数给出，列在同一次确认的摘要中，不另设交互 |
| 迁移 | 新事实表 | 0007 起只前向；满足 data-model §1.1；附加只追加触发器（EV-07）；以 CHECK 约束兜底人类专属的通道（`decisions`、`manual` 批次）；`check_runs.evidence_grade` 在 MVP 中只接受 `observed`（不变量 9 的第二道防线，M7 再迁移放开） |
| 输出 | 语义约束 | `status.v1` 中拒收决定的快照不能为 null（ACC-07）；`check-run.v1` 的 `result` 与 `failure_kind` 一致；`gaps.v1` 中 `ready` 时没有阻塞差距；都补充反例与一致性测试 |
| WP2.9 | oracle 的独立性 | `expected-*.json` 由人按规则写出并审查，不由推导引擎生成 |

## 6. 需要修改的文档或契约

### 6.1 本轮修改（只同步状态，不改契约）

| 文档 | 修改 | 原因 |
|---|---|---|
| `docs/m2-readiness.md` | 新增本文 | 开工准备报告 |
| `AGENTS.md` | 当前阶段改为"M2 开工准备中，尚未授权业务实现"；记录负责人要求的协作闭环 | 状态同步；负责人本轮的明确要求 |
| `README.md` | 状态从"只有 M0"更新为 M0、M1 已验收，并如实列出已有与尚未实现的能力 | 状态同步 |
| `docs/implementation-plan.md` | M2 一节增加状态行，指向本文 | 状态同步 |

本轮不修改 mvp.md、prd.md、architecture.md、data-model.md、acceptance-criteria.md、`schemas/` 与迁移。

### 6.2 批准之后的修改

| 触发 | 文档或契约 | 修改 |
|---|---|---|
| M2-Q1 | architecture §6.4、§16（TD-06） | 默认 shell、字段映射、解释器查找、缺失时的处理、Windows 命令行 |
| | data-model §8.3（V-P04）、acceptance-criteria | `cmd` 不接受多行命令；EV-14、EV-15（或扩展 EV-01、EV-04） |
| | `common.v1`（可选） | `checkSpec.shell` 的描述 |
| M2-Q2 | data-model §4.11、§4.13、§7.4，architecture §16（TD-23） | `inputs_json` 的内容、各类路径、对象读取失败 |
| | acceptance-criteria | 扩展 EV-10 |
| M2-Q3 | architecture §11.3、§11.2 / §11.4，新增 §11.6 | 复制的例外、墓碑与备份的顺序、`db recover` 的流程与外部并发的边界 |
| | mvp.md §2，prd FR-STO-06、§3.3、§6、§9（PD-23） | 范围、需求、权限矩阵、CLI 概览 |
| | AGENTS.md 约束 3，architecture §5 拦截规则，acceptance-criteria（STO-08 ~ 13，ACC-03 的拒绝清单） | 人类专属与验收项 |
| | implementation-plan §3、§8.1、M2 退出条件 | 状态与退出条件 |
| M2-Q4 | `status.v1`、示例与 SCH 反例；architecture §6.1.2 中"M2 的 `status.v1` 中名称的处理留到 M2 设计"一句 | 名称可为 null、禁止代码围栏；记录决定 |
| M2-Q5 ~ Q7 | prd §9（PD-02），architecture §16（TD-13、TD-17）、§6.7，data-model §4.1、§4.12、§10 | 确认的结论与细化 |
| M2-Q8 | `status.v1`、`common.v1`（`gapType`），data-model §6.6、§6.10，acceptance-criteria SRC-04、BL-05 | 需求的标记、`source_removed`、`SOURCE_REMOVED` 警告 |
| M2-Q9 | 新增 `schemas/claim-receipt.v1.schema.json` 与示例；data-model §8；prd §6；architecture §9.1；AGENTS.md | `claim` 的输出契约 |
| 每项决定 | review-resolution | 登记决定及其依据 |

## 7. 维护事项的依赖节点

本轮只规划，不把这些修复混入准备 PR。所有未关闭的事项保持未关闭，不归入已接受的风险。

| 事项 | 依赖节点 | 说明 |
|---|---|---|
| M-7：成功 JSON 输出的测试只检查以 `{` 开头 | **WP2.4、2.5、2.7 新增输出接口之前**（批次 2 开头） | 统一改为按具体 schema 校验；单独的小 PR |
| M0 N1：checkpoint 返回 busy 时误删整理标记 | **WP2.8、WP2.10、WP2.11 兑现保证之前**（批次 5 开头） | 读取 `wal_checkpoint(TRUNCATE)` 的结果，确认完成后才删除标记，否则保留并报告"整理未完成"（architecture §11.4）；SRC-08 的 M2 部分验证"整理被阻塞时保留标记" |
| M-1：Windows 首次建库偶发 `Access is denied` | **M3 之前**（原计划不变） | Hook 与采集让并发成为常态之前处理 |
| M-2：`doctor` 的 `GIT_TRACE` | **v0.1.0 发布之前**（M6 的"必须关闭"清单，原要求不变） | 不是已接受的限制 |
| M-8：确认预览的修订类别、历史中的后续处理理由 | M5 的审核界面之前（原计划） | — |
| M-4、M-5、M-6、M-9、M-10、M-13 | 保留，按原优先级 | 不在 M2 的关键路径上；M-5 在将来支持嵌入或包装输出时处理 |

## 8. 首批实现：WP2.1 与 WP2.2

### 8.1 范围

- **WP2.1**：在 `internal/domain/derive` 中按 data-model §6 实现全部推导规则，测试先行；输入类型在本包中定义（事实截止 `as_of_seq`、目标与 `S`、来源的观测哈希、当前 worktree 的上下文），不依赖存储、git 或 CLI。
- **WP2.2**：独立的 `ready` 复核器，只在测试代码中使用。
- **不包括**：迁移、CLI 命令、schema 修改、读取数据库或 git 的装载代码（属于 WP2.3 ~ WP2.7）。

### 8.2 复核器如何保持独立

1. **依赖隔离**：复核器放在单独的测试专用包中，不导入 `internal/domain/derive`，也不导入 `scope`、`baseline` 等生产包的计算函数。核对逻辑只用标准库；E2E 阶段读取数据库的前端只额外使用 SQLite 驱动。由测试检查导入关系，两个方向都检查：复核器不导入推导引擎，生产代码不导入复核器。
2. **独立的表示与依据**：复核器定义自己的事实结构，按 acceptance-criteria §3 不变量 #2 与 prd §3.2 的 `ready` 定义逐条重新实现，不照搬 data-model §6 的决策表结构。具体检查：
   - `S` 不为 NULL；
   - 当前范围存在、没有过期、不为空，也不是全部豁免；
   - 范围内每个需求至少有一个 `active` AC；
   - 每个范围内的 AC 都有适用于 `S` 的当前决定；
   - `normal` 决定在 `S` 上有对应修订和配置的最新合格 `pass`，所依据的执行确实属于 `S`；MVP 中不存在隔离级执行，因此不存在沿用；
   - 独立重算的 `evidence_grade` 与输出一致；
   - 没有阻塞问题，也没有未确认的来源变化。

   E2E 阶段增加两个前端：直接用自己的 SQL 读取数据库，以及解析 `status.v1` 的输出。
3. **自身正确性不靠推导引擎证明**：
   - 每一条检查都有一个只违反这一条的反例（数据由测试直接构造），复核器必须指出这一条；另有一个完整满足的正例。
   - 这样，复核器的每条检查都被证明会失败，而不是空转。
4. **与推导引擎对照**：性质测试由测试专用的生成器产生随机事实，分别转换为推导引擎的输入与复核器的输入（转换代码各自独立）。
   - 推导为 `ready` ⇒ 复核器通过（不变量 #2）；
   - 复核器认为满足全部 `ready` 条件 ⇒ 推导为 `ready`。
   - 要求样本中有足够数量的 `ready` 结果，以免对照因为从不出现 `ready` 而空转。
   - 任何不一致都必须查明是哪一方或哪条规则的问题，不能为了一致而改动任意一方。

### 8.3 建议的验收标准

**WP2.1**：

1. 纯度：`internal/domain/derive` 只依赖标准库，由测试检查；同样的输入得到同样的输出（ST-14）。
2. ST-01：AC 决策表 11 条规则每条至少命中一次；每对可能同时成立的相邻规则都有优先级用例；第 7、10、11 条的原因码优先级各有用例。
3. ST-02、ST-03、ST-04：需求聚合 9 条、目标交付 6 条（每个状态都输出全部阻塞差距）、进度 8 种情况；§6.6 标记、§6.10 差距（类型、阻塞规则、有无范围、排序）、§6.11 指标各有用例。
4. ST-05：随机事实把物理顺序打乱 1000 次，结果完全相同；ST-13：`ready` ⇔ 没有阻塞差距，每个 AC 至多一个状态差距，范围外的条目没有差距。
5. ST-12 中属于推导的保守规则各有用例；ACC-09 的单元部分（合成的隔离级通过 → `carried_forward`；观测级 → `ready_for_review` / `observed_evidence`；输入变化 → `inputs_changed`）。
6. 列在第 3.1 节的 CX 都有推导层用例，测试名或注释中标明编号。
7. 不变量 9：输入中没有隔离级执行时，`delivery.evidence_grade` 只可能是 `observed` 或 `none`，观测级证据从不在没有人工决定时产生 `accepted`。
8. 实现中发现 data-model §6 的歧义或矛盾时，报告并等待决定，不自行选择。
9. M2-Q5、M2-Q8 有结论之后，对应的指标、标记与警告已实现。
10. 不包含迁移、CLI 与 schema 修改；三平台 CI 通过。

**WP2.2**：

1. 导入隔离由测试保证（两个方向）。
2. 覆盖第 8.2 节列出的全部检查，每条都有专门的反例，另有正例。
3. 与 WP2.1 的性质测试对照，两个方向都没有不一致，并且样本中有足够的 `ready` 结果。
4. 读取数据库与 `status.v1` 的前端，在事实表与 `status` 输出存在之后（WP2.4 ~ WP2.7）补齐，用于 CX 集成测试与 E2E。

## 9. 技术实验记录

实验都在会话的临时目录中进行：TD-23、TD-17 各用一个单独的临时 git 仓库，TD-06 用一个只依赖标准库的 Go 程序，PD-23 用本仓库的本地克隆加一个临时测试包（依赖来自本地模块缓存，没有下载）。没有提交，没有修改生产实现，原仓库不受影响。

环境：Windows 11 Pro 10.0.26300；Git for Windows 2.56.0.windows.2；Go 1.27.1；modernc.org/sqlite v1.60.1（仓库锁定的版本）；PowerShell 7.6.6；Windows PowerShell（`powershell.exe` 10.0.26100.8875）。

**实测结果**

| 实验 | 结果 |
|---|---|
| TD-23 | 见第 5.2 节"实测要点"：mode、类型与对象 ID 的取值；符号链接与普通文件的对象 ID 相同；末尾 `/` 的行为；子模块内部、缺失、大小写不符的路径没有输出；互为前缀的路径在一次调用中会丢失一个；路径按字面匹配；子模块提交不一致时工作区不干净；含空格和中文的路径正常 |
| TD-06 | 见第 5.1 节"实测要点"：PATH 上 `sh` 不存在、`bash` 是 WSL 启动器；默认转义下 Git for Windows `sh` 改写参数，总是加引号则原样；`cmd` 的引号与多行；PowerShell 的 `-Command` 与 `-EncodedCommand`；退出码；Go 1.27 的 `LookPath` 在当前目录含 `sh.exe`、PATH 中没有 `.` 时找不到它 |
| TD-17 | `git log --format='%(trailers:key=Specmetry-Ref,valueonly,separator=%x2C)'` 取出两个 trailer 行中的全部编号，键不区分大小写；正文中的编号不被取出（由正文匹配另行处理） |
| PD-23：缺少 `-wal` 与 `-shm` | 只复制主文件，在副本上以读写方式打开：数据完整，`integrity_check` 为 ok，SQLite 为副本新建了 `-wal`（0 字节）和 `-shm`；原文件逐字节不变 |
| PD-23：只有 `-wal` | 复制主文件与 `-wal`：WAL 中 5 行已提交、未合并的数据被恢复（共 8 行），`-shm` 重建，完整性 ok |
| PD-23：只有 `-shm` | 复制主文件与 `-shm`：只有已合并的 3 行。被删除的 `-wal` 中的数据无法恢复，也无法察觉 |
| PD-23：热日志 | 回滚日志模式，子进程在 `UPDATE` 中途溢出缓存后崩溃（数据库 335872 字节，日志 37376 字节）。复制两者并打开副本：2000 行全部回滚为原值，完整性 ok，日志被删除；原文件逐字节不变 |
| PD-23：`VACUUM INTO` | 由 WAL 模式数据库生成的文件，文件头第 18、19 字节为 1、1（回滚日志模式），与 architecture §11.5 的描述一致 |
| PD-23：锁探测（Windows） | 另一进程持有空闲的 WAL 连接时，对主文件 SQLite 锁字节区（`0x40000000` 起 512 字节）尝试立即失败的排他 `LockFileEx` 会失败，能检测到；探测随即释放，文件不变；该进程退出后检测不到 |

**推断**（没有实验，依据已知语义）：

- `ls-tree` 只读取树对象，不触发 LFS 或其他过滤器。
- Linux、macOS 上 argv 直接传给 `execve`，不存在 Windows 的命令行转义问题。
- 不能在进程内持有原始文件锁来阻挡外部 SQLite 程序（第 5.3 节"外部 SQLite 进程的并发活动"）。
- 数据目录内新建的工作目录继承受保护的 DACL（architecture §11.1）。

**未验证**：

- Linux、macOS 上的 PD-23 实验，以及用 `F_GETLK` 一类查询探测 SQLite 锁；
- Specmetry 自己的数据库（持久 WAL、迁移、触发器）在这些状态下的恢复；大数据库的复制耗时；复制与备份期间崩溃后的清理；
- Windows PowerShell 的 `-Command` 对完整测试字符串集的行为（只测了一个字符串）；`-OutputFormat Text` 能否消除 CLIXML；
- 从 Windows 直接启动的 Git Bash `sh -c` 中，`/usr/bin` 下的工具是否在 PATH 上（MVP-WT-02 用的 `test -f` 是 shell 内建命令，不受影响）；
- Job Object 能否终止 Git Bash 启动的全部子进程（EV-03）；MinGit、scoop 等其他 git 安装方式中 `sh` 的位置；
- 安装了 git-lfs 并启用过滤器时 `ls-tree` 的结果。

## 10. 建议交给 Codex 审查的内容

1. **基线与范围**：本文第 1、2 节与 implementation-plan、acceptance-criteria 的一致性；退出条件有没有遗漏或扩大。
2. **工作包映射**：第 3.1 节中每个标注 M2 的 P0 项与 CX 是否都落在某个工作包上。
3. **契约缺口**：第 1 节列出的三处缺口（M2-Q4、Q8、Q9）是否属实，是否还有同类缺口。
4. **M2-Q1 / Q2 的技术判断**：Windows 命令行的构造、解释器查找的安全性（WSL 启动器、仓库中放置的可执行文件）、`inputs` 记录 mode 的必要性；可以在隔离副本中独立重做第 9 节的实验。
5. **M2-Q3 的安全性**：
   - 恢复流程能否满足 PD-23 的全部既有约束；
   - 拒绝路径是否真的不改动原文件；
   - 中间副本与已删除项目的数据；
   - 外部 SQLite 并发的边界是否如实；
   - 第 12 步之后的失败处理。
6. **M2-Q4 的取舍**：推荐方案 B 是否与方案 1 的批准范围相容。
7. **复核器的独立性**：第 8.2 节的做法能否真正避免复用推导逻辑来证明自己。
8. **分类是否准确**：标为"实施细节"的内容中，有没有实际改变用户可见语义、因而应当由负责人决定的。

## 11. 下一步

| 顺序 | 事项 | 接手方 |
|---|---|---|
| 1 | 独立审查本报告（第 10 节） | Codex |
| 2 | 对 M2-Q1 ~ Q10 作出决定；决定是否授权首批实现（WP2.1、WP2.2） | 产品负责人 |
| 3 | 按决定把批准的内容写入规范文档（第 6.2 节），单独的文档 PR | Claude Code |
| 4 | 首批实现（另一个 PR），之后按第 3.2 节的批次推进 | Claude Code，经 Codex 审查 |
