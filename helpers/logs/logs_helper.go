package logs

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/gomega/gexec"
)

func Tail(useLogCache bool, appName string) *gexec.Session {
	if useLogCache {
		return cf.Cf("tail", appName, "--lines", "125")
	}

	return cf.Cf("logs", "--recent", appName)
}

func TailFollow(useLogCache bool, appName string) *gexec.Session {
	if useLogCache {
		return cf.Cf("tail", "--follow", appName)
	}

	return cf.Cf("logs", appName)
}
