#include "megatorrentcontroller.h"

#include <QJsonArray>
#include <QJsonObject>
#include <QDateTime>

MegatorrentController::MegatorrentController(IApplication *app, QObject *parent)
    : APIController(app, parent)
{
}

void MegatorrentController::statusAction()
{
    // Stub implementation
    QJsonObject status;
    status["dht"] = "active";
    status["network"] = "connected";
    status["blobStore"] = QJsonObject{
        {"blobs", 42},
        {"size", 1024 * 1024 * 50}, // 50MB
        {"max", 1024 * 1024 * 1024 * 10ULL} // 10GB
    };
    status["subscriptions"] = 2;
    setResult(status);
}

void MegatorrentController::generateKeyAction()
{
    // Stub: Normally uses Ed25519
    QJsonObject keypair;
    keypair["publicKey"] = "deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678";
    keypair["secretKey"] = "cafebabe1234567890abcdef1234567890abcdef1234567890abcdef12345678";
    setResult(keypair);
}

void MegatorrentController::ingestAction()
{
    // Stub: Ingest logic requires file I/O and crypto
    requireParams({"filePath"});
    const QString filePath = params()["filePath"];

    QJsonObject fileEntry;
    fileEntry["name"] = filePath.section('/', -1);
    fileEntry["size"] = 1024 * 1024 * 100; // 100MB dummy

    QJsonArray chunks;
    QJsonObject chunk;
    chunk["blobId"] = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678";
    chunk["offset"] = 0;
    chunk["length"] = 1024 * 1024;
    chunks.append(chunk);

    fileEntry["chunks"] = chunks;

    QJsonObject result;
    result["fileEntry"] = fileEntry;
    result["blobCount"] = 100;

    setResult(result);
}

void MegatorrentController::publishAction()
{
    requireParams({"manifest", "privateKey"});
    // Stub: Pretend to sign and publish
    QJsonObject response;
    response["status"] = "published";
    response["sequence"] = QDateTime::currentMSecsSinceEpoch();
    setResult(response);
}

void MegatorrentController::subscribeAction()
{
    requireParams({"publicKey"});
    // Stub: Add subscription
    setResult("Subscription added");
}

void MegatorrentController::subscriptionsAction()
{
    // Stub: Return list
    QJsonArray subs;

    QJsonObject sub1;
    sub1["publicKey"] = "1111111111111111111111111111111111111111111111111111111111111111";
    sub1["lastSequence"] = 100;
    sub1["status"] = "active";
    subs.append(sub1);

    QJsonObject sub2;
    sub2["publicKey"] = "2222222222222222222222222222222222222222222222222222222222222222";
    sub2["lastSequence"] = 250;
    sub2["status"] = "syncing";
    subs.append(sub2);

    setResult(subs);
}

void MegatorrentController::unsubscribeAction()
{
    requireParams({"publicKey"});
    setResult("Unsubscribed");
}

void MegatorrentController::blobsAction()
{
    QJsonArray blobs;
    for (int i=0; i<5; ++i) {
        QJsonObject b;
        b["blobId"] = QString("blob%1").arg(i).repeated(8);
        b["size"] = 1024 * 1024;
        b["addedAt"] = QDateTime::currentMSecsSinceEpoch() - (i * 100000);
        blobs.append(b);
    }
    setResult(blobs);
}
