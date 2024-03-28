package baras

import (
	"fmt"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"os"
)

var _ = Describe("Droplet upload and download", func() {
	var (
		appGUID string
		appName string
		token   string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName := TestSetup.RegularUserContext().Space
		spaceGUID := GetSpaceGuidFromName(spaceName)
		domainGUID := GetDomainGUIDFromName(Config.GetAppsDomain())
		token = GetAuthToken()

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, Config)
		DeleteApp(appGUID)
	})

	Context("When manually performing the droplet workflow", func() {
		It("Downloading the droplet is successful", func() {
			appName2 := random_name.BARARandomName("APP2")
			Expect(cf.Cf("push",
				appName,
				"-b", Config.GetGoBuildpackName(),
				"-p", assets.NewAssets().CatnipZip,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			tmpDir, err := os.CreateTemp("", "droplet")
			dropletPath := fmt.Sprintf("%s.tgz", tmpDir.Name())
			Expect(err).NotTo(HaveOccurred())

			// App droplet needs to be downloaded with curl due to:
			// https://github.com/cloudfoundry/cli/issues/2225
			DownloadAppDroplet(appGUID, dropletPath, token)

			Expect(cf.Cf("push",
				appName2,
				"--droplet", dropletPath,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(helpers.CurlAppRoot(Config, appName2)).Should(Equal("Catnip?"))
		})

		It("The app successfully runs with a user uploaded droplet", func() {
			droplet := app_helpers.AppDroplet{
				AppGUID: appGUID,
				Config:  Config,
			}

			err := droplet.Create()
			Expect(err).ToNot(HaveOccurred())

			droplet.UploadFrom(assets.NewAssets().DoraDroplet)

			AssignDropletToApp(droplet.AppGUID, droplet.GUID)
			session := cf.Cf("start", appName)
			Eventually(session).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(Config, appName)).Should(Equal("Hi, I'm Dora!"))
		})
	})
})
