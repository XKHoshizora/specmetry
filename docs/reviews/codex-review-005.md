# Specmetry M0 第二次独立代码审查意见（005）

审查日期：2026-10-13  
审查提交：`9fcda14`（相对于父提交）  
审查范围：M0 第一次代码审查 B1～B4、N1～N4 的修复及相关调用链、验收标准。

**结论：NO-GO。B4 可以关闭；B1、B2、B3 尚未真正关闭。** 本次确认了实际权限绕过、读取未提交数据，以及拒绝打开后仍改变数据库文件的反例。

审查以 `9fcda14` 相对父提交的变更为重点。原仓库未修改，未提交或推送。竞态测试仅在仓库外临时副本中加入调度暂停点，权限、事务、迁移和文件操作均使用真实系统行为。

## BLOCKER

### B1：权限检查仍可被路径替换竞态绕过

位置：[datadir.go:150](../../internal/datadir/datadir.go#L150)、[store.go:95](../../internal/store/sqlite/store.go#L95)、[store.go:119](../../internal/store/sqlite/store.go#L119)。

已有不安全目录、文件、WAL 的拒绝，以及静态符号链接、目录联接检查已经有效。但检查只覆盖数据目录及内部条目，后续仍按路径重新访问，没有保护可被其他用户改写的父目录。

**Linux 独立复现：**

1. Specmetry 以普通用户 UID 1000 运行，数据目录属主和 `0700` 权限正确。
2. 在检查完成后，另一普通用户 UID 65534 利用可写父目录，将数据目录改名并替换为符号链接。
3. `Open` 成功，继续向攻击者准备的数据库写入。
4. UID 65534 成功读取写入的 canary。

这是不同操作系统用户之间的数据暴露，不属于已接受的“同一用户可绕过保护”边界。

**建议：** MVP 可以先拒绝路径可被其他用户替换的数据目录位置。需要支持这些位置时，应绑定实际目录、文件身份，并采用禁止跟随链接的打开方式，使检查对象与写入对象一致。补充上述跨 UID 反例。

**关闭状态：未关闭。** Windows ACL 继承及已有文件检查通过，但路径竞态仍影响整体保证。

### B2：自实现快照读取器可能成功返回未提交数据

位置：[snapshot.go:122](https://github.com/XKHoshizora/specmetry/blob/9fcda14/internal/store/sqlite/snapshot.go#L122)、[snapshot.go:162](https://github.com/XKHoshizora/specmetry/blob/9fcda14/internal/store/sqlite/snapshot.go#L162)、[snapshot.go:400](https://github.com/XKHoshizora/specmetry/blob/9fcda14/internal/store/sqlite/snapshot.go#L400)。

回滚日志只在读取开始时检查一次。之后不持有 SQLite 读取锁，读后验证只检查所读页面、文件大小和 WAL，无法识别读取期间开始、仍未提交的回滚事务。

**Windows、Linux 均独立复现：**

- 读取开始时没有回滚日志。
- 真实 SQLite 事务开始修改 `meta`，并通过缓存溢出将未提交页面写入主库。
- 文件大小保持不变。
- 快照读取成功返回 `schema_version=9`，错误为 `nil`。
- 事务回滚后，SQLite 读取的已提交版本仍为 `1`。

因此，“文件没有被读取器修改”已经得到改善，但“读取的是某一时刻已提交的一致状态”尚未成立。SQLite 正常读取依靠锁处理这类并发，页面复读不能替代这一保证。[SQLite 并发规则](https://www.sqlite.org/lockingv3.html)。

WAL 的校验和、Salt 和最后提交边界处理，在已审查的静态恢复分支中符合规范；损坏、截断、未提交帧和旧代残留帧测试也使用 SQLite 恢复副本作为独立参照。这些测试有价值，但不能证明整个无锁快照方案正确。[SQLite WAL 格式](https://www.sqlite.org/fileformat2.html#walformat)。

**建议：** 当前方案不宜继续作为 MVP 的可信读取基础。优先让 `doctor` 在不能安全读取时报告明确限制、版本未知；写入预检另外建立可靠的一致性边界。若保留自实现读取器，必须先修复此反例，并补充真实并发事务测试。

**关闭状态：未关闭。** 这是已复现的正确性问题，而非单纯的复杂度偏好。

### B3：打开阶段的残余竞态违反当前 `MVP-STO-02`

位置：[store.go:99](../../internal/store/sqlite/store.go#L99)、[store.go:122](../../internal/store/sqlite/store.go#L122)、[store.go:131](../../internal/store/sqlite/store.go#L131)。

静态不兼容库的初始化前检查已经有效；`WriteTx` 也确实在取得写锁后重新检查版本，不兼容时回调不执行，事实和 `seq` 不变。

但只读预检与读写连接建立之间仍有窗口。

**Windows、Linux 均独立复现：**

1. 旧程序完成兼容性预检。
2. 另一进程升级数据库并退出，留下新版本 WAL。
3. 旧程序建立读写连接，复核后正确返回退出码 6。
4. 错误处理关闭最后一个连接，触发 checkpoint。
5. 主库从 12,288 字节变为 20,480 字节，哈希改变，WAL、SHM 被删除。

这直接违反当前验收要求的“拒绝打开时数据库、WAL、SHM 和目录项都不变”。[验收契约](../acceptance-criteria.md#L264)。

需要区分：

- **已打开连接的 `WriteTx` 拒绝：** 回调、事实、`seq` 的保证已满足；其后关闭连接发生 checkpoint，不单独证明这一部分验收失败。
- **打开过程中最终拒绝：** 上述反例违反文件不变的明确承诺。

[处理记录中的“已知边界”](../review-resolution.md#L522) 不能自行豁免验收。[现有事务拒绝测试](../../internal/store/sqlite/compat_test.go#L114) 也没有检查关闭后的文件变化。

**建议：** 将预检、连接建立和迁移纳入可靠的协调边界，并验证拒绝路径的全部文件副作用。若选择放宽为“逻辑内容不变”，必须由产品负责人明确确认，并同步修改规范和验收；当前不能按已接受风险关闭。

**关闭状态：部分修复，仍阻塞。**

## NON-BLOCKER

### N1：原有崩溃恢复和外键问题已修复，但并发读取时清理仍可能假报成功

位置：[tombstone.go:100](../../internal/store/sqlite/tombstone.go#L100)、[tombstone.go:106](../../internal/store/sqlite/tombstone.go#L106)。

新增标记支持删除提交后的恢复；延迟外键检查及 `projects.id` 删除也通过了独立运行。

但代码使用 `Exec` 执行 checkpoint，没有读取其返回的 busy 状态。SQLite 会通过结果列报告 checkpoint 未完成，而不一定表现为 SQL 执行错误。[SQLite checkpoint 文档](https://www.sqlite.org/pragma.html#pragma_wal_checkpoint)。

**Windows、Linux 均复现：** 另一连接持有读事务时，墓碑处理返回成功、删除整理标记，但 WAL 仍含被删除项目的 canary。

**建议：** 检查 checkpoint 返回值，在 VACUUM 后确认清理完成；无法完成时保留标记并报告未完成。

M0 只要求墓碑挂载点，故继续列为 NON-BLOCKER；必须在 M2 完整删除验收前解决。N1 应标为“部分修复”。

### 另一项读取检测遗漏

位置：[snapshot.go:412](https://github.com/XKHoshizora/specmetry/blob/9fcda14/internal/store/sqlite/snapshot.go#L412)。

Linux 上用等大数据库替换文件后，读取器仍返回旧文件版本，没有检测到替换并重试。验证比较了文件大小和旧句柄内容，没有比较文件身份。

**建议：** 如果保留该读取器，补充文件身份检测及替换测试。此项不另行扩大为架构阻塞。

N2～N4 已合理处理：

| 项目 | 结论 |
|---|---|
| N2 非法墓碑测试假阳性 | 已关闭：相同迁移集合、明确错误来源及目录不变断言有效 |
| N3 doctor 最低版本判断 | 已关闭：复用兼容性规则，实际测试通过 |
| N4 公开文档路径 | 当前文档已关闭：相对链接及检查规则有效；旧提交历史中的路径仍存在 |

## ACCEPTED RISK

- 同一用户或管理员直接改变权限、数据库，仍属于原有信任边界；不能用此边界解释 B1 的跨 UID 绕过。
- `doctor` 无法安全读取时报告限制，可以接受；必须保持“版本未知”，不能报告兼容或正常。
- 历史提交中的本机路径是已披露的元数据。本轮未发现需要清除历史的凭据或客户数据。

## 独立验证

| 检查 | 结果 |
|---|---|
| Windows 原有 Go 全量测试 | 通过；两个符号链接测试因权限不足跳过 |
| Linux 原有 Go 全量测试 | 在 WSL 的 Linux 临时文件系统中实际运行通过，包括权限和符号链接测试 |
| B4：REPLACE、UPSERT、主键、UNIQUE、连接池新连接 | Windows、Linux 通过，B4 可以关闭 |
| vet | Windows、Linux、macOS 目标通过 |
| macOS 测试二进制 | 交叉编译通过，未实际运行 |
| lint、格式及依赖完整性 | Windows 检查通过，依赖验证通过 |
| GitHub 三平台 CI、Linux Race Detector、macOS ACL 实测 | **未验证** |

原有测试通过，但新增独立反例暴露了上述缺陷；两者不能互相替代。

## 最终回答

1. **B1～B4 是否关闭：** B1、B2 未关闭；B3 部分修复但仍阻塞；B4 已关闭。
2. **N1～N4 是否合理处理：** N1 部分修复，保留并发清理遗漏；N2～N4 已合理处理。
3. **自实现 WAL 解析是否值得保留：** 当前整体方案不建议作为 MVP 可信基础保留；应优先采用安全读取受限时明确降级的方式。
4. **B3 残余竞态是否违反验收：** 打开阶段的拒绝仍改变文件，违反当前 `MVP-STO-02`；放宽承诺需要产品负责人确认。
5. **是否出现新风险：** 已确认跨用户路径替换泄露、未提交数据读取及清理假报完成。
6. **能否安全公开推送：** 从公开内容审查角度，可以作为未验收开发提交推送。当前内容及五个可达提交未发现凭据、客户数据或不应公开的构建产物；不能据此发布为已验收版本。
7. **M0 代码审查能否 GO：NO-GO。** 必须解决 B1～B3；此外，`MVP-PLT-01` 仍须等待 GitHub 三平台 CI，尚未通过完整 M0 验收。

