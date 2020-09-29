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

	GetScaledTimeout(time.Duration) time.Duration

	GetBinaryBuildpackName() string
	GetStaticFileBuildpackName() string
	GetGoBuildpackName() string
	GetHwcBuildpackName() string
	GetJavaBuildpackName() string
	GetNodejsBuildpackName() string
	GetRubyBuildpackName() string

	GetNamePrefix() string

	GetReporterConfig() reporterConfig

	GetIncludeKpack() bool
	Lifecycle() string
	GetGcloudProjectName() string
	GetClusterZone() string
	GetClusterName() string
	// Unused?
	GetConfigurableTestPassword() string
	GetExistingOrganization() string
	GetExistingSpace() string
	GetExistingUser() string
	GetExistingUserPassword() string
	GetShouldKeepUser() bool
	GetUseExistingUser() bool
	GetUseExistingOrganization() bool
	GetUseExistingSpace() bool

	GetIncludePrivateDockerRegistry() bool
	GetPrivateDockerRegistryImage() string
	GetPrivateDockerRegistryUsername() string
	GetPrivateDockerRegistryPassword() string
}

func NewBaraConfig(path string) (BaraConfig, error) {
	return NewConfig(path)
}
