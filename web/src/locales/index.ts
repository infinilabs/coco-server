import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { localStg } from '@/utils/storage';

import locales from './locale';

/** Setup plugin i18n */
export function setupI18n() {
  const locale = localStg.get('lang') || 'en-US';

  i18n.use(initReactI18next).init({
    interpolation: {
      escapeValue: false
    },
    lng: locale,
    resources: locales,
    nsSeparator: '___'
  });

  updateDocumentLang(locale);
}

export const $t = i18n.t;

export function setLng(locale: App.I18n.LangType) {
  i18n.changeLanguage(locale);
  updateDocumentLang(locale);
}

function updateDocumentLang(locale: App.I18n.LangType) {
  document.documentElement.lang = locale;
}
