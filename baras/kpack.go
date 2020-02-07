package baras

import (
	"fmt"

	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kpack lifecycle", func() {
	var (

		appName        string
		appGUID        string
		token          string
		dropletGuid    string
		droplet	Droplet
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


	Context("When creating a build with the kpack lifecycle", func() {

		FIt("stages the app successfully", func() {
			By("Creating an App and package")

			packageGUID := CreatePackage(appGUID)

			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)
			token = GetAuthToken()
			By("Uploading a Package")
			UploadPackage(uploadURL, assets.NewAssets().CatnipZip, token)

			WaitForPackageToBeReady(packageGUID)

			By("Creating a Build")
			buildGUID := StageKpackPackage(packageGUID)
			WaitForBuildToStage(buildGUID)

			dropletGuid = GetDropletFromBuild(buildGUID)


			droplet = GetDroplet(dropletGuid)
			Expect(droplet.State).To(Equal("STAGED"))
			Expect(droplet.Lifecycle.Type).To(Equal("kpack"))
			//Expect(droplet.image to  not be empty

			//session := cf.Cf("curl", "-f", fmt.Sprintf("v3/droplets/%s", dropletGuid)).Wait()
			//bytes := session.Wait().Out.Contents()
			//err := json.Unmarshal(bytes, &droplet)
		})
	})

	Context("When assigning a kpack droplet to an app", func() {
		It("run an app successfully", func() {

		})
	})

})
