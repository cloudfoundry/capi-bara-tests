package baras

import (
	"fmt"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("revisions", func() {
	var (
		appName              string
		appGUID              string
		domainGUID           string
		spaceGUID            string
		spaceName            string
		dropletGUID          string
		revisions            []Revision
		originalRevisionGUID string
		instances            int
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())
		instances = 2

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)

		By("Enabling Revisions")
		EnableRevisions(appGUID)

		dropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName())

		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)
		ScaleApp(appGUID, instances)

		StartApp(appGUID)
		Expect(
			string(cf.Cf("apps").Wait().Out.Contents()),
		).To(MatchRegexp(fmt.Sprintf("(v4-)?(%s)*(-web)?(\\s)+(started)", "web")))

		waitForAllInstancesToStart(appGUID, instances)

		By("checking that dora responds")
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		revisions = GetRevisions(appGUID)
		originalWebProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
		originalRevisionGUID = originalWebProcess.Relationships.Revision.Data.Guid
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, Config)
		DeleteApp(appGUID)
	})

	Describe("stopping and starting", func() {
		Context("when there is not a new droplet or env vars", func() {
			It("does not create a new revision", func() {
				StopApp(appGUID)
				WaitForAppToStop(appGUID)
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})

		Context("when environment variables have changed on the app", func() {
			BeforeEach(func() {
				UpdateEnvironmentVariables(appGUID, `{"foo2":"bar2"}`)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				WaitForAppToStop(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				WaitForAppToStop(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				Expect(GetNewestRevision(appGUID).Processes["web"]["command"]).To(Equal(newCommand))

				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/cmd")).To(Equal("real"))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string

			BeforeEach(func() {
				newDropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				WaitForAppToStop(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
			})
		})

		Context("when a sidecar has been added", func() {
			BeforeEach(func() {
				CreateSidecar("sleepy", []string{"web"}, "sleep infinity", 50, appGUID)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				WaitForAppToStop(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Sidecars[0].Name).To(Equal("sleepy"))
			})
		})
	})

	Describe("restarting", func() {
		Context("when there is not a new droplet or env vars", func() {
			It("does not create a new revision", func() {
				RestartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})

		Context("when environment variables have changed on the app", func() {
			BeforeEach(func() {
				UpdateEnvironmentVariables(appGUID, `{"foo2":"bar2"}`)
			})

			It("creates a new revision", func() {
				RestartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("creates a new revision", func() {
				RestartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				Expect(GetNewestRevision(appGUID).Processes["web"]["command"]).To(Equal(newCommand))

				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/cmd")).To(Equal("real"))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string

			BeforeEach(func() {
				newDropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())
			})

			It("creates a new revision", func() {
				RestartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
			})
		})
	})

	Describe("starting a started app", func() {
		Context("when there is a new droplet", func() {
			BeforeEach(func() {
				CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())
			})

			It("does not create a new revision", func() {
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})

		Context("when there is a new command on a process", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("does not create a new revision", func() {
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))
				Expect(newProcess.Command).NotTo(Equal(newCommand))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})
	})

	Describe("deployment", func() {
		Context("when there is not a new droplet or env vars", func() {
			It("does not create a new revision", func() {
				zdtRestartAndWait(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})

		Context("when environment variables have changed on the app", func() {
			BeforeEach(func() {
				UpdateEnvironmentVariables(appGUID, `{"foo2":"bar2"}`)
			})

			It("creates a new revision", func() {
				zdtRestartAndWait(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("creates a new revision", func() {
				zdtRestartAndWait(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				Expect(GetNewestRevision(appGUID).Processes["web"]["command"]).To(Equal(newCommand))

				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/TEST_VAR")).To(Equal("real"))
			})
		})

		Context("when there is a new droplet", func() {
			var newDropletGUID string

			BeforeEach(func() {
				newDropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())
			})

			It("creates a new revision", func() {
				zdtRestartAndWait(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
			})
		})

		Context("rolling back to detected dora command", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
				newCommand = "TEST_VAR=real ./bin/catnip"
				SetCommandOnProcess(appGUID, "web", newCommand)
				zdtRestartAndWait(appGUID)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
			})

			It("creates a new revision with the droplet, environment variables, and detected start command from the specified revision", func() {
				deploymentGUID := RollbackDeployment(appGUID, originalRevisionGUID)
				Expect(deploymentGUID).ToNot(BeEmpty())
				WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 2))
				revision := GetNewestRevision(appGUID)
				Expect(revision.Droplet.Guid).To(Equal(dropletGUID))
				Expect(revision.Guid).NotTo(Equal(originalRevisionGUID))

				Expect(GetRevisionEnvVars(originalRevisionGUID).Var["foo"]).To(Equal("bar"))
				Expect(revision.Processes["web"]["command"]).To(Equal(GetRevision(originalRevisionGUID).Processes["web"]["command"]))

				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo")).To(Equal("bar"))
			})
		})

		Context("rolling back to specified dora command", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
				zdtRestartAndWait(appGUID)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})

			It("creates a new revision with the droplet, environment variables, and detected start command from the specified revision", func() {
				deploymentGUID := RollbackDeployment(appGUID, originalRevisionGUID)
				Expect(deploymentGUID).ToNot(BeEmpty())
				WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 2))
				revision := GetNewestRevision(appGUID)
				Expect(revision.Droplet.Guid).To(Equal(dropletGUID))
				Expect(revision.Guid).NotTo(Equal(originalRevisionGUID))

				Expect(GetRevisionEnvVars(originalRevisionGUID).Var["foo"]).To(Equal("bar"))
				Expect(revision.Processes["web"]["command"]).To(Equal(GetRevision(originalRevisionGUID).Processes["web"]["command"]))

				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo")).To(Equal("bar"))
			})
		})
	})
})

func waitForAllInstancesToStart(appGUID string, instances int) {
	By("waiting until all instances are running")
	Eventually(func() int {
		guids := GetProcessGuidsForType(appGUID, "web")
		Expect(guids).ToNot(BeEmpty())
		return GetRunningInstancesStats(guids[0])
	}, Config.CfPushTimeoutDuration()).Should(Equal(instances))
}

func zdtRestartAndWait(appGUID string) {
	deploymentGUID := CreateDeployment(appGUID)
	Expect(deploymentGUID).ToNot(BeEmpty())
	WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")
}
