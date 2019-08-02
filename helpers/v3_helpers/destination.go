package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type DestinationProcess struct {
	Type string `json:"type"`
}

type App struct {
	GUID    string              `json:"guid"`
	Process *DestinationProcess `json:"process,omitempty"`
}

type Destination struct {
	GUID   string `json:"guid,omitempty"`
	App    App    `json:"app"`
	Port   int    `json:"port,omitempty"`
	Weight int    `json:"weight,omitempty"`
}

type Destinations struct {
	Destinations []Destination `json:"destinations"`
}

func InsertDestinations(routeGUID string, destinations []Destination) []string {
	destinationsJSON, err := json.Marshal(Destinations{Destinations: destinations})
	Expect(err).ToNot(HaveOccurred())

	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "POST", "-d", string(destinationsJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()

	var responseDestinations Destinations
	err = json.Unmarshal(response, &responseDestinations)
	Expect(err).ToNot(HaveOccurred())

	listDstGUIDs := make([]string, 0, len(responseDestinations.Destinations))
	for _, dst := range responseDestinations.Destinations {
		listDstGUIDs = append(listDstGUIDs, dst.GUID)
	}
	return listDstGUIDs
}

func ReplaceDestinations(routeGUID string, destinations []Destination) Destinations {
	destinationsJSON, err := json.Marshal(Destinations{Destinations: destinations})
	Expect(err).ToNot(HaveOccurred())

	session := cf.Cf("curl", "-f",
		fmt.Sprintf("/v3/routes/%s/destinations", routeGUID),
		"-X", "PATCH", "-d", string(destinationsJSON))

	Expect(session.Wait()).To(Exit(0))
	response := session.Out.Contents()

	var responseDestinations Destinations
	err = json.Unmarshal(response, &responseDestinations)
	Expect(err).ToNot(HaveOccurred())
	return responseDestinations
}

func CreateAndMapRoute(appGUID, spaceGUID, domainGUID, host string) {
	routeGUID := CreateRoute(spaceGUID, domainGUID, host)
	destination := Destination{App: App{GUID: appGUID}}
	InsertDestinations(routeGUID, []Destination{destination})
}

func CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, host string, port int) {
	routeGUID := CreateRoute(spaceGUID, domainGUID, host)
	destination := Destination{App: App{GUID: appGUID}, Port: port}
	InsertDestinations(routeGUID, []Destination{destination})
}

func UnmapAllRoutes(appGUID string) {
	getRoutespath := fmt.Sprintf("/v3/apps/%s/routes", appGUID)
	routesBody := cf.Cf("curl", "-f", getRoutespath).Wait().Out.Contents()
	routesJSON := struct {
		Resources []struct {
			GUID string `json:"guid"`
		} `json:"resources"`
	}{}
	err := json.Unmarshal([]byte(routesBody), &routesJSON)
	Expect(err).NotTo(HaveOccurred())

	for _, routeResource := range routesJSON.Resources {
		routeGUID := routeResource.GUID

		var destinations Destinations

		getDestinationspath := fmt.Sprintf("/v3/routes/%s/destinations", routeGUID)
		destinationsBody := cf.Cf("curl", getDestinationspath).Wait().Out.Contents()

		err := json.Unmarshal(destinationsBody, &destinations)
		Expect(err).NotTo(HaveOccurred())

		filteredDestinations := []Destination{}
		for _, destination := range destinations.Destinations {
			if destination.App.GUID != appGUID {
				filteredDestinations = append(filteredDestinations, destination)
			}
		}

		filteredDestinationsJSON, err := json.Marshal(filteredDestinations)
		Expect(err).NotTo(HaveOccurred())

		Expect(cf.Cf("curl", "-f", fmt.Sprintf("/v3/routes/%s/destinations", routeGUID), "-X", "PATCH", "-d", string(filteredDestinationsJSON)).Wait()).To(Exit(0))
	}
}
