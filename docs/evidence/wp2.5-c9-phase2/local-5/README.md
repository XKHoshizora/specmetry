# 本机验证 5：Review 155 的回归在旧实现上运行（回归先行）

| 项 | 值 |
|---|---|
| 代码点 | `991e3292a7ca13bbfcaee2c7786ae27d551e1145`，树 `f3dbfe8d2edc5722c6cec424370753c1320dc86b`，工具模块子树 `e03046f170cf744102467fc8df39631b4c996742`：旧实现（`764da7e` 的控制协议与 LC-9，没有改）加上只用旧代码已有的东西写成的回归（H1、H3、S2 ~ S5、C2），以及它们用到的伪工具的三个新模式（`flood-term`、`flood-exit`、`term`，测试设施）（review-resolution §204） |
| 工具链 | `go1.27.3 windows/amd64`，SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`；`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` |
| 时间 | 2026-11-16T16:12:07Z ~ 16:12:23Z（UTC） |
| 工作区 | 运行前后各有 1 个变更路径：正在写入的本目录（未跟踪）；被测代码没有未提交的改动 |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-regress.log` | Windows 原生：`go test -count=1 -v -run 'TestR155' ./internal/toolctl/ ./internal/controls/` | 退出 1（预期的失败） |
| `wsl-regress.log` | 交叉编译的 linux/amd64 测试二进制（toolctl、controls）在 WSL（内核 6.18.33.2-microsoft-standard-WSL2）中以 `-test.run TestR155` 运行（只有伪工具与接缝） | 退出 1（预期的失败） |

逐项结果（与 §204 登记的预测逐一相同）：

| 回归 | Windows | WSL 中的 Linux | 旧实现的实际表现 |
|---|---|---|---|
| H1 `TestR155RegistryOutlivesItsCaller` | FAIL | FAIL | R2-f 的辅助函数返回、任务全部结束之后做完整的垃圾回收：登记表、记录、`Cmd`、`Process` 与四个文件全部被回收（弱指针为空），没有任何持有者 |
| H3 `TestR155HolderOutlivesTheVariant` | FAIL | FAIL | `output-not-closed` 返回之后（H 的就绪读端关闭失败、释放未知）：这次运行、登记表、H、H 的 `Cmd` 与就绪读端全部被回收 |
| S2 `TestR155OverflowOtherSignal` | SKIP（没有信号） | FAIL | 前提成立（`signal: terminated`、超量且保留 4096 字节），旧的判定按 `overflow` 的完整预期判为**命中** |
| S3 `TestR155OverflowNormalExit` | PASS | PASS | 前提成立（`exit status 141`），旧的判定已经不命中（退出码 141 与 -1 不同） |
| S4 `TestR155OverflowExtraRealError` | PASS | PASS | 额外的真实错误（`stderr-r` 的关闭，接缝代替）已被拒绝（Review 154 的修复） |
| S5 `TestR155ExpiryOtherSignal` | SKIP（只在 Linux） | FAIL | 前提成立（`signal: terminated`、请求完成且为 nil），旧的判定按 `expiry` 的完整预期判为**命中** |
| C2 `TestR155ExitedBeforeRequest` | SKIP（只在 Linux） | PASS | 请求完成、发送 1 次、nil；回收 `exit 0`；没有故障；已到期、判定失败：协议本来就有“请求之前已经结束”的合法分支，旧 RC-3 的断言会拒绝它 |

- 弱指针在对象不可达之后变空是运行时的行为，语言不保证；这里两个平台上都变空，作为旧实现没有根持有的动态观测记录，不是语言层面的证明。
- C2 的结果是本机 WSL 内核上的观测，固定 runner 内核上的核对见专用测试运行。
- 依赖新的持有者、新的规范行或新的判定的回归（H2、H4、H5、S1、S6、S7、C1）没有在这里运行，不计作“已在旧实现上动态复现”。
- `SHA256SUMS` 是两个日志的 SHA-256。
