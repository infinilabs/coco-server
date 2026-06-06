import dayjs, { type ConfigType } from "dayjs";
import relativeTime from 'dayjs/plugin/relativeTime';
dayjs.extend(relativeTime);

export const isWithin7Days = (date: ConfigType) => {
    const targetDate = dayjs(date);
    const now = dayjs();
    const diffInMs = now.diff(targetDate);
    return diffInMs <= 7 * 24 * 60 * 60 * 1000; 
}

export const formatDate = (date: ConfigType) => {
    const targetDate = dayjs(date);
    return isWithin7Days(date) ? targetDate.fromNow() : targetDate.format('YYYY-MM-DD HH:mm:ss')
}