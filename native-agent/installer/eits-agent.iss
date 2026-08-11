#define MyAppName "EITS Monitoring Agent"
#ifndef MyAppVersion
  #define MyAppVersion "0.5.0-alpha.1"
#endif
#define MyAppPublisher "EITS Monitor"
#define MyAppExeName "Eits.Agent.Manager.exe"

[Setup]
AppId={{79B62D46-22FC-4C80-8BE6-5D50BCB1FC42}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\EITS Agent
DefaultGroupName=EITS Monitoring Agent
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\build\installer
OutputBaseFilename=EITS-Agent-Setup-v{#MyAppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
SetupIconFile=..\..\windows-agent\Eits.Agent.Manager\Assets\eits-agent-icon.ico
UninstallDisplayIcon={app}\Eits.Agent.Manager.exe
CloseApplications=force
RestartApplications=no

[Files]
Source: "..\build\windows\Eits.Agent.Service.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\Eits.Agent.Control.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\Eits.Agent.Manager.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\eits-agent-engine.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\eits-agent-updater.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\eits-agent-updater.exe.manifest"; DestDir: "{app}"; Flags: ignoreversion

[InstallDelete]
Type: files; Name: "{app}\eits-agent-manager.exe"
Type: files; Name: "{app}\eits-agent-manager.exe.manifest"
Type: files; Name: "{app}\eits-agent-manager-async.exe"
Type: files; Name: "{app}\eits-agent-manager-async.exe.manifest"
Type: files; Name: "{app}\eits-agent-manager-debug.exe"
Type: files; Name: "{app}\eits-agent-manager-fixed.exe"
Type: files; Name: "{app}\eits-agent-manager-fixed.exe.manifest"
Type: files; Name: "{app}\eits-agent-manager-modern.exe"
Type: files; Name: "{app}\eits-agent-manager-modern.exe.manifest"
Type: files; Name: "{app}\eits-manager-test2.exe"
Type: files; Name: "{app}\eits-manager-test3.exe"

[Icons]
Name: "{group}\EITS Agent Manager"; Filename: "{app}\Eits.Agent.Manager.exe"
Name: "{autostartup}\EITS Agent Manager"; Filename: "{app}\Eits.Agent.Manager.exe"; Tasks: traystartup

[Tasks]
Name: "traystartup"; Description: "Start EITS Agent Manager when I sign in"; Flags: checkedonce

[Run]
Filename: "{app}\Eits.Agent.Manager.exe"; Description: "Open EITS Agent Manager"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\sc.exe"; Parameters: "stop EITSAgent"; Flags: runhidden waituntilterminated; RunOnceId: "StopService"
Filename: "{sys}\sc.exe"; Parameters: "delete EITSAgent"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveService"

[Code]
var
  ServerPage: TInputQueryWizardPage;
  TokenPage: TInputQueryWizardPage;
  DevicePage: TInputQueryWizardPage;
  InsecurePage: TInputOptionWizardPage;

procedure InitializeWizard;
begin
  ServerPage := CreateInputQueryPage(wpSelectDir,
    'EITS Server', 'Connect this computer to your EITS Monitor server.',
    'Enter the server address used by the monitoring agent.');
  ServerPage.Add('Server URL:', False);
  ServerPage.Values[0] := 'https://monitor.example.com';

  TokenPage := CreateInputQueryPage(ServerPage.ID,
    'Agent registration', 'Enter the enrollment token.',
    'The token is used once to register this computer and is removed after successful enrollment.');
  TokenPage.Add('Enrollment token:', True);

  DevicePage := CreateInputQueryPage(TokenPage.ID,
    'Device name', 'Choose how this device appears in EITS Monitor.',
    'The Windows computer name is used by default.');
  DevicePage.Add('Device name:', False);
  DevicePage.Values[0] := GetComputerNameString;

  InsecurePage := CreateInputOptionPage(DevicePage.ID,
    'Connection security', 'Allow plain HTTP only on a trusted network.',
    'HTTPS is strongly recommended.', True, False);
  InsecurePage.Add('Allow insecure HTTP connection');
end;

function ShouldSkipPage(PageID: Integer): Boolean;
var
  ExistingConfig, ExistingIdentity: String;
begin
  ExistingConfig := ExpandConstant('{commonappdata}\EITS\Agent\config.json');
  ExistingIdentity := ExpandConstant('{commonappdata}\EITS\Agent\identity.json');
  Result := FileExists(ExistingConfig) and FileExists(ExistingIdentity) and
    ((PageID = ServerPage.ID) or (PageID = TokenPage.ID) or
     (PageID = DevicePage.ID) or (PageID = InsecurePage.ID));
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if CurPageID = ServerPage.ID then begin
    if Trim(ServerPage.Values[0]) = '' then begin MsgBox('Server URL is required.', mbError, MB_OK); Result := False; end;
  end else if CurPageID = TokenPage.ID then begin
    if Trim(TokenPage.Values[0]) = '' then begin MsgBox('Enrollment token is required.', mbError, MB_OK); Result := False; end;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  DataDir, ConfigPath, JSON, AllowHTTP, EscapedDataDir, EscapedLogDir,
  ResultPath, ResultMessage: String;
  ResultMessageAnsi: AnsiString;
  ResultCode: Integer;
begin
  if CurStep = ssPostInstall then begin
    DataDir := ExpandConstant('{commonappdata}\EITS\Agent');
    ForceDirectories(DataDir);
    ForceDirectories(DataDir + '\logs');
    SaveStringToFile(DataDir + '\logs\eits-agent.log',
      GetDateTimeString('yyyy-mm-dd hh:nn:ss', '-', ':') + ' installer: files copied and data directory prepared' + #13#10, True);

    EscapedDataDir := DataDir;
    StringChangeEx(EscapedDataDir, '\', '\\', True);

    EscapedLogDir := DataDir + '\logs';
    StringChangeEx(EscapedLogDir, '\', '\\', True);

    ConfigPath := DataDir + '\config.json';
    if (not FileExists(ConfigPath)) or (not FileExists(DataDir + '\identity.json')) then begin
      if InsecurePage.SelectedValueIndex = 0 then AllowHTTP := 'true' else AllowHTTP := 'false';
      JSON := '{' + #13#10 +
      '  "server_url": "' + ServerPage.Values[0] + '",' + #13#10 +
      '  "enrollment_token": "' + TokenPage.Values[0] + '",' + #13#10 +
      '  "device_name": "' + DevicePage.Values[0] + '",' + #13#10 +
      '  "collection_interval_seconds": 30,' + #13#10 +
      '  "request_timeout_seconds": 15,' + #13#10 +
      '  "allow_insecure_http": ' + AllowHTTP + ',' + #13#10 +
      '  "skip_tls_verify": false,' + #13#10 +
      '  "state_directory": "' + EscapedDataDir + '",' + #13#10 +
      '  "log_directory": "' + EscapedLogDir + '",' + #13#10 +
      '  "queue": {"enabled": true, "maximum_records": 2880},' + #13#10 +
      '  "logging": {"level": "info", "maximum_size_mb": 10, "maximum_files": 5}' + #13#10 +
      '}';
      SaveStringToFile(ConfigPath, JSON, False);
    end;

    ResultPath := DataDir + '\install.result';
    DeleteFile(ResultPath);
    if (not Exec(ExpandConstant('{app}\Eits.Agent.Control.exe'),
      'install "' + ExpandConstant('{app}\Eits.Agent.Service.exe') + '" "' + ResultPath + '"',
      '', SW_HIDE, ewWaitUntilTerminated, ResultCode)) or (ResultCode <> 0) then begin
      ResultMessage := '';
      if FileExists(ResultPath) then begin
        ResultMessageAnsi := '';
        LoadStringFromFile(ResultPath, ResultMessageAnsi);
        ResultMessage := String(ResultMessageAnsi);
      end;
      if Trim(ResultMessage) = '' then ResultMessage := 'The Windows service control helper returned exit code ' + IntToStr(ResultCode) + '.';
      RaiseException('EITS Monitoring Agent service installation failed: ' + ResultMessage);
    end;
    DeleteFile(ResultPath);
  end;
end;

function ReadControlResult(ResultPath: String): String;
var
  ResultMessageAnsi: AnsiString;
begin
  Result := '';
  if FileExists(ResultPath) then begin
    ResultMessageAnsi := '';
    LoadStringFromFile(ResultPath, ResultMessageAnsi);
    Result := String(ResultMessageAnsi);
  end;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  HelperPath, ResultPath, Detail: String;
  ResultCode: Integer;
begin
  Result := '';
  NeedsRestart := False;
  HelperPath := ExpandConstant('{app}\Eits.Agent.Control.exe');
  ResultPath := ExpandConstant('{commonappdata}\EITS\Agent\install-stop.result');
  ForceDirectories(ExpandConstant('{commonappdata}\EITS\Agent'));
  DeleteFile(ResultPath);
  WizardForm.StatusLabel.Caption := 'Stopping the existing EITS Agent service...';

  if FileExists(HelperPath) then begin
    if (not Exec(HelperPath, 'stop "' + ResultPath + '"', '', SW_HIDE,
      ewWaitUntilTerminated, ResultCode)) or (ResultCode <> 0) then begin
      Detail := ReadControlResult(ResultPath);
      if Trim(Detail) = '' then
        Detail := 'The service control helper returned exit code ' + IntToStr(ResultCode) + '.';
      Result := 'Setup could not stop the existing EITSAgent Windows service. ' + Detail;
    end;
  end else begin
    { Compatibility path for agents installed before the native control helper. }
    Exec(ExpandConstant('{sys}\sc.exe'), 'stop EITSAgent', '', SW_HIDE,
      ewWaitUntilTerminated, ResultCode);
    Sleep(5000);
  end;

  DeleteFile(ResultPath);
end;
