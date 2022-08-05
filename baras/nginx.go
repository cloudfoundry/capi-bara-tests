package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
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

	Describe("hitting /v3/buildpacks/:guid/bits with invalid parameters", func() {
		SkipOnK8s("buildpack upload is not supported on k8s")

		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/buildpacks/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})

	Describe("hitting /v3/droplets/:guid/upload with invalid parameters", func() {
		SkipOnK8s("droplet upload is not supported on k8s")

		It("returns 422 Unprocessable Entity", func() {
			session := cf.Cf("curl", "-X", "POST", "/v3/droplets/literally-any-guid/upload?bits_path='some/path'", "-i")
			Eventually(session).Should(Say("422 Unprocessable Entity"))
		})
	})
})
