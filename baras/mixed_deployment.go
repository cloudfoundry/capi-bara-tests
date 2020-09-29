package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("deploying a v3 app over a v2 app", func() {
	var (
		appName string
	)

	BeforeEach(func() {
		if Config.GetIncludeKpack() {
			Skip(skip_messages.SkipKpackMessage)
		}
	})

	AfterEach(func() {
		appGuid := GetAppGuid(appName)
		FetchRecentLogs(appGuid, Config)
		DeleteApp(appGuid)
	})

	It("it does not prevent pushing with v2 in the future", func() {
		appName = random_name.BARARandomName("APP")

		By("cf push my-app create the app in the first place")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf push --strategy rolling my-app the running app gets new code")
		rollingPushBinaryApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a binary"))

		By("cf push my-app the running app gets new code again")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf push --strategy rolling my-app the running app gets new code a third time")
		rollingPushBinaryApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a binary"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a binary"))
	})

})

func restartApp(appName string) {
	Expect(cf.Cf("restart",
		appName,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}

func rollingPushBinaryApp(appName string) {
	Expect(cf.Cf("push",
		appName,
		"-b", Config.GetBinaryBuildpackName(),
		"-p", assets.NewAssets().Binary,
		"-m", "128MB",
		"--strategy", "rolling",
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}

func pushRubyApp(appName string) {
	Expect(cf.Cf("push",
		appName,
		"-b", Config.GetRubyBuildpackName(),
		"-m", "128MB",
		"-i", "2",
		"-p", assets.NewAssets().Dora,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}
