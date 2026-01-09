/**
 * Formats a date as YYYY-MM-DD in local timezone
 */
export function getLocalDateString(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Parses a date string in YYYY-MM-DD format to a Date object at midnight local time
 */
export function parseDateString(dateStr: string): Date {
  return new Date(dateStr + 'T00:00:00');
}
