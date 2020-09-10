package baras

import (
	"fmt"

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
	var (
		appName string
	)

	BeforeEach(func() {
		//if !Config.GetIncludeKpack() {
		//	Skip(skip_messages.SkipKpackMessage)
		//}

		appName = random_name.BARARandomName("APP")
		session := cf.Cf("target",
			"-o", TestSetup.RegularUserContext().Org,
			"-s", TestSetup.RegularUserContext().Space)
		Eventually(session).Should(gexec.Exit(0))
	})

	Describe("When mapping a route to an app", func() {
		Context("using v2 endpoints", func() {
			BeforeEach(func() {
				session := cf.Cf("push", appName, "--no-route", "-p", assets.NewAssets().Catnip)
				Expect(session.Wait("3m")).To(gexec.Exit(0))

				session = cf.Cf("map-route", appName, Config.GetAppsDomain(), "--hostname", "bar", "--path", "foo")
				Expect(session.Wait("3m")).To(gexec.Exit(0))
			})

			It("creates a Route custom resource in Kubernetes", func() {
				appGuid := GetAppGuid(appName)
				routeGuid := GetRouteGUIDFromAppGuid(appGuid)
				By("Creating the route")
				routeCR, err := KubectlGetRoute("cf-workloads", routeGuid)
				Expect(err).ToNot(HaveOccurred())

				Expect(routeCR.ObjectMeta.Name).To(Equal(routeGuid))
				Expect(routeCR.Spec.Destinations[0].App.Guid).To(Equal(appGuid))
				Expect(routeCR.Spec.Url).To(Equal(fmt.Sprintf("bar.%s/foo", Config.GetAppsDomain())))

				By("Deleting the route")
				session := cf.Cf("delete-route", Config.GetAppsDomain(), "--hostname", "bar", "--path", "foo", "-f")
				Expect(session.Wait("3m")).To(gexec.Exit(0))
				output, err := Kubectl("get", "route", routeGuid, "-n", "cf-workloads", "-o", "json")
				Expect(err).To(HaveOccurred(), "Route CR was not deleted")
				Expect(output).To(ContainSubstring("Error from server (NotFound)"))
			})
		})

		Context("using v3 endpoints", func() {
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
				output, err := Kubectl("get", "route", routeGuid, "-n", "cf-workloads", "-o", "json")
				Expect(err).To(HaveOccurred(), "Route CR was not deleted")
				Expect(output).To(ContainSubstring("Error from server (NotFound)"))
			})
		})
	})
})


