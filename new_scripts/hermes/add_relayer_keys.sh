echo "--- Adding relayer keys to Hermes ---"

# Key for AlphaNet
hermes keys add --chain alphanet-1 --mnemonic-file "$HOME/.hermes/relayer_1.txt" --key-name relayer-alpha --overwrite

# Key for BetaNet
hermes keys add --chain betanet-1 --mnemonic-file "$HOME/.hermes/relayer_2.txt" --key-name relayer-beta --overwrite

# Key for GammaNet
hermes keys add --chain gammanet-1 --mnemonic-file "$HOME/.hermes/relayer_3.txt" --key-name relayer-gamma --overwrite

echo "Relayer keys added. Listing keys:"
hermes keys list --chain alphanet-1
hermes keys list --chain betanet-1
hermes keys list --chain gammanet-1
