package baras

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
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
		appName            string
		appGUID            string
		spaceGUID          string
		spaceName          string
		extraPort          int
		appRoutePrefix     string
		sidecarRoutePrefix string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		extraPort = 8081

		By("Creating an App")
		appGUID = CreateApp(appName, spaceGUID, `{"VAR":"base"}`)
		_ = AssociateNewDroplet(appGUID, assets.NewAssets().DoraZip)
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, GetAuthToken(), Config)
		DeleteApp(appGUID)
	})

	Context("when the app has a sidecar associated with its web process", func() {
		BeforeEach(func() {
			sidecarEndpoint := fmt.Sprintf("/v3/apps/%s/sidecars", appGUID)
			sidecarJSON, err := json.Marshal(
				struct {
					Name         string   `json:"name"`
					Command      string   `json:"command"`
					ProcessTypes []string `json:"process_types"`
				}{
					"anything_you_want",
					fmt.Sprintf("WHAT_AM_I=IM_A_SIDECAR bundle exec rackup config.ru -p %d", extraPort),
					[]string{"web"},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			session := cf.Cf("curl", sidecarEndpoint, "-X", "POST", "-d", string(sidecarJSON))
			Eventually(session).Should(Exit(0))

			appEndpoint := fmt.Sprintf("/v2/apps/%s", appGUID)
			extraPortsJSON, err := json.Marshal(
				struct {
					Ports []int `json:"ports"`
				}{
					[]int{8080, 8081},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			session = cf.Cf("curl", appEndpoint, "-X", "PUT", "-d", string(extraPortsJSON))
			Eventually(session).Should(Exit(0))

			appRoutePrefix = random_name.BARARandomName("ROUTE")
			sidecarRoutePrefix = random_name.BARARandomName("ROUTE")
			CreateAndMapRouteWithPort(appGUID, spaceName, Config.GetAppsDomain(), appRoutePrefix, 8080)
			CreateAndMapRouteWithPort(appGUID, spaceName, Config.GetAppsDomain(), sidecarRoutePrefix, 8081)

			Eventually(session).Should(Exit(0))
		})

		Context("and the app and sidecar are listening on different ports", func() {
			It("and successfully responds on each port", func() {
				session := cf.Cf("start", appName)
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", appRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).ShouldNot(Say("IM_A_SIDECAR"))
				Eventually(session).Should(Exit(0))

				session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix, Config.GetAppsDomain()))
				Eventually(session).Should(Say("IM_A_SIDECAR"))
				Eventually(session).Should(Exit(0))
			})
		})
	})
})
