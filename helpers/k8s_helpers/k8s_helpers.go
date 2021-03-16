package k8s_helpers

import (
	"encoding/json"

	route_crds "code.cloudfoundry.org/cf-k8s-networking/routecontroller/api/v1alpha1"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
)

func Kubectl(args ...string) ([]byte, error) {
	cmdstarter := commandstarter.NewCommandStarter()
	session, err := cmdstarter.Start(commandreporter.NewCommandReporter(), "kubectl", args...)

	return session.Wait().Out.Contents(), err
}

func KubectlGetRoute(namespace, routeGuid string) (route_crds.Route, error) {
	var route route_crds.Route

	output, err := Kubectl("get", "route", routeGuid, "-n", namespace, "-o", "json")
	if err != nil {
		return route_crds.Route{}, err
	}

	err = json.Unmarshal(output, &route)
	if err != nil {
		return route_crds.Route{}, err
	}
	return route, nil
}
