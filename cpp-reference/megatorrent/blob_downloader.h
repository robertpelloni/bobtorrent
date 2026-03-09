#ifndef MEGATORRENT_BLOB_DOWNLOADER_H
#define MEGATORRENT_BLOB_DOWNLOADER_H

#include <QObject>
#include <QMap>
#include <QSet>
#include <QVector>
#include <QFile>
#include <QDir>
#include "secure_socket.h"
#include "megatorrent_global.h"

namespace Megatorrent {

struct BlobRequest {
    QString blobId;
    qint64 size;
    QByteArray key;
    QByteArray iv;
    QString savePath;
};

class BlobDownloader : public QObject {
    Q_OBJECT

public:
    explicit BlobDownloader(QObject *parent = nullptr);
    ~BlobDownloader();

    void queueBlob(const QString &blobId, qint64 size, const QByteArray &key, const QByteArray &iv, const QString &savePath);
    void addPeers(const QString &blobId, const QVector<QString> &endpoints);

signals:
    void blobFinished(const QString &blobId);
    void blobFailed(const QString &blobId, const QString &error);
    void peersNeeded(const QString &blobId); // Signal to DHT to find peers if we run out

private slots:
    void onSocketConnected();
    void onSocketDisconnected();
    void onSocketError(const QString &msg);
    void onMessageReceived(quint8 type, const QByteArray &payload);

private:
    struct ActiveDownload {
        BlobRequest request;
        QVector<QString> peers;
        QSet<QString> triedPeers;
        SecureSocket *socket = nullptr;
        QByteArray buffer;
        bool active = false;
    };

    void startNextDownload();
    void tryNextPeer(const QString &blobId);

    QMap<QString, ActiveDownload> m_downloads; // blobId -> Download State
    QVector<QString> m_queue; // Download queue order

    // Limits
    int m_maxConcurrent = 3;
    int m_currentActive = 0;
};

}

#endif // MEGATORRENT_BLOB_DOWNLOADER_H
