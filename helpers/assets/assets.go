package assets

type Assets struct {
	AspClassic                 string
	BatchScript                string
	Binary                     string
	Catnip                     string
	CredHubEnabledApp          string
	CredHubServiceBroker       string
	BadDora                    string
	Dora                       string
	DoraDroplet                string
	DoraZip                    string
	BadDoraZip                 string
	DotnetCore                 string
	Fuse                       string
	GoCallsRubyZip             string
	Golang                     string
	HelloRouting               string
	HelloWorld                 string
	Java                       string
	JavaSpringZip              string
	JavaUnwriteableZip         string
	LoggingRouteService        string
	LoggregatorLoadGenerator   string
	LoggregatorLoadGeneratorGo string
	MultiPortApp               string
	Node                       string
	NodeWithProcfile           string
	Nora                       string
	Php                        string
	Proxy                      string
	Python                     string
	RubySimple                 string
	SecurityGroupBuildpack     string
	ServiceBroker              string
	SleepySidecarBuildpack     string
	SidecarDependent           string
	SpringSleuthZip            string
	Staticfile                 string
	StaticfileZip              string
	SyslogDrainListener        string
	TCPListener                string
	Wcf                        string
	WindowsWebapp              string
	WindowsWorker              string
	WorkerApp                  string
}

func NewAssets() Assets {
	return Assets{
		AspClassic:                 "assets/asp-classic",
		BatchScript:                "assets/batch-script",
		Binary:                     "assets/binary",
		Catnip:                     "assets/catnip/bin",
		CredHubEnabledApp:          "assets/credhub-enabled-app/credhub-enabled-app.jar",
		CredHubServiceBroker:       "assets/credhub-service-broker",
		BadDora:                    "assets/bad-dora",
		Dora:                       "assets/dora",
		DoraDroplet:                "assets/dora-droplet.tgz",
		DoraZip:                    "assets/dora.zip",
		BadDoraZip:                 "assets/bad-dora.zip",
		DotnetCore:                 "assets/dotnet-core",
		Fuse:                       "assets/fuse-mount",
		GoCallsRubyZip:             "assets/go_calls_ruby.zip",
		Golang:                     "assets/golang",
		HelloRouting:               "assets/hello-routing",
		HelloWorld:                 "assets/hello-world",
		Java:                       "assets/java",
		JavaSpringZip:              "assets/java-spring/java-spring.jar",
		JavaUnwriteableZip:         "assets/java-unwriteable-dir/java-unwriteable-dir.jar",
		LoggingRouteService:        "assets/logging-route-service",
		LoggregatorLoadGenerator:   "assets/loggregator-load-generator",
		LoggregatorLoadGeneratorGo: "assets/loggregator-load-generator-go",
		MultiPortApp:               "assets/multi-port-app",
		Node:                       "assets/node",
		NodeWithProcfile:           "assets/node-with-procfile",
		Nora:                       "assets/nora/NoraPublished",
		Php:                        "assets/php",
		Proxy:                      "vendor/code.cloudfoundry.org/cf-networking-release/src/example-apps/proxy",
		Python:                     "assets/python",
		RubySimple:                 "assets/ruby_simple",
		SecurityGroupBuildpack:     "assets/security_group_buildpack.zip",
		ServiceBroker:              "assets/service_broker",
		SidecarDependent:           "assets/sidecar-dependent",
		SleepySidecarBuildpack:     "assets/sleepy-sidecar_buildpack-cflinuxfs3-v0.1.zip",
		SpringSleuthZip:            "assets/spring-sleuth/spring-sleuth.jar",
		Staticfile:                 "assets/staticfile",
		StaticfileZip:              "assets/staticfile.zip",
		SyslogDrainListener:        "assets/syslog-drain-listener",
		TCPListener:                "assets/tcp-listener",
		Wcf:                        "assets/wcf/Hello.Service.IIS",
		WindowsWebapp:              "assets/webapp",
		WindowsWorker:              "assets/worker",
		WorkerApp:                  "assets/worker-app",
	}
}
