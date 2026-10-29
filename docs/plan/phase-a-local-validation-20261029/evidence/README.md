# 本机补验的持久证据入口

- [机器汇总](oct29-r2-summary.json)：无回执、撤权、保护值、网络补测、独立替换及排除项。
- [候选与交付包指纹](oct29-r2-final-package.json)：生产二进制与原生测试一致；归档只移动/补充文档。
- [包校验](oct29-r2-package-verification.json)、[兼容矩阵](oct29-r2-compatibility.json)。
- [前端回归](oct29-r2-frontend-tests.log)、[Go写入](oct29-r2-go-write-tests.log)、[Go vet](oct29-r2-go-vet.log)、[错误基线拒绝](oct29-r2-baseline-mismatch.log)。
- [无回执终态](oct29-r2-noack-5-result.jpg)、[Metrics回滚连接恢复](oct29-r2-switch-metrics-baseline.jpg)、[清理确认](oct29-r2-cleanup.json)。

完整历史日志、原生截图和夹具脚本位于本机 `D:/project/MyFlowHub3/archives/phase-a-local-validation-20261029/workflow-evidence-private.zip`。私有身份、配置和日志保存在同目录 `native-fixtures-private.zip`，不得提交或发布；主线已有修改的私有恢复副本为 `main-preexisting-private.zip`。ZIP完整性已校验，校验和见该目录preservation.json。预览包仍保存在项目artifacts中；原始JSON内的旧工作树路径仅说明测试来源，不是当前入口。
