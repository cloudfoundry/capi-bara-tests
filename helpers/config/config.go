package config

import (
	"time"
)

type BaraConfig interface {
	GetApiEndpoint() string
	GetAppsDomain() string

	Protocol() string

	GetAdminPassword() string
	GetAdminUser() string

	GetSkipSSLValidation() bool

	GetArtifactsDirectory() string

	AsyncServiceOperationTimeoutDuration() time.Duration
	BrokerStartTimeoutDuration() time.Duration
	CfPushTimeoutDuration() time.Duration
	DefaultTimeoutDuration() time.Duration
	DetectTimeoutDuration() time.Duration
	LongCurlTimeoutDuration() time.Duration
	SleepTimeoutDuration() time.Duration
	CcClockCycleDuration() time.Duration

	GetScaledTimeout(time.Duration) time.Duration

	GetBinaryBuildpackName() string
	GetStaticFileBuildpackName() string
	GetGoBuildpackName() string
	GetHwcBuildpackName() string
	GetJavaBuildpackName() string
	GetNodejsBuildpackName() string
	GetRubyBuildpackName() string
	GetPythonBuildpackName() string

	GetNamePrefix() string

	GetReporterConfig() reporterConfig

	Lifecycle() string
	GetGcloudProjectName() string
	GetClusterZone() string
	GetClusterName() string
	RunningOnK8s() bool

	// Used only by TestConfig?
	GetConfigurableTestPassword() string
	GetExistingOrganization() string
	GetExistingSpace() string
	GetExistingUser() string
	GetExistingUserPassword() string
	GetShouldKeepUser() bool
	GetUseExistingUser() bool
	GetUseExistingOrganization() bool
	GetUseExistingSpace() bool

	// added from bumping cf-test-helper to v2
	GetAddExistingUserToExistingSpace() bool
	GetAdminClient() string
	GetAdminClientSecret() string
	GetExistingClient() string
	GetExistingClientSecret() string
}

func NewBaraConfig(path string) (BaraConfig, error) {
	return NewConfig(path)
}
