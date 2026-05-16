@echo off
setlocal

set COMMIT_MESSAGE=auto up
if not "%~1"=="" set COMMIT_MESSAGE=%~1

call test-auto.cmd
if errorlevel 1 goto fail

echo [1/3] Staging git changes...
git add .
git diff --cached --quiet
if not errorlevel 1 (
	echo No staged changes. Nothing to push.
	goto end
)

echo [2/3] Commit and push...
git commit -m "%COMMIT_MESSAGE%"
if errorlevel 1 goto fail

git push
if errorlevel 1 goto fail

echo [3/3] Done.
goto end

:fail
echo Failed. Push aborted.
exit /b 1

:end
endlocal