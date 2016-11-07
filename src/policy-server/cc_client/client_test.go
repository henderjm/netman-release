package cc_client_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"lib/fakes"
	"lib/testsupport"
	"net/http"
	"policy-server/cc_client"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/lager/lagertest"
)

const v3Apps = `{
  "pagination": {
    "total_results": 5,
    "total_pages": 1,
    "first": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    },
    "last": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    }
  },
  "resources": [
    {
      "guid": "live-app-1-guid"
    },
    {
      "guid": "live-app-2-guid"
    },
    {
      "guid": "live-app-3-guid"
    },
    {
      "guid": "live-app-4-guid"
    },
    {
      "guid": "live-app-5-guid"
    }
  ]
}`

var _ = Describe("Client", func() {
	var (
		client         *cc_client.Client
		fakeHTTPClient *fakes.HTTPClient
		logger         *lagertest.TestLogger
		expectedApps   map[string]interface{}
	)

	BeforeEach(func() {
		fakeHTTPClient = &fakes.HTTPClient{}
		logger = lagertest.NewTestLogger("test")
		client = &cc_client.Client{
			Host:       "some.url",
			HTTPClient: fakeHTTPClient,
			Logger:     logger,
		}
		expectedApps = map[string]interface{}{
			"live-app-1-guid": nil,
			"live-app-2-guid": nil,
			"live-app-3-guid": nil,
			"live-app-4-guid": nil,
			"live-app-5-guid": nil,
		}
	})

	Describe("GetAllAppGuids", func() {
		BeforeEach(func() {
			fakeHTTPClient.DoReturns(
				&http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte(v3Apps))),
				}, nil)
		})

		It("Returns the app guids", func() {
			apps, err := client.GetAllAppGUIDs("some-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			request := fakeHTTPClient.DoArgsForCall(0)
			Expect(request.Method).To(Equal("GET"))
			Expect(request.URL.String()).To(Equal("some.url/v3/apps"))
			authHeader := request.Header["Authorization"]
			Expect(authHeader).To(HaveLen(1))
			Expect(authHeader[0]).To(Equal("bearer some-token"))
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(Equal(expectedApps))
		})

		It("logs the request before sending", func() {
			_, err := client.GetAllAppGUIDs("some-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(logger).To(gbytes.Say("get_cc_apps"))
		})
	})

	Context("when the http client returns an error", func() {
		BeforeEach(func() {
			fakeHTTPClient.DoReturns(nil, errors.New("potato"))
		})

		It("returns a helpful error", func() {
			_, err := client.GetAllAppGUIDs("some-token")
			Expect(err).To(MatchError(ContainSubstring("http client: potato")))
		})
	})

	Context("when reading the body returns an error", func() {
		BeforeEach(func() {
			fakeHTTPClient.DoReturns(&http.Response{Body: &testsupport.BadReader{}}, nil)
		})

		It("returns a helpful error", func() {
			_, err := client.GetAllAppGUIDs("some-token")
			Expect(err).To(MatchError(ContainSubstring("read body: banana")))
		})
	})

	Context("when the response body is not valid json", func() {
		BeforeEach(func() {
			fakeHTTPClient.DoReturns(
				&http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader(`%%%%`)),
				}, nil)
		})

		It("returns a helpful error", func() {
			_, err := client.GetAllAppGUIDs("some-token")
			Expect(err).To(MatchError(ContainSubstring("unmarshal json: invalid character")))
		})
	})

	Context("when there are multiple pages", func() {
		BeforeEach(func() {
			v3AppsMultiplePages := `{
				"pagination": {
					"total_pages": 10
				}
			}`
			fakeHTTPClient.DoReturns(
				&http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte(v3AppsMultiplePages))),
				}, nil)

		})

		It("should immediately return an error", func() {
			_, err := client.GetAllAppGUIDs("some-token")
			Expect(err).To(MatchError("pagination support not yet implemented"))
		})
	})

	Context("if the response status code is not 200", func() {
		BeforeEach(func() {
			fakeHTTPClient.DoReturns(
				&http.Response{
					StatusCode: 418,
					Body:       ioutil.NopCloser(strings.NewReader("bad thing")),
				}, nil)

		})

		It("returns the response body in the error", func() {
			_, err := client.GetAllAppGUIDs("some-token")

			Expect(err).To(Equal(cc_client.BadCCResponse{
				StatusCode:     418,
				CCResponseBody: "bad thing",
			}))
		})
	})

})
