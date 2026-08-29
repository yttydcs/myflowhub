# Brand Identity

## Background

MyFlowHub 是统一权威节点树、可扩展 Resource、Subscription、Command 与可插拔链路共同组成的平台。产品需要一个在 Desktop、托盘、文档、网站和未来移动端中保持一致的品牌入口，但品牌图标不能成为微型架构图，也不能与 Resource type icon 混用。

## Goal

建立一个稳定、可追溯、可多尺寸生产的 MyFlowHub 品牌标志体系。核心识别应来自两个独立节点形成可信连接的单一关系，而不是依赖文字、渐变、阴影或额外说明。

## Scope

- canonical 品牌主标及其版本化源资产；
- 彩色、单色、反白、compact、tray 与 app icon source；
- 品牌色、clear space、小尺寸和平台派生边界；
- 设计研究、淘汰理由和决策记录的长期保留；
- 后续产品接入时的资产选择和验收口径。

## Non-goals

- 不在品牌 symbol 内逐项表达 Resource、Subscription、Command、Transport 或权限 UI；
- 不将品牌 symbol 用作 Variable、Stream、Topic、File 等资源类型图标；
- 不在 canonical app source 中预制 Windows、macOS、Android 或 Web 的平台遮罩与阴影；
- 本需求不决定正式字标字体、商标注册、市场传播文案或各平台打包流程。

## Functional Requirements

- 稳定入口 `design-demos/brand/myflowhub-icon.svg` 必须指向当前已接受的 Coupled Seam 几何。
- 必须保留 32px 以上完整主标、20–31px compact 和 16px tray 的独立资产。
- 必须提供彩色、单色和深色反白版本；单色版本不依赖颜色判断两个主体。
- 必须保留未遮罩的 1024px app source，供平台流水线派生。
- canonical SVG 更新时必须同步生成和检查 512px PNG，并保持版本化源资产可追溯。
- 研究板与历史方向必须保留，至少记录被否决方案的主要误读与淘汰原因。

## Non-functional Requirements

- 标志在 16px 下不得粘连，在 24px 下不得依赖双色才能成立。
- 颜色只使用受治理的 Mineral / Graphite / Warm White 体系；禁止新增渐变、发光和装饰状态色。
- SVG 必须自包含，不依赖外部字体、脚本、滤镜或远程资源。
- 资产命名必须按场景区分，避免产品接入时把大尺寸图形机械缩成 tray glyph。
- 平台派生不得反向覆盖 canonical master。

## Acceptance Criteria

- `myflowhub-icon.svg` 与版本化 V5 主标的路径几何和颜色一致。
- 512px PNG 可透明渲染且视觉中心正确。
- 16、20、24、32、48 和 88px 原尺寸工作台中两个主体保持分离。
- 浅色、单色、反白以及 Desktop/Mobile/Circle 遮罩预览均通过视觉检查。
- 品牌研究板在 1440、768 和 390px 视口无横向溢出，所有本地 SVG 加载成功。

## Related Intake

- [2026-08-29 MyFlowHub 品牌图标定稿](../intake/2026-08-29_brand-icon-coupled-seam.md)

## Related Specs

- [Brand Identity Assets](../specs/brand-identity-assets.md)

## Related Decisions

- [Coupled Seam Brand Identity](../decisions/2026-08-29_coupled-seam-brand-identity.md)

## Related Changes

- [2026-08-29 品牌图标 Coupled Seam 定稿](../change/2026-08-29_brand-icon-coupled-seam.md)
