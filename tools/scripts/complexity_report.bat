@echo off
REM complexity_report.bat - Windows wrapper for complexity_report.sh.
REM Forwards all arguments to the bash script. Uses git-bash explicitly
REM because plain "bash" on Windows often resolves to WSL (C:\Windows\System32\bash.exe),
REM which fails if no WSL distro is installed.

setlocal

REM Prefer git-bash in its usual install locations.
set "GITBASH="
if exist "C:\Program Files\Git\bin\bash.exe"       set "GITBASH=C:\Program Files\Git\bin\bash.exe"
if not defined GITBASH if exist "C:\Program Files\Git\usr\bin\bash.exe" set "GITBASH=C:\Program Files\Git\usr\bin\bash.exe"
if not defined GITBASH if exist "C:\Program Files (x86)\Git\bin\bash.exe" set "GITBASH=C:\Program Files (x86)\Git\bin\bash.exe"

if defined GITBASH (
    "%GITBASH%" "%~dp0complexity_report.sh" %*
    exit /b %errorlevel%
)

echo ERROR: git-bash not found in the usual install locations. 1>&2
echo Install Git for Windows (https://git-scm.com/download/win) 1>&2
echo or run the script directly from a git-bash shell: 1>&2
echo     bash tools/scripts/complexity_report.sh 1>&2
exit /b 1
