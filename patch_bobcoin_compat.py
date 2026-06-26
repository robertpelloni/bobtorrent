with open('cmd/supernode-go/main.go', 'r') as f:
    content = f.read()

# Let's ensure Bobcoin legacy endpoints like /blocks and /bootstrap are mapped
if 'mux.HandleFunc("/blocks"' not in content:
    # Need to add them somewhere around the other routes.
    # We will search for a known route like /mint or /burn and append if missing.
    route_patch = """
	mux.HandleFunc("/blocks", withCORS(handleBlocks))
	mux.HandleFunc("/bootstrap", withCORS(handleBootstrap))
"""
    content = content.replace('mux.HandleFunc("/mint", withCORS(handleMint))', 'mux.HandleFunc("/mint", withCORS(handleMint))\n' + route_patch)

with open('cmd/supernode-go/main.go', 'w') as f:
    f.write(content)
