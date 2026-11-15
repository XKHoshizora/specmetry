# 本机验证 4：Review 154 修复之后的代码点

| 项 | 值 |
|---|---|
| 代码点 | `a0ed8dc0800bc64b8f1c88e715f00fa600d053fd`，树 `9d120b675a08a18ede3fdd2b647c7839c971840e`，工具模块子树 `76a5c49c037b279ba783913cc95a56dd6e93c9eb` |
| 工具链 | `go1.27.3 windows/amd64`，SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`；`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` |
| 时间 | 2026-11-15T16:46Z（UTC） |
| 工作区 | 运行前后各有 2 个变更路径：未跟踪的 `local-3/` 与正在写入的本目录；被测代码没有未提交的改动 |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-unit.log` | Windows 原生：`go test -count=1 -v ./...` | 退出 0；506 条 `--- PASS`（顶层 228，子测试 278），0 条 `--- FAIL`，2 条 SKIP：`TestR154RequestOtherKind`（只在 Linux）与 `TestTakeNegatives/symbolic_link`（本机没有创建符号链接的权限） |
| `windows-static.log` | 三个 GOOS × 三种标签集合的 `go vet`，`gofmt -l`，`go mod verify`，正式入口只编译 | 都退出 0；`gofmt -l` 没有输出 |
| `wsl-linux.log` | 交叉编译的 linux/amd64 测试二进制在 WSL（内核 6.18.33.2-microsoft-standard-WSL2）中运行（只有伪工具与接缝） | 12 个包都退出 0；508 条 `--- PASS`（顶层 229，子测试 279），0 条 `--- FAIL`，没有 SKIP；LC-9 的 20 个变体命中 |

- 测试的数目：顶层测试 229 个、子测试 279 个（Windows 上其中顶层 1 个、子测试 1 个为 SKIP）。第三次专用运行的代码点 `ed84d09` 是顶层 210、子测试 277（共 487 条 PASS）；增加的 19 个顶层测试与 2 个子测试都是 Review 154 的回归。
- 这些不是三平台的证据，没有 `-race`，也没有运行 RC-0 ~ RC-4（本机不执行采集器的真实 Go 调用）。
- `SHA256SUMS` 是三个日志的 SHA-256。
