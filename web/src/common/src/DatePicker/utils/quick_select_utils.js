import moment from 'moment';
import dateMath from '@elastic/datemath';
import { relativeUnitsFromLargestToSmallest } from './relative_options';
import { DATE_MODES } from './date_modes';
import _isString from 'lodash/isString';

const LAST = 'last';
const NEXT = 'next';

const isNow = value => value === DATE_MODES.NOW;

export const parseTimeParts = (start, end) => {
  const results = {
    timeTense: LAST,
    timeUnits: 'm',
    timeValue: 15,
  };

  const value = isNow(start) ? end : start;

  const matches = _isString(value) && value.match(/now(([-+])(\d+)([smhdwMy])(\/[smhdwMy])?)?/);

  if (!matches) {
    return results;
  }

  const operator = matches[2];
  const matchedTimeValue = matches[3];
  const timeUnits = matches[4];

  if (matchedTimeValue && timeUnits && operator) {
    return {
      timeTense: operator === '+' ? NEXT : LAST,
      timeUnits,
      timeValue: parseInt(matchedTimeValue, 10),
    };
  }

  const duration = moment.duration(moment().diff(dateMath.parse(value)));
  let unitOp = '';
  for (let i = 0; i < relativeUnitsFromLargestToSmallest.length; i++) {
    const as = duration.as(relativeUnitsFromLargestToSmallest[i]);
    if (as < 0) {
      unitOp = '+';
    }
    if (Math.abs(as) > 1) {
      return {
        timeValue: Math.round(Math.abs(as)),
        timeUnits: relativeUnitsFromLargestToSmallest[i],
        timeTense: unitOp === '+' ? NEXT : LAST,
      };
    }
  }

  return results;
};
