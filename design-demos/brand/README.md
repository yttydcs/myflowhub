# MyFlowHub Brand Assets

`design-demos/brand/` 同时保存 MyFlowHub 品牌图标的 canonical 资产和历次设计研究。生产接入只应从本文件列出的 canonical 路径取用；`direction-*`、`brand-icon-study-v3*` 和 `brand-icon-study-v4*` 是研究历史，不应直接进入产品。

## Canonical identity

- 核心概念：`Coupled Seam / 耦合缝`
- 决策版本：V5 Calibrated Seam
- Canonical SVG：[myflowhub-icon.svg](myflowhub-icon.svg)
- Canonical 512px PNG：[myflowhub-icon-512.png](myflowhub-icon-512.png)
- 生产规范：[brand-icon-study-v5-spec.md](brand-icon-study-v5-spec.md)
- 精修与验收板：[brand-icon-study-v5.html](brand-icon-study-v5.html)

`myflowhub-icon.svg` 与 `myflowhub-symbol-v5.svg` 的几何和色彩必须保持一致。前者是稳定入口；后者保留明确的设计版本，便于追溯。

## Production variants

| 场景 | 资产 | 建议尺寸 |
| --- | --- | --- |
| 品牌主标 / 浅色背景 | [myflowhub-symbol-v5.svg](myflowhub-symbol-v5.svg) | 32px 以上 |
| 品牌主标 / 单色 | [myflowhub-symbol-v5-mono.svg](myflowhub-symbol-v5-mono.svg) | 32px 以上 |
| 品牌主标 / 深色反白 | [myflowhub-symbol-v5-inverse.svg](myflowhub-symbol-v5-inverse.svg) | 32px 以上 |
| Compact / 彩色 | [myflowhub-symbol-v5-compact.svg](myflowhub-symbol-v5-compact.svg) | 20–31px |
| Compact / 单色 | [myflowhub-symbol-v5-compact-mono.svg](myflowhub-symbol-v5-compact-mono.svg) | 20–31px |
| 系统托盘 / 单色 | [myflowhub-tray-v5.svg](myflowhub-tray-v5.svg) | 16px |
| 系统托盘 / 反白 | [myflowhub-tray-v5-inverse.svg](myflowhub-tray-v5-inverse.svg) | 16px |
| 应用图标方形源层 | [myflowhub-app-icon-source-v5.svg](myflowhub-app-icon-source-v5.svg) | 1024px 源文件 |

平台圆角、遮罩与阴影由目标平台生成，不要写回 `myflowhub-app-icon-source-v5.svg`。Desktop、Android、Web 或安装包接入前，应从上述源资产生成各平台派生文件并单独验证；不要直接修改 canonical master 来适配某个平台。

## Research history

- V1 / V2：树、资源、订阅和多元素徽记探索；已否决。
- V3：[brand-icon-study-v3.html](brand-icon-study-v3.html)；节点键、字母和资源登记方向。
- V4：[brand-icon-study-v4.html](brand-icon-study-v4.html)；建立 Keyed Cell 母形和 Coupled Seam 主标。
- V5：[brand-icon-study-v5.html](brand-icon-study-v5.html)；冻结方向，完成接缝、compact、tray 和 app source 的光学校正。

不要删除历史研究板：它们记录了 Pac-Man、拼图、普通 C、链环和过密连接线等淘汰原因，是后续品牌演进的决策证据。
