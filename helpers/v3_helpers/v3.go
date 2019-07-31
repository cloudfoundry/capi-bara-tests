package v3_helpers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	V3_DEFAULT_MEMORY_LIMIT = "256"
	V3_JAVA_MEMORY_LIMIT    = "1024"
)

type process struct {
	Type string `json:"type"`
}

type app struct {
	Guid    string  `json:"guid"`
	Process process `json:"process"`
}

type Destination struct {
	App  app `json:"app"`
	Port int `json:"port"`
}

type WeightedDestination struct {
	App    app `json:"app"`
	Port   int `json:"port"`
	Weight int `json:"weight"`
}

type ResponseDestination struct {
	Guid   string `json:"guid"`
	App    app    `json:"app"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
}

func CreateDeployment(appGuid string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"relationships": {"app": {"data": {"guid": "%s"}}}}`, appGuid)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		Guid string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &deployment)
	return deployment.Guid
}

func CreateDeploymentForDroplet(appGuid, dropletGuid string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"droplet": {"guid": "%s"}, "relationships": {"app": {"data": {"guid": "%s"}}}}`, dropletGuid, appGuid)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		Guid string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &deployment)
	return deployment.Guid
}

func RollbackDeployment(appGuid, revisionGuid string) string {
	deploymentPath := fmt.Sprintf("/v3/deployments")
	deploymentRequestBody := fmt.Sprintf(`{"revision": { "guid": "%s" },"relationships": {"app": {"data": {"guid": "%s"}}}}`, revisionGuid, appGuid)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-d", deploymentRequestBody).Wait()
	Expect(session).To(Exit(0))
	var deployment struct {
		Guid string `json:"guid"`
	}

	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &deployment)
	return deployment.Guid
}

func CancelDeployment(deploymentGuid string) {
	deploymentPath := fmt.Sprintf("/v3/deployments/%s/actions/cancel", deploymentGuid)
	session := cf.Cf("curl", "-f", deploymentPath, "-X", "POST", "-i").Wait()
	Expect(session.Out.Contents()).To(ContainSubstring("200 OK"))
	Expect(session).To(Exit(0))
}

func WaitUntilDeploymentReachesState(deploymentGuid, status string) {
	deploymentPath := fmt.Sprintf("/v3/deployments/%s", deploymentGuid)
	deploymentJson := struct {
		State string `json:"state"`
	}{}

	Eventually(func() string {
		session := cf.Cf("curl", "-f", deploymentPath).Wait()
		Expect(session.Wait()).To(Exit(0))
		json.Unmarshal(session.Out.Contents(), &deploymentJson)
		return deploymentJson.State
	}, Config.LongCurlTimeoutDuration()).Should(Equal(status))
}

func ScaleApp(appGuid string, instances int) {
	scalePath := fmt.Sprintf("/v3/apps/%s/processes/web/actions/scale", appGuid)
	scaleBody := fmt.Sprintf(`{"instances": "%d"}`, instances)
	Expect(cf.Cf("curl", "-f", scalePath, "-X", "POST", "-d", scaleBody).Wait()).To(Exit(0))
}

func GetRunningInstancesStats(processGuid string) int {
	processPath := fmt.Sprintf("/v3/processes/%s/stats", processGuid)
	session := cf.Cf("curl", "-f", processPath).Wait()
	instancesJson := struct {
		Resources []struct {
			Type  string `json:"type"`
			State string `json:"state"`
		} `json:"resources"`
	}{}

	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &instancesJson)
	numRunning := 0

	for _, instance := range instancesJson.Resources {
		if instance.State == "RUNNING" {
			numRunning += 1
		}
	}
	return numRunning
}

func WaitForAppToStop(appGUID string) {
	processGUIDS := GetProcessGuidsForType(appGUID, "web")

	Eventually(func() int {
		return GetRunningInstancesStats(processGUIDS[0])
	}, Config.CfPushTimeoutDuration(), time.Second).Should(Equal(0))
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

func GetProcessGuidsForType(appGuid string, processType string) []string {
	processesPath := fmt.Sprintf("/v3/apps/%s/processes?types=%s", appGuid, processType)
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

func AssignDropletToApp(appGuid, dropletGuid string) {
	appUpdatePath := fmt.Sprintf("/v3/apps/%s/relationships/current_droplet", appGuid)
	appUpdateBody := fmt.Sprintf(`{"data": {"guid":"%s"}}`, dropletGuid)
	Expect(cf.Cf("curl", "-f", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait()).To(Exit(0))

	for _, process := range GetProcesses(appGuid, "") {
		ScaleProcess(appGuid, process.Type, V3_DEFAULT_MEMORY_LIMIT)
	}
}

func AssociateNewDroplet(appGUID, assetPath string) string {
	By("Creating a Package")
	packageGUID := CreatePackage(appGUID)
	uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

	By("Uploading a Package")
	UploadPackage(uploadURL, assetPath, GetAuthToken())
	WaitForPackageToBeReady(packageGUID)

	By("Creating a Build")
	buildGUID := StageBuildpackPackage(packageGUID)
	WaitForBuildToStage(buildGUID)
	dropletGUID := GetDropletFromBuild(buildGUID)

	AssignDropletToApp(appGUID, dropletGUID)

	return dropletGUID
}

func UpdateEnvironmentVariables(appGuid, envVars string) {
	appUpdatePath := fmt.Sprintf("/v3/apps/%s/environment_variables", appGuid)
	appUpdateBody := fmt.Sprintf(`{"var": %s}`, envVars)
	Expect(cf.Cf("curl", "-f", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait()).To(Exit(0))
}

func AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)),
	).Should(Exit(0))
}

func ToDestination(appGuid, processType string, port int) Destination {
	return Destination{App: app{Guid: appGuid, Process: process{Type: processType}}, Port: port}
}

func ToWeightedDestination(appGuid, processType string, port int, weight int) WeightedDestination {
	return WeightedDestination{App: app{Guid: appGuid, Process: process{Type: processType}}, Port: port, Weight: weight}
}

func InsertDestinations(routeGUID string, appGUIDs []string) []string {
	appGUIDsWithProcessTypes := make(map[string]string)
	for _, appGUID := range appGUIDs {
		appGUIDsWithProcessTypes[appGUID] = "web"
	}
	return InsertDestinationsWithProcessTypes(routeGUID, appGUIDsWithProcessTypes)
}

func InsertDestinationsWithProcessTypes(routeGUID string, appGUIDsWithProcessTypes map[string]string) []string {
	var destinations []Destination
	for appGUID, processType := range appGUIDsWithProcessTypes {
		destinations = append(destinations, ToDestination(appGUID, processType, 8080))
	}

	routeMappingJSON, err := json.Marshal(
		struct {
			Destinations []Destination `json:"destinations"`
		}{
			Destinations: destinations,
		},
	)
	Expect(err).NotTo(HaveOccurred())
	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "POST", "-d", string(routeMappingJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()

	var responseDestinations struct {
		Destinations []ResponseDestination `json:"destinations"`
	}
	err = json.Unmarshal(response, &responseDestinations)
	Expect(err).ToNot(HaveOccurred())

	var listDstGuids []string
	for _, dst := range responseDestinations.Destinations {
		listDstGuids = append(listDstGuids, dst.Guid)
	}
	return listDstGuids
}

func InsertDestinationsWithPorts(routeGUID string, appGUIDsWithPorts map[string]int) []string {
	var destinations []Destination
	for appGUID, port := range appGUIDsWithPorts {
		destinations = append(destinations,
			ToDestination(appGUID, "web", port))
	}

	routeMappingJSON, err := json.Marshal(
		struct {
			Destinations []Destination `json:"destinations"`
		}{
			Destinations: destinations,
		},
	)
	Expect(err).NotTo(HaveOccurred())
	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "POST", "-d", string(routeMappingJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()

	var responseDestinations struct {
		Destinations []ResponseDestination `json:"destinations"`
	}
	err = json.Unmarshal(response, &responseDestinations)
	Expect(err).ToNot(HaveOccurred())

	var listDstGuids []string
	for _, dst := range responseDestinations.Destinations {
		listDstGuids = append(listDstGuids, dst.Guid)
	}
	return listDstGuids
}

func ReplaceDestinationsWithWeights(routeGUID string, destinations []WeightedDestination) []byte {
	routeMappingJSON, err := json.Marshal(
		struct {
			Destinations []WeightedDestination `json:"destinations"`
		}{
			Destinations: destinations,
		},
	)
	Expect(err).NotTo(HaveOccurred())

	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "PATCH", "-d", string(routeMappingJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()
	return response
}

func CreateAndMapRoute(appGUID, spaceGUID, domainGUID, host string) {
	routeGUID := CreateRoute(spaceGUID, domainGUID, host)
	InsertDestinations(routeGUID, []string{appGUID})
}

func CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, host string, port int) {
	routeGUID := CreateRoute(spaceGUID, domainGUID, host)
	InsertDestinationsWithPorts(routeGUID, map[string]int{appGUID: port})
}

func UnmapAllRoutes(appGuid string) {
	getRoutespath := fmt.Sprintf("/v3/apps/%s/routes", appGuid)
	routesBody := cf.Cf("curl", "-f", getRoutespath).Wait().Out.Contents()
	routesJSON := struct {
		Resources []struct {
			Guid string `json:"guid"`
		} `json:"resources"`
	}{}
	json.Unmarshal([]byte(routesBody), &routesJSON)

	for _, routeResource := range routesJSON.Resources {
		routeGuid := routeResource.Guid

		type app struct {
			Guid string `json:"guid"`
		}

		type destination struct {
			Guid string `json:"guid"`
			App  app    `json:"app"`
		}

		type destinations struct {
			Destinations []destination `json:"destinations"`
		}

		getDestinationspath := fmt.Sprintf("/v3/routes/%s/destinations", routeGuid)
		destinationsBody := cf.Cf("curl", getDestinationspath).Wait().Out.Contents()

		var destinationsJSON destinations
		json.Unmarshal([]byte(destinationsBody), &destinationsJSON)

		filteredDestinations := []destination{}
		for _, destination := range destinationsJSON.Destinations {
			if destination.App.Guid != appGuid {
				filteredDestinations = append(filteredDestinations, destination)
			}
		}

		filteredDestinationsJSON, err := json.Marshal(filteredDestinations)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("curl", "-f", fmt.Sprintf("/v3/routes/%s/destinations", routeGuid), "-X", "PATCH", "-d", string(filteredDestinationsJSON)).Wait()).To(Exit(0))
	}
}

func CreateApp(appName, spaceGuid, environmentVariables string) string {
	session := cf.Cf("curl", "-f", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s}`, appName, spaceGuid, environmentVariables))
	bytes := session.Wait().Out.Contents()
	var app struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &app)
	return app.Guid
}

func CreateSidecar(name string, processTypes []string, command string, memoryLimit int, appGuid string) string {
	sidecarEndpoint := fmt.Sprintf("/v3/apps/%s/sidecars", appGuid)
	sidecarOneJSON, err := json.Marshal(
		struct {
			Name         string   `json:"name"`
			Command      string   `json:"command"`
			ProcessTypes []string `json:"process_types"`
			Memory       int      `json:"memory_in_mb"`
		}{
			name,
			command,
			processTypes,
			memoryLimit,
		},
	)
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

func CreateDockerApp(appName, spaceGuid, environmentVariables string) string {
	session := cf.Cf("curl", "-f", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s, "lifecycle": {"type": "docker", "data": {} } }`, appName, spaceGuid, environmentVariables))
	bytes := session.Wait().Out.Contents()
	var app struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &app)
	return app.Guid
}

func CreateDockerPackage(appGuid, imagePath string) string {
	packageCreateUrl := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", "-f", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"docker", "data": {"image": "%s"}}`, appGuid, imagePath))
	bytes := session.Wait().Out.Contents()
	var pac struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &pac)
	return pac.Guid
}

func CreateIsolationSegment(name string) string {
	session := cf.Cf("curl", "-f", "/v3/isolation_segments", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s"}`, name))
	bytes := session.Wait().Out.Contents()

	var isolation_segment struct {
		Guid string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &isolation_segment)
	Expect(err).ToNot(HaveOccurred())

	return isolation_segment.Guid
}

func CreateOrGetIsolationSegment(name string) string {
	isoSegGUID := CreateIsolationSegment(name)
	if isoSegGUID == "" {
		isoSegGUID = GetIsolationSegmentGuid(name)
	}
	return isoSegGUID
}

func CreatePackage(appGuid string) string {
	packageCreateUrl := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", "-f", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"bits"}`, appGuid))
	bytes := session.Wait().Out.Contents()
	var pac struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &pac)
	return pac.Guid
}

func CreateRoute(spaceGUID, domainGUID, host string) string {
	session := cf.Cf(
		"curl", "-f", "/v3/routes",
		"-X", "POST",
		"-d", fmt.Sprintf(`{
			"host": "%s",
			"relationships": {
				"domain": { "data": { "guid": "%s" } },
				"space": { "data": { "guid": "%s" } }
			}
		}`, host, domainGUID, spaceGUID),
	)
	bytes := session.Wait().Out.Contents()

	var response struct {
		GUID string `json:"guid"`
	}
	json.Unmarshal(bytes, &response)
	return response.GUID
}

func HandleAsyncRequest(path string, method string) {
	session := cf.Cf("curl", "-f", path, "-X", method, "-i")
	bytes := session.Wait().Out.Contents()
	Expect(string(bytes)).To(ContainSubstring("202 Accepted"))

	jobPath := GetJobPath(bytes)
	PollJob(jobPath)
}

func GetJobPath(response []byte) string {
	r, err := regexp.Compile(`Location:.*(/v3/jobs/[\w-]*)`)
	Expect(err).ToNot(HaveOccurred())
	return r.FindStringSubmatch(string(response))[1]
}

func PollJob(jobPath string) {
	Eventually(func() string {
		jobSession := cf.Cf("curl", "-f", jobPath)
		return string(jobSession.Wait().Out.Contents())
	}).Should(ContainSubstring("COMPLETE"))
}

func PollJobAsFailed(jobPath string) {
	Eventually(func() string {
		jobSession := cf.Cf("curl", "-f", jobPath)
		return string(jobSession.Wait().Out.Contents())
	}).Should(ContainSubstring("FAILED"))
}

type jobError struct {
	Detail string `json:"detail"`
	Title  string `json:"title"`
	Code   int    `json:"code"`
}

func GetJobErrors(jobPath string) []jobError {
	session := cf.Cf("curl", "-f", jobPath).Wait()
	var job struct {
		Guid   string     `json:"guid"`
		Errors []jobError `json:"errors"`
	}

	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &job)
	return job.Errors
}

func DeleteApp(appGuid string) {
	HandleAsyncRequest(fmt.Sprintf("/v3/apps/%s", appGuid), "DELETE")
}

func DeleteRoute(routeGuid string) {
	HandleAsyncRequest(fmt.Sprintf("/v3/routes/%s", routeGuid), "DELETE")
}

func DeleteIsolationSegment(guid string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments/%s", guid), "-X", "DELETE")).Should(Exit(0))
}

func EntitleOrgToIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations", isoSegGuid),
		"-X",
		"POST",
		"-d",
		fmt.Sprintf(`{"data":[{ "guid":"%s" }]}`, orgGuid)),
	).Should(Exit(0))
}

func FetchRecentLogs(appGuid, oauthToken string, config config.BaraConfig) *Session {
	loggregatorEndpoint := getHttpLoggregatorEndpoint()
	logUrl := fmt.Sprintf("%s/apps/%s/recentlogs", loggregatorEndpoint, appGuid)
	session := helpers.Curl(Config, logUrl, "-H", fmt.Sprintf("Authorization: %s", oauthToken))
	Expect(session.Wait()).To(Exit(0))
	return session
}

func GetAuthToken() string {
	session := cf.Cf("oauth-token")
	bytes := session.Wait().Out.Contents()
	return strings.TrimSpace(string(bytes))
}

func GetDefaultIsolationSegment(orgGuid string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid))
	bytes := session.Wait().Out.Contents()
	return GetIsolationSegmentGuidFromResponse(bytes)
}

func GetDropletFromBuild(buildGuid string) string {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGuid)
	session := cf.Cf("curl", "-f", buildPath).Wait()
	var build struct {
		Droplet struct {
			Guid string `json:"guid"`
		} `json:"droplet"`
	}
	bytes := session.Wait().Out.Contents()
	json.Unmarshal(bytes, &build)
	return build.Droplet.Guid
}

func GetGuidFromResponse(response []byte) string {
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if len(GetResponse.Resources) == 0 {
		Fail("No guid found for response")
	}

	return GetResponse.Resources[0].Guid
}

func GetIsolationSegmentGuid(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

func GetIsolationSegmentGuidFromResponse(response []byte) string {
	type data struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Data data `json:"data"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if (data{}) == GetResponse.Data {
		return ""
	}

	return GetResponse.Data.Guid
}

func GetSpaceGuidFromName(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

func GetDomainGUIDFromName(name string) string {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/domains?names=%s", name))
	bytes := session.Wait().Out.Contents()
	return GetGuidFromResponse(bytes)
}

func IsolationSegmentExists(name string) bool {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait().Out.Contents()
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func OrgEntitledToIsolationSegment(orgGuid string, isoSegName string) bool {
	session := cf.Cf("curl", "-f", fmt.Sprintf("/v3/isolation_segments?names=%s&organization_guids=%s", isoSegName, orgGuid))
	bytes := session.Wait().Out.Contents()

	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func RevokeOrgEntitlementForIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations/%s", isoSegGuid, orgGuid),
		"-X",
		"DELETE",
	)).Should(Exit(0))
}

func ScaleProcess(appGuid, processType, memoryInMb string) {
	scalePath := fmt.Sprintf("/v3/apps/%s/processes/%s/actions/scale", appGuid, processType)
	scaleBody := fmt.Sprintf(`{"memory_in_mb":"%s"}`, memoryInMb)
	session := cf.Cf("curl", "-f", scalePath, "-X", "POST", "-d", scaleBody).Wait()
	Expect(session).To(Exit(0))
	result := session.Out.Contents()
	Expect(strings.Contains(string(result), "errors")).To(BeFalse())
}

func SetDefaultIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)),
	).Should(Exit(0))
}

func StageBuildpackPackage(packageGuid string, buildpacks ...string) string {
	buildpackString := "null"
	if len(buildpacks) > 0 {
		buildpackString = fmt.Sprintf(`["%s"]`, strings.Join(buildpacks, `", "`))
	}

	stageBody := fmt.Sprintf(`{"lifecycle":{ "type": "buildpack", "data": { "buildpacks": %s } }, "package": { "guid" : "%s"}}`, buildpackString, packageGuid)
	stageUrl := "/v3/builds"
	session := cf.Cf("curl", "-f", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait().Out.Contents()
	var build struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &build)
	Expect(build.Guid).NotTo(BeEmpty())
	return build.Guid
}

func StageDockerPackage(packageGuid string) string {
	stageBody := fmt.Sprintf(`{"lifecycle": { "type" : "docker", "data": {} }, "package": { "guid" : "%s"}}`, packageGuid)
	stageUrl := "/v3/builds"
	session := cf.Cf("curl", "-f", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait().Out.Contents()
	var build struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &build)
	return build.Guid
}

func StartApp(appGuid string) {
	startURL := fmt.Sprintf("/v3/apps/%s/actions/start", appGuid)
	Expect(cf.Cf("curl", "-f", startURL, "-X", "POST").Wait()).To(Exit(0))
}

func StopApp(appGuid string) {
	stopURL := fmt.Sprintf("/v3/apps/%s/actions/stop", appGuid)
	Expect(cf.Cf("curl", "-f", stopURL, "-X", "POST").Wait()).To(Exit(0))
}

func RestartApp(appGuid string) {
	restartURL := fmt.Sprintf("/v3/apps/%s/actions/restart", appGuid)
	Expect(cf.Cf("curl", "-f", restartURL, "-X", "POST").Wait()).To(Exit(0))
}

func UnassignIsolationSegmentFromSpace(spaceGuid string) {
	Eventually(cf.Cf("curl", "-f", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGuid),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid":null}}`),
	).Should(Exit(0))
}

func UnsetDefaultIsolationSegment(orgGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid": null}}`),
	).Should(Exit(0))
}

func UploadPackage(uploadUrl, packageZipPath, token string) {
	bits := fmt.Sprintf(`bits=@%s`, packageZipPath)
	curl := helpers.Curl(Config, "-v", "-s", uploadUrl, "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
	Expect(curl).To(Exit(0))
}

func EnableRevisions(appGuid string) {
	path := fmt.Sprintf("/v3/apps/%s/features/revisions", appGuid)
	curl := cf.Cf("curl", "-f", path, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()
	Expect(curl).To(Exit(0))
}

func WaitForBuildToStage(buildGuid string) {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", buildPath).Wait()
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}

func WaitForDropletToCopy(dropletGuid string) {
	dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", dropletPath).Wait()
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}

func WaitForPackageToBeReady(packageGuid string) {
	pkgUrl := fmt.Sprintf("/v3/packages/%s", packageGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", "-f", pkgUrl)
		Expect(session.Wait()).To(Exit(0))
		return session
	}, Config.LongCurlTimeoutDuration()).Should(Say("READY"))
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
		afterGuidParam := ""
		if afterGUID != "" {
			afterGuidParam = fmt.Sprintf("&after_guid=%s", afterGUID)
		}
		usageEventsUrl := fmt.Sprintf("/v2/app_usage_events?order-direction=desc&page=1&results-per-page=150%s", afterGuidParam)
		workflowhelpers.ApiRequest("GET", usageEventsUrl, &response, Config.DefaultTimeoutDuration())
	})

	for _, event := range response.Resources {
		if event.Entity.ProcessType == processType && event.Entity.State == state {
			return true, event
		}
	}

	return false, ProcessAppUsageEvent{}
}

//private

func getHttpLoggregatorEndpoint() string {
	infoCommand := cf.Cf("curl", "-f", "/v2/info")
	Expect(infoCommand.Wait()).To(Exit(0))

	var response struct {
		DopplerLoggingEndpoint string `json:"doppler_logging_endpoint"`
	}

	err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	return strings.Replace(response.DopplerLoggingEndpoint, "ws", "http", 1)
}
