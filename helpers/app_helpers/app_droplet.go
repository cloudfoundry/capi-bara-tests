package app_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/capi-bara-tests/helpers/config"
	"github.com/cloudfoundry/capi-bara-tests/helpers/download"
	"github.com/cloudfoundry/capi-bara-tests/helpers/v3_helpers"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type AppDroplet struct {
	GUID         string
	AppGUID      string
	ProcessTypes *map[string]string
	Config       config.BaraConfig
}

func (droplet AppDroplet) MarshalJSON() ([]byte, error) {
	var apiDroplet struct {
		Relationships struct {
			App struct {
				Data struct {
					GUID string `json:"guid"`
				} `json:"data"`
			} `json:"app"`
		} `json:"relationships"`
		ProcessTypes *map[string]string `json:"process_types,omitempty"`
	}

	apiDroplet.Relationships.App.Data.GUID = droplet.AppGUID
	apiDroplet.ProcessTypes = droplet.ProcessTypes

	return json.Marshal(apiDroplet)
}

func (droplet *AppDroplet) Create() error {
	dropletBytes, err := json.Marshal(droplet)
	Expect(err).ToNot(HaveOccurred())

	session := cf.Cf("curl", "-X", "POST", "/v3/droplets", "-d", fmt.Sprintf("'%s'", string(dropletBytes)))
	Eventually(session).Should(Exit(0))

	// Populate the guid on the struct from the curl output
	var guidGrabber struct {
		GUID string `json:"guid"`
	}

	err = json.Unmarshal(session.Out.Contents(), &guidGrabber)
	Expect(err).ToNot(HaveOccurred())
	// Expect on this so if the api errors, we dont continue with a bad guid
	Expect(guidGrabber.GUID).ToNot(BeEmpty())

	droplet.GUID = guidGrabber.GUID
	return nil
}

func (droplet *AppDroplet) DownloadTo(downloadPath string) (string, error) {
	dropletTarballPath := fmt.Sprintf("%s.tar.gz", downloadPath)
	downloadURL := fmt.Sprintf("/v2/apps/%s/droplet/download", droplet.AppGUID)

	err := download.WithRedirect(downloadURL, dropletTarballPath, droplet.Config)
	return dropletTarballPath, err
}

func (droplet *AppDroplet) UploadFrom(uploadPath string) {
	token := v3_helpers.GetAuthToken()
	uploadURL := fmt.Sprintf("%s%s/v3/droplets/%s/upload", droplet.Config.Protocol(), droplet.Config.GetApiEndpoint(), droplet.GUID)
	bits := fmt.Sprintf(`bits=@%s`, uploadPath)
	curl := helpers.Curl(droplet.Config, "-v", uploadURL, "-X", "POST", "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
	Expect(curl).To(Exit(0))

	var dropletLink struct {
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	}
	bytes := curl.Out.Contents()
	json.Unmarshal(bytes, &dropletLink)
	pollingURL := dropletLink.Links.Self.Href
	fmt.Printf("\n%s\n", pollingURL)

	Eventually(func() *Session {
		return helpers.Curl(droplet.Config, pollingURL, "-H", fmt.Sprintf("Authorization: %s", token)).Wait()
	}).Should(Say("STAGED"))
}
