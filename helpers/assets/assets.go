package assets

// Assets lists the fixture-app paths referenced by the BARA test suite.
// Only entries that are actually pushed, staged, or otherwise loaded by
// tests belong here. When adding a new fixture, register it here AND
// add a comment pointing to the first test that uses it.
type Assets struct {
	// Catnip is a Go app whose binary is rebuilt in SynchronizedBeforeSuite
	// (bara_suite_test.go). Both the source dir and the zip are needed:
	// the dir is compiled, the zip is pushed to CF.
	Catnip    string
	CatnipZip string

	// Dora is a Ruby/Sinatra fixture used by manifest, deployments,
	// revisions, sidecars, and commands tests. DoraDroplet is a
	// pre-staged tarball used only by droplets.go.
	Dora        string
	DoraZip     string
	DoraDroplet string

	// BadDora is a Dora variant that fails to boot (used to exercise
	// failed deployment paths in deployments.go).
	BadDora    string
	BadDoraZip string

	// MultiPortApp exposes two ports; used by zero_downtime tests.
	MultiPortApp string

	// PythonWithoutProcfile is used by commands.go to exercise
	// buildpack-detected start commands.
	PythonWithoutProcfile    string
	PythonWithoutProcfileZip string

	// ServiceBroker is pushed by manifest tests that need a broker.
	ServiceBroker string

	// Sidecar (source) is compiled into a binary that ships inside the
	// SidecarDependent fixture; see bara_suite_test.go.
	SidecarDependent string

	// SleepySidecarBuildpack is a zipped buildpack used by sidecars.go.
	SleepySidecarBuildpack    string
	SleepySidecarBuildpackZip string

	// Staticfile is used as a second droplet in canary deployment tests.
	Staticfile    string
	StaticfileZip string
}

func NewAssets() Assets {
	return Assets{
		Catnip:                    "assets/catnip",
		CatnipZip:                 "assets/catnip.zip",
		Dora:                      "assets/dora",
		DoraZip:                   "assets/dora.zip",
		DoraDroplet:               "assets/dora-droplet.tgz",
		BadDora:                   "assets/bad-dora",
		BadDoraZip:                "assets/bad-dora.zip",
		MultiPortApp:              "assets/multi-port-app",
		PythonWithoutProcfile:     "assets/python-without-procfile",
		PythonWithoutProcfileZip:  "assets/python-without-procfile.zip",
		ServiceBroker:             "assets/service_broker",
		SidecarDependent:          "assets/sidecar-dependent",
		SleepySidecarBuildpack:    "assets/sleepy-sidecar-buildpack",
		SleepySidecarBuildpackZip: "assets/sleepy-sidecar-buildpack.zip",
		Staticfile:                "assets/staticfile",
		StaticfileZip:             "assets/staticfile.zip",
	}
}
