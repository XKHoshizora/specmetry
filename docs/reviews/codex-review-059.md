# Codex Review 059：PR #27 Review 058 修复定向独立复审

审查日期：2026-10-19。审查者：Codex。

**NO-GO。Review 058 的 B2、B3、N1 可以关闭；B1 尚有可复现的替换后误删与身份检查缺口，不能关闭。PR #27 暂不应合并，当前证据不足以支持 WP2.3 本层交付验收。** 将私有空间迁入受保护的数据目录是有效修正，但现有清理不能始终证明被删除对象属于本次读取。绿色 CI 不覆盖本轮独立构造的这些调度。

## 一、固定身份、范围与证据来源

- [PR #27](https://github.com/XKHoshizora/specmetry/pull/27)，开始及收尾均核对为 Open。
- 固定 base：`e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。
- 固定 HEAD：`4519e7a37a744e2b2d0dc2da719314964267c0d0`。
- 上轮对象：`0c9c51d37a39bb53a6d8da0430fa3c8668f464af`。
- 开始及结束的远程 HEAD/base 均与上述身份一致，没有混入其他提交。
- 完整阅读 [Review 058](codex-review-058.md)，审查上轮对象到本轮 HEAD 的增量及必要调用链；复用其对未变读取层、扫描事务、关联、迁移和其他 Clean 判据的结论，没有重新进行整个 WP2.3 或 M0／M1 审计。
- 本轮应用安全修复验证的方法追踪原路径、保护点、使用和删除位置；故障调度仅操作隔离副本和测试自建目录。没有执行真实跨账户的 Windows 攻击。

增量为 26 个文件：Scratch／私有目录重写、Clean 排除规则、位图边界、调用点与测试、跨用户测试基座、datadir 的 `TmpDir` 常量，以及文档和 Review 058 归档。0001～0007 迁移、推导规则、CLI、Schema、生产 SQLite Open／重试／错误映射未修改。

提交依次为 `9648958`（归档）、`1fe57c5`（B2）、`9746459`（B3）、`3901c55`（B1／N1）、`2b73eb9`（文档）、`8dfb31e`、`4519e7a`（测试 lint 修正）。没有把本轮实现细化记作 Review 057 已审内容。

## 二、逐项裁定

| 上轮事项 | 本轮裁定 | 主要依据 |
|---|---|---|
| B1：私有目录的祖先路径、身份与清理 | **未关闭** | 不安全祖先的拒绝和正常生命周期成立；删除重试、替代锁文件和活动读取的 Scratch 根身份仍有实际缺口，见第三节 |
| B2：截断 link 位图 panic | **关闭** | 原 68～71 字节反例旧版双入口 panic，新版双入口受控错误；内外两道载荷边界成立 |
| B3：`.specmetry/` 排除不一致 | **关闭** | 路径候选、过滤器／racy 和 gitlink 采用同一 pathspec；独立定义命令对照成立，整体索引门禁保留 |
| N1：并发跳过快照取晚 | **关闭** | 快照在内层提交后、外层恢复前取得；正常测试通过，额外游标字段写入变异被检出 |

## 三、BLOCKER：B1-R，替换后的清理和根身份仍不完整

这是同一 B1 的残余，不是重新开启无关模块。以下位置均对应本轮固定 HEAD。

### 1. 删除重试没有重新核对身份，可以删除替代目录并报告成功

位置：[privdir.go:222](../../internal/gitx/privdir.go#L222)，特别是 223～233 行；调用者为 [Clean 的关闭路径](../../internal/gitx/clean.go#L73) 和 [scanCall](../../internal/gitx/scan.go#L35)。

`remove()` 只在进入重试循环前调用一次 `verify()`。首次 `removeAll` 返回错误后，后续最多 19 次尝试直接按原路径递归删除，没有重新检查该路径上的目录身份。

**独立复验，Windows 和 WSL Linux 均成立**：使用现有 `removeAll` 注入点，在第一次删除调用中模拟以下有界调度；仅操作本测试刚建立的私有目录。

```text
第一次 removeAll(path)：
    把原私有目录移到 path + ".moved"
    在 path 新建另一个目录，写入 replacement-content = "keep"
    返回模拟的临时删除错误
第二次 removeAll(path)：
    调用真实 os.RemoveAll(path)
```

公开 `Clean` 和 `Log` 各自均得到：删除尝试 2 次，`err=nil`／exit 0，替代目录中的 `replacement-content` 已不存在。Clean 的实际结果为 `StateClean`。原私有目录则留在移走的位置。

这不是断言某个恶意账户已经取得受保护目录的访问权；它证明在本轮明确要求防护的目录替换／失败调度下，库会删除身份不符的对象，并把清理视为成功。第一次删除失败与 50ms 等待构成真实的重试窗口，不需要突破任何 Go 切片或内存边界。

**影响**：违反“被替换时读取失败、不删除替代物”及清理失败不得报告成功的库契约。错误的成功也会使扫描调用方继续后续处理。没有据此宣称已复现错误的数据库事实或跨账户程序执行。

### 2. 根已替换时，拒绝路径仍删除替代根中的同名锁文件

位置：[newPrivateDir:167](../../internal/gitx/privdir.go#L167)、[privateDir.verify:206](../../internal/gitx/privdir.go#L206)、[remove:223](../../internal/gitx/privdir.go#L223)、[release:240](../../internal/gitx/privdir.go#L240)。

`newPrivateDir` 建立前检查 Scratch 的 `tmp` 身份，但 `privateDir` 没有保留 Scratch 根的身份关系。验证失败时 `remove()` 仍调用 `release(true)`，而 `release` 在解锁、关闭原句柄后无条件执行 `os.Remove(d.lockPath)`。这个字符串现在可能指向另一个根中的另一个文件。

**独立复验，WSL Linux／Git 2.43.0 成立**：在现有 `testHookPrivateDir`（目录建立后、fill 或 scanCall 验证前）执行：

1. 将整个 `home/tmp` 移至 `home/tmp.moved`，原目录和打开的原锁句柄留在那里。
2. 在原 `home/tmp` 路径重建根、同名私有子目录和见证文件。
3. 写入同名 `.lock`，内容为 `keep-replacement-lock`，它不是原锁句柄指向的文件。
4. 继续公开 `Clean` 或 `Log`。

两条入口均正确返回 environment／exit 6，替代子目录的见证文件保留，**但替代根中的同名 `.lock` 被删除**。原锁仍在移走的根中。目录拒绝本身没有阻止清理路径越过对象归属边界。

Windows 上尝试同一“持锁时整体重命名 tmp”的调度，在第一步即得到 `Access is denied`；该平台前置条件未成立，不能把它算成此反例在 Windows 已复现，也不能把它算作实现中的身份保护已经验证。上述误删锁的运行证据来自 Linux。

### 3. 活动读取只检查子目录身份，可漏掉其根已变成链接

同样使用 Linux 的建立后钩子，将 `home/tmp` 移至 `tmp.moved`，再令原 `tmp` 路径成为指向 `tmp.moved` 的符号链接。此时最终私有子目录仍是原 inode，`same(d.path, d.id)` 成立；但 Scratch 根已不再是建立时的那个直接目录入口。

**独立结果**：`Clean` 返回 clean／nil，`Log` 返回 nil，写入、Git 使用与删除都继续经过这个新链接。`newPrivateDir` 中对 `s.root` 的检查早已完成，后续 `verify()` 没有再次核对它。下一次新建私有目录时对根的检查，不能替代当前活动读取的检查。

此调度没有把 Git 指向恶意配置，也没有证明另一个账户能穿过已通过的 datadir 防护；它准确证明当前“tmp 被替换后保守失败”的保证尚未实现。Windows 的同类根重命名仍受前述操作系统限制，不冒称跨平台动态复现。

### 最小修复与必要回归

- 将受验证的数据目录／Scratch 根、私有目录及锁句柄的身份关系保留到使用和清理阶段；不能只验证最终子目录，或仅以同一路径字符串判断归属。
- **每次删除重试前**重新验证身份；一旦变化，停止按该路径删除并返回 environment。正常临时失败、身份始终不变的重试仍可保留。
- 锁文件删除同样需要确认父命名空间及锁文件仍对应本次持有的对象。根或锁已替换时，不删除替代物；对无法安全定位的原对象保守保留，避免以后把替代物当作自身遗留清理。
- 将上述规则统一用于 Clean、扫描防护目录、失败清理和遗留清理。可以采用有稳定目录句柄的操作，或能够证明同等保护的实现；不要求重构存储层。
- 增加三组回归：删除失败后替换目录；活动期间替换根并放入同名锁；根变成指向原树的链接。保留同一身份下重试成功、失败遗留、正常取消、并发活跃条目不被清理的正对照。
- 同步规范 §14.3 的保证。其“复核失败后删除它的锁”必须限定为已确认仍属自己的锁，不能给按名称删除替代对象提供例外。

本项不需要新的产品取舍或风险豁免。现有要求已经明确，不应延期到后续工作包。

## 四、B1 已成立的修复与验证边界

以下改进有效，不能因残余问题而抹去：

- [NewScratch](../../internal/gitx/privdir.go#L70) 首先调用 `datadir.Ensure`，再 `EnsureSub(TmpDir)`；没有打开数据库。原有 datadir 的祖先、属主、权限、链接／junction、Windows 可继承 DACL 和 Unix／macOS 权限规则被复用，没有在本轮放宽。
- 不安全祖先在创建数据目录前被拒绝；现有 tmp 对他人开放时拒绝。私有目录不再使用系统临时根，也不自动清理旧系统临时根中的内容。
- 本机 Windows 和 WSL 普通用户运行的 `TestScratchNeedsAPrivateDataDirectory`、`TestPrivateDirectoryReplaced`、`TestScratchRootReplaced`、`TestScratchRemovesOnlyItsOwnLeftovers`、`TestPrivateDirectoryLifecycle`、`TestPrivateDirectoriesArePrivate` 均通过。单独替换私有子目录／链接的已提交用例成立，整体 tmp 在分配前被替换的用例也成立。
- 锁先于目录建立并在正常读取期间持有。正常遗留清理保留活跃锁、无锁目录、非普通文件锁和其他名字，移除符合专用命名约定的失活锁／目录；没有继续按旧前缀和时间清扫系统 tmp。这里的归属依据是受保护专用命名空间、精确命名和锁，不是任意目录名称本身就证明创建者身份。
- 独立验证零值 Scratch 受控返回 exit 6，nil 路径由现有测试返回内部错误；重复使用正常 Scratch 两次都成功并清空临时区，没有 panic。

现有替换测试在建立后替换最终子目录，生命周期测试把删除始终设为失败；它们没有覆盖“第一次失败后对象改变、下一次成功删除”以及“根替换后同名锁不是原锁”的场景。因此这些测试全绿与第三节的失败不矛盾。

Linux CI 的四项新跨用户测试确实使用不同 UID 的进程：可写父目录的读取被拒绝；安全父目录的正常读取成功且攻击者不能重命名数据目录／tmp；他人拥有的祖先被拒绝；sticky 父目录允许正常读取。这证明了其明确的跨账户前置条件和控制，不证明所有同账户替换、清理重试调度均安全。

Windows 的 Everyone `FILE_DELETE_CHILD` 拒绝、同账户目录替换及 junction 用例均有效；**未做 Windows 真实双账户动态验证**。没有把子目录 DACL、父目录防替换、锁与身份检查互相当作替代证据。

`NewScratch` 会创建 tmp 并尝试清理遗留，并非只读版本探测。当前没有接入生产 CLI，普通 Open／doctor 和版本拒绝调用链未改变。后续命令接线时仍须先满足适用的版本／墓碑／只读拒绝顺序，再允许 Scratch 的目录写入与清理；不能把“NewScratch 不打开数据库”扩大成“不兼容数据库命令的目录绝对不变已经得到保证”。本轮不提前实现该接线。

## 五、B2：位图修复可以关闭

位置：[readLink／ewahSize](../../internal/gitx/index.go#L278)，测试：[index_bitmap_test.go](../../internal/gitx/index_bitmap_test.go)。

- `ewahSize` 先要求完整的 12 字节框架，再根据当前载荷长度约束 word 数，计算尺寸后读取 RLW；32 位溢出的 word 数也会拒绝。RLW 在非空位图中必须落在 words 内，空位图只允许零位置。
- `readLink` 对第一、第二位图逐次检查 `n <= len(rest)`，最后要求载荷用尽。不再依赖底层 slice capacity 或外层 hash 的字节提供“隐含空间”。
- 在旧提交的隔离副本中，独立手写外层布局的 68～71 字节 skipHash 索引，`IndexIsSparse` 和 `Clean` 共 8 个调用全部复现 `[12:8]`～`[12:11]` panic。
- 同样原始字节在目标提交的 Windows 和 Linux 两个公开入口均返回 environment／exit 6，没有 panic。
- 已提交测试通过合法外层 checksum／skipHash、两段位图的逐长度截断、异常 word 数／RLW、真实 shared index 及合法 split index 对照，确实到达 link 解析层。`TestCleanWithAShortBitmapHeader` 还检查没有执行读取索引的 Git 命令、原 `.git` 文件内容和 mtime 不变；其 helper 为每次调用准备 Scratch。清理另由本轮 Windows 独立原位图测试显式断言：四种长度的 Clean 失败后，Scratch 的 tmp 中既没有私有目录，也没有锁文件。
- Windows 整包及 Linux 定向测试中，截断、计数、4,000 次有界随机损坏和正常对照全部通过。正常索引版本／split index 等既有回归也在本机整包通过。

两道边界检查提供独立保护。仅撤回内层最小长度检查而外层拒绝仍生效，不构成测试无效的证明。作者报告的 60 秒模糊探索及变异结果只作为辅助记录，本轮未独立重复，不描述为三平台 CI 的压力测试。

## 六、B3：排除一致性可以关闭

位置：[privateRepo.read](../../internal/gitx/clean.go#L445)、[filtered](../../internal/gitx/clean.go#L543)，测试：[clean_exclude_test.go](../../internal/gitx/clean_exclude_test.go)。

超级项目的 status、ls-files、diff-index、diff-files 统一使用 `-- . ':(exclude).specmetry'`。属性候选、racy 判断和 gitlink 来自已经按此 pathspec 筛选的索引列表；没有另外使用宽泛字符串前缀。

独立在受控环境运行冻结的定义命令，确认 `.specmetry/hidden` 的过滤文件发生内容变化时定义输出为空：旧提交 Clean 返回 undecidable，新提交 Windows 和 Linux 均返回 clean。不是只用两个实现彼此一致来证明定义。

已提交用例另覆盖排除区域的内容变化、mtime、racy、暂存和删除、被删除及暂存变化的 gitlink；`.specmetry-not/`、`.specmetry-sub` 继续计入，子模块内部自己的 `.specmetry/` 也继续计入。Windows 整包、Linux 定向以及三平台 CI 均实际通过。

稀疏／损坏的门禁仍作用于整个索引，在任何 pathspec 过滤之前执行。`.specmetry/` 本身是稀疏目录条目时仍得到 undecidable，没有通过排除跳过安全门禁。未发现本轮直接改变冻结的子模块递归语义。

## 七、N1、规范同步与直接回归

[TestScanConcurrentSkip](../../internal/gitscan/scan_test.go#L815) 在 hook 内先完成内层扫描，再保存 `afterInner`，之后才恢复外层写事务。比较包含 seq、commit 字段、关联、游标内容（包括 `scanned_at`）及各表行数，能够区分内层合法写入与外层的额外变化。

本轮独立验证：

- 未修改目标副本的 `TestScanConcurrentSkip`、`TestScanConcurrentProcesses` 通过。
- 在另一临时副本的外层 concurrent 分支注入 `UPDATE git_scan_cursors SET scanned_at = scanned_at + 1`，不增加行数；测试在 `scan_test.go:843` 明确失败，指出外层改变内层提交后的游标。说明新快照能检出旧断言漏掉的字段写入。

规范 §14、architecture、MVP-EV-16／18、开发指南和实施计划同步了 Scratch、子模块四种变化、签名程序阻断、祖先缺失版本差异、worktree 元数据路径、排除和位图边界，明确标为 Review 058 之后、待本轮复审，追溯身份正确。§14.3 关于身份失败和锁删除的保证仍须按 B1-R 收紧实现与表述。

Review 058 从归档提交 `9648958` 到目标未被改写，文件 SHA-256 为 `7c1f4b4dfa98ea62e4d692313beabfd1e4e51649a6a57d33d39dc71fa89b78d8`。§68 保留原 NO-GO、固定 SHA、产品及验证边界，未自行宣布审查通过。

当前生产调用点均已调整 Scratch 参数：Clean 的子模块递归、扫描的 Tips／Classify／IsAncestor／Log、`gitscan.Options`；测试 helper 相应调整。没有改变迁移、推导、CLI／Schema、生产数据库保护、重试和错误映射。复用 Review 058 已成立的事务和迁移结论，没有据此重复宣称完成后续 CLI／E2E 验收。

## 八、实际运行与 CI 核对

### 本轮独立运行

从固定 Git 对象导出仓库外副本，测试源码变动、变异和故障注入均未进入原仓库。

| 平台／对象 | 验证 | 实际结果 |
|---|---|---|
| Windows/amd64；Git 2.56.0.windows.2；Go 1.27.1；未修改的目标副本 | `go test -tags nowebui -count=1 -timeout 25m -v ./internal/gitx ./internal/gitscan ./internal/datadir` | 全部 PASS；gitx 588.586s，gitscan 121.539s，datadir 1.709s |
| 同平台；旧 `0c9c51d` 副本 | 独立原位图双入口和排除定义对照 | 位图 8 次 panic；排除反例错误 undecidable，符合上轮缺陷 |
| 同平台；目标隔离验证副本 | 原位图、排除定义、零值／复用 Scratch | PASS |
| 同平台；目标隔离验证副本 | 删除重试中替换目录；Clean／Log | 均错误成功并删除替代内容，见 B1-R |
| 同平台；目标隔离验证副本 | 持锁时整体重命名 tmp | 前置条件被操作系统 `Access is denied` 阻止，不计为产品反例通过或安全证明 |
| WSL Linux/amd64，普通 UID 1000；Git 2.43.0；由本机 Go 1.27.1 交叉编译测试二进制 | 同一组独立 B1-R 调度及正常、位图、排除对照 | B1-R 三种调度均复现；正常对照 PASS |
| 同一 WSL 环境 | 已提交的 Scratch／私有目录生命周期、位图和排除相关定向测试 | PASS；使用 Linux 临时夹具，不是 Windows 文件系统上的权限替代测试 |
| Windows；未修改目标及独立变异副本 | 并发跳过正常／额外游标字段写入 | 正常 PASS，变异明确 FAIL |

本轮没有跑完整 CLI／全仓测试、独立 Linux Race Detector、macOS 本机测试、真实 Git 2.32 或 Windows 双账户动态测试，也没有重新进行大规模模糊探索或压力测试。WSL 二进制使用现有依赖缓存构建，没有下载或构建外部 Git 源码。

Windows 本机按设计跳过 `TestClassifyWithUnreadableStorage`、`TestWP23ReadsWithAGitThatFetches`；datadir 的一个真实 symlink 用例因权限跳过，junction 用例实际通过。跳过不等于通过。本轮包测试使用显式时间上限并实际完成，未发生超时；作者全仓 CLI 耗时及此前本地超时只保留为作者记录，不推断其根因。

### 最终 CI 与合并树

独立核对 [run 39203362680](https://github.com/XKHoshizora/specmetry/actions/runs/39203362680)：HEAD 为本轮目标、Attempt 1，completed／success，8/8 Job 全部成功。

八个作业实际 checkout 均为：

```text
dab32b4d31ec5f38e8d881f335ddb0e7b34b785b
parents:
  e0a21b46b5ad7e05146f254aa7523d6054ce2e76
  4519e7a37a744e2b2d0dc2da719314964267c0d0
tree:
  fff2648a340b44199a62bf78cbddf75b31ad9bef
```

该 tree 与固定 HEAD 完全一致，作者所报 `dab32b4` 准确。没有将 PR HEAD 元数据与实际检出提交混同。

- Go tests 作业：Ubuntu `117900850024`、Windows `117900850072`、macOS `117900850090`。Git 分别为 2.55.0、2.56.0.windows.2、2.56.0；Go 均为 1.27.2。
- Ubuntu 普通测试步骤按 Linux 分支使用 `-race`。四个 `TestCrossUserScratch*` 在普通步骤因需要 root 而 SKIP，随后在 root 专门步骤全部实际 PASS；该专门步骤另行编译测试二进制，不能称其也用了 `-race`。
- Windows／macOS 不编译 Linux 专属跨用户测试文件，其 root 步骤按设计跳过。不能把 Linux root 结果写成 Windows 双账户结果。
- 三平台的 Scratch、位图、排除、并发跳过等相关回归实际 PASS；稀疏排除门禁测试在三平台实际 PASS，没有通过跳过掩盖该断言。
- unsafe Git 专项在三平台仍 SKIP；不可读存储测试在 Windows SKIP、Unix 平台实际运行。作者真实旧 Git 2.32 的专项及 M-9 三项既有失败不属于此次 CI 的版本验证，本轮也未独立重跑。
- 保留 [run 39202572684](https://github.com/XKHoshizora/specmetry/actions/runs/39202572684)：`8dfb31e4e718f0e4f9270729e1f53eb3f6c2af56`，Attempt 1，7/8；三平台测试成功，Go lint 的两项 unparam 失败，分别位于 scratch 测试的未使用闭包结果和跨用户 repo helper 恒定参数。`4519e7a` 修正后是另一提交的首轮成功，没有改写成整个 PR 从未失败。
- Review 058 已记录的两轮早期 lint 失败继续保留，不因最终 CI 成功删除历史。

## 九、NON-BLOCKER、PRODUCT DECISION、ACCEPTED RISK

- **NON-BLOCKER**：本轮没有新增独立的非阻塞缺陷；上轮 N1 已关闭。B1 中的测试缺口作为该阻塞的修复范围处理。既有 M-9、M-14、M-15 和其他维护事项未关闭。
- **PRODUCT DECISION**：没有新增必须由负责人裁决的产品语义问题。B1-R 是既有私有空间／不误删保证的实现缺口，不通过降低保证解决。
- **ACCEPTED RISK**：仅继续引用已批准的 §62 关联生命周期限制；P1／P2／P3 和最低 Git 能力边界保持既有批准范围。没有新增临时区安全豁免。Windows 未做双账户动态验证属于验证边界，不能改称已接受风险。

## 十、最终结论与接手建议

1. **B1：不能关闭**；B2、B3、N1：可以关闭。
2. **PR #27：NO-GO，暂不合并。** 目标 CI 身份和结果成立，但不能消除本轮已复现的替代目录／锁误删及根身份缺口。
3. **WP2.3 本层交付验收证据尚不充分**。正常读取、扫描事务和迁移的既有结论保留，本轮未发现需要重新开启其全量审查的算法或数据一致性反例。
4. 最小后续任务由实施方修正 B1-R 的根／锁归属与删除重试，补上述正常和失败对照，同步 §14.3；随后仅复核这些路径及直接回归，不重做已经通过的整包审查。
5. 修复与复审完成后，仍由负责人决定合并和本层验收。本报告不宣布 WP2.3 或 M2 验收，不关闭 M-14／M-15，不批准生产重试，不授权或启动后续工作包。

归档说明：只新增本报告，使用仓库相对链接；未修改历史审查原文、实现、测试、规范、Schema 或迁移，未提交、推送或合并。独立反例和测试日志保留在仓库外隔离目录，不移入原仓库。
