const VALID_PROTOCOLS = {
    "wireguard": [443, 1194, 2049, 2082, 51820, 65142],
    "stealth": [80, 443, 8080, 8443],
    "wstunnel": [443],
    "tcp": [21, 22, 80, 110, 143, 443, 587, 1194, 3306, 8080, 5432, 8443],
    "udp": [53, 80, 123, 443, 1194, 54783]
};

function logMsg(msg, type = "log-msg") {
    const logConsole = document.getElementById("logConsole");
    const div = document.createElement("div");
    div.className = type;
    const isScrolledToBottom = logConsole.scrollHeight - logConsole.clientHeight <= logConsole.scrollTop + 50;
    
    const time = new Date().toLocaleTimeString();
    div.innerText = `[${time}] ${msg}`;
    logConsole.appendChild(div);
    
    // Force scroll to bottom only if user was already at the bottom
    if (isScrolledToBottom) {
        setTimeout(() => {
            logConsole.scrollTo({ top: logConsole.scrollHeight, behavior: 'smooth' });
        }, 10);
    }
}

function initProtocols() {
    const container = document.getElementById("protocolPortsContainer");
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
        if (proto === "wireguard" || proto === "stealth") cbProto.checked = true;
        
        header.appendChild(cbProto);
        header.appendChild(document.createTextNode(" " + proto.charAt(0).toUpperCase() + proto.slice(1)));
        
        const grid = document.createElement("div");
        grid.className = "ports-grid";
        
        VALID_PROTOCOLS[proto].forEach((port, idx) => {
            const lbl = document.createElement("label");
            const cbPort = document.createElement("input");
            cbPort.type = "checkbox";
            cbPort.value = port;
            cbPort.className = `cb-port-${proto}`;
            if (idx === 0 || port === 443) cbPort.checked = true;
            
            lbl.appendChild(cbPort);
            lbl.appendChild(document.createTextNode(" " + port));
            grid.appendChild(lbl);
        });
        
        box.appendChild(header);
        box.appendChild(grid);
        container.appendChild(box);
    });
}

async function fetchLocations() {
    try {
        const treeContainer = document.getElementById("locationTree");
        treeContainer.innerHTML = "<div style='padding:10px; color:#888;'>Loading...</div>";
        const locs = await window.go.main.App.GetAllLocations();
        treeContainer.innerHTML = "";
        
        if (!locs || locs.length === 0) {
            treeContainer.innerHTML = "<div style='padding:10px; color:#da3633;'>Failed to load locations.</div>";
            return;
        }

        // Group by Continent -> Region (Country) -> Location
        const tree = {};
        locs.forEach(loc => {
            const cont = loc.continent || "Other";
            const reg = loc.region;
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
                if(e.target === contCb) return; 
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
                    if(e.target === cCb) return; 
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
    initProtocols();
    
    // Wait slightly to ensure Wails runtime is loaded before calling backend
    setTimeout(fetchLocations, 500);

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
    
    document.getElementById("btnSelectAllLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = true);
    });
    
    document.getElementById("btnClearLocs").addEventListener("click", () => {
        document.querySelectorAll(".cb-location").forEach(cb => cb.checked = false);
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

    window.runtime.EventsOn("test_update", (update) => {
        if (update.message) logMsg(update.message, "log-info");
        
        if (update.ping) document.getElementById("valPing").innerText = update.ping;
        if (update.down) document.getElementById("valDown").innerText = update.down;
        if (update.up) document.getElementById("valUp").innerText = update.up;
        if (update.status) document.getElementById("valStatus").innerText = update.status;
        
        if (update.status === "Done" || update.status === "Stopped") {
            btnStart.disabled = false;
            btnStop.disabled = true;
        }
    });

    btnStart.addEventListener("click", () => {
        const selectionMode = document.getElementById("selectionMode").value;
        let locations = [];
        
        if (selectionMode === "manual" || selectionMode === "discovery") {
            document.querySelectorAll(".cb-location:checked").forEach(cb => locations.push(cb.value));
            if (locations.length === 0) {
                alert("Please select at least one location!");
                return;
            }
        }
        
        btnStart.disabled = true;
        btnStop.disabled = false;
        document.getElementById("valPing").innerText = "--";
        document.getElementById("valDown").innerText = "--";
        document.getElementById("valUp").innerText = "--";
        
        logMsg("Starting Analyzer...", "log-success");

        const protocolPorts = {};
        document.querySelectorAll(".cb-proto:checked").forEach(cb => {
            const proto = cb.value;
            const ports = [];
            document.querySelectorAll(`.cb-port-${proto}:checked`).forEach(pCb => ports.push(parseInt(pCb.value)));
            if (ports.length > 0) protocolPorts[proto] = ports;
        });

        const config = {
            locations: locations,
            protocolPorts: protocolPorts,
            internalDns: document.getElementById("internalDns").value,
            selectionMode: selectionMode,
            largeTls: document.getElementById("largeTls").checked,
            circumventCensorship: document.getElementById("circumventCensorship").checked,
            testPing: true,
            testDownload: true,
            testUpload: true,
            timeout: parseInt(document.getElementById("connTimeout").value) || 25,
            speedWait: parseInt(document.getElementById("speedWait").value) || 5
        };
        
        window.go.main.App.StartTest(config);
    });

    btnStop.addEventListener("click", () => {
        logMsg("Requesting stop...", "log-error");
        window.go.main.App.StopTest();
    });
});
