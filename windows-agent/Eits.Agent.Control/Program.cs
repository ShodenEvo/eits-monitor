using Eits.Agent.Shared;
using System.ServiceProcess;
using System.Text.Json;
using System.Text.Json.Nodes;

return await ControlProgram.RunAsync(args);

static class ControlProgram
{
    public static async Task<int> RunAsync(string[] args)
    {
        string? resultPath = args.Length > 0 && args[0].Equals("reconfigure", StringComparison.OrdinalIgnoreCase)
            ? args.ElementAtOrDefault(2)
            : args.ElementAtOrDefault(1);
        try
        {
            if (args.Length == 0) throw new ArgumentException("No operation was specified.");
            switch (args[0].ToLowerInvariant())
            {
                case "start": SetService(true); break;
                case "stop": SetService(false); break;
                case "restart": SetService(false); SetService(true); break;
                case "reconfigure":
                    if (args.Length < 3) throw new ArgumentException("Connection request is incomplete.");
                    await ReconfigureAsync(args[1]);
                    await File.WriteAllTextAsync(args[2], "OK");
                    break;
                default: throw new ArgumentException("Unknown operation.");
            }
            return 0;
        }
        catch (Exception ex)
        {
            if (resultPath is not null) try { await File.WriteAllTextAsync(resultPath, ex.Message); } catch { }
            return 1;
        }
    }

    static void SetService(bool start)
    {
        using var service = new ServiceController("EITSAgent");
        service.Refresh();
        if (start)
        {
            if (service.Status is ServiceControllerStatus.Running) return;
            if (service.Status is not ServiceControllerStatus.StartPending) service.Start();
            service.WaitForStatus(ServiceControllerStatus.Running, TimeSpan.FromSeconds(25));
        }
        else
        {
            if (service.Status is ServiceControllerStatus.Stopped) return;
            if (service.Status is not ServiceControllerStatus.StopPending) service.Stop();
            service.WaitForStatus(ServiceControllerStatus.Stopped, TimeSpan.FromSeconds(25));
        }
        service.Refresh();
        var expected = start ? ServiceControllerStatus.Running : ServiceControllerStatus.Stopped;
        if (service.Status != expected) throw new InvalidOperationException($"Windows service is {service.Status}; expected {expected}.");
    }

    static async Task ReconfigureAsync(string requestPath)
    {
        var request = JsonSerializer.Deserialize<ConnectionRequest>(await File.ReadAllTextAsync(requestPath), AgentConfig.JsonOptions)
                      ?? throw new InvalidDataException("Connection request is empty.");
        if (!Uri.TryCreate(request.ServerUrl, UriKind.Absolute, out var server) ||
            (server.Scheme != "http" && server.Scheme != "https")) throw new InvalidDataException("The server URL is invalid.");
        if (server.Scheme == "http" && !request.AllowInsecureHttp) throw new InvalidDataException("Insecure HTTP is not enabled.");

        using (var handler = new HttpClientHandler())
        using (var client = new HttpClient(handler) { Timeout = TimeSpan.FromSeconds(10) })
        {
            using var response = await client.GetAsync(request.ServerUrl.TrimEnd('/') + "/api/health");
            if (!response.IsSuccessStatusCode) throw new InvalidOperationException($"Server health check returned HTTP {(int)response.StatusCode}.");
        }

        Directory.CreateDirectory(AgentPaths.DataDirectory);
        var backup = AgentPaths.ConfigFile + ".previous";
        var oldIdentity = AgentPaths.IdentityFile + $".previous.{DateTimeOffset.UtcNow.ToUnixTimeSeconds()}";
        if (File.Exists(AgentPaths.ConfigFile)) File.Copy(AgentPaths.ConfigFile, backup, true);
        SetService(false);
        try
        {
            JsonObject config;
            try { config = JsonNode.Parse(await File.ReadAllTextAsync(AgentPaths.ConfigFile))?.AsObject() ?? new(); }
            catch { config = new(); }
            config["server_url"] = request.ServerUrl.TrimEnd('/');
            config["enrollment_token"] = request.EnrollmentToken;
            config["device_name"] = request.DeviceName;
            config["allow_insecure_http"] = request.AllowInsecureHttp;
            config["state_directory"] = AgentPaths.DataDirectory;
            config["log_directory"] = Path.Combine(AgentPaths.DataDirectory, "logs");
            if (File.Exists(AgentPaths.IdentityFile)) File.Move(AgentPaths.IdentityFile, oldIdentity);
            await File.WriteAllTextAsync(AgentPaths.ConfigFile, config.ToJsonString(AgentConfig.JsonOptions));
            SetService(true);

            var deadline = DateTime.UtcNow.AddSeconds(35);
            while (DateTime.UtcNow < deadline)
            {
                if (File.Exists(AgentPaths.IdentityFile)) return;
                await Task.Delay(500);
            }
            var detail = AgentStatusReader.ReadLastLine(AgentPaths.LogFile);
            throw new InvalidOperationException(string.IsNullOrWhiteSpace(detail) ? "Enrollment did not complete. Verify the enrollment token." : detail);
        }
        catch
        {
            try { SetService(false); } catch { }
            if (File.Exists(backup)) File.Copy(backup, AgentPaths.ConfigFile, true);
            if (!File.Exists(AgentPaths.IdentityFile) && File.Exists(oldIdentity)) File.Move(oldIdentity, AgentPaths.IdentityFile);
            try { SetService(true); } catch { }
            throw;
        }
    }
}
