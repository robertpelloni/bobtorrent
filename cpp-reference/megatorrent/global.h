#ifndef MEGATORRENT_GLOBAL_H
#define MEGATORRENT_GLOBAL_H

#include <QString>
#include <QByteArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QVector>

namespace Megatorrent {

struct BlobEntry {
    QString id; // SHA256 hex
    qint64 size;
    QByteArray key; // Encryption key
    QByteArray iv;  // Encryption IV
};

struct FileEntry {
    QString name;
    qint64 size;
    QString mimeType;
    QVector<BlobEntry> chunks;
};

struct Manifest {
    QString publicKey;
    qint64 sequence;
    QVector<FileEntry> files;
    QByteArray signature;
    QJsonObject originalJson;
};

}

#endif // MEGATORRENT_GLOBAL_H
