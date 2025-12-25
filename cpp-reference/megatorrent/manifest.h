#ifndef MEGATORRENT_MANIFEST_H
#define MEGATORRENT_MANIFEST_H

#include "megatorrent_global.h"
#include <QJsonObject>
#include <QJsonArray>
#include <QJsonDocument>

namespace Megatorrent {

class Manifest {
public:
    Manifest();

    // Parse from JSON bytes
    bool parse(const QByteArray &data);

    // Getters
    QString infoHash() const;
    QVector<FileEntry> files() const;
    QByteArray publicKey() const;
    QByteArray signature() const;
    qint64 sequence() const;

    // Validation
    bool verifySignature();

private:
    QString m_infoHash;
    QVector<FileEntry> m_files;
    QByteArray m_publicKey;
    QByteArray m_signature;
    qint64 m_sequence;
};

}

#endif // MEGATORRENT_MANIFEST_H
