package dashboard

import "testing"

func TestPairWallet(t *testing.T) {
	svc := NewService()

	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "MetaMask",
		Address:   "0x123",
		Chain:     "eth",
		Platform:  "pc",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wallet.Chain != "ETH" {
		t.Fatalf("expected ETH chain, got %s", wallet.Chain)
	}

	wallets := svc.ListWallets()
	if len(wallets) != 1 {
		t.Fatalf("expected 1 wallet, got %d", len(wallets))
	}
}

func TestPairWalletInvalidPlatform(t *testing.T) {
	svc := NewService()
	_, err := svc.PairWallet(PairWalletRequest{
		Name:      "Wallet",
		Address:   "addr",
		Chain:     "sol",
		Platform:  "tablet",
		Connector: "walletconnect",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateBridgeSessionPrefersFiredancer(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "Phantom",
		Address:   "sol-address",
		Chain:     "sol",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("pair wallet failed: %v", err)
	}

	session, err := svc.CreateBridgeSession(CreateBridgeSessionRequest{
		FromWalletID:     wallet.ID,
		DestinationChain: "ETH",
		Asset:            "USDC",
		Amount:           "1",
		PreferFiredancer: true,
	})
	if err != nil {
		t.Fatalf("create bridge session failed: %v", err)
	}
	if !session.Route.FiredancerOptimized {
		t.Fatal("expected firedancer optimized route")
	}
}
