package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/logs"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/onsi/gomega/gbytes"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FDescribe("events", func() {
	var (
		appName        string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
	})

	AfterEach(func() {
		DeleteApp(GetAppGuid(appName))
	})


	Describe("Push an app", func() {
		It("streams logs for an app start event", func() {
			session := cf.Cf("push", appName, "-p", assets.NewAssets().Catnip)
			Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			session = logs.Tail(true, appName)
			Eventually(session).Should(Exit(0))
			Eventually(session).Should(gbytes.Say("Restarted app with guid"))
		})
	})
})
