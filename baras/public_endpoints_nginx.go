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

// These tests verify nginx routing of CC's public endpoints from the public `api`
// host. They do NOT test the mTLS listeners (9023/9025) — those are unreachable
// from this suite. See docs/internal/README.md.
var _ = Describe("Public API nginx routing", Label("no-cf-setup"), func() {
	var client *http.Client
	var baseURL string

	baseRequest := func(method, path string) *http.Response {
		req, err := http.NewRequest(method, baseURL+path, nil)
		Expect(err).NotTo(HaveOccurred())
		// Sent without any Authorization header: the public listener's catch-all
		// 404 fires before proxying, so the outcome must not depend on a token.
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		return resp
	}

	// gorouter sets a single-GUID X-Vcap-Request-Id; CC appends a second GUID separated by "::".
	// expectNginxBlocked / expectProxiedToCC use this to tell the two apart.

	expectNginxBlocked := func(resp *http.Response, body []byte) {
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		Expect(resp.Header.Get("X-Vcap-Request-Id")).NotTo(ContainSubstring("::"),
			"expected a nginx 404 (single-GUID X-Vcap-Request-Id), but CC appears to have handled the request")
		Expect(string(body)).To(ContainSubstring(`"CF-NotFound"`))
	}

	expectProxiedToCC := func(resp *http.Response, expectedStatus int) {
		Expect(resp.StatusCode).To(Equal(expectedStatus))
		Expect(resp.Header.Get("X-Vcap-Request-Id")).To(ContainSubstring("::"),
			"expected CC to proxy the request (double-GUID X-Vcap-Request-Id), but nginx appears to have blocked it")
	}

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

	// CF API paths are explicitly allow-listed in nginx.
	Describe("CF API endpoints are proxied to CC", func() {
		proxied := []struct {
			method         string
			path           string
			expectedStatus int
		}{
			// exact-match block (location = /, /v3)
			{"GET", "/", http.StatusOK},
			{"GET", "/v3", http.StatusOK},
			{"GET", "/healthz", http.StatusOK},
			// prefix-match block (location /v3/) — resource endpoints require auth
			{"GET", "/v3/", http.StatusOK},
			{"GET", "/v3/info", http.StatusOK},
			{"GET", "/v3/apps", http.StatusUnauthorized},
			{"GET", "/v3/apps/a-guid", http.StatusUnauthorized},
		}

		for _, e := range proxied {
			e := e
			It(fmt.Sprintf("proxies %s %s to CC (status %d)", e.method, e.path, e.expectedStatus), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				expectProxiedToCC(resp, e.expectedStatus)
			})
		}
	})

	// Unknown paths that don't match any explicit nginx allow block fall through to
	// the catch-all `location /` which returns a nginx-shaped 404 JSON body.
	Describe("unknown paths are blocked by the nginx catch-all", func() {
		unknown := []struct{ method, path string }{
			{"GET", "/unknown"},
			{"GET", "/some/random/path"},
			{"GET", "/v1/apps"},
			{"GET", "/v99"},
			{"GET", "/v3-unknown"},
		}

		for _, e := range unknown {
			e := e
			It(fmt.Sprintf("returns nginx 404 for %s %s", e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				expectNginxBlocked(resp, body)
			})
		}
	})

	// All /internal/* paths must be blocked by nginx's catch-all on the public
	// listener. nginx returns 404 with a CC-shaped JSON.
	Describe("internal endpoints (/internal/*) are blocked on the public listener", func() {
		blocked := []struct{ method, path string }{
			{"GET", "/internal"},
			{"GET", "/internal/"},
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

		for _, e := range blocked {
			e := e
			It(fmt.Sprintf("returns nginx 404 for %s %s", e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				expectNginxBlocked(resp, body)
			})
		}
	})

	Describe("internal api for ssh-proxy is proxied to CC (not nginx-blocked)", func() {
		It("proxies GET /internal/apps/some-guid/ssh_access/0 to CC", func() {
			resp := baseRequest("GET", "/internal/apps/some-guid/ssh_access/0")
			defer resp.Body.Close()
			expectProxiedToCC(resp, http.StatusUnauthorized)
		})
	})

	// V2 paths are blocked by nginx's catch-all except /v2/info, which is
	// explicitly allow-listed regardless of the temporary_enable_v2 flag (which is never set for capi-bara tests).
	Describe("v2 endpoints are blocked on the public listener", func() {
		blocked := []struct{ method, path string }{
			{"GET", "/v2"},
			{"GET", "/v2/"},
			{"GET", "/v2/apps"},
			{"GET", "/v2/buildpacks/some-guid/download"},
			{"GET", "/v2/organizations"},
			{"POST", "/v2/apps/some-guid/bits"},
		}

		for _, e := range blocked {
			e := e
			It(fmt.Sprintf("returns nginx 404 for %s %s", e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				expectNginxBlocked(resp, body)
			})
		}

		It("proxies GET /v2/info to CC", func() {
			resp := baseRequest("GET", "/v2/info")
			defer resp.Body.Close()
			expectProxiedToCC(resp, http.StatusOK)
		})
	})

	// /staging/* downloads are blocked by nginx's catch-all on the public listener
	// only when the blobstore is remote (no local blobstore configured).
	Describe("/staging/* downloads are blocked on the public listener with a remote blobstore", func() {
		staging := []struct{ method, path string }{
			{"GET", "/staging/packages/some-guid"},
			{"GET", "/staging/v3/droplets/some-guid/download"},
			{"GET", "/staging/v3/buildpack_cache/some-stack/some-guid/download"},
		}

		BeforeEach(func() {
			if Config.GetLocalBlobstore() {
				Skip("local blobstore configured; nginx proxies /staging/* to CC instead of blocking")
			}
		})

		for _, e := range staging {
			e := e
			It(fmt.Sprintf("returns nginx 404 for %s %s", e.method, e.path), func() {
				resp := baseRequest(e.method, e.path)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				expectNginxBlocked(resp, body)
			})
		}
	})
})
