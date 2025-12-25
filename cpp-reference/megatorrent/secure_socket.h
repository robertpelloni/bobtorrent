#ifndef MEGATORRENT_SECURE_SOCKET_H
#define MEGATORRENT_SECURE_SOCKET_H

#include <QTcpSocket>
#include <QByteArray>

// Note: Requires OpenSSL headers
// #include <openssl/evp.h>

namespace Megatorrent {

// Implements the Encrypted Transport Protocol (Ephemeral ECDH + ChaCha20-Poly1305)
class SecureSocket : public QObject {
    Q_OBJECT
public:
    explicit SecureSocket(QObject *parent = nullptr);
    void connectToHost(const QString &host, quint16 port);
    void write(const QByteArray &data);
    void close();

signals:
    void connected();
    void dataReceived(const QByteArray &data);
    void errorOccurred(const QString &error);

private slots:
    void onSocketConnected();
    void onReadyRead();

private:
    void processHandshake();
    void processEncryptedData();
    void sendHandshake();

    QTcpSocket *m_socket;
    bool m_handshakeComplete;
    QByteArray m_buffer;

    // Crypto State (Placeholder types)
    QByteArray m_ephemeralPub;
    QByteArray m_ephemeralPriv;
    QByteArray m_remotePub;
    QByteArray m_sharedTx;
    QByteArray m_sharedRx;
    QByteArray m_nonceTx;
    QByteArray m_nonceRx;
};

}

#endif // MEGATORRENT_SECURE_SOCKET_H
