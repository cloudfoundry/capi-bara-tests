package baras

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/assets"
	"github.com/cloudfoundry/capi-bara-tests/helpers/random_name"

	. "github.com/cloudfoundry/capi-bara-tests/bara_suite_helpers"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/services"
	. "github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func makeApp(spaceGUID string) app {
	var newApp app
	newApp.name = random_name.BARARandomName("APP")
	newApp.orgName = TestSetup.RegularUserContext().Org

	newApp.guid = CreateApp(newApp.name, spaceGUID, `{"foo":"bar"}`)
	newApp.packageGUID = CreatePackage(newApp.guid)

	uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), newApp.packageGUID)

	UploadPackage(uploadURL, assets.NewAssets().DoraZip)
	WaitForPackageToBeReady(newApp.packageGUID)

	buildGUID := StagePackage(newApp.packageGUID, Config.Lifecycle(), Config.GetRubyBuildpackName())
	WaitForBuildToStage(buildGUID)
	newApp.dropletGUID = GetDropletFromBuild(buildGUID)
	AssignDropletToApp(newApp.guid, newApp.dropletGUID)

	randomRoutePrefix := random_name.BARARandomName("ROUTE")
	newApp.route = fmt.Sprintf("%s.%s", randomRoutePrefix, Config.GetAppsDomain())

	StartApp(newApp.guid)
	return newApp
}

type app struct {
	guid        string
	name        string
	packageGUID string
	orgName     string
	dropletGUID string
	route       string
}

var _ = Describe("apply_manifest", func() {
	var (
		apps             []app
		broker           ServiceBroker
		serviceInstance  string
		spaceName        string
		spaceGUID        string
		domainGUID       string
		manifestToApply  string
		expectedManifest string

		applyEndpoint       string
		getManifestEndpoint string
	)

	BeforeEach(func() {
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		domainGUID = GetDomainGUIDFromName(Config.GetAppsDomain())
		apps = []app{makeApp(spaceGUID)}
	})

	AfterEach(func() {
		FetchRecentLogs(apps[0].guid, Config)
		for _, app := range apps {
			DeleteApp(app.guid)
		}
	})

	Describe("Applying a manifest to multiple apps", func() {
		BeforeEach(func() {
			apps = append(apps, makeApp(spaceGUID))
			applyEndpoint = fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)

			manifestToApply = fmt.Sprintf(`---
applications:
  - name: "%s"
    env: { foo: app0 }
    routes:
      - route: "%s"
  - name: "%s"
    env: { foo: app1 }
    routes:
      - route: "%s"
`,
				apps[0].name, apps[0].route,
				apps[1].name, apps[1].route,
			)
		})

		It("successfully updates both apps", func() {
			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i").Wait()
			Eventually(session).Should(Say("202 Accepted"))
			Eventually(session).Should(Exit(0))

			response := session.Out.Contents()
			PollJob(GetJobPath(response))

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0))

				session = cf.Cf("env", apps[0].name).Wait()
				Eventually(session).Should(Say("foo:\\s+app0"))
				Eventually(session).Should(Exit(0))

				session = cf.Cf("env", apps[1].name).Wait()
				Eventually(session).Should(Say("foo:\\s+app1"))
				Eventually(session).Should(Exit(0))

				By("setting the routes for both apps", func() {
					Eventually(func() *Session {
						return helpers.Curl(Config, Config.Protocol()+apps[0].route).Wait()
					}).Should(Say("Hi, I'm Dora!"))

					Eventually(func() *Session {
						return helpers.Curl(Config, Config.Protocol()+apps[1].route).Wait()
					}).Should(Say("Hi, I'm Dora!"))
				})
			})
		})
	})

	Describe("Applying a manifest to a single existing app", func() {
		BeforeEach(func() {
			applyEndpoint = fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)
			getManifestEndpoint = fmt.Sprintf("/v3/apps/%s/manifest", apps[0].guid)
		})

		Context("when routes are specified", func() {
			SkipOnK8s("Stacks on k8s aren't populated consistently yet, so get manifest endpoint has different behavior")
			BeforeEach(func() {
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  instances: 2
  memory: 300M
  buildpack: %s
  disk_quota: 1024M
  routes:
  - route: %s
  env: { foo: qux, snack: walnuts }
  health-check-type: http
  health-check-http-endpoint: /env
  timeout: 75
  metadata:
    labels:
      potato: yams
      juice: cherry
    annotations:
      contact: "jack@example.com diane@example.org"
      cougar: mellencamp
`, apps[0].name, Config.GetRubyBuildpackName(), apps[0].route)
				expectedManifest = fmt.Sprintf(`
applications:
- name: %s
  buildpacks:
  - %s
  env:
    foo: qux
    snack: walnuts
  routes:
  - route: %s
    protocol: http1
  metadata:
    labels:
      potato: yams
      juice: cherry
    annotations:
      contact: "jack@example.com diane@example.org"
      cougar: mellencamp
  stack: cflinuxfs4
  processes:
  - disk_quota: 1024M
    health-check-http-endpoint: /env
    health-check-type: http
    instances: 2
    log-rate-limit-per-second: -1
    memory: 300M
    timeout: 75
    type: web
  - disk_quota: 1024M
    health-check-type: process
    instances: 0
    log-rate-limit-per-second: -1
    memory: 256M
    type: worker
`, apps[0].name, Config.GetRubyBuildpackName(), apps[0].route)
			})

			It("successfully completes the job", func() {
				session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				session = cf.Cf("restage", apps[0].name).Wait(Config.CfPushTimeoutDuration())

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
					Expect(target).To(Exit(0), "failed targeting")

					session = cf.Cf("app", apps[0].name).Wait()
					Eventually(session).Should(Say("Showing health"))
					Eventually(session).Should(Say("routes:\\s+(?:%s.%s,\\s+)?%s", apps[0].name, Config.GetAppsDomain(), apps[0].route))
					Eventually(session).Should(Say("instances:\\s+.*?\\d+/2"))
					Eventually(session).Should(Exit(0))

					session = cf.Cf("env", apps[0].name).Wait()
					Eventually(session).Should(Say("foo:\\s+qux"))
					Eventually(session).Should(Say("snack:\\s+walnuts"))
					Eventually(session).Should(Exit(0))

					session = cf.Cf("get-health-check", apps[0].name).Wait()
					Eventually(session).Should(Say(`process\s+health check\s+endpoint\s+\(for http\)\s+invocation timeout`))
					Eventually(session).Should(Say(`web\s+http\s+\/env\s+1`))
					Eventually(session).Should(Say(`worker\s+process\s+1`))
					Eventually(session).Should(Exit(0))

					session = cf.Cf("curl", "-i", getManifestEndpoint)
					Expect(session.Wait()).To(Exit(0))
					Expect(session).To(Say("200 OK"))

					session = cf.Cf("curl", getManifestEndpoint)
					Expect(session.Wait()).To(Exit(0))
					response = session.Out.Contents()
					Expect(string(response)).To(MatchYAML(expectedManifest))
				})
			})
		})

		Context("when specifying no-route", func() {
			BeforeEach(func() {
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  no-route: true
`, apps[0].name)
			})

			It("removes existing routes from the app", func() {
				session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
					Expect(target).To(Exit(0), "failed targeting")

					session = cf.Cf("app", apps[0].name).Wait()
					Eventually(session).Should(Say("Showing health"))
					Eventually(session).Should(Say("routes:\\s*\\n"))
					Eventually(session).Should(Exit(0))
				})
			})
		})

		Context("when random-route is specified", func() {
			BeforeEach(func() {
				UnmapAllRoutes(apps[0].guid)

				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  random-route: true
`, apps[0].name)
			})

			It("successfully adds a random-route", func() {
				session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
					Expect(target).To(Exit(0), "failed targeting")

					session = cf.Cf("app", apps[0].name).Wait()
					Eventually(session).Should(Say("routes:\\s+%s-\\w+-\\w+-\\w+.%s", apps[0].name, Config.GetAppsDomain()))
				})
			})
		})

		Describe("sidecars", func() {
			SkipOnK8s("no sidecars available on k8s (yet)")
			BeforeEach(func() {
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
  environment_variables:
    WHAT_AM_I: MOTORCYCLE
  sidecars:
  - name: 'left_sidecar'
    command: WHAT_AM_I=LEFT_SIDECAR bundle exec rackup config.ru -o 0.0.0.0 -p 8081
    memory: 10M
    process_types: ['web']
  - name: 'right_sidecar'
    process_types: ['web']
    command: WHAT_AM_I=RIGHT_SIDECAR bundle exec rackup config.ru -o 0.0.0.0 -p 8082
    memory: 20M

`, apps[0].name)
			})

			Context("when the manifest defines some sidecars", func() {
				It("successfully runs the sidecar", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait()).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))
					appGUID := apps[0].guid

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
						Expect(target).To(Exit(0), "failed targeting")

						appEndpoint := fmt.Sprintf("/v2/apps/%s", appGUID)
						extraPortsJSON, err := json.Marshal(
							struct {
								Ports []int `json:"ports"`
							}{
								[]int{8080, 8081, 8082},
							},
						)
						Expect(err).NotTo(HaveOccurred())
						session := cf.Cf("curl", appEndpoint, "-X", "PUT", "-d", string(extraPortsJSON))
						Eventually(session).Should(Exit(0))

						appRoutePrefix := random_name.BARARandomName("ROUTE")
						sidecarRoutePrefix1 := random_name.BARARandomName("ROUTE")
						sidecarRoutePrefix2 := random_name.BARARandomName("ROUTE")

						CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, appRoutePrefix, 8080)
						CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, sidecarRoutePrefix1, 8081)
						CreateAndMapRouteWithPort(appGUID, spaceGUID, domainGUID, sidecarRoutePrefix2, 8082)

						session = cf.Cf("start", apps[0].name)
						Eventually(session).Should(Exit(0))

						Eventually(func() *Session {
							return helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()), "-f").Wait()
						}).Should(Exit(0))

						Eventually(func() *Session {
							return helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix1, Config.GetAppsDomain()), "-f").Wait()
						}).Should(Exit(0))

						Eventually(func() *Session {
							return helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix2, Config.GetAppsDomain()), "-f").Wait()
						}).Should(Exit(0))

						session = helpers.Curl(Config, fmt.Sprintf("%s.%s", appRoutePrefix, Config.GetAppsDomain()))
						Eventually(session).Should(Say("Hi, I'm Dora!"))

						session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", appRoutePrefix, Config.GetAppsDomain()))
						Eventually(session).ShouldNot(Say("MOTORCYCLE"))

						session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix1, Config.GetAppsDomain()))
						Eventually(session).Should(Say("LEFT_SIDECAR"))

						session = helpers.Curl(Config, fmt.Sprintf("%s.%s/env/WHAT_AM_I", sidecarRoutePrefix2, Config.GetAppsDomain()))
						Eventually(session).Should(Say("RIGHT_SIDECAR"))
					})

				})
			})
		})

		Describe("processes", func() {
			BeforeEach(func() {
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, apps[0].name)
			})

			Context("when the process exists already", func() {
				It("successfully completes the job", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait()).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", apps[0].name).Wait()
						Eventually(session).Should(Say("Showing health"))
						Eventually(session).Should(Say("instances:\\s+.*?\\d+/2"))
						Eventually(session).Should(Exit(0))

						processes := GetProcesses(apps[0].guid, apps[0].name)
						webProcessWithCommandRedacted := GetFirstProcessByType(processes, "web")
						webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
						Expect(webProcess.Command).To(Equal("new-command"))

						session = cf.Cf("get-health-check", apps[0].name).Wait()
						Eventually(session).Should(Say(fmt.Sprintf("Getting health check type for app %s in org %s / space %s as admin...", apps[0].name, apps[0].orgName, spaceName)))
						Eventually(session).Should(Say(`process\s+health check\s+endpoint\s+\(for http\)\s+invocation timeout`))
						Eventually(session).Should(Say(`web\s+http\s+\/env\s+1`))
						Eventually(session).Should(Say(`worker\s+process\s+1`))
						Eventually(session).Should(Exit(0))
					})
				})

				Context("and there are dependent sidecars and the process memory in the update is less than or equal to sidecar memory", func() {
					BeforeEach(func() {
						CreateSidecar("my_sidecar1", []string{"web"}, "while true; do echo helloworld; sleep 2; done", 100, apps[0].guid)
						manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    instances: 2
    memory: 100M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, apps[0].name)
					})
					It("fails the job and does not change the memory", func() {
						session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
						Expect(session.Wait()).To(Exit(0))
						response := session.Out.Contents()
						Expect(string(response)).To(ContainSubstring("202 Accepted"))

						jobPath := GetJobPath(response)
						PollJobAsFailed(jobPath)

						errors := GetJobErrors(jobPath)
						Expect(errors[0].Detail).To(ContainSubstring("The requested memory allocation is not large enough to run all of your sidecar processes"))
					})
				})
			})

			Context("when the process doesn't exist already", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: potato
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, apps[0].name)
				})

				It("creates the process and completes the job", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait()).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", apps[0].name).Wait()
						Eventually(session).Should(Say("type:\\s+potato"))
						Eventually(session).Should(Say("instances:\\s+0/2"))
						Eventually(session).Should(Exit(0))

						processes := GetProcesses(apps[0].guid, apps[0].name)
						potatoProcessWithCommandRedacted := GetFirstProcessByType(processes, "potato")
						potatoProcess := GetProcessByGuid(potatoProcessWithCommandRedacted.Guid)
						Expect(potatoProcess.Command).To(Equal("new-command"))

						session = cf.Cf("get-health-check", apps[0].name).Wait()
						Eventually(session).Should(Say("potato\\s+http\\s+/env"))
						Eventually(session).Should(Exit(0))
					})
				})
			})

			Context("when setting a new droplet", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: bean
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, apps[0].name)
				})

				It("does not remove existing processes", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait()).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", apps[0].name).Wait()
						Eventually(session).Should(Say("type:\\s+bean"))
						Eventually(session).Should(Say("instances:\\s+0/2"))
						Eventually(session).Should(Exit(0))
						AssignDropletToApp(apps[0].guid, apps[0].dropletGUID)

						processes := GetProcesses(apps[0].guid, apps[0].name)
						beanProcessWithCommandRedacted := GetFirstProcessByType(processes, "bean")
						beanProcess := GetProcessByGuid(beanProcessWithCommandRedacted.Guid)
						Expect(beanProcess.Command).To(Equal("new-command"))
					})
				})
			})
		})

		Describe("buildpacks", func() {
			Context("when multiple buildpacks are specified", func() {
				type Data struct {
					Buildpacks []string `json:"buildpacks"`
				}
				type Lifecycle struct {
					Data Data `json:"data"`
				}
				var app struct {
					Lifecycle Lifecycle `json:"lifecycle"`
				}

				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  buildpacks:
  - %s
  - %s
`, apps[0].name, Config.GetGoBuildpackName(), Config.GetRubyBuildpackName())
				})

				It("successfully adds the buildpacks", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait()).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("curl", fmt.Sprintf("v3/apps/%s", apps[0].guid)).Wait()
						err := json.Unmarshal(session.Out.Contents(), &app)
						Expect(err).ToNot(HaveOccurred())
						Eventually(app.Lifecycle.Data.Buildpacks).Should(Equal([]string{Config.GetGoBuildpackName(), Config.GetRubyBuildpackName()}))
						Eventually(session).Should(Exit(0))
					})
				})

				Context("when buildpack autodetection is specified", func() {
					SkipOnK8s("kpack buildpack autodetection doesn't currently populate docker dropets with lifecycle info")
					var currentDrop struct {
						Buildpacks []map[string]string `json:"buildpacks"`
					}

					BeforeEach(func() {
						manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  buildpacks: []
`, apps[0].name)
					})

					It("successfully updates the buildpacks to autodetect", func() {
						session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
						Expect(session.Wait()).To(Exit(0))
						response := session.Out.Contents()
						Expect(string(response)).To(ContainSubstring("202 Accepted"))

						PollJob(GetJobPath(response))

						workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
							target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
							Expect(target).To(Exit(0), "failed targeting")

							session = cf.Cf("curl", fmt.Sprintf("v3/apps/%s/droplets/current", apps[0].guid)).Wait()
							Eventually(session).Should(Exit(0))
							err := json.Unmarshal(session.Out.Contents(), &currentDrop)
							Expect(err).ToNot(HaveOccurred())
							Expect(currentDrop.Buildpacks).To(HaveLen(1))
							Expect(currentDrop.Buildpacks[0]["name"]).To(Equal(Config.GetRubyBuildpackName()))
							Expect(currentDrop.Buildpacks[0]["detect_output"]).To(ContainSubstring("ruby"))
						})
					})
				})
			})
		})

		Describe("services", func() {
			SkipOnK8s("requires real TLS for broker.Create()")
			BeforeEach(func() {
				apps = append(apps, makeApp(spaceGUID))

				By("Registering a Service Broker")
				broker = NewServiceBroker(
					random_name.BARARandomName("BRKR"),
					spaceGUID,
					domainGUID,
					assets.NewAssets().ServiceBroker,
					TestSetup,
				)
				broker.Push(Config)
				broker.Configure()
				broker.Create()
				broker.PublicizePlans()

				By("Creating a Service Instance")
				serviceInstance = random_name.BARARandomName("SVIN")
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstance).Wait()
				Expect(createService).To(Exit(0), "failed creating service")

				applyEndpoint = fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)
				manifestToApply = fmt.Sprintf(`---
applications:
  - name: "%s"
    services:
      - "%s"
`,
					apps[0].name, serviceInstance,
				)
			})

			AfterEach(func() {
				broker.Destroy()
			})

			It("successfully completes the job", func() {
				session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				session = cf.Cf("restage", apps[0].name).Wait(Config.CfPushTimeoutDuration())

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					target := cf.Cf("target", "-o", apps[0].orgName, "-s", spaceName).Wait()
					Expect(target).To(Exit(0), "failed targeting")

					session = cf.Cf("service", serviceInstance).Wait()
					Eventually(session).Should(Say("(?s)bound apps:.*%s", apps[0].name))
					Eventually(session).Should(Exit(0))
				})
			})
		})

		Context("with multiple processes and custom commands in the manifest", func() {
			SkipOnK8s("custom start commands dont get correct environment")
			It("starts all the processes", func() {
				appName := random_name.BARARandomName("APP")
				session := cf.Cf("create-app", appName)
				Expect(session.Wait()).To(Exit(0))

				session = cf.Cf("app", appName, "--guid")
				Expect(session.Wait()).To(Exit(0))
				appGUID := strings.TrimSpace(string(session.Out.Contents()))

				applyEndpoint = fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)
				manifestToApply := fmt.Sprintf(`
---
applications:
- name: %s
  buildpacks:
  - %s
  processes:
  - type: web
    instances: 1
    memory: 4096M
    disk_quota: 1024M
    health-check-type: http
    health-check-http-endpoint: '/'
  - type: logs
    instances: 1
    memory: 4096M
    command: "bundle exec rackup config.ru -o 0.0.0.0 -p $PORT"
    disk_quota: 1024M
    health-check-type: http
    health-check-http-endpoint: '/'
`, appName, Config.GetRubyBuildpackName())

				session = cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
				Expect(session.Wait()).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))

				session = cf.Cf("push", appName, "-p", assets.NewAssets().Dora)
				Expect(session.Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				waitForAllInstancesToStart(appGUID, 1)

				session = cf.Cf("app", appName).Wait()
				Eventually(session).Should(Say(`type:\s+logs`))
				Eventually(session).Should(Say(`sidecars:`))
				Eventually(session).Should(Say(`instances:\s+1/1`))

				DeleteApp(appGUID)
			})
		})
	})
})
