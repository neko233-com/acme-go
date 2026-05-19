@echo off
setlocal enabledelayedexpansion

set VERSION=%~1

where git >nul 2>nul
if errorlevel 1 (
	echo git is required.
	exit /b 1
)
where go >nul 2>nul
if errorlevel 1 (
	echo go is required.
	exit /b 1
)

if "%VERSION%"=="" (
	for /f "usebackq delims=" %%v in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "$tags = git tag --list 'v[0-9]*.[0-9]*.[0-9]*'; if (-not $tags) { 'v0.0.1'; exit }; $latest = $tags | ForEach-Object { if ($_ -match '^v(\d+)\.(\d+)\.(\d+)$') { [pscustomobject]@{ Tag = $_; Major = [int]$matches[1]; Minor = [int]$matches[2]; Patch = [int]$matches[3] } } } | Sort-Object Major, Minor, Patch | Select-Object -Last 1; 'v{0}.{1}.{2}' -f $latest.Major, $latest.Minor, ($latest.Patch + 1)"`) do set VERSION=%%v
)

if /i "%VERSION:~0,1%" NEQ "v" set VERSION=v%VERSION%

echo Publishing library version %VERSION%

git diff --quiet
if errorlevel 1 (
	echo Working tree has unstaged changes. Commit or stash them before publishing.
	exit /b 1
)
git diff --cached --quiet
if errorlevel 1 (
	echo Working tree has staged changes. Commit or unstage them before publishing.
	exit /b 1
)

git rev-parse "%VERSION%" >nul 2>nul
if not errorlevel 1 (
	echo Tag %VERSION% already exists.
	exit /b 1
)

call test-auto.cmd --skip-integration
if errorlevel 1 exit /b 1

go test ./pkg/acmego
if errorlevel 1 exit /b 1

git tag -a "%VERSION%" -m "publish library %VERSION%"
if errorlevel 1 exit /b 1

git push origin "%VERSION%"
if errorlevel 1 exit /b 1

echo Published %VERSION% to GitHub. Consumers can use: go get github.com/neko233-com/acme-go/pkg/acmego@%VERSION%
endlocal
