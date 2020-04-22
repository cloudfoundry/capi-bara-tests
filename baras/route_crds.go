package baras

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/app_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"

	route_crds "code.cloudfoundry.org/cf-k8s-networking/routecontroller/api/v1alpha1"
)

var _ = Describe("RouteCRDs", func() {
	var (
		appName string
	)

	BeforeEach(func() {
		if !Config.GetIncludeKpack() {
			Skip(skip_messages.SkipKpackMessage)
		}

		appName = random_name.BARARandomName("APP")
	})

	Describe("When mapping a route to an app", func() {
		Context("using v2 endpoints", func() {
			BeforeEach(func() {
				session := cf.Cf("push", appName, "--no-route", "-p", assets.NewAssets().CatnipRoot)
				Expect(session.Wait("3m")).To(gexec.Exit(0))

				session = cf.Cf("map-route", appName, Config.GetAppsDomain(), "--hostname", "bar", "--path", "foo")
				Expect(session.Wait("3m")).To(gexec.Exit(0))
			})

			It("creates a Route custom resource in Kubernetes", func() {
				appGuid := GetAppGuid(appName)
				routeGuid := GetRouteGUIDFromAppGuid(appGuid)

				routeCR, err := kubectlGetRoute("cf-workloads", routeGuid)
				Expect(err).ToNot(HaveOccurred())

				Expect(routeCR.ObjectMeta.Name).To(Equal(routeGuid))
				Expect(routeCR.Spec.Destinations[0].App.Guid).To(Equal(appGuid))
				Expect(routeCR.Spec.Url).To(Equal(fmt.Sprintf("bar.%s/foo", Config.GetAppsDomain())))
			})
		})

		Context("using v3 endpoints", func() {
			var appGuid string

			BeforeEach(func() {
				session := cf.Cf("push", appName, "--no-route", "-p", assets.NewAssets().CatnipRoot)
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
				routeGuid := GetRouteGUIDFromAppGuid(appGuid)

				routeCR, err := kubectlGetRoute("cf-workloads", routeGuid)
				Expect(err).ToNot(HaveOccurred())

				Expect(routeCR.ObjectMeta.Name).To(Equal(routeGuid))
				Expect(routeCR.Spec.Destinations[0].App.Guid).To(Equal(appGuid))
				Expect(routeCR.Spec.Url).To(Equal(fmt.Sprintf("%s.%s/foo", appName, Config.GetAppsDomain())))
			})
		})
	})
})

func kubectl(args ...string) ([]byte, error) {
	cmd := exec.Command("kubectl", args...)
	cmd.Stderr = GinkgoWriter

	fmt.Fprintf(GinkgoWriter, "+ kubectl %s\n", strings.Join(args, " "))
	output, err := cmd.Output()
	return output, err
}

func kubectlGetRoute(namespace, routeGuid string) (route_crds.Route, error) {
	var route route_crds.Route

	output, err := kubectl("get", "route", routeGuid, "-n", namespace, "-o", "json")
	if err != nil {
		return route_crds.Route{}, err
	}

	err = json.Unmarshal(output, &route)
	if err != nil {
		return route_crds.Route{}, err
	}
	return route, nil
}
