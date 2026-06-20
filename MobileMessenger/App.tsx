import React, { useState, useEffect, useRef } from 'react';
import {
  SafeAreaView,
  StatusBar,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
  FlatList,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import base64 from 'base-64';
import { TextEncoder, TextDecoder } from 'text-encoding';
import { omnimesh } from './src/proto/envelope';
import { startBackgroundSync, stopBackgroundSync } from './BackgroundService';

const WS_URL = 'ws://127.0.0.1:8000/ws-messenger';
const DEFAULT_TOPIC = 'bobtorrent-global-gossip';

function encodeBase64(bytes: Uint8Array): string {
  let str = '';
  for (let i = 0; i < bytes.length; i++) {
    str += String.fromCharCode(bytes[i]);
  }
  return base64.encode(str);
}

function decodeBase64(str: string): Uint8Array {
  const binary = base64.decode(str);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

interface Message {
  id: string;
  sender: string;
  data: string;
  timestamp: string;
}

function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputText, setInputText] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    connectWebSocket();

    startBackgroundSync(() => {
      if (ws.current && ws.current.readyState === WebSocket.OPEN) {
        console.log("[App] Background heartbeat ok");
      } else {
        console.log("[App] Background reconnecting WS...");
        connectWebSocket();
      }
    });

    return () => {
      stopBackgroundSync();
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  const connectWebSocket = () => {
    ws.current = new WebSocket(WS_URL);

    ws.current.onopen = () => {
      setIsConnected(true);
      // Join default topic
      ws.current?.send(
        JSON.stringify({
          type: 'JOIN_TOPIC',
          topic: DEFAULT_TOPIC,
          data: '',
        })
      );
    };

    ws.current.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);

        if (payload.type === 'GOSSIP' && payload.topic === DEFAULT_TOPIC) {
          try {
            const bytes = decodeBase64(payload.data);
            const env = omnimesh.envelope.Envelope.decode(bytes);

            if (env.payloadType === omnimesh.envelope.Envelope.Type.CHAT && env.encryptedBody) {
              const dataStr = new TextDecoder().decode(env.encryptedBody);

              setMessages((prev) => [
                ...prev,
                {
                  id: env.id ? encodeBase64(env.id) : Math.random().toString(),
                  sender: env.senderPubkey ? encodeBase64(env.senderPubkey) : (payload.from || 'Anonymous'),
                  data: dataStr,
                  timestamp: env.timestamp ? new Date(Number(env.timestamp)).toISOString() : new Date().toISOString(),
                },
              ]);
            }
          } catch (e) {
            console.warn('Failed to decode GOSSIP protobuf:', e);
          }
        } else if (payload.type === 'HISTORY' && payload.topic === DEFAULT_TOPIC) {
          if (payload.messages && Array.isArray(payload.messages)) {
            const historyMsgs: Message[] = [];
            payload.messages.reverse().forEach((msg: any) => {
              try {
                const bytes = decodeBase64(msg.Data);
                const env = omnimesh.envelope.Envelope.decode(bytes);

                if (env.payloadType === omnimesh.envelope.Envelope.Type.CHAT && env.encryptedBody) {
                  const dataStr = new TextDecoder().decode(env.encryptedBody);
                  historyMsgs.push({
                    id: msg.ID?.toString() || Math.random().toString(),
                    sender: env.senderPubkey ? encodeBase64(env.senderPubkey) : (msg.Sender || 'Anonymous'),
                    data: dataStr,
                    timestamp: msg.Timestamp || (env.timestamp ? new Date(Number(env.timestamp)).toISOString() : new Date().toISOString()),
                  });
                }
              } catch (e) {
                console.warn('Failed to decode HISTORY protobuf for message:', e);
              }
            });
            setMessages(historyMsgs);
          }
        }
      } catch (error) {
        console.error('Error parsing WS message:', error);
      }
    };

    ws.current.onclose = () => {
      setIsConnected(false);
      // Auto-reconnect after 3s
      setTimeout(connectWebSocket, 3000);
    };
  };

  const sendMessage = () => {
    if (!inputText.trim() || !ws.current || !isConnected) return;

    const textBytes = new TextEncoder().encode(inputText.trim());

    const env = omnimesh.envelope.Envelope.create({
      id: new Uint8Array([Math.floor(Math.random() * 256)]),
      senderPubkey: new TextEncoder().encode('LocalUserPubKey'),
      timestamp: Date.now(),
      payloadType: omnimesh.envelope.Envelope.Type.CHAT,
      encryptedBody: textBytes,
    });

    const encodedEnv = omnimesh.envelope.Envelope.encode(env).finish();
    const base64Data = encodeBase64(encodedEnv);

    const payload = {
      type: 'PUBLISH',
      topic: DEFAULT_TOPIC,
      data: base64Data,
    };

    ws.current.send(JSON.stringify(payload));

    // Optimistically add to local state
    setMessages((prev) => [
      ...prev,
      {
        id: Math.random().toString(),
        sender: 'You',
        data: inputText.trim(),
        timestamp: new Date().toISOString(),
      },
    ]);
    setInputText('');
  };

  const renderItem = ({ item }: { item: Message }) => (
    <View style={[styles.messageBubble, item.sender === 'You' ? styles.myMessage : styles.theirMessage]}>
      <Text style={styles.sender}>{item.sender}</Text>
      <Text style={styles.messageText}>{item.data}</Text>
    </View>
  );

  return (
    <SafeAreaView style={styles.safeArea}>
      <StatusBar barStyle="dark-content" />
      <KeyboardAvoidingView
        style={styles.container}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <View style={styles.header}>
          <Text style={styles.headerText}>OmniMesh Chat</Text>
          <Text style={[styles.statusText, { color: isConnected ? 'green' : 'red' }]}>
            {isConnected ? 'Connected' : 'Reconnecting...'}
          </Text>
        </View>

        <FlatList
          data={messages}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
          contentContainerStyle={styles.messageList}
        />

        <View style={styles.inputContainer}>
          <TextInput
            style={styles.input}
            value={inputText}
            onChangeText={setInputText}
            placeholder="Type a message..."
            onSubmitEditing={sendMessage}
          />
          <TouchableOpacity
            style={[styles.sendButton, !isConnected && styles.sendButtonDisabled]}
            onPress={sendMessage}
            disabled={!isConnected}
          >
            <Text style={styles.sendButtonText}>Send</Text>
          </TouchableOpacity>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: '#F5F5F5',
  },
  container: {
    flex: 1,
  },
  header: {
    padding: 15,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerText: {
    fontSize: 18,
    fontWeight: 'bold',
  },
  statusText: {
    fontSize: 14,
  },
  messageList: {
    padding: 15,
  },
  messageBubble: {
    padding: 10,
    borderRadius: 8,
    marginBottom: 10,
    maxWidth: '80%',
  },
  myMessage: {
    backgroundColor: '#DCF8C6',
    alignSelf: 'flex-end',
  },
  theirMessage: {
    backgroundColor: '#FFFFFF',
    alignSelf: 'flex-start',
    borderWidth: 1,
    borderColor: '#E0E0E0',
  },
  sender: {
    fontSize: 10,
    color: '#666',
    marginBottom: 2,
  },
  messageText: {
    fontSize: 16,
  },
  inputContainer: {
    flexDirection: 'row',
    padding: 10,
    backgroundColor: '#fff',
    borderTopWidth: 1,
    borderTopColor: '#E0E0E0',
  },
  input: {
    flex: 1,
    height: 40,
    backgroundColor: '#F0F0F0',
    borderRadius: 20,
    paddingHorizontal: 15,
    marginRight: 10,
  },
  sendButton: {
    backgroundColor: '#007AFF',
    borderRadius: 20,
    paddingHorizontal: 20,
    justifyContent: 'center',
  },
  sendButtonDisabled: {
    backgroundColor: '#A0C0E0',
  },
  sendButtonText: {
    color: '#fff',
    fontWeight: 'bold',
  },
});

export default App;
