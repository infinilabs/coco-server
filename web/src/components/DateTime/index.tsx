import { Tooltip } from "antd";
import dayjs from "dayjs";
import relativeTime from 'dayjs/plugin/relativeTime';
import utc from 'dayjs/plugin/utc';
import { isString } from "lodash";

dayjs.extend(relativeTime);
dayjs.extend(utc);

export const isWithin7Days = (value: string | number) => {
    const targetDate = dayjs(value);
    const now = dayjs();
    const diffInMs = now.diff(targetDate);
    return diffInMs <= 7 * 24 * 60 * 60 * 1000; 
}

export const formatDate = (value: string | number) => {
    const targetDate = dayjs(value);
    const dateTime = targetDate.format('YYYY-MM-DD HH:mm:ss');
    const timezone = `(GMT${targetDate.format('ZZ')})`;
    return `${dateTime} ${timezone}`;
}

export const displayDate = (value: string | number) => {
    if (isWithin7Days(value)) {
        return dayjs(value).fromNow()
    }
    return isWithin7Days(value) ? dayjs(value).fromNow() : formatDate(value);
}

export default function DateTime(props: { value: string | number, showTooltip?: boolean }) {
    const { value, showTooltip = true } = props;
    if (!value || !dayjs(value).isValid()) return "-"
    
    const formatValue = formatDate(value)

    if (showTooltip) {
        return (
            <Tooltip title={isString(value) ? value : formatDate(value)}>
                {formatValue}
            </Tooltip>
        )
    }

    return formatValue
}