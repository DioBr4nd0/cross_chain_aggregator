echo "--- Setting up IBC paths (Clients, Connections, Channels) ---"

# --- Path: AlphaNet <-> BetaNet ---
echo "Creating client: AlphaNet -> BetaNet"
hermes create client --host-chain alphanet-1 --reference-chain betanet-1
sleep 3
echo "Creating client: BetaNet -> AlphaNet"
hermes create client --host-chain betanet-1 --reference-chain alphanet-1
sleep 7 # Wait for clients to initialize

echo "Creating connection: AlphaNet <-> BetaNet"
# This command will output something like:
# Success: CreateConnection(CreateConnection {
#   a_side: Side { ..., connection_id: Some(ConnectionId("connection-X")) }, -> This is <conn_id_on_alpha_for_beta>
#   b_side: Side { ..., connection_id: Some(ConnectionId("connection-Y")) }, -> This is <conn_id_on_beta_for_alpha>
# })
# For the channel command, you need the connection ID from the perspective of the '--a-chain'.
hermes create connection --a-chain alphanet-1 --b-chain betanet-1
echo "ACTION: Note the ConnectionId for 'alphanet-1' from the output above. (e.g., connection-0)"
echo "Waiting for connection handshake..."
sleep 10

echo "Creating channel: AlphaNet <-> BetaNet (Port: transfer)"
# Replace <conn_id_on_alpha_for_beta> with the actual ID you noted.
# Example: if alphanet-1 got connection-0 for BetaNet, use connection-0.
hermes create channel --a-chain alphanet-1 --a-connection <conn_id_on_alpha_for_beta> --a-port transfer --b-port transfer --channel-version ics20-1
echo "ACTION: Note the ChannelId for 'alphanet-1' from this output (e.g., channel-0). Update backend config."
sleep 7

# --- Path: BetaNet <-> GammaNet ---
echo "Creating client: BetaNet -> GammaNet"
hermes create client --host-chain betanet-1 --reference-chain gammanet-1
sleep 3
echo "Creating client: GammaNet -> BetaNet"
hermes create client --host-chain gammanet-1 --reference-chain betanet-1
sleep 7

echo "Creating connection: BetaNet <-> GammaNet"
hermes create connection --a-chain betanet-1 --b-chain gammanet-1
echo "ACTION: Note the ConnectionId for 'betanet-1' from the output above."
echo "Waiting for connection handshake..."
sleep 10

echo "Creating channel: BetaNet <-> GammaNet (Port: transfer)"
# Replace <conn_id_on_beta_for_gamma> with the actual ID.
hermes create channel --a-chain betanet-1 --a-connection <conn_id_on_beta_for_gamma> --a-port transfer --b-port transfer --channel-version ics20-1
echo "ACTION: Note the ChannelId for 'betanet-1' from this output. Update backend config."
sleep 7

# --- Path: AlphaNet <-> GammaNet ---
echo "Creating client: AlphaNet -> GammaNet"
hermes create client --host-chain alphanet-1 --reference-chain gammanet-1
sleep 3
echo "Creating client: GammaNet -> AlphaNet"
hermes create client --host-chain gammanet-1 --reference-chain alphanet-1
sleep 7

echo "Creating connection: AlphaNet <-> GammaNet"
hermes create connection --a-chain alphanet-1 --b-chain gammanet-1
echo "ACTION: Note the ConnectionId for 'alphanet-1' from the output above."
echo "Waiting for connection handshake..."
sleep 10

echo "Creating channel: AlphaNet <-> GammaNet (Port: transfer)"
# Replace <conn_id_on_alpha_for_gamma> with the actual ID.
hermes create channel --a-chain alphanet-1 --a-connection <conn_id_on_alpha_for_gamma> --a-port transfer --b-port transfer --channel-version ics20-1
echo "ACTION: Note the ChannelId for 'alphanet-1' from this output. Update backend config."
sleep 7

echo "IBC path setup commands executed."
echo "IMPORTANT: Review the output of 'hermes create connection' and 'hermes create channel' for each pair."
echo "You MUST use the correct ConnectionIDs when creating channels."
echo "You MUST update your backend's config/app_config.go with the correct ChannelIDs created by these commands."
