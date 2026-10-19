# Codex Review 063：PR #27 完成记录 Sync 失败修复的定向独立复审

审查日期：2026-10-20（Asia/Tokyo）。审查者：Codex。本轮为只读复审，没有参与 `a2f7bd6` 的生产修复；仅在仓库外副本运行验证。完整阅读 [Review 062](codex-review-062.md)，沿用其已成立裁定，不把实施方自测或 CI 绿色代替本轮验证。

**技术结论：GO。B1-R2-S 可以关闭；结合既有有效裁定，B1 可以在已批准方案 B 的契约下整体关闭。未发现本轮直接引入的实质性 BLOCKER。**

**最终合并条件尚未满足。** 代码 CI `39276404802` 已确认 attempt 1、completed/success；最终 HEAD 的 CI `39277972806` 在开始和收尾读取时均为 attempt 1、`in_progress / conclusion=null`。虽已独立读取该最终运行的八份完整日志，不能据此宣称平台汇总成功或完整合并 GO。后续只需核实该固定 HEAD 的最终 CI 汇总；本轮不重跑、不追加提交刷新状态，不合并、不宣布 WP2.3 验收。

## 一、固定身份、提交与归档

| 对象 | 固定身份 |
|---|---|
| PR | [XKHoshizora/specmetry#27](https://github.com/XKHoshizora/specmetry/pull/27) |
| HEAD | `c4dbfb10ccf59899eb1223889fcb3913a7248bfb` |
| base | `e0a21b46b5ad7e05146f254aa7523d6054ce2e76` |
| Review 062 对象／增量起点 | `d121bfb42fdbdb922fc136265a3792a86875709b` |
| 代码修复 | `a2f7bd643fefa39155b883951a786f6cf63353e5` |

开始与收尾经 GitHub 连接器核对远程，HEAD/base 均未变化，PR Open、未合并、`auto_merge=null`。本地分支 `feat/m2-wp23-git-reader` 的 HEAD 与固定目标一致，开始工作区干净，Review 063 文件不存在。没有自行更换目标、切换工作区或改基。

读取适用根 AGENTS.md、完整 Review 062、[review-resolution §73](../review-resolution.md)、[实施记录 §9](../wp2.3-b1-implementation.md)、规范及验收增量和必要调用链；未发现范围内额外 AGENTS.md。沿用 `codex-security:verify-fix` 的原路径、控制、反证和正常对照方法，按本轮指令保存 Markdown 报告。

三次提交范围独立核对：

| 提交 | 实际范围 |
|---|---|
| `ac8f54c31df2ec5e6c877d5fa63e78648b0a308f` | 仅新增 Review 062 原样归档 |
| `a2f7bd643fefa39155b883951a786f6cf63353e5` | `privdir.go`、新增回归，以及 §73、规范、验收和实施记录，共六文件 |
| `c4dbfb10ccf59899eb1223889fcb3913a7248bfb` | 仅更新实施记录，未变生产代码或测试 |

相对 `d121bfb` 合计七文件，与提示范围一致。生产变更仅 [privdir.go](../../internal/gitx/privdir.go)。call、Clean、扫描及探测算法、Windows/Unix 平台创建删除实现、生产 SQLite、迁移、推导、CLI、Schema 和工作流均无改动。

归档文件与 `ac8f54c` 无差异，SHA-256 仍为 `bba92f73c87cb96596e4cd75c910af1e4be00d74f9f313f1b5995d7a105d1fbf`，与 Review 062 收尾值一致；受审身份仍指向 `d121bfb`，NO-GO 原结论未改写。§73 正确保留历史裁定与实施授权，没有由实施方提前宣布关闭。规范和验收同步没有增加正式验收编号或放宽方案 B。

## 二、B1-R2-S：修复成立

### 1. 三个状态得到区分

[privateDir:365](../../internal/gitx/privdir.go#L365) 增加本次对象的 `completionUnconfirmed`，没有序列化到 owner，也没有改变 v3 协议。

[finishConsumer:703](../../internal/gitx/privdir.go#L703) 的顺序为：先拒绝本次已有的完成持久化失败，完整输入预检通过后设 `completionUnconfirmed=true`，再生成 pending=false 的记录；只有 Truncate、完整 WriteAt 及 Sync 全部成功，才清除此标志。

| 当前状态 | 本次调用的行为 |
|---|---|
| Git 仍可能消费，pending=true | begin/remove 拒绝；不能清理 |
| Git 已结束，但完成写入或 Sync 失败 | completionUnconfirmed=true，begin/finish/remove 均拒绝；保留全部现存内容 |
| Git 已结束，完成写入及 Sync 成功 | completionUnconfirmed=false、pending=false，可继续正常读取或按清单清理 |

[beginConsumer:687](../../internal/gitx/privdir.go#L687) 在任何后续记录操作前检查该标志；finish 同样在预检和写入前拒绝；[remove:745](../../internal/gitx/privdir.go#L745) 在删除和重试前拒绝，仅 close 自身句柄、锁和租约。它不会因当前可读记录已经是完整 false，就忽略本次观察到的 Sync 错误。

未发现通过普通重试覆盖失败的路径；再次调用内部 begin/finish，以及再次通过 `p.read` 进入读取，都被拦截。错误继续经现有 call、公开 Clean 或探测返回 environment／exit 6，未吞为成功。探测失败不缓存、不发布用户提交回调。正常同步成功时该状态清零，没有错误保留屏障。

### 2. 精确原反例的独立复验

从固定 Git 对象分别导出旧版 `d121bfb` 和目标版本。在两个仓库外验证副本中，复用 Review 062 原始独立反例源码，仅在真正的 `d.lock.Sync()` 边界注入一次返回错误：Truncate、WriteAt、身份、清单、真实 Git Start/Wait 和原清理顺序保持不变。不是启动前失败，也未用额外条目等预检异常遮蔽目标分支。

Windows/NTFS 与 WSL Linux 均成功编译并实际运行：

| 同一反例／对照 | 旧版 | 目标版 |
|---|---|---|
| 完整 false 写入后完成 Sync 失败，随后清理 | FAIL：remove 返回 nil，目录及内容消失 | PASS：remove 返回环境错误，目录、config、owner 和全部清单内容不变 |
| 公开 Clean 的 status 已 Start/Wait 后完成 Sync 失败 | FAIL：Clean 报错，但整个临时树被删除 | PASS：Clean 报错且完整保留 |
| 启动前 Sync 失败 | 零启动并保留的正对照 PASS | 同样 PASS，没有被当作完成分支证据 |
| 正常完整输出及完成记录成功 | PASS | PASS，正常清理 |

实际完成分支的关键差异：

```text
old:   real Git starts=1; completion error; cleanup=nil; directory=ENOENT
new:   real Git starts=1; completion error; cleanup=environment; directory exists
```

原始公开 Clean 用例在 Wait 钩子之后、完成记录写入后的 Sync 位置命中；新提交的测试进一步显式检查真实 `exec.Cmd.ProcessState.Exited/Success`、starts=1/waits=1、完整规范 false 以及 `checkContents` 成功，本轮也实际运行这些断言。新旧测试都在目标失败点执行，不靠编译失败或无关 Git 错误证明检出。

**B1-R2-S 可以关闭。** 本轮没有复现其原删除行为，也未发现该修复直接引入新的失败放行或内容删除。

## 三、本次失败与后续独立 Scratch 分开判断

本次对象知道它刚收到持久化错误，必须保持失败，不能自行再次 begin/finish 或 cleanup 消除观察。后续独立进程没有前次 Sync 的返回值，仍按实际记录、原身份、清单和锁判断安全性；这两个判断没有混同。

v3 完整规范 false 的来源仍受控制：初建时没有消费者；每次实际 Start 前必须完整写入并 Sync true；之后只有实际 Start 失败或 Wait 已完成、且输入检查通过时才生成 false。完成 Sync 报错不会让已经结束的 Git 重新活跃。因此，后续进程看到可验证的完整 false，可以证明该次消费者已结束，但**不能证明前次 Sync 成功**。true、缺项／无效、截断或归属不明仍不能成为回收依据。

本轮在目标隔离副本新增两项独立行为验证：

- `TestR63CurrentCallCannotOverrideFailure`：真实 status 的完成 Sync 精确失败后，再次 finish、begin 和读取均环境错误；没有追加记录 I/O／Sync，没有额外 Git 启动；remove 仍失败保留，自己的 handles 和租约均释放。
- `TestR63LaterProcessUsesActualRecord`：先确认当前调用完整保留并释放资源，再启动真正独立的辅助进程。完整规范 false 的分支 exit 0，可信回收；true 和截断记录分支均 exit 6，原目录完整不变。没有在父进程中直接调用认领来替代进程隔离。

这两项在 Windows 和 WSL 均 PASS。对照不把后续回收写成旧 Sync 成功，也不要求未经契约支持的永久跨进程失败标记。与 §73 和本轮明确要求一致，无需新产品决定、协议升级、自动恢复命令或断电容灾机制。

新增的提交回归也覆盖 finish-write（截断后写入前失败，留下不完整记录）和 finish-truncate（截断前失败，旧记录仍 true）；两者当前保留，后续独立进程拒绝。它们是 I/O 边界注入，不是对真实硬件短写、掉电或文件系统持久性的实测。

## 四、测试有效性与直接回归

新增 `recordIO` 钩子分别位于 Truncate、WriteAt、Sync 之前，未替换其成功路径。Sync 注入点位于真实完整 WriteAt 之后。钩子非导出，生产为 nil，不新增用户参数、配置或运行时注入接口。

[review062_test.go](../../internal/gitx/review062_test.go) 的七个叶级用例为四种记录失败、公开 Clean/probe 两种路径及同次禁重试。断言包含：

- 实际 Git 进程和成功 Wait、完整清单及正确失败阶段；
- 完整临时树快照，而非仅错误码／存在性；
- close 后本次资源关闭，租约可取得；
- `ioAfterFailure=0`，禁止错误收尾悄悄重写／重试；
- 探测失败不缓存、Log 不发布用户提交回调；
- 后续真正子进程的完整 false 回收或不确定记录保留。

本轮独立复验原例并额外构造同次复用与后续记录分支；没有只依赖提交测试与实现使用同一个状态标志来认定正确。

正常完成、实际 Start 失败、取消前未 Start、运行中取消等待、可信遗留回收、ownerGone 后目录重试，以及 owner 预检和原真实 Git 崩溃六项，均按直接回归运行通过。owner 预检、原消费者生存期、空覆盖和 Windows 子机制沿用既有结论，不重新展开全包审查。

本轮未变 call／scan_probe 的传播和门禁；取消仍先等待 Git，完成记录成功才能正常收尾。Windows 原子创建／同句柄删除直接测试通过，未外推为所有文件系统或双账户动态证明。方案 B 的 Unix 创建／最终删除窗口及 Clean false-clean 限定不变；Log 完整性未获豁免。B2、B3、已关闭 N1 不重开。

## 五、CI：技术证据与平台合并门禁分开

### 1. 代码 CI

[run 39276404802](https://github.com/XKHoshizora/specmetry/actions/runs/39276404802) 元数据独立读取得到：HEAD `a2f7bd643fefa39155b883951a786f6cf63353e5`、attempt 1、completed/success。八份预期 job 的完整日志均独立读取，checkout 后的 `git log -1 --format=%H` 均为：

```text
08a59cc50d584f3589956b3a9cf2d3742e7b0f5e
```

Git Data 对象父提交依次为固定 base、`a2f7bd6`，树 `8a1d4d964378e7130a59d4ac73265ab653040f71`，与代码 HEAD 相同。代码 CI 汇总成功成立，八份日志覆盖工作流的全部八个 job；但 job 列表接口本轮超时，未取得独立的逐 job conclusion 字段，不把作者表格冒称该接口的读取结果。

### 2. 最终 HEAD CI

[run 39277972806](https://github.com/XKHoshizora/specmetry/actions/runs/39277972806) 开始及收尾元数据均为 HEAD `c4dbfb10ccf59899eb1223889fcb3913a7248bfb`、attempt 1、**in_progress / conclusion=null**。

GitHub 连接器的 job 列表接口及直接 jobs 子资源读取出现 ReadTimeout；经权限审批用 gh 对相同只读 REST 端点核对，也收到 HTTP 502。没有改用重跑、空提交或修改 PR 来刷新状态，没有自行判定异常原因。

另从 PR 提供的 job 定位独立读取八份完整日志，每份实际 checkout 均为 `1c008da36f782c36c4de06d1ec008b175f3f8018`。Git Data 父提交为固定 base、最终 HEAD，树 `e9af91501cb611785bc87f051413108582bc6e64`，与最终 HEAD 相同。

这些日志可以证明相应检查实际执行和检出内容，但**不是平台已完成汇总的证明**。最终 HEAD 不记作 8/8 成功，也不作逐 job API 状态成功的断言；合并门禁仍待平台明确 completed/success。

| Job | 代码 run 的日志 ID | 最终 HEAD run 的日志 ID |
|---|---|---|
| Go tests Ubuntu | `118141716289` | `118146862204` |
| Go tests macOS | `118141716102` | `118146862169` |
| Go tests Windows | `118141716562` | `118146861974` |
| Go lint | `118141716222` | `118146862372` |
| Web | `118141716054` | `118146862360` |
| Build with dashboard Ubuntu | `118141961548` | `118147267622` |
| Build with dashboard macOS | `118141962022` | `118147267660` |
| Build with dashboard Windows | `118141961520` | `118147267528` |

两组的三份 Go 日志均实际记录七项新故障／重试用例 PASS，完成分支 starts=1/waits=1、启动前对照 starts=0/waits=0。macOS 为 Go 1.27.2 darwin/arm64、Git 2.56.0；新增用例、owner/FIFO、真实消费者六项、取消／回收及重试均实跑。本机未运行 macOS，不以交叉编译替代该 CI 运行证据。

Ubuntu 为 Go 1.27.2、Git 2.55.0；普通 Test 的脚本选择 `-race`，其中八项 CrossUser 因非 root SKIP；随后单独编译、sudo 执行的八项跨账户测试实际 PASS，该步骤没有 `-race`。Windows 为 Go 1.27.2、Git 2.56.0.windows.2；平台原子机制直接测试实际运行。macOS/Windows 不运行 Linux root 步骤，也不算双账户动态通过。

三平台真实 unsafe Git 测试未配置相应版本而 SKIP；M-14 两项诊断未启用而 SKIP；Windows 的 POSIX 不可读权限用例跳过、FIFO 不参与 Windows 构建。全部不计为运行通过。M-14／M-15 等维护事项不因日志或 CI 汇总变化而关闭。

保留 Review 062 及实施记录中的历史失败、重跑和验证边界；没有把本轮代码 CI 首轮成功写成整个 PR 从未失败。最终 HEAD 的汇总异常也不作为代码缺陷。

## 六、本轮实际运行及证据边界

原仓库仅只读。测试均在固定 Git 对象导出的仓库外副本、新建 Git 仓库及临时数据目录执行，没有修改真实项目数据。

| 本轮独立运行 | 平台与结果 |
|---|---|
| 目标未改动副本：新增三项顶层测试与 owner、完成／取消／启动失败、真实消费者、ownerGone 重试、探测取消、Windows 原子机制直接组 | Windows/amd64、Go 1.27.1、Git 2.56.0.windows.2、NTFS；10 项顶层测试 PASS，62.308s |
| 同组加 Unix FIFO、无 Windows 专属项 | WSL Linux 普通 UID 1000、Git 2.43.0；10 项顶层测试 PASS |
| Review 062 原独立反例：旧 `d121bfb` | Windows、WSL 实际 FAIL：完成 Sync 保留和公开 Clean 保留两项失败；启动前失败、正常完成对照通过 |
| 同一原例：目标版 | Windows 三项顶层测试 PASS，5.989s；WSL 同样 PASS |
| 本轮独立同次禁复用、后续真实进程三种记录 | Windows 两项顶层测试 PASS，24.214s；WSL 同样 PASS，完整 false exit 0、true／截断 exit 6 |

Linux 二进制由 Windows Go 交叉编译，然后在 WSL 实际执行，不称为 Linux 原生构建。本机没有新增 -race、root 跨账户或 macOS 运行；对应证据仅来自上述独立读取的 CI 日志。

故障注入只是指定 I/O 边界返回错误，不是物理磁盘损坏／掉电实测。本轮不新增该类门槛，没有验证断电后的文件系统恢复状态、真实 unsafe 旧 Git、Windows 双账户／ReFS、穷尽压力、完整 CLI/E2E 或其他未变模块。未验证内容不被描述为已修复或已接受风险。

原例副本保留生产清理和状态代码，仅在实际 Sync 位置加入原审查故障钩子。目标未改副本运行提交测试；新增独立对照仅在验证副本中，未移入原仓库。新增本机测试没有编译失败被计为检出的情况；旧版负向测试均成功编译并运行到目标断言。

仓库外材料位置为代码格式路径：

`C:/Users/hoshi/.codex/visualizations/2026/10/12/01a13137-991b-7b42-8ea2-3c24008343a0/review063`

目标导出 tar SHA-256：`343af1fa85479a528fea57a59a8050fe2dbc45a56dd464ee354c6aa066582e5a`；旧对象 tar 为 `016361d0f7ecd2dced821685677889d506b8782a4f6d3bda270a9e66f2dc70c4`，与 Review 062 的对象一致。保留验证源码、实际日志与 CI 身份／摘要回执；这些材料不新增至原仓库。报告使用仓库相对链接，不改历史审查。

## 七、分类、最终裁定与接手条件

**BLOCKER：无新的代码阻塞；B1-R2-S 关闭。**

**NON-BLOCKER：本轮无新增代码问题。** 最终 HEAD CI 汇总待完成属于外部合并门禁，不冒充代码缺陷，也不因技术 GO 自动豁免。

| 问题 | 本轮结论 |
|---|---|
| B1-R2-S 能否关闭？ | **可以**，原精确反例修复，同次禁重试和资源／内容保留成立 |
| B1 能否在方案 B 下整体关闭？ | **可以**，保留既有 owner、生存期、空覆盖及 Windows 裁定，唯一残余已闭合 |
| 是否超出授权或改变协议？ | 未发现；本地失败状态与后续认领分开符合本轮要求，不新增永久屏障或风险接受 |
| PR 技术结论 | **GO** |
| 最终 CI／合并条件 | **尚未满足**；最终 run 仍 in_progress/null，不能完整合并 GO |
| 是否足以支持 WP2.3 本层验收？ | 本轮技术修复证据已闭合，可作为本层验收依据；最终 CI、合并核验及负责人决定仍需完成，未宣布验收 |

最小下一步仅为只读核实 `39277972806` 在固定 HEAD 上的最终平台状态及身份；若成功，结合本技术 GO 由负责人决定合并并核对实际合并结果。若状态变化、失败或 HEAD/base 改变，按具体差异处理，不把本结论延伸到新代码。无需再开同范围全面审查，不自动修复、不重跑 CI、不追加空提交。

方案 B 批准范围和所有既有未解决维护事项保持不变，不关闭 M-14／M-15，不启动后续工作包。本轮仅新增本未提交报告；不修改其他原仓库文件，不提交、推送、合并或改基，不代替负责人宣布验收。
