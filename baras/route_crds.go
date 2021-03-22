package baras

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/cf-k8s-networking/routecontroller/api/v1alpha1"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/k8s_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
)

var _ = Describe("RouteCRDs", func() {
	SkipOnVMs("no route CRDs on VMs")
	var (
		appName string
	)

	BeforeEach(func() {
		appName = random_name.BARARandomName("APP")
		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))
	})

	Describe("When mapping a route to an app", func() {
		var appGuid string

		BeforeEach(func() {
			session := cf.Cf("push", appName, "--no-route", "-p", assets.NewAssets().Catnip)
			Expect(session.Wait("3m")).To(gexec.Exit(0))

			appGuid = GetAppGuid(appName)
			spaceName := TestSetup.RegularUserContext().Space
			spaceGUID := GetSpaceGuidFromName(spaceName)
			domainGUID := GetDomainGUIDFromName(Config.GetAppsDomain())

			routeGUID := CreateRouteWithPath(spaceGUID, domainGUID, appName, "/foo")
			destination := Destination{App: App{GUID: appGuid}}
			InsertDestinations(routeGUID, []Destination{destination})
		})

		It("creates a Route custom resource in Kubernetes", func() {
			By("Creating the route")
			routeGuid := GetRouteGUIDFromAppGuid(appGuid)

			routeCR, err := KubectlGetRoute("cf-workloads", routeGuid)
			Expect(err).ToNot(HaveOccurred())

			Expect(routeCR.ObjectMeta.Name).To(Equal(routeGuid))
			Expect(routeCR.Spec.Destinations[0].App.Guid).To(Equal(appGuid))
			Expect(routeCR.Spec.Url).To(Equal(fmt.Sprintf("%s.%s/foo", appName, Config.GetAppsDomain())))

			By("Deleting the route")
			session := cf.Cf("delete-route", Config.GetAppsDomain(), "--hostname", appName, "--path", "foo", "-f")
			Expect(session.Wait("3m")).To(gexec.Exit(0))
			session = KubectlSession("get", "route", routeGuid, "-n", "cf-workloads", "-o", "json")
			Expect(session.Wait("1m")).ToNot(gexec.Exit(0), "Route CR was not deleted")
			Expect(session.Err.Contents()).Should(ContainSubstring("Error from server (NotFound)"))
		})
	})

	Describe("When the route resource in Kubernetes are not aligned with the routes in CC", func() {
		Context("given there is a route resource in Kubernetes that doesn't match a route in CC", func() {
			BeforeEach(func() {
				// grab PeriodicSync resource from k8s and note the value of `status.lastTransitionTime`
				session := KubectlSession("-n", "cf-system", "get", "periodicsync", "cf-api-periodic-route-sync", "-o", `jsonpath='{.status.conditions[?(@.type=="Synced")].lastTransitionTime}'`)
				Expect(session.Wait("1m")).To(gexec.Exit(0), "Failed to get PeriodicSync resource")
				lastSyncTime := session.Out.Contents()

				// create an extra route resource that we want to see get deleted
				// sufficient to just see that the intent to delete succeeds without checking it propagated
				file, err := ioutil.TempFile("", "route.yml")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(file.Name())

				_, err = file.Write([]byte(`---	
apiVersion: networking.cloudfoundry.org/v1alpha1	
kind: Route	
metadata:
  name: bogus-route
  namespace: cf-workloads
  labels:
    app.kubernetes.io/component: cf-networking
    app.kubernetes.io/managed-by: cloudfoundry
    app.kubernetes.io/name: bogus-route
    app.kubernetes.io/part-of: cloudfoundry
    app.kubernetes.io/version: 0.0.0
    cloudfoundry.org/domain_guid: adc3255b-5a5f-468a-8832-47ae70afde25
    cloudfoundry.org/org_guid: 2a23c52c-0b46-4d7b-9d14-90a791c8719b
    cloudfoundry.org/route_guid: 17c82e6e-33ce-42ce-b28e-8cc5109f4459
    cloudfoundry.org/space_guid: 4d021f3d-13d4-4ded-be6b-d55add346749
spec:	
  destinations:	
  - app:	
      guid: 22ebb23e-0097-420f-aeb3-8903f87e0430	
      process:	
        type: web	
    guid: 28c5c615-91fa-409c-a3c5-3bfb08f3c245	
    port: 8080	
    selector:	
      matchLabels:	
        cloudfoundry.org/app_guid: 22ebb23e-0097-420f-aeb3-8903f87e0430	
        cloudfoundry.org/process_type: web	
  domain:	
    internal: false	
    name: tim.doesnt.want.his.name.here	
  host: bogus	
  url: nevermind.tim.is.vain`))
				Expect(err).ToNot(HaveOccurred())

				session = KubectlSession("apply", "-f", file.Name())
				Expect(session.Wait("2m")).To(gexec.Exit(0), "Failed to apply route resource file to Kubernetes")

				// poll PeriodicSync resource until its `status.lastTransitionTime` has updated from our initial saved-off value
				Eventually(func() string {
					session := KubectlSession("-n", "cf-system", "get", "periodicsync", "cf-api-periodic-route-sync", "-o", `jsonpath='{.status.conditions[?(@.type=="Synced")].lastTransitionTime}'`)
					Expect(session.Wait("1m")).To(gexec.Exit(0), "Failed to get PeriodicSync resource")
					return string(session.Out.Contents())
				}, "30s", "1s").ShouldNot(Equal(string(lastSyncTime)))
			})

			It("should eventually get deleted from Kubernetes", func() {
				session := KubectlSession("get", "route", "bogus-route", "-n", "cf-workloads", "-o", "json")
				Expect(session.Wait("1m")).ToNot(gexec.Exit(0), "Route CR was not deleted")
				Expect(session.Err.Contents()).Should(ContainSubstring("Error from server (NotFound)"))
			})
		})

		Context("given there is a route in CC which doesn't have a corresponding route in Kubernetes", func() {
			var (
				routeGUID string
			)

			BeforeEach(func() {
				// create route in CC
				spaceName := TestSetup.RegularUserContext().Space
				spaceGUID := GetSpaceGuidFromName(spaceName)
				domainGUID := GetDomainGUIDFromName(Config.GetAppsDomain())
				routeGUID = CreateRouteWithPath(spaceGUID, domainGUID, "hello-baras", "/foo")

				// grab PeriodicSync resource from k8s and note the value of `status.lastTransitionTime`
				session := KubectlSession("-n", "cf-system", "get", "periodicsync", "cf-api-periodic-route-sync", "-o", `jsonpath='{.status.conditions[?(@.type=="Synced")].lastTransitionTime}'`)
				Expect(session.Wait("1m")).To(gexec.Exit(0), "Failed to get PeriodicSync resource")
				lastSyncTime := session.Out.Contents()


				// delete route resource in Kubernetes
				session = KubectlSession("delete", "-n", "cf-workloads", "route", routeGUID)
				Expect(session.Wait("3m")).To(gexec.Exit(0), "Failed to delete route resource")
				Expect(session.Out.Contents()).Should(ContainSubstring(fmt.Sprintf(`"%s" deleted`, routeGUID)))

				// poll PeriodicSync resource until its `status.lastTransitionTime` has updated from our initial saved-off value
				Eventually(func() string {
					session := KubectlSession("-n", "cf-system", "get", "periodicsync", "cf-api-periodic-route-sync", "-o", `jsonpath='{.status.conditions[?(@.type=="Synced")].lastTransitionTime}'`)
					Expect(session.Wait("1m")).To(gexec.Exit(0), "Failed to get PeriodicSync resource")
					return string(session.Out.Contents())
				}, "30s", "1s").ShouldNot(Equal(string(lastSyncTime)))
			})

			AfterEach(func() {
				DeleteRoute(routeGUID)
			})

			It("should eventually recreate the route resource in Kubernetes", func() {
				session := KubectlSession("get", "route", routeGUID, "-n", "cf-workloads", "-o", "json")
				Expect(session.Wait("1m")).To(gexec.Exit(0), "Route CR was not recreated")

				var route v1alpha1.Route
				err := json.Unmarshal(session.Out.Contents(), &route)
				Expect(err).ToNot(HaveOccurred())

				Expect(route.Spec.Host).To(Equal("hello-baras"))
				Expect(route.Spec.Path).To(Equal("/foo"))
			})
		})
	})
})
