package baras

import (
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

var _ = Describe("Quotas", func() {
	var (
		orgName    string
		spaceName  string
		spaceGUID  string
		orgGUID    string
		appName    string
		spaceQuota Quota
		orgQuota   Quota
	)

	BeforeEach(func() {
		orgName = "org-with-quota"
		spaceName = "space-with-quota"

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			session := cf.Cf("create-org", orgName)
			Eventually(session).Should(Exit(0))
			orgGUID = GetOrgGUIDFromName(orgName)

			orgQuotaName := random_name.BARARandomName("ORG-QUOTA")
			orgQuota = CreateOrgQuota(orgQuotaName, orgGUID, 2)
			Expect(orgQuota.Apps.TotalInstances).To(Equal(2))

			session = cf.Cf("create-space", spaceName, "-o", orgName)
			Eventually(session).Should(Exit(0))
			spaceGUID = GetSpaceGuidFromName(spaceName)

			spaceQuotaName := random_name.BARARandomName("SPACE-QUOTA")
			spaceQuota = CreateSpaceQuota(spaceQuotaName, spaceGUID, orgGUID, 1)
			Expect(spaceQuota.Apps.TotalInstances).To(Equal(1))

			session = cf.Cf("target", "-o", orgName, "-s", spaceName)
			Eventually(session).Should(Exit(0))

			appName = random_name.BARARandomName("APP")
			Expect(cf.Cf("push",
				appName,
				"-b", Config.GetGoBuildpackName(),
				"-p", assets.NewAssets().CatnipZip,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			session := cf.Cf("delete-org", orgName, "-f")
			Eventually(session).Should(Exit(0))

			DeleteOrgQuota(orgQuota.GUID)
		})
	})

	It("respects space and org quota limits", func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			session := cf.Cf("target", "-o", orgName, "-s", spaceName)
			Eventually(session).Should(Exit(0))

			session = cf.Cf("scale", appName, "-i", "2")
			Eventually(session.Err).Should(Say("app_instance_limit space_app_instance_limit_exceeded"))
			Eventually(session).Should(Exit(1))

			path := fmt.Sprintf("v3/space_quotas/%s/relationships/spaces/%s/", spaceQuota.GUID, spaceGUID)
			session = cf.Cf("curl", "-X", "DELETE", path, "-f", "-v")
			Eventually(session).Should(Exit(0))

			path = fmt.Sprintf("v3/space_quotas/%s", spaceQuota.GUID)
			session = cf.Cf("curl", "-X", "DELETE", path, "-f", "-v")
			Eventually(session).Should(Exit(0))

			session = cf.Cf("scale", appName, "-i", "2")
			Eventually(session).Should(Exit(0))

			session = cf.Cf("scale", appName, "-i", "3")
			Eventually(session.Err).Should(Say("app_instance_limit app_instance_limit_exceeded"))
			Eventually(session).Should(Exit(1))

			SetDefaultOrgQuota(orgGUID)

			session = cf.Cf("scale", appName, "-i", "3")
			Eventually(session).Should(Exit(0))
		})
	})
})
