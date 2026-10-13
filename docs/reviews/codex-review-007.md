# Specmetry M0 最终定向代码复审意见（007）

审查日期：2026-10-13

审查 HEAD：`8e8a7a9`。上一轮报告：[codex-review-006.md](codex-review-006.md)。主要检查 `5294f9a..8e8a7a9` 的修订，其中 `d3274c9` 调整验收边界并加入回归测试，`8e8a7a9` 将外部关闭例外扩展到普通 Open 的版本检查。

**结论：NO-GO。B3 可以关闭；B2 仍有一项实际违反最新版 MVP-STO-07 的诊断缺陷。** 可以保留现有存储实现，只需局部修复，不需要底层重构。

本轮接受产品负责人正式批准的方案 A，没有继续用修订前的绝对目录零变化标准否定已批准例外。所有执行测试和独立反例都在原仓库之外进行；原有代码、测试和设计文档均未修改。本轮仅按用户要求新增本审查文件，未提交、未推送。

## 审查范围与当前契约

本轮只检查 M0 的 B2、B3、最新测试与相关调用链，对 B1、B4 及迁移、备份、回滚、全局 seq 做相关回归检查。没有重新讨论冻结的产品架构，没有扩大到 M1/M2 功能。

当前 [MVP-STO-02](../acceptance-criteria.md#L264)、[MVP-STO-07](../acceptance-criteria.md#L269)、[architecture §11.2](../architecture.md#112-数据库迁移) 与 [§11.5](../architecture.md#115-只读检查) 的有限调整，按以下边界评估：

1. 不兼容数据库必须拒绝打开，不允许执行不兼容写入。
2. schema、业务数据、事实、seq 和已提交内容不能被错误修改；拒绝路径不能改写或删除已有数据库、WAL、SHM，不能执行 checkpoint。
3. 允许首次创建空、私有的 `.migrate.lock`，拒绝后必须释放。
4. 仅在外部 SQLite 进程于版本预检期间关闭数据库并删除辅助文件时，允许原生只读连接创建必要的空、私有 WAL/SHM。
5. 外部进程造成的文件变化与 Specmetry 自身副作用分别判断；上述例外不扩大为一般性文件写入、删除有效 WAL 或绕过版本检查。
6. 无法可靠判断版本时，Open 必须拒绝。doctor 在上述外部关闭竞态中必须报告 Unknown，不报告版本或正常。
7. 没有外部竞争操作时，继续遵守文件不变要求；协调文件首次创建按例外单独处理。

文档之间对这两类例外的当前规定一致。实现仍存在下述 doctor 降级遗漏。

## BLOCKER

### B2：Windows 上重复外部关闭竞态会漏报 Unknown

**位置**：[readonly.go:66](../../internal/store/sqlite/readonly.go#L66)、[readonly.go:124](../../internal/store/sqlite/readonly.go#L124)、[readonly.go:140](../../internal/store/sqlite/readonly.go#L140)、[doctor.go:139](../../internal/cli/doctor.go#L139)、[doctor.go:148](../../internal/cli/doctor.go#L148)。

`fileState` 只记录文件存在情况和大小。`readMetaReadOnly(..., sameSizes=true)` 比较读取前后的存在情况与大小，不能识别同名空文件在两次观察之间被删除、重新创建的情况。

**独立反例：Windows 已通过公开 Inspect 和 doctor 调用复现。**

1. 通过正常 `Open` / `Close` 生成数据库，留下持久 WAL/SHM；SHM 初始大小为 32768 字节。
2. 第一次诊断判断文件状态之后、建立只读连接之前，真实独立子进程用普通 SQLite 连接读取并关闭数据库；确认 WAL、SHM 均已被删除。
3. 诊断继续。Windows 的原生只读连接重新创建两个空文件；由于 SHM 从 32768 字节变为 0，第一次正确报告 Unknown。
4. 直接沿用第一次诊断留下的两个空文件，再通过公开调用制造完全相同的外部关闭竞态。
5. 外部子进程再次实际删除两个文件，诊断再次创建两个空文件。读取前后文件存在情况相同、大小均为 0，检测遗漏。
6. `Inspect` 返回确定版本且 `Limitation` 为空；使用内置 schema v1 数据库的公开 `doctor` 命令输出 `[ok] database schema v1`，没有按契约返回 Unknown。

公开 doctor 独立测试的关键结果：

```text
cycle=0, SHM before=32768, exit=0
[warn] database        version unknown: the database files changed while they were being read; run the command again

cycle=1, SHM before=0, exit=0
[ok]   database        schema v1

FAIL: doctor did not report Unknown after the real external close
```

前置状态由第一次诊断自然产生，没有手工截断 SHM，也没有模拟 SQLite、文件删除或平台权限。调度使用现有测试钩子；公开 CLI 验证仅在仓库外副本中暴露了该钩子，没有修改生产读取行为。两轮均确认子进程完成关闭并实际删除辅助文件；主数据库未变，权限检查通过。

Linux 首轮后没有 SHM，不能直接进入 Windows 的这一重复状态；本项没有被表述为 Linux 也已复现。

**影响**：违反最新 [MVP-STO-07](../acceptance-criteria.md#L269) 和 [architecture.md:521](../architecture.md#L521) 中“此时必须报告版本未知”的明确要求。本反例中主数据库和已提交内容保持完整，没有发现未提交数据读取、错误版本接受或数据损坏。

**阻塞依据**：仅为当前正式批准契约中的诊断降级要求。新建空、私有 WAL/SHM 本身符合方案 A，不是阻塞原因，也没有重新要求绝对目录零变化。

**建议修改方案**：对空 SHM 的诊断保守返回 Unknown，或者增加能够识别删除、重建的文件身份检查；补充连续两次竞态的公开调用回归测试。若保守判断在建立连接之前直接返回 Unknown，则同时验证它不产生文件副作用。该修复可以局部完成，不需要新 VFS 或自实现 WAL 解析器。

**关闭条件**：上述可自然到达的空 WAL/SHM 状态下，公开 Inspect / doctor 不再在外部关闭竞态中报告版本或正常；原有已提交数据、文件权限和有限副作用断言继续通过。

## NON-BLOCKER

### 兼容打开成功分支的断言未覆盖处理记录宣称的全部内容

**位置**：[externalclose_test.go:174](../../internal/store/sqlite/externalclose_test.go#L174)、[externalclose_test.go:187](../../internal/store/sqlite/externalclose_test.go#L187)、[review-resolution.md §9](../review-resolution.md#9-m0-第三次代码审查codex-review-006的处理)。

`TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles` 在拒绝分支调用完整副作用复核；成功分支只断言 schema v2 和两条事实，未断言处理记录宣称的全部 seq、`min_binary_version`、权限与锁释放结果。

**影响**：测试覆盖说明比实际成功分支断言更完整，降低后续回归的判别力。本轮没有发现成功打开的数据错误，不构成额外 BLOCKER。

**建议**：成功分支补齐最低版本、seq、事实内容、权限和锁释放断言。本轮仓库外独立补测已确认 Windows 兼容打开保持主库字节、版本、seq、事实内容和完整性，权限合格、锁已释放，后续正常写入取得 seq 3。Linux 在无法确认版本时正确拒绝，没有把 Unknown 当作兼容。

### N1 墓碑整理保持部分修复，允许延期

N1 仍允许延期至 M2 的 purge 验收，不能标记为完全关闭。本轮未要求提前实现该项，也未将其作为 M0 阻塞理由。

## ACCEPTED RISK

- 首次创建空、私有且拒绝后释放的 `.migrate.lock`，符合批准后的 MVP-STO-02。
- 外部关闭期间创建必要的空、私有 WAL/SHM，符合方案 A；原生只读 WAL 连接允许创建辅助文件，也符合 [SQLite 官方只读 WAL 规则](https://www.sqlite.org/wal.html#read_only_databases)。本轮未把这种批准行为本身当作缺陷。
- Linux 在该竞态中留下空 WAL、返回 Unknown 并拒绝普通打开，属于已批准的可用性限制；Windows 在普通 Open 路径读到可靠的已提交版本后按版本判断，随后读写连接再检查，不要求将这一 Open 行为改为无条件拒绝。
- PD-23 恢复命令安排在 M2，必须在 M4 自测前完成，当前 M0 不实现该命令不构成 BLOCKER。
- 已打开的旧连接在写入被拒绝后关闭，可能合并新进程已经提交的 WAL；本轮回归确认升级后的库完整。这与不兼容 Open 的拒绝路径分别评估，沿用已有批准边界。
- GitHub 三平台 CI、macOS 实测、Linux Race Detector 和 Windows 真实双用户验证尚未完成，不能宣称平台完整验收通过。

## 新增测试的独立检查

| 测试 | 独立判断 |
|---|---|
| `TestIncompatibleRefusalMayOnlyCreateCoordinationFile` | 走公开 Open 路径；明确确认初始锁文件不存在；覆盖新 schema 和最低二进制版本两种拒绝原因；验证只新增空协调文件、权限合格、锁已释放、数据库内容不变；协调文件已存在时再次拒绝要求目录不变。Windows、Linux 实际通过 |
| `TestInspectDuringExternalCloseReportsUnknown` | 走公开 Inspect 路径；真实子进程在文件状态判定后完成普通关闭，并断言辅助文件已删除；以外部关闭完成后的快照归因 Specmetry 副作用。首次竞态 Windows、Linux 实际通过，但遗漏第一次诊断留下空 SHM 后的重复竞态 |
| `TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles` | 不兼容与兼容分支均走公开 Open；使用真实外部进程完成关闭，不依赖固定等待。Windows 不兼容版本被拒绝、兼容版本正常打开；Linux 两者均按无法确认拒绝。成功分支断言不足见 NON-BLOCKER |
| `TestOpenRefusesWhenVersionIsUnknown` | 无外部竞态时，普通关闭导致辅助文件缺失、以及仅剩 WAL 两种状态均拒绝，文件保持不变。Windows、Linux 实际通过 |
| `TestOpenAndConcurrentUpgradeDoNotInterleave` | 升级子进程在首次实际获取 OS 锁失败后发出握手，证明它到达并等待打开锁；不再依赖固定两秒等待。升级后旧 WriteTx 回调拒绝执行，事实与 seq 不变，旧连接关闭后升级库完整。Windows、Linux 实际通过 |

新增测试确实覆盖公开调用路径，实际制造了预期竞态，未发现前置条件不成立导致的假通过。Windows 和 Linux 的首次辅助文件变化均属于批准范围，权限检查使用实际平台实现，没有模拟为已通过。

## 独立测试与回归结果

| 检查 | 本轮独立结果 |
|---|---|
| Windows 当前提交全量 Go 测试，`nowebui`、`count=1` | 通过；先在未加入独立反例的当前提交副本上执行 |
| Linux 存储、datadir、CLI 测试 | 在 WSL 原生 Linux 临时文件系统实际执行通过，没有把交叉编译当作执行通过 |
| Linux 真实跨 UID 路径测试 | 四项实际通过；测试宿主以 root 运行，用于切换到两个非特权用户 |
| 新增协调文件、首次诊断竞态、普通 Open 竞态、Unknown 拒绝测试 | Windows、Linux 通过 |
| 并发升级的锁等待握手、升级后旧写事务拒绝 | Windows、Linux 通过 |
| 无外部竞争时，不兼容备份和崩溃 WAL 的文件不变性 | 原有测试 Windows、Linux 通过 |
| 迁移、一致性备份、失败回滚、备份保留 | 原有存储测试 Windows、Linux 通过 |
| 全局 seq 的连接和多进程并发 | 原有存储测试 Windows、Linux 通过 |
| B4 append-only，含 REPLACE、UPSERT、主键及 UNIQUE 冲突和新连接 | 原有回归 Windows、Linux 通过，未发现本次变更引入的明显回归 |
| 未提交回滚日志 / WAL 写入的诊断 | 原有测试 Windows、Linux 通过，未发现报告未提交版本 |
| 独立最低版本竞态：最低版本为 `9.9.9`、无法解析的 `next` | Windows 按最低版本拒绝；Linux 按无法确认拒绝；拒绝后数据、权限及锁状态符合契约 |
| 独立兼容竞态的完整数据检查 | Windows 保留主库字节、版本、seq、事实内容和完整性，权限合格、锁释放、后续写入正常；Linux 正确拒绝无法确认的版本 |
| 连续外部关闭后的诊断降级 | **Windows 独立反例失败**，公开 Inspect 和 doctor 均复现；Linux 首轮后缺少 SHM，不能进入同一重复状态 |
| GitHub 三平台 CI、macOS 实测、Linux Race Detector、Windows 真实双用户验证 | **未验证** |
| 前端构建与嵌入 | 本轮未重新执行，仍需 CI 验证 |

原有测试和新增首次竞态测试通过，与独立重复竞态反例失败同时成立。不能因为现有套件通过，就将 B2 整体关闭。

## 最终回答

| 问题 | 结论 |
|---|---|
| 1. B2 是否关闭？ | **未完全关闭**。原生锁、事务隔离和首次竞态处理通过；必须修复上述 Unknown 漏报 |
| 2. B3 是否关闭？ | **可以关闭**。批准范围内的文件副作用、版本预检、打开协调、写事务复核和旧连接关闭回归均通过 |
| 3. 是否存在新的实际数据损坏、安全漏洞或错误版本接受问题？ | 本轮未发现；发现一项诊断输出契约缺陷，没有将其夸大为数据损坏或版本绕过 |
| 4. 最新代码与批准验收契约是否一致？ | 文档对方案 A 的两类例外一致；STO-02 实现符合，STO-07 尚有上述遗漏 |
| 5. 是否具备推送开发分支、运行 GitHub CI 的条件？ | **可以**。就本次增量，未发现新增凭据或客户数据；仍需修复阻塞项。现有 workflow 对 push 只匹配 master，开发分支应通过 PR 触发 CI。本轮没有执行任何推送或 GitHub 操作 |
| 6. 最终 M0 代码审查结论？ | **NO-GO，仅剩上述一项 BLOCKER**。可以保留现有实现，局部收紧诊断条件，不建议无必要的存储层重构 |

**代码审查 GO 与 M0 完整验收通过是两回事。** 当前代码审查为 NO-GO；即使局部修复后得到 GO，GitHub 三平台 CI、macOS 实测和 Linux Race Detector 等未验证项仍必须独立完成，MVP-PLT-01 不能提前标记通过。
