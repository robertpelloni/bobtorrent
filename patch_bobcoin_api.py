import re

with open('bobcoin/frontend/src/api.js', 'r') as f:
    content = f.read()

# Replace /proposals with /governance/proposals
content = content.replace('`${LATTICE_URL}/proposals`', '`${LATTICE_URL}/governance/proposals`')

# Ensure block posting is consistent. Let's see if the backend handles `{ block: ... }` vs raw block.
# Actually, our memory says: "some pages POST wrapped blocks as `{ block: ... }`" and "The Go lattice now includes compatibility handling for all of the above, but this is a temporary bridge, not the final state."
# We should update the frontend to POST the raw block if possible, but let's check what the backend expects first.
# Wait, the instruction says "Proceed with resolving the frontend-backend compatibility gap. Focus on the lattice dialect mismatch noted in MEMORY.md—ensure the React app correctly wraps blocks and uses the expected `/proposals` endpoint."
# Actually, the memory says "some pages POST wrapped blocks as `{ block: ... }`". Let's look at `process` in Go backend.
