param([string]$Version = '0.5.0-alpha.11')

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
New-Item -ItemType Directory -Force -Path "$Root\dist", "$Root\build\windows" | Out-Null
Push-Location $Root
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'
$WindowsLdflags = "-s -w -H=windowsgui -X main.version=$Version"
$PortableLdflags = "-s -w -X main.version=$Version"
go build -trimpath -ldflags $WindowsLdflags -o build/windows/eits-agent-engine.exe ./cmd/eits-agent
$Manifest = Join-Path $Root 'assets\windows-common-controls.manifest'
go build -trimpath -ldflags $WindowsLdflags -o build/windows/eits-agent-updater.exe ./cmd/eits-agent-updater
Copy-Item $Manifest "$Root\build\windows\eits-agent-updater.exe.manifest" -Force
$WindowsSolution = Join-Path (Split-Path -Parent $Root) 'windows-agent'
$PublishOptions = @('-c', 'Release', '-r', 'win-x64', '--self-contained', 'true', '-p:PublishSingleFile=true', '-p:IncludeNativeLibrariesForSelfExtract=true', '-o', "$Root\build\windows")
dotnet publish "$WindowsSolution\Eits.Agent.Manager\Eits.Agent.Manager.csproj" @PublishOptions
dotnet publish "$WindowsSolution\Eits.Agent.Control\Eits.Agent.Control.csproj" @PublishOptions
dotnet publish "$WindowsSolution\Eits.Agent.Service\Eits.Agent.Service.csproj" @PublishOptions
$env:GOOS='linux'; $env:GOARCH='amd64'
go build -trimpath -ldflags $PortableLdflags -o dist/eits-agent-linux-amd64 ./cmd/eits-agent
$env:GOARCH='arm64'
go build -trimpath -ldflags $PortableLdflags -o dist/eits-agent-linux-arm64 ./cmd/eits-agent
Pop-Location
Get-FileHash "$Root\dist\*", "$Root\build\windows\*.exe" -Algorithm SHA256 | Format-Table
