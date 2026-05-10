// Render functions — pure DOM, no framework.
window.WaveViews = {
  renderItems(items, root) {
    root.innerHTML = '';
    items.forEach(item => {
      const li = WaveUtil.el('li', {}, [
        WaveUtil.el('span', { textContent: item.name }),
        WaveUtil.el('span', { className: 'price', textContent: WaveUtil.fmt(item.price) }),
        WaveUtil.el('button', {
          textContent: 'Add',
          on: { click: () => WaveStore.add(item) }
        })
      ]);
      root.appendChild(li);
    });
  },
  renderCart(state, listEl, countEl, totalEl) {
    listEl.innerHTML = '';
    let total = 0;
    state.cart.forEach(item => {
      total += item.price;
      listEl.appendChild(WaveUtil.el('li', { textContent: `${item.name} — ${WaveUtil.fmt(item.price)}` }));
    });
    countEl.textContent = state.cart.length;
    totalEl.textContent = WaveUtil.fmt(total);
  }
};
