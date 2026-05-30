package dashboard

import "testing"

func TestPairWallet(t *testing.T) {
	svc := NewService()

	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "MetaMask",
		Address:   "0x0000000000000000000000000000000000000001",
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
		Address:   "11111111111111111111111111111111",
		Chain:     "sol",
		Platform:  "tablet",
		Connector: "walletconnect",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPairWalletInvalidAddress(t *testing.T) {
	svc := NewService()
	_, err := svc.PairWallet(PairWalletRequest{
		Name:      "Bad Wallet",
		Address:   "not-an-address",
		Chain:     "eth",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err == nil {
		t.Fatal("expected invalid address error")
	}
}

func TestCreateBridgeSessionPrefersFiredancer(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "Phantom",
		Address:   "11111111111111111111111111111111",
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

func TestCreateBridgeSessionRejectsUnsupportedAsset(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "Phantom",
		Address:   "11111111111111111111111111111111",
		Chain:     "sol",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("pair wallet failed: %v", err)
	}

	_, err = svc.CreateBridgeSession(CreateBridgeSessionRequest{
		FromWalletID:     wallet.ID,
		DestinationChain: "ETH",
		Asset:            "DOGE",
		Amount:           "1",
	})
	if err == nil {
		t.Fatal("expected asset validation error")
	}
}

func TestUnpairWalletRemovesSessions(t *testing.T) {
	svc := NewService()
	wallet, err := svc.PairWallet(PairWalletRequest{
		Name:      "Phantom",
		Address:   "11111111111111111111111111111111",
		Chain:     "sol",
		Platform:  "mobile",
		Connector: "walletconnect",
	})
	if err != nil {
		t.Fatalf("pair wallet failed: %v", err)
	}

	if _, err := svc.CreateBridgeSession(CreateBridgeSessionRequest{
		FromWalletID:     wallet.ID,
		DestinationChain: "ETH",
		Asset:            "USDC",
		Amount:           "1",
	}); err != nil {
		t.Fatalf("create bridge session failed: %v", err)
	}

	if err := svc.UnpairWallet(wallet.ID); err != nil {
		t.Fatalf("unpair wallet failed: %v", err)
	}
	if len(svc.ListWallets()) != 0 {
		t.Fatal("expected wallets to be removed")
	}
	if len(svc.ListBridgeSessions()) != 0 {
		t.Fatal("expected sessions to be removed with the wallet")
	}
}

func TestDashboardOptionsReturnsDataCopies(t *testing.T) {
	svc := NewService()
	options := svc.DashboardOptions()
	if len(options.Chains) == 0 || len(options.Connectors) == 0 || len(options.WalletTemplates) == 0 {
		t.Fatal("expected dashboard options to be populated")
	}

	options.Chains[0] = "BROKEN"
	refetched := svc.DashboardOptions()
	if refetched.Chains[0] == "BROKEN" {
		t.Fatal("expected dashboard options to return copies")
	}
}
