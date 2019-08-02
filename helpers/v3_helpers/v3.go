package v3_helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const (
	V3_DEFAULT_MEMORY_LIMIT = "256"
	V3_JAVA_MEMORY_LIMIT    = "1024"
)

func CreateSidecar(name string, processTypes []string, command string, memoryLimit int, appGuid string) string {
	sidecarEndpoint := fmt.Sprintf("/v3/apps/%s/sidecars", appGuid)
	sidecarOneJSON, err := json.Marshal(
		struct {
			Name         string   `json:"name"`
			Command      string   `json:"command"`
			ProcessTypes []string `json:"process_types"`
			Memory       int      `json:"memory_in_mb"`
		}{
			name,
			command,
			processTypes,
			memoryLimit,
		},
	)
	Expect(err).NotTo(HaveOccurred())
	session := cf.Cf("curl", "-f", sidecarEndpoint, "-X", "POST", "-d", string(sidecarOneJSON))
	Eventually(session).Should(Exit(0))

	var sidecarData struct {
		Guid string `json:"guid"`
	}
	err = json.Unmarshal(session.Out.Contents(), &sidecarData)
	Expect(err).NotTo(HaveOccurred())
	return sidecarData.Guid
}

func UpdateEnvironmentVariables(appGUID, envVars string) {
	appUpdatePath := fmt.Sprintf("/v3/apps/%s/environment_variables", appGUID)
	appUpdateBody := fmt.Sprintf(`{"var": %s}`, envVars)
	Expect(cf.Cf("curl", "-f", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait()).To(Exit(0))
}

func HandleAsyncRequest(path string, method string) {
	session := cf.Cf("curl", "-f", path, "-X", method, "-i")
	bytes := session.Wait().Out.Contents()
	Expect(string(bytes)).To(ContainSubstring("202 Accepted"))

	jobPath := GetJobPath(bytes)
	PollJob(jobPath)
}

func FetchRecentLogs(appGuid, oauthToken string, config config.BaraConfig) *Session {
	loggregatorEndpoint := getHttpLoggregatorEndpoint()
	logURL := fmt.Sprintf("%s/apps/%s/recentlogs", loggregatorEndpoint, appGuid)
	session := helpers.Curl(Config, logURL, "-H", fmt.Sprintf("Authorization: %s", oauthToken))
	Expect(session.Wait()).To(Exit(0))
	return session
}

func GetGuidFromResponse(response []byte) string {
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if len(GetResponse.Resources) == 0 {
		Fail("No guid found for response")
	}

	return GetResponse.Resources[0].Guid
}

func GetSpaceGuidFromName(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

func GetDomainGUIDFromName(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/domains?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

//private

func getHttpLoggregatorEndpoint() string {
	infoCommand := cf.Cf("curl", "-f", "/v2/info")
	Expect(infoCommand.Wait()).To(Exit(0))

	var response struct {
		DopplerLoggingEndpoint string `json:"doppler_logging_endpoint"`
	}

	err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	return strings.Replace(response.DopplerLoggingEndpoint, "ws", "http", 1)
}
