// Страница-подсказка для тех, кто открыл Штиль в браузере: как поставить приложение на экран «Домой» и зачем.
import { h } from './dom.js';
import { rule, screenTitle } from './components.js';
import { browserOf, isInAppBrowser, platformOf } from '../platform.js';

const SVG = 'http://www.w3.org/2000/svg';
function icon(kind) {
  const paths = {
    share: ['M12 3v12', 'M8 7l4-4 4 4', 'M6 11H5a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-8a1 1 0 0 0-1-1h-1'],
    plus: ['M12 8v8', 'M8 12h8', 'M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z'],
    dots: ['M5 12h.01', 'M12 12h.01', 'M19 12h.01'],
    menu: ['M12 5h.01', 'M12 12h.01', 'M12 19h.01'],
    home: ['M4 11l8-7 8 7', 'M6 10v10h12V10'],
    check: ['M5 12l5 5L20 7'],
    open: ['M14 4h6v6', 'M20 4l-9 9', 'M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5'],
  }[kind];
  const svg = document.createElementNS(SVG, 'svg');
  for (const [k, v] of Object.entries({ viewBox: '0 0 24 24', width: '28', height: '28', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.8', 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'aria-hidden': 'true' })) svg.setAttribute(k, v);
  for (const d of paths) {
    const p = document.createElementNS(SVG, 'path');
    p.setAttribute('d', d);
    svg.append(p);
  }
  return svg;
}

const step = (n, title, hint, kind) => h('div', { class: 'install-step' },
  h('span', { class: 'num' }, String(n)), h('span', { class: 'txt' }, h('div', { class: 't' }, title), hint ? h('div', { class: 'h' }, hint) : null), h('span', { class: 'ico' }, icon(kind)));

const BENEFITS = [
  ['01', 'Как настоящее приложение', 'Своя иконка, во весь экран, без адресной строки и вкладок.'],
  ['02', 'Работает без интернета', 'Последний выпуск открывается и в метро, и в самолёте.'],
  ['03', 'Настройки не пропадут', 'Safari стирает данные сайтов, которые давно не открывали. Установленное приложение он не трогает.'],
  ['04', 'Бесплатно и без регистрации', 'Не нужны ни App Store, ни аккаунт. Профиль хранится только у вас.'],
];

function stepsFor(ctx, ua) {
  const platform = platformOf(ua, navigator.maxTouchPoints);
  const inApp = isInAppBrowser(ua);
  const browser = browserOf(ua);
  if (inApp) {
    return { heading: 'Сначала откройте в браузере', steps: [
      step(1, 'Откройте ссылку в Safari или Chrome', 'В этом окне установить приложение нельзя. Нажмите «⋯» или значок браузера в углу и выберите «Открыть в браузере».', 'open'),
      step(2, 'Скопируйте адрес shtil.tech', 'Или отправьте его себе и откройте в Safari.', 'share'),
    ] };
  }
  if (platform === 'ios') {
    const chrome = browser === 'chrome';
    return { heading: 'Как установить на iPhone', steps: [
      step(1, chrome ? 'Нажмите «Поделиться» в адресной строке' : 'Нажмите «Поделиться» в Safari', chrome ? 'Значок квадрата со стрелкой вверх рядом с адресом.' : 'Квадрат со стрелкой вверх внизу экрана. В новых версиях Safari он в меню «⋯».', 'share'),
      step(2, 'Выберите «На экран «Домой»»', 'Листайте список действий вниз, пока не увидите этот пункт.', 'plus'),
      step(3, 'Нажмите «Добавить»', 'Штиль появится на экране «Домой». Откройте его оттуда, а не из браузера.', 'home'),
    ] };
  }
  if (platform === 'android') {
    return { heading: 'Как установить на Android', steps: [
      step(1, 'Нажмите «⋮» в правом верхнем углу Chrome', null, 'menu'),
      step(2, 'Выберите «Установить приложение»', 'Если такого пункта нет: «Добавить на главный экран».', 'plus'),
      step(3, 'Подтвердите установку', 'Иконка Штиля появится среди приложений.', 'home'),
    ] };
  }
  return { heading: 'Штиль сделан для телефона', steps: [
    step(1, 'Откройте shtil.tech на телефоне', 'Установка нужна именно там: на iPhone через Safari, на Android через Chrome.', 'share'),
    step(2, 'Или установите на компьютер', 'В Chrome и Edge значок установки справа в адресной строке.', 'plus'),
  ] };
}

export function installScreen(ctx, { onContinue, prompt, onInstall }) {
  const ua = navigator.userAgent;
  const { heading, steps } = stepsFor(ctx, ua);
  const copied = h('span', { class: 'meta', style: 'display:block;text-align:center;min-height:16px' });
  const copy = async () => { try { await navigator.clipboard.writeText('https://shtil.tech/'); copied.textContent = 'Адрес скопирован'; } catch { copied.textContent = 'Адрес: shtil.tech'; } };

  return h('section', { class: 'screen' },
    h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Штиль'), h('span', null, 'Установка')),
    screenTitle('Поставь Штиль на телефон', '.'),
    h('p', { class: 'lead' }, 'Это не просто сайт. Если добавить его на экран «Домой», он открывается и работает как приложение.'),
    h('div', { style: 'margin-top:24px' }, BENEFITS.map(([n, title, text]) => h('div', { class: 'point' }, h('span', { class: 'meta accent', style: 'padding-top:4px' }, n),
      h('span', null, h('div', { style: 'font-size:16px;font-weight:600' }, title), h('div', { style: 'font-size:14px;line-height:1.5;color:var(--muted);margin-top:2px' }, text))))),
    h('h2', { class: 'title', style: 'font-size:24px;margin-top:30px' }, heading),
    h('div', { class: 'install-steps' }, steps),
    prompt ? h('div', { style: 'margin-top:16px' }, h('button', { class: 'btn', type: 'button', onClick: onInstall }, h('span', null, 'Установить сейчас'), h('span', { 'aria-hidden': 'true' }, '↓'))) : null,
    isInAppBrowser(ua) ? h('div', { style: 'margin-top:12px' }, h('button', { class: 'link-btn', type: 'button', onClick: copy }, 'Скопировать адрес'), copied) : null,
    rule(),
    h('div', { class: 'footer' }, h('p', { class: 'meta', style: 'text-align:center;padding-top:12px' }, 'В браузере всё тоже работает, но настройки не переедут в установленное приложение'),
      h('button', { class: 'link-btn', type: 'button', onClick: onContinue }, 'Пока продолжить в браузере')),
  );
}
export { icon };
