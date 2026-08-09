const api = require('../../services/api');
const ui = require('../../utils/ui');
const validation = require('../../utils/validation');

function validateField(key, value) {
  if (key === 'contactName') {
	return validation.plainText(value, { label: '称呼', min: 1, max: 50 });
  }
  if (key === 'phone') return validation.phone(value);
  if (key === 'expectedRegion') {
	return validation.plainText(value, { label: '预期开设区域', min: 1, max: 100 });
  }
  return value;
}

Page({
  data: {
    loading: true,
    submitting: false,
    sources: [],
    hotline: '',
    form: {
      contactName: '',
      phone: '',
      expectedRegion: '',
      source: '',
    },
	errors: {},
  },

  onLoad() {
    api.getFranchiseInquiryConfig()
      .then((res) => {
        const sources = (res.data && res.data.sources) || [];
        const hotline = (res.data && res.data.hotline) || '';
        this.setData({ sources, hotline, loading: false });
      })
      .catch((err) => {
        this.setData({ loading: false });
        ui.error((err && err.message) || '信息渠道加载失败，请稍后重试');
      });
  },

  onInput(e) {
    const key = e.currentTarget.dataset.key;
	let value = e.detail.value || '';
	if (key === 'phone') value = value.replace(/\D/g, '').slice(0, 11);
	const patch = { ['form.' + key]: value, ['errors.' + key]: '' };
	if (value && (key !== 'phone' || value.length === 11)) {
	  try {
	    validateField(key, value);
	  } catch (err) {
	    patch['errors.' + key] = err.message;
	  }
	}
	this.setData(patch);
  },

  onBlur(e) {
	const key = e.currentTarget.dataset.key;
	try {
	  const value = validateField(key, this.data.form[key]);
	  this.setData({ ['form.' + key]: value, ['errors.' + key]: '' });
	} catch (err) {
	  this.setData({ ['errors.' + key]: err.message });
	}
  },

  selectSource(e) {
	this.setData({ 'form.source': e.currentTarget.dataset.source, 'errors.source': '' });
  },

  copyHotline() {
    const hotline = this.data.hotline;
    if (!hotline) return ui.toast('加盟热线暂未配置');
    wx.setClipboardData({ data: hotline });
  },

  callHotline() {
    const hotline = this.data.hotline;
    if (!hotline) return ui.toast('加盟热线暂未配置');
    wx.makePhoneCall({ phoneNumber: hotline });
  },

  submit() {
    if (this.data.submitting) return;
    const form = { source: String(this.data.form.source || '').trim() };
	const errors = {};
	for (const key of ['contactName', 'phone', 'expectedRegion']) {
	  try {
	    form[key] = validateField(key, this.data.form[key]);
	  } catch (err) {
	    errors[key] = err.message;
	  }
	}
	if (!form.source) errors.source = '请选择信息渠道';
	if (Object.keys(errors).length) {
	  this.setData({ errors });
	  return ui.toast('请检查填写的信息');
	}

    this.setData({ submitting: true });
    api.createFranchiseInquiry(form)
      .then(() => wx.showModal({
        title: '提交成功',
        content: '我们会尽快与你联系',
        showCancel: false,
        confirmText: '知道了',
        success: () => wx.navigateBack(),
      }))
      .catch((err) => ui.error((err && err.message) || '提交失败，请稍后重试'))
      .finally(() => this.setData({ submitting: false }));
  },
});
