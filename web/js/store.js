// Хранилище на устройстве. Профиль, фильтры и обратная связь живут только в браузере и никуда не отправляются.
import { DEFAULT_PREFS, DEFAULT_PROFILE, DEFAULT_SETTINGS } from './core/taxonomy.js';
import { ACTIONS, applyAction, revertAction } from './core/filter.js';

const KEYS = { profile: 'shtil.profile', prefs: 'shtil.prefs', settings: 'shtil.settings', counter: 'shtil.counter', cache: 'shtil.cache' };
export const HIDDEN_LIMIT = 300;

function read(storage, key, fallback) {
  try {
    const raw = storage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
}

function write(storage, key, value) {
  try {
    storage.setItem(key, JSON.stringify(value));
    return true;
  } catch {
    return false; // приватный режим или переполнение: приложение работает без сохранения
  }
}

/** Недостающие поля (старая версия данных) берутся по умолчанию. */
const withDefaults = (defaults, saved) => ({ ...defaults, ...(saved && typeof saved === 'object' ? saved : {}) });

export function createStore(storage) {
  const listeners = new Set();
  const state = {
    profile: withDefaults(DEFAULT_PROFILE(), read(storage, KEYS.profile, null)),
    prefs: withDefaults(DEFAULT_PREFS(), read(storage, KEYS.prefs, null)),
    settings: withDefaults(DEFAULT_SETTINGS(), read(storage, KEYS.settings, null)),
    counter: withDefaults({ number: 0, slot: '' }, read(storage, KEYS.counter, null)),
    feed: null,
    laws: [],
    fetchedAt: null,
    offline: false,
    loadError: null,
    undo: null,
  };
  const cached = read(storage, KEYS.cache, null);
  if (cached?.feed) {
    state.feed = cached.feed;
    state.laws = cached.laws ?? [];
    state.fetchedAt = cached.fetchedAt ?? null;
  }

  const emit = () => listeners.forEach((fn) => fn(state));
  const save = () => {
    write(storage, KEYS.profile, state.profile);
    write(storage, KEYS.prefs, { ...state.prefs, homeRegion: null }); // регион берётся из профиля при сборке выпуска
    write(storage, KEYS.settings, state.settings);
    write(storage, KEYS.counter, state.counter);
  };

  return {
    state,
    subscribe(fn) { listeners.add(fn); return () => listeners.delete(fn); },

    /** Настройки с регионом из профиля: по нему отбираются региональные сюжеты. */
    effectivePrefs() { return { ...state.prefs, homeRegion: state.profile.regionCode }; },

    setProfile(patch) { state.profile = { ...state.profile, ...patch }; save(); emit(); },
    setPrefs(patch) { state.prefs = typeof patch === 'function' ? patch(state.prefs) : { ...state.prefs, ...patch }; save(); emit(); },
    setSettings(patch) { state.settings = { ...state.settings, ...patch }; save(); emit(); },
    completeOnboarding() {
      state.profile = { ...state.profile, onboardingCompleted: true };
      if (state.counter.number < 1) state.counter = { ...state.counter, number: 1 };
      save(); emit();
    },

    /** Номер выпуска растёт раз в утро и вечер (по слотам), а не при каждом открытии. */
    bumpEdition(now) {
      const slot = `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}-${now.getHours() < 14 ? 'am' : 'pm'}`;
      if (state.counter.slot === slot) return;
      state.counter = { number: Math.max(state.counter.number, 1) + (state.counter.slot ? 1 : 0), slot };
      save();
    },

    setContent({ feed, laws }, now = new Date()) {
      state.feed = feed;
      state.laws = laws;
      state.fetchedAt = now.toISOString();
      state.offline = false;
      state.loadError = null;
      emit();
      // Запись ~0,5 МБ в localStorage синхронна: откладываем на простой, чтобы не подвешивать первую отрисовку.
      const persist = () => write(storage, KEYS.cache, { feed, laws, fetchedAt: state.fetchedAt });
      if (typeof requestIdleCallback === 'function') requestIdleCallback(persist, { timeout: 2000 }); else setTimeout(persist, 0);
    },
    setLoadFailed(message) {
      state.offline = state.feed !== null;
      state.loadError = state.feed ? null : message;
      emit();
    },

    /** «Не интересно» и «Больше такого»: применяется, сохраняется, запоминается для отмены. */
    applyFeedback(action) {
      const { prefs, changed } = applyAction(state.prefs, action);
      if (!changed) return false;
      const hidden = prefs.hiddenStories.length > HIDDEN_LIMIT ? { ...prefs, hiddenStories: prefs.hiddenStories.slice(-HIDDEN_LIMIT) } : prefs;
      state.prefs = hidden;
      state.undo = action;
      save(); emit();
      return true;
    },
    undoFeedback() {
      if (!state.undo) return;
      state.prefs = revertAction(state.prefs, state.undo);
      state.undo = null;
      save(); emit();
    },
    clearUndo() { if (state.undo) { state.undo = null; emit(); } },

    /** Полная замена данных (восстановление из копии). */
    restore({ profile, prefs, settings }) {
      state.profile = withDefaults(DEFAULT_PROFILE(), profile);
      state.prefs = withDefaults(DEFAULT_PREFS(), prefs);
      state.settings = withDefaults(DEFAULT_SETTINGS(), settings);
      state.counter = { number: Math.max(state.counter.number, 1), slot: state.counter.slot };
      save(); emit();
    },
    /** Снимок для резервной копии (без скрытых сюжетов и кэша). */
    snapshot() {
      return { profile: state.profile, prefs: { ...state.prefs, hiddenStories: [], homeRegion: null }, settings: state.settings };
    },
    wipe() {
      for (const key of Object.values(KEYS)) { try { storage.removeItem(key); } catch { /* нечего чистить */ } }
    },
  };
}
export { ACTIONS };
