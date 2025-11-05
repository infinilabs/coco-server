const form: App.I18n.Schema['translation']['form'] = {
  code: {
    invalid: 'Verification code format is incorrect',
    required: 'Please enter verification code'
  },
  confirmPwd: {
    invalid: 'The two passwords are inconsistent',
    required: 'Please enter password again'
  },
  email: {
    invalid: 'Email format is incorrect',
    required: 'Please enter email'
  },
  endpoint: {
    invalid: 'Endpoint is incorrect',
    required: 'Please enter endpoint'
  },
  phone: {
    invalid: 'Phone number format is incorrect',
    required: 'Please enter phone number'
  },
  pwd: {
    invalid: 'Invalid password, 8-18 characters, including uppercase and lowercase letters, numbers, and special characters',
    required: 'Please enter password'
  },
  pwdConfirm: {
    invalid: 'Two passwords do not match',
    required: 'Please enter confirmation password'
  },
  required: 'Cannot be empty',
  userName: {
    invalid: 'User name format is incorrect',
    required: 'Please enter user name'
  },
  noSpecial: {
    invalid: 'Only letters or numbers',
    required: 'Cannot be empty'
  }
};
export default form;
