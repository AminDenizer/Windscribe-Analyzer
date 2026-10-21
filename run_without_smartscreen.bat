@echo off
title Windscribe Ultra Tester Launcher
echo [Windscribe Ultra Tester] Bypassing Windows SmartScreen Mark-of-the-Web...
powershell -NoProfile -Command "Unblock-File -Path '%~dp0windscribe-ultra-tester.exe' -ErrorAction SilentlyContinue"
start "" "%~dp0windscribe-ultra-tester.exe"
