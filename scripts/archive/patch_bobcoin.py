with open('bobcoin/frontend/src/api.js', 'r') as f:
    content = f.read()
content = content.replace('const res = await fetch(`${LATTICE_URL}/proposals`);', 'const res = await fetch(`${LATTICE_URL}/governance/proposals`);')
with open('bobcoin/frontend/src/api.js', 'w') as f:
    f.write(content)
