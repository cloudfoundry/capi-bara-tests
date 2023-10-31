package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func AssignDropletToApp(appGUID, dropletGUID string) {
	appUpdatePath := fmt.Sprintf("/v3/apps/%s/relationships/current_droplet", appGUID)
	appUpdateBody := fmt.Sprintf(`{"data": {"guid":"%s"}}`, dropletGUID)
	Expect(cf.Cf("curl", "-f", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait()).To(Exit(0))

	for _, process := range GetProcesses(appGUID, "") {
		ScaleProcess(appGUID, process.Type, V3_DEFAULT_MEMORY_LIMIT)
	}
}

func CreateAndAssociateNewDroplet(appGUID, assetPath string, buildpacks ...string) string {
	By("Creating a Package")
	packageGUID := CreatePackage(appGUID)
	uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

	By("Uploading a Package")
	UploadPackage(uploadURL, assetPath)
	WaitForPackageToBeReady(packageGUID)

	By("Creating a Build")
	buildGUID := StagePackage(packageGUID, Config.Lifecycle(), buildpacks...)
	WaitForBuildToStage(buildGUID)
	dropletGUID := GetDropletFromBuild(buildGUID)

	AssignDropletToApp(appGUID, dropletGUID)

	return dropletGUID
}

func GetDropletFromBuild(buildGUID string) string {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGUID)
	session := cf.Cf("curl", "-f", buildPath).Wait()
	var build struct {
		Droplet struct {
			GUID string `json:"guid"`
		} `json:"droplet"`
	}
	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &build)
	Expect(err).NotTo(HaveOccurred())
	Expect(build.Droplet.GUID).NotTo(BeEmpty(), "Build response didn't contain a droplet GUID")
	return build.Droplet.GUID
}

func GetDropletFromApp(appGUID string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/apps/%s/droplets/current", appGUID)).Wait()
	var droplet struct {
		GUID string `json:"guid"`
	}
	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &droplet)
	Expect(err).NotTo(HaveOccurred())
	return droplet.GUID
}

type Droplet struct {
	GUID      string `json:"guid"`
	State     string `json:"state"`
	Image     string `json:"image"`
	Lifecycle struct {
		Type string   `json:"type"`
		Data struct{} `json:"data"`
	} `json:"lifecycle"`
}

func GetDroplet(dropletGUID string) Droplet {
	dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGUID)
	session := cf.Cf("curl", "-f", dropletPath)
	bytes := session.Wait().Out.Contents()

	var droplet = Droplet{}
	err := json.Unmarshal(bytes, &droplet)
	Expect(err).NotTo(HaveOccurred())
	return droplet
}

func WaitForDropletToCopy(dropletGUID string) {
	dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGUID)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", dropletPath).Wait()
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}
