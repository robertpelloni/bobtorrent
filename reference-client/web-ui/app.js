/* global localStorage, location, alert, fetch, document, window */

document.addEventListener('DOMContentLoaded', () => {
  let targetNode = localStorage.getItem('targetNode') || 'local'
  const nodeSelector = document.getElementById('node-selector')
  if (nodeSelector) {
    nodeSelector.value = targetNode
    nodeSelector.addEventListener('change', () => {
      targetNode = nodeSelector.value
      localStorage.setItem('targetNode', targetNode)
      location.reload() // Simple reload to refresh all data
    })
  }

  const apiFetch = async (url, options = {}) => {
    if (targetNode !== 'local') {
      options.headers = { ...options.headers, 'x-target-node': targetNode }
    }
    return fetch(url, options)
  }

  // Tabs
  const tabs = document.querySelectorAll('.tab-btn')
  const contents = document.querySelectorAll('.tab-content')

  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      tabs.forEach(t => t.classList.remove('active'))
      contents.forEach(c => c.classList.remove('active'))

      tab.classList.add('active')
      document.getElementById(tab.dataset.tab).classList.add('active')
    })
  })

  // Identity
  const btnGenKey = document.getElementById('btn-generate-key')
  const inputPub = document.getElementById('id-pub')
  const inputPriv = document.getElementById('id-priv')
  const pubStatus = document.getElementById('pub-identity-status')

  let currentIdentity = null

  btnGenKey.addEventListener('click', async () => {
    const res = await apiFetch('/api/key/generate', { method: 'POST' })
    const data = await res.json()
    currentIdentity = data
    inputPub.value = data.publicKey
    inputPriv.value = data.secretKey
    pubStatus.textContent = 'Key Loaded'
    pubStatus.style.color = '#4caf50'
    document.getElementById('btn-publish').disabled = false
    document.getElementById('btn-save-key').disabled = false
  })

  // Publish
  const btnIngest = document.getElementById('btn-ingest')
  const inputPath = document.getElementById('ingest-path')
  const ingestResult = document.getElementById('ingest-result')
  const ingestJson = document.getElementById('ingest-json')

    // Advanced Ingest UI Logic
    const ingestAdvToggle = document.getElementById('ingest-adv-toggle');
    const ingestAdvanced = document.getElementById('ingest-advanced');
    const ingestStrategy = document.getElementById('ingest-strategy');
    const ingestEcSettings = document.getElementById('ingest-ec-settings');
    const inputDataShards = document.getElementById('ingest-data-shards');
    const inputParityShards = document.getElementById('ingest-parity-shards');

    ingestAdvToggle.addEventListener('change', () => {
        if (ingestAdvToggle.checked) {
            ingestAdvanced.classList.remove('hidden');
        } else {
            ingestAdvanced.classList.add('hidden');
        }
    });

    ingestStrategy.addEventListener('change', () => {
        if (ingestStrategy.value === 'erasure') {
            ingestEcSettings.classList.remove('hidden');
        } else {
            ingestEcSettings.classList.add('hidden');
        }
    });

  let currentFileEntry = null

  btnIngest.addEventListener('click', async () => {
    const path = inputPath.value
    if (!path) return alert('Please enter a file path')

        // Build Options
        let options = { enableErasure: false };
        if (ingestAdvToggle.checked) {
            if (ingestStrategy.value === 'erasure') {
                const data = parseInt(inputDataShards.value);
                const parity = parseInt(inputParityShards.value);

                if (data + parity > 255) return alert('Total shards cannot exceed 255');
                if (data < 1 || parity < 1) return alert('Shards must be at least 1');

                options = {
                    enableErasure: true,
                    dataShards: data,
                    parityShards: parity
                };
            }
        }

    btnIngest.textContent = 'Ingesting...'
    btnIngest.disabled = true

    try {
      const res = await apiFetch('/api/ingest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    filePath: path,
                    options: options
                })
      })

      if (!res.ok) throw new Error((await res.json()).error || 'Ingest failed')

      const data = await res.json()
      currentFileEntry = data.fileEntry
      ingestJson.textContent = JSON.stringify(data.fileEntry, null, 2)
      ingestResult.classList.remove('hidden')
    } catch (err) {
      alert(err.message)
    } finally {
      btnIngest.textContent = 'Ingest File'
      btnIngest.disabled = false
    }
  })

  const btnPublish = document.getElementById('btn-publish')
  const publishResult = document.getElementById('publish-result')
  const publishJson = document.getElementById('publish-json')

  btnPublish.addEventListener('click', async () => {
    if (!currentFileEntry || !currentIdentity) return alert('Missing file or identity')

    btnPublish.textContent = 'Publishing...'
    btnPublish.disabled = true

    try {
      const res = await apiFetch('/api/publish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          fileEntry: currentFileEntry,
          identity: currentIdentity
        })
      })

      if (!res.ok) throw new Error((await res.json()).error || 'Publish failed')

      const data = await res.json()
      publishJson.textContent = JSON.stringify(data.manifest, null, 2)
      publishResult.classList.remove('hidden')
    } catch (err) {
      alert(err.message)
    } finally {
      btnPublish.textContent = 'Publish Manifest'
      btnPublish.disabled = false
    }
  })

  // Discovery
  const btnBrowse = document.getElementById('btn-browse')
  const inputDiscoveryPath = document.getElementById('discovery-path')
  const discoveryBox = document.getElementById('discovery-box')
  const discoverySubtopics = document.getElementById('discovery-subtopics')
  const discoveryPublishers = document.getElementById('discovery-publishers')

  btnBrowse.addEventListener('click', async () => {
    const topic = inputDiscoveryPath.value
    btnBrowse.disabled = true
    btnBrowse.textContent = 'Searching...'

    try {
      const res = await apiFetch(`/api/channels/browse?topic=${encodeURIComponent(topic)}`)
      if (!res.ok) throw new Error('Browse failed')
      const result = await res.json()

      discoverySubtopics.innerHTML = ''
      discoveryPublishers.innerHTML = ''

      if (result.subtopics.length === 0 && result.publishers.length === 0) {
        discoverySubtopics.innerHTML = '<li>No results found.</li>'
      }

      result.subtopics.forEach(st => {
        const li = document.createElement('li')
        li.innerHTML = `<a href="#" onclick="document.getElementById('discovery-path').value='${topic ? topic + '/' : ''}${st}'; document.getElementById('btn-browse').click(); return false;">📁 ${st}</a>`
        discoverySubtopics.appendChild(li)
      })

      result.publishers.forEach(pub => {
        const li = document.createElement('li')
        li.innerHTML = `👤 ${pub.name || 'Unknown'} <small>(${pub.pk.substring(0, 8)}...)</small> <button class="secondary-btn" style="padding: 2px 6px; font-size: 0.8rem;" onclick="document.getElementById('sub-key').value='${pub.pk}'; document.querySelector('[data-tab=subscribe]').click();">Sub</button>`
        discoveryPublishers.appendChild(li)
      })

      discoveryBox.classList.remove('hidden')
    } catch (e) {
      alert(e.message)
    } finally {
      btnBrowse.disabled = false
      btnBrowse.textContent = 'Browse'
    }
  })

  // Subscribe
  const btnSubscribe = document.getElementById('btn-subscribe')
  const inputSubKey = document.getElementById('sub-key')
  const subsTable = document.getElementById('subs-table').querySelector('tbody')

  btnSubscribe.addEventListener('click', async () => {
    const key = inputSubKey.value
    if (!key) return alert('Enter public key')

    try {
      const res = await apiFetch('/api/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ publicKey: key })
      })

      if (!res.ok) throw new Error('Subscribe failed')

      inputSubKey.value = ''
      refreshSubscriptions()
    } catch (err) {
      alert(err.message)
    }
  })

  async function refreshSubscriptions () {
    try {
      const res = await apiFetch('/api/subscriptions')
      const subs = await res.json()

      subsTable.innerHTML = subs.length ? '' : '<tr><td colspan="4">No subscriptions yet.</td></tr>'

      subs.forEach(sub => {
        const tr = document.createElement('tr')
        tr.innerHTML = `
                    <td>${sub.topicPath.substring(0, 16)}...</td>
                    <td>${sub.lastSequence || '-'}</td>
                    <td><span class="badge" style="background:#28a745">Active</span></td>
                    <td><button class="secondary-btn" style="padding: 2px 5px; font-size: 0.8rem">Details</button></td>
                `
        subsTable.appendChild(tr)
      })
    } catch (e) {}
  }

  // Wallet
  const btnAirdrop = document.getElementById('btn-airdrop')
  if (btnAirdrop) {
    btnAirdrop.addEventListener('click', async () => {
      btnAirdrop.disabled = true
      btnAirdrop.textContent = 'Requesting...'
      try {
        const res = await apiFetch('/api/wallet/airdrop', { method: 'POST' })
        if (!res.ok) throw new Error('Airdrop failed')
        await updateWallet()
      } catch (err) {
        alert(err.message)
      } finally {
        btnAirdrop.disabled = false
        btnAirdrop.textContent = 'Request Airdrop'
      }
    })
  }

  async function updateWallet () {
    try {
      const res = await apiFetch('/api/wallet')
      const data = await res.json()

      document.getElementById('wallet-address').value = data.address
      document.getElementById('wallet-balance').textContent = data.balance + ' SOL (Devnet)'
      document.getElementById('wallet-pending').textContent = data.pending + ' BOB'

      const table = document.getElementById('wallet-txs').querySelector('tbody')
      table.innerHTML = data.transactions.length ? '' : '<tr><td colspan="4">No transactions.</td></tr>'

      data.transactions.forEach(tx => {
        const tr = document.createElement('tr')
        tr.innerHTML = `<td>${tx.id}</td><td>${tx.type}</td><td>${tx.amount}</td><td>${tx.time}</td>`
        table.appendChild(tr)
      })
    } catch (e) {}
  }

  // Files Table
  async function updateFiles () {
    try {
      const res = await apiFetch('/api/files')
      const files = await res.json()
      const table = document.getElementById('files-table').querySelector('tbody')

      table.innerHTML = files.length ? '' : '<tr><td colspan="4">No files ingested.</td></tr>'

      files.forEach(f => {
        const tr = document.createElement('tr')

        let action = ''
        const ext = f.name.split('.').pop().toLowerCase()
        if (['mp4', 'webm', 'mp3', 'mkv'].includes(ext)) {
          action = `<button class="secondary-btn" style="padding: 2px 8px; margin-left: 10px;" onclick="playFile('${f.id}', '${f.name}')">▶ Play</button>`
        }

                // Add Inspect button
                action += `<button class="secondary-btn" style="padding: 2px 8px; margin-left: 5px;" onclick="inspectFile('${f.id}', '${f.name}')">🔍</button>`;

        tr.innerHTML = `
                    <td>${f.name}</td>
                    <td>${(f.size / 1024 / 1024).toFixed(2)} MB</td>
                    <td>${f.status}</td>
                    <td><progress value="${f.progress}" max="100"></progress> ${f.progress}% ${action}</td>
                `
        table.appendChild(tr)
      })
    } catch (e) {}
  }

    // File Inspector
    window.inspectFile = async (id, name) => {
        const container = document.getElementById('inspector-container');
        const title = document.getElementById('inspector-title');
        const grid = document.getElementById('insp-grid');

        title.textContent = `Health: ${name}`;
        grid.innerHTML = '<div style="grid-column: 1/-1; text-align: center;">Loading...</div>';
        container.classList.remove('hidden');
        container.style.display = 'flex';

        try {
            const res = await apiFetch(`/api/files/${id}/health`);
            const data = await res.json();

            document.getElementById('insp-status').textContent = data.status;
            document.getElementById('insp-status').style.color = data.status === 'Healthy' ? '#4caf50' : '#f44336';
            document.getElementById('insp-chunks').textContent = `${data.healthyChunks} / ${data.totalChunks}`;

            if (data.erasure) {
                document.getElementById('insp-mode').textContent = 'Erasure Coding';
                document.getElementById('insp-config').textContent = `${data.erasure.dataShards} Data + ${data.erasure.parityShards} Parity`;
            } else {
                document.getElementById('insp-mode').textContent = 'Simple Replication';
                document.getElementById('insp-config').textContent = '1x';
            }

            grid.innerHTML = '';

            data.chunks.forEach(chunk => {
                const cell = document.createElement('div');
                cell.style.background = '#333';
                cell.style.border = '1px solid #444';
                cell.style.height = '40px';
                cell.style.display = 'flex';
                cell.title = `Chunk ${chunk.index}: ${chunk.status}`;

                if (data.erasure) {
                    // Render shards
                    if (chunk.shards && chunk.shards.length) {
                        chunk.shards.forEach(shard => {
                            const bar = document.createElement('div');
                            bar.style.flex = '1';
                            bar.style.margin = '1px';
                            // Data shards vs Parity shards logic
                            const isData = shard.index < data.erasure.dataShards;
                            bar.style.background = shard.present
                                ? (isData ? '#4caf50' : '#2196f3')
                                : '#f44336';
                            cell.appendChild(bar);
                        });
                    } else {
                        // Should have shards but doesn't (error state)
                        cell.style.background = '#f44336';
                        cell.textContent = '!';
                        cell.style.justifyContent = 'center';
                        cell.style.alignItems = 'center';
                    }
                } else {
                    // Simple replication
                    cell.style.background = chunk.status === 'Healthy' ? '#4caf50' : '#f44336';
                }

                grid.appendChild(cell);
            });

        } catch (e) {
            grid.innerHTML = `<div style="color:red">Error: ${e.message}</div>`;
        }
    };

    window.closeInspector = () => {
        const container = document.getElementById('inspector-container');
        container.classList.add('hidden');
        container.style.display = 'none';
    };

  // Video Player
  window.playFile = (id, name) => {
    const container = document.getElementById('video-container')
    const player = document.getElementById('video-player')
    const title = document.getElementById('video-title')

    container.classList.remove('hidden')
    container.style.display = 'flex' // Override hidden class logic if needed, or rely on CSS
    title.textContent = name
    player.src = `/api/stream/${id}`
    player.play().catch(e => console.log('Auto-play blocked:', e))
  }

  window.closeVideo = () => {
    const container = document.getElementById('video-container')
    const player = document.getElementById('video-player')

    container.classList.add('hidden')
    container.style.display = 'none'
    player.pause()
    player.src = ''
  }

  // Dashboard Status
  async function updateStatus () {
    try {
      const res = await apiFetch('/api/status')
      const data = await res.json()

      if (data.version) {
        document.getElementById('app-version').textContent = `v${data.version}`
      }

      document.getElementById('dash-blobs').textContent = data.storage.blobs
      document.getElementById('dash-size').textContent = (data.storage.size / 1024 / 1024).toFixed(2) + ' MB'
      document.getElementById('dash-max').textContent = (data.storage.max / 1024 / 1024 / 1024).toFixed(2) + ' GB'
      document.getElementById('dash-util').textContent = (data.storage.utilization * 100).toFixed(1) + '%'
      document.getElementById('dash-dht').textContent = data.dht
      document.getElementById('dash-subs').textContent = data.subscriptions

      document.getElementById('dht-status').querySelector('.value').textContent = data.dht
      document.getElementById('network-status').querySelector('.value').textContent = data.network

      // Update Network Tab Details
      updateNetworkTab(data)
    } catch (e) {}
  }

  function updateNetworkTab (data) {
    // Storage Engine Details
    if (data.storageDetails) {
      document.getElementById('net-iso-size').textContent = data.storageDetails.isoSize
      document.getElementById('net-files-ingested').textContent = data.storageDetails.totalFilesIngested
      document.getElementById('net-bytes-ingested').textContent = (data.storageDetails.totalBytesIngested / 1024 / 1024).toFixed(2) + ' MB'

      if (data.storageDetails.erasure) {
        document.getElementById('net-ec-status').textContent = 'Enabled'
        document.getElementById('net-ec-status').style.color = '#4caf50'
        document.getElementById('net-ec-config').textContent = `${data.storageDetails.erasure.dataShards} Data + ${data.storageDetails.erasure.parityShards} Parity`
        document.getElementById('net-ec-shards').textContent = data.storageDetails.erasure.totalShards
      } else {
        document.getElementById('net-ec-status').textContent = 'Disabled'
        document.getElementById('net-ec-status').style.color = '#f44336'
        document.getElementById('net-ec-config').textContent = 'Standard Replication'
        document.getElementById('net-ec-shards').textContent = 'N/A'
      }
    }

    // Transports Table
    const table = document.getElementById('transports-table').querySelector('tbody')
    if (data.networkDetails && data.networkDetails.transports) {
      table.innerHTML = ''
      Object.entries(data.networkDetails.transports).forEach(([type, t]) => {
        const tr = document.createElement('tr')
        const statusColor = t.status === 'Running' ? '#4caf50' : '#f44336'

        tr.innerHTML = `
                    <td style="font-weight: bold;">${type}</td>
                    <td><span class="badge" style="background:${statusColor}">${t.status}</span></td>
                    <td style="font-family: monospace; font-size: 0.9em;">${t.address || '-'}</td>
                    <td>${t.connectionsIn} In / ${t.connectionsOut} Out</td>
                    <td>${(t.bytesReceived / 1024).toFixed(1)} KB Rx / ${(t.bytesSent / 1024).toFixed(1)} KB Tx</td>
                    <td>${t.errors}</td>
                `
        table.appendChild(tr)
      })
    } else {
      table.innerHTML = '<tr><td colspan="6">No transport details available (Not supported by this node type).</td></tr>'
    }
  }

  // Blobs Table
  async function updateBlobs () {
    try {
      const res = await apiFetch('/api/blobs')
      const blobs = await res.json()
      const table = document.getElementById('blobs-table').querySelector('tbody')

      table.innerHTML = blobs.length ? '' : '<tr><td colspan="3">No blobs found.</td></tr>'

      blobs.forEach(blob => {
        const tr = document.createElement('tr')
        tr.innerHTML = `
                    <td>${blob.blobId.substring(0, 32)}...</td>
                    <td>${blob.size} bytes</td>
                    <td>${new Date(blob.addedAt).toLocaleString()}</td>
                `
        table.appendChild(tr)
      })
    } catch (e) {}
  }

  // Polling
  setInterval(updateStatus, 2000)
  setInterval(refreshSubscriptions, 5000)
  setInterval(updateBlobs, 5000)
  setInterval(updateFiles, 5000)
  setInterval(updateWallet, 10000)

  // Initial Load
  updateStatus()
  refreshSubscriptions()
  updateBlobs()
  updateFiles()
  updateWallet()
})
