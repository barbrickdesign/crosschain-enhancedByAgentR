package dashboard

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/cosmos/btcutil/bech32"
	"github.com/ethereum/go-ethereum/common"
	solanago "github.com/gagliardetto/solana-go"
)

var (
	ErrInvalidWalletRequest = errors.New("invalid wallet request")
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrBridgeRouteNotFound  = errors.New("bridge route not found")
)

var hexAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{1,64}$`)

type Wallet struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	Platform  string `json:"platform"`
	Connector string `json:"connector"`
}

type PairWalletRequest struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	Platform  string `json:"platform"`
	Connector string `json:"connector"`
}

type ConnectorOption struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Platforms []string `json:"platforms"`
	Summary   string   `json:"summary,omitempty"`
}

type WalletTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Chain       string `json:"chain"`
	Platform    string `json:"platform"`
	Connector   string `json:"connector"`
	Description string `json:"description,omitempty"`
}

type DashboardOptions struct {
	Chains          []string            `json:"chains"`
	AssetsByChain   map[string][]string `json:"assets_by_chain"`
	Connectors      []ConnectorOption   `json:"connectors"`
	WalletTemplates []WalletTemplate    `json:"wallet_templates"`
	AddressExamples map[string]string   `json:"address_examples"`
}

type BridgeRoute struct {
	ID                  string `json:"id"`
	FromChain           string `json:"from_chain"`
	ToChain             string `json:"to_chain"`
	Bridge              string `json:"bridge"`
	FiredancerOptimized bool   `json:"firedancer_optimized"`
	PerformanceProfile  string `json:"performance_profile,omitempty"`
}

type CreateBridgeSessionRequest struct {
	FromWalletID     string `json:"from_wallet_id"`
	DestinationChain string `json:"destination_chain"`
	Asset            string `json:"asset"`
	Amount           string `json:"amount"`
	PreferFiredancer bool   `json:"prefer_firedancer"`
}

type BridgeSession struct {
	ID         string      `json:"id"`
	FromWallet Wallet      `json:"from_wallet"`
	Route      BridgeRoute `json:"route"`
	Asset      string      `json:"asset"`
	Amount     string      `json:"amount"`
	Status     string      `json:"status"`
}

type Service struct {
	mu              sync.RWMutex
	wallets         map[string]Wallet
	sessions        []BridgeSession
	routes          []BridgeRoute
	chains          []string
	assetsByChain   map[string][]string
	connectors      []ConnectorOption
	walletTemplates []WalletTemplate
	addressExamples map[string]string
	nextID          int
}

func NewService() *Service {
	return &Service{
		wallets: map[string]Wallet{},
		routes: []BridgeRoute{
			{ID: "pipeline-default", FromChain: "*", ToChain: "*", Bridge: "crosschain-pipeline"},
			{ID: "pipeline-solana-firedancer-out", FromChain: "SOL", ToChain: "*", Bridge: "crosschain-pipeline", FiredancerOptimized: true, PerformanceProfile: "firedancer"},
			{ID: "pipeline-solana-firedancer-in", FromChain: "*", ToChain: "SOL", Bridge: "crosschain-pipeline", FiredancerOptimized: true, PerformanceProfile: "firedancer"},
		},
		chains: []string{"BTC", "ETH", "MATIC", "BNB", "AVAX", "FTM", "SOL", "ATOM", "INJ", "XPLA", "LUNA", "APTOS", "SUI"},
		assetsByChain: map[string][]string{
			"BTC":   {"BTC"},
			"ETH":   {"ETH", "USDC", "USDT", "DAI"},
			"MATIC": {"MATIC", "USDC", "USDT"},
			"BNB":   {"BNB", "USDC", "USDT"},
			"AVAX":  {"AVAX", "USDC", "USDT"},
			"FTM":   {"FTM", "USDC", "USDT"},
			"SOL":   {"SOL", "USDC", "USDT", "BONK"},
			"ATOM":  {"ATOM", "USDC"},
			"INJ":   {"INJ", "USDT"},
			"XPLA":  {"XPLA", "USDC"},
			"LUNA":  {"LUNA", "USDC"},
			"APTOS": {"APT", "USDC"},
			"SUI":   {"SUI", "USDC"},
		},
		connectors: []ConnectorOption{
			{ID: "walletconnect", Label: "WalletConnect", Platforms: []string{"mobile", "pc"}, Summary: "Works well for most mobile wallets and QR pairing."},
			{ID: "deep-link", Label: "Deep Link", Platforms: []string{"mobile"}, Summary: "Use the wallet app directly on the same phone."},
			{ID: "browser-extension", Label: "Browser Extension", Platforms: []string{"pc"}, Summary: "Best for desktop wallets in Chrome or Brave."},
			{ID: "injected", Label: "Injected Provider", Platforms: []string{"pc"}, Summary: "Use wallets already injected into the browser."},
			{ID: "manual", Label: "Manual", Platforms: []string{"mobile", "pc"}, Summary: "Save a wallet you will connect outside the dashboard."},
		},
		walletTemplates: []WalletTemplate{
			{ID: "phantom-mobile", Name: "Phantom", Chain: "SOL", Platform: "mobile", Connector: "deep-link", Description: "Simple Solana mobile setup."},
			{ID: "phantom-desktop", Name: "Phantom", Chain: "SOL", Platform: "pc", Connector: "browser-extension", Description: "Phantom browser wallet for desktop."},
			{ID: "metamask-mobile", Name: "MetaMask", Chain: "ETH", Platform: "mobile", Connector: "walletconnect", Description: "Popular mobile EVM wallet."},
			{ID: "metamask-desktop", Name: "MetaMask", Chain: "ETH", Platform: "pc", Connector: "browser-extension", Description: "Desktop EVM wallet extension."},
			{ID: "trust-wallet", Name: "Trust Wallet", Chain: "BNB", Platform: "mobile", Connector: "walletconnect", Description: "Mobile-first multi-chain wallet."},
			{ID: "coinbase-wallet", Name: "Coinbase Wallet", Chain: "ETH", Platform: "mobile", Connector: "walletconnect", Description: "Easy mobile onboarding for EVM chains."},
			{ID: "keplr-mobile", Name: "Keplr", Chain: "ATOM", Platform: "mobile", Connector: "deep-link", Description: "Cosmos mobile wallet."},
			{ID: "keplr-desktop", Name: "Keplr", Chain: "ATOM", Platform: "pc", Connector: "browser-extension", Description: "Cosmos browser wallet."},
			{ID: "martian-mobile", Name: "Martian", Chain: "APTOS", Platform: "mobile", Connector: "deep-link", Description: "Aptos mobile wallet."},
			{ID: "suiet-mobile", Name: "Suiet", Chain: "SUI", Platform: "mobile", Connector: "deep-link", Description: "Sui mobile wallet."},
		},
		addressExamples: map[string]string{
			"BTC":   "bc1qexample...",
			"ETH":   "0x1234...abcd",
			"MATIC": "0x1234...abcd",
			"BNB":   "0x1234...abcd",
			"AVAX":  "0x1234...abcd",
			"FTM":   "0x1234...abcd",
			"SOL":   "7YyUhZf5...Solana",
			"ATOM":  "atom1example...",
			"INJ":   "inj1example...",
			"XPLA":  "xpla1example...",
			"LUNA":  "terra1example...",
			"APTOS": "0x1",
			"SUI":   "0x1",
		},
	}
}

func (s *Service) PairWallet(req PairWalletRequest) (Wallet, error) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	chain := strings.ToUpper(strings.TrimSpace(req.Chain))
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	connector := strings.ToLower(strings.TrimSpace(req.Connector))

	if name == "" || address == "" || chain == "" || connector == "" {
		return Wallet{}, fmt.Errorf("%w: missing required fields", ErrInvalidWalletRequest)
	}
	if platform != "mobile" && platform != "pc" {
		return Wallet{}, fmt.Errorf("%w: platform must be mobile or pc", ErrInvalidWalletRequest)
	}
	if !s.supportsChain(chain) {
		return Wallet{}, fmt.Errorf("%w: unsupported chain %s", ErrInvalidWalletRequest, chain)
	}
	if !s.supportsConnector(connector, platform) {
		return Wallet{}, fmt.Errorf("%w: unsupported connector %s for %s", ErrInvalidWalletRequest, connector, platform)
	}
	if err := validateAddress(chain, address); err != nil {
		return Wallet{}, fmt.Errorf("%w: %v", ErrInvalidWalletRequest, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	wallet := Wallet{
		ID:        fmt.Sprintf("wallet-%d", s.nextID),
		Name:      name,
		Address:   address,
		Chain:     chain,
		Platform:  platform,
		Connector: connector,
	}
	s.wallets[wallet.ID] = wallet
	return wallet, nil
}

func (s *Service) ListWallets() []Wallet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Wallet, 0, len(s.wallets))
	for _, wallet := range s.wallets {
		result = append(result, wallet)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func (s *Service) ListChains() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return copyStringSlice(s.chains)
}

func (s *Service) DashboardOptions() DashboardOptions {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assetsByChain := make(map[string][]string, len(s.assetsByChain))
	for chain, assets := range s.assetsByChain {
		assetsByChain[chain] = copyStringSlice(assets)
	}

	connectors := make([]ConnectorOption, len(s.connectors))
	for i, connector := range s.connectors {
		connectors[i] = ConnectorOption{
			ID:        connector.ID,
			Label:     connector.Label,
			Platforms: copyStringSlice(connector.Platforms),
			Summary:   connector.Summary,
		}
	}

	templates := make([]WalletTemplate, len(s.walletTemplates))
	copy(templates, s.walletTemplates)

	addressExamples := make(map[string]string, len(s.addressExamples))
	for chain, example := range s.addressExamples {
		addressExamples[chain] = example
	}

	return DashboardOptions{
		Chains:          copyStringSlice(s.chains),
		AssetsByChain:   assetsByChain,
		Connectors:      connectors,
		WalletTemplates: templates,
		AddressExamples: addressExamples,
	}
}

func (s *Service) ListRoutes(fromChain, toChain string) []BridgeRoute {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routesFor(strings.ToUpper(strings.TrimSpace(fromChain)), strings.ToUpper(strings.TrimSpace(toChain)))
}

func (s *Service) CreateBridgeSession(req CreateBridgeSessionRequest) (BridgeSession, error) {
	walletID := strings.TrimSpace(req.FromWalletID)
	asset := strings.ToUpper(strings.TrimSpace(req.Asset))
	amount := strings.TrimSpace(req.Amount)
	toChain := strings.ToUpper(strings.TrimSpace(req.DestinationChain))
	if walletID == "" || asset == "" || amount == "" || toChain == "" {
		return BridgeSession{}, fmt.Errorf("%w: missing required fields", ErrInvalidWalletRequest)
	}
	if !s.supportsChain(toChain) {
		return BridgeSession{}, fmt.Errorf("%w: unsupported destination chain %s", ErrInvalidWalletRequest, toChain)
	}
	if !isPositiveAmount(amount) {
		return BridgeSession{}, fmt.Errorf("%w: amount must be a positive number", ErrInvalidWalletRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fromWallet, ok := s.wallets[walletID]
	if !ok {
		return BridgeSession{}, ErrWalletNotFound
	}
	if fromWallet.Chain == toChain {
		return BridgeSession{}, fmt.Errorf("%w: source and destination chains must be different", ErrInvalidWalletRequest)
	}
	if !containsString(s.assetsByChain[fromWallet.Chain], asset) {
		return BridgeSession{}, fmt.Errorf("%w: %s is not a supported asset for %s", ErrInvalidWalletRequest, asset, fromWallet.Chain)
	}

	route, err := s.selectRoute(fromWallet.Chain, toChain, req.PreferFiredancer)
	if err != nil {
		return BridgeSession{}, err
	}

	s.nextID++
	session := BridgeSession{
		ID:         fmt.Sprintf("bridge-%d", s.nextID),
		FromWallet: fromWallet,
		Route:      route,
		Asset:      asset,
		Amount:     amount,
		Status:     "planned",
	}
	s.sessions = append(s.sessions, session)
	return session, nil
}

func (s *Service) ListBridgeSessions() []BridgeSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]BridgeSession, len(s.sessions))
	copy(result, s.sessions)
	return result
}

func (s *Service) UnpairWallet(walletID string) error {
	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return fmt.Errorf("%w: wallet id is required", ErrInvalidWalletRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.wallets[walletID]; !ok {
		return ErrWalletNotFound
	}
	delete(s.wallets, walletID)

	filtered := s.sessions[:0]
	for _, session := range s.sessions {
		if session.FromWallet.ID != walletID {
			filtered = append(filtered, session)
		}
	}
	s.sessions = filtered
	return nil
}

func (s *Service) selectRoute(fromChain, toChain string, preferFiredancer bool) (BridgeRoute, error) {
	candidates := s.routesFor(fromChain, toChain)
	if len(candidates) == 0 {
		return BridgeRoute{}, ErrBridgeRouteNotFound
	}

	if preferFiredancer {
		for _, candidate := range candidates {
			if candidate.FiredancerOptimized {
				return candidate, nil
			}
		}
	}
	return candidates[0], nil
}

func (s *Service) routesFor(fromChain, toChain string) []BridgeRoute {
	candidates := make([]BridgeRoute, 0, len(s.routes))
	for _, route := range s.routes {
		matchesFrom := route.FromChain == "*" || route.FromChain == fromChain
		matchesTo := route.ToChain == "*" || route.ToChain == toChain
		if matchesFrom && matchesTo {
			candidates = append(candidates, route)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		leftSpecificity := specificity(candidates[i])
		rightSpecificity := specificity(candidates[j])
		if leftSpecificity == rightSpecificity {
			return candidates[i].ID < candidates[j].ID
		}
		return leftSpecificity > rightSpecificity
	})
	return candidates
}

func specificity(route BridgeRoute) int {
	score := 0
	if route.FromChain != "*" {
		score++
	}
	if route.ToChain != "*" {
		score++
	}
	if route.FiredancerOptimized {
		score++
	}
	return score
}

func (s *Service) supportsChain(chain string) bool {
	return containsString(s.chains, chain)
}

func (s *Service) supportsConnector(connector, platform string) bool {
	for _, option := range s.connectors {
		if option.ID == connector && containsString(option.Platforms, platform) {
			return true
		}
	}
	return false
}

func validateAddress(chain, address string) error {
	switch chain {
	case "SOL":
		if _, err := solanago.PublicKeyFromBase58(address); err != nil {
			return errors.New("enter a valid Solana address")
		}
	case "ETH", "MATIC", "BNB", "AVAX", "FTM":
		if !common.IsHexAddress(address) {
			return errors.New("enter a valid EVM address")
		}
	case "BTC":
		if _, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams); err != nil {
			if _, testErr := btcutil.DecodeAddress(address, &chaincfg.TestNet3Params); testErr != nil {
				return errors.New("enter a valid Bitcoin address")
			}
		}
	case "ATOM":
		if err := validateBech32Prefix(address, "atom"); err != nil {
			return errors.New("enter a valid Cosmos address")
		}
	case "INJ":
		if err := validateBech32Prefix(address, "inj"); err != nil {
			return errors.New("enter a valid Injective address")
		}
	case "XPLA":
		if err := validateBech32Prefix(address, "xpla"); err != nil {
			return errors.New("enter a valid XPLA address")
		}
	case "LUNA":
		if err := validateBech32Prefix(address, "terra"); err != nil {
			return errors.New("enter a valid Terra address")
		}
	case "APTOS", "SUI":
		if !hexAddressPattern.MatchString(address) {
			return errors.New("enter a valid hex address starting with 0x")
		}
	}
	return nil
}

func validateBech32Prefix(address, prefix string) error {
	hrp, _, err := bech32.Decode(strings.ToLower(address), 1023)
	if err != nil {
		return err
	}
	if hrp != prefix {
		return fmt.Errorf("expected %s prefix", prefix)
	}
	return nil
}

func isPositiveAmount(amount string) bool {
	value, ok := new(big.Rat).SetString(amount)
	return ok && value.Sign() > 0
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func copyStringSlice(values []string) []string {
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}
