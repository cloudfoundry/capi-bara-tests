package baras

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests verify nginx routing of CC's internal API from the public `api`
// host. They do NOT test the mTLS listeners (9023/9025) — those are unreachable
// from this suite. See docs/internal/README.md.
var _ = Describe("Internal API nginx routing", func() {
	var client *http.Client
	var baseURL string

	baseRequest := func(method, path string) *http.Response {
		req, err := http.NewRequest(method, baseURL+path, nil)
		Expect(err).NotTo(HaveOccurred())
		// Sent without any Authorization header on purpose: the public
		// listener's `location /internal/v { return 403; }` fires in nginx
		// before proxying, so the outcome must not depend on a token.
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		return resp
	}

	BeforeEach(func() {
		transport := &http.Transport{}
		if Config.GetSkipSSLValidation() {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = &http.Client{Transport: transport}
		baseURL = Config.Protocol() + Config.GetApiEndpoint()
	})

	// All /internal/v* paths must be blocked on the public listener (nginx `location /internal/v { return 403 }`).
	Describe("internal endpoints (/internal/v*) are forbidden on the public listener", func() {
		forbidden := []struct {
			method string
			path   string
		}{
			{"POST", "/internal/v3/staging/some-guid/build_completed"},
			{"POST", "/internal/v4/apps/some-guid/crashed"},
			{"POST", "/internal/v4/apps/some-guid/readiness_changed"},
			{"POST", "/internal/v4/apps/some-guid/rescheduling"},
			{"GET", "/internal/v4/log_access/some-guid"},
			{"GET", "/internal/v5/syslog_drain_urls"},
			{"POST", "/internal/v4/tasks/some-guid/completed"},
			{"POST", "/internal/v4/droplets/some-guid/upload"},
			{"GET", "/internal/v4/staging_jobs/some-guid"},
			{"POST", "/internal/v4/buildpack_cache/some-stack/some-guid/upload"},
			{"GET", "/internal/v4/droplets/some-guid/some-checksum/download"},
			{"GET", "/internal/v4/asg_latest_update"},
			{"GET", "/internal/v4/metrics"},
		}

		for _, e := range forbidden {
			e := e
			It(fmt.Sprintf("returns 403 for %s %s", e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden),
					"expected nginx to 403 %s on the public listener, got %d", e.path, resp.StatusCode)
				Expect(string(body)).To(ContainSubstring("Forbidden"))
			})
		}
	})

	// Paths intentionally proxied through the public listener and authenticated
	// in-app. nginx must NOT 403 them.
	Describe("public-listener endpoints are proxied to CC (not nginx-forbidden)", func() {
		proxied := []struct {
			method         string
			path           string
			expectedStatus int
		}{
			{"GET", "/internal/apps/some-guid/ssh_access/0", http.StatusUnauthorized},
			{"GET", "/v2/buildpacks/some-guid/download", http.StatusNotFound}, // 404 because V2 is disabled
		}

		for _, e := range proxied {
			e := e
			It(fmt.Sprintf("returns %d (not nginx 403) for %s %s", e.expectedStatus, e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()

				Expect(resp.StatusCode).NotTo(Equal(http.StatusForbidden),
					"expected nginx to proxy %s to CC, but it returned 403", e.path)
				Expect(resp.StatusCode).To(Equal(e.expectedStatus))
			})
		}
	})

	// /staging/* downloads are 403'd on the public listener only when the blobstore is remote.
	Describe("/staging/* downloads are forbidden on the public listener with a remote blobstore", func() {
		staging := []string{
			"/staging/packages/some-guid",
			"/staging/v3/droplets/some-guid/download",
			"/staging/v3/buildpack_cache/some-stack/some-guid/download",
		}

		BeforeEach(func() {
			if Config.GetLocalBlobstore() {
				Skip("local blobstore configured; nginx proxies /staging/* to CC instead of returning 403")
			}
		})

		for _, path := range staging {
			path := path
			It(fmt.Sprintf("returns 403 for GET %s", path), func() {
				resp := baseRequest("GET", path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden),
					"expected nginx to 403 %s on the public listener, got %d", path, resp.StatusCode)
				Expect(string(body)).To(ContainSubstring("Forbidden"))
			})
		}
	})
})
