# 2026-08-29 Coupled Seam 品牌身份

## Status

Accepted

## Context

MyFlowHub 的架构包含权威节点树、Resource、Subscription、Command、权限和可插拔链路。早期图标尝试直接呈现树、连接线、圆点、六边形或电路，导致元素过多、缩放失败，且容易像普通网络、芯片或图标库素材。品牌 symbol 需要在不复述整套架构的前提下保留项目独有的“节点与可信连接”特征。

## Options Considered

1. **Authority Tree / 多节点树**：语义直接，但微缩后只剩普通网络拓扑；同时错误地把逻辑、权限与物理链路全部固化为一棵视觉树。
2. **Flow Loom / 电路与连接线**：能表达可扩展链路，但线条在 16–24px 不稳定，容易像芯片、分享或 workflow 图标。
3. **Node Key / Resource Register / Monogram**：简洁，但分别落入普通钥匙、文件夹、字母或图标库联想。
4. **Keyed Cell + Coupled Seam**：以同一母形的两个独立节点和一条负空间接缝表达可信连接；可产生主标、compact、tray 与延展纹样。

## Decision

采用 V5 `Calibrated Seam` 作为 MyFlowHub canonical 品牌 symbol。

- 两个同源 Keyed Cell 分别使用 Mineral Blue 和 Mineral Light。
- 中央负空间 Coupled Seam 同时承担主体分离和可信关系表达。
- 主标不直接绘制树、Resource、Subscription、Command、Transport、权限或状态。
- 使用完整、compact 和 tray 三套光学校正，而不是机械缩放单一文件。
- `design-demos/brand/myflowhub-icon.svg` 是稳定入口；V5 文件保留版本追溯。
- O / Recursive Link 只作为可选延展图形，不进入 canonical symbol。

## Consequences

### Positive

- 标志概念可以用一句话复述，并与节点关系直接相关。
- 彩色、单色、反白和 16px 版本共享同一几何语言。
- 不再需要继续向 symbol 添加六边形、圆点、箭头或电路线解释架构。
- 品牌标志与 Resource type icon、连接状态色和平台遮罩边界清楚。

### Trade-offs

- 远距离仍可能产生轻微链环联想，后续传播需要依靠稳定的轮廓、色彩和字标组合建立专属性。
- 正式公开发布前仍需商标近似检索和法律清查。
- 当前字标只是临时系统字体，尚未成为本决策的一部分。
- 各平台派生不在本次定稿内，需要单独接入和验证。

## Confidence

High。用户已在 V4 选择该方向，并在 V5 光学校正后明确要求固定和归档；1440/768/390 研究板、三种背景模式与 16–88px 实物阶梯均已验证。

## Supersedes / Superseded By

- Supersedes：无正式品牌 ADR；V1–V4 是研究历史而非 accepted decision。
- Superseded by：none。

## Related Requirements

- [Brand Identity](../requirements/brand-identity.md)

## Related Specs

- [Brand Identity Assets](../specs/brand-identity-assets.md)

## Related Changes

- [2026-08-29 品牌图标 Coupled Seam 定稿](../change/2026-08-29_brand-icon-coupled-seam.md)
