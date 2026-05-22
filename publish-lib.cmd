@echo off
setlocal enabledelayedexpansion

set "VERSION="
set "DRY_RUN=0"
set "VERSION_FILE=version.txt"

for %%A in (%*) do (
	if /I "%%~A"=="--dry-run" (
		set "DRY_RUN=1"
	) else if not defined VERSION (
		set "VERSION=%%~A"
	) else (
		echo Unexpected argument: %%~A
		exit /b 1
	)
)

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

echo Fetching remote tags from origin...
git fetch origin --tags
if errorlevel 1 exit /b 1

if "%VERSION%"=="" (
	for /f "usebackq delims=" %%v in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "$versions = @(); $recorded = ''; if (Test-Path 'version.txt') { $recorded = (Get-Content 'version.txt' | Select-Object -First 1).Trim() }; foreach ($value in @(git tag --list 'v[0-9]*.[0-9]*.[0-9]*') + @($recorded)) { if ($value -match '^v(\d+)\.(\d+)\.(\d+)$') { $versions += [pscustomobject]@{ Major = [int]$matches[1]; Minor = [int]$matches[2]; Patch = [int]$matches[3] } } }; if (-not $versions) { 'v0.0.1'; exit }; $latest = $versions | Sort-Object Major, Minor, Patch | Select-Object -Last 1; 'v{0}.{1}.{2}' -f $latest.Major, $latest.Minor, ($latest.Patch + 1)"`) do set VERSION=%%v
)

if /i "%VERSION:~0,1%" NEQ "v" set VERSION=v%VERSION%

echo %VERSION%| findstr /r /c:"^v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$" >nul
if errorlevel 1 (
	echo Invalid version: %VERSION%
	exit /b 1
)

echo Publishing library version %VERSION%

for /f "usebackq delims=" %%b in (`git branch --show-current`) do set CURRENT_BRANCH=%%b
if "%CURRENT_BRANCH%"=="" (
	echo Cannot determine current branch. Checkout a branch and retry.
	exit /b 1
)

if "%DRY_RUN%"=="0" (
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

if "%DRY_RUN%"=="1" (
	echo Dry run complete. Would write %VERSION% to %VERSION_FILE%, commit it, push %CURRENT_BRANCH%, and push tag %VERSION%.
	endlocal
	exit /b 0
)

>"%VERSION_FILE%" echo %VERSION%
if errorlevel 1 exit /b 1

git add "%VERSION_FILE%"
if errorlevel 1 exit /b 1

git diff --cached --quiet
if errorlevel 1 (
	git commit -m "chore: release %VERSION%"
	if errorlevel 1 exit /b 1
) else (
	echo %VERSION_FILE% already records %VERSION%. Skipping release commit.
)

git tag -a "%VERSION%" -m "publish library %VERSION%"
if errorlevel 1 exit /b 1

echo Pushing branch %CURRENT_BRANCH% to origin...
git push origin "%CURRENT_BRANCH%"
if errorlevel 1 exit /b 1

echo Pushing tag %VERSION% to origin...
git push origin "%VERSION%"
if errorlevel 1 exit /b 1

echo Published %VERSION% to GitHub. Consumers can use: go get github.com/neko233-com/acme-go/pkg/acmego@%VERSION%
echo 已同步推送分支 %CURRENT_BRANCH% 和标签 %VERSION%。
endlocal
