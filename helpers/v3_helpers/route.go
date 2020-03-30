package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
)

func CreateRoute(spaceGUID, domainGUID, host string) string {
	session := cf.Cf(
		"curl", "-f", "/v3/routes",
		"-X", "POST",
		"-d", fmt.Sprintf(`{
			"host": "%s",
			"relationships": {
				"domain": { "data": { "guid": "%s" } },
				"space": { "data": { "guid": "%s" } }
			}
		}`, host, domainGUID, spaceGUID),
	)
	bytes := session.Wait().Out.Contents()

	var response struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &response)
	Expect(err).NotTo(HaveOccurred())
	return response.GUID
}

func CreateRouteWithPath(spaceGUID, domainGUID, host, path string) string {
	session := cf.Cf(
		"curl", "-f", "/v3/routes",
		"-X", "POST",
		"-d", fmt.Sprintf(`{
			"host": "%s",
			"path": "%s",
			"relationships": {
				"domain": { "data": { "guid": "%s" } },
				"space": { "data": { "guid": "%s" } }
			}
		}`, host, path, domainGUID, spaceGUID),
	)
	bytes := session.Wait().Out.Contents()

	var response struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &response)
	Expect(err).NotTo(HaveOccurred())
	return response.GUID
}

func DeleteRoute(routeGUID string) {
	HandleAsyncRequest(fmt.Sprintf("/v3/routes/%s", routeGUID), "DELETE")
}
