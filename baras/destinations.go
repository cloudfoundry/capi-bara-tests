package baras

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("destinations", func() {
	var (
		routeGUID    string
		app1Name     string
		app2Name     string
		app1GUID     string
		app2GUID     string
		spaceName    string
		spaceGUID    string
		destinations []Destination
	)

	BeforeEach(func() {
		app1Name = random_name.BARARandomName("APP")
		app2Name = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		app1GUID = CreateApp(app1Name, spaceGUID, `{"foo1":"bar1"}`)
		app2GUID = CreateApp(app2Name, spaceGUID, `{"foo2":"bar2"}`)
		routeGUID = CreateRoute(spaceGUID, GetDomainGUIDFromName(Config.GetAppsDomain()), random_name.BARARandomName("route"))
	})

	AfterEach(func() {
		DeleteApp(app1GUID)
		DeleteApp(app2GUID)
		DeleteRoute(routeGUID)
	})

	Describe("Insert destinations", func() {
		var (
			response []byte
		)
		JustBeforeEach(func() {
			routePath := fmt.Sprintf("/v3/routes/%s/destinations", routeGUID)
			session := cf.Cf("curl", routePath)
			response = session.Wait().Out.Contents()
		})

		Describe("Insert with process types", func() {
			BeforeEach(func() {
				destinations = []Destination{
					{
						App: App{
							GUID:    app1GUID,
							Process: &DestinationProcess{Type: "web"},
						},
						Port: 8080,
					},
					{
						App: App{
							GUID:    app2GUID,
							Process: &DestinationProcess{Type: "worker"},
						},
						Port: 8080,
					},
				}
				InsertDestinations(routeGUID, destinations)
			})

			It("inserts both destinations with the appropriate process types", func() {
				var responseDestinations struct {
					Destinations []Destination `json:"destinations"`
				}
				err := json.Unmarshal(response, &responseDestinations)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseDestinations.Destinations[0].App).To(Equal(destinations[0].App))
				Expect(responseDestinations.Destinations[0].Port).To(Equal(destinations[0].Port))
				Expect(responseDestinations.Destinations[1].App).To(Equal(destinations[1].App))
				Expect(responseDestinations.Destinations[1].Port).To(Equal(destinations[1].Port))
			})
		})

		Describe("Insert with ports", func() {
			BeforeEach(func() {
				destinations = []Destination{
					{
						App: App{
							GUID:    app1GUID,
							Process: &DestinationProcess{Type: "web"},
						},
						Port: 8080,
					},
					{
						App: App{
							GUID:    app2GUID,
							Process: &DestinationProcess{Type: "web"},
						},
						Port: 8081,
					},
				}
				InsertDestinations(routeGUID, destinations)
			})

			It("inserts both destinations with the appropriate ports", func() {
				var responseDestinations struct {
					Destinations []Destination `json:"destinations"`
				}
				err := json.Unmarshal(response, &responseDestinations)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseDestinations.Destinations[0].App).To(Equal(destinations[0].App))
				Expect(responseDestinations.Destinations[0].Port).To(Equal(destinations[0].Port))
				Expect(responseDestinations.Destinations[1].App).To(Equal(destinations[1].App))
				Expect(responseDestinations.Destinations[1].Port).To(Equal(destinations[1].Port))
			})
		})
	})

	Describe("Remove destinations", func() {
		var (
			response         []byte
			destinationGUIDs []string
			routePath        string
		)
		BeforeEach(func() {
			destinations = []Destination{
				{
					App: App{GUID: app1GUID},
				},
				{
					App: App{GUID: app2GUID},
				},
			}
			destinationGUIDs = InsertDestinations(routeGUID, destinations)
			routePath = fmt.Sprintf("/v3/routes/%s/destinations", routeGUID)

			session := cf.Cf("curl", "-X", "DELETE", fmt.Sprintf("%s/%s", routePath, destinationGUIDs[0]))
			Eventually(session).Should(Exit(0))

		})

		It("removes one destination", func() {
			var responseDestinations struct {
				Destinations []Destination `json:"destinations"`
			}

			response = cf.Cf("curl", routePath).Wait().Out.Contents()
			err := json.Unmarshal(response, &responseDestinations)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseDestinations.Destinations[0].GUID).To(Equal(destinationGUIDs[1]))
			Expect(len(responseDestinations.Destinations)).To(Equal(1))
		})
	})

	Describe("Replace destinations", func() {
		var (
			app3Name string
			app3GUID string
		)

		BeforeEach(func() {
			app3Name = random_name.BARARandomName("APP")
			app3GUID = CreateApp(app3Name, spaceGUID, `{"foo3":"bar3"}`)
			destinations = []Destination{
				{
					App: App{
						GUID:    app3GUID,
						Process: &DestinationProcess{Type: "web"},
					},
					Port: 8080,
				},
			}
			InsertDestinations(routeGUID, destinations)
		})

		AfterEach(func() {
			DeleteApp(app3GUID)
		})

		It("replaces them", func() {
			destinations = []Destination{
				{
					App: App{
						GUID:    app1GUID,
						Process: &DestinationProcess{Type: "web"},
					},
					Port:   8080,
					Weight: 51,
				},
				{
					App: App{
						GUID:    app2GUID,
						Process: &DestinationProcess{Type: "worker"},
					},
					Port:   8080,
					Weight: 49,
				},
			}

			responseDestinations := ReplaceDestinations(routeGUID, destinations)

			Expect(responseDestinations.Destinations[0].App).To(Equal(destinations[0].App))
			Expect(responseDestinations.Destinations[0].Port).To(Equal(destinations[0].Port))
			Expect(responseDestinations.Destinations[0].Weight).To(Equal(destinations[0].Weight))
			Expect(responseDestinations.Destinations[1].App).To(Equal(destinations[1].App))
			Expect(responseDestinations.Destinations[1].Port).To(Equal(destinations[1].Port))
			Expect(responseDestinations.Destinations[1].Weight).To(Equal(destinations[1].Weight))
		})
	})
})
