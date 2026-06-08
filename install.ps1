param(
  [string]$Version = $env:ACME233_VERSION,
  [string]$InstallDir = $env:ACME233_INSTALL_DIR,
  [switch]$NoPath,
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

function Show-Usage {
  @"
Install acme233 on Windows.

Usage:
  powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/neko233-com/acme233/main/install.ps1 | iex"
  powershell -ExecutionPolicy Bypass -c "`$env:ACME233_VERSION='v0.0.1'; irm https://raw.githubusercontent.com/neko233-com/acme233/main/install.ps1 | iex"

Parameters:
  -Version <version>       Install a release tag. Defaults to latest.
  -InstallDir <dir>        Install directory. Defaults to %LOCALAPPDATA%\acme233\bin.
  -NoPath                  Do not add the install directory to the user PATH.
  -DryRun                  Print what would be installed without changing files.
  -?                       Show this help.

Environment:
  ACME233_VERSION          Same as -Version.
  ACME233_INSTALL_DIR      Same as -InstallDir.
  ACME233_REPO             GitHub repository, owner/name. Defaults to neko233-com/acme233.
  GITHUB_BASE_URL          GitHub base URL. Defaults to https://github.com.
  GITHUB_TOKEN             Optional token for authenticated downloads.
"@
}

if ($args -contains "-?" -or $args -contains "-h" -or $args -contains "--help") {
  Show-Usage
  exit 0
}

if ([string]::IsNullOrWhiteSpace($Version)) {
  $Version = "latest"
}
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
  $InstallDir = Join-Path $env:LOCALAPPDATA "acme233\bin"
}

$repo = if ([string]::IsNullOrWhiteSpace($env:ACME233_REPO)) { "neko233-com/acme233" } else { $env:ACME233_REPO }
$githubBaseUrl = if ([string]::IsNullOrWhiteSpace($env:GITHUB_BASE_URL)) { "https://github.com" } else { $env:GITHUB_BASE_URL.TrimEnd("/") }

if ($Version -ne "latest" -and -not $Version.StartsWith("v")) {
  $Version = "v$Version"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default {
    if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
  }
}

$asset = "acme233_windows_$arch.zip"
if ($Version -eq "latest") {
  $downloadUrl = "$githubBaseUrl/$repo/releases/latest/download/$asset"
} else {
  $downloadUrl = "$githubBaseUrl/$repo/releases/download/$Version/$asset"
}

$headers = @{}
if (-not [string]::IsNullOrWhiteSpace($env:GITHUB_TOKEN)) {
  $headers["Authorization"] = "Bearer $env:GITHUB_TOKEN"
}

Write-Host "Installing $asset to $InstallDir"
if ($DryRun) {
  Write-Host "dry run: would download $downloadUrl"
  Write-Host "dry run: would install acme233.exe into $InstallDir"
  exit 0
}

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("acme233-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $archive = Join-Path $tmp $asset
  Invoke-WebRequest -Uri $downloadUrl -Headers $headers -OutFile $archive
  Expand-Archive -Path $archive -DestinationPath $tmp -Force

  $binary = Get-ChildItem -Path $tmp -Filter "acme233.exe" -Recurse | Select-Object -First 1
  if (-not $binary) {
    throw "acme233.exe not found in release asset"
  }

  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  $target = Join-Path $InstallDir "acme233.exe"
  Copy-Item -Path $binary.FullName -Destination $target -Force

  $currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $pathParts = @()
  if (-not [string]::IsNullOrWhiteSpace($currentUserPath)) {
    $pathParts = $currentUserPath -split ';' | Where-Object { $_ -ne "" }
  }
  $alreadyInPath = $pathParts | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }

  if (-not $NoPath -and -not $alreadyInPath) {
    $newUserPath = if ([string]::IsNullOrWhiteSpace($currentUserPath)) { $InstallDir } else { "$currentUserPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Write-Host "Added $InstallDir to the user PATH."
  }

  if (($env:Path -split ';' | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }).Count -eq 0) {
    $env:Path = "$InstallDir;$env:Path"
  }

  Write-Host "acme233 installed: $target"
  Write-Host "Run 'acme233 version' in this terminal, or open a new terminal if PATH was just updated."
} finally {
  Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
