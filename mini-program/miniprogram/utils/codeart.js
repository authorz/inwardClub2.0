/**
 * Code art — renders a deterministic monochrome matrix from a code string so a
 * ticket / redemption code has a scannable-looking block that is *generated
 * from the interface data* (not a shipped AI mock image). The real scannable
 * payload is produced/verified by the server; this is the on-device visual.
 */
function matrix(seed, size) {
  const n = size || 21;
  let h = 2166136261;
  const str = String(seed || 'inwardclub');
  for (let i = 0; i < str.length; i += 1) {
    h ^= str.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  let state = h >>> 0 || 123456789;
  const rnd = () => {
    state ^= state << 13;
    state ^= state >>> 17;
    state ^= state << 5;
    return (state >>> 0) / 4294967296;
  };
  const rows = [];
  for (let y = 0; y < n; y += 1) {
    const row = [];
    for (let x = 0; x < n; x += 1) {
      // draw QR-style finder squares in three corners for a familiar shape
      const finder =
        (x < 7 && y < 7) || (x >= n - 7 && y < 7) || (x < 7 && y >= n - 7);
      if (finder) {
        const lx = x >= n - 7 ? x - (n - 7) : x;
        const ly = y >= n - 7 ? y - (n - 7) : y;
        const ring = lx === 0 || lx === 6 || ly === 0 || ly === 6;
        const core = lx >= 2 && lx <= 4 && ly >= 2 && ly <= 4;
        row.push(ring || core ? 1 : 0);
      } else {
        row.push(rnd() > 0.5 ? 1 : 0);
      }
    }
    rows.push(row);
  }
  return rows;
}

/** keyable structure for WXML: [{ id, cells: [{ id, on }] }] */
function grid(seed, size) {
  return matrix(seed, size).map((row, y) => ({
    id: y,
    cells: row.map((v, x) => ({ id: x, on: !!v })),
  }));
}

module.exports = { matrix, grid };
