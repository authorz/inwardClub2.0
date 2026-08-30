/**
 * Formatting helpers. All money in the API is integer cents (`*Cent`) per
 * spec §5.1; UI never does floating math on its own — it formats through here.
 */

/** cents (int) -> "56.00" */
function centToYuan(cent) {
  const n = Number(cent || 0);
  return (n / 100).toFixed(2);
}

/** cents (int) -> "¥56.00" */
function yuan(cent) {
  return '¥' + centToYuan(cent);
}

/** integer asset amount (coins/points) with thousands separators */
function amount(value) {
  const n = Number(value || 0);
  return n.toLocaleString('en-US');
}

/** mask a phone number: 13800006272 -> 138****6272 */
function maskPhone(phone) {
  if (!phone) return '';
  const s = String(phone);
  const prefix = s.charAt(0) === '+' ? '+' : '';
  const digits = prefix ? s.slice(1) : s;
  if (digits.length < 7) return s;
  return prefix + digits.slice(0, 3) + '****' + digits.slice(-4);
}

/** group a numeric code in blocks of 4: "83926174" -> "8392 6174" */
function codeGroups(code) {
  if (!code) return '';
  return String(code).replace(/\s+/g, '').replace(/(.{4})/g, '$1 ').trim();
}

function pad2(n) {
  return n < 10 ? '0' + n : '' + n;
}

function toDate(input) {
  if (input instanceof Date) return input;
  if (typeof input === 'number') return new Date(input);
  if (typeof input === 'string') {
    const text = input.trim();
    // ISO/RFC3339 is the only string form parsed natively across iOS, Android
    // and the developer tools. Parsing it before any replacement also keeps
    // the timezone suffix intact and avoids iOS's non-standard-date warning.
    if (/^\d{4}-\d{2}-\d{2}T/.test(text)) {
      const iso = new Date(text);
      return isNaN(iso.getTime()) ? null : iso;
    }
    // Parse API-style local dates ourselves instead of relying on platform-
    // specific support for "YYYY-MM-DD HH:mm:ss" or slash variants.
    const local = text.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})(?:[ T](\d{1,2})(?::(\d{1,2}))?(?::(\d{1,2}))?)?$/);
    if (local) {
      const d = new Date(
        Number(local[1]),
        Number(local[2]) - 1,
        Number(local[3]),
        Number(local[4] || 0),
        Number(local[5] || 0),
        Number(local[6] || 0)
      );
      return isNaN(d.getTime()) ? null : d;
    }
    return null;
  }
  return null;
}

/** Date-like input -> epoch milliseconds; invalid input -> NaN. */
function timestamp(input) {
  const d = toDate(input);
  return d ? d.getTime() : NaN;
}

/** "2026-07-14T20:00:00Z" -> "07-14 20:00" (relative today/昨天 aware) */
function dateTime(input, opts) {
  const d = toDate(input);
  if (!d) return '';
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  const yst = new Date(now.getTime() - 86400000);
  const isYesterday = d.toDateString() === yst.toDateString();
  const hm = `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
  if (opts && opts.timeOnly) return hm;
  if ((!opts || opts.relative !== false) && sameDay) return `今天 ${hm}`;
  if ((!opts || opts.relative !== false) && isYesterday) return `昨天 ${hm}`;
  return `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${hm}`;
}

/** "2026-07-22" style date only */
function dateOnly(input) {
  const d = toDate(input);
  if (!d) return '';
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

/** dotted day: "2026-07-22T..." -> "2026.07.22" */
function dotDay(input) {
  const d = toDate(input);
  if (!d) return '';
  return `${d.getFullYear()}.${pad2(d.getMonth() + 1)}.${pad2(d.getDate())}`;
}

/** dotted month and day: "2026-07-22T..." -> "07.22" */
function dotMonthDay(input) {
  const d = toDate(input);
  if (!d) return '';
  return `${pad2(d.getMonth() + 1)}.${pad2(d.getDate())}`;
}

/** date range "2026.07.13 - 2026.07.25"; one side -> single date; none -> '' */
function dateRange(start, end) {
  const s = dotDay(start);
  const e = dotDay(end);
  if (s && e) return `${s} - ${e}`;
  return s || e;
}

/** month/day range "07.13 - 07.25"; one side -> single date; none -> '' */
function monthDayRange(start, end) {
  const s = dotMonthDay(start);
  const e = dotMonthDay(end);
  if (s && e) return `${s} - ${e}`;
  return s || e;
}

/** distance in meters/km: 1200 -> "1.2km", 800 -> "800m" */
function distance(meters) {
  const m = Number(meters || 0);
  if (!m) return '';
  if (m < 1000) return `${Math.round(m)}m`;
  return `${(m / 1000).toFixed(1)}km`;
}

module.exports = {
  centToYuan,
  yuan,
  amount,
  maskPhone,
  codeGroups,
  timestamp,
  dateTime,
  dateOnly,
  dotDay,
  dotMonthDay,
  dateRange,
  monthDayRange,
  distance,
};
