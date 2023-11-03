package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type RevisionList struct {
	Revisions []Revision `json:"resources"`
}

type Sidecar struct {
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	ProcessTypes []string `json:"process_types"`
	MemoryInMb   int      `json:"memory_in_mb"`
}

type Revision struct {
	Guid    string `json:"guid"`
	Version int    `json:"version"`
	Droplet struct {
		Guid string `json:"guid"`
	} `json:"droplet"`
	Processes map[string]map[string]string `json:"processes"`
	Sidecars  []Sidecar                    `json: "sidecars"`
}

type RevisionEnvVars struct {
	Var map[string]string `json:"var"`
}

func GetRevisions(appGuid string) []Revision {
	revisionsURL := fmt.Sprintf("/v3/apps/%s/revisions", appGuid)
	session := cf.Cf("curl", revisionsURL)
	bytes := session.Wait().Out.Contents()

	revisions := RevisionList{}
	err := json.Unmarshal(bytes, &revisions)
	Expect(err).NotTo(HaveOccurred())

	return revisions.Revisions
}

func GetRevision(revisionGuid string) Revision {
	revisionsURL := fmt.Sprintf("/v3/revisions/%s", revisionGuid)
	session := cf.Cf("curl", revisionsURL)
	bytes := session.Wait().Out.Contents()

	revision := Revision{}
	err := json.Unmarshal(bytes, &revision)
	Expect(err).NotTo(HaveOccurred())

	return revision
}

func GetNewestRevision(appGuid string) Revision {
	revisions := GetRevisions(appGuid)
	return revisions[len(revisions)-1]
}

func GetRevisionEnvVars(revisionGuid string) RevisionEnvVars {
	revisionsEnvVarsURL := fmt.Sprintf("/v3/revisions/%s/environment_variables", revisionGuid)
	session := cf.Cf("curl", revisionsEnvVarsURL)
	bytes := session.Wait().Out.Contents()

	envVars := RevisionEnvVars{}
	err := json.Unmarshal(bytes, &envVars)
	Expect(err).NotTo(HaveOccurred())

	return envVars
}

func EnableRevisions(appGuid string) {
	path := fmt.Sprintf("/v3/apps/%s/features/revisions", appGuid)
	curl := cf.Cf("curl", "-f", path, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()
	Expect(curl).To(Exit(0))
}
