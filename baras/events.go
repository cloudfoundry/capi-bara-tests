package baras

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/logs"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("events", func() {
	var (
		appName        string
		appGuid string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		session := cf.Cf("push", appName, "-p", assets.NewAssets().Catnip)
		Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		appGuid = GetAppGuid(appName)
	})

	AfterEach(func() {
		DeleteApp(appGuid)
	})


	Describe("when we stop an app", func() {
		It("we can see an app stop event in the log stream", func() {
			session := cf.Cf("curl", "-X", "POST", fmt.Sprintf("/v3/apps/%s/actions/stop", appGuid))
			Eventually(session).Should(Exit(0))
			session = logs.Tail(true, appName)
			Eventually(session).Should(Exit(0))
			Eventually(session).Should(gbytes.Say("Stopping app with guid"))
		})
	})
})
