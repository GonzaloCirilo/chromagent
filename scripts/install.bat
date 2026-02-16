@echo off
setlocal

echo Building chromagent...

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
pushd "%PROJECT_DIR%"

go build -o chromagent.exe ./cmd/chromagent
if %errorlevel% neq 0 (
    echo Build failed.
    popd
    exit /b 1
)

set "BINARY_PATH=%CD%\chromagent.exe"
echo Built: %BINARY_PATH%

if not exist "%APPDATA%\claude" mkdir "%APPDATA%\claude"

echo.
echo To register hooks, merge the config into %%APPDATA%%\claude\settings.json
echo   (or use /hooks inside Claude Code):
echo.
echo   Replace /path/to/chromagent with:
echo   %BINARY_PATH%
echo.
echo   Example for a single event:
echo   "Stop": [{ "hooks": [{ "type": "command", "command": "%BINARY_PATH%" }] }]
echo.
echo   See example-settings.json for the full config.
echo.
echo Ensure Razer Synapse is running with Chroma SDK enabled.

popd
endlocal
