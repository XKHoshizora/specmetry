# Codex Review 098 — PR #39，N-096-2 分类修复定向独立审查

## 1．结论

**技术 GO，建议合并本次固定对象。** N-096-2 的两处分类问题在受审的受控原型范围内可以收口：经核验、尚未进行探测或启动检查的“不支持”可以得到 unsupported；自读 unknown 保留为工具观察失败，不能再按可靠的能力不满足得到 refuted。没有发现新的 BLOCKER、confirmed 假通过或直接回归。

修复代码诊断三平台正式成功；最终 HEAD 的常规 CI attempt 1 正式 8/8 成功，当前 CI 合并条件满足。这里的合并建议只针对原型与文档，不代表 G6 全部满足、规范冻结、生产实现授权或工作包验收。

| 事项 | 裁定 |
|---|---|
| N-096-2：有效 unsupported 提前识别 | 关闭；报告、身份、步骤、登记文件和外层子进程观察共同核验，严格最终检查保留 |
| N-096-2：自读 unknown | 关闭；报告及 `/proc` 步骤保存错误，外层持续 flag；有检查的组仍可能先登记超时，但结果为保守 invalid |
| 共享报告协议、门禁变化 | 成立；原 `awaitOwned` 调用者继续走 nil-ended 的严格规则，unsupported/0 仅匹配明确要求此分类的自检 |
| Review 095 B1 | 维持 Review 097 的关闭范围，没有直接回退 |
| BLOCKER | 无新增 |
| NON-BLOCKER | 两处覆盖说明可收窄，见 §7；R5 的未检出与冗余消费者限制保留 |
| PRODUCT DECISION / ACCEPTED RISK | 无新增；本报告不批准保证调整或接受风险 |
| G3 / G6 / G7 / G5 | G6 增加分类及门禁的局部支持；G3/G7 维持既有局部支持；G5 未满足，P-3 未执行 |

Q-1、Q-2(a)、C-4、D21 保持已批准，不重新列为待选。M-1、M-14～M-17、N-087-1 不因本轮通过而关闭。

## 2．固定身份、工作区与范围

开始核验：**2026-10-28 13:29:19 UTC（22:29:19 JST）**。收尾身份核验见文末。开始时本地跟踪文件、暂存区和未跟踪文件均无改动；目标报告不存在。本轮没有切换、清理、重置或修改其他任务的内容。

| 对象 | 核验结果 |
|---|---|
| PR | [XKHoshizora/specmetry#39](https://github.com/XKHoshizora/specmetry/pull/39)，open、未合并，auto_merge=null |
| base / master | `27ce69d4791ceca8f207d0e8161ee395cebd52e7` |
| HEAD | `8b6aca1ecf8a7f0920adbc8aa195d271a11f2fc6` |
| HEAD 树 | `c603115b12b8cf28da2b618ab101c2a2fd0bfb13` |
| 固定代码点 | `18bc6a455e0115774f1ea46adcf82ca7622a333c` |
| 回归先行 | `e23a7ef265f58f7593ce83069447c155971b809a` |
| 最终 CI 合并快照 | `45ac3443e9e70df0baed8721d6aaa7d84e970def` |
| 快照父提交 | 按顺序为上述 base、HEAD；树与 HEAD 相同 |

已读取适用 AGENTS.md、Review 096/097、全部九文件增量、原型文档 §14.19/§14.20、处理记录 §118，以及共享报告、场景终判、门禁、登记、所有权和清理调用链。本轮不是 T1～T6 或已验收生产模块的全面重审。

### 2.1 提交、文件与历史适用性

实际顺序为：

1. `e880902284e03245bb909b497220dc3eb81d87cd`：诊断分支精确匹配及注释。
2. `e23a7ef265f58f7593ce83069447c155971b809a`：七个故障、一个对照、辅助注入位置显示及明确的 unsupported 自检匹配；仍保留旧的 P-2 分类和等待流程，没有提前修正被测判定。
3. `18bc6a455e0115774f1ea46adcf82ca7622a333c`：分类与共享等待修复。
4. `8b6aca1ecf8a7f0920adbc8aa195d271a11f2fc6`：文档收尾。

base→HEAD 实际为九文件、+482/−31：三个原型测试文件（`gate_test.go`、`p2_linux_test.go`、`report_test.go`）、诊断 workflow、AGENTS.md、implementation-plan、review-resolution、check-runner 草案和原型文档。

代码点→HEAD 只有五份文档变化，`prototype/` 与 `.github/` 无差异。`internal/`、Schema、迁移、根模块依赖、根 `ci.yml` 均未变；草案只更新状态与证据引用，没有改变技术语义。诊断 workflow 仅增加精确分支 `test/wp2.5-prototype-n-096-2` 和注释，路径过滤、只读权限、并发规则、三平台、race、20 分钟作业预算、15 分钟测试期限及管道退出码传播未变。

`docs/reviews/` 没有变化，Review 096/097 保持原文；处理记录原有 7219 行前缀逐行相同，只追加 §118 的 86 行。旧裁定没有被新状态替换。§118 没有把“待独立审查”预写成关闭。

## 3．unsupported 的提前识别与报告发布

### 3.1 顺序与证据

关键链路位于 `prototype/wp25/report_test.go:199`、`:210`、`:255`，以及 `p2_linux_test.go:353`、`:423`、`:496`、`:516`、`:703`、`:733`。

`awaitOwnedUnless` 每轮先读取和解析已到的登记记录，检查事件、重复、数量并逐个调用原有归属核验，再考虑提前结束条件。已有记录不会因同时出现 unsupported 而跳过登记。取得全部预期记录时先返回正常完成，随后仍要核验实验报告。所有原 `awaitOwned` 调用者传入 nil-ended，顺序、固定期限、部分记录的短缺登记及错误拒绝规则保持不变。

P-2 的 ended 回调只接受“报告存在、能解析且 Unsupported 非空”这一提前结束候选；不存在继续等待，读取错误或 JSON 解析错误立即 invalid。回调此时不等于已经接受 unsupported：之后先核对报告 PID 等于实际启动的实验进程、Group 等于当前组，再统一调用 `p2NoCheck`。

`p2NoCheck` 同时要求：

- 没有执行步骤、CheckPID、检查状态、回收结果、安装后的状态或错误等；SubEnd 保持未读取哨兵值 −1，声明 Exit=0。
- 检查的登记文件中没有完整记录；读取或解析失败记为观察错误，不能当作空。
- 外层独立读取实际实验进程的子进程集合，必须为空；读取失败持续 flag，不能当作“没有检查”。

这依赖已审查的受控 helper：真实安装失败或缺少审计架构在首个探测步骤之前返回，之后发布报告、等待父端 release，不再派生检查。报告声明、源程序的停止位置、外层观察和严格最终文件检查共同支持“没有启动本次检查”；不是仅凭 CheckPID=0、一次最终空集合或任意程序的自述作证明。

核验通过才调用 `noneExpected`，它只将此报告的声明改为空、短缺清零，不删除已登记的进程、持续错误或其他报告。`scen.done` 仍先释放本轮闸门、清理及确认对象，再对每份报告做严格最终读取；未知事件、额外、重复、损坏、最终截断或晚到记录继续拒绝。任何 invalid 的优先级高于 unsupported。没有因 SkipNow 跳过 defer 收尾或最终检查。

### 3.2 矛盾报告与已启动检查

若 unsupported 与步骤、检查或外层观察矛盾，代码先 flag，再继续原来的状态核验、release、等待回收、步骤判定及最终清理。不会调用 `noneExpected` 消除短缺。

尤其 after-check：实际检查先报告 PID，外层通过其父进程和启动身份核验后登记，写 ack；实验进程随后才回收该检查并发布矛盾报告。因此此回归不是“声明检查启动了”的模型值：日志有真实 start、pidfd/poll、WNOWAIT 观察、外层登记确认及 reap。修复后的 invalid 仍保留两个进程的已结束证据。接缝故意跳过 SET 0，实验进程退出不是内层清除成功。

### 3.3 发布、新鲜性与解析边界

`p2Helper` 先完整 Marshal，写本次 TempDir 中的 `out+".tmp"`，成功后才同目录 Rename 到 `out`，之后等待 release。WriteFile 或 Rename 失败返回工具退出码；父端不能因此取得有效 unsupported，只会等待或拒绝。每次实验使用新临时路径，没有复用旧报告。

父端不读取 `.tmp`，只读取已发布的 `out`；实际 JSON 发布不同于持续追加的检查登记文件。截断接缝发布的确是一份缺尾字节的 JSON，立即解析失败；缺失报告不提前结束等待。不能把一轮“尚无完整登记记录”说成最终不存在：最终 `checkFinal` 对尾部半行仍严格拒绝。

这里确认的是受控 Linux 临时目录的发布/可见性协议，不新增断电持久性、任意写者认证或所有文件系统原子性保证。`json.Unmarshal` 不是未知字段拒绝的 Schema；本轮没有把它描述为任意不可信 JSON 的完整验证器。

## 4．自读 unknown 与持续错误

`p2_linux_test.go:219` 起的探测先进行真实 `procInfo` 读取；接缝只把所得 presence 替换为 unknown，记录“真实读取为 present”的命中，不直接改 verdict。

unknown 有两个消费者：

1. 报告保存 `SelfReadErr`，且 `/proc` 步骤记录错误；探测照常拒绝，而非重试后恢复。
2. 能进入最终报告处理的外层在 `:759` 持续 flag；步骤序列判定也拒绝该计划外错误。

因此结束时 GET、外层状态、子进程及描述符读取随后成功，不能把本次失败恢复成正常能力不满足。正常的 seccomp 拒绝仍按事先声明的步骤及 errno 判断，可以 confirmed；没有修成全部负向实验一律 invalid。

范围限制保留：在预期有检查的组里，探测先拒绝，报告没有 Unsupported，外层仍等待登记直至 10 秒超时，结果 invalid，尚未进入消费 SelfReadErr 的最终分支。本轮只为经核验的不支持增加提前结束条件，没有把所有提前拒绝改成新协议。这限制原因的精度，不形成 confirmed 假通过。文档对此路径的一句字面描述宜修正，见 §7。

## 5．七项故障、门禁和独立运行

### 5.1 从完整 CI 日志复核旧/新实际链路

以下旧结果来自 `e23a7ef` 的回归先行诊断，新结果来自 `18bc6a4` 的修复诊断。它们是本轮独立读取、按 ID 去重并核对内层结果、进程退出、命中、外层 Final/补救的 CI 证据；不是本机 Linux 重跑。

| 故障 / 组 | 旧内层 | 新内层 | 独立核对的实际命中与意义 |
|---|---|---|---|
| unsupported-once / set0 | invalid/1，登记超时 | unsupported/0 | 安装前一次接缝，明确没有真实安装；新结果不是 confirmed |
| withheld / set0 | invalid/1 | invalid/1 | 安装前及报告不写两个命中，等待登记到期；新旧都拒绝 |
| truncated / set0 | invalid/1，登记超时 | invalid/1，JSON 解析失败 | 实际发布缺最后一个字节的报告；不是旧 confirmed 假通过 |
| foreign / set0 | invalid/1，登记超时 | invalid/1，PID 不符 | 报告写父 PID，实际实验进程身份检查拒绝；不是旧 confirmed 假通过 |
| after-probe / set0 | invalid/1，登记超时 | invalid/1，报告矛盾 | 真实安装、真实七步探测，未启动检查；新错误点名步骤及安装后事实 |
| after-check / set0 | unsupported/0 | invalid/1，报告矛盾 | 真实检查已登记并回收，16 步及一条登记记录；旧错误是分类绕过，不是 confirmed |
| self-read-unknown / set1 | refuted/1 | invalid/1 | 真实读取后的一次替换；报告错误和步骤错误均见于新结果 |

修复诊断中七项外层门禁均 PASS，Final=ended、补救 not-needed、保险清理 nothing left，没有以补救成功重写内层 invalid。有效 unsupported 的内层清理确认一个实验进程结束；after-check 确认实验进程与检查两个对象结束；自读故障确认一个实验进程结束。

withheld/truncated/foreign/after-probe 的内层严格保留“预期检查未登记”的 NOT verified 与 shortfall；外层另行核验实际启动的实验进程已结束、没有本轮残留。不能把外层结束证据改写成内层登记完整，也不能把声明短缺一律解释为存在真实残留。

normal、计划的 waitid.EPERM 拒绝、SET 0 拒绝、SET 1 拒绝四个对照在旧/新诊断均 confirmed/0。首轮 Ubuntu 恰为六项门禁失败：除 withheld 外的新增六个故障；不是七项都失败，也不是六个旧 confirmed 假通过。

### 5.2 门禁没有把预期理由当作全部条件

`gate_test.go:700` 只对明确 want=unsupported 的用例接受 unsupported/0；其他用例仍精确匹配 verdict 与退出状态。`because`、辅助注入位置和命中计数是额外要求，不能替代外层判定。

`judge`（`:620`）继续独立要求：预期拒绝命中、没有未允许问题、没有工具错误、需要时补救成功、Final=ended。保险清理发现矛盾或失败仍影响最终结论。辅助进程 `inject_sites` 的解析失败不追加显示内容，字段本身只是诊断信息；本轮的命中判断还核对接缝代码、实际状态/步骤和内层结果，不能仅凭该字段证明真实执行。

### 5.3 本轮本机独立运行与环境限制

在仓库外导出固定 HEAD，未修改其代码。实际平台为 Windows amd64、Go 1.27.1；使用现有工具、离线依赖、独立缓存，预先限定一次运行、120 秒总测试期限，不扩大实验类型。

选择十项已有共享门禁：silent-holder 与 normal-drain 对照，以及 malformed、missing、unknown-kind、duplicate、unexpected-kind、truncated、overlong、deleted 报告负例；另包含根层 `TestPipeNormalDrain`，避免零场景使 TestMain 将运行判无效。

在导出副本的 `prototype/wp25` 中实际选择如下（GOTOOLCHAIN=local、GOPROXY=off）：

```text
go test -count=1 -v -timeout 120s -run '^(TestGate|TestPipeNormalDrain)$/(^control[.](silent-holder|normal-drain)$|^report[.](malformed|missing|unknown-kind|duplicate|unexpected-kind|truncated|overlong|deleted)$)' ./...
```

结果：**首次执行 PASS，19.928 秒；十项门禁均通过，一个根层机制场景 confirmed，零组合。** 这独立支持 nil-ended 共享调用的严格报告规则和正常排空，没有把内层拒绝计作机制 confirmed。测试结束后未发现本轮 `wp25.test` 残留；没有对未知 PID 动作。

本地日志：`C:\Users\hoshi\.codex\visualizations\2026\10\12\01a13137-991b-7b42-8ea2-3c24008343a0\review098\shared-report.log`，SHA-256 `7863068b9c9ba8fd9620020bf3e67528f1908ac8f28305c583af9d2eb0eb6d33`。该哈希只识别本轮已实际读取的日志，不替代结果核对。

本机尝试查询 WSL 环境即得到 `Wsl/E_ACCESSDENIED`，没有启动 Linux 实验、提权、安装或绕过拒绝。因此本轮**没有独立本机复现新的 P-2 故障、原生过滤器安装失败或作者 R1～R6 变异**。Linux 新旧行为依据源码与现有完整 CI 日志；macOS 也没有本机动态运行。

### 5.4 R1～R6 及作者首次失败

R1～R6 只计为作者证据：文档明确 R5 没有检出，没有写成全部检出。R5 仅移除步骤错误，外层 SelfReadErr flag 仍使 invalid 并满足理由；两个消费者确实构成终判的冗余。作者 R3 同时移除两层后变成 refuted、自检失败，支持两个消费者共同必要的错误分类性质；不等于分别证明每个字段都有独立防回归断言。

R4 移除外层 flag 后仍 invalid，但失败理由不满足；R2 移除核验后，严格最终报告检查是 after-check 的第二道防线。不能把所有“变异检出”解释为检出了 confirmed 假通过，也不机械要求撤回每一道冗余检查都单独失败。当前 `/proc` 步骤 errno 的独立输出尚没有同样强度的回归保护，这是测试证据限制，不是已发现的分类漏洞。

作者本机新树第一次的消息子串不匹配失败、第二次 `control.p2-set0` 保险普查读取 PID 641 环境 permission denied 均保留；第三次通过不抹掉它们。PID 641 的归属、原因及当时完整状态未知，事后不存在不能补全。没有访问作者当时原始本机实验文件，以上只按文档列为作者证据。

## 6．诊断、常规 CI 与历史分别核验

### 6.1 本 PR 原型诊断

核对 API 正式 event、attempt、head_sha、run/job 结论，并读取全部六份作业日志，检查实际 checkout、WP25META、逐项结果及测试退出。

| run / 代码 | Job | 正式结果及去重计数：机制场景 / 门禁 / 组合 |
|---|---|---|
| [40689865305](https://github.com/XKHoshizora/specmetry/actions/runs/40689865305) / e23a7ef | macOS 122821615363 | success：33/33、77/77、13/13 |
| 同上，push，attempt 1，run failure | Windows 122821616096 | success：53/53、92/92、13/13 |
| 同上 | Ubuntu 122821616127 | failure：53/53、105/111、13/13，六项失败如 §5.1 |
| [40691692731](https://github.com/XKHoshizora/specmetry/actions/runs/40691692731) / 18bc6a4 | Windows 122827846029 | success：53/53、92/92、13/13 |
| 同上，push，attempt 1，run success | Ubuntu 122827846205 | success：53/53、111/111、13/13 |
| 同上 | macOS 122827846291 | success：33/33、77/77、13/13 |

每个诊断作业检出各自完整 SHA，结果声明一致。Ubuntu 脚本实际按 RUNNER_OS=Linux 将 `-race` 传给测试；并非只看到无告警就推断启用了 race。两个 Ubuntu 内核分别为 6.17.0-1022-azure 和 7.0.0-1017-azure，不能称为完全相同原生环境的前后对照。故障接缝与源码顺序另行核验，不仅凭两个汇总数字判断修复。

三个作业 Go 1.27.3；修复作业为 Linux/amd64、Windows 10.0.26100/amd64、macOS 26.6.2（Darwin 25.6.0）/arm64。P-2 只在 Linux 运行。三个内层专用测试 `TestInnerForeignPID`、`TestInnerLeak`、`TestInnerStopError` 在根层按设计 SKIP；相关自检在内层实际执行，不能把根层跳过计作通过。共享直接检查、既有十三个组合及伪造归属/清理保护仍实际执行，没有新增跳过来规避失败。

诊断结果行重复打印已按种类与 ID 去重：固定场景并集 **86**（既有 53 + P-1 六个 + P-2 九个 + P-4 十八个），不是把三平台的 139 次场景观察说成 139 个独立场景。新增是七个故障门禁和一个门禁对照，Linux 103→111；Windows 92、macOS 77 不变。直接表测试、门禁 PASS、十三个组合、根层 confirmed 分开统计。有效 unsupported 只在明确的内层自检出现，不是根层又增加一项 confirmed。

### 6.2 最终常规 CI

[40693615289](https://github.com/XKHoshizora/specmetry/actions/runs/40693615289)：pull_request，attempt 1，HEAD 为固定对象，正式 **completed/success，8/8**。八份完整日志均实际检出 `45ac3443e9e70df0baed8721d6aaa7d84e970def`；独立读取 Git 对象，父提交依次为固定 base、HEAD，内容树 `c603115b12b8cf28da2b618ab101c2a2fd0bfb13` 与 HEAD 相同。

| 作业 | ID | 正式结果 |
|---|---|---|
| Web | 122834462190 | success |
| Go lint | 122834462284 | success |
| Go tests Windows | 122834462529 | success |
| Go tests Ubuntu | 122834462751 | success |
| Go tests macOS | 122834462759 | success |
| Build with dashboard Ubuntu | 122834624668 | success |
| Build with dashboard Windows | 122834624845 | success |
| Build with dashboard macOS | 122834624890 | success |

Ubuntu 普通测试带 `-race`；八项跨用户测试在该普通步骤跳过，并在独立 sudo/root 步骤实际通过。这两个步骤不是同一次 root race 测试。真实旧 Git、特定 unsafe Git 环境、平台权限用例以及默认关闭的 M-14 诊断等设计 SKIP 继续单列，不计为通过。根模块 CI 没有运行嵌套原型模块，不能代替 §6.1 的证据。

### 6.3 PR #38 与 master 历史

独立读取 [40674978124](https://github.com/XKHoshizora/specmetry/actions/runs/40674978124) 的两个 attempt 元数据及可取得日志：

- **attempt 1 failure，7/8**，对象 `323228eeffa0df60d0e7b91a945385144f0457dd`。Windows 作业 122770957899 的 Test 步骤从 10:48:55 UTC 开始，没有完成时间，作业于 11:34:11 UTC 结束失败。本轮读取该日志也得到 **404 BlobNotFound**。其测试结果、实际 checkout、具体失败原因仍未知；checkout 步骤元数据成功不替代实际检出日志。失联注释按实施方保存记录列示，本轮没有独立取得该注释内容，不据此定根因或维护事项。
- **attempt 2 success**：Windows 新作业 122805030946 实际执行于 12:05:42～12:18:44 UTC，完整日志检出 `677dd563b6237ff5f47ed1feaf2354c80ac843ec`，测试完成通过。其余七项沿用旧结果：各自起止时间相同，本轮解码后的完整日志与 attempt 1 对应日志相同；不能写成八项都重新执行或首轮 8/8。
- `677dd56` 的父提交为 `2c5cf23a8c5a025be49b895f353af77b70a6248f`、`323228eeffa0df60d0e7b91a945385144f0457dd`，树为 `13dc101d085e0e392934beac9ed428e4eb0daf07`；实际合并 `27ce69d` 的父提交及树相同。
- [master run 40687044170](https://github.com/XKHoshizora/specmetry/actions/runs/40687044170) 为 push、attempt 1、正式 8/8 success；八份日志实际检出 `27ce69d`。它不补全 PR #38 首轮 Windows 未知，也不证明此前失败原因消失。

没有把随后绿色写回历史未知，也没有本轮重跑、触发任何 workflow。

## 7．非阻塞措辞建议与证据限制

**N-098-1：两处覆盖说明精度。**

1. `docs/wp2.5-prototype.md:3045` 称自读 unknown 的有检查组“外层不读报告”。实际 `runP2` 等待回调仍读取并解析报告以寻找 Unsupported，只是不进入后续消费 SelfReadErr 的最终判定分支。建议改为后者。仍然登记超时 invalid，没有漏放或新的协议要求。
2. 同文档 `:3031` 把 normal 和 prctl.set0 一起放在“过滤器装不上”的有检查组。`p2Experiment` 的 normal 对照根本不调用过滤器安装；原生安装失败可发生于 set0 等有规则的组。建议将 normal 作为“预期一条检查登记”的一般说明，将真实安装失败表项限定为可安装过滤器的组。其余等待和控制组逻辑未受影响。

这两点是非阻塞的事实/措辞收窄，不要求本轮扩展实现，也不重新否定已核验结果。R5 只保留为 §5.4 的证据强度限制，不把作者已诚实披露的一个冗余防线未检出自动升格为 BLOCKER。

## 8．门槛与最小后续工作

| 门槛 | 本轮增加的支持 | 仍不能推出 / 后续条件 |
|---|---|---|
| G6 | 在受控 Linux 原型中，经核验的安装前 unsupported 可以正确退出；自读未知不再当能力反证；报告矛盾、缺失、损坏与外来身份继续拒绝 | 有效 unsupported 只有注入证据，未见原生安装失败/缺审计架构；仅现有 x86_64 和已测内核；其他提前拒绝仍可能先登记超时；并非生产能力探测全路径已验证 |
| G3 | 维持现有所有权、独立外层盘点、补救及终判；本轮没有破坏严格归属与清理 | 没有补齐 Windows alive 保持 Job 的构造限制、真实不可终止故障、任意宿主或生产接线证据 |
| G7 | 维持已有恢复异常/模型和错误传播的局部支持，Windows 既有场景及门禁仍运行 | 未新增 Windows 恢复机制实验，不证明生产恢复、事务或所有原生失败已成立 |
| G5 / P-3 | 无新增 | G5 未满足、P-3 未执行；不得由本轮绿色自动缩减支持平台 |

本 PR 合并前没有必须补做的新机制实验或代码修复。建议收窄 §7 的文字，并继续保留 R5、本机 Linux 不可用、PID 641 及 PR #38 首轮 Windows 的证据边界。冻结前仍须按既有清单明确并补足所需规格和必要原型；生产接线、静态启动/回收覆盖、真实输出/事务/取消组合及工作包验收属于获准实施后的工作，不自动变成本次分类 PR 的阻塞。

本轮不重新批准任何产品决定，不自动恢复或重试失败，不把 unsupported 当机制验证成功，不宣布规范冻结、生产开工、WP2.5 或 M2 验收。

## 9．操作与收尾

只读访问源码、固定 Git 对象、GitHub API 与既有日志；本机实验仅使用仓库外固定副本、现有场景及自有进程。没有新机制类型、未知进程动作、工具安装升级、CI 重跑或远程写入。

**收尾核验：2026-10-28 13:46:24 UTC（22:46:24 JST）。** 远程 PR 仍为 open、未合并，auto_merge=null；HEAD、base/master 均与 §2 的固定身份相同。最终常规 CI 仍为 attempt 1、completed/success。没有混入其他提交。

本地 HEAD/树仍匹配固定对象；跟踪文件与暂存区无差异，唯一未跟踪文件为 `docs/reviews/codex-review-098.md`。固定提交差异的 `git diff --check` 通过；未发现本轮 `wp25.test` 残留。原仓库除新增本报告外保持不变，报告未提交。
