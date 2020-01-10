package baras

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sidecars", func() {
	var (
		appName             string
		appGUID             string
		spaceGUID           string
		domainGUID          string
		spaceName           string
		appRoutePrefix      string
		sidecarRoutePrefix1 string
		sidecarRoutePrefix2 string
		sidecarGUID         string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())

		By("Creating an app")
		appGUID = CreateApp(appName, spaceGUID, `{"WHAT_AM_I":"MOTORCYCLE"}`)
		_ = AssociateNewDroplet(appGUID, assets.NewAssets().DoraZip)
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	Context("when the app has a sidecar associated with its web process", func() {
		BeforeEach(func() {
			CreateSidecar("my_sidecar1", []string{"web"}, fmt.Sprintf("WHAT_AM_I=LEFT_SIDECAR bundle exec rackup config.ru -p %d", 8081), 50, appGUID)
			CreateSidecar("my_sidecar2", []string{"web"}, fmt.Sprintf("WHAT_AM_I=RIGHT_SIDECAR bundle exec rackup config.ru -p %d", 8082), 100, appGUID)

			appEndpoint := fmt.Sprintf("/v2/apps/%s", appGUID)
			extraPortsJSON, err := json.Marshal(
				struct {
					Ports []int `json:"ports"`
				}{
					[]int{8080, 8081, 8082},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			session := cf.Cf("curl", appEndpoint, "-X", "PUT", "-d", string(extraPortsJSON))
			Eventually(session).Should(Exit(0))

			appRoutePrefix = random_name.BARARandomName("ROUTE")
			sidecarRoutePrefix1 = random_name.BARARandomName("ROUTE")
			sidecarRoutePrefix2 = random_name.BARARandomName("ROUTE")

			CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, appRoutePrefix, 8080)
			CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, sidecarRoutePrefix1, 8081)
			CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, sidecarRoutePrefix2, 8082)

			Eventually(session).Should(Exit(0))
		})

		Context("and the app and sidecar are listening on different ports", func() {
			It("and successfully responds on each port", func() {
				session := cf.Cf("start", appName)
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).Should(Say("Hi, I'm Dora!"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).ShouldNot(Say("MOTORCYCLE"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix1, Config.GetAppsDomain()))
				Eventually(session).Should(Say("LEFT_SIDECAR"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix2, Config.GetAppsDomain()))
				Eventually(session).Should(Say("RIGHT_SIDECAR"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/MEMORY_LIMIT", sidecarRoutePrefix1, Config.GetAppsDomain()))
				Eventually(session).Should(Say("50m"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/MEMORY_LIMIT", sidecarRoutePrefix2, Config.GetAppsDomain()))
				Eventually(session).Should(Say("100m"))
				Eventually(session).Should(Exit(0))
			})
		})

		Context("when the app has a sidecar that just sleeps", func() {
			BeforeEach(func() {
				sidecarGUID = CreateSidecar("my_sidecar", []string{"web"}, "sleep 100000", 50, appGUID)
			})

			It("stops responding only after an app restart", func() {
				session := cf.Cf("start", appName)
				Eventually(session).Should(Exit(0))

				By("verify the sidecar is running")
				session = cf.Cf("ssh", appName, "-c", "ps aux | grep sleep | grep -v grep")
				Eventually(session).Should(Exit(0))

				By("deleted the sidecar")
				session = cf.Cf("curl", fmt.Sprintf("/v3/sidecars/%s", sidecarGUID), "-X", "DELETE")
				Eventually(session).Should(Exit(0))

				By("verify it still responds")
				session = cf.Cf("ssh", appName, "-c", "ps aux | grep sleep | grep -v grep")
				Eventually(session).Should(Exit(0))

				restartApp(appName)

				By("verify it no longer responds")
				session = cf.Cf("ssh", appName, "-c", "ps aux | grep sleep | grep -v grep")
				Eventually(session).Should(Exit(1))

			})
		})

		Context("and a sidecar is crashing", func() {
			It("crashes the main app/second sidecar and Diego brings it back", func() {
				session := cf.Cf("start", appName)
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).Should(Say("Hi, I'm Dora!"))
				Eventually(session).Should(Exit(0))

				By("Crashing the sidecar process")
				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/sigterm/KILL", sidecarRoutePrefix1, Config.GetAppsDomain()))
				Eventually(session).Should(Say("502"))
				Eventually(session).Should(Exit(0))

				By("Polling the app and sidecar for 404s")
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("404 Not Found: Requested route"))
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s", sidecarRoutePrefix2, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("404 Not Found: Requested route"))

				By("Polling for the app to be restarted by Diego")
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("Hi, I'm Dora!"))
			})
		})

		Context("and the app is crashing", func() {
			It("crashes the sidecars as well", func() {
				session := cf.Cf("start", appName)
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).Should(Say("Hi, I'm Dora!"))
				Eventually(session).Should(Exit(0))

				By("Crashing the main app process")
				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/sigterm/KILL", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).Should(Say("502"))
				Eventually(session).Should(Exit(0))

				By("Polling both sidecars for 404s")
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s", sidecarRoutePrefix1, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("404 Not Found: Requested route"))
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s", sidecarRoutePrefix2, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("404 Not Found: Requested route"))

				By("Polling for the sidecars to be restarted by Diego")
				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix1, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("LEFT_SIDECAR"))

				Eventually(func() *Session {
					session := helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix2, Config.GetAppsDomain()))
					Eventually(session).Should(Exit(0))
					return session
				}, Config.DefaultTimeoutDuration()).Should(Say("RIGHT_SIDECAR"))
			})
		})
	})

	Context("when the app uses multiple buildpacks, one of which supplies a sidecar", func() {
		var buildpackName string

		BeforeEach(func() {
			buildpackName = random_name.BARARandomName("sleepy-sidecar-buildpack")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("create-buildpack", buildpackName, assets.NewAssets().SleepySidecarBuildpackZip, "99").Wait()).To(Exit(0))
			})

		})

		AfterEach(func() {
			cf.Cf("delete-buildpack", buildpackName)
		})

		Context("using cf push", func() {
			JustBeforeEach(func() {
				session := cf.Cf("push", appName,
					"-p", assets.NewAssets().Binary,
					"-b", buildpackName,
					"-b", Config.GetBinaryBuildpackName())
				Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("runs the sidecar process", func() {
				By("verifying the sidecar is on the app")
				sidecars := GetAppSidecars(appGUID)
				Expect(sidecars).To(HaveLen(1))
				Expect(sidecars[0].Name).To(Equal("sleepy"))
				Expect(sidecars[0].Command).To(Equal("sleep infinity"))
				Expect(sidecars[0].MemoryInMb).To(Equal(10))
				Expect(sidecars[0].ProcessTypes).To(Equal([]string{"web"}))

				By("verify the sidecar is running")
				session := cf.Cf("ssh", appName, "-c", "ps aux | grep sleep | grep -v grep")
				Eventually(session).Should(Exit(0))
			})

		})

		Context("using v3 endpoints", func() {
			JustBeforeEach(func() {
				session := cf.Cf("v3-push", appName,
					"-p", assets.NewAssets().Binary,
					"-b", buildpackName,
					"-b", Config.GetBinaryBuildpackName())
				Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("runs the sidecar process", func() {
				By("verifying the sidecar is on the app")
				sidecars := GetAppSidecars(appGUID)
				Expect(sidecars).To(HaveLen(1))
				Expect(sidecars[0].Name).To(Equal("sleepy"))
				Expect(sidecars[0].Command).To(Equal("sleep infinity"))
				Expect(sidecars[0].ProcessTypes).To(Equal([]string{"web"}))

				By("verify the sidecar is running")
				session := cf.Cf("ssh", appName, "-c", "ps aux | grep sleep | grep -v grep")
				Eventually(session).Should(Exit(0))
			})
		})

	})
})
