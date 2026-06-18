import Tips, { SUGGESTION_TIPS } from "./Tips";
import Keywords, { SUGGESTION_KEYWORDS } from "./Keywords";
import FilterFields, { SUGGESTION_FILTER_FIELDS } from "./FilterFields";
import FilterValues, { SUGGESTION_FILTER_VALUES } from "./FilterValues";
import Operators, { SUGGESTION_OPERATORS } from "./Operators";
import { ACTION_TYPE_SEARCH } from "../ActionBar/SearchActions";
import { type FC, type RefObject, memo } from "react";

interface SuggestionsProps {
  readonly suggestions: { type?: string; data?: any[] };
  readonly onLoadNext?: () => void;
  readonly query?: string;
  readonly filters?: any[];
  readonly action_type?: string;
  readonly search_type?: string;
  readonly filterState: { type: string; index: number };
  readonly mainInputActive: boolean;
  readonly handleQueryParamsChange: (field: string, value: any) => void;
  readonly handleSuggestionItemClick: (handler: (item: any) => void) => (item: any) => void;
  readonly handleSearch: (query: string, filters: any[], actionType: string | undefined, searchType: string) => void;
  readonly handleAddFilter: (item: any) => void;
  readonly handleFilterValueToggle: (item: any) => void;
  readonly handleOperatorChange: (item: any) => void;
  readonly handleFilterComplete: () => void;
  readonly turnToChat?: (item: any) => void;
  readonly language?: string;
  readonly settings?: Record<string, any>;
  readonly resetKey?: string;
  readonly keyboardRootRef?: RefObject<HTMLElement | null>;
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
