package conf

var (
	App struct {
		// ⚠️ WARNING: Should only be set by the main package (i.e. "zzz.go").
		Version string `ini:"-"`
		Name    string
	}
)
