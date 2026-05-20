# Sakurairo Go Deployment

Use `deploy-sakurairo-go.ps1` from Windows PowerShell to deploy through the preferred Git-based flow:

```powershell
.\deploy\deploy-sakurairo-go.ps1
```

The script:

- refuses to deploy a dirty worktree unless `-AllowDirty` is passed;
- acquires a local lock at `%TEMP%\sakurairo-go-deploy.lock` so overlapping local automation runs fail fast;
- runs local `go test ./...`;
- pushes `master` to the `deploy-server` remote;
- asks the server checkout to fetch and reset to `origin/master`;
- acquires a remote `flock` lock at `/tmp/sakurairo-go-deploy.lock` before touching the server checkout or active app;
- runs server-side `go test ./...` and `go build`;
- stamps the built binary with the deployed commit and UTC build time for `/api/health`;
- backs up `/opt/sakurairo-go/sakurairo` and `/opt/sakurairo-go/web`;
- replaces the active binary and `web` directory;
- restarts `sakurairo-go.service`;
- verifies `/api/health` and the public blog response.

Current production defaults:

- Server: `root@104.194.86.132`
- SSH key: `C:\Users\lhjer\.ssh\koimoe_la_cn2_gia_ed25519`
- Bare repo: `/opt/git/sakurairo-go.git`
- Server build checkout: `/opt/sakurairo-go-src`
- Remote Go: `/usr/local/go/bin/go`

Useful validation commands:

```powershell
.\deploy\deploy-sakurairo-go.ps1 -AllowDirty -SkipLocalTests -SkipPush -SkipRemoteDeploy
.\deploy\deploy-sakurairo-go.ps1 -SkipPush
```

Use `-SkipLock` only for local script debugging when you are certain no other deployment is running.
