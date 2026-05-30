package dashboard

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func NewHandler(service *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(dashboardHTML))
	})

	mux.HandleFunc("/api/chains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, service.ListChains())
	})

	mux.HandleFunc("/api/wallets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, service.ListWallets())
		case http.MethodPost:
			var req PairWalletRequest
			if err := decodeJSON(r, &req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			wallet, err := service.PairWallet(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusCreated, wallet)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/routes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, service.ListRoutes(r.URL.Query().Get("from"), r.URL.Query().Get("to")))
	})

	mux.HandleFunc("/api/bridge/sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, service.ListBridgeSessions())
		case http.MethodPost:
			var req CreateBridgeSessionRequest
			if err := decodeJSON(r, &req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			session, err := service.CreateBridgeSession(req)
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, ErrWalletNotFound) || errors.Is(err, ErrBridgeRouteNotFound) {
					status = http.StatusNotFound
				}
				http.Error(w, err.Error(), status)
				return
			}
			writeJSON(w, http.StatusCreated, session)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func decodeJSON(r *http.Request, out interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body contains multiple JSON values")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Crosschain Wallet Connect Dashboard</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 2rem; background: #0f1220; color: #e6ebff; }
    h1, h2 { margin: 0.5rem 0; }
    .grid { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); }
    .card { background: #161b2f; padding: 1rem; border-radius: 12px; border: 1px solid #273258; }
    input, select, button { width: 100%; margin-top: 0.4rem; margin-bottom: 0.8rem; padding: 0.6rem; border-radius: 8px; border: 1px solid #39508f; background: #0f1430; color: #fff; }
    button { background: #546dff; border: none; cursor: pointer; font-weight: bold; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #2a3358; padding: 0.4rem; text-align: left; font-size: 0.9rem; }
    .muted { color: #a6b0d7; font-size: 0.85rem; }
  </style>
</head>
<body>
  <h1>Crosschain Wallet Connect Dashboard</h1>
  <p class="muted">Pair mobile and PC wallets in one place and create Firedancer-aware bridge sessions.</p>

  <div class="grid">
    <section class="card">
      <h2>Pair Wallet</h2>
      <form id="wallet-form">
        <label>Name</label><input required name="name" placeholder="Phantom Mobile" />
        <label>Address</label><input required name="address" placeholder="Wallet address" />
        <label>Chain</label><select required name="chain" id="chain"></select>
        <label>Platform</label>
        <select required name="platform">
          <option value="mobile">mobile</option>
          <option value="pc">pc</option>
        </select>
        <label>Connector</label><input required name="connector" value="walletconnect" />
        <button type="submit">Pair Wallet</button>
      </form>
    </section>

    <section class="card">
      <h2>Create Bridge Session</h2>
      <form id="bridge-form">
        <label>From Wallet</label><select required name="from_wallet_id" id="wallet"></select>
        <label>Destination Chain</label><select required name="destination_chain" id="destination"></select>
        <label>Asset</label><input required name="asset" placeholder="USDC" />
        <label>Amount</label><input required name="amount" placeholder="12.5" />
        <label><input type="checkbox" name="prefer_firedancer" /> Prefer Firedancer route</label>
        <button type="submit">Plan Bridge</button>
      </form>
    </section>
  </div>

  <div class="grid" style="margin-top: 1rem;">
    <section class="card">
      <h2>Paired Wallets</h2>
      <table>
        <thead><tr><th>ID</th><th>Name</th><th>Chain</th><th>Platform</th></tr></thead>
        <tbody id="wallet-rows"></tbody>
      </table>
    </section>
    <section class="card">
      <h2>Bridge Sessions</h2>
      <table>
        <thead><tr><th>ID</th><th>From</th><th>To</th><th>Route</th><th>Status</th></tr></thead>
        <tbody id="session-rows"></tbody>
      </table>
    </section>
  </div>

  <script>
    async function api(path, options = {}) {
      const res = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...options });
      if (!res.ok) throw new Error(await res.text());
      if (res.status === 204) return null;
      return res.json();
    }

    async function loadChains() {
      const chains = await api('/api/chains');
      const chainSelect = document.getElementById('chain');
      const destination = document.getElementById('destination');
      chainSelect.innerHTML = '';
      destination.innerHTML = '';
      for (const chain of chains) {
        chainSelect.add(new Option(chain, chain));
        destination.add(new Option(chain, chain));
      }
    }

    async function loadWallets() {
      const wallets = await api('/api/wallets');
      const walletRows = document.getElementById('wallet-rows');
      const walletSelect = document.getElementById('wallet');
      walletRows.innerHTML = '';
      walletSelect.innerHTML = '';
      for (const wallet of wallets) {
        const row = document.createElement('tr');
        row.innerHTML = '<td>' + wallet.id + '</td><td>' + wallet.name + '</td><td>' + wallet.chain + '</td><td>' + wallet.platform + '</td>';
        walletRows.appendChild(row);
        walletSelect.add(new Option(wallet.name + ' (' + wallet.chain + ')', wallet.id));
      }
    }

    async function loadSessions() {
      const sessions = await api('/api/bridge/sessions');
      const rows = document.getElementById('session-rows');
      rows.innerHTML = '';
      for (const session of sessions) {
        const row = document.createElement('tr');
        const firedancer = session.route.firedancer_optimized ? ' + firedancer' : '';
        row.innerHTML = '<td>' + session.id + '</td><td>' + session.from_wallet.chain + '</td><td>' + session.route.to_chain + '</td><td>' + session.route.bridge + firedancer + '</td><td>' + session.status + '</td>';
        rows.appendChild(row);
      }
    }

    document.getElementById('wallet-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      const form = new FormData(event.target);
      const payload = Object.fromEntries(form.entries());
      try {
        await api('/api/wallets', { method: 'POST', body: JSON.stringify(payload) });
        event.target.reset();
        await loadWallets();
      } catch (error) {
        alert(error.message);
      }
    });

    document.getElementById('bridge-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      const form = new FormData(event.target);
      const payload = Object.fromEntries(form.entries());
      payload.prefer_firedancer = form.get('prefer_firedancer') === 'on';
      try {
        await api('/api/bridge/sessions', { method: 'POST', body: JSON.stringify(payload) });
        await loadSessions();
      } catch (error) {
        alert(error.message);
      }
    });

    (async function init() {
      await loadChains();
      await loadWallets();
      await loadSessions();
    })();
  </script>
</body>
</html>`
