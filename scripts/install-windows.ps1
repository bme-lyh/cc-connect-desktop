$ErrorActionPreference = 'Stop'

try {
    $sourceDir = $PSScriptRoot
    $installDir = Join-Path $env:LOCALAPPDATA 'Programs\cc-connect'
    $dataDir = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.cc-connect'
    $configPath = Join-Path $dataDir 'config.toml'

    $null = New-Item -ItemType Directory -Path $installDir -Force
    $null = New-Item -ItemType Directory -Path $dataDir -Force
    Copy-Item -LiteralPath (Join-Path $sourceDir 'cc-connect.exe') -Destination (Join-Path $installDir 'cc-connect.exe') -Force
    Copy-Item -LiteralPath (Join-Path $sourceDir 'cc-connect-launch.ps1') -Destination (Join-Path $installDir 'cc-connect-launch.ps1') -Force
    Copy-Item -LiteralPath (Join-Path $sourceDir 'README.md') -Destination (Join-Path $installDir 'README.md') -Force
    Copy-Item -LiteralPath (Join-Path $sourceDir 'LICENSE') -Destination (Join-Path $installDir 'LICENSE') -Force

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

    $exe = Join-Path $installDir 'cc-connect.exe'
    $daemon = Start-Process -FilePath $exe -ArgumentList 'daemon','install' -WorkingDirectory $installDir -WindowStyle Hidden -Wait -PassThru
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

    $null = $shell.Popup("CC-Connect is installed.`n`nDesktop shortcut: CC-Connect`nConfig: $configPath", 0, 'CC-Connect', 64)
} catch {
    $shell = New-Object -ComObject WScript.Shell
    $logPath = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.cc-connect\logs\cc-connect.log'
    $null = $shell.Popup("$($_.Exception.Message)`n`nLog: $logPath", 0, 'CC-Connect installation failed', 16)
    exit 1
}
