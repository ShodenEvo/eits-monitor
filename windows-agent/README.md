# EITS Agent for Windows

The Windows desktop and service integration use .NET 8 while the cross-platform collection engine remains in Go.

- `Eits.Agent.Manager`: responsive WPF desktop and system-tray application.
- `Eits.Agent.Control`: elevated, console-free helper for service and connection changes.
- `Eits.Agent.Service`: Windows Service host that supervises `eits-agent-engine.exe`.
- `Eits.Agent.Shared`: configuration, status, paths, and helper communication.

Build the solution with:

```powershell
dotnet build .\windows-agent\Eits.Agent.Windows.sln -c Release
```

The release workflow publishes self-contained Windows executables, builds the Go engine, and produces the Inno Setup installer. Client computers do not need the .NET runtime installed separately.
