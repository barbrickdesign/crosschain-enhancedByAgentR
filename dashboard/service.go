package dashboard

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrInvalidWalletRequest = errors.New("invalid wallet request")
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrBridgeRouteNotFound  = errors.New("bridge route not found")
)

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
	mu       sync.RWMutex
	wallets  map[string]Wallet
	sessions []BridgeSession
	routes   []BridgeRoute
	chains   []string
	nextID   int
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
	}
}

func (s *Service) PairWallet(req PairWalletRequest) (Wallet, error) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	chain := strings.ToUpper(strings.TrimSpace(req.Chain))
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	connector := strings.TrimSpace(req.Connector)

	if name == "" || address == "" || chain == "" || connector == "" {
		return Wallet{}, fmt.Errorf("%w: missing required fields", ErrInvalidWalletRequest)
	}
	if platform != "mobile" && platform != "pc" {
		return Wallet{}, fmt.Errorf("%w: platform must be mobile or pc", ErrInvalidWalletRequest)
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

	chains := make([]string, len(s.chains))
	copy(chains, s.chains)
	return chains
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

	s.mu.Lock()
	defer s.mu.Unlock()

	fromWallet, ok := s.wallets[walletID]
	if !ok {
		return BridgeSession{}, ErrWalletNotFound
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
