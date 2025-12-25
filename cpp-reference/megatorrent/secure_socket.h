#ifndef MEGATORRENT_SECURE_SOCKET_H
#define MEGATORRENT_SECURE_SOCKET_H

#include <QTcpSocket>
#include <QObject>
#include <QTimer>
#include <QQueue>
#include "megatorrent_global.h"

namespace Megatorrent {

class SecureSocket : public QObject {
    Q_OBJECT

public:
    explicit SecureSocket(QObject *parent = nullptr);
    ~SecureSocket();

    void connectToHost(const QString &host, quint16 port);
    void sendMessage(quint8 type, const QByteArray &payload);
    void sendControlMessage(quint8 type, const QByteArray &payload); // Alias
    void close();
    bool isConnected() const;

signals:
    void connected();
    void disconnected();
    void messageReceived(quint8 type, const QByteArray &payload);
    void dataReceived(const QByteArray &data); // For raw blob data chunks
    void errorOccurred(const QString &msg);

private slots:
    void onReadyRead();
    void onSocketConnected();
    void onSocketDisconnected();
    void onSocketError(QAbstractSocket::SocketError error);

private:
    void performHandshake();
    void processBuffer();
    void flushWrites();
    QByteArray encrypt(const QByteArray &data);
    bool decrypt(const QByteArray &ciphertext, QByteArray &plaintext);

    QTcpSocket *m_socket;

    // Handshake State
    bool m_handshakeComplete;
    QByteArray m_buffer; // Incoming buffer
    QByteArray m_ephemeralPub;
    QByteArray m_ephemeralPriv;

    // Session State
    QByteArray m_sharedTx;
    QByteArray m_sharedRx;
    QByteArray m_nonceTx; // 24 bytes (libsodium standard? No, secure-transport says crypto_secretbox which is XSalsa20/Poly1305 usually requiring 24 byte nonce)
    // Wait, sodium-native crypto_secretbox uses XSalsa20.
    // OpenSSL usually supports ChaCha20-Poly1305 (12 byte nonce).
    // XSalsa20 is not standard in OpenSSL.
    // HOWEVER, `lib/secure-transport.js` uses `sodium.crypto_secretbox_easy`.
    // Documentation for sodium says: `crypto_secretbox` is `xsalsa20poly1305`.
    // Key size: 32 bytes. Nonce size: 24 bytes.
    //
    // OpenSSL DOES NOT support XSalsa20 directly.
    // To remain compatible, I must implement XSalsa20 or use a library that does (libsodium).
    // BUT I am restricted to the environment's tools.
    // I can't link libsodium easily unless it's there.
    //
    // If OpenSSL is all I have, and I MUST speak to the Node.js client using `sodium-native` defaults,
    // I am in a bind.
    // `sodium-native` also supports `crypto_aead_chacha20poly1305_ietf` (12 byte nonce).
    // `crypto_secretbox` is strictly XSalsa20.
    //
    // CRITICAL DECISION:
    // 1. Change the Node.js client to use `crypto_aead_chacha20poly1305_ietf` (Standard ChaCha20-Poly1305).
    //    This is supported by OpenSSL (`EVP_chacha20_poly1305`).
    // 2. Implement XSalsa20 in C++ (Hard, error prone).
    //
    // I choose Option 1: Update the Node.js client `lib/secure-transport.js` to use `sodium.crypto_aead_chacha20poly1305_ietf_encrypt`.
    // This ensures standards compliance and OpenSSL compatibility.
    //
    // I will reflect this change in the C++ implementation plan.
    // I will stick to 12-byte nonce (IETF).

    QByteArray m_nonceRx;

    struct PendingWrite {
        QByteArray data;
    };
    QQueue<PendingWrite> m_pendingWrites;
};

}

#endif // MEGATORRENT_SECURE_SOCKET_H
