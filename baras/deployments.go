package baras

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("deployments", func() {
	var (
		appName        string
		appGUID        string
		domainGUID     string
		packageGUID    string
		newPackageGUID string
		spaceGUID      string
		spaceName      string
		token          string
		dropletGuid    string
		newDropletGuid string
	)

	Describe("", func() {

		BeforeEach(func() {
			appName = random_name.BARARandomName("APP")
			spaceName = TestSetup.RegularUserContext().Space
			spaceGUID = GetSpaceGuidFromName(spaceName)
			domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())
			By("Creating an app")
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

			AssignDropletToApp(appGUID, dropletGuid)

			CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)
			instances := 4

			ScaleApp(appGUID, instances)
			StartApp(appGUID)
			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", "web")))

			By("waiting until all instances are running")
			Eventually(func() int {
				guids := GetProcessGuidsForType(appGUID, "web")
				Expect(guids).ToNot(BeEmpty())
				return GetRunningInstancesStats(guids[0])
			}).Should(Equal(instances))
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
				numWebProcesses := 0
				for _, v3_process := range v3_processes {
					Expect(v3_process.Name).To(Equal(appName))
					if v3_process.Type == "web" {
						numWebProcesses += 1
					}
				}
				Expect(numWebProcesses).To(Equal(2))

				// Ignore older processes in the v2 world
				session := cf.Cf("curl", fmt.Sprintf("/v2/apps?results-per-page=1&page=1&q=space_guid:%s&q=name:%s", spaceGUID, appName))
				bytes := session.Wait().Out.Contents()
				var v2process struct {
					TotalResults int    `json:"total_results"`
					TotalPages   int    `json:"total_pages"`
					PrevURL      string `json:"prev_url"`
					NextURL      string `json:"next_url"`
					Resources    []struct {
						Metadata struct {
							Guid      string `json:"guid"`
							CreatedAt string `json:"created_at"`
						} `json:"metadata"`
						Entity struct {
							Name string `json:"name"`
						} `json:"entity"`
					} `json:"resources"`
				}

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

		Describe("Deploy a bad droplet on the same app", func() {
			It("does not update the last_successful_healthcheck field", func() {
				By("Creating a New Package")
				newPackageGUID = CreatePackage(appGUID)
				token = GetAuthToken()
				uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), newPackageGUID)

				By("Upload Bad Dora the Package")
				UploadPackage(uploadURL, assets.NewAssets().BadDoraZip, token)
				WaitForPackageToBeReady(newPackageGUID)

				By("Creating a Build")
				newBuildGUID := StageBuildpackPackage(newPackageGUID, Config.GetRubyBuildpackName())
				WaitForBuildToStage(newBuildGUID)

				By("Get the New Droplet GUID")
				newDropletGuid = GetDropletFromBuild(newBuildGUID)

				By("Assign the New Droplet GUID to the app")
				AssignDropletToApp(appGUID, newDropletGuid)

				By("Create a new Deployment")
				deploymentGuid := CreateDeployment(appGUID)
				Expect(deploymentGuid).ToNot(BeEmpty())

				time.Sleep(60 * time.Second)

				deploymentPath := fmt.Sprintf("/v3/deployments/%s", deploymentGuid)

				type deploymentStatus struct {
					Value           string `json:"value"`
					Reason          string `json:"reason"`
					HealthCheckTime string `json:"last_successful_healthcheck"`
				}
				deploymentJson := struct {
					State  string           `json:"state"`
					Status deploymentStatus `json:"status"`
				}{}

				session := cf.Cf("curl", "-f", deploymentPath).Wait()
				Expect(session.Wait()).To(Exit(0))
				json.Unmarshal(session.Out.Contents(), &deploymentJson)

				Expect(deploymentJson.State).To(Equal("DEPLOYING"))
				Expect(deploymentJson.Status.Value).To(Equal("ACTIVE"))
				Expect(deploymentJson.Status.Reason).To(Equal("DEPLOYING"))
				Expect(deploymentJson.Status.HealthCheckTime).To(Equal(""))
			})
		})

		Describe("Health check timeout is set on the app", func() {
			BeforeEach(func() {
				ScaleApp(appGUID, 2)
				SetHealthCheckTimeoutOnProcess(appGUID, "web", 5)
			})

			It("completes the deployment", func() {
				deploymentGUID := CreateDeployment(appGUID)
				Expect(deploymentGUID).ToNot(BeEmpty())
				WaitUntilDeploymentReachesState(deploymentGUID, "DEPLOYED")
			})

		})
	})

	FDescribe("Creating an app with kpack lifecycle", func() {
		BeforeEach(func() {
			appName = random_name.BARARandomName("APP")
			spaceName = TestSetup.RegularUserContext().Space
			spaceGUID = GetSpaceGuidFromName(spaceName)
			domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())
			By("Creating an app")
			appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
			By("Creating a Package")
			packageGUID = CreatePackage(appGUID)
			token = GetAuthToken()
			//todo: gcr public url?
			//uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

			By("Creating a Kpack Build")
			buildGUID := StageKpackPackage(packageGUID)
			WaitForBuildToStage(buildGUID)
			dropletGuid = GetDropletFromBuild(buildGUID)

		})
		It("creates a droplet", func() {
			Expect(dropletGuid).ToNot(BeEmpty())
		})
	})
})
