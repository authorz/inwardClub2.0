const api = require('../../services/api');
const auth = require('../../utils/auth');
const invitation = require('../../utils/invitation');
const ui = require('../../utils/ui');
const validation = require('../../utils/validation');
const silentLogin = require('../../utils/silent-login');
const { saveProfile } = require('../../utils/member-profile');

function freshForm() {
  return {
    avatarUrl: '',
    nickName: '',
    gender: '',
    phoneBound: false,
    phoneBinding: false,
    phoneMasked: '',
  };
}

Page({
  data: {
    mode: 'login', // login | register | bindPhone
    submitting: false,
    form: freshForm(),
  },

  onLoad() {
    this._eventChannel = this.getOpenerEventChannel ? this.getOpenerEventChannel() : null;
    if (auth.isLoggedIn()) this.finishLogin();
  },

  startLogin() {
    if (this.data.submitting) return;
    this.setData({ submitting: true });
    // Let an app-launch silent login finish before the deliberate login flow so
    // an older pre_member response can never overwrite the completed session.
    silentLogin
      .ensure()
      .catch(() => null)
      .then(() => {
        wx.login({
          success: (loginRes) => {
            api
              .wechatLogin({ code: loginRes.code })
              .then((res) => {
                const result = res.data || {};
                const token = result.token || {};
                const profile = result.profile || {};
                if (result.isNew) {
                  this._registerTicket = result.registerTicket || '';
                  this.setData({ mode: 'register', submitting: false, form: freshForm() });
                  return;
                }

                auth.save({
                  accessToken: token.accessToken,
                  refreshToken: token.refreshToken,
                  subjectType: result.subjectType,
                  storeId: result.storeId,
                });
                this._returningProfile = profile;
                if (!profile.phone) {
                  this.setData({ mode: 'bindPhone', submitting: false, form: freshForm() });
                  return;
                }
                this.finishLogin(profile);
              })
              .catch((err) => {
                this.setData({ submitting: false });
                ui.error((err && err.message) || '登录失败，请重试');
              });
          },
          fail: () => {
            this.setData({ submitting: false });
            ui.error('登录失败，请重试');
          },
        });
      });
  },

  onChooseAvatar(e) {
    this.setData({ 'form.avatarUrl': (e.detail && e.detail.avatarUrl) || '' });
  },

  onNickname(e) {
    const value = (e.detail && e.detail.value) || '';
    if (e.type === 'blur' && !value) return;
    this.setData({ 'form.nickName': value });
  },

  onGender(e) {
    this.setData({ 'form.gender': e.detail.value });
  },

  onGetPhoneNumber(e) {
    const detail = e.detail || {};
    const code = detail.code || '';
    if ((!code && !detail.encryptedData) || this.data.form.phoneBinding) return;
    this.setData({ 'form.phoneBinding': true });

    const request =
      this.data.mode === 'register'
        ? api.getPhoneMask({ registerTicket: this._registerTicket, phoneCode: code })
        : api.bindPhone({ code, encryptedData: detail.encryptedData, iv: detail.iv });

    request
      .then((res) => {
        const result = (res && res.data) || {};
        if (result.registerTicket) this._registerTicket = result.registerTicket;
        this.setData({
          'form.phoneBound': true,
          'form.phoneBinding': false,
          'form.phoneMasked': result.phoneMasked || '已授权',
        });
      })
      .catch((err) => {
        this.setData({ 'form.phoneBinding': false });
        ui.error((err && err.message) || '手机号获取失败，请重试');
      });
  },

  submitProfile() {
    if (this.data.submitting) return;
    const form = this.data.form;
    let nickName = (form.nickName || '').trim();
    const isRegister = this.data.mode === 'register';

    if (isRegister) {
      if (!form.avatarUrl) return ui.toast('请选择头像');
      if (!nickName) return ui.toast('请填写昵称');
      if (!form.phoneBound) return ui.toast('请获取手机号');
      if (!form.gender) return ui.toast('请选择性别');
      try {
        nickName = validation.nickname(nickName);
      } catch (err) {
        return ui.toast(err.message);
      }
    } else if (!form.phoneBound) {
      return ui.toast('请获取手机号');
    }

    if (!isRegister) {
      this.finishLogin(this._returningProfile);
      return;
    }

    let inviterCode = '';
    try {
      inviterCode = validation.inviteCode(invitation.get(), true);
    } catch {
      invitation.clear();
    }
    this.setData({ submitting: true });
    this.ensureUploadedAvatar(form.avatarUrl)
      .then((avatarUrl) =>
        api.register({
          registerTicket: this._registerTicket,
          avatarUrl,
          nickname: nickName,
          gender: form.gender,
          inviterCode,
        }).then((res) => ({ res, avatarUrl }))
      )
      .then(({ res, avatarUrl }) => {
        const result = res.data || {};
        const token = result.token || {};
        auth.save({
          accessToken: token.accessToken,
          refreshToken: token.refreshToken,
          subjectType: result.subjectType,
          storeId: result.storeId,
        });
        this._registerTicket = '';
        this.finishLogin({ avatarUrl, nickname: nickName, gender: form.gender });
      })
      .catch((err) => {
        this.setData({ submitting: false });
        ui.error((err && err.message) || '注册失败，请重试');
      });
  },

  ensureUploadedAvatar(path) {
    if (/^https:\/\//.test(path)) return Promise.resolve(path);
    return api.uploadRegisterAvatar(path, this._registerTicket).then((res) => {
      const avatarUrl = (res && res.data && res.data.avatarUrl) || '';
      if (!avatarUrl) throw new Error('头像上传失败，请重试');
      return avatarUrl;
    });
  },

  finishLogin(profile) {
    if (profile && (profile.avatarUrl || profile.nickname || profile.gender)) {
      saveProfile({
        avatarUrl: profile.avatarUrl || '',
        nickname: profile.nickname || profile.nickName || '',
        gender: profile.gender || '',
      });
    }
    invitation.clear();
    this.setData({ submitting: false });
    const notifyOpener = () => {
      if (this._eventChannel && typeof this._eventChannel.emit === 'function') {
        this._eventChannel.emit('loginSuccess');
      }
    };
    const pages = getCurrentPages();
    if (pages.length > 1) {
      // Notify only after the login page has closed so a protected destination
      // opened by the caller cannot be accidentally popped by navigateBack.
      wx.navigateBack({ success: notifyOpener });
    } else {
      notifyOpener();
      wx.switchTab({ url: '/pages/index/index' });
    }
  },
});
