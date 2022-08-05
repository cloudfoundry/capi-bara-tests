package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/validationerrors"
)

type config struct {
	ApiEndpoint *string `json:"api"`
	AppsDomain  *string `json:"apps_domain"`

	AdminPassword *string `json:"admin_password"`
	AdminUser     *string `json:"admin_user"`

	SkipSSLValidation *bool `json:"skip_ssl_validation"`

	ArtifactsDirectory *string `json:"artifacts_directory"`

	AsyncServiceOperationTimeout *int `json:"async_service_operation_timeout"`
	BrokerStartTimeout           *int `json:"broker_start_timeout"`
	CfPushTimeout                *int `json:"cf_push_timeout"`
	DefaultTimeout               *int `json:"default_timeout"`
	DetectTimeout                *int `json:"detect_timeout"`
	LongCurlTimeout              *int `json:"long_curl_timeout"`
	SleepTimeout                 *int `json:"sleep_timeout"`
	CcClockCycle                 *int `json:"cc_clock_cycle"`

	TimeoutScale *float64 `json:"timeout_scale"`

	BinaryBuildpackName     *string `json:"binary_buildpack_name"`
	GoBuildpackName         *string `json:"go_buildpack_name"`
	HwcBuildpackName        *string `json:"hwc_buildpack_name"`
	JavaBuildpackName       *string `json:"java_buildpack_name"`
	NodejsBuildpackName     *string `json:"nodejs_buildpack_name"`
	PhpBuildpackName        *string `json:"php_buildpack_name"`
	PythonBuildpackName     *string `json:"python_buildpack_name"`
	RubyBuildpackName       *string `json:"ruby_buildpack_name"`
	StaticFileBuildpackName *string `json:"staticfile_buildpack_name"`

	Infrastructure *string `json:"infrastructure"`

	GcloudProjectName *string `json:"gcloud_project_name""`
	ClusterZone       *string `json:"cluster_zone"`
	ClusterName       *string `json:"cluster_name"`

	NamePrefix *string `json:"name_prefix"`

	ReporterConfig *reporterConfig `json:"reporter_config"`
}

type reporterConfig struct {
	CustomTags        map[string]interface{} `json:"custom_tags"`
	HoneyCombWriteKey string                 `json:"honeycomb_write_key"`
	HoneyCombDataset  string                 `json:"honeycomb_dataset"`
}

var defaults = config{}

func ptrToString(str string) *string {
	return &str
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToFloat(f float64) *float64 {
	return &f
}

func getDefaults() config {
	defaults.BinaryBuildpackName = ptrToString("binary_buildpack")
	defaults.GoBuildpackName = ptrToString("go_buildpack")
	defaults.HwcBuildpackName = ptrToString("hwc_buildpack")
	defaults.JavaBuildpackName = ptrToString("java_buildpack")
	defaults.NodejsBuildpackName = ptrToString("nodejs_buildpack")
	defaults.PhpBuildpackName = ptrToString("php_buildpack")
	defaults.PythonBuildpackName = ptrToString("python_buildpack")
	defaults.RubyBuildpackName = ptrToString("ruby_buildpack")
	defaults.StaticFileBuildpackName = ptrToString("staticfile_buildpack")

	defaults.ReporterConfig = &reporterConfig{}

	defaults.AsyncServiceOperationTimeout = ptrToInt(120)
	defaults.BrokerStartTimeout = ptrToInt(300)
	defaults.CfPushTimeout = ptrToInt(120)
	defaults.DefaultTimeout = ptrToInt(30)
	defaults.DetectTimeout = ptrToInt(300)
	defaults.LongCurlTimeout = ptrToInt(120)
	defaults.SleepTimeout = ptrToInt(30)
	defaults.CcClockCycle = ptrToInt(30)

	defaults.TimeoutScale = ptrToFloat(2.0)

	defaults.Infrastructure = ptrToString("vms")

	defaults.GcloudProjectName = ptrToString("")
	defaults.ClusterZone = ptrToString("")
	defaults.ClusterName = ptrToString("")

	defaults.ArtifactsDirectory = ptrToString(filepath.Join("..", "results"))

	defaults.NamePrefix = ptrToString("BARA")

	return defaults
}

func NewConfig(path string) (*config, error) {
	d := getDefaults()
	cfg := &d
	err := load(path, cfg)
	if err.Empty() {
		return cfg, nil
	}
	return nil, err
}

func validateConfig(config *config) Errors {
	errs := Errors{}

	var err error
	err = validateAdminUser(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateAdminPassword(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateApiEndpoint(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateAppsDomain(config)
	if err != nil {
		errs.Add(err)
	}

	if config.SkipSSLValidation == nil {
		errs.Add(fmt.Errorf("* 'skip_ssl_validation' must not be null"))
	}
	if config.ArtifactsDirectory == nil {
		errs.Add(fmt.Errorf("* 'artifacts_directory' must not be null"))
	}
	if config.AsyncServiceOperationTimeout == nil {
		errs.Add(fmt.Errorf("* 'async_service_operation_timeout' must not be null"))
	}
	if config.BrokerStartTimeout == nil {
		errs.Add(fmt.Errorf("* 'broker_start_timeout' must not be null"))
	}
	if config.CfPushTimeout == nil {
		errs.Add(fmt.Errorf("* 'cf_push_timeout' must not be null"))
	}
	if config.DefaultTimeout == nil {
		errs.Add(fmt.Errorf("* 'default_timeout' must not be null"))
	}
	if config.DetectTimeout == nil {
		errs.Add(fmt.Errorf("* 'detect_timeout' must not be null"))
	}
	if config.LongCurlTimeout == nil {
		errs.Add(fmt.Errorf("* 'long_curl_timeout' must not be null"))
	}
	if config.SleepTimeout == nil {
		errs.Add(fmt.Errorf("* 'sleep_timeout' must not be null"))
	}
	if config.CcClockCycle == nil {
		errs.Add(fmt.Errorf("* 'cc_clock_cycle' must not be null"))
	}
	if config.TimeoutScale == nil {
		errs.Add(fmt.Errorf("* 'timeout_scale' must not be null"))
	}
	if config.BinaryBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'binary_buildpack_name' must not be null"))
	}
	if config.GoBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'go_buildpack_name' must not be null"))
	}
	if config.HwcBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'hwc_buildpack_name' must not be null"))
	}
	if config.JavaBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'java_buildpack_name' must not be null"))
	}
	if config.NodejsBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'nodejs_buildpack_name' must not be null"))
	}
	if config.PhpBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'php_buildpack_name' must not be null"))
	}
	if config.PythonBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'python_buildpack_name' must not be null"))
	}
	if config.RubyBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'ruby_buildpack_name' must not be null"))
	}
	if config.StaticFileBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'staticfile_buildpack_name' must not be null"))
	}
	if config.NamePrefix == nil {
		errs.Add(fmt.Errorf("* 'name_prefix' must not be null"))
	}

	return errs
}

func validateApiEndpoint(config *config) error {
	if config.ApiEndpoint == nil {
		return fmt.Errorf("* 'api' must not be null")
	}

	if config.GetApiEndpoint() == "" {
		return fmt.Errorf("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank")
	}

	u, err := url.Parse(config.GetApiEndpoint())
	if err != nil {
		return fmt.Errorf("* Invalid configuration: 'api' must be a valid URL but was set to '%s'", config.GetApiEndpoint())
	}

	host := u.Host
	if host == "" {
		// url.Parse misunderstood our convention and treated the hostname as a URL path
		host = u.Path
	}

	if _, err = net.LookupHost(host); err != nil {
		return fmt.Errorf("* Invalid configuration for 'api' <%s>: %s", config.GetApiEndpoint(), err)
	}

	return nil
}

func validateAppsDomain(config *config) error {
	if config.AppsDomain == nil {
		return fmt.Errorf("* 'apps_domain' must not be null")
	}

	madeUpAppHostname := "made-up-app-host-name." + config.GetAppsDomain()
	u, err := url.Parse(madeUpAppHostname)
	if err != nil {
		return fmt.Errorf("* Invalid configuration: 'apps_domain' must be a valid URL but was set to '%s'", config.GetAppsDomain())
	}

	host := u.Host
	if host == "" {
		// url.Parse misunderstood our convention and treated the hostname as a URL path
		host = u.Path
	}

	if _, err = net.LookupHost(madeUpAppHostname); err != nil {
		return fmt.Errorf("* Invalid configuration for 'apps_domain' <%s>: %s", config.GetAppsDomain(), err)
	}

	return nil
}

func validateAdminUser(config *config) error {
	if config.AdminUser == nil {
		return fmt.Errorf("* 'admin_user' must not be null")
	}

	if config.GetAdminUser() == "" {
		return fmt.Errorf("* Invalid configuration: 'admin_user' must be provided")
	}

	return nil
}

func validateAdminPassword(config *config) error {
	if config.AdminPassword == nil {
		return fmt.Errorf("* 'admin_password' must not be null")
	}

	if config.GetAdminPassword() == "" {
		return fmt.Errorf("* Invalid configuration: 'admin_password' must be provided")
	}

	return nil
}

func load(path string, config *config) Errors {
	errs := Errors{}
	err := loadConfigFromPath(path, config)
	if err != nil {
		errs.Add(fmt.Errorf("* Failed to unmarshal: %s", err))
		return errs
	}

	errs = validateConfig(config)
	if !errs.Empty() {
		return errs
	}

	if *config.TimeoutScale <= 0 {
		*config.TimeoutScale = 1.0
	}

	return errs
}

func loadConfigFromPath(path string, config interface{}) error {
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	return decoder.Decode(config)
}

func (c config) GetScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * *c.TimeoutScale)
}

func (c *config) DefaultTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.DefaultTimeout) * time.Second)
}

func (c *config) LongCurlTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.LongCurlTimeout) * time.Second)
}

func (c *config) SleepTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.SleepTimeout) * time.Second)
}

func (c *config) CcClockCycleDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.CcClockCycle) * time.Second)
}

func (c *config) DetectTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.DetectTimeout) * time.Second)
}

func (c *config) CfPushTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.CfPushTimeout) * time.Second)
}

func (c *config) BrokerStartTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.BrokerStartTimeout) * time.Second)
}

func (c *config) AsyncServiceOperationTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.AsyncServiceOperationTimeout) * time.Second)
}

func (c *config) Protocol() string {
	return "https://"
}

func (c *config) GetAppsDomain() string {
	return *c.AppsDomain
}

func (c *config) GetSkipSSLValidation() bool {
	return *c.SkipSSLValidation
}

func (c *config) GetArtifactsDirectory() string {
	return *c.ArtifactsDirectory
}

func (c *config) GetNamePrefix() string {
	return *c.NamePrefix
}

func (c *config) GetAdminUser() string {
	return *c.AdminUser
}

func (c *config) GetAdminPassword() string {
	return *c.AdminPassword
}

func (c *config) GetApiEndpoint() string {
	return *c.ApiEndpoint
}

func (c *config) GetPythonBuildpackName() string {
	return *c.PythonBuildpackName
}

func (c *config) GetRubyBuildpackName() string {
	return *c.RubyBuildpackName
}

func (c *config) GetGoBuildpackName() string {
	return *c.GoBuildpackName
}

func (c *config) GetHwcBuildpackName() string {
	return *c.HwcBuildpackName
}

func (c *config) GetJavaBuildpackName() string {
	return *c.JavaBuildpackName
}

func (c *config) GetNodejsBuildpackName() string {
	return *c.NodejsBuildpackName
}

func (c *config) GetBinaryBuildpackName() string {
	return *c.BinaryBuildpackName
}

func (c *config) GetStaticFileBuildpackName() string {
	return *c.StaticFileBuildpackName
}

func (c *config) Lifecycle() string {
	if c.RunningOnK8s() {
		return "kpack"
	} else {
		return "buildpack"
	}
}

func (c *config) GetGcloudProjectName() string {
	return *c.GcloudProjectName
}

func (c *config) GetClusterZone() string {
	return *c.ClusterZone
}

func (c *config) GetClusterName() string {
	return *c.ClusterName
}

func (c *config) GetReporterConfig() reporterConfig {
	reporterConfigFromConfig := c.ReporterConfig

	if reporterConfigFromConfig != nil {
		return *reporterConfigFromConfig
	}

	return reporterConfig{}
}

func (c *config) RunningOnK8s() bool {
	return *c.Infrastructure == "kubernetes"
}

// Used only by TestConfig?
func (c *config) GetConfigurableTestPassword() string { return "" }
func (c *config) GetExistingOrganization() string     { return "" }
func (c *config) GetExistingSpace() string            { return "" }
func (c *config) GetExistingUser() string             { return "" }
func (c *config) GetExistingUserPassword() string     { return "" }
func (c *config) GetShouldKeepUser() bool             { return false }
func (c *config) GetUseExistingUser() bool            { return false }
func (c *config) GetUseExistingOrganization() bool    { return false }
func (c *config) GetUseExistingSpace() bool           { return false }

func (c *config) GetAddExistingUserToExistingSpace() bool {
	//TODO implement me
	panic("implement me")
}

func (c *config) GetAdminClient() string {
	//TODO implement me
	panic("implement me")
}

func (c *config) GetAdminClientSecret() string {
	//TODO implement me
	panic("implement me")
}

func (c *config) GetExistingClient() string {
	//TODO implement me
	panic("implement me")
}

func (c *config) GetExistingClientSecret() string {
	//TODO implement me
	panic("implement me")
}
