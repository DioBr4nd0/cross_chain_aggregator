const API_BASE_URL = "http://localhost:8080/api/v1"; // Your backend URL

// Mock data for initial population and testing without backend
const mockChains = [
    { id: "alphanet-1", name: "AlphaNet (Local)", nativeToken: "ualpha", supportedTokens: ["ualpha", "tokenb", "tokenc"], feeDenom: "ualpha" },
    { id: "betanet-1", name: "BetaNet (Local)", nativeToken: "ubeta", supportedTokens: ["ubeta", "tokena", "tokenc"], feeDenom: "ubeta" },
    { id: "gammanet-1", name: "GammaNet (Local)", nativeToken: "ugamma", supportedTokens: ["ugamma", "tokena", "tokenb"], feeDenom: "ugamma" },
];

const mockTokens = [ // All unique tokens
    { id: "ualpha", name: "ALPHA", logo: "logos/ualpha.svg", chainId: "alphanet-1" },
    { id: "ubeta", name: "BETA", logo: "logos/ubeta.svg", chainId: "betanet-1" },
    { id: "ugamma", name: "GAMMA", logo: "logos/ugamma.svg", chainId: "gammanet-1" },
    { id: "tokena", name: "Token A (IBC'd ALPHA)", logo: "logos/ualpha.svg" }, // Conceptual IBC token
    { id: "tokenb", name: "Token B (IBC'd BETA)", logo: "logos/ubeta.svg" },   // Conceptual IBC token
    { id: "tokenc", name: "Token C (IBC'd GAMMA)", logo: "logos/ugamma.svg" }, // Conceptual IBC token
];

// Global state (minimal)
let currentBestRoute = null;
let allChainsData = [];
let allTokensData = [];


// DOM Elements
const fromChainSelect = document.getElementById('fromChain');
const fromTokenSelect = document.getElementById('fromToken');
const toTokenSelect = document.getElementById('toToken');
const amountInInput = document.getElementById('amountIn');
const checkRateButton = document.getElementById('checkRateButton');
const routeVisualizationDiv = document.getElementById('route-visualization');
const resultsDisplayDiv = document.getElementById('results-display');
const statusMessageDiv = document.getElementById('status-message');

// --- Initialization ---
document.addEventListener('DOMContentLoaded', async () => {
    showStatus("Loading chain and token data...", "loading");
    try {
        // Replace with actual API calls when backend is ready
        // const chainsResponse = await fetch(`${API_BASE_URL}/chains`);
        // allChainsData = await chainsResponse.json();
        // const tokensResponse = await fetch(`${API_BASE_URL}/tokens`); // Assuming this gives all unique tokens with names/logos
        // allTokensData = (await tokensResponse.json()).supportedTokens.map(t => ({id: t, name: t.toUpperCase()})); // Adapt based on actual API

        allChainsData = mockChains; // Using mock for now
        allTokensData = mockTokens; // Using mock for now

        populateChainSelect(fromChainSelect, allChainsData);
        updateFromTokenOptions(); // Initial population based on first chain
        populateToTokenOptions(allTokensData); // Populate with all conceptual tokens
        
        fromChainSelect.addEventListener('change', updateFromTokenOptions);
        checkRateButton.addEventListener('click', handleCheckRate);
        showStatus("Ready to find routes!", "success", 2000);

    } catch (error) {
        console.error("Initialization Error:", error);
        showStatus("Error loading initial data: " + error.message, "error");
    }
});

function populateChainSelect(selectElement, chains) {
    selectElement.innerHTML = ""; // Clear existing
    chains.forEach(chain => {
        const option = document.createElement('option');
        option.value = chain.id;
        option.textContent = chain.name;
        selectElement.appendChild(option);
    });
}

function updateFromTokenOptions() {
    const selectedChainId = fromChainSelect.value;
    const selectedChain = allChainsData.find(c => c.id === selectedChainId);
    
    fromTokenSelect.innerHTML = ""; // Clear existing
    if (selectedChain) {
        // For MVP, assume supportedTokens in chain config are the actual denoms to use for that chain's DEX
        selectedChain.supportedTokens.forEach(tokenId => {
            const tokenInfo = allTokensData.find(t => t.id === tokenId) || { id: tokenId, name: tokenId.toUpperCase().replace('U', '') };
            const option = document.createElement('option');
            option.value = tokenInfo.id;
            option.textContent = tokenInfo.name;
            fromTokenSelect.appendChild(option);
        });
    }
    // Also update "To Token" options to exclude the currently selected "From Token"
    populateToTokenOptions(allTokensData);
}

function populateToTokenOptions(tokens) {
    const currentFromToken = fromTokenSelect.value;
    toTokenSelect.innerHTML = ""; // Clear existing
    tokens.forEach(token => {
        if (token.id !== currentFromToken) { // Don't allow swapping to the same token
            const option = document.createElement('option');
            option.value = token.id;
            option.textContent = token.name;
            toTokenSelect.appendChild(option);
        }
    });
}


// --- Event Handlers & API Calls ---
async function handleCheckRate() {
    const fromChainID = fromChainSelect.value;
    const fromToken = fromTokenSelect.value;
    const toToken = toTokenSelect.value;
    const amountIn = amountInInput.value;

    if (!fromChainID || !fromToken || !toToken || !amountIn || parseFloat(amountIn) <= 0) {
        showStatus("Please fill in all fields with valid values.", "error");
        return;
    }

    showStatus("Checking best rate & route...", "loading");
    resultsDisplayDiv.innerHTML = "";
    routeVisualizationDiv.innerHTML = "";
    checkRateButton.disabled = true;

    try {
        // MOCK API CALL FOR NOW - Replace with actual fetch to your backend
        // const response = await fetch(`${API_BASE_URL}/best-route?fromToken=${fromToken}&toToken=${toToken}&amountIn=${amountIn}&fromChainID=${fromChainID}`);
        // if (!response.ok) {
        //     const errorData = await response.json();
        //     throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
        // }
        // currentBestRoute = await response.json();
        
        // --- Using Mock Route Data ---
        currentBestRoute = generateMockRoute(fromChainID, fromToken, toToken, amountIn);
        if (!currentBestRoute || !currentBestRoute.bestRoute) {
             throw new Error("No route found for this pair (mock data).");
        }
        // --- End Mock Route Data ---


        displayRouteVisualization(currentBestRoute.bestRoute);
        displayRouteResults(currentBestRoute);
        showStatus("Best route found!", "success", 2000);

    } catch (error) {
        console.error("Error fetching best route:", error);
        showStatus("Error: " + error.message, "error");
        currentBestRoute = null;
    } finally {
        checkRateButton.disabled = false;
    }
}

function generateMockRoute(fromChainID, fromTokenID, toTokenID, amountIn) {
    // This function generates plausible mock routes for demonstration
    const fromChain = allChainsData.find(c => c.id === fromChainID);
    const toTokenInfo = allTokensData.find(t => t.id === toTokenID);

    if (!fromChain || !toTokenInfo) return null;

    let bestRouteDetail;
    const fromTokenInfo = allTokensData.find(t => t.id === fromTokenID);

    // Scenario 1: Direct swap on source chain (if ToToken is conceptually on FromChain)
    if (fromChain.supportedTokens.includes(toTokenID)) {
        bestRouteDetail = {
            chainIDSwappingOn: fromChain.id,
            rate: (Math.random() * 0.2 + 0.9).toFixed(4), // Random rate around 0.9-1.1
            amountOut: (parseFloat(amountIn) * (Math.random() * 0.2 + 0.9)).toFixed(0),
            steps: [`Swap ${fromTokenInfo.name} for ${toTokenInfo.name} on ${fromChain.name} DEX`],
            needsIBC: false,
        };
    } else { // Scenario 2: IBC transfer needed
        // Find an intermediate chain that supports both (or the target token natively)
        const intermediateChain = allChainsData.find(c => c.id !== fromChainID && c.supportedTokens.includes(toTokenID)) || 
                                  allChainsData.find(c => c.id !== fromChainID && c.nativeToken === toTokenID) || // Target chain for native
                                  allChainsData.find(c => c.id !== fromChainID); // Fallback to any other chain

        if (!intermediateChain) return null; // Should not happen with 3 chains

        bestRouteDetail = {
            chainIDSwappingOn: intermediateChain.id,
            rate: (Math.random() * 0.3 + 0.85).toFixed(4), // Slightly worse rate due to IBC conceptually
            amountOut: (parseFloat(amountIn) * (Math.random() * 0.3 + 0.85) * 0.98).toFixed(0), // *0.98 for "IBC fee"
            steps: [
                `IBC Transfer ${fromTokenInfo.name} from ${fromChain.name} to ${intermediateChain.name}`,
                `Swap ${fromTokenInfo.name} (as IBC'd) for ${toTokenInfo.name} on ${intermediateChain.name} DEX`
            ],
            needsIBC: true,
        };
         // If intermediate is not the final token's native chain, add another IBC step (conceptual)
        if (intermediateChain.nativeToken !== toTokenInfo.id && toTokenInfo.chainId && intermediateChain.id !== toTokenInfo.chainId) {
            const finalChain = allChainsData.find(c => c.id === toTokenInfo.chainId) || intermediateChain; // target chain for the toToken
             bestRouteDetail.steps = [
                `IBC Transfer ${fromTokenInfo.name} from ${fromChain.name} to ${intermediateChain.name}`,
                `Swap ${fromTokenInfo.name} (as IBC'd) for ${toTokenInfo.name} (conceptual) on ${intermediateChain.name} DEX`,
                `IBC Transfer ${toTokenInfo.name} from ${intermediateChain.name} to ${finalChain.name}`
            ];
            bestRouteDetail.chainIDSwappingOn = finalChain.id; // Final swap happens on final chain
        }

    }
    
    return {
        fromToken: fromTokenID,
        toToken: toTokenID,
        amountIn: amountIn,
        bestRoute: bestRouteDetail,
        otherRoutes: [] // Mock: not generating other routes for simplicity
    };
}


function displayRouteVisualization(bestRouteDetail) {
    routeVisualizationDiv.innerHTML = ""; // Clear previous
    if (!bestRouteDetail || !bestRouteDetail.steps || bestRouteDetail.steps.length === 0) {
        routeVisualizationDiv.innerHTML = "<p>No route steps to display.</p>";
        return;
    }

    // Logic to parse steps and determine chain sequence for visualization
    // This is a simplified parser for the mock step descriptions.
    // A robust solution would get structured chain path data from the backend.
    const chainSequence = new Set();
    let lastChain = fromChainSelect.value; // Start with the source chain of the query
    chainSequence.add(lastChain);

    bestRouteDetail.steps.forEach(step => {
        const ibcMatch = step.match(/IBC Transfer.*?from (.*?) to (.*?)$/i);
        const swapMatch = step.match(/Swap.*?on (.*?) DEX/i);
        if (ibcMatch) {
            // chainSequence.add(ibcMatch[1].replace('Net','net-1')); // Extract chain name, map to ID
            chainSequence.add(getChainIdFromName(ibcMatch[2])); // Add destination chain
            lastChain = getChainIdFromName(ibcMatch[2]);
        } else if (swapMatch) {
            // chainSequence.add(swapMatch[1].replace('Net','net-1'));
            chainSequence.add(getChainIdFromName(swapMatch[1]));
            lastChain = getChainIdFromName(swapMatch[1]);
        }
    });
     // Ensure the final swapping chain is included if not caught by parsing steps
    if (bestRouteDetail.chainIDSwappingOn && !Array.from(chainSequence).includes(bestRouteDetail.chainIDSwappingOn)) {
        // Attempt to insert it logically or just append
        if (lastChain !== bestRouteDetail.chainIDSwappingOn) { // Avoid duplicate if lastChain is already the swapping chain
            chainSequence.add(bestRouteDetail.chainIDSwappingOn);
        }
    }


    const uniqueChainPath = Array.from(chainSequence);

    let html = "";
    uniqueChainPath.forEach((chainId, index) => {
        const chainInfo = allChainsData.find(c => c.id === chainId) || { id: chainId, name: chainId };
        const tokenInfo = allTokensData.find(t => t.id === (index === 0 ? fromTokenSelect.value : (index === uniqueChainPath.length - 1 ? toTokenSelect.value : 'transfer'))); // Conceptual token for step
        const logoSrc = tokenInfo && tokenInfo.logo ? tokenInfo.logo : `logos/${chainId}.svg`; // Fallback to chain logo

        html += `
            <div class="route-step">
                <img src="${logoSrc}" alt="${chainInfo.name} logo" class="logo">
                <span class="chain-name">${chainInfo.name}</span>
            </div>
        `;
        if (index < uniqueChainPath.length - 1) {
            html += `<img src="logos/ibc.svg" alt="IBC Transfer" class="ibc-logo route-arrow">`;
        }
    });
    routeVisualizationDiv.innerHTML = html || "<p>Route will appear here.</p>";
}

// Helper to map chain name from steps to chain ID (simplified)
function getChainIdFromName(chainName) {
    const foundChain = allChainsData.find(c => c.name.toLowerCase().startsWith(chainName.toLowerCase().split(' ')[0]));
    return foundChain ? foundChain.id : chainName.toLowerCase().replace('net', 'net-1'); // Fallback guess
}


function displayRouteResults(routeData) {
    if (!routeData || !routeData.bestRoute) {
        resultsDisplayDiv.innerHTML = "<p>Could not retrieve route details.</p>";
        return;
    }
    const { bestRoute } = routeData;
    let stepsHtml = "<ul>";
    bestRoute.steps.forEach(step => stepsHtml += `<li>${step}</li>`);
    stepsHtml += "</ul>";

    resultsDisplayDiv.innerHTML = `
        <h4>Best Route Details:</h4>
        <p><strong>Swapping On:</strong> ${allChainsData.find(c => c.id === bestRoute.chainIDSwappingOn)?.name || bestRoute.chainIDSwappingOn}</p>
        <p><strong>Rate:</strong> 1 ${allTokensData.find(t=>t.id === routeData.fromToken)?.name || routeData.fromToken} ≈ ${bestRoute.rate} ${allTokensData.find(t=>t.id === routeData.toToken)?.name || routeData.toToken}</p>
        <p><strong>Estimated Output:</strong> ${bestRoute.amountOut} ${allTokensData.find(t=>t.id === routeData.toToken)?.name || routeData.toToken}</p>
        <p><strong>Needs IBC Transfer:</strong> ${bestRoute.needsIBC ? 'Yes' : 'No'}</p>
        <p><strong>Steps:</strong></p>
        ${stepsHtml}
        <button id="executeSwapButton">Execute Swap</button>
    `;

    document.getElementById('executeSwapButton').addEventListener('click', handleExecuteSwap);
}

async function handleExecuteSwap() {
    if (!currentBestRoute) {
        showStatus("No route selected to execute.", "error");
        return;
    }

    showStatus("Submitting swap transaction...", "loading");
    document.getElementById('executeSwapButton').disabled = true;

    const payload = {
        fromToken: currentBestRoute.fromToken,
        toToken: currentBestRoute.toToken,
        amountIn: currentBestRoute.amountIn,
        fromChainID: fromChainSelect.value, // The original source chain
        userAddress: "wasm1dummyUserAddressBackendOp" // MVP: Hardcode your backend operator's address for the source chain
        // recipientAddress: "wasm1..." // Optional: if different from userAddress on final chain
    };

    try {
        // MOCK API CALL FOR NOW - Replace with actual fetch
        // const response = await fetch(`${API_BASE_URL}/swap`, {
        //     method: 'POST',
        //     headers: { 'Content-Type': 'application/json' },
        //     body: JSON.stringify(payload)
        // });
        // if (!response.ok) {
        //     const errorData = await response.json();
        //     throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
        // }
        // const swapResult = await response.json();

        // --- Using Mock Swap Result ---
        await new Promise(resolve => setTimeout(resolve, 1500)); // Simulate network delay
        const swapResult = {
            status: "completed_mock",
            message: "Swap submitted successfully (Mocked).",
            ibcTxHash: currentBestRoute.bestRoute.needsIBC ? "mockIbcTxHash" + Date.now() : "",
            swapTxHash: "mockSwapTxHash" + Date.now(),
            finalAmountOut: currentBestRoute.bestRoute.amountOut
        };
        // --- End Mock Swap Result ---

        resultsDisplayDiv.innerHTML += `
            <h4>Swap Result:</h4>
            <p><strong>Status:</strong> ${swapResult.status}</p>
            <p><strong>Message:</strong> ${swapResult.message}</p>
            ${swapResult.ibcTxHash ? `<p><strong>IBC Tx Hash:</strong> ${swapResult.ibcTxHash}</p>` : ''}
            <p><strong>DEX Swap Tx Hash:</strong> ${swapResult.swapTxHash}</p>
            <p><strong>Final Amount Out (est.):</strong> ${swapResult.finalAmountOut || 'N/A'}</p>
        `;
        showStatus("Swap processed (mocked)!", "success", 3000);

    } catch (error) {
        console.error("Error executing swap:", error);
        showStatus("Swap Error: " + error.message, "error");
    } finally {
        // Re-enable button or reset state if needed, but for MVP, one-shot is fine.
    }
}


function showStatus(message, type = "info", duration = 0) {
    statusMessageDiv.textContent = message;
    statusMessageDiv.className = 'status-message ' + type; // e.g., 'status-message loading'
    statusMessageDiv.style.display = 'block';

    if (duration > 0) {
        setTimeout(() => {
            statusMessageDiv.style.display = 'none';
            statusMessageDiv.className = 'status-message';
        }, duration);
    }
}

