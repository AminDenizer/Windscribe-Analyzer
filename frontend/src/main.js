import { TRANSLATIONS, countryCodeToFlag } from "./i18n.js";

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
    const tweaksMode = document.getElementById("protocolTweaksMode");
    const amneziaGroup = document.getElementById("amneziaGroup");
    const amneziaSelect = document.getElementById("amneziaConfig");
    const btnRefresh = document.getElementById("btnRefreshAmnezia");
    
    const val = (tweaksMode ? tweaksMode.value : "Enabled").toLowerCase();
    const isEnabled = (val === "enabled");

    if (amneziaGroup) {
        amneziaGroup.classList.toggle("is-collapsed", !isEnabled);
    }
    if (amneziaSelect) amneziaSelect.disabled = !isEnabled;
    if (btnRefresh) btnRefresh.disabled = !isEnabled;
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

let connectedCount = 0;

function escapeHtml(str) {
    if (!str) return "";
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

let verifiedNodesList = [];
let countdownTimer = null;
let remainingEtaSec = 0;
let isTestRunning = false;

function getTimingParams() {
    const connTimeout = parseInt(document.getElementById("connTimeout")?.value) || 25;
    const speedWait = parseInt(document.getElementById("speedWait")?.value) || 5;
    const stabEnabled = document.getElementById("stabilizeEnabled")?.checked !== false;
    const stabTimeout = stabEnabled ? (parseInt(document.getElementById("stabilizeTimeout")?.value) || 30) : 0;
    const engine = document.getElementById("speedEngine")?.value || "ookla";
    const speedSec = engine === "both" ? 22 : 12;

    const avgPerTest = Math.max(5, Math.round((connTimeout * 0.6) + (stabTimeout * 0.45) + speedWait + (speedSec * 0.6)));
    const maxPerTest = connTimeout + stabTimeout + speedWait + speedSec;
    return { avgPerTest, maxPerTest };
}

function getPerTestSeconds() {
    return getTimingParams().avgPerTest;
}

function formatCountdown(sec) {
    const s = Math.max(0, Math.round(sec));
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const remS = s % 60;
    if (h > 0) return `${h}h ${String(m).padStart(2, "0")}m ${String(remS).padStart(2, "0")}s`;
    return `${String(m).padStart(2, "0")}:${String(remS).padStart(2, "0")}`;
}

function renderCountdownEta(totalTestsOverride = null) {
    const pillEta = document.getElementById("floatingMatrixEta");
    if (!pillEta) return;
    const lang = localStorage.getItem("ws_lang") || "en";
    const t = TRANSLATIONS[lang] || TRANSLATIONS.en;
    if (isTestRunning) {
        pillEta.textContent = `${t.matrixEst}: ${formatCountdown(remainingEtaSec)}`;
    } else {
        const locsCount = document.querySelectorAll(".cb-location:checked").length;
        const portsCount = document.querySelectorAll('[class*="cb-port-"]:checked').length;
        const totalTests = totalTestsOverride !== null ? totalTestsOverride : (locsCount * portsCount);
        if (totalTests === 0) {
            pillEta.textContent = `${t.matrixEst}: ~0m`;
            return;
        }
        const { avgPerTest, maxPerTest } = getTimingParams();
        const avgMin = Math.max(1, Math.round((totalTests * avgPerTest) / 60));
        const maxMin = Math.max(avgMin, Math.round((totalTests * maxPerTest) / 60));
        pillEta.textContent = avgMin === maxMin ? `${t.matrixEst}: ~${avgMin}m` : `${t.matrixEst}: ~${avgMin}m–${maxMin}m`;
    }
}

function startCountdown(totalTests) {
    isTestRunning = true;
    remainingEtaSec = Math.max(1, totalTests) * getPerTestSeconds();
    const dot = document.getElementById("pillLiveDot");
    const gridIcon = document.getElementById("pillGridIcon");
    if (dot) dot.style.display = "inline-block";
    if (gridIcon) gridIcon.style.display = "none";
    renderCountdownEta();
    if (countdownTimer) clearInterval(countdownTimer);
    countdownTimer = setInterval(() => {
        if (remainingEtaSec > 0) remainingEtaSec--;
        renderCountdownEta();
    }, 1000);
}

function syncCountdown(tested, total) {
    if (!isTestRunning) return;
    const pillText = document.getElementById("floatingMatrixText");
    if (pillText) {
        const lang = localStorage.getItem("ws_lang") || "en";
        const t = TRANSLATIONS[lang] || TRANSLATIONS.en;
        pillText.textContent = `${tested} / ${total} ${t.matrixTests}`;
    }
    const rem = Math.max(0, total - tested);
    remainingEtaSec = rem * getPerTestSeconds();
    renderCountdownEta();
}

function stopCountdown() {
    isTestRunning = false;
    if (countdownTimer) {
        clearInterval(countdownTimer);
        countdownTimer = null;
    }
    const dot = document.getElementById("pillLiveDot");
    const gridIcon = document.getElementById("pillGridIcon");
    if (dot) dot.style.display = "none";
    if (gridIcon) gridIcon.style.display = "inline-block";
    updateFloatingMatrix();
}

function updateFloatingMatrix() {
    if (isTestRunning) return;
    const pillText = document.getElementById("floatingMatrixText");
    if (!pillText) return;
    const lang = localStorage.getItem("ws_lang") || "en";
    const t = TRANSLATIONS[lang] || TRANSLATIONS.en;

    const locsCount = document.querySelectorAll(".cb-location:checked").length;
    const activeProtos = Object.keys(VALID_PROTOCOLS).filter(proto =>
        document.querySelectorAll(`.cb-port-${proto}:checked`).length > 0
    ).length;
    const portsCount = document.querySelectorAll('[class*="cb-port-"]:checked').length;
    const totalTests = locsCount * portsCount;

    remainingEtaSec = totalTests * getPerTestSeconds();
    pillText.textContent = `${t.matrixPrefix}: ${locsCount} ${t.matrixLocs} • ${activeProtos} ${t.matrixProtos} • ${portsCount} ${t.matrixPorts} → ${totalTests} ${t.matrixTests}`;
    renderCountdownEta();
}

function updateProtocolDependentGuards() {
    const lang = localStorage.getItem("ws_lang") || "en";
    const t = TRANSLATIONS[lang] || TRANSLATIONS.en;
    const hasChecked = (proto) => document.querySelectorAll(`.cb-port-${proto}:checked`).length > 0;
    const anyChecked = Object.keys(VALID_PROTOCOLS).some(hasChecked);
    const hasSNIProto = !anyChecked || hasChecked("stealth") || hasChecked("wstunnel");
    const hasTweakProto = !anyChecked || hasChecked("wireguard") || hasChecked("udp") || hasChecked("tcp") || hasChecked("stealth");

    const sniContainer = document.getElementById("sniContainer");
    const customSniInput = document.getElementById("customSni");
    const sniHint = document.getElementById("sniHint");
    if (sniContainer && customSniInput) {
        sniContainer.classList.toggle("is-proto-disabled", !hasSNIProto);
        customSniInput.disabled = !hasSNIProto;
        if (sniHint) {
            sniHint.textContent = hasSNIProto ? t.sniActiveNote : t.sniDisabledNote;
        }
    }

    const tweaksGroup = document.getElementById("tweaksModeGroup");
    const tweaksSelect = document.getElementById("protocolTweaksMode");
    const tweaksDesc = document.getElementById("protocolTweaksDesc");
    const amneziaSelect = document.getElementById("amneziaConfig");
    const btnRefreshAmn = document.getElementById("btnRefreshAmnezia");
    const amneziaGroup = document.getElementById("amneziaGroup");
    if (tweaksGroup && tweaksSelect) {
        tweaksGroup.classList.toggle("is-proto-disabled", !hasTweakProto);
        tweaksSelect.disabled = !hasTweakProto;
        if (!hasTweakProto) {
            if (amneziaGroup) amneziaGroup.classList.add("is-collapsed");
            if (amneziaSelect) amneziaSelect.disabled = true;
            if (btnRefreshAmn) btnRefreshAmn.disabled = true;
            if (tweaksDesc) tweaksDesc.textContent = t.tweaksDisabledNote;
        } else {
            if (tweaksDesc) tweaksDesc.textContent = t.tweaksActiveNote;
            updateTweaksState();
        }
    }
}

function computeNodeScore(item) {
    const down = item.download || 0;
    const up = item.upload || 0;
    const ping = item.ping > 0 ? item.ping : 2000;
    return (down * 10) + (up * 4) - (ping / 50);
}

function buildVerifiedRow(item) {
    const row = document.createElement("tr");
    row.className = "row-success";

    const pingVal = item.ping > 0 ? `${Math.round(item.ping)} ms` : "--";
    const jitterVal = item.jitter >= 0 && item.ping > 0 ? `${Math.round(item.jitter)} ms` : "--";
    const downVal = item.download > 0 ? `${item.download.toFixed(2)} Mbps` : "--";
    const upVal = item.upload > 0 ? `${item.upload.toFixed(2)} Mbps` : (item.upload === 0 ? "0.00 Mbps" : "--");
    const timeVal = item.timestamp ? (item.timestamp.split(" ")[1] || item.timestamp) : new Date().toLocaleTimeString();
    const presetVal = item.antiCensorshipMode && item.antiCensorshipMode !== "-" && item.antiCensorshipMode !== "None" ? item.antiCensorshipMode : "-";
    const sniVal = item.customSni && item.customSni !== "-" ? item.customSni : "-";
    const nickHtml = item.location ? ` <span style="color:#58a6ff; font-size:0.73rem;">(${escapeHtml(item.location)})</span>` : "";

    const cc = countryCodeToFlag(item.countryCode);
    const badgeHtml = cc ? `<span class="country-code-badge">${escapeHtml(cc)}</span>` : "";
    row.innerHTML = `
        <td class="cell-loc">${badgeHtml}<span>${escapeHtml(item.country || "")}</span> <span style="color:#8b949e; font-size:0.75rem;">${escapeHtml(item.city || "")}</span>${nickHtml}</td>
        <td class="cell-proto">${escapeHtml(item.protocol)}:${item.port}</td>
        <td class="cell-preset" style="color:#79c0ff; font-size:0.75rem;">${escapeHtml(presetVal)}</td>
        <td style="color:#8b949e; font-size:0.75rem;">${escapeHtml(sniVal)}</td>
        <td class="cell-ping">${pingVal}</td>
        <td style="color:#e3b341; font-size:0.78rem;">${jitterVal}</td>
        <td class="cell-speed">${downVal}</td>
        <td class="cell-speed" style="color:#d2a8ff;">${upVal}</td>
        <td style="color:#8b949e; font-size:0.75rem;">${escapeHtml(timeVal)}</td>
    `;
    return row;
}

function getSortedVerifiedNodes() {
    const sortVal = document.getElementById("nodesSortSelect")?.value || "time-desc";
    const copy = [...verifiedNodesList];
    switch (sortVal) {
        case "time-asc":
            return copy.sort((a, b) => (a._seq || 0) - (b._seq || 0));
        case "best":
            return copy.sort((a, b) => computeNodeScore(b) - computeNodeScore(a));
        case "download":
            return copy.sort((a, b) => (b.download || 0) - (a.download || 0));
        case "time-desc":
        default:
            return copy.sort((a, b) => (b._seq || 0) - (a._seq || 0));
    }
}

function renderVerifiedNodesTable() {
    const tableBody = document.getElementById("realtimeTableBody");
    const emptyState = document.getElementById("tableEmptyState");
    if (!tableBody) return;

    const sorted = getSortedVerifiedNodes();
    tableBody.innerHTML = "";
    if (sorted.length === 0) {
        if (emptyState) emptyState.style.display = "block";
        return;
    }
    if (emptyState) emptyState.style.display = "none";
    sorted.forEach(item => tableBody.appendChild(buildVerifiedRow(item)));
}

function appendConnectedRow(item) {
    const badge = document.getElementById("connectedBadge");
    connectedCount++;
    item._seq = connectedCount;
    verifiedNodesList.push(item);
    if (badge) badge.innerText = connectedCount;

    const btnShareNodes = document.getElementById("btnShareNodes");
    if (btnShareNodes) btnShareNodes.disabled = false;

    renderVerifiedNodesTable();
}

function resetConnectedTable() {
    const tableBody = document.getElementById("realtimeTableBody");
    const emptyState = document.getElementById("tableEmptyState");
    const badge = document.getElementById("connectedBadge");
    const btnShareNodes = document.getElementById("btnShareNodes");
    if (tableBody) tableBody.innerHTML = "";
    if (emptyState) emptyState.style.display = "block";
    if (badge) badge.innerText = "0";
    if (btnShareNodes) {
        btnShareNodes.disabled = true;
    }
    verifiedNodesList = [];
    connectedCount = 0;
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

function showGlobalErrorModal(title, message) {
    const modal = document.getElementById("errorModal");
    const t = document.getElementById("errorModalTitle");
    const m = document.getElementById("errorModalMessage");
    if (!modal) return;
    if (t) t.textContent = title || "Attention";
    if (m) m.textContent = message || "";
    modal.style.display = "flex";
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
            showGlobalErrorModal("Locations Unavailable", "Could not retrieve server locations from the Windscribe API or local cache. Check your internet connection or connect once and click Refresh.");
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
                const cCode = countryCodeToFlag(locations[0]?.countryCode);
                if (cCode) {
                    const cBadge = document.createElement('span');
                    cBadge.className = 'country-code-badge';
                    cBadge.textContent = cCode;
                    countryHeader.appendChild(document.createTextNode(' '));
                    countryHeader.appendChild(cBadge);
                }
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
        showGlobalErrorModal("Locations Fetch Error", "Failed to load locations: " + e);
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

    const tweaksMode = document.getElementById("protocolTweaksMode");
    if (tweaksMode) {
        tweaksMode.addEventListener("change", updateTweaksState);
    }

    const masterInjection = document.getElementById("masterInjection");
    const manualModeBanner = document.getElementById("manualModeBanner");
    if (masterInjection) {
        masterInjection.addEventListener("change", () => {
            if (manualModeBanner) {
                manualModeBanner.style.display = masterInjection.checked ? "none" : "block";
            }
        });
    }

    const btnReadClientSettings = document.getElementById("btnReadClientSettings");
    if (btnReadClientSettings) {
        btnReadClientSettings.addEventListener("click", async () => {
            btnReadClientSettings.disabled = true;
            const origText = btnReadClientSettings.innerText;
            btnReadClientSettings.innerText = "⏳ Reading Windscribe...";
            try {
                const snap = await window.go.main.App.ReadClientSettings();
                if (snap) {
                    if (snap.protocolTweaksMode && document.getElementById("protocolTweaksMode")) {
                        document.getElementById("protocolTweaksMode").value = snap.protocolTweaksMode;
                        updateTweaksState();
                    }
                    if (snap.amneziaConfig && document.getElementById("amneziaConfig")) {
                        document.getElementById("amneziaConfig").value = snap.amneziaConfig;
                    }
                    if (snap.serverRouting && document.getElementById("serverRouting")) {
                        document.getElementById("serverRouting").value = snap.serverRouting;
                    }
                    if (document.getElementById("customSni")) {
                        document.getElementById("customSni").value = snap.customSni || "";
                    }
                    if (document.getElementById("largeTls")) {
                        document.getElementById("largeTls").checked = !!snap.largeTls;
                    }
                    if (snap.internalDns && document.getElementById("internalDns")) {
                        document.getElementById("internalDns").value = snap.internalDns;
                    }
                    logMsg(`Loaded active Windscribe settings (Tweaks: ${snap.protocolTweaksMode}, Preset: ${snap.amneziaConfig || "-"}, Routing: ${snap.serverRouting}, DNS: ${snap.internalDns}).`, "log-success");
                }
            } catch (err) {
                logMsg(`Could not read Windscribe client settings: ${err}`, "log-error");
            } finally {
                btnReadClientSettings.disabled = false;
                btnReadClientSettings.innerText = origText;
            }
        });
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
    
    // Log & Table Tab Switching
    const tabLiveTable = document.getElementById("tabLiveTable");
    const tabLiveConsole = document.getElementById("tabLiveConsole");
    const liveTableTab = document.getElementById("liveTableTab");
    const logConsole = document.getElementById("logConsole");

    if (tabLiveTable && tabLiveConsole) {
        tabLiveTable.addEventListener("click", () => {
            tabLiveTable.classList.add("active");
            tabLiveConsole.classList.remove("active");
            if (liveTableTab) liveTableTab.style.display = "block";
            if (logConsole) logConsole.style.display = "none";
            const btnShareNodes = document.getElementById("btnShareNodes");
            const sortSelect = document.getElementById("nodesSortSelect");
            if (btnShareNodes) {
                btnShareNodes.style.display = "inline-flex";
                btnShareNodes.disabled = verifiedNodesList.length === 0;
            }
            if (sortSelect) sortSelect.style.display = "inline-block";
        });

        tabLiveConsole.addEventListener("click", () => {
            tabLiveConsole.classList.add("active");
            tabLiveTable.classList.remove("active");
            if (liveTableTab) liveTableTab.style.display = "none";
            if (logConsole) logConsole.style.display = "block";
            const btnShareNodes = document.getElementById("btnShareNodes");
            const sortSelect = document.getElementById("nodesSortSelect");
            if (btnShareNodes) btnShareNodes.style.display = "none";
            if (sortSelect) sortSelect.style.display = "none";
        });
    }

    const nodesSortSelect = document.getElementById("nodesSortSelect");
    if (nodesSortSelect) {
        nodesSortSelect.addEventListener("change", renderVerifiedNodesTable);
    }

    if (btnClearLogs) {
        btnClearLogs.addEventListener("click", () => {
            if (logConsole) logConsole.innerHTML = "";
            resetConnectedTable();
        });
    }

    document.getElementById("btnSelectAllLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = true);
        document.querySelectorAll(".cb-country").forEach(cb => cb.checked = true);
        document.querySelectorAll(".cb-continent").forEach(cb => cb.checked = true);
        updateFloatingMatrix();
    });
    
    document.getElementById("btnClearLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = false);
        document.querySelectorAll(".cb-country").forEach(cb => { cb.checked = false; cb.indeterminate = false; });
        document.querySelectorAll(".cb-continent").forEach(cb => { cb.checked = false; cb.indeterminate = false; });
        updateFloatingMatrix();
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
                showErrorModal("No Locations Selected", "Please select at least one location before starting the test.");
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

        if (!isResume && Object.keys(protocolPorts).length === 0) {
            showErrorModal("No Protocols Selected", "Please select at least one protocol & port combination.");
            return null;
        }

        const countryElem = document.getElementById("networkCountry");
        const ispElem = document.getElementById("networkIsp");
        const countryVal = (countryElem && countryElem.value !== "__custom__") ? countryElem.value : lastValidCountry;
        const ispName = (ispElem && ispElem.value !== "__custom__") ? ispElem.value : lastValidIsp;
        const ispVal = countryVal ? `${countryVal} - ${ispName}` : ispName;

        const masterInjectionElem = document.getElementById("masterInjection");
        const protocolTweaksModeElem = document.getElementById("protocolTweaksMode");
        const tweakModeVal = protocolTweaksModeElem ? protocolTweaksModeElem.value : "Enabled";
        const customSniElem = document.getElementById("customSni");
        let customSniVal = customSniElem ? customSniElem.value.trim() : "";
        if (customSniVal === "example.com") customSniVal = "";

        return {
            selectedLocations: selectedLocations,
            protocolPorts: protocolPorts,
            masterInjection: masterInjectionElem ? masterInjectionElem.checked : true,
            protocolTweaksMode: tweakModeVal,
            protocolTweaks: (tweakModeVal.toLowerCase() === "enabled"),
            amneziaConfig: document.getElementById("amneziaConfig") ? document.getElementById("amneziaConfig").value : "Auto",
            serverRouting: document.getElementById("serverRouting").value,
            customSni: customSniVal,
            largeTls: document.getElementById("largeTls").checked,
            internalDns: document.getElementById("internalDns").value,
            isp: ispVal,
            isResume: isResume,
            selectionMode: selectionMode,
            testPing: true,
            testDownload: true,
            testUpload: true,
            timeout: parseInt(document.getElementById("connTimeout").value) || 25,
            speedWait: parseInt(document.getElementById("speedWait").value) || 3,
            stabilizeEnabled: document.getElementById("stabilizeEnabled") ? document.getElementById("stabilizeEnabled").checked : true,
            stabilizeMinBytes: parseInt(document.getElementById("stabilizeMinBytes")?.value) || 250,
            stabilizeUnit: document.getElementById("stabilizeUnitBtn")?.dataset.unit || "KB",
            stabilizeTimeout: parseInt(document.getElementById("stabilizeTimeout")?.value) || 30,
            speedEngine: document.getElementById("speedEngine")?.value || "ookla",
            shuffleLocations: document.getElementById("shuffleLocations")?.checked || false,
            shuffleProtocols: document.getElementById("shuffleProtocols")?.checked || false,
            connectAndHold: document.getElementById("connectAndHold")?.checked || false
        };
    }

    const btnContinue = document.getElementById("btnContinue");

    // Handle Live Events from Go Orchestrator
    window.runtime.EventsOn("test_update", (update) => {
        if (update.message) logMsg(update.message, "log-info");
        
        // Append successful connection to real-time table
        if (update.connectedItem) {
            appendConnectedRow(update.connectedItem);
        }
        
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

        if (update.errorTitle || update.errorMsg) {
            stopCountdown();
            showErrorModal(update.errorTitle || "Error", update.errorMsg || update.message);
        }

        if (update.totalCount > 0 && typeof update.testedCount === "number") {
            syncCountdown(update.testedCount, update.totalCount);
        }

        // Handle CanResume state
        if (update.canResume) {
            stopCountdown();
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
        } else if (update.status === "Done" || update.status === "Connected & Held" || update.status === "Connected (Paused)") {
            stopCountdown();
            if (btnContinue) btnContinue.style.display = "none";
            btnStart.innerText = "START ANALYZING";
            btnStart.disabled = false;
            btnStop.disabled = true;
            btnStop.innerText = "STOP SESSION";
        } else if (update.status === "Stopped") {
            stopCountdown();
            btnStop.disabled = true;
            btnStop.innerText = "STOP SESSION";
            btnStart.disabled = false;
            if (btnContinue && !update.canResume) {
                btnContinue.style.display = "none";
            }
        }
    });

    // Background Sync Event Listeners
    window.runtime.EventsOn("locations_updated", (newLocs) => {
        logMsg("[Auto-Sync] Received updated locations from Windscribe network.", "log-success");
        fetchLocations(false);
    });

    window.runtime.EventsOn("amnezia_updated", (newConfigs) => {
        logMsg("[Auto-Sync] Received updated Amnezia presets from Windscribe Engine.", "log-success");
        loadAmneziaConfigs(false);
    });

    window.runtime.EventsOn("sync_queued", (msg) => {
        logMsg(`[Sync Queued] ${msg}`, "log-info");
    });

    // Start Button Handler (Starts fresh)
    btnStart.addEventListener("click", () => {
        const config = collectTestConfig(false);
        if (!config) return;

        resetConnectedTable();
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
        const estTests = (config.selectedLocations?.length || 1) * Object.values(config.protocolPorts || {}).reduce((a, b) => a + b.length, 0);
        startCountdown(estTests);
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
            const remCount = parseInt(document.getElementById("remainingCountTxt")?.innerText) || 1;
            startCountdown(remCount);
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
            updateFloatingMatrix();
        });
    });

    // Tunnel Stabilization Toggle & Unit Badge (KB / MB)
    const stabilizeEnabled = document.getElementById("stabilizeEnabled");
    const stabilizeUnitBtn = document.getElementById("stabilizeUnitBtn");
    const stabilizeUnitLabel = document.getElementById("stabilizeUnitLabel");
    const stabilizeControlsRow = document.getElementById("stabilizeControlsRow");
    const stabilizeMinBytes = document.getElementById("stabilizeMinBytes");
    const stabilizeTimeout = document.getElementById("stabilizeTimeout");

    if (stabilizeUnitBtn) {
        stabilizeUnitBtn.dataset.unit = "KB";
        stabilizeUnitBtn.addEventListener("click", () => {
            const nextUnit = stabilizeUnitBtn.dataset.unit === "KB" ? "MB" : "KB";
            stabilizeUnitBtn.dataset.unit = nextUnit;
            stabilizeUnitBtn.textContent = nextUnit;
            stabilizeUnitBtn.classList.toggle("is-mb", nextUnit === "MB");
            if (stabilizeUnitLabel) stabilizeUnitLabel.textContent = nextUnit;
        });
    }

    const syncStabilizeDisabledState = () => {
        const enabled = stabilizeEnabled ? stabilizeEnabled.checked : true;
        if (stabilizeControlsRow) {
            stabilizeControlsRow.classList.toggle("is-disabled", !enabled);
            stabilizeControlsRow.classList.toggle("is-collapsed", !enabled);
        }
        if (stabilizeUnitBtn) stabilizeUnitBtn.disabled = !enabled;
        if (stabilizeMinBytes) stabilizeMinBytes.disabled = !enabled;
        if (stabilizeTimeout) stabilizeTimeout.disabled = !enabled;
        updateFloatingMatrix();
    };

    if (stabilizeEnabled) {
        stabilizeEnabled.addEventListener("change", syncStabilizeDisabledState);
        syncStabilizeDisabledState();
    }

    // Custom Dark Error Modal
    const errorModal = document.getElementById("errorModal");
    const errorModalTitle = document.getElementById("errorModalTitle");
    const errorModalMessage = document.getElementById("errorModalMessage");
    const errorModalOk = document.getElementById("errorModalOk");
    const errorModalClose = document.getElementById("errorModalClose");

    function showErrorModal(title, message) {
        if (!errorModal) return;
        if (errorModalTitle) errorModalTitle.textContent = title || "Attention";
        if (errorModalMessage) errorModalMessage.textContent = message || "";
        errorModal.style.display = "flex";
    }

    function hideErrorModal() {
        if (errorModal) errorModal.style.display = "none";
    }

    if (errorModalOk) errorModalOk.addEventListener("click", hideErrorModal);
    if (errorModalClose) errorModalClose.addEventListener("click", hideErrorModal);
    if (errorModal) {
        errorModal.addEventListener("click", (e) => {
            if (e.target === errorModal) hideErrorModal();
        });
    }
    document.addEventListener("keydown", (e) => {
        if (e.key === "Escape" && errorModal && errorModal.style.display === "flex") {
            hideErrorModal();
        }
    });

    // Master Shuffle All sync
    const shuffleAll = document.getElementById("shuffleAll");
    const shuffleLocs = document.getElementById("shuffleLocations");
    const shuffleProtos = document.getElementById("shuffleProtocols");
    const syncShuffleMaster = () => {
        if (!shuffleAll || !shuffleLocs || !shuffleProtos) return;
        const l = shuffleLocs.checked;
        const p = shuffleProtos.checked;
        shuffleAll.checked = l && p;
        shuffleAll.indeterminate = l !== p;
    };
    if (shuffleAll && shuffleLocs && shuffleProtos) {
        shuffleAll.addEventListener("change", () => {
            shuffleLocs.checked = shuffleAll.checked;
            shuffleProtos.checked = shuffleAll.checked;
            shuffleAll.indeterminate = false;
        });
        shuffleLocs.addEventListener("change", syncShuffleMaster);
        shuffleProtos.addEventListener("change", syncShuffleMaster);
    }

    // Trigger matrix & protocol guards on selection changes
    ["connTimeout", "speedWait", "stabilizeTimeout"].forEach(id => {
        const el = document.getElementById(id);
        if (el) el.addEventListener("input", () => updateFloatingMatrix());
    });
    document.addEventListener("change", (e) => {
        if (e.target && (
            e.target.classList.contains("cb-location") ||
            e.target.classList.contains("cb-country") ||
            e.target.classList.contains("cb-continent") ||
            e.target.classList.contains("cb-proto") ||
            e.target.id === "cbSelectAllProtocols" ||
            e.target.id === "speedEngine" ||
            e.target.id === "speedWait" ||
            e.target.id === "connTimeout" ||
            e.target.id === "stabilizeTimeout" ||
            Array.from(e.target.classList).some(c => c.startsWith("cb-port-"))
        )) {
            updateFloatingMatrix();
            updateProtocolDependentGuards();
        }
    });
    updateFloatingMatrix();
    updateProtocolDependentGuards();

    // Multi-Language (Separated from CLI Path Settings)
    function applyLanguage(langCode) {
        const lang = TRANSLATIONS[langCode] ? langCode : "en";
        const dict = TRANSLATIONS[lang];
        localStorage.setItem("ws_lang", lang);
        document.documentElement.dir = dict.dir;
        document.documentElement.lang = lang;

        document.querySelectorAll("[data-i18n]").forEach(el => {
            const key = el.getAttribute("data-i18n");
            if (dict[key]) el.textContent = dict[key];
        });
        document.querySelectorAll("[data-i18n-placeholder]").forEach(el => {
            const key = el.getAttribute("data-i18n-placeholder");
            if (dict[key]) el.setAttribute("placeholder", dict[key]);
        });

        const btnStartEl = document.getElementById("btnStart");
        if (btnStartEl && !btnStartEl.disabled) btnStartEl.textContent = dict.startAnalyzing;
        const btnStopEl = document.getElementById("btnStop");
        if (btnStopEl && btnStopEl.disabled) btnStopEl.textContent = dict.stopSession;

        updateFloatingMatrix();
        updateProtocolDependentGuards();
    }

    // CLI Path Settings Modal (Checkbox-controlled)
    const btnOpenSettings = document.getElementById("btnOpenSettings");
    const settingsModal = document.getElementById("settingsModal");
    const settingsModalClose = document.getElementById("settingsModalClose");
    const settingsSaveBtn = document.getElementById("settingsSaveBtn");
    const settingsResetBtn = document.getElementById("settingsResetBtn");
    const useCustomCliPath = document.getElementById("useCustomCliPath");
    const customCliPathBox = document.getElementById("customCliPathBox");
    const settingsCliPath = document.getElementById("settingsCliPath");

    const syncCustomCliBox = () => {
        if (!useCustomCliPath || !customCliPathBox) return;
        customCliPathBox.classList.toggle("is-collapsed", !useCustomCliPath.checked);
        if (settingsCliPath) settingsCliPath.disabled = !useCustomCliPath.checked;
    };
    if (useCustomCliPath) {
        useCustomCliPath.addEventListener("change", syncCustomCliBox);
    }

    const savedCliEnabled = localStorage.getItem("ws_custom_cli_enabled") === "true";
    const savedCliPath = localStorage.getItem("ws_custom_cli_path") || "";
    if (savedCliEnabled && savedCliPath && window.go?.main?.App?.SetCustomCLIPath) {
        window.go.main.App.SetCustomCLIPath(savedCliPath);
    }

    if (btnOpenSettings && settingsModal) {
        btnOpenSettings.addEventListener("click", async () => {
            const enabled = localStorage.getItem("ws_custom_cli_enabled") === "true";
            if (useCustomCliPath) useCustomCliPath.checked = enabled;
            if (settingsCliPath) {
                let currentPath = localStorage.getItem("ws_custom_cli_path") || "";
                if (!currentPath && window.go?.main?.App?.GetCustomCLIPath) {
                    currentPath = await window.go.main.App.GetCustomCLIPath();
                }
                settingsCliPath.value = currentPath;
            }
            syncCustomCliBox();
            settingsModal.style.display = "flex";
        });
    }
    if (settingsModalClose) settingsModalClose.addEventListener("click", () => { settingsModal.style.display = "none"; });
    if (settingsModal) {
        settingsModal.addEventListener("click", (e) => {
            if (e.target === settingsModal) settingsModal.style.display = "none";
        });
    }
    if (settingsSaveBtn) {
        settingsSaveBtn.addEventListener("click", async () => {
            const enabled = useCustomCliPath ? useCustomCliPath.checked : false;
            localStorage.setItem("ws_custom_cli_enabled", enabled ? "true" : "false");
            const val = (enabled && settingsCliPath) ? settingsCliPath.value.trim() : "";
            if (enabled) {
                localStorage.setItem("ws_custom_cli_path", val);
            } else {
                localStorage.removeItem("ws_custom_cli_path");
            }
            if (window.go?.main?.App?.SetCustomCLIPath) {
                const resolved = await window.go.main.App.SetCustomCLIPath(val);
                logMsg(`[Settings] Windscribe CLI path: ${resolved}`, "log-info");
            }
            if (settingsModal) settingsModal.style.display = "none";
        });
    }
    if (settingsResetBtn) {
        settingsResetBtn.addEventListener("click", async () => {
            if (useCustomCliPath) useCustomCliPath.checked = false;
            localStorage.setItem("ws_custom_cli_enabled", "false");
            localStorage.removeItem("ws_custom_cli_path");
            syncCustomCliBox();
            if (window.go?.main?.App?.SetCustomCLIPath) {
                const resolved = await window.go.main.App.SetCustomCLIPath("");
                if (settingsCliPath) settingsCliPath.value = resolved;
                logMsg(`[Settings] Windscribe CLI path reset to default: ${resolved}`, "log-info");
            }
        });
    }

    // Language Selector, Startup Network Prompt, Environment Check & Guided Tour
    const onboardingModal = document.getElementById("onboardingLangModal");
    const btnOpenLang = document.getElementById("btnOpenLang");
    const networkSetupModal = document.getElementById("networkSetupModal");
    const modalCountrySelect = document.getElementById("modalCountrySelect");
    const modalIspSelect = document.getElementById("modalIspSelect");
    const modalCustomCountryInput = document.getElementById("modalCustomCountryInput");
    const modalCustomIspInput = document.getElementById("modalCustomIspInput");
    const btnConfirmNetworkSetup = document.getElementById("btnConfirmNetworkSetup");

    function populateModalNetworkDropdowns() {
        if (!modalCountrySelect || !modalIspSelect) return;
        const defaultCountries = ["Iran", "Russia", "China", "Other / Global"];
        const allCountries = [...defaultCountries];
        getStoredCustomCountries().forEach(c => { if (!allCountries.includes(c)) allCountries.push(c); });

        modalCountrySelect.innerHTML = "";
        allCountries.forEach(c => {
            const opt = document.createElement("option");
            opt.value = c;
            opt.textContent = c;
            modalCountrySelect.appendChild(opt);
        });
        const custCOpt = document.createElement("option");
        custCOpt.value = "__custom__";
        custCOpt.textContent = "+ Custom Country...";
        modalCountrySelect.appendChild(custCOpt);

        const sidebarCountry = document.getElementById("networkCountry")?.value;
        if (sidebarCountry && allCountries.includes(sidebarCountry)) {
            modalCountrySelect.value = sidebarCountry;
        }

        const fillModalIsps = (country) => {
            const defs = DEFAULT_OPERATORS[country] || ["Other / Direct"];
            const allOps = [...defs];
            getStoredCustomOperators(country).forEach(o => { if (!allOps.includes(o)) allOps.push(o); });
            modalIspSelect.innerHTML = "";
            allOps.forEach(op => {
                const opt = document.createElement("option");
                opt.value = op;
                opt.textContent = op;
                modalIspSelect.appendChild(opt);
            });
            const custOpOpt = document.createElement("option");
            custOpOpt.value = "__custom__";
            custOpOpt.textContent = "+ Custom ISP / Operator...";
            modalIspSelect.appendChild(custOpOpt);
            if (modalCustomIspInput) modalCustomIspInput.style.display = "none";
        };

        fillModalIsps(modalCountrySelect.value);
        modalCountrySelect.onchange = () => {
            if (modalCountrySelect.value === "__custom__") {
                if (modalCustomCountryInput) modalCustomCountryInput.style.display = "block";
                fillModalIsps("Other / Global");
            } else {
                if (modalCustomCountryInput) modalCustomCountryInput.style.display = "none";
                fillModalIsps(modalCountrySelect.value);
            }
        };
        modalIspSelect.onchange = () => {
            if (modalCustomIspInput) {
                modalCustomIspInput.style.display = modalIspSelect.value === "__custom__" ? "block" : "none";
            }
        };
    }

    function openNetworkSetupModal() {
        if (!networkSetupModal) return;
        populateModalNetworkDropdowns();
        networkSetupModal.style.display = "flex";
    }

    if (btnConfirmNetworkSetup && networkSetupModal) {
        btnConfirmNetworkSetup.addEventListener("click", () => {
            let chosenCountry = modalCountrySelect?.value || "Iran";
            if (chosenCountry === "__custom__") {
                const typed = modalCustomCountryInput?.value.trim();
                if (!typed) {
                    showErrorModal("Country Required", "Please enter your country name.");
                    return;
                }
                saveCustomCountry(typed);
                chosenCountry = typed;
            }
            let chosenIsp = modalIspSelect?.value || "";
            if (chosenIsp === "__custom__") {
                const typedOp = modalCustomIspInput?.value.trim();
                if (!typedOp) {
                    showErrorModal("Operator Required", "Please enter your ISP / operator name.");
                    return;
                }
                saveCustomOperator(chosenCountry, typedOp);
                chosenIsp = typedOp;
            }
            populateCountrySelect(chosenCountry);
            populateIspSelect(chosenCountry, chosenIsp);
            lastValidCountry = chosenCountry;
            lastValidIsp = chosenIsp;
            networkSetupModal.style.display = "none";

            if (!localStorage.getItem("ws_tour_done")) {
                startGuidedTour();
            }
        });
    }

    // Interactive Spotlight Guided Tour
    const tourOverlay = document.getElementById("tourOverlay");
    const tourSpotlight = document.getElementById("tourSpotlight");
    const tourTooltip = document.getElementById("tourTooltip");
    const tourStepTitle = document.getElementById("tourStepTitle");
    const tourStepCounter = document.getElementById("tourStepCounter");
    const tourStepBody = document.getElementById("tourStepBody");
    const btnTourPrev = document.getElementById("btnTourPrev");
    const btnTourNext = document.getElementById("btnTourNext");
    const btnTourSkip = document.getElementById("btnTourSkip");
    const btnStartTour = document.getElementById("btnStartTour");
    let currentTourIdx = 0;
    let tourResizeObserver = null;

    function positionSpotlightForTarget(target) {
        if (!target || !tourSpotlight || !tourTooltip) return;
        const sidebarBody = document.querySelector(".sidebar-scroll-body");
        let rect = target.getBoundingClientRect();

        if (sidebarBody && sidebarBody.contains(target)) {
            const sbRect = sidebarBody.getBoundingClientRect();
            const top = Math.max(sbRect.top + 2, rect.top);
            const bottom = Math.min(sbRect.bottom - 2, rect.bottom);
            if (bottom > top) {
                rect = {
                    top: top,
                    left: rect.left,
                    right: rect.right,
                    bottom: bottom,
                    width: rect.width,
                    height: bottom - top
                };
            }
        }

        const pad = 4;
        tourSpotlight.style.top = `${Math.max(4, rect.top - pad)}px`;
        tourSpotlight.style.left = `${Math.max(4, rect.left - pad)}px`;
        tourSpotlight.style.width = `${rect.width + pad * 2}px`;
        tourSpotlight.style.height = `${rect.height + pad * 2}px`;

        let tipTop = rect.top;
        let tipLeft = rect.right + 16;
        if (rect.right + 345 < window.innerWidth) {
            tipLeft = rect.right + 16;
            tipTop = Math.max(16, Math.min(window.innerHeight - 210, rect.top));
        } else {
            tipTop = rect.top - 190;
            if (tipTop < 16) tipTop = Math.min(window.innerHeight - 200, rect.bottom + 14);
            tipLeft = Math.max(16, Math.min(window.innerWidth - 336, rect.left));
        }
        tourTooltip.style.top = `${tipTop}px`;
        tourTooltip.style.left = `${tipLeft}px`;
    }

    function renderTourStep() {
        if (!tourOverlay || !tourSpotlight || !tourTooltip) return;
        const lang = localStorage.getItem("ws_lang") || "en";
        const dict = TRANSLATIONS[lang] || TRANSLATIONS.en;
        const steps = dict.tourSteps || TRANSLATIONS.en.tourSteps;
        if (!steps || steps.length === 0) return;
        if (currentTourIdx < 0) currentTourIdx = 0;
        if (currentTourIdx >= steps.length) {
            closeGuidedTour(true);
            return;
        }
        if (tourResizeObserver) {
            tourResizeObserver.disconnect();
            tourResizeObserver = null;
        }
        const step = steps[currentTourIdx];
        const target = document.querySelector(step.target);
        if (target) {
            target.scrollIntoView({ behavior: "instant", block: "start" });
            requestAnimationFrame(() => positionSpotlightForTarget(target));
            if (typeof ResizeObserver !== "undefined") {
                tourResizeObserver = new ResizeObserver(() => positionSpotlightForTarget(target));
                tourResizeObserver.observe(target);
            }
        }
        if (tourStepTitle) tourStepTitle.textContent = step.title;
        if (tourStepBody) tourStepBody.textContent = step.body;
        if (tourStepCounter) tourStepCounter.textContent = `${currentTourIdx + 1} / ${steps.length}`;
        if (btnTourPrev) btnTourPrev.disabled = (currentTourIdx === 0);
        if (btnTourNext) btnTourNext.textContent = (currentTourIdx === steps.length - 1) ? dict.tourDone : dict.tourNext;
    }

    function startGuidedTour() {
        if (!tourOverlay) return;
        currentTourIdx = 0;
        tourOverlay.style.display = "block";
        renderTourStep();
    }

    function closeGuidedTour(markDone) {
        if (tourResizeObserver) {
            tourResizeObserver.disconnect();
            tourResizeObserver = null;
        }
        if (tourOverlay) tourOverlay.style.display = "none";
        if (markDone) localStorage.setItem("ws_tour_done", "true");
    }

    if (btnTourPrev) btnTourPrev.addEventListener("click", () => { currentTourIdx--; renderTourStep(); });
    if (btnTourNext) btnTourNext.addEventListener("click", () => { currentTourIdx++; renderTourStep(); });
    if (btnTourSkip) btnTourSkip.addEventListener("click", () => closeGuidedTour(true));
    if (btnStartTour) btnStartTour.addEventListener("click", startGuidedTour);
    const sidebarScrollEl = document.querySelector(".sidebar-scroll-body");
    const syncTourOnScrollOrResize = () => {
        if (!tourOverlay || tourOverlay.style.display !== "block") return;
        const lang = localStorage.getItem("ws_lang") || "en";
        const steps = (TRANSLATIONS[lang] || TRANSLATIONS.en).tourSteps;
        const step = steps && steps[currentTourIdx];
        if (step) {
            const target = document.querySelector(step.target);
            if (target) positionSpotlightForTarget(target);
        }
    };
    if (sidebarScrollEl) sidebarScrollEl.addEventListener("scroll", syncTourOnScrollOrResize);
    window.addEventListener("resize", syncTourOnScrollOrResize);

    async function runStartupEnvCheck() {
        try {
            if (window.go?.main?.App?.CheckEnvironment) {
                const env = await window.go.main.App.CheckEnvironment();
                if (env && !env.cliFound) {
                    showErrorModal("Windscribe CLI Not Found", env.errorMessage || "Windscribe does not appear to be installed or windscribe-cli path is invalid.");
                } else if (env && env.sessionLocked) {
                    showErrorModal("Windscribe Session Mismatch", env.errorMessage);
                }
            }
        } catch (e) {
            console.warn("Env check skipped:", e);
        }
    }

    if (onboardingModal) {
        onboardingModal.querySelectorAll(".lang-choice-btn").forEach(btn => {
            btn.addEventListener("click", () => {
                const isFirstRun = !localStorage.getItem("ws_lang");
                applyLanguage(btn.dataset.lang || "en");
                onboardingModal.style.display = "none";
                if (isFirstRun) openNetworkSetupModal();
            });
        });
        onboardingModal.addEventListener("click", (e) => {
            if (e.target === onboardingModal && localStorage.getItem("ws_lang")) {
                onboardingModal.style.display = "none";
            }
        });
    }
    if (btnOpenLang && onboardingModal) {
        btnOpenLang.addEventListener("click", () => {
            onboardingModal.style.display = "flex";
        });
    }
    const savedLang = localStorage.getItem("ws_lang");
    if (!savedLang && onboardingModal) {
        onboardingModal.style.display = "flex";
    } else {
        applyLanguage(savedLang || "en");
        openNetworkSetupModal();
    }
    runStartupEnvCheck();

    // Clean Share Verified Nodes (Text + High-DPI PNG Card)
    const btnShareNodes = document.getElementById("btnShareNodes");
    const shareModal = document.getElementById("shareModal");
    const shareModalClose = document.getElementById("shareModalClose");
    const sharePreviewCanvas = document.getElementById("sharePreviewCanvas");
    const shareTextArea = document.getElementById("shareTextArea");
    const btnCopyShareText = document.getElementById("btnCopyShareText");
    const btnCopyShareImg = document.getElementById("btnCopyShareImg");
    const btnDownloadShareImg = document.getElementById("btnDownloadShareImg");

    function renderShareContent() {
        if (!sharePreviewCanvas || !shareTextArea) return;
        const rows = getSortedVerifiedNodes().slice(0, 14);
        const activeIsp = rows[0]?.isp || `${lastValidCountry} - ${lastValidIsp}`;
        const lines = [
            "Windscribe Analyzer — Verified Nodes",
            `Network: ${activeIsp}`,
            "========================================"
        ];
        rows.forEach((n, idx) => {
            const nick = n.location ? ` (${n.location})` : "";
            const sni = n.customSni && n.customSni !== "-" ? ` | SNI: ${n.customSni}` : "";
            const preset = n.antiCensorshipMode && n.antiCensorshipMode !== "-" && n.antiCensorshipMode !== "None" ? ` | Preset: ${n.antiCensorshipMode}` : "";
            lines.push(`${idx + 1}. ${n.country} - ${n.city}${nick} | ${n.protocol}:${n.port}${preset}${sni} | Ping: ${Math.round(n.ping)}ms (±${Math.round(n.jitter || 0)}ms) | ↓${(n.download || 0).toFixed(2)} ↑${(n.upload || 0).toFixed(2)} Mbps`);
        });
        shareTextArea.value = lines.join("\n");

        const scale = 2; // HiDPI 2x sharp rendering
        const rowH = 34;
        const headerH = 92;
        const W = 940;
        const H = headerH + Math.max(1, rows.length) * rowH + 26;
        sharePreviewCanvas.width = W * scale;
        sharePreviewCanvas.height = H * scale;
        const ctx = sharePreviewCanvas.getContext("2d");
        ctx.scale(scale, scale);

        ctx.fillStyle = "#0d1117";
        ctx.fillRect(0, 0, W, H);

        ctx.fillStyle = "#161b22";
        ctx.fillRect(0, 0, W, 62);
        ctx.fillStyle = "#58a6ff";
        ctx.font = "bold 18px Segoe UI, sans-serif";
        ctx.fillText("Windscribe Analyzer — Verified Nodes", 22, 30);
        ctx.fillStyle = "#8b949e";
        ctx.font = "12px Segoe UI, sans-serif";
        ctx.fillText(`Network: ${activeIsp}`, 22, 50);
        ctx.fillText(`${verifiedNodesList.length} verified`, W - 95, 36);

        ctx.fillStyle = "#8b949e";
        ctx.font = "bold 11px Segoe UI, sans-serif";
        ctx.fillText("LOCATION", 22, 82);
        ctx.fillText("PROTOCOL", 370, 82);
        ctx.fillText("PRESET", 475, 82);
        ctx.fillText("SNI", 590, 82);
        ctx.fillText("PING / JITTER", 695, 82);
        ctx.fillText("DOWN / UP", 815, 82);

        rows.forEach((n, idx) => {
            const y = headerH + idx * rowH + 23;
            if (idx % 2 === 1) {
                ctx.fillStyle = "rgba(255,255,255,0.028)";
                ctx.fillRect(12, y - 20, W - 24, rowH);
            }
            const locStr = `${n.country || ""} - ${n.city || ""}${n.location ? " (" + n.location + ")" : ""}`;
            ctx.fillStyle = "#e6edf3";
            ctx.font = "12.5px Segoe UI, sans-serif";
            ctx.fillText(locStr, 22, y, 336);

            ctx.fillStyle = "#79c0ff";
            ctx.fillText(`${n.protocol}:${n.port}`, 370, y);

            const presetStr = (n.antiCensorshipMode && n.antiCensorshipMode !== "-" && n.antiCensorshipMode !== "None") ? n.antiCensorshipMode : "-";
            ctx.fillStyle = "#a5d6ff";
            ctx.fillText(presetStr, 475, y, 105);

            ctx.fillStyle = "#8b949e";
            ctx.fillText(n.customSni && n.customSni !== "-" ? n.customSni : "-", 590, y, 95);

            ctx.fillStyle = "#e3b341";
            ctx.fillText(`${Math.round(n.ping)}ms (±${Math.round(n.jitter || 0)})`, 695, y);

            ctx.fillStyle = "#3fb950";
            ctx.fillText(`↓${(n.download || 0).toFixed(1)} ↑${(n.upload || 0).toFixed(1)}`, 815, y);
        });
    }

    if (btnShareNodes && shareModal) {
        btnShareNodes.addEventListener("click", () => {
            renderShareContent();
            shareModal.style.display = "flex";
        });
    }
    if (shareModalClose) shareModalClose.addEventListener("click", () => { shareModal.style.display = "none"; });
    if (shareModal) {
        shareModal.addEventListener("click", (e) => {
            if (e.target === shareModal) shareModal.style.display = "none";
        });
    }
    if (btnCopyShareText && shareTextArea) {
        btnCopyShareText.addEventListener("click", async () => {
            await navigator.clipboard.writeText(shareTextArea.value);
            btnCopyShareText.textContent = "Copied";
            setTimeout(() => { btnCopyShareText.textContent = "Copy Text"; }, 1500);
        });
    }
    if (btnCopyShareImg && sharePreviewCanvas) {
        btnCopyShareImg.addEventListener("click", () => {
            sharePreviewCanvas.toBlob(async (blob) => {
                if (blob && navigator.clipboard && window.ClipboardItem) {
                    await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
                    btnCopyShareImg.textContent = "Copied";
                    setTimeout(() => { btnCopyShareImg.textContent = "Copy Image"; }, 1500);
                }
            }, "image/png");
        });
    }
    if (btnDownloadShareImg && sharePreviewCanvas) {
        btnDownloadShareImg.addEventListener("click", () => {
            const link = document.createElement("a");
            link.download = `windscribe-verified-nodes-${Date.now()}.png`;
            link.href = sharePreviewCanvas.toDataURL("image/png");
            link.click();
        });
    }
});
