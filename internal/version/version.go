package version

var (
	version = "unset"
)

// GetVersion return current version of the API.
func GetVersion() string {
	return version
}
