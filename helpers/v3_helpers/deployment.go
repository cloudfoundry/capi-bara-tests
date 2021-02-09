package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func CreateDeployment(appGUID string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"relationships": {"app": {"data": {"guid": "%s"}}}}`, appGUID)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		GUID string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &deployment)
	Expect(err).NotTo(HaveOccurred())
	return deployment.GUID
}

func CreateDeploymentForDroplet(appGUID, dropletGUID string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"droplet": {"guid": "%s"}, "relationships": {"app": {"data": {"guid": "%s"}}}}`, dropletGUID, appGUID)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		GUID string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &deployment)
	Expect(err).NotTo(HaveOccurred())
	return deployment.GUID
}

func RollbackDeployment(appGUID, revisionGUID string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"revision": { "guid": "%s" },"relationships": {"app": {"data": {"guid": "%s"}}}}`, revisionGUID, appGUID)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		GUID string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &deployment)
	Expect(err).NotTo(HaveOccurred())
	return deployment.GUID
}

func CancelDeployment(deploymentGUID string) {
	deploymentPath := fmt.Sprintf("/v3/deployments/%s/actions/cancel", deploymentGUID)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-i").Wait()
	Expect(session.Out.Contents()).To(ContainSubstring("200 OK"))
	Expect(session).To(Exit(0))
}

func WaitUntilDeploymentReachesStatus(deploymentGUID, statusValue, statusReason string) {
	deploymentPath := fmt.Sprintf("/v3/deployments/%s", deploymentGUID)

	type deploymentStatus struct {
		Value  string `json:"value"`
		Reason string `json:"reason"`
	}
	deploymentJSON := struct {
		Status deploymentStatus `json:"status"`
	}{}

	desiredDeploymentStatus := deploymentStatus{
		Value:  statusValue,
		Reason: statusReason,
	}

	Eventually(func() deploymentStatus {
		session := cf.Cf("curl", "-f", deploymentPath).Wait()
		Expect(session.Wait()).To(Exit(0))
		err := json.Unmarshal(session.Out.Contents(), &deploymentJSON)
		Expect(err).NotTo(HaveOccurred())
		return deploymentJSON.Status
	}, Config.LongCurlTimeoutDuration()).Should(Equal(desiredDeploymentStatus))
}

func GetRunningInstancesStats(processGUID string) int {
	processPath := fmt.Sprintf("/v3/processes/%s/stats", processGUID)
	session := cf.Cf("curl", "-f", processPath).Wait()
	instancesJSON := struct {
		Resources []struct {
			Type  string `json:"type"`
			State string `json:"state"`
		} `json:"resources"`
	}{}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &instancesJSON)
	Expect(err).NotTo(HaveOccurred())
	numRunning := 0

	for _, instance := range instancesJSON.Resources {
		if instance.State == "RUNNING" {
			numRunning += 1
		}
	}
	return numRunning
}

func GetAppInstancesStatsMemory(appGUID string) int {
	processPath := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGUID)
	session := cf.Cf("curl", "-f", processPath).Wait()
	instancesJSON := struct {
		Resources []struct {
			MemoryQuota int `json:"mem_quota"`
		} `json:"resources"`
	}{}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &instancesJSON)
	Expect(err).NotTo(HaveOccurred())
	return instancesJSON.Resources[0].MemoryQuota
}
