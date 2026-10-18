// v17 pixel kit: shaded 16 x 16 items, 9-slice wood frames, the night-sky scene.
// Every picture in the card is drawn on a 3 pt cell.

const MAT = {
  P: ['#6b3a2a', '#c99a6b', '#ead4aa', '#f6e8c8', '#fffaf0'],
  W: ['#3e2731', '#733e39', '#b86f50', '#e4a672', '#f2c38a'],
  R: ['#4a1020', '#a22633', '#e43b44', '#f6757a', '#ffb3b3'],
  G: ['#733e39', '#e06a1a', '#feae34', '#fee761', '#fffbd0'],
  E: ['#3a4466', '#a8b6cc', '#e2e8f2', '#f8faff', '#ffffff'],
  B: ['#3a4466', '#6a9ac8', '#a8d8f0', '#d2f0fb', '#ffffff'],
  K: ['#193c3e', '#265c42', '#3e8948', '#63c74d', '#b8f08a'],
  L: ['#2c1830', '#68386c', '#b55088', '#f6757a', '#ffc0c8'],
  S: ['#262b44', '#5a6988', '#8b9bb4', '#c0cbdc', '#ffffff'],
  D: ['#12305e', '#124e89', '#0099db', '#2ce8f5', '#ffffff'],
};
const DET = { k: '#b86f50', f: '#9aa8c0', w: '#ffffff', d: '#3e2731', g: '#e06a1a', v: '#733e39', x: '#124e89', y: '#fee761' };

function grid(w, h, fn) { const r = []; for (let y = 0; y < h; y++) { let s = ''; for (let x = 0; x < w; x++) s += fn(x, y) || '.'; r.push(s); } return r; }

const ITEMS = {
  gem: { name: '宝石', kind: '新功能', g: ['................', '................', '................', '....DDDDDDDD....', '...DwDDxxDDDD...', '..DDDDxDDxDDDD..', '..xxxxxxxxxxxx..', '...DDDxDDxDDD...', '....DDxDDxDD....', '.....DDxxDD.....', '......DDDD......', '.......DD.......', '................', '................', '................', '................'] },
  jar: { name: '虫子罐', kind: '修 bug', g: ['................', '.....WWWWWW.....', '.....WWWWWW.....', '....BBBBBBBB....', '...BwBBBBBBBB...', '...BwBBBBBBBB...', '...BBBKKKKBBB...', '...BBdKwKwdBB...', '...BBBKKKKBBB...', '...BBdKKKKdBB...', '...BBBBKKBBBB...', '...BBBBBBBBBB...', '...BBBBBBBBBB...', '....BBBBBBBB....', '................', '................'] },
  gear: { name: '齿轮', kind: '重构 · 迁移', g: grid(16, 16, (x, y) => { const dx = x - 7.5, dy = y - 7.5, r = Math.hypot(dx, dy), t = Math.atan2(dy, dx); const out = Math.cos(8 * t) > 0.15 ? 6.9 : 5.3; return r <= out && r >= 2.1 ? 'S' : ''; }) },
  book: { name: '书', kind: '文档 · 设计稿', g: ['................', '................', '...LLLLLLLLLL...', '...LLLLLLLLLLP..', '...LLwwwwwwLLP..', '...LLwkkkkwLLP..', '...LLwwwwwwLLP..', '...LLLLLLLLLLP..', '...LLLLLLLLLLP..', '...LLLLLLLLLLP..', '...LLLLLLLLLLP..', '...LLLLLLLLLLP..', '...LLLLLLLLLLP..', '....PPPPPPPPPP..', '................', '................'] },
  flask: { name: '烧瓶', kind: '测试', g: ['................', '......WWWW......', '......WWWW......', '......BBBB......', '......BwBB......', '......BBBB......', '.....BBBBBB.....', '.....BKKKKB.....', '....BKKwKKKB....', '....BKKKKKKB....', '...BKKKKKwKKB...', '...BKwKKKKKKB...', '...BKKKKKKKKB...', '...BBBBBBBBBB...', '................', '................'] },
  mail: { name: '信', kind: '回复 · 沟通', g: ['................', '................', '................', '.EEEEEEEEEEEEEE.', '.EfEEEEEEEEEEfE.', '.EEfEEEEEEEEfEE.', '.EEEfEEEEEEfEEE.', '.EEEEfEEEEfEEEE.', '.EEEEEfRRfEEEEE.', '.EEEEERRRREEEEE.', '.EEEEEERREEEEEE.', '.EEEEEEEEEEEEEE.', '.EEEEEEEEEEEEEE.', '................', '................', '................'] },
  coin: { name: '金币', kind: '钱 · 账单', g: grid(16, 16, (x, y) => { const r = Math.hypot(x - 7.5, y - 7.5); return r <= 6.3 ? (r >= 3.1 && r <= 4.0 ? 'g' : 'G') : ''; }) },
  scroll: { name: '卷轴', kind: '计划 · 调研', g: ['................', '..WWWWWWWWWWWW..', '..WWWWWWWWWWWW..', '...PPPPPPPPPP...', '...PkkkkkkkkP...', '...PPPPPPPPPP...', '...PkkkkkkPPP...', '...PPPPPPPPPP...', '...PkkkkkkkkP...', '...PPPPPPPPPP...', '...PkkkkPPPPP...', '...PPPPPPPPPP...', '..WWWWWWWWWWWW..', '..WWWWWWWWWWWW..', '................', '................'] },
  crate: { name: '木箱', kind: '其他', g: ['................', '................', '..WWWWWWWWWWWW..', '..WvvvvvvvvvvW..', '..WWWWWWWWWWWW..', '..WvWWWWWWWWvW..', '..WWvWWWWWWvWW..', '..WWWvWWWWvWWW..', '..WWWWvWWvWWWW..', '..WWWWWvvWWWWW..', '..WWWWvWWvWWWW..', '..WWWvWWWWvWWW..', '..WWvWWWWWWvWW..', '..WvvvvvvvvvvW..', '..WWWWWWWWWWWW..', '................'] },
};

// outline, light from the top left, a drop shadow down and to the right
function shade(rows) {
  const h = rows.length, w = rows[0].length;
  const at = (x, y) => (x < 0 || y < 0 || x >= w || y >= h) ? '.' : rows[y][x];
  const up = (c) => c >= 'A' && c <= 'Z';
  const matNear = (x, y) => { for (const [dx, dy] of [[0, 0], [0, -1], [-1, 0], [1, 0], [0, 1], [-1, -1], [1, 1], [1, -1], [-1, 1]]) { const c = at(x + dx, y + dy); if (up(c)) return c; } return 'W'; };
  const body = [], line = [], shadow = [];
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) {
    const c = at(x, y);
    if (c === '.') {
      const n = [at(x, y - 1), at(x - 1, y), at(x + 1, y), at(x, y + 1)].find(v => v !== '.');
      if (n) line.push([x, y, MAT[up(n) ? n : matNear(x, y)][0]]);
      continue;
    }
    if (!up(c)) { body.push([x, y, DET[c]]); continue; }
    const same = (v) => v === c || (v !== '.' && !up(v));
    const eu = !same(at(x, y - 1)), el = !same(at(x - 1, y)), ed = !same(at(x, y + 1)), er = !same(at(x + 1, y));
    const r = MAT[c];
    body.push([x, y, eu && el ? r[4] : (eu || el) ? r[3] : (ed || er) ? r[1] : r[2]]);
  }
  const solid = new Set([...body, ...line].map(([x, y]) => x + ',' + y));
  [...body, ...line].forEach(([x, y]) => { if (!solid.has((x + 1) + ',' + (y + 1))) shadow.push([x + 1, y + 1, 'rgba(40,16,24,.32)']); });
  return [...shadow, ...line, ...body];
}
const rects = (px) => px.map(([x, y, c]) => `<rect x="${x}" y="${y}" width="1" height="1" fill="${c}"/>`).join('');
const itemSVG = (id, size = 51) => `<svg class="px" width="${size}" height="${size}" viewBox="0 0 17 17">${rects(shade(ITEMS[id].g))}</svg>`;

// the gold star in a tile's corner: the work went in as a merged PR
const STAR = ['...O...', '..OyO..', '.OOyOO.', 'OyyyyyO', '.OyyyO.', '.OyOyO.', '.O.O.O.'].map(r => r.replace(/O/g, 'o'));
const starSVG = () => { const px = []; STAR.forEach((r, y) => [...r].forEach((c, x) => { if (c === 'o') px.push([x, y, '#733e39']); if (c === 'y') px.push([x, y, '#fee761']); })); return `<svg class="px star" width="21" height="21" viewBox="0 0 7 7">${rects(px)}</svg>`; };

// 9-slice wood frames as data URLs; the CSS draws them at 3 pt a cell
function frameURL(n, ring) {
  const c = document.createElement('canvas'); c.width = c.height = n; const g = c.getContext('2d'), m = n - 1;
  for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) {
    if ((x === 0 || x === m) && (y === 0 || y === m)) continue;
    const d = Math.min(x, y, m - x, m - y), tl = (y === d || x === d) && x + y <= m;
    g.fillStyle = ring(d, tl); g.fillRect(x, y, 1, 1);
  }
  return c.toDataURL();
}
const PANEL = () => frameURL(9, (d, tl) => ['#4a2318', tl ? '#e8a05c' : '#8a4a22', '#c27434', tl ? '#e8c48c' : '#f9e4b7'][d] || '#f9e4b7');
const TILE = () => frameURL(7, (d, tl) => ['#4a2318', tl ? '#e8a05c' : '#a0582a', tl ? '#e2bd84' : '#f4dcac'][d] || '#f4dcac');

// the night sky: an ordered-dither gradient, stars, two rows of hills, a lit cottage
const BAYER = [[0, 8, 2, 10], [12, 4, 14, 6], [3, 11, 1, 9], [15, 7, 13, 5]];
function rng(seed) { return () => { seed |= 0; seed = seed + 0x6D2B79F5 | 0; let t = Math.imul(seed ^ seed >>> 15, 1 | seed); t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t; return ((t ^ t >>> 14) >>> 0) / 4294967296; }; }
const hex = (h) => [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16)];
function drawSky(cv, opt = {}) {
  const W = cv.width, H = cv.height, g = cv.getContext('2d'), img = g.createImageData(W, H), d = img.data;
  const put = (x, y, c) => { if (x < 0 || y < 0 || x >= W || y >= H) return; const [r, gg, b] = hex(c), i = (y * W + x) * 4; d[i] = r; d[i + 1] = gg; d[i + 2] = b; d[i + 3] = 255; };
  const ramp = opt.ramp || ['#0b0a1f', '#110f2e', '#18143f', '#211a52', '#2c2166', '#3a2878', '#4c2e88', '#613394', '#7a389c', '#933ea1', '#a945a3'];
  const hz = opt.hz || H - 40;
  for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) {
    const t = Math.min(y / hz, 1) * (ramp.length - 1), i = Math.floor(t), f = t - i;
    put(x, y, ramp[Math.min(i + (BAYER[y % 4][x % 4] / 16 < f ? 1 : 0), ramp.length - 1)]);
  }
  const R = rng(opt.seed || 7), stars = [];
  for (let i = 0; i < 90; i++) { const x = Math.floor(R() * W), y = Math.floor(R() * (hz - 20)), k = R(); stars.push([x, y, k]); }
  stars.forEach(([x, y, k]) => {
    if (k > .93) { put(x, y, '#ffffff'); put(x - 1, y, '#a9a4ee'); put(x + 1, y, '#a9a4ee'); put(x, y - 1, '#a9a4ee'); put(x, y + 1, '#a9a4ee'); }
    else put(x, y, k > .6 ? '#e8e6ff' : '#6f6aa8');
  });
  if (opt.moon) { const [mx, my] = opt.moon; for (let y = -6; y <= 6; y++) for (let x = -6; x <= 6; x++) { const r = Math.hypot(x, y); if (r <= 6.2) put(mx + x, my + y, r > 5.2 ? '#d8cc9c' : ((x === 2 && y === -2) || (x === -2 && y === 1) || (x === -1 && y === 1) || (x === 3 && y === 2)) ? '#d8cc9c' : x + y < -3 ? '#fffbe8' : '#f4e9c1'); } }
  // haze band, far hills, near hills with a lit rim
  for (let y = hz; y < H; y++) for (let x = 0; x < W; x++) put(x, y, '#2e2160');
  const far = (x) => hz + 6 - Math.round(5 * Math.abs(Math.sin(x / 23)) + 3 * Math.sin(x / 9 + 1));
  const near = (x) => hz + 22 - Math.round(4 * Math.sin(x / 15 + 2) + 3 * Math.abs(Math.sin(x / 7)));
  for (let x = 0; x < W; x++) {
    const a = far(x); for (let y = a; y < H; y++) put(x, y, y === a ? '#3c2d75' : '#251a52');
    const b = near(x); for (let y = b; y < H; y++) put(x, y, y === b ? '#2d5a3a' : y === b + 1 ? '#204a30' : '#163a26');
  }
  for (let i = 0; i < 26; i++) { const x = Math.floor(R() * W), b = near(x); for (let k = 0; k < 4; k++) for (let j = -k; j <= k; j++) put(x + j, b - 4 + k, k === 0 ? '#2d5a3a' : '#1a4229'); }
  if (opt.house !== false) {
    const hx = 22, hb = near(hx + 6) + 2;
    for (let y = 0; y < 8; y++) for (let x = 0; x < 13; x++) put(hx + x, hb - 8 + y, x === 0 || x === 12 ? '#3e1f22' : '#7a3b2e');
    for (let k = 0; k < 6; k++) for (let x = -1 + k; x <= 13 - k; x++) put(hx + x, hb - 9 - k, k === 0 ? '#2c1418' : '#4a1f2a');
    for (let y = 0; y < 3; y++) for (let x = 0; x < 3; x++) put(hx + 4 + x, hb - 6 + y, y === 0 && x === 0 ? '#fff3b0' : '#feae34');
    put(hx + 3, hb - 3, '#ffd27a'); put(hx + 7, hb - 3, '#ffd27a');
    for (let y = 0; y < 5; y++) for (let x = 0; x < 2; x++) put(hx + 9 + x, hb - 16 + y, '#3e1f22');
  }
  g.putImageData(img, 0, 0);
  return { stars, house: opt.house !== false };
}
