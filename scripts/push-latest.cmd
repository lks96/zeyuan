@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0push-latest.ps1" %*
pause
