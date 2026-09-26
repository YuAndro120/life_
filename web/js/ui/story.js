import { h, externalLink, safeUrl } from './dom.js';
import { meta, rule, topbar } from './components.js';
import { ACTIONS } from '../core/filter.js';
import { countryTitle, infoTitle, topicTitle } from '../core/taxonomy.js';
import { currentEdition } from './today.js';

/** Экран сюжета: пересказ, «что это значит» и ссылки на оригиналы. Полный текст остаётся у источников. */
export function storyScreen(ctx, id) {
  const story = ctx.store.state.feed?.stories.find((s) => s.id === id);
  if (!story) return h('section', { class: 'screen' }, topbar({ backLabel: '← Выпуск', onBack: () => ctx.nav('/') }), h('div', { class: 'empty' }, 'Сюжет не найден'));
  const theme = ctx.theme;
  const e = currentEdition(ctx);
  const sources = story.sources ?? [];
  const note = (s, i) => (s.reprint ? 'пересказывает более раннее сообщение' : sources.length > 1 ? (i === 0 ? 'первоисточник' : 'сообщил независимо') : null);
  const back = () => ctx.nav('/');
  const feedbackRow = (title, action) => h('div', null, rule(), h('button', { class: 'tap', type: 'button', style: 'width:100%;min-height:48px;color:var(--body);font-size:15px', onClick: () => { ctx.store.applyFeedback(action); back(); } }, title));
  const interests = ctx.store.state.prefs.interests;
  let metaText = theme === 'sage' ? `${topicTitle(story.topic)} · ${infoTitle(story.info_type).toLowerCase()}` : `${topicTitle(story.topic)} / ${infoTitle(story.info_type)}`;
  if (story.country && story.country !== 'RU') metaText += ` · ${countryTitle(story.country)}`;
  const shareUrl = sources[0] ? safeUrl(sources[0].url) : null;
  const canShare = shareUrl && navigator.share;

  return h('section', { class: 'screen' },
    topbar({
      backLabel: theme === 'sage' ? '‹ Выпуск' : `← Выпуск ${e?.number ?? 1}`, onBack: back,
      right: canShare ? h('button', { class: 'share', type: 'button', onClick: () => navigator.share({ title: story.title, text: story.title, url: shareUrl }).catch(() => {}) }, 'Поделиться') : h('span'),
    }),
    h('div', { style: 'margin-top:20px' }, meta(metaText)),
    h('h1', { class: 'title', style: 'font-size:28px;margin-top:10px;line-height:1.1' }, story.title),
    h('p', { style: 'font-size:17px;line-height:1.5;color:var(--body);margin-top:16px' }, story.summary),
    story.meaning ? h('p', { style: 'font-size:16px;line-height:1.5;color:var(--body);margin-top:16px' }, h('b', { style: 'color:var(--ink)' }, theme === 'sage' ? 'Значит, ' : 'Значит: '), theme === 'sage' ? story.meaning.charAt(0).toLowerCase() + story.meaning.slice(1) : story.meaning) : null,
    sources.length ? h('div', { style: 'margin-top:24px' }, meta('Читать полностью'), h('div', { style: 'margin-top:6px' }, sources.map((s, i) => h('div', null, rule(), externalLink(s.url, { class: 'link-row' },
      h('span', null, h('div', { class: 't' }, s.title), note(s, i) ? h('div', { class: 's' }, note(s, i)) : null), h('span', { class: 'muted', 'aria-hidden': 'true' }, '↗'))))), rule()) : null,
    h('p', { class: 'disclaimer' }, 'Пересказ сделан автоматически по публикациям источников. Проверяйте важное по оригиналам.'),
    h('div', { style: 'margin-top:26px' }, meta('Лента'), h('div', { style: 'margin-top:6px' },
      feedbackRow('Не интересно, скрыть сюжет', ACTIONS.hideStory(story.id)),
      feedbackRow(`Меньше про «${topicTitle(story.topic)}»`, ACTIONS.muteTopic(story.topic)),
      interests.includes(story.topic) ? null : feedbackRow(`Больше про «${topicTitle(story.topic)}»`, ACTIONS.boostTopic(story.topic)), rule())),
  );
}
