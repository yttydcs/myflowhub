# A 本机界面验证

从仓库根启动 `node apps/desktop/frontend/node_modules/vite/bin/vite.js apps/desktop/frontend --host 127.0.0.1 --port 5299 --strictPort`，访问 `/test/phase-a/preview.html`。

此页面挂载真实 App、总览和检查器，使用明确标记的模拟 API/2000 个六类型值。只用于界面布局、键盘/鼠标和状态交互，不证明 TCP、Wails IPC、授权或真实音频。真实 TCP/SDK/binding 容量及故障由 Go integration 测试独立验证。

验证路径：选择 Node 3 → 资源总览 → 订阅筛选结果 → 等待 2000/2000 实时 → 滚到底部 → 筛选1999 → 检查器数字写40并回读 → 清空筛选 → 切换深色主题/1000px窗口 → 停止全部。滚动/筛选过程中后台句柄保持2000，DOM行数应小于60；不能由父grid的auto min-height将滚动容器撑到88000px。

浏览器控制台可读取 `window.phaseAPreview.handles.size` 核对测试API句柄。最终必须停止预览服务，尤其不要在运行 Vite 的同时通过 `npm ci` 重装其原生 CSS 依赖。
