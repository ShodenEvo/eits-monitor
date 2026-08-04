$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path "$Root\dist" | Out-Null
Push-Location $Root
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'
go build -trimpath -ldflags '-s -w -H=windowsgui' -o dist/eits-agent-windows-amd64.exe ./cmd/eits-agent
$env:GOOS='linux'; $env:GOARCH='amd64'
go build -trimpath -ldflags '-s -w' -o dist/eits-agent-linux-amd64 ./cmd/eits-agent
$env:GOARCH='arm64'
go build -trimpath -ldflags '-s -w' -o dist/eits-agent-linux-arm64 ./cmd/eits-agent
Pop-Location
Get-FileHash "$Root\dist\*" -Algorithm SHA256 | Format-Table
