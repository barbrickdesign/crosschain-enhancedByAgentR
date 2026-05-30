package dashboard

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
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

	mux.HandleFunc("/api/options", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, service.DashboardOptions())
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

	mux.HandleFunc("/api/wallets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		walletID := strings.TrimPrefix(r.URL.Path, "/api/wallets/")
		if walletID == "" || strings.Contains(walletID, "/") {
			http.NotFound(w, r)
			return
		}
		if err := service.UnpairWallet(walletID); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrWalletNotFound) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
    :root {
      color-scheme: dark;
      --bg: #0c1124;
      --panel: #141b35;
      --panel-soft: #10172f;
      --border: #2b3a6f;
      --text: #edf2ff;
      --muted: #a8b4dd;
      --accent: #6d83ff;
      --accent-strong: #8ea0ff;
      --success: #22c55e;
      --error: #ef4444;
      --warning: #f59e0b;
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: Arial, sans-serif; background: linear-gradient(180deg, #0a0f22, #101936 35%, #0c1124 100%); color: var(--text); }
    main { max-width: 1180px; margin: 0 auto; padding: 1.25rem; }
    h1, h2, h3, p { margin-top: 0; }
    .hero { margin-bottom: 1rem; }
    .hero h1 { margin-bottom: 0.45rem; }
    .muted { color: var(--muted); font-size: 0.95rem; line-height: 1.5; }
    .grid { display: grid; gap: 1rem; grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .card { background: rgba(20, 27, 53, 0.96); border: 1px solid var(--border); border-radius: 18px; padding: 1rem; box-shadow: 0 18px 40px rgba(0, 0, 0, 0.18); }
    .card h2 { margin-bottom: 0.35rem; }
    .step { display: inline-flex; align-items: center; justify-content: center; width: 1.7rem; height: 1.7rem; margin-right: 0.45rem; border-radius: 999px; background: rgba(109, 131, 255, 0.18); color: var(--accent-strong); font-weight: 700; }
    .form-grid { display: grid; gap: 0.85rem; grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .field, .full { display: flex; flex-direction: column; gap: 0.35rem; }
    .full { grid-column: 1 / -1; }
    label { font-weight: 700; font-size: 0.95rem; }
    input, select, button { width: 100%; border-radius: 12px; border: 1px solid #405289; background: #0d1530; color: var(--text); font: inherit; }
    input, select { padding: 0.9rem; min-height: 3rem; }
    input[type="checkbox"] { width: auto; min-height: auto; margin-right: 0.45rem; accent-color: var(--accent); }
    .checkbox-row { display: flex; align-items: center; gap: 0.55rem; margin-top: 0.2rem; font-size: 0.95rem; }
    button { padding: 0.95rem 1rem; min-height: 3.1rem; border: none; background: linear-gradient(135deg, var(--accent), var(--accent-strong)); color: white; font-weight: 700; cursor: pointer; }
    button.secondary { background: #22325f; }
    button.danger { background: #542534; }
    button:disabled { opacity: 0.55; cursor: not-allowed; }
    .helper { color: var(--muted); font-size: 0.84rem; line-height: 1.45; }
    .banner { margin-bottom: 1rem; padding: 0.9rem 1rem; border-radius: 14px; border: 1px solid var(--border); background: rgba(19, 26, 50, 0.95); display: none; }
    .banner.info { display: block; border-color: #4257a1; }
    .banner.success { display: block; border-color: rgba(34, 197, 94, 0.6); }
    .banner.error { display: block; border-color: rgba(239, 68, 68, 0.65); }
    .hero-points, .route-list { margin: 0; padding-left: 1.1rem; color: var(--muted); }
    .hero-points li, .route-list li { margin-bottom: 0.45rem; }
    .panel-stack { display: grid; gap: 1rem; }
    .table-wrap { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; min-width: 540px; }
    th, td { border-bottom: 1px solid #273563; padding: 0.75rem 0.5rem; text-align: left; vertical-align: middle; font-size: 0.92rem; }
    th { color: var(--muted); font-size: 0.82rem; text-transform: uppercase; letter-spacing: 0.04em; }
    .pill { display: inline-flex; align-items: center; gap: 0.35rem; border-radius: 999px; padding: 0.25rem 0.65rem; background: rgba(109, 131, 255, 0.16); color: var(--accent-strong); font-size: 0.82rem; font-weight: 700; }
    .empty { padding: 1rem 0; color: var(--muted); }
    .route-preview { min-height: 4.5rem; padding: 0.85rem; border-radius: 14px; border: 1px dashed #4560ab; background: rgba(13, 21, 48, 0.65); }
    .route-preview strong { display: block; margin-bottom: 0.45rem; }
    .two-column { display: grid; gap: 1rem; grid-template-columns: 1.35fr 1fr; margin-top: 1rem; }
    @media (max-width: 900px) {
      .grid, .two-column, .form-grid { grid-template-columns: 1fr; }
      main { padding: 1rem; }
      table { min-width: 0; }
    }
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <h1>Crosschain Wallet Connect Dashboard</h1>
      <p class="muted">A simple control center for popular mobile and desktop wallets. Add a wallet, review the bridge route, and plan the next cross-chain transfer with Solana Firedancer-aware routing when available.</p>
      <ul class="hero-points">
        <li>Large tap targets and guided steps for phone or desktop users.</li>
        <li>Pre-filled wallet presets for Phantom, MetaMask, Trust Wallet, Keplr, Martian, and Suiet.</li>
        <li>Address, asset, and amount checks before a bridge plan is created.</li>
      </ul>
    </section>

    <div id="message" class="banner" role="status" aria-live="polite"></div>

    <div class="grid">
      <section class="card">
        <h2><span class="step">1</span>Add a wallet</h2>
        <p class="muted">Pick a wallet preset or enter the details manually. The dashboard checks the chain and address before saving.</p>
        <form id="wallet-form">
          <div class="form-grid">
            <div class="full">
              <label for="wallet-template">Wallet preset</label>
              <select id="wallet-template">
                <option value="">Choose a popular wallet</option>
              </select>
              <div class="helper" id="wallet-template-help">Choose a preset to auto-fill the wallet, chain, and connection style.</div>
            </div>
            <div class="field">
              <label for="wallet-name">Wallet name</label>
              <input id="wallet-name" required name="name" placeholder="Phantom" />
            </div>
            <div class="field">
              <label for="chain">Chain</label>
              <select id="chain" required name="chain"></select>
            </div>
            <div class="field">
              <label for="platform">Device</label>
              <select id="platform" required name="platform">
                <option value="mobile">Phone / tablet</option>
                <option value="pc">Computer</option>
              </select>
            </div>
            <div class="field">
              <label for="connector">Connection style</label>
              <select id="connector" required name="connector"></select>
            </div>
            <div class="full">
              <label for="wallet-address">Wallet address</label>
              <input id="wallet-address" required name="address" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="Wallet address" />
              <div class="helper" id="connector-help"></div>
            </div>
            <div class="full">
              <button type="submit">Add wallet to dashboard</button>
            </div>
          </div>
        </form>
      </section>

      <section class="card">
        <h2><span class="step">2</span>Plan a bridge</h2>
        <p class="muted">Choose a paired wallet and the dashboard will suggest supported assets and bridge routes.</p>
        <form id="bridge-form">
          <div class="form-grid">
            <div class="field">
              <label for="wallet">From wallet</label>
              <select id="wallet" required name="from_wallet_id"></select>
            </div>
            <div class="field">
              <label for="destination">Destination chain</label>
              <select id="destination" required name="destination_chain"></select>
            </div>
            <div class="field">
              <label for="asset">Asset</label>
              <select id="asset" required name="asset"></select>
            </div>
            <div class="field">
              <label for="amount">Amount</label>
              <input id="amount" required name="amount" type="number" min="0.00000001" step="any" placeholder="12.5" />
            </div>
            <div class="full">
              <label class="checkbox-row" for="prefer-firedancer">
                <input id="prefer-firedancer" type="checkbox" name="prefer_firedancer" />
                Prefer a Firedancer-optimized route when Solana is involved
              </label>
            </div>
            <div class="full">
              <div id="route-preview" class="route-preview">
                <strong>Route preview</strong>
                Pick a wallet and destination chain to see the best available routes.
              </div>
            </div>
            <div class="full">
              <button id="bridge-submit" type="submit">Create bridge plan</button>
            </div>
          </div>
        </form>
      </section>
    </div>

    <div class="two-column">
      <section class="card">
        <h2>Paired wallets</h2>
        <p class="muted">Keep only the wallets you want to manage from this page.</p>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Wallet</th>
                <th>Chain</th>
                <th>Device</th>
                <th>Connector</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody id="wallet-rows"></tbody>
          </table>
        </div>
      </section>

      <section class="card">
        <h2>Bridge plans</h2>
        <p class="muted">Every plan stays visible here so you can review the selected route.</p>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Asset</th>
                <th>From</th>
                <th>To</th>
                <th>Route</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody id="session-rows"></tbody>
          </table>
        </div>
      </section>
    </div>
  </main>

  <script>
    const state = {
      options: null,
      wallets: []
    };

    async function api(path, options) {
      const settings = options || {};
      const headers = settings.headers || {};
      if (!headers["Content-Type"] && settings.body) {
        headers["Content-Type"] = "application/json";
      }
      const response = await fetch(path, Object.assign({}, settings, { headers: headers }));
      const text = await response.text();
      if (!response.ok) {
        throw new Error(text || "Request failed");
      }
      if (!text) {
        return null;
      }
      return JSON.parse(text);
    }

    function setMessage(kind, text) {
      const banner = document.getElementById("message");
      banner.className = "banner " + kind;
      banner.textContent = text;
    }

    function appendCell(row, text) {
      const cell = document.createElement("td");
      cell.textContent = text;
      row.appendChild(cell);
      return cell;
    }

    function setSelectOptions(select, items, valueKey, labelKey, placeholder) {
      const selectedValue = select.value;
      select.innerHTML = "";
      if (placeholder) {
        const option = document.createElement("option");
        option.value = "";
        option.textContent = placeholder;
        select.appendChild(option);
      }
      items.forEach(function(item) {
        const option = document.createElement("option");
        option.value = typeof valueKey === "function" ? valueKey(item) : item[valueKey];
        option.textContent = typeof labelKey === "function" ? labelKey(item) : item[labelKey];
        select.appendChild(option);
      });
      if (selectedValue) {
        select.value = selectedValue;
      }
      if (!select.value && select.options.length > 0) {
        select.selectedIndex = 0;
      }
    }

    function currentWallet() {
      const walletID = document.getElementById("wallet").value;
      return state.wallets.find(function(wallet) {
        return wallet.id === walletID;
      }) || null;
    }

    function syncAddressExample() {
      const chain = document.getElementById("chain").value;
      const input = document.getElementById("wallet-address");
      const example = (state.options.address_examples || {})[chain];
      input.placeholder = example || "Wallet address";
    }

    function populateTemplates() {
      setSelectOptions(
        document.getElementById("wallet-template"),
        state.options.wallet_templates || [],
        "id",
        function(template) {
          return template.name + " — " + template.chain + " — " + (template.platform === "mobile" ? "phone" : "desktop");
        },
        "Choose a popular wallet"
      );
    }

    function populateChains() {
      const chains = (state.options.chains || []).map(function(chain) {
        return { value: chain, label: chain };
      });
      setSelectOptions(document.getElementById("chain"), chains, "value", "label");
    }

    function populateConnectors() {
      const platform = document.getElementById("platform").value;
      const connectors = (state.options.connectors || []).filter(function(connector) {
        return connector.platforms.indexOf(platform) >= 0;
      });
      setSelectOptions(document.getElementById("connector"), connectors, "id", "label");
      updateConnectorHelp();
    }

    function updateConnectorHelp() {
      const connectorID = document.getElementById("connector").value;
      const help = document.getElementById("connector-help");
      const connector = (state.options.connectors || []).find(function(item) {
        return item.id === connectorID;
      });
      help.textContent = connector ? connector.summary : "Choose how this wallet usually connects.";
    }

    function applyTemplate() {
      const templateID = document.getElementById("wallet-template").value;
      const help = document.getElementById("wallet-template-help");
      const template = (state.options.wallet_templates || []).find(function(item) {
        return item.id === templateID;
      });
      if (!template) {
        help.textContent = "Choose a preset to auto-fill the wallet, chain, and connection style.";
        return;
      }
      document.getElementById("wallet-name").value = template.name;
      document.getElementById("chain").value = template.chain;
      document.getElementById("platform").value = template.platform;
      populateConnectors();
      document.getElementById("connector").value = template.connector;
      updateConnectorHelp();
      syncAddressExample();
      help.textContent = template.description || "Preset applied.";
    }

    function syncBridgeDestinations() {
      const wallet = currentWallet();
      const destination = document.getElementById("destination");
      if (!wallet) {
        setSelectOptions(destination, [], "value", "label", "Add a wallet first");
        return;
      }
      const items = (state.options.chains || []).filter(function(chain) {
        return chain !== wallet.chain;
      }).map(function(chain) {
        return { value: chain, label: chain };
      });
      setSelectOptions(destination, items, "value", "label");
    }

    function syncAssets() {
      const assetSelect = document.getElementById("asset");
      const wallet = currentWallet();
      if (!wallet) {
        setSelectOptions(assetSelect, [], "value", "label", "Add a wallet first");
        return;
      }
      const items = ((state.options.assets_by_chain || {})[wallet.chain] || []).map(function(asset) {
        return { value: asset, label: asset };
      });
      setSelectOptions(assetSelect, items, "value", "label");
    }

    async function loadRoutePreview() {
      const preview = document.getElementById("route-preview");
      const wallet = currentWallet();
      const destination = document.getElementById("destination").value;
      if (!wallet || !destination) {
        preview.innerHTML = "<strong>Route preview</strong>Pick a wallet and destination chain to see the best available routes.";
        return;
      }

      try {
        const routes = await api("/api/routes?from=" + encodeURIComponent(wallet.chain) + "&to=" + encodeURIComponent(destination));
        preview.innerHTML = "";
        const title = document.createElement("strong");
        title.textContent = "Route preview";
        preview.appendChild(title);

        if (!routes.length) {
          const empty = document.createElement("div");
          empty.textContent = "No route is available for that chain pair yet.";
          preview.appendChild(empty);
          return;
        }

        const list = document.createElement("ul");
        list.className = "route-list";
        routes.forEach(function(route, index) {
          const item = document.createElement("li");
          let text = route.bridge + " from " + wallet.chain + " to " + destination;
          if (route.firedancer_optimized) {
            text += " (Firedancer optimized)";
          }
          if (route.performance_profile) {
            text += " — " + route.performance_profile;
          }
          if (index === 0) {
            text = "Best match: " + text;
          }
          item.textContent = text;
          list.appendChild(item);
        });
        preview.appendChild(list);
      } catch (error) {
        preview.innerHTML = "<strong>Route preview</strong>Could not load routes right now.";
      }
    }

    function resetWalletForm() {
      document.getElementById("wallet-form").reset();
      document.getElementById("wallet-template").value = "";
      document.getElementById("platform").value = "mobile";
      populateChains();
      populateConnectors();
      syncAddressExample();
      document.getElementById("wallet-template-help").textContent = "Choose a preset to auto-fill the wallet, chain, and connection style.";
    }

    function renderWallets(wallets) {
      const rows = document.getElementById("wallet-rows");
      rows.innerHTML = "";
      if (!wallets.length) {
        const row = document.createElement("tr");
        const cell = document.createElement("td");
        cell.colSpan = 5;
        cell.className = "empty";
        cell.textContent = "No wallets added yet.";
        row.appendChild(cell);
        rows.appendChild(row);
        return;
      }

      wallets.forEach(function(wallet) {
        const row = document.createElement("tr");
        appendCell(row, wallet.name + " (" + wallet.id + ")");
        appendCell(row, wallet.chain);
        appendCell(row, wallet.platform === "mobile" ? "Phone" : "Computer");
        appendCell(row, wallet.connector);
        const actionCell = document.createElement("td");
        const button = document.createElement("button");
        button.type = "button";
        button.className = "secondary";
        button.textContent = "Remove";
        button.addEventListener("click", async function() {
          try {
            await api("/api/wallets/" + encodeURIComponent(wallet.id), { method: "DELETE" });
            setMessage("success", wallet.name + " was removed from the dashboard.");
            await loadWallets();
            await loadSessions();
          } catch (error) {
            setMessage("error", error.message);
          }
        });
        actionCell.appendChild(button);
        row.appendChild(actionCell);
        rows.appendChild(row);
      });
    }

    function renderSessions(sessions) {
      const rows = document.getElementById("session-rows");
      rows.innerHTML = "";
      if (!sessions.length) {
        const row = document.createElement("tr");
        const cell = document.createElement("td");
        cell.colSpan = 6;
        cell.className = "empty";
        cell.textContent = "No bridge plans created yet.";
        row.appendChild(cell);
        rows.appendChild(row);
        return;
      }

      sessions.forEach(function(session) {
        const row = document.createElement("tr");
        appendCell(row, session.id);
        appendCell(row, session.asset + " " + session.amount);
        appendCell(row, session.from_wallet.name + " (" + session.from_wallet.chain + ")");
        appendCell(row, session.route.to_chain);
        appendCell(row, session.route.bridge + (session.route.firedancer_optimized ? " + firedancer" : ""));
        appendCell(row, session.status);
        rows.appendChild(row);
      });
    }

    async function loadWallets() {
      state.wallets = await api("/api/wallets");
      renderWallets(state.wallets);

      const walletSelect = document.getElementById("wallet");
      const items = state.wallets.map(function(wallet) {
        return {
          value: wallet.id,
          label: wallet.name + " — " + wallet.chain + " — " + (wallet.platform === "mobile" ? "phone" : "desktop")
        };
      });
      setSelectOptions(walletSelect, items, "value", "label", state.wallets.length ? null : "Add a wallet first");
      walletSelect.disabled = !state.wallets.length;
      document.getElementById("bridge-submit").disabled = !state.wallets.length;
      syncBridgeDestinations();
      syncAssets();
      await loadRoutePreview();
    }

    async function loadSessions() {
      const sessions = await api("/api/bridge/sessions");
      renderSessions(sessions);
    }

    async function refreshDashboardData() {
      await loadWallets();
      await loadSessions();
    }

    document.getElementById("wallet-template").addEventListener("change", applyTemplate);
    document.getElementById("platform").addEventListener("change", function() {
      populateConnectors();
    });
    document.getElementById("connector").addEventListener("change", updateConnectorHelp);
    document.getElementById("chain").addEventListener("change", syncAddressExample);
    document.getElementById("wallet").addEventListener("change", async function() {
      syncBridgeDestinations();
      syncAssets();
      await loadRoutePreview();
    });
    document.getElementById("destination").addEventListener("change", loadRoutePreview);
    document.getElementById("prefer-firedancer").addEventListener("change", loadRoutePreview);

    document.getElementById("wallet-form").addEventListener("submit", async function(event) {
      event.preventDefault();
      const form = new FormData(event.target);
      const payload = Object.fromEntries(form.entries());
      try {
        await api("/api/wallets", { method: "POST", body: JSON.stringify(payload) });
        setMessage("success", payload.name + " is ready in your dashboard.");
        resetWalletForm();
        await loadWallets();
      } catch (error) {
        setMessage("error", error.message);
      }
    });

    document.getElementById("bridge-form").addEventListener("submit", async function(event) {
      event.preventDefault();
      const form = new FormData(event.target);
      const payload = Object.fromEntries(form.entries());
      payload.prefer_firedancer = form.get("prefer_firedancer") === "on";
      try {
        await api("/api/bridge/sessions", { method: "POST", body: JSON.stringify(payload) });
        setMessage("success", "Bridge plan created.");
        await loadSessions();
        await loadRoutePreview();
      } catch (error) {
        setMessage("error", error.message);
      }
    });

    (async function() {
      try {
        state.options = await api("/api/options");
        populateTemplates();
        populateChains();
        populateConnectors();
        syncAddressExample();
        await refreshDashboardData();
        setMessage("info", state.wallets.length ? "Everything is ready. Pick a wallet and build a simple bridge plan." : "Start with step 1 and add a wallet.");
        window.setInterval(refreshDashboardData, 15000);
      } catch (error) {
        setMessage("error", error.message);
      }
    })();
  </script>
</body>
</html>`
