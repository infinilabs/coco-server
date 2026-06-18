import { Badge, Button, Input } from "antd";
import styles from "./index.module.less";
import Operations from "./ActionBar/Operations";
import Filters from "./Filters";
import { Attachments } from "@infinilabs/attachments";
import useSearchBox from "./useSearchBox";
import Suggestions from "./Suggestions";
import ActionBar from "./ActionBar";
import { ListFilter } from "lucide-react";
import { useCallback, useMemo } from "react";

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
    settings
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

  const handleSearchTypeChange = useCallback((type: string) => {
    sb.handleQueryParamsChange('search_type', type);
  }, [sb.handleQueryParamsChange]);

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

  const actionBarProps = useMemo(() => ({
    action_type: sb.action_type,
    search_type: sb.search_type,
    onSearchTypeChange: handleSearchTypeChange,
    onSearchActionClick: sb.handleSearchActionClick,
    onSearchActionDropdownClose: sb.handleSearchActionDropdownClose,
    attachments: sb.attachments,
    onAttachmentsChange: sb.handleAttachmentsChange,
    onSearch: sb.triggerSearch,
    searchable: sb.searchable,
    onAttachmentUpload: sb.handleAttachmentUpload
  }), [
    sb.action_type,
    sb.search_type,
    handleSearchTypeChange,
    sb.handleSearchActionClick,
    sb.handleSearchActionDropdownClose,
    sb.attachments,
    sb.handleAttachmentsChange,
    sb.triggerSearch,
    sb.searchable,
    sb.handleAttachmentUpload
  ]);

  const suggestionProps = useMemo(() => ({
    suggestions: sb.suggestions,
    onLoadNext: sb.loadNextSuggestion,
    query: sb.query,
    filters: sb.filters,
    action_type: sb.action_type,
    search_type: sb.search_type,
    filterState: sb.filterState,
    mainInputActive: sb.mainInputActive,
    handleQueryParamsChange: sb.handleQueryParamsChange,
    handleSuggestionItemClick: sb.handleSuggestionItemClick,
    handleSearch: sb.handleSearch,
    handleAddFilter: sb.handleAddFilter,
    handleFilterValueToggle: sb.handleFilterValueToggle,
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
    sb.search_type,
    sb.filterState,
    sb.mainInputActive,
    sb.handleQueryParamsChange,
    sb.handleSuggestionItemClick,
    sb.handleSearch,
    sb.handleAddFilter,
    sb.handleFilterValueToggle,
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
    if (sb.searchable) sb.triggerSearch();
  }, [sb.searchable, sb.triggerSearch]);

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
      bg-[rgb(var(--ui-search--layout-bg-color))]
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
              suffix={<Operations size={24} onSearch={sb.triggerSearch} disabled={!sb.searchable} attachments={sb.attachments} setAttachments={sb.handleAttachmentsChange} onAttachmentUpload={sb.handleAttachmentUpload} action_type={sb.action_type}/>}
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