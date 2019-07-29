package baras

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("destinations", func() {
	var (
		routeGUID   string
		app1Name    string
		app2Name    string
		app1GUID    string
		app2GUID    string
		spaceName   string
		spaceGUID   string
		packageGUID string
		token       string
	)

	BeforeEach(func() {
		app1Name = random_name.BARARandomName("APP")
		app2Name = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		app1GUID = CreateApp(app1Name, spaceGUID, `{"foo1":"bar1"}`)
		app2GUID = CreateApp(app2Name, spaceGUID, `{"foo2":"bar2"}`)

		packageGUID = CreatePackage(app1GUID)
		token = GetAuthToken()
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

		UploadPackage(uploadURL, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGUID)

		buildGUID := StageBuildpackPackage(packageGUID, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGUID)
		dropletGUID := GetDropletFromBuild(buildGUID)
		AssignDropletToApp(app1GUID, dropletGUID)
		AssignDropletToApp(app2GUID, dropletGUID)

		routeGUID = CreateRoute(spaceGUID, Config.GetAppsDomain(), "host")
	})

	Describe("Insert destinations", func() {
		var response string
		JustBeforeEach(func() {
			routePath := fmt.Sprintf("/v3/route/%s/destinations", routeGUID)
			session := cf.Cf("curl", routePath)
			response = string(session.Out.Contents())
		})

		Describe("Regular Insert", func() {
			BeforeEach(func() {
				InsertDestinations(routeGUID, []string{app1GUID, app2GUID})
			})

			It("inserts both destinations", func() {
				Expect(response).To(ContainSubstring("something"))
			})
		})

		Describe("Insert with process types", func() {
			BeforeEach(func() {
				InsertDestinationsWithProcessTypes(routeGUID,
					map[string]string{
						app1GUID: "web",
						app2GUID: "worker",
					})
			})

			It("inserts both destinations with the appropriate process types", func() {
				Expect(response).To(ContainSubstring("something else"))
			})
		})

		Describe("Insert with ports", func() {
			BeforeEach(func() {
				InsertDestinationsWithPorts(routeGUID,
					map[string]int{
						app1GUID: 8080,
						app2GUID: 8081,
					})
			})
			It("inserts both destinations with the appropriate ports", func() {
				Expect(response).To(ContainSubstring("something else"))
			})
		})
	})
})
