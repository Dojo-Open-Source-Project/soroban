# SOROBAN NODEJS CLIENT DEMO

## PREREQUISITES

- Docker & Docker-Compose
- URL (over Tor) of the RPC API of a running Soroban 


## PREPARE AND BUILD THE DEMO

```bash
# Open a terminal and move to the demo directory ([...]/soroban/clients/js-nodes)

# Edit docker-compose.yml
> nano docker-compose.yml

# Replace the 2 lines "--url=http://sorg3s..." with the URL of the Soroban RPC API to be used and save the modifications

# Build the docker images
> docker compose build
```

### RUN THE DEMO
```bash
# Start the demo
> docker compose up -d

# Display the logs
> docker compose logs --follow

# Wait for torproxy's logs showing complete boostrapping
...
torproxy     | Dec 08 15:44:40.000 [notice] Bootstrapped 90% (ap_handshake_done): Handshake finished with a relay to build circuits
torproxy     | Dec 08 15:44:40.000 [notice] Bootstrapped 95% (circuit_create): Establishing a Tor circuit
torproxy     | Dec 08 15:44:41.000 [notice] Bootstrapped 100% (done): Done
...

# Wait for initiator and contributor logging the ping/pong loop
...
initiator    | Registering public key
contributor  | Initiator public key found
contributor  | Sending public key
contributor  | Starting exchange loop...
initiator    | Contributor public key found
initiator    | Starting exchange loop...
initiator    | Sending : Ping
contributor  | Received: Ping 1 1733672692150
contributor  | Replying: Pong
initiator    | Recieved: Pong 1 1733672693632
initiator    | Sending : Ping
contributor  | Received: Ping 2 1733672695156
contributor  | Replying: Pong
...
```


## STOP THE DEMO

```bash
# Stop the demo
> docker compose down
```