# docs/10-代码开发指南 完善设计（spec）

> **范围**:仅补 `docs/10-代码开发指南.md`(Developer Handbook),不动 `dev.md` 主体。
> **背景**:2360 行初版完成于 v1.1,未反映 v2.1 上线特性、未覆盖 i18n/a11y/observability/质量门禁,若干章节可加深。
> **目标行数**:2360 → **3200-3400 行**(+37-44%),**只扩展**现有节或新增相邻子节,不动章节边界。
> **配套**:`dev.md §48.Y` 已有 v2.1 实施回溯;本文档面向**未来加入项目的工程师**。
> **基线 commit**:`5a0d44b`(§48.X 占位页升级 + typecheck 全清)。

---

## 1. 设计目标

| 目标 | 验收 |
| :-- | :-- |
| v2.1 特性准确反映 | §3.1 / §3.5 / §5.1 / §5.6 / §5.8 / §8.1 内嵌,v2.1 关键术语 100% 收录 |
| 主题空白填补 | §3.9 i18n / §3.10 a11y / §4.10 observability / §7.7 quality gate 各成节,各 ≥ 60 行,带端到端代码示例 |
| 现有章节质量提升 | §3.7 vitest 2 增量模式、§6.4 测试金字塔 v2.1 兼容、§A.1 术语 v2.1 扩充、§2.5 ADR-0007 |
| 不破坏既有读者路径 | 章节数保持 9(§1-§8 + §A),不引入 §B 与 v2.1 changelog 子树 |
| 索引/锚点可用 | MkDocs 锚点完整、`dev.md` 通过 §A.2 链回 |
| 文件不引入编码损坏 | `file` 命令保持 UTF-8;text 不出现 `�` / `?` |

---

## 2. 预算分配(总计 +940 行,2360 → ~3300)

| 类别 | 预算 | 落点 | 深度标准 |
| :-- | --: | :-- | :-- |
| **v2.1 同步** | 50% (+470 行) | §3.1 / §3.5 / §5.1 / §5.6 / §5.8(新)/ §8.1 / §A.2 | 每节带 `file:line`、1 张 ASCII 图、1 个迁移注意 |
| **主题空白填补** | 35% (+320 行) | §3.9 / §3.10 / §4.10 / §7.7(全新增) | 每节 ≥ 60 行,1 个端到端代码 + 测试要求 |
| **质量修正** | 15% (+149 行,净 +150 - 1 删重) | §3.7 / §6.4 / §A.1 / §2.5 / §7(删重) | 取证风格 |

---

## 3. v2.1 同步(50% 预算)

### B1 · §3.1 三域合一 → +60 行
- **escort 端**:
  - `message/chat` v2.1 timeline(会话页时间线 + 标记/收藏/删除)
  - `checkin` 3 步状态机(pending → arrived → in_service,带 distance 估算)
  - `profile` v2.1 升级(余额 + 评分 + 4 shortcut)
- **patient 端**:
  - 进行中订单 widget(index alias)
  - `profile` snapshot 卡(3 卡:钱包/券/订单)
- **admin 端**(占位页升级):
  - coupons / messages / reviews / sos 实时 dashboard
- 配 ASCII 三域 v2.1 矩阵图

### B2 · §3.5 API 层 → +80 行
- **message v2.1 接口**:
  - `GET /messages/conversations/:id/messages?cursor=`(分页)
  - `DELETE /messages/:id`(软删)
  - `POST /messages/:id/favorite`
- **checkin 三步端点契约**:
  - `POST /orders/:id/checkin/arrived`
  - `POST /orders/:id/checkin/in-service`
  - `POST /orders/:id/checkin/complete`
- 引用 `frontend/unified-app/src/api/message.ts`、`frontend/unified-app/src/api/escort.ts` 最新签名

### B3 · §5.1 订单状态机 → +80 行
- **checkin 3 步状态机**:
  ```
  pending → escort_confirmed → arrived → in_service → completed
                          └─checkin 3 步子状态机
  ```
- distance 估算(Haversine / 高德 API 调用)
- 防作弊 remark 字段(陪诊师填,后台审计)
- 状态转换合法性矩阵扩展

### B4 · §5.6 30s 锁定 → +40 行
- **v2.1 增强**:`CountdownBadge` 30s UI、超时 fallback、EscortAcceptanceState、rejection_count 阈值
- 跨节索引 → §3.1 / §8.1

### B5 · §5.8(新)v2.1 关键变更索引 → +60 行
- 一页索引,所有 v2.1 改动 → §3/§5/§8 锚点
- 读者扫一眼能跳转;不展开论述

### B6 · §8.1 任务指南 → +120 行
- 三个新 step-by-step:
  - **8.1.4 加一个 checkin step**(后端 endpoint + 前端 widget + e2e)
  - **8.1.5 加一个 message 子页**(list → chat ↔ favorite 操作)
  - **8.1.6 profile snapshot 加新卡**(数据源 + card 渲染 + testid)

### B7 · §A.2 附录 → +30 行
- v2.1 changelog 锚点表 + 链接到 `dev.md §48.Y`

---

## 4. 主题空白填补(35% 预算,全新增)

### C1 · §3.9 国际化(i18n) → +80 行
- **核心内容**:
  - vue-i18n + uni-app locale 机制
  - 翻译键命名约定(`<domain>.<entity>.<field>`,例如 `escort.profile.balance`)
  - 三端差异(小程序的 `uni.getStorageSync('locale')` vs h5 的 `navigator.language`)
  - 缺失 key 监控 + CI 报警
- **代码示例**:
  ```ts
  // src/i18n/index.ts
  export const i18n = createI18n({ legacy: false, locale: 'zh-CN', fallbackLocale: 'zh-CN', messages });
  // 监控缺失 key
  i18n.global.missing = (key) => { Sentry.captureMessage(`i18n missing: ${key}`); };
  ```
- **测试要求**:`vitest` 单测断言关键翻译键存在,覆盖所有 v2.1 新页面文案。

### C2 · §3.10 无障碍(a11y) → +80 行
- **核心内容**:
  - WCAG 2.1 AA 基本项:对比度、focus-visible、aria-label
  - UiButton / UiIconButton 必须 `aria-label`
  - role 命名(`button` / `navigation` / `main`)
  - 键盘导航(esc/enter/箭头)
- **代码示例**:
  ```vue
  <!-- components/shared/UiButton.vue -->
  <button :aria-label="ariaLabel ?? text" :aria-disabled="loading" />
  ```
- **测试要求**:`@axe-core/playwright` e2e + `a11y-snapshot` vitest 断言关键路径无 critical violation。

### C3 · §4.10 可观测性 → +80 行
- **核心内容**:
  - trace-id 透传:Hertz / Gin 中间件注入 `X-Trace-Id`
  - `logger.FromContext` 用法(从 `shared/logger` 引用)
  - Kafka event 加 trace header(`sarama` interceptor)
  - 生产环境 OTel 路径:OTLP endpoint、exporter、sampling
- **代码示例**:
  ```go
  // shared/logger/logger.go
  func FromContext(ctx context.Context) *zap.Logger {
      if tid, ok := ctx.Value(traceIDKey{}).(string); ok {
          return base.With(zap.String("trace_id", tid))
      }
      return base
  }
  ```
- **测试要求**:Go test 断言 logger 输出含 trace_id。

### C4 · §7.7 质量门禁 → +80 行
- **核心内容**:
  - Codecov.io 接入 + CODECOV_TOKEN GitHub Secret
  - `frontend-coverage` job(`.github/workflows/ci.yml`)配置
  - `make check-all` 串行执行 go test race + vitest + playwright e2e
  - typecheck 强制(从 47 → 0)
  - 覆盖率门槛(当前 729 it / 106 Go,后续提升策略)
- **代码示例**:
  ```yaml
  # .github/workflows/ci.yml 片段
  frontend-coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cd frontend/unified-app && pnpm install && pnpm vitest run --coverage
      - uses: codecov/codecov-action@v4
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          directory: frontend/unified-app/coverage
  ```

---

## 5. 质量修正(15% 预算)

| # | 改动 | 行 |
|---|------|--:|
| D1 | §7 重复标题修复(删 `## §7 开发工作流与调试` 第二处) | -1 |
| D2 | §3.7 测试策略补 vitest 2 增量模式 | +60 |
| D3 | §6.4 测试金字塔补 v2.1 引入的"快照 vs 实现细节"取舍 | +40 |
| D4 | §A.1 名词解释补 v2.1 术语(checkin / snapshot card / fav conversation) | +20 |
| D5 | §2.5 ADR 简要补 ADR-0007 v2.1 同步模式 | +30 |

**ADR-0007 内容**(新建):
- **背景**:v2.1 上线后前端需要获知后端消息状态(message read / favorite),选了 `sync.ts`(短轮询)而非 webhook
- **决策**:30s 轮询 + visibilitychange 暂停
- **后果**:实现简单、误以为 key 路径;高负载时不可用;暂未评估 SSE 升级
- **替代**:WebSocket / SSE(留待 v2.2)

---

## 6. 验证清单

| 项 | 验证方式 | 期望 |
|---|---------|------|
| 总行数 | `wc -l docs/10-代码开发指南.md` | 3200-3400 |
| `file:line` 引用准确 | `grep -nE` 比对 vs 实际文件 | 0 stale |
| 章节数 = 9 | `grep -c "^## " docs/10-代码开发指南.md` | 9 |
| 编码正确 | `file docs/10-代码开发指南.md` | UTF-8 |
| 无 `�` 残留 | `grep -P "[\x80-\xff]" docs/10-代码开发指南.md` | 仅正常 UTF-8 |
| MkDocs 锚点 | `mkdocs build --strict` | 无 anchor 警告 |
| 与 dev.md 不重复 | 主题扫描 | 仅 §A.2 引用链 |

---

## 7. 不做的事(out of scope)

- ❌ 不重写既有章节标题/编号
- ❌ 不新增独立 §B v2.1 changelog(v2.1 内容集成进相关章节)
- ❌ 不动 `dev.md` 主体(只通过 §A.2 链)
- ❌ 不动 `docs/01-09`(业务文档)
- ❌ 不引入新依赖(vue-i18n / @axe-core 已存在,直接用)
- ❌ 不重命名既有术语(只用 §A.1 解释新术语)

---

## 8. 落地步骤(概要)

1. 修 D1(§7 重复)→ 立即可见
2. 写 §5.8 v2.1 索引(承上启下)
3. 写 §3.9 / §3.10 / §4.10 / §7.7 四个新节
4. 扩 §3.1 / §3.5 / §5.1 / §5.6 / §8.1
5. 扩 §3.7 / §6.4 / §A.1 / §2.5
6. 补 §A.2 链接表
7. `wc -l` 验证、`file` 验证、`grep` 验证
8. 提交 commit

---

## 9. 配套说明

- 本 spec 与 `dev.md §48.Y` 互不重复:`dev.md` 记**本次会话做了什么**,本手册记**未来怎么改代码**
- 本 spec 写完后,移交 `writing-plans` 生成实施 plan
- 实施 plan 完成后,本 spec 应在 `dev.md` 追加一节简短引用(`← ` ↗ `docs/superpowers/specs/2026-10-15-dev-docs-improvements-design.md`)