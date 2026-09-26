// Зашифрованная копия профиля и настроек (WebCrypto: PBKDF2 → AES-GCM). Ключ выводится из пароля и нигде не хранится,
// на сервер копия не уходит: человек сохраняет её сам (файл или код) и вставляет при переустановке.
const ITERATIONS = 250_000;
const enc = new TextEncoder();
const dec = new TextDecoder();

const toB64 = (bytes) => btoa(String.fromCharCode(...bytes)).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
const fromB64 = (s) => Uint8Array.from(atob(s.replaceAll('-', '+').replaceAll('_', '/')), (c) => c.charCodeAt(0));

async function keyFrom(password, salt, iterations) {
  const base = await crypto.subtle.importKey('raw', enc.encode(password), 'PBKDF2', false, ['deriveKey']);
  return crypto.subtle.deriveKey({ name: 'PBKDF2', salt, iterations, hash: 'SHA-256' }, base, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
}

/** Возвращает строку вида «SHTIL1.<base64url>», которую можно скопировать или сохранить в файл. */
export async function encryptBackup(snapshot, password, iterations = ITERATIONS) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const key = await keyFrom(password, salt, iterations);
  const payload = JSON.stringify({ version: 1, savedAt: new Date().toISOString(), ...snapshot });
  const data = new Uint8Array(await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, enc.encode(payload)));
  const envelope = JSON.stringify({ v: 1, i: iterations, s: toB64(salt), n: toB64(iv), d: toB64(data) });
  return `SHTIL1.${toB64(enc.encode(envelope))}`;
}

export class BackupError extends Error {}

/** Расшифровывает копию. Неверный пароль и испорченная строка дают BackupError с понятным текстом. */
export async function decryptBackup(text, password) {
  const trimmed = (text ?? '').trim();
  if (!trimmed.startsWith('SHTIL1.')) throw new BackupError('Это не копия Штиля: код должен начинаться с SHTIL1.');
  let envelope;
  try {
    envelope = JSON.parse(dec.decode(fromB64(trimmed.slice(7))));
  } catch {
    throw new BackupError('Код повреждён: скопируйте его целиком.');
  }
  if (envelope.v !== 1) throw new BackupError('Копия сделана более новой версией Штиля.');
  try {
    const key = await keyFrom(password, fromB64(envelope.s), envelope.i);
    const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: fromB64(envelope.n) }, key, fromB64(envelope.d));
    const data = JSON.parse(dec.decode(plain));
    if (data.version > 1) throw new BackupError('Копия сделана более новой версией Штиля.');
    return { profile: data.profile, prefs: data.prefs, settings: data.settings, savedAt: data.savedAt };
  } catch (e) {
    if (e instanceof BackupError) throw e;
    throw new BackupError('Неверный пароль или повреждённая копия.');
  }
}
