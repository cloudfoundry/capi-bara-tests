package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func CreatePackage(appGUID string) string {
	packageCreateURL := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", "-f", packageCreateURL, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"bits"}`, appGUID))
	bytes := session.Wait().Out.Contents()
	var pac struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &pac)
	Expect(err).NotTo(HaveOccurred())
	return pac.GUID
}

func UploadPackage(uploadURL, packageZipPath string) {
	bits := fmt.Sprintf(`bits=@%s`, packageZipPath)
	curl := helpers.Curl(Config, "--http1.1", "-v", "-s", "-f", "--show-error", uploadURL, "-F", bits, "-H", fmt.Sprintf("Authorization: %s", GetAuthToken())).Wait(Config.CfPushTimeoutDuration())
	Expect(curl).To(Exit(0))
}

func WaitForPackageToBeReady(packageGUID string) {
	pkgURL := fmt.Sprintf("/v3/packages/%s", packageGUID)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", pkgURL)
		Expect(session.Wait()).To(Exit(0))
		return session
	}, Config.LongCurlTimeoutDuration()).Should(Say("READY"))
}
