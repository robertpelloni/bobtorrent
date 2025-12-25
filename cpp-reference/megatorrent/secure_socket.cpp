#include "secure_socket.h"
#include <QDebug>
#include <QtEndian>

// Mock OpenSSL
namespace Crypto {
    void generateKeypair(QByteArray &pub, QByteArray &priv) { pub.resize(32); priv.resize(32); }
    void deriveKeys(const QByteArray &priv, const QByteArray &remotePub, bool isServer, QByteArray &tx, QByteArray &rx) { tx.resize(32); rx.resize(32); }
    QByteArray encrypt(const QByteArray &plain, const QByteArray &key, QByteArray &nonce) { return plain; }
    QByteArray decrypt(const QByteArray &cipher, const QByteArray &key, QByteArray &nonce, bool &ok) { ok = true; return cipher; }
}

namespace Megatorrent {

// Protocol Constants
const quint8 MSG_HELLO = 0x01;
const quint8 MSG_REQUEST = 0x02;
const quint8 MSG_DATA = 0x03;
const quint8 MSG_FIND_PEERS = 0x04;
const quint8 MSG_PEERS = 0x05;
const quint8 MSG_PUBLISH = 0x06;
const quint8 MSG_ANNOUNCE = 0x07;
const quint8 MSG_OK = 0x08;
const quint8 MSG_ERROR = 0xFF;

SecureSocket::SecureSocket(QObject *parent) : QObject(parent), m_handshakeComplete(false) {
    m_socket = new QTcpSocket(this);
    connect(m_socket, &QTcpSocket::connected, this, &SecureSocket::onSocketConnected);
    connect(m_socket, &QTcpSocket::readyRead, this, &SecureSocket::onReadyRead);

    Crypto::generateKeypair(m_ephemeralPub, m_ephemeralPriv);
    m_nonceTx.resize(24); m_nonceTx.fill(0);
    m_nonceRx.resize(24); m_nonceRx.fill(0);
}

void SecureSocket::connectToHost(const QString &host, quint16 port) {
    m_socket->connectToHost(host, port);
}

void SecureSocket::onSocketConnected() {
    sendHandshake();
}

void SecureSocket::sendHandshake() {
    m_socket->write(m_ephemeralPub);
}

void SecureSocket::sendMessage(quint8 type, const QByteArray &payload) {
    if (!m_handshakeComplete) return;

    QByteArray packet;
    packet.append((char)type);
    packet.append(payload);

    QByteArray cipher = Crypto::encrypt(packet, m_sharedTx, m_nonceTx);

    quint16 len = cipher.size();
    QByteArray frame;
    QDataStream ds(&frame, QIODevice::WriteOnly);
    ds << len;
    frame.append(cipher);

    m_socket->write(frame);
}

void SecureSocket::write(const QByteArray &data) {
    sendMessage(MSG_DATA, data);
}

void SecureSocket::onReadyRead() {
    m_buffer.append(m_socket->readAll());

    if (!m_handshakeComplete) {
        processHandshake();
    } else {
        processEncryptedData();
    }
}

void SecureSocket::processHandshake() {
    if (m_buffer.size() < 32) return;

    m_remotePub = m_buffer.left(32);
    m_buffer = m_buffer.mid(32);

    Crypto::deriveKeys(m_ephemeralPriv, m_remotePub, false, m_sharedTx, m_sharedRx);
    m_handshakeComplete = true;
    emit connected();

    if (!m_buffer.isEmpty()) processEncryptedData();

    // Send Hello (Gossip)
    sendMessage(MSG_HELLO, "{}"); // Placeholder JSON
}

void SecureSocket::processEncryptedData() {
    while (m_buffer.size() >= 2) {
        quint16 len = qFromBigEndian<quint16>(reinterpret_cast<const uchar*>(m_buffer.constData()));

        if (m_buffer.size() < 2 + len) return;

        QByteArray frame = m_buffer.mid(2, len);
        m_buffer = m_buffer.mid(2 + len);

        bool ok;
        QByteArray plain = Crypto::decrypt(frame, m_sharedRx, m_nonceRx, ok);

        if (!ok || plain.isEmpty()) {
            emit errorOccurred("Decryption failed");
            m_socket->close();
            return;
        }

        quint8 type = (quint8)plain.at(0);
        QByteArray payload = plain.mid(1);

        if (type == MSG_DATA) {
            emit dataReceived(payload);
        } else if (type == MSG_HELLO) {
            // Handle Gossip
        } else if (type == MSG_FIND_PEERS) {
            // Handle PEX Request
        } else if (type == MSG_PEERS) {
            // Handle PEX Response
        } else if (type == MSG_PUBLISH) {
            // Handle Gateway Publish
        } else if (type == MSG_ANNOUNCE) {
            // Handle Peer Announcement
        } else if (type == MSG_ERROR) {
            emit errorOccurred(QString::fromUtf8(payload));
        }
    }
}

void SecureSocket::close() {
    m_socket->close();
}

}
