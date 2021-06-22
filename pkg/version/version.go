package version

// Version is updated automatically as part of the build process, and is the
// ground source of truth for the current process's build version.
//
// DO NOT EDIT
var Version = undefinedVersion

const (
	// undefinedVersion should take the form `channel-version` to conform to
	// channelVersion functions.
	undefinedVersion = "dev-undefined"

	VersionPlaceHolder = "linkerdSMIVersionValue"
)
