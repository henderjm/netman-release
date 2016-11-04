package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

//go:generate counterfeiter -o ../fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetAppGuids(token string) (map[string]interface{}, error)
}

type PoliciesCleanup struct {
	Logger    lager.Logger
	Store     store.Store
	UAAClient uaaClient
	CCClient  ccClient
	Marshaler marshal.Marshaler
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	policies, err := h.Store.All()
	if err != nil {
		panic(err)
	}

	////	// h.Logger.Error("store-list-policies-failed", err)
	////	// w.WriteHeader(http.StatusInternalServerError)
	////	// w.Write([]byte(`{"error": "database read failed"}`))
	////	return
	////}

	token, err := h.UAAClient.GetToken()
	if err != nil {
		panic(err)
	}

	ccAppGuids, err := h.CCClient.GetAppGuids(token)
	if err != nil {
		panic(err)
	}

	stalePolicies := getStalePolicies(policies, ccAppGuids)

	//ret, err := h.Store.DeleteByGroup(staleAppGuids)

	policyCleanup := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []models.Policy `json:"policies"`
	}{len(stalePolicies), stalePolicies}

	bytes, err := h.Marshaler.Marshal(policyCleanup)
	if err != nil {
		panic(err)
		// h.Logger.Error("marshal-failed", err)
		// w.WriteHeader(http.StatusInternalServerError)
		// w.Write([]byte(`{"error": "database marshaling failed"}`))
		// return
	}
	w.Write(bytes)

	w.WriteHeader(http.StatusOK)

}

func getStalePolicies(policyList []models.Policy, ccList map[string]interface{}) (ret []models.Policy) {
	for _, p := range policyList {
		srcApp := p.Source.ID
		if _, ok := ccList[srcApp]; !ok {
			ret = append(ret, p)
			continue
		}

		dstApp := p.Destination.ID
		if _, ok := ccList[dstApp]; !ok {
			ret = append(ret, p)
		}
	}

	return ret
}
