package baras

import (
	"fmt"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("setting_process_commands", func() {
	var (
		appName             string
		appGUID             string
		manifestToApply     string
		nullCommandManifest string
		applyEndpoint       string
		spaceGUID           string
		spaceName           string
		dropletGUID         string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		applyEndpoint = fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, Config)
		DeleteApp(appGUID)
	})

	Describe("when a buildpack doesn't return process types with start commands", func() {
		SkipOnK8s("the fix made which introduced this test hasn't been made on cf-for-k8s yet")

		BeforeEach(func() {
			By("Creating an app")
			appGUID = CreateApp(appName, spaceGUID, "{}")
		})

		Describe("if the web process doesn't already have a command", func() {
			It("fails staging with an error message", func() {
				packageGUID := CreatePackage(appGUID)
				uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

				By("Uploading a Package")
				UploadPackage(uploadURL, assets.NewAssets().PythonWithoutProcfileZip)
				WaitForPackageToBeReady(packageGUID)

				By("Creating a Build")
				buildGUID := StagePackage(packageGUID, Config.Lifecycle(), Config.GetPythonBuildpackName())
				WaitForBuildToFail(buildGUID)
				Expect(GetBuildError(buildGUID)).To(ContainSubstring("StagingError"))
			})
		})

		Describe("if the web process already has a command", func() {
			It("succeeds at staging, using the existing start command", func() {
				packageGUID := CreatePackage(appGUID)
				uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

				By("Applying Manifest with a Command")
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    command: python server.py
`, appName)

				session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				processes := GetProcesses(appGUID, appName)
				webProcessWithCommandRedacted := GetFirstProcessByType(processes, "web")
				webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
				Expect(webProcess.Command).To(Equal("python server.py"))

				By("Uploading a Package")
				UploadPackage(uploadURL, assets.NewAssets().PythonWithoutProcfileZip)
				WaitForPackageToBeReady(packageGUID)

				By("Creating a Build")
				buildGUID := StagePackage(packageGUID, Config.Lifecycle(), Config.GetPythonBuildpackName())
				WaitForBuildToStage(buildGUID)
			})
		})
	})

	Describe("manifest and Procfile/detected buildpack command interactions", func() {
		BeforeEach(func() {
			By("Creating an app")
			appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
			dropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName())
			manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    command: manifest-command.sh
`, appName)

			nullCommandManifest = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    command: null
`, appName)
		})

		It("prioritizes the manifest command over the Procfile and can be reset via the API", func() {
			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
			Expect(session.Wait()).To(Exit(0))
			response := session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			processes := GetProcesses(appGUID, appName)
			webProcessWithCommandRedacted := GetFirstProcessByType(processes, "web")
			webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			AssignDropletToApp(appGUID, dropletGUID)

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			processEndpoint := fmt.Sprintf("/v3/processes/%s", webProcessWithCommandRedacted.Guid)
			session = cf.Cf("curl", processEndpoint, "-X", "PATCH", "-d", `{ "command": null }`, "-i")
			Expect(session.Wait()).To(Exit(0))
			response = session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("200 OK"))

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT -o 0.0.0.0"))
		})

		It("prioritizes the manifest command over the Procfile and can be reset via manifest", func() {
			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
			Expect(session.Wait()).To(Exit(0))
			response := session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			processes := GetProcesses(appGUID, appName)
			webProcessWithCommandRedacted := GetFirstProcessByType(processes, "web")
			webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			AssignDropletToApp(appGUID, dropletGUID)

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			session = cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", nullCommandManifest, "-i")
			Expect(session.Wait()).To(Exit(0))
			response = session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT -o 0.0.0.0"))
		})
	})
})
