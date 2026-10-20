# WP2.4 Claim 规范

> 状态：**规范草案，待独立审查，尚未授权实现，未冻结**（2026-10-20）。产品负责人 2026-10-20 批准了 M2-Q6（TD-13 的建议默认）与 M2-Q9（`claim-receipt.v1` 的独立回执方向），见 [review-resolution.md §76](review-resolution.md)；之后又决定了本文 §14 的 D1 ~ D3（§77）。本文据此写出可审查的契约；E1 ~ E6 仍是待审的工程细化，具体规范与 schema 仍须独立审查后冻结。本文不是实现：不含 Claim 的生产代码与迁移 SQL。
>
> 规范性的条文仍以 [data-model.md](data-model.md)（§4.10、§6.5、§6.6、§6.11、§6.13、§7.1、V-C01 ~ V-C06）、[architecture.md](architecture.md)（§5.1、§6.1.4、§6.3、§9.1、§10）、[prd.md](prd.md)（FR-CLM、FR-INV、FR-WT-02）、[acceptance-criteria.md](acceptance-criteria.md) 与 [`schemas/`](../schemas/) 为准；本文给出处理顺序、回执契约、待决事项与测试计划。两者不一致时，以那些文档为准，并修正本文。

## 1. 范围

**在范围内**：

- `claim --file <path|-> [--json]`：读取、校验、记录一个 Claim 批次（`claims.v1`），输出 `claim-receipt.v1`（带 `--json`）或文本摘要。
- 三种批次：`sync`、`audit`、`manual`；`audit_scope`；`sync` 的 `compare_from`。
- 写锁下的前置条件：`base_version`、每条 Claim 的 `target_revision`。
- 代码上下文的采集：`worktree_id`、`head`、`tree_state`。
- 引用的存在性检查（V-C06，只标记）；PD-26。
- `manual` 批次的通道拒绝、确认与审计（M2-Q6）。
- 未映射工作（`unmapped_work`，P1）：保存为 `target_type = none` 的 Claim。
- 输出契约 `claim-receipt.v1`，以及它与写入事实的一致性。

**不在范围内**：

- 推导规则与复核器：Claim 如何影响进度、标记、交付状态与指标，仍是 data-model §6 的规则（WP2.1、WP2.2 已验收），本文不改变。
- `status`、`gaps` 的输出（WP2.7）；会话与 Hook（M3）：在 M3 之前没有会话，`session_id` 为 NULL，`compare_from` 不会取 `session_start`（implementation-plan 的 WP2.4 行）。
- 检查执行（WP2.5）、验收决定（WP2.6）。
- 新的隔离机制（签名、独立的操作系统用户），M3 的 PreToolUse 拦截。

## 2. 依据与已有的决定

| 依据 | 内容 | 本文的用法 |
|---|---|---|
| data-model §4.10 | `claim_batches`、`claims` 的字段；一次提交一个 `seq`，批次与全部 Claim（含未映射工作）共用；worktree 首次出现时另取一个 `seq` | §6、§8 |
| data-model §7.1 | Claim 批次携带 `base_version` 与每条 `target_revision`，写事务内比对，不一致整体拒绝（退出码 5） | §5 第 10 步 |
| V-C01 ~ V-C06 | 严格 schema；`manual` 仅限 `human_cli`；当前值；`active` AC、批次内不重复、`unknown` 仅限 `audit`；摘要规则与密钥；`audit` 必须有 `audit_scope`；引用存在性只标记 | §4、§5 |
| FR-CLM-01 ~ 05、FR-INV-01、02、06、07、FR-WT-02 | 提交、不单独产生 `accepted` / `ready`、摘要、前置条件与代码上下文、目标与 `manual` | §4 ~ §9 |
| M2-Q6 / TD-13（2026-10-20 批准） | 沿用启发式通道识别、`RequireHuman`、现有 `confirm` 入口、操作者与通道审计；通道识别不是身份认证；不加签名或独立用户；不提前实现 M3 Hook。Claim 中只有 `manual` 批次是人类专属 | §4 |
| M2-Q9（2026-10-20 批准方向） | 新增 `claim-receipt.v1`：批次身份、`seq`、`kind`、`base_version`；采集的 `worktree_id`、`head`、`tree_state`；只有 `sync` 有 `compare_from`（起点与来源）；`claims` 与输入逐项对应、顺序一致，含索引、AC、修订、断言与引用存在性结果；无引用、空 `claims` 与未映射工作都有明确表达；不回显摘要、正文、命令、输出或环境变量的值；存在性不代表验证或验收通过 | §9 |
| PD-26 | git 只经 `internal/gitx` 的受控入口读取对象；git 会按需抓取对象时不读取 | §7、§8 |
| WP2.3（库层 2026-10-20 验收） | `gitx.Clean`（P1：不能判定为 `undecidable`）、`gitx.NewScratch`、`CommitOf` 等 | §6、§7 |
| M2-Q2、TD-23 | 已冻结，本文不重开 | — |

## 3. 命令与输入

- 命令：`specmetry claim --file <path|-> [--json]`。`-` 表示从标准输入读取，大小上限与 `propose` 相同（8 MiB）。
- 输入：`claims.v1`（严格校验，未知字段视为错误）。
- 任何通道都可以提交 `sync`、`audit` 批次；`manual` 批次只能由人在 `human_cli` 中提交（V-C01，§4）。
- 提交只写入事实，不改变基准、范围、决定或任何人类专属的状态。

## 4. `manual` 批次：通道、确认与审计（M2-Q6）

**只有 `manual` 批次是人类专属**；`claim` 命令本身不是。因此通道检查在识别出批次类型之后进行，而不是在命令开始时。

1. **识别批次类型**：读取输入，按 `claims.v1` 严格校验通过之后，取 `kind`。不合法的输入一律退出码 2（错误只给 JSON Pointer 与约束，不引用内容），与通道无关；两种情况都不写入。
2. **通道拒绝**：`kind = manual` 时，立即调用通道识别结果的 `RequireHuman`：
   - `agent_cli`（包括 stdin 或 stdout 不是终端、设置了已知 Agent 环境变量、`--file -` 从管道读入）→ 退出码 3（`human_required`），不显示提示、不读取确认输入；
   - 这一步之前不做密钥扫描之外的任何处理，**不打开数据库，不运行 git，不读取代码**，不写入任何数据。
3. **输入处理与数据库访问**：通过通道检查之后，与其他批次相同（§5 第 3 ~ 8 步）。
4. **确认**：所有不需要确认就能判定的检查都通过之后，先在一个回滚的写事务中完整演练一次写入（与 `propose --apply` 相同），确认能写入，再经 `internal/cli` 的 `confirm`（它调用 `channel.Guard`）请人确认：
   - 显示：操作名 `claim (manual)`、`base_version`、Claim 的条数，以及每条的 AC 编号、修订与断言、引用的存在性结果；摘要不显示。显示经 `termtext` 转义。
   - 确认内容：`v<base_version>`（与 `propose --apply` 的确认方式一致；工程细化，待审查）。
   - 带 `--json` 时，确认显示在 stderr；stderr 不是终端时退出码 6，不写入（architecture §5.1 规则 2）。
   - 取消、内容不一致、输入结束 → 退出码 3，不写入。不提供 `--yes`。
5. **写入**：确认之后，在新的写事务中重新检查全部前置条件（确认期间状态可能变化：过期退出码 5，目标不再有效退出码 2），然后写入。
6. **审计**：批次记录 `actor` 与 `channel`：人在终端中为 `human:<显示名>` 与 `human_cli`；Agent 通道为 `agent:<agent>`（设置了 Claude Code 的变量时为 `agent:claude_code`，否则为 `agent:unknown`）与 `agent_cli`。实现须在迁移中加约束兜底：`kind = manual` 的批次 `channel` 只能是 `human_cli`（与 M2-Q6 对 `decisions` 的要求相同；工程细化，待审查）。
7. **能力边界**（如实声明）：通道识别不是身份认证。Agent 经伪终端并清除环境变量执行时，可能被识别为 `human_cli`；交互式确认、Skill 禁令与审计记录只是补充（architecture §5.1）。M3 的 PreToolUse 拦截按参数位置匹配，看不出输入 JSON 中的 `kind = manual`，所以对 `manual` 批次只能依靠通道识别（architecture §5 的已知情形）。

## 5. 处理顺序

拒绝为退出码 2，过期为退出码 5，人工为退出码 3，环境为退出码 6；**这些情形都不写入任何数据，也不分配 `seq`**。错误输出不引用输入的值（§10）。

1. **读取输入**（大小上限 8 MiB）。
2. **严格校验** `claims.v1`（V-C01）：包括 `audit` 必须有 `audit_scope`（V-C05）、`unknown` 只能用于 `audit`、摘要的长度与形式（V-C04 中 schema 能表达的部分）、引用的格式。
3. **`manual` 的通道检查**（§4 第 2 步）。
4. **不需要数据库的规则**：
   - 同一批次中同一 AC 至多一条 Claim（V-C03），重复时退出码 2，指向后出现的那条；
   - 摘要单段、不含代码围栏（V-C04；schema 之外再检查一次）；
   - 代码引用的路径按字面规范化之后不得越出仓库根，行号的规则与 `propose` 相同（同一个解析函数，architecture §6.1.4 第 3 步）。
5. **密钥扫描**（V-C04、architecture §10.8）：输入中的**每个**字符串值，包括批次摘要、每条 Claim 与未映射工作的摘要、测试名、路径与提交前缀，以及可能被登记的当前 worktree 路径和 git 目录。命中即退出码 2，指向命中的位置，不引用内容。
6. **确定项目并打开数据库**（architecture §6.1.2）：没有数据库或项目未登记时退出码 4。打开时适用的版本、墓碑与只读拒绝按既有顺序先行；之后才准备私有临时区（`gitx.NewScratch`，WP2.3 §14.3；审查 059 关于 Scratch 先后顺序的要求）。
7. **采集代码上下文**（在写事务之外，§6）：`worktree_id` 所需的 worktree 定位、`head`、`tree_state`。
8. **检查引用**（在写事务之外，§7）：文件引用对照第 7 步采集的 `head` 的提交树；提交引用解析为提交。只标记，不拒绝（V-C06）。
9. **`manual`**：回滚的演练写入与确认（§4 第 4 步）。
10. **写事务**（`BEGIN IMMEDIATE`，先取得写锁，再读取比较）：
    1. `base_version` 不等于当前基准版本（没有版本时为 0）→ 退出码 5，`error.current.base_version` 给出当前值（V-C02）。
    2. 每个 `target` 必须是 `active` 的 AC：不存在、已退役、只存在于待审提案中 → 退出码 2，`details` 逐项给出 `/claims/<i>/target`（V-C03，MVP-CLM-02）。
    3. **D2**（2026-10-20 决定）：`audit_scope.requirements` 中的每个编号必须是当前 `active` 的需求；不存在或已退役 → 退出码 2，`details` 逐项给出 `/audit_scope/requirements/<j>`。
    4. **D3**（2026-10-20 决定）：`audit_scope = {"requirements": [...]}` 时，每条 Claim 的目标 AC 当前所属的需求必须在声明的编号之中；否则退出码 2，`details` 给出 `/claims/<i>/target`。`{"all": true}` 时任何 `active` 的 AC 都属于范围。不要求一次提交覆盖范围内的全部 AC，空的 `claims` 照常合法，不新增完整盘点的门槛。
    5. 第 2 ~ 4 步的问题在同一个错误中一并给出（与 `propose` 相同）；有这类问题时不再判断第 6 步。
    6. 每个 `target_revision` 必须等于该 AC 的当前 revision → 退出码 5，`error.current` 给出 `base_version` 与 `targets`（逐项的 JSON Pointer、编号与当前 revision；与 `propose` 相同）（V-C02，CX-13）。
    7. 当前 worktree 第一次出现时登记，另取一个 `seq`（data-model §4.10，与提案相同）。
    8. `kind = sync` 时按 §6.3 计算 `compare_from`（读取同一事务内的既有批次）。
    9. 分配**一个** `seq`，写入批次、全部 Claim（`target_type = criterion`）与全部未映射工作（`target_type = none`）。
    10. **用刚写入的行构造回执，并按 `claim-receipt.v1` 校验**（§9.3）。校验失败是内部错误（退出码 1），整个事务回滚，不写入任何数据。
    11. 提交。任何一步失败，整个事务回滚。

第 2 ~ 6 步都在写锁之下读取当前的实体、修订与 AC 所属的需求（`BEGIN IMMEDIATE` 之后），不使用写锁之前读到的值。
11. **输出**（§9）。

客户端不自动重试：收到退出码 5 时，重新读取基准与 revision 之后再提交（data-model §7.1）。

## 6. 代码上下文

### 6.1 采集的值

| 字段 | 取得方式 | 说明 |
|---|---|---|
| `worktree_id` | 命令所在的 worktree，经 `gitx.Locate` 定位，按 data-model §4.1 查找或登记 | 非 git 项目为项目根所对应的 worktree 记录（与 `propose` 相同） |
| `head` | `gitx.CommitOf(HEAD)`，完整的 40 位提交 ID | 没有任何提交时为 NULL；非 git 项目为 NULL |
| `tree_state` | `gitx.Clean`（WP2.3）：`clean`、`dirty`；`undecidable`（P1）记为 `unknown` | 非 git 项目为 `unknown`；读取失败见 §6.2 |

- 三个值在第 7 步一次采集，写入批次并原样出现在回执的 `context` 中；第 8 步的文件引用对照**同一个** `head`，不另行解析 HEAD。
- 采集与提交之间 HEAD 或工作区可能变化；批次记录的是提交时 CLI 看到的上下文，不声称在写入时仍然成立（与观测级的说明一致）。

### 6.2 读取失败（D1，2026-10-20 决定）

- `gitx.Clean` **返回实际的环境错误**（例如 git 不满足 PD-26、部分克隆中缺少对象、私有临时区无法建立或按方案 B 保留并失败）时：退出码 6，**不写入本次 Claim 的业务事实**（批次、Claim、未映射工作，以及只为本次提交而做的 worktree 登记），**不分配本次写入的 `seq`**。
- `gitx.Clean` 正常返回 `undecidable`（P1）时仍记 `tree_state = unknown` 并继续；非 git 项目的 `unknown` 也不变（§6.1）。二者不是环境错误。
- `head` 指向的提交无法读取时，`CommitOf` 返回环境错误 → 同样退出码 6，不写入（PD-26）。
- **退出码 6 不表示数据目录或辅助文件绝对不变**：在这之前，打开数据库按既有契约进行（architecture §11）；私有临时区按 WP2.3 §14.3（方案 B）建立，并按其生命周期清理，或保留并报告（§10）。本条只保证不写入本次 Claim 的业务事实、不分配 `seq`，不扩大正常打开或方案 B 的保证。
- **理由与代价**：`tree_state` 是批次必需的代码上下文（data-model §4.10），WP2.3 也规定读取失败不报告 `clean`；把读取失败记成 `unknown` 会让批次在不安全的 git 或残缺的仓库中照常写入。代价是这类环境中所有 `claim` 都不可用，直到环境修复。

### 6.3 `compare_from`（只用于 `sync`）

architecture §6.3 的优先级，在写事务内计算：

1. **`previous_batch`**：同一 worktree 中 `seq` 最大的 `sync` 或 `audit` 批次，其 `tree_state = clean` 且 `head` 不为 NULL → `commit` 为该批次的 `head`。
2. **`session_start`**：本会话的 `head_start`（M3 才有会话；**M2 永不取此来源**）。
3. **`none`**：`commit = NULL`。

只有第 1 条最新的那个批次参与判断：最新的批次不满足条件时，不再向更早的批次回溯（与"上一批次"的含义一致；工程细化，待审查）。`audit`、`manual` 批次不记录 `compare_from`（NULL），回执中也没有这个字段。

## 7. 引用检查（V-C06，只标记）

`refs` 可以有 `files`、`tests`、`commits`，Claim 与未映射工作相同。

| 引用 | 检查 | 结果 | 标记（MVP-INV-02） |
|---|---|---|---|
| 文件 `{path, lines}` | git 项目：路径必须是 §6.1 采集的 `head` 提交树中的普通文件（不是目录、符号链接或子模块），有行号时必须在文件的行数之内；未提交的文件、没有提交的 worktree 都记为不存在。非 git 项目：项目根下的普通文件，解析链接之后仍在根内。复用 `proposal.CheckCodeRefs` 的规则（architecture §6.1.4 第 6 步），只读取行数 | `exists` | `exists = false` 即 `missing` |
| 提交前缀 | 经 `internal/gitx` 的受控入口解析为恰好一个可读的提交对象 | `resolved` | `resolved = false` 即 `unknown_commit` |
| 测试名 | 不检查 | 只计数 | — |

- **提交前缀的结果**：不存在、不是提交、前缀有歧义（对应多个对象）→ `resolved = false`；非 git 项目与没有提交的仓库 → `resolved = false`。存在但对象无法读取（例如部分克隆中缺失）→ 退出码 6，不写入（PD-26）。歧义记为 `false` 而不拒绝，与"只标记，不拒绝"一致（工程细化，待审查）。实现需要一个只读、只接受十六进制前缀、按 PD-26 读取对象的 `gitx` 函数；现有的 `CommitOf` 对歧义前缀返回环境错误，不能直接使用。
- **PD-26**：git 不具备安全读取对象的能力（会按需抓取）时，含文件或提交引用的批次返回退出码 6，不写入；识别出部分克隆时同样（与 `propose` 的代码引用相同）。没有引用的批次只受 §6.2 的影响。
- **保存**：`refs_json` 保存提交的引用（路径与行号、测试名、提交前缀）；`refs_check_json` 保存结果：`{"files": [{"path", "lines", "exists", "checked_head"}], "commits": [{"commit", "resolved"}]}`，顺序与 `refs_json` 相同，`checked_head` 为 §6.1 的 `head`。不保存任何文件内容。
- **含义**：存在性只说明引用的位置存在，**不是**验证、证据或验收（FR-CLM-02）；回执与文本输出都如实说明这一点。

## 8. 写入、`seq` 与身份

- 一次 `claim` 提交是一次逻辑写入：批次、全部 Claim 与全部未映射工作在**同一个**写事务中写入，共用**一个** `seq`（data-model §4.10、§6.13，F4）。worktree 的首次登记另取一个 `seq`，在同一事务中先于批次分配。
- 批次与每条 Claim 各有 ULID `id`；Claim 经 `batch_id` 归属批次。`claim_batches.seq` 在表内唯一，`claims.seq` 可以多行同值。
- 同一批次中同一 AC 至多一条 Claim（V-C03），所以推导按 AC 取 `seq` 最大的 Claim 时没有并列（data-model §6.5）。
- `claim_batches`、`claims` 是只追加的事实表（data-model §2）：不更新、不删除（`purge` 除外）。
- 提交不幂等：同一输入提交两次，得到两个批次与两个 `seq`；按 AC 取最新的那一条。客户端收到错误后不自动重试。
- 字段取值：`kind`、`base_version`、`summary`、`audit_scope_json`（仅 `audit`）来自输入；`worktree_id`、`head`、`tree_state`、`compare_from`、`actor`、`channel` 由 CLI 决定；`session_id` 在 M3 之前为 NULL。每条 Claim 的 `criterion_id` 为目标 AC 的内部 ID，`criterion_revision` 为输入的 `target_revision`（已在写锁下核对为当前值），`assertion`、`summary` 来自输入。

## 9. 输出：`claim-receipt.v1`

### 9.1 字段

规范定义在 [`schemas/claim-receipt.v1.schema.json`](../schemas/claim-receipt.v1.schema.json)。

| 字段 | 内容 | 来自 |
|---|---|---|
| `schema` | `specmetry.claim-receipt/v1` | — |
| `batch_id`、`seq` | 批次的 `id` 与 `seq` | 写入的 `claim_batches` 行 |
| `kind`、`base_version` | 批次类型与提交者读取的基准版本 | 同上 |
| `context` | `worktree_id`、`head`、`tree_state`（`common.v1` 的 `codeContext`） | 同上 |
| `compare_from` | 只在 `kind = sync` 时出现：`commit` 与 `source`；`source = none` 时 `commit` 为 null，否则为完整 SHA | 同上 |
| `claims[]` | 每条输入 Claim 一项，顺序与输入相同：`index`（输入中的下标）、`claim_id`、`target`、`target_revision`、`assertion`、`refs` | 写入的 `claims` 行（`target_type = criterion`） |
| `unmapped_work[]` | 每项输入的未映射工作一项，顺序相同：`index`、`claim_id`、`refs`；没有时为空数组 | 写入的 `claims` 行（`target_type = none`） |
| `refs` | `files[]`（`path`、`lines`、`exists`）、`commits[]`（`commit`、`resolved`）、`tests`（测试名的数目）；没有引用时为两个空数组与 0 | `refs_check_json`（去掉 `checked_head`，它等于 `context.head`）与 `refs_json` 中测试名的数目 |

**不回显**：批次与 Claim 的摘要、未映射工作的摘要、测试名、命令、输出、环境变量的值。回执中出现的路径、行号与提交前缀都是提交者自己给出、并已通过密钥扫描的引用位置。

### 9.2 JSON Schema 能表达的与必须由业务测试验证的

| 条件 | 由谁保证 |
|---|---|
| 字段齐全、没有多余字段（因而不会出现摘要、测试名等）；枚举；编号、SHA、ULID、行号的格式 | schema |
| `compare_from` 当且仅当 `kind = sync` 时出现；`source = none` 当且仅当 `commit = null` | schema |
| `unknown` 断言只出现在 `audit` 回执中 | schema |
| `claims[i].index = i`、`unmapped_work[i].index = i`；条数与输入相同；顺序与输入相同 | 业务测试 |
| `target`、`target_revision`、`assertion` 与输入相同，`target_revision` 与写入的 `criterion_revision` 相同 | 业务测试 |
| `batch_id`、`seq`、`kind`、`base_version`、`context`、`compare_from` 与写入的批次行相同；`seq` 是批次与全部 Claim 共用的那一个 | 业务测试 |
| `claim_id` 与写入的行相同，互不相同 | 业务测试 |
| `refs` 与写入的 `refs_check_json` 一致，`tests` 等于 `refs_json` 中测试名的数目；`exists` / `resolved` 与实际仓库一致 | 业务测试 |
| M2 中 `compare_from.source` 不是 `session_start`；`previous_batch` 的 `commit` 是 §6.3 选出的那个批次的 `head` | 业务测试 |
| 输出（包括错误输出）中没有任何摘要、测试名或环境变量的值（canary） | 业务测试 |

### 9.3 校验与提交的顺序

现有的 `propose --json` 在事务提交之后才按 schema 校验回执（`writeJSON`）。校验失败时数据已经写入，却返回内部错误。`claim` 不沿用这个顺序：

1. 写事务中，用刚写入的行（不是输入）构造回执；
2. 在**提交之前**按 `claim-receipt.v1` 校验；失败 → 回滚，退出码 1，不写入；
3. 提交；
4. 输出已校验的文档。提交之后只剩编码与写出两步：写出失败返回退出码 6（"cannot write the output"），**批次已经写入**，不回滚，也不能撤回。调用者应先重新读取状态，再决定是否重新提交（重新提交会得到一个新的批次，§8）。

`propose` 的现有顺序不在本工作包中改变；是否统一，列为后续事项（§14 的 E5）。

### 9.4 文本输出与错误

- 三种批次都可以带 `--json`；`manual` 批次带 `--json` 时，确认提示在 stderr，成功时 stdout 同样只有一份 `claim-receipt.v1`（与 architecture §9.1 中"已有成功 JSON 契约的人类专属操作，成功时输出该文档"一致，同 `propose --apply`）。
- 不带 `--json` 时输出文本摘要：批次类型、`seq`、基准版本、代码上下文、`compare_from`、各类结果的数目，以及不存在的文件引用与无法解析的提交引用的位置；不含任何摘要或测试名；说明存在性不是验收。
- 错误：带 `--json` 时为完整的 `error.v1`（stdout），stderr 为空（`manual` 确认之后的错误例外，同 architecture §9.1）；不带时为文本。两者都只给 JSON Pointer、约束与当前值，不引用输入的内容。

## 10. 隐私

- **保存**：摘要（C1，≤ 500 字符）、引用的路径与行号、测试名、提交前缀与检查结果；不保存文件内容、命令、输出、环境变量的值或会话正文（architecture §10.1、§10.2）。摘要保存在本地数据库，直到 `purge`（data-model §9）。
- **拒绝而不替换**：任何字符串值命中密钥规则即退出码 2，不写入（§5 第 5 步）。
- **不回显**：回执、文本输出、确认显示与错误都不包含摘要、测试名或环境变量的值；错误只给位置与约束。
- **环境变量**：通道识别只检查已知 Agent 变量是否存在且非空，取到的值随即丢弃，从不保存、记录或输出（architecture §5.1）。
- **对仓库的写入**：无（architecture §10.6）。
- **私有临时区**：按 WP2.3 §14.3（方案 B，review-resolution §71 ~ §74）在数据目录的 `tmp/git-reader-v3/` 中建立，并沿用它的生命周期：
  - 满足可信清理条件时，按创建清单清理：owner 存在、身份与记录一致，清单与对象一致，Git 消费者已确认结束，完成记录已经持久化；
  - 否则按既有要求保留并报告（退出码 6）：例如身份不符、未知、额外或缺失的条目，消费者状态未确认，完成记录写入或 Sync 失败，调用方崩溃。后续的 Scratch 按实际记录认领，或继续保留；
  - 保留的私有副本（索引、白名单配置等）不是项目事实，不进入日志或备份；协议所需的最小锁与标识长期保留，不含项目内容。
  - 本工作包不改变方案 B，也不扩大其中已接受的边界。
- **日志**：不记录 C0 / C1 内容。

## 11. 与推导的关系（不改变）

- 推导读取批次与 Claim 的 `seq`、`kind`、`audit_scope_json`、`criterion_id`、`criterion_revision`、`assertion`（data-model §6.13）。进度取 `seq` 最大、且修订为当前 revision 的 Claim（§6.5）；修订之后，针对旧 revision 的 Claim 不再参与推导（§7.2）。
- Claim 不能单独产生 `accepted` 或 `ready`；自动化 AC 不会因 Claim 越过 `in_progress`；手动 AC 的 `done` 只推出 `ready_for_review`（`manual_claimed`，不是验证）（FR-CLM-02，§6.3 第 9 行）。
- `inventory_coverage`：`active` AC 中，当前 revision 至少有一条 `audit` Claim 的比例（§6.11）；局部 audit 不算完成盘点（FR-INV-07，CX-10）。
- `target_type = none` 的未映射工作不参与 AC 的推导。
- 本工作包只写入这些事实；推导引擎与复核器不改。

## 12. 验收映射

| 验收项 | WP2.4 验证的部分 | 由其他工作包完成的部分 |
|---|---|---|
| MVP-CLM-01 | 写入 `done` 的 Claim；用写入的事实构造推导输入，得到进度 `claimed`、交付状态 `in_progress`（`claimed_without_evidence`）、`metrics.claimed_without_evidence` 加 1（纯函数层，与 WP2.1 的方式相同） | `status.v1` 的输出（WP2.7） |
| MVP-CLM-02 | 目标只在待审提案中、已退役、不存在 → 退出码 2，不写入 | — |
| MVP-CLM-03 | 摘要超过 500 字符、含代码围栏、含空行、含模拟密钥 → 退出码 2，不写入；Claim、未映射工作与批次摘要都覆盖 | — |
| MVP-CLM-04 | 批次记录 `worktree_id`、`head`、`tree_state`、`base_version`，回执的 `context` 与之一致；工作区状态读取的环境错误 → 退出码 6，不写入、不分配 `seq`（D1） | 会话关联（M3） |
| MVP-CLM-05 | 非 TTY 提交 `manual` → 退出码 3；TTY 中但有已知 Agent 变量 → 退出码 3；`--file -` 管道 → 退出码 3；都不写入、不打开数据库（进程级用例） | M3 的拦截 |
| MVP-INV-01 | `audit` 没有 `audit_scope` → 退出码 2；`sync` 中用 `unknown` → 退出码 2；范围中有不存在或已退役的需求（D2）、`audit` Claim 的目标不属于声明的需求（D3）→ 退出码 2 | — |
| MVP-INV-02 | 不存在的文件、未知的提交 → 照常写入，`exists = false` / `resolved = false`，回执与 `refs_check_json` 一致 | 状态中的展示（WP2.7） |
| MVP-INV-04 | 全量 `audit` 的事实交给推导，`inventory_coverage = 1`、`init_progress = inventoried`（纯函数层） | `status.v1`（WP2.7） |
| MVP-INV-06（P1） | `unmapped_work` 保存为 `target_type = none` 的 Claim，回执逐项给出 | — |
| MVP-WT-03 | Claim 批次有 `worktree_id` 与完整的代码上下文 | 检查执行（WP2.5）、会话与事件（M3） |
| CX-12 (a)(e) | `done` 的事实交给推导：(a) `in_progress`；(e) 手动 AC 为 `ready_for_review`（纯函数层） | 其余分支依赖执行证据（WP2.5） |
| CX-13 | `target_revision` 过期 → 退出码 5，不写入 | — |
| CX-10 | 局部 `audit` 的事实交给推导，`inventory_coverage < 1`（纯函数层） | `INVENTORY_INCOMPLETE` 的输出（WP2.7） |
| 不变量 3、6、7（acceptance-criteria §3） | 没有仅由 Claim 推出的 `ready`；输入与回执都通过 schema；`seq` 严格递增 | — |

## 13. 正反例

**正例**：

- `sync` 批次带文件、测试与提交引用，引用都存在；回执的 `exists`、`resolved` 为 true。
- `sync` 批次，上一批次干净 → `compare_from = previous_batch`；上一批次不干净或没有批次 → `none`。
- `audit` 批次，`audit_scope = {"all": true}`，含 `unknown` 断言；回执没有 `compare_from`。
- 空的 `claims`（`sync`、`audit` 都可以）；只有未映射工作的批次。
- `audit` 批次声明 `REQ-001`、`REQ-002`，只对 `REQ-001` 下的一个 AC 提交 Claim：照常写入（D3 不要求覆盖范围内的全部 AC）。
- `gitx.Clean` 正常返回 `undecidable`：`tree_state = unknown`，照常写入（D1 不适用）。
- 人在终端中提交 `manual` 批次并确认。
- 非 git 项目：`head = null`、`tree_state = unknown`，提交引用 `resolved = false`。

**反例**（都不写入、不分配 `seq`）：

| 输入或状态 | 期望 |
|---|---|
| 不合法的 JSON、未知字段、`kind` 不在枚举中 | 退出码 2 |
| `audit` 没有 `audit_scope`；`sync` / `manual` 中有 `unknown`；非 `audit` 带 `audit_scope` | 退出码 2 |
| 同一 AC 出现两次 | 退出码 2，指向第二条 |
| 摘要 501 字符、含代码围栏、含空行、含模拟密钥（任一摘要、测试名或路径） | 退出码 2，输出中没有该值 |
| 路径越出仓库根、行号倒置或超出上限 | 退出码 2 |
| 目标不存在、已退役、只在待审提案中 | 退出码 2 |
| `audit_scope.requirements` 中有不存在或已退役的需求（D2） | 退出码 2，指向 `/audit_scope/requirements/<j>` |
| `audit` 批次声明 `REQ-001`，其中一条 Claim 指向 `REQ-002` 下的 AC（D3） | 退出码 2，指向 `/claims/<i>/target` |
| 读取之后、写锁之前，声明的需求或目标 AC 被退役（D2、V-C03） | 退出码 2（在写锁下判断） |
| `gitx.Clean` 返回实际的环境错误（D1） | 退出码 6，不写入本次的批次、Claim 与 worktree 登记，`seq` 计数器不变 |
| `base_version` 过期；任一 `target_revision` 过期（CX-13） | 退出码 5，`error.current` 给出当前值 |
| `agent_cli` 提交 `manual`（非 TTY、Agent 变量、管道） | 退出码 3，不打开数据库 |
| `manual` 确认取消、内容不一致、输入结束；带 `--json` 时 stderr 不是终端 | 退出码 3；6 |
| 确认期间基准被修订 | 退出码 5（确认之后的写事务） |
| 含引用的批次，git 会按需抓取对象（PD-26），或部分克隆 | 退出码 6 |
| 提交前缀存在但对象无法读取 | 退出码 6 |
| 未登记的项目 | 退出码 4 |
| 回执校验失败（测试中注入） | 退出码 1，事务回滚，没有任何新行 |

**标记而不拒绝**：不存在的文件、未提交的文件、行号超出文件、未知或有歧义的提交前缀 → 照常写入，`exists = false` / `resolved = false`。

## 14. 已决定的事项与待审的工程细化

**已由负责人决定**（2026-10-20，review-resolution §77）：

| ID | 问题 | 决定 | 理由与代价 |
|---|---|---|---|
| D1 | 工作区状态无法读取 | `gitx.Clean` 返回实际的环境错误时退出码 6，不写入本次的业务事实，不分配本次写入的 `seq`；正常返回的 `undecidable` → `unknown` 与非 git 项目的 `unknown` 不变 | 见 §6.2；代价是这类环境中 `claim` 不可用；不扩大正常打开与方案 B 的保证 |
| D2 | `audit_scope.requirements` 中的编号不存在或已退役 | 退出码 2 | 声明的范围必须指向当前的需求，与 V-C03 对 AC 编号的要求一致；代价是编号写错的批次被整体拒绝 |
| D3 | `audit` Claim 的目标 AC 不属于声明的需求 | 退出码 2，在写锁下按当前的实体与所属关系判断；不要求覆盖范围内的全部 AC，空 `claims` 仍合法 | 声明的范围与实际写入的 `audit` Claim 保持一致，局部 audit 不能计入范围外的覆盖率；代价是跨需求的盘点要么一并声明，要么分批提交 |

以下是决定之前的选项记录（保留，供追溯）：

| ID | 问题 | 当时的推荐 | 当时的备选 |
|---|---|---|---|
| D1 | 工作区状态无法读取（`gitx.Clean` 返回环境错误，例如 git 不满足 PD-26、部分克隆、私有临时区无法建立）时 | 退出码 6，不写入：`tree_state` 是批次必需的代码上下文，WP2.3 也规定读取失败不报告 `clean`；后果是这类环境中所有 `claim` 都不可用 | 记 `tree_state = unknown` 并写入（`unknown` 不会被当作 `clean`，`compare_from` 也不会取这样的批次）；代价是批次可能在不安全的 git 下提交 |
| D2 | `audit_scope.requirements` 中的编号不存在或已退役时 | 退出码 2（与 V-C03 对 AC 编号的要求一致） | 接受，推导只按存在的需求计算 |
| D3 | `audit` 批次中，Claim 的目标不属于 `audit_scope` 声明的需求时 | 退出码 2：声明的范围应与实际盘点的条目一致；否则局部 audit 可以计入范围外的覆盖率 | 接受，覆盖率按实际写入的 `audit` Claim 计算 |

**待审的工程细化**（在已批准的 Q6 / Q9 与既有规则之内，由审查确认；**不是本轮已获批的内容**）：

- E1：`manual` 的确认内容为 `v<base_version>`（§4 第 4 步）。
- E2：`kind = manual` 只能来自 `human_cli` 的数据库约束（§4 第 6 步）。
- E3：`compare_from` 只看最新的那个批次，不向更早回溯（§6.3）。
- E4：有歧义的提交前缀记为 `resolved = false`（§7）。
- E5：`propose` 在提交之后才校验回执；本文不改变它，是否统一由后续决定（§9.3）。
- E6：回执中 `refs.tests` 只给数目，不回显测试名；`audit_scope` 不在回执中（输入中已有）。

**其中可能含有的产品取舍**（审查时请负责人确认，本文不自行决定）：

- E4：有歧义的提交前缀，是标记为 `resolved = false`，还是作为输入错误拒绝；回执是否需要把"有歧义"与"不存在"区分开。两者都是 Agent 可见的语义。
- E6：回执的内容范围：测试名只计数而不回显，`audit_scope` 不回显。M2-Q9 批准的字段清单没有直接规定这两项。
- E1、E2、E3、E5 是在既有机制与规则之内的工程细化：E3 是对 architecture §6.3"上一个批次"的读法；E5 不改变 `propose`。

## 15. 测试计划

**单元与库层**（不需要 CLI 进程）：

- 输入校验：`claims.v1` 的合法示例与全部 schema 反例；数据库之外的规则（重复 AC、围栏、路径、行号）。
- 回执：`claim-receipt.v1` 的合法示例与反例（`schemas/examples`，契约测试）；构造函数从写入的行构造回执，与 §9.2 的业务条件逐项比较。
- `compare_from` 的选择（§6.3）：没有批次、上一批次 `dirty` / `unknown` / `head = null`、上一批次是 `audit`、另一个 worktree 的批次、`manual` 批次。
- 提交前缀解析：存在、不存在、不是提交、有歧义、对象缺失（PD-26，退出码 6）。

**集成**（真实的 git 仓库与数据库，进程内执行 CLI）：

- §13 的每个反例：退出码，以及**数据库中没有任何新行、`seq` 计数器不变**（比较全部事实表，不只比较批次表）。
- D1：注入 `gitx.Clean` 的环境错误（不安全的 git、部分克隆、私有临时区失败），退出码 6，没有新的批次、Claim 与 worktree 记录，`seq` 不变；正常的 `undecidable` 与非 git 项目照常写入 `unknown`。不断言数据目录或私有临时区完全不变：只按方案 B 的契约断言保留或清理。
- D2、D3：在写锁之下判断——两个连接，一个在另一个读取之后、写锁之前退役声明的需求或目标 AC，提交方得到退出码 2。AC 所属的需求不能经更新改变（`proposal.v1` 的 `updateCriterion` 不允许 `requirement`），所以 D3 的所属关系只随 AC 本身的增加或退役变化。D3 的正例：声明两个需求、只覆盖其中一个 AC；`{"all": true}`；空 `claims`。
- 写入成功：批次与 Claim 的行、共用的 `seq`、worktree 的首次登记（另一个 `seq`）、`refs_json` 与 `refs_check_json`；回执与这些行逐项一致（§9.2）。
- 回执校验失败的注入（测试钩子把回执改为不合法）：退出码 1，事务回滚。
- 推导：用写入的事实构造推导输入，验证 CLM-01、INV-04、CX-10、CX-12 (a)(e) 的结论（§12）；推导引擎与复核器不改。
- 并发：两个进程同时提交，`seq` 唯一且连续，各自全有或全无。
- 隐私：摘要、测试名中的 canary 不出现在 stdout、stderr、日志与数据目录的非数据库文件中；密钥一律拒绝。

**真实命令（进程级）**：

- 构建出的二进制或辅助进程在伪终端中执行：
  - `manual`：有终端且没有 Agent 变量 → 显示确认；正确输入 → 写入；取消 → 退出码 3、不写入；
  - 设置 `CLAUDECODE` 或 `CLAUDE_CODE_ENTRYPOINT` → 退出码 3，不打开数据库；
  - stdin 为管道（`--file -`）、stdout 重定向 → 退出码 3；
  - 带 `--json` 且 stderr 重定向 → 退出码 6。
- 真实的 unsafe git（`SPECMETRY_TEST_GIT_OBJECTS=unsafe`，与 WP2.3 相同的开关）：含引用的批次退出码 6。

**平台**：三个平台的 CI；伪终端用例按现有基座在支持的平台运行，跳过的平台如实记录。

## 16. 同步的文档

本 PR 同步：data-model §8（新增 `claim-receipt.v1`）、prd §6（面向 Agent 的输出）、architecture §9.1（输出列表）与 TD-13 的状态、AGENTS.md（必须通过 schema 校验的面向 Agent 的输出）、acceptance-criteria 的 MVP-CLM-04 与 MVP-INV-02（成功输出按 `claim-receipt.v1` 校验，待审查）。实现时再同步：迁移、`internal` 的实现与测试、development.md。
