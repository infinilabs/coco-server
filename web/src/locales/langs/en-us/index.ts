import common from './common';
import form from './form';
import page from './page';
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
    about: 'About',
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
    title: "License",
    labels: {
      version: "Version",
      build_time: "Build Time",
      build_number: "Build Number",
    }
  }
};

export default local;
