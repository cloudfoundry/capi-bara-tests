package baras

import (
	"encoding/json"
	"fmt"
	"strings"

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
		// Default web-process scale is 2. Most specs in this file
		// assert only on 'canary process has N instances' where N is
		// specific to the moment in the deployment lifecycle (usually
		// 1 during pause, then `instances` after finalisation) — they
		// work correctly with any 'instances >= 2' and gain nothing
		// from 4. The three specs that genuinely need 4
		// ('with instance steps' asserts canary=i for i in 1..4, both
		// max-in-flight specs assert `Consistently Instances == 4`)
		// scale up to 4 locally in their own BeforeEach.
		instances = 2
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
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.GetApiProtocol(), Config.GetApiEndpoint(), packageGUID)

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

	Describe("Deploy a bad droplet on the same app", func() {
		It("does not update the last_successful_healthcheck field", func() {
			By("Creating a New Package")
			newPackageGUID = CreatePackage(appGUID)
			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.GetApiProtocol(), Config.GetApiEndpoint(), newPackageGUID)

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

			deploymentPath := fmt.Sprintf("/v3/deployments/%s", deploymentGuid)

			type deploymentStatus struct {
				Value           string `json:"value"`
				Reason          string `json:"reason"`
				HealthCheckTime string `json:"last_successful_healthcheck"`
			}
			deploymentJson := struct {
				Status deploymentStatus `json:"status"`
			}{}

			// Verify that for at least one CC deployment-updater cycle the
			// deployment stays ACTIVE/DEPLOYING and no successful healthcheck
			// is ever recorded (bad-dora never boots). Polling instead of a
			// hard Sleep means we fail fast if the state regresses.
			Consistently(func() deploymentStatus {
				session := cf.Cf("curl", "-f", deploymentPath).Wait()
				Expect(session).To(Exit(0))
				deploymentJson.Status = deploymentStatus{}
				json.Unmarshal(session.Out.Contents(), &deploymentJson)
				return deploymentJson.Status
			}, Config.CcClockCycleDuration(), "2s").Should(Equal(deploymentStatus{
				Value:           "ACTIVE",
				Reason:          "DEPLOYING",
				HealthCheckTime: "",
			}))
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

		Context("with instance steps", func() {
			// Instance-steps asserts canary process running-instance
			// count = i at each step for i in 1..4, so the app needs
			// 4 web instances. The Describe-level default was reduced
			// to 2; scale back up to 4 locally.
			BeforeEach(func() {
				ScaleApp(appGUID, 4)
				Eventually(func() int {
					guids := GetProcessGuidsForType(appGUID, "web")
					Expect(guids).ToNot(BeEmpty())
					return GetRunningInstancesStats(guids[0])
				}, Config.CfPushTimeoutDuration()).Should(Equal(4))
			})

			It("deploys an app, transitions to pause and can be continued multiple times and then deploys successfully", func() {
				By("Pushing a canary deployment")
				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}).Should(ContainSubstring("Hi, I'm Dora"))

				_, originalWorkerStartEvent := GetLastAppUseEventForProcess("worker", "STARTED", "")

				instanceWeights := []int{1, 50, 75, 99}

				deploymentGuid := CreateCanaryDeploymentWithWeightsForDroplet(appGUID, secondDropletGuid, instanceWeights)
				Expect(deploymentGuid).ToNot(BeEmpty())

				Eventually(func() int { return len(GetProcessGuidsForType(appGUID, "web")) }, Config.CfPushTimeoutDuration()).
					Should(BeNumerically(">", 1))

				for i := 1; i <= len(instanceWeights); i++ {
					By(fmt.Sprintf("Waiting for the a canary deployment to be paused on step %d", i))
					WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "PAUSED")

					By(fmt.Sprintf("Verifying both original and canary apps are available on step %d", i))
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
					}).Should(Equal(i))

					By(fmt.Sprintf("Continuing the deployment on step %d", i))
					ContinueDeployment(deploymentGuid)
					WaitUntilDeploymentReachesStatus(deploymentGuid, "ACTIVE", "DEPLOYING")
				}

				By("Verfiying the canary process is rolled out successfully")
				WaitUntilDeploymentReachesStatus(deploymentGuid, "FINALIZED", "DEPLOYED")

				processGuids := GetProcessGuidsForType(appGUID, "web")
				canaryProcessGuid := processGuids[len(processGuids)-1]

				// The 'with instance steps' Context scaled the app up
				// to 4 in its BeforeEach; the outer default 'instances'
				// is 2. Hardcode 4 here to reflect what the test
				// actually set up.
				Eventually(func() int {
					return GetRunningInstancesStats(canaryProcessGuid)
				}).Should(Equal(4))

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
		})
	})

	Describe("max-in-flight deployments", func() {
		// Both max-in-flight specs assert 'Consistently new_process
		// Instances == 4' and Eventually run count == 4 — the app
		// must have 4 web instances. The Describe-level default is 2;
		// scale back up to 4 locally.
		BeforeEach(func() {
			ScaleApp(appGUID, 4)
			Eventually(func() int {
				guids := GetProcessGuidsForType(appGUID, "web")
				Expect(guids).ToNot(BeEmpty())
				return GetRunningInstancesStats(guids[0])
			}, Config.CfPushTimeoutDuration()).Should(Equal(4))
		})

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
			}).Should(Equal(4))
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
			}).Should(Equal(4))
		})
	})
})

// The "health-check timeout" scenario needs an app that boots fast
// enough for a 5-second timeout to be a comfortable margin, otherwise
// the deployment races the healthcheck and the test flakes. Ruby Dora
// can easily exceed 5s of boot on a busy environment; Catnip is a
// static Go binary that starts in well under a second. This suite is
// therefore its own top-level Describe with a Catnip-based setup — it
// deliberately does not share the Dora BeforeEach above.
var _ = Describe("deployment with a tight healthcheck timeout", func() {
	var (
		appName    string
		appGUID    string
		spaceGUID  string
		spaceName  string
		domainGUID string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())

		By("Creating a Catnip-backed app")
		appGUID = CreateApp(appName, spaceGUID, `{}`)
		_ = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())

		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)
		ScaleApp(appGUID, 2)
		StartApp(appGUID)

		By("waiting until all instances are running")
		Eventually(func() int {
			guids := GetProcessGuidsForType(appGUID, "web")
			Expect(guids).ToNot(BeEmpty())
			return GetRunningInstancesStats(guids[0])
		}, Config.CfPushTimeoutDuration()).Should(Equal(2))

		SetHealthCheckTimeoutOnProcess(appGUID, "web", 5)
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		DeleteApp(appGUID)
	})

	It("completes the deployment", func() {
		deploymentGUID := CreateDeployment(appGUID, "rolling", 1)
		Expect(deploymentGUID).ToNot(BeEmpty())
		WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")
	})
})

func uploadDroplet(appGuid, zipFile, buildpackName string) string {
	packageGuid := CreatePackage(appGuid)
	url := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.GetApiProtocol(), Config.GetApiEndpoint(), packageGuid)

	UploadPackage(url, zipFile)
	WaitForPackageToBeReady(packageGuid)

	buildGuid := StagePackage(packageGuid, "buildpack", buildpackName)
	WaitForBuildToStage(buildGuid)
	return GetDropletFromBuild(buildGuid)
}
