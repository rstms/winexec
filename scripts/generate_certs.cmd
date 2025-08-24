@ECHO OFF
set program=winexec
set dir=%APPDATA%\%PROGRAM%
set config=%dir%\config.yaml
mkdir 2>NUL %dir%
echo updating certificates in %dir%
set mkcert=%USERPROFILE%\go\bin\mkcert.exe
%mkcert% --force --chain %dir%\ca.pem
%mkcert% --force 127.0.0.1 --cert-file %dir%\cert.pem --key-file %dir%\key.pem --duration 10y -- --san localhost
