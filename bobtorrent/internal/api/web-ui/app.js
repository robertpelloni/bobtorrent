function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}
document.addEventListener('DOMContentLoaded', () => {
    // Tabs
    const tabs = document.querySelectorAll('.tab-btn');
    const contents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            contents.forEach(c => c.classList.remove('active'));

            tab.classList.add('active');
            document.getElementById(tab.dataset.tab).classList.add('active');
        });
    });

    // Identity
    const btnGenKey = document.getElementById('btn-generate-key');
    const inputPub = document.getElementById('id-pub');
    const inputPriv = document.getElementById('id-priv');
    const pubStatus = document.getElementById('pub-identity-status');

    let currentIdentity = null;

    btnGenKey.addEventListener('click', async () => {
        const res = await fetch('/key/generate', { method: 'POST' });
        const data = await res.json();
        currentIdentity = data;
        inputPub.value = data.publicKey;
        inputPriv.value = data.secretKey;
        pubStatus.textContent = 'Key Loaded';
        pubStatus.style.color = '#4caf50';
        document.getElementById('btn-publish').disabled = false;
        document.getElementById('btn-save-key').disabled = false;
    });

    // Publish
    const btnIngest = document.getElementById('btn-ingest');
    const inputPath = document.getElementById('ingest-path');
    const ingestResult = document.getElementById('ingest-result');
    const ingestJson = document.getElementById('ingest-json');
    let currentFileEntry = null;

    btnIngest.addEventListener('click', async () => {
        const path = inputPath.value;
        if (!path) return alert('Please enter a file path');

        btnIngest.textContent = 'Ingesting...';
        btnIngest.disabled = true;

        try {
            const res = await fetch('/ingest', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ filePath: path })
            });

            if (!res.ok) throw new Error((await res.json()).error || 'Ingest failed');

            const data = await res.json();
            currentFileEntry = data.fileEntry;
            ingestJson.textContent = JSON.stringify(data.fileEntry, null, 2);
            ingestResult.classList.remove('hidden');
        } catch (err) {
            alert(err.message);
        } finally {
            btnIngest.textContent = 'Ingest File';
            btnIngest.disabled = false;
        }
    });

    const btnPublish = document.getElementById('btn-publish');
    const publishResult = document.getElementById('publish-result');
    const publishJson = document.getElementById('publish-json');

    btnPublish.addEventListener('click', async () => {
        if (!currentFileEntry || !currentIdentity) return alert('Missing file or identity');

        btnPublish.textContent = 'Publishing...';
        btnPublish.disabled = true;

        try {
            const res = await fetch('/publish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    fileEntry: currentFileEntry,
                    identity: currentIdentity
                })
            });

            if (!res.ok) throw new Error((await res.json()).error || 'Publish failed');

            const data = await res.json();
            publishJson.textContent = JSON.stringify(data.manifest, null, 2);
            publishResult.classList.remove('hidden');
        } catch (err) {
            alert(err.message);
        } finally {
            btnPublish.textContent = 'Publish Manifest';
            btnPublish.disabled = false;
        }
    });

    // Subscribe
    const btnSubscribe = document.getElementById('btn-subscribe');
    const inputSubKey = document.getElementById('sub-key');
    const subsTable = document.getElementById('subs-table').querySelector('tbody');

    btnSubscribe.addEventListener('click', async () => {
        const key = inputSubKey.value;
        if (!key) return alert('Enter public key');

        try {
            const res = await fetch('/subscribe', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ publicKey: key })
            });

            if (!res.ok) throw new Error('Subscribe failed');

            inputSubKey.value = '';
            refreshSubscriptions();
        } catch (err) {
            alert(err.message);
        }
    });

    async function refreshSubscriptions() {
        try {
            const res = await fetch('/subscriptions');
            const subs = await res.json();

            subsTable.innerHTML = subs.length ? '' : '<tr><td colspan="4">No subscriptions yet.</td></tr>';

            subs.forEach(sub => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${sub.topicPath.substring(0, 16)}...</td>
                    <td>${sub.lastSequence || '-'}</td>
                    <td><span class="badge" style="background:#28a745">Active</span></td>
                    <td><button class="secondary-btn" style="padding: 2px 5px; font-size: 0.8rem">Details</button></td>
                `;
                subsTable.appendChild(tr);
            });
        } catch (e) {}
    }

    // Dashboard Status
    async function updateStatus() {
        try {
            const res = await fetch('/status');
            const data = await res.json();

            document.getElementById('dash-blobs').textContent = data.storage.blobs;
            document.getElementById('dash-size').textContent = (data.storage.size / 1024 / 1024).toFixed(2) + ' MB';
            document.getElementById('dash-max').textContent = (data.storage.max / 1024 / 1024 / 1024).toFixed(2) + ' GB';
            document.getElementById('dash-util').textContent = (data.storage.utilization * 100).toFixed(1) + '%';
            document.getElementById('dash-dht').textContent = data.dht;
            document.getElementById('dash-subs').textContent = data.subscriptions;

            document.getElementById('dht-status').querySelector('.value').textContent = data.dht;
            document.getElementById('network-status').querySelector('.value').textContent = data.network;
        } catch (e) {}
    }

    // Blobs Table
    async function updateBlobs() {
        try {
            const res = await fetch('/blobs');
            const blobs = await res.json();
            const table = document.getElementById('blobs-table').querySelector('tbody');

            table.innerHTML = blobs.length ? '' : '<tr><td colspan="3">No blobs found.</td></tr>';

            blobs.forEach(blob => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${blob.blobId.substring(0, 32)}...</td>
                    <td>${blob.size} bytes</td>
                    <td>${new Date(blob.addedAt).toLocaleString()}</td>
                `;
                table.appendChild(tr);
            });
        } catch (e) {}
    }

    // Polling
    setInterval(updateStatus, 2000);
    setInterval(refreshSubscriptions, 5000);
    setInterval(updateBlobs, 5000);

    // Initial Load
    updateStatus();
    refreshSubscriptions();
    updateBlobs();
});


// Asset Registry
async function fetchAssets() {
    try {
        const res = await fetch('/assets');
        const assets = await res.json();
        const tbody = document.querySelector('#assets-table tbody');
        
        if (!assets || assets.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4">No assets found.</td></tr>';
            return;
        }

        tbody.innerHTML = assets.map(a => `
            <tr>
                <td title="${a.id}"><a href="/stream/${a.id}" target="_blank" title="Stream this asset directly">${a.id.substring(0, 16)}...</a></td>
                <td>${a.filename}</td>
                <td>${formatBytes(a.size)}</td>
                <td>${a.chunks}</td>
            </tr>
        `).join('');
    } catch (e) {
        console.error('Failed to fetch assets:', e);
        document.querySelector('#assets-table tbody').innerHTML = '<tr><td colspan="4" class="error">Failed to load registry.</td></tr>';
    }
}

// Ensure fetchAssets runs when switching tabs
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        if (btn.dataset.tab === 'assets') {
            fetchAssets();
        }
    });
});


// Wallet & Lattice
async function fetchWallet() {
    try {
        const res = await fetch('/api/wallet');
        const data = await res.json();
        
        if (data.address) {
            document.getElementById('wallet-address').textContent = data.address;
            document.getElementById('wallet-balance').textContent = (data.balance / 1000000000).toFixed(4) + ' SOL';
        }
    } catch (e) {
        console.error('Failed to fetch wallet info:', e);
        document.getElementById('wallet-address').textContent = 'Error connecting to Wallet';
    }
}

const btnAirdrop = document.getElementById('btn-airdrop');
if (btnAirdrop) {
    btnAirdrop.addEventListener('click', async () => {
        try {
            btnAirdrop.disabled = true;
            btnAirdrop.textContent = 'Requesting...';
            
            const res = await fetch('/api/wallet/airdrop', { method: 'POST' });
            const data = await res.json();
            
            const resultBox = document.getElementById('airdrop-result');
            const resultJson = document.getElementById('airdrop-json');
            
            resultJson.textContent = JSON.stringify(data, null, 2);
            resultBox.classList.remove('hidden');
            
            // Refresh wallet balance
            setTimeout(fetchWallet, 5000);
        } catch (e) {
            console.error('Airdrop failed:', e);
            document.getElementById('airdrop-json').textContent = 'Error requesting airdrop: ' + e.message;
            document.getElementById('airdrop-result').classList.remove('hidden');
        } finally {
            btnAirdrop.disabled = false;
            btnAirdrop.textContent = 'Request Airdrop';
        }
    });
}

// Ensure fetchWallet runs when switching tabs
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        if (btn.dataset.tab === 'wallet') {
            fetchWallet();
        }
    });
});


// Identity Verification Badges
const btnVerify = document.getElementById('btn-verify');
if (btnVerify) {
    btnVerify.addEventListener('click', async () => {
        try {
            btnVerify.disabled = true;
            btnVerify.textContent = 'Verifying...';
            
            const provider = document.getElementById('verify-provider').value;
            const identifier = document.getElementById('verify-identifier').value;
            const attestation = document.getElementById('verify-attestation').value;
            const publicKey = document.getElementById('id-pub').value;

            if (!publicKey) {
                alert('Please generate an identity first.');
                return;
            }

            const res = await fetch('/api/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider, identifier, attestation, publicKey })
            });

            const data = await res.json();
            
            const resultBox = document.getElementById('verify-result');
            const resultJson = document.getElementById('verify-json');
            
            resultJson.textContent = JSON.stringify(data, null, 2);
            resultBox.classList.remove('hidden');
            
            if (data.valid) {
                resultBox.style.borderLeftColor = 'var(--success-color)';
            } else {
                resultBox.style.borderLeftColor = 'var(--danger-color)';
            }
        } catch (e) {
            console.error('Verification failed:', e);
            document.getElementById('verify-json').textContent = 'Error connecting to verifier: ' + e.message;
            document.getElementById('verify-result').classList.remove('hidden');
        } finally {
            btnVerify.disabled = false;
            btnVerify.textContent = 'Verify Identity';
        }
    });
}


// Lattice Visualization
async function fetchLattice() {
    try {
        const res = await fetch('/api/lattice');
        const blocks = await res.json();
        const tbody = document.querySelector('#lattice-table tbody');
        
        if (!blocks || blocks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4">Lattice is empty or syncing.</td></tr>';
            return;
        }

        tbody.innerHTML = blocks.map(b => `
            <tr>
                <td title="${b.hash}">${b.hash.substring(0, 16)}...</td>
                <td>${b.height}</td>
                <td title="${b.producer}">${b.producer.substring(0, 12)}...</td>
                <td>${new Date(b.timestamp).toLocaleTimeString()}</td>
            </tr>
        `).join('');
    } catch (e) {
        console.error('Failed to fetch lattice:', e);
        document.querySelector('#lattice-table tbody').innerHTML = '<tr><td colspan="4" class="error">Failed to load lattice blocks.</td></tr>';
    }
}

// Add fetchLattice to the wallet tab listener
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        if (btn.dataset.tab === 'wallet') {
            fetchLattice();
        }
    });
});
