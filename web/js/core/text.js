// Общие помощники для разбора текста: строчные буквы, «ё» → «е», разбиение на слова.
export const normalize = (s) => (s ?? '').toLowerCase().replaceAll('ё', 'е');

/** Слова из букв (цифры — разделители). */
export const letterTokens = (s) => s.split(/[^\p{L}]+/u).filter(Boolean);

/** Слова из букв и цифр. */
export const alnumTokens = (s) => s.split(/[^\p{L}\p{N}]+/u).filter(Boolean);
