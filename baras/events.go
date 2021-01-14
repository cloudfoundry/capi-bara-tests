package baras

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("events", func() {
	var (
		appName string
		appGuid string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))

		session = cf.Cf("push", appName, "-p", assets.NewAssets().Catnip)
		Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		appGuid = GetAppGuid(appName)
	})

	AfterEach(func() {
		DeleteApp(appGuid)
	})

	Context("when we stop an app", func() {
		BeforeEach(func() {
			stopSession := cf.Cf("curl", "-X", "POST", fmt.Sprintf("/v3/apps/%s/actions/stop", appGuid))
			Eventually(stopSession).Should(Exit(0))
		})

		It("we can see an app stop event in the log stream", func() {
			Eventually(func() *gexec.Session {
				session := cf.Cf("logs", appName, "--recent")
				Eventually(session).Should(Exit(0))
				return session
			}).Should(gbytes.Say("Stopping app with guid"))
		})
	})

	Context("when we apply a manifest", func() {
		It("we can see the manifest application in the log stream", func() {
			manifestToApply := fmt.Sprintf(`
---
applications:
- name: %s
  processes:
  - type: web
    instances: 1
    memory: 4096M
    disk_quota: 1024M
    health-check-type: http
    health-check-http-endpoint: '/'
`, appName)
			applyEndpoint := fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", GetSpaceGuidFromName(TestSetup.RegularUserContext().Space))

			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
			Eventually(session).Should(Exit(0))

			Eventually(func() *gexec.Session {
				session = cf.Cf("logs", appName, "--recent")
				Eventually(session).Should(Exit(0))
				return session
			}).Should(gbytes.Say("Applied manifest to app"))
		})
	})
})
