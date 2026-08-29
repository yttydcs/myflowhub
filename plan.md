# Plan - MyFlowHub canonical local status

## Current State

- Canonical monorepo: `repo/MyFlowHub`
- Desktop stack: Wails 2 + React 18 + TypeScript + Vite
- Desktop production interface: Codex-inspired flat resource workspace complete
- Current workflow: none
- Remote/publish state: no remote; no push, release or publish performed

## Completed Baseline

- Extensible Resource descriptor/capability runtime、SDK、bindings 与 first-party products
- Desktop persistent multi-Profile/CredentialStore、real Topology/Catalog、Renderer 与 View persistence
- 浅色默认与完整深色主题、per-Profile non-secret UI preferences
- Resource/View sidebar、arbitrary-depth flat Explorer、search/focus/breadcrumb/back 与 WAI-ARIA keyboard
- Content tabs、right Inspector、Workspace drag/add/save 与完整 Connection/Profile/Appearance Settings
- Desktop Vitest、performance、TypeScript/Vite、Go packages、Wails Windows production build 与 1440×900 UI validation

## Deferred / Separate Ownership

- TREE02 — server-side topology/catalog pagination 或 lazy loading
- SYNC01 — 打开的 content tab session 跨重启持久化
- ICON01 — 项目品牌图标设计、替换、生成与归档，由独立任务负责
- PUB01 — remote push、release、publish，需要单独授权

## Stable Documentation

- [Desktop feature](docs/features/desktop.md)
- [Desktop resource workspace requirement](docs/requirements/desktop-resource-workspace.md)
- [Desktop resource workspace v2 spec](docs/specs/desktop-resource-workspace-v2.md)
- [Desktop interface intake](docs/intake/2026-08-29_desktop-codex-interface-production.md)
- [Desktop interface execution plan archive](docs/plan/plan_archive_2026-08-29_desktop-codex-interface-production.md)
- [Desktop interface change archive](docs/change/2026-08-29_desktop-codex-interface-production.md)

## Gate

- Blocked: no
- Active workflow: none
- Local closeout: complete
