# FluxQuery Agent Installer (Windows)
# Repo: https://github.com/lexiumindustries/fluxquery-backend

$ErrorActionPreference = "Stop"

$Repo = "lexiumindustries/fluxquery-backend"
$BinaryName = "fluxquery-agent-windows-amd64.exe"
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/$BinaryName"
$Output = "fluxquery-agent.exe"

Write-Host "FluxQuery Agent Installer (Windows)"
Write-Host "==================================="
Write-Host "Downloading latest release from $Repo..."

# Create Install Directory
$InstallDir = "$env:LOCALAPPDATA\FluxQuery\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$OutputPath = Join-Path $InstallDir $Output

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutputPath
} catch {
    Write-Error "Failed to download agent. Please check your internet connection."
    exit 1
}

# Add to PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to User PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "PATH updated. You may need to restart your terminal."
}

Write-Host ""
Write-Host "Successfully installed to $OutputPath"
Write-Host "Run it from anywhere:"
Write-Host "  fluxquery-agent"
