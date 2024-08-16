package v3_helpers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func CreateDeployment(appGUID, strategy string, max_in_flight int) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"strategy": "%s", "options": {"max_in_flight": %d }, "relationships": {"app": {"data": {"guid": "%s"}}}}`, strategy, max_in_flight, appGUID)
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

func CreateDeploymentForDroplet(appGUID, dropletGUID, strategy string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"strategy": "%s", "droplet": {"guid": "%s"}, "relationships": {"app": {"data": {"guid": "%s"}}}}`, strategy, dropletGUID, appGUID)
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

func ContinueDeployment(deploymentGUID string) {
	deploymentPath := fmt.Sprintf("/v3/deployments/%s/actions/continue", deploymentGUID)
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
	processStatsURL := fmt.Sprintf("%s%s/v3/processes/%s/stats", Config.Protocol(), Config.GetApiEndpoint(), processGUID)

	client := buildHTTPClient()
	req, err := http.NewRequest("GET", processStatsURL, nil)
	req.Header.Add("Authorization", GetAuthToken())
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())

	instancesJSON := struct {
		Resources []struct {
			Type  string `json:"type"`
			State string `json:"state"`
		} `json:"resources"`
	}{}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(200))

	err = json.Unmarshal(body, &instancesJSON)
	Expect(err).NotTo(HaveOccurred())
	numRunning := 0

	for _, instance := range instancesJSON.Resources {
		if instance.State == "RUNNING" {
			numRunning += 1
		}
	}
	return numRunning
}

func buildHTTPClient() *http.Client {
	var client *http.Client
	if Config.GetSkipSSLValidation() {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}
	return client
}
