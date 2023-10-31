package app_helpers

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
)

type Entity struct {
	AppName       string `json:"app_name"`
	AppGuid       string `json:"app_guid"`
	State         string `json:"state"`
	BuildpackName string `json:"buildpack_name"`
	BuildpackGuid string `json:"buildpack_guid"`
	ParentAppName string `json:"parent_app_name"`
	ParentAppGuid string `json:"parent_app_guid"`
	ProcessType   string `json:"process_type"`
	TaskGuid      string `json:"task_guid"`
}

type Metadata struct {
	Guid string `json:"guid"`
}

type AppUsageEvent struct {
	Entity   `json:"entity"`
	Metadata `json:"metadata"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
	NextUrl   string          `json:"next_url"`
}

func UsageEventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	for _, e := range events {
		found = event.Entity.ParentAppName == e.Entity.ParentAppName &&
			event.Entity.ParentAppGuid == e.Entity.ParentAppGuid &&
			event.Entity.ProcessType == e.Entity.ProcessType &&
			event.Entity.State == e.Entity.State &&
			event.Entity.AppGuid == e.Entity.AppGuid &&
			event.Entity.TaskGuid == e.Entity.TaskGuid
		if found {
			break
		}
	}
	return found
}

func LastAppUsageEventGuid(testSetup *workflowhelpers.ReproducibleTestSuiteSetup) string {
	var response AppUsageEvents

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v2/app_usage_events?results-per-page=1&order-direction=desc&page=1", &response, Config.DefaultTimeoutDuration())
	})

	return response.Resources[0].Metadata.Guid
}

// Returns all app usage events that occured since the given app usage event guid
func UsageEventsAfterGuid(guid string) []AppUsageEvent {
	resources := make([]AppUsageEvent, 0)

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		firstPageUrl := "/v2/app_usage_events?results-per-page=150&order-direction=desc&page=1&after_guid=" + guid
		url := firstPageUrl

		for {
			var response AppUsageEvents
			workflowhelpers.ApiRequest("GET", url, &response, Config.DefaultTimeoutDuration())

			resources = append(resources, response.Resources...)

			if len(response.Resources) == 0 || response.NextUrl == "" {
				break
			}

			url = response.NextUrl
		}
	})

	return resources
}
