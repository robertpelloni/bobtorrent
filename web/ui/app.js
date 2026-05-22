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

    // Messenger / Chat
    let messengerWs = null;
    let activeTopic = 'bobtorrent-global-gossip';
    const chatTopicHistory = {}; // topic -> [messages]
    let nodePeerId = 'unknown';

    function initMessenger() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        messengerWs = new WebSocket(`${protocol}//${window.location.host}/ws-messenger`);

        messengerWs.onopen = () => {
            console.log('Messenger connected');
            // Request history for global topic
            messengerWs.send(JSON.stringify({ type: 'FETCH_HISTORY', topic: 'bobtorrent-global-gossip' }));
        };

        messengerWs.onmessage = (event) => {
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
        };

        messengerWs.onclose = () => {
            console.log('Messenger disconnected, retrying...');
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
        // Hydrate history
        chatTopicHistory[msg.topic] = msg.messages.map(m => ({
            topic: m.topic,
            from: m.sender,
            data: m.data,
            timestamp: m.timestamp
        }));

        if (msg.topic === activeTopic) {
            refreshChatMessages();
        }
    }

    function refreshChatMessages() {
        const container = document.getElementById('chat-messages');
        container.innerHTML = '';
        const history = chatTopicHistory[activeTopic] || [];
        history.forEach(renderMessage);
    }

    function renderMessage(msg) {
        const container = document.getElementById('chat-messages');
        const div = document.createElement('div');
        div.className = 'chat-message';
        // Check if message is from self (stub: comparison with nodePeerId later)
        if (msg.from === nodePeerId) div.classList.add('self');

        const sender = document.createElement('div');
        sender.className = 'sender';
        sender.textContent = msg.from.substring(0, 12) + '...';

        const text = document.createElement('div');
        text.className = 'text';
        text.textContent = msg.data;

        div.appendChild(sender);
        div.appendChild(text);
        container.appendChild(div);
        container.scrollTop = container.scrollHeight;
    }

    function addNotice(text) {
        if (activeTopic === 'bobtorrent-global-gossip') { // Notice logic simplified
            const container = document.getElementById('chat-messages');
            const div = document.createElement('div');
            div.className = 'chat-notice';
            div.textContent = text;
            container.appendChild(div);
            container.scrollTop = container.scrollHeight;
        }
    }

    // Chat UI Events
    const btnSendChat = document.getElementById('btn-send-chat');
    const inputChat = document.getElementById('chat-input');

    btnSendChat.addEventListener('click', () => {
        const text = inputChat.value;
        if (!text) return;

        messengerWs.send(JSON.stringify({
            type: 'PUBLISH',
            topic: activeTopic,
            data: text
        }));

        // Optimistically add to UI if it's gossip (backend doesn't echo)
        const myMsg = { from: nodePeerId, data: text, topic: activeTopic };
        handleIncomingGossip(myMsg);

        inputChat.value = '';
    });

    inputChat.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') btnSendChat.click();
    });

    const btnJoinTopic = document.getElementById('btn-join-topic');
    const inputNewTopic = document.getElementById('new-topic-name');
    const topicList = document.getElementById('topic-list');

    btnJoinTopic.addEventListener('click', () => {
        const name = inputNewTopic.value.trim();
        if (!name) return;

        messengerWs.send(JSON.stringify({ type: 'JOIN_TOPIC', topic: name }));

        // Add to sidebar if not already there
        if (!document.querySelector(`li[data-topic="${name}"]`)) {
            const li = document.createElement('li');
            li.dataset.topic = name;
            li.textContent = `# ${name}`;
            li.addEventListener('click', () => switchTopic(name));
            topicList.appendChild(li);
        }

        inputNewTopic.value = '';
    });

    function switchTopic(name) {
        activeTopic = name;
        document.querySelectorAll('.topic-list li').forEach(li => {
            li.classList.toggle('active', li.dataset.topic === name);
        });
        refreshChatMessages();
    }

    document.querySelectorAll('.topic-list li').forEach(li => {
        li.addEventListener('click', () => switchTopic(li.dataset.topic));
    });

    initMessenger();

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
                    <td>${sub.publicKey.substring(0, 16)}...</td>
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
            const res = await fetch('/stats');
            const data = await res.json();

            document.getElementById('dash-blobs').textContent = data.storage.blobs;
            document.getElementById('dash-size').textContent = (data.storage.size / 1024 / 1024).toFixed(2) + ' MB';
            document.getElementById('dash-max').textContent = (data.storage.max / 1024 / 1024 / 1024).toFixed(2) + ' GB';
            document.getElementById('dash-util').textContent = (data.storage.utilization * 100).toFixed(1) + '%';
            document.getElementById('dash-dht').textContent = data.dht;
            document.getElementById('dash-subs').textContent = data.subscriptions;

            document.getElementById('dht-status').querySelector('.value').textContent = data.dht;
            document.getElementById('network-status').querySelector('.value').textContent = data.status || "online";

            if (nodePeerId === 'unknown' && data.address) {
                // Using wallet address as a stand-in for self identification in chat for now
                nodePeerId = data.address;
            }
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
