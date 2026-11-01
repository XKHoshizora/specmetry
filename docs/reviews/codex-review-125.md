# Codex Review 125 — PR #48 / B-123-2-R 定向复审

**结论：B-123-2-R 可关闭，B-123-2 在受审第二批范围内整体关闭；B-123-1、N-123-1 保持 Review 124 的关闭结论。未发现新增 BLOCKER 或直接回退。固定对象技术 GO，建议负责人决定合并该固定对象。**

本轮结论来自错误链的静态核验、审查者的定向运行和正式 CI，不能由 CI 绿色单独推出。两项新回归证明多个关闭接缝错误不再丢失；同次读取与关闭都失败的保留，目前由完整对象传递的源码证明，没有动态复现该组合，也没有独立执行作者变异。本报告不产生合并、第三批开工或 WP2.5／M2 验收授权。

## 1. 固定身份、范围与归档

开始核验：2026-11-01 13:06:32 +09:00。读取当前 AGENTS.md、完整 Review 124、处理记录 §160、生产计划 §11.13，检查上轮 HEAD → HEAD 全部差异及相关直接调用链。

| 项目 | 独立核验值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #48](https://github.com/XKHoshizora/specmetry/pull/48)，open、未合并 |
| 本地／远程分支 | `feat/wp2.5-batch2` |
| HEAD | `eafaf5c39592fcf121b87ffea650abadd2f07e4a` |
| HEAD 树 | `6878f16770dc21c9c7aae4c5660d726aafdef6a0` |
| base／GitHub master | `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4` |
| 上轮受审 HEAD | `b6542be72315260d5ad04049627fcf773b7aaa3a` |
| 最终 CI 合并快照 | `ef065da762a59eab8100c5c523f6e49178a659de` |
| 快照父提交，依次 | 上述 base、上述 HEAD |
| 快照树 | `6878f16770dc21c9c7aae4c5660d726aafdef6a0`，与 HEAD 相同 |

开始时工作区和暂存区干净。与前两轮相同，本地旧 master 引用是 `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`，不是本次 base；未更新、切换或用于取差异。受审 HEAD 与固定 base 的 merge-base 为 `47f5680`。远程 PR、两个 ref 与快照父提交／树分别通过 GitHub API 核验。

本轮增量为 **7 文件、450 行增加、15 行删除**：Review 124 归档、AGENTS／实施计划状态、处理记录、生产计划登记，以及 `interpreter.go` 和 Windows 回归。生产变化仅在 finder 的错误累积及返回，不是重写整个第二批。

Review 124 单独归档于 `b90a14d163b821d2f3cab76d3411ab9eab0805a4`，该提交只有报告文件；当前文件与归档 blob、上一轮交付相同，21933 字节，SHA-256：

`c2d7b2457e6439c1e21e77cb501bb673e03211661dd3ed0b5d4f6d1cb2cbb17c`

随后 `d901ca657a14c9b2716c3604ab78e5f39ecae6c9` 先登记 §160／计划 §11.13，`eafaf5c` 才实施修复和回归，符合先登记、后代码的授权顺序。旧处理记录是当前文件的完整字节前缀，只追加 §160；历史报告未改写。

Prepare、Batch.Close、errors.go／ReleaseError、selection、Windows／Unix finalPath 实现、两处局部 lint 豁免，以及 Schema、共享 apperr、gitx／SQLite、迁移、derive／toderive／dbdump、factgen/legality、冻结规范、原型、依赖和 workflow 均未变化。没有新增禁用条件或第四批任务管理。状态同步准确保留“待独立复审、不合并、后续批次未授权”。

## 2. B-123-2-R：完整错误累积与终判

### 2.1 每次失败保存完整对象，按发生顺序返回

核心位置：`internal/checkrun/interpreter.go:46`、`:66`～`:72`、`:157`～`:163`。

原单个 `released error` 改为 `released []error`。final 每次调用 finalPath 后，用 errors.As 判断该返回是否包含 ReleaseError；满足时执行：

```go
f.released = append(f.released, err)
```

这里保存的是 **finalPath 的整个返回错误 err**，不是被 errors.As 取出的 rel。没有“仅首次才保存”的条件，也没有覆盖已有项或修改旧对象。newFinder、onPath 和 gitForWindows 的调用在本次 finder 中按顺序执行，追加顺序就是实际发生顺序。

releaseFailure 使用新建的切片，先放受控的 environment 错误，再追加保存列表的全部项，交给 errors.Join；没有仅拼首项、清空列表或在生成错误时改变原记录。新返回值持有各项原始错误，errors.Is／As 可以访问它们。已返回的链也不会因下一次查询新建外层错误而失效。顺序结论来自这些明确的 append／Join 步骤；新测试主要断言成员存在，没有把它说成动态检查了每一层顺序。

### 2.2 同次读取错误为何不会被剥掉

未变化的 `interpreter_windows.go:30`～`:35` 先取得 finalName 的读取结果 R，再消费真实 CloseHandle 与追加接缝的结果 C。关闭失败时构造：

```text
E = Join(ReleaseError{关闭结果 C}, 读取错误 R)
```

本轮 final 追加 E；releaseFailure 再构造 `Join(受控环境错误, E1, E2, …)`。因此 R 若非 nil，仍在 E 的子树中，未被 rel 提取替代。即使同次还含原生关闭错误及接缝错误，released 的 errors.Join 也不替换原始结果。

继续追踪调用者：interpreters 直接返回 lookup 的错误；Prepare 的失败 defer 仍调用 `failed(step, b.Close())`；failed 把受控的 apperr 副本放在前面，同时保留完整 step 和 Batch 的关闭错误。于是控制展示、原步骤码和全部底层错误可以同时成立：ToEnvelope 用前面的受控 apperr，不会为了避免回显路径而删除底层读取／关闭结果。

**静态证据足以支持当前实现的这项要求。** 这是从具体构造对象 E、追加 E、展开全部 E 到最终保留 step 的直接控制流证明，没有依赖“测试都通过所以一定完整”的推断，也没有要求先增加未经授权的原生故障实验。

### 2.3 失败、健康、重复查询及 Windows Git 路径

- **newFinder**：可继续处理其余根；每个含关闭失败的返回都已在 final 中保存。即使 unverified 被置位，lookup 的 `:135` 检查也先返回关闭失败，不能降为普通“无法核实”。
- **onPath**：可继续寻找后续候选；关闭失败留在列表。即使后来找到健康路径，lookup 的 `:142` 检查在返回路径之前拒绝，早先错误不会清除。
- **普通不存在**：不含 ReleaseError，不加入列表，仍按原规则跳过候选、选择后面的合法路径。
- **gitForWindows**：`interpreter_windows.go:109` 对 Git exec-path 使用 f.final，运行时标记 DLL 和 shell 使用 f.accept，均经过同一保存点；相关失败提前返回 false 后，外层 lookup 仍消费列表，不会误报为普通 shell_unavailable。
- **重复查询**：已有保存项时，在 find 之前拒绝，不再取得／关闭新句柄。Interpreters 再消费同一个 finder 不会新增关闭或丢旧结果。
- **单次所有权**：句柄仍只由 finalPath 关闭一次；本轮的列表只存错误，不调用 Close。未新增重试、信号、句柄补救或宿主禁用条件。
- **Unix**：最终路径实现仍为 EvalSymlinks，不取得上述 Windows 句柄、不产生该 ReleaseError；普通查找规则不变。

退出码仍是 environment／6；原优先级、受控 message 和空 details 保持。最终信封只通过 apperr.From／ToEnvelope 取受控内容，原始路径和注入文本不进入它。这个结论针对生产 Prepare 及实际信封接口，不把 errors.Join 的调试字符串本身称为已脱敏。

**裁定：B-123-2-R 关闭。**

## 3. 新回归、既有对照及变异限制

### 3.1 两个新顶层测试、五个子测试

`release_windows_test.go:178` 的 TestFinderKeepsEveryCloseFailure 含两个子测试：

| 子测试 | 实际构造与判据 |
|---|---|
| two roots | 两个自建目录作为登记根输入；注入 E1／E2 后构造 finder；两次实际关闭，lookup 在 find 前拒绝，路径和 problem 为空，exit=6，errors.Is 同时取得两项 |
| two candidates | 先在无注入时解析根，再对两个实际 pwsh.exe 候选的关闭追加 E1／E2；两次关闭，仍无路径／problem，exit=6，两项均可取得 |

共享 keepsBoth 继续调用实际第 7 步入口 Interpreters，核对两项仍可取得、关闭数仍为 2；经实际 ToEnvelope、JSON 序列化与 `contract.ValidateJSON("error.v1", ...)` 校验。它检查受控说明、零 details、没有 shell_unavailable、没有注入文本，并检查原始及 JSON 转义形式的隐藏路径。

这验证了实际 finder → lookup → 第 7 步，不是假模型；“登记根”是测试输入，不冒称本测试执行了真实 DB worktree 登记。双错误组合继续进入 Prepare 的保留方式由未变化的 failed 调用链证明；新测试本身没有跑完整 Prepare。

`release_windows_test.go:222` 的 TestFinderCloseControls 含三个固定对照：

1. 第一次关闭失败、后续健康：仍退出 6，E1 可取得，无路径，关闭 2 次。
2. 首个目录没有候选、后续健康：选择第二个，关闭 1 次。
3. 全部健康：选择第一个，关闭 1 次。

预期路径来自自建夹具，不由 finder 反算；关闭次数、错误成员、出口和信封共同约束，不仅检查错误文本。子测试非并行，各自的 OnRelease 接缝在 Cleanup 重置，没有串到其他用例。

E1／E2 是追加在真实关闭结果之后的接缝错误。测试没有故意使这些关闭原生失败，不能把它们写成自然故障或泄漏证据。实际原生调用及接缝追加方式未变，未以接缝替换真实调用。

### 3.2 既有七项回归保持

四项通用回归仍验证数量边界、Prepare 原错误与 Scratch／DB 关闭错误并存、Batch.Close 缓存身份、数量说明与清理说明组合。三项原 Windows 回归仍验证 finalPath 单次关闭、关闭失败后仍拒绝、Prepare 的第 7 步错误消费。

其中 TestFinalPathConsumesItsClose 是**受控触发的真实系统调用失败**：测试设置 HANDLE_FLAG_PROTECT_FROM_CLOSE，真实 CloseHandle 返回 ERROR_INVALID_HANDLE；另测接缝与两者并存。它不是自然发生的故障。测试解除保护并关闭自有句柄，检查两个返回结果；清理成功不回写被测关闭为成功。

Scratch／DB 的关闭故障仍只具有注入证据；未构造原生失败。本轮不扩大 Review 124 对其主体修复和所有权的认可范围，也不声称实现了第四批的晚到任务管理。

### 3.3 作者未检出的变异：明确保留覆盖缺口

作者在 PR 回执中披露：把 `append(..., err)` 改成只保存 `rel` 的变异，未被新测试检出。本轮未独立重做该变异，不能将它计为审查者检出或“所有变异均检出”。

原因与源码一致：新测试的 E1／E2 位于 ReleaseError 的关闭子树内；没有同时构造其兄弟节点“读取错误 R”。只保存 rel 仍能满足这些动态断言，所以这些测试不能单独证明 R 的保留。

当前实现实际保存的是 err，并且所有后续层都保留完整对象，故 §2.2 的静态证明覆盖该要求。**这项动态覆盖限制不构成本次实现阻塞，也不意味着接受丢弃读取错误。** 若以后修改该构造／保存链，适合增加“读取与关闭错误并存”的固定对照；本轮不把新增读取接缝或原生实验设为合并前置，不自动授权实施。

## 4. 独立执行与环境限制

实际工具：先确认 go shim 所在路径，随后显式使用已有的 `C:\Users\hoshi\AppData\Local\mise\installs\go\1.27.1\bin\go.exe`；实际为 go1.27.1 windows/amd64。设置 GOTOOLCHAIN=local、GOPROXY=off、GOSUMDB=off，没有调用 shim 安装、升级或扩大权限。它与 CI 的 Go 1.27.3 分别采证。

一次定向运行：

```text
go test -tags nowebui -count=1 -timeout 90s ./internal/checkrun -v
  -run '^(TestFinderKeepsEveryCloseFailure|TestFinderCloseControls|TestFinalPathConsumesItsClose|TestLookupFailsOnACloseFailure|TestRefusalDetailsBoundary)$'
```

**结果：五个顶层测试全部 PASS；其中新增两个顶层下的五个子测试全部 PASS；进程退出 0。** 没有重复运行求绿色，没有建立新机制实验或修改受审测试。保护句柄回归的清理也没有报错。

本轮未本机运行依赖 Git／DB 夹具的四项 Prepare／Batch 回归，不沿用上轮的权限失败把它们计为本机通过，也不声称本轮再次遇到相同失败。其当前固定对象的动态证据来自 CI。没有跑全量本机套件、用户配置检查、原型或作者其他本地变异。

另检查现有本机 Go 的 errors.Join 源码：非 nil 项按传入顺序保存、经 Unwrap 返回；该静态依据适用于本机 1.27.1。CI 1.27.3 上 E1／E2 的实际可达性另由新回归验证，不借本机源码冒称已审阅另一补丁版本全部实现。

`git diff --check b6542be..HEAD` 通过。未新增审查用代码或隔离副本，本轮临时目录／句柄由已提交测试自身负责并完成清理。

## 5. CI 正式结论、实际检出与历史

[最终 run 41267895060](https://github.com/XKHoshizora/specmetry/actions/runs/41267895060)：pull_request、attempt 1、completed/success，8/8。八份原始日志的 checkout 均为完整 `ef065da762a59eab8100c5c523f6e49178a659de`；父提交及同树关系另经提交 API 核验，没有只看 job.head_sha。

| 作业 | Job ID | 正式结果／实际 checkout |
|---|---|---|
| Go lint | 124734435210 | success／ef065da |
| Go tests (Ubuntu) | 124734435361 | success／ef065da |
| Go tests (Windows) | 124734435367 | success／ef065da |
| Web | 124734435423 | success／ef065da |
| Go tests (macOS) | 124734435429 | success／ef065da |
| Build with dashboard (Ubuntu) | 124734632932 | success／ef065da |
| Build with dashboard (macOS) | 124734632951 | success／ef065da |
| Build with dashboard (Windows) | 124734632962 | success／ef065da |

实际 Go 为 1.27.3，Linux/amd64、Windows/amd64、Darwin/arm64。八份日志无测试 FAIL 或作业级 `##[error]` 记录。两处既有局部 lint 豁免保留原范围：Stat 元数据那一行的 gosec，以及缓存结果 `second != first` 的 errorlint，没有改配置或弱化断言。

### 5.1 实际执行集合

| 平台 | 通用旧回归 | Windows 旧回归 | 新顶层／新子测试 |
|---|---|---|---|
| Ubuntu | 四项均执行并 PASS | 不在该平台编译 | 不在该平台编译 |
| macOS | 四项均执行并 PASS | 不在该平台编译 | 不在该平台编译 |
| Windows | 四项均执行并 PASS | 三项均执行并 PASS | 两个顶层、五个子测试均执行并 PASS |

不能把 `_windows_test.go` 在其他平台不编译描述成三平台均通过或正式 SKIP。Windows 共九个相关顶层测试，五个新子测试不再次当作顶层相加。没有按日志重印重复计数。

三个平台各有 27 个带测试包输出 ok，`[no test files]` 另计；Windows CLI 分包不造成重复。普通 Go 测试 SKIP 为 Ubuntu 15、macOS 7、Windows 10，与 Review 124 保留日志逐名比较一致，没有新增回归 SKIP。Scoop shim 仍未构造，不能据此称为已验证。

Ubuntu 普通测试步骤确实以 RUNNER_OS=Linux 选择并传入 `-race`；八项 CrossUser 在该非 root 步骤跳过，后续独立编译 store.test 并以 sudo 运行，八项全部 PASS。root 步骤没有宣称带 race，后来的 PASS 不改写前一步 SKIP。

本次 checkrun 包耗时 Ubuntu 37.855 s、macOS 8.872 s、Windows 36.028 s，仅为观测，不作为上界或 M-16 已解决的依据。常规 CI 不运行独立原型模块，不替代未来生产执行机制、事务组合或整个工作包验收。

### 5.2 旧运行不改写

本轮再次查询四次历史运行的正式元数据，均仍为 pull_request、attempt 1：

| Run | 对象 | 正式结果与历史 |
|---|---|---|
| 41219007149 | 501612d | failure；G703，7/8 |
| 41224740139 | 1a2eda1 | success，8/8；Review 123 对象 |
| 41228654793 | 6e5747e | failure；测试 errorlint，7/8 |
| 41232924416 | b6542be | success，8/8；Review 124 对象 |

旧失败的日志依据及回执按前两轮保留；本轮没有重读全部历史作业或重新诊断其原因。新 run 是新固定 HEAD 的运行，不称为旧 run 重跑，也不称为首次全部通过。

## 6. 整体裁定、剩余事项与交接

| 项目 | 裁定 |
|---|---|
| B-123-2-R | 关闭：每次完整错误均累积、最终链包含全部保存项，后续成功不清除 |
| B-123-2 整体 | 在受审第二批的 Prepare／Batch.Close／Windows 路径句柄及测试清理范围内关闭 |
| B-123-1、N-123-1 | 按 Review 124 原限定保持关闭，未发现直接回退 |
| 新 BLOCKER／其他直接回退 | 未发现 |
| 固定对象技术结论 | GO |
| 合并建议 | 建议负责人决定合并上表固定对象；本报告不执行合并，也不产生该授权 |

没有必须先修改当前实现或补做原生故障实验的技术前置。完整读取错误的动态对照是覆盖增强，不撤销当前静态证明，也不等于允许未来改成只保存 ReleaseError。

以下事项保持原裁定：

- 已删除登记 worktree 造成持续拒绝符合现行冻结规则，仍待负责人决定，不自行放宽、不接受为 L；最迟在公开命令／用户验收前明确，若调整影响第四批预检接线，应先决定相应影响。
- factgen/legality 的差额须在首次用它验证新 Runner 历史前另行处理，不因关闭结果回归绿色而关闭。
- 第四批的任务观察、已结束记录、晚到任务错误与受控输出能力仍未接入；第三批及之后未授权。维护事项、N-087-1 和冻结时保留的证据限制不变。

交接建议：负责人可采纳本轮限定技术 GO，单独决定固定对象的合并及其核验；任何后续批次、资源配置或工作包验收另行授权。不得把本轮资源错误链收口扩大为完整 Runner、WP2.5／M2 验收或新的风险接受。

## 7. 结束核验

结束核验：2026-11-01 13:09:49 +09:00。再次读取 GitHub PR、两个分支 ref 及本地身份：PR 仍 open、未合并，受审 HEAD／远程分支仍为 `eafaf5c39592fcf121b87ffea650abadd2f07e4a`，树仍为 `6878f16770dc21c9c7aae4c5660d726aafdef6a0`，base／GitHub master 仍为 `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4`，没有对象漂移。

工作区唯一新增为未跟踪、未提交的 `docs/reviews/codex-review-125.md`；已跟踪文件和暂存区无改动。没有修改实现、测试或冻结规范，没有触发或重跑 CI／诊断，没有提交、推送或合并，没有授权第三批或宣布验收。报告完成后暂停，等待负责人决定。
