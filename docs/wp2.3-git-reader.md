# WP2.3 Git Reader 规范

> 状态：**规范草案，待独立审查**（2026-10-18）。本文写出 WP2.3 的读取机制（TD-23）、提交关联规则（M2-Q7 / TD-17）、关联的生命周期（审查 044 N1）、`internal/gitx` 的新接口、扫描层的职责、迁移设计、正反例、验收映射与测试计划。
>
> 本文不是实现：本 PR 不提交生产代码、迁移 SQL 或实现测试。WP2.3 的实现尚未授权。
>
> 规范性的条文同步写入 [data-model.md](data-model.md)（§4.1、§4.11.2、§4.12、§6.13；§1 待 D1 的决定）、[architecture.md](architecture.md)（§6.7、§16）与 [acceptance-criteria.md](acceptance-criteria.md)。本文给出依据、实验、设计细节与待决事项；两者不一致时，以那几份文档为准。

## 1. 范围

**WP2.3 交付**：

- `internal/gitx` 的新读取操作：
  - 输入路径的元数据（TD-23）；
  - 工作区状态；
  - ref 与分离 HEAD；
  - 祖先判断；
  - 提交的读取。
- 扫描层（建议的包名 `internal/gitscan`）：
  - 扫描范围；
  - 编号匹配；
  - 写入事务。
- 迁移：`commits`、`commit_links`、`git_scan_cursors`。

**不在 WP2.3 之内**：

| 内容 | 归属 |
|---|---|
| 调用扫描的命令（`status`、`ingest`、`sync`），以及"同一项目至少间隔 30 秒"的节流 | WP2.7、M3 |
| 检查执行中对输入记录与工作区状态的使用 | WP2.5 |
| 任何输出接口与输出 Schema | 不新增 |
| `purge` 与备份 | WP2.8 |

## 2. 负责人的决定（2026-10-18）

决定记录见 [review-resolution.md §62](review-resolution.md)。这些决定不修改已冻结的进度或 ready 规则，不批准其他 M2 草案。

1. **提交关联规则**（M2-Q7 / TD-17）：
   - MVP 固定采用"`Specmetry-Ref` trailer 加正文编号匹配"，不提供关闭正文匹配的开关。
   - trailer 键不区分大小写。
   - 编号遵守既有编号规范（TD-14、`common.v1`），不扩大类型，也不放宽格式。
   - 只关联本项目已有的需求与 AC，包括已退役的。
   - 同一提交对同一实体只保留一个关联，trailer 优先。
   - 关联只提供参考进度（data-model §6.5），不代表完成、检查通过或验收。
2. **关联的生命周期**（审查 044 N1）：
   - 已入库 SHA 的关联在首次入库时固定，MVP 不重算、不补写。
   - 没有入库的提交，以后重扫时如果首次满足入库条件，按这次的发现序号入库，不回填旧序号。
   - 常规增量扫描不主动回头重算没有入库的旧提交。
   - 代价（明确接受）：
     - 后来新增的编号，不保证能与此前扫描过的提交建立关联；
     - 重扫后的首次入库，可能产生新的参考进度。
   - 不新增"所有已见 SHA"的账本，也不为此给 `commit_links` 增加 `seq`。
3. **没有新入库提交的扫描**：
   - 不分配业务 `seq`，可以更新游标。
   - `scanned_seq` 的含义、首次扫描没有匹配的提交、不同 ref 共用 SHA 等情况，写成可审查的规则（§5）。
   - 有新增提交时，沿用 F4 的逻辑写入与共享 `seq`（data-model §4.12、§6.13）。

## 3. 已冻结、本文据以细化的规则

- **data-model §4.11.2**（M2-Q2）：
  - 输入记录 `{path, mode, type, oid}`，`path` 是保存的原值；
  - 查找按 V-P04 规范化，精确匹配，区分大小写；
  - 读取的四个层次（L1 ~ L4），只要求路径元数据。
- **data-model §7.4 第 2 步**：
  - 路径不存在时为 `error` / `input_missing`；
  - 读取路径所需的对象失败时退出码 6，不执行检查、不写执行记录（PD-26）。
- **data-model §1**：代码上下文与 `clean` 的定义。见 §8 的冲突 D1。
- **data-model §4.12 与 F4（§6.13）**：
  - 一次扫描中新写入的提交是一次逻辑写入，共享一个 `seq`；
  - `commit_links` 没有 `seq`；
  - `(project_id, sha)` 唯一；
  - 只保存至少引用了一个基准编号的提交。
- **data-model §6.5**：进度只看发现序号大于 `revision_seq` 的关联提交。WP2.1 已实现并验收，本文不改。
- **architecture §6.7**：
  - 扫描所有本地分支与各 worktree 的分离 HEAD；
  - 每个 ref 一个游标；
  - 历史被改写时重新扫描该 ref，按 SHA 去重；
  - 只保存 SHA、时间与关联，正文只在内存中处理。
- **architecture §6.1.2**：
  - git 的受控环境；
  - 可能读取对象的命令都经过 `internal/gitx` 的统一入口（PD-26）；
  - git 的错误信息不进入输出、日志或保存；
  - 只支持 SHA-1（TD-26）；
  - 最低 git 2.32。
- **architecture §10.2**：
  - 第 4 条：要持久化的字符串先做密钥扫描，命中即拒绝写入；
  - 第 6 条：提交只保存 SHA、时间与关联。

## 4. TD-23：输入路径的读取机制

### 4.1 规则

对一次执行的 `head_start`（完整的 40 位提交 ID），逐个处理声明的输入 `p`（保存的原值）：

1. **查找路径**：`l = path.Clean(strings.ReplaceAll(p, "\\", "/"))`。V-P04 已保证 `l` 不越出仓库、不为空。
2. **仓库根**（`l == "."`）：
   - 命令：`git rev-parse --verify <head_start>^{tree}`。
   - 记录为 `{path: p, mode: "040000", type: "tree", oid: 根树}`。
   - 根树不是任何树的条目，没有自己的 mode，这里约定为 `040000`。这是对 §4.11.2"`.` 表示仓库根的树"的细化，见 §8 的 D3。
3. **其他路径**：每个路径单独运行一次 `git --literal-pathspecs ls-tree -z --full-tree <head_start> -- <l>`，在 worktree 根目录中执行。
   - 退出状态 0、没有输出：路径不存在（L2），结果为 `input_missing`。
   - 退出状态 0、恰好一条记录 `<mode> SP <type> SP <oid> TAB <path> NUL`，并且 `<path>` 与 `l` 完全相同：记录 `{path: p, mode, type, oid}`。
   - 退出状态 0，但输出是别的样子（多条记录，或路径不同）：不可信，按读取失败处理（退出码 6）。见 D8。
   - 退出状态非 0：读取路径所需的对象失败（L1），退出码 6，不执行、不写执行记录。不能把它当作路径不存在。
4. **`head_start` 为 NULL**（分支还没有提交）：声明了 `inputs` 时为 `input_missing`（§4.11.2 读取层次表中的"路径不存在"一行）。
5. **排序**：记录按 `path`（保存的原值）排序。
6. **门禁**：所有命令都经过 `internal/gitx` 的 `runObjects`（PD-26）。git 不具备安全读取能力时，在运行之前返回退出码 6。
7. **不运行**：LFS、子模块、任何过滤器或外部程序。

参数的理由，都由 §4.2 的实验支撑：

- `--literal-pathspecs`：不把路径当作通配模式。
- 每个路径一次调用：两个互为前缀的路径放在一次调用中时，结果会改变。
- 先规范化：`dir/` 与 `.` 原样传入时，列出的是目录的内容，而不是目录本身的条目。
- `--full-tree`：不受当前目录影响。
- `-z`：不转义路径。

### 4.2 实验

**环境**：

- Git for Windows 2.56.0（`git version 2.56.0.windows.2`），Windows 11（MINGW64_NT-10.0-26300）。
- 隔离的临时仓库，`core.ignorecase=true`、`core.symlinks=false`（Windows 的默认值）。
- 与 `gitx` 相同的受控环境与全局参数：
  - 环境变量：`GIT_CONFIG_NOSYSTEM=1`，`GIT_CONFIG_SYSTEM` 与 `GIT_CONFIG_GLOBAL` 指向空文件，`GIT_TRACE2*=0`，`GIT_OPTIONAL_LOCKS=0`，`GIT_TERMINAL_PROMPT=0`，`GIT_NO_LAZY_FETCH=1`，`GIT_ALLOW_PROTOCOL=` 为空；
  - 全局参数：`--no-pager --no-replace-objects -c core.fsmonitor=`。
- 符号链接、gitlink 与带 `*`、`?` 的路径，用 `update-index --cacheinfo` 直接写入树条目。Windows 文件系统不允许 `*`、`?`，所以加了 `-c core.protectNTFS=false`。
- L1 与 L3 用删除松散对象模拟。

| # | 情形 | 命令 | 结果 |
|---|---|---|---|
| T1 | 普通文件 `a` | `ls-tree` 字面 | 0；`100644 blob <oid>\ta` |
| T2 | 可执行文件 | 同上 | 0；`100755 blob` |
| T3 | 目录 `dir`、`dir/sub` | 同上 | 0；`040000 tree <oid>`，是目录自身的条目 |
| T4 | 深层文件 `dir/sub/y` | 同上 | 0；一条记录，路径为完整路径 |
| T5 | 符号链接 `link` | 同上 | 0；`120000 blob`，与内容相同的普通文件对象 ID 相同 |
| T6 | gitlink `gitlink` | 同上 | 0；`160000 commit <sha>`，不要求所指的提交存在 |
| T7 | 含空格的 `a b`、`sp ace/f.txt`，非 ASCII 的 `ü/n.txt`，含方括号的 `[b]` | 同上 | 0；精确匹配，`-z` 下不转义 |
| T8 | `st*r`、`q?`（另有 `stXr`） | 带 `--literal-pathspecs` | 0；只匹配同名条目 |
| T9 | `q?` | 不带 `--literal-pathspecs` | 0；匹配到 `q?`；模式路径的含义不可靠，必须带该参数 |
| T10 | 不存在的 `missing`；大小写不同的 `A`、`DIR/x`；穿过文件的 `a/b`、`dir/x/y`；子模块内部的 `gitlink/inner` | 字面 | 0；没有输出（L2） |
| T11 | 大小写不同的 `A` | 加 `-c core.ignorecase=true` | 0；没有输出：树的查找始终区分大小写 |
| T12 | 末尾带 `/` 的 `dir/`（未规范化） | 字面 | 0；列出 `dir` 的**内容**，所以必须先规范化 |
| T13 | `.`（未规范化） | 字面 | 0；列出根目录的全部条目，所以仓库根要单独处理 |
| T14 | `./a`（未规范化） | 字面 | 0；与 `a` 相同（实现仍先规范化） |
| T15 | 一次给出 `dir` 与 `dir/x` | 字面 | 0；展开了 `dir`，列出 `dir/sub`、`dir/x`，所以每个路径单独调用 |
| T16 | 仓库根 | `rev-parse --verify <c>^{tree}` | 0；根树 ID，`cat-file -t` 为 `tree` |
| T17 | 在子目录中执行 | `--full-tree` | 0；结果与在根目录中相同 |
| T18 | `dir/sub` 的树对象缺失，查 `dir/sub/y` | 字面 | **1**，`error: Could not read <oid>`，没有输出（L1） |
| T19 | `dir/sub` 的树对象缺失，查 `dir/sub` | 字面 | 0；列出 `dir/sub` 的条目（它在 `dir` 的树中）。目录自身的树缺失是 L3 |
| T20 | `a` 的 blob 缺失 | 字面 | 0；照常列出（L3）；`cat-file -e` 返回 1 |
| T21 | `dir` 的树对象缺失，查 `dir/x` 与 `dir` | 字面 | `dir/x`：**1**（L1）；`dir`：0，照常列出（L3） |
| T22 | 根树缺失 | `ls-tree`；`rev-parse ^{tree}` | **128**，`fatal: not a tree object`；**128**，`Needed a single revision`（L1） |
| T23 | 提交对象缺失；参数是 blob 的 ID；参数是不存在的 ID | `ls-tree` | **128**（L1） |

**结论**：

- 在这个版本与平台上，退出状态足以区分三种情况：路径不存在（0 且没有输出）、L1（非 0），以及 L3 与 gitlink（0 且列出）。
- 实现不依赖 git 的错误文字：错误文字一律丢弃。

**未验证的边界**：

- Linux、macOS 上的 git，以及 2.32 ~ 2.43 之间的版本。
- 真实的部分克隆（blobless、treeless）。这里只删除了松散对象，没有测试打包对象缺失。
- macOS 不区分大小写的文件系统。树的查找不经过文件系统，预期与 T11 相同，但没有实测。
- 真实创建的符号链接（`core.symlinks=true`）、LFS、超长路径。

**实现时须用夹具重做的项**：T1 ~ T23 全部，并在三个平台的 CI 上运行（§11）。

### 4.3 工作区状态

建议的命令，在 worktree 根目录中执行：

```
git status --porcelain -z --untracked-files=all --ignore-submodules=none -- . ':(exclude).specmetry'
```

- 输出为空：`clean`；有输出：`dirty`。
- 命令失败：由调用方决定（检查执行在 WP2.5 中处理）。
- 命令经过 `runObjects`：它要比较 HEAD 的树。

**实验**（同一环境）：

| # | 情形 | 结果 |
|---|---|---|
| S1 | 只有 `.specmetry/project.json` | 空（clean） |
| S2 | 一个未跟踪的文件 | `?? untracked.txt`（dirty） |
| S3 | 被 `info/exclude` 忽略的文件 | 空（clean） |
| S4 | 在分离 HEAD 的 worktree 中执行 | 空（clean） |
| S5 | `GIT_OPTIONAL_LOCKS=0` | 执行前后索引的修改时间不变 |
| S6 | 子模块检出的提交与记录的不同 | ` M sub`（dirty） |
| S7 | 同上，仓库配置 `diff.ignoreSubmodules=all`，**不带** `--ignore-submodules=none` | **空（clean）** |
| S8 | 同上，带 `--ignore-submodules=none` | ` M sub`（dirty） |

- S7 说明：data-model §1 冻结的命令中没有 `--ignore-submodules=none`。在这样配置的仓库中，子模块不一致会被判为 `clean`，与 §4.11.2"检出的提交不一致时工作区不是 `clean`"矛盾。见 §8 的 D1。
- 未验证：`.gitmodules` 中的 `submodule.<name>.ignore`、`assume-unchanged` 与 `skip-worktree` 标记、`core.autocrlf`、其他平台。
  - 前两类标记属于用户对索引的设置，`status` 的参数无法覆盖。

## 5. 提交关联与扫描

### 5.1 读取的 ref

- **本地分支**：
  - 命令：`git for-each-ref --format=%(refname)%00%(objectname)%00%(objecttype) refs/heads`，经过 `runObjects`（`objecttype` 要读取对象头）。
  - 只取 `objecttype = commit` 的 ref。指向其他对象的分支跳过，不建立游标。
  - 还没有提交的分支不会列出。
- **分离 HEAD**：
  - 命令：`git worktree list --porcelain -z`。
  - 取每个非 bare、标为 `detached` 的 worktree 的 `HEAD <sha>`，包括标为 `prunable` 的（实验：目录被移走后仍列出，并带 `prunable`）。
  - 游标的身份见 §8 的 D2。
- **固定快照**：
  - 扫描开始时一次取得所有 tip（完整的 SHA），之后的所有 git 读取都使用这些 SHA，不再按 ref 名称读取。
  - 扫描期间移动的 ref，由下一次扫描读取；被删除的分支，它的 SHA 仍可读取。
  - 对象在扫描期间被回收、无法读取时，git 返回非 0，扫描失败（§5.6）。
- **密钥扫描**（architecture §10.2 第 4 条）：
  - 要写进游标的 ref 名称与路径，写入之前按 §10.8 的规则扫描。
  - 命中的 ref 本次跳过：不读取它的提交，不建立、不更新游标，也不回显内容。只在这个 ref 上的提交因此不会被关联。
  - 不因为一个分支名而让整个扫描失败，因为分支名可能不由用户控制。见 D7。

### 5.2 扫描范围

扫描开始时，读取项目已有的游标集合 C，每个游标有自己的 `last_tip`。对每个 ref R，tip 为 T：

| 情形 | 判断 | 读取的提交 |
|---|---|---|
| R 有游标，`last_tip = T` | — | 无 |
| R 有游标，`last_tip` 是 T 的祖先 | `git merge-base --is-ancestor <last_tip> <T>` 返回 0 | `<last_tip>..<T>`（architecture §6.7 的增量） |
| R 有游标，`last_tip` 不是 T 的祖先（历史被改写） | 返回 1；或返回 128 且 `git cat-file -e <last_tip>^{commit}` 返回 1（旧 tip 已不存在） | 重新扫描 R 的全部历史 `<T>`，按 SHA 去重（architecture §6.7） |
| R 没有游标（新的 ref） | — | `<T> --not <C 中所有 last_tip>`，带 `--ignore-missing`：只读 C 中的游标还没有覆盖的历史 |
| 项目还没有任何游标（第一次扫描） | — | 每个 ref 的全部历史 |

- **祖先判断**：返回 128 且旧 tip 仍然存在时，属于读取失败（§5.6）。
- **新 ref**：
  - 没有被冻结规则覆盖，这是本文的建议，见 D4。
  - 理由：在已扫描的 `main` 上新建一个分支时，不回头重读 `main` 的全部历史。这样符合负责人的决定"常规增量扫描不主动回头重算没有入库的旧提交"。
  - `--ignore-missing` 用于已被回收的旧 tip：范围因此变大，偏向多读，不会漏读。
- **历史改写**：
  - 按冻结的文字，重新扫描该 ref 的全部历史。
  - 负责人的决定允许其中首次满足条件的未入库提交，按本次的发现序号入库。这就是"重扫后的首次入库可能产生新的参考进度"。
- **合并**：所有 ref 的范围取并集，同一个 SHA 只处理一次（不同 ref 共用 SHA，§5.5）。
- **第一次扫描**：项目还没有任何需求或 AC 时，建议不扫描、不建立游标，见 D5。
  - 否则第一次扫描就会把整个历史的游标定下来；而那时没有任何编号可以匹配，之后的增量扫描也不再回头。

**读取提交**：

- 命令：`git log -z --stdin --format=%H%x00%ct%x00%(trailers:key=Specmetry-Ref,valueonly,separator=%x2C)%x00%B`。
- 范围从 stdin 传入（tip、`--not` 与排除的 tip），不受命令行长度限制；排除的 tip 带 `--ignore-missing`。
- 输出按 NUL 分隔，每个提交四个字段：
  - SHA；
  - 提交时间（`%ct`，秒）；
  - trailer 的值；
  - 完整说明（标题、正文与 trailer 行）。
- 每条记录都要校验：SHA 为 40 位小写十六进制，时间为十进制整数。不合格时整个扫描失败（§5.6）。

**实验**（同一环境）：

| # | 情形 | 结果 |
|---|---|---|
| L1 | 七个提交 | 记录之间与字段之间都是 NUL，可以按 4 个一组切分 |
| L2 | trailer 行 `Specmetry-Ref: REQ-010, AC-010-1`、`specmetry-ref: REQ-011 REQ-012`、`SPECMETRY-REF:REQ-013` | 值为 `REQ-010, AC-010-1,REQ-011 REQ-012,REQ-013`：键不区分大小写，冒号后可以没有空格 |
| L3 | `Specmetry-Ref:` 行在倒数第二段，最后一段是普通文字 | trailer 的值为空：git 只认最后一段的 trailer，这一行仍在正文中 |
| L4 | 以 CRLF 结尾的说明 | trailer 的值为 `REQ-030` |
| L5 | `GOAL-1` 与 `REQ-040, nonsense, AC-040-1; REQ-041` | 原样返回值，由 Specmetry 自己切分（§5.3） |
| L6 | 说明中含 NUL | `git commit-tree` 拒绝（`a NUL byte in commit log message not allowed`）。用原始对象写入的，`log` 在 NUL 处截断，不会打乱字段 |
| L7 | `merge-base --is-ancestor` | 是祖先返回 0，不是返回 1，ID 未知返回 128 |
| L8 | `log <T> --not <未知 ID>` | 128，`bad object`；加 `--ignore-missing` 后忽略该 ID |
| L9 | 从 `--stdin` 传入范围 | 结果与命令行相同 |
| L10 | `for-each-ref`、`worktree list --porcelain -z` | 分支与 SHA；各 worktree 的 `HEAD` 与 `detached`、`prunable` 标记 |

**未验证**：

- 其他平台与版本。
- 说明的编码不是 UTF-8 的仓库：编号是 ASCII，预期不受影响。
- 大型仓库第一次扫描的耗时（没有性能基准）。
- 真实的并发扫描（§5.6 只是设计）。

### 5.3 编号的匹配

- **词**：
  - 词是由 ASCII 字母、数字、`_`、`-` 组成的最长连续串。其他字符都是边界，包括空白、标点、CR，以及中文等非 ASCII 字符。
  - 只有整个词与编号格式完全一致时才算提到编号：
    - 需求：`^REQ-[0-9]{3,}$`；
    - AC：`^AC-[0-9]{3,}-[1-9][0-9]*$`（`schemas/common.v1`、TD-14）。
  - 不放宽格式：小写的 `req-006`、`REQ-1`、`AC-007-0` 都不匹配。
  - 不扩大类型：`GOAL-n`、`Q-nnn` 不关联（`commit_links.target_type` 只有需求与 AC）。
- **trailer 提到的编号**：把 git 返回的 trailer 值按上述规则切成词，例如 `AC-040-1;` 得到 `AC-040-1`。
  - 哪些行算 trailer，由 git 判定（L2 ~ L4）。
- **正文提到的编号**：把完整说明（`%B`，包括标题与 trailer 行）按同样规则切成词。
- **关联**：
  - 每个提到的编号，在写入事务中查本项目的需求或 AC，状态不限：已退役的也关联，历史保留；进度只看 `active` 条目（§6.5）。
  - 本项目没有这个编号就忽略，不报错。
  - 同一提交对同一实体只有一条关联：编号在 trailer 中出现时 `link_source = trailer`，否则为 `message`。
  - trailer 行也属于正文，所以 trailer 中的编号总是记为 `trailer`。
- **入库**：至少有一条关联的提交才入库。说明中没有任何像编号的词时，不必进入写入事务。
- **跨项目的编号**：编号格式不带项目，所以 Specmetry 无法区分别的项目的 `REQ-001`。本项目也有 `REQ-001` 时，会关联到本项目的条目。这是格式本身的限制；在 MVP 中不扩展 trailer 的格式（负责人决定第 1 条）。
- **隐私**（architecture §10.2 第 6 条）：
  - 标题、正文与 trailer 的值只在内存中切词，用完即丢。
  - 不进入数据库、日志、错误输出或 `error.v1`，也不出现在测试失败的消息中，后者由测试自身保证。
  - git 的错误信息照旧丢弃（§6.1.2）。

### 5.4 关联的生命周期

- 一个 SHA 入库时，它的关联同时写入，之后不再重算、不再补写。`commit_links` 没有 `seq`，它的发现时序就是所属提交的 `commits.seq`。
- 没有入库的提交不留任何记录，也就没有"已见"的账本。
- 一次扫描只处理 §5.2 的范围。没有入库的旧提交，只有在落入以后某次扫描的范围时才会再被读取：
  - 历史改写后的重扫；
  - 新 ref 中还没有被覆盖的部分。
  这时它如果首次满足入库条件，就按这次的 `seq` 入库。
- 所以截止点 `as_of_seq` 的取法（`commits.seq ≤ as_of_seq` 的提交及其关联）成立：关联不会在以后被补写到较早的序号上（data-model §6.13 的前提）。

### 5.5 写入与 `seq`

**一次扫描的步骤**：

1. **计划**：在一个读事务中读取项目的游标，并确认项目是否已有需求或 AC（D5）。
2. **读取 git**：在任何数据库事务之外进行（§5.1 ~ §5.3），得到候选提交：说明中至少有一个像编号的词。
3. **写事务**（`BEGIN IMMEDIATE`）：
   1. 重新读取游标。只要有任何一个与计划时不同（`last_tip` 改变、游标新出现或消失），本次**什么都不写**，返回"并发跳过"。这不是错误，下一次扫描会重做。见 D6。
   2. 去掉已经入库的 SHA：它们的关联是固定的。
   3. 在本项目的需求与 AC 中解析编号。有关联的提交就是新入库的提交。
   4. 有新入库的提交时，分配一个 `seq` = s，写入这些 `commits`（`seq = s`）及其 `commit_links`。
   5. 写游标：每个计划中的 ref 写入 `last_tip = T`、`scanned_at = 当前时间`。分配了 s 时，这些游标的 `scanned_seq = s`；否则 `scanned_seq` 不变，新游标为 NULL。
   6. 提交事务。
4. 任何失败都回滚，什么都不留。

**`scanned_seq` 的含义**：

- 它是最近一次"有新入库提交、并且写过这个游标"的扫描的 `seq`。从来没有这样的扫描时为 NULL。
- 它只用于诊断，不是推导的输入。推导只用 `commits.seq`（§6.5）。

**各种情形**：

| 情形 | 业务写入 | `seq` | 游标 |
|---|---|---|---|
| 范围中没有候选，或候选都已入库 | 无 | 不分配 | 写 `last_tip`、`scanned_at`；`scanned_seq` 不变 |
| 第一次扫描，没有匹配的提交 | 无 | 不分配 | 新建游标，`scanned_seq` 为 NULL |
| 有新入库的提交 | `commits` 与 `commit_links` | 一个，共享 | 写 `last_tip`、`scanned_at`，`scanned_seq = s` |
| 不同 ref 共用一个 SHA | 只入库一次，关联一次 | 不变 | 各 ref 的游标照常写 |
| 同一个 SHA 在以后的扫描中再次出现 | 无（已入库，关联固定） | 不分配 | 照常写 |
| 并发跳过、读取失败、写入失败 | 无 | 不分配 | 不变 |

- `scanned_at`（毫秒）只用于显示与"至少间隔 30 秒"的节流，不参与任何先后判断（先后只由 `seq` 决定）。节流由调用方执行；被节流跳过的扫描什么都不写。

### 5.6 失败与重试

- **门禁**：git 不具备安全读取能力时（PD-26），在运行任何读取对象的命令之前返回退出码 6，什么都不写。
- **git 的失败**：以下任何一种都是读取失败，退出码 6，不写入，游标不变：
  - 非 0 退出（`is-ancestor` 的 0 与 1、`ls-tree` 的"0 且没有输出"属于正常结果）；
  - 不合格的输出；
  - 被取消。
- 错误只说明是哪个 git 子命令、退出状态是多少（§6.1.2）。
- **数据库的失败**：沿用存储层既有的分类。打开数据库时的偶发失败另见维护事项 M-14。
- **不自动重试**：一次命令中不重试扫描。失败与并发跳过都不留下状态，下一次扫描从不变的游标重新开始。

## 6. 接口设计

**`internal/gitx` 新增**（名称与签名供审查，实现时可调整；行为以本文为准）：

```go
// Entry is a tree entry's metadata (data-model §4.11.2).
type Entry struct{ Mode, Type, OID string }

// TreeEntry reads the entry of a lookup path (V-P04 normalized) in the
// tree of a full commit ID: "." is the root tree. found is false when the
// tree has no such path (L2); an error is an object the path needs that
// cannot be read (L1, exit 6).
func TreeEntry(ctx context.Context, worktree, commit, lookup string) (e Entry, found bool, err error)

// Clean reports whether the worktree is clean (data-model §1).
func Clean(ctx context.Context, worktree string) (bool, error)

// Tip is a scanned ref: a local branch, or the detached HEAD of a worktree.
type Tip struct {
	Ref          string // "refs/heads/<name>", or "" for a detached HEAD
	WorktreePath string // the canonical path of a detached HEAD's worktree
	Commit       string // full SHA-1
}

// Tips lists the local branches that point to commits and the detached
// HEADs of the non-bare worktrees.
func Tips(ctx context.Context, worktree string) ([]Tip, error)

// IsAncestor reports whether ancestor is an ancestor of commit; missing is
// true when ancestor is not a commit of the repository any more.
func IsAncestor(ctx context.Context, worktree, ancestor, commit string) (yes, missing bool, err error)

// Commit is one commit read by Log. Trailers and Message are only in
// memory: callers never store, log or echo them.
type Commit struct {
	SHA      string
	Time     int64 // committer time, seconds
	Trailers string
	Message  string
}

// Log reads the commits reachable from include and from none of exclude
// (missing ones are ignored), newest first, calling fn for each.
func Log(ctx context.Context, worktree string, include, exclude []string, fn func(Commit) error) error
```

- 除 `worktree list` 之外，这些操作都经过 `runObjects`。`worktree list` 只读取 git 目录的布局。
- 错误都是 `apperr.Environment`（退出码 6），不带 git 的输出。
- `gitx` 不知道编号、项目或数据库。

**扫描层**（建议 `internal/gitscan`）：

| 职责 | 说明 |
|---|---|
| `Mentions(trailers, message string) []Mention` | 纯函数：§5.3 的切词与 trailer 优先，结果为编号与来源 |
| 计划 | 读游标，确认是否已有编号（D5） |
| 读取 | 调用 `gitx`：tip、范围、日志；做密钥扫描；得到候选提交 |
| 写入 | §5.5 的写事务：比较游标、去重、解析编号、分配 `seq`、写入 |
| `Scan(ctx, store, project, worktree) (Result, error)` | 编排以上步骤。`Result` 只含计数：读取的提交数、新入库数、跳过的 ref 数、是否并发跳过 |

- 扫描层不输出、不节流，也不决定在哪些命令中调用。

## 7. 迁移设计（`0007_commits`，WP2.3 实现）

本节只是设计，不是迁移 SQL。写法沿用既有迁移：

- 26 位 ID；
- `(id, project_id)` 唯一，同项目内的外键用 `DEFERRABLE INITIALLY DEFERRED`；
- 只追加表用触发器阻止 UPDATE 与 DELETE（维护模式除外）。

**`commits`**（只追加）：

| 列 | 定义 |
|---|---|
| `id` | `TEXT PRIMARY KEY`，长度 26 |
| `seq` | `INTEGER NOT NULL`；同一次扫描新写入的提交共用，**不唯一** |
| `project_id` | `TEXT NOT NULL`，引用 `projects` |
| `sha` | `TEXT NOT NULL`，40 位小写十六进制（TD-26） |
| `committed_at` | `INTEGER NOT NULL`，提交者时间（秒），只用于显示 |

- 约束：`UNIQUE (project_id, sha)`、`UNIQUE (id, project_id)`。
- 索引：`(project_id, seq)`。
- 触发器：只追加。

**`commit_links`**（只追加）：

| 列 | 定义 |
|---|---|
| `commit_id` | `TEXT NOT NULL` |
| `project_id` | `TEXT NOT NULL` |
| `target_type` | `TEXT NOT NULL`，`requirement` / `criterion` |
| `target_id` | `TEXT NOT NULL`，长度 26 |
| `link_source` | `TEXT NOT NULL`，`trailer` / `message` |

- 约束：
  - `PRIMARY KEY (commit_id, target_type, target_id)`：同一提交对同一实体只有一条；
  - `FOREIGN KEY (commit_id, project_id) REFERENCES commits (id, project_id)`。
- 建议：一个 `BEFORE INSERT` 触发器，要求 `target_id` 是同一项目中对应类型的条目。`source_refs` 目前只在应用中检查，审查时可以选择沿用。
- 索引：`(project_id, target_type, target_id)`，供进度查询使用。
- 触发器：只追加。关联只与它的提交在同一事务中写入，由写入路径保证，并有测试覆盖。

**`git_scan_cursors`**（受控可变）：

| 列 | 定义 |
|---|---|
| `project_id` | `TEXT NOT NULL`，引用 `projects` |
| `kind` | `TEXT NOT NULL`，`branch` / `detached` |
| `name` | `TEXT NOT NULL`：分支为完整的 ref 名称；分离 HEAD 按 D2 的选项 |
| `last_tip` | `TEXT NOT NULL`，40 位小写十六进制 |
| `scanned_seq` | `INTEGER`，可为 NULL（§5.5） |
| `scanned_at` | `INTEGER NOT NULL`，毫秒，只用于显示与节流 |

- 约束：`PRIMARY KEY (project_id, kind, name)`。
- 只由扫描的写事务更新，没有只追加触发器。
- 分支被删除后，游标保留，它的 `last_tip` 继续用于新 ref 的范围（§5.2）。

**事务与 `seq`**：

- 一次扫描至多一个写事务，至多分配一个 `seq`。
- 分配 `seq` 的写入同时包含这次的全部 `commits`、`commit_links`，以及游标的更新。
- 没有新入库的提交时，写事务只更新游标，不分配 `seq`，不改变 `meta.seq_counter`。

**保留与清除**：

- `commits`、`commit_links` 是 C2 数据，保存到 `purge`（data-model §10）。
- 游标随项目清除；`purge` 与备份在 WP2.8 中覆盖这三张表。

## 8. 与现有契约的冲突与待决事项

| # | 内容 | 选项 | 建议 |
|---|---|---|---|
| **D1（冲突）** | data-model §1 冻结的 `clean` 命令没有 `--ignore-submodules=none`。仓库配置 `diff.ignoreSubmodules=all` 时，子模块不一致被判为 `clean`（S7），与 §4.11.2 的说法矛盾 | (a) 在定义中加入 `--ignore-submodules=none`；(b) 保留原文，并写明仓库配置可能隐藏子模块的变化 | **(a)**：只收紧到原意，覆盖仓库配置。需要负责人确认修改冻结文字 |
| **D2（冲突）** | architecture §6.7 要求扫描"各 worktree 的分离 HEAD"，但没有登记的 worktree 没有 `worktree_id`，游标没有身份 | (a) 分离 HEAD 的游标按 worktree 的规范化路径识别（`name` 为该路径，写入前做密钥扫描）；(b) 只扫描已登记的 worktree，按 `worktree_id` 识别 | **(a)**：符合"各 worktree"。路径与 `worktrees.path` 是同一类数据 |
| **D3（细化）** | §4.11.2 规定 `.` 表示仓库根的树，但根树没有 mode | (a) 记为 `{".", "040000", "tree", 根树 ID}` | **(a)**；没有其他不改变 M1 已保存含义的选项 |
| **D4（审查）** | 新 ref 的第一次扫描范围（冻结文字没有规定） | (a) 排除已有游标覆盖的历史（`--not`、`--ignore-missing`）；(b) 读取该 ref 的全部历史 | **(a)**：符合"常规增量扫描不回头"；(b) 会在每次新建分支时重读主干的全部历史 |
| **D5（审查）** | 项目还没有任何需求或 AC 时的扫描 | (a) 不扫描、不建立游标；(b) 照常扫描并建立游标 | **(a)**：没有编号时什么都不能匹配；(b) 会把整个历史的游标定在"什么都没有关联"的状态 |
| **D6（审查）** | 写入时发现游标已被并发的扫描改变 | (a) 整次扫描什么都不写；(b) 只放弃那几个游标的更新 | **(a)**：结果与"另一次扫描已完成"等价，避免两次扫描交错写入 |
| **D7（审查）** | 分支名或路径命中密钥规则 | (a) 跳过该 ref；(b) 整个扫描失败 | **(a)**：分支名可能不由用户控制，不应让每次 `status` 都失败 |
| **D8（审查）** | `ls-tree` 退出 0 但输出不是一条匹配的记录 | (a) 按读取失败处理，退出码 6；(b) 按路径不存在处理 | **(a)**：不把无法解释的输出写成检查结果（PD-26 的原则） |

**其他待决或需要说明的事项**：

- 跨项目的同名编号无法区分（§5.3），这是现有格式的限制，不在 MVP 中扩展。
- `commit_links` 的目标检查用触发器还是只在应用中检查（§7），由审查选择。
- 第一次扫描大型仓库的耗时没有基准，也没有上限。实现时测量，必要时另立事项。
- 维护事项 M-14（macOS 上多个进程打开同一个数据库时偶发 `SQLITE_READONLY_RECOVERY`）未解决。扫描与它所在的命令同样要打开数据库，所以可能受到影响；M-14 单独诊断，不在本规范中处理。

## 9. 正反例

**编号匹配**（§5.3）。设本项目已有 REQ-001、REQ-003（已退役）、REQ-004、REQ-010、REQ-011、REQ-012、REQ-013、REQ-030、REQ-040、REQ-041、REQ-060 与 AC-001-1、AC-010-1、AC-040-1：

| # | 说明中的内容 | 关联 |
|---|---|---|
| M1 | 标题 `fix REQ-001` | REQ-001（message） |
| M2 | trailer `Specmetry-Ref: REQ-010, AC-010-1` | REQ-010、AC-010-1（trailer） |
| M3 | trailer `specmetry-ref: REQ-011 REQ-012`、`SPECMETRY-REF:REQ-013` | 三者都是 trailer |
| M4 | `Specmetry-Ref: REQ-030` 在倒数第二段 | REQ-030（message：不是 trailer） |
| M5 | 正文与 trailer 都有 REQ-010 | 一条：trailer |
| M6 | `XREQ-002`、`REQ-0003x`、`REQ-005-fix`、`req-006`、`AC-007-0`、`REQ-1` | 无 |
| M7 | `(REQ-004)`、`REQ-004.`、`完成REQ-060了` | REQ-004、REQ-060（message） |
| M8 | trailer `AC-040-1; REQ-041` | AC-040-1、REQ-041（trailer） |
| M9 | trailer `GOAL-1` | 无（类型不关联） |
| M10 | `REQ-999`（本项目没有） | 无；没有其他关联时，提交不入库 |
| M11 | `REQ-003`（已退役） | REQ-003（message） |

**生命周期与扫描**（§5.2、§5.4 ~ §5.6）：

| # | 情形 | 期望 |
|---|---|---|
| L1 | 提交 X 提到 REQ-020，扫描时 REQ-020 还不存在 | X 不入库；之后的常规增量扫描不再读取 X |
| L2 | X 已入库（关联 REQ-001），后来新增的 REQ-020 也出现在 X 的说明中 | 不补写 REQ-020 的关联 |
| L3 | L1 之后分支被改写，重扫时 X 仍在历史中，REQ-020 已存在 | X 以这次扫描的 `seq` 入库 |
| L4 | 范围中没有新的候选 | 不分配 `seq`，`meta.seq_counter` 不变；游标更新；`scanned_seq` 不变 |
| L5 | `main` 与 `feature` 共用的提交提到 REQ-001 | 只入库一次，一条关联 |
| L6 | 从已扫描的 `main` 新建分支并提交一个新提交 | 只读取新提交（D4 (a)） |
| L7 | `feature` 被 reset 后提交新提交（MVP-WT-04） | 重扫 `feature`；已入库的不重复；新提交入库 |
| L8 | 分离 HEAD 上的提交提到 REQ-001 | 入库（D2 (a)） |
| L9 | 写入时游标已被另一个进程改变 | 什么都不写，结果为并发跳过 |
| L10 | 扫描中某个 git 命令失败 | 退出码 6，什么都不写 |
| L11 | 分支名命中密钥规则 | 该 ref 跳过，不回显（D7 (a)） |
| L12 | 项目还没有需求或 AC | 不扫描、不建立游标（D5 (a)） |
| L13 | 说明中有 canary 字符串（标题、正文、trailer 的值） | 数据库、WAL、日志与错误中都找不到（MVP-EV-11） |

**输入路径与工作区状态**：见 §4.2 的 T1 ~ T23 与 §4.3 的 S1 ~ S8，实现时逐项写成测试。

## 10. 验收映射

| 验收项 | 内容 | WP2.3 的部分 | 其余部分 |
|---|---|---|---|
| MVP-WT-04 | 每个 ref 的扫描、改写后重扫、不重复 | 扫描层的集成测试（L5 ~ L7） | 命令中的触发：WP2.7 |
| MVP-EV-11 | 提交的标题中有 canary → 只保存 SHA 与时间 | 扫描层的集成测试（L13） | — |
| MVP-EV-10 | 输入记录 | `gitx.TreeEntry`（T1 ~ T23） | 在执行中记录：WP2.5 |
| MVP-EV-08、EV-09、WT-03 | 工作区状态与代码上下文 | `gitx.Clean`（S1 ~ S8） | 在执行与命令中使用：WP2.5、WP2.4 |
| MVP-EV-07 | 只追加表的触发器 | `commits`、`commit_links` 的迁移测试 | — |
| **MVP-EV-14**（新增） | 编号匹配的规则 | `Mentions` 的单元测试（M1 ~ M11）与集成测试 | — |
| **MVP-EV-15**（新增） | 关联的生命周期 | 集成测试（L1 ~ L3） | — |
| **MVP-EV-16**（新增） | 扫描的写入：`seq`、游标、原子性、并发、失败 | 集成测试（L4、L9、L10） | — |
| **MVP-EV-17**（新增） | 输入路径的读取机制（TD-23） | `gitx` 的集成测试 | 在执行中的退出码与不写记录：WP2.5 |
| **MVP-WT-06**（新增） | 扫描的范围：分离 HEAD、新 ref、密钥命中 | 集成测试（L6、L8、L11、L12） | — |
| §6.5 进度中的关联提交 | — | — | 已由 WP2.1 覆盖 |
| CX | 没有直接针对提交关联的反例项 | — | E2E-2 的"追加带 trailer 的提交"属于 WP2.9 |

## 11. 实现测试计划

- **单元**：
  - `Mentions`：M1 ~ M11，以及边界字符、CR、非 ASCII、空输入。
  - `ls-tree`、`log`、`worktree list` 的输出解析：合格与不合格的输入，不合格时返回错误。
  - 范围规划：给定游标与祖先判断的结果，覆盖 §5.2 的各行。
- **`gitx` 集成**：
  - 用 `testutil` 的 git 夹具在临时仓库中重做 T1 ~ T23 与 S1 ~ S8，三个平台。
  - L1、L3 通过删除松散对象模拟。
  - 能力门禁：沿用 `SPECMETRY_TEST_GIT_OBJECTS=unsafe` 的做法（维护事项 M-9），确认新命令在不具备能力时都在运行前被拒绝。
- **存储**：
  - 迁移 0007 的约束、唯一性、外键与只追加触发器（MVP-EV-07）。
  - 没有新入库时 `seq_counter` 不变。
- **扫描层集成**（真实 git 与临时数据库）：
  - L1 ~ L13；
  - 两个进程同时扫描时的并发跳过（`testutil.HelperCommand`）；
  - 扫描中途注入 git 失败；
  - canary 在数据库、WAL、日志与错误中都不出现。
- **不在 WP2.3**：命令级的验收（WP2.7）、E2E（WP2.9）。

## 12. M-7 的处理顺序

- M-7（部分成功的 JSON 输出只检查以 `{` 开头）必须在新增输出接口之前修复：WP2.4（`claim`）、WP2.5（`check run`）、WP2.7（`status`、`gaps`）。
- WP2.3 不新增输出接口，所以 M-7 不是 WP2.3 的开工条件。
- 建议在本规范冻结之后，作为单独的 PR 修复。它可以与 WP2.3 的实现并行，但须在 WP2.4、WP2.5、WP2.7 任何一个开工之前完成。
- M-7 保持未关闭，直到那个 PR 经审查合并。
