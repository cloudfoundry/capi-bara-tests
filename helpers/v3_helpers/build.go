package v3_helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func StageBuildpackPackage(packageGUID string, buildpacks ...string) string {
	buildpackString := "null"
	if len(buildpacks) > 0 {
		buildpackString = fmt.Sprintf(`["%s"]`, strings.Join(buildpacks, `", "`))
	}

	stageBody := fmt.Sprintf(`{"lifecycle":{ "type": "buildpack", "data": { "buildpacks": %s } }, "package": { "guid" : "%s"}}`, buildpackString, packageGUID)
	stageURL := "/v3/builds"
	session := cf.Cf("curl", "-f", stageURL, "-X", "POST", "-d", stageBody)
	bytes := session.Wait().Out.Contents()
	var build struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &build)
	Expect(err).NotTo(HaveOccurred())
	Expect(build.GUID).NotTo(BeEmpty())
	return build.GUID
}

func StageDockerPackage(packageGUID string) string {
	stageBody := fmt.Sprintf(`{"lifecycle": { "type" : "docker", "data": {} }, "package": { "guid" : "%s"}}`, packageGUID)
	stageURL := "/v3/builds"
	session := cf.Cf("curl", "-f", stageURL, "-X", "POST", "-d", stageBody)
	bytes := session.Wait().Out.Contents()
	var build struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &build)
	Expect(err).NotTo(HaveOccurred())
	return build.GUID
}

func WaitForBuildToStage(buildGUID string) {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGUID)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", buildPath).Wait()
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}
