# 第二阶段的本机验证 1（代码点 `267012b`）

| 项 | 值 |
|---|---|
| 代码点 | `267012b71ea149e036038e3bc7d3eafa4e08b1c7`，树 `6dfff00592d460c537654ced835f1d810cd48491`，工具模块子树 `bb314889c4788e527dfc7e84fd2b52095957fdaa`；运行前后工作区都干净（0 个改动的路径） |
| 工具链 | `C:\Users\hoshi\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.3.windows-amd64\bin\go.exe`，`go1.27.3 windows/amd64`，SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`；环境 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` |
| 时间 | 2026-11-15T05:13Z ~ 05:15Z（与测试 workflow 的 run 43680105408 同时） |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-unit.log` | Windows 原生：`go test -count=1 -v ./...`（工具模块，不含带标签的真实调用包） | 退出 0；486 个 `--- PASS`，0 个 `--- FAIL`；1 个 SKIP：`TestTakeNegatives/symbolic_link`（本机没有创建符号链接的权限，第一阶段已登记） |
| `windows-static.log` | 三个 GOOS × 三种标签集合的 `go vet`，`gofmt -l`，`go mod verify`，正式入口只编译 | 每一项都退出 0；`gofmt -l` 没有输出；`all modules verified` |
| `wsl-linux.log` | 每个包交叉编译为 linux/amd64（`CGO_ENABLED=0`）的测试二进制，在 WSL（内核 `6.18.33.2-microsoft-standard-WSL2`）中以 `-test.v` 运行；只有伪工具与接缝，不含真实调用 | 12 个包都退出 0；487 个 `--- PASS`，0 个 `--- FAIL`，没有 SKIP；LC-9 的 20 个变体（含 9 个 pidfd 的变体）都命中 |

- 这些是开发机上的运行，不是三平台的证据，也没有 `-race`（本机不可用）；三平台的证据以测试 workflow 为准。
- 本机没有运行 RC-1 ~ RC-4，也没有构建带 `c9evidence_realcall` 标签的测试二进制去运行它们。
- `SHA256SUMS` 是三个日志的 SHA-256。
