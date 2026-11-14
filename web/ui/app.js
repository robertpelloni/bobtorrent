document.addEventListener('DOMContentLoaded', () => {
    // Nav Tab Switching (Sovereign Modules)
    const tabBtns = document.querySelectorAll('.nav-tab-btn');
    const sections = document.querySelectorAll('.console-section');

    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            tabBtns.forEach(b => b.classList.remove('active'));
            sections.forEach(s => s.classList.remove('active'));

            btn.classList.add('active');
            const target = btn.dataset.target;
            const targetSection = document.getElementById(target);
            if (targetSection) {
                targetSection.classList.add('active');
            }
        });
    });

    // Identity Management
    const btnGenKey = document.getElementById('btn-generate-key');
    const btnQuickGenKey = document.getElementById('btn-quick-gen-key');
    const inputPub = document.getElementById('id-pub');
    const inputPriv = document.getElementById('id-priv');
    const pubStatus = document.getElementById('pub-identity-status');
    const btnSaveKey = document.getElementById('btn-save-key');
    const btnPublish = document.getElementById('btn-publish');

    let currentIdentity = null;

    async function generateIdentity() {
        try {
            const res = await fetch('/key/generate', { method: 'POST' });
            if (!res.ok) throw new Error('Key generation failed');
            const data = await res.json();
            currentIdentity = data;

            inputPub.value = data.publicKey;
            inputPriv.value = data.secretKey;

            pubStatus.textContent = `Active (${data.publicKey.substring(0, 8)}...)`;
            pubStatus.className = 'status-pill success';

            btnPublish.disabled = false;
            btnSaveKey.disabled = false;
        } catch (err) {
            alert('Failed to generate keypair: ' + err.message);
        }
    }

    if (btnGenKey) btnGenKey.addEventListener('click', generateIdentity);
    if (btnQuickGenKey) btnQuickGenKey.addEventListener('click', generateIdentity);

    if (btnSaveKey) {
        btnSaveKey.addEventListener('click', () => {
            if (!currentIdentity) return;
            const blob = new Blob([JSON.stringify(currentIdentity, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `identity-${currentIdentity.publicKey.substring(0, 8)}.json`;
            a.click();
            URL.revokeObjectURL(url);
        });
    }

    // Verify Attestation (SSRF Protected)
    const btnVerify = document.getElementById('btn-verify-attestation');
    const verifyResultBox = document.getElementById('verify-result');
    const verifyJson = document.getElementById('verify-json');
    const verifyBadge = document.getElementById('verify-status-badge');

    if (btnVerify) {
        btnVerify.addEventListener('click', async () => {
            const provider = document.getElementById('verify-provider').value;
            const identifier = document.getElementById('verify-identifier').value.trim();
            const url = document.getElementById('verify-url').value.trim();
            const account = document.getElementById('verify-account').value.trim();

            if (!url) return alert('Proof URL is required.');

            btnVerify.textContent = 'Verifying Attestation...';
            btnVerify.disabled = true;

            try {
                const res = await fetch('/verify-attestation', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        kind: provider,
                        url: url,
                        account: account || identifier
                    })
                });

                const data = await res.json();
                verifyJson.textContent = JSON.stringify(data, null, 2);
                verifyResultBox.classList.remove('hidden');

                if (data.success) {
                    verifyBadge.textContent = 'VERIFIED';
                    verifyBadge.className = 'badge success';
                } else {
                    verifyBadge.textContent = 'UNVERIFIED';
                    verifyBadge.className = 'badge danger';
                }
            } catch (err) {
                alert('Verification request failed: ' + err.message);
            } finally {
                btnVerify.textContent = 'Verify Attestation Now';
                btnVerify.disabled = false;
            }
        });
    }

    // Ingest & Publish
    const btnIngest = document.getElementById('btn-ingest');
    const inputPath = document.getElementById('ingest-path');
    const ingestResult = document.getElementById('ingest-result');
    const ingestJson = document.getElementById('ingest-json');
    let currentFileEntry = null;

    if (btnIngest) {
        btnIngest.addEventListener('click', async () => {
            const path = inputPath.value.trim();
            if (!path) return alert('Please enter a filename relative to data/ingest/');

            btnIngest.textContent = 'Ingesting...';
            btnIngest.disabled = true;

            try {
                const res = await fetch('/ingest', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ filePath: path })
                });

                if (!res.ok) {
                    const errData = await res.json().catch(() => ({ error: 'Ingest failed' }));
                    throw new Error(errData.error || errData.message || 'Ingest failed');
                }

                const data = await res.json();
                currentFileEntry = data.fileEntry;
                ingestJson.textContent = JSON.stringify(data.fileEntry, null, 2);
                ingestResult.classList.remove('hidden');

                if (currentIdentity) {
                    btnPublish.disabled = false;
                }
            } catch (err) {
                alert('Ingestion error: ' + err.message);
            } finally {
                btnIngest.textContent = 'Ingest File';
                btnIngest.disabled = false;
            }
        });
    }

    const publishResult = document.getElementById('publish-result');
    const publishJson = document.getElementById('publish-json');

    if (btnPublish) {
        btnPublish.addEventListener('click', async () => {
            if (!currentFileEntry) return alert('Please ingest a file first.');
            if (!currentIdentity) return alert('No cryptographic identity loaded.');

            btnPublish.textContent = 'Broadcasting Manifest...';
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

                if (!res.ok) {
                    const errData = await res.json().catch(() => ({ error: 'Publish failed' }));
                    throw new Error(errData.error || 'Publish failed');
                }

                const data = await res.json();
                publishJson.textContent = JSON.stringify(data.manifest, null, 2);
                publishResult.classList.remove('hidden');

                // Refresh assets index
                updateAssets();
            } catch (err) {
                alert('Manifest publish error: ' + err.message);
            } finally {
                btnPublish.textContent = 'Publish Manifest to Network';
                btnPublish.disabled = false;
            }
        });
    }

    // Subscriptions
    const btnSubscribe = document.getElementById('btn-subscribe');
    const inputSubKey = document.getElementById('sub-key');
    const subsTable = document.getElementById('subs-table')?.querySelector('tbody');

    if (btnSubscribe) {
        btnSubscribe.addEventListener('click', async () => {
            const key = inputSubKey.value.trim();
            if (!key) return alert('Please enter publisher public key.');

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
    }

    async function refreshSubscriptions() {
        if (!subsTable) return;
        try {
            const res = await fetch('/subscriptions');
            if (!res.ok) return;
            const subs = await res.json();

            subsTable.innerHTML = subs.length ? '' : '<tr><td colspan="4">No subscriptions registered.</td></tr>';

            subs.forEach(sub => {
                const tr = document.createElement('tr');
                const keyStr = typeof sub.publicKey === 'string' ? sub.publicKey : JSON.stringify(sub.publicKey);
                tr.innerHTML = `
                    <td class="mono">${keyStr.substring(0, 16)}...</td>
                    <td>${sub.lastSequence || '1'}</td>
                    <td><span class="badge success">Tracking</span></td>
                    <td><button class="secondary-btn small-btn" onclick="alert('Viewing Channel: ${keyStr}')">Inspect</button></td>
                `;
                subsTable.appendChild(tr);
            });
        } catch (e) {}
    }

    // Decentralized libp2p GossipSub Chat Mesh
    let messengerWs = null;
    let activeTopic = 'bobtorrent-global-gossip';
    const chatTopicHistory = {};
    let nodePeerId = 'LocalSupernode';

    function initMessenger() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        messengerWs = new WebSocket(`${protocol}//${window.location.host}/ws-messenger`);

        messengerWs.onopen = () => {
            console.log('[Messenger] Connected to libp2p gossip mesh bridge');
            const chatNodeLabel = document.getElementById('chat-node-id');
            if (chatNodeLabel) chatNodeLabel.textContent = 'Mesh Synchronized';
            messengerWs.send(JSON.stringify({ type: 'FETCH_HISTORY', topic: activeTopic }));
        };

        messengerWs.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'GOSSIP') {
                    handleIncomingGossip(msg);
                } else if (msg.type === 'HISTORY') {
                    handleIncomingHistory(msg);
                } else if (msg.type === 'JOINED') {
                    addNotice(`Joined #${msg.topic}`);
                } else if (msg.type === 'LEFT') {
                    addNotice(`Left #${msg.topic}`);
                }
            } catch (e) {
                console.warn('WS parse error:', e);
            }
        };

        messengerWs.onclose = () => {
            const chatNodeLabel = document.getElementById('chat-node-id');
            if (chatNodeLabel) chatNodeLabel.textContent = 'Disconnected, retrying...';
            setTimeout(initMessenger, 3000);
        };
    }

    function handleIncomingGossip(msg) {
        if (!chatTopicHistory[msg.topic]) chatTopicHistory[msg.topic] = [];
        chatTopicHistory[msg.topic].push(msg);

        if (msg.topic === activeTopic) {
            renderMessage(msg);
        }
    }

    function handleIncomingHistory(msg) {
        if (!msg.messages || !Array.isArray(msg.messages)) return;
        chatTopicHistory[msg.topic] = msg.messages.map(m => ({
            topic: m.topic || msg.topic,
            from: m.sender || m.from || 'Anonymous',
            data: m.data,
            timestamp: m.timestamp
        }));

        if (msg.topic === activeTopic) {
            refreshChatMessages();
        }
    }

    function refreshChatMessages() {
        const container = document.getElementById('chat-messages');
        if (!container) return;
        container.innerHTML = '';
        const history = chatTopicHistory[activeTopic] || [];
        history.forEach(renderMessage);
    }

    function renderMessage(msg) {
        const container = document.getElementById('chat-messages');
        if (!container) return;
        const div = document.createElement('div');
        div.className = 'chat-message';
        if (msg.from === nodePeerId || msg.from === 'You') div.classList.add('self');

        const sender = document.createElement('div');
        sender.className = 'sender';
        const senderStr = String(msg.from || 'Anonymous');
        sender.textContent = senderStr.length > 16 ? senderStr.substring(0, 14) + '...' : senderStr;

        const text = document.createElement('div');
        text.className = 'text';
        text.textContent = msg.data;

        div.appendChild(sender);
        div.appendChild(text);
        container.appendChild(div);
        container.scrollTop = container.scrollHeight;
    }

    function addNotice(text) {
        const container = document.getElementById('chat-messages');
        if (!container) return;
        const div = document.createElement('div');
        div.className = 'chat-notice';
        div.textContent = text;
        container.appendChild(div);
        container.scrollTop = container.scrollHeight;
    }

    const btnSendChat = document.getElementById('btn-send-chat');
    const inputChat = document.getElementById('chat-input');

    if (btnSendChat && inputChat) {
        btnSendChat.addEventListener('click', () => {
            const text = inputChat.value.trim();
            if (!text || !messengerWs || messengerWs.readyState !== WebSocket.OPEN) return;

            messengerWs.send(JSON.stringify({
                type: 'PUBLISH',
                topic: activeTopic,
                data: text
            }));

            // Optimistic echo
            const myMsg = { from: 'You', data: text, topic: activeTopic };
            handleIncomingGossip(myMsg);

            inputChat.value = '';
        });

        inputChat.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') btnSendChat.click();
        });
    }

    const btnJoinTopic = document.getElementById('btn-join-topic');
    const inputNewTopic = document.getElementById('new-topic-name');
    const topicList = document.getElementById('topic-list');
    const activeTopicLabel = document.getElementById('active-topic-name');

    function switchTopic(name) {
        activeTopic = name;
        if (activeTopicLabel) activeTopicLabel.textContent = `# ${name}`;
        document.querySelectorAll('.topic-list li').forEach(li => {
            li.classList.toggle('active', li.dataset.topic === name);
        });
        refreshChatMessages();
    }

    if (btnJoinTopic && inputNewTopic) {
        btnJoinTopic.addEventListener('click', () => {
            const name = inputNewTopic.value.trim().replace(/^#\s*/, '');
            if (!name) return;

            if (messengerWs && messengerWs.readyState === WebSocket.OPEN) {
                messengerWs.send(JSON.stringify({ type: 'JOIN_TOPIC', topic: name }));
            }

            if (!document.querySelector(`li[data-topic="${name}"]`)) {
                const li = document.createElement('li');
                li.dataset.topic = name;
                li.textContent = `# ${name}`;
                li.addEventListener('click', () => switchTopic(name));
                topicList.appendChild(li);
            }

            switchTopic(name);
            inputNewTopic.value = '';
        });
    }

    document.querySelectorAll('.topic-list li').forEach(li => {
        li.addEventListener('click', () => switchTopic(li.dataset.topic));
    });

    initMessenger();

    // Telemetry & Stats Polling
    async function updateStatus() {
        try {
            const res = await fetch('/stats');
            if (!res.ok) return;
            const data = await res.json();

            const blobsElem = document.getElementById('dash-blobs');
            const sizeElem = document.getElementById('dash-size');
            const maxElem = document.getElementById('dash-max');
            const utilElem = document.getElementById('dash-util');
            const dhtElem = document.getElementById('dash-dht');
            const subsElem = document.getElementById('dash-subs');
            const downSpeedElem = document.getElementById('dash-down-speed');
            const upSpeedElem = document.getElementById('dash-up-speed');

            if (blobsElem) blobsElem.textContent = data.storage?.blobs ?? 0;
            if (sizeElem) sizeElem.textContent = ((data.storage?.size || 0) / (1024 * 1024)).toFixed(2) + ' MB';
            if (maxElem) maxElem.textContent = data.storage?.max ? ((data.storage.max / (1024 * 1024 * 1024)).toFixed(1) + ' GB') : 'Unlimited';
            if (utilElem) utilElem.textContent = ((data.storage?.utilization || 0) * 100).toFixed(1) + '%';
            if (dhtElem) dhtElem.textContent = data.dht ?? 0;
            if (subsElem) subsElem.textContent = data.subscriptions ?? 0;

            if (downSpeedElem && data.network) {
                downSpeedElem.textContent = ((data.network.downloadSpeed || 0) / 1024).toFixed(1) + ' KB/s';
            }
            if (upSpeedElem && data.network) {
                upSpeedElem.textContent = ((data.network.uploadSpeed || 0) / 1024).toFixed(1) + ' KB/s';
            }

            if (data.address) {
                const walletAddr = document.getElementById('wallet-address');
                if (walletAddr) walletAddr.textContent = data.address;
                nodePeerId = data.address;
                const chatNodeLabel = document.getElementById('chat-node-id');
                if (chatNodeLabel) chatNodeLabel.textContent = data.address.substring(0, 12) + '...';
            }

            // Populate Downloads
            updateDownloadsTable(data.storage?.torrents || []);
        } catch (e) {}
    }

    function updateDownloadsTable(torrents) {
        const table = document.getElementById('downloads-table')?.querySelector('tbody');
        if (!table) return;

        table.innerHTML = torrents.length ? '' : '<tr><td colspan="5">No active downloads or torrents.</td></tr>';

        torrents.forEach(t => {
            const tr = document.createElement('tr');
            const progressPct = ((t.progress || 0) * 100).toFixed(1);
            const totalMb = ((t.totalSize || 0) / (1024 * 1024)).toFixed(2);
            tr.innerHTML = `
                <td class="mono">${t.name || t.infoHash}</td>
                <td style="min-width: 140px;">
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${progressPct}%"></div>
                    </div>
                    <small>${progressPct}%</small>
                </td>
                <td><span class="badge success">Seeding</span></td>
                <td>${t.peers || 0}</td>
                <td>${totalMb} MB</td>
            `;
            table.appendChild(tr);
        });
    }

    // Blobs Table
    async function updateBlobs() {
        try {
            const res = await fetch('/blobs');
            if (!res.ok) return;
            const data = await res.json();
            const blobs = data.blobs || data;
            const table = document.getElementById('blobs-table')?.querySelector('tbody');
            if (!table) return;

            table.innerHTML = (Array.isArray(blobs) && blobs.length) ? '' : '<tr><td colspan="3">No physical blobs found in storage directory.</td></tr>';

            if (Array.isArray(blobs)) {
                blobs.forEach(b => {
                    const tr = document.createElement('tr');
                    const bId = b.id || b.blobId || 'Unknown';
                    const bSize = b.size ? `${(b.size / 1024).toFixed(1)} KB` : '1024 KB';
                    const bTime = b.timestamp || b.addedAt ? new Date(b.timestamp || b.addedAt).toLocaleString() : 'Now';
                    tr.innerHTML = `
                        <td class="mono">${bId.length > 36 ? bId.substring(0, 34) + '...' : bId}</td>
                        <td>${bSize}</td>
                        <td>${bTime}</td>
                    `;
                    table.appendChild(tr);
                });
            }
        } catch (e) {}
    }

    // Assets Registry
    async function updateAssets() {
        try {
            const res = await fetch('/assets');
            if (!res.ok) return;
            const data = await res.json();
            const assets = data.assets || (Array.isArray(data) ? data : []);
            const table = document.getElementById('assets-table')?.querySelector('tbody');
            if (!table) return;

            table.innerHTML = assets.length ? '' : '<tr><td colspan="4">No manifests anchored in local registry yet.</td></tr>';

            assets.forEach(a => {
                const tr = document.createElement('tr');
                const id = a.id || a.manifestId || 'unknown';
                const name = a.name || a.filename || 'asset.bin';
                const size = a.size ? `${(a.size / (1024 * 1024)).toFixed(2)} MB` : '1 MB';
                const chunks = a.chunks || a.chunkCount || 1;
                tr.innerHTML = `
                    <td class="mono">${id.substring(0, 20)}...</td>
                    <td><strong>${name}</strong></td>
                    <td>${size}</td>
                    <td>${chunks}</td>
                `;
                table.appendChild(tr);
            });
        } catch (e) {}
    }

    // Lattice Blocks Feed
    async function updateLattice() {
        try {
            const res = await fetch('/lattice');
            if (!res.ok) return;
            const blocks = await res.json();
            const table = document.getElementById('lattice-table')?.querySelector('tbody');
            if (!table || !Array.isArray(blocks)) return;

            table.innerHTML = blocks.length ? '' : '<tr><td colspan="4">No blocks confirmed yet.</td></tr>';

            blocks.forEach(b => {
                const tr = document.createElement('tr');
                const hash = b.hash || 'unknown';
                const producer = b.producer || 'Autonomous';
                const timeStr = b.timestamp ? new Date(b.timestamp).toLocaleTimeString() : 'Recent';
                tr.innerHTML = `
                    <td class="mono">${hash.substring(0, 24)}...</td>
                    <td>#${b.height}</td>
                    <td class="mono">${producer.substring(0, 16)}...</td>
                    <td>${timeStr}</td>
                `;
                table.appendChild(tr);
            });
        } catch (e) {}
    }

    // Devnet Airdrop
    const btnAirdrop = document.getElementById('btn-airdrop');
    const airdropResult = document.getElementById('airdrop-result');
    const airdropJson = document.getElementById('airdrop-json');

    if (btnAirdrop) {
        btnAirdrop.addEventListener('click', async () => {
            btnAirdrop.textContent = 'Requesting Airdrop...';
            btnAirdrop.disabled = true;

            try {
                // No airdrop endpoint exists on this daemon. Show the real
                // state instead of fabricating a confirmation/balance.
                const res = await fetch('/api/wallet/airdrop', { method: 'POST' }).catch(() => null);
                if (!res || !res.ok) {
                    airdropJson.textContent = JSON.stringify({
                        status: 'UNAVAILABLE',
                        detail: 'This daemon exposes no /api/wallet/airdrop endpoint, so no SOL was credited. Use a Solana Devnet faucet against the address above.'
                    }, null, 2);
                    airdropResult.classList.remove('hidden');
                    return;
                }
                const resultData = await res.json();
                airdropJson.textContent = JSON.stringify(resultData, null, 2);
                airdropResult.classList.remove('hidden');
            } catch (err) {
                alert('Airdrop failed: ' + err.message);
            } finally {
                btnAirdrop.textContent = 'Request Devnet Airdrop';
                btnAirdrop.disabled = false;
            }
        });
    }

    // Services & Network Health + Economy + Torrent actions
    async function jget(url) { const r = await fetch(url); if (!r.ok) throw new Error(url + ' ' + r.status); return r.json(); }
    function showBox(boxId, preId, data) { const b = document.getElementById(boxId), p = document.getElementById(preId); if (b && p) { p.textContent = JSON.stringify(data, null, 2); b.classList.remove('hidden'); } }

    async function updateServices() {
        const table = document.getElementById('services-table')?.querySelector('tbody');
        if (!table) return;
        const rows = [];
        try { const s = await jget('/status'); rows.push(['Supernode (HTTP :8000)', s.status === 'online', `v${s.version || '?'} · ${s.service || ''}`]); const fv = document.getElementById('footer-version'); if (fv && s.version) fv.textContent = s.version; } catch { rows.push(['Supernode (HTTP :8000)', false, 'unreachable']); }
        try { const s = await jget('/status/i2p'); rows.push(['I2P Anonymous Datagrams', !!s.enabled, s.status || '']); } catch { rows.push(['I2P Anonymous Datagrams', false, 'unreachable']); }
        try { const s = await jget('/filecoin/status'); rows.push(['Filecoin Archival Bridge', s.enabled !== false && s.mode !== 'disabled', s.mode || JSON.stringify(s).substring(0, 60)]); } catch { rows.push(['Filecoin Archival Bridge', false, 'unreachable']); }
        try { const c = await jget('/channels/browse'); const first = Array.isArray(c) ? c[0] : null; rows.push(['Public DHT Swarm Channel', !!first, first ? `${first.peers ?? 0} peers · ${first.dhtNodes ?? 0} DHT nodes` : 'none']); } catch { rows.push(['Public DHT Swarm Channel', false, 'unreachable']); }
        table.innerHTML = rows.map(([n, on, d]) => `<tr><td>${n}</td><td><span class="badge ${on ? 'success' : 'danger'}">${on ? 'ONLINE' : 'OFF'}</span></td><td class="mono">${d}</td></tr>`).join('');
    }

    async function updateEconomy() {
        try { const b = await jget('/bankroll'); const bal = document.getElementById('bankroll-balance'), st = document.getElementById('bankroll-status'); if (bal) bal.textContent = b.balance ?? '—'; if (st) st.textContent = b.status || '—'; } catch {}
        try {
            const txs = await jget('/transactions'); const table = document.getElementById('transactions-table')?.querySelector('tbody'); if (!table) return;
            const list = Array.isArray(txs) ? txs : (txs.transactions || []);
            table.innerHTML = list.length ? '' : '<tr><td colspan="4">No economy transactions yet.</td></tr>';
            list.slice(-10).reverse().forEach(tx => { const tr = document.createElement('tr'); tr.innerHTML = `<td>${tx.type || tx.kind || ''}</td><td>${tx.amount ?? ''}</td><td class="mono">${String(tx.hash || tx.id || '').substring(0, 20)}</td><td>${tx.timestamp ? new Date(tx.timestamp).toLocaleString() : ''}</td>`; table.appendChild(tr); });
        } catch {}
    }

    async function economyAction(kind) {
        const amount = parseFloat(document.getElementById('economy-amount')?.value);
        const reason = document.getElementById('economy-reason')?.value.trim() || '';
        if (!amount || amount <= 0) return alert('Enter an amount greater than 0.');
        try { const r = await fetch('/' + kind, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ amount, reason }) }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || kind + ' failed'); showBox('economy-result', 'economy-json', d); updateEconomy(); } catch (e) { alert(e.message); }
    }
    document.getElementById('btn-mint')?.addEventListener('click', () => economyAction('mint'));
    document.getElementById('btn-burn')?.addEventListener('click', () => economyAction('burn'));

    document.getElementById('btn-add-torrent')?.addEventListener('click', async () => {
        const magnet = document.getElementById('magnet-input')?.value.trim();
        if (!magnet) return alert('Paste a magnet link first.');
        try { const r = await fetch('/add-torrent', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ magnet }) }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'Add failed'); showBox('torrent-action-result', 'torrent-action-json', d); updateStatus(); } catch (e) { alert(e.message); }
    });
    document.getElementById('btn-remove-torrent')?.addEventListener('click', async () => {
        const infoHash = document.getElementById('magnet-input')?.value.trim();
        if (!infoHash) return alert('Enter the 40-char InfoHash to remove.');
        try { const r = await fetch('/remove-torrent', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ infoHash }) }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'Remove failed'); showBox('torrent-action-result', 'torrent-action-json', d); updateStatus(); } catch (e) { alert(e.message); }
    });
    document.getElementById('btn-upload-torrent')?.addEventListener('click', async () => {
        const file = document.getElementById('upload-file')?.files?.[0];
        if (!file) return alert('Choose a file to upload & seed.');
        const fd = new FormData(); fd.append('file', file);
        try { const r = await fetch('/upload', { method: 'POST', body: fd }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'Upload failed'); showBox('torrent-action-result', 'torrent-action-json', d); updateStatus(); } catch (e) { alert(e.message); }
    });

    document.getElementById('btn-spora')?.addEventListener('click', async () => {
        const c = document.getElementById('spora-challenge')?.value.trim();
        if (!c) return alert('Enter an integer SPoRA challenge.');
        try { const r = await fetch('/spora/' + encodeURIComponent(c)); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'SPoRA failed'); showBox('operator-result', 'operator-json', d); } catch (e) { alert(e.message); }
    });
    document.getElementById('btn-fhe')?.addEventListener('click', async () => {
        const cipherText = document.getElementById('fhe-ciphertext')?.value.trim();
        if (!cipherText) return alert('Enter an FHE ciphertext.');
        try { const r = await fetch('/fhe-oracle', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ cipherText }) }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'FHE oracle failed'); showBox('operator-result', 'operator-json', d); } catch (e) { alert(e.message); }
    });

    // Lattice state panels (proxied from lattice-go: /proposals /nfts /swaps /anchors)
    function fillTable(id, rows, emptyText, cols) {
        const table = document.getElementById(id)?.querySelector('tbody');
        if (!table) return;
        table.innerHTML = rows.length ? '' : `<tr><td colspan="${cols}">${emptyText}</td></tr>`;
        rows.forEach(cells => { const tr = document.createElement('tr'); tr.innerHTML = cells; table.appendChild(tr); });
    }
    const short = s => String(s || '').length > 18 ? String(s).substring(0, 16) + '…' : String(s || '');
    const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    async function updateLatticeState() {
        try { const d = await jget('/proposals'); const list = d.proposals || []; fillTable('proposals-table', list.map(p => [`<td>${esc(p.title || short(p.id))}</td><td><span class="badge ${p.status === 'passed' ? 'success' : ''}">${p.status}</span></td><td>${Number(p.votesFor || 0).toFixed(1)} / ${Number(p.votesAgainst || 0).toFixed(1)}</td>`]), 'No proposals yet.', 3); } catch {}
        try { const d = await jget('/nfts'); const list = d.nfts || []; fillTable('nfts-table', list.map(n => [`<td>${esc(n.name || short(n.id))}</td><td class="mono">${esc(short(n.owner))}</td>`]), 'No NFTs minted yet.', 2); } catch {}
        try { const d = await jget('/swaps'); const list = d.swaps || []; fillTable('swaps-table', list.map(x => [`<td>${x.amount}</td><td><span class="badge ${x.status === 'CLAIMED' ? 'success' : ''}">${x.status}</span></td><td class="mono">${esc(short(x.recipient))}</td>`]), 'No swaps yet.', 3); } catch {}
        try { const d = await jget('/anchors'); const list = d.anchors || []; fillTable('anchors-table', list.map(a => [`<td class="mono">${esc(short(a.manifestId || a.id))}</td><td>${esc(a.type)}</td>`]), 'No anchored manifests yet.', 2); } catch {}
    }

    async function updatePeers() {
        try {
            const d = await jget('/peers');
            const diags = d.diagnostics || [];
            fillTable('peers-table', diags.map(p => [`<td class="mono">${esc(short(p.peer))}</td><td><span class="badge ${p.health === 'healthy' ? 'success' : 'danger'}">${esc(p.health || 'unknown')}</span></td><td>${esc(p.lastStatus || '—')}</td><td>${p.totalSyncSuccesses || 0}/${p.totalSyncFailures || 0}</td>`]), 'No peers registered. Single-node mode.', 4);
        } catch {}
    }
    document.getElementById('btn-add-peer')?.addEventListener('click', async () => {
        const addr = document.getElementById('peer-addr')?.value.trim();
        if (!addr) return alert('Enter a peer lattice URL, e.g. http://127.0.0.1:4002');
        try { const r = await fetch('/peers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ addr, sync: true }) }); const d = await r.json().catch(() => ({})); if (!r.ok) throw new Error(d.error || 'Add peer failed'); updatePeers(); updateLattice(); } catch (e) { alert(e.message); }
    });

    // Polling intervals
    setInterval(updateStatus, 2000);
    setInterval(updatePeers, 5000);
    setInterval(updateLatticeState, 5000);
    setInterval(updateServices, 5000);
    setInterval(updateEconomy, 5000);
    setInterval(updateBlobs, 4000);
    setInterval(updateAssets, 5000);
    setInterval(updateLattice, 3000);
    setInterval(refreshSubscriptions, 5000);

    // Initial triggers
    updatePeers();
    updateLatticeState();
    updateServices();
    updateEconomy();
    updateStatus();
    updateBlobs();
    updateAssets();
    updateLattice();
    refreshSubscriptions();
});
