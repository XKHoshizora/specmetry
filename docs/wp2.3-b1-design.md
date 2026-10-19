# WP2.3 B1-R：定向设计分析与最小隔离验证

> 日期：2026-10-19。实现前技术报告；未提交。固定对象为 PR #27 的 `ee144af61e00cb4d24bbae27feab10d8be60ecfc`，base 为 `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。
>
> 本文不是 Review 061，不重作合并审查，不修改冻结保证，不宣布 B1 关闭或 WP2.3 验收。B2、B3、已关闭的 N1 保持关闭；M-14／M-15 未解决；不启动后续工作包。

## 1. 结论与当前可实施程度

三个问题均成立。本轮用固定对象的仓库外副本、真实 Git 和独立的普通用户子进程重新复现，不操作真实项目的数据目录：

1. Windows／Linux 的 `mkdir → open` 间换入空目录，Clean、Log 均接纳它并删除；创建出的原目录保留在移走处。
2. Linux 的最终身份比较通过后、`unlinkat` 前换入空目录或所有者文件，Clean、Log 均删除替代物并返回成功。
3. Linux 在 Git 真正运行时换入防护路径，Git 返回后恢复，Log 成功但仅回调 **1/2** 个提交。

**Log 可以摆脱这里的临时路径依赖。** 隔离原型把扫描调用的两个覆盖值显式设为 `GIT_GRAFT_FILE=` 和 `GIT_SHALLOW_FILE=`，不再创建扫描防护目录。本机 Git 2.56.0.windows.2 和 WSL Git 2.43.0 均读全历史、正确应用排除集合，缺失父提交时失败；直接严格读取回归通过。该结果验证的是这两版 Git 的机制与本次最小原型，不是三平台生产修复完成。macOS、其他 Git 构建及完整扫描事务仍需验收。

**目录创建和删除没有同样小的可移植补丁。** Windows 的 `NtCreateFile`（`CreateDisposition=FILE_CREATE`，`CreateOptions=FILE_DIRECTORY_FILE`）可一次创建并返回句柄，句柄删除也已完成最小验证。Linux／macOS 当前使用的目录 API 不提供相同组合；父目录 fd、no-follow、前后身份比较都不能成为最后名称的条件删除。不能继续用更密的检查宣称消除窗口。

推荐先落实“不用临时路径的扫描”这一共同技术方向；完整 B1 路线优先保留现有保证，见方案 A。方案 A 涉及新的本地权限主体和安装方式，尚未验证端到端，也超出现有单一普通用户进程的部署约定，必须先由负责人决定。若坚持现有架构与原保证，当前证据不足以提供 Linux／macOS 正常 Clean 的实现；安全的缺省是执行前拒绝，而不是带窗口成功。方案 B 是明确的保证调整备选，**尚未获批**。

## 2. 工作区、来源和保留情况

已读取工作区根 `AGENTS.md`、开发指南、MVP 范围、完整 [Review 060](reviews/codex-review-060.md)、Review 059 的保证来源与接手要求、WP2.3 §14.3、相关生产源码与测试。未发现上级适用 AGENTS.md，也没有 gitx／docs 子目录的额外 AGENTS.md。

| 项目 | 本轮核对 |
|---|---|
| 工作区 | `C:/Users/hoshi/Desktop/specmetry` |
| origin fetch／push | `https://github.com/XKHoshizora/specmetry.git` |
| 分支 | `feat/m2-wp23-git-reader` |
| 本地实际 HEAD | `659b914901bfd4fc9a251fc102c69b2ebcfb509b` |
| 与固定 HEAD 的差异 | 仅新增 Review 060 原文归档，未变生产代码 |
| 归档操作 | Claude 已提交 `659b914 docs: archive Codex review 060 unchanged`；开始时工作区、暂存区均干净，无待完成的复制／暂存操作 |
| 归档文件 SHA-256 | `25c6f4e2ea6e0ebbf2e24fe261fd0f929ae3c138c30534ea91c9641854b4fab2` |
| 远程 | GitHub 连接器核对 [PR #27](https://github.com/XKHoshizora/specmetry/pull/27)：open、未合并，head／base 与固定对象一致，head 仓库为 `XKHoshizora/specmetry` |

本轮从固定 HEAD 导出副本，没有切换／重置工作区，也没有更新其索引。保留已有归档提交；本轮在仓库中只新增本文。没有修改生产代码、冻结规范、历史审查、Schema、迁移、PR 描述或 CI，也没有提交、推送、合并。没有重跑或重新裁定远程 CI。

## 3. 反例逐项复核

以下位置均针对固定 HEAD；源码推理、测试钩子与独立进程证据分开记录。

### 3.1 创建归属：mkdir 成功与首次打开之间

精确窗口：[privdir.go](../internal/gitx/privdir.go) 的 `newPrivateDir`，`s.root.Mkdir(name, 0700)` 返回成功（325 行附近）之后、`s.open(name)` 取得目录句柄与 `keyOf` 之前。

- `s.root` 绑定 tmp，能避免祖先路径把本次 mkdir 引向另一个父目录。
- `s.open` 与随后的 `d.verify` 比较的是“打开的目录”和“此刻名字指向的目录”；空检查只证明此刻为空。
- mkdir 的返回值不包含创建对象的身份。原目录尚未打开，不能把后取得的 inode／文件 ID 追溯为 mkdir 创建的对象。
- `O_EXCL` 创建所有者文件证明该文件是后来创建的；不能证明容纳它的目录是原目录。

调度：原有 `testHookMkdir` 阻塞读取，在独立子进程中执行“原目录改名到 .original；在原名建立空目录”，随后恢复生产调用链。Clean、Log 都返回 nil，替代目录被写入并删除，原目录仍在。Windows 此时没有该目录的保护句柄，替换确实发生；不是系统拒绝 rename 的用例。

**证据：Windows NTFS 与 WSL Linux 运行；原有精确钩子 + 独立子进程；现象断言通过。**

### 3.2 最终删除：身份比较与按名删除之间

精确窗口：[privid_unix.go](../internal/gitx/privid_unix.go) 的 `removeEntry`，`Fstatat(parentFD, name, AT_SYMLINK_NOFOLLOW)` 与身份／类型比较全部通过后，实际 `Unlinkat(parentFD, name, flags)` 之前。

覆盖两个不同对象：

| 对象 | 固定父对象 | 仍可改变的最后名称 | 本轮结果 |
|---|---|---|---|
| 私有目录 | tmp 的 fd | `read-…` | 原目录移走，换入空目录；rmdir 删除替代目录，Clean／Log 均 nil |
| 所有者文件 | 私有目录 fd | `specmetry-owner` | 原文件移至本夹具 tmp 下另一名字，换入见证文件；unlink 删除见证文件，Clean／Log 均 nil |

在隔离副本的最终比较之后加入同步点，独立子进程执行替换；没有把较早的 `testHookRemove("entry")` 冒充这一窗口。既有 hook 位于 `removeEntry` 内部最终比较之前，能证明更早替换会被拒绝，其证据仍成立。

父 fd 确保操作在指定父目录中进行；不能给 unlinkat 附加“只删除刚才那个 inode”的条件。flock 作用于原文件，不能阻止同名新文件被 unlink。空目录限制避免此调度递归删掉非空树，但仍是删除未知对象。

Windows 的实现不同：no-follow 打开后，文件 ID 比较和 `FileDispositionInfo` 使用同一个 handle。该对象不会因为旧名称被换掉而变成另一个对象。本轮原生实验验证了这一点，不把 Unix 反例套用于 Windows。

另外，`os.Root.RemoveAll` 绑定目录根，并不自动证明其每一个子条目由本次创建。完整清理设计必须覆盖内容、所有者文件与目录条目；不能只修最后 rmdir，然后继续把随时可注入的内容全部递归删除。这是同一归属保证的设计要求，本轮未扩展为新的审查项或开展额外攻击扫描。

**证据：Linux 的精确同步点 + 独立子进程 + 公开 Clean／Log；Windows 句柄机制另有原型，见 §4。macOS 只作源码／API 分析，未运行。**

### 3.3 Git 实际读取：验证后换入、Git 返回后恢复

精确窗口：[scan.go](../internal/gitx/scan.go) 的 `scanCall` 在 `d.verify()` 后构造 `d.join("none")`，到 [call.go](../internal/gitx/call.go) 真正执行 `commandIO`；替代路径保持到 Git 退出，然后在 `d.remove()` 的下一次身份复核之前恢复。

隔离副本仅在 `commandIO` 两侧加同步点。独立子进程移走防护目录，在原名建立替代目录，并将两提交夹具的 tip SHA 写入 `none`。这个内容同时可成为浅边界和无父 graft。真实 Git 退出后，子进程把替代目录移到 `.replacement`，把原目录恢复原名。未伪造 Git 输出，也未使用远端或外部 helper。

结果：公开 Log 返回 nil，回调只收到 tip，**1 个而非 2 个**。后续验证看到原对象，正常清理了原对象，替代目录留在实验位置。

当前目录 fd、Root 和锁保护 Specmetry 自己的相对 I/O；Git 没有使用这些句柄，而是重新解析环境变量的绝对路径。根链最终身份相同只能发现持续变化，不能证明中间没有另一对象。

Clean 同样在 [clean.go](../internal/gitx/clean.go) 的 `privateRepo.read` 检查后，将 `--git-dir=<p.gitDir>` 交给 Git；其实际读取不绑定 Root。本轮复核该源码链，并沿用 Review 060 已运行的“删除替代 index 导致 dirty／nil”证据，**没有本轮独立重跑该 Clean 往返用例，也不声称复现 false-clean**。

**证据：Log 在 Linux 的独立进程同步运行；Clean 为本轮源码推理 + Review 060 历史运行证据。没有验证数据库游标错误发布。**

## 4. 本轮最小实验与复现材料

仓库外实验目录：

`C:/Users/hoshi/.codex/visualizations/2026/10/19/01a153a5-24ad-7d82-b219-2787d737ce0a/b1-lab-e9860056`

可打开 实验材料清单 `C:/Users/hoshi/.codex/visualizations/2026/10/19/01a153a5-24ad-7d82-b219-2787d737ce0a/b1-lab-e9860056/manifest.json`、Windows 定向日志 `C:/Users/hoshi/.codex/visualizations/2026/10/19/01a153a5-24ad-7d82-b219-2787d737ce0a/b1-lab-e9860056/windows-go.log` 和 Linux 定向日志 `C:/Users/hoshi/.codex/visualizations/2026/10/19/01a153a5-24ad-7d82-b219-2787d737ce0a/b1-lab-e9860056/linux-go.log`。这些绝对链接仅在本机可用；归档转移时应同时保留整个实验目录。

固定 HEAD 的 `target.tar` SHA-256：

`181186633d01bf095301b54f2c628453cf094e94f4df84fdba11b02ce4b3b7c4`

该目录保存导出副本、只用于副本的钩子／原型、原生 API 脚本、日志与 `manifest.json`。manifest 记录源码与日志 SHA-256。材料没有复制进真实仓库。

| 实验 | 平台／方式 | 最终结果与范围 |
|---|---|---|
| TestB1MkdirEmpty | Windows／Linux；既有钩子调度独立测试进程 | Clean／Log 均复现接纳并删除空替代目录 |
| TestB1FinalDelete | Linux；最终比较后调度独立测试进程 | 目录和 owner 两种 × Clean／Log，共四例均复现删除替代物／nil |
| TestB1LogRoundtrip | Linux；commandIO 前后调度独立测试进程，真实 Git | nil；1/2 提交 |
| TestB1PathFreeLog | Windows／Linux；只在副本替换 scanCall 后端 | 无 Scratch；仓库 shallow 与 grafts 同时含 tip 时仍 2/2；排除根后 1/1；删掉父对象后失败 |
| TestB1PathFreeRegression | Windows／Linux；在候选后端下调用既有直接回归 | LogIsStrict、LogReadsWhatItVisits、LogRunsNoSignatureProgram、CheckShallow、IsAncestor 全部通过 |
| windows_native.py | Windows NTFS；ctypes 调用原生 API，普通子进程试 rename | 原子创建返回 handle；已存在名称为 0xc0000035；不共享删除时 rename 返回 WinError 32；共享删除对照中移走原对象并放入同名目录，handle disposition 只删原对象，替代目录保留 |
| linux_primitives.py | WSL 普通用户；原生调用和独立子进程 | 目录 fd 在 rename 后仍指向原对象；本内核 O_CREAT\|O_DIRECTORY 为 EINVAL；unlinkat 的空路径／AT_EMPTY_PATH 及 rmdir "." 均 EINVAL；两目录保留 |

环境：Windows/amd64，Go 1.27.1，Git 2.56.0.windows.2，本地 NTFS；WSL Ubuntu 普通用户 UID 1000，Linux 6.18.33.2-microsoft-standard-WSL2，真实 Git 2.43.0，Go 测试二进制由固定副本交叉编译，夹具在 Linux `/tmp` 新建。没有以管理员／root 身份运行攻击调度。

日志：`windows-go.log` 最终定向测试 31.405s；`linux-go.log` 全部上述适用项通过；`windows-native.log`、`linux-primitives.log` 保存原生结果。**反例测试 PASS 的含义是准确观察到违规行为，不是原代码满足保证。** Windows 两项 Unix 专属调度明确 SKIP。

复现时在一个新的隔离目录解开 target.tar，复制／运行 prepare.py，再在该副本执行：

```text
go test -tags nowebui -run '^TestB1' -count=1 -v ./internal/gitx
GOOS=linux GOARCH=amd64 go test -tags nowebui -c -o <isolated>/gitx-linux.test ./internal/gitx
SPECMETRY_TEST_GIT_OBJECTS=safe <isolated>/gitx-linux.test -test.run '^TestB1' -test.v
python windows_native.py
python3 linux_primitives.py
```

prepare.py 只可对新解开的固定副本执行一次；不可指向真实 checkout。副本内新增测试文件为 `internal/gitx/b1_experiment_test.go`，修改仅限 call.go 的调用同步点、privid_unix.go 的删除同步点、scan.go 的可切换候选分支和 Windows 测试 stub。基线反例使用原算法；只有候选测试打开 path-free 分支。

初跑有环境和夹具纠正：受限进程的账户映射导致新建目录不可访问，改为正常用户运行自建夹具；候选夹具最初缺少 `.git/info`，补齐后才计成功；Windows 原生实验最初持有过宽父句柄，rename 对照被系统阻止，改为最小属性句柄并共享删除后，替换才真正发生。owner 原型最初把原文件留在目录内，使最后 rmdir 报错；最终版本移到本测试 tmp 的独立名字，四例均 nil。以上初跑不计作产品反例或候选成功。

未运行：macOS、本轮跨用户动态验证、ReFS／网络文件系统、旧 Git 构建、全仓测试、Race Detector、性能／压力测试、生产 broker 端到端、原生 Windows 创建与 Clean 的集成。没有把 Review 060 的 CI 证据写成本轮候选方案的 CI 证据。

## 5. 平台能力与剩余窗口

标签：**已验证**＝本轮实际运行指定平台与调度；**文档依据**＝核对官方文档／源码；**尚未验证**＝完整候选尚未运行，不能作为关闭依据。

| 机制／阶段 | Linux | macOS | Windows |
|---|---|---|---|
| 目录 mkdir 后取句柄 | 已验证：是两次操作，空替代可被接管 | 文档依据：mkdirat 返回状态，另行 open；未运行 | 已验证：现有 Go 两步方式同样接管 |
| 一次排他创建目录并返回其 handle | 所检查的 mkdirat／openat／openat2 无此组合；O_CREAT\|O_DIRECTORY 本机失败 | 所检查的 mkdirat／openat 无此组合；未找到可用的公开通用接口，不断言所有专用机制均不存在 | 已验证：NtCreateFile、相对已打开父目录、FILE_CREATE + FILE_DIRECTORY_FILE；冲突即失败，不 OPEN_IF |
| 已打开目录移动后仍引用原对象 | 已验证 fd 固定对象；不固定其父中的名字 | 文档／当前 Go 与 Darwin 路径依据；未运行 | 已验证可持 handle；指定不共享删除可拒绝 rename；不能泛化为所有祖先永久固定 |
| 最后条目的删除 | 已验证 fstatat → unlinkat 可误删；空路径不能按目录 fd 删除 | 文档依据：unlinkat／rmdirat 仍接受 fd + name，未见 expected-ID 参数；未运行 | 已验证同 handle 上查询身份／disposition 可只删打开对象；必须保留原始创建身份 |
| Git 的扫描防护输入 | 已验证两空值覆盖，不再查 scanCall 的临时路径 | 同 Git 源码有文档依据；发行版／环境传递未运行 | 已验证 Go 子进程确实传入空值并压制两个仓库防护文件 |
| Git 的 Clean 私有目录输入 | 当前是名字，Root 不传给 Git；仍有往返窗口 | 当前同样是名字；未运行 | 原生目录 handle + 明确的 no-delete guards 可固定相关目录名字；**尚未验证 Clean 集成**，也不能防内部条目注入 |
| 不共享删除／flock 等锁 | flock 只约束合作方，不能禁止 rename／unlink | 文件锁不能充当目录名称互斥；未运行 | no-delete share 为内核访问约束；本机已验证单对象，不以“有子文件打开”替代逐级证明 |
| 无法证明归属时 | 不写入、不接管、不按路径清理；保持原／替代对象并失败 | 同左 | 冲突／reparse／ID 不符时不接管、不置删除；释放自己实际持有的 handles |

Linux 的目录 fd 可以固定父对象，O_TMPFILE／memfd 可以提供匿名**普通文件**；它们不是匿名目录的创建和删除接口。[Linux open](https://man7.org/linux/man-pages/man2/open.2.html)、[unlink](https://man7.org/linux/man-pages/man2/unlink.2.html)、[mkdir](https://man7.org/linux/man-pages/man2/mkdir.2.html) 与本轮原生实验支持这些区分。这里的判断限于检查到的接口和本机内核，不把“未找到”写成数学上不存在任何方案。

macOS 的官方 [mkdir 手册](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/mkdir.2.html) 及 [XNU vfs_syscalls.c](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/vfs/vfs_syscalls.c) 显示 mkdirat 和 unlinkat 的分离及按名语义。XNU 的 no-follow／beneath／nodeletebusy 选项不是调用者提交 expected inode 的接口；尚未在目标 macOS 版本证明其他机制可闭合窗口。

Windows 的 [NtCreateFile](https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntcreatefile) 规定 FILE_CREATE 遇已存在名称失败，并可用 RootDirectory 绑定父目录；共享访问可约束后续 delete／rename。生产适配仍须验证继承 DACL、祖先保护、文件系统能力、异步错误和 Go 句柄接管，不能以本机成功替代跨平台验收。

### 5.1 明确不能单独承担保证的机制

- 更多 Stat／Fstatat、随机长名字、空目录、owner 内容和 inode 记录：可诊断或降低碰撞，不把两次系统调用合成一次。
- rename 到随机“隔离区”后再检查：rename 本身仍按可变名称，可能先移动未知对象；移回又有新竞态。不作为安全删除。
- fsnotify／inotify／FSEvents／轮询：本轮未验证完整性、事件丢失和屏障，不能证明 Git 的每次 open 都未经过替代物。
- Linux `/proc/self/fd/N`：对继承的普通文件描述符有绑定用途；`/dev/fd/N` 在 macOS 的目录后缀和 Git 行为尚未验证。Git 可能 canonicalize 目录路径；不能只换路径前缀就宣布整个 Git 目录树被绑定。即使父 fd 固定，子名字仍可换。
- fchdir 到打开目录：绑定 cwd，不阻止目录内名字被替换，也不证明 Git 后续不重新解析绝对路径。
- chmod 0500／0700、同 UID ACL、advisory lock：属主仍可改权限或忽略锁，不满足本轮主动同用户名称调度的保证。
- Linux 私有 mount／user namespace：单独采用未验证；同 host UID 对进程与 `/proc` 的访问、fd 暴露、exec 后 dumpable 行为需一起证明，不能把“另一个 mount namespace”直接写成不可干预。没有在本轮启用特权挂载或系统服务。
- 仅在 Unix 停止删除：可以避免误删，但没有解决 mkdir 接管与 Git 误读；且长期保留索引／配置副本会改变隐私与清理契约，不能据此返回成功。

这些都是对候选机制的限制说明，不是新增独立方案或保证豁免。

## 6. 两个完整设计方案的共同部分：去掉扫描防护路径

两方案都采用这一部分；其价值不依赖负责人是否批准范围调整。

### 6.1 数据与调用路径

`CheckShallow → Tips／Classify／IsAncestor／Log → 既有严格解析 → 既有事务`。保留 P3、Git 能力门禁、无 lazy fetch、禁 replace／commit-graph／multi-pack-index、no-show-signature、丢弃部分读取及回调失败语义。

将 scanCall 中“创建目录、构造 none、读取、清理”替换为受控环境下的两个**显式空值**。不能把空值变量从环境过滤器删除；删除意味着回退仓库的 shallow／grafts。调用参数与环境由结构化 argv／Env 传递，禁止经 shell 拼接。空字符串不是一个可被 rename 成文件的路径名，也不使用可被换掉的 /dev/null／NUL 路径作为正确性根基。

Git 2.43 [environment.c](https://github.com/git/git/blob/v2.43.0/environment.c) 区分空值与不存在；[repository.c](https://github.com/git/git/blob/v2.43.0/repository.c) 保存非 NULL 的覆盖值。[shallow.c](https://github.com/git/git/blob/v2.43.0/shallow.c) 对空值直接表示不用 shallow；[commit.c](https://github.com/git/git/blob/v2.43.0/commit.c) 尝试打开空的 graft 文件名，失败后不载入 graft。对 Git 2.56 的对应源码亦核对了相同语义。本轮两版实际行为与之相符；这里不把 graft 空值称为所有 Git 厂商承诺的永久公共开关。

**Log 的完整性于是无需依赖“none 在 Git 运行期间不存在”这一可变路径事实。** 本轮把两个仓库文件持续设为恶意截断内容，是比该输入短暂出现再恢复更直接的机制对照：只要显式空值始终送达，这两个名字不参与该读取。仍须增加真实进程往返回归，防止未来错误地重新引入临时目录。

### 6.2 能力门禁、兼容性与接口

- 在现有 Git 能力体系内，为所选二进制增加独立的“空覆盖有效”行为探测：合成两提交仓库，同时放入可截断 shallow／grafts；受控调用须得到完整历史；缺失父对象须失败。探测本身的夹具归属和生命周期也必须经选定的安全后端证明，不得把同一创建问题转移到新 probe。
- 以实际 Git 二进制身份缓存探测，不只按版本号猜测。此处尚未实现缓存／探测生命周期；不得用最小实验替代生产探测。
- 不支持／无法探测时返回 environment／退出码 6。**不回退原来的临时 missing 路径。**
- MVP 不扫描 shallow clone 的前置门禁仍在；覆盖只是防止检查之后新增 shallow 文件影响读取，不是批准扫描浅仓库。
- 最小生产修复可暂时保留 Tips／Classify／IsAncestor／Log 的 Scratch 参数及现有“缺少／已关闭属于内部错误”契约；只停止为扫描调用分配私有目录。随后单独决定是否去掉扫描的 Scratch 依赖、从 gitscan.Options 解耦，不能像实验中的 nil 测试一样悄然改变生产 API。
- Clean 仍需要受保护的私有目录，不能从扫描原型通过推导为 Clean 问题已解决。

资源成本减少为每条扫描 Git 调用不再创建、写锁、删除目录。现有解析、排除集合、严格失败和事务算法无需重写。macOS 与支持 Git 的行为探测、所有扫描入口直接回归、失败不写库仍是生产验收成本。

## 7. 方案 A：保持原保证，由独立权限主体管理只读临时区

**优先保留保证的完整候选；仅权限机制有文档依据，端到端尚未验证。** 不是已获授权的生产改造。

核心变化：临时区的属主不是运行 Specmetry／Git 的普通用户，而是受信任的本地 broker。调用用户仅有读取／遍历权限，没有修改名称、写内容、改权限或改属主的权利。broker 是本地文件服务，不是 Agent，不调用 LLM、不联网、不调度 Agent。Git 继续以原用户身份读原项目，避免扩大读取项目的权限。

### 7.1 建立、封存、读取、删除

1. **可信根**：安装阶段建立 broker 专属根，整个祖先链都不受调用用户控制。不能在用户可改名的数据目录 tmp 内简单 chown 一个子目录；父目录可替换会重新开放 Git 路径窗口。
2. **建立归属**：broker 按租约串行操作新目录。普通用户不能在 mkdir/open 之间改名字，其他 broker 请求与回收也不得触碰该租约。Unix 因外部改名被内核拒绝而闭合窗口；Windows 还可用 FILE_CREATE 原子返回句柄。记录创建身份和预期条目清单。
3. **填充**：由客户端从原仓库读取当前已有规范允许的 index／shared index、白名单配置、HEAD、info 等数据，流式交给 broker；不接受任意 shell、可执行程序、任意清理路径或配置原文直接执行。对象库继续使用既有 alternates 指向原库，不复制源码／整座对象库或提交正文到新持久存储。
4. **封存**：建立与填充时客户端不可遍历；完成后只授予该用户读取／遍历，文件不授予写、delete、改 ACL／owner。broker 在租约结束前不改目录树，Git 也不能写这个树；保留禁可选锁等受控环境。客户端可直接使用稳定绝对路径，因为威胁主体不能改其任一前缀或子条目。此为内核权限互斥，不是事后检查。
5. **Git 读取**：扫描使用 §6，不需要临时区。Clean 使用封存路径，沿用当前过滤器、稀疏门禁、子模块和 pathspec 行为。私有区封存不冻结原工作区、objects 或引用；不能提升为 snapshot_isolated 证据。
6. **清理**：Git 子进程已经退出后客户端释放租约；broker 才回收。仅 broker 可变更名称，内部每个租约的回收串行化；按创建清单和持有的身份处理内容、owner 与目录。Windows 尽量使用创建时句柄的 disposition；Unix 可在独占命名空间按 fd + 名字删除，因为替换者没有写权限，其他 broker 操作又被排除。
7. **崩溃／未知对象**：连接断开不能立刻认定 Git 已退出；客户端以 OS 进程句柄／受保护租约保证子进程生命周期，无法确认则保留。broker 重启只收回有可信创建记录、无活动租约的对象；不凭 PID 或时间判断。缺记录、身份不符、额外条目、ACL 异常均保留**整个目录**，停止清理并返回固定环境错误。不得向用户转发 broker 任意路径／内容。
8. **失败**：broker 缺失、租约失败、权限能力不足、无法封存或清理失败时环境错误，不运行不安全的 Clean、不返回 clean。不回退当前普通用户 tmp。旧版用户 tmp 遗留不自动接管；迁移清理政策须单独决定。

此处“按名称删除安全”的前提可检查、可由攻击进程实际验证：所有能修改该命名空间的主体均被排除或在 broker 内串行化。它不是“同用户可信”的重命名。

### 7.2 三平台实现与成本

| 平台 | 所需机制与对象绑定 | 剩余窗口／失败策略 | 兼容性与验证成本 |
|---|---|---|---|
| Linux | system service／专用 UID 或 root 管理，系统控制的祖先；POSIX ACL 只给调用 UID r-x／r--，broker 持父 fd／目录 fd；IPC 获取 OS 对端身份 | 普通调用 UID 无法改目录或 ACL；broker 自己必须串行化。ACL／可信根不可建立则不执行。未知条目保留 | 新安装与升级权限；ACL／本地文件系统能力；双普通 UID 的 create/open、Git 中间、最终 unlink 调度；crash、取消、并发租约；尚未实现 |
| macOS | launchd 管理的受信任 helper，独立权限主体；系统控制的祖先、扩展 ACL 限制调用用户为只读；目录 fd；认证的本地 IPC | 同上；不能用用户可控 LaunchAgent 进程加 0700 冒充独立权限主体。无法证明 ACL／祖先即失败 | helper 安装、签名／授权、系统版本与 ACL 行为、实际 Git 打开路径、TCC 与沙箱兼容；均需目标系统验证 |
| Windows | SYSTEM／独立 service SID 所有的固定根，受保护 DACL 只授调用 SID read／traverse；NtCreateFile 相对父 handle，FILE_CREATE；同 handle 查询与 disposition | 名称和内容的修改均由权限阻止；不能只阻止目录 rename 而允许新建内部文件。ACL、ID 或 no-follow 不满足则失败；未知对象不置删除 | 服务安装、ACL／token／IPC、继承规则、权限恢复／更新；NTFS 原子创建和删除子机制已验证，完整 broker／Clean 尚未验证，不能泛化 ReFS |

Linux 的 [ACL 权限模型](https://man7.org/linux/man-pages/man5/acl.5.html)、Windows 的 [文件权限模型](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights) 为这一架构提供依据；macOS 的 [daemon／helper 设计文档](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/DesigningDaemons.html) 提供服务形态依据。三者都不是本方案已经成功集成的证明。

接口建议为 `AcquireReadArea → Populate → Seal → Release`，对外只有不透明租约，不提供任意 remove／rename 服务。每个租约持有创建清单、身份、进程／连接引用；内存和 fd 数量随同时读取及私有树条目增长，需要有界限额。IPC 只承载规范允许的临时数据，不落协议日志，不成为项目事实或备份。broker 的启动、升级、崩溃恢复和未完成回收均进入测试范围。

**与冻结设计的差别**：新增受信任服务、非用户 tmp 的位置和只读 ACL；改变单一二进制／无需管理员安装的工程预期及遗留管理。还须核对 architecture §10 的临时数据保留边界。虽然不降低“不接管／不误删／不因替代路径漏读”的安全保证，也不能在现有授权下直接改规范和上线。

## 8. 方案 B：无特权协作域，明确批准局部保证调整后实施

**保证调整备选；未获批准，不推荐作为默认选择。** 保留普通用户单二进制和数据目录 tmp，要求所有会改变该私有命名空间的进程遵守协议。这里的“所有”包括普通误操作，不只是恶意程序。

### 8.1 完整处理链

- **建立**：每个 Scratch／遗留清理进程共同持有一个稳定的全局或 tmp 命名空间锁；锁文件长期保留，不在每次读取后删除。锁覆盖建立、封存、全部 Git 调用与清理。无法取得锁立即或有界等待后环境错误，不自动重试业务写入。锁的首次建立仍须排他创建和身份验证；它不能抵抗外部替换，这一前提必须写入批准范围。
- **平台建立差异**：Windows 用 NtCreateFile 的 FILE_CREATE 原子返回目录 handle；Linux／macOS 仍 mkdirat + openat，合作锁防其他合规进程插入，前后检查只作异常诊断。不得声称 Unix 两步已原子化。
- **Git 读取**：扫描全部采用 §6，无临时防护路径。Clean 保持当前私有树；Windows 明确持有必要祖先／目录 no-delete guards 到 Git 退出，并保留预期文件 handle。Linux／macOS 在合作锁内给 Git 路径。三平台都不声称能阻止不合作进程改写内部文件；空的缺失名称也不能只靠一个目录 handle 保护。
- **清理**：按创建清单清理，核对各条目身份；遇额外条目或不符保留整个对象，不递归删除猜测的树。Windows 按已验证的同一 handle 删除，不为方便退回名字；Unix 仍在合作锁内执行 fstatat + unlinkat，明确依赖域内没有其他名称修改者。owner 文件最后处理，失败释放实际 handles；清理失败仍环境错误、不报告 clean。
- **遗留**：只在相同合作锁内处理可信记录，活跃对象与未知对象保留，不依前缀／年龄认领；旧版本不遵守新锁时拒绝并发使用，不能静默混用。
- **取消与崩溃**：等待 Git 退出后释放租约／锁；崩溃由 OS 释放锁，下一进程仍要验证记录及生命周期，不按 PID 猜测。无记录对象保留，不转为成功。

Windows 原子创建／handle 删除子机制已验证；其与 Clean、所有祖先／文件 guard 的集成尚未验证。Unix 合作锁的完整并发／崩溃协议尚未验证。macOS 无本轮运行证据。新增接口主要是 Scratch 内部锁／租约和平台 create／dispose 适配；资源为长持锁与 handles，代价是并行 Clean 被串行化、锁竞争时返回环境错误。测试须覆盖双进程合作竞争、旧版混用、取消／崩溃、正常回收与无法回收，不能用被假设排除的干预测试“通过”来证明安全。

### 8.2 必须批准的精确边界

| 要调整的保证 | 排除的具体情形 | 仍保留什么 | 用户可能看到什么 |
|---|---|---|---|
| Unix 创建归属 | 不合作的同 UID 进程在私有区 mkdir/open 之间替换对象，**包括普通误操作**；Windows 用原子创建收回该窗口 | 其他账户保护、合作进程并发、已观测的异常拒绝 | 越界调度下仍可能接管空替代目录；不能承诺总能报错 |
| Unix 最终删除身份 | 不合作的同 UID 进程在最后比较与 unlinkat 之间换名称，含 owner 文件和空目录 | 合作锁内正确清理、检查已见不符则保留、Windows handle 删除 | 越界调度下可能误删替代 owner／空目录并成功 |
| Clean 私有输入的独占性 | 不合作的同 UID 进程在 Git 运行时替换目录或内部条目，随后恢复 | 正常项目改动仍按现有观测级规则；扫描的空覆盖独立保留 | Unix Clean 可能返回受替代输入影响的 dirty／clean；原型只复现 dirty，不能据此排除 false-clean |
| Log 临时路径完整性 | **不申请这一项豁免**；两空值机制必须落实并验收 | 不因 scanCall 防护路径往返而漏读成功；缺失对象仍失败 | 不支持该机制的 Git 返回环境错误，不能带窗口成功 |

批准方案 B 不是“同用户不属于权限边界”一句话。它会把“不合作同用户、甚至误操作改变私有区时仍不接管／不误删”的已有承诺改为前置假设。该变更只限 Git 临时命名空间，不扩展到数据完整性、SQLite 版本拒绝、seq、来源权限、隐私和交付证据。**本轮没有批准任何一行。**

## 9. 威胁模型、能力边界与不能兑现时的选择

| 主体／行为 | 当前保证如何适用 | 方案 A | 方案 B |
|---|---|---|---|
| 其他普通账户 | 既有数据目录、祖先、DACL／权限保护仍须成立；本轮不否定 Review 060 跨账户结论 | 新根也必须对其他账户不可读写，并动态验证 | 原规则不降级，继续跨用户回归 |
| 正常同用户并发 | 多个 Specmetry 正常读取、清理必须正确；不能因“同 UID”忽略竞态 | broker 租约与封存协议排除内部并发修改 | 合作锁串行化；未遵守协议的版本不能混用 |
| 同用户普通误操作改变私有区 | 原保证要求不接管未知物、不误删；本轮反例无需特权 | 对调用用户写操作由内核拒绝 | 属于必须明确批准的排除项，不能悄然归为恶意行为 |
| 主动同用户、仅 rename／mkdir／unlink／文件写入 | 本轮未获豁免的核心调度；不等于任意控制进程 | 不能改受信任 broker 区；通过只读权限落实 | 排除域外；完整后果见 §8.2 |
| 直接注入／控制 Specmetry 或 Git 的内存、替换所执行的 Git、盗用运行中的能力 | 本轮不声称抵抗任意同用户进程控制；不能据此取消上一行有限承诺 | broker 不接受任意路径删除；仍不证明被控制客户端的业务判断 | 不由合作锁解决 |
| root／管理员主动干预 | 沿用可信特权主体边界，本轮未测试其越权 | 可改 broker／文件系统，因此不承诺抵抗 | 同样不承诺；不能把普通同用户调度混同为管理员能力 |

在**现有普通用户 Unix 架构 + 原保证 + 全部现有正常 Clean 能力**三项同时保持的条件下，本轮没有形成经验证的完整实现。不能把这个结论改写为保证可以自动放宽。

负责人可作的具体调整是：

1. **保持保证，允许新的本地权限隔离架构**：选择方案 A 的下一轮验证；原有语义保留，安装、资源、临时区位置和生命周期需批准。当前还不能生产验收。
2. **保持保证，调整可用平台／能力**：没有安全后端时，在任何私有区创建／Git 使用之前返回 environment 6；Linux／macOS Clean 暂不可用，Log 可独立走经验证的空覆盖。这样保留不接管／不误删／不漏读，但损失现有正常场景可用性；“所有平台每次失败”不能冒充正常 Clean 验收。生产启用这一限制也需负责人决定。
3. **坚持现有部署和正常可用性，批准精确安全边界调整**：仅方案 B §8.2 所列情形，不接受泛化豁免；仍必须消除 Log 临时路径问题及 Windows 可原子解决的窗口。

以上是取舍事项，不是三个已验证的实现方案。继续维持原保证时，可替代路线是方案 A 或安全失败；“继续前后检查并宣称没有新增风险”不是替代路线。

## 10. 需要负责人决定的具体事项

本文只登记 B1 内待决项，不替负责人新增“已批准”PD／TD，也不修改冻结正文。

| 事项 | 推荐 | 不作决定时 |
|---|---|---|
| B1-D1：扫描去路径方向 | 采用 §6，授权 Claude 在后续生产修复中实施并做三平台／能力探测验收；不改变 P3 和严格失败 | 保留技术证据，生产代码不变 |
| B1-D2：完整目录路线 | 优先不降低保证；先决定是否允许方案 A 的本地权限主体／安装与部署变化。若拒绝该架构，明确选择安全失败的能力限制，或逐项审议方案 B | B1 保持未闭合，不能假定 Unix 补丁已找到 |
| B1-D3：临时数据与遗留生命周期 | 未知对象始终保留；正常清理失败仍退出码 6。若改服务区或暂不清理旧 tmp，批准位置、保留数据类型、回收条件、错误行为 | 沿用冻结契约，不允许以成功但无限保留副本替代 |
| B1-D4：扫描 Scratch API | 第一轮尽量保留公开签名和缺失／关闭错误契约，仅更换 scanCall 后端；去参另行明确范围 | 不把实验中 nil Scratch 的便利带入生产 |

不建议为尽快关闭 B1 批准同用户窗口。选择任何保证调整后，都要由负责人明确记录，再同步规范、验收和实现；本文本身不是批准记录。

## 11. 下一轮生产修复的准确范围与验收条件

下一轮由 **Claude Code 实施，Codex 定向复核，负责人决定合并与验收**。本节是可交接范围，不是本轮开工或合并授权。

### 11.1 范围

共同部分只涉及 `internal/gitx/scan.go`、受控环境／Git 能力门禁的最小接点及相应直接测试。可保留公开签名，避免无关调用方重构。不得重写扫描事务、迁移、索引解析、推导引擎或 CLI。

目录部分限于 `privdir.go`、平台创建／身份／删除适配、Clean 把私有输入交给 Git 的接点、Scratch 租约／生命周期与直接测试。方案 A 必须先有新的具体授权和平台原型，不能借 B1 修复顺带安装服务；方案 B 必须先有逐项保证调整批准。未作决定时只允许整理设计证据，不能选其中一条生产实现。

选定路线后再同步 WP2.3 §14.3／§14.8、architecture 的临时区与隐私／权限契约、development、acceptance-criteria 相关项与 review-resolution。历史 Review 058～060 原文不修改；只澄清较早 hook 与最终 unlink 窗口的层次，不重开已关闭 N1、B2、B3。

### 11.2 必须给出的验收证据

1. **创建归属**：原 mkdir/open 调度中的空目录、非空目录、链接／reparse point；三平台分别证明“内核拒绝替换”或“替换后拒绝且双方保留”。正常空目录创建成功对照必需。Windows 预先存在的空目录必须 FILE_CREATE 冲突，禁止退回 OPEN_IF；Unix 不能仅加 Stat。
2. **实际读取**：真实 Git 的 Log 往返调度不再能造成成功漏读；须对两提交、合并／多父、增量排除集合、提交对象缺失／损坏、commit-graph／MIDX、shallow／grafts 持续与往返变化给出完整结果或失败，不能只比较退出码。候选禁 signature、no lazy fetch 保持。Clean 同步点覆盖 status、ls-files、diff-index、diff-files；证明每次消费都处于受保护对象／不可变命名空间中。
3. **删除**：精确在最后比较后调度 owner 文件及空目录替换；同时覆盖内容、取消、失败、正常结束、重试、遗留清理。替代文件字节、替代目录及原对象分别断言；不能只断言没有删非空目录。Windows 是同一创建／已知身份对应的 handle；Unix 按名删除要提供命名空间写者已被排除的证据。
4. **权限与生命周期**：三平台实际运行；其他账户、正常同用户双进程、同用户误操作／主动普通文件操作分别测试。替换未发生必须记为 OS 阻止，不能记成代码检测。未知归属对象完整保留；不因 PID 复用、复制 owner 记录或无 owner 而认领；新 Scratch 也不得删除替代物。方案 A 需覆盖客户端／Git／broker 各自崩溃、连接断开、重启与未释放租约，方案 B 需覆盖合作锁和旧版本并发。
5. **失败语义**：能力不足、保护建立失败、清理失败、回调／Git 失败一律不发布成功结果；提交、关联、seq、游标和其他表不变。直接复用 `gitscan` 的失败不写与部分读取丢弃用例验证，不重审其未变事务设计。不吞环境错误、不添加生产重试绕过 M-14／M-15。
6. **可用性与隐私**：三平台正常 Clean／Log、取消、重复 Close、无索引、shared index、稀疏门禁、过滤器、子模块和 .specmetry 排除的直接回归；不重新设计 B2／B3。正常读取回收全部自己的临时对象；未知对象保留后错误明确。若采用能力禁用／生命周期变更，先出负责人批准，再以批准后的条件验收。提交消息、文件内容、工具输出不得新增持久化／日志／备份。
7. **机制回归检出力**：把空覆盖退回 missing 路径应被 Log 往返测试检出；移除 Windows 原子创建／同 handle 删除或新权限封存应被对应测试检出。测试中编译失败不计作检出产品问题。
8. **证据边界**：保留固定 HEAD／base、实验源码与日志、实际平台／Git 版本和成功／失败历史；macOS、未支持文件系统、旧 Git 不得用交叉编译或别的平台通过冒充。最终再运行改动所需检查与 CI，由后续定向复核裁定，本文不预先给 GO。

### 11.3 可直接交给实施方的提示词

> 接手 Specmetry PR #27 的 B1-R。先读 AGENTS.md、Review 060 和 docs/wp2.3-b1-design.md，并核对负责人对 B1-D1～D4 的最新决定、固定对象及工作区；保留全部既有改动。先按已批准范围为扫描读取去除防护临时路径，保留 P3、Git 能力门禁、严格失败、禁外部程序、部分读取丢弃和现有 Scratch API 契约；不支持空覆盖时环境错误，不回退 missing 路径。目录创建／Git 实际消费／内容与最终条目清理必须按负责人选定的完整路线处理；没有决定就停在该依赖之前，不以多加身份检查代替原子性或权限证明。实施本报告 §11 的精确调度及正常对照，区分 OS 阻止与替换后拒绝，同步获批规范和验收项。只做 B1 及直接回归，B2／B3／N1 保持关闭，M-14／M-15 未解决，不改生产 SQLite 重试／错误映射，不启动后续包，不提交、推送或合并，除非负责人另行明确授权。完成后整理提交／diff、测试、剩余窗口、遗留对象和待决项，交 Codex 定向复核与负责人决定。

本轮交付止于本文及仓库外实验材料。B1 的技术证据与路线已具体化；完整候选仍须经过负责人选择、生产实施与独立验证。


## 12. 归档格式说明与后续授权

本文件原稿 SHA-256：`266d5fbaa130da323a0558806e47eb62531343d760cd6ff336c703601b92c2a4`。入库时仅将三个本机绝对路径 Markdown 链接整理为代码格式纯文本，并添加本说明；整理后的文件不与原稿逐字节一致。原分析对象、原型证据和未验证边界保持原样。第 11.3 节是当时建议；后续实施以负责人的 B1-D1～D4 明确批准为依据，见 [活跃决定 §71](review-resolution.md#71b1-方案-b-的批准与实施范围)。不由本文批准产品取舍或宣布修复通过审查。
