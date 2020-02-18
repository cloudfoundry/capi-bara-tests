package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
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
	session := cf.Cf("curl", "-X", "POST", requestPath, "-d", requestBody)

	var createdOrgQuota Quota
	response := session.Wait().Out.Contents()
	err := json.Unmarshal(response, &createdOrgQuota)
	Expect(err).ToNot(HaveOccurred())

	return createdOrgQuota
}

func CreateSpaceQuota(name string, spaceGUID string, orgGUID string, totalInstances int) Quota {
	requestPath := fmt.Sprintf("/v3/space_quotas")
	session := cf.Cf("curl", "-X", "POST", requestPath, "-d", fmt.Sprintf(`{
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
}`, name, totalInstances, orgGUID, spaceGUID))

	var createdSpaceQuota Quota
	response := session.Wait().Out.Contents()
	err := json.Unmarshal(response, &createdSpaceQuota)
	Expect(err).ToNot(HaveOccurred())

	return createdSpaceQuota
}

func SetDefaultOrgQuota(orgGUID string) {
	session := cf.Cf("curl", "-f", "/v3/organization_quotas?names=default")
	bytes := session.Wait().Out.Contents()
	defaultOrgQuotaGUID := GetGuidFromResponse(bytes)

	path := fmt.Sprintf("v3/organization_quotas/%s/relationships/organizations", defaultOrgQuotaGUID)
	session = cf.Cf("curl", "-X", "POST", path, "-d", fmt.Sprintf(`{"data": [{"guid": "%s"}]}`, orgGUID))

	Eventually(session).Should(Exit(0))
}

func DeleteOrgQuota(orgQuotaGUID string) {
	path := fmt.Sprintf("v3/organization_quotas/%s", orgQuotaGUID)
	session := cf.Cf("curl", "-X", "DELETE", path, "-v")
	Eventually(session).Should(Say("HTTP/1.1 202 Accepted"))
	Eventually(session).Should(Exit(0))
}
