package bara_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"

	_ "github.com/cloudfoundry/capi-bara-tests/baras"

	. "github.com/cloudfoundry/capi-bara-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
		t.FailNow()
	}

	var _ = SynchronizedBeforeSuite(func() []byte {
		installedVersion, err := GetInstalledCliVersionString()

		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")
		fmt.Println("Running BARAs with CF CLI version ", installedVersion)

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")

		buildCmd := exec.Command("go", "build", "-o", "bin/catnip")
		buildCmd.Dir = "assets/catnip"
		buildCmd.Env = append(os.Environ(),
			"GOOS=linux",
			"GOARCH=amd64",
		)
		buildCmd.Stdout = GinkgoWriter
		buildCmd.Stderr = GinkgoWriter

		session, err := gexec.Start(buildCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))

		buildCmd = exec.Command("go", "build", "-o", "../sidecar-dependent/sidecar")
		buildCmd.Dir = "assets/sidecar"
		buildCmd.Env = append(os.Environ(),
			"GOOS=linux",
			"GOARCH=amd64",
		)

		session, err = gexec.Start(buildCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))

		assetPaths := assets.NewAssets()
		ZipAsset(assetPaths.Dora, assetPaths.DoraZip)
		ZipAsset(assetPaths.BadDora, assetPaths.BadDoraZip)
		ZipAsset(assetPaths.Staticfile, assetPaths.StaticfileZip)
		ZipAsset(assetPaths.Catnip, assetPaths.CatnipZip)
		ZipAsset(assetPaths.PythonWithoutProcfile, assetPaths.PythonWithoutProcfileZip)
		ZipAsset(assetPaths.SleepySidecarBuildpack, assetPaths.SleepySidecarBuildpackZip)

		if Config.GetGcloudProjectName() != "" {
			gcloudCommand := exec.Command("gcloud", "container", "clusters", "get-credentials", Config.GetClusterName(), "--project", Config.GetGcloudProjectName(), "--zone", Config.GetClusterZone())
			session, err = gexec.Start(gcloudCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		}

		return []byte{}
	}, func([]byte) {})

	BeforeEach(func() {
		SetDefaultEventuallyTimeout(Config.DefaultTimeoutDuration())
		SetDefaultEventuallyPollingInterval(1 * time.Second)

		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)
		TestSetup.Setup()
	})

	AfterEach(func() {
		if TestSetup != nil {
			TestSetup.Teardown()
		}
	})

	SynchronizedAfterSuite(func() {}, func() {
		os.Remove(assets.NewAssets().DoraZip)
		os.Remove(assets.NewAssets().BadDoraZip)
		os.Remove(assets.NewAssets().StaticfileZip)
		os.Remove(assets.NewAssets().CatnipZip)
		os.Remove(assets.NewAssets().PythonWithoutProcfileZip)
		os.Remove(assets.NewAssets().SleepySidecarBuildpackZip)
	})

	_, rc := GinkgoConfiguration()

	if validationError == nil {
		if Config.GetArtifactsDirectory() != "" {
			helpers.EnableCFTrace(Config, "BARA")
			rc.JUnitReport = filepath.Join(Config.GetArtifactsDirectory(), fmt.Sprintf("junit-%s-%d.xml", "BARA", GinkgoParallelProcess()))
		}
	}

	RunSpecs(t, "BARA", rc)
}
