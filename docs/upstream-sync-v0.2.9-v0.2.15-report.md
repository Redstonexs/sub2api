# 上游同步交付记录：v0.2.9 → v0.2.15

- **仓库**：`Redstonexs/sub2api`（fork of `Wei-Shaw/sub2api`）
- **交付 commit**：`168742591`（`main`）
- **状态**：本地完成并验证通过；**未推送**（按约定等维护者确认）

## 1. 目标

把上游 7 个 release 逐个按 tag `--no-ff` 合并进 fork `main`，在保留 fork 特色行为的前提下
完整吸收上游的功能与安全修复，并让 `main` 保持可用。

## 2. 范围

合并的 tag：`v0.2.9` `v0.2.10` `v0.2.11` `v0.2.12` `v0.2.13` `v0.2.14` `v0.2.15`。

不在范围内：推送到 `origin/main`；改动 Go module path；重新引入上游推广段落；
重写 fork 专属服务；顺手修复与本同步无关的既有问题。

## 3. 合并序列

| Commit | Release | 非白名单冲突数 |
|---|---|---|
| `349e101aa` | v0.2.9 | 6 |
| `4ea00ead6` | v0.2.10 | 3 |
| `59b2e1801` | v0.2.11 | 1 |
| `37897744d` | v0.2.12 | 10 |
| `42ef7a436` | v0.2.13 | 0（仅白名单 `VERSION`） |
| `d2bb4eb9a` | v0.2.14 | 11 |
| `6b3929d64` | v0.2.15 | 10 |
| `d074d40e7` | — | 状态文件补齐到 v0.2.15（原为滞后的 v0.2.7） |
| `168742591` | — | 修正 v0.2.15 中一处 i18n 键误判（见 §6） |

`.github/workflows/upstream-sync.yml` 的自动化在第 1 步就会 `exit 10`（6 个冲突落在
fork 白名单之外），因此本次为手工逐 tag 合并。

## 4. 验收标准与结果

| # | 标准 | 结果 |
|---|---|---|
| 1 | 7 个 tag 均为 HEAD 祖先 | ✅ 7/7 |
| 2 | 每个 tag 一个独立 merge commit | ✅ 见 §3 |
| 3 | fork 特色保留：`VERSION` 不回落 0.2.x（=1.9.0）；白名单文件为 fork 版本；README/install 指向 `Redstonexs/sub2api` | ✅ 三份 README `Wei-Shaw` 出现次数为 0 |
| 4 | 上游修复不丢：fork 机制承载上游 beta 行为，且上游测试通过 | ✅ 见 §5 |
| 5 | 验证全绿 | ✅ 见 §5 |
| 6 | `.github/upstream-sync-tag` = `v0.2.15` | ✅ |
| 7 | 工作树干净、无冲突标记 | ✅ 0 改动、0 标记 |

## 5. 验证证据（SHA `168742591`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| 后端单元 | `go test -tags=unit ./... -count=1`（go1.27.2，`-mod=readonly`，pristine `go.sum`） | ✅ exit 0，59 包 ok / 0 fail |
| 前端门禁 | `make test-frontend` | ✅ exit 0，37 文件 / 647 测试 |
| 前端类型 | `pnpm run typecheck` | ✅ exit 0 |
| 前端构建 | `make build-frontend` | ✅ exit 0 |
| 发布式构建 | `go build -tags embed ./cmd/server` | ✅ exit 0，169,716,798 B |
| 部署脚本测试 | `deploy/tests/*.sh`（6 个） | ✅ |
| gofmt | 本次解冲突涉及文件 | ✅ 无输出 |

> 注：后端证据在 `d074d40e7` 上取得；`d074d40e7..168742591` 之间只有
> `frontend/src/views/admin/GroupsView.vue` 一个文件变化，无 Go 侧改动，故仍然成立。
> 前端门禁在 `168742591` 上重跑通过。

**上游测试未被削弱**（逐项核对）：

- `gateway_structured_outputs_beta_test.go` 与上游 v0.2.9 **逐字节一致**；
- `gateway_tool_changes_beta_test.go` 与上游 v0.2.15 **逐字节一致**，8 个子用例全通过；
- `SettingsView.spec.ts`：上游 44 个用例名**一个不缺**，另含 fork 的 17 个额外用例；
- 7 个 merge commit 中**没有任何** `*_test.go` / `*.spec.ts` 被删除。

## 6. 关键取舍（保 fork 特色 vs 不丢上游修复）

1. **beta 继承**：上游对 structured-outputs、mid-conversation 工具变更、inline tools
   逐项开窄口子；fork 统一由 `claude.ClientCapabilityBetas` 承载。三个 token 并入 fork
   清单后上游行为等价落地，管理端 drop set 仍优先。
2. **管理员账号（v0.2.14 安全版）**：采纳上游"不再回退可猜测默认邮箱"
   （4 个 compose 变体 + `docker-deploy.sh` 生成随机登录邮箱）；保留 fork 更严的
   `ADMIN_PASSWORD` 必填与"凭据不回显、只落 mode-600 `.env`"。
3. **充值码**：保留 fork 的单次写入。上游是在同一 ent builder 链上
   `SetBonusAmount` + `SetRechargeCode("")` 再事后补写，照抄会因 last-wins 清空充值码、
   破坏支付履约。
4. **Go 工具链**：`go.mod` 随上游升到 1.27.2，配套把 `backend-ci` / `release` /
   `security-scan` / **`release-cache`**（未冲突但同样钉了旧字面量）统一为 `go1.27.2`。
5. **平台分层（v0.2.15）**：上游把 `/v1/responses` 的 Anthropic 转换统一为
   `anthropicReasoningEffort`（显式 `thinking.type=disabled` 优先于
   `output_config.effort` 并产出 `none`），它落在 fork 的 `resolveUpstreamProtocol` /
   provider-profile 分层之上；`buildUpstreamRequest` 仍保持单一入站实现。

### 已发现并修正的自身缺陷

`168742591`：合并 v0.2.15 时我用"按列号推断缩进"的方式核对了语言包结构，错误地判定
`admin.groups.accountsCount` 不存在，据此把上游的键改成了 fork 的 `accountCountLabel`
并写下错误注释。实际上两者都存在且文案相同，且 `ProxiesView.vue` 已有两处
`admin.groups.accountsCount` 调用，属既有约定。已恢复上游的键并修正注释。

成因值得记录：`localeKeyCompleteness.spec.ts` 只扫描 `keypath` / `titleKey` /
`descriptionKey` 三类**静态引用**，不会覆盖 `t('...')` 字面量，所以这类错误
测试抓不到，只能靠人工核对语言包。

## 7. 已知遗留（不属本次范围）

- `deploy/tests/release-workflow-security-test.sh` 失败：断言要求
  `steps.validated_tag.outputs.tag`，而 `release.yml`（本次**未改动**）用
  `steps.plan.outputs.tag`。该失败在本同步之前的 fork `main` 上即存在。
- `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts` 的
  `stubs` 中仍列有 `ProxyAdBanner: true`。该组件已由 fork 自身提交 `5040acef6`
  移除（上游仍保留），模板已不再使用它，因此该 stub 是惰性条目，不影响任何测试。

## 8. 复现命令

```bash
# tag 祖先关系
for t in v0.2.9 v0.2.10 v0.2.11 v0.2.12 v0.2.13 v0.2.14 v0.2.15; do
  git merge-base --is-ancestor refs/upstream-sync/$t HEAD && echo "OK $t" || echo "FAIL $t"
done

# 门禁
cd backend && go test -tags=unit ./... -count=1
cd .. && make test-frontend
cd backend && go build -tags embed -o /tmp/sub2api ./cmd/server
```
