package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type Quota struct {
	Name string `json:"name"`
	GUID string `json:"guid"`
	Apps struct {
		TotalInstances int `json:"total_instances"`
	} `json:"apps"`
}

func CreateOrgQuota(name string, orgGUID string, totalInstances int) Quota {
	requestPath := fmt.Sprintf("/v3/organization_quotas")
	requestBody := fmt.Sprintf(`{
  "name": "%s",
  "apps": {
    "total_instances": %v
  },
  "relationships": {
    "organizations": {
      "data": [
        {
          "guid": "%s"
        }
      ]
    }
  }
}`, name, totalInstances, orgGUID)
	session := cf.Cf("curl", "-X", "POST", requestPath, "-d", requestBody, "-f")

	var createdOrgQuota Quota
	response := session.Wait().Out.Contents()
	err := json.Unmarshal(response, &createdOrgQuota)
	Expect(err).ToNot(HaveOccurred())

	return createdOrgQuota
}

func CreateSpaceQuota(name string, spaceGUID string, orgGUID string, totalInstances int) Quota {
	requestPath := fmt.Sprintf("/v3/space_quotas")
	request := fmt.Sprintf(`{
  "name": "%s",
  "apps": {
    "total_instances": %v
  },
  "relationships": {
    "organization": {
      "data": {
        "guid": "%s"
      }
    },
    "spaces": {
      "data": [
        {
          "guid": "%s"
        }
      ]
    }
  }
}`, name, totalInstances, orgGUID, spaceGUID)
	session := cf.Cf("curl", "-X", "POST", requestPath, "-d", request, "-f")

	var createdSpaceQuota Quota
	response := session.Wait().Out.Contents()
	err := json.Unmarshal(response, &createdSpaceQuota)
	Expect(err).ToNot(HaveOccurred())

	return createdSpaceQuota
}

func SetDefaultOrgQuota(orgGUID string) {
	session := cf.Cf("curl", "/v3/organization_quotas?names=default", "-f")
	bytes := session.Wait().Out.Contents()
	defaultOrgQuotaGUID := GetGuidFromResponse(bytes)

	path := fmt.Sprintf("v3/organization_quotas/%s/relationships/organizations", defaultOrgQuotaGUID)
	session = cf.Cf("curl", "-X", "POST", path, "-d", fmt.Sprintf(`{"data": [{"guid": "%s"}]}`, orgGUID), "-f", "-v")

	Eventually(session).Should(Exit(0))
}

func DeleteOrgQuota(orgQuotaGUID string) {
	path := fmt.Sprintf("v3/organization_quotas/%s", orgQuotaGUID)
	session := cf.Cf("curl", "-X", "DELETE", path, "-f", "-v")
	Eventually(session).Should(Exit(0))
}
