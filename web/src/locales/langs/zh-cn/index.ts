import common from './common';
import form from './form';
import page from './page';
import permission from './permission';
import route from './route';
import theme from './theme';

const local: App.I18n.Schema['translation'] = {
  common,
  datatable: {
    itemCount: '共 {{total}} 条'
  },
  dropdown: {
    closeAll: '关闭所有',
    closeCurrent: '关闭',
    closeLeft: '关闭左侧',
    closeOther: '关闭其它',
    closeRight: '关闭右侧'
  },
  form,
  icon: {
    collapse: '折叠菜单',
    expand: '展开菜单',
    fullscreen: '全屏',
    fullscreenExit: '退出全屏',
    lang: '切换语言',
    pin: '固定',
    reload: '刷新页面',
    themeConfig: '主题配置',
    themeSchema: '主题',
    unpin: '取消固定',
    about: '关于'
  },
  page,
  request: {
    logout: '请求失败后登出用户',
    logoutMsg: '用户状态失效，请重新登录',
    logoutWithModal: '请求失败后弹出模态框再登出用户',
    logoutWithModalMsg: '用户状态失效，请重新登录',
    refreshToken: '请求的token已过期，刷新token',
    tokenExpired: 'token已过期'
  },
  route,
  system: {
    errorReason: '错误原因',
    reload: '重新渲染页面',
    title: '',
    updateCancel: '稍后再说',
    updateConfirm: '立即刷新',
    updateContent: '检测到系统有新版本发布，是否立即刷新页面？',
    updateTitle: '系统版本更新通知'
  },
  theme,
  license: {
    labels: {
      build_number: '编译版本号',
      build_time: '编译时间',
      build_snapshot: '编译快照',
      lucene_version: 'Lucene 版本',
      version: '版本',
      license_type: '授权类型',
      max_nodes: '最大节点数',
      issue_to: '授权对象',
      issue_at: '授权颁发时间',
      expire_at: '授权到期时间',
      organization: '单位名称',
      contact: '联系人',
      email: '单位邮箱',
      phone: '联系电话',
      agree: '同意',
      agreement: '授权协议',
      agreeRequired: '请勾选同意授权协议'
    },
    titles: {
      version: '版本信息',
      license: '授权信息',
      eula: '用户协议'
    },
    actions: {
      buy: '购买咨询',
      trial: '申请试用',
      apply: '更新授权',
      submit: '提交'
    },
    tips: {
      trial: '请如实填写信息，我们将在审核后将授权信息发送至您的邮箱。',
      error: '错误',
      failed: '申请失败，您也可以前往官网申请',
      succeeded: '免费授权申请提交成功，等待系统审核。以下是临时的全功能试用授权。',
      website: '申请地址：',
      timeout: '请求超时或者网络错误'
    }
  },
  assistant: {
    chat: {
      timedout: '请求超时，请稍后再试。',
      greetings: '可以向我提问，与搜索相关或与业务相关的问题。'
    },
    message: {
      logo: '助手图标'
    }
  },
  history_list: {
    search: {
      placeholder: '搜索历史...'
    },
    no_history: '暂无历史记录',
    date: {
      today: '今天',
      yesterday: '昨天',
      last7Days: '过去 7 天',
      last30Days: '过去 30 天'
    },
    menu: {
      rename: '重命名',
      delete: '删除'
    },
    delete_modal: {
      title: '删除会话',
      button: {
        delete: '删除',
        cancel: '取消'
      },
      description: '确定要删除会话 "{{title}}" 吗？此操作无法撤销。'
    }
  },
  permission
};

export default local;
