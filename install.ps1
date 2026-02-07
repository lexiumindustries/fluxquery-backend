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

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $Output
} catch {
    Write-Error "Failed to download agent. Please check your internet connection or try identifying the specific version."
    exit 1
}

Write-Host ""
Write-Host "Successfully downloaded $Output"
Write-Host "Run it with:"
Write-Host "  ./$Output"
