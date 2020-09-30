package baras

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Private Docker Registry Application Lifecycle", func() {
	var (
		appName  string
		username string
		password string
	)

	type dockerCreds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type createAppRequest struct {
		Name              string      `json:"name"`
		SpaceGuid         string      `json:"space_guid"`
		DockerImage       string      `json:"docker_image"`
		DockerCredentials dockerCreds `json:"docker_credentials"`
	}

	BeforeEach(func() {
		if !Config.GetIncludePrivateDockerRegistry() {
			Skip(skip_messages.SkipPrivateDockerRegistryMessage)
		}
	})

	JustBeforeEach(func() {
		spaceName := TestSetup.RegularUserContext().Space
		session := cf.Cf("space", spaceName, "--guid")
		Eventually(session).Should(Exit(0))
		spaceGuid := string(session.Out.Contents())
		spaceGuid = strings.TrimSpace(spaceGuid)
		appName = random_name.BARARandomName("APP")

		newAppRequest, err := json.Marshal(createAppRequest{
			Name:        appName,
			SpaceGuid:   spaceGuid,
			DockerImage: Config.GetPrivateDockerRegistryImage(),
			DockerCredentials: dockerCreds{
				Username: username,
				Password: password,
			}})

		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("cf", "curl", "-X", "POST", "/v2/apps", "-d", string(newAppRequest))
		cfCurlSession, err := Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		// Redact the docker password from the test logs
		cmd.Args[6] = strings.Replace(cmd.Args[6], `"password":"`+password+`"`, `"password":"***"`, 1)
		reporter := commandreporter.NewCommandReporter()
		reporter.Report(time.Now(), cmd)

		Eventually(cfCurlSession).Should(Exit(0))
	})

	AfterEach(func() {
		appGuid := GetAppGuid(appName)
		FetchRecentLogs(appGuid, Config)
		DeleteApp(appGuid)
	})

	Context("when an incorrect username and password are given", func() {
		BeforeEach(func() {
			username = Config.GetPrivateDockerRegistryUsername() + "wrong"
			password = Config.GetPrivateDockerRegistryPassword() + "wrong"
		})

		It("fails to start the docker app since the credentials are invalid", func() {
			session := cf.Cf("start", appName)
			Eventually(session, Config.CfPushTimeoutDuration()).Should(gbytes.Say("(invalid username/password|[Uu]nauthorized)"))
			Eventually(session, Config.CfPushTimeoutDuration()).Should(Exit(1))
		})
	})
})
