package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("nginx config logic", func() {
	Describe("hitting /v3/packages/:guid/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/packages/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})

	Describe("hitting /v2/apps/:guid/bits with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v2/apps/literally-any-guid/bits?application_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})

	Describe("hitting /v2/buildpacks/:guid/bits with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v2/buildpacks/literally-any-guid/bits?buildpack_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})

	Describe("hitting /v2/apps/:guid/droplet/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v2/apps/literally-any-guid/droplet/upload?droplet_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})
})
