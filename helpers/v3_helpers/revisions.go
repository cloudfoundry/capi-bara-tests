package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

type RevisionList struct {
	Revisions []Revision `json:"resources"`
}

type Revision struct {
	Guid    string `json:"guid"`
	Version int    `json:"version"`
	Droplet struct {
		Guid string `json:"guid"`
	} `json:"droplet"`
	Processes             map[string]map[string]string `json:"processes"`
}

type RevisionEnvVars struct {
	Var map[string]string `json:"var"`
}

func GetRevisions(appGuid string) []Revision {
	revisionsURL := fmt.Sprintf("/v3/apps/%s/revisions", appGuid)
	session := cf.Cf("curl", revisionsURL)
	bytes := session.Wait().Out.Contents()

	revisions := RevisionList{}
	json.Unmarshal(bytes, &revisions)

	return revisions.Revisions
}

func GetNewestRevision(appGuid string) Revision {
	revisions := GetRevisions(appGuid)
	return revisions[len(revisions)-1]
}

func GetNewestRevisionEnvVars(revisionGuid string) RevisionEnvVars {
	revisionsEnvVarsURL := fmt.Sprintf("/v3/revisions/%s/environment_variables", revisionGuid)
	session := cf.Cf("curl", revisionsEnvVarsURL)
	bytes := session.Wait().Out.Contents()

	envVars := RevisionEnvVars{}
	json.Unmarshal(bytes, &envVars)

	return envVars
}
