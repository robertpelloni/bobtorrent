#ifndef MEGATORRENT_DHT_CLIENT_H
#define MEGATORRENT_DHT_CLIENT_H

#include "global.h"
#include <QObject>
#include <QMap>
#include <QVector>

namespace Megatorrent {

// Stub for interacting with the BitTorrent DHT (libtorrent)
// Specifically for BEP 44 (Mutable Items) and Blob Announcements
class DHTClient : public QObject {
    Q_OBJECT
public:
    explicit DHTClient(QObject *parent = nullptr);

    // BEP 44: Put Signed Manifest
    void putManifest(const QByteArray &publicKey, const QByteArray &privateKey, const Manifest &manifest);

    // BEP 44: Get Manifest by Public Key
    void getManifest(const QByteArray &publicKey);

    // Announce Blob (InfoHash)
    void announceBlob(const QString &blobId, int port);

    // Find Blob Peers
    void findBlobPeers(const QString &blobId);

signals:
    void manifestFound(const Manifest &manifest);
    void peersFound(const QString &blobId, const QVector<QString> &endpoints); // "IP:Port"

private:
    // In a real implementation, this would wrap libtorrent::session
};

}

#endif // MEGATORRENT_DHT_CLIENT_H
