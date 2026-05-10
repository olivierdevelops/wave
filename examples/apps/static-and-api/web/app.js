document.getElementById('ping').addEventListener('click', async () => {
  const out = document.getElementById('out');
  out.textContent = 'loading...';
  try {
    const r = await fetch('/api/get');
    const text = await r.text();
    out.textContent = `HTTP ${r.status}\n\n${text}`;
  } catch (err) {
    out.textContent = `error: ${err.message}`;
  }
});
