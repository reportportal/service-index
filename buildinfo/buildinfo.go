package buildinfo

var (
	// Branch contains the current Git revision. Use make to build to make
	// sure this gets set.
	branch string

	// BuildDate contains the date of the current build.
	buildDate string

	// Version contains version
	version string

	// Version contains repository name
	repo string
)

// BuildInfo contains information about the current build
type BuildInfo struct {
	Version   string `json:"version,omitempty"`
	Branch    string `json:"branch,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	Name      string `json:"name,omitempty"`
	Repo      string `json:"repo,omitempty"`
}

// GetBuildInfo returns build info data
func GetBuildInfo() *BuildInfo {
	return &BuildInfo{
		Version:   version,
		Branch:    branch,
		BuildDate: buildDate,
		Repo:      repo,
	}
}
