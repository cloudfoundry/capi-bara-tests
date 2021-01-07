package baras

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Zero downtime operations", func() {
	var (
		appName string
		appGUID string
	)

	BeforeEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			session := cf.Cf("target", "-o", TestSetup.RegularUserContext().Org, "-s", TestSetup.RegularUserContext().Space)
			Eventually(session).Should(Exit(0))

			appName = random_name.BARARandomName("APP")
			Expect(cf.Cf("push",
				appName,
				"-p", assets.NewAssets().Dora,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			session = cf.Cf("app", appName, "--guid")
			Expect(session.Wait()).To(Exit(0))
			appGUID = strings.TrimSpace(string(session.Out.Contents()))
			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Hi, I'm Dora!"))
		})
	})

	Context("When scaling memory on an app", func() {
		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := strconv.ParseFloat(helpers.CurlApp(Config, appName, "/uptime"), 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime).To(BeNumerically(">", 0))

			ScaleProcess(appGUID, "web", "1500")

			// Allow time for diego clock sync to occur (default=30s)
			time.Sleep(35 * time.Second)
			currentUptime, err := strconv.ParseFloat(helpers.CurlApp(Config, appName, "/uptime"), 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(currentUptime).To(BeNumerically(">", originalUptime+33))
		})
	})

	Context("When changing the healthcheck type on an app", func() {
		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := strconv.ParseFloat(helpers.CurlApp(Config, appName, "/uptime"), 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime).To(BeNumerically(">", 0))

			process := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			path := fmt.Sprintf("v3/processes/%s", process.Guid)
			session := cf.Cf("curl", "-X", "PATCH", path, "-d", `{"health_check": {"type": "process"}}`).Wait()
			Eventually(session).Should(Exit(0))
			result := session.Out.Contents()
			Expect(strings.Contains(string(result), "errors")).To(BeFalse())
			// Allow time for diego clock sync to occur (default=30s)
			time.Sleep(35 * time.Second)
			currentUptime, err := strconv.ParseFloat(helpers.CurlApp(Config, appName, "/uptime"), 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(currentUptime).To(BeNumerically(">", originalUptime+33))
		})
	})
})
