# Review 158：PR #59 的 Review 157 定向复审

## 1. 裁定

**固定对象的准备变更技术 GO，建议作为首次正式取证的准备成果合并，合并仍由负责人决定。** B-157-1 在本轮分类文字范围关闭；N-157-1、N-157-2 完成。没有发现新增 BLOCKER，未发现工具实现、既有分类优先级、预算或授权范围的直接回退。

当前常规 CI `46079832844` 已结束：`pull_request`、attempt 1、正式 success、8/8。本轮只查询一次该运行状态，再读取八份已有作业日志与合并快照对象；没有等待、轮询或重跑。当前没有未完成或失败的常规 CI 条件。

有三项非阻塞收尾意见，详见第 6 节：四份本机输出没有进入 Git 提交；F-2 对原名检查顺序的一处概括；“所有读取为空即失败”的说明应排除工作区状态。它们不使实际修复失效，也不要求新增实验。特别是第一项应在后续归档收尾中补齐，不能把当前提交说成已包含完整输出。

**准备 GO 不等于正式运行已可执行。** N2、用于正式 R 的 E-10(ii)、授权 C/T、运行分支与一次正式执行仍须单独决定。没有采集 R，没有实际正式取证或预算结果；不据此宣布 G 项满足、批准精确例外、授权第三批实施或验收 WP2.5／M2。

## 2. 固定身份、增量与记录

审查日期：2026-11-28。开始及结束核对本地 Git、远程分支、远程 master、PR head/base 和工作区。

| 对象 | 实际核验值 |
|---|---|
| PR／分支 | #59，OPEN；`prep/wp2.5-c9-formal-run` |
| HEAD／远程分支／PR head | `b7ed0362af95228e6f372f0f9bbcdf5d4fa6fafd` |
| HEAD tree | `20a1c8b04f7d71c66969afc19ff66d7972f257d6` |
| base／远程 master／PR base | `c34baf86bac1524bcfdabf2b6b67f8f2169e79b5` |
| 上轮对象 | `dcc01bd247a3c04a16103a5bbe5042c5c4a7a34f` |
| 工具模块子树 | `755ffc0c2465959fa5528d07b5c079154e8e4feb` |
| c9-evidence.yml | `9ef8b58766200b7ca35aa6757a5f226d1e562b0b` |
| c9-tool-test.yml | `4387acdc683a2014c235f3991bf854a30d6c7c2c` |
| ci.yml | `8ca60930a5407151358046929738c141d8d52f84` |

依据包括当前 AGENTS、[Review 157](codex-review-157.md) 原文、[处理记录](../review-resolution.md) §208／§209、[实施计划](../wp2.5-implementation-plan.md) §12.6／§12.12 的直接接点、[限定取证计划](../wp2.5-c9-limited-evidence-plan.md) §12、[第二阶段记录](../wp2.5-c9-evidence-phase2.md) 的更正关联，以及[本轮验证材料](../evidence/wp2.5-c9-prep-r157/README.md)。没有重新全面审查未变的第二阶段实现。

上轮 HEAD→本轮 HEAD 的 Git 增量为 **15 个文件，838 行增加、44 行删除**，顺序为：

1. `ea61465`：只归档 Review 157。
2. `942f316`：只追加 §208，先于修订。
3. `43b9f99`：只修正式 workflow 的目标谓词与读取错误消费。
4. `b7ed036`：F-2、验证脚本及摘要、§209 和状态同步。

Review 157 当前归档为 **20164 字节**，SHA-256 `aaadee6bc50f0e2934b54a14d9bc30cd2f58f773b9a250fe3caae1d44016357a`，与上轮交付及 §208 登记相符，归档提交只含该文件。按 Git 原始字节比较，旧 `review-resolution.md` 的 **1515725 字节**是当前 **1535332 字节**的完整相同前缀；§207 及旧历史没有回改。§208 明确撤回 §207 的错误推论，§209 仍写“已修订／已精化，待独立复审”，没有提前宣布关闭。

工具模块、根模块依赖、常规 CI、专用测试 workflow、冻结正文及验收项无差异。其余文档只作本轮允许的状态、更正或失败处置同步。`git diff --check dcc01bd HEAD` 无提示。

开始的 Git 工作区干净，报告不存在。本机另有早已生成、被忽略的四份输出，属于用户指定的既有材料，详见第 6 节；本轮没有生成、修改或清理它们。

## 3. B-157-1：分类前提关闭，保留一处非阻塞说明精化

固定 HEAD 的[实施计划](../wp2.5-implementation-plan.md)第 1494～1507 行已将判据改成实际 `lookExtensions` 结果与 `cmd.Err`，而非原名存在性。独立只读核对固定 Go 1.27.3 提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`：

| 分支 | 固定源码及本稿结论 |
|---|---|
| Windows 绝对路径的预解析 | `src/os/exec/exec.go:468–481`：成功缓存 in/out；失败才将实际错误放入 `cmd.Err`。`681–687` 在 Path 未改时使用缓存结果 |
| 已知扩展名 | `src/os/exec/lp_windows.go:84–92`：与 PATHEXT 项不区分大小写匹配便直接接受，不 stat。已核对的、带 `.exe` 且该扩展名在集合内的不存在路径，到 Start 才失败，保留本行 C 类输入 |
| 未直接匹配扩展名，候选成功 | `lp_windows.go:37–57、132–139`：追加扩展名可以找到后一个文件。本稿已明确 `probe→probe.exe`、`probe.odd→probe.odd.exe`，不把它们判成 A② |
| 所有适用查找失败 | 对这里限定的绝对路径，`lookPathExts` 包装 `*exec.Error`；实际错误被 Command 写入 `cmd.Err` 后，才按既有决策树处理 A② |
| 非空扩展名集合，最终未找到 | `37–57`：`hasExt(file)` 为真时内层 `fs.ErrNotExist`，否则 `ErrNotFound`；不把这些合成错误说成每次 stat 的原始错误 |
| PATHEXT 非空但全是空项，扩展名集合为空 | `112–129、38–39`：直接 `chkStat`。不存在时保留 `os.Stat` 错误；非目录且 stat 成功可以查找成功，不再无条件承诺 `ErrNotFound`。目录等其他错误仍按实际错误保存 |

源码链接：[lp_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec/lp_windows.go)、[exec.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec/exec.go)。本轮没有执行 Go 验证这些分支。

上述 A② 仍从属于 §12.6 有序决策树，不能覆盖 pre-started／已有 ProcessState 的优先处理，也不把调用过 Start 但结果未知改为 A。新增文字明确引用既有规则，既有表第 1477～1479 行的重叠反例未改。Windows 本行继续只用已核对的 `.exe` 夹具；Linux／darwin 不经过 Windows 扩展名解析的原结论保持。

历史定位也已闭合：§208 第 13441～13445 行明确更正 §207 和第二阶段记录 §9.2 第 2 项、§9.5 F-2；第二阶段记录文首直接点名这两处并指向修订后的 §12.12。§10.6 原有“预查找先失败时”已经带条件，不必改写历史。读者可以得到唯一的当前判据。

因此 **B-157-1 关闭**。第 1497 行“它先检查原名”的流程概括尚可更精确，但后续分类已以实际结果为准，不恢复原阻塞；见 N-158-2。

## 4. N-157-1：完整路径谓词与失败传播完成

[正式 workflow](../../.github/workflows/c9-evidence.yml)第 118～140 行的实际顺序是：保存 `@($i.Target)` → 恰好一个目标 → `IsNullOrEmpty` → `IsPathFullyQualified` → `Join-Path` → 两份 `go.exe` 哈希 → 相等后才写 `GITHUB_PATH`。没有 `GetFullPath` 或先按当前目录补全再校验的接点。

null／空串、多个目标、盘符相对路径 `C:relative`、根相对路径 `\relative`、普通相对路径均不能取得该步骤的目标资格。数量、哈希比较和非 junction 不改 PATH 的分支保持。PowerShell cmdlet 的真实读取／哈希／写入错误在 Actions 的 stop 规则下使步骤失败；显式 throw 同样失败。Collect 不含 `always()`／continue-on-error，失败后不执行；上传步骤的 always／warn 只保留现有文件，不使缺证有效。

带空格路径作为参数传给 LiteralPath／Join-Path／Add-Content，没有重组为待执行命令；GITHUB_PATH 的用途仍是后续步骤前置 PATH。实际工具的 Go1.27.3、GOROOT、布局及哈希守卫未改。两份 SHA-256 相同只说明本次分别读到的字节相同，不认证安装来源、整个 GOROOT、同一文件对象或未来读取；新注释准确保留了这一边界。

本机证据分三类：

- `real-junction` 与 `no-junction` 使用真实临时目录／junction；日志中分别写入目标 bin、保持 PATH 不变。
- 其余目标字符串由 `Get-Item` 函数替身给出。旧版本在可解析的 `C:relative` 替身输入下接受并写入相对路径，新版本在谓词处拒绝。这证明该脚本输入的区别，**不证明真实 runner junction 会返回这种字符串**。
- 新正式步骤仍未在 runner 执行。盘符完整路径与 `\\?\` 设备路径的现有用例，不扩张为已经动态验证普通 UNC／任意安装布局。

因此 **N-157-1 完成**。没有以本机替身证据扩大 F-1 的适用范围。

## 5. N-157-2：非零、空值与工作区状态已分别消费

### 5.1 实际 shell 路径

workflow 第 53～77、80～90、150～156 行已将相关读取从 echo 的命令替换中移出。`value=$(command) || fail ...` 的状态来自赋值中的命令替换；失败显式执行 `exit 1`，不再靠 echo 或对 `set -e` 的错误理解处理。成功之后另用 `[ -n "$value" ] || fail` 检查必须非空的身份值。

- 身份：HEAD 成功且非空，并等于 GITHUB_SHA；tree、子树、workflow blob、go 路径分别成功且非空之后才打印这些取得的值。
- LE-4：下载命令保持独立执行；go.mod／go.sum blob 各自检查成功及非空，随后打印；工作区检查再独立消费。后一步失败不会被已打印的 blob 恢复。
- Collect：重新取得 PATH 上 go，失败或空值先退出，再打印和执行未改的 go version／go test。
- 独立的 `go version`、`go mod download`、最终哈希命令不再被 echo 包裹，真实非零仍进入步骤失败。哈希工具的选择条件只检测工具是否存在，没有吞掉所选哈希命令的失败。

`GITHUB_REF`／`GITHUB_WORKFLOW_REF`／`GITHUB_WORKFLOW_SHA` 是既有 Actions 环境关联记录，不是本轮新加的读取命令；`set -u` 会拒绝未设置的变量，但没有新增格式／非空认证。本轮不将这些打印或 hash 记录提升为独立的运行授权证明，FR-4 的外部固定身份核验仍需要执行。

### 5.2 新增 git status 接点

第 58～59、89～90 行遵守下表，没有把空状态误拒绝：

| 查询结果 | 后续 |
|---|---|
| 非零，包括失败但无 stdout | `|| fail`，步骤失败，不作为干净 |
| 零且输出为空 | `[ -z ]` 成立，按既有干净工作区规则继续 |
| 零且输出非空 | `[ -z ]` 失败，明确拒绝工作区变动 |

这与身份值必须非空是两种不同规则。现有本机记录包含两步 status 查询失败、身份 dirty、下载后 dirty 及成功空状态；没有通过放宽干净条件来关闭问题。

**N-157-2 完成**。说明中的“每个读取失败或为空即失败”需要补上状态例外（N-158-3），实际代码和 §208／§209 已给出正确例外，不是新的健康路径拒绝。

### 5.3 未改变的执行设置

逐项核对完整 workflow 和 diff：push 分支仍仅 `evidence/c9-run-1`；无 PR、手动或定时触发；contents:read；concurrency 不取消旧作业；三 runner、fail-fast:false、20 分钟外层期限；GOTOOLCHAIN=local、GOFLAGS 空、setup-go1.27.3/cache:false；根模块 `go mod download` 不变；Collect 命令、记录目录、artifact 的 always／90 天／warn 规则不变。

缓存准备没有增加命令或扩张权限；没有改工具内部 120 秒预算。合并 prep PR 本身不会匹配正式 workflow 的分支，但以后建立／推送该运行分支会触发，仍需明确授权。

## 6. 非阻塞意见与材料的实际可核验范围

### N-158-1：四份输出只在本机，尚未随固定提交归档

**位置**：[材料 README](../evidence/wp2.5-c9-prep-r157/README.md)第 15～25 行、[SHA256SUMS](../evidence/wp2.5-c9-prep-r157/SHA256SUMS)第 3～6 行；处理记录 §208 第 13475 行及 §209 第 13514 行。

README 和登记称输出随材料入库，但 `git ls-tree -r HEAD docs/evidence/wp2.5-c9-prep-r157` 只列 README、SHA256SUMS 和四份脚本。以下四份文件虽存在于本机，却被 `.gitignore:6` 的 `*.out` 忽略，不在固定 HEAD 中：

| 本机文件（outputs/ 下） | 字节数 | 独立核对的 SHA-256 |
|---|---:|---|
| bash-cases.out | 11271 | `89ed5f6f11a427d7dabb23795e6e218de3c5093db86962e93c1af84f7c3b84a2` |
| pwsh-cases.out | 14582 | `4fcca69126e6e8852c3da331b459a6acc790ac87dfaac946645b35322104540b` |
| syntax.out | 520 | `006092876d02861a1a58695deeaae78b0339a55984c05e3f73d056dc42f0278c` |
| yaml-check.out | 6333 | `bd0f5840dd92a4094aeea300e7b52686d66a42bc2f96b5ec6cbdb6e4d6e9aff0` |

**影响**：只取得固定提交的其他审查者无法读取 README 所称的四份回执，清单中的哈希不能替代缺失内容。本轮能读取本机副本且四项都与提交中的清单相符，所以可以按“本机可取得、由清单关联的既有输出”采证，不能写成固定提交已包含日志。

**最小处理**：经授权的收尾中仅补入这四份现有脱敏副本，或改用明确的持久归档并准确记录关联；不用重跑验证，也不需扩大忽略规则或修改正式 workflow。建议合并收尾时补齐。本轮三项关闭主要有独立源码依据，现有输出也确实可读，故此为非阻塞归档缺陷，不把它升级成执行协议阻塞。

### N-158-2：原名的 stat 有条件

**位置**：实施计划第 1497 行“它先检查原名，再依次检查……”。固定 `findExecutable` 在扩展名集合非空时，只有 `hasExt(file)` 才先 stat 原名；无扩展名时直接查追加候选。建议改成“非空集合且原名带扩展名时先检查原名，之后尝试追加候选；空集合单独走 chkStat”。

例如原名 `probe` 存在但无扩展名、PATHEXT=.EXE，而 `probe.exe` 不存在，不能按这句流程概括推导原名会被接受。本文第 1502～1505 行已经以实际错误／成功为最终判据，分类、保留责任和已知 .exe 夹具不因此改变；属于不改变判据的措辞精化，不重开 B-157-1。

### N-158-3：空输出说明应点明 status 的例外

**位置**：工具记录第 140 行“每个读取……失败或为空即步骤失败”；workflow 第 49～51 行的概括。限定计划第 875 行列的是身份、路径和 blob，可以明确其不包括工作区状态。

建议统一写成“查询非零即失败；身份／路径／blob 须非空；status 成功且为空才是干净”。§208／§209 已明确这一区别，真实代码也正确，故不会实际拒绝健康空状态。LE-4 的 blob 打印早于最后 status 检查，或后续 go version／哈希失败时，日志仍可能有部分输出；不宜把注释理解为整个步骤的日志原子提交。步骤失败及 FR-8 停止规则始终有效。

### 6.1 夹具、脱敏与证据层次

完整读取了驱动、YAML 提取器、bash／PowerShell 夹具及四份本机输出，没有执行它们。九个清单项（README、四脚本、四输出）的本机 SHA-256 均匹配 SHA256SUMS。

- YAML 输出记录被测代码点 `43b9f99999d29edd4d7c5f1cb6e4387a77144e73`，workflow blob 为 `9ef8b58…`，等于最终 HEAD 对象。提取器按步骤名关联、取 `run` 原文，不另写替代业务逻辑。源码与日志中的文件／步骤哈希形成关联；这不是不可变执行环境认证。本轮独立比较实际 workflow diff，而非只采用提取器打印的 equal。
- bash 共 15 例×两版本。新版本 3 个健康例退出 0、12 个负例退出 1；旧版本的 7 个相应故障例仍退出 0，与 N-157-2 的静态路径一致。`git` 包装器只拦截指定参数，其余交给真实 Git；`go` 是打印固定文字的脚本，不是 Go 工具链。no-gomod 的含义是在成功的假下载之后检验 blob 读取，不能说真实 go mod download 在缺 go.mod 时也会成功。
- PowerShell 共 13 例×两版本：新版本 4 个成功、9 个失败，失败例 GITHUB_PATH 均为空；分别读取了错误位置与退出状态，不用总驱动退出 0 当作所有断言成立。驱动主要记录结果，没有统一自动断言全部预期。
- `run-all.sh` 的 scrub 明确将临时目录／用户目录的 Windows、斜杠和 MSYS 路径形式替换为 `<W>`／`<HOME>`；PowerShell Show 也替换工作目录，并去除颜色、统一输出编码／语言。**这些是脱敏和展示规范化副本，不是未经修改的原始输出字节。** README 与源码披露了范围，但其“原始输出”的简称必须按此理解；SHA256SUMS 只证明实际副本的内容。
- 替换保留 `C:relative`、`\relative`、`relative`、目标数量、后缀、哈希、错误位置和退出状态，因此没有抹去本轮负例所依赖的区别。成功完整路径被缩成 `<W>` 后，不能仅由输出认证原路径的完整字符串；需连同夹具实际建目录及构造代码判断。没有取得未脱敏原始字节的历史认证。
- MSYS2 转换关闭是本机夹具条件。只读核对第二阶段 `ci-43906849891/job-windows.log:218` 确实打印预期 workflow blob，支持那一次该查询取得正确对象；不证明未来所有 runner／全部参数都不转换。本轮正式 workflow 没有新增 MSYS_NO_PATHCONV 设置，也不把夹具环境等同于 runner。
- 语法检查输出逐项 exit 0，支持作者在该本机环境做过相应检查的记录；不是审查者重新运行解析器的结果。新 PATH 步骤及身份读取还没有 runner 动态证据，真实 Go 没有在这些夹具中执行。

## 7. 当前常规 CI 的独立核验

[run 46079832844](https://github.com/XKHoshizora/specmetry/actions/runs/46079832844)：一次状态读取为 completed／success，attempt 1，event 为 pull_request，head 为固定 `b7ed036…`。随后取得八份现有日志，全部实际检出：

`83a659acf09b46c9bf640032919509d50b5a2de9`

Git commit 对象的父提交依次为 `c34baf86bac1524bcfdabf2b6b67f8f2169e79b5`、`b7ed0362af95228e6f372f0f9bbcdf5d4fa6fafd`；tree 为 `20a1c8b04f7d71c66969afc19ff66d7972f257d6`，与受审 HEAD 相同。不只使用 run.head_sha。

| 作业 ID | 作业 | 实际证据 |
|---|---|---|
| 140776009287 | Windows Go tests | success；27 个 ok 包，10 条顶层 SKIP，无 FAIL |
| 140776009548 | macOS Go tests | success；27 个 ok 包，7 条顶层 SKIP，无 FAIL |
| 140776009597 | Ubuntu Go tests | success；27 个 ok 包，15 条顶层 SKIP；随后独立 root CrossUser 8 项 PASS |
| 140776009576 | Go lint | success；0 issues |
| 140776009522 | Web | success；没有安装／显示 Go 版本 |
| 140776131603 | Ubuntu dashboard build | success |
| 140776131615 | Windows dashboard build | success |
| 140776131623 | macOS dashboard build | success |

七个显示 Go 版本的作业都是 **Go1.27.4**。Ubuntu 主测试的 race 与之后独立 root CrossUser 分开采证；主步骤中被跳过的 CrossUser 不等于从未执行。日志未见测试 FAIL 或步骤错误。Ubuntu CLI 555.878 秒只是本次观测，不推导上界、根因或维护事项已解决。

常规根模块 CI 没有运行独立工具模块，也没有执行本轮正式 workflow。它不是固定 Go1.27.3 方法、PATH 接线或正式 R 预算的证据。上轮成功、此前失败及单次重跑按 Review 157 已取得的范围保留，本轮没有重新全量查询历史运行。

## 8. 合并建议、运行前决定与结束状态

三项定向问题已在上述范围收口；可以交负责人决定合并准备成果。N-158-1 的归档补齐与 N-158-2／3 的文字精化可在明确授权的收尾中处理，不需要新实验。若任何收尾产生新提交，应重新绑定该对象，不能直接把本报告扩成新 HEAD 的核验。

正式执行仍须分别写定／接受：

1. N2 的模块准备方案及声明边界；静态依赖对应关系不等于 runner 已执行验证。
2. E-10(ii) 对正式 R 的限定前提与清理未知代价；第二阶段测试接受不自动延伸，也不接受 T-2／L-b1b 或其他依赖运行风险。
3. 授权的实际 C、T=tree(C)、随 C 固定的工具／workflow 身份；本次 PR 合并测试快照不是正式 C。
4. 三 runner／六分析配置、一次 `evidence/c9-run-1` 推送和执行人；它不是六种原生运行。保持 120 秒共享预算及失败即停，不隐含重跑、延长或放宽守卫。
5. 结论 1～4 的后续处理、原 ZIP 与成员哈希／manifest 的取回核验。退出 0 不能直接批准清单；结论 3 不能自动扩充例外；日志／artifact 不完整须保留并停止。

结束时本地、远程、PR 身份与本报告第 2 节一致。受版本控制文件无改动；唯一新增未提交文件为本报告，原有被忽略的验证输出未改。本轮只读文档、源码、Git、哈希与已有日志，没有运行 Go、测试、脚本夹具、helper、伪工具或新实验，没有修改 PR／CI，没有建立运行分支、采集 R、提交、推送或合并。

完成后暂停。技术 GO、合并决定、提案／风险接受、运行授权、G 项、精确例外、生产实施与 WP2.5／M2 验收保持独立。
