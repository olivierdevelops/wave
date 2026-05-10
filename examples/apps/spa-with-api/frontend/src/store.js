// Tiny pub-sub cart store.
window.WaveStore = (function () {
  const state = { cart: [] };
  const listeners = [];
  return {
    add(item) {
      state.cart.push(item);
      listeners.forEach(fn => fn(state));
    },
    state,
    subscribe(fn) { listeners.push(fn); fn(state); }
  };
})();
