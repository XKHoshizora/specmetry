# Codex Review 049：WP2.1／WP2.2 最后一处合法域残余定向复审

审查日期：2026-10-18。审查者：Codex。

**结论：PR #21 GO；PR #22 GO。Review 048 最后一处 B1-R2 可以关闭，B1 可以整体关闭。** 两个门禁分别拒绝零声明输入的 input_missing，手写决策表已使用合法配置与可成立的缺失路径。没有发现本轮直接引入的 BLOCKER 或确定算法反例。该结论限于两个工作包的纯函数与测试复核器范围，不宣布工作包验收或 M2 整体验收。

## 一、审查身份与实际增量

| 对象 | 上轮目标 | 本轮固定目标 | base |
|---|---|---|---|
| [PR #21](https://github.com/XKHoshizora/specmetry/pull/21) | `ca2aec2309e0e3b7e556e927f62bde7cfb9b4b88` | `6178f9d1d347a1ecf013c343b483d212df380ae8` | `666efe2c907d518b91cd86710a18989efa5dfdb4` |
| [PR #22](https://github.com/XKHoshizora/specmetry/pull/22) | `53a186c3a286ce7269126fcc9d7a619be347cb69` | `d0b3c01afa7aca34047782da5d2f9138cebd1ba1` | `6178f9d1d347a1ecf013c343b483d212df380ae8` |

开始和结束均通过 GitHub API 核对，两个 PR 仍为 Open，HEAD／base 与上表一致。没有把 #22 当作以 master 为基准的独立 PR，也没有将结论延伸到其他提交。

完整阅读 [Review 048](codex-review-048.md)。保留 B1-R1、B1-R3、N1、N2 已关闭、P1 已确认的结论；不重开未变推导算法、生成器或 M1 审查。

#21 的增量共 6 个文件：

- `f36d121`：归档 Review 048；
- `03b599f`：新增 94 行 review048_test.go，只有回归测试；
- `bb762be`：修改 legality.go 和 criterion_test.go；
- `6178f9d`：更新 implementation-plan 和 review-resolution。

生产推导算法、Random、History、复核器及转换器、Schema、迁移、冻结规范均未修改。没有调整种子、例外验收概率、样本数量或覆盖门槛。

#22 本轮只有合并提交 `d0b3c01`，父提交为 `53a186c` 与 `6178f9d`。共享 derive／factgen 源码与 #21 本轮目标相同；readycheck 及其转换器相对上轮没有差异，没有独立修改或冲突带入的新逻辑。

原工作区开始时干净，HEAD 为 `d0b3c01`，本报告此前不存在。隔离复验和新增校准测试均位于原仓库之外，原仓库只新增本报告。

## 二、BLOCKER：无；B1-R2 可以关闭

### 1. legality.Check

[legality.go](../../internal/testutil/factgen/legality/legality.go) 862～881 行，在取得执行绑定配置的声明列表后，独立检查：

```text
failure_kind == input_missing && len(declared) == 0 → 拒绝
```

这个检查不依赖 head_start 非空、不依赖有其他输入记录，也不依赖调用 CheckTrees。合法普通无输入执行仍可没有 inputs_json；确有声明输入时，原输入记录、可见提交树矛盾及配置绑定检查继续生效。

### 2. legality.CheckTrees

同文件 1027～1058 行先区分配置已知与未知，再检查空声明列表。对已知有效配置，零 inputs 的 input_missing 在原先“直接跳过”的分支中被明确拒绝；HEAD 为空也不能绕过这条前提。

非空声明列表、非空 head_start 的场景继续核对树见证：

- input_missing 必须有至少一个声明路径在树中缺失；
- 普通执行的输入记录为 NULL 时拒绝；
- 即使提供了输入记录，树中缺少任何声明路径仍拒绝；
- 记录中的 mode／type／OID 继续与规范化查找路径对应的见证比较。

未发现这几项新增检查误拒本轮合法正例。它们不修改事实或树见证。

### 3. 两个门禁的证据不同

Check 根据转储中可见记录检查矛盾与可合法补全性；CheckTrees 根据提供的固定提交树见证检查。以下差异合理：

- 只有见证显示某路径存在，而没有其他执行记录时，Check 可以接受该路径缺失的转储投影，因为它没有接收树见证；CheckTrees 应拒绝与见证冲突的 input_missing。
- 对提供了输入记录、但见证缺少路径的转储，Check 可以接受记录内部一致的投影，CheckTrees 必须拒绝。

CheckTrees 的配置表只来自变更中的已知 Check；配置无法绑定时仍由 Check 拒绝。CheckTrees 也不替代 Check 对完整输入记录结构、历史状态、执行分类等其他规则的检查。本轮没有把它当作完整生产写入校验器，也没有要求无见证的 Check 证明完整提交树。

### 4. 独立场景校准

在固定 `d0b3c01` 的隔离副本新增独立矩阵，两个函数分别调用、分别断言结果，不能以合并后的错误集合掩盖单个门禁漏检。实际通过：

| 场景 | Check | CheckTrees |
|---|---|---|
| 未声明 inputs，普通执行，无输入记录 | 接受 | 接受 |
| 未声明 inputs，error/input_missing | 拒绝 | 拒绝 |
| 未声明 inputs、HEAD 为 NULL，仍记录 input_missing | 拒绝 | 拒绝 |
| 声明 a，确实缺失，input_missing，无记录 | 接受 | 接受 |
| 声明 a，普通执行却没有输入记录，见证有 a | 拒绝 | 拒绝 |
| 声明 a，记录 a，但见证缺少 a | 接受投影 | 拒绝 |
| 声明 a，记录及见证均有 a | 接受 | 接受 |
| 声明 a、b，只有 a 存在，input_missing | 接受 | 接受 |
| 声明 a、b，普通执行有两个正确记录及见证 | 接受 | 接受 |
| 声明／记录 ./a，见证键为 a | 接受 | 接受 |
| 声明 ./a，见证有 a，只有见证能否定缺失 | 接受投影 | 拒绝 |

这覆盖了本轮要求区分的五类情况，并保留多输入和规范化正反对照。原同提交一致性、跨 AC、上下文变化、脏工作区及树见证变异测试仍通过。

## 三、决策表样例合法性

[criterion_test.go](../../internal/domain/derive/criterion_test.go) 53～66 行现在按以下顺序构造：

1. 使用原 fixture 的合法 Baseline 和自动化 AC；
2. 通过一次人工编辑更新 Check，声明 `testdata/orders.golden`，产生 material 修改，AC revision 从 1 增至 2；
3. 在 commit A 上记录 error/input_missing；
4. 断言执行绑定 revision=2、最新 check_config_hash，且 inputs_json 为 NULL；
5. 继续经合法域门禁，断言 `failing / check_failed`。

这没有被改成域外用例。Requirement 没有修改，仍为 revision=1，AC 对父需求的适用修订不发生冲突。

审查方另在隔离副本独立重建该场景，额外核对：

- 执行的 RequirementRevision=1；
- 执行配置哈希确实对应修订后的 Check；
- 转储只有这一条执行，没有同提交上证明声明路径存在的其他输入记录；
- 以 commit A 的缺失路径树见证调用 CheckTrees 可以通过；
- Check 同样通过，最终仍为 failing/check_failed。

因此不是只加一个 inputs 字段便假定合法：该历史存在符合“一提交一棵树”的补全，绑定修订、配置及父修订也正确。这里使用合成树见证，不声称读过真实 Git 提交。

## 四、修复前后与本轮实际运行

本机平台：Windows／PowerShell，Go 1.27.1 windows/amd64。使用 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local、-mod=readonly，没有下载依赖或连接真实远端。

### 1. 测试先行反例

在隔离归档 `03b599f` 上执行：

```text
go test -tags nowebui -mod=readonly ./internal/testutil/factgen/legality -run TestInputMissingNeedsADeclaredInput -v -count=1
```

实际 exit 1，失败内容与原漏洞一致：

- 两个零声明输入场景中，Check 都没有找到 input_missing 问题；
- 同两个场景中，CheckTrees 也没有找到问题；
- “声明输入但没有输入记录”的场景，旧 CheckTrees 没有拒绝。

不是因无效配置或其他测试前提失败。其余合法控制没有报错。

在隔离归档 `bb762be` 执行同一测试，实际 exit 0、PASS。该修复到最终 #21 目标之间只有两个文档文件变化，相关源码一致。

最终组合树的完整包测试也通过该回归；两组 CI 的三平台日志都显示该测试实际 PASS，没有 SKIP。由此可以把“旧版失败、新版通过”从作者说明提升为本轮独立复验结论。

### 2. 固定最终源码的受影响包

在固定 `d0b3c01` 组合树执行：

```text
go test -tags nowebui -mod=readonly ./internal/domain/derive ./internal/testutil/factgen/... ./internal/testutil/readycheck/... -count=1
```

| 包 | 本机结果 |
|---|---|
| domain/derive | PASS，2.665s |
| factgen | PASS，0.126s |
| factgen/legality | PASS，1.726s |
| factgen/toderive | PASS，0.197s |
| readycheck | PASS，0.147s |
| readycheck/fromdump | PASS，2.207s |

#21 共享源码与组合树一致；WP2.2 包的结果绑定 #22 固定目标。本轮没有用 #22 的额外测试冒充 #21 已包含复核器。

隔离副本的独立 11 场景门禁矩阵全部 PASS；独立决策表合法补全用例 PASS。没有把这些临时测试移入原仓库。

### 3. 性质、双向对照与覆盖

400 个性质样本、600 个对照样本的门禁、乱序、重放、截止点、ChangesSince、设置不影响 ready、GradeConsistent、已有正反例均随包测试运行通过。生成器、样本范围、概率、门槛和两套算法没有改变。

从最新三平台 CI 日志分别核对到的分布与 Review 048 完全一致：

| 测试／域 | 非 ready | ready observed | ready none | ready snapshot_isolated | 合计 |
|---|---:|---:|---:|---:|---:|
| WP2.1／MVP | 198 | 61 | 7 | 0 | 266 |
| WP2.1／通用 | 87 | 14 | 4 | 29 | 134 |
| WP2.2／MVP | 209 | 79 | 12 | 0 | 300 |
| WP2.2／通用 | 194 | 29 | 12 | 65 | 300 |

WP2.2 非 ready 失败归因也未变化：

| 域 | R1 | R2 | R3 | R4 | R5 | R6 | R7 | R8 | R9 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| MVP | 5 | 18 | 121 | 35 | 97 | 105 | 7 | 64 | 29 |
| 通用 | 5 | 16 | 118 | 31 | 84 | 99 | 12 | 62 | 25 |

上述具体分布来自本轮核查的 CI 日志；本机完整包测试另外证明这些测试在固定源码上通过，不混称本机重新统计了所有日志计数。

MVP 仍只有观测级执行；通用域的合成隔离级只用于纯函数用例。统计数量不是生产分布，也不是对全部合法历史的穷尽证明。

## 五、CI 身份与适用性

独立读取两次运行、每次 8 个 Job 元数据和共 16 份日志：

| CI | workflow HEAD | Attempt／最终结果 | 实际检出合并快照 | 父提交 | 内容树 |
|---|---|---|---|---|---|
| [#21 run 39104694681](https://github.com/XKHoshizora/specmetry/actions/runs/39104694681) | `6178f9d` | 1，8／8 SUCCESS | `e0fd1b0df8b38478c7f1dba3748d7aa1b577ff3a` | `666efe2`、`6178f9d` | `ea136089c4ce4aa0596967c0f531a7b95f9bb2c1` |
| [#22 run 39104720866](https://github.com/XKHoshizora/specmetry/actions/runs/39104720866) | `d0b3c01` | 1，8／8 SUCCESS | `77d6b14a355af4a7e03c580c1cb20e325c3ab3b3` | `6178f9d`、`d0b3c01` | `d799ff204e2300f2f31d729b7f0b29784f9da83b` |

两棵 CI 合并快照树分别与固定目标 HEAD 相同。所有 Job 都检出了对应快照；八类 Job 为 Web、Go lint、三个平台 Go tests、三个平台 embedded-dashboard Build。

Windows、Ubuntu、macOS 均实际运行并通过新的 TestInputMissingNeedsADeclaredInput、原提交树／级联控制和推导包测试；#22 也实际运行了 600 个对照。没有用测试名称或运行成功代替断言核查。

Ubuntu 的测试步骤在 Linux 分支设置 -race，执行 Go 测试并成功。本机没有运行 Linux Race Detector，也没有独立本机复验 macOS。Windows／macOS 日志含同一条件脚本，不代表它们也使用了 -race。

作者本地证据与本轮证据的区分：

- 作者的本地修复与测试说明见 §60、PR 描述；不据此宣称全部已独立验证。
- 本轮独立复验了 03b599f／bb762be 对照、最终包测试、两个门禁矩阵和合法树补全。
- 跨平台与 Race Detector 结论来自核查过身份及日志的 GitHub CI。
- 原始桩阶段、真实 Git／SQLite／CLI 等不在本轮重跑范围。

保留既有失败／取消记录：旧 run 39066356194 的 lint 失败与 Windows 取消、39071770560／39071848979 的共享 unparam 失败均未被此次成功覆盖或改写。本轮只说两个最终固定 HEAD 的 Attempt 1 成功，不说两个 PR 全过程从未失败。

## 六、归档与状态登记

Review 048 在归档提交 f36d121 与本轮目标中的 blob 相同：

```text
eb94560451f1ad3590f19a266627ba752a7f1a07
```

其审查身份仍为 #21 ca2aec2、#22 53a186c，旧 NO-GO 没有改写成针对当前提交的结论。

[review-resolution §59／§60](../review-resolution.md) 正确记录：

- R1、R3 根据 Review 048 关闭；
- R2 的原树矛盾和随机生成修复有效，仅剩零 inputs 残余；
- N1／N2 保持关闭，P1 保持已确认；
- 新修复标为“已修订，待复审”，明确没有自行宣布 B1 关闭或工作包验收；
- [implementation-plan](../implementation-plan.md) 仍为等待复审、工作包尚未验收；
- 历史分布、维护事项、其他未批准草案和只授权 WP2.1／WP2.2 的边界均保持。

本轮 GO 是新的独立审查结论，尚需负责人决定合并与工作包验收；不会倒改这些历史记录。

## 七、分类与最终回答

- **BLOCKER：无。** Review 048 最后一处 B1-R2 已关闭，连同已关闭的其他部分，Review 046 的 B1 可以整体关闭。
- **NON-BLOCKER：无本轮新增事项。** 既有维护项继续独立保留，不因 CI 成功自动关闭。
- **PRODUCT DECISION：无新增。** P1 已确认，不要求再次批准；本轮不批准其他产品草案。
- **ACCEPTED RISK：无新增。** 只保留已批准的 MVP observed／通用合成隔离级区分及畸形输入选项 (a)。转储与树见证的证明边界不等于接受新的安全风险。

最终逐项裁定：

1. **最后一处 B1-R2 关闭，B1 整体关闭。**
2. **PR #21：GO；PR #22：GO。** #22 仍以先合并 #21、再核验改基后的内容及 CI 为合并条件。
3. **两个工作包纯函数层验收证据充分。** 该判断结合此前有效审查、本轮窄范围修复验证、合法域正反例及双向证据，不仅依赖两个算法一致或 CI 绿色。
4. **未发现新的直接回归或确定算法反例。** 本轮未修改生产推导算法，也未发现需要扩大审查的具体证据。
5. **可以建议负责人按顺序安排合并与验收核查；本报告不代替负责人宣布验收。**

CLI、实际数据库读快照／高水位、Git／磁盘读取、生产写入前置条件与并发、Schema 输出、SQLite 复核器前端和 WP2.9 E2E 仍由后续工作包验证。本轮纯函数 GO 不等于这些集成条件通过，更不等于 M2 整体验收。

## 八、合并顺序与下一步交接

建议接手角色为产品负责人：

1. 先合并 #21，核对实际合并提交、父提交以及 derive／factgen／legality 与已审 6178f9d 的内容一致；核查合并后 master CI。
2. 再将 #22 base 改为实际 master。核对差异只剩已审 WP2.2 内容，没有重复或丢失共享修复；readycheck／fromdump、共享测试前端与转换器没有未审变更或冲突改写。
3. 核对改基后的 HEAD、实际检出合并快照、树适用性及完整 CI，包含 Ubuntu Race Detector。仅因文档、提交父链或合并方式不同不能机械要求整棵树 SHA 与旧 HEAD 一样；应检查相关路径的等价内容。
4. 若只是无冲突、内容等价的改基，做内容和 CI 核验即可，不重新开启 M1 或整个 M2 审查；如有实际新逻辑，只审其直接影响。
5. 由负责人单独登记工作包验收决定，再决定后续授权。本轮不授权或启动 WP2.3。

**审查完成后暂停。原仓库仅新增本报告；未提交、推送、合并，未宣布工作包验收，未启动 WP2.3。**

