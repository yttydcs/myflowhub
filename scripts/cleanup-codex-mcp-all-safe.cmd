@echo off
setlocal
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_PATH=%SCRIPT_DIR%cleanup-codex-mcp.ps1"

where pwsh >nul 2>nul
if %errorlevel%==0 (
  set "PS_EXE=pwsh"
) else (
  set "PS_EXE=powershell"
)

echo Kill old chrome + ace + context7 MCP runtimes...
"%PS_EXE%" -NoLogo -ExecutionPolicy Bypass -File "%SCRIPT_PATH%" -Mode Kill -Kinds chrome,ace,context7 -MinAgeMinutes 90
echo.
pause
