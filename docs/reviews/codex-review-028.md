# Codex Review 028：WP1.7 最终定向复审

- 审查日期：2026-10-16（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #10](https://github.com/XKHoshizora/specmetry/pull/10)
- 目标提交：`58659cd025395bb48854a4c743a73a28d8965eee`
- 增量基准：`78ca8826daaee8bba04a68f462e9af8bd8dbef64`
- 上轮报告：[Codex Review 027](codex-review-027.md)
- CI：[GitHub Actions 38803933170](https://github.com/XKHoshizora/specmetry/actions/runs/38803933170)
- 范围：仅复审 Review 027 的 B1、B2、B3 及直接回归，不重新全面审查已通过的 Question 状态转换、并发保护、数据库持久化和 Schema 业务语义。
- 方法：实际源码与契约静态审查、三平台 CI 日志核查、隔离副本中的旧版与新版对照测试。

**结论：GO。** Review 027 的 **B1、B2、B3 均可关闭**；未发现本轮修改直接引入的实质性 BLOCKER。

原仓库在审查期间保持干净，未修改、提交、推送或合并。

## BLOCKER

无未关闭项。

| 原问题 | 复审结论与证据 |
|---|---|
| **B1：安全终端显示** | **关闭。** [termtext.go:28](../../internal/termtext/termtext.go#L28) 转义 ESC、其他 C0、DEL、C1、双向控制字符及非法 UTF-8 字节。Question 列表、确认摘要和最终结果均接入转义。公开 CLI 测试覆盖问题、上下文、回答、驳回理由和显示名；JSON 解码及数据库中的原始业务值保持不变，取消确认不写入。 |
| **B2：未知子命令** | **关闭。** [question.go:65](../../internal/cli/question.go#L65) 明确拒绝未知子命令，只建议可信的现有命令名。`unknown`、`answr` 和合成敏感子命令均返回 `invalid_input / exit 2`，不回显参数；JSON 模式输出合法 `error.v1`。裸命令与显式帮助正常返回 0。 |
| **B3：JSON 错误模式** | **关闭。** [output.go:62](../../internal/cli/output.go#L62) 与 [root.go:53](../../internal/cli/root.go#L53) 正确衔接：仅对解析未完成的 Question flag 错误重新识别 JSON 请求。独立参数矩阵覆盖前后顺序、布尔值、重复 flag、取值参数、简写、未知 flag 和 `--` 边界，未发现误识别或参数回显。 |

CRLF 与缩进的组合没有形成新的视觉边界漏洞：`A\r\nB` 经 `indented` 成为 `A\r\n    B`，随后规范化为 `A\n    B`，续行仍保持缩进。独立检查通过。

## NON-BLOCKER

没有新增项。其他 CLI 参数回显、其他命令文本未转义，以及 WP1.6 的既有 NON-BLOCKER，继续按已登记计划处理；本轮未将它们视为已修复或已接受风险。

## ACCEPTED RISK

没有新增风险接受。通道识别仍沿用已批准的防误操作边界，不构成身份认证。

## 验证证据

- **目标提交：**9 项 Question CLI 测试全部通过，包含真实 PTY、确认期间竞争、取消及持久化断言；终端转义的 3 项单元测试通过。
- **独立补测：**46 组公开 `Execute` 参数组合及 CRLF 缩进检查全部通过。JSON 错误均检查完整 Schema、`error.code`、stdout 和 stderr，不依赖 Pointer 是否存在。
- **旧提交对照：**在隔离的 `78ca8826` 副本重放同一组断言，实际复现原始控制字符输出、未知子命令退出码 0，以及错误 flag 后缺少 `error.v1`。失败发生在被测行为，而非夹具准备阶段。
- **CI：**已独立核对[指定运行](https://github.com/XKHoshizora/specmetry/actions/runs/38803933170)，对应目标提交，8/8 成功。Windows、Ubuntu、macOS 日志均显示新增修复测试实际通过；Ubuntu 启用 Race Detector。本地验证在 Windows 完成，其他平台依据 CI 日志。
- 首次受限沙箱运行无法创建临时 Git 夹具，未计为代码失败；隔离环境重跑成功。控制字符测试仅使用内存或日志文件，没有向真实终端发送控制序列。

## 最终判断

**B1、B2、B3 全部关闭；未发现直接回归。WP1.7 可以正式验收，PR #10 可以合并。最终结论：GO。**

未执行合并，未开始 WP1.8。本文件为审查完成后按用户要求保存的完整复审意见。
