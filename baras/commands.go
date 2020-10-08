package baras

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("setting_process_commands", func() {
	SkipOnK8s("process command comes back wrong, this is a known bug")
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
		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		dropletGUID = CreateAndAssociateNewDroplet(appGUID, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName())
		applyEndpoint = fmt.Sprintf("/v3/apps/%s/actions/apply_manifest", appGUID)
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, Config)
		DeleteApp(appGUID)
	})

	Describe("manifest and Procfile/detected buildpack command interactions", func() {
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
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT"))
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
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT"))
		})
	})
})
