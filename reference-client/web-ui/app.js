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
        const res = await fetch('/api/key/generate', { method: 'POST' });
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
            const res = await fetch('/api/ingest', {
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
            const res = await fetch('/api/publish', {
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
            const res = await fetch('/api/subscribe', {
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
            const res = await fetch('/api/subscriptions');
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

    // Discovery
    const btnBrowse = document.getElementById('btn-browse');
    const inputDiscoveryPath = document.getElementById('discovery-path');
    const discoveryBox = document.getElementById('discovery-box');
    const discoverySubtopics = document.getElementById('discovery-subtopics');
    const discoveryPublishers = document.getElementById('discovery-publishers');

    btnBrowse.addEventListener('click', async () => {
        const topic = inputDiscoveryPath.value;
        btnBrowse.disabled = true;
        btnBrowse.textContent = 'Searching...';

        try {
            const res = await fetch(`/api/channels/browse?topic=${encodeURIComponent(topic)}`);
            if (!res.ok) throw new Error('Browse failed');
            const result = await res.json();

            discoverySubtopics.innerHTML = '';
            discoveryPublishers.innerHTML = '';

            if (result.subtopics.length === 0 && result.publishers.length === 0) {
                discoverySubtopics.innerHTML = '<li>No results found.</li>';
            }

            result.subtopics.forEach(st => {
                const li = document.createElement('li');
                li.innerHTML = `<a href="#" onclick="document.getElementById('discovery-path').value='${topic ? topic+'/' : ''}${st}'; document.getElementById('btn-browse').click(); return false;">📁 ${st}</a>`;
                discoverySubtopics.appendChild(li);
            });

            result.publishers.forEach(pub => {
                const li = document.createElement('li');
                li.innerHTML = `👤 ${pub.name || 'Unknown'} <small>(${pub.pk.substring(0,8)}...)</small> <button class="secondary-btn" style="padding: 2px 6px; font-size: 0.8rem;" onclick="document.getElementById('sub-key').value='${pub.pk}'; document.querySelector('[data-tab=subscribe]').click();">Sub</button>`;
                discoveryPublishers.appendChild(li);
            });

            discoveryBox.classList.remove('hidden');
        } catch (e) {
            alert(e.message);
        } finally {
            btnBrowse.disabled = false;
            btnBrowse.textContent = 'Browse';
        }
    });

    // Files Table
    async function updateFiles() {
        try {
            const res = await fetch('/api/files');
            const files = await res.json();
            const table = document.getElementById('files-table').querySelector('tbody');

            table.innerHTML = files.length ? '' : '<tr><td colspan="4">No files ingested.</td></tr>';

            files.forEach(f => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${f.name}</td>
                    <td>${(f.size / 1024 / 1024).toFixed(2)} MB</td>
                    <td>${f.status}</td>
                    <td><progress value="${f.progress}" max="100"></progress> ${f.progress}%</td>
                `;
                table.appendChild(tr);
            });
        } catch (e) {}
    }

    // Dashboard Status
    async function updateStatus() {
        try {
            const res = await fetch('/api/status');
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
            const res = await fetch('/api/blobs');
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
    setInterval(updateFiles, 5000);

    // Initial Load
    updateStatus();
    refreshSubscriptions();
    updateBlobs();
    updateFiles();
});
