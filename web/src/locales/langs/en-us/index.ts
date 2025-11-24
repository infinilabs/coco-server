import common from './common';
import form from './form';
import page from './page';
import permission from './permission';
import request from './request';
import route from './route';
import theme from './theme';

const local: App.I18n.Schema['translation'] = {
  common,
  datatable: {
    itemCount: 'Total {total} items'
  },
  dropdown: {
    closeAll: 'Close All',
    closeCurrent: 'Close Current',
    closeLeft: 'Close Left',
    closeOther: 'Close Other',
    closeRight: 'Close Right'
  },
  form,
  icon: {
    collapse: 'Collapse Menu',
    expand: 'Expand Menu',
    fullscreen: 'Fullscreen',
    fullscreenExit: 'Exit Fullscreen',
    lang: 'Switch Language',
    pin: 'Pin',
    reload: 'Reload Page',
    themeConfig: 'Theme Configuration',
    themeSchema: 'Theme',
    unpin: 'Unpin',
    about: 'About'
  },
  page,
  request,
  route,
  system: {
    errorReason: 'Cause Error',
    reload: 'Reload Page',
    title: 'Coco Server',
    updateCancel: 'Later',
    updateConfirm: 'Refresh immediately',
    updateContent: 'A new version of the system has been detected. Do you want to refresh the page immediately?',
    updateTitle: 'System Version Update Notification'
  },
  theme,
  license: {
    labels: {
      build_number: 'Build Number',
      build_snapshot: 'Build Snapshot',
      build_time: 'Build Time',
      lucene_version: 'Lucene Version',
      version: 'Version',
      license_type: 'License Type',
      max_nodes: 'Max Nodes',
      issue_to: 'Issue To',
      issue_at: 'Issue At',
      expire_at: 'Expire At',
      organization: 'Organization',
      contact: 'Contact',
      email: 'Email',
      phone: 'Phone',
      agree: 'I have read the ',
      agreement: 'agreement',
      agreeRequired: 'Please check the agreement'
    },
    titles: {
      version: 'Version',
      license: 'License',
      eula: 'EULA'
    },
    actions: {
      buy: 'Purchase',
      trial: 'Free Trial',
      apply: 'Apply License',
      submit: 'Submit'
    },
    tips: {
      trial: 'Please provide the information accurately, we will email you the license information after verification.',
      error: 'Error message',
      failed: 'Submission failed. You can try visiting our website to apply for a free license.',
      succeeded:
        'You have successfully applied for a free trial license. The system will review your application soon. Meanwhile, you can use this temporary trial license with full features.',
      website: 'Please Visit:',
      timeout: 'Request timeout or network problem'
    }
  },
  permission
};

export default local;
