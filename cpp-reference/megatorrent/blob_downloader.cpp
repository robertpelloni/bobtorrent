#include "blob_downloader.h"
#include <QDebug>
#include <QCryptographicHash>

namespace Megatorrent {

BlobDownloader::BlobDownloader(QObject *parent) : QObject(parent) {}

BlobDownloader::~BlobDownloader() {
    for (auto &dl : m_downloads) {
        if (dl.socket) {
            dl.socket->deleteLater();
        }
    }
}

void BlobDownloader::queueBlob(const QString &blobId, qint64 size, const QByteArray &key, const QByteArray &iv, const QString &savePath) {
    if (m_downloads.contains(blobId)) return;

    BlobRequest req;
    req.blobId = blobId;
    req.size = size;
    req.key = key;
    req.iv = iv;
    req.savePath = savePath;

    ActiveDownload dl;
    dl.request = req;

    m_downloads.insert(blobId, dl);
    m_queue.append(blobId);

    // Initial check for peers
    emit peersNeeded(blobId);

    startNextDownload();
}

void BlobDownloader::addPeers(const QString &blobId, const QVector<QString> &endpoints) {
    if (!m_downloads.contains(blobId)) return;

    ActiveDownload &dl = m_downloads[blobId];
    bool newPeers = false;
    for (const QString &ep : endpoints) {
        if (!dl.triedPeers.contains(ep) && !dl.peers.contains(ep)) {
            dl.peers.append(ep);
            newPeers = true;
        }
    }

    if (newPeers && !dl.active) {
        startNextDownload();
    }
}

void BlobDownloader::startNextDownload() {
    if (m_currentActive >= m_maxConcurrent) return;

    for (const QString &blobId : m_queue) {
        ActiveDownload &dl = m_downloads[blobId];
        if (!dl.active && !dl.peers.isEmpty()) {
            tryNextPeer(blobId);
            if (m_currentActive >= m_maxConcurrent) break;
        }
    }
}

void BlobDownloader::tryNextPeer(const QString &blobId) {
    if (!m_downloads.contains(blobId)) return;
    ActiveDownload &dl = m_downloads[blobId];

    if (dl.peers.isEmpty()) {
        dl.active = false;
        m_currentActive--;
        emit peersNeeded(blobId);
        return;
    }

    QString peer = dl.peers.takeFirst();
    dl.triedPeers.insert(peer);

    // Parse IP:Port
    QStringList parts = peer.split(":");
    if (parts.size() != 2) {
        tryNextPeer(blobId);
        return;
    }

    QString host = parts[0];
    quint16 port = parts[1].toUShort();

    if (dl.socket) {
        dl.socket->deleteLater();
    }
    dl.socket = new SecureSocket(this);
    dl.socket->setProperty("blobId", blobId); // Tag socket

    connect(dl.socket, &SecureSocket::connected, this, &BlobDownloader::onSocketConnected);
    connect(dl.socket, &SecureSocket::disconnected, this, &BlobDownloader::onSocketDisconnected);
    connect(dl.socket, &SecureSocket::errorOccurred, this, &BlobDownloader::onSocketError);
    connect(dl.socket, &SecureSocket::messageReceived, this, &BlobDownloader::onMessageReceived);

    qDebug() << "BlobDownloader: Connecting to" << peer << "for blob" << blobId;
    dl.socket->connectToHost(host, port);

    dl.active = true;
    m_currentActive++;
}

void BlobDownloader::onSocketConnected() {
    SecureSocket *socket = qobject_cast<SecureSocket*>(sender());
    if (!socket) return;
    QString blobId = socket->property("blobId").toString();

    // Send Request: [MSG_REQUEST][BlobID (Hex)]
    QByteArray payload = blobId.toLatin1();
    socket->sendMessage(Protocol::MSG_REQUEST, payload);
}

void BlobDownloader::onMessageReceived(quint8 type, const QByteArray &payload) {
    SecureSocket *socket = qobject_cast<SecureSocket*>(sender());
    if (!socket) return;
    QString blobId = socket->property("blobId").toString();

    if (type == Protocol::MSG_DATA) {
        // payload is the blob data
        // Verify Hash
        QByteArray hash = QCryptographicHash::hash(payload, QCryptographicHash::Sha256);
        if (hash.toHex() != blobId.toLatin1()) {
            qWarning() << "BlobDownloader: Hash mismatch for" << blobId;
            socket->close(); // Triggers disconnected -> next peer
            return;
        }

        // Save to disk
        ActiveDownload &dl = m_downloads[blobId];

        // Decrypt (Stub - in real impl use dl.request.key/iv)
        // For reference implementation, we assume payload is cleartext or we write encrypted?
        // Node.js client writes ENCRYPTED blobs to disk (obfuscated storage).
        // So we write directly.

        QFile file(dl.request.savePath);
        if (file.open(QIODevice::WriteOnly)) {
            file.write(payload);
            file.close();
            emit blobFinished(blobId);

            // Cleanup
            m_downloads.remove(blobId);
            m_queue.removeAll(blobId);
            m_currentActive--;
            socket->deleteLater();
            startNextDownload();
        } else {
            qWarning() << "BlobDownloader: Failed to write file" << dl.request.savePath;
            socket->close();
        }
    } else if (type == Protocol::MSG_ERROR) {
        qWarning() << "BlobDownloader: Peer returned error for" << blobId;
        socket->close();
    }
}

void BlobDownloader::onSocketDisconnected() {
    SecureSocket *socket = qobject_cast<SecureSocket*>(sender());
    if (!socket) return;
    QString blobId = socket->property("blobId").toString();

    if (m_downloads.contains(blobId)) {
        // Try next peer
        ActiveDownload &dl = m_downloads[blobId];
        dl.active = false;
        dl.socket = nullptr;
        m_currentActive--;

        socket->deleteLater();
        tryNextPeer(blobId);
    }
}

void BlobDownloader::onSocketError(const QString &msg) {
    qWarning() << "BlobDownloader: Socket Error:" << msg;
    // handled by disconnected
}

}
