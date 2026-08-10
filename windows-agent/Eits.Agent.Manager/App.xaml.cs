using System.Configuration;
using System.Data;
using System.Windows;

namespace Eits.Agent.Manager;

/// <summary>
/// Interaction logic for App.xaml
/// </summary>
public partial class App : System.Windows.Application
{
    Mutex? instanceMutex;

    protected override void OnStartup(StartupEventArgs e)
    {
        instanceMutex = new Mutex(true, "EITS.Agent.Manager.SingleInstance", out var firstInstance);
        if (!firstInstance) { Shutdown(); return; }
        base.OnStartup(e);
    }

    protected override void OnExit(ExitEventArgs e)
    {
        instanceMutex?.Dispose();
        base.OnExit(e);
    }
}

