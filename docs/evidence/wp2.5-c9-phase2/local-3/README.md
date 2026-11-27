# 本机验证 3：Review 154 的回归在旧实现上运行（回归先行）

| 项 | 值 |
|---|---|
| 代码点 | `2cdbc1dec9177191b729b241b98ea6d5cedd06e5`，树 `68790ae83a644423eafef1942351e0ed060b228d`，工具模块子树 `396064f678d5be5b81bedb027208f1771a163d2c`：旧实现（`8e228f8`）加上只用旧字段写成的回归，以及 `close-real-error`、`blocked-send` 的接缝中一个只供回归替换真实操作的入口（review-resolution §202） |
| 工具链 | `go1.27.3 windows/amd64`，SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`；`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` |
| 时间 | 2026-11-15T16:28Z ~ 16:29Z（UTC） |
| 工作区 | 运行前后各有 1 个变更路径：正在写入的本目录（未跟踪）；被测代码没有未提交的改动 |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-regress.log` | Windows 原生：`go test -count=1 -v -run 'TestR154' ./internal/toolctl/ ./internal/controls/` | 退出 1（预期的失败） |
| `wsl-regress.log` | 交叉编译的 linux/amd64 测试二进制（toolctl、controls）在 WSL 中以 `-test.run TestR154` 运行（只有伪工具与接缝） | 退出 1（预期的失败） |

逐项结果（两个平台相同，除 R4-c 只在 Linux 适用）：

| 回归 | 结果 | 旧实现的实际表现 |
|---|---|---|
| R1-a `TestR154AdmitEntry` | FAIL | 在 `W − 4 s` 调用仍建立两条管道并启动伪工具（B 类，退出 0） |
| R1-b `TestR154AdmitPipeCrossesD` | FAIL | 第一条管道在 `D` 之后返回，第二条管道照样建立，伪工具照样启动 |
| R1-c `TestR154AdmitStartNotInitiated` | FAIL | 时钟过了 `W` 之后照样 `Start` |
| R2-a `TestR154CmdHeldAfterStartFailure` | FAIL | C 类的启动失败之后记录没有持有 `Cmd` |
| R2-b `TestR154PanicAfterHandover` | FAIL | 移交之后的 panic：写端没有关闭，读取任务一直等待，`Invoke` 在 20 秒的守卫内没有返回 |
| R2-c `TestR154PanicBetweenPipes` | FAIL | 第一条管道的两端保持打开 |
| R4-a `TestR154CloseRealErrorOtherSource` | FAIL | 计划的站点上真实 `Close` 返回另一个错误，旧的判定仍为命中 |
| R4-b `TestR154CloseRealErrorElsewhere` | PASS | 旧的展示行在预期为 ok 的站点显示 `error`，旧的判定已经不命中；它作为固定的拒绝预期保留（登记时预测旧实现命中，实际不是） |
| R4-c `TestR154RequestOtherKind`（Linux） | FAIL（Windows 上 SKIP） | 请求返回 EBADF 而不是 ESRCH，旧的判定仍为命中 |

- 依赖新增钩子、接缝或状态的回归（R1-d、R1-e、R2-d ~ R2-f、R3-a ~ R3-e）不能在旧实现上编译或建立前提，没有在这里运行；它们不计作“已在旧实现上动态复现”。
- R2-b 的失败使该测试进程中留下两个阻塞的读取 goroutine，随测试进程结束。
- `SHA256SUMS` 是两个日志的 SHA-256。
