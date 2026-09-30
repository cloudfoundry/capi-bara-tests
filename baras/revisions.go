package baras

import (
	"fmt"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The revisions suite exercises how CC creates / restores revisions
// under stop/start, restart, and rolling deploys. None of those state
// transitions are Ruby-specific, so the base fixture is Catnip (Go,
// ~10s to stage).
//
// Contexts that swap to a second droplet mostly use Staticfile —
// nginx-based, no compilation, the fastest buildpack to stage — since
// they only need a fixture that produces a distinguishable HTTP
// response and does not need to accept a custom start command. The
// single exception is the "rolling back to detected dora command"
// context, which mutates the start command to a Ruby-flavoured
// `bundle exec rackup ...` string that must actually run when the app
// starts; that context alone swaps to Dora.
//
// Custom start commands are kept fixture-appropriate: rackup for
// Ruby/Dora, ./bin/catnip for Go/Catnip. The tests only care that the
// command was set and that a matching revision was produced, so the
// literal command string is an implementation detail.
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
		// None of the revisions specs assert on multi-instance
		// behaviour — every assertion is about revision GUIDs,
		// droplet GUIDs, and process→revision links, none of which
		// need more than one running instance. Halving the instance
		// count halves the Diego "wait for all instances to start"
		// cost that every spec in this file pays, and halves the
		// per-step count of rolling deployments in the "deployment"
		// Describe's specs.
		instances = 1

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)

		By("Enabling Revisions")
		EnableRevisions(appGUID)

		dropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().CatnipZip, Config.GetGoBuildpackName())

		CreateAndMapRoute(appGUID, spaceGUID, domainGUID, appName)
		ScaleApp(appGUID, instances)

		StartApp(appGUID)
		Expect(
			string(cf.Cf("apps").Wait().Out.Contents()),
		).To(MatchRegexp(fmt.Sprintf("(v4-)?(%s)*(-web)?(\\s)+(started)", "web")))

		waitForAllInstancesToStart(appGUID, instances)

		By("checking that the app responds")
		Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))

		revisions = GetRevisions(appGUID)
		originalWebProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
		originalRevisionGUID = originalWebProcess.Relationships.Revision.Data.Guid
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real ./bin/catnip"
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/cmd")).To(Equal("real"))
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real ./bin/catnip"
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/cmd")).To(Equal("real"))
			})
		})
	})

	Describe("starting a started app", func() {
		Context("when there is a new droplet", func() {
			BeforeEach(func() {
				CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip, Config.GetStaticFileBuildpackName())
			})

			It("does not create a new revision", func() {
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
			})
		})

		Context("when there is a new command on a process", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "cmd=real ./bin/catnip"
				SetCommandOnProcess(appGUID, "web", newCommand)
			})

			It("does not create a new revision", func() {
				StartApp(appGUID)

				Expect(GetRevisions(appGUID)).To(Equal(revisions))
				newProcess := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
				Expect(newProcess.Relationships.Revision.Data.Guid).To(Equal(originalRevisionGUID))
				Expect(newProcess.Command).NotTo(Equal(newCommand))

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
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

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
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

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo2")).To(Equal("bar2"))
			})
		})

		Context("when the start command has changed on the app's processes", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real ./bin/catnip"
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
				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/TEST_VAR")).To(Equal("real"))
			})
		})

		// Rollback: the app is initially Catnip, gets swapped to Dora
		// with a Ruby-flavoured custom command and mutated env, then
		// rolled back to the very first (Catnip) revision. The context
		// name is kept for historical/git-blame stability.
		Context("rolling back to detected dora command", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName())
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
				newCommand = "TEST_VAR=real bundle exec rackup config.ru -p $PORT -o 0.0.0.0"
				SetCommandOnProcess(appGUID, "web", newCommand)
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

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo")).To(Equal("bar"))
			})
		})

		// Rollback with no droplet swap — only the command / env change.
		Context("rolling back to specified dora command", func() {
			var (
				newCommand string
			)

			BeforeEach(func() {
				newCommand = "TEST_VAR=real ./bin/catnip"
				SetCommandOnProcess(appGUID, "web", newCommand)
				UpdateEnvironmentVariables(appGUID, `{"foo":"deffo-not-bar"}`)
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

				Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
				Expect(helpers.CurlApp(Config, appName, "/env/foo")).To(Equal("bar"))
			})
		})
	})

	// Consolidated coverage of "a droplet swap creates a new revision
	// when the app is restarted, via any of the three restart-style
	// triggers".
	//
	// Previously this behaviour was covered by three sibling Contexts
	// (one per trigger: stop/start, restart, zdt deployment). Each
	// paid the full outer BeforeEach cost — push Catnip, scale, wait
	// for instances — plus staged a Staticfile droplet in its own
	// BeforeEach, for ~330s per spec. Wall time on 12-worker CI:
	// three specs × ~330s = ~1000s of duplicated setup.
	//
	// Here we run all three triggers in one spec, alternating the
	// current droplet between the base Catnip and a single-staged
	// Staticfile so each trigger has a genuinely-pending swap to
	// apply. Every original assertion (revision count grew, latest
	// revision's droplet is what we swapped in, per-process revision
	// link is up to date, HTTP response matches the running droplet)
	// is preserved for each trigger.
	//
	// Coverage: identical to the three original specs combined. The
	// scenario is sequential — if trigger 1 fails, triggers 2 and 3
	// are not exercised — but the failure diagnostic still identifies
	// which trigger regressed via the By() step markers.
	Describe("a droplet swap applies on each restart-style trigger", func() {
		var newDropletGUID string

		BeforeEach(func() {
			// Stage the alternate droplet once. We flip current-droplet
			// between this and the base `dropletGUID` before each
			// trigger so each has something new to apply.
			newDropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().StaticfileZip, Config.GetStaticFileBuildpackName())
		})

		It("stop/start, restart, and zdt each create a new revision reflecting the swapped droplet", func() {
			baseRevisions := GetRevisions(appGUID)

			By("stop/start applies the pending swap (Catnip → Staticfile)")
			StopApp(appGUID)
			WaitForAppToStop(appGUID)
			StartApp(appGUID)
			Expect(len(GetRevisions(appGUID))).To(Equal(len(baseRevisions) + 1))
			Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
			webProc := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			Expect(webProc.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			waitForAllInstancesToStart(appGUID, instances)
			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))

			By("queue a swap back to Catnip, then Restart applies it (Staticfile → Catnip)")
			AssignDropletToApp(appGUID, dropletGUID)
			RestartApp(appGUID)
			Expect(len(GetRevisions(appGUID))).To(Equal(len(baseRevisions) + 2))
			Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(dropletGUID))
			webProc = GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			Expect(webProc.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			waitForAllInstancesToStart(appGUID, instances)
			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))

			By("queue a swap to Staticfile again, then zdt applies it (Catnip → Staticfile)")
			AssignDropletToApp(appGUID, newDropletGUID)
			zdtRestartAndWait(appGUID)
			Expect(len(GetRevisions(appGUID))).To(Equal(len(baseRevisions) + 3))
			Expect(GetNewestRevision(appGUID).Droplet.Guid).To(Equal(newDropletGUID))
			webProc = GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			Expect(webProc.Relationships.Revision.Data.Guid).To(Equal(GetNewestRevision(appGUID).Guid))
			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hello from a staticfile"))
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
	deploymentGUID := CreateDeployment(appGUID, "rolling", 1)
	Expect(deploymentGUID).ToNot(BeEmpty())
	WaitUntilDeploymentReachesStatus(deploymentGUID, "FINALIZED", "DEPLOYED")
}
