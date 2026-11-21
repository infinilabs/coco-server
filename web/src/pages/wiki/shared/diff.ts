/* Line diff for the version history view (design doc D1c).
 *
 * Small dependency-free LCS diff: common prefix/suffix are trimmed first,
 * and oversized middles fall back to a wholesale replace so the DP table
 * stays bounded even for very large documents.
 */

export interface DiffLine {
  type: 'add' | 'del' | 'same';
  text: string;
}

const MAX_DP_LINES = 1500;

export function diffLines(oldText: string, newText: string): DiffLine[] {
  const a = oldText.split('\n');
  const b = newText.split('\n');

  const out: DiffLine[] = [];

  // common prefix
  let start = 0;
  while (start < a.length && start < b.length && a[start] === b[start]) {
    out.push({ type: 'same', text: a[start] });
    start += 1;
  }
  // common suffix
  let endA = a.length;
  let endB = b.length;
  const suffix: DiffLine[] = [];
  while (endA > start && endB > start && a[endA - 1] === b[endB - 1]) {
    suffix.unshift({ type: 'same', text: a[endA - 1] });
    endA -= 1;
    endB -= 1;
  }

  const midA = a.slice(start, endA);
  const midB = b.slice(start, endB);

  if (midA.length > MAX_DP_LINES || midB.length > MAX_DP_LINES) {
    for (const line of midA) out.push({ type: 'del', text: line });
    for (const line of midB) out.push({ type: 'add', text: line });
  } else {
    // classic LCS table over the differing middles
    const n = midA.length;
    const m = midB.length;
    const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
    for (let i = n - 1; i >= 0; i -= 1) {
      for (let j = m - 1; j >= 0; j -= 1) {
        dp[i][j] = midA[i] === midB[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
      }
    }
    let i = 0;
    let j = 0;
    while (i < n && j < m) {
      if (midA[i] === midB[j]) {
        out.push({ type: 'same', text: midA[i] });
        i += 1;
        j += 1;
      } else if (dp[i + 1][j] >= dp[i][j + 1]) {
        out.push({ type: 'del', text: midA[i] });
        i += 1;
      } else {
        out.push({ type: 'add', text: midB[j] });
        j += 1;
      }
    }
    while (i < n) {
      out.push({ type: 'del', text: midA[i] });
      i += 1;
    }
    while (j < m) {
      out.push({ type: 'add', text: midB[j] });
      j += 1;
    }
  }

  return out.concat(suffix);
}
