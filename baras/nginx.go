package baras

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"regexp"
)

var _ = Describe("nginx config logic", func() {
	Describe("hitting /v3/packages/:guid/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/packages/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422"))
		})
	})

	Describe("hitting /v3/buildpacks/:guid/bits with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/buildpacks/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422"))
		})
	})

	Describe("hitting /v3/droplets/:guid/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/droplets/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422"))
		})
	})

	Describe("Response headers", func() {
		It("does not contain 'Server: nginx'", func() {
			session := cf.Cf("curl", "/v3/info/usage_summary", "-i")
			Eventually(session).ShouldNot(Say(regexp.QuoteMeta("Server: nginx") + `\/?\d+(\.\d+){0,2}`))
		})
	})
})
