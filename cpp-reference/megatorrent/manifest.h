#ifndef MEGATORRENT_MANIFEST_H
#define MEGATORRENT_MANIFEST_H

#include "global.h"

#include <QObject>
#include <QJsonArray>
#include <QJsonValue>
#include <QDebug>

// Note: In a real build, we would include OpenSSL headers here.
// #include <openssl/evp.h>
// #include <openssl/pem.h>

namespace Megatorrent {

class ManifestVerifier : public QObject {
    Q_OBJECT
public:
    explicit ManifestVerifier(QObject *parent = nullptr);

    // Parses and validates the signature of a manifest JSON
    static bool parseAndValidate(const QByteArray &jsonData, Manifest &outManifest);

private:
    static bool verifySignature(const QByteArray &pubKeyHex, const QByteArray &message, const QByteArray &signatureHex);
};

}

#endif // MEGATORRENT_MANIFEST_H
