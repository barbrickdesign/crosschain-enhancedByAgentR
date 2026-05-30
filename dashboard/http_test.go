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
		Address:   "11111111111111111111111111111111",
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
		Address:   "11111111111111111111111111111111",
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

func TestOptionsEndpoint(t *testing.T) {
	h := NewHandler(NewService())

	req := httptest.NewRequest(http.MethodGet, "/api/options", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var options DashboardOptions
	if err := json.Unmarshal(rec.Body.Bytes(), &options); err != nil {
		t.Fatalf("failed to decode options: %v", err)
	}
	if len(options.Chains) == 0 || len(options.Connectors) == 0 || len(options.WalletTemplates) == 0 {
		t.Fatal("expected populated dashboard options")
	}
}

func TestDeleteWalletEndpoint(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "MetaMask",
		Address:   "0x0000000000000000000000000000000000000001",
		Chain:     "ETH",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("pair wallet failed: %v", err)
	}

	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/wallets/"+wallet.ID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/wallets", nil)
	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, listReq)

	var wallets []Wallet
	if err := json.Unmarshal(listRec.Body.Bytes(), &wallets); err != nil {
		t.Fatalf("failed to decode wallets response: %v", err)
	}
	if len(wallets) != 0 {
		t.Fatalf("expected no wallets after delete, got %d", len(wallets))
	}
}
