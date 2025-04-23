import moment from 'moment-timezone';

const FORMAT_GMT = `G[M]TZ`;

export const getGMTString = timeZone => {
  if (!timeZone) return moment().format(FORMAT_GMT);
  return moment()
    .tz(timeZone)
    .format(FORMAT_GMT);
};

export const getDateString = (date, timeZone, dateFormat) => {
  if (!timeZone) return moment(date).format(dateFormat);
  return moment(date)
    .tz(timeZone)
    .format(dateFormat);
};

export const getDateStringWithGMT = (date, timeZone, dateFormat) => {
  if (!timeZone) return moment(date).format(dateFormat);
  return moment(date)
    .tz(timeZone)
    .format(`${dateFormat} (${FORMAT_GMT})`);
};

const MILLISECONDS_IN_SECOND = 1000;
const MILLISECONDS_IN_MINUTE = MILLISECONDS_IN_SECOND * 60;
const MILLISECONDS_IN_HOUR = MILLISECONDS_IN_MINUTE * 60;

export function toMilliseconds(units, value) {
  switch (units) {
    case 'h':
      return Math.round(value * MILLISECONDS_IN_HOUR);
    case 'm':
      return Math.round(value * MILLISECONDS_IN_MINUTE);
    case 's':
    default:
      return Math.round(value * MILLISECONDS_IN_SECOND);
  }
}

export function fromMilliseconds(milliseconds) {
  const round = (value) => parseFloat(value.toFixed(2));
  if (milliseconds > MILLISECONDS_IN_HOUR) {
    return {
      units: "h",
      value: round(milliseconds / MILLISECONDS_IN_HOUR),
    };
  }

  if (milliseconds > MILLISECONDS_IN_MINUTE) {
    return {
      units: "m",
      value: round(milliseconds / MILLISECONDS_IN_MINUTE),
    };
  }

  return {
    units: "s",
    value: round(milliseconds / MILLISECONDS_IN_SECOND),
  };
}
