const DATA_URL = (location.search.match(/[?&]data=([^&]+)/) || [, '../data/latest.json'])[1];

// --- formatting helpers (null-safe) ---
const len = x => (x || []).length;

// base-unit (1e18) decimal string -> human ETH, integer math to avoid float drift
function fmtEth(wei) {
  if (wei == null || wei === '') return '—';
  let n;
  try { n = Number(BigInt(wei) / 10n ** 14n) / 1e4; } catch { return '—'; }
  return n.toLocaleString(undefined, { maximumFractionDigits: n >= 1 ? 2 : 4 });
}
const short = a => a ? a.slice(0, 6) + '…' + a.slice(-4) : '—';
const link = (a, label) => a
  ? `<a class="addr" href="https://etherscan.io/address/${a}" target="_blank" rel="noopener" title="${a}">${label || short(a)}</a>`
  : '—';

function concBar(v) {
  if (v == null) return '—';
  const p = Math.min(100, v * 100);
  return `<span class="bar"><span style="width:${p}%"></span></span><span class="mut">${v.toFixed(3)}</span>`;
}
const card = (label, value) => `<div class="card"><div class="num">${value}</div><div class="lbl">${label}</div></div>`;

function fillTable(sel, items, cells, emptyMsg) {
  const table = document.querySelector(sel);
  const body = table.querySelector('tbody');
  body.innerHTML = '';
  if (!items || !items.length) {
    const cols = table.querySelectorAll('thead th').length;
    body.innerHTML = `<tr><td class="empty" colspan="${cols}">${emptyMsg || 'No data'}</td></tr>`;
    return;
  }
  for (const it of items) {
    const tr = document.createElement('tr');
    tr.innerHTML = cells(it).map(c => `<td>${c}</td>`).join('');
    body.appendChild(tr);
  }
}

async function main() {
  const res = await fetch(DATA_URL);
  if (!res.ok) throw new Error(`fetch ${DATA_URL} → ${res.status}`);
  const s = await res.json();
  const g = s.graph || {};

  // header
  document.getElementById('protocol').textContent = s.protocol || '—';
  document.getElementById('block').textContent = '#' + Number(s.block || 0).toLocaleString();
  const inv = s.invariants || [];
  const ok = inv.filter(i => i.ok).length;
  document.getElementById('invariants').textContent = `${ok}/${inv.length} invariants ✓`;
  document.getElementById('updated').textContent = s.timestamp
    ? 'updated ' + new Date(s.timestamp * 1000).toUTCString() : '';

  // stat cards
  const totalRestaked = (g.lrts || []).reduce((a, l) => a + BigInt(l.restaked || '0'), 0n).toString();
  document.getElementById('stats').innerHTML = [
    card('Total restaked', fmtEth(totalRestaked) + ' <span class="mut">ETH</span>'),
    card('LRTs', len(g.lrts)),
    card('Operators', len(g.operators)),
    card('AVSs', len(g.avss)),
    card('Warnings', len(s.warnings)),
  ].join('');

  // LRTs
  fillTable('#lrts', g.lrts, l => [
    `<strong>${l.symbol}</strong> ${link(l.address, '')}`,
    `${fmtEth(l.restaked)} <span class="mut">ETH</span>`,
    `<span class="num">${len(l.delegations)}</span>`,
    concBar((s.concentration || {})[l.symbol]),
  ], 'No LRTs');

  // systemic operators
  fillTable('#systemic', (s.systemic || {}).operators, o => [
    link(o.operator, o.name || short(o.operator)),
    `${fmtEth(o.total_amount)} <span class="mut">ETH</span>`,
    (o.lrts || []).join(', ') || '—',
    `<span class="num">${len(o.avss)}</span>`,
  ], 'No operator data');

  // contagion
  fillTable('#contagion', s.contagion, o => [
    o.a, o.b, len(o.shared_operators), len(o.shared_avss), (o.score ?? 0).toFixed(3),
  ], 'No shared operators or AVSs detected — shared-AVS overlaps surface once the AVS scan runs at full depth.');

  // warnings
  const sev = v => (v === 'crit' || v === 'critical') ? 'crit' : (v === 'warn' ? 'warn' : 'info');
  fillTable('#warnings', s.warnings, w => [
    `<span class="badge ${sev(w.severity)}">${w.severity}</span>`,
    w.lrt || '—',
    w.message || '',
  ], 'No warnings.');

  // raw-data link honours ?data=
  document.getElementById('datalink').href = DATA_URL;
}

main().catch(e => {
  document.getElementById('updated').textContent = 'error: ' + e.message;
});
