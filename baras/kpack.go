package baras

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Kpack lifecycle decomposed", func() {
	SkipOnVMs("no kpack on vms")
	var (
		appName     string
		appGUID     string
		dropletGUID string
		droplet     Droplet
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName := TestSetup.RegularUserContext().Space
		spaceGUID := GetSpaceGuidFromName(spaceName)
		domainGUID := GetDomainGUIDFromName(Config.GetAppsDomain())

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)

		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))
	})

	AfterEach(func() {
		DeleteApp(appGUID)
	})

	Context("When creating a build with the kpack lifecycle", func() {
		It("stages and starts the app successfully", func() {
			By("Creating an App and package")

			packageGUID := CreatePackage(appGUID)

			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)
			By("Uploading a Package")
			UploadPackage(uploadURL, assets.NewAssets().CatnipZip)

			WaitForPackageToBeReady(packageGUID)

			By("Creating a Build")
			buildGUID := StagePackage(packageGUID, Config.Lifecycle())
			WaitForBuildToStage(buildGUID)

			dropletGUID = GetDropletFromBuild(buildGUID)

			droplet = GetDroplet(dropletGUID)
			Expect(droplet.State).To(Equal("STAGED"))
			Expect(droplet.Lifecycle.Type).To(Equal("kpack"))
			Expect(droplet.Image).ToNot(BeEmpty())

			AssignDropletToApp(appGUID, dropletGUID)
			session := cf.Cf("start", appName)

			Eventually(session).Should(gexec.Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("Catnip?"))
		})
	})

	Context("When diego_docker is disabled", func() {
		var response map[string]interface{}

		BeforeEach(func() {
			response = make(map[string]interface{})
			TestSetup.AdminUserContext().Login()
			Eventually(cf.Cf("disable-feature-flag", "diego_docker")).Should(gexec.Exit(0))
			TestSetup.RegularUserContext().Login()
		})

		It("starts and restarts the app successfully", func() {
			By("Creating an App and package")
			packageGUID := CreatePackage(appGUID)

			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)
			By("Uploading a Package")
			UploadPackage(uploadURL, assets.NewAssets().CatnipZip)

			WaitForPackageToBeReady(packageGUID)

			By("Creating a Build")
			buildGUID := StagePackage(packageGUID, Config.Lifecycle())
			WaitForBuildToStage(buildGUID)

			dropletGUID = GetDropletFromBuild(buildGUID)

			droplet = GetDroplet(dropletGUID)
			Expect(droplet.State).To(Equal("STAGED"))
			Expect(droplet.Lifecycle.Type).To(Equal("kpack"))
			Expect(droplet.Image).ToNot(BeEmpty())

			AssignDropletToApp(appGUID, dropletGUID)

			By("Starting the app")
			session := cf.Cf("curl", "-X", "POST", fmt.Sprintf("/v3/apps/%s/actions/start", appGUID))
			Eventually(session).Should(gexec.Exit(0))

			Expect(json.Unmarshal(session.Out.Contents(), &response)).To(Succeed())
			errors, errorPresent := response["errors"]
			Expect(errorPresent).ToNot(BeTrue(), fmt.Sprintf("%v", errors))

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("Catnip?"))

			By("Restarting the app")
			StopApp(appGUID)

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("no healthy upstream"))

			session = cf.Cf("curl", "-X", "POST", fmt.Sprintf("/v3/apps/%s/actions/restart", appGUID))
			Eventually(session).Should(gexec.Exit(0))

			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("Catnip?"))
		})
	})
})
