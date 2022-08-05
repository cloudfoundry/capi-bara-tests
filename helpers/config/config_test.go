package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/cloudfoundry/capi-bara-tests/helpers/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type requiredConfig struct {
	// required
	ApiEndpoint       *string `json:"api"`
	AdminUser         *string `json:"admin_user"`
	AdminPassword     *string `json:"admin_password"`
	SkipSSLValidation *bool   `json:"skip_ssl_validation"`
	AppsDomain        *string `json:"apps_domain"`
}

type testConfig struct {
	// required
	ApiEndpoint       *string `json:"api"`
	AdminUser         *string `json:"admin_user"`
	AdminPassword     *string `json:"admin_password"`
	SkipSSLValidation *bool   `json:"skip_ssl_validation"`
	AppsDomain        *string `json:"apps_domain"`

	// timeouts
	DefaultTimeout               *int `json:"default_timeout,omitempty"`
	CfPushTimeout                *int `json:"cf_push_timeout,omitempty"`
	LongCurlTimeout              *int `json:"long_curl_timeout,omitempty"`
	BrokerStartTimeout           *int `json:"broker_start_timeout,omitempty"`
	AsyncServiceOperationTimeout *int `json:"async_service_operation_timeout,omitempty"`
	DetectTimeout                *int `json:"detect_timeout,omitempty"`
	SleepTimeout                 *int `json:"sleep_timeout,omitempty"`
	CcClockCycle                 *int `json:"cc_clock_cycle,omitempty"`

	TimeoutScale *float64 `json:"timeout_scale,omitempty"`

	// optional
	UnallocatedIPForSecurityGroup *string `json:"unallocated_ip_for_security_group"`
	RequireProxiedAppTraffic      *bool   `json:"require_proxied_app_traffic"`

	ReporterConfig *testReporterConfig `json:"reporter_config"`
}

type allConfig struct {
	ApiEndpoint *string `json:"api"`
	AppsDomain  *string `json:"apps_domain"`

	AdminPassword *string `json:"admin_password"`
	AdminUser     *string `json:"admin_user"`

	ExistingUser         *string `json:"existing_user"`
	ExistingUserPassword *string `json:"existing_user_password"`

	UseExistingOrganization *bool   `json:"use_existing_organization"`
	ExistingOrganization    *string `json:"existing_organization"`

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

	ReporterConfig *testReporterConfig `json:"reporter_config"`

	NamePrefix *string `json:"name_prefix"`
}

type testReporterConfig struct {
	HoneyCombWriteKey string                 `json:"honeycomb_write_key"`
	HoneyCombDataset  string                 `json:"honeycomb_dataset"`
	CustomTags        map[string]interface{} `json:"custom_tags"`
}

var tmpFilePath string
var testCfg testConfig

func writeConfigFile(updatedConfig interface{}) string {
	configFile, err := ioutil.TempFile("", "cf-test-helpers-config")
	Expect(err).NotTo(HaveOccurred())

	encoder := json.NewEncoder(configFile)
	err = encoder.Encode(updatedConfig)

	Expect(err).NotTo(HaveOccurred())

	err = configFile.Close()
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

func ptrToString(str string) *string {
	return &str
}

func ptrToBool(b bool) *bool {
	return &b
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToFloat(f float64) *float64 {
	return &f
}

var _ = Describe("Config", func() {
	BeforeEach(func() {
		testCfg = testConfig{}
		testCfg.ApiEndpoint = ptrToString("api.bosh-lite.com")
		testCfg.AdminUser = ptrToString("admin")
		testCfg.AdminPassword = ptrToString("admin")
		testCfg.SkipSSLValidation = ptrToBool(true)
		testCfg.AppsDomain = ptrToString("cf-app.bosh-lite.com")
	})

	JustBeforeEach(func() {
		tmpFilePath = writeConfigFile(&testCfg)
	})

	AfterEach(func() {
		err := os.Remove(tmpFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should have the right defaults", func() {
		requiredCfg := requiredConfig{}
		requiredCfg.ApiEndpoint = testCfg.ApiEndpoint
		requiredCfg.AdminUser = testCfg.AdminUser
		requiredCfg.AdminPassword = testCfg.AdminPassword
		requiredCfg.SkipSSLValidation = testCfg.SkipSSLValidation
		requiredCfg.AppsDomain = testCfg.AppsDomain

		requiredCfgFilePath := writeConfigFile(requiredCfg)
		config, err := cfg.NewBaraConfig(requiredCfgFilePath)
		Expect(err).ToNot(HaveOccurred())

		testReporterConfig := config.GetReporterConfig()
		Expect(testReporterConfig.HoneyCombDataset).To(Equal(""))
		Expect(testReporterConfig.HoneyCombWriteKey).To(Equal(""))

		Expect(config.GetUseExistingUser()).To(Equal(false))
		Expect(config.GetConfigurableTestPassword()).To(Equal(""))
		Expect(config.GetShouldKeepUser()).To(Equal(false))

		Expect(config.GetExistingOrganization()).To(Equal(""))
		Expect(config.GetUseExistingOrganization()).To(Equal(false))

		Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(4 * time.Minute))
		Expect(config.BrokerStartTimeoutDuration()).To(Equal(10 * time.Minute))
		Expect(config.CfPushTimeoutDuration()).To(Equal(4 * time.Minute))
		Expect(config.DefaultTimeoutDuration()).To(Equal(60 * time.Second))
		Expect(config.LongCurlTimeoutDuration()).To(Equal(4 * time.Minute))

		Expect(config.GetScaledTimeout(1)).To(Equal(time.Duration(2)))

		Expect(config.GetArtifactsDirectory()).To(Equal(filepath.Join("..", "results")))

		Expect(config.GetNamePrefix()).To(Equal("BARA"))

		Expect(config.Protocol()).To(Equal("https://"))

		// undocumented
		Expect(config.DetectTimeoutDuration()).To(Equal(10 * time.Minute))
		Expect(config.SleepTimeoutDuration()).To(Equal(60 * time.Second))
	})

	Context("when all values are null", func() {
		It("returns an error", func() {
			allCfgFilePath := writeConfigFile(&allConfig{})
			_, err := cfg.NewBaraConfig(allCfgFilePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'api' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'apps_domain' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'admin_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'admin_user' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'skip_ssl_validation' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'artifacts_directory' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'async_service_operation_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'broker_start_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'cf_push_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'default_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'detect_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'long_curl_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'sleep_timeout' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'timeout_scale' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'binary_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'go_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'java_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'nodejs_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'php_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'python_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'ruby_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'staticfile_buildpack_name' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'name_prefix' must not be null"))
		})
	})

	Context("when values with default are overriden", func() {
		BeforeEach(func() {
			testCfg.DefaultTimeout = ptrToInt(12)
			testCfg.CfPushTimeout = ptrToInt(34)
			testCfg.LongCurlTimeout = ptrToInt(56)
			testCfg.BrokerStartTimeout = ptrToInt(78)
			testCfg.AsyncServiceOperationTimeout = ptrToInt(90)
			testCfg.DetectTimeout = ptrToInt(100)
			testCfg.SleepTimeout = ptrToInt(101)
			testCfg.CcClockCycle = ptrToInt(65)
			testCfg.TimeoutScale = ptrToFloat(1.0)
			testCfg.UnallocatedIPForSecurityGroup = ptrToString("192.168.0.1")
			testCfg.RequireProxiedAppTraffic = ptrToBool(true)
		})

		It("respects the overriden values", func() {
			config, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(config.DefaultTimeoutDuration()).To(Equal(12 * time.Second))
			Expect(config.CfPushTimeoutDuration()).To(Equal(34 * time.Second))
			Expect(config.LongCurlTimeoutDuration()).To(Equal(56 * time.Second))
			Expect(config.BrokerStartTimeoutDuration()).To(Equal(78 * time.Second))
			Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(90 * time.Second))
			Expect(config.DetectTimeoutDuration()).To(Equal(100 * time.Second))
			Expect(config.SleepTimeoutDuration()).To(Equal(101 * time.Second))
			Expect(config.CcClockCycleDuration()).To(Equal(65 * time.Second))
		})
	})

	Context("when including a reporter config", func() {
		BeforeEach(func() {
			reporterConfig := &testReporterConfig{
				HoneyCombWriteKey: "some-write-key",
				HoneyCombDataset:  "some-dataset",
			}
			testCfg.ReporterConfig = reporterConfig
		})

		It("is loaded into the config", func() {
			config, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())

			testReporterConfig := config.GetReporterConfig()
			Expect(testReporterConfig.HoneyCombWriteKey).To(Equal("some-write-key"))
			Expect(testReporterConfig.HoneyCombDataset).To(Equal("some-dataset"))
		})
		Context("when the reporter config includes custom tags", func() {
			BeforeEach(func() {
				customTags := map[string]interface{}{
					"some-tag": "some-tag-value",
				}
				testCfg.ReporterConfig.CustomTags = customTags
			})
			It("is loaded into the config", func() {
				config, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).ToNot(HaveOccurred())

				testReporterConfig := config.GetReporterConfig()
				Expect(testReporterConfig.CustomTags).To(Equal(map[string]interface{}{
					"some-tag": "some-tag-value",
				}))
			})
		})
	})

	Describe("error aggregation", func() {
		BeforeEach(func() {
			testCfg.AdminPassword = nil
			testCfg.ApiEndpoint = ptrToString("invalid-url.asdf")
		})

		It("aggregates all errors", func() {
			config, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(config).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("* 'admin_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("* Invalid configuration for 'api' <invalid-url.asdf>"))
		})
	})

	Describe("GetApiEndpoint", func() {
		It(`returns the URL`, func() {
			cfg, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetApiEndpoint()).To(Equal("api.bosh-lite.com"))
		})

		Context("when url is an IP address", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("10.244.0.34") // api.bosh-lite.com
			})

			It("returns the IP address", func() {
				cfg, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetApiEndpoint()).To(Equal("10.244.0.34"))
			})
		})

		Context("when the domain does not resolve", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("some-url-that-does-not-resolve.com.some-url-that-does-not-resolve.com")
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
			})
		})

		Context("when the url is empty", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank"))
			})
		})

		Context("when the url is invalid", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("_bogus%%%")
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'api' must be a valid URL but was set to '_bogus%%%'"))
			})
		})

		Context("when the ApiEndpoint is nil", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'api' must not be null"))
			})
		})
	})

	Describe("GetAppsDomain", func() {
		It("returns the domain", func() {
			c, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.GetAppsDomain()).To(Equal("cf-app.bosh-lite.com"))
		})

		Context("when the domain is not valid", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = ptrToString("_bogus%%%")
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'apps_domain' must be a valid URL but was set to '_bogus%%%'"))
			})
		})

		Context("when the AppsDomain is an IP address (which is invalid for AppsDomain)", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = ptrToString("10.244.0.34")
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
			})
		})

		Context("when the AppsDomain is nil", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'apps_domain' must not be null"))
			})
		})
	})

	Describe("GetAdminUser", func() {
		It("returns the admin user", func() {
			c, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.GetAdminUser()).To(Equal("admin"))
		})

		Context("when the admin user is blank", func() {
			BeforeEach(func() {
				*testCfg.AdminUser = ""
			})
			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_user' must be provided"))
			})
		})

		Context("when the admin user is nil", func() {
			BeforeEach(func() {
				testCfg.AdminUser = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_user' must not be null"))
			})
		})
	})

	Describe("GetAdminPassword", func() {
		It("returns the admin password", func() {
			c, err := cfg.NewBaraConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.GetAdminPassword()).To(Equal("admin"))
		})

		Context("when the admin user password is blank", func() {
			BeforeEach(func() {
				testCfg.AdminPassword = ptrToString("")
			})
			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_password' must be provided"))
			})
		})

		Context("when the admin user password is nil", func() {
			BeforeEach(func() {
				testCfg.AdminPassword = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewBaraConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_password' must not be null"))
			})
		})
	})
})
