with open('internal/consensus/server.go', 'r') as f:
    content = f.read()

# Let's check handleProcess to see if it supports the wrapped block {"block": {...}} format
# We need to look at how it decodes the JSON request
# MEMORY.md says "The Go lattice now includes compatibility handling for all of the above, but this is a temporary bridge, not the final state."
# So it probably already handles it. I will verify.
