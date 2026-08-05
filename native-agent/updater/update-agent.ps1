#Requires -RunAsAdministrator
param(
  [ValidateSet('stable','prerelease')][string]$Channel='prerelease',
  [ValidateSet('automatic','notify','disabled')][string]$Mode='automatic',
  [string]$Repository='ShodenEvo/eits-monitor'
)
$ErrorActionPreference='Stop'
if ($Mode -eq 'disabled') { exit 0 }
$InstallDir='C:\Program Files\EITS Agent'; $DataDir='C:\ProgramData\EITS\Agent'
$Exe=Join-Path $InstallDir 'eits-agent.exe'; $Task='EITS Monitoring Agent'; $Log=Join-Path $DataDir 'update.log'
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
function Write-Log($m){Add-Content -Path $Log -Value "$(Get-Date -Format o) $m"}
function Get-Releases {
  $h=@{'Accept'='application/vnd.github+json';'X-GitHub-Api-Version'='2022-11-28'}
  if($env:GITHUB_TOKEN){$h.Authorization="Bearer $env:GITHUB_TOKEN"}
  Invoke-RestMethod -Headers $h -Uri "https://api.github.com/repos/$Repository/releases?per_page=20"
}
try {
  if(!(Test-Path $Exe)){throw "Agent binary not found: $Exe"}
  $current=& $Exe version
  $releases=Get-Releases | Where-Object {-not $_.draft}
  if($Channel -eq 'stable'){$release=$releases|Where-Object{-not $_.prerelease}|Select-Object -First 1}else{$release=$releases|Select-Object -First 1}
  if(!$release){throw 'No eligible GitHub release found.'}
  $latest=$release.tag_name.TrimStart('v')
  if($latest -eq $current){Write-Log "Already current: $current"; exit 0}
  Write-Log "Update available $current -> $latest"
  if($Mode -eq 'notify'){exit 10}
  $asset=$release.assets|Where-Object{$_.name -match '^eits-agent-windows-v.*\.zip$'}|Select-Object -First 1
  if(!$asset){throw 'Windows agent ZIP asset not found.'}
  $tmp=Join-Path $env:TEMP ("eits-update-"+[guid]::NewGuid()); New-Item -ItemType Directory $tmp|Out-Null
  $zip=Join-Path $tmp $asset.name
  Invoke-WebRequest -UseBasicParsing -Uri $asset.browser_download_url -OutFile $zip
  if($asset.digest -and $asset.digest -match '^sha256:(.+)$'){
    $actual=(Get-FileHash $zip -Algorithm SHA256).Hash.ToLower(); if($actual -ne $Matches[1].ToLower()){throw 'Package SHA-256 mismatch.'}
  }
  Expand-Archive $zip -DestinationPath $tmp -Force
  $new=Get-ChildItem $tmp -Recurse -Filter 'eits-agent-windows-amd64.exe'|Select-Object -First 1
  if(!$new){throw 'Agent binary missing from package.'}
  $reported=& $new.FullName version; if($reported -ne $latest){throw "Package version $reported does not match release $latest"}
  $backup="$Exe.bak"
  Stop-ScheduledTask -TaskName $Task -ErrorAction SilentlyContinue; Start-Sleep 2
  Copy-Item $Exe $backup -Force; Copy-Item $new.FullName $Exe -Force
  Start-ScheduledTask -TaskName $Task; Start-Sleep 8
  $info=Get-ScheduledTaskInfo -TaskName $Task
  if($info.LastTaskResult -ne 0 -and $info.LastTaskResult -ne 267009){throw "Agent task failed after update: $($info.LastTaskResult)"}
  Remove-Item $backup -Force -ErrorAction SilentlyContinue
  Write-Log "Updated successfully to $latest"
} catch {
  Write-Log "ERROR: $($_.Exception.Message)"
  if(Test-Path "$Exe.bak") { Stop-ScheduledTask -TaskName $Task -ErrorAction SilentlyContinue; Copy-Item "$Exe.bak" $Exe -Force; Start-ScheduledTask -TaskName $Task -ErrorAction SilentlyContinue }
  throw
} finally { if($tmp -and (Test-Path $tmp)){Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue} }
