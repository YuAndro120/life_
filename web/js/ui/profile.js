import { h } from './dom.js';
import { meta, rule, screenTitle, sectionTitle, sheet, switchRow, button } from './components.js';
import { profileForm } from './profile-form.js';
import { THEMES } from '../core/taxonomy.js';
import { THEME_TOKENS } from '../theme.js';
import { encryptBackup, decryptBackup, BackupError } from '../backup.js';

/** Мини-макет темы: заголовок, плашка с карточкой и строки текста в её собственных цветах (как в iOS-приложении). */
function themePreview(id) {
  const t = THEME_TOKENS[id];
  const bar = (width, height, color, opacity, extra = '') => h('div', { style: `width:${width}%;height:${height}px;border-radius:99px;background:${color};opacity:${opacity};${extra}` });
  return h('div', { class: 'theme-preview', 'aria-hidden': 'true', style: `background:${t.bg};border-color:${t.dark ? '#3a3c46' : t.line}` },
    bar(46, 9, t.ink, 1), bar(72, 5, t.muted, 0.6, 'margin-top:6px'),
    h('div', { style: `margin-top:10px;padding:5px;border-radius:10px;background:${t.plate}` },
      h('div', { style: `display:flex;align-items:center;gap:5px;height:30px;padding:0 7px;border-radius:7px;background:${t.card}` },
        h('span', { style: `width:6px;height:6px;border-radius:50%;background:${t.accent};flex:none` }), h('div', { style: `flex:1;height:4px;border-radius:99px;background:${t.ink};opacity:0.55` }))),
    bar(88, 5, t.ink, 0.5, 'margin-top:10px'), bar(60, 5, t.muted, 0.5, 'margin-top:6px'));
}

/** Выбор темы (три мини-макета) и «Сумерки по вечерам». */
export function themePicker(ctx) {
  const { store } = ctx;
  return h('div', { class: 'theme-picker' },
    h('div', { class: 'theme-grid' }, THEMES.map((t) => h('button', { class: 'theme-opt', type: 'button', 'aria-pressed': String(store.state.settings.theme === t.id), onClick: () => store.setSettings({ theme: t.id }) },
      themePreview(t.id), h('div', { class: 't' }, t.title), h('div', { class: 'h' }, t.hint)))),
    h('div', { style: 'margin-top:14px' }, switchRow({ title: 'Сумерки по вечерам', hint: 'Тёмная тема сама включится после 19:00', on: store.state.settings.autoDusk, onChange: (v) => store.setSettings({ autoDusk: v }) })));
}

/** Резервная копия: зашифрованный код или файл; ключ выводится из пароля и нигде не хранится. */
export function openBackupSheet(ctx, mode, onDone) {
  const { store } = ctx;
  const close = () => { backdrop.dispose?.(); backdrop.remove(); };
  const status = h('div');
  const say = (text, err = false) => status.replaceChildren(h('div', { class: `toast${err ? ' err' : ''}`, role: err ? 'alert' : 'status' }, text));
  const password = h('input', { class: 'field', type: 'password', placeholder: 'Пароль для копии', autocomplete: mode === 'save' ? 'new-password' : 'current-password', 'aria-label': 'Пароль' });
  const code = h('textarea', { class: 'field', rows: 4, placeholder: 'Код копии SHTIL1.…', spellcheck: 'false', 'aria-label': 'Код копии', style: 'font-size:13px;font-family:monospace;margin-top:10px' });
  const fileInput = h('input', { type: 'file', accept: '.txt,.shtil', hidden: true, onChange: async (e) => { const f = e.target.files?.[0]; if (f) code.value = (await f.text()).trim(); } });

  const save = async () => {
    if (password.value.length < 6) return say('Пароль не короче 6 знаков. Без него копию не открыть.', true);
    try {
      code.value = await encryptBackup(store.snapshot(), password.value);
      say('Копия готова: скопируйте код или сохраните файл. Пароль нигде не хранится: забудете его — копию не восстановить.');
    } catch { say('Не удалось создать копию в этом браузере.', true); }
  };
  const restore = async () => {
    try {
      const data = await decryptBackup(code.value, password.value);
      store.restore(data);
      store.completeOnboarding();
      close();
      onDone?.();
    } catch (e) { say(e instanceof BackupError ? e.message : 'Не удалось прочитать копию.', true); }
  };
  const copy = async () => { try { await navigator.clipboard.writeText(code.value); say('Код скопирован.'); } catch { say('Не удалось скопировать: выделите код вручную.', true); } };
  const download = () => {
    const url = URL.createObjectURL(new Blob([code.value], { type: 'text/plain' }));
    const a = h('a', { href: url, download: 'shtil-backup.txt' });
    document.body.append(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 5000);
  };

  const body = mode === 'save'
    ? h('div', null, h('p', { class: 'lead' }, 'Профиль и фильтры шифруются на этом устройстве. Сохраните код: при переустановке или на новом телефоне его можно вставить обратно.'),
      h('div', { style: 'margin-top:14px' }, password), h('div', { style: 'margin-top:12px' }, button({ title: 'Создать код', trailing: '', onClick: save })), code,
      h('div', { class: 'row', style: 'margin-top:8px' }, h('button', { class: 'tap', type: 'button', style: 'color:var(--accent);font-weight:500', onClick: copy }, 'Скопировать'), h('button', { class: 'tap', type: 'button', style: 'color:var(--accent);font-weight:500', onClick: download }, 'Сохранить файлом')), status)
    : h('div', null, h('p', { class: 'lead' }, 'Вставьте код копии (или выберите файл) и введите пароль, с которым она создавалась.'),
      code, h('div', { class: 'row', style: 'margin-top:8px' }, h('button', { class: 'tap', type: 'button', style: 'color:var(--accent);font-weight:500', onClick: () => fileInput.click() }, 'Выбрать файл'), fileInput),
      h('div', { style: 'margin-top:10px' }, password), h('div', { style: 'margin-top:12px' }, button({ title: 'Восстановить', trailing: '', onClick: restore })), status);
  const backdrop = sheet(h('div', null, h('div', { class: 'row', style: 'padding-top:14px' }, h('h2', { class: 'title', style: 'font-size:22px' }, mode === 'save' ? 'Сохранить копию' : 'Восстановить настройки'),
    h('button', { class: 'tap', type: 'button', onClick: close, style: 'color:var(--accent);font-weight:500' }, 'Закрыть')), body), close);
  document.body.append(backdrop);
}

export function profileScreen(ctx) {
  const { store } = ctx;
  const theme = ctx.theme;
  const persisted = ctx.local.persisted;
  return h('section', { class: 'screen' },
    theme === 'sage' ? h('p', { class: 'meta', style: 'padding-top:14px' }, 'Хранится только в этом браузере') : h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Профиль'), h('span', null, 'Хранится в браузере')),
    screenTitle('Профиль'),
    h('p', { class: 'lead' }, 'Покажем только те законы и изменения, которые касаются тебя.'),
    h('div', { style: 'margin-top:28px' }, profileForm(ctx)),
    h('div', null, sectionTitle('09', 'Оформление', theme), h('div', { class: 'section-body' }, themePicker(ctx))),
    h('div', null, sectionTitle('10', 'Резервная копия', theme), h('div', { class: 'section-body' },
      h('p', { style: 'font-size:14px;line-height:1.5;color:var(--muted)' }, 'Браузер может стереть данные сайта, а при переустановке они пропадают. Сохраните зашифрованную копию профиля и фильтров: она открывается только вашим паролем и не уходит на сервер.'),
      h('div', { style: 'display:grid;gap:8px;margin-top:12px' }, button({ title: 'Сохранить копию', trailing: '', onClick: () => openBackupSheet(ctx, 'save') }),
        h('button', { class: 'link-btn', type: 'button', onClick: () => openBackupSheet(ctx, 'restore', () => ctx.rerender()) }, 'Восстановить из копии')),
      persisted === false ? h('p', { class: 'meta', style: 'margin-top:12px' }, 'Браузер не гарантирует сохранность данных. Добавьте Штиль на экран «Домой»: так данные хранятся надёжнее.') : null)),
    h('p', { class: 'disclaimer', style: 'margin-top:28px' }, 'Профиль хранится только в этом браузере. Сервер не знает, кто ты и что читаешь.'),
    h('button', { class: 'link-btn', type: 'button', style: 'margin-top:8px', onClick: () => { if (confirm('Стереть профиль и настройки на этом устройстве?')) { store.wipe(); location.hash = '#/'; location.reload(); } } }, 'Стереть данные на этом устройстве'),
  );
}
export { meta, rule };
