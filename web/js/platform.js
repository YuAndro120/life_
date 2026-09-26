// Определение окружения: установлено ли приложение на экран «Домой», на какой платформе и в каком браузере открыт сайт.

/** Открыт как установленное приложение (iOS: navigator.standalone, остальные: display-mode). */
export function isStandalone(nav = globalThis.navigator, media = globalThis.matchMedia) {
  if (nav?.standalone === true) return true;
  try {
    return Boolean(media?.('(display-mode: standalone)')?.matches || media?.('(display-mode: fullscreen)')?.matches);
  } catch {
    return false;
  }
}

/** ios | android | desktop; iPad в режиме «как компьютер» распознаётся по сенсорному экрану. */
export function platformOf(ua = '', maxTouchPoints = 0) {
  if (/iPhone|iPad|iPod/i.test(ua)) return 'ios';
  if (/Macintosh/i.test(ua) && maxTouchPoints > 1) return 'ios';
  if (/Android/i.test(ua)) return 'android';
  return 'desktop';
}

/** Встроенные браузеры мессенджеров и соцсетей: установить приложение из них нельзя, нужно открыть ссылку в Safari или Chrome. */
export function isInAppBrowser(ua = '') {
  return /FBAN|FBAV|Instagram|Line\/|Telegram|VKAndroidApp|VKClient|OKApp|MicroMessenger|Snapchat|TikTok|; wv\)/i.test(ua);
}

/** Какой браузер: safari | chrome | firefox | yandex | other (для подсказки, где искать «Поделиться»). */
export function browserOf(ua = '') {
  if (/YaBrowser/i.test(ua)) return 'yandex';
  if (/CriOS|Chrome\//i.test(ua) && !/Edg\//i.test(ua)) return 'chrome';
  if (/FxiOS|Firefox/i.test(ua)) return 'firefox';
  if (/Safari/i.test(ua) && !/Chrome|CriOS|FxiOS/i.test(ua)) return 'safari';
  return 'other';
}
