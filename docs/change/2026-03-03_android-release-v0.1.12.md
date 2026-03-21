# 2026-03-03 Android：发布 Release v0.1.12

## 背景 / 目标
- 背景：Android `main` 已包含近期变更（含 `hubmobile` 升级 `management v0.1.2`）。
- 目标：打 `v0.1.12` tag 触发 `release` workflow，产出签名的 Release APK 与 AAR，并发布 GitHub Release。

## 变更内容
- 发布：
  - 新增并推送 tag：`v0.1.12`
  - tag 指向 Android 提交：`6708fa2`
- CI 说明：
  - `.github/workflows/release.yml` 监听 `push tags: v*.*.*`，会：
    - `assembleRelease`（signed）
    - `gomobile bind` 生成 `myflowhub.aar`
    - 生成 `build-info.txt`
    - 发布 GitHub Release（`generate_release_notes=true`）

## Plan 任务映射
- REL0：确认 tag 未占用
- REL1：创建并推送 tag
- REL2：记录与归档

## 验证方式
- 本地（tag 指向确认）：
  - `git show --no-patch --oneline v0.1.12`
- 远端（tag 存在）：
  - `git ls-remote --tags origin v0.1.12`
- 发布结果：
  - 等待 GitHub Actions `release` workflow 完成，并检查 Release 附件包含：
    - `app-release.apk`
    - `myflowhub.aar`
    - `build-info.txt`

## 回滚方案
- 不建议删除已推送 tag。
- 若 release 构建失败或需要修复：在 `main` 修复后发布下一个 patch（例如 `v0.1.13`）。

