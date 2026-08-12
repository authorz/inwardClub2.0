// Shared form-boundary validation. Keep this aligned with the Go validation
// package: the client gives immediate feedback; the API remains authoritative.
const DANGEROUS_TEXT = /(?:[<>]|&(?:lt|gt|#0*60|#0*62|#x0*3c|#x0*3e);|javascript\s*:|data\s*:\s*text\/html|(?:^|[\s"'])on(?:abort|blur|change|click|error|focus|input|load|mouseover|submit)\s*=)/i;
const PHONE = /^1[3-9]\d{9}$/;
const INVITE_CODE = /^[A-Za-z0-9]{4,32}$/;
const VERIFICATION_CODE = /^\d{6}$/;

function runeLength(value) {
  return Array.from(value).length;
}

function containsControlCharacter(value) {
  return Array.from(value).some((char) => {
    const code = char.charCodeAt(0);
    return code <= 8 || code === 11 || code === 12 || (code >= 14 && code <= 31) || code === 127;
  });
}

function plainText(value, options) {
  const opts = options || {};
  const label = opts.label || '内容';
  const text = String(value == null ? '' : value).trim();
  if (!text) {
    if (opts.allowEmpty) return '';
    throw new Error(`请填写${label}`);
  }
  if (containsControlCharacter(text) || DANGEROUS_TEXT.test(text)) {
    throw new Error(`${label}包含不安全内容`);
  }
  const length = runeLength(text);
  if (opts.min && length < opts.min) throw new Error(`${label}至少需要${opts.min}个字`);
  if (opts.max && length > opts.max) throw new Error(`${label}不能超过${opts.max}个字`);
  return text;
}

function nickname(value) {
  return plainText(value, { label: '昵称', min: 1, max: 30 });
}

function phone(value) {
  const text = String(value == null ? '' : value).trim();
  if (!PHONE.test(text)) {
	throw new Error('请输入正确的11位手机号');
  }
  return text;
}

function inviteCode(value, allowEmpty) {
  const text = String(value == null ? '' : value).trim();
  if (!text && allowEmpty) return '';
  if (!INVITE_CODE.test(text)) throw new Error('邀请码格式不正确');
  return text;
}

function verificationCode(value) {
  const text = String(value == null ? '' : value).trim();
  if (!VERIFICATION_CODE.test(text)) throw new Error('请输入6位数字核销码');
  return text;
}

function integer(value, options) {
  const opts = options || {};
  const label = opts.label || '数量';
  const text = String(value == null ? '' : value).trim();
  if (!/^\d+$/.test(text)) throw new Error(`请输入有效的${label}`);
  const number = Number(text);
  if (!Number.isSafeInteger(number) || number < (opts.min == null ? 1 : opts.min)) {
    throw new Error(`请输入有效的${label}`);
  }
  if (opts.max != null && number > opts.max) throw new Error(`${label}不能超过${opts.max}`);
  return number;
}

module.exports = {
  plainText,
  nickname,
  phone,
  inviteCode,
  verificationCode,
  integer,
};
