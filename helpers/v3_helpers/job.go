package v3_helpers

import (
	"encoding/json"
	"regexp"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"

	. "github.com/onsi/gomega"
)

func GetJobPath(response []byte) string {
	r, err := regexp.Compile(`Location:.*(/v3/jobs/[\w-]*)`)
	Expect(err).ToNot(HaveOccurred())
	return r.FindStringSubmatch(string(response))[1]
}

func PollJob(jobPath string) {
	Eventually(func() string {
		jobSession := cf.Cf("curl", "-f", jobPath)
		return string(jobSession.Wait().Out.Contents())
	}).Should(ContainSubstring("COMPLETE"))
}

func PollJobAsFailed(jobPath string) {
	Eventually(func() string {
		jobSession := cf.Cf("curl", "-f", jobPath)
		return string(jobSession.Wait().Out.Contents())
	}).Should(ContainSubstring("FAILED"))
}

type jobError struct {
	Detail string `json:"detail"`
	Title  string `json:"title"`
	Code   int    `json:"code"`
}

func GetJobErrors(jobPath string) []jobError {
	session := cf.Cf("curl", "-f", jobPath).Wait()
	var job struct {
		GUID   string     `json:"guid"`
		Errors []jobError `json:"errors"`
	}

	bytes := session.Wait().Out.Contents()
	err := json.Unmarshal(bytes, &job)
	Expect(err).NotTo(HaveOccurred())
	return job.Errors
}
