package processes

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
)

import (
	"fmt"
	"encoding/json"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type appsResultType struct {
	TotalResults int `json:"total_results"`
	TotalPages int `json:"total_pages"`
	PrevURL string `json:"prev_url"`
	NextURL string `json:"next_url"`
	Resources []struct {
		Metadata struct {
			Guid string `json:"guid"`
			CreatedAt string `json:"created_at"`
		} `json:"metadata"`
		Entity struct {
			Name string `json:"name"`
		} `json:"entity"`
	} `json:"resources"`
}

var _ = FDescribe("webish_processes", func() {
	var (
		appName             string
		appGUID             string
		packageGUID         string
		spaceGUID           string
		spaceName           string
		token               string
		dropletGuid         string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		By("Creating an App")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		By("Creating a Package")
		packageGUID = CreatePackage(appGUID)
		token = GetAuthToken()
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

		By("Uploading a Package")
		UploadPackage(uploadURL, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGUID)

		By("Creating a Build")
		buildGUID := StageBuildpackPackage(packageGUID, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGUID)
		dropletGuid = GetDropletFromBuild(buildGUID)

	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, token, Config)
		DeleteApp(appGUID)
	})

	Describe("Creating new processes on the same app", func() {
		It("ignores older processes on the same app", func() {
			deploymentGuid := CreateDeployment(appGUID)
			Expect(deploymentGuid).ToNot(BeEmpty())
			v3_processes := GetProcesses(appGUID, appName)
			Expect(len(v3_processes)).To(Equal(2))
			for _, v3_process := range(v3_processes) {
				Expect(v3_process.Name).To(Equal(appName))
			}

			// Ignore older processes in the v2 world
			session := cf.Cf("curl", "/v2/apps?results-per-page=1&page=1")
			Expect(session).To(Exit(0))
			bytes := session.Wait().Out.Contents()
			var v2process appsResultType
			json.Unmarshal(bytes, &v2process)
			Expect(len(v2process.Resources)).To(Equal(1))
			Expect(v2process.TotalResults).To(Equal(1))
			Expect(v2process.TotalPages).To(Equal(1))
			Expect(v2process.PrevURL).To(Equal(""))
			Expect(v2process.NextURL).To(Equal(""))
			Expect(v2process.Resources[0].Metadata.Guid).To(Equal(appGUID))
			Expect(v2process.Resources[0].Entity.Name).To(Equal(appName))
		})
	})
})
			
