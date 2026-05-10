package appindex

type AppIndex struct {
	Names       map[string]struct{}
	BundleIDs   map[string]struct{}
	Vendors     map[string]struct{}
	Executables map[string]struct{}
}

type AppInfo struct {
	BundleID string `plist:"CFBundleIdentifier"`
	Name     string `plist:"CFBundleName"`
	Version  string `plist:"CFBundleShortVersionString"`
}

type AppFamily struct {
	Vendor    string
	BundleIDs []string
	Folders   []string
}

func NewAppIndex() *AppIndex {
	return &AppIndex{
		Names:       make(map[string]struct{}),
		BundleIDs:   make(map[string]struct{}),
		Vendors:     make(map[string]struct{}),
		Executables: make(map[string]struct{}),
	}
}
