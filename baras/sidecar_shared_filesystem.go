package baras

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sidecars", func() {
	var (
		appName    string
		appGUID    string
		spaceGUID  string
		domainGUID string
		spaceName  string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{}`)
		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)

		// SidecarDependent is an app that talks to its adjacent ./sidecar binary
		// over a unix domain socket in /tmp/sidecar.sock
		Expect(cf.Cf("push",
			appName,
			"-b", "go_buildpack",
			"-p", assets.NewAssets().SidecarDependent,
			"-f", filepath.Join(assets.NewAssets().SidecarDependent, "manifest.yml"),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	Context("when the app has a sidecar associated with its web process", func() {
		BeforeEach(func() {
			CreateSidecar("my_sidecar1", []string{"web"}, "./sidecar", 5, appGUID)
			RestartApp(appGUID)
		})

		It("the main app can communicate with its sidecar on a unix domain socket", func() {
			Eventually(func() *Session {
				session := helpers.Curl(Config, fmt.Sprintf("%s.%s", appName, Config.GetAppsDomain()))
				Eventually(session).Should(Exit(0))
				return session
			}, Config.DefaultTimeoutDuration()).Should(Say("Sidecar received your data"))
		})
	})
})
