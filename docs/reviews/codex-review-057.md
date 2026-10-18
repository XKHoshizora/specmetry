# Codex Review 057：PR #24 在 Review 055 后的规范定向复审

审查日期：2026-10-19。审查者：Codex。

**规范技术结论：GO。** Review 055 的 B2-R2、B3-R2 及浅克隆生命周期问题均可关闭。规范具备整体冻结条件；这表示读取方案足以作为实现依据，不表示 Git Reader 已实现、实现测试已通过或负责人已授权 WP2.3。

## 一、固定身份与只读范围

- [PR #24](https://github.com/XKHoshizora/specmetry/pull/24)。
- 固定 base：`46a4b26855cc582b6387883260433c7badb1f2f9`。
- 固定 HEAD：`61c69736613323bc7c8e7ec17a56d2b48e1e0ff7`。
- Review 055 原对象：`5d707eb28e95631cf97808a3287d81375fd0e60f`，其固定 base 为 `a236e5350ff92dcffd9bc05d09b1f3aac3996704`。
- 已完整阅读归档 [Review 055](codex-review-055.md)，保留关联、seq、事务、严格遍历、过滤器判据及其他已成立结论，仅检查新差异与必要调用顺序。

开始和收尾核对远程 HEAD/base，均为本轮固定身份；PR 为 Open。最终 mergeable=true、mergeable_state=clean，不再沿用上轮“尚有文档冲突”的状态，也没有替负责人合并。

相对新 base 共十个文档文件：WP2.3 规范、data-model、architecture、acceptance-criteria、implementation-plan、m1-exit-evidence、review-resolution 及 Review 050／052／055 归档；没有生产代码、测试、Schema 或迁移 SQL 修改。相对原目标的测试夹具变化来自已合入的新 master，本轮不重审 PR #26。

## 二、BLOCKER 与逐项裁定

**本轮没有剩余实质性 BLOCKER。**

### B2-R2：包与索引双向配对——关闭

位置：[WP2.3 规范 §5.2，第 445～451 行](https://github.com/XKHoshizora/specmetry/blob/61c69736613323bc7c8e7ec17a56d2b48e1e0ff7/docs/wp2.3-git-reader.md#L445)。

新规则检查 pack→idx 和 idx→pack 两个方向，随后验证 idx 校验和及其记录的 pack 校验和与包尾匹配。允许已识别的伴随文件与 multi-pack-index 文件；不允许残留临时包、临时索引、没有对应 pack 的 rev 等不完整布局。alternates 递归适用同一检查；目录不可列出、文件不可读或无法确认时返回 exit 6，不发布结果或推进游标。

这不是“多读总是安全”的论证，而是对原反例的明确拒绝：pack 存在、idx 缺失不能被当成正常回收。检查只读，不能重建索引、修复包或调用可能写原库的维护命令。

**本轮独立复验**：Windows，Git 2.56.0.windows.2，仓库外合成夹具，无真实远端、无传输。使用独立的窄范围存储规则见证，不是尚未实现的生产分类器。

| 情形 | Git 实际结果 | 新规则的独立见证 |
|---|---|---|
| 正常成对 pack／idx | 对象可读，exit 0 | 通过 |
| 移走 idx、保留 pack | batch-check 输出 missing；裸 ID cat-file -e 为 1；两者 stderr 均空 | 拒绝；检查前后原 .git 内容及 mtime 不变 |
| 对象位于 alternate，移走其 idx | 同样 missing／1，无 stderr | 递归检查拒绝 |
| 恢复原 idx | 主对象库、alternate 均重新可读 | 均通过 |
| tmp_pack 文件或孤立 rev | 不依赖 Git 是否输出 stderr | 均拒绝 |
| 合法 reset／reflog expire／显式 gc 后的旧孤立提交 | missing／1，无 stderr | 存储布局完整，正常回收路径通过 |

没有把主动制造的故障或显式回收写入误计为“读取发生写入”。独立见证仅核查本轮配对／校验和规则，不声称穷尽验证所有 pack 内部损坏。作者 K18～K22 的 Windows／WSL 记录仍是作者证据；其中 bitmap、rev、keep 的组合正例没有被本机这次运行独立全量重放。

### B3-R2：任何索引读取之前的门禁——关闭

位置：[§4.3 读取顺序，第 209～253 行](https://github.com/XKHoshizora/specmetry/blob/61c69736613323bc7c8e7ec17a56d2b48e1e0ff7/docs/wp2.3-git-reader.md#L209)。

新顺序为：取得元数据、复制私有目录及索引 → Specmetry 直接解析索引副本 → 仅普通布局才运行 status／ls-files／check-attr／diff-files。

- 建立私有目录的引导只使用不读取索引的 rev-parse、config；不是先调用 status 或 ls-files 再决定是否安全。
- mode 040000 或 sdir 即 unknown；不能完整解析即 exit 6；两种情况都停止该仓库及其后续子模块命令。
- 超级项目 status 固定 --ignore-submodules=all；本轮还明确 diff-files 必须带同一限制，防止它在子模块门禁之前进入子模块索引。
- 子模块 gitfile 先解析实际 GitDir，以受控元数据调用取 HEAD；然后建立它自己的私有副本、先过自己的门禁，才运行它自己的状态命令。无初始化的子模块不额外读取。
- 每次对象读取继续受 PD-26 与受控环境保护，私有 GitDir 不能绕过安全能力门禁。

**独立 mtime 对照**：使用相同的真实 sparse index 和稀疏目录同名的工作区目录，按旧顺序运行私有 status，复现三个原树对象 mtime 改变、内容不变；按新顺序先解析得到 sdir／040000，后续 Git 调用数为 0，原 .git 所有文件的内容与 mtime 均不变。普通索引正例继续运行私有 status，报告实际修改 ` M f`，原 .git 不变。截断索引副本被独立解析见证拒绝。

这里验证的是方案顺序的可行性和原反例，不冒充未来 Go 调用链已经实现了门禁。子模块的动态 mtime 组合本轮未在本机重放，静态顺序检查与作者 S18b／X6 记录分别标注；实现必须用调用记录和真实布局同时验证。

## 三、直接索引解析设计与实现义务

规则方向与 Git 官方格式一致：版本 2／3 的条目对齐、版本 3 扩展标志，以及版本 4 的前缀压缩和不填充规则必须分别处理；扩展需按长度界定，不能用路径字节搜索 sdir。小写必需扩展不能当成可忽略的未知扩展。split index 的 link 指向共享索引，须检查两份实际条目；共享索引错误不能退化为“没有稀疏项”。[Git 索引格式](https://git-scm.com/docs/gitformat-index)

按官方格式，v4 的前缀整数沿用 OFS_DELTA 编码；link 的全零哈希表示没有共享文件，不应寻找 sharedindex.000…；索引尾部全零也可以是 Git 有意省略哈希，须与非零错误校验和区别处理。[Git 索引格式](https://git-scm.com/docs/gitformat-index)、[Git 配置 index.skipHash](https://git-scm.com/docs/git-config/2.55.0)

上述是“完整解析、不能解析就失败”的具体工程落实，不新增产品取舍。实现至少要兑现：

1. 条目计数、整数、前缀回退、路径终止、对齐与扩展长度不能越界或溢出；区分可跳过的可选扩展与不支持的必需扩展。
2. link／共享索引的引用、边界和必要内容有效，缺失、损坏或无法完整理解时失败；不能只扫描主索引而遗漏共享条目。
3. sdir 或任一主／共享索引中的稀疏目录条目都先停止，不运行可能展开索引的 Git 命令。即使更复杂布局暂不支持，也应安全拒绝，不能猜测成普通索引。
4. 对直接读取、复制失败及中途变化保守处理，输出安全错误、清理私有内容，不修复原索引。

本轮独立格式见证实际读取 Git 生成的 v2、带 skip-worktree 的 v3、v4、split 主／共享索引，以及真正的 sparse index；普通格式通过，稀疏格式被识别，截断拒绝。这个窄见证没有实现完整 split-index EWAH 重建，不可写成完整解析器验证；作者的 Python 原型也不能代替 Go 实现和 CI 回归。

### NON-BLOCKER N1：将“主索引不存在”的正常分支明确写进实现测试

位置：§4.3 第 1～2 步、第 253 行失败总述，以及第 1030 行解析测试计划。

本机新建空仓库的 .git/index 确实不存在，正常 Git status 仍成功。规范已经要求 unborn 仓库照常读取，但未单列 ENOENT 的分支。实现时应按空索引语义处理真正不存在的主索引，不能把它与零字节／截断索引、缺失的被引用共享索引、权限／I/O 错误混同；后几种仍须 fail-closed。

这是已有正常仓库语义下的工程分支和测试补充，不是允许未知格式继续运行 Git，也没有发现由新设计必然导致的 false-safe 路径，因此不阻塞规范 GO。不需要为此新增改产品语义的决定。建议在实现首个解析器用例时明确该分支，并覆盖空仓库、已有 HEAD 但没有 index、零 link 哈希及缺失 shared index 的对照。

## 四、浅克隆生命周期与已批准范围

位置：[§5.2，第 568～590 行](https://github.com/XKHoshizora/specmetry/blob/61c69736613323bc7c8e7ec17a56d2b48e1e0ff7/docs/wp2.3-git-reader.md#L568)。

P1、P2、P3 和 worktree list -z 能力要求已经由负责人明确批准，本轮不再列为待决定。它们不提高所有命令的 Git 最低版本，也不批准错误 clean 或原库写入。

检测直接 lstat 公共 GitDir 的 shallow 项：只有确定不存在才继续；存在任何类型或无法确认都 exit 6。关联 worktree 必须取共同 GitDir，不各自检查错误的私有位置。拒绝期间不写提交、关联、seq、游标，不自动 fetch，不返回成功空扫描。

在读取阶段固定 GIT_SHALLOW_FILE 指向不存在的私有路径，使检测后新出现的真实 shallow 文件不能截断遍历；真正缺失父对象时，严格 log 失败，部分结果丢弃。它不声称为所有外部文件系统变化提供全局原子锁。

**本轮独立无网络模拟**：

- 完整四提交仓库：检测为完整，读出 4 个提交。
- 添加本地 shallow 边界：检测拒绝；原生 log 只能读出 3 个，固定私有 GIT_SHALLOW_FILE 后读出 4 个。
- 移走真实父提交：固定路径下 log exit 128，不能把部分输出当成完整成功。
- shallow 改为目录：直接检测拒绝；Git 原生 --is-shallow-repository 反而返回 false／0，支持不依赖该布尔查询的设计。
- 关联 worktree 的 --git-common-dir 指向相同公共目录。
- 移除边界恢复完整：无旧游标时完整范围为 4 个；已有第三个提交游标时增量为 1 个。

因此原来的“浅边界上建立 T 游标，加深后永远漏掉 T 之前新增可见历史”的生命周期缺口被消除：浅克隆期间根本不发布这种游标。完整仓库已有游标的情况仍按既有增量／改写规则处理。异常 shallow、检测错误不能偷偷创建游标。

以上是本地状态模拟和范围验证，不是真实网络 deepen、不是真实数据库扫描事务。实际无写入、并发改写后拒绝和游标状态断言仍属于实现集成测试义务。

## 五、整合、同步与历史保留

`7c7da3454ac3151266d0e621fc7ce612afb049b9` 的父提交为旧目标 5d707eb 和 master 46a4b26。独立逐节比较：原分支 §62、§63、§64 与最终文本一致；master §65 完整保留。Review 053 的 blob 与 master 相同，Review 055 的 blob 与首次归档 de54f7a 相同。没有因解决追加位置冲突而删改既有决定或历史身份。

§66 准确记录 Review 055 的 NO-GO、固定身份、实验边界、负责人新决定及修订；状态仍为“修订，待复审”，没有由实施方提前宣布关闭。

data-model、architecture、MVP-EV-16／EV-18、MVP-WT-06 与实施计划同步了双向配对、门禁前置、unknown 和浅克隆生命周期。接口注释及测试规划保留相同拒绝语义，未改已验收的进度、ready 或关联生命周期。§62 正式批准的关联限制仍是既有边界，不能用它豁免可识别存储故障或对象 mtime 修改。

**PRODUCT DECISION：本轮没有新增必须由负责人裁决的产品语义。** P1／P2／P3／-z 已获批；其他不在本轮内的未批准草案仍保留原状。规范冻结与实现授权仍由负责人控制。

**ACCEPTED RISK：仅保留已有 §62 关联生命周期限制，没有新增接受风险。** M-14、M-15 及其余维护事项未关闭。

## 六、CI 与验证边界

[最终 run 39144293701](https://github.com/XKHoshizora/specmetry/actions/runs/39144293701)，Attempt 1，HEAD 为本轮 61c6973，八个作业全部 success：Web、Go lint、三平台 Go tests、三平台 dashboard 构建。

逐个作业日志的实际检出均为 PR 合并快照 `f5510c5ab7321943d1cd082cc63d3a2ffb2ffc55`，父提交依次为 base `46a4b26`、HEAD `61c6973`；树 `07b29b46601580ce73a70e8cb25546fa93dc5fcf` 与 HEAD 相同。Ubuntu Test 步骤使用 -race，Linux 专属 root 步骤成功，其他平台按设计跳过该步骤。

这证明目标文档树对应的既有仓库检查通过，不能证明仓库外 K／S／X／H 实验已经三平台运行，更不能证明新 Go 解析器或扫描层已经存在。原 run 39121984050 的 Ubuntu 失败、后续修订和 PR #26 的独立处理没有被本次绿色覆盖或改写。

独立运行平台只有 Windows，Git 2.56.0.windows.2，Python 3.14.1；无真实远端、无传输或 Helper 实验。临时夹具首次准备遇到 Windows 只读对象文件属性和 alternates 文件的 CRLF 格式问题，更正隔离夹具后才取得有效结果；未把准备失败当作规范反例。

本轮未独立重跑 WSL／Linux 2.43、macOS、Git 2.32、真实部分克隆、网络加深、真实并发 gc、权限／I/O 全矩阵、长驻过滤器或 LFS。相应作者记录仅作为标明来源的可行性证据。过滤器判据、严格遍历等未变结论复用 Review 055，不重新做整包安全实验。

实现阶段必须兑现：索引格式和失败分支测试；超级项目／子模块门禁顺序的调用记录；真实文件内容与 mtime 快照；pack／idx／alternates 正反例；shallow 检测与恢复生命周期；严格读取失败后的事务、seq、游标不变。之后的命令呈现仍属 WP2.7，E2E 属 WP2.9，不被本次文档 CI 提前验收。

## 七、最终回答与接手建议

1. Review 055 B2-R2：**关闭**，双向配对不再在空 idx 集合上错误通过。
2. Review 055 B3-R2：**关闭**，直接解析门禁前置，超级项目不再先进入子模块索引；原 mtime 反例与正对照支持这一设计。
3. 浅克隆生命周期：**关闭准备报告层面的缺口**，P3 与完整恢复后的扫描规则一致，实际游标集成仍待实现验证。
4. PR #24：**技术 GO，可以进入负责人控制的合并／冻结收尾**；规范具备整体冻结条件，保留 N1 的工程测试补充。
5. 本轮没有新的产品待决事项；不能把规范 GO、既有 CI 或作者 Python 原型视为 WP2.3 已实现或开工授权。
6. 下一步由负责人确认冻结并另行决定是否授权实现；实施方随后按已冻结的顺序、格式与失败边界开发和补齐三平台回归，不需再重开已关闭部分的全面审查。

本轮只新增本报告和同时获准的 Review 056 报告；未修改原有文件、提交、推送或合并，不关闭 M-14／M-15，不启动 WP2.3。
