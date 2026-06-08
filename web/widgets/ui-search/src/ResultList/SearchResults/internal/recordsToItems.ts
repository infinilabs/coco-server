import { recordToListItem } from "./recordToListItem";

import type { SearchResultsItem, SearchResultsRecord } from "../types";

export function recordsToItems(records: SearchResultsRecord[]): SearchResultsItem[] {
  return records.map((record, index) => recordToListItem(record, index));
}

