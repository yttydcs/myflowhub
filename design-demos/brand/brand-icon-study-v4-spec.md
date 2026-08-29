# MyFlowHub 品牌图标研究 V4 · Research-backed brief

> 状态：选型研究，不替换 `myflowhub-icon.svg`，不发送给 Desktop 实现任务。V1–V3 全部保留。

## 1. 研究结论

本轮吸收 Apple App Icons、Microsoft Windows App Icon、Google Play Icon、Figma Brand Book、Slack/Pentagram、Mozilla、Docker、Atlassian 与 IBM Carbon 的公开规范。共同结论不是某一种流行造型，而是：品牌标志必须先有一个独有、可复述的 idea；主标尽量只承担一个隐喻；用少量母形构成可扩展视觉语言；品牌 symbol、平台 app icon 与 16px interface/tray glyph 应分别绘制；深色、单色和小尺寸不是导出阶段的附属品，而是几何设计阶段的输入。

参考：

- Apple: https://developer.apple.com/design/human-interface-guidelines/app-icons
- Microsoft: https://learn.microsoft.com/windows/apps/design/iconography/app-icon-design
- Google Play: https://developer.android.com/distribute/google-play/resources/icon-design-specifications
- Figma Brand Book: https://www.figma.com/using-the-figma-brand/
- Slack identity: https://www.pentagram.com/work/slack
- Mozilla: https://mozilla.design/
- Docker: https://www.docker.com/company/newsroom/media-resources/
- Atlassian: https://atlassian.design/foundations/iconography
- IBM Carbon: https://carbondesignsystem.com/elements/icons/usage/

## 2. 项目事实与现有资产

MyFlowHub 的当前权威架构是一棵 authenticated parent-child Node tree。Node 是 identity、authority、routing 与 Resource ownership 的共同主体；Resource type 可扩展；Subscription 是一级关系；Transport 可以替换但不改变上层语义。Desktop 已建立矿物蓝、石墨灰、浅色默认的工具界面。

现有资产：

- 现用临时图标：`myflowhub-icon.svg`，仅作历史基线，不视为不可修改的官方资产。
- UI 原型：`../myflowhub-desktop-codex-inspired.html`。
- UI 色值：Mineral `#3F6F8F`、Mineral Light `#86B2C9`、Graphite `#202222`、Warm White `#F5F5F4`、Night `#191A1A`。
- 已有研究：`icon-directions.html`、`icon-emblem-directions.html`、`icon-emblem-directions-v2.html`、`brand-icon-study-v3.html`。

资产完整度：**部分 / 正在建立**。当前任务正是定义品牌 symbol，因此不存在需要照抄的既有正式 Logo。项目 UI 与架构文档是本轮的真实性来源。

## 3. 唯一视觉母题

### Keyed Cell · 定向节点单元

母形是一块带有唯一内凹键口的稳定单元。完整实体表示 Node 的可信边界；唯一键口表示一个 Node 在 authority 域内只有一个当前有效父方向。它不是拼图、聊天气泡或播放按钮，因此键口不使用圆形接点、箭头尖头或凸出的 puzzle tab。

母形可以通过镜像、配对和递归形成更丰富的品牌构图。Resource、Subscription 与 tree 不再分别画成圆点和连接线；它们只在母形的组合关系中被暗示。换句话说，主标只有一个隐喻：**能够被定向连接的可信节点单元**。

## 4. 三种构图实验

1. **M · Keyed Cell / 节点单元**：只使用一个母形，用来解释几何语法；独立使用仍有马蹄形或字母 C 的误读风险，不作为最终主标。
2. **N · Coupled Seam / 耦合缝**：两个同源母形相对衔接，中央形成不可替换的负空间缝。表达经过认证的关系，作为本轮主品牌标志。
3. **O · Recursive Link / 递归链**：同一母形按比例递归排列。表达 Node tree 可以继续生长，只作为延展图形，不进入主标选型。

三者不是三个互不相关的隐喻，而是同一个母形的三种强度。本轮采用 N 为主标、N 的光学校正版为 16px glyph、O 为品牌图案；M 只保留为几何母形。

## 5. 几何与生产规范

- 母形先在 `96 × 96` 网格绘制，再映射到 48px 产品图标基准。
- 外轮廓圆角在 48px 下约 5px；键口保持直线与锐利内部交点，建立“外柔内准”的工具性格。
- 主标视觉安全区不小于画布的 9%；平台 app icon 使用完整正方形源层，让系统负责最终圆角和阴影。
- 24px 是完整品牌 symbol 的建议最小尺寸；16px 使用单独的 tray glyph。
- 16px glyph 的最小负空间不低于 2px；辅助色与内部细节全部移除。
- 几何在彩色、纯黑、纯白版本中完全一致；不依赖渐变、透明叠色和阴影维持识别。
- 主色只有 Mineral Blue；Mineral Light 仅用于多单元之间的层次。绿色继续只表示连接状态。
- 品牌 symbol 与字标之间的间距固定为 symbol 高度的 28%；最小 clear space 暂定为 symbol 高度的 25%，等待选型后按光学校准收口。

## 6. 应用图标与 UI glyph 分离

- **Brand symbol**：透明背景，可用于文档、官网、Splash 与品牌组合。
- **App icon source**：1024/512 方形 Warm White 背景层 + 居中 symbol；不预制平台圆角或外部投影。
- **Tray / sidebar glyph**：16/20px 单色 Coupled Seam，中央缝隙扩宽为常规版约四倍，保证缩小时仍保持两个独立节点。
- **Resource type icons**：不直接复制品牌 symbol。它们继续使用 Desktop 的统一描边系统，只继承“外部圆角、内部锐角、方形端点”的形式语言。

## 7. Form 推导五问

- 叙事角色：品牌选型板，先让用户识别母形，再理解组合与生产规则。
- 观众距离：主要为 1m 桌面浏览，同时必须通过 16px 实物检查。
- 视觉温度：冷静、可信、精确，但通过外圆角避免机械和古板。
- 容量估算：首屏只展示一个主母形和一句 idea；随后依次展示三组合、尺寸、平台和评审，不堆品牌故事。
- 视觉母题：来自“唯一父方向的可信 Node”，以唯一内凹键口而不是树状连线表达。

## 8. 淘汰门槛

- 遮住文字后，若第一联想是 Pac-Man、聊天气泡、播放键、拼图、链条或普通 C 字母，则母形需要重画。
- 16px 键口闭合或主体视觉偏左超过一个像素，则 compact 不通过。
- N 若只能依靠双色区分两个单元，则不具备单色生产能力。
- O 若变成信号波、俄罗斯套娃或单向箭头，只保留为延展图案，不进入主标候选。
- 任一版本若需要再加入六边形、圆点或连接线才“讲得通”，说明单一隐喻失败，应停止加元素。
