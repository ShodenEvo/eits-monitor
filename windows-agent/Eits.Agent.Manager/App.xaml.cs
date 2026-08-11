using System.Configuration;
using System.Data;
using System.Windows;

namespace Eits.Agent.Manager;

/// <summary>
/// Interaction logic for App.xaml
/// </summary>
public partial class App : System.Windows.Application
{
    const string InstanceMutexName = "EITS.Agent.Manager.SingleInstance";
    const string ActivationEventName = "EITS.Agent.Manager.Activate";

    Mutex? instanceMutex;
    EventWaitHandle? activationEvent;
    RegisteredWaitHandle? activationWait;

    protected override void OnStartup(StartupEventArgs e)
    {
        instanceMutex = new Mutex(true, InstanceMutexName, out var firstInstance);
        if (!firstInstance)
        {
            SignalExistingInstance();
            Shutdown();
            return;
        }

        activationEvent = new EventWaitHandle(false, EventResetMode.AutoReset, ActivationEventName);
        activationWait = ThreadPool.RegisterWaitForSingleObject(
            activationEvent,
            (_, _) => Dispatcher.Invoke(ShowMainWindow),
            null,
            Timeout.Infinite,
            false);

        base.OnStartup(e);
    }

    protected override void OnExit(ExitEventArgs e)
    {
        activationWait?.Unregister(null);
        activationEvent?.Dispose();
        instanceMutex?.Dispose();
        base.OnExit(e);
    }

    static void SignalExistingInstance()
    {
        try
        {
            using var activation = EventWaitHandle.OpenExisting(ActivationEventName);
            activation.Set();
        }
        catch (WaitHandleCannotBeOpenedException) { }
    }

    void ShowMainWindow()
    {
        if (MainWindow is MainWindow manager) manager.ShowManager();
    }
}

