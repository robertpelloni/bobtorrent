#include "secure_socket.h"
#include <QDataStream>
#include <QDebug>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/kdf.h>

namespace Megatorrent {

// Constants
const int NONCE_SIZE = 12; // IETF ChaCha20-Poly1305
const int MAC_SIZE = 16;
const int KEY_SIZE = 32;

SecureSocket::SecureSocket(QObject *parent)
    : QObject(parent), m_handshakeComplete(false)
{
    m_socket = new QTcpSocket(this);
    connect(m_socket, &QTcpSocket::connected, this, &SecureSocket::onSocketConnected);
    connect(m_socket, &QTcpSocket::disconnected, this, &SecureSocket::onSocketDisconnected);
    connect(m_socket, &QTcpSocket::readyRead, this, &SecureSocket::onReadyRead);
    connect(m_socket, QOverload<QAbstractSocket::SocketError>::of(&QTcpSocket::error),
            this, &SecureSocket::onSocketError);

    m_nonceTx.resize(NONCE_SIZE);
    m_nonceRx.resize(NONCE_SIZE);
    m_nonceTx.fill(0);
    m_nonceRx.fill(0);
}

SecureSocket::~SecureSocket() {
    if (m_socket->isOpen()) m_socket->close();
}

void SecureSocket::connectToHost(const QString &host, quint16 port) {
    m_socket->connectToHost(host, port);
}

void SecureSocket::close() {
    m_socket->close();
}

bool SecureSocket::isConnected() const {
    return m_socket->state() == QAbstractSocket::ConnectedState && m_handshakeComplete;
}

void SecureSocket::onSocketConnected() {
    performHandshake();
}

void SecureSocket::onSocketDisconnected() {
    emit disconnected();
}

void SecureSocket::performHandshake() {
    // 1. Generate X25519 Keypair
    EVP_PKEY *pkey = EVP_PKEY_Q_keygen(NULL, NULL, "X25519");
    if (!pkey) {
        emit errorOccurred("Keygen failed");
        m_socket->close();
        return;
    }

    size_t len = 32;
    m_ephemeralPub.resize(32);
    EVP_PKEY_get_raw_public_key(pkey, (unsigned char*)m_ephemeralPub.data(), &len);

    // Save private key for later derivation
    m_ephemeralPriv.resize(32);
    EVP_PKEY_get_raw_private_key(pkey, (unsigned char*)m_ephemeralPriv.data(), &len);

    EVP_PKEY_free(pkey);

    // 2. Send Public Key
    m_socket->write(m_ephemeralPub);
}

void SecureSocket::onReadyRead() {
    m_buffer.append(m_socket->readAll());

    if (!m_handshakeComplete) {
        if (m_buffer.size() >= 32) {
            QByteArray remotePub = m_buffer.mid(0, 32);
            m_buffer.remove(0, 32);

            // ECDH
            EVP_PKEY *priv = EVP_PKEY_new_raw_private_key(EVP_PKEY_X25519, NULL, (const unsigned char*)m_ephemeralPriv.data(), 32);
            EVP_PKEY *pub = EVP_PKEY_new_raw_public_key(EVP_PKEY_X25519, NULL, (const unsigned char*)remotePub.data(), 32);

            EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new(priv, NULL);
            EVP_PKEY_derive_init(ctx);
            EVP_PKEY_derive_set_peer(ctx, pub);

            size_t secretLen;
            EVP_PKEY_derive(ctx, NULL, &secretLen);
            QByteArray sharedSecret(secretLen, 0);
            EVP_PKEY_derive(ctx, (unsigned char*)sharedSecret.data(), &secretLen);

            EVP_PKEY_CTX_free(ctx);
            EVP_PKEY_free(pub);
            EVP_PKEY_free(priv);

            // KDF (BLAKE2b)
            auto kdf = [&](const char* salt) {
                EVP_MD_CTX *mdctx = EVP_MD_CTX_new();
                EVP_DigestInit_ex(mdctx, EVP_blake2b512(), NULL);
                EVP_DigestUpdate(mdctx, sharedSecret.data(), sharedSecret.size());
                EVP_DigestUpdate(mdctx, salt, 1);

                unsigned char hash[64];
                unsigned int size;
                EVP_DigestFinal_ex(mdctx, hash, &size);
                EVP_MD_CTX_free(mdctx);

                return QByteArray((char*)hash, 32);
            };

            m_sharedTx = kdf("C"); // Client Tx
            m_sharedRx = kdf("S"); // Client Rx

            m_handshakeComplete = true;
            emit connected();
            flushWrites();

            if (!m_buffer.isEmpty()) processBuffer();
        }
    } else {
        processBuffer();
    }
}

void SecureSocket::processBuffer() {
    while (m_buffer.size() >= 4) {
        QDataStream ds(m_buffer);
        quint32 len;
        ds >> len; // Protocol v5: 4-byte length

        if (m_buffer.size() < 4 + len) return; // Wait for data

        QByteArray frame = m_buffer.mid(4, len);
        m_buffer.remove(0, 4 + len);

        QByteArray plain;
        if (decrypt(frame, plain)) {
            if (plain.size() > 0) {
                quint8 type = (quint8)plain[0];
                if (type == Protocol::MSG_DATA) {
                    emit dataReceived(plain.mid(1));
                } else {
                    emit messageReceived(type, plain.mid(1));
                }
            }
        } else {
            emit errorOccurred("Decryption Failed");
            m_socket->close();
            return;
        }
    }
}

void SecureSocket::sendMessage(quint8 type, const QByteArray &payload) {
    QByteArray plain;
    plain.append((char)type);
    plain.append(payload);

    PendingWrite pw;
    pw.data = plain;
    m_pendingWrites.enqueue(pw);

    if (m_handshakeComplete) flushWrites();
}

void SecureSocket::sendControlMessage(quint8 type, const QByteArray &payload) {
    sendMessage(type, payload);
}

void SecureSocket::write(const QByteArray &data) {
    sendMessage(Protocol::MSG_DATA, data);
}

void SecureSocket::flushWrites() {
    while (!m_pendingWrites.isEmpty()) {
        PendingWrite pw = m_pendingWrites.dequeue();
        QByteArray encrypted = encrypt(pw.data);

        // Header: Len (4 bytes BE)
        QByteArray packet;
        QDataStream ds(&packet, QIODevice::WriteOnly);
        ds << (quint32)encrypted.size();
        packet.append(encrypted);

        m_socket->write(packet);
    }
}

QByteArray SecureSocket::encrypt(const QByteArray &data) {
    // ChaCha20-Poly1305 IETF
    EVP_CIPHER_CTX *ctx = EVP_CIPHER_CTX_new();
    EVP_EncryptInit_ex(ctx, EVP_chacha20_poly1305(), NULL, NULL, NULL);

    EVP_EncryptInit_ex(ctx, NULL, NULL, (const unsigned char*)m_sharedTx.data(), (const unsigned char*)m_nonceTx.data());

    // Increment Nonce
    for (int i = 0; i < NONCE_SIZE; i++) {
        m_nonceTx[i] = m_nonceTx[i] + 1;
        if (m_nonceTx[i] != 0) break;
    }

    int outlen;
    QByteArray cipher(data.size() + MAC_SIZE, 0);

    EVP_EncryptUpdate(ctx, (unsigned char*)cipher.data(), &outlen, (const unsigned char*)data.data(), data.size());

    int finalLen;
    EVP_EncryptFinal_ex(ctx, (unsigned char*)cipher.data() + outlen, &finalLen);

    // Get Tag
    unsigned char tag[MAC_SIZE];
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_AEAD_GET_TAG, MAC_SIZE, tag);

    std::copy(tag, tag + MAC_SIZE, cipher.begin() + outlen);

    EVP_CIPHER_CTX_free(ctx);
    return cipher;
}

bool SecureSocket::decrypt(const QByteArray &ciphertext, QByteArray &plaintext) {
    if (ciphertext.size() < MAC_SIZE) return false;

    QByteArray tag = ciphertext.right(MAC_SIZE);
    QByteArray cipher = ciphertext.left(ciphertext.size() - MAC_SIZE);

    EVP_CIPHER_CTX *ctx = EVP_CIPHER_CTX_new();
    EVP_DecryptInit_ex(ctx, EVP_chacha20_poly1305(), NULL, NULL, NULL);
    EVP_DecryptInit_ex(ctx, NULL, NULL, (const unsigned char*)m_sharedRx.data(), (const unsigned char*)m_nonceRx.data());

    for (int i = 0; i < NONCE_SIZE; i++) {
        m_nonceRx[i] = m_nonceRx[i] + 1;
        if (m_nonceRx[i] != 0) break;
    }

    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_AEAD_SET_TAG, MAC_SIZE, (void*)tag.data());

    plaintext.resize(cipher.size());
    int outlen;
    EVP_DecryptUpdate(ctx, (unsigned char*)plaintext.data(), &outlen, (const unsigned char*)cipher.data(), cipher.size());

    int ret = EVP_DecryptFinal_ex(ctx, (unsigned char*)plaintext.data() + outlen, &outlen);

    EVP_CIPHER_CTX_free(ctx);
    return (ret > 0);
}

void SecureSocket::onSocketError(QAbstractSocket::SocketError error) {
    Q_UNUSED(error);
    emit errorOccurred(m_socket->errorString());
}

}
