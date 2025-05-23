import dateMath from '@elastic/datemath';
import moment from 'moment';

import { relativeUnitsFromLargestToSmallest } from './relative_options';
import _get from 'lodash/get';
import _isString from 'lodash/isString';

const ROUND_DELIMETER = '/';

export function parseRelativeParts(value) {
  const matches = _isString(value) && value.match(/now(([-+])([0-9]+)([smhdwMy])(\/[smhdwMy])?)?/);

  const operator = matches && matches[2];
  const count = matches && matches[3];
  const unit = matches && matches[4];
  const roundBy = matches && matches[5];

  if (count && unit) {
    const isRounded = roundBy ? true : false;
    const roundUnit = isRounded && roundBy ? roundBy.replace(ROUND_DELIMETER, '') : undefined;
    return {
      count: parseInt(count, 10),
      unit: operator === '+' ? `${unit}+` : unit,
      round: isRounded,
      ...(roundUnit ? { roundUnit } : {}),
    };
  }

  const results = { count: 0, unit: 's', round: false };
  const duration = moment.duration(moment().diff(dateMath.parse(value)));
  let unitOp = '';
  for (let i = 0; i < relativeUnitsFromLargestToSmallest.length; i++) {
    const asRelative = duration.as(relativeUnitsFromLargestToSmallest[i]);
    if (asRelative < 0) unitOp = '+';
    if (Math.abs(asRelative) > 1) {
      results.count = Math.round(Math.abs(asRelative));
      results.unit = relativeUnitsFromLargestToSmallest[i] + unitOp;
      results.round = false;
      break;
    }
  }
  return results;
}

export const toRelativeStringFromParts = relativeParts => {
  const count = _get(relativeParts, 'count', 0);
  const isRounded = _get(relativeParts, 'round', false);

  if (count === 0 && !isRounded) {
    return 'now';
  }

  const matches = _get(relativeParts, 'unit', 's').match(/([smhdwMy])(\+)?/);
  const unit = matches[1];
  const operator = matches && matches[2] ? matches[2] : '-';
  const round = isRounded ? `${ROUND_DELIMETER}${unit}` : '';

  return `now${operator}${count}${unit}${round}`;
};
