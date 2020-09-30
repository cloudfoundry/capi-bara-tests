package bara_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cloudfoundry/custom-cats-reporters/honeycomb"
	"github.com/cloudfoundry/custom-cats-reporters/honeycomb/client"
	"github.com/honeycombio/libhoney-go"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"

	_ "github.com/cloudfoundry/capi-bara-tests/baras"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"
	. "github.com/onsi/ginkgo"
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
		ZipAsset(assetPaths.SleepySidecarBuildpack, assetPaths.SleepySidecarBuildpackZip)

		if Config.GetGcloudProjectName() != "" {
			gcloudCommand := exec.Command("gcloud", "container", "clusters", "get-credentials", Config.GetClusterName(), "--project", Config.GetGcloudProjectName(), "--zone", Config.GetClusterZone())
			session, err = gexec.Start(gcloudCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		}

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
		os.Remove(assets.NewAssets().BadDoraZip)
		os.Remove(assets.NewAssets().StaticfileZip)
		os.Remove(assets.NewAssets().CatnipZip)
		os.Remove(assets.NewAssets().SleepySidecarBuildpackZip)
	})

	rs := []Reporter{}

	if validationError == nil {
		if Config.GetArtifactsDirectory() != "" {
			helpers.EnableCFTrace(Config, "BARA")
			rs = append(rs, helpers.NewJUnitReporter(Config, "BARA"))
		}
	}

	reporterConfig := Config.GetReporterConfig()

	if reporterConfig.HoneyCombDataset != "" && reporterConfig.HoneyCombWriteKey != "" {
		honeyCombClient := client.New(libhoney.Config{
			WriteKey: reporterConfig.HoneyCombWriteKey,
			Dataset:  reporterConfig.HoneyCombDataset,
		})

		globalTags := map[string]interface{}{
			"run_id":  os.Getenv("RUN_ID"),
			"env_api": Config.GetApiEndpoint(),
		}

		honeyCombReporter := honeycomb.New(honeyCombClient)
		honeyCombReporter.SetGlobalTags(globalTags)
		honeyCombReporter.SetCustomTags(reporterConfig.CustomTags)

		rs = append(rs, honeyCombReporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "BARA", rs)
}
