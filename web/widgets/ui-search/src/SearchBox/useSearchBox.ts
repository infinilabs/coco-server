import { useEffect, useMemo, useState, useRef, useCallback } from "react";
import { ACTION_TYPE_SEARCH, ACTION_TYPE_SEARCH_KEYWORD } from "./ActionBar/SearchActions";
import { SUGGESTION_TIPS } from "./Suggestions/Tips";
import { SUGGESTION_KEYWORDS } from "./Suggestions/Keywords";
import { SUGGESTION_FILTER_FIELDS } from "./Suggestions/FilterFields";
import { SUGGESTION_FILTER_VALUES } from "./Suggestions/FilterValues";
import { SUGGESTION_OPERATORS } from "./Suggestions/Operators";
import { calculateCharLength } from "../utils/utils";
import { isEmpty } from "lodash";

export const DEFAULT_SUGGESTIONS_SIZE = 5;

// Extract colon field pattern: "word:" at end of text before cursor
function extractColonFieldQuery(query: string, cursorPosition: number) {
  if (!query || cursorPosition <= 0) return null;
  const textBeforeCursor = query.substring(0, cursorPosition);
  const match = textBeforeCursor.match(/(\S+):$/);
  return match ? match[1] : null;
}

// Extract slash+keyword pattern: "/word" at end of text before cursor
function extractSlashFieldQuery(query: string, cursorPosition: number) {
  if (!query || cursorPosition <= 0) return null;
  const textBeforeCursor = query.substring(0, cursorPosition);
  const match = textBeforeCursor.match(/\/(\S+)$/);
  return match ? match[1] : null;
}

interface UseSearchBoxParams {
  queryParams?: any;
  onSearch?: (...args: any[]) => void;
  onSuggestion?: (...args: any[]) => void;
  filterFieldsMeta?: Record<string, any>;
  onUpload?: (files: File[], cb: (res: any) => void) => void;
  attachments?: any[];
  setAttachments?: (attachments: any) => void;
}

export default function useSearchBox({ 
  queryParams, 
  onSearch, 
  onSuggestion, 
  filterFieldsMeta = {}, 
  onUpload,
  attachments = [],
  setAttachments, 
}: UseSearchBoxParams) {
  const [currentQueryParams, setCurrentQueryParams] = useState<any>(queryParams);
  const { query, filter = {}, filters = [], action_type, search_type = ACTION_TYPE_SEARCH_KEYWORD } = currentQueryParams;
  const [suggestions, setSuggestions] = useState<any>({});
  const [mainInputActive, setMainInputActive] = useState(false);
  const [filterState, setFilterState] = useState({ type: 'none', index: -1 });
  const [cursorPosition, setCursorPosition] = useState(0);
  const [shouldFocusNewFilter, setShouldFocusNewFilter] = useState(false);
  const [filterSearchValue, setFilterSearchValue] = useState('');

  const isClickingSuggestion = useRef(false);
  const isClickingSearchAction = useRef(false);
  const isSearchTriggered = useRef(false);
  const blurTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastClickOutsideTime = useRef(0);
  const lastMouseDownInsideTime = useRef(0);
  const filterStateRef = useRef(filterState);
  const inputRef = useRef<any>(null);
  const textAreaRef = useRef<any>(null);
  const expandedInputRef = useRef<any>(null);
  // Root container ref of the SearchBox. Used as the reliable boundary for
  // click-outside detection (replaces the fragile `[class*="searchbox"]`
  // selector that depends on CSS-module hashed class names and breaks when
  // the `.searchbox` class is renamed or when multiple instances exist).
  // Typed as HTMLDivElement to match the root <div> it is attached to.
  const rootRef = useRef<HTMLDivElement | null>(null);

  const showExpandedPanel = useMemo(() => {
    return mainInputActive || filterState.type !== 'none';
  }, [mainInputActive, filterState.type]);

  useEffect(() => {
    filterStateRef.current = filterState;
  }, [filterState]);

  const searchable = useMemo(() => {
    return (
      (query || '').trim().length > 0 ||
      filters.some((f: any) => !!f.value && !(Array.isArray(f.value) && f.value.length === 0)) ||
      !isEmpty(filter) || attachments.length > 0
    );
  }, [query, filters, filter, attachments]);

  const isSlashAtCursor = useMemo(() => {
    if (!query || cursorPosition < 0 || cursorPosition > query.length) return false;
    return query.charAt(cursorPosition - 1) === '/' || query.charAt(cursorPosition) === '/';
  }, [query, cursorPosition]);

  // Detect "keyword:" and "/keyword" patterns at cursor for field matching
  const colonFieldQuery = useMemo(() => {
    if (!mainInputActive) return null;
    return extractColonFieldQuery(query, cursorPosition);
  }, [mainInputActive, query, cursorPosition]);

  const slashFieldQuery = useMemo(() => {
    if (!mainInputActive) return null;
    return extractSlashFieldQuery(query, cursorPosition);
  }, [mainInputActive, query, cursorPosition]);

  // Derived: determine which suggestion type to display based on current input context
  const suggestionType = useMemo(() => {
    if (!mainInputActive && filterState.type === 'none') return null;
    if (mainInputActive) {
      if (!(query || '').trim()) return SUGGESTION_TIPS;
      if (isSlashAtCursor) return SUGGESTION_FILTER_FIELDS;
      if (colonFieldQuery || slashFieldQuery) return SUGGESTION_FILTER_FIELDS;
      return SUGGESTION_KEYWORDS;
    }
    if (filterState.type === 'filterInput') return SUGGESTION_FILTER_VALUES;
    if (filterState.type === 'filterActive') return SUGGESTION_OPERATORS;
    return null;
  }, [mainInputActive, filterState.type, query, isSlashAtCursor, colonFieldQuery, slashFieldQuery]);

  const handleSearchActionClick = useCallback(() => {
    isClickingSearchAction.current = true;
  }, []);

  const handleSearchActionDropdownClose = useCallback(() => {
    setTimeout(() => {
      isClickingSearchAction.current = false;
      if (expandedInputRef.current) expandedInputRef.current.focus();
      else if (textAreaRef.current) textAreaRef.current.focus();
      else if (inputRef.current) inputRef.current.focus();
    }, 50);
  }, []);

  const handleQueryParamsChange = useCallback((field: string, value: any) => {
    setCurrentQueryParams((prev: any) => ({ ...prev, [field]: value }));
  }, []);

  // Wrap handleSearch in useCallback so that consumers (triggerSearch,
  // handleFilterComplete, etc.) always reference the latest version and
  // stale-closure bugs are eliminated. Dependencies are only the values
  // that can change between renders; refs and state setters are stable.
  const handleSearch = useCallback((searchQuery: any, searchFilters: any, actionType: any, searchType: any) => {
    if (attachments.length > 0) {
      onSearch?.({
        query: searchQuery,
        attachments: attachments,
        mode: 'chat'
      })
    } else {
      const newFilter: Record<string, any> = {};
      // Rebuild filter from current filters array
      if (Array.isArray(searchFilters) && searchFilters.length > 0) {
        searchFilters.forEach((item: any) => {
          const field = item.field?.field_name;
          if (field && item.value) {
            const key = item.operator === 'not' ? `!${field}` : field;
            const values = Array.isArray(item.value) ? item.value : [item.value];
            newFilter[key] = Array.from(new Set([...(newFilter[key] || []), ...values]));
          }
        });
      }
      onSearch?.({
        query: searchQuery,
        filter: newFilter,
        action_type: actionType,
        search_type: searchType,
        mode: !actionType || actionType === ACTION_TYPE_SEARCH ? 'search' : 'chat'
      }, true, true);
    }
    setMainInputActive(false);
    setFilterState({ type: 'none', index: -1 });
    setSuggestions({});
    isSearchTriggered.current = true;
    setTimeout(() => { isSearchTriggered.current = false; }, 200);
    // Blur inputs to prevent re-triggering focus events
    if (inputRef.current?.input) inputRef.current.input.blur();
    if (expandedInputRef.current?.resizableTextArea?.textArea) {
      expandedInputRef.current.resizableTextArea.textArea.blur();
    }
  }, [attachments, onSearch]);

  const triggerSearch = useCallback(() => {
    handleSearch(query, filters, action_type, search_type);
  }, [handleSearch, query, filters, action_type, search_type]);

  const handleAttachmentUpload = useCallback((files: File[], cb: (res: any) => void) => {
    if (onUpload) onUpload(files, cb);
  }, [onUpload]);

  const handleCursorPositionChange = useCallback((e: any) => {
    setCursorPosition(e.target.selectionStart);
  }, []);

  const handleInputChange = useCallback((e: any) => {
    handleQueryParamsChange('query', e.target.value);
    setCursorPosition(e.target.selectionStart);
  }, [handleQueryParamsChange]);

  const handleSuggestionItemClick = useCallback((handler: any) => {
    return (item: any) => {
      isClickingSuggestion.current = true;
      handler(item);
      setTimeout(() => (isClickingSuggestion.current = false), 200);
    };
  }, []);

  const loadNextSuggestion = useCallback(() => {
    setSuggestions((prev: any) => ({ ...prev, from: prev.type === SUGGESTION_FILTER_VALUES ? prev.from + 10 : prev.from + DEFAULT_SUGGESTIONS_SIZE }));
  }, []);

  const handleAddFilter = useCallback((item: any) => {
    const newFilters = [...filters, { field: item.payload, operator: 'and' }];
    handleQueryParamsChange('filters', newFilters);

    // Remove the trigger text ("/" or "keyword:") from query
    if (colonFieldQuery || slashFieldQuery) {
      // Remove "keyword:" or "/keyword" pattern from query
      const textBeforeCursor = query.substring(0, cursorPosition);
      const textAfterCursor = query.substring(cursorPosition);
      const cleanedBefore = colonFieldQuery ? textBeforeCursor.replace(/\S+:$/, '') : textBeforeCursor.replace(/\/\S*$/, '');
      handleQueryParamsChange('query', cleanedBefore + textAfterCursor);
    } else {
      const chars = query?.split('');
      if (chars && cursorPosition > 0 && chars[cursorPosition - 1] === '/') {
        chars.splice(cursorPosition - 1, 1);
        handleQueryParamsChange('query', chars.join(''));
      } else {
        handleQueryParamsChange('query', query?.endsWith('/') ? query.slice(0, -1) : query);
      }
    }

    setShouldFocusNewFilter(true);
    setFilterState({ type: 'filterInput', index: newFilters.length - 1 });
    setMainInputActive(false);
  }, [filters, colonFieldQuery, slashFieldQuery, query, cursorPosition, handleQueryParamsChange]);

  const handleOperatorChange = useCallback((item: any) => {
    const { index } = filterState;
    if (filterState.type !== 'filterActive' || index === -1 || index >= filters.length) return;

    const newFilters = filters.map((filterItem: any, filterIndex: number) => {
      if (filterIndex !== index) return filterItem;
      return { ...filterItem, operator: item.suggestion };
    });
    handleQueryParamsChange('filters', newFilters);
  }, [filterState, filters, handleQueryParamsChange]);

  const handleFilterValueToggle = useCallback((item: any) => {
    const { index } = filterState;
    if ((filterState.type !== 'filterInput' && filterState.type !== 'filterActive') || index === -1) return;

    setCurrentQueryParams((prev: any) => {
      const prevFilters = prev.filters || [];
      if (index >= prevFilters.length) return prev;

      const currentFilter = prevFilters[index];
      const currentValue = Array.isArray(currentFilter.value) ? currentFilter.value : [];
      let nextValue;

      const valueIndex = currentValue.findIndex((v: any) => v === item.suggestion);
      if (valueIndex === -1) nextValue = [...currentValue, item.suggestion];
      else nextValue = currentValue.filter((v: any) => v !== item.suggestion);

      const nextFilter = { ...currentFilter, value: nextValue };
      const newFilters = prevFilters.map((filterItem: any, filterIndex: number) => {
        if (filterIndex !== index) return filterItem;
        return nextFilter;
      });

      // Single-select: auto-complete after selecting a value
      if (!nextFilter.field?.support_multi_select && nextFilter.value.length > 0) {
        setTimeout(() => {
          setFilterState({ type: 'none', index: -1 });
          setMainInputActive(true);
          if (expandedInputRef.current) expandedInputRef.current.focus();
          setSuggestions({});
        }, 100);
      }

      return { ...prev, filters: newFilters };
    });
  }, [filterState]);

  // Complete filter editing: close panel and trigger search.
  // NOTE: handleSearch is included in the dependency array because it is now
  // a useCallback whose identity can change (e.g. when `attachments` or
  // `onSearch` changes). Without it, the setTimeout closure would capture a
  // stale handleSearch, causing the search to use outdated attachments/filter.
  const handleFilterComplete = useCallback(() => {
    setFilterState({ type: 'none', index: -1 });
    setSuggestions({});
    // Trigger search after a brief delay to allow state to settle
    setTimeout(() => {
      handleSearch(query, filters, action_type, search_type);
    }, 50);
  }, [query, filters, action_type, search_type, handleSearch]);

  // Delete a filter by index
  const handleFilterDelete = useCallback((index: number) => {
    if (index < 0 || index >= filters.length) return;
    const newFilters = filters.filter((_: any, filterIndex: number) => filterIndex !== index);
    handleQueryParamsChange('filters', newFilters);
    if (filterState.index === index) {
      setFilterState({ type: 'none', index: -1 });
      setSuggestions({});
      // Cancel pending blur to prevent panel flicker
      if (blurTimeoutRef.current) {
        clearTimeout(blurTimeoutRef.current);
        blurTimeoutRef.current = null;
      }
      // Keep panel open by activating the main input
      setMainInputActive(true);
      setTimeout(() => {
        if (expandedInputRef.current) expandedInputRef.current.focus();
      }, 50);
    } else if (filterState.index > index) {
      setFilterState(prev => ({ ...prev, index: prev.index - 1 }));
    }
  }, [filters, filterState, handleQueryParamsChange]);

  // Re-enter filter value editing by clicking value area
  const handleFilterValueEdit = useCallback((index: number) => {
    setFilterState({ type: 'filterInput', index });
    setMainInputActive(false);
  }, []);

  const handleFilterActiveToggle = useCallback((index: number) => {
    lastMouseDownInsideTime.current = Date.now();
    if (index === -1) {
      const nextFilterState = { type: 'none', index: -1 };
      filterStateRef.current = nextFilterState;
      setFilterState(nextFilterState);
      setSuggestions({});
      return;
    }

    const isCurrentActive = filterState.type === 'filterActive' && filterState.index === index;
    if (isCurrentActive) {
      const nextFilterState = { type: 'none', index: -1 };
      filterStateRef.current = nextFilterState;
      setFilterState(nextFilterState);
      setSuggestions({});
      setMainInputActive(true);
      return;
    }
    const nextFilterState = { type: 'filterActive', index };
    filterStateRef.current = nextFilterState;
    setFilterState(nextFilterState);
    setMainInputActive(false);
  }, [filterState]);

  // Click outside to close expanded panel
  useEffect(() => {
    if (filterState.type === 'none' && !mainInputActive) return;

    const rootNode = expandedInputRef.current?.resizableTextArea?.textArea?.getRootNode()
      || inputRef.current?.input?.getRootNode()
      || document;
    const eventTarget = rootNode === document ? document : rootNode;

    const handleClickOutside = (e: any) => {
      // Use the stable rootRef as the click-outside boundary instead of the
      // fragile `[class*="searchbox"]` selector. The previous selector relied
      // on CSS-module hashed class names containing the substring "searchbox",
      // which breaks silently if the class is renamed, and also matches the
      // WRONG instance when multiple SearchBox widgets coexist on a page.
      const searchboxEl = rootRef.current;
      // Use composedPath() to get the actual target inside Shadow DOM
      const actualTarget = e.composedPath ? e.composedPath()[0] : e.target;
      const isInsideSearchbox = !!searchboxEl && searchboxEl.contains(actualTarget);
      if (isInsideSearchbox) {
        lastMouseDownInsideTime.current = Date.now();
        return;
      }
      if (searchboxEl && !isClickingSearchAction.current) {
        if (filterState.type !== 'none') {
          setFilterState({ type: 'none', index: -1 });
          setSuggestions({});
        }
        if (mainInputActive) {
          setMainInputActive(false);
          lastClickOutsideTime.current = Date.now();
        }
      }
    };

    eventTarget.addEventListener('mousedown', handleClickOutside);
    return () => eventTarget.removeEventListener('mousedown', handleClickOutside);
  }, [filterState.type, mainInputActive]);

  const handleInputFocus = useCallback(() => {
    if (isSearchTriggered.current) {
      isSearchTriggered.current = false;
      return;
    }
    // Prevent reopening right after click-outside closed the panel
    if (Date.now() - lastClickOutsideTime.current < 200) {
      return;
    }
    // Cancel any pending blur timeout to prevent race condition
    if (blurTimeoutRef.current) {
      clearTimeout(blurTimeoutRef.current);
      blurTimeoutRef.current = null;
    }
    setMainInputActive(true);
    if (filterState.type !== 'none' || filterState.index !== -1) {
      setFilterState({ type: 'none', index: -1 });
    }
    setTimeout(() => {
      if (isSearchTriggered.current) return;
      const textareaDom = expandedInputRef.current?.resizableTextArea?.textArea;
      if (textareaDom) {
        textareaDom.focus();
        const len = textareaDom.value.length;
        textareaDom.setSelectionRange(len, len);
        setCursorPosition(len);
      }
    }, 0);
  }, [filterState.type, filterState.index]);

  const handleInputBlur = useCallback(() => {
    blurTimeoutRef.current = setTimeout(() => {
      blurTimeoutRef.current = null;
      if (!document.hasFocus()) return;
      if (Date.now() - lastMouseDownInsideTime.current < 200) return;
      if (!isClickingSuggestion.current && !isClickingSearchAction.current) {
        setMainInputActive(false);
      }
    }, 100);
  }, []);

  const handleFilterInputFocus = useCallback((index: number) => {
    setFilterState({ type: 'filterInput', index });
  }, []);

  const handleFilterInputBlur = useCallback(() => {
    setTimeout(() => {
      if (filterStateRef.current.type === 'filterActive') return;
      if (Date.now() - lastMouseDownInsideTime.current < 250) return;
      if (!isClickingSuggestion.current) setFilterState({ type: 'none', index: -1 });
    }, 100);
  }, []);

  const handleSuggestionsResult = useCallback((expectedType: any, res: any) => {
    setSuggestions((prev: any) => {
      // Ignore stale results from a different suggestion type
      if (prev.type !== expectedType) return prev;
      return {
        ...prev,
        data: Array.isArray(res?.suggestions) ? res.suggestions : []
      };
    });
  }, []);

  const handleAttachmentsChange = useCallback((newAttachments: any) => {
    setAttachments?.(newAttachments);
    setMainInputActive(true);
  }, [setAttachments]);

  const handleAttachmentRemove = useCallback((item: any) => {
    const index = attachments.findIndex(a => a.id === item.id);
    if (index !== -1) {
      const newAttachments = attachments.filter((_, attachmentIndex) => attachmentIndex !== index);
      setAttachments?.(newAttachments);
      // Cancel pending blur to prevent panel flicker
      if (blurTimeoutRef.current) {
        clearTimeout(blurTimeoutRef.current);
        blurTimeoutRef.current = null;
      }
      // Keep panel open by activating the main input
      setMainInputActive(true);
      setTimeout(() => {
        if (expandedInputRef.current) expandedInputRef.current.focus();
      }, 50);
    }
  }, [attachments, setAttachments]);

  // Refs to cache the serialized form of queryParams / filterFieldsMeta,
  // so we only reset internal state when their *content* actually changes.
  // This avoids the anti-pattern of JSON.stringify inside the dependency
  // array (which recomputes every render) and prevents spurious resets /
  // potential update loops caused by new object references on each parent
  // render (e.g. overwriting the query the user is currently typing).
  const prevQueryParamsRef = useRef('');
  const prevFilterMetaRef = useRef('');

  // Sync queryParams prop to internal state only when content changes
  useEffect(() => {
    const qpStr = JSON.stringify(queryParams);
    const fmStr = JSON.stringify(filterFieldsMeta);
    // Skip if neither the query params nor the filter meta actually changed
    if (qpStr === prevQueryParamsRef.current && fmStr === prevFilterMetaRef.current) return;
    prevQueryParamsRef.current = qpStr;
    prevFilterMetaRef.current = fmStr;

    const fields = Object.keys(queryParams?.filter || {});
    setCurrentQueryParams({
      ...(queryParams || {}),
      query: queryParams?.query || '',
      filters: fields.map((field) => {
        const isNot = field.startsWith('!');
        const rawField = isNot ? field.slice(1) : field;
        const meta = filterFieldsMeta[rawField] || filterFieldsMeta[field];
        if (meta) {
          return { field: meta, value: queryParams?.filter?.[field], operator: isNot ? 'not' : 'and' };
        }
        return undefined;
      }).filter(Boolean)
    });
  }, [queryParams, filterFieldsMeta]);

  // Reset suggestions when suggestion type or query context changes
  useEffect(() => {
    if (!suggestionType) {
      // If user is interacting with search action (changing search type), don't clear suggestions
      if (isClickingSearchAction.current) return;
      setSuggestions({});
      return;
    }
    setSuggestions({ type: suggestionType, from: 0, size: DEFAULT_SUGGESTIONS_SIZE });
  }, [suggestionType, query]);

  // Clean empty filters when not actively editing
  useEffect(() => {
    if (filterState.type !== 'filterInput' && filterState.type !== 'filterActive') {
      const cleanedFilters = filters.filter((f: any) => {
        const value = f.value;
        return !!value && !(Array.isArray(value) && value.length === 0);
      });
      if (cleanedFilters.length !== filters.length) {
        handleQueryParamsChange('filters', cleanedFilters);
      }
    }
  }, [filterState.type, filters]);

  // Fetch suggestion data from server
  useEffect(() => {
    const { from = 0, size = DEFAULT_SUGGESTIONS_SIZE } = suggestions;
    if (!suggestionType || !onSuggestion) return;

    let suggestionParams: any = { from, size };
    switch (suggestionType) {
      case SUGGESTION_KEYWORDS:
        if (calculateCharLength(query) < 40) {
          suggestionParams.query = query;
          onSuggestion(undefined, suggestionParams, (res: any) => handleSuggestionsResult(SUGGESTION_KEYWORDS, res));
        }
        break;
      case SUGGESTION_FILTER_FIELDS: {
        if (colonFieldQuery) {
          // "keyword:" pattern - use keyword as search query for fields
          suggestionParams.query = colonFieldQuery;
        } else if (slashFieldQuery) {
          // "/keyword" pattern - use keyword after slash
          suggestionParams.query = slashFieldQuery;
        } else {
          const queryBeforeCursor = query?.substring(0, cursorPosition);
          suggestionParams.query = queryBeforeCursor?.endsWith('/') ? queryBeforeCursor.slice(0, -1) : queryBeforeCursor;
        }
        onSuggestion(suggestionType, suggestionParams, (res: any) => handleSuggestionsResult(SUGGESTION_FILTER_FIELDS, res));
        break;
      }
      case SUGGESTION_FILTER_VALUES: {
        const f = filters[filterState.index];
        if (f?.field?.field_name) {
          suggestionParams = { ...suggestionParams, field_name: f.field.field_name, query: filterSearchValue, size: 10 };
          onSuggestion(suggestionType, suggestionParams, (res: any) => handleSuggestionsResult(SUGGESTION_FILTER_VALUES, res));
        }
        break;
      }
    }
  }, [suggestionType, suggestions.from, suggestions.size, query, filterState.type, filterState.index, filters, onSuggestion, cursorPosition, colonFieldQuery, filterSearchValue]);

  // Global Tab key handler
  useEffect(() => {
    const rootNode = expandedInputRef.current?.resizableTextArea?.textArea?.getRootNode()
      || inputRef.current?.input?.getRootNode()
      || document;
    const eventTarget = rootNode === document ? document : rootNode;

    const handleTabKeyDown = (e: any) => {
      if (e.key === 'Tab') {
        e.preventDefault();
        expandedInputRef.current?.focus();
      }
    };
    eventTarget.addEventListener('keydown', handleTabKeyDown);
    return () => eventTarget.removeEventListener('keydown', handleTabKeyDown);
  }, []);

  return {
    query, filters, action_type, search_type,
    suggestions, loadNextSuggestion,
    attachments,
    showExpandedPanel, searchable,
    shouldFocusNewFilter,
    filterState, mainInputActive,
    inputRef, textAreaRef, expandedInputRef, rootRef,
    handleSearchActionClick,
    handleSearchActionDropdownClose,
    handleQueryParamsChange,
    handleCursorPositionChange,
    handleInputChange,
    handleInputFocus,
    handleInputBlur,
    handleFilterInputFocus,
    handleFilterInputBlur,
    handleFilterActiveToggle,
    handleSuggestionItemClick,
    handleSearch,
    handleAddFilter,
    handleFilterValueToggle,
    handleOperatorChange,
    handleFilterComplete,
    handleFilterDelete,
    handleFilterValueEdit,
    filterSearchValue,
    handleFilterSearchChange: setFilterSearchValue,
    handleAttachmentsChange,
    handleAttachmentRemove,
    triggerSearch,
    // expose for parent components to detect context changes
    suggestionType,
    colonFieldQuery,
    slashFieldQuery,
    handleAttachmentUpload
  };
}
