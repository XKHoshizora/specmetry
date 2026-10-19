# Codex Review 058：PR #27 WP2.3 Git Reader 首次独立代码审查

审查日期：2026-10-19。审查者：Codex。

**NO-GO。当前实现不足以支持 WP2.3 本层验收，PR #27 暂不应合并。** 发现三项实际阻塞：私有目录没有保护祖先路径和目录身份；截断 link 位图可使公开读取接口 panic；被排除的 `.specmetry/` 内容仍能影响 Clean。正常读取、扫描事务和迁移的多项证据成立，但不能抵消这些缺陷。

## 一、身份、依据与边界

- [PR #27](https://github.com/XKHoshizora/specmetry/pull/27)。
- 固定 base：`e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。
- 固定 HEAD：`0c9c51d37a39bb53a6d8da0430fa3c8668f464af`。
- 完整阅读 base 中冻结的 [Git Reader 规范](../wp2.3-git-reader.md)、对应 data-model／architecture／验收项、[Review 057](codex-review-057.md) 及 PR 描述。冻结规范与实际实现分别判断，未把作者的新发现倒写成 Review 057 已审内容。
- 开始及收尾核对远程 PR：均为上述 HEAD/base，状态 Open，没有混入其他提交。

实际增量为 26 个文件：gitx 读取、索引解析、私有目录及测试；gitscan 及测试；迁移 0007 和约束测试；测试用 Git 的一个新模式；开发文档。0001～0006、冻结推导规则、Schema、CLI、生产 Open／重试未修改。

原工作区开始时干净，位于固定 HEAD；报告此前不存在。未切换、清理或改动其他任务工作区。固定树解包到仓库外，包测试和独立测试仅使用临时仓库、临时数据库；路径替换的同步钩子只加在隔离副本中。原仓库唯一新增文件为本报告。

## 二、BLOCKER

### B1：系统临时目录下的子目录 DACL，不能替代祖先路径和目录身份保护

**位置：**[privdir.go:46～74](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/privdir.go#L46)、[清理路径:86～133](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/privdir.go#L86)、[privperm_windows.go](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/privperm_windows.go#L9)、[scan.go:33～47](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/scan.go#L33)。

newPrivateDir 先在 os.TempDir 下 MkdirTemp，再按路径修改 DACL、检查为空、创建并锁住 lock 文件。没有验证临时根及祖先路径的删除／重命名权限，没有保留和复核最初创建的目录身份；Windows 的 securePrivateDir 只设置 DACL，没有验证替换后目录的属主。后续写配置、复制索引、Git 使用和删除仍靠同一个路径字符串。

Windows 删除／重命名权限可以来自父目录的 FILE_DELETE_CHILD；只收紧子目录 DACL 不能消除这条路径。新建对象默认继承父目录安全描述符，而移动／重命名不会重新赋予默认描述符。[Microsoft 文件权限说明](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-deletefilew)、[文件安全描述符](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights)

**可达条件和源码证明：**

1. TMP／TEMP 本身或一个祖先允许其他账户替换目录。实现没有拒绝这种配置，并非仅在一个不可达的内部函数上假设不可信参数。
2. 另一个账户在 MkdirTemp 返回后、取得 lock 以前，把刚创建的目录移走，在原路径放入空目录或重定向路径。
3. 后续按路径的 DACL 设置及空目录检查可以作用于替代物；空目录检查不证明它是最初创建的目录。如果替代目录由该账户拥有，仅设置 DACL 也不使它变成可信属主：属主可以修改权限。[Microsoft 对象属主说明](https://learn.microsoft.com/en-us/windows/security/identity-protection/access-control/access-control)
4. O_EXCL 只保证创建时该文件名不存在，不能证明父目录身份，也不保证文件之后不被替换。字节锁作用于 lock 文件；Windows 打开的句柄可能阻止部分后续重命名，但这里的替换窗口在取得锁之前已经存在。它也不证明该目录的属主可信。Unix flock 同样不是路径身份或祖先权限保护；标准 sticky 临时目录的较强条件不能推广到任意非 sticky TMPDIR。

**独立验证，证据层次分开：**

- 在 Windows 隔离临时根上显式添加允许其他用户替换的 ACL。newPrivateDir 成功；完整 datadir.Check 明确报告祖先不安全。没有修改系统临时根的 ACL，也没有使用真实客户目录。
- 在隔离副本仅于 MkdirTemp 返回后插入一次同步钩子：写入身份见证，将原目录移到旁边，在原路径建立空目录；通过见证确认是替代目录，随后继续原代码。公开 Clean 返回 **clean、nil**，没有发现身份替换。
- 这个调度使用同一测试账户，证明实际调用链不验证替换；它不是“真实双账户已经执行了恶意程序”的实验。跨账户可达性来自上述 ACL 条件、Windows 权限规则和完整源码顺序；本轮没有冒充双用户动态验证。
- 额外在隔离临时根建立一个未经过 privateDir 构造器、只有匹配前缀和旧 mtime 的目录。公开 Clean 删除了其中的无关见证文件。当前清理器在系统临时根中，仅凭前缀、年龄／可锁文件不能证明内容属于本工具；没有据此声称 Go RemoveAll 会跟随所有符号链接删除外部目录。

**影响：**这同时覆盖 Clean 的白名单配置／索引副本，以及扫描的 grafts／shallow 防护路径。后者把 `<private>/none` 当作必定不存在的文件；若名字空间被替换，这个假设也失效。读取受攻击者控制的配置或防护文件，可能破坏“不执行仓库配置的程序”“严格遍历、不隐瞒缺失对象”的核心保证。这里的后续影响是源码组合推导，不是本轮已完整执行的跨账户攻击链。

d0b2dad 修正了直接继承宽 DACL 的一部分问题，但没有关闭以上窗口。[TestPrivateDirectoriesArePrivate:827～845](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/clean_test.go#L827) 调用 datadir.Check 后，只对以 p.gitDir 开头的问题失败，恰好丢弃了祖先路径报告；因此该测试通过不能证明祖先安全。

**最小修复方向：**共用 newPrivateDir 的所有路径现在就使用受保护、经过祖先／链接／属主检查的 Specmetry 专用临时区，优先放在既有受保护数据目录下；或提供能证明等效安全的机制。保证创建、权限设置、写入、Git 使用、清理都处于可信名字空间，不能只在创建后追加一次普通路径检查。清理只处理可信根内能够归属本工具且确认不活跃的遗留，保留锁与身份检查；不要全局扫系统临时目录并删除不明内容。补足不安全祖先、替换调度、活跃锁及非工具目录的正反例，并覆盖 Clean 和扫描读取。

这是当前库自身的安全契约，不能因暂无 CLI 调用而延期到 WP2.5，也没有获准的安全豁免。

### B2：8～11 字节的截断 EWAH 头在 skipHash 下导致 panic

**位置：**[index.go:288～297](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/index.go#L288)、[ewahSize:308～321](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/index.go#L308)。

ewahSize 只检查 len(b) < 8，但一个零 word 位图仍需 12 字节（bit count、word count、RLW position）。当长度是 8～11 时，`(len(b)-12)/8` 的负值按 Go 整数除法截断为 0，words=0 通过。后续切片可能从底层数组读到扩展外的校验和；若那里为零，就返回 size=12，readLink 再执行 rest[12:]，越过实际 payload 长度。

**独立最小反例：**

- DIRC，version=2，entry count=0。
- link 扩展，声明长度 28：20 字节零共享哈希，加 8 字节零位图头。
- 索引尾部 20 字节全零，使用 Git 允许的 skipHash 外层形式；link payload 本身是应当安全拒绝的损坏输入。最小文件共 68 字节。
- 公开 IndexIsSparse 对此 panic：`slice bounds out of range [12:8]`。
- 将相同文件放到一个隔离、无提交仓库的 .git/index，公开 Clean 也 panic。
- 将位图头改为 9、10、11 字节分别复现 `[12:9]`、`[12:10]`、`[12:11]`；两条公开路径共八个断言均捕获 panic。

普通非零校验和下的同类候选可以被拒绝；不能把这种成功拒绝推广到合法 skipHash 封装。现有 EveryPrefix 用例没有截取并同步修正 link 内层的这种长度，不能证明所有嵌套边界都安全。现有 split 位图用例也没有覆盖这一组短头。

**影响：**损坏／不可信索引使进程崩溃，不能按冻结规范返回 environment／exit 6；Clean 的正常清理流程也可能被中断。没有证据表明这里产生数据库写入，不把 panic 扩大成已发生的数据损坏。

**最小修复：**先要求位图至少 12 字节，再检查 word 数、总尺寸与 RLW 范围；任何切片前确认 n 不超过当前 payload 长度，避免依赖底层 capacity。对两个位图分别做逐长度截断、正确外层校验和与 skipHash 对照，并经 IndexIsSparse、Clean 验证返回错误且清理完成。无需重写整个索引解析器。

### B3：辅助枚举未沿用根级排除规则，`.specmetry/` 仍能影响 Clean

**位置：**[clean.go:459～479](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/clean.go#L459)、[filtered:529～578](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitx/clean.go#L529)。

status、diff-index 对顶层使用 `:(exclude).specmetry`，但 ls-files --stage 枚举所有路径，随后 paths／links 不加筛选。filtered 又对这些路径运行 check-attr／diff-files 并判定 racy，因此已被定义排除的文件仍能触发 unknown；同样的范围遗漏也应在 gitlink 遍历中避免。

**独立反例：**在临时仓库提交：

```text
.gitattributes: .specmetry/probe.txt filter=review058
.specmetry/probe.txt: original
```

随后配置合法的本地 `filter.review058.clean=cat`，只修改 `.specmetry/probe.txt` 并使其 mtime 更新。受控环境中的冻结定义命令：

```text
git status --porcelain --untracked-files=all --ignore-submodules=none -- . ':(exclude).specmetry'
```

成功且输出为空；公开 Clean 却返回 **StateUndecidable、nil**。没有依赖执行危险过滤器、网络或其他用户项目。先用仅声明 required 的配置观察到同一差异，再补有效 clean 配置重验，结果一致。

**冲突和影响：**data-model §1、MVP-EV-18 明确规定 `.specmetry/` 不计入。P1 允许在确实需要比较受过滤内容时保守 unknown，不允许排除目录中的内容单独改变结果；这里并不需要判定该内容。错误 unknown 会使本来合格的观测执行被拒为不合格，不能仅以“没有错误 ready”当作符合 Clean 契约。

**最小修复：**顶层辅助枚举／属性候选／gitlink 遍历沿用相同的 Git pathspec 排除语义；不要用宽泛字符串前缀误排除 `.specmetry-not/`。保留整个索引的 P2 稀疏门禁，不为修复范围筛选而跳过索引安全检查。增加被排除的受过滤路径、相关 gitlink 和普通相邻目录对照；子模块内部的排除规则仍按冻结的递归语义处理。

## 三、已成立的实现与证据

### 1. 读取层与错误分类

- TreeEntry 单路径 literal ls-tree、根树记录、mode/type/OID、规范化及缺失层次与冻结规则一致。叶 blob／目录自身缺失与读取路径所需的父树／根树缺失有区分；没有通过读取文件内容来替代元数据。
- call.run 对对象读取使用 objectGit／listGit；TreeEntry、Clean 的对象命令、Tips／Classify／IsAncestor／Log 均有门禁。CheckShallow 和 worktree 布局读取不假装需要读取对象。新增 commandIO 保留旧 command 的默认行为，stderr 仅记录有无，未把 Git 原文包装进错误。
- Tips 的 NUL worktree 协议、分离 HEAD、必需 -z 能力拒绝成立。Classify 保留双向配对、校验和、alternates 和无法确认时失败，不把空 idx 集合当作不存在的证据。
- 严格扫描关闭 commit-graph／multi-pack-index／replace 路径，并覆盖 grafts／shallow。B1 仍使“私有防护文件路径可信”这个前提不足，故不能给整条安全链 GO。
- 独立四提交夹具：写 commit-graph 后移走中间提交，公开 Log 已调用一次回调，随后 exit 128；没有把部分输出的出现当作成功。独立 pack 夹具移走 idx，公开 Classify 返回 environment，原 .git 快照不变；恢复 idx 后返回 Present。alternates、静默 idx 损坏和正常回收的其他正反例在原包测试与 CI 实际通过。

### 2. Clean、子模块与索引

独立手写四种子模块变化，均以冻结命令为对照，而不是把两个实现相同当成证明：

| 变化 | 定义命令的实际输出 | Clean |
|---|---|---|
| 暂存新的 gitlink | `M  sub` | dirty |
| 从索引移除 gitlink、目录仍在 | `D  sub` 与未跟踪目录 | dirty |
| 删除子模块目录 | 删除标记 | dirty |
| 以文件替换目录 | 类型变化标记 | dirty |

四项通过；这是保留冻结定义的必要实现补足，不是“所有差异只会多报 dirty”的一般证明。

原有包测试实际覆盖过滤器两个方向与 racy、子模块递归、gitfile、linked worktree、worktreeConfig、unborn、缺失主索引、split index、skipHash、未知必需扩展、解析失败、稀疏门禁先于索引命令，以及原对象内容／mtime 不变。旧顺序的 control 在本机与三个 CI 平台都实际改动三个 .git 条目的 mtime。门禁布局与正常布局对照有实际调用记录；这支持相应用例，不消除 B1～B3，也不是任意 Git 布局的穷尽证明。

版本 2／3／4 的主要解析方式与官方格式一致；link／shared index 和全零哈希、可选／必需扩展已有分支。B2 是嵌套长度检查的具体遗漏，不能因这些正常夹具通过而忽略。[Git 索引格式](https://git-scm.com/docs/gitformat-index)

### 3. 扫描事务、关联与迁移

- 计划使用已读游标；写入前 BEGIN IMMEDIATE 内比较 last_tip、键集合，符合 §5.5 对并发跳过的定义，不要求用时间戳判断先后。
- 编号匹配、trailer 优先、同 SHA 去重、首次入库关联固定、未入库提交的后续发现、不同 ref 的范围、共享一个 seq、空扫描不分配 seq、保留已删除 ref 游标等路径符合冻结规则。
- 读取失败在写入事务之前退出；写入失败由 WriteTx 回滚。readPlan 实际使用短的 WriteTx 取得计划，而非普通读事务，会多占一次写锁，但本轮没有发现由此导致的错误事实或破坏隔离，不把工程锁选择机械升级为产品决定。
- 原测试的 dbState 不只检查行数，也保存提交的 sha／seq／时间、关联及游标全部列。独立另加临时 INSERT 触发器，在 commits 已进入事务、commit_links 即将写入时拒绝；完整业务表全部列的前后快照一致，seq、既有游标及事实未留下变化。
- 迁移 0007 的同项目复合外键、target 类型／项目 BEFORE INSERT 校验、唯一性与 append-only 触发器成立；正常 UPDATE／DELETE／REPLACE 拒绝，维护删除仍受外键约束，purge 按关联顺序移除，其他项目保留。约束测试本机通过。
- 关联不能以后补写的保证按冻结方案由唯一入口、已存 SHA 跳过和回归共同提供，不能把 INSERT 触发器误称为能证明“同一事务首次关联”。没有要求引入已排除的事务令牌模型。
- 提交消息及 trailer 仅在内存里匹配，持久化候选只含 sha、时间、编号与来源。游标名称写入前密钥扫描；原隐私测试检查数据库／WAL／SHM，业务错误采用固定标签。本轮没有发现新的原文持久化或错误回显反例；B1 的私有空间完整性风险仍单独阻塞。

## 四、NON-BLOCKER

**N1：进程内并发跳过测试的“对方提交后快照”取晚了。**

位置：[scan_test.go:808～833](https://github.com/XKHoshizora/specmetry/blob/0c9c51d37a39bb53a6d8da0430fa3c8668f464af/internal/gitscan/scan_test.go#L808)。afterInner 实际在外层 Scan 返回后才读取；末尾再比较 e.state，无法检测外层已在返回前错误改写字段。建议在钩子里的 inner 扫描成功后、外层继续前捕获快照，再与最终状态比较。

同文件的真实跨进程测试在第二个进程完成后、释放第一个进程前捕获状态，提供了更有效的比较；生产比较逻辑与独立回滚对照也成立。因此这是一个具体的测试断言改进，不把它升级为确定的事务算法缺陷。

## 五、实现发现的规范同步清单

本 PR 没有修改冻结规范，这一点如实记录；以下应在修复收尾时同步，不能称作 Review 057 已验证的实现细节：

1. **子模块四种变化：**在 §4.3、EV-18 测试映射补充 cached diff-index 和缺失／非目录 gitlink 路径的检查。保留同一冻结定义，不批准泛化的“只会多报 dirty”。
2. **签名程序：**§5.2 的 Log 命令明确 --no-show-signature。原 TestLogRunsNoSignatureProgram 构造带签名的提交，先验证普通 log 确实写入标记，再删除标记验证 Log 不执行；本轮包测试与三平台 CI 均通过，对照有效。这是落实既有不执行外部程序承诺，不是新产品取舍。
3. **私有空间：**记录 B1 修复后的实际可信根、创建／身份／锁／遗留清理边界，覆盖 Clean 与扫描；删除“留待 WP2.5 接线”的安全延期表述，不新增豁免。
4. **祖先读取：**区别“作为参数的祖先本身缺失”和“路径中间提交缺失”。后者在部分 Git 版本可能返回 1；1 只能走保守全量重扫，严格 Log 之后失败则整次不写。相关原测试通过；2.32／2.43 的具体返回值来自作者记录，本轮没有独立重跑这些版本。
5. **worktree 元数据路径：**info/sparse-checkout 等按 --git-path 的实际位置复制；明确 linked worktree 的私有位置，不能统一写成公共目录。
6. **边界与范围：**加入 B2 的短位图嵌套测试和 B3 的排除路径一致性；保持 Review 057 的正常空索引、未知必需扩展和门禁前置义务。

**PRODUCT DECISION：本轮没有需要新增核心风险接受才能继续的事项。** B1～B3 是现有契约的修复，不应改成要求负责人批准降低安全／准确性。必要的小范围调用接口调整属于实现设计，另行审查即可。

**ACCEPTED RISK：只保留 §62 已批准的关联生命周期限制。** P1／P2／P3 和 worktree list -z 的能力要求继续按已批准范围执行；它们不豁免 B1～B3。未新增接受风险，未关闭 M-14／M-15。

## 六、独立运行与 CI

### 本轮本机

Windows/amd64，Git 2.56.0.windows.2，Go 1.27.1。仓库外固定树，离线依赖模式：

| 验证 | 结果及含义 |
|---|---|
| 原有 gitx 包测试，nowebui、count=1、8 分钟上限 | exit 0；439.680 秒。日志未包含后加的 Review058 测试 |
| 原有 gitscan 包测试，同一次运行 | exit 0；105.166 秒，含真正跨进程竞争 |
| TestCommitTablesConstraints | exit 0；测试体约 0.10 秒 |
| 独立四种子模块变化、commit-graph／部分输出、pack 丢失 idx／恢复 | 通过相应预期与原 .git 不变断言 |
| 独立事务中途拒绝 | 正确失败；全表列值、seq、游标一致 |
| 独立短 link 位图 | 普通非零外层校验和候选被拒绝；skipHash 的 8～11 字节候选在 IndexIsSparse／Clean 均 panic，按预期检出 B2 |
| 独立排除路径 | 有效 clean 驱动下仍错误 unknown，检出 B3 |
| 不安全临时祖先／创建后替换／未登记前缀目录 | 分别确认没有拒绝、Clean 返回 clean、无关见证被删除，支持 B1；替换是同账户故障调度，不冒充双用户完整攻击 |

独立调度钩子的初次测试接线编译失败已修正，不作为生产缺陷证据。除路径替换需要的调度钩子外，解析、范围和事务反例调用的被测实现未改。没有把独立测试故意检出问题的 exit 1 写成原包回归失败。

本机没有运行 CLI 整包、Linux／macOS、Race Detector、真实旧 Git、真实双用户切换、实际网络加深、真实部分克隆、并发 gc、LFS、长驻过滤器或大型仓库基准。作者报告的 CLI 默认超时、单独运行耗时和 master 对照保留为观察；本轮没有独立证明根因是并发负载，也不要求为此修改全局超时。

### 最终 CI 与历史

[run 39165490748](https://github.com/XKHoshizora/specmetry/actions/runs/39165490748)：Attempt 1，HEAD 为固定 0c9c51d，8/8 success。逐个核查八个作业日志，实际检出均为：

`2dd41f6c10789323e1b4a77a09f6cb973c9fc0ab`

其父提交依次是 base e0a21b4、HEAD 0c9c51d；树为 `c98223c1fc5480a078cbfc9571a59a2e9364114d`，与固定 HEAD 的树一致。作者所报 2dd41f6 身份准确。

- Web、Go lint、三平台 Go tests、三平台 dashboard 构建全部成功。
- Windows 作业 117787442944，Git 2.56.0.windows.2；Ubuntu 117787442958，Git 2.55.0；macOS 117787442984，Git 2.56.0。Go 均为 1.27.2。
- Ubuntu Test 步骤按 Linux 分支使用 -race，Linux 专属 root 跨用户步骤成功；后者仍是既有路径测试，不能当作 B1 已经做过双用户覆盖。Windows／macOS 按设计跳过该 root 步骤。
- 三平台的正常索引、Clean、扫描、迁移等新测试实际运行。TestWP23ReadsWithAGitThatFetches 三平台均 SKIP；TestClassifyWithUnreadableStorage 在 Windows SKIP，在 Ubuntu／macOS 的 K10～K16 实际 PASS。
- 作者 WSL 2.43、真实 Git 2.32 记录不是本轮独立复验，也不是本次 CI 的版本。unsafe 模式的专项只能证明门禁拒绝；跳过其他对象读取测试不能称为该版本完成了正常扫描验收。原始 Git 返回值实验也不等于生产入口绕过 PD-26 后执行。

保留前两轮失败：

| 运行 | SHA | Attempt／结果 | 失败内容 |
|---|---|---|---|
| [39163839528](https://github.com/XKHoshizora/specmetry/actions/runs/39163839528) | d0b2dad758d15b8b7b25cf8315169a79ae7c364d | 1；7/8 | lint 7 项：unlambda、3 个 unparam、3 个 G703；三平台 Go tests 成功 |
| [39164780269](https://github.com/XKHoshizora/specmetry/actions/runs/39164780269) | 5d9b93ffb4d918f92c39de3a2231f490c8f81008 | 1；7/8 | lint 另 2 个 G703；三平台 Go tests 成功 |
| [39165490748](https://github.com/XKHoshizora/specmetry/actions/runs/39165490748) | 0c9c51d37a39bb53a6d8da0430fa3c8668f464af | 1；8/8 | 无 CI 失败 |

这些是不同 HEAD 的首轮运行，没有改写成整个 PR 从未失败或某次失败已经通过重跑。最终 CI 绿色没有覆盖本轮新手写反例，不能替代独立安全判断。

## 七、最终裁定与接手建议

1. **PR #27：NO-GO，暂不合并。** 当前证据不足以支持 WP2.3 本层验收；不宣布工作包已验收。
2. 修复 B1 的可信私有临时区／身份与清理保护，必须同时覆盖 Clean 和扫描的防护文件路径，不能延期到后续工作包。
3. 修复 B2 的 EWAH 最短长度与切片边界；补两条公开路径和清理结果回归。
4. 修复 B3 的根级排除范围一致性，保留全索引的 P2 门禁。
5. 同步第五节的实现发现和测试映射；N1 可作为非阻塞测试改进处理。修复后针对这三项及直接回归复审，再由负责人决定合并与验收。

没有要求提前接入 WP2.7 命令、生产 SQLite 复核器或 WP2.9 E2E，也未重审已验收的 WP2.1／WP2.2、M0／M1 或诊断工具。未修改实现、提交、推送、合并，不关闭 M-14／M-15，不授权或启动后续工作包。
