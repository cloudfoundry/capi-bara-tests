package v3_helpers

import (
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
)

func GetAuthToken() string {
	session := cf.CfRedact("bearer", "oauth-token")
	bytes := session.Wait().Out.Contents()
	return strings.TrimSpace(string(bytes))
}
