import Tips, { SUGGESTION_TIPS } from "./Tips";
import Keywords, { SUGGESTION_KEYWORDS } from "./Keywords";
import FilterFields, { SUGGESTION_FILTER_FIELDS } from "./FilterFields";
import FilterValues, { SUGGESTION_FILTER_VALUES } from "./FilterValues";
import Operators, { SUGGESTION_OPERATORS } from "./Operators";
import { ACTION_TYPE_SEARCH } from "../ActionBar/SearchActions";

export default function Suggestions({
  suggestions,
  onLoadNext,
  query,
  filters,
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
}) {
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
          onItemSelect={(item) => handleQueryParamsChange('action_type', item.action || ACTION_TYPE_SEARCH)}
          onItemClick={handleSuggestionItemClick((item) => handleSearch(item.suggestion || query, filters, item.action || action_type, search_type))}
        />
      );
    case SUGGESTION_FILTER_FIELDS:
      return (
        <FilterFields
          data={data}
          onItemClick={handleSuggestionItemClick(handleAddFilter)}
          loadNext={onLoadNext}
        />
      );
    case SUGGESTION_FILTER_VALUES:
      return (
        <FilterValues
          data={data}
          filter={filters[filterState.index] || null}
          onItemClick={handleSuggestionItemClick(handleFilterValueToggle)}
          onComplete={handleFilterComplete}
        />
      );
    case SUGGESTION_OPERATORS:
      return <Operators currentOperator={filters[filterState.index]?.operator} onItemClick={handleSuggestionItemClick(handleOperatorChange)} />;
    default:
      return null;
  }
}
