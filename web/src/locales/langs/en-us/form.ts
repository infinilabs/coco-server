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
    invalid: '6-18 characters, including letters, numbers, and special character',
    required: 'Please enter password'
  },
  required: 'Cannot be empty',
  userName: {
    invalid: 'User name format is incorrect',
    required: 'Please enter user name'
  }
};
export default form;
