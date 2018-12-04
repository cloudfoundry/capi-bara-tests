package bara_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/mholt/archiver"

	_ "github.com/cloudfoundry/capi-bara-tests/processes"
	_ "github.com/cloudfoundry/capi-bara-tests/rolling_deployments"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.33.1"

func TestBARA(t *testing.T) {
	RegisterFailHandler(Fail)

	var validationError error

	Config, validationError = config.NewBaraConfig(os.Getenv("CONFIG"))
	if validationError != nil {
		defer GinkgoRecover()
		fmt.Println("Invalid configuration.  ")
		fmt.Println(validationError)
		fmt.Println("Please fix the contents of $CONFIG:\n  " + os.Getenv("CONFIG") + "\nbefore proceeding.")
		t.Fail()
	}

	var _ = SynchronizedBeforeSuite(func() []byte {
		installedVersion, err := GetInstalledCliVersionString()

		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")
		fmt.Println("Running BARAs with CF CLI version ", installedVersion)

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")

		buildCmd := exec.Command("go", "build", "-o", "bin/catnip")
		buildCmd.Dir = "assets/catnip"
		buildCmd.Env = []string{
			fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")),
			fmt.Sprintf("GOROOT=%s", os.Getenv("GOROOT")),
			"GOOS=linux",
			"GOARCH=amd64",
		}
		buildCmd.Stdout = GinkgoWriter
		buildCmd.Stderr = GinkgoWriter

		err = buildCmd.Run()
		Expect(err).NotTo(HaveOccurred())

		doraFiles, err := ioutil.ReadDir(assets.NewAssets().Dora)
		Expect(err).NotTo(HaveOccurred())

		var doraFileNames []string
		for _, doraFile := range doraFiles {
			doraFileNames = append(doraFileNames, assets.NewAssets().Dora+"/"+doraFile.Name())
		}

		err = archiver.Zip.Make(assets.NewAssets().DoraZip, doraFileNames)
		Expect(err).NotTo(HaveOccurred())

		return []byte{}
	}, func([]byte) {
		SetDefaultEventuallyTimeout(Config.DefaultTimeoutDuration())
		SetDefaultEventuallyPollingInterval(1 * time.Second)

		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)
		TestSetup.Setup()
	})

	SynchronizedAfterSuite(func() {
		if TestSetup != nil {
			TestSetup.Teardown()
		}
	}, func() {
		os.Remove(assets.NewAssets().DoraZip)
	})

	rs := []Reporter{}

	if validationError == nil {
		if Config.GetArtifactsDirectory() != "" {
			helpers.EnableCFTrace(Config, "BARA")
			rs = append(rs, helpers.NewJUnitReporter(Config, "BARA"))
		}
	}

	RunSpecsWithDefaultAndCustomReporters(t, "BARA", rs)
}
