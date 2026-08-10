using Eits.Agent.Shared;
using System.Windows;
using MessageBox = System.Windows.MessageBox;

namespace Eits.Agent.Manager;

public partial class ConnectionWindow : Window
{
    public string ServerUrl { get; private set; } = "";
    public string Token { get; private set; } = "";
    public string DeviceNameValue { get; private set; } = "";
    public bool AllowHttp { get; private set; }

    public ConnectionWindow()
    {
        InitializeComponent();
        try
        {
            var config = AgentConfig.Load();
            ServerBox.Text = config.ServerUrl;
            DeviceBox.Text = config.DeviceName;
            AllowHttpBox.IsChecked = config.AllowInsecureHttp;
        }
        catch { DeviceBox.Text = Environment.MachineName; }
    }

    private void Save_Click(object sender, RoutedEventArgs e)
    {
        var raw = ServerBox.Text.Trim();
        var allowHttp = AllowHttpBox.IsChecked == true;
        if (!raw.Contains("://", StringComparison.Ordinal)) raw = (allowHttp ? "http://" : "https://") + raw;
        if (!Uri.TryCreate(raw, UriKind.Absolute, out var uri) ||
            (uri.Scheme != Uri.UriSchemeHttp && uri.Scheme != Uri.UriSchemeHttps) ||
            string.IsNullOrWhiteSpace(uri.Host))
        {
            MessageBox.Show("Enter a server address such as https://server.example or http://10.0.0.4:8088.", Title, MessageBoxButton.OK, MessageBoxImage.Warning);
            return;
        }
        if (uri.Scheme == Uri.UriSchemeHttp && !allowHttp)
        {
            MessageBox.Show("Enable insecure HTTP for an http:// server, or use HTTPS.", Title, MessageBoxButton.OK, MessageBoxImage.Warning);
            return;
        }
        if (string.IsNullOrWhiteSpace(TokenBox.Password) || string.IsNullOrWhiteSpace(DeviceBox.Text))
        {
            MessageBox.Show("Enrollment token and device name are required.", Title, MessageBoxButton.OK, MessageBoxImage.Warning);
            return;
        }
        ServerUrl = raw.TrimEnd('/'); Token = TokenBox.Password; DeviceNameValue = DeviceBox.Text.Trim(); AllowHttp = allowHttp;
        DialogResult = true;
    }
}
