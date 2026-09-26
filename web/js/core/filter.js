// Сборка выпуска на устройстве: сюжеты и законы × профиль × настройки → выпуск. Порт FilterEngine из iOS-приложения.
import { relevantLaws } from './audience.js';
import { isPolitical, isWar, isWarEnd, isWarLaw, matchesWords } from './markers.js';
import { topicTitle } from './taxonomy.js';

export const CHARS_PER_MINUTE = 1200;
const RANK = { official: 0, fact: 1, opinion: 2, forecast: 3, rumor: 4 };

const has = (list, x) => list.includes(x);
const add = (list, x) => (has(list, x) ? list : [...list, x]);
const remove = (list, x) => list.filter((y) => y !== x);

/** Можно ли показывать сюжет: скрытое пользователем, слова, СВО, регион, тема, тип, страна, «только интересы». */
export function isAllowed(story, p) {
  if (has(p.hiddenStories, story.id)) return false;
  if (story.sources?.length && story.sources.every((s) => has(p.mutedSources, s.title))) return false;
  if (matchesWords(story, p.blockedWords)) return false;
  // Единственное исключение из «всё про СВО скрыто»: официальное заявление об окончании СВО.
  if (p.hideWar && isWarEnd(story)) return true;
  if (p.hideWar && isWar(story)) return false;
  if (p.hideOtherRegions && story.region_code && story.region_code !== p.homeRegion) return false;
  if (has(p.stopTopics, story.topic) || !has(p.infoTypes, story.info_type)) return false;
  if (!has(p.countries, story.country ?? 'RU')) return false;
  if (has(p.stopTopics, 'politics') && !has(p.interests, story.topic) && isPolitical(story)) return false;
  if (p.onlyInterests && p.interests.length && !has(p.interests, story.topic)) return false;
  return true;
}

/** Интересные темы первыми; дальше официальное выше фактов, больше источников выше, затем свежее; id — для детерминизма. */
export function storyOrder(a, b, interests = []) {
  const ia = has(interests, a.topic), ib = has(interests, b.topic);
  if (ia !== ib) return ia ? -1 : 1;
  if (RANK[a.info_type] !== RANK[b.info_type]) return RANK[a.info_type] - RANK[b.info_type];
  if (a.source_count !== b.source_count) return b.source_count - a.source_count;
  const ta = Date.parse(a.updated_at), tb = Date.parse(b.updated_at);
  if (ta !== tb) return tb - ta;
  return a.id < b.id ? -1 : 1;
}

export function readingMinutes(laws, stories) {
  const lawChars = laws.reduce((n, l) => n + l.title.length + l.what_changed.length + l.who_affected.length + l.actions.reduce((m, a) => m + a.length, 0), 0);
  const storyChars = stories.reduce((n, s) => n + s.title.length + s.summary.length + (s.meaning?.length ?? 0), 0);
  const total = lawChars + storyChars;
  return total > 0 ? Math.ceil(total / CHARS_PER_MINUTE) : 0;
}

/** window: {start: ISO|null, end: ISO}; today: {y, m, d}. */
export function buildEdition({ number, feed, laws, profile, prefs, window, today }) {
  const relevant = relevantLaws(laws, profile, today).filter((l) => !(prefs.hideWar && isWarLaw(l)));
  const end = Date.parse(window.end), start = window.start ? Date.parse(window.start) : null;
  const inWindow = feed.stories.filter((s) => Date.parse(s.updated_at) <= end && (start === null || Date.parse(s.updated_at) > start));
  const allowed = inWindow.filter((s) => isAllowed(s, prefs));

  const heavy = allowed.filter((s) => s.heaviness === 'heavy').sort((a, b) => storyOrder(a, b));
  const regular = allowed.filter((s) => s.heaviness !== 'heavy');
  let shownHeavy = [], folded = [];
  if (prefs.heavyMode === 'fold') folded = heavy;
  else if (prefs.heavyMode === 'show') {
    const limit = Math.max(0, prefs.maxHeavy);
    shownHeavy = heavy.slice(0, limit);
    folded = heavy.slice(limit);
  }

  const ranked = [...regular, ...shownHeavy].sort((a, b) => storyOrder(a, b, prefs.interests));
  const stories = ranked.slice(0, Math.max(1, prefs.storyLimit));
  const kept = ranked.length + folded.length;
  const interestIds = new Set(stories.filter((s) => has(prefs.interests, s.topic)).map((s) => s.id));
  return {
    number, laws: relevant, stories, foldedHeavy: folded, interestIds,
    stats: {
      aboutYou: relevant.length, stories: stories.length, readingMinutes: readingMinutes(relevant, stories),
      postsTotal: feed.stats?.posts_total ?? 0, adsHidden: feed.stats?.ads_hidden ?? 0,
      filteredOut: inWindow.length - kept, trimmed: ranked.length - stories.length,
    },
  };
}

// --- Настройки: интересы и стоп-темы не пересекаются ---
export function toggleInterest(p, topic) {
  if (has(p.interests, topic)) return { ...p, interests: remove(p.interests, topic) };
  return { ...p, interests: add(p.interests, topic), stopTopics: remove(p.stopTopics, topic) };
}
export function toggleStopTopic(p, topic) {
  if (has(p.stopTopics, topic)) return { ...p, stopTopics: remove(p.stopTopics, topic) };
  return { ...p, stopTopics: add(p.stopTopics, topic), interests: remove(p.interests, topic) };
}

// --- «Не интересно»: действия хранятся только на устройстве ---
export const ACTIONS = {
  hideStory: (id) => ({ type: 'hideStory', id }),
  muteSource: (title) => ({ type: 'muteSource', title }),
  muteTopic: (topic) => ({ type: 'muteTopic', topic }),
  boostTopic: (topic) => ({ type: 'boostTopic', topic }),
};

export function actionMessage(a) {
  switch (a.type) {
    case 'hideStory': return 'Сюжет скрыт';
    case 'muteSource': return `Источник скрыт: ${a.title}`;
    case 'muteTopic': return `Меньше про: ${topicTitle(a.topic)}`;
    default: return `Больше про: ${topicTitle(a.topic)}`;
  }
}

/** Применяет действие; возвращает {prefs, changed} — changed=false, если менять было нечего. */
export function applyAction(p, a) {
  switch (a.type) {
    case 'hideStory': return has(p.hiddenStories, a.id) ? { prefs: p, changed: false } : { prefs: { ...p, hiddenStories: add(p.hiddenStories, a.id) }, changed: true };
    case 'muteSource': return has(p.mutedSources, a.title) ? { prefs: p, changed: false } : { prefs: { ...p, mutedSources: add(p.mutedSources, a.title) }, changed: true };
    case 'muteTopic': return has(p.stopTopics, a.topic) ? { prefs: p, changed: false } : { prefs: toggleStopTopic(p, a.topic), changed: true };
    default: return has(p.interests, a.topic) ? { prefs: p, changed: false } : { prefs: toggleInterest(p, a.topic), changed: true };
  }
}

export function revertAction(p, a) {
  switch (a.type) {
    case 'hideStory': return { ...p, hiddenStories: remove(p.hiddenStories, a.id) };
    case 'muteSource': return { ...p, mutedSources: remove(p.mutedSources, a.title) };
    case 'muteTopic': return { ...p, stopTopics: remove(p.stopTopics, a.topic) };
    default: return { ...p, interests: remove(p.interests, a.topic) };
  }
}
