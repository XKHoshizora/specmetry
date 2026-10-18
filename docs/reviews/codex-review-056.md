# Codex Review 056：PR #25 诊断结果尾缀修复定向独立复审

审查日期：2026-10-19。审查者：Codex。

**GO。Review 054 B1-R 可以关闭；结合 Review 054 已成立的其他修复结论，Review 051 B1 可以整体关闭。** 本结论针对诊断工具的有效性，不表示 M-14、M-15 已解决，不批准插桩、生产重试或错误分类调整。

## 一、固定身份、范围与整合

- [PR #25](https://github.com/XKHoshizora/specmetry/pull/25)。
- 固定 base：`46a4b26855cc582b6387883260433c7badb1f2f9`。
- 固定 HEAD：`ccc2e4a47e90d498d64fbcf9d9d35cb12d0d5791`。
- Review 054 原对象：`d2f1d35a5d38564893d991b7c6d872b854e5e8b0`，当时固定 base 为 `a236e5350ff92dcffd9bc05d09b1f3aac3996704`。
- 已完整阅读归档的 [Review 054](codex-review-054.md)，复用其对 N1～N4 和其他有效性修复的结论，没有重开 M0／M1 审查。

开始和收尾核对，远程 PR 均为 Open，HEAD/base 均与本轮固定身份相同。最终 API 的 mergeable 为 true、mergeable_state 为 clean；这不是合并授权。

`8650ab40a0f8947a2d87e6a30488a12ff028da92` 的两个父提交为原目标 `d2f1d35` 和新 master `46a4b26`。相对新 base 共七个文件：诊断工作流、两个诊断测试文件、TestMain 登记、诊断说明、Review 051／054 归档。生产 Open、目录与版本保护、错误分类、重试均未修改。继承的 PR #26 夹具修复属于新 base，本轮不重审。

修复顺序为：`ac57da6` 归档 Review 054；`ff09fad` 先提交反例；`94d1b78` 修改解析；`ab53f16` 记录说明；最终 `ccc2e4a` 仅更新诊断文档。原仓库开始时干净，未切换分支；所有运行来自仓库外固定树副本。

## 二、BLOCKER

**无。B1-R 已关闭。**

### 1. 真正要求第二次 Decode 得到 EOF

位置：[m14_diag_test.go 第 421～440 行](https://github.com/XKHoshizora/specmetry/blob/ccc2e4a47e90d498d64fbcf9d9d35cb12d0d5791/internal/store/sqlite/m14_diag_test.go#L421)。

第一次 Decode 仍设置 DisallowUnknownFields；成功之后再次 Decode 到独立的 json.RawMessage，只有 errors.Is(err, io.EOF) 成立才接受。第二个合法值产生 nil，非法尾缀产生语法错误，均不能通过。原来的 Decoder.More 已不再承担顶层结束判断。

外围 TrimSpace 保留合法结果前后的空白以及 Println 的末尾换行；去除外围空白后，内部 LF 仍使结果拒绝。未知字段、空结果、首个 JSON 不完整依旧拒绝。没有把“第二次解析没有拿到有效对象”误当成 EOF。

### 2. 整条诊断链路拒绝损坏结果

`m14Spawn → m14Parse → m14CheckOpen／m14CheckProbe → 汇总有效性` 保留原来的失败传播。解析失败产生 Invalid，不会把零值 Outcome 当成真实 SQLite 错误或有效调用；诊断记录 valid=false、无效原因，并使测试失败。

新增的 trailing-bracket／trailing-brace 仅在明确设置 M14_FAULT 时附加尾缀。工作流没有设置该变量；常规回归在隔离子诊断中才启用。它们不修改生产 SQLite，不把故障注入混入常规压力观测。

### 3. 修复前后独立对照

Windows/amd64，Go 1.27.1，SQLite 3.53.4，modernc.org/sqlite v1.60.1；两个固定树均离线构建 SQLite 测试二进制。修复前使用测试先行提交 `ff09fad`：生产诊断解析仍是旧实现，新增尾缀回归已存在，没有人为改写旧解析器。

| 检查 | 修复前 ff09fad | 最终 ccc2e4a |
|---|---|---|
| TestValidityOfM14Parse，16 个输入 | `]`、`}`、`}]` 被错误接受，测试失败 | 全部通过 |
| 真实 Open helper 输出合法 JSON 后附加 `]` | 2 个 helper 真正 Open、各提交 5 条事实；父诊断 valid=true、exit 0，回归失败 | 父诊断失败，结果记为无效；verdict、records 两个断言均通过 |
| 同上，附加 `}` | 同样错误 PASS，回归失败 | 正确拒绝，两个断言通过 |
| 真实只读探测／写入子进程输出附加 `]` | 对照组和有提交组仍错误 valid=true，回归失败 | 实验无效，父诊断失败，两个断言通过 |

旧版 Open 尾缀实验确实提交了 10 条合成事实；旧版探测实验也形成了探测和提交。这排除了“辅助程序没有执行，所以没有碰到尾缀”的假阳性。新版测试检查诊断子进程的真实退出结果和 M14-RESULT，不仅检查解析辅助函数返回值。

纯解析正例包括原串、Println 换行和外围空白；反例包括额外 token、第二份 JSON、数字、截断、未知字段等。正常输入的解码内容也有断言。

## 三、直接回归与 NON-BLOCKER

有效对照、无写入控制组和真实 SQLite 写入错误的观测均保持：小规模正常 Open 为 2 次调用、10 条事实；触发器拒绝所有写入时记录 code 1811、零提交，但实验本身有效；探测控制组有实际探测、零提交，有提交组具有实际提交和时间区间交集。取消、工具失败和 SQLite 错误没有重新混为一类。

**N1：本机观察到既有挂起用例的总耗时断言偶发超限。**

- 位置：[m14_validity_test.go 第 191、208～220 行](https://github.com/XKHoshizora/specmetry/blob/ccc2e4a47e90d498d64fbcf9d9d35cb12d0d5791/internal/store/sqlite/m14_validity_test.go#L191)。
- 首次较宽运行中，21 个诊断场景有 20 个通过；挂起场景正确拒绝、records 通过，但总耗时 15.142 秒，超过 15 秒断言，整次测试退出 1。不能把这一批记为全通过。
- 单独原样重跑该场景为 5.49 秒，verdict、records 均通过。没有扩大 Timeout、改断言或跳过该场景。
- 这不足以定位调度、主机负载或启动开销的具体根因，也没有显示损坏结果被放行或未回收进程。尾缀修复不改变期限实现，故不阻塞本轮。
- 后续若维护此测试，建议把精确耗时断言与大量并行子诊断的启动负载分开验证，保留硬期限和失败断言。此条是本轮的新测试稳定性观测，不重新打开 Review 054 已合理处理的 N2，也不关闭任何既有维护事项。

其余本机结果：seq 缺口、阶段分类、区间交集及解析测试通过；未显式启用的两个 TestM14 按设计 SKIP。没有重跑大规模压力诊断，没有本机复现 264／1288，也没有本机运行 Linux、macOS 或 Race Detector。

## 四、分 SHA 的 Actions 证据

### 1. ab53f16 的诊断运行

[run 39143131311](https://github.com/XKHoshizora/specmetry/actions/runs/39143131311)，Attempt 1，push 事件，macOS／Windows 两个作业成功。两个日志实际检出和结果行的 SHA 均为 `ab53f161786b604075a01ac817cb24c2abda62d3`。每组独特结果只计一次，没有把 Results 步骤重复打印的行再次计入。

| 项目 | macOS | Windows |
|---|---|---|
| 无写入对照首轮探测 | 421,445，全部 ok；零提交 | 74,875，全部 ok；零提交 |
| 有提交首轮探测 | 300,908：300,885 ok、22 次 264、1 次 1288 | 18,543，全部 ok |
| 立即重试 | 23 次，全部成功 | 0 |
| 实际提交 | 121,289 | 11,286 |
| writer 观测 | 3 个中 2 个以 code 5 提前停止 | 同左 |
| overlap_ms | 59,873 | 59,969 |
| 真实 Open 路径 | 600 次调用且打开成功，60,000 条事实，零失败 | 同左 |
| 工具／数据库检查 | 三个独特结果 valid=true；无数据库问题 | 同左 |

此数据与 [诊断说明 §5.3](../m14-diagnostics.md#53-本版工具审查-054-之后ab53f16) 的计数相符。两个平台都至少有一个 writer 在 1 秒内停止；不能把首末区间重叠说成所有 writer 全程并发，更不能证明某次 SQLite 临界区重叠。

264／1288 仍只发生在辅助只读探测中，此次真实 Open 没有失败；Windows 此次真实 Open 写入阶段没有 code 5，不表示 M-15 消失。23 次重试成功也不是生产重试的批准依据。没有将旧版本运行和本次运行合并统计。

### 2. 两次常规 CI

| 运行 | HEAD | Attempt／结果 | 实际检出 PR 合并快照 |
|---|---|---|---|
| [39143132116](https://github.com/XKHoshizora/specmetry/actions/runs/39143132116) | ab53f16 | 1；8/8 success | `247021ada508fa4438b9f3dab1974c6a1b4d28ec` |
| [39144394274](https://github.com/XKHoshizora/specmetry/actions/runs/39144394274) | ccc2e4a | 1；8/8 success | `67aaf609c2839eed11792814614aa4a9317fa28d` |

逐个核查八个作业：Web、Go lint、三平台 Go tests、三平台 dashboard 构建全部成功。合并快照的父提交分别是 base `46a4b26` 和各自 HEAD，树分别为 `0cd0c399a533df28a0d4bafe62bdf4c779a0d012`、`d52c37981af0f1db128f766a81807931ef2807e6`，均与相应 HEAD 树相同。

三平台日志均实际 PASS 新解析测试和三个尾缀链路场景；Ubuntu Test 步骤启用了 -race，Linux 专属 root 步骤另行成功，其他平台按设计跳过该步骤。两个顶层诊断在常规 CI 默认 SKIP，不能把这种 SKIP 写成新的压力运行。

`ab53f16 → ccc2e4a` 仅修改 docs/m14-diagnostics.md。最终 HEAD 的常规 CI 可以证明该树的既有测试通过；ab53 的诊断观测适用于相同诊断源码，仍必须保留原运行 SHA，不能改写成 ccc2e4a 又做了一轮压力运行。

## 五、归档、保证与最终裁定

Review 054 归档 blob 与首次归档提交 ac57da6 完全相同，仍指向 d2f1d35 和旧固定 base。Review 051 原结论、历史 lint 失败和取消记录保留；文档没有由作者自行宣布 B1-R 关闭。PR #26 的整合未改变生产数据库路径。

**ACCEPTED RISK：本轮无新增接受风险。** Review 054 对 N1～N4 的既有裁定不被扩大成全局数据安全证明。业务事实未提交、目录检查／打开锁，以及批准的 WAL／SHM 辅助文件边界仍须分别理解；本轮没有证明所有失败 Open 都绝对无文件副作用或所有数据库内容均完整。

最终回答：

1. Review 054 B1-R：**关闭**，非法尾缀、第二份 JSON 必须得到非 EOF，真实父诊断随之失败。
2. Review 051 B1：**整体关闭**，复用已验证的其他有效性修复，不作穷尽证明。
3. PR #25：**GO，可以进入负责人控制的合并收尾**；本机一次耗时断言失败如实保留为 NON-BLOCKER。
4. 工具有效性通过不关闭 M-14／M-15，不批准插桩、生产重试、错误映射或 WP2.3 实现。下一步由负责人决定是否合并及另行授权诊断工作。

本轮只新增本报告和同时获准的 Review 057 报告；未修改原有文件、提交、推送或合并。
