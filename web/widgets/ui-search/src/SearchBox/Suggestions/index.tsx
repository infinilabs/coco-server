import Tips, { SUGGESTION_TIPS } from "./Tips";
import Keywords, { SUGGESTION_KEYWORDS } from "./Keywords";
import FilterFields, { SUGGESTION_FILTER_FIELDS } from "./FilterFields";
import FilterValues, { SUGGESTION_FILTER_VALUES } from "./FilterValues";
import Operators, { SUGGESTION_OPERATORS } from "./Operators";
import { ACTION_TYPE_SEARCH } from "../ActionBar/SearchActions";
import { memo, type FC, type RefObject } from "react";

interface SuggestionsProps {
  suggestions: { type?: string; data?: any[] };
  onLoadNext?: () => void;
  query?: string;
  filters?: any[];
  action_type?: string;
  search_type?: string;
  filterState: { type: string; index: number };
  mainInputActive: boolean;
  handleQueryParamsChange: (field: string, value: any) => void;
  handleSuggestionItemClick: (handler: (item: any) => void) => (item: any) => void;
  handleSearch: (query: string, filters: any[], actionType: string | undefined, searchType: string) => void;
  handleAddFilter: (item: any) => void;
  handleFilterValueToggle: (item: any) => void;
  handleOperatorChange: (item: any) => void;
  handleFilterComplete: () => void;
  turnToChat?: (item: any) => void;
  language?: string;
  settings?: Record<string, any>;
  resetKey?: string;
  keyboardRootRef?: RefObject<HTMLElement | null>;
}

const Suggestions: FC<SuggestionsProps> = ({
  suggestions,
  onLoadNext,
  query,
  filters = [],
  action_type,
  search_type,
  filterState,
  mainInputActive,
  handleQueryParamsChange,
  handleSuggestionItemClick,
  handleSearch,
  handleAddFilter,
  handleFilterValueToggle,
  handleOperatorChange,
  handleFilterComplete,
  turnToChat,
  language,
  settings,
  resetKey,
  keyboardRootRef
}) => {

  const { type, data = [] } = suggestions;
  if (!type || (!mainInputActive && filterState.type === 'none')) return null;

  switch (type) {
    case SUGGESTION_TIPS:
      return <Tips />;
    case SUGGESTION_KEYWORDS:
      return (
        <Keywords
          keyword={query}
          data={data}
          action_type={action_type}
          settings={settings}
          onItemSelect={(item) => handleQueryParamsChange('action_type', item.action || ACTION_TYPE_SEARCH)}
          onItemClick={handleSuggestionItemClick((item) => {
            if (item.action === 'deepthink' || item.action === 'deepresearch') {
              turnToChat?.(item);
              return;
            }
            const nextQuery = item.suggestion || query;
            handleQueryParamsChange('query', nextQuery);
            handleSearch(nextQuery, filters, item.action || action_type, search_type || '')
          })}
          language={language}
          keyboardRootRef={keyboardRootRef}
        />
      );
    case SUGGESTION_FILTER_FIELDS:
      return (
        <FilterFields
          data={data}
          onItemClick={handleSuggestionItemClick(handleAddFilter)}
          loadNext={onLoadNext}
          language={language}
          resetKey={resetKey}
          keyboardRootRef={keyboardRootRef}
        />
      );
    case SUGGESTION_FILTER_VALUES:
      return (
        <FilterValues
          data={data}
          filter={filters[filterState.index] || null}
          onItemClick={handleSuggestionItemClick(handleFilterValueToggle)}
          onComplete={handleFilterComplete}
          language={language}
          loadNext={onLoadNext}
          resetKey={resetKey}
          keyboardRootRef={keyboardRootRef}
        />
      );
    case SUGGESTION_OPERATORS:
      return <Operators currentOperator={filters[filterState.index]?.operator} onItemClick={handleSuggestionItemClick(handleOperatorChange)} language={language} keyboardRootRef={keyboardRootRef} />;
    default:
      return null;
  }
}

export default memo(Suggestions);
