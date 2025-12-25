'use strict'

/* global $$, $, Element, fetch */

const RPC_URL = 'http://localhost:3000/api/rpc'

window.addEvent('domready', function () {
  $('megatorrentTabLink').addEvent('click', function () {
    loadMegatorrentTab()
  })
})

function loadMegatorrentTab () {
  $$('.tab-content').setStyle('display', 'none')
  $$('.toolbarTabs li').removeClass('selected')
  $('megatorrentTabLink').addClass('selected')

  let container = $('megatorrentContainer')
  if (!container) {
    container = new Element('div', {
      id: 'megatorrentContainer',
      class: 'tab-content',
      styles: { padding: '20px', color: '#ccc' }
    }).inject($('pageWrapper'))

    renderMegatorrentUI(container)
  }
  container.setStyle('display', 'block')
  refreshSubscriptions()
}

function renderMegatorrentUI (container) {
  container.innerHTML = `
        <h2>Megatorrent Subscriptions</h2>
        <div style="margin-bottom: 20px;">
            <input type="text" id="megaUriInput" placeholder="megatorrent://<pubkey>" style="width: 300px; padding: 5px;">
            <button id="megaSubscribeBtn" style="padding: 5px 10px;">Subscribe</button>
            <span id="megaStatus" style="margin-left: 20px;"></span>
        </div>

        <table class="dynamicTable" style="width: 100%;">
            <thead>
                <tr>
                    <th>URI</th>
                    <th>Status</th>
                    <th>Last Sequence</th>
                </tr>
            </thead>
            <tbody id="megaSubsList"></tbody>
        </table>
    `

  document.getElementById('megaSubscribeBtn').addEventListener('click', function () {
    const uri = document.getElementById('megaUriInput').value
    if (uri) rpcCall('addSubscription', { uri }).then(refreshSubscriptions)
  })
}

async function refreshSubscriptions () {
  try {
    const subs = await rpcCall('getSubscriptions', {})
    const status = await rpcCall('getStatus', {})

    document.getElementById('megaStatus').innerText = `Peers: ${status.peers} | Blobs: ${status.heldBlobs}`

    const list = document.getElementById('megaSubsList')
    list.innerHTML = ''

    if (subs.subscriptions.length === 0) {
      list.innerHTML = '<tr><td colspan="3" style="text-align:center">No subscriptions</td></tr>'
    } else {
      subs.subscriptions.forEach(sub => {
        const row = new Element('tr')
        row.innerHTML = `
                <td>${sub.uri.substring(0, 50)}...</td>
                <td>${sub.status}</td>
                <td>${sub.lastSequence}</td>
            `
        list.appendChild(row)
      })
    }
  } catch (e) {
    document.getElementById('megaStatus').innerText = 'Daemon Offline'
  }
}

async function rpcCall (method, params) {
  const res = await fetch(RPC_URL, {
    method: 'POST',
    body: JSON.stringify({ method, params })
  })
  const json = await res.json()
  return json.result
}
