// tiny muted dot between meta entries — same treatment as the preview panel's
// meta line, so list rows and the detail pane read as one design
export function MetaDot() {
  return <span aria-hidden="true" className="size-3px shrink-0 rounded-full bg-current opacity-35" />;
}
