# Specmetry 架构设计

> 状态：Draft v0.3 · 2026-10-13（依据 [codex-review-001](reviews/codex-review-001.md) 与 [codex-review-002](reviews/codex-review-002.md) 修订）
> 实体与推导规则见 [data-model.md](data-model.md)，交换契约见 [`schemas/`](../schemas/)，范围见 [mvp.md](mvp.md)，实施顺序见 [implementation-plan.md](implementation-plan.md)。本文中的决策均为"提议"。

## 1. 架构目标与约束

| 目标 / 约束 | 来源 |
|---|---|
| 不接管 Agent：只通过 Skill（Agent 主动调用）和 Hook（被动观察）接入 | 原则 1 |
| 三层事实严格区分，并由不同通道产生 | 原则 2 |
| **交付判定只针对确定的代码快照**，并与基准版本和人工确认的范围绑定 | 审查 #2、#4 |
| 项目身份共享；执行、证据和代码上下文按 worktree 隔离 | 审查 #3 |
| 先后关系由数据库序号决定；关键写入携带读取时的版本 | 审查 #6 |
| 数据默认只在本地，并遵循统一的隐私与保留契约（§10） | 原则 6、审查 #1 |
| Hook 有明确的延迟预算，出错不影响 Agent | 原则 4、审查 #10 |
| 安装简单、跨平台 | NFR-05、NFR-06 |

## 2. 系统上下文

```mermaid
flowchart LR
    Dev([开发者])
    subgraph AgentRT["Agent 运行时（Specmetry 不接管）"]
        Agent[Claude Code 等 Agent]
        Skill["Specmetry Skill<br/>init / status / sync / goals / audit"]
        HookCfg[Hooks 配置]
    end
    subgraph Bin["specmetry 单一二进制"]
        CLI[CLI]
        HookH[Hook 入口]
        Ingest[Collector / Ingest]
        Core["Core<br/>基准 · 范围 · 事实"]
        Engine["推导引擎（纯函数）<br/>进度 · 交付状态 · 差距"]
        Runner[Check Runner]
        GitR[Git Reader]
        API[本地 HTTP API + 内嵌前端]
    end
    Spool[("spool/")]
    DB[("SQLite")]
    subgraph Repo["项目仓库"]
        WT1[主 worktree]
        WT2[其他 worktree]
    end

    Dev -- 对话 --> Agent
    Agent --> Skill
    Skill -- "非交互 CLI + JSON（携带读取版本）" --> CLI
    HookCfg -- stdin --> HookH
    HookH -- "白名单字段，原子写入" --> Spool
    Spool --> Ingest --> Core
    CLI --> Core
    API --> Core
    Core --> DB
    Core --> Engine
    Core --> Runner -- "在调用者所在的 worktree 中执行" --> Repo
    Core --> GitR -- 只读 --> Repo
    Agent -- 读写代码 --> Repo
    Dev -- "审核 / 范围 / 验收（浏览器）" --> API
    Dev -- 交互式终端 --> CLI
```

## 3. 关键设计决策（提议）

| ID | 决策 | 理由 | 代价 |
|---|---|---|---|
| AD-01 | **核心不内置 LLM**；语义工作由用户的 Agent 通过 Skill 完成 | 无需 API Key、账号和网络；Specmetry 本体可确定、可测试 | 提取和盘点的质量取决于用户的 Agent |
| AD-02 | **单一 Go 二进制** | 安装简单、Hook 冷启动快、跨平台 | 前端构建需要 Node |
| AD-03 | **没有必需的常驻进程**：Hook 写 spool，任何 `specmetry` 进程负责入库 | Hook 不依赖守护进程；至少投递一次，入库幂等 | Dashboard 未运行时，事件延迟入库 |
| AD-04 | **在 Hook 进程内、落盘前完成字段白名单过滤**；白名单之外的字段一律不落盘 | 敏感原文永远不会落盘 | 白名单要跟随 Agent 的 Hook 格式维护 |
| AD-05 | **事实只追加，状态由纯函数推导** | 可复现、可测试 | 每次读取都要计算 |
| AD-06 | **所有基准变更统一走"提案 → 审核 → 版本"** | 单一路径，历史完整 | — |
| AD-07 | **只有 Specmetry 自己执行的、经人确认的检查，才产生验证性证据** | 区分自述和证据 | 没有检查的自动化 AC 无法验证 |
| AD-08 | **人类专属操作依靠通道识别和拦截保护**，属于"防误操作"，**不能证明操作者身份** | 同一个 OS 用户下，本地无法做到强隔离 | 恶意 Agent 可以绕过，但会留下痕迹 |
| AD-09 | **全局单一 SQLite 数据库**，仓库内只放 marker | 支持多项目视图；数据不进入客户仓库 | 迁移项目需要导出 |
| AD-10 | **Dashboard 只监听回环地址**，使用令牌，校验 Host 和 Origin，设置 CSP | 防御 DNS rebinding 和 CSRF | — |
| AD-11 | **Agent 适配器隔离**；Claude Code 是第一个适配器 | 不绑定单一 Agent | 其他 Agent 降级使用 |
| AD-12 | **交付判定绑定 Delivery Target 的快照，证据分级**：观测级执行只证明开始和结束两个时刻，worktree 与快照一致；隔离级执行在私有快照目录中、由外部强制只读的环境中执行。验收是否适用于当前快照，由推导决定；沿用只接受隔离级证据。 | 杜绝"旧代码上的验收证明新代码"，并且不对观测级证据作出它无法支持的保证 | 每次目标前进都需要重新运行检查；在隔离执行器上线前（MVP），自动化 AC 也需要人重新验收 |
| AD-13 | **全局 `seq` 决定先后**；关键写入携带读取版本，不一致时拒绝 | 消除同毫秒、时钟回拨和读后写竞争带来的歧义 | 调用方要处理退出码 5 / HTTP 409 并重新读取 |
| AD-14 | **项目身份共享，执行上下文按 worktree 隔离**：检查在调用者所在的 worktree 中执行，所有事实都记录 worktree 和代码上下文 | 不同 worktree 有各自的 HEAD 和 index | 数据模型多一个维度 |
| AD-15 | **不持久化检查输出**：输出只实时透传给调用者的终端，只保存结果码、时长和上下文 | 与隐私契约一致，同时保留可调试性 | 事后无法从 Specmetry 中查看失败日志，需要重新运行 |

## 4. 组件

| 组件 | 职责 | 不负责 |
|---|---|---|
| **CLI** | 解析命令；识别通道；输出人类可读格式或 `--json`（符合 `schemas/`）；在命令开始时机会性地入库 spool；解析 cwd 所在的 worktree 并登记 | 业务规则 |
| **Hook 入口** | 读取 stdin（有大小上限）；不访问数据库、不启动子进程；解析项目、worktree 和会话起点 HEAD；只把白名单字段原子写入 spool；PreToolUse 拦截；遵守延迟预算 | 入库、推导 |
| **Collector / Ingest** | 按顺序入库 spool（分配 `seq`，按事件 ID 去重）；更新会话投影；汇总 drop 计数；清理过期事件 | 解析 Agent 原始格式 |
| **Agent 适配器** | Claude Code：Go 包 `claudecode`，CLI 参数 `claude-code`，数据中的标识 `claude_code`。负责事件映射、拦截规则、`setup` 和卸载。 | 通用逻辑 |
| **Core：基准与范围** | 来源、引用、提案、审核、`reaffirm`、版本、问题、交付目标、范围确认；在写事务内校验版本前置条件 | 推导 |
| **Core：事实** | Claim 批次、检查执行记录、提交关联、验收决定；校验通道和前置条件 | 推导 |
| **推导引擎** | 纯函数：输入事实、快照和 `as_of_seq`，输出进度、AC 交付状态、需求状态、交付状态、差距和指标 | 任何 I/O |
| **Check Runner** | 固定开始时的上下文；在调用者所在的 worktree 中执行（观测级）；超时并终止进程树；转发输出（`--json` 时转发到 stderr）；检测开始和结束时的上下文；标注证据等级。隔离执行器在 MVP 之后提供。 | 决定执行什么；把观测级说成隔离级 |
| **Git Reader** | 封装 `git` CLI：解析 ref、HEAD、工作区状态、对象 ID、按 ref 增量扫描提交、worktree 列表 | 写 git |
| **HTTP API + 前端** | REST JSON；携带前置条件的写接口；内嵌 SPA；安全中间件；变更通知 | 执行检查、设置状态 |
| **Skill 包** | `SKILL.md`、参考文档和 schemas，由 `setup` 安装 | 直接访问数据 |

## 5. 信任边界与操作通道

| 通道 | 识别方式（启发式） | 可执行的操作 |
|---|---|---|
| `agent_cli` | 默认值：stdin 或 stdout 不是 TTY，**或**存在已知的 Agent 环境变量 | 见 [prd.md §3.3](prd.md#33-权限矩阵) 的"Agent"列 |
| `human_cli` | stdin 和 stdout 都是 TTY，没有已知的 Agent 环境变量，并通过交互式确认 | 人类专属操作 |
| `dashboard` | 有效令牌、Host 在白名单内、写请求的 Origin 是同源 | 审核、人工编辑、范围确认、来源确认、回答问题、验收决定 |
| `hook` | 由 `specmetry hook` 进入 | 只写 spool；PreToolUse 拦截 |
| `system` | Specmetry 内部 | 检查执行记录、提交发现、事件入库 |

**能力边界（如实声明）**：这些识别方式都是启发式的。

- 其他 Agent 如果使用伪终端，可能被识别成 `human_cli`。
- 有效的 Dashboard 令牌也不能证明操作者是人。

因此，"人类专属"的含义是"**防止 Agent 误执行**"，不是身份认证。为此有三层配套措施：

- Skill 明确禁止 Agent 通过 CLI、HTTP、浏览器自动化或直接写文件代做人工操作；
- PreToolUse 拦截命中的命令；
- 所有人工操作都记录通道与 actor，可以审计。

PreToolUse 拦截规则（第二道防线，第一道是通道识别）。它适用于 Bash 和 PowerShell 两种工具，按**参数位置**匹配，而不是按关键词匹配：

1. 用 `;`、`&&`、`||`、`|` 和换行，把命令拆成若干段。heredoc 的正文（从 `<<` 标记行之后，到结束定界符）是数据，不参与匹配。
2. 对每一段按 shell 的引号规则切分出参数，找出 basename 为 `specmetry` 或 `specmetry.exe` 的那个参数。
3. 它之后第一个不以 `-` 开头的参数是子命令；对 `scope`、`target`、`source`、`question`、`backup`、`project` 这几个命令组，再取下一个非选项参数作为二级子命令。
4. 命中以下任一情况就拒绝：
   - 子命令为 `accept`、`reject`、`waive`、`review`、`purge`、`setup`；
   - 二级子命令为 `scope confirm`、`target set`、`source ack`、`source remove`、`question answer`、`question dismiss`、`backup restore`、`project set`；
   - 子命令为 `propose`，且参数中有 `--apply`。
5. 如果参数是 `bash -c` / `sh -c` / `pwsh -Command` 的字符串，就对这个字符串递归执行一次上述规则。

因此 `specmetry dashboard --page review`、`echo "specmetry accept"`、`propose --file -` 的 heredoc 中出现的 accept 等字样，都**不会**被误拦。

拦截无法覆盖的情形有两类，它们依赖通道识别（退出码 3）兜底：

- 通过变量间接调用，例如 `s=specmetry; $s accept`；
- 命令参数看不出、要读输入内容才知道的操作，例如 JSON 中的 `claim` `kind = manual`。

`specmetry dashboard` 不在拦截之列：令牌只通过浏览器 URL 的 fragment 传递，不打印到输出（§10.4）。

### 5.1 CLI 通道识别规则与已知边界（M1 实现）

**规则**（每次执行命令时判断一次）：

1. 同时满足以下两项时为 `human_cli`，否则为 `agent_cli`：
   - stdin 和 stdout 都是终端（Windows 上是控制台）；stderr 不参与判断；
   - 没有设置任何已知的 Agent 环境变量。只检查变量是否存在且非空：取到的值只用于这一次判断，随即丢弃，**从不保存、记录或输出**（环境变量的值属于 C0，§10.1）。
2. 人类专属操作在 `human_cli` 中还要通过交互式确认：在终端中显示操作摘要和前置条件，要求输入确认内容（例如对象编号或项目名）。确认只从终端读取，**不提供** `--yes` 之类的跳过参数。在 `agent_cli` 中执行人类专属操作，返回退出码 3，不写入任何数据。
   - 确认时，在标准输出（`human_cli` 中它是终端）显示操作名称和摘要，然后从标准输入（同样是终端）读取一行。只有读到以回车结束的完整一行、且内容与确认内容完全一致（忽略首尾空白）才继续；内容不一致、直接回车或输入结束，都视为取消，返回退出码 3（`human_required`），不写入任何数据。输入结束时，即使已经读到完整的确认内容（只是没有回车），也视为取消。
   - 操作名称、摘要或确认提示无法完整输出（写入出错，或只写入了一部分）时，不读取输入，返回退出码 6（`environment`），不写入任何数据：人没有看到完整的操作内容，就不能算作确认。读取确认输入出错时，同样返回退出码 6。
   - 在 `agent_cli` 中直接拒绝，不显示提示，也不读取输入。
3. `--json` 和 `--file -` 不改变通道。只读命令在任何通道中都可以执行。用 `--file -` 从管道读入内容时，标准输入不是终端，因此是 `agent_cli`。
4. 通道在每次执行命令时识别一次（`cli.Execute`），交给命令使用；没有识别结果时按 `agent_cli` 处理。实现位于 `internal/channel`：`Detect` 识别通道，`Guard` 是人类专属操作写入之前必须调用的拦截与确认。

**已知的 Agent 环境变量**（内置清单，只随版本增加，不提供配置项）：

| 变量 | Agent | 核实情况 |
|---|---|---|
| `CLAUDECODE` | Claude Code | 已核实（2026-10-14：Claude Code 桌面版执行命令的子进程中存在） |
| `CLAUDE_CODE_ENTRYPOINT` | Claude Code | 已核实（同上） |

其他 Agent 的变量在核实之后才加入（M4 开工前，见 [implementation-plan.md §3](implementation-plan.md#3-设计细化截止点)"其他 Agent 的约束"）。在此之前，对这些 Agent 只能依靠 TTY 判断、Skill 禁令和审计记录。

**已知边界**（如实声明；通道识别不是身份认证）：

| 情形 | 识别结果 | 说明 |
|---|---|---|
| Agent 通过伪终端执行命令，并清除了上述环境变量 | 可能被识别为 `human_cli` | 交互式确认、Skill 禁令、PreToolUse 拦截和审计记录只是补充措施，仍然不能证明操作者是人 |
| 在 Agent 会话里打开的终端（继承了 Agent 环境变量） | `agent_cli`，人类专属操作返回退出码 3 | 偏向安全的一侧；换一个独立的终端执行即可 |
| 人通过管道或重定向执行（如 `specmetry review \| tee log`），或在 CI 中执行 | `agent_cli` | 偏向安全的一侧 |
| Windows 上不提供控制台的终端（如不经 winpty 的 mintty） | `agent_cli` | 请使用 Windows Terminal、PowerShell 或 cmd |
| IDE 的内置终端 | 一般为 `human_cli` | IDE 中 Agent 的工具调用通常不是终端 |
| `ssh -t` 远程会话 | `human_cli` | 不带 `-t` 时不是终端，为 `agent_cli` |

## 6. 关键流程

### 6.1 项目初始化（`in_development`）

```mermaid
sequenceDiagram
    actor Dev as 开发者
    participant Ag as Agent + Skill
    participant CLI as specmetry CLI
    participant UI as Dashboard

    Dev->>Ag: /specmetry init
    Ag->>CLI: specmetry init --json
    CLI-->>Ag: stage、候选文档、默认交付目标（主 worktree 的分支）、init_progress
    Ag->>Dev: 请确认来源文档
    Dev-->>Ag: 确认
    Ag->>CLI: specmetry source add ...
    Ag->>CLI: specmetry status --json（取得 base_version = 0）
    Ag->>Ag: 阅读来源与代码，生成提案
    Ag->>CLI: specmetry propose --file -（携带 base_version）
    Ag->>CLI: specmetry dashboard --page review
    Dev->>UI: 审核条目、回答问题，并在同一次提交中确认交付范围
    UI-->>Dev: 基准 v1 + 范围确认
    Dev->>Ag: /specmetry init（继续）
    Ag->>CLI: specmetry status --json（取得各 AC 的 revision）
    Ag->>Ag: 逐条对照代码盘点
    Ag->>CLI: specmetry claim --file -（kind=audit，audit_scope=all，携带 revision）
    Ag->>CLI: specmetry check run --all（在当前 worktree 中执行）
    Ag->>CLI: specmetry gaps --json
    Ag-->>Dev: 交付状态、盘点覆盖率、差距
```

### 6.1.1 阶段识别与候选来源发现（M1 实现）

两者都**只读取路径和文件大小，不读取文件内容**；结果是确定的：同一组文件，结果相同。这是启发式判断，只决定 Skill 走哪条初始化路径（[prd.md §4.1.1](prd.md#411-初始化流程)），不参与交付推导；判断不准时，人用 `--stage` 覆盖。

**扫描范围**：主 worktree。git 项目使用 `git ls-files --cached --others --exclude-standard`（已跟踪的文件，加上没有被忽略的未跟踪文件）；非 git 项目遍历目录。两者都排除：名称以 `.` 开头的目录（包括 `.git`、`.specmetry`）、`node_modules`、`vendor`、`third_party`、`dist`、`build`、`out`、`target`、`venv`、`__pycache__`；不跟随符号链接。

**候选来源**（`init.v1` 的 `candidate_sources`）：

1. 文件类型与来源登记的白名单相同（[data-model.md §4.3](data-model.md#43-sources-与-source_acks)）；排除空文件和超过 20 MiB 的文件。
2. 排除不属于需求的常见文件（不区分大小写）：`CHANGELOG*`、`LICENSE*`、`COPYING*`、`NOTICE*`、`CODE_OF_CONDUCT*`、`SECURITY*`、`CONTRIBUTING*`。
3. 按以下顺序排列，同一级内按路径排序：
   1. 文件名或所在目录名含有需求类关键词：英文按完整单词匹配 `brief`、`requirement(s)`、`req`、`spec(s)`、`prd`、`srs`、`scope`、`proposal`；中文按子串匹配 `需求`、`规格`、`要求`、`说明书`；
   2. 位于 `docs/`、`doc/`、`requirements/`、`spec/`、`specs/`、`design/` 之下；
   3. 根目录的 `README*`；
   4. 其他。
4. 最多返回 200 个，按上述顺序截断。
5. 候选来源**从不自动登记**；人确认之后，由 Agent 执行 `source add`。

**阶段识别**（`stage_detected`；用 `--stage` 覆盖后，`stage` 与 `stage_detected` 不同）：

- **代码文件**：扩展名属于常见源代码类型（`.go`、`.js`、`.jsx`、`.ts`、`.tsx`、`.mjs`、`.cjs`、`.py`、`.rb`、`.java`、`.kt`、`.scala`、`.rs`、`.c`、`.h`、`.cc`、`.cpp`、`.hpp`、`.cs`、`.swift`、`.m`、`.php`、`.dart`、`.lua`、`.ex`、`.exs`、`.erl`、`.hs`、`.clj`、`.vue`、`.svelte`、`.sql`、`.sh`、`.ps1`）的文件，或者项目清单文件（`go.mod`、`package.json`、`pyproject.toml`、`setup.py`、`requirements.txt`、`Cargo.toml`、`pom.xml`、`build.gradle`、`build.gradle.kts`、`*.csproj`、`*.sln`、`Gemfile`、`composer.json`、`mix.exs`、`pubspec.yaml`、`Package.swift`、`CMakeLists.txt`）。
- **实质文档**：候选来源中，除根目录 `README*` 之外、不小于 512 字节的文件。

| 条件（自上而下，命中即停） | 阶段 |
|---|---|
| 没有代码文件 | `docs_only` |
| 有代码文件，并且至少有 1 份实质文档 | `in_development` |
| 有代码文件，但没有实质文档（例如只有 README） | `incomplete_docs` |

**实现细化（WP1.1）**：

- 关键词匹配不区分大小写。英文的"完整单词"指连续的 ASCII 字母：数字、`-`、`_`、`.` 等字符，以及小写字母后接大写字母的位置（`ClientBrief`），都是单词边界。文件名和路径上每一级目录的名称都参与匹配（`old/specs/x.md` 属于第 1 级）。
- 第 2 级的目录指仓库根目录下的第一级目录，名称比较不区分大小写（`Docs/` 同样计入）。同一级内按路径的字节序排列。
- 项目清单文件（阶段识别中所列，例如 `requirements.txt`）不作为候选来源：它们属于代码，不是需求文档。
- 排除目录的名称区分大小写；文件类型白名单和代码扩展名按小写比较；代码文件和清单文件在任何目录层级都计入（排除目录中的除外），`*.csproj`、`*.sln` 按后缀匹配。
- 无法作为 `relPath` 输出的路径（含换行等字符，或超过 1024 个字符）不列为候选。
- 实质文档在全部候选中判断，不受"最多 200 个"的截断影响。
- 测试命令提示（P1，`test_hints`）只看根目录的 `go.mod`、`package.json`、`pytest.ini`、`Cargo.toml`，同样只看文件名。

### 6.1.2 `init` 的登记规则（M1 实现）

`init` 可以在任何通道执行，可以重复执行。

**项目的位置与身份**：

- **git 项目**：从当前目录向上查找 `.git`（与 Hook 的查找方式相同，§12.2），再用 `git rev-parse` 取得当前 worktree、它的 git 目录和公共目录，用 `git worktree list` 取得主 worktree。项目由**公共目录**识别（`projects.git_common_dir`，唯一），所以同一仓库的所有 worktree 都属于同一个项目（FR-WT-01）。
- 当前目录位于 git 目录内部或裸仓库中时，没有可以登记的 worktree，返回退出码 6；不会把它当作非 git 项目。
- **非 git 项目**：项目目录是从当前目录向上最近的、含有 marker 的目录，没有时就是当前目录；项目由这个目录（主 worktree 的路径）识别。返回 `not_a_git_repository` 警告，`target = null`（FR-INIT-09）。
- 所有路径都先规范化，再保存和比较：绝对路径，解析符号链接；Windows 上还统一短文件名和大小写。经由符号链接或不同大小写的路径进入同一目录，得到的是同一个项目和同一个 worktree。
- **git 的执行环境**（codex-review-015 B1）：`init` 只运行读取类 git 命令（`rev-parse`、`worktree list`、`symbolic-ref`、`ls-files`；探测 git 能力时，还在临时目录中运行 `cat-file` 和 `ls-files`，见下文"不按需抓取对象"）。这些命令不应执行其他程序、写入文件或修改提交、引用、索引和配置；为此：
  - 不传递任何继承的 `GIT_*` 变量（不区分大小写），包括配置注入（`GIT_CONFIG_COUNT` / `KEY` / `VALUE`、`GIT_CONFIG_PARAMETERS`、`GIT_CONFIG_GLOBAL`）、跟踪（`GIT_TRACE*`）、仓库定位（`GIT_DIR`、`GIT_WORK_TREE`、`GIT_CEILING_DIRECTORIES` 等）和 `GIT_EXEC_PATH`。仓库只由目录决定，与 Hook 一致。
  - 只重新设置受控的变量：`GIT_CONFIG_NOSYSTEM=1`、`GIT_CONFIG_SYSTEM` 与 `GIT_CONFIG_GLOBAL` 指向空文件（不读取系统和用户的全局配置，也不修改它们；`trace2.*` 等只在这两处生效的设置因此不起作用）、`GIT_TRACE2=0`、`GIT_TRACE2_EVENT=0`、`GIT_TRACE2_PERF=0`、`GIT_OPTIONAL_LOCKS=0`（读取不写索引）、`GIT_TERMINAL_PROMPT=0`，以及禁止按需抓取对象的 `GIT_NO_LAZY_FETCH=1` 和 `GIT_ALLOW_PROTOCOL=`（**设置为空**：允许列表中没有任何协议，file、ssh、git、http、https、ext 和任何 remote helper 都被拒绝，并覆盖仓库中的 `protocol.*` 配置；不能省略这个变量，未设置时按 git 的默认策略允许大多数传输；codex-review-021 B1-2）。
  - 这些变量对 Specmetry 启动的**每一个** git 进程都生效，包括第一个运行的版本探测 `git --version`。探测在版本未知时运行，而低于 2.32 的 git 不认识 `GIT_CONFIG_GLOBAL`，会在打印版本之前就按用户全局配置中的 `trace2.*Target` 开启跟踪，连同 `trace2.envVars` 指定的环境变量值写入文件；三个 `GIT_TRACE2*=0` 的优先级高于这些配置，所以旧版 git 在被拒绝之前也不会写出任何内容（codex-review-016 B1）。已用从官方源码构建的 git 2.31.1 和 2.32.0 验证：在版本探测阶段，用户全局配置中只有 trace2 会产生副作用（`core.fsmonitor`、`core.pager`、`core.hooksPath`、经 `include` 引入的设置都不会）。
  - 仓库自己的配置照常生效（例如 `core.worktree`），只有其中会让读取命令执行程序的 `core.fsmonitor` 在命令行上被关闭（`-c core.fsmonitor=`，空值在各个 git 版本中都表示关闭）。命令还带 `--no-pager` 和 `--no-replace-objects`。这些命令不触发 Hook。
  - **不按需抓取对象**（codex-review-020 B1、codex-review-022；PD-26）：部分克隆（有 promisor remote 的仓库）缺少的对象，git 会在读取时从远端抓取（Lazy Fetch）。这可能发起网络请求、运行远端的 upload-pack，或在失败前改写仓库配置（已用真实的 git 2.32.0 验证，它会记录远端的过滤条件）。规则如下：
    - **可能读取对象的命令，只用经实测不会抓取的 git 运行。** 这类命令包括：解析 ref 指向的提交（`rev-parse --verify`）、`ls-tree`、`cat-file`，以及 `ls-files`。`ls-files` 在稀疏检出中会从索引读取工作区之外的 `.gitignore`；不带 `--sparse` 时还会由树对象展开稀疏索引。只读取 ref、配置或目录布局的命令不受此限制，包括 `rev-parse --show-toplevel` / `--git-path`、`worktree list`、`symbolic-ref` 和 `config`。`internal/gitx` 中所有可能读取对象的命令都经过同一个入口；M2 起新增的 git 操作（例如 `status`）也必须经过它。
    - **能力探测**：每个进程在第一次需要读取对象时探测一次，只在 Specmetry 新建的临时目录中进行，不使用用户仓库，不联网。探测先手工写出一个裸的部分克隆：不含任何对象，promisor remote 指向一个不存在的本地路径。然后让 git 读取一个不存在的对象，共运行两次，用 trace2 的结构化事件（JSON，每行一个事件）记录 git 的行为：
      - **对照**：不设置 `GIT_NO_LAZY_FETCH`。事件中必须出现 git 自己启动的抓取子进程（`child_start` 事件，`argv` 中含 `fetch`）。这证明这个仓库确实会触发抓取，也证明抓取能在事件中观察到。
      - **测试**：设置 `GIT_NO_LAZY_FETCH=1`。只有同时满足以下条件，才判定为"不会抓取"：事件中有 git 自己的 `start` 和 `exit`；`exit` 中的退出状态与进程实际的退出状态一致，并且非零；没有任何子进程。
      - 测试中出现抓取子进程，判定为"会抓取"。其他任何情况都是"无法判定"，按不具备能力处理（fail-closed），例如：没有事件、对照中没有抓取、事件不是 JSON 或属于其他进程、出现其他子进程、退出状态不一致、超时或被取消。不依据版本号，也不依据 git 是否认识 `--no-lazy-fetch` 来推断：Ubuntu 的 git 2.43 不认识这个选项，却遵守这个变量。
      - 两次运行都使用受控环境（包括空的 `GIT_ALLOW_PROTOCOL`），所以对照中的抓取在连接或运行任何程序之前就失败。git 的输出全部丢弃。临时目录在成功、失败、超时（30 秒）和取消后都会删除。探测内容不写入用户仓库、数据库或错误输出。
    - **受控复用**：只保存明确的结论（"会抓取"或"不会抓取"），并且只在 PATH 找到的仍是同一个 git 文件（路径、大小、修改时间都不变）时复用；git 文件变了，就重新检查版本并重新探测。失败、取消、超时和"无法判定"都不保存，下次重新探测。实际运行的就是 PATH 找到的那个文件。
    - **不具备能力时**：所有可能读取对象的操作都在运行之前返回退出码 6，在任何仓库中都是如此。不再先识别仓库是否为部分克隆，因为识别与读取是两个 git 进程，仓库可能在两者之间发生变化（codex-review-022）。不读取对象的操作不受影响，例如项目定位、`source` 命令、不含代码引用的 `propose`。
    - **具备能力时**：git 读取缺失的对象会直接失败，不抓取、不运行程序、不写仓库。缺失的对象一律不会被当作存在。仓库在两次读取之间变成部分克隆也是如此，因为读取进程自己就遵守 `GIT_NO_LAZY_FETCH`。
    - **实测（2026-10-15）**：官方源码构建的 git 2.32.0 判定为"会抓取"。Ubuntu 的 git 2.43.0（`1:2.43.0-1ubuntu7.3`）和 Git for Windows 2.56.0 判定为"不会抓取"。三者的结论都与一个独立判据一致：不设置协议限制，由 promisor remote 的 upload-pack 替身写标记文件，以此判断是否发生了抓取。
    - 已知边界：探测证明的是 PATH 上的 git 文件的行为。在探测与运行之间，有人在原位置替换 git 文件（并保持大小和修改时间不变），不在防护范围之内。
  - **列出文件不展开稀疏索引**（PD-26）：
    - `ls-files` 带 `--sparse` 运行。稀疏检出之外的目录以目录项（末尾带 `/`）列出，不由树对象展开，也就不读取这些树。目录项不是常规文件，不参与候选来源和阶段识别。这与展开后的结果相同，因为稀疏检出之外的文件本来就不在工作区中。
    - 不带 `--sparse` 时，git 2.56 和 2.43 展开稀疏索引都会改写对象文件的修改时间（重新写入已存在的树对象）；树对象缺失时，还会静默漏掉其中的文件。
    - `ls-files` 不认识 `--sparse` 的 git（例如 2.32.0），在解析参数之前就已读取并展开索引，所以不能在用户仓库中试探。是否认识这个选项，同样在临时目录中探测一次，并按同样的方式复用结论。不认识时，列出文件返回退出码 6，所以这类 git 上 Git 项目的 `init` 返回退出码 6（**实施细化，待产品负责人确认**，见 review-resolution §27）。
    - 稀疏检出中，git 从索引读取工作区之外的 `.gitignore`。它的 blob 缺失时，具备能力的 git 不抓取、不报错，按没有这些规则处理（git 2.56、2.43 实测）。这只影响候选来源列表，不影响安全。
  - **部分克隆的识别**（codex-review-021 B1-1）：只用于 §6.1.4 代码引用的提示。在具备能力的 git 上，检查代码引用之前仍先识别部分克隆，是部分克隆就返回退出码 6，提示改用完整克隆。识别方法如下：
    - `extensions.partialClone` 有任何取值，即为部分克隆（它的值是远端名称，不按布尔值解释）；
    - `remote.<name>.promisor` 由 git 自己按布尔值解析（`git config --bool --get-regexp`），git 认为是真值的（包括 `2` 这样的数值），即为部分克隆；
    - git 无法把它解析为布尔值，或无法读取这些配置时，返回退出码 6，不当作完整克隆。
    - 识别只读取配置，不读取对象；错误中不出现配置值。
  - 需要 git 2.32 或更高版本（第一个支持 `GIT_CONFIG_GLOBAL` / `GIT_CONFIG_SYSTEM` 的版本）；更旧的 git 无法隔离用户配置，返回退出码 6。最低版本仍是 2.32（PD-26 不提高它）；读取对象另外需要上述能力。
  - 用户全局排除文件的默认位置（`$XDG_CONFIG_HOME/git/ignore`）仍然生效，因为 `HOME` 照常传递；`core.excludesFile` 指向别处时，那里的规则不生效（候选来源可能多出这些文件，不影响正确性）。
- **git 的错误**（codex-review-015 B3）：git 自己的错误信息（可能引用配置值或路径）一律丢弃，不进入输出、`error.v1` 或日志，也不保存。错误只说明是哪个 git 子命令、退出状态是多少（退出码 6）；找不到仓库等错误也不包含当前目录。

**marker 与数据库**：数据库是项目身份的唯一依据；marker（§8）只是指向它的指针，供不访问数据库的 Hook 使用。

- 数据库中有这个项目，但 marker 不存在：补写 marker。
- marker 中的 `project_id` 与数据库中的项目不同，或者数据库中没有 marker 所指的项目（例如该项目已被 `purge`，或登记在另一个 `SPECMETRY_HOME` 中，或者目录是从其他项目复制或移动来的）：返回退出码 6，**不登记项目、不分配 `seq`，marker 保持原样**，提示人检查。数据目录中还没有数据库时，在创建数据库之前就拒绝，数据目录逐字节不变（codex-review-015 N1）。人确认后删除 marker 再执行 `init`，目录就会登记为新项目。marker 中的 ID 从不被沿用，所以已删除的项目不会因此复活。仓库移动或复制后如何继承原项目，见 TD-25。
- marker 无法解析、不是常规文件，或者 `.specmetry` 不是目录（包括符号链接）：返回退出码 6，不写入。

**首次执行**（数据库中没有这个项目）：在一个写事务中登记项目、主 worktree，以及当前 worktree（如果与主 worktree 不同）；并按 FR-INIT-10 创建默认交付目标：名称为 `default`，ref 是主 worktree 当前所在的分支（即使 `init` 在关联 worktree 中执行），`actor` 和 `channel` 为 `system`。分支还没有提交时也创建目标，快照为 `null`；主 worktree 处于分离 HEAD 时不创建，并返回 `detached_head_no_target`。分支已有提交、但提交对象无法读取（例如部分克隆缺少它），不等于"还没有提交"：返回退出码 6，不写入任何内容（PD-26）。先解析 ref（不读取对象），ref 不存在时快照为 `null`；ref 存在时再读取它指向的提交，读取失败即报错。阶段取 `--stage`，否则取识别结果；名称取 `--name`，否则取主 worktree 的目录名。

**再次执行**：

- 返回 `created = false` 和已有的项目。`stage_detected` 每次重新识别；`stage` 保持已保存的值，只有 `--stage` 会修改它。
- 第一次出现的 worktree 被登记（MVP-WT-01）；已登记的只更新 `last_seen_at`，不分配 `seq`。
- **不创建交付目标**：项目创建之后，交付目标只能由人设置（`target set`）。首次执行时主 worktree 处于分离 HEAD 的项目，之后即使回到分支上也没有目标（差距 `NO_TARGET`）。
- **不改名**：`--name` 与已登记的名称不同时，返回退出码 2，不写入。
- `--skip-sources`、`--skip-inventory` 只会设置 `settings_json` 中对应的标记，不会清除它。

**写入顺序与失败**：

1. 参数、路径和分支的校验都在打开数据库之前完成；校验失败时，连数据目录都不会创建。git 项目在定位仓库之后，先完成对象读取能力的检查（PD-26），然后才列出文件。不具备能力时返回退出码 6，数据目录、项目登记、marker、exclude 和 `seq` 都不变。主 worktree 分支的快照也在打开数据库之前解析。已登记项目的目标 ref 与主 worktree 当前的分支不同时，这个 ref 在写事务中解析，解析失败则整个事务回滚。扫描被取消（例如按下 Ctrl-C）时，`init` 立即停止，返回退出码 6，不会把中断的扫描当作完整结果，也不登记任何内容（codex-review-015 N2）。
2. 数据库写入在一个写事务中完成（`BEGIN IMMEDIATE`），所以并发执行的 `init` 只会登记一个项目；`git_common_dir` 的唯一约束是第二道防线。事务中的任何失败都会回滚，不留下数据。
3. 事务提交之后写 marker（先写临时文件，再原子替换），最后写 exclude 条目（§10.6）。marker 写入失败时返回退出码 6；项目已经登记，再次执行 `init` 会补写 marker。

**密钥扫描**（§10.8）：项目名（`--name` 或目录名）、要保存的路径（主 worktree、当前 worktree、git 目录、公共目录），以及主 worktree 的分支（新项目默认交付目标的 ref；codex-review-015 B2），都在打开数据库之前扫描。分支在项目已经登记、不会新建目标时也扫描，因为打开数据库之前无法知道是否会新建；命中时返回退出码 2，错误中只给出字段和规则，不回显内容。项目名还必须非空、单行、不含控制字符，并且不超过 200 个字符。

**输出**：`--json` 输出 `init.v1`，输出前按 schema 校验；失败时输出 `error.v1`。

- `init_progress` 按 data-model §6.12 推导。来源、提案、基准和范围随 WP1.4 ~ WP1.8 加入；在此之前这些事实都不存在，所以只有 `--skip-sources` 能让进度越过 `initialized`。基准版本表出现之前，`baseline_version` 恒为 0。
- `data_dir_permissions_loose` 警告不会出现：数据目录的权限不满足要求时，写入方直接拒绝（§11.1，退出码 6）。
- P1 的 `git` 字段（提交数、最近提交时间）暂不输出。

### 6.1.3 来源的登记与确认（M1 实现）

命令：`source add <path> [--kind] [--allow-any-type] [--json]`、`source list [--include-removed] [--refs] [--json]`、`source ack <path|ID> --observed <sha256> -m <理由>`（也可以用 `--file` 传入 `source-ack.v1`）、`source remove <path|ID> [--erase-excerpts]`。

所在项目按 §6.1.2 的方式由当前目录确定，只读。数据目录中还没有数据库，或者目录不属于已登记的项目时，返回退出码 4，不创建任何文件。

**路径**：

- 相对路径按当前目录解析；符号链接会被解析，来源就是内容所在的位置。
- 文件位于项目根目录（主 worktree）之内时为 `in_repo`，保存相对路径（`/` 分隔）；否则为 `external`，保存规范化的绝对路径。路径可以含空格和非 ASCII 字符。
- 以下情况返回退出码 2：
  - 文件位于项目的关联 worktree 中：来源总在主 worktree 中观测，应在那里登记。不论命令在主 worktree 还是哪个关联 worktree 中执行，规则都相同。文件属于哪个 worktree 由 git 判断（codex-review-018 B3）：在文件所在的目录用 `internal/gitx`（受控环境）查出所属 worktree 及其 git 公共目录；公共目录就是项目的公共目录、而 worktree 不是主 worktree 时拒绝。目录按文件身份比较，不比较路径写法，也不按路径前缀判断，所以放在主 worktree 目录之内的关联 worktree 同样拒绝。不在任何 git 仓库中的文件、其他独立仓库中的文件不受此限制；
  - git 无法判断文件属于哪个 worktree 时（例如文件在 git 目录或裸仓库中，或者 git 拒绝读取它所在的仓库）：保守拒绝，不当作项目以外的文件；
  - git 目录中的文件；
  - 目录；
  - 不存在的路径。
- 这些错误只给出原始参数（已经过密钥扫描）和固定的说明，不给出解析后的路径：名称无害的符号链接可能指向路径中含密钥的文件（codex-review-018 B1）。解析后的路径在使用之前另行扫描。
- `kind` 默认为 `other`。

**类型白名单**（V-S01）：按扩展名判断，不区分大小写。`--allow-any-type` 只能由人在 `human_cli` 中使用，并且需要交互式确认；Agent 通道使用时返回退出码 3。白名单只约束**新建**来源：已登记、未移除的来源（例如人用 `--allow-any-type` 登记的非文档文件），在任何通道中重复登记，都只返回已有的来源，不写入（MVP-SRC-07）。Agent 仍然不能新建非文档来源（退出码 2）：写事务中会再次确认，避免来源在两次检查之间被移除后，被重新创建。

**重复登记**：

- 同一路径已有未移除的来源时，返回该来源（`created = false`）。不写入、不分配 `seq`，也不更新 `acked_sha256`，即使文件已经变化。
- 数据库约束保证每个项目的每个路径只有一个未移除的来源，所以并发登记同一路径只会产生一个来源。
- 已移除的路径再次登记时，创建新的来源（新 ID）。

**哈希与观测**：

- 文件只为计算 sha256 而读取，每次读 1 MiB，读完即丢弃。每读一块之前检查是否已取消；取消时返回退出码 6，不返回部分结果，也不写入。不设文件大小上限。
- `source list`、`source add`、`source ack` 每次都重新读取磁盘，得到三种状态：`unchanged`（当前哈希等于已确认的哈希）、`changed`、`missing`。
- 文件不存在、不是常规文件或无法读取（例如没有读权限），都报告为 `missing`，`current_sha256 = null`。
- 观测结果不保存，也不会改变已确认的哈希；变化只能由人通过 `source ack` 确认。
- 已移除的来源不再观测，`state` 和 `current_sha256` 都为 null。

**`source ack`（仅限人）**：

1. 先检查通道，`agent_cli` 返回退出码 3。输入按 `source-ack.v1` 校验，理由做密钥扫描。`--file` 指定的文件无法读取时，错误中不出现它的路径（路径尚未经过密钥扫描）。
2. `--observed` 与磁盘上的当前哈希不一致时，不显示确认提示，直接返回退出码 5，不写入；`error.current` 中给出 `source` 和 `current_sha256`。来源没有变化时也是如此：过期的观测哈希总是返回退出码 5。
3. `--observed` 正确、来源也没有变化时，提示无需确认，返回 0，不写入。
4. 交互式确认。要输入的确认内容：仓库内的来源为路径，外部来源为 ID（外部的绝对路径较长，可能需要输入法）。
5. 确认之后**再次**读取磁盘并比较，仍然一致才写入：
   - 写入一条只追加的 `source_acks`，包括 `previous_sha256`、`acked_sha256`、理由、actor 和 `channel = human_cli`；
   - 把来源的 `acked_sha256` 改为观测到的哈希。

   确认期间文件发生变化时，返回退出码 5，不写入。
6. actor 为 `human:<显示名>`：取 `config.toml` 中 `[user]` 的 `display_name`，未设置时取操作系统用户名。显示名同样做密钥扫描。

**`source remove`（仅限人）**：

- 交互式确认之后写入 `removed_seq`。已移除的来源不会再改变，来源行也不会被删除（`purge` 与墓碑除外），两者都由数据库触发器保护。引用保留。
- `--erase-excerpts`：M1 中项目只能是 `hash_only`，没有保存任何片段原文，所以没有内容需要擦除，确认提示中会说明这一点。`full` 模式下的擦除与整理在 WP2.11 实现；在那之前，遇到不是 `hash_only` 的项目时拒绝（退出码 6），不改动任何内容。

**引用**：`source_refs` 由提案写入（§6.1.4）：WP1.5 写入问题的来源引用，审核接受条目时写入条目的来源引用（WP1.6）。`--refs` 列出每个来源的全部引用（当前与历史），包括已移除来源的引用（加上 `--include-removed`）；`hash_only` 下 `excerpt_state = not_stored`、`excerpt` 为 null。MVP-SRC-04 中引用以 `source_removed` 的形式出现在状态与差距中，随 M2 验证。

**来源集合哈希**：按 data-model §4.3 计算，在写事务中从数据库读取未移除的来源。它只随登记、确认和移除改变，观测不影响它。MVP-SRC-06 的含义见 PD-25（已确认）：文件发生变化、但尚未经人工确认时，集合哈希不变。

**输出**：`source add --json` 输出 `source-add.v1`，`source list --json` 输出 `source-list.v1`，输出前都按 schema 校验。

- `state` 与 `current_sha256` 的关系由 schema 约束：`missing` 没有当前哈希，`unchanged` 和 `changed` 有。
- "`unchanged` 当且仅当当前哈希等于已确认的哈希"，以及来源集合哈希与列出的来源一致，schema 无法表达，由输出一致性测试验证。
- `source ack`、`source remove` 是人类专属命令，只输出文本；出错时，带 `--json` 输出 `error.v1`。

**隐私**：来源原文不进入数据库、输出或日志。路径（原始参数和要保存的路径）、理由和显示名，在写入之前做密钥扫描；命中时返回退出码 2，错误中只给出字段和规则。

### 6.1.4 提案的提交（M1 实现）

命令：`propose --file <path|-> [--json]`（`-` 表示从标准输入读取，最大 8 MiB）。任何通道都可以提交；提交**不改变基准**：条目以 `pending` 等待人审核（WP1.6），问题在提交时立即以 `open` 登记。不创建基准版本，不写入 `goals`、`requirements`、`acceptance_criteria`。

**通道与人工编辑**（V-P01）：

- `--apply`（人工编辑、立即应用）只能由人在 `human_cli` 中使用，Agent 通道返回退出码 3。应用机制在 WP1.6 实现；在那之前，人在终端中使用 `--apply` 也返回退出码 2，不写入。
- `origin = human_edit`：Agent 通道返回退出码 3；人在终端中不带 `--apply` 提交时返回退出码 2（指向 `/origin`）。它永远不会被当作普通的待审提案保存。
- actor：Agent 通道为 `agent:<agent>`，设置了 Claude Code 的环境变量时为 `agent:claude_code`，否则为 `agent:unknown`；人在终端中提交时为 `human:<显示名>`（同 §6.1.3）。

**处理顺序**（拒绝为退出码 2，过期为退出码 5；两者都不写入任何数据，也不分配 `seq`）：

1. 读取输入；所在项目按 §6.1.2 确定，没有数据库或项目未登记时返回退出码 4。
2. 按 `proposal.v1` 严格校验（V-P01）；错误按 §10.8 脱敏，只给出 JSON Pointer 和约束。
3. 不需要数据库的规则：
   - 所有字符串值都不能包含代码围栏（V-P05）；
   - `ref` 在提案内唯一；`goal`、`requirement`、`about` 指向本提案内的 `ref` 时，被指向的条目类型必须正确（需求的 `goal` 指向目标，AC 的 `requirement` 指向需求，问题的 `about` 指向需求或 AC）；同一 `target` 至多出现一次，并且编号的类型与条目的 `type` 一致（V-P03）；
   - `workdir`、`inputs` 和代码引用的路径，统一分隔符并按字面规范化之后，不得越出仓库根（V-P04；schema 已拒绝绝对路径和 `..` 段，这里是第二道防线）；
   - 代码引用的行号：可以不填；填写时为 `a` 或 `a-b`，`1 ≤ a ≤ b ≤ 2147483647`（32 位上限，32 位与 64 位构建的解释一致）。超出上限（包括 schema 允许、但无法表示的超长数字）或倒置的范围都拒绝，绝不当作"没有行号"（codex-review-020 B4）；静态检查与代码引用检查使用同一个解析函数。片段去掉空白之后不能为空（否则在任何来源中都能"找到"）。
   - 需求的 `goal` 可以为 `null`（不关联目标；更新时表示清除关联），不作为编号查找，并原样保留在 payload 中（codex-review-020 B3）。
4. 密钥扫描（V-P05，§10.8）：提案中的**每个**字符串值（包括片段、路径和检查命令），以及可能被登记的当前 worktree 路径和 git 目录。
5. 解析来源（V-P06）：`source` 按登记时保存的路径或来源 ID 精确匹配；已移除或未登记的来源都拒绝，错误指向每个这样的引用。
6. 在事务之外读取来源与代码：
   - **片段校验**（V-P07，标记）：每个来源文件只流式读取一次，同时计算 sha256，读完即丢弃。规范化方式：去掉所有 Unicode 空白字符（空格、制表符、换行，包括 CRLF 和全角空格），再做子串匹配，因此只有空白或换行不同的片段为 `verified`，改动任何一个字为 `not_found`。`.pdf`、`.docx`、`.doc`、`.odt`、`.rtf`，以及不是有效 UTF-8 或含有 NUL 字节的文件，为 `not_checkable`；文件无法读取、或不是普通文件时为 `source_missing`：来源文件以不会因 FIFO、设备等特殊文件而阻塞的方式打开（Unix 用 `O_NONBLOCK`，Windows 打开设备或管道本身不等待），再通过已打开的描述符或句柄确认是普通文件，所以检查与打开之间不会被替换；取消在每块读取之前生效，不产生部分结果（codex-review-020 B5）。来源登记与观测的哈希计算（§6.1.3）使用同一种打开方式。其他类型逐字匹配文件原文，不解析 HTML 或邮件编码。`verified` 只说明片段出现在来源中，**不说明**条目语义正确或覆盖完整（data-model §4.5），文本输出中如实说明这一点。
   - **代码引用**（V-P08，标记）：git 项目中，对照**当前 worktree 的 HEAD 提交**检查：路径必须是该提交树中的普通文件（目录、符号链接和子模块都不算），有行号时行号必须在文件的行数之内。未提交的文件、没有任何提交的 worktree，都记为不存在。非 git 项目对照项目根目录下的文件，解析链接之后仍须在根目录之内。只读取行数，不保存任何代码内容。代码引用只说明文件位置存在，**不是**验收证据。git 调用沿用 `internal/gitx` 的受控环境，只接受完整的对象 ID。git 不具备安全读取对象的能力时（§6.1.2 的"不按需抓取对象"，PD-26），含代码引用的提案返回退出码 6，不写入、不分配 `seq`；不含代码引用的提案不受影响。部分克隆中不检查代码引用：读取对象之前识别出部分克隆时返回退出码 6，提示改用完整克隆或不带代码引用提交（codex-review-020 B1）。
7. 写事务（`BEGIN IMMEDIATE`）：
   - `base_version` 不等于当前基准版本（不存在版本时为 0）时返回退出码 5，`error.current` 给出 `base_version`（V-P02）；
   - 编号必须指向未退役、类型正确的条目，否则退出码 2（V-P03）；
   - `target_revision` 不是目标当前的 revision 时返回退出码 5，`error.current.targets` 逐项给出 JSON Pointer、编号和当前 revision（V-P02）；
   - 更新 AC 时，结果不能是"有检查但不是自动化"：设置检查需要 AC 是 `automated`，改为 `manual` 需要同时去掉检查（V-P04）；
   - 引用的来源在此期间被移除时拒绝；
   - 当前 worktree 第一次出现时自动登记（data-model §4.1）；
   - 分配**一个** `seq`，写入提案、全部条目（`pending`）、问题（按 TD-14 分配 `Q-nnn`）和问题的来源引用。任何一步失败，整个事务回滚。

**保存的内容**（`hash_only`，M1 唯一的模式）：

- `proposal_items.payload_json` 为提交的 `data`（`retire`、`reaffirm` 为 null）；提议者对 `update` 建议的修订类别单独保存在 `suggested_change_class`，不会代替审核者的选择（V-R01）。
- `sources_json` 的每一项为 `source_id`、`locator`、`excerpt_sha256`（提交的片段原文的 sha256）、`verification` 和 `source_sha256`（校验时来源文件的哈希，无法读取时为 null），**不含片段原文**；`code_refs_json` 的每一项为 `path`、`lines`、`exists` 和 `checked_head`。数据库触发器拒绝在 `hash_only` 项目中保存片段原文（第二道防线）：`proposal_items` 在插入和更新时检查 `sources_json` 与 `final_payload_json` 中任何层级、取值不为 null 的 `excerpt` 键（用 SQLite 的 `json_tree`），`source_refs` 在插入和更新时检查 `excerpt` 列；这些触发器在维护模式下同样生效，擦除只会去掉原文（codex-review-020 B2）。
- 问题指向已有编号时，立即关联到该条目；指向本提案内的 `ref` 时，记录 `proposal_item_id`，审核接受该条目时再回填（WP1.6，data-model §4.8）。问题的来源引用立即写入 `source_refs`（`entity_type = question`，`entity_revision = 1`，`excerpt` 为 NULL），因此会出现在 `source list --refs` 中；条目的来源引用在审核接受时才写入 `source_refs`（WP1.6）。

**输出**：`propose --json` 输出 `proposal-receipt.v1`，输出前按 schema 校验：待审提案 `applied_version = null`，每个条目有 `item_id`、`review_state = pending`、`assigned_key = null`，来源引用只给出 `source_id` 和校验结果，代码引用给出 `exists`，**不回显片段**。schema 的字段间条件见 MVP-BL-28。文本输出给出各类校验结果的数量和有问题的引用位置，同样不包含片段。

### 6.2 Hook 采集与入库

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant H as specmetry hook
    participant SP as spool/
    participant IG as Ingest
    participant DB as SQLite

    CC->>H: stdin JSON
    H->>H: 读取 stdin（上限 1 MiB；超出则计一次 drop 并退出）
    H->>H: cwd → marker / worktree（只读文件）
    alt 不属于 Specmetry 项目
        H-->>CC: exit 0
    else
        H->>H: 只取白名单字段；SessionStart 时通过读文件解析 HEAD，失败则记为 unknown
        H->>SP: 写临时文件，再原子 rename（超出预算则放弃并计 drop）
        H-->>CC: exit 0
    end
    IG->>SP: 按文件名顺序读取
    IG->>DB: 事务：分配 seq，插入事件（按 ID 去重），更新会话
    IG->>SP: 提交后删除文件
```

### 6.3 `sync`

1. `specmetry status --json --session current`：返回基准版本、各 AC 的 revision，以及 `workspace.compare_from`。比较起点按以下优先级选取：
   1. 本 worktree 中上一个 `sync` 或 `audit` 批次记录的 `head`，前提是那时工作区是 `clean`；
   2. 本会话的 `head_start`，前提是已知且在同一 worktree；
   3. 否则为 `none`。
2. 同一输出给出 `workspace.changed_paths`：从比较起点到工作区的已跟踪改动、暂存改动，以及**未跟踪的文件**。只包含路径，不保存。
3. Agent 判断受影响的 AC，用 `claim --file -` 提交批次，携带 `base_version` 和每条的 `target_revision`。比较起点为 `none` 时，Skill 要求 Agent 只对实际核实过的 AC 作出断言。
4. `check run <AC...>`：在当前 worktree 中执行。只有当前 worktree 的 HEAD 恰好等于交付快照、且工作区干净时，结果才会影响交付判定；否则只更新"工作区视图"。
5. `status --json --diff-since <seq>`：输出状态变化。

### 6.4 检查执行与证据等级

见 [data-model.md §4.11.1](data-model.md#4111-证据等级)（等级定义与保证边界）和 [§7.4](data-model.md#74-检查执行的上下文绑定)（执行步骤）。补充说明：

- **进程管理**：Unix 上使用独立进程组；Windows 上使用 Job Object。超时后终止整个进程树。
- **环境**：继承用户的环境，并注入 `SPECMETRY_CHECK=1`。环境变量的值不记录。
- **执行方式**：多个检查默认串行执行。
- **输出流**：带 `--json` 时，stdout 只有一份 `check-run.v1` JSON，检查自身的输出全部转发到 stderr；任何模式下都不保存检查输出。
- **分阶段实现**：

| 阶段 | 执行器 | 证据等级 | 对交付的影响 |
|---|---|---|---|
| MVP（M2） | 观测级：在调用者的 worktree 中执行，只观测开始和结束两个时刻 | `observed` | 可以让 AC 进入 `ready_for_review`；人验收时必须确认 `observed_evidence` 风险；**不沿用**；交付判定标为 `observed` |
| MVP 之后 | 隔离执行器：把快照物化到私有目录，在由隔离后端强制只读的环境中执行，并在执行前后验证 | `snapshot_isolated` | 可以支撑沿用；交付判定可以标为 `snapshot_isolated` |

- **降级与拒绝**：
  - `--isolation required` 在无法建立并验证隔离时，一律不执行，返回退出码 6；在 MVP 中总是如此。
  - `--isolation auto` 只在隔离执行器上线后提供，失败时降级为观测级，并记录原因。
  - 系统在任何情况下都不会把观测级执行记为隔离级。
- **易用性**：普通项目的初始化和日常使用都**不依赖** Docker 或其他隔离后端；只有需要隔离级证据时，才需要准备隔离后端。

### 6.5 验收

1. Dashboard（或交互式 CLI）读取需求详情，包括：
   - 交付快照 `S`；
   - AC 的 `criterion_revision` 和 `requirement_revision`；
   - `S` 上最新的合格执行 `evidence_id` 及其证据等级；
   - **当前决定 `current_decision_id`**（及其类型和操作者）；
   - 当前允许的决定类型，以及必须确认的风险（按 [data-model.md §7.3](data-model.md#73-人工验收的前置条件)）。
2. 界面展示风险说明，人逐项确认。然后提交 `decision.v1`，携带以上全部读取值、`seen_decision_id` 和 `acknowledged_risks`。
3. 服务端在**同一个写事务**内依次完成：
   1. 重新解析 ref；
   2. 比对全部前置条件，包括"最新决定的 ID 是否仍等于 `seen_decision_id`"；
   3. 校验风险确认是否与服务端推导的结果一致。

   任何前置条件不一致，都返回 409（CLI 返回退出码 5），不写入；`error.current` 中给出最新决定。一致时写入 `decisions`（分配 `seq`）。
4. **不自动重试**：客户端收到 409 后，必须重新读取并展示新的状态（例如"已被其他页面拒收"），只有人再次明确操作，才会提交新的请求（携带新的 `seen_decision_id`）。

**批量验收**（P1）只是界面上的便利：对多个 `ready_for_review` 的 AC，逐条提交 `decision.v1`，每条都带各自的前置条件和风险确认。任何一条返回 409，只影响那一条，其余照常处理，**不会**自动重试。

### 6.6 基准变更、父需求修订与范围重新确认

1. 审核者在审核队列中处理提案条目。对 `material` 的需求更新，界面列出它的所有 `active` AC，要求对每个 AC 选择 `reaffirm` / `retire` / `defer`。需要修订、但本提案中没有对应更新条目的 AC，选择 `defer`，之后再通过人工编辑修订。审核不能新增条目。应用顺序见 [data-model.md §5.5](data-model.md#55-一次审核或人工编辑的应用顺序)。
2. 审核提交（`review.v1`）携带 `seen_base_version`，可以附带新的范围确认（`scope`）。
3. 服务端在一个事务内完成以下步骤：
   1. 校验版本；
   2. 应用变更，生成新的基准版本；
   3. 写入 AC 的 `reaffirm` 或退役；
   4. 用新版本号写入范围确认（如果附带了的话）。
4. 未附带范围确认时，范围变为 `SCOPE_STALE`，直到重新确认。

### 6.7 git 扫描

在 ingest、`sync` 和 `status` 时执行，同一项目至少间隔 30 秒。

- 扫描对象：所有本地分支，以及各 worktree 的分离 HEAD。
- 每个 ref 维护自己的游标；用 `git log --format=%H%x00%ct%x00%B <last_tip>..<tip>` 读取增量。
- 只匹配编号；只保存 SHA、时间和关联关系。
- 正文在内存中处理，不保存。

## 7. 推导引擎

- 位于 `internal/domain/derive`，是不依赖 I/O 的纯函数包。
- 输入：截止 `as_of_seq` 的全部事实（含基准和范围）、目标与 `S`、各来源的观测哈希，以及当前 worktree 的代码上下文（只用于工作区视图）。读取磁盘和 git 的工作由调用方在推导**之前**完成，并作为参数传入。
- 规则以 [data-model.md §6](data-model.md#6-推导) 为准，**每条规则都有表驱动测试，以及对应的反例测试**。
- 推导结果不入库。任何输出都附带 `as_of_seq`、`S` 和 `baseline_version`，因此可以复现。

## 8. 本地文件布局

```text
$SPECMETRY_HOME/                  默认 ~/.specmetry（TD-02）；权限见 §11.1
├── config.toml
├── specmetry.db (+ -wal, -shm)   SQLite（WAL）
├── tombstones.json               已删除项目的标识（只追加，§11.4）
├── run/dashboard.json            Dashboard 端口与令牌（仅当前用户可读）
├── spool/<ulid>.json             待入库事件（只包含白名单字段）
├── spool/drops/                  丢弃计数标记（空文件，文件名中带原因）
├── spool/dead/                   无法解析的记录（7 天）
├── logs/specmetry.log            程序日志（不包含 C0 / C1 内容）
└── backups/                      一致性备份（§11.3）

<主 worktree>/.specmetry/project.json   {"schema":"specmetry.project/v1","project_id":"…","name":"…"}
```

Agent 侧的安装位置（TD-03）：

- Skill：`~/.claude/skills/specmetry/`
- Hooks：用户级 `~/.claude/settings.json`（默认），或项目级 `.claude/settings.local.json`

## 9. 接口约定

### 9.1 CLI

- 面向 Agent 的命令都支持 `--json`，输出符合 `schemas/`。
- 输入通过 `--file <path|->` 传入。
- 写命令在 JSON 中携带读取版本（[data-model.md §7.1](data-model.md#71-写入前置条件)）。
- M1 的结果与读取模型输出：`source-add.v1`、`proposal-receipt.v1`、`source-list.v1`、`proposal-list.v1`、`proposal-show.v1`、`baseline-show.v1`、`baseline-history.v1`、`question-list.v1`、`target-scope.v1`（[data-model.md §8](data-model.md#8-交换格式)）。人类专属的写命令在 MVP 中只输出人类可读文本；出错时带 `--json` 仍输出 `error.v1`。

| 退出码 | 含义 |
|---|---|
| 0 | 成功 |
| 1 | 内部错误 |
| 2 | 输入或校验错误，或引用的对象不存在（`invalid_input`、`not_found`；未写入任何数据） |
| 3 | 需要人工操作 |
| 4 | 当前目录不属于已初始化的项目 |
| 5 | 读取的版本已过期（`stale_precondition`），需要重新读取（未写入任何数据） |
| 6 | 环境或权限问题：数据目录不可写、git 不可用、数据库 schema 版本比程序新、`--isolation required` 时无法建立隔离（`isolation_unavailable`），等等 |
| 10 | `check run` 已完成，但至少有一个检查不是 `pass` |

### 9.2 HTTP API（MVP）

前缀为 `/api/v1`，所有请求都需要 `Authorization: Bearer <token>`。写请求的请求体符合 `schemas/`，错误一律返回 `error.v1`。HTTP 状态码：

| 状态码 | 错误码 |
|---|---|
| 400 | `invalid_input` |
| 401 | `unauthorized` |
| 403 | `forbidden`（Host 或 Origin 不对）、`human_required` |
| 404 | `not_found`、`not_initialized` |
| 409 | `stale_precondition` |
| 503 | `environment` |
| 500 | `internal` |

| 方法 | 路径 | 用途 | 请求体 / 响应 |
|---|---|---|---|
| GET | `/projects` | 项目列表 | — |
| GET | `/projects/{id}/status` | 完整的读取模型：目标、`S`、范围、来源与观测哈希、交付状态、阻塞差距、`as_of_seq`。Dashboard 从这里取得写请求需要回传的全部版本值。 | 响应：`status.v1` |
| GET | `/projects/{id}/overview` | 交付判定摘要、指标、最近的会话 | — |
| GET | `/projects/{id}/baseline` | 基准树，含进度和交付状态 | — |
| GET | `/projects/{id}/baseline/history` | 版本与变更 | — |
| GET | `/projects/{id}/requirements/{key}` | 需求详情：各 AC 在 `S` 上的执行、在其他上下文中的执行、Claim、决定、允许的决定类型 | — |
| GET | `/projects/{id}/gaps` | 差距 | — |
| GET | `/projects/{id}/proposals?state=pending`、`/proposals/{pid}` | 审核队列、三方对比 | — |
| POST | `/projects/{id}/proposals/{pid}/review` | 审核提交 | `review.v1` |
| POST | `/projects/{id}/baseline/edits` | 人工编辑 | `proposal.v1`（服务端强制 `origin = human_edit`） |
| POST | `/projects/{id}/scope` | 范围确认 | `scope.v1` |
| POST | `/projects/{id}/sources/{sid}/ack` | 来源变化确认 | `source-ack.v1` |
| POST | `/projects/{id}/criteria/{key}/decisions` | 验收决定 | `decision.v1` |
| POST | `/projects/{id}/questions/{key}/answer` | 回答或驳回问题 | `question-answer.v1` |
| GET | `/health` | 版本、spool 积压、drop 计数、Hook 状态、worktree 列表 | — |
| GET | `/changes` | 变更通知（SSE，TD-08） | — |

**不提供**的接口：设置状态、执行检查、删除事实、`purge`、修改交付目标（MVP 中只能在 CLI 中修改）。

### 9.3 交换格式的版本策略（TD-20）

1. **首次正式发布之前**，可以继续完善 v1，但每次修改都必须同步 schema、示例（`schemas/examples/`）和测试。
2. **正式发布之后**，按实际消费者的兼容性判断一次修改是否属于破坏性变更。
3. **输入**：目标是同时接受当前和上一个主版本。只有在实现了对上一个主版本的实际转换，并有测试验证之后，才能声称支持它。
4. **输出**：只使用当前主版本。
5. 所有 schema 都严格拒绝未知字段。如果新增的输出字段会让旧消费者校验失败，就必须升级主版本，或者采用经过验证的扩展兼容机制。
6. Skill 与本地二进制保持配套版本：Skill 由同一个二进制安装（`setup`），`doctor` 报告两者不一致。

本策略只针对本地的 CLI、Skill 和 Dashboard，不涉及云端协议协商。

## 10. 隐私与数据保留契约

本节是隐私与保留的**唯一规范来源**。其他文档只能引用本节，不能另作规定。

### 10.1 数据分级

| 级别 | 定义 | 本地持久化 | 未来云端同步（MVP 不实现） |
|---|---|---|---|
| **C0 禁止** | 源代码与文件内容；diff；**任意原始命令文本**（包括 Hook 观察到的命令）；工具输出和检查的 stdout / stderr；**环境变量的值**；提示词、完整 Agent 对话和 transcript；密钥；来源文档全文 | **不持久化**（数据库、spool、日志、备份中都不出现） | 永不同步 |
| **C1 仅本地** | 来源引用片段（≤ 500 字符，只能来自已登记的来源，按 §10.7 的策略保存）；各类路径（来源、代码引用、worktree）；检查配置（命令、工作目录、输入路径、理由——它们属于经人确认的基准）；Claim 摘要（≤ 500 字符，不能包含代码块）；人填写的理由和问题回答 | 允许（PD-20，已确认） | **默认只存在本地**；未来同步必须经过**独立授权**（见下文） |
| **C2 同步候选** | 基准条目文本（目标、需求、AC 陈述）；编号与修订；状态与计数；检查结果码、时长、提交 SHA、序号；活动元数据（会话起止、工具类别、成功或失败） | 允许 | 同样需要独立授权（见下文） |

在 MVP 中，所有数据都只在本地，**不存在任何上传路径**。

**未来云端同步的授权边界**（约束未来的设计，MVP 不实现）：

- 同步必须由用户**独立授权**：与登录或订阅分开，作出明确的同意。
- 授权按项目、按数据类别分别给出，可以随时撤销；撤销后停止同步，并说明已上传数据的处理方式。
- C1 默认不在任何授权范围内，必须单独勾选；C0 永不同步。
- 没有授权时，任何代码路径都不得上传数据（由 MVP-PRV-01 及其后续版本的网络测试保证）。

### 10.2 采集与写入规则

1. **Hook**：只写入 [`spool-event.v1`](../schemas/spool-event.v1.schema.json) 允许的字段。白名单中没有任何自由文本字段，所以不需要做脱敏。PreToolUse 拦截时，命令文本只在内存中匹配，不落盘。
2. **检查执行**：输出只实时转发给调用者（带 `--json` 时转发到 stderr，stdout 只有 JSON），不保存；只保存 [data-model.md §4.11](data-model.md#411-check_runs) 中列出的字段。
3. **检查命令**：属于 C1 基准数据，经人确认。命令中**不得**包含密钥字面量（命中密钥规则即拒绝写入）。需要秘密值时，引用环境变量名。
4. **自由文本与其他字符串**（提案文本、问题、Claim 摘要、理由、回答、备注，以及路径、检查命令和来源片段）：写入前按 §10.8 的规则集扫描写入输入中的**每个字符串值**，命中即**拒绝写入**并提示改写；不做替换（替换会改变语义）。来源片段无论 `excerpt_storage` 取何值都要扫描。所有自由文本都不能包含代码围栏（由 schema 强制）。
5. **来源**：只能登记文档类文件（[data-model.md §4.3](data-model.md#43-sources-与-source_acks)）；每个来源的片段总量有上限（§4.5），防止逐段保存来源全文。
6. **提交**：只保存 SHA、提交时间和关联关系；标题和正文在内存中匹配编号后即丢弃。
7. **程序日志**：只记录错误码、ID 和计数，不记录 C0 / C1 内容。

### 10.3 保留期

见 [data-model.md §9](data-model.md#9-数据分级保留与删除)。

### 10.4 本地 Web 服务

- 只绑定 `127.0.0.1`。
- 令牌每次启动时随机生成，通过系统浏览器打开带 fragment 的 URL 传递，**不打印到 stdout**，只保存在 `run/dashboard.json` 中（权限同 §11.1）。
- 无法打开浏览器时：在 `human_cli` 通道中打印带令牌的 URL；在 `agent_cli` 通道中只提示用户到终端执行。
- 校验 Host（只允许 `127.0.0.1:<port>` 和 `localhost:<port>`）；写请求校验 Origin。
- 设置 CSP `default-src 'self'`、`nosniff`、`no-referrer`；不启用 CORS。

### 10.5 网络边界

| 部分 | 网络 | Specmetry 的保证 |
|---|---|---|
| Specmetry 核心（CLI、Hook、Collector、Dashboard） | 不发出任何出站请求；只监听回环地址 | 保证，并有测试（MVP-PRV-01） |
| 用户的 Agent | 自行联网（模型调用等） | 不控制 |
| 检查命令 | 可能联网（例如下载依赖） | 不控制，也不保证能离线执行 |
| 离线可用的流程 | 人工或录制的提案与自述输入，加上本地可执行的检查，可以完成全部推导 | 保证 |

### 10.6 对仓库的写入

Specmetry 对项目仓库的写入仅限于：

- `.specmetry/project.json`（只在主 worktree 中；先写临时文件，再以硬链接放到位，读取方不会看到写了一半的文件。只创建、不替换：并发的 `init` 同时写入时，第一个成功，其余发现内容相同即视为已写入，内容不同则报错（退出码 6）。硬链接失败的文件系统（例如不支持硬链接的 FAT、exFAT，或不允许硬链接的挂载）上不创建 marker，返回退出码 6，并清理临时文件；不使用“先确认不存在、再重命名”的回退，因为另一个进程可能在两步之间写入 marker，随后被覆盖（codex-review-018 B2；产品负责人批准，2026-10-15）。已存在且内容相同的 marker 仍视为已写入。`init` 在登记之后才写 marker，所以此时项目已登记，提示修复问题后重新执行 `init`）；
- `.git/info/exclude`（默认写入，可用 `--no-exclude` 关闭；不可写时只给出警告）。写入的条目是 `/.specmetry/`，写进 `git rev-parse --git-path info/exclude` 所指的文件（在公共目录中，所有 worktree 共用）。只追加，原有内容逐字节保留；已有等价条目（`/.specmetry/`、`/.specmetry`、`.specmetry/`、`.specmetry`）时不再写入。
- 用户选择项目级安装时，经确认写入的 `.claude/settings.local.json`。前提是这个路径已被 git 忽略（用 `git check-ignore` 检查）；否则拒绝项目级安装，并建议改用用户级安装。

Specmetry 从不修改代码和 git 历史。

计算 `tree_state` 时，**始终排除 `.specmetry/`**（pathspec 排除），所以 marker 文件无论是否写入了 exclude，都不会让工作区变脏（[data-model.md §1](data-model.md#1-建模约定)）。检查命令本身的副作用不在此列：工作区一旦被它弄脏，这次执行就不合格。

### 10.7 来源片段的采集策略、保护与用户控制

PD-20 允许本地保存来源片段，但这**不等于**允许自动持久化任意客户文档的原文。

**采集策略**：

- **只保存显式提交的片段。** Specmetry 从不扫描、复制或导入文档全文。片段只有一个来源：提案中针对某个基准条目、作为引用依据显式提交的 `excerpt`。
- **只来自显式登记的文档。** 来源必须由人在对话或终端中确认后才登记，候选来源永远不会自动登记；只接受文档类文件（[data-model.md §4.3](data-model.md#43-sources-与-source_acks)）。
- **必须可以验证。** 每个片段不超过 500 字符，并且必须能在来源中逐字找到；每个来源的片段总量有上限（data-model §4.5）。
- **全文只在内存中短暂读取。** 读取全文只用于计算哈希和校验片段，读完即丢弃，不写入数据库、日志或备份。

**敏感数据保护**：

- 片段、摘要、理由等文本写入前都做密钥扫描，命中即拒绝；
- 不能包含代码围栏；
- 只存在本地，数据目录受权限保护（§11.1）；
- 程序日志不记录 C1 内容；
- 没有任何网络出口（§10.5）。

**用户控制**（M1 实现 `hash_only`、查看和移除来源；`full` 模式和擦除后的整理在 M2 实现，依赖 §11.4 中 N1 的修复）：

| 控制 | 作用 |
|---|---|
| 片段保存模式（项目设置 `excerpt_storage`，PD-24） | `hash_only`（**默认**）：只在内存中校验片段，数据库中只保存 `locator`、片段的 sha256 和校验结果；提案中保存的引用、输出、错误信息和日志都不包含原文（[data-model.md §4.5](data-model.md#45-source_refs-与-code_refs)）。`full`：按 PD-20 保存片段原文，只能由人用 `specmetry project set excerpt-storage full` 开启（Agent 通道返回退出码 3），开启前显示提示并要求确认；M2 起可用。切换只影响之后写入的引用。 |
| 移除来源时擦除片段（`source remove --erase-excerpts`，仅限人） | 把该来源所有引用的片段原文置空，只保留 `locator`、片段哈希和校验结果；在维护模式下执行（§11.4 中只追加约束的受控例外），并记录擦除的 `seq`。 擦除之后执行 `wal_checkpoint(TRUNCATE)` 和 `VACUUM`，与 `purge` 共用整理步骤和 `.compaction-pending` 标记（§11.4）。只有在 `full` 模式下写入过的引用才有原文可擦除，因此字节级的擦除验证与 `full` 模式一起在 M2 进行。 |
| 查看 | `source list --refs --json`（`source-list.v1`，M1）和 Dashboard（M5）列出每个来源的全部引用，以及每条引用是否保存了原文（`excerpt_state`）；人可以随时查看保存了哪些片段。 |
| 删除整个项目 | `purge`（§11.4）。 |

### 10.8 密钥识别规则集（TD-21）

**扫描范围**：所有写入类输入（提案、审核、Claim、验收决定、范围确认、来源确认、问题回答）中的**每个字符串值**，以及会被保存的命令行参数（例如 `init --name`、`-m` 理由）。包括路径、检查命令和来源片段；片段无论 `excerpt_storage` 取何值都要扫描。扫描只在内存中进行，在写事务开始之前完成。

**校验错误不回显输入**：输入按 schema 校验失败时，错误（文本与 `error.v1`）只给出字段的 JSON Pointer 和约束（关键字，以及 schema 的期望值，例如 pattern、枚举值、长度上限），不回显提交的值、意外的属性名，也不包含解析器的原始信息；无法识别的错误种类只给出关键字。指针沿提交的文档逐级对应：只有落在数组上的下标、落在对象上的 schema 声明的属性名才保留；经过提交方自定的键（例如映射的键，即使全由数字组成）时，截止到它的父级。这在共享的校验器（`internal/contract`）中统一实现，适用于所有经过校验的输入和输出。

**命中时**：

- 整次写入被拒绝（退出码 2，`invalid_input`），不写入任何数据。
- `error.details` 对每个命中的字段给出 JSON Pointer（命令行参数写成 `--<参数名>`）和 `problem = "secret_detected: <规则编号>"`；每个字段只报告按下表顺序第一条命中的规则。
- 错误输出和程序日志**绝不**包含命中的文本或它的上下文。
- 提示用户改写：检查命令改为引用环境变量名；其他文本删去或改写密钥部分。不提供绕过开关，也不做替换（§10.2）。

**不视为密钥的环境变量引用**：只有四种写法，并且被判断的值必须**整体**就是其中一个引用：`$NAME`、`${NAME}`、`%NAME%`、`$env:NAME`。命令替换（如 `$(…)`）不是环境变量引用：它和相邻的字面量照常扫描（codex-review-010 B1）；Specmetry 不解析 shell 语法。豁免只作用于对应的那一处匹配：同一字段中的其他匹配照常判断，不会因为字段里有环境变量引用就跳过整个字段。

**内置规则**（Go `regexp` 语法；"条件"是正则匹配之后的附加判断）：

| 编号 | 检测对象 | 正则 | 条件 |
|---|---|---|---|
| `private_key` | 私钥块的开头 | `-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----` | — |
| `aws_access_key_id` | AWS 访问密钥 ID | `\b(?:AKIA\|ASIA\|ABIA\|ACCA)[A-Z0-9]{16}\b` | — |
| `github_token` | GitHub 令牌 | `\b(?:(?:ghp\|gho\|ghu\|ghs\|ghr)_[A-Za-z0-9]{36,255}\|github_pat_[A-Za-z0-9_]{22,255})\b` | — |
| `gitlab_token` | GitLab 个人令牌 | `\bglpat-[A-Za-z0-9_-]{20,}` | — |
| `slack_token` | Slack 令牌 | `\bxox[abposr]-[0-9A-Za-z-]{10,}` | — |
| `stripe_secret_key` | Stripe 密钥（含测试密钥） | `\b(?:sk\|rk)_(?:live\|test)_[0-9A-Za-z]{16,}` | — |
| `google_api_key` | Google API 密钥 | `\bAIza[0-9A-Za-z_-]{35}` | — |
| `anthropic_api_key` | Anthropic API 密钥 | `\bsk-ant-(?:api\|admin)[0-9]{2}-[A-Za-z0-9_-]{20,}` | — |
| `openai_api_key` | OpenAI API 密钥 | `\bsk-(?:proj-\|svcacct-\|admin-)?[A-Za-z0-9_-]{20,}` | — |
| `npm_token` | npm 令牌 | `\bnpm_[A-Za-z0-9]{36}\b` | — |
| `jwt` | JSON Web Token | `\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}` | — |
| `url_credentials` | URL 中的用户名和密码 | `\b[A-Za-z][A-Za-z0-9+.-]{1,20}://[^\s/:@]{1,256}:([^\s/@]{1,256})@[^\s/]+` | 捕获的密码不是环境变量引用 |
| `bearer_token` | Bearer 令牌 | `(?i)\bbearer\s+([A-Za-z0-9._~+/-]+=*)` | 捕获部分不少于 20 个字符，并且同时含有字母和数字 |
| `generic_assignment` | 形如"密码 = 值"的赋值 | `(?i)(password\|passwd\|pwd\|secret\|api[_-]?key\|access[_-]?key\|auth[_-]?token\|access[_-]?token\|private[_-]?key)["']?\s*[:=]\s*["']?([^\s"',;]+)` | 值不是环境变量引用或占位符（`<…>`、`{{…}}`、全部由 `*`、`x`、`X` 或 `.` 组成）；长度不少于 8；并且同时含有字母和数字，或者长度不少于 20 |

**样本**（MVP-PRV-07：每条规则至少 2 个命中、2 个不命中样本）。命中样本在测试代码中**运行时拼接**，仓库中不保存形似密钥的字面量，以免触发代码托管平台的密钥扫描，也避免干扰 canary 搜索。写法：`+` 表示拼接，`×n` 表示重复 n 次。

| 编号 | 命中 | 不命中 |
|---|---|---|
| `private_key` | `-----BEGIN RSA PRIVATE KEY-----`；`-----BEGIN OPENSSH PRIVATE KEY-----` | `-----BEGIN PUBLIC KEY-----`；`-----BEGIN CERTIFICATE-----` |
| `aws_access_key_id` | `"AKIA" + "Q7"×8`；`"ASIA" + "ZX12"×4` | `"AKIA" + "Q7"×7 + "Q"`（只有 15 位）；`"akia" + "q7"×8`（小写） |
| `github_token` | `"ghp_" + "A1b2"×9`；`"github_pat_" + "11AB"×6` | `"ghp_" + "A1b2"×8`（32 位）；`"ghx_" + "A1b2"×9` |
| `gitlab_token` | `"glpat-" + "Ab3-"×5`；`"glpat-" + "x"×20` | `"glpat-" + "x"×19`；`"gl-pat-" + "x"×20` |
| `slack_token` | `"xoxb-" + "1234567890-abc"`；`"xoxp-" + "a1"×6` | `"xoxz-" + "a1"×6`；`"xoxb-short"` |
| `stripe_secret_key` | `"sk_live_" + "4eC3"×4`；`"rk_test_" + "Zz9"×6` | `"pk_live_" + "4eC3"×4`（可公开的 publishable key）；`"sk_live_abc"` |
| `google_api_key` | `"AIza" + "b"×35`；`"AIza" + "Xy-_"×8 + "Q9z"` | `"AIza" + "b"×34`；`"AIZA" + "b"×35` |
| `anthropic_api_key` | `"sk-ant-api03-" + "Ab9_"×6`；`"sk-ant-admin01-" + "x"×20` | `"sk-ant-short"`；`"sk_ant_api03" + "x"×20` |
| `openai_api_key` | `"sk-" + "A1b2"×12`；`"sk-proj-" + "Ab_9-"×5` | `"sk-short1"`；`"task-" + "a1"×10` |
| `npm_token` | `"npm_" + "Ab12"×9`；`"npm_" + "z9"×18` | `"npm_" + "Ab12"×8`；`"npm-" + "Ab12"×9` |
| `jwt` | `"eyJ" + "a"×10 + ".eyJ" + "b"×10 + "." + "c"×10`；`"eyJ" + "hb"×8 + ".eyJ" + "zd"×8 + "." + "S1"×8` | `"eyJ" + "a"×10 + ".eyJ" + "b"×10`（只有两段）；`"eyJ" + "a"×5 + ".eyJ" + "b"×10 + "." + "c"×10` |
| `url_credentials` | `"postgres://app:" + "s3cr3tPass" + "@db.internal:5432/app"`；`"https://deploy:" + "x9"×6 + "@example.com/repo.git"` | `postgres://app:${DB_PASSWORD}@db.internal/app`；`https://example.com/path:with:colons` |
| `bearer_token` | `"Authorization: Bearer " + "abc123"×4`；`"bearer " + "Z9y8"×6` | `Bearer $API_TOKEN`；`bearer token-based-authentication-flow` |
| `generic_assignment` | `"DB_PASSWORD=" + "s3cr3t" + "Pass9"`；`"api_key: " + "a1B2"×5` | `password: $DB_PASSWORD`；`password=<your-password>` |

测试还要断言：每个不命中样本不命中**任何**内置规则，而不只是对应的那一条。

**用户追加规则**（只能追加，不能关闭或覆盖内置规则，也没有白名单）：

```toml
# $SPECMETRY_HOME/config.toml
[secrets]
extra_rules = [
  { id = "custom_acme_token", pattern = "acme_[0-9a-f]{32}" },
]
```

- `id` 必须匹配 `^custom_[a-z0-9_]{1,40}$`，不能重复；最多 50 条。
- `pattern` 使用 Go `regexp` 语法，不超过 500 个字符，并且不能匹配空字符串。
- 任何一条无效，配置加载失败（退出码 6），Specmetry 拒绝运行，而不是忽略这条规则。错误信息只给出规则的序号、通过格式校验的规则编号和错误类别（例如正则的语法错误类型、TOML 错误的行列位置），**不回显 pattern 或配置文件的原文**。
- 追加的规则只会让检查更严格。`config.toml` 位于受保护的数据目录中（§11.1）。

**已知限制**（接受的风险）：规则集只能**降低**泄露风险，**不能保证**检测出所有秘密。

- 漏报：清单外的格式、被拆开或编码过的密钥、非文本来源中的密钥。
- 误报：由用户改写文本；不提供绕过开关。

实现位于 `internal/secrets/`（§15）。

## 11. 本地数据生命周期

### 11.1 权限

- **Unix**：数据目录权限为 `0700`，文件为 `0600`。
- **Windows**：对数据目录设置受保护的 DACL（关闭继承），只授予**当前用户 SID 和 SYSTEM** 完全控制权（TD-22，已确认）。SYSTEM 本来就能凭借特权绕过 DACL，加入它不会削弱对其他本地用户的保护，还能让 Defender、备份等系统服务正常工作。新建的文件继承这个 DACL。`doctor` 会检查实际权限，不符合时报告。
- `run/dashboard.json` 和备份文件的权限与数据目录相同。
- **写入前验证**（codex-review-004 B1）：数据目录不存在时按上述权限创建；**已存在时不予信任**，每个写入方在写入任何数据之前检查目录及其中所有条目（包括数据库、WAL、SHM、备份），任何一项不满足就拒绝写入（退出码 6），不自动修正，也不继续。检查内容：
  - Unix：属主是当前用户，没有 group 和 other 权限位。在 Linux 上这已足够：POSIX ACL 只能通过 group 类权限位（mask）授权，mask 为 0 时其他条目都不生效。macOS 的扩展 ACL 先于权限位生效，单独检查；有允许访问的条目即视为不满足。
  - Windows：只有当前用户和 SYSTEM 的允许条目；属主是当前用户、SYSTEM 或 Administrators 组；数据目录本身的 DACL 受保护；每个目录都有可继承给新文件的当前用户条目（否则 SQLite 新建的 WAL、SHM 会得到进程的默认 DACL）。
  - 两个平台都拒绝符号链接和联接（包括数据目录本身）：链接可能把写入引向不私有的位置。
  - 检查期间消失的条目跳过：另一个进程可能正在使用数据目录（例如并发的首次 `init` 中，一个进程创建数据库时 SQLite 的 `-journal` 随即出现又删除；备份轮换、压缩标记也会删除文件）。已经不存在的条目不会暴露任何内容；仍然存在的条目照常检查，数据目录本身不存在仍是错误。
- **路径上的目录**（codex-review-005 B1）：只检查数据目录本身不够。能写入某个上级目录的其他用户，可以在检查之后把数据目录改名移走，换成指向自己文件的链接，之后的写入就会落到他们手里。因此，从根目录到数据目录的每一级目录都必须**只能由当前用户或 root / 管理员改动**，否则在创建或写入任何内容之前拒绝（退出码 6）。MVP 对这类位置一律保守拒绝，不尝试防御：
  - Unix：按内核解析路径的方式逐级检查（跟随路径中的符号链接，链接经过的目录也检查）。每一级目录的属主必须是当前用户或 root；目录不能被 group 或 other 写入，除非它带 sticky 位（如 `/tmp`）且下一级条目属于当前用户或 root（此时只有条目的属主能改名或删除它）。macOS 上，路径目录带有允许访问的扩展 ACL 也视为不满足。
  - Windows：路径上的目录不能是联接或符号链接；属主必须是当前用户、SYSTEM、Administrators 或 TrustedInstaller；除这些主体外，任何人都不能对目录本身拥有 DELETE、WRITE_DAC、WRITE_OWNER、FILE_DELETE_CHILD 或完全控制权限。应用包和能力 SID（`S-1-15-…`）不算其他用户：AppContainer 进程只有在其用户本身也被授权时才能访问。
  - 这样，检查结果在检查之后不会被其他用户改变，数据目录及路径上的目录也就无法被其他用户替换。仍然可以改变它们的，只有当前用户本人和 root / 管理员，属于已声明的信任边界。网络文件系统、绑定挂载、自定义 ACL 的具体语义不在 MVP 支持范围内。

  `doctor` 使用同一套检查报告问题，但它是事后诊断，不能替代写入前的验证。

### 11.2 迁移

- 迁移只能前向执行。
- **打开数据库的固定顺序**：
  1. 数据目录与路径验证（§11.1）；
  2. 取得打开锁（见下面的"迁移协调"），直到打开完成才释放；
  3. 版本保护检查：用不修改任何文件的只读连接读取版本（§11.5），**在建立任何可能修改文件的连接之前完成**，例如 `journal_mode=WAL` 会改写回滚日志模式数据库（如恢复的 `VACUUM INTO` 备份）的文件头；
  4. **执行墓碑**（§11.4）；
  5. 如果需要迁移：生成迁移前备份，然后迁移；
  6. 释放打开锁，正常使用。

  墓碑一定在迁移前备份之前执行，因此从旧备份恢复后再迁移，新生成的备份中也不会含有已删除项目的数据。
- **迁移协调**（codex-review-005 B3）：从版本检查到连接建立、墓碑和迁移完成，整个打开过程都在 `.migrate.lock` 的操作系统排他锁内进行（Unix 用 `flock`，Windows 用 `LockFileEx`）。因此，一个进程在"判断兼容"与"建立读写连接"之间时，其他进程不可能升级数据库；升级会等它打开完成。等待超时则返回退出码 6，不修改任何文件。锁由操作系统持有，进程崩溃时自动释放，锁文件本身保留；`doctor` 只探测锁是否被持有，不创建也不删除锁文件。迁移在锁内用 `VACUUM INTO` 生成一致性备份，然后在 `BEGIN EXCLUSIVE` 事务内执行，再写入 `meta.schema_version` 和 `min_binary_version`。
- **版本保护**：程序发现数据库的 `schema_version` 高于自己支持的版本，或者自己的版本低于 `min_binary_version` 时，**拒绝打开**（退出码 6），提示升级；`min_binary_version` 无法解析时同样拒绝。拒绝时 schema、业务数据、事实和 `seq` 不变，已有的数据库、WAL、SHM 文件不被改写或删除，不执行 checkpoint；无法确定版本时一律拒绝。Specmetry 自身**只允许两种文件副作用**（产品负责人批准，2026-10-13，MVP-STO-02）：① 数据目录中还没有协调文件 `.migrate.lock` 时，打开会先创建它（空文件，权限同 §11.1，拒绝后释放），所以拒绝之后它可能留下；② 仅当另一个 SQLite 进程在版本检查期间并发关闭数据库时，原生只读连接新建的空 `-wal` / `-shm`（§11.5）。外部进程自己造成的文件变化不算 Specmetry 的副作用。这两项例外不扩大为一般性的文件修改许可，也不放宽版本保护或数据完整性约束；没有外部竞争操作时，除①外不改变任何文件。
- **写入时再次检查**（codex-review-004 B3）：已经打开的进程不会因为其他进程升级了 schema 而继续按旧结构写入。每个写事务取得写锁之后、执行写入之前，重新检查 `schema_version` 和 `min_binary_version`，不兼容时回滚并返回退出码 6，写入逻辑不会执行。同时确认连接开启了 `recursive_triggers`（[data-model.md §1](data-model.md#1-建模约定)）。这样的旧进程随后关闭最后一个连接时，SQLite 会把新程序已提交的 WAL 合并进数据库；这只是移动已提交的内容，不改变逻辑内容（codex-review-005 认定不违反 MVP-STO-02）。
- **保留 WAL 文件**：每个连接都设置 `SQLITE_FCNTL_PERSIST_WAL`，并设 `journal_size_limit = 0`。最后一个连接关闭时，SQLite 合并 WAL 并把它截断为 0 字节，但保留 `-wal` 和 `-shm` 文件。这样，关闭后的数据库始终处于只读连接可以零修改读取的状态（§11.5），WAL 中也不留下旧的页面内容。

### 11.3 备份

- 只使用一致性备份机制（`VACUUM INTO`），**不直接复制**正在使用的数据库文件。
- 自动备份只在迁移前生成，保留最近 3 份。手动执行 `backup create`、`backup restore` 属于 P1。
- 备份包含**全部项目**的数据，权限同 §11.1。

### 11.4 删除（`purge`）与墓碑

`purge` 只能在 `human_cli` 中执行，需要输入项目名确认。步骤如下：

1. 先向 `tombstones.json` 追加 `{project_id, purged_at}`，并 fsync。这一步放在最前面，保证中途崩溃后也能继续完成删除。
2. 在一个事务内删除该项目的全部行。
3. 删除属于该项目的 spool 文件。
4. 执行 `wal_checkpoint(TRUNCATE)`，然后 `VACUUM`，清除空闲页和 WAL 中的残留。
5. 列出所有在 `purge` 之前生成的备份（它们包含该项目的数据）。默认全部删除；用户也可以选择保留，此时会明确警告"这些备份中仍有该项目的数据"。
6. 询问是否删除仓库中的 `.specmetry/`。

**墓碑生效**：每次打开数据库时（在任何迁移之前，见 §11.2），如果发现库中存在墓碑里列出的项目，就自动重新执行第 2–4 步，并记录日志。这样即使从旧备份恢复（无论用的是 `backup restore` 还是手动复制文件），已删除的项目也不会复活。

删除采用与 schema 版本无关的方式：通过 `sqlite_master` 和 `pragma table_info` 找出所有带 `project_id` 列的表，逐表删除。因此，即使是旧版本 schema 的备份，也能在迁移之前完成删除。

删除时延迟外键检查（`defer_foreign_keys`），删除顺序与表名无关；项目登记表 `projects` 按 `id` 删除。整理文件（第 4 步）之前先写入标记文件 `.compaction-pending`，整理完成后才删除；如果在删除提交之后、整理完成之前崩溃，下次打开时即使已经没有可删除的行，也会完成整理（codex-review-004 N1）。

**尚未完成（N1 部分修复，最晚在 M2 的 `purge` 验收前解决）**：`wal_checkpoint(TRUNCATE)` 在其他连接持有读事务时不会报错，而是在结果中返回 busy；目前没有检查这个结果，可能在 WAL 仍含已删除数据时就删除整理标记（codex-review-005 N1）。修复方向：读取 checkpoint 的返回值，在确认合并和整理完成之后才删除标记；无法完成时保留标记并报告"整理未完成"。

**承诺边界**：`purge` 之后，项目数据不会再出现在当前数据库、WAL、spool 和已删除的备份中。Specmetry **不承诺**物理擦除（文件系统或 SSD 上的残留），也不处理用户自己制作的副本。如果 `tombstones.json` 被删除，墓碑保护随之失效（在文档中说明）。

### 11.5 只读检查

`doctor` 是**非破坏性诊断**（MVP-STO-07，2026-10-13 产品决定调整）：不执行 SQL 写入、迁移、恢复或 checkpoint；只报告已提交的数据（codex-review-005 B2）；无法保证一致读取时报告版本未知。没有外部竞争操作时，检查前后数据库、WAL、SHM 的内容和目录项完全一致（codex-review-004 B2）。它**不提供取证级的绝对只读保证**，见下文"受限的已知行为"；命令帮助（`specmetry doctor --help`）也这样说明。打开数据库前的版本检查使用同一读取方式，它的文件副作用以 MVP-STO-02 为准（§11.2）。

**做法：只使用 SQLite 自身的读取与事务隔离**，以只读连接（`mode=ro`、`readonly_shm=1`）读取，由 SQLite 的锁决定什么是已提交的数据；但只在实测确认不会修改文件的文件状态下读取。已对所用驱动（modernc.org/sqlite 1.60.1，SQLite 3.53.4）在 Windows 和 Linux 上实测，并以表征测试固定：

| 文件状态 | 只读连接的行为 | 处理 |
|---|---|---|
| `-wal` 和 `-shm` 都存在（崩溃遗留、另一进程正在使用、specmetry 关闭后保留的状态） | 不修改任何文件；只看到已提交的帧 | 读取 |
| 回滚日志模式（如恢复的 `VACUUM INTO` 备份），没有 `-wal`、`-shm`、`-journal` | 不修改任何文件；正在写入的进程持有锁时返回 `SQLITE_BUSY` | 读取；busy 时版本未知 |
| WAL 模式但缺少 `-wal` 和 `-shm`（被其他程序以普通方式关闭，或文件被删除） | 会新建 `-wal`、`-shm` 并留下 | 版本未知 |
| 只有 `-wal` 和 `-shm` 中的一个 | 会新建或改写文件 | 版本未知 |
| 存在非空的 `-journal` | 可能需要回放 | 版本未知 |

读取后再次比较数据库文件的存在情况（`doctor` 还比较大小）；有变化则同样报告版本未知。这是诊断手段，用来避免报告不可靠的结果，不能证明读取没有写入文件。`doctor` 另外**不读取 `-shm` 为空的数据库**，直接报告版本未知（codex-review-007 B2）：只读连接在外部关闭之后重建的 `-shm` 总是空的；如果读取前 `-shm` 就是空的，下一次同样的竞态会以相同的文件名和大小重建 `-wal`、`-shm`，读取前后的比较看不出来。specmetry 自己的连接关闭后不会留下空的 `-shm`，所以下一次正常打开之后，`doctor` 又能报告版本。打开前的版本检查不适用这一条：它读到的仍是已提交的数据，并在读写连接上再检查一次。不使用 `immutable=1` 或 `nolock=1`，它们会绕过 SQLite 的锁。

**受限的已知行为**（codex-review-006 R2，2026-10-13 产品决定接受）：判断文件状态与打开只读连接之间，如果另一个 SQLite 程序以普通方式关闭数据库并删除 `-wal`、`-shm`，SQLite 的只读连接会新建空的 `-wal`（Linux）或空的 `-wal` 和 `-shm`（Windows），权限与数据库相同。SQLite 的 `mode=ro` 本来就允许在目录可写时创建这些辅助文件（[SQLite 文档](https://www.sqlite.org/wal.html#read_only_databases)）。此时 `doctor` 报告版本未知，主数据库、业务数据和已有文件内容不变。只读连接以只读方式打开数据库文件，不能 checkpoint、不能删除 WAL、不能改写数据。回归测试 `TestInspectDuringExternalCloseReportsUnknown` 固定这一行为。

**同一行为也适用于打开前的版本检查**（产品负责人批准，2026-10-13，MVP-STO-02 的例外 ②）：打开会新建同样的空辅助文件，然后：Linux 上因为文件变化而报告无法确认，拒绝打开；Windows 上 SQLite 重新建出了被删除的文件，存在情况的比较看不出变化，于是按读到的已提交版本判断——不兼容则拒绝，兼容则正常打开（之后在读写连接上再检查一次）。两种情况都不会把未确认的版本当作兼容，也不会改动数据。外部关闭本身已使数据库进入 PD-23 的"缺少辅助文件"状态；Linux 上只读连接留下的空 `-wal` 使它变为"只有其一"，同样需要 `db recover`（M2）。回归测试 `TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles`。

版本未知时：`doctor` 报告警告"version unknown"及原因，不报告版本，也不报告正常；打开数据库时拒绝（退出码 6），因为无法确认兼容性。

**已知的可用性限制**：被其他程序以普通方式关闭过的数据库（`-wal`、`-shm` 被删除），或只剩 `-wal` 没有 `-shm` 的数据库，目前任何版本的 specmetry 都会拒绝打开。specmetry 自己关闭的数据库不会进入这些状态（保留 WAL 文件，§11.2）。产品负责人已决定（PD-23）：提供独立的显式恢复命令 `specmetry db recover`，普通打开和 `doctor` 保持严格；恢复必须由用户明确授权，实际检查 schema 版本、`min_binary_version` 和数据库一致性，恢复前先有一致性备份，不直接删除 `-wal` / `-shm`。尚未实现，排期与依赖见 [implementation-plan.md §8](implementation-plan.md#8-已决策待排期的功能)。

> 第二轮修复（codex-review-004 之后）曾自行解析 WAL、在内存中重建页面并通过只读 VFS 读取。该方案读取时不持有 SQLite 的锁，被 codex-review-005 证明会返回回滚日志模式下尚未提交的数据，已删除。

## 12. Hook 规格与 Agent 兼容性

### 12.1 Claude Code Hook

M3 开工前要对照官方文档核实，并固定一个经过测试的版本范围（TD-12）。

| 事件 | 用途 | 读取的输入字段（只在内存中使用） | 写入 spool 的字段 |
|---|---|---|---|
| SessionStart | 会话开始 | `session_id`、`cwd`、`source` | 会话 ID、worktree 路径、来源类型、`head_start`、`head_start_source` |
| SessionEnd | 会话结束 | `session_id`、`reason` | 会话 ID、结束原因 |
| PostToolUse | 工具成功 | `session_id`、`tool_name` | 会话 ID、工具类别（`shell` / `edit` / `read` / `other`）、`outcome = success` |
| PostToolUseFailure | 工具失败 | 同上 | 同上，`outcome = failure` |
| Stop | 一轮结束 | `session_id` | 会话 ID |
| PreToolUse（Bash、PowerShell） | 拦截 | `tool_input.command`（只在内存中匹配） | 只在拦截时写入：命中的子命令名 |

`tool_input`、`tool_response`、`transcript_path`、提示词，以及所有其他字段，一律不读取或直接丢弃。

### 12.2 项目解析与会话起点 HEAD 的获取

**项目解析**：Hook 不访问数据库，也不启动子进程。

1. 从 cwd 向上查找 `.specmetry/project.json`，找到就用它。
2. 否则，找到最近的 `.git`：
   - 如果它是**文件**（说明在关联 worktree 中），读取其中的 `gitdir:`，再读取该目录下的 `commondir`，得到公共目录；当公共目录名为 `.git` 时，主 worktree 的根目录就是它的父目录，在那里读取 `.specmetry/project.json`（marker 只存在于主 worktree）；
   - 如果它是目录，就在它所在的目录读取 marker。
3. 都找不到（例如裸仓库或非标准布局），视为非 Specmetry 项目：不写 spool，直接以 0 退出。

CLI 则通过 `git rev-parse --git-common-dir` 和数据库中的 `git_common_dir` 匹配项目，不依赖 marker 的位置。CLI 同样从目录出发：调用 git 时不传递 `GIT_DIR` 等定位变量，marker 必须与数据库一致（§6.1.2）。

**会话起点 HEAD**：同样不启动子进程，只读文件：

1. 读取 `<gitdir>/HEAD`。`gitdir` 由 cwd 向上查找 `.git` 得到；如果 `.git` 是文件（说明处于 worktree 中），就按其中的指针定位。
2. 如果 HEAD 的内容是 40 位 SHA，直接使用。
3. 如果是 `ref: <name>`，依次查找 `<commondir>/<name>` 文件和 `<commondir>/packed-refs`。
4. 遇到 reftable 格式、文件读取失败或内容格式异常时，记为 `unknown`。

工作区是否干净无法用这种方式得知，所以不采集。**在任何情况下，都不会用入库时的 HEAD 代替。**

### 12.3 延迟预算与丢弃

| 项 | 规则 |
|---|---|
| 输入上限 | stdin 最多读取 1 MiB；超出则丢弃，计一次 `oversize` |
| 内部预算 | 从启动起 200 ms 内没有完成写入，就放弃并计一次 `timeout` |
| 外部超时 | 在 Hook 配置中设置 5 s，作为兜底 |
| 退出 | 始终以 0 退出（PreToolUse 拦截按 Claude Code 协议输出拒绝决定） |
| 丢弃计数 | 在 `spool/drops/` 中写入空标记文件，ingest 时汇总到 `hook_drops`，并在健康页显示 |
| 性能目标 | p95 ≤ 100 ms（含进程启动）；非 Specmetry 项目内 p95 ≤ 30 ms |

### 12.4 兼容性矩阵

| 维度 | Claude Code（MVP 正式支持） | Codex 等（尽力而为，M4 前核实） | 无 Hook 的通用 Agent |
|---|---|---|---|
| 入口 | Skill（`/specmetry …`） | AGENTS.md 片段 + CLI（是否支持 Skill 以核实结果为准） | AGENTS.md 或人工 |
| 活动采集 | Hook | 无 | 无 |
| 人类专属操作拦截 | PreToolUse（Bash、PowerShell）+ 通道识别 | 只有通道识别 | 只有通道识别 |
| 写入数据目录 | 沙箱模式下需要把 `$SPECMETRY_HOME` 加入可写路径 | 默认的工作区沙箱通常禁止写工作区外的目录，需要用户配置可写目录；否则 CLI 返回退出码 6 并给出指引 | 视环境而定 |
| `.git/info/exclude` | 通常可写 | 可能受保护：跳过并警告 | — |
| Shell 工具 | Bash（Windows 上为 Git Bash）、PowerShell；拦截覆盖两者 | 由 Agent 决定，M4 前核实 | 视环境而定 |
| 会话标识 | Hook 提供 | 无，Claim 不关联会话 | 无 |
| `sync` 比较起点 | 上一批次或会话起点 | 上一批次或 `none` | 上一批次或 `none` |
| 人工操作路径 | 用户在**自己的终端**中执行人类专属命令，或使用 Dashboard；Agent 只负责提示，或用 `dashboard --page` 打开页面 | 同左 | 同左 |

`setup` 和 `doctor` 只**检测并给出指引**，不会自动修改其他 Agent 的沙箱配置，也不要求关闭沙箱。

## 13. 性能与可靠性

| 方面 | 设计 |
|---|---|
| 写入并发 | WAL 模式，`busy_timeout` 5 s；每个进程单一写连接；`seq` 在写事务内分配 |
| 至少一次投递 | spool 在事务提交后才删除；重复入库按事件 ID 去重 |
| 崩溃安全 | spool 先写临时文件再 rename；迁移和 `purge` 的顺序保证可以重做（§11） |
| 推导规模 | 每个 AC 只看它在 `S` 上的合格执行和最新的决定，单次遍历即可；索引为 `(project_id, criterion_id, seq)` 和 `(project_id, head_start)` |
| 指标 | 2000 个 AC、10 万条事件时，`status --json` < 1 s，Dashboard 首屏 < 2 s |

## 14. 技术栈选择与取舍

### 14.1 核心语言：Go（建议）

| 选项 | 优点 | 缺点 | 结论 |
|---|---|---|---|
| **Go** | 单一静态二进制，交叉编译简单；冷启动快，适合 Hook；标准库自带 HTTP；可以用 `go:embed` 内嵌前端；贡献者多 | 前后端两种语言 | **采用** |
| Rust | 性能好，类型系统强 | 迭代慢，贡献门槛高，对 I/O 型负载收益有限 | 不采用 |
| TypeScript（Node / Bun） | 前后端统一 | 冷启动不利于 Hook；需要运行时或体积较大的打包 | 不采用 |
| Python | 生态丰富 | 分发困难，冷启动慢 | 不采用 |

### 14.2 存储：SQLite（建议）

| 选项 | 优点 | 缺点 | 结论 |
|---|---|---|---|
| **SQLite（WAL）** | 嵌入式、单文件；事务能力足以支撑 `seq` 与前置条件校验；有 `VACUUM INTO` | 多进程写需要协调（已通过 spool 和迁移锁处理） | **采用** |
| 仓库内文件 | 可以 diff | 难以查询；与"不污染客户仓库"冲突 | 不作为事实来源（TD-01） |
| 嵌入式 KV | 简单 | 没有关系查询 | 不采用 |

驱动的选择见 TD-04，建议使用 `modernc.org/sqlite`。

### 14.3 前端：React + TypeScript + Vite（建议）

| 选项 | 优点 | 缺点 | 结论 |
|---|---|---|---|
| **React + TS + Vite**（用 `go:embed` 内嵌） | 审核队列、三方对比、时间线等交互适合 SPA；贡献者多 | 构建需要 Node；类型需要同步（TD-07） | **采用** |
| Go 模板 + htmx | 单一语言 | 复杂交互实现成本高 | 备选 |
| Svelte / Vue | 轻量 | 相对 React 没有决定性优势 | 不采用 |

### 14.4 交付形态：本地 Web（建议）

不采用桌面应用或 TUI，理由同 v0.1。

## 15. 测试策略与代码结构

| 层次 | 内容 |
|---|---|
| 单元 | 推导决策表逐条覆盖；性质测试（打乱插入顺序后结果相同）；版本前置条件校验；密钥规则；schema 校验 |
| **反例** | [acceptance-criteria.md §4](acceptance-criteria.md#4-反例与异常场景cx) 中的 CX 系列：验证设计本身不会得出错误的交付结论 |
| 存储 | 迁移、迁移锁、版本保护、只追加触发器、`purge` 加墓碑 |
| CLI 黄金测试 | `--json` 输出能通过 `schemas/` 校验并与 golden 文件一致；退出码 |
| Hook | 录制的 payload；延迟基准；canary 测试；大小和时间上限 |
| E2E | 三类夹具加录制的 Agent 输出（确定性回放），三平台 |
| 前端 | Playwright：页面过期时的 409 处理、没有外部请求 |
| Skill 评测 | 真实 Agent 运行夹具，不作为 CI 门槛 |

```text
cmd/specmetry/
internal/
  cli/  app/
  domain/
    baseline/     基准、提案、reaffirm、版本、编号
    scope/        交付目标、范围确认、来源集合哈希
    facts/        Claim、检查执行、提交关联、决定、前置条件
    derive/       纯函数：进度、交付状态、差距、指标
  store/sqlite/   迁移、seq、只追加、墓碑
  worktree/       项目与 worktree 解析、登记
  collector/  adapters/claudecode/  checkrun/  gitx/  secrets/  server/  skillpack/
schemas/  skill/  web/  testdata/fixtures/  docs/
```

依赖方向：`cli` / `server` → `app` → `domain` ← `store`。`domain/derive` 不依赖任何其他包。

## 16. 技术待决策事项

| ID | 问题 | 建议默认 | 最晚决定于 |
|---|---|---|---|
| TD-01 | 基准的事实来源 | **已确认（2026-10-14，产品负责人）**：SQLite；导出延后到 MVP 之后 | 已确认（2026-10-14） |
| TD-02 | 数据目录与数据库粒度 | **已确认**：全局单库，`~/.specmetry`，可用 `SPECMETRY_HOME` 覆盖 | 已确认（2026-10-13） |
| TD-03 | Skill 与 Hook 的安装范围 | 默认用户级，可选项目级 | M3 |
| TD-04 | SQLite 驱动 | **已确认**：`modernc.org/sqlite`。spike（v1.60.1，Windows）验证了 WAL、`wal_checkpoint(TRUNCATE)`、`BEGIN IMMEDIATE` / `EXCLUSIVE`、触发器 `RAISE(ABORT)`、`VACUUM INTO`，以及两个连接各并发写 200 次时 `seq` 唯一且连续 | 已确认（2026-10-13） |
| TD-05 | git 访问方式 | **已确认（2026-10-14，产品负责人）**：调用 `git` CLI；Hook 中只读文件 | 已确认（2026-10-14） |
| TD-06 | Windows 上执行检查的 shell | 优先 `sh`（Git for Windows），找不到时用 `cmd /C`；AC 可以指定 `shell` | M2 |
| TD-07 | API 契约与 TS 类型 | OpenAPI 引用 `schemas/` 作为请求体定义；生成 TS 类型 | M5 |
| TD-08 | Dashboard 刷新 | SSE 通知，断线时退回轮询 | M5 |
| TD-09 | 前端 UI 方案 | 待定；约束：资源全部内嵌 | M5 |
| TD-10 | CLI 框架 | **已确认**：cobra | 已确认（2026-10-13） |
| ~~TD-11~~ | ~~证据新鲜度~~ | **已关闭**：由交付快照模型取代（AD-12） | — |
| TD-12 | Claude Code Hook 事件名与字段的核实，以及兼容版本范围 | 以 §12.1 为初稿 | M3 开工前 |
| TD-13 | 人类专属操作的防护强度 | 启发式通道识别 + 拦截 + 审计记录，并如实声明能力边界 | M2 |
| TD-14 | 编号格式 | **已确认（2026-10-14，产品负责人）**：稳定的编号，永不复用：`GOAL-1`、`REQ-001`、`AC-001-1`、`Q-001` | 已确认（2026-10-14） |
| TD-15 | 分发渠道 | Releases + 安装脚本 | M6 |
| TD-16 | 迁移策略 | **已确认**：只前向迁移 + 迁移锁 + `VACUUM INTO` 备份 + `min_binary_version` | 已确认（2026-10-13） |
| TD-17 | 提交关联规则 | trailer 优先；正文中的编号也匹配，可以关闭 | M2 |
| TD-18 | 每个 AC 的检查数量 | **已确认（2026-10-14，产品负责人）**：至多 1 个 | 已确认（2026-10-14） |
| TD-19 | Dashboard 端口 | 固定默认端口，被占用时递增 | M5 |
| TD-20 | 交换格式的版本策略 | **已确认（2026-10-14，产品负责人）**：调整后的六条策略，见 §9.3（发布前可完善 v1，但须同步 schema、示例和测试；输出只用当前主版本；兼容上一个主版本的输入必须有实际转换和验证；Skill 与二进制配套） | 已确认（2026-10-14） |
| TD-21 | 密钥识别规则集与误报处理 | **已确认（2026-10-14，产品负责人）**：§10.8 的 14 条内置规则及其正反样本；命中即拒绝持久化，不回显敏感内容；只能追加规则；误报时由用户改写文本（不提供绕过开关）；规则集只能降低泄露风险，不能保证检测出所有秘密 | 已确认（2026-10-14） |
| TD-22 | Windows ACL 的具体授权对象 | **已确认**：当前用户 + SYSTEM，受保护的 DACL（关闭继承），子对象继承 | 已确认（2026-10-13） |
| TD-23 | `check.inputs` 的对象 ID 粒度 | 文件取 blob ID，目录取 tree ID | M2 |
| TD-24 | 隔离执行器的后端与验证方法：物化方式（`git worktree add --detach` 或 `git archive`）；各平台的只读强制机制（容器只读挂载、Linux 上的 bubblewrap / Landlock、macOS 沙箱、Windows 待评估）；执行前写入探测与执行后树比对的实现；子模块、LFS 和依赖缓存的处理 | 待定；约束：不能依赖文件权限位；无法验证时绝不标为隔离级；普通项目不强制安装 | 隔离执行器开工前（MVP 之后） |
| TD-25 | 仓库移动或复制后的项目身份：marker 与数据库不一致时，如何判断是原项目换了位置（应继承原项目），还是复制出的新项目 | M1（WP1.1）采取保守做法：拒绝并说明原因，不写入；由人删除 marker 后，登记为新项目（§6.1.2）。是否提供人类专属的"迁移项目位置"命令，待定 | M4（自测之前） |
