// 个人资料 — 头像/昵称/手机号/等级/邀请码/绑定邀请人
// Reference: design/mini-program/final/member-subpages/01-profile-edit-v23.png
const api = require('../../services/api');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const { saveProfilePersistently, mergeCachedProfile } = require('../../utils/member-profile');

Page({
  data: { me: {}, phoneMasked: '', saving: false },

  onLoad() {
    api.getMe().then((res) => {
      // getMe returns no avatarUrl (and no avatar-upload endpoint exists), so
      // fall back to the locally-cached avatar/nickname/gender picked at login —
      // same restore the home / 我的 pages do — otherwise the avatar shows blank.
      const me = mergeCachedProfile(res.data || {});
      me.nickname = me.nickname || me.nickName || '';
      this.setData({ me, phoneMasked: fmt.maskPhone(me.phone) });
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
        if (file) this.setData({ 'me.avatarUrl': file.tempFilePath });
      },
      fail: () => {},
    });
  },

  bindPhone() {
    ui.toast('请通过微信授权绑定手机号');
  },

  bindInviter() {
    if (this.data.me.boundInviter) return;
    wx.showModal({
      title: '绑定邀请人',
      editable: true,
      placeholderText: '输入邀请码',
      confirmColor: '#111111',
      success: (r) => {
        if (r.confirm && r.content) {
          api
            .bindInvitation({ code: r.content })
            .then(() => {
              ui.success('绑定成功');
              this.setData({ 'me.boundInviter': r.content });
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
    this.setData({ saving: true });
    const me = this.data.me;
    api
      .updateMe({ nickname: me.nickname })
      .then(() => {
        // Persist the edited avatar/nickname/gender locally so they survive a
        // reload — getMe returns an empty avatarUrl and there is no avatar
        // upload endpoint, so without this the chosen avatar is lost.
        const profile = {};
        if (me.avatarUrl) profile.avatarUrl = me.avatarUrl;
        if (me.nickname) profile.nickname = me.nickname;
        if (me.gender) profile.gender = me.gender;
        return Object.keys(profile).length ? saveProfilePersistently(profile) : null;
      })
      .then((saved) => {
        if (saved && saved.avatarUrl && saved.avatarUrl !== me.avatarUrl) {
          this.setData({ 'me.avatarUrl': saved.avatarUrl });
        }
        this.setData({ saving: false });
        ui.success('已保存');
      })
      .catch((err) => {
        this.setData({ saving: false });
        ui.error((err && err.message) || '保存失败');
      });
  },
});
