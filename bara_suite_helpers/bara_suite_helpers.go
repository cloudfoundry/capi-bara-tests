package bara_suite_helpers

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/config"
)

const (
	CF_JAVA_TIMEOUT      = 10 * time.Minute
	V3_PROCESS_TIMEOUT   = 45 * time.Second
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	Config    BaraConfig
	TestSetup *workflowhelpers.ReproducibleTestSuiteSetup
	ScpPath   string
	SftpPath  string
)
