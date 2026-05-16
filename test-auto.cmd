@echo off
setlocal

set RUN_INTEGRATION=1
for %%A in (%*) do (
	if /I "%%~A"=="--skip-integration" set RUN_INTEGRATION=0
)

echo [1/4] Running unit tests...
go test ./...
if errorlevel 1 goto fail

echo [2/4] Running vet...
go vet ./...
if errorlevel 1 goto fail

if "%RUN_INTEGRATION%"=="0" (
	echo [3/4] Skipping integration tests because --skip-integration was provided.
	goto success
)

if exist .local.json (
	echo [3/4] Running integration tests...
	go test -tags=integration -timeout 20m ./...
	if errorlevel 1 goto fail
) else (
	echo [3/4] Skipping integration tests because .local.json was not found.
)

:success
echo [4/4] All checks passed.
exit /b 0

:fail
echo Failed.
exit /b 1