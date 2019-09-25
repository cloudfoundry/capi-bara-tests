package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type RevisionList struct {
	Revisions []Revision `json:"resources"`
}

type SidecarList struct {
	Sidecars []Sidecar `json:"resources"`
}

type Sidecar struct {
	Name         string   `json:"name"`
	ProcessTypes []string `json:"process_types"`
	Command      string   `json:"command"`
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

func GetSidecar(sidecarGuid string) Sidecar {
	sidecarEndpoint := fmt.Sprintf("/v3/sidecars/%s", sidecarGuid)
	session := cf.Cf("curl", sidecarEndpoint)
	bytes := session.Wait().Out.Contents()

	var sidecar Sidecar
	err := json.Unmarshal(bytes, &sidecar)
	Expect(err).NotTo(HaveOccurred())

	return sidecar
}

func GetSidecars(appGuid string) []Sidecar {
	sidecarsEndpoint := fmt.Sprintf("/v3/apps/%s/sidecars", appGuid)
	session := cf.Cf("curl", sidecarsEndpoint)
	bytes := session.Wait().Out.Contents()

	var sidecars SidecarList
	err := json.Unmarshal(bytes, &sidecars)
	Expect(err).NotTo(HaveOccurred())

	fmt.Printf("!!!! %s\n\n", string(bytes))
	fmt.Printf("!!!! %#v\n\n", sidecars.Sidecars[0])
	return sidecars.Sidecars
}

func CreateSidecar(appGuid string, sidecar Sidecar) string {
	sidecarEndpoint := fmt.Sprintf("/v3/apps/%s/sidecars", appGuid)
	sidecarOneJSON, err := json.Marshal(sidecar)
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

func DeleteSidecar(sidecarGuid string) {
	sidecarEndpoint := fmt.Sprintf("/v3/sidecars/%s", sidecarGuid)
	session := cf.Cf("curl", "-f", sidecarEndpoint, "-X", "DELETE")
	Eventually(session).Should(Exit(0))
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
