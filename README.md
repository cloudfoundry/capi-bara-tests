# CAPI BARAS - Banausic Acceptance & Regression Avoidance Suite
This suite exercises a [Cloud Foundry](https://github.com/cloudfoundry/cf-deployment)
deployment using the `cf` CLI and `curl`.
It is scoped to testing user-facing,
end-to-end features, 
focusing on failure paths and edge-cases in the Cloud Controller

Any tests with a Cloud Controller focus
that are being removed from the cf-acceptance-tests repo are
good candidates for being moved here.

For more info on how to write BARA tests, please see the 
[CATS README](https://github.com/cloudfoundry/cf-acceptance-tests).

## Test Setup
### Prerequisites for running BARAS

- Same as for [CATS](https://github.com/cloudfoundry/cf-acceptance-tests),
  with the following exceptions:

- Check out a copy of `capi-bara-tests`
  and make sure that it is added to your `$GOPATH`.
  The recommended way to do this is to run:

  ```bash
  go get -d github.com/cloudfoundry/capi-bara-tests
  ```

  You will receive a warning:
  `no buildable Go source files`.
  This can be ignored, as there is only test code in the package.

### Updating `go` dependencies
- Same as for [CATS](https://github.com/cloudfoundry/cf-acceptance-tests).

## Test Configuration
- Same as for [CATS](https://github.com/cloudfoundry/cf-acceptance-tests).

```bash
cat > integration_config.json <<EOF
{
  "api": "api.bosh-lite.com",
  "apps_domain": "bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "skip_ssl_validation": true,
}
EOF
export CONFIG=$PWD/integration_config.json
```

## Test Execution
To execute all test groups, run the following from the root directory of cf-acceptance-tests:
```bash
./bin/test
```

##### Parallel execution
To execute all test groups, and have tests run in parallel across four processes one would run:

```bash
./bin/test -nodes=4
```

Be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.


##### Focusing Test Groups
If you are already familiar with CATs you probably know that there are many test groups. You may not wish to run all the tests in all contexts, and sometimes you may want to focus individual test groups to pinpoint a failure. To execute a specific group of acceptance tests, e.g. `routing/`, edit your [`integration_config.json`](#test-configuration) file and set all `include_*` values to `false` except for `include_routing` then run the following:

```bash
./bin/test
```

To execute tests in a single file use an `FDescribe` block around the tests in that file:
```go
var _ = BackendCompatibilityDescribe("Backend Compatibility", func() {
  FDescribe("Focused tests", func() { // Add this line here
  // ... rest of file
  }) // Close here
})

```

The test group names correspond to directory names.

##### Verbose Output
To see verbose output from `ginkgo`, use the `-v` flag.

```bash
./bin/test -v
```

You can of course combine the `-v` flag with the `-nodes=N` flag.

## Contributing

- See [CATS](https://github.com/cloudfoundry/cf-acceptance-tests).

### Code Conventions

- See [CATS](https://github.com/cloudfoundry/cf-acceptance-tests).

