# 许可证与第三方声明方案（草案）

> 状态：**待确认**（PD-10）· 2026-10-14
> 产品负责人倾向采用 Apache-2.0。在 §4 的确认项完成之前，仓库中**不添加** LICENSE，也不宣布已采用任何许可证。

## 1. 现状与影响

- 仓库已经公开，但没有 LICENSE：按默认规则，他人不能合法地使用、修改或分发这些代码，项目目前不算开源。
- G3 门槛（implementation-plan §2）只要求在**接受外部贡献之前**确定许可证，不阻塞内部开发。
- 依赖的许可证都与 Apache-2.0 兼容（2026-10-14 调查）：
  - 链接进二进制的第三方模块都是 MIT、BSD-3-Clause 或 Apache-2.0；SQLite 本身属于公有领域。
  - 没有 GPL、LGPL、AGPL 或 MPL 代码进入发行物。
  - MPL-2.0（lightningcss）和 CC-BY-4.0（caniuse-lite 数据）只出现在前端的构建工具链中，不随产品发布。
  - 依赖中的 Apache-2.0 模块都没有 NOTICE 文件，所以不需要转发上游的 NOTICE。

采用 Apache-2.0 的影响：

| 方面 | 说明 |
|---|---|
| 使用者的权利 | 可以使用、修改、分发，包括商业用途 |
| 专利 | 贡献者授予明确的专利许可；对他人提起专利诉讼时，该许可终止 |
| 分发义务 | 附带许可证全文；标注修改过的文件；保留版权声明和 NOTICE（如果有） |
| 托管服务 | **不限制**他人用本项目运营竞争性的托管服务（与 PD-11 的 Hosted 模式有关） |
| 不可撤回 | 已经按 Apache-2.0 发布的版本，不能收回授权；以后只能对新版本更换许可证 |
| 更换许可证 | 接受外部贡献之后，更换许可证或双许可需要贡献者同意；事先签署 CLA 可以保留这种可能 |

## 2. 需要添加的文件（确认之后）

| 文件 | 内容 |
|---|---|
| `LICENSE` | Apache License 2.0 的标准全文，**原样**使用，不做修改。已准备好与 apache.org 发布的 `LICENSE-2.0.txt` 逐字节一致的文本（sha256 `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`） |
| `NOTICE`（可选） | 一行版权声明，例如"Specmetry / Copyright 2026 <版权人>"。Apache-2.0 不强制要求；一旦创建，下游再分发时必须保留 |
| `README.md` | 增加"许可证"一节 |

## 3. 第三方声明

**原则**：随二进制发行的每个第三方组件，都要附带它的版权声明和许可证全文。Specmetry 是单一二进制，内嵌前端产物，所以第三方声明必须随发行物一起提供，不能只放在源码仓库中。

**需要随发行物提供的声明**（以 2026-10-14 的依赖为准；发行时按实际链接的模块重新生成）：

| 组件 | 许可证 | 进入发行物的方式 | 需要附带的文件 |
|---|---|---|---|
| Go 标准库与运行时 | BSD-3-Clause | 链接进二进制 | Go 的 `LICENSE` |
| `spf13/cobra` | Apache-2.0 | 链接 | `LICENSE.txt` |
| `spf13/pflag` | BSD-3-Clause | 链接 | `LICENSE` |
| `inconshreveable/mousetrap` | Apache-2.0 | 链接（仅 Windows） | `LICENSE` |
| `pelletier/go-toml/v2` | MIT | 链接 | `LICENSE` |
| `golang.org/x/mod`、`golang.org/x/sys` | BSD-3-Clause | 链接 | `LICENSE`、`PATENTS` |
| `modernc.org/sqlite` | BSD-3-Clause；内含 SQLite（公有领域） | 链接 | `LICENSE`、`LICENSE-SQLITE`、`LICENSE-3RD-PARTY.md` |
| `modernc.org/libc` | BSD-3-Clause | 链接 | `LICENSE`、`LICENSE-3RD-PARTY.md`（musl 为 MIT，Go 为 BSD-3-Clause，go-netdb 与 NixOS/nixpkgs 为 MIT） |
| `modernc.org/memory` | BSD-3-Clause | 链接 | `LICENSE`、`LICENSE-GO`、`LICENSE-MMAP-GO` |
| `modernc.org/mathutil`、`remyoudompheng/bigfft` | BSD-3-Clause | 链接 | `LICENSE` |
| `dustin/go-humanize` | MIT | 链接 | `LICENSE` |
| `ncruces/go-strftime`、`mattn/go-isatty` | MIT | 链接（Windows、macOS） | `LICENSE` |
| `google/uuid` | BSD-3-Clause | 链接（Linux、macOS） | `LICENSE` |
| `santhosh-tekuri/jsonschema/v6` | Apache-2.0 | `internal/contract` 接入命令之后链接（M1 起） | `LICENSE`；它内嵌的 json-schema.org 元模式的许可证需要另行核实 |
| `dlclark/regexp2` | MIT | 同上 | `LICENSE`、`ATTRIB`（移植自 dotnet/corefx，MIT） |
| `golang.org/x/text` | BSD-3-Clause | 同上 | `LICENSE`、`PATENTS` |
| React、React DOM、scheduler | MIT | 内嵌的前端产物（`web/dist`） | 各自的 `LICENSE` |
| Vite 的运行时辅助代码 | MIT | 同上 | Vite 的 `LICENSE` |

**不随发行物提供**：只在测试中使用的模块（`aymanbagabas/go-pty`、`creack/pty`、`u-root/u-root`、`golang.org/x/crypto`、`golang.org/x/term`），以及前端的构建工具链。

**生成方式**（M6 发布流程中实现）：

1. 每个目标平台用 `go list -deps` 取得实际链接的模块，从模块缓存中收集许可证文件；前端从 `package-lock.json` 取生产依赖。
2. 生成 `THIRD_PARTY_NOTICES.txt`，与 `LICENSE`（以及 `NOTICE`，如果有）一起放进每个发行包。
3. CI 增加许可证检查：发行物中出现不在允许清单（MIT、BSD-2/3-Clause、Apache-2.0、ISC、公有领域）内的许可证时失败。

## 4. PD-10 的确认项

1. 项目整体（代码、schemas、前端、文档）采用 Apache-2.0。
2. 版权人的名称（用于 `NOTICE` 和 README）。
3. 是否创建 `NOTICE` 文件。建议：创建，只写一行版权声明。
4. 外部贡献的方式：DCO（`Signed-off-by`）、CLA，或都不要。CLA 能保留将来更换许可证或双许可的可能。
5. 源文件是否加 SPDX 许可证头。建议：不加，只依靠根目录的 `LICENSE`。
6. 第三方声明的分发方式：随发行包附带 `THIRD_PARTY_NOTICES.txt`；是否同时内嵌到二进制中，使单独下载的二进制也能查看。
7. 接受"Apache-2.0 不限制他人运营托管服务"这一点，以及它与 PD-11（Local 与 Hosted 的边界）的关系。
8. 此前没有许可证的公开提交：是否在 README 中声明 Apache-2.0 也适用于这些历史版本。
9. 确认之后，在单独的提交中添加 `LICENSE`（和 `NOTICE`），并更新 README 和 PD-10。
