package v3_helpers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func ScaleApp(appGUID string, instances int) {
	scalePath := fmt.Sprintf("/v3/apps/%s/processes/web/actions/scale", appGUID)
	scaleBody := fmt.Sprintf(`{"instances": "%d"}`, instances)
	Expect(cf.Cf("curl", "-f", scalePath, "-X", "POST", "-d", scaleBody).Wait()).To(Exit(0))
}

func WaitForAppToStop(appGUID string) {
	processGUIDS := GetProcessGuidsForType(appGUID, "web")

	Eventually(func() int {
		return GetRunningInstancesStats(processGUIDS[0])
	}, Config.CfPushTimeoutDuration(), time.Second).Should(Equal(0))
}

func CreateApp(appName, spaceGUID, environmentVariables string) string {
	session := cf.Cf("curl", "-f", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s}`, appName, spaceGUID, environmentVariables))
	bytes := session.Wait().Out.Contents()
	var app struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &app)
	Expect(err).NotTo(HaveOccurred())
	return app.GUID
}

func GetAppGUID(appName string) string {
	session := cf.Cf("app", appName, "--guid")
	Expect(session.Wait()).To(Exit(0))
	return strings.TrimSpace(string(session.Out.Contents()))
}

func CreateDockerApp(appName, spaceGUID, environmentVariables string) string {
	session := cf.Cf("curl", "-f", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s, "lifecycle": {"type": "docker", "data": {} } }`, appName, spaceGUID, environmentVariables))
	bytes := session.Wait().Out.Contents()
	var app struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &app)
	Expect(err).NotTo(HaveOccurred())
	return app.GUID
}

func StartApp(appGUID string) {
	startURL := fmt.Sprintf("/v3/apps/%s/actions/start", appGUID)
	Expect(cf.Cf("curl", "-f", startURL, "-X", "POST").Wait()).To(Exit(0))
}

func StopApp(appGUID string) {
	stopURL := fmt.Sprintf("/v3/apps/%s/actions/stop", appGUID)
	Expect(cf.Cf("curl", "-f", stopURL, "-X", "POST").Wait()).To(Exit(0))
}

func RestartApp(appGUID string) {
	restartURL := fmt.Sprintf("/v3/apps/%s/actions/restart", appGUID)
	Expect(cf.Cf("curl", "-f", restartURL, "-X", "POST").Wait()).To(Exit(0))
}

func DeleteApp(appGUID string) {
	HandleAsyncRequest(fmt.Sprintf("/v3/apps/%s", appGUID), "DELETE")
}
