#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Print Service Installation Script for Windows

.DESCRIPTION
    This script installs the ERP Print Service as a Windows Service.
    Must be run as Administrator.

.PARAMETER Prefix
    Installation prefix directory (default: C:\Program Files\PrintService)

.PARAMETER ConfigFile
    Path to config file (default: PREFIX\config.yaml)

.PARAMETER ServiceName
    Name of the Windows service (default: PrintService)

.PARAMETER ServiceDisplayName
    Display name of the Windows service (default: ERP Print Service)

.PARAMETER Uninstall
    Uninstall the service

.EXAMPLE
    .\install.ps1
    Install with default settings

.EXAMPLE
    .\install.ps1 -Prefix "D:\PrintService"
    Install to custom directory

.EXAMPLE
    .\install.ps1 -Uninstall
    Uninstall the service
#>

[CmdletBinding()]
param(
    [string]$Prefix = "C:\Program Files\PrintService",
    [string]$ConfigFile = "",
    [string]$ServiceName = "PrintService",
    [string]$ServiceDisplayName = "ERP Print Service",
    [switch]$Uninstall
)

# Strict mode
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinaryName = "print-service.exe"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-ErrorMsg {
    param([string]$Message)
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Find the binary
function Find-Binary {
    $searchPaths = @(
        (Join-Path $ScriptDir $BinaryName),
        (Join-Path $ScriptDir "print-service-windows-amd64.exe"),
        (Join-Path (Split-Path -Parent $ScriptDir) "dist\print-service-windows-amd64.exe")
    )

    foreach ($path in $searchPaths) {
        if (Test-Path $path) {
            return $path
        }
    }

    Write-ErrorMsg "Binary not found. Searched:"
    foreach ($path in $searchPaths) {
        Write-ErrorMsg "  - $path"
    }
    exit 1
}

# Install files
function Install-Files {
    Write-Info "Installing files to $Prefix..."

    # Create directories
    New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $Prefix "logs") | Out-Null

    # Find and copy binary
    $binarySrc = Find-Binary
    $binaryDst = Join-Path $Prefix $BinaryName
    Copy-Item -Path $binarySrc -Destination $binaryDst -Force
    Write-Info "Installed binary: $binaryDst"

    # Verify the binary works
    try {
        $version = & $binaryDst -version 2>&1
        Write-Info "Binary version: $($version[0])"
    }
    catch {
        Write-ErrorMsg "Binary verification failed: $_"
        exit 1
    }

    # Copy configuration if it doesn't exist
    if ([string]::IsNullOrEmpty($ConfigFile)) {
        $ConfigFile = Join-Path $Prefix "config.yaml"
    }

    if (-not (Test-Path $ConfigFile)) {
        $configSrcPaths = @(
            (Join-Path $ScriptDir "config.example.yaml"),
            (Join-Path (Split-Path -Parent $ScriptDir) "configs\config.example.yaml")
        )

        $configSrc = $null
        foreach ($path in $configSrcPaths) {
            if (Test-Path $path) {
                $configSrc = $path
                break
            }
        }

        if ($configSrc) {
            Copy-Item -Path $configSrc -Destination $ConfigFile -Force
            Write-Info "Created configuration: $ConfigFile"
            Write-Warning "Please edit $ConfigFile with your settings"
        }
        else {
            Write-Warning "No config template found. You need to create $ConfigFile manually."
        }
    }
    else {
        Write-Info "Configuration already exists: $ConfigFile"
    }

    Write-Info "Files installed successfully"
    return $binaryDst
}

# Create Windows Service using sc.exe
function Install-PrintService {
    param([string]$BinaryPath)

    Write-Info "Creating Windows Service..."

    # Check if service already exists
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

    if ($existingService) {
        Write-Info "Service already exists, updating..."

        # Stop if running
        if ($existingService.Status -eq "Running") {
            Write-Info "Stopping existing service..."
            Stop-Service -Name $ServiceName -Force
            Start-Sleep -Seconds 2
        }

        # Delete the existing service
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }

    # Create the service using sc.exe
    # Use NetworkService for better security (LocalSystem has excessive privileges)
    $binPathWithArgs = "`"$BinaryPath`" -config `"$ConfigFile`""

    $result = sc.exe create $ServiceName `
        binPath= $binPathWithArgs `
        DisplayName= $ServiceDisplayName `
        start= auto `
        obj= "NT AUTHORITY\NetworkService"

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorMsg "Failed to create service: $result"
        exit 1
    }

    Write-Info "Service created successfully"

    # Set service description
    sc.exe description $ServiceName "ERP Print Service - Handles printing jobs from the ERP system" | Out-Null

    # Configure failure recovery
    sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
    Write-Info "Configured service recovery options"

    # Set delayed auto-start
    sc.exe config $ServiceName start= delayed-auto | Out-Null
    Write-Info "Configured delayed auto-start"
}

# Start the service
function Start-PrintService {
    Write-Info "Starting service..."

    try {
        Start-Service -Name $ServiceName
        Start-Sleep -Seconds 3

        $service = Get-Service -Name $ServiceName
        if ($service.Status -eq "Running") {
            Write-Info "Service started successfully"

            # Check health endpoint
            $maxAttempts = 10
            $attempt = 1
            while ($attempt -le $maxAttempts) {
                try {
                    $health = Invoke-RestMethod -Uri "http://localhost:9999/health" -TimeoutSec 2
                    Write-Info "Service is running and healthy"
                    Write-Info "  Status: $($health.status)"
                    Write-Info "  Version: $($health.version)"
                    return
                }
                catch {
                    Start-Sleep -Seconds 1
                    $attempt++
                }
            }

            Write-Warning "Service started but health check not responding"
            Write-Warning "Check logs in: $(Join-Path $Prefix 'logs')"
        }
        else {
            Write-Warning "Service failed to start. Status: $($service.Status)"
            Write-Warning "Check Event Viewer for details"
        }
    }
    catch {
        Write-Warning "Failed to start service: $_"
        Write-Warning "Check configuration and Event Viewer for details"
    }
}

# Uninstall the service
function Uninstall-PrintService {
    Write-Info "Uninstalling Print Service..."

    # Stop and remove service
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq "Running") {
            Write-Info "Stopping service..."
            Stop-Service -Name $ServiceName -Force
            Start-Sleep -Seconds 2
        }

        Write-Info "Removing service..."
        sc.exe delete $ServiceName | Out-Null
        Write-Info "Service removed"
    }
    else {
        Write-Info "Service not found"
    }

    # Ask about removing files
    Write-Host ""
    $response = Read-Host "Remove installation directory $Prefix? [y/N]"
    if ($response -eq "y" -or $response -eq "Y") {
        Remove-Item -Path $Prefix -Recurse -Force
        Write-Info "Removed $Prefix"
    }
    else {
        Write-Info "Kept $Prefix"
    }

    Write-Info "Uninstall complete"
}

# Print summary
function Show-Summary {
    Write-Host ""
    Write-Host "========================================"
    Write-Host "  Print Service Installation Complete"
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Installation directory: $Prefix"
    Write-Host "Configuration file:     $ConfigFile"
    Write-Host "Service name:           $ServiceName"
    Write-Host ""
    Write-Host "Service commands (run as Administrator):"
    Write-Host "  Start:   Start-Service -Name $ServiceName"
    Write-Host "  Stop:    Stop-Service -Name $ServiceName"
    Write-Host "  Status:  Get-Service -Name $ServiceName"
    Write-Host "  Logs:    Get-Content $(Join-Path $Prefix 'logs\stderr.log') -Wait"
    Write-Host ""
    Write-Host "Or use services.msc for GUI management"
    Write-Host ""
    Write-Host "Health check:"
    Write-Host "  Invoke-RestMethod -Uri 'http://localhost:9999/health'"
    Write-Host ""

    # Show version
    $binaryPath = Join-Path $Prefix $BinaryName
    Write-Host "Version:"
    & $binaryPath -version
    Write-Host ""
}

# Add firewall rule for health endpoint
function Add-FirewallRule {
    Write-Info "Adding firewall rule for health endpoint..."

    $ruleName = "PrintService Health Check"

    # Remove existing rule if any
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if ($existingRule) {
        Remove-NetFirewallRule -DisplayName $ruleName
    }

    # Add new rule
    New-NetFirewallRule -DisplayName $ruleName `
        -Direction Inbound `
        -Protocol TCP `
        -LocalPort 9999 `
        -Action Allow `
        -Profile Domain,Private `
        -Description "Allow access to ERP Print Service health endpoint" | Out-Null

    Write-Info "Firewall rule added for port 9999"
}

# Main function
function Main {
    # Check administrator
    if (-not (Test-Administrator)) {
        Write-ErrorMsg "This script must be run as Administrator"
        exit 1
    }

    # Set default config file
    if ([string]::IsNullOrEmpty($ConfigFile)) {
        $script:ConfigFile = Join-Path $Prefix "config.yaml"
    }

    if ($Uninstall) {
        Uninstall-PrintService
        return
    }

    # Install
    $binaryPath = Install-Files
    Install-PrintService -BinaryPath $binaryPath
    Add-FirewallRule
    Start-PrintService
    Show-Summary
}

# Run main
Main
