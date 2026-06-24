import dayjs, { type ConfigType } from "dayjs";
import relativeTime from 'dayjs/plugin/relativeTime';
dayjs.extend(relativeTime);

export const isWithin7Days = (date: ConfigType) => {
    const targetDate = dayjs(date);
    const now = dayjs();
    const diffInMs = now.diff(targetDate);
    return diffInMs <= 7 * 24 * 60 * 60 * 1000; 
}

export const formatDateWithRelative = (date: ConfigType) => {
    const targetDate = dayjs(date);
    if (!targetDate.isValid()) {
        return undefined;
    }
    if (isWithin7Days(date)) {
        return targetDate.fromNow();
    }
    const dateTime = targetDate.format('YYYY-MM-DD HH:mm:ss');
    const timezone = `(GMT${targetDate.format('ZZ')})`;
    return `${dateTime} ${timezone}`;
}

export const formatDate = (date: ConfigType) => {
    const targetDate = dayjs(date);
    if (!targetDate.isValid()) {
        return undefined;
    }
    const dateTime = targetDate.format('YYYY-MM-DD HH:mm:ss');
    const timezone = `(GMT${targetDate.format('ZZ')})`;
    return `${dateTime} ${timezone}`;
}

export const calcFixedBucketCount = (start: number, end: number) => {
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return 60;
  }

  const duration = end - start;
  return duration >= 7 * 24 * 60 * 60 * 1000 ? 30 : 60;
};