package stack

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/k8s_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"encoding/json"
	"fmt"
)

var _ = Describe("Stack", func() {
	var (
		appName             string
		appGUID             string
		dropletGUID         string
		dropletImage        string
		originalStackImage       string
	)
	type RunImage struct{
		Image string `json:"image"`
	}
	type Spec struct {
		RunImage RunImage `json:"runImage"`
	}
	type Stack struct {
		Spec Spec `json:"spec"`
	}
	BeforeEach(func() {
		if !Config.GetIncludeKpack() {
			Skip(skip_messages.SkipKpackMessage)
		}

		appName = random_name.BARARandomName("APP")
		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))

		By("Pushing an app")
		session = cf.Cf("push", appName, "-p", "../" + assets.NewAssets().Catnip)
		Expect(session.Wait("3m")).To(gexec.Exit(0))
		appGUID = GetAppGUID(appName)
		dropletGUID = GetDropletFromApp(appGUID)
		dropletImage = GetDroplet(dropletGUID).Image

		By("Updating the stack")
		bytes, err := Kubectl("get", "stack/cflinuxfs3-stack", "-o", "json")
		Expect(err).ToNot(HaveOccurred())
		var originalStack Stack
		json.Unmarshal(bytes, &originalStack)
		originalStackImage = originalStack.Spec.RunImage.Image
		output, err := Kubectl("patch", "stack/cflinuxfs3-stack", "--type=merge", "-p", `{"spec":{"runImage":{"image":"gcr.io/paketo-buildpacks/run:0.0.50-full-cnb-cf"}}}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(ContainSubstring("stack.experimental.kpack.pivotal.io/cflinuxfs3-stack patched"))
	})

	AfterEach(func() {
		DeleteApp(appGUID)
		output, err := Kubectl("patch", "stack/cflinuxfs3-stack", "--type=merge", "-p", fmt.Sprintf(`{"spec":{"runImage":{"image":"%s"}}}`, originalStackImage))
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(ContainSubstring("stack.experimental.kpack.pivotal.io/cflinuxfs3-stack patched"))
	})

	Context("When restarting an app with an updated stack", func() {
		It("starts the app successfully and the droplet contains the rebased image reference", func() {
			Eventually(func() string {
				return GetDroplet(dropletGUID).Image
			}, "15s", "1s").ShouldNot(Equal(dropletImage))

			By("Restarting the app")
			Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(gexec.Exit(0))
			restartedAppDroplet := GetDroplet(dropletGUID)
			Expect(restartedAppDroplet.Image).ToNot(Equal(dropletImage))

			Expect(helpers.CurlAppRoot(Config, appName)).To(Equal("Catnip?"))
		})
	})
})
