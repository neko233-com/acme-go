$ErrorActionPreference = 'Stop'

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$shellCheckVersion = '0.11.0'
$shellCheck = Get-Command shellcheck -ErrorAction SilentlyContinue

if (-not $shellCheck) {
    $toolsDir = Join-Path $repoRoot '.cache\tools\shellcheck'
    $isWindows = [System.Environment]::OSVersion.Platform -eq 'Win32NT'
    $isMacOS = $PSVersionTable.PSEdition -eq 'Core' -and $IsMacOS

    if ($isWindows) {
        $archiveName = "shellcheck-v$shellCheckVersion.zip"
        $downloadUrl = "https://github.com/koalaman/shellcheck/releases/download/v$shellCheckVersion/$archiveName"
        $shellCheckPath = Join-Path $toolsDir 'shellcheck.exe'
    } elseif ($isMacOS) {
        $archiveName = "shellcheck-v$shellCheckVersion.darwin.x86_64.tar.xz"
        $downloadUrl = "https://github.com/koalaman/shellcheck/releases/download/v$shellCheckVersion/$archiveName"
        $shellCheckPath = Join-Path $toolsDir 'shellcheck'
    } else {
        $archiveName = "shellcheck-v$shellCheckVersion.linux.x86_64.tar.xz"
        $downloadUrl = "https://github.com/koalaman/shellcheck/releases/download/v$shellCheckVersion/$archiveName"
        $shellCheckPath = Join-Path $toolsDir 'shellcheck'
    }

    if (-not (Test-Path $shellCheckPath)) {
        New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
        $archivePath = Join-Path $toolsDir $archiveName

        Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath
        if ($isWindows) {
            Expand-Archive -Path $archivePath -DestinationPath $toolsDir -Force
        } else {
            & tar -xJf $archivePath -C $toolsDir --strip-components=1
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }
        }
    }

    $shellCheckCommand = $shellCheckPath
} else {
    $shellCheckCommand = $shellCheck.Source
}

Set-Location $repoRoot
& go run github.com/rhysd/actionlint/cmd/actionlint@latest -shellcheck $shellCheckCommand .github/workflows/ci.yml .github/workflows/publish-lib.yml .github/workflows/release.yml
