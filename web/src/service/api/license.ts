import { request } from '../request';

export function fetchLicense() {
  return request({
    method: 'get',
    url: `/_license/info`
  });
}

export function applyLicense(code: string) {
  return request({
    data: {
      license: code
    },
    method: 'post',
    url: `/_license/apply`
  });
}

export function requestTrialLicense(body: any) {
  const { locale = 'en-US', ...rest } = body;
  return request({
    data: rest,
    method: 'post',
    url: `https://api.infini.cloud/_license/request_trial?lang=${locale}`
  });
}
