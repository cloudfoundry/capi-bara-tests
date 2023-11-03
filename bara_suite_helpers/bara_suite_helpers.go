package bara_suite_helpers

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	"github.com/mholt/archiver"
	"io/ioutil"

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

func ZipAsset(assetPath, zipPath string) {
	files, err := ioutil.ReadDir(assetPath)
	Expect(err).NotTo(HaveOccurred())

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, assetPath+"/"+file.Name())
	}

	err = archiver.Zip.Make(zipPath, fileNames)
	Expect(err).NotTo(HaveOccurred())
}
