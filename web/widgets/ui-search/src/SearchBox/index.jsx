import { Badge, Button, Input } from "antd";
import styles from "./index.module.less";
import Operations from "./ActionBar/Operations";
import Filters from "./Filters";
import { Attachments } from "@infinilabs/attachments";
import useSearchBox from "./useSearchBox";
import Suggestions from "./Suggestions";
import ActionBar from "./ActionBar";
import { ListFilter } from "lucide-react";

export function SearchBox(props) {
  const { placeholder, queryParams, setQueryParams, onSearch, minimize = false, onSuggestion, filterFieldsMeta = {} } = props;
  const sb = useSearchBox({ queryParams, onSearch, onSuggestion, filterFieldsMeta });

  const actionBarProps = {
    action_type: sb.action_type,
    search_type: sb.search_type,
    onSearchTypeChange: (type) => sb.handleQueryParamsChange('search_type', type),
    onSearchActionClick: sb.handleSearchActionClick,
    onSearchActionDropdownClose: sb.handleSearchActionDropdownClose,
    attachments: sb.attachments,
    onAttachmentsChange: sb.handleAttachmentsChange,
    onSearch: sb.triggerSearch,
    searchable: sb.searchable,
  };

  const suggestionProps = {
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
    turnToChat: (item) => {
      setQueryParams({
        query: item.suggestion,
        mode: 'chat',
      })
    }
  };

  const renderTextArea = (ref, className = "", onBlur) => (
    <Input.TextArea
      ref={ref}
      placeholder={placeholder}
      autoSize={{ minRows: 1, maxRows: 6 }}
      classNames={{ textarea: '!text-16px !bg-transparent' }}
      value={sb.query}
      onChange={sb.handleInputChange}
      onSelect={sb.handleCursorPositionChange}
      onClick={sb.handleCursorPositionChange}
      onFocus={sb.handleInputFocus}
      onBlur={onBlur}
      className={`${styles.input} ${className}`}
    />
  );

  const renderFilters = () => {
    const validFilters = sb.filters.filter(f => !!f.field?.field_label);
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
    <div className={`
      ${styles.searchbox}
      relative w-full rounded-12px 
      ${sb.showExpandedPanel ? '' : 'border'} 
      border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)] 
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
              suffix={<Operations size={24} onSearch={sb.triggerSearch} disabled={!sb.searchable} />}
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
            {renderTextArea(sb.textAreaRef, '!px-0')}
          </div>
          <ActionBar {...actionBarProps} />
        </div>
      )}
      {/* Expanded Panel */}
      <div className={`absolute left-0 top-0 z-100 w-full ${sb.showExpandedPanel ? '' : 'h-0 overflow-hidden'} `}>
        <div className={`${styles.gradientBorder} rounded-12px overflow-visible`}> 
          <div className={`py-12px rounded-12px bg-[rgb(var(--ui-search--layout-bg-color))] overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] border border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)]`}>
            {sb.attachments.length > 0 && (
              <div className="mb-14px px-16px">
                <Attachments data={sb.attachments} onItemRemove={sb.handleAttachmentRemove} />
              </div>
            )}
            <Filters
              className="mb-14px px-12px"
              filters={sb.filters}
              onFiltersChange={(filters) => sb.handleQueryParamsChange('filters', filters)}
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