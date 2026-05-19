# Sakurairo Go Deployment

Use `deploy-sakurairo-go.ps1` from Windows PowerShell to deploy through the preferred Git-based flow:

```powershell
.\deploy\deploy-sakurairo-go.ps1
```

The script:

- refuses to deploy a dirty worktree unless `-AllowDirty` is passed;
- runs local `go test ./...`;
- pushes `master` to the `deploy-server` remote;
- asks the server checkout to fetch and reset to `origin/master`;
- runs server-side `go test ./...` and `go build`;
- backs up `/opt/sakurairo-go/sakurairo` and `/opt/sakurairo-go/web`;
- replaces the active binary and `web` directory;
- restarts `sakurairo-go.service`;
- verifies `/api/health` and the public blog response.

Useful validation commands:

```powershell
.\deploy\deploy-sakurairo-go.ps1 -AllowDirty -SkipLocalTests -SkipPush -SkipRemoteDeploy
.\deploy\deploy-sakurairo-go.ps1 -SkipPush
```
