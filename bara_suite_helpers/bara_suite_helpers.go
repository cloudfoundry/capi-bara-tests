package bara_suite_helpers

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	"github.com/mholt/archiver"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/config"
	. "github.com/onsi/gomega"
)

const (
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	Config    BaraConfig
	TestSetup *workflowhelpers.ReproducibleTestSuiteSetup
	ScpPath   string
	SftpPath  string
)

// zipSkipNames lists top-level file/directory basenames that are pure
// dev-time scaffolding for our fixture apps and should never be shipped
// to Cloud Foundry. Filtering them here (instead of relying on .cfignore,
// which is not honored when uploading a pre-built zip via the v3 API)
// keeps the pushed package small and staging fast.
var zipSkipNames = map[string]bool{
	".git":                        true,
	".gitignore":                  true,
	".rspec":                      true,
	"README.md":                   true,
	"readme.md":                   true,
	"spec":                        true,
	"scripts":                     true,
	"stress":                      true,
	"get_instance_cookie_jars.sh": true,
}

func ZipAsset(assetPath, zipPath string) {
	files, err := ioutil.ReadDir(assetPath)
	Expect(err).NotTo(HaveOccurred())

	var fileNames []string
	for _, file := range files {
		if zipSkipNames[file.Name()] {
			continue
		}
		fileNames = append(fileNames, filepath.Join(assetPath, file.Name()))
	}

	err = archiver.Archive(fileNames, zipPath)
	Expect(err).NotTo(HaveOccurred())
}
