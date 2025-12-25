#ifndef MEGATORRENT_DHT_CLIENT_H
#define MEGATORRENT_DHT_CLIENT_H

#include <QObject>
#include <QByteArray>
#include <QString>
#include <QVector>
#include <QMap>

// Forward declarations for libtorrent
namespace libtorrent {
    class session;
    class alert;
    struct sha1_hash;
}

namespace Megatorrent {

// Data Structures
struct Manifest {
    QByteArray publicKey;
    int64_t sequence;
    QByteArray signature;
    QByteArray payload; // Raw JSON/Bencode
};

class DHTClient : public QObject {
    Q_OBJECT
public:
    explicit DHTClient(libtorrent::session *session, QObject *parent = nullptr);

    // BEP 44: Put Signed Manifest (Author Mode)
    // Signs the manifest locally using privateKey
    void putManifest(const QByteArray &publicKey, const QByteArray &privateKey, const QByteArray &payload, int64_t sequence);

    // BEP 44: Relay Signed Put (Gateway Mode)
    // Puts a manifest that is ALREADY signed by the author (received via secure transport)
    void relaySignedPut(const QByteArray &publicKey, int64_t sequence, const QByteArray &value, const QByteArray &signature);

    // BEP 44: Get Manifest by Public Key
    void getManifest(const QByteArray &publicKey);

    // Announce Blob (InfoHash)
    void announceBlob(const QString &blobId, int port);

    // Find Blob Peers
    void findBlobPeers(const QString &blobId);

    // Handle Libtorrent Alerts forwarded from SessionImpl
    void handleDhtAlert(const libtorrent::alert *alert);

signals:
    void manifestFound(const Manifest &manifest);
    void peersFound(const QString &blobId, const QVector<QString> &endpoints); // "IP:Port"

private:
    libtorrent::session *m_session;

    // Cache queries to map request ID or key back to context if needed
};

}

#endif // MEGATORRENT_DHT_CLIENT_H
