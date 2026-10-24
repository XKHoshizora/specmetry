# Codex Review 086：PR #34 完成协议与期限回调的定向独立复审

## 1．结论

**技术 NO-GO，当前不建议合并 PR #34。B-N4-R 不能整体关闭，N4 系列尚未收口。**

Review 085 的三类原反例已经由完成标记正确拒绝；正常 epipe、合法 ok/EOF/迟到字节反证、epipe-late 也保留正确结果。但发现两项当前完成条件的确定残余：

- **B-N4-R/T（BLOCKER）**：真实期限回调已经进入、limit 报告仍在进行时，主路径仍能发布 done。随后父端结束持有者，epipe 路径错误 confirmed/0，ok 路径错误 refuted/1。
- **B-N4-R/I（BLOCKER）**：已有 helper 故障分支忽略报告返回值。ok 报告真实写失败后，仍可生成 epipe 和有效 done，内层错误 confirmed/0。

这不是要求无限增加认证层，也不是一般性的未来错误：并发回调与忽略返回值都存在于当前 helper 内部，反例均经过真实启动、写入、完成改名、父端读取和最终分类，见 §4、§5。

| 问题 | 裁定 |
|---|---|
| Review 085 三类原反例 | 已修复；旧机制错误 confirmed，新机制 invalid，见 §3 |
| B-N4-R 整体 / N4 系列 | **不能关闭**；两项残余仍会接受失效的观察 |
| 完成标记支持什么 | 正常串行路径的报告/后处理/完成文件交接，及缺失、错误 PID、格式错误的保守拒绝 |
| 完成标记尚不能证明什么 | 本 helper 中所有相关写者都已结束、没有被忽略的早先报告错误 |
| 新的直接问题 | B-N4-R/T、B-N4-R/I；另有 N-RENAME 的平台保证措辞收紧建议 |
| B1、N6、已关闭 B-N4/N1 | 未发现本轮直接回退，保留既有范围 |
| 最终原型诊断 / 常规 CI | 均正式 success；没有覆盖本轮独立组合 |
| 当前合并建议 | **暂不合并**，修复完成判定的两项残余并补对应回归 |

14 项产品选择仍未批准，规范未冻结，生产 WP2.5 未授权，WP2.5/M2 未验收；M-14、M-15、M-16 保持未解决。本审查不批准保证调整、接受风险或生产实施。

## 2．身份、范围与记录

审查日为 **2026-10-25（Asia/Tokyo）**；下文核验时间使用 UTC，便于与 GitHub 日志对应。

| 对象 | 身份 |
|---|---|
| PR | [XKHoshizora/specmetry#34](https://github.com/XKHoshizora/specmetry/pull/34) |
| base | `5e9a42c5bbe9b6f7bbb36174a5b53c76c701719a` |
| Review 085 对象 | `4d1e5754175a9b93d7c73faf9999f13ee6f2b3f0` |
| 归档 Review 085 | `69afa87628d920ff4fc1f8300c4a9eac94932afd` |
| 回归先行 | `acfee1fa9d2a4665b2f14858aa526b833dc63130` |
| 最终代码 | `fb3f56d8caebc1bc5d464f1db4cbec29a8faf24b` |
| 文档 | `8c8054bea668462252e781f78fe3581509a21001` |
| HEAD | `bc5ad7841e31eefe08194019eed3282b72c2b7c0` |

**开始核验：2026-10-24 15:20:25 UTC。** 远程 HEAD/base 与固定对象一致，PR open、未合并、auto_merge=null；本地 HEAD 同为固定目标，工作区干净，报告不存在。**15:36:46 UTC** 再查身份未变；原工作区仍无修改，没有自行切换对象。落盘后另记最终核验。

已完整读取适用 [AGENTS](../../AGENTS.md)、[Review 085](codex-review-085.md)、本轮完整增量、[原型文档](../wp2.5-prototype.md) §2.17 的完成协议与错误传播表、[处理记录](../review-resolution.md) §101，以及 helper/报告/完成文件/观察/终止/最终分类及直接回归的必要调用链。

相对上轮 **10 文件，+781/−41**：

- 文档 5 份：AGENTS、implementation-plan、review-resolution、新增 Review 085、wp2.5-prototype。
- 原型 5 份：f1_check、gate、helper、pipe、report。

**fb3f56d → HEAD 两个提交仅四份文档变化**：AGENTS、implementation-plan、review-resolution、wp2.5-prototype。prototype/.github 无差异，原型子树均为 `906fed8374332eadcbfb8d287d3f2e44567292cb`。生产代码、Schema、迁移、根依赖、ci.yml、原型 workflow 及既有冻结契约未改。

Review 085 SHA-256 为 `77693601fde7c66c72f545b6933463e59f9599e29201b14ee6fd4649ee61e0c4`，与登记一致。review-resolution 只追加 §101（101 行），§84～§100 与历史审查未改写。活跃记录准确保留原对象、NO-GO、B-N4/N1 的关闭范围和 B-N4-R 待复审状态。

## 3．已成立的修复：串行完成路径与原三类反例

### 3.1 完成检查实际参与两种结论

| 环节 | 源码与结果 |
|---|---|
| 正常 epipe | 报告返回成功 → afterRecord 成功 → complete；之前的任一步失败均不进入正常完成调用 |
| 正常 ok | 两路输出 Close 成功 → ok 报告成功 → afterRecord 成功 → complete |
| 完成发布 | [helper_test.go:289](../../prototype/wp25/helper_test.go#L289) 写自己的 done-part，检查 Write/Close 错误，再改名为 done |
| 父端 | [pipe_test.go:318](../../prototype/wp25/pipe_test.go#L318) 在终止前等待 done；缺失保留 done=0，读/格式错误进入 failures |
| 两种分类 | [425～432 行](../../prototype/wp25/pipe_test.go#L425) 同时要求已核验 holder 的 done；不是只给 confirmed 加条件 |
| 结束与最终读取 | 完成等待结束后才 kill/confirm/reap，再严格读报告。未确认结束仍记失败，不因报告暂时稳定放行 |
| 错误优先 | 原有冲突、PID、重复、工具错误优先检查保持；成功记录、done 和清理成功都不能清除已收集的 failures |

**正常串行调用链的改动有效。** 问题是“每个相关工具操作都已经纳入这条链”的前提仍不成立，不能从此表推出没有并发报告或忽略返回值。

### 3.2 旧机制与新机制独立复验

| 真实 helper 回归 | acfee1f 旧判定 | 固定目标 |
|---|---|---|
| 完整 epipe 后报告 Close 失败 | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| epipe 后工具期限故障已到达，报告被结束截断 | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| epipe 后期限报告真实写失败 | confirmed/0，外层 FAIL | invalid/1，外层 PASS |
| 同三种故障作用于 ok 路径 | refuted/1，外层 FAIL | invalid/1，外层 PASS |
| 报告记录本身截断 | 旧代码已 invalid | 仍 invalid |
| done-close-fails / done-foreign | 旧代码无此协议，不作旧对照 | 均 invalid/1，外层 PASS |
| 健康 epipe | confirmed/0 | confirmed/0 |
| 合法 ok/EOF/每路迟到字节 | refuted/1 | refuted/1 |
| report.epipe-late | 原有正常结果保留 | confirmed/0 |

本轮独立运行了旧回归先行树、固定目标定向回归与健康对照。表中的 Close 故障是第二次 Close 的真实错误，期限故障点是既有模拟点；没有把它们写成自然硬件故障。新机制对这三类**串行且返回值受检查**的反例有效。

[TestF1Classify](../../prototype/wp25/f1_check_test.go) 的 **23 行固定期望**合理：新增四行检查两条路径缺少完成/他人完成，原来的冲突、错误、PID、观察未结束、缺字节等继续保留，每行另以记录逆序判定。真实 helper 门禁与纯判定表分开计证据；表测试不能证明 done 的生产者遵守所有完成前提。

## 4．BLOCKER B-N4-R/T：期限回调已进入，主路径仍发布 done

### 4.1 当前源码缺少顺序保证

[helper_test.go:34～37](../../prototype/wp25/helper_test.go#L34) 对每个 helper 启动 `time.AfterFunc`，回调独立执行 `report("limit")` 并退出。没有保存 Timer、停止/仲裁回调、标记期限已发生或等待已进入回调的机制。

[complete:276～306](../../prototype/wp25/helper_test.go#L276) 不读取这项状态，也不与回调同步。主路径自己的 report/afterRecord 返回 nil，不代表另一个已经开始的报告也完成了。

因此如下顺序可达：

1. 真正的期限回调到期并开始执行；
2. 回调暂停在 limit 报告之前或报告过程中；
3. 主路径正常写 epipe 或 ok，并成功发布 done；
4. 父端读取 done，把它当作观察完成；
5. 父端结束 holder，截断回调；最终报告不含 limit；
6. 分类接受这份不完整的终态。

这属于**完成前已开始的相关失败/报告**，不是完成之后的无关行为。完成文件的原子可见性即使完全成立，也不能补出缺失的线程间顺序。

Go 官方说明 AfterFunc 在独立 goroutine 中执行回调；对已开始的回调，Stop 不负责等待其结束，需调用方自行协调。不能仅增加一个无条件 Stop 再继续发布 done。[Go time（本轮网页为 Go 1.27.3）](https://pkg.go.dev/time#Timer.Stop)；本机 Go 1.27.1 对应源码的规则相同。

### 4.2 独立真实到期调度

仓库外固定目标副本中：

- 仅对本轮新增的 holder 调度将真实 AfterFunc 时长缩为 **100 ms**，避免等待默认长时间；仍由真实 Go 定时器执行原回调，不直接调用假的回调或伪造 verdict。
- 回调进入后留下本轮标记，在可释放闸门等待，之后仍执行原有真实 limit 报告与退出。
- complete 入口只等待“回调已经进入”的标记，再继续**原有** Write/Close/Rename。父端仅记录标记和实际 done PID，未修改分类器。
- epipe 与保留两路读端的 ok 两种真实路径均测试，外层新增门禁固定要求 invalid。

| 路径 | 关键证据 | 实际结论 |
|---|---|---|
| epipe | 回调已进入、报告仍 gated；done=51284，最终为该 holder 的合法 epipe | **confirmed/0** |
| ok | 回调已进入、报告仍 gated；done=68664，真实两路 EOF/迟到字节成立 | **refuted/1** |
| 健康 epipe 对照 | 无故障，done 正常 | confirmed/0 |
| 合法反证对照 | 无到期故障，done 正常 | refuted/1 |
| 正常排空 | 正常 helper 完成工作 | confirmed/0 |

epipe 路径父 67804/holder 51284、ok 路径父 49136/holder 68664 均由内外层核实结束。**内层错误绿色/错误反证，外层因正确发现它而 FAIL**，没有声称整个故障诊断仍绿色。

反向对照在同一闸门上先放行回调，并使主路径不发布 done，直到回调执行真实 limit 报告和退出。修正夹具的负载断言后，两条内层均 invalid/1，外层接受；另有正常排空，整个对照 PASS/0。最终报告实际包含 limit，拒绝来自原有观察/报告/完成分类，不是伪造最终结论。

这验证的是**实际到期且回调先进入**的窗口，不是默认 60 s 自然长跑，也不依赖任意恶意程序或 PID 复用。

### 4.3 最小修正

完成与期限必须具有共同的、可验证的先后关系。至少要做到：

- 期限已经赢得执行权/回调已进入时，主路径不能发布有效 done；
- 允许主路径完成时，保证没有尚未解决的本次期限报告；
- 若用 Timer.Stop，正确处理其返回值及已开始回调，不能把 Stop 当作 join；
- 回调错误与主路径错误都持续影响工具有效性。

不要求引入 broker、系统服务、无限层确认或生产 Runner；这是本 helper 已有两个执行者之间的仲裁。完成之后真正无关的行为仍可在受控协议中排除，但不能靠重命名观察区间排除上述先到回调。

## 5．BLOCKER B-N4-R/I：被忽略的报告错误仍可携带完成标记

### 5.1 具体调用点

[helper_test.go:129～137](../../prototype/wp25/helper_test.go#L129) 的 `fail-after-ok / ok-epipe-reported` 分支仍忽略 `report("ok")` 和 stderr Close 的返回值。随后继续写出失败、调用 `report("epipe") → afterRecord → complete`。

此外，[151～152 行](../../prototype/wp25/helper_test.go#L151) 的反序故障分支也忽略 epipe 报告返回值。因此文档“之前任一报告操作失败必不提交完成”不能只凭正常分支成立。

### 5.2 独立真实写失败组合

本轮在外部副本复用已有 `ok-epipe-reported` 场景，只让这次 ok 报告以只读方式打开并执行真实 Write。保留：

- 真实报告函数的错误返回；
- 原调用点忽略返回值的行为；
- 之后真实 stderr 写失败、epipe 报告成功及原有 complete；
- 原有分类、门禁期望及进程确认/清理。

夹具记录确认 ok Write 实际得到 Access is denied；此处错误是**故意只读打开导致的原生写入错误**，与早先受限执行环境的 Rename 拒绝是两回事。

结果：合法 done 的 PID 为 **14952**，最终只有合法 epipe，内层 **confirmed/0**。原有 `report.ok-epipe-reported` 门禁想要 invalid，正确 FAIL；父 10008、holder 14952 均核实结束。健康与可靠反证对照仍正确。

这不是输入记录冲突没有修好，而是**报告错误被当前调用点忽略，完成生产者错误地声明成功**。回调问题修好后，此项仍会独立存在。

### 5.3 最小修正

在所有相关 helper 分支中保留/传播报告及关闭错误，或由报告层记录不可清除的本次工具错误状态，使 complete 在任何这类失败后拒绝发布。特别故障分支不能因“本来用于构造反例”而豁免门禁正确拒绝的要求。

新增组合应覆盖“先前报告真实失败、后续记录成功”，而不是只测试完整冲突或同时损坏两条记录。无需扩大为任意不可信写者认证。上述反序分支另有忽略返回值的源码风险，本轮没有为其单独声称动态绿色复现。

## 6．完成文件、超时与平台前提

### 6.1 当前受控文件协议可认可的范围

| 项目 | 结果与边界 |
|---|---|
| 新鲜性 | 每个场景使用新的临时目录与独占新建 report 路径；done/part 派生自它，同一次正常 holder 只发布一次，没有复用旧场景标记 |
| 文件内容 | 严格要求规范单行 `done <正整数PID>\n`；空、截断、重复、垃圾、错误 kind、非规范数字等拒绝 |
| 错误 PID | 解析得到 PID 后由分类器核对 holder；不是写者认证 |
| 缺失/只有 done-part | 不当作完成；done=0，分类 invalid |
| Read 错误 | awaitDone 返回错误，持续进入 failures |
| Write/短写 | WriteString 经 os.File.Write，短写会返回错误；代码据错误阻止 Rename。此项核对本机 Go 1.27.1 标准库，不声称注入了内核短写 |
| Close 失败 | done-close-fails 第二次 Close 的真实错误使 Rename 不执行；门禁实际 invalid |
| Rename 失败 | 返回错误；当前受控正反例显示无有效完成，不把 part 或目标目录当完成 |
| 超时 | 完成等待后再终止；缺失完成不会因杀后文件稳定而恢复有效 |
| 截止附近到达 | 当前实现在轮询截止后还读一次文件，允许恰好附近到达的有效 done；未测试精确纳秒边界，不把它表述为硬实时拒绝线 |

本轮在外部副本增加 **15 项独立读取边界检查**（包括合法、外来 PID、空、截断、重复、垃圾、零/负/前导零、错误 kind、坏 PID、额外空白、part-only、missing、读取目录错误），均通过。另有正常发布、part 为目录导致 Open 失败、目标为目录导致 Rename 失败 **3 项发布对照**，均通过，连同正常排空整体 PASS/0。

其中外来 PID 在解析器层只是读出数值，最终拒绝由原 23 行分类表及真实 done-foreign 门禁证明；没有把解析等同于归属验证。未独立构造磁盘故障、内核短写、恶意文件替换或断电，不增加这些门槛。

### 6.2 NON-BLOCKER N-RENAME：限定平台保证措辞

§2.17 把同目录 os.Rename/Windows MoveFileEx 直接表述为“原子：全有或全无”。Go API 对非 Unix 平台并不提供这种无条件承诺；MoveFileEx 的调用名称也不足以证明所有文件系统的可见性语义。[Go os.Rename](https://pkg.go.dev/os#Rename)、[Microsoft MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw)。

本机为 **C: NTFS**，当前原生受控临时目录中，先写/关闭 part、发布到原先不存在的同目录 done、严格读取单行的正常/失败对照成立；三平台 CI 也提供当前布局证据。未复现本布局中的部分可见或错误发布，所以**不据一般性 API 边界新增一个原子性 BLOCKER**，也不要求全文件系统审计。

建议将措辞限定为已支持、已验证的同目录同卷临时目录/文件系统条件，以及严格完整内容校验；不要由 os.Rename 自动推广为所有 Windows 平台原子性。该措辞问题不能解释或豁免 §4、§5 的确定反例。

## 7．独立运行、环境失败与直接回归

实际平台为 **Windows 11 / 10.0.26300 / amd64，Go 1.27.1**。使用已有工具、`GOTOOLCHAIN=local`、`GOPROXY=off`，外部副本均由固定 Git 对象导出。没有运行用户配置、安装升级工具或修改原仓库代码。

| 独立运行 | 结果 |
|---|---|
| 旧 acfee1f，60 s 期限 | 六项旧误分类复现（三项 epipe confirmed，三项 ok refuted）；epipe-cut 原已拒绝；健康与正常排空保持 |
| 固定目标定向回归，360 s 期限，原生环境 | **PASS**：顶层 3 场景 confirmed、72 门禁、13 组合，及 23 行 F1Classify、停止/时钟/归属等直接检查 |
| 真实回调重叠，30 s | epipe 内层 confirmed/0、ok 内层 refuted/1，外层各正确 FAIL；健康对照成立 |
| 放行回调对照，25 s | 两条内层 invalid/1，外层 PASS；正常排空 confirmed，整体 PASS/0 |
| 忽略报告错误组合，25 s | 内层 confirmed/0，原外层门禁 FAIL；正常/反证对照成立 |
| 完成文件边界，20 s | 15 项读取、3 项发布、正常排空，PASS/0 |

固定目标定向选择为：

`^(TestGate|TestGateCombos|TestF1Classify|TestStopRead|TestStopAPIErrors|TestLingeringFailedStop|TestPhaseClock|TestCandidates|TestGateJudge|TestPipeNormalDrain|TestPipeEscapedHolder|TestPipeF1SkipDrain)$`

没有将门禁正确拒绝计入机制 confirmed；没有把本机 3 个场景说成全部 29 个。新调度只操作自己启动且已核验归属的对象；所有被测 holder/父 helper 均由协议核实结束，结束核查没有 review086 进程残留。可释放闸门没有作为永久残留留在运行进程中。

**首次失败与无效夹具明确保留：**

1. 受限执行环境最初拒绝 done-part → done 的 Rename，实际错误为 Access is denied。未经改动的目标因此出现健康 F1 拒绝；整个初次定向运行 FAIL。该环境结果不能证明产品逻辑错误。
2. 初次定时器调度同样没有 done，连健康对照都 invalid，**不计为有效反例**。用观察副本取得实际 Rename 错误后，经工具权限审查批准，仅在同一仓库外夹具的原生环境运行；正常改名/健康对照成功，再取得 §4 的有效反例。没有改算法、替换 Rename 或降低文件校验来绕过失败。
3. 放行回调的第一次对照被夹具自身“报告仍 gated”的负载断言提前拒绝，不能算作最终分类检出。调整该观察断言以区分放行模式后重新编译，第二次实际经过报告/完成/最终分类，结果见 §4。
4. 忽略返回值夹具第一次文本替换误及 complete 内同名错误检查，编译失败；接着一次启动因二进制不存在未运行测试，shell 的表面退出值不计 PASS。修正外部夹具、成功编译后才记录 §5 的实跑结果。
5. 没有把编译失败计作变异检出，也没有覆盖原日志；初次受限运行、观察运行、首次放行对照及实际反例日志分别保存。

作者 M63～M65 属于作者 Windows/WSL 辅助证据，未冒称本轮独立撤回。它们验证等待完成、串行错误后不完成、PID 等检查，没有验证期限回调仲裁或所有忽略返回值的组合。

本机未运行 Linux/macOS 动态调度；平台相关结论来自源码、官方机制和对应 CI。没有把受控调度当作原生硬件故障，也没有声称测试了默认长期限下的所有竞争。

B1 的晚到错误/API 错误/任务移交与 N6 的同一时钟、固定预算、提前唤醒均直接回归通过。forge-owner 对照仍零动作/零信号，自有混合遗留仍实际清理；补救失败、观察失败或状态未知的原有组合不能 PASS。此前关闭范围不重开。

## 8．CI 与历史证据身份

已取得最终两套运行全部 11 个作业的完整日志并核对实际 checkout、正式状态、相关直接检查与 SKIP；未触发或重跑 CI。

### 8.1 最终代码原型诊断

[run 40103135762](https://github.com/XKHoshizora/specmetry/actions/runs/40103135762)：**push，attempt 1，completed/success**。三个 Job 均实际检出 `fb3f56d8caebc1bc5d464f1db4cbec29a8faf24b`。

| Job / ID | 机制场景 | 门禁 | 组合 |
|---|---:|---:|---:|
| Ubuntu / 120899255309 | 38/38 confirmed | 76/76 PASS | 13/13 PASS |
| Windows / 120899255170 | 29/29 confirmed | 72/72 PASS | 13/13 PASS |
| macOS / 120899255305 | 27/27 confirmed | 67/67 PASS | 13/13 PASS |

场景 ID 去重并集 **53**，Results 重复输出不重复统计。Ubuntu 实际 Linux 分支启用 `-race` 并执行原型 go test，未报告竞态；另两平台不是 Race Detector 证据。

三项顶层内层专用 SKIP 为 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError`，相应门禁内层真实运行，不把 SKIP 计为通过。23 行 F1Classify、适用平台的 StopAPIErrors/CensusUnreadable、StopRead/LingeringFailedStop/PhaseClock 等实际 PASS。

新增九项门禁均 invalid/非零，外层接受；健康 epipe、合法反证、epipe-late 结果正确。这套已提交证据没有包含本轮两项完成前提反例。

### 8.2 最终文档 HEAD 常规 CI

[run 40105337157](https://github.com/XKHoshizora/specmetry/actions/runs/40105337157)：**pull_request，attempt 1，completed/success，8/8**。八个 Job 实际 checkout 全部为：

`dd8e6c57bb7e0464c8f79ef694c5914c0a9a0e9e`

GitHub commit API 核对其父提交依次为固定 base、固定 HEAD，树：

`4578e109a9bc417295b5932d6844d7ec1da38290`

与本地固定 HEAD 树一致。

| Job | ID | 最终结果 |
|---|---|---|
| Web | 120905750613 | success |
| Go lint | 120905750826 | success |
| Go tests Windows | 120905750842 | success |
| Go tests Ubuntu | 120905750850 | success |
| Go tests macOS | 120905750876 | success |
| Build Windows | 120905856043 | success |
| Build macOS | 120905856048 | success |
| Build Ubuntu | 120905856161 | success |

Ubuntu 普通 Go 测试带 -race；单独 root 跨账户步骤实际执行并通过，与普通步骤的非 root SKIP 分开。平台/旧 Git/默认关闭诊断等设计跳过不算新通过。**常规 CI 不运行原型独立模块**，不能替代原型证据或本轮反例。

代码到 HEAD 只有四份文档变化，因此固定代码原型诊断的内容适用性成立；不是文档 HEAD 新做了一次三平台原型诊断。

### 8.3 回归先行、取消与中间绿色

[run 40102730930](https://github.com/XKHoshizora/specmetry/actions/runs/40102730930)：acfee1f，push，attempt 1，**failure**。三个 Job 的完整日志与实际 checkout 已核对：正好六项新回归失败，Ubuntu 68/74、macOS 59/65、Windows 64/70；机制 38、27、29 confirmed，组合各 13/13。没有将旧绿色或无 FAIL 字样代替正式结果。

同提交常规 [40102733031](https://github.com/XKHoshizora/specmetry/actions/runs/40102733031) 为 **cancelled**；最终代码常规 [40103137419](https://github.com/XKHoshizora/specmetry/actions/runs/40103137419) 正式 success。它们不替代最终 HEAD 的身份与状态核验。

macOS 旧 2088/1989 的归属、根因、完整清理仍未知；本轮新绿色既不回填它们，也不构成永久不许合并的理由。M-14/M-15/M-16 以及更早失败、取消和旧 N6 观测保持原状。

## 9．必要修正与后续边界

**当前原型必须处理的最小事项**：

1. 完成发布与实际期限回调进行可靠仲裁；已开始的相关回调/报告不得在 done 之外被截断后隐去。
2. 检查并持续保存所有相关报告/关闭错误，修复忽略返回值后仍能完成的分支。
3. 为真实回调已进入、ok 报告失败后 epipe 成功两组增加端到端门禁；保留正常 confirmed/refuted 与原三类回归。
4. 同步完成协议和错误传播表，不再无条件宣称 done 是唯一最后写者或之前任何失败都必然阻止发布。

N-RENAME 为非阻塞的支持范围/措辞建议，不要求断电保证或全文件系统证明。旧 macOS 不可追溯作为历史观测保留，不需反复施压追到 EINVAL。

**冻结前门槛**仍是十四项产品选择、C-4/D20 资源与宿主后续调用、Linux 全局闸门/C-9 拓扑、macOS ABI/平台前提、T5 入口/PATH 等既有规格与必要原型。**授权后的生产工作**仍包括 Runner 接线、事务/记录、取消、配置固定、空批次、§8.6 持久化组合及 CLI/E2E。没有把这些未授权生产工作作为本原型 PR 的新增 BLOCKER。

原三类反例已修复，完成文件本身的受控检查得到支持；但两项具体完成前提仍不成立，**B-N4-R 与 N4 系列不能整体关闭，技术 NO-GO，当前不建议合并**。两套最终 CI 正式成功与该技术裁定分别成立。

本轮只新增本报告，未修改其他原仓库文件，未提交、推送、合并、改基、触发/重跑 CI、安装升级工具或执行用户项目配置。未批准新的产品选择、保证调整或接受风险，未冻结规范、授权生产实施或宣布 WP2.5/M2 验收。

**落盘后最终核验（2026-10-24 15:41:22 UTC，即 2026-10-25 00:41:22 JST）**：远程 HEAD/base 仍为固定对象，PR open、未合并、auto_merge=null；两套最终 CI 仍为 attempt 1、completed/success。本地仅新增未跟踪的 `docs/reviews/codex-review-086.md`，跟踪文件无差异，Review 085 哈希未变，仓库相对链接目标均存在，未发现本轮 review086 测试进程残留。没有扩大审查对象。

