package rolling_deployments

import (
	"time"
	
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("mixed v2 and v3 rolling deploys", func() {
	var (
		appName string
	)

	It("can replace a zdt process with a v2 process", func() {
		appName = random_name.BARARandomName("APP")

		By("cf push my-app create the app in the first place")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf v3-zdt-push my-app the running app gets new code")
		zdtPushStaticApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))

		By("cf push my-app the running app gets new code again")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf v3-zdt-push my-app the running app gets new code a third time")
		zdtPushStaticApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
	})
	
	It("can immediately replace one process with another", func() {
		appName = random_name.BARARandomName("APP")

		By("cf push my-app create the app in the first place")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf push my-app the running app gets new code")
		pushStaticApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))

		By("cf push my-app the running app goes back to dora")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf push my-app the running app goes back to staticfile")
		pushStaticApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
	})
	
	FIt("can eventually replace one process with another", func() {
		appName = random_name.BARARandomName("APP")

		By("cf push my-app create the app in the first place")
		pushRubyApp(appName)
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		By("cf push my-app the running app gets new code")
		pushStaticApp(appName)
		Eventually(helpers.CurlAppRoot(Config, appName), 330 * time.Second).Should(Equal("Hello from a staticfile"))

		By("cf push my-app the running app goes back to dora")
		pushRubyApp(appName)
		Eventually(helpers.CurlAppRoot(Config, appName), 330 * time.Second).Should(Equal("Hi, I'm Dora!"))
/*

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Eventually(helpers.CurlAppRoot(Config, appName), 330 * time.Second).Should(Equal("Hi, I'm Dora!"))

		By("cf push my-app the running app goes back to staticfile")
		pushStaticApp(appName)
		Eventually(helpers.CurlAppRoot(Config, appName), 330 * time.Second).Should(Equal("Hello from a staticfile"))

		By("cf restart my-app the running app does not change")
		restartApp(appName)
		Eventually(helpers.CurlAppRoot(Config, appName), 330 * time.Second).Should(Equal("Hello from a staticfile"))
*/
	})

})

func restartApp(appName string) {
	Expect(cf.Cf("restart",
		appName,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}

func zdtPushStaticApp(appName string) {
	Expect(cf.Cf("v3-zdt-push",
		appName,
		"-b", "staticfile_buildpack",
		"-p", assets.NewAssets().Staticfile,
		"--wait-for-deploy-complete",
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}

func pushStaticApp(appName string) {
	Expect(cf.Cf("push",
		"-v",
		appName,
		"-b", "staticfile_buildpack",
		"-m", "128MB",
		"-i", "2",
		"-p", assets.NewAssets().Staticfile,
		"-d", Config.GetAppsDomain(),
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}

func pushRubyApp(appName string) {
	Expect(cf.Cf("push",
		"-v",
		appName,
		"-b", "ruby_buildpack",
		"-m", "128MB",
		"-i", "2",
		"-p", assets.NewAssets().Dora,
		"-d", Config.GetAppsDomain(),
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}
