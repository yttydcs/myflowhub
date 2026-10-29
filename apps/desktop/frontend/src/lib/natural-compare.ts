// Reuse the locale-aware comparator; creating collation options for every
// comparison dominates sorting large resource catalogs.
export const compareNatural = new Intl.Collator(undefined, { numeric: true }).compare
