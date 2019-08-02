package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func CreateDockerPackage(appGUID, imagePath string) string {
	packageCreateURL := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", "-f", packageCreateURL, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"docker", "data": {"image": "%s"}}`, appGUID, imagePath))
	bytes := session.Wait().Out.Contents()
	var pac struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &pac)
	Expect(err).NotTo(HaveOccurred())
	return pac.GUID
}

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

func UploadPackage(uploadURL, packageZipPath, token string) {
	bits := fmt.Sprintf(`bits=@%s`, packageZipPath)
	curl := helpers.Curl(Config, "-v", "-s", uploadURL, "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
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
