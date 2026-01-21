import { Input } from "antd";
import styles from "./index.module.less";
import { useEffect, useMemo, useState, useRef, useCallback } from "react"; // 新增 useCallback
import { Search } from "lucide-react";
import SearchActions, { ACTION_TYPE_SEARCH } from "./SearchActions";
import Operations from "./Operations";
import Tips, { SUGGESTION_TIPS } from "./suggestions/Tips";
import Keywords, { SUGGESTION_KEYWORDS } from "./suggestions/Keywords";
import FilterFields, { SUGGESTION_FILTER_FIELDS } from "./suggestions/FilterFields";
import FilterValues, { SUGGESTION_FILTER_VALUES } from "./suggestions/FilterValues";
import Filters from "./Filters";
import { calculateCharLength } from "../utils/utils";
import cloneDeep from "lodash/cloneDeep";
import Operators, { SUGGESTION_OPERATORS } from "./Suggestions/Operators";
import { Attachments } from "@infinilabs/attachments";

const DEFAULT_SUGGESTIONS_SIZE = 5;

export function SearchBox(props) {
  const { placeholder, queryParams, setQueryParams, onSearch, minimize = false, onSuggestion } = props;
  const [currentQueryParams, setCurrentQueryParams] = useState(queryParams);
  const { query, filters = [], action_type, search_type } = currentQueryParams
  const [suggestions, setSuggestions] = useState({});
  const [attachments, setAttachments] = useState([]);
  const [mainInputActive, setMainInputActive] = useState(false);
  const [filterState, setFilterState] = useState({ type: 'none', index: -1 });
  const [attachmentActive, setAttachmentActive] = useState(false);

  const isClickingSuggestion = useRef(false);
  const isClickingSearchAction = useRef(false);
  const inputRef = useRef(null);
  const textAreaRef = useRef(null);
  const expandedInputRef = useRef(null);

  const showExpandedPanel = useMemo(() => {
    return mainInputActive || filterState.type !== 'none' || attachmentActive;
  }, [mainInputActive, filterState.type, attachmentActive]);

  const searchable = useMemo(() => {
    return (
      (query || '').trim().length > 0 ||
      filters.some(filter => !!filter.value && !(Array.isArray(filter.value) && filter.value.length === 0))
    );
  }, [query, filters]);

  const handleSearchActionClick = () => {
    isClickingSearchAction.current = true;
  };

  const handleSearchActionDropdownClose = useCallback(() => {
    isClickingSearchAction.current = false;
    setTimeout(() => {
      if (expandedInputRef.current) {
        expandedInputRef.current.focus();
      } else if (textAreaRef.current) {
        textAreaRef.current.focus();
      } else if (inputRef.current) {
        inputRef.current.focus();
      }
    }, 50);
  }, []);

  const handleQueryParamsChange = (field, value) => {
    setCurrentQueryParams((prev) => {
      const newQueryParams = cloneDeep(prev);
      newQueryParams[field] = value;
      return newQueryParams
    })
  }

  const handleSearch = (query, filters, actionType, searchType) => {
    let shouldAsk = false
    let shouldAgg = false
    if (query !== queryParams?.query || JSON.stringify(filters) !== JSON.stringify(queryParams?.filters)) {
      shouldAsk = true
      shouldAgg = true
    }
    const newFilter = cloneDeep(queryParams?.filter)
    if (Array.isArray(filters) && filters.length > 0) {
      filters.forEach((item) => {
        const field = item.field?.payload?.field_name
        if (item.field?.payload?.field_name && item.value) {
          newFilter[field] = Array.isArray(item.value) ? item.value : [item.value]
        }
      })
    }
    onSearch({ query, filter: newFilter, action_type: actionType, search_type: searchType, mode: actionType !== ACTION_TYPE_SEARCH ? 'chat' : 'search' }, shouldAsk, shouldAgg);
    setMainInputActive(false);
    setFilterState({ type: 'none', index: -1 });
    setAttachmentActive(false)
    setSuggestions({});
  };

  const changeSuggestions = (keyword) => {
    const formatKeyword = (keyword || '').trim();
    const hasKeyword = formatKeyword.length > 0;

    if (!mainInputActive && filterState.type === 'none') {
      setSuggestions({});
      return;
    }

    let suggestionType = null;
    if (mainInputActive) {
      if (!hasKeyword) {
        suggestionType = SUGGESTION_TIPS;
      } else if (formatKeyword === '/') {
        suggestionType = SUGGESTION_FILTER_FIELDS;
      } else {
        suggestionType = SUGGESTION_KEYWORDS;
      }
    } else if (filterState.type === 'filterInput') {
      suggestionType = SUGGESTION_FILTER_VALUES;
    } else if (filterState.type === 'filterActive') {
      suggestionType = SUGGESTION_OPERATORS;
    }

    setSuggestions({
      type: suggestionType,
      from: 0,
      size: DEFAULT_SUGGESTIONS_SIZE
    });
  };

  const handleSuggestionItemClick = (handler) => {
    return (item) => {
      isClickingSuggestion.current = true;
      handler(item);
      setTimeout(() => (isClickingSuggestion.current = false), 200);
    };
  };

  const handleAddFilter = (item) => {
    const newFilters = cloneDeep(filters);
    newFilters.push({ field: item, operator: 'or' });
    handleQueryParamsChange('filters', newFilters)
    handleQueryParamsChange('query', query?.endsWith('/') ? query.slice(0, -1) : query)
    setFilterState({ type: 'filterInput', index: newFilters.length - 1 });
    setMainInputActive(false);
  };

  const handleOperatorChange = (item) => {
    const { index } = filterState;
    if (filterState.type !== 'filterActive' || index === -1 || index >= filters.length) return;

    const newFilters = cloneDeep(filters);
    newFilters[index].operator = item.suggestion;
    handleQueryParamsChange('filters', newFilters)
  };

  const handleFilterValueToggle = (item) => {
    const { index } = filterState;
    if ((filterState.type !== 'filterInput' && filterState.type !== 'filterActive') || index === -1 || index >= filters.length) return;

    const newFilters = cloneDeep(filters);
    const filter = newFilters[index];
    if (!filter.value) filter.value = [];

    const valueIndex = filter.value.findIndex(v => v === item.suggestion);
    if (valueIndex === -1) {
      filter.value.push(item.suggestion);
    } else {
      filter.value.splice(valueIndex, 1);
    }

    handleQueryParamsChange('filters', newFilters)
  };

  const handleFilterActiveToggle = (index) => {
    if (index === -1) {
      setFilterState({ type: 'none', index: -1 });
      setSuggestions({});
      return;
    }

    const isCurrentActive = filterState.type === 'filterActive' && filterState.index === index;
    setFilterState(isCurrentActive
      ? { type: 'none', index: -1 }
      : { type: 'filterActive', index });

    if (isCurrentActive) setSuggestions({});
  };

  const handleInputFocus = () => {
    setMainInputActive(true);
    setFilterState({ type: 'none', index: -1 });
    setTimeout(() => {
      const textareaDom = expandedInputRef.current?.resizableTextArea?.textArea;
      if (textareaDom) {
        textareaDom.focus();
        const len = textareaDom.value.length;
        textareaDom.setSelectionRange(len, len);
      }
    }, 0);
  };

  const handleInputBlur = () => {
    setTimeout(() => {
      if (!isClickingSuggestion.current && !isClickingSearchAction.current) {
        setMainInputActive(false);
      }
    }, 100);
  };

  const handleFilterInputFocus = (index) => {
    setFilterState({ type: 'filterInput', index });
  };

  const handleFilterInputBlur = () => {
    setTimeout(() => {
      if (!isClickingSuggestion.current) setFilterState({ type: 'none', index: -1 });
    }, 100);
  };

  const handleSuggestionsResult = (res) => {
    setSuggestions(prev => ({
      ...prev,
      data: Array.isArray(res?.suggestions) ? res.suggestions : []
    }));
  };

  const renderSuggestions = () => {
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
            onItemSelect={(item) => {
              handleQueryParamsChange('action_type', item.action || ACTION_TYPE_SEARCH)
            }}
            onItemClick={handleSuggestionItemClick((item) => handleSearch(item.suggestion || query, filters, action_type, search_type))}
          />
        );
      case SUGGESTION_FILTER_FIELDS:
        return (
          <FilterFields
            data={data}
            onItemClick={handleSuggestionItemClick(handleAddFilter)}
            loadNext={() => setSuggestions(prev => ({
              ...prev,
              from: prev.from + DEFAULT_SUGGESTIONS_SIZE
            }))}
          />
        );
      case SUGGESTION_FILTER_VALUES:
        return (
          <FilterValues
            data={data}
            filter={filters[filterState.index] || null}
            onItemClick={handleSuggestionItemClick(handleFilterValueToggle)}
          />
        );
      case SUGGESTION_OPERATORS:
        return (
          <Operators onItemClick={handleSuggestionItemClick(handleOperatorChange)} />
        );
      default:
        return null;
    }
  };

  const renderTextArea = (ref, className = "", onBlur) => (
    <Input.TextArea
      ref={ref}
      placeholder={placeholder}
      autoSize={{ minRows: 1, maxRows: 6 }}
      classNames={{ textarea: '!text-16px !px-16px !mb-14px !bg-transparent' }}
      value={query}
      onChange={(e) => handleQueryParamsChange('query', e.target.value)}
      onFocus={handleInputFocus}
      onBlur={onBlur}
      className={`${styles.input} ${className}`}
    />
  );

  const renderActionBar = () => (
    <div className="flex justify-between items-center px-12px">
      <SearchActions
        actionType={action_type}
        searchType={search_type}
        onSearchTypeChange={(type) => handleQueryParamsChange('search_type', type)}
        onButtonClick={handleSearchActionClick}
        onDropdownClose={handleSearchActionDropdownClose} // 传递关闭回调
      />
      <Operations
        attachments={attachments}
        setAttachments={(attachments) => {
          setAttachments(attachments)
          setAttachmentActive(attachments.length > 0)
        }}
        onSearch={() => handleSearch(query, filters, action_type, search_type)}
        disabled={!searchable}
      />
    </div>
  );

  const renderExpandedPanel = () => {
    return (
      <div className={`absolute left-0 top-0 z-100 w-full ${showExpandedPanel ? '' : 'h-0 overflow-hidden'}`}>
        <div className={`py-12px rounded-12px bg-white overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] border border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)] `}>
          {
            attachments.length > 0 && (
              <div className="mb-14px px-16px">
                <Attachments data={attachments} onItemRemove={(item) => {
                  const index = attachments.findIndex((a) => a.id === item.id)
                  if (index !== -1) {
                    const newAttachments = cloneDeep(attachments)
                    newAttachments.splice(index, 1)
                    setAttachments(newAttachments)
                  }
                }} />
              </div>
            )
          }
          <Filters
            className="mb-14px px-16px"
            filters={filters}
            onFiltersChange={(filters) => handleQueryParamsChange('filters', filters)}
            onFilterInputFocus={handleFilterInputFocus}
            onFilterInputBlur={handleFilterInputBlur}
            onFilterActiveToggle={handleFilterActiveToggle}
            focusIndex={filterState.type === 'filterInput' ? filterState.index : -1}
            activeIndex={filterState.type === 'filterActive' ? filterState.index : -1}
          />
          {renderTextArea(expandedInputRef, '', handleInputBlur)}
          {renderSuggestions()}
          {renderActionBar()}
        </div>
      </div>
    );
  };

  useEffect(() => {
    setCurrentQueryParams({
      ...(queryParams || {}),
      query: queryParams?.query || '',
      filters: Array.isArray(queryParams?.filters) ? queryParams.filters : []
    })
  }, [JSON.stringify(queryParams)])

  useEffect(() => {
    changeSuggestions(query);
  }, [query, mainInputActive, filterState]);

  useEffect(() => {
    if (filterState.type !== 'filterInput' && filterState.type !== 'filterActive') {
      const cleanedFilters = cloneDeep(filters).filter(filter => {
        const value = filter.value;
        return !!value && !(Array.isArray(value) && value.length === 0);
      });
      if (cleanedFilters.length !== filters.length) {
        handleQueryParamsChange('filters', cleanedFilters)
      }
    }
  }, [filterState.type, filters]);

  useEffect(() => {
    const { type, from = 0, size = DEFAULT_SUGGESTIONS_SIZE } = suggestions;
    if (!type || !onSuggestion) return;

    let suggestionParams = { from, size };
    switch (type) {
      case SUGGESTION_KEYWORDS:
        if (calculateCharLength(query) < 40) {
          suggestionParams.query = query;
          onSuggestion(undefined, suggestionParams, handleSuggestionsResult);
        }
        break;
      case SUGGESTION_FILTER_FIELDS:
        suggestionParams.query = query?.endsWith('/') ? query.slice(0, -1) : query;
        onSuggestion(type, suggestionParams, handleSuggestionsResult);
        break;
      case SUGGESTION_FILTER_VALUES:
        const filter = filters[filterState.index];
        if (filter?.field?.payload?.field_name) {
          suggestionParams = {
            ...suggestionParams,
            field_name: filter.field.payload.field_name,
            query: query
          };
          onSuggestion(type, suggestionParams, handleSuggestionsResult);
        }
        break;
    }
  }, [suggestions.type, suggestions.from, suggestions.size, query, filterState, filters, onSuggestion]);

  useEffect(() => {
    const handleTabKeyDown = (e) => {
      if (e.key === 'Tab') {
        e.preventDefault();
        expandedInputRef.current?.focus();
      }
    };

    document.addEventListener('keydown', handleTabKeyDown);
    return () => document.removeEventListener('keydown', handleTabKeyDown);
  }, []);

  return (
    <div className={`
      ${styles.searchbox}
      relative w-full rounded-12px 
      ${showExpandedPanel ? '' : 'border'} 
    border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)] 
      ${minimize ? 'h-48px' : `h-105px ${showExpandedPanel ? '' : 'shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]'}`}
      ${!minimize ? styles.gradientBorder : ''}
      bg-[rgb(var(--ui-search--layout-bg-color))]
    `}>
      {minimize ? (
        <div className="px-12px items-center w-full h-full flex gap-8px">
          {
            filters.length > 0 && (
              <div className="flex-shrink-0">
                <Filters
                  filters={filters}
                  onFiltersChange={(filters) => handleQueryParamsChange('filters', filters)}
                  onFilterInputFocus={handleFilterInputFocus}
                  onFilterActiveToggle={handleFilterActiveToggle}
                  focusIndex={-1}
                  activeIndex={-1}
                />
              </div>
            )
          }
          <div className={`${styles.inputWrapper} w-full`}>
            <Input
              ref={inputRef}
              value={query}
              size="large"
              onChange={(e) => handleQueryParamsChange('query', e.target.value)}
              suffix={<Operations size={24} onSearch={() => handleSearch(query, filters, action_type, search_type)} disabled={!searchable} />}
              placeholder={placeholder}
              className="flex-1 w-full"
              onFocus={handleInputFocus}
              onBlur={() => { return }}
            />
          </div>
        </div>
      ) : (
        <div className="py-12px">
          {renderTextArea(textAreaRef, '!mb-14px')}
          {renderActionBar()}
        </div>
      )}
      {renderExpandedPanel()}
    </div>
  );
}

export default SearchBox;