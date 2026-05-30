package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWalletEndpoints(t *testing.T) {
	h := NewHandler(NewService())

	pairPayload := PairWalletRequest{
		Name:      "Phantom",
		Address:   "sol-addr",
		Chain:     "SOL",
		Platform:  "mobile",
		Connector: "walletconnect",
	}
	body, _ := json.Marshal(pairPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/wallets", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/wallets", nil)
	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}

	var wallets []Wallet
	if err := json.Unmarshal(listRec.Body.Bytes(), &wallets); err != nil {
		t.Fatalf("failed to decode wallets response: %v", err)
	}
	if len(wallets) != 1 {
		t.Fatalf("expected one wallet, got %d", len(wallets))
	}
}

func TestBridgeSessionEndpoint(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "Phantom",
		Address:   "sol-addr",
		Chain:     "SOL",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("pair wallet failed: %v", err)
	}

	h := NewHandler(svc)
	payload := CreateBridgeSessionRequest{
		FromWalletID:     wallet.ID,
		DestinationChain: "ETH",
		Asset:            "USDC",
		Amount:           "10",
		PreferFiredancer: true,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/bridge/sessions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var session BridgeSession
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("failed to decode session: %v", err)
	}
	if !session.Route.FiredancerOptimized {
		t.Fatal("expected firedancer optimized route")
	}
}
