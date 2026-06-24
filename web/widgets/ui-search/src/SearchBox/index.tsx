import { Badge, Button, Input } from "antd";
import styles from "./index.module.less";
import Operations from "./ActionBar/Operations";
import Filters from "./Filters";
import { Attachments } from "@infinilabs/attachments";
import useSearchBox from "./useSearchBox";
import Suggestions from "./Suggestions";
import ActionBar from "./ActionBar";
import { ListFilter } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { SUGGESTION_TIPS } from "./Suggestions/Tips";
import { DEFAULT_SEARCH_FUZZINESS, DEFAULT_SEARCH_SORT, normalizeSearchFuzziness, normalizeSearchSort } from "./ActionBar/SearchActions";

interface SearchBoxProps {
  placeholder?: string;
  queryParams?: Record<string, any>;
  setQueryParams?: (params: any) => void;
  onSearch?: (...args: any[]) => void;
  minimize?: boolean;
  onSuggestion?: (...args: any[]) => void;
  filterFieldsMeta?: Record<string, any>;
  language?: string;
  onUpload?: (files: File[], cb: (res: any) => void) => void;
  attachments?: any[];
  setAttachments?: (attachments: any[]) => void;
  settings?: Record<string, any>;
  searchType?: string;
  onSearchTypeChange?: (type: string) => void;
  fuzziness?: number;
  sort?: string;
  [key: string]: any;
}

export function SearchBox(props: SearchBoxProps) {
  const { 
    placeholder, 
    queryParams, 
    setQueryParams, 
    onSearch, 
    minimize = false, 
    onSuggestion, 
    filterFieldsMeta = {}, 
    language, 
    onUpload,
    attachments,
    setAttachments,
    settings,
    searchType,
    fuzziness,
    sort
  } = props;
  const sb = useSearchBox({ 
    queryParams, 
    onSearch, 
    onSuggestion, 
    filterFieldsMeta, 
    onUpload,
    attachments: attachments as any,
    setAttachments
  });
  const filtersRef = useRef<any>(null);
  const lastSearchTypeRef = useRef(searchType);

  useEffect(() => {
    if (searchType === lastSearchTypeRef.current) return;
    lastSearchTypeRef.current = searchType;
    if (!searchType) return;
    sb.handleQueryParamsChange('search_type', searchType);
  }, [searchType, sb.handleQueryParamsChange]);

  useEffect(() => {
    if (typeof fuzziness !== 'number' || fuzziness === sb.fuzziness) return;
    sb.handleQueryParamsChange('fuzziness', fuzziness);
  }, [fuzziness, sb.fuzziness, sb.handleQueryParamsChange]);

  useEffect(() => {
    if (!sort || sort === sb.sort) return;
    sb.handleQueryParamsChange('sort', sort);
  }, [sort, sb.sort, sb.handleQueryParamsChange]);

  const handleSearchTypeChange = useCallback((type: string) => {
    sb.handleQueryParamsChange('search_type', type);
  }, [sb.handleQueryParamsChange]);

  const effectiveSearchType = sb.search_type || searchType;
  const effectiveFuzziness = normalizeSearchFuzziness(typeof fuzziness === 'number' ? fuzziness : sb.fuzziness || DEFAULT_SEARCH_FUZZINESS);
  const effectiveSort = normalizeSearchSort(sort || sb.sort || DEFAULT_SEARCH_SORT);

  const triggerSearch = useCallback(() => {
    sb.handleSearch(sb.query, sb.filters, sb.action_type, effectiveSearchType, effectiveFuzziness, effectiveSort);
  }, [effectiveFuzziness, effectiveSearchType, effectiveSort, sb.action_type, sb.filters, sb.handleSearch, sb.query]);

  const handleSearch = useCallback((searchQuery: any, searchFilters: any, actionType: any, searchType: any) => {
    sb.handleSearch(searchQuery, searchFilters, actionType, searchType, effectiveFuzziness, effectiveSort);
  }, [effectiveFuzziness, effectiveSort, sb.handleSearch]);

  const handleFiltersChange = useCallback((filters: any) => {
    sb.handleQueryParamsChange('filters', filters);
  }, [sb.handleQueryParamsChange]);

  const turnToChat = useCallback((item: any) => {
    onSearch?.({
      query: item.suggestion,
      attachments: attachments,
      mode: 'chat',
      action: item.action,
      assistant_id: item.assistant_id
    })
  }, [onSearch, attachments]);

  const suggestionResetKey = useMemo(() => {
    return `${sb.suggestionType || ''}::${sb.colonFieldQuery || ''}::${sb.slashFieldQuery || ''}::${sb.filterSearchValue || ''}`;
  }, [sb.suggestionType, sb.colonFieldQuery, sb.slashFieldQuery, sb.filterSearchValue]);

  const handleFilterValueToggle = useCallback((item: any) => {
    const filterIndex = sb.filterState.index;
    const filter = sb.filters[filterIndex];
    sb.handleFilterValueToggle(item);
    if (filter?.field?.support_multi_select) {
      filtersRef.current?.focusFilterInput?.(filterIndex);
    }
  }, [sb.filterState.index, sb.filters, sb.handleFilterValueToggle]);

  const actionBarProps = useMemo(() => ({
    action_type: sb.action_type,
    search_type: effectiveSearchType,
    onSearchTypeChange: handleSearchTypeChange,
    onSearchActionClick: sb.handleSearchActionClick,
    onSearchActionDropdownClose: sb.handleSearchActionDropdownClose,
    attachments: sb.attachments,
    onAttachmentsChange: sb.handleAttachmentsChange,
    onSearch: triggerSearch,
    searchable: sb.searchable,
    onAttachmentUpload: sb.handleAttachmentUpload
  }), [
    sb.action_type,
    sb.search_type,
    effectiveSearchType,
    handleSearchTypeChange,
    sb.handleSearchActionClick,
    sb.handleSearchActionDropdownClose,
    sb.attachments,
    sb.handleAttachmentsChange,
    triggerSearch,
    sb.searchable,
    sb.handleAttachmentUpload
  ]);

  const suggestionProps = useMemo(() => ({
    suggestions: sb.suggestions,
    onLoadNext: sb.loadNextSuggestion,
    query: sb.query,
    filters: sb.filters,
    action_type: sb.action_type,
    search_type: effectiveSearchType,
    filterState: sb.filterState,
    mainInputActive: sb.mainInputActive,
    handleQueryParamsChange: sb.handleQueryParamsChange,
    handleSuggestionItemClick: sb.handleSuggestionItemClick,
    handleSearch,
    handleAddFilter: sb.handleAddFilter,
    handleFilterValueToggle,
    handleOperatorChange: sb.handleOperatorChange,
    handleFilterComplete: sb.handleFilterComplete,
    turnToChat,
    language,
    settings,
    resetKey: suggestionResetKey,
    keyboardRootRef: sb.rootRef
  }), [
    sb.suggestions,
    sb.loadNextSuggestion,
    sb.query,
    sb.filters,
    sb.action_type,
    effectiveSearchType,
    sb.filterState,
    sb.mainInputActive,
    sb.handleQueryParamsChange,
    sb.handleSuggestionItemClick,
    handleSearch,
    sb.handleAddFilter,
    handleFilterValueToggle,
    sb.handleOperatorChange,
    sb.handleFilterComplete,
    turnToChat,
    language,
    settings,
    suggestionResetKey,
    sb.rootRef
  ]);

  const handleTextAreaKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== 'Enter' || e.shiftKey) return;
    e.preventDefault();
    if (sb.showExpandedPanel && sb.suggestionType && sb.suggestionType !== SUGGESTION_TIPS) return;
    if (sb.searchable) triggerSearch();
  }, [sb.showExpandedPanel, sb.suggestionType, sb.searchable, triggerSearch]);

  const renderTextArea = (ref: any, className = "", onBlur?: any, maxRows = 6) => (
    <Input.TextArea
      ref={ref}
      placeholder={placeholder}
      autoSize={{ minRows: 1, maxRows }}
      classNames={{ textarea: `!text-16px !bg-transparent ${maxRows === 1 ? '!overflow-hidden' : ''}` }}
      value={sb.query}
      onChange={sb.handleInputChange}
      onSelect={sb.handleCursorPositionChange}
      onClick={sb.handleCursorPositionChange}
      onFocus={sb.handleInputFocus}
      onBlur={onBlur}
      onKeyDown={handleTextAreaKeyDown}
      className={`${styles.input} ${className}`}
    />
  );

  const renderFilters = () => {
    const validFilters = sb.filters.filter((f: any) => !!f.field?.field_label);
    if (validFilters.length === 0) return null;

    return (
      <Badge count={validFilters.length} size="small" classNames={{ indicator: '!text-10px'}}>
        <Button
          style={{ minWidth: 24, width: 24, height: 24 }}
          classNames={{ icon: `w-16px h-16px !text-16px` }}
          icon={<ListFilter className="w-16px h-16px" />}
          type="text"
          shape="circle"
          onClick={() => sb.handleInputFocus()}
        />
      </Badge>
    )
  };

  return (
    <div
      ref={sb.rootRef}
      className={`
      ${styles.searchbox}
      relative w-full rounded-12px 
      ${minimize ? 'border border-solid' : ''} 
      border-[#F0F0F0] dark:border-[#303030] 
      ${minimize ? 'h-48px' : `h-103px ${sb.showExpandedPanel ? '' : 'shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]'}`}
      ${!minimize && !sb.showExpandedPanel ? styles.gradientBorder : ''}
      ${!minimize && !sb.showExpandedPanel ? styles.gradientBorder : ''}
      ${minimize ? 'bg-[rgba(243,244,246,1)] dark:bg-[rgba(31,33,37,1)]' : 'bg-[rgb(var(--ui-search--layout-bg-color))]'}
      
    `}>
      {minimize ? (
        <div className="px-12px items-center w-full h-full flex gap-8px">
          {sb.showExpandedPanel ? null : renderFilters()}
          <div className={`${styles.inputWrapper} w-full`}>
            <Input
              ref={sb.inputRef}
              value={sb.query}
              size="large"
              onChange={sb.handleInputChange}
              onSelect={sb.handleCursorPositionChange}
              onClick={sb.handleCursorPositionChange}
              suffix={<Operations size={24} onSearch={triggerSearch} disabled={!sb.searchable} attachments={sb.attachments} setAttachments={sb.handleAttachmentsChange} onAttachmentUpload={sb.handleAttachmentUpload} action_type={sb.action_type}/>} 
              placeholder={placeholder}
              className="flex-1 w-full"
              onFocus={sb.handleInputFocus}
              onBlur={() => {}}
            />
          </div>
        </div>
      ) : (
        <div className="py-12px">
          <div className="items-center w-full h-full flex gap-8px px-12px mb-14px">
            {sb.showExpandedPanel ? null : renderFilters()}
            {renderTextArea(sb.textAreaRef, '!px-0', undefined, 1)}
          </div>
          <ActionBar {...actionBarProps} />
        </div>
      )}
      {/* Expanded Panel */}
      {/* Keep `h-0 overflow-hidden` (NOT display:none) so the virtual scroller
          inside Suggestions/ListContainer stays laid-out and keeps measuring
          item sizes even when hidden. Add `pointer-events-none` when closed to
          guarantee the zero-height absolutely-positioned panel can never
          intercept clicks / block the real input underneath (focus anomaly). */}
      <div className={`absolute left-0 top-0 z-100 w-full ${sb.showExpandedPanel ? '' : 'h-0 overflow-hidden pointer-events-none'} `}>
        <div className={`${styles.gradientBorder} rounded-12px overflow-visible`}> 
          <div className={`py-12px rounded-12px bg-[rgb(var(--ui-search--layout-bg-color))] overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`}>
            {sb.attachments.length > 0 && (
              <div className="mb-14px px-8px">
                <Attachments data={sb.attachments} onItemRemove={sb.handleAttachmentRemove} />
              </div>
            )}
            <Filters
              ref={filtersRef}
              className="mb-14px px-12px"
              filters={sb.filters}
              onFiltersChange={handleFiltersChange}
              onFilterInputFocus={sb.handleFilterInputFocus}
              onFilterInputBlur={sb.handleFilterInputBlur}
              onFilterActiveToggle={sb.handleFilterActiveToggle}
              onFilterDelete={sb.handleFilterDelete}
              onFilterValueEdit={sb.handleFilterValueEdit}
              onFilterComplete={sb.handleFilterComplete}
              onFilterSearch={sb.handleFilterSearchChange}
              filterSearchValue={sb.filterSearchValue}
              focusIndex={sb.filterState.type === 'filterInput' ? sb.filterState.index : -1}
              activeIndex={sb.filterState.type === 'filterActive' ? sb.filterState.index : -1}
              shouldFocusNewFilter={sb.shouldFocusNewFilter}
            />
            {renderTextArea(sb.expandedInputRef, '!mb-14px !px-12px', sb.handleInputBlur)}
            <Suggestions {...suggestionProps} />
            <ActionBar {...actionBarProps} />
          </div>
        </div>
      </div>
    </div>
  );
}

export default SearchBox;