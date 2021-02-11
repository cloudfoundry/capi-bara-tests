package baras

import (
	"fmt"
	"path/filepath"
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
				"-b", Config.GetGoBuildpackName(),
				"-p", assets.NewAssets().MultiPortApp,
				"-c", "workspace --ports=8080,8081",
				"-f", filepath.Join(assets.NewAssets().MultiPortApp, "manifest.yml"),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			session = cf.Cf("app", appName, "--guid")
			Expect(session.Wait()).To(Exit(0))
			appGUID = strings.TrimSpace(string(session.Out.Contents()))
			Expect(helpers.CurlAppRoot(Config, appName)).To(ContainSubstring("8080"))
		})
	})

	Context("When scaling memory on an app", func() {
		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime.Seconds()).To(BeNumerically(">", 0))

			ScaleProcess(appGUID, "web", "1500")

			Consistently(func() float64 {
				currentUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
				Expect(err).ToNot(HaveOccurred())
				return currentUptime.Seconds()
			}, Config.CcClockCycleDuration(), "1s").Should(BeNumerically(">", originalUptime.Seconds()))
		})
	})

	Context("When changing the healthcheck http endpoint on an app", func() {
		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime.Seconds()).To(BeNumerically(">", 0))

			process := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			path := fmt.Sprintf("v3/processes/%s", process.Guid)
			session := cf.Cf("curl", "-X", "PATCH", path, "-d", `{"health_check": {"data": {"endpoint": "/two"}}}`).Wait()
			Eventually(session).Should(Exit(0))
			result := session.Out.Contents()
			Expect(strings.Contains(string(result), "errors")).To(BeFalse())

			Consistently(func() float64 {
				currentUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
				Expect(err).ToNot(HaveOccurred())
				return currentUptime.Seconds()
			}, Config.CcClockCycleDuration(), "1s").Should(BeNumerically(">", originalUptime.Seconds()))
		})
	})

	Context("When changing the healthcheck type on an app", func() {
		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime.Seconds()).To(BeNumerically(">", 0))

			process := GetFirstProcessByType(GetProcesses(appGUID, appName), "web")
			path := fmt.Sprintf("v3/processes/%s", process.Guid)
			session := cf.Cf("curl", "-X", "PATCH", path, "-d", `{"health_check": {"type": "process"}}`).Wait()
			Eventually(session).Should(Exit(0))
			result := session.Out.Contents()
			Expect(strings.Contains(string(result), "errors")).To(BeFalse())

			Consistently(func() float64 {
				currentUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
				Expect(err).ToNot(HaveOccurred())
				return currentUptime.Seconds()
			}, Config.CcClockCycleDuration(), "1s").Should(BeNumerically(">", originalUptime.Seconds()))
		})
	})

	Context("When adding a route destination to a process", func() {
		BeforeEach(func() {
			SkipOnK8s("not entirely clear why this doesn't work on cf-for-k8s...")
		})

		It("downtime does not occur until the app is restarted", func() {
			originalUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime.Seconds()).To(BeNumerically(">", 0))

			routeGUID := GetRouteGUIDFromAppGuid(appGUID)
			path := fmt.Sprintf("v3/routes/%s/destinations", routeGUID)
			body := fmt.Sprintf(`{"destinations": [{"app": {"guid": "%s"}, "port": 8081}]}`, appGUID)
			session := cf.Cf("curl", "-X", "POST", path, "-d", body).Wait()
			Eventually(session).Should(Exit(0))
			result := session.Out.Contents()
			Expect(strings.Contains(string(result), "errors")).To(BeFalse())

			Consistently(func() float64 {
				currentUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
				Expect(err).ToNot(HaveOccurred())
				return currentUptime.Seconds()
			}, Config.CcClockCycleDuration(), "1s").Should(BeNumerically(">", originalUptime.Seconds()))

			Consistently(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration(), "5s").ShouldNot(ContainSubstring("8081"))
		})
	})

	Context("When updating route destinations on processes", func() {
		BeforeEach(func() {
			SkipOnK8s("not entirely clear why this doesn't work on cf-for-k8s...")
		})

		It("downtime does not occur until an app is restarted", func() {
			originalUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
			Expect(err).ToNot(HaveOccurred())
			Expect(originalUptime.Seconds()).To(BeNumerically(">", 0))

			routeGUID := GetRouteGUIDFromAppGuid(appGUID)
			path := fmt.Sprintf("v3/routes/%s/destinations", routeGUID)
			body := fmt.Sprintf(`{"destinations": [{"app": {"guid": "%s"}, "port": 8080}, {"app": {"guid": "%s"}, "port": 8081}]}`, appGUID, appGUID)
			session := cf.Cf("curl", "-X", "PATCH", path, "-d", body).Wait()
			Eventually(session).Should(Exit(0))
			result := session.Out.Contents()
			Expect(strings.Contains(string(result), "errors")).To(BeFalse())

			Consistently(func() float64 {
				currentUptime, err := time.ParseDuration(helpers.CurlApp(Config, appName, "/uptime"))
				Expect(err).ToNot(HaveOccurred())
				return currentUptime.Seconds()
			}, Config.CcClockCycleDuration(), "1s").Should(BeNumerically(">", originalUptime.Seconds()))

			Consistently(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration(), "5s").Should(ContainSubstring("8080"))
		})
	})
})
