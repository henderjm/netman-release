package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

type UAAClient interface {
	GetToken() (string, error)
}

type CCClient interface {
	GetApps(token string, appsFilter []string)
}

type PoliciesCleanup struct {
	Logger    lager.Logger
	Store     store.Store
	Marshaler marshal.Marshaler
	CCclient  CCClient
	UAAclient UAAClient
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request, currentUserName string) {
	// use Warrent, exchange secret for token
	token, err := h.UAAclient.GetToken()

	// ask DB for all policies
	candidateApps, err := h.Store.Groups()
	if err != nil {
		// h.Logger.Error("store-list-policies-failed", err)
		// w.WriteHeader(http.StatusInternalServerError)
		// w.Write([]byte(`{"error": "database read failed"}`))
		return
	}

	aliveApps, err := h.CCClient.GetApps(token)

	deadApps := getDeadApps(candidateApps, aliveApps)

	ret, err := h.Store.DeleteByGroupd(deadApps)

	//return

}
