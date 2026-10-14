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

// .cfignore is not honored when uploading a pre-built zip via the v3 API,
// so filter dev scaffolding here instead.
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
