# WP2.3 Git Reader 规范

> 状态：**规范草案，按审查 050（NO-GO）修订，待复审**（2026-10-18）。负责人已决定 D1、D2、D7（§2、§8）；其余没有获批的草案保持原状。本文写出 WP2.3 的读取机制（TD-23）、提交关联规则（M2-Q7 / TD-17）、关联的生命周期（审查 044 N1）、`internal/gitx` 的新接口、扫描层的职责、迁移设计、正反例、验收映射与测试计划。
>
> 本文不是实现：本 PR 不提交生产代码、迁移 SQL 或实现测试。WP2.3 的实现尚未授权。
>
> 规范性的条文同步写入 [data-model.md](data-model.md)（§1、§4.1、§4.11.2、§4.12、§6.13）、[architecture.md](architecture.md)（§6.7、§16）与 [acceptance-criteria.md](acceptance-criteria.md)。本文给出依据、实验、设计细节与待决事项；两者不一致时，以那几份文档为准。

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

**第二轮决定**（2026-10-18，审查 050 之后；记录见 [review-resolution.md §63](review-resolution.md)）：

4. **D1**：采用在 `clean` 定义中加入 `--ignore-submodules=none` 的方向，同步 `clean` 的定义与验收。外部过滤器的问题另行解决，不能因此宣称原来的 `status` 方案安全（§4.3）。
5. **D2**：保留所有 worktree 的分离 HEAD，以规范化路径作为游标身份，不缩小到已登记的 worktree。路径的移动、删除、重用、大小写与规范化行为见 §5.1。
6. **D7**：不静默跳过敏感的 ref 或路径。命中密钥规则时拒绝本次扫描，不写提交、关联、`seq` 或游标，不回显敏感值。错误分类遵守既有契约，不新增隐私豁免（§5.1）。

其余没有获批的草案保持原状。

## 3. 已冻结、本文据以细化的规则

- **data-model §4.11.2**（M2-Q2）：
  - 输入记录 `{path, mode, type, oid}`，`path` 是保存的原值；
  - 查找按 V-P04 规范化，精确匹配，区分大小写；
  - 读取的四个层次（L1 ~ L4），只要求路径元数据。
- **data-model §7.4 第 2 步**：
  - 路径不存在时为 `error` / `input_missing`；
  - 读取路径所需的对象失败时退出码 6，不执行检查、不写执行记录（PD-26）。
- **data-model §1**：代码上下文与 `clean` 的定义。D1 已决定加入 `--ignore-submodules=none`（§4.3）。
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

### 4.3 工作区状态（审查 050 B3、负责人决定 D1）

**语义**（data-model §1，D1）：`clean` 指以下命令在 worktree 根目录中的输出为空：

```
git status --porcelain --untracked-files=all --ignore-submodules=none -- . ':(exclude).specmetry'
```

- `--ignore-submodules=none`：仓库配置（例如 `diff.ignoreSubmodules=all`）不能隐藏子模块的变化。
- 这个命令只定义"什么算变化"，**不是读取方式**。

**读取方式**：Specmetry **不直接在用户的仓库上运行这条命令**。原因：

- `status` 判断只改了修改时间的文件是否真的被修改时，会经过仓库属性所选的 `filter.<name>.clean` / `process` 转换内容，从而执行仓库配置的程序（审查 050 B3；实验 C1）。
- 带 `--ignore-submodules=none` 时，它还会在每个子模块中启动一个 `git status`。那个子进程读取子模块自己的配置，同样会执行过滤器（实验 C9）。
- 受控环境、关闭 FSMonitor 与对象读取门禁，都不能阻止这些程序执行。

读取步骤如下：

1. **私有的 git 目录**：Specmetry 在自己的私有临时目录中建立一个 git 目录，一次读取结束后删除，内容如下：
   - `HEAD`：worktree 当前的 HEAD（完整 SHA；分支还没有提交时没有 `HEAD`）；
   - `objects/info/alternates`：指向仓库的对象目录（只读引用，不复制对象）；
   - `index`：worktree 索引文件的副本；拆分索引时，同时复制 `sharedindex.*`；
   - `info/exclude`、`info/attributes`：公共目录中这两个文件的副本；
   - `config`：只有白名单中的键。
     - 白名单：
       - `core.autocrlf`、`core.eol`、`core.safecrlf`、`core.ignorecase`、`core.precomposeunicode`；
       - `core.protecthfs`、`core.protectntfs`、`core.symlinks`、`core.filemode`、`core.checkstat`、`core.trustctime`；
       - `core.excludesfile`、`core.attributesfile`、`core.sparsecheckout`、`core.sparsecheckoutcone`、`index.sparse`；
       - `core.longpaths`、`core.fscache`、`core.checkroundtripencoding`。
     - 另外固定写入 `core.repositoryformatversion = 0`、`core.bare = false`、`core.untrackedCache = false`。
     - 值取自 `git config --list --local --includes -z`；仓库启用了 `extensions.worktreeConfig` 时，再读 `--worktree` 的值，按同一份白名单过滤，worktree 的值优先。
     - 没有任何 `filter.*`、`core.fsmonitor`、`core.hooksPath`、`include*`、`extensions.partialClone` 或远端配置。
   - 这些键只影响内置的转换、比较与忽略规则，都不会让 git 执行程序。
2. **超级项目**：
   - 命令：`git --git-dir=<私有目录> --work-tree=<worktree> status --porcelain -z --untracked-files=all --ignore-submodules=all -- . ':(exclude).specmetry'`，经过对象读取门禁（PD-26）。
   - `--ignore-submodules=all` 使 git 不进入子模块，也不启动子进程（实验 C9：子进程数为 0）。
3. **子模块**：
   - 对私有索引中每个 `160000` 条目（`ls-files --stage -z`），如果 worktree 中这个路径有 `.git`（已初始化）：
     - 用 `git --git-dir=<子模块的 .git> rev-parse --verify HEAD` 取得子模块的 HEAD。它只读 ref，不读取工作区，不执行过滤器。
     - HEAD 与 gitlink 不同即为 `dirty`。
     - 再对子模块递归执行第 1 ~ 3 步，它自己不是 `clean` 也为 `dirty`。
   - 没有初始化的子模块不读取，与 `status` 相同。
4. **结果**：
   - 超级项目与所有已初始化子模块都没有变化：`clean`；否则 `dirty`。
   - 任何一步失败（复制失败、git 非 0 退出、输出不合格）时，不报告 `clean`：
     - 返回环境错误（退出码 6）；
     - 或者由调用方按其契约记为 `unknown`（检查执行在 WP2.5 中决定）。

**这种读取方式的保证与差异**：

- **调用之前就保证**：git 读取的配置是 Specmetry 刚写出的白名单副本，其中没有任何会启动程序的设置。
  - 用户仓库的配置在这之后被修改，也不影响这次读取。
  - 这不依赖"仓库可信"，不依赖事后检测程序是否执行，也没有放宽"不执行外部程序"的规则。
- **不写用户的仓库**：
  - 索引与配置都是副本；`GIT_OPTIONAL_LOCKS=0` 另外保证不写这些副本的锁。
  - 私有目录的位置与权限，在实现时与数据目录的规则一起确定。
- **保守的差异**：
  - 属性指定了过滤器、并且 stat 信息发生了变化的文件，git 无法在不执行过滤器的情况下比较内容。所以它按工作区原样内容与索引对象比较，不一致就是 `dirty`（实验 C7）。
  - 直接运行 `status` 时，过滤器会把内容转换后再比较，可能得到 `clean`。
  - 差异只会让结果从 `clean` 变成 `dirty`（执行因此不合格），不会把真实的变化报告为 `clean`。
  - 这写进 data-model §1 的定义（D1 的同步）。
- **没有关闭的边界**：
  - `assume-unchanged`、`skip-worktree` 是用户在索引上的标记，`status` 的参数无法覆盖，这里也不处理。
  - `core.autocrlf` 等内置转换的结果与直接运行 `status` 相同（同一份白名单配置）。

**实验**（Git for Windows 2.56.0，Windows 11；隔离夹具；与 `gitx` 相同的受控环境）：

- 标记程序都是夹具中的 shell 脚本，运行时向标记目录追加一行。
- clean 过滤器原样输出；另一个过滤器把内容转为大写；process 过滤器记录后退出 1。
- 只在原草案的对照命令中允许它们运行，用来证明风险确实存在。

| # | 情形 | 原草案的命令 | 本方案 |
|---|---|---|---|
| C1 | `f`、`g` 属性指定过滤器，提交之后才配置过滤器，只改修改时间 | clean 与 process 过滤器**都执行了**（有标记），命令失败 | 没有标记；输出为空（`clean`） |
| C2 | 修改一个普通文件 | — | ` M plain.txt`；没有标记 |
| C3 | 未跟踪的文件 | — | `?? untracked.txt`；没有标记 |
| C4 | 被 `info/exclude` 忽略的文件 | — | 空；没有标记 |
| C5 | 已暂存的修改 | — | `M  plain.txt`；没有标记 |
| C6 | 删除一个跟踪的文件 | — | ` D plain.txt`；没有标记 |
| C7 | 提交时就有会改写内容的过滤器（索引中的对象与文件不同），只改修改时间 | 过滤器**执行了**，结果 `clean` | ` M f`（`dirty`，保守的差异）；没有标记 |
| C8 | 普通文件只改修改时间 | — | 空（内容比较后为 `clean`） |
| C9 | 已初始化的子模块，它自己的配置中有过滤器，子模块中的文件只改修改时间 | 启动了 **2 个**子进程，子模块的过滤器**执行了** | 子进程 **0** 个；没有标记；`clean` |
| C10 | 子模块检出的提交与 gitlink 不同 | ` M sub` | 子模块的提交不同（`dirty`）；没有标记 |
| C11 | 同上，仓库配置 `diff.ignoreSubmodules=all` | 带 `--ignore-submodules=none`：` M sub` | `dirty`；没有标记 |
| S5 | `GIT_OPTIONAL_LOCKS=0` | 执行前后索引的修改时间不变（上一轮实验） | 只读副本 |

**未验证**：

- Linux、macOS 以及 2.56 以外的 git 版本。
- 真实的长驻 process 过滤器协议、LFS、拆分索引、稀疏检出与稀疏索引（包括是否还需要复制 `info/sparse-checkout`）。
- `extensions.worktreeConfig` 下的 worktree 配置。
- 以 gitfile 指向 `.git/modules/<name>` 的子模块（实验只用了内嵌仓库），以及嵌套多层的子模块。
- 大仓库复制索引的耗时。

以上都列入实现测试（§11）。

## 5. 提交关联与扫描

### 5.1 读取的 ref

- **本地分支**：
  - 命令：`git for-each-ref --format=%(refname)%00%(objectname)%00%(objecttype) refs/heads`，经过 `runObjects`（`objecttype` 要读取对象头）。
  - 只取 `objecttype = commit` 的 ref；指向其他对象的分支跳过，不建立游标。
  - 还没有提交的分支不会列出。
- **分离 HEAD**（负责人决定 D2：保留所有 worktree，不缩小到已登记的）：
  - 命令：`git worktree list --porcelain -z`。不认识 `-z` 的 git（usage 错误，退出 129）上，本次扫描返回退出码 6。不改按行解析：没有 `-z` 时，路径中的换行无法可靠切分（审查 050 N1，§13）。
  - 取每个非 bare、标为 `detached` 的 worktree 的 `HEAD <sha>`，包括标为 `prunable` 的。
  - 每个分离 HEAD 都是必需读取的正向 tip（§5.2）。
- **分离 HEAD 的游标身份**（D2）：
  - 规则：`name = gitx.Canonical(<worktree list 给出的路径>)`。
    - 路径存在时，取 `filepath.Clean` 后再 `EvalSymlinks` 的结果；Windows 上这一步同时统一长短文件名与大小写。
    - 路径已不存在（`prunable`）时，用父目录 `EvalSymlinks` 的结果加上最后一级。
    - 父目录也不存在时，只做字面的 `Clean`。不要求不存在的路径 `EvalSymlinks` 成功。
  - **移动**：`git worktree move` 或 `repair` 之后路径改变，它是新的游标，按新 ref 的规则读取（§5.2）；原来的游标保留，继续用于排除。
  - **删除**：
    - `prune` 之前，worktree 仍以 `prunable` 列出，照常扫描。这时路径不存在，身份可能从解析后的路径变为字面路径，于是成为新的游标；这是无害的重复，排除规则使它不重读历史。
    - `prune` 之后不再列出，游标保留、不再更新。
  - **重用**：在同一路径上新建 worktree，它沿用原游标。旧 tip 通常不是新 HEAD 的祖先，按历史改写重新扫描（§5.2）；已入库的 SHA 去重，关联不变（§5.4）。
  - **大小写**：
    - Windows 上，同一个存在的目录以不同大小写报告时，统一为一个身份。
    - macOS、Linux 上按字节比较，大小写不同就是不同的身份。git 报告的是创建 worktree 时记录的路径，通常不变。
    - Unicode 的组合方式（macOS 的 NFC / NFD）没有验证。
  - **持久化**：绝对路径与 `worktrees.path` 是同一类数据，保存之前做密钥扫描（下一条）。
- **固定读取结果**：
  - 扫描开始时一次取得所有 tip 的完整 SHA，之后的 git 读取只使用这些 SHA，不再按 ref 名称读取。
  - 这是固定下来的读取结果，**不是**数据库、所有 ref 与所有 worktree 的全局原子快照。
  - 扫描期间移动的 ref，由下一次扫描读取。
- **密钥扫描**（负责人决定 D7）：
  - 要写进游标的 ref 名称与分离 HEAD 的路径，在读取任何提交之前，都按 architecture §10.8 的规则扫描。
  - 任何一个命中时，**拒绝本次扫描**：不写提交、关联、`seq` 或游标，也不回显敏感值。
  - 错误分类沿用既有契约：architecture §10.2 第 4 条（命中即拒绝写入），以及 §6.1.2 对分支名的先例。
    - 返回 `invalid_input`（退出码 2）；
    - 错误只说明是哪一类名称（分支名或 worktree 路径）与哪条规则，并提示改名或移除；
    - 不新增隐私豁免。
  - 不静默跳过：只跳过一个 ref 会漏掉只在它上面的提交，又把扫描报告为完整。
  - 调用扫描的命令如何呈现这个错误，由 WP2.7 确定；它不能把这次扫描当作已完成。

### 5.2 扫描范围与严格读取（审查 050 B1、B2）

扫描开始时，在读事务中读取项目已有的游标集合 C。C 是计划开始时的集合，不混入本轮刚扫描的 ref。每个游标有自己的 `last_tip`；每个 ref R 有本轮固定的 tip T（§5.1）。

**第一步：确认正向 tip**。

- 所有 T 都是必需读取的对象。分支的 T 已由 `for-each-ref` 的 `objecttype = commit` 确认；分离 HEAD 的 T 用下一步的分类确认。
- 分类结果不是"存在的提交"时，本次扫描返回退出码 6。

**第二步：给旧 tip 分类**。

- 对 C 中所有不同的 `last_tip`（以及分离 HEAD 的 T），一次 `git cat-file --batch-check`，从 stdin 读入完整 SHA。按协议输出分类，不看 stderr：

  | `--batch-check` 的输出 | 再确认 | 分类 |
  |---|---|---|
  | `<sha> commit <size>` | — | **存在** |
  | `<sha> missing` | `git cat-file -e <sha>`（裸的完整 ID，不带 `^{commit}`）返回 1 | **可靠地不存在** |
  | `<sha> missing` | `cat-file -e` 返回 0 | **存在但不可读**（例如损坏的松散对象） |
  | 其他类型、其他输出，或任一命令的其他退出码 | — | **无法确认** |

- **存在但不可读**、**无法确认**：fail-closed，本次扫描返回退出码 6，不写入。
- 不使用 `cat-file -e <sha>^{commit}`：带剥离表达式时，对象不存在会在解析阶段失败（128），到不了存在性检查（实验 R4）。

**第三步：每个 ref 的范围**：

| 情形 | 判断 | 读取的提交 |
|---|---|---|
| R 有游标，`last_tip = T` | — | 无 |
| R 有游标，旧 tip **存在** | `git merge-base --is-ancestor <last_tip> <T>`：0 | `<last_tip>..<T>`（architecture §6.7 的增量） |
| 同上 | 1（历史被改写） | R 的全部历史 `<T>`，按 SHA 去重（architecture §6.7） |
| 同上 | 其他退出码 | 读取失败，退出码 6（旧 tip 已确认存在，所以这不是"旧 tip 不存在"） |
| R 有游标，旧 tip **可靠地不存在**（例如历史改写后被回收） | — | 按历史改写处理：R 的全部历史 `<T>` |
| R 没有游标（新的 ref，D4） | — | `<T>`，排除 C 中**存在**的所有 `last_tip`；可靠地不存在的旧 tip 不作排除项（范围变大，偏向多读） |
| 项目还没有任何游标（第一次扫描） | — | 每个 ref 的全部历史 |

**第四步：严格读取**：

- 每个有工作的 ref 一次 `git log -z --stdin --format=%H%x00%ct%x00%(trailers:key=Specmetry-Ref,valueonly,separator=%x2C)%x00%B`。
- stdin 中先写正向 tip T，再为每个排除项写一行 `^<完整 SHA>`。
  - 不写 `--not` 行（git 2.32 的 `--stdin` 不接受选项行，审查 050 N1）；
  - **不使用 `--ignore-missing`**：它不只作用于排除项，正向 tip 缺失时会让命令以 0 退出、没有输出（审查 050 B1；实验 R1）。
- 结果：
  - 退出 0：读取成功；没有输出就确实没有新提交。
  - 任何非 0：本次扫描失败，退出码 6，不写入，游标不变。这包括正向 tip 在第一步之后消失或不可读，排除项在第二步之后消失（实验 R2、R3）。
- 每条记录都要校验：SHA 为 40 位小写十六进制，时间为十进制整数。不合格时整个扫描失败。

**为什么这样不会漏读、也不会错误地推进游标**：

- 游标只在所有 ref 的严格读取都成功之后，才在写事务中推进（§5.5）。
- 读取的范围是"正向 tip 可达、且不被**已确认存在**的排除项可达"的全部提交，每一个都由 git 严格读取。
- 只有被确认存在、并且读取成功的排除项，才会缩小范围。
- 被判为"可靠地不存在"的旧 tip，只会让范围变大。
  - 如果这个判断在某种无法区分的 I/O 情形下有误，被放弃的排除项本身若是 T 的祖先，严格读取 T 时仍要读到它，会失败（退出码 6）。
  - 若不是祖先，它本来就不需要读取。
  - 所以误判只会多读，不会少读。
- 因放弃排除项而多读到的、此前没有入库的提交，如果首次满足入库条件，按这次的 `seq` 入库。这属于负责人已接受的"重扫后的首次入库"（§2）。

**新 ref**：

- 排除已有游标覆盖的历史（D4）：在已扫描的 `main` 上新建分支时，不回头重读 `main` 的全部历史，符合"常规增量扫描不主动回头"。
- 选择读取全部历史，会改变新 ref 带来的首次关联机会，不只是性能上的差别（审查 050 的 D4 裁定）。

**还没有任何编号**：

- 项目还没有任何需求或 AC 时，不扫描、不建立游标（D5），记为"没有扫描"。
- 这与"扫描了但没有匹配"不同，后者会建立游标（§5.5）。

**实验**（Git for Windows 2.56.0，Windows 11；隔离夹具；受控环境）：

| # | 情形 | 结果 |
|---|---|---|
| R1 | 正向 tip 不存在，stdin 读入 | 严格：**128**；带 `--ignore-missing`：**0，没有输出**（审查 050 B1 的陷阱） |
| R2 | 正向 tip 损坏（松散对象内容不是 zlib） | `log` **128** |
| R3 | 排除项 `^<不存在的 SHA>`，严格 | **128** |
| R4 | 不存在的 SHA：`merge-base --is-ancestor`；`cat-file -e <sha>^{commit}`；`cat-file -e <sha>` | 128；**128**；**1** |
| R5 | `--batch-check`：存在的提交、不存在的、一个树 | `commit 194`、`missing`、`tree 0`，都在一次调用中，退出 0 |
| R6 | 松散对象内容损坏 | `--batch-check`：**`missing`**（退出 0）；`cat-file -e`：**0**；`is-ancestor`：128 |
| R7 | 松散对象为空文件 | 同 R6 |
| R8 | 松散对象文件被删除；从来没有写入的 SHA | `--batch-check`：`missing`；`cat-file -e`：**1** |
| R9 | 历史改写并 `gc --prune=now` 之后的旧 tip | `--batch-check`：`missing`；`is-ancestor`：128 |
| R10 | 打包对象的数据被改写（索引完好；`gc` 写出了 commit-graph） | `--batch-check`：`commit 194`；`cat-file -e`：0；`is-ancestor`：0（commit-graph 不读对象）。没有被误判为不存在；严格的 `log` 读取该提交时才会发现 |
| R11 | 正常扫描：`c3` 与 `^c1` | 0，两个提交 |
| R12 | `c3` 与 `^c3` | 0，没有输出（确实没有新提交） |
| R13 | stdin 中的 `--not` 行 | 2.56 接受；2.32 不接受（审查 050 的静态核查），所以改用 `^<sha>` |

**结论**：

- `--batch-check` 的 `missing` 不足以证明对象不存在：损坏的松散对象也报告 `missing`（R6、R7）。
- 加上裸 ID 的 `cat-file -e`，可以用退出码区分"可靠地不存在"（1）与"存在但不可读"（0）。

**未验证**：

- Linux、macOS 与其他版本。
- 真实的部分克隆。按 PD-26 的门禁，不会按需抓取，预期本地缺失即为"不存在"。
- 对象目录或包文件没有读取权限、I/O 错误等情形下 `cat-file -e` 的结果。
- 真实的并发 `gc`。

**其他实验**（上一轮，仍然有效）：

| # | 情形 | 结果 |
|---|---|---|
| L1 | 七个提交，`log -z` 按 NUL 分隔 | 记录之间与字段之间都是 NUL，可以按 4 个一组切分 |
| L2 | trailer 行 `Specmetry-Ref: REQ-010, AC-010-1`、`specmetry-ref: REQ-011 REQ-012`、`SPECMETRY-REF:REQ-013` | 值为 `REQ-010, AC-010-1,REQ-011 REQ-012,REQ-013`：键不区分大小写，冒号后可以没有空格 |
| L3 | `Specmetry-Ref:` 行在倒数第二段，最后一段是普通文字 | trailer 的值为空：git 只认最后一段的 trailer，这一行仍在正文中 |
| L4 | 以 CRLF 结尾的说明 | trailer 的值为 `REQ-030` |
| L5 | `GOAL-1` 与 `REQ-040, nonsense, AC-040-1; REQ-041` | 原样返回值，由 Specmetry 自己切分（§5.3） |
| L6 | 说明中含 NUL | `git commit-tree` 拒绝；用原始对象写入的，`log` 在 NUL 处截断，不会打乱字段 |
| L10 | `for-each-ref`、`worktree list --porcelain -z` | 分支与 SHA；各 worktree 的 `HEAD` 与 `detached`、`prunable` 标记 |

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
- **入库**：至少有一条关联的提交才入库。说明中没有任何像编号的词的提交，不成为提交或关联的候选；游标仍按整次扫描的写事务更新（§5.5，审查 050 N2）。
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
   1. 重新读取游标。只要有任何一个与计划时不同（`last_tip` 改变、游标新出现或消失），本次**什么都不写**，返回"并发跳过"。这不是错误，下一次扫描会重做。这只保证本次不发布过期的计划，不表示 git 的 ref 也被锁定（D6）。
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
- **git 的失败**：以下任何一种都使本次扫描失败，退出码 6，不写入，游标不变：
  - 非 0 退出，不论正向 tip 还是排除项。例外只有 `merge-base --is-ancestor` 与裸 ID 的 `cat-file -e` 返回的 1：它们是协议规定的"否"（§5.2），其他退出码仍是失败。
  - 不合格的输出；
  - 旧 tip 或正向 tip 的分类为"存在但不可读"或"无法确认"（§5.2）；
  - git 不支持需要的选项（§13）；
  - 被取消。
- 错误只说明是哪个 git 子命令、退出状态是多少（§6.1.2），不依赖、也不回显 git 的错误文字。
- **密钥命中**：拒绝本次扫描，返回 `invalid_input`（退出码 2），不写入（§5.1，D7）。
- **数据库的失败**：沿用存储层既有的分类。
  - 打开数据库时的偶发失败见维护事项 M-14；
  - 写入阶段的 `SQLITE_BUSY` 见维护事项 M-15。
  - 两者都没有据此批准新的错误映射或重试。
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

// Clean reports whether the worktree is clean (data-model §1), read
// through a private git directory whose configuration Specmetry wrote, and
// submodule by submodule, so that no program the repository configures
// runs (§4.3).
func Clean(ctx context.Context, worktree string) (bool, error)

// Tip is a scanned ref: a local branch, or the detached HEAD of a worktree.
type Tip struct {
	Ref          string // "refs/heads/<name>", or "" for a detached HEAD
	WorktreePath string // a detached HEAD's worktree, as gitx.Canonical gives it
	Commit       string // full SHA-1
}

// Tips lists the local branches that point to commits and the detached
// HEADs of the non-bare worktrees; a git without `worktree list -z` is an
// environment error.
func Tips(ctx context.Context, worktree string) ([]Tip, error)

// Presence is what the object database says of a commit ID.
type Presence int

const (
	Present Presence = iota // a readable commit
	Absent                  // reliably not in the object database
)

// Classify classifies commit IDs (§5.2): Present or Absent; anything else
// (present but unreadable, another type, an unexpected answer) is an
// environment error.
func Classify(ctx context.Context, worktree string, ids []string) (map[string]Presence, error)

// IsAncestor reports whether ancestor, a Present commit, is an ancestor of
// commit; any answer but yes or no is an environment error.
func IsAncestor(ctx context.Context, worktree, ancestor, commit string) (bool, error)

// Commit is one commit read by Log. Trailers and Message are only in
// memory: callers never store, log or echo them.
type Commit struct {
	SHA      string
	Time     int64 // committer time, seconds, as git gives it
	Trailers string
	Message  string
}

// Log strictly reads the commits reachable from tip and from none of
// exclude (Present commits), newest first, calling fn for each; any
// object it cannot read is an environment error.
func Log(ctx context.Context, worktree, tip string, exclude []string, fn func(Commit) error) error
```

- 除 `worktree list` 之外，这些操作都经过 `runObjects`。`worktree list` 只读取 git 目录的布局。
- `Clean` 的私有 git 目录由 `gitx` 创建并删除，不写用户的仓库。
- 错误都是 `apperr.Environment`（退出码 6），不带 git 的输出。
- `gitx` 不知道编号、项目或数据库。

**扫描层**（建议 `internal/gitscan`）：

| 职责 | 说明 |
|---|---|
| `Mentions(trailers, message string) []Mention` | 纯函数：§5.3 的切词与 trailer 优先，结果为编号与来源 |
| 计划 | 读游标，确认是否已有编号（D5） |
| 读取 | 调用 `gitx`：tip、密钥扫描（D7）、分类、范围、严格日志；得到候选提交 |
| 写入 | §5.5 的写事务：比较游标、去重、解析编号、分配 `seq`、写入 |
| `Scan(ctx, store, project, worktree) (Result, error)` | 编排以上步骤。`Result` 只含计数：读取的提交数、新入库数、是否并发跳过 |

- 扫描层不输出、不节流，也不决定在哪些命令中调用。
- 关联只由这个唯一的写入入口写出：已入库的 SHA 直接跳过（§7 的不变量）。

## 7. 迁移设计（`0007_commits`，WP2.3 实现）

本节只是设计，不是迁移 SQL。写法沿用既有迁移：

- 26 位 ID；
- `(id, project_id)` 唯一，同项目内的外键用 `DEFERRABLE INITIALLY DEFERRED`；
- 只追加表用触发器阻止 UPDATE、DELETE 与 REPLACE（维护模式除外），沿用每个连接的 `recursive_triggers`。

**`commits`**（只追加）：

| 列 | 定义 |
|---|---|
| `id` | `TEXT PRIMARY KEY`，长度 26 |
| `seq` | `INTEGER NOT NULL`；同一次扫描新写入的提交共用，**不唯一** |
| `project_id` | `TEXT NOT NULL`，引用 `projects` |
| `sha` | `TEXT NOT NULL`，40 位小写十六进制（TD-26） |
| `committed_at` | `INTEGER NOT NULL`，提交者时间，**UTC Unix 毫秒**（data-model §1 的通用约定；git 给出的秒 × 1000），只用于显示 |

- 约束：`UNIQUE (project_id, sha)`、`UNIQUE (id, project_id)`。
- 索引：`(project_id, seq)`。

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
- **`BEFORE INSERT` 触发器**（审查 050 §7 的建议，采用）：`target_id` 必须是同一项目中、对应 `target_type` 的条目。应用仍按 `project_id` 查询。
- 索引：`(project_id, target_type, target_id)`，供进度查询使用。

**必需的不变量：关联只在它的提交首次入库的事务中写入**：

- 只追加的约束本身不能禁止以后再 INSERT 新的关联，所以这条不变量由以下几点共同保证：
  - 唯一的扫描写入入口；
  - 已入库的 SHA 直接跳过；
  - 回归测试：以后的扫描不会给旧 SHA 补写关联。
- 不新增事务令牌之类的模型。也不以"父提交的 `seq` 等于当前计数器"作为同事务的证明，因为计数器在两个事务之间可能不变。

**`git_scan_cursors`**（受控可变）：

| 列 | 定义 |
|---|---|
| `project_id` | `TEXT NOT NULL`，引用 `projects` |
| `kind` | `TEXT NOT NULL`，`branch` / `detached` |
| `name` | `TEXT NOT NULL`：分支为完整的 ref 名称；分离 HEAD 为 `gitx.Canonical` 给出的 worktree 路径（D2，§5.1） |
| `last_tip` | `TEXT NOT NULL`，40 位小写十六进制 |
| `scanned_seq` | `INTEGER`，可为 NULL（§5.5） |
| `scanned_at` | `INTEGER NOT NULL`，UTC Unix 毫秒，只用于显示与节流 |

- 约束：`PRIMARY KEY (project_id, kind, name)`。
- 只由扫描的写事务更新，没有只追加触发器。
- 分支被删除、worktree 被移走后，游标保留，它的 `last_tip` 继续用于新 ref 的排除（§5.2）。

**事务与 `seq`**：

- 一次扫描至多一个写事务，至多分配一个 `seq`。
- 分配 `seq` 的写入同时包含这次的全部 `commits`、`commit_links`，以及游标的更新。
- 没有新入库的提交时，写事务只更新游标，不分配 `seq`，不改变 `meta.seq_counter`。

**迁移测试**（WP2.3 实现）：

- 跨项目的 `target`、类型不符、重复的关联；
- 维护模式与 `purge` 的外键；
- UPDATE、DELETE、REPLACE 都被拒绝；
- 以后的扫描不给旧 SHA 补写关联。

**保留与清除**：

- `commits`、`commit_links` 是 C2 数据，保存到 `purge`（data-model §10）。
- 游标随项目清除；`purge` 与备份在 WP2.8 中覆盖这三张表。

## 8. 决定与待决事项

| # | 内容 | 状态 |
|---|---|---|
| **D1** | `clean` 定义加入 `--ignore-submodules=none` | **负责人已决定（2026-10-18）**。语义写入 data-model §1。读取方式另由 §4.3 解决（B3）；这个参数本身不解决外部过滤器的问题 |
| **D2** | 分离 HEAD 的游标身份 | **负责人已决定**：保留所有 worktree，按规范化路径识别；移动、删除、重用、大小写与规范化的行为见 §5.1 |
| D3 | 仓库根的记录 `{".", "040000", "tree", 根树}` | 审查 050 认可为工程细化；保留原 `path`，补测试 |
| D4 | 新 ref 的范围：排除计划开始时已有游标覆盖的历史 | 审查 050 认可方向；排除项只用**已确认存在**的旧 tip，读取严格（§5.2，B1、B2） |
| D5 | 还没有编号时不扫描 | 审查 050 认可；"没有扫描"与"扫描了但没有匹配"分开（§5.2、§5.5） |
| D6 | 并发时整次不写 | 审查 050 认可。只保证本次不发布过期的计划，下次重扫；不声称 git 的 ref 也被锁定，也不声称另一次扫描一定覆盖了同样的 tip |
| **D7** | ref 名称或路径命中密钥规则 | **负责人已决定**：拒绝本次扫描，不写任何内容，不回显；`invalid_input`（退出码 2），不新增豁免（§5.1） |
| D8 | `ls-tree` 退出 0 但输出不可信 | 审查 050 认可：按读取失败处理（退出码 6） |

**审查 050 的阻塞项**：

| # | 问题 | 修订 |
|---|---|---|
| B1 | `--ignore-missing` 会吞掉必需读取的正向 tip | 不再使用；正向 tip 严格读取；排除项只用已确认存在的旧 tip，写成 `^<sha>`（§5.2） |
| B2 | 旧 tip 缺失的退出码判断不成立 | 三态分类：`--batch-check` 加裸 ID 的 `cat-file -e`；不使用 `^{commit}`；不确定时 fail-closed（§5.2） |
| B3 | `status` 会执行 clean / process 过滤器 | 私有 git 目录加逐个子模块的读取，调用之前就保证不执行程序（§4.3） |

**其他待决或需要说明的事项**：

- 跨项目的同名编号无法区分（§5.3），这是现有格式的限制，不在 MVP 中扩展。
- 第一次扫描大型仓库的耗时，以及 `Clean` 复制索引的耗时，都没有基准。实现时测量，必要时另立事项。
- 维护事项 M-14、M-15 未解决，都与扫描所在命令打开或写入数据库有关，各自单独处理（§5.6）。
- **低于 2.36 的 git 上的分离 HEAD**（§13）：推荐保持 fail-closed（退出码 6）。按行解析无法可靠切分含换行的路径；若负责人以后要求兼容这些版本，须另行设计能证明安全的读取方式，并经审查。

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

**生命周期与扫描**（§5.1、§5.2、§5.4 ~ §5.6）。L1 ~ L3 的前提是：项目已有其他需求或 AC，所以 D5 不会跳过整次扫描（审查 050 N2）。

| # | 情形 | 期望 |
|---|---|---|
| L1 | 项目已有 REQ-001；提交 X 提到 REQ-020，扫描时 REQ-020 还不存在 | X 不入库，游标前进；之后的常规增量扫描不再读取 X |
| L2 | X 已入库（关联 REQ-001），后来新增的 REQ-020 也出现在 X 的说明中 | 不补写 REQ-020 的关联 |
| L3 | L1 之后分支被改写，重扫时 X 仍在历史中，REQ-020 已存在 | X 以这次扫描的 `seq` 入库 |
| L4 | 范围中没有新的候选 | 不分配 `seq`，`meta.seq_counter` 不变；游标更新；`scanned_seq` 不变 |
| L5 | `main` 与 `feature` 共用的提交提到 REQ-001 | 只入库一次，一条关联 |
| L6 | 从已扫描的 `main` 新建分支并提交一个新提交 | 只读取新提交（D4） |
| L7 | `feature` 被 reset 后提交新提交（MVP-WT-04） | 旧 tip 存在且不是祖先：重扫 `feature`；已入库的不重复；新提交入库 |
| L8 | 未登记的 worktree 处于分离 HEAD，其上的提交提到 REQ-001 | 入库；游标的身份为该 worktree 的规范化路径（D2） |
| L9 | 写入时游标已被另一个进程改变 | 什么都不写，结果为并发跳过 |
| L10 | 扫描中某个 git 命令失败 | 退出码 6，什么都不写 |
| L11 | 分支名或分离 HEAD 的路径命中密钥规则 | 拒绝整次扫描：退出码 2，不写提交、关联、`seq` 或游标，不回显（D7） |
| L12 | 项目还没有需求或 AC | 不扫描、不建立游标（D5） |
| L13 | 说明中有 canary 字符串（标题、正文、trailer 的值） | 数据库、WAL、日志与错误中都找不到（MVP-EV-11） |
| L14 | 正向 tip 在分类之后变得不可读（模拟：tip 对象被删除或损坏） | `log` 非 0：退出码 6，游标不前进，下一次仍读取它（B1；实验 R1、R2） |
| L15 | 游标的旧 tip 在历史改写后被回收（可靠地不存在） | 按改写处理：严格读取新 tip 的全部历史；已入库的去重（B2；实验 R8、R9） |
| L16 | 游标的旧 tip 是损坏的松散对象（存在但不可读） | 退出码 6，不写入（B2；实验 R6） |
| L17 | 新 ref 的排除项中有一个可靠地不存在的旧 tip | 不作排除项，范围变大，严格读取；其余照常（B1） |
| L18 | 排除项在分类之后消失 | `log` 非 0：退出码 6，不写入（实验 R3） |
| L19 | 正常扫描：分支有新提交 | `log` 退出 0；新入库与游标在一个事务中写出 |
| L20 | 移动一个分离 HEAD 的 worktree；在原路径上新建 worktree | 前者成为新游标、按新 ref 读取；后者沿用原游标、按改写重扫（D2） |

**输入路径与工作区状态**：

- 输入路径见 §4.2 的 T1 ~ T23，工作区状态见 §4.3 的 C1 ~ C11，实现时逐项写成测试。
- 此外，仓库根的记录保留原 `path`（D3）。

## 10. 验收映射

| 验收项 | 内容 | WP2.3 的部分 | 其余部分 |
|---|---|---|---|
| MVP-WT-04 | 每个 ref 的扫描、改写后重扫、不重复 | 扫描层的集成测试（L5 ~ L7） | 命令中的触发：WP2.7 |
| MVP-EV-11 | 提交的标题中有 canary → 只保存 SHA 与时间 | 扫描层的集成测试（L13） | — |
| MVP-EV-10 | 输入记录 | `gitx.TreeEntry`（T1 ~ T23） | 在执行中记录：WP2.5 |
| MVP-EV-08、EV-09、WT-03 | 工作区状态与代码上下文 | `gitx.Clean`（C1 ~ C11） | 在执行与命令中使用：WP2.5、WP2.4 |
| MVP-EV-07 | 只追加表的触发器 | `commits`、`commit_links` 的迁移测试（§7） | — |
| MVP-EV-14 | 编号匹配的规则 | `Mentions` 的单元测试（M1 ~ M11）与集成测试 | — |
| MVP-EV-15 | 关联的生命周期 | 集成测试（L1 ~ L3，前提见 §9） | — |
| MVP-EV-16 | 扫描的写入、严格读取与失败 | 集成测试（L4、L9、L10、L14 ~ L19） | — |
| MVP-EV-17 | 输入路径的读取机制（TD-23） | `gitx` 的集成测试 | 在执行中的退出码与不写记录：WP2.5 |
| **MVP-EV-18**（新增） | 工作区状态的读取不执行仓库配置的程序；子模块变化；保守的差异 | `gitx.Clean` 的集成测试（C1 ~ C11） | — |
| MVP-WT-06 | 扫描的范围：分离 HEAD 与其身份（D2）、新 ref、密钥命中的拒绝（D7）、没有编号时不扫描（D5） | 集成测试（L6、L8、L11、L12、L20） | 命令如何呈现扫描被拒绝：WP2.7 |
| §6.5 进度中的关联提交 | — | — | 已由 WP2.1 覆盖 |
| CX | 没有直接针对提交关联的反例项 | — | E2E-2 的"追加带 trailer 的提交"属于 WP2.9 |

- 只看"git 返回非 0 就拒绝"，发现不了 B1 的假成功。所以 L14 必须断言**游标没有前进**，不只断言退出码。

## 11. 实现测试计划

- **单元**：
  - `Mentions`：M1 ~ M11，以及边界字符、CR、非 ASCII、空输入。
  - `ls-tree`、`log`、`worktree list`、`--batch-check` 的输出解析：合格与不合格的输入，不合格时返回错误。
  - 范围规划与分类：给定游标、分类结果与祖先判断的结果，覆盖 §5.2 的每一行。
- **`gitx` 集成**（`testutil` 的 git 夹具，临时仓库，三个平台）：
  - T1 ~ T23、C1 ~ C11、R1 ~ R13。
  - L1、L3 状态、损坏与缺失的对象，通过删除或改写松散对象、改写包数据来模拟。
  - **过滤器不执行**：
    - clean、process 过滤器与子模块中的过滤器，都配置成运行时写标记的夹具程序。
    - 断言 `Clean` 运行之后没有标记，结果符合 §4.3。
    - 对照组直接运行 `status`，断言有标记，证明夹具确实会触发。
  - **能力门禁**：沿用 `SPECMETRY_TEST_GIT_OBJECTS=unsafe` 的做法（维护事项 M-9），确认新命令在不具备能力时都在运行前被拒绝。
  - **兼容性**：
    - 不认识 `worktree list -z` 的 git，用夹具中的替身 git 模拟 usage 错误，断言返回退出码 6、不按行解析。
    - stdin 中的 `^<sha>` 行。
- **存储**：
  - 迁移 0007 的约束、唯一性、外键、`commit_links` 的 `BEFORE INSERT` 检查与只追加触发器（MVP-EV-07）；
  - 没有新入库时 `seq_counter` 不变。
- **扫描层集成**（真实 git 与临时数据库）：
  - L1 ~ L20；
  - 两个进程同时扫描时的并发跳过（`testutil.HelperCommand`）；
  - 扫描中途注入 git 失败；
  - canary 在数据库、WAL、日志与错误中都不出现；
  - 密钥命中时，所有表的行数与 `seq` 计数器都不变（"不写入"的定义）。
- **不在 WP2.3**：命令级的验收（WP2.7）、E2E（WP2.9）。

## 12. M-7 的处理顺序

- M-7（部分成功的 JSON 输出只检查以 `{` 开头）必须在新增输出接口之前修复：WP2.4（`claim`）、WP2.5（`check run`）、WP2.7（`status`、`gaps`）。
- WP2.3 不新增输出接口，所以 M-7 不是 WP2.3 的开工条件。
- 建议在本规范冻结之后，作为单独的 PR 修复。它可以与 WP2.3 的实现并行，但须在 WP2.4、WP2.5、WP2.7 任何一个开工之前完成。
- M-7 保持未关闭，直到那个 PR 经审查合并。

## 13. git 版本与选项的兼容（审查 050 N1）

**三者的关系**：

- **最低版本 2.32**（architecture §6.1.2）只是配置隔离的下限：低于它的 git 不认识 `GIT_CONFIG_GLOBAL`，返回退出码 6。它不保证每一项对象读取都能完成。
- **PD-26** 另外要求实际的对象读取能力：
  - 由探测决定，不按版本号推断。
  - 官方源码构建的 2.32.0 被判为"会抓取"，所以在它上面，所有读取对象的命令都在运行之前返回退出码 6，这里列出的读取命令都不会运行。
- **本文**不提高最低版本，也不推翻 PD-26。一个通过了 PD-26 探测、却不支持某个选项的 git，在用到该选项的操作上 fail-closed（退出码 6），由明确的能力判断处理，不解析 git 的错误文字。

| 命令或选项 | 用途 | 2.32（静态：审查 050 引用的 v2.32.0 源码与文档） | 2.43（静态，审查 050） | 2.56（本文实验） | 处理 |
|---|---|---|---|---|---|
| `worktree list --porcelain -z` | 分离 HEAD | **不支持**（既有 `gitx.mainWorktree` 的注释记为 2.36 加入，未运行验证） | 支持 | 支持（L10） | 不支持时（usage 错误，退出 129）扫描返回退出码 6；不退回按行解析 |
| `log --stdin` 中的 `--not` 行 | 排除项 | **不接受** | 接受 | 接受（R13） | 不使用；改写 `^<sha>` 行 |
| `log --stdin` 中的 `^<sha>` 行 | 排除项 | 未在 2.32 上运行 | 未运行 | 支持（R11 ~ R12） | 实现时在 CI 的各平台上验证；不支持时退出码 6 |
| `%(trailers:key=…,valueonly,separator=…)` | trailer 的值 | 支持（审查 050 核对文档） | 未单独核对（2.32 已支持） | 支持（L2 ~ L5） | — |
| `ls-tree -z --full-tree`，全局 `--literal-pathspecs` | 输入路径 | 支持（审查 050 核对源码） | 未单独核对（2.32 已支持） | 支持（T1 ~ T23） | — |
| `cat-file --batch-check`、裸 ID 的 `cat-file -e`、`merge-base --is-ancestor`、`for-each-ref %(objecttype)` | 分类、祖先、ref | 未逐一核对引入版本 | 未运行 | 支持（R4 ~ R10） | 实现时核对；任何意外的退出码都是退出码 6 |
| `rev-parse --path-format=absolute` | `Clean` 的私有目录 | 未核对（按发布说明是 2.31 加入，未运行验证） | 未运行 | 支持（C1 ~ C11） | 实现时核对；不支持时退出码 6 |
| `status --porcelain -z --ignore-submodules=all`，`config --list --local --includes -z`，`--git-dir`、`--work-tree` | `Clean` | 未逐一核对 | 未运行 | 支持（C1 ~ C11） | 实现时核对 |

- 只有 2.56 是实际运行的结果；2.32、2.43 的列来自审查 050 的静态核查。
- **后果**：一个低于 2.36、却通过了 PD-26 探测的 git（例如回移了 `GIT_NO_LAZY_FETCH` 的发行版构建，是否存在未核实），每次提交扫描都会因 `worktree list -z` 返回退出码 6。
  - 这是 fail-closed 的拒绝，不是提高最低版本：初始化、基准与其他不扫描提交的操作不受影响。
  - 调用扫描的命令（`status`、`ingest`、`sync`）如何呈现这个错误，由 WP2.7 确定（§5.1 对 D7 的同一要求）。
  - 是否另外提供不依赖 `-z` 的兼容读取，列为待决事项（§8），本文不自行决定。
- 实现时在 CI 的三个平台上运行全部用到的命令，并为不支持的选项写出拒绝路径的测试（§11）。
