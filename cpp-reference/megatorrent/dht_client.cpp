#include "dht_client.h"

#include <libtorrent/session.hpp>
#include <libtorrent/kademlia/item.hpp>
#include <libtorrent/sha1_hash.hpp>
#include <libtorrent/bencode.hpp>
#include <libtorrent/alert_types.hpp>

#include <QDebug>
#include <array>

namespace Megatorrent {

DHTClient::DHTClient(libtorrent::session *session, QObject *parent)
    : QObject(parent), m_session(session) {
    if (!m_session) {
        qWarning() << "DHTClient: Invalid session";
    }
}

// Helper: Hex String to SHA1 Hash (Truncated SHA256)
static libtorrent::sha1_hash toInfoHash(const QString &blobId) {
    QByteArray blobBytes = QByteArray::fromHex(blobId.toLatin1());
    if (blobBytes.size() < 20) return libtorrent::sha1_hash(); // Invalid

    std::string s(blobBytes.constData(), 20); // First 20 bytes of SHA256
    return libtorrent::sha1_hash(s);
}

void DHTClient::putManifest(const QByteArray &publicKey, const QByteArray &privateKey, const QByteArray &payload, int64_t sequence) {
    if (!m_session || publicKey.size() != 32 || privateKey.size() != 64) return;

    std::array<char, 32> pk;
    std::copy(publicKey.begin(), publicKey.end(), pk.begin());

    std::array<char, 64> sk;
    std::copy(privateKey.begin(), privateKey.end(), sk.begin());

    libtorrent::entry e = libtorrent::bdecode(payload.begin(), payload.end());

    m_session->dht_put_item(pk, [e, sequence](libtorrent::entry& item, std::array<char,64>&, std::int64_t& seq, std::string const&) {
        item = e;
        seq = sequence;
        return true;
    });
}

void DHTClient::relaySignedPut(const QByteArray &publicKey, int64_t sequence, const QByteArray &value, const QByteArray &signature) {
    if (!m_session) return;

    // libtorrent::dht_put_item high-level API assumes we have the private key to sign.
    // To relay an already-signed item, we effectively need to act as a DHT node receiving a 'put' message.
    // This is not exposed in the standard `lt::session` API.
    //
    // Workaround/Stub:
    // In a real implementation we would modify libtorrent to accept pre-signed items
    // or use `dht_direct_request` if available to inject it.

    qDebug() << "DHTClient: [Stub] Relaying signed put for" << publicKey.toHex() << "Seq:" << sequence;
}

void DHTClient::getManifest(const QByteArray &publicKey) {
    if (!m_session || publicKey.size() != 32) return;

    std::array<char, 32> pk;
    std::copy(publicKey.begin(), publicKey.end(), pk.begin());

    m_session->dht_get_item(pk);
}

void DHTClient::announceBlob(const QString &blobId, int port) {
    if (!m_session) return;
    libtorrent::sha1_hash ih = toInfoHash(blobId);
    if (ih.is_all_zeros()) return;

    m_session->dht_announce(ih, port, 0);
}

void DHTClient::findBlobPeers(const QString &blobId) {
    if (!m_session) return;
    libtorrent::sha1_hash ih = toInfoHash(blobId);
    if (ih.is_all_zeros()) return;

    m_session->dht_get_peers(ih);
}

void DHTClient::handleDhtAlert(const libtorrent::alert *alert) {
    switch (alert->type()) {
        case libtorrent::dht_mutable_item_alert::alert_type: {
            const auto* a = static_cast<const libtorrent::dht_mutable_item_alert*>(alert);

            Manifest m;
            m.publicKey = QByteArray(a->key.data(), 32);
            m.signature = QByteArray(a->signature.data(), 64);
            m.sequence = a->seq;

            // Extract payload from entry
            std::string buf;
            libtorrent::bencode(std::back_inserter(buf), a->item);
            m.payload = QByteArray::fromStdString(buf);

            emit manifestFound(m);
            break;
        }
        case libtorrent::dht_get_peers_reply_alert::alert_type: {
            const auto* a = static_cast<const libtorrent::dht_get_peers_reply_alert*>(alert);
            QString blobId = QString(QByteArray(a->info_hash.data(), 20).toHex());

            QVector<QString> endpoints;
            for (const auto& tcp_endpoint : a->peers) {
                QString ep = QString::fromStdString(tcp_endpoint.address().to_string()) + ":" + QString::number(tcp_endpoint.port());
                endpoints.append(ep);
            }
            emit peersFound(blobId, endpoints);
            break;
        }
        default:
            break;
    }
}

}
