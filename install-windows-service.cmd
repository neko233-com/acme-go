@echo off
setlocal

set SERVICE_NAME=%~1
set CONFIG_PATH=%~2
set BINARY_PATH=%~3

if "%SERVICE_NAME%"=="" set SERVICE_NAME=acme233-auto-renew
if "%CONFIG_PATH%"=="" set CONFIG_PATH=%CD%\config_acme.json
if "%BINARY_PATH%"=="" set BINARY_PATH=%CD%\acme233.exe

where nssm >nul 2>nul
if errorlevel 1 (
	echo nssm is required. Install it first and make sure nssm.exe is in PATH.
	exit /b 1
)

if not exist "%CONFIG_PATH%" (
	echo config file not found: %CONFIG_PATH%
	exit /b 1
)
if not exist "%BINARY_PATH%" (
	echo binary not found: %BINARY_PATH%
	exit /b 1
)

if not exist "%CD%\logs" mkdir "%CD%\logs"

nssm install "%SERVICE_NAME%" "%BINARY_PATH%" auto-renew -config "%CONFIG_PATH%"
if errorlevel 1 exit /b 1

nssm set "%SERVICE_NAME%" AppDirectory "%CD%"
nssm set "%SERVICE_NAME%" DisplayName "%SERVICE_NAME%"
nssm set "%SERVICE_NAME%" Description "acme233 automatic renewal loop"
nssm set "%SERVICE_NAME%" Start SERVICE_AUTO_START
nssm set "%SERVICE_NAME%" AppStdout "%CD%\logs\%SERVICE_NAME%.log"
nssm set "%SERVICE_NAME%" AppStderr "%CD%\logs\%SERVICE_NAME%.err.log"
nssm start "%SERVICE_NAME%"

echo Installed and started %SERVICE_NAME%
echo Check status with: nssm status %SERVICE_NAME%
endlocal