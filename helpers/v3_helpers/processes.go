package v3_helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type ProcessList struct {
	Processes []Process `json:"resources"`
}

type Process struct {
	Guid          string `json:"guid"`
	Type          string `json:"type"`
	Command       string `json:"command"`
	Name          string `json:"-"`
	Relationships struct {
		Revision struct {
			Data struct {
				Guid string `json:"guid"`
			} `json:"data"`
		} `json:"revision"`
	} `json:"relationships"`
}

func GetProcesses(appGUID, appName string) []Process {
	processesURL := fmt.Sprintf("/v3/apps/%s/processes", appGUID)
	session := cf.Cf("curl", processesURL)
	bytes := session.Wait().Out.Contents()

	processes := ProcessList{}
	err := json.Unmarshal(bytes, &processes)
	Expect(err).NotTo(HaveOccurred())

	for i, _ := range processes.Processes {
		processes.Processes[i].Name = appName
	}

	return processes.Processes
}

func GetFirstProcessByType(processes []Process, processType string) Process {
	for _, process := range processes {
		if process.Type == processType {
			return process
		}
	}
	return Process{}
}

func GetProcessByGuid(processGUID string) Process {
	processURL := fmt.Sprintf("/v3/processes/%s", processGUID)
	session := cf.Cf("curl", processURL)
	bytes := session.Wait().Out.Contents()

	var process Process
	err := json.Unmarshal(bytes, &process)
	Expect(err).NotTo(HaveOccurred())

	return process
}

func SetCommandOnProcess(appGUID, processType, command string) {
	process := GetFirstProcessByType(GetProcesses(appGUID, "appName"), processType)

	processURL := fmt.Sprintf("/v3/processes/%s", process.Guid)
	processJSON, _ := json.Marshal(map[string]string{"command": command})

	session := cf.Cf("curl", "-f", "-v", "-X", "PATCH", processURL, "-d", string(processJSON)).Wait()
	Expect(session).To(Say("200 OK"))
}

func SetHealthCheckTimeoutOnProcess(appGUID, processType string, healthCheckTimeout int) {
	process := GetFirstProcessByType(GetProcesses(appGUID, "appName"), processType)

	type processHealthCheck struct {
		HealthCheck struct {
			Data struct {
				Timeout int `json:"timeout"`
			} `json:"data"`
		} `json:"health_check"`
	}

	processUpdate := &processHealthCheck{}
	processUpdate.HealthCheck.Data.Timeout = healthCheckTimeout
	processURL := fmt.Sprintf("/v3/processes/%s", process.Guid)
	processJSON, _ := json.Marshal(&processUpdate)

	session := cf.Cf("curl", "-f", "-v", "-X", "PATCH", processURL, "-d", string(processJSON)).Wait()
	Expect(session).To(Say("200 OK"))
}

func GetProcessGuidsForType(appGUID string, processType string) []string {
	processesPath := fmt.Sprintf("/v3/apps/%s/processes?types=%s", appGUID, processType)
	session := cf.Cf("curl", "-f", processesPath).Wait()

	processesJSON := struct {
		Resources []struct {
			Guid string `json:"guid"`
		} `json:"resources"`
	}{}
	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &processesJSON)

	guids := []string{}
	if err != nil || len(processesJSON.Resources) == 0 {
		return guids
	}

	for _, resource := range processesJSON.Resources {
		guids = append(guids, resource.Guid)
	}

	return guids
}

func ScaleProcess(appGUID, processType, memoryInMb string) {
	scalePath := fmt.Sprintf("/v3/apps/%s/processes/%s/actions/scale", appGUID, processType)
	scaleBody := fmt.Sprintf(`{"memory_in_mb":"%s"}`, memoryInMb)
	session := cf.Cf("curl", "-f", scalePath, "-X", "POST", "-d", scaleBody).Wait()
	Expect(session).To(Exit(0))
	result := session.Out.Contents()
	Expect(strings.Contains(string(result), "errors")).To(BeFalse())
}

type ProcessAppUsageEvent struct {
	Metadata struct {
		Guid string `json:"guid"`
	} `json:"metadata"`
	Entity struct {
		ProcessType string `json:"process_type"`
		State       string `json:"state"`
	} `json:"entity"`
}

type ProcessAppUsageEvents struct {
	Resources []ProcessAppUsageEvent `struct:"resources"`
}

func GetLastAppUseEventForProcess(processType string, state string, afterGUID string) (bool, ProcessAppUsageEvent) {
	var response ProcessAppUsageEvents
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		afterGUIDParam := ""
		if afterGUID != "" {
			afterGUIDParam = fmt.Sprintf("&after_guid=%s", afterGUID)
		}
		usageEventsUrl := fmt.Sprintf("/v2/app_usage_events?order-direction=desc&page=1&results-per-page=150%s", afterGUIDParam)
		workflowhelpers.ApiRequest("GET", usageEventsUrl, &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.Entity.ProcessType == processType && event.Entity.State == state {
			return true, event
		}
	}

	return false, ProcessAppUsageEvent{}
}
