$ErrorActionPreference = 'Stop'
$serviceName = 'EITSAgent'
$serviceExecutable = 'C:\Program Files\EITS Agent\Eits.Agent.Service.exe'

$service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($service -and $service.Status -ne 'Stopped') {
    Stop-Service -Name $serviceName -Force
    $service.WaitForStatus('Stopped', [TimeSpan]::FromSeconds(30))
}

if (-not $service) {
    New-Service -Name $serviceName -BinaryPathName ('"' + $serviceExecutable + '"') -DisplayName 'EITS Monitoring Agent' -StartupType Automatic
} else {
    Set-ItemProperty -LiteralPath "HKLM:\SYSTEM\CurrentControlSet\Services\$serviceName" -Name ImagePath -Value ('"' + $serviceExecutable + '"')
    Set-Service -Name $serviceName -StartupType Automatic
}

Start-Service -Name $serviceName
(Get-Service -Name $serviceName).WaitForStatus('Running', [TimeSpan]::FromSeconds(30))
