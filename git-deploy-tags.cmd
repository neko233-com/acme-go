@echo off
setlocal

call "%~dp0publish-lib.cmd" %*
exit /b %errorlevel%