package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func AssignIsolationSegmentToSpace(spaceGUID, isoSegGUID string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGUID),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGUID)),
	).Should(Exit(0))
}

func CreateIsolationSegment(name string) string {
	session := cf.Cf("curl", "-f", "/v3/isolation_segments", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s"}`, name))
	bytes := session.Wait().Out.Contents()

	var isolationSegment struct {
		GUID string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &isolationSegment)
	Expect(err).ToNot(HaveOccurred())

	return isolationSegment.GUID
}

func CreateOrGetIsolationSegment(name string) string {
	isoSegGUID := CreateIsolationSegment(name)
	if isoSegGUID == "" {
		isoSegGUID = GetIsolationSegmentGuid(name)
	}
	return isoSegGUID
}

func DeleteIsolationSegment(guid string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments/%s", guid), "-X", "DELETE")).Should(Exit(0))
}

func EntitleOrgToIsolationSegment(orgGUID, isoSegGUID string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations", isoSegGUID),
		"-X",
		"POST",
		"-d",
		fmt.Sprintf(`{"data":[{ "guid":"%s" }]}`, orgGUID)),
	).Should(Exit(0))
}

func GetDefaultIsolationSegment(orgGUID string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGUID))
	bytes := session.Wait().Out.Contents()
	return GetIsolationSegmentGuidFromResponse(bytes)
}

func GetIsolationSegmentGuid(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

func GetIsolationSegmentGuidFromResponse(response []byte) string {
	type data struct {
		GUID string `json:"guid"`
	}
	var GetResponse struct {
		Data data `json:"data"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if (data{}) == GetResponse.Data {
		return ""
	}

	return GetResponse.Data.GUID
}

func IsolationSegmentExists(name string) bool {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait().Out.Contents()
	type resource struct {
		GUID string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func OrgEntitledToIsolationSegment(orgGUID string, isoSegName string) bool {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s&organization_guids=%s", isoSegName, orgGUID))
	bytes := session.Wait().Out.Contents()

	type resource struct {
		GUID string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func SetDefaultIsolationSegment(orgGUID, isoSegGUID string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGUID),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGUID)),
	).Should(Exit(0))
}

func UnassignIsolationSegmentFromSpace(spaceGUID string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGUID),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid":null}}`),
	).Should(Exit(0))
}

func UnsetDefaultIsolationSegment(orgGUID string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGUID),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid": null}}`),
	).Should(Exit(0))
}

func RevokeOrgEntitlementForIsolationSegment(orgGUID, isoSegGUID string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations/%s", isoSegGUID, orgGUID),
		"-X",
		"DELETE",
	)).Should(Exit(0))
}
