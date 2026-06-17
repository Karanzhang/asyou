# asyou Client Install Script (Windows PowerShell)
# Usage: pwsh -c "iwr -UseBasicParsing https://raw.githubusercontent.com/Karanzhang/asyou/main/scripts/install-client.ps1 | iex"
# Or set ASYOU_SERVER before running: $env:ASYOU_SERVER="https://asyou.karanz.com"

param(
    [string]$Server = "https://asyou.karanz.com"
)

Write-Host "=== asyou Client Installer (Windows) ===" -ForegroundColor Cyan
Write-Host "Server: $Server" -ForegroundColor Cyan
Write-Host ""

# Step 1: Check prerequisites
$hasGo = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
$hasGit = $null -ne (Get-Command git -ErrorAction SilentlyContinue)

Write-Host "[1/4] Checking prerequisites..." -ForegroundColor Yellow
if (-not $hasGo) {
    Write-Host "  Go not found. Would you like to install Go? (y/n)" -ForegroundColor Yellow
    $ans = Read-Host
    if ($ans -eq "y") {
        Write-Host "  Downloading Go installer..." -ForegroundColor Yellow
        $goUrl = "https://go.dev/dl/go1.24.1.windows-amd64.msi"
        $goInstaller = "$env:TEMP\go.msi"
        Invoke-WebRequest -UseBasicParsing $goUrl -OutFile $goInstaller
        Write-Host "  Installing Go..." -ForegroundColor Yellow
        Start-Process msiexec -ArgumentList "/i $goInstaller /quiet" -Wait
        $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine")
        Write-Host "  Go installed!" -ForegroundColor Green
    } else {
        Write-Host "  Please install Go manually: https://go.dev/dl/" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "  Go: $(go version)" -ForegroundColor Green
}

# Step 2: Get recommended frpc version from server
Write-Host "[2/4] Getting frpc version from server..." -ForegroundColor Yellow
try {
    $verJson = Invoke-WebRequest -UseBasicParsing "$Server/api/v1/version" -TimeoutSec 10 | ConvertFrom-Json
    $frpcVer = $verJson.recommended_frpc_version
    Write-Host "  Recommended frpc version: $frpcVer" -ForegroundColor Green
} catch {
    Write-Host "  Cannot connect to $Server, using default version 0.69.1" -ForegroundColor Yellow
    $frpcVer = "0.69.1"
}

# Step 3: Install frpc
Write-Host "[3/4] Installing frpc v$frpcVer..." -ForegroundColor Yellow
$frpUrl = "https://github.com/fatedier/frp/releases/download/v${frpcVer}/frp_${frpcVer}_windows_amd64.zip"
$frpZip = "$env:TEMP\frp.zip"
$frpDir = "$env:TEMP\frp_$frpcVer"

try {
    Invoke-WebRequest -UseBasicParsing $frpUrl -OutFile $frpZip -TimeoutSec 60
    Expand-Archive -Path $frpZip -DestinationPath $frpDir -Force
    $frpcPath = "$env:SystemRoot\System32\frpc.exe"
    Copy-Item "$frpDir\frp_${frpcVer}_windows_amd64\frpc.exe" $frpcPath -Force
    Write-Host "  frpc installed to: $frpcPath" -ForegroundColor Green
} catch {
    Write-Host "  Failed to download frpc: $_" -ForegroundColor Red
    exit 1
} finally {
    Remove-Item $frpZip -ErrorAction SilentlyContinue
    Remove-Item $frpDir -Recurse -ErrorAction SilentlyContinue
}

# Step 4: Build asyou CLI
Write-Host "[4/4] Building asyou CLI..." -ForegroundColor Yellow
$cliDir = "$env:TEMP\asyou-cli"
if (Test-Path $cliDir) { Remove-Item $cliDir -Recurse -Force }

try {
    # Try to use pre-built binary first, fall back to source build
    git clone --depth 1 https://github.com/Karanzhang/asyou.git $cliDir 2>$null
    if ($LASTEXITCODE -eq 0) {
        Push-Location $cliDir\cli
        go build -o "$env:USERPROFILE\go\bin\asyou.exe" .
        Pop-Location
        Write-Host "  CLI built: $env:USERPROFILE\go\bin\asyou.exe" -ForegroundColor Green
    } else {
        # If git fails (no network), try to build from local source
        $localCli = "d:\project\asyou\cli"
        if (Test-Path $localCli) {
            Push-Location $localCli
            go build -o "$env:USERPROFILE\go\bin\asyou.exe" .
            Pop-Location
            Write-Host "  CLI built from local source" -ForegroundColor Green
        } else {
            Write-Host "  Cannot build CLI. Please build manually: cd cli && go build" -ForegroundColor Red
        }
    }
} finally {
    Remove-Item $cliDir -Recurse -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "=== Installation Complete! ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Green
Write-Host "  1. Login:     asyou login --s $Server <email> <password>"
Write-Host "  2. Expose:    asyou expose 3000 --n my-app"
Write-Host "  3. Check:     asyou list"
Write-Host ""
