#include "dht_client.h"
#include <QDebug>

namespace Megatorrent {

DHTClient::DHTClient(QObject *parent) : QObject(parent) {
    qDebug() << "Megatorrent: DHT Client initialized (Stub)";
}

void DHTClient::putManifest(const QByteArray &publicKey, const QByteArray &privateKey, const Manifest &manifest) {
    qDebug() << "Megatorrent: Putting manifest for" << publicKey.toHex();
    // 1. Serialize Manifest to JSON/Bencode
    // 2. Sign with Ed25519 (privateKey)
    // 3. Call libtorrent::dht_put_item
}

void DHTClient::getManifest(const QByteArray &publicKey) {
    qDebug() << "Megatorrent: Getting manifest for" << publicKey.toHex();
    // 1. Call libtorrent::dht_get_item(publicKey)
    // 2. On callback: verify signature
    // 3. emit manifestFound()
}

void DHTClient::announceBlob(const QString &blobId, int port) {
    qDebug() << "Megatorrent: Announcing blob" << blobId << "on port" << port;
    // 1. Convert blobId (hex) to sha1_hash/sha256_hash
    // 2. Call libtorrent::announce_peer
}

void DHTClient::findBlobPeers(const QString &blobId) {
    qDebug() << "Megatorrent: Finding peers for blob" << blobId;
    // 1. Call libtorrent::dht_get_peers
    // 2. On callback: emit peersFound()
}

}
