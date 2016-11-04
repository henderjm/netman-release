package cc_client_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"lib/fakes"
	"net/http"
	"policy-server/cc_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/lagertest"
)

var _ = Describe("Client", func() {
	var (
		client       *cc_client.Client
		httpClient   *fakes.HTTPClient
		logger       *lagertest.TestLogger
		expectedApps []string
	)

	Describe("CheckToken", func() {
		BeforeEach(func() {
			httpClient = &fakes.HTTPClient{}
			logger = lagertest.NewTestLogger("test")
			client = &cc_client.Client{
				Host:       "some.url",
				Name:       "test",
				Secret:     "test",
				HTTPClient: httpClient,
				Logger:     logger,
			}
			expectedApps = []string{
				"live-app-1-guid",
				"live-app-2-guid",
				"live-app-3-guid",
				"live-app-4-guid",
				"live-app-5-guid",
			}
			httpClient.DoReturns(makePage(1, expectedApps), nil)

		})

		It("Returns the apps", func() {
			apps, err := client.GetAllAppGUIDs("xxx")
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(Equal(expectedApps))
		})

	})

	Context("when there is one page", func() {

	})

	Context("when there are multiple pages", func() {
		BeforeEach(func() {
			httpClient.DoReturns(makePage(2,
				[]string{
					"live-app-1-guid",
					"live-app-2-guid",
				}), nil)
		})

		It("should immediately return an error", func() {
			//Expect(err).To(MatchError("pagination support not yet implemented: you have too many apps!"))
		})

	})
})

type AppResult struct {
	GUID string `json:"guid"`
}

type AppsResult struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
	} `json: "pagination"`
	Resources []AppResult `json:"resources"`
}

func makePage(totalPages int, guids []string) *http.Response {
	appsResult := AppsResult{}
	appsResult.Pagination.TotalPages = totalPages
	for _, guid := range guids {
		appsResult.Resources = append(appsResult.Resources, AppResult{GUID: guid})
	}

	jsonBytes, _ := json.Marshal(appsResult)
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader(jsonBytes)),
	}
}
