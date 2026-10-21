const VALID_PROTOCOLS = {
    "wireguard": [443, 80, 53, 123, 1194, 65142],
    "udp": [443, 80, 53, 123, 1194, 54783],
    "tcp": [443, 587, 21, 22, 80, 123, 143, 3306, 8080, 54783, 1194],
    "stealth": [443, 587, 21, 22, 80, 123, 143, 3306, 8080, 54783, 8443],
    "wstunnel": [443],
    "ikev2": [500]
};

const DEFAULT_OPERATORS = {
    "Iran": [
        "MCI",
        "Irancell",
        "Rightel",
        "Mokhaberat (TCI)",
        "Shatel",
        "Asiatech",
        "Mobinnet",
        "Zitel",
        "HiWeb",
        "Pars Online",
        "Pishgaman",
        "Fanava",
        "Other / Direct"
    ],
    "Russia": [
        "Rostelecom",
        "MTS",
        "Beeline",
        "MegaFon",
        "Tele2",
        "ER-Telecom (Dom.ru)",
        "NetByNet",
        "Other / Direct"
    ],
    "China": [
        "China Telecom",
        "China Unicom",
        "China Mobile",
        "Other / Direct"
    ],
    "Other / Global": [
        "Direct / LAN",
        "Starlink",
        "Cellular / 4G / 5G",
        "Other / Direct"
    ]
};

function getStoredCustomCountries() {
    try {
        const val = localStorage.getItem("windscribe_custom_countries");
        return val ? JSON.parse(val) : [];
    } catch {
        return [];
    }
}

function saveCustomCountry(countryName) {
    const list = getStoredCustomCountries();
    if (!list.includes(countryName)) {
        list.push(countryName);
        localStorage.setItem("windscribe_custom_countries", JSON.stringify(list));
    }
}

function getStoredCustomOperators(country) {
    try {
        const val = localStorage.getItem(`windscribe_custom_ops_${country}`);
        return val ? JSON.parse(val) : [];
    } catch {
        return [];
    }
}

function saveCustomOperator(country, opName) {
    const list = getStoredCustomOperators(country);
    if (!list.includes(opName)) {
        list.push(opName);
        localStorage.setItem(`windscribe_custom_ops_${country}`, JSON.stringify(list));
    }
}

let lastValidCountry = "Iran";
let lastValidIsp = "MCI";

function populateCountrySelect(selectedVal = "Iran") {
    const select = document.getElementById("networkCountry");
    if (!select) return;

    const defaultCountries = ["Iran", "Russia", "China", "Other / Global"];
    const customCountries = getStoredCustomCountries();
    const allCountries = [...defaultCountries];
    customCountries.forEach(c => {
        if (!allCountries.includes(c)) allCountries.push(c);
    });

    select.innerHTML = "";
    allCountries.forEach(country => {
        const opt = document.createElement("option");
        opt.value = country;
        opt.textContent = country;
        if (country === selectedVal) opt.selected = true;
        select.appendChild(opt);
    });

    const customOpt = document.createElement("option");
    customOpt.value = "__custom__";
    customOpt.textContent = "+ Add Custom Country...";
    select.appendChild(customOpt);

    if (allCountries.includes(selectedVal)) {
        select.value = selectedVal;
        lastValidCountry = selectedVal;
    } else {
        select.value = allCountries[0] || "Iran";
        lastValidCountry = select.value;
    }
}

function populateIspSelect(country, selectedVal = "") {
    const select = document.getElementById("networkIsp");
    if (!select) return;

    const defaults = DEFAULT_OPERATORS[country] || ["Other / Direct"];
    const customs = getStoredCustomOperators(country);
    const allOps = [...defaults];
    customs.forEach(op => {
        if (!allOps.includes(op)) allOps.push(op);
    });

    select.innerHTML = "";
    allOps.forEach(op => {
        const opt = document.createElement("option");
        opt.value = op;
        opt.textContent = op;
        if (op === selectedVal) opt.selected = true;
        select.appendChild(opt);
    });

    const customOpt = document.createElement("option");
    customOpt.value = "__custom__";
    customOpt.textContent = "+ Add Custom Operator...";
    select.appendChild(customOpt);

    if (selectedVal && allOps.includes(selectedVal)) {
        select.value = selectedVal;
        lastValidIsp = selectedVal;
    } else {
        select.value = allOps[0] || "Other / Direct";
        lastValidIsp = select.value;
    }
}

function initCountryAndIspSelectors() {
    const countrySelect = document.getElementById("networkCountry");
    const ispSelect = document.getElementById("networkIsp");

    const customCountryBox = document.getElementById("customCountryBox");
    const customCountryInput = document.getElementById("customCountryInput");
    const btnAddCountry = document.getElementById("btnAddCountry");
    const btnSaveCountry = document.getElementById("btnSaveCountry");
    const btnCancelCountry = document.getElementById("btnCancelCountry");

    const customIspBox = document.getElementById("customIspBox");
    const customIspInput = document.getElementById("customIspInput");
    const btnAddIsp = document.getElementById("btnAddIsp");
    const btnSaveIsp = document.getElementById("btnSaveIsp");
    const btnCancelIsp = document.getElementById("btnCancelIsp");

    // Load initial state
    populateCountrySelect("Iran");
    populateIspSelect("Iran", "MCI");

    if (countrySelect) {
        countrySelect.addEventListener("change", () => {
            if (countrySelect.value === "__custom__") {
                if (customCountryBox) {
                    customCountryBox.style.display = "flex";
                    if (customCountryInput) customCountryInput.focus();
                }
            } else {
                lastValidCountry = countrySelect.value;
                if (customCountryBox) customCountryBox.style.display = "none";
                populateIspSelect(countrySelect.value);
            }
        });
    }

    if (btnAddCountry && customCountryBox) {
        btnAddCountry.addEventListener("click", () => {
            const isHidden = customCountryBox.style.display === "none";
            customCountryBox.style.display = isHidden ? "flex" : "none";
            if (isHidden && customCountryInput) customCountryInput.focus();
        });
    }

    const handleSaveCountry = () => {
        if (!customCountryInput) return;
        const name = customCountryInput.value.trim();
        if (name) {
            saveCustomCountry(name);
            populateCountrySelect(name);
            populateIspSelect(name);
            customCountryInput.value = "";
            if (customCountryBox) customCountryBox.style.display = "none";
            logMsg(`Added custom country: ${name}`, "log-success");
        }
    };

    if (btnSaveCountry) btnSaveCountry.addEventListener("click", handleSaveCountry);
    if (customCountryInput) {
        customCountryInput.addEventListener("keydown", (e) => {
            if (e.key === "Enter") {
                e.preventDefault();
                handleSaveCountry();
            } else if (e.key === "Escape") {
                if (customCountryBox) customCountryBox.style.display = "none";
                if (countrySelect) countrySelect.value = lastValidCountry;
            }
        });
    }

    if (btnCancelCountry && customCountryBox) {
        btnCancelCountry.addEventListener("click", () => {
            customCountryBox.style.display = "none";
            if (countrySelect) countrySelect.value = lastValidCountry;
        });
    }

    // ISP Handlers
    if (ispSelect) {
        ispSelect.addEventListener("change", () => {
            if (ispSelect.value === "__custom__") {
                if (customIspBox) {
                    customIspBox.style.display = "flex";
                    if (customIspInput) customIspInput.focus();
                }
            } else {
                lastValidIsp = ispSelect.value;
                if (customIspBox) customIspBox.style.display = "none";
            }
        });
    }

    if (btnAddIsp && customIspBox) {
        btnAddIsp.addEventListener("click", () => {
            const isHidden = customIspBox.style.display === "none";
            customIspBox.style.display = isHidden ? "flex" : "none";
            if (isHidden && customIspInput) customIspInput.focus();
        });
    }

    const handleSaveIsp = () => {
        if (!customIspInput || !countrySelect) return;
        const name = customIspInput.value.trim();
        if (name) {
            const country = countrySelect.value !== "__custom__" ? countrySelect.value : lastValidCountry;
            saveCustomOperator(country, name);
            populateIspSelect(country, name);
            customIspInput.value = "";
            if (customIspBox) customIspBox.style.display = "none";
            logMsg(`Added custom operator for ${country}: ${name}`, "log-success");
        }
    };

    if (btnSaveIsp) btnSaveIsp.addEventListener("click", handleSaveIsp);
    if (customIspInput) {
        customIspInput.addEventListener("keydown", (e) => {
            if (e.key === "Enter") {
                e.preventDefault();
                handleSaveIsp();
            } else if (e.key === "Escape") {
                if (customIspBox) customIspBox.style.display = "none";
                if (ispSelect) ispSelect.value = lastValidIsp;
            }
        });
    }

    if (btnCancelIsp && customIspBox) {
        btnCancelIsp.addEventListener("click", () => {
            customIspBox.style.display = "none";
            if (ispSelect) ispSelect.value = lastValidIsp;
        });
    }
}

let lastExcelPath = "";

function logMsg(msg, type = "log-msg") {
    const logConsole = document.getElementById("logConsole");
    if (!logConsole) return;

    const div = document.createElement("div");
    div.className = type;
    const isScrolledToBottom = logConsole.scrollHeight - logConsole.clientHeight <= logConsole.scrollTop + 50;
    
    const time = new Date().toLocaleTimeString();
    div.innerText = `[${time}] ${msg}`;
    logConsole.appendChild(div);
    
    // Smooth auto-scroll only if already at the bottom
    if (isScrolledToBottom) {
        setTimeout(() => {
            logConsole.scrollTo({ top: logConsole.scrollHeight, behavior: 'smooth' });
        }, 10);
    }
}

function updateSniFieldState() {
    const customSni = document.getElementById("customSni");
    if (!customSni) return;

    const stealthProto = document.querySelector(`.cb-proto[value="stealth"]`);
    const wstunnelProto = document.querySelector(`.cb-proto[value="wstunnel"]`);

    const hasStealthPort = document.querySelectorAll('.cb-port-stealth:checked').length > 0;
    const hasWstunnelPort = document.querySelectorAll('.cb-port-wstunnel:checked').length > 0;

    const isSupported = (stealthProto && stealthProto.checked) ||
                        (wstunnelProto && wstunnelProto.checked) ||
                        hasStealthPort ||
                        hasWstunnelPort;

    customSni.disabled = !isSupported;
}

function updateTweaksState() {
    const tweaksCb = document.getElementById("protocolTweaks");
    const amneziaSelect = document.getElementById("amneziaConfig");
    const serverRoutingSelect = document.getElementById("serverRouting");
    const btnRefresh = document.getElementById("btnRefreshAmnezia");
    
    const enabled = tweaksCb ? tweaksCb.checked : true;
    if (amneziaSelect) amneziaSelect.disabled = !enabled;
    if (serverRoutingSelect) serverRoutingSelect.disabled = !enabled;
    if (btnRefresh) btnRefresh.disabled = !enabled;
}

let toastTimeout = null;

function showExcelToast(filePath) {
    const toast = document.getElementById("excelToast");
    const desc = document.getElementById("excelToastMsg");
    if (!toast) return;

    if (toastTimeout) {
        clearTimeout(toastTimeout);
        toastTimeout = null;
    }

    const fileName = filePath ? filePath.split(/[\/\\]/).pop() : "Report saved";
    if (desc) desc.innerText = fileName;

    toast.classList.remove("toast-hiding");
    toast.style.display = "flex";

    // Auto-dismiss after 6 seconds
    toastTimeout = setTimeout(() => {
        hideExcelToast();
    }, 6000);
}

function hideExcelToast() {
    const toast = document.getElementById("excelToast");
    if (!toast) return;
    toast.classList.add("toast-hiding");
    setTimeout(() => {
        toast.style.display = "none";
        toast.classList.remove("toast-hiding");
    }, 250);
}

function updateProtocolsSummary() {
    const activeProtos = Object.keys(VALID_PROTOCOLS).filter(proto => {
        return document.querySelectorAll(`.cb-port-${proto}:checked`).length > 0;
    });
    const allPorts = document.querySelectorAll('[class*="cb-port-"]');
    const checkedPorts = document.querySelectorAll('[class*="cb-port-"]:checked');
    
    const masterCb = document.getElementById("cbSelectAllProtocols");
    const badge = document.getElementById("protocolsCountBadge");

    if (badge) {
        badge.innerText = `${activeProtos.length} Protos, ${checkedPorts.length} Ports`;
    }

    if (masterCb) {
        if (checkedPorts.length === allPorts.length && allPorts.length > 0) {
            masterCb.checked = true;
            masterCb.indeterminate = false;
        } else if (checkedPorts.length > 0) {
            masterCb.checked = false;
            masterCb.indeterminate = true;
        } else {
            masterCb.checked = false;
            masterCb.indeterminate = false;
        }
    }
}

function initProtocols() {
    const container = document.getElementById("protocolPortsContainer");
    if (!container) return;
    container.innerHTML = "";
    
    Object.keys(VALID_PROTOCOLS).forEach(proto => {
        const box = document.createElement("div");
        box.className = "protocol-box";
        
        const header = document.createElement("label");
        header.className = "protocol-header";
        const cbProto = document.createElement("input");
        cbProto.type = "checkbox";
        cbProto.value = proto;
        cbProto.className = "cb-proto";
        const isProtoActive = (proto === "stealth" || proto === "wstunnel");
        cbProto.checked = isProtoActive;
        
        header.appendChild(cbProto);
        header.appendChild(document.createTextNode(" " + proto.toUpperCase()));
        
        const grid = document.createElement("div");
        grid.className = "ports-grid";
        
        VALID_PROTOCOLS[proto].forEach((port, idx) => {
            const lbl = document.createElement("label");
            const cbPort = document.createElement("input");
            cbPort.type = "checkbox";
            cbPort.value = port;
            cbPort.className = `cb-port-${proto}`;
            if (isProtoActive && port === 443) {
                cbPort.checked = true;
            } else {
                cbPort.checked = false;
            }
            
            lbl.appendChild(cbPort);
            lbl.appendChild(document.createTextNode(" " + port));
            grid.appendChild(lbl);

            cbPort.addEventListener("change", () => {
                const totalPorts = grid.querySelectorAll(`.cb-port-${proto}`).length;
                const activePorts = grid.querySelectorAll(`.cb-port-${proto}:checked`).length;
                cbProto.checked = (activePorts === totalPorts);
                cbProto.indeterminate = (activePorts > 0 && activePorts < totalPorts);
                updateSniFieldState();
                updateProtocolsSummary();
            });
        });

        cbProto.addEventListener("change", (e) => {
            const isChecked = e.target.checked;
            grid.querySelectorAll(`.cb-port-${proto}`).forEach(p => p.checked = isChecked);
            cbProto.indeterminate = false;
            updateSniFieldState();
            updateProtocolsSummary();
        });
        
        box.appendChild(header);
        box.appendChild(grid);
        container.appendChild(box);
    });

    const masterCb = document.getElementById("cbSelectAllProtocols");
    if (masterCb) {
        masterCb.addEventListener("change", (e) => {
            const isChecked = e.target.checked;
            document.querySelectorAll(".cb-proto").forEach(p => {
                p.checked = isChecked;
                p.indeterminate = false;
            });
            document.querySelectorAll('[class*="cb-port-"]').forEach(p => p.checked = isChecked);
            updateSniFieldState();
            updateProtocolsSummary();
        });
    }

    updateSniFieldState();
    updateProtocolsSummary();
}

async function loadAmneziaConfigs(forceOnline = false) {
    try {
        const select = document.getElementById("amneziaConfig");
        if (!select) return;

        let configs = [];
        if (forceOnline) {
            logMsg("Refreshing Amnezia configs online...", "log-info");
            configs = await window.go.main.App.RefreshAmneziaConfigs();
            logMsg(`Loaded ${configs.length} Amnezia presets.`, "log-success");
        } else {
            configs = await window.go.main.App.GetAmneziaConfigs();
        }

        if (configs && configs.length > 0) {
            const currentVal = select.value;
            select.innerHTML = "";
            configs.forEach(cfg => {
                const opt = document.createElement("option");
                opt.value = cfg.name;
                opt.innerText = cfg.name;
                select.appendChild(opt);
            });
            if (currentVal) select.value = currentVal;
        }
    } catch (e) {
        logMsg("Failed to load Amnezia configs: " + e, "log-error");
    }
}

async function fetchLocations(forceOnline = false) {
    try {
        const treeContainer = document.getElementById("locationTree");
        if (!treeContainer) return;

        treeContainer.innerHTML = "<div style='padding:10px; color:#888;'>Loading locations...</div>";
        
        let locs = [];
        if (forceOnline) {
            logMsg("Fetching server locations from live Windscribe API...", "log-info");
            locs = await window.go.main.App.RefreshLocations();
            logMsg(`Updated ${locs.length} locations from live network!`, "log-success");
        } else {
            locs = await window.go.main.App.GetAllLocations();
        }

        treeContainer.innerHTML = "";
        
        if (!locs || locs.length === 0) {
            treeContainer.innerHTML = "<div style='padding:10px; color:#da3633;'>Failed to load locations.</div>";
            return;
        }

        // Group by Continent -> Country (Region) -> Location
        const tree = {};
        locs.forEach(loc => {
            const cont = loc.continent || "Other";
            const reg = loc.region || "Other";
            if (!tree[cont]) tree[cont] = {};
            if (!tree[cont][reg]) tree[cont][reg] = [];
            tree[cont][reg].push(loc);
        });

        for (const [continent, countries] of Object.entries(tree)) {
            const contDiv = document.createElement('div');
            contDiv.className = 'tree-continent';
            
            const contHeader = document.createElement('div');
            contHeader.className = 'tree-continent-header';
            
            const contIcon = document.createElement('span');
            contIcon.className = 'tree-icon';
            contIcon.innerText = '▶';
            
            const contCb = document.createElement('input');
            contCb.type = 'checkbox';
            contCb.className = 'cb-continent';
            contCb.value = continent;
            
            contHeader.appendChild(contIcon);
            contHeader.appendChild(contCb);
            contHeader.appendChild(document.createTextNode(` ${continent}`));
            
            const countriesDiv = document.createElement('div');
            countriesDiv.className = 'tree-countries';
            
            contHeader.onclick = (e) => {
                if (e.target === contCb) return; 
                const isOpen = countriesDiv.style.display === 'block';
                countriesDiv.style.display = isOpen ? 'none' : 'block';
                contIcon.className = isOpen ? 'tree-icon' : 'tree-icon open';
            };
            
            // Add countries
            for (const [country, locations] of Object.entries(countries)) {
                const countryDiv = document.createElement('div');
                countryDiv.className = 'tree-country';
                
                const countryHeader = document.createElement('div');
                countryHeader.className = 'tree-country-header';
                
                const cIcon = document.createElement('span');
                cIcon.className = 'tree-icon';
                cIcon.innerText = '▶';
                
                const cCb = document.createElement('input');
                cCb.type = 'checkbox';
                cCb.className = 'cb-country';
                cCb.dataset.continent = continent;
                cCb.value = country;
                
                countryHeader.appendChild(cIcon);
                countryHeader.appendChild(cCb);
                countryHeader.appendChild(document.createTextNode(` ${country}`));
                
                const locsDiv = document.createElement('div');
                locsDiv.className = 'tree-locations';
                
                countryHeader.onclick = (e) => {
                    if (e.target === cCb) return; 
                    const isOpen = locsDiv.style.display === 'block';
                    locsDiv.style.display = isOpen ? 'none' : 'block';
                    cIcon.className = isOpen ? 'tree-icon' : 'tree-icon open';
                };
                
                // Add locations
                locations.forEach(loc => {
                    const locDiv = document.createElement('label');
                    locDiv.className = 'tree-location';
                    locDiv.dataset.search = `${continent} ${country} ${loc.city} ${loc.location}`.toLowerCase();
                    
                    const lCb = document.createElement('input');
                    lCb.type = 'checkbox';
                    lCb.className = 'cb-location';
                    lCb.dataset.continent = continent;
                    lCb.dataset.country = country;
                    lCb.dataset.city = loc.city;
                    lCb.value = loc.location;
                    
                    locDiv.appendChild(lCb);
                    locDiv.appendChild(document.createTextNode(` ${loc.city} (${loc.location})`));
                    
                    locsDiv.appendChild(locDiv);
                    
                    // Cascade from location up
                    lCb.addEventListener('change', () => {
                        const allLocs = locsDiv.querySelectorAll('.cb-location');
                        const checkedLocs = locsDiv.querySelectorAll('.cb-location:checked');
                        cCb.checked = (allLocs.length === checkedLocs.length);
                        cCb.indeterminate = (checkedLocs.length > 0 && checkedLocs.length < allLocs.length);
                        
                        const allCountries = countriesDiv.querySelectorAll('.cb-country');
                        const checkedCountries = countriesDiv.querySelectorAll('.cb-country:checked');
                        contCb.checked = (allCountries.length === checkedCountries.length);
                        contCb.indeterminate = (checkedCountries.length > 0 && checkedCountries.length < allCountries.length);
                    });
                });
                
                // Cascade from country down
                cCb.addEventListener('change', (e) => {
                    locsDiv.querySelectorAll('.cb-location').forEach(cb => cb.checked = e.target.checked);
                    
                    const allCountries = countriesDiv.querySelectorAll('.cb-country');
                    const checkedCountries = countriesDiv.querySelectorAll('.cb-country:checked');
                    contCb.checked = (allCountries.length === checkedCountries.length);
                    contCb.indeterminate = (checkedCountries.length > 0 && checkedCountries.length < allCountries.length);
                });
                
                countryDiv.appendChild(countryHeader);
                countryDiv.appendChild(locsDiv);
                countriesDiv.appendChild(countryDiv);
            }
            
            // Cascade from continent down
            contCb.addEventListener('change', (e) => {
                countriesDiv.querySelectorAll('.cb-country, .cb-location').forEach(cb => {
                    cb.checked = e.target.checked;
                    cb.indeterminate = false;
                });
            });
            
            contDiv.appendChild(contHeader);
            contDiv.appendChild(countriesDiv);
            treeContainer.appendChild(contDiv);
        }
    } catch (e) {
        logMsg("Failed to fetch locations: " + e, "log-error");
    }
}

document.addEventListener("DOMContentLoaded", async () => {
    initCountryAndIspSelectors();

    // Query OS from backend to customize supported protocols (e.g. Linux does not support IKEv2)
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetOS) {
            const currentOS = await window.go.main.App.GetOS();
            if (currentOS === "linux") {
                delete VALID_PROTOCOLS["ikev2"];
            }
        }
    } catch (e) {
        console.warn("Could not query OS from backend:", e);
    }

    initProtocols();
    updateTweaksState();

    const tweaksCb = document.getElementById("protocolTweaks");
    if (tweaksCb) {
        tweaksCb.addEventListener("change", updateTweaksState);
    }
    
    setTimeout(() => {
        loadAmneziaConfigs();
        fetchLocations();
    }, 400);

    // Refresh Buttons with rotation animation
    const btnRefreshLocs = document.getElementById("btnRefreshLocs");
    if (btnRefreshLocs) {
        btnRefreshLocs.addEventListener("click", async () => {
            btnRefreshLocs.classList.add("spinning");
            try {
                await fetchLocations(true);
            } finally {
                setTimeout(() => btnRefreshLocs.classList.remove("spinning"), 600);
            }
        });
    }

    const btnRefreshAmnezia = document.getElementById("btnRefreshAmnezia");
    if (btnRefreshAmnezia) {
        btnRefreshAmnezia.addEventListener("click", async () => {
            btnRefreshAmnezia.classList.add("spinning");
            try {
                await loadAmneziaConfigs(true);
            } finally {
                setTimeout(() => btnRefreshAmnezia.classList.remove("spinning"), 600);
            }
        });
    }

    // Search Box Filter
    const locSearch = document.getElementById("locSearch");
    if (locSearch) {
        locSearch.addEventListener("input", (e) => {
            const queryWords = e.target.value.toLowerCase().trim().split(/\s+/);
            const isSearching = e.target.value.trim().length > 0;
            
            document.querySelectorAll(".tree-continent").forEach(contDiv => {
                let contHasMatch = false;
                
                contDiv.querySelectorAll(".tree-country").forEach(countryDiv => {
                    let countryHasMatch = false;
                    
                    countryDiv.querySelectorAll(".tree-location").forEach(loc => {
                        if (!isSearching) {
                            loc.style.display = "flex";
                            countryHasMatch = true;
                            return;
                        }
                        
                        const text = loc.dataset.search || loc.innerText.toLowerCase();
                        const isMatch = queryWords.every(word => text.includes(word));
                        if (isMatch) {
                            loc.style.display = "flex";
                            countryHasMatch = true;
                            contHasMatch = true;
                        } else {
                            loc.style.display = "none";
                        }
                    });
                    
                    if (isSearching) {
                        countryDiv.style.display = countryHasMatch ? "block" : "none";
                        const locsDiv = countryDiv.querySelector(".tree-locations");
                        if (locsDiv) locsDiv.style.display = countryHasMatch ? "block" : "none";
                        
                        const cIcon = countryDiv.querySelector(".tree-icon");
                        if (cIcon) cIcon.className = countryHasMatch ? "tree-icon open" : "tree-icon";
                    } else {
                        countryDiv.style.display = "block";
                        const locsDiv = countryDiv.querySelector(".tree-locations");
                        if (locsDiv) locsDiv.style.display = "none";
                        
                        const cIcon = countryDiv.querySelector(".tree-icon");
                        if (cIcon) cIcon.className = "tree-icon";
                    }
                });
                
                if (isSearching) {
                    contDiv.style.display = contHasMatch ? "block" : "none";
                    const countriesDiv = contDiv.querySelector(".tree-countries");
                    if (countriesDiv) countriesDiv.style.display = contHasMatch ? "block" : "none";
                    
                    const contIcon = contDiv.querySelector(".tree-icon");
                    if (contIcon) contIcon.className = contHasMatch ? "tree-icon open" : "tree-icon";
                } else {
                    contDiv.style.display = "block";
                    const countriesDiv = contDiv.querySelector(".tree-countries");
                    if (countriesDiv) countriesDiv.style.display = "none";
                    
                    const contIcon = contDiv.querySelector(".tree-icon");
                    if (contIcon) contIcon.className = "tree-icon";
                }
            });
        });
    }

    const btnStart = document.getElementById("btnStart");
    const btnStop = document.getElementById("btnStop");
    const btnClearLogs = document.getElementById("btnClearLogs");
    const btnExportExcel = document.getElementById("btnExportExcel");
    const btnOpenExcel = document.getElementById("btnOpenExcel");
    const excelToast = document.getElementById("excelToast");
    
    if (btnClearLogs) {
        btnClearLogs.addEventListener("click", () => {
            const logConsole = document.getElementById("logConsole");
            if (logConsole) logConsole.innerHTML = "";
        });
    }

    document.getElementById("btnSelectAllLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = true);
        document.querySelectorAll(".cb-country").forEach(cb => cb.checked = true);
        document.querySelectorAll(".cb-continent").forEach(cb => cb.checked = true);
    });
    
    document.getElementById("btnClearLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = false);
        document.querySelectorAll(".cb-country").forEach(cb => { cb.checked = false; cb.indeterminate = false; });
        document.querySelectorAll(".cb-continent").forEach(cb => { cb.checked = false; cb.indeterminate = false; });
    });

    const selMode = document.getElementById("selectionMode");
    const locsWrapper = document.getElementById("locationsWrapper");
    
    selMode.addEventListener("change", (e) => {
        if (e.target.value === "manual" || e.target.value === "discovery") {
            locsWrapper.style.display = "block";
        } else {
            locsWrapper.style.display = "none";
        }
    });

    function collectTestConfig(isResume = false) {
        const selectionMode = document.getElementById("selectionMode").value;
        const selectedLocations = [];
        
        if (selectionMode === "manual" || selectionMode === "discovery") {
            document.querySelectorAll(".cb-location:checked").forEach(cb => {
                selectedLocations.push({
                    location: cb.value,
                    city: cb.dataset.city || "",
                    region: cb.dataset.country || "",
                    continent: cb.dataset.continent || "",
                    display: `${cb.dataset.country} - ${cb.dataset.city} (${cb.value})`
                });
            });
            if (!isResume && selectedLocations.length === 0) {
                alert("Please select at least one location!");
                return null;
            }
        }

        const protocolPorts = {};
        Object.keys(VALID_PROTOCOLS).forEach(proto => {
            const protoCb = document.querySelector(`.cb-proto[value="${proto}"]`);
            if (protoCb && !protoCb.checked && !protoCb.indeterminate) {
                return; // Protocol is explicitly disabled
            }
            const ports = [];
            document.querySelectorAll(`.cb-port-${proto}:checked`).forEach(pCb => ports.push(parseInt(pCb.value)));
            if (ports.length > 0) protocolPorts[proto] = ports;
        });

        const countryElem = document.getElementById("networkCountry");
        const ispElem = document.getElementById("networkIsp");
        const countryVal = (countryElem && countryElem.value !== "__custom__") ? countryElem.value : lastValidCountry;
        const ispName = (ispElem && ispElem.value !== "__custom__") ? ispElem.value : lastValidIsp;
        const ispVal = countryVal ? `${countryVal} - ${ispName}` : ispName;

        return {
            selectedLocations: selectedLocations,
            protocolPorts: protocolPorts,
            protocolTweaks: document.getElementById("protocolTweaks").checked,
            amneziaConfig: document.getElementById("amneziaConfig").value,
            serverRouting: document.getElementById("serverRouting").value,
            customSni: document.getElementById("customSni").value.trim(),
            largeTls: document.getElementById("largeTls").checked,
            internalDns: document.getElementById("internalDns").value,
            isp: ispVal,
            isResume: isResume,
            selectionMode: selectionMode,
            testPing: true,
            testDownload: true,
            testUpload: true,
            timeout: parseInt(document.getElementById("connTimeout").value) || 25,
            speedWait: parseInt(document.getElementById("speedWait").value) || 3
        };
    }

    const btnContinue = document.getElementById("btnContinue");

    // Handle Live Events from Go Orchestrator
    window.runtime.EventsOn("test_update", (update) => {
        if (update.message) logMsg(update.message, "log-info");
        
        // Reset Cards between tests
        if (update.resetCards) {
            document.getElementById("valPing").innerText = "--";
            document.getElementById("valDown").innerText = "--";
            document.getElementById("valUp").innerText = "--";
        }

        if (update.ping) document.getElementById("valPing").innerText = update.ping;
        if (update.down) document.getElementById("valDown").innerText = update.down;
        if (update.up) document.getElementById("valUp").innerText = update.up;
        if (update.status) document.getElementById("valStatus").innerText = update.status;
        
        // Show Excel Ready toast if generated
        if (update.excelPath) {
            lastExcelPath = update.excelPath;
            showExcelToast(update.excelPath);
            if (btnExportExcel) {
                btnExportExcel.disabled = false;
                btnExportExcel.title = `Click to open ${update.excelPath.split(/[\/\\]/).pop()}`;
            }
        }

        // Handle CanResume state
        if (update.canResume) {
            if (btnContinue) {
                const remaining = update.totalCount - update.testedCount;
                const remTxt = document.getElementById("remainingCountTxt");
                if (remTxt) remTxt.innerText = remaining;
                btnContinue.style.display = "flex";
                btnContinue.disabled = false;
            }
            btnStart.innerText = "START NEW TEST";
            btnStart.disabled = false;
            btnStop.disabled = true;
            btnStop.innerText = "STOP SESSION";
        } else if (update.status === "Done") {
            if (btnContinue) btnContinue.style.display = "none";
            btnStart.innerText = "START ANALYZING";
            btnStart.disabled = false;
            btnStop.disabled = true;
            btnStop.innerText = "STOP SESSION";
        } else if (update.status === "Stopped") {
            btnStop.disabled = true;
            btnStop.innerText = "STOP SESSION";
            btnStart.disabled = false;
            if (btnContinue && !update.canResume) {
                btnContinue.style.display = "none";
            }
        }
    });

    // Start Button Handler (Starts fresh)
    btnStart.addEventListener("click", () => {
        const config = collectTestConfig(false);
        if (!config) return;

        btnStart.disabled = true;
        btnStart.innerText = "TESTING IN PROGRESS...";
        btnStop.disabled = false;
        btnStop.innerText = "STOP SESSION";
        if (btnContinue) btnContinue.style.display = "none";
        if (btnExportExcel) btnExportExcel.disabled = true;
        document.getElementById("valPing").innerText = "--";
        document.getElementById("valDown").innerText = "--";
        document.getElementById("valUp").innerText = "--";
        document.getElementById("valStatus").innerText = "Starting...";
        hideExcelToast();
        
        logMsg(`Starting Windscribe Analyzer [ISP: ${config.isp}]...`, "log-success");
        window.go.main.App.StartTest(config);
    });

    // Continue Button Handler (Resumes remaining queue)
    if (btnContinue) {
        btnContinue.addEventListener("click", () => {
            const config = collectTestConfig(true);
            if (!config) return;

            btnStart.disabled = true;
            btnContinue.style.display = "none";
            btnStop.disabled = false;
            btnStop.innerText = "STOP SESSION";
            document.getElementById("valStatus").innerText = "Resuming...";
            hideExcelToast();

            logMsg("Resuming Windscribe Analyzer with remaining locations...", "log-info");
            window.go.main.App.StartTest(config);
        });
    }

    // Stop Button Handler
    btnStop.addEventListener("click", () => {
        logMsg("Stopping test session and disconnecting Windscribe...", "log-error");
        btnStop.disabled = true;
        btnStop.innerText = "STOPPING...";
        window.go.main.App.StopTest();
    });

    // Open Excel Button in Header
    if (btnExportExcel) {
        btnExportExcel.addEventListener("click", () => {
            if (lastExcelPath) {
                logMsg(`Opening Excel report: ${lastExcelPath.split(/[\/\\]/).pop()}`, "log-info");
                window.go.main.App.OpenExcelFile(lastExcelPath);
            }
        });
    }

    // Open Excel Button inside toast
    if (btnOpenExcel) {
        btnOpenExcel.addEventListener("click", () => {
            if (lastExcelPath) {
                logMsg(`Opening Excel report: ${lastExcelPath.split(/[\/\\]/).pop()}`, "log-info");
                window.go.main.App.OpenExcelFile(lastExcelPath);
            }
            hideExcelToast();
        });
    }

    // Close button inside toast
    const btnCloseToast = document.getElementById("btnCloseToast");
    if (btnCloseToast) {
        btnCloseToast.addEventListener("click", () => {
            hideExcelToast();
        });
    }

    // Number Stepper Handlers
    document.querySelectorAll(".step-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            const targetId = btn.dataset.target;
            const input = document.getElementById(targetId);
            if (!input) return;
            let val = parseInt(input.value) || 0;
            const step = btn.dataset.step === "up" ? 1 : -1;
            const min = input.hasAttribute("min") ? parseInt(input.getAttribute("min")) : 1;
            const max = input.hasAttribute("max") ? parseInt(input.getAttribute("max")) : 999;
            val = Math.max(min, Math.min(max, val + step));
            input.value = val;
        });
    });
});
