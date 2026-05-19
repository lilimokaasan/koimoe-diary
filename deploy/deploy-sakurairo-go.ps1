param(
	[string]$RepoPath = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path,
	[string]$RemoteName = "deploy-server",
	[string]$Branch = "master",
	[string]$SshKey = "C:\Users\lhjer\.ssh\sakurairo_server_ed25519",
	[string]$Server = "ubuntu@124.156.182.231",
	[int]$Port = 22,
	[string]$ServerCheckout = "/home/ubuntu/sakurairo-go-src",
	[string]$AppDir = "/opt/sakurairo-go",
	[string]$ServiceName = "sakurairo-go.service",
	[string]$GoExe = "C:\Program Files\Go\bin\go.exe",
	[string]$LockPath = (Join-Path $env:TEMP "sakurairo-go-deploy.lock"),
	[switch]$AllowDirty,
	[switch]$SkipLock,
	[switch]$SkipLocalTests,
	[switch]$SkipPush,
	[switch]$SkipRemoteDeploy
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Step {
	param(
		[string]$Label,
		[scriptblock]$Action
	)
	Write-Host ""
	Write-Host "==> $Label" -ForegroundColor Magenta
	& $Action
}

function Invoke-Checked {
	param(
		[string]$FilePath,
		[string[]]$ArgumentList,
		[string]$WorkingDirectory = $RepoPath
	)
	Push-Location $WorkingDirectory
	try {
		& $FilePath @ArgumentList
		if ($LASTEXITCODE -ne 0) {
			throw "$FilePath exited with code $LASTEXITCODE"
		}
	} finally {
		Pop-Location
	}
}

$script:lockStream = $null
try {
if (-not $SkipLock) {
	Invoke-Step "Acquire deployment lock" {
		try {
			$script:lockStream = [System.IO.File]::Open($LockPath, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
			$lockInfo = [System.Text.Encoding]::UTF8.GetBytes("pid=$PID started=$(Get-Date -Format o)`n")
			$script:lockStream.Write($lockInfo, 0, $lockInfo.Length)
			$script:lockStream.Flush()
			Write-Host "Lock: $LockPath"
		} catch {
			throw "Another deployment appears to be running, or a stale lock exists at $LockPath. Remove it only after confirming no deploy is active."
		}
	}
}

if (-not (Test-Path -LiteralPath $RepoPath)) {
	throw "RepoPath does not exist: $RepoPath"
}
if (-not (Test-Path -LiteralPath $SshKey)) {
	throw "SSH key does not exist: $SshKey"
}
if (-not (Test-Path -LiteralPath $GoExe)) {
	throw "Go executable does not exist: $GoExe"
}

Invoke-Step "Check git worktree" {
	$status = git -C $RepoPath status --porcelain
	if ($status -and -not $AllowDirty) {
		$status | Write-Host
		throw "Working tree is not clean. Commit or stash changes, or pass -AllowDirty for a dry validation."
	}
	git -C $RepoPath rev-parse --short HEAD
}

if (-not $SkipLocalTests) {
	Invoke-Step "Run local tests" {
		Invoke-Checked -FilePath $GoExe -ArgumentList @("test", "./...") -WorkingDirectory $RepoPath
	}
}

if (-not $SkipPush) {
	Invoke-Step "Push to $RemoteName/$Branch" {
		$oldGitSsh = [Environment]::GetEnvironmentVariable("GIT_SSH_COMMAND", "Process")
		try {
			$sshCommand = "ssh -o BatchMode=yes -i `"$SshKey`" -p $Port"
			[Environment]::SetEnvironmentVariable("GIT_SSH_COMMAND", $sshCommand, "Process")
			Invoke-Checked -FilePath "git" -ArgumentList @("-C", $RepoPath, "push", $RemoteName, $Branch)
		} finally {
			[Environment]::SetEnvironmentVariable("GIT_SSH_COMMAND", $oldGitSsh, "Process")
		}
	}
}

if (-not $SkipRemoteDeploy) {
	Invoke-Step "Build and deploy on server" {
		$head = (git -C $RepoPath rev-parse --short HEAD).Trim()
		$stamp = "deploy_${head}_$(Get-Date -Format yyyyMMddHHmmss)"
		$remoteScript = @"
set -e
if command -v flock >/dev/null 2>&1; then
  exec 9>/tmp/sakurairo-go-deploy.lock
  flock -n 9 || { echo 'Another remote deployment is already running.'; exit 75; }
fi
if [ ! -d '$ServerCheckout/.git' ]; then
  git clone '/home/ubuntu/sakurairo-go.git' '$ServerCheckout'
else
  git -C '$ServerCheckout' fetch origin '$Branch'
  git -C '$ServerCheckout' checkout '$Branch'
  git -C '$ServerCheckout' reset --hard 'origin/$Branch'
fi
cd '$ServerCheckout'
go test ./...
build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
go build -ldflags "-X sakurairo-go/internal/buildinfo.Version=$head -X sakurairo-go/internal/buildinfo.Commit=$head -X sakurairo-go/internal/buildinfo.BuiltAt=$build_time" -o /tmp/sakurairo-built ./cmd/server
cd '$AppDir'
sudo cp sakurairo sakurairo.bak.$stamp
sudo tar -czf web.bak.$stamp.tar.gz web
sudo install -m 755 /tmp/sakurairo-built '$AppDir/sakurairo'
sudo rm -rf '$AppDir/web'
sudo cp -a '$ServerCheckout/web' '$AppDir/web'
sudo systemctl restart '$ServiceName'
sleep 1
systemctl is-active '$ServiceName'
curl -fsS http://127.0.0.1:8081/api/health
curl -s -o /dev/null -w '%{http_code} %{content_type}\n' https://blog.koimoe.com/
"@
		Invoke-Checked -FilePath "ssh" -ArgumentList @("-i", $SshKey, "-p", "$Port", $Server, $remoteScript)
	}
}

Write-Host ""
Write-Host "Deployment flow completed." -ForegroundColor Green
} finally {
	if ($script:lockStream) {
		$script:lockStream.Close()
		$script:lockStream.Dispose()
		if (Test-Path -LiteralPath $LockPath) {
			Remove-Item -LiteralPath $LockPath -Force
		}
	}
}
