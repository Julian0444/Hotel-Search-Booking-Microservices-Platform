// Retired: the previous backfill deleted all inventory and guessed missing
// reservation fields. It cannot safely distinguish an applied write from an
// ambiguous old operation. Preserve existing data and inspect the audit report.
throw new Error('Retired migration: run hotels-api/seed/audit-reservations.js with hotels-api stopped. It reports inconsistencies without rebuilding counters.');
