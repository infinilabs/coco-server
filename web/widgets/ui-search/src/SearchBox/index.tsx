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
import { createPortal } from "react-dom";
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
  /** called when Backspace is pressed while the query is empty and no
   * attachments are pending — the host clears its active filter conditions
   * (chips above the results) and returns true if anything was cleared */
  onEmptyBackspace?: () => boolean;
  /** appended to the root element — lets a host override geometry (e.g. the
   * app shell header embeds the box at 40px instead of the default 48px) */
  className?: string;
  /** antd size of the compact input in minimize mode — lets a host shrink the
   * box below the default 48px (the app shell header runs a 32px box) */
  inputSize?: 'small' | 'middle' | 'large';
  /** open the expanded panel as a viewport-centered dialog over a fullscreen
   * mask (command-palette style) instead of the anchored dropdown below the
   * box — meant for hosts embedding the box into a fixed bar (app shell
   * header), where a dropdown would hug the bar's geometry */
  centeredPanel?: boolean;
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
    sort,
    className,
    onEmptyBackspace,
    inputSize = 'large',
    centeredPanel = false
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
    onFilterTabComplete: sb.handleFilterTabComplete,
    turnToChat,
    language,
    settings,
    resetKey: suggestionResetKey,
    // keyboard nav (arrows/Enter on the suggestion list) listens for keydown
    // on this element — it must be an ANCESTOR of whatever holds focus. In
    // centered mode the focus lives in the body portal, where the searchbox
    // root can never see the events; the portal container is on that path.
    keyboardRootRef: centeredPanel ? sb.panelRootRef : sb.rootRef,
    // Tab completes the highlighted keyword/action row into the query with
    // a trailing space instead of searching (omnibox convention) — the
    // highlighted action row echoes the query, so completing it is the same
    // as typing the space itself and multi-word queries keep working.
    // Enter selects and runs. completeQueryText also moves the tracked
    // cursor to the end so the '/'-pattern detection can't misfire.
    onItemComplete: (item: any) => {
      const text = `${item?.suggestion ?? sb.query ?? ''}`.trimEnd();
      if (text) sb.completeQueryText(`${text} `);
    }
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
    sb.completeQueryText,
    sb.handleSuggestionItemClick,
    handleSearch,
    sb.handleAddFilter,
    handleFilterValueToggle,
    sb.handleOperatorChange,
    sb.handleFilterComplete,
    sb.handleFilterTabComplete,
    turnToChat,
    language,
    settings,
    suggestionResetKey,
    centeredPanel,
    sb.panelRootRef,
    sb.rootRef
  ]);

  // Backspace on an empty query (no attachments either) pops the LAST filter
  // chip — one Backspace, one filter — so the keyboard flow can undo the
  // construction step by step. Only when no chips are left does it fall back
  // to the host-wide clear (onEmptyBackspace). preventDefault fires only when
  // something was actually removed, so plain editing behavior is untouched.
  const handleBackspaceClear = useCallback((e: React.KeyboardEvent<HTMLElement>) => {
    if (e.key !== 'Backspace' || e.metaKey || e.ctrlKey || e.altKey) return;
    if ((sb.query || '').length > 0 || sb.attachments.length > 0) return;
    if (sb.filters.length > 0) {
      e.preventDefault();
      sb.handleFilterDelete(sb.filters.length - 1);
      return;
    }
    if (onEmptyBackspace?.()) e.preventDefault();
  }, [sb.query, sb.attachments, sb.filters, sb.handleFilterDelete, onEmptyBackspace]);

  const handleTextAreaKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    handleBackspaceClear(e);
    if (e.key === 'Escape' && centeredPanel && sb.showExpandedPanel) {
      e.preventDefault();
      sb.closeExpandedPanel();
      return;
    }
    if (e.key !== 'Enter' || e.shiftKey) return;
    e.preventDefault();
    if (sb.showExpandedPanel && sb.suggestionType && sb.suggestionType !== SUGGESTION_TIPS) return;
    if (sb.searchable) triggerSearch();
  }, [handleBackspaceClear, centeredPanel, sb.showExpandedPanel, sb.suggestionType, sb.searchable, sb.closeExpandedPanel, triggerSearch]);

  // centered-panel mode is a modal: freeze the page behind the fullscreen
  // mask so the dialog's own scroll areas are the only ones that move
  useEffect(() => {
    if (!centeredPanel || !sb.showExpandedPanel) return undefined;
    const body = document.body;
    const prevOverflow = body.style.overflow;
    body.style.overflow = 'hidden';
    return () => {
      body.style.overflow = prevOverflow;
    };
  }, [centeredPanel, sb.showExpandedPanel]);

  const renderTextArea = (ref: any, className = "", onBlur?: any, maxRows = 6) => (
    <Input.TextArea
      ref={ref}
      placeholder={placeholder}
      autoSize={{ minRows: 1, maxRows }}
      classNames={{ textarea: `!text-17px !bg-transparent placeholder:text-#9AA6B2 dark:placeholder:text-#7E8790 ${maxRows === 1 ? '!overflow-hidden' : ''}` }}
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

  // capped column layout: header embeds (minimize) and the centered dialog
  // both pin the ActionBar and let only the suggestions area scroll — an
  // uncapped panel could push its action bar below the fold forever
  const cappedPanel = minimize || centeredPanel;

  // shared body of the expanded panel — only the wrapper differs (anchored
  // dropdown below the box vs centered dialog over a fullscreen mask)
  const renderExpandedPanelBody = () => (
    <div className={`${styles.gradientBorder} rounded-12px overflow-visible`}>
      <div className={`py-12px rounded-12px bg-[rgba(255,255,255,0.95)] dark:bg-[rgba(31,33,37,0.95)] overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] text-[rgba(0,0,0,0.88)] dark:text-[rgba(255,255,255,0.85)] ${centeredPanel ? 'flex flex-col max-h-[calc(85vh-32px)]' : minimize ? 'flex flex-col max-h-[calc(100vh-64px)]' : ''}`}>
        {sb.attachments.length > 0 && (
          <div className={`mb-14px px-8px shrink-0 ${minimize ? styles.compactAttachments : ''}`}>
            <Attachments data={sb.attachments} onItemRemove={sb.handleAttachmentRemove} />
          </div>
        )}
        <Filters
          ref={filtersRef}
          className={`mb-14px px-12px${cappedPanel ? ' shrink-0' : ''}`}
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
        {renderTextArea(sb.expandedInputRef, `!mb-14px !px-12px${cappedPanel ? ' shrink-0' : ''}`, sb.handleInputBlur)}
        <div className={cappedPanel ? 'flex-1 min-h-0 overflow-y-auto' : undefined}>
          <Suggestions {...suggestionProps} />
        </div>
        <ActionBar {...actionBarProps} className={cappedPanel ? 'shrink-0' : ''} />
      </div>
    </div>
  );

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
      ${className || ''}
      
    `}>
      {minimize ? (
        <>
          <div className="px-12px items-center w-full h-full flex gap-8px">
            {sb.showExpandedPanel ? null : renderFilters()}
            <div className={`${styles.inputWrapper} w-full`}>
              <Input
                ref={sb.inputRef}
                value={sb.query}
                size={inputSize}
                onChange={sb.handleInputChange}
                onSelect={sb.handleCursorPositionChange}
                onClick={sb.handleCursorPositionChange}
                suffix={<Operations size={24} onSearch={triggerSearch} disabled={!sb.searchable} attachments={sb.attachments} setAttachments={sb.handleAttachmentsChange} onAttachmentUpload={sb.handleAttachmentUpload} action_type={sb.action_type}/>}
                placeholder={placeholder}
                className="flex-1 w-full"
                onKeyDown={handleBackspaceClear}
                onFocus={sb.handleInputFocus}
                onBlur={() => {}}
              />
            </div>
          </div>
        </>
      ) : (
        <div className="py-12px h-full flex flex-col">
          {/* input row takes the space above the action bar so the text sits
              vertically centered in the tall home box instead of hugging the top */}
          <div className="items-center w-full flex-1 flex gap-8px px-12px">
            {sb.showExpandedPanel ? null : renderFilters()}
            {renderTextArea(sb.textAreaRef, '!px-0', undefined, 1)}
          </div>
          <ActionBar {...actionBarProps} />
        </div>
      )}
      {/* Expanded Panel */}
      {centeredPanel ? (
        // command-palette form: fullscreen mask + viewport-centered dialog.
        // Portaled to body so host stacking contexts (contain/transform on an
        // ancestor) can neither clip nor re-anchor the fixed overlay. Stays
        // mounted while closed — hidden via opacity/pointer-events, NOT
        // display:none (kills the virtual scroller's measuring) and NOT
        // visibility:hidden (the browser refuses to focus() a hidden element,
        // so the focus handoff from the compact input into the dialog dies).
        createPortal(
          <div
            className={`fixed inset-0 z-[2000] transition-opacity duration-150 ${sb.showExpandedPanel ? 'opacity-100' : 'pointer-events-none opacity-0'}`}
            ref={sb.panelRootRef}
          >
            {/* the mask is inside the panel root, so the hook's click-outside
                never fires for it — it closes through its own handler */}
            <div
              className="absolute inset-0 bg-[rgba(0,0,0,0.45)] dark:bg-[rgba(0,0,0,0.65)]"
              onMouseDown={sb.closeExpandedPanel}
            />
            {/* anchored by its top edge, NOT vertically centered: suggestion
                content loads in asynchronously, and a centered box would ride
                up/down with every height change — pinned top means the input
                row stays put and only the list grows downward */}
            <div className="absolute left-1/2 top-[15vh] w-[min(720px,calc(100vw-48px))] -translate-x-1/2">
              {renderExpandedPanelBody()}
            </div>
          </div>,
          document.body
        )
      ) : (
        /* Keep `h-0 overflow-hidden` (NOT display:none) so the virtual scroller
          inside Suggestions/ListContainer stays laid-out and keeps measuring
          item sizes even when hidden. Add `pointer-events-none` when closed to
          guarantee the zero-height absolutely-positioned panel can never
          intercept clicks / block the real input underneath (focus anomaly).
          The panel overlays the compact input (top-0) by design — it needs an
          OPAQUE background or the compact input ghosts through it.
          `--ui-search--layout-bg-color` is never declared anywhere in host or
          widget, so the var-based bg resolves invalid and paints nothing; use
          explicit near-opaque (95%) theme-aware colors instead. */
        <div className={`absolute left-0 top-0 z-100 w-full ${sb.showExpandedPanel ? '' : 'h-0 overflow-hidden pointer-events-none'} `}>
          {renderExpandedPanelBody()}
        </div>
      )}
    </div>
  );
}

export default SearchBox;