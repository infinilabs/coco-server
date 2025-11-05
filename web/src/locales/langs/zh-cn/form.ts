const form: App.I18n.Schema['translation']['form'] = {
  code: {
    invalid: '验证码格式不正确',
    required: '请输入验证码'
  },
  confirmPwd: {
    invalid: '两次输入密码不一致',
    required: '请输入确认密码'
  },
  email: {
    invalid: '邮箱格式不正确',
    required: '请输入邮箱'
  },
  endpoint: {
    invalid: 'Endpoint格式不正确',
    required: '请输入endpoint'
  },
  phone: {
    invalid: '手机号格式不正确',
    required: '请输入手机号'
  },
  pwd: {
    invalid: '密码格式不正确，8-18位字符，包含大小字母、数字、特殊字符',
    required: '请输入密码'
  },
  pwdConfirm: {
    invalid: '两次密码输入不一致',
    required: '请输入确认密码'
  },
  required: '不能为空',
  userName: {
    invalid: '用户名格式不正确',
    required: '请输入用户名'
  },
  noSpecial: {
    invalid: '只能输入字母或数字',
    required: '不能为空'
  }
};

export default form;
