// App entry — wires the API to the store + views.
(async function main() {
  const res = await fetch('/api/items');
  const items = await res.json();

  WaveViews.renderItems(items, document.getElementById('items'));

  const listEl  = document.getElementById('cart-list');
  const countEl = document.getElementById('cart-count');
  const totalEl = document.getElementById('cart-total');

  WaveStore.subscribe(state => WaveViews.renderCart(state, listEl, countEl, totalEl));
})();
