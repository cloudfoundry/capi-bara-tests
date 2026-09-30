package bara_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	"github.com/onsi/ginkgo/v2/types"
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

	// Per-spec timing marker printed to stdout so CI logs contain the
	// data even when the JUnit XML artifact is not accessible. Format
	// is deliberately grep-friendly:
	//
	//   [BARA_SPEC_TIMING] node=3 state=passed time=45.20s num_attempts=1 spec="deployments Canary deployments deploys an app, transitions to pause, is continued and then deploys successfully"
	//
	// Grep with:  grep BARA_SPEC_TIMING <ci-log>
	// then sort on the time= field to see the distribution across all
	// parallel processes.
	ReportAfterEach(func(report SpecReport) {
		fmt.Printf("[BARA_SPEC_TIMING] node=%d state=%s time=%.2fs num_attempts=%d spec=%q\n",
			GinkgoParallelProcess(),
			report.State,
			report.RunTime.Seconds(),
			report.NumAttempts,
			report.FullText())
	})

	// At the end of the whole run (aggregated across all parallel
	// processes) print a top-40 slowest-specs summary block. This is
	// the primary artefact for prioritising optimisations: whichever
	// spec sits at position #1 is the current critical-path anchor on
	// the slowest node.
	ReportAfterSuite("bara timings summary", func(report Report) {
		specs := make([]SpecReport, 0, len(report.SpecReports))
		for _, s := range report.SpecReports {
			if s.LeafNodeType != types.NodeTypeIt {
				continue
			}
			specs = append(specs, s)
		}
		sort.Slice(specs, func(i, j int) bool {
			return specs[i].RunTime > specs[j].RunTime
		})

		n := 40
		if len(specs) < n {
			n = len(specs)
		}

		fmt.Println()
		fmt.Println("================ BARA TOP SLOWEST SPECS ================")
		fmt.Printf("Suite wall time: %s | Total specs: %d | Showing top %d\n",
			report.RunTime.Round(time.Second), len(specs), n)
		fmt.Println("--------------------------------------------------------")
		for i := 0; i < n; i++ {
			s := specs[i]
			fmt.Printf("%3d. %7.1fs [%s]%s  %s\n",
				i+1,
				s.RunTime.Seconds(),
				s.State,
				retryTag(s),
				s.FullText())
		}
		fmt.Println("========================================================")
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

// retryTag returns a compact marker for specs that Ginkgo retried
// (i.e. flaked and were re-run). Empty string when the spec ran once.
func retryTag(s SpecReport) string {
	if s.NumAttempts > 1 {
		return fmt.Sprintf(" (RETRIED×%d)", s.NumAttempts)
	}
	return ""
}
