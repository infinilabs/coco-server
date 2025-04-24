import { request } from '../request';

export function fetchApplicationInfo() {
    return request({
        method: 'get',
        url: '/_info'
    });
}