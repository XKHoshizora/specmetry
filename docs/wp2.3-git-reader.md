# WP2.3 Git Reader 规范

> 状态：**已冻结**（2026-10-19，产品负责人批准；审查 057 技术 GO，见 [review-resolution.md §67](review-resolution.md)）。实现中的细化写在 §14（审查 058 之后，经审查 059 复审；其中 §14.3 已按负责人 B1-D1～D4 的批准调整（审查 060 之后，§71），实现结果待独立复核；不是审查 057 的审查对象；业务定义不变，§14.3 的局部安全保证按 §71 明确调整）。此前按审查 050、052、055（都为 NO-GO）修订。负责人已决定 D1、D2、D7（2026-10-18），以及 P1、P2、P3 与 `worktree list -z` 的能力要求（2026-10-19；§2、§8）；其余没有获批的草案保持原状。本文写出 WP2.3 的读取机制（TD-23）、提交关联规则（M2-Q7 / TD-17）、关联的生命周期（审查 044 N1）、`internal/gitx` 的新接口、扫描层的职责、迁移设计、正反例、验收映射与测试计划。
>
> 本文不是实现。WP2.3 的实现获得条件授权（2026-10-19）：本规范的 PR 合并、合并核验与对应的 master push CI 全部通过之后开工，实现以冻结的本文为依据，另开 PR。
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

**第三轮决定**（2026-10-19，审查 055 之后；记录见 [review-resolution.md §66](review-resolution.md)）：

7. **P1**：不可判定使用 `unknown`，不能当作 `clean`（§4.3）。
8. **P2**：含稀疏目录条目的索引一律不可判定，安全门禁放在任何可能展开索引的命令之前；不扩大到所有稀疏检出（§4.3 第 2 步）。
9. **P3**：MVP 暂不支持浅克隆的提交扫描（§5.2"浅克隆"）：
   - 需要扫描时，以只读方式明确检测；是浅克隆，或检测无法确认，都返回环境错误（退出码 6），不写提交、关联、`seq` 或游标，不伪装成成功的空扫描；
   - 不自动拉取，也不修改浅边界；
   - 触发扫描的命令应明确呈现这一限制（WP2.7）；
   - 将来支持浅克隆时，另行设计 deepen / unshallow 与游标的生命周期。
10. **`worktree list -z` 的能力**：缺少这一能力时，MVP 的扫描返回环境错误（退出码 6），不增加按行解析的回退；这是扫描的能力要求，其他命令的支持范围不变（§5.1、§13）。

这些是范围的限制，不是允许误报 `clean`、修改原对象或放行损坏存储的风险豁免。

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

### 4.3 工作区状态（审查 050 B3、审查 052 B3-R、负责人决定 D1）

**语义**（data-model §1，D1）：`clean` 指以下命令在 worktree 根目录中的输出为空：

```
git status --porcelain --untracked-files=all --ignore-submodules=none -- . ':(exclude).specmetry'
```

- `--ignore-submodules=none`：仓库配置（例如 `diff.ignoreSubmodules=all`）不能隐藏子模块的变化。
- 这条命令在 Specmetry 的受控环境中定义语义（architecture §6.1.2）：没有系统与全局配置，只有仓库自己的（local 与 worktree）配置。例如只写在全局配置中的 LFS 过滤器，不在定义之内。
- 这个命令只定义"什么算变化"，**不是读取方式**。

**读取方式**：Specmetry **不直接在用户的仓库上运行这条命令**。原因：

- `status` 判断只改了修改时间的文件是否真的被修改时，会经过仓库属性所选的 `filter.<name>.clean` / `process` 转换内容，从而执行仓库配置的程序（审查 050 B3；实验 C1）。
- 带 `--ignore-submodules=none` 时，它还会在每个子模块中启动一个 `git status`，同样会执行子模块配置的过滤器（实验 C9）。
- 受控环境、关闭 FSMonitor 与对象读取门禁，都不能阻止这些程序执行。

**读取步骤**（审查 055 B3-R2 之后的顺序：门禁在任何可能展开索引的命令之前）：

1. **私有的 git 目录**：Specmetry 在自己的私有临时目录中建立一个 git 目录，内容如下：
   - `HEAD`：worktree 当前的 HEAD（完整 SHA）。分支还没有提交时，写成指向该分支的符号引用 `ref: refs/heads/<分支>`（unborn），`status` 照常运行；不能省略 `HEAD`，否则 `status` 返回 128（审查 052 N1；实验 N1-u）。
   - `objects/info/alternates`：指向仓库的对象目录。**它不是只读保护**：私有目录中的 git 一旦需要写对象，会更新原仓库中同一对象文件的修改时间（审查 052 B3-R2）。所以只允许已验证不写对象的命令与布局，并由第 2 步的门禁先排除稀疏索引。
   - `index`：worktree 索引文件的副本；拆分索引时，同时复制 `sharedindex.*`。
   - `info/exclude`、`info/attributes`、`info/sparse-checkout`：公共目录中这些文件的副本。
   - `config`：只有白名单中的键。
     - 白名单：
       - `core.autocrlf`、`core.eol`、`core.safecrlf`、`core.ignorecase`、`core.precomposeunicode`；
       - `core.protecthfs`、`core.protectntfs`、`core.symlinks`、`core.filemode`、`core.checkstat`、`core.trustctime`；
       - `core.excludesfile`、`core.attributesfile`、`core.sparsecheckout`、`core.sparsecheckoutcone`、`index.sparse`；
       - `core.longpaths`、`core.fscache`、`core.checkroundtripencoding`。
     - 另外固定写入 `core.repositoryformatversion = 0`、`core.bare = false`、`core.untrackedCache = false`。
     - 值取自 `git config --list --local --includes -z`；仓库启用了 `extensions.worktreeConfig` 时，再读 `--worktree` 的值，按同一份白名单过滤，worktree 的值优先。读取配置的命令同样在受控环境中运行；这些路径值与原配置都不写入日志或错误。
     - 没有任何 `filter.*`、`core.fsmonitor`、`core.hooksPath`、`include*`、`extensions.partialClone` 或远端配置。
   - 建立私有目录只运行不读取索引的 git 命令（`rev-parse` 取路径、`config` 读配置）。
2. **稀疏门禁**（审查 055 B3-R2；负责人决定 P2）：在任何 git 命令读取这个索引副本**之前**，Specmetry 自己解析它，不运行 git：
   - 解析索引文件的格式（版本 2、3、4 的条目与扩展；末尾的 SHA-1 校验和，全零表示 git 省略了它）。有 `link` 扩展（拆分索引）时，同样解析它引用的 `sharedindex.<哈希>`。
   - 有 mode 为 `040000` 的条目（稀疏目录条目），或者有 `sdir` 扩展：这是稀疏索引，结果为 `unknown`（P1、P2）。这个仓库，以及它的子模块，都不再运行任何 git 命令。
   - 不能完整解析（不是索引文件、未知的版本、截断、校验和错误、扩展越界）：环境错误（退出码 6），同样不再运行任何命令。
   - 不用 git 做这个门禁的原因：git 读取稀疏索引时，只要工作区中有与稀疏目录同名的目录，就会展开索引并更新原对象的修改时间，连 `ls-files --sparse --stage` 也是这样（实验 X2 ~ X4）；只有另外设置 `sparse.expectFilesOutsideOfPatterns=true` 时它才不展开（实验 X8）。那样的门禁依赖多项配置与 git 的版本，直接解析文件没有这种依赖。
   - 没有稀疏目录条目的稀疏检出（只有 skip-worktree 标记）不受这一条限制，照常读取（实验 S11）。P2 不扩大到所有稀疏检出。
3. **超级项目的读取**：在私有目录上运行，都经过对象读取门禁（PD-26）：
   - `git --git-dir=<私有目录> --work-tree=<worktree> status --porcelain -z --untracked-files=all --ignore-submodules=all -- . ':(exclude).specmetry'`。`--ignore-submodules=all` 使 git 不进入子模块，也不启动子进程（实验 C9：子进程数为 0）。
   - 需要列出索引时只用 `ls-files -z --sparse`。
   - （实现细化，待复审：`ls-files`、`diff-files` 与 `status` 使用相同的 pathspec，见 §14.6；另以 `diff-index --cached` 比较索引与 HEAD 的 gitlink，见 §14.1。）
   - 第 5 步的 `diff-files` 必须带 `--ignore-submodules=all`：不带时，它会读取子模块的索引；子模块是稀疏索引时，这会更新子模块原对象的修改时间（实验 X6）。
4. **子模块**：
   - 对私有索引中每个 `160000` 条目，如果 worktree 中这个路径有 `.git`（已初始化）：
     - 先用受控的 git 元数据解析出它实际的 git 目录（`git -C <子模块路径> rev-parse --absolute-git-dir`）。`.git` 常常是指向 `.git/modules/<名称>` 的 gitfile，不能当作目录直接使用：审查 052 在它的环境中直接传 gitfile 得到 128，本文的环境中得到 0（实验 N1-g），行为不可依赖。
     - 用 `git --git-dir=<解析出的 git 目录> rev-parse --verify HEAD` 取得子模块的 HEAD，与 gitlink 不同即为 `dirty`。
     - 再对子模块按第 1 ~ 6 步读取：先建立它的私有目录，**先过它自己的稀疏门禁**，再运行它的 `status` 等命令；它的配置与驱动按它自己的 local 配置判断。
   - 没有初始化的子模块不读取，与 `status` 相同。
   - （实现细化，待复审：gitlink 的路径不存在或不是目录时为 `dirty`，见 §14.1。）
   - 每一次 git 调用都经过对象读取门禁与受控环境；私有目录不是跳过门禁的理由。
5. **过滤器下不可判定的情形**（审查 052 B3-R1）：不执行程序就无法得出与定义一致的结果时，结果为 `unknown`（P1）：
   - 驱动集合 D：仓库 local（及 worktree）配置中定义了 `filter.<名称>.clean`、`filter.<名称>.process`，或 `filter.<名称>.required = true` 的名称。D 为空时跳过本条。
   - 受过滤的路径 F：私有目录中 `ls-files -z --sparse` 的路径，经 `check-attr -z --stdin filter`，属性值属于 D 的那些。
   - 需要比较内容的路径：F 中满足以下任一条件的路径：
     - 在私有目录的 `diff-files --name-only -z --ignore-submodules=all` 中列出（stat 与索引不同，或是 racy 并且原样内容不同）；
     - racy：工作区文件的修改时间不早于原索引文件的修改时间。`diff-files` 对 racy 条目只做原样比较，可能不列出（实验 F7），所以这一条单独判断。
   - 存在需要比较内容的受过滤路径时，结果为 `unknown`。否则，F 中的路径 stat 与索引一致、也不 racy，定义中的 `status` 也不会比较它们的内容，私有目录的结果与定义一致（实验 F3、F5、F6）。
6. **结果**：
   - 第 2 步或第 5 步判定的不可判定：`unknown`（负责人决定 P1，2026-10-19）。它不是 `clean`，执行因此不合格。
   - 否则：超级项目与所有已初始化子模块都没有变化为 `clean`，有任何变化为 `dirty`。
   - 任何一步失败（复制失败、索引不能解析、git 非 0 退出、输出不合格）时，不报告 `clean`：返回环境错误（退出码 6），或者由调用方按其契约记为 `unknown`（检查执行在 WP2.5 中决定）。

**这种读取方式的保证与边界**：

- **调用之前就保证不执行程序**：git 读取的配置是 Specmetry 刚写出的白名单副本，其中没有任何会启动程序的设置。
  - 用户仓库的配置在这之后被修改，也不影响这次读取。
  - 这不依赖"仓库可信"，不依赖事后检测程序是否执行，也没有放宽"不执行外部程序"的规则。
  - 实验 C1 ~ C11、F1 ~ F7 中，私有目录的读取都没有执行过滤器。
- **不报告错误的 `clean`**：
  - 审查 052 指出，"差异只会把 `clean` 变成 `dirty`"不成立：取消过滤器后，原样内容相同不等于过滤后相同（实验 F1、F7 是反方向，定义下为 `dirty`、旧方案为 `clean`）。**本文撤回这一保证。**
  - 第 5 步的规则下，F1、F7 都判为不可判定，不报告 `clean`；F2、F4 定义下为 `clean`，也判为不可判定（保守）。
  - 这只覆盖实验中的情形与本文列出的判断依据，不是对所有过滤器配置的证明。
- **不修改用户的仓库**：
  - 索引与配置都是副本；`GIT_OPTIONAL_LOCKS=0` 另外保证不写这些副本的锁。
  - 原仓库对象文件的**内容与修改时间**：在普通索引、拆分索引、untracked cache、没有稀疏索引的稀疏检出中，本文整套命令（`status`、`ls-files`、`check-attr`、`diff-files`）执行前后都没有变化（实验 S12 ~ S14、S11、S19）。稀疏索引由第 2 步的门禁在这些命令之前排除（实验 S16 ~ S18b）。
  - 这依据的是这些布局下的实验，不是操作系统层面的只读保证。新的布局或命令须先做同样的实验。
- **临时目录的生命周期**（审查 052 N2）：
  - 私有目录建在受数据目录保护的 `tmp/git-reader-v3/`，按 §14.3／决定 §71、§72 的合作协议使用。名字只用于识别候选，不能据前缀、年龄或 PID 接管遗留。
  - 正常结束删除自身临时内容；失败／取消也尝试按清单清理。发现未知、额外或身份不符时保留对象并返回环境错误（退出码 6），清理失败同样不能报告 `clean`。
  - 下一次租约只清理有效 v3 owner、完整创建清单和已确认消费者结束记录能证明安全的遗留；未确认结束、缺失 owner 或未知对象保留并失败。v1 顶层及 v2 子树均不接管。
  - 它不属于项目事实，不进入日志、错误输出或备份。成功返回后只可保留无项目内容的最小协议锁／标识；异常遗留按 D3 保留并报错，不是保存源码或完整配置的许可（architecture §10、决定 §71）。
- **没有关闭的边界**：
  - `assume-unchanged`、`skip-worktree` 是用户在索引上的标记，`status` 的参数无法覆盖，这里也不处理。
  - `core.autocrlf` 等内置转换的结果与定义相同（同一份白名单配置）。

**实验**（隔离夹具；受控环境；Git for Windows 2.56.0 / Windows 11，另在 WSL Ubuntu 的 git 2.43.0 上重做了 F、S、N1，结果相同）：

审查 050 之后的实验（标记程序只在原草案的对照命令中允许运行）：

| # | 情形 | 原草案的命令 | 私有目录 |
|---|---|---|---|
| C1 | `f`、`g` 属性指定过滤器，提交之后才配置过滤器，只改修改时间 | clean 与 process 过滤器**都执行了**，命令失败 | 没有标记；输出为空 |
| C2 ~ C6 | 修改、未跟踪、被忽略、已暂存、删除 | — | 与定义一致；没有标记 |
| C7 | 提交时就有会改写内容的过滤器，只改修改时间 | 过滤器**执行了**，结果 `clean` | ` M f`；没有标记（这一行的"保守"解释已撤回，见 F2） |
| C8 | 普通文件只改修改时间 | — | 空 |
| C9 | 已初始化的子模块中有过滤器 | 启动了 **2 个**子进程，过滤器**执行了** | 子进程 0 个；没有标记 |
| C10、C11 | 子模块的提交与 gitlink 不同；另加 `diff.ignoreSubmodules=all` | ` M sub` | `dirty`；没有标记 |

审查 052 之后的实验。参考结果是定义中的 `status`（受控环境，过滤器为合成的 `sed s/raw/canonical/g`，只在参考命令中运行）：

| # | 情形 | 参考 `status` | 私有目录的 `status` | 第 5 步的判定 |
|---|---|---|---|---|
| F1 | 审查 052 的第一行：提交时没有驱动（索引为 raw），之后才配置驱动；stat 改变 | ` M f`（`dirty`） | 空（**错误的 `clean`**） | 不可判定 |
| F2 | 审查 052 的第二行：先配置驱动（索引为 canonical），工作区为 raw；stat 改变 | 空（`clean`） | ` M f` | 不可判定（保守） |
| F3 | 同 F2，stat 与索引一致、不 racy | 空，过滤器没有运行 | 空 | `clean` |
| F4 | 同 F3，但条目 racy | 空，过滤器运行了 | ` M f` | 不可判定（保守） |
| F5 | 属性指定的驱动没有在 local 配置中定义；stat 改变 | 空 | 空 | `clean` |
| F6 | 有 local 驱动，只修改了不受过滤的文件 | ` M plain` | ` M plain` | `dirty` |
| F7 | F1 的 racy 版本：索引为 raw，刷新后才配置驱动，条目 racy | ` M f`（`dirty`） | 空（**错误的 `clean`**）；`diff-files` 也不列出 | 不可判定（只有 racy 规则能发现） |

- 所有 F 行中，私有目录的读取都没有执行过滤器。

原对象文件的内容与修改时间（每次之前把原 `.git` 下所有文件设为旧的修改时间，之后比较哈希与修改时间）：

| # | 布局与命令 | 原 `.git` 的变化 |
|---|---|---|
| S1 | 稀疏索引（cone），没有复制 `sparse-checkout`；私有 `status` | **4 个对象文件的修改时间改变**，内容不变 |
| S2 | 同上；`ls-files --stage -z`（不带 `--sparse`） | **4 个**，同上 |
| S3 | 复制了 `sparse-checkout`；私有 `status` | 无 |
| S4 | 复制了；`ls-files --stage -z`（不带 `--sparse`） | **4 个** |
| S5 ~ S7 | 复制了；`ls-files --stage -z --sparse`、`diff-files --name-only -z`、`ls-files --sparse` 加 `check-attr --stdin filter` | 无 |
| S8 | 复制了；工作区中稀疏目录 `out/` 里有一个未跟踪文件；私有 `status` | **4 个**（新的反例） |
| S8e | 复制了；只有一个空目录 `out/` | **4 个** |
| S9 | 复制了；修改范围内的已跟踪文件 | 无 |
| S11 | 没有稀疏索引的 cone 稀疏检出，范围外有未跟踪文件；整套命令（复制或不复制模式文件） | 无 |
| S12 | 普通索引；整套命令；有未跟踪、已修改与受过滤的（stat 改变的）文件 | 无 |
| S13 | 拆分索引；整套命令 | 无 |
| S14 | 原索引有 untracked cache；整套命令 | 无 |

- 私有目录自己的对象目录在所有情形中都没有新写入。
- S8 的几种变体（S8b ~ S8d，Windows）：以 `-c` 设置 `index.sparse=false`、`core.sparseCheckout=false`，或改用 `--untracked-files=no`，都仍然改变 4 个对象的修改时间。

审查 055 之后的门禁实验（Windows 2.56 与 Linux 2.43 结果相同；X8 只在 Windows 上）：

| # | 情形 | 原 `.git` 的变化 |
|---|---|---|
| S15 | 审查 055 的反例：稀疏索引，`out/` 中有未跟踪文件，私有目录复制了模式文件；先运行私有 `status`（审查 052 版的顺序） | **4 个对象文件的修改时间改变**，内容不变 |
| S16 | 同一布局；先过第 2 步的门禁（解析索引副本：版本 3，1 个稀疏目录条目，`sdir`），判为 `unknown`，不再运行任何命令 | 无 |
| S17 | 门禁先行，工作区中没有 `out/` | 无（同样判为 `unknown`） |
| S18a | 普通的超级项目，子模块是稀疏索引并且有 `out/`；按审查 052 版的顺序在子模块中先运行 `status` | 子模块：**4 个** |
| S18b | 同一布局；超级项目门禁通过（不是稀疏索引）、运行整套命令（`diff-files` 带 `--ignore-submodules=all`），子模块门禁判为 `unknown`、不再运行命令 | 子模块：无 |
| S19 | 正对照：普通索引；门禁通过，再运行整套命令 | 无；输出 ` M d/x`、`?? d/u` |
| S20 | 失败路径：私有的索引副本被截断；门禁不能解析 | 无；环境错误（退出码 6），没有运行任何 git 命令 |
| S22 | 门禁对其他格式与损坏的索引：版本 2、版本 4、带 skip-worktree 的版本 3、拆分索引（`link`）、untracked cache（`UNTR`）；校验和错误；截断；版本 4 的稀疏索引 | 前五种判为不是稀疏索引；校验和错误与截断为环境错误；版本 4 的稀疏索引判为 `unknown` |
| X2 ~ X4 | 稀疏索引，`out/` 中有未跟踪文件：在原仓库或私有目录中运行 `ls-files --sparse --stage -z` | **4 个**；输出中稀疏目录条目为 0（已被展开） |
| X5 | 同上，原索引文件本身 | 没有被改写 |
| X6 | 超级项目的私有目录中，子模块是稀疏索引：`status --ignore-submodules=all`、`ls-files --sparse`、`check-attr` 与带 `--ignore-submodules=all` 的 `diff-files` | 子模块：都无 |
| X6' | 同上，`diff-files` **不带** `--ignore-submodules` | 子模块：**4 个** |
| X7 | `ls-files --sparse` 读取被截断的索引副本 | 无；退出 128 |
| X8 | `ls-files --sparse --stage` 的变体：关闭 commit-graph、关闭 multi-pack-index、关闭提示 | 都是 **4 个** |
| X8' | 另加 `-c sparse.expectFilesOutsideOfPatterns=true` | 无；报告 1 个稀疏目录条目 |

- 一次较早的运行（exp055 的 S16 第一版，用 `ls-files --sparse` 作门禁）没有发生展开，原因没有查明；对照实验 X2 ~ X4、X8 都发生了展开，所以本文不依赖那一次的结果，门禁改为直接解析索引文件。

子模块与 unborn：

| # | 情形 | 结果 |
|---|---|---|
| N1-g | `absorbgitdirs` 之后的子模块：`.git` 为 `gitdir: ../.git/modules/sub` | 本文环境中直接传 gitfile 为 0（审查 052 的环境中为 128）；`-C sub rev-parse --absolute-git-dir` 解析出 `.git/modules/sub`，用它读取的 HEAD 与在子模块中读取的相同 |
| N1-u | 没有提交的仓库 | 私有 `HEAD` 为 `ref: refs/heads/main`：`status` 退出 0；省略 `HEAD`：退出 128 |

**未验证**：

- macOS，以及 2.43、2.56 以外的版本。
- 真实的长驻 process 过滤器协议、LFS 的本地驱动、多层与 worktreeConfig 下的子模块。
- 稀疏索引在本文之外的触发条件；非 cone 模式。
- 大仓库复制索引的耗时。

以上都列入实现测试（§11）。
## 5. 提交关联与扫描

### 5.1 读取的 ref

- **本地分支**：
  - 命令：`git for-each-ref --format=%(refname)%00%(objectname)%00%(objecttype) refs/heads`，经过 `runObjects`（`objecttype` 要读取对象头）。
  - 只取 `objecttype = commit` 的 ref；指向其他对象的分支跳过，不建立游标。
  - 还没有提交的分支不会列出。
- **分离 HEAD**（负责人决定 D2：保留所有 worktree，不缩小到已登记的）：
  - 命令：`git worktree list --porcelain -z`。不认识 `-z` 的 git（usage 错误，退出 129）上，本次扫描返回退出码 6。不改按行解析：没有 `-z` 时，路径中的换行无法可靠切分（审查 050 N1，§13）。负责人 2026-10-19 决定：这是 MVP 扫描的能力要求，不增加回退；其他命令的支持范围不变。
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
  - **重用**：在同一路径上新建 worktree，它沿用原游标。读取范围仍按 §5.2 判断：旧 tip 不是新 HEAD 的祖先、或可靠地不存在时，才重新扫描；旧 tip 仍是祖先时照常增量读取（审查 052 N3）。已入库的 SHA 去重，关联不变（§5.4）。
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

### 5.2 扫描范围与严格读取（审查 050 B1、B2，审查 052 B1-R、B2-R）

扫描开始时，在读事务中读取项目已有的游标集合 C。C 是计划开始时的集合，不混入本轮刚扫描的 ref。每个游标有自己的 `last_tip`；每个 ref R 有本轮固定的 tip T（§5.1）。

**遍历必须读取对象**（审查 052 B1-R）：

- git 可以不读取提交对象就完成遍历，使缺失或损坏的对象被绕过。本文的扫描命令逐一关闭这些途径：

  | 途径 | 绕过的方式 | 扫描中的处理 | 实验 |
  |---|---|---|---|
  | commit-graph | 从图文件取父提交与时间，不读取提交对象 | 每次调用固定 `-c core.commitGraph=false`，不依赖用户配置 | G1、G2 |
  | 替换引用 | 用另一个对象代替 | 既有的全局参数 `--no-replace-objects`（§6.1.2） | — |
  | grafts（`info/grafts`） | 改写父提交，缺失的父提交不再被访问 | 显式 `GIT_GRAFT_FILE=`，按实际 Git 身份行为探测；不回退 missing 路径（§14.3、§71） | G5 |
  | multi-pack-index | 对象查找经过它；它损坏时 git 回退并报告错误 | 分类与读取都固定 `-c core.multiPackIndex=false`，只用各个包自己的索引 | K7 |
  | 浅克隆（`shallow` 文件） | git 声明的边界之外的父提交不访问 | MVP 不扫描浅克隆（P3）：扫描之前直接检测，是浅克隆或无法确认就返回退出码 6；读取另设经行为探测的显式 `GIT_SHALLOW_FILE=`，使检测之后才出现的浅边界也不能截断读取（见下文"浅克隆"） | G6、H1 ~ H7 |

- 关闭这些途径之后，git 访问到的每一个提交都要读取并解析它的对象；读取失败就是非 0 退出。
- 没有被访问到的提交，不影响"T 可达、且不被排除项可达"的集合，所以不需要读取（实验 G4：排除项之后的祖先缺失，不影响结果）。
- **边界**：
  - git 读取对象时不校验内容与对象 ID 是否一致。一个能解压、能解析、但内容与 ID 不符的对象，会被当作正常对象读取。本文不声称检测这类损坏。
  - 关闭 commit-graph 之后，git 只能依据提交时间裁剪排除的一侧。在提交时间倒置的历史中，它是否会多读已被排除覆盖的提交，本文没有验证；多读的已入库提交去重，未入库的提交则可能首次入库。实现时测试（§11）。

**第一步：确认正向 tip**：

- 所有 T 都是必需读取的对象。分支的 T 已由 `for-each-ref` 的 `objecttype = commit` 确认对象头是提交；分离 HEAD 的 T 用下一步的分类确认。
- 分类结果不是"存在"时，本次扫描返回退出码 6。
- "存在"只表示对象头可读、类型是提交；对象能否完整读取，由第四步的严格读取确认。

**第二步：给旧 tip 分类**：

- 对 C 中所有不同的 `last_tip`（以及分离 HEAD 的 T），一次 `git cat-file --batch-check`，从 stdin 读入完整 SHA；对其中报告 `missing` 的，再运行裸 ID 的 `git cat-file -e <sha>`（不带 `^{commit}`，实验 R4）。
- 这些命令都固定 `-c core.multiPackIndex=false`。
- 分类：

  | `--batch-check` | 裸 ID 的 `cat-file -e` | 另外的条件 | 分类 |
  |---|---|---|---|
  | `<sha> commit <size>` | — | — | **存在**（对象头可读） |
  | `<sha> missing` | 0 | — | **存在但不可读**，退出码 6（实验 R6、K11） |
  | `<sha> missing` | 1 | 两条命令都**没有写出任何 stderr**，并且**存储检查通过**（下一条） | **可靠地不存在** |
  | `<sha> missing` | 1 | 有 stderr 输出，或存储检查不通过 | **无法确认**，退出码 6 |
  | 其他类型、其他输出、任一命令的其他退出码 | — | — | **无法确认**，退出码 6 |

- **存储检查**：只在出现"`missing` 且 1"时进行，由 Specmetry 直接读取 git 目录中的文件，不运行 git，不读取源码或对象内容，也不写入任何文件（不重建索引、不修复包）：
  1. `objects/` 与其下所有目录（包括 `objects/pack`）都能列出；
  2. **包与索引双向配对**（审查 055 B2-R2）：`objects/pack` 中每一个 `pack-<哈希>.pack` 都有同名的 `.idx`，每一个 `.idx` 都有同名的 `.pack`。只检查"每个 `.idx` 都有 `.pack`"，在一个 `.pack` 失去了 `.idx` 时会在空集合上成立（实验 K18）；
  3. 每一个 `.idx` 都能读取，长度不小于最小值，末尾 20 字节等于它前面全部内容的 SHA-1；它对应的 `.pack` 能读取，末尾 20 字节等于 `.idx` 中记录的包校验和；
  4. `objects/pack` 中只有可识别的文件：成对的 `.pack` 与 `.idx`；与某个现存的 `.pack` 同名的 `.keep`、`.promisor`、`.bitmap`、`.rev`、`.mtimes`；multi-pack-index 的文件（读取时已关闭它）。其他任何文件（例如 `tmp_pack_*`、`tmp_idx_*`，或没有 `.pack` 的 `.rev`）说明存储不完整或正在写入（实验 K20、K22）；
  5. 有 `objects/info/alternates` 时，它能读取，其中列出的每个对象目录都存在，并对它们递归执行 1 ~ 5（层数上限与 git 相同；实验 K19）。
  - 任何一条不成立：**无法确认**，退出码 6，不发布扫描结果、不推进游标。这区分了"对象被正常回收"（检查通过，实验 K1、K21）与"存储不完整"（检查不通过）。
  - 它读取全部 `.idx` 文件，耗时与对象数成正比；但只在出现缺失的旧 tip 时才进行，通常是历史改写并回收之后。
- **stderr 只看有没有**：不看、不比较、不保存、不回显其中的文字，这与按本地化错误文字匹配不同。
- 这一分类保留了历史改写后重扫的要求：真正被回收的旧 tip（实验 K1）分类为可靠地不存在，按改写处理。

**为什么需要 stderr 与存储检查两个条件**（实验 K）：

- 审查 052 的反例（K2，`.idx` 被截断）：`missing`、`cat-file -e` 为 1，**有** stderr；存储检查也不通过。
- 但 stderr 不够：`.idx` 的 SHA 表被静默改坏（K3）、`.pack` 被删除（K6）、`.pack` 或 `.idx` 不可读（K12、K13）、包目录可列出但不可进入（K15）时，都是 `missing`、1、**没有** stderr。这些只有存储检查能发现。
- 存储检查也不够：松散对象所在的目录不可读（K10）时，`cat-file -e` 为 1、没有 stderr，但 `--batch-check` 有 stderr；存储检查同样能发现。两者合用。
- 审查 055 的反例（K18，`.pack` 失去 `.idx`）：`missing`、1、**没有** stderr；审查 052 版的存储检查**通过**（漏检），本版的双向配对不通过。恢复同一个 `.idx` 之后，对象又能读出，说明它不是被回收的对象。

**仍然不能证明的**：

- 文件系统本身错误地报告"不存在"（例如网络或虚拟文件系统的缓存），或者在存储检查与 git 调用之间存储被改变。后一种情形中，第三、四步的读取若需要该对象，会失败（实验 K8）；若不需要，则按不存在处理。
- 一个松散对象文件因磁盘故障而丢失：它确实不在对象库中，分类为可靠地不存在，按改写重扫。这与"被回收"无法区分，但都不是"存在却读不到"。
- 所以"可靠地不存在"的含义是：git 的协议输出、退出码与 stderr，以及上述存储检查，都与"对象库中没有这个对象"一致。它不是对所有权限与 I/O 状态的证明。

**第三步：每个 ref 的范围**：

| 情形 | 判断 | 读取的提交 |
|---|---|---|
| R 有游标，`last_tip = T` | — | 无 |
| R 有游标，旧 tip **存在** | `git merge-base --is-ancestor <last_tip> <T>`（固定 `-c core.commitGraph=false`）：0 | `<last_tip>..<T>`（architecture §6.7 的增量） |
| 同上 | 1（历史被改写） | R 的全部历史 `<T>`，按 SHA 去重 |
| 同上 | 其他退出码 | 读取失败，退出码 6（实验 K8：分类之后被回收） |
| R 有游标，旧 tip **可靠地不存在** | — | 按历史改写处理：R 的全部历史 `<T>` |
| R 有游标，旧 tip **存在但不可读**或**无法确认** | — | 本次扫描返回退出码 6，不写入 |
| R 没有游标（新的 ref，D4） | — | `<T>`，排除 C 中**存在**的所有 `last_tip`；可靠地不存在的旧 tip 不作排除项（范围变大）；有任何一个旧 tip 存在但不可读或无法确认，本次扫描返回退出码 6 |
| 项目还没有任何游标（第一次扫描） | — | 每个 ref 的全部历史 |

- 实验 R10 中，打包对象的数据被改写、commit-graph 存在时，`is-ancestor` 返回 0：它用图文件回答，没有读取对象。固定 `core.commitGraph=false` 之后，祖先判断也要读取对象（实验 G2）。（实现细化，待复审：路径中间的提交缺失时，部分版本返回 1 而不是失败，扫描的保证不变，见 §14.4。）

**第四步：严格读取**：

- 每个有工作的 ref 一次 `git log -z --stdin --format=%H%x00%ct%x00%(trailers:key=Specmetry-Ref,valueonly,separator=%x2C)%x00%B`，固定 `-c core.commitGraph=false -c core.multiPackIndex=false`，并设置 `GIT_GRAFT_FILE`（见上）。（实现细化，待复审：另加 `--no-show-signature`，见 §14.2。）
- stdin 中先写正向 tip T，再为每个排除项写一行 `^<完整 SHA>`。
  - 不写 `--not` 行（git 2.32 的 `--stdin` 不接受选项行，审查 050 N1）；
  - **不使用 `--ignore-missing`**（审查 050 B1；实验 R1）。
- 结果：
  - 退出 0：读取成功；没有输出就确实没有新提交。
  - 任何非 0：本次扫描失败，退出码 6，不写入，游标不变。
  - **失败之前已经读到的部分结果一律丢弃**：git 可能先输出一部分提交再失败（实验 G3）。整次扫描的写事务只在所有 ref 都读取成功之后开始（§5.5）。
- 每条记录都要校验：SHA 为 40 位小写十六进制，时间为十进制整数。不合格时整个扫描失败。

**为什么这样不会漏读、也不会错误地推进游标**：

- 游标只在所有 ref 的严格读取都成功之后，才在写事务中推进（§5.5）。
- 读取的范围是"正向 tip 可达、且不被**已确认存在**的排除项可达"的全部提交；git 访问到的每一个都读取了它的对象（上文"遍历必须读取对象"）。
- 只有被确认存在、并且在读取中被成功解析的排除项，才会缩小范围。分类之后才消失的排除项，在读取中解析失败（实验 G1 的关闭 commit-graph 一行）。
- 被判为"可靠地不存在"的旧 tip，只会让范围变大。因此而多读到的、此前没有入库的提交，如果首次满足入库条件，按这次的 `seq` 入库，属于负责人已接受的"重扫后的首次入库"（§2）。
- 存在但不可读、无法确认的旧 tip，不再被当作"可靠地不存在"而放宽范围（审查 052 B2-R）。

**新 ref**：

- 排除已有游标覆盖的历史（D4）：在已扫描的 `main` 上新建分支时，不回头重读 `main` 的全部历史，符合"常规增量扫描不主动回头"。
- 选择读取全部历史，会改变新 ref 带来的首次关联机会，不只是性能上的差别（审查 050 的 D4 裁定）。

**还没有任何编号**：

- 项目还没有任何需求或 AC 时，不扫描、不建立游标（D5），记为"没有扫描"。
- 这与"扫描了但没有匹配"不同，后者会建立游标（§5.5）。

**实验**（隔离夹具；受控环境；Git for Windows 2.56.0 / Windows 11，另在 WSL Ubuntu 的 git 2.43.0 上重做）：

审查 050 之后的实验（Windows 2.56）：

| # | 情形 | 结果 |
|---|---|---|
| R1 | 正向 tip 不存在，stdin 读入 | 严格：**128**；带 `--ignore-missing`：**0，没有输出** |
| R2 | 正向 tip 损坏（松散对象内容不是 zlib） | `log` **128** |
| R3 | 排除项 `^<从来不存在的 SHA>`，严格 | **128** |
| R4 | 不存在的 SHA：`merge-base --is-ancestor`；`cat-file -e <sha>^{commit}`；`cat-file -e <sha>` | 128；**128**；**1** |
| R5 | `--batch-check`：存在的提交、不存在的、一个树 | `commit 194`、`missing`、`tree 0`，退出 0 |
| R6、R7 | 松散对象内容损坏、为空文件 | `--batch-check`：**`missing`**；`cat-file -e`：**0** |
| R8、R9 | 松散对象被删除；历史改写并 `gc --prune=now` 之后的旧 tip | `missing`；`cat-file -e`：**1** |
| R10 | 打包对象的数据被改写，有 commit-graph | `--batch-check`：`commit 194`；`is-ancestor`：0（图文件回答） |
| R11 ~ R13 | 正常扫描；`c3 ^c3`；stdin 中的 `--not` 行 | 0，两个提交；0，没有输出；2.56 接受、2.32 不接受 |

审查 052 之后的实验（G、K：Windows 2.56 与 Linux 2.43 都运行；K10 ~ K17：Linux 2.43，非 root 用户）：

| # | 情形 | 默认 | 本文的处理 |
|---|---|---|---|
| G1 | A → B → C，写出 commit-graph，删除 A；`log C ^A` | 2.56：**0**，返回 C、B（审查 052 B1-R）；2.43：128 | `core.commitGraph=false`：两个版本都 **128** |
| G2 | 同上，`is-ancestor A C` | 128 | 128 |
| G3 | commit-graph，删除范围内的 B；`log C ^A` | 128；2.56 先输出了 C | 128；部分结果丢弃 |
| G4 | 删除排除项 Y 之后的祖先 X；`log Z ^Y` | — | 0，结果正确（X 不需要） |
| G5 | `info/grafts` 把 C 改成没有父提交；再删除 A | 0，只返回 C；删除 A 后仍为 0 | `GIT_GRAFT_FILE` 指向不存在的路径：返回 C、B、A；删除 A 后 **128** |
| G6 | `shallow` 文件声明 B 为边界，删除 A | 0，返回 C、B | 关闭 commit-graph 后仍为 0；`GIT_SHALLOW_FILE` 指向不存在的路径：128，`is-shallow` 变为 false（P3） |
| G7 | 什么都不删除，有 commit-graph | 0，两个提交 | 关闭 commit-graph 与 multi-pack-index：0，两个提交 |
| K1 | 真正被回收的旧 tip | `missing`；1；没有 stderr | 存储检查通过：**可靠地不存在** |
| K2 | 审查 052 的反例：`.idx` 被截断 | `missing`；1；**有** stderr | 存储检查：校验和错误；**无法确认** |
| K3 | `.idx` 的 SHA 表中改坏一个字节 | `missing`；1；**没有** stderr | 存储检查：校验和错误；**无法确认** |
| K4 | 只改 `.idx` 末尾的校验和 | `commit`（照常读取） | 不进入存储检查 |
| K5 | 包中 A 的数据被改写，`.idx` 完好 | `tag 1326`（类型错误），没有 stderr | 其他类型：**无法确认** |
| K6 | 删除 `.pack`，保留 `.idx` | `missing`；1；**没有** stderr | 存储检查：包缺失；**无法确认** |
| K7 | multi-pack-index 损坏 | 回退读取，有 stderr | `core.multiPackIndex=false`：照常读取，没有 stderr |
| K8 | 分类时存在，之后被回收 | — | `is-ancestor` 128：退出码 6 |
| K9 | 分类前后被重新打包 | — | 仍然存在 |
| K10 | 松散对象所在的目录不可读 | `--batch-check` 有 stderr；`cat-file -e` 1、没有 stderr | 存储检查：目录不可列出；**无法确认** |
| K11 | 松散对象文件不可读 | `missing`；`cat-file -e` **0** | **存在但不可读** |
| K12、K13 | `.pack` 不可读；`.idx` 不可读 | `missing`；1；**没有** stderr | 存储检查不通过；**无法确认** |
| K14、K15 | 包目录不可读；可列出但不可进入 | `missing`；1；前者有 stderr，后者**没有** | 存储检查不通过；**无法确认** |
| K16 | `objects/info/alternates` 不可读（对象在备用对象库中） | `missing`；1；有 stderr | 存储检查：alternates 不可读；**无法确认** |
| K17 | 备用对象目录不存在 | `missing`；1；有 stderr | 存储检查：备用目录缺失；**无法确认** |
| K18 | 审查 055 的反例：已打包的旧提交，`.idx` 移开，`.pack` 保留 | `missing`；1；**没有** stderr；审查 052 版的检查**通过** | 双向配对：`.pack` 没有 `.idx`；**无法确认**。恢复 `.idx` 后为 `commit` |
| K19 | 同上，发生在备用对象目录中 | `missing`；1；没有 stderr；审查 052 版的检查通过 | 递归检查备用目录：**无法确认** |
| K20 | 真正被回收的旧 tip，但 `objects/pack` 中残留 `tmp_pack_*` | `missing`；1；没有 stderr | 不可识别的文件：**无法确认**（保守） |
| K21 | 正对照：真正被回收的旧 tip；包旁有 `.bitmap`、`.rev`、`.keep` | `missing`；1；没有 stderr | 检查通过：**可靠地不存在** |
| K22 | 一个没有 `.pack` 的 `.rev` | `missing`；1；没有 stderr | **无法确认** |

**结论**：

- commit-graph 的绕过与版本有关：2.56 会，2.43 不会。关闭它之后，两个版本都严格。
- `--batch-check` 的 `missing` 加裸 ID 的 `cat-file -e` 为 1，不足以证明不存在；加上"没有 stderr"也不够（K3、K6、K12、K13、K15、K18、K19）。三者再加上双向配对的存储检查，本文的 22 种情形中，只有真正被回收的旧 tip（K1、K21）被判为可靠地不存在。
- K18 ~ K22 在 Windows 2.56 与 Linux 2.43 上结果相同。

**浅克隆**（负责人决定 P3，2026-10-19）：

- **检测**：扫描之前，Specmetry 直接检查公共 git 目录中的 `shallow` 项（路径由 `rev-parse --git-common-dir` 给出；这个命令不读取对象，也不读取 `shallow` 文件的内容）：
  - 不存在（lstat 报告"不存在"）：不是浅克隆，照常扫描；
  - 存在（不论是普通文件、目录或其他）：浅克隆，返回环境错误（退出码 6）；
  - 其他任何错误：无法确认，同样返回退出码 6。
  - 不用 `git rev-parse --is-shallow-repository`：`shallow` 文件不可读或是一个目录时，它报告 `false`、退出 0，`log` 也照常把仓库当作完整的仓库读取（实验 H6、H7）。
- **退出码 6 时**：不写提交、关联、`seq` 或游标，不伪装成成功的空扫描；不自动拉取，不修改浅边界。触发扫描的命令应明确呈现这一限制（WP2.7）。
- **读取中的防护**：扫描的读取另设 `GIT_SHALLOW_FILE` 指向一个不存在的路径。检测之后才出现的浅边界因此不会截断读取；父提交真的缺失时，严格读取失败（实验 H5）。
- **生命周期**：
  - 浅克隆期间不建立、也不推进任何游标（审查 055 的加深场景因此不会出现：不存在在浅边界上建立的游标）。
  - 恢复为完整仓库之后，按既有的规则扫描：从未扫描过的仓库是第一次扫描，读取全部历史（实验 H3）；在变为浅克隆之前已有游标的，按 §5.2 的增量或重扫（实验 H4：变为浅克隆之前已扫描 A、B、C，恢复之后只需读取新提交 D）。
  - 将来支持浅克隆时，另行设计 deepen / unshallow 与游标的生命周期，不在本文之内。

| # | 情形 | 结果 |
|---|---|---|
| H1 | 完整仓库 | 检测为不是浅克隆；原 `.git` 没有变化 |
| H2 | 有 `.git/shallow` | 浅克隆：退出码 6；原 `.git` 没有变化 |
| H3 | 浅克隆期间拒绝扫描（没有游标），之后恢复为完整仓库 | 第一次扫描读取 3 个提交（A、B、C） |
| H4 | 完整仓库已扫描（游标在 C），变为浅克隆时拒绝、游标不变，恢复后新增 D | 增量 `C..D` 读取 1 个提交，没有遗漏 |
| H5 | 检测之后才出现 `shallow` 文件 | 默认的 `log` 截断为 3 个提交；设置 `GIT_SHALLOW_FILE` 后读取 4 个；父提交缺失时 128 |
| H6 | `shallow` 文件不可读（Linux，非 root） | `--is-shallow-repository` 为 `false`、退出 0，`log` 退出 0；本文的检测：无法确认，退出码 6 |
| H7 | `shallow` 是一个目录 | `--is-shallow-repository` 为 `false`；本文的检测：退出码 6 |

**未验证**：

- macOS，以及 2.43、2.56 以外的版本。
- 真实的部分克隆；真实的并发 `gc`；网络与虚拟文件系统。
- 提交时间倒置时关闭 commit-graph 的遍历范围。
- 存储检查在大型仓库中的耗时。

**其他实验**（审查 050 之前，仍然有效）：

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
  - 旧 tip 或正向 tip 的分类为"存在但不可读"或"无法确认"（§5.2），包括分类命令写出了 stderr、存储检查不通过；
  - 读取中途失败：已经读到的部分结果一律丢弃（§5.2 第四步）；
  - 存储检查不通过：包与索引不配对、不可识别的文件、不可读的目录或 alternates（§5.2）；
  - 浅克隆，或浅克隆的检测无法确认（§5.2，P3）；
  - git 不支持需要的选项（§13）；
  - 被取消。
- 错误只说明是哪个 git 子命令、退出状态是多少（§6.1.2），不依赖、也不回显 git 的错误文字。错误中的子命令名称是固定的标签（例如 `status`、`log`），不从参数中取：参数中可能有 `--git-dir=<路径>` 之类的全局选项（审查 052 N2）。
- **密钥命中**：拒绝本次扫描，返回 `invalid_input`（退出码 2），不写入（§5.1，D7）。
- **数据库的失败**：沿用存储层既有的分类。
  - 打开数据库时的偶发失败见维护事项 M-14；
  - 写入阶段的 `SQLITE_BUSY` 见维护事项 M-15。
  - 两者都没有据此批准新的错误映射或重试。
- **不自动重试**：一次命令中不重试扫描。失败与并发跳过都不留下状态，下一次扫描从不变的游标重新开始。

## 6. 接口设计

**`internal/gitx` 新增**（名称与签名供审查，实现时可调整；行为以本文为准。实现细化，待复审：`Clean`、`Tips`、`Classify`、`IsAncestor`、`Log` 另带私有临时区参数 `*Scratch`，由 `NewScratch` 在数据目录中准备，见 §14.3）：

```go
// Entry is a tree entry's metadata (data-model §4.11.2).
type Entry struct{ Mode, Type, OID string }

// TreeEntry reads the entry of a lookup path (V-P04 normalized) in the
// tree of a full commit ID: "." is the root tree. found is false when the
// tree has no such path (L2); an error is an object the path needs that
// cannot be read (L1, exit 6).
func TreeEntry(ctx context.Context, worktree, commit, lookup string) (e Entry, found bool, err error)

// State is a worktree state as Clean reads it.
type State int

const (
	StateClean State = iota
	StateDirty
	// StateUndecidable: telling clean from dirty would need a program the
	// repository configures (§4.3 step 5), or the index is a sparse index
	// (step 2). It is recorded as unknown, never as clean (decision P1).
	StateUndecidable
)

// Clean reads the worktree state (data-model §1) through a private git
// directory whose configuration Specmetry wrote, and submodule by
// submodule, so that no program the repository configures runs and the
// repository's files are left as they were (§4.3).
func Clean(ctx context.Context, worktree string) (State, error)

// IndexIsSparse parses an index file without running git (versions 2 to
// 4, and the shared index that a split index links to) and reports whether
// it has sparse directory entries or the "sdir" extension; anything it
// cannot parse completely is an error (§4.3 step 2). Clean calls it on the
// private copy before any git command reads that copy.
func IndexIsSparse(path string) (bool, error)

// CheckShallow returns an environment error for a shallow repository, or
// one whose "shallow" entry in the common git directory cannot be shown
// absent; it reads that entry itself, since git reports false when the file
// is unreadable (§5.2, decision P3).
func CheckShallow(ctx context.Context, worktree string) error

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
	Present Presence = iota // the object header says commit; the full read comes later
	Absent                  // reliably not in the object database (§5.2, step 2)
)

// Classify classifies commit IDs (§5.2): Present or Absent. Absent needs
// git's protocol answers, no stderr output at all, and the storage check;
// anything else (present but unreadable, another type, an unexpected
// answer, any stderr output, a failed storage check) is an environment
// error.
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
// exclude (Present commits), newest first, calling fn for each, with the
// commit-graph, grafts and the multi-pack-index out of use (§5.2); any
// object it cannot read is an environment error, and a caller keeps
// nothing fn received before it.
func Log(ctx context.Context, worktree, tip string, exclude []string, fn func(Commit) error) error
```

- 除 `worktree list` 之外，这些操作都经过 `runObjects`。`worktree list` 只读取 git 目录的布局。
- 扫描的读取（`Classify`、`IsAncestor`、`Log`）固定 `-c core.commitGraph=false -c core.multiPackIndex=false`，并设置 `GIT_GRAFT_FILE` 与 `GIT_SHALLOW_FILE`（§5.2）。扫描开始之前先运行 `CheckShallow`。
- `Clean` 的私有 git 目录由 `gitx` 创建并删除（§4.3 的生命周期）；在实验覆盖的布局中，它不写用户的仓库。
- 错误都是 `apperr.Environment`（退出码 6），不带 git 的输出。
- `gitx` 不知道编号、项目或数据库。

**扫描层**（建议 `internal/gitscan`）：

| 职责 | 说明 |
|---|---|
| `Mentions(trailers, message string) []Mention` | 纯函数：§5.3 的切词与 trailer 优先，结果为编号与来源 |
| 计划 | 读游标，确认是否已有编号（D5） |
| 读取 | 调用 `gitx`：浅克隆的检测（P3）、tip、密钥扫描（D7）、分类与存储检查、范围、严格日志；得到候选提交 |
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
| **D1** | `clean` 定义加入 `--ignore-submodules=none` | **负责人已决定（2026-10-18）**。语义写入 data-model §1。读取方式另由 §4.3 解决（B3）；这个参数本身不解决外部过滤器的问题，也不包括过滤器或稀疏索引下的退化（P1、P2） |
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

**审查 052 的阻塞项**（详见 [review-resolution.md §64](review-resolution.md)）：

| # | 问题 | 修订 | 状态 |
|---|---|---|---|
| B1-R | commit-graph 使已经消失的排除对象仍被严格 log 接受 | 扫描的读取固定 `core.commitGraph=false`、`core.multiPackIndex=false`，并让 grafts 失效；部分结果丢弃（§5.2；实验 G1 ~ G7） | 修订；审查 055 认为原反例已修正，浅克隆由 P3 决定；审查 057 GO，没有剩余的阻塞项（[review-resolution.md §67](review-resolution.md)） |
| B2-R | 损坏的包索引被判为可靠地不存在 | "可靠地不存在"另需没有 stderr 与存储检查；其他一律无法确认（§5.2；实验 K1 ~ K17） | 修订；审查 055 发现 B2-R2，再次修订；审查 057 关闭 B2-R2（§67） |
| B3-R1 | 取消过滤器可能把 `dirty` 判为 `clean` | 撤回"只会 clean → dirty"；需要比较受过滤文件的内容时为不可判定（§4.3 第 5 步；实验 F1 ~ F7） | 修订；审查 055 认为判据方向成立，取值由 P1 决定为 `unknown`；审查 057 GO，没有剩余的阻塞项（§67） |
| B3-R2 | 稀疏索引的展开借 alternates 改变原对象的修改时间 | 复制 `sparse-checkout`、只用带 `--sparse` 的列举；带稀疏目录条目的索引为不可判定（§4.3；实验 S1 ~ S14） | 审查 055 发现判断晚于 `status`（B3-R2），门禁已前置，范围由 P2 决定；审查 057 关闭 B3-R2（§67） |

**审查 052 引出的取舍**（负责人 2026-10-19 决定；D1 不包括这些）：

| # | 内容 | 决定 |
|---|---|---|
| **P1** | §4.3 的"不可判定"记为什么 | **`unknown`**，不能当作 `clean`；执行因此不合格 |
| **P2** | 稀疏索引的范围 | **含稀疏目录条目的索引一律不可判定**，安全门禁放在任何可能展开索引的命令之前（§4.3 第 2 步）；不扩大到所有稀疏检出 |
| **P3** | 浅克隆 | **MVP 暂不支持浅克隆的提交扫描**：只读地明确检测，浅克隆或无法确认都返回退出码 6，不写入、不伪装成空扫描、不自动拉取或改变浅边界；命令应呈现这一限制；将来支持时另行设计 deepen / unshallow 与游标的生命周期（§5.2） |
| **`-z`** | 不支持 `worktree list -z` 的 git | MVP 的扫描返回退出码 6，不增加按行解析的回退；这是扫描的能力要求，其他命令的支持范围不变（§5.1、§13） |

**审查 055 的阻塞项**（详见 [review-resolution.md §66](review-resolution.md)）：

| # | 问题 | 修订 | 状态 |
|---|---|---|---|
| B2-R2 | 存储检查只查 idx → pack，`.pack` 失去 `.idx` 时被当作可靠缺失 | 包与索引双向配对；不可识别的文件为存储不完整；alternates 递归同样检查（§5.2；实验 K18 ~ K22） | 修订；审查 057 关闭（§67） |
| B3-R2 | 稀疏索引的判断晚于 `status`，返回不可判定不能撤销已发生的修改 | 门禁移到第 2 步：Specmetry 解析索引副本，不运行 git，在任何可能展开索引的命令之前；子模块同样先过门禁；超级项目的 `diff-files` 带 `--ignore-submodules=all`（§4.3；实验 S15 ~ S22、X1 ~ X8） | 修订；审查 057 关闭（§67） |
| 浅克隆的加深 | P3 的静态描述不足以决定加深后的扫描 | 负责人选择不支持浅克隆的扫描；浅克隆期间不建立游标，恢复为完整仓库之后按既有规则（§5.2"浅克隆"；实验 H1 ~ H7） | 修订；审查 057 关闭（§67） |

**其他待决或需要说明的事项**：

- 跨项目的同名编号无法区分（§5.3），这是现有格式的限制，不在 MVP 中扩展。
- 第一次扫描大型仓库的耗时、`Clean` 复制索引的耗时，以及 §5.2 的存储检查在大型仓库中的耗时，都没有基准。实现时测量，必要时另立事项。
- 维护事项 M-14、M-15 未解决，都与扫描所在命令打开或写入数据库有关，各自单独处理（§5.6）。
- **低于 2.36 的 git 上的分离 HEAD**（§13）：负责人 2026-10-19 决定，MVP 的扫描以 `worktree list -z` 为能力要求，缺少时返回退出码 6，不增加按行解析的回退。

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
| L14 | 正向 tip 在分类之后变得不可读（模拟：tip 对象被删除或损坏） | `log` 非 0：退出码 6，游标不前进，下一次仍读取它（审查 050 B1；实验 R1、R2） |
| L15 | 游标的旧 tip 在历史改写后被回收（可靠地不存在） | 按改写处理：严格读取新 tip 的全部历史；已入库的去重（B2；实验 R8、R9） |
| L16 | 游标的旧 tip 是损坏的松散对象（存在但不可读） | 退出码 6，不写入（B2；实验 R6） |
| L17 | 新 ref 的排除项中有一个可靠地不存在的旧 tip | 不作排除项，范围变大，严格读取；其余照常（B1） |
| L18 | 排除项在分类之后消失（先存在、形成计划、写出 commit-graph，再删除） | `log` 非 0：退出码 6，不写入；commit-graph 不能掩盖它（审查 052 B1-R；实验 G1） |
| L19 | 正常扫描：分支有新提交 | `log` 退出 0；新入库与游标在一个事务中写出 |
| L20 | 移动一个分离 HEAD 的 worktree；在原路径上新建 worktree | 前者成为新游标、按新 ref 读取；后者沿用原游标，按 §5.2 判断：旧 tip 不是祖先或可靠地不存在时重扫，仍是祖先时增量（D2；审查 052 N3） |
| L21 | 已经读到一部分提交之后 `log` 失败（实验 G3） | 退出码 6；部分结果丢弃；所有提交、关联、`seq` 与游标都不变 |
| L22 | 旧 tip 所在包的 `.idx` 被截断（审查 052 的反例）、SHA 表被静默改坏、`.pack` 缺失或不可读，或者 `.pack` 失去了 `.idx`（审查 055 的反例），包括发生在备用对象目录中 | 无法确认：退出码 6，不写入，不当作改写重扫（审查 052 B2-R、审查 055 B2-R2；实验 K2、K3、K6、K12 ~ K15、K18、K19） |
| L23 | `info/grafts` 让缺失的父提交不再被访问 | grafts 失效，严格读取遇到缺失：退出码 6（实验 G5） |
| L24 | 浅克隆 | 退出码 6；不写提交、关联、`seq` 或游标；不显示为成功的空扫描；不自动拉取（P3；实验 H2） |
| L25 | `shallow` 项存在但不可读，或是一个目录 | 无法确认：退出码 6（实验 H6、H7） |
| L26 | 浅克隆期间被拒绝，之后恢复为完整仓库 | 没有游标时为第一次扫描，读取全部历史；已有游标的按增量或重扫，没有遗漏（实验 H3、H4） |

**输入路径与工作区状态**：

- 输入路径见 §4.2 的 T1 ~ T23，工作区状态见 §4.3 的 C1 ~ C11、F1 ~ F7、S1 ~ S14 与 N1，实现时逐项写成测试。
- 此外，仓库根的记录保留原 `path`（D3）。

## 10. 验收映射

| 验收项 | 内容 | WP2.3 的部分 | 其余部分 |
|---|---|---|---|
| MVP-WT-04 | 每个 ref 的扫描、改写后重扫、不重复 | 扫描层的集成测试（L5 ~ L7） | 命令中的触发：WP2.7 |
| MVP-EV-11 | 提交的标题中有 canary → 只保存 SHA 与时间 | 扫描层的集成测试（L13） | — |
| MVP-EV-10 | 输入记录 | `gitx.TreeEntry`（T1 ~ T23） | 在执行中记录：WP2.5 |
| MVP-EV-08、EV-09、WT-03 | 工作区状态与代码上下文 | `gitx.Clean`（C1 ~ C11、F3、F6、S9） | 在执行与命令中使用：WP2.5、WP2.4 |
| MVP-EV-07 | 只追加表的触发器 | `commits`、`commit_links` 的迁移测试（§7） | — |
| MVP-EV-14 | 编号匹配的规则 | `Mentions` 的单元测试（M1 ~ M11）与集成测试 | — |
| MVP-EV-15 | 关联的生命周期 | 集成测试（L1 ~ L3，前提见 §9） | — |
| MVP-EV-16 | 扫描的写入、严格读取与失败 | 集成测试（L4、L9、L10、L14 ~ L19、L21 ~ L23；G1 ~ G7、K1 ~ K22） | — |
| MVP-EV-17 | 输入路径的读取机制（TD-23） | `gitx` 的集成测试 | 在执行中的退出码与不写记录：WP2.5 |
| **MVP-EV-18**（新增） | 工作区状态的读取不执行仓库配置的程序；子模块变化；不报告错误的 `clean`；稀疏门禁前置；原仓库文件的内容与修改时间不变 | `gitx.Clean` 的集成测试（C1 ~ C11、F1 ~ F7、S1 ~ S22、X1 ~ X8、N1）；`IndexIsSparse` 的单元测试 | — |
| MVP-WT-06 | 扫描的范围：分离 HEAD 与其身份（D2）、新 ref、密钥命中的拒绝（D7）、没有编号时不扫描（D5）、浅克隆（P3） | 集成测试（L6、L8、L11、L12、L20、L24 ~ L26） | 命令如何呈现扫描被拒绝与浅克隆的限制：WP2.7 |
| §6.5 进度中的关联提交 | — | — | 已由 WP2.1 覆盖 |
| CX | 没有直接针对提交关联的反例项 | — | E2E-2 的"追加带 trailer 的提交"属于 WP2.9 |

- 只看"git 返回非 0 就拒绝"，发现不了审查 050 B1 与审查 052 B1-R 的假成功。所以 L14、L18、L21 ~ L23 必须断言**所有提交、关联、`seq` 与游标都不变**，不只断言退出码。

## 11. 实现测试计划

- **单元**：
  - `Mentions`：M1 ~ M11，以及边界字符、CR、非 ASCII、空输入。
  - `ls-tree`、`log`、`worktree list`、`--batch-check` 的输出解析：合格与不合格的输入，不合格时返回错误。
  - 范围规划与分类：给定游标、分类结果与祖先判断的结果，覆盖 §5.2 的每一行。
- **`gitx` 集成**（`testutil` 的 git 夹具，临时仓库，三个平台）：
  - T1 ~ T23、C1 ~ C11、R1 ~ R13，以及审查 052 之后的 G1 ~ G7、K1 ~ K17、F1 ~ F7、S1 ~ S14、N1。
  - L1、L3 状态、损坏与缺失的对象，通过删除或改写松散对象、改写包数据与包索引、改变文件权限（K10 ~ K17 只在 Linux 与 macOS 上）来模拟。
  - **对象读取不被绕过**：写出 commit-graph 之后删除排除对象；`info/grafts`；损坏的 multi-pack-index。断言扫描失败、不写入。
  - **原仓库不变**：每个 `Clean` 测试前后，比较原 `.git` 下所有文件的内容哈希**与修改时间**（之前先设为旧的修改时间），不只比较索引与配置。
  - **临时目录**：成功、失败、取消之后都没有遗留；删除失败时返回环境错误；遗留目录在下一次读取时被清理。
  - **门禁的执行顺序**（审查 055 B3-R2）：稀疏索引的布局（超级项目与子模块，有无稀疏目录同名的目录）中，断言门禁之后**没有运行任何 git 命令**（例如用记录调用的替身 git），并且原 `.git` 下所有文件的内容与修改时间都不变；只断言最终的 `unknown` 不够。正对照：普通布局照常运行整套命令。
  - **存储检查**：K18 ~ K22（`.pack` 失去 `.idx`、备用目录中同样的情形、残留的临时文件、伴随文件、孤立的 `.rev`），断言存储检查不写入任何文件。
  - **浅克隆**：H1 ~ H7；检测与拒绝都不写入。
- **`IndexIsSparse` 的单元测试**：版本 2、3、4；扩展标志；拆分索引与它的共享索引；untracked cache；稀疏目录条目与 `sdir`；校验和错误、截断、未知版本、扩展越界都为错误。
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

**审查 057 的实现约束**（纳入可追踪的记录，见 [review-resolution.md §67](review-resolution.md)；它们是"完整解析、不能解析就失败"等已有要求的工程落实，不是新的产品决定）：

1. **索引解析的边界**：条目计数、整数、前缀回退、路径终止、对齐与扩展长度都不能越界或溢出；区分可以跳过的可选扩展（大写开头）与不支持的必需扩展（小写开头的必需扩展不能当作可忽略）。
2. **`link` 与共享索引**：引用、边界与必要内容有效；缺失、损坏或无法完整理解时失败；不能只扫描主索引而遗漏共享索引的条目；`link` 的哈希全零表示没有共享索引，不去找 `sharedindex.000…`。索引末尾的校验和全零是 git 有意省略（`index.skipHash`），与非零的错误校验和分开处理。
3. **门禁先于一切**：主索引或任一共享索引中有稀疏目录条目或 `sdir`，都在运行可能展开索引的 git 命令之前停止；更复杂的布局暂不支持时安全拒绝，不能猜测为普通索引。
4. **保守处理**：直接读取、复制失败与读取中途的变化都输出安全的错误，清理私有内容，不修复原索引。
5. **主索引不存在**（审查 057 N1）：真正不存在（ENOENT）的主索引按空索引处理（新建的空仓库，已有 HEAD 但没有索引）；零字节或截断的索引、缺失的被引用共享索引、权限与 I/O 错误 fail-closed。
6. **必须兑现的测试**：索引格式与失败分支；超级项目与子模块门禁顺序的调用记录；真实文件的内容与修改时间快照；pack / idx / alternates 的正反例；浅克隆的检测与恢复生命周期；严格读取失败之后，事务、`seq` 与游标都不变。
7. 命令的呈现属 WP2.7，E2E 属 WP2.9，不被 WP2.3 的验收提前覆盖。

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
| `status --porcelain -z --ignore-submodules=all`，`config --list --local --includes -z`，`--git-dir`、`--work-tree` | `Clean` | 未逐一核对 | Linux 上运行（F、S、N1） | 支持（C1 ~ C11、F、S、N1） | 实现时核对 |
| `ls-files --sparse`、`check-attr -z --stdin`、`diff-files --name-only -z`、`rev-parse --absolute-git-dir` | `Clean` 的第 3、4 步 | 未核对（`--sparse` 的引入版本未核实） | Linux 上运行（S、F、N1） | 支持 | 不支持时退出码 6 |
| `-c core.commitGraph=false`、`-c core.multiPackIndex=false`、`GIT_GRAFT_FILE` | 严格读取（§5.2） | 未运行 | Linux 上运行（G、K） | 支持（G、K） | 实现时在 CI 的三个平台上验证 |
| `GIT_SHALLOW_FILE` | 读取中的浅克隆防护（P3） | 未运行 | Linux 上运行（H5） | 支持（G6、H5） | 实现时在 CI 的三个平台上验证 |
| 浅克隆的检测、稀疏门禁 | §5.2、§4.3 第 2 步 | 不运行 git | 不运行 git | 不运行 git | Specmetry 直接检查 `shallow` 项、直接解析索引文件，与 git 的版本无关 |

- 2.56 是 Windows 上实际运行的结果；2.43 一列中标为"Linux 上运行"的，是审查 052 之后在 WSL Ubuntu 的 git 2.43.0 上实际运行的结果，其余仍来自审查 050 的静态核查；2.32 一列都是静态核查。
- **后果**：一个低于 2.36、却通过了 PD-26 探测的 git（例如回移了 `GIT_NO_LAZY_FETCH` 的发行版构建，是否存在未核实），每次提交扫描都会因 `worktree list -z` 返回退出码 6。
  - 这是 fail-closed 的拒绝，不是提高最低版本：初始化、基准与其他不扫描提交的操作不受影响。
  - 调用扫描的命令（`status`、`ingest`、`sync`）如何呈现这个错误，由 WP2.7 确定（§5.1 对 D7 的同一要求）。
  - 是否另外提供不依赖 `-z` 的兼容读取，列为待决事项（§8），本文不自行决定。
- 实现时在 CI 的三个平台上运行全部用到的命令，并为不支持的选项写出拒绝路径的测试（§11）。

## 14. 实现中的细化（审查 058 之后，待复审）

> 本节写于 2026-10-19，是 WP2.3 实现（PR #27）中发现、并按审查 058（NO-GO）修复后同步的实现细化，经审查 059 复审；其中 §14.3 已按审查 060 后负责人 B1-D1～D4 的明确批准调整（§71），实现结果待独立复核（[review-resolution.md §69](review-resolution.md)）。它们**不是审查 057 审查过的内容**，除 §14.3／§71 明确批准的局部保证调整外，不改变已冻结的业务语义：`clean` 的定义（data-model §1、D1）、P1 / P2 / P3、`worktree list -z` 的能力要求、关联的生命周期、`seq` 与写事务（§5.4、§5.5）都不变。记录见 [review-resolution.md §68](review-resolution.md)。

### 14.1 子模块：被 `--ignore-submodules=all` 隐藏的四种变化（§4.3 第 3、4 步）

- **发现**：超级项目的私有 `status` 固定 `--ignore-submodules=all`，它不只隐藏子模块自身的状态，还隐藏以下四种变化；定义命令（`=none`）都会显示它们（git 2.56 与 WSL 的 2.43 结果相同）：
  - 已暂存的 gitlink（`M  sub`）；
  - 从索引中移除的 gitlink（`D  sub`）；
  - 子模块目录被删除（` D sub`）；
  - 子模块目录被换成文件（` T sub`）。
- 只按 §4.3 第 4 步比较子模块的 HEAD 与 gitlink，这四种情形会被报告为 `clean`。
- **补充的读取**：
  - 在私有目录上运行 `git diff-index --cached --raw -z --ignore-submodules=none <HEAD> -- <pathspec>`，比较索引与 HEAD 的树。它只比较索引与树，不进入子模块的工作区，trace2 显示没有子进程。有输出即为 `dirty`。分支还没有提交时，索引中有 gitlink 即为 `dirty`。
  - 对索引中每个 stage 0 的 gitlink，检查它的路径：不存在为 `dirty`（` D`）；不是目录为 `dirty`（` T`）；是目录而没有 `.git` 时为未初始化，与定义相同，不读取；有 `.git` 时按 §4.3 第 4 步读取。
  - stage 不为 0 的 gitlink（冲突中）为 `dirty`。
- 这是保持定义所必需的补足，只对这几种情形成立；**不是**"私有读取与定义的差异只会多报 `dirty`"的一般证明。

### 14.2 严格日志不运行签名程序（§5.2 第四步）

- 仓库 local 配置有 `log.showSignature=true` 与 `gpg.program` 时，§5.2 第四步的命令按原文会为带签名的提交运行 `gpg.program`（两个版本都复现）。这违反"不执行仓库配置的程序"（architecture §6.1.2）。
- `Log` 因此另加 `--no-show-signature`（git 2.10 起支持）。测试先以普通 `git log` 证明夹具确实会运行标记程序，再证明扫描的读取不运行它。

### 14.3 私有临时区：获批方案 B（Review 060 之后）

负责人已批准 B1-D1～D4（[决定 §71](review-resolution.md#71b1-方案-b-的批准与实施范围)）。这是**局部保证调整**，不是宣称审查 060 的旧保证已全部实现。历史反例及其精确窗口保留在原审查与 [设计报告](wp2.3-b1-design.md)。

扫描：所有扫描对象调用显式传 `GIT_GRAFT_FILE=` 与 `GIT_SHALLOW_FILE=`；首次使用实际 Git 二进制前，在受保护的 v3 租约内以正反对照验证两空值完整读取、缺失／损坏严格失败。缓存绑定文件身份、大小与修改时间，变化重新探测；失败 exit 6，不回退路径。扫描实际读取不再使用私有防护路径。P3 浅克隆拒绝、PD-26、禁传输／外部程序、部分结果丢弃不变。

Clean 与探测：`NewScratch` 先执行 `datadir.Ensure/EnsureSub`，不打开数据库。在 `tmp/git-reader-v3/` 中使用稳定合作锁，锁覆盖每个私有目录的建立、写入、Git 使用及清理；递归子模块逐个租约，不并行使用同一租约。竞争环境错误，不自动重试业务写入。正常合作进程互不清理活跃对象。v3 与 v1 顶层 `tmp/read-*`、v2 子树隔离，不接管两者的遗留；v2 没有消费者结束记录，不能安全升级认领。v3 内未知格式或条目保留并失败，不混用协议。旧二进制并发运行尚未动态验证；隔离依据为各版选择路径的源码与命名空间测试。

**消费者生存期（Review 061 B1-R2，决定 §72）**：调用方持锁不能证明其崩溃后的真实 Git 已退出。每次消费私有目录的 Git（包括 Clean 全部有关调用与空覆盖行为探测）启动前，先将 owner 的 `consumer_pending=true` 完整写入并同步；不能建立屏障时不启动。只有实际 `Start` 失败，或已经 `Wait` 到 Git 结束，且完整输入检查通过，才写回并同步 `false`。取消同样等待实际 Git 退出。合作回收者一律拒绝未清除标记、损坏／缺失标记及无效记录，保留所有现存内容、返回 exit 6；不以父进程死亡、锁释放、PID、年龄或等待时长判断消费者结束。本机制不依赖 fd／handle 继承，也不执行仓库配置的程序。

**完成持久化失败（Review 062 B1-R2-S）**：已 Wait 到 Git 结束与完成记录写入、Sync 成功是两个状态。只有后者成功，本次调用才允许清理／复用；完成记录写入或 Sync 失败时，保留本次不可覆盖的未确认状态、返回环境错误，清理只释放自身资源，保留目录、owner 及全部现存清单内容。不因当前已能读到完整 `false` 而忽略本次 Sync 错误，不通过再次 begin／finish 自动重试消除错误。

后续 Scratch 按现行 v3 的实际可读记录、身份、清单及锁认领，不持有前次 Sync 的返回值。若仍是 `true`、记录不完整／无效或归属无法确认，保留并失败；若是完整规范 `false` 且全部归属预检通过，仍可可信回收（该记录仅在 Git 已结束并验证输入后生成）。这不证明前次 Sync 成功，也不承诺一次 Sync 错误永久禁止跨进程回收；若记录未保存下来，按其实际不确定状态拒绝。本次修复不升级协议、不增加自动恢复写入或恢复命令。

调用方崩溃可能留下永久不可自动回收的目录，即使 Git 后来退出也如此；没有完成记录，不能从当前不活跃反推整个生存期安全。该分支明确保留并失败，需负责人／操作者另行确认消费者均已结束后人工处理，本轮不新增自动恢复命令。无在途消费者的可信完整遗留，或已由调用方持久记录正常／取消结束的遗留，仍可按清单回收。这是安全失败的可用性代价，不是把调用方崩溃归入不合作同用户排除范围。

Windows 原子 FILE_CREATE 返回目录句柄，已有名字拒绝，不使用 OPEN_IF；持有必要目录保护句柄，删除在核对身份的同一个句柄上进行。Unix mkdir/open 与 fstatat/unlinkat 仍是分离操作，合作锁内没有其他合规名称修改者。所有平台的根链与条目检查只对发现的变化保守拒绝，不声称排除未观测到的往返替换。

每个临时目录以创建清单记录相对路径、类型和身份；正常结束按清单清理，先复核所有内容，不递归猜测删除未知树。**在任何删除之前**必须验证路径上的 owner 存在、是普通文件、身份为原创建身份且记录与当前清单／消费状态一致；仍有效的旧文件句柄不能代替名称上的存在性。读取前后、清理预检及遗留认领都执行此校验。发现未知、额外、缺失或身份不符时保留对象、停止操作并返回环境错误；不能先删其他内容再发现 owner 缺失。只有本次可信清理已成功删除 owner 后，才进入不再要求该文件存在的目录删除重试阶段；进程崩溃后不能据此认领无 owner 目录。

只有有效 v3 owner 记录、可取得锁、清单与对象一致且没有未确认消费者的崩溃遗留可清理；不按前缀、年龄或 PID 推断归属。失败只释放自身句柄和锁。清理失败 exit 6，不能报告 clean／发布扫描事实。协议需要的最小锁和标识长期保留，不含项目内容。

**获批的局部边界**：不合作同用户进程干预 Git 私有临时命名空间（包括普通误操作）被排除；Unix mkdir/open 可接纳空替代目录，最后比较/unlink 或 rmdir 可删替代 owner 文件或空目录；Clean 可读替代输入，不能排除 false-clean。这不扩展到其他账户、合作进程并发、数据库、来源权限、隐私或交付证据；不豁免 Log 漏读成功。管理员/root 的主动控制仍不作为普通用户防线。Windows 重命名被 OS 阻止与实际替换后拒绝分别记录；未做的双账户／文件系统验证不得外推。

公开 Scratch API、缺少／关闭时的内部错误不变。调用方正常 Close；取消等 Git 子进程结束后才清理与释放租约。读取失败不发布提交、关联、seq 或游标，沿用扫描事务，不引入业务重试。

### 14.4 祖先判断：祖先本身缺失与路径中间的提交缺失（§5.2 第三步）

- **祖先本身缺失**（实验 G2）：`merge-base --is-ancestor <A> <C>` 在 git 2.56、2.43、2.32 上都返回 128，与原文一致，扫描失败（退出码 6）。
- **路径中间的提交缺失**：
  - 关闭 commit-graph 时，git 2.56 返回 128；git 2.43 与 2.32 返回 **1**（"不是祖先"）。三者都不会返回 0。
  - 2.43、2.32 的结果是作者在 WSL 中的实验记录；CI 上的 2.55、2.56 在测试中都报错。
- 原文"关闭 commit-graph 之后，祖先判断也要读取对象（G2）"只对 G2 的情形成立。
- **扫描的保证不变**：
  - 返回 1 只会使该 ref 按改写处理，读取整段历史；随后的严格 `log` 在缺失的提交处失败，整次扫描不写提交、关联、`seq` 或游标。
  - 实验中，路径中间的提交缺失时，三个版本都没有返回 0。即使返回 0，随后的增量读取 `<T> ^<A>` 仍然严格：它访问到的每一个提交都要读取，缺失即失败。

### 14.5 worktree 的元数据文件（§4.3 第 1 步）

- `info/exclude`、`info/attributes`、`info/sparse-checkout` 按 `git rev-parse --git-path` 给出的实际位置复制，这正是 git 自己读取的位置。
- 主 worktree 的这三个文件都在公共目录中，与原文一致。关联 worktree 的 `info/sparse-checkout` 在它自己的 git 目录中（git 的 per-worktree 规则），不能统一写成公共目录。

### 14.6 `.specmetry/` 的排除在所有读取中一致（审查 058 B3）

- 原实现只有 `status` 与 `diff-index` 带了定义的 pathspec，`ls-files` 列出整个索引。因此由它得到的过滤器候选（`check-attr`、`diff-files`、racy 判断）与 gitlink 检查，仍会受 `.specmetry/` 中内容的影响：那里一个受过滤文件的变化会使结果成为 `unknown`，那里一个子模块目录被删除会使结果成为 `dirty`，而定义命令的输出为空。
- 现在 `ls-files`、`diff-files` 与 `status`、`diff-index` 使用相同的 git pathspec：超级项目中为 `-- . ':(exclude).specmetry'`，子模块中为 `-- .`。子模块内部不排除：定义命令询问的是子模块自己的 `status`。
- 由 git 自己匹配 pathspec，不用字符串前缀，因此 `.specmetry-not/` 之类名称相近的路径照常计入。
- 整个索引的安全门禁不受 pathspec 影响：稀疏目录条目即使是 `.specmetry/`，也仍判为 `unknown`（P2）；损坏的索引仍是环境错误。

### 14.7 `link` 扩展的位图（审查 058 B2，§4.3 第 2 步）

- 一个 EWAH 位图至少有 12 字节：位数、字数与 running-length 位置。原实现只要求 8 字节：剩余 8～11 字节时，读取越过 `link` 的载荷；外层校验和为零（skipHash）时，`IndexIsSparse` 与 `Clean` 会 panic。
- 现在的要求：
  - 先要求完整的 12 字节框架；
  - 字数必须容得下剩余的长度；
  - running-length 位置必须是某个字；
  - 两个位图必须正好用完载荷；
  - 任何切片都不超出当前载荷。
- 损坏时返回环境错误，不 panic；在任何读取索引的 git 命令之前失败。

### 14.8 对应的测试（补充 §11）

| 项 | 测试 |
|---|---|
| 14.1 | `TestCleanSubmodules`、`TestCleanSubmodulesAgreeWithTheDefinition` |
| 14.2 | `TestLogRunsNoSignatureProgram`（含对照） |
| 14.3 | `TestScratchNeedsAPrivateDataDirectory`、`TestPrivateDirectoryReplaced`、`TestScratchRootReplaced`、`TestScratchRemovesOnlyItsOwnLeftovers`、`TestPrivateDirectoryLifecycle`、`TestPrivateDirectoriesArePrivate`；审查 059 B1-R：`TestRemovalAttemptsCheckTheDirectory`（反例 1）、`TestScratchRootReplacedWhileInUse`（反例 2）、`TestScratchRootRedirected`（反例 3）、`TestRemovalChecksWhatItRemoves`、`TestPrivateDirectoryReplacedAsItIsMade`、正对照 `TestRemovalRetriesTheSameDirectory`；跨用户：`TestCrossUserScratch*`（`internal/store/sqlite`，以 root 运行） |
| 14.4 | `TestIsAncestor`、`TestScanFailuresWriteNothing`、`TestScanDropsAPartialRead` |
| 14.6 | `TestCleanLeavesOutSpecmetry`、`TestCleanLeavesOutSpecmetryGitlinks`（都以真实运行的定义命令为对照）、`TestCleanSparseGateIgnoresTheExclusion` |
| 14.7 | `TestIndexIsSparseShortBitmapHeader`、`TestCleanWithAShortBitmapHeader`、`TestIndexIsSparseCutBitmaps`、`TestIndexIsSparseBadBitmapCounts`、`TestIndexIsSparseRandomDamage`、`FuzzIndexIsSparse` |

### 14.9 B1 方案 B 的实现证据（本轮实施，不是独立审查）

批准范围、代码／规范身份、平台与失败历史见 [实施交付记录](wp2.3-b1-implementation.md)。新测试包括 `TestScanEmptyOverrides`、`TestScanRefusesUnsupportedEmptyOverrides`、`TestScanCapabilityFollowsExecutableIdentity`、`TestScanProbeCancellationReleasesLease`、`TestScanEmptyMultiParentAndFailures`、`TestScanScratchContract`、`TestScratchManifestAnomalyIsRetained`、`TestScratchVersionNamespacesDoNotMix`、`TestScratchProtocolProcesses`、Windows 的 `TestWindowsAtomicPrivateDirectory`。原私有目录替换调度现在针对 Clean；Log 不再使用私有防护路径，以公开 Log 的真实往返回归覆盖。

`TestCreationWindowWithApprovedBoundary` 与 Unix 的 `TestApprovedUnixFinalWindow` 明确记录获批的残余窗口，其通过表示边界如实保留，不表示域外干预已被阻止。早期 hook、最后身份核对／unlink 与实际 Git 消费是不同层次；不改写 Review 060 原文。B2、B3、N1 保持关闭，B1 结果交独立会话复核。
