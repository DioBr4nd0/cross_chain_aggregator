
 temp="/home/rupesh/.gammanet"
 sudo rm -rf .wasmd
 sudo rm -rf $temp
 wasmd init gammanet --chain-id=gammanet-1 --home="$temp"
 wasmd keys add alice --keyring-backend=test --home="$temp"
 wasmd keys add bob --keyring-backend=test --home="$temp"
 wasmd keys add backendop --keyring-backend=test --home="$temp"
 wasmd keys add replayer --keyring-backend=test --home="$temp"
 wasmd genesis add-genesis-account alice "1000000000stake" --keyring-backend=test --home="$temp"
 wasmd genesis add-genesis-account bob "1000000000stake" --keyring-backend=test --home="$temp"
 wasmd genesis gentx alice "2500000stake" --chain-id=gammanet-1 --amount="2500000stake" --keyring-backend=test --home="$temp"
 wasmd genesis collect-gentxs --home="$temp"
