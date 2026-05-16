@echo off
setlocal

echo [1/4] Running unit tests...
go test ./...
if errorlevel 1 goto fail

if exist .local.json (
	echo [2/4] Running integration tests...
	go test -tags=integration -timeout 20m ./...
	if errorlevel 1 goto fail
) else (
	echo [2/4] Skipping integration tests because .local.json was not found.
)

echo [3/4] Staging git changes...
git add .
git diff --cached --quiet
if not errorlevel 1 (
	echo No staged changes. Nothing to push.
	goto end
)

echo [4/4] Commit and push...
git commit -m "auto up"
if errorlevel 1 goto fail

git push
if errorlevel 1 goto fail

echo Done.
goto end

:fail
echo Failed. Push aborted.
exit /b 1

:end
endlocal