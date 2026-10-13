**结论：GO。B1、B2 的核心设计可以关闭；没有发现需要继续阻止核心开发的问题。** 全程未修改文件。仓库仍无业务代码，本次确认的是设计闭合，运行时验证留待实现验收。

## BLOCKER

无。

- **B1 已明确区分两类证据。** `observed` 只保证开始和结束两个时刻；`snapshot_isolated` 要求私有快照、外部强制只读、执行前探测和执行后验证，文件权限位不算隔离。[等级定义](C:/Users/hoshi/Desktop/specmetry/docs/data-model.md:360)
- **状态规则落实了等级限制。** MVP 不产生隔离级证据，因此不能自动沿用验收；观测级正常验收必须由人确认风险。隔离不可用时，`required` 拒绝执行；未来的 `auto` 明确降级并记录原因。[沿用规则](C:/Users/hoshi/Desktop/specmetry/docs/data-model.md:508)、[降级规则](C:/Users/hoshi/Desktop/specmetry/docs/data-model.md:659)
- **B2 已补齐并发保护。** `seen_decision_id` 必填，读取模型返回 `current_decision_id`；比较与插入必须在同一写事务中完成，过期返回退出码 5／409，不写入、不自动重试。[事务规则](C:/Users/hoshi/Desktop/specmetry/docs/data-model.md:762)

实现时应确保比较前已取得写锁，例如使用 `BEGIN IMMEDIATE`，并用两个独立连接同时竞争同一个决定版本验证。普通延迟事务先读取可能遇到旧快照升级失败，不能仅凭“放在事务里”就宣称并发已验证。[SQLite 官方说明](https://www.sqlite.org/isolation.html)

## NON-BLOCKER

新增反例覆盖了原漏洞：CX-26/27 覆盖修改后恢复与 HEAD 往返；CX-28 覆盖无法保证隔离；CX-08(e)(f) 和 ACC-11 覆盖旧接受覆盖新拒收及重新读取后的明确改判。[反例](C:/Users/hoshi/Desktop/specmetry/docs/acceptance-criteria.md:310)

还有两处小范围同步事项：

1. **`gaps.v1` 缺少证据等级。** 它允许返回 `delivery_status = ready`，却拒绝添加 `evidence_grade`，与“展示 ready 时同时展示等级”的规则不一致。实测确认了这一点。补齐字段、示例和负例即可。[schema](C:/Users/hoshi/Desktop/specmetry/schemas/gaps.v1.schema.json:8)、[展示规则](C:/Users/hoshi/Desktop/specmetry/docs/data-model.md:640)
2. **修正反例的夹具和预期。** CX-08 从无决定开始，拒收成功、旧接受失败、重新接受成功，应保留**两条**决定。CX-26/27 应明确其他快照已有验收的前置；若唯一 AC 尚未重新验收，目标级 `evidence_grade` 应为 `none`，人工正常验收后才为 `observed`。[CX-08](C:/Users/hoshi/Desktop/specmetry/docs/acceptance-criteria.md:310)、[CX-26/27](C:/Users/hoshi/Desktop/specmetry/docs/acceptance-criteria.md:327)

结构校验通过：**14 份 schema 可编译，21 个合法示例通过，62 个非法示例被拒绝。** 主要写入协议、API 与推导规则一致，上述 `gaps` 遗漏需要同步。

## ACCEPTED RISK

MVP 的 `ready` 表示**人在明确确认观测级风险后完成验收**，不能解释为隔离快照已验证。中途修改后恢复仍可能被人工接受，但系统已停止夸大证据保证，也禁止据此自动沿用。

**最终结论：GO。** 按既有门槛记录 PD-22 的产品确认，并在 M2/M5 完成证据等级与并发实现验收。
