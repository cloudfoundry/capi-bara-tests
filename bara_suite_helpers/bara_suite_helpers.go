package bara_suite_helpers

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/mholt/archiver"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/config"
	"github.com/cloudfoundry/capi-bara-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

func SkipOnK8s(reason string) {
	BeforeEach(func() {
		if Config.RunningOnK8s() {
			Skip(fmt.Sprintf(skip_messages.SkipK8sMessage, reason))
		}
	})
}

func SkipOnVMs(reason string) {
	BeforeEach(func() {
		if Config.RunningOnK8s() {
			Skip(fmt.Sprintf(skip_messages.SkipVMsMessage, reason))
		}
	})
}
