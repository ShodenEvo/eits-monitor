using System.Diagnostics;
using System.ServiceProcess;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace Eits.Agent.Shared;

public static class AgentPaths
{
    public static string DataDirectory => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.CommonApplicationData), "EITS", "Agent");
    public static string InstallDirectory => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ProgramFiles), "EITS Agent");
    public static string ConfigFile => Path.Combine(DataDirectory, "config.json");
    public static string IdentityFile => Path.Combine(DataDirectory, "identity.json");
    public static string LogFile => Path.Combine(DataDirectory, "logs", "eits-agent.log");
}

public sealed class AgentConfig
{
    [JsonPropertyName("server_url")] public string ServerUrl { get; set; } = "";
    [JsonPropertyName("enrollment_token")] public string? EnrollmentToken { get; set; }
    [JsonPropertyName("device_name")] public string DeviceName { get; set; } = Environment.MachineName;
    [JsonPropertyName("collection_interval_seconds")] public int CollectionIntervalSeconds { get; set; } = 30;
    [JsonPropertyName("request_timeout_seconds")] public int RequestTimeoutSeconds { get; set; } = 15;
    [JsonPropertyName("allow_insecure_http")] public bool AllowInsecureHttp { get; set; }
    [JsonPropertyName("skip_tls_verify")] public bool SkipTlsVerify { get; set; }
    [JsonPropertyName("state_directory")] public string StateDirectory { get; set; } = AgentPaths.DataDirectory;
    [JsonPropertyName("log_directory")] public string LogDirectory { get; set; } = Path.Combine(AgentPaths.DataDirectory, "logs");
    [JsonPropertyName("queue")] public JsonElement Queue { get; set; }
    [JsonPropertyName("logging")] public JsonElement Logging { get; set; }

    public static AgentConfig Load() => JsonSerializer.Deserialize<AgentConfig>(File.ReadAllText(AgentPaths.ConfigFile), JsonOptions) ?? throw new InvalidDataException("Configuration is empty.");
    public static readonly JsonSerializerOptions JsonOptions = new() { WriteIndented = true, PropertyNameCaseInsensitive = true };
}

public sealed record AgentStatus(string ServiceState, string ServerUrl, string DeviceName, bool Registered, string AgentId, string LastActivity);

public static class AgentStatusReader
{
    public static AgentStatus Read()
    {
        var state = "Not installed";
        try { using var service = new ServiceController("EITSAgent"); state = service.Status.ToString(); } catch { }
        AgentConfig? config = null;
        try { config = AgentConfig.Load(); } catch { }
        var agentId = "";
        try { using var doc = JsonDocument.Parse(File.ReadAllText(AgentPaths.IdentityFile)); agentId = doc.RootElement.GetProperty("agent_id").GetString() ?? ""; } catch { }
        return new(state, config?.ServerUrl ?? "Not configured", config?.DeviceName ?? Environment.MachineName, agentId.Length > 0, agentId, ReadLastLine(AgentPaths.LogFile));
    }

    public static string ReadLastLine(string path)
    {
        try
        {
            using var stream = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);
            var length = (int)Math.Min(4096, stream.Length); stream.Seek(-length, SeekOrigin.End);
            var buffer = new byte[length]; stream.ReadExactly(buffer);
            return System.Text.Encoding.UTF8.GetString(buffer).Trim().Split('\n').LastOrDefault()?.Trim() ?? "";
        }
        catch { return ""; }
    }
}

public static class ElevatedControl
{
    public static async Task RunAsync(params string[] args)
    {
        var helper = Path.Combine(AgentPaths.InstallDirectory, "Eits.Agent.Control.exe");
        var start = new ProcessStartInfo(helper) { UseShellExecute = true, Verb = "runas", WindowStyle = ProcessWindowStyle.Hidden };
        foreach (var arg in args) start.ArgumentList.Add(arg);
        using var process = Process.Start(start) ?? throw new InvalidOperationException("Unable to start the elevated control helper.");
        await process.WaitForExitAsync();
        if (process.ExitCode != 0)
        {
            if (args.Length > 2 && File.Exists(args[2]))
            {
                var detail = await File.ReadAllTextAsync(args[2]);
                if (!string.IsNullOrWhiteSpace(detail)) throw new InvalidOperationException(detail);
            }
            throw new InvalidOperationException("The requested operation failed. Review the latest agent log.");
        }
    }

    public static async Task ReconfigureAsync(ConnectionRequest request)
    {
        Directory.CreateDirectory(AgentPaths.DataDirectory);
        var id = Guid.NewGuid().ToString("N");
        var requestPath = Path.Combine(Path.GetTempPath(), $"eits-connection-{id}.json");
        var resultPath = requestPath + ".result";
        await File.WriteAllTextAsync(requestPath, JsonSerializer.Serialize(request, AgentConfig.JsonOptions));
        try
        {
            await RunAsync("reconfigure", requestPath, resultPath);
            if (File.Exists(resultPath))
            {
                var message = await File.ReadAllTextAsync(resultPath);
                if (!string.Equals(message.Trim(), "OK", StringComparison.OrdinalIgnoreCase)) throw new InvalidOperationException(message);
            }
        }
        finally
        {
            try { File.Delete(requestPath); } catch { }
            try { File.Delete(resultPath); } catch { }
        }
    }
}

public sealed record ConnectionRequest(string ServerUrl, string EnrollmentToken, string DeviceName, bool AllowInsecureHttp);
