// Megatorrent UI Logic
// Interacts with /api/v2/megatorrent/ endpoints

function showMegaTab(tabId) {
    document.querySelectorAll('.mega-tab').forEach(el => el.classList.add('invisible'));
    document.getElementById(tabId).classList.remove('invisible');
}

// Ensure the invisible class is available in context (it's usually in style.css)
// If not, we might need to inline style or rely on qBittorrent's existing classes.
// Assuming 'invisible' works as it's used in index.html

async function megaGenerateKey() {
    try {
        const response = await fetch('api/v2/megatorrent/generateKeyAction', { method: 'POST' });
        if (!response.ok) throw new Error('Failed to generate key');
        const data = await response.json();

        document.getElementById('mega-pub').value = data.publicKey;
        document.getElementById('mega-priv').value = data.secretKey;
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

async function megaIngest() {
    const path = document.getElementById('mega-ingest-path').value;
    if (!path) return alert('Please enter a file path');

    document.getElementById('mega-publish-log').innerText = 'Ingesting...';
    document.getElementById('mega-publish-result').classList.remove('invisible');

    try {
        // Ingest
        const ingestRes = await fetch('api/v2/megatorrent/ingestAction', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `filePath=${encodeURIComponent(path)}`
        });

        if (!ingestRes.ok) throw new Error('Ingest failed');
        const ingestData = await ingestRes.json();

        document.getElementById('mega-publish-log').innerText =
            `Ingested ${ingestData.fileEntry.name}\n` +
            `Blobs: ${ingestData.blobCount}\n` +
            `Publishing Manifest...`;

        // Publish (stubbed parameters)
        const pubRes = await fetch('api/v2/megatorrent/publishAction', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `manifest=${encodeURIComponent(JSON.stringify(ingestData))}&privateKey=stub`
        });

        if (!pubRes.ok) throw new Error('Publish failed');
        const pubData = await pubRes.json();

        document.getElementById('mega-publish-log').innerText +=
            `\nSuccess! Sequence: ${pubData.sequence}`;

    } catch (e) {
        document.getElementById('mega-publish-log').innerText = 'Error: ' + e.message;
    }
}

async function megaSubscribe() {
    const key = document.getElementById('mega-sub-key').value;
    if (!key) return;

    try {
        await fetch('api/v2/megatorrent/subscribeAction', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `publicKey=${encodeURIComponent(key)}`
        });
        document.getElementById('mega-sub-key').value = '';
        updateMegaSubs();
    } catch (e) {
        alert(e.message);
    }
}

async function updateMegaSubs() {
    try {
        const res = await fetch('api/v2/megatorrent/subscriptionsAction');
        const subs = await res.json();

        const tbody = document.getElementById('mega-subs-list');
        tbody.innerHTML = '';

        if (subs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4">No subscriptions.</td></tr>';
            return;
        }

        subs.forEach(sub => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td>${sub.publicKey.substring(0, 16)}...</td>
                <td>${sub.lastSequence}</td>
                <td>${sub.status}</td>
                <td><button onclick="megaUnsubscribe('${sub.publicKey}')">Unsub</button></td>
            `;
            tbody.appendChild(tr);
        });
    } catch (e) {}
}

async function megaUnsubscribe(key) {
    if (!confirm('Unsubscribe?')) return;
    try {
        await fetch('api/v2/megatorrent/unsubscribeAction', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `publicKey=${encodeURIComponent(key)}`
        });
        updateMegaSubs();
    } catch (e) {}
}

async function updateMegaStatus() {
    try {
        const res = await fetch('api/v2/megatorrent/statusAction');
        const data = await res.json();

        document.getElementById('mega-dht').innerText = data.dht;
        document.getElementById('mega-net').innerText = data.network;
        document.getElementById('mega-blobs-count').innerText = data.blobStore.blobs;
    } catch (e) {}
}

// Initial Load
document.addEventListener('DOMContentLoaded', () => {
    // We need to wait for qBittorrent to load, or just poll
    setInterval(updateMegaStatus, 5000);
    setInterval(updateMegaSubs, 5000);
    updateMegaStatus();
    updateMegaSubs();
});
