const API_BASE_URL = "http://localhost:8080/api/v1";

// Global state
let allChainsData = [];
let allTokensWithDetails = [];
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
    checkRouteButton.disabled = true; // Disable until loaded
    try {
        const chainsResponse = await fetch(`${API_BASE_URL}/chains`);
        if (!chainsResponse.ok) throw new Error(`Failed to fetch chains: ${chainsResponse.statusText}`);
        allChainsData = await chainsResponse.json();
        if (!Array.isArray(allChainsData) || allChainsData.length === 0) {
            throw new Error("No chains data received or data is invalid.");
        }
        populateChainSelect(fromChainSelect, allChainsData);

        const tokensResponse = await fetch(`${API_BASE_URL}/tokens`);
        if (!tokensResponse.ok) throw new Error(`Failed to fetch tokens: ${tokensResponse.statusText}`);
        const tokenListPayload = await tokensResponse.json();
        if (!tokenListPayload || !Array.isArray(tokenListPayload.supportedTokens)) {
            throw new Error("Tokens data format from API is incorrect.");
        }
        allTokensWithDetails = deriveTokenDetails(tokenListPayload.supportedTokens, allChainsData);

        // Set default for "From Chain" and trigger dependent dropdown updates
        if (allChainsData.length > 0) {
            fromChainSelect.value = allChainsData[0].id; // Default to first chain
        }
        await updateFromTokenOptions(); // Ensure this completes and sets a default fromToken
        await populateToTokenOptions();   // Ensure this completes and sets a default toToken

        // Set a default amount for easier testing
        amountInInput.value = "1000000"; // Default amount, e.g., 1 native token unit

        fromChainSelect.addEventListener('change', async () => {
            await updateFromTokenOptions();
            // No need to explicitly call populateToTokenOptions here, updateFromTokenOptions does it
        });
        fromTokenSelect.addEventListener('change', populateToTokenOptions); // Re-filter 'To Token'
        checkRouteButton.addEventListener('click', handleCheckBestRoute);
        
        showStatus("Ready to find routes!", "success", 2000);
        checkRouteButton.disabled = false; // Enable after successful load

    } catch (error) {
        console.error("Initialization Error:", error);
        showStatus(`Initialization Error: ${error.message}. Ensure backend is running and APIs are correct.`, "error");
        // Keep button disabled if init fails
    }
}

function deriveTokenDetails(tokenIds, chains) {
    const detailedTokens = [];
    const seenIds = new Set();

    if (!Array.isArray(tokenIds)) {
        console.error("deriveTokenDetails: tokenIds is not an array", tokenIds);
        return detailedTokens;
    }

    tokenIds.forEach(id => {
        if (seenIds.has(id) || typeof id !== 'string') return;

        let name = id.toUpperCase().replace(/^U/, '');
        let logo = `logos/${id.split('/')[0]}.svg`;
        let originChainId = null;

        const nativeChain = chains.find(c => c.nativeToken === id);
        if (nativeChain) {
            name = `${name} (${nativeChain.name.split(' ')[0]})`;
            originChainId = nativeChain.id;
            logo = `logos/${id}.svg`;
        } else if (id.startsWith("ibc/")) {
            const conceptualName = id.split('/')[1] || "UnknownIBC"; // e.g. BetaOnAlpha
            let baseName = conceptualName;
            if (conceptualName.includes("On")) {
                baseName = conceptualName.substring(0, conceptualName.indexOf("On")); // e.g. Beta
            }
            name = `${baseName.toUpperCase()} (IBC)`;
            const potentialNativeDenom = `u${baseName.toLowerCase()}`;
            if (chains.find(c => c.nativeToken === potentialNativeDenom)) {
                logo = `logos/${potentialNativeDenom}.svg`;
            } else {
                logo = "logos/ibc.svg";
            }
        } else {
             // Handle conceptual tokens like "tokena"
            const firstChar = id.charAt(0); // "t" from "tokena"
            const baseTokenLetter = id.substring(5); // "a" from "tokena"
            const potentialNativeToken = `u${baseTokenLetter}lpha`; // Assuming 'a' -> alpha, 'b' -> beta etc.

            if (firstChar === 't' && baseTokenLetter.length === 1) {
                let nativeMatch = null;
                if (baseTokenLetter === 'a') nativeMatch = chains.find(c => c.nativeToken === 'ualpha');
                else if (baseTokenLetter === 'b') nativeMatch = chains.find(c => c.nativeToken === 'ubeta');
                else if (baseTokenLetter === 'c') nativeMatch = chains.find(c => c.nativeToken === 'ugamma');

                if (nativeMatch) {
                    name = `${nativeMatch.nativeToken.replace(/^u/,'').toUpperCase()} (Conceptual)`;
                    logo = `logos/${nativeMatch.nativeToken}.svg`;
                }
            }
        }
        detailedTokens.push({ id, name, logo, originChainId });
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

async function updateFromTokenOptions() { // Make async if fetching involved, though not here
    const selectedChainId = fromChainSelect.value;
    fromTokenSelect.innerHTML = '<option value="">-- Select From Token --</option>';
    
    if (!selectedChainId) {
        await populateToTokenOptions();
        return;
    }

    const selectedChain = allChainsData.find(c => c.id === selectedChainId);
    if (selectedChain && Array.isArray(selectedChain.supportedTokens)) {
        selectedChain.supportedTokens.forEach(tokenId => {
            const tokenInfo = allTokensWithDetails.find(t => t.id === tokenId) || 
                              { id: tokenId, name: tokenId.toUpperCase().replace(/^U/, ''), logo: `logos/${tokenId}.svg` };
            const option = document.createElement('option');
            option.value = tokenInfo.id;
            option.textContent = tokenInfo.name;
            fromTokenSelect.appendChild(option);
        });
    }
    
    if (fromTokenSelect.options.length > 1) {
        fromTokenSelect.selectedIndex = 1; // Default to the first actual token
    } else if (fromTokenSelect.options.length === 1 && selectedChain && selectedChain.nativeToken) {
        // If supportedTokens was empty but chain has a native token, add it as a fallback
        const nativeTokenInfo = allTokensWithDetails.find(t => t.id === selectedChain.nativeToken) ||
                                { id: selectedChain.nativeToken, name: selectedChain.nativeToken.toUpperCase().replace(/^U/, ''), logo: `logos/${selectedChain.nativeToken}.svg`};
        const option = document.createElement('option');
        option.value = nativeTokenInfo.id;
        option.textContent = nativeTokenInfo.name;
        fromTokenSelect.appendChild(option);
        fromTokenSelect.selectedIndex = 1;
    }
    await populateToTokenOptions(); // Refresh "To Token" options
}

async function populateToTokenOptions() { // Make async if fetching involved
    const currentFromTokenId = fromTokenSelect.value;
    toTokenSelect.innerHTML = '<option value="">-- Select To Token --</option>';

    allTokensWithDetails.forEach(token => {
        if (token.id !== currentFromTokenId) {
            const option = document.createElement('option');
            option.value = token.id;
            option.textContent = token.name;
            toTokenSelect.appendChild(option);
        }
    });
    if (toTokenSelect.options.length > 1) {
        // Try to select a different token than fromToken, if possible
        if (currentFromTokenId === toTokenSelect.options[1].value && toTokenSelect.options.length > 2) {
            toTokenSelect.selectedIndex = 2;
        } else {
            toTokenSelect.selectedIndex = 1;
        }
    }
}


async function handleCheckBestRoute() {
    const fromChainID = fromChainSelect.value;
    const fromToken = fromTokenSelect.value;
    const toToken = toTokenSelect.value;
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
            } catch (e) { /* responseText might not be JSON */ errorMsg = `${errorMsg} - Response: ${responseText.substring(0,100)}...` }
            throw new Error(errorMsg);
        }
        currentBestRouteData = JSON.parse(responseText);

        if (!currentBestRouteData || !currentBestRouteData.bestRoute || !currentBestRouteData.bestRoute.steps || currentBestRouteData.bestRoute.steps.length === 0) {
            throw new Error("No viable route found. Try different tokens or increase amount.");
        }

        displayRouteVisualization(currentBestRouteData.bestRoute);
        displayRouteResults(currentBestRouteData); // This adds the "Execute Swap" button
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

    const { fromToken, toToken, amountIn } = currentBestRouteData;
    const fromChainID = fromChainSelect.value; // This is where the tokens originate

    const fromChainConfig = allChainsData.find(c => c.id === fromChainID);
    if (!fromChainConfig) {
        showStatus("Critical Error: Source chain configuration missing.", "error");
        return;
    }
    // The userAddress for the API is the key name the backend operator will use for this chain.
    const userAddressForApi = fromChainConfig.OperatorKeyName; 

    const payload = {
        fromToken,
        toToken,
        amountIn,
        fromChainID,
        userAddress: userAddressForApi, // This tells backend which key to use for signing on fromChainID
        // recipientAddress: "" // For MVP, backend can default this to userAddressForApi on target chain
    };

    showStatus("Executing swap...", "loading");
    const executeSwapButton = document.getElementById('executeSwapButton'); // Get it again, might be re-rendered
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
            resultHTML += `<p><strong>Est. Final Amount Out:</strong> ${swapResult.finalAmountOut}</p>`;
        }
        resultsDisplayDiv.innerHTML += resultHTML; // Append swap execution result
        showStatus(swapResult.message || "Swap submitted!", "success", 7000);

    } catch (error) {
        console.error("Error executing swap:", error);
        showStatus(`${error.message}`, "error", 10000);
        if (executeSwapButton) executeSwapButton.disabled = false; // Re-enable on error
    } finally {
        // checkRouteButton typically should be re-enabled unless a swap is in a state that prevents new routing
         checkRouteButton.disabled = false;
    }
}

// --- UI Display Functions --- (Mostly same as before, ensure robustness)
function displayRouteVisualization(bestRouteDetail) {
    routeVisualizationDiv.innerHTML = "";
    if (!bestRouteDetail || !bestRouteDetail.steps || bestRouteDetail.steps.length === 0) {
        routeVisualizationDiv.innerHTML = "<p style='color: var(--text-secondary);'>Route details will appear here.</p>";
        return;
    }

    let htmlPath = "";
    let currentChainForLogo = fromChainSelect.value;
    const initialFromTokenInfo = allTokensWithDetails.find(t => t.id === fromTokenSelect.value);

    htmlPath += createRouteStepHTML(currentChainForLogo, initialFromTokenInfo, true); // Mark as start

    bestRouteDetail.steps.forEach((step, index) => {
        const stepLower = step.toLowerCase();
        let actionLogo = "logos/ibc.svg"; 
        let nextChainIdForLogo = null;
        let nextTokenDisplayInfo = null;

        if (stepLower.includes("ibc transfer")) {
            const toMatch = step.match(/to ([\w-]+(?:net-1)?)/i);
            if (toMatch) nextChainIdForLogo = getChainIdByNameOrId(toMatch[1]);
            // After IBC, the token is conceptually still the one that was sent, just on a new chain
            nextTokenDisplayInfo = initialFromTokenInfo; 
        } else if (stepLower.includes("swap")) {
            const onMatch = step.match(/on ([\w-]+(?:net-1)?)/i);
            if (onMatch) nextChainIdForLogo = getChainIdByNameOrId(onMatch[1]);
            else nextChainIdForLogo = currentChainForLogo; 
            // After swap, the token becomes the overall target token
            nextTokenDisplayInfo = allTokensWithDetails.find(t => t.id === toTokenSelect.value);
        }

        if (nextChainIdForLogo) {
            htmlPath += `<img src="${actionLogo}" alt="Route Action" class="ibc-logo route-arrow">`;
            htmlPath += createRouteStepHTML(nextChainIdForLogo, nextTokenDisplayInfo, index === bestRouteDetail.steps.length -1 && !stepLower.includes("ibc transfer"));
            currentChainForLogo = nextChainIdForLogo;
        }
    });
    routeVisualizationDiv.innerHTML = htmlPath || "<p style='color: var(--text-secondary);'>Route details will appear here.</p>";
}


function getChainIdByNameOrId(nameOrId) {
    const nameOrIdLower = nameOrId.toLowerCase();
    const foundChain = allChainsData.find(c => 
        c.name.toLowerCase() === nameOrIdLower || 
        c.id.toLowerCase() === nameOrIdLower ||
        c.name.toLowerCase().startsWith(nameOrIdLower.split(' ')[0])
    );
    return foundChain ? foundChain.id : nameOrId;
}

function createRouteStepHTML(chainId, tokenInfoInput, isFinalToken = false) {
    const chainInfo = allChainsData.find(c => c.id === chainId) || { id: chainId, name: chainId.replace('-1', '').toUpperCase() + "Net" };
    
    let tokenInfo = tokenInfoInput;
    if (!tokenInfo && isFinalToken) { // If it's the final step and no specific tokenInfo, use the overall toToken
        tokenInfo = allTokensWithDetails.find(t => t.id === toTokenSelect.value);
    }
    if (!tokenInfo && chainInfo.nativeToken) { // Fallback to chain's native token
        tokenInfo = allTokensWithDetails.find(t => t.id === chainInfo.nativeToken);
    }
    if (!tokenInfo) { // Absolute fallback
        tokenInfo = { name: "Token on " + chainInfo.name, logo: `logos/${chainId}.svg` };
    }
    
    const logoSrc = tokenInfo.logo || `logos/${chainId}.svg`;
    const displayName = tokenInfo.name;
    
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
    // Ensure button exists before adding listener
    const swapButton = document.getElementById('executeSwapButton');
    if (swapButton) {
        swapButton.addEventListener('click', handleExecuteSwap);
    } else {
        console.error("Could not find executeSwapButton to attach listener.");
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

// Fallback for broken image links
document.addEventListener('error', function (event) {
    if (event.target.tagName.toLowerCase() === 'img' && event.target.classList.contains('logo')) {
        event.target.src = 'logos/unknown.svg';
        event.target.alt = 'Unknown Token/Chain';
    }
}, true);
