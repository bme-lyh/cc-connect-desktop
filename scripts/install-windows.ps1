[CmdletBinding()]
param(
    [string]$ExePath,
    [string]$GoPath,
    [switch]$Quiet
)

$ErrorActionPreference = 'Stop'

function Resolve-PayloadFile {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$ScriptDir,
        [Parameter(Mandatory = $true)][string]$RepoRoot
    )

    $packagedPath = Join-Path $ScriptDir $Name
    if (Test-Path -LiteralPath $packagedPath) {
        return $packagedPath
    }

    $repositoryPath = Join-Path $RepoRoot $Name
    if (Test-Path -LiteralPath $repositoryPath) {
        return $repositoryPath
    }

    throw "Required installer file was not found: $Name"
}

function Resolve-GoExecutable {
    param([string]$RequestedPath)

    if ($RequestedPath) {
        return (Resolve-Path -LiteralPath $RequestedPath -ErrorAction Stop).Path
    }

    $command = Get-Command 'go' -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        return $command.Source
    }

    return $null
}

function Resolve-ExecutableSource {
    param(
        [string]$RequestedPath,
        [string]$RequestedGoPath,
        [Parameter(Mandatory = $true)][string]$ScriptDir,
        [Parameter(Mandatory = $true)][string]$RepoRoot
    )

    if ($RequestedPath) {
        return (Resolve-Path -LiteralPath $RequestedPath -ErrorAction Stop).Path
    }

    $packagedExe = Join-Path $ScriptDir 'cc-connect.exe'
    if (Test-Path -LiteralPath $packagedExe) {
        return $packagedExe
    }

    $builtExe = Join-Path $RepoRoot 'build\cc-connect.exe'
    if (Test-Path -LiteralPath $builtExe) {
        return $builtExe
    }

    $goModule = Join-Path $RepoRoot 'go.mod'
    $mainPackage = Join-Path $RepoRoot 'cmd\cc-connect'
    if (-not (Test-Path -LiteralPath $goModule) -or -not (Test-Path -LiteralPath $mainPackage)) {
        throw "cc-connect.exe was not found next to the installer. Download and extract the Windows portable package before running install-windows.ps1."
    }

    $goExe = Resolve-GoExecutable -RequestedPath $RequestedGoPath
    if (-not $goExe) {
        throw "This is a source checkout and cc-connect.exe has not been built. Install Go or pass -GoPath, or download the Windows portable package."
    }

    $webIndex = Join-Path $RepoRoot 'web\dist\index.html'
    if (-not (Test-Path -LiteralPath $webIndex)) {
        $npm = Get-Command 'npm.cmd' -ErrorAction SilentlyContinue
        if ($null -eq $npm) {
            throw "The web interface has not been built. Install Node.js, run npm install and npm run build in the web directory, then retry."
        }
        Push-Location (Join-Path $RepoRoot 'web')
        try {
            if (-not (Test-Path -LiteralPath 'node_modules')) {
                & $npm.Source install
                if ($LASTEXITCODE -ne 0) { throw "npm install failed with code $LASTEXITCODE." }
            }
            & $npm.Source run build
            if ($LASTEXITCODE -ne 0) { throw "Frontend build failed with code $LASTEXITCODE." }
        } finally {
            Pop-Location
        }
    }

    $buildDir = Split-Path -Parent $builtExe
    $null = New-Item -ItemType Directory -Path $buildDir -Force
    Push-Location $RepoRoot
    try {
        & $goExe build -trimpath -o $builtExe ./cmd/cc-connect
        if ($LASTEXITCODE -ne 0) { throw "Go build failed with code $LASTEXITCODE." }
    } finally {
        Pop-Location
    }

    return $builtExe
}

try {
    $sourceDir = $PSScriptRoot
    $repoRoot = (Resolve-Path (Join-Path $sourceDir '..')).Path
    $exeSource = Resolve-ExecutableSource -RequestedPath $ExePath -RequestedGoPath $GoPath -ScriptDir $sourceDir -RepoRoot $repoRoot
    $launchSource = Resolve-PayloadFile -Name 'cc-connect-launch.ps1' -ScriptDir $sourceDir -RepoRoot $repoRoot
    $readmeSource = Resolve-PayloadFile -Name 'README.md' -ScriptDir $sourceDir -RepoRoot $repoRoot
    $licenseSource = Resolve-PayloadFile -Name 'LICENSE' -ScriptDir $sourceDir -RepoRoot $repoRoot
    $installDir = Join-Path $env:LOCALAPPDATA 'Programs\cc-connect'
    $dataDir = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.cc-connect'
    $configPath = Join-Path $dataDir 'config.toml'

    $null = New-Item -ItemType Directory -Path $installDir -Force
    $null = New-Item -ItemType Directory -Path $dataDir -Force
    $installedExe = Join-Path $installDir 'cc-connect.exe'
    if (Test-Path -LiteralPath $installedExe) {
        $stop = Start-Process -FilePath $installedExe -ArgumentList 'daemon','stop' -WorkingDirectory $installDir -WindowStyle Hidden -Wait -PassThru
        if ($stop.ExitCode -eq 0) {
            Start-Sleep -Milliseconds 500
        }
        $runningProcesses = Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $installedExe }
        foreach ($process in $runningProcesses) {
            Stop-Process -Id $process.ProcessId -Force -ErrorAction Stop
        }
        if ($runningProcesses) {
            Start-Sleep -Milliseconds 500
        }
    }
    Copy-Item -LiteralPath $exeSource -Destination $installedExe -Force
    Copy-Item -LiteralPath $launchSource -Destination (Join-Path $installDir 'cc-connect-launch.ps1') -Force
    Copy-Item -LiteralPath $readmeSource -Destination (Join-Path $installDir 'README.md') -Force
    Copy-Item -LiteralPath $licenseSource -Destination (Join-Path $installDir 'LICENSE') -Force

    if (-not (Test-Path -LiteralPath $configPath)) {
        $tokenBytes = New-Object byte[] 24
        $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
        $rng.GetBytes($tokenBytes)
        $rng.Dispose()
        $token = -join ($tokenBytes | ForEach-Object { $_.ToString('x2') })
        $config = @"
# CC-Connect local desktop configuration
[log]
level = "info"

[management]
enabled = true
port = 9820
token = "$token"
"@
        [IO.File]::WriteAllText($configPath, $config, [Text.UTF8Encoding]::new($false))
    }

    $exe = $installedExe
    $daemonArgs = @(
        'daemon', 'install',
        '--config', "`"$configPath`"",
        '--work-dir', "`"$dataDir`"",
        '--force'
    )
    $daemon = Start-Process -FilePath $exe -ArgumentList $daemonArgs -WorkingDirectory $installDir -WindowStyle Hidden -Wait -PassThru
    if ($daemon.ExitCode -ne 0) {
        throw "Task Scheduler installation failed with code $($daemon.ExitCode)."
    }

    $powershell = Join-Path $PSHOME 'powershell.exe'
    if (-not (Test-Path -LiteralPath $powershell)) { $powershell = 'powershell.exe' }
    $shortcutArgs = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$(Join-Path $installDir 'cc-connect-launch.ps1')`""
    $shell = New-Object -ComObject WScript.Shell

    $desktopShortcut = $shell.CreateShortcut((Join-Path ([Environment]::GetFolderPath('Desktop')) 'CC-Connect.lnk'))
    $desktopShortcut.TargetPath = $powershell
    $desktopShortcut.Arguments = $shortcutArgs
    $desktopShortcut.WorkingDirectory = $installDir
    $desktopShortcut.IconLocation = "$exe,0"
    $desktopShortcut.Save()

    $startMenuDir = Join-Path ([Environment]::GetFolderPath('StartMenu')) 'Programs\CC-Connect'
    $null = New-Item -ItemType Directory -Path $startMenuDir -Force
    $startShortcut = $shell.CreateShortcut((Join-Path $startMenuDir 'CC-Connect.lnk'))
    $startShortcut.TargetPath = $powershell
    $startShortcut.Arguments = $shortcutArgs
    $startShortcut.WorkingDirectory = $installDir
    $startShortcut.IconLocation = "$exe,0"
    $startShortcut.Save()

    if ($Quiet) {
        Write-Output "CC-Connect is installed. Config: $configPath"
    } else {
        $null = $shell.Popup("CC-Connect is installed.`n`nDesktop shortcut: CC-Connect`nConfig: $configPath", 0, 'CC-Connect', 64)
    }
} catch {
    $logPath = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.cc-connect\logs\cc-connect.log'
    if ($Quiet) {
        Write-Error "$($_.Exception.Message) Log: $logPath"
    } else {
        $shell = New-Object -ComObject WScript.Shell
        $null = $shell.Popup("$($_.Exception.Message)`n`nLog: $logPath", 0, 'CC-Connect installation failed', 16)
    }
    exit 1
}
