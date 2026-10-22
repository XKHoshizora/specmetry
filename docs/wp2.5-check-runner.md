# WP2.5 Check Runner（观测级）规范

> 状态：**草案，第 1 版**（2026-10-22），**未经独立审查，未冻结**。产品负责人 2026-10-22 授权 WP2.5 的规范准备、必要的隔离实验、文档提交与独立 PR；**不授权** WP2.5 的生产实现、规范冻结或本 PR 的合并（[review-resolution.md §84](review-resolution.md)）。
>
> 本文不是实现：不含生产代码、迁移 SQL 或测试代码。它细化观测级检查执行的处理顺序、解释器、结果分类、进程树、输出、持久化与验收，并把需要负责人决定的事项逐项列出（§17）。
>
> 规范性的条文仍以 [data-model.md](data-model.md)（§4.11、§4.11.1、§4.11.2、§6、§7.4）、[architecture.md](architecture.md)（§6.4、§9.1、§10、§16）、[prd.md](prd.md)（FR-EV、FR-WT、FR-INV-04）、[acceptance-criteria.md](acceptance-criteria.md) 与 [`schemas/`](../schemas/) 为准。本文中标为“提议”“推荐”“草案”的内容都**没有获批**，不改变那些文档；获批之后再按 §18 同步。两者不一致时，以那些文档为准，并修正本文。

## 0. 摘要

- **范围**：`check run [AC...] [--all] [--isolation off|required] [--json]`，在调用者所在的 worktree 中以观测级执行已确认的检查，记录 `check_runs`，输出 `check-run.v1`。隔离执行器不在范围内（M7）。
- **已经可以依据的规则**（§2）：FR-EV-01 ~ 06、08；data-model §4.11、§4.11.1、§4.11.2（M2-Q2）、§7.4；PD-22（MVP 只有观测级）；PD-26；TD-23 的读取机制（WP2.3 规范 §4，已冻结，`gitx.TreeEntry` 已随 WP2.3 库层验收）；WP2.3 的 `gitx.Clean`（P1）与私有临时区（方案 B）；推导规则（WP2.1、WP2.2 已验收）。
- **还没有批准的**：TD-06 / M2-Q1（默认 shell、解释器、结果分类、Windows 命令行、`cmd` 多行），以及 EV-04 的契约冲突。产品负责人 2026-10-18 决定把 TD-06 的截止调整为 **WP2.5 开工前**（M2-Q10，§55）；准备报告 §5.1 的推荐**不是已批准的默认**（§55 明确列为未批准）。
- **本轮实验**（§16，作者，Windows 本机与 WSL Ubuntu；macOS 未验证）得出的新事实，改变了准备报告中的部分推荐：
  - Windows 上直接启动 Git for Windows 的 `usr\bin\sh.exe` 时，PATH 中没有 `/usr/bin`，`grep`、`sed`、`sleep` 都找不到；`bin\sh.exe` 包装程序会补上它们（§5.4）。
  - Git for Windows 的 MSYS 运行时用自己的规则解析命令行：按 MSVCRT 规则“总是加引号”时，`\\` 会变成 `\`；改为 MSYS 规则后 23/23 个测试字符串原样到达（§5.5）。准备报告“11 个字符串全部原样到达”的结论只覆盖了不含连续反斜杠的字符串。
  - `cmd /d /s /c` 遇到多行命令时，CRLF 与 LF 都只执行第一行，退出码 0，没有任何提示（§5.6）。
  - 同一个含 CR 的 `sh` 命令，Linux（dash）把 CR 当作普通字符，Git for Windows 的 shell 忽略它（§5.7）。
  - Windows Job Object：挂起创建、加入 Job、恢复，Git Bash 的全部后代都在 Job 中并被终止；不允许 breakaway；持有 Job 的进程意外退出时，后代被终止（§8.2）。
  - Linux 进程组：`setsid` 可以离开进程组；创建进程组的进程被 SIGKILL 时，组内进程全部存活；`Pdeathsig` 只覆盖直接子进程。把 Specmetry 设为 child subreaper 之后，离开进程组的后代在父进程死亡时回到 Specmetry 名下，可以被清扫（§8.2）。
  - 通过管道复制输出时，孙进程持有管道会让 `Wait` 一直阻塞；让检查直接继承输出句柄则不会（§9.1）。
- **待决事项**：18 项（D1 ~ D18，§17），每项给出推荐、代价与最晚决定点。其中 D1 ~ D5 属于 TD-06 / M2-Q1 与 EV-04，必须在 WP2.5 实现开工前决定。

## 1. 范围

**在范围内**：

- 命令：`specmetry check run [AC...] [--all] [--isolation off|required] [--json]`（prd §6）。
- 选择：显式给出的 AC，或 `--all`（全部 `active`、自动化、有检查的 AC）。
- 执行前的检查：参数、隔离模式、选择、静态规则、解释器、PD-26、私有临时区。
- 观测级执行：开始与结束时的代码上下文、`inputs` 的输入记录（TD-23，已冻结）、在 `W/<workdir>` 中执行、环境、进程树、超时、取消、输出转发。
- 结果分类、合格判定、`check_runs` 的写入（迁移 0009）、`seq`、worktree 的首次登记。
- 输出：`check-run.v1`（带 `--json`）或文本；错误为 `error.v1`；退出码 0 / 10 等。
- `--isolation required`：按冻结的规则一律退出码 6，不执行、不记录（不实现隔离执行器）。

**不在范围内**：

| 内容 | 归属 |
|---|---|
| 隔离执行器、`snapshot_isolated` 证据、`--isolation auto`、沿用的实际发生 | M7（TD-24、FR-EV-09） |
| `status`、`gaps` 中对执行结果的展示，以及触发扫描的命令 | WP2.7 |
| 验收决定（`evidence_id` 的使用） | WP2.6 |
| 会话关联（`session_id`）、Hook、PreToolUse 拦截 | M3（M2 中 `session_id` 为 NULL） |
| 推导规则与复核器 | WP2.1、WP2.2（已验收，不改） |
| 完整 CX 套件、E2E 的 CLI 版本、夹具 | WP2.9 |
| 从 Dashboard 执行检查 | MVP 不做（mvp.md §3） |
| 导入 CI 结果 | P2（FR-EV-07） |

## 2. 依据：已批准的规则与历史草案

| 依据 | 状态 | 内容 | 本文的用法 |
|---|---|---|---|
| FR-EV-01 ~ 06、08；FR-WT-02、03；FR-INV-04 | 已批准（prd） | 绑定开始时的修订、配置哈希、输入与上下文；输出只转发不保存；`--json` 时 stdout 只有一份 `check-run.v1`；超时终止整个进程树；只执行基准中已确认的命令；`context_changed`；只追加；观测级标注；`--isolation required` 退出码 6 | 全文 |
| data-model §4.11、§4.11.1 | 已冻结 | `check_runs` 的字段；观测级只保证两个时刻；合格执行的条件；每次执行一个 `seq`，结束之后写入 | §7、§10 |
| data-model §4.11.2（M2-Q2，2026-10-18 批准） | 已冻结 | 输入记录 `{path, mode, type, oid}`；查找按 V-P04 规范化；只要求路径元数据；四个读取层次 | §7.3，**不重开** |
| WP2.3 规范 §4（TD-23 的读取机制，2026-10-19 冻结） | 已冻结；`gitx.TreeEntry` 已随库层验收（§75） | 每个路径一次 `ls-tree`，仓库根用 `rev-parse`；只读条目元数据 | §7.3，**不重开** |
| WP2.3 规范 §4.3、§14.3（P1、方案 B） | 已冻结；库层已验收 | `gitx.Clean`：不能判定时为 `undecidable`（记 `unknown`），从不误报 `clean`；私有临时区按方案 B | §7.1 |
| data-model §7.4 | 已冻结 | 观测级执行的 5 个步骤；`--isolation required` 一律退出码 6 | §6 |
| data-model §6（WP2.1、WP2.2） | 已验收 | `runs(S)`、`V`、`qualifying` 的使用 | §12 |
| architecture §6.4 | 已批准 | 进程管理（Unix 进程组、Windows Job Object）；继承用户环境并注入 `SPECMETRY_CHECK=1`；默认串行；输出流 | §8、§9 |
| architecture §9.1 | 已批准 | 退出码，含 `10`（检查已完成但不是全部 `pass`） | §11 |
| PD-22、PD-26、TD-18、TD-26 | 已确认 | 只有观测级；对象读取的安全门禁；每个 AC 至多一个检查；只支持 SHA-1 | 全文 |
| `common.v1` 的 `checkSpec` | 已冻结 | `command`（1 ~ 2000）、`workdir`（默认 `.`）、`shell`（`sh`、`bash`、`cmd`、`pwsh`、`powershell`，可省略）、`timeout_sec`（1 ~ 3600，默认 300）、`inputs`（≤ 50） | §5 |
| `check-run.v1` | 已冻结 | 输出字段 | §11 |
| M2-Q10（2026-10-18 决定） | 已决定 | TD-06 / M2-Q1 的截止为 WP2.5 开工前 | §17 |
| TD-06 的建议默认（architecture §16） | **未确认** | “优先 `sh`（Git for Windows），找不到时用 `cmd /C`” | 只作为历史草案，§17 D1 列出取舍 |
| M2 准备报告 §5.1（M2-Q1） | **未批准**（§55） | 默认 `sh`、不回退、字段映射、解释器查找、结果层次、EV-04 的方案 A ~ C、Windows 命令行、`cmd` 多行、进程树 | 历史草案；本文按实验修订，§17 D1 ~ D5 |
| M2 准备报告 §5.11 的 WP2.5 行 | 实施细节的推荐（未批准） | `--all` 的对象、串行、退出码 10；`check_runs.evidence_grade` 在 MVP 中只接受 `observed` | 工程细化，§17 列出 |
| 审查 040 B1、N1；审查 041 | 历史审查 | 127 不能反证“命令不存在”；解释器查找需排除 worktree；各 shell 各自适配；进程树需验证 | 本文的约束 |

## 3. 命令与选择

- 语法：`specmetry check run [AC...] [--all] [--isolation off|required] [--json]`。
  - 不接受临时命令或命令片段（FR-EV-03，MVP-EV-06）：执行的只能是基准中已确认的检查配置。
  - `--isolation` 省略时为 `off`。`auto` 不是 MVP 的取值（prd §6 的语法只有 `off|required`），按命令行错误处理（退出码 2）。
  - 同时给出 AC 与 `--all`，或两者都没有 → 退出码 2。
- Agent 与人的 CLI 都可以执行（prd §3 的权限表：“触发检查”对两者都是 ✓，Dashboard 为 ✗；prd §6 的命令表为“任意”）。它不是人类专属操作，不调用 `RequireHuman`；通道识别的结果只作为 `triggered_channel` 记录。
- **显式的 AC**：每个编号必须是当前 `active`、`verification = automated` 且有检查的 AC。不存在、已退役、只在待审提案中、手动 AC、没有检查 → 退出码 2，`details` 逐项指出参数的位置；整批都不执行（MVP-EV-06）。重复的编号见 §17 D15。
- **`--all`**：全部 `active`、自动化、有检查的 AC，**与交付范围无关**（准备报告 §5.11；FR-INV-04、MVP-INV-03）。没有符合条件的 AC 时，不执行任何检查，退出码 0，`runs` 为空数组。
- **顺序**：按 AC 编号的数值顺序（需求号，再按 AC 序号）串行执行；显式给出时同样按编号排序，不按参数顺序（工程细化 E1）。
- **配置的固定时刻**：见 §17 D13（推荐在命令开始时一次读取全部所选 AC 的配置）。

## 4. 处理顺序

拒绝为退出码 2，未初始化为 4，环境为 6，内部错误为 1。**“不执行、不记录”与“执行之后不合格”严格区分**（§7.5 的总表）。

### 4.1 执行任何检查之前（失败时不执行、不写记录、不分配 `seq`）

1. **解析命令行**（architecture §9.1 的统一处理）。
2. **`--isolation required`**：立即退出码 6（`environment`，`details` 中有 `isolation_unavailable`），**不打开数据库，不运行 git，不启动任何进程**（data-model §7.4，MVP-EV-13，CX-28(a)）。
3. **确定项目与 `W`，打开数据库**（architecture §6.1.2）：`W` 是 `gitx.Locate` 给出的 worktree 根，非 git 项目为项目根；没有数据库或项目未登记 → 退出码 4；版本、墓碑与只读拒绝按既有顺序先行。第 6 步的 worktree 排除用到 `W` 与本项目已登记的全部 worktree。
4. **读取选择**（一个读事务）：所选 AC 的 `r`、`ar`、`h` 与检查配置（§17 D13）；不符合 §3 的条件 → 退出码 2。
5. **静态规则**：`shell = cmd` 的命令含 CR 或 LF（§17 D3）、`sh` / `bash` 的命令含 CR（§17 D4）→ 退出码 2，在获批之前不实施。
6. **解释器**（§5.3）：对所选 AC 用到的每一种 `shell` 查找解释器。任何一个不可用 → 退出码 6，`details` 给出 `shell_unavailable` 与受影响的 AC 编号，不回显路径（准备报告 §5.1 第 ① 层；§17 D1）。
7. **git 项目**：
   - PD-26：git 不具备安全读取对象的能力 → 退出码 6，在运行任何读取对象的命令之前拒绝（与 `claim`、`propose` 相同的入口）；
   - 准备私有临时区（`gitx.NewScratch`，WP2.3 §14.3）；
   - 解析交付目标的快照 `S`（一次，§11.2 `is_target_snapshot`）：没有 `active` 目标或 ref 无法解析 → `S` 为 NULL，不是错误；读取失败（对象损坏、PD-26）→ 退出码 6。
8. **worktree 的登记**：`W` 尚未登记时，在第一次写入的事务中登记（§10.3）；登记的路径先做密钥扫描（§9.3）。

### 4.2 每一次执行（按 §3 的顺序串行）

对第 k 个 AC：

1. **开始上下文**（§7.1）：`head_start`、`tree_state_start`。读取失败（环境错误）→ 退出码 6，**第 k 个不执行、不记录**，之前已写入的执行保留，之后的不再执行（§11.4）。
2. **输入记录**（§7.3）：声明了 `inputs` 时，在 `head_start` 上逐个读取：
   - 读取路径所需的对象失败（L1）→ 同第 1 步：退出码 6，第 k 个不执行、不记录；
   - 任一路径不存在（L2，含 `head_start` 为 NULL）→ 结果为 `error` / `input_missing`，**不启动进程**，转第 5 步（仍读取结束上下文并记录）。
3. **启动**（§5、§8）：在 `W/<workdir>` 中，经所选解释器的适配器启动；标准输入为空设备；输出句柄按 §9.1；进入新的进程组或 Job。
   - 进程没有被创建（操作系统在创建时返回错误）→ `error` / `spawn_error`，`exit_code` 为 NULL，转第 5 步；
   - 进程已创建、但无法建立进程树的管理（Windows 上加入 Job 失败）→ 终止这个仍处于挂起状态的进程，退出码 6，第 k 个不记录（工程细化 E5：用户的命令一行也没有运行）。
4. **等待**：主进程退出、超时（`timeout_sec`）或取消（§8.3、§8.4）。主进程退出之后，终止进程树中剩余的进程（§17 D11），并等它们全部结束。
5. **结束上下文**（§7.1）：`head_end`、`tree_state_end`，在进程树全部结束之后读取。读取失败见 §17 D6。
6. **分类**（§6.2）：`result`、`failure_kind`、`exit_code`、`duration_ms`。
7. **写事务**（`BEGIN IMMEDIATE`，§10）：必要时首次登记 worktree（另取一个 `seq`）；分配一个 `seq`，写入这一条 `check_runs`；用写入的行构造输出中的这一项，并在**提交之前**按 `check-run.v1` 中执行项的定义校验（§11.3）；提交。失败 → 见 §11.4（检查已经执行，没有记录；不重新执行）。
8. 下一个 AC。

### 4.3 全部执行之后

1. 用写入的行构造 `check-run.v1`，校验，编码。
2. 输出（§11）。任一执行不是 `pass` → 退出码 10；否则 0（MVP-EV-02、MVP-INV-03）。
3. 提交之后的输出失败见 §11.4。

客户端不自动重试：重试会再次执行检查命令。收到退出码 6 时，先读取状态（WP2.7），再决定是否重新执行。

## 5. 解释器（TD-06 / M2-Q1，待决）

### 5.1 已有的约束（不变）

- `shell` 的取值与可省略（`common.v1`）；`check_config_hash` 包含 `shell`（省略时为 null），`shell` 是强制 `material` 的字段（data-model §4.4、§7.2）。所以同一个配置哈希必须对应同一种解释器的选择规则，不能因机器而变。
- 检查继承用户的环境，注入 `SPECMETRY_CHECK=1`（architecture §6.4）。
- TD-06 的建议默认与准备报告 §5.1 的推荐都**没有获批**（§2）。

### 5.2 默认解释器（§17 D1）

**推荐**：省略 `shell` 时，**所有平台都用 `sh`**，**不回退到 `cmd`**。解释器不可用时整批不执行（§4.1 第 6 步）。

- 统一的是**解释器的种类与语言约定**（POSIX shell），不是同一个二进制或相同的执行环境：Linux 的 `/bin/sh` 常是 dash（实验环境即是）；Git for Windows 的 `sh.exe` 是 bash 的 POSIX 模式；macOS 的 `/bin/sh` 默认也是 bash 的 POSIX 模式（未验证）。
- 理由：同一个已确认的配置（同一个 `check_config_hash`）不应在有、没有 Git Bash 的 Windows 机器上执行语法不同的程序；实验表明 `cmd` 对多行命令只执行第一行且不报错（§5.6），静默回退会把一个多行 `sh` 检查变成“只执行第一行、然后通过”。
- 代价：没有 Git for Windows `sh` 的 Windows 机器上，省略 `shell` 的检查返回退出码 6，需要安装 Git for Windows 的完整安装，或在审核中把配置改为 `pwsh` / `cmd`。

### 5.3 解释器的查找（§17 D1、D2）

共同规则（准备报告 §5.1 第 3 条与审查 040 N1，按实验细化）：

- 只使用**绝对路径**；PATH 中的相对路径项一律忽略（Go 1.27 的 `exec.LookPath` 也不在当前目录查找）。
- 候选路径先**规范化**（解析符号链接与联接；Windows 上统一短文件名与大小写，取最终路径），再判断实际位置：位于本项目**任何已登记 worktree** 之内，或当前 worktree 之内的，一律不用。理由：PATH 可能含有某个 worktree 中 `bin` 目录的绝对路径。
- 查找在 §4.1 第 6 步对所有用到的 `shell` 一次完成，之后启动时使用同一个路径；启动时它已不存在 → `spawn_error`（§6.1 第 ② 层）。
- 不使用 `ComSpec`、`SystemRoot` 等环境变量决定解释器的位置；系统目录由 `GetSystemDirectory` 取得（实验：`C:\WINDOWS\system32`）。

| `shell` | Linux / macOS | Windows |
|---|---|---|
| 省略、`sh` | `/bin/sh`（POSIX 规定的位置，不经 PATH） | Git for Windows 的 `sh`：由 `internal/gitx` 所用的 git 推出安装根（§5.4） |
| `bash` | PATH 上第一个合格的绝对路径 | Git for Windows 的 `bash`（同上） |
| `cmd` | 不可用（退出码 6） | `<系统目录>\cmd.exe` |
| `pwsh` | PATH 上第一个合格的绝对路径 | PATH 上第一个合格的绝对路径（实验：`C:\Program Files\PowerShell\7\pwsh.exe`） |
| `powershell` | 不可用（退出码 6） | `<系统目录>\WindowsPowerShell\v1.0\powershell.exe` |

- Windows 上的 `sh`、`bash` **只取 Git for Windows**，不回退到 PATH 上的其他 `sh`（例如 WSL 的 `WindowsApps\bash.exe` 启动器、Cygwin、独立的 MSYS2、busybox）：它们各有不同的命令行解析规则（§5.5），本轮只验证了 Git for Windows 的 MSYS 运行时。这比准备报告“其次是 PATH 上的绝对路径项”更窄（§17 D2）。
- `git --exec-path` 在 `gitx` 的受控环境中运行，不继承 `GIT_EXEC_PATH` 等变量（审查 040 N1）。实验中它给出 `C:/Program Files/Git/ucrt64/libexec/git-core`：安装根是 `git-core` 往上第三级的目录（`git-core` → `libexec` → `ucrt64` → 安装根；`mingw64`、`clang64` 等与 `ucrt64` 在同一层）。推不出、或推出的根中没有 `usr\bin\sh.exe` → `sh` 不可用。

### 5.4 Windows 的 `sh` 入口与 PATH（新发现，§17 D2）

**实验**（§16 X6）：PATH 只含系统目录时：

| 入口 | 检查中能找到的工具 | 检查看到的 PATH | 其他变化 |
|---|---|---|---|
| `<Git>\usr\bin\sh.exe` | `test`（内建）；`sleep`、`grep`、`sed`、`git` 都找不到 | 与调用者相同 | — |
| `<Git>\bin\sh.exe`（包装程序） | 全部找到 | 前面加上 `/ucrt64/bin`、`/usr/bin`、`~/bin` | 另设 `HOME`、`EXEPATH`、`PLINK_PROTOCOL=ssh`、`TERM=xterm-256color` |

在普通的 PowerShell 会话中，Git for Windows 的默认安装只把 `Git\cmd` 放进 PATH（准备报告 §5.1 的实测：PATH 上没有 `sh`）。所以直接使用 `usr\bin\sh.exe` 时，`go test ./... | grep -q ok` 这类检查会因 `grep` 不存在而失败，并且这种失败与真正的检查失败无法区分（§6）。

| 方案 | 做法 | 代价 |
|---|---|---|
| **W2（推荐）** | 使用 `<Git>\usr\bin\sh.exe` / `bash.exe`，Specmetry 只在检查的 PATH **前面**加上 `<Git>\usr\bin`，其他变量不动 | 偏离“只继承用户环境、只注入 `SPECMETRY_CHECK=1`”（architecture §6.4），需要批准并写明；`<Git>\usr\bin` 中的 `find`、`sort` 等在 `sh` 中优先于 Windows 自带的同名程序（这正是 POSIX shell 的预期） |
| W1 | 使用包装程序 `<Git>\bin\sh.exe` / `bash.exe` | 与用户打开 Git Bash 时的环境最接近；但 PATH 之外还改了 `HOME`、`TERM` 等，改动由 Git for Windows 决定、随版本变化；多一层进程（实验：退出码、参数与进程树都正常） |
| W3 | 使用 `usr\bin\sh.exe`，不改环境 | 检查能用哪些工具取决于调用者的 PATH，同一配置在不同会话中结果不同 |

无论哪种方案，实验只覆盖 Git for Windows 2.56（UCRT64）的一个安装；其他版本与安装方式（精简安装、便携版、Scoop 等）未验证。

### 5.5 命令行与参数（实施细节，按实验修订）

- **Linux / macOS**：直接以 argv 传给 `execve`：`[解释器, -c, 命令]`，或 PowerShell 的 `[pwsh, -NoLogo, -NoProfile, -NonInteractive, -Command, 命令]`。实验（WSL，dash 与 bash）：23/23 个测试字符串原样到达。macOS 未验证（推断相同）。
- **Windows**：一律自行构造完整的命令行（`SysProcAttr.CmdLine`），不使用 Go 的默认转义。**每种解释器的规则不同**：

| 解释器 | 命令行 | 实验（23 个测试字符串，§16 X5） |
|---|---|---|
| Git for Windows `sh`、`bash` | `"<解释器>" -c "<命令>"`，命令按 **MSYS 规则**加引号：每个 `\` 写成 `\\`，每个 `"` 写成 `\"` | 23/23 原样到达；按 MSVCRT 规则（只在引号前加倍反斜杠）时 21/23，`two\\`、`\\server\share\` 中的 `\\` 变成 `\`；Go 的默认转义同样 21/23 |
| `pwsh`、`powershell` | `"<解释器>" -NoLogo -NoProfile -NonInteractive -Command "<命令>"`，按 **MSVCRT 规则**总是加引号 | 23/23（逐字节，经 UTF-8 十六进制比较） |
| `cmd` | `"<系统目录>\cmd.exe" /d /s /c "<命令>"`，命令**原样**放在引号之间 | `cmd` 自己解析引号与 `&`、`|`、`^`、`%` 等元字符，这是所选 shell 的语义；Specmetry 不转义 |

- 测试字符串含：空格、单双引号、`\`（单个、末尾、成对、UNC 前缀）、`\"`、`*`、`{a,b}`、`[ab]`、`~`、`$HOME`、`%PATH%`、`!`、`^`、`&|<>`、制表符与中文。
- `/d` 关闭 `cmd` 的 AutoRun，`-NoProfile` 不加载 PowerShell 的配置文件：它们不属于经人确认的检查配置。`sh`、`bash` 以非交互方式运行，不读启动文件；继承的 `BASH_ENV`、`ENV` 照常生效，属于“继承用户环境”的边界，如实说明。
- 不使用 `-EncodedCommand`（准备报告：输出被重定向时会混入 CLIXML）。
- 每种解释器各有自己的适配器与测试，不以一种解释器的实验推论其他解释器（审查 040 N1）。

### 5.6 `cmd` 与多行命令（§17 D3）

- **实验**（§16 X4）：`cmd /d /s /c "echo one<CRLF>echo two"` 与 `<LF>` 两种写法都只输出 `one`，退出码 0，没有任何提示。`sh` 与 `powershell` 正常执行多行。
- **推荐**（与准备报告相同，需批准）：`shell = cmd` 时命令不得含 CR 或 LF。提交、审核与人工编辑时拒绝（扩展 V-P04，退出码 2）；已保存的此类检查，`check run` 在执行前拒绝（退出码 2，§4.1 第 5 步），不写记录。批准之前，不改变已保存配置的含义。
- 代价：多行的 Windows 批处理需要写成单行（`&` 连接）或改用 `pwsh`。

### 5.7 `sh` / `bash` 命令中的 CR（新发现，§17 D4）

- **实验**（§16 X4）：命令 `echo one\necho two\r\necho three`：Linux 的 dash 输出 `two\r`（CR 是 `echo` 参数的一部分）；Git for Windows 的 `sh` 输出 `two`（CR 被忽略）。
- 影响：在 Windows 编辑器中写成 CRLF 的命令，同一个配置哈希在 Linux 上可能失败（例如 `test -f a.txt\r` 查找名为 `a.txt\r` 的文件）。
- **推荐**：`sh`、`bash` 的命令不得含 CR（LF 可以），与 D3 同样处理（V-P04 的扩展，退出码 2；已保存的在执行前拒绝）。备选：不限制，在文档中说明 CR 的平台差异。

## 6. 结果分类与 EV-04（§17 D5）

### 6.1 层次

不用任何普通退出码反向证明“命令不存在”，也不依赖 stderr 文本（审查 040 B1）。本轮实验补充了 Linux 的数据（§16 X3）：

| 层次 | 例子 | Specmetry 得到的证据（实验） | 推荐的处理 |
|---|---|---|---|
| ① 解释器不可用 | 找不到解释器；平台上不存在的取值（Unix 的 `cmd`、`powershell`）；Windows 上没有 Git for Windows 的 `sh` | §4.1 第 6 步查找失败，进程没有创建 | 整批不执行、不记录，退出码 6，`shell_unavailable` |
| ② 进程无法创建 | 工作目录不存在或不是目录；查找之后解释器被删除；不是可执行文件；没有执行权限 | 创建进程时操作系统返回的错误，进程不存在。实验：Windows 为 errno 267（目录）、2（不存在）、216（不是程序）；Linux 为 ENOENT、ENOTDIR、ENOEXEC、EACCES | `error` / `spawn_error`，`exit_code` 为 NULL；只记录类别，不记录错误文本或路径（工程细化 E6：`check_runs` 不增加错误类别列，类别只用于输出的说明） |
| ③ 解释器已运行，命令不存在 | `no-such-tool --check` | 只有退出码与 stderr。实验：dash 127 = `exit 127`；bash 127 = `exit 127`；`cmd` 1 = `exit /b 1`；`pwsh`、`powershell` 1 = `exit 1`。stderr 文本随语言与版本变化，且属于不读取、不保存的检查输出 | **不能可靠识别**，见 §6.3 |
| ④ 命令以非零状态退出 | 测试失败；`exit 1` | 退出码 | `fail` / `nonzero_exit`，记录实际退出码 |
| ⑤ 解释器启动之后自身失败 | MSYS 初始化失败（审查 040 的 `0xC0000022`）；崩溃 | 只有退出码，一般无法与 ④ 区分 | `fail` / `nonzero_exit`。不增加“空命令预检”（准备报告的可选缓解）：它只能减少、不能排除这类误记，还要为每种解释器多启动一个进程（工程细化 E7） |
| ⑥ 被信号终止（Unix） | 检查被外部 `kill -9`、OOM | 没有退出状态，只有信号（实验：Go 的 `ExitCode()` 为 −1） | `fail` / `nonzero_exit`，`exit_code` 的取值见 §17 D12 |
| ⑦ 超时 | 长时间运行 | Specmetry 自己的计时 | `error` / `timeout`，终止整个进程树，`exit_code` 为 NULL |
| ⑧ 取消 | Ctrl+C、SIGTERM、控制台关闭 | Specmetry 收到的信号或控制台事件 | §17 D8 |

- Windows 上被他人终止的进程，退出码是终止者给出的任意值（例如 `TerminateProcess(h, 1)`），与 ④ 无法区分，按 ④ 处理。
- 实验中，Git for Windows 的 `sh` 被 `kill -9 $$` 终止时，Windows 退出码为 2304（MSYS 的约定，信号号乘以 256），按 ④ 记录实际退出码。

### 6.2 分类规则（按优先级，命中第一条即停止）

| # | 条件 | `result` | `failure_kind` | `exit_code` |
|---|---|---|---|---|
| 1 | 上下文变化（§7.2） | `error` | `context_changed` | 主进程正常退出时为实际退出码，否则 NULL |
| 2 | 输入路径不存在（§7.3 L2） | `error` | `input_missing` | NULL |
| 3 | 进程无法创建（§6.1 ②） | `error` | `spawn_error` | NULL |
| 4 | 超时 | `error` | `timeout` | NULL |
| 5 | 主进程退出码 0 | `pass` | NULL | 0 |
| 6 | 其他（非零退出、信号） | `fail` | `nonzero_exit` | 实际退出码；信号见 D12 |

- 第 1 条优先：data-model §7.4 第 4 步无条件地把上下文变化记为 `error` / `context_changed`，它也使这次执行不合格。
- `not_found` 在 MVP 中**不产生**（推荐的方案 A，§6.3）。
- 一致性（迁移兜底，§10；`check-run.v1` 的约束见 D14）：`pass` ⇔ `failure_kind` 为 NULL，`pass` 时 `exit_code = 0`；`fail` ⇔ `nonzero_exit`；`error` ⇔ `timeout`、`spawn_error`、`input_missing`、`context_changed` 之一。`exit_code = 0` 不推出 `pass`：上下文变化的执行可以以 0 退出。

### 6.3 EV-04 的契约冲突（§17 D5）

- **现有契约**：MVP-EV-04“检查命令不存在 → `error` / `not_found`”。
- **冲突**：检查命令都经由 shell 执行，“命令不存在”只出现在第 ③ 层，而第 ③ 层没有可靠证据；能可靠识别的“不存在”只有第 ① 层（解释器），而第 ① 层推荐为退出码 6、不写记录。所以按推荐的分类，经 shell 执行的检查不会产生 `not_found`，EV-04 无法满足。

| 方案 | 内容 | 代价 | 验收的变化 |
|---|---|---|---|
| **A（推荐）** | 经 shell 执行的检查不报告 `not_found`；命令不存在时为 `fail` / `nonzero_exit`，记录实际退出码；人类可读输出说明 Specmetry 不判断失败的原因。`not_found` 保留在枚举中（不改 `check-run.v1`、`status.v1` 的取值），留给将来可能由 Specmetry 直接启动程序的执行方式 | “命令不存在”与“测试失败”在记录中看起来一样；人需要看转发的输出 | 改写 MVP-EV-04（见 §14 的草案）；data-model §4.11 的 `failure_kind` 说明；`check-run.v1`、`status.v1` 的描述；交付状态不受影响（`fail` 与 `error` 在推导中都是失败） |
| A' | A 的变体：第 ① 层记为 `error` / `not_found`，不再是退出码 6 | 把环境问题写成检查结果：一次解释器缺失就会让已验收的 AC 变为 `failing`（`regressed`） | 改写 EV-04 为“解释器不存在” |
| B | 在能识别的 shell 上注入前置代码报告“未找到”：bash 的 `command_not_found_handle`、PowerShell 的 `CommandNotFoundAction` | 实验：dash（Linux 常见的 `/bin/sh`）没有等价机制，定义了处理函数仍返回 127；bash 有效；PowerShell 的机制未验证。`cmd` 没有已知机制。注入的代码改变了经人确认的检查的执行环境 | EV-04 只对部分 shell 成立，同样要改写 |
| C | 增加不经 shell 的执行形式（例如 `argv` 数组），操作系统能可靠报告程序不存在 | 修改 `checkSpec`、审核、人工编辑与 Skill 指南；较大的契约变化 | EV-04 只对 argv 形式成立 |

不采用：按退出码 127 或 9009 推断（实验：`exit 127` 同样得到 127；`cmd` 的“未找到”是 1，`exit /b 9009` 才是 9009）；按 stderr 文本推断。

## 7. 观测级执行：上下文、输入与合格判定

### 7.1 代码上下文

| 值 | 取得方式 | 非 git 项目 |
|---|---|---|
| `W`、`worktree_id` | `gitx.Locate`；按 data-model §4.1 查找或登记 | 项目根对应的 worktree 记录（与 `claim`、`propose` 相同） |
| `head_start`、`head_end` | `gitx.CommitOf(HEAD)`，完整的 40 位 ID；没有提交时为 NULL | NULL |
| `tree_state_start`、`tree_state_end` | `gitx.Clean`：`clean`、`dirty`；`undecidable`（P1）记为 `unknown` | `unknown` |
| `workdir` | 检查配置中保存的值（V-P04 已保证不越出仓库根） | 同左 |

- 开始上下文在启动进程之前读取；结束上下文在**进程树全部结束之后**读取（§4.2 第 4、5 步），使仍在运行的后代不能在读取之后再改动 worktree（§17 D11）。
- 开始上下文读取失败（`gitx.Clean` 或 `CommitOf` 返回实际的环境错误）→ 退出码 6，这一次不执行、不记录（与 WP2.4 的 D1 一致）。
- 结束上下文读取失败 → §17 D6。
- `.specmetry/` 不影响 `tree_state`（WP2.3 已实现，CX-24 的检查部分）。
- `workdir` 是保存的配置值；它经 worktree 中的符号链接或联接指向仓库之外时，Specmetry 不阻止（检查本来可以访问任何位置，这不是安全边界），记录的仍是配置值。工作目录不存在或不是目录 → `spawn_error`（§6.1 第 ② 层）。

### 7.2 上下文变化（data-model §7.4 第 4 步，已冻结）

| 开始 | 结束 | 记为 `context_changed` | 合格 |
|---|---|---|---|
| `head_start = h`，`clean` | `head_end = h`，`clean` | 否 | 是 |
| `clean` | `dirty` 或 `unknown` | **是**（“结束时不再是 `clean`”） | 否 |
| 任一 | `head_end ≠ head_start`（含一方为 NULL） | **是**（“HEAD 发生了变化”） | 否 |
| `dirty` 或 `unknown` | 同一 HEAD，任一状态 | 否 | 否（MVP-EV-08：开始不是 `clean`） |

- `clean` → `unknown` 按冻结条文的字面含义记为 `context_changed`：它不证明发生了变化，只是无法证明没有变化；结果同样不合格。
- 中途变化后又恢复（CX-26、CX-27）观测级无法发现，这是已声明的边界（data-model §4.11.1）。

### 7.3 输入记录（M2-Q2、TD-23，已冻结，不重开）

- 读取：对 `head_start`，按 WP2.3 规范 §4.1 逐个调用 `gitx.TreeEntry`（每个路径一次 `--literal-pathspecs ls-tree -z --full-tree`；仓库根用 `rev-parse --verify <head_start>^{tree}`）。
- **只读取路径的元数据**：不读取叶对象（文件的 blob、目录的树、子模块的提交）。Claim 与 `propose` 的文件引用在 Review 068 之后对 blob 做完整校验（`gitx.FileLines`），**这一要求不适用于 `inputs`**：两者回答的问题不同（存在性与行数、对比快照身份），Review 070 也明确不要求 `TreeEntry` 因此读取 blob。
- 层次与结果（data-model §4.11.2）：
  - L1 读取路径所需的对象失败、PD-26 → 退出码 6，**不执行、不记录**；
  - L2 路径不存在（含大小写不同、子模块内部、`head_start` 为 NULL）→ `error` / `input_missing`，**不启动进程，但写入记录**；
  - L3 叶对象缺失 → 照常记录元数据；
  - L4 gitlink → 照常记录元数据。
- `inputs_json`：按保存的路径原值排序的 `{path, mode, type, oid}` 数组；没有声明 `inputs` 或非 git 项目时为 NULL。保存的值与 `check_config_hash` 不变。
- `input_missing` 的记录照常读取结束上下文：开始与结束一致且干净时，它是合格执行，计为失败（data-model §6.9“计为失败”要求它参与 `V`）。

### 7.4 合格判定（data-model §4.11.1，已冻结）

`qualifying = head_start = head_end ≠ NULL` 且 `tree_state_start = tree_state_end = clean` 且 `failure_kind ≠ context_changed` 且（MVP 中）`evidence_grade = observed`。

- 与推导引擎的 `qualifying()`（`internal/domain/derive`）是同一公式；输出中的 `qualifying` 由写入的行按这一公式计算，不另行判断。
- `timeout`、`spawn_error`、`input_missing`、`nonzero_exit` 的执行可以是合格的（上下文一致时），它们都使 `V = fail`。

### 7.5 不执行 / 不记录与执行之后不合格

| 情形 | 启动进程 | 写 `check_runs` | 分配 `seq` | 退出码 |
|---|---|---|---|---|
| 命令行、选择或静态规则错误 | 否 | 否 | 否 | 2 |
| `--isolation required` | 否 | 否 | 否 | 6 |
| 解释器不可用 | 否 | 否 | 否 | 6 |
| PD-26、私有临时区或 `S` 的读取失败（第一次执行之前） | 否 | 否 | 否 | 6 |
| 第 k 次的开始上下文或 L1 输入读取失败 | 否（第 k 次） | 否（第 k 次；之前的保留） | 否（第 k 次） | 6 |
| 加入 Job 失败（Windows） | 已创建、从未运行，随即终止 | 否（第 k 次） | 否 | 6 |
| 输入路径不存在（L2） | 否 | 是：`error` / `input_missing` | 是 | 10 |
| 进程无法创建 | 否（操作系统拒绝） | 是：`error` / `spawn_error` | 是 | 10 |
| 开始时 `dirty` / `unknown` | 是 | 是，不合格 | 是 | 视结果 |
| 上下文变化 | 是 | 是：`error` / `context_changed`，不合格 | 是 | 10 |
| 超时 | 是（被终止） | 是：`error` / `timeout` | 是 | 10 |
| 结束上下文读取失败 | 是 | §17 D6 | | |
| 取消 | 是（被终止） | §17 D8 | | |
| 执行之后写入失败 | 是 | 否（写入失败） | 否 | §11.4 |
| 提交之后输出失败 | 是 | 是 | 是 | 6（§11.4） |

## 8. 进程树、超时与取消

### 8.1 共同规则

- 每次执行一个独立的进程组（Unix）或 Job（Windows），只包含这一次检查的进程。
- 终止的时机：超时、取消，以及主进程退出之后仍有剩余进程（§17 D11）。终止之后等待所有成员结束，再读取结束上下文。
- 标准输入为空设备（`/dev/null`、`NUL`）：检查是非交互的；等待输入的检查读到 EOF。直接打开终端（`/dev/tty`）的检查在 Unix 上处于后台进程组，读取时被 `SIGTTIN` 停止，直到超时被终止（如实说明，工程细化 E8）。

### 8.2 机制与实验

**Unix**（Linux 有实验，macOS 未验证）：

- `Setpgid`：Go 在 fork 之后、exec 之前于子进程中调用 `setpgid`，所以检查的任何代码都在新的进程组中运行，没有“启动之后、加入之前”的窗口。
- 终止：`kill(-pgid, SIGKILL)`。实验（§16 X1）：后台、双重 fork（`(sleep &)`）与嵌套 `sh` 的 6 个成员全部终止。
- **边界**（实验，§16 X2）：
  1. `setsid`（或 `setpgid`）主动离开进程组的后代，不受 `kill(-pgid)` 影响：实验中它在组被终止之后仍存活。
  2. 创建进程组的进程（Specmetry）被 SIGKILL 时，组内进程全部存活；`Pdeathsig` 只让直接子进程收到信号，孙进程仍存活。
- **Linux 的加强**（推荐，§17 D7）：Specmetry 在执行检查之前把自己设为 child subreaper（`prctl(PR_SET_CHILD_SUBREAPER)`）。实验（§16 X2）：组被终止之后，`setsid` 离开的子树回到 Specmetry 名下，按 `/proc` 清扫 Specmetry 的全部后代（逐个 `SIGKILL`，只回收这些已知的 PID，不调用 `wait(-1)`，以免与 `os/exec` 的 `Wait` 冲突）之后，没有存活的进程。它不能覆盖不是后代的进程（由 systemd、`at`、D-Bus 等启动的），也不能在 Specmetry 本身被 SIGKILL 时生效。设置是整个进程的：Specmetry 自己运行的 git 留下的孤儿同样会回到 Specmetry 名下；检查执行时 Specmetry 没有其他子进程，清扫只针对这一次检查开始之后出现的后代，命令结束时其余的由系统回收（未验证的细节在实现中测试）。
- macOS 没有 subreaper：边界 1、2 都保留（§17 D7）。

**Windows**（实验，§16 X1、X2）：

- Job Object，设 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`，**不**设 `BREAKAWAY_OK` 或 `SILENT_BREAKAWAY_OK`。
- 顺序：以 `CREATE_SUSPENDED` 创建 → `AssignProcessToJobObject` → 用 `IsProcessInJob` 确认 → 恢复主线程。Go 的 `syscall.StartProcess` 会关闭主线程的句柄，所以经线程快照（`CreateToolhelp32Snapshot`）找到并恢复它（`golang.org/x/sys/windows`，已有依赖）。检查的任何代码都在加入 Job 之后才运行。
- 终止：`TerminateJobObject`。实验：Git for Windows `sh` 的后台、双重 fork 与嵌套成员（6 个；经包装程序 `bin\sh.exe` 时 7 个）全部终止。
- 持有 Job 的进程直接退出（不做清理）时，`KILL_ON_JOB_CLOSE` 终止了全部后代（实验）。
- Job 中的进程以 `CREATE_BREAKAWAY_FROM_JOB` 创建子进程 → 拒绝（errno 5）；普通创建的子进程在 Job 中（实验）。
- **边界**：不是由检查的进程树创建的进程不在 Job 中，例如经服务、计划任务、WMI 或 COM 的进程外服务器启动的进程，以及检查连接到的已在运行的守护进程（§17 D7）。

### 8.3 超时

- 计时从进程恢复运行（Windows）或启动（Unix）开始，到主进程退出为止；超过 `timeout_sec` → 终止整个进程树，`error` / `timeout`，`exit_code` 为 NULL（MVP-EV-03）。
- `duration_ms` 用单调时钟，计时范围同上；没有启动进程（`input_missing`、`spawn_error`）时为 0（工程细化 E9）。

### 8.4 取消（§17 D8）

- **Unix**：终端的 Ctrl+C 把 SIGINT 发给前台进程组，即 Specmetry 所在的组；检查在自己的组中，不直接收到。Specmetry 捕获 SIGINT、SIGTERM（也包括上层工具超时发出的 SIGTERM），终止检查的进程树，等待结束。SIGKILL 无法捕获，见 §8.2 的边界 2。
- **Windows**：控制台的 Ctrl+C 发给附着在同一控制台上的所有进程，检查也会收到，可能自行退出；Specmetry 捕获控制台事件，终止 Job。Specmetry 被强制结束时，`KILL_ON_JOB_CLOSE` 终止检查。
- 推荐（D8）：被取消的这一次**不记录**，之后的不再执行，之前已写入的保留，退出码 3（与人工确认中“取消”的既有约定一致，例如 WP2.4 §4）。
- 竞争：检查在取消的同一时刻自行退出时，Specmetry 可能先看到它的退出状态。推荐：主进程退出之后、写入之前再检查一次取消标记；已取消的不记录。信号到达 Go 程序的通道有延迟，不能完全排除“取消之前已经写入”的情况，此时记录的是检查自己的退出状态（如实说明）。

## 9. 输出与隐私

### 9.1 检查输出的转发

- **推荐**（工程细化 E2）：检查**直接继承** Specmetry 的输出句柄，Specmetry 不建立管道、不复制、不读取检查的输出：
  - 不带 `--json`：检查的 stdout、stderr 分别是 Specmetry 的 stdout、stderr；
  - 带 `--json`：检查的 stdout 与 stderr **都是** Specmetry 的 stderr；Specmetry 的 stdout 只写最后的一份 `check-run.v1`（FR-EV-01，MVP-EV-12）。
- 理由（实验，§16 X7）：经管道复制时，退出的检查留下的后台进程持有管道，`Wait` 一直阻塞到那个进程结束；`WaitDelay` 只能在超时之后返回错误。直接继承句柄时，主进程退出即返回。同时，Specmetry 从不读取输出，也就不可能保存它（隐私契约，architecture §10）。
- 句柄继承让检查看到的终端属性与调用者相同（例如 stdout 是终端时，测试工具的彩色输出照常）。
- CLI 的输出不是操作系统文件时（只在进程内测试中），只能经管道转发：数据只在复制时经过内存，不保留、不检查内容。按 §17 D11 在主进程退出之后终止剩余的进程树，管道随之关闭，`Wait` 不会被留下的后台进程拖住。
- **显示的信任边界**：检查的输出原样出现在终端中（data-model §7.4“原样转发”），可以含有控制字符，能改变终端或模仿 Specmetry 的文本行。Specmetry 自己的文本仍经 `termtext` 转义（architecture §9.1），但文本输出不是可信的通道；以 `--json` 的结果与数据库为准。
- 调用者把 stderr 合并到 stdout（`2>&1`）时，`--json` 的 stdout 不再只有 JSON；这是调用者的选择，文档说明。

### 9.2 环境

- 继承调用者的环境，加上 `SPECMETRY_CHECK=1`（已有同名变量时覆盖）；Windows 的 `sh` / `bash` 另见 §5.4（D2）。
- Specmetry 不读取、不记录、不输出任何环境变量的值。
- `internal/gitx` 的受控环境（清除 `GIT_DIR` 等）只用于 Specmetry 自己运行的 git，**不**施加给检查：检查继承用户的环境，包括其中的 `GIT_*` 变量（如实说明）。

### 9.3 隐私

- **不保存**：命令文本（`check_runs` 没有命令列，命令在基准中，由 `check_config_hash` 找到）、检查输出及其尾部与哈希、环境变量的值、启动错误的文本与路径。
- **保存**：data-model §4.11 的字段；`workdir` 来自已确认并经过密钥扫描的检查配置；首次登记的 worktree 路径按 `claim` 的做法先做密钥扫描（命中即退出码 2，不执行任何检查）。
- **日志**：不记录 C0 / C1 内容。
- **对仓库的写入**：Specmetry 不写入（architecture §10.6）；检查自己对 worktree 的写入属于检查的行为，按 §7.2 判定。
- 验收：MVP-EV-05、MVP-EV-12 的 canary（stdout、stderr、数据库、WAL、日志、备份、数据目录中的其他文件）。

## 10. 持久化：迁移 `0009_check_runs`

### 10.1 表

字段来自 data-model §4.11；约束是第二道防线，写法沿用 0008。

| 列 | 类型与约束 | 说明 |
|---|---|---|
| `id` | TEXT PRIMARY KEY，ULID（26） | 即 `evidence_id` |
| `seq` | INTEGER NOT NULL **UNIQUE** | 每次执行一个，表内唯一（F4） |
| `project_id` | TEXT NOT NULL，引用 `projects` | |
| `criterion_id` | TEXT NOT NULL，`(criterion_id, project_id)` 引用 `acceptance_criteria (id, project_id)` | 与 0008 中 `claims` 的写法相同 |
| `criterion_revision`、`requirement_revision` | INTEGER NOT NULL ≥ 1 | 开始时读取的值 |
| `check_config_hash` | TEXT NOT NULL，64 位小写十六进制 | |
| `inputs_json` | TEXT，NULL 或 JSON 数组 | §7.3 |
| `worktree_id` | TEXT NOT NULL，`(worktree_id, project_id)` 引用 `worktrees (id, project_id)` | 同上 |
| `workdir` | TEXT NOT NULL | 保存的配置值 |
| `evidence_grade` | TEXT NOT NULL CHECK = `observed` | MVP 的第二道防线（不变量 9）；M7 迁移放开 |
| `isolation_mode` | TEXT NOT NULL CHECK = `off` | `required` 从不写入；`auto` 不在 MVP 中；M7 迁移放开 |
| `isolation_backend`、`isolation_fallback_reason` | TEXT，CHECK 为 NULL | 只用于隔离执行器 |
| `head_start`、`head_end` | TEXT，NULL 或 40 位小写十六进制 | |
| `tree_state_start`、`tree_state_end` | TEXT NOT NULL ∈ {`clean`, `dirty`, `unknown`} | |
| `result` | TEXT NOT NULL ∈ {`pass`, `fail`, `error`} | |
| `failure_kind` | TEXT，NULL 或枚举 | 方案 A 获批时，CHECK 排除 `not_found`（D5） |
| `exit_code` | INTEGER，可为 NULL | |
| `duration_ms` | INTEGER NOT NULL ≥ 0 | |
| `triggered_channel` | TEXT NOT NULL ∈ {`human_cli`, `agent_cli`} | |
| `session_id` | TEXT，M3 之前为 NULL | 与 `claim_batches` 相同，不加外键（会话表在 M3） |
| `started_at`、`finished_at` | INTEGER NOT NULL | 只用于展示；表示方式与 0007 的 `scanned_at` 相同 |

**CHECK 约束**（§6.2 的一致性）：

- `result = 'pass'` ⇔ `failure_kind IS NULL`；`pass` 时 `exit_code = 0`；
- `result = 'fail'` ⇔ `failure_kind = 'nonzero_exit'`；此时 `exit_code` 不为 0（信号见 D12）；
- `result = 'error'` ⇔ `failure_kind` ∈ {`timeout`, `spawn_error`, `input_missing`, `context_changed`}（D5 方案 A 获批时不含 `not_found`）；`timeout`、`spawn_error`、`input_missing` 时 `exit_code IS NULL`；
- `input_missing`、`spawn_error` 时 `duration_ms = 0`。

**只追加**：UPDATE、DELETE 触发器（`maintenance.active = 0` 时中止），与 0008 相同；`REPLACE` 与 UPSERT 由主键与 `UNIQUE` 冲突和触发器共同拒绝（MVP-EV-07）。`purge` 按既有的维护机制处理。

**索引**：`(project_id, criterion_id, seq)`，供推导按 AC 取执行。

### 10.2 与推导输入的对应

`internal/domain/derive` 的 `CheckRun` 与测试中的 `factgen.CheckRunRow` 已经定义了推导需要的字段：`id`、`seq`、`criterion_id`、`criterion_revision`、`requirement_revision`、`check_config_hash`、`inputs_json`、`worktree_id`、`evidence_grade`、`head_start`、`head_end`、`tree_state_start`、`tree_state_end`、`result`、`failure_kind`。迁移的列名与它们一一对应；`dbdump` 在实现时补上这张表的映射（它目前把还不存在的 `check_runs` 视为没有行）。

### 10.3 `seq` 与事务

- 每次执行在结束之后**单独**写入一个事务，分配一个 `seq`（data-model §4.11）。`--all` 中的执行按结束的顺序得到递增的 `seq`。
- worktree 首次出现时，在第一次写入的事务中先登记（`project.EnsureWorktree`，另取一个 `seq`），再写执行。没有任何执行要写时（`--all` 选不到 AC），输出中的 `worktree_id` 见工程细化 E3 与 §17 D18。
- 写入**没有版本前置条件**：执行绑定开始时读取的修订与配置（CX-15），之后的基准变化不拒绝这次写入。
- 写事务沿用 `DB.WriteTx`（`BEGIN IMMEDIATE`，`busy_timeout`）。M-15（`SQLITE_BUSY`）是未解决的维护事项；本工作包不改变既有的重试与错误映射。

## 11. 输出：`check-run.v1`

### 11.1 字段的来源

| 字段 | 来源 |
|---|---|
| `schema` | `specmetry.check-run/v1` |
| `worktree_id` | `W` 的 worktree 记录 |
| `is_target_snapshot` | §11.2 |
| `isolation` | `off`（`required` 不产生 `check-run.v1`） |
| `runs[]` | 写入的行，按执行的顺序 |
| `runs[].criterion` | AC 的编号 |
| `runs[].criterion_revision` | 行的 `criterion_revision` |
| `runs[].evidence_id`、`seq` | 行的 `id`、`seq` |
| `runs[].result`、`failure_kind`、`exit_code`、`duration_ms` | 行的同名列 |
| `runs[].head_start`、`tree_state_start`、`evidence_grade` | 行的同名列 |
| `runs[].qualifying` | 由行按 §7.4 的公式计算 |
| `runs[].isolation_fallback_reason` | 不输出（只用于 `auto`） |

### 11.2 `is_target_snapshot`（§17 D9）

- 现有描述：“开始时 worktree 的 HEAD 等于交付目标的快照时为 true”。一次命令可以有多次执行，后面的执行可能在不同的 HEAD 上开始（前一次检查切换了 HEAD 且没有切回，那一次本身是 `context_changed`）。
- 推荐：`S` 在命令开始时解析一次；`is_target_snapshot` 为 true，当且仅当 `S` 不为 NULL，并且**每一次**执行的 `head_start` 都等于 `S`；没有执行时，比较命令开始时的 HEAD。只有这时，这些执行中的合格执行才能影响交付状态。需要修改 `check-run.v1` 中这个字段的描述（不改结构）。

### 11.3 JSON Schema 能表达的与必须由业务测试验证的

| 条件 | 由谁保证 |
|---|---|
| 字段齐全、没有多余字段（因而不会出现输出或命令）；枚举；格式 | schema（已有；反例 SCH-49、SCH-59） |
| `result`、`failure_kind`、`exit_code` 一致（§6.2） | 现在只有业务测试；**提议**在 `check-run.v1` 中加入 `if/then` 约束（准备报告 §5.11“输出的语义约束”；§17 D14） |
| `evidence_grade = observed`（MVP） | 业务测试与迁移约束 |
| `qualifying` 与行的上下文一致 | 业务测试 |
| `runs` 与写入的行逐项一致，`seq` 递增，`evidence_id` 互不相同 | 业务测试 |
| 条数等于所选 AC 的数目（正常完成时） | 业务测试 |
| 输出（含错误）中没有检查输出、命令文本或环境变量的值（canary） | 业务测试 |

**提交之前校验**：每次执行的那一项在写事务中用刚写入的行构造，并在提交之前按 `check-run.v1` 中执行项的定义校验；失败 → 回滚（包括这个事务中首次的 worktree 登记），退出码 1，这一次没有记录，停止，不执行之后的（检查已经执行，不重新执行）。全部执行之后再校验整份文档；由已校验的各项构成，它失败只可能是内部错误（退出码 1，执行已经写入）。这比 `propose` 的现有顺序更严格，与 WP2.4 §9.3 的做法相同。

**缺口**（不在本工作包中改变，§17 D14）：`check-run.v1` 没有 `head_end`、`tree_state_end`、`requirement_revision`、`check_config_hash`、`workdir`；调用方只能从 `failure_kind`、`tree_state_start` 与 `qualifying` 看出不合格的原因。推荐保持现状（MVP 的 Agent 需要的是结果与是否合格），需要时另行提出。

### 11.4 失败的呈现

| 阶段 | 已经写入的 | 呈现 |
|---|---|---|
| 执行任何检查之前（§4.1） | 无 | 带 `--json` 时 stdout 一份 `error.v1`，stderr 为空；不带时为文本 |
| 第 k 次之前的环境错误（开始上下文、L1、加入 Job） | 第 1 ~ k−1 次 | §17 D10 |
| 第 k 次执行之后写入失败 | 第 1 ~ k−1 次；第 k 次**已执行、没有记录** | 停止，不执行后面的；退出码 6；§17 D10；**不重新执行第 k 次** |
| 第 k 次被取消 | 第 1 ~ k−1 次 | §17 D8 |
| 全部提交之后，写出失败或短写 | 全部 | 退出码 6；不回滚；stdout 可能被截断，不再追加第二份 JSON；尽力在 stderr 说明“执行已经记录，不要盲目重试”（与 WP2.4 §9.3 第 4 步相同） |

### 11.5 文本输出（工程细化 E4）

- 每次执行之前，一行 `checking AC-001-1 (sh, timeout 300 s)`；之后一行结果：`AC-001-1 pass (8.4 s)`、`AC-001-2 fail (exit 1)`、`AC-003-1 error: timeout`，以及不合格的原因（`not qualifying: the worktree was dirty at the start` 等）。都写到 stdout，经 `termtext` 转义。
- 最后一行汇总：通过、失败、错误的数目，`is_target_snapshot`，证据等级 `observed`，以及“observed: only the start and the end were checked”（不称为“快照已验证”，CX-26）。
- `fail` / `nonzero_exit` 的说明写明 Specmetry 不判断失败的原因（方案 A）。

## 12. 与推导的关系（不改变）

- 推导读取 §10.2 的字段；`runs(S)`、`V`、`inconsistent`、`HIST` 的定义不变（data-model §6.1）。
- 观测级证据只能把 AC 推到 `ready_for_review`，或支撑人在同一快照上的直接验收；MVP 中沿用不会发生（PD-22，MVP-ACC-09 的集成部分）。
- `timeout`、`spawn_error`、`input_missing` 与 `nonzero_exit` 一样使 `V = fail`；`context_changed` 与不干净的执行不参与 `V`。
- 推导引擎与复核器不改；MVP 合法事实域只含观测级执行（data-model §6.14）。

## 13. 验收映射

| 验收项 | WP2.5 验证的部分 | 由其他工作包完成的部分 |
|---|---|---|
| MVP-EV-01 | 通过的检查：全部字段、`qualifying = true`；数据库中没有命令文本与输出 | — |
| MVP-EV-02 | 失败的检查：`fail` / `nonzero_exit` 与实际退出码；退出码 10 | — |
| MVP-EV-03 | 2 s 超时、长时间睡眠的后代：`error` / `timeout`，之后整个进程树都不存在（含 Git Bash 的后代、双重 fork；边界见 §8.2） | — |
| MVP-EV-04 | **按 D5 的决定改写**（§14） | — |
| MVP-EV-05、MVP-EV-12 | canary 与输出流 | — |
| MVP-EV-06 | 不接受临时命令；没有检查的 AC → 退出码 2 | — |
| MVP-EV-07 | `check_runs` 的只追加 | 其他事实表（各自的迁移） |
| MVP-EV-08、MVP-EV-09、CX-16 | 未跟踪文件 → `dirty`、不合格；写入未忽略的文件、`git commit` → `context_changed` | — |
| MVP-EV-10、MVP-EV-17 | 输入记录在执行中的使用：L1 → 退出码 6、不执行、没有新行；L2 → `input_missing`、不启动进程；L3、L4 照常记录 | 读取机制本身（WP2.3 已验收） |
| MVP-EV-13、CX-28(a) | `--isolation required` → 退出码 6、`isolation_unavailable`、没有新行、没有启动进程 | M7（b） |
| MVP-WT-02、CX-04 | 在 feature worktree 中执行：`pass`、`worktree_id` 为 feature worktree；主 worktree 中 `fail` | — |
| MVP-WT-03 | 执行记录有 `worktree_id` 与完整的代码上下文 | 会话与事件（M3） |
| MVP-INV-03 | 4 个带检查的 AC → `--all` → 恰好 4 条，都属于当前 worktree；任一不是 `pass` → 退出码 10 | — |
| CX-02 | 有未提交修改时执行并失败：不合格（推导部分由 WP2.1 已验证；决定被拒绝的部分在 WP2.6） | WP2.6、WP2.7 |
| CX-15 | 执行期间 AC 被修订：记录旧的 `criterion_revision`，写入不被拒绝 | 推导部分已验收 |
| CX-24（检查部分） | 未跟踪的 `.specmetry/project.json` 不影响 `tree_state`，执行合格 | `setup` 部分（M3） |
| CX-26、CX-27（执行部分） | 改写后恢复、切换 HEAD 后切回：`observed`、合格；输出不称为“快照已验证” | 验收与状态（WP2.6、WP2.7） |
| MVP-ACC-09（集成部分） | 在 B 上执行：`ready_for_review`（`observed_evidence`），没有人参与时不变为 `accepted` | 与 WP2.6、WP2.7 的集成闭环 |
| 不变量 4、6、7、9（acceptance-criteria §3） | canary；输出通过 schema；`seq` 严格递增；只有 `observed` | — |

**建议新增的验收项**（草案，编号待定，获批后写入 acceptance-criteria）：

1. **解释器与参数**（D1、D2）：每个平台上每种可用的解释器，用 §5.5 的字符串集回显，结果与输入完全一致；Windows 上 PATH 中只有 WSL 的 `bash.exe` 时不使用它；位于 worktree 之内的解释器（经 PATH 的绝对路径、符号链接或联接到达的）不被执行。
2. **解释器不可用**：退出码 6，`shell_unavailable`，没有新行，没有启动任何进程；Unix 上的 `cmd`、`powershell` 同样。
3. **进程无法创建**：工作目录不存在 → `error` / `spawn_error`。
4. **`cmd` 多行与 CR**（D3、D4）：提案与审核中拒绝；已保存的在执行前拒绝，没有新行。
5. **进程树**：挂起创建与加入 Job 的顺序；Job 不允许 breakaway；持有 Job 的进程意外退出时后代被终止（Windows）；Linux 的 subreaper 清扫（D7）。
6. **取消**（D8）、**结束上下文读取失败**（D6）、**写入失败不重新执行**（D10）、**剩余后代被终止之后才读取结束上下文**（D11）。

## 14. EV-04 改写的草案（方案 A，待 D5）

> MVP-EV-04（改写草案）：检查中的命令不存在（经 shell 执行）→ 结果不是 `pass`，为 `fail` / `nonzero_exit`，记录实际退出码，`failure_kind` 不是 `not_found`；`exit 127`（sh、bash）、`exit 1`（cmd、pwsh、powershell）等主动退出得到同样的分类；人类可读输出说明 Specmetry 不判断失败的原因。解释器本身不可用时，见“解释器不可用”（退出码 6，不记录）。

## 15. 测试计划

**单元与库层**：

- 各解释器的命令行构造：MSYS 规则（每个 `\` 双写、`"` 转义）、MSVCRT 规则（总是加引号）、`cmd` 的原样；§5.5 的字符串集，外加空字符串以外的边界（末尾反斜杠、只有引号）。
- 解释器的查找：绝对路径、规范化、worktree 排除（符号链接、联接、短文件名）、WindowsApps 排除、`git --exec-path` 的解析（`ucrt64`、`mingw64`）、找不到时的结果。
- 分类：§6.2 的表；§7.2 的上下文变化表；`qualifying` 的公式与推导引擎相同。
- 输出项与 `check-run.v1`：合法示例与反例；一致性条件（D14 获批时在 schema 中）。

**集成**（真实的 git 仓库、数据库与子进程；进程内执行 CLI）：

- §7.5 总表的每一行：退出码，以及与比较基点相比的新行与 `seq`（比较全部事实表）。
- 真实子进程：通过、失败、超时（含后代、双重 fork、Git Bash）、工作目录不存在、输入缺失（用标记文件证明命令没有运行）。
- 上下文变化：检查写入未忽略的文件、`git commit`、切换 HEAD 不切回；改写后恢复（CX-26）、切换后切回（CX-27）记为合格的观测级。
- PD-26：不安全的 git（`SPECMETRY_TEST_GIT_OBJECTS=unsafe`）→ 退出码 6，没有新行，检查没有运行（标记文件）。
- 事务：注入写入失败 → 退出码 6，没有这一次的行，标记文件表明命令只运行了一次；注入执行项校验失败 → 退出码 1，回滚。
- 并发：两个进程同时执行，`seq` 唯一，各自的行完整。
- 取消：向 CLI 进程发送信号或控制台事件，进程树不存在，按 D8 的结果断言。
- 隐私：stdout、stderr 中的 canary 只出现在转发中；数据库、WAL、日志与数据目录中没有。

**终端与重定向**（进程级）：

- stdout 是终端时检查看到终端（继承句柄）；stdout 重定向到文件、管道时同样只转发。
- `--json`：stdout 恰好一份 JSON；检查的 stdout 与 stderr 都在 stderr（MVP-EV-12）。
- 伪终端用例按现有基座在支持的平台运行，跳过的平台如实记录。

**平台**：三个平台的 CI。CI 必须明确准备并验证所需的解释器（三个平台的 `sh`、`bash`、`pwsh`，Windows 的 `cmd`、`powershell` 与 Git for Windows 的 `sh`）；缺失时测试失败，不静默跳过。macOS 的进程组行为、Windows 的不同 Git 安装方式只有 CI 或人工环境能验证，本轮没有证据。

## 16. 实验记录（本轮，作者）

**方法**：一个独立的 Go 程序（只用标准库与本地模块缓存中的 `golang.org/x/sys` v0.49.0，`GOPROXY=off`），放在会话的临时目录中，不在仓库中，没有修改生产代码。每个子进程都有硬性时限（`sleep 300` 在数秒内被终止），每项实验结束时检查它见过的全部 PID，存活的强制终止并报告。从临时目录中的夹具运行，不运行用户仓库配置的任何检查。Linux 版交叉编译后，在 WSL 中以脚本文件和绝对路径启动（Review 070 记录的内联参数问题）。

**环境**：Windows 11 Pro 10.0.26300，Go 1.27.1，Git for Windows 2.56.0.windows.2（UCRT64，bash 5.3.15），pwsh 7.6.7，Windows PowerShell 5.1.26100；WSL2 Ubuntu 24.04.2（内核 6.18.33.2），`/bin/sh` 为 dash，bash 5.2.21，普通用户（UID 1000）。**macOS 没有实验。**

| # | 实验 | Windows | Linux（WSL） |
|---|---|---|---|
| X1 | 终止整个进程树：后台、双重 fork、嵌套 `sh` | Job：6 个成员全部终止（经 `bin\sh.exe` 时 7 个）；主进程的 `Wait` 立即返回 | 进程组：6 个成员全部终止 |
| X2 | 边界：持有者意外退出；主动脱离 | 持有 Job 的进程直接退出 → 全部终止（`KILL_ON_JOB_CLOSE`）；`CREATE_BREAKAWAY_FROM_JOB` → errno 5；普通子进程在 Job 中 | 持有者被 SIGKILL → 组内全部存活；`Pdeathsig` → 只有直接子进程退出；`setsid` 的进程离开组，组被终止后存活；**subreaper**：组被终止后，`setsid` 的子树回到 Specmetry 名下，清扫之后没有存活 |
| X3 | 退出码与无法创建的进程 | `sh` 未找到 127 = `exit 127`；`cmd` 未找到 1 = `exit /b 1`（`exit /b 9009` 为 9009）；`pwsh`、`powershell` 未找到 1 = `exit 1`；`sh` 被 `kill -9 $$` → 2304；工作目录不存在 / 是文件 → errno 267；解释器不存在 → 不存在；不是程序 → errno 216；都在 `Start` 时失败，进程不存在 | dash、bash 未找到 127 = `exit 127`；dash 定义 `command_not_found_handle` 之后仍为 127，bash 为处理函数的返回值；`kill -9 $$` → 信号 9，`ExitCode()` −1；工作目录不存在 ENOENT、是文件 ENOTDIR、解释器不存在 ENOENT、不是程序 ENOEXEC、没有执行权限 EACCES，都在 `Start` 时失败 |
| X4 | 多行命令与 CR | `cmd`：CRLF 与 LF 都只执行第一行，退出码 0；`sh`：多行正常，CR 被忽略；`powershell`：多行正常 | dash：多行正常，CR 保留为字符（`two\r`） |
| X5 | 命令文本原样到达（23 个字符串，§5.5） | `sh`、`bash`：MSYS 规则 23/23；MSVCRT 规则与 Go 默认 21/23（`\\` → `\`）；包装程序同样；`powershell`、`pwsh`：MSVCRT 规则 23/23 | `sh`、`bash`（argv）23/23 |
| X6 | 解释器的位置与 PATH | `git --exec-path` → `…/Git/ucrt64/libexec/git-core`；`GetSystemDirectory` → `C:\WINDOWS\system32`；系统 PATH 下，`usr\bin\sh.exe` 找不到 `sleep`、`grep`、`sed`、`git`，`bin\sh.exe` 都能找到并另设 `HOME`、`TERM` 等（§5.4） | `/bin/sh` → `/usr/bin/dash`；PATH 上没有 `pwsh` |
| X7 | 输出管道 | 经管道复制：后台进程持有管道，`Wait` 4 s 内没有返回，终止进程树之后才返回；继承文件句柄：立即返回；`WaitDelay` 1 s：返回 `ErrWaitDelay` | 同左 |

- 实验中的一次无效运行如实保留：第一次“MSYS 规则”的对照中，辅助函数因书写错误没有双写反斜杠，结果与 MSVCRT 规则相同；修正之后重跑得到上表的 23/23。
- 这些是作者在两个本地环境中的观察，**不是** CI 证据，也不是独立审查的复验；不能推广到 macOS、其他 Linux 发行版或其他 Git for Windows 版本。本 PR 的 CI 只验证仓库中已有的代码。

**未验证**：macOS 的全部行为；真实控制台中的 Ctrl+C 与控制台关闭；Specmetry 本身被强制结束时 Linux 子树的去向（subreaper 不覆盖）；其他 Git for Windows 版本与安装方式、Cygwin、独立 MSYS2；带 `-race` 的子进程管理；Windows 上检查通过服务或计划任务启动的进程；`/dev/tty` 的 `SIGTTIN` 行为。

## 17. 待决事项

| ID | 事项 | 推荐 | 代价 | 分类 | 最晚决定于 |
|---|---|---|---|---|---|
| **D1** | 默认解释器与不可用时的行为（TD-06 / M2-Q1） | 所有平台默认 `sh`，不回退到 `cmd`；解释器不可用时整批不执行、不记录，退出码 6（`shell_unavailable`） | 没有 Git for Windows `sh` 的机器上，省略 `shell` 的检查不可用 | 新增产品取舍 | WP2.5 实现开工前（M2-Q10） |
| **D2** | Windows 的 `sh` / `bash`：入口、PATH 与来源 | W2：`<Git>\usr\bin\sh.exe`，只在检查的 PATH 前面加 `<Git>\usr\bin`；只取 `gitx` 所用 git 的 Git for Windows，不回退到其他 `sh` | 偏离 architecture §6.4 的环境规则；其他 `sh` 不可用 | 新增产品取舍（新发现） | WP2.5 实现开工前 |
| **D3** | `cmd` 的多行命令 | 不得含 CR / LF：提案、审核与人工编辑拒绝（V-P04 的扩展）；已保存的在执行前拒绝，退出码 2 | 多行批处理要改写 | 新增产品取舍（校验规则变化） | WP2.5 实现开工前 |
| **D4** | `sh` / `bash` 命令中的 CR | 不得含 CR，同 D3 处理 | CRLF 写成的命令要改写 | 新增产品取舍（新发现） | WP2.5 实现开工前 |
| **D5** | EV-04 与 `not_found` | 方案 A：改写 EV-04；不产生 `not_found`，枚举保留；迁移的 CHECK 排除它 | 记录中“命令不存在”与“测试失败”相同 | 新增产品取舍（验收变化） | WP2.5 实现开工前 |
| **D6** | 结束上下文读取失败 | 照常记录：`head_end` 按能读到的值（读不到为 NULL），`tree_state_end = unknown`，`error` / `context_changed`（无法证明没有变化），不合格 | 环境故障留下一条不合格的执行记录 | 新增规则（冻结条文没有覆盖） | WP2.5 实现开工前 |
| **D7** | 进程树的边界 | Linux 用 subreaper 清扫后代；macOS 的 `setsid` 脱离与 Specmetry 被 SIGKILL 后的存活、Windows 中不在 Job 里的进程，作为能力边界如实写明，MVP-EV-03 按此限定 | 只有 Linux 能收回主动脱离的后代；边界需要负责人接受，**本文不把它写成已接受的风险** | 未解决的技术问题 + 产品取舍 | WP2.5 实现开工前 |
| **D8** | 取消（Ctrl+C、SIGTERM、控制台事件） | 被取消的这一次不记录；之后的不执行；已写入的保留；退出码 3 | 取消的那一次在事实中看不到；与自行退出的竞争只能缩小 | 新增产品取舍 | WP2.5 实现开工前 |
| D9 | 多次执行时 `is_target_snapshot` 的含义 | 每一次的 `head_start` 都等于 `S`（没有执行时比较命令开始时的 HEAD）；修改 schema 描述 | 字段含义比现有描述更严格 | 契约澄清 | WP2.5 实现开工前 |
| D10 | 批次中途中止的呈现（环境错误、写入失败） | 停止；`error.v1`（`environment`）的 `message` 说明之前的执行可能已经记录、第 k 次可能已执行而没有记录；`details` 逐项给出已记录执行的 AC 编号与 `evidence_id`；不重新执行 | `error.v1` 的 `details` 用于列举已记录的执行（没有新字段） | 契约细化 | WP2.5 实现开工前 |
| D11 | 主进程退出后仍在运行的后代 | 终止它们，等全部结束之后再读取结束上下文 | 检查有意留下的后台进程（例如服务）会被终止 | 新增产品取舍 | WP2.5 实现开工前 |
| D12 | Unix 上被信号终止时的 `exit_code` | NULL（没有退出状态），`fail` / `nonzero_exit` | 记录看不出是信号还是哪个信号 | 契约细化 | WP2.5 实现开工前 |
| D13 | 配置的固定时刻 | 命令开始时一次读取全部所选 AC 的配置，每次执行绑定这些值 | 前一次执行期间修订的 AC，后面仍按旧配置执行并记为旧修订（只对旧修订有效，保守） | 实施细节（需确认与 data-model §7.4 一致） | WP2.5 实现开工前 |
| D14 | `check-run.v1` 的补充 | 加入 `result` / `failure_kind` / `exit_code` 的 `if/then` 约束与反例；不增加 `head_end` 等字段 | schema 变化（只收紧） | 契约补充 | WP2.5 实现开工前 |
| D15 | 显式参数中重复的 AC | 退出码 2，指向后出现的位置 | 不能“多写一次也无妨” | 实施细节 | WP2.5 实现开工前 |
| D16 | `--isolation auto` | 命令行拒绝（退出码 2），与 prd 的语法一致；schema 的 `auto` 取值留给 M7 | — | 确认既有契约 | WP2.5 实现开工前 |
| D17 | Windows 上 `sh` 被信号终止时的退出码（2304 等） | 记录实际退出码（它是 Windows 的退出码），不换算 | 与 Unix 的表示不同 | 实施细节 | WP2.5 实现开工前 |
| D18 | `--all` 选不到任何 AC、当前 worktree 又尚未登记时，`check-run.v1` 必需的 `worktree_id` | 在一个只做登记的写事务中登记 worktree（与首次执行相同，另取一个 `seq`），然后输出空的 `runs` | 一次没有执行任何检查的命令写入了一条 worktree 事实与一个 `seq`；备选：此时带 `--json` 输出 `error.v1`（`invalid_input`，“没有可执行的检查”），代价是 `--all` 在空项目上不是成功 | 契约细化 | WP2.5 实现开工前 |

**工程细化**（符合既有契约，审查时确认）：

- E1：执行顺序按 AC 编号的数值顺序。
- E2：输出句柄直接继承（§9.1）。
- E3：`--all` 选不到 AC、而当前 worktree 已登记时，输出已有登记的 `worktree_id`；尚未登记时见 D18。
- E4：文本输出的格式（§11.5）。
- E5：加入 Job 失败 → 终止挂起的进程，退出码 6，不记录。
- E6：`spawn_error` 不增加错误类别列。
- E7：不增加“空命令预检”。
- E8：标准输入为空设备；`/dev/tty` 的行为如实说明。
- E9：`duration_ms` 的计时范围；没有启动进程时为 0。

## 18. 获批之后需要同步的文档

| 决定 | 文档 |
|---|---|
| D1 ~ D4 | architecture §6.4、§16（TD-06 的状态）；data-model §8.3（V-P04 的扩展）；`common.v1` 中 `checkSpec.shell` 的描述；acceptance-criteria 的新增项 |
| D5 | acceptance-criteria MVP-EV-04；data-model §4.11（`failure_kind` 的说明）；`check-run.v1`、`status.v1` 的描述 |
| D6、D8、D10、D11 | data-model §7.4；architecture §6.4；acceptance-criteria 的新增项 |
| D7 | architecture §6.4（进程管理的边界）；acceptance-criteria MVP-EV-03 的限定 |
| D9、D14、D16 | `check-run.v1` 的描述与约束；`schemas/examples` 的合法示例与反例 |
| 实现时 | 迁移 `0009_check_runs`；`internal` 的实现与测试；`dbdump` 的映射；development.md；AGENTS.md（面向 Agent 的输出已含 `check-run`） |
