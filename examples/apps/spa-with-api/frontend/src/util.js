// Tiny helpers — concatenated first so other files can use them.
window.WaveUtil = {
  fmt: (n) => '$' + n.toFixed(2),
  el:  (tag, attrs = {}, children = []) => {
    const node = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs)) {
      if (k === 'on') Object.entries(v).forEach(([ev, fn]) => node.addEventListener(ev, fn));
      else node[k] = v;
    }
    children.forEach(c => node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c));
    return node;
  }
};
