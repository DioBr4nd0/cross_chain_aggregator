const API_BASE_URL = "http://localhost:8080/api/v1";

// Global state
let allChainsData = [];
let allTokensWithDetails = []; // Will store [{id, name (descriptive), logo, originChainId (for native), onChainId (for IBC)}, ...]
let currentBestRouteData = null;

// DOM Elements
const fromChainSelect = document.getElementById('fromChain');
const fromTokenSelect = document.getElementById('fromToken');
const toTokenSelect = document.getElementById('toToken');
const amountInInput = document.getElementById('amountIn');
const checkRouteButton = document.getElementById('checkRouteButton');
const routeVisualizationDiv = document.getElementById('route-visualization');
const resultsDisplayDiv = document.getElementById('results-display');
const statusMessageDiv = document.getElementById('status-message');

document.addEventListener('DOMContentLoaded', initializeApp);

async function initializeApp() {
    showStatus("Initializing: Loading chain and token data...", "loading");
    checkRouteButton.disabled = true;
    try {
        const chainsResponse = await fetch(`${API_BASE_URL}/chains`);
        if (!chainsResponse.ok) throw new Error(`Failed to fetch chains: ${chainsResponse.statusText}`);
        allChainsData = await chainsResponse.json();
        if (!Array.isArray(allChainsData) || allChainsData.length === 0) {
            throw new Error("No chains data received or data is invalid.");
        }
        populateChainSelect(fromChainSelect, allChainsData);

        // Assuming /tokens returns { supportedTokens: ["id1", "id2", "ibc/denomOnChainX", "ibc/denomOnChainY"] }
        // These IDs are the actual denoms your backend and contracts use.
        const tokensResponse = await fetch(`${API_BASE_URL}/tokens`);
        if (!tokensResponse.ok) throw new Error(`Failed to fetch tokens: ${tokensResponse.statusText}`);
        const tokenListPayload = await tokensResponse.json();
        if (!tokenListPayload || !Array.isArray(tokenListPayload.supportedTokens)) {
            throw new Error("Tokens data format from API is incorrect.");
        }
        
        // deriveTokenDetails will now try to create very descriptive names
        allTokensWithDetails = deriveTokenDetails(tokenListPayload.supportedTokens, allChainsData);

        if (allChainsData.length > 0) {
            fromChainSelect.value = allChainsData[0].id;
        }
        await updateFromTokenOptions(); // Populates fromToken based on fromChain
        // populateToTokenOptions will be called by updateFromTokenOptions

        amountInInput.value = "1000000"; // Default amount

        fromChainSelect.addEventListener('change', updateFromTokenOptions);
        fromTokenSelect.addEventListener('change', populateToTokenOptions);
        checkRouteButton.addEventListener('click', handleCheckBestRoute);
        
        showStatus("Ready!", "success", 2000);
        checkRouteButton.disabled = false;

    } catch (error) {
        console.error("Initialization Error:", error);
        showStatus(`Initialization Error: ${error.message}. Ensure backend is running.`, "error");
    }
}

function deriveTokenDetails(tokenIds, chains) {
    const detailedTokens = [];
    const seenIds = new Set();

    if (!Array.isArray(tokenIds) || !Array.isArray(chains)) {
        console.error("deriveTokenDetails: Invalid input", tokenIds, chains);
        return detailedTokens;
    }

    tokenIds.forEach(id => {
        if (seenIds.has(id) || typeof id !== 'string') return;

        let name = id; // Default to ID if no better name found
        let logo = `logos/unknown.svg`; // Default logo
        let originChainId = null; // For native tokens
        let onChainId = null; // For IBC tokens, which chain they are currently "on"

        // Check if it's a native token of any configured chain
        const nativeChain = chains.find(c => c.nativeToken === id);
        if (nativeChain) {
            name = `${id.replace(/^u/, '').toUpperCase()} (Native on ${nativeChain.name})`;
            logo = `logos/${id}.svg`; // e.g., ualpha.svg
            originChainId = nativeChain.id;
            onChainId = nativeChain.id; // Native token is "on" its origin chain
        } else if (id.startsWith("ibc/")) {
            // Attempt to parse conceptual IBC denoms like "ibc/BetaOnAlpha"
            // id = "ibc/BetaOnAlpha" -> conceptual_base_denom="ubeta", on_chain_name="AlphaNet"
            const parts = id.split('/'); // ["ibc", "BetaOnAlpha"]
            if (parts.length === 2) {
                const conceptualName = parts[1]; // "BetaOnAlpha"
                const nameParts = conceptualName.match(/^([A-Za-z]+)On([A-Za-z]+)$/); // ["BetaOnAlpha", "Beta", "Alpha"]
                if (nameParts && nameParts.length === 3) {
                    const baseTokenName = nameParts[1]; // "Beta"
                    const onChainShortName = nameParts[2]; // "Alpha"

                    const originChainOfBaseToken = chains.find(c => c.name.startsWith(baseTokenName));
                    const currentChainTokenIsOn = chains.find(c => c.name.startsWith(onChainShortName));

                    if (originChainOfBaseToken && currentChainTokenIsOn) {
                        name = `${baseTokenName.toUpperCase()} (on ${currentChainTokenIsOn.name} via IBC from ${originChainOfBaseToken.name})`;
                        logo = `logos/${originChainOfBaseToken.nativeToken}.svg`; // Logo of the original token
                        originChainId = originChainOfBaseToken.id;
                        onChainId = currentChainTokenIsOn.id;
                    } else {
                        name = `${id} (IBC)`; // Fallback if conceptual parsing fails
                        logo = `logos/ibc.svg`;
                    }
                } else {
                    name = `${id} (IBC)`; // Fallback for non-standard conceptual names
                    logo = `logos/ibc.svg`;
                }
            } else {
                 name = `${id} (IBC Hash)`; // Actual IBC hash
                 logo = `logos/ibc.svg`;
            }
        } else {
             // Handle other supported tokens if they are not native or standard "ibc/Concept"
             // This might be for your "tokena", "tokenb" if they are still used.
             // Try to find if this token is listed as a "supportedToken" on a specific chain
             // and infer its nature. This part is tricky without more context on "tokena".
             let foundOnChain = null;
             for (const chain of chains) {
                 if(chain.supportedTokens.includes(id)){
                     foundOnChain = chain;
                     break;
                 }
             }
             if(foundOnChain){
                name = `${id.toUpperCase()} (on ${foundOnChain.name})`;
                // Try to guess logo based on convention or map
                const potentialNative = `u${id.replace('token','').toLowerCase()}`; // e.g. ualpha from tokena
                const logoChain = chains.find(c => c.nativeToken === potentialNative);
                if(logoChain) logo = `logos/${logoChain.nativeToken}.svg`;
             } else {
                name = id.toUpperCase(); // Generic fallback
             }
        }

        detailedTokens.push({ id, name, logo, originChainId, onChainId });
        seenIds.add(id);
    });
    return detailedTokens;
}


function populateChainSelect(selectElement, chains) {
    selectElement.innerHTML = '<option value="">-- Select From Chain --</option>';
    if (!Array.isArray(chains) || chains.length === 0) {
        selectElement.innerHTML = '<option value="">No chains available</option>';
        return;
    }
    chains.forEach(chain => {
        const option = document.createElement('option');
        option.value = chain.id;
        option.textContent = chain.name;
        selectElement.appendChild(option);
    });
}

async function updateFromTokenOptions() {
    const selectedChainId = fromChainSelect.value;
    fromTokenSelect.innerHTML = '<option value="">-- Select From Token --</option>';
    
    if (!selectedChainId) {
        await populateToTokenOptions(); // Update "To Token" with all tokens if no "From Chain"
        return;
    }

    const selectedChain = allChainsData.find(c => c.id === selectedChainId);
    if (selectedChain && Array.isArray(selectedChain.supportedTokens)) {
        // Iterate through allTokensWithDetails and filter those whose ID is in selectedChain.supportedTokens
        // OR whose onChainId matches the selectedChainId (for IBC tokens residing on this chain)
        // OR whose originChainId matches (for native tokens of this chain)
        allTokensWithDetails.forEach(tokenInfo => {
            if (selectedChain.supportedTokens.includes(tokenInfo.id) || 
                tokenInfo.onChainId === selectedChainId || // If it's an IBC token residing on this chain
                tokenInfo.originChainId === selectedChainId // If it's native to this chain
                ) {
                const option = document.createElement('option');
                option.value = tokenInfo.id;
                option.textContent = tokenInfo.name; // Use the descriptive name
                fromTokenSelect.appendChild(option);
            }
        });
    }
    
    if (fromTokenSelect.options.length > 1) {
        fromTokenSelect.selectedIndex = 1;
    }
    await populateToTokenOptions();
}

async function populateToTokenOptions() {
    const currentFromTokenId = fromTokenSelect.value;
    const currentFromChainId = fromChainSelect.value;
    toTokenSelect.innerHTML = '<option value="">-- Select To Token --</option>';

    allTokensWithDetails.forEach(token => {
        // Basic filter: don't swap to the exact same token ID.
        // More advanced: don't swap to the same token if it's on the same chain.
        if (token.id !== currentFromTokenId) {
            // Additionally, if fromToken is native, don't show its IBC representation on the same chain as a "toToken"
            const fromTokenDetails = allTokensWithDetails.find(t => t.id === currentFromTokenId);
            if (fromTokenDetails && fromTokenDetails.originChainId === currentFromChainId && // fromToken is native
                token.originChainId === fromTokenDetails.originChainId && // toToken has same origin
                token.onChainId === currentFromChainId && // toToken is on the same current chain
                token.id !== fromTokenDetails.id // and it's an IBC representation
            ) {
                // Skip e.g. ualpha (Native on AlphaNet) -> ALPHA (on AlphaNet via IBC from AlphaNet)
                // This case should ideally not exist in allTokensWithDetails if derivation is perfect.
            } else {
                const option = document.createElement('option');
                option.value = token.id;
                option.textContent = token.name; // Use the descriptive name
                toTokenSelect.appendChild(option);
            }
        }
    });
    if (toTokenSelect.options.length > 1) {
        // Try to pick a default different from fromToken
        if (toTokenSelect.options[1].value === currentFromTokenId && toTokenSelect.options.length > 2) {
            toTokenSelect.selectedIndex = 2;
        } else {
            toTokenSelect.selectedIndex = 1;
        }
    }
}

// --- API Call Handlers (handleCheckBestRoute, handleExecuteSwap) remain largely the same ---
// Ensure they use fromTokenSelect.value, toTokenSelect.value which are the IDs.
// The user sees the descriptive names, but the value sent to backend is the actual denom.

async function handleCheckBestRoute() {
    const fromChainID = fromChainSelect.value;
    const fromToken = fromTokenSelect.value; // This is the ID, e.g., "ualpha" or "ibc/BetaOnAlpha"
    const toToken = toTokenSelect.value;     // This is the ID
    const amountIn = amountInInput.value;

    if (!fromChainID) { showStatus("Please select a 'From Chain'.", "error", 4000); return; }
    if (!fromToken) { showStatus("Please select a 'From Token'.", "error", 4000); return; }
    if (!toToken) { showStatus("Please select a 'To Token'.", "error", 4000); return; }
    if (!amountIn || parseFloat(amountIn) <= 0) {
        showStatus("Please enter a valid 'Amount' (must be > 0).", "error", 4000);
        return;
    }

    showStatus("Finding best route...", "loading");
    resultsDisplayDiv.innerHTML = "";
    routeVisualizationDiv.innerHTML = "";
    checkRouteButton.disabled = true;
    currentBestRouteData = null;

    try {
        const queryParams = new URLSearchParams({ fromToken, toToken, amountIn, fromChainID });
        const response = await fetch(`${API_BASE_URL}/best-route?${queryParams.toString()}`);
        
        const responseText = await response.text();
        if (!response.ok) {
            let errorMsg = `Route Error (${response.status}): ${response.statusText}`;
            try {
                const errorData = JSON.parse(responseText);
                errorMsg = errorData.error || errorData.message || errorMsg;
            } catch (e) { errorMsg = `${errorMsg} - Response: ${responseText.substring(0,100)}...` }
            throw new Error(errorMsg);
        }
        currentBestRouteData = JSON.parse(responseText);

        if (!currentBestRouteData || !currentBestRouteData.bestRoute || !currentBestRouteData.bestRoute.steps || currentBestRouteData.bestRoute.steps.length === 0) {
            throw new Error("No viable route found. Try different tokens or increase amount.");
        }

        displayRouteVisualization(currentBestRouteData.bestRoute);
        displayRouteResults(currentBestRouteData);
        showStatus("Best route found!", "success", 3000);

    } catch (error) {
        console.error("Error fetching best route:", error);
        showStatus(`${error.message}`, "error", 7000);
    } finally {
        checkRouteButton.disabled = false;
    }
}

async function handleExecuteSwap() {
    if (!currentBestRouteData || !currentBestRouteData.bestRoute) {
        showStatus("No route selected to execute. Please find a route first.", "error", 4000);
        return;
    }

    const { fromToken, toToken, amountIn } = currentBestRouteData; // These are the IDs
    const fromChainID = fromChainSelect.value;

    const fromChainConfig = allChainsData.find(c => c.id === fromChainID);
    if (!fromChainConfig) {
        showStatus("Critical Error: Source chain configuration missing.", "error");
        return;
    }
    const userAddressForApi = fromChainConfig.OperatorKeyName; 

    const payload = {
        fromToken, // ID
        toToken,   // ID
        amountIn,
        fromChainID,
        userAddress: userAddressForApi,
    };

    showStatus("Executing swap...", "loading");
    const executeSwapButton = document.getElementById('executeSwapButton');
    if (executeSwapButton) executeSwapButton.disabled = true;
    checkRouteButton.disabled = true;

    try {
        const response = await fetch(`${API_BASE_URL}/swap`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const responseText = await response.text();
        const swapResult = JSON.parse(responseText);

        if (!response.ok) {
             throw new Error(swapResult.error || swapResult.message || `Swap Error (${response.status}): ${responseText.substring(0,100)}...`);
        }
        
        let resultHTML = `<hr style="margin: 15px 0; border-color: var(--border-color);">
                          <h4>Swap Request Submitted:</h4>
                          <p><strong>Status:</strong> ${swapResult.status || 'N/A'}</p>
                          <p><strong>Message:</strong> ${swapResult.message || 'Processing...'}</p>`;
        if (swapResult.ibcTxHash) {
            resultHTML += `<p><strong>IBC Tx Hash:</strong> <span style="word-break:break-all;">${swapResult.ibcTxHash}</span></p>`;
        }
        if (swapResult.swapTxHash) {
            resultHTML += `<p><strong>DEX Swap Tx Hash:</strong> <span style="word-break:break-all;">${swapResult.swapTxHash}</span></p>`;
        }
        if (swapResult.finalAmountOut && !swapResult.finalAmountOut.toLowerCase().includes("unknown")) {
            resultHTML += `<p><strong>Actual Final Amount Out:</strong> ${swapResult.finalAmountOut}</p>`;
        }
        // Instead of +=, let's update a specific part of resultsDisplayDiv if it exists, or append carefully
        const existingContent = resultsDisplayDiv.querySelector('h4')?.parentElement?.innerHTML || "";
        if (existingContent.includes("Route Details")) {
            resultsDisplayDiv.innerHTML = existingContent + resultHTML; // Append to route details
        } else {
            resultsDisplayDiv.innerHTML = resultHTML; // Overwrite if no route details were there
        }

        showStatus(swapResult.message || "Swap submitted successfully!", "success", 7000);

    } catch (error) {
        console.error("Error executing swap:", error);
        showStatus(`${error.message}`, "error", 10000);
    } finally {
        if (executeSwapButton) executeSwapButton.disabled = false;
        checkRouteButton.disabled = false;
    }
}


// --- UI Display Functions (displayRouteVisualization, createRouteStepHTML, displayRouteResults) ---
// These need to use the descriptive names from allTokensWithDetails for display purposes,
// but the underlying logic for API calls must use the token IDs.

function displayRouteVisualization(bestRouteDetail) {
    routeVisualizationDiv.innerHTML = "";
    if (!bestRouteDetail || !bestRouteDetail.steps || bestRouteDetail.steps.length === 0) {
        routeVisualizationDiv.innerHTML = "<p style='color: var(--text-secondary);'>Route details will appear here.</p>";
        return;
    }

    let htmlPath = "";
    // The fromTokenSelect.value is the actual ID (e.g. ualpha) of the starting token
    const initialFromTokenDetails = allTokensWithDetails.find(t => t.id === fromTokenSelect.value);
    // The fromChainSelect.value is the ID of the starting chain (e.g. alphanet-1)
    let currentChainContextId = fromChainSelect.value; 

    htmlPath += createRouteStepHTML(currentChainContextId, initialFromTokenDetails);

    bestRouteDetail.steps.forEach((step, index) => {
        const stepLower = step.toLowerCase();
        let actionLogo = "logos/ibc.svg"; // Default
        let nextChainDisplayId = null;
        let tokenForNextStepDisplay = null; // This is what the token becomes *after* the step

        if (stepLower.includes("ibc transfer")) {
            // Step: "IBC Transfer ualpha from AlphaNet to BetaNet (becomes ibc/AlphaOnBeta)"
            const toChainMatch = step.match(/to ([\w-]+(?:net-1)?)/i); // e.g., "BetaNet" or "betanet-1"
            const becomesMatch = step.match(/\(becomes (ibc\/[\w\/-]+)\)/i); // e.g., "ibc/AlphaOnBeta"
            
            if (toChainMatch) nextChainDisplayId = getChainIdByNameOrId(toChainMatch[1]);
            if (becomesMatch) {
                tokenForNextStepDisplay = allTokensWithDetails.find(t => t.id === becomesMatch[1]);
            } else { // If "becomes" part is missing, assume original token on new chain
                tokenForNextStepDisplay = initialFromTokenDetails; 
            }
            
        } else if (stepLower.includes("swap")) {
            // Step: "Swap ibc/AlphaOnBeta for ubeta on BetaNet DEX"
            const onChainMatch = step.match(/on ([\w-]+(?:net-1)?)/i);
            const forTokenMatch = step.match(/for ([\w\/]+) on/i); // e.g., "ubeta"

            if (onChainMatch) nextChainDisplayId = getChainIdByNameOrId(onChainMatch[1]);
            else nextChainDisplayId = currentChainContextId; // Swap happens on the current chain

            if (forTokenMatch) {
                tokenForNextStepDisplay = allTokensWithDetails.find(t => t.id === forTokenMatch[1]);
            }
            actionLogo = "logos/swap_action.svg"; // Or reuse ibc.svg
            if(!document.querySelector(`img[src$="${actionLogo}"]`)) actionLogo = "logos/ibc.svg";
        }

        if (nextChainDisplayId) {
            htmlPath += `<img src="${actionLogo}" alt="Action" class="ibc-logo route-arrow">`;
            htmlPath += createRouteStepHTML(nextChainDisplayId, tokenForNextStepDisplay);
            currentChainContextId = nextChainDisplayId; // Update context for next step
        }
    });
    routeVisualizationDiv.innerHTML = htmlPath || "<p style='color: var(--text-secondary);'>Route details will appear here.</p>";
}

function getChainIdByNameOrId(nameOrId) {
    const nameOrIdLower = nameOrId.toLowerCase().replace("net", "net-1"); // Normalize "AlphaNet" to "alphanet-1"
    const foundChain = allChainsData.find(c => 
        c.id.toLowerCase() === nameOrIdLower ||
        c.name.toLowerCase() === nameOrIdLower ||
        c.name.toLowerCase().startsWith(nameOrIdLower.split(' ')[0])
    );
    return foundChain ? foundChain.id : nameOrId; // Fallback
}

function createRouteStepHTML(chainId, tokenInfo) {
    const chainInfo = allChainsData.find(c => c.id === chainId) || { id: chainId, name: "Unknown Chain" };
    
    // Ensure tokenInfo is an object, provide defaults if not
    const displayToken = tokenInfo && typeof tokenInfo === 'object' ? tokenInfo : 
                         { name: (tokenInfo || "Token") , logo: `logos/${(tokenInfo || 'unknown')}.svg` };

    const logoSrc = displayToken.logo || `logos/unknown.svg`;
    const displayName = displayToken.name;
    
    return `
        <div class="route-step" title="${chainInfo.name} - ${displayName}">
            <img src="${logoSrc}" alt="${displayName} logo" class="logo" onerror="this.src='logos/unknown.svg'; this.alt='Unknown Token';">
            <span class="chain-name">${displayName}</span>
        </div>
    `;
}


function displayRouteResults(routeData) {
    const { bestRoute, fromToken, toToken } = routeData;
    if (!bestRoute) {
        resultsDisplayDiv.innerHTML = "<p>No valid route details to display.</p>";
        return;
    }

    let stepsHtml = "<ul style='list-style-type: decimal; padding-left: 20px;'>";
    bestRoute.steps.forEach(step => stepsHtml += `<li>${step}</li>`);
    stepsHtml += "</ul>";

    const fromTokenDetails = allTokensWithDetails.find(t => t.id === fromToken) || { name: fromToken, id: fromToken };
    const toTokenDetails = allTokensWithDetails.find(t => t.id === toToken) || { name: toToken, id: toToken };
    const swappingChainDetails = allChainsData.find(c => c.id === bestRoute.chainIDSwappingOn) || { name: bestRoute.chainIDSwappingOn };

    resultsDisplayDiv.innerHTML = `
        <h4>Route Details:</h4>
        <p><strong>Swap On:</strong> ${swappingChainDetails.name}</p>
        <p><strong>Rate:</strong> 1 ${fromTokenDetails.name} ≈ <strong>${bestRoute.rate}</strong> ${toTokenDetails.name}</p>
        <p><strong>Est. Output:</strong> <strong>${bestRoute.amountOut}</strong> ${toTokenDetails.name}</p>
        <p><strong>Requires IBC:</strong> ${bestRoute.needsIBC ? 'Yes' : 'No'}</p>
        <p><strong>Steps:</strong></p>
        ${stepsHtml}
        <button id="executeSwapButton">Execute This Swap</button> 
    `;
    const swapButton = document.getElementById('executeSwapButton');
    if (swapButton) {
        swapButton.addEventListener('click', handleExecuteSwap);
    }
}

function showStatus(message, type = "info", duration = 0) {
    statusMessageDiv.textContent = message;
    statusMessageDiv.className = 'status-message ' + type;
    statusMessageDiv.style.display = 'block';

    if (statusMessageDiv.timeoutId) clearTimeout(statusMessageDiv.timeoutId);

    if (duration > 0) {
        statusMessageDiv.timeoutId = setTimeout(() => {
            statusMessageDiv.style.display = 'none';
            statusMessageDiv.className = 'status-message';
        }, duration);
    }
}

document.addEventListener('error', function (event) {
    if (event.target.tagName.toLowerCase() === 'img' && event.target.classList.contains('logo')) {
        event.target.src = 'logos/unknown.svg';
        event.target.alt = 'Unknown Token/Chain';
    }
}, true);
