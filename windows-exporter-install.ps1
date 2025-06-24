# Windows Exporter Installation Script
# This script downloads and installs Windows Exporter for Prometheus

# Set TLS 1.2 for compatibility with modern HTTPS sites
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Define variables
$url = "https://github.com/prometheus-community/windows_exporter/releases/download/v0.25.0/windows_exporter-0.25.0-amd64.msi"
$output = "$env:TEMP\windows_exporter.msi"

# Download Windows Exporter
Write-Host "Downloading Windows Exporter..."
try {
    # Method 1: Using Invoke-WebRequest
    Invoke-WebRequest -Uri $url -OutFile $output
    Write-Host "Download successful using Invoke-WebRequest."
} 
catch {
    Write-Host "Invoke-WebRequest failed. Trying alternative download method..."
    try {
        # Method 2: Using .NET WebClient
        $client = New-Object System.Net.WebClient
        $client.DownloadFile($url, $output)
        Write-Host "Download successful using WebClient."
    }
    catch {
        Write-Host "ERROR: Both download methods failed."
        Write-Host "Please download the file manually from:"
        Write-Host $url
        Write-Host "and save it to: $output"
        exit 1
    }
}

# Install Windows Exporter
Write-Host "Installing Windows Exporter..."
try {
    Start-Process msiexec.exe -ArgumentList "/i $output ENABLED_COLLECTORS=cpu,memory,logical_disk,os,system,net,tcp LISTEN_PORT=9182 /quiet" -Wait
    Write-Host "Installation completed successfully."
} 
catch {
    Write-Host "ERROR: Installation failed."
    Write-Host $_.Exception.Message
    exit 1
}

# Configure Windows Firewall
Write-Host "Configuring Windows Firewall..."
try {
    New-NetFirewallRule -DisplayName "Windows Exporter" -Direction Inbound -LocalPort 9182 -Protocol TCP -Action Allow
    Write-Host "Firewall rule created successfully."
} 
catch {
    Write-Host "ERROR: Failed to create firewall rule."
    Write-Host $_.Exception.Message
    Write-Host "Please create the firewall rule manually to allow inbound connections to port 9182."
}

# Verify service status
Write-Host "Verifying Windows Exporter service..."
$service = Get-Service windows_exporter -ErrorAction SilentlyContinue
if ($service -and $service.Status -eq "Running") {
    Write-Host "Windows Exporter service is running."
} else {
    Write-Host "WARNING: Windows Exporter service is not running."
    Write-Host "Please check the service status manually."
}

# Test metrics endpoint
Write-Host "Testing metrics endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9182/metrics" -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "Metrics endpoint is accessible."
    } else {
        Write-Host "WARNING: Metrics endpoint returned status code $($response.StatusCode)."
    }
} 
catch {
    Write-Host "WARNING: Could not access metrics endpoint."
    Write-Host "Please check if the service is running and the port is accessible."
}

Write-Host "Setup complete. Windows Exporter should be running on port 9182."
Write-Host "To verify from Prometheus, check the target status for 'windows-server' job." 