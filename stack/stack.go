package stack

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/k8s_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Stack", func() {
	SkipOnVMs("on VMs, this is orchestrated with BOSH magic")
	const (
		defaultStack = "clusterstacks/bionic-stack"
	)
	var (
		appName            string
		appGUID            string
		dropletGUID        string
		dropletImage       string
		originalStackImage string
	)
	type RunImage struct {
		Image string `json:"image"`
	}
	type Spec struct {
		RunImage RunImage `json:"runImage"`
	}
	type Stack struct {
		Spec Spec `json:"spec"`
	}

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))

		By("Pushing an app")
		session = cf.Cf("push", appName, "-p", "../"+assets.NewAssets().Catnip)
		Expect(session.Wait("3m")).To(gexec.Exit(0))
		appGUID = GetAppGUID(appName)
		dropletGUID = GetDropletFromApp(appGUID)
		dropletImage = GetDroplet(dropletGUID).Image

		By("Updating the stack")
		session = Kubectl("get", defaultStack, "-o", "json")
		Expect(session.Wait("1m")).To(gexec.Exit(0))
		originalStack := &Stack{}
		err := json.Unmarshal(session.Out.Contents(), originalStack)
		Expect(err).NotTo(HaveOccurred())
		originalStackImage = originalStack.Spec.RunImage.Image
		session = Kubectl("patch", defaultStack, "--type=merge", "-p", `{"spec":{"runImage":{"image":"index.docker.io/paketobuildpacks/run:0.0.74-full-cnb"}}}`)
		Expect(session.Wait("3m")).To(gexec.Exit(0))
		Expect(session.Out.Contents()).Should(ContainSubstring("clusterstack.kpack.io/bionic-stack patched"))
	})

	AfterEach(func() {
		DeleteApp(appGUID)
		session := Kubectl("patch", defaultStack, "--type=merge", "-p", fmt.Sprintf(`{"spec":{"runImage":{"image":"%s"}}}`, originalStackImage))
		Expect(session.Wait("3m")).To(gexec.Exit(0))
		Expect(session.Out.Contents()).Should(ContainSubstring("clusterstack.kpack.io/bionic-stack patched"))
	})

	Context("When restarting an app with an updated stack", func() {
		It("starts the app successfully and the droplet contains the rebased image reference", func() {
			Eventually(func() string {
				return GetDroplet(dropletGUID).Image
			}, "45s", "1s").ShouldNot(Equal(dropletImage))

			By("Restarting the app")
			Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(gexec.Exit(0))
			restartedAppDroplet := GetDroplet(dropletGUID)
			Expect(restartedAppDroplet.Image).ToNot(Equal(dropletImage))

			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
		})
	})
})
