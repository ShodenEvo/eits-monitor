using Eits.Agent.Shared;
using System.Diagnostics;
using System.ServiceProcess;

if (Environment.UserInteractive && args.Contains("--console", StringComparer.OrdinalIgnoreCase))
{
    using var host = new AgentService();
    host.StartInteractive();
    Console.CancelKeyPress += (_, e) => { e.Cancel = true; host.StopInteractive(); };
    await Task.Delay(Timeout.Infinite);
}
else ServiceBase.Run(new AgentService());

sealed class AgentService : ServiceBase
{
    readonly object gate = new();
    CancellationTokenSource? cancellation;
    Process? engine;

    public AgentService() { ServiceName = "EITSAgent"; CanStop = true; CanShutdown = true; AutoLog = true; }
    protected override void OnStart(string[] args) { WriteHostLog("service host starting"); cancellation = new(); _ = Task.Run(() => SuperviseAsync(cancellation.Token)); }
    protected override void OnStop() { WriteHostLog("service host stopping"); cancellation?.Cancel(); StopEngine(); }
    protected override void OnShutdown() => OnStop();
    public void StartInteractive() => OnStart([]);
    public void StopInteractive() => OnStop();

    async Task SuperviseAsync(CancellationToken token)
    {
        while (!token.IsCancellationRequested)
        {
            try
            {
                var executable = Path.Combine(AppContext.BaseDirectory, "eits-agent-engine.exe");
                if (!File.Exists(executable)) throw new FileNotFoundException("Agent engine executable was not found.", executable);
                var start = new ProcessStartInfo(executable) { UseShellExecute = false, CreateNoWindow = true };
                start.ArgumentList.Add("run"); start.ArgumentList.Add("-config"); start.ArgumentList.Add(AgentPaths.ConfigFile);
                var process = Process.Start(start) ?? throw new InvalidOperationException("Unable to start the agent engine.");
                lock (gate) engine = process;
                await process.WaitForExitAsync(token);
            }
            catch (OperationCanceledException) { break; }
            catch (Exception ex) { WriteHostLog(ex.Message); }
            finally { lock (gate) { engine?.Dispose(); engine = null; } }
            try { await Task.Delay(TimeSpan.FromSeconds(30), token); } catch (OperationCanceledException) { break; }
        }
    }

    void StopEngine()
    {
        lock (gate)
        {
            try { if (engine is { HasExited: false }) engine.Kill(true); } catch { }
        }
    }

    static void WriteHostLog(string message)
    {
        try { Directory.CreateDirectory(Path.GetDirectoryName(AgentPaths.LogFile)!); File.AppendAllText(AgentPaths.LogFile, $"{DateTime.UtcNow:u} service host: {message}{Environment.NewLine}"); } catch { }
    }
}
