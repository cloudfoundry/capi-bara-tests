package v3_helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	FAILED string = "FAILED"
	STAGED string = "STAGED"
)

func StagePackage(packageGUID string, lifecycle string, buildpacks ...string) string {
	buildpackString := "null"
	if len(buildpacks) > 0 {
		buildpackString = fmt.Sprintf(`["%s"]`, strings.Join(buildpacks, `", "`))
	}

	stageBody := fmt.Sprintf(
		`{"lifecycle":{ "type": "%s", "data": { "buildpacks": %s } }, "package": { "guid": "%s"}}`,
		lifecycle, buildpackString, packageGUID,
	)
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

func GetBuildError(buildGUID string) string {
	buildPath := buildPath(buildGUID)

	session := cf.Cf("curl", "-f", buildPath).Wait()
	bytes := session.Wait().Out.Contents()

	var build struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(bytes, &build)
	Expect(err).NotTo(HaveOccurred())

	return build.Error
}

func WaitForBuildToStage(buildGUID string) {
	buildPath := buildPath(buildGUID)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", buildPath).Wait()
		Expect(session).NotTo(Say(FAILED))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say(STAGED))
}

func WaitForBuildToFail(buildGUID string) {
	buildPath := buildPath(buildGUID)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", buildPath).Wait()
		Expect(session).NotTo(Say(STAGED))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say(FAILED))
}

func buildPath(buildGUID string) string {
	return fmt.Sprintf("/v3/builds/%s", buildGUID)
}
