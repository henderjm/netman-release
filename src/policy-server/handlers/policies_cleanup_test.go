package handlers_test

import (
	"encoding/json"
	lfakes "lib/fakes"
	"net/http"
	"net/http/httptest"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoliciesCleanup", func() {
	var (
		request       *http.Request
		handler       *handlers.PoliciesCleanup
		resp          *httptest.ResponseRecorder
		fakeStore     *fakes.Store
		fakeUAAClient *fakes.UAAClient
		fakeCCClient  *fakes.CCClient
		logger        *lagertest.TestLogger
		fakeMarshaler *lfakes.Marshaler
		allPolicies   []models.Policy
	)

	BeforeEach(func() {
		allPolicies = []models.Policy{{
			Source: models.Source{ID: "live-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{ID: "dead-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "udp",
				Port:     1234,
			},
		}, {
			Source: models.Source{ID: "live-guid", Tag: "tag"},
			Destination: models.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		fakeStore = &fakes.Store{}
		fakeUAAClient = &fakes.UAAClient{}
		fakeCCClient = &fakes.CCClient{}
		logger = lagertest.NewTestLogger("test")

		fakeMarshaler = &lfakes.Marshaler{}
		fakeMarshaler.MarshalStub = json.Marshal
		handler = &handlers.PoliciesCleanup{
			Logger:    logger,
			Store:     fakeStore,
			UAAClient: fakeUAAClient,
			CCClient:  fakeCCClient,
			Marshaler: fakeMarshaler,
		}

		resp = httptest.NewRecorder()
		request, _ = http.NewRequest("POST", "/networking/v0/external/policies/cleanup", nil)

		fakeUAAClient.GetTokenReturns("valid-token", nil)
		fakeStore.AllReturns(allPolicies, nil)
		fakeCCClient.GetAppGuidsReturns(map[string]interface{}{"live-guid": nil}, nil)
	})

	It("Returns the policies which should be cleaned up", func() {

		handler.ServeHTTP(resp, request)
		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetAppGuidsCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetAppGuidsArgsForCall(0)).To(Equal("valid-token"))
		Expect(fakeMarshaler.MarshalCallCount()).To(Equal(1))
		policyCleanup := struct {
			TotalPolicies int             `json:"total_policies"`
			Policies      []models.Policy `json:"policies"`
		}{len(allPolicies[1:]), allPolicies[1:]}
		Expect(fakeMarshaler.MarshalArgsForCall(0)).To(Equal(policyCleanup))

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Body.String()).To(MatchJSON(`{
			"total_policies":2,
			"policies": [
			{
				"source": {
					"id": "dead-guid",
					"tag": "tag"
				},
				"destination": {
					"id": "live-guid",
					"tag": "tag",
					"protocol": "udp",
					"port": 1234
				}
			},
			{
				"source": {
					"id": "live-guid",
					"tag": "tag"
				},
				"destination": {
					"id": "dead-guid",
					"tag": "tag",
					"protocol": "udp",
					"port": 1234
				}
			}
			]
		}
			`))

	})
})
