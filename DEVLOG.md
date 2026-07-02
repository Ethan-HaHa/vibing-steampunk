# DEVLOG — vibing-steampunk 个人修改记录

> 记录用户（Ethan-HaHa）对 vibing-steampunk 的使用与修改决策。
>
> **职责分工**：
> - `git log` 记录"改了什么、什么时候、谁改的"（事实层）
> - 本文件记录"**为什么改**、踩过什么坑、有什么约定"（决策层）
>
> 写法约定：
> - 一条修改一个段落，标题 `## YYYY-MM-DD · 简述`
> - 不确定的用 `**TODO**:` / `**疑问**:` 标注
> - 过期内容用 `~~删除线~~` + 说明何时被取代，不要直接擦掉
>
> **安全**：本文件位于公开仓库，禁止出现真实 SAP 用户名、主机、传输号、客户命名空间。一律用 `TESTUSER`、`TR-EXAMPLE`、`ZDEMO_*` 等合成占位。

---

## 当前工作状态（每次会话开始时校准）

- **仓库位置**：`D:\GitHub\vibing-steampunk\`
- **Git remotes**：
  - `origin` → `https://github.com/Ethan-HaHa/vibing-steampunk.git`（你的 fork，可 push）
  - `upstream` → `https://github.com/oisee/vibing-steampunk.git`（原作者，跟踪上游更新）
- **运行二进制**：`D:\tools\vsp.exe`（与源码解耦）
  - 改源码后需 `go build -o D:/tools/vsp.exe ./cmd/vsp` 才能生效到运行环境

校准命令：
```bash
git -C D:/GitHub/vibing-steampunk status
git -C D:/GitHub/vibing-steampunk remote -v
git -C D:/GitHub/vibing-steampunk log --oneline -10
```

---

## 使用与修改历史

### 2026-07-02 · 验证历史版本读取功能（使用，非源码改动）

**动机**：需要读取一个 ABAP 函数模块（占位记为 `ZDEMO_FUNC`，函数组 `ZDEMO_FGROUP`）的历史版本 33 的源码，用于对比历史实现 / 排查回归。这是 vsp MCP 的实际使用场景，记录下来作为该功能可用的验证证据，并为日后调用提供模板。

**做了什么**：
1. 调用 MCP 列出函数的全部历史版本：
   ```
   SAP(
     action="revisions",
     target="FUNC ZDEMO_FUNC",
     params={"parent": "ZDEMO_FGROUP"}
   )
   ```
   - 返回该函数的全部版本历史（N 个版本，从最旧 `00001` 到当前活动版本 `00000`）
   - 每个版本包含：`uri`、`version`、`versionTitle`、`date`、`author`、`transport`
   - 版本按时间倒序排列（最新在前）
2. 从结果中找到目标版本的 `uri`，形如：
   ```
   /sap/bc/adt/functions/groups/<FOBGROUP>/fmodules/<FMONAME>/source/main/versions/<TIMESTAMP>/<VERSION_PADDED>/content
   ```
3. 调用 MCP 取该版本源码：
   ```
   SAP(
     action="revision_source",
     params={"version_uri": "<上一步拿到的 uri>"}
   )
   ```
   - 返回该版本的完整 ABAP 源码

**验证结论**：
- ✅ `revisions` action 工作正常，能列出函数的所有历史版本
- ✅ `revision_source` action 工作正常，能根据 version_uri 取任意历史版本源码
- ✅ 对 FUNC 类型，`parent` 参数是必需的，值是函数组名
- ✅ 即使多个版本共用同一个版本树时间戳（`<TIMESTAMP>` 段），各自的 `<VERSION_PADDED>` 仍能唯一定位

**关联 commit**：无（仅使用，无源码改动）

**注意事项 / 给未来的提示**：
- 若要比较两个版本之间的差异，可用 `compare_versions` action，传 `version1_uri` 和 `version2_uri`（后者可用 `"current"`）
- URI 路径中的 `<TIMESTAMP>` 段（如 `20260624062337`）是版本树创建时间，**不等于**单个版本的修改时间 —— 单个版本的修改时间在 `date` 字段
- 版本号格式是 5 位零填充字符串（`00033`），不是整数

---

## 修改模板（复制即用）

```
### YYYY-MM-DD · 一句话标题

**动机**：为什么要改？业务诉求 / 修复 bug / 扩展功能

**做了什么**：
- 改了 X
- 改了 Y

**关联 commit**：`<sha>` 或 PR 链接

**踩坑/注意事项**：
- （如果有）什么坑下次别再踩

**遗留 TODO**：
- （如果有）
```

---

## 已知坑点 / 跨次决策（待累积）

> 把"多次踩到"或"反复需要决定"的坑放在这里，避免每次重新思考。

- **tracked 文件必须脱敏**：DEVLOG/reports/docs/测试 中禁止出现真实 SAP 标识（用户名、传输号、主机、客户命名空间）。一律用 `TESTUSER`、`TR-EXAMPLE`、`ZDEMO_*` 替换。详见仓库 `CLAUDE.md` 的 Sanitize Policy 章节。

---

## 待办清单（候选工作）

> 用户提到"后续我可能会进行添加和修改某些方法"。具体清单等用户提出后再补。

- _（暂无）_
