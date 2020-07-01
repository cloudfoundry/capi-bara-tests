package baras

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("cloud native buildpacks", func() {
	BeforeEach(func() {
		if !Config.GetIncludeKpack() {
			Skip(skip_messages.SkipKpackMessage)
		}
	})

	It("displays buildpack information", func() {
		session := cf.Cf("buildpacks")
		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("paketo-buildpacks/java"))
	})
})
