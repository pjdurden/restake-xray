const DATA_URL = (location.search.match(/[?&]data=([^&]+)/) || [, '../data/latest.json'])[1];

function rows(sel, items, cells) {
  const body = document.querySelector(sel + ' tbody');
  for (const it of (items || [])) {
    const tr = document.createElement('tr');
    tr.innerHTML = cells(it).map(c => `<td>${c}</td>`).join('');
    body.appendChild(tr);
  }
}

async function main() {
  const s = await (await fetch(DATA_URL)).json();
  document.getElementById('meta').textContent =
    `${s.protocol} @ block ${s.block} — ${s.graph.lrts.length} LRTs, ${s.graph.operators.length} operators, ${(s.warnings || []).length} warnings`;

  rows('#lrts', s.graph.lrts, l => [l.symbol, l.restaked, l.delegations.length, (s.concentration[l.symbol] || 0).toFixed(3)]);
  rows('#systemic', (s.systemic || {}).operators, o => [o.name || o.operator, o.total_amount, o.lrts.length, o.avss.length]);
  rows('#contagion', s.contagion, o => [o.a, o.b, o.shared_operators.length, o.shared_avss.length, o.score.toFixed(3)]);
  rows('#warnings', s.warnings, w => [w.severity, w.lrt || '—', w.message]);
}
main().catch(e => document.getElementById('meta').textContent = 'error: ' + e);
