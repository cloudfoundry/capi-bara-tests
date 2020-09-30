package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Droplets", func() {
	SkipOnK8s("test uses staticfile_buildpack")
	var (
		appGUID string
		appName string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName := TestSetup.RegularUserContext().Space
		spaceGUID := GetSpaceGuidFromName(spaceName)
		domainGUID := GetDomainGUIDFromName(Config.GetAppsDomain())

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
				"-b", "staticfile_buildpack",
				"-p", assets.NewAssets().Staticfile,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("download-droplet",
				appName,
				"--path", "/tmp/app.tgz",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("push",
				appName2,
				"--droplet", "/tmp/app.tgz",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(helpers.CurlAppRoot(Config, appName2)).Should(Equal("Hello from a staticfile"))
		})

		It("The app successfully runs with a user uploaded droplet", func() {
			droplet := app_helpers.AppDroplet{
				AppGUID: appGUID,
				Config:  Config,
			}

			err := droplet.Create()
			Expect(err).ToNot(HaveOccurred())

			droplet.UploadFrom(assets.NewAssets().DoraDroplet)

			v3_helpers.AssignDropletToApp(droplet.AppGUID, droplet.GUID)
			session := cf.Cf("start", appName)
			Eventually(session).Should(Exit(0))
			Eventually(helpers.CurlAppRoot(Config, appName)).Should(Equal("Hi, I'm Dora!"))
		})
	})
})
