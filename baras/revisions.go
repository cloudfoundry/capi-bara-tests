package baras

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("revisions", func() {
	var (
		appName              string
		appGUID              string
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

		waitForAllInstancesToStart(appGUID, instances)

		By("checking that dora responds")
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))

		revisions = GetRevisions(appGUID)
		originalWebProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
		originalRevisionGUID = originalWebProcess.Relationships.Revision.Data.Guid
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	Describe("stopping and starting", func() {
		Context("when there is not a new droplet or env vars", func() {
			It("does not create a new revision", func() {
				StopApp(appGUID)
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
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
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
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				StopApp(appGUID)
				StartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
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
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT"
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
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				RestartApp(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				waitForAllInstancesToStart(appGUID, instances)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
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
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})
		})

		Context("when there is a new command on a process", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real bundle exec rackup config.ru -p $PORT"
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
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real bundle exec rackup config.ru -p $PORT"
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
				newDropletGUID = AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
			})

			It("creates a new revision", func() {
				zdtRestartAndWait(appGUID)

				Expect(len(GetRevisions(appGUID))).To(Equal(len(revisions) + 1))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
				Expect(GetNewestRevision(appGUID).Guid).NotTo(Equal(originalRevisionGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
			})
		})

		Context("rolling back to detected dora command", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				AssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip)
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
				newCommand = "TEST_VAR=real /home/vcap/app/boot.sh"
				SetCommandOnProcess(appGUID, "web", newCommand)
				zdtRestartAndWait(appGUID)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
			})

			It("creates a new revision with the droplet, environment variables, and detected start command from the specified revision", func() {
				deploymentGUID := RollbackDeployment(appGUID, originalRevisionGUID)
				Expect(deploymentGUID).ToNot(BeEmpty())
				WaitUntilDeployed(deploymentGUID)

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
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real bundle exec rackup config.ru -p $PORT"
				SetCommandOnProcess(appGUID, "web", newCommand)
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
				zdtRestartAndWait(appGUID)
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
			})

			It("creates a new revision with the droplet, environment variables, and detected start command from the specified revision", func() {
				deploymentGUID := RollbackDeployment(appGUID, originalRevisionGUID)
				Expect(deploymentGUID).ToNot(BeEmpty())
				WaitUntilDeployed(deploymentGUID)

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

var _ = XDescribe("mix v2 apps and v3 revisions", func() {
	var (
		appName              string
		appGUID              string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		session := cf.Cf("push", appName, "-p", assets.NewAssets().Dora)
		Expect(session.Wait()).To(Exit(0))
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
		session = cf.Cf("app", appName, "--guid")
		Expect(session.Wait()).To(Exit(0))
		appGUID = strings.TrimSpace(string(session.Out.Contents()))
		session = cf.Cf("curl", "-X", "PATCH", fmt.Sprintf("/v3/apps/%s/features/revisions",appGUID), "-d",
			`{"enabled" : true }`)
		Expect(session.Wait()).To(Exit(0))

	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	It("runs the latest droplet and doesn't add revisions", func() {
		Expect(cf.Cf("push",
			appName,
			"-b", "staticfile_buildpack",
			"-p", assets.NewAssets().Staticfile,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
		session := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/revisions",appGUID))
		Expect(session.Wait()).To(Exit(0))
		revstr := session.Out.Contents()

		type revisionsType struct {
			Pagination struct {
				TotalResults int `json:"total_results"`
			} `json:"pagination"`
		}

		revs := revisionsType{}
		err := json.Unmarshal(revstr, &revs)
		Expect(err).NotTo(HaveOccurred())
		Expect(revs.Pagination.TotalResults).To(Equal(0))

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
	WaitUntilDeployed(deploymentGUID)
}
