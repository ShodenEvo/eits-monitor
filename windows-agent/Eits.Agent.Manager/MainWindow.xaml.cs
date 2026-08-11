using Eits.Agent.Shared;
using System.Diagnostics;
using System.Windows;
using Forms = System.Windows.Forms;
using MessageBox = System.Windows.MessageBox;
using Brush = System.Windows.Media.Brush;
using Color = System.Windows.Media.Color;
using SolidColorBrush = System.Windows.Media.SolidColorBrush;

namespace Eits.Agent.Manager;

public partial class MainWindow : Window
{
    readonly Forms.NotifyIcon tray;
    readonly System.Windows.Threading.DispatcherTimer timer;
    bool exit;

    public MainWindow()
    {
        InitializeComponent();
        var trayIcon = System.Drawing.Icon.ExtractAssociatedIcon(Environment.ProcessPath!) ?? System.Drawing.SystemIcons.Application;
        tray = new Forms.NotifyIcon { Icon = trayIcon, Text = "EITS Agent Manager", Visible = true };
        var menu = new Forms.ContextMenuStrip();
        menu.Items.Add("Open", null, (_, _) => Dispatcher.Invoke(ShowManager));
        menu.Items.Add("Start service", null, async (_, _) => await ControlAsync("start"));
        menu.Items.Add("Stop service", null, async (_, _) => await ControlAsync("stop"));
        menu.Items.Add("Restart service", null, async (_, _) => await ControlAsync("restart"));
        menu.Items.Add(new Forms.ToolStripSeparator());
        menu.Items.Add("Exit", null, (_, _) => Dispatcher.Invoke(() => { exit = true; Close(); }));
        tray.ContextMenuStrip = menu;
        tray.DoubleClick += (_, _) => Dispatcher.Invoke(ShowManager);
        timer = new() { Interval = TimeSpan.FromSeconds(10) };
        timer.Tick += async (_, _) => await RefreshStatusAsync();
        timer.Start();
        Loaded += async (_, _) => await RefreshStatusAsync();
    }

    public void ShowManager() { Show(); WindowState = WindowState.Normal; Activate(); }
    protected override void OnClosing(System.ComponentModel.CancelEventArgs e) { if (!exit) { e.Cancel = true; Hide(); return; } tray.Dispose(); base.OnClosing(e); }

    async Task RefreshStatusAsync()
    {
        var status = await Task.Run(AgentStatusReader.Read);
        ServiceStateText.Text = status.ServiceState;
        var running = status.ServiceState == "Running";
        var stateBrush = BrushFrom(running ? "#45E6D4" : "#FF6B7A");
        ServiceStateText.Foreground = stateBrush;
        StatusDot.Fill = stateBrush;
        HeaderStatusDot.Fill = stateBrush;
        HeaderStatusText.Text = running ? "SYSTEM ONLINE" : status.ServiceState.ToUpperInvariant();
        HeaderStatusText.Foreground = stateBrush;
        RegistrationText.Text = status.Registered ? $"Registered - {status.AgentId}" : "Not registered";
        DeviceText.Text = status.DeviceName;
        ServerText.Text = status.ServerUrl;
        ActivityText.Text = string.IsNullOrWhiteSpace(status.LastActivity) ? "No activity recorded." : status.LastActivity;
    }

    static Brush BrushFrom(string hex) => new SolidColorBrush((Color)System.Windows.Media.ColorConverter.ConvertFromString(hex));

    async Task ControlAsync(string action)
    {
        try { IsEnabled = false; ServiceStateText.Text = $"{action}..."; await ElevatedControl.RunAsync(action); await RefreshStatusAsync(); }
        catch (Exception ex) { MessageBox.Show(ex.Message, "EITS Agent", MessageBoxButton.OK, MessageBoxImage.Error); }
        finally { IsEnabled = true; }
    }

    async void Refresh_Click(object sender, RoutedEventArgs e) => await RefreshStatusAsync();
    async void Start_Click(object sender, RoutedEventArgs e) => await ControlAsync("start");
    async void Stop_Click(object sender, RoutedEventArgs e) => await ControlAsync("stop");
    async void Restart_Click(object sender, RoutedEventArgs e) => await ControlAsync("restart");
    void Dashboard_Click(object sender, RoutedEventArgs e) => Process.Start(new ProcessStartInfo(AgentConfig.Load().ServerUrl) { UseShellExecute = true });
    void Logs_Click(object sender, RoutedEventArgs e) => Process.Start(new ProcessStartInfo("notepad.exe", AgentPaths.LogFile) { UseShellExecute = true });
    void Data_Click(object sender, RoutedEventArgs e) => Process.Start(new ProcessStartInfo("explorer.exe", AgentPaths.DataDirectory) { UseShellExecute = true });
    void ChangeConnection_Click(object sender, RoutedEventArgs e) { var dialog = new ConnectionWindow { Owner = this }; if (dialog.ShowDialog() == true) _ = SaveConnectionAsync(dialog); }

    async Task SaveConnectionAsync(ConnectionWindow dialog)
    {
        try
        {
            IsEnabled = false;
            await ElevatedControl.ReconfigureAsync(new(dialog.ServerUrl, dialog.Token, dialog.DeviceNameValue, dialog.AllowHttp));
            await RefreshStatusAsync();
            MessageBox.Show("Connection saved. The agent is connected to the server.", "Connection settings", MessageBoxButton.OK, MessageBoxImage.Information);
        }
        catch (Exception ex) { MessageBox.Show(ex.Message, "Connection settings", MessageBoxButton.OK, MessageBoxImage.Error); }
        finally { IsEnabled = true; }
    }
}
