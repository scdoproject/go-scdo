@echo off
goto comment
    Build the command lines and tests in Windows.
    Must install gcc tool before building.
:comment

set para=%*
if not defined para (
    set para=all
)

for %%i in (%para%) do (
    call :%%i
)
pause
goto:eof

:all
call :copylib
call :discovery
call :node
call :client
call :light
call :tool
call :vm
goto:eof

:copylib
for /f "delims=" %%i in ('go env GOPAH') do set gopath=%%i
for /f "delims=" %%i in ('go env GOOS') do set goos=%%i
for /f "delims=" %%i in ('go env GOARCH') do set goarch=%%i
md %gopath%\pkg\%goos%_%goarch%\github.com
md %gopath%\pkg\%goos%_%goarch%\github.com\scdoproject
md %gopath%\pkg\%goos%_%goarch%\github.com\scdoproject\go-scdo
md %gopath%\pkg\%goos%_%goarch%\github.com\scdoproject\go-scdo\consensus
echo on
echo copyTo=%gopath%\pkg\%goos%_%goarch%\github.com\scdoproject\go-scdo\consensus\scdorand.a
copy .\consensus\scdorand\scdorand_windows_amd64.a  %gopath%\pkg\%goos%_%goarch%\github.com\scdoproject\go-scdo\consensus\scdorand.a
@echo off
goto:eof

:discovery
echo on
go build -o ./build/discovery.exe ./cmd/discovery
@echo "Done discovery building"
@echo off
goto:eof

:node
echo on
go build -o ./build/node.exe ./cmd/node
@echo "Done node building"
@echo off
goto:eof

:client
echo on
go build -o ./build/client.exe ./cmd/client
@echo "Done full node client building"
@echo off
goto:eof

:light
echo on
go build -o ./build/light.exe ./cmd/client/light
@echo "Done light node client building"
@echo off
goto:eof

:tool
echo on
go build -o ./build/tool.exe ./cmd/tool
@echo "Done tool building"
@echo off
goto:eof

:vm
echo on
go build -o ./build/vm.exe ./cmd/vm
@echo "Done vm building"
@echo off
goto:eof

:clean
del build\* /q /f /s
@echo "Done clean the build dir"
@echo off
goto:eof