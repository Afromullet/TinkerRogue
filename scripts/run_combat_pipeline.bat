@echo off
setlocal

REM Combat simulation pipeline: simulate -> analyze -> compress
REM Run from the TinkerRogue directory: scripts\run_combat_pipeline.bat

set "PROJECT_ROOT=%~dp0.."
set "SIM_LOGS=%PROJECT_ROOT%\simulation_logs"
set "BALANCE_REPORT=%PROJECT_ROOT%\docs\combat_balance_report.csv"
set "COMPRESSED_REPORT=%PROJECT_ROOT%\docs\combat_balance_compressed.csv"

cd /d "%PROJECT_ROOT%"

echo === Cleaning old simulation logs ===
if exist "%SIM_LOGS%\*.json" del /Q "%SIM_LOGS%\*.json"

echo === Step 1/3: Running Combat Simulator ===
go run ./tools/combat_simulator %*
if %errorlevel% neq 0 (
    echo ERROR: Combat simulator failed
    exit /b %errorlevel%
)

echo.
echo === Step 2/3: Generating Balance Report ===
go run ./tools/combat_balance --dir "%SIM_LOGS%" --output "%BALANCE_REPORT%"
if %errorlevel% neq 0 (
    echo ERROR: Balance report generation failed
    exit /b %errorlevel%
)

echo.
echo === Step 3/3: Compressing Report ===
go run ./tools/report_compressor --input "%BALANCE_REPORT%" --output "%COMPRESSED_REPORT%"
if %errorlevel% neq 0 (
    echo ERROR: Report compression failed
    exit /b %errorlevel%
)

echo.
echo === Pipeline Complete ===
echo Simulation logs:   %SIM_LOGS%
echo Full report:       %BALANCE_REPORT%
echo Compressed report: %COMPRESSED_REPORT%
