import test from 'node:test';
import assert from 'node:assert/strict';
import { isStandalone, platformOf, isInAppBrowser, browserOf } from '../js/platform.js';

const IPHONE_SAFARI = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1';
const IPHONE_CHROME = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/130.0 Mobile/15E148 Safari/604.1';
const ANDROID_CHROME = 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Mobile Safari/537.36';
const TELEGRAM_IOS = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Telegram-iOS';
const DESKTOP_MAC = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15';

test('установленное приложение определяется на iOS и в остальных браузерах', () => {
  assert.equal(isStandalone({ standalone: true }, () => ({ matches: false })), true);
  assert.equal(isStandalone({}, (q) => ({ matches: q.includes('standalone') })), true);
  assert.equal(isStandalone({ standalone: false }, () => ({ matches: false })), false);
  assert.equal(isStandalone({}, () => { throw new Error('нет matchMedia'); }), false);
  assert.equal(isStandalone(undefined, undefined), false);
});

test('платформа: iPhone, Android, компьютер, iPad как Mac', () => {
  assert.equal(platformOf(IPHONE_SAFARI), 'ios');
  assert.equal(platformOf(ANDROID_CHROME), 'android');
  assert.equal(platformOf(DESKTOP_MAC, 0), 'desktop');
  assert.equal(platformOf(DESKTOP_MAC, 5), 'ios');
});

test('встроенные браузеры мессенджеров узнаются, обычные — нет', () => {
  assert.equal(isInAppBrowser(TELEGRAM_IOS), true);
  assert.equal(isInAppBrowser('Mozilla/5.0 (Linux; Android 14; wv) Chrome/130 Mobile Safari/537.36 Instagram 300.0'), true);
  assert.equal(isInAppBrowser(IPHONE_SAFARI), false);
  assert.equal(isInAppBrowser(ANDROID_CHROME), false);
});

test('браузер: Safari, Chrome на iOS и Android', () => {
  assert.equal(browserOf(IPHONE_SAFARI), 'safari');
  assert.equal(browserOf(IPHONE_CHROME), 'chrome');
  assert.equal(browserOf(ANDROID_CHROME), 'chrome');
  assert.equal(browserOf('Mozilla/5.0 (Macintosh) Gecko Firefox/130.0'), 'firefox');
});
