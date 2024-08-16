package baras

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		dropletGuid    string
		newDropletGuid string
		instances      = 4
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())
		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		By("Creating a Package")
		packageGUID = CreatePackage(appGUID)
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

		By("Uploading a Package")
		UploadPackage(uploadURL, assets.NewAssets().DoraZip)
		WaitForPackageToBeReady(packageGUID)

		By("Creating a Build")
		buildGUID := StagePackage(packageGUID, Config.Lifecycle(), Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGUID)
		dropletGuid = GetDropletFromBuild(buildGUID)

		AssignDropletToApp(appGUID, dropletGuid)

		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)

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
		app_helpers.AppReport(appName)
		DeleteApp(appGUID)
	})

	// TODO: delete me once we delete v2
	Describe("Creating new processes on the same app", func() {
		It("ignores older processes on the same app", func() {
			deploymentGuid := CreateDeployment(appGUID, "rolling", 1)
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
			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), newPackageGUID)

			By("Upload Bad Dora the Package")
			UploadPackage(uploadURL, assets.NewAssets().BadDoraZip)
			WaitForPackageToBeReady(newPackageGUID)

			By("Creating a Build")
			newBuildGUID := StagePackage(newPackageGUID, Config.Lifecycle(), Config.GetRubyBuildpackName())
			WaitForBuildToStage(newBuildGUID)

			By("Get the New Droplet GUID")
			newDropletGuid = GetDropletFromBuild(newBuildGUID)

			By("Assign the New Droplet GUID to the app")
			AssignDropletToApp(appGUID, newDropletGuid)

			By("Create a new Deployment")
			deploymentGuid := CreateDeploymentForDroplet(appGUID, newDropletGuid, "rolling")
			Expect(deploymentGuid).ToNot(BeEmpty())

			time.Sleep(60 * time.Second)

			deploymentPath := fmt.Sprintf("/v3/deployments/%s", deploymentGuid)

			type deploymentStatus struct {
				Value           string `json:"value"`
				Reason          string `json:"reason"`
				HealthCheckTime string `json:"last_successful_healthcheck"`
			}
			deploymentJson := struct {
				Status deploymentStatus `json:"status"`
			}{}

			session := cf.Cf("curl", "-f", deploymentPath).Wait()
			Expect(session.Wait()).To(Exit(0))
			json.Unmarshal(session.Out.Contents(), &deploymentJson)

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
			deploymentGUID := CreateDeployment(appGUID, "rolling", 1)
			Expect(deploymentGUID).ToNot(BeEmpty())
			WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")
		})
	})

	Describe("Canary deployments", func() {
		var secondDropletGuid string
		BeforeEach(func() {
			By("Creating a second droplet for the app")
			secondDropletGuid = uploadDroplet(appGUID, assets.NewAssets().StaticfileZip, Config.GetStaticFileBuildpackName())
		})

		It("deploys an app, transitions to pause, is continued and then deploys successfully", func() {
			By("Pushing a canary deployment")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			_, originalWorkerStartEvent := GetLastAppUseEventForProcess("worker", "STARTED", "")

			deploymentGuid := CreateDeploymentForDroplet(appGUID, secondDropletGuid, "canary")
			Expect(deploymentGuid).ToNot(BeEmpty())

			Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
				Should(BeNumerically(">", 1))

			By("Waiting for the a canary deployment to be paused")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "PAUSED")

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			processGuids := GetProcessGuidsForType(appGUID, "web")
			canaryProcessGuid := processGuids[len(processGuids)-1]

			Eventually(func() int {
				return GetRunningInstancesStats(canaryProcessGuid)
			}).Should(Equal(1))

			By("Continuing the deployment")
			ContinueDeployment(deploymentGuid)

			By("Verfiying the canary process is rolled out successfully")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "FINALIZED", "DEPLOYED")

			Eventually(func() int {
				return GetRunningInstancesStats(canaryProcessGuid)
			}).Should(Equal(instances))

			counter := 0
			Eventually(func() int {
				if strings.Contains(helpers.CurlAppRoot(Config, appName), "Hello from a staticfile") {
					counter++
				} else {
					counter = 0
				}
				return counter
			}).Should(Equal(10))

			Eventually(func() bool {
				restartEventExists, _ := GetLastAppUseEventForProcess("worker", "STARTED", originalWorkerStartEvent.Guid)
				return restartEventExists
			}).Should(BeTrue(), "Did not find a start event indicating the 'worker' process restarted")
		})

		It("deploys an app, transitions to pause and can be cancelled", func() {
			By("Pushing a canary deployment")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			deploymentGuid := CreateDeploymentForDroplet(appGUID, secondDropletGuid, "canary")
			Expect(deploymentGuid).ToNot(BeEmpty())

			Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
				Should(BeNumerically(">", 1))

			By("Waiting for the a canary deployment to be paused")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "PAUSED")

			By("Checking that both the canary and original apps exist simultaneously")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			processGuids := GetProcessGuidsForType(appGUID, "web")
			originalProcessGuid := processGuids[len(processGuids)-2]
			canaryProcessGuid := processGuids[len(processGuids)-1]

			Eventually(func() int {
				return GetRunningInstancesStats(canaryProcessGuid)
			}).Should(Equal(1))

			By("Cancelling the deployment")
			CancelDeployment(deploymentGuid)

			By("Verifying the cancel succeeded and we rolled back to old process")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "FINALIZED", "CANCELED")

			Eventually(func() int {
				return GetRunningInstancesStats(originalProcessGuid)
			}).Should(Equal(instances))

			Consistently(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.CcClockCycleDuration(), "2s").ShouldNot(ContainSubstring("Hello from a staticfile"))

			counter := 0
			Eventually(func() int {
				if strings.Contains(helpers.CurlAppRoot(Config, appName), "Hi, I'm Dora") {
					counter++
				} else {
					counter = 0
				}
				return counter
			}).Should(Equal(10))
		})

		It("deploys an app, transitions to pause and can be superseded", func() {
			By("Pushing a canary deployment")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			deploymentGuid := CreateDeploymentForDroplet(appGUID, secondDropletGuid, "canary")
			Expect(deploymentGuid).ToNot(BeEmpty())

			Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
				Should(BeNumerically(">", 1))

			By("Waiting for the a canary deployment to be paused")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "PAUSED")

			processGuids := GetProcessGuidsForType(appGUID, "web")
			canaryProcessGuid := processGuids[len(processGuids)-1]

			Eventually(func() int {
				return GetRunningInstancesStats(canaryProcessGuid)
			}).Should(Equal(1))

			By("Superseding the deployment with a new rolling deployment")
			newDeploymentGuid := CreateDeploymentForDroplet(appGUID, dropletGuid, "rolling")

			By("Verifying the new deployment is used")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "FINALIZED", "SUPERSEDED")
			WaitUntilDeploymentReachesStatus(newDeploymentGuid, "FINALIZED", "DEPLOYED")

			processGuids = GetProcessGuidsForType(appGUID, "web")
			newProcessGuid := processGuids[len(processGuids)-1]
			Eventually(func() int {
				return GetRunningInstancesStats(newProcessGuid)
			}).Should(Equal(instances))

			Consistently(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.CcClockCycleDuration(), "2s").ShouldNot(ContainSubstring("Hello from a staticfile"))

			counter := 0
			Eventually(func() int {
				if strings.Contains(helpers.CurlAppRoot(Config, appName), "Hi, I'm Dora") {
					counter++
				} else {
					counter = 0
				}
				return counter
			}).Should(Equal(10))
		})
	})

	Describe("max-in-flight deployments", func() {
		It("deploys an app with max_in_flight with a rolling deployment", func() {
			By("Pushing a new rolling deployment with max in flight of 4")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			deploymentGuid := CreateDeployment(appGUID, "rolling", 4)
			Expect(deploymentGuid).ToNot(BeEmpty())

			Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
				Should(BeNumerically(">", 1))

			processGuids := GetProcessGuidsForType(appGUID, "web")
			newDeploymentGuid := processGuids[len(processGuids)-1]

			By("Ensuring that the new process starts at 4")
			Consistently(func() int {
				return GetProcessByGuid(newDeploymentGuid).Instances
			}).Should(Equal(4))

			Eventually(func() int {
				return GetRunningInstancesStats(newDeploymentGuid)
			}).Should(Equal(instances))
		})

		It("deploys an app with max_in_flight after a canary deployment has been continued", func() {
			By("Pushing a canary deployment")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			deploymentGuid := CreateDeployment(appGUID, "canary", 4)
			Expect(deploymentGuid).ToNot(BeEmpty())

			Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
				Should(BeNumerically(">", 1))

			By("Waiting for the a canary deployment to be paused")
			WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "PAUSED")

			processGuids := GetProcessGuidsForType(appGUID, "web")
			newDeploymentGuid := processGuids[len(processGuids)-1]

			By("Continuing the deployment")
			ContinueDeployment(deploymentGuid)
			Eventually(func() int {
				return GetProcessByGuid(newDeploymentGuid).Instances
			}).ShouldNot(Equal(1))

			By("Ensuring that the new process continues at max-in-flight 4")
			Consistently(func() int {
				return GetProcessByGuid(newDeploymentGuid).Instances
			}).Should(Equal(4))

			Eventually(func() int {
				return GetRunningInstancesStats(newDeploymentGuid)
			}).Should(Equal(instances))
		})
	})
})

func uploadDroplet(appGuid, zipFile, buildpackName string) string {
	packageGuid := CreatePackage(appGuid)
	url := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

	UploadPackage(url, zipFile)
	WaitForPackageToBeReady(packageGuid)

	buildGuid := StagePackage(packageGuid, "buildpack", buildpackName)
	WaitForBuildToStage(buildGuid)
	return GetDropletFromBuild(buildGuid)
}
