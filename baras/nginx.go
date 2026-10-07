package baras

import (
	"crypto/tls"
	"net/http"
	"regexp"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("nginx config logic", Label("no-cf-setup"), func() {
	var client *http.Client
	var baseURL string

	BeforeEach(func() {
		if client != nil {
			return
		}
		transport := &http.Transport{}
		if Config.GetSkipSSLValidation() {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = &http.Client{Transport: transport}
		baseURL = Config.Protocol() + Config.GetApiEndpoint()
	})

	postRequest := func(path string) *http.Response {
		req, err := http.NewRequest("POST", baseURL+path, nil)
		Expect(err).NotTo(HaveOccurred())
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		return resp
	}

	Describe("hitting /v3/packages/:guid/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			resp := postRequest("/v3/packages/literally-any-guid/upload?bits_path='some/path'")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnprocessableEntity))
		})
	})

	Describe("hitting /v3/buildpacks/:guid/bits with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			resp := postRequest("/v3/buildpacks/literally-any-guid/upload?bits_path='some/path'")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnprocessableEntity))
		})
	})

	Describe("hitting /v3/droplets/:guid/upload with invalid parameters", func() {
		It("returns 422 Unprocessable Entity", func() {
			resp := postRequest("/v3/droplets/literally-any-guid/upload?bits_path='some/path'")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnprocessableEntity))
		})
	})

	Describe("Response headers", func() {
		It("does not contain 'Server: nginx'", func() {
			req, err := http.NewRequest("GET", baseURL+"/v3/info", nil)
			Expect(err).NotTo(HaveOccurred())
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.Header.Get("Server")).NotTo(MatchRegexp(regexp.QuoteMeta("nginx") + `\/?\d+(\.\d+){0,2}`))
		})
	})
})
