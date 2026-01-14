import { Input } from "antd";
import styles from "./index.module.less";
import { useEffect, useMemo, useState, useRef } from "react";
import { Search } from "lucide-react";
import SearchActions from "./SearchActions";
import Operations from "./Operations";
import Tips, { SUGGESTION_TIPS } from "./suggestions/Tips";
import Keywords, { SUGGESTION_KEYWORDS } from "./suggestions/Keywords";
import FilterFields, { SUGGESTION_FILTER_FIELDS } from "./suggestions/FilterFields";
import FilterValues, { SUGGESTION_FILTER_VALUES } from "./suggestions/FilterValues";
import Filters from "./Filters";
import { calculateCharLength } from "../utils/utils";
import cloneDeep from "lodash/cloneDeep";
import Operators, { SUGGESTION_OPERATORS } from "./Suggestions/Operators";

export function SearchBox(props) {
  const { placeholder, query, filters = [], onSearch, minimize = false, onSuggestion } = props;

  const [currentKeyword, setCurrentKeyword] = useState(query);
  const [currentFilters, setCurrentFilters] = useState([]);
  const [suggestions, setSuggestions] = useState({});
  const [actionType, setActionType] = useState();

  const [mainInputState, setMainInputState] = useState({
    active: false
  });

  const [filterState, setFilterState] = useState({
    type: 'none',
    index: -1
  });

  const isClickingSuggestion = useRef(false);
  const inputRef = useRef(null);
  const textAreaRef = useRef(null);

  const handleSearch = (keyword, filters) => {
    setMainInputState({ active: false });
    setFilterState({ type: 'none', index: -1 });
    setSuggestions({});
    setActionType();
    onSearch(keyword, filters);
  };

  const changeSuggestions = async (keyword) => {
    const formatKeyword = (keyword || '').trim();
    const hasKeyword = formatKeyword.length > 0;
    if (!mainInputState.active && filterState.type === 'none') {
      setSuggestions({});
      return;
    }

    if (mainInputState.active) {
      if (!hasKeyword) {
        setSuggestions({ type: SUGGESTION_TIPS });
        return;
      }

      if (formatKeyword === '/') {
        setSuggestions({
          type: SUGGESTION_FILTER_FIELDS,
        });
        return;
      }

      setSuggestions({
        type: SUGGESTION_KEYWORDS,
      });
      return; 
    }

    if (filterState.type === 'filterInput' || filterState.type === 'filterActive') {
      if (filterState.type === 'filterInput') {
        setSuggestions({
          type: SUGGESTION_FILTER_VALUES,
        });
      } else if (filterState.type === 'filterActive') {
        setSuggestions({
          type: SUGGESTION_OPERATORS,
        });
      }
    }
  };

  const handleMainInputFocus = () => {
    setMainInputState({ active: true });
    setFilterState({ type: 'none', index: -1 });
  };

  const handleMainInputBlur = () => {
    setTimeout(() => {
      if (!isClickingSuggestion.current) {
        setMainInputState({ active: false });
      }
    }, 100);
  };

  const handleFilterInputFocus = (index) => {
    setFilterState({
      type: 'filterInput',
      index: index
    });
  };

  const handleFilterInputBlur = () => {
    setTimeout(() => {
      if (!isClickingSuggestion.current) {
        setFilterState({
          type: 'none',
          index: -1
        });
      }
    }, 100);
  };

  const handleFilterActiveToggle = (index) => {
    if (index === -1) {
      setFilterState({ type: 'none', index: -1 });
      setSuggestions({});
      return;
    }
    const isCurrentActive = filterState.type === 'filterActive' && filterState.index === index;

    if (isCurrentActive) {
      setFilterState({ type: 'none', index: -1 });
      setSuggestions({});
    } else {
      setFilterState({ type: 'filterActive', index: index });
    }
  };

  const handleOperatorItemClick = (item) => {
    const { type, index } = filterState;
    if (type !== 'filterActive' || index === -1 || index >= currentFilters.length) {
      return;
    }

    const newFilters = cloneDeep(currentFilters);
    newFilters[index].operator = item.suggestion;
    setCurrentFilters(newFilters);
  };

  const handleFilterValueItemClick = (item) => {
    const { type, index } = filterState;
    if ((type !== 'filterInput' && type !== 'filterActive') || index === -1 || index >= currentFilters.length) {
      return;
    }

    setCurrentFilters((prev) => {
      const newFilters = cloneDeep(prev);
      if (!newFilters[index].value) {
        newFilters[index].value = []
      }
      const operatorIndex = newFilters[index].value.findIndex((v) => v === item.suggestion)
      if (operatorIndex === -1) {
        newFilters[index].value.push(item.suggestion)
      } else {
        newFilters[index].value.splice(operatorIndex, 1)
      }
      return newFilters;
    });
  };

  const wrapSuggestionClick = (handler) => {
    return (item) => {
      isClickingSuggestion.current = true;
      handler(item);
      setTimeout(() => {
        isClickingSuggestion.current = false;
      }, 200);
    };
  };

  const handleAddFilter = (item) => {
    let index = 0

    setCurrentFilters((prev) => {
      const newFilters = cloneDeep(prev);
      newFilters.push({ field: item, operator: 'or' });
      index = newFilters.length - 1;
      return newFilters;
    });

    setCurrentKeyword((prevKeyword) => {
      if (prevKeyword && prevKeyword.endsWith('/')) {
        return prevKeyword.slice(0, -1);
      }
      return prevKeyword;
    });

    setMainInputState({ active: false });
    setFilterState({ type: 'filterInput', index });
  };

  useEffect(() => {
    setCurrentKeyword(query);
  }, [query]);

  useEffect(() => {
    if (filters) {
      setCurrentFilters(filters);
    }
  }, [JSON.stringify(filters)]);

  useEffect(() => {
    changeSuggestions(currentKeyword);
  }, [currentKeyword, mainInputState, filterState]);

  const handleSuggestionsResult = (res) => {
    if (Array.isArray(res.suggestions)) {
      setSuggestions((prev) => ({
        ...prev,
        data: res.suggestions
      }))
    }
  }

  useEffect(() => {
    if (suggestions.type === SUGGESTION_KEYWORDS) {
      const showSuggestions = calculateCharLength(currentKeyword) < 40;
      if (showSuggestions) {
        onSuggestion(undefined, {
          query: currentKeyword,
          form: 0,
          size: 10
        }, handleSuggestionsResult)
      }
    } else if (suggestions.type === SUGGESTION_FILTER_FIELDS) {
      onSuggestion(suggestions.type, {
        query: currentKeyword,
        form: 0,
        size: 10
      }, handleSuggestionsResult)
    } else if (suggestions.type === SUGGESTION_FILTER_VALUES) {
      const filter = filterState.index >= 0 && filterState.index < currentFilters.length ? currentFilters[filterState.index] : null
      if (filter?.field?.payload?.field_name) {
        onSuggestion(suggestions.type, {
          field_name: filter?.field?.payload?.field_name,
          query: currentKeyword,
          form: 0,
          size: 10
        }, handleSuggestionsResult)
      }
    }
  }, [suggestions.type, currentKeyword, filterState, currentFilters])

  useEffect(() => {
    if (filterState.type !== 'filterInput' && filterState.type !== 'filterActive') {
      const cleanedFilters = cloneDeep(currentFilters).filter(filter => {
        const value = filter.value;
        if (!value || (Array.isArray(value) && value.length === 0)) return false;
        return true;
      });
      if (cleanedFilters.length !== currentFilters.length) {
        setCurrentFilters(cleanedFilters);
      }
    }
  }, [filterState.type, currentFilters]);

  const handleTabKeyDown = (e) => {
    if (e.key === 'Tab') {
      e.preventDefault();

      if (minimize) {
        if (inputRef.current) {
          inputRef.current.focus();
        }
      } else {
        if (textAreaRef.current) {
          textAreaRef.current.focus();
        }
      }
    }
  };

  useEffect(() => {
    document.addEventListener('keydown', handleTabKeyDown);

    return () => {
      document.removeEventListener('keydown', handleTabKeyDown);
    };
  }, [minimize]);

  const searchable = useMemo(() => {
    return (
      (currentKeyword || '').trim().length > 0 ||
      currentFilters.some(filter => !!filter.value && !(Array.isArray(filter.value) && filter.value.length === 0))
    );
  }, [currentKeyword, currentFilters]);

  const renderSuggestions = () => {
    if ((!mainInputState.active && filterState.type === 'none') || !suggestions.type) return null;
    switch (suggestions.type) {
      case SUGGESTION_TIPS:
        return <Tips />;
      case SUGGESTION_KEYWORDS:
        return (
          <Keywords
            keyword={currentKeyword}
            data={suggestions.data || []}
            onItemSelect={(item) => {
              if (item.action) setActionType(item.action)
            }}
            onItemClick={wrapSuggestionClick((item) => handleSearch(item.keyword || currentKeyword, currentFilters))}
          />
        );
      case SUGGESTION_FILTER_FIELDS:
        return (
          <FilterFields
            data={suggestions.data || []}
            onItemClick={wrapSuggestionClick(handleAddFilter)}
          />
        );
      case SUGGESTION_FILTER_VALUES:
        return (
          <FilterValues
            data={suggestions.data || []}
            filter={filterState.index >= 0 && filterState.index < currentFilters.length ? currentFilters[filterState.index] : null}
            onItemClick={wrapSuggestionClick(handleFilterValueItemClick)}
          />
        );
      case SUGGESTION_OPERATORS:
        return <Operators
          onItemClick={wrapSuggestionClick(handleOperatorItemClick)}
        />;
      default:
        return null;
    }
  };

  const showExpandedPanel = mainInputState.active || filterState.type !== 'none';

  const activedInput = (
    <div
      className={`pt-16px pb-12px rounded-12px overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] border border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)] ${styles.searchbox}`}
    >
      <Filters
        className={"mb-14px px-16px"}
        filters={currentFilters}
        onFiltersChange={setCurrentFilters}
        onFilterInputFocus={handleFilterInputFocus}
        onFilterInputBlur={handleFilterInputBlur}
        onFilterActiveToggle={handleFilterActiveToggle}
        focusIndex={filterState.type === 'filterInput' ? filterState.index : -1}
        activeIndex={filterState.type === 'filterActive' ? filterState.index : -1}
      />
      <Input.TextArea
        ref={textAreaRef}
        placeholder={placeholder}
        autoSize={{ minRows: 1, maxRows: 6 }}
        classNames={{ textarea: '!text-16px !px-16px  !mb-14px !bg-transparent' }}
        value={currentKeyword}
        onChange={(e) => setCurrentKeyword(e.target.value)}
        onFocus={handleMainInputFocus}
        onBlur={handleMainInputBlur}
        className={styles.input}
      />
      {renderSuggestions()}
      <div className="flex justify-between items-center px-12px">
        <SearchActions type={actionType} />
        <Operations onSearch={() => handleSearch(currentKeyword, currentFilters)} disabled={!searchable} />
      </div>
    </div>
  );

  return minimize ? (
    <div className={`relative flex w-full h-full items-center justify-center ${styles.searchbox} rounded-8px`}>
      <div className={`w-full h-48px pl-16px pr-10px rounded-12px border border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)]`}>
        <div className="items-center w-full h-full flex gap-8px">
          <Search className="relative top-2px w-16px h-16px flex-shrink-0 text-#999" />
          <div className="flex-shrink-0">
            <Filters
              filters={currentFilters}
              onFiltersChange={setCurrentFilters}
              onFilterInputFocus={handleFilterInputFocus}
              onFilterActiveToggle={handleFilterActiveToggle}
              focusIndex={filterState.type === 'filterInput' ? filterState.index : -1}
              activeIndex={filterState.type === 'filterActive' ? filterState.index : -1}
            />
          </div>
          <div className={`${styles.inputWrapper} w-full`}>
            <Input
              ref={inputRef}
              value={currentKeyword}
              size="large"
              onChange={(e) => setCurrentKeyword(e.target.value)}
              suffix={<Operations size={24} onSearch={() => handleSearch(currentKeyword, currentFilters)} disabled={!searchable} />}
              placeholder={placeholder}
              autoFocus
              className="flex-1 w-full"
              onFocus={() => {
                handleMainInputFocus()
                setTimeout(() => {
                  textAreaRef.current.focus()
                }, 0)
              }}
            />
          </div>
        </div>
      </div>
      <div className={`absolute left-0 top-0 z-10 w-full bg-#fff ${showExpandedPanel ? '' : 'hidden'}`}>
        {activedInput}
      </div>
    </div>
  ) : (
    activedInput
  );
}

export default SearchBox;
