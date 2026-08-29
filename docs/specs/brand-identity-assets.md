# Brand Identity Assets

## Scope

本文定义 MyFlowHub Coupled Seam 品牌图标的 canonical 入口、几何、色彩、多尺寸资产路由和平台派生边界。完整视觉研究和示例位于 `design-demos/brand/`；本文是产品接入应遵循的稳定技术契约。

## Canonical Sources

- 稳定 SVG：`design-demos/brand/myflowhub-icon.svg`
- 稳定 512px PNG：`design-demos/brand/myflowhub-icon-512.png`
- 版本化 master：`design-demos/brand/myflowhub-symbol-v5.svg`
- 生产规范：`design-demos/brand/brand-icon-study-v5-spec.md`
- 资产清单：`design-demos/brand/README.md`

`myflowhub-icon.svg` 与 `myflowhub-symbol-v5.svg` 必须保持相同路径、transform、接缝和色值。稳定入口不携带版本号，版本化 master 用于追溯和回滚。

## Geometry Contract

- 画布：`256 × 256`，SVG intrinsic size 为 `512 × 512`。
- Keyed Cell 源网格：`96 × 96`。
- 第一单元中心：`(96, 96)`；旋转 `45°`；缩放 `1.2`。
- 第二单元中心：`(160, 160)`；旋转 `225°`；缩放 `1.2`。
- 完整主标接缝：`9`；compact 接缝：`24`；tray 接缝：`30`。
- 建议 clear space：主标总高的 `25%`。
- 不允许独立改变两单元的角度、比例、位置或圆角，也不允许将负空间接缝填成第三个主体。

## Color Contract

| Token | Value | Use |
| --- | --- | --- |
| Mineral Blue | `#3F6F8F` | 第一单元、品牌主色 |
| Mineral Light | `#86B2C9` | 第二单元、关系层次 |
| Graphite | `#202222` | 单色主标与浅色托盘 |
| Warm White | `#F5F5F2` | 反白主标与 app source 背景 |

绿色只表示运行时连接状态，不属于品牌 symbol。

## Asset Selection

| Target | Source | Range |
| --- | --- | --- |
| Brand color | `myflowhub-symbol-v5.svg` | 32px+ |
| Brand mono | `myflowhub-symbol-v5-mono.svg` | 32px+ |
| Brand inverse | `myflowhub-symbol-v5-inverse.svg` | 32px+ |
| Compact color | `myflowhub-symbol-v5-compact.svg` | 20–31px |
| Compact mono | `myflowhub-symbol-v5-compact-mono.svg` | 20–31px |
| Tray light surface | `myflowhub-tray-v5.svg` | 16px |
| Tray dark surface | `myflowhub-tray-v5-inverse.svg` | 16px |
| Platform app source | `myflowhub-app-icon-source-v5.svg` | 1024px source |

## Platform Derivation

- Windows/macOS/Linux installer、Web favicon/PWA、Android adaptive icon 等派生资产必须从上表对应源文件生成。
- app source 提供完整方形背景；圆角、圆形 mask、safe zone 和阴影由目标平台规范控制。
- 平台派生需要单独测试裁切、透明度、深浅背景、任务栏/托盘和安装包显示，不得只验证源 SVG。
- 生成后的 `.ico`、多密度 PNG 或平台资源目录属于各应用模块，必须在接入 workflow 中记录来源版本 `V5`。

## Error Prevention

- 16px 使用完整主标会使接缝闭合；必须切换 tray asset。
- 将反白 SVG 放在透明浅色背景会不可见；调用方必须按 surface 选择版本。
- 不得把 `direction-*` 或 V3/V4 研究文件直接接入产品。
- canonical master 发生变更时，必须同步更新本 spec、decision/change 记录和资产 README。

## Validation

- SVG 可由 XML parser 读取；不允许外部引用。
- canonical PNG 必须为 512×512，并保留透明背景。
- V5 研究板必须验证所有本地图片加载、三种预览交互和 1440/768/390 视口。
- 接入某个平台时，还必须运行该平台自己的构建与图标显示检查。

## Related Requirements

- [Brand Identity](../requirements/brand-identity.md)

## Related Decisions

- [Coupled Seam Brand Identity](../decisions/2026-08-29_coupled-seam-brand-identity.md)

## Related Changes

- [2026-08-29 品牌图标 Coupled Seam 定稿](../change/2026-08-29_brand-icon-coupled-seam.md)
