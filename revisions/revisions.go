package processes

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func AssociateNewDroplet(appGUID, assetPath string) string {
	By("Creating a Package")
	packageGUID := CreatePackage(appGUID)
	uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

	By("Uploading a Package")
	UploadPackage(uploadURL, assetPath, GetAuthToken())
	WaitForPackageToBeReady(packageGUID)

	By("Creating a Build")
	buildGUID := StageBuildpackPackage(packageGUID, Config.GetRubyBuildpackName(), Config.GetStaticFileBuildpackName())
	WaitForBuildToStage(buildGUID)
	dropletGUID := GetDropletFromBuild(buildGUID)

	AssignDropletToApp(appGUID, dropletGUID)

	return dropletGUID
}

var _ = Describe("revisions", func() {
	var (
		appName      string
		appGUID      string
		spaceGUID    string
		spaceName    string
		dropletGUID  string
		revisions    []Revision
		revisionGUID string
		instances    int
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		instances = 2

		By("Creating an App")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)

		By("Enabling Revisions")
		EnableRevisions(appGUID)

		dropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().DoraZip)

		CreateAndMapRoute(appGUID, spaceName, Config.GetAppsDomain(), appName)
		ScaleApp(appGUID, instances)

		StartApp(appGUID)
		Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v4-)?(%s)*(-web)?(\\s)+(started)", "web")))

		By("waiting until all instances are running")
		Eventually(func() int {
			guids := GetProcessGuidsForType(appGUID, "web")
			Expect(guids).ToNot(BeEmpty())
			return GetRunningInstancesStats(guids[0])
		}).Should(Equal(instances))

		revisions = GetRevisions(appGUID)
		process := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
		revisionGUID = process.Relationships.Revision.Data.Guid
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	Describe("stopping and starting", func() {
		Context("when there is not a new droplet", func() {
			It("does not create a new revision", func() {
				StopApp(appGUID)
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(revisionGUID))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string
			BeforeEach(func() {
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(revisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			})
		})
	})

	Describe("restarting", func() {
		Context("when there is not a new droplet", func() {
			It("does not create a new revision", func() {
				RestartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(revisionGUID))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string
			BeforeEach(func() {
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				RestartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(revisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			})
		})
	})

	Describe("starting a started app", func() {
		Context("when there is a new droplet", func() {
			BeforeEach(func() {
				AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("does not create a new revision", func() {
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(revisionGUID))
			})
		})
	})

	Describe("deployment", func() {
		Context("when there is not a new droplet", func() {
			It("does not create a new revision", func() {
				deploymentGuid := CreateDeployment(appGUID)
				Expect(deploymentGuid).ToNot(BeEmpty())
				WaitUntilDeployed(deploymentGuid)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(revisionGUID))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string
			BeforeEach(func() {
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				deploymentGuid := CreateDeployment(appGUID)
				Expect(deploymentGuid).ToNot(BeEmpty())
				WaitUntilDeployed(deploymentGuid)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(revisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			})
		})
	})
})
