package app_helpers

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func GetAppGuid(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Eventually(cfApp).Should(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func printStartAppReport(appName string) {
	printAppReportBanner(fmt.Sprintf("***** APP REPORT: %s *****", appName))
}

func printEndAppReport(appName string) {
	printAppReportBanner(fmt.Sprintf("*** END APP REPORT: %s ***", appName))
}

func printAppReportBanner(announcement string) {
	startColor, endColor := getColor()
	sequence := strings.Repeat("*", len(announcement))
	fmt.Fprintf(ginkgo.GinkgoWriter,
		"\n\n%s%s\n%s\n%s%s\n",
		startColor,
		sequence,
		announcement,
		sequence,
		endColor)
}

func getColor() (string, string) {
	startColor := ""
	endColor := ""
	_, reporterConfig := ginkgo.GinkgoConfiguration()
	if !reporterConfig.NoColor {
		startColor = "\x1b[35m"
		endColor = "\x1b[0m"
	}

	return startColor, endColor
}
