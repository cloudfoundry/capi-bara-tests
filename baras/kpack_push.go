package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("kpack push", func() {
	var (
		appName string
	)

	BeforeEach(func() {
		if !Config.GetIncludeKpack() {
			Skip(skip_messages.SkipKpackMessage)
		}

		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))

		appName = random_name.BARARandomName("APP")
	})

	It("can detect which buildpack an app needs", func() {
		session := cf.Cf("push", appName, "-p", assets.NewAssets().CatnipZip)
		Eventually(session, Config.CfPushTimeoutDuration()).Should(gexec.Exit(0))
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
	})

	It("allows the user to select a buildpack", func() {
		session := cf.Cf("push", appName, "-p", assets.NewAssets().CatnipZip, "-b", "paketo-buildpacks/go")
		Eventually(session, Config.CfPushTimeoutDuration()).Should(gexec.Exit(0))
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
	})

	It("fails loudly when an incorrect buildpack is selected", func() {
		session := cf.Cf("push", appName, "-p", assets.NewAssets().CatnipZip, "-b", "paketo-buildpacks/dotnet-core")
		Eventually(session, Config.CfPushTimeoutDuration()).Should(gexec.Exit(1))
		Eventually(func() *gexec.Session {
			session := cf.Cf("logs", appName, "--recent")
			Eventually(session).Should(gexec.Exit(0))
			return session
		}).Should(gbytes.Say("No buildpack groups passed detection"))
	})

	It("accepts changes to buildpacks after an incorrect buildpack was selected", func() {
		session := cf.Cf("push", appName, "-p", assets.NewAssets().CatnipZip, "-b", "paketo-buildpacks/dotnet-core")
		Eventually(session, Config.CfPushTimeoutDuration()).Should(gexec.Exit(1))

		session = cf.Cf("push", appName, "-p", assets.NewAssets().CatnipZip, "-b", "paketo-buildpacks/go")
		Eventually(session, Config.CfPushTimeoutDuration()).Should(gexec.Exit(0))
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
	})
})
