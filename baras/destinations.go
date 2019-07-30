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
		routeGUID string
		app1Name  string
		app2Name  string
		app1GUID  string
		app2GUID  string
		spaceName string
		spaceGUID string
	)

	BeforeEach(func() {
		app1Name = random_name.BARARandomName("APP")
		app2Name = random_name.BARARandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		app1GUID = CreateApp(app1Name, spaceGUID, `{"foo1":"bar1"}`)
		app2GUID = CreateApp(app2Name, spaceGUID, `{"foo2":"bar2"}`)
		routeGUID = CreateRoute(spaceGUID, GetDomainGUIDFromName(Config.GetAppsDomain()), "anotherunrealhostname")
	})

	AfterEach(func() {
		DeleteApp(app1GUID)
		DeleteApp(app2GUID)
		DeleteRoute(routeGUID)
	})

	Describe("Insert destinations", func() {
		var response []byte
		JustBeforeEach(func() {
			routePath := fmt.Sprintf("/v3/routes/%s/destinations", routeGUID)
			session := cf.Cf("curl", routePath)
			response = session.Wait().Out.Contents()
		})

		Describe("Regular Insert", func() {
			BeforeEach(func() {
				InsertDestinations(routeGUID, []string{app1GUID, app2GUID})
			})

			It("inserts both destinations", func() {
				dst1 := ToDestination(app1GUID, "web", 8080)
				dst2 := ToDestination(app2GUID, "web", 8080)
				destinations := []Destination{dst1, dst2}

				var responseDestinations struct {
					Destinations []Destination `json:"destinations"`
				}
				err := json.Unmarshal(response, &responseDestinations)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseDestinations.Destinations).To(ConsistOf(destinations))
			})
		})

		Describe("Insert with process types", func() {
			BeforeEach(func() {
				InsertDestinationsWithProcessTypes(routeGUID,
					map[string]string{
						app1GUID: "web",
						app2GUID: "worker",
					})
			})

			It("inserts both destinations with the appropriate process types", func() {
				dst1 := ToDestination(app1GUID, "web", 8080)
				dst2 := ToDestination(app2GUID, "worker", 8080)
				destinations := []Destination{dst1, dst2}

				var responseDestinations struct {
					Destinations []Destination `json:"destinations"`
				}
				err := json.Unmarshal(response, &responseDestinations)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseDestinations.Destinations).To(ConsistOf(destinations))
			})
		})

		Describe("Insert with ports", func() {
			BeforeEach(func() {
				InsertDestinationsWithPorts(routeGUID,
					map[string]int{
						app1GUID: 8080,
						app2GUID: 8081,
					})
			})
			It("inserts both destinations with the appropriate ports", func() {
				dst1 := ToDestination(app1GUID, "web", 8080)
				dst2 := ToDestination(app2GUID, "web", 8081)
				destinations := []Destination{dst1, dst2}

				var responseDestinations struct {
					Destinations []Destination `json:"destinations"`
				}
				err := json.Unmarshal(response, &responseDestinations)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseDestinations.Destinations).To(ConsistOf(destinations))
			})
		})
	})

	Describe("Remove destinations", func() {
		var (
			response  []byte
			routePath string
		)
		BeforeEach(func() {
			destinations := InsertDestinations(routeGUID, []string{app1GUID})
			InsertDestinations(routeGUID, []string{app2GUID})
			routePath = fmt.Sprintf("/v3/routes/%s/destinations", routeGUID)

			session := cf.Cf("curl", "-X", "DELETE", fmt.Sprintf("%s/%s", routePath, destinations[0]))
			Eventually(session).Should(Exit(0))

		})

		It("removes one destination", func() {
			dst2 := ToDestination(app2GUID, "web", 8080)
			destinations := []Destination{dst2}

			var responseDestinations struct {
				Destinations []Destination `json:"destinations"`
			}

			response = cf.Cf("curl", routePath).Wait().Out.Contents()
			err := json.Unmarshal(response, &responseDestinations)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseDestinations.Destinations).To(Equal(destinations))
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
			InsertDestinations(routeGUID, []string{app1GUID})
		})

		AfterEach(func() {
			DeleteApp(app3GUID)
		})

		It("replaces them", func() {
			dst1 := ToWeightedDestination(app1GUID, "web", 8080, 51)
			dst2 := ToWeightedDestination(app2GUID, "worker", 8080, 49)
			destinations := []WeightedDestination{dst1, dst2}

			response := ReplaceDestinationsWithWeights(routeGUID, destinations)

			var responseDestinations struct {
				Destinations []WeightedDestination `json:"destinations"`
			}
			err := json.Unmarshal(response, &responseDestinations)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseDestinations.Destinations).To(ConsistOf(destinations))
		})
	})
})
