import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { localStg } from '@/utils/storage';

import locales from './locale';

/** Setup plugin i18n */
export function setupI18n() {
  i18n.use(initReactI18next).init({
    interpolation: {
      escapeValue: false
    },
    lng: localStg.get('lang') || 'zh-CN',
    resources: locales
  });
}

export const $t = i18n.t;

export function setLng(locale: App.I18n.LangType) {
  i18n.changeLanguage(locale);
}
