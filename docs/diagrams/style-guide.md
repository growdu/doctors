# Diagram Style Guide · 医疗蓝绿主题

> 本项目所有架构图采用统一的"医疗蓝绿"主题。  
> 文件命名 `docs/diagrams/style-guide.md`，约定所有图的语义角色映射。

## 1. Semantic Roles

| Role | Hex / RGBA | 用途 |
| :-- | :-- | :-- |
| `paper` | `#fafbfc` | 页面背景 |
| `paper-2` | `#ffffff` | 卡片 / 容器背景 |
| `ink` | `#1f3a4d` | 主文字、深色 ink（深墨蓝） |
| `muted` | `#6b8294` | 辅助文字、默认箭头 |
| `soft` | `#97a7b6` | 弱化元素 |
| `rule` | `#e5eaf0` | hairline 边框 |
| `rule-solid` | `#cdd7e0` | 实线边框 |
| `accent` | `#1a8f7a` | 医疗绿主 accent |
| `accent-tint` | `rgba(26,143,122,0.10)` | accent 浅底 |
| `accent-deep` | `#0f6b5a` | accent 深色 / hover |
| `link` | `#2e5aa8` | link 蓝 |
| `danger` | `#c0392b` | 错误 / 风控 |
| `warning` | `#d97706` | 警告 / 异常 |

## 2. Node Treatment

| 类型 | Fill | Stroke | 备注 |
| :-- | :-- | :-- | :-- |
| Focal（1~2 个） | `accent-tint` | `accent` | 编辑强调 |
| Backend / API / Step | `paper-2` | `ink` | 主体框 |
| Store / State | `ink @ 0.05` | `muted` | 数据存储 |
| External / Cloud | `ink @ 0.03` | `ink @ 0.30` | 外部系统 |
| Input / User | `muted @ 0.10` | `soft` | 用户输入 |
| Optional / Async | `ink @ 0.02` | `ink @ 0.20` dashed `4,3` | 可选/异步 |
| Security / Boundary | `accent @ 0.05` | `accent @ 0.50` dashed `4,4` | 安全边界 |

## 3. Typography

| 用途 | 字体 | 字号 | 字重 |
| :-- | :-- | :--: | :--: |
| Title | Instrument Serif | 28px | 400 |
| Node name | Geist | 12px | 600 |
| Sublabel | Geist Mono | 9px | 400 |
| Eyebrow | Geist Mono | 8px | 500 |
| Arrow label | Geist Mono | 8px | 400 |
| Editorial aside | Instrument Serif italic | 14px | 400 |
| 中文标签 | Geist（兼容中文回退 PingFang/Microsoft YaHei） | ≥12px | 400~600 |

## 4. 焦点（accent）使用规则

- 每张图最多 1~2 个 accent 节点
- 主流程链路用 accent tint 高亮
- 危险/异常路径用 `danger`，警告用 `warning`

## 5. 字体加载

```html
<link href="https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=Geist:wght@400;500;600&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
```