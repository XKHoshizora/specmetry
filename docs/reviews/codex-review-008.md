# Specmetry M0 最终定向代码复审意见（008）

审查日期：2026-10-14

审查 HEAD / 修复提交：`d212524`（完整提交：`d212524a477ba69cb7763b94b6949d4f09de043b`）。比较基准：父提交 `8e8a7a9`。上一轮报告：[codex-review-007.md](codex-review-007.md)。

**代码审查结论：GO。B2 可以正式关闭，未发现新的可复现 BLOCKER。** `d212524` 满足本轮已批准的 M0 代码审查边界，可以推送开发分支并启动三平台 CI。可以保留现有实现，不需要存储层重构。

**M0 完整验收尚未通过。** 三平台 GitHub CI、macOS 实测、Linux Race Detector、Windows 真实双用户测试仍未全部完成。代码审查 GO 不代表这些验收已经通过。

所有执行测试、修复前后对照和独立补测均在原仓库之外进行。原有代码、测试、设计文档均未修改；本轮仅按用户要求新增本报告，未提交、未推送。

## 审查范围与依据

本轮只复核 `d212524` 对 Windows 下连续外部 SQLite 关闭竞态的 B2 修复，阅读本次变更涉及的代码、测试、处理记录和最新版 [MVP-STO-07](../acceptance-criteria.md#L269)，并检查相关公开调用链及最小回归。

B1、B3、B4 沿用已经独立审查关闭的结论，本轮只检查此次变更是否引入明显回归。N1 继续在 M2 处理；PD-23 恢复功能不属于本轮 M0 阻塞条件。没有重新评审产品架构，没有要求提前实现后续里程碑。

继续接受产品负责人批准的方案 A：首次私有空协调文件，以及外部关闭期间原生只读连接创建必要的私有空 WAL/SHM，属于受限例外；数据完整性、版本保护、无法可靠判断时的保守处理和无外部竞争时的文件不变性要求保持不变。本次提交未修改 MVP-STO-02 或 MVP-STO-07 的验收文本。

## BLOCKER

**无。B2 可以正式关闭。**

### B2 修复的代码判断

**位置**：[readonly.go:135](../../internal/store/sqlite/readonly.go#L135)、[readonly.go:145](../../internal/store/sqlite/readonly.go#L145)、[inspect.go:36](../../internal/store/sqlite/inspect.go#L36)、[doctor.go:139](../../internal/cli/doctor.go#L139)。

`readMetaReadOnly` 将参数名从 `sameSizes` 改为 `diagnostic`，新增规则：WAL、SHM 均存在且诊断发现 SHM 为零字节时，直接返回 `uninspectableError`。该返回发生在调度钩子和 `queryReadOnly` 之前，因此不会建立 SQLite 连接。

`Inspect` 将此结果映射为带 `Limitation` 的 Unknown，版本字段保持未知；公开 `doctor` 优先处理 `Limitation`，输出数据库警告，不会落入 `[ok] database schema v1` 分支。

普通 Open 传入 `diagnostic=false`，不适用新增空 SHM 规则。它的原生只读版本检查、后续读写连接复核和写事务内版本保护保持原样。测试专用导出 [export_test.go](../../internal/store/sqlite/export_test.go) 仅编译进入测试，没有向生产 API 新增导出入口。

### 独立验证结果

- 上一轮连续关闭反例不再输出错误的数据库 OK。
- Windows 第一次真实外部关闭之后自然产生空 WAL/SHM；从这一状态反复调用公开 Inspect 和 doctor，均返回 Unknown，不进入 SQLite 读取路径，目录保持不变。
- Windows 正常重新打开并关闭数据库后，SHM 恢复为非空状态；公开 Inspect 和 doctor 重新报告正确版本，诊断前后文件不变。
- 正常、稳定的 Specmetry 数据库仍能报告版本。稳定但 SHM 为空的有效数据库会保守返回 Unknown，没有被判断为损坏或版本不兼容。
- 未发现新增文件副作用、数据完整性问题或版本保护回归。

空 SHM 的保守处理是有意的诊断限制；本轮已经验证正常打开后的恢复行为，不构成正常数据库永久无法诊断的问题。

## 新增回归测试的独立检查

### 前置条件与第二轮实际证明的内容

**位置**：[TestInspectRepeatedExternalCloseReportsUnknown](../../internal/store/sqlite/externalclose_test.go#L277)、[TestDoctorRepeatedExternalCloseReportsUnknown](../../internal/store/sqlite/doctor_race_test.go#L30)。

第一轮使用真实 SQLite 子进程，在文件状态判定之后、只读连接建立之前完成普通关闭。测试确认 WAL、SHM 实际被删除；Inspect 测试还在 Windows 上明确断言第一轮自然留下的两个辅助文件均为零字节。前置状态不是手工截断或模拟 VFS 生成的。

修复后的第二轮调度钩子没有执行，外部关闭也没有发生：诊断在读取前保守返回 Unknown，避免再次进入原有漏洞路径。因此第二轮证明的是**读取前保守拒绝**，不能描述为“成功检测到了第二次删除和重建竞态”。

Inspect 测试在没有执行外部关闭时要求整个目录不变，并逐轮核验权限、schema、最低版本、seq 和事实。公开 doctor 测试验证 Unknown 输出、不出现版本文本以及主数据库字节不变。两种公开入口均得到了实际验证。

本轮仓库外独立补测进一步确认：连续三轮空 SHM 诊断均没有到达读取钩子，公开 doctor 同样返回 Unknown，目录完全不变；正常 Open / Close 之后，两个入口均恢复正确版本报告且诊断不修改文件。

### 修复前失败、修复后通过的对照可信

在独立副本中保留本次新增测试，只将读取器恢复为父提交 `8e8a7a9` 的实现：

| 测试 | 父提交读取器 | 修复后读取器 |
|---|---|---|
| `TestInspectRepeatedExternalCloseReportsUnknown` | 第二轮外部关闭实际发生，返回 schema v2、Limitation 为空，测试失败 | 第一轮真实竞态返回 Unknown；第二轮读取前返回 Unknown，目录不变，测试通过 |
| `TestDoctorRepeatedExternalCloseReportsUnknown` | 第二轮外部关闭实际发生，输出 `[ok] database schema v1`，测试失败 | 两轮均输出数据库 Unknown；第二轮没有进入读取路径，测试通过 |

修复前公开 doctor 的关键输出：

```text
cycle 1 (outside close during the check: true): exit 0
[ok]   database        schema v1
FAIL: TestDoctorRepeatedExternalCloseReportsUnknown
```

修复后两项测试的执行记录均为：第一轮外部关闭 `true`，第二轮 `false`。失败与通过的差异符合新增空 SHM 分支的实际控制流，没有依赖测试前置条件缺失而假通过。

### 正常无竞争与重新打开

原有关闭、崩溃 WAL、活动数据库和恢复备份的无竞争诊断测试继续通过。新增 Inspect 测试在 Windows 正常重开后重新报告 schema v2，文件不变；本轮独立补测通过公开 doctor 确认正常重开后恢复 `[ok] database schema v1`。

Linux 第一次竞态只留下空 WAL、没有 SHM；第二轮按“只有其一”的既有规则直接报告 Unknown，普通 Open 继续按 PD-23 状态拒绝。这是已批准的可用性限制，不能将 Windows 的恢复成功表述成 Linux 也已恢复成功。

## NON-BLOCKER

N1 按既定计划在 M2 的 purge 验收前解决，仍不能标记为完全关闭。本轮没有发现新的必须修改的非阻塞代码问题。

上一轮对兼容打开成功分支的测试覆盖意见已合理处理：[externalclose_test.go:194](../../internal/store/sqlite/externalclose_test.go#L194) 起增加主数据库文件不变、schema / 最低版本 / seq / 事实、权限、锁释放和下一次 seq 的断言，实际回归通过。

## ACCEPTED RISK

### 同大小比较不能完整记录外部活动

针对 [review-resolution.md:707](../review-resolution.md#L707) 提及的同大小重建，本轮使用两个真实外部 SQLite 进程进行了独立验证：第一个进程普通关闭并删除辅助文件，第二个进程通过 SQLite 原生连接和持久 WAL 正常重建辅助文件，再完成关闭；读取前后的文件存在情况、大小相同。

这种**外部活动漏检可以复现**，不只是理论描述。但 Windows、Linux 上均读到了重建后的真实已提交版本，诊断自身没有创建或修改文件。将最低二进制版本分别设为 `9.9.9` 和无法解析的 `next` 后，诊断返回实际已提交值，普通 Open 均拒绝且文件不变。

回答本轮要求的三个问题：

1. 可以复现外部活动漏检；本轮没有复现错误的数据库版本或健康状态报告。
2. 未复现未提交数据读取或错误接受不兼容版本；原生事务读取和版本拒绝回归通过。
3. 在这一复现中，辅助文件由第二个外部进程重建，诊断连接没有重建空辅助文件；按当前非破坏性诊断边界，未发现实质违约。当前实现没有提供完整捕获外部活动历史的保证，不需要因此重构存储层。

本判断限定于实际验证的原生 SQLite 操作，没有将它扩展为对任意外部文件篡改的安全证明。

### 保留的其他边界

- 稳定但 SHM 为空的有效数据库可以保守报告 Unknown；正常 Specmetry 打开后可恢复诊断，符合本轮修复要求。
- 方案 A 的有限文件副作用例外保持不变，没有扩大为一般性写入、checkpoint、删除有效 WAL 或版本绕过。
- Linux 缺少 SHM 的状态继续遵循 PD-23；恢复功能仍安排在 M2、M4 自测前完成，不阻塞当前 M0。
- 平台未验证项继续保留，不能因本轮代码审查 GO 而提前通过。

## 独立测试与最小回归结果

| 检查 | 本轮结果 |
|---|---|
| Windows 当前提交存储、CLI、datadir 测试，`nowebui`、`count=1` | 通过；两个符号链接测试因缺少所需权限跳过 |
| Linux 同三包测试 | 在 WSL 原生 Linux 临时文件系统实际执行通过，没有把交叉编译当作执行通过 |
| 新增 Inspect 与公开 doctor 测试的修复前后对照 | 父提交读取器预期失败，修复后通过，结果可信 |
| Windows 空 SHM 的重复读取前拒绝、目录不变及正常重开后的两个公开诊断 | 仓库外独立补测通过 |
| 无竞争场景的正确版本报告与文件不变性 | 原有回归通过 |
| 两个真实外部进程同大小重建后的已提交读取及最低版本拒绝 | Windows、Linux 独立验证通过；存在外部活动漏检，未发现错误版本接受 |
| 未提交回滚日志 / WAL 写入的诊断 | 原有测试通过，未发现报告未提交版本 |
| B1 权限与路径保护相关回归 | Windows 已执行的 ACL 测试、Linux 实际跨 UID 测试通过；未重新全面审查 B1 |
| B3 打开协调、升级后旧写事务拒绝及最低版本保护 | 原有回归通过，未发现本次变更导致的明显回归 |
| B4 append-only，含 REPLACE、UPSERT、冲突及新连接保护 | 原有回归通过 |
| 迁移、一致性备份、失败回滚、全局 seq | 原有回归通过 |
| GitHub 三平台 CI、macOS 实测、Linux Race Detector、Windows 真实双用户测试 | **未全部完成，本轮未验证** |
| Windows `TestOpenRefusesSymlinkedDatabase`、`TestCheckReportsSymlinkEntry` | **因所需符号链接权限不足而跳过，不能算作验证通过** |
| 前端构建与嵌入 | 本轮未重新执行，仍需 CI 验证 |

## 最终结论

| 问题 | 结论 |
|---|---|
| B2 是否可以正式关闭？ | **是**。上一轮已复现的错误 OK 路径已被读取前保守拒绝关闭，两种公开入口及恢复路径均独立验证 |
| 是否存在新的、可复现的 BLOCKER？ | **未发现** |
| 当前提交是否满足已批准的 M0 代码验收要求？ | **是，代码审查 GO**；结论结合此前已关闭的 B1、B3、B4，并限定于本轮针对性复审 |
| 是否可以推送 GitHub 开发分支、开始三平台 CI？ | **可以**。现有 workflow 对 push 只匹配 master，开发分支应通过 PR 触发 CI。本轮未执行推送或 GitHub 操作 |
| 是否需要重新实现 SQLite 存储层？ | **不需要**。可以保留现有实现，不提出无必要重构 |
| M0 完整验收是否通过？ | **尚未通过**。三平台 GitHub CI、macOS 实测、Linux Race Detector、Windows 真实双用户测试及明确跳过的平台测试仍需独立完成 |

**最终 M0 代码审查：GO。MVP-PLT-01 和其他未验证的平台验收不能提前标记通过。**
