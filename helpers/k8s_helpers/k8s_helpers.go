package k8s_helpers

import (
	"encoding/json"

	route_crds "code.cloudfoundry.org/cf-k8s-networking/routecontroller/api/v1alpha1"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/gomega"
)

func Kubectl(args ...string) ([]byte, error) {
	cmdstarter := commandstarter.NewCommandStarter()
	session, err := cmdstarter.Start(commandreporter.NewCommandReporter(), "kubectl", args...)
	if err != nil {
		return nil, err
	}
	return session.Wait().Out.Contents(), nil
}

// TODO: rename this l8r
func KubectlSession(args ...string) *gexec.Session {
	cmdstarter := commandstarter.NewCommandStarter()
	session, err := cmdstarter.Start(commandreporter.NewCommandReporter(), "kubectl", args...)
	if err != nil {
		panic(err)
	}
	return session
}

func KubectlGetRoute(namespace, routeGuid string) (route_crds.Route, error) {
	var route route_crds.Route

	session := KubectlSession("get", "route", routeGuid, "-n", namespace, "-o", "json")
	Expect(session.Wait("3m")).To(gexec.Exit(0), "Failed to get route resource from Kubernetes")

	err := json.Unmarshal(session.Out.Contents(), &route)
	if err != nil {
		return route_crds.Route{}, err
	}
	return route, nil
}
