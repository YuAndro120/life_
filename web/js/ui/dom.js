// Минимальный помощник для построения DOM без innerHTML: данные с сервера всегда попадают в текстовые узлы.
export function h(tag, props, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(props ?? {})) {
    if (value === undefined || value === null || value === false) continue;
    if (key === 'class') node.className = value;
    else if (key.startsWith('on') && typeof value === 'function') node.addEventListener(key.slice(2).toLowerCase(), value);
    else if (key === 'dataset') Object.assign(node.dataset, value);
    else if (value === true) node.setAttribute(key, '');
    else node.setAttribute(key, String(value));
  }
  append(node, children);
  return node;
}

function append(node, children) {
  for (const child of children.flat(Infinity)) {
    if (child === null || child === undefined || child === false) continue;
    node.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
}

/** Ссылка наружу: только http(s), открывается в новой вкладке без передачи referrer. */
export function safeUrl(url) {
  try {
    const u = new URL(url, location.href);
    return u.protocol === 'https:' || u.protocol === 'http:' ? u.href : null;
  } catch {
    return null;
  }
}

export const externalLink = (url, props, ...children) => h('a', { ...props, href: safeUrl(url) ?? '#', target: '_blank', rel: 'noopener noreferrer' }, ...children);

export function clear(node) {
  while (node.firstChild) node.removeChild(node.firstChild);
}
