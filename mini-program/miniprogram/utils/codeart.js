/**
 * QR encoder for ticket / redemption verification codes.
 *
 * The server currently issues 16-character ASCII codes. Version 2-L byte mode
 * stores up to 32 bytes, so this compact implementation can generate a real,
 * scanner-readable QR matrix without a network request or runtime dependency.
 */
const SIZE = 25;
const DATA_CODEWORDS = 34;
const ERROR_CODEWORDS = 10;
const TYPE_INFO_GENERATOR = 0x537;
const TYPE_INFO_MASK = 0x5412;

const EXP = new Array(512);
const LOG = new Array(256);

let value = 1;
for (let i = 0; i < 255; i += 1) {
  EXP[i] = value;
  LOG[value] = i;
  value <<= 1;
  if (value & 0x100) value ^= 0x11d;
}
for (let i = 255; i < EXP.length; i += 1) EXP[i] = EXP[i - 255];

function utf8Bytes(input) {
  const text = String(input || '');
  const encoded = unescape(encodeURIComponent(text));
  const bytes = [];
  for (let i = 0; i < encoded.length; i += 1) bytes.push(encoded.charCodeAt(i));
  return bytes;
}

function appendBits(bits, number, length) {
  for (let i = length - 1; i >= 0; i -= 1) bits.push((number >>> i) & 1);
}

function dataCodewords(seed) {
  const bytes = utf8Bytes(seed);
  if (!bytes.length || bytes.length > 32) throw new Error('二维码内容长度必须为 1 到 32 字节');

  const bits = [];
  appendBits(bits, 0x4, 4); // byte mode
  appendBits(bits, bytes.length, 8);
  bytes.forEach((byte) => appendBits(bits, byte, 8));
  const capacity = DATA_CODEWORDS * 8;
  appendBits(bits, 0, Math.min(4, capacity - bits.length));
  while (bits.length % 8) bits.push(0);

  const out = [];
  for (let i = 0; i < bits.length; i += 8) {
    let byte = 0;
    for (let offset = 0; offset < 8; offset += 1) byte = (byte << 1) | bits[i + offset];
    out.push(byte);
  }
  let pad = 0;
  while (out.length < DATA_CODEWORDS) {
    out.push(pad % 2 === 0 ? 0xec : 0x11);
    pad += 1;
  }
  return out;
}

function multiply(a, b) {
  if (a === 0 || b === 0) return 0;
  return EXP[LOG[a] + LOG[b]];
}

function errorCorrection(data) {
  let generator = [1];
  for (let i = 0; i < ERROR_CODEWORDS; i += 1) {
    const next = new Array(generator.length + 1).fill(0);
    for (let j = 0; j < generator.length; j += 1) {
      next[j] ^= generator[j];
      next[j + 1] ^= multiply(generator[j], EXP[i]);
    }
    generator = next;
  }

  const remainder = new Array(ERROR_CODEWORDS).fill(0);
  data.forEach((byte) => {
    const factor = byte ^ remainder[0];
    remainder.shift();
    remainder.push(0);
    for (let i = 0; i < ERROR_CODEWORDS; i += 1) remainder[i] ^= multiply(generator[i + 1], factor);
  });
  return remainder;
}

function placeFinder(modules, row, col) {
  for (let r = -1; r <= 7; r += 1) {
    if (row + r < 0 || row + r >= SIZE) continue;
    for (let c = -1; c <= 7; c += 1) {
      if (col + c < 0 || col + c >= SIZE) continue;
      const dark =
        (r >= 0 && r <= 6 && (c === 0 || c === 6)) ||
        (c >= 0 && c <= 6 && (r === 0 || r === 6)) ||
        (r >= 2 && r <= 4 && c >= 2 && c <= 4);
      modules[row + r][col + c] = dark;
    }
  }
}

function placeAlignment(modules) {
  const center = 18;
  for (let r = -2; r <= 2; r += 1) {
    for (let c = -2; c <= 2; c += 1) {
      modules[center + r][center + c] = Math.max(Math.abs(r), Math.abs(c)) !== 1;
    }
  }
}

function bchTypeInfo(data) {
  let valueWithSpace = data << 10;
  const digit = (number) => {
    let count = 0;
    let n = number;
    while (n) {
      count += 1;
      n >>>= 1;
    }
    return count;
  };
  while (digit(valueWithSpace) - digit(TYPE_INFO_GENERATOR) >= 0) {
    valueWithSpace ^= TYPE_INFO_GENERATOR << (digit(valueWithSpace) - digit(TYPE_INFO_GENERATOR));
  }
  return ((data << 10) | valueWithSpace) ^ TYPE_INFO_MASK;
}

function placeTypeInfo(modules, mask) {
  const bits = bchTypeInfo((1 << 3) | mask); // error correction level L
  for (let i = 0; i < 15; i += 1) {
    const dark = ((bits >>> i) & 1) === 1;
    if (i < 6) modules[i][8] = dark;
    else if (i < 8) modules[i + 1][8] = dark;
    else modules[SIZE - 15 + i][8] = dark;

    if (i < 8) modules[8][SIZE - i - 1] = dark;
    else if (i < 9) modules[8][15 - i] = dark;
    else modules[8][14 - i] = dark;
  }
  modules[SIZE - 8][8] = true;
}

function maskBit(mask, row, col) {
  switch (mask) {
    case 0: return (row + col) % 2 === 0;
    case 1: return row % 2 === 0;
    case 2: return col % 3 === 0;
    case 3: return (row + col) % 3 === 0;
    case 4: return (Math.floor(row / 2) + Math.floor(col / 3)) % 2 === 0;
    case 5: return ((row * col) % 2) + ((row * col) % 3) === 0;
    case 6: return (((row * col) % 2) + ((row * col) % 3)) % 2 === 0;
    default: return (((row * col) % 3) + ((row + col) % 2)) % 2 === 0;
  }
}

function placeData(modules, codewords, mask) {
  let row = SIZE - 1;
  let direction = -1;
  let byteIndex = 0;
  let bitIndex = 7;

  for (let col = SIZE - 1; col > 0; col -= 2) {
    if (col === 6) col -= 1;
    while (true) {
      for (let offset = 0; offset < 2; offset += 1) {
        const x = col - offset;
        if (modules[row][x] !== null) continue;
        let dark = false;
        if (byteIndex < codewords.length) dark = ((codewords[byteIndex] >>> bitIndex) & 1) === 1;
        if (maskBit(mask, row, x)) dark = !dark;
        modules[row][x] = dark;
        bitIndex -= 1;
        if (bitIndex < 0) {
          byteIndex += 1;
          bitIndex = 7;
        }
      }
      row += direction;
      if (row < 0 || row >= SIZE) {
        row -= direction;
        direction = -direction;
        break;
      }
    }
  }
}

function matrix(seed) {
  const data = dataCodewords(seed);
  const codewords = data.concat(errorCorrection(data));
  const modules = Array.from({ length: SIZE }, () => new Array(SIZE).fill(null));

  placeFinder(modules, 0, 0);
  placeFinder(modules, SIZE - 7, 0);
  placeFinder(modules, 0, SIZE - 7);
  placeAlignment(modules);
  for (let i = 8; i < SIZE - 8; i += 1) {
    if (modules[i][6] === null) modules[i][6] = i % 2 === 0;
    if (modules[6][i] === null) modules[6][i] = i % 2 === 0;
  }
  placeTypeInfo(modules, 0);
  placeData(modules, codewords, 0);
  return modules.map((row) => row.map(Boolean));
}

/** Keyable structure for WXML: [{ id, cells: [{ id, on }] }]. */
function grid(seed) {
  return matrix(seed).map((row, y) => ({
    id: y,
    cells: row.map((on, x) => ({ id: x, on })),
  }));
}

module.exports = { matrix, grid };
