package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("cloud native buildpacks", func() {
	SkipOnVMs("no cnbs on VMs")

	It("displays buildpack information", func() {
		session := cf.Cf("buildpacks")
		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("paketo-buildpacks/java"))
	})
})
