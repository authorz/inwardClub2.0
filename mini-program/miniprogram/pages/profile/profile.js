// 个人资料 — 头像/昵称/性别/微信手机号/等级/邀请码/绑定邀请人
// Reference: design/mini-program/final/member-subpages/01-profile-edit-v23.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const { saveProfilePersistently, mergeCachedProfile } = require('../../utils/member-profile');
const validation = require('../../utils/validation');

const GENDER_OPTIONS = [
  { label: '男', value: 'male', icon: '/assets/icons/gender-male.svg' },
  { label: '女', value: 'female', icon: '/assets/icons/gender-female.svg' },
  { label: '保密', value: 'other', icon: '' },
];

function genderState(gender) {
  const index = GENDER_OPTIONS.findIndex((item) => item.value === gender);
  const selected = index >= 0 ? GENDER_OPTIONS[index] : null;
  return {
    genderIndex: index >= 0 ? index : 0,
    genderLabel: selected ? selected.label : '',
    genderIcon: selected ? selected.icon : '',
  };
}

Page({
  data: {
    me: {},
    phoneMasked: '',
    phoneBinding: false,
    saving: false,
    avatarDirty: false,
    genderOptions: GENDER_OPTIONS,
    genderIndex: 0,
    genderLabel: '',
    genderIcon: '',
  },

  onLoad() {
    api.getMe().then((res) => {
      const me = mergeCachedProfile(res.data || {});
      me.nickname = me.nickname || me.nickName || '';
      const avatarDirty = Boolean(me.avatarUrl && !/^https?:\/\//i.test(me.avatarUrl));
      this.setData(Object.assign({ me, avatarDirty, phoneMasked: fmt.maskPhone(me.phone) }, genderState(me.gender)));
    });
  },

  onNickname(e) {
    this.setData({ 'me.nickname': e.detail.value });
  },

  chooseAvatar() {
    wx.chooseMedia({
      count: 1,
      mediaType: ['image'],
      success: (res) => {
        const file = res.tempFiles && res.tempFiles[0];
        if (file) this.setData({ 'me.avatarUrl': file.tempFilePath, avatarDirty: true });
      },
      fail: () => {},
    });
  },

  onGenderChange(e) {
    const index = Number(e.detail.value);
    const selected = GENDER_OPTIONS[index];
    if (!selected) return;
    this.setData(Object.assign({ 'me.gender': selected.value }, genderState(selected.value)));
  },

  onGetPhoneNumber(e) {
    if (this.data.phoneBinding) return;
    const detail = e.detail || {};
    const code = detail.code || '';
    if (!code) {
      ui.toast('未获取到微信手机号');
      return;
    }
    this.setData({ phoneBinding: true });
    api
      .bindPhone({ code, encryptedData: detail.encryptedData, iv: detail.iv })
      .then((res) => {
        const result = (res && res.data) || {};
        this.setData({
          phoneBinding: false,
          phoneMasked: result.phoneMasked || '已绑定',
        });
        ui.success(result.changed === false ? '手机号未变更' : '手机号已更新');
      })
      .catch((err) => {
        this.setData({ phoneBinding: false });
        ui.error((err && err.message) || '手机号获取失败，请重试');
      });
  },

  bindInviter() {
    if (this.data.me.inviterBound) return;
    wx.showModal({
      title: '绑定邀请人',
      editable: true,
      placeholderText: '输入邀请码',
      confirmColor: '#111111',
      success: (r) => {
        if (r.confirm && r.content) {
		  let inviteCode;
		  try {
		    inviteCode = validation.inviteCode(r.content, false);
		  } catch (err) {
		    ui.toast(err.message);
		    return;
		  }
          api
		    .bindInvitation({ code: inviteCode })
            .then(() => {
              ui.success('绑定成功');
              this.setData({ 'me.inviterBound': true });
            })
            .catch((err) => ui.error((err && err.message) || '绑定失败'));
        }
      },
    });
  },

  copyInvite() {
    if (this.data.me.inviteCode) ui.copy(this.data.me.inviteCode, '邀请码已复制');
  },

  save() {
    if (this.data.saving) return;
    const me = this.data.me;
	let nickname;
	try {
	  nickname = validation.nickname(me.nickname);
	} catch (err) {
	  ui.toast(err.message);
	  return;
	}
	const gender = me.gender;
	if (!GENDER_OPTIONS.some((item) => item.value === gender)) {
	  ui.toast('请选择性别');
	  return;
	}
	this.setData({ saving: true, 'me.nickname': nickname });
    const avatarUpload = this.data.avatarDirty && me.avatarUrl
      ? api.uploadAvatar(me.avatarUrl).then((res) => {
          const avatarUrl = res && res.data && res.data.avatarUrl;
          if (!avatarUrl) throw new Error('头像上传失败，请重试');
          return avatarUrl;
        })
      : Promise.resolve('');
    avatarUpload
      .then((avatarUrl) => {
        const payload = { nickname, gender };
        if (avatarUrl) payload.avatarUrl = avatarUrl;
        return api.updateMe(payload);
      })
      .then((res) => {
        const updated = Object.assign({}, me, (res && res.data) || {}, { nickname, gender });
        return saveProfilePersistently({
          avatarUrl: updated.avatarUrl || '',
          nickname: updated.nickname || '',
          gender: updated.gender || '',
        }).then(() => updated);
      })
      .then((updated) => {
        this.setData({ me: updated, avatarDirty: false, saving: false });
        ui.success('已保存');
      })
      .catch((err) => {
        this.setData({ saving: false });
        ui.error((err && err.message) || '保存失败');
      });
  },
});
