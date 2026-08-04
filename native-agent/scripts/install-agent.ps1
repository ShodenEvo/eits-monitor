#Requires -RunAsAdministrator
param(
  [Parameter(Mandatory=$true)][string]$ServerUrl,
  [Parameter(Mandatory=$true)][string]$EnrollmentToken,
  [string]$DeviceName=$env:COMPUTERNAME,
  [switch]$AllowInsecureHttp
)

$ErrorActionPreference = 'Stop'

$InstallDir = 'C:\Program Files\EITS Agent'
$DataDir = 'C:\ProgramData\EITS\Agent'
$LogDir = Join-Path $DataDir 'logs'
$ExeSource = Join-Path $PSScriptRoot 'eits-agent.exe'
$ExeTarget = Join-Path $InstallDir 'eits-agent.exe'
$ConfigPath = Join-Path $DataDir 'config.json'
$TaskName = 'EITS Monitoring Agent'

if (!(Test-Path -LiteralPath $ExeSource)) {
    throw "Binary not found beside installer: $ExeSource"
}

if ($ServerUrl -match '^http://' -and -not $AllowInsecureHttp) {
    throw 'HTTP requires -AllowInsecureHttp. Use HTTPS for production.'
}

New-Item -ItemType Directory -Force -Path $InstallDir, $DataDir, $LogDir | Out-Null

# Stop an existing task before replacing the binary.
$ExistingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($null -ne $ExistingTask) {
    Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
}

Copy-Item -LiteralPath $ExeSource -Destination $ExeTarget -Force

$configObject = @{
    server_url = $ServerUrl.TrimEnd('/')
    enrollment_token = $EnrollmentToken
    device_name = $DeviceName
    collection_interval_seconds = 30
    request_timeout_seconds = 15
    allow_insecure_http = [bool]$AllowInsecureHttp
    skip_tls_verify = $false
    state_directory = $DataDir
    log_directory = $LogDir
    queue = @{
        enabled = $true
        maximum_records = 2880
    }
    logging = @{
        level = 'info'
        maximum_size_mb = 10
        maximum_files = 5
    }
}

$configJson = $configObject | ConvertTo-Json -Depth 5
# Write UTF-8 without BOM so Go's JSON parser can read it on Windows PowerShell 5.1.
[System.IO.File]::WriteAllText($ConfigPath, $configJson, [System.Text.UTF8Encoding]::new($false))

$Action = New-ScheduledTaskAction -Execute $ExeTarget -Argument 'run'
$Trigger = New-ScheduledTaskTrigger -AtStartup
$Settings = New-ScheduledTaskSettingsSet `
    -RestartCount 999 `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries
$Principal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $Action `
    -Trigger $Trigger `
    -Settings $Settings `
    -Principal $Principal `
    -Force | Out-Null

Start-ScheduledTask -TaskName $TaskName
Start-Sleep -Seconds 3

$Task = Get-ScheduledTask -TaskName $TaskName
$TaskInfo = Get-ScheduledTaskInfo -TaskName $TaskName

Write-Host 'EITS Agent installed and started.' -ForegroundColor Green
Write-Host "Executable:    $ExeTarget"
Write-Host "Configuration: $ConfigPath"
Write-Host "Log:           $LogDir\eits-agent.log"
Write-Host "Task state:    $($Task.State)"
Write-Host "Last result:   $($TaskInfo.LastTaskResult)"
