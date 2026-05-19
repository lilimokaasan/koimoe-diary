package buildinfo

var (
	Version = "dev"
	Commit  = "dev"
	BuiltAt = "unknown"
)

func Snapshot() map[string]string {
	return map[string]string{
		"version":  Version,
		"commit":   Commit,
		"built_at": BuiltAt,
	}
}
