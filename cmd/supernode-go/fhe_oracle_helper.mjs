import { homomorphicAddPlain, homomorphicMultiplyPlain } from '../../bobcoin/game-server/fheUtils.js';

async function readStdin() {
  return await new Promise((resolve, reject) => {
    let data = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', chunk => {
      data += chunk;
    });
    process.stdin.on('end', () => resolve(data));
    process.stdin.on('error', reject);
  });
}

try {
  const raw = await readStdin();
  const request = JSON.parse(raw || '{}');
  const cipherText = typeof request.cipherText === 'string' ? request.cipherText.trim() : '';
  const multiply = Number.isFinite(request.multiply) ? request.multiply : 2;
  const add = Number.isFinite(request.add) ? request.add : 500;

  if (!cipherText) {
    process.stdout.write(JSON.stringify({ success: false, error: 'Encrypted payload missing' }));
    process.exit(0);
  }

  const multipliedCipher = await homomorphicMultiplyPlain(cipherText, multiply);
  const finalCipher = await homomorphicAddPlain(multipliedCipher, add);
  process.stdout.write(JSON.stringify({ success: true, resultCipher: finalCipher }));
} catch (error) {
  process.stdout.write(JSON.stringify({ success: false, error: error?.message || String(error) }));
}
