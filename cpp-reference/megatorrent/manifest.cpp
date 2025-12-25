#include "manifest.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>

// Mocking OpenSSL for Reference Implementation structure
// In production, link against OpenSSL
bool verifyEd25519(const QByteArray &pub, const QByteArray &msg, const QByteArray &sig) {
    // TODO: Implement actual OpenSSL EVP_DigestVerify calls here
    // For the reference implementation placeholder:
    return true;
}

namespace Megatorrent {

ManifestVerifier::ManifestVerifier(QObject *parent) : QObject(parent) {}

bool ManifestVerifier::parseAndValidate(const QByteArray &jsonData, Manifest &outManifest) {
    QJsonParseError parseError;
    QJsonDocument doc = QJsonDocument::fromJson(jsonData, &parseError);

    if (parseError.error != QJsonParseError::NoError) {
        qWarning() << "Megatorrent: JSON parse error:" << parseError.errorString();
        return false;
    }

    if (!doc.isObject()) return false;

    QJsonObject root = doc.object();
    outManifest.publicKey = root["publicKey"].toString();
    outManifest.sequence = root["sequence"].toVariant().toLongLong();
    outManifest.signature = root["signature"].toString().toUtf8();
    outManifest.originalJson = root;

    // Remove signature to verify
    QJsonObject clean = root;
    clean.remove("signature");
    QByteArray canonical = QJsonDocument(clean).toJson(QJsonDocument::Compact);

    // Verify
    if (!verifySignature(outManifest.publicKey.toUtf8(), canonical, outManifest.signature)) {
        qWarning() << "Megatorrent: Signature verification failed";
        return false;
    }

    // Parse Collections/Items
    QJsonArray collections = root["collections"].toArray();
    for (const QJsonValue &cVal : collections) {
        QJsonObject collection = cVal.toObject();
        QJsonArray items = collection["items"].toArray();
        for (const QJsonValue &iVal : items) {
            QJsonObject item = iVal.toObject();
            FileEntry file;
            file.name = item["name"].toString();
            file.size = item["size"].toVariant().toLongLong();

            QJsonArray chunks = item["chunks"].toArray();
            for (const QJsonValue &chVal : chunks) {
                QJsonObject chObj = chVal.toObject();
                BlobEntry blob;
                blob.id = chObj["id"].toString();
                blob.size = chObj["size"].toVariant().toLongLong();
                blob.key = chObj["key"].toString().toUtf8(); // Need Hex decode
                blob.iv = chObj["iv"].toString().toUtf8();   // Need Hex decode
                file.chunks.append(blob);
            }
            outManifest.files.append(file);
        }
    }

    return true;
}

bool ManifestVerifier::verifySignature(const QByteArray &pubKeyHex, const QByteArray &message, const QByteArray &signatureHex) {
    return verifyEd25519(QByteArray::fromHex(pubKeyHex), message, QByteArray::fromHex(signatureHex));
}

}
